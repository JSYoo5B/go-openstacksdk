package jsonfilter

import (
	"fmt"
	"math/big"
	"strings"
	"unicode/utf8"
)

// PythonIntegerString parses the decimal string form used by version tuples.
// Unlike a Resource int descriptor, signs, surrounding space and digit
// separators are accepted. Decimal digits use the shared Unicode 16 table.
func PythonIntegerString(value string) (*big.Int, error) {
	if !utf8.ValidString(value) {
		return nil, fmt.Errorf("integer string must be UTF-8")
	}
	text := strings.TrimSpace(value)
	sign := ""
	if strings.HasPrefix(text, "+") || strings.HasPrefix(text, "-") {
		sign, text = text[:1], text[1:]
	}
	var digits strings.Builder
	previousDigit := false
	for _, character := range text {
		if character == '_' && previousDigit {
			previousDigit = false
			continue
		}
		digit, ok := descriptorDecimalDigit(character)
		if !ok {
			return nil, fmt.Errorf("invalid decimal integer %q", value)
		}
		digits.WriteByte('0' + byte(digit))
		previousDigit = true
	}
	if !previousDigit {
		return nil, fmt.Errorf("invalid decimal integer %q", value)
	}
	integer, ok := new(big.Int).SetString(sign+digits.String(), 10)
	if !ok {
		return nil, fmt.Errorf("invalid decimal integer %q", value)
	}
	return integer, nil
}
