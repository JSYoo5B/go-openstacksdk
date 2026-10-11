package backups_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonBackupCall records the wire request openstacksdk block_storage v2 would also send.
type pythonBackupCall struct{ method, path, query, body, version string }

type pythonBackupTransport func(*http.Request) (*http.Response, error)

func (transport pythonBackupTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonBackupWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonBackupAPI(t *testing.T, reply func(*http.Request) *http.Response) (*backups.API, *[]pythonBackupCall) {
	t.Helper()
	calls := &[]pythonBackupCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonBackupTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonBackupCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("OpenStack-API-Version")})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v2/project/"
	return backups.New(client), calls
}

const pythonBackupBase = "/cinder/v2/project/backups"

const pythonBackupRow = `{"id":"bk-1","name":"nightly","status":"available","volume_id":"vol-1","size":1,"object_count":2,"is_incremental":true,"has_dependent_backups":false,"container":"c","availability_zone":"nova","links":[{"rel":"self","href":"x"}]}`

func TestPythonBackupListDetailsAndQuery(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonBackupAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("marker") != "" {
			return pythonBackupWire(200, `{"backups":[{"id":"bk-2","name":"nightly"}]}`)
		}
		return pythonBackupWire(200, `{"backups":[`+pythonBackupRow+`],"backups_links":[{"rel":"next","href":"http://`+req.Host+req.URL.Path+`?marker=bk-1"}]}`)
	})
	// backups(name=..., status=..., volume_id=..., all_tenants=True, sort_key=..., sort_dir=..., limit=1)
	var ids []string
	for value, err := range api.ListDetail(ctx,
		backups.WithListDetailOptions(backups.ListDetailOpts{Limit: 1}),
		backups.WithListDetailQuery("name", "nightly"),
		backups.WithListDetailQuery("status", "available"),
		backups.WithListDetailQuery("volume_id", "vol-1"),
		backups.WithListDetailQuery("all_tenants", "True"),
		backups.WithListDetailQuery("sort_key", "name"),
		backups.WithListDetailQuery("sort_dir", "asc"),
	) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	// backups(details=False)
	for value, err := range api.List(ctx, backups.WithListOptions(backups.ListOpts{Name: "nightly"})) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"bk-1", "bk-2", "bk-1", "bk-2"}) || len(*calls) != 4 {
		t.Fatalf("%v %+v", ids, *calls)
	}
	first, _ := url.ParseQuery((*calls)[0].query)
	want := url.Values{"limit": {"1"}, "name": {"nightly"}, "status": {"available"}, "volume_id": {"vol-1"}, "all_tenants": {"True"}, "sort_key": {"name"}, "sort_dir": {"asc"}}
	if (*calls)[0].path != pythonBackupBase+"/detail" || !reflect.DeepEqual(first, want) || (*calls)[1].query != "marker=bk-1" {
		t.Fatalf("%+v", *calls)
	}
	if (*calls)[2].path != pythonBackupBase || (*calls)[2].query != "name=nightly" || (*calls)[0].version != "" {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonBackupGetCreateRestoreExportImportAndReset(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonBackupAPI(t, func(req *http.Request) *http.Response {
		switch {
		case strings.HasSuffix(req.URL.Path, "/export_record"):
			return pythonBackupWire(200, `{"backup-record":{"backup_service":"cinder.backup.drivers.swift","backup_url":"eyJpZCI6ICJiay0xIn0="}}`)
		case strings.HasSuffix(req.URL.Path, "/import_record"):
			return pythonBackupWire(201, `{"backup":{"id":"bk-9","name":"imported","links":[]}}`)
		case strings.HasSuffix(req.URL.Path, "/restore"):
			return pythonBackupWire(202, `{"restore":{"backup_id":"bk-1","volume_id":"vol-2","volume_name":"restored"}}`)
		case strings.HasSuffix(req.URL.Path, "/action"):
			return pythonBackupWire(202, "")
		case req.Method == http.MethodPost:
			return pythonBackupWire(202, `{"backup":{"id":"bk-1","name":"nightly","links":[]}}`)
		}
		return pythonBackupWire(200, `{"backup":`+pythonBackupRow+`}`)
	})
	// get_backup(backup)
	got, err := api.Get(ctx, "bk-1")
	if err != nil || got.ID != "bk-1" || got.Name != "nightly" || got.Status != "available" || got.VolumeID != "vol-1" || got.ObjectCount != 2 || !got.IsIncremental || got.Container != "c" || got.AvailabilityZone == nil || *got.AvailabilityZone != "nova" {
		t.Fatal(got, err)
	}
	// create_backup(volume_id="vol-1", name="nightly", is_incremental=True) renames is_incremental.
	created, err := api.Create(ctx, backups.CreateOpts{VolumeID: "vol-1", Name: "nightly", Incremental: true})
	if err != nil || created.ID != "bk-1" {
		t.Fatal(created, err)
	}
	// restore_backup(backup, volume="vol-2", name="restored")
	restored, err := api.RestoreFromBackup(ctx, "bk-1", backups.RestoreOpts{VolumeID: "vol-2", Name: "restored"})
	if err != nil || restored.BackupID != "bk-1" || restored.VolumeID != "vol-2" || restored.VolumeName != "restored" {
		t.Fatal(restored, err)
	}
	// export_backup(backup) and its alias export_record(backup)
	record, err := api.Export(ctx, "bk-1")
	if err != nil || record.BackupService != "cinder.backup.drivers.swift" || string(record.BackupURL) != `{"id": "bk-1"}` {
		t.Fatal(record, err)
	}
	// import_backup(service, url) posts to the import route; the locator is the decoded export record.
	imported, err := api.Import(ctx, backups.ImportOpts{BackupService: record.BackupService, BackupURL: record.BackupURL})
	if err != nil || imported.ID != "bk-9" || imported.Name != "imported" {
		t.Fatal(imported, err)
	}
	// reset_backup_status(backup, "error") and its alias reset_backup(backup, "error")
	if err := api.ResetStatus(ctx, "bk-1", backups.ResetStatusOpts{Status: "error"}); err != nil {
		t.Fatal(err)
	}
	// delete_backup(backup, force=True)
	if err := api.ForceDelete(ctx, "bk-1"); err != nil {
		t.Fatal(err)
	}
	want := []pythonBackupCall{
		{http.MethodGet, pythonBackupBase + "/bk-1", "", "", ""},
		{http.MethodPost, pythonBackupBase, "", `{"backup":{"incremental":true,"name":"nightly","volume_id":"vol-1"}}`, ""},
		{http.MethodPost, pythonBackupBase + "/bk-1/restore", "", `{"restore":{"name":"restored","volume_id":"vol-2"}}`, ""},
		{http.MethodGet, pythonBackupBase + "/bk-1/export_record", "", "", ""},
		{http.MethodPost, pythonBackupBase + "/import_record", "", `{"backup-record":{"backup_service":"cinder.backup.drivers.swift","backup_url":"eyJpZCI6ICJiay0xIn0="}}`, ""},
		{http.MethodPost, pythonBackupBase + "/bk-1/action", "", `{"os-reset_status":{"status":"error"}}`, ""},
		{http.MethodPost, pythonBackupBase + "/bk-1/action", "", `{"os-force_delete":{}}`, ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonBackupDeleteIgnoresMissing(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		code    int
		missing error
		strict  bool
	}{
		{http.StatusAccepted, nil, false},
		{http.StatusNotFound, nil, false},
		{http.StatusNotFound, resource.ErrNotFound, true},
		{http.StatusConflict, nil, false},
	} {
		t.Run(http.StatusText(tc.code), func(t *testing.T) {
			api, calls := pythonBackupAPI(t, func(*http.Request) *http.Response {
				return pythonBackupWire(tc.code, `{}`)
			})
			var options []resource.LookupOption
			if tc.strict {
				options = append(options, resource.WithMissingError())
			}
			// delete_backup(backup) defaults to ignore_missing=True.
			err := api.Remove(ctx, resource.ID("bk-1"), options...)
			switch {
			case tc.code == http.StatusConflict:
				if !gophercloud.ResponseCodeIs(err, http.StatusConflict) {
					t.Fatal(err)
				}
			case tc.missing != nil:
				if !errors.Is(err, tc.missing) {
					t.Fatal(err)
				}
			case err != nil:
				t.Fatal(err)
			}
			if len(*calls) != 1 || (*calls)[0].method != http.MethodDelete || (*calls)[0].path != pythonBackupBase+"/bk-1" {
				t.Fatalf("%+v", *calls)
			}
		})
	}
}

func TestPythonBackupFindHasNoProxyFallback(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonBackupAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Path == pythonBackupBase+"/detail" {
			return pythonBackupWire(200, `{"backups":[`+pythonBackupRow+`]}`)
		}
		return pythonBackupWire(http.StatusForbidden, `{"forbidden":{"message":"no"}}`)
	})
	// find_backup("nightly") treats 400/403/404 on GET as a miss and then lists backups/detail?name=nightly.
	// Go keeps the 403 and its name lookup lists without the name query.
	if _, err := api.Find(ctx, resource.ID("nightly"), resource.WithIgnoreMissing()); !gophercloud.ResponseCodeIs(err, http.StatusForbidden) {
		t.Fatal(err)
	}
	byName, err := api.Find(ctx, resource.Name("nightly"))
	if err != nil || byName.ID != "bk-1" {
		t.Fatal(byName, err)
	}
	if len(*calls) != 2 || (*calls)[0].path != pythonBackupBase+"/nightly" || (*calls)[1].path != pythonBackupBase+"/detail" || (*calls)[1].query != "" {
		t.Fatalf("%+v", *calls)
	}
}
