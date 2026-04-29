package backend

import (
	"encoding/json"

	"github.com/tkw1536/quickpid/internal/required"
	"github.com/tkw1536/quickpid/pid"
)

// NamespaceCreateRequest is the JSON body for createNamespace.
type NamespaceCreateRequest struct {
	Tag       string     `json:"tag"`
	PIDFormat pid.Format `json:"pid_format"`
}

func (r *NamespaceCreateRequest) UnmarshalJSON(data []byte) error {
	if err := required.Required(data, "tag", "pid_format"); err != nil {
		return err
	}

	type alias NamespaceCreateRequest
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*r = NamespaceCreateRequest(out)
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
	if err := required.Required(data, "url", "metadata", "tag"); err != nil {
		return err
	}

	type alias ResourceCreateRequest
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*r = ResourceCreateRequest(out)
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
	if err := required.Required(data, "url", "metadata", "tag", "deleted"); err != nil {
		return err
	}

	type alias ResourceUpdateRequest
	var out alias
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	*r = ResourceUpdateRequest(out)
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
