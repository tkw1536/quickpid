package can

//spellchecker:words errors github quickpid
import (
	"errors"

	"github.com/tkw1536/quickpid/api"
)

//spellchecker:words nolint err113
//nolint:err113 // these aren't dynamic, they're used once while creating a new error value
var (
	errCanNamespaceUnauthorized  error = api.WithErrorCode(errors.New("unauthorized"), api.Unauthorized)
	errCanNamespaceForbidden     error = api.WithErrorCode(errors.New("forbidden"), api.Forbidden)
	errCanNamespaceAnonymousMode error = api.WithErrorCode(errors.New("anonymous mode"), api.UnavailableInAnonymousMode)
)

// NewAnonymousModeNamespace creates a new [Namespace] struct for anonymous mode.
func NewAnonymousModeNamespace(namespace api.ValidNamespaceID) Namespace {
	return Namespace{
		anonymousMode: true,
		namespace:     namespace,
	}
}

// NewNamespace creates a new [Namespace] struct for authenticated mode.
func NewNamespace(namespace api.ValidNamespaceID, explicitRole api.Role, user *api.Caller) Namespace {
	return Namespace{
		namespace:    namespace,
		explicitRole: explicitRole,
		user:         user,
	}
}

// Namespace determines if a user can perform a namespace-specific action.
//
// Methods of this struct implement permission checks, returning err != nil iff the action is forbidden.
// The errors contain one of three possible [api.ErrorCode]s:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.UnavailableInAnonymousMode]
//
// It should only be created using [NewAnonymousModeNamespace] or [NewNamespace].
type Namespace struct {
	// anonymousMode is true if the server is in anonymous mode.
	anonymousMode bool

	// namespace is the current namespace the permissions should be checked for.
	namespace api.ValidNamespaceID

	// explicitRole is the role that has been explicitly assigned to the user for the given namespace.
	// This does not take into account the superuser status.
	//
	// If a user is assigned no explicit role, this will be [api.RoleNone].
	explicitRole api.Role

	// user is the currently authenticated user.
	// If no user is authenticated, or the server is in anonymous mode, this will be nil.
	user *api.Caller
}

// ReadMetadata determines if a user can read namespace metadata.
func (c Namespace) ReadMetadata() error {
	switch {
	case c.anonymousMode:
		return nil
	case c.explicitRole == api.RoleContributor || c.explicitRole == api.RoleEditor || c.explicitRole == api.RoleManager:
		return nil
	case c.user == nil:
		return errCanNamespaceUnauthorized
	case c.user.Superuser():
		return nil
	default:
		return errCanNamespaceForbidden
	}
}

// ListMounts determines if a user can list mounts for a namespace.
func (c Namespace) ListMounts() error {
	switch {
	case c.anonymousMode:
		return nil
	case c.explicitRole == api.RoleContributor || c.explicitRole == api.RoleEditor || c.explicitRole == api.RoleManager:
		return nil
	case c.user == nil:
		return errCanNamespaceUnauthorized
	case c.user.Superuser():
		return nil
	default:
		return errCanNamespaceForbidden
	}
}

// PatchMetadata determines if a user can patch namespace metadata.
func (c Namespace) PatchMetadata() error {
	switch {
	case c.anonymousMode:
		return nil
	case c.explicitRole == api.RoleManager:
		return nil
	case c.user == nil:
		return errCanNamespaceUnauthorized
	case c.user.Superuser():
		return nil
	default:
		return errCanNamespaceForbidden
	}
}

// ListResources determines if a user can list all resources in a namespace.
func (c Namespace) ListResources() error {
	switch {
	case c.anonymousMode:
		return nil
	case c.explicitRole == api.RoleEditor || c.explicitRole == api.RoleManager:
		return nil
	case c.user == nil:
		return errCanNamespaceUnauthorized
	case c.user.Superuser():
		return nil
	default:
		return errCanNamespaceForbidden
	}
}

// CreateResource determines if a user can create a resource in a namespace.
func (c Namespace) CreateResource() error {
	switch {
	case c.anonymousMode:
		return nil
	case c.explicitRole == api.RoleContributor || c.explicitRole == api.RoleEditor || c.explicitRole == api.RoleManager:
		return nil
	case c.user == nil:
		return errCanNamespaceUnauthorized
	case c.user.Superuser():
		return nil
	default:
		return errCanNamespaceForbidden
	}
}

// BatchCreateResources determines if a user can batch create resources in a namespace.
func (c Namespace) BatchCreateResources() error {
	return c.CreateResource()
}

// GetResource determines if a user can get a resource by namespace and pid.
func (c Namespace) GetResource() error {
	return nil
}

// SeeDeletedResource determines if a user can see an un-redacted deleted resource.
func (c Namespace) SeeDeletedResource() error {
	switch {
	case c.anonymousMode:
		return nil
	case c.explicitRole == api.RoleEditor || c.explicitRole == api.RoleManager:
		return nil
	case c.user == nil:
		return errCanNamespaceUnauthorized
	case c.user.Superuser():
		return nil
	default:
		return errCanNamespaceForbidden
	}
}

// UpdateResource determines if a user can update a resource by namespace and pid.
func (c Namespace) UpdateResource() error {
	switch {
	case c.anonymousMode:
		return nil
	case c.explicitRole == api.RoleEditor || c.explicitRole == api.RoleManager:
		return nil
	case c.user == nil:
		return errCanNamespaceUnauthorized
	case c.user.Superuser():
		return nil
	default:
		return errCanNamespaceForbidden
	}
}

// GetOwnNamespaceRole determines if a user can get their own role in a namespace.
func (c Namespace) GetOwnNamespaceRole() error {
	switch {
	case c.anonymousMode:
		return errCanNamespaceAnonymousMode
	case c.user == nil:
		return errCanNamespaceUnauthorized
	default:
		return nil
	}
}

// GetOtherUserNamespaceRole determines if a user can get another user's role in a namespace.
func (c Namespace) GetOtherUserNamespaceRole() error {
	switch {
	case c.anonymousMode:
		return errCanNamespaceAnonymousMode
	case c.explicitRole == api.RoleManager:
		return nil
	case c.user == nil:
		return errCanNamespaceUnauthorized
	case c.user.Superuser():
		return nil
	default:
		return errCanNamespaceForbidden
	}
}

// ListNamespaceRoles determines if a user can list all roles in a namespace.
func (c Namespace) ListNamespaceRoles() error {
	switch {
	case c.anonymousMode:
		return errCanNamespaceAnonymousMode
	case c.explicitRole == api.RoleManager:
		return nil
	case c.user == nil:
		return errCanNamespaceUnauthorized
	case c.user.Superuser():
		return nil
	default:
		return errCanNamespaceForbidden
	}
}

// SetOtherNamespaceRole determines if a user can set another user's role in a namespace.
func (c Namespace) SetOtherNamespaceRole() error {
	switch {
	case c.anonymousMode:
		return errCanNamespaceAnonymousMode
	case c.explicitRole == api.RoleManager:
		return nil
	case c.user == nil:
		return errCanNamespaceUnauthorized
	case c.user.Superuser():
		return nil
	default:
		return errCanNamespaceForbidden
	}
}

// SetOwnNamespaceRole determines if a user can set their own role in a namespace.
func (c Namespace) SetOwnNamespaceRole() error {
	switch {
	case c.anonymousMode:
		return errCanNamespaceAnonymousMode
	case c.user == nil:
		return errCanNamespaceUnauthorized
	case c.user.Superuser():
		return nil
	default:
		return errCanNamespaceForbidden
	}
}

// ClearOtherNamespaceRole determines if a user can clear another user's role in a namespace.
func (c Namespace) ClearOtherNamespaceRole() error {
	switch {
	case c.anonymousMode:
		return errCanNamespaceAnonymousMode
	case c.explicitRole == api.RoleManager:
		return nil
	case c.user == nil:
		return errCanNamespaceUnauthorized
	case c.user.Superuser():
		return nil
	default:
		return errCanNamespaceForbidden
	}
}

// ClearOwnNamespaceRole determines if a user can clear their own role in a namespace.
func (c Namespace) ClearOwnNamespaceRole() error {
	switch {
	case c.anonymousMode:
		return errCanNamespaceAnonymousMode
	case c.user == nil:
		return errCanNamespaceUnauthorized
	case c.user.Superuser():
		return nil
	default:
		return errCanNamespaceForbidden
	}
}
