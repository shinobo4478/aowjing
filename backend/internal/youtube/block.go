package youtube

import (
	"fmt"
	"strings"
)

// The product block is the part of a YouTube description that lists pinned
// Shopee products. It is plain text in Thai, to match the channels it goes on.
// Other platforms (Facebook, …) will get their own renderer next to this one;
// the pinned products themselves belong to the clip, not to YouTube.
const (
	blockHeading = "สินค้าในคลิป"
	// Shown whenever a listed link is an affiliate link.
	affiliateNotice = "* ลิงก์ที่ระบุเป็นลิงก์พันธมิตร (affiliate) เจ้าของช่องอาจได้รับค่าคอมมิชชันเมื่อคุณสั่งซื้อผ่านลิงก์ โดยราคาที่คุณจ่ายไม่เปลี่ยนแปลง"
)

type blockItem struct {
	Name         string
	ShopeeURL    string
	AffiliateURL string
	PriceTHB     int
}

// link is what viewers should click: the affiliate link when there is one,
// otherwise the plain product page.
func (b blockItem) link() string {
	if b.AffiliateURL != "" {
		return b.AffiliateURL
	}
	return b.ShopeeURL
}

// productBlock renders the pinned products for a description, or "" when none
// are pinned. The affiliate notice is added only if at least one listed link is
// an affiliate link.
func productBlock(items []blockItem) string {
	if len(items) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(blockHeading)
	sb.WriteString("\n")
	hasAffiliate := false
	for i, it := range items {
		title := it.Name
		if it.PriceTHB > 0 {
			title = fmt.Sprintf("%s (฿%s)", it.Name, withCommas(it.PriceTHB))
		}
		fmt.Fprintf(&sb, "%d. %s\n%s\n", i+1, title, it.link())
		if it.AffiliateURL != "" {
			hasAffiliate = true
		}
	}
	if hasAffiliate {
		sb.WriteString("\n")
		sb.WriteString(affiliateNotice)
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// withCommas formats 1890 as "1,890".
func withCommas(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
