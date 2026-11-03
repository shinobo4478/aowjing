package products

import "testing"

func TestParseProductIDs(t *testing.T) {
	const a = "11111111-1111-1111-1111-111111111111"
	const b = "22222222-2222-2222-2222-222222222222"
	const c = "33333333-3333-3333-3333-333333333333"

	tests := []struct {
		name    string
		in      []string
		wantLen int
		wantErr bool
	}{
		{"none clears the clip", nil, 0, false},
		{"empty list", []string{}, 0, false},
		{"one", []string{a}, 1, false},
		{"two keeps order", []string{b, a}, 2, false},
		{"three is over the cap", []string{a, b, c}, 0, true},
		{"same product twice", []string{a, a}, 0, true},
		{"same product twice, different case", []string{"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "AAAAAAAA-AAAA-AAAA-AAAA-AAAAAAAAAAAA"}, 0, true},
		{"not a uuid", []string{"nope"}, 0, true},
		{"one good one bad", []string{a, ""}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseProductIDs(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && len(got) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestParseProductIDsKeepsPinOrder(t *testing.T) {
	const a = "11111111-1111-1111-1111-111111111111"
	const b = "22222222-2222-2222-2222-222222222222"
	got, err := ParseProductIDs([]string{b, a})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Bytes[0] != 0x22 || got[1].Bytes[0] != 0x11 {
		t.Errorf("order not kept: %x then %x", got[0].Bytes[0], got[1].Bytes[0])
	}
}
