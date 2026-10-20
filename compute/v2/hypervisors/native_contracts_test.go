package hypervisors_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeHypervisorTransport func(*http.Request) (*http.Response, error)

func (transport nativeHypervisorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeHypervisorWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeHypervisorCall struct{ method, path, query, body string }

func nativeHypervisorAPI(t *testing.T, calls *[]nativeHypervisorCall, reply func(*http.Request) *http.Response) (*hypervisors.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeHypervisorTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeHypervisorCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	return hypervisors.New(client), cloud
}

func nativeHypervisorOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "hypervisors" {
		t.Fatal("generated hypervisors context", err, wrapped)
	}
}

// 2.88+ row: string UUID id, no cpu_info/free_disk_gb/local_gb, servers from with_servers.
const nativeHypervisorModern = `{"id":"hv-uuid","hypervisor_hostname":"cmp1","hypervisor_type":"QEMU","hypervisor_version":8002000,"state":"up","status":"enabled","host_ip":"10.0.0.1","service":{"host":"cmp1","id":"svc-uuid","disabled_reason":null},"servers":[{"name":"vm","uuid":"s-1"}]}`

// Pre-2.28 row: integer id, cpu_info as an encoded JSON string, scientific-notation version.
const nativeHypervisorLegacy = `{"id":7,"hypervisor_hostname":"cmp2","hypervisor_version":2.012e6,"cpu_info":"{\"arch\":\"x86_64\",\"topology\":{\"cores\":4}}","free_disk_gb":1.5e1,"local_gb":40,"vcpus":8,"service":{"host":"cmp2","id":3}}`

func TestNativeHypervisorRoutesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeHypervisorCall
	var cloud *testcloud.Cloud
	api, cloud := nativeHypervisorAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.URL.Path == "/nova/v2.1/os-hypervisors/detail":
			return nativeHypervisorWire(200, `{"hypervisors":[`+nativeHypervisorModern+`],"hypervisors_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/hv?marker=hv-uuid"}]}`)
		case req.URL.Path == "/other/hv":
			return nativeHypervisorWire(200, `{"hypervisors":[`+nativeHypervisorLegacy+`]}`)
		case strings.HasSuffix(req.URL.Path, "/statistics"):
			return nativeHypervisorWire(200, `{"hypervisor_statistics":{"count":2,"vcpus":16,"vcpus_used":3,"local_gb":80,"free_disk_gb":70,"running_vms":3}}`)
		case strings.HasSuffix(req.URL.Path, "/uptime"):
			return nativeHypervisorWire(200, `{"hypervisor":{"id":7,"hypervisor_hostname":"cmp2","state":"up","status":"enabled","uptime":" 08:32:11 up 93 days"}}`)
		case req.URL.Path == "/nova/v2.1/os-hypervisors/7":
			return nativeHypervisorWire(200, `{"hypervisor":`+nativeHypervisorLegacy+`}`)
		}
		// 2.28-2.87 row: cpu_info as an object.
		return nativeHypervisorWire(200, `{"hypervisor":{"id":"hv-uuid","hypervisor_version":1,"cpu_info":{"vendor":"Intel","features":["vmx"]},"service":{"host":"cmp1","id":"svc-uuid"}}}`)
	})
	limit, marker, pattern, withServers := 1, "m", "cmp", false
	var rows []*hypervisors.Hypervisor
	for value, err := range api.List(ctx, hypervisors.WithListOptions(hypervisors.ListOpts{Limit: &limit, Marker: &marker, HypervisorHostnamePattern: &pattern, WithServers: &withServers}), hypervisors.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if len(rows) != 2 {
		t.Fatal(rows)
	}
	modern, legacy := rows[0], rows[1]
	if modern.ID != "hv-uuid" || modern.HypervisorVersion != 8002000 || modern.Service.ID != "svc-uuid" || modern.Servers == nil || (*modern.Servers)[0].UUID != "s-1" || modern.CPUInfo.Arch != "" || modern.LocalGB != 0 {
		t.Fatalf("%+v", modern)
	}
	if legacy.ID != "7" || legacy.HypervisorVersion != 2012000 || legacy.CPUInfo.Arch != "x86_64" || legacy.CPUInfo.Topology.Cores != 4 || legacy.FreeDiskGB != 15 || legacy.LocalGB != 40 || legacy.Service.ID != "3" || legacy.Servers != nil {
		t.Fatalf("%+v", legacy)
	}
	got, err := api.Get(ctx, "7")
	if err != nil || got.ID != "7" || got.VCPUs != 8 {
		t.Fatal(got, err)
	}
	withServers = true
	ext, err := api.GetExt(ctx, "hv-uuid", hypervisors.WithGetExtOptions(hypervisors.GetOpts{WithServers: &withServers}), hypervisors.WithGetExtQuery("extra", "1"))
	if err != nil || ext.CPUInfo.Vendor != "Intel" || !reflect.DeepEqual(ext.CPUInfo.Features, []string{"vmx"}) {
		t.Fatal(ext, err)
	}
	if _, err := api.GetExt(ctx, "hv-uuid"); err != nil {
		t.Fatal(err)
	}
	stats, err := api.GetStatistics(ctx)
	if err != nil || stats.Count != 2 || stats.VCPUsUsed != 3 || stats.FreeDiskGB != 70 {
		t.Fatal(stats, err)
	}
	uptime, err := api.GetUptime(ctx, "7")
	if err != nil || uptime.ID != "7" || uptime.Uptime != " 08:32:11 up 93 days" {
		t.Fatal(uptime, err)
	}
	base := "/nova/v2.1/os-hypervisors"
	want := []nativeHypervisorCall{
		// A non-nil false pointer is still sent.
		{http.MethodGet, base + "/detail", "extra=1&hypervisor_hostname_pattern=cmp&limit=1&marker=m&with_servers=false", ""},
		{http.MethodGet, "/other/hv", "marker=hv-uuid", ""},
		{http.MethodGet, base + "/7", "", ""},
		{http.MethodGet, base + "/hv-uuid", "extra=1&with_servers=true", ""},
		{http.MethodGet, base + "/hv-uuid", "", ""},
		{http.MethodGet, base + "/statistics", "", ""},
		{http.MethodGet, base + "/7/uptime", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeHypervisorStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name string
		call func(*hypervisors.API) error
	}{
		{"Get", func(api *hypervisors.API) error { _, err := api.Get(ctx, "hv"); return err }},
		{"GetExt", func(api *hypervisors.API) error { _, err := api.GetExt(ctx, "hv"); return err }},
		{"GetStatistics", func(api *hypervisors.API) error { _, err := api.GetStatistics(ctx); return err }},
		{"GetUptime", func(api *hypervisors.API) error { _, err := api.GetUptime(ctx, "hv"); return err }},
	} {
		for _, code := range []int{201, 202, 204, 404} {
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeHypervisorCall
				api, _ := nativeHypervisorAPI(t, &calls, func(*http.Request) *http.Response { return nativeHypervisorWire(code, `{}`) })
				err := call.call(api)
				nativeHypervisorOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("hypervisor decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			// Without the envelope the custom decoder never runs, but a null
			// envelope still reaches it and fails like an empty object.
			`{}`:                  false,
			`{"hypervisor":null}`: true,
			// An object without id or hypervisor_version fails the custom decoder.
			`{"hypervisor":{}}`:                                                true,
			`{"hypervisor":{"id":"hv"}}`:                                       true,
			`{"hypervisor":{"hypervisor_version":1}}`:                          true,
			`{"hypervisor":{"id":true,"hypervisor_version":1}}`:                true,
			`{"hypervisor":{"id":"hv","hypervisor_version":"1"}}`:              true,
			`{"hypervisor":{"id":"hv","hypervisor_version":1,"cpu_info":5}}`:   true,
			`{"hypervisor":{"id":"hv","hypervisor_version":1,"local_gb":"5"}}`: true,
			// A present service object also needs an id.
			`{"hypervisor":{"id":"hv","hypervisor_version":1,"service":{"host":"h"}}}`: true,
			`{"hypervisor":{"id":"hv","hypervisor_version":1,"cpu_info":""}}`:          false,
		} {
			var calls []nativeHypervisorCall
			api, _ := nativeHypervisorAPI(t, &calls, func(*http.Request) *http.Response { return nativeHypervisorWire(200, body) })
			_, err := api.Get(ctx, "hv")
			if wantErr {
				nativeHypervisorOperation(t, err, "Get")
			} else if err != nil {
				t.Fatal(body, err)
			}
		}
	})
	t.Run("uptime decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			`{}`:                         false,
			`{"hypervisor":{}}`:          true,
			`{"hypervisor":{"id":"hv"}}`: false,
		} {
			var calls []nativeHypervisorCall
			api, _ := nativeHypervisorAPI(t, &calls, func(*http.Request) *http.Response { return nativeHypervisorWire(200, body) })
			_, err := api.GetUptime(ctx, "hv")
			if wantErr {
				nativeHypervisorOperation(t, err, "GetUptime")
			} else if err != nil {
				t.Fatal(body, err)
			}
		}
	})
	t.Run("list pager status, empty page, bad row and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"hypervisors":[]}`}, {200, `{"hypervisors":[{"id":"hv"}]}`}, {204, ""}} {
			var calls []nativeHypervisorCall
			api, _ := nativeHypervisorAPI(t, &calls, func(*http.Request) *http.Response { return nativeHypervisorWire(tc.code, tc.body) })
			var errs []error
			var rows int
			for value, err := range api.List(ctx) {
				if value != nil {
					rows++
				}
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			var wrapped *resource.OperationError
			switch {
			case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}) && !errors.As(errs[0], &wrapped):
			case tc.code == 200 && strings.Contains(tc.body, "[]") && len(errs) == 0:
			// A row without hypervisor_version fails the whole page before any row is published.
			case tc.code == 200 && rows == 0 && len(errs) == 1 && errs[0] != nil && strings.Contains(errs[0].Error(), "HypervisorVersion"):
			case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
			default:
				t.Fatal(tc.code, tc.body, errs)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeHypervisorCall
		api, _ := nativeHypervisorAPI(t, &calls, func(*http.Request) *http.Response { return nativeHypervisorWire(200, `{}`) })
		_, err := api.GetExt(ctx, "hv", nil)
		nativeHypervisorOperation(t, err, "GetExt")
		_, err = api.GetExt(ctx, "hv", hypervisors.WithGetExtQuery(" ", "x"))
		nativeHypervisorOperation(t, err, "GetExt")
		for _, err := range api.List(ctx, nil) {
			nativeHypervisorOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
