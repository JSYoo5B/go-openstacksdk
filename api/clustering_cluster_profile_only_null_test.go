package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/clustering/v1/clusters"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// Explicit null is a Go presence option for the pinned mutable profile_only
// attribute. Null meaning remains controller-owned; every present value uses
// the same >=1.6 gate as false/true and an accepted PATCH202 action response.
const clusterProfileOnlyNullPrefix = "/proxy/tenant/senlin/v1"

func clusterProfileOnlyNullCloud(t *testing.T) (*testcloud.Cloud, *gophercloud.ServiceClient) {
	t.Helper()
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/catalog/v1")
	client.ResourceBase = cloud.Server.URL + clusterProfileOnlyNullPrefix + "/"
	client.Microversion = "1.6"
	return cloud, client
}

func clusterProfileOnlyNullWire(t *testing.T, r *http.Request, expected map[string]string) {
	t.Helper()
	if r.Method != http.MethodPatch || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" {
		t.Error("unexpected mutation request", r.Method, r.URL, r.Header)
	}
	var body map[string]map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Error(err)
	}
	if len(body) != 1 || len(body["cluster"]) != len(expected) {
		t.Error("unexpected envelope/fields", body, expected)
	}
	for key, want := range expected {
		if string(body["cluster"][key]) != want {
			t.Error(key, string(body["cluster"][key]), want)
		}
	}
}

func clusterProfileOnlyNullAccept(w http.ResponseWriter) {
	w.Header().Set("Location", clusterProfileOnlyNullPrefix+"/actions/profile-only-update")
	w.Header().Set("X-Request-Id", "profile-only-request")
	testcloud.JSON(w, http.StatusAccepted, `{"cluster":{}}`)
}

func clusterProfileOnlyNullTrack(t *testing.T, client *gophercloud.ServiceClient) *clusters.TrackedCluster {
	t.Helper()
	var seed clusters.Cluster
	if err := json.Unmarshal([]byte(`{"id":"fixed","name":"Initial","metadata":{"off":false}}`), &seed); err != nil {
		t.Fatal(err)
	}
	tracked, err := clusters.New(client).Track(&seed)
	if err != nil {
		t.Fatal(err)
	}
	return tracked
}

func clusterProfileOnlyNullDecoded(t *testing.T, encoded string) clusters.UpdateOpts {
	t.Helper()
	var opts clusters.UpdateOpts
	if err := json.Unmarshal([]byte(encoded), &opts); err != nil {
		t.Fatal(err)
	}
	return opts
}

func TestClusteringClusterProfileOnlyNullStatelessWireSnapshotsAndLastWins(t *testing.T) {
	for _, mode := range []string{"helper-null", "null-then-false", "false-then-null", "json-null", "json-uppercase-null", "json-false-then-uppercase-null", "json-null-then-uppercase-false", "json-repeated-key-last-null", "json-repeated-key-last-true", "snapshot-reuse", "bulk-empty-clears-null", "unmarshal-empty-clears-null"} {
		t.Run(mode, func(t *testing.T) {
			cloud, client := clusterProfileOnlyNullCloud(t)
			client.MoreHeaders = map[string]string{"X-Source": "original"}
			beforeHeaders := map[string]string{"X-Source": "original"}
			opts := clusters.UpdateOpts{}
			options := []clusters.UpdateOption{}
			expected := map[string]string{"profile_only": "null"}
			callsExpected := int32(1)
			switch mode {
			case "helper-null":
				options = append(options, clusters.WithUpdateProfileOnlyNull())
			case "null-then-false":
				options = append(options, clusters.WithUpdateProfileOnlyNull(), clusters.WithUpdateProfileOnly(false))
				expected["profile_only"] = "false"
			case "false-then-null":
				options = append(options, clusters.WithUpdateProfileOnly(false), clusters.WithUpdateProfileOnlyNull())
			case "json-null", "json-uppercase-null", "json-false-then-uppercase-null", "json-null-then-uppercase-false", "json-repeated-key-last-null", "json-repeated-key-last-true":
				input := map[string]string{
					"json-null":                      `{"profile_only":null}`,
					"json-uppercase-null":            `{"PROFILE_ONLY":null}`,
					"json-false-then-uppercase-null": `{"profile_only":false,"PROFILE_ONLY":null}`,
					"json-null-then-uppercase-false": `{"profile_only":null,"PROFILE_ONLY":false}`,
					"json-repeated-key-last-null":    `{"profile_only":false,"profile_only":null}`,
					"json-repeated-key-last-true":    `{"profile_only":null,"profile_only":true}`,
				}[mode]
				opts = clusterProfileOnlyNullDecoded(t, input)
				if mode == "json-null-then-uppercase-false" {
					expected["profile_only"] = "false"
				} else if mode == "json-repeated-key-last-true" {
					expected["profile_only"] = "true"
				}
				encoded, err := json.Marshal(opts)
				want := expected["profile_only"]
				if err != nil || string(encoded) != `{"profile_only":`+want+`}` || (opts.ProfileOnly == nil) != (want == "null") {
					t.Fatal("JSON case-folded presence/last value did not round trip", input, string(encoded), opts.ProfileOnly, err)
				}
				options = append(options, clusters.WithUpdateOptions(opts))
			case "snapshot-reuse":
				opts = clusterProfileOnlyNullDecoded(t, `{"profile_only":null,"metadata":{"big":9007199254740993,"off":false}}`)
				option := clusters.WithUpdateOptions(opts)
				options = append(options, option)
				opts.Metadata[0] = '['
				enabled := true
				opts.ProfileOnly = &enabled
				opts = clusters.UpdateOpts{}
				expected["metadata"] = `{"big":9007199254740993,"off":false}`
				callsExpected = 2
			case "bulk-empty-clears-null":
				options = append(options, clusters.WithUpdateProfileOnlyNull(), clusters.WithUpdateOptions(clusters.UpdateOpts{}), clusters.WithUpdateName("Bulk"))
				expected = map[string]string{"name": `"Bulk"`}
			case "unmarshal-empty-clears-null":
				opts = clusterProfileOnlyNullDecoded(t, `{"profile_only":null}`)
				if err := json.Unmarshal([]byte(`{}`), &opts); err != nil {
					t.Fatal(err)
				}
				opts.Name = request.Present("Bulk")
				expected = map[string]string{"name": `"Bulk"`}
			}
			options = append(options, clusters.WithUpdateHeader("X-Request-Option", "original"))
			var patches, gets atomic.Int32
			cloud.Mux.HandleFunc("PATCH "+clusterProfileOnlyNullPrefix+"/clusters/fixed", func(w http.ResponseWriter, r *http.Request) {
				patches.Add(1)
				clusterProfileOnlyNullWire(t, r, expected)
				if r.Header.Get("OpenStack-API-Version") != "clustering 1.6" || r.Header.Get("X-Source") != "original" || r.Header.Get("X-Request-Option") != "original" {
					t.Error(r.Header)
				}
				clusterProfileOnlyNullAccept(w)
			})
			cloud.Mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); w.WriteHeader(500) })
			for i := int32(0); i < callsExpected; i++ {
				value, err := clusters.New(client).Update(context.Background(), resource.ID("fixed"), opts, options...)
				if err != nil || value == nil || value.Operation == nil || value.Operation.ActionID != "profile-only-update" || value.Operation.StatusCode != 202 || value.Operation.Header.Get("X-Request-Id") != "profile-only-request" {
					t.Fatal(value, err)
				}
			}
			if patches.Load() != callsExpected || gets.Load() != 0 || client.Microversion != "1.6" || !reflect.DeepEqual(client.MoreHeaders, beforeHeaders) {
				t.Fatal("options changed source or caused extra HTTP", patches.Load(), gets.Load(), client.Microversion, client.MoreHeaders)
			}
		})
	}
	t.Run("failed-decode-preserves-existing-options", func(t *testing.T) {
		cloud, client := clusterProfileOnlyNullCloud(t)
		opts := clusterProfileOnlyNullDecoded(t, `{"profile_only":null,"metadata":{"big":9007199254740993,"off":false}}`)
		before, err := json.Marshal(opts)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(`{"profile_only":false,"metadata":{},"timeout":"invalid"}`), &opts); err == nil {
			t.Fatal("invalid scalar was decoded")
		}
		after, err := json.Marshal(opts)
		if err != nil || string(after) != string(before) || opts.ProfileOnly != nil {
			t.Fatal("failed decode partially changed options/null presence", string(before), string(after), opts.ProfileOnly, err)
		}
		var patches atomic.Int32
		cloud.Mux.HandleFunc("PATCH "+clusterProfileOnlyNullPrefix+"/clusters/fixed", func(w http.ResponseWriter, r *http.Request) {
			patches.Add(1)
			clusterProfileOnlyNullWire(t, r, map[string]string{"profile_only": "null", "metadata": `{"big":9007199254740993,"off":false}`})
			clusterProfileOnlyNullAccept(w)
		})
		if _, err := clusters.New(client).Update(context.Background(), resource.ID("fixed"), clusters.UpdateOpts{}, clusters.WithUpdateOptions(opts)); err != nil || patches.Load() != 1 {
			t.Fatal(err, patches.Load())
		}
	})
	t.Run("protected-core-fields", func(t *testing.T) {
		cloud, client := clusterProfileOnlyNullCloud(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
		for _, option := range []clusters.UpdateOption{clusters.WithUpdateField("profile_only", nil), clusters.WithUpdateField("is_profile_only", nil)} {
			if _, err := clusters.New(client).Update(context.Background(), resource.Name("not-looked-up"), clusters.UpdateOpts{}, clusters.WithUpdateProfileOnlyNull(), option); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		}
		if calls.Load() != 0 {
			t.Fatal(calls.Load())
		}
	})
}

func TestClusteringClusterProfileOnlyNullTrackedPresenceDirtyAndVersionGate(t *testing.T) {
	t.Run("presence-last-wins-and-removal", func(t *testing.T) {
		cloud, client := clusterProfileOnlyNullCloud(t)
		tracked := clusterProfileOnlyNullTrack(t, client)
		var patches atomic.Int32
		want := "null"
		cloud.Mux.HandleFunc("PATCH "+clusterProfileOnlyNullPrefix+"/clusters/fixed", func(w http.ResponseWriter, r *http.Request) {
			patches.Add(1)
			clusterProfileOnlyNullWire(t, r, map[string]string{"profile_only": want})
			clusterProfileOnlyNullAccept(w)
		})
		if err := tracked.RemoveProfileOnly(); err != nil || tracked.Dirty() {
			t.Fatal("removing absent field made a tombstone", err, tracked.Dirty())
		}
		if _, err := tracked.Commit(context.Background()); err != nil || patches.Load() != 0 {
			t.Fatal(err, patches.Load())
		}
		if err := tracked.Edit(clusters.UpdateOpts{}, clusters.WithUpdateProfileOnlyNull()); err != nil || !tracked.Dirty() || string(tracked.Value().Body["profile_only"]) != "null" {
			t.Fatal("explicit null was treated as absence", err, tracked.Value())
		}
		value, err := tracked.Commit(context.Background())
		if err != nil || tracked.Dirty() || string(value.Body["profile_only"]) != "null" || value.Operation == nil || patches.Load() != 1 {
			t.Fatal(value, err, tracked.Dirty(), patches.Load())
		}
		if _, present := tracked.Response().Body["profile_only"]; present {
			t.Fatal("actual response was replaced by merged cache", tracked.Response())
		}
		if err := tracked.Edit(clusters.UpdateOpts{}, clusters.WithUpdateProfileOnlyNull()); err != nil || tracked.Dirty() {
			t.Fatal("same cached null became dirty", err)
		}
		if _, err := tracked.Commit(context.Background()); err != nil || patches.Load() != 1 {
			t.Fatal(err, patches.Load())
		}
		if err := tracked.RemoveProfileOnly(); err != nil || !tracked.Dirty() {
			t.Fatal(err)
		}
		if _, present := tracked.Value().Body["profile_only"]; present {
			t.Fatal("tombstone retained cached field", tracked.Value())
		}
		if _, err := tracked.Commit(context.Background()); err != nil || tracked.Dirty() || patches.Load() != 2 {
			t.Fatal(err, tracked.Dirty(), patches.Load())
		}
		if err := tracked.RemoveProfileOnly(); err != nil || tracked.Dirty() {
			t.Fatal(err)
		}
		if _, err := tracked.Commit(context.Background()); err != nil || patches.Load() != 2 {
			t.Fatal(err, patches.Load())
		}
		want = "false"
		if err := tracked.Edit(clusters.UpdateOpts{}, clusters.WithUpdateProfileOnlyNull(), clusters.WithUpdateProfileOnly(false)); err != nil || string(tracked.Value().Body["profile_only"]) != "false" {
			t.Fatal(err, tracked.Value())
		}
		if _, err := tracked.Commit(context.Background()); err != nil || patches.Load() != 3 {
			t.Fatal(err, patches.Load())
		}
		want = "null"
		if err := tracked.Edit(clusters.UpdateOpts{}, clusters.WithUpdateProfileOnly(false), clusters.WithUpdateProfileOnlyNull()); err != nil || string(tracked.Value().Body["profile_only"]) != "null" {
			t.Fatal(err, tracked.Value())
		}
		if _, err := tracked.Commit(context.Background()); err != nil || tracked.Dirty() || patches.Load() != 4 {
			t.Fatal(err, patches.Load())
		}
	})
	t.Run("snapshot-null-gate-and-empty-bulk", func(t *testing.T) {
		cloud, client := clusterProfileOnlyNullCloud(t)
		client.Microversion = "1.5"
		tracked := clusterProfileOnlyNullTrack(t, client)
		var patches atomic.Int32
		cloud.Mux.HandleFunc("PATCH "+clusterProfileOnlyNullPrefix+"/clusters/fixed", func(w http.ResponseWriter, r *http.Request) {
			patches.Add(1)
			clusterProfileOnlyNullWire(t, r, map[string]string{"profile_only": "null"})
			if r.Header.Get("OpenStack-API-Version") != "clustering 1.6" {
				t.Error(r.Header)
			}
			clusterProfileOnlyNullAccept(w)
		})
		if err := tracked.Edit(clusters.UpdateOpts{}, clusters.WithUpdateProfileOnlyNull(), clusters.WithUpdateOptions(clusters.UpdateOpts{})); err != nil || tracked.Dirty() {
			t.Fatal("bulk empty failed to clear null option", err)
		}
		opts := clusterProfileOnlyNullDecoded(t, `{"profile_only":null}`)
		snapshot := clusters.WithUpdateOptions(opts)
		enabled := true
		opts.ProfileOnly = &enabled
		if err := tracked.Edit(clusters.UpdateOpts{}, snapshot); err != nil || !tracked.Dirty() || string(tracked.Value().Body["profile_only"]) != "null" {
			t.Fatal("snapshot lost null presence", err, tracked.Value())
		}
		if _, err := tracked.Commit(context.Background()); !errors.Is(err, resource.ErrUnsupported) || !tracked.Dirty() || patches.Load() != 0 {
			t.Fatal("pending null bypassed 1.6 gate", err, tracked.Dirty(), patches.Load())
		}
		client.Microversion = "1.6"
		if _, err := tracked.Commit(context.Background()); err != nil || tracked.Dirty() || patches.Load() != 1 {
			t.Fatal(err, tracked.Dirty(), patches.Load())
		}
		// A snapshot option can be reused; equality against current cached null
		// keeps both applications clean after the accepted revision.
		for i := 0; i < 2; i++ {
			if err := tracked.Edit(clusters.UpdateOpts{}, snapshot); err != nil || tracked.Dirty() {
				t.Fatal(err, tracked.Dirty())
			}
			if _, err := tracked.Commit(context.Background()); err != nil || patches.Load() != 1 {
				t.Fatal(err, patches.Load())
			}
		}
	})
}

func TestClusteringClusterProfileOnlyNullBoolCompatibilityAndLookupGate(t *testing.T) {
	t.Run("existing-public-pointer-is-authoritative", func(t *testing.T) {
		for _, enabled := range []bool{false, true} {
			t.Run(fmt.Sprint(enabled), func(t *testing.T) {
				cloud, client := clusterProfileOnlyNullCloud(t)
				var patches atomic.Int32
				want := fmt.Sprint(enabled)
				cloud.Mux.HandleFunc("PATCH "+clusterProfileOnlyNullPrefix+"/clusters/fixed", func(w http.ResponseWriter, r *http.Request) {
					patches.Add(1)
					clusterProfileOnlyNullWire(t, r, map[string]string{"profile_only": want})
					clusterProfileOnlyNullAccept(w)
				})
				api := clusters.New(client)
				original := enabled
				opts := clusters.UpdateOpts{ProfileOnly: &enabled}
				if _, err := api.Update(context.Background(), resource.ID("fixed"), opts); err != nil {
					t.Fatal(err)
				}
				// A public pointer assigned after JSON null remains an explicit bool.
				decoded := clusterProfileOnlyNullDecoded(t, `{"profile_only":null}`)
				decoded.ProfileOnly = &enabled
				if _, err := api.Update(context.Background(), resource.ID("fixed"), decoded, clusters.WithUpdateOptions(decoded)); err != nil {
					t.Fatal(err)
				}
				if enabled != original || opts.ProfileOnly != &enabled || patches.Load() != 2 {
					t.Fatal("existing bool pointer was coerced or mutated", enabled, opts.ProfileOnly, patches.Load())
				}
			})
		}
	})
	t.Run("null-helper-does-not-mutate-pointer-input", func(t *testing.T) {
		cloud, client := clusterProfileOnlyNullCloud(t)
		enabled := true
		opts := clusters.UpdateOpts{ProfileOnly: &enabled}
		cloud.Mux.HandleFunc("PATCH "+clusterProfileOnlyNullPrefix+"/clusters/fixed", func(w http.ResponseWriter, r *http.Request) {
			clusterProfileOnlyNullWire(t, r, map[string]string{"profile_only": "null"})
			clusterProfileOnlyNullAccept(w)
		})
		if _, err := clusters.New(client).Update(context.Background(), resource.ID("fixed"), opts, clusters.WithUpdateProfileOnlyNull()); err != nil || !enabled || opts.ProfileOnly != &enabled {
			t.Fatal(err, enabled, opts.ProfileOnly)
		}
	})
	t.Run("clean-commit-header-only-and-source-validation", func(t *testing.T) {
		cloud, client := clusterProfileOnlyNullCloud(t)
		tracked := clusterProfileOnlyNullTrack(t, client)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
		nullOpts := clusterProfileOnlyNullDecoded(t, `{"profile_only":null}`)
		for _, option := range []clusters.UpdateOption{clusters.WithUpdateProfileOnlyNull(), clusters.WithUpdateOptions(nullOpts), clusters.WithUpdateProfileOnly(false)} {
			if _, err := tracked.Commit(context.Background(), option); !errors.Is(err, resource.ErrInvalidOption) || tracked.Dirty() {
				t.Fatal("clean Commit accepted a body option", err, tracked.Dirty())
			}
		}
		client.Type = "compute"
		if _, err := tracked.Commit(context.Background()); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal("clean commit skipped source validation", err, calls.Load())
		}
	})
	t.Run("null-gate-before-name-lookup", func(t *testing.T) {
		cloud, client := clusterProfileOnlyNullCloud(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
		for _, version := range []string{"", "1.0", "1.5", "latest"} {
			client.Microversion = version
			if _, err := clusters.New(client).Update(context.Background(), resource.Name("Selected"), clusters.UpdateOpts{}, clusters.WithUpdateProfileOnlyNull()); !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 0 {
				t.Fatal(version, err, calls.Load())
			}
		}
	})
	t.Run("null-gate-rechecked-after-name-resolution", func(t *testing.T) {
		cloud, client := clusterProfileOnlyNullCloud(t)
		var lookups, patches atomic.Int32
		cloud.Mux.HandleFunc("GET "+clusterProfileOnlyNullPrefix+"/clusters", func(w http.ResponseWriter, r *http.Request) {
			lookups.Add(1)
			if r.URL.Query().Get("name") != "Selected" || r.Header.Get("OpenStack-API-Version") != "clustering 1.6" {
				t.Error(r.URL, r.Header)
			}
			testcloud.JSON(w, 200, `{"clusters":[{"id":"fixed","name":"Selected"}]}`)
		})
		cloud.Mux.HandleFunc("PATCH "+clusterProfileOnlyNullPrefix+"/clusters/fixed", func(w http.ResponseWriter, r *http.Request) { patches.Add(1); w.WriteHeader(500) })
		parent := client.ProviderClient.HTTPClient.Transport
		if parent == nil {
			parent = http.DefaultTransport
		}
		client.ProviderClient.HTTPClient.Transport = clusterRoundTrip(func(r *http.Request) (*http.Response, error) {
			response, err := parent.RoundTrip(r)
			if r.Method == http.MethodGet {
				client.Microversion = "1.5"
			}
			return response, err
		})
		if _, err := clusters.New(client).Update(context.Background(), resource.Name("Selected"), clusters.UpdateOpts{}, clusters.WithUpdateProfileOnlyNull()); !errors.Is(err, resource.ErrUnsupported) || lookups.Load() != 1 || patches.Load() != 0 {
			t.Fatal(err, lookups.Load(), patches.Load())
		}
	})
}
