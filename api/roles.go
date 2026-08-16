package api

//spellchecker:words errors github quickpid internal strict
import (
	"errors"
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

var (
	errInvalidRole = errors.New("invalid role")
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

// rank returns the rank of the role, which can be used to compare roles.
func (role Role) rank() int {
	switch role {
	case RoleContributor:
		return 1
	case RoleEditor:
		return 2
	case RoleManager:
		return 3
	case RoleNone:
		return 0
	}
	return -1
}

// LessThan reports if this role is less than or equal to the other role.
func (role Role) LessThan(other Role) bool {
	return role.rank() <= other.rank()
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
	decoded, err := strict.UnmarshalStrict[internal](data)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToUnmarshalFields, err)
	}
	if !decoded.Role.Present {
		return missingRequiredFieldError("role")
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
