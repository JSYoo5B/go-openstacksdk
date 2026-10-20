package cloudfilter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// PythonJSONTruthy uses CPython JSON float truthiness for decimal/exponent
// tokens and preserves arbitrary integer truthiness. PythonTruthy retains the
// separately reviewed exact-decimal policy for existing callers.
func PythonJSONTruthy(raw json.RawMessage) (bool, error) {
	if raw == nil {
		return false, nil
	}
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return false, fmt.Errorf("truthiness requires complete UTF-8 JSON")
	}
	token := bytes.TrimSpace(raw)
	if (token[0] == '-' || token[0] >= '0' && token[0] <= '9') && strings.ContainsAny(string(token), ".eE") {
		number, err := strconv.ParseFloat(string(token), 64)
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			return false, err
		}
		return number != 0, nil
	}
	return PythonTruthy(raw)
}
