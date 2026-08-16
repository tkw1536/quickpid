//spellchecker:words memory
package memory

//spellchecker:words context sort github quickpid backend
import (
	"context"
	"sort"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
)

func (s *MemoryBackend) GetMount(_ context.Context, baseURI api.ValidBaseURI) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	namespaceID, ok := s.mounts[baseURI.String()]
	if !ok {
		return "", backend.ErrMountNotFound
	}
	return namespaceID, nil
}

func (s *MemoryBackend) SetMount(_ context.Context, baseURI api.ValidBaseURI, namespace api.ValidNamespaceID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	namespaceString := namespace.String()

	if _, ok := s.namespaces[namespaceString]; !ok {
		return backend.ErrNamespaceNotFound
	}
	s.mounts[baseURI.String()] = namespaceString
	return nil
}

func (s *MemoryBackend) DeleteMount(_ context.Context, baseURI api.ValidBaseURI) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	baseURIString := baseURI.String()

	if _, ok := s.mounts[baseURIString]; !ok {
		return backend.ErrMountNotFound
	}
	delete(s.mounts, baseURIString)
	return nil
}

func (s *MemoryBackend) ListMounts(_ context.Context, params api.ListMountsParams) (*api.PaginatedMountsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]api.MountResponse, 0, len(s.mounts))
	for baseURI, namespaceID := range s.mounts {
		all = append(all, api.MountResponse{
			BaseURI:   baseURI,
			Namespace: namespaceID,
		})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Namespace != all[j].Namespace {
			return all[i].Namespace < all[j].Namespace
		}
		return all[i].BaseURI < all[j].BaseURI
	})

	total := len(all)
	limit := params.Limit
	offset := params.Offset
	if offset >= total {
		return &api.PaginatedMountsResponse{Total: total, Offset: offset, Items: []api.MountResponse{}}, nil
	}
	end := min(offset+limit, total)
	items := append([]api.MountResponse(nil), all[offset:end]...)
	if items == nil {
		items = []api.MountResponse{}
	}
	return &api.PaginatedMountsResponse{Total: total, Offset: offset, Items: items}, nil
}

func (s *MemoryBackend) ListNamespaceMounts(_ context.Context, namespace api.ValidNamespaceID, params api.ListNamespaceMountsParams) (*api.PaginatedBaseURIResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	namespaceString := namespace.String()

	if _, ok := s.namespaces[namespaceString]; !ok {
		return nil, backend.ErrNamespaceNotFound
	}

	all := make([]string, 0)
	for baseURI, namespaceID := range s.mounts {
		if namespaceID == namespaceString {
			all = append(all, baseURI)
		}
	}
	sort.Strings(all)

	total := len(all)
	limit := params.Limit
	offset := params.Offset
	if offset >= total {
		return &api.PaginatedBaseURIResponse{Total: total, Offset: offset, Items: []string{}}, nil
	}
	end := min(offset+limit, total)
	items := append([]string(nil), all[offset:end]...)
	if items == nil {
		items = []string{}
	}
	return &api.PaginatedBaseURIResponse{Total: total, Offset: offset, Items: items}, nil
}
