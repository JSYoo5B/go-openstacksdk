package volumes_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonVolumeCall records the wire request openstacksdk would also send.
type pythonVolumeCall struct{ method, path, query, body string }

type pythonVolumeTransport func(*http.Request) (*http.Response, error)

func (transport pythonVolumeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonVolumeWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonVolumeAPI(t *testing.T, reply func(*http.Request) *http.Response) (*volumes.API, *[]pythonVolumeCall) {
	t.Helper()
	calls := &[]pythonVolumeCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonVolumeTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonVolumeCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	return volumes.New(client), calls
}

const pythonVolumeBase = "/cinder/v3/project/volumes"

const pythonVolumeRow = `{"id":"vol-1","name":"data","description":"d","status":"available","size":1,"availability_zone":"nova","volume_type":"fast","bootable":"false","multiattach":true,"metadata":{"k":"v"},"os-vol-tenant-attr:tenant_id":"project","os-vol-host-attr:host":"host@lvm","created_at":"2026-10-11T01:02:03.000000"}`

func TestPythonVolumeGetCreateUpdateMatchProxyRequests(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonVolumeAPI(t, func(req *http.Request) *http.Response {
		if req.Method == http.MethodPost {
			return pythonVolumeWire(202, `{"volume":`+pythonVolumeRow+`}`)
		}
		return pythonVolumeWire(200, `{"volume":`+pythonVolumeRow+`}`)
	})
	// get_volume(volume)
	got, err := api.Get(ctx, "vol-1")
	if err != nil || got.ID != "vol-1" || got.Name != "data" || got.Size != 1 || !got.Multiattach || got.TenantID != "project" || got.Host != "host@lvm" || got.Metadata["k"] != "v" {
		t.Fatal(got, err)
	}
	// create_volume(name=..., size=1, image_id=..., volume_type=..., is_multiattach=True,
	// scheduler_hints={"same_host": [...]}) keeps the hints outside the volume envelope.
	created, err := api.Create(ctx, volumes.CreateOpts{Name: "data", Size: 1, ImageID: "img", VolumeType: "fast"},
		volumes.WithCreateField("multiattach", true),
		volumes.WithCreateHintOpts(volumes.SchedulerHintOpts{SameHost: []string{"0b3c1a3e-5f4d-4c8e-9a51-1f2e3d4c5b6a"}}))
	if err != nil || created.ID != "vol-1" {
		t.Fatal(created, err)
	}
	// update_volume(volume, name=..., description=...)
	name, description := "data2", "d2"
	if _, err := api.Update(ctx, "vol-1", volumes.UpdateOpts{Name: &name, Description: &description}); err != nil {
		t.Fatal(err)
	}
	want := []pythonVolumeCall{
		{http.MethodGet, pythonVolumeBase + "/vol-1", "", ""},
		{http.MethodPost, pythonVolumeBase, "", `{"OS-SCH-HNT:scheduler_hints":{"same_host":["0b3c1a3e-5f4d-4c8e-9a51-1f2e3d4c5b6a"]},"volume":{"imageRef":"img","multiattach":true,"name":"data","size":1,"volume_type":"fast"}}`},
		{http.MethodPut, pythonVolumeBase + "/vol-1", "", `{"volume":{"description":"d2","name":"data2"}}`},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonVolumeDeleteDefaultsCascadeForceAndMissing(t *testing.T) {
	ctx := context.Background()
	status := http.StatusNotFound
	api, calls := pythonVolumeAPI(t, func(req *http.Request) *http.Response {
		if req.Method == http.MethodPost {
			return pythonVolumeWire(202, "")
		}
		return pythonVolumeWire(status, `{"itemNotFound":{"message":"gone"}}`)
	})
	// delete_volume(volume) ignores a missing volume by default.
	if err := api.Remove(ctx, resource.ID("vol-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("vol-1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	// With microversion 3.23 Python always sends cascade and force; API.Delete reproduces the query
	// but keeps the 404, which the caller drops to match ignore_missing=True.
	err := api.Delete(ctx, "vol-1", volumes.WithDeleteQuery("cascade", "False"), volumes.WithDeleteQuery("force", "False"))
	if !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	status = http.StatusAccepted
	if err := api.Delete(ctx, "vol-1", volumes.WithDeleteQuery("cascade", "True"), volumes.WithDeleteQuery("force", "True")); err != nil {
		t.Fatal(err)
	}
	// delete_volume(volume, force=True) below microversion 3.23 posts os-force_delete; Python sends null, Gophercloud "".
	if err := api.ForceDelete(ctx, "vol-1"); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 5 {
		t.Fatalf("%+v", *calls)
	}
	queries := []url.Values{{}, {}, {"cascade": {"False"}, "force": {"False"}}, {"cascade": {"True"}, "force": {"True"}}}
	for i, want := range queries {
		got, _ := url.ParseQuery((*calls)[i].query)
		if (*calls)[i].method != http.MethodDelete || (*calls)[i].path != pythonVolumeBase+"/vol-1" || !reflect.DeepEqual(got, want) {
			t.Fatalf("%d %+v", i, (*calls)[i])
		}
	}
	if (*calls)[4] != (pythonVolumeCall{http.MethodPost, pythonVolumeBase + "/vol-1/action", "", `{"os-force_delete":""}`}) {
		t.Fatalf("%+v", (*calls)[4])
	}
}

func TestPythonVolumeListDefaultUsesDetailRouteOnly(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonVolumeAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("marker") == "vol-1" {
			return pythonVolumeWire(200, `{"volumes":[{"id":"vol-2"}]}`)
		}
		return pythonVolumeWire(200, `{"volumes":[`+pythonVolumeRow+`],"volumes_links":[{"rel":"next","href":"http://`+req.Host+req.URL.Path+`?marker=vol-1"}]}`)
	})
	// volumes(all_projects=True, name="data", status="available")
	var ids []string
	for value, err := range api.List(ctx, volumes.WithListOptions(volumes.ListOpts{Name: "data", Status: "available"}), volumes.WithListQuery("all_tenants", "True")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"vol-1", "vol-2"}) || len(*calls) != 2 {
		t.Fatalf("%v %+v", ids, *calls)
	}
	first, _ := url.ParseQuery((*calls)[0].query)
	// There is no public List variant for details=False (GET /volumes).
	if (*calls)[0].path != pythonVolumeBase+"/detail" || !reflect.DeepEqual(first, url.Values{"all_tenants": {"True"}, "name": {"data"}, "status": {"available"}}) || (*calls)[1].query != "marker=vol-1" {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonVolumeConnectionActionsOnlyTypedConnectorKeys(t *testing.T) {
	ctx := context.Background()
	terminating := false
	api, calls := pythonVolumeAPI(t, func(req *http.Request) *http.Response {
		if terminating {
			return pythonVolumeWire(202, "")
		}
		return pythonVolumeWire(200, `{"connection_info":{"driver_volume_type":"iscsi","data":{"target_lun":1}}}`)
	})
	multipath := false
	connector := volumes.InitializeConnectionOpts{Host: "compute-1", Initiator: "iqn.x", IP: "10.0.0.1", Multipath: &multipath, Platform: "x86_64", OSType: "linux2"}
	// init_volume_attachment(volume, connector) returns connection_info.
	info, err := api.InitializeConnection(ctx, "vol-1", connector)
	if err != nil || info["driver_volume_type"] != "iscsi" {
		t.Fatal(info, err)
	}
	// terminate_volume_attachment(volume, connector)
	terminating = true
	if err := api.TerminateConnection(ctx, "vol-1", volumes.TerminateConnectionOpts{Host: "compute-1", Initiator: "iqn.x", IP: "10.0.0.1", Multipath: &multipath, Platform: "x86_64", OSType: "linux2"}); err != nil {
		t.Fatal(err)
	}
	terminating = false
	// An extension field lands beside "connector", so connector keys such as nqn cannot be sent.
	if _, err := api.InitializeConnection(ctx, "vol-1", connector, volumes.WithInitializeConnectionField("nqn", "nqn.x")); err != nil {
		t.Fatal(err)
	}
	if _, err := api.InitializeConnection(ctx, "vol-1", connector, volumes.WithInitializeConnectionField("connector", map[string]any{"nqn": "nqn.x"})); err == nil {
		t.Fatal("connector replacement accepted")
	}
	typed := `{"host":"compute-1","initiator":"iqn.x","ip":"10.0.0.1","multipath":false,"os_type":"linux2","platform":"x86_64"}`
	action := pythonVolumeBase + "/vol-1/action"
	want := []pythonVolumeCall{
		{http.MethodPost, action, "", `{"os-initialize_connection":{"connector":` + typed + `}}`},
		{http.MethodPost, action, "", `{"os-terminate_connection":{"connector":` + typed + `}}`},
		{http.MethodPost, action, "", `{"os-initialize_connection":{"connector":` + typed + `,"nqn":"nqn.x"}}`},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}
