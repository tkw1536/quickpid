package backend

import (
	"context"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/tkw1536/quickpid/spec"
)

// NewInMemoryBackend returns a new backend backed by an in-memory map.
//
// maxPIDAttempts is an internal limit on the number of attempts to allocate a PID.
// It should be greater than 0.
func NewInMemoryBackend(maxPIDAttempts int) Backend {
	return &inMemoryBackend{
		namespaces:     make(map[string]spec.NamespaceResponse),
		resources:      make(map[string]map[string]spec.ResourceResponse),
		maxPIDAttempts: maxPIDAttempts,
	}
}

// inMemoryBackend is an in-memory Resolver implementation protected by a single RWMutex.
type inMemoryBackend struct {
	// protects the namespace and resource maps.
	mu sync.RWMutex

	namespaces map[string]spec.NamespaceResponse
	resources  map[string]map[string]spec.ResourceResponse

	// maximum number of attempts to allocate a PID.
	// must be > 0.
	maxPIDAttempts int
}

func (s *inMemoryBackend) ListNamespaces(_ context.Context, params spec.ListNamespacesParams) (*spec.PaginatedNamespacesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]spec.NamespaceResponse, 0, len(s.namespaces))
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
		return &spec.PaginatedNamespacesResponse{Total: total, Offset: offset, Items: []spec.NamespaceResponse{}}, nil
	}
	end := min(offset+limit, total)
	items := append([]spec.NamespaceResponse(nil), all[offset:end]...)
	return &spec.PaginatedNamespacesResponse{Total: total, Offset: offset, Items: items}, nil
}

func (s *inMemoryBackend) CreateNamespace(_ context.Context, namespace string, req spec.NamespaceCreateRequest, now func() time.Time) (*spec.NamespaceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.namespaces[namespace]; exists {
		return nil, ErrNamespaceIDAllocationFailed
	}
	created := now().UTC().Format(time.RFC3339)
	ns := spec.NamespaceResponse{
		ID:          namespace,
		Tag:         req.Tag,
		PIDFormat:   req.PIDFormat,
		DateCreated: created,
	}
	s.namespaces[namespace] = ns
	s.resources[namespace] = make(map[string]spec.ResourceResponse)
	return &ns, nil
}

func (s *inMemoryBackend) ListResources(_ context.Context, params spec.ListResourcesParams) (*spec.PaginatedResourcesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.namespaces[params.Namespace]; !ok {
		return nil, ErrNamespaceNotFound
	}

	byPID := s.resources[params.Namespace]
	filtered := make([]spec.ResourceResponse, 0, len(byPID))
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
		return &spec.PaginatedResourcesResponse{Total: total, Offset: offset, Items: []spec.ResourceResponse{}}, nil
	}
	end := min(offset+limit, total)
	items := append([]spec.ResourceResponse(nil), filtered[offset:end]...)
	return &spec.PaginatedResourcesResponse{Total: total, Offset: offset, Items: items}, nil
}

func (s *inMemoryBackend) CreateResource(_ context.Context, namespace string, req spec.ResourceCreateRequest, rand io.Reader, now func() time.Time) (*spec.ResourceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ns, ok := s.namespaces[namespace]
	if !ok {
		return nil, ErrNamespaceNotFound
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
		res := spec.ResourceResponse{
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
	return nil, ErrPIDAllocationFailed
}

func (s *inMemoryBackend) BatchCreateResources(_ context.Context, namespace string, reqs []spec.ResourceCreateRequest, rand io.Reader, now func() time.Time) ([]spec.ResourceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ns, ok := s.namespaces[namespace]
	if !ok {
		return nil, ErrNamespaceNotFound
	}

	byPID := s.resources[namespace]
	out := make([]spec.ResourceResponse, 0, len(reqs))
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
			res := spec.ResourceResponse{
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
			return nil, ErrPIDAllocationFailed
		}
	}
	return out, nil
}

func (s *inMemoryBackend) GetResource(_ context.Context, namespace, pid string) (*spec.ResourceResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.namespaces[namespace]; !ok {
		return nil, ErrNamespaceNotFound
	}
	res, ok := s.resources[namespace][pid]
	if !ok {
		return nil, ErrResourceNotFound
	}
	return &res, nil
}

func (s *inMemoryBackend) UpdateResource(_ context.Context, namespace, pid string, req spec.ResourceUpdateRequest, now func() time.Time) (*spec.ResourceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.namespaces[namespace]; !ok {
		return nil, ErrNamespaceNotFound
	}
	prev, ok := s.resources[namespace][pid]
	if !ok {
		return nil, ErrResourceNotFound
	}
	updated := now().UTC().Format(time.RFC3339)
	res := spec.ResourceResponse{
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
