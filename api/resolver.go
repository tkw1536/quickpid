// Package api holds type definitions for the PID Resolver API.
package api

//spellchecker:words encoding json errors http time github quickpid internal strict embed
import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/tkw1536/quickpid/internal/strict"
	"github.com/tkw1536/quickpid/pid"

	_ "embed"
)

// NamespaceCreateRequest is the JSON body for createNamespace.
type NamespaceCreateRequest struct {
	Tags      []string
	PIDFormat pid.Format
}

var (
	errInvalidPIDFormat = errors.New("invalid PID format")
)

func (r *NamespaceCreateRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		Tags      strict.Optional[strict.StringSlice] `json:"tags"`
		PIDFormat strict.Optional[pid.Format]         `json:"pidFormat"`
	}
	decoded, err := strict.UnmarshalStrict[internal](data)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToUnmarshalFields, err)
	}
	if !decoded.Tags.Present {
		return missingRequiredFieldError("tags")
	}
	r.Tags = decoded.Tags.Value.Strings()
	if !decoded.PIDFormat.Present {
		return missingRequiredFieldError("pidFormat")
	}
	r.PIDFormat = decoded.PIDFormat.Value
	if err := r.PIDFormat.Validate(); err != nil {
		return fmt.Errorf("%w: %w", errInvalidPIDFormat, err)
	}

	return nil
}

// NamespaceUpdateRequest is the JSON body for updateNamespace.
//
// A nil Tags slice indicates that no update should be performed on that field.
// When Tags is non-nil, it replaces the namespace tags (which might be empty).
type NamespaceUpdateRequest struct {
	Tags []string
}

func (r *NamespaceUpdateRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		Tags strict.Optional[strict.StringSlice] `json:"tags"`
	}
	decoded, err := strict.UnmarshalStrict[internal](data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal namespace update request: %w", err)
	}
	if decoded.Tags.Present {
		r.Tags = decoded.Tags.Value.Strings()
	} else {
		r.Tags = nil
	}
	return nil
}

// Validate checks if the given request is valid.
func (r *NamespaceUpdateRequest) Validate() (ValidNamespaceUpdateRequest, error) {
	return ValidNamespaceUpdateRequest{
		Tags: r.Tags,
	}, nil
}

// ValidNamespaceUpdateRequest is like a [NamespaceUpdateRequest].
type ValidNamespaceUpdateRequest struct {
	Tags []string
}

// NamespaceResponse is returned for namespace operations.
type NamespaceResponse struct {
	ID          string
	Tags        []string
	PIDFormat   pid.Format
	DateCreated time.Time
	DateUpdated time.Time
}

func (n NamespaceResponse) MarshalJSON() ([]byte, error) {
	type out struct {
		ID          string     `json:"id"`
		Tags        []string   `json:"tags"`
		PIDFormat   pid.Format `json:"pidFormat"`
		DateCreated time.Time  `json:"dateCreated"`
		DateUpdated time.Time  `json:"dateUpdated"`
	}
	tags := n.Tags
	if tags == nil {
		tags = []string{}
	}
	bytes, err := json.Marshal(out{
		ID:          n.ID,
		Tags:        tags,
		PIDFormat:   n.PIDFormat,
		DateCreated: n.DateCreated,
		DateUpdated: n.DateUpdated,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal namespace response: %w", err)
	}
	return bytes, nil
}

type PaginatedNamespacesResponse struct {
	Total  int `json:"total"`
	Offset int `json:"offset"`

	Items []NamespaceResponse `json:"items"`
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

type ListNamespacesParams struct {
	Tag *string // optionally filter by tag

	Limit  int
	Offset int
}

// InfoResponse provides information about the resolver.
type InfoResponse struct {
	MaxBodyBytes         int64 `json:"maxBodyBytes"`
	DefaultPageLimit     int64 `json:"defaultPageLimit"`
	MaxPageLimit         int64 `json:"maxPageLimit"`
	MaxBatchItems        int64 `json:"maxBatchItems"`
	MaxAutocompleteUsers int64 `json:"maxAutocompleteUsers,omitzero"`
	Authentication       bool  `json:"authentication,omitzero"`
}

// MetaResponse provides optional public meta information about the server.
// All fields are optional; when present they must be non-empty.
type MetaResponse struct {
	About    string `json:"about,omitempty"`
	Comment  string `json:"comment,omitempty"`
	Software string `json:"software,omitempty"`
	Version  string `json:"version,omitempty"`
}
