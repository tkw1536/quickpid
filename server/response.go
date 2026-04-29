package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/tkw1536/quickpid/backend"
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
	{backend.ErrEmptyRequestBody, http.StatusBadRequest},
	{backend.ErrInvalidJSON, http.StatusBadRequest},
	{backend.ErrTrailingJSON, http.StatusBadRequest},
	{backend.ErrInvalidQueryParameter, http.StatusBadRequest},
	{backend.ErrInvalidNamespace, http.StatusBadRequest},
	{backend.ErrInvalidPID, http.StatusBadRequest},
	{backend.ErrTooManyItems, http.StatusBadRequest},
	{backend.ErrNamespaceNotFound, http.StatusNotFound},
	{backend.ErrResourceNotFound, http.StatusNotFound},
	{backend.ErrRequestBodyTooLarge, http.StatusRequestEntityTooLarge},
	{backend.ErrPIDAllocationFailed, http.StatusInternalServerError},
	{backend.ErrNamespaceIDAllocationFailed, http.StatusInternalServerError},
}

// writeError writes an error response to the client.
func writeError(w http.ResponseWriter, err error) {
	for _, e := range apiClientErrors {
		if errors.Is(err, e.sentinel) {
			writeJSONResponse(w, e.status, backend.ErrorResponse{Error: err.Error()})
			return
		}
	}
	writeJSONResponse(w, http.StatusInternalServerError, backend.ErrorResponse{Error: "internal server error"})
}
