//spellchecker:words main
package main

//spellchecker:words slog github bicpid backend memory
import (
	"log/slog"

	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/backend/memory"
	"github.com/tkw1536/bicpid/cmd"
)

func main() {
	cmd.Main("quickpid-mem",
		nil,
		func(*slog.Logger) (backend.Store, error) {
			return memory.NewStore(), nil
		},
	)
}
