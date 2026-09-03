package generations

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/generate"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
)

// Runner executes a queued generation: load it, pick the generator for its
// provider, run it over the stored prompt, and record the outcome. Called by
// cmd/worker.
type Runner struct {
	q        *sqlc.Queries
	falModel string
	fakeFal  bool
}

func NewRunner(q *sqlc.Queries, falModel string, fakeFal bool) *Runner {
	return &Runner{q: q, falModel: falModel, fakeFal: fakeFal}
}

// generatorFor maps a profile's provider key to a Generator. "fal" reads the
// API key from the settings store each time (it may have just been set).
func (rn *Runner) generatorFor(ctx context.Context, provider string) (generate.Generator, error) {
	if provider == "fal" {
		if rn.fakeFal {
			return generate.FakeFalGenerator{}, nil
		}
		key, err := rn.q.GetSetting(ctx, "falApiKey")
		if errors.Is(err, pgx.ErrNoRows) {
			key = "" // NewFalVideoGenerator reports the missing-key error itself
		} else if err != nil {
			return nil, err
		}
		return generate.NewFalVideoGenerator(key, rn.falModel), nil
	}
	return generate.TextGenerator{}, nil
}

// Run processes one generation by id. A provider failure is stored on the row
// (status "failed") and is not returned — the job is done. A real error (bad
// id, DB down) is returned so asynq retries.
func (rn *Runner) Run(ctx context.Context, generationID string) error {
	id, err := pgconv.ParseUUID(generationID)
	if err != nil {
		return err
	}
	row, err := rn.q.GetGeneration(ctx, id)
	if err != nil {
		return err
	}

	gen, err := rn.generatorFor(ctx, row.Provider)
	if err != nil {
		return err
	}

	finish := sqlc.FinishGenerationParams{ID: id, Provider: row.Provider, OutputKind: "text"}
	if res, genErr := gen.Generate(ctx, row.InputPrompt); genErr != nil {
		finish.Status = "failed"
		finish.Error = genErr.Error()
	} else {
		finish.Status = "succeeded"
		finish.Output = res.Output
		finish.OutputKind = res.Kind
		finish.Provider = res.Provider
		finish.Model = res.Model
	}
	return rn.q.FinishGeneration(ctx, finish)
}
