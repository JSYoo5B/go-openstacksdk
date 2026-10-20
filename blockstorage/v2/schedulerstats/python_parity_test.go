package schedulerstats_test

import (
	"context"
	"io"
	"math"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/schedulerstats"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

type pythonPoolsTransport func(*http.Request) (*http.Response, error)

func (transport pythonPoolsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func TestPythonBackendPoolsSendsDetailQuery(t *testing.T) {
	var requests []string
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonPoolsTransport(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.Method+" "+req.URL.Path+"?"+req.URL.RawQuery)
		body := `{"pools":[{"name":"host@lvm#pool","capabilities":{"volume_backend_name":"lvm","vendor_name":"Open Source","storage_protocol":"iSCSI","free_capacity_gb":"infinite","total_capacity_gb":100,"vendor_specific":"dropped"}}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v2/project/"
	api := schedulerstats.New(client)
	// backend_pools() uses base_path /scheduler-stats/get_pools?detail=True.
	var names []string
	for pool, err := range api.List(context.Background(), schedulerstats.WithListQuery("detail", "True")) {
		if err != nil {
			t.Fatal(err)
		}
		if pool.Capabilities.VolumeBackendName != "lvm" || pool.Capabilities.StorageProtocol != "iSCSI" || !math.IsInf(pool.Capabilities.FreeCapacityGB, 1) || pool.Capabilities.TotalCapacityGB != 100 {
			t.Fatal(pool)
		}
		names = append(names, pool.Name)
	}
	// The typed Detail option sends the lowercase value.
	for _, err := range api.List(context.Background(), schedulerstats.WithListOptions(schedulerstats.ListOpts{Detail: true})) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(names) != 1 || names[0] != "host@lvm#pool" || len(requests) != 2 || requests[0] != "GET /cinder/v2/project/scheduler-stats/get_pools?detail=True" || requests[1] != "GET /cinder/v2/project/scheduler-stats/get_pools?detail=true" {
		t.Fatal(names, requests)
	}
}
