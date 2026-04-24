package apitest

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func mustGET(t *testing.T, url string) *http.Response {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func mustPOST(t *testing.T, url, body string) *http.Response {
	t.Helper()
	return doBody(t, http.MethodPost, url, body)
}

func mustPATCH(t *testing.T, url, body string) *http.Response {
	t.Helper()
	return doBody(t, http.MethodPatch, url, body)
}

func doBody(t *testing.T, method, url, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", strconv.Itoa(len(body)))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d, want %d; body %s", resp.StatusCode, want, b)
	}
}

func decodeJSON[T any](t *testing.T, r io.Reader) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func mustMarshal(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

type errBody struct {
	Error string `json:"error"`
}

func assertErrorJSON(t *testing.T, resp *http.Response, wantMsg string) {
	t.Helper()
	eb := decodeJSON[errBody](t, resp.Body)
	if eb.Error != wantMsg {
		t.Fatalf("error %q, want %q", eb.Error, wantMsg)
	}
}
