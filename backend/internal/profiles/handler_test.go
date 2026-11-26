package profiles

import "testing"

func TestVideoStyleValidation(t *testing.T) {
	tests := []struct {
		name      string
		style     string
		wantStyle string // after normalizing
		wantErr   bool
	}{
		{"empty becomes the default", "", "3d", false},
		{"blank becomes the default", "   ", "3d", false},
		{"3d", "3d", "3d", false},
		{"trims", " 3d ", "3d", false},
		{"unknown", "claymation", "claymation", true},
		{"case matters", "3D", "3D", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := profileInput{Name: "Retro", Niche: "retro games", VideoStyle: tt.style}
			errs := in.normalizeAndValidate()
			if _, bad := errs["videoStyle"]; bad != tt.wantErr {
				t.Errorf("videoStyle error = %v, want %v (errs: %v)", bad, tt.wantErr, errs)
			}
			if in.VideoStyle != tt.wantStyle {
				t.Errorf("VideoStyle = %q, want %q", in.VideoStyle, tt.wantStyle)
			}
			if !tt.wantErr && len(errs) != 0 {
				t.Errorf("unexpected errors: %v", errs)
			}
		})
	}
}
