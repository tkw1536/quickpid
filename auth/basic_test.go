//spellchecker:words auth
package auth

//spellchecker:words http httptest testing
import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBasicAuthorization_OnUnauthorized(t *testing.T) {
	t.Parallel()

	var ba BasicAuthorization
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	ba.OnUnauthorized(nil, rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	want := `Basic realm="quickpid"`
	if got := rec.Header().Get("WWW-Authenticate"); got != want {
		t.Fatalf("WWW-Authenticate = %q, want %q", got, want)
	}
}
