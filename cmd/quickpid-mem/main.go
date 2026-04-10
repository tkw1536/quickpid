package main

import (
	"log"
	"net/http"
	"os"

	"github.com/tkw1536/quickpid/mem"
	"github.com/tkw1536/quickpid/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	const mountPath = "/api/v2"

	store := mem.NewStore()
	apiHandler := server.NewHandler(mountPath, store)
	mux := http.NewServeMux()
	mux.Handle(mountPath+"/", http.StripPrefix(mountPath, apiHandler))
	mux.Handle("GET "+mountPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, mountPath+"/", http.StatusMovedPermanently)
	}))

	log.Printf("listening on %s (in-memory API and Swagger UI at %s/)", addr, mountPath)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
