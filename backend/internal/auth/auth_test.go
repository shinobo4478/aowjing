package auth

import "testing"

func TestCheckCredentials(t *testing.T) {
	a := &Authenticator{adminUser: "admin", adminPass: "s3cret"}

	tests := []struct {
		name string
		user string
		pass string
		want bool
	}{
		{"exact match", "admin", "s3cret", true},
		{"wrong password", "admin", "nope", false},
		{"wrong username", "root", "s3cret", false},
		{"both wrong", "root", "nope", false},
		{"empty", "", "", false},
		{"password is username", "admin", "admin", false},
		{"case sensitive user", "Admin", "s3cret", false},
		{"trailing space in password", "admin", "s3cret ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := a.checkCredentials(tt.user, tt.pass); got != tt.want {
				t.Errorf("checkCredentials(%q, %q) = %v, want %v", tt.user, tt.pass, got, tt.want)
			}
		})
	}
}

func TestClearCookieExpiresImmediately(t *testing.T) {
	a := &Authenticator{}
	c := a.clearCookie()

	if c.Name != cookieName {
		t.Errorf("cookie name = %q, want %q", c.Name, cookieName)
	}
	if c.MaxAge >= 0 {
		t.Errorf("MaxAge = %d, want negative", c.MaxAge)
	}
	if c.Value != "" {
		t.Errorf("Value = %q, want empty", c.Value)
	}
}
