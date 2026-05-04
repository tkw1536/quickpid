package cmd

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/tkw1536/quickpid"
	"github.com/tkw1536/quickpid/backend"
	"github.com/tkw1536/quickpid/server"
)

// Main runs and invokes a new main function using the given backend factory.
func Main(name string, backendFactory func() (backend.Backend, error)) {

	// parse command line flags
	var (
		listenHost string = "127.0.0.1"
		listenPort int    = 8080
		mountPath  string = "/api/v2"

		disableSwagger bool = false
		disableInfo    bool = false

		limits server.Limits = server.Limits{}.WithDefaults()

		legal bool = false
	)

	flag.StringVar(&listenHost, "host", listenHost, "host or IP to listen on")
	flag.IntVar(&listenPort, "port", listenPort, "port to listen on")
	flag.StringVar(&mountPath, "mount-path", mountPath, "mount path for the API")

	flag.BoolVar(&disableSwagger, "disable-swagger", disableSwagger, "disable swagger UI and spec file being served")
	flag.BoolVar(&disableInfo, "disable-info", disableInfo, "disable info endpoint")

	flag.Int64Var(&limits.MaxBodyBytes, "max-body-bytes", limits.MaxBodyBytes, "maximum size of request body")
	flag.IntVar(&limits.DefaultPageLimit, "default-page-limit", limits.DefaultPageLimit, "default number of items per page")
	flag.IntVar(&limits.MaxPageLimit, "max-page-limit", limits.MaxPageLimit, "maximum number of items per page")
	flag.IntVar(&limits.MaxBatchItems, "max-batch-items", limits.MaxBatchItems, "maximum number of items in a batch")
	flag.IntVar(&limits.MaxNamespaceIDAttempts, "max-namespace-id-attempts", limits.MaxNamespaceIDAttempts, "maximum number of attempts to allocate a namespace ID")
	flag.IntVar(&limits.MaxPIDAttempts, "max-pid-attempts", limits.MaxPIDAttempts, "maximum number of attempts to allocate a PID")

	flag.BoolVar(&legal, "legal", legal, "print license notices and exit")

	flag.Parse()

	if legal {
		fmt.Printf("%s is %s\n", name, quickpid.CopyrightNotice)
		fmt.Print(quickpid.License())
		fmt.Println()
		fmt.Println("================================================================================")
		fmt.Print(notices)
		return
	}

	mountPath = strings.TrimSuffix(mountPath, "/")

	// MAIN LOGIC
	backend, err := backendFactory()
	if err != nil {
		log.Fatal(err)
	}

	addr := net.JoinHostPort(listenHost, strconv.Itoa(listenPort))

	handler := server.NewHandler(
		server.Options{
			MountPath:        mountPath,
			DisableSwaggerUI: disableSwagger,
			Limits:           limits,
		},
		server.NewRuntime(),
		backend,
	)

	http.Handle(mountPath+"/", http.StripPrefix(mountPath, handler))

	log.Printf("listening on %s (in-memory API and Swagger UI at %s/)", addr, mountPath)
	if err := http.ListenAndServe(addr, http.DefaultServeMux); err != nil {
		log.Fatal(err)
	}

}

//go:generate go tool gogenlicense -m -n notices
