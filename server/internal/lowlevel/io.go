//spellchecker:words lowlevel
package lowlevel

//spellchecker:words context encoding json slog http strings time
import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// writeJSONResponse writes a JSON response to the client.
func (h *Handler) writeJSONResponse(w http.ResponseWriter, r *http.Request, status int, v any) {
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.logInternal(r.Context(), r, slog.LevelError, "error writing json response", slog.Any("error", err))
	}
}

// readCredentials reads [credentials] from a request.
// The request object must not be nil.
func readCredentials(r *http.Request) credentials {
	var creds credentials

	for _, auth := range r.Header.Values("Authorization") {
		switch {
		case strings.HasPrefix(auth, "Bearer "):
			token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
			if token == "" || creds.hasBearer() {
				creds.invalid = true
				continue
			}
			creds.bearerToken = token
		case strings.HasPrefix(auth, "Basic "):
			username, password, ok := r.BasicAuth()
			if !ok || creds.hasBasic() {
				creds.invalid = true
				continue
			}
			creds.basicUsername = username
			creds.basicPassword = password
		default:
			creds.invalid = true
		}
	}

	if creds.hasBearer() && creds.hasBasic() {
		creds.invalid = true
	}

	return creds
}

type credentials struct {
	bearerToken   string
	basicUsername string
	basicPassword string
	invalid       bool
}

func (c credentials) hasBearer() bool {
	return c.bearerToken != ""
}

func (c credentials) hasBasic() bool {
	return c.basicUsername != "" || c.basicPassword != ""
}

// Log writes a structured request log entry.
func (h *Handler) Log(ctx context.Context, r *http.Request, duration time.Duration, status int, extra ...any) {
	if h.logger == nil {
		return
	}

	var (
		level = slog.LevelInfo
		msg   = "unknown event"
	)

	switch {
	case status >= 200 && status < 300:
		level = slog.LevelInfo
		msg = "request success"
	case status >= 400 && status < 500:
		level = slog.LevelWarn
		msg = "request client error"
	case status >= 500 && status < 600:
		level = slog.LevelError
		msg = "request server error"
	}

	extra = append([]any{
		slog.Duration("duration", duration),
		slog.Int("status", status),
	}, extra...)

	h.logInternal(ctx, r, level, msg, extra...)
}

func (h *Handler) logInternal(ctx context.Context, r *http.Request, level slog.Level, msg string, extra ...any) {
	if h.logger == nil {
		return
	}

	attrs := make([]any, 0, 3+len(extra))
	attrs = append(attrs, slog.String("method", r.Method))
	attrs = append(attrs, slog.String("uri", r.URL.RequestURI()))
	attrs = append(attrs, slog.String("remote_addr", r.RemoteAddr))
	attrs = append(attrs, extra...)
	h.logger.Log(ctx, level, msg, attrs...)
}
