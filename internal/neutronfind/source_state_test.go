package neutronfind

import (
	"encoding/json"
	"maps"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
)

func sourceStateMembers(t *testing.T, raw string) []cloudfilter.JSONMember {
	t.Helper()
	members, err := cloudfilter.ObjectMembers(json.RawMessage(raw))
	if err != nil {
		t.Fatal(err)
	}
	return members
}

func sourceStateText(values map[string]json.RawMessage) map[string]string {
	out := make(map[string]string, len(values))
	for key, raw := range values {
		out[key] = string(raw)
	}
	return out
}

// TestSourceStateConsumeAttrsMapping pins the Resource._consume_attrs rules:
// exact outer match on wire or attribute name, case-insensitive inner
// assignment to every matching wire key, CPython dict duplicate semantics,
// translate keeping header-named members as unknown Body members, and the
// constructor consuming declared Header names (case-insensitive wire).
func TestSourceStateConsumeAttrsMapping(t *testing.T) {
	profile := Profile{
		PreserveUnknownBody: true,
		Body: []Descriptor{
			{Name: "id", Wire: "id"},
			{Name: "is_admin_state_up", Wire: "admin_state_up"},
			{Name: "revision_number", Wire: "revision"},
		},
		Headers: []Descriptor{{Name: "if_match", Wire: "if-match"}},
	}
	mapping := profile.sourceMapping()
	got := sourceStateText(mapping.translate(sourceStateMembers(t, `{"id":"a","Admin_State_Up":1,"revision_number":2,"revision":3,"IF-MATCH":"h","if_match":"i","self":"s","x":null}`)))
	want := map[string]string{"id": `"a"`, "Admin_State_Up": "1", "revision": "3", "IF-MATCH": `"h"`, "if_match": `"i"`, "x": "null"}
	if !maps.Equal(got, want) {
		t.Fatalf("translate = %v, want %v", got, want)
	}
	got = sourceStateText(mapping.construct(sourceStateMembers(t, `{"id":"a","IF-MATCH":"h","if_match":"i","admin_state_up":true,"is_admin_state_up":false,"admin_state_up":true}`)))
	want = map[string]string{"id": `"a"`, "admin_state_up": "false"}
	if !maps.Equal(got, want) {
		t.Fatalf("construct = %v, want %v", got, want)
	}
	strict := Profile{Body: profile.Body}.sourceMapping()
	got = sourceStateText(strict.translate(sourceStateMembers(t, `{"id":"a","x":1}`)))
	if !maps.Equal(got, map[string]string{"id": `"a"`}) {
		t.Fatalf("unknown members must be dropped without PreserveUnknownBody: %v", got)
	}
	// A later attribute declared for the same wire replaces the earlier
	// attribute name, as Resource._get_mapping inverts its mapping.
	shadow := Profile{Body: []Descriptor{{Name: "first", Wire: "w"}, {Name: "second", Wire: "w"}}}.sourceMapping()
	got = sourceStateText(shadow.translate(sourceStateMembers(t, `{"first":1,"second":2}`)))
	if !maps.Equal(got, map[string]string{"w": "2"}) {
		t.Fatalf("shadowed mapping = %v", got)
	}
}
