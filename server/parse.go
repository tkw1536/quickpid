//spellchecker:words server
package server

//spellchecker:words errors http strconv github quickpid internal strict pkglib errorsx
import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/tkw1536/quickpid/api"
	"github.com/tkw1536/quickpid/internal/strict"
	"go.tkw01536.de/pkglib/errorsx"
)

var (
	errLimitInvalid            = errors.New("invalid limit")
	errLimitMustBeNonNegative  = errors.New("limit must be non-negative")
	errOffsetInvalid           = errors.New("invalid offset")
	errOffsetMustBeNonNegative = errors.New("offset must be non-negative")
	errDeletedInvalid          = errors.New("invalid deleted query parameter")
	errMissingUsernameQuery    = errors.New("missing username query parameter")
	errMissingQueryParameter   = errors.New("missing query query parameter")
	errInvalidBooleanQuery     = errors.New("invalid boolean query parameter")
)

// decodeJSON decodes the request body into v.
//
// It can return the following errors:
//
// - [api.BodySizeExceeded]
// - [api.BodyMissing]
// - [api.BodyInvalidJSON].
func (h *Server) decodeJSON(w http.ResponseWriter, r *http.Request, v any) (decodeErr error) {
	var body = r.Body
	if maxBody := h.svc.Options().Limits.MaxBodyBytes; maxBody > 0 {
		body = http.MaxBytesReader(w, body, maxBody)
	}
	defer func() {
		if err := body.Close(); err != nil {
			// This uses [api.BodyInvalidJSON] which isn't entirely correct.
			//
			// This if most likely this happens when the underlying network connection had some error;
			// meaning the client will never see if anyways.
			decodeErr = api.WithErrorCode(
				errorsx.Combine(decodeErr, fmt.Errorf("failed to close body: %w", err)),
				api.BodyInvalidJSON,
			)
		}
	}()

	err := strict.UnmarshalStrictTo(body, v)
	if errors.Is(err, io.EOF) {
		return api.WithErrorCode(fmt.Errorf("body is missing or incomplete: %w", err), api.BodyMissing)
	}
	if _, isMaxBytesError := errors.AsType[*http.MaxBytesError](err); isMaxBytesError {
		return api.WithErrorCode(fmt.Errorf("maximum body size exceeded: %w", err), api.BodySizeExceeded)
	}
	if errors.Is(err, strict.ErrJsonNull) || errors.Is(err, strict.ErrFailedToDecode) || errors.Is(err, strict.ErrTrailingData) {
		return api.WithErrorCode(err, api.BodyInvalidJSON)
	}
	if err != nil {
		return fmt.Errorf("unknown error while decoding json: %w", err)
	}
	return nil
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
			return 0, 0, api.WithErrorCode(fmt.Errorf("%w: %w", errLimitInvalid, err), api.InvalidQueryParameter)
		}
		if limit < 0 {
			return 0, 0, api.WithErrorCode(errLimitMustBeNonNegative, api.InvalidQueryParameter)
		}
	}
	if limits.MaxPageLimit > 0 && limit > limits.MaxPageLimit {
		limit = limits.MaxPageLimit
	}

	offset = 0
	if query.Has("offset") {
		offset, err = strconv.Atoi(query.Get("offset"))
		if err != nil {
			return 0, 0, api.WithErrorCode(fmt.Errorf("%w: %w", errOffsetInvalid, err), api.InvalidQueryParameter)
		}
		if offset < 0 {
			return 0, 0, api.WithErrorCode(errOffsetMustBeNonNegative, api.InvalidQueryParameter)
		}
	}
	return limit, offset, nil
}

// readPidParam gets the pid from the request path.
//
// It can return the following errors:
//
// - [api.InvalidPID].
func (*Server) readPidParam(r *http.Request) (api.ValidPID, error) {
	pid, err := api.NewPID(r.PathValue("pid"))
	if err != nil {
		return api.ValidPID{}, api.WithErrorCode(fmt.Errorf("failed to parse pid: %w", err), api.InvalidPID)
	}
	return pid, nil
}

// readUsernameParam gets the username from the request path.
//
// It can return the following errors:
//
// - [api.InvalidUsername].
func (*Server) readUsernameParam(r *http.Request) (api.ValidUsername, error) {
	username, err := api.NewUsername(r.PathValue("username"))
	if err != nil {
		return api.ValidUsername{}, api.WithErrorCode(fmt.Errorf("failed to parse username: %w", err), api.InvalidUsername)
	}
	return username, nil
}

// readBaseURIParam gets the base URI from the request path.
//
// It can return the following errors:
//
// - [api.InvalidBaseURI].
func (*Server) readBaseURIParam(r *http.Request) (api.ValidBaseURI, error) {
	baseURI, err := api.NewBaseURI(r.PathValue("baseUri"))
	if err != nil {
		return api.ValidBaseURI{}, api.WithErrorCode(fmt.Errorf("failed to parse base uri: %w", err), api.InvalidBaseURI)
	}
	return baseURI, nil
}

// readRequiredUsernameFromQuery parses a required username query parameter.
//
// It can return the following errors:
//
// - [api.InvalidQueryParameter]
// - [api.InvalidUsername].
func (s *Server) readRequiredUsernameFromQuery(r *http.Request) (api.ValidUsername, error) {
	query := r.URL.Query()
	if !query.Has("username") {
		return api.ValidUsername{}, api.WithErrorCode(errMissingUsernameQuery, api.InvalidQueryParameter)
	}
	username, err := api.NewUsername(r.URL.Query().Get("username"))
	if err != nil {
		return api.ValidUsername{}, api.WithErrorCode(fmt.Errorf("failed to parse username: %w", err), api.InvalidUsername)
	}
	return username, nil
}

// readRequiredAutocompleteFromQuery parses a required query query parameter.
//
// It can return the following errors:
//
// - [api.InvalidQueryParameter]
// - [api.InvalidAutocompleteQuery].
func (*Server) readRequiredAutocompleteFromQuery(r *http.Request) (api.ValidAutocompleteQuery, error) {
	// HACK: The query itself isn't really a username
	// but we use it because it's guaranteed to be the same format.

	q := r.URL.Query()
	if !q.Has("query") {
		return api.ValidAutocompleteQuery{}, api.WithErrorCode(errMissingQueryParameter, api.InvalidQueryParameter)
	}
	query, err := api.NewAutocompleteQuery(q.Get("query"))
	if err != nil {
		return api.ValidAutocompleteQuery{}, api.WithErrorCode(fmt.Errorf("failed to parse autocomplete query: %w", err), api.InvalidAutocompleteQuery)
	}
	return query, nil
}

// readOptionalBooleanFromQuery parses an optional boolean query parameter.
//
// It can return the following errors:
//
// - [api.InvalidQueryParameter].
func (*Server) readOptionalBooleanFromQuery(r *http.Request, name string) (*bool, error) {
	query := r.URL.Query()
	if !query.Has(name) {
		return nil, nil
	}

	value := query.Get(name)
	switch value {
	case "true":
		return new(true), nil
	case "false":
		return new(false), nil
	default:
		return nil, api.WithErrorCode(fmt.Errorf("%w: value %q for %q is not a boolean", errInvalidBooleanQuery, value, name), api.InvalidQueryParameter)
	}
}
