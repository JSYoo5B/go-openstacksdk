package blockstorage

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestVolumeSearchViewKnownNullsAliasesAndIndependentLocation(t *testing.T) {
	wire := json.RawMessage(`{"id":false,"name":[1],"unknown":{"keep":9007199254740993},"image_id":"first","imageRef":"second","source_volid":"wire-first","source_volume_id":"normalized-last","is_bootable":"FALSE","bootable":"TrUe","location":{"cloud":"untrusted"},"attachments":{"device":false},"metadata":[],"OS-SCH-HNT:scheduler_hints":[["x",1]],"volume_image_metadata":{"n":9007199254740993}}`)
	original := string(wire)
	location := json.RawMessage(`{"cloud":"owned","zone":false,"project":{"id":true}}`)
	view, err := volumeSearchView(wire, location)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(view, &fields); err != nil {
		t.Fatal(err)
	}
	wantKeys := strings.Fields("attachments availability_zone backup_id consistency_group_id consumes_quota cluster_name created_at description encryption_key_id extended_replication_status group_id host image_id is_bootable is_encrypted is_multiattach migration_id migration_status project_id replication_driver_data provider_id replication_status scheduler_hints service_uuid shared_targets size snapshot_id source_volume_id status updated_at user_id volume_image_metadata volume_type volume_type_id id name location metadata")
	if len(fields) != 38 {
		t.Fatal("missing or extra source descriptor", len(fields), string(view))
	}
	for _, key := range wantKeys {
		if _, ok := fields[key]; !ok {
			t.Fatal("missing known descriptor", key)
		}
	}
	expected := map[string]string{"id": "false", "name": "[1]", "image_id": `"second"`, "source_volume_id": `"normalized-last"`, "is_bootable": "true", "attachments": `[{"device":false}]`, "metadata": "{}", "scheduler_hints": "{}", "status": "null", "is_encrypted": "null", "size": "null", "volume_image_metadata": `{"n":9007199254740993}`, "location": string(location)}
	for key, want := range expected {
		if string(fields[key]) != want {
			t.Error(key, string(fields[key]), want)
		}
	}
	if _, ok := fields["unknown"]; ok {
		t.Fatal("unknown wire field entered normalized view")
	}
	if string(wire) != original {
		t.Fatal("wire evidence changed")
	}
	location[0] = '!'
	wire[0] = '!'
	if !json.Valid(view) {
		t.Fatal("view aliases caller bytes")
	}
	reverse, err := volumeSearchView(json.RawMessage(`{"imageRef":"first","image_id":"last","bootable":"false","is_bootable":true}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(reverse, &fields); err != nil {
		t.Fatal(err)
	}
	if string(fields["image_id"]) != `"last"` || string(fields["is_bootable"]) != "true" || string(fields["location"]) != "null" {
		t.Fatal(string(reverse))
	}
}

func TestVolumeSearchViewDescriptorConversionsAndSourceSize(t *testing.T) {
	for _, tc := range []struct{ name, row, key, want string }{
		{"null list", `{"attachments":null}`, "attachments", "null"},
		{"literal list rows", `{"attachments":[null,false,7]}`, "attachments", "[null,false,7]"},
		{"list wraps string", `{"attachments":"raw"}`, "attachments", `["raw"]`},
		{"dict retains raw", `{"metadata":{"n":9007199254740993}}`, "metadata", `{"n":9007199254740993}`},
		{"falsey empty object", `{"multiattach":{}}`, "is_multiattach", "false"},
		{"truthy string false", `{"multiattach":"false"}`, "is_multiattach", "true"},
		{"truthy nonzero exact", `{"shared_targets":1e-9999}`, "shared_targets", "true"},
		{"falsey numeric zero", `{"shared_targets":-0.00e99}`, "shared_targets", "false"},
		{"boolstr bool", `{"encrypted":false}`, "is_encrypted", "false"},
		{"bool size", `{"size":true}`, "size", "true"},
		{"negative integer", `{"size":-7}`, "size", "-7"},
		{"negative fractional", `{"size":-7.9}`, "size", "-7"},
		{"large finite float", `{"size":1e30}`, "size", "1000000000000000019884624838656"},
		{"big exact integer", `{"size":900719925474099312345}`, "size", "900719925474099312345"},
		{"decimal unicode", `{"size":"١２৩"}`, "size", "123"},
		{"Unicode16 decimal", `{"size":"𑯰𑯱"}`, "size", "1"},
		{"signed string", `{"size":"-12"}`, "size", "0"},
		{"padded string", `{"size":" 12 "}`, "size", "0"},
		{"fractional string", `{"size":"1.2"}`, "size", "0"},
		{"container size", `{"size":[10]}`, "size", "0"},
		{"nondigit after superscript", `{"size":"²a"}`, "size", "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := volumeSearchView(json.RawMessage(tc.row), nil)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(raw, &fields); err != nil {
				t.Fatal(err)
			}
			if string(fields[tc.key]) != tc.want {
				t.Fatal(string(fields[tc.key]), tc.want)
			}
		})
	}
	// Unknown/untyped fields do not borrow the strict workflow model.
	raw, err := volumeSearchView(json.RawMessage(`{"status":{},"created_at":false,"consumes_quota":[1],"project_id":true}`), nil)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	if !reflect.DeepEqual([]string{string(fields["status"]), string(fields["created_at"]), string(fields["consumes_quota"]), string(fields["project_id"])}, []string{"{}", "false", "[1]", "true"}) {
		t.Fatal(string(raw))
	}
}

func TestVolumeSearchViewEagerErrorsAndDiscardedAlias(t *testing.T) {
	for _, row := range []json.RawMessage{json.RawMessage(`{"bootable":"yes"}`), json.RawMessage(`{"encrypted":1}`), json.RawMessage(`{"size":"²"}`), json.RawMessage(`{"size":1e9999}`), json.RawMessage(`[]`), json.RawMessage(`{} {}`), json.RawMessage{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}} {
		if view, err := volumeSearchView(row, nil); err == nil || view != nil {
			t.Fatal("invalid retained descriptor/JSON was accepted", string(row), string(view), err)
		}
	}
	if view, err := volumeSearchView(json.RawMessage(`{"bootable":"bad","is_bootable":false}`), nil); err != nil || view == nil {
		t.Fatal("discarded alias was converted", string(view), err)
	}
	for _, location := range []json.RawMessage{json.RawMessage(`broken`), json.RawMessage(`{} {}`), json.RawMessage{0xff}} {
		if view, err := volumeSearchView(json.RawMessage(`{}`), location); err == nil || view != nil {
			t.Fatal(string(view), err)
		}
	}
}
