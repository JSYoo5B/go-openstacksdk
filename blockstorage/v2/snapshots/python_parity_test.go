package snapshots_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pythonSnapshotCall struct {
	method, path string
	query        url.Values
	body         map[string]any
}

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

func pythonSnapshotAPI(t *testing.T, calls *[]pythonSnapshotCall, reply func(*http.Request) *http.Response) (*snapshots.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonSnapshotTransport(func(req *http.Request) (*http.Response, error) {
		call := pythonSnapshotCall{method: req.Method, path: req.URL.Path, query: req.URL.Query()}
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			if len(data) != 0 {
				if err := json.Unmarshal(data, &call.body); err != nil {
					t.Errorf("request body %q: %v", data, err)
				}
			}
		}
		*calls = append(*calls, call)
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v2/project/"
	return snapshots.New(client), cloud
}

func pythonSnapshotExpect(t *testing.T, call pythonSnapshotCall, method, path, query, body string) {
	t.Helper()
	if call.method != method || call.path != path {
		t.Fatalf("got %s %s, want %s %s", call.method, call.path, method, path)
	}
	wantQuery, _ := url.ParseQuery(query)
	if (len(call.query) != 0 || len(wantQuery) != 0) && !reflect.DeepEqual(call.query, wantQuery) {
		t.Fatalf("query %v, want %v", call.query, wantQuery)
	}
	if body == "" {
		if call.body != nil {
			t.Fatalf("unexpected body %v", call.body)
		}
		return
	}
	var want map[string]any
	if err := json.Unmarshal([]byte(body), &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(call.body, want) {
		t.Fatalf("body %v, want %v", call.body, want)
	}
}

const pythonSnapshotRow = `{"id":"snap-1","name":"nightly","description":"d","volume_id":"vol-1","status":"available","size":10,"metadata":{"k":"v"},"created_at":"2026-10-11T01:02:03.000000"}`

func TestPythonSnapshotGetCreateMatchProxyRequests(t *testing.T) {
	ctx := context.Background()
	var calls []pythonSnapshotCall
	api, _ := pythonSnapshotAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.Method == http.MethodPost {
			return pythonSnapshotWire(202, `{"snapshot":`+pythonSnapshotRow+`}`)
		}
		return pythonSnapshotWire(200, `{"snapshot":`+pythonSnapshotRow+`}`)
	})
	got, err := api.Get(ctx, "snap-1")
	if err != nil || !(got.ID == "snap-1" && got.Name == "nightly" && got.VolumeID == "vol-1" && got.Status == "available" && got.Size == 10 && got.Metadata["k"] == "v" && got.CreatedAt.Second() == 3) {
		t.Fatal(got, err)
	}
	pythonSnapshotExpect(t, calls[0], http.MethodGet, "/cinder/v2/project/snapshots/snap-1", "", "")

	// create_snapshot(volume_id=..., name=..., description=..., is_forced=True, metadata=...)
	created, err := api.Create(ctx, snapshots.CreateOpts{VolumeID: "vol-1", Name: "nightly", Description: "d", Force: true, Metadata: map[string]string{"k": "v"}}, snapshots.WithCreateField("x_extension", 1))
	if err != nil || created.ID != "snap-1" {
		t.Fatal(created, err)
	}
	pythonSnapshotExpect(t, calls[1], http.MethodPost, "/cinder/v2/project/snapshots", "", `{"snapshot":{"volume_id":"vol-1","name":"nightly","description":"d","force":true,"metadata":{"k":"v"},"x_extension":1}}`)

	// Python is_forced=False sends "force": false; Go omits it and cannot add it.
	if _, err := api.Create(ctx, snapshots.CreateOpts{VolumeID: "vol-1"}); err != nil {
		t.Fatal(err)
	}
	pythonSnapshotExpect(t, calls[2], http.MethodPost, "/cinder/v2/project/snapshots", "", `{"snapshot":{"volume_id":"vol-1"}}`)
	if _, err := api.Create(ctx, snapshots.CreateOpts{VolumeID: "vol-1"}, snapshots.WithCreateField("force", false)); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 3 {
		t.Fatal(err, len(calls))
	}
}

func TestPythonSnapshotDeleteIgnoreMissingDefault(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		status int
		ignore bool
	}{{"deleted", 202, true}, {"no content", 204, true}, {"missing", 404, true}, {"conflict", 409, false}} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonSnapshotCall
			api, _ := pythonSnapshotAPI(t, &calls, func(*http.Request) *http.Response { return pythonSnapshotWire(tc.status, `{}`) })
			if err := api.Remove(ctx, resource.ID("snap-1")); (err == nil) != tc.ignore {
				t.Fatal(err)
			}
			pythonSnapshotExpect(t, calls[0], http.MethodDelete, "/cinder/v2/project/snapshots/snap-1", "", "")
			err := api.Remove(ctx, resource.ID("snap-1"), resource.WithMissingError())
			if tc.status == 404 && !errors.Is(err, resource.ErrNotFound) || tc.status < 300 && err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPythonSnapshotListAndFindUseSummaryRoute(t *testing.T) {
	ctx := context.Background()
	var calls []pythonSnapshotCall
	var cloud *testcloud.Cloud
	api, cloud := pythonSnapshotAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path == "/cinder/v2/project/snapshots/nightly" {
			return pythonSnapshotWire(404, `{"itemNotFound":{}}`)
		}
		return pythonSnapshotWire(200, `{"snapshots":[`+pythonSnapshotRow+`],"snapshots_links":[{"rel":"next","href":"`+cloud.Server.URL+`/next"}]}`)
	})
	var ids []string
	for value, err := range api.List(ctx, snapshots.WithListQuery("all_tenants", "True"), snapshots.WithListQuery("name", "nightly"), snapshots.WithListQuery("volume_id", "vol-1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	// Python details=False lists /snapshots and follows snapshots_links; Go reads one page.
	if !reflect.DeepEqual(ids, []string{"snap-1"}) || len(calls) != 1 {
		t.Fatal(ids, calls)
	}
	pythonSnapshotExpect(t, calls[0], http.MethodGet, "/cinder/v2/project/snapshots", "all_tenants=True&name=nightly&volume_id=vol-1", "")

	byID, err := api.Find(ctx, resource.ID("nightly"), resource.WithIgnoreMissing())
	if err != nil || byID != nil {
		t.Fatal(byID, err)
	}
	byName, err := api.Find(ctx, resource.Name("nightly"))
	if err != nil || byName == nil || byName.ID != "snap-1" {
		t.Fatal(byName, err)
	}
	pythonSnapshotExpect(t, calls[1], http.MethodGet, "/cinder/v2/project/snapshots/nightly", "", "")
	// Python find_snapshot(details=True) lists /snapshots/detail?name=nightly instead.
	pythonSnapshotExpect(t, calls[2], http.MethodGet, "/cinder/v2/project/snapshots", "name=nightly", "")
}
