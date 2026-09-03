// Package pgconv converts between pgx's pgtype values and the plain strings the
// API exposes on the wire.
package pgconv

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// UUIDString renders a pgtype.UUID as the canonical 8-4-4-4-12 hex form, or ""
// when it is null. Formatted by hand to stay independent of pgtype versions.
func UUIDString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// TimeString renders a pgtype.Timestamptz as RFC 3339 UTC, or "" when null.
func TimeString(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.UTC().Format(time.RFC3339)
}

// ParseUUID converts a string (path or query parameter) into a pgtype.UUID.
func ParseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(s)
	return u, err
}
