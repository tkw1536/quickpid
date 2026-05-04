package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"sync"

	"github.com/swaggest/swgui"
	"github.com/swaggest/swgui/v5emb"
	"github.com/tkw1536/quickpid"
	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/pid"
)

type Handler struct {
	// m allows options to be updated without having to stop requests
	m sync.RWMutex

	ops     Options
	runtime Runtime
	backend backend.Backend
	mux     *http.ServeMux

	logger *slog.Logger
}

// NewHandler returns an http.Handler for the PID Resolver API and Swagger UI.
//
// Routes on the returned handler are rooted at / (e.g. GET /resolver/namespaces);
// mount with http.StripPrefix(mountPath, NewHandler(Options{MountPath: mountPath}, res)) at mountPath+"/".
func NewHandler(options Options, runtime Runtime, backend backend.Backend, logger *slog.Logger) *Handler {
	options = options.withDefaults()

	h := &Handler{
		backend: backend,
		ops:     options,
		runtime: runtime,
		mux:     http.NewServeMux(),
		logger:  logger,
	}

	h.mux.Handle("GET /resolver", handle(
		h,
		h.getResolverInfo,
		http.StatusOK,
		[]api.Error{
			api.InfoUnavailable,
		},
	))
	h.mux.Handle("GET /resolver/namespaces", handle(
		h,
		h.listNamespaces,
		http.StatusOK,
		[]api.Error{
			api.InvalidQueryParameter,
			api.DatabaseError,
		},
	))
	h.mux.Handle("POST /resolver/namespaces", handle(
		h,
		h.createNamespace,
		http.StatusCreated,
		[]api.Error{
			api.BodySizeExceeded,
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.DatabaseError,
			api.BadIDGeneration,
			api.InsufficientEntropy,
		},
	))

	h.mux.Handle("GET /resolver/namespaces/{namespace}/resources", handle(
		h,
		h.listResources,
		http.StatusOK,
		[]api.Error{
			api.InvalidNamespaceID,
			api.InvalidQueryParameter,
			api.NamespaceNotFound,
			api.DatabaseError,
		},
	))

	h.mux.Handle("POST /resolver/namespaces/{namespace}/resources", handle(
		h,
		h.createResource,
		http.StatusCreated,
		[]api.Error{
			api.BodySizeExceeded,
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.InvalidNamespaceID,
			api.NamespaceNotFound,
			api.DatabaseError,
			api.BadIDGeneration,
			api.InsufficientEntropy,
		},
	))
	h.mux.Handle("POST /resolver/namespaces/{namespace}/resources:batch", handle(
		h,
		h.batchCreateResources,
		http.StatusCreated,
		[]api.Error{
			api.BodySizeExceeded,
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.ItemLimitExceeded,
			api.InvalidNamespaceID,
			api.NamespaceNotFound,
			api.DatabaseError,
			api.BadIDGeneration,
			api.InsufficientEntropy,
		},
	))

	h.mux.Handle("GET /resolver/namespaces/{namespace}/resources/{pid}", handle(
		h,
		h.getResource,
		http.StatusOK,
		[]api.Error{
			api.InvalidNamespaceID,
			api.InvalidPID,
			api.NamespaceNotFound,
			api.ResourceNotFound,
			api.DatabaseError,
		},
	))
	h.mux.Handle("PATCH /resolver/namespaces/{namespace}/resources/{pid}", handle(
		h,
		h.updateResource,
		http.StatusOK,
		[]api.Error{
			api.BodySizeExceeded,
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.InvalidNamespaceID,
			api.InvalidPID,
			api.DatabaseError,
			api.NamespaceNotFound,
			api.ResourceNotFound,
		},
	))

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

var errSpecInfoPrivate = errors.New("info is private")

// getResolverInfo returns information about the resolver.
//
// It can return the following errors:
//
// - [api.InfoUnavailable]
func (h *Handler) getResolverInfo(w http.ResponseWriter, r *http.Request) (*api.InfoResponse, api.Error, error) {
	if !h.ops.InfoEnabled {
		return nil, api.InfoUnavailable, errSpecInfoPrivate
	}
	return &api.InfoResponse{
		MaxBodyBytes:     h.ops.Limits.MaxBodyBytes,
		DefaultPageLimit: int64(h.ops.Limits.DefaultPageLimit),
		MaxPageLimit:     int64(h.ops.Limits.MaxPageLimit),
		MaxBatchItems:    int64(h.ops.Limits.MaxBatchItems),
	}, "", nil
}

// SetOptions updates the options for this handler.
// It is safe to call this method concurrently with ServeHTTP.
func (h *Handler) SetOptions(options Options) {
	h.m.Lock()
	defer h.m.Unlock()

	h.ops = options.withDefaults()
}

// ServeHTTP implements [http.Handler].
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.m.RLock()
	defer h.m.RUnlock()

	h.mux.ServeHTTP(w, r)
}

var openapiYAML = []byte(quickpid.Spec())

func (h *Handler) handleOpenAPISpec() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(openapiYAML)
	}
}

// listNamespaces lists namespaces.
//
// It can return the following errors:
//
// - [api.InvalidQueryParameter]
// - [api.DatabaseError]
func (h *Handler) listNamespaces(w http.ResponseWriter, r *http.Request) (*api.PaginatedNamespacesResponse, api.Error, error) {
	limit, offset, specError, err := h.parsePagination(r)
	if err != nil {
		return nil, specError, err
	}

	query := r.URL.Query()
	var tag *string
	if query.Has("tag") {
		v := query.Get("tag")
		tag = &v
	}

	out, err := h.backend.ListNamespaces(r.Context(), api.ListNamespacesParams{
		Tag:    tag,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return out, "", nil
}

// createNamespace creates a new namespace.
//
// It can return the following errors:
//
// - [api.BodySizeExceeded]
// - [api.BodyMissing]
// - [api.BodyInvalidJSON]
//
// - [api.DatabaseError]
// - [api.BadIDGeneration]
// - [api.InsufficientEntropy]
func (h *Handler) createNamespace(w http.ResponseWriter, r *http.Request) (*api.NamespaceResponse, api.Error, error) {
	var req api.NamespaceCreateRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}

	for range h.ops.Limits.MaxNamespaceIDAttempts {
		name, err := h.runtime.NewNamespaceID()
		if err != nil {
			return nil, api.BadIDGeneration, err
		}
		if !namespaceIDRE.MatchString(name) {
			return nil, api.BadIDGeneration, fmt.Errorf("%w: %q is not a valid namespace id", errBadNamespaceID, name)
		}
		out, err := h.backend.CreateNamespace(r.Context(), name, req, h.runtime.Now)
		if err == nil {
			return out, "", nil
		}
		if !errors.Is(err, backend.ErrDuplicateNamespaceID) {
			return nil, api.DatabaseError, err
		}
	}
	return nil, api.InsufficientEntropy, fmt.Errorf("%w: gave up namespace id generation after %d attempts", errInsufficientEntropy, h.ops.Limits.MaxNamespaceIDAttempts)
}

var errDeletedInvalid = errors.New("invalid deleted query parameter")

// listResources lists resources in a namespace.
//
// It can return the following errors:
//
// - [api.InvalidNamespaceID]
//
// - [api.InvalidQueryParameter]
//
// - [api.NamespaceNotFound]
// - [api.DatabaseError]
func (h *Handler) listResources(w http.ResponseWriter, r *http.Request) (*api.PaginatedResourcesResponse, api.Error, error) {
	namespace, specError, err := getNamespace(r)
	if err != nil {
		return nil, specError, err
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
			return nil, api.InvalidQueryParameter, fmt.Errorf("%w: %w", errDeletedInvalid, err)
		}
		deleted = &b
	}

	limit, offset, specError, err := h.parsePagination(r)
	if err != nil {
		return nil, specError, err
	}

	out, err := h.backend.ListResources(r.Context(), api.ListResourcesParams{
		Namespace: namespace,
		Tag:       tag,
		Deleted:   deleted,

		Limit:  limit,
		Offset: offset,
	})
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.NamespaceNotFound, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return out, "", nil
}

// createResource creates a single resource.
//
// It can return the following errors:
//
// - [api.BodySizeExceeded]
// - [api.BodyMissing]
// - [api.BodyInvalidJSON]
//
// - [api.InvalidNamespaceID]
// - [api.NamespaceNotFound]
// - [api.DatabaseError]
// - [api.BadIDGeneration]
// - [api.InsufficientEntropy]
func (h *Handler) createResource(w http.ResponseWriter, r *http.Request) (*api.ResourceResponse, api.Error, error) {
	var req api.ResourceCreateRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}

	namespace, specError, err := getNamespace(r)
	if err != nil {
		return nil, specError, err
	}

	ns, err := h.backend.GetNamespace(r.Context(), namespace)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.NamespaceNotFound, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}

	out, specError, err := h.allocatePID(ns.PIDFormat, func(pid string) (*api.ResourceResponse, error) {
		return h.backend.CreateResource(r.Context(), namespace, pid, req, h.runtime.Now)
	})
	if err != nil {
		return nil, specError, err
	}
	return out, "", nil
}

// batchCreateResources creates multiple resources in a namespace.
//
// It can return the following errors:
//
// - [api.BodySizeExceeded]
// - [api.BodyMissing]
// - [api.BodyInvalidJSON]
//
// - [api.ItemLimitExceeded]
// - [api.InvalidNamespaceID]
// - [api.NamespaceNotFound]
// - [api.DatabaseError]
//
// - [api.BadIDGeneration]
// - [api.InsufficientEntropy]
func (h *Handler) batchCreateResources(w http.ResponseWriter, r *http.Request) ([]api.ResourceResponse, api.Error, error) {
	var reqs []api.ResourceCreateRequest
	if specError, err := h.decodeJSON(w, r, &reqs); err != nil {
		return nil, specError, err
	}
	if len(reqs) > h.ops.Limits.MaxBatchItems {
		return nil, api.ItemLimitExceeded, fmt.Errorf("%d > %d", len(reqs), h.ops.Limits.MaxBatchItems)
	}

	namespace, specError, err := getNamespace(r)
	if err != nil {
		return nil, specError, err
	}

	ns, err := h.backend.GetNamespace(r.Context(), namespace)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.NamespaceNotFound, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}

	out, specError, err := h.allocatePIDs(ns.PIDFormat, len(reqs), func(pids []string) ([]api.ResourceResponse, error) {
		return h.backend.BatchCreateResources(r.Context(), namespace, pids, reqs, h.runtime.Now)
	})
	if err != nil {
		return nil, specError, err
	}
	return out, "", nil
}

// getResource gets a resource by namespace and pid.
//
// It can return the following errors:
//
// - [api.InvalidNamespaceID]
// - [api.InvalidPID]
// - [api.NamespaceNotFound]
// - [api.ResourceNotFound]
// - [api.DatabaseError]
func (h *Handler) getResource(w http.ResponseWriter, r *http.Request) (*api.ResourceResponse, api.Error, error) {
	namespace, specError, err := getNamespace(r)
	if err != nil {
		return nil, specError, err
	}
	pid, specError, err := getPID(r)
	if err != nil {
		return nil, specError, err
	}

	out, err := h.backend.GetResource(r.Context(), namespace, pid)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.NamespaceNotFound, err
	}
	if errors.Is(err, backend.ErrResourceNotFound) {
		return nil, api.ResourceNotFound, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return out, "", nil
}

// updateResource updates a resource by namespace and pid.
//
// It can return the following errors:
//
// - [api.BodySizeExceeded]
// - [api.BodyMissing]
// - [api.BodyInvalidJSON]
//
// - [api.InvalidNamespaceID]
// - [api.InvalidPID]
//
// - [api.DatabaseError]
// - [api.NamespaceNotFound]
// - [api.ResourceNotFound]
func (h *Handler) updateResource(w http.ResponseWriter, r *http.Request) (*api.ResourceResponse, api.Error, error) {
	var req api.ResourceUpdateRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}
	namespace, specError, err := getNamespace(r)
	if err != nil {
		return nil, specError, err
	}

	pid, specError, err := getPID(r)
	if err != nil {
		return nil, specError, err
	}
	out, err := h.backend.UpdateResource(r.Context(), namespace, pid, req, h.runtime.Now)
	if errors.Is(err, backend.ErrNamespaceNotFound) {
		return nil, api.NamespaceNotFound, err
	}
	if errors.Is(err, backend.ErrResourceNotFound) {
		return nil, api.ResourceNotFound, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return out, "", nil
}

var errTrailingJSON = errors.New("trailing json after value")

// decodeJSON decodes the request body into v.
//
// It can return the following errors:
//
// - [api.BodySizeExceeded]
// - [api.BodyMissing]
// - [api.BodyInvalidJSON]
func (h *Handler) decodeJSON(w http.ResponseWriter, r *http.Request, v any) (api.Error, error) {
	body := http.MaxBytesReader(w, r.Body, h.ops.Limits.MaxBodyBytes)
	defer body.Close()

	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return api.BodySizeExceeded, err
		}
		if errors.Is(err, io.EOF) {
			return api.BodyMissing, err
		}
		return api.BodyInvalidJSON, err
	}
	_, err := dec.Token()
	if !errors.Is(err, io.EOF) || err == nil {
		if err == nil {
			err = errTrailingJSON
		}
		return api.BodyInvalidJSON, err
	}
	return "", nil
}

var (
	errLimitInvalid            = errors.New("invalid limit")
	errLimitMustBePositive     = errors.New("limit must be positive")
	errOffsetInvalid           = errors.New("invalid offset")
	errOffsetMustBeNonNegative = errors.New("offset must be non-negative")
)

// parsePagination parses pagination parameters from the query string.
//
// It can return the following errors:
//
// - [api.InvalidQueryParameter]
func (h *Handler) parsePagination(r *http.Request) (limit int, offset int, specError api.Error, err error) {
	query := r.URL.Query()

	limit = h.ops.Limits.DefaultPageLimit
	if query.Has("limit") {
		limit, err = parseInt(query.Get("limit"))
		if err != nil {
			return 0, 0, api.InvalidQueryParameter, fmt.Errorf("%w: %w", errLimitInvalid, err)
		}
		if limit <= 0 {
			return 0, 0, api.InvalidQueryParameter, errLimitMustBePositive
		}
	}
	if limit > h.ops.Limits.MaxPageLimit {
		limit = h.ops.Limits.MaxPageLimit
	}

	offset = 0
	if query.Has("offset") {
		offset, err = parseInt(query.Get("offset"))
		if err != nil {
			return 0, 0, api.InvalidQueryParameter, fmt.Errorf("%w: %w", errOffsetInvalid, err)
		}
		if offset < 0 {
			return 0, 0, api.InvalidQueryParameter, errOffsetMustBeNonNegative
		}
	}
	return limit, offset, "", nil
}

var errInvalidNamespaceID = errors.New("invalid namespace id")

// getNamespace gets the namespace from the request path.
//
// It can return the following errors:
//
// - [api.InvalidNamespaceID]
func getNamespace(r *http.Request) (namespace string, specError api.Error, err error) {
	namespace = r.PathValue("namespace")
	if !namespaceIDRE.MatchString(namespace) {
		return "", api.InvalidNamespaceID, errInvalidNamespaceID
	}
	return namespace, "", nil
}

var errInvalidPID = errors.New("invalid pid")

// getPID gets the pid from the request path.
//
// It can return the following errors:
//
// - [api.InvalidPID]
func getPID(r *http.Request) (pid string, specError api.Error, err error) {
	pid = r.PathValue("pid")
	if !pidRE.MatchString(pid) {
		return "", api.InvalidPID, errInvalidPID
	}
	return pid, "", nil
}

func parseInt(v string) (int, error) {
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	return n, nil
}

var (
	namespaceIDRE = regexp.MustCompile(`^[a-z0-9_-]+$`)
	pidRE         = regexp.MustCompile(`^[a-z0-9_-]+$`)
)

var (
	errInsufficientEntropy = errors.New("insufficient entropy")
	errBadPID              = errors.New("bad pid generated")
	errBadNamespaceID      = errors.New("bad namespace id generated")
)

// allocatePIDs allocates n unique pids in the given namespace.
//
// It can return the following errors:
//
// - [api.BadIDGeneration]
// - [api.DatabaseError]
// - [api.InsufficientEntropy]
func (h *Handler) allocatePIDs(format pid.Format, n int, insert func([]string) ([]api.ResourceResponse, error)) ([]api.ResourceResponse, api.Error, error) {
	if n == 0 {
		return []api.ResourceResponse{}, "", nil
	}
	for range h.ops.Limits.MaxPIDAttempts {
		pids := make([]string, n)
		seen := make(map[string]struct{}, n)
		for i := range n {
			// Ensure uniqueness within this batch.
			for range h.ops.Limits.MaxPIDAttempts {
				candidate, err := h.runtime.NewPID(format)
				if err != nil {
					return nil, api.BadIDGeneration, err
				}
				if _, exists := seen[candidate]; exists {
					continue
				}
				seen[candidate] = struct{}{}
				pids[i] = candidate
				break
			}
			if !format.IsValid(pids[i]) {
				return nil, api.BadIDGeneration, fmt.Errorf("%w: %q is not a valid pid", errBadPID, pids[i])
			}
		}

		out, err := insert(pids)
		if err == nil {
			return out, "", nil
		}
		if !errors.Is(err, backend.ErrPIDAllocationFailed) {
			return nil, api.DatabaseError, err
		}
	}
	return nil, api.InsufficientEntropy, fmt.Errorf("%w: gave up pid generation after %d attempts", errInsufficientEntropy, h.ops.Limits.MaxPIDAttempts)
}

// allocatePIDs is like [Handler.allocatePIDs] but for a single PID.
func (h *Handler) allocatePID(format pid.Format, insert func(string) (*api.ResourceResponse, error)) (*api.ResourceResponse, api.Error, error) {
	pids, specError, err := h.allocatePIDs(format, 1, func(pids []string) ([]api.ResourceResponse, error) {
		res, err := insert(pids[0])
		if err != nil {
			return nil, err
		}
		return []api.ResourceResponse{*res}, nil
	})
	if err != nil {
		return nil, specError, err
	}
	p := pids[0]
	return &p, "", nil
}
