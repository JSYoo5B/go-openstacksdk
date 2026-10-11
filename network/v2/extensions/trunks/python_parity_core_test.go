package trunks_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
)

type pythonParityCoreTrunkTransport func(*http.Request) (*http.Response, error)

func (transport pythonParityCoreTrunkTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

// Python get_trunk_subports sends GET trunks/{id}/get_subports and returns
// the sub_ports body; Go GetSubports sends the same GET and decodes each row.
func TestPythonParityCoreTrunkGetSubports(t *testing.T) {
	ctx := context.Background()
	var calls []string
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonParityCoreTrunkTransport(func(req *http.Request) (*http.Response, error) {
		calls = append(calls, req.Method+" "+req.URL.Path+"?"+req.URL.RawQuery)
		body := `{"sub_ports":[{"port_id":"p-2","segmentation_type":"vlan","segmentation_id":100}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	got, err := trunks.New(client).GetSubports(ctx, "tr-1")
	if err != nil || !reflect.DeepEqual(got, []trunks.Subport{{PortID: "p-2", SegmentationType: "vlan", SegmentationID: 100}}) {
		t.Fatal(got, err)
	}
	if !reflect.DeepEqual(calls, []string{"GET /neutron/v2.0/trunks/tr-1/get_subports?"}) {
		t.Fatal(calls)
	}
}
