package api

//spellchecker:words github quickpid internal strict
import (
	"fmt"

	"github.com/tkw1536/quickpid/internal/strict"
)

// MountResponse is returned for mount operations.
type MountResponse struct {
	BaseURI   string `json:"baseUri"`
	Namespace string `json:"namespace"`
}

// MountUpsertRequest is the JSON body for setMount.
type MountUpsertRequest struct {
	Namespace string `json:"namespace"`
}

func (r *MountUpsertRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		Namespace strict.Optional[strict.String] `json:"namespace"`
	}
	decoded, err := strict.UnmarshalStruct[internal](data)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToUnmarshalFields, err)
	}
	if !decoded.Namespace.Present {
		return missingRequiredFieldError("namespace")
	}
	r.Namespace = string(decoded.Namespace.Value)
	return nil
}

// Validate checks if the given request is valid.
func (r *MountUpsertRequest) Validate() (ValidMountUpsertRequest, error) {
	namespace, err := NewNamespaceID(r.Namespace)
	if err != nil {
		return ValidMountUpsertRequest{}, err
	}
	return ValidMountUpsertRequest{Namespace: namespace}, nil
}

// ValidMountUpsertRequest is like a [MountUpsertRequest] but with a validated namespace id.
type ValidMountUpsertRequest struct {
	Namespace ValidNamespaceID
}

// ListMountsParams holds pagination for listing all mounts.
type ListMountsParams struct {
	Limit  int
	Offset int
}

// PaginatedMountsResponse is a paginated list of mounts.
type PaginatedMountsResponse struct {
	Total  int             `json:"total"`
	Offset int             `json:"offset"`
	Items  []MountResponse `json:"items"`
}

// ListNamespaceMountsParams holds pagination for listing mounts of a namespace.
type ListNamespaceMountsParams struct {
	Limit  int
	Offset int
}

// PaginatedBaseURIResponse is a paginated list of base URIs.
type PaginatedBaseURIResponse struct {
	Total  int      `json:"total"`
	Offset int      `json:"offset"`
	Items  []string `json:"items"`
}
