package api

//spellchecker:words errors github bicpid internal strict
import (
	"errors"
	"fmt"

	"github.com/tkw1536/bicpid/internal/strict"
)

// PermissionLevel is a user's permission level within a namespace.
type PermissionLevel string

const (
	PermissionLevelNone        PermissionLevel = "none"
	PermissionLevelContributor PermissionLevel = "contributor"
	PermissionLevelEditor      PermissionLevel = "editor"
	PermissionLevelManager     PermissionLevel = "manager"
)

var errInvalidPermissionLevel = errors.New("invalid permission level")

// Valid reports whether the level is a known permission level.
func (level PermissionLevel) Validate() error {
	switch level {
	case PermissionLevelNone, PermissionLevelContributor, PermissionLevelEditor, PermissionLevelManager:
		return nil
	default:
		return errInvalidPermissionLevel
	}
}

// NamespacePermission describes a user's permission level in a namespace.
type NamespacePermission struct {
	Username string          `json:"username"`
	Level    PermissionLevel `json:"level"`
}

// SetNamespacePermissionRequest is the JSON body for setNamespacePermission.
type SetNamespacePermissionRequest struct {
	Level PermissionLevel `json:"level"`
}

func (r *SetNamespacePermissionRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		Level strict.Optional[strict.String] `json:"level"`
	}
	decoded, err := strict.UnmarshalStruct[internal](data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal fields: %w", err)
	}
	if !decoded.Level.Present {
		return fmt.Errorf("missing required field: level")
	}
	r.Level = PermissionLevel(decoded.Level.Value)
	if !(r.Level.Validate() == nil) {
		return fmt.Errorf("invalid permission level: %q", r.Level)
	}
	return nil
}

type ListNamespacePermissionsParams struct {
	Limit  int
	Offset int
}

type PaginatedNamespacePermissionsResponse struct {
	Total  int                   `json:"total"`
	Offset int                   `json:"offset"`
	Items  []NamespacePermission `json:"items"`
}
