package keyframes

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

// A picture request with no scene and no cast ticked used to fail with "Failed to
// save the request": its uuid[] columns are NOT NULL and a nil slice is written as
// NULL. The helper must turn "nothing" into an empty, non-nil slice.
func TestNonNilUUIDs(t *testing.T) {
	if got := nonNilUUIDs(nil); got == nil || len(got) != 0 {
		t.Errorf("nil must become an empty non-nil slice, got %#v", got)
	}
	if got := nonNilUUIDs([]pgtype.UUID{}); got == nil {
		t.Error("an empty slice must stay non-nil")
	}
	id := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	got := nonNilUUIDs([]pgtype.UUID{id})
	if len(got) != 1 || got[0] != id {
		t.Errorf("ids must pass through unchanged, got %#v", got)
	}
}

func TestContextRE(t *testing.T) {
	const id = "0a1b2c3d-4e5f-6789-abcd-ef0123456789"
	cases := []struct {
		in   string
		want bool
	}{
		{"shot:" + id + ":start", true},
		{"shot:" + id + ":end", true},
		{"", false}, // an empty context is handled by the caller, not matched
		{"shot:" + id + ":middle", false},
		{"shot:not-a-uuid:start", false},
		{"shot:" + id + ":start%", false},
		{"x shot:" + id + ":start", false},
	}
	for _, c := range cases {
		if got := contextRE.MatchString(c.in); got != c.want {
			t.Errorf("contextRE(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
