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

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonBackupCall records the wire request openstacksdk would also send.
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
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	return backups.New(client), calls
}

const pythonBackupBase = "/cinder/v3/project/backups"

const pythonBackupRow = `{"id":"bk-1","name":"nightly","description":"d","volume_id":"vol-1","snapshot_id":"snap-1","container":"c","status":"available","size":10,"object_count":2,"is_incremental":true,"has_dependent_backups":false,"os-backup-project-attr:project_id":"project","metadata":{"k":"v"},"availability_zone":"az1","created_at":"2026-10-11T01:02:03.000000"}`

func TestPythonBackupGetCreateUpdateMatchProxyRequests(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonBackupAPI(t, func(req *http.Request) *http.Response {
		if req.Method == http.MethodPost {
			return pythonBackupWire(202, `{"backup":`+pythonBackupRow+`}`)
		}
		return pythonBackupWire(200, `{"backup":`+pythonBackupRow+`}`)
	})
	// get_backup(backup)
	got, err := api.Get(ctx, "bk-1")
	if err != nil || got.ID != "bk-1" || got.Name != "nightly" || !got.IsIncremental || got.ProjectID != "project" || got.ObjectCount != 2 || (*got.Metadata)["k"] != "v" || *got.AvailabilityZone != "az1" {
		t.Fatal(got, err)
	}
	// create_backup(volume_id=..., name=..., is_incremental=True, force=True, ...): Python renames is_incremental to incremental.
	created, err := api.Create(ctx, backups.CreateOpts{VolumeID: "vol-1", Name: "nightly", Description: "d", Container: "c", Incremental: true, Force: true, SnapshotID: "snap-1", Metadata: map[string]string{"k": "v"}, AvailabilityZone: "az1"})
	if err != nil || created.ID != "bk-1" {
		t.Fatal(created, err)
	}
	// update_backup(backup, name=..., description=...)
	name, description := "weekly", "d2"
	if _, err := api.Update(ctx, "bk-1", backups.UpdateOpts{Name: &name, Description: &description}); err != nil {
		t.Fatal(err)
	}
	want := []pythonBackupCall{
		{http.MethodGet, pythonBackupBase + "/bk-1", "", "", ""},
		{http.MethodPost, pythonBackupBase, "", `{"backup":{"availability_zone":"az1","container":"c","description":"d","force":true,"incremental":true,"metadata":{"k":"v"},"name":"nightly","snapshot_id":"snap-1","volume_id":"vol-1"}}`, ""},
		{http.MethodPut, pythonBackupBase + "/bk-1", "", `{"backup":{"description":"d2","name":"weekly"}}`, ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonBackupListDetailsAndQuery(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name  string
		list  func(*backups.API) func(func(*backups.Backup, error) bool)
		path  string
		query url.Values
	}{
		{
			// backups() defaults to details=True.
			name: "details", path: pythonBackupBase + "/detail", query: url.Values{},
			list: func(api *backups.API) func(func(*backups.Backup, error) bool) { return api.ListDetail(ctx) },
		},
		{
			// backups(name="nightly", status="available", all_projects=True) maps all_projects to all_tenants=True.
			name: "query", path: pythonBackupBase + "/detail", query: url.Values{"name": {"nightly"}, "status": {"available"}, "all_tenants": {"True"}},
			list: func(api *backups.API) func(func(*backups.Backup, error) bool) {
				return api.ListDetail(ctx, backups.WithListDetailQuery("name", "nightly"), backups.WithListDetailQuery("status", "available"), backups.WithListDetailQuery("all_tenants", "True"))
			},
		},
		{
			// backups(details=False, volume_id="vol-1") uses the summary route.
			name: "summary", path: pythonBackupBase, query: url.Values{"volume_id": {"vol-1"}},
			list: func(api *backups.API) func(func(*backups.Backup, error) bool) {
				return api.List(ctx, backups.WithListOptions(backups.ListOpts{VolumeID: "vol-1"}))
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api, calls := pythonBackupAPI(t, func(req *http.Request) *http.Response {
				if req.URL.Query().Get("marker") == "bk-1" {
					return pythonBackupWire(200, `{"backups":[{"id":"bk-2"}]}`)
				}
				return pythonBackupWire(200, `{"backups":[`+pythonBackupRow+`],"backups_links":[{"rel":"next","href":"http://`+req.Host+req.URL.Path+`?marker=bk-1"}]}`)
			})
			var ids []string
			for value, err := range tc.list(api) {
				if err != nil {
					t.Fatal(err)
				}
				ids = append(ids, value.ID)
			}
			if !reflect.DeepEqual(ids, []string{"bk-1", "bk-2"}) || len(*calls) != 2 {
				t.Fatalf("%v %+v", ids, *calls)
			}
			first, _ := url.ParseQuery((*calls)[0].query)
			if (*calls)[0].path != tc.path || !reflect.DeepEqual(first, tc.query) || (*calls)[1].query != "marker=bk-1" {
				t.Fatalf("%+v", *calls)
			}
		})
	}
}

func TestPythonBackupDeleteForceAndResetAlias(t *testing.T) {
	ctx := context.Background()
	status := http.StatusNotFound
	api, calls := pythonBackupAPI(t, func(req *http.Request) *http.Response {
		if req.Method == http.MethodPost {
			return pythonBackupWire(202, "")
		}
		return pythonBackupWire(status, `{"itemNotFound":{"message":"gone"}}`)
	})
	// delete_backup(backup) ignores a missing backup by default.
	if err := api.Remove(ctx, resource.ID("bk-1")); err != nil {
		t.Fatal(err)
	}
	// delete_backup(backup, ignore_missing=False) raises NotFound.
	if err := api.Remove(ctx, resource.ID("bk-1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "bk-1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	status = http.StatusAccepted
	// delete_backup(backup, force=True) posts os-force_delete; Python sends null, Gophercloud {}.
	if err := api.ForceDelete(ctx, "bk-1"); err != nil {
		t.Fatal(err)
	}
	// reset_backup(backup, status) is the deprecated alias of reset_backup_status and pins 3.64 like Backup._action.
	reset, err := api.ResetBackupStatus(ctx, "bk-1", "error")
	if err != nil || reset.BackupID != "bk-1" || !reset.Completed {
		t.Fatal(reset, err)
	}
	want := []pythonBackupCall{
		{http.MethodDelete, pythonBackupBase + "/bk-1", "", "", ""},
		{http.MethodDelete, pythonBackupBase + "/bk-1", "", "", ""},
		{http.MethodDelete, pythonBackupBase + "/bk-1", "", "", ""},
		{http.MethodPost, pythonBackupBase + "/bk-1/action", "", `{"os-force_delete":{}}`, ""},
		{http.MethodPost, pythonBackupBase + "/bk-1/action", "", `{"os-reset_status":{"status":"error"}}`, "volume 3.64"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonBackupFindHasNoIDThenNameFallback(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonBackupAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Path == pythonBackupBase+"/detail" {
			return pythonBackupWire(200, `{"backups":[`+pythonBackupRow+`]}`)
		}
		return pythonBackupWire(404, `{"itemNotFound":{"message":"gone"}}`)
	})
	// find_backup("nightly") would GET /backups/nightly and then list /backups/detail?name=nightly.
	// Go's Find(Name) lists without a name query and matches names locally.
	byID, err := api.Find(ctx, resource.ID("nightly"), resource.WithIgnoreMissing())
	if err != nil || byID != nil || len(*calls) != 1 {
		t.Fatal(byID, err, *calls)
	}
	byName, err := api.Find(ctx, resource.Name("nightly"))
	if err != nil || byName.ID != "bk-1" {
		t.Fatal(byName, err)
	}
	if (*calls)[1].path != pythonBackupBase+"/detail" || (*calls)[1].query != "" {
		t.Fatalf("%+v", *calls)
	}
}
