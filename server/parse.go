//spellchecker:words server
package server

//spellchecker:words encoding json errors http strconv github quickpid pkglib errorsx
import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/tkw1536/quickpid/api"
	"go.tkw01536.de/pkglib/errorsx"
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
	errPasswordInvalid         = errors.New("invalid password query parameter")
)

// decodeJSON decodes the request body into v.
//
// It can return the following errors:
//
// - [api.BodySizeExceeded]
// - [api.BodyMissing]
// - [api.BodyInvalidJSON].
func (h *Server) decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	var body = r.Body
	if maxBody := h.svc.Options().Limits.MaxBodyBytes; maxBody > 0 {
		body = http.MaxBytesReader(w, body, maxBody)
	}
	var decodeErr error
	defer func() {
		if err := body.Close(); err != nil {
			// This uses [api.BodyInvalidJSON] which isn't entirely correct.
			//
			// This if most likely this happens when the underlying network connection had some error;
			// meaning the client will never see if anyways.
			decodeErr = api.WithErrorString(
				errorsx.Combine(decodeErr, fmt.Errorf("body.Close: %w", err)),
				api.BodyInvalidJSON,
			)
		}
	}()

	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return api.WithErrorString(fmt.Errorf("json.Decode: %w", err), api.BodySizeExceeded)
		}
		if errors.Is(err, io.EOF) {
			return api.WithErrorString(fmt.Errorf("json.Decode: %w", err), api.BodyMissing)
		}
		return api.WithErrorString(fmt.Errorf("json.Decode: %w", err), api.BodyInvalidJSON)
	}
	_, err := dec.Token()
	if !errors.Is(err, io.EOF) || err == nil {
		if err == nil {
			err = errTrailingJSON
		}
		return api.WithErrorString(err, api.BodyInvalidJSON)
	}
	return decodeErr
}

// parsePagination parses pagination parameters from the query string.
//
// A limit of 0 means no items are returned; the list handler still computes total.
//
// It can return the following errors:
//
// - [api.InvalidQueryParameter].
func (h *Server) parsePagination(r *http.Request) (limit int, offset int, err error) {
	query := r.URL.Query()
	limits := h.svc.Options().Limits

	limit = limits.DefaultPageLimit
	if query.Has("limit") {
		limit, err = strconv.Atoi(query.Get("limit"))
		if err != nil {
			return 0, 0, api.WithErrorString(fmt.Errorf("%w: %w", errLimitInvalid, err), api.InvalidQueryParameter)
		}
		if limit < 0 {
			return 0, 0, api.WithErrorString(errLimitMustBeNonNegative, api.InvalidQueryParameter)
		}
	}
	if limits.MaxPageLimit > 0 && limit > limits.MaxPageLimit {
		limit = limits.MaxPageLimit
	}

	offset = 0
	if query.Has("offset") {
		offset, err = strconv.Atoi(query.Get("offset"))
		if err != nil {
			return 0, 0, api.WithErrorString(fmt.Errorf("%w: %w", errOffsetInvalid, err), api.InvalidQueryParameter)
		}
		if offset < 0 {
			return 0, 0, api.WithErrorString(errOffsetMustBeNonNegative, api.InvalidQueryParameter)
		}
	}
	return limit, offset, nil
}

// getNamespace gets the namespace from the request path.
//
// It can return the following errors:
//
// - [api.InvalidNamespaceID].
func (*Server) getNamespace(r *http.Request) (api.ValidNamespaceID, error) {
	namespace, err := api.NewNamespaceID(r.PathValue("namespace"))
	if err != nil {
		return api.ValidNamespaceID{}, api.WithErrorString(fmt.Errorf("api.NewNamespaceID: %w", err), api.InvalidNamespaceID)
	}
	return namespace, nil
}

// getPID gets the pid from the request path.
//
// It can return the following errors:
//
// - [api.InvalidPID].
func (*Server) getPID(r *http.Request) (api.ValidPID, error) {
	pid, err := api.NewPID(r.PathValue("pid"))
	if err != nil {
		return api.ValidPID{}, api.WithErrorString(fmt.Errorf("api.NewPID: %w", err), api.InvalidPID)
	}
	return pid, nil
}

// getUsername gets the username from the request path.
func (*Server) getUsername(r *http.Request) (api.ValidUsername, error) {
	username, err := api.NewUsername(r.PathValue("username"))
	if err != nil {
		return api.ValidUsername{}, api.WithErrorString(fmt.Errorf("api.NewUsername: %w", err), api.InvalidUsername)
	}
	return username, nil
}

// getBaseURI gets the base URI from the request path.
//
// It can return the following errors:
//
// - [api.InvalidBaseURI].
func (*Server) getBaseURI(r *http.Request) (api.ValidBaseURI, error) {
	baseURI, err := api.NewBaseURI(r.PathValue("base_uri"))
	if err != nil {
		return api.ValidBaseURI{}, api.WithErrorString(fmt.Errorf("api.NewBaseURI: %w", err), api.InvalidBaseURI)
	}
	return baseURI, nil
}

// parseOptionalUsernameQuery parses an optional username query parameter.
//
// It can return the following errors:
//
// - [api.InvalidUsername].
func (*Server) parseOptionalUsernameQuery(r *http.Request) (*api.ValidUsername, error) {
	query := r.URL.Query()
	if !query.Has("username") {
		return nil, nil
	}
	username, err := api.NewUsername(query.Get("username"))
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("api.NewUsername: %w", err), api.InvalidUsername)
	}
	return &username, nil
}

// parseRequiredUsernameQuery parses a required username query parameter.
//
// It can return the following errors:
//
// - [api.InvalidQueryParameter]
// - [api.InvalidUsername].
func (*Server) parseRequiredUsernameQuery(r *http.Request) (api.ValidUsername, error) {
	query := r.URL.Query()
	if !query.Has("username") {
		return api.ValidUsername{}, api.WithErrorString(errMissingUsernameQuery, api.InvalidQueryParameter)
	}
	username, err := api.NewUsername(query.Get("username"))
	if err != nil {
		return api.ValidUsername{}, api.WithErrorString(fmt.Errorf("api.NewUsername: %w", err), api.InvalidUsername)
	}
	return username, nil
}

// parseRequiredAutocompleteQuery parses a required query query parameter.
//
// It can return the following errors:
//
// - [api.InvalidQueryParameter]
// - [api.InvalidUsername].
func (*Server) parseRequiredAutocompleteQuery(r *http.Request) (api.ValidUsername, error) {
	// HACK: The query itself isn't really a username
	// but we use it because it's guaranteed to be the same format.

	q := r.URL.Query()
	if !q.Has("query") {
		return api.ValidUsername{}, api.WithErrorString(errMissingQueryParameter, api.InvalidQueryParameter)
	}
	query, err := api.NewUsername(q.Get("query"))
	if err != nil {
		return api.ValidUsername{}, api.WithErrorString(fmt.Errorf("api.NewUsername: %w", err), api.InvalidUsername)
	}
	return query, nil
}

// parseSuperuserQuery parses an optional superuser filter query parameter.
//
// It can return the following errors:
//
// - [api.InvalidQueryParameter].
func (*Server) parseSuperuserQuery(r *http.Request) (superuser *bool, err error) {
	query := r.URL.Query()
	if !query.Has("superuser") {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(query.Get("superuser"))
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("%w: %w", errSuperuserInvalid, err), api.InvalidQueryParameter)
	}
	return &parsed, nil
}

// parsePasswordQuery parses an optional password filter query parameter.
//
// It can return the following errors:
//
// - [api.InvalidQueryParameter].
func (*Server) parsePasswordQuery(r *http.Request) (password *bool, err error) {
	query := r.URL.Query()
	if !query.Has("password") {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(query.Get("password"))
	if err != nil {
		return nil, api.WithErrorString(fmt.Errorf("%w: %w", errPasswordInvalid, err), api.InvalidQueryParameter)
	}
	return &parsed, nil
}
