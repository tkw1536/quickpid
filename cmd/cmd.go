package cmd

//spellchecker:words context encoding json flag slog http signal strconv strings syscall time github quickpid auth backend server
import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	quickpid "github.com/tkw1536/bicpid"
	"github.com/tkw1536/bicpid/auth"
	"github.com/tkw1536/bicpid/backend"
	"github.com/tkw1536/bicpid/server"
)

type mainCmd struct {
	name           string
	backendFactory func(*slog.Logger) (backend.Backend, error)
	basicAuthFile  string

	listenHost string
	listenPort int
	mountPath  string

	disableSwagger bool
	disableInfo    bool
	limits         server.Limits

	logLevel string
	logJSON  bool

	legal bool

	shutdownTimeout time.Duration

	addr   string
	logger *slog.Logger
}

//go:generate go tool gogenlicense -m -n notices

// Main is the main entry point using the given backend factory.
//
// Backends are initialized once using the backendFactory function.
//
// To add additional flags, callers should add to [flag.CommandLine] prior to the call to Main
// and access variables in the factory function.
func Main(name string, backendFactory func(logger *slog.Logger) (backend.Backend, error)) {
	os.Exit(
		new(mainCmd{
			name:           name,
			backendFactory: backendFactory,

			listenHost: "127.0.0.1",
			listenPort: 8080,
			mountPath:  "/api/v2",

			disableSwagger: false,
			disableInfo:    false,
			limits:         server.DefaultLimits(),

			logLevel: "info",
			logJSON:  false,

			legal: false,

			shutdownTimeout: time.Minute,
		}).run(),
	)
}

func (main *mainCmd) run() int {
	if err := main.parseFlags(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	main.parseOptions()

	if main.legal {
		main.printLegalNotices()
		return 0
	}

	if err := main.setupLogger(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	main.printStartupBanner()

	b, err := main.backendFactory(main.logger)
	if err != nil {
		main.logger.Error("backend initialization failed", slog.Any("error", err))
		return 1
	}

	defer func() {
		main.logger.Info("shutting down backend")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), main.shutdownTimeout)
		defer cancel()
		if err := b.Shutdown(shutdownCtx); err != nil {
			main.logger.Error("backend shutdown failed", slog.Any("error", err))
			return
		}
		main.logger.Info("backend shutdown complete")
	}()

	h := main.newServerHandler(b)
	return main.serve(h)
}

func (main *mainCmd) printStartupBanner() {
	fmt.Printf("%s — %s\n", main.name, quickpid.CopyrightNotice)
	fmt.Println("Use -legal to view licensing information and notices.")
}

func (main *mainCmd) parseFlags() error {
	flag.StringVar(&main.listenHost, "host", main.listenHost, "host or IP to listen on")
	flag.IntVar(&main.listenPort, "port", main.listenPort, "port to listen on")
	flag.StringVar(&main.mountPath, "mount-path", main.mountPath, "mount path for the API")

	flag.BoolVar(&main.disableSwagger, "disable-swagger", main.disableSwagger, "disable swagger UI and spec file being served")
	flag.BoolVar(&main.disableInfo, "disable-info", main.disableInfo, "disable info endpoint")

	flag.StringVar(&main.basicAuthFile, "basic-auth-file", main.basicAuthFile, "path to HTTP Basic Authentication file")

	flag.Int64Var(&main.limits.MaxBodyBytes, "max-body-bytes", main.limits.MaxBodyBytes, "maximum size of request body")
	flag.IntVar(&main.limits.DefaultPageLimit, "default-page-limit", main.limits.DefaultPageLimit, "default number of items per page")
	flag.IntVar(&main.limits.MaxPageLimit, "max-page-limit", main.limits.MaxPageLimit, "maximum number of items per page")
	flag.IntVar(&main.limits.MaxBatchItems, "max-batch-items", main.limits.MaxBatchItems, "maximum number of items in a batch")
	flag.IntVar(&main.limits.MaxNamespaceIDAttempts, "max-namespace-id-attempts", main.limits.MaxNamespaceIDAttempts, "maximum number of attempts to allocate a namespace ID")
	flag.IntVar(&main.limits.MaxPIDAttempts, "max-pid-attempts", main.limits.MaxPIDAttempts, "maximum number of attempts to allocate a PID")

	flag.StringVar(&main.logLevel, "log-level", main.logLevel, "log level: none, error, warn, info, debug")
	flag.BoolVar(&main.logJSON, "log-json", main.logJSON, "output logs as json")

	flag.BoolVar(&main.legal, "legal", main.legal, "print license notices and exit")

	flag.DurationVar(&main.shutdownTimeout, "shutdown-timeout", main.shutdownTimeout, "timeout applied to backend and HTTP server shutdowns")

	flag.Parse()

	return nil
}

func (main *mainCmd) parseOptions() {
	main.mountPath = strings.TrimSuffix(main.mountPath, "/")
	main.addr = net.JoinHostPort(main.listenHost, strconv.Itoa(main.listenPort))
}

func (main *mainCmd) printLegalNotices() {
	fmt.Printf("%s is %s\n", main.name, quickpid.CopyrightNotice)
	fmt.Print(quickpid.License())
	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Print(notices)
}

func (main *mainCmd) setupLogger() error {
	var level slog.Level
	switch strings.ToLower(main.logLevel) {
	case "error":
		level = slog.LevelError
	case "warn", "warning":
		level = slog.LevelWarn
	case "info":
		level = slog.LevelInfo
	case "debug":
		level = slog.LevelDebug
	case "none":
		main.logger = slog.New(slog.DiscardHandler)
		slog.SetDefault(main.logger)
		return nil
	default:
		return fmt.Errorf("invalid -log-level %q (expected none|error|warn|info|debug)", main.logLevel)
	}

	var logHandler slog.Handler
	if main.logJSON {
		logHandler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	} else {
		logHandler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	}
	main.logger = slog.New(logHandler)
	slog.SetDefault(main.logger)
	return nil
}

func (main *mainCmd) newServerHandler(b backend.Backend) *server.Handler {
	return server.NewHandler(
		server.Options{
			MountPath:               main.mountPath,
			DisableSwaggerUI:        main.disableSwagger,
			RegisterBasicAuthInSpec: main.basicAuthFile != "",
			Limits:                  main.limits,
			InfoEnabled:             !main.disableInfo,
		},
		server.NewRuntime(),
		b,
		main.logger,
	)
}

func (main *mainCmd) readAuthFile() (auth.Authorization, error) {
	if main.basicAuthFile == "" {
		main.logger.Info("authorization disabled")
		return auth.NoAuthentication{}, nil
	}

	data, err := os.ReadFile(main.basicAuthFile)
	if err != nil {
		return nil, fmt.Errorf("read auth file %q: %w", main.basicAuthFile, err)
	}

	var authorization auth.BasicAuthorization
	if err := json.Unmarshal(data, &authorization); err != nil {
		return nil, fmt.Errorf("parse auth file %q: %w", main.basicAuthFile, err)
	}

	main.logger.Info(
		"loaded authorization",
		slog.String("auth_file", main.basicAuthFile),
		slog.Int("users", len(authorization.Authentication.Users)),
		slog.Int("superusers", len(authorization.Superusers)),
		slog.Int("namespaces", len(authorization.Permissions)),
	)

	return &authorization, nil
}

func (main *mainCmd) serve(h http.Handler) int {
	authorization, err := main.readAuthFile()
	if err != nil {
		main.logger.Error("failed to read auth file", slog.Any("error", err))
		return 1
	}

	proxy := auth.Proxy{
		Forward: func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		},
		Logger: main.logger,
		Auth:   authorization,
	}
	http.Handle(main.mountPath+"/", http.StripPrefix(main.mountPath, &proxy))

	srv := &http.Server{
		Addr:    main.addr,
		Handler: http.DefaultServeMux,
	}

	main.logger.Info(
		"listening",
		slog.String("addr", main.addr),
		slog.String("mount_path", main.mountPath),
	)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		main.logger.Info("starting server shutdown")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), main.shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			main.logger.Error("http server shutdown failed", slog.Any("error", err))
			return 1
		}
		main.logger.Info("http server shutdown complete")
		return 0
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			main.logger.Error("http server exited", slog.Any("error", err))
			return 1
		}
		return 0
	}
}
