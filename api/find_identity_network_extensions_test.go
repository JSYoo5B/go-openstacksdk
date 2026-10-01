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

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/network"
	"gophercloudsdk/network/v2/extensions/layer3/routers"
	"gophercloudsdk/network/v2/extensions/security/groups"
	"gophercloudsdk/resource"
)

// Python find_router/find_security_group accept query and ignore logical
// absence by default. Both native getters accept 200; both native page types
// use plural_links. The SDK security-group collection's raw-query pager keeps
// native SecGroupPage decoding despite upstream List's concrete ListOpts.
type networkExtensionView struct {
	id, name, project string
	nested            bool
}

type networkExtensionAccess struct {
	find func(context.Context, string, ...resource.IdentityFindOption) (*networkExtensionView, error)
	all  func(context.Context, ...resource.ListOption) ([]*networkExtensionView, error)
	ref  func(context.Context, resource.Ref) (*networkExtensionView, error)
	raw  *gophercloud.ServiceClient
}

func networkExtensionAccessFor[T any](find func(context.Context, string, ...resource.IdentityFindOption) (*T, error), collection *resource.Collection[T], raw *gophercloud.ServiceClient, view func(*T) *networkExtensionView) networkExtensionAccess {
	return networkExtensionAccess{
		find: func(ctx context.Context, identity string, opts ...resource.IdentityFindOption) (*networkExtensionView, error) {
			value, err := find(ctx, identity, opts...)
			if value == nil {
				return nil, err
			}
			return view(value), err
		},
		all: func(ctx context.Context, opts ...resource.ListOption) ([]*networkExtensionView, error) {
			values, err := collection.All(ctx, opts...)
			result := make([]*networkExtensionView, 0, len(values))
			for _, value := range values {
				result = append(result, view(value))
			}
			return result, err
		},
		ref: func(ctx context.Context, ref resource.Ref) (*networkExtensionView, error) {
			value, err := collection.Find(ctx, ref)
			if value == nil {
				return nil, err
			}
			return view(value), err
		},
		raw: raw,
	}
}

type networkExtensionFixture struct {
	name, singular, plural string
	new                    func(*testing.T, *gophercloud.ServiceClient) networkExtensionAccess
}

func networkExtensionConnection(t *testing.T, c *gophercloud.ServiceClient) *network.Service {
	t.Helper()
	conn, err := sdk.FromProvider(c.ProviderClient, sdk.WithEndpoint(sdk.Network, c.Endpoint))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.Network(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	raw := service.RawClient()
	raw.ResourceBase, raw.Microversion, raw.MoreHeaders = c.ResourceBase, c.Microversion, c.MoreHeaders
	again, err := conn.Network(context.Background())
	if err != nil || again != service || raw.ProviderClient != c.ProviderClient || service.API.Routers.RawClient() != raw || service.API.SecurityGroups.RawClient() != raw {
		t.Fatal("Connection did not share its cached network client", service, again, err)
	}
	return service
}

func networkExtensionFixtures() []networkExtensionFixture {
	routerView := func(v *routers.Router) *networkExtensionView {
		nested := v.Status == "ACTIVE" && v.GatewayInfo.NetworkID == "network" && v.GatewayInfo.EnableSNAT != nil && *v.GatewayInfo.EnableSNAT && len(v.GatewayInfo.ExternalFixedIPs) == 1 && v.GatewayInfo.ExternalFixedIPs[0].SubnetID == "subnet" && len(v.Routes) == 1 && v.Routes[0].NextHop == "192.0.2.2" && !v.CreatedAt.IsZero() && v.RevisionNumber == 9
		return &networkExtensionView{v.ID, v.Name, v.ProjectID, nested}
	}
	groupView := func(v *groups.SecGroup) *networkExtensionView {
		nested := !v.Stateful && len(v.Rules) == 1 && v.Rules[0].Protocol == "tcp" && v.Rules[0].PortRangeMin == 22 && v.Rules[0].RemoteAddressGroupID == "remote-address" && !v.CreatedAt.IsZero() && v.RevisionNumber == 9
		return &networkExtensionView{v.ID, v.Name, v.ProjectID, nested}
	}
	return []networkExtensionFixture{
		{"router-leaf", "router", "routers", func(_ *testing.T, c *gophercloud.ServiceClient) networkExtensionAccess {
			a := routers.New(c)
			return networkExtensionAccessFor(a.FindIdentity, a.Resources, c, routerView)
		}},
		{"router-connection", "router", "routers", func(t *testing.T, c *gophercloud.ServiceClient) networkExtensionAccess {
			s := networkExtensionConnection(t, c)
			return networkExtensionAccessFor(s.API.Routers.FindIdentity, s.API.Routers.Resources, s.RawClient(), routerView)
		}},
		{"security-group-leaf", "security_group", "security_groups", func(_ *testing.T, c *gophercloud.ServiceClient) networkExtensionAccess {
			a := groups.New(c)
			return networkExtensionAccessFor(a.FindIdentity, a.Resources, c, groupView)
		}},
		{"security-group-connection", "security_group", "security_groups", func(t *testing.T, c *gophercloud.ServiceClient) networkExtensionAccess {
			s := networkExtensionConnection(t, c)
			return networkExtensionAccessFor(s.API.SecurityGroups.FindIdentity, s.API.SecurityGroups.Resources, s.RawClient(), groupView)
		}},
	}
}

const networkExtensionPrefix = "/reverse/neutron/v2.0/"

func networkExtensionPath(f networkExtensionFixture) string {
	if f.plural == "security_groups" {
		return networkExtensionPrefix + "security-groups"
	}
	return networkExtensionPrefix + "routers"
}

func networkExtensionClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	c := cloud.Client("network", "/unused/catalog")
	c.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + networkExtensionPrefix)
	return c
}

func networkExtensionFullRow(f networkExtensionFixture, id, name string) string {
	base := fmt.Sprintf(`"id":%q,"name":%q,"project_id":"project","created_at":"2025-01-02T03:04:05Z","revision_number":9`, id, name)
	if f.plural == "routers" {
		return `{` + base + `,"status":"ACTIVE","external_gateway_info":{"network_id":"network","enable_snat":true,"external_fixed_ips":[{"subnet_id":"subnet","ip_address":"192.0.2.1"}]},"routes":[{"destination":"10.0.0.0/24","nexthop":"192.0.2.2"}]}`
	}
	return `{` + base + `,"stateful":false,"security_group_rules":[{"id":"rule","protocol":"tcp","direction":"ingress","ethertype":"IPv4","port_range_min":22,"port_range_max":22,"remote_address_group_id":"remote-address"}]}`
}

func networkExtensionPage(f networkExtensionFixture, rows, next string) string {
	links := ""
	if next != "" {
		links = fmt.Sprintf(`,%q:[{"rel":"next","href":%q}]`, f.plural+"_links", next)
	}
	return fmt.Sprintf(`{%q:[%s]%s}`, f.plural, rows, links)
}

func TestNativeFindIdentityNetworkExtensionsGETModelsAndNativeCodes(t *testing.T) {
	for _, f := range networkExtensionFixtures() {
		for _, mode := range []string{"200", "203", "204", "decode", "missing-id"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets, lists atomic.Int32
				base := networkExtensionPath(f)
				cloud.Mux.HandleFunc("GET "+base+"/lookup", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					if !reflect.DeepEqual(r.URL.Query()["fields"], []string{"id", "name"}) {
						t.Error("native query GET lost repeated fields", r.URL)
					}
					status, row := 200, networkExtensionFullRow(f, "canonical", "Different")
					switch mode {
					case "203":
						status = 203
					case "204":
						w.WriteHeader(204)
						return
					case "decode":
						row = `{"id":"canonical","revision_number":false}`
					case "missing-id":
						row = `{"name":"lookup"}`
					}
					testcloud.JSON(w, status, fmt.Sprintf(`{%q:%s}`, f.singular, row))
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { lists.Add(1); w.WriteHeader(500) })
				access := f.new(t, networkExtensionClient(cloud))
				value, err := access.find(context.Background(), "lookup", resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: url.Values{"fields": {"id", "name"}}}))
				if gets.Load() != 1 || lists.Load() != 0 {
					t.Fatal("native GET failure or success restarted lookup", value, err, gets.Load(), lists.Load())
				}
				if mode == "200" {
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

func TestNativeFindIdentityNetworkExtensionsFallbackQueryAndMissingPolicies(t *testing.T) {
	for _, f := range networkExtensionFixtures() {
		for _, code := range []int{400, 403, 404} {
			for _, policy := range []resource.FindFallbackPolicy{resource.FindFallbackCompatible, resource.FindFallbackNotFoundOnly, resource.FindFallbackNever} {
				t.Run(fmt.Sprintf("%s/%d/policy-%d", f.name, code, policy), func(t *testing.T) {
					cloud := testcloud.New(t)
					var gets, lists atomic.Int32
					base := networkExtensionPath(f)
					query := url.Values{"name": {"caller-pattern"}, "project_id": {"project"}, "domain_id": {"vendor-domain"}, "fields": {"id", "name"}, "tags": {"one", "two"}, "shared": {"false"}, "vendor": {"first", "second"}}
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
						testcloud.JSON(w, 300, networkExtensionPage(f, nativeIdentityFindRow("lookup", "Different"), ""))
					})
					value, err := f.new(t, networkExtensionClient(cloud)).find(context.Background(), "lookup", resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: query, Fallback: policy}))
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
			t.Run(fmt.Sprintf("%s/missing/strict-%t", f.name, strict), func(t *testing.T) {
				cloud := testcloud.New(t)
				base := networkExtensionPath(f)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+base+"/lookup", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(404) })
				cloud.Mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(204) })
				value, err := f.new(t, networkExtensionClient(cloud)).find(context.Background(), "lookup", resource.WithIdentityFindIgnoreMissing(!strict))
				if value != nil || calls.Load() != 2 || strict && !errors.Is(err, resource.ErrNotFound) || !strict && err != nil {
					t.Fatal(value, err, calls.Load())
				}
			})
		}
	}
}

func TestNativeFindIdentityNetworkExtensionsAllPagesAndTerminalObservations(t *testing.T) {
	for _, f := range networkExtensionFixtures() {
		for _, mode := range []string{"unique", "same-id-duplicate", "cross-project-duplicate", "late-http", "whole-page-decode", "late-decode", "cycle", "list-403", "list-404"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				base := networkExtensionPath(f)
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
					rows := `{"id":"first","name":"lookup","project_id":"project-a"}`
					next := cloud.Server.URL + base + "?marker=second&name=lookup"
					if page > 1 {
						rows, next = `{"id":"other","name":"Different"}`, ""
					}
					switch mode {
					case "same-id-duplicate":
						rows = `{"id":"first","name":"lookup"}`
					case "cross-project-duplicate":
						if page > 1 {
							rows = `{"id":"second","name":"lookup","project_id":"project-b"}`
						}
					case "late-http":
						if page > 1 {
							w.WriteHeader(503)
							return
						}
					case "whole-page-decode":
						rows += `,{"id":"other","revision_number":false}`
					case "late-decode":
						if page > 1 {
							rows = `{"id":"other","revision_number":false}`
						}
					case "cycle":
						if page > 1 {
							next = cloud.Server.URL + base + "?marker=second&name=lookup"
						}
					}
					testcloud.JSON(w, 200, networkExtensionPage(f, rows, next))
				})
				value, err := f.new(t, networkExtensionClient(cloud)).find(context.Background(), "lookup")
				if gets.Load() != 1 {
					t.Fatal("lookup GET repeated", gets.Load())
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

func TestNativeFindIdentityNetworkExtensionsQuerySnapshotsAndConcurrentReuse(t *testing.T) {
	for _, f := range networkExtensionFixtures() {
		for _, name := range [][]string{nil, {}, {""}} {
			t.Run(fmt.Sprintf("%s/name-%v", f.name, name), func(t *testing.T) {
				cloud := testcloud.New(t)
				base := networkExtensionPath(f)
				query := url.Values{"name": name, "fields": {"id", "name"}, "tags": {"one", "two"}, "project_id": {"project"}, "vendor": {"first", "second"}}
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
					testcloud.JSON(w, 200, networkExtensionPage(f, nativeIdentityFindRow("lookup", "Different"), ""))
				})
				access := f.new(t, networkExtensionClient(cloud))
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

func TestNativeFindIdentityNetworkExtensionsUnsafeNamesAndCapabilityPreflight(t *testing.T) {
	for _, f := range networkExtensionFixtures() {
		t.Run(f.name+"/preflight", func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			access := f.new(t, networkExtensionClient(cloud))
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
			if calls.Load() != 0 {
				t.Fatal("preflight performed HTTP", calls.Load())
			}
		})
		for _, identity := range []string{"unsafe/name", "literal%2Fname", " name with spaces "} {
			t.Run(f.name+"/unsafe/"+identity, func(t *testing.T) {
				cloud := testcloud.New(t)
				var lists, gets atomic.Int32
				base := networkExtensionPath(f)
				cloud.Mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					if r.URL.Query().Get("name") != identity {
						t.Error("unsafe identity was trimmed or unescaped", r.URL)
					}
					testcloud.JSON(w, 200, networkExtensionPage(f, nativeIdentityFindRow("found", identity), ""))
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); w.WriteHeader(500) })
				value, err := f.new(t, networkExtensionClient(cloud)).find(context.Background(), identity)
				if err != nil || value == nil || value.name != identity || gets.Load() != 0 || lists.Load() != 1 {
					t.Fatal(value, err, gets.Load(), lists.Load())
				}
			})
		}
	}
}

type networkExtensionContextKey struct{}

func TestNativeFindIdentityNetworkExtensionsLiveSourceAndCancellation(t *testing.T) {
	for _, f := range networkExtensionFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := networkExtensionClient(cloud)
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
			path := networkExtensionPath(f)
			cloud.Mux.HandleFunc("GET "+path+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				check(r, "test-token")
				cloud.Provider.SetToken("changed-token")
				w.WriteHeader(404)
			})
			cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
				check(r, "changed-token")
				testcloud.JSON(w, 200, networkExtensionPage(f, nativeIdentityFindRow("found", "lookup"), ""))
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
			path := networkExtensionPath(f)
			cloud.Mux.HandleFunc("GET "+path+"/lookup", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) })
			cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				cancel()
				testcloud.JSON(w, 200, networkExtensionPage(f, nativeIdentityFindRow("found", "lookup"), cloud.Server.URL+path+"?marker=second"))
			})
			value, err := f.new(t, networkExtensionClient(cloud)).find(ctx, "lookup")
			if value != nil || !errors.Is(err, context.Canceled) || lists.Load() != 1 {
				t.Fatal("cancellation returned a partial match or followed a page", value, err, lists.Load())
			}
		})
	}
}

func TestNativeFindIdentityNetworkExtensionsOrdinaryCollectionsQueriesAndControls(t *testing.T) {
	for _, f := range networkExtensionFixtures() {
		for _, mode := range []string{"all", "raw-cap-before-name", "single-page", "explicit-id", "explicit-name"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				base := networkExtensionPath(f)
				var lists, gets atomic.Int32
				cloud.Mux.HandleFunc("GET "+base+"/direct", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					if r.URL.RawQuery != "" {
						t.Error("ordinary ID Find gained query defaults", r.URL)
					}
					testcloud.JSON(w, 200, fmt.Sprintf(`{%q:%s}`, f.singular, nativeIdentityFindRow("direct", "Different")))
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
					rows := nativeIdentityFindRow("first", "Other") + "," + nativeIdentityFindRow("target", "Target")
					nextQuery := r.URL.Query()
					nextQuery.Set("marker", "next")
					next := cloud.Server.URL + base + "?" + nextQuery.Encode()
					if page > 1 {
						rows, next = nativeIdentityFindRow("last", "Last"), ""
					}
					testcloud.JSON(w, 200, networkExtensionPage(f, rows, next))
				})
				access := f.new(t, networkExtensionClient(cloud))
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
