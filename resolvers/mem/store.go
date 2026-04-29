package mem

import (
	"context"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/tkw1536/quickpid/backend"
)

// Store is an in-memory Resolver implementation protected by a single RWMutex.
type Store struct {
	// protects the namespace and resource maps.
	mu sync.RWMutex

	namespaces map[string]backend.NamespaceResponse
	resources  map[string]map[string]backend.ResourceResponse

	// maximum number of attempts to allocate a PID.
	// must be > 0.
	maxPIDAttempts int
}

// NewStore returns an empty Store. PID allocation is supplied per CreateResource / BatchCreateResources
func NewStore(maxPIDAttempts int) *Store {
	return &Store{
		namespaces:     make(map[string]backend.NamespaceResponse),
		resources:      make(map[string]map[string]backend.ResourceResponse),
		maxPIDAttempts: maxPIDAttempts,
	}
}

var _ backend.ResolverBackend = (*Store)(nil)

func (s *Store) ListNamespaces(_ context.Context, params backend.ListNamespacesParams) (*backend.PaginatedNamespacesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]backend.NamespaceResponse, 0, len(s.namespaces))
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
		return &backend.PaginatedNamespacesResponse{Total: total, Offset: offset, Items: []backend.NamespaceResponse{}}, nil
	}
	end := min(offset+limit, total)
	items := append([]backend.NamespaceResponse(nil), all[offset:end]...)
	return &backend.PaginatedNamespacesResponse{Total: total, Offset: offset, Items: items}, nil
}

func (s *Store) CreateNamespace(_ context.Context, namespace string, req backend.NamespaceCreateRequest, now func() time.Time) (*backend.NamespaceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.namespaces[namespace]; exists {
		return nil, backend.ErrNamespaceIDAllocationFailed
	}
	created := now().UTC().Format(time.RFC3339)
	ns := backend.NamespaceResponse{
		ID:          namespace,
		Tag:         req.Tag,
		PIDFormat:   req.PIDFormat,
		DateCreated: created,
	}
	s.namespaces[namespace] = ns
	s.resources[namespace] = make(map[string]backend.ResourceResponse)
	return &ns, nil
}

func (s *Store) ListResources(_ context.Context, params backend.ListResourcesParams) (*backend.PaginatedResourcesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.namespaces[params.Namespace]; !ok {
		return nil, backend.ErrNamespaceNotFound
	}

	byPID := s.resources[params.Namespace]
	filtered := make([]backend.ResourceResponse, 0, len(byPID))
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
		return &backend.PaginatedResourcesResponse{Total: total, Offset: offset, Items: []backend.ResourceResponse{}}, nil
	}
	end := min(offset+limit, total)
	items := append([]backend.ResourceResponse(nil), filtered[offset:end]...)
	return &backend.PaginatedResourcesResponse{Total: total, Offset: offset, Items: items}, nil
}

func (s *Store) CreateResource(_ context.Context, namespace string, req backend.ResourceCreateRequest, rand io.Reader, now func() time.Time) (*backend.ResourceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ns, ok := s.namespaces[namespace]
	if !ok {
		return nil, backend.ErrNamespaceNotFound
	}
	byPID := s.resources[namespace]
	for attempt := 0; attempt < s.maxPIDAttempts; attempt++ {
		candidate, err := ns.PIDFormat.Generate(rand)
		if err != nil {
			return nil, err
		}
		if _, exists := byPID[candidate]; exists {
			continue
		}
		ts := now().UTC().Format(time.RFC3339)
		res := backend.ResourceResponse{
			PID:         candidate,
			URL:         req.URL,
			Metadata:    req.Metadata,
			DateCreated: ts,
			DateUpdated: ts,
			Tag:         req.Tag,
			Deleted:     false,
		}
		byPID[candidate] = res
		return &res, nil
	}
	return nil, backend.ErrPIDAllocationFailed
}

func (s *Store) BatchCreateResources(_ context.Context, namespace string, reqs []backend.ResourceCreateRequest, rand io.Reader, now func() time.Time) ([]backend.ResourceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ns, ok := s.namespaces[namespace]
	if !ok {
		return nil, backend.ErrNamespaceNotFound
	}

	byPID := s.resources[namespace]
	out := make([]backend.ResourceResponse, 0, len(reqs))
	for _, req := range reqs {
		var inserted bool
		for attempt := 0; attempt < s.maxPIDAttempts; attempt++ {
			candidate, err := ns.PIDFormat.Generate(rand)
			if err != nil {
				return nil, err
			}
			if _, exists := byPID[candidate]; exists {
				continue
			}
			ts := now().UTC().Format(time.RFC3339)
			res := backend.ResourceResponse{
				PID:         candidate,
				URL:         req.URL,
				Metadata:    req.Metadata,
				DateCreated: ts,
				DateUpdated: ts,
				Tag:         req.Tag,
				Deleted:     false,
			}
			byPID[candidate] = res
			out = append(out, res)
			inserted = true
			break
		}
		if !inserted {
			return nil, backend.ErrPIDAllocationFailed
		}
	}
	return out, nil
}

func (s *Store) GetResource(_ context.Context, namespace, pid string) (*backend.ResourceResponse, error) {
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

func (s *Store) UpdateResource(_ context.Context, namespace, pid string, req backend.ResourceUpdateRequest, now func() time.Time) (*backend.ResourceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.namespaces[namespace]; !ok {
		return nil, backend.ErrNamespaceNotFound
	}
	prev, ok := s.resources[namespace][pid]
	if !ok {
		return nil, backend.ErrResourceNotFound
	}
	updated := now().UTC().Format(time.RFC3339)
	res := backend.ResourceResponse{
		PID:         prev.PID,
		URL:         req.URL,
		Metadata:    req.Metadata,
		DateCreated: prev.DateCreated,
		DateUpdated: updated,
		Tag:         req.Tag,
		Deleted:     req.Deleted,
	}
	s.resources[namespace][pid] = res
	return &res, nil
}
