//spellchecker:words resolver
package resolver_test

//spellchecker:words testing github bicpid backend resolver internal servertest
import (
	"testing"

	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/backend/resolver"
	"github.com/tkw1536/bicpid/internal/servertest"
)

func TestInMemoryBackend(t *testing.T) {
	servertest.TestBackend(t, func(t *testing.T) backend.ResolverBackend {
		return resolver.NewInMemoryBackend()
	})
}
