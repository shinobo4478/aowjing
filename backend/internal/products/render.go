package products

import (
	"fmt"
	"regexp"
	"strings"
)

// Prompt templates can mention the products chosen when you press Generate:
//
//	{{product.name}}   {{product.price}}   {{product.note}}
//	{{product2.name}}  …                      (the second product)
//
// "product" and "product1" both mean the first product. The price renders as
// "฿1,890" (empty when the product has no price). Anything else in double
// braces — including {{product3.name}} — is left exactly as written, so a
// template can still contain braces on purpose.
var varPattern = regexp.MustCompile(`\{\{\s*(product[12]?)\.(name|price|note)\s*\}\}`)

// Item is the part of a product a template can use.
type Item struct {
	Name     string
	PriceTHB int
	Note     string
}

// slot turns a variable's object name into a 1-based product position.
func slot(object string) int {
	if object == "product2" {
		return 2
	}
	return 1
}

// SlotsNeeded reports how many products a template body needs chosen: the
// highest product number it mentions, or 0 if it mentions none.
func SlotsNeeded(body string) int {
	need := 0
	for _, m := range varPattern.FindAllStringSubmatch(body, -1) {
		if s := slot(m[1]); s > need {
			need = s
		}
	}
	return need
}

// MissingProductError means the template refers to a product that wasn't
// chosen. Its message is shown to the user as is.
type MissingProductError struct {
	Variable string // as written, e.g. "{{product2.name}}"
	Chosen   int
}

func (e *MissingProductError) Error() string {
	if e.Chosen == 0 {
		return fmt.Sprintf("The template uses %s. Choose a product to generate it.", e.Variable)
	}
	return fmt.Sprintf("The template uses %s, but only %d product(s) were chosen.", e.Variable, e.Chosen)
}

// RenderTemplate fills a template's product variables from the chosen
// products (in pin order). A template with no product variables comes back
// unchanged whatever is chosen.
func RenderTemplate(body string, items []Item) (string, error) {
	var missing *MissingProductError
	out := varPattern.ReplaceAllStringFunc(body, func(match string) string {
		m := varPattern.FindStringSubmatch(match)
		s := slot(m[1])
		if s > len(items) {
			if missing == nil {
				missing = &MissingProductError{Variable: strings.TrimSpace(match), Chosen: len(items)}
			}
			return match
		}
		it := items[s-1]
		switch m[2] {
		case "name":
			return it.Name
		case "price":
			if it.PriceTHB > 0 {
				return "฿" + Commas(it.PriceTHB)
			}
			return ""
		default: // note
			return it.Note
		}
	})
	if missing != nil {
		return "", missing
	}
	return out, nil
}
