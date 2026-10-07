package cloudfilter_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudfilter"
	"github.com/JSYoo5B/gophercloudsdk/internal/jsonfilter"
)

func cloudFilterContractRows(t *testing.T, raw string) []json.RawMessage {
	t.Helper()
	var rows []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		t.Fatal(err)
	}
	return rows
}
func cloudFilterContractFilter(raw string) *json.RawMessage {
	value := json.RawMessage(raw)
	return &value
}
func cloudFilterContractEqual(t *testing.T, actual json.RawMessage, expected string) {
	t.Helper()
	equal, err := jsonfilter.EqualJSON(actual, json.RawMessage(expected))
	if err != nil || !equal {
		t.Fatalf("JSON result = %s, want %s; comparison error=%v", actual, expected, err)
	}
}
func cloudFilterContractSelect(t *testing.T, rows []json.RawMessage, pattern string, filters *json.RawMessage, indices []int) cloudfilter.Result {
	t.Helper()
	result, err := cloudfilter.Select(rows, pattern, filters, nil)
	if err != nil || result.Expression || result.Indices == nil || !reflect.DeepEqual(result.Indices, indices) {
		t.Fatal(result, err, indices)
	}
	selected := make([]json.RawMessage, len(indices))
	for i, index := range indices {
		selected[i] = rows[index]
	}
	expected, err := json.Marshal(selected)
	if err != nil {
		t.Fatal(err)
	}
	cloudFilterContractEqual(t, result.Value, string(expected))
	return result
}

func TestCloudFilterIdentifierPreservesPerRowExactGlobMatchesDuplicatesAndSourceOrder(t *testing.T) {
	rows := cloudFilterContractRows(t, `[{"id":"literal","name":"db*","row":0},{"id":"one","name":"db01","row":1},{"id":"DB02","name":"other","row":2},{"id":"one","name":"db01","row":1},{"id":"db03","name":"other","row":4}]`)
	cloudFilterContractSelect(t, rows, "db*", nil, []int{0, 1, 3, 4})
	cloudFilterContractSelect(t, rows, "one", nil, []int{1, 3})
	cloudFilterContractSelect(t, rows, "", nil, []int{0, 1, 2, 3, 4})
}

func TestCloudFilterGlobUsesCPythonRuneDotAllClassesAndLiteralBackslashes(t *testing.T) {
	for _, tc := range []struct {
		pattern string
		values  []string
		want    []int
	}{
		{"a?b", []string{"a/b", "a\nb", "a중b", "ab", "aXYb"}, []int{0, 1, 2}},
		{"*", []string{".ab", "a/b", "a\nb", "", "중"}, []int{0, 1, 2, 4}},
		{`a\*`, []string{`a\xyz`, "a*", `a\`}, []int{0, 2}},
		{"[z-a]", []string{"a", "z", "[z-a]"}, []int{2}},
		{"[z-aq]", []string{"q", "z", "a"}, []int{0}},
		{"[!z-a]", []string{"a", "z", "\n", "", "ab"}, []int{0, 1, 2}},
		{"[]", []string{"[]", "[", "]"}, []int{0}},
		{"[]]", []string{"]", "[", "x"}, []int{0}},
		{"[^a]", []string{"^", "a", "b"}, []int{0, 1}},
		{"[[]", []string{"[", "]"}, []int{0}},
		{"[a-b-c]", []string{"a", "b", "c", "-", "d"}, []int{0, 1, 2, 3}},
		{"[a--b]", []string{"b", "a", "-"}, []int{0}},
		{"[&~|]", []string{"&", "~", "|", "a"}, []int{0, 1, 2}},
		{"[!]]", []string{"]", "a", "\n", "ab"}, []int{1, 2}},
		{"[가-힣]", []string{"중", "A", "\n"}, []int{0}},
		{"db*", []string{"DB1", "db1"}, []int{1}},
		{"**?**", []string{"", "\n", "abc"}, []int{1, 2}},
		{"?", []string{"중", "🙂", "ab"}, []int{0, 1}},
	} {
		t.Run(tc.pattern, func(t *testing.T) {
			rows := make([]json.RawMessage, len(tc.values))
			for i, value := range tc.values {
				encoded, err := json.Marshal(map[string]any{"id": value, "name": ""})
				if err != nil {
					t.Fatal(err)
				}
				rows[i] = encoded
			}
			cloudFilterContractSelect(t, rows, tc.pattern, nil, tc.want)
		})
	}
}

func TestCloudFilterIdentifiersUsePythonJSONDomainStringAndContainerRepresentation(t *testing.T) {
	for _, tc := range []struct{ value, pattern string }{
		{`null`, "None"}, {`true`, "True"}, {`false`, "False"}, {`-0`, "0"}, {`9007199254740993`, "9007199254740993"},
		{`1.0`, "1.0"}, {`-0.0`, "-0.0"}, {`0.0001`, "0.0001"}, {`0.00001`, "1e-05"}, {`1e16`, "1e+16"},
		{`[true,null,"a"]`, "[True, None, 'a']"}, {`{"z":false,"a":1}`, "{'z': False, 'a': 1}"},
		{`["a'b","a\"b","a\\b","\u0000","\u00a0","중"]`, `["a'b", 'a"b', 'a\\b', '\x00', '\xa0', '중']`},
	} {
		t.Run(tc.pattern, func(t *testing.T) {
			rows := cloudFilterContractRows(t, `[{"id":`+tc.value+`,"name":"outside"}]`)
			cloudFilterContractSelect(t, rows, tc.pattern, nil, []int{0})
		})
	}
}

func TestCloudFilterMissingAndNullIdentifiersDifferFromEmptyStringsAndRemainCaseSensitive(t *testing.T) {
	rows := cloudFilterContractRows(t, `[{"id":"","name":""},{"id":null,"name":"other"},{"id":"different"},{"id":"None","name":"other"},{"id":false,"name":"other"},{"id":"true","name":"other"}]`)
	cloudFilterContractSelect(t, rows, "None", nil, []int{1, 2, 3})
	cloudFilterContractSelect(t, rows, "False", nil, []int{4})
	cloudFilterContractSelect(t, rows, "True", nil, []int{})
	cloudFilterContractSelect(t, rows, "*", nil, []int{1, 2, 3, 4, 5})
	cloudFilterContractSelect(t, rows, "", nil, []int{0, 1, 2, 3, 4, 5})
	cloudFilterContractSelect(t, cloudFilterContractRows(t, `[{"id":"discarded","id":"actual","name":"other"}]`), "actual", nil, []int{0})
}
