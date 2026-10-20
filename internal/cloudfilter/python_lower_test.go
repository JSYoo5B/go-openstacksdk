package cloudfilter

import (
	"strings"
	"testing"
)

// Fixed expected values exercise full lowercase and original-string context;
// neither the Go Unicode package nor Python is used as the test oracle.
func TestPythonLowerFullUnicode16AndOriginalContext(t *testing.T) {
	for _, test := range []struct{ name, value, want string }{
		{"empty", "", ""},
		{"ASCII exact spelling", " AcTiVe ERROR\u0009", " active error\u0009"},
		{"NUL stays present", "A\u0000B", "a\u0000b"},
		{"dotted I expands", "\u0130I\u0131i", "i\u0307i\u0131i"},
		{"no normalization or Turkic dot removal", "I\u0307", "i\u0307"},
		{"Latin titlecase and sharp S", "\u00C0\u00C9\u01C5\u1E9E", "\u00E0\u00E9\u01C6\u00DF"},
		{"standalone sigma", "\u03A3", "\u03C3"},
		{"final sigma", "\u0391\u03A3", "\u03B1\u03C2"},
		{"medial sigma", "\u0391\u03A3\u0391", "\u03B1\u03C3\u03B1"},
		{"repeated sigma context", "\u03A3\u03A3\u03A3", "\u03C3\u03C3\u03C2"},
		{"titlecase is cased", "\u01C5\u03A3", "\u01C6\u03C2"},
		{"Greek with accents", "\u1F48\u0394\u03A5\u03A3\u03A3\u0395\u038E\u03A3", "\u1F40\u03B4\u03C5\u03C3\u03C3\u03B5\u03CD\u03C2"},
		{"final sigma before combining mark", "\u0391\u03A3\u0301", "\u03B1\u03C2\u0301"},
		{"medial sigma across combining mark", "\u0391\u03A3\u0301\u0391", "\u03B1\u03C3\u0301\u03B1"},
		{"ignorable apostrophe and marks on both sides", "A'\u0301\u03A3'\u0301", "a'\u0301\u03C2'\u0301"},
		{"cased after apostrophe is medial", "A\u03A3'B", "a\u03C3'b"},
		{"ignorable without preceding cased", "'\u0301\u03A3", "'\u0301\u03C3"},
		{"noncased boundary stops lookback", "A1\u03A3", "a1\u03C3"},
		{"noncased boundary stops lookahead", "A\u03A31B", "a\u03C21b"},
		{"cased ignorable modifier is ignored", "A\u03A3\u02B0", "a\u03C2\u02B0"},
		{"cased ignorable modifier before real cased", "A\u03A3\u02B0B", "a\u03C3\u02B0b"},
		{"cased ignorable alone is not preceding cased", "\u02B0\u03A3", "\u02B0\u03C3"},
		{"supplementary Deseret", "\U00010400\U00010427", "\U00010428\U0001044F"},
		{"supplementary Osage", "\U000104B0", "\U000104D8"},
		{"noncased text unchanged", "\u7A7A \U0001F600 \u0301\U0010FFFF", "\u7A7A \U0001F600 \u0301\U0010FFFF"},
		{"newer codepoint remains unchanged", "\uA7CE", "\uA7CE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := PythonLower(test.value); got != test.want {
				t.Fatalf("PythonLower(%q)=%q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestPythonLowerLongIgnoredContextAndRepeatedSigma(t *testing.T) {
	ignored := strings.Repeat("'\u0301", 10000)
	for _, test := range []struct{ value, want string }{
		{"A" + ignored + "\u03A3", "a" + ignored + "\u03C2"},
		{"A\u03A3" + ignored + "B", "a\u03C3" + ignored + "b"},
		{strings.Repeat("\u03A3", 10000), strings.Repeat("\u03C3", 9999) + "\u03C2"},
	} {
		if got := PythonLower(test.value); got != test.want {
			t.Fatalf("long context mismatch: got length %d, want length %d", len(got), len(test.want))
		}
	}
}
