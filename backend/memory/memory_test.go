//spellchecker:words memory
package memory_test

//spellchecker:words slog testing github quickpid backend memory internal pidtest servertest
import (
	"log/slog"
	"testing"

	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/backend/memory"
	servertest "github.com/tkw1536/quickpid/internal/pidtest"
)

func newStore(t *testing.T, l *slog.Logger) backend.Store {
	t.Helper()
	return memory.NewStore()
}

func TestStore(t *testing.T) {
	t.Parallel()

	servertest.RunStoreTests(t, newStore, nil)
}

func TestStore_Flows(t *testing.T) {
	t.Parallel()

	servertest.RunFlowTests(t, newStore, nil)
}
