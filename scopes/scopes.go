//spellchecker:words scopes
package scopes

//spellchecker:words errors github quickpid
import (
	"errors"
	"fmt"

	"github.com/tkw1536/quickpid/api"
)

//spellchecker:words nolint err
//nolint:err113 // these aren't dynamic, they're used once while creating a new error value
var (
	errAnonymousMode     error = api.WithErrorCode(errors.New("unavailable in anonymous mode"), api.UnavailableInAnonymousMode)
	errRequireUser       error = api.WithErrorCode(errors.New("requires an authenticated user"), api.Unauthorized)
	errRequireSuperuser  error = api.WithErrorCode(errors.New("requires a superuser"), api.Forbidden)
	errRequireHigherRole error = api.WithErrorCode(errors.New("requires a higher role"), api.Forbidden)
)

// EvaluateAnonymousModeUserScope evaluates if the given scope is fulfilled in anonymous mode.
//
// When the action is not available, an error with code [api.UnavailableInAnonymousMode] is returned.
func EvaluateAnonymousModeUserScope(scope UserScope) error {
	definition := getUserActionDefinition(scope)
	if definition.AnonymousMode {
		return nil
	}
	return errAnonymousMode
}

// EvaluateUserScope evaluates if the given scope if fulfilled for the given user.
//
// It is assumed that the server is not in anonymous mode.
//
// When the scope is not fulfilled, an error of one of the following [api.ErrorCode]s is returned:
//
// - [api.Unauthorized]
// - [api.Forbidden].
func EvaluateUserScope(user *api.Caller, scope UserScope) error {
	definition := getUserActionDefinition(scope)

	if user == nil {
		if definition.AllowUnauthenticated {
			return nil
		}
		return errRequireUser
	}

	if !user.Superuser() {
		if !definition.RequireSuperuser {
			return nil
		}
		return errRequireSuperuser
	}

	return nil
}

// EvaluateAnonymousModeNamespaceScope evaluates if the given scope is fulfilled for the given namespace in anonymous mode.
//
// When the action is not available, an error with code [api.UnavailableInAnonymousMode] is returned.
func EvaluateAnonymousModeNamespaceScope(namespace api.ValidNamespaceID, scope NamespaceScope) error {
	definition := getNamespace(scope)
	if definition.AnonymousMode {
		return nil
	}
	return errAnonymousMode
}

// EvaluateNamespaceScope evaluates if the given scope is fulfilled for the given namespace.
//
// When the action is not available, an error of one of the following [api.ErrorCode]s is returned:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.UnavailableInAnonymousMode]
//
// It should only be created using [NewNamespace].
func EvaluateNamespaceScope(namespace api.ValidNamespaceID, explicitRole api.Role, caller *api.Caller, scope NamespaceScope) error {
	action := getNamespace(scope)

	if caller == nil {
		if action.AllowUnauthenticated {
			return nil
		}
		return errRequireUser
	}

	if caller.Superuser() {
		return nil
	}

	if action.RequireSuperuser {
		return errRequireSuperuser
	}

	if action.MinRole != api.RoleNone && roleAtLeast(explicitRole, action.MinRole) {
		return nil
	}

	if action.MinRole == api.RoleNone {
		return nil
	}

	return fmt.Errorf("%w: %s", errRequireHigherRole, action.MinRole)
}
