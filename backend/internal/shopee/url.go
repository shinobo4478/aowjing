// Package shopee understands Shopee links. Today that is only URL parsing —
// there is no Shopee account or API client yet (see CLAUDE.md, Phase 3).
package shopee

import (
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// ErrNotShopee means the text isn't an http(s) link on a Shopee Thailand host.
var ErrNotShopee = errors.New("not a Shopee Thailand link")

// Link is a pasted Shopee product link, cleaned up.
type Link struct {
	// URL is the canonical form: https, no query string, no fragment. Tracking
	// parameters (sp_atk, utm_*, ...) are dropped so the stored link is the
	// plain product page.
	URL string
	// ShopID and ItemID identify the product. HasIDs is false for short links
	// (s.shopee.co.th/…, shp.ee/…), which only a redirect reveals — we don't
	// follow redirects on Shopee's servers.
	ShopID, ItemID int64
	HasIDs         bool
}

var (
	// /Some-Product-Name-i.123456.7890123
	slugIDs = regexp.MustCompile(`(?:^|[-/])i\.(\d+)\.(\d+)/?$`)
	// /product/123456/7890123
	pathIDs = regexp.MustCompile(`^/product/(\d+)/(\d+)/?$`)
)

// isShopeeHost reports whether host is shopee.co.th, a subdomain of it, or the
// shp.ee short-link domain. Suffix matching requires the leading dot so that
// "evilshopee.co.th" and "shopee.co.th.evil.com" are rejected.
func isShopeeHost(host string) bool {
	return host == "shopee.co.th" ||
		strings.HasSuffix(host, ".shopee.co.th") ||
		host == "shp.ee"
}

// ParseLink validates and canonicalises a pasted Shopee link. A link without a
// scheme ("shopee.co.th/…") is accepted.
func ParseLink(raw string) (Link, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Link{}, ErrNotShopee
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return Link{}, ErrNotShopee
	}
	host := strings.ToLower(u.Hostname())
	if !isShopeeHost(host) {
		return Link{}, ErrNotShopee
	}

	link := Link{URL: "https://" + host + u.EscapedPath()}

	var m []string
	if m = slugIDs.FindStringSubmatch(u.Path); m == nil {
		m = pathIDs.FindStringSubmatch(u.Path)
	}
	if m != nil {
		shop, err1 := strconv.ParseInt(m[1], 10, 64)
		item, err2 := strconv.ParseInt(m[2], 10, 64)
		if err1 == nil && err2 == nil && shop > 0 && item > 0 {
			link.ShopID, link.ItemID, link.HasIDs = shop, item, true
		}
	}
	return link, nil
}
