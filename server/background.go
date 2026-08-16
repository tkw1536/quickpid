//spellchecker:words server
package server

//spellchecker:words context slog time
import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// StartBackground starts server background processes in a separate goroutine.
// If background tasks are already started, this does nothing.
// Once the background processes have been started, they cannot be stopped again.
//
// The return function should be called to stop any background processes.
// The returned function returns when the server is closed.
func (server *Server) StartBackground() func(ctx context.Context) error {
	if !server.createdViaNew {
		panic("not initialized via New")
	}

	server.backgroundStart.Do(server.startBackground)
	return server.stopBackground
}

func (server *Server) startBackground() {
	var ctx context.Context
	server.backgroundDone = make(chan struct{})
	ctx, server.cancelBackground = context.WithCancel(context.Background())
	go func() {
		defer close(server.backgroundDone)
		server.doBackground(ctx)
	}()
}

// stopBackground stops the background process, waits for it to complete, and then exists.
// If ctx closes, stopBackground returns early.
func (server *Server) stopBackground(ctx context.Context) error {
	server.cancelBackground()
	select {
	case <-server.backgroundDone:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	}
}

// doBackground implements background processes.
func (server *Server) doBackground(ctx context.Context) {
	// grab the interval for the background
	// if it is zero, don't do anything.
	interval := server.ops.BackgroundInterval
	if interval <= 0 {
		return
	}

	// invoke initially
	server.doCron(ctx)

	ticker := time.NewTicker(interval)
	for {
		ticker.Reset(interval)
		select {
		case <-ticker.C:
			ticker.Stop()
			server.doCron(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// doCron is the invoked cron function.
func (server *Server) doCron(ctx context.Context) {
	server.logger.Info("starting to clean out expired api keys")
	if err := server.svc.CleanupExpiredAPIKeys(ctx); err != nil {
		server.logger.Error("failed to clean expired api keys", slog.Any("error", err))
		return
	}
	server.logger.Info("cleaned expired api keys")
}
