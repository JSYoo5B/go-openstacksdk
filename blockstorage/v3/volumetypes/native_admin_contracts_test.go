package volumetypes_test

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

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeTypeAdminTransport func(*http.Request) (*http.Response, error)

func (transport nativeTypeAdminTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeTypeAdminWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeTypeAdminCall struct{ method, path, query, body string }

func nativeTypeAdminAPI(t *testing.T, calls *[]nativeTypeAdminCall, reply func(*http.Request) *http.Response) (*volumetypes.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeTypeAdminTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeTypeAdminCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	return volumetypes.New(client), cloud
}

func nativeTypeAdminOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "volumetypes" {
		t.Fatal("generated volumetypes context", err, wrapped)
	}
}

const nativeTypeAdminRow = `{"id":"vt-1","name":"ssd","description":"d","is_public":false,"os-volume-type-access:is_public":false,"qos_specs_id":"q-1","extra_specs":{"a":"b"}}`

const nativeTypeAdminEncryption = `{"volume_type_id":"vt-1","encryption_id":"e-1","key_size":256,"provider":"luks","control_location":"front-end","cipher":"aes-xts-plain64"}`

func TestNativeVolumeTypeAdminRoutesBodiesAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeTypeAdminCall
	var cloud *testcloud.Cloud
	api, cloud := nativeTypeAdminAPI(t, &calls, func(req *http.Request) *http.Response {
		path := strings.TrimPrefix(req.URL.Path, "/cinder/v3/project/types")
		switch {
		case req.Method == http.MethodDelete || path == "/vt-1/action":
			return nativeTypeAdminWire(202, "")
		case path == "/vt-1/extra_specs":
			return nativeTypeAdminWire(200, `{"extra_specs":{"a":"b"}}`)
		case path == "/vt-1/extra_specs/a":
			// A single extra spec update answers with the bare key/value object.
			return nativeTypeAdminWire(200, `{"a":"c"}`)
		case path == "/vt-1/os-volume-type-access":
			// Access lists are one page; the link is never followed.
			return nativeTypeAdminWire(200, `{"volume_type_access":[{"volume_type_id":"vt-1","project_id":"p-1"}],"volume_type_access_links":[{"rel":"next","href":"`+cloud.Server.URL+`/never"}]}`)
		case path == "/vt-1/encryption" && req.Method == http.MethodGet:
			// The encryption show response has no envelope.
			return nativeTypeAdminWire(200, `{"volume_type_id":"vt-1","encryption_id":"e-1","deleted":false,"created_at":"2016-12-28T02:32:25.000000","updated_at":null,"deleted_at":null,"key_size":256,"provider":"luks","control_location":"front-end","cipher":"aes-xts-plain64"}`)
		case path == "/vt-1/encryption/cipher":
			return nativeTypeAdminWire(200, `{"cipher":"aes-xts-plain64","key_size":256}`)
		case strings.HasPrefix(path, "/vt-1/encryption"):
			return nativeTypeAdminWire(200, `{"encryption":`+nativeTypeAdminEncryption+`}`)
		}
		return nativeTypeAdminWire(200, `{"volume_type":`+nativeTypeAdminRow+`}`)
	})
	private, public := false, true
	name, description := "gold", ""
	created, err := api.Create(ctx, volumetypes.CreateOpts{Name: "ssd", Description: "d", IsPublic: &private, ExtraSpecs: map[string]string{"a": "b"}}, volumetypes.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "vt-1" && !created.IsPublic && !created.PublicAccess && created.QosSpecID == "q-1" && created.ExtraSpecs["a"] == "b") {
		t.Fatal(created, err)
	}
	if _, err := api.Create(ctx, volumetypes.CreateOpts{Name: "ssd"}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "vt-1", volumetypes.UpdateOpts{Name: &name, Description: &description, IsPublic: &public}, volumetypes.WithUpdateField("x_extension", "y")); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "vt-1", volumetypes.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "vt-1"); err != nil {
		t.Fatal(err)
	}
	specs, err := api.CreateExtraSpecs(ctx, "vt-1", volumetypes.ExtraSpecsOpts{"a": "b"}, volumetypes.WithCreateExtraSpecsField("x_extension", 1))
	if err != nil || !reflect.DeepEqual(specs, map[string]string{"a": "b"}) {
		t.Fatal(specs, err)
	}
	if _, err := api.CreateExtraSpecs(ctx, "vt-1", nil); err != nil {
		t.Fatal(err)
	}
	spec, err := api.UpdateExtraSpec(ctx, "vt-1", volumetypes.ExtraSpecsOpts{"a": "c"})
	if err != nil || !reflect.DeepEqual(spec, map[string]string{"a": "c"}) {
		t.Fatal(spec, err)
	}
	if err := api.DeleteExtraSpec(ctx, "vt-1", "a"); err != nil {
		t.Fatal(err)
	}
	if err := api.AddAccess(ctx, "vt-1", volumetypes.AddAccessOpts{Project: "p-1"}); err != nil {
		t.Fatal(err)
	}
	if err := api.AddAccess(ctx, "vt-1", volumetypes.AddAccessOpts{}); err != nil {
		t.Fatal(err)
	}
	if err := api.RemoveAccess(ctx, "vt-1", volumetypes.RemoveAccessOpts{Project: "p-1"}, volumetypes.WithRemoveAccessField("x_extension", true)); err != nil {
		t.Fatal(err)
	}
	var accesses []string
	for value, err := range api.ListAccesses(ctx, "vt-1") {
		if err != nil {
			t.Fatal(err)
		}
		accesses = append(accesses, value.VolumeTypeID+"/"+value.ProjectID)
	}
	encryption, err := api.CreateEncryption(ctx, "vt-1", volumetypes.CreateEncryptionOpts{Provider: "luks"}, volumetypes.WithCreateEncryptionField("x_extension", 1))
	if err != nil || !reflect.DeepEqual(*encryption, volumetypes.EncryptionType{VolumeTypeID: "vt-1", EncryptionID: "e-1", KeySize: 256, Provider: "luks", ControlLocation: "front-end", Cipher: "aes-xts-plain64"}) {
		t.Fatal(encryption, err)
	}
	shown, err := api.GetEncryption(ctx, "vt-1")
	// Timestamps stay raw strings and null becomes the empty string.
	if err != nil || !(shown.EncryptionID == "e-1" && shown.CreatedAt == "2016-12-28T02:32:25.000000" && shown.UpdatedAt == "" && shown.KeySize == 256 && !shown.Deleted) {
		t.Fatal(shown, err)
	}
	value, err := api.GetEncryptionSpec(ctx, "vt-1", "cipher")
	// The whole envelope-free body is returned, so numbers decode as float64.
	if err != nil || !reflect.DeepEqual(value, map[string]any{"cipher": "aes-xts-plain64", "key_size": float64(256)}) {
		t.Fatal(value, err)
	}
	if _, err := api.UpdateEncryption(ctx, "vt-1", "e-1", volumetypes.UpdateEncryptionOpts{KeySize: 512}); err != nil {
		t.Fatal(err)
	}
	if err := api.DeleteEncryption(ctx, "vt-1", "e-1"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(accesses, []string{"vt-1/p-1"}) {
		t.Fatal(accesses)
	}
	base := "/cinder/v3/project/types"
	want := []nativeTypeAdminCall{
		{http.MethodPost, base, "", `{"volume_type":{"description":"d","extra_specs":{"a":"b"},"name":"ssd","os-volume-type-access:is_public":false,"x_extension":1}}`},
		{http.MethodPost, base, "", `{"volume_type":{"name":"ssd"}}`},
		// Update sends is_public, not the access-prefixed create key, and keeps explicit empty strings.
		{http.MethodPut, base + "/vt-1", "", `{"volume_type":{"description":"","is_public":true,"name":"gold","x_extension":"y"}}`},
		{http.MethodPut, base + "/vt-1", "", `{"volume_type":{}}`},
		{http.MethodDelete, base + "/vt-1", "", ""},
		// The extra spec envelope is a string map, so extensions land beside it.
		{http.MethodPost, base + "/vt-1/extra_specs", "", `{"extra_specs":{"a":"b"},"x_extension":1}`},
		{http.MethodPost, base + "/vt-1/extra_specs", "", `{"extra_specs":null}`},
		{http.MethodPut, base + "/vt-1/extra_specs/a", "", `{"a":"c"}`},
		{http.MethodDelete, base + "/vt-1/extra_specs/a", "", ""},
		{http.MethodPost, base + "/vt-1/action", "", `{"addProjectAccess":{"project":"p-1"}}`},
		// The project ID is not required by the native builder.
		{http.MethodPost, base + "/vt-1/action", "", `{"addProjectAccess":{"project":""}}`},
		{http.MethodPost, base + "/vt-1/action", "", `{"removeProjectAccess":{"project":"p-1","x_extension":true}}`},
		{http.MethodGet, base + "/vt-1/os-volume-type-access", "", ""},
		// Encryption fields have no omitempty, so zero values are sent.
		{http.MethodPost, base + "/vt-1/encryption", "", `{"encryption":{"cipher":"","control_location":"","key_size":0,"provider":"luks","x_extension":1}}`},
		{http.MethodGet, base + "/vt-1/encryption", "", ""},
		{http.MethodGet, base + "/vt-1/encryption/cipher", "", ""},
		{http.MethodPut, base + "/vt-1/encryption/e-1", "", `{"encryption":{"cipher":"","control_location":"","key_size":512,"provider":""}}`},
		{http.MethodDelete, base + "/vt-1/encryption/e-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeVolumeTypeAdminStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*volumetypes.API) error
	}{
		{"Create", []int{200}, func(api *volumetypes.API) error {
			_, err := api.Create(ctx, volumetypes.CreateOpts{Name: "ssd"})
			return err
		}},
		{"Update", []int{200}, func(api *volumetypes.API) error {
			_, err := api.Update(ctx, "vt-1", volumetypes.UpdateOpts{})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *volumetypes.API) error { return api.Delete(ctx, "vt-1") }},
		{"CreateExtraSpecs", []int{200}, func(api *volumetypes.API) error {
			_, err := api.CreateExtraSpecs(ctx, "vt-1", volumetypes.ExtraSpecsOpts{"a": "b"})
			return err
		}},
		{"UpdateExtraSpec", []int{200}, func(api *volumetypes.API) error {
			_, err := api.UpdateExtraSpec(ctx, "vt-1", volumetypes.ExtraSpecsOpts{"a": "b"})
			return err
		}},
		// DeleteExtraSpec accepts only 202, unlike the default delete codes.
		{"DeleteExtraSpec", []int{202}, func(api *volumetypes.API) error { return api.DeleteExtraSpec(ctx, "vt-1", "a") }},
		{"AddAccess", []int{202}, func(api *volumetypes.API) error {
			return api.AddAccess(ctx, "vt-1", volumetypes.AddAccessOpts{Project: "p-1"})
		}},
		{"RemoveAccess", []int{202}, func(api *volumetypes.API) error {
			return api.RemoveAccess(ctx, "vt-1", volumetypes.RemoveAccessOpts{Project: "p-1"})
		}},
		{"CreateEncryption", []int{200}, func(api *volumetypes.API) error {
			_, err := api.CreateEncryption(ctx, "vt-1", volumetypes.CreateEncryptionOpts{Provider: "luks"})
			return err
		}},
		{"GetEncryption", []int{200}, func(api *volumetypes.API) error { _, err := api.GetEncryption(ctx, "vt-1"); return err }},
		{"GetEncryptionSpec", []int{200}, func(api *volumetypes.API) error {
			_, err := api.GetEncryptionSpec(ctx, "vt-1", "cipher")
			return err
		}},
		{"UpdateEncryption", []int{200}, func(api *volumetypes.API) error {
			_, err := api.UpdateEncryption(ctx, "vt-1", "e-1", volumetypes.UpdateEncryptionOpts{})
			return err
		}},
		{"DeleteEncryption", []int{202, 204}, func(api *volumetypes.API) error { return api.DeleteEncryption(ctx, "vt-1", "e-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeTypeAdminCall
				api, _ := nativeTypeAdminAPI(t, &calls, func(*http.Request) *http.Response { return nativeTypeAdminWire(code, `{}`) })
				err := call.call(api)
				nativeTypeAdminOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("decode", func(t *testing.T) {
		var calls []nativeTypeAdminCall
		api, _ := nativeTypeAdminAPI(t, &calls, func(req *http.Request) *http.Response {
			if strings.HasSuffix(req.URL.Path, "/extra_specs/a") {
				// Non-string spec values fail the string map decode.
				return nativeTypeAdminWire(200, `{"a":1}`)
			}
			return nativeTypeAdminWire(200, `{}`)
		})
		if got, err := api.Create(ctx, volumetypes.CreateOpts{Name: "ssd"}); err != nil || got == nil || got.ID != "" {
			t.Fatal(got, err)
		}
		if got, err := api.Update(ctx, "vt-1", volumetypes.UpdateOpts{}); err != nil || got == nil || got.ID != "" {
			t.Fatal(got, err)
		}
		if specs, err := api.CreateExtraSpecs(ctx, "vt-1", volumetypes.ExtraSpecsOpts{"a": "b"}); err != nil || specs != nil {
			t.Fatal(specs, err)
		}
		if got, err := api.CreateEncryption(ctx, "vt-1", volumetypes.CreateEncryptionOpts{Provider: "luks"}); err != nil || got == nil || got.EncryptionID != "" {
			t.Fatal(got, err)
		}
		// A type without encryption answers {} and decodes to a zero value.
		if got, err := api.GetEncryption(ctx, "vt-1"); err != nil || got == nil || got.EncryptionID != "" {
			t.Fatal(got, err)
		}
		if got, err := api.GetEncryptionSpec(ctx, "vt-1", "cipher"); err != nil || len(got) != 0 {
			t.Fatal(got, err)
		}
		_, err := api.UpdateExtraSpec(ctx, "vt-1", volumetypes.ExtraSpecsOpts{"a": "c"})
		nativeTypeAdminOperation(t, err, "UpdateExtraSpec")
	})
	t.Run("access list status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"volume_type_access":[]}`}, {204, ""}} {
			var calls []nativeTypeAdminCall
			api, _ := nativeTypeAdminAPI(t, &calls, func(*http.Request) *http.Response { return nativeTypeAdminWire(tc.code, tc.body) })
			var errs []error
			for _, err := range api.ListAccesses(ctx, "vt-1") {
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			switch {
			case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
			case tc.code == 200 && len(errs) == 0:
			case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
			default:
				t.Fatal(tc.code, errs)
			}
			if len(calls) != 1 {
				t.Fatal(calls)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeTypeAdminCall
		api, _ := nativeTypeAdminAPI(t, &calls, func(*http.Request) *http.Response { return nativeTypeAdminWire(200, `{}`) })
		value := func(_ any, err error) error { return err }
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create name":       {"Create", value(api.Create(ctx, volumetypes.CreateOpts{}))},
			"create core field": {"Create", value(api.Create(ctx, volumetypes.CreateOpts{Name: "ssd"}, volumetypes.WithCreateField("extra_specs", map[string]string{})))},
			"create nil option": {"Create", value(api.Create(ctx, volumetypes.CreateOpts{Name: "ssd"}, nil))},
			"update core field": {"Update", value(api.Update(ctx, "vt-1", volumetypes.UpdateOpts{}, volumetypes.WithUpdateField("is_public", true)))},
			"extra specs envelope": {"CreateExtraSpecs", value(api.CreateExtraSpecs(ctx, "vt-1", volumetypes.ExtraSpecsOpts{"a": "b"},
				volumetypes.WithCreateExtraSpecsField("extra_specs", map[string]string{"c": "d"})))},
			"update spec empty": {"UpdateExtraSpec", value(api.UpdateExtraSpec(ctx, "vt-1", volumetypes.ExtraSpecsOpts{}))},
			"update spec pair":  {"UpdateExtraSpec", value(api.UpdateExtraSpec(ctx, "vt-1", volumetypes.ExtraSpecsOpts{"a": "b", "c": "d"}))},
			// The update option shares the create option type, but body extensions are rejected.
			"update spec field":   {"UpdateExtraSpec", value(api.UpdateExtraSpec(ctx, "vt-1", volumetypes.ExtraSpecsOpts{"a": "b"}, volumetypes.WithCreateExtraSpecsField("x", 1)))},
			"add access core":     {"AddAccess", api.AddAccess(ctx, "vt-1", volumetypes.AddAccessOpts{Project: "p-1"}, volumetypes.WithAddAccessField("project", "p-2"))},
			"remove access nil":   {"RemoveAccess", api.RemoveAccess(ctx, "vt-1", volumetypes.RemoveAccessOpts{Project: "p-1"}, nil)},
			"encryption provider": {"CreateEncryption", value(api.CreateEncryption(ctx, "vt-1", volumetypes.CreateEncryptionOpts{KeySize: 256}))},
			"encryption core field": {"CreateEncryption", value(api.CreateEncryption(ctx, "vt-1", volumetypes.CreateEncryptionOpts{Provider: "luks"},
				volumetypes.WithCreateEncryptionField("cipher", "x")))},
			"update encryption core": {"UpdateEncryption", value(api.UpdateEncryption(ctx, "vt-1", "e-1", volumetypes.UpdateEncryptionOpts{},
				volumetypes.WithUpdateEncryptionField("provider", "x")))},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeTypeAdminOperation(t, check.err, check.operation)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
