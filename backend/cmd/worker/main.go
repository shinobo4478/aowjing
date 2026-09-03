// Command worker consumes the generation job queue.
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"

	"github.com/shinobo4478/aowjing/backend/internal/config"
	"github.com/shinobo4478/aowjing/backend/internal/database"
	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/generate"
	"github.com/shinobo4478/aowjing/backend/internal/generations"
	"github.com/shinobo4478/aowjing/backend/internal/queue"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	config.LoadDotEnv(".env")

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	log.Println("connected to database")

	// Phase 2: swap TextGenerator for a factory keyed on the generation's
	// provider once FalVideoGenerator exists (item 1c).
	runner := generations.NewRunner(sqlc.New(db), generate.TextGenerator{})

	mux := asynq.NewServeMux()
	mux.HandleFunc(queue.TaskGenerationRun, func(ctx context.Context, t *asynq.Task) error {
		var p queue.GenerationRunPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			// A malformed payload will never succeed — don't retry.
			return asynq.SkipRetry
		}
		return runner.Run(ctx, p.GenerationID)
	})

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.RedisAddr},
		asynq.Config{Concurrency: 5},
	)

	log.Printf("worker listening on redis %s", cfg.RedisAddr)
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(mux) }()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Println("shutdown signal received")
		srv.Shutdown()
		return nil
	}
}
