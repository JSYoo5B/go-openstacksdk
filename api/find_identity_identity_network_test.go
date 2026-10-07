package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/identity/v3/domains"
	"github.com/JSYoo5B/gophercloudsdk/identity/v3/groups"
	"github.com/JSYoo5B/gophercloudsdk/identity/v3/projects"
	"github.com/JSYoo5B/gophercloudsdk/identity/v3/roles"
	"github.com/JSYoo5B/gophercloudsdk/identity/v3/users"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/networks"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/subnets"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Pinned Python find_network/find_subnet and four Keystone find methods accept
// **query; find_domain has only identity/ignore_missing. Explicit Go queries on
// domains are an extension. Resource.find forwards caller params to fetch and
// then list (resource.py:2536-2573; fetch:1824-1828), without a default domain.
// Native getters below accept 200 and preserve their singular envelopes. The
// shared nine HTTP groups additionally run these seven bindings, using Neutron
// plural_links and Keystone's top-level links.next rather than invented paging.
func nativeIdentityNetworkFindFixtures() []nativeIdentityFindFixture {
	return []nativeIdentityFindFixture{
		{"network", "/reverse/neutron/v2.0", "/reverse/neutron/v2.0/networks", "/reverse/neutron/v2.0/networks", "network", "networks", false,
			func(_ *testing.T, client *gophercloud.ServiceClient) nativeIdentityFindAccess {
				return nativeIdentityFindAccessFor(networks.New(client).FindIdentity, func(v *networks.Network) string { return v.ID }, func(v *networks.Network) string { return v.Name })
			}},
		{"network", "/reverse/neutron/v2.0", "/reverse/neutron/v2.0/subnets", "/reverse/neutron/v2.0/subnets", "subnet", "subnets", false,
			func(_ *testing.T, client *gophercloud.ServiceClient) nativeIdentityFindAccess {
				return nativeIdentityFindAccessFor(subnets.New(client).FindIdentity, func(v *subnets.Subnet) string { return v.ID }, func(v *subnets.Subnet) string { return v.Name })
			}},
		{"identity", "/reverse/keystone/v3", "/reverse/keystone/v3/projects", "/reverse/keystone/v3/projects", "project", "projects", false,
			func(_ *testing.T, client *gophercloud.ServiceClient) nativeIdentityFindAccess {
				return nativeIdentityFindAccessFor(projects.New(client).FindIdentity, func(v *projects.Project) string { return v.ID }, func(v *projects.Project) string { return v.Name })
			}},
		{"identity", "/reverse/keystone/v3", "/reverse/keystone/v3/users", "/reverse/keystone/v3/users", "user", "users", false,
			func(_ *testing.T, client *gophercloud.ServiceClient) nativeIdentityFindAccess {
				return nativeIdentityFindAccessFor(users.New(client).FindIdentity, func(v *users.User) string { return v.ID }, func(v *users.User) string { return v.Name })
			}},
		{"identity", "/reverse/keystone/v3", "/reverse/keystone/v3/groups", "/reverse/keystone/v3/groups", "group", "groups", false,
			func(_ *testing.T, client *gophercloud.ServiceClient) nativeIdentityFindAccess {
				return nativeIdentityFindAccessFor(groups.New(client).FindIdentity, func(v *groups.Group) string { return v.ID }, func(v *groups.Group) string { return v.Name })
			}},
		{"identity", "/reverse/keystone/v3", "/reverse/keystone/v3/domains", "/reverse/keystone/v3/domains", "domain", "domains", false,
			func(_ *testing.T, client *gophercloud.ServiceClient) nativeIdentityFindAccess {
				return nativeIdentityFindAccessFor(domains.New(client).FindIdentity, func(v *domains.Domain) string { return v.ID }, func(v *domains.Domain) string { return v.Name })
			}},
		{"identity", "/reverse/keystone/v3", "/reverse/keystone/v3/roles", "/reverse/keystone/v3/roles", "role", "roles", false,
			func(_ *testing.T, client *gophercloud.ServiceClient) nativeIdentityFindAccess {
				return nativeIdentityFindAccessFor(roles.New(client).FindIdentity, func(v *roles.Role) string { return v.ID }, func(v *roles.Role) string { return v.Name })
			}},
	}
}

func TestNativeFindIdentityIdentityAndNetworkGETStatusContracts(t *testing.T) {
	for _, fixture := range nativeIdentityNetworkFindFixtures() {
		for _, status := range []int{200, 203} {
			for _, withQuery := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%d/query-%t", fixture.plural, status, withQuery), func(t *testing.T) {
					cloud := testcloud.New(t)
					var gets, lists atomic.Int32
					cloud.Mux.HandleFunc("GET "+fixture.base+"/lookup", func(w http.ResponseWriter, r *http.Request) {
						gets.Add(1)
						if withQuery && !reflect.DeepEqual(r.URL.Query()["fields"], []string{"id", "name"}) || !withQuery && r.URL.RawQuery != "" {
							t.Error("native GET query changed", r.URL)
						}
						testcloud.JSON(w, status, nativeIdentityFindGetBody(fixture, nativeIdentityFindRow("returned-id", "DifferentName")))
					})
					cloud.Mux.HandleFunc("GET "+fixture.path, func(w http.ResponseWriter, r *http.Request) { lists.Add(1); w.WriteHeader(500) })
					var options []resource.IdentityFindOption
					if withQuery {
						options = append(options, resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: url.Values{"fields": {"id", "name"}}}))
					}
					value, err := fixture.new(t, nativeIdentityFindClient(cloud, fixture))(context.Background(), "lookup", options...)
					if gets.Load() != 1 || lists.Load() != 0 {
						t.Fatal("getter status triggered fallback or resend", gets.Load(), lists.Load(), err)
					}
					if status == 200 {
						if err != nil || value == nil || value.id != "returned-id" || value.name != "DifferentName" {
							t.Fatal(value, err)
						}
					} else if value != nil || !gophercloud.ResponseCodeIs(err, 203) {
						t.Fatal("query GET broadened the native accepted codes", value, err)
					}
				})
			}
		}
	}
}

func TestNativeFindIdentityKeystoneDomainFiltersAndCrossDomainDuplicates(t *testing.T) {
	for _, fixture := range nativeIdentityNetworkFindFixtures() {
		if fixture.service != "identity" || fixture.plural == "domains" {
			continue
		}
		for _, filtered := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/filtered-%t", fixture.plural, filtered), func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets, lists atomic.Int32
				cloud.Mux.HandleFunc("GET "+fixture.base+"/lookup", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					query := r.URL.Query()
					if query.Has("name") || filtered && query.Get("domain_id") != "domain-a" || !filtered && query.Has("domain_id") {
						t.Error("GET gained a default name/domain or dropped the caller domain", r.URL)
					}
					w.WriteHeader(404)
				})
				cloud.Mux.HandleFunc("GET "+fixture.path, func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					query := r.URL.Query()
					if query.Get("name") != "lookup" || filtered && query.Get("domain_id") != "domain-a" || !filtered && query.Has("domain_id") {
						t.Error("List changed name/domain selection", r.URL)
					}
					row := `{"id":"domain-a-id","name":"lookup","domain_id":"domain-a"}`
					next := ""
					if !filtered && query.Get("marker") == "" {
						next = cloud.Server.URL + fixture.path + "?marker=next&name=lookup"
					} else if !filtered {
						row = `{"id":"domain-b-id","name":"lookup","domain_id":"domain-b"}`
					}
					testcloud.JSON(w, 200, nativeIdentityFindListBody(fixture, row, next))
				})
				var options []resource.IdentityFindOption
				if filtered {
					options = append(options, resource.WithIdentityFindQuery("domain_id", "domain-a"))
				}
				value, err := fixture.new(t, nativeIdentityFindClient(cloud, fixture))(context.Background(), "lookup", options...)
				if filtered {
					if err != nil || value == nil || value.id != "domain-a-id" || gets.Load() != 1 || lists.Load() != 1 {
						t.Fatal(value, err, gets.Load(), lists.Load())
					}
				} else if value != nil || !errors.Is(err, resource.ErrAmbiguous) || gets.Load() != 1 || lists.Load() != 2 {
					t.Fatal("same names in different domains were not ambiguous", value, err, gets.Load(), lists.Load())
				}
			})
		}
	}
}

type nativeIdentityFindContextKey struct{}

func TestNativeFindIdentityDualPhaseQueryKeepsLiveProviderAndHTTPClient(t *testing.T) {
	for _, fixture := range nativeIdentityFindFixtures() {
		t.Run(fixture.plural, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := nativeIdentityFindClient(cloud, fixture)
			client.MoreHeaders = map[string]string{"X-Configured": "source-header"}
			endpoint, resourceBase := client.Endpoint, client.ResourceBase
			originalTransport := cloud.Provider.HTTPClient.Transport
			if originalTransport == nil {
				originalTransport = http.DefaultTransport
			}
			var middlewareCalls, gets, lists atomic.Int32
			cloud.Provider.HTTPClient.Transport = nativeIdentityFindTransport(func(r *http.Request) (*http.Response, error) {
				middlewareCalls.Add(1)
				if r.Context().Value(nativeIdentityFindContextKey{}) != "caller-context" {
					t.Error("query GET or List lost caller context")
				}
				owned := r.Clone(r.Context())
				owned.Header.Set("X-Identity-Middleware", "source-transport")
				return originalTransport.RoundTrip(owned)
			})
			check := func(r *http.Request, token string) {
				if r.Header.Get("X-Auth-Token") != token || r.Header.Get("X-Configured") != "source-header" || r.Header.Get("X-Identity-Middleware") != "source-transport" || !reflect.DeepEqual(r.URL.Query()["fields"], []string{"id", "name"}) {
					t.Error("query request lost live source configuration", r.URL, r.Header)
				}
			}
			cloud.Mux.HandleFunc("GET "+fixture.base+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				check(r, "test-token")
				if r.URL.Query().Has("name") {
					t.Error("automatic list hint leaked into GET", r.URL)
				}
				cloud.Provider.SetToken("changed-token")
				w.WriteHeader(404)
			})
			cloud.Mux.HandleFunc("GET "+fixture.path, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				check(r, "changed-token")
				if r.URL.Query().Get("name") != nativeIdentityFindName(fixture, "lookup") {
					t.Error("automatic list hint changed", r.URL)
				}
				testcloud.JSON(w, 200, nativeIdentityFindListBody(fixture, nativeIdentityFindRow("found", "lookup"), ""))
			})
			ctx := context.WithValue(context.Background(), nativeIdentityFindContextKey{}, "caller-context")
			option := resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: url.Values{"fields": {"id", "name"}}})
			value, err := fixture.new(t, client)(ctx, "lookup", option)
			if err != nil || value == nil || value.id != "found" || gets.Load() != 1 || lists.Load() != 1 || middlewareCalls.Load() != 2 {
				t.Fatal(value, err, gets.Load(), lists.Load(), middlewareCalls.Load())
			}
			if client.Endpoint != endpoint || client.ResourceBase != resourceBase || !reflect.DeepEqual(client.MoreHeaders, map[string]string{"X-Configured": "source-header"}) {
				t.Fatal("identity lookup mutated the service client", client)
			}
		})
	}
}
