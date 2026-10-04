package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

const vmID = "returned-한글"
const vmMember = volumeReadContractBase + "volumes/ref"
const vmUpdatePath = volumeReadContractBase + "volumes/" + vmID
const vmActionPath = vmUpdatePath + "/action"

type vmOutcome struct {
	nilResult bool
	resolved  *blockstorage.GetVolumeResult
	id        string
	applied   *blockstorage.VolumeMutationPage
	volume    *resource.RawResource
	value     json.RawMessage
	err       error
}

func vmClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	return volumeReadContractClient(cloud)
}
func vmFields(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	return volumeReadContractFields(t, raw)
}
func vmCall(ctx context.Context, client *gophercloud.ServiceClient, op, name string, updates []blockstorage.UpdateVolumeOption, bootable []blockstorage.SetVolumeBootableOption) vmOutcome {
	if op == "UpdateVolume" {
		result, err := blockstorage.UpdateVolume(ctx, client, blockstorage.UpdateVolumeRequest{NameOrID: name}, updates...)
		if result == nil {
			return vmOutcome{nilResult: true, err: err}
		}
		return vmOutcome{resolved: result.Resolved, id: result.VolumeID, applied: result.Applied, volume: result.Volume, value: result.Value, err: err}
	}
	result, err := blockstorage.SetVolumeBootable(ctx, client, blockstorage.SetVolumeBootableRequest{NameOrID: name}, bootable...)
	if result == nil {
		return vmOutcome{nilResult: true, err: err}
	}
	return vmOutcome{resolved: result.Resolved, id: result.VolumeID, applied: result.Applied, err: err}
}
func vmChanged() []blockstorage.UpdateVolumeOption {
	return []blockstorage.UpdateVolumeOption{blockstorage.WithUpdateVolumeDescription("requested")}
}
func vmLookup(t *testing.T, r *http.Request, token string) {
	t.Helper()
	volumeReadContractWire(t, r, vmMember, token)
	if r.URL.RawQuery != "" {
		t.Error("member lookup query changed", r.URL)
	}
}
func vmMutation(t *testing.T, r *http.Request, op, token string) map[string]json.RawMessage {
	t.Helper()
	method, path := http.MethodPut, vmUpdatePath
	if op == "SetVolumeBootable" {
		method, path = http.MethodPost, vmActionPath
	}
	if r.Method != method || r.URL.Path != path || r.URL.RawQuery != "" || r.Header.Get("X-Source") != "entry" || r.Header.Get("X-Auth-Token") != token || r.Header.Get("OpenStack-API-Version") != "volume 3.60" {
		t.Error("fixed mutation request changed", r.Method, r.URL, r.Header)
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
		return nil
	}
	return vmFields(t, body)
}
func vmResponse(r *http.Request, status int, body io.ReadCloser, proof string) *http.Response {
	return getVolumesContractResponse(r, status, body, proof)
}

func TestUpdateVolumeNoOpPreservesOriginalRawRecursivePythonEquality(t *testing.T) {
	cloud := testcloud.New(t)
	client := vmClient(cloud)
	var calls atomic.Int32
	original := `{"id":"returned-한글","description":"same","consumes_quota":true,"size":1.00,"metadata":{"nested":[true,{"x":0}],"ordered":{"b":2,"a":1}},"unknown":{"precise":9007199254740993}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		vmLookup(t, r, "test-token")
		if calls.Add(1) != 1 {
			t.Error("equal update sent PUT", r.URL)
		}
		w.Header().Set("X-Proof", "lookup")
		testcloud.JSON(w, 200, `{"volume":`+original+`}`)
	})
	fields := map[string]json.RawMessage{"description": json.RawMessage(`"same"`), "consumes_quota": json.RawMessage(`1`), "size": json.RawMessage(`true`), "metadata": json.RawMessage(`{"ordered":{"a":1,"b":2},"nested":[1,{"x":false}]}`)}
	result, err := blockstorage.UpdateVolume(context.Background(), client, blockstorage.UpdateVolumeRequest{NameOrID: "ref"}, blockstorage.WithUpdateVolumeFields(fields))
	if err != nil || result == nil || result.VolumeID != vmID || result.Applied != nil || result.Volume == nil || result.Resolved == nil || result.Resolved.Volume == nil || calls.Load() != 1 || string(result.Volume.Body["consumes_quota"]) != "true" || string(result.Volume.Body["size"]) != "1.00" || string(result.Volume.Body["metadata"]) != `{"nested":[true,{"x":0}],"ordered":{"b":2,"a":1}}` {
		t.Fatal(result, err, calls.Load())
	}
	result.Volume.Body["metadata"][0] = '!'
	result.Volume.Header.Set("X-Proof", "caller")
	result.Resolved.Observed.Body[0] = '!'
	if !json.Valid(result.Value) || !json.Valid(result.Resolved.Value) || result.Resolved.Volume.Header.Get("X-Proof") != "lookup" || string(result.Resolved.Volume.Body["unknown"]) != `{"precise":9007199254740993}` {
		t.Fatal("no-op result/lookup/raw proof alias", result)
	}
}

func TestUpdateVolumeDropsFalseyCloudNamesAndIgnoresUnknownAttributes(t *testing.T) {
	for _, raw := range []string{`null`, `false`, `0`, `""`, `[]`, `{}`} {
		t.Run(raw, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vmClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				vmLookup(t, r, "test-token")
				if calls.Add(1) != 1 {
					t.Error("dropped cloud kwargs caused PUT", r.URL)
				}
				testcloud.JSON(w, 200, `{"volume":{"id":"returned-한글","name":"original","description":"original"}}`)
			})
			fields := map[string]json.RawMessage{"name": json.RawMessage(raw), "display_name": json.RawMessage(`"unused display"`), "description": json.RawMessage(raw), "display_description": json.RawMessage(`"unused display"`), "unknown": json.RawMessage(`{"opaque":false}`), "source_replica": json.RawMessage(`"ignored native extension"`)}
			result, err := blockstorage.UpdateVolume(context.Background(), client, blockstorage.UpdateVolumeRequest{NameOrID: "ref"}, blockstorage.WithUpdateVolumeFields(fields))
			if err != nil || result == nil || result.Applied != nil || calls.Load() != 1 || string(vmFields(t, result.Value)["name"]) != `"original"` || string(vmFields(t, result.Value)["description"]) != `"original"` {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
	t.Run("truthy display aliases without canonical fields", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := vmClient(cloud)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch calls.Add(1) {
			case 1:
				vmLookup(t, r, "test-token")
				testcloud.JSON(w, 200, `{"volume":{"id":"returned-한글"}}`)
			case 2:
				body := vmMutation(t, r, "UpdateVolume", "test-token")
				volume := vmFields(t, body["volume"])
				if len(volume) != 2 || string(volume["name"]) != `"display name"` || string(volume["description"]) != `"display description"` {
					t.Error(body)
				}
				w.WriteHeader(204)
			default:
				t.Error(r.URL)
				w.WriteHeader(500)
			}
		})
		got := vmCall(context.Background(), client, "UpdateVolume", "ref", []blockstorage.UpdateVolumeOption{blockstorage.WithUpdateVolumeFields(map[string]json.RawMessage{"display_name": json.RawMessage(`"display name"`), "display_description": json.RawMessage(`"display description"`)})}, nil)
		if got.err != nil || got.nilResult || got.applied == nil || calls.Load() != 2 {
			t.Fatal(got, calls.Load())
		}
	})
}

func TestUpdateVolumeSendsOnlyDirtyFieldsForStringNumberListOrderAndExactDecimals(t *testing.T) {
	for _, tc := range []struct{ key, prior, requested, viewKey, wantView string }{{"consumes_quota", `1`, `"1"`, "", ""}, {"metadata", `{"items":[1,2]}`, `{"items":[2,1]}`, "", ""}, {"consumes_quota", `9007199254740992`, `9007199254740993`, "", ""}, {"size", `0`, `"3"`, "size", "3"}, {"bootable", `true`, `"FaLsE"`, "is_bootable", "false"}, {"metadata", `{}`, `[["key",1]]`, "metadata", "{}"}} {
		t.Run(tc.key+tc.prior+tc.requested, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vmClient(cloud)
			var calls atomic.Int32
			key := tc.key
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					vmLookup(t, r, "test-token")
					testcloud.JSON(w, 200, `{"volume":{"id":"returned-한글","description":"equal","`+key+`":`+tc.prior+`}}`)
				case 2:
					body := vmMutation(t, r, "UpdateVolume", "test-token")
					volume := vmFields(t, body["volume"])
					if len(body) != 1 || len(volume) != 1 || string(volume[key]) != tc.requested {
						t.Error("equal fields leaked or unequal field omitted", body)
					}
					w.WriteHeader(204)
				default:
					t.Error(r.URL)
					w.WriteHeader(500)
				}
			})
			got := vmCall(context.Background(), client, "UpdateVolume", "ref", []blockstorage.UpdateVolumeOption{blockstorage.WithUpdateVolumeFields(map[string]json.RawMessage{key: json.RawMessage(tc.requested), "description": json.RawMessage(`"equal"`)})}, nil)
			if got.err != nil || got.nilResult || got.applied == nil || got.volume == nil || calls.Load() != 2 || string(got.volume.Body[key]) != tc.requested {
				t.Fatal(got, calls.Load())
			}
			if tc.viewKey != "" && string(vmFields(t, got.value)[tc.viewKey]) != tc.wantView {
				t.Fatal("raw dirty field replaced by descriptor coercion", string(got.value))
			}
		})
	}
}

func TestUpdateVolumeAbsentKnownFieldVersusExplicitNullIsDirty(t *testing.T) {
	cloud := testcloud.New(t)
	client := vmClient(cloud)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			vmLookup(t, r, "test-token")
			testcloud.JSON(w, 200, `{"volume":{"id":"returned-한글"}}`)
		case 2:
			body := vmMutation(t, r, "UpdateVolume", "test-token")
			volume := vmFields(t, body["volume"])
			if len(body) != 1 || len(volume) != 1 || string(volume["status"]) != "null" {
				t.Error(body, volume)
			}
			w.WriteHeader(204)
		default:
			t.Error("refresh/cleanup", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.UpdateVolume(context.Background(), client, blockstorage.UpdateVolumeRequest{NameOrID: "ref"}, blockstorage.WithUpdateVolumeFields(map[string]json.RawMessage{"status": nil}))
	if err != nil || result == nil || result.Applied == nil || result.Applied.StatusCode != 204 || result.Volume == nil || string(result.Volume.Body["status"]) != "null" || calls.Load() != 2 {
		t.Fatal(result, err, calls.Load())
	}
	if _, present := result.Resolved.Volume.Body["status"]; present {
		t.Fatal("prior absent key was fabricated", result)
	}
}

func TestUpdateVolumeForwardsAllKnownBodyAttributesWithoutCreationDefaults(t *testing.T) {
	// Explicit source descriptor coverage exercises the public request, not a table derived from Go implementation.
	pairs := [][2]string{{"attachments", "attachments"}, {"availability_zone", "availability_zone"}, {"backup_id", "backup_id"}, {"consistency_group_id", "consistencygroup_id"}, {"consumes_quota", "consumes_quota"}, {"cluster_name", "cluster_name"}, {"created_at", "created_at"}, {"description", "description"}, {"encryption_key_id", "encryption_key_id"}, {"extended_replication_status", "os-volume-replication:extended_status"}, {"group_id", "group_id"}, {"host", "os-vol-host-attr:host"}, {"image_id", "imageRef"}, {"is_bootable", "bootable"}, {"is_encrypted", "encrypted"}, {"is_multiattach", "multiattach"}, {"migration_id", "os-vol-mig-status-attr:name_id"}, {"migration_status", "os-vol-mig-status-attr:migstat"}, {"project_id", "os-vol-tenant-attr:tenant_id"}, {"replication_driver_data", "os-volume-replication:driver_data"}, {"provider_id", "provider_id"}, {"replication_status", "replication_status"}, {"scheduler_hints", "OS-SCH-HNT:scheduler_hints"}, {"service_uuid", "service_uuid"}, {"shared_targets", "shared_targets"}, {"size", "size"}, {"snapshot_id", "snapshot_id"}, {"source_volume_id", "source_volid"}, {"status", "status"}, {"updated_at", "updated_at"}, {"user_id", "user_id"}, {"volume_image_metadata", "volume_image_metadata"}, {"volume_type", "volume_type"}, {"volume_type_id", "volume_type_id"}, {"name", "name"}, {"metadata", "metadata"}}
	fields := map[string]json.RawMessage{}
	expected := map[string]string{}
	for _, pair := range pairs {
		value := `"literal-` + pair[0] + `"`
		switch pair[0] {
		case "attachments":
			value = `[false]`
		case "is_bootable", "is_encrypted", "is_multiattach", "shared_targets":
			value = `false`
		case "size":
			value = `0`
		case "metadata":
			value = `{"precise":9007199254740993}`
		case "scheduler_hints":
			value = `{"hint":[false,9007199254740993]}`
		}
		fields[pair[0]] = json.RawMessage(value)
		expected[pair[1]] = value
	}
	cloud := testcloud.New(t)
	client := vmClient(cloud)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			vmLookup(t, r, "test-token")
			testcloud.JSON(w, 200, `{"volume":{"id":"returned-한글"}}`)
		case 2:
			body := vmMutation(t, r, "UpdateVolume", "test-token")
			volume := vmFields(t, body["volume"])
			if len(body) != 2 || len(volume) != 35 || string(body["OS-SCH-HNT:scheduler_hints"]) != expected["OS-SCH-HNT:scheduler_hints"] {
				t.Error("full descriptor/top-level hints breadth", body, volume)
			}
			for key, want := range expected {
				if key == "OS-SCH-HNT:scheduler_hints" {
					continue
				}
				if string(volume[key]) != want {
					t.Error(key, string(volume[key]), want)
				}
			}
			if _, present := volume["id"]; present {
				t.Error("route ID in update body")
			}
			w.WriteHeader(204)
		default:
			t.Error(r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.UpdateVolume(context.Background(), client, blockstorage.UpdateVolumeRequest{NameOrID: "ref"}, blockstorage.WithUpdateVolumeFields(fields))
	if err != nil || result == nil || result.Volume == nil || result.Applied == nil || calls.Load() != 2 || string(vmFields(t, result.Value)["size"]) != "0" {
		t.Fatal(result, err, calls.Load())
	}
}

func TestUpdateVolumeWireAliasesAndTypedAttributesHaveDeterministicPrecedence(t *testing.T) {
	cloud := testcloud.New(t)
	client := vmClient(cloud)
	var calls atomic.Int32
	fields := map[string]json.RawMessage{"image_id": json.RawMessage(`"wrong"`), "imageRef": json.RawMessage(`"wire-image"`), "host": json.RawMessage(`"wrong"`), "os-vol-host-attr:host": json.RawMessage(`"wire-host"`), "name": json.RawMessage(`"raw"`), "description": json.RawMessage(`"raw"`), "metadata": json.RawMessage(`null`)}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			vmLookup(t, r, "test-token")
			testcloud.JSON(w, 200, `{"volume":{"id":"returned-한글"}}`)
		case 2:
			body := vmMutation(t, r, "UpdateVolume", "test-token")
			volume := vmFields(t, body["volume"])
			want := map[string]string{"imageRef": `"wire-image"`, "os-vol-host-attr:host": `"wire-host"`, "name": `"typed name"`, "description": `"typed description"`, "metadata": `{}`}
			if len(body) != 1 || len(volume) != len(want) {
				t.Error(body)
			}
			for key, value := range want {
				if string(volume[key]) != value {
					t.Error(key, string(volume[key]), value)
				}
			}
			w.WriteHeader(204)
		default:
			t.Error(r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.UpdateVolume(context.Background(), client, blockstorage.UpdateVolumeRequest{NameOrID: "ref"}, blockstorage.WithUpdateVolumeFields(fields), blockstorage.WithUpdateVolumeName("typed name"), blockstorage.WithUpdateVolumeDescription("typed description"), blockstorage.WithUpdateVolumeMetadata(map[string]string{}))
	if err != nil || result == nil || result.Applied == nil || calls.Load() != 2 {
		t.Fatal(result, err, calls.Load())
	}
}

func TestUpdateVolumeDirtyComparisonConsumesLookupAliasesInActualWireOrder(t *testing.T) {
	for _, row := range []string{`{"id":"returned-한글","host":"first","os-vol-host-attr:host":"last"}`, `{"id":"returned-한글","os-vol-host-attr:host":"first","host":"last"}`, `{"id":"returned-한글","host":"first","os-vol-host-attr:host":"middle","host":"last"}`} {
		t.Run(row, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vmClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				vmLookup(t, r, "test-token")
				if calls.Add(1) != 1 {
					t.Error("wire-order equal value caused PUT", r.URL)
				}
				testcloud.JSON(w, 200, `{"volume":`+row+`}`)
			})
			result, err := blockstorage.UpdateVolume(context.Background(), client, blockstorage.UpdateVolumeRequest{NameOrID: "ref"}, blockstorage.WithUpdateVolumeFields(map[string]json.RawMessage{"host": json.RawMessage(`"last"`)}))
			if err != nil || result == nil || result.Applied != nil || calls.Load() != 1 || string(vmFields(t, result.Value)["host"]) != `"last"` {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestUpdateVolumeSchedulerHintsDirtyFalseyValuesStillCommitEmptyVolume(t *testing.T) {
	for _, hints := range []string{`null`, `false`, `0`, `""`, `[]`, `{}`, `{"precise":9007199254740993}`, `"truthy"`, `[["hint",1]]`} {
		t.Run(hints, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vmClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					vmLookup(t, r, "test-token")
					testcloud.JSON(w, 200, `{"volume":{"id":"returned-한글"}}`)
				case 2:
					body := vmMutation(t, r, "UpdateVolume", "test-token")
					volume := vmFields(t, body["volume"])
					if len(volume) != 0 {
						t.Error("scheduler hints remained in volume", body)
					}
					if hints == `{"precise":9007199254740993}` || hints == `"truthy"` || hints == `[["hint",1]]` {
						if len(body) != 2 || string(body["OS-SCH-HNT:scheduler_hints"]) != hints {
							t.Error(body)
						}
					} else if len(body) != 1 {
						t.Error("falsey hints entered top level", body)
					}
					w.WriteHeader(204)
				default:
					t.Error(r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := blockstorage.UpdateVolume(context.Background(), client, blockstorage.UpdateVolumeRequest{NameOrID: "ref"}, blockstorage.WithUpdateVolumeFields(map[string]json.RawMessage{"scheduler_hints": json.RawMessage(hints)}))
			if err != nil || result == nil || result.Applied == nil || calls.Load() != 2 {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestUpdateVolumePartialAndFlatResponsesMergeRequestedPriorStateAtomically(t *testing.T) {
	for _, response := range []string{`{"volume":{"description":"server","name":null,"id":"changed-response-id","consumes_quota":true,"unknown":"ignored response"}}`, `{"description":"server","name":null,"id":false,"consumes_quota":true,"unknown":"ignored response"}`} {
		t.Run(response, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vmClient(cloud)
			var calls atomic.Int32
			lookup := `{"volume":{"id":"returned-한글","name":"original","consumes_quota":1,"metadata":{"old":"kept"},"unknown":{"precise":9007199254740993}}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					vmLookup(t, r, "test-token")
					w.Header().Set("X-Proof", "lookup")
					testcloud.JSON(w, 200, lookup)
				case 2:
					body := vmMutation(t, r, "UpdateVolume", "test-token")
					volume := vmFields(t, body["volume"])
					if len(volume) != 2 || string(volume["description"]) != `"requested"` || string(volume["status"]) != `"local status"` {
						t.Error(body)
					}
					w.Header().Set("X-Proof", "mutation")
					testcloud.JSON(w, 203, response)
				default:
					t.Error("update refresh/verification/cleanup", r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := blockstorage.UpdateVolume(context.Background(), client, blockstorage.UpdateVolumeRequest{NameOrID: "ref"}, blockstorage.WithUpdateVolumeDescription("requested"), blockstorage.WithUpdateVolumeFields(map[string]json.RawMessage{"status": json.RawMessage(`"local status"`)}))
			if err != nil || result == nil || result.VolumeID != vmID || result.Applied == nil || result.Applied.StatusCode != 203 || string(result.Applied.Body) != response || result.Volume == nil || calls.Load() != 2 {
				t.Fatal(result, err, calls.Load())
			}
			view := vmFields(t, result.Value)
			if string(view["description"]) != `"server"` || string(view["name"]) != "null" || string(view["status"]) != `"local status"` || string(view["metadata"]) != `{"old":"kept"}` || string(result.Volume.Body["unknown"]) != `{"precise":9007199254740993}` || result.Volume.Header.Get("X-Proof") != "mutation" || result.Volume.StatusCode != 203 {
				t.Fatal("server/prior/request merge lost", result)
			}
			server := vmFields(t, json.RawMessage(response))
			if envelope, present := server["volume"]; present {
				server = vmFields(t, envelope)
			}
			if string(view["id"]) != string(server["id"]) || string(result.Volume.Body["id"]) != string(server["id"]) || string(result.Volume.Body["consumes_quota"]) != "true" || string(view["consumes_quota"]) != "true" {
				t.Fatal("server ID or equal numeric/bool raw replacement lost", result)
			}
			if _, present := view["unknown"]; present {
				t.Fatal("unknown response entered normalized view", string(result.Value))
			}
			result.Applied.Body[0] = '!'
			result.Applied.Header.Set("X-Proof", "caller")
			result.Volume.Body["metadata"][0] = '!'
			result.Volume.Header.Set("X-Proof", "row caller")
			result.Resolved.Observed.Body[0] = '!'
			if !json.Valid(result.Value) || !json.Valid(result.Resolved.Value) || result.Resolved.Volume.Header.Get("X-Proof") != "lookup" || string(result.Resolved.Volume.Body["metadata"]) != `{"old":"kept"}` {
				t.Fatal("merged/raw/physical/lookup ownership aliases", result)
			}
		})
	}
}

func TestUpdateVolumeAcceptedEmptyAndMalformedJSONKeepRequestedMergedState(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
	}{{202, ""}, {204, ""}, {304, ""}, {399, "not JSON"}, {203, "{"}, {200, "{} {}"}} {
		t.Run(http.StatusText(tc.status)+tc.body, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vmClient(cloud)
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
				if calls.Add(1) == 1 {
					vmLookup(t, r, "test-token")
					return vmResponse(r, 200, io.NopCloser(strings.NewReader(`{"volume":{"id":"returned-한글","unknown":false}}`)), "lookup"), nil
				}
				body := vmMutation(t, r, "UpdateVolume", "test-token")
				if string(vmFields(t, body["volume"])["description"]) != `"requested"` {
					t.Error(body)
				}
				return vmResponse(r, tc.status, io.NopCloser(strings.NewReader(tc.body)), "tolerated"), nil
			})
			got := vmCall(context.Background(), client, "UpdateVolume", "ref", vmChanged(), nil)
			if got.err != nil || got.nilResult || got.applied == nil || got.applied.StatusCode != tc.status || string(got.applied.Body) != tc.body || got.volume == nil || string(vmFields(t, got.value)["description"]) != `"requested"` || string(got.volume.Body["unknown"]) != "false" || calls.Load() != 2 {
				t.Fatal(got, calls.Load())
			}
		})
	}
}

func TestUpdateVolumeValidWrongShapesAndInvalidUTF8RetainAppliedWithoutLogicalResult(t *testing.T) {
	for _, body := range []string{`null`, `false`, `[]`, `"string"`, `{"volume":null}`, `{"volume":false}`, `{"volume":[]}`, `{"volume":"string"}`, `{"unknown":"` + string([]byte{0xff}) + `"}`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vmClient(cloud)
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
				if calls.Add(1) == 1 {
					vmLookup(t, r, "test-token")
					return vmResponse(r, 200, io.NopCloser(strings.NewReader(`{"volume":{"id":"returned-한글"}}`)), "lookup"), nil
				}
				vmMutation(t, r, "UpdateVolume", "test-token")
				return vmResponse(r, 203, io.NopCloser(strings.NewReader(body)), "bad-current"), nil
			})
			got := vmCall(context.Background(), client, "UpdateVolume", "ref", vmChanged(), nil)
			var physical *resource.ResponseError
			if got.nilResult || got.resolved == nil || got.resolved.Volume == nil || got.applied == nil || string(got.applied.Body) != body || got.value != nil || got.volume != nil || calls.Load() != 2 || !errors.As(got.err, &physical) || physical.StatusCode != 203 || string(physical.Body) != body {
				t.Fatal(got, calls.Load())
			}
			volumeReadContractOperation(t, got.err, "UpdateVolume")
		})
	}
}

func TestUpdateVolumeProposedDescriptorFailureIsLocalBeforePUT(t *testing.T) {
	for _, fields := range []map[string]json.RawMessage{{"bootable": json.RawMessage(`"invalid"`)}, {"encrypted": json.RawMessage(`{}`)}, {"size": json.RawMessage(`"²"`)}} {
		t.Run(string(mustVMJSON(t, fields)), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vmClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				vmLookup(t, r, "test-token")
				if calls.Add(1) != 1 {
					t.Error("bad proposed descriptor reached PUT", r.URL)
				}
				w.Header().Set("X-Proof", "lookup")
				testcloud.JSON(w, 200, `{"volume":{"id":"returned-한글"}}`)
			})
			got := vmCall(context.Background(), client, "UpdateVolume", "ref", []blockstorage.UpdateVolumeOption{blockstorage.WithUpdateVolumeFields(fields)}, nil)
			var physical *resource.ResponseError
			if got.nilResult || got.err == nil || got.resolved == nil || got.resolved.Volume == nil || got.resolved.Observed == nil || got.applied != nil || got.value != nil || got.volume != nil || calls.Load() != 1 || errors.As(got.err, &physical) {
				t.Fatal(got, calls.Load())
			}
		})
	}
}
func mustVMJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestUpdateVolumeReturnedDescriptorFailureIsPhysicalAndAtomic(t *testing.T) {
	for _, body := range []string{`{"volume":{"bootable":"invalid"}}`, `{"size":"²"}`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vmClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					vmLookup(t, r, "test-token")
					w.Header().Set("X-Proof", "lookup")
					testcloud.JSON(w, 200, `{"volume":{"id":"returned-한글"}}`)
				case 2:
					vmMutation(t, r, "UpdateVolume", "test-token")
					w.Header().Set("X-Proof", "current")
					testcloud.JSON(w, 203, body)
				default:
					t.Error(r.URL)
					w.WriteHeader(500)
				}
			})
			got := vmCall(context.Background(), client, "UpdateVolume", "ref", vmChanged(), nil)
			var physical *resource.ResponseError
			if got.nilResult || got.resolved == nil || got.resolved.Volume == nil || got.applied == nil || got.value != nil || got.volume != nil || calls.Load() != 2 || !errors.As(got.err, &physical) || physical.StatusCode != 203 || string(physical.Body) != body || physical.Header.Get("X-Proof") != "current" || got.resolved.Observed.Header.Get("X-Proof") != "lookup" {
				t.Fatal(got, calls.Load())
			}
		})
	}
}

func TestSetVolumeBootableDefaultTrueExplicitFalseAndOpaqueSub400Acknowledgement(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		for _, code := range []int{200, 201, 202, 203, 204, 302, 399} {
			t.Run(http.StatusText(code)+map[bool]string{false: " default", true: " false"}[explicit], func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vmClient(cloud)
				var calls atomic.Int32
				opaque := []byte{0xff, 0, 'o', 'p', 'a', 'q', 'u', 'e'}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					switch calls.Add(1) {
					case 1:
						vmLookup(t, r, "test-token")
						testcloud.JSON(w, 200, `{"volume":{"id":"returned-한글","status":"error","bootable":true}}`)
					case 2:
						body := vmMutation(t, r, "SetVolumeBootable", "test-token")
						want := "true"
						if explicit {
							want = "false"
						}
						inner := vmFields(t, body["os-set_bootable"])
						if len(body) != 1 || len(inner) != 1 || string(inner["bootable"]) != want {
							t.Error(body)
						}
						w.Header().Set("X-Proof", "action")
						w.WriteHeader(code)
						if code != 204 {
							_, _ = w.Write(opaque)
						}
					default:
						t.Error("bootable wait/verify/cleanup", r.URL)
						w.WriteHeader(500)
					}
				})
				var options []blockstorage.SetVolumeBootableOption
				if explicit {
					options = append(options, blockstorage.WithSetVolumeBootable(false))
				}
				got := vmCall(context.Background(), client, "SetVolumeBootable", "ref", nil, options)
				if got.err != nil || got.nilResult || got.id != vmID || got.applied == nil || got.applied.StatusCode != code || calls.Load() != 2 || got.resolved == nil || got.resolved.Volume == nil || string(vmFields(t, got.resolved.Value)["status"]) != `"error"` {
					t.Fatal(got, calls.Load())
				}
				if code != 204 && string(got.applied.Body) != string(opaque) {
					t.Fatal(got)
				}
			})
		}
	}
}

func TestVolumeMutationsOrdinaryFindFallbackUsesNameQueryAndActualReturnedID(t *testing.T) {
	for _, op := range []string{"UpdateVolume", "SetVolumeBootable"} {
		for _, code := range []int{400, 403, 404} {
			t.Run(op+http.StatusText(code), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vmClient(cloud)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					switch calls.Add(1) {
					case 1:
						vmLookup(t, r, "test-token")
						testcloud.JSON(w, code, `{"error":"missing"}`)
					case 2:
						volumeReadContractWire(t, r, volumeReadContractPath, "test-token")
						if r.URL.Query().Get("name") != "ref" || len(r.URL.Query()) != 1 {
							t.Error("ordinary find hint changed", r.URL)
						}
						w.Header().Set("X-Proof", "list")
						testcloud.JSON(w, 200, volumeReadContractPage(`[{"id":"returned-한글","name":"ref"}]`, ""))
					case 3:
						vmMutation(t, r, op, "test-token")
						w.WriteHeader(204)
					default:
						t.Error(r.URL)
						w.WriteHeader(500)
					}
				})
				got := vmCall(context.Background(), client, op, "ref", vmChanged(), nil)
				if got.err != nil || got.nilResult || got.id != vmID || got.resolved == nil || got.resolved.Volume == nil || got.resolved.Observed != nil || len(got.resolved.Pages) != 1 || got.applied == nil || calls.Load() != 3 {
					t.Fatal(got, calls.Load())
				}
			})
		}
	}
}

func TestVolumeMutationsCompletedAbsenceAndSecondMatchAmbiguityStopBeforeMutation(t *testing.T) {
	for _, op := range []string{"UpdateVolume", "SetVolumeBootable"} {
		for _, multiple := range []bool{false, true} {
			t.Run(op+map[bool]string{false: " absent", true: " ambiguous"}[multiple], func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vmClient(cloud)
				var calls atomic.Int32
				name := "unsafe / name"
				body := volumeReadContractPage(`[]`, "")
				if multiple {
					body = `{"volumes":[{"id":"one","name":"unsafe / name"},{"id":"two","name":"unsafe / name"},false],"volumes_links":false}`
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					volumeReadContractWire(t, r, volumeReadContractPath, "test-token")
					if r.URL.Query().Get("name") != name || calls.Add(1) != 1 {
						t.Error(r.URL)
					}
					w.Header().Set("X-Proof", "find")
					testcloud.JSON(w, 200, body)
				})
				got := vmCall(context.Background(), client, op, name, vmChanged(), nil)
				var physical *resource.ResponseError
				if got.nilResult || got.resolved == nil || got.resolved.Volume != nil || len(got.resolved.Pages) != 1 || got.applied != nil || calls.Load() != 1 || errors.As(got.err, &physical) {
					t.Fatal(got, calls.Load())
				}
				if multiple {
					if !errors.Is(got.err, resource.ErrAmbiguous) {
						t.Fatal(got.err)
					}
				} else {
					var missing *resource.NotFoundError
					if !errors.As(got.err, &missing) || missing.Reference != name || missing.Resource != "volume" {
						t.Fatal(got.err)
					}
				}
				volumeReadContractOperation(t, got.err, op)
			})
		}
	}
}

func TestVolumeMutationsRejectInvalidActualCanonicalIDLocallyWithResolvedProof(t *testing.T) {
	for _, op := range []string{"UpdateVolume", "SetVolumeBootable"} {
		for _, id := range []string{"", `null`, `false`, `0`, `""`, `[]`, `{}`, `"."`, `"a/b"`, `"a%b"`, `"a?b"`, `"a b"`, `"a\u2000b"`} {
			t.Run(op+id, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vmClient(cloud)
				var calls atomic.Int32
				row := `{}`
				if id != "" {
					row = `{"id":` + id + `}`
				}
				body := `{"volume":` + row + `}`
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					vmLookup(t, r, "test-token")
					if calls.Add(1) != 1 {
						t.Error("bad returned ID reached mutation", r.URL)
					}
					w.Header().Set("X-Proof", "actual lookup")
					testcloud.JSON(w, 200, body)
				})
				got := vmCall(context.Background(), client, op, "ref", vmChanged(), nil)
				var physical *resource.ResponseError
				if got.nilResult || got.id != "" || got.resolved == nil || got.resolved.Volume == nil || got.resolved.Observed == nil || string(got.resolved.Observed.Body) != body || got.applied != nil || got.volume != nil || got.value != nil || calls.Load() != 1 || !errors.Is(got.err, resource.ErrInvalidOption) || errors.As(got.err, &physical) {
					t.Fatal(got, calls.Load())
				}
			})
		}
	}
}
