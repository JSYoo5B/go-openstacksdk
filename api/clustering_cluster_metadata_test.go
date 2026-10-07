package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/clustering/v1/clusters"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func clusterMetadataClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	return cloud.Client("clustering", "/reverse/senlin/v1")
}

func TestClusteringClusterMetadataReadViewsPreservePresenceAndRawAuthority(t *testing.T) {
	for _, mode := range []string{"omitted", "null", "empty", "values"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			body := `{"cluster":{"id":"incidental","Metadata":{"spoof":true},"future":null}}`
			if mode == "null" {
				body = `{"cluster":{"id":"incidental","metadata":null,"Metadata":{"spoof":true}}}`
			} else if mode == "empty" {
				body = `{"cluster":{"id":"incidental","metadata":{}}}`
			} else if mode == "values" {
				body = `{"cluster":{"id":"incidental","metadata":{"big":9007199254740993,"decimal":1.234567890123456789,"nullable":null,"nested":{"x":[false,9007199254740993]}},"Metadata":{"spoof":true}}}`
			}
			cloud.Mux.HandleFunc("GET /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != "" {
					t.Error(r.URL, r.Header)
				}
				w.Header().Set("Location", "https://foreign.example/actions/incidental")
				w.Header().Set("X-Request-ID", "metadata-read")
				testcloud.JSON(w, 200, body)
			})
			api := clusters.New(clusterMetadataClient(cloud))
			for _, read := range []func(context.Context, resource.Ref) (*clusters.MetadataView, error){api.FetchMetadata, api.GetMetadata} {
				view, err := read(context.Background(), resource.ID("selected"))
				if err != nil || view.Present != (mode != "omitted") || view.Null != (mode == "null") || view.Cluster.ID != "incidental" || view.Cluster.StatusCode != 200 || view.Cluster.Header.Get("X-Request-ID") != "metadata-read" || view.Cluster.Operation != nil {
					t.Fatal(view, err)
				}
				if mode == "values" {
					if string(view.Values["big"]) != "9007199254740993" || string(view.Values["decimal"]) != "1.234567890123456789" || string(view.Values["nullable"]) != "null" || len(view.Values) != 4 {
						t.Fatal(view.Values)
					}
					view.Values["big"][0] = '0'
					if string(view.Cluster.UserMetadata["big"]) != "9007199254740993" || !strings.Contains(string(view.Cluster.Body["metadata"]), "9007199254740993") {
						t.Fatal("view values alias cluster evidence", view.Cluster)
					}
				} else if len(view.Values) != 0 || len(view.Cluster.UserMetadata) != 0 {
					t.Fatal("casefold metadata replaced canonical raw field", view)
				}
				if mode == "empty" && view.Values == nil {
					t.Fatal("empty object became null")
				}
			}
			if calls.Load() != 2 {
				t.Fatal("read followed Location or cached the alias", calls.Load())
			}
		})
	}
}

func TestClusteringClusterMetadataShallowMergeSnapshotsAndQueuedResult(t *testing.T) {
	cloud := testcloud.New(t)
	client := clusterMetadataClient(cloud)
	client.Microversion = "1.13"
	var gets, patches atomic.Int32
	nested := map[string]any{"replacement": false}
	input := map[string]any{"nested": nested, "nullable": nil, "new": json.Number("9007199254740995")}
	var captured *request.Config[clusters.MetadataOpts]
	option := func(config *request.Config[clusters.MetadataOpts]) error {
		captured = config
		config.Headers["X-Trace"] = "prepared"
		return nil
	}
	cloud.Mux.HandleFunc("GET /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if r.Header.Get("X-Trace") != "prepared" || r.Header.Get("OpenStack-API-Version") != "clustering 1.13" {
			t.Error(r.Header)
		}
		input["new"] = "changed"
		nested["replacement"] = true
		captured.Headers["X-Trace"] = "changed"
		cloud.Provider.SetToken("updated-token")
		testcloud.JSON(w, 200, `{"cluster":{"id":"incidental-get-id","metadata":{"kept":9007199254740993,"nested":{"original":true},"nullable":"old","future":[null,1.234567890123456789]}}}`)
	})
	var sent map[string]json.RawMessage
	responseBody := `{"cluster":{"id":"incidental-patch-id","metadata":{"response":"not requested state"},"status":"ACTIVE"}}`
	cloud.Mux.HandleFunc("PATCH /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
		patches.Add(1)
		if r.Header.Get("X-Trace") != "prepared" || r.Header.Get("X-Auth-Token") != "updated-token" || r.Header.Get("OpenStack-API-Version") != "clustering 1.13" {
			t.Error(r.Header)
		}
		var envelope struct {
			Cluster struct {
				Metadata map[string]json.RawMessage `json:"metadata"`
			} `json:"cluster"`
		}
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Error(err)
		}
		sent = envelope.Cluster.Metadata
		w.Header().Set("Location", "/reverse/senlin/v1/actions/queued")
		w.Header().Set("X-Request-ID", "queued-proof")
		testcloud.JSON(w, 202, responseBody)
	})
	result, err := clusters.New(client).SetMetadata(context.Background(), resource.ID("selected"), input, option)
	if err != nil || !result.Changed || result.Operation == nil || result.Operation.ActionID != "queued" || result.Operation.StatusCode != 202 || string(result.Operation.Body) != responseBody || result.ResponseCluster.ID != "incidental-patch-id" || string(result.ResponseCluster.UserMetadata["response"]) != `"not requested state"` {
		t.Fatal(result, err)
	}
	if string(sent["kept"]) != "9007199254740993" || string(sent["new"]) != "9007199254740995" || string(sent["nullable"]) != "null" || string(sent["nested"]) != `{"replacement":false}` || string(sent["future"]) != `[null,1.234567890123456789]` {
		t.Fatal(sent)
	}
	if string(result.Previous["nested"]) != `{"original":true}` || string(result.Requested["nested"]) != `{"replacement":false}` || !result.PreviousPresent || result.PreviousNull || gets.Load() != 1 || patches.Load() != 1 {
		t.Fatal(result, gets.Load(), patches.Load())
	}
	result.Requested["kept"][0] = '0'
	result.Operation.Header.Set("X-Request-ID", "consumer")
	result.Operation.Body[0] = '['
	if string(result.Previous["kept"]) != "9007199254740993" || result.ResponseCluster.Operation.Header.Get("X-Request-ID") != "queued-proof" || string(result.ResponseCluster.Operation.Body) != responseBody {
		t.Fatal("returned snapshots alias each other", result)
	}
}

func TestClusteringClusterMetadataNameResolutionUsesCanonicalRawFields(t *testing.T) {
	for _, mode := range []string{"success", "spoof-name", "missing-id", "null-id", "duplicate"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, gets, patches atomic.Int32
			listBody := `{"clusters":[{"id":"other","name":"other","NAME":"selected"},{"id":"canonical","ID":"spoof-id","name":"selected","NAME":"other"}]}`
			if mode == "spoof-name" {
				listBody = `{"clusters":[{"id":"other","NAME":"selected"}]}`
			} else if mode == "missing-id" {
				listBody = `{"clusters":[{"name":"selected","ID":"spoof-id"}]}`
			} else if mode == "null-id" {
				listBody = `{"clusters":[{"id":null,"ID":"spoof-id","name":"selected"}]}`
			} else if mode == "duplicate" {
				listBody = `{"clusters":[{"id":"one","name":"selected"},{"id":"two","name":"selected"}]}`
			}
			cloud.Mux.HandleFunc("GET /reverse/senlin/v1/clusters", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("name") != "selected" {
					t.Error(r.URL)
				}
				w.Header().Set("X-Request-ID", "name-proof")
				testcloud.JSON(w, 200, listBody)
			})
			cloud.Mux.HandleFunc("GET /reverse/senlin/v1/clusters/canonical", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 200, `{"cluster":{"id":"incidental","metadata":{"keep":true}}}`)
			})
			cloud.Mux.HandleFunc("PATCH /reverse/senlin/v1/clusters/canonical", func(w http.ResponseWriter, r *http.Request) {
				patches.Add(1)
				w.Header().Set("Location", "/reverse/senlin/v1/actions/queued")
				testcloud.JSON(w, 202, `{"cluster":{"metadata":{}}}`)
			})
			result, err := clusters.New(clusterMetadataClient(cloud)).SetMetadata(context.Background(), resource.Name("selected"), map[string]any{"new": false})
			if mode == "success" {
				if err != nil || !result.Changed || gets.Load() != 1 || patches.Load() != 1 {
					t.Fatal(result, err, gets.Load(), patches.Load())
				}
			} else {
				if result != nil || err == nil || gets.Load() != 0 || patches.Load() != 0 {
					t.Fatal(result, err, gets.Load(), patches.Load())
				}
				if mode == "spoof-name" && !errors.Is(err, resource.ErrNotFound) || mode == "duplicate" && !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(err)
				}
				if mode == "missing-id" || mode == "null-id" {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || string(proof.Body) != listBody || proof.StatusCode != 200 || proof.Header.Get("X-Request-ID") != "name-proof" {
						t.Fatal(err, proof)
					}
				}
			}
			if lists.Load() != 1 {
				t.Fatal(lists.Load())
			}
		})
	}
}

func TestClusteringClusterMetadataNoOpsUseExactJSONEquality(t *testing.T) {
	type fixture struct {
		name     string
		metadata string
		values   map[string]any
		delete   bool
		keys     []string
		gets     int32
		patches  int32
	}
	fixtures := []fixture{
		{name: "empty-set", metadata: `{"keep":true}`, values: map[string]any{}, gets: 1},
		{name: "nil-set", metadata: `{"keep":true}`, gets: 1},
		{name: "equal-numbers-object-order", metadata: `{"nested":{"b":[1.0,null],"a":9007199254740993},"zero":-0e99}`, values: map[string]any{"nested": json.RawMessage(`{"a":9007199254740993.0,"b":[1e0,null]}`), "zero": 0}, gets: 1},
		{name: "bool-number-distinct", metadata: `{"flag":false}`, values: map[string]any{"flag": 0}, gets: 1, patches: 1},
		{name: "nested-missing-null-distinct", metadata: `{"object":{"old":null}}`, values: map[string]any{"object": map[string]any{"new": nil}}, gets: 1, patches: 1},
		{name: "empty-selected-delete", delete: true, keys: []string{}, gets: 0},
		{name: "missing-selected-delete", metadata: `{"keep":true}`, delete: true, keys: []string{"missing"}, gets: 1},
		{name: "clear-null", metadata: `null`, delete: true, gets: 1},
		{name: "clear-omitted", delete: true, gets: 1},
		{name: "clear-empty", metadata: `{}`, delete: true, gets: 1},
		{name: "clear-values", metadata: `{"keep":true}`, delete: true, gets: 1, patches: 1},
	}
	for _, item := range fixtures {
		t.Run(item.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, patches atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				metadata := ""
				if item.metadata != "" {
					metadata = `,"metadata":` + item.metadata
				}
				testcloud.JSON(w, 200, `{"cluster":{"id":"incidental"`+metadata+`}}`)
			})
			cloud.Mux.HandleFunc("PATCH /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
				patches.Add(1)
				if item.name == "clear-values" {
					body, _ := io.ReadAll(r.Body)
					if string(body) != `{"cluster":{"metadata":{}}}` {
						t.Error(string(body))
					}
				}
				w.Header().Set("Location", "/reverse/senlin/v1/actions/queued")
				testcloud.JSON(w, 202, `{"cluster":{}}`)
			})
			api := clusters.New(clusterMetadataClient(cloud))
			var result *clusters.MetadataResult
			var err error
			if item.delete {
				result, err = api.DeleteMetadata(context.Background(), resource.ID("selected"), item.keys)
			} else {
				result, err = api.SetMetadata(context.Background(), resource.ID("selected"), item.values)
			}
			if err != nil || result.Changed != (item.patches > 0) || gets.Load() != item.gets || patches.Load() != item.patches {
				t.Fatal(result, err, gets.Load(), patches.Load())
			}
			if item.gets == 0 && (result.ResponseCluster != nil || result.Previous != nil || result.Requested != nil) {
				t.Fatal("zero-HTTP result fabricated an observation", result)
			}
			if item.gets > 0 && item.patches == 0 && (result.Operation != nil || result.ResponseCluster.StatusCode != 200) {
				t.Fatal(result)
			}
			if item.name == "clear-null" && (!result.PreviousPresent || !result.PreviousNull) || item.name == "clear-omitted" && (result.PreviousPresent || result.PreviousNull) {
				t.Fatal(result)
			}
		})
	}
}

func TestClusteringClusterMetadataSelectedDeleteIsOneOwnedReplacement(t *testing.T) {
	cloud := testcloud.New(t)
	keys := []string{"remove", "slash/key", "", "remove"}
	var gets, patches atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		keys[0], keys[1], keys[2] = "keep", "keep", "keep"
		testcloud.JSON(w, 200, `{"cluster":{"metadata":{"remove":1,"slash/key":null,"":false,"keep":{"precise":9007199254740993}}}}`)
	})
	cloud.Mux.HandleFunc("PATCH /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
		patches.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"cluster":{"metadata":{"keep":{"precise":9007199254740993}}}}` || r.Header.Get("X-Trace") != "batch" {
			t.Error(string(body), r.Header)
		}
		w.Header().Set("Location", "/reverse/senlin/v1/actions/queued")
		testcloud.JSON(w, 202, `{"cluster":{"metadata":{}}}`)
	})
	result, err := clusters.New(clusterMetadataClient(cloud)).DeleteMetadata(context.Background(), resource.ID("selected"), keys, clusters.WithMetadataHeader("X-Trace", "batch"))
	if err != nil || !result.Changed || len(result.Requested) != 1 || len(result.Previous) != 4 || gets.Load() != 1 || patches.Load() != 1 {
		t.Fatal(result, err, gets.Load(), patches.Load())
	}
}

func TestClusteringClusterMetadataPreflightRejectsUnsupportedInputs(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Error("invalid input reached HTTP", r.Method, r.URL)
	})
	client := clusterMetadataClient(cloud)
	api := clusters.New(client)
	options := []clusters.MetadataOption{
		nil,
		clusters.WithMetadataHeader("X-Auth-Token", "override"),
		clusters.WithMetadataHeader("openstack-api-version", "clustering 1.13"),
		clusters.WithMetadataHeader("Host", "other"),
		clusters.WithMetadataHeader("Content-Type", "other"),
		clusters.WithMetadataHeader("X-Trace", "bad\nvalue"),
		request.WithField[clusters.MetadataOpts]("metadata", map[string]any{}),
		request.WithQuery[clusters.MetadataOpts]("vendor", "value"),
		request.WithArgument[clusters.MetadataOpts]("vendor", true),
		func(config *request.Config[clusters.MetadataOpts]) error {
			config.Headers["bad header"] = "value"
			return nil
		},
	}
	for _, option := range options {
		if _, err := api.SetMetadata(context.Background(), resource.ID("selected"), map[string]any{"key": true}, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
		if _, err := api.DeleteMetadata(context.Background(), resource.Name("selected"), []string{}, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	for _, values := range []map[string]any{{"value": make(chan int)}, {"value": math.NaN()}, {"value": json.RawMessage(`{`)}} {
		if _, err := api.SetMetadata(context.Background(), resource.Name("selected"), values); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	for _, ref := range []resource.Ref{resource.ID(""), resource.ID("bad/id"), resource.Name(" ")} {
		if _, err := api.FetchMetadata(context.Background(), ref); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
		if _, err := api.DeleteMetadata(context.Background(), ref, []string{}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	for _, invalid := range []*clusters.API{nil, clusters.New(nil), clusters.New(&gophercloud.ServiceClient{}), clusters.New(cloud.Client("compute", "/reverse/senlin/v1"))} {
		if _, err := invalid.FetchMetadata(context.Background(), resource.ID("selected")); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	client.Microversion = "latest"
	if _, err := api.FetchMetadata(context.Background(), resource.ID("selected")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	client.Microversion = "1.01"
	if _, err := api.DeleteMetadata(context.Background(), resource.ID("selected"), []string{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	client.Microversion = ""
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.SetMetadata(ctx, resource.ID("selected"), nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringClusterMetadataRevalidatesSourceBeforeEveryRequest(t *testing.T) {
	for _, mode := range []string{"after-name", "after-get", "version-after-get", "same-value-after-get"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := clusterMetadataClient(cloud)
			var lists, gets, patches atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/senlin/v1/clusters", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				client.Type = "compute"
				testcloud.JSON(w, 200, `{"clusters":[{"id":"canonical","name":"selected"}]}`)
			})
			cloud.Mux.HandleFunc("GET /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if mode == "version-after-get" {
					client.Microversion = "latest"
				} else {
					client.Type = "compute"
				}
				testcloud.JSON(w, 200, `{"cluster":{"metadata":{"keep":true}}}`)
			})
			cloud.Mux.HandleFunc("PATCH /reverse/senlin/v1/clusters/{identity}", func(w http.ResponseWriter, r *http.Request) { patches.Add(1) })
			ref := resource.ID("selected")
			if mode == "after-name" {
				ref = resource.Name("selected")
			}
			values := map[string]any{"new": true}
			if mode == "same-value-after-get" {
				values = map[string]any{}
			}
			result, err := clusters.New(client).SetMetadata(context.Background(), ref, values)
			if result != nil || err == nil || patches.Load() != 0 {
				t.Fatal(result, err, patches.Load())
			}
			if mode == "after-name" && (lists.Load() != 1 || gets.Load() != 0) || mode != "after-name" && (lists.Load() != 0 || gets.Load() != 1) {
				t.Fatal(lists.Load(), gets.Load())
			}
			if mode == "version-after-get" && !errors.Is(err, resource.ErrUnsupported) || mode != "version-after-get" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestClusteringClusterMetadataZeroHTTPDeleteRevalidatesOptionSideEffects(t *testing.T) {
	for _, mode := range []string{"source", "context"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := clusterMetadataClient(cloud)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			option := func(config *request.Config[clusters.MetadataOpts]) error {
				if mode == "source" {
					client.Type = "compute"
				} else {
					cancel()
				}
				return nil
			}
			result, err := clusters.New(client).DeleteMetadata(ctx, resource.Name("selected"), []string{}, option)
			if result != nil || mode == "source" && !errors.Is(err, resource.ErrInvalidOption) || mode == "context" && !errors.Is(err, context.Canceled) {
				t.Fatal(result, err)
			}
		})
	}
}

func TestClusteringClusterMetadataReadFailuresPreserveWholeResponse(t *testing.T) {
	for _, code := range []int{200, 403, 404} {
		bodies := []string{`{`, `{"cluster":null}`, `{"cluster":{"metadata":[]}}`, `{"cluster":{"metadata":false}}`, `{"cluster":{"metadata":"text"}}`}
		if code != 200 {
			bodies = []string{`{"error":"server failure"}`}
		}
		for _, body := range bodies {
			t.Run(fmt.Sprint(code)+body, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("X-Request-ID", "read-failure")
					testcloud.JSON(w, code, body)
				})
				result, err := clusters.New(clusterMetadataClient(cloud)).SetMetadata(context.Background(), resource.ID("selected"), map[string]any{"new": true})
				if result != nil || calls.Load() != 1 {
					t.Fatal(result, err, calls.Load())
				}
				var proof *resource.ResponseError
				if code == 200 {
					if !errors.As(err, &proof) || string(proof.Body) != body || proof.Header.Get("X-Request-ID") != "read-failure" || proof.StatusCode != code {
						t.Fatal(err, proof)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || string(native.Body) != body || native.ResponseHeader.Get("X-Request-ID") != "read-failure" || !gophercloud.ResponseCodeIs(err, code) || errors.As(err, &proof) {
						t.Fatal(err)
					}
					if code == 404 && !errors.Is(err, resource.ErrNotFound) {
						t.Fatal(err)
					}
				}
			})
		}
	}
}

func TestClusteringClusterMetadataAcceptedFailuresNeverResendOrFollowLocation(t *testing.T) {
	for _, mode := range []string{"bad-json", "bad-envelope", "bad-metadata", "missing-location", "foreign-location", "multiple-location", "wrong-code", "server-error"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, patches atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 200, `{"cluster":{"metadata":{"keep":true}}}`)
			})
			body := `{"cluster":{"metadata":{}}}`
			code := 202
			cloud.Mux.HandleFunc("PATCH /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
				patches.Add(1)
				location := "/reverse/senlin/v1/actions/queued"
				switch mode {
				case "bad-json":
					body = `{`
				case "bad-envelope":
					body = `{"cluster":null}`
				case "bad-metadata":
					body = `{"cluster":{"metadata":[]}}`
				case "missing-location":
					location = ""
				case "foreign-location":
					location = "https://foreign.example/actions/queued"
				case "multiple-location":
					w.Header().Add("Location", location)
				case "wrong-code":
					code = 200
				case "server-error":
					code, body = 409, `{"error":"conflict"}`
				}
				if location != "" {
					w.Header().Add("Location", location)
				}
				w.Header().Set("X-Request-ID", "accepted-failure")
				testcloud.JSON(w, code, body)
			})
			result, err := clusters.New(clusterMetadataClient(cloud)).SetMetadata(context.Background(), resource.ID("selected"), map[string]any{"new": true})
			if result != nil || gets.Load() != 1 || patches.Load() != 1 {
				t.Fatal(result, err, gets.Load(), patches.Load())
			}
			var proof *resource.ResponseError
			if code == 202 {
				if !errors.As(err, &proof) || string(proof.Body) != body || proof.StatusCode != 202 || proof.Header.Get("X-Request-ID") != "accepted-failure" {
					t.Fatal(err, proof)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || !gophercloud.ResponseCodeIs(err, code) || string(native.Body) != body || errors.As(err, &proof) {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestClusteringClusterMetadataReadModifyWriteDoesNotClaimConcurrencyOrCompletion(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, patches atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
		get := gets.Add(1)
		body := `{"cluster":{"metadata":{"keep":true}}}`
		if get > 1 {
			body = `{"cluster":{"metadata":{"keep":true,"concurrent":9007199254740993}}}`
		}
		testcloud.JSON(w, 200, body)
	})
	cloud.Mux.HandleFunc("PATCH /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
		patches.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"cluster":{"metadata":{"keep":true,"new":false}}}` || r.Header.Get("If-Match") != "" {
			t.Error(string(body), r.Header)
		}
		w.Header().Set("Location", "/reverse/senlin/v1/actions/queued")
		testcloud.JSON(w, 202, `{"cluster":{"metadata":{"keep":true,"concurrent":9007199254740993}}}`)
	})
	api := clusters.New(clusterMetadataClient(cloud))
	result, err := api.SetMetadata(context.Background(), resource.ID("selected"), map[string]any{"new": false})
	if err != nil || !result.Changed || result.Operation == nil || gets.Load() != 1 || patches.Load() != 1 || result.Requested["concurrent"] != nil || result.ResponseCluster.UserMetadata["new"] != nil {
		t.Fatal(result, err, gets.Load(), patches.Load())
	}
	observed, err := api.FetchMetadata(context.Background(), resource.ID("selected"))
	if err != nil || string(observed.Values["concurrent"]) != "9007199254740993" || observed.Values["new"] != nil || gets.Load() != 2 || patches.Load() != 1 {
		t.Fatal(observed, err, gets.Load(), patches.Load())
	}
}

type clusterMetadataRoundTripper func(*http.Request) (*http.Response, error)

func (transport clusterMetadataRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return transport(r)
}

type clusterMetadataCancelReader struct {
	reader io.Reader
	cancel context.CancelFunc
}

func (reader *clusterMetadataCancelReader) Read(buffer []byte) (int, error) {
	n, err := reader.reader.Read(buffer)
	reader.cancel()
	return n, err
}

func (*clusterMetadataCancelReader) Close() error { return nil }

func TestClusteringClusterMetadataCancellationAfterAcceptanceRetainsEvidence(t *testing.T) {
	cloud := testcloud.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var gets, patches atomic.Int32
	body := `{"cluster":{"metadata":{"new":true}}}`
	cloud.Provider.HTTPClient.Transport = clusterMetadataRoundTripper(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/reverse/senlin/v1/clusters/selected" {
			t.Fatal(r.Method, r.URL)
		}
		if r.Method == http.MethodGet {
			gets.Add(1)
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"cluster":{"metadata":{}}}`)), Request: r}, nil
		}
		patches.Add(1)
		return &http.Response{StatusCode: 202, Header: http.Header{"Location": {"/reverse/senlin/v1/actions/queued"}, "X-Request-Id": {"cancelled-acceptance"}}, Body: &clusterMetadataCancelReader{reader: strings.NewReader(body), cancel: cancel}, Request: r}, nil
	})
	result, err := clusters.New(clusterMetadataClient(cloud)).SetMetadata(ctx, resource.ID("selected"), map[string]any{"new": true})
	var proof *resource.ResponseError
	if result != nil || !errors.Is(err, context.Canceled) || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != body || proof.Header.Get("X-Request-ID") != "cancelled-acceptance" || gets.Load() != 1 || patches.Load() != 1 {
		t.Fatal(result, err, proof, gets.Load(), patches.Load())
	}
}
