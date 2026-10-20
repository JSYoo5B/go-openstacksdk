package resource_test

import (
	"context"
	"errors"
	"fmt"
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

func identityMissingListAdapter(cloud *testcloud.Cloud) resource.Adapter[identityFindItem] {
	adapter := identityFindAdapter(cloud)
	adapter.IdentityMissingListQuery = url.Values{"os_hidden": {"true"}}
	return adapter
}

func TestCollectionFindIdentityMissingListWaitsForCompleteSuccessfulAbsence(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, visible, hidden atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/v1/items/identity", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); w.WriteHeader(404) })
	cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("os_hidden") == "true" {
			hidden.Add(1)
			if visible.Load() != 2 || query.Has("name") || query.Has("marker") {
				t.Error("second search retained automatic hints or started early", r.URL, visible.Load())
			}
			testcloud.JSON(w, 200, identityFindBody(`{"id":"identity","name":"different"}`, ""))
			return
		}
		visible.Add(1)
		if query.Get("name") != "^identity$" {
			t.Error(r.URL)
		}
		if query.Get("marker") == "" {
			testcloud.JSON(w, 200, identityFindBody(`{"id":"unrelated","name":"different"}`, cloud.Server.URL+"/reverse/v1/items?name=%5Eidentity%24&marker=next"))
		} else {
			testcloud.JSON(w, 200, identityFindBody("", ""))
		}
	})
	value, err := resource.NewCollection(identityMissingListAdapter(cloud)).FindIdentity(context.Background(), "identity")
	if err != nil || value == nil || value.ID != "identity" || gets.Load() != 1 || visible.Load() != 2 || hidden.Load() != 1 {
		t.Fatal(value, err, gets.Load(), visible.Load(), hidden.Load())
	}
}

func TestCollectionFindIdentityMissingListPreservesCallerQueryAndMandatoryOverlay(t *testing.T) {
	for _, name := range [][]string{nil, {}, {"explicit", "second"}} {
		for _, hidden := range [][]string{nil, {}, {"false", "false"}} {
			t.Run(fmt.Sprintf("name-%v/hidden-%v", name, hidden), func(t *testing.T) {
				cloud := testcloud.New(t)
				query := url.Values{"name": name, "os_hidden": hidden, "tag": {"a", "b"}, "empty": nil}
				var gets, lists atomic.Int32
				cloud.Mux.HandleFunc("GET /reverse/v1/items/identity", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					if r.URL.RawQuery != query.Encode() {
						t.Error("SDK overlay leaked into GET", r.URL)
					}
					w.WriteHeader(403)
				})
				cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) {
					if lists.Add(1) == 1 {
						if r.URL.RawQuery != query.Encode() {
							t.Error("first search changed explicit caller fields", r.URL)
						}
						testcloud.JSON(w, 200, identityFindBody("", ""))
					} else {
						if !reflect.DeepEqual(r.URL.Query()["os_hidden"], []string{"true"}) || !reflect.DeepEqual(r.URL.Query()["tag"], []string{"a", "b"}) || r.URL.Query().Get("name") != query.Get("name") {
							t.Error("fixed overlay/caller query changed", r.URL)
						}
						testcloud.JSON(w, 200, identityFindBody(`{"id":"found","name":"identity"}`, ""))
					}
				})
				adapter := identityMissingListAdapter(cloud)
				list := adapter.List
				adapter.List = func(values url.Values) pagination.Pager {
					if !values.Has("name") || !values.Has("empty") || !reflect.DeepEqual(values["name"], query["name"]) && len(query["name"]) != 0 {
						t.Error("explicit nil/empty key presence lost", values)
					}
					return list(values)
				}
				value, err := resource.NewCollection(adapter).FindIdentity(context.Background(), "identity", resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: query}))
				if err != nil || value == nil || value.ID != "found" || gets.Load() != 1 || lists.Load() != 2 {
					t.Fatal(value, err, gets.Load(), lists.Load())
				}
			})
		}
	}
}

func TestCollectionFindIdentityMissingListStrictAndFallbackPolicies(t *testing.T) {
	for _, check := range []struct {
		name     string
		policy   url.Values
		status   int
		fallback resource.FindFallbackPolicy
		strict   bool
		lists    int32
	}{
		{"default", url.Values{"os_hidden": {"true"}}, 403, resource.FindFallbackCompatible, false, 2},
		{"strict", url.Values{"os_hidden": {"true"}}, 403, resource.FindFallbackCompatible, true, 2},
		{"nil-policy", nil, 404, resource.FindFallbackCompatible, true, 1},
		{"empty-policy", url.Values{}, 404, resource.FindFallbackCompatible, false, 1},
		{"404-only", url.Values{"os_hidden": {"true"}}, 404, resource.FindFallbackNotFoundOnly, false, 2},
		{"403-not-fallback", url.Values{"os_hidden": {"true"}}, 403, resource.FindFallbackNotFoundOnly, false, 0},
		{"never-missing", url.Values{"os_hidden": {"true"}}, 404, resource.FindFallbackNever, false, 0},
		{"never-strict", url.Values{"os_hidden": {"true"}}, 404, resource.FindFallbackNever, true, 0},
		{"GET-success", url.Values{"os_hidden": {"true"}}, 200, resource.FindFallbackCompatible, true, 0},
	} {
		t.Run(check.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, lists atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/v1/items/identity", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, check.status, `{"item":{"id":"canonical","name":"different"}}`)
			})
			cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				testcloud.JSON(w, 200, identityFindBody("", ""))
			})
			adapter := identityFindAdapter(cloud)
			adapter.IdentityMissingListQuery = check.policy
			value, err := resource.NewCollection(adapter).FindIdentity(context.Background(), "identity", resource.WithIdentityFindFallback(check.fallback), resource.WithIdentityFindIgnoreMissing(!check.strict))
			if gets.Load() != 1 || lists.Load() != check.lists {
				t.Fatal(value, err, gets.Load(), lists.Load())
			}
			if check.status == 200 {
				if err != nil || value == nil || value.ID != "canonical" {
					t.Fatal(value, err)
				}
			} else if check.lists != 0 && check.strict {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.Is(err, resource.ErrNotFound) || errors.As(err, &native) {
					t.Fatal("logical missing retained a suppressed GET cause", err)
				}
			} else if check.lists == 0 && (check.status == 403 || check.strict) {
				if !gophercloud.ResponseCodeIs(err, check.status) {
					t.Fatal(err)
				}
			} else if value != nil || err != nil {
				t.Fatal(value, err)
			}
		})
	}
}

func TestCollectionFindIdentityMissingListErrorsAndCancellationAreTerminal(t *testing.T) {
	for _, phase := range []int{1, 2} {
		for _, failure := range []string{"403", "404", "decode", "invalid-id", "duplicate", "late-error", "cancel"} {
			t.Run(fmt.Sprintf("phase-%d/%s", phase, failure), func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var gets, calls atomic.Int32
				cloud.Mux.HandleFunc("GET /reverse/v1/items/identity", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); w.WriteHeader(404) })
				cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					current := 1
					if r.URL.Query().Get("os_hidden") == "true" {
						current = 2
					}
					if current != phase {
						testcloud.JSON(w, 200, identityFindBody("", ""))
						return
					}
					switch failure {
					case "403", "404":
						code := 403
						if failure == "404" {
							code = 404
						}
						w.Header().Set("X-Evidence", "terminal")
						testcloud.JSON(w, code, `{"error":"lookup failed"}`)
					case "decode":
						testcloud.JSON(w, 200, `{"items":[{"id":true}]}`)
					case "invalid-id":
						testcloud.JSON(w, 200, identityFindBody(`{"id":"","name":"unrelated"}`, ""))
					case "duplicate":
						testcloud.JSON(w, 200, identityFindBody(`{"id":"same","name":"identity"},{"id":"same","name":"identity"}`, ""))
					case "late-error":
						if r.URL.Query().Has("marker") {
							w.WriteHeader(500)
						} else {
							testcloud.JSON(w, 200, identityFindBody(`{"id":"found","name":"identity"}`, cloud.Server.URL+r.URL.Path+"?marker=next&os_hidden="+r.URL.Query().Get("os_hidden")))
						}
					case "cancel":
						cancel()
						testcloud.JSON(w, 200, identityFindBody("", ""))
					}
				})
				value, err := resource.NewCollection(identityMissingListAdapter(cloud)).FindIdentity(ctx, "identity")
				wanted := int32(phase)
				if failure == "late-error" {
					wanted++
				}
				if value != nil || err == nil || gets.Load() != 1 || calls.Load() != wanted {
					t.Fatal(value, err, gets.Load(), calls.Load(), wanted)
				}
				if failure == "duplicate" && !errors.Is(err, resource.ErrAmbiguous) || failure == "cancel" && !errors.Is(err, context.Canceled) || failure == "invalid-id" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				if failure == "403" || failure == "404" {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.ResponseHeader.Get("X-Evidence") != "terminal" || string(native.Body) != `{"error":"lookup failed"}` {
						t.Fatal(err, native)
					}
				}
			})
		}
	}
}

func TestCollectionFindIdentityMissingListSnapshotsAndConcurrentReuse(t *testing.T) {
	cloud := testcloud.New(t)
	policy := url.Values{"os_hidden": {"true"}}
	adapter := identityFindAdapter(cloud)
	adapter.IdentityMissingListQuery = policy
	collection := resource.NewCollection(adapter)
	policy["os_hidden"][0] = "false"
	policy["headers"] = []string{"bad"}
	var gets, lists atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/v1/items/identity", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if !reflect.DeepEqual(r.URL.Query()["tag"], []string{"a", "b"}) {
			t.Error(r.URL)
		}
		w.WriteHeader(404)
	})
	cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if !reflect.DeepEqual(r.URL.Query()["tag"], []string{"a", "b"}) || r.URL.Query().Has("headers") {
			t.Error(r.URL)
		}
		rows := ""
		if r.URL.Query().Get("os_hidden") == "true" {
			if r.URL.Query().Has("name") {
				t.Error("automatic name hint leaked to second search", r.URL)
			}
			rows = `{"id":"found","name":"identity"}`
		}
		testcloud.JSON(w, 200, identityFindBody(rows, ""))
	})
	query := url.Values{"tag": {"a", "b"}}
	option := resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: query})
	query["tag"][0] = "caller changed"
	var wait sync.WaitGroup
	for range 6 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			value, err := collection.FindIdentity(context.Background(), "identity", option)
			if err != nil || value == nil || value.ID != "found" {
				t.Error(value, err)
			}
		}()
	}
	wait.Wait()
	if gets.Load() != 6 || lists.Load() != 12 {
		t.Fatal(gets.Load(), lists.Load())
	}
}

func TestCollectionFindIdentityMissingListFreezesOriginalQueryAcrossCallbacksAndUnsafeNames(t *testing.T) {
	for _, unsafe := range []bool{false, true} {
		t.Run(fmt.Sprintf("unsafe-%t", unsafe), func(t *testing.T) {
			cloud := testcloud.New(t)
			identity := "identity"
			if unsafe {
				identity = "name /with%space"
			}
			var captured *resource.IdentityFindOpts
			var gets, lists atomic.Int32
			mutate := func() {
				captured.Query["tag"][0] = "captured mutation"
				captured.Query["name"] = []string{"captured name"}
				captured.Query["os_hidden"] = []string{"captured hidden"}
			}
			cloud.Mux.HandleFunc("GET /reverse/v1/items/identity", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				mutate()
				w.WriteHeader(404)
			})
			cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) {
				call := lists.Add(1)
				query := r.URL.Query()
				if !reflect.DeepEqual(query["tag"], []string{"a", "b"}) {
					t.Error("captured callback changed frozen query", r.URL)
				}
				if call == 1 {
					if query.Get("name") != "^"+identity+"$" || query.Get("os_hidden") != "false" {
						t.Error("normal search lost original fields/hint", r.URL)
					}
					testcloud.JSON(w, 200, identityFindBody("", ""))
				} else {
					if query.Has("name") || query.Get("os_hidden") != "true" {
						t.Error("second search retained callback/automatic hint", r.URL)
					}
					row := fmt.Sprintf(`{"id":"found","name":%q}`, identity)
					testcloud.JSON(w, 200, identityFindBody(row, ""))
				}
			})
			adapter := identityMissingListAdapter(cloud)
			originalList := adapter.List
			adapter.List = func(query url.Values) pagination.Pager {
				pager := originalList(query)
				mutate()
				query.Set("name", "list callback")
				query["tag"][0] = "list callback"
				return pager
			}
			option := func(config *resource.IdentityFindOpts) error {
				config.Query = url.Values{"tag": {"a", "b"}, "os_hidden": {"false", "false"}}
				captured = config
				return nil
			}
			collection := resource.NewCollection(adapter)
			value, err := collection.FindIdentity(context.Background(), identity, option)
			wantGets := int32(1)
			if unsafe {
				wantGets = 0
			}
			if err != nil || value == nil || value.ID != "found" || gets.Load() != wantGets || lists.Load() != 2 {
				t.Fatal(value, err, gets.Load(), lists.Load())
			}
			if unsafe {
				if _, err := collection.FindIdentity(context.Background(), identity, resource.WithIdentityFindFallback(resource.FindFallbackNever)); !errors.Is(err, resource.ErrInvalidOption) || gets.Load() != 0 || lists.Load() != 2 {
					t.Fatal("GET-only unsafe name bypassed route policy", err, gets.Load(), lists.Load())
				}
			}
		})
	}
}

func TestCollectionFindIdentityMissingListPolicyPreflightAndOtherOperationsStayUnchanged(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, lists atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/v1/items/identity", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		testcloud.JSON(w, 200, `{"item":{"id":"identity","name":"identity"}}`)
	})
	cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if r.URL.Query().Has("os_hidden") {
			t.Error("policy leaked into ordinary lookup/list", r.URL)
		}
		testcloud.JSON(w, 200, identityFindBody(`{"id":"identity","name":"identity"}`, ""))
	})
	for _, key := range []string{"", "Headers", "details", "max_items", "base_path"} {
		adapter := identityFindAdapter(cloud)
		adapter.IdentityMissingListQuery = url.Values{key: {"true"}}
		if _, err := resource.NewCollection(adapter).FindIdentity(context.Background(), "identity"); !errors.Is(err, resource.ErrInvalidOption) || gets.Load() != 0 || lists.Load() != 0 {
			t.Fatal(key, err, gets.Load(), lists.Load())
		}
	}
	collection := resource.NewCollection(identityMissingListAdapter(cloud))
	if _, err := collection.Get(context.Background(), "identity"); err != nil {
		t.Fatal(err)
	}
	if _, err := collection.All(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := collection.Find(context.Background(), resource.Name("identity")); err != nil {
		t.Fatal(err)
	}
	if gets.Load() != 1 || lists.Load() != 2 {
		t.Fatal(gets.Load(), lists.Load())
	}
}
