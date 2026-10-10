package backups_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups"
	"github.com/gophercloud/gophercloud/v2"
)

func TestNativeBackupAdminCalls(t *testing.T) {
	ctx := context.Background()
	var calls []nativeBackupCall
	api, _ := nativeBackupAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodGet:
			// backup_url is a base64 JSON string that decodes into bytes.
			return nativeBackupWire(200, `{"backup-record":{"backup_service":"cinder.backup.drivers.swift","backup_url":"AAEC"}}`)
		case req.URL.Path == "/cinder/v2/project/backups/import_record":
			return nativeBackupWire(201, `{"backup":{"id":"bk-9","name":"imported","links":[]}}`)
		}
		return nativeBackupWire(202, "")
	})
	record, err := api.Export(ctx, "bk-1")
	if err != nil || record.BackupService != "cinder.backup.drivers.swift" || !reflect.DeepEqual(record.BackupURL, []byte{0, 1, 2}) {
		t.Fatal(record, err)
	}
	imported, err := api.Import(ctx, backups.ImportOpts{BackupService: "cinder.backup.drivers.swift", BackupURL: []byte{0, 1, 2}}, backups.WithImportField("x_extension", 1))
	if err != nil || imported.ID != "bk-9" || imported.Name != "imported" {
		t.Fatal(imported, err)
	}
	if err := api.ResetStatus(ctx, "bk-1", backups.ResetStatusOpts{Status: "available"}); err != nil {
		t.Fatal(err)
	}
	if err := api.ForceDelete(ctx, "bk-1"); err != nil {
		t.Fatal(err)
	}
	base := "/cinder/v2/project/backups"
	want := []nativeBackupCall{
		{http.MethodGet, base + "/bk-1/export_record", "", ""},
		// The record bytes are serialized back to base64.
		{http.MethodPost, base + "/import_record", "", `{"backup-record":{"backup_service":"cinder.backup.drivers.swift","backup_url":"AAEC","x_extension":1}}`},
		{http.MethodPost, base + "/bk-1/action", "", `{"os-reset_status":{"status":"available"}}`},
		{http.MethodPost, base + "/bk-1/action", "", `{"os-force_delete":{}}`},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeBackupAdminStatusesAndDecode(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*backups.API) error
	}{
		{"Export", []int{200}, func(api *backups.API) error { _, err := api.Export(ctx, "bk-1"); return err }},
		// Import accepts only 201.
		{"Import", []int{201}, func(api *backups.API) error { _, err := api.Import(ctx, backups.ImportOpts{}); return err }},
		{"ResetStatus", []int{202}, func(api *backups.API) error { return api.ResetStatus(ctx, "bk-1", backups.ResetStatusOpts{}) }},
		{"ForceDelete", []int{202}, func(api *backups.API) error { return api.ForceDelete(ctx, "bk-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeBackupCall
				api, _ := nativeBackupAPI(t, &calls, func(*http.Request) *http.Response { return nativeBackupWire(code, `{}`) })
				err := call.call(api)
				nativeBackupOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("decode", func(t *testing.T) {
		var calls []nativeBackupCall
		api, _ := nativeBackupAPI(t, &calls, func(req *http.Request) *http.Response {
			if req.URL.Path == "/cinder/v2/project/backups/bad/export_record" {
				return nativeBackupWire(200, `{"backup-record":{"backup_url":"not base64!"}}`)
			}
			return nativeBackupWire(200, `{}`)
		})
		// A missing record key yields an empty record without an error.
		if record, err := api.Export(ctx, "empty"); err != nil || record.BackupService != "" || record.BackupURL != nil {
			t.Fatal(record, err)
		}
		_, err := api.Export(ctx, "bad")
		nativeBackupOperation(t, err, "Export")
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeBackupCall
		api, _ := nativeBackupAPI(t, &calls, func(*http.Request) *http.Response { return nativeBackupWire(201, `{}`) })
		for operation, err := range map[string]error{
			"Import": func() error {
				_, err := api.Import(ctx, backups.ImportOpts{}, backups.WithImportField("backup_url", "x"))
				return err
			}(),
			"ResetStatus": api.ResetStatus(ctx, "bk-1", backups.ResetStatusOpts{}, nil),
		} {
			nativeBackupOperation(t, err, operation)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
