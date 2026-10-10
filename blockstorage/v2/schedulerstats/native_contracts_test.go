package schedulerstats_test

import (
	"context"
	"errors"
	"io"
	"math"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/schedulerstats"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativePoolTransport func(*http.Request) (*http.Response, error)

func (transport nativePoolTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativePoolWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativePoolCall struct{ method, path, query, body string }

func nativePoolAPI(t *testing.T, calls *[]nativePoolCall, reply func(*http.Request) *http.Response) (*schedulerstats.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativePoolTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativePoolCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v2/project/"
	return schedulerstats.New(client), cloud
}

func TestNativeSchedulerStatsListRouteAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativePoolCall
	var cloud *testcloud.Cloud
	api, cloud := nativePoolAPI(t, &calls, func(*http.Request) *http.Response {
		// The pool list is a single page, so the link is never followed.
		return nativePoolWire(200, `{"pools":[
			{"name":"host@lvm#LVM","capabilities":{"driver_version":"3.0.0","free_capacity_gb":"infinite","total_capacity_gb":"unknown","allocated_capacity_gb":12.5,"max_over_subscription_ratio":20.0,"reserved_percentage":5,"QoS_support":true,"multiattach":true,"volume_backend_name":"lvm","storage_protocol":"iSCSI","total_volumes":3}},
			{"name":"host@ceph#ceph","capabilities":{"free_capacity_gb":"100","max_over_subscription_ratio":"1.5"}}],
			"pools_links":[{"rel":"next","href":"`+cloud.Server.URL+`/never"}]}`)
	})
	var pools []schedulerstats.StoragePool
	for value, err := range api.List(ctx, schedulerstats.WithListOptions(schedulerstats.ListOpts{Detail: true, TenantID: "p-1"}), schedulerstats.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		pools = append(pools, *value)
	}
	for _, err := range api.List(ctx, schedulerstats.WithListOptions(schedulerstats.ListOpts{Detail: false})) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(pools) != 2 {
		t.Fatal(pools)
	}
	first, second := pools[0].Capabilities, pools[1].Capabilities
	// "infinite" becomes +Inf, while "unknown" and numeric strings become zero.
	if !(pools[0].Name == "host@lvm#LVM" && math.IsInf(first.FreeCapacityGB, 1) && first.TotalCapacityGB == 0 && first.AllocatedCapacityGB == 12.5 &&
		first.MaxOverSubscriptionRatio == "20" && first.ReservedPercentage == 5 && first.QoSSupport && first.Multiattach && first.TotalVolumes == 3) {
		t.Fatalf("%+v", first)
	}
	if !(second.FreeCapacityGB == 0 && second.MaxOverSubscriptionRatio == "1.5") {
		t.Fatalf("%+v", second)
	}
	path := "/cinder/v2/project/scheduler-stats/get_pools"
	// A false Detail is omitted rather than sent as detail=false.
	want := []nativePoolCall{
		{http.MethodGet, path, "detail=true&extra=1&tenant_id=p-1", ""},
		{http.MethodGet, path, "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeSchedulerStatsListStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	t.Run("pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {201, `{}`}, {200, `{"pools":[]}`}, {200, `{}`}, {204, ""}} {
			var calls []nativePoolCall
			api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { return nativePoolWire(tc.code, tc.body) })
			var errs []error
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			switch {
			case (tc.code == 404 || tc.code == 201) && len(errs) == 1 && errors.As(errs[0], &native) && native.Actual == tc.code && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
			case tc.code == 200 && len(errs) == 0:
			case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
			default:
				t.Fatal(tc.code, errs)
			}
			if len(calls) != 1 {
				t.Fatal(calls)
			}
		}
	})
	t.Run("decode", func(t *testing.T) {
		var calls []nativePoolCall
		api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response {
			return nativePoolWire(200, `{"pools":[{"name":"p","capabilities":{"total_volumes":"many"}}]}`)
		})
		var errs []error
		for _, err := range api.List(ctx) {
			errs = append(errs, err)
		}
		if len(errs) != 1 || errs[0] == nil {
			t.Fatal(errs)
		}
	})
	t.Run("nil option", func(t *testing.T) {
		var calls []nativePoolCall
		api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { return nativePoolWire(200, `{}`) })
		for _, err := range api.List(ctx, nil) {
			var wrapped *resource.OperationError
			if !errors.As(err, &wrapped) || wrapped.Operation != "List" || wrapped.Resource != "schedulerstats" {
				t.Fatal(err)
			}
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
