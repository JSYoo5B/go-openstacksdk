package image

import (
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

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/image/v2/imagedata"
	"gophercloudsdk/image/v2/imageimport"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

type createImportReader struct {
	data   io.Reader
	reads  atomic.Int32
	closed atomic.Int32
}

func (value *createImportReader) Read(buffer []byte) (int, error) {
	value.reads.Add(1)
	return value.data.Read(buffer)
}
func (value *createImportReader) Close() error { value.closed.Add(1); return nil }

func TestCreateImportPreflightValidatesEveryPhaseBeforeCreationOrReading(t *testing.T) {
	negative, badInterval, badTimeout := int64(-1), time.Duration(0), -time.Second
	for _, test := range []struct {
		name    string
		options []CreateImportOption
	}{
		{"nil option", []CreateImportOption{nil}},
		{"metadata core property", []CreateImportOption{WithCreateImportMetadata(CreateImportMetadataOpts{Properties: map[string]any{"name": "override"}})}},
		{"metadata invalid header", []CreateImportOption{WithCreateImportMetadata(CreateImportMetadataOpts{Headers: map[string]string{"Authorization": "override"}})}},
		{"metadata unserializable", []CreateImportOption{WithCreateImportMetadata(CreateImportMetadataOpts{Properties: map[string]any{"vendor": func() {}}})}},
		{"stage negative size", []CreateImportOption{WithCreateImportStage(imagedata.StageOpts{Size: &negative})}},
		{"stage callback", []CreateImportOption{WithCreateImportStageOptions(imagedata.WithStageHeader("Accept", "bad"))}},
		{"import callback", []CreateImportOption{WithCreateImportImportOptions(imageimport.WithImportAllStores(true), imageimport.WithImportStore("conflict"))}},
		{"import missing remote", []CreateImportOption{WithCreateImportImport(imageimport.ImportOpts{Method: imageimport.GlanceDownloadMethod, RemoteRegion: "region"})}},
		{"unsupported copy", []CreateImportOption{WithCreateImportImport(imageimport.ImportOpts{Method: imageimport.CopyImageMethod})}},
		{"unsupported extension method", []CreateImportOption{WithCreateImportImport(imageimport.ImportOpts{Method: "custom"})}},
		{"wait interval", []CreateImportOption{WithCreateImportWait(CreateImportWaitOpts{PollInterval: &badInterval})}},
		{"wait timeout", []CreateImportOption{WithCreateImportWait(CreateImportWaitOpts{Timeout: &badTimeout})}},
		{"wait states", []CreateImportOption{WithCreateImportWait(CreateImportWaitOpts{FailureStates: []string{" "}})}},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { requests.Add(1); http.Error(w, "unexpected", 500) })
			reader := &createImportReader{data: strings.NewReader("data")}
			result, err := New(cloud.Client("image", "/v2")).CreateAndImport(context.Background(), CreateAndImportRequest{Name: "image", Data: reader}, test.options...)
			if result != nil || err == nil || requests.Load() != 0 || reader.reads.Load() != 0 || reader.closed.Load() != 0 {
				t.Fatalf("result=%v err=%v requests=%d reads=%d closes=%d", result, err, requests.Load(), reader.reads.Load(), reader.closed.Load())
			}
		})
	}
	cloud := testcloud.New(t)
	client := cloud.Client("image", "/v2")
	client.ResourceBase = "https://foreign.test/v2/"
	var callbacks int
	_, err := New(client).CreateAndImport(context.Background(), CreateAndImportRequest{Name: "image", Data: strings.NewReader("data")}, func(*CreateImportOpts) error { callbacks++; return nil })
	if !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
		t.Fatalf("source preflight err=%v callbacks=%d", err, callbacks)
	}
}

func TestCreateImportFixedCanonicalRouteAndIndependentActualEvidence(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("image", "/v2")
	client.MoreHeaders = map[string]string{"X-Source": "before"}
	service := New(client)
	var sequence []string
	var retained *CreateImportOpts
	var metadata map[string]any
	cloud.Mux.HandleFunc("POST /v2/images", func(w http.ResponseWriter, r *http.Request) {
		sequence = append(sequence, "create")
		decoder := json.NewDecoder(r.Body)
		decoder.UseNumber()
		if err := decoder.Decode(&metadata); err != nil {
			t.Error(err)
		}
		if r.Header.Get("X-Metadata") != "metadata" || r.Header.Get("X-Stage") != "" || r.Header.Get("X-Import") != "" {
			t.Errorf("creation headers=%v", r.Header)
		}
		retained.Metadata.Properties["vendor"] = nil
		retained.Stage.Headers["X-Stage"] = "after"
		retained.Import.Stores[0] = "after"
		client.MoreHeaders["X-Source"] = "after"
		w.Header().Set("OpenStack-image-import-methods", "glance-direct,web-download")
		testcloud.JSON(w, 201, `{"id":"fixed","status":"queued","disk_format":"qcow2","container_format":"bare","vendor":9007199254740993}`)
	})
	cloud.Mux.HandleFunc("PUT /v2/images/fixed/stage", func(w http.ResponseWriter, r *http.Request) {
		sequence = append(sequence, "stage")
		data, err := io.ReadAll(r.Body)
		if err != nil || string(data) != "data" || r.Header.Get("X-Stage") != "stage" || r.Header.Get("X-Source") != "before" || r.Header.Get("X-Metadata") != "" {
			t.Errorf("data=%q err=%v headers=%v", data, err, r.Header)
		}
		w.Header().Set("X-Stage-Ack", "actual")
		w.WriteHeader(204)
	})
	cloud.Mux.HandleFunc("GET /v2/images/fixed", func(w http.ResponseWriter, r *http.Request) {
		sequence = append(sequence, "fetch")
		if r.Header.Get("X-Stage") != "stage" {
			t.Errorf("stage GET headers=%v", r.Header)
		}
		testcloud.JSON(w, 200, `{"id":"typed-decoy","status":"uploading","disk_format":"raw","container_format":"bare"}`)
	})
	cloud.Mux.HandleFunc("POST /v2/images/fixed/import", func(w http.ResponseWriter, r *http.Request) {
		sequence = append(sequence, "import")
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(body, map[string]any{"method": map[string]any{"name": "glance-direct"}, "stores": []any{"fast"}}) || r.Header.Get("X-Import") != "import" || r.Header.Get("X-Stage") != "" {
			t.Errorf("body=%#v headers=%v", body, r.Header)
		}
		w.Header().Set("X-Import-Ack", "actual")
		w.WriteHeader(202)
	})
	reader := &createImportReader{data: strings.NewReader("data")}
	callbacks := 0
	result, err := service.CreateAndImport(context.Background(), CreateAndImportRequest{Name: "authoritative", Data: reader}, func(value *CreateImportOpts) error {
		callbacks++
		retained = value
		value.Metadata = CreateImportMetadataOpts{Properties: map[string]any{"vendor": json.Number("9007199254740993")}, Headers: map[string]string{"X-Metadata": "metadata"}}
		value.Stage = imagedata.StageOpts{Headers: map[string]string{"X-Stage": "stage"}}
		value.Import = imageimport.ImportOpts{Stores: []string{"fast"}, Headers: map[string]string{"X-Import": "import"}}
		return nil
	})
	if err != nil || result == nil || callbacks != 1 || result.ImageID != "fixed" || result.Created.StatusCode != 201 || result.Staged.Acknowledgement.StatusCode != 204 || result.Staged.Image.ID != "typed-decoy" || result.Imported.StatusCode != 202 || result.Ready != nil || reader.closed.Load() != 0 {
		t.Fatalf("result=%+v err=%v callbacks=%d closes=%d", result, err, callbacks, reader.closed.Load())
	}
	if !reflect.DeepEqual(sequence, []string{"create", "stage", "fetch", "import"}) || metadata["name"] != "authoritative" || metadata["disk_format"] != "qcow2" || metadata["visibility"] != "private" || metadata["vendor"] != json.Number("9007199254740993") {
		t.Fatalf("sequence=%v metadata=%#v", sequence, metadata)
	}
	if !reflect.DeepEqual(result.Created.Image.OpenStackImageImportMethods, []string{"glance-direct", "web-download"}) {
		t.Fatalf("creation capability projection=%v", result.Created.Image.OpenStackImageImportMethods)
	}
	result.Staged.Header.Set("X-Stage-Ack", "changed")
	if result.Staged.Acknowledgement.Header.Get("X-Stage-Ack") != "actual" || result.Imported.Header.Get("X-Import-Ack") != "actual" {
		t.Fatal("phase evidence aliases")
	}
}

func TestCreateImportCanonicalFailureRetainsAcceptedCreationAndStage(t *testing.T) {
	for _, test := range []struct {
		name, created, staged string
		wantRequests          int
		wantStage             bool
	}{
		{"bad creation native", `{"id":"fixed","status":"queued","min_ram":"bad"}`, "", 1, false},
		{"missing canonical id", `{"ID":"decoy","status":"queued"}`, "", 1, false},
		{"wrong canonical state", `{"id":"fixed","status":"active"}`, "", 1, false},
		{"poststage missing canonical format", `{"id":"fixed","status":"queued"}`, `{"id":"decoy","status":"uploading","DiskFormat":"raw","container_format":"bare"}`, 3, true},
		{"poststage null format", `{"id":"fixed","status":"queued"}`, `{"id":"decoy","status":"uploading","disk_format":null,"container_format":"bare"}`, 3, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests atomic.Int32
			cloud.Mux.HandleFunc("POST /v2/images", func(w http.ResponseWriter, r *http.Request) { requests.Add(1); testcloud.JSON(w, 201, test.created) })
			cloud.Mux.HandleFunc("PUT /v2/images/fixed/stage", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				_, _ = io.Copy(io.Discard, r.Body)
				w.WriteHeader(204)
			})
			cloud.Mux.HandleFunc("GET /v2/images/fixed", func(w http.ResponseWriter, r *http.Request) { requests.Add(1); testcloud.JSON(w, 200, test.staged) })
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				t.Errorf("unexpected %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			reader := &createImportReader{data: strings.NewReader("data")}
			result, err := New(cloud.Client("image", "/v2")).CreateAndImport(context.Background(), CreateAndImportRequest{Name: "image", Data: reader})
			var responseErr *resource.ResponseError
			if err == nil || !errors.As(err, &responseErr) || result == nil || result.Created == nil || result.Created.StatusCode != 201 || string(result.Created.Body) != test.created || (result.Staged != nil) != test.wantStage || result.Imported != nil || requests.Load() != int32(test.wantRequests) {
				t.Fatalf("result=%+v err=%v requests=%d", result, err, requests.Load())
			}
			if !test.wantStage && reader.reads.Load() != 0 {
				t.Fatal("creation rejection read data")
			}
		})
	}
}

type createImportTransport func(*http.Request) (*http.Response, error)

func (value createImportTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return value(request)
}

func TestCreateImportOptionalWaitUsesPreparedDefaultBudgetAndFixedID(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("image", "/v2")
	var gets atomic.Int32
	var observedBudget time.Duration
	transport := client.ProviderClient.HTTPClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	client.ProviderClient.HTTPClient.Transport = createImportTransport(func(request *http.Request) (*http.Response, error) {
		if request.Method == http.MethodGet && gets.Load() == 1 {
			deadline, ok := request.Context().Deadline()
			if !ok {
				t.Error("default wait did not constrain GET")
			} else {
				observedBudget = time.Until(deadline)
			}
		}
		return transport.RoundTrip(request)
	})
	cloud.Mux.HandleFunc("POST /v2/images", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 201, `{"id":"fixed","status":"queued"}`)
	})
	cloud.Mux.HandleFunc("PUT /v2/images/fixed/stage", func(w http.ResponseWriter, r *http.Request) { _, _ = io.Copy(io.Discard, r.Body); w.WriteHeader(204) })
	cloud.Mux.HandleFunc("POST /v2/images/fixed/import", func(w http.ResponseWriter, r *http.Request) {
		client.ProviderClient.SetToken("live-token")
		w.WriteHeader(202)
	})
	cloud.Mux.HandleFunc("GET /v2/images/fixed", func(w http.ResponseWriter, r *http.Request) {
		if gets.Add(1) == 1 {
			testcloud.JSON(w, 200, `{"id":"stage-decoy","status":"uploading","disk_format":"qcow2","container_format":"bare"}`)
			return
		}
		if r.Header.Get("X-Auth-Token") != "live-token" {
			t.Errorf("wait token=%v", r.Header)
		}
		testcloud.JSON(w, 200, `{"id":"wait-decoy","status":"active"}`)
	})
	result, err := New(client).CreateAndImport(context.Background(), CreateAndImportRequest{Name: "image", Data: strings.NewReader("data")}, WithCreateImportWait(CreateImportWaitOpts{}))
	if err != nil || result == nil || result.Ready == nil || result.Ready.ID != "wait-decoy" || result.ImageID != "fixed" || gets.Load() != 2 || observedBudget < 4*time.Minute || observedBudget > 5*time.Minute {
		t.Fatalf("result=%+v err=%v gets=%d budget=%v", result, err, gets.Load(), observedBudget)
	}
}

var _ io.ReadCloser = (*createImportReader)(nil)
var _ = gophercloud.ErrUnexpectedResponseCode{}
