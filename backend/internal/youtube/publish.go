package youtube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
)

// maxVideoBytes caps what we'll pull from the generator's URL into memory
// before uploading. Generator clips are a few seconds long; this is a guard,
// not a real limit.
const maxVideoBytes = 256 << 20 // 256 MiB

type videoMeta struct {
	Title       string
	Description string
	Privacy     string // "private" | "unlisted" | "public"
}

// downloadVideo fetches the generator's output URL into memory.
func downloadVideo(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download video: %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxVideoBytes+1))
}

// uploadVideo sends a multipart/related request to the YouTube Data API and
// returns the new video id. client must carry a valid OAuth token.
func uploadVideo(ctx context.Context, client *http.Client, meta videoMeta, video []byte) (string, error) {
	if len(video) > maxVideoBytes {
		return "", fmt.Errorf("video is larger than %d bytes", maxVideoBytes)
	}

	metaJSON, _ := json.Marshal(map[string]any{
		"snippet": map[string]any{
			"title":       meta.Title,
			"description": meta.Description,
		},
		"status": map[string]any{
			"privacyStatus": meta.Privacy,
		},
	})

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	jsonPart, _ := mw.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"application/json; charset=UTF-8"},
	})
	jsonPart.Write(metaJSON)

	videoPart, _ := mw.CreatePart(textproto.MIMEHeader{
		"Content-Type": {"video/*"},
	})
	videoPart.Write(video)
	mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://www.googleapis.com/upload/youtube/v3/videos?part=snippet,status&uploadType=multipart",
		&body)
	if err != nil {
		return "", err
	}
	// The API wants multipart/related, not multipart/form-data.
	req.Header.Set("Content-Type", "multipart/related; boundary="+mw.Boundary())

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<10))
		return "", fmt.Errorf("youtube upload: %s: %s", resp.Status, snippet)
	}

	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.ID == "" {
		return "", fmt.Errorf("youtube upload: no video id in response")
	}
	return out.ID, nil
}
