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
	Tag       string     `json:"tag"`
	PIDFormat pid.Format `json:"pidFormat"`
}

var (
	errInvalidPIDFormat = errors.New("invalid PID format")
)

func (r *NamespaceCreateRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		Tag       strict.Optional[strict.String] `json:"tag"`
		PIDFormat strict.Optional[pid.Format]    `json:"pidFormat"`
	}
	decoded, err := strict.UnmarshalStruct[internal](data)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToUnmarshalFields, err)
	}
	if !decoded.Tag.Present {
		return missingRequiredFieldError("tag")
	}
	r.Tag = string(decoded.Tag.Value)
	if !decoded.PIDFormat.Present {
		return missingRequiredFieldError("pidFormat")
	}
	r.PIDFormat = decoded.PIDFormat.Value
	if err := r.PIDFormat.Validate(); err != nil {
		return fmt.Errorf("%w: %w", errInvalidPIDFormat, err)
	}

	return nil
}

// NamespaceResponse is returned for namespace operations.
type NamespaceResponse struct {
	ID          string     `json:"id"`
	Tag         string     `json:"tag"`
	PIDFormat   pid.Format `json:"pidFormat"`
	DateCreated time.Time  `json:"dateCreated"`
}

type PaginatedNamespacesResponse struct {
	Total  int `json:"total"`
	Offset int `json:"offset"`

	Items []NamespaceResponse `json:"items"`
}

// ResourceCreateRequest is the JSON body for createResource and batchCreateResources items.
type ResourceCreateRequest struct {
	URL      string
	Metadata *string
	Tags     []string
}

func (r *ResourceCreateRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		URL      strict.Optional[strict.String]      `json:"url"`
		Metadata strict.Optional[*string]            `json:"metadata"`
		Tags     strict.Optional[strict.StringSlice] `json:"tags"`
	}
	decoded, err := strict.UnmarshalStruct[internal](data)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToUnmarshalFields, err)
	}

	if !decoded.URL.Present {
		return missingRequiredFieldError("url")
	}
	r.URL = string(decoded.URL.Value)

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

// ResourceResponse is returned for resource operations.
type ResourceResponse struct {
	PID         string
	URL         string
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
		URL         string   `json:"url"`
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
type ResourceUpdateRequest struct {
	URL      *string
	Metadata **string
	Tags     []string
	Deleted  *bool
}

func (r *ResourceUpdateRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		URL      strict.Optional[strict.String]      `json:"url"`
		Metadata strict.Optional[*string]            `json:"metadata"`
		Tags     strict.Optional[strict.StringSlice] `json:"tags"`
		Deleted  strict.Optional[strict.Bool]        `json:"deleted"`
	}
	decoded, err := strict.UnmarshalStruct[internal](data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal resource update request: %w", err)
	}
	r.URL = strict.OptionalStringToPointer(decoded.URL)
	r.Metadata = decoded.Metadata.ToPointer()
	r.Deleted = strict.OptionalBoolToPointer(decoded.Deleted)

	if decoded.Tags.Present {
		r.Tags = decoded.Tags.Value.Strings()
	} else {
		r.Tags = nil
	}

	return nil
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
