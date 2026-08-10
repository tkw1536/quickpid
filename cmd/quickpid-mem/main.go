//spellchecker:words main
package main

//spellchecker:words slog github quickpid backend memory
import (
	"log/slog"

	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/backend/memory"
	"github.com/tkw1536/quickpid/cmd"
)

func main() {
	cmd.Main("quickpid-mem",
		nil,
		func(log *slog.Logger) (backend.Store, error) {
			log.Info("starting in-memory backend")
			log.Info("this backend is not suitable for production use, as any changes will be lost on process exit")
			return memory.NewStore(), nil
		},
	)
}
