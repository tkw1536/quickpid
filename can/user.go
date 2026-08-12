package can

//spellchecker:words errors github quickpid
import (
	"errors"

	"github.com/tkw1536/quickpid/api"
)

// NewUser creates a new [User] struct for in any mode.
func NewUser(user *api.Caller) User {
	return User{
		anonymousMode: false,
		user:          user,
	}
}

func NewAnonymousModeUser() User {
	return User{
		anonymousMode: true,
	}
}

//spellchecker:words nolint err113
//nolint:err113 // these aren't dynamic, they're used once while creating a new error value
var (
	errCanUserUnauthorized  error = api.WithErrorCode(errors.New("unauthorized"), api.Unauthorized)
	errCanUserForbidden     error = api.WithErrorCode(errors.New("forbidden"), api.Forbidden)
	errCanUserAnonymousMode error = api.WithErrorCode(errors.New("anonymous mode"), api.UnavailableInAnonymousMode)
)

// User determines if a user can perform actions not associated with a specific namespace.
//
// Methods of this struct implement permission checks, returning err != nil iff the action is forbidden.
//
// The errors contain one of three possible [api.ErrorCode]s:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.UnavailableInAnonymousMode]
//
// It should only be created using [NewUser].
type User struct {
	// anonymousMode is true if the user is in anonymousMode mode.
	anonymousMode bool

	// user is the currently authenticated user.
	// If no user is authenticated, or the server is in anonymous mode, this will be nil.
	user *api.Caller
}

// Impersonate determines if a user can impersonate another user.
func (u User) Impersonate() error {
	switch {
	case u.anonymousMode:
		return errCanUserAnonymousMode
	case u.user == nil:
		return errCanUserUnauthorized
	case u.user.Superuser():
		return nil
	default:
		return errCanUserForbidden
	}
}

// SetPassword determines if a user can set or clear a password.
func (u User) SetPassword() error {
	switch {
	case u.anonymousMode:
		return errCanUserAnonymousMode
	case u.user == nil:
		return errCanUserUnauthorized
	default:
		return nil
	}
}

// CreateUser determines if a user can create a new user.
func (u User) CreateUser() error {
	switch {
	case u.anonymousMode:
		return errCanUserAnonymousMode
	case u.user == nil:
		return errCanUserUnauthorized
	case !u.user.Superuser():
		return errCanUserForbidden
	default:
		return nil
	}
}

// DeleteUser determines if a user can delete a user.
func (u User) DeleteUser() error {
	switch {
	case u.anonymousMode:
		return errCanUserAnonymousMode
	case u.user == nil:
		return errCanUserUnauthorized
	case !u.user.Superuser():
		return errCanUserForbidden
	default:
		return nil
	}
}

// ListUsers determines if a user can list all users.
func (u User) ListUsers() error {
	switch {
	case u.anonymousMode:
		return errCanUserAnonymousMode
	case u.user == nil:
		return errCanUserUnauthorized
	case !u.user.Superuser():
		return errCanUserForbidden
	default:
		return nil
	}
}

// ListOwnKeys determines if a user can list their own API keys.
func (u User) ListOwnKeys() error {
	switch {
	case u.anonymousMode:
		return errCanUserAnonymousMode
	case u.user == nil:
		return errCanUserUnauthorized
	default:
		return nil
	}
}

// IssueKey determines if a user can issue a new API key for themselves.
func (u User) IssueKey() error {
	switch {
	case u.anonymousMode:
		return errCanUserAnonymousMode
	case u.user == nil:
		return errCanUserUnauthorized
	default:
		return nil
	}
}

// RevokeKey determines if a user can revoke an API key for themselves.
func (u User) RevokeKey() error {
	switch {
	case u.anonymousMode:
		return errCanUserAnonymousMode
	case u.user == nil:
		return errCanUserUnauthorized
	default:
		return nil
	}
}

// UpdateUser determines if a user can update another user.
func (u User) UpdateUser() error {
	switch {
	case u.anonymousMode:
		return errCanUserAnonymousMode
	case u.user == nil:
		return errCanUserUnauthorized
	case u.user.Superuser():
		return nil
	default:
		return errCanUserForbidden
	}
}

// ListUserRoles determines if a user can list their own roles across all namespaces.
func (u User) ListUserRoles() error {
	switch {
	case u.anonymousMode:
		return errCanUserAnonymousMode
	case u.user == nil:
		return errCanUserUnauthorized
	default:
		return nil
	}
}

// GetMount determines if a user can resolve a mount by base URI.
func (u User) GetMount() error {
	return nil
}

// ListMounts determines if a user can list all mounts.
func (u User) ListMounts() error {
	switch {
	case u.anonymousMode:
		return nil
	case u.user == nil:
		return errCanUserUnauthorized
	case u.user.Superuser():
		return nil
	default:
		return errCanUserForbidden
	}
}

// SetMount determines if a user can set a mount by base URI.
func (u User) SetMount() error {
	switch {
	case u.anonymousMode:
		return nil
	case u.user == nil:
		return errCanUserUnauthorized
	case u.user.Superuser():
		return nil
	default:
		return errCanUserForbidden
	}
}

// DeleteMount determines if a user can delete a mount by base URI.
func (u User) DeleteMount() error {
	switch {
	case u.anonymousMode:
		return nil
	case u.user == nil:
		return errCanUserUnauthorized
	case u.user.Superuser():
		return nil
	default:
		return errCanUserForbidden
	}
}

// ListNamespaces determines if a user can list namespaces.
func (u User) ListNamespaces() error {
	switch {
	case u.anonymousMode:
		return nil
	case u.user == nil:
		return errCanUserUnauthorized
	default:
		return nil
	}
}

// CountAllResources determines if a user can count all resources across all namespaces.
func (u User) CountAllResources() error {
	return nil
}

// CreateNamespace determines if a user can create a namespace.
func (u User) CreateNamespace() error {
	switch {
	case u.anonymousMode:
		return nil
	case u.user == nil:
		return errCanUserUnauthorized
	default:
		return nil
	}
}
