package httptester

import (
	"io"
	"net/http"
	"testing"
)

func TestCaseRun_StatusHeadersTextOK(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/hello" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/hello")
		}
		if got := r.Header.Get("X-Req"); got != "1" {
			t.Fatalf("X-Req = %q, want %q", got, "1")
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want %q", got, "application/json")
		}

		w.Header().Set("X-Resp", "ok")
		w.WriteHeader(201)
		_, _ = w.Write([]byte("hi"))
	})

	step := TestCase{
		Request: Request{
			Method:  "POST",
			Path:    "/hello",
			Headers: map[string]string{"X-Req": "1"},
			Body:    `{"a":1}`,
		},
		Response: Response{
			Status:  201,
			Headers: map[string]string{"X-Resp": "ok"},
			Body:    Body{Text: "hi"},
		},
	}

	step.Run(t, h)
}

func TestCaseRun_JSONCanonicalizationOK(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\n  \"b\": 2,\n  \"a\": 1\n}\n"))
	})

	step := TestCase{
		Request: Request{Method: "GET", Path: "/"},
		Response: Response{
			Status: 200,
			Body: Body{
				JSON: map[string]any{"a": float64(1), "b": float64(2)},
			},
		},
	}

	step.Run(t, h)
}

func TestCaseRun_MethodIsTrimmed(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q, want %q", r.Method, http.MethodGet)
		}
		w.WriteHeader(200)
	})

	TestCase{
		Request:  Request{Method: "  GET  ", Path: "/"},
		Response: Response{Status: 200},
	}.Run(t, h)
}

func TestCaseRun_DoesNotSetContentTypeWhenNoBody(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "" {
			t.Fatalf("Content-Type = %q, want empty", got)
		}
		w.WriteHeader(204)
	})

	TestCase{
		Request:  Request{Method: "GET", Path: "/"},
		Response: Response{Status: 204},
	}.Run(t, h)
}

func TestCaseRun_DoesNotOverrideExplicitContentType(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "text/plain" {
			t.Fatalf("Content-Type = %q, want %q", got, "text/plain")
		}
		w.WriteHeader(200)
	})

	TestCase{
		Request: Request{
			Method:  "POST",
			Path:    "/",
			Headers: map[string]string{"Content-Type": "text/plain"},
			Body:    "hello",
		},
		Response: Response{Status: 200},
	}.Run(t, h)
}

func TestCaseRun_RequestBodyIsPassedThrough(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if string(b) != "abc" {
			t.Fatalf("body = %q, want %q", string(b), "abc")
		}
		w.WriteHeader(200)
	})

	TestCase{
		Request:  Request{Method: "POST", Path: "/", Body: "abc"},
		Response: Response{Status: 200},
	}.Run(t, h)
}

func TestCaseRun_JSONCanonicalizationNestedOK(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\n  \"x\": {\"b\":2,\"a\":1},\n  \"arr\": [3,2,1]\n}\n"))
	})

	TestCase{
		Request: Request{Method: "GET", Path: "/"},
		Response: Response{
			Status: 200,
			Body: Body{
				JSON: map[string]any{
					"arr": []any{float64(3), float64(2), float64(1)},
					"x":   map[string]any{"a": float64(1), "b": float64(2)},
				},
			},
		},
	}.Run(t, h)
}
