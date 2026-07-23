package api

//spellchecker:words github quickpid internal strict
import (
	"fmt"

	"github.com/tkw1536/quickpid/internal/strict"
)

// Role is a user's role within a namespace.
type Role string

const (
	RoleNone        Role = "none"
	RoleContributor Role = "contributor"
	RoleEditor      Role = "editor"
	RoleManager     Role = "manager"
)

// Valid reports whether the value is a known role.
func (role Role) Validate() error {
	switch role {
	case RoleNone, RoleContributor, RoleEditor, RoleManager:
		return nil
	default:
		return errInvalidRole
	}
}

// NamespaceRole describes a user's role in a namespace.
type NamespaceRole struct {
	Username string `json:"username"`
	Role     Role   `json:"role"`
}

// SetNamespaceRoleRequest is the JSON body for setNamespaceRole.
type SetNamespaceRoleRequest struct {
	Role Role `json:"role"`
}

func (r *SetNamespaceRoleRequest) UnmarshalJSON(data []byte) error {
	type internal struct {
		Role strict.Optional[strict.String] `json:"role"`
	}
	decoded, err := strict.UnmarshalStruct[internal](data)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToUnmarshalFields, err)
	}
	if !decoded.Role.Present {
		return fmt.Errorf("%w: role", errMissingRequiredField)
	}
	r.Role = Role(decoded.Role.Value)
	if r.Role.Validate() != nil {
		return fmt.Errorf("%w: %q", errInvalidRole, r.Role)
	}
	return nil
}

type ListNamespaceRolesParams struct {
	Limit  int
	Offset int
}

type PaginatedNamespaceRolesResponse struct {
	Total  int             `json:"total"`
	Offset int             `json:"offset"`
	Items  []NamespaceRole `json:"items"`
}

// UserRole describes a user's role in a specific namespace.
type UserRole struct {
	Namespace string `json:"namespace"`
	Role      Role   `json:"role"`
}

type ListUserRolesParams struct {
	Limit  int
	Offset int
}

type PaginatedUserRolesResponse struct {
	Total  int        `json:"total"`
	Offset int        `json:"offset"`
	Items  []UserRole `json:"items"`
}
