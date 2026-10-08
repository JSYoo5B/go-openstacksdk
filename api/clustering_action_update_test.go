package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestClusteringActionUpdateForceQueryAndUnspecifiedResponseBody(t *testing.T) {
	for _, mode := range []string{"omitted", "false", "true"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			body := map[string]string{"omitted": "", "false": `{"accepted":true,"big":9007199254740993}`, "true": "accepted without a declared JSON schema"}[mode]
			cloud.Mux.HandleFunc("PATCH /reverse/v1/actions/action-id", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				payload, _ := io.ReadAll(r.Body)
				wantForce := mode
				if mode == "omitted" {
					wantForce = ""
				}
				if string(payload) != `{"action":{"status":"CANCELLED"}}` || r.URL.Query().Get("force") != wantForce || r.Header.Get("OpenStack-API-Version") != "clustering 1.12" || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Error(string(payload), r.URL, r.Header)
				}
				w.Header().Set("X-Request-Id", "accepted-update")
				w.WriteHeader(202)
				_, _ = io.WriteString(w, body)
			})
			client := cloud.Client("clustering", "/reverse/v1")
			client.Microversion = "1.12"
			api := actions.New(client)
			var options []actions.UpdateOption
			if mode != "omitted" {
				force := mode == "true"
				options = append(options, actions.WithUpdateOptions(actions.UpdateOpts{Status: "CANCELLED", Force: &force}))
				force = !force
			}
			for range 2 {
				value, err := api.Cancel(context.Background(), resource.ID("action-id"), options...)
				if err != nil || value == nil || value.StatusCode != 202 || string(value.Body) != body || value.Header.Get("X-Request-Id") != "accepted-update" {
					t.Fatal(value, err)
				}
				value.Header.Set("X-Request-Id", "consumer-change")
			}
			if calls.Load() != 2 {
				t.Fatal("accepted cancellation was fetched or resent", calls.Load())
			}
		})
	}
}

func TestClusteringActionUpdateFreezesBodyQueryAndHeadersBeforeNameLookup(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/v1")
	client.Microversion = "1.12"
	var captured *request.Config[actions.UpdateOpts]
	var reads, writes atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/actions", func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		if r.URL.Query().Get("name") != "selected" {
			t.Error(r.URL)
		}
		captured.Options.Status = "RUNNING"
		*captured.Options.Force = true
		captured.Query.Set("vendor", "changed")
		captured.Headers["X-Vendor"] = "changed"
		captured.Fields["vendor"] = json.RawMessage(`true`)
		testcloud.JSON(w, 200, `{"actions":[{"id":"resolved-action","name":"selected"}]}`)
	})
	cloud.Mux.HandleFunc("PATCH /v1/actions/resolved-action", func(w http.ResponseWriter, r *http.Request) {
		writes.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"action":{"status":"CANCELLED","vendor":{"big":9007199254740993,"off":false}}}` || r.URL.Query().Get("force") != "false" || r.URL.Query().Get("vendor") != "original" || r.Header.Get("X-Vendor") != "original" {
			t.Error(string(body), r.URL, r.Header)
		}
		w.WriteHeader(202)
	})
	vendor := map[string]any{"big": json.Number("9007199254740993"), "off": false}
	extension := actions.WithUpdateField("vendor", vendor)
	vendor["off"] = true
	capture := func(config *request.Config[actions.UpdateOpts]) error { captured = config; return nil }
	value, err := actions.New(client).Cancel(context.Background(), resource.Name("selected"), actions.WithUpdateForce(false), actions.WithUpdateQuery("vendor", "original"), actions.WithUpdateHeader("X-Vendor", "original"), extension, capture)
	if err != nil || value == nil || value.StatusCode != 202 || reads.Load() != 1 || writes.Load() != 1 {
		t.Fatal(value, err, reads.Load(), writes.Load())
	}
}

func TestClusteringActionUpdatePreflightAndVersionRecheck(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Error("invalid action update reached HTTP", r.URL)
	})
	client := cloud.Client("clustering", "/v1")
	api := actions.New(client)
	for _, version := range []string{"", "1.11", "latest"} {
		client.Microversion = version
		if _, err := api.Cancel(context.Background(), resource.Name("selected")); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(version, err)
		}
	}
	client.Microversion = "1.12"
	for _, status := range []string{"", "RUNNING", "cancelled"} {
		if _, err := api.Update(context.Background(), resource.ID("selected"), actions.UpdateOpts{Status: status}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(status, err)
		}
	}
	for _, option := range []actions.UpdateOption{nil, actions.WithUpdateField("status", "RUNNING"), actions.WithUpdateField("force", true), actions.WithUpdateField("target", "changed"), actions.WithUpdateHeader("x-auth-token", "changed"), actions.WithUpdateQuery("force", "true"), actions.WithUpdateQuery("status", "RUNNING"), actions.WithUpdateField("vendor", make(chan int)), request.WithArgument[actions.UpdateOpts]("ignored", true)} {
		if _, err := api.Cancel(context.Background(), resource.ID("selected"), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Cancel(ctx, resource.ID("selected")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
	for _, change := range []string{"version", "type"} {
		t.Run(change, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("clustering", "/v1")
			client.Microversion = "1.12"
			var reads, writes atomic.Int32
			cloud.Mux.HandleFunc("GET /v1/actions", func(w http.ResponseWriter, r *http.Request) {
				reads.Add(1)
				if change == "version" {
					client.Microversion = "1.11"
				} else {
					client.Type = "network"
				}
				testcloud.JSON(w, 200, `{"actions":[{"id":"selected-id","name":"selected"}]}`)
			})
			cloud.Mux.HandleFunc("PATCH /v1/actions/selected-id", func(w http.ResponseWriter, r *http.Request) { writes.Add(1) })
			_, err := actions.New(client).Cancel(context.Background(), resource.Name("selected"))
			want := resource.ErrUnsupported
			if change == "type" {
				want = resource.ErrInvalidOption
			}
			if !errors.Is(err, want) || reads.Load() != 1 || writes.Load() != 0 {
				t.Fatal(err, reads.Load(), writes.Load())
			}
		})
	}
}

func TestClusteringActionUpdateOnlyAcceptsPublished202AndPreservesNativeErrors(t *testing.T) {
	for _, code := range []int{200, 201, 204, 400, 403, 404, 409} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("PATCH /v1/actions/selected", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, code, `{"error":"update rejected"}`)
			})
			client := cloud.Client("clustering", "/v1")
			client.Microversion = "1.12"
			value, err := actions.New(client).Cancel(context.Background(), resource.ID("selected"))
			if value != nil || !gophercloud.ResponseCodeIs(err, code) || code != 204 && !strings.Contains(err.Error(), "update rejected") || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
		})
	}
}
