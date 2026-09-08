package youtube

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/shinobo4478/aowjing/backend/internal/config"
	"github.com/shinobo4478/aowjing/backend/internal/database/sqlc"
)

// Google's OAuth endpoints, inlined so we don't pull in golang.org/x/oauth2/google
// (and its GCE-metadata dependency) just for two constants. These match the
// auth_uri / token_uri in the downloaded client JSON.
var googleEndpoint = oauth2.Endpoint{
	AuthURL:  "https://accounts.google.com/o/oauth2/auth",
	TokenURL: "https://oauth2.googleapis.com/token",
}

// Scopes we ask the user to grant: upload = publish a video; readonly = read
// back the connected channel's name for display.
var scopes = []string{
	"https://www.googleapis.com/auth/youtube.upload",
	"https://www.googleapis.com/auth/youtube.readonly",
}

func oauthConfig(cfg config.Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.YouTubeClientID,
		ClientSecret: cfg.YouTubeClientSecret,
		RedirectURL:  cfg.YouTubeRedirectURL,
		Scopes:       scopes,
		Endpoint:     googleEndpoint,
	}
}

// authCodeURL builds the Google consent URL. access_type=offline +
// prompt=consent make Google return a refresh token on every run, not only the
// first time the user consents.
func authCodeURL(cfg config.Config, state string) string {
	return oauthConfig(cfg).AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)
}

// --- CSRF state (stateless) ---------------------------------------------
//
// The OAuth "state" round-trips through Google's servers, so it must be
// unforgeable. We sign "<channelID>:<unix-expiry>" with HMAC-SHA256 keyed on
// the admin password (always set) — no server-side session or cookie needed.

const stateTTL = 10 * time.Minute

func signState(secret, channelID string) string {
	msg := channelID + ":" + strconv.FormatInt(time.Now().Add(stateTTL).Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(msg)) + "." + sig
}

// verifyState returns the channelID carried by a state string, or ok=false if
// the signature is wrong or it has expired.
func verifyState(secret, state string) (channelID string, ok bool) {
	msgB64, sig, found := strings.Cut(state, ".")
	if !found {
		return "", false
	}
	msg, err := base64.RawURLEncoding.DecodeString(msgB64)
	if err != nil {
		return "", false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(msg)
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(want), []byte(sig)) {
		return "", false
	}
	cid, expStr, found := strings.Cut(string(msg), ":")
	if !found {
		return "", false
	}
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return "", false
	}
	return cid, true
}

// --- Tokens for API calls -------------------------------------------

// freshToken turns the stored token into a valid one, refreshing via the
// refresh token if the access token has expired. The caller compares the
// result's AccessToken with what was stored and persists it if it changed.
func freshToken(ctx context.Context, cfg config.Config, acct sqlc.YoutubeAccount) (*oauth2.Token, error) {
	stored := &oauth2.Token{
		AccessToken:  acct.AccessToken,
		RefreshToken: acct.RefreshToken,
		Expiry:       acct.TokenExpiry.Time,
	}
	return oauthConfig(cfg).TokenSource(ctx, stored).Token()
}

// --- YouTube identity --------------------------------------------------

type ytChannel struct {
	ID    string
	Title string
}

// fetchChannel reads the authenticated user's own channel (id + title) for
// display. A Google account with no channel yields a zero value and no error.
func fetchChannel(ctx context.Context, c *http.Client) (ytChannel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://www.googleapis.com/youtube/v3/channels?part=snippet&mine=true", nil)
	if err != nil {
		return ytChannel{}, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return ytChannel{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ytChannel{}, fmt.Errorf("youtube channels.list: %s", resp.Status)
	}

	var body struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title string `json:"title"`
			} `json:"snippet"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return ytChannel{}, err
	}
	if len(body.Items) == 0 {
		return ytChannel{}, nil
	}
	return ytChannel{ID: body.Items[0].ID, Title: body.Items[0].Snippet.Title}, nil
}
