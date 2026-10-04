package jmespath

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

// Ordering uses normalized decimal coefficients and arbitrary-size exponents.
// It never expands exponent-sized coefficients or converts them to float64.
type decimalNumber struct {
	negative bool
	digits   string
	exponent big.Int
}

func isNumber(value any) bool { _, ok := value.(json.Number); return ok }

func parseNumber(value json.Number) (decimalNumber, error) {
	text := value.String()
	if strings.TrimSpace(text) != text {
		return decimalNumber{}, fmt.Errorf("invalid JSON number %q", text)
	}
	decoder := json.NewDecoder(bytes.NewBufferString(text))
	decoder.UseNumber()
	var decoded any
	if !json.Valid([]byte(text)) {
		return decimalNumber{}, fmt.Errorf("invalid JSON number %q", text)
	}
	if err := decoder.Decode(&decoded); err != nil {
		return decimalNumber{}, err
	}
	if _, ok := decoded.(json.Number); !ok {
		return decimalNumber{}, fmt.Errorf("invalid JSON number %q", text)
	}
	var result decimalNumber
	if strings.HasPrefix(text, "-") {
		result.negative = true
		text = text[1:]
	}
	exponent := "0"
	if i := strings.IndexAny(text, "eE"); i >= 0 {
		exponent, text = text[i+1:], text[:i]
	}
	fraction := 0
	if i := strings.IndexByte(text, '.'); i >= 0 {
		fraction = len(text) - i - 1
		text = text[:i] + text[i+1:]
	}
	text = strings.TrimLeft(text, "0")
	if text == "" {
		return decimalNumber{digits: "0"}, nil
	}
	result.digits = strings.TrimRight(text, "0")
	result.exponent.SetString(exponent, 10)
	result.exponent.Add(&result.exponent, big.NewInt(int64(len(text)-len(result.digits)-fraction)))
	return result, nil
}

func compareNumbers(left, right json.Number) (int, error) {
	a, err := parseNumber(left)
	if err != nil {
		return 0, err
	}
	b, err := parseNumber(right)
	if err != nil {
		return 0, err
	}
	if a.negative != b.negative {
		if a.negative {
			return -1, nil
		}
		return 1, nil
	}
	comparison := 0
	if a.digits == "0" || b.digits == "0" {
		if a.digits != b.digits {
			if a.digits == "0" {
				comparison = -1
			} else {
				comparison = 1
			}
		}
	} else {
		var aMagnitude, bMagnitude big.Int
		aMagnitude.Add(&a.exponent, big.NewInt(int64(len(a.digits))))
		bMagnitude.Add(&b.exponent, big.NewInt(int64(len(b.digits))))
		comparison = aMagnitude.Cmp(&bMagnitude)
		if comparison == 0 {
			length := max(len(a.digits), len(b.digits))
			for i := 0; i < length; i++ {
				x, y := byte('0'), byte('0')
				if i < len(a.digits) {
					x = a.digits[i]
				}
				if i < len(b.digits) {
					y = b.digits[i]
				}
				if x < y {
					comparison = -1
					break
				}
				if x > y {
					comparison = 1
					break
				}
			}
		}
	}
	if a.negative {
		comparison = -comparison
	}
	return comparison, nil
}

func formatNumber(value decimalNumber) json.Number {
	if value.digits == "0" {
		return json.Number("0")
	}
	text := value.digits
	if value.exponent.IsInt64() {
		exponent := value.exponent.Int64()
		if exponent >= 0 && exponent <= int64(maxArithmeticCoefficientDigits-len(value.digits)) {
			text += strings.Repeat("0", int(exponent))
		} else if exponent < 0 && exponent >= -maxArithmeticCoefficientDigits {
			position := int64(len(value.digits)) + exponent
			if position > 0 {
				text = value.digits[:int(position)] + "." + value.digits[int(position):]
			} else {
				text = "0." + strings.Repeat("0", int(-position)) + value.digits
			}
		} else if value.exponent.Sign() != 0 {
			text += "e" + value.exponent.String()
		}
	} else {
		text += "e" + value.exponent.String()
	}
	if value.negative {
		text = "-" + text
	}
	return json.Number(text)
}

func numberToFloat(value json.Number) (float64, error) {
	if _, err := parseNumber(value); err != nil {
		return 0, err
	}
	result, err := strconv.ParseFloat(value.String(), 64)
	if err != nil || math.IsNaN(result) || math.IsInf(result, 0) {
		return 0, fmt.Errorf("number %q is outside finite IEEE-754 arithmetic", value)
	}
	return result, nil
}

func numberFromFloat(value float64) (json.Number, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "", fmt.Errorf("arithmetic produced a nonfinite JSON number")
	}
	return json.Number(strconv.FormatFloat(value, 'g', -1, 64)), nil
}

func absNumber(value json.Number) (json.Number, error) {
	parsed, err := parseNumber(value)
	if err != nil {
		return "", err
	}
	parsed.negative = false
	return formatNumber(parsed), nil
}

func roundNumber(value json.Number, ceiling bool) (json.Number, error) {
	parsed, err := parseNumber(value)
	if err != nil {
		return "", err
	}
	if parsed.exponent.Sign() >= 0 || parsed.digits == "0" {
		return formatNumber(parsed), nil
	}
	var position big.Int
	position.Add(&parsed.exponent, big.NewInt(int64(len(parsed.digits))))
	if position.Sign() <= 0 {
		if ceiling && !parsed.negative {
			return json.Number("1"), nil
		}
		if !ceiling && parsed.negative {
			return json.Number("-1"), nil
		}
		return json.Number("0"), nil
	}
	// A negative exponent puts the integer prefix strictly inside digits.
	integer := new(big.Int)
	integer.SetString(parsed.digits[:int(position.Int64())], 10)
	if ceiling && !parsed.negative || !ceiling && parsed.negative {
		integer.Add(integer, big.NewInt(1))
	}
	if parsed.negative {
		integer.Neg(integer)
	}
	return json.Number(integer.String()), nil
}

const maxArithmeticCoefficientDigits = 1048576

func addNumbers(values []json.Number) (json.Number, error) {
	if len(values) == 0 {
		return json.Number("0"), nil
	}
	parsed := make([]decimalNumber, len(values))
	allIntegral := true
	for i, value := range values {
		var err error
		parsed[i], err = parseNumber(value)
		if err != nil {
			return "", err
		}
		if parsed[i].exponent.Sign() < 0 {
			allIntegral = false
		}
	}
	if !allIntegral {
		sum := 0.0
		for _, value := range values {
			n, err := numberToFloat(value)
			if err != nil {
				return "", err
			}
			sum += n
			if math.IsInf(sum, 0) || math.IsNaN(sum) {
				return "", fmt.Errorf("sum produced a nonfinite JSON number")
			}
		}
		return numberFromFloat(sum)
	}
	var exponent big.Int
	// Zero contributes no coefficient and need not force a compact integer
	// with a huge exponent into an expanded decimal representation.
	first := true
	for i := range parsed {
		if parsed[i].digits != "0" {
			if first || parsed[i].exponent.Cmp(&exponent) < 0 {
				exponent.Set(&parsed[i].exponent)
			}
			first = false
		}
	}
	if first {
		return json.Number("0"), nil
	}
	sum := new(big.Int)
	for i := range parsed {
		if parsed[i].digits == "0" {
			continue
		}
		var shift big.Int
		shift.Sub(&parsed[i].exponent, &exponent)
		remaining := maxArithmeticCoefficientDigits - len(parsed[i].digits)
		if remaining < 0 || !shift.IsInt64() || shift.Cmp(big.NewInt(int64(remaining))) > 0 {
			return "", fmt.Errorf("integer arithmetic coefficient exceeds %d decimal digits", maxArithmeticCoefficientDigits)
		}
		coefficient := new(big.Int)
		coefficient.SetString(parsed[i].digits+strings.Repeat("0", int(shift.Int64())), 10)
		if parsed[i].negative {
			coefficient.Neg(coefficient)
		}
		sum.Add(sum, coefficient)
	}
	text := sum.String()
	if len(strings.TrimPrefix(text, "-")) > maxArithmeticCoefficientDigits {
		return "", fmt.Errorf("integer arithmetic coefficient exceeds %d decimal digits", maxArithmeticCoefficientDigits)
	}
	if sum.Sign() == 0 {
		return json.Number("0"), nil
	}
	if exponent.Sign() == 0 {
		return json.Number(text), nil
	}
	return json.Number(text + "e" + exponent.String()), nil
}
