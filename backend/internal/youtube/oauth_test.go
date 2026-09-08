package youtube

import (
	"strings"
	"testing"

	"github.com/shinobo4478/aowjing/backend/internal/config"
)

func testConfig() config.Config {
	return config.Config{
		YouTubeClientID:     "client-id.apps.googleusercontent.com",
		YouTubeClientSecret: "client-secret",
		YouTubeRedirectURL:  "http://localhost:8080/youtube/oauth/callback",
	}
}

func TestVerifyState(t *testing.T) {
	const secret = "s3cret-admin-pw"
	const channelID = "11111111-2222-3333-4444-555555555555"

	good := signState(secret, channelID)

	tests := []struct {
		name   string
		secret string
		state  string
		wantID string
		wantOK bool
	}{
		{"round trip", secret, good, channelID, true},
		{"wrong secret", "other", good, "", false},
		{"tampered signature", secret, good[:len(good)-2] + "xx", "", false},
		{"not two parts", secret, "no-dot-here", "", false},
		{"empty", secret, "", "", false},
		{"garbage base64", secret, "!!!.###", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := verifyState(tt.secret, tt.state)
			if gotOK != tt.wantOK || gotID != tt.wantID {
				t.Errorf("verifyState() = (%q, %v), want (%q, %v)", gotID, gotOK, tt.wantID, tt.wantOK)
			}
		})
	}
}

func TestAuthCodeURLHasOfflineConsent(t *testing.T) {
	cfg := testConfig()
	got := authCodeURL(cfg, "state123")

	for _, want := range []string{
		"access_type=offline",
		"prompt=consent",
		"state=state123",
		"accounts.google.com",
		"youtube.upload",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("auth URL missing %q\n got: %s", want, got)
		}
	}
}
