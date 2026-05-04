package httpfixture_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tkw1536/quickpid/internal/httpfixture"
)

func TestRequestToRequest_EncodesBodyAsJSON(t *testing.T) {
	req := httpfixture.Request{
		Method: "POST",
		Path:   "/hello",
		Body:   `{"a":1}`,
	}

	httpReq, err := req.ToRequest(context.Background())
	if err != nil {
		t.Fatalf("ToRequest: %v", err)
	}
	if httpReq.Method != "POST" {
		t.Fatalf("Method = %q, want %q", httpReq.Method, "POST")
	}
	if httpReq.URL.Path != "/hello" {
		t.Fatalf("Path = %q, want %q", httpReq.URL.Path, "/hello")
	}

	b, err := io.ReadAll(httpReq.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if strings.TrimSpace(string(b)) != `{"a":1}` {
		t.Fatalf("body = %q, want %q", strings.TrimSpace(string(b)), `{"a":1}`)
	}
}

func TestResponseCompare_CodeHeadersAndBodyMatchOK(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Add("X-Resp", "ok")
	rec.Header().Add("X-Resp", "also-ok")
	rec.WriteHeader(201)
	_, _ = rec.Write([]byte("{\n  \"b\": 2,\n  \"a\": 1\n}\n"))

	want := httpfixture.Response{
		Code:    201,
		Headers: [][2]string{{"x-resp", "ok"}},
		Body:    json.RawMessage(`{"a":1,"b":2}`),
	}

	if err := want.Compare(rec); err != nil {
		t.Fatalf("Compare: %v", err)
	}
}

func TestResponseCompare_JoinsMultipleErrors(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("X-Resp", "nope")
	rec.WriteHeader(200)
	_, _ = rec.Write([]byte(`{"a":2}`))

	want := httpfixture.Response{
		Code:    201,
		Headers: [][2]string{{"X-Resp", "ok"}},
		Body:    json.RawMessage(`{"a":1}`),
	}

	err := want.Compare(rec)
	if err == nil {
		t.Fatalf("Compare err = nil, want non-nil")
	}
	var joined interface{ Unwrap() []error }
	if !errors.As(err, &joined) || len(joined.Unwrap()) < 2 {
		t.Fatalf("Compare err = %T %v, want a multi-error (errors.Join)", err, err)
	}
}

func TestFixtureRun_ReportsMismatchViaTErrors(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":false}`))
	})

	tb := &mockT{ctx: t.Context()}
	httpfixture.Fixture{
		Request:  httpfixture.Request{Method: "GET", Path: "/"},
		Response: httpfixture.Response{Code: 200, Body: json.RawMessage(`{"ok":true}`)},
	}.Run(tb, h)

	if len(tb.errs) == 0 {
		t.Fatalf("no errors recorded; wanted Fixture.Run to report mismatch")
	}
}

type mockT struct {
	ctx  context.Context
	errs []string
}

func (m *mockT) Helper() {}
func (m *mockT) Context() context.Context {
	if m.ctx == nil {
		return context.Background()
	}
	return m.ctx
}
func (m *mockT) Errorf(format string, args ...any) {
	m.errs = append(m.errs, format)
}

func ExampleRequest_ToRequest() {
	req := httpfixture.Request{
		Method: "POST",
		Path:   "/hello",
		Body:   `{"a":1}`,
	}

	httpReq, err := req.ToRequest(context.Background())
	if err != nil {
		fmt.Println("err:", err)
		return
	}

	body, _ := io.ReadAll(httpReq.Body)
	fmt.Println(httpReq.Method, httpReq.URL.Path, strings.TrimSpace(string(body)))
	// Output: POST /hello {"a":1}
}

func ExampleResponse_Compare() {
	rec := httptest.NewRecorder()
	rec.Header().Set("X-Resp", "ok")
	rec.WriteHeader(201)
	_, _ = rec.Write([]byte(`{"b":2,"a":1}`))

	want := httpfixture.Response{
		Code:    201,
		Headers: [][2]string{{"X-Resp", "ok"}},
		Body:    json.RawMessage(`{"a":1,"b":2}`),
	}

	fmt.Println(want.Compare(rec))
	// Output: <nil>
}
