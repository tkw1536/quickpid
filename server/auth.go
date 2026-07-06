//spellchecker:words server
package server

//spellchecker:words errors log slog net http strings time context github quickpid backend
import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/backend"
)

var errUnauthorized = errors.New("unauthorized")

var errForbidden = errors.New("forbidden")

type authHandler struct {
	backend backend.AuthBackend
}

// Authenticate authenticates
func (a *authHandler) Authenticate(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return "", errUnauthorized
	}
	key := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	if key == "" {
		return "", errUnauthorized
	}

	username, err := a.backend.LookupUserByKey(r.Context(), key)
	if errors.Is(err, backend.ErrInvalidKey) {
		return "", errUnauthorized
	}
	if err != nil {
		return "", err
	}
	return username, nil
}

func (a *authHandler) getCurrentUser(_ http.ResponseWriter, r *http.Request) (*api.UserInfo, error) {
	return a.currentUser(r)
}

func (a *authHandler) currentUser(r *http.Request) (*api.UserInfo, error) {
	username, err := a.Authenticate(r)
	if err != nil {
		return nil, err
	}
	return a.backend.GetUser(r.Context(), username)
}

func (a *authHandler) requireSuperuser(r *http.Request) (*api.UserInfo, error) {
	user, err := a.currentUser(r)
	if err != nil {
		return nil, err
	}
	if !user.Superuser {
		return nil, errForbidden
	}
	return user, nil
}

func (a *authHandler) issueKey(ctx context.Context, username, keyID string, req api.IssueKeyRequest, now func() time.Time) (*api.IssueKeyResponse, error) {
	return a.backend.IssueKey(ctx, username, keyID, req, now)
}

func mapAuthBackendError(err error) (api.Error, bool) {
	switch {
	case errors.Is(err, backend.ErrDuplicateUsername):
		return api.DuplicateUsername, true
	case errors.Is(err, backend.ErrUserNotFound):
		return api.UserNotFound, true
	case errors.Is(err, backend.ErrKeyNotFound):
		return api.KeyNotFound, true
	default:
		return "", false
	}
}

func (h *Handler) getCurrentUserHTTP(w http.ResponseWriter, r *http.Request) (*api.UserInfo, api.Error, error) {
	user, err := h.auth.getCurrentUser(w, r)
	if errors.Is(err, errUnauthorized) {
		return nil, "", err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return user, "", nil
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) (*api.UserInfo, api.Error, error) {
	caller, err := h.auth.requireSuperuser(r)
	if errors.Is(err, errUnauthorized) || errors.Is(err, errForbidden) {
		return nil, "", err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}

	var req api.UserCreateRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}
	if req.Superuser && !caller.Superuser {
		return nil, "", errForbidden
	}

	user, err := h.auth.backend.CreateUser(r.Context(), req, h.runtime.Now)
	if specError, ok := mapAuthBackendError(err); ok {
		return nil, specError, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return user, "", nil
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) (*api.PaginatedUsersResponse, api.Error, error) {
	if _, err := h.auth.requireSuperuser(r); errors.Is(err, errUnauthorized) || errors.Is(err, errForbidden) {
		return nil, "", err
	} else if err != nil {
		return nil, api.DatabaseError, err
	}

	limit, offset, specError, err := h.parsePagination(r)
	if err != nil {
		return nil, specError, err
	}

	page, err := h.auth.backend.ListUsers(r.Context(), api.ListUsersParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return page, "", nil
}

func (h *Handler) listKeys(w http.ResponseWriter, r *http.Request) (*api.PaginatedAPIKeysResponse, api.Error, error) {
	user, err := h.auth.currentUser(r)
	if errors.Is(err, errUnauthorized) {
		return nil, "", err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}

	limit, offset, specError, err := h.parsePagination(r)
	if err != nil {
		return nil, specError, err
	}

	page, err := h.auth.backend.ListKeys(r.Context(), user.Username, api.ListKeysParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		if specError, ok := mapAuthBackendError(err); ok {
			return nil, specError, err
		}
		return nil, api.DatabaseError, err
	}
	return page, "", nil
}

func (h *Handler) issueKey(w http.ResponseWriter, r *http.Request) (*api.IssueKeyResponse, api.Error, error) {
	username, err := h.auth.Authenticate(r)
	if errors.Is(err, errUnauthorized) {
		return nil, "", err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}

	var req api.IssueKeyRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}

	keyID, err := h.runtime.NewNamespaceID()
	if err != nil {
		return nil, api.BadIDGeneration, err
	}

	issued, err := h.auth.issueKey(r.Context(), username, keyID, req, h.runtime.Now)
	if specError, ok := mapAuthBackendError(err); ok {
		return nil, specError, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return issued, "", nil
}

func (h *Handler) revokeKey(w http.ResponseWriter, r *http.Request) (*api.RevokeKeyResponse, api.Error, error) {
	username, err := h.auth.Authenticate(r)
	if errors.Is(err, errUnauthorized) {
		return nil, "", err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}

	var req api.RevokeKeyRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}

	info, err := h.auth.backend.RevokeKey(r.Context(), username, req.ID)
	if specError, ok := mapAuthBackendError(err); ok {
		return nil, specError, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return &api.RevokeKeyResponse{APIKeyInfo: *info}, "", nil
}

func (h *Handler) updateCurrentUser(w http.ResponseWriter, r *http.Request) (*api.UserInfo, api.Error, error) {
	current, err := h.auth.currentUser(r)
	if errors.Is(err, errUnauthorized) {
		return nil, "", err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}

	var req api.UpdateUserRequest
	if specError, err := h.decodeJSON(w, r, &req); err != nil {
		return nil, specError, err
	}

	target := current.Username
	if req.Username != nil {
		target = *req.Username
	}

	// only own account can be updated, or the user needs to be a superuser.
	if target != current.Username && !current.Superuser {
		return nil, "", errForbidden
	}

	// cannot change superuser flag, unless the current user is a superuser.
	if req.Superuser != nil && !current.Superuser {
		return nil, "", errForbidden
	}

	user, err := h.auth.backend.UpdateUser(r.Context(), target, req)
	if specError, ok := mapAuthBackendError(err); ok {
		return nil, specError, err
	}
	if err != nil {
		return nil, api.DatabaseError, err
	}
	return user, "", nil
}

func handleAuthReq[T any](h *Handler, impl func(w http.ResponseWriter, r *http.Request) (T, api.Error, error), successCode int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		value, specError, err := impl(w, r)
		duration := time.Since(start)

		if errors.Is(err, errUnauthorized) {
			writeUnauthorized(h, w, r, duration)
			return
		}

		if errors.Is(err, errForbidden) {
			writeForbidden(h, w, r, duration)
			return
		}

		if err != nil {
			if specError == "" {
				specError = api.DatabaseError
			}
			h.log(
				r.Context(),
				r,
				duration,
				specError.HTTPCode(),
				slog.String("error", string(specError)),
				slog.Any("cause", err),
			)
			writeJSONResponse(w, specError.HTTPCode(), api.ErrorResponse{Error: string(specError)})
			return
		}

		h.log(r.Context(), r, duration, successCode)
		writeJSONResponse(w, successCode, value)
	}
}

func writeUnauthorized(h *Handler, w http.ResponseWriter, r *http.Request, duration time.Duration) {
	h.log(
		r.Context(),
		r,
		duration,
		api.Unauthorized.HTTPCode(),
		slog.String("error", string(api.Unauthorized)),
	)
	writeJSONResponse(w, api.Unauthorized.HTTPCode(), api.ErrorResponse{Error: string(api.Unauthorized)})
}

func writeForbidden(h *Handler, w http.ResponseWriter, r *http.Request, duration time.Duration) {
	h.log(
		r.Context(),
		r,
		duration,
		api.Forbidden.HTTPCode(),
		slog.String("error", string(api.Forbidden)),
	)
	writeJSONResponse(w, api.Forbidden.HTTPCode(), api.ErrorResponse{Error: string(api.Forbidden)})
}
