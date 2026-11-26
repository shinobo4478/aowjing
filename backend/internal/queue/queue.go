// Package queue is the thin seam over asynq: task names, payloads, and an
// enqueue-only client for the API. The worker (cmd/worker) owns the consuming
// side.
package queue

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
)

// TaskGenerationRun runs one generation through its provider and records the
// result. Payload: GenerationRunPayload.
const TaskGenerationRun = "generation:run"

type GenerationRunPayload struct {
	GenerationID string `json:"generationId"`
}

// TaskKeyframeRun makes the keyframe pictures of one keyframe request.
// Payload: KeyframeRunPayload.
const TaskKeyframeRun = "keyframe:run"

type KeyframeRunPayload struct {
	KeyframeID string `json:"keyframeId"`
}

// Client enqueues tasks. Safe for concurrent use; call Close on shutdown.
type Client struct {
	c *asynq.Client
}

func NewClient(redisAddr string) *Client {
	return &Client{c: asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})}
}

func (c *Client) Close() error { return c.c.Close() }

func (c *Client) EnqueueGenerationRun(ctx context.Context, generationID string) error {
	payload, err := json.Marshal(GenerationRunPayload{GenerationID: generationID})
	if err != nil {
		return err
	}
	_, err = c.c.EnqueueContext(ctx, asynq.NewTask(TaskGenerationRun, payload))
	return err
}

// EnqueueKeyframeRun queues a keyframe request. It retries at most twice: a
// retry calls the image model again, which is paid for each time.
func (c *Client) EnqueueKeyframeRun(ctx context.Context, keyframeID string) error {
	payload, err := json.Marshal(KeyframeRunPayload{KeyframeID: keyframeID})
	if err != nil {
		return err
	}
	_, err = c.c.EnqueueContext(ctx, asynq.NewTask(TaskKeyframeRun, payload), asynq.MaxRetry(2))
	return err
}
