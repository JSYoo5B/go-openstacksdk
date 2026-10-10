package snapshots_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeSnapTransport func(*http.Request) (*http.Response, error)

func (transport nativeSnapTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeSnapWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeSnapCall struct{ method, path, query, body string }

func nativeSnapAPI(t *testing.T, calls *[]nativeSnapCall, reply func(*http.Request) *http.Response) (*snapshots.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeSnapTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeSnapCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	return snapshots.New(client), cloud
}

func nativeSnapOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "snapshots" {
		t.Fatal("generated snapshots context", err, wrapped)
	}
}

const nativeSnapRow = `{"id":"snap-1","name":"daily","volume_id":"vol-1","status":"available","size":10,"metadata":{"k":"v"},"os-extended-snapshot-attributes:progress":"100%","os-extended-snapshot-attributes:project_id":"project","consumes_quota":true,"created_at":"2026-10-11T01:02:03.000000","updated_at":null}`

func TestNativeSnapshotRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeSnapCall
	var cloud *testcloud.Cloud
	api, cloud := nativeSnapAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeSnapWire(202, "")
		case req.Method == http.MethodPost:
			return nativeSnapWire(202, `{"snapshot":`+nativeSnapRow+`}`)
		case req.URL.Path == "/cinder/v3/project/snapshots" || req.URL.Path == "/cinder/v3/project/snapshots/detail":
			return nativeSnapWire(200, `{"snapshots":[`+nativeSnapRow+`],"snapshots_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/snapshots?marker=x"}]}`)
		case req.URL.Path == "/other/snapshots":
			return nativeSnapWire(200, `{"snapshots":[{"id":"snap-2","metadata":null}]}`)
		}
		return nativeSnapWire(200, `{"snapshot":`+nativeSnapRow+`}`)
	})
	created, err := api.Create(ctx, snapshots.CreateOpts{VolumeID: "vol-1", Name: "daily", Force: true, Metadata: map[string]string{"k": "v"}}, snapshots.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "snap-1" && created.Progress == "100%" && created.ProjectID == "project" && created.ConsumesQuota && created.CreatedAt.Second() == 3 && created.UpdatedAt.IsZero()) {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "snap-1")
	if err != nil || got.Metadata["k"] != "v" {
		t.Fatal(got, err)
	}
	empty := ""
	if _, err := api.Update(ctx, "snap-1", snapshots.UpdateOpts{Description: &empty}, snapshots.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	collect := func(seq func(func(*snapshots.Snapshot, error) bool)) []string {
		var ids []string
		for value, err := range seq {
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, value.ID)
		}
		return ids
	}
	listed := collect(api.List(ctx, snapshots.WithListOptions(snapshots.ListOpts{VolumeID: "vol-1", AllTenants: true, Limit: 1}), snapshots.WithListQuery("extra", "1")))
	detailed := collect(api.ListDetail(ctx, snapshots.WithListDetailOptions(snapshots.ListOpts{Status: "available"}), snapshots.WithListDetailQuery("extra", "1")))
	if !reflect.DeepEqual(listed, []string{"snap-1", "snap-2"}) || !reflect.DeepEqual(detailed, []string{"snap-1", "snap-2"}) {
		t.Fatal(listed, detailed)
	}
	if err := api.Delete(ctx, "snap-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 8 {
		t.Fatalf("%+v", calls)
	}
	listQuery, _ := url.ParseQuery(calls[3].query)
	detailQuery, _ := url.ParseQuery(calls[5].query)
	base := "/cinder/v3/project/snapshots"
	want := []nativeSnapCall{
		{http.MethodPost, base, "", `{"snapshot":{"force":true,"metadata":{"k":"v"},"name":"daily","volume_id":"vol-1","x_extension":1}}`},
		{http.MethodGet, base + "/snap-1", "", ""},
		{http.MethodPut, base + "/snap-1", "", `{"snapshot":{"description":"","x_extension":1}}`},
		// List uses the summary route and ListDetail the detail route with the same options.
		{http.MethodGet, base, calls[3].query, ""},
		{http.MethodGet, "/other/snapshots", "marker=x", ""},
		{http.MethodGet, base + "/detail", calls[5].query, ""},
		{http.MethodGet, "/other/snapshots", "marker=x", ""},
		{http.MethodDelete, base + "/snap-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(listQuery, url.Values{"volume_id": {"vol-1"}, "all_tenants": {"true"}, "limit": {"1"}, "extra": {"1"}}) || !reflect.DeepEqual(detailQuery, url.Values{"status": {"available"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v %v", calls, listQuery, detailQuery)
	}
}

func TestNativeSnapshotStrictStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*snapshots.API) error
	}{
		// Create accepts only 202.
		{"Create", []int{202}, func(api *snapshots.API) error {
			_, err := api.Create(ctx, snapshots.CreateOpts{VolumeID: "vol-1"})
			return err
		}},
		{"Get", []int{200}, func(api *snapshots.API) error { _, err := api.Get(ctx, "snap-1"); return err }},
		{"Update", []int{200}, func(api *snapshots.API) error {
			_, err := api.Update(ctx, "snap-1", snapshots.UpdateOpts{})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *snapshots.API) error { return api.Delete(ctx, "snap-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeSnapCall
				api, _ := nativeSnapAPI(t, &calls, func(*http.Request) *http.Response { return nativeSnapWire(code, `{"snapshot":{}}`) })
				err := call.call(api)
				nativeSnapOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope and timestamp decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			// A missing or null envelope yields a nil snapshot without an error.
			`{}`:                false,
			`{"snapshot":null}`: false,
			`{"snapshot":[]}`:   true,
			`{"snapshot":{"created_at":"2026-10-11T01:02:03Z"}}`: true,
		} {
			var calls []nativeSnapCall
			api, _ := nativeSnapAPI(t, &calls, func(*http.Request) *http.Response { return nativeSnapWire(200, body) })
			got, err := api.Get(ctx, "snap-1")
			if wantErr {
				nativeSnapOperation(t, err, "Get")
			} else if err != nil || got != nil {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"snapshots":[]}`}, {204, ""}} {
			for name, list := range map[string]func(*snapshots.API) []error{
				"List": func(api *snapshots.API) (errs []error) {
					for _, err := range api.List(ctx) {
						errs = append(errs, err)
					}
					return errs
				},
				"ListDetail": func(api *snapshots.API) (errs []error) {
					for _, err := range api.ListDetail(ctx) {
						errs = append(errs, err)
					}
					return errs
				},
			} {
				var calls []nativeSnapCall
				api, _ := nativeSnapAPI(t, &calls, func(*http.Request) *http.Response { return nativeSnapWire(tc.code, tc.body) })
				errs := list(api)
				var native gophercloud.ErrUnexpectedResponseCode
				switch {
				case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
				case tc.code == 200 && len(errs) == 0:
				case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
				default:
					t.Fatal(name, tc.code, errs)
				}
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeSnapCall
		api, _ := nativeSnapAPI(t, &calls, func(*http.Request) *http.Response { return nativeSnapWire(202, `{}`) })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"volume id": {"Create", func() error { _, err := api.Create(ctx, snapshots.CreateOpts{}); return err }()},
			"core extension": {"Create", func() error {
				_, err := api.Create(ctx, snapshots.CreateOpts{VolumeID: "vol-1"}, snapshots.WithCreateField("force", true))
				return err
			}()},
			"update nil option": {"Update", func() error { _, err := api.Update(ctx, "snap-1", snapshots.UpdateOpts{}, nil); return err }()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeSnapOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeSnapOperation(t, err, "List")
		}
		for _, err := range api.ListDetail(ctx, nil) {
			nativeSnapOperation(t, err, "ListDetail")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}

func TestNativeSnapshotWaitForStatus(t *testing.T) {
	ctx := context.Background()
	t.Run("immediate match", func(t *testing.T) {
		var calls []nativeSnapCall
		api, _ := nativeSnapAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeSnapWire(200, `{"snapshot":{"id":"snap-1","status":"available"}}`)
		})
		if err := api.WaitForStatus(ctx, "snap-1", "available"); err != nil || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
	t.Run("stops on Get errors", func(t *testing.T) {
		var calls []nativeSnapCall
		api, _ := nativeSnapAPI(t, &calls, func(*http.Request) *http.Response { return nativeSnapWire(404, `{}`) })
		err := api.WaitForStatus(ctx, "snap-1", "available")
		nativeSnapOperation(t, err, "WaitForStatus")
		if len(calls) != 1 {
			t.Fatal(calls)
		}
	})
	t.Run("missing envelope panics on the nil snapshot", func(t *testing.T) {
		var calls []nativeSnapCall
		api, _ := nativeSnapAPI(t, &calls, func(*http.Request) *http.Response { return nativeSnapWire(200, `{}`) })
		defer func() {
			if recover() == nil {
				t.Fatal("no panic")
			}
		}()
		_ = api.WaitForStatus(ctx, "snap-1", "available")
	})
	t.Run("case-sensitive status polls until the deadline", func(t *testing.T) {
		var calls []nativeSnapCall
		api, _ := nativeSnapAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeSnapWire(200, `{"snapshot":{"id":"snap-1","status":"AVAILABLE"}}`)
		})
		deadline, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()
		err := api.WaitForStatus(deadline, "snap-1", "available")
		nativeSnapOperation(t, err, "WaitForStatus")
		if !errors.Is(err, context.DeadlineExceeded) || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
}
