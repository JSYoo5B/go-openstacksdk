package api_test

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/policytypes"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/services"
	"github.com/JSYoo5B/go-openstacksdk/internal/senlin"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type catalogControlFixture struct {
	path, key string
	row       func(string) string
	list      func(context.Context, *gophercloud.ServiceClient, ...senlin.ListOption) iter.Seq2[string, error]
	max       func(int) senlin.ListOption
	paginated func(bool) senlin.ListOption
	snapshot  func(senlin.ListOpts) senlin.ListOption
	query     func(string, string) senlin.ListOption
}

func catalogControlFixtures() []catalogControlFixture {
	return []catalogControlFixture{
		{path: "profile-types", key: "profile_types", row: func(id string) string { return fmt.Sprintf(`{"name":%q}`, id) },
			max: profiletypes.WithListMaxItems, paginated: profiletypes.WithListPaginated, snapshot: profiletypes.WithListOptions, query: profiletypes.WithListQuery,
			list: func(ctx context.Context, client *gophercloud.ServiceClient, options ...senlin.ListOption) iter.Seq2[string, error] {
				return func(yield func(string, error) bool) {
					for value, err := range profiletypes.New(client).List(ctx, options...) {
						id := ""
						if value != nil {
							id = value.Name
						}
						if !yield(id, err) {
							return
						}
					}
				}
			}},
		{path: "policy-types", key: "policy_types", row: func(id string) string { return fmt.Sprintf(`{"name":%q}`, id) },
			max: policytypes.WithListMaxItems, paginated: policytypes.WithListPaginated, snapshot: policytypes.WithListOptions, query: policytypes.WithListQuery,
			list: func(ctx context.Context, client *gophercloud.ServiceClient, options ...senlin.ListOption) iter.Seq2[string, error] {
				return func(yield func(string, error) bool) {
					for value, err := range policytypes.New(client).List(ctx, options...) {
						id := ""
						if value != nil {
							id = value.Name
						}
						if !yield(id, err) {
							return
						}
					}
				}
			}},
		{path: "services", key: "services", row: func(id string) string { return fmt.Sprintf(`{"id":%q,"status":"UP"}`, id) },
			max: services.WithListMaxItems, paginated: services.WithListPaginated, snapshot: services.WithListOptions, query: services.WithListQuery,
			list: func(ctx context.Context, client *gophercloud.ServiceClient, options ...senlin.ListOption) iter.Seq2[string, error] {
				return func(yield func(string, error) bool) {
					for value, err := range services.New(client).List(ctx, options...) {
						id := ""
						if value != nil {
							id = value.ID
						}
						if !yield(id, err) {
							return
						}
					}
				}
			}},
	}
}

func TestClusteringCatalogListControlsHintsCapsAndSnapshots(t *testing.T) {
	for _, fixture := range catalogControlFixtures() {
		for _, mode := range []string{"cap-hint", "explicit-limit", "single-page-snapshot", "last-wins", "zero-unbounded"} {
			t.Run(fixture.path+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("clustering", "/senlin/v1")
				client.Microversion = "1.7"
				var calls atomic.Int32
				var captured *request.Config[senlin.ListOpts]
				var options []senlin.ListOption
				wantLimit, wantRows, wantCalls := "2", 2, int32(1)
				switch mode {
				case "cap-hint":
					options = []senlin.ListOption{fixture.max(2)}
				case "explicit-limit":
					options = []senlin.ListOption{fixture.snapshot(senlin.ListOpts{Limit: 7, MaxItems: 2})}
					wantLimit = "7"
				case "single-page-snapshot":
					paginated := false
					options = []senlin.ListOption{fixture.snapshot(senlin.ListOpts{Paginated: &paginated})}
					paginated = true
					wantLimit, wantRows = "", 3
				case "last-wins":
					options = []senlin.ListOption{fixture.max(1), fixture.max(2), fixture.paginated(false), fixture.paginated(true)}
					wantCalls = 2
				case "zero-unbounded":
					options = []senlin.ListOption{fixture.max(1), fixture.max(0)}
					wantLimit, wantRows, wantCalls = "", 4, 2
				}
				options = append(options, fixture.query("vendor", "retained"), func(config *request.Config[senlin.ListOpts]) error { captured = config; return nil })
				cloud.Mux.HandleFunc("GET /senlin/v1/"+fixture.path, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					query := r.URL.Query()
					if query.Get("limit") != wantLimit || query.Get("vendor") != "retained" || query.Has("max_items") || query.Has("paginated") || r.Header.Get("OpenStack-API-Version") != "clustering 1.7" {
						t.Error(query, r.Header)
					}
					captured.Options.MaxItems = 99
					if captured.Options.Paginated != nil {
						*captured.Options.Paginated = true
					}
					captured.Query.Set("vendor", "mutated")
					rows := fixture.row("first") + "," + fixture.row("second") + "," + fixture.row("third")
					if query.Get("marker") == "" {
						w.Header().Set("Link", fmt.Sprintf(`</senlin/v1/%s?marker=next>; rel="next"`, fixture.path))
						if mode == "last-wins" {
							rows = fixture.row("first")
						}
					} else {
						if query.Get("marker") != "next" {
							t.Error(query)
						}
						rows = fixture.row("last")
					}
					testcloud.JSON(w, 200, `{"`+fixture.key+`":[`+rows+`]}`)
				})
				iterator := fixture.list(context.Background(), client, options...)
				if calls.Load() != 0 {
					t.Fatal("list was eager")
				}
				for attempt := range 2 {
					seen := 0
					for _, err := range iterator {
						if err != nil {
							t.Fatal(err)
						}
						seen++
					}
					if seen != wantRows || calls.Load() != int32(attempt+1)*wantCalls {
						t.Fatal("cap, single-page or reusable snapshot changed", seen, calls.Load())
					}
				}
			})
		}
	}
}

func TestClusteringCatalogListControlsStopBeforeUnusedRowsAndLinks(t *testing.T) {
	for _, fixture := range catalogControlFixtures() {
		for _, mode := range []string{"cap-before-decode", "single-before-link", "uncapped-decode", "uncapped-link", "empty-with-next"} {
			t.Run(fixture.path+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("clustering", "/senlin/v1")
				client.Microversion = "1.7"
				var calls atomic.Int32
				rows := fixture.row("first")
				var options []senlin.ListOption
				if mode == "cap-before-decode" || mode == "uncapped-decode" {
					rows += ",false"
				}
				if mode == "cap-before-decode" {
					options = []senlin.ListOption{fixture.max(1)}
				} else if mode == "single-before-link" {
					options = []senlin.ListOption{fixture.paginated(false)}
				} else if mode == "empty-with-next" {
					rows = ""
				}
				body := `{"` + fixture.key + `":[` + rows + `]}`
				cloud.Mux.HandleFunc("GET /senlin/v1/"+fixture.path, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("Link", `<https://foreign.example/escape?marker=next>; rel="next"`)
					w.Header().Set("X-Request-ID", "whole-page")
					testcloud.JSON(w, 200, body)
				})
				seen := 0
				var err error
				for _, itemErr := range fixture.list(context.Background(), client, options...) {
					if itemErr != nil {
						err = itemErr
					} else {
						seen++
					}
				}
				wantRows := 1
				if mode == "empty-with-next" {
					wantRows = 0
				}
				if seen != wantRows || calls.Load() != 1 {
					t.Fatal(seen, calls.Load(), err)
				}
				if mode == "uncapped-decode" || mode == "uncapped-link" {
					var evidence *resource.ResponseError
					if !errors.As(err, &evidence) || string(evidence.Body) != body || evidence.StatusCode != 200 || evidence.Header.Get("X-Request-ID") != "whole-page" {
						t.Fatal(err, evidence)
					}
				} else if err != nil {
					t.Fatal("unused row/link was processed after local termination", err)
				}
			})
		}
	}
}

func TestClusteringCatalogListControlsPreflightAndPageVersions(t *testing.T) {
	for _, fixture := range catalogControlFixtures() {
		t.Run(fixture.path, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("clustering", "/senlin/v1")
			client.Microversion = "1.7"
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /senlin/v1/"+fixture.path, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Link", fmt.Sprintf(`</senlin/v1/%s?marker=next>; rel="next"`, fixture.path))
				testcloud.JSON(w, 200, `{"`+fixture.key+`":[`+fixture.row("first")+`]}`)
			})
			for _, option := range []senlin.ListOption{fixture.max(-1), fixture.snapshot(senlin.ListOpts{Limit: -1}), fixture.query("max_items", "1"), fixture.query("paginated", "false")} {
				var err error
				for _, itemErr := range fixture.list(context.Background(), client, option) {
					err = itemErr
				}
				if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
					t.Fatal("invalid control reached HTTP", err, calls.Load())
				}
			}
			if fixture.path == "services" {
				client.Microversion = "1.6"
				iterator := fixture.list(context.Background(), client)
				if calls.Load() != 0 {
					t.Fatal("service version check eagerly sent HTTP")
				}
				for _, err := range iterator {
					if !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 0 {
						t.Fatal(err, calls.Load())
					}
				}
				client.Microversion = "1.7"
			}
			seen := 0
			var err error
			for _, itemErr := range fixture.list(context.Background(), client) {
				if itemErr != nil {
					err = itemErr
				} else {
					seen++
					client.Microversion = "latest"
					if fixture.path == "services" {
						client.Microversion = "1.6"
					}
				}
			}
			if seen != 1 || !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 1 {
				t.Fatal("source version not checked before next page", seen, err, calls.Load())
			}
		})
	}
}

func TestClusteringBindingListControlsStayLocalAndOwnSnapshots(t *testing.T) {
	for _, mode := range []string{"cap", "single-page-snapshot", "last-wins", "zero-unbounded", "cap-before-decode", "default-decode", "empty-with-next"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("clustering", "/senlin/v1")
			var parents, lists atomic.Int32
			cloud.Mux.HandleFunc("GET /senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
				parents.Add(1)
				testcloud.JSON(w, 200, `{"cluster":{"id":"canonical"}}`)
			})
			row := func(id string) string {
				return fmt.Sprintf(`{"id":%q,"policy_id":"policy-%s","cluster_id":"canonical","enabled":false}`, id, id)
			}
			options := []clusterpolicies.ListOption{clusterpolicies.WithListEnabled(false), clusterpolicies.WithListQuery("vendor", "retained")}
			wantRows, wantCalls := 2, int32(1)
			var captured *request.Config[clusterpolicies.ListOpts]
			switch mode {
			case "cap":
				options = append(options, clusterpolicies.WithListMaxItems(2))
			case "single-page-snapshot":
				paginated, enabled := false, false
				options = []clusterpolicies.ListOption{clusterpolicies.WithListOptions(clusterpolicies.ListOpts{Enabled: &enabled, Paginated: &paginated}), clusterpolicies.WithListQuery("vendor", "retained")}
				paginated, enabled = true, true
			case "last-wins":
				options = append(options, clusterpolicies.WithListMaxItems(1), clusterpolicies.WithListMaxItems(2), clusterpolicies.WithListPaginated(false), clusterpolicies.WithListPaginated(true))
				wantCalls = 2
			case "zero-unbounded":
				options = append(options, clusterpolicies.WithListMaxItems(1), clusterpolicies.WithListMaxItems(0))
				wantRows, wantCalls = 3, 2
			case "cap-before-decode":
				options = append(options, clusterpolicies.WithListMaxItems(1))
				wantRows = 1
			case "default-decode":
				wantRows = 1
			case "empty-with-next":
				wantRows = 0
			}
			options = append(options, func(config *request.Config[clusterpolicies.ListOpts]) error { captured = config; return nil })
			cloud.Mux.HandleFunc("GET /senlin/v1/clusters/canonical/policies", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				query := r.URL.Query()
				if query.Has("limit") || query.Has("max_items") || query.Has("paginated") || query.Get("enabled") != "false" || query.Get("vendor") != "retained" {
					t.Error("binding local control leaked or query changed", query)
				}
				captured.Options.MaxItems = 99
				if captured.Options.Paginated != nil {
					*captured.Options.Paginated = true
				}
				captured.Query.Set("vendor", "mutated")
				rows := row("first") + "," + row("second")
				if query.Get("marker") == "" {
					w.Header().Set("Link", `</senlin/v1/clusters/canonical/policies?marker=opaque>; rel="next"`)
					if mode == "last-wins" {
						rows = row("first")
					} else if mode == "cap-before-decode" || mode == "default-decode" {
						rows = row("first") + `,{"id":"unused","policy_id":null,"cluster_id":"canonical"}`
						w.Header().Set("Link", `<https://foreign.example/escape>; rel="next"`)
					} else if mode == "empty-with-next" {
						rows = ""
						w.Header().Set("Link", `<https://foreign.example/escape>; rel="next"`)
					}
				} else {
					if query.Get("marker") != "opaque" {
						t.Error(query)
					}
					rows = row("last")
				}
				w.Header().Set("X-Request-ID", "binding-page")
				testcloud.JSON(w, 200, `{"cluster_policies":[`+rows+`]}`)
			})
			scope, err := clusterpolicies.New(client).InCluster(context.Background(), resource.ID("selected"))
			if err != nil {
				t.Fatal(err)
			}
			for _, option := range []clusterpolicies.ListOption{clusterpolicies.WithListMaxItems(-1), clusterpolicies.WithListQuery("max_items", "1"), clusterpolicies.WithListQuery("paginated", "false"), clusterpolicies.WithListQuery("limit", "1"), clusterpolicies.WithListQuery("marker", "initial")} {
				_, err := scope.All(context.Background(), option)
				if err == nil || lists.Load() != 0 {
					t.Fatal("invalid binding control reached HTTP", err, lists.Load())
				}
			}
			iterator := scope.List(context.Background(), options...)
			if lists.Load() != 0 {
				t.Fatal("binding list was eager")
			}
			for attempt := range 2 {
				seen := 0
				var listErr error
				for _, itemErr := range iterator {
					if itemErr != nil {
						listErr = itemErr
					} else {
						seen++
					}
				}
				if seen != wantRows || lists.Load() != int32(attempt+1)*wantCalls || parents.Load() != 1 {
					t.Fatal(seen, lists.Load(), parents.Load(), listErr)
				}
				if mode == "default-decode" {
					var evidence *resource.ResponseError
					if !errors.As(listErr, &evidence) || evidence.StatusCode != 200 || evidence.Header.Get("X-Request-ID") != "binding-page" {
						t.Fatal(listErr, evidence)
					}
				} else if listErr != nil {
					t.Fatal(listErr)
				}
			}
		})
	}
}
