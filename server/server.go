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

	// All the documented resolver routes.
	// These should remain in sync with the openapi.yaml file:
	// In particular if you update this file make sure to:
	//
	// 1. keep the same order of routes.
	// 2. keep the same names for path variable names as in the spec.
	// 3. keep an empty line between different paths, but no space between handlers for the same path but different method.
	// 4. keep the error strings matching any 'Error*' spec type, in the same order.
	// 5. name the internal the same as the 'operationId' in the spec.

	h.mux.Handle("GET /resolver", h.lowlevel.Public(
		h.getResolverInfo,
		lowlevel.FixedStatusCode[*api.InfoResponse](http.StatusOK),
		[]api.ErrorCode{
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
		[]api.ErrorCode{
			api.InvalidQueryParameter,
			api.Unauthorized,
			api.DatabaseError,
		},
	))

	h.mux.Handle("POST /resolver/namespaces", h.lowlevel.Open(
		h.createNamespace,
		lowlevel.FixedStatusCode[*api.NamespaceResponse](http.StatusCreated),
		[]api.ErrorCode{
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.Unauthorized,
			api.BodySizeExceeded,
			api.DatabaseError,
			api.BadIDGeneration,
			api.InsufficientEntropy,
		},
	))

	h.mux.Handle("GET /resolver/resources/count", h.lowlevel.Public(
		h.countAllResources,
		lowlevel.FixedStatusCode[*api.ResourceCountResponse](http.StatusOK),
		[]api.ErrorCode{
			api.DatabaseError,
		},
	))

	h.mux.Handle("GET /resolver/mounts", h.lowlevel.Open(
		h.listMounts,
		lowlevel.FixedStatusCode[*api.PaginatedMountsResponse](http.StatusOK),
		[]api.ErrorCode{
			api.InvalidQueryParameter,
			api.Unauthorized,
			api.Forbidden,
			api.DatabaseError,
		},
	))

	h.mux.Handle("GET /resolver/mounts/{baseUri}", h.lowlevel.Public(
		h.resolveMountByBaseUri,
		lowlevel.FixedStatusCode[*api.MountResponse](http.StatusOK),
		[]api.ErrorCode{
			api.InvalidBaseURI,
			api.MountNotFound,
			api.DatabaseError,
		},
	))
	h.mux.Handle("PUT /resolver/mounts/{baseUri}", h.lowlevel.Open(
		h.upsertNamespaceMount,
		lowlevel.FixedStatusCode[*api.MountResponse](http.StatusOK),
		[]api.ErrorCode{
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.InvalidBaseURI,
			api.InvalidNamespaceID,
			api.Unauthorized,
			api.Forbidden,
			api.NamespaceNotFound,
			api.BodySizeExceeded,
			api.DatabaseError,
		},
	))

	h.mux.Handle("DELETE /resolver/mounts/{baseUri}", h.lowlevel.Open(
		h.deleteNamespaceMount,
		lowlevel.FixedStatusCode[struct{}](http.StatusNoContent),
		[]api.ErrorCode{
			api.InvalidBaseURI,
			api.Unauthorized,
			api.Forbidden,
			api.MountNotFound,
			api.DatabaseError,
		},
	))

	h.mux.Handle("GET /resolver/mounts/{baseUri}/{pid}", h.lowlevel.Open(
		h.resolveResourceByMountAndPID,
		lowlevel.FixedStatusCode[api.ResourceGetResult](http.StatusFound),
		[]api.ErrorCode{
			api.InvalidBaseURI,
			api.InvalidPID,
			api.Unauthorized,
			api.MountNotFound,
			api.NamespaceNotFound,
			api.ResourceNotFound,
			api.DatabaseError,
		},
	))

	h.mux.Handle("GET /resolver/namespaces/{namespace}", h.lowlevel.Open(
		h.getNamespace,
		lowlevel.FixedStatusCode[*api.NamespaceResponse](http.StatusOK),
		[]api.ErrorCode{
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
		[]api.ErrorCode{
			api.InvalidNamespaceID,
			api.InvalidQueryParameter,
			api.Unauthorized,
			api.Forbidden,
			api.NamespaceNotFound,
			api.DatabaseError,
		},
	))

	h.mux.Handle("GET /resolver/namespaces/{namespace}/roles", h.lowlevel.Restricted(
		h.listNamespaceRoles,
		lowlevel.FixedStatusCode[*api.PaginatedNamespaceRolesResponse](http.StatusOK),
		[]api.ErrorCode{
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
		[]api.ErrorCode{
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
		[]api.ErrorCode{
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.InvalidNamespaceID,
			api.InvalidUsername,
			api.InvalidRole,
			api.Unauthorized,
			api.Forbidden,
			api.NamespaceNotFound,
			api.UnavailableInAnonymousMode,
			api.BodySizeExceeded,
			api.DatabaseError,
		},
	))

	h.mux.Handle("DELETE /resolver/namespaces/{namespace}/roles/{username}", h.lowlevel.Restricted(
		h.removeNamespaceRole,
		lowlevel.FixedStatusCode[struct{}](http.StatusNoContent),
		[]api.ErrorCode{
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

	h.mux.Handle("GET /resolver/namespaces/{namespace}/resources", h.lowlevel.Open(
		h.listResources,
		lowlevel.FixedStatusCode[*api.PaginatedResourcesResponse](http.StatusOK),
		[]api.ErrorCode{
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
		[]api.ErrorCode{
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.InvalidNamespaceID,
			api.InvalidResourceURL,
			api.Unauthorized,
			api.Forbidden,
			api.BodySizeExceeded,
			api.NamespaceNotFound,
			api.DatabaseError,
			api.BadIDGeneration,
			api.InsufficientEntropy,
		},
	))

	h.mux.Handle("POST /resolver/namespaces/{namespace}/resources/batch", h.lowlevel.Open(
		h.createResourceBatch,
		lowlevel.FixedStatusCode[[]api.ResourceResponse](http.StatusCreated),
		[]api.ErrorCode{
			api.BodySizeExceeded,
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.ItemLimitExceeded,
			api.InvalidNamespaceID,
			api.InvalidResourceURL,
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
		[]api.ErrorCode{
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
		[]api.ErrorCode{
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.InvalidNamespaceID,
			api.InvalidPID,
			api.InvalidResourceURL,
			api.Unauthorized,
			api.Forbidden,
			api.NamespaceNotFound,
			api.ResourceNotFound,
			api.BodySizeExceeded,
			api.DatabaseError,
		},
	))

	h.mux.Handle("GET /user", h.lowlevel.Restricted(
		h.getUserInfo,
		lowlevel.FixedStatusCode[*api.UserInfo](http.StatusOK),
		[]api.ErrorCode{
			api.Unauthorized,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))
	h.mux.Handle("PATCH /user", h.lowlevel.Restricted(
		h.updateUser,
		lowlevel.FixedStatusCode[*api.UserInfo](http.StatusOK),
		[]api.ErrorCode{
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.InvalidQueryParameter,
			api.InvalidUsername,
			api.Unauthorized,
			api.Forbidden,
			api.UnavailableInAnonymousMode,
			api.UserNotFound,
			api.BodySizeExceeded,
			api.DatabaseError,
		},
	))
	h.mux.Handle("POST /user", h.lowlevel.Restricted(
		h.createUser,
		lowlevel.FixedStatusCode[*api.UserInfo](http.StatusCreated),
		[]api.ErrorCode{
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.InvalidUsername,
			api.Unauthorized,
			api.Forbidden,
			api.UnavailableInAnonymousMode,
			api.DuplicateUsername,
			api.BodySizeExceeded,
			api.DatabaseError,
		},
	))
	h.mux.Handle("DELETE /user", h.lowlevel.Restricted(
		h.deleteUser,
		lowlevel.FixedStatusCode[struct{}](http.StatusNoContent),
		[]api.ErrorCode{
			api.InvalidUsername,
			api.InvalidQueryParameter,
			api.Unauthorized,
			api.Forbidden,
			api.UnavailableInAnonymousMode,
			api.UserNotFound,
			api.DatabaseError,
		},
	))

	h.mux.Handle("GET /users/autocomplete", h.lowlevel.Restricted(
		h.autocompleteUsers,
		lowlevel.FixedStatusCode[[]string](http.StatusOK),
		[]api.ErrorCode{
			api.InvalidQueryParameter,
			api.InvalidAutocompleteQuery,
			api.Unauthorized,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))

	h.mux.Handle("GET /users", h.lowlevel.Restricted(
		h.listUsers,
		lowlevel.FixedStatusCode[*api.PaginatedUsersResponse](http.StatusOK),
		[]api.ErrorCode{
			api.InvalidQueryParameter,
			api.Unauthorized,
			api.Forbidden,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))

	h.mux.Handle("GET /user/roles", h.lowlevel.Restricted(
		h.getUserRoles,
		lowlevel.FixedStatusCode[*api.PaginatedUserRolesResponse](http.StatusOK),
		[]api.ErrorCode{
			api.InvalidQueryParameter,
			api.Unauthorized,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))

	h.mux.Handle("GET /user/key", h.lowlevel.Restricted(
		h.listUserKeys,
		lowlevel.FixedStatusCode[*api.PaginatedAPIKeysResponse](http.StatusOK),
		[]api.ErrorCode{
			api.InvalidQueryParameter,
			api.Unauthorized,
			api.UnavailableInAnonymousMode,
			api.DatabaseError,
		},
	))
	h.mux.Handle("POST /user/key", h.lowlevel.Restricted(
		h.issueUserKey,
		lowlevel.FixedStatusCode[*api.IssueKeyResponse](http.StatusCreated),
		[]api.ErrorCode{
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.InvalidQueryParameter,
			api.Unauthorized,
			api.UnavailableInAnonymousMode,
			api.BodySizeExceeded,
			api.DatabaseError,
			api.BadIDGeneration,
			api.InsufficientEntropy,
		},
	))

	h.mux.Handle("POST /user/password", h.lowlevel.Restricted(
		h.setUserPassword,
		lowlevel.FixedStatusCode[*api.SetPasswordResponse](http.StatusOK),
		[]api.ErrorCode{
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.InvalidPassword,
			api.Unauthorized,
			api.UnavailableInAnonymousMode,
			api.BodySizeExceeded,
			api.DatabaseError,
		},
	))

	h.mux.Handle("POST /user/key/revoke", h.lowlevel.Restricted(
		h.revokeUserKey,
		lowlevel.FixedStatusCode[struct{}](http.StatusNoContent),
		[]api.ErrorCode{
			api.BodyMissing,
			api.BodyInvalidJSON,
			api.Unauthorized,
			api.KeyNotFound,
			api.UnavailableInAnonymousMode,
			api.BodySizeExceeded,
			api.DatabaseError,
		},
	))

	// Extra routes - not in the openapi spec.
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
	processed, err := openapi.SetServersPath(openapiYAML, h.ops.MountPath)
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
