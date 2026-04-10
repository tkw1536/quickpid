package api

// NamespaceCreateRequest is the JSON body for createNamespace.
type NamespaceCreateRequest struct {
	Name string `json:"name"`
}

// NamespaceResponse is returned for namespace operations.
type NamespaceResponse struct {
	Name        string `json:"name"`
	DateCreated string `json:"date_created"`
}

// ResourceCreateRequest is the JSON body for createResource and batchCreateResources items.
type ResourceCreateRequest struct {
	URL          string `json:"url"`
	IdInTarget   string `json:"IdInTarget"`
	TargetSystem string `json:"targetSystem"`
	Tag          string `json:"tag,omitempty"`
}

// ResourceResponse is returned for resource operations.
type ResourceResponse struct {
	PID          string `json:"pid"`
	URL          string `json:"url"`
	IdInTarget   string `json:"IdInTarget"`
	DateCreated  string `json:"date_created"`
	DateUpdated  string `json:"date_updated"`
	TargetSystem string `json:"targetSystem"`
	Tag          string `json:"tag"`
	Deleted      bool   `json:"deleted"`
}

// ResourceUpdateRequest is the JSON body for updateResource.
type ResourceUpdateRequest struct {
	URL          string `json:"url"`
	IdInTarget   string `json:"IdInTarget"`
	TargetSystem string `json:"targetSystem"`
	Tag          string `json:"tag"`
	Deleted      bool   `json:"deleted"`
}

// ListResourcesParams carries path and query parameters for listResources.
type ListResourcesParams struct {
	Namespace string
	Tag       string
}
