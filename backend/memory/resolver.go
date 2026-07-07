package memory

//spellchecker:words context errors fmt time github bicpid backend
import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
)

var errShutdownMemoryStore = errors.New("stopped waiting for shutdown to complete")

func (s *Store) Shutdown(ctx context.Context) error {
	done := make(chan struct{}, 1)
	go func() {
		defer close(done)

		s.mu.Lock()
		defer s.mu.Unlock()

		if err := ctx.Err(); err != nil {
			return
		}

		s.users = nil
		s.namespaces = nil
		s.resources = nil
		s.permissions = nil
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%w: %w", errShutdownMemoryStore, ctx.Err())
	}
}

func (s *Store) ListNamespaces(_ context.Context, params api.ListNamespacesParams) (*api.PaginatedNamespacesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]api.NamespaceResponse, 0, len(s.namespaces))
	for _, ns := range s.namespaces {
		if params.Tag != nil && ns.Tag != *params.Tag {
			continue
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

func (s *Store) CreateNamespace(_ context.Context, namespace string, req api.NamespaceCreateRequest, owner string, now func() time.Time) (*api.NamespaceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[owner]; !exists {
		return nil, backend.ErrUserNotFound
	}
	if _, exists := s.namespaces[namespace]; exists {
		return nil, backend.ErrDuplicateNamespaceID
	}

	created := now().UTC().Format(time.RFC3339)
	ns := api.NamespaceResponse{
		ID:          namespace,
		Tag:         req.Tag,
		PIDFormat:   req.PIDFormat,
		DateCreated: created,
	}
	s.namespaces[namespace] = ns
	s.resources[namespace] = make(map[string]api.ResourceResponse)

	if s.permissions[namespace] == nil {
		s.permissions[namespace] = make(map[string]api.PermissionLevel)
	}
	s.permissions[namespace][owner] = api.PermissionLevelManager

	return &ns, nil
}

func (s *Store) GetNamespace(_ context.Context, namespace string) (*api.NamespaceResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ns, ok := s.namespaces[namespace]
	if !ok {
		return nil, backend.ErrNamespaceNotFound
	}
	return &ns, nil
}

func (s *Store) ListResources(_ context.Context, params api.ListResourcesParams) (*api.PaginatedResourcesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.namespaces[params.Namespace]; !ok {
		return nil, backend.ErrNamespaceNotFound
	}

	byPID := s.resources[params.Namespace]
	filtered := make([]api.ResourceResponse, 0, len(byPID))
	for _, r := range byPID {
		if params.Tag != nil && r.Tag != *params.Tag {
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

func (s *Store) CreateResource(_ context.Context, namespace, pid string, req api.ResourceCreateRequest, now func() time.Time) (*api.ResourceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.namespaces[namespace]; !ok {
		return nil, backend.ErrNamespaceNotFound
	}
	byPID := s.resources[namespace]
	if _, exists := byPID[pid]; exists {
		return nil, backend.ErrPIDAllocationFailed
	}
	ts := now().UTC().Format(time.RFC3339)
	res := api.ResourceResponse{
		PID:         pid,
		URL:         req.URL,
		Metadata:    req.Metadata,
		DateCreated: ts,
		DateUpdated: ts,
		Tag:         req.Tag,
		Deleted:     false,
	}
	byPID[pid] = res
	return &res, nil
}

func (s *Store) BatchCreateResources(_ context.Context, namespace string, pids []string, reqs []api.ResourceCreateRequest, now func() time.Time) ([]api.ResourceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.namespaces[namespace]; !ok {
		return nil, backend.ErrNamespaceNotFound
	}
	if len(pids) != len(reqs) {
		return nil, backend.ErrPIDAllocationFailed
	}

	byPID := s.resources[namespace]
	seen := make(map[string]struct{}, len(pids))
	for _, pid := range pids {
		if _, dup := seen[pid]; dup {
			return nil, backend.ErrPIDAllocationFailed
		}
		seen[pid] = struct{}{}
		if _, exists := byPID[pid]; exists {
			return nil, backend.ErrPIDAllocationFailed
		}
	}

	out := make([]api.ResourceResponse, 0, len(reqs))
	ts := now().UTC().Format(time.RFC3339)
	for i, req := range reqs {
		res := api.ResourceResponse{
			PID:         pids[i],
			URL:         req.URL,
			Metadata:    req.Metadata,
			DateCreated: ts,
			DateUpdated: ts,
			Tag:         req.Tag,
			Deleted:     false,
		}
		byPID[pids[i]] = res
		out = append(out, res)
	}
	return out, nil
}

func (s *Store) GetResource(_ context.Context, namespace, pid string) (*api.ResourceResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.namespaces[namespace]; !ok {
		return nil, backend.ErrNamespaceNotFound
	}
	res, ok := s.resources[namespace][pid]
	if !ok {
		return nil, backend.ErrResourceNotFound
	}
	return &res, nil
}

func (s *Store) UpdateResource(_ context.Context, namespace, pid string, req api.ResourceUpdateRequest, now func() time.Time) (*api.ResourceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.namespaces[namespace]; !ok {
		return nil, backend.ErrNamespaceNotFound
	}
	prev, ok := s.resources[namespace][pid]
	if !ok {
		return nil, backend.ErrResourceNotFound
	}

	res := prev
	if req.URL != nil {
		res.URL = *req.URL
	}
	if req.Tag != nil {
		res.Tag = *req.Tag
	}
	if req.Deleted != nil {
		res.Deleted = *req.Deleted
	}
	if req.Metadata != nil {
		res.Metadata = *req.Metadata
	}

	res.DateUpdated = now().UTC().Format(time.RFC3339)
	s.resources[namespace][pid] = res
	return &res, nil
}
