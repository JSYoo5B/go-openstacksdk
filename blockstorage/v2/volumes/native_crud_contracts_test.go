package volumes_test

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
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeVolumeTransport func(*http.Request) (*http.Response, error)

func (transport nativeVolumeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeVolumeWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeVolumeCall struct{ method, path, query, body string }

func nativeVolumeAPI(t *testing.T, calls *[]nativeVolumeCall, reply func(*http.Request) *http.Response) (*volumes.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeVolumeTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeVolumeCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v2/project/"
	return volumes.New(client), cloud
}

func nativeVolumeOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "volumes" {
		t.Fatal("generated volumes context", err, wrapped)
	}
}

const nativeVolumeRow = `{"id":"vol-1","name":"data","status":"available","size":10,"bootable":"false","backup_id":null,"metadata":{"k":"v"},"attachments":[{"id":"vol-1","attachment_id":"att","server_id":"srv","attached_at":"2026-10-11T01:02:03.000000"}],"os-vol-tenant-attr:tenant_id":"project","created_at":"2026-10-11T01:02:03.123456","updated_at":null}`

func TestNativeVolumeRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeVolumeCall
	var cloud *testcloud.Cloud
	api, cloud := nativeVolumeAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeVolumeWire(202, "")
		case req.Method == http.MethodPost:
			return nativeVolumeWire(202, `{"volume":`+nativeVolumeRow+`}`)
		case req.URL.Path == "/cinder/v2/project/volumes/detail":
			return nativeVolumeWire(200, `{"volumes":[`+nativeVolumeRow+`],"volumes_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/volumes?marker=x"}]}`)
		case req.URL.Path == "/other/volumes":
			return nativeVolumeWire(200, `{"volumes":[{"id":"vol-2","metadata":null}]}`)
		}
		return nativeVolumeWire(200, `{"volume":`+nativeVolumeRow+`}`)
	})
	hostID := "0f2b8b8e-1c1a-4c1e-9b1a-0a0b0c0d0e0f"
	created, err := api.Create(ctx, volumes.CreateOpts{Size: 10, Name: "data", ImageID: "img", Metadata: map[string]string{"k": "v"}}, volumes.WithCreateField("x_extension", 1), volumes.WithCreateHintOpts(volumes.SchedulerHintOpts{SameHost: []string{hostID}, Query: "q"}))
	if err != nil || !(created.ID == "vol-1" && created.TenantID == "project" && created.Attachments[0].AttachedAt.Second() == 3 && created.CreatedAt.Nanosecond() == 123456000 && created.UpdatedAt.IsZero()) {
		t.Fatal(created, err)
	}
	// Empty hints add no scheduler key.
	if _, err := api.Create(ctx, volumes.CreateOpts{}, volumes.WithCreateHintOpts(volumes.SchedulerHintOpts{})); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "vol-1")
	if err != nil || got.Bootable != "false" || got.Metadata["k"] != "v" {
		t.Fatal(got, err)
	}
	empty := ""
	if _, err := api.Update(ctx, "vol-1", volumes.UpdateOpts{Description: &empty, Metadata: map[string]string{"a": "b"}}, volumes.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	var rows []*volumes.Volume
	for value, err := range api.List(ctx, volumes.WithListOptions(volumes.ListOpts{Name: "data", Metadata: map[string]string{"k": "v"}, AllTenants: true, Limit: 1}), volumes.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "vol-1" && rows[1].ID == "vol-2" && rows[1].Metadata == nil) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "vol-1", volumes.WithDeleteOptions(volumes.DeleteOpts{Cascade: true}), volumes.WithDeleteQuery("force", "true")); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "vol-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 8 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[4].query)
	deleteQuery, _ := url.ParseQuery(calls[6].query)
	base := "/cinder/v2/project/volumes"
	want := []nativeVolumeCall{
		// Scheduler hints sit beside the volume envelope; extensions go inside it.
		{http.MethodPost, base, "", `{"OS-SCH-HNT:scheduler_hints":{"query":"q","same_host":["` + hostID + `"]},"volume":{"imageRef":"img","metadata":{"k":"v"},"name":"data","size":10,"x_extension":1}}`},
		// v2 Size has no omitempty and its required tag does not reject zero.
		{http.MethodPost, base, "", `{"volume":{"size":0}}`},
		{http.MethodGet, base + "/vol-1", "", ""},
		{http.MethodPut, base + "/vol-1", "", `{"volume":{"description":"","metadata":{"a":"b"},"x_extension":1}}`},
		// The list uses the detail route.
		{http.MethodGet, base + "/detail", calls[4].query, ""},
		{http.MethodGet, "/other/volumes", "marker=x", ""},
		{http.MethodDelete, base + "/vol-1", calls[6].query, ""},
		{http.MethodDelete, base + "/vol-1", "", ""},
	}
	// The metadata filter is a Python-dict-like string; v2 has no bootable filter and false cascade is omitted.
	wantQuery := url.Values{"name": {"data"}, "metadata": {"{'k':'v'}"}, "all_tenants": {"true"}, "limit": {"1"}, "extra": {"1"}}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, wantQuery) || !reflect.DeepEqual(deleteQuery, url.Values{"cascade": {"true"}, "force": {"true"}}) {
		t.Fatalf("%+v %v %v", calls, query, deleteQuery)
	}
}

func TestNativeVolumeStrictStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*volumes.API) error
	}{
		// Create accepts only 202.
		{"Create", []int{202}, func(api *volumes.API) error { _, err := api.Create(ctx, volumes.CreateOpts{}); return err }},
		{"Get", []int{200}, func(api *volumes.API) error { _, err := api.Get(ctx, "vol-1"); return err }},
		{"Update", []int{200}, func(api *volumes.API) error {
			_, err := api.Update(ctx, "vol-1", volumes.UpdateOpts{})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *volumes.API) error { return api.Delete(ctx, "vol-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeVolumeCall
				api, _ := nativeVolumeAPI(t, &calls, func(*http.Request) *http.Response { return nativeVolumeWire(code, `{"volume":{}}`) })
				err := call.call(api)
				nativeVolumeOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope and timestamp decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			// An empty object or null envelope yields a zero volume.
			`{}`:              false,
			`{"volume":null}`: false,
			`{"other":{}}`:    true,
			`{"volume":[]}`:   true,
			// The timestamp layout has no zone, so a trailing Z fails.
			`{"volume":{"created_at":"2026-10-11T01:02:03Z"}}`: true,
		} {
			var calls []nativeVolumeCall
			api, _ := nativeVolumeAPI(t, &calls, func(*http.Request) *http.Response { return nativeVolumeWire(200, body) })
			got, err := api.Get(ctx, "vol-1")
			if wantErr {
				nativeVolumeOperation(t, err, "Get")
			} else if err != nil || got == nil || got.ID != "" {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"volumes":[]}`}, {204, ""}} {
			var calls []nativeVolumeCall
			api, _ := nativeVolumeAPI(t, &calls, func(*http.Request) *http.Response { return nativeVolumeWire(tc.code, tc.body) })
			var errs []error
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			switch {
			case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
			case tc.code == 200 && len(errs) == 0:
			case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
			default:
				t.Fatal(tc.code, errs)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeVolumeCall
		api, _ := nativeVolumeAPI(t, &calls, func(*http.Request) *http.Response { return nativeVolumeWire(202, `{}`) })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			// Host hints must be UUIDs.
			"hint host": {"Create", func() error {
				_, err := api.Create(ctx, volumes.CreateOpts{}, volumes.WithCreateHintOpts(volumes.SchedulerHintOpts{DifferentHost: []string{"host-a"}}))
				return err
			}()},
			"hint instance": {"Create", func() error {
				_, err := api.Create(ctx, volumes.CreateOpts{}, volumes.WithCreateHintOpts(volumes.SchedulerHintOpts{LocalToInstance: "srv"}))
				return err
			}()},
			"core extension": {"Create", func() error {
				_, err := api.Create(ctx, volumes.CreateOpts{}, volumes.WithCreateField("size", 1))
				return err
			}()},
			"update nil option":  {"Update", func() error { _, err := api.Update(ctx, "vol-1", volumes.UpdateOpts{}, nil); return err }()},
			"delete empty query": {"Delete", api.Delete(ctx, "vol-1", volumes.WithDeleteQuery("", "x"))},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeVolumeOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeVolumeOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}

func TestNativeVolumeWaitForStatus(t *testing.T) {
	ctx := context.Background()
	t.Run("polls Get until the exact status", func(t *testing.T) {
		var calls []nativeVolumeCall
		var polls atomic.Int32
		api, _ := nativeVolumeAPI(t, &calls, func(*http.Request) *http.Response {
			if polls.Add(1) == 1 {
				return nativeVolumeWire(200, `{"volume":{"id":"vol-1","status":"creating"}}`)
			}
			return nativeVolumeWire(200, `{"volume":{"id":"vol-1","status":"available"}}`)
		})
		if err := api.WaitForStatus(ctx, "vol-1", "available"); err != nil {
			t.Fatal(err)
		}
		// The first poll is immediate and the next follows the one-second ticker.
		if len(calls) != 2 || calls[1] != (nativeVolumeCall{http.MethodGet, "/cinder/v2/project/volumes/vol-1", "", ""}) {
			t.Fatal(calls)
		}
	})
	t.Run("stops on Get errors", func(t *testing.T) {
		var calls []nativeVolumeCall
		api, _ := nativeVolumeAPI(t, &calls, func(*http.Request) *http.Response { return nativeVolumeWire(404, `{}`) })
		err := api.WaitForStatus(ctx, "vol-1", "available")
		nativeVolumeOperation(t, err, "WaitForStatus")
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != 404 || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
	t.Run("case-sensitive status polls until the deadline", func(t *testing.T) {
		var calls []nativeVolumeCall
		api, _ := nativeVolumeAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeVolumeWire(200, `{"volume":{"id":"vol-1","status":"AVAILABLE"}}`)
		})
		deadline, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()
		err := api.WaitForStatus(deadline, "vol-1", "available")
		nativeVolumeOperation(t, err, "WaitForStatus")
		if !errors.Is(err, context.DeadlineExceeded) || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
}
