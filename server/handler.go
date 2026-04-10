package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/swaggest/swgui"
	"github.com/swaggest/swgui/v5emb"
	"github.com/tkw1536/quickpid/api"

	_ "embed"
)

//go:embed openapi.yaml
var openapiYAML []byte

const maxBodyBytes = 1 << 20

// NewHandler returns an http.Handler for the PID Resolver API and Swagger UI.
//
// mountPath is the URL prefix where the caller will mount this handler (e.g. "/api/v2"); it must
// not have a trailing slash.
//
// pidGen generates new PID strings for POST /resources and batch creates. It must not use client
// request fields. The handler passes it to CreateResource / BatchCreateResources on each request.
//
// Routes on the returned handler are rooted at / (e.g. GET /resolver/namespaces);
// mount with http.StripPrefix(mountPath, NewHandler(mountPath, res, pidGen)) at mountPath+"/".
func NewHandler(mountPath string, res api.Resolver, pidGen func() (string, error)) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /resolver/namespaces", handleListNamespaces(res))
	mux.Handle("POST /resolver/namespaces", handleCreateNamespace(res))
	mux.Handle("GET /resolver/namespaces/{namespace}/resources", handleListResources(res))
	mux.Handle("POST /resolver/namespaces/{namespace}/resources", handleCreateResource(res, pidGen))
	mux.Handle("POST /resolver/namespaces/{namespace}/resources:batch", handleBatchCreateResources(res, pidGen))
	mux.Handle("GET /resolver/namespaces/{namespace}/resources/{pid}", handleGetResource(res))
	mux.Handle("PATCH /resolver/namespaces/{namespace}/resources/{pid}", handleUpdateResource(res))
	mux.Handle("GET /openapi.yaml", handleOpenAPISpec())
	// BasePath is the public URL for Swagger; InternalBasePath is the path this handler sees
	// after the caller mounts with http.StripPrefix(mountPath, ...).
	mux.Handle("/", v5emb.NewHandlerWithConfig(swgui.Config{
		Title:            "PID Resolver API",
		SwaggerJSON:      mountPath + "/openapi.yaml",
		BasePath:         mountPath + "/",
		InternalBasePath: "/",
	}))
	return mux
}

func handleOpenAPISpec() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(openapiYAML)
	}
}

func handleListNamespaces(res api.Resolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		out, err := res.ListNamespaces(ctx)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleCreateNamespace(res api.Resolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.NamespaceCreateRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		out, err := res.CreateNamespace(r.Context(), req)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

func handleListResources(res api.Resolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ns := r.PathValue("namespace")
		query := r.URL.Query()

		var tag *string
		if query.Has("tag") {
			v := query.Get("tag")
			tag = &v
		}
		out, err := res.ListResources(r.Context(), api.ListResourcesParams{Namespace: ns, Tag: tag})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleCreateResource(res api.Resolver, pidGen func() (string, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.ResourceCreateRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		ns := r.PathValue("namespace")
		out, err := res.CreateResource(r.Context(), ns, req, pidGen)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

func handleBatchCreateResources(res api.Resolver, pidGen func() (string, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var reqs []api.ResourceCreateRequest
		if err := decodeJSON(r, &reqs); err != nil {
			writeError(w, err)
			return
		}
		ns := r.PathValue("namespace")
		out, err := res.BatchCreateResources(r.Context(), ns, reqs, pidGen)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, out)
	}
}

func handleGetResource(res api.Resolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ns := r.PathValue("namespace")
		pid := r.PathValue("pid")
		out, err := res.GetResource(r.Context(), ns, pid)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleUpdateResource(res api.Resolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.ResourceUpdateRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, err)
			return
		}
		ns := r.PathValue("namespace")
		pid := r.PathValue("pid")
		out, err := res.UpdateResource(r.Context(), ns, pid, req)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func decodeJSON(r *http.Request, v any) error {
	body := http.MaxBytesReader(nil, r.Body, maxBodyBytes)
	defer body.Close()
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type errResp struct {
	Error string `json:"error"`
}

// Order is fixed: first match wins. Use sentinel.Error() for the JSON body (not err.Error()) so
// wrapped errors still emit exact OpenAPI messages.
var apiClientErrors = []struct {
	sentinel error
	status   int
}{
	{api.ErrEmptyRequestBody, http.StatusBadRequest},
	{api.ErrInvalidJSON, http.StatusBadRequest},
	{api.ErrTrailingJSON, http.StatusBadRequest},
	{api.ErrNamespaceNotFound, http.StatusNotFound},
	{api.ErrResourceNotFound, http.StatusNotFound},
	{api.ErrNamespaceAlreadyExists, http.StatusConflict},
	{api.ErrPIDAllocationFailed, http.StatusInternalServerError},
}

func writeError(w http.ResponseWriter, err error) {
	for _, e := range apiClientErrors {
		if errors.Is(err, e.sentinel) {
			writeJSONError(w, e.status, e.sentinel.Error())
			return
		}
	}
	writeJSONError(w, http.StatusInternalServerError, "internal server error")
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errResp{Error: msg})
}
