package image

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// The Cloud entry finds by name first, then the lower download repeats the
// mandatory metadata GET before the binary GET of the fetched identity.
func cloudDownloadTransport(t *testing.T, calls *int, binary func(*http.Request) *http.Response) func(*http.Request) *http.Response {
	return func(req *http.Request) *http.Response {
		if req.Header.Get("X-Cloud") != "download" {
			t.Fatal(*calls, req.URL, req.Header)
		}
		switch *calls {
		case 1:
			if req.URL.EscapedPath() != "/reverse/glance/v2/images/ubuntu" {
				t.Fatal(req.URL)
			}
			return taskCoreJSON(req, 404, `{"message":"missing"}`)
		case 2:
			if req.URL.Query().Get("name") != "ubuntu" {
				t.Fatal(req.URL)
			}
			return taskCoreJSON(req, 200, `{"images":[{"id":"img","name":"ubuntu","status":"queued"}]}`)
		case 3:
			if req.URL.EscapedPath() != "/reverse/glance/v2/images/img" {
				t.Fatal(req.URL)
			}
			return taskCoreJSON(req, 200, `{"id":"img","name":"ubuntu","status":"active","os_hash_algo":"sha256","os_hash_value":"`+imageRecordDownloadABCSHA256+`"}`)
		case 4:
			if req.URL.EscapedPath() != "/reverse/glance/v2/images/img/file" {
				t.Fatal(req.URL)
			}
			return binary(req)
		}
		t.Fatal("unexpected request", req.URL)
		return nil
	}
}

func TestCloudImageRecordDownloadFindsThenDownloadsToRequiredOutput(t *testing.T) {
	binary := func(req *http.Request) *http.Response {
		return taskCoreHTTP(req, 200, io.NopCloser(strings.NewReader("abc")))
	}
	header := WithImageRecordCloudDownloadHeader("X-Cloud", "download")
	t.Run("writer", func(t *testing.T) {
		calls := 0
		service := cloudImageService(t, &calls, cloudDownloadTransport(t, &calls, binary))
		var output bytes.Buffer
		// Stream is forwarded but the required destination takes precedence.
		result, err := service.DownloadCloudImageRecord(context.Background(), ImageRecordCloudDownloadRequest{NameOrID: "ubuntu", Output: &output}, header, WithImageRecordCloudDownloadStream(true), WithImageRecordCloudDownloadChunkSize(1))
		if err != nil || calls != 4 || output.String() != "abc" || result.Found == nil || imageRecordText(result.Found, "status") != "queued" {
			t.Fatal(result, err, calls, output.String())
		}
		if result.Download == nil || result.Download.BytesWritten != 3 || imageRecordText(result.Download.Record, "status") != "active" || result.Download.Checksum == nil || !result.Download.Checksum.Verified || result.Download.Downloaded.Stream != nil {
			t.Fatal(result.Download)
		}
	})
	t.Run("filename", func(t *testing.T) {
		calls := 0
		service := cloudImageService(t, &calls, cloudDownloadTransport(t, &calls, binary))
		path := filepath.Join(t.TempDir(), "image.raw")
		result, err := service.DownloadCloudImageRecord(context.Background(), ImageRecordCloudDownloadRequest{NameOrID: "ubuntu", Filename: path}, WithImageRecordCloudDownloadOpts(ImageRecordCloudDownloadOpts{Headers: map[string]string{"X-Cloud": "download"}}))
		data, readErr := os.ReadFile(path)
		if err != nil || readErr != nil || string(data) != "abc" || calls != 4 || result.Download.BytesWritten != 3 {
			t.Fatal(result, err, readErr, string(data))
		}
	})
	t.Run("binary failure keeps found and lower evidence", func(t *testing.T) {
		calls := 0
		service := cloudImageService(t, &calls, cloudDownloadTransport(t, &calls, func(req *http.Request) *http.Response {
			return taskCoreJSON(req, 500, `{"message":"server"}`)
		}))
		var output bytes.Buffer
		result, err := service.DownloadCloudImageRecord(context.Background(), ImageRecordCloudDownloadRequest{NameOrID: "ubuntu", Output: &output}, header)
		var operation *resource.OperationError
		if !gophercloud.ResponseCodeIs(err, http.StatusInternalServerError) || !errors.As(err, &operation) || operation.Operation != "DownloadCloudImageRecord" || errors.As(operation.Cause, &operation) {
			t.Fatal(err)
		}
		if result == nil || result.Found == nil || result.Download == nil || result.Download.Record == nil || output.Len() != 0 {
			t.Fatal(result)
		}
	})
}

func TestCloudImageRecordDownloadStrictFindAndPreflight(t *testing.T) {
	t.Run("missing image is an error before download", func(t *testing.T) {
		calls := 0
		service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
			if calls == 1 {
				return taskCoreJSON(req, 404, `{}`)
			}
			if calls > 3 {
				t.Fatal("unexpected request", req.URL)
			}
			return taskCoreJSON(req, 200, `{"images":[]}`)
		})
		var output bytes.Buffer
		result, err := service.DownloadCloudImageRecord(context.Background(), ImageRecordCloudDownloadRequest{NameOrID: "absent", Output: &output})
		if !errors.Is(err, resource.ErrNotFound) || result != nil || calls != 3 {
			t.Fatal(result, err, calls)
		}
	})
	calls := 0
	service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
		t.Fatal("unexpected request", req.URL)
		return nil
	})
	var typedNil *bytes.Buffer
	ctx := context.Background()
	for name, input := range map[string]ImageRecordCloudDownloadRequest{
		"no output":        {NameOrID: "ubuntu"},
		"both outputs":     {NameOrID: "ubuntu", Output: &bytes.Buffer{}, Filename: "image.raw"},
		"typed nil writer": {NameOrID: "ubuntu", Output: typedNil},
		"blank identity":   {NameOrID: " ", Output: &bytes.Buffer{}},
		"invalid filename": {NameOrID: "ubuntu", Filename: "\xff"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := service.DownloadCloudImageRecord(ctx, input); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(err, calls)
			}
		})
	}
	t.Run("nil option and invalid header", func(t *testing.T) {
		for _, option := range []ImageRecordCloudDownloadOption{nil, WithImageRecordCloudDownloadHeader("X-Bad", "a\nb")} {
			if _, err := service.DownloadCloudImageRecord(ctx, ImageRecordCloudDownloadRequest{NameOrID: "ubuntu", Output: &bytes.Buffer{}}, option); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(err, calls)
			}
		}
	})
}
