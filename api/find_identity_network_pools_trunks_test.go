package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/subnetpools"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/trunks"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Python find_subnet_pool/find_trunk accept string identity, ignore_missing,
// and query. These fixtures retain Gophercloud v2.15.0's native models and
// pagination contracts: SubnetPool requires all three prefix lengths, while
// TrunkPage uses top-level links.next rather than trunks_links.
func networkPoolsTrunksFixtures() []networkExtensionFixture {
	poolView := func(v *subnetpools.SubnetPool) *networkExtensionView {
		nested := v.DefaultPrefixLen == 24 && v.MinPrefixLen == 16 && v.MaxPrefixLen == 28 && v.IPversion == 4 && v.IsDefault && len(v.Prefixes) == 1 && v.Prefixes[0] == "10.0.0.0/16" && !v.CreatedAt.IsZero() && v.RevisionNumber == 9
		return &networkExtensionView{v.ID, v.Name, v.ProjectID, nested}
	}
	trunkView := func(v *trunks.Trunk) *networkExtensionView {
		nested := v.Status == "ACTIVE" && v.PortID == "parent-port" && len(v.Subports) == 1 && v.Subports[0].PortID == "child-port" && v.Subports[0].SegmentationType == "vlan" && v.Subports[0].SegmentationID == 101 && !v.CreatedAt.IsZero() && !v.UpdatedAt.IsZero() && v.RevisionNumber == 9
		return &networkExtensionView{v.ID, v.Name, v.ProjectID, nested}
	}
	return []networkExtensionFixture{
		{"subnet-pool-leaf", "subnetpool", "subnetpools", func(_ *testing.T, c *gophercloud.ServiceClient) networkExtensionAccess {
			a := subnetpools.New(c)
			return networkExtensionAccessFor(a.FindIdentity, a.Resources, c, poolView)
		}},
		{"subnet-pool-connection", "subnetpool", "subnetpools", func(t *testing.T, c *gophercloud.ServiceClient) networkExtensionAccess {
			s := networkExtensionConnection(t, c)
			if s.API.SubnetPools.RawClient() != s.RawClient() {
				t.Fatal("SubnetPool did not share cached network client")
			}
			return networkExtensionAccessFor(s.API.SubnetPools.FindIdentity, s.API.SubnetPools.Resources, s.RawClient(), poolView)
		}},
		{"trunk-leaf", "trunk", "trunks", func(_ *testing.T, c *gophercloud.ServiceClient) networkExtensionAccess {
			a := trunks.New(c)
			return networkExtensionAccessFor(a.FindIdentity, a.Resources, c, trunkView)
		}},
		{"trunk-connection", "trunk", "trunks", func(t *testing.T, c *gophercloud.ServiceClient) networkExtensionAccess {
			s := networkExtensionConnection(t, c)
			if s.API.Trunks.RawClient() != s.RawClient() {
				t.Fatal("Trunk did not share cached network client")
			}
			return networkExtensionAccessFor(s.API.Trunks.FindIdentity, s.API.Trunks.Resources, s.RawClient(), trunkView)
		}},
	}
}

func networkPoolsTrunksPath(f networkExtensionFixture) string {
	return networkExtensionPrefix + f.plural
}
func networkPoolsTrunksClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	return networkExtensionClient(cloud)
}

func networkPoolsTrunksRow(f networkExtensionFixture, id, name string) string {
	return networkPoolsTrunksProjectRow(f, id, name, "project")
}
func networkPoolsTrunksProjectRow(f networkExtensionFixture, id, name, project string) string {
	base := fmt.Sprintf(`"id":%q,"name":%q,"project_id":%q,"created_at":"2025-01-02T03:04:05Z","updated_at":"2025-01-03T03:04:05Z","revision_number":9`, id, name, project)
	if f.plural == "subnetpools" {
		return `{` + base + `,"default_prefixlen":"24","min_prefixlen":16,"max_prefixlen":"28","prefixes":["10.0.0.0/16"],"is_default":true,"ip_version":4}`
	}
	return `{` + base + `,"status":"ACTIVE","port_id":"parent-port","sub_ports":[{"port_id":"child-port","segmentation_type":"vlan","segmentation_id":101}]}`
}
func networkPoolsTrunksInvalidRow(f networkExtensionFixture) string {
	return strings.Replace(networkPoolsTrunksRow(f, "other", "Different"), `"revision_number":9`, `"revision_number":false`, 1)
}
func networkPoolsTrunksChangeRow(t *testing.T, row, key, bad string) string {
	t.Helper()
	var body map[string]json.RawMessage
	if err := json.Unmarshal([]byte(row), &body); err != nil {
		t.Fatal(err)
	}
	switch bad {
	case "missing":
		delete(body, key)
	case "null":
		body[key] = json.RawMessage(`null`)
	case "bool":
		body[key] = json.RawMessage(`false`)
	}
	result, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return string(result)
}
func networkPoolsTrunksPage(f networkExtensionFixture, rows, next string) string {
	links := ""
	if next != "" {
		if f.plural == "trunks" {
			links = fmt.Sprintf(`,"links":{"next":%q}`, next)
		} else {
			links = fmt.Sprintf(`,"subnetpools_links":[{"rel":"next","href":%q}]`, next)
		}
	}
	return fmt.Sprintf(`{%q:[%s]%s}`, f.plural, rows, links)
}

func TestNativeFindIdentityNetworkPoolsTrunksGETModelsAndNativeCodes(t *testing.T) {
	for _, f := range networkPoolsTrunksFixtures() {
		modes := []string{"200", "no-query", "203", "204", "decode", "missing-id", "timestamp"}
		if f.plural == "subnetpools" {
			modes = append(modes, "prefix-strings", "prefix-numbers")
			for _, key := range []string{"default_prefixlen", "min_prefixlen", "max_prefixlen"} {
				for _, bad := range []string{"missing", "null", "bool"} {
					modes = append(modes, key+"/"+bad)
				}
			}
		}
		for _, mode := range modes {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets, lists atomic.Int32
				base := networkPoolsTrunksPath(f)
				cloud.Mux.HandleFunc("GET "+base+"/lookup", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					if mode == "no-query" && r.URL.RawQuery != "" || mode != "no-query" && !reflect.DeepEqual(r.URL.Query()["fields"], []string{"id", "name"}) {
						t.Error("native query GET lost repeated fields", r.URL)
					}
					status, row := 200, networkPoolsTrunksRow(f, "canonical", "Different")
					switch mode {
					case "203":
						status = 203
					case "204":
						w.WriteHeader(204)
						return
					case "decode":
						row = networkPoolsTrunksInvalidRow(f)
					case "missing-id":
						row = networkPoolsTrunksChangeRow(t, row, "id", "missing")
					case "prefix-strings":
						row = strings.Replace(row, `"min_prefixlen":16`, `"min_prefixlen":"16"`, 1)
					case "prefix-numbers":
						row = strings.ReplaceAll(strings.ReplaceAll(row, `"default_prefixlen":"24"`, `"default_prefixlen":24`), `"max_prefixlen":"28"`, `"max_prefixlen":28`)
					case "timestamp":
						// SubnetPool's native custom decoder handles old Neutron
						// no-Z time; Trunk's native time.Time decoder rejects it.
						row = strings.ReplaceAll(row, "03:04:05Z", "03:04:05")
					}
					if parts := strings.Split(mode, "/"); len(parts) == 2 {
						row = networkPoolsTrunksChangeRow(t, row, parts[0], parts[1])
					}
					testcloud.JSON(w, status, fmt.Sprintf(`{%q:%s}`, f.singular, row))
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { lists.Add(1); w.WriteHeader(500) })
				access := f.new(t, networkPoolsTrunksClient(cloud))
				var opts []resource.IdentityFindOption
				if mode != "no-query" {
					opts = []resource.IdentityFindOption{resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: url.Values{"fields": {"id", "name"}}})}
				}
				value, err := access.find(context.Background(), "lookup", opts...)
				if gets.Load() != 1 || lists.Load() != 0 {
					t.Fatal("native GET failure or success restarted lookup", value, err, gets.Load(), lists.Load())
				}
				if mode == "200" || mode == "no-query" || mode == "prefix-strings" || mode == "prefix-numbers" || mode == "timestamp" && f.plural == "subnetpools" {
					if err != nil || value == nil || value.id != "canonical" || value.name != "Different" || value.project != "project" || !value.nested {
						t.Fatal("native nested model extraction changed", value, err)
					}
				} else if value != nil || err == nil || mode == "203" && !gophercloud.ResponseCodeIs(err, 203) || mode == "204" && !gophercloud.ResponseCodeIs(err, 204) || mode == "missing-id" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(value, err)
				}
			})
		}
	}
}

func TestNativeFindIdentityNetworkPoolsTrunksFallbackQueryAndMissingPolicies(t *testing.T) {
	for _, f := range networkPoolsTrunksFixtures() {
		for _, code := range []int{400, 403, 404} {
			for _, policy := range []resource.FindFallbackPolicy{resource.FindFallbackCompatible, resource.FindFallbackNotFoundOnly, resource.FindFallbackNever} {
				t.Run(fmt.Sprintf("%s/%d/policy-%d", f.name, code, policy), func(t *testing.T) {
					cloud := testcloud.New(t)
					var gets, lists atomic.Int32
					base := networkPoolsTrunksPath(f)
					query := url.Values{"name": {"caller-pattern"}, "project_id": {"project"}, "domain_id": {"vendor-domain"}, "fields": {"id", "name"}, "tags": {"one", "two"}, "shared": {"false"}, "status": {"vendor-status"}, "vendor": {"first", "second"}}
					cloud.Mux.HandleFunc("GET "+base+"/lookup", func(w http.ResponseWriter, r *http.Request) {
						gets.Add(1)
						if r.URL.RawQuery != query.Encode() {
							t.Error("caller filters changed on GET", r.URL)
						}
						w.WriteHeader(code)
					})
					cloud.Mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) {
						lists.Add(1)
						if r.URL.RawQuery != query.Encode() {
							t.Error("caller name, repeated or unknown raw filters changed on LIST", r.URL)
						}
						// Match only ID, preserving the caller's server-side name filter.
						testcloud.JSON(w, 300, networkPoolsTrunksPage(f, networkPoolsTrunksRow(f, "lookup", "Different"), ""))
					})
					value, err := f.new(t, networkPoolsTrunksClient(cloud)).find(context.Background(), "lookup", resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: query, Fallback: policy}))
					fallback := policy == resource.FindFallbackCompatible || policy == resource.FindFallbackNotFoundOnly && code == 404
					if gets.Load() != 1 {
						t.Fatal(gets.Load())
					}
					if fallback {
						if err != nil || value == nil || value.id != "lookup" || value.name != "Different" || lists.Load() != 1 {
							t.Fatal(value, err, lists.Load())
						}
					} else if value != nil || lists.Load() != 0 || code == 404 && err != nil || code != 404 && !gophercloud.ResponseCodeIs(err, code) {
						t.Fatal("fallback policy widened", value, err, lists.Load())
					}
				})
			}
		}
		for _, strict := range []bool{false, true} {
			for _, getOnly := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/missing/strict-%t/get-only-%t", f.name, strict, getOnly), func(t *testing.T) {
					cloud := testcloud.New(t)
					base := networkPoolsTrunksPath(f)
					var calls atomic.Int32
					cloud.Mux.HandleFunc("GET "+base+"/lookup", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(404) })
					cloud.Mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(204) })
					opts := []resource.IdentityFindOption{resource.WithIdentityFindIgnoreMissing(!strict)}
					wantCalls := int32(2)
					if getOnly {
						opts = append(opts, resource.WithIdentityFindFallback(resource.FindFallbackNever))
						wantCalls = 1
					}
					value, err := f.new(t, networkPoolsTrunksClient(cloud)).find(context.Background(), "lookup", opts...)
					if value != nil || calls.Load() != wantCalls || strict && !errors.Is(err, resource.ErrNotFound) || !strict && err != nil {
						t.Fatal(value, err, calls.Load())
					}
				})
			}
		}
	}
}

func TestNativeFindIdentityNetworkPoolsTrunksAllPagesAndTerminalObservations(t *testing.T) {
	for _, f := range networkPoolsTrunksFixtures() {
		for _, mode := range []string{"unique", "same-id-duplicate", "cross-project-duplicate", "late-http", "whole-page-decode", "late-decode", "cycle", "list-403", "list-404", "native-link-shape"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				base := networkPoolsTrunksPath(f)
				var gets, lists atomic.Int32
				cloud.Mux.HandleFunc("GET "+base+"/lookup", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); w.WriteHeader(404) })
				cloud.Mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) {
					page := lists.Add(1)
					if r.URL.Query().Get("name") != "lookup" || r.URL.Query().Has("project_id") || r.URL.Query().Has("domain_id") {
						t.Error("lookup inferred scope or changed the exact hint", r.URL)
					}
					if mode == "list-403" || mode == "list-404" {
						code := 403
						if mode == "list-404" {
							code = 404
						}
						w.WriteHeader(code)
						return
					}
					rows := networkPoolsTrunksProjectRow(f, "first", "lookup", "project-a")
					next := cloud.Server.URL + base + "?marker=second&name=lookup"
					if page > 1 {
						rows, next = networkPoolsTrunksRow(f, "other", "Different"), ""
					}
					switch mode {
					case "same-id-duplicate":
						rows = networkPoolsTrunksRow(f, "first", "lookup")
					case "cross-project-duplicate":
						if page > 1 {
							rows = networkPoolsTrunksProjectRow(f, "second", "lookup", "project-b")
						}
					case "late-http":
						if page > 1 {
							w.WriteHeader(503)
							return
						}
					case "whole-page-decode":
						rows += "," + networkPoolsTrunksInvalidRow(f)
					case "late-decode":
						if page > 1 {
							rows = networkPoolsTrunksInvalidRow(f)
						}
					case "cycle":
						if page > 1 {
							next = cloud.Server.URL + base + "?marker=second&name=lookup"
						}
					}
					if mode == "native-link-shape" {
						// Native TrunkPage inherits links.next; trunks_links is ignored.
						// Native SubnetPoolPage overrides it with subnetpools_links.
						wrong := `,"links":{"next":` + strconv.Quote(next) + `}`
						if f.plural == "trunks" {
							wrong = `,"trunks_links":[{"rel":"next","href":` + strconv.Quote(next) + `}]`
						}
						testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[%s]%s}`, f.plural, rows, wrong))
						return
					}
					testcloud.JSON(w, 200, networkPoolsTrunksPage(f, rows, next))
				})
				value, err := f.new(t, networkPoolsTrunksClient(cloud)).find(context.Background(), "lookup")
				if gets.Load() != 1 {
					t.Fatal("lookup GET repeated", gets.Load())
				}
				if mode == "native-link-shape" {
					if err != nil || value == nil || value.id != "first" || lists.Load() != 1 {
						t.Fatal(value, err, lists.Load())
					}
					return
				}
				if mode == "unique" {
					if err != nil || value == nil || value.id != "first" || lists.Load() != 2 {
						t.Fatal(value, err, lists.Load())
					}
					return
				}
				if value != nil || err == nil || (mode == "same-id-duplicate" || mode == "cross-project-duplicate") && !errors.Is(err, resource.ErrAmbiguous) || mode == "late-http" && !gophercloud.ResponseCodeIs(err, 503) || mode == "list-403" && !gophercloud.ResponseCodeIs(err, 403) || mode == "list-404" && !gophercloud.ResponseCodeIs(err, 404) {
					t.Fatal("partial match or list error was hidden", value, err, lists.Load())
				}
			})
		}
	}
}

func TestNativeFindIdentityNetworkPoolsTrunksQuerySnapshotsAndConcurrentReuse(t *testing.T) {
	for _, f := range networkPoolsTrunksFixtures() {
		for _, name := range [][]string{nil, {}, {""}} {
			t.Run(fmt.Sprintf("%s/name-%v", f.name, name), func(t *testing.T) {
				cloud := testcloud.New(t)
				base := networkPoolsTrunksPath(f)
				query := url.Values{"name": name, "fields": {"id", "name"}, "tags": {"one", "two"}, "project_id": {"project"}, "status": {"vendor-status"}, "vendor": {"first", "second"}}
				option := resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: query})
				want := query.Encode()
				query["fields"][0], query["name"] = "mutated", []string{"mutated"}
				var gets, lists atomic.Int32
				cloud.Mux.HandleFunc("GET "+base+"/lookup", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					if r.URL.RawQuery != want {
						t.Error("GET did not use the owned query", r.URL, want)
					}
					w.WriteHeader(404)
				})
				cloud.Mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					if r.URL.RawQuery != want {
						t.Error("nil/empty caller name was replaced or repeated query collapsed", r.URL, want)
					}
					testcloud.JSON(w, 200, networkPoolsTrunksPage(f, networkPoolsTrunksRow(f, "lookup", "Different"), ""))
				})
				access := f.new(t, networkPoolsTrunksClient(cloud))
				var group sync.WaitGroup
				for range 6 {
					group.Add(1)
					go func() {
						defer group.Done()
						value, err := access.find(context.Background(), "lookup", option)
						if err != nil || value == nil || value.id != "lookup" || value.name != "Different" {
							t.Error(value, err)
						}
					}()
				}
				group.Wait()
				if gets.Load() != 6 || lists.Load() != 6 {
					t.Fatal(gets.Load(), lists.Load())
				}
			})
		}
	}
}

func TestNativeFindIdentityNetworkPoolsTrunksUnsafeNamesAndCapabilityPreflight(t *testing.T) {
	for _, f := range networkPoolsTrunksFixtures() {
		t.Run(f.name+"/preflight", func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			access := f.new(t, networkPoolsTrunksClient(cloud))
			canceled, cancel := context.WithCancel(context.Background())
			cancel()
			for _, flag := range []bool{false, true} {
				for _, option := range []resource.IdentityFindOption{resource.WithIdentityFindDetails(flag), resource.WithIdentityFindAllProjects(flag), resource.WithIdentityFindExtraSpecs(flag)} {
					if value, err := access.find(context.Background(), "lookup", option); value != nil || !errors.Is(err, resource.ErrUnsupported) {
						t.Fatal(value, err)
					}
				}
			}
			if value, err := access.find(canceled, "lookup"); value != nil || !errors.Is(err, context.Canceled) {
				t.Fatal(value, err)
			}
			if value, err := access.find(nil, "lookup"); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(value, err)
			}
			if value, err := access.find(context.Background(), "unsafe/name", resource.WithIdentityFindFallback(resource.FindFallbackNever)); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(value, err)
			}
			if f.plural == "subnetpools" {
				if _, err := access.all(context.Background(), resource.WithStatus("ACTIVE")); !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal("SubnetPool fabricated a status selector", err)
				}
			}
			if calls.Load() != 0 {
				t.Fatal("preflight performed HTTP", calls.Load())
			}
		})
		for _, identity := range []string{"unsafe/name", "literal%2Fname", " name with spaces "} {
			t.Run(f.name+"/unsafe/"+identity, func(t *testing.T) {
				cloud := testcloud.New(t)
				var lists, gets atomic.Int32
				base := networkPoolsTrunksPath(f)
				cloud.Mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					if r.URL.Query().Get("name") != identity {
						t.Error("unsafe identity was trimmed or unescaped", r.URL)
					}
					testcloud.JSON(w, 200, networkPoolsTrunksPage(f, networkPoolsTrunksRow(f, "found", identity), ""))
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); w.WriteHeader(500) })
				value, err := f.new(t, networkPoolsTrunksClient(cloud)).find(context.Background(), identity)
				if err != nil || value == nil || value.name != identity || gets.Load() != 0 || lists.Load() != 1 {
					t.Fatal(value, err, gets.Load(), lists.Load())
				}
			})
		}
	}
}

func TestNativeFindIdentityNetworkPoolsTrunksLiveSourceAndCancellation(t *testing.T) {
	for _, f := range networkPoolsTrunksFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := networkPoolsTrunksClient(cloud)
			client.MoreHeaders = map[string]string{"X-Configured": "source"}
			client.Microversion = "2.0"
			access := f.new(t, client)
			endpoint, base := access.raw.Endpoint, access.raw.ResourceBase
			transport := cloud.Provider.HTTPClient.Transport
			if transport == nil {
				transport = http.DefaultTransport
			}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = nativeIdentityFindTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Context().Value(networkExtensionContextKey{}) != "caller" {
					t.Error("caller context lost")
				}
				copy := r.Clone(r.Context())
				copy.Header.Set("X-Network-Middleware", "source")
				return transport.RoundTrip(copy)
			})
			check := func(r *http.Request, token string) {
				if r.Header.Get("X-Auth-Token") != token || r.Header.Get("X-Configured") != "source" || r.Header.Get("X-Network-Middleware") != "source" || r.Header.Get("OpenStack-API-Version") != "network 2.0" || r.URL.Query().Get("vendor") != "both-phases" {
					t.Error(r.URL, r.Header)
				}
			}
			path := networkPoolsTrunksPath(f)
			cloud.Mux.HandleFunc("GET "+path+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				check(r, "test-token")
				cloud.Provider.SetToken("changed-token")
				w.WriteHeader(404)
			})
			cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
				check(r, "changed-token")
				testcloud.JSON(w, 200, networkPoolsTrunksPage(f, networkPoolsTrunksRow(f, "found", "lookup"), ""))
			})
			ctx := context.WithValue(context.Background(), networkExtensionContextKey{}, "caller")
			value, err := access.find(ctx, "lookup", resource.WithIdentityFindQuery("vendor", "both-phases"))
			if err != nil || value == nil || value.id != "found" || calls.Load() != 2 || access.raw.Endpoint != endpoint || access.raw.ResourceBase != base || access.raw.ProviderClient != cloud.Provider || access.raw.Microversion != "2.0" || !reflect.DeepEqual(access.raw.MoreHeaders, map[string]string{"X-Configured": "source"}) {
				t.Fatal("lookup changed the shared source configuration", value, err, calls.Load(), access.raw)
			}
		})
		t.Run(f.name+"/cancel-after-match", func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var lists atomic.Int32
			path := networkPoolsTrunksPath(f)
			cloud.Mux.HandleFunc("GET "+path+"/lookup", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) })
			cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				cancel()
				testcloud.JSON(w, 200, networkPoolsTrunksPage(f, networkPoolsTrunksRow(f, "found", "lookup"), cloud.Server.URL+path+"?marker=second"))
			})
			value, err := f.new(t, networkPoolsTrunksClient(cloud)).find(ctx, "lookup")
			if value != nil || !errors.Is(err, context.Canceled) || lists.Load() != 1 {
				t.Fatal("cancellation returned a partial match or followed a page", value, err, lists.Load())
			}
		})
	}
}

func TestNativeFindIdentityNetworkPoolsTrunksOrdinaryCollectionsQueriesAndControls(t *testing.T) {
	for _, f := range networkPoolsTrunksFixtures() {
		for _, mode := range []string{"all", "raw-cap-before-name", "single-page", "explicit-id", "explicit-name"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				base := networkPoolsTrunksPath(f)
				var lists, gets atomic.Int32
				cloud.Mux.HandleFunc("GET "+base+"/direct", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					if r.URL.RawQuery != "" {
						t.Error("ordinary ID Find gained query defaults", r.URL)
					}
					testcloud.JSON(w, 200, fmt.Sprintf(`{%q:%s}`, f.singular, networkPoolsTrunksRow(f, "direct", "Different")))
				})
				cloud.Mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) {
					page := lists.Add(1)
					if mode != "explicit-name" {
						for key, want := range map[string]string{"status": "vendor-status", "fields": "id,name", "shared": "false", "vendor": "value"} {
							if r.URL.Query().Get(key) != want {
								t.Error("ordinary raw query lost through adapter", r.URL, key)
							}
						}
						if r.URL.Query().Has("limit") || r.URL.Query().Has("max_items") || r.URL.Query().Has("paginated") {
							t.Error("local controls leaked or invented a wire limit", r.URL)
						}
					} else if r.URL.Query().Get("name") != "Target" {
						t.Error("explicit name lookup lost its literal hint", r.URL)
					}
					rows := networkPoolsTrunksRow(f, "first", "Other") + "," + networkPoolsTrunksRow(f, "target", "Target")
					nextQuery := r.URL.Query()
					nextQuery.Set("marker", "next")
					next := cloud.Server.URL + base + "?" + nextQuery.Encode()
					if page > 1 {
						rows, next = networkPoolsTrunksRow(f, "last", "Last"), ""
					}
					testcloud.JSON(w, 200, networkPoolsTrunksPage(f, rows, next))
				})
				access := f.new(t, networkPoolsTrunksClient(cloud))
				if mode == "explicit-id" || mode == "explicit-name" {
					ref := resource.ID("direct")
					if mode == "explicit-name" {
						ref = resource.Name("Target")
					}
					value, err := access.ref(context.Background(), ref)
					if err != nil || value == nil || mode == "explicit-id" && (value.id != "direct" || gets.Load() != 1 || lists.Load() != 0) || mode == "explicit-name" && (value.id != "target" || gets.Load() != 0 || lists.Load() != 2) {
						t.Fatal("explicit Ref semantics changed", value, err, gets.Load(), lists.Load())
					}
					return
				}
				opts := []resource.ListOption{resource.WithQuery("status", "vendor-status"), resource.WithQuery("fields", "id,name"), resource.WithQuery("shared", "false"), resource.WithQuery("vendor", "value")}
				if mode == "raw-cap-before-name" {
					opts = append(opts, resource.WithMaxItems(1), resource.WithName("Target"))
				} else if mode == "single-page" {
					opts = append(opts, resource.WithPaginated(false))
				}
				values, err := access.all(context.Background(), opts...)
				wantRows, wantPages := 3, int32(2)
				if mode == "raw-cap-before-name" {
					wantRows, wantPages = 0, 1
				} else if mode == "single-page" {
					wantRows, wantPages = 2, 1
				}
				if err != nil || len(values) != wantRows || lists.Load() != wantPages || gets.Load() != 0 {
					t.Fatal("ordinary raw query or controls changed", values, err, lists.Load(), gets.Load())
				}
			})
		}
	}
}
