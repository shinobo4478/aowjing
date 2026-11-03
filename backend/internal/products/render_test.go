package products

import (
	"errors"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	strip := Item{Name: "Power strip", PriceTHB: 1890, Note: "4 outlets"}
	stick := Item{Name: "ปลั๊กพ่วง", PriceTHB: 0, Note: ""}

	tests := []struct {
		name  string
		body  string
		items []Item
		want  string
	}{
		{"no variables, nothing chosen", "A plain prompt.", nil, "A plain prompt."},
		{"no variables, products chosen anyway", "A plain prompt.", []Item{strip}, "A plain prompt."},
		{"name", "Review the {{product.name}}.", []Item{strip}, "Review the Power strip."},
		{"product1 is the same as product", "{{product1.name}}={{product.name}}", []Item{strip}, "Power strip=Power strip"},
		{"price is formatted", "Only {{product.price}}!", []Item{strip}, "Only ฿1,890!"},
		{"price empty when not set", "Only {{product.price}}!", []Item{stick}, "Only !"},
		{"note", "({{product.note}})", []Item{strip}, "(4 outlets)"},
		{"two products", "{{product.name}} vs {{product2.name}}", []Item{strip, stick}, "Power strip vs ปลั๊กพ่วง"},
		{"extra products are ignored", "{{product.name}}", []Item{strip, stick}, "Power strip"},
		{"repeated variable", "{{product.name}}, again {{product.name}}", []Item{strip}, "Power strip, again Power strip"},
		{"spaces inside the braces", "{{ product.name }}|{{product.name  }}", []Item{strip}, "Power strip|Power strip"},
		{"unknown variables are left alone", "{{product3.name}} {{foo}} {{product.size}}", []Item{strip}, "{{product3.name}} {{foo}} {{product.size}}"},
		{"Thai text around it", "รีวิว {{product.name}} หน่อย", []Item{stick}, "รีวิว ปลั๊กพ่วง หน่อย"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderTemplate(tt.body, tt.items)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderTemplateMissingProduct(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		items    []Item
		wantVar  string
		wantHave int
	}{
		{"none chosen", "About {{product.name}}", nil, "{{product.name}}", 0},
		{"second missing", "{{product.name}} and {{product2.price}}", []Item{{Name: "A"}}, "{{product2.price}}", 1},
		{"reports the first missing one", "{{product2.name}} {{product2.note}}", []Item{{Name: "A"}}, "{{product2.name}}", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RenderTemplate(tt.body, tt.items)
			var me *MissingProductError
			if !errors.As(err, &me) {
				t.Fatalf("err = %v, want *MissingProductError", err)
			}
			if me.Variable != tt.wantVar || me.Chosen != tt.wantHave {
				t.Errorf("got %+v, want var %q chosen %d", me, tt.wantVar, tt.wantHave)
			}
		})
	}
}

func TestSlotsNeeded(t *testing.T) {
	for body, want := range map[string]int{
		"":                                   0,
		"nothing here":                       0,
		"{{foo}} {{product3.name}}":          0,
		"{{product.name}}":                   1,
		"{{product1.price}}":                 1,
		"{{product2.name}}":                  2,
		"{{product.name}} {{product2.note}}": 2,
	} {
		if got := SlotsNeeded(body); got != want {
			t.Errorf("SlotsNeeded(%q) = %d, want %d", body, got, want)
		}
	}
}
