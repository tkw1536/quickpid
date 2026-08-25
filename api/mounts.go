package api

//spellchecker:words errors regexp github quickpid internal strict
import (
	"errors"
	"fmt"
	"net/url"
	"regexp"

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

// MountGrant is a stored mount-management grant for a user.
type MountGrant struct {
	Pattern string `json:"pattern"`
}

// MountGrantRequest is the JSON body for setMountGrant.
type MountGrantRequest struct {
	Pattern string `json:"pattern"`
}

func (r *MountGrantRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		Pattern strict.Optional[strict.String] `json:"pattern"`
	}
	decoded, err := strict.UnmarshalStrict[internal](data)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToUnmarshalFields, err)
	}
	if !decoded.Pattern.Present {
		return missingRequiredFieldError("pattern")
	}
	r.Pattern = string(decoded.Pattern.Value)
	return nil
}

var errInvalidMountGrantPattern = errors.New("invalid mount grant pattern")

// Validate compiles the pattern into a [ValidMountGrantRequest].
func (r *MountGrantRequest) Validate() (ValidMountGrantRequest, error) {
	return NewMountGrantPattern(r.Pattern)
}

// ValidMountGrantRequest is a validated mount grant holding a compiled Go regexp.
type ValidMountGrantRequest struct {
	Regexp *regexp.Regexp
}

// String returns the original pattern string.
func (r ValidMountGrantRequest) String() string {
	if r.Regexp == nil {
		panic("invalid mount grant request")
	}
	return r.Regexp.String()
}

// MatchString reports whether the grant matches the given base URI string.
func (r ValidMountGrantRequest) MatchString(baseURI string) bool {
	if r.Regexp == nil {
		panic("invalid mount grant request")
	}
	return r.Regexp.MatchString(baseURI)
}

// NewMountGrantPattern compiles a Go regexp pattern for a mount grant.
func NewMountGrantPattern(pattern string) (ValidMountGrantRequest, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return ValidMountGrantRequest{}, fmt.Errorf("%w: %w", errInvalidMountGrantPattern, err)
	}
	return ValidMountGrantRequest{Regexp: re}, nil
}

// ListUserMountGrantsParams holds pagination for listing a user's mount grants.
type ListUserMountGrantsParams struct {
	Limit  int
	Offset int
}

// PaginatedMountGrantsResponse is a paginated list of mount grant patterns.
type PaginatedMountGrantsResponse struct {
	Total  int          `json:"total"`
	Offset int          `json:"offset"`
	Items  []MountGrant `json:"items"`
}
