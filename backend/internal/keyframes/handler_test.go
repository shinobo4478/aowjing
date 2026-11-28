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
