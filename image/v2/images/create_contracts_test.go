package images_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/images"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func nativeCreateOperation(t *testing.T, err error) {
	t.Helper()
	var operation *resource.OperationError
	if !errors.As(err, &operation) || operation.Operation != "Create" || operation.Resource != "images" {
		t.Fatal("generated create context", err, operation)
	}
	var receipt *resource.ResponseError
	if errors.As(err, &receipt) {
		t.Fatal("native create fabricated an owned receipt", receipt)
	}
}

// The facade forwards the pinned BuildRequestBody/Properties map and appends
// only non-core extension fields before the native POST with OkCodes 201.
func TestNativeImageCreateBodyPropertiesExtensionsAndProjection(t *testing.T) {
	visibility := images.ImageVisibilityPrivate
	hidden, protected := false, true
	for _, test := range []struct {
		name    string
		opts    images.CreateOpts
		options []images.CreateOption
		want    string
	}{
		{"name only omits empty fields", images.CreateOpts{Name: "n"}, nil, `{"name":"n"}`},
		{"declared and pointer fields", images.CreateOpts{Name: "n", ID: "fixed", Visibility: &visibility, Hidden: &hidden, Tags: []string{"a"}, ContainerFormat: "bare", DiskFormat: "raw", MinDisk: 1, MinRAM: 2, Protected: &protected},
			nil, `{"container_format":"bare","disk_format":"raw","id":"fixed","min_disk":1,"min_ram":2,"name":"n","os_hidden":false,"protected":true,"tags":["a"],"visibility":"private"}`},
		// Properties are copied after the declared body and can replace declared keys.
		{"properties override declared keys", images.CreateOpts{Name: "n", Properties: map[string]string{"name": "property", "hw_vendor": "x"}}, nil, `{"hw_vendor":"x","name":"property"}`},
		{"extension field", images.CreateOpts{Name: "n"}, []images.CreateOption{images.WithCreateField("os_distro", map[string]any{"raw": json.RawMessage(`1e400`)})}, `{"name":"n","os_distro":{"raw":1e400}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := nativeUpdateClient(cloud)
			client.MoreHeaders = map[string]string{"X-Source": "direct"}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = nativeDeleteTransport(func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				raw, _ := io.ReadAll(req.Body)
				// Native map bodies marshal with sorted keys; raw extension bytes stay exact.
				if string(raw) != test.want {
					t.Error(string(raw))
				}
				if req.Method != http.MethodPost || req.URL.Path != "/reverse/glance/v2/images" || req.URL.RawQuery != "" || req.Header.Get("Content-Type") != "application/json" || req.Header.Get("Accept") != "application/json" || req.Header.Get("X-Source") != "direct" {
					t.Error(req.Method, req.URL, req.Header)
				}
				wire := nativeDeleteWire(201, io.NopCloser(strings.NewReader(`{"id":"fixed","name":"n","status":"queued","size":null,"hw_vendor":"x"}`)))
				wire.Header.Set("OpenStack-image-import-methods", "glance-direct")
				return wire, nil
			})
			got, err := images.New(client).Create(context.Background(), test.opts, test.options...)
			if err != nil || calls.Load() != 1 || got.ID != "fixed" || got.Status != "queued" || got.SizeBytes != 0 || !reflect.DeepEqual(got.Properties, map[string]any{"hw_vendor": "x"}) || !reflect.DeepEqual(got.OpenStackImageImportMethods, []string{"glance-direct"}) {
				t.Fatal(got, err, calls.Load())
			}
		})
	}
}

func TestNativeImageCreatePreflightAndStrictStatus(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	code := 201
	cloud.Provider.HTTPClient.Transport = nativeDeleteTransport(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		return nativeDeleteWire(code, io.NopCloser(strings.NewReader(`{"id":"fixed"}`))), nil
	})
	api := images.New(nativeUpdateClient(cloud))
	for name, call := range map[string]func() error{
		"missing required name": func() error { _, err := api.Create(context.Background(), images.CreateOpts{}); return err },
		"extension collides with core field": func() error {
			_, err := api.Create(context.Background(), images.CreateOpts{Name: "n"}, images.WithCreateField("name", "x"))
			return err
		},
		"extension collides with property": func() error {
			_, err := api.Create(context.Background(), images.CreateOpts{Name: "n", Properties: map[string]string{"os_distro": "a"}}, images.WithCreateField("os_distro", "b"))
			return err
		},
		"nil option": func() error {
			_, err := api.Create(context.Background(), images.CreateOpts{Name: "n"}, nil)
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := call()
			nativeCreateOperation(t, err)
			if err == nil || calls.Load() != 0 {
				t.Fatal(err, calls.Load())
			}
		})
	}
	for _, status := range []int{200, 202, 409} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			code = status
			before := calls.Load()
			got, err := api.Create(context.Background(), images.CreateOpts{Name: "n"})
			nativeCreateOperation(t, err)
			var native gophercloud.ErrUnexpectedResponseCode
			if got != nil || !errors.As(err, &native) || native.Actual != status || !reflect.DeepEqual(native.Expected, []int{201}) || calls.Load() != before+1 {
				t.Fatal(got, err, native)
			}
		})
	}
}
