package httptester

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCase represents a pair of http request and response.
type TestCase struct {
	Request  Request  `json:"request"`
	Response Response `json:"response"`
}

// Run runs a test case against the given handler.
func (step TestCase) Run(t *testing.T, handler http.Handler) {
	t.Helper()

	method := strings.TrimSpace(step.Request.Method)
	if method == "" {
		t.Fatalf("missing request.method")
	}

	path := step.Request.Path

	var body io.Reader
	if step.Request.Body != "" {
		body = strings.NewReader(step.Request.Body)
	}
	req := httptest.NewRequest(method, path, body)

	for k, v := range step.Request.Headers {
		req.Header.Set(k, v)
	}
	if step.Request.Body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if resp.StatusCode != step.Response.Status {
		t.Fatalf("%s %s: status %d, want %d; body %s", method, path, resp.StatusCode, step.Response.Status, bytes.TrimSpace(respBody))
	}

	for k, want := range step.Response.Headers {
		got := resp.Header.Get(k)
		if got != want {
			t.Fatalf("%s %s: header %q = %q, want %q", method, path, k, got, want)
		}
	}

	if step.Response.Body.Text != "" && string(respBody) != step.Response.Body.Text {
		t.Fatalf("%s %s: body mismatch\ngot:  %q\nwant: %q", method, path, string(respBody), step.Response.Body.Text)
	}

	if step.Response.Body.JSON != nil {
		gotCanon, err := canonicalJSON(respBody)
		if err != nil {
			t.Fatalf("%s %s: response is not valid JSON: %v; body %s", method, path, err, bytes.TrimSpace(respBody))
		}
		wantCanon, err := canonicalJSONValue(step.Response.Body.JSON)
		if err != nil {
			t.Fatalf("%s %s: expected JSON cannot be marshaled: %v; want %#v", method, path, err, step.Response.Body.JSON)
		}
		if gotCanon != wantCanon {
			t.Fatalf("%s %s: JSON mismatch\n--- got\n%s\n--- want\n%s", method, path, gotCanon, wantCanon)
		}
	}
}

type Request struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type Response struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    Body              `json:"body"`
}

type Body struct {
	// JSON is semantically compared (whitespace and key ordering don't matter).
	JSON any `json:"json,omitempty"`
	// Text is compared byte-for-byte.
	Text string `json:"text,omitempty"`
}

func canonicalJSON(b []byte) (string, error) {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return "", err
	}
	out, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func canonicalJSONValue(v any) (string, error) {
	out, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return canonicalJSON(out)
}
