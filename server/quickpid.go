//spellchecker:words server
package server

//spellchecker:words http sync github quickpid
import (
	"fmt"
	"net/http"
	"sync"

	"github.com/tkw1536/quickpid/api"
)

func (h *Server) getResolverInfo(w http.ResponseWriter, r *http.Request) (*api.InfoResponse, error) {
	info, err := h.svc.GetResolverInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get resolver info: %w", err)
	}
	return info, nil
}

func (h *Server) getResolverMeta(w http.ResponseWriter, r *http.Request) (*api.MetaResponse, error) {
	meta := h.ops.Meta
	return &meta, nil
}

var cachedScopesResponse = sync.OnceValue(func() *api.ScopesResponse {
	return &api.ScopesResponse{
		User:      api.GetUserScopeDefinitions(),
		Namespace: api.GetNamespaceScopeDefinitions(),
	}
})

func (h *Server) listScopes(w http.ResponseWriter, r *http.Request) (*api.ScopesResponse, error) {
	if h.svc.AnonymousMode() {
		return &api.ScopesResponse{
			User:      []api.UserScopeDefinition{},
			Namespace: []api.NamespaceScopeDefinition{},
		}, nil
	}
	return cachedScopesResponse(), nil
}
