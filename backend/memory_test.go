//spellchecker:words backend
package backend_test

//spellchecker:words testing github quickpid backend internal servertest
import (
	"testing"

	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/internal/servertest"
)

func TestInMemoryBackend(t *testing.T) {
	servertest.TestBackend(t, func(t *testing.T) backend.Backend {
		return backend.NewInMemoryBackend()
	})
}
