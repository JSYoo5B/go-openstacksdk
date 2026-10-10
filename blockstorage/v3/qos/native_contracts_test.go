package qos_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeQoSTransport func(*http.Request) (*http.Response, error)

func (transport nativeQoSTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeQoSWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeQoSCall struct{ method, path, query, body string }

func nativeQoSAPI(t *testing.T, calls *[]nativeQoSCall, reply func(*http.Request) *http.Response) (*qos.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeQoSTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeQoSCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	return qos.New(client), cloud
}

func nativeQoSOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "qos" {
		t.Fatal("generated qos context", err, wrapped)
	}
}

const nativeQoSRow = `{"id":"q-1","name":"gold","consumer":"back-end","specs":{"read_iops_sec":"1000"}}`

func TestNativeQoSRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeQoSCall
	var cloud *testcloud.Cloud
	api, cloud := nativeQoSAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeQoSWire(202, "")
		case req.Method == http.MethodPost:
			return nativeQoSWire(200, `{"qos_specs":`+nativeQoSRow+`}`)
		case req.Method == http.MethodPut && strings.HasSuffix(req.URL.Path, "/delete_keys"):
			return nativeQoSWire(202, "")
		case req.Method == http.MethodPut:
			// Update answers with the merged spec map only.
			return nativeQoSWire(200, `{"qos_specs":{"read_iops_sec":"2000","consumer":"front-end"}}`)
		case strings.HasSuffix(req.URL.Path, "/associate") || strings.HasSuffix(req.URL.Path, "/disassociate") || strings.HasSuffix(req.URL.Path, "/disassociate_all"):
			return nativeQoSWire(202, "")
		case strings.HasSuffix(req.URL.Path, "/associations"):
			return nativeQoSWire(200, `{"qos_associations":[{"id":"vt-1","name":"ssd","association_type":"volume_type"}]}`)
		case req.URL.Path == "/cinder/v3/project/qos-specs":
			return nativeQoSWire(200, `{"qos_specs":[`+nativeQoSRow+`],"qos_specs_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/qos?marker=x"}]}`)
		case req.URL.Path == "/other/qos":
			return nativeQoSWire(200, `{"qos_specs":[{"id":"q-2","specs":null}]}`)
		}
		return nativeQoSWire(200, `{"qos_specs":`+nativeQoSRow+`}`)
	})
	created, err := api.Create(ctx, qos.CreateOpts{Name: "gold", Consumer: qos.ConsumerBack, Specs: map[string]string{"read_iops_sec": "1000"}}, qos.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "q-1" && created.Consumer == "back-end" && created.Specs["read_iops_sec"] == "1000") {
		t.Fatal(created, err)
	}
	// Specs are merged after the typed fields, so a name spec overrides Name.
	if _, err := api.Create(ctx, qos.CreateOpts{Name: "gold", Specs: map[string]string{"name": "spec"}}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "q-1")
	if err != nil || got.Name != "gold" {
		t.Fatal(got, err)
	}
	specs, err := api.Update(ctx, "q-1", qos.UpdateOpts{Consumer: qos.ConsumerFront, Specs: map[string]string{"read_iops_sec": "2000"}})
	if err != nil || !reflect.DeepEqual(specs, map[string]string{"read_iops_sec": "2000", "consumer": "front-end"}) {
		t.Fatal(specs, err)
	}
	if err := api.DeleteKeys(ctx, "q-1", qos.WithDeleteKeysOptions(qos.DeleteKeysOpts{"read_iops_sec"})); err != nil {
		t.Fatal(err)
	}
	// Without keys the native body is {"keys": null}.
	if err := api.DeleteKeys(ctx, "q-1"); err != nil {
		t.Fatal(err)
	}
	if err := api.Associate(ctx, "q-1", qos.AssociateOpts{VolumeTypeID: "vt-1"}); err != nil {
		t.Fatal(err)
	}
	if err := api.Disassociate(ctx, "q-1", qos.DisassociateOpts{VolumeTypeID: "vt-1"}); err != nil {
		t.Fatal(err)
	}
	if err := api.DisassociateAll(ctx, "q-1"); err != nil {
		t.Fatal(err)
	}
	var associations []string
	for value, err := range api.ListAssociations(ctx, "q-1") {
		if err != nil {
			t.Fatal(err)
		}
		associations = append(associations, value.ID+"/"+value.AssociationType)
	}
	var ids []string
	for value, err := range api.List(ctx, qos.WithListOptions(qos.ListOpts{Limit: 1, Sort: "name"}), qos.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if err := api.Delete(ctx, "q-1", qos.WithDeleteOptions(qos.DeleteOpts{Force: true})); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(associations, []string{"vt-1/volume_type"}) || !reflect.DeepEqual(ids, []string{"q-1", "q-2"}) {
		t.Fatal(associations, ids)
	}
	base := "/cinder/v3/project/qos-specs"
	want := []nativeQoSCall{
		{http.MethodPost, base, "", `{"qos_specs":{"consumer":"back-end","name":"gold","read_iops_sec":"1000","x_extension":1}}`},
		{http.MethodPost, base, "", `{"qos_specs":{"name":"spec"}}`},
		{http.MethodGet, base + "/q-1", "", ""},
		{http.MethodPut, base + "/q-1", "", `{"qos_specs":{"consumer":"front-end","read_iops_sec":"2000"}}`},
		{http.MethodPut, base + "/q-1/delete_keys", "", `{"keys":["read_iops_sec"]}`},
		{http.MethodPut, base + "/q-1/delete_keys", "", `{"keys":null}`},
		// Association changes are GET requests with a vol_type_id query.
		{http.MethodGet, base + "/q-1/associate", "vol_type_id=vt-1", ""},
		{http.MethodGet, base + "/q-1/disassociate", "vol_type_id=vt-1", ""},
		{http.MethodGet, base + "/q-1/disassociate_all", "", ""},
		{http.MethodGet, base + "/q-1/associations", "", ""},
		{http.MethodGet, base, "extra=1&limit=1&sort=name", ""},
		{http.MethodGet, "/other/qos", "marker=x", ""},
		{http.MethodDelete, base + "/q-1", "force=true", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeQoSStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*qos.API) error
	}{
		// Create accepts only 200 and the association calls only 202.
		{"Create", []int{200}, func(api *qos.API) error { _, err := api.Create(ctx, qos.CreateOpts{Name: "gold"}); return err }},
		{"Get", []int{200}, func(api *qos.API) error { _, err := api.Get(ctx, "q-1"); return err }},
		{"Update", []int{200}, func(api *qos.API) error { _, err := api.Update(ctx, "q-1", qos.UpdateOpts{}); return err }},
		{"Delete", []int{202, 204}, func(api *qos.API) error { return api.Delete(ctx, "q-1") }},
		{"DeleteKeys", []int{202}, func(api *qos.API) error { return api.DeleteKeys(ctx, "q-1") }},
		{"Associate", []int{202}, func(api *qos.API) error { return api.Associate(ctx, "q-1", qos.AssociateOpts{VolumeTypeID: "vt"}) }},
		{"Disassociate", []int{202}, func(api *qos.API) error {
			return api.Disassociate(ctx, "q-1", qos.DisassociateOpts{VolumeTypeID: "vt"})
		}},
		{"DisassociateAll", []int{202}, func(api *qos.API) error { return api.DisassociateAll(ctx, "q-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeQoSCall
				api, _ := nativeQoSAPI(t, &calls, func(*http.Request) *http.Response { return nativeQoSWire(code, `{}`) })
				err := call.call(api)
				nativeQoSOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeQoSCall
		api, _ := nativeQoSAPI(t, &calls, func(*http.Request) *http.Response { return nativeQoSWire(200, `{}`) })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"associate type":    {"Associate", api.Associate(ctx, "q-1", qos.AssociateOpts{})},
			"disassociate type": {"Disassociate", api.Disassociate(ctx, "q-1", qos.DisassociateOpts{})},
			"core extension": {"Create", func() error {
				_, err := api.Create(ctx, qos.CreateOpts{Name: "gold"}, qos.WithCreateField("name", "x"))
				return err
			}()},
			"keys extension": {"DeleteKeys", api.DeleteKeys(ctx, "q-1", qos.WithDeleteKeysField("keys", nil))},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeQoSOperation(t, check.err, check.operation)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
