// Package microversions shares source-audited version comparisons and finite
// operation-owned discovery without changing a shared service client's policy.
package microversions

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
)

// A nil component denotes the positive-infinity 'latest' token. Tuples keep
// their length: (3,64) compares below (3,64,0), as in the Source utility.
type Version []*big.Int

func ParseVersion(text string) (Version, error) {
	text = strings.TrimLeft(text, "v")
	if text == "latest" {
		return Version{nil, nil}, nil
	}
	parts := strings.Split(text, ".")
	if len(parts) == 1 {
		parts = append(parts, "0")
	}
	value := make(Version, len(parts))
	for index, part := range parts {
		if part == "latest" {
			continue
		}
		integer, err := jsonfilter.PythonIntegerString(part)
		if err != nil {
			return nil, fmt.Errorf("invalid advertised version %q: %w", text, err)
		}
		value[index] = integer
	}
	return value, nil
}

func (v Version) Compare(other Version) int {
	for index := 0; index < len(v) && index < len(other); index++ {
		a, b := v[index], other[index]
		if a == nil && b != nil {
			return 1
		}
		if a != nil && b == nil {
			return -1
		}
		if a != nil && a.Cmp(b) != 0 {
			return a.Cmp(b)
		}
	}
	if len(v) < len(other) {
		return -1
	}
	if len(v) > len(other) {
		return 1
	}
	return 0
}

func (v Version) Header() (string, error) {
	parts := make([]string, len(v))
	allLatest := true
	for index, part := range v {
		if part == nil {
			if index > 0 && v[0] != nil {
				return "", invalid("a finite major microversion cannot contain latest")
			}
			parts[index] = "latest"
		} else {
			allLatest = false
			parts[index] = part.String()
		}
	}
	if allLatest {
		return "latest", nil
	}
	return strings.Join(parts, "."), nil
}

func SelectVersion(maximum, minimum, ceiling string) (string, error) {
	if maximum == "" {
		return "", nil
	}
	cap, err := ParseVersion(ceiling)
	if err != nil {
		return "", err
	}
	high, err := ParseVersion(maximum)
	if err != nil {
		return "", err
	}
	if minimum != "" {
		low, err := ParseVersion(minimum)
		if err != nil {
			return "", err
		}
		if cap.Compare(low) < 0 {
			return "", nil
		}
	}
	if cap.Compare(high) < 0 {
		high = cap
	}
	return high.Header()
}

// Matches follows Keystoneauth version_match: the selected version must have
// the same finite major and be at least the required tuple. Global latest is
// not a finite-major match; a finite major with latest minor can match.
func Matches(selected, required string) (bool, error) {
	want, err := ParseVersion(required)
	if err != nil {
		return false, err
	}
	candidate, err := ParseVersion(selected)
	if err != nil {
		return false, err
	}
	return matches(candidate, want), nil
}

func matches(candidate, want Version) bool {
	return candidate[0] != nil && want[0] != nil && candidate[0].Cmp(want[0]) == 0 && candidate.Compare(want) >= 0
}

// Supports follows supports_microversion, including its separate advertised
// range and selected-default tests. Missing either bound means false, not an
// error. It does not clamp a selected version to the advertised maximum.
func Supports(maximum, minimum, selected, required string) (bool, error) {
	if maximum == "" || minimum == "" {
		return false, nil
	}
	want, err := ParseVersion(required)
	if err != nil {
		return false, err
	}
	low, err := ParseVersion(minimum)
	if err != nil {
		return false, err
	}
	high, err := ParseVersion(maximum)
	if err != nil {
		return false, err
	}
	if want.Compare(low) < 0 || want.Compare(high) > 0 {
		return false, nil
	}
	if selected == "" {
		return true, nil
	}
	candidate, err := ParseVersion(selected)
	if err != nil {
		return false, err
	}
	return matches(candidate, want), nil
}
