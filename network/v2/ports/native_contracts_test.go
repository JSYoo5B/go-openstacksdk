package ports_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativePortTransport func(*http.Request) (*http.Response, error)

func (transport nativePortTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativePortWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func nativePortClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return client
}

func nativePortOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "ports" {
		t.Fatal("generated ports context", err, wrapped)
	}
}

type nativePortCall struct{ method, path, query, body string }

func nativePortRecorder(cloud *testcloud.Cloud, calls *[]nativePortCall, reply func(*http.Request) *http.Response) {
	cloud.Provider.HTTPClient.Transport = nativePortTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativePortCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
}

const nativePortRow = `{"id":"id-1","network_id":"n1","name":"p","status":"ACTIVE","mac_address":"fa:16:3e:00:00:01","fixed_ips":[{"subnet_id":"sn","ip_address":"10.0.0.5"}],"security_groups":["sg"],"created_at":"2026-10-10T01:02:03Z"}`

func TestNativePortRoutesBodiesPagingAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativePortCall
	nativePortRecorder(cloud, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativePortWire(204, "")
		case req.Method == http.MethodPost:
			return nativePortWire(201, `{"port":`+nativePortRow+`}`)
		case req.URL.Path == "/neutron/v2.0/ports" && req.URL.Query().Get("marker") == "":
			return nativePortWire(200, `{"ports":[`+nativePortRow+`],"ports_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/ports?marker=x"}]}`)
		case req.URL.Path == "/other/ports":
			return nativePortWire(200, `{"ports":[{"id":"id-2","created_at":"2026-10-10T01:02:03"}]}`)
		}
		return nativePortWire(200, `{"port":`+nativePortRow+`}`)
	})
	api := ports.New(nativePortClient(cloud))
	ctx := context.Background()
	created, err := api.Create(ctx, ports.CreateOpts{NetworkID: "n1", Name: "p", FixedIPs: []ports.IP{{SubnetID: "sn", IPAddress: "10.0.0.5"}}, SecurityGroups: &[]string{"sg"}, ValueSpecs: &map[string]string{"binding:host_id": "h"}}, ports.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "id-1" && created.MACAddress == "fa:16:3e:00:00:01" && created.FixedIPs[0].IPAddress == "10.0.0.5") {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "id-1")
	if err != nil || !(got.Status == "ACTIVE" && reflect.DeepEqual(got.SecurityGroups, []string{"sg"})) {
		t.Fatal(got, err)
	}
	var rows []*ports.Port
	for value, err := range api.List(ctx, ports.WithListOptions(ports.ListOpts{NetworkID: "n1", DeviceOwner: "compute:nova", FixedIPs: []ports.FixedIPOpts{{IPAddress: "10.0.0.5", SubnetID: "sn"}}, SecurityGroups: []string{"sg1", "sg2"}}), ports.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "id-1" && rows[1].ID == "id-2" && rows[1].CreatedAt.Second() == 3) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "id-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 5 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[2].query)
	want := []nativePortCall{
		{http.MethodPost, "/neutron/v2.0/ports", "", `{"port":{"binding:host_id":"h","fixed_ips":[{"ip_address":"10.0.0.5","subnet_id":"sn"}],"name":"p","network_id":"n1","security_groups":["sg"],"x_extension":1}}`},
		{http.MethodGet, "/neutron/v2.0/ports/id-1", "", ""},
		{http.MethodGet, "/neutron/v2.0/ports", calls[2].query, ""},
		// The next href is followed as received.
		{http.MethodGet, "/other/ports", "marker=x", ""},
		{http.MethodDelete, "/neutron/v2.0/ports/id-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"network_id": {"n1"}, "device_owner": {"compute:nova"}, "fixed_ips": {"ip_address=10.0.0.5", "subnet_id=sn"}, "security_groups": {"sg1", "sg2"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativePortStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*ports.API) error
	}{
		{"Create", []int{201, 202}, func(api *ports.API) error { _, err := api.Create(ctx, ports.CreateOpts{NetworkID: "n1"}); return err }},
		{"Get", []int{200}, func(api *ports.API) error { _, err := api.Get(ctx, "id-1"); return err }},
		{"Delete", []int{202, 204}, func(api *ports.API) error { return api.Delete(ctx, "id-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls []nativePortCall
				nativePortRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativePortWire(code, `{"port":{}}`) })
				err := call.call(ports.New(nativePortClient(cloud)))
				nativePortOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"ports":[]}`}, {204, ""}} {
			cloud := testcloud.New(t)
			var calls []nativePortCall
			nativePortRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativePortWire(tc.code, tc.body) })
			var errs []error
			for _, err := range ports.New(nativePortClient(cloud)).List(ctx) {
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
		cloud := testcloud.New(t)
		var calls []nativePortCall
		nativePortRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativePortWire(201, `{}`) })
		api := ports.New(nativePortClient(cloud))
		for name, check := range map[string]func() error{
			"missing network": func() error { _, err := api.Create(ctx, ports.CreateOpts{Name: "p"}); return err },
			"banned value spec": func() error {
				_, err := api.Create(ctx, ports.CreateOpts{NetworkID: "n", ValueSpecs: &map[string]string{"tenant_id": "x"}})
				return err
			},
			"value spec overwrite": func() error {
				_, err := api.Create(ctx, ports.CreateOpts{NetworkID: "n", Name: "p", ValueSpecs: &map[string]string{"name": "x"}})
				return err
			},
			"network_id extension": func() error {
				_, err := api.Create(ctx, ports.CreateOpts{NetworkID: "n"}, ports.WithCreateField("network_id", "x"))
				return err
			},
			"nil option": func() error { _, err := api.Create(ctx, ports.CreateOpts{NetworkID: "n"}, nil); return err },
		} {
			err := check()
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativePortOperation(t, err, "Create")
		}
		for _, err := range api.List(ctx, nil) {
			nativePortOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}

func contains(values []int, value int) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
