//spellchecker:words service
package service

//spellchecker:words context errors github quickpid backend scopes
import (
	"context"
	"errors"
	"fmt"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/scopes"
)

// checkNamespaceScope evaluates if the given action is available for the given namespace.
//
// When the action is not available, an error of one of the following [api.ErrorCode]s is returned:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.UnavailableInAnonymousMode]
// - [api.NamespaceNotFound]
// - [api.DatabaseError]
//
// It should only be called in [Service.checkNamespaceScope].
func (s *Service) checkNamespaceScope(ctx context.Context, namespace api.ValidNamespaceID, caller *api.Caller, scope scopes.NamespaceScope) error {
	if s.AnonymousMode() {
		if err := scopes.EvaluateAnonymousModeNamespaceScope(namespace, scope); err != nil {
			return fmt.Errorf("scope %q not fulfilled for namespace %q: %w", scope, namespace.String(), err)
		}
		return nil
	}

	// by default, the caller has no explicit role
	role := api.RoleNone

	// if the caller is authenticated, we should ask the backend for their explicit role.
	if caller != nil {
		var err error

		role, err = s.store.GetNamespaceRole(ctx, namespace, caller.Username())
		if errors.Is(err, backend.ErrNamespaceNotFound) {
			return api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
		}
		if err != nil {
			return api.WithErrorCode(fmt.Errorf("backend failed to get namespace role: %w", err), api.DatabaseError)
		}
	}

	if err := scopes.EvaluateNamespaceScope(namespace, role, caller, scope); err != nil {
		return fmt.Errorf("scope %q not fulfilled for namespace %q: %w", scope, namespace.String(), err)
	}
	return nil
}

// checkUserScope evaluates if the given action is available for the given user.
//
// When the action is not available, an error of one of the following [api.ErrorCode]s is returned:
//
// - [api.Unauthorized]
// - [api.Forbidden]
// - [api.UnavailableInAnonymousMode].
func (s *Service) checkUserScope(caller *api.Caller, scope scopes.UserScope) error {
	if s.AnonymousMode() {
		if err := scopes.EvaluateAnonymousModeUserScope(scope); err != nil {
			return fmt.Errorf("scope %q not fulfilled: %w", scope, err)
		}
		return nil
	}
	if err := scopes.EvaluateUserScope(caller, scope); err != nil {
		return fmt.Errorf("scope %q not fulfilled for user: %w", scope, err)
	}
	return nil
}
