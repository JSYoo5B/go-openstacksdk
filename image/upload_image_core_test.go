package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type imageUploadCoreReader struct {
	reader               io.Reader
	reads, closes, seeks atomic.Int64
}

func (r *imageUploadCoreReader) Read(p []byte) (int, error) { r.reads.Add(1); return r.reader.Read(p) }
func (r *imageUploadCoreReader) Close() error               { r.closes.Add(1); return errors.New("borrowed close") }
func (r *imageUploadCoreReader) Seek(int64, int) (int64, error) {
	r.seeks.Add(1)
	return 0, errors.New("borrowed seek")
}
func imageUploadCoreInput(reader io.Reader) UploadImageRequest {
	return UploadImageRequest{Name: "literal\nname", Data: reader}
}
func imageUploadCoreRead(t *testing.T, req *http.Request) string {
	t.Helper()
	value, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(value)
}

func TestImageUploadCoreFixedRoutesAndIndependentEvidence(t *testing.T) {
	id := "이미지% ?#:parent"
	metadata := `{"id":"이미지% ?#:parent","name":"response","status":"queued","created_at":"literal","size":9007199254740993,"custom":{"precise":900719925474099312345},"links":false,"file":"https://foreign/file"}`
	reader := &imageUploadCoreReader{reader: strings.NewReader("skip-current-data")}
	if _, err := reader.reader.(*strings.Reader).Seek(5, 0); err != nil {
		t.Fatal(err)
	}
	calls := 0
	sourceHeaders := map[string]string{"Content-Type": "application/source", "Accept": "application/source", "X-Policy": "source"}
	var client *gophercloud.ServiceClient
	client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.RawQuery != "" || req.Header.Get("X-Policy") != "option" {
			t.Fatal(req.URL, req.Header)
		}
		if calls == 1 {
			if req.Method != "POST" || req.URL.EscapedPath() != "/reverse/glance/v2/images" || req.Header.Get("Content-Type") != "application/json" || req.Header.Get("Accept") != "application/json" || req.Header.Get("X-OpenStack-Image-Size") != "" {
				t.Fatal(req.Method, req.URL, req.Header)
			}
			body := taskCorePayload(t, req)
			if len(body) != 4 || taskCoreText(t, body["name"]) != "literal\nname" || taskCoreText(t, body["disk_format"]) != "qcow2" || taskCoreText(t, body["container_format"]) != "bare" || taskCoreText(t, body["visibility"]) != "private" {
				t.Fatal(body)
			}
			client.ProviderClient.SetToken("fresh-token")
			client.MoreHeaders["X-Policy"] = "later-source"
			response := taskCoreJSON(req, 201, metadata)
			response.Header.Set("Location", "https://foreign/other")
			return response, nil
		}
		if calls != 2 || req.Method != "PUT" || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(id)+"/file" || req.Header.Get("Content-Type") != "application/octet-stream" || req.Header.Get("Accept") != "" || req.Header.Get("X-OpenStack-Image-Size") != "0" || req.Header.Get("X-Auth-Token") != "fresh-token" || req.GetBody != nil || req.ContentLength != 0 {
			t.Fatal(req.Method, req.URL, req.Header, req.ContentLength)
		}
		if data := imageUploadCoreRead(t, req); data != "current-data" {
			t.Fatal(data)
		}
		return taskCoreJSON(req, 204, "opaque acknowledgement"), nil
	})
	client.MoreHeaders = sourceHeaders
	result, err := New(client).UploadImage(context.Background(), imageUploadCoreInput(reader), WithImageUploadHeader("x-policy", "option"), WithImageUploadSize(0))
	if err != nil || result == nil || result.ImageID != id || result.Image == nil || *result.Image.Status != "queued" || *result.Image.Size != 9007199254740993 || result.Image.Links != nil || result.Metadata.StatusCode != 201 || result.Acknowledgement.StatusCode != 204 || string(result.Metadata.Body) != metadata || string(result.Acknowledgement.Body) != "opaque acknowledgement" || calls != 2 || reader.closes.Load() != 0 || reader.seeks.Load() != 0 {
		t.Fatal(result, err, calls, reader)
	}
	result.Image.Properties["custom"][0] = '['
	result.Image.Header.Set("X-Task-Proof", "typed")
	result.Metadata.Header.Set("X-Task-Proof", "metadata")
	result.Metadata.Body[0] = '['
	if string(result.Image.Body["custom"]) != `{"precise":900719925474099312345}` || result.Acknowledgement.Header.Get("X-Task-Proof") != "actual" || sourceHeaders["Content-Type"] != "application/source" || sourceHeaders["Accept"] != "application/source" {
		t.Fatal("phase/DTO/source aliases", result)
	}
}

func TestImageUploadCoreCompletePreflight(t *testing.T) {
	calls := 0
	client := taskCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
	service := New(client)
	reader := &imageUploadCoreReader{reader: strings.NewReader("data")}
	var typedNil *imageUploadCoreReader
	for _, input := range []UploadImageRequest{{Name: "", Data: reader}, {Name: " \t", Data: reader}, {Name: string([]byte{0xff}), Data: reader}, {Name: "name"}, {Name: "name", Data: typedNil}} {
		callbacks := 0
		if value, err := service.UploadImage(context.Background(), input, func(*ImageUploadOpts) error { callbacks++; return nil }); value != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
			t.Fatal(value, err, callbacks)
		}
	}
	for _, s := range []*Service{nil, New(nil), New(&gophercloud.ServiceClient{})} {
		if value, err := s.UploadImage(context.Background(), imageUploadCoreInput(reader)); value != nil || err == nil {
			t.Fatal(value, err)
		}
	}
	if v, e := service.UploadImage(nil, imageUploadCoreInput(reader)); v != nil || !errors.Is(e, resource.ErrInvalidOption) {
		t.Fatal(v, e)
	}
	cause := errors.New("canceled preflight")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if v, e := service.UploadImage(ctx, imageUploadCoreInput(reader)); v != nil || !errors.Is(e, cause) || !errors.Is(e, context.Canceled) {
		t.Fatal(v, e)
	}
	invalid := []ImageUploadOption{nil, WithImageUploadSize(-1), WithImageUploadHeader("Content-Type", "owned"), WithImageUploadHeader("X-OpenStack-Image-Size", "1"), WithImageUploadField("name", "rogue"), WithImageUploadField(" x", 1), WithImageUploadFields(map[string]json.RawMessage{"x": nil}), WithImageUploadFields(map[string]json.RawMessage{"x": json.RawMessage{}}), WithImageUploadFields(map[string]json.RawMessage{"x": json.RawMessage("no-json")}), WithImageUploadFields(map[string]json.RawMessage{"x": json.RawMessage{'"', 0xff, '"'}})}
	for _, raw := range []string{"null", "false", "1", `[]`, `{}`, `""`} {
		invalid = append(invalid, WithImageUploadFields(map[string]json.RawMessage{"disk_format": json.RawMessage(raw)}), WithImageUploadFields(map[string]json.RawMessage{"container_format": json.RawMessage(raw)}))
	}
	for _, option := range invalid {
		if v, e := service.UploadImage(context.Background(), imageUploadCoreInput(reader), option); v != nil || !errors.Is(e, resource.ErrInvalidOption) {
			t.Fatal(v, e)
		}
	}
	client.MoreHeaders = map[string]string{"x-openstack-image-size": "1"}
	callbacks := 0
	if v, e := service.UploadImage(context.Background(), imageUploadCoreInput(reader), func(*ImageUploadOpts) error { callbacks++; return nil }); v != nil || !errors.Is(e, resource.ErrInvalidOption) || callbacks != 0 {
		t.Fatal(v, e, callbacks)
	}
	if calls != 0 || reader.reads.Load() != 0 || reader.closes.Load() != 0 || reader.seeks.Load() != 0 {
		t.Fatal(calls, reader)
	}
}

func TestImageUploadCoreCanonicalMetadataAndChosenID(t *testing.T) {
	for _, body := range []string{"", `[]`, `null`, `{"id":42}`, `{"id":"id","protected":"false"}`, `{"id":"id","tags":[null]}`, `{"id":"id","size":1.5}`, string([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'})} {
		reader := &imageUploadCoreReader{reader: strings.NewReader("unread")}
		calls := 0
		value, err := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, 201, body), nil })).UploadImage(context.Background(), imageUploadCoreInput(reader))
		if value == nil || value.Metadata == nil || value.Image != nil || value.Acknowledgement != nil || err == nil || calls != 1 || reader.reads.Load() != 0 {
			t.Fatal(body, value, err, calls)
		}
		taskCoreProof(t, err, 201, body)
	}
	for _, id := range []any{nil, "", ".", "..", "a/b", "a\\b", "a\n"} {
		body, _ := json.Marshal(map[string]any{"id": id, "status": "response-status", "name": "response-name"})
		reader := &imageUploadCoreReader{reader: strings.NewReader("unread")}
		value, err := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreJSON(req, 201, string(body)), nil })).UploadImage(context.Background(), imageUploadCoreInput(reader))
		if value == nil || value.Image == nil || *value.Image.Name != "response-name" || value.Acknowledgement != nil || !errors.Is(err, resource.ErrInvalidOption) || reader.reads.Load() != 0 {
			t.Fatal(id, value, err)
		}
		taskCoreProof(t, err, 201, string(body))
	}
	for _, id := range []string{"space :%?#", strings.Repeat("한", 300)} {
		calls := 0
		body, _ := json.Marshal(map[string]any{"id": id, "status": "server-owned"})
		value, err := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return taskCoreJSON(req, 201, string(body)), nil
			}
			if req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(id)+"/file" {
				t.Fatal(req.URL)
			}
			return taskCoreJSON(req, 204, ""), nil
		})).UploadImage(context.Background(), imageUploadCoreInput(strings.NewReader("data")))
		if value == nil || err != nil || value.ImageID != id || *value.Image.Status != "server-owned" || calls != 2 {
			t.Fatal(value, err, calls)
		}
	}
}

func TestImageUploadCoreAcceptedPhaseFailures(t *testing.T) {
	for _, phase := range []int{201, 204} {
		t.Run(fmt.Sprint(phase), func(t *testing.T) {
			readErr, closeErr, cause := errors.New("read"), errors.New("close"), errors.New("custom cancel")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			body := &taskCoreBody{reader: &taskCoreReader{body: "prefix", err: readErr, action: func() { cancel(cause) }}, closeErr: closeErr}
			calls := 0
			reader := &imageUploadCoreReader{reader: strings.NewReader("data")}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 && phase == 204 {
					return taskCoreJSON(req, 201, `{"id":"id","status":"queued"}`), nil
				}
				return taskCoreHTTP(req, phase, body), nil
			})
			value, err := New(client).UploadImage(ctx, imageUploadCoreInput(reader))
			if value == nil || !errors.Is(err, readErr) || !errors.Is(err, closeErr) || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || body.closes != 1 {
				t.Fatal(value, err, body.closes)
			}
			proof := taskCoreProof(t, err, phase, "prefix")
			if phase == 201 {
				if calls != 1 || value.Image != nil || value.Acknowledgement != nil || reader.reads.Load() != 0 {
					t.Fatal(value, calls)
				}
			} else if calls != 2 || value.Image == nil || value.ImageID != "id" || value.Acknowledgement == nil || string(value.Metadata.Body) != `{"id":"id","status":"queued"}` {
				t.Fatal(value, calls)
			}
			proof.Body[0] = 'X'
			proof.Header.Set("X-Task-Proof", "error")
			evidence := value.Metadata
			if phase == 204 {
				evidence = value.Acknowledgement
			}
			if string(evidence.Body) != "prefix" || evidence.Header.Get("X-Task-Proof") != "actual" {
				t.Fatal("error evidence aliases result", value)
			}
		})
	}
	for _, phase := range []int{201, 204} {
		t.Run("source"+fmt.Sprint(phase), func(t *testing.T) {
			calls := 0
			var client *gophercloud.ServiceClient
			raw := `{"id":"id"}`
			if phase == 204 {
				raw = "opaque"
			}
			body := &taskCoreBody{reader: &taskCoreReader{body: raw, err: io.EOF, action: func() { client.Microversion = "changed" }}}
			client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 && phase == 204 {
					return taskCoreJSON(req, 201, `{"id":"id"}`), nil
				}
				return taskCoreHTTP(req, phase, body), nil
			})
			value, err := New(client).UploadImage(context.Background(), imageUploadCoreInput(strings.NewReader("data")))
			if value == nil || !errors.Is(err, resource.ErrInvalidOption) || body.closes != 1 || calls != (phase-201)/3+1 {
				t.Fatal(value, err, calls)
			}
			taskCoreProof(t, err, phase, raw)
		})
	}
}

func TestImageUploadCoreNativePolicyAndNoReplay(t *testing.T) {
	for _, code := range []int{200, 202, 206, 401, 403, 404, 409, 410, 413, 415, 429, 498, 503, 301, 302, 303, 307, 308} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls, retries, reauth, backoff, redirects := 0, 0, 0, 0, 0
			reader := &imageUploadCoreReader{reader: strings.NewReader("prefix-tail")}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 201, `{"id":"id"}`), nil
				}
				prefix := make([]byte, 6)
				if _, err := io.ReadFull(req.Body, prefix); err != nil || string(prefix) != "prefix" {
					t.Fatal(string(prefix), err)
				}
				response := taskCoreJSON(req, code, "native-error")
				response.Header.Set("Location", "https://foreign/other")
				return response, nil
			})
			client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			client.ProviderClient.ReauthFunc = func(context.Context) error { reauth++; return nil }
			client.ProviderClient.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				backoff++
				return nil
			}
			client.ProviderClient.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects++; return nil }
			value, err := New(client).UploadImage(context.Background(), imageUploadCoreInput(reader))
			var native gophercloud.ErrUnexpectedResponseCode
			if value == nil || value.ImageID != "id" || value.Acknowledgement != nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != "native-error" || calls != 2 || retries != 0 || reauth != 0 || backoff != 0 || redirects != 0 || reader.closes.Load() != 0 || reader.seeks.Load() != 0 {
				t.Fatal(value, err, native, calls, retries, reauth, backoff, redirects)
			}
			if remaining := reader.reader.(*strings.Reader).Len(); remaining != 5 {
				t.Fatal("reader replayed", remaining)
			}
		})
	}
	t.Run("metadata prebody only", func(t *testing.T) {
		calls, retries := 0, 0
		var first string
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls <= 2 {
				raw := imageUploadCoreRead(t, req)
				if calls == 1 {
					first = raw
					return taskCoreJSON(req, 503, "retry"), nil
				}
				if raw != first {
					t.Fatal(raw, first)
				}
				return taskCoreJSON(req, 201, `{"id":"id"}`), nil
			}
			return taskCoreJSON(req, 204, ""), nil
		})
		client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
			retries++
			opts.JSONBody = copyImagePatchRaw(opts.JSONBody.(json.RawMessage))
			return nil
		}
		value, err := New(client).UploadImage(context.Background(), imageUploadCoreInput(strings.NewReader("data")))
		if value == nil || err != nil || calls != 3 || retries != 1 {
			t.Fatal(value, err, calls, retries)
		}
	})
	for _, code := range []int{200, 202, 204, 400} {
		t.Run("metadata"+fmt.Sprint(code), func(t *testing.T) {
			reader := &imageUploadCoreReader{reader: strings.NewReader("unread")}
			calls := 0
			value, err := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return taskCoreJSON(req, code, "native"), nil
			})).UploadImage(context.Background(), imageUploadCoreInput(reader))
			if value != nil || !gophercloud.ResponseCodeIs(err, code) || calls != 1 || reader.reads.Load() != 0 {
				t.Fatal(value, err, calls)
			}
		})
	}
}

type imageUploadCoreBlocked struct {
	started, release chan struct{}
	once             sync.Once
	closes           atomic.Int64
}

func (r *imageUploadCoreBlocked) Read([]byte) (int, error) {
	r.once.Do(func() { close(r.started) })
	<-r.release
	return 0, io.EOF
}
func (r *imageUploadCoreBlocked) Close() error { r.closes.Add(1); return nil }
func TestImageUploadCoreCancellationAndBorrowedFailures(t *testing.T) {
	t.Run("reader cause", func(t *testing.T) {
		cause := errors.New("borrowed read")
		reader := &imageUploadCoreReader{reader: &taskCoreReader{body: "prefix", err: cause}}
		calls := 0
		service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return taskCoreJSON(req, 201, `{"id":"id"}`), nil
			}
			_, err := io.ReadAll(req.Body)
			return nil, err
		}))
		value, err := service.UploadImage(context.Background(), imageUploadCoreInput(reader))
		if value == nil || value.ImageID != "id" || value.Acknowledgement != nil || !errors.Is(err, cause) || reader.closes.Load() != 0 || reader.seeks.Load() != 0 || calls != 2 {
			t.Fatal(value, err, reader)
		}
	})
	t.Run("caller releases", func(t *testing.T) {
		reader := &imageUploadCoreBlocked{started: make(chan struct{}), release: make(chan struct{})}
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		cause := errors.New("cancel upload")
		calls := 0
		service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return taskCoreJSON(req, 201, `{"id":"id"}`), nil
			}
			_, _ = io.Copy(io.Discard, req.Body)
			return nil, req.Context().Err()
		}))
		type outcome struct {
			value *ImageUploadResult
			err   error
		}
		done := make(chan outcome, 1)
		go func() {
			value, err := service.UploadImage(ctx, imageUploadCoreInput(reader))
			done <- outcome{value, err}
		}()
		select {
		case <-reader.started:
		case <-time.After(time.Second):
			t.Fatal("reader not started")
		}
		cancel(cause)
		if reader.closes.Load() != 0 {
			t.Fatal("SDK closed borrowed reader")
		}
		close(reader.release)
		select {
		case got := <-done:
			if got.value == nil || got.value.ImageID != "id" || !errors.Is(got.err, cause) || !errors.Is(got.err, context.Canceled) || reader.closes.Load() != 0 || calls != 2 {
				t.Fatal(got, calls)
			}
		case <-time.After(time.Second):
			t.Fatal("released operation did not finish")
		}
	})
	t.Run("HTTP timeout retained", func(t *testing.T) {
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			if req.Method == "POST" {
				return taskCoreJSON(req, 201, `{"id":"id"}`), nil
			}
			<-req.Context().Done()
			return nil, req.Context().Err()
		})
		client.ProviderClient.HTTPClient.Timeout = 20 * time.Millisecond
		value, err := New(client).UploadImage(context.Background(), imageUploadCoreInput(strings.NewReader("data")))
		if value == nil || !errors.Is(err, context.DeadlineExceeded) || client.ProviderClient.HTTPClient.Timeout != 20*time.Millisecond {
			t.Fatal(value, err)
		}
	})
}
