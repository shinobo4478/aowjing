package generations

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestDataURI(t *testing.T) {
	data := []byte{0x89, 'P', 'N', 'G', 0x00, 0xff, 0x10}
	got := dataURI("image/png", data)

	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(got, prefix) {
		t.Fatalf("got %q, want prefix %q", got, prefix)
	}
	back, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(got, prefix))
	if err != nil {
		t.Fatalf("payload is not valid base64: %v", err)
	}
	if string(back) != string(data) {
		t.Errorf("round trip = %v, want %v", back, data)
	}
}
