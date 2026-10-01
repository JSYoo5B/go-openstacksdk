package api_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/compute"
	"gophercloudsdk/compute/v2/flavors"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

// Pinned find_flavor defaults get_extra_specs=False and is_public=None. Its
// optional extra-specs GET follows a successfully resolved flavor only when
// its inline map is empty. These tests keep native Flavor/FlavorPage decoding;
// missing/null extra-specs wrappers retain native nil-map semantics.
type flavorIdentityFind func(context.Context, string, ...resource.IdentityFindOption) (*flavors.Flavor, error)

type flavorIdentityFixture struct {
	name string
	new  func(*gophercloud.ServiceClient) flavorIdentityFind
}

func flavorIdentityFixtures() []flavorIdentityFixture {
	return []flavorIdentityFixture{
		{"generated-leaf", func(c *gophercloud.ServiceClient) flavorIdentityFind { return flavors.New(c).FindIdentity }},
		{"handwritten-collection", func(c *gophercloud.ServiceClient) flavorIdentityFind {
			return compute.New(c, compute.Dependencies{}).Flavors.FindIdentity
		}},
	}
}

const flavorIdentityPath = "/reverse/nova/v2.1/project/flavors"

func flavorIdentityClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("compute", "/unused/catalog")
	client.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + "/reverse/nova/v2.1/project")
	return client
}

func flavorIdentityPage(rows, next string) string {
	links := ""
	if next != "" {
		links = fmt.Sprintf(`,"flavors_links":[{"rel":"next","href":%q}]`, next)
	}
	return fmt.Sprintf(`{"flavors":[%s]%s}`, rows, links)
}

func flavorIdentityBool(value bool) *bool { return &value }

func TestNativeFindIdentityFlavorsGETAndExtraSpecsPresence(t *testing.T) {
	for _, f := range flavorIdentityFixtures() {
		for _, test := range []struct {
			name   string
			option *bool
			inline string
			fetch  bool
		}{
			{"default-omitted", nil, "", false},
			{"default-inline", nil, `,"extra_specs":{"inline":"kept"}`, false},
			{"explicit-false", flavorIdentityBool(false), "", false},
			{"true-omitted", flavorIdentityBool(true), "", true},
			{"true-null", flavorIdentityBool(true), `,"extra_specs":null`, true},
			{"true-empty", flavorIdentityBool(true), `,"extra_specs":{}`, true},
			{"true-inline", flavorIdentityBool(true), `,"extra_specs":{"inline":"kept"}`, false},
		} {
			t.Run(f.name+"/"+test.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets, extras, other atomic.Int32
				client := flavorIdentityClient(cloud)
				client.Microversion = "2.55"
				cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/lookup", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					if r.URL.Query().Get("vendor") != "kept" || r.URL.Query().Has("is_public") || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.55" {
						t.Error("list defaults or version changed the native GET", r.URL, r.Header)
					}
					testcloud.JSON(w, 200, `{"flavor":{"id":"7","name":"Different","ram":2048,"disk":20,"vcpus":2,"swap":"4","rxtx_factor":1.5,"os-flavor-access:is_public":true,"OS-FLV-EXT-DATA:ephemeral":3,"description":"preserved"`+test.inline+`}}`)
				})
				cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/7/os-extra_specs", func(w http.ResponseWriter, r *http.Request) {
					extras.Add(1)
					if r.URL.RawQuery != "" || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.55" {
						t.Error("caller query leaked into enrichment or version was upgraded", r.URL, r.Header)
					}
					testcloud.JSON(w, 200, `{"extra_specs":{"hw:cpu_policy":"dedicated","vendor":"value"}}`)
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { other.Add(1); w.WriteHeader(500) })
				value, err := f.new(client)(context.Background(), "lookup", resource.WithIdentityFindOptions(resource.IdentityFindOpts{GetExtraSpecs: test.option, Query: url.Values{"vendor": {"kept"}}}))
				wantExtras := int32(0)
				if test.fetch {
					wantExtras = 1
				}
				if err != nil || value == nil || value.ID != "7" || value.Name != "Different" || value.RAM != 2048 || value.Disk != 20 || value.VCPUs != 2 || value.Swap != 4 || value.RxTxFactor != 1.5 || !value.IsPublic || value.Ephemeral != 3 || value.Description != "preserved" || gets.Load() != 1 || extras.Load() != wantExtras || other.Load() != 0 {
					t.Fatal("native result or canonical numeric route changed", value, err, gets.Load(), extras.Load(), other.Load())
				}
				if test.fetch && !reflect.DeepEqual(value.ExtraSpecs, map[string]string{"hw:cpu_policy": "dedicated", "vendor": "value"}) || strings.Contains(test.inline, "inline") && !reflect.DeepEqual(value.ExtraSpecs, map[string]string{"inline": "kept"}) {
					t.Fatal("extra-specs map replaced unrelated flavor fields or inline data", value)
				}
			})
		}
		for _, body := range []string{`{}`, `{"extra_specs":null}`, `{"extra_specs":{}}`} {
			t.Run(f.name+"/native-empty-wrapper/"+body, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/7", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					testcloud.JSON(w, 200, `{"flavor":{"id":"7","name":"found"}}`)
				})
				cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/7/os-extra_specs", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, body) })
				value, err := f.new(flavorIdentityClient(cloud))(context.Background(), "7", resource.WithIdentityFindExtraSpecs(true))
				if err != nil || value == nil || len(value.ExtraSpecs) != 0 || calls.Load() != 2 || (body == `{"extra_specs":{}}`) != (value.ExtraSpecs != nil) {
					t.Fatal("native missing/null/empty map semantics changed", value, err, calls.Load())
				}
			})
		}
	}
}

func TestNativeFindIdentityFlavorsListDefaultsAndCallerQuery(t *testing.T) {
	for _, f := range flavorIdentityFixtures() {
		for _, test := range []struct {
			name  string
			query url.Values
		}{
			{"default-all-access", url.Values{"vendor": {"first", "second"}}},
			{"public", url.Values{"is_public": {"true"}}},
			{"private", url.Values{"is_public": {"false"}}},
			{"nil-access", url.Values{"is_public": nil}},
			{"empty-access-slice", url.Values{"is_public": {}}},
			{"empty-access-value", url.Values{"is_public": {""}}},
			{"repeated-explicit-query", url.Values{"is_public": {"false", "None"}, "name": {"first", "second"}, "status": {"vendor-status"}}},
		} {
			t.Run(f.name+"/"+test.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets, lists, extras atomic.Int32
				want := make(url.Values)
				for key, values := range test.query {
					want[key] = append([]string(nil), values...)
				}
				if !want.Has("is_public") {
					want.Set("is_public", "None")
				}
				cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/lookup", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					if r.URL.RawQuery != test.query.Encode() {
						t.Error("list default leaked into GET or caller query was lost", r.URL, test.query)
					}
					w.WriteHeader(404)
				})
				cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/detail", func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					if r.URL.RawQuery != want.Encode() {
						t.Error("flavor query changed or automatic name hint was invented", r.URL, want)
					}
					testcloud.JSON(w, 300, flavorIdentityPage(`{"id":"lookup","name":"Different","ram":512,"swap":""}`, ""))
				})
				cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/lookup/os-extra_specs", func(w http.ResponseWriter, r *http.Request) {
					extras.Add(1)
					if r.URL.RawQuery != "" {
						t.Error("enrichment inherited list parameters", r.URL)
					}
					testcloud.JSON(w, 200, `{"extra_specs":{"found":"yes"}}`)
				})
				value, err := f.new(flavorIdentityClient(cloud))(context.Background(), "lookup", resource.WithIdentityFindOptions(resource.IdentityFindOpts{GetExtraSpecs: flavorIdentityBool(true), Query: test.query}))
				if err != nil || value == nil || value.ID != "lookup" || value.Name != "Different" || value.RAM != 512 || value.Swap != 0 || value.ExtraSpecs["found"] != "yes" || gets.Load() != 1 || lists.Load() != 1 || extras.Load() != 1 {
					t.Fatal(value, err, gets.Load(), lists.Load(), extras.Load())
				}
			})
		}
	}
}

func TestNativeFindIdentityFlavorsLookupClosesBeforeEnrichment(t *testing.T) {
	for _, f := range flavorIdentityFixtures() {
		for _, mode := range []string{"unique-all-pages", "same-id-duplicate", "late-http", "whole-page-decode", "late-decode", "invalid-row", "cycle", "empty-204", "list-403", "list-404"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets, lists, extras atomic.Int32
				cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/lookup", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); w.WriteHeader(404) })
				cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/detail", func(w http.ResponseWriter, r *http.Request) {
					page := lists.Add(1)
					if r.URL.Query().Has("name") || r.URL.Query().Get("is_public") != "None" {
						t.Error("native detail list defaults changed", r.URL)
					}
					if mode == "empty-204" {
						w.WriteHeader(204)
						return
					}
					if mode == "list-403" || mode == "list-404" {
						status := 403
						if mode == "list-404" {
							status = 404
						}
						w.WriteHeader(status)
						return
					}
					rows := `{"id":"7","name":"lookup","swap":"2"}`
					next := cloud.Server.URL + flavorIdentityPath + "/detail?marker=next&is_public=None"
					if page > 1 {
						rows, next = `{"id":"other","name":"unrelated"}`, ""
					}
					switch mode {
					case "same-id-duplicate":
						rows = `{"id":"7","name":"lookup"}`
					case "late-http":
						if page > 1 {
							w.WriteHeader(503)
							return
						}
					case "whole-page-decode":
						rows += `,{"id":"other","ram":false}`
					case "late-decode":
						if page > 1 {
							rows = `{"id":"other","extra_specs":{"bad":false}}`
						}
					case "invalid-row":
						rows += `,{"name":"irrelevant"}`
					case "cycle":
						if page > 1 {
							next = cloud.Server.URL + flavorIdentityPath + "/detail?marker=next&is_public=None"
						}
					}
					testcloud.JSON(w, 200, flavorIdentityPage(rows, next))
				})
				cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/7/os-extra_specs", func(w http.ResponseWriter, r *http.Request) {
					extras.Add(1)
					if lists.Load() != 2 {
						t.Error("enriched a match before closing all pages", lists.Load())
					}
					testcloud.JSON(w, 200, `{"extra_specs":{"resolved":"yes"}}`)
				})
				value, err := f.new(flavorIdentityClient(cloud))(context.Background(), "lookup", resource.WithIdentityFindExtraSpecs(true))
				if gets.Load() != 1 {
					t.Fatal("GET repeated", gets.Load())
				}
				if mode == "unique-all-pages" {
					if err != nil || value == nil || value.ID != "7" || value.Swap != 2 || value.ExtraSpecs["resolved"] != "yes" || lists.Load() != 2 || extras.Load() != 1 {
						t.Fatal(value, err, lists.Load(), extras.Load())
					}
					return
				}
				if value != nil || extras.Load() != 0 || mode != "empty-204" && err == nil || mode == "empty-204" && err != nil {
					t.Fatal("incomplete or absent lookup was enriched", value, err, lists.Load(), extras.Load())
				}
				if mode == "same-id-duplicate" && !errors.Is(err, resource.ErrAmbiguous) || mode == "invalid-row" && !errors.Is(err, resource.ErrInvalidOption) || mode == "late-http" && !gophercloud.ResponseCodeIs(err, 503) {
					t.Fatal(err)
				}
				if mode == "list-403" && !gophercloud.ResponseCodeIs(err, 403) || mode == "list-404" && !gophercloud.ResponseCodeIs(err, 404) {
					t.Fatal("list failure was ignored", err)
				}
			})
		}
	}
}

type flavorIdentityReadFailure struct{ cause error }

func (r flavorIdentityReadFailure) Read([]byte) (int, error) { return 0, r.cause }
func (r flavorIdentityReadFailure) Close() error             { return nil }

func TestNativeFindIdentityFlavorsEnrichmentFailuresAreTerminal(t *testing.T) {
	for _, f := range flavorIdentityFixtures() {
		for _, mode := range []string{"403", "404", "203", "204", "typed-decode", "transport", "read", "cancel"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := flavorIdentityClient(cloud)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var gets, extras, lists atomic.Int32
				origin := errors.New("original enrichment failure")
				transport := cloud.Provider.HTTPClient.Transport
				if transport == nil {
					transport = http.DefaultTransport
				}
				cloud.Provider.HTTPClient.Transport = nativeIdentityFindTransport(func(r *http.Request) (*http.Response, error) {
					if strings.HasSuffix(r.URL.Path, "/os-extra_specs") && (mode == "transport" || mode == "read") {
						extras.Add(1)
						if mode == "transport" {
							return nil, &url.Error{Op: "GET", URL: r.URL.String(), Err: origin}
						}
						return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: flavorIdentityReadFailure{origin}, Request: r}, nil
					}
					return transport.RoundTrip(r)
				})
				cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/lookup", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					testcloud.JSON(w, 200, `{"flavor":{"id":"7","name":"found"}}`)
				})
				cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/7/os-extra_specs", func(w http.ResponseWriter, r *http.Request) {
					extras.Add(1)
					if r.URL.RawQuery != "" {
						t.Error(r.URL)
					}
					switch mode {
					case "typed-decode":
						testcloud.JSON(w, 200, `{"extra_specs":{"value":17}}`)
					case "cancel":
						cancel()
						testcloud.JSON(w, 200, `{"extra_specs":{"value":"kept"}}`)
					default:
						status := 403
						if mode == "404" {
							status = 404
						} else if mode == "203" {
							status = 203
						} else if mode == "204" {
							status = 204
						}
						w.WriteHeader(status)
					}
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { lists.Add(1); w.WriteHeader(500) })
				value, err := f.new(client)(ctx, "lookup", resource.WithIdentityFindExtraSpecs(true), resource.WithIdentityFindQuery("vendor", "lookup-only"))
				if value != nil || err == nil || gets.Load() != 1 || extras.Load() != 1 || lists.Load() != 0 {
					t.Fatal("enrichment failure retried lookup or was ignored", value, err, gets.Load(), extras.Load(), lists.Load())
				}
				if (mode == "transport" || mode == "read") && !errors.Is(err, origin) || mode == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal("original failure cause lost", err)
				}
				for _, code := range []int{403, 404, 203, 204} {
					if mode == fmt.Sprint(code) && !gophercloud.ResponseCodeIs(err, code) {
						t.Fatal("native enrichment status lost", err)
					}
				}
			})
		}
	}
}

func TestNativeFindIdentityFlavorsMissingPoliciesAndUnsafeNames(t *testing.T) {
	for _, f := range flavorIdentityFixtures() {
		for _, code := range []int{400, 403, 404} {
			for _, policy := range []resource.FindFallbackPolicy{resource.FindFallbackCompatible, resource.FindFallbackNotFoundOnly, resource.FindFallbackNever} {
				for _, strict := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%d/policy-%d/strict-%t", f.name, code, policy, strict), func(t *testing.T) {
						cloud := testcloud.New(t)
						var gets, lists, extras atomic.Int32
						cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/lookup", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); w.WriteHeader(code) })
						cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/detail", func(w http.ResponseWriter, r *http.Request) {
							lists.Add(1)
							testcloud.JSON(w, 200, flavorIdentityPage("", ""))
						})
						cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { extras.Add(1); w.WriteHeader(500) })
						value, err := f.new(flavorIdentityClient(cloud))(context.Background(), "lookup", resource.WithIdentityFindExtraSpecs(true), resource.WithIdentityFindFallback(policy), resource.WithIdentityFindIgnoreMissing(!strict))
						fallback := policy == resource.FindFallbackCompatible || policy == resource.FindFallbackNotFoundOnly && code == 404
						if value != nil || gets.Load() != 1 || extras.Load() != 0 {
							t.Fatal(value, err, gets.Load(), extras.Load())
						}
						if fallback {
							if lists.Load() != 1 || strict && !errors.Is(err, resource.ErrNotFound) || !strict && err != nil {
								t.Fatal(err, lists.Load())
							}
						} else if lists.Load() != 0 || code == 404 && !strict && err != nil || (code != 404 || strict) && !gophercloud.ResponseCodeIs(err, code) {
							t.Fatal("GET fallback policy widened", err, lists.Load())
						}
					})
				}
			}
		}
		for _, identity := range []string{"unsafe/name", "literal%2Fname", " name with spaces "} {
			t.Run(f.name+"/unsafe/"+identity, func(t *testing.T) {
				cloud := testcloud.New(t)
				var lists, extras, gets atomic.Int32
				cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/detail", func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					if r.URL.Query().Has("name") || r.URL.Query().Get("is_public") != "None" {
						t.Error("unsafe identity became a flavor query hint", r.URL)
					}
					testcloud.JSON(w, 200, flavorIdentityPage(nativeIdentityFindRow("7", identity), ""))
				})
				cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/7/os-extra_specs", func(w http.ResponseWriter, r *http.Request) {
					extras.Add(1)
					testcloud.JSON(w, 200, `{"extra_specs":{"found":"yes"}}`)
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); w.WriteHeader(500) })
				value, err := f.new(flavorIdentityClient(cloud))(context.Background(), identity, resource.WithIdentityFindExtraSpecs(true))
				if err != nil || value == nil || value.Name != identity || gets.Load() != 0 || lists.Load() != 1 || extras.Load() != 1 {
					t.Fatal(value, err, gets.Load(), lists.Load(), extras.Load())
				}
			})
		}
	}
}

func TestNativeFindIdentityFlavorsSnapshotsConcurrentReuseAndLastWins(t *testing.T) {
	for _, f := range flavorIdentityFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, lists, extras atomic.Int32
			enrich := true
			query := url.Values{"is_public": nil, "vendor": {"first", "second"}, "name": {}}
			option := resource.WithIdentityFindOptions(resource.IdentityFindOpts{GetExtraSpecs: &enrich, Query: query})
			enrich, query["vendor"][0], query["is_public"] = false, "mutated", []string{"mutated"}
			var capturedMu sync.Mutex
			var captured *resource.IdentityFindOpts
			cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if !reflect.DeepEqual(r.URL.Query()["vendor"], []string{"first", "second"}) || r.URL.Query().Has("is_public") || r.URL.Query().Has("name") {
					t.Error("captured query changed", r.URL)
				}
				capturedMu.Lock()
				if captured != nil {
					*captured.GetExtraSpecs = false
					captured.Query.Set("is_public", "mutated-during-GET")
					captured.Query["vendor"][0] = "mutated-during-GET"
				}
				capturedMu.Unlock()
				w.WriteHeader(404)
			})
			cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/detail", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if !reflect.DeepEqual(r.URL.Query()["vendor"], []string{"first", "second"}) || r.URL.Query().Has("is_public") || r.URL.Query().Has("name") {
					t.Error("parsed snapshot or nil-key default suppression lost", r.URL)
				}
				testcloud.JSON(w, 200, flavorIdentityPage(`{"id":"7","name":"lookup"}`, ""))
			})
			cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/7/os-extra_specs", func(w http.ResponseWriter, r *http.Request) {
				extras.Add(1)
				testcloud.JSON(w, 200, `{"extra_specs":{"owned":"value"}}`)
			})
			find := f.new(flavorIdentityClient(cloud))
			var group sync.WaitGroup
			for range 8 {
				group.Add(1)
				go func() {
					defer group.Done()
					value, err := find(context.Background(), "lookup", option)
					if err != nil || value == nil || value.ExtraSpecs["owned"] != "value" {
						t.Error(value, err)
						return
					}
					value.ExtraSpecs["owned"] = "caller-mutated"
				}()
			}
			group.Wait()
			custom := func(options *resource.IdentityFindOpts) error {
				options.Query = url.Values{"is_public": nil, "name": {}, "vendor": {"first", "second"}}
				options.GetExtraSpecs = flavorIdentityBool(true)
				capturedMu.Lock()
				captured = options
				capturedMu.Unlock()
				return nil
			}
			value, err := find(context.Background(), "lookup", custom)
			if err != nil || value == nil || value.ExtraSpecs["owned"] != "value" || gets.Load() != 9 || lists.Load() != 9 || extras.Load() != 9 {
				t.Fatal("custom captured config or returned map escaped ownership", value, err, gets.Load(), lists.Load(), extras.Load())
			}
			capturedMu.Lock()
			captured = nil
			capturedMu.Unlock()
			value, err = find(context.Background(), "lookup", resource.WithIdentityFindExtraSpecs(true), option, resource.WithIdentityFindExtraSpecs(false))
			if err != nil || value == nil || value.ExtraSpecs != nil || gets.Load() != 10 || lists.Load() != 10 || extras.Load() != 9 {
				t.Fatal("last explicit false did not win", value, err, extras.Load())
			}
		})
	}
}

type flavorIdentityContextKey struct{}

func TestNativeFindIdentityFlavorsKeepsLiveProviderVersionAndContext(t *testing.T) {
	for _, f := range flavorIdentityFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.1"
			client.MoreHeaders = map[string]string{"X-Configured": "source"}
			endpoint, base := client.Endpoint, client.ResourceBase
			transport := cloud.Provider.HTTPClient.Transport
			if transport == nil {
				transport = http.DefaultTransport
			}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = nativeIdentityFindTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Context().Value(flavorIdentityContextKey{}) != "caller" {
					t.Error("caller context lost")
				}
				copy := r.Clone(r.Context())
				copy.Header.Set("X-Flavor-Middleware", "source")
				return transport.RoundTrip(copy)
			})
			check := func(r *http.Request, token string) {
				if r.Header.Get("X-Auth-Token") != token || r.Header.Get("X-Configured") != "source" || r.Header.Get("X-Flavor-Middleware") != "source" || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.1" {
					t.Error(r.URL, r.Header)
				}
			}
			cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				check(r, "test-token")
				if r.URL.Query().Get("vendor") != "lookup-only" || r.URL.Query().Has("is_public") {
					t.Error(r.URL)
				}
				cloud.Provider.SetToken("list-token")
				w.WriteHeader(404)
			})
			cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/detail", func(w http.ResponseWriter, r *http.Request) {
				check(r, "list-token")
				if r.URL.Query().Get("vendor") != "lookup-only" || r.URL.Query().Get("is_public") != "None" || r.URL.Query().Has("name") {
					t.Error(r.URL)
				}
				cloud.Provider.SetToken("extra-token")
				testcloud.JSON(w, 200, flavorIdentityPage(`{"id":"7","name":"lookup"}`, ""))
			})
			cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/7/os-extra_specs", func(w http.ResponseWriter, r *http.Request) {
				check(r, "extra-token")
				if r.URL.RawQuery != "" {
					t.Error("identity params leaked into extra specs", r.URL)
				}
				testcloud.JSON(w, 200, `{"extra_specs":{"native":"yes"}}`)
			})
			ctx := context.WithValue(context.Background(), flavorIdentityContextKey{}, "caller")
			value, err := f.new(client)(ctx, "lookup", resource.WithIdentityFindExtraSpecs(true), resource.WithIdentityFindQuery("vendor", "lookup-only"))
			if err != nil || value == nil || value.ExtraSpecs["native"] != "yes" || calls.Load() != 3 || client.Endpoint != endpoint || client.ResourceBase != base || client.Microversion != "2.1" || !reflect.DeepEqual(client.MoreHeaders, map[string]string{"X-Configured": "source"}) {
				t.Fatal("enrichment changed or cloned the live service transport", value, err, calls.Load(), client)
			}
		})
	}
}

func TestNativeFindIdentityFlavorsPreflightAndNativeGETFailures(t *testing.T) {
	for _, f := range flavorIdentityFixtures() {
		t.Run(f.name+"/preflight", func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			find := f.new(flavorIdentityClient(cloud))
			canceled, cancel := context.WithCancel(context.Background())
			cancel()
			for _, test := range []struct {
				ctx      context.Context
				identity string
				opts     []resource.IdentityFindOption
				want     error
			}{
				{nil, "lookup", nil, resource.ErrInvalidOption},
				{canceled, "lookup", nil, context.Canceled},
				{context.Background(), "lookup", []resource.IdentityFindOption{nil}, resource.ErrInvalidOption},
				{context.Background(), "lookup", []resource.IdentityFindOption{resource.WithIdentityFindQuery("GET_EXTRA_SPECS", "true")}, resource.ErrInvalidOption},
				{context.Background(), "lookup", []resource.IdentityFindOption{resource.WithIdentityFindDetails(false)}, resource.ErrUnsupported},
				{context.Background(), "lookup", []resource.IdentityFindOption{resource.WithIdentityFindDetails(true)}, resource.ErrUnsupported},
				{context.Background(), "lookup", []resource.IdentityFindOption{resource.WithIdentityFindAllProjects(false)}, resource.ErrUnsupported},
				{context.Background(), "unsafe/name", []resource.IdentityFindOption{resource.WithIdentityFindFallback(resource.FindFallbackNever)}, resource.ErrInvalidOption},
			} {
				if value, err := find(test.ctx, test.identity, test.opts...); value != nil || !errors.Is(err, test.want) {
					t.Fatal(value, err, test.want)
				}
			}
			for _, enrich := range []bool{false, true} {
				if value, err := compute.New(flavorIdentityClient(cloud), compute.Dependencies{}).Servers.FindIdentity(context.Background(), "lookup", resource.WithIdentityFindExtraSpecs(enrich)); value != nil || !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal("extra-specs control was enabled on an unaudited resource", value, err)
				}
			}
			if calls.Load() != 0 {
				t.Fatal("invalid options performed HTTP", calls.Load())
			}
		})
		for _, test := range []struct {
			name   string
			status int
			body   string
		}{
			{"unexpected-203", 203, `{"flavor":{"id":"7"}}`},
			{"unexpected-204", 204, ""},
			{"typed-malformed", 200, `{"flavor":{"id":"7","swap":"bad"}}`},
			{"nil-result", 200, `{"flavor":null}`},
			{"unsafe-returned-id", 200, `{"flavor":{"id":"unsafe/id"}}`},
		} {
			t.Run(f.name+"/"+test.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets, other atomic.Int32
				cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/lookup", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					if test.status == 204 {
						w.WriteHeader(204)
					} else {
						testcloud.JSON(w, test.status, test.body)
					}
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { other.Add(1); w.WriteHeader(500) })
				value, err := f.new(flavorIdentityClient(cloud))(context.Background(), "lookup", resource.WithIdentityFindExtraSpecs(true), resource.WithIdentityFindQuery("vendor", "get"))
				if value != nil || err == nil || gets.Load() != 1 || other.Load() != 0 {
					t.Fatal("terminal GET or invalid result was enriched", value, err, gets.Load(), other.Load())
				}
				if test.status != 200 && !gophercloud.ResponseCodeIs(err, test.status) {
					t.Fatal("native GET status lost", err)
				}
			})
		}
	}
}

var _ io.ReadCloser = flavorIdentityReadFailure{}

func TestNativeFindIdentityFlavorsOrdinaryCollectionsRetainQueryWithoutIdentityPolicies(t *testing.T) {
	for _, factory := range []struct {
		name string
		new  func(*gophercloud.ServiceClient) *resource.Collection[flavors.Flavor]
	}{
		{"generated-leaf", func(c *gophercloud.ServiceClient) *resource.Collection[flavors.Flavor] {
			return flavors.New(c).Resources
		}},
		{"handwritten-collection", func(c *gophercloud.ServiceClient) *resource.Collection[flavors.Flavor] {
			return compute.New(c, compute.Dependencies{}).Flavors
		}},
	} {
		for _, mode := range []string{"all-with-raw-query", "explicit-name-find"} {
			t.Run(factory.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				var lists, other atomic.Int32
				query := make(url.Values)
				if mode == "all-with-raw-query" {
					query = url.Values{"status": {"vendor"}, "minRam": {"1024"}, "name": {""}}
				}
				cloud.Mux.HandleFunc("GET "+flavorIdentityPath+"/detail", func(w http.ResponseWriter, r *http.Request) {
					page := lists.Add(1)
					want := make(url.Values)
					for key, values := range query {
						want[key] = append([]string(nil), values...)
					}
					if page == 2 {
						want.Set("marker", "next")
					}
					if r.URL.RawQuery != want.Encode() || r.URL.Query().Has("is_public") {
						t.Error("ordinary query was dropped or identity defaults leaked", r.URL, want)
					}
					id, name := "7", "Target"
					if mode == "all-with-raw-query" && page == 2 || mode == "explicit-name-find" && page == 1 {
						id, name = "8", "Other"
					}
					next := ""
					if page == 1 {
						want.Set("marker", "next")
						next = cloud.Server.URL + flavorIdentityPath + "/detail?" + want.Encode()
					}
					testcloud.JSON(w, 200, flavorIdentityPage(nativeIdentityFindRow(id, name), next))
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					other.Add(1)
					t.Error("ordinary collection performed a member or enrichment GET", r.URL)
					w.WriteHeader(500)
				})
				collection := factory.new(flavorIdentityClient(cloud))
				if mode == "all-with-raw-query" {
					values, err := collection.All(context.Background(), resource.WithQuery("status", "vendor"), resource.WithQuery("minRam", "1024"), resource.WithQuery("name", ""))
					if err != nil || len(values) != 2 || values[0].ID != "7" || values[1].ID != "8" || values[0].ExtraSpecs != nil || values[1].ExtraSpecs != nil {
						t.Fatal("ordinary All changed results or enriched models", values, err)
					}
				} else {
					value, err := collection.Find(context.Background(), resource.Name("Target"))
					if err != nil || value == nil || value.ID != "7" || value.Name != "Target" || value.ExtraSpecs != nil {
						t.Fatal("explicit name Find changed lookup or enriched its result", value, err)
					}
				}
				if lists.Load() != 2 || other.Load() != 0 {
					t.Fatal("ordinary pagination or route contract changed", lists.Load(), other.Load())
				}
			})
		}
	}
}
