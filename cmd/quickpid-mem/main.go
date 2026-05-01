package main

import (
	"log"
	"net/http"
	"os"

	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	const mountPath = "/api/v2"

	memory := backend.NewInMemoryBackend()

	apiHandler := server.NewHandler(server.Options{MountPath: mountPath}, server.NewRuntime(), memory)
	mux := http.NewServeMux()
	mux.Handle(mountPath+"/", http.StripPrefix(mountPath, apiHandler))

	log.Printf("listening on %s (in-memory API and Swagger UI at %s/)", addr, mountPath)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
