//spellchecker:words server
package server

//spellchecker:words encoding json errors http strconv github bicpid service
import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/tkw1536/bicpid/api"
	"github.com/tkw1536/bicpid/service"
)

var errTrailingJSON = errors.New("trailing json after value")

var (
	errLimitInvalid            = errors.New("invalid limit")
	errLimitMustBeNonNegative  = errors.New("limit must be non-negative")
	errOffsetInvalid           = errors.New("invalid offset")
	errOffsetMustBeNonNegative = errors.New("offset must be non-negative")
	errDeletedInvalid          = errors.New("invalid deleted query parameter")
)

// decodeJSON decodes the request body into v.
//
// It can return the following errors:
//
// - [api.BodySizeExceeded]
// - [api.BodyMissing]
// - [api.BodyInvalidJSON]
func (h *Server) decodeJSON(w http.ResponseWriter, r *http.Request, v any) (api.Error, error) {
	var body io.ReadCloser = r.Body
	if maxBody := h.svc.Options().Limits.MaxBodyBytes; maxBody > 0 {
		body = http.MaxBytesReader(w, body, maxBody)
	}
	defer body.Close()

	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return api.BodySizeExceeded, err
		}
		if errors.Is(err, io.EOF) {
			return api.BodyMissing, err
		}
		return api.BodyInvalidJSON, err
	}
	_, err := dec.Token()
	if !errors.Is(err, io.EOF) || err == nil {
		if err == nil {
			err = errTrailingJSON
		}
		return api.BodyInvalidJSON, err
	}
	return "", nil
}

// parsePagination parses pagination parameters from the query string.
//
// A limit of 0 means no items are returned; the list handler still computes total.
//
// It can return the following errors:
//
// - [api.InvalidQueryParameter]
func (h *Server) parsePagination(r *http.Request) (limit int, offset int, specError api.Error, err error) {
	query := r.URL.Query()
	limits := h.svc.Options().Limits

	limit = limits.DefaultPageLimit
	if query.Has("limit") {
		limit, err = strconv.Atoi(query.Get("limit"))
		if err != nil {
			return 0, 0, api.InvalidQueryParameter, fmt.Errorf("%w: %w", errLimitInvalid, err)
		}
		if limit < 0 {
			return 0, 0, api.InvalidQueryParameter, errLimitMustBeNonNegative
		}
	}
	if limits.MaxPageLimit > 0 && limit > limits.MaxPageLimit {
		limit = limits.MaxPageLimit
	}

	offset = 0
	if query.Has("offset") {
		offset, err = strconv.Atoi(query.Get("offset"))
		if err != nil {
			return 0, 0, api.InvalidQueryParameter, fmt.Errorf("%w: %w", errOffsetInvalid, err)
		}
		if offset < 0 {
			return 0, 0, api.InvalidQueryParameter, errOffsetMustBeNonNegative
		}
	}
	return limit, offset, "", nil
}

// getNamespace gets the namespace from the request path.
//
// It can return the following errors:
//
// - [api.InvalidNamespaceID]
func (*Server) getNamespace(r *http.Request) (namespace string, specError api.Error, err error) {
	namespace = r.PathValue("namespace")
	if err := service.ValidateNamespaceID(namespace); err != nil {
		return "", api.InvalidNamespaceID, err
	}
	return namespace, "", nil
}

// getPID gets the pid from the request path.
//
// It can return the following errors:
//
// - [api.InvalidPID]
func (*Server) getPID(r *http.Request) (pid string, specError api.Error, err error) {
	pid = r.PathValue("pid")
	if err := service.ValidatePID(pid); err != nil {
		return "", api.InvalidPID, err
	}
	return pid, "", nil
}

// getUsername gets the username from the request path.
func (*Server) getUsername(r *http.Request) (username string, specError api.Error, err error) {
	username = r.PathValue("username")
	if err := service.ValidateUsername(username); err != nil {
		return "", api.InvalidUsername, err
	}
	return username, "", nil
}
