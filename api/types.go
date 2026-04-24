package api

// NamespaceCreateRequest is the JSON body for createNamespace.
type NamespaceCreateRequest struct {
	Name      string    `json:"name"`
	PIDFormat PIDFormat `json:"pid_format"`
}

// NamespaceResponse is returned for namespace operations.
type NamespaceResponse struct {
	Name        string    `json:"name"`
	PIDFormat   PIDFormat `json:"pid_format"`
	DateCreated string    `json:"date_created"`
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

// ListResourcesParams carries path and query parameters for listResources.
type ListResourcesParams struct {
	Namespace string

	Tag     *string // optionally filter by tag
	Deleted *bool   // optionally filter by deletion status

	Limit  int
	Offset int
}

type ListNamespacesParams struct {
	Limit  int
	Offset int
}

type ErrorResponse struct {
	Error string `json:"error"`
}
