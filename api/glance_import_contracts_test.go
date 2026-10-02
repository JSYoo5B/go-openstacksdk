package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/image/v2/imageimport"
	"gophercloudsdk/image/v2/images"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

// Pinned Proxy.import_image/Image.import_image use a supplied image's formats
// and return the import POST response. Fresh Ref lookup and header-only Store
// are documented Go policies; HTTP202 acknowledges asynchronous submission.
const glanceImportPrefix = "/reverse/glance/v2/"

type glanceImportRoundTrip func(*http.Request) (*http.Response, error)

func (f glanceImportRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func glanceImportHTTP(code int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}, "X-Evidence": {"actual"}}, Body: body}
}

func glanceImportKnown() *images.Image {
	return &images.Image{ID: "selected", ContainerFormat: "bare", DiskFormat: "qcow2", Status: images.ImageStatusQueued}
}

func glanceImportJSON(id string) string {
	return fmt.Sprintf(`{"id":%q,"name":"worker","status":"queued","container_format":"bare","disk_format":"qcow2","created_at":"2024-01-02T03:04:05Z","vendor":{"exact":9007199254740993}}`, id)
}

func glanceImportFields(t *testing.T, r *http.Request) map[string]json.RawMessage {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatal(err, string(body))
	}
	return fields
}

func TestGlanceImportFreshAndKnownAcknowledgements(t *testing.T) {
	for _, known := range []bool{false, true} {
		t.Run(fmt.Sprint("known=", known), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", "/catalog/unused/")
			client.ResourceBase = cloud.Server.URL + glanceImportPrefix
			var gets, posts, wrong atomic.Int32
			cloud.Mux.HandleFunc(glanceImportPrefix+"images/selected", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if r.Method != http.MethodGet {
					t.Error(r.Method)
				}
				testcloud.JSON(w, 200, glanceImportJSON("incidental-response-ID"))
			})
			cloud.Mux.HandleFunc(glanceImportPrefix+"images/selected/import", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				if r.Method != http.MethodPost || r.URL.RawQuery != "" {
					t.Error(r.Method, r.URL.String())
				}
				fields := glanceImportFields(t, r)
				if len(fields) != 1 || string(fields["method"]) != `{"name":"glance-direct"}` {
					t.Error(fields)
				}
				w.Header().Set("X-Ack", "actual")
				w.Header().Set("Location", "https://foreign.invalid/tasks/do-not-follow")
				testcloud.JSON(w, 202, `{"ack":{"precise":9007199254740993},"state":"server-extension"}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { wrong.Add(1); w.WriteHeader(500) })
			a := imageimport.New(client)
			var v *imageimport.ImportResult
			var err error
			if known {
				v, err = a.ImportKnownImage(context.Background(), glanceImportKnown())
			} else {
				v, err = a.ImportImage(context.Background(), resource.ID("selected"))
			}
			wantGets := int32(1)
			if known {
				wantGets = 0
			}
			if err != nil || v == nil || v.ImageID != "selected" || v.StatusCode != 202 || v.Header.Get("X-Ack") != "actual" || !bytes.Contains(v.Body, []byte("9007199254740993")) || posts.Load() != 1 || gets.Load() != wantGets || wrong.Load() != 0 {
				t.Fatal(v, err, gets.Load(), posts.Load(), wrong.Load())
			}
			v.Body[0] = '!'
			v.Header.Set("X-Ack", "caller changed")
			if client.ResourceBase != cloud.Server.URL+glanceImportPrefix || client.Endpoint != cloud.Server.URL+"/catalog/unused/" {
				t.Fatal("shared client was modified", client)
			}
		})
	}
}

func TestGlanceImportMethodsAndRootStoreControls(t *testing.T) {
	for _, tc := range []struct {
		name                string
		opts                []imageimport.ImportOption
		method, root, store string
	}{
		{"default", nil, `{"name":"glance-direct"}`, `{}`, ""},
		{"web", []imageimport.ImportOption{imageimport.WithImportMethod(imageimport.WebDownloadMethod), imageimport.WithImportURI("https://remote.invalid/image?q=%2F")}, `{"name":"web-download","uri":"https://remote.invalid/image?q=%2F"}`, `{}`, ""},
		{"remote default interface", []imageimport.ImportOption{imageimport.WithImportMethod(imageimport.GlanceDownloadMethod), imageimport.WithImportRemoteRegion("RegionTwo"), imageimport.WithImportRemoteImageID("remote-id")}, `{"glance_image_id":"remote-id","glance_region":"RegionTwo","name":"glance-download"}`, `{}`, ""},
		{"remote interface", []imageimport.ImportOption{imageimport.WithImportMethod(imageimport.GlanceDownloadMethod), imageimport.WithImportRemoteRegion("RegionTwo"), imageimport.WithImportRemoteImageID("remote-id"), imageimport.WithImportRemoteServiceInterface("internal")}, `{"glance_image_id":"remote-id","glance_region":"RegionTwo","glance_service_interface":"internal","name":"glance-download"}`, `{}`, ""},
		{"copy", []imageimport.ImportOption{imageimport.WithImportMethod(imageimport.CopyImageMethod)}, `{"name":"copy-image"}`, `{}`, ""},
		{"vendor server method", []imageimport.ImportOption{imageimport.WithImportMethod("vendor-method")}, `{"name":"vendor-method"}`, `{}`, ""},
		{"singular header only", []imageimport.ImportOption{imageimport.WithImportStore("store-a"), imageimport.WithImportAllStores(false), imageimport.WithImportAllStoresMustSucceed(false)}, `{"name":"glance-direct"}`, `{"all_stores":false,"all_stores_must_succeed":false}`, "store-a"},
		{"plural root only", []imageimport.ImportOption{imageimport.WithImportStores("store-a", "store-b")}, `{"name":"glance-direct"}`, `{"stores":["store-a","store-b"]}`, ""},
		{"all stores", []imageimport.ImportOption{imageimport.WithImportAllStores(true)}, `{"name":"glance-direct"}`, `{"all_stores":true}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(glanceImportPrefix+"images/selected/import", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				fields := glanceImportFields(t, r)
				if string(fields["method"]) != tc.method || r.Header.Get("X-Image-Meta-Store") != tc.store {
					t.Error(fields, r.Header)
				}
				delete(fields, "method")
				root, _ := json.Marshal(fields)
				if string(root) != tc.root {
					t.Error("store controls belong at root", string(root), tc.root)
				}
				if tc.store == "" {
					if _, exists := r.Header["X-Image-Meta-Store"]; exists {
						t.Error("plural/default method set legacy header", r.Header)
					}
				}
				w.WriteHeader(202)
			})
			if v, err := imageimport.New(cloud.Client("image", glanceImportPrefix)).ImportKnownImage(context.Background(), glanceImportKnown(), tc.opts...); err != nil || v == nil || v.StatusCode != 202 || len(v.Body) != 0 || calls.Load() != 1 {
				t.Fatal(v, err, calls.Load())
			}
		})
	}
	for _, opts := range [][]imageimport.ImportOption{
		{imageimport.WithImportMethod(imageimport.WebDownloadMethod)},
		{imageimport.WithImportURI("https://remote.invalid/image")},
		{imageimport.WithImportMethod(imageimport.GlanceDownloadMethod), imageimport.WithImportRemoteRegion("RegionTwo")},
		{imageimport.WithImportMethod(imageimport.GlanceDownloadMethod), imageimport.WithImportRemoteImageID("remote-id")},
		{imageimport.WithImportRemoteRegion("RegionTwo"), imageimport.WithImportRemoteImageID("remote-id")},
		{imageimport.WithImportStore("a"), imageimport.WithImportStores("b")},
		{imageimport.WithImportAllStores(true), imageimport.WithImportStore("a")},
		{imageimport.WithImportAllStores(true), imageimport.WithImportStores("a")},
	} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = glanceImportRoundTrip(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			return glanceImportHTTP(500, io.NopCloser(strings.NewReader(`{}`))), nil
		})
		if v, err := imageimport.New(cloud.Client("image", glanceImportPrefix)).ImportImage(context.Background(), resource.ID("selected"), opts...); v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal("invalid selectors reached fresh GET", v, err, calls.Load())
		}
	}
}

func TestGlanceImportFreshFormatsAndAllPageNameBinding(t *testing.T) {
	for _, body := range []string{
		`{`, `null`, `[]`, `{"container_format":"bare"}`, `{"disk_format":"raw","container_format":null}`, `{"disk_format":"","container_format":"bare"}`,
		`{"disk_format":"raw","container_format":7}`, `{"CONTAINER_FORMAT":"bare","DISK_FORMAT":"raw"}`,
		`{"container_format":"bare","disk_format":"raw","min_ram":"bad"}`, `{"container_format":"bare","disk_format":"raw","created_at":"not-a-date"}`,
		`{"container_format":"bare","disk_format":"raw","tags":[{}]}`, `{"container_format":"bare","disk_format":"raw","size":"not-a-size"}`,
		string([]byte{'{', '"', 'c', 'o', 'n', 't', 'a', 'i', 'n', 'e', 'r', '_', 'f', 'o', 'r', 'm', 'a', 't', '"', ':', '"', 0xff, '"', '}'}),
	} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, posts atomic.Int32
			cloud.Mux.HandleFunc(glanceImportPrefix+"images/selected", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				w.Header().Set("X-GET", "actual")
				testcloud.JSON(w, 200, body)
			})
			cloud.Mux.HandleFunc(glanceImportPrefix+"images/selected/import", func(w http.ResponseWriter, r *http.Request) { posts.Add(1); w.WriteHeader(202) })
			v, err := imageimport.New(cloud.Client("image", glanceImportPrefix)).ImportImage(context.Background(), resource.ID("selected"))
			var response *resource.ResponseError
			if v != nil || !errors.As(err, &response) || response.StatusCode != 200 || string(response.Body) != body || response.Header.Get("X-GET") != "actual" || gets.Load() != 1 || posts.Load() != 0 {
				t.Fatal(v, err, response, gets.Load(), posts.Load())
			}
		})
	}
	t.Run("exact Name all pages then canonical GET", func(t *testing.T) {
		cloud := testcloud.New(t)
		var lists, gets, posts atomic.Int32
		cloud.Mux.HandleFunc(glanceImportPrefix+"images", func(w http.ResponseWriter, r *http.Request) {
			lists.Add(1)
			if r.URL.Query().Get("name") != "worker" || r.Header.Get("X-Trace") != "prepared" {
				t.Error(r.URL.String(), r.Header)
			}
			if r.URL.Query().Get("marker") == "" {
				testcloud.JSON(w, 200, `{"images":[{"id":"case-decoy","name":"Worker"}],"next":"/v2/images?marker=second&name=worker"}`)
			} else {
				testcloud.JSON(w, 200, `{"images":[{"id":"selected","name":"worker","container_format":"invalid-cached-format"}]}`)
			}
		})
		cloud.Mux.HandleFunc(glanceImportPrefix+"images/selected", func(w http.ResponseWriter, r *http.Request) {
			gets.Add(1)
			testcloud.JSON(w, 200, glanceImportJSON("response-must-not-retarget"))
		})
		cloud.Mux.HandleFunc(glanceImportPrefix+"images/selected/import", func(w http.ResponseWriter, r *http.Request) { posts.Add(1); testcloud.JSON(w, 202, `{"ack":true}`) })
		v, err := imageimport.New(cloud.Client("image", glanceImportPrefix)).ImportImage(context.Background(), resource.Name("worker"), imageimport.WithImportHeader("X-Trace", "prepared"))
		if err != nil || v == nil || v.ImageID != "selected" || lists.Load() != 2 || gets.Load() != 1 || posts.Load() != 1 {
			t.Fatal(v, err, lists.Load(), gets.Load(), posts.Load())
		}
	})
	for _, mode := range []string{"missing", "duplicate", "late error"} {
		t.Run("Name "+mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(glanceImportPrefix+"images", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if mode == "missing" {
					testcloud.JSON(w, 200, `{"images":[]}`)
					return
				}
				if r.URL.Query().Get("marker") == "" {
					testcloud.JSON(w, 200, `{"images":[{"id":"selected","name":"worker"}],"next":"/v2/images?marker=second&name=worker"}`)
					return
				}
				if mode == "duplicate" {
					testcloud.JSON(w, 200, `{"images":[{"id":"other","name":"worker"}]}`)
				} else {
					testcloud.JSON(w, 403, `{"denied":true}`)
				}
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(100); w.WriteHeader(500) })
			v, err := imageimport.New(cloud.Client("image", glanceImportPrefix)).ImportImage(context.Background(), resource.Name("worker"))
			if v != nil || err == nil || (mode == "missing" && (!errors.Is(err, resource.ErrNotFound) || calls.Load() != 1)) || (mode == "duplicate" && (!errors.Is(err, resource.ErrAmbiguous) || calls.Load() != 2)) || (mode == "late error" && (!gophercloud.ResponseCodeIs(err, 403) || calls.Load() != 2)) {
				t.Fatal("Name failure reached GET/POST", v, err, calls.Load())
			}
		})
	}
}

func TestGlanceImportOptionSnapshotsReplacementAndReuse(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc(glanceImportPrefix+"images/selected/import", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		fields := glanceImportFields(t, r)
		if string(fields["stores"]) != `["a","a","b"]` || string(fields["all_stores"]) != "false" || string(fields["all_stores_must_succeed"]) != "false" || r.Header.Get("X-Trace") != "owned" || !bytes.Contains(fields["root_vendor"], []byte("9007199254740993")) || !bytes.Contains(fields["method"], []byte(`"method_vendor":[null,{"nested":"owned"}]`)) {
			t.Error("mutable options reached wire", fields, r.Header)
		}
		w.Header().Set("X-Ack", "actual")
		testcloud.JSON(w, 202, `{"ack":9007199254740993}`)
	})
	all, must := false, false
	stores := []string{"a", "a", "b"}
	headers := map[string]string{"X-Trace": "owned"}
	root := map[string]any{"root_vendor": json.RawMessage(`{"precise":9007199254740993}`)}
	nested := map[string]any{"nested": "owned"}
	method := map[string]any{"method_vendor": []any{nil, nested}}
	option := imageimport.WithImportOpts(imageimport.ImportOpts{Stores: stores, AllStores: &all, AllStoresMustSucceed: &must, Headers: headers, Fields: root, MethodFields: method})
	stores[0] = "changed"
	all, must = true, true
	headers["X-Trace"] = "changed"
	root["root_vendor"] = false
	nested["nested"] = "changed"
	a := imageimport.New(cloud.Client("image", glanceImportPrefix))
	var wg sync.WaitGroup
	errorsOut := make(chan error, 5)
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := a.ImportKnownImage(context.Background(), glanceImportKnown(), imageimport.WithImportMethod(imageimport.WebDownloadMethod), option)
			if err != nil || v == nil {
				errorsOut <- fmt.Errorf("result=%v error=%v", v, err)
				return
			}
			v.Body[0] = '!'
			v.Header.Set("X-Ack", "changed")
		}()
	}
	wg.Wait()
	close(errorsOut)
	for err := range errorsOut {
		t.Error(err)
	}
	v, err := a.ImportKnownImage(context.Background(), glanceImportKnown(), option)
	if err != nil || v == nil || v.Header.Get("X-Ack") != "actual" || v.Body[0] != '{' || calls.Load() != 6 {
		t.Fatal(v, err, calls.Load())
	}
	t.Run("seed and custom config are copied before HTTP", func(t *testing.T) {
		cloud := testcloud.New(t)
		seed := glanceImportKnown()
		var retained *imageimport.ImportOpts
		var calls atomic.Int32
		option := imageimport.ImportOption(func(o *imageimport.ImportOpts) error {
			seed.ID = "other"
			seed.ContainerFormat = ""
			seed.DiskFormat = ""
			o.Store = new(string)
			*o.Store = "selected-store"
			o.Headers = map[string]string{"X-Trace": "captured"}
			o.Fields = map[string]any{"vendor": []any{"captured"}}
			retained = o
			return nil
		})
		cloud.Provider.HTTPClient.Transport = glanceImportRoundTrip(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			retained.Headers["X-Trace"] = "late"
			*retained.Store = "late"
			retained.Fields["vendor"] = false
			if r.Method != http.MethodPost || r.URL.Path != glanceImportPrefix+"images/selected/import" || r.Header.Get("X-Trace") != "captured" || r.Header.Get("X-Image-Meta-Store") != "selected-store" {
				t.Error(r.Method, r.URL.String(), r.Header)
			}
			fields := glanceImportFields(t, r)
			if string(fields["vendor"]) != `["captured"]` {
				t.Error(fields)
			}
			return glanceImportHTTP(202, io.NopCloser(strings.NewReader(`{}`))), nil
		})
		if v, err := imageimport.New(cloud.Client("image", glanceImportPrefix)).ImportKnownImage(context.Background(), seed, option); err != nil || v == nil || v.ImageID != "selected" || calls.Load() != 1 {
			t.Fatal(v, err, calls.Load())
		}
	})
	t.Run("bulk replacement clears old controls", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc(glanceImportPrefix+"images/selected/import", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			fields := glanceImportFields(t, r)
			if len(fields) != 1 || string(fields["method"]) != `{"name":"glance-direct"}` || r.Header.Get("X-Image-Meta-Store") != "" {
				t.Error(fields, r.Header)
			}
			w.WriteHeader(202)
		})
		if v, err := imageimport.New(cloud.Client("image", glanceImportPrefix)).ImportKnownImage(context.Background(), glanceImportKnown(), imageimport.WithImportStore("old"), imageimport.WithImportURI("old"), imageimport.WithImportField("vendor", true), imageimport.WithImportOpts(imageimport.ImportOpts{})); err != nil || v == nil || calls.Load() != 1 {
			t.Fatal(v, err, calls.Load())
		}
	})
}

func TestGlanceImportExtensionsAndPreflightAreSDKOwned(t *testing.T) {
	t.Run("owned root and method extension snapshots", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		root := json.RawMessage(`{"large":9007199254740993,"nested":[null,{}]}`)
		method := []any{json.Number("1e+123"), map[string]any{"value": "owned"}}
		opts := []imageimport.ImportOption{imageimport.WithImportField("vendor", false), imageimport.WithImportField("vendor", root), imageimport.WithImportMethodField("vendor_method", method)}
		root[0] = '!'
		method[1].(map[string]any)["value"] = "changed"
		cloud.Mux.HandleFunc(glanceImportPrefix+"images/selected/import", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			fields := glanceImportFields(t, r)
			if string(fields["vendor"]) != `{"large":9007199254740993,"nested":[null,{}]}` || !bytes.Contains(fields["method"], []byte(`"vendor_method":[1e+123,{"value":"owned"}]`)) {
				t.Error(fields)
			}
			w.WriteHeader(202)
		})
		if v, err := imageimport.New(cloud.Client("image", glanceImportPrefix)).ImportKnownImage(context.Background(), glanceImportKnown(), opts...); err != nil || v == nil || calls.Load() != 1 {
			t.Fatal(v, err, calls.Load())
		}
	})
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Provider.HTTPClient.Transport = glanceImportRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		return glanceImportHTTP(500, io.NopCloser(strings.NewReader(`{}`))), nil
	})
	client := cloud.Client("image", glanceImportPrefix)
	a := imageimport.New(client)
	for _, option := range []imageimport.ImportOption{
		nil, imageimport.WithImportStore(""), imageimport.WithImportStores("valid", ""), imageimport.WithImportHeader("X-Auth-Token", "forged"), imageimport.WithImportHeader("X-Image-Meta-Store", "forged"), imageimport.WithImportHeader("OpenStack-API-Version", "image 2.8"), imageimport.WithImportHeaders(map[string]string{"X-Trace": "a", "x-trace": "b"}),
		imageimport.WithImportField("method", true), imageimport.WithImportField("ALL-STORES", true), imageimport.WithImportField("stores", []string{"a"}), imageimport.WithImportMethodField("NAME", "copy-image"), imageimport.WithImportMethodField("glance-region", "remote"), imageimport.WithImportMethodField("URI", "https://remote.invalid"), imageimport.WithImportField("vendor", json.RawMessage(`{`)), imageimport.WithImportField("vendor", make(chan int)),
	} {
		if v, err := a.ImportImage(context.Background(), resource.ID("selected"), option); v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal("closed option reached GET", v, err, calls.Load())
		}
	}
	for _, ref := range []resource.Ref{resource.Ref{}, resource.ID("bad/id"), resource.ID("%2F"), resource.ID(".."), resource.ID("bad:scheme"), resource.ID("white\u00a0space")} {
		if v, err := a.ImportImage(context.Background(), ref); v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal(ref, v, err, calls.Load())
		}
	}
	for _, known := range []*images.Image{nil, {ID: "selected", ContainerFormat: "bare"}, {ID: "selected", DiskFormat: "qcow2"}, {ID: "bad/id", ContainerFormat: "bare", DiskFormat: "raw"}} {
		if v, err := a.ImportKnownImage(context.Background(), known); v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal(known, v, err, calls.Load())
		}
	}
	if v, err := a.ImportImage(nil, resource.ID("selected")); v != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(v, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if v, err := a.ImportImage(ctx, resource.ID("selected")); v != nil || !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatal(v, err, calls.Load())
	}
	var nilAPI *imageimport.API
	for _, api := range []*imageimport.API{nilAPI, imageimport.New(nil), imageimport.New(&gophercloud.ServiceClient{Type: "image"})} {
		if v, err := api.ImportImage(context.Background(), resource.ID("selected")); v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal(v, err, calls.Load())
		}
	}
	client.Type = "compute"
	if v, err := a.ImportImage(context.Background(), resource.ID("selected")); v != nil || !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 0 {
		t.Fatal(v, err, calls.Load())
	}
	client.Type = "image"
	client.MoreHeaders = map[string]string{"X-Image-Meta-Store": "source-selected"}
	if v, err := a.ImportImage(context.Background(), resource.ID("selected"), imageimport.WithImportStores("a")); v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
		t.Fatal("source Store conflict reached GET", v, err, calls.Load())
	}
	client.MoreHeaders = map[string]string{"X-Trace": "a", "x-trace": "b"}
	if v, err := a.ImportImage(context.Background(), resource.ID("selected")); v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
		t.Fatal(v, err, calls.Load())
	}
}

func TestGlanceImportHTTPFailuresAndAcceptedEvidence(t *testing.T) {
	for _, phase := range []string{"GET", "POST"} {
		for _, code := range []int{200, 201, 204, 403, 404} {
			if phase == "GET" && code == 200 {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", phase, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets, posts atomic.Int32
				cloud.Mux.HandleFunc(glanceImportPrefix+"images/selected", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					if phase == "GET" {
						testcloud.JSON(w, code, `{"failure":true}`)
					} else {
						testcloud.JSON(w, 200, glanceImportJSON("selected"))
					}
				})
				cloud.Mux.HandleFunc(glanceImportPrefix+"images/selected/import", func(w http.ResponseWriter, r *http.Request) {
					posts.Add(1)
					testcloud.JSON(w, code, `{"failure":true}`)
				})
				v, err := imageimport.New(cloud.Client("image", glanceImportPrefix)).ImportImage(context.Background(), resource.ID("selected"))
				var native gophercloud.ErrUnexpectedResponseCode
				wantPosts := int32(1)
				if phase == "GET" {
					wantPosts = 0
				}
				if v != nil || !errors.As(err, &native) || native.Actual != code || gets.Load() != 1 || posts.Load() != wantPosts {
					t.Fatal(v, err, gets.Load(), posts.Load())
				}
			})
		}
	}
	t.Run("202 body is actual acknowledgement not an Image envelope", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		body := []byte("server acknowledgement\x00\xff")
		cloud.Mux.HandleFunc(glanceImportPrefix+"images/selected/import", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			w.Header().Set("X-Ack", "binary")
			w.WriteHeader(202)
			_, _ = w.Write(body)
		})
		v, err := imageimport.New(cloud.Client("image", glanceImportPrefix)).ImportKnownImage(context.Background(), glanceImportKnown())
		if err != nil || v == nil || v.StatusCode != 202 || !bytes.Equal(v.Body, body) || v.Header.Get("X-Ack") != "binary" || calls.Load() != 1 {
			t.Fatal("acknowledgement was decoded or fabricated", v, err, calls.Load())
		}
	})
}

type glanceImportBrokenBody struct {
	first []byte
	cause error
}

func (r *glanceImportBrokenBody) Read(p []byte) (int, error) {
	if len(r.first) > 0 {
		n := copy(p, r.first)
		r.first = r.first[n:]
		return n, nil
	}
	return 0, r.cause
}
func (*glanceImportBrokenBody) Close() error { return nil }

func TestGlanceImportReadTransportAndCancellationAreTerminal(t *testing.T) {
	for _, phase := range []string{"GET", "POST"} {
		for _, outcome := range []string{"read", "transport404", "canceled404", "canceled200"} {
			t.Run(phase+"/"+outcome, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var attempts, gets, retries atomic.Int32
				cause := errors.New("accepted import body stopped")
				transportCause := &gophercloud.ErrUnexpectedResponseCode{Actual: 404}
				cloud.Mux.HandleFunc(glanceImportPrefix+"images/selected", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					testcloud.JSON(w, 200, glanceImportJSON("selected"))
				})
				transport := cloud.Provider.HTTPClient.Transport
				cloud.Provider.HTTPClient.Transport = glanceImportRoundTrip(func(r *http.Request) (*http.Response, error) {
					if r.Method != phase {
						return transport.RoundTrip(r)
					}
					attempts.Add(1)
					code := 200
					if phase == "POST" {
						code = 202
					}
					switch outcome {
					case "read":
						return glanceImportHTTP(code, &glanceImportBrokenBody{first: []byte(`{"partial":`), cause: cause}), nil
					case "transport404":
						return nil, transportCause
					case "canceled404":
						cancel()
						return glanceImportHTTP(404, io.NopCloser(strings.NewReader(`{"missing":true}`))), nil
					default:
						cancel()
						return glanceImportHTTP(code, io.NopCloser(strings.NewReader(`{}`))), nil
					}
				})
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					retries.Add(1)
					return err
				}
				v, err := imageimport.New(cloud.Client("image", glanceImportPrefix)).ImportImage(ctx, resource.ID("selected"))
				if err == nil || attempts.Load() != 1 || (phase == "GET" && v != nil) || (phase == "POST" && gets.Load() != 1) {
					t.Fatal(v, err, attempts.Load(), gets.Load())
				}
				if outcome == "read" {
					var response *resource.ResponseError
					code := 200
					if phase == "POST" {
						code = 202
					}
					if !errors.Is(err, cause) || !errors.As(err, &response) || response.StatusCode != code || string(response.Body) != `{"partial":` || response.Header.Get("X-Evidence") != "actual" || retries.Load() != 0 {
						t.Fatal("accepted read failure replayed or evidence changed", v, err, response, retries.Load())
					}
					if phase == "POST" && (v == nil || v.ImageID != "selected" || v.StatusCode != 202 || string(v.Body) != `{"partial":`) {
						t.Fatal("accepted202 result missing", v, err)
					}
				} else if outcome == "transport404" {
					if !errors.Is(err, transportCause) || errors.Is(err, resource.ErrNotFound) || v != nil {
						t.Fatal(v, err)
					}
				} else if !errors.Is(err, context.Canceled) {
					t.Fatal("caller cancellation hidden by status", v, err)
				}
			})
		}
	}
}

func TestGlanceImportLiveProviderFixedRoutesAndSourceRechecks(t *testing.T) {
	t.Run("current token middleware reauth and retry", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := cloud.Client("image", "/catalog/unused/")
		client.ResourceBase = cloud.Server.URL + glanceImportPrefix
		client.MoreHeaders = map[string]string{"X-Source": "configured"}
		client.Microversion = "2.7"
		cloud.Provider.SetToken("initial")
		var gets, posts, middleware, reauth, retries atomic.Int32
		type key struct{}
		transport := cloud.Provider.HTTPClient.Transport
		cloud.Provider.HTTPClient.Transport = glanceImportRoundTrip(func(r *http.Request) (*http.Response, error) {
			middleware.Add(1)
			if r.Context().Value(key{}) != "caller" || r.Header.Get("X-Source") != "configured" || r.Header.Get("X-Trace") != "final" || r.Header.Get("OpenStack-API-Version") != "image 2.7" {
				t.Error("original client context/headers lost", r.Header)
			}
			return transport.RoundTrip(r)
		})
		cloud.Provider.ReauthFunc = func(context.Context) error { reauth.Add(1); cloud.Provider.SetToken("reauthenticated"); return nil }
		cloud.Provider.RetryFunc = func(_ context.Context, method, target string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			retries.Add(1)
			if method != http.MethodPost || target != cloud.Server.URL+glanceImportPrefix+"images/selected/import" || !gophercloud.ResponseCodeIs(err, 503) {
				return err
			}
			return nil
		}
		cloud.Mux.HandleFunc(glanceImportPrefix+"images/selected", func(w http.ResponseWriter, r *http.Request) {
			gets.Add(1)
			if r.Header.Get("X-Auth-Token") != "initial" {
				t.Error(r.Header)
			}
			cloud.Provider.SetToken("between-phases")
			testcloud.JSON(w, 200, glanceImportJSON("response-decoy"))
		})
		cloud.Mux.HandleFunc(glanceImportPrefix+"images/selected/import", func(w http.ResponseWriter, r *http.Request) {
			n := posts.Add(1)
			fields := glanceImportFields(t, r)
			if string(fields["method"]) != `{"name":"glance-direct"}` {
				t.Error("retry changed body", fields)
			}
			if n == 1 {
				if r.Header.Get("X-Auth-Token") != "between-phases" {
					t.Error(r.Header)
				}
				testcloud.JSON(w, 401, `{}`)
				return
			}
			if r.Header.Get("X-Auth-Token") != "reauthenticated" {
				t.Error(r.Header)
			}
			if n == 2 {
				testcloud.JSON(w, 503, `{}`)
				return
			}
			w.Header().Set("X-Submit", "actual")
			testcloud.JSON(w, 202, `{"submission":true}`)
		})
		v, err := imageimport.New(client).ImportImage(context.WithValue(context.Background(), key{}, "caller"), resource.ID("selected"), imageimport.WithImportHeader("X-Trace", "earlier"), imageimport.WithImportHeader("x-trace", "final"))
		if err != nil || v == nil || v.ImageID != "selected" || v.Header.Get("X-Submit") != "actual" || gets.Load() != 1 || posts.Load() != 3 || middleware.Load() != 4 || reauth.Load() != 1 || retries.Load() != 1 || client.Endpoint != cloud.Server.URL+"/catalog/unused/" || client.ResourceBase != cloud.Server.URL+glanceImportPrefix {
			t.Fatal(v, err, gets.Load(), posts.Load(), middleware.Load(), reauth.Load(), retries.Load())
		}
	})
	for _, when := range []string{"options", "after GET"} {
		t.Run("source recheck "+when, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", glanceImportPrefix)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(glanceImportPrefix+"images/selected", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, glanceImportJSON("selected"))
			})
			opts := []imageimport.ImportOption{}
			if when == "options" {
				opts = append(opts, func(*imageimport.ImportOpts) error { client.Type = "compute"; return nil })
			} else {
				transport := cloud.Provider.HTTPClient.Transport
				cloud.Provider.HTTPClient.Transport = glanceImportRoundTrip(func(r *http.Request) (*http.Response, error) {
					response, err := transport.RoundTrip(r)
					client.Type = "compute"
					return response, err
				})
			}
			v, err := imageimport.New(client).ImportImage(context.Background(), resource.ID("selected"), opts...)
			wantCalls := int32(1)
			if when == "options" {
				wantCalls = 0
			}
			if v != nil || !errors.Is(err, resource.ErrUnsupported) || calls.Load() != wantCalls {
				t.Fatal(v, err, calls.Load())
			}
		})
	}
	t.Run("import redirect cannot change image", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls, wrong atomic.Int32
		cloud.Mux.HandleFunc(glanceImportPrefix+"images/selected/import", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			http.Redirect(w, r, glanceImportPrefix+"images/other/import", 307)
		})
		cloud.Mux.HandleFunc(glanceImportPrefix+"images/other/import", func(w http.ResponseWriter, r *http.Request) { wrong.Add(1); w.WriteHeader(202) })
		v, err := imageimport.New(cloud.Client("image", glanceImportPrefix)).ImportKnownImage(context.Background(), glanceImportKnown())
		if v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 || wrong.Load() != 0 {
			t.Fatal(v, err, calls.Load(), wrong.Load())
		}
	})
	t.Run("configured legacy store may be explicitly replaced", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := cloud.Client("image", glanceImportPrefix)
		client.MoreHeaders = map[string]string{"x-image-meta-store": "source"}
		var calls atomic.Int32
		cloud.Mux.HandleFunc(glanceImportPrefix+"images/selected/import", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.Header.Get("X-Image-Meta-Store") != "owned" {
				t.Error(r.Header)
			}
			if _, exists := glanceImportFields(t, r)["stores"]; exists {
				t.Error("Store was duplicated in JSON")
			}
			w.WriteHeader(202)
		})
		if v, err := imageimport.New(client).ImportKnownImage(context.Background(), glanceImportKnown(), imageimport.WithImportStore("owned")); err != nil || v == nil || calls.Load() != 1 || client.MoreHeaders["x-image-meta-store"] != "source" {
			t.Fatal(v, err, calls.Load(), client.MoreHeaders)
		}
	})
}

func TestGlanceImportNativeCreateGetAndImageModelsStayUnchanged(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("image", glanceImportPrefix)
	var infoGets, imports, imageGets atomic.Int32
	cloud.Mux.HandleFunc(glanceImportPrefix+"info/import", func(w http.ResponseWriter, r *http.Request) {
		infoGets.Add(1)
		testcloud.JSON(w, 200, `{"import-methods":{"description":"available","type":"array","value":["glance-direct","web-download"]}}`)
	})
	cloud.Mux.HandleFunc(glanceImportPrefix+"images/native/import", func(w http.ResponseWriter, r *http.Request) {
		imports.Add(1)
		fields := glanceImportFields(t, r)
		if len(fields) != 1 || string(fields["method"]) != `{"name":"web-download","uri":"https://remote.invalid/native"}` {
			t.Error("native Create body changed", fields)
		}
		w.WriteHeader(202)
	})
	cloud.Mux.HandleFunc(glanceImportPrefix+"images/native", func(w http.ResponseWriter, r *http.Request) {
		imageGets.Add(1)
		w.Header().Set("Openstack-Image-Import-Methods", "glance-direct, web-download")
		w.Header().Set("Openstack-Image-Store-Ids", "a,b")
		testcloud.JSON(w, 200, glanceImportJSON("native"))
	})
	a := imageimport.New(client)
	get := a.Get
	create := a.Create
	info, err := get(context.Background())
	if err != nil || info == nil || len(info.ImportMethods.Value) != 2 || infoGets.Load() != 1 {
		t.Fatal(info, err, infoGets.Load())
	}
	if err := create(context.Background(), "native", imageimport.CreateOpts{Name: imageimport.WebDownloadMethod, URI: "https://remote.invalid/native"}); err != nil || imports.Load() != 1 {
		t.Fatal(err, imports.Load())
	}
	model, err := images.New(client).Get(context.Background(), "native")
	if err != nil || model == nil || model.ContainerFormat != "bare" || model.DiskFormat != "qcow2" || len(model.OpenStackImageImportMethods) != 2 || len(model.OpenStackImageStoreIDs) != 2 || imageGets.Load() != 1 {
		t.Fatal("native image Get/header decoder changed", model, err, imageGets.Load())
	}
}
