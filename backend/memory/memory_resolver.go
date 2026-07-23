//spellchecker:words memory
package memory

//spellchecker:words context slices sort time github quickpid backend
import (
	"context"
	"slices"
	"sort"
	"time"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
)

func (s *Store) ListNamespaces(_ context.Context, user *api.ValidUsername, params api.ListNamespacesParams) (*api.PaginatedNamespacesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if user != nil {
		if _, exists := s.users[user.String()]; !exists {
			return nil, backend.ErrUserNotFound
		}
	}

	all := make([]api.NamespaceResponse, 0, len(s.namespaces))
	for id, ns := range s.namespaces {
		if params.Tag != nil && ns.Tag != *params.Tag {
			continue
		}
		if user != nil {
			byUser, ok := s.roles[id]
			if !ok {
				continue
			}
			if _, ok := byUser[user.String()]; !ok {
				continue
			}
		}
		all = append(all, ns)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })

	total := len(all)
	limit := params.Limit
	offset := params.Offset

	if offset >= total {
		return &api.PaginatedNamespacesResponse{Total: total, Offset: offset, Items: []api.NamespaceResponse{}}, nil
	}
	end := min(offset+limit, total)
	items := append([]api.NamespaceResponse(nil), all[offset:end]...)
	if items == nil {
		items = []api.NamespaceResponse{}
	}
	return &api.PaginatedNamespacesResponse{Total: total, Offset: offset, Items: items}, nil
}

func (s *Store) CreateNamespace(_ context.Context, namespace api.ValidNamespaceID, req api.NamespaceCreateRequest, owner *api.ValidUsername, now func() time.Time) (*api.NamespaceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if owner != nil {
		if _, exists := s.users[owner.String()]; !exists {
			return nil, backend.ErrUserNotFound
		}
	}
	if _, exists := s.namespaces[namespace.String()]; exists {
		return nil, backend.ErrDuplicateNamespaceID
	}

	created := now().UTC().Format(time.RFC3339)
	ns := api.NamespaceResponse{
		ID:          namespace.String(),
		Tag:         req.Tag,
		PIDFormat:   req.PIDFormat,
		DateCreated: created,
	}
	s.namespaces[namespace.String()] = ns
	s.resources[namespace.String()] = make(map[string]api.ResourceResponse)

	if owner != nil {
		if s.roles[namespace.String()] == nil {
			s.roles[namespace.String()] = make(map[string]api.Role)
		}
		s.roles[namespace.String()][owner.String()] = api.RoleManager
	}

	return &ns, nil
}

func (s *Store) GetNamespace(_ context.Context, namespace api.ValidNamespaceID) (*api.NamespaceResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ns, ok := s.namespaces[namespace.String()]
	if !ok {
		return nil, backend.ErrNamespaceNotFound
	}
	return &ns, nil
}

func (s *Store) ListResources(_ context.Context, namespace api.ValidNamespaceID, params api.ListResourcesParams) (*api.PaginatedResourcesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.namespaces[namespace.String()]; !ok {
		return nil, backend.ErrNamespaceNotFound
	}

	byPID := s.resources[namespace.String()]
	filtered := make([]api.ResourceResponse, 0, len(byPID))
	for _, r := range byPID {
		if params.Tag != nil && !slices.Contains(r.Tags, *params.Tag) {
			continue
		}
		if params.Deleted != nil && r.Deleted != *params.Deleted {
			continue
		}
		filtered = append(filtered, r)
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].PID < filtered[j].PID })

	total := len(filtered)
	limit := params.Limit
	offset := params.Offset

	if offset >= total {
		return &api.PaginatedResourcesResponse{Total: total, Offset: offset, Items: []api.ResourceResponse{}}, nil
	}
	end := min(offset+limit, total)
	items := append([]api.ResourceResponse(nil), filtered[offset:end]...)
	if items == nil {
		items = []api.ResourceResponse{}
	}
	return &api.PaginatedResourcesResponse{Total: total, Offset: offset, Items: items}, nil
}

func (s *Store) CountAllResources(_ context.Context) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var n int64
	for _, byPID := range s.resources {
		n += int64(len(byPID))
	}
	return n, nil
}

func (s *Store) CreateResource(_ context.Context, namespace api.ValidNamespaceID, pid api.ValidPID, req api.ResourceCreateRequest, now func() time.Time) (*api.ResourceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.namespaces[namespace.String()]; !ok {
		return nil, backend.ErrNamespaceNotFound
	}
	byPID := s.resources[namespace.String()]
	if _, exists := byPID[pid.String()]; exists {
		return nil, backend.ErrPIDAllocationFailed
	}
	ts := now().UTC().Format(time.RFC3339)
	res := api.ResourceResponse{
		PID:         pid.String(),
		URL:         req.URL,
		Metadata:    req.Metadata,
		DateCreated: ts,
		DateUpdated: ts,
		Tags:        append([]string(nil), req.Tags...),
		Deleted:     false,
	}
	byPID[pid.String()] = res
	return &res, nil
}

func (s *Store) BatchCreateResources(_ context.Context, namespace api.ValidNamespaceID, pids []api.ValidPID, reqs []api.ResourceCreateRequest, now func() time.Time) ([]api.ResourceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.namespaces[namespace.String()]; !ok {
		return nil, backend.ErrNamespaceNotFound
	}
	if len(pids) != len(reqs) {
		return nil, backend.ErrPIDAllocationFailed
	}

	byPID := s.resources[namespace.String()]
	seen := make(map[string]struct{}, len(pids))
	for _, pid := range pids {
		if _, dup := seen[pid.String()]; dup {
			return nil, backend.ErrPIDAllocationFailed
		}
		seen[pid.String()] = struct{}{}
		if _, exists := byPID[pid.String()]; exists {
			return nil, backend.ErrPIDAllocationFailed
		}
	}

	out := make([]api.ResourceResponse, 0, len(reqs))
	ts := now().UTC().Format(time.RFC3339)
	for i, req := range reqs {
		res := api.ResourceResponse{
			PID:         pids[i].String(),
			URL:         req.URL,
			Metadata:    req.Metadata,
			DateCreated: ts,
			DateUpdated: ts,
			Tags:        append([]string(nil), req.Tags...),
			Deleted:     false,
		}
		byPID[pids[i].String()] = res
		out = append(out, res)
	}
	return out, nil
}

func (s *Store) GetResource(_ context.Context, namespace api.ValidNamespaceID, pid api.ValidPID) (*api.ResourceResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.namespaces[namespace.String()]; !ok {
		return nil, backend.ErrNamespaceNotFound
	}
	res, ok := s.resources[namespace.String()][pid.String()]
	if !ok {
		return nil, backend.ErrResourceNotFound
	}
	return &res, nil
}

func (s *Store) UpdateResource(_ context.Context, namespace api.ValidNamespaceID, pid api.ValidPID, req api.ResourceUpdateRequest, now func() time.Time) (*api.ResourceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.namespaces[namespace.String()]; !ok {
		return nil, backend.ErrNamespaceNotFound
	}
	prev, ok := s.resources[namespace.String()][pid.String()]
	if !ok {
		return nil, backend.ErrResourceNotFound
	}

	res := prev
	if req.URL != nil {
		res.URL = *req.URL
	}
	if req.Tags != nil {
		res.Tags = append([]string(nil), req.Tags...)
	}
	if req.Deleted != nil {
		res.Deleted = *req.Deleted
	}
	if req.Metadata != nil {
		res.Metadata = *req.Metadata
	}

	res.DateUpdated = now().UTC().Format(time.RFC3339)
	s.resources[namespace.String()][pid.String()] = res
	return &res, nil
}
