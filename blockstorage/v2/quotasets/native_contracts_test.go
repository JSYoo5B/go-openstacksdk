package quotasets_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeQuotaV2Transport func(*http.Request) (*http.Response, error)

func (transport nativeQuotaV2Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeQuotaV2Wire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeQuotaV2Call struct{ method, path, query, body string }

func nativeQuotaV2API(t *testing.T, calls *[]nativeQuotaV2Call, reply func(*http.Request) *http.Response) *quotasets.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeQuotaV2Transport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeQuotaV2Call{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v2/project/"
	return quotasets.New(client)
}

func nativeQuotaV2Operation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "quotasets" {
		t.Fatal("generated quotasets context", err, wrapped)
	}
}

const nativeQuotaV2Row = `{"id":"p-1","volumes":10,"snapshots":-1,"gigabytes":1000,"per_volume_gigabytes":-1,"backups":0,"backup_gigabytes":1000,"groups":10,"volumes_ssd":5,"gigabytes_ssd":-1}`

func TestNativeQuotaSetV2RoutesBodiesAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeQuotaV2Call
	api := nativeQuotaV2API(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			// Reset answers 200 rather than 202 or 204.
			return nativeQuotaV2Wire(200, "")
		case req.URL.RawQuery == "usage=true":
			return nativeQuotaV2Wire(200, `{"quota_set":{"id":"p-1","volumes":{"in_use":1,"limit":10,"reserved":0,"allocated":0},"gigabytes":{"in_use":5,"limit":-1,"reserved":1},"volumes_ssd":{"in_use":0,"limit":5}}}`)
		}
		return nativeQuotaV2Wire(200, `{"quota_set":`+nativeQuotaV2Row+`}`)
	})
	got, err := api.Get(ctx, "p-1")
	want := quotasets.QuotaSet{ID: "p-1", Volumes: 10, Snapshots: -1, Gigabytes: 1000, PerVolumeGigabytes: -1, Backups: 0, BackupGigabytes: 1000, Groups: 10,
		Extra: map[string]any{"volumes_ssd": float64(5), "gigabytes_ssd": float64(-1)}}
	// Unknown per-type keys are kept in Extra as decoded JSON numbers.
	if err != nil || !reflect.DeepEqual(*got, want) {
		t.Fatalf("%+v %v", got, err)
	}
	if defaults, err := api.GetDefaults(ctx, "p-1"); err != nil || defaults.Volumes != 10 {
		t.Fatal(defaults, err)
	}
	usage, err := api.GetUsage(ctx, "p-1")
	// Per-type usage keys have no place in QuotaUsageSet and are dropped.
	if err != nil || !(usage.ID == "p-1" && usage.Volumes == quotasets.QuotaUsage{InUse: 1, Limit: 10} && usage.Gigabytes == quotasets.QuotaUsage{InUse: 5, Limit: -1, Reserved: 1} && usage.Backups == quotasets.QuotaUsage{}) {
		t.Fatalf("%+v %v", usage, err)
	}
	volumes, zero, unlimited := 1, 0, -1
	if _, err := api.Update(ctx, "p-1", quotasets.UpdateOpts{Volumes: &volumes, Snapshots: &zero, Gigabytes: &unlimited, Force: true, Extra: map[string]any{"volumes_ssd": 3}},
		quotasets.WithUpdateField("x_extension", true)); err != nil {
		t.Fatal(err)
	}
	// Extra is merged after the typed fields, so it can override a typed limit.
	if _, err := api.Update(ctx, "p-1", quotasets.UpdateOpts{Volumes: &volumes, Extra: map[string]any{"volumes": 9}}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "p-1", quotasets.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "p-1"); err != nil {
		t.Fatal(err)
	}
	path := "/cinder/v2/project/os-quota-sets/p-1"
	wantCalls := []nativeQuotaV2Call{
		{http.MethodGet, path, "", ""},
		{http.MethodGet, path + "/defaults", "", ""},
		{http.MethodGet, path, "usage=true", ""},
		{http.MethodPut, path, "", `{"quota_set":{"force":true,"gigabytes":-1,"snapshots":0,"volumes":1,"volumes_ssd":3,"x_extension":true}}`},
		{http.MethodPut, path, "", `{"quota_set":{"volumes":9}}`},
		{http.MethodPut, path, "", `{"quota_set":{}}`},
		{http.MethodDelete, path, "", ""},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeQuotaSetV2StatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*quotasets.API) error
	}{
		{"Get", []int{200}, func(api *quotasets.API) error { _, err := api.Get(ctx, "p-1"); return err }},
		{"GetDefaults", []int{200}, func(api *quotasets.API) error { _, err := api.GetDefaults(ctx, "p-1"); return err }},
		{"GetUsage", []int{200}, func(api *quotasets.API) error { _, err := api.GetUsage(ctx, "p-1"); return err }},
		{"Update", []int{200}, func(api *quotasets.API) error {
			_, err := api.Update(ctx, "p-1", quotasets.UpdateOpts{})
			return err
		}},
		{"Delete", []int{200}, func(api *quotasets.API) error { return api.Delete(ctx, "p-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeQuotaV2Call
				api := nativeQuotaV2API(t, &calls, func(*http.Request) *http.Response { return nativeQuotaV2Wire(code, `{}`) })
				err := call.call(api)
				nativeQuotaV2Operation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("decode", func(t *testing.T) {
		body := `{}`
		var calls []nativeQuotaV2Call
		api := nativeQuotaV2API(t, &calls, func(*http.Request) *http.Response { return nativeQuotaV2Wire(200, body) })
		// A missing quota_set envelope yields a nil quota set without an error.
		if got, err := api.Get(ctx, "p-1"); err != nil || got != nil {
			t.Fatal(got, err)
		}
		if got, err := api.Update(ctx, "p-1", quotasets.UpdateOpts{}); err != nil || got != nil {
			t.Fatal(got, err)
		}
		if got, err := api.GetUsage(ctx, "p-1"); err != nil || !reflect.DeepEqual(got, quotasets.QuotaUsageSet{}) {
			t.Fatal(got, err)
		}
		body = `{"quota_set":{"volumes":"ten"}}`
		_, err := api.GetDefaults(ctx, "p-1")
		nativeQuotaV2Operation(t, err, "GetDefaults")
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeQuotaV2Call
		api := nativeQuotaV2API(t, &calls, func(*http.Request) *http.Response { return nativeQuotaV2Wire(200, `{}`) })
		for name, update := range map[string]func() error{
			"omitted core field": func() error {
				_, err := api.Update(ctx, "p-1", quotasets.UpdateOpts{}, quotasets.WithUpdateField("groups", 1))
				return err
			},
			"extra key": func() error {
				_, err := api.Update(ctx, "p-1", quotasets.UpdateOpts{Extra: map[string]any{"volumes_ssd": 1}}, quotasets.WithUpdateField("volumes_ssd", 2))
				return err
			},
			"nil option": func() error {
				_, err := api.Update(ctx, "p-1", quotasets.UpdateOpts{}, nil)
				return err
			},
		} {
			err := update()
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeQuotaV2Operation(t, err, "Update")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
