package servers_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

func TestNativeServerMetadataRoutesBodiesAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeServerCall
	nativeServerRecorder(t, cloud, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeServerWire(204, "")
		case req.URL.Path == "/nova/v2.1/servers/s1/metadata":
			return nativeServerWire(200, `{"metadata":{"a":"1","b":"2"}}`)
		}
		return nativeServerWire(200, `{"meta":{"a":"1"}}`)
	})
	api := servers.New(nativeServerClient(cloud))
	ctx := context.Background()
	all, err := api.Metadata(ctx, "s1")
	if err != nil || !reflect.DeepEqual(all, map[string]string{"a": "1", "b": "2"}) {
		t.Fatal(all, err)
	}
	// Reset replaces every item with PUT; Update merges with POST.
	reset, err := api.ResetMetadata(ctx, "s1", servers.MetadataOpts{"a": "1"})
	if err != nil || reset["b"] != "2" {
		t.Fatal("native reset returns the server response", reset, err)
	}
	if _, err := api.UpdateMetadata(ctx, "s1", servers.MetadataOpts{"c": "3"}, servers.WithUpdateMetadataField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	one, err := api.Metadatum(ctx, "s1", "a")
	if err != nil || !reflect.DeepEqual(one, map[string]string{"a": "1"}) {
		t.Fatal(one, err)
	}
	created, err := api.CreateMetadatum(ctx, "s1", servers.MetadatumOpts{"a": "9"})
	if err != nil || created["a"] != "1" {
		t.Fatal(created, err)
	}
	if err := api.DeleteMetadatum(ctx, "s1", "a/b"); err != nil {
		t.Fatal(err)
	}
	want := []nativeServerCall{
		{http.MethodGet, "/nova/v2.1/servers/s1/metadata", "", ""},
		{http.MethodPut, "/nova/v2.1/servers/s1/metadata", "", `{"metadata":{"a":"1"}}`},
		// The typed metadata map is not a generic envelope; the extension sits beside it.
		{http.MethodPost, "/nova/v2.1/servers/s1/metadata", "", `{"metadata":{"c":"3"},"x_extension":1}`},
		{http.MethodGet, "/nova/v2.1/servers/s1/metadata/a", "", ""},
		{http.MethodPut, "/nova/v2.1/servers/s1/metadata/a", "", `{"meta":{"a":"9"}}`},
		{http.MethodDelete, "/nova/v2.1/servers/s1/metadata/a/b", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeServerMetadataStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*servers.API) error
	}{
		{"Metadata", []int{200}, func(api *servers.API) error { _, err := api.Metadata(ctx, "s1"); return err }},
		{"Metadatum", []int{200}, func(api *servers.API) error { _, err := api.Metadatum(ctx, "s1", "a"); return err }},
		{"ResetMetadata", []int{200}, func(api *servers.API) error {
			_, err := api.ResetMetadata(ctx, "s1", servers.MetadataOpts{})
			return err
		}},
		{"UpdateMetadata", []int{200}, func(api *servers.API) error {
			_, err := api.UpdateMetadata(ctx, "s1", servers.MetadataOpts{})
			return err
		}},
		{"CreateMetadatum", []int{200}, func(api *servers.API) error {
			_, err := api.CreateMetadatum(ctx, "s1", servers.MetadatumOpts{"a": "1"})
			return err
		}},
		{"DeleteMetadatum", []int{202, 204}, func(api *servers.API) error { return api.DeleteMetadatum(ctx, "s1", "a") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Provider.HTTPClient.Transport = nativeServerTransport(func(req *http.Request) (*http.Response, error) {
					requests.Add(1)
					return nativeServerWire(code, `{"metadata":{},"meta":{}}`), nil
				})
				err := call.call(servers.New(nativeServerClient(cloud)))
				nativeServerOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || requests.Load() != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("one metadatum pair, collisions and nil options", func(t *testing.T) {
		cloud := testcloud.New(t)
		var requests atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeServerTransport(func(req *http.Request) (*http.Response, error) {
			requests.Add(1)
			return nativeServerWire(200, `{}`), nil
		})
		api := servers.New(nativeServerClient(cloud))
		checks := map[string]func() error{
			"empty metadatum": func() error { _, err := api.CreateMetadatum(ctx, "s1", servers.MetadatumOpts{}); return err },
			"two metadatum pairs": func() error {
				_, err := api.CreateMetadatum(ctx, "s1", servers.MetadatumOpts{"a": "1", "b": "2"})
				return err
			},
			"metadata collision": func() error {
				_, err := api.ResetMetadata(ctx, "s1", servers.MetadataOpts{}, servers.WithResetMetadataField("metadata", map[string]string{}))
				return err
			},
			"nil option": func() error { _, err := api.UpdateMetadata(ctx, "s1", servers.MetadataOpts{}, nil); return err },
		}
		for name, check := range checks {
			if err := check(); err == nil {
				t.Fatal(name, "accepted")
			}
		}
		if requests.Load() != 0 {
			t.Fatal(requests.Load())
		}
	})
}
