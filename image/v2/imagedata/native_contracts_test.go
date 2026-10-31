package imagedata_test

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

	"github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func nativeDataClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("image", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/reverse/glance/v2/"
	client.MoreHeaders = map[string]string{"X-Source": "direct"}
	return client
}

func nativeDataOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "imagedata" {
		t.Fatal("generated imagedata context", err, wrapped)
	}
	var receipt *resource.ResponseError
	if errors.As(err, &receipt) {
		t.Fatal("native data call fabricated an owned receipt", receipt)
	}
}

// Upload and Stage stream the caller reader once with octet-stream and accept
// only 204; neither reads a response model.
func TestNativeImageDataUploadAndStage(t *testing.T) {
	for _, test := range []struct {
		name, suffix string
		call         func(*imagedata.API, context.Context, string, io.Reader) error
	}{
		{"Upload", "file", (*imagedata.API).Upload},
		{"Stage", "stage", (*imagedata.API).Stage},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			code := 204
			var calls atomic.Int32
			cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				raw, _ := io.ReadAll(req.Body)
				if req.Method != http.MethodPut || req.URL.Path != "/reverse/glance/v2/images/a/b/"+test.suffix || string(raw) != "payload" || req.Header.Get("Content-Type") != "application/octet-stream" || req.Header.Get("X-Source") != "direct" {
					t.Error(req.Method, req.URL, string(raw), req.Header)
				}
				return testcloud.Response(code, "opaque", http.Header{"X-Data-Proof": {"actual"}}), nil
			}))
			api := imagedata.New(nativeDataClient(cloud))
			if err := test.call(api, context.Background(), "a/b", strings.NewReader("payload")); err != nil || calls.Load() != 1 {
				t.Fatal(err, calls.Load())
			}
			for _, status := range []int{200, 201, 202, 409, 413} {
				code = status
				err := test.call(api, context.Background(), "a/b", strings.NewReader("payload"))
				nativeDataOperation(t, err, test.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != status || !reflect.DeepEqual(native.Expected, []int{204}) || native.ResponseHeader.Get("X-Data-Proof") != "actual" {
					t.Fatal(status, err)
				}
			}
		})
	}
}

// Download hands the open native body to the caller. The generated Header
// field holds the same ReadCloser returned by native Extract, not HTTP headers.
func TestNativeImageDataDownloadOwnershipAndStatus(t *testing.T) {
	cloud := testcloud.New(t)
	code := 200
	var body *testcloud.Body
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/reverse/glance/v2/images/img/file" || req.Header.Get("X-Source") != "direct" {
			t.Error(req.Method, req.URL, req.Header)
		}
		body = &testcloud.Body{Data: "binary\xff"}
		return testcloud.StreamResponse(code, body, http.Header{"Content-Md5": {"x"}}), nil
	}))
	api := imagedata.New(nativeDataClient(cloud))
	download, err := api.Download(context.Background(), "img")
	if err != nil || download == nil || download.Header != download.Body {
		t.Fatal(download, err)
	}
	data, err := io.ReadAll(download)
	if err != nil || string(data) != "binary\xff" || body.Closes() != 0 {
		t.Fatal(string(data), err, body.Closes())
	}
	if err := download.Close(); err != nil || body.Closes() != 1 {
		t.Fatal(err, body.Closes())
	}
	for _, status := range []int{204, 206, 404} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			code = status
			download, err := api.Download(context.Background(), "img")
			nativeDataOperation(t, err, "Download")
			var native gophercloud.ErrUnexpectedResponseCode
			if download != nil || !errors.As(err, &native) || native.Actual != status || !reflect.DeepEqual(native.Expected, []int{200}) {
				t.Fatal(download, err)
			}
		})
	}
}
