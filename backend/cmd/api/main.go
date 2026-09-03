// Command api is the ACMP backend HTTP server.
package main

import (
	"bufio"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/shinobo4478/aowjing/backend/internal/config"
	"github.com/shinobo4478/aowjing/backend/internal/database"
	"github.com/shinobo4478/aowjing/backend/internal/server"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run does the real work so it can return errors instead of calling exit deep
// in the call stack — a common Go pattern for testable entrypoints.
func run() error {
	loadDotEnv(".env")

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Root context cancelled on SIGINT/SIGTERM; everything hangs off it.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	log.Println("connected to database")

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.New(db, cfg),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Serve in a goroutine so main can wait for a shutdown signal.
	errCh := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Println("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// loadDotEnv reads simple KEY=VALUE lines from path into the process
// environment. Missing file is not an error. Existing variables are not
// overridden, so real environment config still wins in deployed setups.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}
