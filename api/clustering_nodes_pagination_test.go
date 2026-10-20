package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestClusteringNodesListWireOptionsAndRawMarkerSnapshots(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/senlin/v1")
	var calls atomic.Int32
	var captured *request.Config[nodes.ListOpts]
	cloud.Mux.HandleFunc("GET /senlin/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		query := r.URL.Query()
		want := url.Values{
			"limit": {"5"}, "marker": {query.Get("marker")}, "name": {"selected"},
			"cluster_id": {"cluster-selector"}, "status": {"ACTIVE"}, "sort": {"vendor_key:desc"},
			"global_project": {"false"}, "show_details": {"false"}, "vendor": {"retained"},
		}
		if !reflect.DeepEqual(query, want) {
			t.Errorf("wire options or local filter leaked: got %v, want %v", query, want)
		}
		w.Header().Set("X-Request-ID", "node-page-evidence")
		switch query.Get("marker") {
		case "start":
			if r.Header.Get("X-Auth-Token") != "test-token" {
				t.Error("first page token", r.Header)
			}
			// Prepared query and filters must already own their inputs, even
			// while the response to the first request is still being produced.
			captured.Options.Limit = 1
			*captured.Options.GlobalProject = true
			captured.Query.Set("vendor", "changed-after-preparation")
			filters := captured.Arguments["nodes.local_filters"].(map[string]json.RawMessage)
			filters["metadata"][0] = '!'
			testcloud.JSON(w, 200, `{"nodes":[{"id":"wire-first","name":"selected","project":"project-a","metadata":{"quota":9007199254740992,"nested":{"team":"infra"}}}]}`)
		case "wire-first":
			testcloud.JSON(w, 200, `{"nodes":[{"id":"wire-second","name":"selected","physical_id":"physical","project":"project-a","index":9007199254740993,"metadata":{"quota":9007199254740993,"nested":{"team":"infra","extra":true}}}]}`)
		case "wire-second":
			if r.Header.Get("X-Auth-Token") != "refreshed-node-token" {
				t.Error("next page did not retain the live source token", r.Header)
			}
			testcloud.JSON(w, 200, `{"nodes":[]}`)
		default:
			t.Error("marker came from changed input or yielded model", query)
			testcloud.JSON(w, 200, `{"nodes":[]}`)
		}
	})
	global, details := false, false
	typed := nodes.ListOpts{Limit: 5, Marker: "start", Name: "selected", ClusterID: "cluster-selector", Status: "ACTIVE", Sort: "vendor_key:desc", GlobalProject: &global, ShowDetails: &details}
	nested := map[string]any{"team": "infra"}
	metadata := map[string]any{"quota": json.Number("9007199254740993"), "nested": nested}
	options := []nodes.ListOption{
		nodes.WithListOptions(typed), nodes.WithListQuery("vendor", "retained"),
		nodes.WithListFilter("project_id", "project-a"), nodes.WithListFilter("metadata", metadata),
		func(config *request.Config[nodes.ListOpts]) error { captured = config; return nil },
	}
	api := nodes.New(client)
	stream := api.List(context.Background(), options...)
	if calls.Load() != 0 {
		t.Fatal("List was eager", calls.Load())
	}
	typed.Limit, global, details = 1, true, true
	nested["team"], metadata["quota"] = "changed", json.Number("0")
	options[0] = nodes.WithListOptions(nodes.ListOpts{Limit: 1})
	for iteration := range 2 {
		client.SetToken("test-token")
		var found int
		for value, err := range stream {
			if err != nil {
				t.Fatal(err)
			}
			found++
			if value.ID != "wire-second" || value.Index == nil || value.Index.String() != "9007199254740993" || value.Header.Get("X-Request-ID") != "node-page-evidence" || value.StatusCode != 200 {
				t.Fatal("filtered list lost exact response evidence", value)
			}
			value.ID, value.Body["id"] = "consumer-changed", json.RawMessage(`"consumer-changed"`)
			client.SetToken("refreshed-node-token")
		}
		if found != 1 || calls.Load() != int32((iteration+1)*3) {
			t.Fatal("short/filtered page stopped pagination or iterator was not reusable", found, calls.Load())
		}
	}
}

func TestClusteringNodesFindAcrossLinkedPages(t *testing.T) {
	for _, mode := range []string{"unique", "duplicate", "late-error"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /senlin/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("name") != "selected" || r.URL.Query().Get("limit") != "" {
					t.Error("Find lost its name query or invented a page limit", r.URL)
				}
				switch r.URL.Query().Get("marker") {
				case "":
					testcloud.JSON(w, 200, `{"nodes":[{"id":"near","name":"selected-other"}],"nodes_links":[{"rel":"next","href":"?marker=second"}]}`)
				case "second":
					testcloud.JSON(w, 200, `{"nodes":[{"id":"found-one","name":"selected"}],"next":"?marker=third"}`)
				case "third":
					switch mode {
					case "duplicate":
						testcloud.JSON(w, 200, `{"nodes":[{"id":"found-two","name":"selected"}]}`)
					case "late-error":
						testcloud.JSON(w, 403, `{"error":"later node page denied"}`)
					default:
						testcloud.JSON(w, 200, `{"nodes":[]}`)
					}
				default:
					t.Error(r.URL)
					testcloud.JSON(w, 200, `{"nodes":[]}`)
				}
			})
			value, err := nodes.New(cloud.Client("clustering", "/senlin/v1")).Find(context.Background(), resource.Name("selected"))
			switch mode {
			case "unique":
				if err != nil || value == nil || value.ID != "found-one" {
					t.Fatal(value, err)
				}
			case "duplicate":
				var ambiguous *resource.AmbiguousError
				if value != nil || !errors.As(err, &ambiguous) || !reflect.DeepEqual(ambiguous.IDs, []string{"found-one", "found-two"}) {
					t.Fatal(value, err, ambiguous)
				}
			case "late-error":
				if value != nil || !gophercloud.ResponseCodeIs(err, 403) || !strings.Contains(err.Error(), "later node page denied") {
					t.Fatal("Find returned a partial match or ignored a later nonmissing error", value, err)
				}
			}
			if calls.Load() != 3 {
				t.Fatal("Find did not inspect all linked name pages", calls.Load())
			}
		})
	}
}

func TestClusteringNodesListPreflightAndSharedStatus(t *testing.T) {
	t.Run("preflight", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			t.Error("invalid list reached HTTP", r.URL)
		})
		api := nodes.New(cloud.Client("clustering", "/senlin/v1"))
		for _, key := range []string{"limit", "marker", "name", "cluster_id", "status", "sort", "global_project", "show_details", "metadata", "project_id", "details"} {
			if _, err := api.All(context.Background(), nodes.WithListQuery(key, "override")); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("concrete/local query was not protected", key, err)
			}
		}
		cases := []struct {
			name   string
			option nodes.ListOption
		}{
			{"nil-option", nil},
			{"negative-limit", nodes.WithListOptions(nodes.ListOpts{Limit: -1})},
			{"invalid-marker", nodes.WithListOptions(nodes.ListOpts{Marker: "bad/marker"})},
			{"invalid-sort", nodes.WithListOptions(nodes.ListOpts{Sort: "name:wrong"})},
			{"unknown-filter", nodes.WithListFilter("vendor", true)},
			{"filter-json-error", nodes.WithListFilter("metadata", make(chan int))},
			{"unsupported-fields", request.WithField[nodes.ListOpts]("vendor", true)},
			{"protected-header", request.WithHeader[nodes.ListOpts]("X-Auth-Token", "ignored")},
			{"unsupported-argument", request.WithArgument[nodes.ListOpts]("vendor", true)},
			{"wrong-filter-argument", request.WithArgument[nodes.ListOpts]("nodes.local_filters", true)},
			{"invalid-filter-json", request.WithArgument[nodes.ListOpts]("nodes.local_filters", map[string]json.RawMessage{"metadata": json.RawMessage(`{`)})},
		}
		for _, test := range cases {
			t.Run(test.name, func(t *testing.T) {
				if _, err := api.All(context.Background(), test.option); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			})
		}
		if calls.Load() != 0 {
			t.Fatal(calls.Load())
		}
	})
	t.Run("shared-status", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("GET /senlin/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.URL.Query().Get("status") != "ACTIVE" || r.URL.Query().Get("sort") != "vendor_key:desc" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, `{"nodes":[{"id":"active-node","status":"active"},{"id":"failed-node","status":"ERROR"}]}`)
		})
		api := nodes.New(cloud.Client("clustering", "/senlin/v1"))
		values, err := api.Resources.All(context.Background(), resource.WithStatus("ACTIVE"), resource.WithQuery("sort", "vendor_key:desc"))
		if err != nil || len(values) != 1 || values[0].ID != "active-node" || calls.Load() != 1 {
			t.Fatal(values, err, calls.Load())
		}
		if _, err := api.Resources.All(context.Background(), resource.WithQuery("sort", "name:wrong")); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
			t.Fatal("shared list bypassed node sort validation", err, calls.Load())
		}
	})
	for _, mode := range []string{"source-type", "source-version-header", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("clustering", "/senlin/v1")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /senlin/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, `{"nodes":[{"id":"first"}],"next":"?marker=second"}`)
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var resultErr error
			for _, err := range nodes.New(client).List(ctx) {
				if err != nil {
					resultErr = err
					break
				}
				switch mode {
				case "source-type":
					client.Type = "network"
				case "source-version-header":
					client.Microversion = "1.12"
					client.MoreHeaders = map[string]string{"OpenStack-API-Version": "clustering 1.13"}
				case "cancel":
					cancel()
				}
			}
			want := resource.ErrInvalidOption
			if mode == "cancel" {
				want = context.Canceled
			}
			if !errors.Is(resultErr, want) || calls.Load() != 1 {
				t.Fatal("next HTTP preceded source/context validation", resultErr, calls.Load())
			}
		})
	}
}
