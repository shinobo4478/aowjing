package generations

import (
	"context"
	"encoding/base64"
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
	falModel string // text-to-video model id
	// falImageModel is the image-to-video model id, used when a clip has an image.
	falImageModel string
	fakeFal       bool
}

func NewRunner(q *sqlc.Queries, falModel, falImageModel string, fakeFal bool) *Runner {
	return &Runner{q: q, falModel: falModel, falImageModel: falImageModel, fakeFal: fakeFal}
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
		return generate.NewFalVideoGenerator(key, rn.falModel, rn.falImageModel), nil
	}
	return generate.TextGenerator{}, nil
}

// imageFor returns the starting image to hand the generator: the pasted link,
// or — for an uploaded picture — a data URI ("data:image/png;base64,…"), which
// fal accepts in place of a public URL. "" means no image.
func (rn *Runner) imageFor(ctx context.Context, row sqlc.GetGenerationRow) (string, error) {
	if !row.ImageID.Valid {
		return row.ImageUrl, nil
	}
	img, err := rn.q.GetImage(ctx, row.ImageID)
	if err != nil {
		return "", err
	}
	return dataURI(img.ContentType, img.Data), nil
}

func dataURI(contentType string, data []byte) string {
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)
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

	imageURL, err := rn.imageFor(ctx, row)
	if err != nil {
		return err
	}

	finish := sqlc.FinishGenerationParams{ID: id, Provider: row.Provider, OutputKind: "text"}
	if res, genErr := gen.Generate(ctx, generate.Request{Prompt: row.InputPrompt, ImageURL: imageURL}); genErr != nil {
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
