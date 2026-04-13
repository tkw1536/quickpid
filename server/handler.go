package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"

	"github.com/swaggest/swgui"
	"github.com/swaggest/swgui/v5emb"
	"github.com/tkw1536/quickpid/api"

	_ "embed"
)

//go:embed openapi.yaml
var openapiYAML []byte

type Handler struct {
	ops Options
	res api.Resolver
	mux *http.ServeMux
}

// NewHandler returns an http.Handler for the PID Resolver API and Swagger UI.
//
// Routes on the returned handler are rooted at / (e.g. GET /resolver/namespaces);
// mount with http.StripPrefix(mountPath, NewHandler(Options{MountPath: mountPath}, res)) at mountPath+"/".
func NewHandler(options Options, resolver api.Resolver) *Handler {
	options = options.withDefaults()

	h := &Handler{
		res: resolver,
		ops: options,
		mux: http.NewServeMux(),
	}

	h.mux.Handle("GET /resolver/namespaces", h.handleListNamespaces())
	h.mux.Handle("POST /resolver/namespaces", h.handleCreateNamespace())
	h.mux.Handle("GET /resolver/namespaces/{namespace}/resources", h.handleListResources())
	h.mux.Handle("POST /resolver/namespaces/{namespace}/resources", h.handleCreateResource())
	h.mux.Handle("POST /resolver/namespaces/{namespace}/resources:batch", h.handleBatchCreateResources())
	h.mux.Handle("GET /resolver/namespaces/{namespace}/resources/{pid}", h.handleGetResource())
	h.mux.Handle("PATCH /resolver/namespaces/{namespace}/resources/{pid}", h.handleUpdateResource())

	if !options.DisableSwaggerUI {
		h.mux.Handle("GET /openapi.yaml", h.handleOpenAPISpec())
		h.mux.Handle("/", v5emb.NewHandlerWithConfig(swgui.Config{
			Title:            "PID Resolver API",
			SwaggerJSON:      h.ops.MountPath + "/openapi.yaml",
			BasePath:         h.ops.MountPath + "/",
			InternalBasePath: "/",
		}))
	}

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) handleOpenAPISpec() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(openapiYAML)
	}
}

func (h *Handler) handleListNamespaces() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, offset, err := h.parsePagination(r)
		if err != nil {
			writeError(w, err)
			return
		}

		out, err := h.res.ListNamespaces(r.Context(), api.ListNamespacesParams{
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSONResponse(w, http.StatusOK, out)
	}
}

func (h *Handler) handleCreateNamespace() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.NamespaceCreateRequest
		if err := h.decodeJSON(w, r, &req); err != nil {
			writeError(w, err)
			return
		}
		if !isValidNamespaceName(req.Name) {
			writeError(w, api.ErrInvalidNamespace)
			return
		}
		out, err := h.res.CreateNamespace(r.Context(), req, h.ops.Now)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSONResponse(w, http.StatusCreated, out)
	}
}

func (h *Handler) handleListResources() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ns := r.PathValue("namespace")
		if !isValidNamespaceName(ns) {
			writeError(w, api.ErrInvalidNamespace)
			return
		}
		query := r.URL.Query()

		var tag *string
		if query.Has("tag") {
			v := query.Get("tag")
			tag = &v
		}

		var deleted *bool
		if query.Has("deleted") {
			b, err := strconv.ParseBool(query.Get("deleted"))
			if err != nil {
				writeError(w, fmt.Errorf("%w %q", api.ErrInvalidQueryParameter, "deleted"))
				return
			}
			deleted = &b
		}

		limit, offset, err := h.parsePagination(r)
		if err != nil {
			writeError(w, err)
			return
		}

		out, err := h.res.ListResources(r.Context(), api.ListResourcesParams{
			Namespace: ns,
			Tag:       tag,
			Deleted:   deleted,

			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSONResponse(w, http.StatusOK, out)
	}
}

func (h *Handler) handleCreateResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.ResourceCreateRequest
		if err := h.decodeJSON(w, r, &req); err != nil {
			writeError(w, err)
			return
		}
		ns := r.PathValue("namespace")
		if !isValidNamespaceName(ns) {
			writeError(w, api.ErrInvalidNamespace)
			return
		}

		out, err := h.res.CreateResource(r.Context(), ns, req, h.ops.GeneratePID, h.ops.Now)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSONResponse(w, http.StatusCreated, out)
	}
}

func (h *Handler) handleBatchCreateResources() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var reqs []api.ResourceCreateRequest
		if err := h.decodeJSON(w, r, &reqs); err != nil {
			writeError(w, err)
			return
		}
		if len(reqs) > h.ops.Limits.MaxBatchItems {
			writeError(w, fmt.Errorf("%w: %d > %d", api.ErrTooManyItems, len(reqs), h.ops.Limits.MaxBatchItems))
			return
		}

		ns := r.PathValue("namespace")
		if !isValidNamespaceName(ns) {
			writeError(w, api.ErrInvalidNamespace)
			return
		}
		out, err := h.res.BatchCreateResources(r.Context(), ns, reqs, h.ops.GeneratePID, h.ops.Now)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSONResponse(w, http.StatusCreated, out)
	}
}

func (h *Handler) handleGetResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ns := r.PathValue("namespace")
		if !isValidNamespaceName(ns) {
			writeError(w, api.ErrInvalidNamespace)
			return
		}

		pid := r.PathValue("pid")
		if !isValidPID(pid) {
			writeError(w, api.ErrInvalidPID)
			return
		}

		out, err := h.res.GetResource(r.Context(), ns, pid)
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSONResponse(w, http.StatusOK, out)
	}
}

func (h *Handler) handleUpdateResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.ResourceUpdateRequest
		if err := h.decodeJSON(w, r, &req); err != nil {
			writeError(w, err)
			return
		}
		ns := r.PathValue("namespace")
		if !isValidNamespaceName(ns) {
			writeError(w, api.ErrInvalidNamespace)
			return
		}
		pid := r.PathValue("pid")
		if !isValidPID(pid) {
			writeError(w, api.ErrInvalidPID)
			return
		}
		out, err := h.res.UpdateResource(r.Context(), ns, pid, req, h.ops.Now)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSONResponse(w, http.StatusOK, out)
	}
}

func (h *Handler) decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	body := http.MaxBytesReader(w, r.Body, h.ops.Limits.MaxBodyBytes)
	defer body.Close()
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return api.ErrRequestBodyTooLarge
		}
		if errors.Is(err, io.EOF) {
			return api.ErrEmptyRequestBody
		}
		return api.ErrInvalidJSON
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return api.ErrTrailingJSON
	}
	return nil
}

// parsePagination parses pagination parameters from the query string
func (h *Handler) parsePagination(r *http.Request) (limit int, offset int, err error) {
	query := r.URL.Query()

	limit = h.ops.Limits.DefaultPageLimit
	if query.Has("limit") {
		limit, err = parseInt(query.Get("limit"))
		if err != nil || limit < 1 {
			return 0, 0, fmt.Errorf("%w %q", api.ErrInvalidQueryParameter, "limit")
		}
	}
	if limit > h.ops.Limits.MaxPageLimit {
		limit = h.ops.Limits.MaxPageLimit
	}

	offset = 0
	if query.Has("offset") {
		offset, err = parseInt(query.Get("offset"))
		if err != nil || offset < 0 {
			return 0, 0, fmt.Errorf("%w %q", api.ErrInvalidQueryParameter, "offset")
		}
	}
	return limit, offset, nil
}

func parseInt(v string) (int, error) {
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	return n, nil
}

var (
	namespaceNameRE = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	pidRE           = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

func isValidNamespaceName(s string) bool {
	return namespaceNameRE.MatchString(s)
}

func isValidPID(s string) bool {
	return pidRE.MatchString(s)
}
