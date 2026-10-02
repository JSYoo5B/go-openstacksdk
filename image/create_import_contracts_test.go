package image_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/image"
	"gophercloudsdk/image/v2/imagedata"
	"gophercloudsdk/image/v2/imageimport"
	sdkimages "gophercloudsdk/image/v2/images"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

// Pinned create_image's non-task upload/import branch does not wait for active.
// This explicit workflow owns its preflight, fixed route and phase evidence;
// optional waiting and retaining failed side effects are documented Go policy.
const createImportPrefix = "/reverse/glance/v2/"

type createImportRoundTrip func(*http.Request) (*http.Response, error)

func (f createImportRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func createImportHTTP(code int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}, "X-Evidence": {"actual"}}, Body: body}
}
func createImportJSON(id, status string) string {
	return fmt.Sprintf(`{"id":%q,"name":"worker","status":%q,"disk_format":"qcow2","container_format":"bare","vendor":{"number":9007199254740993}}`, id, status)
}
func createImportFields(t *testing.T, r *http.Request) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
		t.Error(err)
	}
	return fields
}

type createImportReader struct {
	data                          *strings.Reader
	reads, closes, seeks, lengths atomic.Int32
}

func createImportData(s string) *createImportReader {
	return &createImportReader{data: strings.NewReader(s)}
}
func (r *createImportReader) Read(p []byte) (int, error) { r.reads.Add(1); return r.data.Read(p) }
func (r *createImportReader) Close() error               { r.closes.Add(1); return nil }
func (r *createImportReader) Seek(o int64, w int) (int64, error) {
	r.seeks.Add(1)
	return r.data.Seek(o, w)
}
func (r *createImportReader) Len() int { r.lengths.Add(1); return r.data.Len() }

type createImportBlockingReader struct {
	started, release chan struct{}
	start, close     sync.Once
	closes           atomic.Int32
}

func (r *createImportBlockingReader) Read([]byte) (int, error) {
	r.start.Do(func() { close(r.started) })
	<-r.release
	return 0, io.EOF
}
func (r *createImportBlockingReader) Close() error {
	r.close.Do(func() { r.closes.Add(1); close(r.release) })
	return nil
}

type createImportSequence struct {
	mu    sync.Mutex
	steps []string
}

func (s *createImportSequence) add(step string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.steps = append(s.steps, step)
}
func (s *createImportSequence) value() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.steps...)
}

func TestCreateAndImportDefaultDirectSequenceAndEvidence(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("image", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + createImportPrefix
	data := createImportData("prefix-payload")
	_, _ = data.Seek(7, io.SeekStart)
	data.seeks.Store(0)
	var sequence createImportSequence
	var wrong atomic.Int32
	createdBody, stagedBody := createImportJSON("created", "queued"), createImportJSON("stage-response-id", "uploading")
	cloud.Mux.HandleFunc("POST "+createImportPrefix+"images", func(w http.ResponseWriter, r *http.Request) {
		sequence.add("create")
		fields := createImportFields(t, r)
		want := map[string]json.RawMessage{"name": json.RawMessage(`"worker"`), "disk_format": json.RawMessage(`"qcow2"`), "container_format": json.RawMessage(`"bare"`), "visibility": json.RawMessage(`"private"`)}
		if !reflect.DeepEqual(fields, want) {
			t.Error(fields, want)
		}
		w.Header().Set("X-Create", "actual")
		w.Header().Set("Location", "https://foreign.invalid/images/not-followed")
		testcloud.JSON(w, 201, createdBody)
	})
	cloud.Mux.HandleFunc("PUT "+createImportPrefix+"images/created/stage", func(w http.ResponseWriter, r *http.Request) {
		sequence.add("stage")
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != "payload" || r.Header.Get("Content-Type") != "application/octet-stream" || r.Header.Get("Accept") != "" || r.Header.Get("X-OpenStack-Image-Size") != "" || r.ContentLength != -1 {
			t.Error(string(body), err, r.Header, r.ContentLength)
		}
		w.Header().Set("X-Stage", "actual")
		w.WriteHeader(204)
	})
	cloud.Mux.HandleFunc("GET "+createImportPrefix+"images/created", func(w http.ResponseWriter, r *http.Request) {
		sequence.add("metadata")
		w.Header().Set("X-Metadata", "actual")
		testcloud.JSON(w, 200, stagedBody)
	})
	cloud.Mux.HandleFunc("POST "+createImportPrefix+"images/created/import", func(w http.ResponseWriter, r *http.Request) {
		sequence.add("import")
		fields := createImportFields(t, r)
		if len(fields) != 1 || string(fields["method"]) != `{"name":"glance-direct"}` {
			t.Error(fields)
		}
		w.Header().Set("X-Import", "actual")
		testcloud.JSON(w, 202, `{"accepted":true}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		wrong.Add(1)
		t.Error("unexpected implicit phase", r.Method, r.URL.String())
		w.WriteHeader(500)
	})
	v, err := image.New(client).CreateAndImport(context.Background(), image.CreateAndImportRequest{Name: "worker", Data: data})
	if err != nil || v == nil || v.ImageID != "created" || v.Created == nil || v.Created.StatusCode != 201 || v.Created.Image == nil || v.Created.Image.ID != "created" || string(v.Created.Body) != createdBody || v.Created.Header.Get("X-Create") != "actual" || v.Staged == nil || v.Staged.ImageID != "created" || v.Staged.Image == nil || v.Staged.Image.ID != "stage-response-id" || string(v.Staged.Body) != stagedBody || v.Staged.StatusCode != 200 || v.Staged.Acknowledgement == nil || v.Staged.Acknowledgement.StatusCode != 204 || v.Staged.Acknowledgement.Header.Get("X-Stage") != "actual" || v.Imported == nil || v.Imported.ImageID != "created" || v.Imported.StatusCode != 202 || string(v.Imported.Body) != `{"accepted":true}` || v.Imported.Header.Get("X-Import") != "actual" || v.Ready != nil || !reflect.DeepEqual(sequence.value(), []string{"create", "stage", "metadata", "import"}) || wrong.Load() != 0 || data.closes.Load() != 0 || data.seeks.Load() != 0 || data.lengths.Load() != 0 {
		t.Fatal(v, err, sequence.value(), wrong.Load(), data.closes.Load(), data.seeks.Load(), data.lengths.Load())
	}
	v.Created.Header.Set("X-Stage", "caller")
	v.Created.Body[0] = '!'
	v.Created.Image.Status = "caller"
	if v.Staged.Acknowledgement.Header.Get("X-Stage") != "actual" || string(v.Staged.Body) != stagedBody || v.Staged.Image.Status != "uploading" || string(v.Imported.Body) != `{"accepted":true}` {
		t.Fatal("phase evidence aliases", v)
	}
	if client.Endpoint != cloud.Server.URL+"/catalog/unused/" || client.ResourceBase != cloud.Server.URL+createImportPrefix {
		t.Fatal(client)
	}
	t.Run("explicit zero size does not inspect or expose reader", func(t *testing.T) {
		cloud := testcloud.New(t)
		data := createImportData("payload")
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = createImportRoundTrip(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			switch {
			case r.URL.Path == createImportPrefix+"images":
				_, _ = io.Copy(io.Discard, r.Body)
				return createImportHTTP(201, io.NopCloser(strings.NewReader(createImportJSON("created", "queued")))), nil
			case r.Method == http.MethodPut:
				_, seeker := r.Body.(io.Seeker)
				_, length := r.Body.(interface{ Len() int })
				if r.URL.Path != createImportPrefix+"images/created/stage" || r.GetBody != nil || seeker || length || r.ContentLength != 0 || r.Header.Get("X-OpenStack-Image-Size") != "0" {
					t.Error(r.URL, r.Header, r.ContentLength, r.GetBody != nil, seeker, length)
				}
				body, err := io.ReadAll(r.Body)
				_ = r.Body.Close()
				if err != nil || string(body) != "payload" {
					t.Error(string(body), err)
				}
				return createImportHTTP(204, io.NopCloser(strings.NewReader(""))), nil
			case r.Method == http.MethodGet:
				if r.Header.Get("X-OpenStack-Image-Size") != "" {
					t.Error("stage size leaked into metadata", r.Header)
				}
				return createImportHTTP(200, io.NopCloser(strings.NewReader(createImportJSON("decoy", "uploading")))), nil
			default:
				_, _ = io.Copy(io.Discard, r.Body)
				if r.Header.Get("X-OpenStack-Image-Size") != "" {
					t.Error("stage size leaked into import", r.Header)
				}
				return createImportHTTP(202, io.NopCloser(strings.NewReader(`{}`))), nil
			}
		})
		v, err := image.New(cloud.Client("image", createImportPrefix)).CreateAndImport(context.Background(), image.CreateAndImportRequest{Name: "worker", Data: data}, image.WithCreateImportStage(imagedata.StageOpts{Size: new(int64)}))
		if err != nil || v == nil || v.Imported == nil || calls.Load() != 4 || data.reads.Load() == 0 || data.closes.Load() != 0 || data.seeks.Load() != 0 || data.lengths.Load() != 0 {
			t.Fatal(v, err, calls.Load(), data.reads.Load(), data.closes.Load(), data.seeks.Load(), data.lengths.Load())
		}
	})
}

func TestCreateAndImportCanonicalIdentityAndPostStageFormats(t *testing.T) {
	for _, tc := range []struct {
		name, created, staged string
		success               bool
		stage                 bool
	}{
		{"incidental ID and case extensions", `{"id":"created","ID":"typed-create-decoy","status":"queued","STATUS":"active","disk_format":"qcow2","container_format":"bare"}`, `{"id":"../unsafe-response-id","ID":"typed-stage-decoy","disk_format":"raw","DISK_FORMAT":"","container_format":"bare","CONTAINER_FORMAT":"","status":"saving"}`, true, true},
		{"missing canonical created ID", `{"ID":"typed-create-decoy","status":"queued"}`, "", false, false},
		{"null canonical created ID", `{"id":null,"ID":"typed-create-decoy","status":"queued"}`, "", false, false},
		{"typed canonical created ID", `{"id":false,"ID":"typed-create-decoy","status":"queued"}`, "", false, false},
		{"unsafe canonical created ID", `{"id":"../escape","status":"queued"}`, "", false, false},
		{"foreign canonical created ID", `{"id":"https:escape","status":"queued"}`, "", false, false},
		{"missing canonical queued", `{"id":"created","STATUS":"queued"}`, "", false, false},
		{"case in canonical queued", `{"id":"created","status":"QUEUED"}`, "", false, false},
		{"missing canonical disk", createImportJSON("created", "queued"), `{"id":"decoy","DISK_FORMAT":"raw","container_format":"bare"}`, false, true},
		{"null canonical container", createImportJSON("created", "queued"), `{"id":"decoy","disk_format":"qcow2","container_format":null,"CONTAINER_FORMAT":"bare"}`, false, true},
		{"empty canonical disk", createImportJSON("created", "queued"), `{"id":"decoy","disk_format":"","DISK_FORMAT":"raw","container_format":"bare"}`, false, true},
		{"typed canonical container", createImportJSON("created", "queued"), `{"id":"decoy","disk_format":"qcow2","container_format":false,"CONTAINER_FORMAT":"bare"}`, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			data := createImportData("payload")
			var puts, gets, imports, wrong atomic.Int32
			cloud.Mux.HandleFunc("POST "+createImportPrefix+"images", func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				testcloud.JSON(w, 201, tc.created)
			})
			cloud.Mux.HandleFunc("PUT "+createImportPrefix+"images/created/stage", func(w http.ResponseWriter, r *http.Request) {
				puts.Add(1)
				_, _ = io.Copy(io.Discard, r.Body)
				w.WriteHeader(204)
			})
			cloud.Mux.HandleFunc("GET "+createImportPrefix+"images/created", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); testcloud.JSON(w, 200, tc.staged) })
			cloud.Mux.HandleFunc("POST "+createImportPrefix+"images/created/import", func(w http.ResponseWriter, r *http.Request) {
				imports.Add(1)
				_, _ = io.Copy(io.Discard, r.Body)
				testcloud.JSON(w, 202, `{}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { wrong.Add(1); w.WriteHeader(500) })
			v, err := image.New(cloud.Client("image", createImportPrefix)).CreateAndImport(context.Background(), image.CreateAndImportRequest{Name: "worker", Data: data})
			if v == nil || v.Created == nil || string(v.Created.Body) != tc.created || v.Created.StatusCode != 201 || wrong.Load() != 0 {
				t.Fatal(v, err, wrong.Load())
			}
			if tc.success {
				if err != nil || v.ImageID != "created" || v.Created.Image.ID != "created" || v.Staged.Image.ID != "../unsafe-response-id" || v.Imported == nil || v.Imported.ImageID != "created" || imports.Load() != 1 {
					t.Fatal(v, err, imports.Load())
				}
			} else {
				var proof *resource.ResponseError
				if err == nil || !errors.As(err, &proof) || imports.Load() != 0 {
					t.Fatal(v, err, proof, imports.Load())
				}
				want := 201
				body := tc.created
				if tc.stage {
					want, body = 200, tc.staged
				}
				if proof.StatusCode != want || string(proof.Body) != body {
					t.Fatal(proof, want, body)
				}
			}
			want := int32(0)
			if tc.stage {
				want = 1
				if v.Staged == nil || v.Staged.Acknowledgement == nil || string(v.Staged.Body) != tc.staged {
					t.Fatal(v)
				}
			} else if data.reads.Load() != 0 || v.Staged != nil {
				t.Fatal(v, data.reads.Load())
			}
			if puts.Load() != want || gets.Load() != want {
				t.Fatal(v, err, puts.Load(), gets.Load())
			}
		})
	}
}

func TestCreateAndImportCompletePreflight(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input image.CreateAndImportRequest
		opts  []image.CreateImportOption
	}{
		{"missing name", image.CreateAndImportRequest{}, nil},
		{"missing direct reader", image.CreateAndImportRequest{Name: "worker"}, nil},
		{"typed nil reader", image.CreateAndImportRequest{Name: "worker", Data: (*createImportReader)(nil)}, nil},
		{"nil option", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{nil}},
		{"nil stage callback", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportStageOptions(nil)}},
		{"nil import callback", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportImportOptions(nil)}},
		{"negative metadata", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportMetadata(image.CreateImportMetadataOpts{MinDisk: func() *int { v := -1; return &v }()})}},
		{"protected core property", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportMetadata(image.CreateImportMetadataOpts{Properties: map[string]any{"status": "active"}})}},
		{"nonJSON property", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportMetadata(image.CreateImportMetadataOpts{Properties: map[string]any{"vendor": make(chan int)}})}},
		{"metadata auth header", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportMetadata(image.CreateImportMetadataOpts{Headers: map[string]string{"X-Auth-Token": "override"}})}},
		{"metadata header newline", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportMetadata(image.CreateImportMetadataOpts{Headers: map[string]string{"X-Trace": "line\nbreak"}})}},
		{"negative stage size", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportStage(imagedata.StageOpts{Size: func() *int64 { v := int64(-1); return &v }()})}},
		{"stage owned header", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportStageOptions(imagedata.WithStageHeader("Accept", "override"))}},
		{"empty selected store", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportImportOptions(imageimport.WithImportStore(""))}},
		{"conflicting selectors", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportImportOptions(imageimport.WithImportAllStores(true), imageimport.WithImportStores("fast"))}},
		{"import owned field", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportImportOptions(imageimport.WithImportField("method", map[string]any{"name": "web-download"}))}},
		{"import owned method field", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportImportOptions(imageimport.WithImportMethodField("uri", "https://source.invalid"))}},
		{"import nonJSON extension", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportImportOptions(imageimport.WithImportField("vendor", make(chan int)))}},
		{"web missing URI", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportImport(imageimport.ImportOpts{Method: imageimport.WebDownloadMethod})}},
		{"partial remote", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportImport(imageimport.ImportOpts{Method: imageimport.GlanceDownloadMethod, RemoteRegion: "region"})}},
		{"copy existing-only", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportImport(imageimport.ImportOpts{Method: imageimport.CopyImageMethod})}},
		{"unknown fresh method", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportImport(imageimport.ImportOpts{Method: "vendor-method"})}},
		{"negative wait budget", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportWait(image.CreateImportWaitOpts{Timeout: func() *time.Duration { v := -time.Second; return &v }()})}},
		{"zero wait interval", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportWait(image.CreateImportWaitOpts{PollInterval: new(time.Duration)})}},
		{"blank wait failure state", image.CreateAndImportRequest{Name: "worker"}, []image.CreateImportOption{image.WithCreateImportWait(image.CreateImportWaitOpts{FailureStates: []string{" "}})}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			data := createImportData("payload")
			var calls atomic.Int32
			if tc.name != "missing name" && tc.name != "missing direct reader" && tc.name != "typed nil reader" {
				tc.input.Data = data
			}
			cloud.Provider.HTTPClient.Transport = createImportRoundTrip(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return createImportHTTP(500, io.NopCloser(strings.NewReader(`{}`))), nil
			})
			v, err := image.New(cloud.Client("image", createImportPrefix)).CreateAndImport(context.Background(), tc.input, tc.opts...)
			want := error(resource.ErrInvalidOption)
			if tc.name == "copy existing-only" || tc.name == "unknown fresh method" {
				want = resource.ErrUnsupported
			}
			if v != nil || !errors.Is(err, want) || calls.Load() != 0 || data.reads.Load() != 0 || data.closes.Load() != 0 {
				t.Fatal(v, err, calls.Load(), data.reads.Load(), data.closes.Load())
			}
		})
	}
	for _, mode := range []string{"nil service", "nil client", "nil provider", "wrong type", "foreign base", "query base", "source selector conflict", "canceled", "nil context", "option source change"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", createImportPrefix)
			ctx := context.Background()
			data := createImportData("payload")
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = createImportRoundTrip(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return createImportHTTP(500, io.NopCloser(strings.NewReader(`{}`))), nil
			})
			var opts []image.CreateImportOption
			want := error(resource.ErrInvalidOption)
			switch mode {
			case "nil client":
				client = nil
			case "nil provider":
				client.ProviderClient = nil
			case "wrong type":
				client.Type = "compute"
				want = resource.ErrUnsupported
			case "foreign base":
				client.ResourceBase = "https://foreign.invalid/v2/"
			case "query base":
				client.ResourceBase = cloud.Server.URL + createImportPrefix + "?unsafe=base"
			case "source selector conflict":
				client.MoreHeaders = map[string]string{"X-Image-Meta-Store": "legacy"}
				opts = []image.CreateImportOption{image.WithCreateImportImportOptions(imageimport.WithImportStores("selected"))}
			case "canceled":
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
				want = context.Canceled
			case "nil context":
				ctx = nil
			case "option source change":
				opts = []image.CreateImportOption{func(*image.CreateImportOpts) error { client.Type = "compute"; return nil }}
				want = resource.ErrUnsupported
			}
			service := image.New(client)
			if mode == "nil service" {
				service = nil
			}
			v, err := service.CreateAndImport(ctx, image.CreateAndImportRequest{Name: "worker", Data: data}, opts...)
			if v != nil || !errors.Is(err, want) || calls.Load() != 0 || data.reads.Load() != 0 {
				t.Fatal(v, err, calls.Load(), data.reads.Load())
			}
		})
	}
	t.Run("callbacks finish once before later preflight failure", func(t *testing.T) {
		cloud := testcloud.New(t)
		data := createImportData("payload")
		var calls, workflow, stage, importing atomic.Int32
		cloud.Provider.HTTPClient.Transport = createImportRoundTrip(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return createImportHTTP(500, io.NopCloser(strings.NewReader(`{}`))), nil
		})
		v, err := image.New(cloud.Client("image", createImportPrefix)).CreateAndImport(context.Background(), image.CreateAndImportRequest{Name: "worker", Data: data},
			func(*image.CreateImportOpts) error { workflow.Add(1); return nil },
			image.WithCreateImportStageOptions(func(*imagedata.StageOpts) error { stage.Add(1); return nil }),
			image.WithCreateImportImportOptions(func(*imageimport.ImportOpts) error { importing.Add(1); return nil }),
			image.WithCreateImportWait(image.CreateImportWaitOpts{PollInterval: new(time.Duration)}))
		if v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 || data.reads.Load() != 0 || workflow.Load() != 1 || stage.Load() != 1 || importing.Load() != 1 {
			t.Fatal(v, err, calls.Load(), data.reads.Load(), workflow.Load(), stage.Load(), importing.Load())
		}
	})
}

func TestCreateAndImportPreparedSnapshotsAndCallbacksOnce(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, workflowCallbacks, stageCallbacks, importCallbacks atomic.Int32
	min := 0
	size := int64(7)
	flag := false
	visibility := image.VisibilityShared
	tags := []string{"linux"}
	stores := []string{"fast", "fast", "slow"}
	properties := map[string]any{"vendor": map[string]any{"number": json.Number("9007199254740993")}}
	fields := map[string]any{"vendor_import": map[string]any{"number": json.Number("9007199254740993")}}
	bulk := image.WithCreateImportOpts(image.CreateImportOpts{
		Metadata: image.CreateImportMetadataOpts{DiskFormat: "raw", ContainerFormat: "bare", Visibility: &visibility, Protected: &flag, Hidden: &flag, MinDisk: &min, MinRAM: &min, Tags: tags, Properties: properties, Headers: map[string]string{"X-Metadata": "only-create"}},
		Stage:    imagedata.StageOpts{Size: &size, Headers: map[string]string{"X-Stage": "captured"}},
		Import:   imageimport.ImportOpts{Stores: stores, AllStores: &flag, AllStoresMustSucceed: &flag, Fields: fields, Headers: map[string]string{"X-Import": "captured"}},
	})
	min = -1
	size = -1
	flag = true
	visibility = image.VisibilityPublic
	tags[0] = "changed"
	stores[0] = "changed"
	properties["vendor"].(map[string]any)["number"] = json.Number("0")
	fields["vendor_import"].(map[string]any)["number"] = json.Number("0")
	var retained *image.CreateImportOpts
	var retainedStage *imagedata.StageOpts
	var retainedImport *imageimport.ImportOpts
	options := []image.CreateImportOption{bulk,
		func(o *image.CreateImportOpts) error { workflowCallbacks.Add(1); retained = o; return nil },
		image.WithCreateImportStageOptions(func(o *imagedata.StageOpts) error { stageCallbacks.Add(1); retainedStage = o; return nil }),
		image.WithCreateImportImportOptions(func(o *imageimport.ImportOpts) error { importCallbacks.Add(1); retainedImport = o; return nil }),
	}
	cloud.Provider.HTTPClient.Transport = createImportRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if workflowCallbacks.Load() != 1 || stageCallbacks.Load() != 1 || importCallbacks.Load() != 1 {
			t.Error("callbacks deferred or repeated", workflowCallbacks.Load(), stageCallbacks.Load(), importCallbacks.Load())
		}
		// Retained custom callback state must no longer own the prepared request.
		retained.Metadata.Properties["vendor"] = map[string]any{"number": json.Number("0")}
		retained.Metadata.Tags[0] = "late"
		retainedStage.Headers["X-Stage"] = "late"
		*retainedStage.Size = 999
		retainedImport.Fields["vendor_import"] = map[string]any{"number": json.Number("0")}
		retainedImport.Stores[0] = "late"
		switch {
		case r.URL.Path == createImportPrefix+"images":
			fields := createImportFields(t, r)
			if r.Method != http.MethodPost || string(fields["disk_format"]) != `"raw"` || string(fields["visibility"]) != `"shared"` || string(fields["min_disk"]) != "0" || string(fields["min_ram"]) != "0" || string(fields["protected"]) != "false" || string(fields["os_hidden"]) != "false" || string(fields["tags"]) != `["linux"]` || string(fields["vendor"]) != `{"number":9007199254740993}` || r.Header.Get("X-Metadata") != "only-create" || r.Header.Get("X-Stage") != "" || r.Header.Get("X-Import") != "" {
				t.Error(fields, r.Header)
			}
			return createImportHTTP(201, io.NopCloser(strings.NewReader(createImportJSON("created", "queued")))), nil
		case r.Method == http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != "payload" || r.Header.Get("X-OpenStack-Image-Size") != "7" || r.Header.Get("X-Stage") != "captured" || r.Header.Get("X-Metadata") != "" || r.Header.Get("X-Import") != "" {
				t.Error(string(body), err, r.Header)
			}
			return createImportHTTP(204, io.NopCloser(strings.NewReader(""))), nil
		case strings.HasSuffix(r.URL.Path, "/import"):
			fields := createImportFields(t, r)
			if string(fields["stores"]) != `["fast","fast","slow"]` || string(fields["all_stores"]) != "false" || string(fields["all_stores_must_succeed"]) != "false" || string(fields["vendor_import"]) != `{"number":9007199254740993}` || r.Header.Get("X-Import") != "captured" || r.Header.Get("X-Metadata") != "" || r.Header.Get("X-Stage") != "" {
				t.Error(fields, r.Header)
			}
			return createImportHTTP(202, io.NopCloser(strings.NewReader(`{}`))), nil
		default:
			if r.Method != http.MethodGet || r.Header.Get("X-Stage") != "captured" || r.Header.Get("X-Metadata") != "" || r.Header.Get("X-Import") != "" {
				t.Error(r.Method, r.Header)
			}
			return createImportHTTP(200, io.NopCloser(strings.NewReader(createImportJSON("decoy", "uploading")))), nil
		}
	})
	v, err := image.New(cloud.Client("image", createImportPrefix)).CreateAndImport(context.Background(), image.CreateAndImportRequest{Name: "worker", Data: strings.NewReader("payload")}, options...)
	if err != nil || v == nil || v.Imported == nil || calls.Load() != 4 || workflowCallbacks.Load() != 1 || stageCallbacks.Load() != 1 || importCallbacks.Load() != 1 {
		t.Fatal(v, err, calls.Load(), workflowCallbacks.Load(), stageCallbacks.Load(), importCallbacks.Load())
	}
	t.Run("bulk replacement and concurrent reusable policy", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = createImportRoundTrip(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.URL.Path == createImportPrefix+"images" {
				fields := createImportFields(t, r)
				if string(fields["visibility"]) != `"shared"` || string(fields["vendor"]) != `{"number":9007199254740993}` {
					t.Error(fields)
				}
				return createImportHTTP(201, io.NopCloser(strings.NewReader(createImportJSON("created", "queued")))), nil
			}
			if r.Method == http.MethodPut {
				_, _ = io.Copy(io.Discard, r.Body)
				if r.Header.Get("X-OpenStack-Image-Size") != "7" {
					t.Error(r.Header)
				}
				return createImportHTTP(204, io.NopCloser(strings.NewReader(""))), nil
			}
			if r.Method == http.MethodGet {
				return createImportHTTP(200, io.NopCloser(strings.NewReader(createImportJSON("created", "uploading")))), nil
			}
			_, _ = io.Copy(io.Discard, r.Body)
			return createImportHTTP(202, io.NopCloser(strings.NewReader(`{}`))), nil
		})
		service := image.New(cloud.Client("image", createImportPrefix))
		done := make(chan error, 2)
		for range 2 {
			go func() {
				v, err := service.CreateAndImport(context.Background(), image.CreateAndImportRequest{Name: "worker", Data: strings.NewReader("payload")}, image.WithCreateImportStage(imagedata.StageOpts{Size: func() *int64 { v := int64(-1); return &v }()}), bulk)
				if err == nil && (v == nil || v.Imported == nil) {
					err = errors.New("actual import missing")
				}
				done <- err
			}()
		}
		for range 2 {
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		}
		if calls.Load() != 8 {
			t.Fatal(calls.Load())
		}
	})
}

func TestCreateAndImportRemoteModesAndSelectors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		policy imageimport.ImportOpts
		method string
	}{
		{"web plural and explicit false", imageimport.ImportOpts{Method: imageimport.WebDownloadMethod, URI: "https://source.invalid/image?unchanged=true", Stores: []string{"fast", "fast", "slow"}, AllStores: func() *bool { v := false; return &v }(), AllStoresMustSucceed: func() *bool { v := false; return &v }()}, `{"name":"web-download","uri":"https://source.invalid/image?unchanged=true"}`},
		{"remote optional interface absent", imageimport.ImportOpts{Method: imageimport.GlanceDownloadMethod, RemoteRegion: "RegionTwo", RemoteImageID: "remote-id", Store: func() *string { v := "fast"; return &v }()}, `{"glance_image_id":"remote-id","glance_region":"RegionTwo","name":"glance-download"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var creates, imports, wrong atomic.Int32
			cloud.Mux.HandleFunc("POST "+createImportPrefix+"images", func(w http.ResponseWriter, r *http.Request) {
				creates.Add(1)
				_, _ = io.Copy(io.Discard, r.Body)
				testcloud.JSON(w, 201, createImportJSON("created", "queued"))
			})
			cloud.Mux.HandleFunc("POST "+createImportPrefix+"images/created/import", func(w http.ResponseWriter, r *http.Request) {
				imports.Add(1)
				fields := createImportFields(t, r)
				if string(fields["method"]) != tc.method {
					t.Error(fields)
				}
				if tc.policy.Store != nil {
					if r.Header.Get("X-Image-Meta-Store") != "fast" || fields["store"] != nil || fields["stores"] != nil {
						t.Error(fields, r.Header)
					}
				} else if string(fields["stores"]) != `["fast","fast","slow"]` || string(fields["all_stores"]) != "false" || string(fields["all_stores_must_succeed"]) != "false" || r.Header.Get("X-Image-Meta-Store") != "" {
					t.Error(fields, r.Header)
				}
				testcloud.JSON(w, 202, `{"accepted":true}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { wrong.Add(1); w.WriteHeader(500) })
			v, err := image.New(cloud.Client("image", createImportPrefix)).CreateAndImport(context.Background(), image.CreateAndImportRequest{Name: "worker"}, image.WithCreateImportImport(tc.policy))
			if err != nil || v == nil || v.Created == nil || v.Staged != nil || v.Imported == nil || v.Ready != nil || creates.Load() != 1 || imports.Load() != 1 || wrong.Load() != 0 {
				t.Fatal(v, err, creates.Load(), imports.Load(), wrong.Load())
			}
		})
	}
	for _, unused := range []string{"reader", "stage size", "stage headers"} {
		t.Run("remote rejects "+unused, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			data := createImportData("payload")
			cloud.Provider.HTTPClient.Transport = createImportRoundTrip(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return createImportHTTP(500, io.NopCloser(strings.NewReader(`{}`))), nil
			})
			input := image.CreateAndImportRequest{Name: "worker"}
			policy := image.CreateImportOpts{Import: imageimport.ImportOpts{Method: imageimport.WebDownloadMethod, URI: "https://source.invalid/image"}}
			switch unused {
			case "reader":
				input.Data = data
			case "stage size":
				policy.Stage.Size = new(int64)
			default:
				policy.Stage.Headers = map[string]string{"X-Trace": "unused"}
			}
			v, err := image.New(cloud.Client("image", createImportPrefix)).CreateAndImport(context.Background(), input, image.WithCreateImportOpts(policy))
			if v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 || data.reads.Load() != 0 {
				t.Fatal(v, err, calls.Load(), data.reads.Load())
			}
		})
	}
}

type createImportBrokenBody struct {
	first []byte
	cause error
}

func (r *createImportBrokenBody) Read(p []byte) (int, error) {
	if len(r.first) > 0 {
		n := copy(p, r.first)
		r.first = r.first[n:]
		return n, nil
	}
	return 0, r.cause
}
func (*createImportBrokenBody) Close() error { return nil }

func TestCreateAndImportPartialProofAndTerminalFailures(t *testing.T) {
	for _, tc := range []struct {
		phase, outcome string
		code           int
	}{
		{"create", "native", 200}, {"create", "native", 403}, {"create", "read", 201}, {"create", "decode", 201},
		{"stage", "native", 202}, {"stage", "native", 401}, {"stage", "read", 204}, {"stage", "transport", 404},
		{"metadata", "native", 404}, {"metadata", "read", 200}, {"metadata", "decode", 200},
		{"import", "native", 400}, {"import", "read", 202}, {"import", "transport", 404},
		{"create", "canceled", 404}, {"create", "canceled", 201}, {"stage", "canceled", 204},
		{"metadata", "canceled", 200}, {"import", "canceled", 404}, {"import", "canceled", 202},
	} {
		t.Run(tc.phase+"/"+tc.outcome+fmt.Sprint(tc.code), func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("actual phase body stopped")
			cancelCause := errors.New("caller canceled phase")
			transportCause := &gophercloud.ErrUnexpectedResponseCode{Actual: 404}
			var sequence createImportSequence
			var callbacks atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				callbacks.Add(1)
				return err
			}
			cloud.Provider.HTTPClient.Transport = createImportRoundTrip(func(r *http.Request) (*http.Response, error) {
				phase, code, body := "metadata", 200, createImportJSON("decoy", "uploading")
				switch {
				case r.URL.Path == createImportPrefix+"images":
					phase, code, body = "create", 201, createImportJSON("created", "queued")
				case r.Method == http.MethodPut:
					phase, code, body = "stage", 204, "ack"
				case strings.HasSuffix(r.URL.Path, "/import"):
					phase, code, body = "import", 202, `{"accepted":true}`
				}
				sequence.add(phase)
				if r.Method == http.MethodPut || r.Method == http.MethodPost {
					_, _ = io.Copy(io.Discard, r.Body)
				}
				if phase == tc.phase {
					code = tc.code
					if tc.outcome == "transport" {
						return nil, transportCause
					}
					if tc.outcome == "canceled" {
						cancel(cancelCause)
					}
					if tc.outcome == "decode" {
						body = `{"min_ram":"bad"}`
						if phase == "create" {
							body = `{"id":"created","status":"queued","min_ram":"bad"}`
						}
					}
					if tc.outcome == "read" {
						body = phase + "-prefix"
						wire := createImportHTTP(code, &createImportBrokenBody{first: []byte(body), cause: cause})
						wire.Header.Set("X-Phase", phase)
						return wire, nil
					}
				}
				wire := createImportHTTP(code, io.NopCloser(strings.NewReader(body)))
				wire.Header.Set("X-Phase", phase)
				return wire, nil
			})
			data := createImportData("payload")
			v, err := image.New(cloud.Client("image", createImportPrefix)).CreateAndImport(ctx, image.CreateAndImportRequest{Name: "worker", Data: data})
			if err == nil || data.closes.Load() != 0 || data.seeks.Load() != 0 {
				t.Fatal(v, err, data.closes.Load(), data.seeks.Load())
			}
			want := []string{"create"}
			switch tc.phase {
			case "stage":
				want = append(want, "stage")
			case "metadata":
				want = append(want, "stage", "metadata")
			case "import":
				want = append(want, "stage", "metadata", "import")
			}
			if !reflect.DeepEqual(sequence.value(), want) {
				t.Fatal("failed phase replayed or cleaned up", sequence.value(), want, err)
			}
			if tc.phase == "create" && tc.code != 201 {
				if v != nil || data.reads.Load() != 0 {
					t.Fatal(v, err, data.reads.Load())
				}
			} else if v == nil || v.Created == nil || v.Created.StatusCode != 201 || v.Ready != nil {
				t.Fatal(v, err)
			}
			if tc.phase != "create" && (v.ImageID != "created" || string(v.Created.Body) != createImportJSON("created", "queued")) {
				t.Fatal(v, err)
			}
			if tc.phase == "stage" {
				if tc.code == 204 && (v.Staged == nil || v.Staged.Acknowledgement == nil || v.Staged.Acknowledgement.StatusCode != 204 || v.Staged.StatusCode != 0) {
					t.Fatal(v, err)
				}
				if tc.code != 204 && v.Staged != nil {
					t.Fatal(v, err)
				}
			}
			if tc.phase == "metadata" && (v.Staged == nil || v.Staged.Acknowledgement == nil || v.Staged.Acknowledgement.StatusCode != 204 || v.Staged.Image != nil || (tc.code == 200 && v.Staged.StatusCode != 200)) {
				t.Fatal(v, err)
			}
			if tc.phase == "import" {
				if v.Staged == nil || v.Staged.StatusCode != 200 || v.Staged.Image == nil {
					t.Fatal(v, err)
				}
				body := `{"accepted":true}`
				if tc.outcome == "read" {
					body = "import-prefix"
				}
				if tc.code == 202 && (v.Imported == nil || v.Imported.StatusCode != 202 || string(v.Imported.Body) != body) {
					t.Fatal(v, err)
				}
				if tc.code != 202 && v.Imported != nil {
					t.Fatal(v, err)
				}
			}
			if tc.outcome == "read" || tc.outcome == "decode" {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != tc.code || proof.Header.Get("X-Phase") != tc.phase {
					t.Fatal(v, err, proof)
				}
				if tc.outcome == "read" && (!errors.Is(err, cause) || string(proof.Body) != tc.phase+"-prefix" || callbacks.Load() != 0) {
					t.Fatal(v, err, proof, callbacks.Load())
				}
				if tc.outcome == "decode" {
					var native *json.UnmarshalTypeError
					if !errors.As(err, &native) {
						t.Fatal(err)
					}
				}
			}
			if tc.outcome == "transport" && (!errors.Is(err, transportCause) || errors.Is(err, resource.ErrNotFound)) {
				t.Fatal(v, err)
			}
			if tc.outcome == "native" && !gophercloud.ResponseCodeIs(err, tc.code) {
				t.Fatal(v, err)
			}
			if tc.outcome == "canceled" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
				t.Fatal(v, err)
			}
		})
	}
	for _, tc := range []struct{ name, body string }{
		{"malformed JSON", `{"id":"created"`},
		{"trailing JSON", `{"id":"created","status":"queued"}{}`},
		{"array body", `[]`},
		{"null body", `null`},
		{"invalid native time", `{"id":"created","status":"queued","created_at":"invalid"}`},
		{"invalid UTF8", "{\"id\":\"created\",\"status\":\"queued\",\"vendor\":\"" + string([]byte{255}) + "\"}"},
	} {
		t.Run("accepted creation "+tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			data := createImportData("payload")
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = createImportRoundTrip(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != createImportPrefix+"images" {
					t.Error("invalid creation triggered later phase", r.Method, r.URL)
				}
				_, _ = io.Copy(io.Discard, r.Body)
				return createImportHTTP(201, io.NopCloser(strings.NewReader(tc.body))), nil
			})
			v, err := image.New(cloud.Client("image", createImportPrefix)).CreateAndImport(context.Background(), image.CreateAndImportRequest{Name: "worker", Data: data})
			var proof *resource.ResponseError
			if err == nil || !errors.As(err, &proof) || proof.StatusCode != 201 || string(proof.Body) != tc.body || proof.Header.Get("X-Evidence") != "actual" || v == nil || v.Created == nil || v.Created.StatusCode != 201 || string(v.Created.Body) != tc.body || v.Created.Image != nil || v.Staged != nil || v.Imported != nil || calls.Load() != 1 || data.reads.Load() != 0 {
				t.Fatal(v, err, proof, calls.Load(), data.reads.Load())
			}
			v.Created.Body[0] = '!'
			v.Created.Header.Set("X-Evidence", "caller")
			if string(proof.Body) != tc.body || proof.Header.Get("X-Evidence") != "actual" {
				t.Fatal("error proof aliases result evidence", proof)
			}
		})
	}
}

func TestCreateAndImportExplicitWaitAndFailureRetention(t *testing.T) {
	for _, tc := range []struct {
		name     string
		states   []string
		policy   image.CreateImportWaitOpts
		want     error
		deadline bool
	}{
		{"default active wait", []string{"active"}, image.CreateImportWaitOpts{}, nil, true},
		{"default killed failure", []string{"killed"}, image.CreateImportWaitOpts{}, resource.ErrFailedState, true},
		{"default deleted failure", []string{"deleted"}, image.CreateImportWaitOpts{}, resource.ErrFailedState, true},
		{"empty failures disable", []string{"killed", "active"}, image.CreateImportWaitOpts{PollInterval: func() *time.Duration { v := time.Millisecond; return &v }(), FailureStates: []string{}}, nil, true},
		{"target precedes failure", []string{"active"}, image.CreateImportWaitOpts{FailureStates: []string{"active"}}, nil, true},
		{"explicit unlimited", []string{"active"}, image.CreateImportWaitOpts{Timeout: new(time.Duration)}, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, imports, wrong atomic.Int32
			cloud.Mux.HandleFunc("POST "+createImportPrefix+"images", func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				testcloud.JSON(w, 201, createImportJSON("created", "queued"))
			})
			cloud.Mux.HandleFunc("PUT "+createImportPrefix+"images/created/stage", func(w http.ResponseWriter, r *http.Request) { _, _ = io.Copy(io.Discard, r.Body); w.WriteHeader(204) })
			cloud.Mux.HandleFunc("POST "+createImportPrefix+"images/created/import", func(w http.ResponseWriter, r *http.Request) {
				imports.Add(1)
				_, _ = io.Copy(io.Discard, r.Body)
				testcloud.JSON(w, 202, `{"accepted":true}`)
			})
			cloud.Mux.HandleFunc("GET "+createImportPrefix+"images/created", func(w http.ResponseWriter, r *http.Request) {
				n := gets.Add(1)
				if n == 1 {
					testcloud.JSON(w, 200, createImportJSON("stage-decoy", "uploading"))
					return
				}
				if imports.Load() != 1 {
					t.Error("wait started before import accepted")
				}
				state := tc.states[len(tc.states)-1]
				if int(n)-2 < len(tc.states) {
					state = tc.states[int(n)-2]
				}
				testcloud.JSON(w, 200, createImportJSON("wait-response-decoy", state))
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { wrong.Add(1); w.WriteHeader(500) })
			transport := cloud.Provider.HTTPClient.Transport
			cloud.Provider.HTTPClient.Transport = createImportRoundTrip(func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodGet && imports.Load() == 1 {
					deadline, present := r.Context().Deadline()
					if present != tc.deadline {
						t.Error("wait deadline presence", present, tc.deadline)
					}
					if present && (time.Until(deadline) < 4*time.Minute || time.Until(deadline) > 5*time.Minute) {
						t.Error("default wait timeout changed", deadline)
					}
				}
				return transport.RoundTrip(r)
			})
			v, err := image.New(cloud.Client("image", createImportPrefix)).CreateAndImport(context.Background(), image.CreateAndImportRequest{Name: "worker", Data: strings.NewReader("payload")}, image.WithCreateImportWait(tc.policy))
			if !errors.Is(err, tc.want) || v == nil || v.ImageID != "created" || v.Created == nil || v.Staged == nil || v.Imported == nil || v.Imported.StatusCode != 202 || string(v.Imported.Body) != `{"accepted":true}` || wrong.Load() != 0 || gets.Load() != int32(1+len(tc.states)) {
				t.Fatal(v, err, gets.Load(), wrong.Load())
			}
			if tc.want == nil {
				if v.Ready == nil || v.Ready.ID != "wait-response-decoy" || v.Ready.Status != "active" {
					t.Fatal(v, err)
				}
				v.Ready.Status = "caller"
				if v.Staged.Image.Status != "uploading" || v.Created.Image.Status != "queued" {
					t.Fatal("Ready aliases previous phase", v)
				}
			} else if v.Ready != nil {
				t.Fatal("failed wait fabricated result", v)
			}
		})
	}
	for _, mode := range []string{"timeout", "parent deadline", "caller canceled"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			if mode == "parent deadline" {
				parent, finish := context.WithTimeout(ctx, 200*time.Millisecond)
				defer finish()
				ctx = parent
			}
			cause := errors.New("caller canceled active wait")
			var gets atomic.Int32
			policy := image.CreateImportWaitOpts{Timeout: func() *time.Duration { v := 200 * time.Millisecond; return &v }()}
			if mode == "parent deadline" {
				policy.Timeout = new(time.Duration)
			}
			cloud.Provider.HTTPClient.Transport = createImportRoundTrip(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == createImportPrefix+"images" {
					_, _ = io.Copy(io.Discard, r.Body)
					return createImportHTTP(201, io.NopCloser(strings.NewReader(createImportJSON("created", "queued")))), nil
				}
				if r.Method == http.MethodPut {
					_, _ = io.Copy(io.Discard, r.Body)
					return createImportHTTP(204, io.NopCloser(strings.NewReader(""))), nil
				}
				if r.Method == http.MethodPost {
					_, _ = io.Copy(io.Discard, r.Body)
					return createImportHTTP(202, io.NopCloser(strings.NewReader(`{}`))), nil
				}
				if gets.Add(1) == 1 {
					return createImportHTTP(200, io.NopCloser(strings.NewReader(createImportJSON("created", "uploading")))), nil
				}
				if mode == "caller canceled" {
					cancel(cause)
				}
				<-r.Context().Done()
				return nil, r.Context().Err()
			})
			v, err := image.New(cloud.Client("image", createImportPrefix)).CreateAndImport(ctx, image.CreateAndImportRequest{Name: "worker", Data: strings.NewReader("payload")}, image.WithCreateImportWait(policy))
			want := error(context.DeadlineExceeded)
			if mode == "caller canceled" {
				want = context.Canceled
			}
			if !errors.Is(err, want) || v == nil || v.Created == nil || v.Staged == nil || v.Imported == nil || v.Imported.StatusCode != 202 || v.Ready != nil || gets.Load() < 1 || gets.Load() > 2 {
				t.Fatal(v, err, gets.Load())
			}
			if mode == "caller canceled" && !errors.Is(err, cause) {
				t.Fatal("custom caller cause lost", err)
			}
		})
	}
	t.Run("disabled wait replacement performs no extra GET", func(t *testing.T) {
		cloud := testcloud.New(t)
		var gets atomic.Int32
		cloud.Provider.HTTPClient.Transport = createImportRoundTrip(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path == createImportPrefix+"images" {
				_, _ = io.Copy(io.Discard, r.Body)
				return createImportHTTP(201, io.NopCloser(strings.NewReader(createImportJSON("created", "queued")))), nil
			}
			if r.Method == http.MethodPut {
				_, _ = io.Copy(io.Discard, r.Body)
				return createImportHTTP(204, io.NopCloser(strings.NewReader(""))), nil
			}
			if r.Method == http.MethodPost {
				_, _ = io.Copy(io.Discard, r.Body)
				return createImportHTTP(202, io.NopCloser(strings.NewReader(`{}`))), nil
			}
			gets.Add(1)
			return createImportHTTP(200, io.NopCloser(strings.NewReader(createImportJSON("created", "uploading")))), nil
		})
		v, err := image.New(cloud.Client("image", createImportPrefix)).CreateAndImport(context.Background(), image.CreateAndImportRequest{Name: "worker", Data: strings.NewReader("payload")}, image.WithCreateImportWait(image.CreateImportWaitOpts{}), image.WithoutCreateImportWait())
		if err != nil || v == nil || v.Imported == nil || v.Ready != nil || gets.Load() != 1 {
			t.Fatal(v, err, gets.Load())
		}
	})
}

func TestCreateAndImportLiveProviderSourceBoundariesAndNativeIsolation(t *testing.T) {
	t.Run("metadata reauth import retry live tokens and captured route", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := cloud.Client("image", "/catalog/unused/")
		client.ResourceBase = cloud.Server.URL + createImportPrefix
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		client.Microversion = "2.7"
		cloud.Provider.SetToken("initial")
		var creates, puts, gets, imports, middleware, reauth, retries atomic.Int32
		type contextKey struct{}
		transport := cloud.Provider.HTTPClient.Transport
		cloud.Provider.HTTPClient.Transport = createImportRoundTrip(func(r *http.Request) (*http.Response, error) {
			middleware.Add(1)
			if r.Context().Value(contextKey{}) != "caller" || r.Header.Get("X-Source") != "captured" || r.Header.Get("OpenStack-API-Version") != "image 2.7" {
				t.Error(r.Header, r.Context())
			}
			return transport.RoundTrip(r)
		})
		cloud.Provider.ReauthFunc = func(context.Context) error { reauth.Add(1); cloud.Provider.SetToken("reauthenticated"); return nil }
		cloud.Provider.RetryFunc = func(_ context.Context, method, target string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			retries.Add(1)
			if method != http.MethodPost || target != cloud.Server.URL+createImportPrefix+"images/created/import" || !gophercloud.ResponseCodeIs(err, 503) {
				return err
			}
			cloud.Provider.SetToken("retry-import")
			return nil
		}
		cloud.Mux.HandleFunc("POST "+createImportPrefix+"images", func(w http.ResponseWriter, r *http.Request) {
			n := creates.Add(1)
			fields := createImportFields(t, r)
			if string(fields["name"]) != `"worker"` || r.Header.Get("X-Metadata") != "only-create" || r.Header.Get("X-Stage") != "" {
				t.Error(fields, r.Header)
			}
			if n == 1 {
				if r.Header.Get("X-Auth-Token") != "initial" {
					t.Error(r.Header)
				}
				testcloud.JSON(w, 401, `{}`)
				return
			}
			if r.Header.Get("X-Auth-Token") != "reauthenticated" {
				t.Error(r.Header)
			}
			cloud.Provider.SetToken("after-create")
			testcloud.JSON(w, 201, createImportJSON("created", "queued"))
		})
		cloud.Mux.HandleFunc("PUT "+createImportPrefix+"images/created/stage", func(w http.ResponseWriter, r *http.Request) {
			puts.Add(1)
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != "payload" || r.Header.Get("X-Auth-Token") != "after-create" || r.Header.Get("X-Stage") != "stage" || r.Header.Get("X-Metadata") != "" {
				t.Error(string(body), err, r.Header)
			}
			cloud.Provider.SetToken("after-stage")
			w.WriteHeader(204)
		})
		cloud.Mux.HandleFunc("GET "+createImportPrefix+"images/created", func(w http.ResponseWriter, r *http.Request) {
			gets.Add(1)
			if r.Header.Get("X-Auth-Token") != "after-stage" || r.Header.Get("X-Stage") != "stage" || r.Header.Get("X-Metadata") != "" {
				t.Error(r.Header)
			}
			testcloud.JSON(w, 200, createImportJSON("typed-decoy", "uploading"))
		})
		cloud.Mux.HandleFunc("POST "+createImportPrefix+"images/created/import", func(w http.ResponseWriter, r *http.Request) {
			n := imports.Add(1)
			fields := createImportFields(t, r)
			if string(fields["method"]) != `{"name":"glance-direct"}` || r.Header.Get("X-Import") != "import" || r.Header.Get("X-Metadata") != "" {
				t.Error(fields, r.Header)
			}
			want := "after-stage"
			if n > 1 {
				want = "retry-import"
			}
			if r.Header.Get("X-Auth-Token") != want {
				t.Error(r.Header)
			}
			if n == 1 {
				testcloud.JSON(w, 503, `{}`)
				return
			}
			testcloud.JSON(w, 202, `{"accepted":true}`)
		})
		v, err := image.New(client).CreateAndImport(context.WithValue(context.Background(), contextKey{}, "caller"), image.CreateAndImportRequest{Name: "worker", Data: strings.NewReader("payload")}, image.WithCreateImportMetadata(image.CreateImportMetadataOpts{Headers: map[string]string{"X-Metadata": "only-create"}}), image.WithCreateImportStage(imagedata.StageOpts{Headers: map[string]string{"X-Stage": "stage"}}), image.WithCreateImportImport(imageimport.ImportOpts{Headers: map[string]string{"X-Import": "import"}}))
		if err != nil || v == nil || v.Imported == nil || creates.Load() != 2 || puts.Load() != 1 || gets.Load() != 1 || imports.Load() != 2 || middleware.Load() != 6 || reauth.Load() != 1 || retries.Load() != 1 || client.ResourceBase != cloud.Server.URL+createImportPrefix || client.Endpoint != cloud.Server.URL+"/catalog/unused/" || client.MoreHeaders["X-Source"] != "captured" || cloud.Provider.ReauthFunc == nil || cloud.Provider.RetryFunc == nil {
			t.Fatal(v, err, creates.Load(), puts.Load(), gets.Load(), imports.Load(), middleware.Load(), reauth.Load(), retries.Load(), client)
		}
	})
	for _, boundary := range []struct{ phase, change string }{
		{"create", "type"}, {"metadata", "type"}, {"import", "type"},
		{"create", "provider"}, {"metadata", "provider"}, {"import", "provider"},
	} {
		t.Run("original source "+boundary.change+" changes after "+boundary.phase, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", createImportPrefix)
			changedPhase := boundary.phase
			var sequence createImportSequence
			cloud.Provider.HTTPClient.Transport = createImportRoundTrip(func(r *http.Request) (*http.Response, error) {
				phase, code, body := "metadata", 200, createImportJSON("decoy", "uploading")
				switch {
				case r.URL.Path == createImportPrefix+"images":
					phase, code, body = "create", 201, createImportJSON("created", "queued")
				case r.Method == http.MethodPut:
					phase, code, body = "stage", 204, "ack"
				case r.Method == http.MethodPost:
					phase, code, body = "import", 202, `{}`
				}
				sequence.add(phase)
				if r.Method != http.MethodGet {
					_, _ = io.Copy(io.Discard, r.Body)
				}
				if phase == changedPhase {
					if boundary.change == "type" {
						client.Type = "compute"
					} else {
						client.ProviderClient = &gophercloud.ProviderClient{HTTPClient: cloud.Provider.HTTPClient}
					}
				}
				return createImportHTTP(code, io.NopCloser(strings.NewReader(body))), nil
			})
			v, err := image.New(client).CreateAndImport(context.Background(), image.CreateAndImportRequest{Name: "worker", Data: strings.NewReader("payload")}, image.WithCreateImportWait(image.CreateImportWaitOpts{}))
			want := []string{"create"}
			if changedPhase != "create" {
				want = append(want, "stage", "metadata")
			}
			if changedPhase == "import" {
				want = append(want, "import")
			}
			wantError := error(resource.ErrUnsupported)
			if boundary.change == "provider" {
				wantError = resource.ErrInvalidOption
			}
			if !errors.Is(err, wantError) || v == nil || v.Created == nil || !reflect.DeepEqual(sequence.value(), want) || v.Ready != nil {
				t.Fatal(v, err, sequence.value(), want)
			}
			if changedPhase == "create" && v.Staged != nil || changedPhase == "metadata" && v.Imported != nil || changedPhase == "import" && v.Imported == nil {
				t.Fatal(v)
			}
		})
	}
	t.Run("later valid source prefix and headers do not retarget", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := cloud.Client("image", createImportPrefix)
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = createImportRoundTrip(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if !strings.HasPrefix(r.URL.Path, createImportPrefix) || r.Header.Get("X-Source") != "captured" {
				t.Error(r.URL.String(), r.Header)
			}
			if r.URL.Path == createImportPrefix+"images" {
				_, _ = io.Copy(io.Discard, r.Body)
				client.ResourceBase = cloud.Server.URL + "/caller-changed/v2/"
				client.MoreHeaders = map[string]string{"X-Source": "later"}
				return createImportHTTP(201, io.NopCloser(strings.NewReader(createImportJSON("created", "queued")))), nil
			}
			if r.Method == http.MethodPut {
				_, _ = io.Copy(io.Discard, r.Body)
				return createImportHTTP(204, io.NopCloser(strings.NewReader(""))), nil
			}
			if r.Method == http.MethodGet {
				return createImportHTTP(200, io.NopCloser(strings.NewReader(createImportJSON("decoy", "uploading")))), nil
			}
			_, _ = io.Copy(io.Discard, r.Body)
			return createImportHTTP(202, io.NopCloser(strings.NewReader(`{}`))), nil
		})
		v, err := image.New(client).CreateAndImport(context.Background(), image.CreateAndImportRequest{Name: "worker", Data: strings.NewReader("payload")})
		if err != nil || v == nil || v.Imported == nil || calls.Load() != 4 || client.ResourceBase != cloud.Server.URL+"/caller-changed/v2/" || client.MoreHeaders["X-Source"] != "later" {
			t.Fatal(v, err, calls.Load(), client)
		}
	})
	for _, code := range []int{401, 429, 498, 503, 303, 0} {
		t.Run("binary failure no replay "+fmt.Sprint(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			var creates, puts, wrong, callbacks, redirects atomic.Int32
			data := createImportData("prefix-payload")
			_, _ = data.Seek(7, io.SeekStart)
			data.seeks.Store(0)
			transportCause := errors.New("transport failed after consuming prefix")
			callbackCause := errors.New("unexpected provider callback")
			cloud.Provider.ReauthFunc = func(context.Context) error { callbacks.Add(1); return callbackCause }
			cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				callbacks.Add(1)
				return callbackCause
			}
			cloud.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				callbacks.Add(1)
				return callbackCause
			}
			cloud.Provider.MaxBackoffRetries = 3
			cloud.Provider.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects.Add(1); return nil }
			closeGate, closeDone := make(chan struct{}), make(chan struct{})
			cloud.Provider.HTTPClient.Transport = createImportRoundTrip(func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodPost && r.URL.Path == createImportPrefix+"images" {
					creates.Add(1)
					_, _ = io.Copy(io.Discard, r.Body)
					return createImportHTTP(201, io.NopCloser(strings.NewReader(createImportJSON("created", "queued")))), nil
				}
				if r.Method != http.MethodPut || r.URL.Path != createImportPrefix+"images/created/stage" {
					wrong.Add(1)
					return createImportHTTP(500, io.NopCloser(strings.NewReader(`{}`))), nil
				}
				puts.Add(1)
				prefix := make([]byte, 3)
				if n, err := io.ReadFull(r.Body, prefix); n != 3 || err != nil || string(prefix) != "pay" || r.GetBody != nil {
					t.Error("binary reader was replayed or exposed", string(prefix), n, err, r.GetBody != nil)
				}
				// Join the delayed body close before leaving the test. It must not
				// close, rewind or otherwise acquire the caller's borrowed reader.
				go func() { <-closeGate; _ = r.Body.Close(); close(closeDone) }()
				if code == 0 {
					return nil, transportCause
				}
				wire := createImportHTTP(code, io.NopCloser(strings.NewReader(`{"failed":true}`)))
				if code == 303 {
					wire.Header.Set("Location", cloud.Server.URL+"/redirected")
				}
				return wire, nil
			})
			v, err := image.New(cloud.Client("image", createImportPrefix)).CreateAndImport(context.Background(), image.CreateAndImportRequest{Name: "worker", Data: data})
			close(closeGate)
			select {
			case <-closeDone:
			case <-time.After(time.Second):
				t.Fatal("delayed transport body close did not complete")
			}
			if v == nil || v.Created == nil || v.Staged != nil || v.Imported != nil || err == nil || creates.Load() != 1 || puts.Load() != 1 || wrong.Load() != 0 || callbacks.Load() != 0 || redirects.Load() != 0 || data.seeks.Load() != 0 || data.closes.Load() != 0 || data.lengths.Load() != 0 || cloud.Provider.ReauthFunc == nil || cloud.Provider.RetryFunc == nil || cloud.Provider.RetryBackoffFunc == nil || cloud.Provider.MaxBackoffRetries != 3 {
				t.Fatal(v, err, creates.Load(), puts.Load(), wrong.Load(), callbacks.Load(), redirects.Load(), data.seeks.Load(), data.closes.Load(), data.lengths.Load())
			}
			if code == 0 && !errors.Is(err, transportCause) || code != 0 && !gophercloud.ResponseCodeIs(err, code) {
				t.Fatal("original binary failure lost", err)
			}
		})
	}
	t.Run("caller releases blocked borrowed reader after cancellation", func(t *testing.T) {
		cloud := testcloud.New(t)
		data := &createImportBlockingReader{started: make(chan struct{}), release: make(chan struct{})}
		t.Cleanup(func() { _ = data.Close() })
		var wrong atomic.Int32
		cloud.Mux.HandleFunc("POST "+createImportPrefix+"images", func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			testcloud.JSON(w, 201, createImportJSON("created", "queued"))
		})
		cloud.Mux.HandleFunc("PUT "+createImportPrefix+"images/created/stage", func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			w.WriteHeader(204)
		})
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { wrong.Add(1); w.WriteHeader(500) })
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		cause := errors.New("caller canceled blocked stage")
		type outcome struct {
			value *image.CreateImportResult
			err   error
		}
		finished := make(chan outcome, 1)
		go func() {
			v, err := image.New(cloud.Client("image", createImportPrefix)).CreateAndImport(ctx, image.CreateAndImportRequest{Name: "worker", Data: data})
			finished <- outcome{v, err}
		}()
		select {
		case <-data.started:
		case <-time.After(time.Second):
			t.Fatal("stage did not start its borrowed read")
		}
		cancel(cause)
		if data.closes.Load() != 0 {
			t.Error("SDK closed caller input after cancellation")
		}
		_ = data.Close()
		select {
		case got := <-finished:
			if got.value == nil || got.value.Created == nil || got.value.ImageID != "created" || got.value.Imported != nil || got.value.Ready != nil || !errors.Is(got.err, context.Canceled) || !errors.Is(got.err, cause) || data.closes.Load() != 1 || wrong.Load() != 0 {
				t.Fatal(got, data.closes.Load(), wrong.Load())
			}
		case <-time.After(time.Second):
			t.Fatal("workflow did not complete after caller release")
		}
	})
	t.Run("binary unauthorized is one attempt no cleanup", func(t *testing.T) {
		cloud := testcloud.New(t)
		var puts, wrong, reauth, retries, redirect atomic.Int32
		data := createImportData("payload")
		cloud.Provider.ReauthFunc = func(context.Context) error { reauth.Add(1); return nil }
		cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			retries.Add(1)
			return nil
		}
		cloud.Provider.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirect.Add(1); return nil }
		cloud.Mux.HandleFunc("POST "+createImportPrefix+"images", func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			testcloud.JSON(w, 201, createImportJSON("created", "queued"))
		})
		cloud.Mux.HandleFunc("PUT "+createImportPrefix+"images/created/stage", func(w http.ResponseWriter, r *http.Request) {
			n := puts.Add(1)
			buf := make([]byte, 3)
			_, _ = io.ReadFull(r.Body, buf)
			if n > 1 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, 401, `{"denied":true}`)
		})
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { wrong.Add(1); w.WriteHeader(500) })
		v, err := image.New(cloud.Client("image", createImportPrefix)).CreateAndImport(context.Background(), image.CreateAndImportRequest{Name: "worker", Data: data})
		if v == nil || v.Created == nil || v.Staged != nil || v.Imported != nil || !gophercloud.ResponseCodeIs(err, 401) || puts.Load() != 1 || wrong.Load() != 0 || reauth.Load() != 0 || retries.Load() != 0 || redirect.Load() != 0 || data.seeks.Load() != 0 || data.closes.Load() != 0 {
			t.Fatal(v, err, puts.Load(), wrong.Load(), reauth.Load(), retries.Load(), redirect.Load(), data.seeks.Load(), data.closes.Load())
		}
	})
	t.Run("existing Upload remains direct file route", func(t *testing.T) {
		cloud := testcloud.New(t)
		var creates, files, wrong atomic.Int32
		cloud.Mux.HandleFunc("POST "+createImportPrefix+"images", func(w http.ResponseWriter, r *http.Request) {
			creates.Add(1)
			fields := createImportFields(t, r)
			if string(fields["disk_format"]) != `"qcow2"` || string(fields["visibility"]) != `"private"` {
				t.Error(fields)
			}
			testcloud.JSON(w, 201, createImportJSON("created", "queued"))
		})
		cloud.Mux.HandleFunc("PUT "+createImportPrefix+"images/created/file", func(w http.ResponseWriter, r *http.Request) {
			files.Add(1)
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != "native" {
				t.Error(string(body), err)
			}
			w.WriteHeader(204)
		})
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { wrong.Add(1); w.WriteHeader(500) })
		v, err := image.New(cloud.Client("image", createImportPrefix)).Upload(context.Background(), image.UploadImageRequest{Name: "worker", Data: strings.NewReader("native")})
		if err != nil || v == nil || v.ID != "created" || v.Status != "queued" || creates.Load() != 1 || files.Load() != 1 || wrong.Load() != 0 {
			t.Fatal(v, err, creates.Load(), files.Load(), wrong.Load())
		}
	})
	t.Run("native Create Get Stage and Upload retain leaf wire contracts", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := cloud.Client("image", "/catalog/unused/")
		client.ResourceBase = cloud.Server.URL + createImportPrefix
		var sequence createImportSequence
		var wrong atomic.Int32
		cloud.Mux.HandleFunc("POST "+createImportPrefix+"images", func(w http.ResponseWriter, r *http.Request) {
			sequence.add("create")
			fields := createImportFields(t, r)
			if len(fields) != 1 || string(fields["name"]) != `"native"` {
				t.Error("workflow metadata defaults reached native Create", fields)
			}
			testcloud.JSON(w, 201, createImportJSON("native-id", "queued"))
		})
		cloud.Mux.HandleFunc("GET "+createImportPrefix+"images/native-id", func(w http.ResponseWriter, r *http.Request) {
			sequence.add("get")
			testcloud.JSON(w, 200, createImportJSON("native-id", "queued"))
		})
		cloud.Mux.HandleFunc("PUT "+createImportPrefix+"images/native-id/stage", func(w http.ResponseWriter, r *http.Request) {
			sequence.add("stage")
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != "native-stage" {
				t.Error(string(body), err)
			}
			w.WriteHeader(204)
		})
		cloud.Mux.HandleFunc("PUT "+createImportPrefix+"images/native-id/file", func(w http.ResponseWriter, r *http.Request) {
			sequence.add("file")
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != "native-file" {
				t.Error(string(body), err)
			}
			w.WriteHeader(204)
		})
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { wrong.Add(1); w.WriteHeader(500) })
		images := sdkimages.New(client)
		created, err := images.Create(context.Background(), sdkimages.CreateOpts{Name: "native"})
		if err != nil || created == nil || created.ID != "native-id" {
			t.Fatal(created, err)
		}
		fetched, err := images.Get(context.Background(), created.ID)
		if err != nil || fetched == nil || fetched.ID != "native-id" {
			t.Fatal(fetched, err)
		}
		data := imagedata.New(client)
		if err := data.Stage(context.Background(), fetched.ID, strings.NewReader("native-stage")); err != nil {
			t.Fatal(err)
		}
		if err := data.Upload(context.Background(), fetched.ID, strings.NewReader("native-file")); err != nil {
			t.Fatal(err)
		}
		if wrong.Load() != 0 || !reflect.DeepEqual(sequence.value(), []string{"create", "get", "stage", "file"}) {
			t.Fatal(wrong.Load(), sequence.value())
		}
	})
}
