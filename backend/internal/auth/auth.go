// Package auth implements login/session handling for the single admin user.
//
// A successful login mints a random opaque token, stores only its SHA-256 hash
// in the sessions table, and sends the raw token to the browser in an
// HttpOnly cookie. Every protected request looks the hash up again, so logout
// (and expiry) revoke access immediately.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
)

const (
	cookieName = "acmp_session"
	sessionTTL = 7 * 24 * time.Hour
	tokenBytes = 32
)

// Authenticator holds the admin credentials and session store.
type Authenticator struct {
	q            *sqlc.Queries
	adminUser    string
	adminPass    string
	cookieSecure bool
}

func New(q *sqlc.Queries, adminUser, adminPass string, cookieSecure bool) *Authenticator {
	return &Authenticator{
		q:            q,
		adminUser:    adminUser,
		adminPass:    adminPass,
		cookieSecure: cookieSecure,
	}
}

// checkCredentials compares both fields in constant time so a wrong username
// and a wrong password take the same time to reject.
func (a *Authenticator) checkCredentials(user, pass string) bool {
	userOK := subtle.ConstantTimeCompare([]byte(user), []byte(a.adminUser)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(pass), []byte(a.adminPass)) == 1
	return userOK && passOK
}

// createSession generates a token, persists its hash, and returns the raw
// token (for the cookie) and its expiry.
func (a *Authenticator) createSession(ctx context.Context) (string, time.Time, error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	expires := time.Now().Add(sessionTTL)

	sum := sha256.Sum256([]byte(token))
	err := a.q.CreateSession(ctx, sqlc.CreateSessionParams{
		TokenHash: sum[:],
		ExpiresAt: pgtype.Timestamptz{Time: expires, Valid: true},
	})
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

// validToken reports whether token maps to a live (unexpired) session.
func (a *Authenticator) validToken(ctx context.Context, token string) bool {
	if token == "" {
		return false
	}
	sum := sha256.Sum256([]byte(token))
	_, err := a.q.GetSession(ctx, sum[:])
	return err == nil
}

func (a *Authenticator) destroySession(ctx context.Context, token string) error {
	sum := sha256.Sum256([]byte(token))
	return a.q.DeleteSession(ctx, sum[:])
}

// sessionCookie builds the Set-Cookie for a fresh session.
func (a *Authenticator) sessionCookie(token string, expires time.Time) *http.Cookie {
	c := &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Expires:  expires,
		SameSite: http.SameSiteLaxMode,
	}
	if a.cookieSecure {
		c.Secure = true
		c.SameSite = http.SameSiteNoneMode
	}
	return c
}

// clearCookie builds a Set-Cookie that immediately expires the session cookie.
func (a *Authenticator) clearCookie() *http.Cookie {
	c := a.sessionCookie("", time.Unix(0, 0))
	c.MaxAge = -1
	return c
}
