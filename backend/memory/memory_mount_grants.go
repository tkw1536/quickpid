//spellchecker:words memory
package memory

//spellchecker:words context sort github quickpid backend
import (
	"context"
	"sort"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
)

func (s *MemoryBackend) ListUserMountGrants(_ context.Context, username api.ValidUsername, params api.ListUserMountGrantsParams) (*api.PaginatedMountGrantsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	usernameString := username.String()
	if _, exists := s.users[usernameString]; !exists {
		return nil, backend.ErrUserNotFound
	}

	patterns := s.mountGrants[usernameString]
	all := make([]api.MountGrant, 0, len(patterns))
	for pattern := range patterns {
		all = append(all, api.MountGrant{Pattern: pattern})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Pattern < all[j].Pattern })

	total := len(all)
	limit := params.Limit
	offset := params.Offset

	if offset >= total {
		return &api.PaginatedMountGrantsResponse{Total: total, Offset: offset, Items: []api.MountGrant{}}, nil
	}
	end := min(offset+limit, total)
	items := append([]api.MountGrant(nil), all[offset:end]...)
	if items == nil {
		items = []api.MountGrant{}
	}
	return &api.PaginatedMountGrantsResponse{Total: total, Offset: offset, Items: items}, nil
}

func (s *MemoryBackend) SetMountGrant(_ context.Context, username api.ValidUsername, req api.ValidMountGrantRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	usernameString := username.String()
	if _, exists := s.users[usernameString]; !exists {
		return backend.ErrUserNotFound
	}

	if s.mountGrants[usernameString] == nil {
		s.mountGrants[usernameString] = make(map[string]struct{})
	}
	s.mountGrants[usernameString][req.String()] = struct{}{}
	return nil
}

func (s *MemoryBackend) DeleteMountGrant(_ context.Context, username api.ValidUsername, req api.ValidMountGrantRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	usernameString := username.String()
	if _, exists := s.users[usernameString]; !exists {
		return backend.ErrUserNotFound
	}

	patterns, ok := s.mountGrants[usernameString]
	if !ok {
		return backend.ErrMountGrantNotFound
	}
	pattern := req.String()
	if _, ok := patterns[pattern]; !ok {
		return backend.ErrMountGrantNotFound
	}
	delete(patterns, pattern)
	if len(patterns) == 0 {
		delete(s.mountGrants, usernameString)
	}
	return nil
}
