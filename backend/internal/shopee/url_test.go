package shopee

import (
	"errors"
	"testing"
)

func TestParseLink(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    Link
		wantErr bool
	}{
		{
			"slug form, query dropped",
			"https://shopee.co.th/ปลั๊กพ่วง-4-ช่อง-i.123456.7890123?sp_atk=abc&xptdk=1",
			Link{URL: "https://shopee.co.th/%E0%B8%9B%E0%B8%A5%E0%B8%B1%E0%B9%8A%E0%B8%81%E0%B8%9E%E0%B9%88%E0%B8%A7%E0%B8%87-4-%E0%B8%8A%E0%B9%88%E0%B8%AD%E0%B8%87-i.123456.7890123", ShopID: 123456, ItemID: 7890123, HasIDs: true},
			false,
		},
		{
			"ascii slug",
			"https://shopee.co.th/Power-Strip-i.111.222",
			Link{URL: "https://shopee.co.th/Power-Strip-i.111.222", ShopID: 111, ItemID: 222, HasIDs: true},
			false,
		},
		{
			"/product/ form",
			"https://shopee.co.th/product/111/222",
			Link{URL: "https://shopee.co.th/product/111/222", ShopID: 111, ItemID: 222, HasIDs: true},
			false,
		},
		{
			"no scheme, www, mixed case host",
			"  WWW.Shopee.co.th/Power-Strip-i.111.222  ",
			Link{URL: "https://www.shopee.co.th/Power-Strip-i.111.222", ShopID: 111, ItemID: 222, HasIDs: true},
			false,
		},
		{
			"mobile subdomain",
			"https://m.shopee.co.th/product/111/222/",
			Link{URL: "https://m.shopee.co.th/product/111/222/", ShopID: 111, ItemID: 222, HasIDs: true},
			false,
		},
		{
			"short link has no ids",
			"https://s.shopee.co.th/AbCdEf",
			Link{URL: "https://s.shopee.co.th/AbCdEf"},
			false,
		},
		{
			"shp.ee short link",
			"https://shp.ee/xyz123",
			Link{URL: "https://shp.ee/xyz123"},
			false,
		},
		{
			"shop page, not a product",
			"https://shopee.co.th/some_shop",
			Link{URL: "https://shopee.co.th/some_shop"},
			false,
		},
		{
			"ids that overflow are ignored",
			"https://shopee.co.th/x-i.99999999999999999999.1",
			Link{URL: "https://shopee.co.th/x-i.99999999999999999999.1"},
			false,
		},
		{"other country", "https://shopee.sg/x-i.1.2", Link{}, true},
		{"lookalike suffix", "https://evilshopee.co.th/x-i.1.2", Link{}, true},
		{"lookalike prefix", "https://shopee.co.th.evil.com/x-i.1.2", Link{}, true},
		{"another site", "https://lazada.co.th/x-i.1.2", Link{}, true},
		{"not http", "ftp://shopee.co.th/x-i.1.2", Link{}, true},
		{"javascript", "javascript:alert(1)", Link{}, true},
		{"empty", "   ", Link{}, true},
		{"garbage", "ปลั๊กพ่วง", Link{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLink(tt.in)
			if tt.wantErr {
				if !errors.Is(err, ErrNotShopee) {
					t.Fatalf("ParseLink(%q) err = %v, want ErrNotShopee", tt.in, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseLink(%q) unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("ParseLink(%q)\n got  %+v\n want %+v", tt.in, got, tt.want)
			}
		})
	}
}
