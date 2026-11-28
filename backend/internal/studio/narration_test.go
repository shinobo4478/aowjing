package studio

import "testing"

func TestSpeechCharsIgnoresWhiteSpace(t *testing.T) {
	if got := SpeechChars("ก ข\tค\nง"); got != 4 {
		t.Errorf("SpeechChars = %d, want 4", got)
	}
}

func TestNarrationBudget(t *testing.T) {
	for sec, want := range map[int]int{0: 0, 5: 49, 8: 82, 15: 159} {
		if got := NarrationBudget(sec); got != want {
			t.Errorf("NarrationBudget(%d) = %d, want %d", sec, got, want)
		}
	}
}

func TestSecondsForNarration(t *testing.T) {
	kling := []int{3, 4, 5, 6, 7, 8, 9, 10}
	veo := []int{4, 6, 8}
	repeat := func(n int) string {
		s := ""
		for i := 0; i < n; i++ {
			s += "ก"
		}
		return s
	}
	tests := []struct {
		name    string
		text    string
		options []int
		want    int
		fits    bool
	}{
		{"short text takes the shortest length", repeat(11), kling, 3, true},
		{"59 chars is 5.4 s, plus the tail 5.9 → 6", repeat(59), kling, 6, true},
		{"only the lengths the model takes", repeat(59), veo, 6, true},
		{"between two Veo lengths rounds up", repeat(66), veo, 8, true}, // 6.5 s
		{"too long for the longest", repeat(200), veo, 8, false},
		{"no options", repeat(10), nil, 0, false},
		{"unsorted options", repeat(40), []int{8, 4, 6}, 6, true}, // 4.1 s needs 6
	}
	for _, tt := range tests {
		got, fits := SecondsForNarration(tt.text, tt.options)
		if got != tt.want || fits != tt.fits {
			t.Errorf("%s: got (%d, %v), want (%d, %v)", tt.name, got, fits, tt.want, tt.fits)
		}
	}
}

func TestExplainerSeconds(t *testing.T) {
	for _, tt := range []struct {
		options []int
		want    int
	}{
		{[]int{3, 4, 5, 6, 7, 8, 9, 10, 15}, 8},
		{[]int{4, 6, 8}, 8},
		{[]int{4, 6}, 6},
		{[]int{10, 12}, 10},
		{nil, 0},
	} {
		if got := ExplainerSeconds(tt.options); got != tt.want {
			t.Errorf("ExplainerSeconds(%v) = %d, want %d", tt.options, got, tt.want)
		}
	}
}
