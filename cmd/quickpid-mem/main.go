//spellchecker:words main
package main

//spellchecker:words slog github bicpid backend authentication resolver
import (
	"log/slog"

	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/backend/authentication"
	"github.com/tkw1536/bicpid/backend/resolver"
	"github.com/tkw1536/bicpid/cmd"
)

func main() {
	cmd.Main("quickpid-mem",
		nil,
		func(*slog.Logger) (backend.ResolverBackend, error) {
			return resolver.NewInMemoryBackend(), nil
		},
		func(*slog.Logger) (backend.AuthBackend, error) {
			return authentication.NewInMemoryBackend(), nil
		},
	)
}
