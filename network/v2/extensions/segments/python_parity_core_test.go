package segments_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pythonParityCoreSegmentCall struct{ method, path, query, body string }

func pythonParityCoreSegmentAPI(t *testing.T, calls *[]pythonParityCoreSegmentCall, reply func(*http.Request) (int, string)) (*segments.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonParityCoreSegmentCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return segments.New(client), cloud
}

const (
	pythonParityCoreSegmentBase = "/neutron/v2.0/segments"
	pythonParityCoreSegmentRow  = `{"id":"seg-1","name":"rack1","description":"","network_id":"net-1","network_type":"vlan","physical_network":"physnet1","segmentation_id":101}`
)

// Python create_segment/update_segment send the given attributes inside the
// segment envelope with POST and PUT.
func TestPythonParityCoreSegmentCreateUpdate(t *testing.T) {
	ctx := context.Background()
	var calls []pythonParityCoreSegmentCall
	api, _ := pythonParityCoreSegmentAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.Method == http.MethodPost {
			return 201, `{"segment":` + pythonParityCoreSegmentRow + `}`
		}
		return 200, `{"segment":` + pythonParityCoreSegmentRow + `}`
	})
	created, err := api.Create(ctx, segments.CreateOpts{Name: "rack1", NetworkID: "net-1", NetworkType: "vlan", PhysicalNetwork: "physnet1", SegmentationID: 101})
	if err != nil || created.ID != "seg-1" || created.SegmentationID != 101 {
		t.Fatal(created, err)
	}
	name, empty, vlan := "rack2", "", 0
	if _, err := api.Update(ctx, "seg-1", segments.UpdateOpts{Name: &name, Description: &empty, SegmentationID: &vlan}, segments.WithUpdateField("x_extension", true)); err != nil {
		t.Fatal(err)
	}
	want := []pythonParityCoreSegmentCall{
		{http.MethodPost, pythonParityCoreSegmentBase, "", `{"segment":{"name":"rack1","network_id":"net-1","network_type":"vlan","physical_network":"physnet1","segmentation_id":101}}`},
		{http.MethodPut, pythonParityCoreSegmentBase + "/seg-1", "", `{"segment":{"description":"","name":"rack2","segmentation_id":0,"x_extension":true}}`},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
	// Python would POST without network_type and let Neutron decide; Go
	// rejects the missing required fields before HTTP.
	calls = nil
	if _, err := api.Create(ctx, segments.CreateOpts{NetworkID: "net-1"}); err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

// Python segments(**query) sends every QueryParameters key to the server and
// follows segments_links. The native SegmentPage only reads links.next, so Go
// stops after the first page and the caller continues with a marker query.
// get_segment without fields sends a bare GET.
func TestPythonParityCoreSegmentListAndGet(t *testing.T) {
	ctx := context.Background()
	var calls []pythonParityCoreSegmentCall
	var cloud *testcloud.Cloud
	api, cloud := pythonParityCoreSegmentAPI(t, &calls, func(req *http.Request) (int, string) {
		if strings.HasSuffix(req.URL.Path, "/seg-1") {
			return 200, `{"segment":` + pythonParityCoreSegmentRow + `}`
		}
		if req.URL.Query().Get("marker") == "" {
			return 200, `{"segments":[` + pythonParityCoreSegmentRow + `],"segments_links":[{"rel":"next","href":"` + cloud.Server.URL + pythonParityCoreSegmentBase + `?marker=seg-1"}]}`
		}
		return 200, `{"segments":[{"id":"seg-2","network_type":"flat","segmentation_id":null}]}`
	})
	collect := func(options ...segments.ListOption) []*segments.Segment {
		var rows []*segments.Segment
		for value, err := range api.List(ctx, options...) {
			if err != nil {
				t.Fatal(err)
			}
			rows = append(rows, value)
		}
		return rows
	}
	rows := collect(
		segments.WithListOptions(segments.ListOpts{NetworkID: "net-1", NetworkType: "vlan", SortKey: "name", SortDir: "asc"}),
		segments.WithListQuery("segmentation_id", "0"))
	if len(rows) != 1 || rows[0].PhysicalNetwork != "physnet1" || len(calls) != 1 {
		t.Fatalf("%+v %+v", rows, calls)
	}
	first, _ := url.ParseQuery(calls[0].query)
	if !reflect.DeepEqual(first, url.Values{"network_id": {"net-1"}, "network_type": {"vlan"}, "sort_key": {"name"}, "sort_dir": {"asc"}, "segmentation_id": {"0"}}) {
		t.Fatalf("%+v", calls)
	}
	rows = collect(segments.WithListQuery("marker", "seg-1"))
	// A null segmentation_id decodes to 0 where Python keeps None.
	if len(rows) != 1 || rows[0].ID != "seg-2" || rows[0].SegmentationID != 0 || calls[1].query != "marker=seg-1" {
		t.Fatalf("%+v %+v", rows, calls)
	}
	calls = nil
	got, err := api.Get(ctx, "seg-1")
	if err != nil || got.Name != "rack1" || !reflect.DeepEqual(calls, []pythonParityCoreSegmentCall{{http.MethodGet, pythonParityCoreSegmentBase + "/seg-1", "", ""}}) {
		t.Fatalf("%+v %v %v", calls, got, err)
	}
}

func TestPythonParityCoreSegmentDeleteIgnoreMissing(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		status  int
		options []resource.LookupOption
		missing bool
	}{
		{"deleted", 204, nil, false},
		{"missing ignored by default", 404, nil, false},
		{"ignore_missing=False", 404, []resource.LookupOption{resource.WithMissingError()}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonParityCoreSegmentCall
			api, _ := pythonParityCoreSegmentAPI(t, &calls, func(*http.Request) (int, string) { return tc.status, "" })
			err := api.Remove(ctx, resource.ID("seg-1"), tc.options...)
			if tc.missing != errors.Is(err, resource.ErrNotFound) || !tc.missing && err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, []pythonParityCoreSegmentCall{{http.MethodDelete, pythonParityCoreSegmentBase + "/seg-1", "", ""}}) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

// find_segment stays unresolved: the Go chain sends GET by ID and then a
// name-filtered list, but cannot attach Python's extra query to either step.
func TestPythonParityCoreSegmentFindChainWithoutQuery(t *testing.T) {
	ctx := context.Background()
	var calls []pythonParityCoreSegmentCall
	api, _ := pythonParityCoreSegmentAPI(t, &calls, func(req *http.Request) (int, string) {
		if strings.HasSuffix(req.URL.Path, "/rack1") {
			return 404, `{"NeutronError":{"type":"SegmentNotFound"}}`
		}
		return 200, `{"segments":[` + pythonParityCoreSegmentRow + `]}`
	})
	found, err := api.Find(ctx, resource.ID("rack1"), resource.WithIgnoreMissing())
	if err != nil || found != nil {
		t.Fatal(found, err)
	}
	found, err = api.Find(ctx, resource.Name("rack1"))
	if err != nil || found == nil || found.ID != "seg-1" {
		t.Fatal(found, err)
	}
	want := []pythonParityCoreSegmentCall{
		{http.MethodGet, pythonParityCoreSegmentBase + "/rack1", "", ""},
		{http.MethodGet, pythonParityCoreSegmentBase, "name=rack1", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}
