package studio

import "testing"

// Every registered model must be usable: named, with an image endpoint, lengths
// that include its default, and reachable through its own key.
func TestModelRegistryIsComplete(t *testing.T) {
	seen := map[ModelKey]bool{}
	for _, m := range Models() {
		if m.Key == "" || m.Label == "" || m.Wire == "" || m.ImageModel == "" {
			t.Errorf("model %+v has an empty required field", m)
		}
		if seen[m.Key] {
			t.Errorf("model %q is registered twice", m.Key)
		}
		seen[m.Key] = true
		if !ValidModel(string(m.Key)) {
			t.Errorf("model %q is registered but not Valid", m.Key)
		}
		if len(m.Seconds) == 0 || !m.ValidSeconds(m.DefaultSeconds) {
			t.Errorf("model %q: default %d s is not among its lengths %v", m.Key, m.DefaultSeconds, m.Seconds)
		}
		for _, n := range m.Seconds {
			if !ValidClipSeconds(n) {
				t.Errorf("model %q accepts %d s, outside the stored range", m.Key, n)
			}
		}
		// A model with a separate first-last endpoint must say it takes an end picture.
		if m.FirstLastModel != "" && !m.EndPicture {
			t.Errorf("model %q has a first-last endpoint but EndPicture is false", m.Key)
		}
	}
	if !ValidModel(string(DefaultModel)) {
		t.Errorf("DefaultModel %q is not registered", DefaultModel)
	}
	if ModelFor("no-such-model").Key != DefaultModel {
		t.Error("an unknown key should fall back to the default model")
	}
}

func TestVideoModelSeconds(t *testing.T) {
	kling, veo := ModelFor("kling-v3"), ModelFor("veo-3.1")
	for n, want := range map[int]bool{2: false, 3: true, 5: true, 15: true, 16: false} {
		if got := kling.ValidSeconds(n); got != want {
			t.Errorf("kling.ValidSeconds(%d) = %v, want %v", n, got, want)
		}
	}
	// Veo takes only 4, 6 or 8: the old default of 5 is not one of them.
	for n, want := range map[int]bool{3: false, 4: true, 5: false, 6: true, 8: true, 10: false} {
		if got := veo.ValidSeconds(n); got != want {
			t.Errorf("veo.ValidSeconds(%d) = %v, want %v", n, got, want)
		}
	}
	if got := veo.SecondsList(); got != "4, 6 or 8" {
		t.Errorf("SecondsList = %q", got)
	}
}

func TestClipUSD(t *testing.T) {
	kling := ModelFor("kling-v3")
	// 5 s at $0.084/s is $0.42 (the first real clip); float rounding must not leak.
	if got := kling.ClipUSD(5, false); got != 0.42 {
		t.Errorf("kling 5 s silent = %v, want 0.42", got)
	}
	if got := kling.ClipUSD(10, false); got != 0.84 {
		t.Errorf("kling 10 s silent = %v, want 0.84", got)
	}
	// A real 6 s Veo clip with sound was billed $2.40: that is $0.40/s.
	if got := ModelFor("veo-3.1").ClipUSD(6, true); got != 2.4 {
		t.Errorf("veo 6 s with sound = %v, want 2.4", got)
	}
	if got := ModelFor("veo-3.1").ClipUSD(8, false); got != 1.6 {
		t.Errorf("veo 8 s silent = %v, want 1.6", got)
	}
}

func TestFastModelOffersOnlyWhatWasRead(t *testing.T) {
	fast := ModelFor("veo-3.1-fast")
	if fast.SupportsText() || fast.EndPicture {
		t.Errorf("Veo Fast should not offer text-only clips or an end picture until those endpoints are read: %+v", fast)
	}
}
