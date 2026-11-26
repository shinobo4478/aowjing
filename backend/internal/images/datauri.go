package images

import "encoding/base64"

// DataURI turns a stored picture into a "data:image/png;base64,…" string, which
// fal accepts in place of a public URL (the picture never has to be hosted).
func DataURI(contentType string, data []byte) string {
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)
}
