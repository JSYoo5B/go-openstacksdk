package volumetypes_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonTypeCall records the wire request openstacksdk would also send.
type pythonTypeCall struct{ method, path, query, body string }

type pythonTypeTransport func(*http.Request) (*http.Response, error)

func (transport pythonTypeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonTypeWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonTypeAPI(t *testing.T, reply func(*http.Request) *http.Response) (*volumetypes.API, *[]pythonTypeCall) {
	t.Helper()
	calls := &[]pythonTypeCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonTypeTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonTypeCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	return volumetypes.New(client), calls
}

const pythonTypeBase = "/cinder/v3/project/types"

const pythonTypeRow = `{"id":"vt-1","name":"fast","description":"d","extra_specs":{"volume_backend_name":"lvm"},"os-volume-type-access:is_public":false,"is_public":false,"qos_specs_id":"qos-1"}`

func TestPythonTypeCrudAndExtraSpecsMatchProxyRequests(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonTypeAPI(t, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return pythonTypeWire(202, "")
		case strings.HasSuffix(req.URL.Path, "/extra_specs"):
			return pythonTypeWire(200, `{"extra_specs":{"a":"b"}}`)
		}
		return pythonTypeWire(200, `{"volume_type":`+pythonTypeRow+`}`)
	})
	// get_type(type)
	got, err := api.Get(ctx, "vt-1")
	if err != nil || got.ID != "vt-1" || got.Name != "fast" || got.ExtraSpecs["volume_backend_name"] != "lvm" || got.PublicAccess || got.QosSpecID != "qos-1" {
		t.Fatal(got, err)
	}
	// create_type(name=..., description=..., extra_specs=..., is_public=False)
	public := false
	if _, err := api.Create(ctx, volumetypes.CreateOpts{Name: "fast", Description: "d", IsPublic: &public, ExtraSpecs: map[string]string{"volume_backend_name": "lvm"}}); err != nil {
		t.Fatal(err)
	}
	// update_type(type, name=..., description=...)
	name, description := "faster", "d2"
	if _, err := api.Update(ctx, "vt-1", volumetypes.UpdateOpts{Name: &name, Description: &description}); err != nil {
		t.Fatal(err)
	}
	// update_type_extra_specs(type, a="b") returns the server's extra_specs.
	specs, err := api.CreateExtraSpecs(ctx, "vt-1", volumetypes.ExtraSpecsOpts{"a": "b"})
	if err != nil || !reflect.DeepEqual(specs, map[string]string{"a": "b"}) {
		t.Fatal(specs, err)
	}
	// delete_type_extra_specs(type, ["a", "c"]) sends one DELETE per key.
	for _, key := range []string{"a", "c"} {
		if err := api.DeleteExtraSpec(ctx, "vt-1", key); err != nil {
			t.Fatal(err)
		}
	}
	// delete_type(type)
	if err := api.Remove(ctx, resource.ID("vt-1")); err != nil {
		t.Fatal(err)
	}
	want := []pythonTypeCall{
		{http.MethodGet, pythonTypeBase + "/vt-1", "", ""},
		{http.MethodPost, pythonTypeBase, "", `{"volume_type":{"description":"d","extra_specs":{"volume_backend_name":"lvm"},"name":"fast","os-volume-type-access:is_public":false}}`},
		{http.MethodPut, pythonTypeBase + "/vt-1", "", `{"volume_type":{"description":"d2","name":"faster"}}`},
		{http.MethodPost, pythonTypeBase + "/vt-1/extra_specs", "", `{"extra_specs":{"a":"b"}}`},
		{http.MethodDelete, pythonTypeBase + "/vt-1/extra_specs/a", "", ""},
		{http.MethodDelete, pythonTypeBase + "/vt-1/extra_specs/c", "", ""},
		{http.MethodDelete, pythonTypeBase + "/vt-1", "", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonTypeDeleteIgnoresMissing(t *testing.T) {
	ctx := context.Background()
	api, _ := pythonTypeAPI(t, func(*http.Request) *http.Response {
		return pythonTypeWire(404, `{"itemNotFound":{"message":"gone"}}`)
	})
	if err := api.Remove(ctx, resource.ID("vt-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("vt-1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestPythonTypeListVisibilityAndQuery(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		options []volumetypes.ListOption
		query   url.Values
	}{
		// types() sends no is_public, which Cinder reads as true; Go needs VisibilityPublic for that.
		{"default", []volumetypes.ListOption{volumetypes.WithListOptions(volumetypes.ListOpts{IsPublic: volumetypes.VisibilityPublic})}, url.Values{"is_public": {"true"}}},
		// Without options Go asks for public and private types.
		{"go default", nil, url.Values{"is_public": {"None"}}},
		// types(is_public="none", all_projects=True, sort="name:asc")
		{"query", []volumetypes.ListOption{volumetypes.WithListQuery("is_public", "none"), volumetypes.WithListQuery("all_tenants", "True"), volumetypes.WithListOptions(volumetypes.ListOpts{Sort: "name:asc"})}, url.Values{"is_public": {"none"}, "all_tenants": {"True"}, "sort": {"name:asc"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api, calls := pythonTypeAPI(t, func(*http.Request) *http.Response {
				return pythonTypeWire(200, `{"volume_types":[`+pythonTypeRow+`]}`)
			})
			var ids []string
			for value, err := range api.List(ctx, tc.options...) {
				if err != nil {
					t.Fatal(err)
				}
				ids = append(ids, value.ID)
			}
			query, _ := url.ParseQuery((*calls)[0].query)
			if !reflect.DeepEqual(ids, []string{"vt-1"}) || len(*calls) != 1 || (*calls)[0].path != pythonTypeBase || !reflect.DeepEqual(query, tc.query) {
				t.Fatalf("%v %+v", ids, *calls)
			}
		})
	}
}

func TestPythonTypeAccessActions(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonTypeAPI(t, func(req *http.Request) *http.Response {
		if req.Method == http.MethodPost {
			return pythonTypeWire(202, "")
		}
		return pythonTypeWire(200, `{"volume_type_access":[{"volume_type_id":"vt-1","project_id":"p1"}]}`)
	})
	// get_type_access(type)
	var projects []string
	for value, err := range api.ListAccesses(ctx, "vt-1") {
		if err != nil {
			t.Fatal(err)
		}
		projects = append(projects, value.VolumeTypeID+"/"+value.ProjectID)
	}
	// add_type_access(type, project) and remove_type_access(type, project)
	if err := api.AddAccess(ctx, "vt-1", volumetypes.AddAccessOpts{Project: "p2"}); err != nil {
		t.Fatal(err)
	}
	if err := api.RemoveAccess(ctx, "vt-1", volumetypes.RemoveAccessOpts{Project: "p2"}); err != nil {
		t.Fatal(err)
	}
	want := []pythonTypeCall{
		{http.MethodGet, pythonTypeBase + "/vt-1/os-volume-type-access", "", ""},
		{http.MethodPost, pythonTypeBase + "/vt-1/action", "", `{"addProjectAccess":{"project":"p2"}}`},
		{http.MethodPost, pythonTypeBase + "/vt-1/action", "", `{"removeProjectAccess":{"project":"p2"}}`},
	}
	if !reflect.DeepEqual(projects, []string{"vt-1/p1"}) || !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%v %+v", projects, *calls)
	}
}

const pythonTypeEncryption = `{"volume_type_id":"vt-1","encryption_id":"enc-1","provider":"luks","cipher":"aes-xts-plain64","key_size":256,"control_location":"front-end","deleted":false,"created_at":"2026-10-11T01:02:03.000000"}`

func TestPythonTypeEncryptionGetDeleteAndFullBodies(t *testing.T) {
	ctx := context.Background()
	status := http.StatusAccepted
	api, calls := pythonTypeAPI(t, func(req *http.Request) *http.Response {
		switch req.Method {
		case http.MethodDelete:
			return pythonTypeWire(status, `{"itemNotFound":{"message":"gone"}}`)
		case http.MethodGet:
			return pythonTypeWire(200, pythonTypeEncryption)
		}
		return pythonTypeWire(200, `{"encryption":`+pythonTypeEncryption+`}`)
	})
	// get_type_encryption(volume_type) reads the flat show body.
	enc, err := api.GetEncryption(ctx, "vt-1")
	if err != nil || enc.EncryptionID != "enc-1" || enc.Provider != "luks" || enc.KeySize != 256 || enc.ControlLocation != "front-end" {
		t.Fatal(enc, err)
	}
	// delete_type_encryption(volume_type=type) fetches the encryption and deletes it by encryption_id.
	if err := api.DeleteEncryption(ctx, "vt-1", enc.EncryptionID); err != nil {
		t.Fatal(err)
	}
	status = http.StatusNotFound
	// ignore_missing=True needs the caller to drop the 404 from DeleteEncryption.
	if err := api.DeleteEncryption(ctx, "vt-1", "enc-1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	// create_type_encryption(type, provider="luks", control_location="front-end") omits cipher and key_size in Python.
	if _, err := api.CreateEncryption(ctx, "vt-1", volumetypes.CreateEncryptionOpts{Provider: "luks", ControlLocation: "front-end"}); err != nil {
		t.Fatal(err)
	}
	// update_type_encryption(volume_type=type, key_size=512) sends only key_size in Python.
	if _, err := api.UpdateEncryption(ctx, "vt-1", "enc-1", volumetypes.UpdateEncryptionOpts{KeySize: 512}); err != nil {
		t.Fatal(err)
	}
	want := []pythonTypeCall{
		{http.MethodGet, pythonTypeBase + "/vt-1/encryption", "", ""},
		{http.MethodDelete, pythonTypeBase + "/vt-1/encryption/enc-1", "", ""},
		{http.MethodDelete, pythonTypeBase + "/vt-1/encryption/enc-1", "", ""},
		{http.MethodPost, pythonTypeBase + "/vt-1/encryption", "", `{"encryption":{"cipher":"","control_location":"front-end","key_size":0,"provider":"luks"}}`},
		{http.MethodPut, pythonTypeBase + "/vt-1/encryption/enc-1", "", `{"encryption":{"cipher":"","control_location":"","key_size":512,"provider":""}}`},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonTypeFindHasNoIsPublicNoneFallback(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonTypeAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Path == pythonTypeBase {
			return pythonTypeWire(200, `{"volume_types":[`+pythonTypeRow+`]}`)
		}
		return pythonTypeWire(404, `{"itemNotFound":{"message":"gone"}}`)
	})
	// find_type("fast") sends GET /types/fast?is_public=none, then GET /types?is_public=none and matches locally.
	byID, err := api.Find(ctx, resource.ID("fast"), resource.WithIgnoreMissing())
	if err != nil || byID != nil {
		t.Fatal(byID, err)
	}
	byName, err := api.Find(ctx, resource.Name("fast"))
	if err != nil || byName.ID != "vt-1" {
		t.Fatal(byName, err)
	}
	query, _ := url.ParseQuery((*calls)[1].query)
	if (*calls)[0].query != "" || (*calls)[1].path != pythonTypeBase || !reflect.DeepEqual(query, url.Values{"is_public": {"None"}, "name": {"fast"}}) {
		t.Fatalf("%+v", *calls)
	}
}
