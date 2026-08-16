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

func newMemoryBackend(t *testing.T, l *slog.Logger) backend.Backend {
	t.Helper()
	return memory.NewMemoryBackend()
}

func TestMemoryBackend(t *testing.T) {
	t.Parallel()

	servertest.RunBackendTests(t, newMemoryBackend, nil)
}

func TestMemoryBackend_Flows(t *testing.T) {
	t.Parallel()

	servertest.RunFlowTests(t, newMemoryBackend, nil)
}
