package cloudfilter

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// pythonString covers ordinary JSON values, including Python spellings of
// null/booleans and insertion-ordered container repr. Float spelling uses Go's
// shortest float64 formatting with Python's fixed/scientific cutoff; exact
// Python float repr and Unicode database versions remain representation bounds.
func pythonString(n *node, guard func() error) (string, error) {
	if err := check(guard); err != nil {
		return "", err
	}
	if n.kind == '"' {
		return n.text, nil
	}
	return pythonRepr(n, guard)
}

func pythonRepr(n *node, guard func() error) (string, error) {
	if err := check(guard); err != nil {
		return "", err
	}
	switch n.kind {
	case 'n':
		return "None", nil
	case 't':
		return "True", nil
	case 'f':
		return "False", nil
	case '"':
		return quotePython(n.text), nil
	case '[':
		parts := make([]string, len(n.items))
		for i, item := range n.items {
			value, err := pythonRepr(item, guard)
			if err != nil {
				return "", err
			}
			parts[i] = value
		}
		return "[" + strings.Join(parts, ", ") + "]", nil
	case '{':
		parts := make([]string, len(n.members))
		for i, item := range n.members {
			value, err := pythonRepr(item.value, guard)
			if err != nil {
				return "", err
			}
			parts[i] = quotePython(item.key) + ": " + value
		}
		return "{" + strings.Join(parts, ", ") + "}", nil
	default:
		literal := string(n.raw)
		if !strings.ContainsAny(literal, ".eE") {
			if literal == "-0" {
				return "0", nil
			}
			return literal, nil
		}
		return pythonFloat(literal)
	}
}

func pythonFloat(literal string) (string, error) {
	value, err := strconv.ParseFloat(literal, 64)
	if err != nil {
		if numeric, ok := err.(*strconv.NumError); !ok || numeric.Err != strconv.ErrRange {
			return "", fmt.Errorf("identifier number: %w", err)
		}
	}
	if math.IsInf(value, 1) {
		return "inf", nil
	}
	if math.IsInf(value, -1) {
		return "-inf", nil
	}
	if value == 0 {
		if math.Signbit(value) {
			return "-0.0", nil
		}
		return "0.0", nil
	}
	scientific := strconv.FormatFloat(value, 'e', -1, 64)
	position := strings.LastIndexByte(scientific, 'e')
	exponent, err := strconv.Atoi(scientific[position+1:])
	if err != nil {
		return "", err
	}
	if exponent >= -4 && exponent < 16 {
		fixed := strconv.FormatFloat(value, 'f', -1, 64)
		if !strings.ContainsRune(fixed, '.') {
			fixed += ".0"
		}
		return fixed, nil
	}
	return scientific, nil
}

func quotePython(value string) string {
	quote := '\''
	if strings.ContainsRune(value, '\'') && !strings.ContainsRune(value, '"') {
		quote = '"'
	}
	var out strings.Builder
	out.WriteRune(quote)
	for _, c := range value {
		switch c {
		case '\\':
			out.WriteString("\\\\")
		case '\t':
			out.WriteString("\\t")
		case '\n':
			out.WriteString("\\n")
		case '\r':
			out.WriteString("\\r")
		default:
			if c == quote {
				out.WriteByte('\\')
				out.WriteRune(c)
			} else if c == ' ' || unicode.IsGraphic(c) && !unicode.Is(unicode.Zs, c) {
				out.WriteRune(c)
			} else if c <= 0xff {
				fmt.Fprintf(&out, "\\x%02x", c)
			} else if c <= 0xffff {
				fmt.Fprintf(&out, "\\u%04x", c)
			} else {
				fmt.Fprintf(&out, "\\U%08x", c)
			}
		}
	}
	out.WriteRune(quote)
	return out.String()
}
