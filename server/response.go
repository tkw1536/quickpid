package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/tkw1536/quickpid/spec"
)

func handle[T any](h *Handler, impl func(w http.ResponseWriter, r *http.Request) (T, spec.Error, error), successCode int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		value, specError, err := impl(w, r)
		if err != nil {
			if specError == "" {
				panic("never reached: err != nil, but specError == \"\"")
			}
			log.Printf("error: %s %d %v", specError, specError.HTTPCode(), err)
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
