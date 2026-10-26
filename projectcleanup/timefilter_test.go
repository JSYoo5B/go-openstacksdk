package projectcleanup

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTimeFilterFollowsPythonISO8601Rules(t *testing.T) {
	cutoff := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		raw    string
		match  bool
		reason string
	}{
		{`"2025-12-31T23:59:59Z"`, true, ""},
		{`"2026-01-01T00:00:00Z"`, true, ""}, // equal to the cutoff matches (<=)
		{`"2026-01-01T00:00:00.000001"`, false, "after the cutoff"},
		{`"2026-01-01T08:00:00+09:00"`, true, ""}, // 2025-12-31T23:00Z
		{`"2025-12-31 23:00:00"`, true, ""},
		{`"2025-12-31"`, true, ""},
		{`"2026-02-30T00:00:00Z"`, false, "cannot be parsed"},
		{`"yesterday"`, false, "cannot be parsed"},
		{`17`, false, "not a time string"},
		{`null`, false, "not available"},
		{`""`, false, "not available"},
		{``, false, "not available"},
	}
	for _, tc := range cases {
		match, reason := timeMatches("created_at", json.RawMessage(tc.raw), cutoff)
		if match != tc.match || !strings.Contains(reason, tc.reason) {
			t.Errorf("%s: match=%v reason=%q", tc.raw, match, reason)
		}
	}
}
