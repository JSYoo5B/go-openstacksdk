package cloudfilter

import "strings"

// PythonLower implements locale-independent str.lower for valid UTF-8 strings
// with pinned Unicode 16.0.0 full mappings and original-string Final_Sigma
// context. It performs no normalization. Codepoints absent from the pinned
// tables stay unchanged; it does not depend on the Go toolchain's Unicode data.
func PythonLower(value string) string {
	runes := []rune(value)
	// Compute the following non-ignorable character once. Two passes avoid
	// rescanning long combining-mark spans for each sigma.
	followingCased := make([]bool, len(runes))
	next := false
	for at := len(runes) - 1; at >= 0; at-- {
		followingCased[at] = next
		if !pythonLowerHas(runes[at], pythonLowerCaseIgnorable[:]) {
			next = pythonLowerHas(runes[at], pythonLowerCased[:])
		}
	}
	var result strings.Builder
	result.Grow(len(value))
	previousCased := false
	for at, code := range runes {
		if code == '\u03A3' && previousCased && !followingCased[at] {
			result.WriteRune('\u03C2')
		} else if mapped, ok := pythonLowerMapped(code); ok {
			result.WriteString(mapped)
		} else {
			result.WriteRune(code)
		}
		// Case_Ignorable takes precedence when a character is also Cased.
		if !pythonLowerHas(code, pythonLowerCaseIgnorable[:]) {
			previousCased = pythonLowerHas(code, pythonLowerCased[:])
		}
	}
	return result.String()
}

func pythonLowerMapped(code rune) (string, bool) {
	lo, hi := 0, len(pythonLowerMappings)
	for lo < hi {
		middle := lo + (hi-lo)/2
		if pythonLowerMappings[middle].upper < code {
			lo = middle + 1
		} else {
			hi = middle
		}
	}
	if lo < len(pythonLowerMappings) && pythonLowerMappings[lo].upper == code {
		return pythonLowerMappings[lo].lower, true
	}
	return "", false
}

func pythonLowerHas(code rune, ranges []pythonLowerRange) bool {
	lo, hi := 0, len(ranges)
	for lo < hi {
		middle := lo + (hi-lo)/2
		if ranges[middle].last < code {
			lo = middle + 1
		} else {
			hi = middle
		}
	}
	return lo < len(ranges) && ranges[lo].first <= code
}
