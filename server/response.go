package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/tkw1536/quickpid/api"
)

func handle[T any](
	h *Handler,
	impl func(w http.ResponseWriter, r *http.Request) (T, api.Error, error),
	successCode int,
	allowedErrors []api.Error,
) http.HandlerFunc {
	errors := make(map[api.Error]struct{})
	for _, err := range allowedErrors {
		errors[err] = struct{}{}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		value, specError, err := impl(w, r)
		duration := time.Since(start)

		if err != nil {
			if _, ok := errors[specError]; !ok {
				panic("implementation error: unexpected error returned")
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

		if specError != "" {
			panic("never reached: specError != \"\", but err == nil")
		}

		h.log(
			r.Context(),
			r,
			duration,
			successCode,
		)
		writeJSONResponse(w, successCode, value)
	}
}

// writeJSONResponse writes a JSON response to the client.
func writeJSONResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) log(ctx context.Context, r *http.Request, duration time.Duration, status int, extra ...any) {
	if h.logger == nil {
		return
	}

	var (
		level slog.Level = slog.LevelInfo
		msg   string     = "unknown event"
	)
	switch {
	case status >= 200 && status < 300: // Success
		level = slog.LevelInfo
		msg = "request success"
	case status >= 400 && status < 500: // Client Error
		level = slog.LevelWarn
		msg = "request client error"
	case status >= 500 && status < 600: // Server Error
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
