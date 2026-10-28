package flavors

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestFlavorRecordDeclaredDefaultsDescriptorsAndRawIdentity(t *testing.T) {
	defaults := map[string]string{
		"id": "null", "name": "null", "original_name": "null", "description": "null",
		"disk": "0", "ram": "0", "vcpus": "0", "swap": "0", "ephemeral": "0",
		"is_public": "true", "is_disabled": "null", "rxtx_factor": "null", "extra_specs": "{}", "location": "null",
	}
	for _, check := range []struct {
		name, body string
		want       map[string]string
	}{
		{"missing", `{}`, nil},
		{"present null", `{"id":null,"name":null,"original_name":null,"description":null,"disk":null,"ram":null,"vcpus":null,"swap":null,"ephemeral":null,"is_public":null,"is_disabled":null,"rxtx_factor":null,"extra_specs":null}`, map[string]string{"disk": "null", "ram": "null", "vcpus": "null", "swap": "null", "ephemeral": "null", "is_public": "null", "extra_specs": "null"}},
		{"original name fallback", `{"id":"","original_name":"legacy"}`, map[string]string{"id": `"legacy"`, "name": `"legacy"`, "original_name": `"legacy"`}},
		{"explicit empty name", `{"id":null,"name":"","original_name":"legacy"}`, map[string]string{"id": `"legacy"`, "name": `""`, "original_name": `"legacy"`}},
		{"explicit null name", `{"name":null,"original_name":"legacy"}`, map[string]string{"id": `"legacy"`, "original_name": `"legacy"`}},
		{"response descriptors", `{"disk":" +3 ","ram":"٢٤","vcpus":true,"swap":7.9,"ephemeral":[2],"is_public":"false","is_disabled":[],"rxtx_factor":" +1.25 ","extra_specs":[1]}`, map[string]string{"ram": "24", "vcpus": "true", "swap": "7", "is_disabled": "false", "rxtx_factor": "1.25"}},
		{"raw identity and values", `{"id":900719925474099312345,"name":false,"description":{"vendor":true},"rxtx_factor":true,"extra_specs":{"raw":null}}`, map[string]string{"id": "900719925474099312345", "name": "false", "description": `{"vendor":true}`, "rxtx_factor": "1", "extra_specs": `{"raw":null}`}},
		{"integer precision", `{"id":0,"name":false,"original_name":[1],"ram":900719925474099312345}`, map[string]string{"id": "[1]", "name": "false", "original_name": "[1]", "ram": "900719925474099312345"}},
		{"flat list and computed location", `{"flavor":{"id":"nested","name":"nested","disk":8},"location":{"cloud":"wire"},"vendor":900719925474099312345}`, nil},
	} {
		t.Run(check.name, func(t *testing.T) {
			var row flavorListRecord
			if err := json.Unmarshal([]byte(check.body), &row); err != nil {
				t.Fatal(err)
			}
			before, err := json.Marshal(row.Wire)
			if err != nil {
				t.Fatal(err)
			}
			if err := prepareFlavorListRecord(&row.FlavorRecord); err != nil {
				t.Fatal(err)
			}
			want := make(map[string]string, len(defaults))
			for key, raw := range defaults {
				want[key] = raw
			}
			for key, raw := range check.want {
				want[key] = raw
			}
			actual := make(map[string]string, len(row.Resource.Body))
			for key, raw := range row.Resource.Body {
				actual[key] = string(raw)
			}
			if !reflect.DeepEqual(actual, want) {
				t.Fatalf("actual = %#v, want %#v", actual, want)
			}
			after, err := json.Marshal(row.Wire)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("projection changed the actual response", string(after), err)
			}
			if string(row.Envelope) != check.body {
				t.Fatalf("string(row.Envelope) = %v, want %v", string(row.Envelope), check.body)
			}
		})
	}
}

func TestFlavorRecordResponseAliasOrderAndIndependentReceipts(t *testing.T) {
	for _, check := range []struct {
		name, body, public, ephemeral, disabled string
	}{
		{"wire aliases last", `{"is_public":true,"os-flavor-access:is_public":false,"ephemeral":1,"OS-FLV-EXT-DATA:ephemeral":2,"is_disabled":true,"OS-FLV-DISABLED:disabled":false}`, "false", "2", "false"},
		{"canonical aliases last", `{"os-flavor-access:is_public":false,"is_public":true,"OS-FLV-EXT-DATA:ephemeral":2,"ephemeral":1,"OS-FLV-DISABLED:disabled":false,"is_disabled":true}`, "true", "1", "true"},
		{"duplicate key keeps first position", `{"is_public":false,"os-flavor-access:is_public":true,"is_public":null,"ephemeral":1,"is_disabled":null}`, "true", "1", "null"},
	} {
		t.Run(check.name, func(t *testing.T) {
			var row flavorListRecord
			if err := json.Unmarshal([]byte(check.body), &row); err != nil {
				t.Fatal(err)
			}
			if err := prepareFlavorListRecord(&row.FlavorRecord); err != nil {
				t.Fatal(err)
			}
			if string(row.Resource.Body["is_public"]) != check.public {
				t.Fatalf("string(row.Resource.Body[\"is_public\"]) = %v, want %v", string(row.Resource.Body["is_public"]), check.public)
			}
			if string(row.Resource.Body["ephemeral"]) != check.ephemeral {
				t.Fatalf("string(row.Resource.Body[\"ephemeral\"]) = %v, want %v", string(row.Resource.Body["ephemeral"]), check.ephemeral)
			}
			if string(row.Resource.Body["is_disabled"]) != check.disabled {
				t.Fatalf("string(row.Resource.Body[\"is_disabled\"]) = %v, want %v", string(row.Resource.Body["is_disabled"]), check.disabled)
			}
		})
	}
	seed := map[string]json.RawMessage{"is_public": json.RawMessage("false"), "os-flavor-access:is_public": json.RawMessage("true"), "vendor": json.RawMessage("9")}
	normalized, err := normalizedFlavorRecordFields(seed, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(normalized) != 1 {
		t.Fatalf("len(normalized) = %v, want %v", len(normalized), 1)
	}
	if string(normalized["is_public"]) != "false" {
		t.Fatalf("string(normalized[\"is_public\"]) = %v, want %v", string(normalized["is_public"]), "false")
	}
	normalized["is_public"][0] = 'x'
	if string(seed["is_public"]) != "false" {
		t.Fatalf("string(seed[\"is_public\"]) = %v, want %v", string(seed["is_public"]), "false")
	}

	row := flavorListRecord{FlavorRecord: FlavorRecord{Enrichment: &FlavorExtraSpecsRecord{ExtraSpecs: json.RawMessage(`{"old":1}`)}}}
	metadata := flavorRecordMetadata(&row.FlavorRecord)
	metadata.Header = http.Header{"X-Proof": {"actual page"}}
	metadata.StatusCode = 203
	body := []byte(`{"id":"one","extra_specs":{"number":900719925474099312345},"vendor":null}`)
	if err := json.Unmarshal(body, &row); err != nil {
		t.Fatal(err)
	}
	if metadata != flavorRecordMetadata(&row.FlavorRecord) || row.Enrichment != nil {
		t.Fatal("list decoding changed the metadata address or retained old enrichment")
	}
	if err := prepareFlavorListRecord(&row.FlavorRecord); err != nil {
		t.Fatal(err)
	}
	if row.StatusCode != 203 {
		t.Fatalf("row.StatusCode = %v, want %v", row.StatusCode, 203)
	}
	if got := row.Header.Get("X-Proof"); got != "actual page" {
		t.Fatalf("row.Header.Get(\"X-Proof\") = %v, want %v", got, "actual page")
	}
	if got := row.Resource.Header.Get("X-Proof"); got != "actual page" {
		t.Fatalf("row.Resource.Header.Get(\"X-Proof\") = %v, want %v", got, "actual page")
	}
	row.Resource.Body["extra_specs"][0] = 'x'
	row.Resource.Header["X-Proof"][0] = "view caller"
	row.Header["X-Proof"][0] = "receipt caller"
	row.Envelope[0] = 'x'
	if string(row.Wire.Body["extra_specs"]) != `{"number":900719925474099312345}` {
		t.Fatalf("string(row.Wire.Body[\"extra_specs\"]) = %v, want %v", string(row.Wire.Body["extra_specs"]), `{"number":900719925474099312345}`)
	}
	if got := row.Wire.Header.Get("X-Proof"); got != "actual page" {
		t.Fatalf("row.Wire.Header.Get(\"X-Proof\") = %v, want %v", got, "actual page")
	}
	if want := byte('{'); body[0] != want {
		t.Fatalf("body[0] = %v, want %v", body[0], want)
	}
	if err := json.Unmarshal([]byte(`{"original_name":"next"}`), &row); err != nil || row.Resource != nil {
		t.Fatal("reused decoder retained an old projected Resource", err)
	}
	if metadata != flavorRecordMetadata(&row.FlavorRecord) {
		t.Fatal("reused decoder changed the metadata address")
	}
	if err := prepareFlavorListRecord(&row.FlavorRecord); err != nil {
		t.Fatal(err)
	}
	if got := flavorRecordIdentity(&row.FlavorRecord); got != "next" {
		t.Fatalf("flavorRecordIdentity(&row.FlavorRecord) = %v, want %v", got, "next")
	}
	if got := flavorRecordName(&row.FlavorRecord); got != "next" {
		t.Fatalf("flavorRecordName(&row.FlavorRecord) = %v, want %v", got, "next")
	}
	if _, present := row.Wire.Body["vendor"]; present {
		t.Fatal("new row retained an old actual response field")
	}
}

func TestFlavorRecordDescriptorErrorsAndPassiveStringIdentity(t *testing.T) {
	for _, check := range []struct {
		name, key, raw string
	}{
		{"integer digit conversion", "ram", `"²"`},
		{"float string conversion", "rxtx_factor", `"opaque"`},
		{"float nonfinite", "rxtx_factor", `"NaN"`},
		{"malformed known value", "description", `{"incomplete":`},
	} {
		t.Run(check.name, func(t *testing.T) {
			view, err := projectFlavorRecord(map[string]json.RawMessage{check.key: json.RawMessage(check.raw)}, resource.Metadata{})
			if view != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("descriptor error was lost or returned a partially projected view", view, err)
			}
		})
	}
	if view, err := projectFlavorRecord(map[string]json.RawMessage{"name": {'"', 0xff, '"'}}, resource.Metadata{}); view != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("invalid UTF-8 was replaced in the projected name", view, err)
	}
	for _, invalid := range []string{`null`, `[]`, `{"ram":`} {
		var row flavorListRecord
		if err := json.Unmarshal([]byte(invalid), &row); err == nil {
			t.Fatal("invalid flat row was accepted", invalid)
		}
		if _, err := normalizedFlavorRecordFields(nil, json.RawMessage(invalid)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("invalid selected object was accepted", invalid, err)
		}
	}
	for _, check := range []struct {
		body, id, name string
		markerOK       bool
	}{
		{`{"id":7,"name":"name"}`, "", "name", false},
		{`{"id":null,"name":"name"}`, "name", "name", true},
		{`{"id":false,"name":null,"original_name":"legacy"}`, "legacy", "", true},
		{`{"id":[],"name":"","original_name":0}`, "", "", false},
		{`{"id":" a/b+% ","name":""}`, " a/b+% ", "", true},
		{`{"id":"  "}`, "  ", "", false},
	} {
		var row flavorListRecord
		if err := json.Unmarshal([]byte(check.body), &row); err != nil {
			t.Fatal(err)
		}
		if err := prepareFlavorListRecord(&row.FlavorRecord); err != nil {
			t.Fatal(err)
		}
		if got := flavorRecordIdentity(&row.FlavorRecord); got != check.id {
			t.Fatalf("flavorRecordIdentity(&row.FlavorRecord) = %v, want %v", got, check.id)
		}
		if got := flavorRecordName(&row.FlavorRecord); got != check.name {
			t.Fatalf("flavorRecordName(&row.FlavorRecord) = %v, want %v", got, check.name)
		}
		marker, err := flavorRecordMarker(&row.FlavorRecord)
		if check.markerOK {
			if err != nil || marker != check.id {
				t.Fatal("marker changed the logical string identity", marker, check.id, err)
			}
		} else if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("nonstring or empty logical marker acquired a string identity", marker, err)
		}
		filtered, err := flavorRecordFilterValue(&row.FlavorRecord, "id")
		if err != nil || !bytes.Equal(filtered, row.Resource.Body["id"]) {
			t.Fatal("filter did not use the descriptor view", string(filtered), err)
		}
		filtered[0] = 'x'
		if !json.Valid(row.Resource.Body["id"]) {
			t.Fatal("filter result aliases the Resource")
		}
	}
	if flavorRecordMetadata(nil) != nil || flavorRecordIdentity(nil) != "" || flavorRecordName(nil) != "" {
		t.Fatal("passive nil record helpers invented a value")
	}
	if err := prepareFlavorListRecord(nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := flavorRecordMarker(nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := flavorRecordFilterValue(nil, "id"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
