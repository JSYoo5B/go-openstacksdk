package image_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/image"
	nativeData "github.com/JSYoo5B/gophercloudsdk/image/v2/imagedata"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type uploadImageContractReader struct {
	io.Reader
	reads, closes, seeks, lengths atomic.Int32
}

func (r *uploadImageContractReader) Read(p []byte) (int, error) {
	r.reads.Add(1)
	return r.Reader.Read(p)
}
func (r *uploadImageContractReader) Close() error { r.closes.Add(1); return nil }
func (r *uploadImageContractReader) Seek(int64, int) (int64, error) {
	r.seeks.Add(1)
	return 0, errors.New("caller Seek forbidden")
}
func (r *uploadImageContractReader) Len() int { r.lengths.Add(1); return 999 }
func uploadImageContractInput(r io.Reader) image.UploadImageRequest {
	return image.UploadImageRequest{Name: " literal\nname ", Data: r}
}
func uploadImageContractResponse(code int, raw string) *http.Response {
	return taskContractsWire(code, &taskContractsBody{Reader: strings.NewReader(raw)})
}
func uploadImageContractRead(t *testing.T, r *http.Request) []byte {
	t.Helper()
	raw, e := io.ReadAll(r.Body)
	if e != nil {
		t.Fatal(e)
	}
	return raw
}
func uploadImageContractMetadata(t *testing.T, result *image.ImageUploadResult, raw string) {
	t.Helper()
	if result == nil || result.Metadata == nil || result.Metadata.StatusCode != 201 || string(result.Metadata.Body) != raw || result.Metadata.Header.Get("X-Request-Id") != "actual-task" {
		t.Fatalf("metadata evidence: %+v", result)
	}
}
func uploadImageContractBorrowed(t *testing.T, r *uploadImageContractReader) {
	t.Helper()
	if r.closes.Load() != 0 || r.seeks.Load() != 0 || r.lengths.Load() != 0 {
		t.Fatal("caller reader ownership", r.closes.Load(), r.seeks.Load(), r.lengths.Load())
	}
}

func TestImageUploadContractsWireAndOwnership(t *testing.T) {
	t.Run("defaults and literal selected route with independent evidence", func(t *testing.T) {
		id := "snow 雪 %2F :?#"
		raw := strings.Replace(imageContractObject, `"response-id"`, fmt.Sprintf("%q", id), 1)
		ack := string([]byte{'o', 0, 255, 'k'})
		reader := &uploadImageContractReader{Reader: strings.NewReader("binary\x00payload")}
		var calls atomic.Int32
		var wireMeta, wireAck *http.Response
		c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			n := calls.Add(1)
			if r.URL.RawQuery != "" || r.GetBody != nil && n == 2 {
				t.Error(r.URL, r.GetBody != nil)
			}
			if n == 1 {
				if reader.reads.Load() != 0 || r.Method != "POST" || r.URL.String() != taskContractsBase+"images" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" || r.Header.Get("X-OpenStack-Image-Size") != "" {
					t.Error(r.Method, r.URL, r.Header, reader.reads.Load())
				}
				taskContractsJSONEqual(t, uploadImageContractRead(t, r), map[string]any{"name": " literal\nname ", "disk_format": "qcow2", "container_format": "bare", "visibility": "private"})
				wireMeta = uploadImageContractResponse(201, raw)
				wireMeta.Header.Set("Location", "https://foreign.invalid/other")
				return wireMeta, nil
			}
			if n != 2 || r.Method != "PUT" || r.URL.String() != taskContractsBase+"images/"+url.PathEscape(id)+"/file" || r.URL.EscapedPath() != taskContractsPrefix+"images/"+url.PathEscape(id)+"/file" || r.Header.Get("Content-Type") != "application/octet-stream" || r.Header.Get("Accept") != "" || r.Header.Get("X-OpenStack-Image-Size") != "" {
				t.Error(n, r.Method, r.URL, r.Header)
			}
			if string(uploadImageContractRead(t, r)) != "binary\x00payload" {
				t.Error("binary payload changed")
			}
			_ = r.Body.Close()
			wireAck = uploadImageContractResponse(204, ack)
			wireAck.Header.Set("X-Phase", "binary")
			return wireAck, nil
		})
		c.Microversion = "2.0"
		c.MoreHeaders = map[string]string{"Accept": "source/accept", "Content-Type": "source/type"}
		provider := c.ProviderClient
		result, e := image.New(c).UploadImage(context.Background(), uploadImageContractInput(reader))
		uploadImageContractMetadata(t, result, raw)
		if e != nil || result.ImageID != id || result.Image == nil || result.Image.ID == nil || *result.Image.ID != id || result.Image.Status == nil || *result.Image.Status != "future/status" || result.Image.Size == nil || *result.Image.Size != 9007199254740993 || result.Image.Links != nil || result.Image.CreatedAt == nil || *result.Image.CreatedAt != "literal-created" || result.Acknowledgement == nil || result.Acknowledgement.StatusCode != 204 || string(result.Acknowledgement.Body) != ack || result.Acknowledgement.Header.Get("X-Phase") != "binary" || calls.Load() != 2 || c.ProviderClient != provider || c.MoreHeaders["Accept"] != "source/accept" || c.MoreHeaders["Content-Type"] != "source/type" {
			t.Fatal(result, e, calls.Load(), c.MoreHeaders)
		}
		if string(result.Image.Properties["x-number"]) != "9007199254740995" || string(result.Image.Body["ID"]) != "42" || result.Image.StatusCode != 201 {
			t.Fatal(result.Image)
		}
		wireMeta.Header.Set("X-Request-Id", "changed-wire")
		wireAck.Header.Set("X-Phase", "changed-wire")
		if result.Metadata.Header.Get("X-Request-Id") != "actual-task" || result.Image.Header.Get("X-Request-Id") != "actual-task" || result.Acknowledgement.Header.Get("X-Phase") != "binary" {
			t.Fatal("header alias")
		}
		result.Metadata.Body[0] = 'x'
		result.Metadata.Header.Set("X-Request-Id", "changed-metadata")
		result.Image.Properties["x-number"][0] = '8'
		if string(result.Image.Body["x-number"]) != "9007199254740995" || result.Image.Header.Get("X-Request-Id") != "actual-task" || result.Acknowledgement.Header.Get("X-Request-Id") != "actual-task" || string(result.Acknowledgement.Body) != ack {
			t.Fatal("phase/model evidence aliased")
		}
		uploadImageContractBorrowed(t, reader)
	})
	t.Run("all field helpers forward exact server owned values and size zero", func(t *testing.T) {
		var calls atomic.Int32
		reader := &uploadImageContractReader{Reader: strings.NewReader("bytes")}
		fields := map[string]json.RawMessage{"extra": json.RawMessage(`9007199254740993`), "id": json.RawMessage(`"requested-id"`), "nullable": json.RawMessage(`null`)}
		c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			if calls.Add(1) == 1 {
				taskContractsJSONEqual(t, uploadImageContractRead(t, r), map[string]any{"name": " literal\nname ", "disk_format": "future/disk", "container_format": "future/container", "visibility": "future/visibility", "owner": "", "protected": false, "os_hidden": false, "min_disk": -1, "min_ram": 0, "tags": []string{"repeat", "repeat", "line\n", ""}, "extra": json.Number("9007199254740993"), "id": "requested-id", "nullable": nil, "custom": map[string]any{"flag": false}})
				if r.Header.Get("X-OpenStack-Image-Size") != "" || r.Header.Get("X-One") != "one" || r.Header.Get("X-Two") != "two" {
					t.Error(r.Header)
				}
				return uploadImageContractResponse(201, `{"id":"observed","status":"queued"}`), nil
			}
			if r.URL.String() != taskContractsBase+"images/observed/file" || r.Header.Get("X-OpenStack-Image-Size") != "0" || r.Header.Get("X-One") != "one" || r.Header.Get("X-Two") != "two" {
				t.Error(r.URL, r.Header)
			}
			_ = uploadImageContractRead(t, r)
			return uploadImageContractResponse(204, ""), nil
		})
		result, e := image.New(c).UploadImage(context.Background(), uploadImageContractInput(reader),
			image.WithImageUploadHeader("X-One", "one"), image.WithImageUploadHeaders(map[string]string{"X-Two": "two"}),
			image.WithImageUploadFields(fields), image.WithImageUploadDiskFormat("future/disk"), image.WithImageUploadContainerFormat("future/container"),
			image.WithImageUploadVisibility(image.Visibility("future/visibility")), image.WithImageUploadOwner(""), image.WithImageUploadProtected(false), image.WithImageUploadHidden(false),
			image.WithImageUploadMinDisk(-1), image.WithImageUploadMinRAM(0), image.WithImageUploadTags("repeat", "repeat", "line\n", ""), image.WithImageUploadField("custom", map[string]any{"flag": false}), image.WithImageUploadSize(0))
		if e != nil || result == nil || result.ImageID != "observed" || result.Image.Name != nil || result.Image.DiskFormat != nil || result.Image.Status == nil || *result.Image.Status != "queued" || calls.Load() != 2 {
			t.Fatal(result, e, calls.Load())
		}
		uploadImageContractBorrowed(t, reader)
	})
	t.Run("size reset fields replacement and zero tags", func(t *testing.T) {
		for _, size := range []int64{0, 4} {
			for _, reset := range []bool{false, true} {
				t.Run(fmt.Sprintf("size=%d reset=%v", size, reset), func(t *testing.T) {
					var calls atomic.Int32
					c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
						if calls.Add(1) == 1 {
							body := taskContractsJSON(t, uploadImageContractRead(t, r))
							if _, ok := body["cleared"]; ok || string(body["tags"]) != "[]" {
								t.Error(body)
							}
							return uploadImageContractResponse(201, `{"id":"fixed"}`), nil
						}
						expected := fmt.Sprint(size)
						if reset {
							expected = ""
						}
						if r.Header.Get("X-OpenStack-Image-Size") != expected {
							t.Error(r.Header)
						}
						_ = uploadImageContractRead(t, r)
						return uploadImageContractResponse(204, ""), nil
					})
					opts := []image.ImageUploadOption{image.WithImageUploadField("cleared", true), image.WithImageUploadFields(nil), image.WithImageUploadTags(), image.WithImageUploadSize(size)}
					if reset {
						opts = append(opts, image.WithoutImageUploadSize())
					}
					result, e := image.New(c).UploadImage(context.Background(), uploadImageContractInput(strings.NewReader("data")), opts...)
					if e != nil || result == nil || calls.Load() != 2 {
						t.Fatal(result, e, calls.Load())
					}
				})
			}
		}
	})
	t.Run("old upload and native ABI remains available", func(t *testing.T) {
		var old func(context.Context, image.UploadImageRequest, ...image.UploadImageOption) (*image.Image, error) = image.New(nil).Upload
		var native func(context.Context, string, io.Reader) error = nativeData.New(nil).Upload
		if old == nil || native == nil || image.WithDiskFormat("raw") == nil || image.WithUploadSize(0) == nil || image.WithWait() == nil {
			t.Fatal("legacy ABI changed")
		}
	})
}

func TestImageUploadContractsPreflightAndSnapshots(t *testing.T) {
	t.Run("full replacement factories and retained callback storage", func(t *testing.T) {
		var calls, callbacks, marshals atomic.Int32
		size := int64(4)
		raw := json.RawMessage(`{"number":1}`)
		headers := map[string]string{"X-Snapshot": "owned"}
		fields := map[string]json.RawMessage{"snapshot": raw}
		full := image.WithImageUploadOpts(image.ImageUploadOpts{Headers: headers, Fields: fields, Size: &size})
		raw[10] = '9'
		headers["X-Snapshot"] = "mutated"
		fields["extra"] = json.RawMessage(`true`)
		size = 99
		one := image.WithImageUploadField("factory", taskContractsMarshaler{calls: &marshals, raw: []byte(`9007199254740995`)})
		var oldFields map[string]json.RawMessage
		var oldHeaders map[string]string
		var oldSize *int64
		c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			if calls.Add(1) == 1 {
				body := taskContractsJSON(t, uploadImageContractRead(t, r))
				if string(body["snapshot"]) != `{"number":1}` || string(body["factory"]) != "9007199254740995" || string(body["callback"]) != "1" || r.Header.Get("X-Snapshot") != "owned" || r.Header.Get("X-Callback") != "first" {
					t.Error(body, r.Header)
				}
				if _, ok := body["extra"]; ok {
					t.Error(body)
				}
				return uploadImageContractResponse(201, `{"id":"fixed"}`), nil
			}
			if r.Header.Get("X-OpenStack-Image-Size") != "4" || r.Header.Get("X-Snapshot") != "owned" || r.Header.Get("X-Callback") != "first" {
				t.Error(r.Header)
			}
			_ = uploadImageContractRead(t, r)
			return uploadImageContractResponse(204, ""), nil
		})
		result, e := image.New(c).UploadImage(context.Background(), uploadImageContractInput(strings.NewReader("data")), image.WithImageUploadHeader("X-Removed", "before"), full, one,
			func(o *image.ImageUploadOpts) error {
				callbacks.Add(1)
				if o.Headers == nil || o.Fields == nil {
					t.Error("uninitialized options")
				}
				o.Fields["callback"] = json.RawMessage(`1`)
				o.Headers["X-Callback"] = "first"
				oldFields = o.Fields
				oldHeaders = o.Headers
				oldSize = o.Size
				return nil
			},
			func(o *image.ImageUploadOpts) error {
				callbacks.Add(1)
				oldFields["callback"][0] = '9'
				oldHeaders["X-Callback"] = "retained mutation"
				*oldSize = 9
				if _, ok := o.Headers["X-Removed"]; ok {
					t.Error(o.Headers)
				}
				return nil
			})
		if e != nil || result == nil || calls.Load() != 2 || callbacks.Load() != 2 || marshals.Load() != 1 {
			t.Fatal(result, e, calls.Load(), callbacks.Load(), marshals.Load())
		}
	})
	t.Run("invalid input and source before callback HTTP or reader", func(t *testing.T) {
		for _, mode := range []string{"nil context", "cancelled context", "nil service", "nil client", "nil provider", "wrong type", "bad endpoint", "bad source media", "bad source size", "empty name", "whitespace name", "invalid name UTF8", "nil data", "typed nil data"} {
			t.Run(mode, func(t *testing.T) {
				var calls, callbacks atomic.Int32
				reader := &uploadImageContractReader{Reader: strings.NewReader("data")}
				c := taskContractsClient(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					return uploadImageContractResponse(201, `{"id":"fixed"}`), nil
				})
				s := image.New(c)
				ctx := context.Background()
				input := uploadImageContractInput(reader)
				cause := errors.New("preflight context cause")
				switch mode {
				case "nil context":
					ctx = nil
				case "cancelled context":
					var cancel context.CancelCauseFunc
					ctx, cancel = context.WithCancelCause(ctx)
					cancel(cause)
				case "nil service":
					s = nil
				case "nil client":
					s = image.New(nil)
				case "nil provider":
					c.ProviderClient = nil
				case "wrong type":
					c.Type = "compute"
				case "bad endpoint":
					c.Endpoint = "https://user:pass@bad.invalid/v2/"
				case "bad source media":
					c.MoreHeaders = map[string]string{"content-TYPE": "source/media\n"}
				case "bad source size":
					c.MoreHeaders = map[string]string{"x-openstack-image-size": "99"}
				case "empty name":
					input.Name = ""
				case "whitespace name":
					input.Name = " \t\n "
				case "invalid name UTF8":
					input.Name = string([]byte{255})
				case "nil data":
					input.Data = nil
				case "typed nil data":
					var v *uploadImageContractReader
					input.Data = v
				}
				result, e := s.UploadImage(ctx, input, func(*image.ImageUploadOpts) error { callbacks.Add(1); return nil })
				expected := resource.ErrInvalidOption
				if mode == "wrong type" {
					expected = resource.ErrUnsupported
				}
				if mode == "cancelled context" {
					expected = context.Canceled
				}
				if result != nil || !errors.Is(e, expected) || calls.Load() != 0 || callbacks.Load() != 0 || reader.reads.Load() != 0 || mode == "cancelled context" && !errors.Is(e, cause) {
					t.Fatal(result, e, calls.Load(), callbacks.Load(), reader.reads.Load())
				}
				uploadImageContractBorrowed(t, reader)
			})
		}
	})
	t.Run("final raw config and header validation before metadata", func(t *testing.T) {
		cases := []struct {
			name   string
			option image.ImageUploadOption
		}{
			{"nil option", nil}, {"negative size", image.WithImageUploadSize(-1)},
			{"name owned", image.WithImageUploadFields(map[string]json.RawMessage{"name": json.RawMessage(`"other"`)})},
			{"empty key", image.WithImageUploadFields(map[string]json.RawMessage{"": json.RawMessage(`1`)})},
			{"edge python whitespace", image.WithImageUploadFields(map[string]json.RawMessage{"\x1ckey": json.RawMessage(`1`)})},
			{"invalid key UTF8", image.WithImageUploadFields(map[string]json.RawMessage{string([]byte{255}): json.RawMessage(`1`)})},
			{"raw nil", image.WithImageUploadFields(map[string]json.RawMessage{"value": nil})},
			{"raw empty", image.WithImageUploadFields(map[string]json.RawMessage{"value": json.RawMessage{}})},
			{"raw malformed", image.WithImageUploadFields(map[string]json.RawMessage{"value": json.RawMessage(`{`)})},
			{"raw UTF8", image.WithImageUploadFields(map[string]json.RawMessage{"value": json.RawMessage{'"', 255, '"'}})},
			{"format empty", image.WithImageUploadDiskFormat("")}, {"format null", image.WithImageUploadFields(map[string]json.RawMessage{"disk_format": json.RawMessage(`null`)})},
			{"format numeric", image.WithImageUploadFields(map[string]json.RawMessage{"container_format": json.RawMessage(`1`)})},
			{"protected auth", image.WithImageUploadHeader("x-auth-token", "override")}, {"protected media", image.WithImageUploadHeader("Accept", "x")},
			{"protected size", image.WithImageUploadHeader("X-OpenStack-Image-Size", "1")}, {"protected length", image.WithImageUploadHeader("Content-Length", "1")},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var calls atomic.Int32
				reader := &uploadImageContractReader{Reader: strings.NewReader("data")}
				c := taskContractsClient(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					return uploadImageContractResponse(201, `{"id":"fixed"}`), nil
				})
				result, e := image.New(c).UploadImage(context.Background(), uploadImageContractInput(reader), tc.option)
				if result != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 || reader.reads.Load() != 0 {
					t.Fatal(result, e, calls.Load(), reader.reads.Load())
				}
				uploadImageContractBorrowed(t, reader)
			})
		}
	})
	t.Run("factory encoding and callback causes remain inspectable", func(t *testing.T) {
		for _, mode := range []string{"invalid original key", "custom encoding", "unsupported encoding", "callback"} {
			t.Run(mode, func(t *testing.T) {
				cause := errors.New("option cause")
				var calls, marshals atomic.Int32
				c := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, cause })
				var option image.ImageUploadOption
				switch mode {
				case "invalid original key":
					option = image.WithImageUploadField(" key", taskContractsMarshaler{calls: &marshals, raw: []byte(`1`)})
				case "custom encoding":
					option = image.WithImageUploadField("key", taskContractsMarshaler{calls: &marshals, cause: cause})
				case "unsupported encoding":
					option = image.WithImageUploadField("key", make(chan int))
				case "callback":
					option = func(*image.ImageUploadOpts) error { return cause }
				}
				result, e := image.New(c).UploadImage(context.Background(), uploadImageContractInput(strings.NewReader("data")), option)
				if result != nil || e == nil || calls.Load() != 0 {
					t.Fatal(result, e, calls.Load())
				}
				if mode == "invalid original key" && (marshals.Load() != 0 || !errors.Is(e, resource.ErrInvalidOption)) {
					t.Fatal(e, marshals.Load())
				}
				if mode == "custom encoding" {
					var encoding *json.MarshalerError
					if !errors.As(e, &encoding) || !errors.Is(e, cause) || marshals.Load() != 1 {
						t.Fatal(e, marshals.Load())
					}
				}
				if mode == "unsupported encoding" {
					var unsupported *json.UnsupportedTypeError
					if !errors.As(e, &unsupported) {
						t.Fatal(e)
					}
				}
				if mode == "callback" && !errors.Is(e, cause) {
					t.Fatal(e)
				}
			})
		}
	})
}

func TestImageUploadContractsPartialEvidence(t *testing.T) {
	t.Run("metadata accepted handling and model errors retain evidence", func(t *testing.T) {
		for _, mode := range []string{"read", "Close", "context", "joined", "invalid JSON", "invalid UTF8", "bad canonical bool", "bad canonical integer", "bad tags", "bad locations"} {
			t.Run(mode, func(t *testing.T) {
				readCause, closeCause, cancelCause := errors.New("metadata read"), errors.New("metadata Close"), errors.New("metadata context")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				raw := `{"id":"fixed","status":"queued"}`
				switch mode {
				case "invalid JSON":
					raw = `{`
				case "invalid UTF8":
					raw = string([]byte{'{', '"', 'x', '"', ':', '"', 255, '"', '}'})
				case "bad canonical bool":
					raw = `{"id":"fixed","protected":0}`
				case "bad canonical integer":
					raw = `{"id":"fixed","size":9223372036854775808}`
				case "bad tags":
					raw = `{"id":"fixed","tags":[null]}`
				case "bad locations":
					raw = `{"id":"fixed","locations":[null]}`
				}
				body := &taskContractsBody{Reader: strings.NewReader(raw)}
				if mode == "read" || mode == "joined" {
					body.Reader = taskContractsReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
				}
				if mode == "Close" || mode == "joined" {
					body.closeErr = closeCause
				}
				if mode == "context" || mode == "joined" {
					body.onClose = func() { cancel(cancelCause) }
				}
				var calls atomic.Int32
				reader := &uploadImageContractReader{Reader: strings.NewReader("data")}
				c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					_ = uploadImageContractRead(t, r)
					return taskContractsWire(201, body), nil
				})
				result, e := image.New(c).UploadImage(ctx, uploadImageContractInput(reader))
				uploadImageContractMetadata(t, result, raw)
				proof := taskContractsProof(t, e, 201, []byte(raw))
				if result.Image != nil || result.ImageID != "" || result.Acknowledgement != nil || calls.Load() != 1 || reader.reads.Load() != 0 || body.closes.Load() != 1 {
					t.Fatal(result, e, calls.Load(), reader.reads.Load(), body.closes.Load())
				}
				if (mode == "read" || mode == "joined") && !errors.Is(e, readCause) || (mode == "Close" || mode == "joined") && !errors.Is(e, closeCause) || (mode == "context" || mode == "joined") && (!errors.Is(e, context.Canceled) || !errors.Is(e, cancelCause)) {
					t.Fatal(e)
				}
				proof.Body[0] = 'x'
				proof.Header.Set("X-Request-Id", "mutated")
				if string(result.Metadata.Body) != raw || result.Metadata.Header.Get("X-Request-Id") != "actual-task" {
					t.Fatal("error/result evidence aliased")
				}
				uploadImageContractBorrowed(t, reader)
			})
		}
	})
	t.Run("missing unsafe or decoy id retains decoded created image without reading input", func(t *testing.T) {
		for _, id := range []string{"missing", "null", "", ".", "..", "a/b", "a\\b", "line\n"} {
			t.Run(fmt.Sprintf("ID=%q", id), func(t *testing.T) {
				raw := `{"status":"queued","ID":"typed-decoy","file":"https://foreign.invalid/data"}`
				if id != "missing" {
					field := fmt.Sprintf("%q", id)
					if id == "null" {
						field = "null"
					}
					raw = `{"id":` + field + `,"status":"queued","ID":"typed-decoy","file":"https://foreign.invalid/data"}`
				}
				var calls atomic.Int32
				reader := &uploadImageContractReader{Reader: strings.NewReader("data")}
				c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					_ = uploadImageContractRead(t, r)
					v := uploadImageContractResponse(201, raw)
					v.Header.Set("Location", taskContractsBase+"images/typed-decoy")
					return v, nil
				})
				result, e := image.New(c).UploadImage(context.Background(), uploadImageContractInput(reader))
				uploadImageContractMetadata(t, result, raw)
				taskContractsProof(t, e, 201, []byte(raw))
				if !errors.Is(e, resource.ErrInvalidOption) || result.Image == nil || result.Image.Status == nil || *result.Image.Status != "queued" || result.Acknowledgement != nil || calls.Load() != 1 || reader.reads.Load() != 0 {
					t.Fatal(result, e, calls.Load(), reader.reads.Load())
				}
				expected := id
				if id == "null" || id == "missing" {
					expected = ""
				}
				if result.ImageID != expected {
					t.Fatal(result.ImageID, expected)
				}
				uploadImageContractBorrowed(t, reader)
			})
		}
	})
	t.Run("binary accepted opaque body preserves independent joined failures", func(t *testing.T) {
		for _, mode := range []string{"read", "Close", "context", "joined"} {
			t.Run(mode, func(t *testing.T) {
				readCause, closeCause, cancelCause := errors.New("binary read"), errors.New("binary Close"), errors.New("binary context")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				raw := string([]byte{0, 255, 'x'})
				metadata := `{"id":"fixed","status":"queued"}`
				body := &taskContractsBody{Reader: strings.NewReader(raw)}
				if mode == "read" || mode == "joined" {
					body.Reader = taskContractsReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
				}
				if mode == "Close" || mode == "joined" {
					body.closeErr = closeCause
				}
				if mode == "context" || mode == "joined" {
					body.onClose = func() { cancel(cancelCause) }
				}
				var calls atomic.Int32
				reader := &uploadImageContractReader{Reader: strings.NewReader("data")}
				c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
					n := calls.Add(1)
					_ = uploadImageContractRead(t, r)
					if n == 1 {
						return uploadImageContractResponse(201, metadata), nil
					}
					return taskContractsWire(204, body), nil
				})
				result, e := image.New(c).UploadImage(ctx, uploadImageContractInput(reader))
				uploadImageContractMetadata(t, result, metadata)
				proof := taskContractsProof(t, e, 204, []byte(raw))
				if result.Image == nil || result.ImageID != "fixed" || result.Acknowledgement == nil || result.Acknowledgement.StatusCode != 204 || string(result.Acknowledgement.Body) != raw || calls.Load() != 2 || body.closes.Load() != 1 {
					t.Fatal(result, e, calls.Load(), body.closes.Load())
				}
				if (mode == "read" || mode == "joined") && !errors.Is(e, readCause) || (mode == "Close" || mode == "joined") && !errors.Is(e, closeCause) || (mode == "context" || mode == "joined") && (!errors.Is(e, context.Canceled) || !errors.Is(e, cancelCause)) {
					t.Fatal(e)
				}
				proof.Body[0] = 'y'
				proof.Header.Set("X-Request-Id", "changed")
				if string(result.Acknowledgement.Body) != raw || result.Acknowledgement.Header.Get("X-Request-Id") != "actual-task" || string(result.Metadata.Body) != metadata {
					t.Fatal("phase evidence alias")
				}
				uploadImageContractBorrowed(t, reader)
			})
		}
	})
	t.Run("binary HTTP transport and caller read failures preserve created result", func(t *testing.T) {
		for _, mode := range []string{"HTTP", "transport", "source Read"} {
			t.Run(mode, func(t *testing.T) {
				cause := errors.New("binary cause")
				var calls atomic.Int32
				reader := &uploadImageContractReader{Reader: strings.NewReader("data")}
				if mode == "source Read" {
					reader.Reader = taskContractsReader(func([]byte) (int, error) { return 0, cause })
				}
				c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
					if calls.Add(1) == 1 {
						_ = uploadImageContractRead(t, r)
						return uploadImageContractResponse(201, `{"id":"fixed"}`), nil
					}
					if mode == "source Read" {
						_, e := io.ReadAll(r.Body)
						return nil, e
					}
					if mode == "transport" {
						return nil, cause
					}
					_ = uploadImageContractRead(t, r)
					return uploadImageContractResponse(503, "binary rejected"), nil
				})
				result, e := image.New(c).UploadImage(context.Background(), uploadImageContractInput(reader))
				uploadImageContractMetadata(t, result, `{"id":"fixed"}`)
				if result.Image == nil || result.ImageID != "fixed" || result.Acknowledgement != nil || calls.Load() != 2 || e == nil {
					t.Fatal(result, e, calls.Load())
				}
				if mode == "HTTP" {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(e, &native) || native.Actual != 503 || !reflect.DeepEqual(native.Expected, []int{204}) || native.Method != "PUT" || native.URL != taskContractsBase+"images/fixed/file" || string(native.Body) != "binary rejected" {
						t.Fatal(e, native)
					}
				} else if !errors.Is(e, cause) {
					t.Fatal(e)
				}
				uploadImageContractBorrowed(t, reader)
			})
		}
	})
}

func TestImageUploadContractsSourceAndBinaryPolicy(t *testing.T) {
	t.Run("ordinary headers frozen and original authentication stays live", func(t *testing.T) {
		var calls atomic.Int32
		var c *gophercloud.ServiceClient
		reader := &uploadImageContractReader{Reader: strings.NewReader("data")}
		c = taskContractsClient(func(r *http.Request) (*http.Response, error) {
			n := calls.Add(1)
			_ = uploadImageContractRead(t, r)
			if r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Option") != "owned" {
				t.Error(r.Header)
			}
			if n == 1 {
				if r.Header.Get("X-Auth-Token") != "initial" {
					t.Error(r.Header)
				}
				b := &taskContractsBody{Reader: strings.NewReader(`{"id":"fixed"}`), onClose: func() { c.MoreHeaders["X-Source"] = "latest"; c.SetToken("latest-token") }}
				return taskContractsWire(201, b), nil
			}
			if r.Header.Get("X-Auth-Token") != "latest-token" {
				t.Error(r.Header)
			}
			return uploadImageContractResponse(204, ""), nil
		})
		c.MoreHeaders = map[string]string{"X-Source": "captured"}
		provider := c.ProviderClient
		result, e := image.New(c).UploadImage(context.Background(), uploadImageContractInput(reader), image.WithImageUploadHeader("X-Option", "owned"))
		if e != nil || result == nil || calls.Load() != 2 || c.ProviderClient != provider || c.MoreHeaders["X-Source"] != "latest" || len(c.MoreHeaders) != 1 {
			t.Fatal(result, e, c.MoreHeaders, calls.Load())
		}
		uploadImageContractBorrowed(t, reader)
	})
	t.Run("target source drift retains the accepted phase and stops further activity", func(t *testing.T) {
		for _, phase := range []string{"callback", "metadata", "binary"} {
			for _, field := range []string{"endpoint", "base", "provider", "type", "microversion"} {
				t.Run(phase+" "+field, func(t *testing.T) {
					var calls atomic.Int32
					reader := &uploadImageContractReader{Reader: strings.NewReader("data")}
					var c *gophercloud.ServiceClient
					mutate := func() {
						switch field {
						case "endpoint":
							c.Endpoint = "https://other.invalid/v2/"
						case "base":
							c.ResourceBase = "https://other.invalid/v2/"
						case "provider":
							c.ProviderClient = &gophercloud.ProviderClient{}
						case "type":
							c.Type = "compute"
						case "microversion":
							c.Microversion = "2.99"
						}
					}
					c = taskContractsClient(func(r *http.Request) (*http.Response, error) {
						n := calls.Add(1)
						_ = uploadImageContractRead(t, r)
						code, raw := 201, `{"id":"fixed"}`
						if n == 2 {
							code, raw = 204, "opaque"
						}
						b := &taskContractsBody{Reader: strings.NewReader(raw)}
						if phase == "metadata" && n == 1 || phase == "binary" && n == 2 {
							b.onClose = mutate
						}
						return taskContractsWire(code, b), nil
					})
					opts := []image.ImageUploadOption{}
					if phase == "callback" {
						opts = append(opts, func(*image.ImageUploadOpts) error { mutate(); return nil })
					}
					result, e := image.New(c).UploadImage(context.Background(), uploadImageContractInput(reader), opts...)
					if !errors.Is(e, resource.ErrInvalidOption) {
						t.Fatal(result, e)
					}
					if phase == "callback" {
						if result != nil || calls.Load() != 0 || reader.reads.Load() != 0 {
							t.Fatal(result, calls.Load(), reader.reads.Load())
						}
					} else {
						uploadImageContractMetadata(t, result, `{"id":"fixed"}`)
						code, raw := 201, `{"id":"fixed"}`
						count := int32(1)
						if phase == "binary" {
							code, raw, count = 204, "opaque", 2
							if result.Image == nil || result.Acknowledgement == nil {
								t.Fatal(result)
							}
						} else if result.Acknowledgement != nil || reader.reads.Load() != 0 {
							t.Fatal(result, reader.reads.Load())
						}
						taskContractsProof(t, e, code, []byte(raw))
						if calls.Load() != count {
							t.Fatal(calls.Load())
						}
					}
					uploadImageContractBorrowed(t, reader)
				})
			}
		}
	})
	t.Run("binary never reauthenticates retries redirects or replays prefix", func(t *testing.T) {
		for _, code := range []int{401, 429, 503, 303} {
			t.Run(fmt.Sprint(code), func(t *testing.T) {
				var calls, hooks, reauth, backoffs, redirects atomic.Int32
				reader := &uploadImageContractReader{Reader: strings.NewReader("prefix must not replay")}
				c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
					if calls.Add(1) == 1 {
						_ = uploadImageContractRead(t, r)
						return uploadImageContractResponse(201, `{"id":"fixed"}`), nil
					}
					p := make([]byte, 3)
					if n, e := r.Body.Read(p); n != 3 || e != nil || string(p) != "pre" {
						t.Error(n, e, string(p))
					}
					_ = r.Body.Close()
					v := uploadImageContractResponse(code, "binary failure")
					v.Header.Set("Location", taskContractsBase+"images/fixed/file")
					return v, nil
				})
				c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks.Add(1)
					return nil
				}
				c.ReauthFunc = func(context.Context) error { reauth.Add(1); return nil }
				c.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
					backoffs.Add(1)
					return nil
				}
				c.MaxBackoffRetries = 3
				c.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects.Add(1); return nil }
				provider := c.ProviderClient
				retryPtr := reflect.ValueOf(c.RetryFunc).Pointer()
				redirectPtr := reflect.ValueOf(c.HTTPClient.CheckRedirect).Pointer()
				result, e := image.New(c).UploadImage(context.Background(), uploadImageContractInput(reader))
				var native gophercloud.ErrUnexpectedResponseCode
				if result == nil || result.Image == nil || result.Acknowledgement != nil || !errors.As(e, &native) || native.Actual != code || calls.Load() != 2 || reader.reads.Load() != 1 || hooks.Load() != 0 || reauth.Load() != 0 || backoffs.Load() != 0 || redirects.Load() != 0 || c.ProviderClient != provider || reflect.ValueOf(c.RetryFunc).Pointer() != retryPtr || reflect.ValueOf(c.HTTPClient.CheckRedirect).Pointer() != redirectPtr || c.MaxBackoffRetries != 3 {
					t.Fatal(result, e, calls.Load(), reader.reads.Load(), hooks.Load(), reauth.Load(), backoffs.Load(), redirects.Load())
				}
				uploadImageContractBorrowed(t, reader)
			})
		}
	})
	t.Run("caller releases blocked reader and client timeout survives", func(t *testing.T) {
		for _, mode := range []string{"blocked reader", "timeout"} {
			t.Run(mode, func(t *testing.T) {
				var calls atomic.Int32
				cause := errors.New("caller cancellation")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				started, release := make(chan struct{}), make(chan struct{})
				reader := &uploadImageContractReader{Reader: taskContractsReader(func([]byte) (int, error) { close(started); <-release; return 0, io.EOF })}
				c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
					if calls.Add(1) == 1 {
						_ = uploadImageContractRead(t, r)
						return uploadImageContractResponse(201, `{"id":"fixed"}`), nil
					}
					if mode == "timeout" {
						close(started)
						<-r.Context().Done()
						return nil, r.Context().Err()
					}
					_, e := io.ReadAll(r.Body)
					if e != nil {
						return nil, e
					}
					return uploadImageContractResponse(204, ""), nil
				})
				if mode == "timeout" {
					c.HTTPClient.Timeout = 100 * time.Millisecond
				}
				type outcome struct {
					result *image.ImageUploadResult
					err    error
				}
				done := make(chan outcome, 1)
				go func() { v, e := image.New(c).UploadImage(ctx, uploadImageContractInput(reader)); done <- outcome{v, e} }()
				select {
				case <-started:
				case <-time.After(time.Second):
					t.Fatal("binary did not start")
				}
				if mode == "blocked reader" {
					cancel(cause)
					select {
					case v := <-done:
						t.Fatal("reader was silently released", v)
					default:
					}
					uploadImageContractBorrowed(t, reader)
					close(release)
				}
				select {
				case v := <-done:
					if v.result == nil || v.result.Image == nil || v.err == nil || calls.Load() != 2 {
						t.Fatal(v, calls.Load())
					}
					if mode == "timeout" {
						if !errors.Is(v.err, context.DeadlineExceeded) || c.HTTPClient.Timeout != 100*time.Millisecond {
							t.Fatal(v.err, c.HTTPClient.Timeout)
						}
					} else if !errors.Is(v.err, context.Canceled) || !errors.Is(v.err, cause) {
						t.Fatal(v.err)
					}
				case <-time.After(2 * time.Second):
					t.Fatal("binary did not finish after release")
				}
				uploadImageContractBorrowed(t, reader)
			})
		}
	})
}

func TestImageUploadContractsNativeMetadataPolicy(t *testing.T) {
	t.Run("metadata native prebody retries and reauthentication only", func(t *testing.T) {
		for _, mode := range []string{"503", "transport", "reauth", "same JSON"} {
			t.Run(mode, func(t *testing.T) {
				cause := errors.New("metadata transport")
				var post, put, hooks atomic.Int32
				var c *gophercloud.ServiceClient
				c = taskContractsClient(func(r *http.Request) (*http.Response, error) {
					if r.Method == "PUT" {
						put.Add(1)
						if r.Header.Get("X-Auth-Token") != "fresh" || r.Header.Get("Content-Type") != "application/octet-stream" || r.Header.Get("Accept") != "" {
							t.Error(r.Header)
						}
						_ = uploadImageContractRead(t, r)
						return uploadImageContractResponse(204, ""), nil
					}
					n := post.Add(1)
					body := taskContractsJSON(t, uploadImageContractRead(t, r))
					if string(body["number"]) != "1" || r.Method != "POST" || r.URL.String() != taskContractsBase+"images" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" {
						t.Error(r.Method, r.URL, r.Header, body)
					}
					if n == 1 {
						if mode == "transport" {
							return nil, cause
						}
						code := 503
						if mode == "reauth" {
							code = 401
						}
						return uploadImageContractResponse(code, "original failure"), nil
					}
					if r.Header.Get("X-Auth-Token") != "fresh" || mode != "reauth" && r.Header.Get("X-Hook") != "ordinary" {
						t.Error(r.Header)
					}
					return uploadImageContractResponse(201, `{"id":"fixed"}`), nil
				})
				if mode == "reauth" {
					c.ReauthFunc = func(context.Context) error { hooks.Add(1); c.SetToken("fresh"); return nil }
				} else {
					c.RetryFunc = func(_ context.Context, method, target string, o *gophercloud.RequestOpts, original error, _ uint) error {
						hooks.Add(1)
						if method != "POST" || target != taskContractsBase+"images" || o.RawBody != nil || o.JSONResponse != nil || !o.KeepResponseBody || mode == "transport" && !errors.Is(original, cause) || mode != "transport" && !gophercloud.ResponseCodeIs(original, 503) {
							t.Error(method, target, o, original)
						}
						if mode == "same JSON" {
							raw, e := json.Marshal(o.JSONBody)
							if e != nil {
								t.Fatal(e)
							}
							var body map[string]json.RawMessage
							if e = json.Unmarshal(raw, &body); e != nil {
								t.Fatal(e)
							}
							o.JSONBody = body
						}
						if o.MoreHeaders == nil {
							o.MoreHeaders = map[string]string{}
						}
						o.MoreHeaders["X-Hook"] = "ordinary"
						c.SetToken("fresh")
						return nil
					}
				}
				originalProvider := c.ProviderClient
				result, e := image.New(c).UploadImage(context.Background(), uploadImageContractInput(strings.NewReader("data")), image.WithImageUploadField("number", 1))
				if e != nil || result == nil || result.Acknowledgement == nil || post.Load() != 2 || put.Load() != 1 || hooks.Load() != 1 || c.ProviderClient != originalProvider {
					t.Fatal(result, e, post.Load(), put.Load(), hooks.Load())
				}
			})
		}
	})
	t.Run("metadata body ownership guard retains original and callback causes", func(t *testing.T) {
		for _, change := range []string{"changed JSON", "in-place JSON", "JSON null", "KeepResponseBody", "JSONResponse", "RawBody", "unsupported JSON"} {
			t.Run(change, func(t *testing.T) {
				cause := errors.New("metadata callback cause")
				var calls, hooks atomic.Int32
				rejected := &taskContractsBody{Reader: strings.NewReader("original503")}
				borrowed := &uploadImageContractReader{Reader: strings.NewReader("must not read")}
				c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					if r.Method != "POST" {
						t.Error(r.Method)
					}
					_ = uploadImageContractRead(t, r)
					return taskContractsWire(503, rejected), nil
				})
				c.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, original error, _ uint) error {
					hooks.Add(1)
					if !gophercloud.ResponseCodeIs(original, 503) {
						t.Error(original)
					}
					switch change {
					case "changed JSON":
						o.JSONBody = json.RawMessage(`{"different":1}`)
					case "in-place JSON":
						raw, ok := o.JSONBody.(json.RawMessage)
						if !ok {
							t.Fatalf("body %T", o.JSONBody)
						}
						needle := []byte(`"number":1`)
						at := bytes.Index(raw, needle)
						if at < 0 {
							t.Fatal(string(raw))
						}
						raw[at+len(needle)-1] = '2'
					case "JSON null":
						o.JSONBody = json.RawMessage(`null`)
					case "KeepResponseBody":
						o.KeepResponseBody = false
					case "JSONResponse":
						o.JSONResponse = new(any)
					case "RawBody":
						o.RawBody = borrowed
					case "unsupported JSON":
						o.JSONBody = make(chan int)
					}
					return cause
				}
				reader := &uploadImageContractReader{Reader: strings.NewReader("data")}
				originalRetry := reflect.ValueOf(c.RetryFunc).Pointer()
				result, e := image.New(c).UploadImage(context.Background(), uploadImageContractInput(reader), image.WithImageUploadField("number", 1))
				if result != nil || !errors.Is(e, resource.ErrInvalidOption) || !errors.Is(e, cause) || !gophercloud.ResponseCodeIs(e, 503) || calls.Load() != 1 || hooks.Load() != 1 || reader.reads.Load() != 0 || borrowed.reads.Load() != 0 || rejected.closes.Load() != 1 || reflect.ValueOf(c.RetryFunc).Pointer() != originalRetry {
					t.Fatal(result, e, calls.Load(), hooks.Load(), reader.reads.Load(), borrowed.reads.Load(), rejected.closes.Load())
				}
				if change == "unsupported JSON" {
					var unsupported *json.UnsupportedTypeError
					if !errors.As(e, &unsupported) {
						t.Fatal(e)
					}
				}
				uploadImageContractBorrowed(t, reader)
				uploadImageContractBorrowed(t, borrowed)
			})
		}
	})
	t.Run("expanded metadata codes own response then reject and binary only204", func(t *testing.T) {
		t.Run("expanded metadata200", func(t *testing.T) {
			readCause, closeCause, cancelCause := errors.New("expanded read"), errors.New("expanded Close"), errors.New("expanded context")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var calls, hooks atomic.Int32
			raw := `{"id":"false-created"}`
			accepted := &taskContractsBody{Reader: taskContractsReader(func(p []byte) (int, error) { return copy(p, raw), readCause }), closeErr: closeCause, onClose: func() { cancel(cancelCause) }}
			reader := &uploadImageContractReader{Reader: strings.NewReader("data")}
			c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
				_ = uploadImageContractRead(t, r)
				if calls.Add(1) == 1 {
					return uploadImageContractResponse(503, "original503"), nil
				}
				return taskContractsWire(200, accepted), nil
			})
			c.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				hooks.Add(1)
				o.OkCodes = []int{201, 200}
				return nil
			}
			result, e := image.New(c).UploadImage(ctx, uploadImageContractInput(reader))
			var native gophercloud.ErrUnexpectedResponseCode
			var owned *resource.ResponseError
			if result != nil || !errors.As(e, &native) || native.Actual != 200 || !reflect.DeepEqual(native.Expected, []int{201}) || native.Method != "POST" || native.URL != taskContractsBase+"images" || string(native.Body) != raw || native.ResponseHeader.Get("X-Request-Id") != "actual-task" || errors.As(e, &owned) || !errors.Is(e, readCause) || !errors.Is(e, closeCause) || !errors.Is(e, context.Canceled) || !errors.Is(e, cancelCause) || calls.Load() != 2 || hooks.Load() != 1 || reader.reads.Load() != 0 || accepted.closes.Load() != 1 {
				t.Fatal(result, e, native, calls.Load(), hooks.Load(), reader.reads.Load(), accepted.closes.Load())
			}
		})
		for _, phase := range []string{"POST", "PUT"} {
			for _, code := range []int{200, 201, 202, 204, 400, 403, 404, 409, 410, 413, 415, 503} {
				if phase == "POST" && code == 201 || phase == "PUT" && code == 204 {
					continue
				}
				t.Run(fmt.Sprintf("%s actual%d", phase, code), func(t *testing.T) {
					var calls atomic.Int32
					rejected := &taskContractsBody{Reader: strings.NewReader("native rejected")}
					reader := &uploadImageContractReader{Reader: strings.NewReader("data")}
					c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
						n := calls.Add(1)
						_ = uploadImageContractRead(t, r)
						if phase == "PUT" && n == 1 {
							return uploadImageContractResponse(201, `{"id":"fixed"}`), nil
						}
						return taskContractsWire(code, rejected), nil
					})
					result, e := image.New(c).UploadImage(context.Background(), uploadImageContractInput(reader))
					var native gophercloud.ErrUnexpectedResponseCode
					expected, target, count := 201, taskContractsBase+"images", int32(1)
					if phase == "PUT" {
						expected, target, count = 204, taskContractsBase+"images/fixed/file", 2
						if result == nil || result.Image == nil || result.Acknowledgement != nil {
							t.Fatal(result)
						}
					} else if result != nil || reader.reads.Load() != 0 {
						t.Fatal(result, reader.reads.Load())
					}
					if !errors.As(e, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{expected}) || native.Method != phase || native.URL != target || string(native.Body) != "native rejected" || calls.Load() != count || rejected.closes.Load() != 1 {
						t.Fatal(result, e, native, calls.Load(), rejected.closes.Load())
					}
					uploadImageContractBorrowed(t, reader)
				})
			}
		}
	})
	t.Run("advanced metadata media policy and fixed POST redirect boundary", func(t *testing.T) {
		t.Run("advanced retry media is native but binary media stays owned", func(t *testing.T) {
			var post, put, hooks atomic.Int32
			c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
				_ = uploadImageContractRead(t, r)
				if r.Method == "PUT" {
					put.Add(1)
					if r.Header.Get("Content-Type") != "application/octet-stream" || r.Header.Get("Accept") != "" || r.Header.Get("X-Native") != "" {
						t.Error(r.Header)
					}
					return uploadImageContractResponse(204, ""), nil
				}
				if post.Add(1) == 1 {
					return uploadImageContractResponse(503, "original503"), nil
				}
				if r.Header.Get("Content-Type") != "advanced/native" || r.Header.Get("Accept") != "advanced/accept" || r.Header.Get("X-Native") != "policy" {
					t.Error(r.Header)
				}
				return uploadImageContractResponse(201, `{"id":"fixed"}`), nil
			})
			c.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				hooks.Add(1)
				o.MoreHeaders = map[string]string{"Content-Type": "advanced/native", "Accept": "advanced/accept", "X-Native": "policy"}
				return nil
			}
			result, e := image.New(c).UploadImage(context.Background(), uploadImageContractInput(strings.NewReader("data")))
			if e != nil || result == nil || post.Load() != 2 || put.Load() != 1 || hooks.Load() != 1 || len(c.MoreHeaders) != 0 {
				t.Fatal(result, e, post.Load(), put.Load(), hooks.Load(), c.MoreHeaders)
			}
		})
		for _, mode := range []string{"same target", "foreign target", "query", "method"} {
			t.Run("POST redirect "+mode, func(t *testing.T) {
				var post, put, redirects atomic.Int32
				c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
					_ = uploadImageContractRead(t, r)
					if r.Method == "PUT" {
						put.Add(1)
						return uploadImageContractResponse(204, ""), nil
					}
					if post.Add(1) == 2 {
						return uploadImageContractResponse(201, `{"id":"fixed"}`), nil
					}
					v := uploadImageContractResponse(307, "redirect")
					target := taskContractsBase + "images"
					if mode == "foreign target" {
						target = "https://foreign.invalid/images"
					}
					if mode == "query" {
						target += "?retarget=1"
					}
					v.Header.Set("Location", target)
					return v, nil
				})
				c.HTTPClient.CheckRedirect = func(r *http.Request, _ []*http.Request) error {
					redirects.Add(1)
					if mode == "method" {
						r.Method = "GET"
					}
					return nil
				}
				result, e := image.New(c).UploadImage(context.Background(), uploadImageContractInput(strings.NewReader("data")))
				if mode == "same target" {
					if e != nil || result == nil || post.Load() != 2 || put.Load() != 1 {
						t.Fatal(result, e, post.Load(), put.Load())
					}
				} else if result != nil || !errors.Is(e, resource.ErrInvalidOption) || post.Load() != 1 || put.Load() != 0 {
					t.Fatal(result, e, post.Load(), put.Load())
				}
				if redirects.Load() != 1 {
					t.Fatal(redirects.Load())
				}
			})
		}
	})
}
