package products

import "fmt"

// Commas formats a whole number of baht with thousands separators: 1890 →
// "1,890". Shared by the description block and prompt-template variables so a
// price always reads the same.
func Commas(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
