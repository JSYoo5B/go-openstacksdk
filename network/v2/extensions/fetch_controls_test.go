package extensions_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestExtensionFindFetchControlsCollectionKeepsConstructorSeedAndAliasProjection(t *testing.T) {
	body := `{"selected":{"alias":"wire-alias","updated":"literal","links":[false,{"href":"/qos"}],"x_new":{"n":9007199254740993}},"extension":{"id":"wrong"}}`
	api, count := recordExtensionAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/neutron/v2.0/alternate-extensions" || r.URL.Query().Get("x_seed") != "seed-key" || r.Header.Get("Cache-Control") != "" {
			t.Error(r.URL, r.Header)
		}
		for _, key := range []string{"requires_id", "base_path", "skip_cache", "error_message", "resource_response_key"} {
			if r.URL.Query().Has(key) {
				t.Error(r.URL)
			}
		}
		testcloud.JSON(w, 203, body)
	})
	row, err := api.FindRecord(context.Background(), "requested", extensions.WithRecordFindQuery("requires_id", false), extensions.WithRecordFindQuery("base_path", "/alternate-extensions"), extensions.WithRecordFindQuery("skip_cache", true), extensions.WithRecordFindQuery("error_message", "custom missing"), extensions.WithRecordFindQuery("resource_response_key", "selected"), extensions.WithRecordFindQuery("x_seed", map[string]any{"seed-key": nil}))
	if err != nil || row == nil || count.Load() != 1 || row.StatusCode != 203 || string(row.Envelope) != body || row.Header.Get("X-Request-Id") != "extension-actual" {
		t.Fatal(row, err, count.Load())
	}
	for key, want := range map[string]string{"id": `"requested"`, "alias": `"wire-alias"`, "updated_at": `"literal"`, "links": `[{},{"href":"/qos"}]`, "x_new": `{"n":9007199254740993}`, "x_seed": `{"seed-key":null}`, "requires_id": "false", "base_path": `"/alternate-extensions"`, "skip_cache": "true", "error_message": `"custom missing"`, "resource_response_key": `"selected"`} {
		extensionRecordField(t, row, key, want)
	}
	if row.Wire.Body["skip_cache"] != nil || row.Resource.Body["updated"] != nil || string(row.Wire.Body["links"]) != `[false,{"href":"/qos"}]` {
		t.Fatal(row)
	}
}

func TestExtensionFindFetchControlsEnvelopeNullIDAndMalformedProof(t *testing.T) {
	t.Run("explicit null id wins alias", func(t *testing.T) {
		api, count := recordExtensionAPI(t, func(w http.ResponseWriter, r *http.Request) {
			testcloud.JSON(w, 200, `{"selected":{"id":null,"alias":"fallback","updated":null,"links":null}}`)
		})
		row, err := api.FindRecord(context.Background(), "requested", extensions.WithRecordFindQuery("resource_response_key", "selected"))
		if err != nil || row == nil || count.Load() != 1 {
			t.Fatal(row, err, count.Load())
		}
		extensionRecordField(t, row, "id", "null")
		extensionRecordField(t, row, "updated_at", "null")
		extensionRecordField(t, row, "links", "null")
	})
	t.Run("malformed selected envelope keeps accepted response", func(t *testing.T) {
		body := `{"selected":[],"extension":{"id":"wrong"}}`
		api, count := recordExtensionAPI(t, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 202, body) })
		row, err := api.FindRecord(context.Background(), "requested", extensions.WithRecordFindQuery("resource_response_key", "selected"))
		if row != nil || err == nil || count.Load() != 1 {
			t.Fatal(row, err, count.Load())
		}
		extensionRecordProof(t, err, 202, body)
	})
}

func TestExtensionFindFetchControlsRawHeadersUseGETQueryThenFallbackTransport(t *testing.T) {
	var calls atomic.Int32
	api, count := recordExtensionAPI(t, func(w http.ResponseWriter, r *http.Request) {
		phase := calls.Add(1)
		if r.Header.Get("X-Typed") != "kept" {
			t.Error(r.Header)
		}
		if phase == 1 {
			if r.URL.Path != "/neutron/v2.0/alternate-extensions/target" || !reflect.DeepEqual(r.URL.Query()["headers"], []string{"Accept", "X-Raw"}) || r.Header.Get("X-Raw") != "typed" {
				t.Error(r.URL, r.Header)
			}
			testcloud.JSON(w, 404, `{}`)
			return
		}
		if r.URL.Path != "/neutron/v2.0/list-extensions" || r.URL.Query().Has("headers") || r.URL.Query().Has("name") || r.Header.Get("Accept") != "application/vnd.extension+json" || r.Header.Get("X-Raw") != "raw" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"extensions":[{"alias":"selected","name":"target","updated":"literal","links":false}]}`)
	})
	row, err := api.FindRecord(context.Background(), "target", extensions.WithRecordFindQuery("base_path", "/alternate-extensions"), extensions.WithRecordFindListBasePath("/list-extensions"), extensions.WithRecordFindQuery("headers", json.RawMessage(`{"Accept":"application/vnd.extension+json","X-Raw":"raw"}`)), extensions.WithRecordFindHeaders(map[string]string{"X-Typed": "kept", "x-raw": "typed"}))
	if err != nil || row == nil || count.Load() != 2 {
		t.Fatal(row, err, count.Load())
	}
	extensionRecordField(t, row, "id", `"selected"`)
	extensionRecordField(t, row, "updated_at", `"literal"`)
	extensionRecordField(t, row, "links", `[{}]`)
	if row.Resource.Body["headers"] != nil || row.Resource.Body["base_path"] != nil {
		t.Fatal(row.Resource)
	}
}

func TestExtensionFindFetchControlsRawHeadersValidateOnlyAtFallback(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		t.Run(map[bool]string{false: "successful-get", true: "fallback"}[fallback], func(t *testing.T) {
			api, count := recordExtensionAPI(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("headers") != "X-Raw" {
					t.Error(r.URL)
				}
				if fallback {
					testcloud.JSON(w, 403, `{}`)
					return
				}
				testcloud.JSON(w, 200, `{"extension":{"alias":"qos"}}`)
			})
			row, err := api.FindRecord(context.Background(), "requested", extensions.WithRecordFindQuery("headers", json.RawMessage(`{"X-Raw":7}`)))
			if count.Load() != 1 {
				t.Fatal(count.Load())
			}
			if fallback {
				if row != nil || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(row, err)
				}
				return
			}
			if err != nil || row == nil {
				t.Fatal(row, err)
			}
			extensionRecordField(t, row, "headers", `{"X-Raw":7}`)
		})
	}
}

func TestExtensionFindFetchControlsOwnOptionsWithoutCacheAndKeepHTTPCause(t *testing.T) {
	t.Run("owned options perform fresh GET", func(t *testing.T) {
		query := map[string]json.RawMessage{"base_path": json.RawMessage(`"/owned-extensions"`), "requires_id": json.RawMessage(`false`), "resource_response_key": json.RawMessage(`"selected"`), "skip_cache": json.RawMessage(`false`)}
		headers := map[string]string{"X-Owned": "original"}
		option := extensions.WithRecordFindOpts(extensions.RecordFindOpts{Query: query, Headers: headers})
		copy(query["base_path"], []byte(`"/other-extensions"`))
		query["resource_response_key"] = json.RawMessage(`"other"`)
		headers["X-Owned"] = "mutated"
		var calls atomic.Int32
		api, count := recordExtensionAPI(t, func(w http.ResponseWriter, r *http.Request) {
			n := calls.Add(1)
			if r.URL.Path != "/neutron/v2.0/owned-extensions" || r.URL.RawQuery != "" || r.Header.Get("X-Owned") != "original" || r.Header.Get("Cache-Control") != "" {
				t.Error(r.URL, r.Header)
			}
			if n == 1 {
				testcloud.JSON(w, 200, `{"selected":{"alias":"first","updated":"one"}}`)
				return
			}
			testcloud.JSON(w, 200, `{"selected":{"alias":"second","updated":"two"}}`)
		})
		for _, want := range []string{`"one"`, `"two"`} {
			row, err := api.FindRecord(context.Background(), "requested", option)
			if err != nil || row == nil {
				t.Fatal(row, err)
			}
			extensionRecordField(t, row, "updated_at", want)
			extensionRecordField(t, row, "skip_cache", "false")
		}
		if count.Load() != 2 {
			t.Fatal(count.Load())
		}
	})
	t.Run("custom HTTP cause", func(t *testing.T) {
		body := `{"error":"upstream proof"}`
		api, count := recordExtensionAPI(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Has("error_message") {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 500, body)
		})
		row, err := api.FindRecord(context.Background(), "requested", extensions.WithRecordFindQuery("error_message", "extension lookup failed"))
		var native gophercloud.ErrUnexpectedResponseCode
		if row != nil || count.Load() != 1 || !errors.As(err, &native) || native.Actual != 500 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "extension-actual" || !strings.Contains(err.Error(), "extension lookup failed") {
			t.Fatal(row, err, native, count.Load())
		}
	})
}

func TestExtensionFindFetchControlsPhysicalFaultsNeverFallback(t *testing.T) {
	for _, mode := range []string{"read", "close", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, closed atomic.Int32
			cause := errors.New("extension controlled " + mode)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cloud.Mux.HandleFunc("/neutron/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 404, `{"error":"unclean"}`)
			})
			transport := extensionObservedTransport{base: cloud.Provider.HTTPClient.Transport, closed: &closed}
			if transport.base == nil {
				transport.base = http.DefaultTransport
			}
			if mode == "read" {
				transport.readErr = cause
			}
			if mode == "close" {
				transport.closeErr = cause
			}
			if mode == "cancel" {
				transport.onClose = func() { cancel(cause) }
			}
			cloud.Provider.HTTPClient.Transport = transport
			client := cloud.Client("network", "/catalog/")
			client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
			row, err := extensions.New(client).FindRecord(ctx, "requested", extensions.WithRecordFindQuery("requires_id", false), extensions.WithRecordFindQuery("skip_cache", true), extensions.WithRecordFindQuery("error_message", "controlled extension lookup"))
			if row != nil || calls.Load() != 1 || closed.Load() != 1 || !errors.Is(err, cause) {
				t.Fatal(row, err, calls.Load(), closed.Load())
			}
		})
	}
}
