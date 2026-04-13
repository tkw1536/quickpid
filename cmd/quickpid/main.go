package main

import (
	"log"
	"net/http"
	"os"

	"github.com/glebarez/sqlite"
	"github.com/tkw1536/quickpid/resolvers/gormstore"
	"github.com/tkw1536/quickpid/server"
	"gorm.io/gorm"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	dsn := os.Getenv("QUICKPID_DSN")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		dsn = "quickpid.db?_pragma=foreign_keys(1)"
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	if err := gormstore.Migrate(db); err != nil {
		log.Fatal(err)
	}
	resolver := gormstore.NewResolver(db, server.DefaultPIDMaxAttempts)

	const mountPath = "/api/v2"
	apiHandler := server.NewHandler(server.Options{MountPath: mountPath}, resolver, server.SixCharPIDWithDashes)
	mux := http.NewServeMux()
	mux.Handle(mountPath+"/", http.StripPrefix(mountPath, apiHandler))
	mux.Handle("GET "+mountPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, mountPath+"/", http.StatusMovedPermanently)
	}))

	log.Printf("listening on %s (SQLite API and Swagger UI at %s/) dsn=%q", addr, mountPath, dsn)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
