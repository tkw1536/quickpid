//spellchecker:words service
package service

//spellchecker:words context errors github quickpid backend
import (
	"context"
	"errors"
	"fmt"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/can"
)

// canNamespace returns a [can.Namespace] struct representing the callers capabilities for the given namespace.
//
// When err is not nil, it is annotated with [api.DatabaseError] or [api.NamespaceNotFound].
func (s *Service) canNamespace(ctx context.Context, caller *api.Caller, namespace api.ValidNamespaceID) (can.Namespace, error) {
	// in anonymous role, everyone magically is a manager everywhere.
	if s.AnonymousMode() {
		return can.NewAnonymousModeNamespace(namespace), nil
	}

	// by default, the caller has no explicit role
	role := api.RoleNone

	// if the caller is authenticated, we should ask the backend for their explicit role.
	if caller != nil {
		var err error
		role, err = s.effectiveRole(ctx, *caller, namespace)
		if err != nil {
			return can.Namespace{}, err
		}
	}

	// and create the new can struct
	return can.NewNamespace(namespace, role, caller), nil
}

// effectiveRole returns the caller's role in namespace.
// This function should only be called in [Service.canNamespace].
//
// It can return the following errors:
//
// - [api.NamespaceNotFound]
// - [api.DatabaseError].
func (s *Service) effectiveRole(ctx context.Context, caller api.Caller, namespace api.ValidNamespaceID) (api.Role, error) {
	role, err := s.store.GetNamespaceRole(ctx, namespace, caller.Username())
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return role, api.WithErrorCode(fmt.Errorf("namespace not found: %w", err), api.NamespaceNotFound)
	}
	if err != nil {
		return role, api.WithErrorCode(fmt.Errorf("backend failed to get namespace role: %w", err), api.DatabaseError)
	}
	return role, nil
}

// canUser returns a [can.User] struct representing the callers capabilities for the current caller.
func (s *Service) canUser(caller *api.Caller) can.User {
	if s.AnonymousMode() {
		return can.NewAnonymousModeUser()
	}
	return can.NewUser(caller)
}
