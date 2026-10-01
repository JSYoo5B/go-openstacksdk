package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage/v3/volumes"
	"gophercloudsdk/compute/v2/aggregates"
	"gophercloudsdk/compute/v2/servers"
	"gophercloudsdk/dns/v2/recordsets"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/loadbalancer/v2/pools"
	"gophercloudsdk/network/v2/ports"
	"gophercloudsdk/resource"
)

// Pinned Resource.find tries GET, falls back only for 400/403/404, and scans
// ID OR name matches through all pages (resource.py:2439-2456,2536-2582).
// These audited native tests retain each Gophercloud getter, extractor, detail
// list and parent route. They do not claim raw HTTP metadata on typed getters.
type nativeIdentityFindView struct{ id, name string }
type nativeIdentityFindAccess func(context.Context, string, ...resource.IdentityFindOption) (*nativeIdentityFindView, error)

func nativeIdentityFindAccessFor[T any](find func(context.Context, string, ...resource.IdentityFindOption) (*T, error), id, name func(*T) string) nativeIdentityFindAccess {
	return func(ctx context.Context, identity string, options ...resource.IdentityFindOption) (*nativeIdentityFindView, error) {
		value, err := find(ctx, identity, options...)
		if value == nil {
			return nil, err
		}
		return &nativeIdentityFindView{id(value), name(value)}, err
	}
}

type nativeIdentityFindFixture struct {
	service, endpoint, base, path, singular, plural string
	regex                                           bool
	new                                             func(*testing.T, *gophercloud.ServiceClient) nativeIdentityFindAccess
}

func nativeIdentityFindFixtures() []nativeIdentityFindFixture {
	return append([]nativeIdentityFindFixture{
		{"compute", "/reverse/nova/v2.1/project", "/reverse/nova/v2.1/project/servers", "/reverse/nova/v2.1/project/servers/detail", "server", "servers", true,
			func(_ *testing.T, client *gophercloud.ServiceClient) nativeIdentityFindAccess {
				return nativeIdentityFindAccessFor(servers.New(client).FindIdentity, func(v *servers.Server) string { return v.ID }, func(v *servers.Server) string { return v.Name })
			}},
		{"volumev3", "/reverse/cinder/v3/project", "/reverse/cinder/v3/project/volumes", "/reverse/cinder/v3/project/volumes/detail", "volume", "volumes", false,
			func(_ *testing.T, client *gophercloud.ServiceClient) nativeIdentityFindAccess {
				return nativeIdentityFindAccessFor(volumes.New(client).FindIdentity, func(v *volumes.Volume) string { return v.ID }, func(v *volumes.Volume) string { return v.Name })
			}},
		{"network", "/reverse/neutron/v2.0", "/reverse/neutron/v2.0/ports", "/reverse/neutron/v2.0/ports", "port", "ports", false,
			func(_ *testing.T, client *gophercloud.ServiceClient) nativeIdentityFindAccess {
				return nativeIdentityFindAccessFor(ports.New(client).FindIdentity, func(v *ports.Port) string { return v.ID }, func(v *ports.Port) string { return v.Name })
			}},
		{"dns", "/reverse/designate/v2", "/reverse/designate/v2/zones/parent/recordsets", "/reverse/designate/v2/zones/parent/recordsets", "recordset", "recordsets", false,
			func(t *testing.T, client *gophercloud.ServiceClient) nativeIdentityFindAccess {
				scope, err := recordsets.New(client).InZone(context.Background(), resource.ID("parent"))
				if err != nil {
					t.Fatal(err)
				}
				return nativeIdentityFindAccessFor(scope.FindIdentity, func(v *recordsets.RecordSet) string { return v.ID }, func(v *recordsets.RecordSet) string { return v.Name })
			}},
		{"load-balancer", "/reverse/octavia/v2.0", "/reverse/octavia/v2.0/lbaas/pools/parent/members", "/reverse/octavia/v2.0/lbaas/pools/parent/members", "member", "members", false,
			func(t *testing.T, client *gophercloud.ServiceClient) nativeIdentityFindAccess {
				scope, err := pools.New(client).Members(context.Background(), resource.ID("parent"))
				if err != nil {
					t.Fatal(err)
				}
				return nativeIdentityFindAccessFor(scope.FindIdentity, func(v *pools.Member) string { return v.ID }, func(v *pools.Member) string { return v.Name })
			}},
	}, nativeIdentityNetworkFindFixtures()...)
}

func nativeIdentityFindClient(cloud *testcloud.Cloud, fixture nativeIdentityFindFixture) *gophercloud.ServiceClient {
	client := cloud.Client(fixture.service, "/unused/catalog")
	client.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + fixture.endpoint)
	return client
}

func nativeIdentityFindRow(id, name string) string {
	return fmt.Sprintf(`{"id":%q,"name":%q}`, id, name)
}

func nativeIdentityFindGetBody(fixture nativeIdentityFindFixture, row string) string {
	if fixture.plural == "recordsets" {
		return row // Designate returns an unwrapped RecordSet.
	}
	return fmt.Sprintf(`{%q:%s}`, fixture.singular, row)
}

func nativeIdentityFindListBody(fixture nativeIdentityFindFixture, rows, next string) string {
	links := ""
	if next != "" {
		if fixture.plural == "recordsets" || fixture.service == "identity" {
			links = fmt.Sprintf(`,"links":{"next":%q}`, next)
		} else {
			links = fmt.Sprintf(`,%q:[{"rel":"next","href":%q}]`, fixture.plural+"_links", next)
		}
	}
	return fmt.Sprintf(`{%q:[%s]%s}`, fixture.plural, rows, links)
}

func nativeIdentityFindName(fixture nativeIdentityFindFixture, identity string) string {
	if fixture.regex {
		return "^" + regexp.QuoteMeta(identity) + "$"
	}
	return identity
}

func TestNativeFindIdentitySafeGETReturnsImmediatelyWithNativeGetter(t *testing.T) {
	for _, fixture := range nativeIdentityFindFixtures() {
		t.Run(fixture.plural, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, lists atomic.Int32
			cloud.Mux.HandleFunc("GET "+fixture.base+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if r.URL.Query().Get("vendor") != "both-phases" || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Error(r.URL, r.Header)
				}
				status := 200
				if fixture.plural == "servers" {
					status = 203 // Native Nova Get explicitly accepts 200 and 203.
				}
				testcloud.JSON(w, status, nativeIdentityFindGetBody(fixture, nativeIdentityFindRow("returned-id", "DifferentName")))
			})
			cloud.Mux.HandleFunc("GET "+fixture.path, func(w http.ResponseWriter, r *http.Request) { lists.Add(1); w.WriteHeader(500) })
			find := fixture.new(t, nativeIdentityFindClient(cloud, fixture))
			value, err := find(context.Background(), "lookup", resource.WithIdentityFindQuery("vendor", "both-phases"))
			if err != nil || value == nil || value.id != "returned-id" || value.name != "DifferentName" || gets.Load() != 1 || lists.Load() != 0 {
				t.Fatal(value, err, gets.Load(), lists.Load())
			}
		})
	}
}

func TestNativeFindIdentityFallbackPoliciesAndMissingDefaults(t *testing.T) {
	for _, fixture := range nativeIdentityFindFixtures() {
		for _, status := range []int{400, 403, 404} {
			for _, policy := range []resource.FindFallbackPolicy{resource.FindFallbackCompatible, resource.FindFallbackNotFoundOnly, resource.FindFallbackNever} {
				t.Run(fmt.Sprintf("%s/%d/%d", fixture.plural, status, policy), func(t *testing.T) {
					cloud := testcloud.New(t)
					var gets, lists atomic.Int32
					cloud.Mux.HandleFunc("GET "+fixture.base+"/lookup", func(w http.ResponseWriter, r *http.Request) {
						gets.Add(1)
						if r.URL.Query().Get("vendor") != "both-phases" {
							t.Error("GET dropped the caller query", r.URL)
						}
						testcloud.JSON(w, status, `{"error":"direct failure"}`)
					})
					cloud.Mux.HandleFunc("GET "+fixture.path, func(w http.ResponseWriter, r *http.Request) {
						lists.Add(1)
						if r.URL.Query().Get("name") != nativeIdentityFindName(fixture, "lookup") || r.URL.Query().Get("vendor") != "both-phases" {
							t.Error(r.URL)
						}
						testcloud.JSON(w, 200, nativeIdentityFindListBody(fixture, nativeIdentityFindRow("found", "lookup"), ""))
					})
					find := fixture.new(t, nativeIdentityFindClient(cloud, fixture))
					value, err := find(context.Background(), "lookup", resource.WithIdentityFindFallback(policy), resource.WithIdentityFindQuery("vendor", "both-phases"))
					fallback := policy == resource.FindFallbackCompatible || policy == resource.FindFallbackNotFoundOnly && status == 404
					if fallback {
						if err != nil || value == nil || value.id != "found" || lists.Load() != 1 {
							t.Fatal(value, err, lists.Load())
						}
					} else if status == 404 {
						if value != nil || err != nil || lists.Load() != 0 {
							t.Fatal("GET-only default did not ignore 404", value, err, lists.Load())
						}
						if value, err := find(context.Background(), "lookup", resource.WithIdentityFindFallback(policy), resource.WithIdentityFindIgnoreMissing(false), resource.WithIdentityFindQuery("vendor", "both-phases")); value != nil || !errors.Is(err, resource.ErrNotFound) || !gophercloud.ResponseCodeIs(err, 404) {
							t.Fatal(value, err)
						}
					} else if value != nil || !gophercloud.ResponseCodeIs(err, status) || lists.Load() != 0 {
						t.Fatal(value, err, lists.Load())
					}
					if gets.Load() < 1 {
						t.Fatal("safe identity skipped GET")
					}
				})
			}
		}
	}
	t.Run("successful-list-absence-is-logical", func(t *testing.T) {
		fixture := nativeIdentityFindFixtures()[0]
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("GET "+fixture.base+"/lookup", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 403, `{"error":"forbidden GET"}`) })
		cloud.Mux.HandleFunc("GET "+fixture.path, func(w http.ResponseWriter, r *http.Request) {
			testcloud.JSON(w, 200, nativeIdentityFindListBody(fixture, "", ""))
		})
		find := fixture.new(t, nativeIdentityFindClient(cloud, fixture))
		if value, err := find(context.Background(), "lookup"); value != nil || err != nil {
			t.Fatal(value, err)
		}
		if value, err := find(context.Background(), "lookup", resource.WithIdentityFindIgnoreMissing(false)); value != nil || !errors.Is(err, resource.ErrNotFound) || gophercloud.ResponseCodeIs(err, 403) {
			t.Fatal("logical absence retained suppressed direct error", value, err)
		}
	})
}

func TestNativeFindIdentityUnsafeNamesRemainLiteralQueryOnlyAndIDMatches(t *testing.T) {
	for _, fixture := range nativeIdentityFindFixtures() {
		for _, identity := range []string{"name/with/slash", "name%2Fstill-literal", " web[1]. ", "web\u00a0name"} {
			t.Run(fixture.plural+"/"+identity, func(t *testing.T) {
				cloud := testcloud.New(t)
				var lists, other atomic.Int32
				cloud.Mux.HandleFunc("GET "+fixture.path, func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					if r.URL.Query().Get("name") != nativeIdentityFindName(fixture, identity) {
						t.Error(r.URL)
					}
					if strings.Contains(identity, "%2F") && !strings.Contains(r.URL.RawQuery, "%252F") {
						t.Error("literal percent string was decoded", r.URL.RawQuery)
					}
					testcloud.JSON(w, 200, nativeIdentityFindListBody(fixture, nativeIdentityFindRow("found", identity), ""))
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { other.Add(1); w.WriteHeader(500) })
				find := fixture.new(t, nativeIdentityFindClient(cloud, fixture))
				value, err := find(context.Background(), identity)
				if err != nil || value == nil || value.id != "found" || value.name != identity || lists.Load() != 1 || other.Load() != 0 {
					t.Fatal(value, err, lists.Load(), other.Load())
				}
				if _, err := find(context.Background(), identity, resource.WithIdentityFindFallback(resource.FindFallbackNever)); !errors.Is(err, resource.ErrInvalidOption) || lists.Load() != 1 || other.Load() != 0 {
					t.Fatal(err, lists.Load(), other.Load())
				}
			})
		}
		t.Run(fixture.plural+"/caller-query-and-ID-only-match", func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("GET "+fixture.base+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("name") != "caller-pattern" || r.URL.Query().Get("vendor") != "kept" {
					t.Error("caller query was not preserved in GET", r.URL)
				}
				w.WriteHeader(404)
			})
			cloud.Mux.HandleFunc("GET "+fixture.path, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("name") != "caller-pattern" || r.URL.Query().Get("vendor") != "kept" {
					t.Error(r.URL)
				}
				rows := nativeIdentityFindRow("other", "caller-pattern") + "," + nativeIdentityFindRow("lookup", "DifferentName")
				testcloud.JSON(w, 200, nativeIdentityFindListBody(fixture, rows, ""))
			})
			value, err := fixture.new(t, nativeIdentityFindClient(cloud, fixture))(context.Background(), "lookup", resource.WithIdentityFindQuery("name", "caller-pattern"), resource.WithIdentityFindQuery("vendor", "kept"))
			if err != nil || value == nil || value.id != "lookup" || value.name != "DifferentName" {
				t.Fatal("local name filter removed ID-only match", value, err)
			}
		})
	}
}

func TestNativeFindIdentityAllPagesDuplicatesAndTerminalErrorsAfterMatch(t *testing.T) {
	for _, fixture := range nativeIdentityFindFixtures() {
		for _, mode := range []string{"unique", "duplicate", "same-id-duplicate", "late-http", "late-decode", "whole-page-decode", "cycle", "cancel"} {
			t.Run(fixture.plural+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var lists atomic.Int32
				cloud.Mux.HandleFunc("GET "+fixture.base+"/lookup", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) })
				cloud.Mux.HandleFunc("GET "+fixture.path, func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					if r.URL.Query().Get("marker") == "" {
						rows := nativeIdentityFindRow("first", "lookup")
						next := cloud.Server.URL + fixture.path + "?marker=next&name=" + url.QueryEscape(nativeIdentityFindName(fixture, "lookup"))
						if mode == "cycle" {
							next = cloud.Server.URL + r.URL.RequestURI()
						}
						if mode == "whole-page-decode" {
							rows += `,{"id":false,"name":"unrelated"}`
						}
						testcloud.JSON(w, 200, nativeIdentityFindListBody(fixture, rows, next))
						return
					}
					if mode == "late-http" {
						testcloud.JSON(w, 500, `{"error":"later page"}`)
						return
					}
					if mode == "cancel" {
						cancel()
					}
					row := nativeIdentityFindRow("unrelated", "Elsewhere")
					if mode == "duplicate" {
						row = nativeIdentityFindRow("second", "lookup")
					} else if mode == "same-id-duplicate" {
						row = nativeIdentityFindRow("first", "lookup")
					} else if mode == "late-decode" {
						row = `{"id":false,"name":"Elsewhere"}`
					}
					testcloud.JSON(w, 200, nativeIdentityFindListBody(fixture, row, ""))
				})
				value, err := fixture.new(t, nativeIdentityFindClient(cloud, fixture))(ctx, "lookup")
				wantLists := int32(2)
				if mode == "cycle" || mode == "whole-page-decode" {
					wantLists = 1
				}
				if lists.Load() != wantLists {
					t.Fatal(lists.Load(), err)
				}
				if mode == "unique" {
					if err != nil || value == nil || value.id != "first" {
						t.Fatal(value, err)
					}
				} else if value != nil || err == nil {
					t.Fatal("match hid later failure or duplicate", value, err)
				} else if (mode == "duplicate" || mode == "same-id-duplicate") && !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(err)
				} else if mode == "late-http" && !gophercloud.ResponseCodeIs(err, 500) || mode == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestNativeFindIdentityListHTTPAbsenceIsNeverIgnored(t *testing.T) {
	for _, fixture := range nativeIdentityFindFixtures() {
		for _, status := range []int{403, 404} {
			t.Run(fmt.Sprintf("%s/%d", fixture.plural, status), func(t *testing.T) {
				cloud := testcloud.New(t)
				var lists atomic.Int32
				cloud.Mux.HandleFunc("GET "+fixture.base+"/lookup", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) })
				cloud.Mux.HandleFunc("GET "+fixture.path, func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					testcloud.JSON(w, status, `{"error":"list failure"}`)
				})
				value, err := fixture.new(t, nativeIdentityFindClient(cloud, fixture))(context.Background(), "lookup", resource.WithIdentityFindIgnoreMissing(true))
				if value != nil || !gophercloud.ResponseCodeIs(err, status) || lists.Load() != 1 {
					t.Fatal(value, err, lists.Load())
				}
			})
		}
	}
}

func TestNativeFindIdentityValidatesOptionsInputAndContextBeforeHTTP(t *testing.T) {
	fixture := nativeIdentityFindFixtures()[0]
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	find := fixture.new(t, nativeIdentityFindClient(cloud, fixture))
	for _, identity := range []string{"", " \u00a0 ", "line\nname", "nul\x00name", string([]byte{0xff})} {
		if value, err := find(context.Background(), identity); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(identity, value, err)
		}
	}
	invalid := []resource.IdentityFindOption{nil, resource.WithIdentityFindFallback(resource.FindFallbackPolicy(99))}
	for _, key := range []string{"", "headers", "microversion", "base_path", "list_base_path", "max_items", "paginated", "jmespath_filters", "ignore_missing", "fallback", "allow_unknown_params"} {
		invalid = append(invalid, resource.WithIdentityFindQuery(key, "invalid"), resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: url.Values{key: {"invalid"}}}))
	}
	for _, option := range invalid {
		if value, err := find(context.Background(), "lookup", option); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(value, err)
		}
	}
	sentinel := errors.New("custom option failed")
	if value, err := find(context.Background(), "lookup", func(*resource.IdentityFindOpts) error { return sentinel }); value != nil || !errors.Is(err, sentinel) {
		t.Fatal(value, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if value, err := find(ctx, "lookup"); value != nil || !errors.Is(err, context.Canceled) {
		t.Fatal(value, err)
	}
	if value, err := find(nil, "lookup"); value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
		t.Fatal(value, err, calls.Load())
	}
}

func TestNativeFindIdentityQuerySnapshotsRepeatedValuesAndConcurrentReuse(t *testing.T) {
	for _, fixture := range nativeIdentityFindFixtures() {
		for _, mode := range []string{"repeated", "nil-name", "zero-values-name", "empty-name"} {
			t.Run(fixture.plural+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				var captured *resource.IdentityFindOpts
				var gets, lists atomic.Int32
				checkQuery := func(r *http.Request) {
					q := r.URL.Query()
					if !reflect.DeepEqual(q["tags"], []string{"first", "second"}) || !reflect.DeepEqual(q["fields"], []string{"id", "name"}) || mode == "repeated" && q.Get("name") != "explicit-pattern" || (mode == "nil-name" || mode == "zero-values-name") && q.Has("name") || mode == "empty-name" && (!q.Has("name") || q.Get("name") != "") {
						t.Error("query ownership or name-key presence changed", r.URL)
					}
				}
				cloud.Mux.HandleFunc("GET "+fixture.base+"/snapshot", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					checkQuery(r)
					if captured != nil {
						captured.Query.Set("name", "mutated")
						captured.Query["tags"][0] = "mutated"
						*captured.IgnoreMissing = true
						captured.Fallback = resource.FindFallbackNever
					}
					w.WriteHeader(404)
				})
				cloud.Mux.HandleFunc("GET "+fixture.base+"/reuse", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					checkQuery(r)
					w.WriteHeader(404)
				})
				cloud.Mux.HandleFunc("GET "+fixture.path, func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					checkQuery(r)
					q := r.URL.Query()
					rows := nativeIdentityFindRow("found", q.Get("identity"))
					if q.Get("phase") == "missing" {
						rows = ""
					}
					testcloud.JSON(w, 200, nativeIdentityFindListBody(fixture, rows, ""))
				})
				ignore := false
				query := url.Values{"name": {"explicit-pattern"}, "tags": {"first", "second"}, "fields": {"id", "name"}}
				if mode == "nil-name" {
					query["name"] = nil
				} else if mode == "zero-values-name" {
					query["name"] = []string{}
				} else if mode == "empty-name" {
					query["name"] = []string{""}
				}
				option := resource.WithIdentityFindOptions(resource.IdentityFindOpts{IgnoreMissing: &ignore, Query: query})
				query["tags"][0], ignore = "caller-mutated", true
				query.Set("name", "caller-mutated")
				find := fixture.new(t, nativeIdentityFindClient(cloud, fixture))
				capture := func(options *resource.IdentityFindOpts) error { captured = options; return nil }
				value, err := find(context.Background(), "snapshot", option, resource.WithIdentityFindQuery("identity", "snapshot"), capture)
				if err != nil || value == nil || value.id != "found" {
					t.Fatal(value, err)
				}
				captured = nil
				if value, err := find(context.Background(), "snapshot", option, resource.WithIdentityFindQuery("phase", "missing")); value != nil || !errors.Is(err, resource.ErrNotFound) {
					t.Fatal("bulk option lost its ignoreMissing snapshot", value, err)
				}
				var wait sync.WaitGroup
				for i := 0; i < 2; i++ {
					wait.Add(1)
					go func() {
						defer wait.Done()
						value, err := find(context.Background(), "reuse", option, resource.WithIdentityFindQuery("identity", "reuse"))
						if err != nil || value == nil || value.id != "found" {
							t.Error(value, err)
						}
					}()
				}
				wait.Wait()
				if gets.Load() != 4 || lists.Load() != 4 {
					t.Fatal(gets.Load(), lists.Load())
				}
			})
		}
	}
}

type nativeIdentityFindTransport func(*http.Request) (*http.Response, error)

func (transport nativeIdentityFindTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return transport(r)
}

func TestNativeFindIdentityDecodeAndTransportFailuresNeverFallback(t *testing.T) {
	for _, fixture := range nativeIdentityFindFixtures() {
		for _, withQuery := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/decode/query-%t", fixture.plural, withQuery), func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets, lists atomic.Int32
				cloud.Mux.HandleFunc("GET "+fixture.base+"/lookup", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					testcloud.JSON(w, 200, nativeIdentityFindGetBody(fixture, `{"id":false,"name":"bad"}`))
				})
				cloud.Mux.HandleFunc("GET "+fixture.path, func(w http.ResponseWriter, r *http.Request) { lists.Add(1); w.WriteHeader(500) })
				var options []resource.IdentityFindOption
				if withQuery {
					options = append(options, resource.WithIdentityFindQuery("fields", "id"))
				}
				value, err := fixture.new(t, nativeIdentityFindClient(cloud, fixture))(context.Background(), "lookup", options...)
				if value != nil || err == nil || gets.Load() != 1 || lists.Load() != 0 {
					t.Fatal(value, err, gets.Load(), lists.Load())
				}
			})
		}
	}
	for _, withQuery := range []bool{false, true} {
		t.Run(fmt.Sprintf("transport-wrapping-native-404/query-%t", withQuery), func(t *testing.T) {
			fixture := nativeIdentityFindFixtures()[0]
			cloud := testcloud.New(t)
			client := nativeIdentityFindClient(cloud, fixture)
			original := &gophercloud.ErrUnexpectedResponseCode{Actual: 404}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = nativeIdentityFindTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				return nil, &url.Error{Op: "GET", URL: r.URL.String(), Err: original}
			})
			var options []resource.IdentityFindOption
			if withQuery {
				options = append(options, resource.WithIdentityFindQuery("fields", "id"))
			}
			value, err := fixture.new(t, client)(context.Background(), "lookup", options...)
			var transport *url.Error
			if value != nil || !errors.As(err, &transport) || !errors.Is(err, original) || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
		})
	}
}

func TestNativeFindIdentityExplicitOptInExcludesOtherBindings(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	compute := aggregates.New(cloud.Client("compute", "/v2.1/project"))
	loadbalancer := pools.New(cloud.Client("load-balancer", "/v2.0"))
	if value, err := compute.Resources.FindIdentity(context.Background(), "lookup"); value != nil || !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(value, err)
	}
	if value, err := loadbalancer.Resources.FindIdentity(context.Background(), "lookup"); value != nil || !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(value, err)
	}
	for _, api := range []any{compute, loadbalancer, recordsets.New(cloud.Client("dns", "/v2"))} {
		if _, enabled := reflect.TypeOf(api).MethodByName("FindIdentity"); enabled {
			t.Fatalf("unscoped or unaudited API gained identity fallback: %T", api)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("unsupported binding performed HTTP", calls.Load())
	}
}
