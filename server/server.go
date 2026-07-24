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

	logger      *slog.Logger
	authHandler *lowlevel.AuthHandler

	createdViaNew bool
}

// NewServer returns an [Server for the PID Resolver API and Swagger UI.
//
// Routes on the returned handler are rooted at / (e.g. GET /resolver/namespaces);
// mount with http.StripPrefix(mountPath, NewServer(Options{MountPath: mountPath}, svc)) at mountPath+"/".
func NewServer(options Options, svc *service.Service, logger *slog.Logger) *Server {
	options = options.withValidValues()

	h := &Server{
		svc:         svc,
		ops:         options,
		mux:         http.NewServeMux(),
		logger:      logger,
		authHandler: lowlevel.NewAuthHandler(svc, logger),

		createdViaNew: true,
	}

	h.mux.Handle("GET /resolver", lowlevel.HandleNoAuth(
		h.authHandler,
		h.getResolverInfo,
		lowlevel.FixedStatusCode[*api.InfoResponse](http.StatusOK),
		[]api.ErrorString{
			api.InfoUnavailable,
		},
	))
	h.mux.Handle("GET /resolver/info", lowlevel.HandleNoAuth(
		h.authHandler,
		h.getResolverMeta,
		lowlevel.FixedStatusCode[*api.MetaResponse](http.StatusOK),
		nil,
	))
	h.mux.Handle("GET /resolver/namespaces", lowlevel.HandleOptionalUser(
		h.authHandler,
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
	h.mux.Handle("GET /resolver/resources/count", lowlevel.HandleNoAuth(
		h.authHandler,
		h.countAllResources,
		lowlevel.FixedStatusCode[*api.ResourceCountResponse](http.StatusOK),
		[]api.ErrorString{
			api.Unauthorized,
			api.Forbidden,
			api.DatabaseError,
		},
	))
	h.mux.Handle("GET /resolver/mounts", lowlevel.HandleOptionalUser(
		h.authHandler,
		h.listMounts,
		lowlevel.FixedStatusCode[*api.PaginatedMountsResponse](http.StatusOK),
		[]api.ErrorString{
			api.InvalidQueryParameter,
			api.Unauthorized,
			api.Forbidden,
			api.DatabaseError,
		},
	))
	h.mux.Handle("GET /resolver/mounts/{base_uri}", lowlevel.HandleNoAuth(
		h.authHandler,
		h.getMount,
		lowlevel.FixedStatusCode[*api.MountResponse](http.StatusOK),
		[]api.ErrorString{
			api.InvalidBaseURI,
			api.MountNotFound,
			api.DatabaseError,
		},
	))
	h.mux.Handle("PUT /resolver/mounts/{base_uri}", lowlevel.HandleOptionalUser(
		h.authHandler,
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
	h.mux.Handle("DELETE /resolver/mounts/{base_uri}", lowlevel.HandleOptionalUser(
		h.authHandler,
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
	h.mux.Handle("POST /resolver/namespaces", lowlevel.HandleOptionalUser(
		h.authHandler,
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
	h.mux.Handle("GET /resolver/namespaces/{namespace}", lowlevel.HandleOptionalUser(
		h.authHandler,
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
	h.mux.Handle("GET /resolver/namespaces/{namespace}/mounts", lowlevel.HandleOptionalUser(
		h.authHandler,
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

	h.mux.Handle("GET /resolver/namespaces/{namespace}/resources", lowlevel.HandleOptionalUser(
		h.authHandler,
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

	h.mux.Handle("POST /resolver/namespaces/{namespace}/resources", lowlevel.HandleOptionalUser(
		h.authHandler,
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
	h.mux.Handle("POST /resolver/namespaces/{namespace}/resources:batch", lowlevel.HandleOptionalUser(
		h.authHandler,
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

	h.mux.Handle("GET /resolver/namespaces/{namespace}/resources/{pid}", lowlevel.HandleOptionalUser(
		h.authHandler,
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
			api.Forbidden,
			api.NamespaceNotFound,
			api.ResourceNotFound,
			api.DatabaseError,
		},
	))
	h.mux.Handle("PATCH /resolver/namespaces/{namespace}/resources/{pid}", lowlevel.HandleOptionalUser(
		h.authHandler,
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

	h.mux.Handle("GET /resolver/namespaces/{namespace}/roles", lowlevel.HandleRequiredUserInAuthMode(
		h.authHandler,
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
	h.mux.Handle("GET /resolver/namespaces/{namespace}/roles/{username}", lowlevel.HandleRequiredUserInAuthMode(
		h.authHandler,
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
	h.mux.Handle("PUT /resolver/namespaces/{namespace}/roles/{username}", lowlevel.HandleRequiredUserInAuthMode(
		h.authHandler,
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
	h.mux.Handle("DELETE /resolver/namespaces/{namespace}/roles/{username}", lowlevel.HandleRequiredUserInAuthMode(
		h.authHandler,
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

	h.mux.Handle("GET /user/", lowlevel.HandleRequiredUserInAuthMode(
		h.authHandler,
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
	h.mux.Handle("GET /user/roles", lowlevel.HandleRequiredUserInAuthMode(
		h.authHandler,
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
	h.mux.Handle("PATCH /user/", lowlevel.HandleRequiredUserInAuthMode(
		h.authHandler,
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
	h.mux.Handle("POST /user/", lowlevel.HandleRequiredUserInAuthMode(
		h.authHandler,
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
	h.mux.Handle("POST /user/password", lowlevel.HandleRequiredUserInAuthMode(
		h.authHandler,
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
	h.mux.Handle("GET /users/autocomplete", lowlevel.HandleRequiredUserInAuthMode(
		h.authHandler,
		h.autocompleteUsers,
		lowlevel.FixedStatusCode[[]string](http.StatusOK),
		[]api.ErrorString{
			api.InvalidQueryParameter,
			api.InvalidUsername,
			api.Unauthorized,
			api.Forbidden,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))
	h.mux.Handle("GET /users/", lowlevel.HandleRequiredUserInAuthMode(
		h.authHandler,
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
	h.mux.Handle("GET /user/key", lowlevel.HandleRequiredUserInAuthMode(
		h.authHandler,
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
	h.mux.Handle("POST /user/key", lowlevel.HandleRequiredUserInAuthMode(
		h.authHandler,
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
	h.mux.Handle("POST /user/key/revoke", lowlevel.HandleRequiredUserInAuthMode(
		h.authHandler,
		h.revokeKey,
		lowlevel.FixedStatusCode[*api.RevokeKeyResponse](http.StatusOK),
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
