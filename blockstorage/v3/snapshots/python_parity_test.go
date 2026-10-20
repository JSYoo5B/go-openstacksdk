package snapshots_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonSnapshotCall records the wire request openstacksdk would also send.
type pythonSnapshotCall struct{ method, path, query, body, version string }

type pythonSnapshotTransport func(*http.Request) (*http.Response, error)

func (transport pythonSnapshotTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonSnapshotWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonSnapshotAPI(t *testing.T, microversion string, reply func(*http.Request) *http.Response) (*snapshots.API, *[]pythonSnapshotCall) {
	t.Helper()
	calls := &[]pythonSnapshotCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonSnapshotTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonSnapshotCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("OpenStack-API-Version")})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	client.Microversion = microversion
	return snapshots.New(client), calls
}

const pythonSnapshotBase = "/cinder/v3/project/snapshots"

const pythonSnapshotRow = `{"id":"snap-1","name":"daily","description":"d","volume_id":"vol-1","status":"available","size":10,"metadata":{"k":"v"},"os-extended-snapshot-attributes:progress":"100%","os-extended-snapshot-attributes:project_id":"project","user_id":"user","created_at":"2026-10-11T01:02:03.000000"}`

func TestPythonSnapshotGetCreateUpdateMatchProxyRequests(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonSnapshotAPI(t, "", func(req *http.Request) *http.Response {
		if req.Method == http.MethodPost {
			return pythonSnapshotWire(202, `{"snapshot":`+pythonSnapshotRow+`}`)
		}
		return pythonSnapshotWire(200, `{"snapshot":`+pythonSnapshotRow+`}`)
	})
	// get_snapshot(snapshot)
	got, err := api.Get(ctx, "snap-1")
	if err != nil || got.ID != "snap-1" || got.Name != "daily" || got.Status != "available" || got.ProjectID != "project" || got.Progress != "100%" || got.Metadata["k"] != "v" {
		t.Fatal(got, err)
	}
	// create_snapshot(volume_id=..., name=..., description=..., is_forced=True, metadata=...)
	created, err := api.Create(ctx, snapshots.CreateOpts{VolumeID: "vol-1", Name: "daily", Description: "d", Force: true, Metadata: map[string]string{"k": "v"}})
	if err != nil || created.ID != "snap-1" || created.VolumeID != "vol-1" {
		t.Fatal(created, err)
	}
	// update_snapshot(snapshot, name=..., description=...)
	name, description := "weekly", "d2"
	if _, err := api.Update(ctx, "snap-1", snapshots.UpdateOpts{Name: &name, Description: &description}); err != nil {
		t.Fatal(err)
	}
	want := []pythonSnapshotCall{
		{http.MethodGet, pythonSnapshotBase + "/snap-1", "", "", ""},
		{http.MethodPost, pythonSnapshotBase, "", `{"snapshot":{"description":"d","force":true,"metadata":{"k":"v"},"name":"daily","volume_id":"vol-1"}}`, ""},
		{http.MethodPut, pythonSnapshotBase + "/snap-1", "", `{"snapshot":{"description":"d2","name":"weekly"}}`, ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonSnapshotListDetailsAllProjectsAndQuery(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name  string
		list  func(*snapshots.API) func(func(*snapshots.Snapshot, error) bool)
		path  string
		query url.Values
	}{
		{
			// snapshots() defaults to details=True.
			name: "details", path: pythonSnapshotBase + "/detail", query: url.Values{},
			list: func(api *snapshots.API) func(func(*snapshots.Snapshot, error) bool) {
				return api.ListDetail(ctx)
			},
		},
		{
			// snapshots(all_projects=True, name="daily", volume_id="vol-1") sends all_tenants=True.
			name: "all projects", path: pythonSnapshotBase + "/detail", query: url.Values{"all_tenants": {"True"}, "name": {"daily"}, "volume_id": {"vol-1"}},
			list: func(api *snapshots.API) func(func(*snapshots.Snapshot, error) bool) {
				return api.ListDetail(ctx, snapshots.WithListDetailOptions(snapshots.ListOpts{Name: "daily", VolumeID: "vol-1"}), snapshots.WithListDetailQuery("all_tenants", "True"))
			},
		},
		{
			// snapshots(details=False, limit=1) uses the summary route.
			name: "summary", path: pythonSnapshotBase, query: url.Values{"limit": {"1"}},
			list: func(api *snapshots.API) func(func(*snapshots.Snapshot, error) bool) {
				return api.List(ctx, snapshots.WithListOptions(snapshots.ListOpts{Limit: 1}))
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api, calls := pythonSnapshotAPI(t, "", func(req *http.Request) *http.Response {
				if req.URL.Query().Get("marker") == "snap-1" {
					return pythonSnapshotWire(200, `{"snapshots":[{"id":"snap-2","name":"daily"}]}`)
				}
				return pythonSnapshotWire(200, `{"snapshots":[`+pythonSnapshotRow+`],"snapshots_links":[{"rel":"next","href":"http://`+req.Host+req.URL.Path+`?marker=snap-1"}]}`)
			})
			var ids []string
			for value, err := range tc.list(api) {
				if err != nil {
					t.Fatal(err)
				}
				ids = append(ids, value.ID)
			}
			if !reflect.DeepEqual(ids, []string{"snap-1", "snap-2"}) || len(*calls) != 2 {
				t.Fatalf("%v %+v", ids, *calls)
			}
			first, _ := url.ParseQuery((*calls)[0].query)
			if (*calls)[0].path != tc.path || !reflect.DeepEqual(first, tc.query) || (*calls)[1].path != tc.path || (*calls)[1].query != "marker=snap-1" {
				t.Fatalf("%+v", *calls)
			}
		})
	}
}

func TestPythonSnapshotDeleteIgnoresMissingAndForceUsesAction(t *testing.T) {
	ctx := context.Background()
	status := http.StatusNotFound
	api, calls := pythonSnapshotAPI(t, "", func(req *http.Request) *http.Response {
		if req.Method == http.MethodPost {
			return pythonSnapshotWire(202, "")
		}
		return pythonSnapshotWire(status, `{"itemNotFound":{"message":"gone"}}`)
	})
	// delete_snapshot(snapshot) ignores a missing snapshot by default.
	if err := api.Remove(ctx, resource.ID("snap-1")); err != nil {
		t.Fatal(err)
	}
	// delete_snapshot(snapshot, ignore_missing=False) raises NotFound.
	if err := api.Remove(ctx, resource.ID("snap-1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	// The plain API.Delete keeps the 404 like ignore_missing=False.
	if err := api.Delete(ctx, "snap-1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	status = http.StatusAccepted
	if err := api.Remove(ctx, resource.ID("snap-1")); err != nil {
		t.Fatal(err)
	}
	// delete_snapshot(snapshot, force=True) posts os-force_delete; Python sends null, Gophercloud {}.
	if err := api.ForceDelete(ctx, "snap-1"); err != nil {
		t.Fatal(err)
	}
	want := []pythonSnapshotCall{
		{http.MethodDelete, pythonSnapshotBase + "/snap-1", "", "", ""},
		{http.MethodDelete, pythonSnapshotBase + "/snap-1", "", "", ""},
		{http.MethodDelete, pythonSnapshotBase + "/snap-1", "", "", ""},
		{http.MethodDelete, pythonSnapshotBase + "/snap-1", "", "", ""},
		{http.MethodPost, pythonSnapshotBase + "/snap-1/action", "", `{"os-force_delete":{}}`, ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonSnapshotStatusActionsWithPinnedMicroversion(t *testing.T) {
	ctx := context.Background()
	// Snapshot._action always sends microversion 3.65; Go sends the client's selected version.
	api, calls := pythonSnapshotAPI(t, "3.65", func(*http.Request) *http.Response { return pythonSnapshotWire(202, "") })
	// reset_snapshot_status(snapshot, "error") and its deprecated alias reset_snapshot.
	if err := api.ResetStatus(ctx, "snap-1", snapshots.ResetStatusOpts{Status: "error"}); err != nil {
		t.Fatal(err)
	}
	// set_snapshot_status(snapshot, "creating", progress="50%")
	if err := api.UpdateStatus(ctx, "snap-1", snapshots.UpdateStatusOpts{Status: "creating", Progress: "50%"}); err != nil {
		t.Fatal(err)
	}
	// set_snapshot_status(snapshot, "available") omits progress.
	if err := api.UpdateStatus(ctx, "snap-1", snapshots.UpdateStatusOpts{Status: "available"}); err != nil {
		t.Fatal(err)
	}
	action := pythonSnapshotBase + "/snap-1/action"
	want := []pythonSnapshotCall{
		{http.MethodPost, action, "", `{"os-reset_status":{"status":"error"}}`, "volume 3.65"},
		{http.MethodPost, action, "", `{"os-update_snapshot_status":{"progress":"50%","status":"creating"}}`, "volume 3.65"},
		{http.MethodPost, action, "", `{"os-update_snapshot_status":{"status":"available"}}`, "volume 3.65"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonSnapshotFindHasNoIDThenNameFallback(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonSnapshotAPI(t, "", func(req *http.Request) *http.Response {
		if req.URL.Path == pythonSnapshotBase+"/detail" {
			return pythonSnapshotWire(200, `{"snapshots":[`+pythonSnapshotRow+`]}`)
		}
		return pythonSnapshotWire(404, `{"itemNotFound":{"message":"gone"}}`)
	})
	// find_snapshot("daily") would GET /snapshots/daily and then list /snapshots/detail?name=daily.
	byID, err := api.Find(ctx, resource.ID("daily"), resource.WithIgnoreMissing())
	if err != nil || byID != nil || len(*calls) != 1 {
		t.Fatal(byID, err, *calls)
	}
	byName, err := api.Find(ctx, resource.Name("daily"))
	if err != nil || byName.ID != "snap-1" {
		t.Fatal(byName, err)
	}
	want := []pythonSnapshotCall{
		{http.MethodGet, pythonSnapshotBase + "/daily", "", "", ""},
		{http.MethodGet, pythonSnapshotBase + "/detail", "name=daily", "", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}
