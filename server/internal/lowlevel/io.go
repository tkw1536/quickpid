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
func writeJSONResponse(w http.ResponseWriter, status int, v any) {
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// readBearerToken extracts a bearer token from the Authorization header.
func readBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	token, found := strings.CutPrefix(auth, "Bearer ")
	if !found {
		return ""
	}
	return strings.TrimSpace(token)
}

// Log writes a structured request log entry.
func (h *AuthHandler) Log(ctx context.Context, r *http.Request, duration time.Duration, status int, extra ...any) {
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

	attrs := []any{
		slog.String("method", r.Method),
		slog.String("uri", r.URL.RequestURI()),
		slog.String("remote_addr", r.RemoteAddr),
		slog.Duration("duration", duration),
		slog.Int("status", status),
	}
	h.logger.Log(ctx, level, msg, append(attrs, extra...)...)
}
