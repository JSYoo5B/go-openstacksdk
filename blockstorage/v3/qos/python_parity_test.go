package qos_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonQoSCall records the wire request openstacksdk would also send.
type pythonQoSCall struct{ method, path, query, body string }

type pythonQoSTransport func(*http.Request) (*http.Response, error)

func (transport pythonQoSTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonQoSWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonQoSAPI(t *testing.T, reply func(*http.Request) *http.Response) (*qos.API, *[]pythonQoSCall) {
	t.Helper()
	calls := &[]pythonQoSCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonQoSTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonQoSCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	return qos.New(client), calls
}

const pythonQoSBase = "/cinder/v3/project/qos-specs"

const pythonQoSRow = `{"id":"qos-1","name":"gold","consumer":"back-end","specs":{"read_iops_sec":"100"}}`

func TestPythonQoSCrudMatchesProxyRequests(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonQoSAPI(t, func(req *http.Request) *http.Response {
		switch req.Method {
		case http.MethodDelete:
			return pythonQoSWire(202, "")
		case http.MethodPut:
			return pythonQoSWire(200, `{"qos_specs":{"read_iops_sec":"200"}}`)
		}
		return pythonQoSWire(200, `{"qos_specs":`+pythonQoSRow+`}`)
	})
	// create_qos_spec(name="gold", consumer="back-end", read_iops_sec="100") sends specs flat.
	created, err := api.Create(ctx, qos.CreateOpts{Name: "gold", Consumer: qos.ConsumerBack, Specs: map[string]string{"read_iops_sec": "100"}})
	if err != nil || created.ID != "qos-1" || created.Specs["read_iops_sec"] != "100" {
		t.Fatal(created, err)
	}
	// get_qos_spec(qos_spec)
	got, err := api.Get(ctx, "qos-1")
	if err != nil || got.Name != "gold" || got.Consumer != "back-end" {
		t.Fatal(got, err)
	}
	// update_qos_spec(qos_spec, read_iops_sec="200") returns the updated specs.
	updated, err := api.Update(ctx, "qos-1", qos.UpdateOpts{Specs: map[string]string{"read_iops_sec": "200"}})
	if err != nil || !reflect.DeepEqual(updated, map[string]string{"read_iops_sec": "200"}) {
		t.Fatal(updated, err)
	}
	// delete_qos_spec(qos_spec) always sends force=False; force=True sends force=True.
	if err := api.Delete(ctx, "qos-1", qos.WithDeleteQuery("force", "False")); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "qos-1", qos.WithDeleteQuery("force", "True")); err != nil {
		t.Fatal(err)
	}
	want := []pythonQoSCall{
		{http.MethodPost, pythonQoSBase, "", `{"qos_specs":{"consumer":"back-end","name":"gold","read_iops_sec":"100"}}`},
		{http.MethodGet, pythonQoSBase + "/qos-1", "", ""},
		{http.MethodPut, pythonQoSBase + "/qos-1", "", `{"qos_specs":{"read_iops_sec":"200"}}`},
		{http.MethodDelete, pythonQoSBase + "/qos-1", "force=False", ""},
		{http.MethodDelete, pythonQoSBase + "/qos-1", "force=True", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonQoSDeleteMissingPolicies(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonQoSAPI(t, func(*http.Request) *http.Response {
		return pythonQoSWire(404, `{"itemNotFound":{"message":"gone"}}`)
	})
	// delete_qos_spec(qos_spec) ignores 404; Remove matches but cannot add force.
	if err := api.Remove(ctx, resource.ID("qos-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("qos-1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "qos-1", qos.WithDeleteQuery("force", "True")); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	if len(*calls) != 3 || (*calls)[0].query != "" || (*calls)[2].query != "force=True" {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonQoSAssociationsKeysAndList(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonQoSAPI(t, func(req *http.Request) *http.Response {
		switch {
		case strings.HasSuffix(req.URL.Path, "/associations"):
			return pythonQoSWire(200, `{"qos_associations":[{"association_type":"volume_type","id":"vt-1","name":"fast"}]}`)
		case req.URL.Path == pythonQoSBase && req.URL.Query().Get("marker") == "":
			return pythonQoSWire(200, `{"qos_specs":[`+pythonQoSRow+`],"qos_specs_links":[{"rel":"next","href":"http://`+req.Host+pythonQoSBase+`?marker=qos-1"}]}`)
		case req.URL.Path == pythonQoSBase:
			return pythonQoSWire(200, `{"qos_specs":[{"id":"qos-2","name":"silver"}]}`)
		}
		return pythonQoSWire(202, "")
	})
	// associate_qos_spec, disassociate_qos_spec and disassociate_all_qos_spec are GET requests.
	if err := api.Associate(ctx, "qos-1", qos.AssociateOpts{VolumeTypeID: "vt-1"}); err != nil {
		t.Fatal(err)
	}
	if err := api.Disassociate(ctx, "qos-1", qos.DisassociateOpts{VolumeTypeID: "vt-1"}); err != nil {
		t.Fatal(err)
	}
	if err := api.DisassociateAll(ctx, "qos-1"); err != nil {
		t.Fatal(err)
	}
	// delete_qos_spec_metadata(qos_spec, ["a", "b"])
	if err := api.DeleteKeys(ctx, "qos-1", qos.WithDeleteKeysOptions(qos.DeleteKeysOpts{"a", "b"})); err != nil {
		t.Fatal(err)
	}
	// qos_spec_associations(qos_spec)
	var associations []string
	for value, err := range api.ListAssociations(ctx, "qos-1") {
		if err != nil {
			t.Fatal(err)
		}
		associations = append(associations, value.AssociationType+"/"+value.ID+"/"+value.Name)
	}
	// qos_specs(limit=1) follows qos_specs_links.
	var ids []string
	for value, err := range api.List(ctx, qos.WithListOptions(qos.ListOpts{Limit: 1})) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	want := []pythonQoSCall{
		{http.MethodGet, pythonQoSBase + "/qos-1/associate", "vol_type_id=vt-1", ""},
		{http.MethodGet, pythonQoSBase + "/qos-1/disassociate", "vol_type_id=vt-1", ""},
		{http.MethodGet, pythonQoSBase + "/qos-1/disassociate_all", "", ""},
		{http.MethodPut, pythonQoSBase + "/qos-1/delete_keys", "", `{"keys":["a","b"]}`},
		{http.MethodGet, pythonQoSBase + "/qos-1/associations", "", ""},
		{http.MethodGet, pythonQoSBase, "limit=1", ""},
		{http.MethodGet, pythonQoSBase, "marker=qos-1", ""},
	}
	if !reflect.DeepEqual(associations, []string{"volume_type/vt-1/fast"}) || !reflect.DeepEqual(ids, []string{"qos-1", "qos-2"}) || !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%v %v %+v", associations, ids, *calls)
	}
}

func TestPythonQoSFindHasNoIDThenNameFallback(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonQoSAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Path == pythonQoSBase {
			return pythonQoSWire(200, `{"qos_specs":[`+pythonQoSRow+`]}`)
		}
		return pythonQoSWire(404, `{"itemNotFound":{"message":"gone"}}`)
	})
	// find_qos_spec("gold", **query) would GET by ID with query and then list all specs with the same query.
	byID, err := api.Find(ctx, resource.ID("gold"), resource.WithIgnoreMissing())
	if err != nil || byID != nil {
		t.Fatal(byID, err)
	}
	byName, err := api.Find(ctx, resource.Name("gold"))
	if err != nil || byName.ID != "qos-1" {
		t.Fatal(byName, err)
	}
	query, _ := url.ParseQuery((*calls)[1].query)
	if len(*calls) != 2 || (*calls)[0].path != pythonQoSBase+"/gold" || (*calls)[1].path != pythonQoSBase || len(query) != 0 {
		t.Fatalf("%+v", *calls)
	}
}
