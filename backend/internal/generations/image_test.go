package generations

import (
	"strings"
	"testing"
)

func TestValidateImageURL(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"empty means no image", "", "", false},
		{"blank means no image", "   ", "", false},
		{"https", "https://cf.shopee.co.th/file/abc.jpg", "https://cf.shopee.co.th/file/abc.jpg", false},
		{"http", "http://example.com/a.png", "http://example.com/a.png", false},
		{"trims", "  https://example.com/a.png \n", "https://example.com/a.png", false},
		{"no scheme", "example.com/a.png", "", true},
		{"ftp", "ftp://example.com/a.png", "", true},
		{"data uri", "data:image/png;base64,AAAA", "", true},
		{"no host", "https:///a.png", "", true},
		{"too long", "https://example.com/" + strings.Repeat("a", maxImageURLLen), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateImageURL(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
