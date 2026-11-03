package products

import "testing"

func TestCommas(t *testing.T) {
	for in, want := range map[int]string{
		0: "0", 7: "7", 299: "299", 1890: "1,890", 12000: "12,000", 1234567: "1,234,567",
	} {
		if got := Commas(in); got != want {
			t.Errorf("Commas(%d) = %q, want %q", in, got, want)
		}
	}
}
