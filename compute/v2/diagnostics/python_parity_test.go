package diagnostics_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/diagnostics"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

// pythonDiagnosticsCall records the wire request openstacksdk would also send.
type pythonDiagnosticsCall struct{ method, path, query, version string }

type pythonDiagnosticsTransport func(*http.Request) (*http.Response, error)

func (transport pythonDiagnosticsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func TestPythonServerDiagnosticsGetMatchesProxyRequest(t *testing.T) {
	var calls []pythonDiagnosticsCall
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonDiagnosticsTransport(func(req *http.Request) (*http.Response, error) {
		calls = append(calls, pythonDiagnosticsCall{req.Method, req.URL.Path, req.URL.RawQuery, req.Header.Get("X-OpenStack-Nova-API-Version")})
		// 2.48 returns the normalized keys without an envelope.
		body := `{"state":"running","driver":"libvirt","hypervisor":"kvm","hypervisor_os":"ubuntu","uptime":46664,"config_drive":true,"num_cpus":1,"num_disks":1,"num_nics":1,"memory_details":{"maximum":2048,"used":1024},"cpu_details":[{"id":0,"time":17300000000,"utilisation":15}],"disk_details":[{"read_bytes":262144,"read_requests":112,"write_bytes":5778432,"write_requests":488,"errors_count":1}],"nic_details":[{"mac_address":"01:23:45:67:89:ab","rx_octets":2070139,"tx_octets":140208}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	client.Microversion = "2.48"

	// get_server_diagnostics("s1")
	got, err := diagnostics.New(client).Get(context.Background(), "s1")
	if err != nil || got["state"] != "running" || got["driver"] != "libvirt" || got["hypervisor"] != "kvm" || got["hypervisor_os"] != "ubuntu" || got["uptime"] != float64(46664) || got["config_drive"] != true || got["num_cpus"] != float64(1) {
		t.Fatal(got, err)
	}
	if memory, ok := got["memory_details"].(map[string]any); !ok || memory["maximum"] != float64(2048) {
		t.Fatal(got["memory_details"])
	}
	for _, key := range []string{"cpu_details", "disk_details", "nic_details"} {
		if rows, ok := got[key].([]any); !ok || len(rows) != 1 {
			t.Fatal(key, got[key])
		}
	}
	want := []pythonDiagnosticsCall{{http.MethodGet, "/nova/v2.1/servers/s1/diagnostics", "", "2.48"}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}
