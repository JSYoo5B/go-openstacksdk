package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"gophercloudsdk/blockstorage/v3/backups"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestBackupProxyServiceActionsExposeOwnedLocalFactoriesAndCorrectOuterErrorLabels(t *testing.T) {
	volume, name := "volume before", "name before"
	seed := json.RawMessage(`{"id":"literal"}`)
	location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"current"`)}}
	whole := backups.WithRestoreBackupOptions(backups.RestoreBackupOpts{VolumeID: &volume, Name: &name, Seed: seed, Location: &location})
	individualSeed := backups.WithRestoreBackupSeed(seed)
	individualLocation := backups.WithRestoreBackupLocation(location)
	volume = "caller changed"
	name = "caller changed"
	seed[1] = '!'
	location.Project.ID[1] = '!'
	prepared, err := backups.PrepareRestoreBackupOptions(bmwContext(t), whole, backups.WithRestoreBackupVolumeID("literal volume"), backups.WithRestoreBackupName("literal name"), individualSeed, individualLocation)
	if err != nil || prepared.VolumeID == nil || *prepared.VolumeID != "literal volume" || prepared.Name == nil || *prepared.Name != "literal name" || string(prepared.Seed) != `{"id":"literal"}` || prepared.Location == nil || string(prepared.Location.Project.ID) != `"current"` {
		t.Fatal(prepared, err)
	}
	prepared.Seed[1] = '?'
	prepared.Location.Project.ID[1] = '?'
	again, err := backups.PrepareRestoreBackupOptions(bmwContext(t), whole)
	if err != nil || again.VolumeID == nil || again.Name == nil || again.Location == nil || *again.VolumeID != "volume before" || *again.Name != "name before" || string(again.Seed) != `{"id":"literal"}` || string(again.Location.Project.ID) != `"current"` {
		t.Fatal(again, err)
	}
	for _, operation := range []string{"RestoreBackup", "ResetBackupStatus"} {
		for _, kind := range []string{"nil API", "nil context", "custom cancellation", "unsafe ID", "invalid input", "wrong role"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := bmwClient(cloud)
				api := backups.New(client)
				ctx := bmwContext(t)
				id := "literal"
				status := "available"
				options := []backups.RestoreBackupOption{backups.WithRestoreBackupName("destination")}
				want := resource.ErrInvalidOption
				cause := errors.New("service custom cancellation")
				switch kind {
				case "nil API":
					api = nil
				case "nil context":
					ctx = nil
				case "custom cancellation":
					child, cancel := context.WithCancelCause(ctx)
					cancel(cause)
					ctx = child
					want = context.Canceled
				case "unsafe ID":
					id = "a/b"
				case "invalid input":
					options = nil
					status = string([]byte{0xff})
				case "wrong role":
					client.Type = "compute"
					want = resource.ErrUnsupported
				}
				var calls atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					t.Error("service preflight reached HTTP", r.URL)
					w.WriteHeader(500)
				})
				var err error
				if operation == "RestoreBackup" {
					result, cause := api.RestoreBackup(ctx, id, options...)
					if result != nil {
						t.Fatal(result, cause)
					}
					err = cause
				} else {
					result, cause := api.ResetBackupStatus(ctx, id, status)
					if result != nil {
						t.Fatal(result, cause)
					}
					err = cause
				}
				var outer *resource.OperationError
				var physical *resource.ResponseError
				if !errors.Is(err, want) || !errors.As(err, &outer) || outer.Operation != operation || outer.Resource != "backups" || outer.Cause == nil || errors.As(err, &physical) || calls.Load() != 0 {
					t.Fatal(err, outer, calls.Load())
				}
				if kind == "custom cancellation" && !errors.Is(err, cause) {
					t.Fatal("service wrapper lost context.Cause", err)
				}
			})
		}
	}
}

func TestBackupProxyServiceActionsPreserveNativeTypedRestoreAndSelectedVersionReset(t *testing.T) {
	cloud := testcloud.New(t)
	client := bmwClient(cloud)
	api := backups.New(client)
	var calls atomic.Int32
	reply := `{"restore":{"backup_id":"native-id","volume_id":"restored-volume","volume_name":"made-volume","unknown":"physical-only"}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		call := calls.Add(1)
		path, version := bmwCollection+"/literal/restore", "3.60"
		if call >= 3 {
			path = bmwCollection + "/literal/action"
		}
		if call == 4 {
			version = "3.64"
		}
		body := smwFields(t, bmwRequest(t, r, "POST", path, version, "test-token"))
		switch call {
		case 1:
			if len(body) != 1 || !reflect.DeepEqual(smwFields(t, body["restore"]), map[string]json.RawMessage{}) {
				t.Error("native empty RestoreOpts changed", body)
			}
			testcloud.JSON(w, 202, reply)
		case 2:
			if len(body) != 1 || string(smwFields(t, body["restore"])["name"]) != `"destination"` {
				t.Error(body)
			}
			testcloud.JSON(w, 202, reply)
		case 3:
			if len(body) != 1 || string(smwFields(t, body["os-reset_status"])["status"]) != `"available"` {
				t.Error(body)
			}
			w.WriteHeader(202)
		case 4:
			if len(body) != 1 || string(smwFields(t, body["os-reset_status"])["status"]) != `""` {
				t.Error(body)
			}
			w.Header().Set("X-Proof", "new opaque reset")
			w.WriteHeader(203)
			_, _ = w.Write([]byte("not-json"))
		default:
			t.Error("unexpected service replay", r.URL)
			w.WriteHeader(500)
		}
	})
	// The new Source operation requires a target; the native zero options retain
	// their original empty body and independently typed three-field result.
	invalid, err := api.RestoreBackup(bmwContext(t), "literal")
	var outer *resource.OperationError
	if invalid != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &outer) || outer.Operation != "RestoreBackup" || outer.Resource != "backups" || calls.Load() != 0 {
		t.Fatal(invalid, err, calls.Load())
	}
	native, err := api.RestoreFromBackup(bmwContext(t), "literal", backups.RestoreOpts{})
	if err != nil || native == nil || native.BackupID != "native-id" || native.VolumeID != "restored-volume" || native.VolumeName != "made-volume" || calls.Load() != 1 {
		t.Fatal(native, err, calls.Load())
	}
	merged, err := api.RestoreBackup(bmwContext(t), "literal", backups.WithRestoreBackupName("destination"))
	if err != nil || merged == nil || string(merged.BackupID) != `"literal"` || merged.Value == nil || merged.Backup == nil || string(merged.Backup.Body["backup_id"]) != `"native-id"` || calls.Load() != 2 {
		t.Fatal(merged, err, calls.Load())
	}
	view := smwFields(t, merged.Value)
	if len(view) != 24 || string(view["id"]) != `"literal"` || string(view["volume_name"]) != `"made-volume"` || string(view["volume_id"]) != `"restored-volume"` || string(view["name"]) != "null" {
		t.Fatal(string(merged.Value))
	}
	if _, ok := view["backup_id"]; ok {
		t.Fatal("native Restore DTO replaced the full Backup model")
	}
	if err := api.ResetStatus(bmwContext(t), "literal", backups.ResetStatusOpts{Status: "available"}); err != nil || calls.Load() != 3 {
		t.Fatal(err, calls.Load())
	}
	reset, err := api.ResetBackupStatus(bmwContext(t), "literal", "")
	if err != nil || reset == nil || !reset.Completed || reset.Applied == nil || reset.Applied.StatusCode != 203 || !bytes.Equal(reset.Applied.Body, []byte("not-json")) || reset.Applied.Header.Get("X-Proof") != "new opaque reset" || calls.Load() != 4 || client.Microversion != "3.60" {
		t.Fatal(reset, err, calls.Load(), client)
	}
}
