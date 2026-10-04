package blockstorage

import (
	"encoding/json"
	"reflect"
	"testing"
)

func volumeTypeViewContractFields(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(string(raw), err)
	}
	return fields
}

func TestVolumeTypeViewHasOnlySixDescriptorsAndOwnsAllValues(t *testing.T) {
	row := json.RawMessage(`{"description":[false,9007199254740993],"extra_specs":{"nested":[null,{"number":9007199254740993}]},"is_public":"false","id":{"opaque":9007199254740993},"name":[1],"metadata":{"not":"a-Type-field"},"project_id":"foreign","availability_zone":"unknown-zone","location":{"source":"wire"}}`)
	before := string(row)
	location := json.RawMessage(`{"source":"owned","project":{"id":"scope"},"zone":null}`)
	view, err := volumeTypeView(row, location, false)
	if err != nil {
		t.Fatal(err)
	}
	fields := volumeTypeViewContractFields(t, view)
	if len(fields) != 6 {
		t.Fatal("Type inherited MetadataMixin incorrectly", string(view))
	}
	for key, expected := range map[string]string{"description": `[false,9007199254740993]`, "extra_specs": `{"nested":[null,{"number":9007199254740993}]}`, "is_public": "true", "id": `{"opaque":9007199254740993}`, "name": "[1]", "location": string(location)} {
		if string(fields[key]) != expected {
			t.Error(key, string(fields[key]), expected)
		}
	}
	if string(row) != before {
		t.Fatal("raw Type row mutated")
	}
	row[0] = '!'
	location[0] = '!'
	if !json.Valid(view) {
		t.Fatal("view aliases caller raw bytes", string(view))
	}
	empty, err := volumeTypeView(json.RawMessage(`{}`), nil, false)
	if err != nil || string(empty) != `{"description":null,"extra_specs":null,"is_public":null,"id":null,"name":null,"location":null}` {
		t.Fatal(string(empty), err)
	}
}

func TestVolumeTypeViewBooleanAndDictionaryDescriptorsRetainDeclaredJSONPolicy(t *testing.T) {
	for _, tc := range []struct{ row, key, want string }{
		{`{"is_public":null}`, "is_public", "null"},
		{`{"is_public":false}`, "is_public", "false"},
		{`{"is_public":-0.00e99}`, "is_public", "false"},
		{`{"is_public":[]}`, "is_public", "false"},
		{`{"is_public":{}}`, "is_public", "false"},
		{`{"is_public":"false"}`, "is_public", "true"},
		{`{"is_public":[false]}`, "is_public", "true"},
		{`{"is_public":{"x":null}}`, "is_public", "true"},
		{`{"is_public":1e-9999}`, "is_public", "true"},
		{`{"extra_specs":null}`, "extra_specs", "null"},
		{`{"extra_specs":false}`, "extra_specs", "{}"},
		{`{"extra_specs":[["pair",1]]}`, "extra_specs", "{}"},
		{`{"extra_specs":{"opaque":[false,9007199254740993]}}`, "extra_specs", `{"opaque":[false,9007199254740993]}`},
	} {
		t.Run(tc.row, func(t *testing.T) {
			view, err := volumeTypeView(json.RawMessage(tc.row), nil, false)
			if err != nil || string(volumeTypeViewContractFields(t, view)[tc.key]) != tc.want {
				t.Fatal(string(view), err)
			}
		})
	}
}

func TestVolumeTypeViewAliasOrderAndDeclaredDuplicatePolicy(t *testing.T) {
	for _, tc := range []struct{ row, want string }{
		{`{"is_public":true,"os-volume-type-access:is_public":false}`, "false"},
		{`{"os-volume-type-access:is_public":false,"is_public":true}`, "true"},
		{`{"is_public":null,"os-volume-type-access:is_public":[]}`, "false"},
		{`{"is_public":true,"os-volume-type-access:is_public":false,"is_public":true}`, "true"},
	} {
		t.Run(tc.row, func(t *testing.T) {
			view, err := volumeTypeView(json.RawMessage(tc.row), nil, false)
			if err != nil || string(volumeTypeViewContractFields(t, view)["is_public"]) != tc.want {
				t.Fatal(string(view), err)
			}
		})
	}
	view, err := volumeTypeView(json.RawMessage(`{"Is_Public":true,"Description":false,"ID":"wrong","Location":{"raw":true}}`), json.RawMessage(`{"source":"owned"}`), false)
	fields := volumeTypeViewContractFields(t, view)
	if err != nil || string(fields["is_public"]) != "null" || string(fields["description"]) != "null" || string(fields["id"]) != "null" || string(fields["location"]) != `{"source":"owned"}` {
		t.Fatal("case-insensitive response alias was adopted", string(view), err)
	}
}

func TestVolumeTypeViewListWireLocationIsLazyAndMemberAlwaysUsesOwnedLocation(t *testing.T) {
	for _, rawLocation := range []string{"null", "false", "0", `""`, `[]`, `{}`, `"raw"`, `{"opaque":[false,9007199254740993]}`} {
		t.Run(rawLocation, func(t *testing.T) {
			row := json.RawMessage(`{"unknown":false,"project_id":"foreign","availability_zone":"other","metadata":{},"location":` + rawLocation + `}`)
			use, err := volumeTypeUsesWireLocation(row, false)
			if err != nil || !use {
				t.Fatal(use, err)
			}
			view, err := volumeTypeView(row, json.RawMessage(`invalid-unused-location`), false)
			if err != nil || string(volumeTypeViewContractFields(t, view)["location"]) != rawLocation {
				t.Fatal("unused owned policy was consumed or raw location coerced", string(view), err)
			}
			use, err = volumeTypeUsesWireLocation(row, true)
			if err != nil || use {
				t.Fatal(use, err)
			}
			owned := json.RawMessage(`{"source":"owned","zone":null}`)
			view, err = volumeTypeView(row, owned, true)
			if err != nil || string(volumeTypeViewContractFields(t, view)["location"]) != string(owned) {
				t.Fatal("member adopted raw computed location", string(view), err)
			}
			if view, err := volumeTypeView(row, json.RawMessage(`invalid-used-location`), true); err == nil || view != nil {
				t.Fatal(string(view), err)
			}
		})
	}
	row := json.RawMessage(`{"location":{"first":true},"location":false}`)
	view, err := volumeTypeView(row, nil, false)
	if err != nil || string(volumeTypeViewContractFields(t, view)["location"]) != "false" {
		t.Fatal("duplicate raw computed location did not retain last value", string(view), err)
	}
}

func TestVolumeTypeViewKnownPresenceIncludingNullDiscardsRawLocation(t *testing.T) {
	for _, key := range []string{"description", "extra_specs", "is_public", "os-volume-type-access:is_public", "id", "name"} {
		for _, value := range []string{"null", "false", `""`, `[]`, `{}`} {
			t.Run(key+"/"+value, func(t *testing.T) {
				encoded, _ := json.Marshal(key)
				row := json.RawMessage(`{"location":{"source":"wire"},` + string(encoded) + `:` + value + `}`)
				use, err := volumeTypeUsesWireLocation(row, false)
				if err != nil || use {
					t.Fatal("known falsey value was treated as absent Body", use, err)
				}
				view, err := volumeTypeView(row, json.RawMessage(`{"source":"owned"}`), false)
				if err != nil || string(volumeTypeViewContractFields(t, view)["location"]) != `{"source":"owned"}` {
					t.Fatal(string(view), err)
				}
				if view, err := volumeTypeView(row, json.RawMessage(`invalid-used-location`), false); err == nil || view != nil {
					t.Fatal("used owned location not validated", string(view), err)
				}
			})
		}
	}
}

func TestVolumeTypeViewRejectsNonobjectsMalformedJSONAndUTF8Atomically(t *testing.T) {
	for _, row := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(`false`), json.RawMessage(`[]`), json.RawMessage(`{} {}`), json.RawMessage(`{"id":`), json.RawMessage(`{"location":true} garbage`), json.RawMessage([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'})} {
		before := append(json.RawMessage(nil), row...)
		for _, member := range []bool{false, true} {
			if use, err := volumeTypeUsesWireLocation(row, member); err == nil || use {
				t.Fatal(string(row), use, err)
			}
			if view, err := volumeTypeView(row, nil, member); err == nil || view != nil {
				t.Fatal(string(row), string(view), err)
			}
		}
		if !reflect.DeepEqual(row, before) {
			t.Fatal("bad input mutated", string(row))
		}
	}
	for _, location := range []json.RawMessage{json.RawMessage{}, json.RawMessage(`invalid`), json.RawMessage(`{} {}`), json.RawMessage{0xff}} {
		if view, err := volumeTypeView(json.RawMessage(`{}`), location, false); err == nil || view != nil {
			t.Fatal(string(view), err)
		}
	}
}
