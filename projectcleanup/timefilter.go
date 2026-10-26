package projectcleanup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// isoTime accepts the ISO 8601 forms that Python's iso8601.parse_date reads
// from OpenStack APIs: a calendar date, optionally followed by T or a space and
// hours:minutes, optional seconds and fraction, and an optional Z or numeric
// offset. A value without an offset is UTC, as in iso8601's default.
var isoTime = regexp.MustCompile(`^(\d{4})-?(\d{2})-?(\d{2})(?:[T ](\d{2}):?(\d{2})(?::?(\d{2})(?:[.,](\d+))?)?)?\s*(Z|z|[+-]\d{2}(?::?\d{2})?)?$`)

func parseISOTime(value string) (time.Time, error) {
	m := isoTime.FindStringSubmatch(value)
	if m == nil {
		return time.Time{}, fmt.Errorf("unsupported time %q", value)
	}
	num := func(s string) int {
		if s == "" {
			return 0
		}
		n, _ := strconv.Atoi(s)
		return n
	}
	nanos := 0
	if m[7] != "" {
		frac := m[7]
		if len(frac) > 9 {
			frac = frac[:9]
		}
		frac += strings.Repeat("0", 9-len(frac))
		nanos = num(frac)
	}
	location := time.UTC
	if zone := m[8]; zone != "" && zone != "Z" && zone != "z" {
		digits := strings.ReplaceAll(zone[1:], ":", "")
		offset := num(digits[:2]) * 3600
		if len(digits) == 4 {
			offset += num(digits[2:]) * 60
		}
		if zone[0] == '-' {
			offset = -offset
		}
		location = time.FixedZone(zone, offset)
	}
	year, month, day := num(m[1]), num(m[2]), num(m[3])
	hour, minute, second := num(m[4]), num(m[5]), num(m[6])
	t := time.Date(year, time.Month(month), day, hour, minute, second, nanos, location)
	if t.Year() != year || int(t.Month()) != month || t.Day() != day || t.Hour() != hour || t.Minute() != minute || t.Second() != second {
		return time.Time{}, fmt.Errorf("invalid time %q", value)
	}
	return t, nil
}

// timeMatches applies Python's filter rule: a value at or before the cutoff
// matches. ok is false with a reason when the resource must be excluded.
func timeMatches(attribute string, raw json.RawMessage, cutoff time.Time) (bool, string) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || bytes.Equal(trimmed, []byte(`""`)) {
		return false, attribute + " is not available"
	}
	var text string
	if err := json.Unmarshal(trimmed, &text); err != nil {
		return false, fmt.Sprintf("%s %s is not a time string", attribute, trimmed)
	}
	value, err := parseISOTime(text)
	if err != nil {
		return false, fmt.Sprintf("%s %q cannot be parsed", attribute, text)
	}
	if value.After(cutoff) {
		return false, fmt.Sprintf("%s %s is after the cutoff", attribute, text)
	}
	return true, ""
}
