package api

//spellchecker:words errors github quickpid internal strict
import (
	"errors"
	"fmt"
	"net/url"

	"github.com/tkw1536/quickpid/internal/strict"
)

// ValidBaseURI represents a valid absolute base URI for a namespace mount.
// The zero value is not valid.
//
// Use [NewBaseURI] to create a new base URI.
type ValidBaseURI struct {
	valid bool
	value string
}

// String returns the base URI as a string.
func (u ValidBaseURI) String() string {
	if !u.valid {
		panic("invalid base uri")
	}
	return u.value
}

var (
	errNotAbsoluteURI   = errors.New("not an absolute URI")
	errFragmentNotEmpty = errors.New("fragment is not empty")
	errRawQueryNotEmpty = errors.New("raw query is not empty")
)

// NewBaseURI creates a new BaseURI.
// The value must be a valid absolute URI as accepted by [url.ParseRequestURI] with a non-empty scheme.
func NewBaseURI(value string) (ValidBaseURI, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return ValidBaseURI{}, fmt.Errorf("failed to parse as url: %w", err)
	}
	if !parsed.IsAbs() {
		return ValidBaseURI{}, errNotAbsoluteURI
	}
	if parsed.Fragment != "" {
		return ValidBaseURI{}, errFragmentNotEmpty
	}
	if parsed.RawQuery != "" {
		return ValidBaseURI{}, errRawQueryNotEmpty
	}
	return ValidBaseURI{valid: true, value: value}, nil
}

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
	decoded, err := strict.UnmarshalStrict[internal](data)
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
