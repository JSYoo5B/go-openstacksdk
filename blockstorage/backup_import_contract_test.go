package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func bipCall(t *testing.T, client *gophercloud.ServiceClient, service, locator string) (*blockstorage.ImportVolumeBackupResult, error) {
	t.Helper()
	return blockstorage.ImportVolumeBackup(bmwContext(t), client, blockstorage.ImportVolumeBackupRequest{BackupService: service, BackupURL: locator})
}

func bipPayload(t *testing.T, r *http.Request, service, locator, version, token string) {
	t.Helper()
	outer := smwFields(t, bmwRequest(t, r, http.MethodPost, bmwCollection+"/import_record", version, token))
	body := smwFields(t, outer["backup-record"])
	wantService, _ := json.Marshal(service)
	wantURL, _ := json.Marshal(locator)
	want := map[string]json.RawMessage{"backup_service": wantService, "backup_url": wantURL}
	if len(outer) != 1 || !reflect.DeepEqual(body, want) || r.URL.RawQuery != "" || r.Header.Get("X-OpenStack-Volume-API-Version") != version {
		t.Error("import route/literal record changed", r.URL, r.Header, outer, body, want)
	}
}

func bipNullableFields(t *testing.T, value json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	fields := smwFields(t, value)
	expected := []string{"availability_zone", "container", "created_at", "data_timestamp", "description", "encryption_key_id", "fail_reason", "force", "has_dependent_backups", "is_incremental", "links", "metadata", "name", "object_count", "project_id", "size", "snapshot_id", "status", "updated_at", "user_id", "volume_id", "volume_name", "id", "location"}
	if len(fields) != len(expected) {
		t.Fatal("import is not the full nullable Backup", string(value))
	}
	for _, key := range expected {
		if _, present := fields[key]; !present {
			t.Fatal("missing Backup field", key, string(value))
		}
	}
	if string(fields["location"]) != "null" {
		t.Fatal("disconnected import computed a location", string(value))
	}
	return fields
}

func TestImportVolumeBackupRepairsRouteAndForwardsBothLiteralStrings(t *testing.T) {
	cases := []struct {
		name, service, locator string
		code                   int
	}{
		{"ordinary locator", "external-driver", "https://record.invalid/path?raw=%2F#record", 201},
		{"empty strings", "", "", 203},
		{"controls and Unicode", " \tservice\x00", "\nbackup/한글 % value\r", 300},
		{"base64-looking string", "driver/name", "YWJj/+/==", 399},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				bipPayload(t, r, tc.service, tc.locator, "3.60", "test-token")
				w.Header().Set("X-Proof", tc.name)
				testcloud.JSON(w, tc.code, `{"backup":{"id":"imported"}}`)
			})
			result, err := bipCall(t, client, tc.service, tc.locator)
			if err != nil || result == nil || result.Applied == nil || result.Applied.StatusCode != tc.code || result.Applied.Header.Get("X-Proof") != tc.name || result.Value == nil || string(result.BackupID) != `"imported"` || result.Backup == nil || result.Microversion != "3.60" || len(result.Discovery) != 0 || calls.Load() != 1 || client.Microversion != "3.60" {
				t.Fatal(result, err, calls.Load(), client)
			}
			bipNullableFields(t, result.Value)
		})
	}
}

func TestImportVolumeBackupBuildsDisconnectedNullableBackupFromActualKnownFields(t *testing.T) {
	cloud := testcloud.New(t)
	client := bmwClient(cloud)
	reply := `{"backup":{"id":false,"name":"returned","force":"false","has_dependent_backups":0,"is_incremental":[],"links":{},"metadata":[["not","a dict"]],"object_count":[],"size":"0003","os-backup-project-attr:project_id":"earlier","project_id":{"after":true},"availability_zone":["zone"],"location":{"caller":"wire"},"connection":true,"microversion":false,"_synchronized":null,"self":"physical-only","unknown":9007199254740993}}`
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		bipPayload(t, r, "svc", "caller record", "3.60", "test-token")
		w.Header().Set("X-Proof", "actual")
		testcloud.JSON(w, 202, reply)
	})
	result, err := bipCall(t, client, "svc", "caller record")
	if err != nil || result == nil || result.Backup == nil || result.Applied == nil || result.Value == nil || string(result.BackupID) != "false" || calls.Load() != 1 {
		t.Fatal(result, err, calls.Load())
	}
	fields := bipNullableFields(t, result.Value)
	expected := map[string]string{"id": "false", "name": `"returned"`, "force": "true", "has_dependent_backups": "false", "is_incremental": "false", "links": "[{}]", "metadata": "{}", "object_count": "0", "size": "3", "project_id": `{"after":true}`, "availability_zone": `["zone"]`}
	for key, raw := range fields {
		want, present := expected[key]
		if !present {
			want = "null"
		}
		if string(raw) != want {
			t.Fatal("descriptor result", key, string(raw), want, string(result.Value))
		}
	}
	for _, key := range []string{"unknown", "self", "location", "connection", "microversion", "_synchronized"} {
		if _, wrong := fields[key]; wrong && key != "location" {
			t.Fatal("unknown Body entered logical model", key)
		}
		if _, missing := result.Backup.Body[key]; !missing {
			t.Fatal("actual response field lost", key, result.Backup)
		}
	}
	if string(result.Backup.Body["unknown"]) != "9007199254740993" || string(result.Backup.Body["object_count"]) != "[]" || string(result.Backup.Body["metadata"]) != `[["not","a dict"]]` || string(result.Applied.Body) != reply {
		t.Fatal("logical conversion replaced actual response", result.Backup, result.Applied)
	}
}

func TestImportVolumeBackupKeepsArbitraryResponseIDWithoutInventingSeed(t *testing.T) {
	for _, id := range []string{"<missing>", "null", "false", "0", `""`, "[false,1]", `{"n":9007199254740993}`, "9007199254740993"} {
		t.Run(id, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			reply, want := `{"backup":{}}`, "null"
			if id != "<missing>" {
				reply, want = `{"backup":{"id":`+id+`}}`, id
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				bipPayload(t, r, "svc", "not an ID", "3.60", "test-token")
				testcloud.JSON(w, 201, reply)
			})
			result, err := bipCall(t, client, "svc", "not an ID")
			if err != nil || result == nil || result.BackupID == nil || string(result.BackupID) != want || result.Backup == nil || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			fields := bipNullableFields(t, result.Value)
			if string(fields["id"]) != want {
				t.Fatal(string(result.Value), want)
			}
			actual, present := result.Backup.Body["id"]
			if id == "<missing>" && present || id != "<missing>" && (!present || string(actual) != id) {
				t.Fatal("request invented or changed physical id", result.Backup, id)
			}
		})
	}
}

func TestImportVolumeBackupUsesEnvelopePresenceAndOnlyMalformedJSONLeavesDefaults(t *testing.T) {
	cases := []struct {
		name              string
		body              []byte
		success, physical bool
		id                string
	}{
		{"backup wins", []byte(`{"id":"flat","backup":{"id":"envelope"}}`), true, true, `"envelope"`},
		{"flat", []byte(`{"id":"flat","unknown":false}`), true, true, `"flat"`},
		{"genuine empty object", []byte(`{}`), true, true, "null"},
		{"empty", nil, true, false, "null"},
		{"malformed", []byte(`{broken`), true, false, "null"},
		{"null envelope", []byte(`{"backup":null,"id":"unused"}`), false, false, ""},
		{"array envelope", []byte(`{"backup":[]}`), false, false, ""},
		{"null document", []byte(`null`), false, false, ""},
		{"array document", []byte(`[]`), false, false, ""},
		{"scalar document", []byte(`1`), false, false, ""},
		{"descriptor failure", []byte(`{"backup":{"id":"physical","object_count":"²"}}`), false, true, ""},
		{"invalid UTF8", []byte{0xff}, false, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries.Add(1)
				return errors.New("accepted parser must not call native retry")
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				bipPayload(t, r, "svc", "record", "3.60", "test-token")
				w.Header().Set("X-Proof", "current")
				w.WriteHeader(202)
				_, _ = w.Write(tc.body)
			})
			result, err := bipCall(t, client, "svc", "record")
			if result == nil || result.Applied == nil || !bytes.Equal(result.Applied.Body, tc.body) || result.Applied.StatusCode != 202 || result.Applied.Header.Get("X-Proof") != "current" || calls.Load() != 1 || retries.Load() != 0 || (result.Backup != nil) != tc.physical {
				t.Fatal(result, err, calls.Load(), retries.Load())
			}
			if tc.success {
				if err != nil || result.Value == nil || string(result.BackupID) != tc.id {
					t.Fatal(result, err)
				}
				fields := bipNullableFields(t, result.Value)
				if string(fields["id"]) != tc.id {
					t.Fatal(string(result.Value))
				}
			} else {
				var physical *resource.ResponseError
				if err == nil || result.Value != nil || !errors.As(err, &physical) || physical.StatusCode != 202 || !bytes.Equal(physical.Body, tc.body) || physical.Header.Get("X-Proof") != "current" {
					t.Fatal(result, err, physical)
				}
				bmwOperation(t, err, "ImportVolumeBackup")
			}
		})
	}
}

func TestImportVolumeBackupPreflightRejectsOnlyInvalidUTF8AndInvalidCapturedSource(t *testing.T) {
	cases := []string{"nil context", "nil client", "nil provider", "canceled", "wrong role", "invalid service UTF8", "invalid URL UTF8", "source query", "reserved auth header", "conflicting version header"}
	for _, kind := range cases {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			ctx := bmwContext(t)
			service, locator := "svc", "record"
			cause := errors.New("import preflight custom cancel")
			want := resource.ErrInvalidOption
			switch kind {
			case "nil context":
				ctx = nil
			case "nil client":
				client = nil
			case "nil provider":
				client.ProviderClient = nil
			case "canceled":
				child, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx = child
				want = context.Canceled
			case "wrong role":
				client.Type = "compute"
				service = string([]byte{0xff})
				want = resource.ErrUnsupported
			case "invalid service UTF8":
				service = string([]byte{0xff})
			case "invalid URL UTF8":
				locator = string([]byte{0xff})
			case "source query":
				client.ResourceBase += "?caller=1"
			case "reserved auth header":
				client.MoreHeaders["x-auth-token"] = "caller token"
			case "conflicting version header":
				client.MoreHeaders["OpenStack-API-Version"] = "volume 3.64"
			}
			var requests atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				t.Error("invalid input reached HTTP", r.URL)
				w.WriteHeader(500)
			})
			result, err := blockstorage.ImportVolumeBackup(ctx, client, blockstorage.ImportVolumeBackupRequest{BackupService: service, BackupURL: locator})
			var physical *resource.ResponseError
			if result != nil || !errors.Is(err, want) || errors.As(err, &physical) || requests.Load() != 0 {
				t.Fatal(result, err, requests.Load())
			}
			if kind == "canceled" && !errors.Is(err, cause) {
				t.Fatal("custom cause lost", err)
			}
			bmwOperation(t, err, "ImportVolumeBackup")
		})
	}
}

func TestImportVolumeBackupLogicalPhysicalAndSeparateCallsOwnIndependentStorage(t *testing.T) {
	cloud := testcloud.New(t)
	client := bmwClient(cloud)
	reply := []byte(`{"backup":{"id":"owned","metadata":{"a":"b"},"unknown":9007199254740993}}`)
	original := bytes.Clone(reply)
	headers := http.Header{"X-Proof": {"original"}, "X-Multiple": {"one", "two"}}
	var calls atomic.Int32
	cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		bipPayload(t, r, "svc", "record", "3.60", "test-token")
		return &http.Response{StatusCode: 201, Header: headers, Body: io.NopCloser(bytes.NewReader(reply)), Request: r}, nil
	})
	first, err := bipCall(t, client, "svc", "record")
	if err != nil || first == nil || first.Applied == nil || first.Backup == nil || first.Value == nil || first.BackupID == nil {
		t.Fatal(first, err)
	}
	first.Value[0] = '!'
	if string(first.BackupID) != `"owned"` || string(first.Backup.Body["id"]) != `"owned"` || !bytes.Equal(first.Applied.Body, original) {
		t.Fatal("Value shares physical or ID storage", first)
	}
	first.BackupID[0] = '?'
	first.Backup.Body["id"][0] = '^'
	first.Backup.Header.Set("X-Proof", "resource caller")
	first.Applied.Body[0] = '#'
	first.Applied.Header["X-Multiple"][0] = "caller"
	if !bytes.Equal(reply, original) || headers.Get("X-Proof") != "original" || headers.Values("X-Multiple")[0] != "one" {
		t.Fatal("result aliases HTTP fixture", headers, string(reply))
	}
	second, err := bipCall(t, client, "svc", "record")
	if err != nil || second == nil || second.Applied == nil || second.Backup == nil || second.Value == nil || string(second.BackupID) != `"owned"` || string(second.Backup.Body["id"]) != `"owned"` || !bytes.Equal(second.Applied.Body, original) || second.Applied.Header.Values("X-Multiple")[0] != "one" || calls.Load() != 2 {
		t.Fatal(second, err, calls.Load())
	}
	reply[1] = '@'
	headers["X-Multiple"][1] = "later fixture"
	if !bytes.Equal(second.Applied.Body, original) || second.Applied.Header.Values("X-Multiple")[1] != "two" || second.Backup.Header.Get("X-Proof") != "original" {
		t.Fatal("later fixture changes result", second)
	}
	if first.Applied == second.Applied || first.Backup == second.Backup || string(second.Value)[0] != '{' {
		t.Fatal("separate calls share result", first, second)
	}
}

func TestImportVolumeBackupHTTPRejectionNeverBecomesAppliedOrLogicalSuccess(t *testing.T) {
	for _, code := range []int{400, 404, 500} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			reply := []byte{0xff, 'r', 'e', 'j', 'e', 'c', 't', 'e', 'd'}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				bipPayload(t, r, "svc", "record", "3.60", "test-token")
				w.Header().Set("X-Proof", "native")
				w.WriteHeader(code)
				_, _ = w.Write(reply)
			})
			result, err := bipCall(t, client, "svc", "record")
			var native gophercloud.ErrUnexpectedResponseCode
			var physical *resource.ResponseError
			if result == nil || result.Applied != nil || result.Value != nil || result.Backup != nil || len(result.Discovery) != 0 || err == nil || !errors.As(err, &native) || native.Actual != code || !bytes.Equal(native.Body, reply) || native.ResponseHeader.Get("X-Proof") != "native" || errors.As(err, &physical) || calls.Load() != 1 {
				t.Fatal(result, err, native, physical, calls.Load())
			}
			bmwOperation(t, err, "ImportVolumeBackup")
		})
	}
}

func TestImportVolumeBackupComposesWithParsedExportWithoutBase64Transformation(t *testing.T) {
	cloud := testcloud.New(t)
	client := bmwClient(cloud)
	service, locator := "driver-name", "not base64! /\t한글"
	serviceRaw, _ := json.Marshal(service)
	urlRaw, _ := json.Marshal(locator)
	exported := `{"backup-record":{"backup_service":` + string(serviceRaw) + `,"backup_url":` + string(urlRaw) + `,"unknown":"do not forward"},"arbitrary":[false,9007199254740993]}`
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		call := calls.Add(1)
		if call == 1 {
			bmwRequest(t, r, http.MethodGet, bmwCollection+"/source/export_record", "3.60", "test-token")
			testcloud.JSON(w, 200, exported)
		} else if call == 2 {
			bipPayload(t, r, service, locator, "3.60", "test-token")
			testcloud.JSON(w, 201, `{"backup":{"id":"copied"}}`)
		} else {
			t.Error("composition performed resolver/replay", r.URL)
			w.WriteHeader(500)
		}
	})
	record, err := blockstorage.ExportVolumeBackupRecord(bmwContext(t), client, blockstorage.ExportVolumeBackupRecordRequest{BackupID: "source"})
	if err != nil || record == nil || string(record.Value) != exported {
		t.Fatal(record, err)
	}
	outer := smwFields(t, record.Value)
	fields := smwFields(t, outer["backup-record"])
	var callerService, callerURL string
	if err := json.Unmarshal(fields["backup_service"], &callerService); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(fields["backup_url"], &callerURL); err != nil {
		t.Fatal(err)
	}
	result, err := bipCall(t, client, callerService, callerURL)
	if err != nil || result == nil || string(result.BackupID) != `"copied"` || result.Value == nil || calls.Load() != 2 || record.Exported == nil || string(record.Exported.Body) != exported {
		t.Fatal(result, record, err, calls.Load())
	}
}
