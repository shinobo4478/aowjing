package keyframes

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
	"github.com/shinobo4478/aowjing/backend/internal/generate"
	"github.com/shinobo4478/aowjing/backend/internal/images"
	"github.com/shinobo4478/aowjing/backend/internal/pgconv"
	"github.com/shinobo4478/aowjing/backend/internal/studio"
)

// Runner executes a queued keyframe request: send the prompt (and the profile's
// reference pictures) to the image model, store every picture that comes back,
// and record the outcome. Called by cmd/worker.
type Runner struct {
	q       *sqlc.Queries
	fakeFal bool
}

func NewRunner(q *sqlc.Queries, fakeFal bool) *Runner {
	return &Runner{q: q, fakeFal: fakeFal}
}

// generator picks the image generator. The fal key is read from the settings
// store each time (it may have just been set).
func (rn *Runner) generator(ctx context.Context) (generate.ImageGenerator, error) {
	if rn.fakeFal {
		return generate.FakeImageGenerator{}, nil
	}
	key, err := rn.q.GetSetting(ctx, "falApiKey")
	if errors.Is(err, pgx.ErrNoRows) {
		key = "" // the generator reports the missing-key error itself
	} else if err != nil {
		return nil, err
	}
	return generate.NewFalImageGenerator(key), nil
}

// Run processes one keyframe request by id. A model failure is stored on the
// row (status "failed") and is not returned — the job is done. Only a real
// error before any picture is paid for (bad id, DB down) is returned so asynq
// retries; once the model has been called, every outcome is recorded and nil
// is returned, because a retry would pay for the pictures again.
func (rn *Runner) Run(ctx context.Context, keyframeID string) error {
	id, err := pgconv.ParseUUID(keyframeID)
	if err != nil {
		return err
	}
	row, err := rn.q.GetKeyframe(ctx, id)
	if err != nil {
		return err
	}
	if row.Status != "pending" {
		return nil // already handled (a retry or a duplicate delivery)
	}

	model, ok := studio.KeyframeModelFor(row.Model)
	if !ok {
		return rn.fail(ctx, id, fmt.Errorf("unknown picture model %q", row.Model))
	}
	gen, err := rn.generator(ctx)
	if err != nil {
		return err
	}

	// The profile's reference pictures go along as data URIs. Fetched now, not
	// when the request was made, so what is sent is what is stored today.
	// Pictures of the same scene come first (they matter most), then the
	// profile's own, if the request asked for them.
	var refs []string
	if len(row.ExtraImageIds) > 0 {
		scene, err := rn.q.ListImagesByIDs(ctx, row.ExtraImageIds)
		if err != nil {
			return err
		}
		byID := make(map[string]sqlc.ListImagesByIDsRow, len(scene))
		for _, s := range scene {
			byID[pgconv.UUIDString(s.ID)] = s
		}
		for _, id := range row.ExtraImageIds { // keep the order that was asked for
			if s, ok := byID[pgconv.UUIDString(id)]; ok { // one deleted meanwhile is skipped
				refs = append(refs, images.DataURI(s.ContentType, s.Data))
			}
		}
	}
	if profileRefs := int(row.ReferenceCount) - len(row.ExtraImageIds); profileRefs > 0 {
		stored, err := rn.q.ListProfileReferenceData(ctx, row.ProfileID)
		if err != nil {
			return err
		}
		for _, s := range stored {
			refs = append(refs, images.DataURI(s.ContentType, s.Data))
		}
	}

	pictures, err := gen.GenerateImages(ctx, generate.ImageRequest{
		Model:      model.Key,
		ModelID:    model.ModelID(len(refs) > 0),
		Prompt:     row.Prompt,
		References: refs,
		Count:      int(row.Count),
		Aspect:     studio.KeyframeAspect,
	})
	if err != nil {
		return rn.fail(ctx, id, err)
	}

	stored, firstErr := 0, error(nil)
	for _, data := range pictures {
		contentType, err := images.Check(data)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		img, err := rn.q.CreateImage(ctx, sqlc.CreateImageParams{
			ContentType: contentType,
			SizeBytes:   int32(len(data)),
			Data:        data,
		})
		if err != nil {
			return rn.fail(ctx, id, fmt.Errorf("could not save a picture: %w", err))
		}
		if err := rn.q.AddKeyframeImage(ctx, sqlc.AddKeyframeImageParams{
			KeyframeID: id,
			ImageID:    img.ID,
			Position:   int16(stored + 1),
		}); err != nil {
			return rn.fail(ctx, id, fmt.Errorf("could not save a picture: %w", err))
		}
		stored++
	}
	if stored == 0 {
		return rn.fail(ctx, id, fmt.Errorf("the model's pictures cannot be used: %w", firstErr))
	}
	return rn.q.FinishKeyframe(ctx, sqlc.FinishKeyframeParams{ID: id, Status: "succeeded"})
}

// fail records an outcome of "failed" and returns nil (see Run).
func (rn *Runner) fail(ctx context.Context, id pgtype.UUID, cause error) error {
	return rn.q.FinishKeyframe(ctx, sqlc.FinishKeyframeParams{
		ID: id, Status: "failed", Error: cause.Error(),
	})
}
