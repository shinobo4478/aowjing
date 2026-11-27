package generations

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/generate"
	"github.com/shinobo4478/aowjing/backend/internal/images"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

// Runner executes a queued generation: load it, pick the generator for its
// provider, run it over the stored prompt, and record the outcome. Called by
// cmd/worker.
type Runner struct {
	q       *sqlc.Queries
	fakeFal bool
}

func NewRunner(q *sqlc.Queries, fakeFal bool) *Runner {
	return &Runner{q: q, fakeFal: fakeFal}
}

// generatorFor maps a profile's provider key to a Generator. "fal" reads the
// API key from the settings store each time (it may have just been set); which
// fal models it calls, and whether the clip has sound, come from the profile's
// video model and sound setting (see studio.VideoModel).
func (rn *Runner) generatorFor(ctx context.Context, provider string, profile sqlc.Profile) (generate.Generator, error) {
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
		return generate.NewFalVideoGenerator(key, falSpec(profile)), nil
	}
	return generate.TextGenerator{}, nil
}

// falSpec turns a profile's video model and sound setting into the fal models
// and input shape the generator uses.
func falSpec(profile sqlc.Profile) generate.FalVideoSpec {
	m := studio.ModelFor(profile.VideoModel)
	return generate.FalVideoSpec{
		Wire:           string(m.Wire),
		TextModel:      m.TextModel,
		ImageModel:     m.ImageModel,
		FirstLastModel: m.FirstLastModel,
		EndPicture:     m.EndPicture,
		DefaultSeconds: m.DefaultSeconds,
		Audio:          profile.VideoAudio,
	}
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
	return images.DataURI(img.ContentType, img.Data), nil
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

	profile, err := rn.q.GetProfile(ctx, row.ProfileID)
	if err != nil {
		return err
	}
	gen, err := rn.generatorFor(ctx, row.Provider, profile)
	if err != nil {
		return err
	}

	imageURL, err := rn.imageFor(ctx, row)
	if err != nil {
		return err
	}
	// The picture the clip should end on, if the shot has one (a stored picture;
	// a data URI like the starting one).
	endURL := ""
	if row.EndImageID.Valid {
		end, err := rn.q.GetImage(ctx, row.EndImageID)
		if err != nil {
			return err
		}
		endURL = images.DataURI(end.ContentType, end.Data)
	}

	finish := sqlc.FinishGenerationParams{ID: id, Provider: row.Provider, OutputKind: "text"}
	req := generate.Request{Prompt: row.InputPrompt, ImageURL: imageURL, EndImageURL: endURL, Seconds: int(row.Seconds)}
	res, genErr := gen.Generate(ctx, req)
	// Keep fal's request id even when the clip failed after it was submitted:
	// the bill can still exist, and the id is how to find it.
	finish.FalRequestID = res.RequestID
	if genErr != nil {
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
