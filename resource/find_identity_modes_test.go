package resource_test

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

func identityFindModesAdapter(cloud *testcloud.Cloud) resource.Adapter[identityFindItem] {
	adapter := identityFindAdapter(cloud)
	client := cloud.Client("test", "/reverse/v1")
	adapter.IdentityAllProjectsQuery = "all_tenants"
	adapter.IterateIdentity = func(ctx context.Context, query url.Values, details bool) iter.Seq2[*identityFindItem, error] {
		path := client.ServiceURL("items")
		if details {
			path = client.ServiceURL("items", "detail")
		}
		pager := pagination.NewPager(client, path+"?"+query.Encode(), func(result pagination.PageResult) pagination.Page {
			return identityFindPage{pagination.LinkedPageBase{PageResult: result}}
		})
		return resource.Stream(ctx, pager, identityFindItems)
	}
	return adapter
}

func TestCollectionFindIdentityModesSelectFallbackWithoutChangingGETOrList(t *testing.T) {
	for _, check := range []struct {
		name    string
		options []resource.IdentityFindOption
		details bool
	}{
		{"default", nil, true},
		{"explicit-detailed", []resource.IdentityFindOption{resource.WithIdentityFindDetails(true)}, true},
		{"summary", []resource.IdentityFindOption{resource.WithIdentityFindDetails(false)}, false},
		{"last-wins", []resource.IdentityFindOption{resource.WithIdentityFindDetails(false), resource.WithIdentityFindDetails(true)}, true},
		{"bulk-resets", []resource.IdentityFindOption{resource.WithIdentityFindDetails(false), resource.WithIdentityFindAllProjects(true), resource.WithIdentityFindOptions(resource.IdentityFindOpts{})}, true},
	} {
		t.Run(check.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, lists, ordinary atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/v1/items/identity", func(w http.ResponseWriter, request *http.Request) {
				gets.Add(1)
				if request.URL.RawQuery != "" {
					t.Error("details changed direct GET", request.URL)
				}
				w.WriteHeader(404)
			})
			for _, details := range []bool{false, true} {
				path := "/reverse/v1/items"
				if details {
					path += "/detail"
				}
				cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, request *http.Request) {
					lists.Add(1)
					if details != check.details || request.URL.Query().Get("name") != "^identity$" || request.URL.Query().Has("details") || request.URL.Query().Has("all_projects") || request.URL.Query().Has("all_tenants") {
						t.Error("list mode/default query changed", request.URL)
					}
					testcloud.JSON(w, 200, identityFindBody(`{"id":"canonical","name":"identity"}`, ""))
				})
			}
			adapter := identityFindModesAdapter(cloud)
			ordinaryError := errors.New("ordinary controlled iterator")
			adapter.IterateControlled = func(context.Context, url.Values, resource.ListControl) iter.Seq2[*identityFindItem, error] {
				ordinary.Add(1)
				return func(yield func(*identityFindItem, error) bool) { yield(nil, ordinaryError) }
			}
			collection := resource.NewCollection(adapter)
			value, err := collection.FindIdentity(context.Background(), "identity", check.options...)
			if err != nil || value == nil || value.ID != "canonical" || gets.Load() != 1 || lists.Load() != 1 || ordinary.Load() != 0 {
				t.Fatal(value, err, gets.Load(), lists.Load(), ordinary.Load())
			}
			if _, err := collection.All(context.Background()); !errors.Is(err, ordinaryError) || ordinary.Load() != 1 {
				t.Fatal("identity fallback mutated the ordinary List binding", err, ordinary.Load())
			}
		})
	}
}

func TestCollectionFindIdentityTypedModesKeepSuccessfulMemberGET(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, modes atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/v1/items/identity", func(w http.ResponseWriter, request *http.Request) {
		gets.Add(1)
		if request.URL.RawQuery != "" {
			t.Error("typed list policy leaked into GET", request.URL)
		}
		testcloud.JSON(w, 203, `{"item":{"id":"canonical","name":"different","extra":"kept by native mapping"}}`)
	})
	adapter := identityFindModesAdapter(cloud)
	adapter.IterateIdentity = func(context.Context, url.Values, bool) iter.Seq2[*identityFindItem, error] {
		modes.Add(1)
		return nil
	}
	value, err := resource.NewCollection(adapter).FindIdentity(context.Background(), "identity", resource.WithIdentityFindDetails(false), resource.WithIdentityFindAllProjects(true), resource.WithIdentityFindFallback(resource.FindFallbackNever))
	if err != nil || value == nil || value.ID != "canonical" || value.Name != "different" || gets.Load() != 1 || modes.Load() != 0 {
		t.Fatal(value, err, gets.Load(), modes.Load())
	}
}

func TestCollectionFindIdentityTypedAllProjectsIsListOnlyAndRawQueryRemainsDualPhase(t *testing.T) {
	for _, check := range []struct {
		name      string
		options   []resource.IdentityFindOption
		getValue  string
		listValue string
	}{
		{"absent", nil, "", ""},
		{"false", []resource.IdentityFindOption{resource.WithIdentityFindAllProjects(false)}, "", ""},
		{"true", []resource.IdentityFindOption{resource.WithIdentityFindAllProjects(true)}, "", "true"},
		{"last-false", []resource.IdentityFindOption{resource.WithIdentityFindAllProjects(true), resource.WithIdentityFindAllProjects(false)}, "", ""},
		{"raw-wire", []resource.IdentityFindOption{resource.WithIdentityFindQuery("all_tenants", "1")}, "1", "1"},
	} {
		t.Run(check.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, lists atomic.Int32
			checkQuery := func(request *http.Request, allProjects string, name bool) {
				query := request.URL.Query()
				if !reflect.DeepEqual(query["fields"], []string{"id", "name"}) || query.Get("all_tenants") != allProjects || query.Has("all_tenants") != (allProjects != "") || query.Has("name") != name || query.Has("details") || query.Has("all_projects") {
					t.Error("typed/wire query phase changed", request.URL)
				}
			}
			cloud.Mux.HandleFunc("GET /reverse/v1/items/identity", func(w http.ResponseWriter, request *http.Request) {
				gets.Add(1)
				checkQuery(request, check.getValue, false)
				w.WriteHeader(403)
			})
			cloud.Mux.HandleFunc("GET /reverse/v1/items/detail", func(w http.ResponseWriter, request *http.Request) {
				lists.Add(1)
				checkQuery(request, check.listValue, true)
				testcloud.JSON(w, 200, identityFindBody(`{"id":"canonical","name":"identity"}`, ""))
			})
			options := []resource.IdentityFindOption{resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: url.Values{"fields": {"id", "name"}}})}
			options = append(options, check.options...)
			value, err := resource.NewCollection(identityFindModesAdapter(cloud)).FindIdentity(context.Background(), "identity", options...)
			if err != nil || value == nil || value.ID != "canonical" || gets.Load() != 1 || lists.Load() != 1 {
				t.Fatal(value, err, gets.Load(), lists.Load())
			}
		})
	}
}

func TestCollectionFindIdentityModesFreezeBulkAndCapturedOptions(t *testing.T) {
	for _, custom := range []bool{false, true} {
		t.Run(fmt.Sprintf("custom-%t", custom), func(t *testing.T) {
			cloud := testcloud.New(t)
			details, allProjects := false, true
			input := resource.IdentityFindOpts{Details: &details, AllProjects: &allProjects, Query: url.Values{"fields": {"id", "name"}, "name": nil}}
			var captured *resource.IdentityFindOpts
			option := resource.WithIdentityFindOptions(input)
			if custom {
				option = func(value *resource.IdentityFindOpts) error {
					*value = input
					captured = value
					return nil
				}
			} else {
				details, allProjects = true, false
				input.Query["fields"][0] = "caller changed"
				input.Query["name"] = []string{"caller changed"}
			}
			var gets, lists atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/v1/items/identity", func(w http.ResponseWriter, request *http.Request) {
				gets.Add(1)
				if !reflect.DeepEqual(request.URL.Query()["fields"], []string{"id", "name"}) || request.URL.Query().Has("name") || request.URL.Query().Has("all_tenants") {
					t.Error(request.URL)
				}
				if custom {
					*captured.Details, *captured.AllProjects = true, false
					captured.Query["fields"][0] = "during GET"
					captured.Query["name"] = []string{"during GET"}
				}
				w.WriteHeader(404)
			})
			cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, request *http.Request) {
				lists.Add(1)
				if !reflect.DeepEqual(request.URL.Query()["fields"], []string{"id", "name"}) || request.URL.Query().Has("name") || request.URL.Query().Get("all_tenants") != "true" {
					t.Error("captured config changed frozen list settings", request.URL)
				}
				testcloud.JSON(w, 200, identityFindBody(`{"id":"identity","name":"unrelated"}`, ""))
			})
			value, err := resource.NewCollection(identityFindModesAdapter(cloud)).FindIdentity(context.Background(), "identity", option)
			if err != nil || value == nil || value.ID != "identity" || gets.Load() != 1 || lists.Load() != 1 {
				t.Fatal(value, err, gets.Load(), lists.Load())
			}
		})
	}
}

func TestCollectionFindIdentityModesPreflightUnsupportedAndConflicts(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /", func(w http.ResponseWriter, request *http.Request) { calls.Add(1); w.WriteHeader(500) })
	for _, option := range []resource.IdentityFindOption{
		resource.WithIdentityFindDetails(false), resource.WithIdentityFindDetails(true),
		resource.WithIdentityFindAllProjects(false), resource.WithIdentityFindAllProjects(true),
	} {
		_, err := resource.NewCollection(identityFindAdapter(cloud)).FindIdentity(context.Background(), "identity", option, resource.WithIdentityFindFallback(resource.FindFallbackNever))
		if !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal("explicit mode requires capability even for GET-only", err)
		}
	}
	for _, key := range []string{"details", "DeTaIlS", "all_projects", "ALL_PROJECTS"} {
		for _, values := range [][]string{nil, {}, {"true", "false"}} {
			_, err := resource.NewCollection(identityFindModesAdapter(cloud)).FindIdentity(context.Background(), "identity", resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: url.Values{key: values}}))
			if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(key, values, err)
			}
		}
	}
	for _, key := range []string{"all_tenants", "ALL_TENANTS", "All_Tenants"} {
		for _, values := range [][]string{nil, {}, {""}, {"true", "false"}} {
			for _, typed := range []bool{false, true} {
				for _, reverse := range []bool{false, true} {
					options := []resource.IdentityFindOption{
						func(config *resource.IdentityFindOpts) error { config.Query[key] = values; return nil },
						resource.WithIdentityFindAllProjects(typed),
					}
					if reverse {
						options[0], options[1] = options[1], options[0]
					}
					_, err := resource.NewCollection(identityFindModesAdapter(cloud)).FindIdentity(context.Background(), "identity", options...)
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(key, values, typed, reverse, err)
					}
				}
			}
		}
	}
	if calls.Load() != 0 {
		t.Fatal("invalid modes reached HTTP", calls.Load())
	}
}

func TestCollectionFindIdentityModesPreserveAllPageErrorsAndCancellation(t *testing.T) {
	for _, details := range []bool{false, true} {
		for _, failure := range []string{"duplicate", "http", "decode", "cancel"} {
			t.Run(fmt.Sprintf("details-%t/%s", details, failure), func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var gets, pages atomic.Int32
				cloud.Mux.HandleFunc("GET /reverse/v1/items/identity", func(w http.ResponseWriter, request *http.Request) { gets.Add(1); w.WriteHeader(400) })
				path := "/reverse/v1/items"
				if details {
					path += "/detail"
				}
				cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, request *http.Request) {
					if pages.Add(1) == 1 {
						testcloud.JSON(w, 200, identityFindBody(`{"id":"canonical","name":"identity"}`, cloud.Server.URL+path+"?marker=next"))
						return
					}
					switch failure {
					case "duplicate":
						testcloud.JSON(w, 200, identityFindBody(`{"id":"canonical","name":"identity"}`, ""))
					case "http":
						w.Header().Set("X-Evidence", "late")
						testcloud.JSON(w, 403, `{"error":"later page"}`)
					case "decode":
						testcloud.JSON(w, 200, `{"items":[{"id":false}]}`)
					case "cancel":
						cancel()
						<-request.Context().Done()
					}
				})
				value, err := resource.NewCollection(identityFindModesAdapter(cloud)).FindIdentity(ctx, "identity", resource.WithIdentityFindDetails(details))
				if value != nil || err == nil || gets.Load() != 1 || pages.Load() != 2 {
					t.Fatal(value, err, gets.Load(), pages.Load())
				}
				switch failure {
				case "duplicate":
					if !errors.Is(err, resource.ErrAmbiguous) {
						t.Fatal(err)
					}
				case "http":
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != 403 || native.ResponseHeader.Get("X-Evidence") != "late" || string(native.Body) != `{"error":"later page"}` {
						t.Fatal(err, native)
					}
				case "cancel":
					if !errors.Is(err, context.Canceled) {
						t.Fatal(err)
					}
				}
			})
		}
	}
}

func TestCollectionFindIdentityModeIteratorCancellationIsTerminal(t *testing.T) {
	cloud := testcloud.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var gets, modes atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/v1/items/identity", func(w http.ResponseWriter, request *http.Request) { gets.Add(1); w.WriteHeader(404) })
	adapter := identityFindModesAdapter(cloud)
	adapter.IterateIdentity = func(context.Context, url.Values, bool) iter.Seq2[*identityFindItem, error] {
		modes.Add(1)
		cancel()
		return func(yield func(*identityFindItem, error) bool) {
			yield(&identityFindItem{ID: "canonical", Name: "identity"}, nil)
		}
	}
	value, err := resource.NewCollection(adapter).FindIdentity(ctx, "identity", resource.WithIdentityFindDetails(false))
	if value != nil || !errors.Is(err, context.Canceled) || gets.Load() != 1 || modes.Load() != 1 {
		t.Fatal(value, err, gets.Load(), modes.Load())
	}
}

func TestCollectionFindIdentityModeOptionsCanBeReusedConcurrently(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, lists, modes atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/v1/items/identity", func(w http.ResponseWriter, request *http.Request) {
		gets.Add(1)
		if request.URL.Query().Has("all_tenants") || !reflect.DeepEqual(request.URL.Query()["fields"], []string{"id", "name"}) {
			t.Error(request.URL)
		}
		w.WriteHeader(404)
	})
	cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, request *http.Request) {
		lists.Add(1)
		if request.URL.Query().Get("all_tenants") != "true" || !reflect.DeepEqual(request.URL.Query()["fields"], []string{"id", "name"}) {
			t.Error(request.URL)
		}
		testcloud.JSON(w, 200, identityFindBody(`{"id":"canonical","name":"identity"}`, ""))
	})
	details, allProjects := false, true
	query := url.Values{"fields": {"id", "name"}}
	option := resource.WithIdentityFindOptions(resource.IdentityFindOpts{Details: &details, AllProjects: &allProjects, Query: query})
	details, allProjects = true, false
	query["fields"][0] = "caller changed"
	adapter := identityFindModesAdapter(cloud)
	original := adapter.IterateIdentity
	adapter.IterateIdentity = func(ctx context.Context, query url.Values, details bool) iter.Seq2[*identityFindItem, error] {
		modes.Add(1)
		// A hook owns its query copy. Mutating it must not change the next
		// invocation of the same option or collection.
		iterator := original(ctx, query, details)
		query["fields"][0] = "hook changed"
		query["all_tenants"] = nil
		return iterator
	}
	collection := resource.NewCollection(adapter)
	var wait sync.WaitGroup
	for range 10 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			value, err := collection.FindIdentity(context.Background(), "identity", option)
			if err != nil || value == nil || value.ID != "canonical" {
				t.Error(value, err)
			}
		}()
	}
	wait.Wait()
	if gets.Load() != 10 || lists.Load() != 10 || modes.Load() != 10 {
		t.Fatal(gets.Load(), lists.Load(), modes.Load())
	}
}
