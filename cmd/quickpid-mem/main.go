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
		func(*slog.Logger) (backend.Store, error) {
			return memory.NewStore(), nil
		},
	)
}
