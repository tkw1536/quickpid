package steptest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// Doer is the minimal interface implemented by http.Client.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type Step struct {
	Name     string   `json:"name"`
	Request  Request  `json:"request"`
	Response Response `json:"response"`
}

type Request struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

type Response struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    Body              `json:"body,omitempty"`
}

type Body struct {
	// JSON is semantically compared (whitespace and key ordering don't matter).
	JSON any `json:"json,omitempty"`
	// Text is compared byte-for-byte.
	Text string `json:"text,omitempty"`
}

// Runner executes a list of HTTP steps against a base URL.
type Runner struct {
	BaseURL string
	Client  Doer
}

func New(baseURL string, client Doer) *Runner {
	if client == nil {
		client = http.DefaultClient
	}
	return &Runner{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Client:  client,
	}
}

func (r *Runner) Run(t *testing.T, steps []Step) {
	t.Helper()
	for i, step := range steps {
		name := step.Name
		if name == "" {
			name = fmt.Sprintf("step_%d", i)
		}
		t.Run(name, func(t *testing.T) {
			t.Helper()
			r.runStep(t, step)
		})
	}
}

func (r *Runner) runStep(t *testing.T, step Step) {
	t.Helper()

	method := strings.TrimSpace(step.Request.Method)
	if method == "" {
		t.Fatalf("missing request.method")
	}

	path := step.Request.Path
	url := r.BaseURL + path

	var body io.Reader
	if step.Request.Body != "" {
		body = strings.NewReader(step.Request.Body)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	for k, v := range step.Request.Headers {
		req.Header.Set(k, v)
	}
	if step.Request.Body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := r.Client.Do(req)
	if err != nil {
		t.Fatalf("do request %s %s: %v", method, path, err)
	}
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
