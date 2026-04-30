package steptest

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func clientFor(fn func(*http.Request) (*http.Response, error)) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(fn),
	}
}

func resp(status int, headers map[string]string, body string) *http.Response {
	h := http.Header{}
	for k, v := range headers {
		h.Set(k, v)
	}
	return &http.Response{
		StatusCode: status,
		Header:     h,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestRunner_TemplateAndHeadersAndJSON(t *testing.T) {
	calls := 0
	c := clientFor(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != http.MethodGet {
			t.Fatalf("method: got %q want %q", r.Method, http.MethodGet)
		}
		if r.URL.String() != "http://example.test/api/items/abc" {
			t.Fatalf("url: got %q", r.URL.String())
		}
		return resp(200, map[string]string{"X-Server": "ok"}, `{"b":2,"a":1}`), nil
	})

	r := New("http://example.test/api", c)
	r.Run(t, []Step{
		{
			Name: "get_item",
			Request: Request{
				Method: http.MethodGet,
				Path:   "/items/abc",
			},
			Response: Response{
				Status:  200,
				Headers: map[string]string{"X-Server": "ok"},
				Body: Body{
					JSON: map[string]any{"a": 1, "b": 2},
				},
			},
		},
	})
	if calls != 1 {
		t.Fatalf("calls=%d, want 1", calls)
	}
}

func TestCanonicalJSON_Equivalent(t *testing.T) {
	got, err := canonicalJSON([]byte("{\"b\":2,\"a\":1}"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := canonicalJSON([]byte("{\"a\":1,\"b\":2}"))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}
