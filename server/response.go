package server

import (
	"encoding/json"
	"net/http"

	"github.com/tkw1536/quickpid/spec"
)

func handle[T any](
	h *Handler,
	impl func(w http.ResponseWriter, r *http.Request) (T, spec.Error, error),
	successCode int,
	allowedErrors []spec.Error,
) http.HandlerFunc {
	errors := make(map[spec.Error]struct{})
	for _, err := range allowedErrors {
		errors[err] = struct{}{}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		value, specError, err := impl(w, r)
		if err != nil {
			if _, ok := errors[specError]; !ok {
				panic("implementation error: unexpected error returned")
			}

			writeJSONResponse(w, specError.HTTPCode(), spec.ErrorResponse{Error: string(specError)})
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
