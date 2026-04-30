// Package spec holds type definitions for the PID Resolver API.
package spec

import (
	"encoding/json"
	"fmt"

	"github.com/tkw1536/quickpid/internal/optional"
	"github.com/tkw1536/quickpid/pid"

	_ "embed"
)

// NamespaceCreateRequest is the JSON body for createNamespace.
type NamespaceCreateRequest struct {
	Tag       string     `json:"tag"`
	PIDFormat pid.Format `json:"pid_format"`
}

func (r *NamespaceCreateRequest) UnmarshalJSON(data []byte) error {
	var internal struct {
		Tag       optional.Optional[string]     `json:"tag"`
		PIDFormat optional.Optional[pid.Format] `json:"pid_format"`
	}
	if err := json.Unmarshal(data, &internal); err != nil {
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}
	if !internal.Tag.Present {
		return fmt.Errorf("missing required field: tag")
	}
	r.Tag = internal.Tag.Value
	if !internal.PIDFormat.Present {
		return fmt.Errorf("missing required field: pid_format")
	}
	r.PIDFormat = internal.PIDFormat.Value

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
	var internal struct {
		URL      optional.Optional[string]  `json:"url"`
		Metadata optional.Optional[*string] `json:"metadata"`
		Tag      optional.Optional[string]  `json:"tag"`
	}
	if err := json.Unmarshal(data, &internal); err != nil {
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}

	if !internal.URL.Present {
		return fmt.Errorf("missing required field: url")
	}
	r.URL = internal.URL.Value

	if !internal.Metadata.Present {
		return fmt.Errorf("missing required field: metadata")
	}
	r.Metadata = internal.Metadata.Value

	if !internal.Tag.Present {
		return fmt.Errorf("missing required field: tag")
	}
	r.Tag = internal.Tag.Value

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

// ResourceUpdateRequest is the JSON body for updateResource.
type ResourceUpdateRequest struct {
	URL      string  `json:"url"`
	Metadata *string `json:"metadata"`
	Tag      string  `json:"tag"`
	Deleted  bool    `json:"deleted"`
}

func (r *ResourceUpdateRequest) UnmarshalJSON(data []byte) error {
	var internal struct {
		URL      optional.Optional[string]  `json:"url"`
		Metadata optional.Optional[*string] `json:"metadata"`
		Tag      optional.Optional[string]  `json:"tag"`
		Deleted  optional.Optional[bool]    `json:"deleted"`
	}
	if err := json.Unmarshal(data, &internal); err != nil {
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}

	if !internal.URL.Present {
		return fmt.Errorf("missing required field: url")
	}
	r.URL = internal.URL.Value

	if !internal.Metadata.Present {
		return fmt.Errorf("missing required field: metadata")
	}
	r.Metadata = internal.Metadata.Value

	if !internal.Tag.Present {
		return fmt.Errorf("missing required field: tag")
	}
	r.Tag = internal.Tag.Value

	if !internal.Deleted.Present {
		return fmt.Errorf("missing required field: deleted")
	}
	r.Deleted = internal.Deleted.Value

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

//go:embed openapi.yaml
var apiSpec string

// Spec returns the api specification as a string.
func Spec() string {
	return apiSpec
}
