package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func imageRecordStageRead(t *testing.T, request *http.Request) string {
	t.Helper()
	if request.Body == nil {
		return ""
	}
	return imageUploadCoreRead(t, request)
}

func imageRecordStageFetched(t *testing.T, service *Service, handler *taskCoreTransport) *ImageRecord {
	t.Helper()
	return imageRecordUpdateFetched(t, service, `{"id":"fixed","status":"queued","name":"before","size":"04","protected":"false","vendor":{"precise":900719925474099312345}}`, handler)
}

// This reader adapter observes only the public size-inference IO contract.
// Transport, response failures and record constructors use the shared fixtures.
type imageRecordStageSeeker struct {
	reader  *strings.Reader
	whences []int
	offsets []int64
	failAt  int
	cause   error
	action  func(int)
}

func (r *imageRecordStageSeeker) Read(p []byte) (int, error) { return r.reader.Read(p) }
func (r *imageRecordStageSeeker) Seek(offset int64, whence int) (int64, error) {
	r.whences = append(r.whences, whence)
	r.offsets = append(r.offsets, offset)
	if r.action != nil {
		r.action(len(r.whences))
	}
	if len(r.whences) == r.failAt {
		return 0, r.cause
	}
	return r.reader.Seek(offset, whence)
}

func TestImageRecordStageFixedRoutesAndOpaqueAcceptedReceipts(t *testing.T) {
	for _, code := range []int{200, 204, 299, 300, 302, 304, 399} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			id := "name-like /한:%?\\b"
			rawID, _ := json.Marshal(id)
			calls, redirects, callbacks := 0, 0, 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			client.ProviderClient.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects++; return nil }
			service := New(client)
			seed := imageRecordStageFetched(t, service, &handler)
			seed.bodyState.current["id"], seed.bodyState.original["id"] = rawID, append(json.RawMessage(nil), rawID...)
			seed.Resource.Body["id"] = json.RawMessage(`"public decoy"`)
			seed.Wire.Body["id"] = json.RawMessage(`"wire decoy"`)
			original := cloneImageRecord(seed)
			client.ProviderClient.ReauthFunc = func(context.Context) error { callbacks++; return nil }
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				callbacks++
				return err
			}
			ackRaw := "opaque\xff"
			if code == 204 {
				ackRaw = ""
			}
			if code == 299 {
				ackRaw = `{"id":"ACK decoy","status":"queued","size":null}`
			}
			if code == 304 {
				ackRaw = "null"
			}
			ack := &taskCoreBody{reader: strings.NewReader(ackRaw)}
			var actualHeader http.Header
			handler = func(req *http.Request) (*http.Response, error) {
				if req.URL.RawQuery != "" || req.Header.Get("X-Ordinary") != "yes" {
					t.Fatal(req.URL, req.Header)
				}
				if calls == 2 {
					if req.Method != http.MethodPut || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(id)+"/stage" || req.Header.Get("Content-Type") != "application/octet-stream" || req.Header.Get("Accept") != "" || req.GetBody != nil || req.ContentLength != 0 {
						t.Fatal(req.Method, req.URL, req.Header, req.ContentLength)
					}
					if data := imageRecordStageRead(t, req); data != "current bytes" {
						t.Fatal(data)
					}
					_ = req.Body.Close()
					client.SetToken("fetch token")
					response := taskCoreHTTP(req, code, ack)
					response.Header.Set("Location", "https://foreign.test/image")
					response.Header.Set("OpenStack-image-import-methods", "stage-only")
					actualHeader = response.Header
					return response, nil
				}
				if calls != 3 || req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(id) || req.Body != nil || req.Header.Get("X-Auth-Token") != "fetch token" {
					t.Fatal(calls, req.Method, req.URL, req.Header)
				}
				response := taskCoreJSON(req, 299, `{"status":"uploading","name":"after"}`)
				response.Header.Set("OpenStack-image-import-methods", "glance-direct, web-download")
				return response, nil
			}
			reader := &imageUploadCoreReader{reader: strings.NewReader("current bytes")}
			got, err := service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: seed, Data: reader}, WithImageRecordStageSize(0), WithImageRecordStageHeader("X-Ordinary", "yes"))
			if got == nil || got.Record == nil || got.Staged == nil || got.Metadata == nil || err != nil || calls != 3 || redirects != 0 || callbacks != 0 || ack.closes != 1 || reader.closes.Load() != 0 || reader.seeks.Load() != 0 {
				t.Fatal(got, err, calls, redirects, callbacks, reader, ack.closes)
			}
			if got.Staged.StatusCode != code || string(got.Staged.Body) != ackRaw || got.Metadata.StatusCode != 299 || string(got.Metadata.Body) != `{"status":"uploading","name":"after"}` || got.Record.StatusCode != 299 || got.Record.Resource.StatusCode != 299 || len(got.Record.Resource.Body) != 65 || string(got.Record.Resource.Body["status"]) != `"uploading"` || string(got.Record.Resource.Body["size"]) != "4" || string(got.Record.Resource.Body["is_protected"]) != "true" || len(got.Record.bodyState.dirty) != 0 {
				t.Fatal("phase evidence or seeded translation changed", got)
			}
			th.CheckDeepEquals(t, []string{"glance-direct", " web-download"}, got.Record.ImportMethods)
			th.AssertEquals(t, id, taskCoreText(t, got.Record.Resource.Body["id"]))
			if !reflect.DeepEqual(seed, original) || got.Record == seed || got.Record.data != reader {
				t.Fatal("input mutation or borrowed retention", got, seed)
			}
			got.Staged.Header.Set("X-Task-Proof", "mutated")
			if len(got.Staged.Body) != 0 {
				got.Staged.Body[0] = '!'
			}
			got.Metadata.Body[0] = '!'
			got.Metadata.Header.Set("X-Task-Proof", "mutated")
			got.Record.Envelope[0] = '!'
			got.Record.Header.Set("X-Task-Proof", "record")
			got.Record.bodyState.current["name"][1] = '!'
			if actualHeader.Get("X-Task-Proof") != "actual" || string(got.Record.Wire.Body["name"]) != `"after"` || !reflect.DeepEqual(seed, original) {
				t.Fatal("receipt or raw channels alias", got)
			}
		})
	}
}

func TestImageRecordStageRequiresExactQueuedBeforeAnyDataIO(t *testing.T) {
	for _, raw := range []string{"", `null`, `"Queued"`, `"queued "`, `"active"`, `true`, `1`, `[]`, `{"queued":true}`} {
		t.Run("status="+raw, func(t *testing.T) {
			calls, options := 0, 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordStageFetched(t, service, &handler)
			if raw == "" {
				delete(seed.bodyState.current, "status")
			} else {
				seed.bodyState.current["status"] = json.RawMessage(raw)
			}
			seed.Resource.Body["status"] = json.RawMessage(`"queued"`)
			reader := &imageUploadCoreReader{reader: strings.NewReader("must not read")}
			handler = func(req *http.Request) (*http.Response, error) {
				t.Fatal("queued failure performed HTTP", req.URL)
				return nil, nil
			}
			got, err := service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: seed, Data: reader}, func(*ImageRecordStageOpts) error { options++; return nil })
			if got != nil || err == nil || calls != 1 || reader.reads.Load() != 0 || reader.seeks.Load() != 0 || reader.closes.Load() != 0 || options != 0 {
				t.Fatal(got, err, calls, options, reader)
			}
			// A missing filename must never mask the earlier queued failure.
			got, err = service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: seed, Filename: filepath.Join(t.TempDir(), "missing")})
			var pathError *os.PathError
			if got != nil || err == nil || errors.As(err, &pathError) || calls != 1 {
				t.Fatal("opened before queued check", got, err, calls)
			}
		})
	}
	t.Run("literal ID has no queued metadata", func(t *testing.T) {
		calls := 0
		reader := &imageUploadCoreReader{reader: strings.NewReader("unused")}
		client := taskCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
		got, err := New(client).StageImageRecord(context.Background(), ImageRecordStageRequest{ID: "fixed", Data: reader})
		if got != nil || err == nil || calls != 0 || reader.reads.Load() != 0 || reader.seeks.Load() != 0 {
			t.Fatal(got, err, calls, reader)
		}
	})
}

func TestImageRecordStageProjectsPrivateDescriptorsBeforeQueuedOrHTTP(t *testing.T) {
	for _, field := range []string{"size", "instance_type_rxtx_factor", "is_hw_vif_multiqueue_enabled"} {
		t.Run(field, func(t *testing.T) {
			calls := 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordStageFetched(t, service, &handler)
			seed.bodyState.current[field] = json.RawMessage(`"invalid descriptor"`)
			if field == "size" {
				seed.bodyState.current[field] = json.RawMessage(`1e9999`)
			}
			seed.bodyState.current["status"] = json.RawMessage(`"active"`)
			reader := &imageUploadCoreReader{reader: strings.NewReader("untouched")}
			got, err := service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: seed, Data: reader})
			if got != nil || err == nil || !strings.Contains(err.Error(), field) || calls != 1 || reader.reads.Load() != 0 || reader.seeks.Load() != 0 {
				t.Fatal(got, err, calls, reader)
			}
		})
	}
	t.Run("raw lone surrogate identity is never replacement-targeted", func(t *testing.T) {
		calls := 0
		var handler taskCoreTransport
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
		service := New(client)
		seed := imageRecordStageFetched(t, service, &handler)
		seed.bodyState.current["id"] = json.RawMessage(`"\ud800"`)
		got, err := service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: seed})
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
			t.Fatal(got, err, calls)
		}
	})
}

func TestImageRecordStageInvalidFetchPreservesPendingBodyForLaterCommit(t *testing.T) {
	for _, final := range []string{"", "not JSON"} {
		t.Run("fetch="+final, func(t *testing.T) {
			calls := 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			fetched := imageRecordStageFetched(t, service, &handler)
			handler = func(req *http.Request) (*http.Response, error) {
				imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"replace","path":"/name","value":"pending"}]`)
				return taskCoreJSON(req, 203, "invalid update JSON"), nil
			}
			pending, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: fetched, Attributes: map[string]any{"name": "pending"}})
			if err != nil || pending == nil {
				t.Fatal(pending, err)
			}
			before := cloneImageRecord(pending)
			handler = func(req *http.Request) (*http.Response, error) {
				if calls == 3 {
					if req.Method != http.MethodPut {
						t.Fatal(req.Method)
					}
					response := taskCoreJSON(req, 204, "stage receipt")
					response.Header.Set("OpenStack-image-import-methods", "stage")
					return response, nil
				}
				if calls != 4 || req.Method != http.MethodGet {
					t.Fatal(calls, req.Method)
				}
				response := taskCoreJSON(req, 203, final)
				response.Header.Set("OpenStack-image-import-methods", "fetch")
				return response, nil
			}
			got, err := service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: pending})
			if err != nil || got == nil || got.Record == nil || got.Record.Wire != nil || string(got.Record.Envelope) != final || !reflect.DeepEqual(got.Record.bodyState, before.bodyState) || !reflect.DeepEqual(pending, before) || len(got.Record.ImportMethods) != 1 || got.Record.ImportMethods[0] != "fetch" {
				t.Fatal("header-only stage or invalid fetch cleaned pending state", got, err)
			}
			handler = func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPatch {
					t.Fatal(req.Method)
				}
				imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"replace","path":"/name","value":"pending"}]`)
				return taskCoreJSON(req, 200, `{}`), nil
			}
			committed, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: got.Record})
			if err != nil || committed == nil || calls != 5 || len(committed.bodyState.dirty) != 0 {
				t.Fatal(committed, err, calls)
			}
		})
	}
}

func TestImageRecordStageValidFetchOverlaysSeedAndCleansRawBody(t *testing.T) {
	calls, locations := 0, 0
	cloud := "initial"
	var handler taskCoreTransport
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
	service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return resource.CloudLocation{Cloud: &cloud}, nil }})
	seed := imageRecordStageFetched(t, service, &handler)
	seed.bodyState.current["name"] = json.RawMessage(`"pending"`)
	seed.bodyState.dirty["name"] = struct{}{}
	seed.Resource.Body["size"] = json.RawMessage(`"public invalid"`)
	created := "passive created"
	seed.Resource.CreatedAt = &created
	before := cloneImageRecord(seed)
	cloud = "stage location"
	handler = func(req *http.Request) (*http.Response, error) {
		if calls == 2 {
			return taskCoreJSON(req, 204, "opaque"), nil
		}
		if calls != 3 || req.Method != http.MethodGet {
			t.Fatal(calls, req.Method)
		}
		return taskCoreJSON(req, 200, `{"status":"uploading","vendor":{"new":true}}`), nil
	}
	got, err := service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: seed})
	if err != nil || got == nil || got.Record == nil || calls != 3 || locations != 2 || !reflect.DeepEqual(before, seed) || len(got.Record.bodyState.dirty) != 0 || !reflect.DeepEqual(got.Record.bodyState.current, got.Record.bodyState.original) {
		t.Fatal(got, err, calls, locations)
	}
	th.AssertEquals(t, `"pending"`, string(got.Record.Resource.Body["name"]))
	th.AssertEquals(t, "4", string(got.Record.Resource.Body["size"]))
	th.AssertEquals(t, `{"new":true}`, string(imageRecordProperties(t, got.Record)["vendor"]))
	if len(got.Record.ImportMethods) != 0 {
		t.Fatal("stage/fetch headers did not reset ImportMethods", got.Record.ImportMethods)
	}
	var facts resource.CloudLocation
	if err := json.Unmarshal(got.Record.Resource.Body["location"], &facts); err != nil || facts.Cloud == nil || *facts.Cloud != "stage location" {
		t.Fatal(facts, err)
	}
	handler = func(req *http.Request) (*http.Response, error) {
		t.Fatal("clean final fetch retried pending update", req.URL)
		return nil, nil
	}
	unchanged, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: got.Record})
	if err != nil || unchanged == nil || calls != 3 {
		t.Fatal(unchanged, err, calls)
	}
}

func TestImageRecordStageNilDataAndBorrowedFallbackPreserveCurrentCursor(t *testing.T) {
	calls, put := 0, 0
	var handler taskCoreTransport
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
	service := New(client)
	seed := imageRecordStageFetched(t, service, &handler)
	reader := &imageUploadCoreReader{reader: strings.NewReader("firstrest")}
	handler = func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			return taskCoreJSON(req, 200, `{}`), nil
		}
		put++
		if put == 1 {
			p := make([]byte, 5)
			n, err := req.Body.Read(p)
			if n != 5 || err != nil || string(p) != "first" {
				t.Fatal(n, err, string(p))
			}
		} else if put == 2 {
			if imageRecordStageRead(t, req) != "rest" {
				t.Fatal("fallback reset cursor")
			}
		} else {
			if imageRecordStageRead(t, req) != "" {
				t.Fatal("empty nil-data PUT acquired body")
			}
		}
		if req.Body != nil {
			_ = req.Body.Close()
		}
		return taskCoreJSON(req, 204, "ack"), nil
	}
	first, err := service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: seed, Data: reader}, WithImageRecordStageSize(-1))
	if err != nil || first == nil || first.Record.data != reader || seed.data != nil {
		t.Fatal(first, err, seed.data)
	}
	second, err := service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: first.Record}, WithImageRecordStageSize(0))
	if err != nil || second == nil || second.Record.data != reader || reader.seeks.Load() != 0 || reader.closes.Load() != 0 {
		t.Fatal(second, err, reader)
	}
	empty, err := service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: seed})
	if err != nil || empty == nil || empty.Record.data != nil || calls != 7 || put != 3 {
		t.Fatal(empty, err, calls, put)
	}
}

func TestImageRecordStageSizePolicyUsesTotalLengthWithoutChangingCursor(t *testing.T) {
	for _, mode := range []string{"default total", "plain reader", "explicit zero", "explicit negative", "inference off", "without size restores inference", "explicit overrides disabled", "inference enabled again", "full options replace"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordStageFetched(t, service, &handler)
			raw := strings.NewReader("prefixDATA")
			_, _ = raw.Seek(6, io.SeekStart)
			seeker := &imageRecordStageSeeker{reader: raw}
			var data io.Reader = seeker
			var options []ImageRecordStageOption
			wantSize := "10"
			infer := true
			switch mode {
			case "plain reader":
				data = borrowedUploadReader{Reader: raw}
				wantSize = ""
				infer = false
			case "explicit zero":
				options = []ImageRecordStageOption{WithImageRecordStageSize(0)}
				wantSize = "0"
				infer = false
			case "explicit negative":
				options = []ImageRecordStageOption{WithImageRecordStageSize(-7)}
				wantSize = "-7"
				infer = false
			case "inference off":
				options = []ImageRecordStageOption{WithImageRecordStageSizeInference(false)}
				wantSize = ""
				infer = false
			case "without size restores inference":
				options = []ImageRecordStageOption{WithImageRecordStageSize(99), WithoutImageRecordStageSize()}
			case "explicit overrides disabled":
				options = []ImageRecordStageOption{WithImageRecordStageSizeInference(false), WithImageRecordStageSize(0)}
				wantSize = "0"
				infer = false
			case "inference enabled again":
				options = []ImageRecordStageOption{WithImageRecordStageSizeInference(false), WithImageRecordStageSizeInference(true)}
			case "full options replace":
				options = []ImageRecordStageOption{WithImageRecordStageSize(-9), WithImageRecordStageSizeInference(false), WithImageRecordStageOpts(ImageRecordStageOpts{})}
			}
			handler = func(req *http.Request) (*http.Response, error) {
				if calls == 2 {
					if req.Method != http.MethodPut || req.Header.Get("X-OpenStack-Image-Size") != wantSize || req.ContentLength != 0 || imageRecordStageRead(t, req) != "DATA" {
						t.Fatal(req.Method, req.Header, req.ContentLength)
					}
					return taskCoreJSON(req, 204, ""), nil
				}
				if calls != 3 || req.Header.Get("X-OpenStack-Image-Size") != "" {
					t.Fatal("binary size leaked into final fetch", calls, req.Header)
				}
				return taskCoreJSON(req, 200, `{}`), nil
			}
			got, err := service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: seed, Data: data}, options...)
			if got == nil || err != nil || calls != 3 {
				t.Fatal(got, err, calls)
			}
			if infer {
				th.CheckDeepEquals(t, []int{io.SeekCurrent, io.SeekEnd, io.SeekStart}, seeker.whences)
				th.CheckDeepEquals(t, []int64{0, 0, 6}, seeker.offsets)
			} else if len(seeker.whences) != 0 {
				t.Fatal("size policy sought reader", seeker.whences)
			}
		})
	}
}

func TestImageRecordStageSizeInferenceKeepsSeekFailuresAndESPIPEUnknown(t *testing.T) {
	for _, test := range []struct {
		name    string
		at      int
		cause   error
		allowed bool
	}{{"current error", 1, errors.New("current seek"), false}, {"end error", 2, errors.New("end seek"), false}, {"restore error", 3, errors.New("restore seek"), false}, {"current ESPIPE", 1, syscall.ESPIPE, true}, {"end ESPIPE", 2, syscall.ESPIPE, true}, {"restore ESPIPE", 3, syscall.ESPIPE, true}} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordStageFetched(t, service, &handler)
			seeker := &imageRecordStageSeeker{reader: strings.NewReader("data"), failAt: test.at, cause: test.cause}
			handler = func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodPut {
					if req.Header.Get("X-OpenStack-Image-Size") != "" {
						t.Fatal(req.Header)
					}
					return taskCoreJSON(req, 204, ""), nil
				}
				return taskCoreJSON(req, 200, `{}`), nil
			}
			got, err := service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: seed, Data: seeker})
			if test.allowed {
				if got == nil || err != nil || calls != 3 {
					t.Fatal(got, err, calls)
				}
			} else if got != nil || !errors.Is(err, test.cause) || calls != 1 {
				t.Fatal(got, err, calls)
			}
		})
	}
	for _, mode := range []string{"source drift during Seek", "cancel during Seek"} {
		t.Run(mode, func(t *testing.T) {
			marker := errors.New("seek cancellation")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			calls := 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordStageFetched(t, service, &handler)
			seeker := &imageRecordStageSeeker{reader: strings.NewReader("data"), action: func(n int) {
				if n == 1 {
					if strings.HasPrefix(mode, "source") {
						client.Endpoint = "https://foreign.test/"
					} else {
						cancel(marker)
					}
				}
			}}
			got, err := service.StageImageRecord(ctx, ImageRecordStageRequest{Record: seed, Data: seeker})
			if got != nil || err == nil || calls != 1 || len(seeker.whences) != 1 {
				t.Fatal(got, err, calls, seeker.whences)
			}
			if strings.HasPrefix(mode, "source") {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, marker) || !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordStageFilenameAndBorrowedFileHaveSeparateLifetimes(t *testing.T) {
	for _, mode := range []string{"owned success", "owned final fetch rejection", "owned PUT rejection", "borrowed file"} {
		t.Run(mode, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "image")
			if err := os.WriteFile(filename, []byte("prefixDATA"), 0600); err != nil {
				t.Fatal(err)
			}
			calls := 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordStageFetched(t, service, &handler)
			request := ImageRecordStageRequest{Record: seed, Filename: filename}
			var borrowed *os.File
			want := "prefixDATA"
			if mode == "borrowed file" {
				var err error
				borrowed, err = os.Open(filename)
				if err != nil {
					t.Fatal(err)
				}
				defer borrowed.Close()
				_, _ = borrowed.Seek(6, io.SeekStart)
				request.Filename = ""
				request.Data = borrowed
				want = "DATA"
			}
			handler = func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodPut {
					if req.Header.Get("X-OpenStack-Image-Size") != "10" || imageRecordStageRead(t, req) != want {
						t.Fatal(req.Header)
					}
					_ = req.Body.Close()
					if mode == "owned PUT rejection" {
						return taskCoreJSON(req, 403, "rejected stage"), nil
					}
					return taskCoreJSON(req, 204, "ack"), nil
				}
				if mode == "owned final fetch rejection" {
					return taskCoreJSON(req, 404, "rejected fetch"), nil
				}
				return taskCoreJSON(req, 200, `{}`), nil
			}
			got, err := service.StageImageRecord(context.Background(), request)
			if mode == "owned PUT rejection" {
				if got != nil || err == nil || calls != 2 {
					t.Fatal(got, err, calls)
				}
			} else {
				if got == nil || got.Record == nil || got.Staged == nil || calls != 3 {
					t.Fatal(got, err, calls)
				}
				if mode == "owned final fetch rejection" {
					if err == nil || got.Metadata != nil {
						t.Fatal(got, err)
					}
				} else if err != nil || got.Metadata == nil {
					t.Fatal(got, err)
				}
				if mode == "borrowed file" {
					if got.Record.data != borrowed {
						t.Fatal("borrowed file not retained")
					}
				} else if got.Record.data != nil {
					t.Fatal("owned closed file retained", got.Record.data)
				}
			}
			if borrowed != nil {
				if _, err := borrowed.Stat(); err != nil {
					t.Fatal("SDK closed borrowed file", err)
				}
				if cursor, err := borrowed.Seek(0, io.SeekCurrent); err != nil || cursor != 10 {
					t.Fatal(cursor, err)
				}
			}
			if seed.data != nil {
				t.Fatal("source seed acquired file", seed.data)
			}
		})
	}
	t.Run("queued missing filename preserves open cause without PUT", func(t *testing.T) {
		calls := 0
		var handler taskCoreTransport
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
		service := New(client)
		seed := imageRecordStageFetched(t, service, &handler)
		got, err := service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: seed, Filename: filepath.Join(t.TempDir(), "missing")})
		var cause *os.PathError
		if got != nil || !errors.As(err, &cause) || !errors.Is(err, os.ErrNotExist) || calls != 1 {
			t.Fatal(got, err, calls)
		}
	})
}

func TestImageRecordStageRejectsInvalidInputsAndOwnedHeadersBeforeIO(t *testing.T) {
	calls := 0
	var handler taskCoreTransport
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
	service := New(client)
	seed := imageRecordStageFetched(t, service, &handler)
	reader := &imageUploadCoreReader{reader: strings.NewReader("unused")}
	var typedNil *imageUploadCoreReader
	for _, test := range []struct {
		name    string
		request ImageRecordStageRequest
		option  ImageRecordStageOption
	}{
		{"missing selector", ImageRecordStageRequest{Data: reader}, nil},
		{"conflicting selector", ImageRecordStageRequest{ID: "fixed", Record: seed, Data: reader}, nil},
		{"handcrafted record", ImageRecordStageRequest{Record: &ImageRecord{Resource: seed.Resource.Clone()}, Data: reader}, nil},
		{"typed nil", ImageRecordStageRequest{Record: seed, Data: typedNil}, nil},
		{"filename conflicts with data", ImageRecordStageRequest{Record: seed, Data: reader, Filename: filepath.Join(t.TempDir(), "missing")}, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := service.StageImageRecord(context.Background(), test.request)
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || reader.reads.Load() != 0 || reader.seeks.Load() != 0 {
				t.Fatal(got, err, calls, reader)
			}
		})
	}
	for _, key := range []string{"Authorization", "X-Auth-Token", "Content-Type", "Accept", "Content-Length", "X-OpenStack-Image-Size", "OpenStack-API-Version"} {
		t.Run("owned header="+key, func(t *testing.T) {
			got, err := service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: seed, Data: reader}, WithImageRecordStageHeader(key, "bad"))
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || reader.reads.Load() != 0 || reader.seeks.Load() != 0 {
				t.Fatal(got, err, calls, reader)
			}
		})
	}
	t.Run("nil option", func(t *testing.T) {
		got, err := service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: seed, Data: reader}, nil)
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
			t.Fatal(got, err, calls)
		}
	})
	t.Run("nil and canceled context do not run options", func(t *testing.T) {
		callbacks := 0
		option := func(*ImageRecordStageOpts) error { callbacks++; return nil }
		got, err := service.StageImageRecord(nil, ImageRecordStageRequest{Record: seed, Data: reader}, option)
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
			t.Fatal(got, err, callbacks)
		}
		marker := errors.New("stage already canceled")
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(marker)
		got, err = service.StageImageRecord(ctx, ImageRecordStageRequest{Record: seed, Data: reader}, option)
		if got != nil || !errors.Is(err, marker) || !errors.Is(err, context.Canceled) || callbacks != 0 || calls != 1 {
			t.Fatal(got, err, callbacks, calls)
		}
	})
	t.Run("invalid source before callback", func(t *testing.T) {
		client.MoreHeaders = map[string]string{"X-Auth-Token": "forged"}
		callbacks := 0
		got, err := service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: seed, Data: reader}, func(*ImageRecordStageOpts) error { callbacks++; return nil })
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 || calls != 1 {
			t.Fatal(got, err, callbacks, calls)
		}
	})
}

func TestImageRecordStageCapturesRequestOptionsLocationAndOrdinaryHeadersOnce(t *testing.T) {
	calls, callbacks, locations := 0, 0, 0
	var handler taskCoreTransport
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
	service := New(client)
	seed := imageRecordStageFetched(t, service, &handler)
	data := &imageUploadCoreReader{reader: strings.NewReader("captured data")}
	request := ImageRecordStageRequest{Record: seed, Data: data}
	original := cloneImageRecord(seed)
	size := int64(0)
	headers := map[string]string{"x-note": "captured"}
	option := WithImageRecordStageOpts(ImageRecordStageOpts{Headers: headers, Size: &size})
	headers["x-note"] = "external changed"
	size = 9
	var retained *ImageRecordStageOpts
	options := []ImageRecordStageOption{option, func(config *ImageRecordStageOpts) error {
		callbacks++
		retained = config
		return WithImageRecordStageHeader("X-Added", "yes")(config)
	}}
	cloud := "captured location"
	client.MoreHeaders = map[string]string{"X-Source": "captured source", "Content-Type": "source metadata", "Accept": "source accept"}
	service.dependencies.CloudLocation = func() (resource.CloudLocation, error) {
		locations++
		seed.bodyState.current["id"] = json.RawMessage(`"later private identity"`)
		seed.bodyState.current["name"] = json.RawMessage(`"later private name"`)
		request.Data = strings.NewReader("later request")
		options[1] = func(*ImageRecordStageOpts) error { t.Fatal("options slice was not captured"); return nil }
		client.MoreHeaders["X-Source"] = "later source"
		client.SetToken("live token")
		return resource.CloudLocation{Cloud: &cloud}, nil
	}
	handler = func(req *http.Request) (*http.Response, error) {
		if req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed/stage" && req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed" {
			t.Fatal(req.URL)
		}
		if req.Header.Get("X-Note") != "captured" || req.Header.Get("X-Added") != "yes" || req.Header.Get("X-Source") != "captured source" || req.Header.Get("X-Auth-Token") != "live token" {
			t.Fatal("phase policy not frozen", req.Header)
		}
		if req.Method == http.MethodPut {
			if req.Header.Get("X-OpenStack-Image-Size") != "0" || imageRecordStageRead(t, req) != "captured data" {
				t.Fatal(req.Header)
			}
			retained.Headers["X-Note"] = "retained changed"
			*retained.Size = 12
			cloud = "later location"
			return taskCoreJSON(req, 204, "ack"), nil
		}
		if req.Header.Get("Content-Type") != "source metadata" || req.Header.Get("Accept") != "source accept" {
			t.Fatal("binary media leaked into metadata", req.Header)
		}
		return taskCoreJSON(req, 200, `{}`), nil
	}
	got, err := service.StageImageRecord(context.Background(), request, options...)
	if got == nil || got.Record == nil || err != nil || calls != 3 || callbacks != 1 || locations != 1 || data.seeks.Load() != 0 || string(got.Record.Resource.Body["name"]) != string(original.Resource.Body["name"]) {
		t.Fatal(got, err, calls, callbacks, locations, data)
	}
	var facts resource.CloudLocation
	if err := json.Unmarshal(got.Record.Resource.Body["location"], &facts); err != nil || facts.Cloud == nil || *facts.Cloud != "captured location" {
		t.Fatal(facts, err)
	}
}

func TestImageRecordStageOptionsKeepBulkMergeFullReplacementAndGuardOwnership(t *testing.T) {
	for _, mode := range []string{"bulk merge", "full replace", "cancel", "source", "binding", "outer", "callback cause"} {
		t.Run(mode, func(t *testing.T) {
			calls, callbacks, later := 0, 0, 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordStageFetched(t, service, &handler)
			marker := errors.New("stage option cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			outerBad := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerBad {
					return marker
				}
				return nil
			})
			options := []ImageRecordStageOption{WithImageRecordStageHeader("X-Discarded", "old")}
			if mode == "bulk merge" {
				options = append(options, WithImageRecordStageHeaders(map[string]string{"X-Final": "new"}))
			} else if mode == "full replace" {
				options = append(options, WithImageRecordStageOpts(ImageRecordStageOpts{Headers: map[string]string{"X-Final": "new"}}))
			} else {
				options = append(options, func(*ImageRecordStageOpts) error {
					callbacks++
					switch mode {
					case "cancel":
						cancel(marker)
					case "source":
						client.Endpoint = "https://foreign.test/"
					case "binding":
						service.API = nil
					case "outer":
						outerBad = true
					case "callback cause":
						return marker
					}
					return nil
				}, func(*ImageRecordStageOpts) error { later++; return nil })
			}
			handler = func(req *http.Request) (*http.Response, error) {
				want := ""
				if mode == "bulk merge" {
					want = "old"
				}
				if req.Header.Get("X-Discarded") != want || req.Header.Get("X-Final") != "new" {
					t.Fatal(req.Header)
				}
				if req.Method == http.MethodPut {
					return taskCoreJSON(req, 204, ""), nil
				}
				return taskCoreJSON(req, 200, `{}`), nil
			}
			got, err := service.StageImageRecord(ctx, ImageRecordStageRequest{Record: seed}, options...)
			if mode == "bulk merge" || mode == "full replace" {
				if got == nil || err != nil || calls != 3 {
					t.Fatal(got, err, calls)
				}
				return
			}
			if got != nil || err == nil || calls != 1 || callbacks != 1 || later != 0 {
				t.Fatal(got, err, calls, callbacks, later)
			}
			if mode == "source" || mode == "binding" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, marker) {
				t.Fatal("callback/guard cause lost", err)
			}
		})
	}
}

func TestImageRecordStageAcceptedPUTHandlingFailuresRetainOnlyActualStageReceipt(t *testing.T) {
	for _, mode := range []string{"read", "close", "cancel", "source drift restored on Close", "outer drift restored on Close"} {
		t.Run(mode, func(t *testing.T) {
			calls, retries := 0, 0
			marker := errors.New("accepted stage handling")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			outerBad := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerBad {
					return marker
				}
				return nil
			})
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordStageFetched(t, service, &handler)
			body := &taskCoreBody{reader: strings.NewReader("actual accepted stage")}
			action := func() {}
			switch mode {
			case "read":
				body.reader = &taskCoreReader{body: "actual accepted stage", err: marker}
			case "close":
				body.closeErr = marker
			case "cancel":
				action = func() { cancel(marker) }
			case "source drift restored on Close":
				action = func() { client.Endpoint = "https://foreign.test/" }
			case "outer drift restored on Close":
				action = func() { outerBad = true }
			}
			if mode != "read" {
				body.reader = &taskCoreReader{body: "actual accepted stage", err: io.EOF, action: action}
			}
			var selected io.ReadCloser = body
			if strings.Contains(mode, "restored") {
				selected = &imageRecordCloseBody{taskCoreBody: body, after: func() { client.Endpoint = "https://glance.example/reverse/glance/v2/"; outerBad = false }}
			}
			handler = func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPut {
					t.Fatal("accepted PUT failure fetched/replayed", req.Method)
				}
				return taskCoreHTTP(req, 299, selected), nil
			}
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			got, err := service.StageImageRecord(ctx, ImageRecordStageRequest{Record: seed})
			if got == nil || got.Record != nil || got.Staged == nil || got.Metadata != nil || got.Staged.StatusCode != 299 || string(got.Staged.Body) != "actual accepted stage" || err == nil || calls != 2 || retries != 0 || body.closes != 1 {
				t.Fatal(got, err, calls, retries, body.closes)
			}
			if strings.HasPrefix(mode, "source") {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, marker) {
				t.Fatal(err)
			}
			taskCoreProof(t, err, 299, "actual accepted stage")
		})
	}
}

func TestImageRecordStageFinalFetchErrorsKeepBothPhaseEvidenceAndPendingSeed(t *testing.T) {
	for _, mode := range []string{"native", "read", "close", "nonobject", "projection", "invalid UTF8", "cancel", "source"} {
		t.Run(mode, func(t *testing.T) {
			marker := errors.New("fetch handling")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			calls, retries := 0, 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordStageFetched(t, service, &handler)
			seed.bodyState.current["name"] = json.RawMessage(`"pending"`)
			seed.bodyState.dirty["name"] = struct{}{}
			before := cloneImageRecord(seed)
			raw := `{"status":"uploading"}`
			switch mode {
			case "native":
				raw = "rejected fetch"
			case "nonobject":
				raw = "null"
			case "projection":
				raw = `{"instance_type_rxtx_factor":"invalid descriptor"}`
			case "invalid UTF8":
				raw = "{\"vendor\":\"\xff\"}"
			}
			body := &taskCoreBody{reader: strings.NewReader(raw)}
			if mode == "read" {
				body.reader = &taskCoreReader{body: raw, err: marker}
			}
			if mode == "close" {
				body.closeErr = marker
			}
			if mode == "cancel" {
				body.reader = &taskCoreReader{body: raw, err: io.EOF, action: func() { cancel(marker) }}
			}
			if mode == "source" {
				body.reader = &taskCoreReader{body: raw, err: io.EOF, action: func() { client.Endpoint = "https://foreign.test/" }}
			}
			handler = func(req *http.Request) (*http.Response, error) {
				if calls == 2 {
					reply := taskCoreJSON(req, 204, "actual stage")
					reply.Header.Set("OpenStack-image-import-methods", "stage methods")
					return reply, nil
				}
				if calls != 3 || req.Method != http.MethodGet {
					t.Fatal(calls, req.Method)
				}
				code := 203
				if mode == "native" {
					code = 404
				}
				return taskCoreHTTP(req, code, body), nil
			}
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			got, err := service.StageImageRecord(ctx, ImageRecordStageRequest{Record: seed})
			if got == nil || got.Record == nil || got.Staged == nil || string(got.Staged.Body) != "actual stage" || err == nil || calls != 3 || body.closes != 1 || !reflect.DeepEqual(seed, before) || len(got.Record.bodyState.dirty) == 0 {
				t.Fatal(got, err, calls, retries, body.closes)
			}
			if mode == "native" {
				var native gophercloud.ErrUnexpectedResponseCode
				if got.Metadata != nil || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != raw || retries != 1 || got.Record.StatusCode != 204 || len(got.Record.ImportMethods) != 1 || got.Record.ImportMethods[0] != "stage methods" {
					t.Fatal(got, err, native, retries)
				}
			} else {
				if got.Metadata == nil || got.Metadata.StatusCode != 203 || string(got.Metadata.Body) != raw || retries != 0 {
					t.Fatal(got, err, retries)
				}
				taskCoreProof(t, err, 203, raw)
				if mode == "read" || mode == "close" || mode == "cancel" {
					if !errors.Is(err, marker) {
						t.Fatal(err)
					}
				}
				if mode == "source" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestImageRecordStageBinaryRejectionsNeverReplayReauthenticateOrBackoff(t *testing.T) {
	for _, code := range []int{400, 401, 404, 429, 500, 599} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls, retries, reauth, backoff := 0, 0, 0, 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordStageFetched(t, service, &handler)
			reader := &imageUploadCoreReader{reader: strings.NewReader("prefix-tail")}
			handler = func(req *http.Request) (*http.Response, error) {
				if calls != 2 || req.Method != http.MethodPut {
					t.Fatal("binary replay/followup", calls, req.Method)
				}
				p := make([]byte, 6)
				if _, err := io.ReadFull(req.Body, p); err != nil || string(p) != "prefix" {
					t.Fatal(string(p), err)
				}
				_ = req.Body.Close()
				return taskCoreJSON(req, code, "native stage failure"), nil
			}
			client.ProviderClient.ReauthFunc = func(context.Context) error { reauth++; return nil }
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			client.ProviderClient.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				backoff++
				return nil
			}
			got, err := service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: seed, Data: reader}, WithImageRecordStageSizeInference(false))
			var native gophercloud.ErrUnexpectedResponseCode
			if got != nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != "native stage failure" || native.ResponseHeader.Get("X-Task-Proof") != "actual" || calls != 2 || retries != 0 || reauth != 0 || backoff != 0 || reader.closes.Load() != 0 || reader.seeks.Load() != 0 || reader.reader.(*strings.Reader).Len() != 5 {
				t.Fatal(got, err, native, calls, retries, reauth, backoff, reader)
			}
			if client.RetryFunc == nil || client.ProviderClient.ReauthFunc == nil || client.ProviderClient.RetryBackoffFunc == nil {
				t.Fatal("binary policy modified caller provider")
			}
		})
	}
}

func TestImageRecordStageBorrowedReadErrorsAndTransportCausesStopFinalFetch(t *testing.T) {
	for _, mode := range []string{"reader error", "transport error", "cancel reader", "source reader"} {
		t.Run(mode, func(t *testing.T) {
			marker := errors.New("stage input/transport")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			calls := 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordStageFetched(t, service, &handler)
			input := &taskCoreReader{body: "partial", err: marker}
			if mode == "cancel reader" {
				input.err = io.EOF
				input.action = func() { cancel(marker) }
			}
			if mode == "source reader" {
				input.err = io.EOF
				input.action = func() { client.Endpoint = "https://foreign.test/" }
			}
			handler = func(req *http.Request) (*http.Response, error) {
				if calls != 2 || req.Method != http.MethodPut {
					t.Fatal("failed binary input replayed/fetched", calls, req.Method)
				}
				if mode == "transport error" {
					return nil, marker
				}
				_, err := io.ReadAll(req.Body)
				if mode == "reader error" {
					return nil, err
				}
				return taskCoreJSON(req, 204, "accepted after reader action"), nil
			}
			got, err := service.StageImageRecord(ctx, ImageRecordStageRequest{Record: seed, Data: input})
			if err == nil || calls != 2 {
				t.Fatal(got, err, calls)
			}
			if mode == "reader error" || mode == "transport error" {
				if got != nil || !errors.Is(err, marker) {
					t.Fatal(got, err)
				}
			} else {
				if got == nil || got.Record != nil || got.Staged == nil {
					t.Fatal(got, err)
				}
				if mode == "source reader" {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, marker) {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestImageRecordStageFinalFetchRetainsSafeNativeRetryFixedTargetAndLiveAuth(t *testing.T) {
	for _, mode := range []string{"successful ordinary header retry", "expanded OkCodes rejection", "changed request body", "source mutation", "hook cause"} {
		t.Run(mode, func(t *testing.T) {
			calls, retries := 0, 0
			marker := errors.New("metadata retry")
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordStageFetched(t, service, &handler)
			handler = func(req *http.Request) (*http.Response, error) {
				if calls == 2 {
					return taskCoreJSON(req, 204, "stage receipt"), nil
				}
				if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed" || req.URL.RawQuery != "" || req.Body != nil {
					t.Fatal("GET retry moved fixed scope/body", req.Method, req.URL, req.Body)
				}
				if calls == 3 {
					return taskCoreJSON(req, 503, "initial metadata rejection"), nil
				}
				if calls != 4 || req.Header.Get("X-Auth-Token") != "retry token" || req.Header.Get("X-Retry") != "ordinary" {
					t.Fatal(calls, req.Header)
				}
				if mode == "expanded OkCodes rejection" {
					return taskCoreJSON(req, 418, "actual metadata rejection"), nil
				}
				return taskCoreJSON(req, 203, `{"status":"uploading"}`), nil
			}
			client.RetryFunc = func(_ context.Context, method, endpoint string, opts *gophercloud.RequestOpts, original error, count uint) error {
				retries++
				if method != http.MethodGet || endpoint != "https://glance.example/reverse/glance/v2/images/fixed" {
					t.Fatal(method, endpoint)
				}
				if count > 1 {
					return original
				}
				client.SetToken("retry token")
				opts.MoreHeaders = map[string]string{"X-Retry": "ordinary"}
				switch mode {
				case "expanded OkCodes rejection":
					opts.OkCodes = append(opts.OkCodes, 418)
				case "changed request body":
					opts.JSONBody = map[string]bool{"unexpected": true}
				case "source mutation":
					client.Endpoint = "https://foreign.test/"
				case "hook cause":
					return errors.Join(original, marker)
				}
				return nil
			}
			got, err := service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: seed})
			if mode == "successful ordinary header retry" {
				if got == nil || got.Record == nil || got.Metadata == nil || got.Staged == nil || got.Metadata.StatusCode != 203 || err != nil || calls != 4 || retries != 1 {
					t.Fatal(got, err, calls, retries)
				}
				return
			}
			expectedCalls := 3
			code := 503
			body := "initial metadata rejection"
			if mode == "expanded OkCodes rejection" {
				expectedCalls = 4
				code = 418
				body = "actual metadata rejection"
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if got == nil || got.Staged == nil || got.Record == nil || got.Metadata != nil || err == nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != body || calls != expectedCalls {
				t.Fatal(got, err, native, calls, retries)
			}
			if mode == "changed request body" || mode == "source mutation" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if mode == "hook cause" && !errors.Is(err, marker) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordStageRejectedBodyCloseAndFinalFetchReauthKeepNativePolicy(t *testing.T) {
	t.Run("rejected PUT Close cause joins native error", func(t *testing.T) {
		calls := 0
		var handler taskCoreTransport
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
		service := New(client)
		seed := imageRecordStageFetched(t, service, &handler)
		marker := errors.New("rejected stage Close")
		body := &taskCoreBody{reader: strings.NewReader("actual denied"), closeErr: marker}
		handler = func(req *http.Request) (*http.Response, error) { return taskCoreHTTP(req, 403, body), nil }
		got, err := service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: seed})
		var native gophercloud.ErrUnexpectedResponseCode
		if got != nil || !errors.Is(err, marker) || !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != "actual denied" || calls != 2 || body.closes != 1 {
			t.Fatal(got, err, native, calls, body.closes)
		}
	})
	t.Run("final GET reauth retains stage evidence and fixed URL", func(t *testing.T) {
		calls, reauth := 0, 0
		var handler taskCoreTransport
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
		service := New(client)
		seed := imageRecordStageFetched(t, service, &handler)
		handler = func(req *http.Request) (*http.Response, error) {
			if calls == 2 {
				return taskCoreJSON(req, 204, "staged once"), nil
			}
			if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed" || req.Body != nil {
				t.Fatal(req.Method, req.URL)
			}
			if calls == 3 {
				return taskCoreJSON(req, 401, "expired metadata token"), nil
			}
			if calls != 4 || req.Header.Get("X-Auth-Token") != "reauth token" {
				t.Fatal(calls, req.Header)
			}
			return taskCoreJSON(req, 200, `{}`), nil
		}
		client.ProviderClient.ReauthFunc = func(context.Context) error { reauth++; client.SetToken("reauth token"); return nil }
		got, err := service.StageImageRecord(context.Background(), ImageRecordStageRequest{Record: seed})
		if got == nil || got.Staged == nil || got.Record == nil || got.Metadata == nil || err != nil || calls != 4 || reauth != 1 || string(got.Staged.Body) != "staged once" {
			t.Fatal(got, err, calls, reauth)
		}
	})
}
