package cloudsnapshot

import (
	"fmt"
	"math/big"
	"strings"

	"gophercloudsdk/internal/jsonfilter"
)

// A nil component denotes the positive-infinity 'latest' token. Tuples keep
// their length: (3,64) compares below (3,64,0), as in the Source utility.
type backupImportVersion []*big.Int

func parseBackupImportVersion(text string) (backupImportVersion, error) {
	text = strings.TrimLeft(text, "v")
	if text == "latest" {
		return backupImportVersion{nil, nil}, nil
	}
	parts := strings.Split(text, ".")
	if len(parts) == 1 {
		parts = append(parts, "0")
	}
	value := make(backupImportVersion, len(parts))
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

func (v backupImportVersion) compare(other backupImportVersion) int {
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

func (v backupImportVersion) header() (string, error) {
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

func selectBackupImportVersion(maximum, minimum string) (string, error) {
	if maximum == "" {
		return "", nil
	}
	cap, _ := parseBackupImportVersion("3.64")
	high, err := parseBackupImportVersion(maximum)
	if err != nil {
		return "", err
	}
	if minimum != "" {
		low, err := parseBackupImportVersion(minimum)
		if err != nil {
			return "", err
		}
		if cap.compare(low) < 0 {
			return "", nil
		}
	}
	if cap.compare(high) < 0 {
		high = cap
	}
	return high.header()
}
