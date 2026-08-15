//spellchecker:words memory
package memory_test

//spellchecker:words testing github quickpid backend memory internal pidtest servertest
import (
	"testing"

	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/backend/memory"
	servertest "github.com/tkw1536/quickpid/internal/pidtest"
)

func newStore() backend.Store {
	return memory.NewStore()
}

func TestStore(t *testing.T) {
	t.Parallel()

	servertest.RunStoreTests(t, func(t *testing.T) backend.Store {
		t.Helper()
		return newStore()
	})
}

func TestStore_Flows(t *testing.T) {
	t.Parallel()

	servertest.RunFlowTests(t, func(t *testing.T) backend.Store {
		t.Helper()
		return memory.NewStore()
	})
}
