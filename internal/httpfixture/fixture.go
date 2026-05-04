// Package httpfixture provides tools for testing HTTP handlers.
package httpfixture

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
)

// Fixture represents a HTTP test case.
type Fixture struct {
	Request  Request  `json:"request"`
	Response Response `json:"response,omitzero"`
}

// TestingT is implemented by [testing.T].
//
// The interface enables testing of the test code.
type TestingT interface {
	Helper()
	Context() context.Context
	Errorf(format string, args ...any)
}

// Run runs the test case against the given handler.
// If something goes wrong, [t.Errorf] is called.
func (fix Fixture) Run(t TestingT, handler http.Handler) {
	t.Helper()

	req, err := fix.Request.ToRequest(t.Context())
	if err != nil {
		t.Errorf("failed to create request: %v", err)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if err := fix.Response.Compare(recorder); err != nil {
		t.Errorf("response did not match expected response: %v", err)
	}
}

type Request struct {
	// The request method.
	Method string `json:"method"`

	// The request path, starting with a "/", but relative to the API resolver root.
	Path string `json:"path"`

	// Set of request headers to set.
	Headers [][2]string `json:"headers,omitzero"`

	// The request body content.
	// Note: We cannot use json.RawMessage here, because we want to be able to represent invalid and valid JSON alike.
	Body string `json:"body,omitzero"`
}

// ToRequest converts this request to a [http.Request].
//
// If something goes wrong encoding the body and onError is not nil, it will call onError with that error.
func (req Request) ToRequest(ctx context.Context) (*http.Request, error) {
	var bodyReader io.Reader = http.NoBody
	if req.Body != "" {
		bodyReader = strings.NewReader(req.Body)
	}
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	for _, header := range req.Headers {
		httpReq.Header.Add(header[0], header[1])
	}
	return httpReq, nil
}

// Response represents a set of assertions about the response.
type Response struct {
	// Code is the expected status code of the response.
	Code int `json:"code"`

	// Headers represents a list of expected key-value pairs in the response headers.
	// Additional headers are permitted, and not checked.
	Headers [][2]string `json:"headers,omitzero"`

	// Body is the expected json serialization of the response body.
	// It is compared with json semantics, meaning that map key order does not matter.
	Body json.RawMessage `json:"body,omitzero"`
}

// Compare compares the actual response against this expected response.
//
// It returns nil if the actual response matches, and a non-nil error if it does not.
// The returned error may wrap multiple underlying errors.
func (resp Response) Compare(actual *httptest.ResponseRecorder) error {
	var errs []error

	if actual.Code != resp.Code {
		errs = append(errs, fmt.Errorf("got code = %d, want %d", actual.Code, resp.Code))
	}

	actualHeaders := actual.Header()
	for _, want := range resp.Headers {
		key, value := want[0], want[1]
		key = http.CanonicalHeaderKey(key)

		values := actualHeaders.Values(key)
		if slices.Contains(values, value) {
			continue
		}

		quotedValues := slices.Clone(values)
		for i, v := range values {
			quotedValues[i] = strconv.Quote(v)
		}

		errs = append(
			errs,
			fmt.Errorf(
				"wanted header %q to contain %q, but got only %s.",
				key,
				value,
				strings.Join(quotedValues, ", "),
			),
		)
	}

	if resp.Body != nil {
		wantCanon, err := canonicalJSON(bytes.NewReader(resp.Body))
		if err != nil {
			errs = append(errs, fmt.Errorf("expected body is not valid JSON: %w", err))
			goto join_and_return
		}
		gotCanon, err := canonicalJSON(actual.Body)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to read body as JSON: %w", err))
			goto join_and_return
		}
		if gotCanon != wantCanon {
			errs = append(errs, fmt.Errorf("body response mismatch\n--- got\n%s\n--- want\n%s", gotCanon, wantCanon))
			goto join_and_return
		}
	}

join_and_return:
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return errors.Join(errs...)
	}
}

// canonicalJSON reads a JSON value from r, and returns its canonical JSON serialization.
func canonicalJSON(r io.Reader) (string, error) {
	var v any
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return "", fmt.Errorf("failed to decode JSON: %w", err)
	}

	out, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(out), nil
}

// canonicalJSONValue returns the canonical JSON serialization of v.
func canonicalJSONValue(v any) (string, error) {
	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		return "", fmt.Errorf("failed to encode as JSON: %w", err)
	}
	return canonicalJSON(&buf)
}
