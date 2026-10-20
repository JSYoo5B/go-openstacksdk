package microversions

import "testing"

// Source utils.supports_microversion and Keystoneauth version_match have
// separate advertisement and selected-default tests; neither is negotiation.
func TestSupportsAdvertisedBoundsAndSelectedVersionAreSeparate(t *testing.T) {
	for _, tc := range []struct {
		name, maximum, minimum, selected, required string
		want, failure                              bool
	}{
		{"unselected advertised", "2.110", "2.1", "", "2.6", true, false},
		{"selected ceiling", "2.110", "2.1", "2.5", "2.6", false, false},
		{"selected does not grant advertisement", "2.5", "2.1", "2.100", "2.6", false, false},
		{"selected above cloud max is not clamped", "2.99", "2.1", "2.100", "2.6", true, false},
		{"advertised minimum above requirement", "2.110", "2.8", "", "2.6", false, false},
		{"missing maximum ignores selected parse", "", "2.1", "invalid", "2.6", false, false},
		{"missing minimum", "2.110", "", "2.100", "2.6", false, false},
		{"unusable range ignores selected parse", "2.5", "2.1", "invalid", "2.6", false, false},
		{"different selected major", "2.110", "2.1", "3.100", "2.6", false, false},
		{"global latest is not finite major", "2.110", "2.1", "latest", "2.6", false, false},
		{"finite major latest minor", "2.110", "2.1", "2.latest", "2.6", true, false},
		{"tuple suffix meets shorter requirement", "2.110", "2.1", "2.6.0", "2.6", true, false},
		{"tuple suffix requirement is higher", "2.110", "2.1", "2.6", "2.6.0", false, false},
		{"bad maximum", "bad", "2.1", "", "2.6", false, true},
		{"bad minimum", "2.110", "bad", "", "2.6", false, true},
		{"bad selected when range permits", "2.110", "2.1", "bad", "2.6", false, true},
		{"bad required", "2.110", "2.1", "", "bad", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Supports(tc.maximum, tc.minimum, tc.selected, tc.required)
			if got != tc.want || (err != nil) != tc.failure {
				t.Fatal(got, err, tc.want, tc.failure)
			}
		})
	}
	for _, tc := range []struct {
		selected, required string
		want, failure      bool
	}{
		{"2.100", "2.99", true, false}, {"2.99", "2.100", false, false},
		{"3.100", "2.6", false, false}, {"latest", "2.6", false, false},
		{"2.latest", "2.6", true, false}, {"2.6.0", "2.6", true, false},
		{"2.6", "2.6.0", false, false}, {"bad", "2.6", false, true},
	} {
		t.Run("matches/"+tc.selected+"/"+tc.required, func(t *testing.T) {
			got, err := Matches(tc.selected, tc.required)
			if got != tc.want || (err != nil) != tc.failure {
				t.Fatal(got, err, tc.want, tc.failure)
			}
		})
	}
}
