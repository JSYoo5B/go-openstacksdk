package jsonfilter

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// DescriptorFloatJSON applies the source float descriptor to a response value.
// Null stays null, booleans become numbers, and nonnumeric containers become
// zero. Every string is converted with Python's decimal float grammar; an
// invalid string is an error rather than zero. Returned numbers are finite
// IEEE-754 values, because nonfinite Python floats have no JSON representation.
// Caller filters and the original response bytes are never modified.
func DescriptorFloatJSON(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.RawMessage("null"), nil
	}
	if !utf8.Valid(raw) {
		return nil, fmt.Errorf("descriptor float expects UTF-8 JSON")
	}
	value, err := filterJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("descriptor float JSON: %w", err)
	}
	var text string
	switch value := value.(type) {
	case nil:
		return json.RawMessage("null"), nil
	case bool:
		if value {
			return json.RawMessage("1"), nil
		}
		return json.RawMessage("0"), nil
	case json.Number:
		text = value.String()
		// Python's JSON decoder makes plain -0 an int(0), while fractional
		// or exponent forms are floats and retain their negative zero.
		if text == "-0" {
			text = "0"
		}
	case string:
		// fields._convert_type's float string condition is a generator
		// object, so even an empty or nonnumeric string reaches float().
		text = strings.TrimSpace(value)
		if text == "" {
			return nil, fmt.Errorf("descriptor float string is empty")
		}
		unsigned := text
		if unsigned[0] == '+' || unsigned[0] == '-' {
			unsigned = unsigned[1:]
		}
		if strings.EqualFold(unsigned, "nan") || strings.EqualFold(unsigned, "inf") || strings.EqualFold(unsigned, "infinity") {
			return nil, fmt.Errorf("descriptor float is nonfinite")
		}
		var decimal strings.Builder
		decimal.Grow(len(text))
		for _, character := range text {
			if digit, ok := descriptorDecimalDigit(character); ok {
				decimal.WriteByte('0' + byte(digit))
				continue
			}
			switch character {
			case '+', '-', '.', 'e', 'E', '_':
				decimal.WriteByte(byte(character))
			default:
				return nil, fmt.Errorf("descriptor float string is not decimal")
			}
		}
		// ParseFloat validates sign, exponent and underscore placement.
		// The alphabet above excludes Go's additional hex-float grammar.
		text = decimal.String()
	default:
		return json.RawMessage("0"), nil
	}
	floating, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return nil, fmt.Errorf("descriptor float conversion: %w", err)
	}
	if math.IsNaN(floating) || math.IsInf(floating, 0) {
		return nil, fmt.Errorf("descriptor float is nonfinite")
	}
	return json.RawMessage(strconv.FormatFloat(floating, 'g', -1, 64)), nil
}
