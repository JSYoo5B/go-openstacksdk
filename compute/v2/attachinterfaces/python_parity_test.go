package attachinterfaces_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/attachinterfaces"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// pythonInterfaceCall records the wire request openstacksdk would also send.
type pythonInterfaceCall struct{ method, path, query, body string }

type pythonInterfaceTransport func(*http.Request) (*http.Response, error)

func (transport pythonInterfaceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonInterfaceWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonInterfaceAPI(t *testing.T, reply func(*http.Request) *http.Response) (*attachinterfaces.API, *[]pythonInterfaceCall) {
	t.Helper()
	calls := &[]pythonInterfaceCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonInterfaceTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonInterfaceCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	client.Microversion = "2.70"
	return attachinterfaces.New(client), calls
}

const pythonInterfaceBase = "/nova/v2.1/servers/s1/os-interface"

const pythonInterfaceRow = `{"port_id":"p1","net_id":"n1","mac_addr":"fa:16:3e:00:00:01","port_state":"ACTIVE","fixed_ips":[{"subnet_id":"sub1","ip_address":"10.0.0.5"}],"tag":"nic1"}`

func TestPythonServerInterfaceCreateGetListDeleteMatchProxyRequests(t *testing.T) {
	ctx := context.Background()
	deleteStatus := http.StatusNotFound
	api, calls := pythonInterfaceAPI(t, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return pythonInterfaceWire(deleteStatus, `{"itemNotFound":{"message":"gone"}}`)
		case req.URL.Path == pythonInterfaceBase && req.Method == http.MethodGet:
			return pythonInterfaceWire(200, `{"interfaceAttachments":[`+pythonInterfaceRow+`]}`)
		}
		return pythonInterfaceWire(200, `{"interfaceAttachment":`+pythonInterfaceRow+`}`)
	})
	check := func(value *attachinterfaces.Interface) {
		t.Helper()
		if value == nil || value.PortID != "p1" || value.NetID != "n1" || value.MACAddr != "fa:16:3e:00:00:01" || value.PortState != "ACTIVE" ||
			!reflect.DeepEqual(value.FixedIPs, []attachinterfaces.FixedIP{{SubnetID: "sub1", IPAddress: "10.0.0.5"}}) {
			t.Fatalf("%+v", value)
		}
	}
	// create_server_interface(server, net_id="n1", fixed_ips=[{"ip_address": ...}], tag="nic1")
	created, err := api.Create(ctx, "s1", attachinterfaces.CreateOpts{NetworkID: "n1", FixedIPs: []attachinterfaces.FixedIP{{IPAddress: "10.0.0.5"}}},
		attachinterfaces.WithCreateField("tag", "nic1"))
	if err != nil {
		t.Fatal(err)
	}
	check(created)
	// create_server_interface(server, port_id="p1")
	if _, err := api.Create(ctx, "s1", attachinterfaces.CreateOpts{PortID: "p1"}); err != nil {
		t.Fatal(err)
	}
	// get_server_interface(port, server=server)
	got, err := api.Get(ctx, "s1", "p1")
	if err != nil {
		t.Fatal(err)
	}
	check(got)
	// server_interfaces(server)
	var listed []*attachinterfaces.Interface
	for value, err := range api.List(ctx, "s1") {
		if err != nil {
			t.Fatal(err)
		}
		listed = append(listed, value)
	}
	if len(listed) != 1 {
		t.Fatal(listed)
	}
	check(listed[0])
	// delete_server_interface(port, server=server) ignores a missing interface by default.
	scope, err := api.InServer(ctx, resource.ID("s1"))
	if err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(ctx, resource.ID("p1")); err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(ctx, resource.ID("p1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	deleteStatus = http.StatusAccepted
	if err := scope.Delete(ctx, resource.ID("p1")); err != nil {
		t.Fatal(err)
	}
	want := []pythonInterfaceCall{
		{http.MethodPost, pythonInterfaceBase, "", `{"interfaceAttachment":{"fixed_ips":[{"ip_address":"10.0.0.5"}],"net_id":"n1","tag":"nic1"}}`},
		{http.MethodPost, pythonInterfaceBase, "", `{"interfaceAttachment":{"port_id":"p1"}}`},
		{http.MethodGet, pythonInterfaceBase + "/p1", "", ""},
		{http.MethodGet, pythonInterfaceBase, "", ""},
		{http.MethodDelete, pythonInterfaceBase + "/p1", "", ""},
		{http.MethodDelete, pythonInterfaceBase + "/p1", "", ""},
		{http.MethodDelete, pythonInterfaceBase + "/p1", "", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}
