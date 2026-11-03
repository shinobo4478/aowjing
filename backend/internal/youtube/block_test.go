package youtube

import (
	"strings"
	"testing"
)

func TestProductBlock(t *testing.T) {
	strip := blockItem{
		Name: "ปลั๊กพ่วง 4 ช่อง", ShopeeURL: "https://shopee.co.th/Strip-i.1.2",
		AffiliateURL: "https://s.shopee.co.th/AbC", PriceTHB: 1299,
	}
	pillow := blockItem{Name: "Neck pillow", ShopeeURL: "https://shopee.co.th/Pillow-i.3.4"}

	tests := []struct {
		name         string
		items        []blockItem
		want         []string // substrings that must appear
		wantNot      []string // substrings that must not
		wantEmpty    bool
		wantNotice   bool
		wantOrderIdx []string // appear in this order
	}{
		{name: "nothing pinned", items: nil, wantEmpty: true},
		{
			name: "affiliate link wins, price formatted, notice shown", items: []blockItem{strip},
			want:       []string{blockHeading, "1. ปลั๊กพ่วง 4 ช่อง (฿1,299)", "https://s.shopee.co.th/AbC"},
			wantNot:    []string{"https://shopee.co.th/Strip-i.1.2"},
			wantNotice: true,
		},
		{
			name: "no affiliate link: plain page, no price, no notice", items: []blockItem{pillow},
			want:       []string{"1. Neck pillow\nhttps://shopee.co.th/Pillow-i.3.4"},
			wantNot:    []string{"฿"},
			wantNotice: false,
		},
		{
			name: "two in pin order, notice because one is affiliate", items: []blockItem{pillow, strip},
			wantNotice:   true,
			wantOrderIdx: []string{"1. Neck pillow", "2. ปลั๊กพ่วง"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := productBlock(tt.items)
			if tt.wantEmpty {
				if got != "" {
					t.Fatalf("want empty, got %q", got)
				}
				return
			}
			for _, s := range tt.want {
				if !strings.Contains(got, s) {
					t.Errorf("missing %q in:\n%s", s, got)
				}
			}
			for _, s := range tt.wantNot {
				if strings.Contains(got, s) {
					t.Errorf("unexpected %q in:\n%s", s, got)
				}
			}
			if has := strings.Contains(got, affiliateNotice); has != tt.wantNotice {
				t.Errorf("affiliate notice present = %v, want %v", has, tt.wantNotice)
			}
			last := -1
			for _, s := range tt.wantOrderIdx {
				i := strings.Index(got, s)
				if i < 0 || i < last {
					t.Errorf("%q out of order in:\n%s", s, got)
				}
				last = i
			}
			if strings.HasSuffix(got, "\n") {
				t.Errorf("block ends with a newline: %q", got)
			}
		})
	}
}
