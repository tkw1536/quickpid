//spellchecker:words main
package main

//spellchecker:words slog github quickpid backend
import (
	"log/slog"

	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/backend/resolver"
	"github.com/tkw1536/bicpid/cmd"
)

func main() {
	cmd.Main("quickpid-mem", func(*slog.Logger) (backend.ResolverBackend, error) {
		return resolver.NewInMemoryBackend(), nil
	})
}
