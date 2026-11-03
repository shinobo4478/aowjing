package products

import "testing"

func TestNormalizeAndValidate(t *testing.T) {
	valid := productInput{
		Name:      "Power strip",
		ShopeeURL: "https://shopee.co.th/Power-Strip-i.111.222?sp_atk=x",
	}

	tests := []struct {
		name     string
		mutate   func(in *productInput)
		wantKeys []string // fields expected to have an error
	}{
		{"valid minimal", func(in *productInput) {}, nil},
		{"valid with affiliate + price", func(in *productInput) {
			in.AffiliateURL = "https://s.shopee.co.th/AbCdEf"
			in.PriceTHB = 299
		}, nil},
		{"name too short", func(in *productInput) { in.Name = " a " }, []string{"name"}},
		{"name too long", func(in *productInput) {
			for len(in.Name) <= 120 {
				in.Name += "x"
			}
		}, []string{"name"}},
		{"not a Shopee link", func(in *productInput) { in.ShopeeURL = "https://example.com/x" }, []string{"shopeeUrl"}},
		{"empty Shopee link", func(in *productInput) { in.ShopeeURL = "" }, []string{"shopeeUrl"}},
		{"affiliate not a URL", func(in *productInput) { in.AffiliateURL = "just text" }, []string{"affiliateUrl"}},
		{"affiliate wrong scheme", func(in *productInput) { in.AffiliateURL = "javascript:alert(1)" }, []string{"affiliateUrl"}},
		{"negative price", func(in *productInput) { in.PriceTHB = -1 }, []string{"priceThb"}},
		{"absurd price", func(in *productInput) { in.PriceTHB = 20_000_000 }, []string{"priceThb"}},
		{"note too long", func(in *productInput) {
			for len(in.Note) <= 600 {
				in.Note += "x"
			}
		}, []string{"note"}},
		{"several at once", func(in *productInput) {
			in.Name = ""
			in.ShopeeURL = "nope"
			in.PriceTHB = -5
		}, []string{"name", "shopeeUrl", "priceThb"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := valid
			tt.mutate(&in)
			_, errs := in.normalizeAndValidate()

			if len(errs) != len(tt.wantKeys) {
				t.Fatalf("errors = %v, want keys %v", errs, tt.wantKeys)
			}
			for _, k := range tt.wantKeys {
				if _, ok := errs[k]; !ok {
					t.Errorf("missing error for %q; got %v", k, errs)
				}
			}
		})
	}
}

func TestNormalizeReturnsCleanLink(t *testing.T) {
	in := productInput{
		Name:      "  Power strip ",
		ShopeeURL: "  shopee.co.th/Power-Strip-i.111.222?sp_atk=x#frag ",
	}
	link, errs := in.normalizeAndValidate()
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if in.Name != "Power strip" {
		t.Errorf("name not trimmed: %q", in.Name)
	}
	if link.URL != "https://shopee.co.th/Power-Strip-i.111.222" || !link.HasIDs || link.ShopID != 111 || link.ItemID != 222 {
		t.Errorf("link = %+v", link)
	}
}
