//spellchecker:words service
package service

//spellchecker:words context errors github quickpid backend
import (
	"context"
	"errors"
	"fmt"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
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
