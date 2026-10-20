package imageimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/images"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func importCoreClient(server *httptest.Server) *gophercloud.ServiceClient {
	return &gophercloud.ServiceClient{Type: "image", Endpoint: server.URL + "/v2/", ProviderClient: &gophercloud.ProviderClient{TokenID: "first", HTTPClient: *server.Client()}}
}

func TestImportCoreKnownSeedAndHTTPBodySnapshots(t *testing.T) {
	seed := &images.Image{ID: "fixed", ContainerFormat: "bare", DiskFormat: "qcow2"}
	var calls atomic.Int32
	var captured *ImportOpts
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		if req.Method != http.MethodPost || req.URL.Path != "/v2/images/fixed/import" || req.Header.Get("X-Image-Meta-Store") != "selected" || req.Header.Get("X-Note") != "initial" {
			t.Errorf("unexpected request %s %s %#v", req.Method, req.URL.Path, req.Header)
		}
		body, _ := io.ReadAll(req.Body)
		if string(body) != `{"all_stores_must_succeed":false,"method":{"name":"glance-direct","vendor":9007199254740993},"root":{"nested":["initial"]}}` {
			t.Errorf("body %s", body)
		}
		captured.Headers["X-Note"] = "during"
		captured.Fields["root"] = "during"
		writer.Header().Set("X-Ack", "real")
		writer.Header().Set("Location", "https://elsewhere/tasks/not-inferred")
		writer.WriteHeader(http.StatusAccepted)
		fmt.Fprint(writer, "arbitrary asynchronous bytes")
	}))
	defer server.Close()
	api := New(importCoreClient(server))
	options := []ImportOption{WithImportStore("selected"), WithImportAllStoresMustSucceed(false),
		WithImportHeader("X-Note", "initial"), WithImportMethodField("vendor", json.Number("9007199254740993")),
		WithImportField("root", map[string]any{"nested": []string{"initial"}}), func(config *ImportOpts) error {
			captured = config
			seed.ID, seed.ContainerFormat, seed.DiskFormat = "wrong", "", ""
			return nil
		}}
	result, err := api.ImportKnownImage(context.Background(), seed, options...)
	if err != nil || calls.Load() != 1 || result == nil || result.ImageID != "fixed" || result.StatusCode != 202 || result.Header.Get("X-Ack") != "real" || string(result.Body) != "arbitrary asynchronous bytes" {
		t.Fatalf("result %#v error %v calls %d", result, err, calls.Load())
	}
	if len(options) != 6 {
		t.Fatal("caller option slice modified")
	}
}

func TestImportCoreFreshCanonicalFormatsAndNativeDecodeEvidence(t *testing.T) {
	cases := []struct {
		name, body string
		valid      bool
	}{
		{"canonical", `{"id":"different","container_format":"bare","disk_format":"raw"}`, true},
		{"missing", `{"disk_format":"raw"}`, false},
		{"case-only", `{"Container_Format":"bare","disk_format":"raw"}`, false},
		{"null", `{"container_format":null,"disk_format":"raw"}`, false},
		{"empty", `{"container_format":"","disk_format":"raw"}`, false},
		{"type", `{"container_format":true,"disk_format":"raw"}`, false},
		{"unrelated-native-type", `{"container_format":"bare","disk_format":"raw","min_ram":"bad"}`, false},
		{"unrelated-native-time", `{"container_format":"bare","disk_format":"raw","created_at":"bad"}`, false},
		{"not-object", `[]`, false},
		{"trailing", `{"container_format":"bare","disk_format":"raw"}{}`, false},
		{"invalid-utf8", "{\"container_format\":\"bare\",\"disk_format\":\"raw\",\"vendor\":\"" + string([]byte{255}) + "\"}", false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var gets, posts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodGet {
					gets.Add(1)
					writer.Header().Set("Content-Type", "application/json")
					writer.Header().Set("X-Metadata-Proof", "actual")
					fmt.Fprint(writer, test.body)
					return
				}
				posts.Add(1)
				if req.URL.Path != "/v2/images/requested/import" {
					t.Errorf("response ID rerouted: %s", req.URL.Path)
				}
				writer.WriteHeader(202)
			}))
			defer server.Close()
			value, err := New(importCoreClient(server)).ImportImage(context.Background(), resource.ID("requested"))
			if test.valid {
				if err != nil || value == nil || gets.Load() != 1 || posts.Load() != 1 {
					t.Fatalf("valid result %#v %v GET%d POST%d", value, err, gets.Load(), posts.Load())
				}
				return
			}
			var evidence *resource.ResponseError
			if value != nil || err == nil || gets.Load() != 1 || posts.Load() != 0 || !errors.As(err, &evidence) || evidence.StatusCode != 200 || string(evidence.Body) != test.body || evidence.Header.Get("X-Metadata-Proof") != "actual" {
				t.Fatalf("invalid result %#v %v GET%d POST%d proof%#v", value, err, gets.Load(), posts.Load(), evidence)
			}
		})
	}
}

func TestImportCorePreflightAndSourceBeforeEveryPhase(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		writer.Header().Set("Content-Type", "application/json")
		fmt.Fprint(writer, `{"container_format":"bare","disk_format":"raw"}`)
	}))
	defer server.Close()
	client := importCoreClient(server)
	api := New(client)
	seed := &images.Image{ID: "id", ContainerFormat: "bare", DiskFormat: "raw"}
	invalid := []func() error{
		func() error { _, err := api.ImportImage(nil, resource.ID("id")); return err },
		func() error { _, err := (*API)(nil).ImportImage(context.Background(), resource.ID("id")); return err },
		func() error { _, err := api.ImportImage(context.Background(), resource.ID("../other")); return err },
		func() error { _, err := api.ImportKnownImage(context.Background(), nil); return err },
		func() error {
			_, err := api.ImportKnownImage(context.Background(), &images.Image{ID: "id"})
			return err
		},
		func() error {
			_, err := api.ImportImage(context.Background(), resource.ID("id"), WithImportMethod(WebDownloadMethod))
			return err
		},
		func() error {
			_, err := api.ImportKnownImage(context.Background(), seed, WithImportHeader("Authorization", "bad"))
			return err
		},
		func() error {
			_, err := api.ImportKnownImage(context.Background(), seed, WithImportField("stores", nil))
			return err
		},
	}
	for index, run := range invalid {
		if err := run(); err == nil {
			t.Errorf("invalid case %d accepted", index)
		}
	}
	client.MoreHeaders = map[string]string{"x-image-meta-store": "configured"}
	if _, err := api.ImportImage(context.Background(), resource.ID("id"), WithImportStores("plural")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Errorf("source selector conflict: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("preflight HTTP count %d", calls.Load())
	}
	client.MoreHeaders = nil
	client.ProviderClient.HTTPClient.Transport = importCoreTransport(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		client.Type = "other"
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"container_format":"bare","disk_format":"raw"}`))}, nil
	})
	result, err := api.ImportImage(context.Background(), resource.ID("id"))
	if result != nil || !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 1 {
		t.Fatalf("source change continued to POST: %#v, %v, %d", result, err, calls.Load())
	}
}

type importCoreTransport func(*http.Request) (*http.Response, error)

func (run importCoreTransport) RoundTrip(req *http.Request) (*http.Response, error) { return run(req) }

type importCoreReadFailure struct{ cause error }

func (reader importCoreReadFailure) Read(bytes []byte) (int, error) {
	return copy(bytes, "part"), reader.cause
}
func (reader importCoreReadFailure) Close() error { return nil }

func TestImportCoreAcceptedReadFailureAndNativeErrors(t *testing.T) {
	readCause := errors.New("accepted body failed")
	for _, code := range []int{202, 400, 403, 404, 409, 503} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls int
			client := &gophercloud.ServiceClient{Type: "image", Endpoint: "https://example.invalid/reverse/v2/", ProviderClient: &gophercloud.ProviderClient{TokenID: "live"}}
			client.ProviderClient.HTTPClient.Transport = importCoreTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.Path != "/reverse/v2/images/id/import" || req.Header.Get("X-Auth-Token") != "live" {
					t.Errorf("route/token %s %#v", req.URL, req.Header)
				}
				body := io.ReadCloser(io.NopCloser(strings.NewReader("native error")))
				if code == 202 {
					body = importCoreReadFailure{readCause}
				}
				return &http.Response{StatusCode: code, Header: http.Header{"X-Evidence": {"actual"}}, Body: body}, nil
			})
			seed := &images.Image{ID: "id", ContainerFormat: "bare", DiskFormat: "raw"}
			result, err := New(client).ImportKnownImage(context.Background(), seed)
			if calls != 1 || err == nil {
				t.Fatalf("calls%d result%#v error%v", calls, result, err)
			}
			if code == 202 {
				var evidence *resource.ResponseError
				if result == nil || result.ImageID != "id" || result.StatusCode != 202 || string(result.Body) != "part" || !errors.Is(err, readCause) || !errors.As(err, &evidence) || evidence.StatusCode != 202 || string(evidence.Body) != "part" || evidence.Header.Get("X-Evidence") != "actual" {
					t.Fatalf("accepted evidence: %#v, %v, %#v", result, err, evidence)
				}
				return
			}
			if result != nil || !gophercloud.ResponseCodeIs(err, code) {
				t.Fatalf("native code%d fabricated ack: %#v, %v", code, result, err)
			}
		})
	}
}

func TestImportCoreCanceledNativeErrorsPreserveBothCauses(t *testing.T) {
	for _, known := range []bool{false, true} {
		t.Run(fmt.Sprint(known), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client := &gophercloud.ServiceClient{Type: "image", Endpoint: "https://example.invalid/v2/", ProviderClient: &gophercloud.ProviderClient{}}
			native := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Body: []byte("real404"), ResponseHeader: http.Header{"X-Native": {"actual"}}}
			var calls int
			client.ProviderClient.HTTPClient.Transport = importCoreTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				cancel()
				return nil, native
			})
			api := New(client)
			var value *ImportResult
			var err error
			if known {
				value, err = api.ImportKnownImage(ctx, &images.Image{ID: "id", ContainerFormat: "bare", DiskFormat: "raw"})
			} else {
				value, err = api.ImportImage(ctx, resource.ID("id"))
			}
			var actual gophercloud.ErrUnexpectedResponseCode
			if value != nil || calls != 1 || !errors.Is(err, context.Canceled) || !errors.As(err, &actual) || actual.Actual != 404 || string(actual.Body) != "real404" || actual.ResponseHeader.Get("X-Native") != "actual" {
				t.Fatalf("canceled native evidence value%#v calls%d err%v native%#v", value, calls, err, actual)
			}
		})
	}
}
