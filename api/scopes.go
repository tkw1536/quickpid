package api

//spellchecker:words errors
import (
	"errors"
	"fmt"
)

//spellchecker:words nolint err
//nolint:err113 // these aren't dynamic, they're used once while creating a new error value
var (
	errAnonymousMode     error = WithErrorCode(errors.New("unavailable in anonymous mode"), UnavailableInAnonymousMode)
	errRequireUser       error = WithErrorCode(errors.New("requires an authenticated user"), Unauthorized)
	errRequireSuperuser  error = WithErrorCode(errors.New("requires a superuser"), Forbidden)
	errRequireHigherRole error = WithErrorCode(errors.New("requires a higher role"), Forbidden)
)

// EvaluateAnonymousModeUserScope evaluates if the given scope is fulfilled in anonymous mode.
//
// When the action is not available, an error with code [UnavailableInAnonymousMode] is returned.
func EvaluateAnonymousModeUserScope(scope UserScope) error {
	definition, err := getUserActionDefinition(scope)
	if err != nil {
		// This is technically an internal error, but let's not do that.
		return WithErrorCode(err, UnavailableInAnonymousMode)
	}
	if definition.AnonymousMode {
		return nil
	}
	return errAnonymousMode
}

// EvaluateUserScope evaluates if the given scope if fulfilled for the given user.
//
// It is assumed that the server is not in anonymous mode.
//
// When the scope is not fulfilled, an error of one of the following [ErrorCode]s is returned:
//
// - [Unauthorized]
// - [Forbidden].
func EvaluateUserScope(caller *Caller, scope UserScope) error {
	definition, err := getUserActionDefinition(scope)
	if err != nil {
		return WithErrorCode(err, Unauthorized)
	}

	if err := caller.Method().AllowsUserScope(scope); err != nil {
		return WithErrorCode(fmt.Errorf("authentication method does not allow scope: %w", err), Unauthorized)
	}

	if caller == nil {
		if definition.AllowUnauthenticated {
			return nil
		}
		return errRequireUser
	}

	if !caller.Superuser() {
		if !definition.RequireSuperuser {
			return nil
		}
		return errRequireSuperuser
	}

	return nil
}

// EvaluateAnonymousModeNamespaceScope evaluates if the given scope is fulfilled for the given namespace in anonymous mode.
//
// When the action is not available, an error with code [UnavailableInAnonymousMode] is returned.
func EvaluateAnonymousModeNamespaceScope(namespace ValidNamespaceID, scope NamespaceScope) error {
	definition, err := getNamespaceScopeDefinition(scope)
	if err != nil {
		return WithErrorCode(err, UnavailableInAnonymousMode)
	}
	if definition.AnonymousMode {
		return nil
	}
	return errAnonymousMode
}

// EvaluateNamespaceScope evaluates if the given scope is fulfilled for the given namespace.
//
// When the action is not available, an error of one of the following [ErrorCode]s is returned:
//
// - [Unauthorized]
// - [Forbidden]
// - [UnavailableInAnonymousMode]
//
// It should only be created using [NewNamespace].
func EvaluateNamespaceScope(namespace ValidNamespaceID, explicitRole Role, caller *Caller, scope NamespaceScope) error {
	action, err := getNamespaceScopeDefinition(scope)
	if err != nil {
		return WithErrorCode(err, Unauthorized)
	}

	if err := caller.Method().AllowsNamespaceScope(namespace, scope); err != nil {
		return WithErrorCode(fmt.Errorf("authentication method does not allow scope: %w", err), Unauthorized)
	}

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

	if action.MinRole != RoleNone && roleAtLeast(explicitRole, action.MinRole) {
		return nil
	}

	if action.MinRole == RoleNone {
		return nil
	}

	return fmt.Errorf("%w: %s", errRequireHigherRole, action.MinRole)
}
