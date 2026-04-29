package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"

	"github.com/google/uuid"
	"github.com/swaggest/swgui"
	"github.com/swaggest/swgui/v5emb"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/pid"
	"github.com/tkw1536/quickpid/spec"
)

type Handler struct {
	ops     Options
	backend backend.Backend
	mux     *http.ServeMux
}

var (
	errEmptyRequestBody      = errors.New("empty request body")
	errInvalidJSON           = errors.New("invalid JSON")
	errTrailingJSON          = errors.New("trailing JSON")
	errInvalidQueryParameter = errors.New("invalid query parameter")

	errRequestBodyTooLarge = errors.New("request payload too large")
	errTooManyItems        = errors.New("too many items")

	errUnableToAllocateNamespaceID = errors.New("unable to allocate a unique namespace id (is the number of namespaces exhausted?)")
	errUnableToAllocatePID         = errors.New("unable to allocate a unique pid (is the namespace exhausted?)")

	errInvalidNamespaceID = errors.New("invalid namespace")
	errInvalidPID         = errors.New("invalid pid")
)

// NewHandler returns an http.Handler for the PID Resolver API and Swagger UI.
//
// Routes on the returned handler are rooted at / (e.g. GET /resolver/namespaces);
// mount with http.StripPrefix(mountPath, NewHandler(Options{MountPath: mountPath}, res)) at mountPath+"/".
func NewHandler(options Options, backend backend.Backend) *Handler {
	options = options.withDefaults()

	h := &Handler{
		backend: backend,
		ops:     options,
		mux:     http.NewServeMux(),
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

var openapiYAML = []byte(spec.Spec())

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

		query := r.URL.Query()
		var tag *string
		if query.Has("tag") {
			v := query.Get("tag")
			tag = &v
		}

		out, err := h.backend.ListNamespaces(r.Context(), spec.ListNamespacesParams{
			Tag:    tag,
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
		var req spec.NamespaceCreateRequest
		if err := h.decodeJSON(w, r, &req); err != nil {
			writeError(w, err)
			return
		}
		if err := req.PIDFormat.Validate(); err != nil {
			writeError(w, err)
			return
		}

		for range maxNamespaceIDAttempts {
			name, err := uuid.NewRandomFromReader(h.ops.Rand)
			if err != nil {
				writeError(w, err)
				return
			}
			out, err := h.backend.CreateNamespace(r.Context(), name.String(), req, h.ops.Now)
			if err == nil {
				writeJSONResponse(w, http.StatusCreated, out)
				return
			}
			if !errors.Is(err, backend.ErrDuplicateNamespaceID) {
				writeError(w, err)
				return
			}
		}
		writeError(w, errUnableToAllocateNamespaceID)
	}
}

func (h *Handler) handleListResources() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("namespace")
		if !isValidNamespaceID(id) {
			writeError(w, errInvalidNamespaceID)
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
				writeError(w, fmt.Errorf("%w %q", errInvalidQueryParameter, "deleted"))
				return
			}
			deleted = &b
		}

		limit, offset, err := h.parsePagination(r)
		if err != nil {
			writeError(w, err)
			return
		}

		out, err := h.backend.ListResources(r.Context(), spec.ListResourcesParams{
			Namespace: id,
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
		var req spec.ResourceCreateRequest
		if err := h.decodeJSON(w, r, &req); err != nil {
			writeError(w, err)
			return
		}
		namespace := r.PathValue("namespace")
		if !isValidNamespaceID(namespace) {
			writeError(w, errInvalidNamespaceID)
			return
		}

		ns, err := h.backend.GetNamespace(r.Context(), namespace)
		if err != nil {
			writeError(w, err)
			return
		}

		out, err := h.allocatePID(ns.PIDFormat, func(pid string) (*spec.ResourceResponse, error) {
			return h.backend.CreateResource(r.Context(), namespace, pid, req, h.ops.Now)
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSONResponse(w, http.StatusCreated, out)
	}
}

func (h *Handler) handleBatchCreateResources() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var reqs []spec.ResourceCreateRequest
		if err := h.decodeJSON(w, r, &reqs); err != nil {
			writeError(w, err)
			return
		}
		if len(reqs) > h.ops.Limits.MaxBatchItems {
			writeError(w, fmt.Errorf("%w: %d > %d", errTooManyItems, len(reqs), h.ops.Limits.MaxBatchItems))
			return
		}

		namespace := r.PathValue("namespace")
		if !isValidNamespaceID(namespace) {
			writeError(w, errInvalidNamespaceID)
			return
		}

		ns, err := h.backend.GetNamespace(r.Context(), namespace)
		if err != nil {
			writeError(w, err)
			return
		}

		out, err := h.allocatePIDs(ns.PIDFormat, len(reqs), func(pids []string) ([]spec.ResourceResponse, error) {
			return h.backend.BatchCreateResources(r.Context(), namespace, pids, reqs, h.ops.Now)
		})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSONResponse(w, http.StatusCreated, out)
	}
}

func (h *Handler) handleGetResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("namespace")
		if !isValidNamespaceID(id) {
			writeError(w, errInvalidNamespaceID)
			return
		}

		pid := r.PathValue("pid")
		if !isValidPID(pid) {
			writeError(w, errInvalidPID)
			return
		}

		out, err := h.backend.GetResource(r.Context(), id, pid)
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSONResponse(w, http.StatusOK, out)
	}
}

func (h *Handler) handleUpdateResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req spec.ResourceUpdateRequest
		if err := h.decodeJSON(w, r, &req); err != nil {
			writeError(w, err)
			return
		}
		namespace := r.PathValue("namespace")
		if !isValidNamespaceID(namespace) {
			writeError(w, errInvalidNamespaceID)
			return
		}
		pid := r.PathValue("pid")
		if !isValidPID(pid) {
			writeError(w, errInvalidPID)
			return
		}
		out, err := h.backend.UpdateResource(r.Context(), namespace, pid, req, h.ops.Now)
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
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return errRequestBodyTooLarge
		}
		if errors.Is(err, io.EOF) {
			return errEmptyRequestBody
		}
		return fmt.Errorf("%w: %v", errInvalidJSON, err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errTrailingJSON
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
			return 0, 0, fmt.Errorf("%w %q", errInvalidQueryParameter, "limit")
		}
	}
	if limit > h.ops.Limits.MaxPageLimit {
		limit = h.ops.Limits.MaxPageLimit
	}

	offset = 0
	if query.Has("offset") {
		offset, err = parseInt(query.Get("offset"))
		if err != nil || offset < 0 {
			return 0, 0, fmt.Errorf("%w %q", errInvalidQueryParameter, "offset")
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
	maxNamespaceIDAttempts = 32
	namespaceIDRE          = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	pidRE                  = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

func isValidNamespaceID(s string) bool {
	return namespaceIDRE.MatchString(s)
}

func isValidPID(s string) bool {
	return pidRE.MatchString(s)
}

func (h *Handler) allocatePIDs(format pid.Format, n int, insert func([]string) ([]spec.ResourceResponse, error)) ([]spec.ResourceResponse, error) {
	if n == 0 {
		return []spec.ResourceResponse{}, nil
	}
	for range h.ops.Limits.MaxPIDAttempts {
		pids := make([]string, n)
		seen := make(map[string]struct{}, n)
		for i := range n {
			// Ensure uniqueness within this batch.
			for range h.ops.Limits.MaxPIDAttempts {
				candidate, err := format.Generate(h.ops.Rand)
				if err != nil {
					return nil, err
				}
				if _, exists := seen[candidate]; exists {
					continue
				}
				seen[candidate] = struct{}{}
				pids[i] = candidate
				break
			}
			if pids[i] == "" {
				return nil, errUnableToAllocatePID
			}
		}

		out, err := insert(pids)
		if err == nil {
			return out, nil
		}
		if errors.Is(err, backend.ErrPIDAllocationFailed) {
			continue
		}
		return nil, err
	}
	return nil, errUnableToAllocatePID
}

func (h *Handler) allocatePID(format pid.Format, insert func(string) (*spec.ResourceResponse, error)) (*spec.ResourceResponse, error) {
	pids, err := h.allocatePIDs(format, 1, func(pids []string) ([]spec.ResourceResponse, error) {
		res, err := insert(pids[0])
		if err != nil {
			return nil, err
		}
		return []spec.ResourceResponse{*res}, nil
	})
	if err != nil {
		return nil, err
	}
	p := pids[0]
	return &p, nil
}
