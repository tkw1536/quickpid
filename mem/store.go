package mem

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/tkw1536/quickpid/api"
)

// Store is an in-memory Resolver implementation protected by a single RWMutex.
type Store struct {
	mu         sync.RWMutex
	namespaces map[string]api.NamespaceResponse
	resources  map[string]map[string]api.ResourceResponse
	nextPID    map[string]int
}

// NewStore returns an empty Store.
func NewStore() *Store {
	return &Store{
		namespaces: make(map[string]api.NamespaceResponse),
		resources:  make(map[string]map[string]api.ResourceResponse),
		nextPID:    make(map[string]int),
	}
}

var _ api.Resolver = (*Store)(nil)

func (s *Store) ListNamespaces(_ context.Context) ([]api.NamespaceResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]api.NamespaceResponse, 0, len(s.namespaces))
	for _, ns := range s.namespaces {
		out = append(out, ns)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Store) CreateNamespace(_ context.Context, req api.NamespaceCreateRequest) (*api.NamespaceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.namespaces[req.Name]; exists {
		return nil, api.ErrNamespaceAlreadyExists
	}
	now := time.Now().UTC().Format(time.RFC3339)
	ns := api.NamespaceResponse{Name: req.Name, DateCreated: now}
	s.namespaces[req.Name] = ns
	s.resources[req.Name] = make(map[string]api.ResourceResponse)
	s.nextPID[req.Name] = 0
	return &ns, nil
}

func (s *Store) ListResources(_ context.Context, params api.ListResourcesParams) ([]api.ResourceResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.namespaces[params.Namespace]; !ok {
		return nil, api.ErrNamespaceNotFound
	}
	byPID := s.resources[params.Namespace]
	out := make([]api.ResourceResponse, 0, len(byPID))
	for _, r := range byPID {
		if params.Tag != "" && r.Tag != params.Tag {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PID < out[j].PID })
	return out, nil
}

func (s *Store) CreateResource(_ context.Context, namespace string, req api.ResourceCreateRequest) (*api.ResourceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.namespaces[namespace]; !ok {
		return nil, api.ErrNamespaceNotFound
	}
	s.nextPID[namespace]++
	pid := strconv.Itoa(s.nextPID[namespace])
	now := time.Now().UTC().Format(time.RFC3339)
	res := api.ResourceResponse{
		PID:          pid,
		URL:          req.URL,
		IdInTarget:   req.IdInTarget,
		DateCreated:  now,
		DateUpdated:  now,
		TargetSystem: req.TargetSystem,
		Tag:          req.Tag,
		Deleted:      false,
	}
	s.resources[namespace][pid] = res
	return &res, nil
}

func (s *Store) BatchCreateResources(_ context.Context, namespace string, reqs []api.ResourceCreateRequest) ([]api.ResourceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.namespaces[namespace]; !ok {
		return nil, api.ErrNamespaceNotFound
	}
	out := make([]api.ResourceResponse, 0, len(reqs))
	for _, req := range reqs {
		s.nextPID[namespace]++
		pid := strconv.Itoa(s.nextPID[namespace])
		now := time.Now().UTC().Format(time.RFC3339)
		res := api.ResourceResponse{
			PID:          pid,
			URL:          req.URL,
			IdInTarget:   req.IdInTarget,
			DateCreated:  now,
			DateUpdated:  now,
			TargetSystem: req.TargetSystem,
			Tag:          req.Tag,
			Deleted:      false,
		}
		s.resources[namespace][pid] = res
		out = append(out, res)
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

func (s *Store) UpdateResource(_ context.Context, namespace, pid string, req api.ResourceUpdateRequest) (*api.ResourceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.namespaces[namespace]; !ok {
		return nil, api.ErrNamespaceNotFound
	}
	prev, ok := s.resources[namespace][pid]
	if !ok {
		return nil, api.ErrResourceNotFound
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res := api.ResourceResponse{
		PID:          prev.PID,
		URL:          req.URL,
		IdInTarget:   req.IdInTarget,
		DateCreated:  prev.DateCreated,
		DateUpdated:  now,
		TargetSystem: req.TargetSystem,
		Tag:          req.Tag,
		Deleted:      req.Deleted,
	}
	s.resources[namespace][pid] = res
	return &res, nil
}
