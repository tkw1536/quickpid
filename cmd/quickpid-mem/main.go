package main

import (
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/cmd"
)

func main() {
	cmd.Main(func() (backend.Backend, error) {
		return backend.NewInMemoryBackend(), nil
	})
}
