package blockstorage_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	volumesv2 "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v2/volumes"
	volumesv3 "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/resource"
)

func volumeDeviceAccessorString(value string) *string { return &value }
func volumeDeviceAccessorOwned(kind string, rows []*blockstorage.AttachVolumeRecord, body map[string]json.RawMessage, server string) (*string, error) {
	if kind == "attachment observation" {
		return blockstorage.GetVolumeAttachDevice(&blockstorage.AttachVolumeObservation{Metadata: resource.Metadata{Body: body}, Attachments: rows}, server)
	}
	return blockstorage.GetVolumeAttachDevice(&blockstorage.VolumeInfo{Metadata: resource.Metadata{Body: body}, Attachments: rows}, server)
}
func volumeDeviceAccessorError(t *testing.T, err error, operation, key string, row int) {
	t.Helper()
	var context *resource.OperationError
	var response *resource.ResponseError
	var missing *resource.NotFoundError
	if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &context) || context.Operation != operation || context.Resource != "volume" || context.Cause == nil || errors.As(err, &response) || errors.As(err, &missing) || key != "" && !strings.Contains(err.Error(), key) || row >= 0 && !strings.Contains(err.Error(), fmt.Sprint(row)) {
		t.Fatalf("local consumed-input error lost identity/context key=%q row=%d: %v", key, row, err)
	}
}

func TestVolumeAttachDeviceOwnedModelsUseLiteralCurrentSelectionWithoutCloudState(t *testing.T) {
	for _, kind := range []string{"attachment observation", "volume info"} {
		t.Run(kind, func(t *testing.T) {
			rows := []*blockstorage.AttachVolumeRecord{{ServerID: volumeDeviceAccessorString("different"), Body: map[string]json.RawMessage{"server_id": json.RawMessage(`"different"`)}}, {ServerID: volumeDeviceAccessorString(" /target?x# "), Device: volumeDeviceAccessorString(" /dev/disk/by-id/vol?raw# ")}}
			body := map[string]json.RawMessage{"id": json.RawMessage(`false`), "status": json.RawMessage(`{"stale":true}`), "attachments": json.RawMessage(`null`)}
			device, err := volumeDeviceAccessorOwned(kind, rows, body, " /target?x# ")
			if err != nil || device == nil || *device != " /dev/disk/by-id/vol?raw# " {
				t.Fatal(device, err)
			}
			device, err = volumeDeviceAccessorOwned(kind, rows, body, " /TARGET?x# ")
			if err != nil || device != nil {
				t.Fatal("server comparison must remain literal/case-sensitive", device, err)
			}
			device, err = volumeDeviceAccessorOwned(kind, []*blockstorage.AttachVolumeRecord{{ServerID: volumeDeviceAccessorString(""), Device: volumeDeviceAccessorString("")}}, nil, "")
			if err != nil || device == nil || *device != "" {
				t.Fatal("explicit empty server/device is a match", device, err)
			}
			for _, rows := range [][]*blockstorage.AttachVolumeRecord{nil, {}} {
				device, err = volumeDeviceAccessorOwned(kind, rows, map[string]json.RawMessage{"attachments": json.RawMessage(`[]`)}, "server")
				if rows == nil {
					if device != nil {
						t.Fatal(device)
					}
					volumeDeviceAccessorError(t, err, "GetVolumeAttachDevice", "attachments", -1)
				} else if err != nil || device != nil {
					t.Fatal(device, err)
				}
			}
		})
	}
	for _, call := range []func() (*string, error){func() (*string, error) {
		return blockstorage.GetVolumeAttachDevice((*blockstorage.AttachVolumeObservation)(nil), "server")
	}, func() (*string, error) {
		return blockstorage.GetVolumeAttachDevice((*blockstorage.VolumeInfo)(nil), "server")
	}} {
		device, err := call()
		if device != nil {
			t.Fatal(device)
		}
		volumeDeviceAccessorError(t, err, "GetVolumeAttachDevice", "", -1)
	}
	created := &blockstorage.CreateVolumeResponse{Volume: &blockstorage.VolumeInfo{Attachments: []*blockstorage.AttachVolumeRecord{{ServerID: volumeDeviceAccessorString("server"), Device: volumeDeviceAccessorString("/dev/vdc")}}}}
	deleted := &blockstorage.DeleteVolumeResult{Ready: created.Volume}
	for _, volume := range []*blockstorage.VolumeInfo{created.Volume, deleted.Ready} {
		device, err := blockstorage.GetVolumeAttachDevice(volume, "server")
		if err != nil || device == nil || *device != "/dev/vdc" {
			t.Fatal("existing workflow model needs no caller adapter", device, err)
		}
	}
}

func TestVolumeAttachDeviceOwnedFirstMatchStopsOnEmptyNullOrMissingAndVisitsOnlyPrefix(t *testing.T) {
	for _, kind := range []string{"attachment observation", "volume info"} {
		for _, tc := range []struct {
			name string
			rows []*blockstorage.AttachVolumeRecord
			want *string
			key  string
			row  int
		}{
			{name: "first empty ignores tail", rows: []*blockstorage.AttachVolumeRecord{{ServerID: volumeDeviceAccessorString("server"), Device: volumeDeviceAccessorString("")}, nil, {ServerID: volumeDeviceAccessorString("server"), Device: volumeDeviceAccessorString("later")}}, want: volumeDeviceAccessorString("")},
			{name: "first canonical null ignores tail", rows: []*blockstorage.AttachVolumeRecord{{ServerID: volumeDeviceAccessorString("server"), Body: map[string]json.RawMessage{"device": json.RawMessage(`null`)}}, nil, {ServerID: volumeDeviceAccessorString("server"), Device: volumeDeviceAccessorString("later")}}},
			{name: "first missing device cannot fall back", rows: []*blockstorage.AttachVolumeRecord{{ServerID: volumeDeviceAccessorString("server"), Body: map[string]json.RawMessage{}}, {ServerID: volumeDeviceAccessorString("server"), Device: volumeDeviceAccessorString("later")}}, key: "device", row: 0},
			{name: "visited nil row fails", rows: []*blockstorage.AttachVolumeRecord{nil, {ServerID: volumeDeviceAccessorString("server"), Device: volumeDeviceAccessorString("later")}}, key: "attachment", row: 0},
			{name: "visited missing server fails", rows: []*blockstorage.AttachVolumeRecord{{Body: map[string]json.RawMessage{}}, {ServerID: volumeDeviceAccessorString("server"), Device: volumeDeviceAccessorString("later")}}, key: "server_id", row: 0},
			{name: "wrong case server fails", rows: []*blockstorage.AttachVolumeRecord{{Body: map[string]json.RawMessage{"Server_ID": json.RawMessage(`"server"`)}}, {ServerID: volumeDeviceAccessorString("server"), Device: volumeDeviceAccessorString("later")}}, key: "server_id", row: 0},
			{name: "null server skips missing device", rows: []*blockstorage.AttachVolumeRecord{{Body: map[string]json.RawMessage{"server_id": json.RawMessage(`null`)}}, {ServerID: volumeDeviceAccessorString("server"), Device: volumeDeviceAccessorString("selected")}}, want: volumeDeviceAccessorString("selected")},
			{name: "second visited missing device index", rows: []*blockstorage.AttachVolumeRecord{{ServerID: volumeDeviceAccessorString("other")}, {ServerID: volumeDeviceAccessorString("server"), Body: map[string]json.RawMessage{"Device": json.RawMessage(`"wrong-case"`)}}}, key: "device", row: 1},
		} {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				device, err := volumeDeviceAccessorOwned(kind, tc.rows, nil, "server")
				if tc.key != "" {
					if device != nil {
						t.Fatal(device)
					}
					volumeDeviceAccessorError(t, err, "GetVolumeAttachDevice", tc.key, tc.row)
				} else if err != nil || !reflect.DeepEqual(device, tc.want) {
					t.Fatal(device, err)
				}
			})
		}
	}
}

func TestVolumeAttachDeviceOwnedManualPresenceAndTypedCurrentAuthority(t *testing.T) {
	for _, kind := range []string{"attachment observation", "volume info"} {
		for _, tc := range []struct {
			name, server string
			rows         []*blockstorage.AttachVolumeRecord
			want         *string
			key          string
		}{
			{name: "manual nil server is explicit null", server: "", rows: []*blockstorage.AttachVolumeRecord{{}, {ServerID: volumeDeviceAccessorString(""), Device: volumeDeviceAccessorString("selected")}}, want: volumeDeviceAccessorString("selected")},
			{name: "manual matched nil device is explicit null", server: "server", rows: []*blockstorage.AttachVolumeRecord{{ServerID: volumeDeviceAccessorString("server")}, nil}},
			{name: "manual nonnil empty Body makes missing distinguishable", server: "server", rows: []*blockstorage.AttachVolumeRecord{{ServerID: volumeDeviceAccessorString("server"), Body: map[string]json.RawMessage{}}}, key: "device"},
			{name: "caller values override stale or absent Body", server: "current", rows: []*blockstorage.AttachVolumeRecord{{ServerID: volumeDeviceAccessorString("current"), Device: volumeDeviceAccessorString("selected"), Body: map[string]json.RawMessage{"server_id": json.RawMessage(`false`), "device": json.RawMessage(`{"stale":true}`), "volume_id": json.RawMessage(`[]`), "attached_at": json.RawMessage(`false`)}}}, want: volumeDeviceAccessorString("selected")},
			{name: "nonnull caller device overrides missing Body", server: "current", rows: []*blockstorage.AttachVolumeRecord{{ServerID: volumeDeviceAccessorString("current"), Device: volumeDeviceAccessorString("selected"), Body: map[string]json.RawMessage{}}}, want: volumeDeviceAccessorString("selected")},
			{name: "present stale server is current typed null", server: "", rows: []*blockstorage.AttachVolumeRecord{{Device: volumeDeviceAccessorString("stale"), Body: map[string]json.RawMessage{"server_id": json.RawMessage(`""`)}}, {ServerID: volumeDeviceAccessorString(""), Device: volumeDeviceAccessorString("selected")}}, want: volumeDeviceAccessorString("selected")},
			{name: "present stale device is current typed null", server: "current", rows: []*blockstorage.AttachVolumeRecord{{ServerID: volumeDeviceAccessorString("current"), Body: map[string]json.RawMessage{"device": json.RawMessage(`"stale"`)}}, {ServerID: volumeDeviceAccessorString("current"), Device: volumeDeviceAccessorString("later")}}},
			{name: "present nil raw device is current null", server: "current", rows: []*blockstorage.AttachVolumeRecord{{ServerID: volumeDeviceAccessorString("current"), Body: map[string]json.RawMessage{"device": nil}}}},
		} {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				device, err := volumeDeviceAccessorOwned(kind, tc.rows, map[string]json.RawMessage{}, tc.server)
				if tc.key != "" {
					volumeDeviceAccessorError(t, err, "GetVolumeAttachDevice", tc.key, 0)
				} else if err != nil || !reflect.DeepEqual(device, tc.want) {
					t.Fatal(device, err)
				}
			})
		}
	}
}

func TestVolumeAttachDeviceNativeV2V3ModelsExposeDeclaredZeroStringPresenceLoss(t *testing.T) {
	for _, kind := range []string{"v2", "v3"} {
		t.Run(kind, func(t *testing.T) {
			selectNative := func(raw string, server string) (*string, error) {
				if kind == "v2" {
					var value volumesv2.Volume
					if err := json.Unmarshal([]byte(raw), &value); err != nil {
						t.Fatal(err)
					}
					return blockstorage.GetVolumeAttachDevice(&value, server)
				}
				var value volumesv3.Volume
				if err := json.Unmarshal([]byte(raw), &value); err != nil {
					t.Fatal(err)
				}
				return blockstorage.GetVolumeAttachDevice(&value, server)
			}
			for _, deviceField := range []string{"", `,"device":null`, `,"device":""`} {
				device, err := selectNative(`{"attachments":[{"server_id":"server"`+deviceField+`},{"server_id":"server","device":"later"}]}`, "server")
				if err != nil || device == nil || *device != "" {
					t.Fatal("native missing/null/empty collapses to first empty string", device, err)
				}
			}
			device, err := selectNative(`{"attachments":[{}, {"server_id":"","device":"later"}]}`, "")
			if err != nil || device == nil || *device != "" {
				t.Fatal("native missing server_id zero string can match empty selector", device, err)
			}
			device, err = selectNative(`{"attachments":[{"server_id":null,"device":null}]}`, "")
			if err != nil || device == nil || *device != "" {
				t.Fatal("native explicit null server/device are current empty strings", device, err)
			}
			original, err := blockstorage.GetVolumeAttachDeviceFields(map[string]json.RawMessage{"attachments": json.RawMessage(`[{"server_id":null,"device":null}]`)}, "")
			if err != nil || original != nil {
				t.Fatal("raw presence-sensitive null server must remain a nonmatch", original, err)
			}
			device, err = selectNative(`{"attachments":[{"Server_ID":"server","Device":" /dev/literal?x# "}]}`, "server")
			if err != nil || device == nil || *device != " /dev/literal?x# " {
				t.Fatal("native decoder's existing case collapse is retained", device, err)
			}
			device, err = selectNative(`{"attachments":[]}`, "server")
			if err != nil || device != nil {
				t.Fatal(device, err)
			}
			for _, raw := range []string{`{}`, `{"attachments":null}`} {
				device, err = selectNative(raw, "server")
				if device != nil {
					t.Fatal(device)
				}
				volumeDeviceAccessorError(t, err, "GetVolumeAttachDevice", "attachments", -1)
			}
		})
	}
	for _, call := range []func() (*string, error){func() (*string, error) { return blockstorage.GetVolumeAttachDevice((*volumesv2.Volume)(nil), "server") }, func() (*string, error) { return blockstorage.GetVolumeAttachDevice((*volumesv3.Volume)(nil), "server") }} {
		device, err := call()
		if device != nil {
			t.Fatal(device)
		}
		volumeDeviceAccessorError(t, err, "GetVolumeAttachDevice", "", -1)
	}
	native2 := &volumesv2.Volume{Attachments: []volumesv2.Attachment{{ServerID: "server", Device: "native2"}}}
	native3 := &volumesv3.Volume{Attachments: []volumesv3.Attachment{{ServerID: "server", Device: "native3"}}}
	device2, err2 := blockstorage.GetVolumeAttachDevice(native2, "server")
	device3, err3 := blockstorage.GetVolumeAttachDevice(native3, "server")
	if err2 != nil || err3 != nil {
		t.Fatal(err2, err3)
	}
	*device2 = "caller"
	*device3 = "caller"
	if native2.Attachments[0].Device != "native2" || native3.Attachments[0].Device != "native3" {
		t.Fatal("native result pointer aliases a native record")
	}
}

func TestVolumeAttachDeviceRawIterableBoundaryAndLocalConsumedInputErrors(t *testing.T) {
	for _, raw := range []string{`[]`, `{}`, `""`} {
		device, err := blockstorage.GetVolumeAttachDeviceFields(map[string]json.RawMessage{"attachments": json.RawMessage(raw)}, "server")
		if err != nil || device != nil {
			t.Fatalf("empty plain JSON iterable=%s device=%s error=%v", raw, device, err)
		}
	}
	for _, fields := range []map[string]json.RawMessage{nil, {}, {"Attachments": json.RawMessage(`[]`)}, {"attachments": nil}, {"attachments": json.RawMessage(`null`)}, {"attachments": json.RawMessage(`false`)}, {"attachments": json.RawMessage(`1`)}, {"attachments": json.RawMessage(`{"server":"value"}`)}, {"attachments": json.RawMessage(`"x"`)}} {
		device, err := blockstorage.GetVolumeAttachDeviceFields(fields, "server")
		if device != nil {
			t.Fatal(device)
		}
		volumeDeviceAccessorError(t, err, "GetVolumeAttachDeviceFields", "attachments", -1)
	}
	for _, tc := range []struct {
		rows, key string
		row       int
	}{{`[null,{"server_id":"server","device":"later"}]`, "attachment", 0}, {`[{}, {"server_id":"server","device":"later"}]`, "server_id", 0}, {`[{"Server_ID":"server","device":"later"}]`, "server_id", 0}, {`[{"server_id":"server"},{"server_id":"server","device":"later"}]`, "device", 0}, {`[{"server_id":"other"},{"server_id":"server","Device":"wrong"}]`, "device", 1}, {`[[],{"server_id":"server","device":"later"}]`, "attachment", 0}} {
		device, err := blockstorage.GetVolumeAttachDeviceFields(map[string]json.RawMessage{"attachments": json.RawMessage(tc.rows)}, "server")
		if device != nil {
			t.Fatal(device)
		}
		volumeDeviceAccessorError(t, err, "GetVolumeAttachDeviceFields", tc.key, tc.row)
	}
	for _, raw := range []json.RawMessage{{}, []byte(" \n\t"), []byte(`[] true`)} {
		device, err := blockstorage.GetVolumeAttachDeviceFields(map[string]json.RawMessage{"attachments": raw}, "server")
		if device != nil {
			t.Fatal(device)
		}
		volumeDeviceAccessorError(t, err, "GetVolumeAttachDeviceFields", "attachments", -1)
	}
	malformed := map[string]json.RawMessage{"attachments": json.RawMessage(`[{"server_id":"server","device":"match"},]`)}
	device, err := blockstorage.GetVolumeAttachDeviceFields(malformed, "server")
	if device != nil {
		t.Fatal(device)
	}
	volumeDeviceAccessorError(t, err, "GetVolumeAttachDeviceFields", "attachments", -1)
	var syntax *json.SyntaxError
	if !errors.As(err, &syntax) {
		t.Fatal("JSON syntax cause was lost", err)
	}
	invalidUTF8 := append([]byte(`[{"server_id":"server","device":"`), 0xff)
	invalidUTF8 = append(invalidUTF8, []byte(`"}]`)...)
	device, err = blockstorage.GetVolumeAttachDeviceFields(map[string]json.RawMessage{"attachments": invalidUTF8}, "server")
	if device != nil {
		t.Fatal(device)
	}
	volumeDeviceAccessorError(t, err, "GetVolumeAttachDeviceFields", "attachments", -1)
	invalidTail := append([]byte(`[{"server_id":"server","device":"match"},"`), 0xff)
	invalidTail = append(invalidTail, []byte(`"]`)...)
	device, err = blockstorage.GetVolumeAttachDeviceFields(map[string]json.RawMessage{"attachments": invalidTail}, "server")
	if device != nil {
		t.Fatal("first match hid invalid UTF-8 in the later JSON input", device)
	}
	volumeDeviceAccessorError(t, err, "GetVolumeAttachDeviceFields", "attachments", -1)
}

func TestVolumeAttachDeviceRawFirstMatchPreservesTypingCastValuesAndNumericPrecision(t *testing.T) {
	for _, value := range []string{`""`, `" /dev/literal?x# "`, `"\u002fdev\u002fvdc"`, `false`, `0`, `900719925474099312345678901234567890`, `1.2345678901234567890123456789`, `[]`, `[null,9007199254740993]`, `{"nested":9007199254740993}`} {
		t.Run(value, func(t *testing.T) {
			rows := `[{"server_id":null,"device":{"ignored":true}},{"server_id":false},{"server_id":9007199254740993},{"server_id":[]},{"server_id":{}},{"server_id":"Server","device":null},{"server_id":"server","device":` + value + `,"volume_id":false,"attached_at":23},{"device":null},null]`
			fields := map[string]json.RawMessage{"attachments": json.RawMessage(rows), "id": json.RawMessage(`false`), "status": json.RawMessage(`[]`), "unrelated": json.RawMessage(`not even JSON`)}
			device, err := blockstorage.GetVolumeAttachDeviceFields(fields, "server")
			if err != nil || device == nil || string(device) != value {
				t.Fatalf("first raw cast value=%s device=%s error=%v", value, device, err)
			}
		})
	}
	for _, rows := range []string{`[{"server_id":"server","device":null},null,{}]`, `[{"server_id":"other"}]`, `[{"server_id":null},{"server_id":false},{"server_id":0},{"server_id":[]},{"server_id":{}}]`} {
		device, err := blockstorage.GetVolumeAttachDeviceFields(map[string]json.RawMessage{"attachments": json.RawMessage(rows)}, "server")
		if err != nil || device != nil {
			t.Fatal("null/no match is nil and skips device on unrelated rows", device, err)
		}
	}
	for _, nonstring := range []string{`false`, `0`, `9007199254740993`, `null`, `[]`, `{}`} {
		device, err := blockstorage.GetVolumeAttachDeviceFields(map[string]json.RawMessage{"attachments": json.RawMessage(`[{"server_id":` + nonstring + `,"device":"must not match"}]`)}, nonstring)
		if err != nil || device != nil {
			t.Fatal("raw server_id was coerced into a Go string", device, err)
		}
	}
	device, err := blockstorage.GetVolumeAttachDeviceFields(map[string]json.RawMessage{"attachments": json.RawMessage(`[{"server_id":" /target?x# ","device":{"literal":true}}]`)}, " /target?x# ")
	if err != nil || string(device) != `{"literal":true}` {
		t.Fatal(device, err)
	}
}

func TestVolumeAttachDevicePureOwnershipAndTypedCurrentVersusOriginalRawEvidence(t *testing.T) {
	raw := []byte(`{"id":"original-volume","status":"in-use","attachments":[{"server_id":"original","device":"/dev/original"}],"vendor":9007199254740993}`)
	var volume blockstorage.VolumeInfo
	if err := json.Unmarshal(raw, &volume); err != nil {
		t.Fatal(err)
	}
	volume.Header = http.Header{"X-Proof": {"original"}}
	volume.StatusCode = 200
	volume.Attachments[0].ServerID = volumeDeviceAccessorString("current")
	volume.Attachments[0].Device = volumeDeviceAccessorString("/dev/current")
	var before blockstorage.VolumeInfo
	if err := json.Unmarshal(raw, &before); err != nil {
		t.Fatal(err)
	}
	before.Header = volume.Header.Clone()
	before.StatusCode = 200
	before.Attachments[0].ServerID = volumeDeviceAccessorString("current")
	before.Attachments[0].Device = volumeDeviceAccessorString("/dev/current")
	typed, err := blockstorage.GetVolumeAttachDevice(&volume, "current")
	if err != nil || typed == nil || *typed != "/dev/current" {
		t.Fatal(typed, err)
	}
	original, err := blockstorage.GetVolumeAttachDeviceFields(volume.Body, "original")
	if err != nil || string(original) != `"/dev/original"` {
		t.Fatal(original, err)
	}
	absent, err := blockstorage.GetVolumeAttachDeviceFields(volume.Body, "current")
	if err != nil || absent != nil {
		t.Fatal("raw companion silently used edited typed state", absent, err)
	}
	if !reflect.DeepEqual(volume, before) {
		t.Fatal("local accessor mutated supplied model/header/raw proof")
	}
	*typed = "caller"
	original[0] = '!'
	if !reflect.DeepEqual(volume, before) {
		t.Fatal("returned typed/raw values alias supplied fields")
	}
	again, err := blockstorage.GetVolumeAttachDevice(&volume, "current")
	if err != nil || again == nil || *again != "/dev/current" || again == typed {
		t.Fatal(again, err)
	}
	rawAgain, err := blockstorage.GetVolumeAttachDeviceFields(volume.Body, "original")
	if err != nil || string(rawAgain) != `"/dev/original"` {
		t.Fatal(rawAgain, err)
	}
}

func TestVolumeAttachDeviceWholeModelDecoderAndDirectRawHaveExplicitDifferentBoundaries(t *testing.T) {
	body := []byte(`{"id":false,"status":23,"attachments":[{"server_id":"server","device":"first","attached_at":false},null,{"server_id":false,"device":[]}]}`)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatal(err)
	}
	direct, err := blockstorage.GetVolumeAttachDeviceFields(fields, "server")
	if err != nil || string(direct) != `"first"` {
		t.Fatal("direct helper should consume only the required ordered prefix", direct, err)
	}
	var observation blockstorage.AttachVolumeObservation
	var volume blockstorage.VolumeInfo
	if err := json.Unmarshal(body, &observation); err == nil {
		t.Fatal("owned whole observation decoder accepted malformed known field/tail")
	}
	if err := json.Unmarshal(body, &volume); err == nil {
		t.Fatal("owned whole volume decoder accepted malformed known field/tail")
	}
	tailOnly := []byte(`{"attachments":[{"server_id":"server","device":"first"},null]}`)
	if err := json.Unmarshal(tailOnly, &observation); err == nil {
		t.Fatal("owned whole decoder must still inspect null tail")
	}
	if err := json.Unmarshal(tailOnly, &fields); err != nil {
		t.Fatal(err)
	}
	direct, err = blockstorage.GetVolumeAttachDeviceFields(fields, "server")
	if err != nil || string(direct) != `"first"` {
		t.Fatal(direct, err)
	}
}
