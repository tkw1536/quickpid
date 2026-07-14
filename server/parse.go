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
	errMissingUsernameQuery    = errors.New("missing username query parameter")
	errMissingQueryParameter   = errors.New("missing query query parameter")
	errSuperuserInvalid        = errors.New("invalid superuser query parameter")
)

// decodeJSON decodes the request body into v.
//
// It can return the following errors:
//
// - [api.BodySizeExceeded]
// - [api.BodyMissing]
// - [api.BodyInvalidJSON].
func (h *Server) decodeJSON(w http.ResponseWriter, r *http.Request, v any) (api.Error, error) {
	var body = r.Body
	if maxBody := h.svc.Options().Limits.MaxBodyBytes; maxBody > 0 {
		body = http.MaxBytesReader(w, body, maxBody)
	}
	defer body.Close()

	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return api.BodySizeExceeded, fmt.Errorf("json.Decode: %w", err)
		}
		if errors.Is(err, io.EOF) {
			return api.BodyMissing, fmt.Errorf("json.Decode: %w", err)
		}
		return api.BodyInvalidJSON, fmt.Errorf("json.Decode: %w", err)
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
// - [api.InvalidQueryParameter].
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
// - [api.InvalidNamespaceID].
func (*Server) getNamespace(r *http.Request) (namespace string, specError api.Error, err error) {
	namespace = r.PathValue("namespace")
	if err := service.ValidateNamespaceID(namespace); err != nil {
		return "", api.InvalidNamespaceID, fmt.Errorf("service.ValidateNamespaceID: %w", err)
	}
	return namespace, "", nil
}

// getPID gets the pid from the request path.
//
// It can return the following errors:
//
// - [api.InvalidPID].
func (*Server) getPID(r *http.Request) (pid string, specError api.Error, err error) {
	pid = r.PathValue("pid")
	if err := service.ValidatePID(pid); err != nil {
		return "", api.InvalidPID, fmt.Errorf("service.ValidatePID: %w", err)
	}
	return pid, "", nil
}

// getUsername gets the username from the request path.
func (*Server) getUsername(r *http.Request) (username string, specError api.Error, err error) {
	username = r.PathValue("username")
	if err := service.ValidateUsername(username); err != nil {
		return "", api.InvalidUsername, fmt.Errorf("service.ValidateUsername: %w", err)
	}
	return username, "", nil
}

// parseOptionalUsernameQuery parses an optional username query parameter.
//
// It can return the following errors:
//
// - [api.InvalidUsername].
func (*Server) parseOptionalUsernameQuery(r *http.Request) (target *string, specError api.Error, err error) {
	query := r.URL.Query()
	if !query.Has("username") {
		return nil, "", nil
	}
	username := query.Get("username")
	if err := service.ValidateUsername(username); err != nil {
		return nil, api.InvalidUsername, fmt.Errorf("service.ValidateUsername: %w", err)
	}
	return &username, "", nil
}

// parseRequiredUsernameQuery parses a required username query parameter.
//
// It can return the following errors:
//
// - [api.InvalidQueryParameter]
// - [api.InvalidUsername].
func (*Server) parseRequiredUsernameQuery(r *http.Request) (username string, specError api.Error, err error) {
	query := r.URL.Query()
	if !query.Has("username") {
		return "", api.InvalidQueryParameter, errMissingUsernameQuery
	}
	username = query.Get("username")
	if err := service.ValidateUsername(username); err != nil {
		return "", api.InvalidUsername, fmt.Errorf("service.ValidateUsername: %w", err)
	}
	return username, "", nil
}

// parseRequiredAutocompleteQuery parses a required query query parameter.
//
// It can return the following errors:
//
// - [api.InvalidQueryParameter]
// - [api.InvalidUsername].
func (*Server) parseRequiredAutocompleteQuery(r *http.Request) (query string, specError api.Error, err error) {
	q := r.URL.Query()
	if !q.Has("query") {
		return "", api.InvalidQueryParameter, errMissingQueryParameter
	}
	query = q.Get("query")
	if err := service.ValidateUsername(query); err != nil {
		return "", api.InvalidUsername, fmt.Errorf("service.ValidateUsername: %w", err)
	}
	return query, "", nil
}

// parseSuperuserQuery parses an optional superuser filter query parameter.
//
// It can return the following errors:
//
// - [api.InvalidQueryParameter].
func (*Server) parseSuperuserQuery(r *http.Request) (superuser *bool, specError api.Error, err error) {
	query := r.URL.Query()
	if !query.Has("superuser") {
		return nil, "", nil
	}
	parsed, err := strconv.ParseBool(query.Get("superuser"))
	if err != nil {
		return nil, api.InvalidQueryParameter, fmt.Errorf("%w: %w", errSuperuserInvalid, err)
	}
	return &parsed, "", nil
}
