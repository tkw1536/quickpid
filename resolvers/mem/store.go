package mem

import (
	"context"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/tkw1536/quickpid/api"
)

// Store is an in-memory Resolver implementation protected by a single RWMutex.
type Store struct {
	// protects the namespace and resource maps.
	mu sync.RWMutex

	// holds the namespaces and actual resources
	namespaces map[string]api.NamespaceResponse
	resources  map[string]map[string]api.ResourceResponse

	// maximum number of attempts to allocate a PID.
	// must be > 0.
	maxPIDAttempts int
}

// NewStore returns an empty Store. PID allocation is supplied per CreateResource / BatchCreateResources
func NewStore(maxPIDAttempts int) *Store {
	return &Store{
		namespaces:     make(map[string]api.NamespaceResponse),
		resources:      make(map[string]map[string]api.ResourceResponse),
		maxPIDAttempts: maxPIDAttempts,
	}
}

var _ api.Resolver = (*Store)(nil)

func (s *Store) ListNamespaces(_ context.Context, params api.ListNamespacesParams) (*api.PaginatedNamespacesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]api.NamespaceResponse, 0, len(s.namespaces))
	for _, ns := range s.namespaces {
		all = append(all, ns)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Name < all[j].Name })

	total := len(all)
	limit := params.Limit
	offset := params.Offset

	if offset >= total {
		return &api.PaginatedNamespacesResponse{Total: total, Offset: offset, Items: []api.NamespaceResponse{}}, nil
	}
	end := min(offset+limit, total)
	items := append([]api.NamespaceResponse(nil), all[offset:end]...)
	return &api.PaginatedNamespacesResponse{Total: total, Offset: offset, Items: items}, nil
}

func (s *Store) CreateNamespace(_ context.Context, req api.NamespaceCreateRequest, now func() time.Time) (*api.NamespaceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.namespaces[req.Name]; exists {
		return nil, api.ErrNamespaceAlreadyExists
	}

	created := now().UTC().Format(time.RFC3339)
	ns := api.NamespaceResponse{Name: req.Name, PIDFormat: req.PIDFormat, DateCreated: created}
	s.namespaces[req.Name] = ns
	s.resources[req.Name] = make(map[string]api.ResourceResponse)
	return &ns, nil
}

func (s *Store) ListResources(_ context.Context, params api.ListResourcesParams) (*api.PaginatedResourcesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.namespaces[params.Namespace]; !ok {
		return nil, api.ErrNamespaceNotFound
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
	return &api.PaginatedResourcesResponse{Total: total, Offset: offset, Items: items}, nil
}

func (s *Store) CreateResource(_ context.Context, namespace string, req api.ResourceCreateRequest, rand io.Reader, now func() time.Time) (*api.ResourceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ns, ok := s.namespaces[namespace]
	if !ok {
		return nil, api.ErrNamespaceNotFound
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
		res := api.ResourceResponse{
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
	return nil, api.ErrPIDAllocationFailed
}

func (s *Store) BatchCreateResources(_ context.Context, namespace string, reqs []api.ResourceCreateRequest, rand io.Reader, now func() time.Time) ([]api.ResourceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ns, ok := s.namespaces[namespace]
	if !ok {
		return nil, api.ErrNamespaceNotFound
	}

	byPID := s.resources[namespace]
	out := make([]api.ResourceResponse, 0, len(reqs))
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
			res := api.ResourceResponse{
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
			return nil, api.ErrPIDAllocationFailed
		}
	}
	return out, nil
}

func (s *Store) GetResource(_ context.Context, namespace, pid string) (*api.ResourceResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.namespaces[namespace]; !ok {
		return nil, api.ErrNamespaceNotFound
	}
	res, ok := s.resources[namespace][pid]
	if !ok {
		return nil, api.ErrResourceNotFound
	}
	return &res, nil
}

func (s *Store) UpdateResource(_ context.Context, namespace, pid string, req api.ResourceUpdateRequest, now func() time.Time) (*api.ResourceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.namespaces[namespace]; !ok {
		return nil, api.ErrNamespaceNotFound
	}
	prev, ok := s.resources[namespace][pid]
	if !ok {
		return nil, api.ErrResourceNotFound
	}
	updated := now().UTC().Format(time.RFC3339)
	res := api.ResourceResponse{
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
