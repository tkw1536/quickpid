//spellchecker:words server
package server

//spellchecker:words slog http sync github swaggest swgui quickpid internal openapi server lowlevel service
import (
	"log/slog"
	"net/http"
	"sync"

	"github.com/swaggest/swgui"
	"github.com/swaggest/swgui/v5emb"
	quickpid "github.com/tkw1536/quickpid"
	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/internal/openapi"
	"github.com/tkw1536/quickpid/server/internal/lowlevel"
	"github.com/tkw1536/quickpid/service"
)

// Server implements [http.Handler] for the PID resolver API and Swagger UI.
// The zero value is not ready to use; use [NewServer] instead.
type Server struct {
	m sync.RWMutex // m allows options to be updated without having to stop requests

	svc *service.Service

	ops Options
	mux *http.ServeMux

	logger   *slog.Logger
	lowlevel *lowlevel.Handler

	createdViaNew bool
}

// NewServer returns an [Server for the PID Resolver API and Swagger UI.
//
// Routes on the returned handler are rooted at / (e.g. GET /resolver/namespaces);
// mount with http.StripPrefix(mountPath, NewServer(Options{MountPath: mountPath}, svc)) at mountPath+"/".
func NewServer(options Options, svc *service.Service, logger *slog.Logger) *Server {
	options = options.withValidValues()

	h := &Server{
		svc:      svc,
		ops:      options,
		mux:      http.NewServeMux(),
		logger:   logger,
		lowlevel: lowlevel.NewHandler(svc, logger),

		createdViaNew: true,
	}

	h.mux.Handle("GET /resolver", h.lowlevel.Public(
		h.getResolverInfo,
		lowlevel.FixedStatusCode[*api.InfoResponse](http.StatusOK),
		[]api.ErrorString{
			api.InfoUnavailable,
		},
	))
	h.mux.Handle("GET /resolver/info", h.lowlevel.Public(

		h.getResolverMeta,
		lowlevel.FixedStatusCode[*api.MetaResponse](http.StatusOK),
		nil,
	))
	h.mux.Handle("GET /resolver/namespaces", h.lowlevel.Open(

		h.listNamespaces,
		func(result *api.PaginatedNamespacesResponse) int {
			return http.StatusOK
		},
		[]api.ErrorString{
			api.InvalidQueryParameter,
			api.Unauthorized,
			api.UserNotFound,
			api.DatabaseError,
		},
	))
	h.mux.Handle("GET /resolver/resources/count", h.lowlevel.Public(

		h.countAllResources,
		lowlevel.FixedStatusCode[*api.ResourceCountResponse](http.StatusOK),
		[]api.ErrorString{
			api.Unauthorized,
			api.Forbidden,
			api.DatabaseError,
		},
	))
	h.mux.Handle("GET /resolver/mounts", h.lowlevel.Open(

		h.listMounts,
		lowlevel.FixedStatusCode[*api.PaginatedMountsResponse](http.StatusOK),
		[]api.ErrorString{
			api.InvalidQueryParameter,
			api.Unauthorized,
			api.Forbidden,
			api.DatabaseError,
		},
	))
	h.mux.Handle("GET /resolver/mounts/{base_uri}", h.lowlevel.Public(

		h.getMount,
		lowlevel.FixedStatusCode[*api.MountResponse](http.StatusOK),
		[]api.ErrorString{
			api.InvalidBaseURI,
			api.MountNotFound,
			api.DatabaseError,
		},
	))
	h.mux.Handle("GET /resolver/mounts/{base_uri}/{pid}", h.lowlevel.Open(

		h.resolveMountedResource,
		lowlevel.FixedStatusCode[api.ResourceGetResult](http.StatusFound),
		[]api.ErrorString{
			api.InvalidBaseURI,
			api.InvalidPID,
			api.Unauthorized,
			api.MountNotFound,
			api.NamespaceNotFound,
			api.ResourceNotFound,
			api.DatabaseError,
		},
	))
	h.mux.Handle("PUT /resolver/mounts/{base_uri}", h.lowlevel.Open(

		h.setMount,
		lowlevel.FixedStatusCode[*api.MountResponse](http.StatusOK),
		[]api.ErrorString{
			api.BodySizeExceeded,
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.InvalidBaseURI,
			api.InvalidNamespaceID,
			api.Unauthorized,
			api.Forbidden,
			api.NamespaceNotFound,
			api.DatabaseError,
		},
	))
	h.mux.Handle("DELETE /resolver/mounts/{base_uri}", h.lowlevel.Open(

		h.deleteMount,
		lowlevel.FixedStatusCode[struct{}](http.StatusNoContent),
		[]api.ErrorString{
			api.InvalidBaseURI,
			api.Unauthorized,
			api.Forbidden,
			api.MountNotFound,
			api.DatabaseError,
		},
	))
	h.mux.Handle("POST /resolver/namespaces", h.lowlevel.Open(

		h.createNamespace,
		lowlevel.FixedStatusCode[*api.NamespaceResponse](http.StatusCreated),
		[]api.ErrorString{
			api.BodySizeExceeded,
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.Unauthorized,
			api.UserNotFound,
			api.DatabaseError,
			api.BadIDGeneration,
			api.InsufficientEntropy,
		},
	))
	h.mux.Handle("GET /resolver/namespaces/{namespace}", h.lowlevel.Open(

		h.getNamespaceDetail,
		lowlevel.FixedStatusCode[*api.NamespaceResponse](http.StatusOK),
		[]api.ErrorString{
			api.InvalidNamespaceID,
			api.Unauthorized,
			api.Forbidden,
			api.NamespaceNotFound,
			api.DatabaseError,
		},
	))
	h.mux.Handle("GET /resolver/namespaces/{namespace}/mounts", h.lowlevel.Open(

		h.listNamespaceMounts,
		lowlevel.FixedStatusCode[*api.PaginatedBaseURIResponse](http.StatusOK),
		[]api.ErrorString{
			api.InvalidNamespaceID,
			api.InvalidQueryParameter,
			api.Unauthorized,
			api.Forbidden,
			api.NamespaceNotFound,
			api.DatabaseError,
		},
	))

	h.mux.Handle("GET /resolver/namespaces/{namespace}/resources", h.lowlevel.Open(

		h.listResources,
		lowlevel.FixedStatusCode[*api.PaginatedResourcesResponse](http.StatusOK),
		[]api.ErrorString{
			api.InvalidNamespaceID,
			api.InvalidQueryParameter,
			api.Unauthorized,
			api.Forbidden,
			api.NamespaceNotFound,
			api.DatabaseError,
		},
	))

	h.mux.Handle("POST /resolver/namespaces/{namespace}/resources", h.lowlevel.Open(

		h.createResource,
		lowlevel.FixedStatusCode[*api.ResourceResponse](http.StatusCreated),
		[]api.ErrorString{
			api.BodySizeExceeded,
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.InvalidNamespaceID,
			api.Unauthorized,
			api.Forbidden,
			api.NamespaceNotFound,
			api.DatabaseError,
			api.BadIDGeneration,
			api.InsufficientEntropy,
		},
	))
	h.mux.Handle("POST /resolver/namespaces/{namespace}/resources/batch", h.lowlevel.Open(

		h.batchCreateResources,
		lowlevel.FixedStatusCode[[]api.ResourceResponse](http.StatusCreated),
		[]api.ErrorString{
			api.BodySizeExceeded,
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.ItemLimitExceeded,
			api.InvalidNamespaceID,
			api.Unauthorized,
			api.Forbidden,
			api.NamespaceNotFound,
			api.DatabaseError,
			api.BadIDGeneration,
			api.InsufficientEntropy,
		},
	))

	h.mux.Handle("GET /resolver/namespaces/{namespace}/resources/{pid}", h.lowlevel.Open(

		h.getResource,
		func(result api.ResourceGetResult) int {
			if redacted, ok := result.(api.RedactedResourceResponse); ok {
				return redacted.StatusCode()
			}
			return http.StatusOK
		},
		[]api.ErrorString{
			api.InvalidNamespaceID,
			api.InvalidPID,
			api.Unauthorized,
			api.NamespaceNotFound,
			api.ResourceNotFound,
			api.DatabaseError,
		},
	))
	h.mux.Handle("PATCH /resolver/namespaces/{namespace}/resources/{pid}", h.lowlevel.Open(

		h.updateResource,
		lowlevel.FixedStatusCode[*api.ResourceResponse](http.StatusOK),
		[]api.ErrorString{
			api.BodySizeExceeded,
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.InvalidNamespaceID,
			api.InvalidPID,
			api.Unauthorized,
			api.Forbidden,
			api.DatabaseError,
			api.NamespaceNotFound,
			api.ResourceNotFound,
		},
	))

	h.mux.Handle("GET /resolver/namespaces/{namespace}/roles", h.lowlevel.Restricted(

		h.listNamespaceRoles,
		lowlevel.FixedStatusCode[*api.PaginatedNamespaceRolesResponse](http.StatusOK),
		[]api.ErrorString{
			api.InvalidNamespaceID,
			api.InvalidQueryParameter,
			api.Unauthorized,
			api.Forbidden,
			api.NamespaceNotFound,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))
	h.mux.Handle("GET /resolver/namespaces/{namespace}/roles/{username}", h.lowlevel.Restricted(

		h.getNamespaceRole,
		lowlevel.FixedStatusCode[*api.NamespaceRole](http.StatusOK),
		[]api.ErrorString{
			api.InvalidNamespaceID,
			api.InvalidUsername,
			api.Unauthorized,
			api.Forbidden,
			api.NamespaceNotFound,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))
	h.mux.Handle("PUT /resolver/namespaces/{namespace}/roles/{username}", h.lowlevel.Restricted(

		h.setNamespaceRole,
		lowlevel.FixedStatusCode[*api.NamespaceRole](http.StatusOK),
		[]api.ErrorString{
			api.BodySizeExceeded,
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.InvalidNamespaceID,
			api.InvalidUsername,
			api.InvalidRole,
			api.Unauthorized,
			api.Forbidden,
			api.NamespaceNotFound,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))
	h.mux.Handle("DELETE /resolver/namespaces/{namespace}/roles/{username}", h.lowlevel.Restricted(

		h.deleteNamespaceRole,
		lowlevel.FixedStatusCode[struct{}](http.StatusNoContent),
		[]api.ErrorString{
			api.InvalidNamespaceID,
			api.InvalidUsername,
			api.Unauthorized,
			api.Forbidden,
			api.NamespaceNotFound,
			api.RoleNotFound,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))

	h.mux.Handle("GET /user", h.lowlevel.Restricted(

		h.getCurrentUserHTTP,
		lowlevel.FixedStatusCode[*api.UserInfo](http.StatusOK),
		[]api.ErrorString{
			api.InvalidQueryParameter,
			api.InvalidUsername,
			api.Unauthorized,
			api.Forbidden,
			api.UserNotFound,
			api.UnavailableInAnonymousMode,
		},
	))
	h.mux.Handle("GET /user/roles", h.lowlevel.Restricted(

		h.listUserRoles,
		lowlevel.FixedStatusCode[*api.PaginatedUserRolesResponse](http.StatusOK),
		[]api.ErrorString{
			api.InvalidQueryParameter,
			api.InvalidUsername,
			api.Unauthorized,
			api.Forbidden,
			api.UserNotFound,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))
	h.mux.Handle("PATCH /user", h.lowlevel.Restricted(

		h.updateCurrentUser,
		lowlevel.FixedStatusCode[*api.UserInfo](http.StatusOK),
		[]api.ErrorString{
			api.BodySizeExceeded,
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.InvalidQueryParameter,
			api.InvalidUsername,
			api.Unauthorized,
			api.Forbidden,
			api.UserNotFound,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))
	h.mux.Handle("POST /user", h.lowlevel.Restricted(

		h.createUser,
		lowlevel.FixedStatusCode[*api.UserInfo](http.StatusCreated),
		[]api.ErrorString{
			api.BodySizeExceeded,
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.InvalidUsername,
			api.Unauthorized,
			api.Forbidden,
			api.DuplicateUsername,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))
	h.mux.Handle("DELETE /user", h.lowlevel.Restricted(

		h.deleteUser,
		lowlevel.FixedStatusCode[struct{}](http.StatusNoContent),
		[]api.ErrorString{
			api.InvalidQueryParameter,
			api.InvalidUsername,
			api.Unauthorized,
			api.Forbidden,
			api.UserNotFound,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))
	h.mux.Handle("POST /user/password", h.lowlevel.Restricted(

		h.setUserPassword,
		lowlevel.FixedStatusCode[*api.SetPasswordResponse](http.StatusOK),
		[]api.ErrorString{
			api.BodySizeExceeded,
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.InvalidPassword,
			api.InvalidUsername,
			api.Unauthorized,
			api.Forbidden,
			api.UserNotFound,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))
	h.mux.Handle("GET /users/autocomplete", h.lowlevel.Restricted(

		h.autocompleteUsers,
		lowlevel.FixedStatusCode[[]string](http.StatusOK),
		[]api.ErrorString{
			api.InvalidQueryParameter,
			api.InvalidAutocompleteQuery,
			api.Unauthorized,
			api.Forbidden,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))
	h.mux.Handle("GET /users", h.lowlevel.Restricted(

		h.listUsers,
		lowlevel.FixedStatusCode[*api.PaginatedUsersResponse](http.StatusOK),
		[]api.ErrorString{
			api.InvalidQueryParameter,
			api.Unauthorized,
			api.Forbidden,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))
	h.mux.Handle("GET /user/key", h.lowlevel.Restricted(

		h.listKeys,
		lowlevel.FixedStatusCode[*api.PaginatedAPIKeysResponse](http.StatusOK),
		[]api.ErrorString{
			api.InvalidQueryParameter,
			api.InvalidUsername,
			api.Unauthorized,
			api.Forbidden,
			api.UserNotFound,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))
	h.mux.Handle("POST /user/key", h.lowlevel.Restricted(

		h.issueKey,
		lowlevel.FixedStatusCode[*api.IssueKeyResponse](http.StatusCreated),
		[]api.ErrorString{
			api.BodySizeExceeded,
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.InvalidQueryParameter,
			api.InvalidUsername,
			api.Unauthorized,
			api.Forbidden,
			api.UserNotFound,
			api.BadIDGeneration,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))
	h.mux.Handle("POST /user/key/revoke", h.lowlevel.Restricted(

		h.revokeKey,
		lowlevel.FixedStatusCode[struct{}](http.StatusNoContent),
		[]api.ErrorString{
			api.BodySizeExceeded,
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.InvalidQueryParameter,
			api.InvalidUsername,
			api.Unauthorized,
			api.Forbidden,
			api.KeyNotFound,
			api.UserNotFound,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))

	h.mux.Handle("GET /openapi.yaml", h.handleOpenAPISpec())
	h.mux.Handle("/", h.handleSwaggerUI())

	return h
}

func (h *Server) handleSwaggerUI() http.HandlerFunc {
	handler := v5emb.NewHandlerWithConfig(swgui.Config{
		Title:            "PID Resolver API",
		SwaggerJSON:      h.ops.MountPath + "/openapi.yaml",
		BasePath:         h.ops.MountPath + "/",
		InternalBasePath: "/",
	})
	return func(w http.ResponseWriter, r *http.Request) {
		// read options while holding the lock, then immediately release!
		h.m.RLock()
		disableSwaggerUI := h != nil && h.ops.DisableSwaggerUI
		h.m.RUnlock()

		if disableSwaggerUI {
			http.NotFound(w, r)
			return
		}
		handler.ServeHTTP(w, r)
	}
}

// SetOptions updates HTTP and service options.
// It is safe to call this method concurrently with ServeHTTP.
func (h *Server) SetOptions(options Options) {
	h.m.Lock()
	defer h.m.Unlock()

	options = options.withValidValues()
	h.ops.MountPath = options.MountPath
	h.ops.DisableSwaggerUI = options.DisableSwaggerUI
	h.ops.Limits = options.Limits
	h.ops.InfoEnabled = options.InfoEnabled
	h.ops.Meta = options.Meta

	h.svc.SetOptions(service.Options{
		Limits:      options.Limits,
		InfoEnabled: options.InfoEnabled,
		Anonymous:   options.Anonymous,
	})
}

// ServeHTTP implements [http.Handler].
func (h *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.createdViaNew {
		panic("not initialized via NewHandler")
	}

	h.m.RLock()
	defer h.m.RUnlock()

	h.mux.ServeHTTP(w, r)
}

var openapiYAML = []byte(quickpid.Spec())

func (h *Server) handleOpenAPISpec() http.HandlerFunc {
	processed, err := openapi.Rewrite(openapiYAML, openapi.Server{
		MountPath: h.ops.MountPath,
	})
	if err != nil {
		h.logger.Error("failed to preprocess openapi.yaml", slog.Any("error", err))
		processed = openapiYAML
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(processed)
	}
}
