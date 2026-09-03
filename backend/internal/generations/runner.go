package generations

import (
	"context"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/generate"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
)

// Runner executes a queued generation: load it, run the generator over the
// stored prompt, and record the outcome. Called by cmd/worker.
type Runner struct {
	q   *sqlc.Queries
	gen generate.Generator
}

func NewRunner(q *sqlc.Queries, gen generate.Generator) *Runner {
	return &Runner{q: q, gen: gen}
}

// Run processes one generation by id. A provider failure is stored on the row
// (status "failed") and is not returned as an error — the job is done. A real
// error (bad id, DB down) is returned so asynq retries.
func (rn *Runner) Run(ctx context.Context, generationID string) error {
	id, err := pgconv.ParseUUID(generationID)
	if err != nil {
		return err
	}
	row, err := rn.q.GetGeneration(ctx, id)
	if err != nil {
		return err
	}

	finish := sqlc.FinishGenerationParams{ID: id, Provider: row.Provider}
	if res, genErr := rn.gen.Generate(ctx, row.InputPrompt); genErr != nil {
		finish.Status = "failed"
		finish.Error = genErr.Error()
	} else {
		finish.Status = "succeeded"
		finish.Output = res.Output
		finish.Provider = res.Provider
		finish.Model = res.Model
	}
	return rn.q.FinishGeneration(ctx, finish)
}
