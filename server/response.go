package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/spec"
)

// writeJSONResponse writes a JSON response to the client.
func writeJSONResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

var apiClientErrors = []struct {
	sentinel error
	status   int
}{
	{errEmptyRequestBody, http.StatusBadRequest},
	{errRequestBodyTooLarge, http.StatusRequestEntityTooLarge},

	{errInvalidJSON, http.StatusBadRequest},
	{errTrailingJSON, http.StatusBadRequest},

	{errInvalidQueryParameter, http.StatusBadRequest},

	{errInvalidNamespaceID, http.StatusBadRequest},
	{errInvalidPID, http.StatusBadRequest},
	{errTooManyItems, http.StatusBadRequest},

	{backend.ErrNamespaceNotFound, http.StatusNotFound},
	{backend.ErrResourceNotFound, http.StatusNotFound},

	{errUnableToAllocatePID, http.StatusInternalServerError},
	{errUnableToAllocateNamespaceID, http.StatusInternalServerError},
}

// writeError writes an error response to the client.
func writeError(w http.ResponseWriter, err error) {
	for _, e := range apiClientErrors {
		if errors.Is(err, e.sentinel) {
			writeJSONResponse(w, e.status, spec.ErrorResponse{Error: err.Error()})
			return
		}
	}
	writeJSONResponse(w, http.StatusInternalServerError, spec.ErrorResponse{Error: "internal server error"})
}
