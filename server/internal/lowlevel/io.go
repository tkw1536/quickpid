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
func (h *AuthHandler) writeJSONResponse(w http.ResponseWriter, r *http.Request, status int, v any) {
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

	extra = append([]any{
		slog.Duration("duration", duration),
		slog.Int("status", status),
	}, extra...)

	h.logInternal(ctx, r, level, msg, extra...)
}

func (h *AuthHandler) logInternal(ctx context.Context, r *http.Request, level slog.Level, msg string, extra ...any) {
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
