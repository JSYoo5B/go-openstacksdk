package cloudsnapshot

import (
	"strings"
	"testing"
)

func TestBackupImportVersionIndependentStringAndHeaderCases(t *testing.T) {
	cases := []struct {
		input                   string
		wantParts               []string
		wantHeader              string
		parseError, headerError bool
	}{
		{input: "3", wantParts: []string{"3", "0"}, wantHeader: "3.0"},
		{input: "v3", wantParts: []string{"3", "0"}, wantHeader: "3.0"},
		{input: "vv3.064", wantParts: []string{"3", "64"}, wantHeader: "3.64"},
		{input: "3.64.0", wantParts: []string{"3", "64", "0"}, wantHeader: "3.64.0"},
		{input: "+3.-1", wantParts: []string{"3", "-1"}, wantHeader: "3.-1"},
		{input: " 3 . +064 ", wantParts: []string{"3", "64"}, wantHeader: "3.64"},
		{input: "3.6_4", wantParts: []string{"3", "64"}, wantHeader: "3.64"},
		{input: "٣.٦٤", wantParts: []string{"3", "64"}, wantHeader: "3.64"},
		{input: "latest", wantParts: []string{"latest", "latest"}, wantHeader: "latest"},
		{input: "vlatest", wantParts: []string{"latest", "latest"}, wantHeader: "latest"},
		{input: "3.latest", wantParts: []string{"3", "latest"}, headerError: true},
		{input: "latest.3", wantParts: []string{"latest", "3"}, wantHeader: "latest.3"},
		{input: "", parseError: true},
		{input: "v", parseError: true},
		{input: "V3.64", parseError: true},
		{input: "LATEST", parseError: true},
		{input: "3.", parseError: true},
		{input: "3..64", parseError: true},
		{input: " v3.64", parseError: true},
		{input: "3. latest", parseError: true},
		{input: "3.6__4", parseError: true},
		{input: "3.0x40", parseError: true},
		{input: "3.6e1", parseError: true},
		{input: "3.9999999999999999999999999999999999999999", wantParts: []string{"3", "9999999999999999999999999999999999999999"}, wantHeader: "3.9999999999999999999999999999999999999999"},
		{input: "3.²", parseError: true},
		{input: "3.⑥", parseError: true},
		{input: "3.6_", parseError: true},
		{input: "3._64", parseError: true},
		{input: "3.+ 64", parseError: true},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			value, err := parseBackupImportVersion(tc.input)
			if tc.parseError {
				if err == nil {
					t.Fatalf("parse %q unexpectedly succeeded: %#v", tc.input, value)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse %q: %v", tc.input, err)
			}
			if len(value) != len(tc.wantParts) {
				t.Fatalf("parts=%d want %d", len(value), len(tc.wantParts))
			}
			for i, want := range tc.wantParts {
				got := "latest"
				if value[i] != nil {
					got = value[i].String()
				}
				if got != want {
					t.Errorf("part[%d]=%q want %q", i, got, want)
				}
			}
			got, err := value.header()
			if tc.headerError {
				if err == nil {
					t.Fatalf("header %q unexpectedly succeeded: %q", tc.input, got)
				}
				return
			}
			if err != nil || got != tc.wantHeader {
				t.Fatalf("header=%q err=%v want=%q", got, err, tc.wantHeader)
			}
		})
	}
}

func TestBackupImportVersionIndependentMaximumMinimumCases(t *testing.T) {
	cases := []struct {
		name, maximum, minimum, want string
		errorText                    string
	}{
		{name: "major-only", maximum: "3", want: "3.0"},
		{name: "latest-is-capped", maximum: "latest", want: "3.64"},
		{name: "finite-latest-is-capped", maximum: "3.latest", want: "3.64"},
		{name: "longer-equal-prefix-max-is-capped", maximum: "3.64.0", want: "3.64"},
		{name: "longer-equal-prefix-min-is-too-high", maximum: "3.64", minimum: "3.64.0"},
		{name: "finite-latest-min-is-too-high", maximum: "latest", minimum: "3.latest"},
		{name: "latest-min-is-too-high", maximum: "latest", minimum: "latest"},
		{name: "selected-later-latest-fails-header", maximum: "3.63.latest", errorText: "latest"},
		{name: "cross-major-later-latest-fails-header", maximum: "2.latest", errorText: "latest"},
		{name: "maximum-error-precedes-too-high-minimum", maximum: "bad", minimum: "9.0", errorText: "bad"},
		{name: "missing-maximum-ignores-malformed-minimum", minimum: "bad"},
		{name: "utility-does-not-check-inverted-range", maximum: "3.20", minimum: "3.60", want: "3.20"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := selectBackupImportVersion(tc.maximum, tc.minimum)
			if tc.errorText != "" {
				if err == nil || !strings.Contains(err.Error(), tc.errorText) {
					t.Fatalf("selection=%q err=%v want error containing %q", got, err, tc.errorText)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("selection=%q err=%v want=%q", got, err, tc.want)
			}
		})
	}
}
