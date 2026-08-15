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

// ExplicitNamespaceRole returns the user's explicit role in a namespace for scope checks.
// Returns [api.RoleNone] when no role is stored.
//
// It can return the following errors:
//
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) ExplicitNamespaceRole(ctx context.Context, namespace api.ValidNamespaceID, username api.ValidUsername) (api.Role, error) {
	role, err := s.store.GetNamespaceRole(ctx, namespace, username)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return api.RoleNone, api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return api.RoleNone, api.WithErrorCode(fmt.Errorf("backend failed to get namespace role: %w", err), api.DatabaseError)
	}
	return role, nil
}

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
// It should only be called from [Service.GetResource] for [scopes.ScopeSeeDeletedResource].
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

		role, err = s.ExplicitNamespaceRole(ctx, namespace, caller.Username())
		if err != nil {
			return fmt.Errorf("scope %q not fulfilled for namespace %q: %w", scope, namespace.String(), err)
		}
	}

	if err := scopes.EvaluateNamespaceScope(namespace, role, caller, scope); err != nil {
		return fmt.Errorf("scope %q not fulfilled for namespace %q: %w", scope, namespace.String(), err)
	}
	return nil
}
