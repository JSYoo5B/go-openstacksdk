package imageimport_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeImportTransport func(*http.Request) (*http.Response, error)

func (transport nativeImportTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeImportWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}, "X-Import-Proof": {"actual"}}}
}

func nativeImportClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("image", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/reverse/glance/v2/"
	client.MoreHeaders = map[string]string{"X-Source": "direct"}
	return client
}

func nativeImportOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "imageimport" {
		t.Fatal("generated import context", err, wrapped)
	}
	var receipt *resource.ResponseError
	if errors.As(err, &receipt) {
		t.Fatal("native import call fabricated an owned receipt", receipt)
	}
}

func TestNativeImageImportInfoGet(t *testing.T) {
	cloud := testcloud.New(t)
	client := nativeImportClient(cloud)
	code := 200
	var calls atomic.Int32
	cloud.Provider.HTTPClient.Transport = nativeImportTransport(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		if req.Method != http.MethodGet || req.URL.Path != "/reverse/glance/v2/info/import" || req.URL.RawQuery != "" || req.Body != nil || req.Header.Get("Accept") != "application/json" || req.Header.Get("X-Source") != "direct" {
			t.Error(req.Method, req.URL, req.Header)
		}
		return nativeImportWire(code, `{"import-methods":{"description":"d","type":"array","value":["glance-direct","web-download"]},"extra":1}`), nil
	})
	api := imageimport.New(client)
	if api.RawClient() != client {
		t.Fatal("native client identity changed")
	}
	info, err := api.Get(context.Background())
	want := imageimport.ImportMethods{Description: "d", Type: "array", Value: []string{"glance-direct", "web-download"}}
	if err != nil || !reflect.DeepEqual(info.ImportMethods, want) || calls.Load() != 1 {
		t.Fatal(info, err)
	}
	for _, status := range []int{201, 203, 204, 403} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			code = status
			info, err := api.Get(context.Background())
			nativeImportOperation(t, err, "Get")
			var native gophercloud.ErrUnexpectedResponseCode
			if info != nil || !errors.As(err, &native) || native.Actual != status || !reflect.DeepEqual(native.Expected, []int{200}) || native.ResponseHeader.Get("X-Import-Proof") != "actual" {
				t.Fatal(info, err, native)
			}
		})
	}
}

// Create wraps CreateOpts as {"method": {...}}. Because that body has one
// object member, the shared extension merge writes fields into the method.
func TestNativeImageImportCreateBodyExtensionsAndStatus(t *testing.T) {
	for _, test := range []struct {
		name, id, path string
		opts           imageimport.CreateOpts
		options        []imageimport.CreateOption
		want           string
	}{
		{"web download", "img", "/reverse/glance/v2/images/img/import", imageimport.CreateOpts{Name: imageimport.WebDownloadMethod, URI: "https://example.test/i.qcow2"}, nil,
			`{"method":{"name":"web-download","uri":"https://example.test/i.qcow2"}}`},
		{"empty values are sent", "img", "/reverse/glance/v2/images/img/import", imageimport.CreateOpts{}, nil, `{"method":{"name":"","uri":""}}`},
		{"extension joins method", "img", "/reverse/glance/v2/images/img/import", imageimport.CreateOpts{Name: "glance-download"}, []imageimport.CreateOption{imageimport.WithCreateField("glance_image_id", "src")},
			`{"method":{"glance_image_id":"src","name":"glance-download","uri":""}}`},
		{"raw id path", "a/b", "/reverse/glance/v2/images/a/b/import", imageimport.CreateOpts{Name: imageimport.GlanceDirectMethod}, nil, `{"method":{"name":"glance-direct","uri":""}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = nativeImportTransport(func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				raw, _ := io.ReadAll(req.Body)
				if req.Method != http.MethodPost || req.URL.Path != test.path || string(raw) != test.want || req.Header.Get("Content-Type") != "application/json" || req.Header.Get("X-Source") != "direct" {
					t.Error(req.Method, req.URL, string(raw), req.Header)
				}
				return nativeImportWire(202, "opaque accepted bytes"), nil
			})
			if err := imageimport.New(nativeImportClient(cloud)).Create(context.Background(), test.id, test.opts, test.options...); err != nil || calls.Load() != 1 {
				t.Fatal(err, calls.Load())
			}
		})
	}
	t.Run("strict 202 and preflight", func(t *testing.T) {
		cloud := testcloud.New(t)
		code := 200
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeImportTransport(func(req *http.Request) (*http.Response, error) {
			calls.Add(1)
			return nativeImportWire(code, `{}`), nil
		})
		api := imageimport.New(nativeImportClient(cloud))
		for _, option := range []imageimport.CreateOption{imageimport.WithCreateField("name", "x"), imageimport.WithCreateField("uri", "x"), nil} {
			err := api.Create(context.Background(), "img", imageimport.CreateOpts{}, option)
			nativeImportOperation(t, err, "Create")
			if calls.Load() != 0 {
				t.Fatal(err, calls.Load())
			}
		}
		for _, status := range []int{200, 201, 204, 409} {
			code = status
			err := api.Create(context.Background(), "img", imageimport.CreateOpts{Name: imageimport.GlanceDirectMethod})
			nativeImportOperation(t, err, "Create")
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != status || !reflect.DeepEqual(native.Expected, []int{202}) {
				t.Fatal(status, err)
			}
		}
	})
}
