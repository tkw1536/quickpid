// Package api holds type definitions for the PID Resolver API.
package api

import (
	"fmt"

	"github.com/tkw1536/quickpid/internal/strict"
	"github.com/tkw1536/quickpid/pid"

	_ "embed"
)

// NamespaceCreateRequest is the JSON body for createNamespace.
type NamespaceCreateRequest struct {
	Tag       string     `json:"tag"`
	PIDFormat pid.Format `json:"pid_format"`
}

func (r *NamespaceCreateRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		Tag       strict.Optional[strict.String] `json:"tag"`
		PIDFormat strict.Optional[pid.Format]    `json:"pid_format"`
	}
	decoded, err := strict.UnmarshalStruct[internal](data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}
	if !decoded.Tag.Present {
		return fmt.Errorf("missing required field: tag")
	}
	r.Tag = string(decoded.Tag.Value)
	if !decoded.PIDFormat.Present {
		return fmt.Errorf("missing required field: pid_format")
	}
	r.PIDFormat = decoded.PIDFormat.Value
	if err := r.PIDFormat.Validate(); err != nil {
		return fmt.Errorf("invalid pid format: %w", err)
	}

	return nil
}

// NamespaceResponse is returned for namespace operations.
type NamespaceResponse struct {
	ID          string     `json:"id"`
	Tag         string     `json:"tag"`
	PIDFormat   pid.Format `json:"pid_format"`
	DateCreated string     `json:"date_created"`
}

type PaginatedNamespacesResponse struct {
	Total  int `json:"total"`
	Offset int `json:"offset"`

	Items []NamespaceResponse `json:"items"`
}

// ResourceCreateRequest is the JSON body for createResource and batchCreateResources items.
type ResourceCreateRequest struct {
	URL      string  `json:"url"`
	Metadata *string `json:"metadata"`
	Tag      string  `json:"tag"`
}

func (r *ResourceCreateRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		URL      strict.Optional[strict.String] `json:"url"`
		Metadata strict.Optional[*string]       `json:"metadata"`
		Tag      strict.Optional[strict.String] `json:"tag"`
	}
	decoded, err := strict.UnmarshalStruct[internal](data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}

	if !decoded.URL.Present {
		return fmt.Errorf("missing required field: url")
	}
	r.URL = string(decoded.URL.Value)

	if !decoded.Metadata.Present {
		return fmt.Errorf("missing required field: metadata")
	}
	r.Metadata = decoded.Metadata.Value

	if !decoded.Tag.Present {
		return fmt.Errorf("missing required field: tag")
	}
	r.Tag = string(decoded.Tag.Value)

	return nil
}

// ResourceResponse is returned for resource operations.
type ResourceResponse struct {
	PID         string  `json:"pid"`
	URL         string  `json:"url"`
	Metadata    *string `json:"metadata"`
	DateCreated string  `json:"date_created"`
	DateUpdated string  `json:"date_updated"`
	Tag         string  `json:"tag"`
	Deleted     bool    `json:"deleted"`
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
// A nil pointer indicates that no update should be performed on that field.
type ResourceUpdateRequest struct {
	URL      *string  `json:"url"`
	Metadata **string `json:"metadata"`
	Tag      *string  `json:"tag"`
	Deleted  *bool    `json:"deleted"`
}

func (r *ResourceUpdateRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		URL      strict.Optional[strict.String] `json:"url"`
		Metadata strict.Optional[*string]       `json:"metadata"`
		Tag      strict.Optional[strict.String] `json:"tag"`
		Deleted  strict.Optional[strict.Bool]   `json:"deleted"`
	}
	decoded, err := strict.UnmarshalStruct[internal](data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}
	r.URL = strict.OptionalStringToPointer(decoded.URL)
	r.Metadata = decoded.Metadata.ToPointer()
	r.Tag = strict.OptionalStringToPointer(decoded.Tag)
	r.Deleted = strict.OptionalBoolToPointer(decoded.Deleted)

	return nil
}

// ListResourcesParams carries path and query parameters for listResources.
type ListResourcesParams struct {
	Namespace string // namespace id from path

	Tag     *string // optionally filter by tag
	Deleted *bool   // optionally filter by deletion status

	Limit  int
	Offset int
}

type ListNamespacesParams struct {
	Tag *string // optionally filter by tag

	Limit  int
	Offset int
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// InfoResponse provides information about the resolver.
type InfoResponse struct {
	MaxBodyBytes     int64 `json:"max_body_bytes"`
	DefaultPageLimit int64 `json:"default_page_limit"`
	MaxPageLimit     int64 `json:"max_page_limit"`
	MaxBatchItems    int64 `json:"max_batch_items"`
}
