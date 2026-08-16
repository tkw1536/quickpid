package api

//spellchecker:words encoding json errors http regexp github quickpid internal strict
import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/tkw1536/quickpid/internal/strict"
)

//spellchecker:words nolint recvcheck

// ValidPID represents a valid pid.
//
// Use [NewPID] to create a new pid.
type ValidPID struct {
	valid bool
	value string
}

func (pid ValidPID) String() string {
	if !pid.valid {
		panic("invalid pid")
	}
	return pid.value
}

var (
	pidRE         = regexp.MustCompile(`^[a-z0-9_-]+$`)
	errInvalidPID = errors.New("invalid pid")
)

// NewPID creates a new PID.
func NewPID(value string) (ValidPID, error) {
	if !pidRE.MatchString(value) {
		return ValidPID{}, errInvalidPID
	}
	return ValidPID{valid: true, value: value}, nil
}

// ValidResourceURL represents a valid absolute URI for a resource target URL.
// The zero value is not valid.
//
// Use [NewResourceURL] to create a new resource URL.
//
//nolint:recvcheck // StringPtr method is intentionally used with a pointer.
type ValidResourceURL struct {
	valid bool
	value string
}

// String returns the resource URL as a string.
func (u ValidResourceURL) String() string {
	if !u.valid {
		panic("invalid resource url")
	}
	return u.value
}

// StringPtr returns the resource URL as a pointer to a string.
func (u *ValidResourceURL) StringPtr() *string {
	if u == nil {
		return nil
	}
	return new(u.value)
}

// NewResourceURL creates a new resource URL.
// The value must be a valid absolute URI with a non-empty scheme.
// Unlike [NewBaseURI], query strings and fragments are allowed.
func NewResourceURL(value string) (ValidResourceURL, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return ValidResourceURL{}, fmt.Errorf("failed to parse as url: %w", err)
	}
	if !parsed.IsAbs() {
		return ValidResourceURL{}, errNotAbsoluteURI
	}
	return ValidResourceURL{valid: true, value: value}, nil
}

// ResourceCreateRequest is the JSON body for createResource and batchCreateResources items.
type ResourceCreateRequest struct {
	URL      *string
	Metadata *string
	Tags     []string
}

func (r *ResourceCreateRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		URL      strict.Optional[*string]            `json:"url"`
		Metadata strict.Optional[*string]            `json:"metadata"`
		Tags     strict.Optional[strict.StringSlice] `json:"tags"`
	}
	decoded, err := strict.UnmarshalStrict[internal](data)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToUnmarshalFields, err)
	}

	if !decoded.URL.Present {
		return missingRequiredFieldError("url")
	}
	r.URL = decoded.URL.Value

	if !decoded.Metadata.Present {
		return missingRequiredFieldError("metadata")
	}
	r.Metadata = decoded.Metadata.Value

	if !decoded.Tags.Present {
		return missingRequiredFieldError("tags")
	}
	r.Tags = decoded.Tags.Value.Strings()

	return nil
}

// Validate checks if the given request is valid.
func (r *ResourceCreateRequest) Validate() (ValidResourceCreateRequest, error) {
	var url *ValidResourceURL
	if r.URL != nil {
		validated, err := NewResourceURL(*r.URL)
		if err != nil {
			return ValidResourceCreateRequest{}, err
		}
		url = &validated
	}
	return ValidResourceCreateRequest{
		URL:      url,
		Metadata: r.Metadata,
		Tags:     r.Tags,
	}, nil
}

// ValidResourceCreateRequest is like a [ResourceCreateRequest] but with a validated URL.
type ValidResourceCreateRequest struct {
	URL      *ValidResourceURL
	Metadata *string
	Tags     []string
}

// ResourceResponse is returned for resource operations.
type ResourceResponse struct {
	PID         string
	URL         *string
	Metadata    *string
	DateCreated string
	DateUpdated string
	Tags        []string
	Deleted     bool
}

// ResourceGetResult is the JSON body for GET resource (full 200 or censored 410).
//
// It is either a [ResourceResponse] or a [RedactedResourceResponse].
type ResourceGetResult interface {
	resourceGetResult()
	json.Marshaler
}

func (r ResourceResponse) MarshalJSON() ([]byte, error) {
	type out struct {
		PID         string   `json:"pid"`
		URL         *string  `json:"url"`
		Metadata    *string  `json:"metadata"`
		DateCreated string   `json:"dateCreated"`
		DateUpdated string   `json:"dateUpdated"`
		Tags        []string `json:"tags"`
		Deleted     bool     `json:"deleted"`
	}
	tags := r.Tags
	if tags == nil {
		tags = []string{}
	}
	bytes, err := json.Marshal(out{
		PID:         r.PID,
		URL:         r.URL,
		Metadata:    r.Metadata,
		DateCreated: r.DateCreated,
		DateUpdated: r.DateUpdated,
		Tags:        tags,
		Deleted:     r.Deleted,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resource response: %w", err)
	}
	return bytes, nil
}

func (ResourceResponse) resourceGetResult() {}

// RedactedResourceResponse is returned for GET resource when the resource is
// deleted and the caller is not allowed to see the full object.
type RedactedResourceResponse struct {
	PID         string
	DateCreated string
	DateUpdated string
}

func (r ResourceResponse) Redact() RedactedResourceResponse {
	return RedactedResourceResponse{
		PID:         r.PID,
		DateCreated: r.DateCreated,
		DateUpdated: r.DateUpdated,
	}
}

func (RedactedResourceResponse) resourceGetResult() {}

// StatusCode returns HTTP 410 Gone for censored deleted resources.
func (RedactedResourceResponse) StatusCode() int {
	return http.StatusGone
}

func (r RedactedResourceResponse) MarshalJSON() ([]byte, error) {
	type out struct {
		PID         string `json:"pid"`
		DateCreated string `json:"dateCreated"`
		DateUpdated string `json:"dateUpdated"`
		Deleted     bool   `json:"deleted"`
	}
	bytes, err := json.Marshal(out{
		PID:         r.PID,
		DateCreated: r.DateCreated,
		DateUpdated: r.DateUpdated,
		Deleted:     true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal redacted resource response: %w", err)
	}
	return bytes, nil
}

type PaginatedResourcesResponse struct {
	Total  int `json:"total"`
	Offset int `json:"offset"`

	Items []ResourceResponse `json:"items"`
}

// ResourceCountResponse is returned for countAllResources.
type ResourceCountResponse struct {
	Total int `json:"total"`
}

// ResourceUpdateRequest is the JSON body for updateResource.
//
// A nil pointer (or nil Tags slice) indicates that no update should be performed on that field.
// When Tags is non-nil, it replaces the resource tags (which might be empty).
// URL and Metadata use three-state pointers: nil = omit, &nil = clear/set null, &&value = set.
type ResourceUpdateRequest struct {
	URL      **string
	Metadata **string
	Tags     []string
	Deleted  *bool
}

func (r *ResourceUpdateRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		URL      strict.Optional[*string]            `json:"url"`
		Metadata strict.Optional[*string]            `json:"metadata"`
		Tags     strict.Optional[strict.StringSlice] `json:"tags"`
		Deleted  strict.Optional[strict.Bool]        `json:"deleted"`
	}
	decoded, err := strict.UnmarshalStrict[internal](data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal resource update request: %w", err)
	}
	r.URL = decoded.URL.ToPointer()
	r.Metadata = decoded.Metadata.ToPointer()
	r.Deleted = strict.OptionalBoolToPointer(decoded.Deleted)

	if decoded.Tags.Present {
		r.Tags = decoded.Tags.Value.Strings()
	} else {
		r.Tags = nil
	}

	return nil
}

// Validate checks if the given request is valid.
func (r *ResourceUpdateRequest) Validate() (ValidResourceUpdateRequest, error) {
	var url **ValidResourceURL
	switch {
	case r.URL == nil: /* keep url = nil */
	case *r.URL == nil:
		url = new(*ValidResourceURL)
	default:
		validated, err := NewResourceURL(**r.URL)
		if err != nil {
			return ValidResourceUpdateRequest{}, err
		}
		url = new(&validated)
	}
	return ValidResourceUpdateRequest{
		URL:      url,
		Metadata: r.Metadata,
		Tags:     r.Tags,
		Deleted:  r.Deleted,
	}, nil
}

// ValidResourceUpdateRequest is like a [ResourceUpdateRequest] but with a validated URL.
type ValidResourceUpdateRequest struct {
	URL      **ValidResourceURL
	Metadata **string
	Tags     []string
	Deleted  *bool
}

// ListResourcesParams carries path and query parameters for listResources.
type ListResourcesParams struct {
	Tag     *string // optionally filter by tag membership
	Deleted *bool   // optionally filter by deletion status

	Limit  int
	Offset int
}
