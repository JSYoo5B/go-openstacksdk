package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestRestoreVolumeBackupSendsOnlyNonemptyLiteralTargetsAndKeepsSelectedMicroversion(t *testing.T) {
	cases := []struct {
		name, volume, targetName, version string
		code                              int
	}{
		{"volume only", "literal/volume % id", "", "3.60", 200},
		{"name only", "", "destination-name", "3.80", 201},
		{"both", "volume-id", "destination-name", "3.60", 202},
		{"literal controls and whitespace", " \tvolume\x00", "\nname\r", "", 203},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			client.Microversion = tc.version
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				outer := smwFields(t, bmwRequest(t, r, "POST", bmwCollection+"/backup:한글/restore", tc.version, "test-token"))
				body := smwFields(t, outer["restore"])
				want := map[string]json.RawMessage{}
				if tc.volume != "" {
					want["volume_id"], _ = json.Marshal(tc.volume)
				}
				if tc.targetName != "" {
					want["name"], _ = json.Marshal(tc.targetName)
				}
				if len(outer) != 1 || !reflect.DeepEqual(body, want) || r.URL.RawQuery != "" || r.URL.EscapedPath() != bmwCollection+"/"+url.PathEscape("backup:한글")+"/restore" || r.Header.Get("X-OpenStack-Volume-API-Version") != tc.version {
					t.Error(outer, body, want, r.URL, r.Header)
				}
				w.Header().Set("X-Proof", tc.name)
				testcloud.JSON(w, tc.code, `{}`)
			})
			result, err := blockstorage.RestoreVolumeBackup(bmwContext(t), client, blockstorage.RestoreVolumeBackupRequest{BackupID: "backup:한글"}, blockstorage.WithRestoreVolumeBackupVolumeID(tc.volume), blockstorage.WithRestoreVolumeBackupName(tc.targetName))
			if err != nil || result == nil || string(result.BackupID) != `"backup:한글"` || result.Value == nil || result.Backup == nil || len(result.Backup.Body) != 0 || calls.Load() != 1 || client.Microversion != tc.version {
				t.Fatal(result, err, calls.Load())
			}
			bmwPage(t, result.Applied, tc.code, `{}`, tc.name)
			view := smwFields(t, result.Value)
			if len(view) != 24 || string(view["id"]) != `"backup:한글"` || string(view["name"]) != "null" || string(view["volume_id"]) != "null" || string(view["volume_name"]) != "null" || string(view["status"]) != "null" {
				t.Fatal("request target was fabricated as a Backup response field", string(result.Value))
			}
		})
	}
}

func TestRestoreVolumeBackupMergesKnownSeedAndResponseWhileKeepingPhysicalObjectAndValueIndependent(t *testing.T) {
	cloud := testcloud.New(t)
	client := bmwClient(cloud)
	seed := json.RawMessage(`{"id":"literal","name":"cached-backup","status":"creating","description":"cached-description","volume_id":"cached-source-volume","force":"false","has_dependent_backups":{},"is_incremental":null,"size":"7","links":"cached-link","metadata":{"precision":900719925474099312345},"project_id":"seed-attribute","os-backup-project-attr:project_id":"seed-wire","availability_zone":"seed-zone","unknown_seed":"ignored"}`)
	cloudName, region, projectName := "configured-cloud", "configured-region", "current-project-name"
	location := resource.CloudLocation{Cloud: &cloudName, RegionName: &region, Zone: json.RawMessage(`"unused configured zone"`), Project: resource.CloudProject{ID: json.RawMessage(`"current"`), Name: &projectName}}
	seedFactory := blockstorage.WithRestoreVolumeBackupSeed(seed)
	locationFactory := blockstorage.WithRestoreVolumeBackupLocation(location)
	seed[1] = '!'
	cloudName = "caller changed"
	location.Project.ID[1] = '!'
	reply := `{"restore":{"backup_id":"native-only-backup-id","volume_id":"restored-volume","volume_name":"made-volume","size":6.9,"object_count":[],"os-backup-project-attr:project_id":"reply-wire","project_id":"reply-attribute","availability_zone":[false,0],"self":false,"connection":true,"microversion":"ignored","_synchronized":null,"unknown":900719925474099312345},"backup":{"name":"must not select"}}`
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body := smwFields(t, bmwRequest(t, r, "POST", bmwCollection+"/literal/restore", "3.60", "test-token"))
		targets := smwFields(t, body["restore"])
		if len(body) != 1 || len(targets) != 2 || string(targets["volume_id"]) != `"destination-volume"` || string(targets["name"]) != `"new-volume-name"` {
			t.Error(body)
		}
		w.Header().Set("X-Proof", "restore selected")
		testcloud.JSON(w, 202, reply)
	})
	result, err := blockstorage.RestoreVolumeBackup(bmwContext(t), client, blockstorage.RestoreVolumeBackupRequest{BackupID: "literal"}, seedFactory, locationFactory, blockstorage.WithRestoreVolumeBackupVolumeID("destination-volume"), blockstorage.WithRestoreVolumeBackupName("new-volume-name"))
	if err != nil || result == nil || result.Backup == nil || result.Value == nil || calls.Load() != 1 || string(result.BackupID) != `"literal"` {
		t.Fatal(result, err, calls.Load())
	}
	bmwPage(t, result.Applied, 202, reply, "restore selected")
	view := smwFields(t, result.Value)
	if len(view) != 24 || string(view["name"]) != `"cached-backup"` || string(view["description"]) != `"cached-description"` || string(view["status"]) != `"creating"` || string(view["force"]) != "true" || string(view["has_dependent_backups"]) != "false" || string(view["is_incremental"]) != "null" || string(view["size"]) != "6" || string(view["object_count"]) != "0" || string(view["links"]) != `["cached-link"]` || string(view["metadata"]) != `{"precision":900719925474099312345}` || string(view["volume_id"]) != `"restored-volume"` || string(view["volume_name"]) != `"made-volume"` || string(view["project_id"]) != `"reply-attribute"` {
		t.Fatal(string(result.Value))
	}
	for _, key := range []string{"backup_id", "unknown", "unknown_seed", "self", "connection", "microversion", "_synchronized", "os-backup-project-attr:project_id"} {
		if _, ok := view[key]; ok {
			t.Fatal("raw or constructor field leaked into normalized Backup", key)
		}
	}
	computed := smwFields(t, view["location"])
	project := smwFields(t, computed["project"])
	if string(computed["cloud"]) != `"configured-cloud"` || string(computed["region_name"]) != `"configured-region"` || string(computed["zone"]) != `[false,0]` || string(project["id"]) != `"reply-attribute"` || string(project["name"]) != "null" {
		t.Fatal(string(view["location"]))
	}
	if _, ok := result.Backup.Body["id"]; ok {
		t.Fatal("physical object invented seeded ID")
	}
	if _, ok := result.Backup.Body["name"]; ok {
		t.Fatal("physical object invented cached name")
	}
	if string(result.Backup.Body["backup_id"]) != `"native-only-backup-id"` || string(result.Backup.Body["size"]) != "6.9" || string(result.Backup.Body["object_count"]) != "[]" || string(result.Backup.Body["unknown"]) != "900719925474099312345" || result.Backup.StatusCode != 202 || result.Backup.Header.Get("X-Proof") != "restore selected" {
		t.Fatal(result.Backup)
	}
	valueBefore := bytes.Clone(result.Value)
	pageBefore := bytes.Clone(result.Applied.Body)
	result.Backup.Body["unknown"][0] = '!'
	result.Backup.Header.Set("X-Proof", "raw changed")
	if !bytes.Equal(result.Value, valueBefore) || !bytes.Equal(result.Applied.Body, pageBefore) || result.Applied.Header.Get("X-Proof") != "restore selected" {
		t.Fatal("raw selected object aliases merged Value or exchange proof")
	}
	result.Value[0] = '!'
	result.Applied.Body[0] = '?'
	result.Applied.Header.Set("X-Proof", "page changed")
	if result.Backup.Header.Get("X-Proof") != "raw changed" || string(result.Backup.Body["volume_name"]) != `"made-volume"` || string(result.BackupID) != `"literal"` {
		t.Fatal("merged Value or proof aliases physical object")
	}
}

func TestRestoreVolumeBackupUsesEnvelopePresenceAndToleratesOnlyMalformedJSONWithoutInventingObject(t *testing.T) {
	cases := []struct {
		name, reply, want string
		code              int
		failed, actual    bool
	}{
		{"restore priority", `{"restore":{"name":"restore"},"backup":{"name":"backup"},"name":"flat"}`, `"restore"`, 200, false, true},
		{"backup fallback", `{"backup":{"name":"backup"},"name":"flat"}`, `"backup"`, 201, false, true},
		{"flat", `{"name":"flat","backup_id":"physical only"}`, `"flat"`, 203, false, true},
		{"empty object", `{}`, `"cached"`, 202, false, true},
		{"empty204", "", `"cached"`, 204, false, false},
		{"malformed", `{"restore":`, `"cached"`, 202, false, false},
		{"trailing invalid JSON", `{} garbage`, `"cached"`, 399, false, false},
		{"null restore blocks backup", `{"restore":null,"backup":{"name":"must not select"}}`, "", 202, true, false},
		{"false restore blocks backup", `{"restore":false,"backup":{}}`, "", 202, true, false},
		{"list restore blocks backup", `{"restore":[],"backup":{}}`, "", 202, true, false},
		{"null backup blocks flat", `{"backup":null,"name":"flat"}`, "", 202, true, false},
		{"null root", `null`, "", 202, true, false},
		{"list root", `[]`, "", 202, true, false},
		{"number root", `1`, "", 202, true, false},
		{"descriptor failure", `{"restore":{"id":false,"size":"²","unknown":"physical"}}`, "", 202, true, true},
		{"invalid UTF8", string([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}), "", 202, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				retries.Add(1)
				return original
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				bmwRequest(t, r, "POST", bmwCollection+"/literal/restore", "3.60", "test-token")
				w.Header().Set("X-Proof", tc.name)
				testcloud.JSON(w, tc.code, tc.reply)
			})
			result, err := blockstorage.RestoreVolumeBackup(bmwContext(t), client, blockstorage.RestoreVolumeBackupRequest{BackupID: "literal"}, blockstorage.WithRestoreVolumeBackupName("destination"), blockstorage.WithRestoreVolumeBackupSeed(json.RawMessage(`{"name":"cached","description":"unchanged"}`)))
			if result == nil || calls.Load() != 1 || retries.Load() != 0 || (result.Backup != nil) != tc.actual {
				t.Fatal(result, err, calls.Load(), retries.Load())
			}
			bmwPage(t, result.Applied, tc.code, tc.reply, tc.name)
			if tc.failed {
				var physical *resource.ResponseError
				if err == nil || result.Value != nil || !errors.As(err, &physical) || physical.StatusCode != tc.code || string(physical.Body) != tc.reply || physical.Header.Get("X-Proof") != tc.name {
					t.Fatal(result, err)
				}
				if tc.actual && (string(result.BackupID) != "false" || string(result.Backup.Body["size"]) != `"²"` || string(result.Backup.Body["unknown"]) != `"physical"`) {
					t.Fatal("descriptor failure lost actual selected object or state ID", result)
				}
				bmwOperation(t, err, "RestoreVolumeBackup")
				result.Applied.Header.Set("X-Proof", "returned page")
				if physical.Header.Get("X-Proof") != tc.name {
					t.Fatal("current failure proof aliases returned page")
				}
				return
			}
			if err != nil || result.Value == nil || string(result.BackupID) != `"literal"` {
				t.Fatal(result, err)
			}
			view := smwFields(t, result.Value)
			if string(view["name"]) != tc.want || string(view["description"]) != `"unchanged"` || len(view) != 24 {
				t.Fatal(string(result.Value))
			}
		})
	}
}

func TestRestoreVolumeBackupRetainsArbitraryResponseIDWithoutChangingFixedRequestTarget(t *testing.T) {
	for _, raw := range []string{`null`, `false`, `0`, `""`, `"different/unsafe % id"`, `[]`, `{"n":9007199254740993}`, `9007199254740993123456789`} {
		t.Run(raw, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var calls atomic.Int32
			reply := `{"restore":{"id":` + raw + `,"status":false,"volume_name":[1,null]}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				bmwRequest(t, r, "POST", bmwCollection+"/literal/restore", "3.60", "test-token")
				testcloud.JSON(w, 202, reply)
			})
			result, err := blockstorage.RestoreVolumeBackup(bmwContext(t), client, blockstorage.RestoreVolumeBackupRequest{BackupID: "literal"}, blockstorage.WithRestoreVolumeBackupVolumeID("destination"))
			if err != nil || result == nil || result.BackupID == nil || string(result.BackupID) != raw || result.Backup == nil || string(result.Backup.Body["id"]) != raw || result.Value == nil || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			view := smwFields(t, result.Value)
			if string(view["id"]) != raw || string(view["status"]) != "false" || string(view["volume_name"]) != `[1,null]` {
				t.Fatal(string(result.Value))
			}
			valueBefore := bytes.Clone(result.Value)
			physicalBefore := bytes.Clone(result.Backup.Body["id"])
			result.BackupID[0] ^= 1
			if !bytes.Equal(result.Value, valueBefore) || !bytes.Equal(result.Backup.Body["id"], physicalBefore) {
				t.Fatal("logical ID aliases merged view or raw ID")
			}
		})
	}
}

func TestResetVolumeBackupStatusForwardsEveryUTF8StringWithForcedMicroversionAndOpaqueAck(t *testing.T) {
	cases := []struct {
		status string
		code   int
		reply  []byte
	}{
		{"", 200, []byte{0xff, 0, 'x'}},
		{"not-a-known-status", 202, []byte(`{broken`)},
		{" AVAILABLE ", 204, nil},
		{"\n\r\x00한글", 304, nil},
		{"error", 399, []byte(`[false,null]`)},
	}
	for _, tc := range cases {
		t.Run(http.StatusText(tc.code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			client.Microversion = "3.80"
			client.MoreHeaders["X-Openstack-Volume-Api-Version"] = "3.80"
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				retries.Add(1)
				return original
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				body := smwFields(t, bmwRequest(t, r, "POST", bmwCollection+"/literal/action", "3.64", "test-token"))
				status := smwFields(t, body["os-reset_status"])
				want, _ := json.Marshal(tc.status)
				if len(body) != 1 || len(status) != 1 || !bytes.Equal(status["status"], want) || r.URL.RawQuery != "" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.64" {
					t.Error(body, r.URL, r.Header)
				}
				w.Header().Set("X-Proof", "opaque reset")
				w.WriteHeader(tc.code)
				_, _ = w.Write(tc.reply)
			})
			result, err := blockstorage.ResetVolumeBackupStatus(bmwContext(t), client, blockstorage.ResetVolumeBackupStatusRequest{BackupID: "literal", Status: tc.status})
			if err != nil || result == nil || result.BackupID != "literal" || !result.Completed || calls.Load() != 1 || retries.Load() != 0 || client.Microversion != "3.80" || client.MoreHeaders["X-Openstack-Volume-Api-Version"] != "3.80" {
				t.Fatal(result, err, calls.Load(), retries.Load(), client)
			}
			bmwPage(t, result.Applied, tc.code, string(tc.reply), "opaque reset")
		})
	}
}

func TestBackupProxyPublicActionsRejectNativeHTTPWithoutBorrowedProofOrLogicalCompletion(t *testing.T) {
	for _, operation := range []string{"RestoreVolumeBackup", "ResetVolumeBackupStatus"} {
		for _, code := range []int{400, 403, 404, 503} {
			t.Run(operation+"/"+http.StatusText(code), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := bmwClient(cloud)
				var calls atomic.Int32
				body := []byte{0xff, '{', 'b', 'a', 'd'}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					path, version := bmwCollection+"/literal/restore", "3.60"
					if operation == "ResetVolumeBackupStatus" {
						path, version = bmwCollection+"/literal/action", "3.64"
					}
					bmwRequest(t, r, "POST", path, version, "test-token")
					w.Header().Set("X-Proof", "native rejected")
					w.WriteHeader(code)
					_, _ = w.Write(body)
				})
				var err error
				if operation == "RestoreVolumeBackup" {
					var result *blockstorage.RestoreVolumeBackupResult
					result, err = blockstorage.RestoreVolumeBackup(bmwContext(t), client, blockstorage.RestoreVolumeBackupRequest{BackupID: "literal"}, blockstorage.WithRestoreVolumeBackupName("destination"))
					if result == nil || string(result.BackupID) != `"literal"` || result.Value != nil || result.Backup != nil || result.Applied != nil {
						t.Fatal(result, err)
					}
				} else {
					var result *blockstorage.ResetVolumeBackupStatusResult
					result, err = blockstorage.ResetVolumeBackupStatus(bmwContext(t), client, blockstorage.ResetVolumeBackupStatusRequest{BackupID: "literal", Status: ""})
					if result == nil || result.BackupID != "literal" || result.Completed || result.Applied != nil {
						t.Fatal(result, err)
					}
				}
				var native gophercloud.ErrUnexpectedResponseCode
				var physical *resource.ResponseError
				if !errors.As(err, &native) || native.Actual != code || !bytes.Equal(native.Body, body) || native.ResponseHeader.Get("X-Proof") != "native rejected" || errors.As(err, &physical) || calls.Load() != 1 {
					t.Fatal(err, calls.Load())
				}
				bmwOperation(t, err, operation)
			})
		}
	}
}

func TestResetVolumeBackupStatusPreflightRejectsUnsafeIDInvalidUTF8AndInvalidSourceBeforeHTTP(t *testing.T) {
	for _, kind := range []string{"empty ID", "path ID", "escaped ID", "unicode space ID", "invalid UTF8 status", "nil context", "canceled custom cause", "nil client", "wrong role", "source query", "reserved auth"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			ctx := bmwContext(t)
			id, status := "literal", ""
			want := resource.ErrInvalidOption
			cause := errors.New("reset preflight custom cancellation")
			switch kind {
			case "empty ID":
				id = ""
			case "path ID":
				id = "a/b"
			case "escaped ID":
				id = "a%2Fb"
			case "unicode space ID":
				id = "a\u00a0b"
			case "invalid UTF8 status":
				status = string([]byte{0xff})
			case "nil context":
				ctx = nil
			case "canceled custom cause":
				child, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx = child
				want = context.Canceled
			case "nil client":
				client = nil
			case "wrong role":
				client.Type = "compute"
				id = "a/b"
				want = resource.ErrUnsupported
			case "source query":
				client.ResourceBase += "?x=1"
			case "reserved auth":
				client.MoreHeaders["x-auth-token"] = "caller"
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				t.Error("reset preflight reached HTTP", r.URL)
				w.WriteHeader(500)
			})
			result, err := blockstorage.ResetVolumeBackupStatus(ctx, client, blockstorage.ResetVolumeBackupStatusRequest{BackupID: id, Status: status})
			var physical *resource.ResponseError
			if result != nil || !errors.Is(err, want) || errors.As(err, &physical) || calls.Load() != 0 {
				t.Fatal(result, err, calls.Load())
			}
			if kind == "canceled custom cause" && !errors.Is(err, cause) {
				t.Fatal("custom context cause lost", err)
			}
			bmwOperation(t, err, "ResetVolumeBackupStatus")
		})
	}
}
