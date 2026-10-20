package image_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

type ownedUploadReader struct {
	*strings.Reader
	closed atomic.Int32
}

func (r *ownedUploadReader) Close() error { r.closed.Add(1); return nil }

func uploadInput(data io.Reader) image.UploadImageRequest {
	return image.UploadImageRequest{Name: "ubuntu", Data: data}
}

func TestUploadCreatesFlatMetadataThenStreamsDataAndWaits(t *testing.T) {
	cloud := testcloud.New(t)
	service := image.New(cloud.Client("image", "/v2"))
	var sequence []string
	cloud.Mux.HandleFunc("POST /v2/images", func(w http.ResponseWriter, r *http.Request) {
		sequence = append(sequence, "create")
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		want := map[string]any{
			"name": "ubuntu", "disk_format": "raw", "container_format": "bare", "visibility": "shared",
			"min_disk": float64(0), "min_ram": float64(512), "protected": false, "os_hidden": false,
			"tags": []any{"linux"}, "hw_architecture": "aarch64", "vendor_hint": map[string]any{"enabled": false},
		}
		if !reflect.DeepEqual(body, want) {
			t.Errorf("metadata=%#v want=%#v", body, want)
		}
		testcloud.JSON(w, 201, `{"id":"created","name":"ubuntu","status":"queued"}`)
	})
	cloud.Mux.HandleFunc("PUT /v2/images/created/file", func(w http.ResponseWriter, r *http.Request) {
		sequence = append(sequence, "upload")
		if r.Header.Get("Content-Type") != "application/octet-stream" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-OpenStack-Image-Size") != "4" {
			t.Errorf("headers=%v", r.Header)
		}
		data, err := io.ReadAll(r.Body)
		if err != nil || !bytes.Equal(data, []byte{0, 1, 2, 255}) {
			t.Errorf("data=%v err=%v", data, err)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	cloud.Mux.HandleFunc("GET /v2/images/created", func(w http.ResponseWriter, r *http.Request) {
		sequence = append(sequence, "wait")
		testcloud.JSON(w, 200, `{"id":"created","name":"ubuntu","status":"active","size":4,"hw_architecture":"aarch64"}`)
	})
	properties := map[string]any{"hw_architecture": "aarch64", "vendor_hint": map[string]any{"enabled": false}}
	propertiesOption := image.WithProperties(properties)
	properties["hw_architecture"] = "mutated"
	properties["vendor_hint"].(map[string]any)["enabled"] = true
	tags := []string{"linux"}
	tagsOption := image.WithTags(tags...)
	tags[0] = "mutated"
	created, err := service.Upload(context.Background(), uploadInput(bytes.NewReader([]byte{0, 1, 2, 255})),
		image.WithDiskFormat("raw"), image.WithContainerFormat("bare"), image.WithVisibility(image.VisibilityShared),
		image.WithMinDisk(0), image.WithMinRAM(512), image.WithProtected(false), image.WithHidden(false),
		propertiesOption, tagsOption, image.WithUploadSize(4), image.WithWait())
	if err != nil || created == nil || created.Status != "active" || created.SizeBytes != 4 {
		t.Fatalf("created=%v err=%v", created, err)
	}
	if !reflect.DeepEqual(sequence, []string{"create", "upload", "wait"}) {
		t.Fatalf("sequence=%v", sequence)
	}
	if service.RawClient().MoreHeaders["X-OpenStack-Image-Size"] != "" {
		t.Fatal("upload size must not mutate the shared service client")
	}
}

func TestUploadDefaultsAndCallerReaderOwnership(t *testing.T) {
	cloud := testcloud.New(t)
	service := image.New(cloud.Client("image", "/v2"))
	cloud.Mux.HandleFunc("POST /v2/images", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		want := map[string]any{"name": "ubuntu", "disk_format": "qcow2", "container_format": "bare", "visibility": "private"}
		if !reflect.DeepEqual(body, want) {
			t.Errorf("default metadata=%#v", body)
		}
		testcloud.JSON(w, 201, `{"id":"created","name":"ubuntu","status":"queued"}`)
	})
	cloud.Mux.HandleFunc("PUT /v2/images/created/file", func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil || string(data) != "image-data" {
			t.Errorf("data=%q err=%v", data, err)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("without WithWait, upload must not fetch or delete: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	reader := &ownedUploadReader{Reader: strings.NewReader("prefix-image-data")}
	_, _ = reader.Seek(int64(len("prefix-")), io.SeekStart)
	created, err := service.Upload(context.Background(), uploadInput(reader))
	if err != nil || created == nil || created.Status != "queued" || reader.closed.Load() != 0 {
		t.Fatalf("created=%v err=%v close calls=%d", created, err, reader.closed.Load())
	}
	if err := reader.Close(); err != nil || reader.closed.Load() != 1 {
		t.Fatal("caller must retain Close ownership")
	}
}

func TestUploadValidatesBeforeMetadataRequestOrReadingData(t *testing.T) {
	cloud := testcloud.New(t)
	service := image.New(cloud.Client("image", "/v2"))
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("invalid input must not request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	reader := &ownedUploadReader{Reader: strings.NewReader("data")}
	var nilReader *ownedUploadReader
	for _, tc := range []struct {
		name    string
		input   image.UploadImageRequest
		options []image.UploadImageOption
	}{
		{name: "no name", input: image.UploadImageRequest{Data: reader}},
		{name: "no reader", input: uploadInput(nil)},
		{name: "typed nil reader", input: uploadInput(nilReader)},
		{name: "disk format", input: uploadInput(reader), options: []image.UploadImageOption{image.WithDiskFormat(" ")}},
		{name: "container format", input: uploadInput(reader), options: []image.UploadImageOption{image.WithContainerFormat("")}},
		{name: "visibility", input: uploadInput(reader), options: []image.UploadImageOption{image.WithVisibility("unknown")}},
		{name: "negative min disk", input: uploadInput(reader), options: []image.UploadImageOption{image.WithMinDisk(-1)}},
		{name: "negative min RAM", input: uploadInput(reader), options: []image.UploadImageOption{image.WithMinRAM(-1)}},
		{name: "negative upload size", input: uploadInput(reader), options: []image.UploadImageOption{image.WithUploadSize(-1)}},
		{name: "empty property", input: uploadInput(reader), options: []image.UploadImageOption{image.WithProperty("", true)}},
		{name: "core property", input: uploadInput(reader), options: []image.UploadImageOption{image.WithProperty("name", "override")}},
		{name: "omitted core property", input: uploadInput(reader), options: []image.UploadImageOption{image.WithProperty("protected", true)}},
		{name: "readonly property", input: uploadInput(reader), options: []image.UploadImageOption{image.WithProperty("status", "active")}},
		{name: "invalid JSON property", input: uploadInput(reader), options: []image.UploadImageOption{image.WithProperty("vendor", make(chan int))}},
		{name: "invalid waiter", input: uploadInput(reader), options: []image.UploadImageOption{image.WithWait(resource.WithTimeout(0))}},
		{name: "nil option", input: uploadInput(reader), options: []image.UploadImageOption{nil}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := service.Upload(context.Background(), tc.input, tc.options...); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("err=%v", err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Upload(ctx, uploadInput(reader)); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if reader.Len() != len("data") || reader.closed.Load() != 0 {
		t.Fatalf("reader changed before POST: unread=%d close calls=%d", reader.Len(), reader.closed.Load())
	}
}

func TestUploadFailureKeepsCreatedImageAndHTTPError(t *testing.T) {
	cloud := testcloud.New(t)
	service := image.New(cloud.Client("image", "/v2"))
	cloud.Mux.HandleFunc("POST /v2/images", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 201, `{"id":"created","status":"queued"}`)
	})
	cloud.Mux.HandleFunc("PUT /v2/images/created/file", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		testcloud.JSON(w, 400, `{"message":"upload rejected"}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("upload failure must not wait or delete: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	reader := &ownedUploadReader{Reader: strings.NewReader("data")}
	created, err := service.Upload(context.Background(), uploadInput(reader), image.WithWait())
	var responseError gophercloud.ErrUnexpectedResponseCode
	if created == nil || created.ID != "created" || !errors.As(err, &responseError) || responseError.Actual != 400 || reader.closed.Load() != 0 {
		t.Fatalf("created=%v err=%v close calls=%d", created, err, reader.closed.Load())
	}
}

func TestUploadWaitFailureKeepsCreatedImage(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status string
		want   error
	}{
		{name: "failed state", status: "killed", want: resource.ErrFailedState},
		{name: "timeout", status: "saving", want: context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			service := image.New(cloud.Client("image", "/v2"))
			cloud.Mux.HandleFunc("POST /v2/images", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 201, `{"id":"created","status":"queued"}`)
			})
			cloud.Mux.HandleFunc("PUT /v2/images/created/file", func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				w.WriteHeader(http.StatusNoContent)
			})
			cloud.Mux.HandleFunc("GET /v2/images/created", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"id":"created","status":"`+tc.status+`"}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("wait failure must not delete: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected request", 500)
			})
			created, err := service.Upload(context.Background(), uploadInput(strings.NewReader("data")), image.WithWait(resource.WithTimeout(10*time.Millisecond)))
			if created == nil || created.ID != "created" || created.Status != "queued" || !errors.Is(err, tc.want) {
				t.Fatalf("created=%v err=%v", created, err)
			}
		})
	}
}

func TestUploadMetadataFailureDoesNotConsumeData(t *testing.T) {
	cloud := testcloud.New(t)
	service := image.New(cloud.Client("image", "/v2"))
	cloud.Mux.HandleFunc("POST /v2/images", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 403, `{"message":"permission denied"}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("failed metadata must not upload or delete: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	reader := &ownedUploadReader{Reader: strings.NewReader("data")}
	created, err := service.Upload(context.Background(), uploadInput(reader))
	var responseError gophercloud.ErrUnexpectedResponseCode
	if created != nil || !errors.As(err, &responseError) || responseError.Actual != 403 || reader.Len() != len("data") || reader.closed.Load() != 0 {
		t.Fatalf("created=%v err=%v unread=%d close calls=%d", created, err, reader.Len(), reader.closed.Load())
	}
}

func TestUploadRejectsMalformedCreatedIDWithoutUpload(t *testing.T) {
	cloud := testcloud.New(t)
	service := image.New(cloud.Client("image", "/v2"))
	cloud.Mux.HandleFunc("POST /v2/images", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 201, `{"id":"../unexpected","status":"queued"}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("invalid image ID must not upload: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	created, err := service.Upload(context.Background(), uploadInput(strings.NewReader("data")))
	if created == nil || created.ID != "../unexpected" || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("created=%v err=%v", created, err)
	}
}

func TestUploadPartialMetadataDecodeKeepsCreatedImage(t *testing.T) {
	cloud := testcloud.New(t)
	service := image.New(cloud.Client("image", "/v2"))
	cloud.Mux.HandleFunc("POST /v2/images", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 201, `{"id":"created","status":"queued","size":"invalid"}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("invalid metadata must not upload or delete: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	reader := &ownedUploadReader{Reader: strings.NewReader("data")}
	created, err := service.Upload(context.Background(), uploadInput(reader))
	if created == nil || created.ID != "created" || err == nil || !strings.Contains(err.Error(), "unknown type for SizeBytes") || reader.Len() != len("data") || reader.closed.Load() != 0 {
		t.Fatalf("created=%v err=%v unread=%d close calls=%d", created, err, reader.Len(), reader.closed.Load())
	}
	var operationError *resource.OperationError
	if !errors.As(err, &operationError) || operationError.Cause == nil {
		t.Fatalf("decode error cause was not retained: %v", err)
	}
}
