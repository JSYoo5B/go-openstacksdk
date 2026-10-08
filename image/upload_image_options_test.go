package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestImageUploadOptionsFactoriesAndSnapshots(t *testing.T) {
	count := 0
	single := WithImageUploadField("custom", taskOptionMarshaler{calls: &count})
	raw := json.RawMessage(`{"snap":"owned"}`)
	headers := map[string]string{"X-Policy": "owned"}
	size := int64(0)
	full := WithImageUploadOpts(ImageUploadOpts{Headers: headers, Fields: map[string]json.RawMessage{"custom": raw}, Size: &size})
	fields := WithImageUploadFields(map[string]json.RawMessage{"custom": raw})
	raw[9] = 'X'
	headers["X-Policy"] = "changed"
	size = 99
	calls := 0
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method == "POST" {
			body := taskCorePayload(t, req)
			if string(body["custom"]) != `{"snap":"owned"}` {
				t.Fatal(body)
			}
			if calls == 3 && req.Header.Get("X-Policy") != "owned" {
				t.Fatal(req.Header)
			}
			return taskCoreJSON(req, 201, `{"id":"id"}`), nil
		}
		if calls == 4 && req.Header.Get("X-OpenStack-Image-Size") != "0" {
			t.Fatal(req.Header)
		}
		return taskCoreJSON(req, 204, ""), nil
	}))
	for _, option := range []ImageUploadOption{single, full, fields} {
		if v, e := service.UploadImage(context.Background(), imageUploadCoreInput(strings.NewReader("data")), option); v == nil || e != nil {
			t.Fatal(v, e)
		}
	}
	if calls != 6 || count != 1 {
		t.Fatal(calls, count)
	}
	for _, raw := range []json.RawMessage{nil, {}, json.RawMessage("bad"), {'"', 0xff, '"'}} {
		for _, option := range []ImageUploadOption{WithImageUploadFields(map[string]json.RawMessage{"x": raw}), WithImageUploadOpts(ImageUploadOpts{Fields: map[string]json.RawMessage{"x": raw}})} {
			if v, e := service.UploadImage(context.Background(), imageUploadCoreInput(strings.NewReader("unread")), option); v != nil || !errors.Is(e, resource.ErrInvalidOption) {
				t.Fatal(v, e)
			}
		}
	}
	marshalCount := 0
	cause := errors.New("factory cause")
	invalidKey := WithImageUploadField(string([]byte{0xff}), taskOptionMarshaler{calls: &marshalCount})
	ownedName := WithImageUploadField("name", taskOptionMarshaler{calls: &marshalCount})
	failed := WithImageUploadField("x", taskOptionMarshaler{calls: &marshalCount, cause: cause})
	if marshalCount != 1 {
		t.Fatal("bad key invoked marshaler", marshalCount)
	}
	for _, option := range []ImageUploadOption{invalidKey, ownedName, failed, WithImageUploadField("x", make(chan int))} {
		if v, e := service.UploadImage(context.Background(), imageUploadCoreInput(strings.NewReader("unread")), option); v != nil || e == nil {
			t.Fatal(v, e)
		}
	}
	if v, e := service.UploadImage(context.Background(), imageUploadCoreInput(strings.NewReader("unread")), failed); v != nil || !errors.Is(e, cause) || calls != 6 || marshalCount != 1 {
		t.Fatal(v, e, calls, marshalCount)
	}
}

func TestImageUploadOptionsReplacementAndTypedPresence(t *testing.T) {
	calls := 0
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Erased") != "" || req.Header.Get("X-Policy") != "last" {
			t.Fatal(req.Header)
		}
		if req.Method == "POST" {
			body := taskCorePayload(t, req)
			if calls == 1 {
				expected := map[string]string{"name": `"literal\nname"`, "disk_format": `"future-disk"`, "container_format": `"future-container"`, "visibility": `"future"`, "owner": `""`, "protected": "false", "os_hidden": "false", "min_disk": "0", "min_ram": "-1", "tags": "[]", "custom": "null", "id": `"forwarded"`}
				if len(body) != len(expected) {
					t.Fatal(body)
				}
				for key, value := range expected {
					if string(body[key]) != value {
						t.Fatal(key, string(body[key]), value)
					}
				}
			} else if len(body) != 4 || taskCoreText(t, body["disk_format"]) != "qcow2" {
				t.Fatal(body)
			}
			return taskCoreJSON(req, 201, `{"id":"observed","name":"response"}`), nil
		}
		if req.Header.Get("X-OpenStack-Image-Size") != "" {
			t.Fatal("size not cleared", req.Header)
		}
		return taskCoreJSON(req, 204, ""), nil
	}))
	value, err := service.UploadImage(context.Background(), imageUploadCoreInput(strings.NewReader("data")), WithImageUploadField("erased", 1), WithImageUploadHeader("X-Erased", "old"), WithImageUploadSize(123), WithImageUploadOpts(ImageUploadOpts{}), WithImageUploadHeaders(map[string]string{"x-policy": "first"}), WithImageUploadHeader("X-Policy", "last"), WithImageUploadDiskFormat("future-disk"), WithImageUploadContainerFormat("future-container"), WithImageUploadVisibility(Visibility("future")), WithImageUploadOwner(""), WithImageUploadProtected(false), WithImageUploadHidden(false), WithImageUploadMinDisk(0), WithImageUploadMinRAM(-1), WithImageUploadTags(), WithImageUploadField("custom", nil), WithImageUploadField("id", "forwarded"), WithImageUploadSize(999), WithoutImageUploadSize())
	if value == nil || err != nil || value.ImageID != "observed" || *value.Image.Name != "response" {
		t.Fatal(value, err)
	}
	value, err = service.UploadImage(context.Background(), imageUploadCoreInput(strings.NewReader("data")), WithImageUploadHeader("X-Policy", "last"), WithImageUploadFields(map[string]json.RawMessage{"erased": json.RawMessage("1")}), WithImageUploadFields(nil))
	if value == nil || err != nil || calls != 4 {
		t.Fatal(value, err, calls)
	}
	tags := []string{"one", "one", "two"}
	option := WithImageUploadTags(tags...)
	tags[0] = "changed"
	config := copyImageUploadOpts(ImageUploadOpts{})
	if err := option(&config); err != nil || string(config.Fields["tags"]) != `["one","one","two"]` {
		t.Fatal(config, err)
	}
}

func TestImageUploadOptionsRetainedCallbacksAndSource(t *testing.T) {
	callbacks, calls := 0, 0
	var retained *ImageUploadOpts
	sourceHeaders := map[string]string{"X-Source": "captured"}
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		retained.Headers["X-Policy"] = "late-change"
		retained.Fields["custom"][0] = '0'
		*retained.Size = 9
		sourceHeaders["X-Source"] = "later"
		if req.Header.Get("X-Source") != "captured" || req.Header.Get("X-Policy") != "owned" {
			t.Fatal(req.Header)
		}
		if req.Method == "POST" {
			body := taskCorePayload(t, req)
			if string(body["custom"]) != "null" {
				t.Fatal(body)
			}
			return taskCoreJSON(req, 201, `{"id":"id"}`), nil
		}
		if req.Header.Get("X-OpenStack-Image-Size") != "0" {
			t.Fatal(req.Header)
		}
		return taskCoreJSON(req, 204, ""), nil
	})
	client.MoreHeaders = sourceHeaders
	first := func(config *ImageUploadOpts) error {
		callbacks++
		if config.Headers == nil || config.Fields == nil {
			t.Fatal("callback maps nil")
		}
		config.Headers["X-Policy"] = "owned"
		config.Fields["custom"] = json.RawMessage("null")
		config.Size = taskOptionPointer(int64(0))
		retained = config
		return nil
	}
	second := func(config *ImageUploadOpts) error {
		callbacks++
		retained.Headers["X-Policy"] = "previous"
		retained.Fields["custom"][0] = '0'
		*retained.Size = 99
		if config.Headers["X-Policy"] != "owned" || string(config.Fields["custom"]) != "null" || *config.Size != 0 {
			t.Fatal(config)
		}
		return nil
	}
	if value, err := New(client).UploadImage(context.Background(), imageUploadCoreInput(strings.NewReader("data")), first, second); value == nil || err != nil || callbacks != 2 || calls != 2 {
		t.Fatal(value, err, callbacks, calls)
	}
	cause := errors.New("callback cause")
	if value, err := New(client).UploadImage(context.Background(), imageUploadCoreInput(strings.NewReader("unread")), func(*ImageUploadOpts) error { return cause }); value != nil || !errors.Is(err, cause) || calls != 2 {
		t.Fatal(value, err, calls)
	}
	for _, field := range []string{"endpoint", "base", "type", "version", "provider", "client"} {
		t.Run(field, func(t *testing.T) {
			requests := 0
			client := taskCoreClient(func(*http.Request) (*http.Response, error) { requests++; return nil, nil })
			service := New(client)
			option := func(*ImageUploadOpts) error {
				switch field {
				case "endpoint":
					client.Endpoint = "https://glance.example/other/"
				case "base":
					client.ResourceBase = "https://glance.example/other/"
				case "type":
					client.Type = "other"
				case "version":
					client.Microversion = "changed"
				case "provider":
					client.ProviderClient = taskCoreClient(nil).ProviderClient
				case "client":
					service.client = taskCoreClient(nil)
				}
				return nil
			}
			if value, err := service.UploadImage(context.Background(), imageUploadCoreInput(strings.NewReader("unread")), option); value != nil || !errors.Is(err, resource.ErrInvalidOption) || requests != 0 {
				t.Fatal(value, err, requests)
			}
		})
	}
}

func TestImageUploadOptionsParallelReuseAndHeaderPolicies(t *testing.T) {
	var calls atomic.Int64
	fields := map[string]json.RawMessage{"precision": json.RawMessage("900719925474099312345"), "custom": json.RawMessage(`{"n":1}`)}
	headers := map[string]string{"X-Policy": "fixed"}
	size := int64(4)
	policy := WithImageUploadOpts(ImageUploadOpts{Headers: headers, Fields: fields, Size: &size})
	fields["custom"][5] = '9'
	headers["X-Policy"] = "changed"
	size = 99
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		if req.Header.Get("X-Policy") != "fixed" {
			return nil, fmt.Errorf("changed header: %v", req.Header)
		}
		if req.Method == "POST" {
			body := taskCorePayload(t, req)
			if string(body["precision"]) != "900719925474099312345" || string(body["custom"]) != `{"n":1}` {
				return nil, fmt.Errorf("changed JSON: %v", body)
			}
			return taskCoreJSON(req, 201, `{"id":"id"}`), nil
		}
		if req.Header.Get("X-OpenStack-Image-Size") != "4" || imageUploadCoreRead(t, req) != "data" {
			return nil, fmt.Errorf("changed binary request")
		}
		return taskCoreJSON(req, 204, ""), nil
	})
	service := New(client)
	var wg sync.WaitGroup
	outcomes := make(chan error, 12)
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, err := service.UploadImage(context.Background(), imageUploadCoreInput(strings.NewReader("data")), policy)
			if err == nil && (value == nil || value.Acknowledgement == nil) {
				err = errors.New("missing result")
			}
			outcomes <- err
		}()
	}
	wg.Wait()
	close(outcomes)
	for err := range outcomes {
		if err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 24 {
		t.Fatal(calls.Load())
	}
	for _, key := range []string{"Authorization", "Host", "Content-Type", "Accept", "Content-Length", "X-OpenStack-Image-Size", "x-openstack-image-size", "OpenStack-API-Version"} {
		if value, err := service.UploadImage(context.Background(), imageUploadCoreInput(strings.NewReader("unread")), WithImageUploadHeader(key, "owned")); value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 24 {
			t.Fatal(key, value, err)
		}
	}
	value := copyImageUploadOpts(ImageUploadOpts{})
	merge := WithImageUploadHeaders(map[string]string{"x-policy": "one"})
	if err := merge(&value); err != nil {
		t.Fatal(err)
	}
	if err := WithImageUploadHeader("X-Policy", "two")(&value); err != nil || !reflect.DeepEqual(value.Headers, map[string]string{"X-Policy": "two"}) {
		t.Fatal(value, err)
	}
}
