package schedulerstats_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/schedulerstats"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

// pythonPoolCall records the wire request openstacksdk would also send.
type pythonPoolCall struct{ method, path, query, body, version string }

type pythonPoolTransport func(*http.Request) (*http.Response, error)

func (transport pythonPoolTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonPoolWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonPoolAPI(t *testing.T, reply func(*http.Request) *http.Response) (*schedulerstats.API, *[]pythonPoolCall) {
	t.Helper()
	calls := &[]pythonPoolCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonPoolTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonPoolCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("OpenStack-API-Version")})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	return schedulerstats.New(client), calls
}

const pythonPoolPath = "/cinder/v3/project/scheduler-stats/get_pools"

const pythonPoolRows = `{"pools":[{"name":"host@lvm#LVM","capabilities":{"volume_backend_name":"lvm","vendor_name":"Open Source","driver_version":"3.0.0","storage_protocol":"iSCSI","total_capacity_gb":100.0,"free_capacity_gb":40.5,"reserved_percentage":0,"multiattach":true,"thin_provisioning_support":true,"custom_key":"kept by python"}}]}`

func TestPythonBackendPoolsDetailQuery(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		options []schedulerstats.ListOption
		query   string
	}{
		// backend_pools() always sends Pools.base_path ?detail=True.
		{"python detail spelling", []schedulerstats.ListOption{schedulerstats.WithListQuery("detail", "True")}, "detail=True"},
		// The typed option sends lowercase true, which Cinder parses as the same boolean.
		{"typed detail", []schedulerstats.ListOption{schedulerstats.WithListOptions(schedulerstats.ListOpts{Detail: true})}, "detail=true"},
		// backend_pools(limit=..., marker=...) adds the default pagination QueryParameters after detail.
		{"pagination keys", []schedulerstats.ListOption{schedulerstats.WithListQuery("detail", "True"), schedulerstats.WithListQuery("limit", "1"), schedulerstats.WithListQuery("marker", "m")}, "detail=True&limit=1&marker=m"},
		// The Go default omits detail, so Cinder returns names only; Python cannot send this request.
		{"go default", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api, calls := pythonPoolAPI(t, func(*http.Request) *http.Response { return pythonPoolWire(200, pythonPoolRows) })
			var rows []schedulerstats.StoragePool
			for value, err := range api.List(ctx, tc.options...) {
				if err != nil {
					t.Fatal(err)
				}
				rows = append(rows, *value)
			}
			if want := []pythonPoolCall{{http.MethodGet, pythonPoolPath, tc.query, "", ""}}; !reflect.DeepEqual(*calls, want) {
				t.Fatalf("%+v", *calls)
			}
			if len(rows) != 1 {
				t.Fatal(rows)
			}
			// Python keeps the whole capabilities dict; Go decodes the typed subset and drops custom_key.
			pool := rows[0]
			c := pool.Capabilities
			if !(pool.Name == "host@lvm#LVM" && c.VolumeBackendName == "lvm" && c.VendorName == "Open Source" && c.DriverVersion == "3.0.0" && c.StorageProtocol == "iSCSI" &&
				c.TotalCapacityGB == 100 && c.FreeCapacityGB == 40.5 && c.Multiattach && c.ThinProvisioningSupport) {
				t.Fatalf("%+v", pool)
			}
		})
	}
}
