package manageablevolumes_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablevolumes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeManageTransport func(*http.Request) (*http.Response, error)

func (transport nativeManageTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeManageWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeManageCall struct{ method, path, query, body string }

func nativeManageAPI(t *testing.T, calls *[]nativeManageCall, reply func(*http.Request) *http.Response) *manageablevolumes.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeManageTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeManageCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	return manageablevolumes.New(client)
}

func nativeManageOperation(t *testing.T, err error) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != "ManageExisting" || wrapped.Resource != "manageablevolumes" {
		t.Fatal("generated manageablevolumes context", err, wrapped)
	}
}

func TestNativeManageableVolumeRouteBodyAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeManageCall
	api := nativeManageAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeManageWire(202, `{"volume":{"id":"v-1","status":"creating","size":0,"name":"imported","bootable":"true","created_at":"2017-06-29T05:50:35.000000","updated_at":null,"metadata":{"k":"v"}}}`)
	})
	volume, err := api.ManageExisting(ctx, manageablevolumes.ManageExistingOpts{
		Host: "host@lvm#LVM", Ref: map[string]string{"source-name": "existing"}, Name: "imported", AvailabilityZone: "nova",
		Description: "d", VolumeType: "lvm", Bootable: true, Metadata: map[string]string{"k": "v"},
	}, manageablevolumes.WithManageExistingField("x_extension", 1))
	// Volume timestamps use the Cinder millisecond format without a zone.
	if err != nil || !(volume.ID == "v-1" && volume.Status == "creating" && volume.Bootable == "true" && volume.CreatedAt.Equal(time.Date(2017, 6, 29, 5, 50, 35, 0, time.UTC)) && volume.UpdatedAt.IsZero() && volume.Metadata["k"] == "v") {
		t.Fatal(volume, err)
	}
	// Host and cluster are both optional in the native builder, so an empty request is sent.
	if _, err := api.ManageExisting(ctx, manageablevolumes.ManageExistingOpts{Cluster: "c@lvm", Ref: map[string]string{"source-id": "x"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.ManageExisting(ctx, manageablevolumes.ManageExistingOpts{}); err != nil {
		t.Fatal(err)
	}
	path := "/cinder/v3/project/manageable_volumes"
	want := []nativeManageCall{
		{http.MethodPost, path, "", `{"volume":{"availability_zone":"nova","bootable":true,"description":"d","host":"host@lvm#LVM","metadata":{"k":"v"},"name":"imported","ref":{"source-name":"existing"},"volume_type":"lvm","x_extension":1}}`},
		{http.MethodPost, path, "", `{"volume":{"cluster":"c@lvm","ref":{"source-id":"x"}}}`},
		{http.MethodPost, path, "", `{"volume":{}}`},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeManageableVolumeStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, code := range []int{200, 201, 204, 404} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []nativeManageCall
			api := nativeManageAPI(t, &calls, func(*http.Request) *http.Response { return nativeManageWire(code, `{}`) })
			_, err := api.ManageExisting(ctx, manageablevolumes.ManageExistingOpts{Host: "h"})
			nativeManageOperation(t, err)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{202}) || len(calls) != 1 {
				t.Fatal(err, native)
			}
		})
	}
	t.Run("decode", func(t *testing.T) {
		body := `{}`
		var calls []nativeManageCall
		api := nativeManageAPI(t, &calls, func(*http.Request) *http.Response { return nativeManageWire(202, body) })
		if volume, err := api.ManageExisting(ctx, manageablevolumes.ManageExistingOpts{Host: "h"}); err != nil || volume == nil || volume.ID != "" {
			t.Fatal(volume, err)
		}
		// A zone suffix does not match the native timestamp layout.
		body = `{"volume":{"id":"v-1","created_at":"2017-06-29T05:50:35Z"}}`
		_, err := api.ManageExisting(ctx, manageablevolumes.ManageExistingOpts{Host: "h"})
		nativeManageOperation(t, err)
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeManageCall
		api := nativeManageAPI(t, &calls, func(*http.Request) *http.Response { return nativeManageWire(202, `{}`) })
		for name, option := range map[string]manageablevolumes.ManageExistingOption{
			"omitted core field": manageablevolumes.WithManageExistingField("cluster", "c"),
			"set core field":     manageablevolumes.WithManageExistingField("host", "other"),
			"empty key":          manageablevolumes.WithManageExistingField(" ", 1),
			"nil option":         nil,
		} {
			_, err := api.ManageExisting(ctx, manageablevolumes.ManageExistingOpts{Host: "h"}, option)
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeManageOperation(t, err)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
