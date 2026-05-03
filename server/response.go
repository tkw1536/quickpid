package server

import (
	"encoding/json"
	"net/http"

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
		value, specError, err := impl(w, r)
		if err != nil {
			if _, ok := errors[specError]; !ok {
				panic("implementation error: unexpected error returned")
			}

			writeJSONResponse(w, specError.HTTPCode(), api.ErrorResponse{Error: string(specError)})
			return
		}

		if specError != "" {
			panic("never reached: specError != \"\", but err == nil")
		}
		writeJSONResponse(w, successCode, value)
	}
}

// writeJSONResponse writes a JSON response to the client.
func writeJSONResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
