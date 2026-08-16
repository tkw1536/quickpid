//spellchecker:words server
package server

//spellchecker:words http github quickpid
import (
	"fmt"
	"net/http"

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
