//spellchecker:words server
package server

//spellchecker:words slog http sync github swaggest swgui bicpid quickpid internal openapi server lowlevel service
import (
	"log/slog"
	"net/http"
	"sync"

	"github.com/swaggest/swgui"
	"github.com/swaggest/swgui/v5emb"
	quickpid "github.com/tkw1536/bicpid"
	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/internal/openapi"
	"github.com/tkw1536/bicpid/server/internal/lowlevel"
	"github.com/tkw1536/bicpid/service"
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
		http.StatusOK,
		[]api.Error{
			api.InfoUnavailable,
		},
	))
	h.mux.Handle("GET /resolver/namespaces", lowlevel.HandleOptionalUser(
		h.authHandler,
		h.listNamespaces,
		http.StatusOK,
		[]api.Error{
			api.InvalidQueryParameter,
			api.Unauthorized,
			api.UserNotFound,
			api.DatabaseError,
		},
	))
	h.mux.Handle("GET /resolver/resources/count", lowlevel.HandleNoAuth(
		h.authHandler,
		h.countAllResources,
		http.StatusOK,
		[]api.Error{
			api.Unauthorized,
			api.Forbidden,
			api.DatabaseError,
		},
	))
	h.mux.Handle("POST /resolver/namespaces", lowlevel.HandleOptionalUser(
		h.authHandler,
		h.createNamespace,
		http.StatusCreated,
		[]api.Error{
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
		http.StatusOK,
		[]api.Error{
			api.InvalidNamespaceID,
			api.Unauthorized,
			api.Forbidden,
			api.NamespaceNotFound,
			api.DatabaseError,
		},
	))

	h.mux.Handle("GET /resolver/namespaces/{namespace}/resources", lowlevel.HandleOptionalUser(
		h.authHandler,
		h.listResources,
		http.StatusOK,
		[]api.Error{
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
		http.StatusCreated,
		[]api.Error{
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
		http.StatusCreated,
		[]api.Error{
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
		http.StatusOK,
		[]api.Error{
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
		http.StatusOK,
		[]api.Error{
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

	h.mux.Handle("GET /resolver/namespaces/{namespace}/permissions", lowlevel.HandleRequiredUserInAuthMode(h.authHandler, h.listNamespacePermissions, http.StatusOK, []api.Error{
		api.InvalidNamespaceID,
		api.InvalidQueryParameter,
		api.Unauthorized,
		api.Forbidden,
		api.NamespaceNotFound,
		api.AnonymousModeUnavailable,
		api.DatabaseError,
	}))
	h.mux.Handle("GET /resolver/namespaces/{namespace}/permissions/{username}", lowlevel.HandleRequiredUserInAuthMode(h.authHandler, h.getNamespacePermission, http.StatusOK, []api.Error{
		api.InvalidNamespaceID,
		api.InvalidUsername,
		api.Unauthorized,
		api.Forbidden,
		api.NamespaceNotFound,
		api.AnonymousModeUnavailable,
		api.DatabaseError,
	}))
	h.mux.Handle("PUT /resolver/namespaces/{namespace}/permissions/{username}", lowlevel.HandleRequiredUserInAuthMode(h.authHandler, h.setNamespacePermission, http.StatusOK, []api.Error{
		api.BodySizeExceeded,
		api.BodyMissing,
		api.BodyInvalidJSON,
		api.InvalidNamespaceID,
		api.InvalidUsername,
		api.Unauthorized,
		api.Forbidden,
		api.NamespaceNotFound,
		api.AnonymousModeUnavailable,
		api.DatabaseError,
	}))
	h.mux.Handle("DELETE /resolver/namespaces/{namespace}/permissions/{username}", lowlevel.HandleRequiredUserInAuthMode(h.authHandler, h.deleteNamespacePermission, http.StatusNoContent, []api.Error{
		api.InvalidNamespaceID,
		api.InvalidUsername,
		api.Unauthorized,
		api.Forbidden,
		api.NamespaceNotFound,
		api.AnonymousModeUnavailable,
		api.DatabaseError,
	}))

	h.mux.Handle("GET /user/", lowlevel.HandleRequiredUserInAuthMode(h.authHandler, h.getCurrentUserHTTP, http.StatusOK, []api.Error{
		api.Unauthorized,
		api.AnonymousModeUnavailable,
	}))
	h.mux.Handle("PATCH /user/", lowlevel.HandleRequiredUserInAuthMode(h.authHandler, h.updateCurrentUser, http.StatusOK, []api.Error{
		api.BodySizeExceeded,
		api.BodyMissing,
		api.BodyInvalidJSON,
		api.InvalidUsername,
		api.Unauthorized,
		api.Forbidden,
		api.UserNotFound,
		api.AnonymousModeUnavailable,
		api.DatabaseError,
	}))
	h.mux.Handle("POST /user/", lowlevel.HandleRequiredUserInAuthMode(h.authHandler, h.createUser, http.StatusCreated, []api.Error{
		api.BodySizeExceeded,
		api.BodyMissing,
		api.BodyInvalidJSON,
		api.InvalidUsername,
		api.Unauthorized,
		api.Forbidden,
		api.DuplicateUsername,
		api.AnonymousModeUnavailable,
		api.DatabaseError,
	}))
	h.mux.Handle("GET /users/", lowlevel.HandleRequiredUserInAuthMode(h.authHandler, h.listUsers, http.StatusOK, []api.Error{
		api.InvalidQueryParameter,
		api.Unauthorized,
		api.Forbidden,
		api.AnonymousModeUnavailable,
		api.DatabaseError,
	}))
	h.mux.Handle("GET /user/key", lowlevel.HandleRequiredUserInAuthMode(h.authHandler, h.listKeys, http.StatusOK, []api.Error{
		api.InvalidQueryParameter,
		api.Unauthorized,
		api.UserNotFound,
		api.AnonymousModeUnavailable,
		api.DatabaseError,
	}))
	h.mux.Handle("POST /user/key", lowlevel.HandleRequiredUserInAuthMode(h.authHandler, h.issueKey, http.StatusCreated, []api.Error{
		api.BodySizeExceeded,
		api.BodyMissing,
		api.BodyInvalidJSON,
		api.InvalidUsername,
		api.Unauthorized,
		api.UserNotFound,
		api.BadIDGeneration,
		api.AnonymousModeUnavailable,
		api.DatabaseError,
	}))
	h.mux.Handle("POST /user/key/revoke", lowlevel.HandleRequiredUserInAuthMode(h.authHandler, h.revokeKey, http.StatusOK, []api.Error{
		api.BodySizeExceeded,
		api.BodyMissing,
		api.BodyInvalidJSON,
		api.Unauthorized,
		api.KeyNotFound,
		api.UserNotFound,
		api.AnonymousModeUnavailable,
		api.DatabaseError,
	}))

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
