package api

//spellchecker:words encoding json errors regexp time github quickpid internal strict
import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/tkw1536/quickpid/internal/strict"
	"github.com/tkw1536/quickpid/pid"
)

// ValidNamespaceID represents a valid namespace id.
// The zero value is not valid.
//
// Use [NewNamespaceID] to create a new namespace id.
type ValidNamespaceID struct {
	valid bool
	value string
}

// String returns the namespace id as a string.
func (ns ValidNamespaceID) String() string {
	if !ns.valid {
		panic("invalid namespace id")
	}
	return ns.value
}

// regular expressions to validate various identifiers.
var (
	namespaceIDRE = regexp.MustCompile(`^[a-z0-9_-]+$`)

	// ErrInvalidNamespaceID is returned when a namespace id fails format validation.
	ErrInvalidNamespaceID = errors.New("invalid namespace id")
)

// NewNamespaceID creates a new NamespaceID.
func NewNamespaceID(value string) (ValidNamespaceID, error) {
	if !namespaceIDRE.MatchString(value) {
		return ValidNamespaceID{}, ErrInvalidNamespaceID
	}
	return ValidNamespaceID{valid: true, value: value}, nil
}

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

type ListNamespacesParams struct {
	Tag *string // optionally filter by tag

	Limit  int
	Offset int
}
