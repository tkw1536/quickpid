package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/tkw1536/quickpid/api"
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
	{api.ErrEmptyRequestBody, http.StatusBadRequest},
	{api.ErrInvalidJSON, http.StatusBadRequest},
	{api.ErrTrailingJSON, http.StatusBadRequest},
	{api.ErrInvalidQueryParameter, http.StatusBadRequest},
	{api.ErrInvalidNamespace, http.StatusBadRequest},
	{api.ErrInvalidPID, http.StatusBadRequest},
	{api.ErrTooManyItems, http.StatusBadRequest},
	{api.ErrNamespaceNotFound, http.StatusNotFound},
	{api.ErrResourceNotFound, http.StatusNotFound},
	{api.ErrRequestBodyTooLarge, http.StatusRequestEntityTooLarge},
	{api.ErrPIDAllocationFailed, http.StatusInternalServerError},
	{api.ErrNamespaceIDAllocationFailed, http.StatusInternalServerError},
}

// writeError writes an error response to the client.
func writeError(w http.ResponseWriter, err error) {
	for _, e := range apiClientErrors {
		if errors.Is(err, e.sentinel) {
			writeJSONResponse(w, e.status, api.ErrorResponse{Error: err.Error()})
			return
		}
	}
	writeJSONResponse(w, http.StatusInternalServerError, api.ErrorResponse{Error: "internal server error"})
}
