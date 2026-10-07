package api_test

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

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/gophercloudsdk/compute"
	"github.com/JSYoo5B/gophercloudsdk/compute/v2/servers"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/ports"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Pinned find_server/find_volume default to details=True/all_projects=False
// (_proxy.py:1023-1061 / :855-892). Resource.find:2556-2563 applies both
// policies after the direct GET. A summary result is returned without fetching
// detail. Tests retain native ServerPage/VolumePage and their complete models.
type identityModesValue struct {
	id, name, status string
	size             int
}

type identityModesFind func(context.Context, string, ...resource.IdentityFindOption) (*identityModesValue, error)

func identityModesAccess[T any](find func(context.Context, string, ...resource.IdentityFindOption) (*T, error), convert func(*T) *identityModesValue) identityModesFind {
	return func(ctx context.Context, identity string, options ...resource.IdentityFindOption) (*identityModesValue, error) {
		value, err := find(ctx, identity, options...)
		if value == nil {
			return nil, err
		}
		return convert(value), err
	}
}

type identityModesFixture struct {
	name   string
	native nativeIdentityFindFixture
	new    func(*gophercloud.ServiceClient) identityModesFind
}

func identityModesFixtures() []identityModesFixture {
	server, volume := nativeIdentityFindFixtures()[0], nativeIdentityFindFixtures()[1]
	serverValue := func(v *servers.Server) *identityModesValue {
		return &identityModesValue{id: v.ID, name: v.Name, status: v.Status}
	}
	volumeValue := func(v *volumes.Volume) *identityModesValue {
		return &identityModesValue{id: v.ID, name: v.Name, status: v.Status, size: v.Size}
	}
	return []identityModesFixture{
		{"server-leaf", server, func(c *gophercloud.ServiceClient) identityModesFind {
			return identityModesAccess(servers.New(c).FindIdentity, serverValue)
		}},
		{"server-high-level", server, func(c *gophercloud.ServiceClient) identityModesFind {
			return identityModesAccess(compute.New(c, compute.Dependencies{}).Servers.FindIdentity, serverValue)
		}},
		{"volume-leaf", volume, func(c *gophercloud.ServiceClient) identityModesFind {
			return identityModesAccess(volumes.New(c).FindIdentity, volumeValue)
		}},
		{"volume-high-level", volume, func(c *gophercloud.ServiceClient) identityModesFind {
			return identityModesAccess(blockstorage.New(c).Volumes.FindIdentity, volumeValue)
		}},
	}
}

func identityModesBool(value bool) *bool { return &value }

func identityModesRow(f identityModesFixture, name string, detailed bool) string {
	row := fmt.Sprintf(`{"id":"found","name":%q`, name)
	if detailed {
		row += `,"status":"ACTIVE"`
		if f.native.plural == "volumes" {
			row += `,"size":23`
		}
	}
	return row + "}"
}

func identityModesCheckFields(t *testing.T, f identityModesFixture, value *identityModesValue, detailed bool) {
	t.Helper()
	if value == nil || value.id != "found" {
		t.Fatal("missing native identity result", value)
	}
	if detailed && value.status != "ACTIVE" || !detailed && (value.status != "" || value.size != 0) || detailed && f.native.plural == "volumes" && value.size != 23 {
		t.Fatal("native detail/summary fields changed", value, detailed)
	}
}

func TestNativeFindIdentityModesDefaultsRoutesAndListOnlyFlags(t *testing.T) {
	modes := []struct {
		name         string
		details, all *bool
		detailed     bool
		allWire      string
	}{
		{"defaults", nil, nil, true, ""},
		{"explicit-defaults", identityModesBool(true), identityModesBool(false), true, ""},
		{"summary", identityModesBool(false), identityModesBool(false), false, ""},
		{"summary-all-projects", identityModesBool(false), identityModesBool(true), false, "true"},
		{"detail-all-projects", identityModesBool(true), identityModesBool(true), true, "true"},
	}
	for _, f := range identityModesFixtures() {
		for _, mode := range modes {
			t.Run(f.name+"/"+mode.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets, lists atomic.Int32
				cloud.Mux.HandleFunc("GET "+f.native.base+"/lookup", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					q := r.URL.Query()
					if q.Has("all_tenants") || q.Has("all_projects") || q.Has("details") || !reflect.DeepEqual(q["vendor"], []string{"first", "second"}) {
						t.Error("typed list flags leaked into GET or raw query was lost", r.URL)
					}
					w.WriteHeader(404)
				})
				listPath := f.native.base
				if mode.detailed {
					listPath += "/detail"
				}
				cloud.Mux.HandleFunc("GET "+listPath, func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					q := r.URL.Query()
					if q.Get("all_tenants") != mode.allWire || mode.allWire == "" && q.Has("all_tenants") || q.Has("details") || q.Has("all_projects") || q.Get("name") != nativeIdentityFindName(f.native, "lookup") || !reflect.DeepEqual(q["vendor"], []string{"first", "second"}) {
						t.Error("list route or policy query changed", r.URL)
					}
					testcloud.JSON(w, 200, nativeIdentityFindListBody(f.native, identityModesRow(f, "lookup", mode.detailed), ""))
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Error("unexpected route", r.URL); w.WriteHeader(500) })
				options := resource.IdentityFindOpts{Details: mode.details, AllProjects: mode.all, Query: url.Values{"vendor": {"first", "second"}}}
				value, err := f.new(nativeIdentityFindClient(cloud, f.native))(context.Background(), "lookup", resource.WithIdentityFindOptions(options))
				if err != nil || gets.Load() != 1 || lists.Load() != 1 {
					t.Fatal(value, err, gets.Load(), lists.Load())
				}
				identityModesCheckFields(t, f, value, mode.detailed)
			})
		}
		t.Run(f.name+"/successful-GET-remains-detailed", func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+f.native.base+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Error("list flags changed GET", r.URL)
				}
				testcloud.JSON(w, 200, nativeIdentityFindGetBody(f.native, identityModesRow(f, "lookup", true)))
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				t.Error("implicit list/detail fetch", r.URL)
				w.WriteHeader(500)
			})
			value, err := f.new(nativeIdentityFindClient(cloud, f.native))(context.Background(), "lookup", resource.WithIdentityFindDetails(false), resource.WithIdentityFindAllProjects(true))
			if err != nil || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
			identityModesCheckFields(t, f, value, true)
		})
	}
}

func TestNativeFindIdentityModesOwnedSnapshotsLastWinsAndConcurrentReuse(t *testing.T) {
	for _, f := range identityModesFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, lists atomic.Int32
			var captured *resource.IdentityFindOpts
			cloud.Mux.HandleFunc("GET "+f.native.base+"/{id}", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if !reflect.DeepEqual(r.URL.Query()["tags"], []string{"first", "second"}) || r.URL.Query().Has("all_tenants") {
					t.Error("GET lost query snapshot or gained typed scope", r.URL)
				}
				if captured != nil {
					*captured.Details, *captured.AllProjects, *captured.IgnoreMissing = true, false, true
					captured.Query["tags"][0] = "captured mutation"
				}
				w.WriteHeader(404)
			})
			respond := func(detailed bool) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					q := r.URL.Query()
					lastWins := q.Get("identity") == "last-wins"
					if detailed != lastWins || !reflect.DeepEqual(q["tags"], []string{"first", "second"}) || !lastWins && q.Get("all_tenants") != "true" || lastWins && q.Has("all_tenants") {
						t.Error("frozen modes or last-wins policy changed", r.URL)
					}
					row := identityModesRow(f, q.Get("identity"), detailed)
					if q.Get("phase") == "missing" {
						row = ""
					}
					testcloud.JSON(w, 200, nativeIdentityFindListBody(f.native, row, ""))
				}
			}
			cloud.Mux.HandleFunc("GET "+f.native.base, respond(false))
			cloud.Mux.HandleFunc("GET "+f.native.base+"/detail", respond(true))
			details, all, ignore := false, true, false
			query := url.Values{"tags": {"first", "second"}}
			frozen := resource.WithIdentityFindOptions(resource.IdentityFindOpts{Details: &details, AllProjects: &all, IgnoreMissing: &ignore, Query: query})
			details, all, ignore, query["tags"][0] = true, false, true, "caller mutation"
			find := f.new(nativeIdentityFindClient(cloud, f.native))
			capture := func(options *resource.IdentityFindOpts) error { captured = options; return nil }
			value, err := find(context.Background(), "snapshot", frozen, resource.WithIdentityFindQuery("identity", "snapshot"), capture)
			if err != nil {
				t.Fatal(err)
			}
			identityModesCheckFields(t, f, value, false)
			captured = nil
			if value, err := find(context.Background(), "snapshot", frozen, resource.WithIdentityFindQuery("phase", "missing")); value != nil || !errors.Is(err, resource.ErrNotFound) {
				t.Fatal("ignoreMissing pointer snapshot lost", value, err)
			}
			value, err = find(context.Background(), "last-wins", frozen, resource.WithIdentityFindQuery("identity", "last-wins"), resource.WithIdentityFindDetails(true), resource.WithIdentityFindAllProjects(false))
			if err != nil {
				t.Fatal(err)
			}
			identityModesCheckFields(t, f, value, true)
			var wait sync.WaitGroup
			for range 2 {
				wait.Add(1)
				go func() {
					defer wait.Done()
					value, err := find(context.Background(), "reuse", frozen, resource.WithIdentityFindQuery("identity", "reuse"))
					if err != nil || value == nil || value.id != "found" || value.status != "" || value.size != 0 {
						t.Error(value, err)
					}
				}()
			}
			wait.Wait()
			if gets.Load() != 5 || lists.Load() != 5 {
				t.Fatal(gets.Load(), lists.Load())
			}
		})
	}
}

func TestNativeFindIdentityModesConflictsAndCapabilitiesBeforeHTTP(t *testing.T) {
	for _, f := range identityModesFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, nativeIdentityFindGetBody(f.native, identityModesRow(f, "lookup", true)))
			})
			find := f.new(nativeIdentityFindClient(cloud, f.native))
			for _, enabled := range []bool{false, true} {
				for _, key := range []string{"all_tenants", "ALL_TENANTS"} {
					for _, values := range [][]string{nil, {}, {""}, {"true", "false"}} {
						raw := func(options *resource.IdentityFindOpts) error {
							options.Query[key] = append([]string(nil), values...)
							return nil
						}
						for _, options := range [][]resource.IdentityFindOption{{raw, resource.WithIdentityFindAllProjects(enabled)}, {resource.WithIdentityFindAllProjects(enabled), raw}} {
							if value, err := find(context.Background(), "lookup", options...); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
								t.Fatal("raw scope conflict accepted", key, values, value, err)
							}
						}
					}
				}
			}
			for _, key := range []string{"details", "DETAILS", "all_projects", "All_Projects"} {
				for _, option := range []resource.IdentityFindOption{resource.WithIdentityFindQuery(key, "true"), resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: url.Values{key: {"true"}}})} {
					if value, err := find(context.Background(), "lookup", option); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal("local control became a wire query", key, value, err)
					}
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := find(ctx, "lookup", resource.WithIdentityFindDetails(false), resource.WithIdentityFindAllProjects(true)); !errors.Is(err, context.Canceled) || calls.Load() != 0 {
				t.Fatal(err, calls.Load())
			}
		})
	}
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(200) })
	find := ports.New(cloud.Client("network", "/v2.0")).FindIdentity
	for _, value := range []bool{false, true} {
		for _, option := range []resource.IdentityFindOption{resource.WithIdentityFindDetails(value), resource.WithIdentityFindAllProjects(value)} {
			if result, err := find(context.Background(), "lookup", option); result != nil || !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 0 {
				t.Fatal("unsupported explicit mode reached HTTP", result, err, calls.Load())
			}
		}
	}
}

func TestNativeFindIdentityModesFallbackPoliciesNativeCodesAndRawScopeExtension(t *testing.T) {
	for _, f := range identityModesFixtures() {
		for _, status := range []int{400, 403, 404} {
			for _, policy := range []resource.FindFallbackPolicy{resource.FindFallbackCompatible, resource.FindFallbackNotFoundOnly, resource.FindFallbackNever} {
				t.Run(fmt.Sprintf("%s/fallback-%d-%d", f.name, status, policy), func(t *testing.T) {
					cloud := testcloud.New(t)
					var lists atomic.Int32
					cloud.Mux.HandleFunc("GET "+f.native.base+"/lookup", func(w http.ResponseWriter, r *http.Request) {
						if r.URL.Query().Get("vendor") != "kept" || r.URL.Query().Has("all_tenants") {
							t.Error(r.URL)
						}
						w.WriteHeader(status)
					})
					cloud.Mux.HandleFunc("GET "+f.native.base, func(w http.ResponseWriter, r *http.Request) {
						lists.Add(1)
						if r.URL.Query().Get("all_tenants") != "true" || r.URL.Query().Get("vendor") != "kept" {
							t.Error(r.URL)
						}
						testcloud.JSON(w, 200, nativeIdentityFindListBody(f.native, identityModesRow(f, "lookup", false), ""))
					})
					value, err := f.new(nativeIdentityFindClient(cloud, f.native))(context.Background(), "lookup", resource.WithIdentityFindDetails(false), resource.WithIdentityFindAllProjects(true), resource.WithIdentityFindFallback(policy), resource.WithIdentityFindIgnoreMissing(false), resource.WithIdentityFindQuery("vendor", "kept"))
					fallback := policy == resource.FindFallbackCompatible || policy == resource.FindFallbackNotFoundOnly && status == 404
					if fallback {
						if err != nil || value == nil || lists.Load() != 1 {
							t.Fatal(value, err, lists.Load())
						}
					} else if value != nil || !gophercloud.ResponseCodeIs(err, status) || lists.Load() != 0 {
						t.Fatal(value, err, lists.Load())
					}
				})
			}
		}
		for _, status := range []int{200, 203, 204, 300} {
			t.Run(fmt.Sprintf("%s/GET-%d", f.name, status), func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+f.native.base+"/lookup", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					testcloud.JSON(w, status, nativeIdentityFindGetBody(f.native, identityModesRow(f, "lookup", true)))
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
				value, err := f.new(nativeIdentityFindClient(cloud, f.native))(context.Background(), "lookup", resource.WithIdentityFindDetails(false), resource.WithIdentityFindAllProjects(true), resource.WithIdentityFindQuery("vendor", "kept"))
				accepted := status == 200 || status == 203 && f.native.plural == "servers"
				if accepted {
					if err != nil {
						t.Fatal(err)
					}
					identityModesCheckFields(t, f, value, true)
				} else if value != nil || !gophercloud.ResponseCodeIs(err, status) {
					t.Fatal(value, err)
				}
				if calls.Load() != 1 {
					t.Fatal("unexpected getter code caused fallback", calls.Load())
				}
			})
		}
		for _, status := range []int{200, 204, 300} {
			t.Run(fmt.Sprintf("%s/LIST-%d-raw-scope", f.name, status), func(t *testing.T) {
				cloud := testcloud.New(t)
				cloud.Mux.HandleFunc("GET "+f.native.base+"/lookup", func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Query().Get("all_tenants") != "1" {
						t.Error(r.URL)
					}
					w.WriteHeader(404)
				})
				cloud.Mux.HandleFunc("GET "+f.native.base, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Query().Get("all_tenants") != "1" {
						t.Error(r.URL)
					}
					if status == 204 {
						w.WriteHeader(204)
						return
					}
					testcloud.JSON(w, status, nativeIdentityFindListBody(f.native, identityModesRow(f, "lookup", false), ""))
				})
				find := f.new(nativeIdentityFindClient(cloud, f.native))
				options := []resource.IdentityFindOption{resource.WithIdentityFindDetails(false), resource.WithIdentityFindQuery("all_tenants", "1")}
				value, err := find(context.Background(), "lookup", options...)
				if status == 204 {
					if value != nil || err != nil {
						t.Fatal(value, err)
					}
					if value, err := find(context.Background(), "lookup", append(options, resource.WithIdentityFindIgnoreMissing(false))...); value != nil || !errors.Is(err, resource.ErrNotFound) {
						t.Fatal(value, err)
					}
				} else {
					if err != nil {
						t.Fatal(err)
					}
					identityModesCheckFields(t, f, value, false)
				}
			})
		}
	}
}

func TestNativeFindIdentityModesSummaryPagesDuplicatesAndTerminalFailures(t *testing.T) {
	for _, f := range identityModesFixtures() {
		for _, mode := range []string{"unique", "duplicate", "same-id", "late-500", "list-403", "list-404", "whole-page-decode", "late-decode", "cycle", "cancel"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var gets, lists atomic.Int32
				cloud.Mux.HandleFunc("GET "+f.native.base+"/lookup", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); w.WriteHeader(403) })
				bad := `{"id":"unrelated","name":"Elsewhere","status":false}`
				if f.native.plural == "volumes" {
					bad = `{"id":"unrelated","name":"Elsewhere","size":"bad"}`
				}
				cloud.Mux.HandleFunc("GET "+f.native.base, func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					if mode == "list-403" || mode == "list-404" {
						code := 403
						if mode == "list-404" {
							code = 404
						}
						w.WriteHeader(code)
						return
					}
					if r.URL.Query().Get("all_tenants") != "true" {
						t.Error("scope missing on advertised page", r.URL)
					}
					if r.URL.Query().Get("marker") == "" {
						rows := identityModesRow(f, "lookup", false)
						next := cloud.Server.URL + f.native.base + "?marker=next&all_tenants=true&name=" + url.QueryEscape(nativeIdentityFindName(f.native, "lookup"))
						if mode == "whole-page-decode" {
							rows += "," + bad
						}
						if mode == "cycle" {
							next = cloud.Server.URL + r.URL.RequestURI()
						}
						testcloud.JSON(w, 200, nativeIdentityFindListBody(f.native, rows, next))
						return
					}
					if mode == "late-500" {
						w.WriteHeader(500)
						return
					}
					if mode == "cancel" {
						cancel()
					}
					row := nativeIdentityFindRow("other", "Elsewhere")
					if mode == "duplicate" {
						row = nativeIdentityFindRow("second", "lookup")
					}
					if mode == "same-id" {
						row = identityModesRow(f, "lookup", false)
					}
					if mode == "late-decode" {
						row = bad
					}
					testcloud.JSON(w, 200, nativeIdentityFindListBody(f.native, row, ""))
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					t.Error("summary routed through detail", r.URL)
					w.WriteHeader(500)
				})
				value, err := f.new(nativeIdentityFindClient(cloud, f.native))(ctx, "lookup", resource.WithIdentityFindDetails(false), resource.WithIdentityFindAllProjects(true))
				wantLists := int32(2)
				if mode == "cycle" || mode == "whole-page-decode" || mode == "list-403" || mode == "list-404" {
					wantLists = 1
				}
				if gets.Load() != 1 || lists.Load() != wantLists {
					t.Fatal(gets.Load(), lists.Load(), err)
				}
				if mode == "unique" {
					if err != nil {
						t.Fatal(err)
					}
					identityModesCheckFields(t, f, value, false)
				} else if value != nil || err == nil {
					t.Fatal("match hid later error/duplicate", value, err)
				}
				if (mode == "duplicate" || mode == "same-id") && !errors.Is(err, resource.ErrAmbiguous) || mode == "cancel" && !errors.Is(err, context.Canceled) || mode == "late-500" && !gophercloud.ResponseCodeIs(err, 500) || mode == "list-403" && !gophercloud.ResponseCodeIs(err, 403) || mode == "list-404" && !gophercloud.ResponseCodeIs(err, 404) {
					t.Fatal(err)
				}
			})
		}
	}
}

type identityModesContextKey struct{}

func TestNativeFindIdentityModesPreserveSourceClientAndContext(t *testing.T) {
	for _, f := range identityModesFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := nativeIdentityFindClient(cloud, f.native)
			client.MoreHeaders = map[string]string{"X-Configured": "source"}
			endpoint, base := client.Endpoint, client.ResourceBase
			original := cloud.Provider.HTTPClient.Transport
			if original == nil {
				original = http.DefaultTransport
			}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = nativeIdentityFindTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Context().Value(identityModesContextKey{}) != "caller" {
					t.Error("caller context lost")
				}
				copy := r.Clone(r.Context())
				copy.Header.Set("X-Mode-Middleware", "source")
				return original.RoundTrip(copy)
			})
			check := func(r *http.Request, token string) {
				if r.Header.Get("X-Configured") != "source" || r.Header.Get("X-Mode-Middleware") != "source" || r.Header.Get("X-Auth-Token") != token || r.URL.Query().Get("vendor") != "kept" {
					t.Error(r.URL, r.Header)
				}
			}
			cloud.Mux.HandleFunc("GET "+f.native.base+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				check(r, "test-token")
				if r.URL.Query().Has("all_tenants") {
					t.Error(r.URL)
				}
				cloud.Provider.SetToken("changed-token")
				w.WriteHeader(404)
			})
			cloud.Mux.HandleFunc("GET "+f.native.base, func(w http.ResponseWriter, r *http.Request) {
				check(r, "changed-token")
				if r.URL.Query().Get("all_tenants") != "true" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, nativeIdentityFindListBody(f.native, identityModesRow(f, "lookup", false), ""))
			})
			ctx := context.WithValue(context.Background(), identityModesContextKey{}, "caller")
			value, err := f.new(client)(ctx, "lookup", resource.WithIdentityFindDetails(false), resource.WithIdentityFindAllProjects(true), resource.WithIdentityFindQuery("vendor", "kept"))
			if err != nil || calls.Load() != 2 {
				t.Fatal(value, err, calls.Load())
			}
			identityModesCheckFields(t, f, value, false)
			if client.Endpoint != endpoint || client.ResourceBase != base || !reflect.DeepEqual(client.MoreHeaders, map[string]string{"X-Configured": "source"}) {
				t.Fatal("lookup mutated source client", client)
			}
		})
	}
}
