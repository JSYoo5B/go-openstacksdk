package cloudfilter_test

import (
	"bytes"
	"encoding/json"
	"net/url"
	"reflect"
	"slices"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudfilter"
)

// The 16 expected wire queries come from the separate, explicitly unpinned
// Requests2.34.2 documented _encode_params AST + real urllib.parse.urlencode
// receipt gophercloudsdk-volume-limits-requests-query-context-probes.json.
// They are transport-dependency evidence, not SDK-pinned HTTP integration.
func TestRequestQueryValuesMatchesRequestsTwoLevelEncoding(t *testing.T) {
	cases := []struct {
		name, raw, query string
		values           []string
	}{
		{"none", `null`, "", []string{}},
		{"false", `false`, "project_id=False", []string{"False"}},
		{"true", `true`, "project_id=True", []string{"True"}},
		{"zero", `0`, "project_id=0", []string{"0"}},
		{"large-int", `9007199254740993`, "project_id=9007199254740993", []string{"9007199254740993"}},
		{"float", `1.5`, "project_id=1.5", []string{"1.5"}},
		{"empty-string", `""`, "project_id=", []string{""}},
		{"literal-reserved-unicode", `"a /?&= Ω"`, "project_id=a+%2F%3F%26%3D+%CE%A9", []string{"a /?&= Ω"}},
		{"empty-array", `[]`, "", []string{}},
		{"empty-object", `{}`, "", []string{}},
		{"repeated-array", `["a",null,false,""]`, "project_id=a&project_id=False&project_id=", []string{"a", "False", ""}},
		{"object-iterates-keys", `{"a":1,"b":false}`, "project_id=a&project_id=b", []string{"a", "b"}},
		{"nested-array", `[[1,2],[],null]`, "project_id=1&project_id=2", []string{"1", "2"}},
		{"nested-object", `[{"a":1,"b":2}]`, "project_id=a&project_id=b", []string{"a", "b"}},
		{"nested-null-array", `[[null]]`, "project_id=None", []string{"None"}},
		{"nested-false-array", `[[false]]`, "project_id=False", []string{"False"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := json.RawMessage(tc.raw)
			before := bytes.Clone(raw)
			got, err := cloudfilter.RequestQueryValues(raw)
			if err != nil || !slices.Equal(got, tc.values) || !bytes.Equal(raw, before) {
				t.Fatalf("values=%q error=%v input=%q; want=%q", got, err, raw, tc.values)
			}
			wire := url.Values{}
			for _, v := range got {
				wire.Add("project_id", v)
			}
			if actual := wire.Encode(); actual != tc.query {
				t.Fatalf("query=%q; want=%q", actual, tc.query)
			}
		})
	}
}

func TestObjectMembersKeepsDuplicateInsertionPositionAndOwnsEveryRawValue(t *testing.T) {
	raw := json.RawMessage(` {"z":1,"a":{"opaque":[null,false,9007199254740993]},"\u007a":1e-9999,"b":"last"} `)
	got, err := cloudfilter.ObjectMembers(raw)
	if err != nil || len(got) != 3 {
		t.Fatal(got, err)
	}
	keys := []string{got[0].Key, got[1].Key, got[2].Key}
	if !reflect.DeepEqual(keys, []string{"z", "a", "b"}) || string(got[0].Value) != "1e-9999" || string(got[1].Value) != `{"opaque":[null,false,9007199254740993]}` || string(got[2].Value) != `"last"` {
		t.Fatal(got)
	}
	// Each output is owned independently from input, other outputs, and a repeat.
	repeated, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		t.Fatal(err)
	}
	for i := range raw {
		raw[i] = '!'
	}
	if string(got[0].Value) != "1e-9999" || string(got[1].Value) != `{"opaque":[null,false,9007199254740993]}` {
		t.Fatal("input aliases outputs", got)
	}
	got[1].Value[0] = '!'
	if string(repeated[1].Value) != `{"opaque":[null,false,9007199254740993]}` || string(got[0].Value) != "1e-9999" {
		t.Fatal("outputs alias", got, repeated)
	}
}

func TestRequestQueryValuesKeepsObjectOrderAndUsesReprOnlyAfterDoseqExpansion(t *testing.T) {
	cases := []struct {
		raw  string
		want []string
	}{
		{`{"z":1,"a":2,"z":3,"b":4}`, []string{"z", "a", "b"}},
		{`[{"z":1,"a":2,"z":3,"b":4}]`, []string{"z", "a", "b"}},
		{`[[[null,false],{"z":1,"z":2,"a":3}]]`, []string{"[None, False]", "{'z': 2, 'a': 3}"}},
		{`[[{"x":"a'b","y":[true,null]}]]`, []string{`{'x': "a'b", 'y': [True, None]}`}},
		{`[["plain",[],{},null]]`, []string{"plain", "[]", "{}", "None"}},
	}
	for _, tc := range cases {
		got, err := cloudfilter.RequestQueryValues(json.RawMessage(tc.raw))
		if err != nil || !slices.Equal(got, tc.want) {
			t.Fatalf("raw=%s values=%q error=%v want=%q", tc.raw, got, err, tc.want)
		}
	}
}

func TestRequestQueryValuesRetainsScalarJSONNumberAndStringDomain(t *testing.T) {
	cases := []struct {
		raw  string
		want []string
	}{
		{`9007199254740993123456789`, []string{"9007199254740993123456789"}},
		{`-0`, []string{"0"}}, {`-0.0`, []string{"-0.0"}},
		{`1e-9999`, []string{"0.0"}}, {`1e9999`, []string{"inf"}},
		{`-1e9999`, []string{"-inf"}},
		{`"a\u0000\n /?&= Ω"`, []string{"a\x00\n /?&= Ω"}},
	}
	for _, tc := range cases {
		got, err := cloudfilter.RequestQueryValues(json.RawMessage(tc.raw))
		if err != nil || !slices.Equal(got, tc.want) {
			t.Fatalf("raw=%s values=%q error=%v want=%q", tc.raw, got, err, tc.want)
		}
	}
	got, err := cloudfilter.RequestQueryValues(nil)
	if err != nil || len(got) != 0 {
		t.Fatal(got, err)
	}
}

func TestObjectMembersAndRequestQueryValuesRejectInvalidDocumentsAtomically(t *testing.T) {
	bad := []json.RawMessage{json.RawMessage(``), json.RawMessage(`{`), json.RawMessage(`{"a":}`), json.RawMessage(`{} []`), json.RawMessage(`NaN`), json.RawMessage([]byte{'"', 0xff, '"'})}
	for _, raw := range bad {
		members, merr := cloudfilter.ObjectMembers(raw)
		values, qerr := cloudfilter.RequestQueryValues(raw)
		if merr == nil || members != nil || qerr == nil || values != nil {
			t.Fatalf("raw=%q members=%v err=%v values=%v err=%v", raw, members, merr, values, qerr)
		}
	}
	for _, raw := range []json.RawMessage{json.RawMessage(`null`), json.RawMessage(`[]`), json.RawMessage(`false`), json.RawMessage(`"x"`), json.RawMessage(`1`)} {
		members, err := cloudfilter.ObjectMembers(raw)
		if err == nil || members != nil {
			t.Fatal(string(raw), members, err)
		}
	}
}
