//spellchecker:words lowlevel
package lowlevel

//spellchecker:words context encoding json slog http time
import (
	"context"
	"encoding/json/v2"
	"log/slog"
	"net/http"
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

	if err := json.MarshalWrite(w, v); err != nil {
		h.logInternal(r.Context(), r, slog.LevelError, "error writing json response", slog.Any("error", err))
	}
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
	case status >= 200 && status < 400:
		level = slog.LevelDebug
		msg = "request success"
	case status >= 400 && status < 500:
		level = slog.LevelDebug
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
