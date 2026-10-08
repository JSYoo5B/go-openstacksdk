package blockstorage_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

func TestVolumeBackupReadConcurrentSnapshotAndBackupCallsKeepSchemaPolicyIsolated(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var snapshots, backups atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case backupReadContractDetail:
			backups.Add(1)
			if !reflect.DeepEqual(r.URL.Query()["all_tenants"], []string{"literal"}) {
				t.Error("Backup adopted Snapshot all_projects coercion", r.URL.Query())
			}
			testcloud.JSON(w, 203, backupReadContractPage(`[{"id":"b","force":"false","availability_zone":"backup-zone","object_count":"2","links":false}]`, ""))
		case snapshotReadContractDetail:
			snapshots.Add(1)
			if !reflect.DeepEqual(r.URL.Query()["all_tenants"], []string{"True"}) {
				t.Error("Snapshot adopted literal Backup all_projects", r.URL.Query())
			}
			testcloud.JSON(w, 203, snapshotReadContractPage(`[{"id":"s","force":"false","availability_zone":"ignored-zone","links":false}]`, ""))
		default:
			t.Error("schema route drift", r.URL)
			w.WriteHeader(500)
		}
	})
	const each = 12
	ctx := backupReadContractContext(t)
	failures := make(chan error, each*2)
	var wait sync.WaitGroup
	for i := 0; i < each; i++ {
		wait.Add(2)
		go func() {
			defer wait.Done()
			result, err := blockstorage.ListVolumeBackups(ctx, client, blockstorage.WithVolumeBackupListFilters(json.RawMessage(`{"all_projects":"literal"}`)))
			if err != nil || result == nil || len(result.Backups) != 1 || len(result.Pages) != 1 {
				failures <- fmt.Errorf("Backup failed: result=%v error=%v", result, err)
				return
			}
			var rows []map[string]json.RawMessage
			if err := json.Unmarshal(result.Value, &rows); err != nil || len(rows) != 1 {
				failures <- fmt.Errorf("Backup value invalid: %s error=%v", result.Value, err)
				return
			}
			var location map[string]json.RawMessage
			if err := json.Unmarshal(rows[0]["location"], &location); err != nil || len(rows[0]) != 24 || string(rows[0]["force"]) != "true" || string(rows[0]["object_count"]) != "2" || string(rows[0]["links"]) != "[false]" || string(location["zone"]) != `"backup-zone"` {
				failures <- fmt.Errorf("Backup schema contaminated: %s error=%v", result.Value, err)
				return
			}
			if _, ok := rows[0]["is_forced"]; ok {
				failures <- fmt.Errorf("Snapshot field entered Backup value: %s", result.Value)
			}
		}()
		go func() {
			defer wait.Done()
			result, err := blockstorage.ListVolumeSnapshots(ctx, client, blockstorage.WithVolumeSnapshotListFilters(json.RawMessage(`{"all_projects":"literal"}`)))
			if err != nil || result == nil || len(result.Snapshots) != 1 || len(result.Pages) != 1 {
				failures <- fmt.Errorf("Snapshot failed: result=%v error=%v", result, err)
				return
			}
			var rows []map[string]json.RawMessage
			if err := json.Unmarshal(result.Value, &rows); err != nil || len(rows) != 1 {
				failures <- fmt.Errorf("Snapshot value invalid: %s error=%v", result.Value, err)
				return
			}
			var location map[string]json.RawMessage
			if err := json.Unmarshal(rows[0]["location"], &location); err != nil || len(rows[0]) != 16 || string(rows[0]["is_forced"]) != "false" || string(location["zone"]) != "null" {
				failures <- fmt.Errorf("Snapshot schema contaminated: %s error=%v", result.Value, err)
				return
			}
			if _, ok := rows[0]["force"]; ok {
				failures <- fmt.Errorf("Backup field entered Snapshot value: %s", result.Value)
			}
		}()
	}
	wait.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if snapshots.Load() != each || backups.Load() != each {
		t.Fatal("concurrent schemas restarted or dropped a request", snapshots.Load(), backups.Load())
	}
}

func TestVolumeBackupReadSuccessfulAbsenceIsDistinctFromPresentEmptyMember(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.URL.Path {
		case backupReadContractBasic + "/present":
			testcloud.JSON(w, 203, `{"backup":{}}`)
		case backupReadContractDetail:
			if r.URL.Query().Get("name") != "missing/name" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 203, `{"backups":[],"backups_links":[null]}`)
		default:
			t.Error("unsafe name was used as member route", r.URL)
			w.WriteHeader(500)
		}
	})
	absent, err := blockstorage.GetVolumeBackup(backupReadContractContext(t), client, blockstorage.GetVolumeBackupRequest{NameOrID: "missing/name"})
	if err != nil || absent == nil || absent.Value != nil || absent.Backup != nil || absent.Observed != nil || len(absent.Pages) != 1 {
		t.Fatal(absent, err)
	}
	present, err := blockstorage.GetVolumeBackup(backupReadContractContext(t), client, blockstorage.GetVolumeBackupRequest{NameOrID: "present"})
	if err != nil || present == nil || present.Backup == nil || present.Observed == nil || !present.SeededID || len(present.Backup.Body) != 0 || len(present.Pages) != 0 || string(snapshotReadContractFields(t, present.Value)["id"]) != `"present"` || calls.Load() != 2 {
		t.Fatal(present, err, calls.Load())
	}
}
