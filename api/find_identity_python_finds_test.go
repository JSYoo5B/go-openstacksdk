package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/policies"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/regions"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns"
	fwgroups "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups"
	fwpolicies "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies"
	fwrules "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies"
	vpnservices "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Pinned Python find_* (Resource.find) sends GET {base}/{name_or_id} with
// **query, falls back after 400/403/404 to a list (adding name= only when the
// Resource query mapping declares name), and matches the row id or
// Resource.name. These fixtures cover each binding opted in by sdkgen.
type pythonFindFixture struct {
	name, prefix, base, singular, plural string
	// nameKey is the response key behind Python Resource.name ("" = none).
	nameKey string
	// nameQuery reports whether the Python list sends name=.
	nameQuery bool
	// listQuery reports whether the native list accepts caller query.
	listQuery bool
	numericID bool
	find      func(*gophercloud.ServiceClient) func(context.Context, string, ...resource.IdentityFindOption) (string, error)
}

func pythonFindAccess[T any](find func(context.Context, string, ...resource.IdentityFindOption) (*T, error), id func(*T) string) func(context.Context, string, ...resource.IdentityFindOption) (string, error) {
	return func(ctx context.Context, identity string, options ...resource.IdentityFindOption) (string, error) {
		value, err := find(ctx, identity, options...)
		if value == nil {
			return "", err
		}
		return id(value), err
	}
}

const pythonFindNeutron = "/reverse/neutron/v2.0"

func pythonFindFixtures() []pythonFindFixture {
	neutron := func(name, base, singular, plural string, find func(*gophercloud.ServiceClient) func(context.Context, string, ...resource.IdentityFindOption) (string, error)) pythonFindFixture {
		return pythonFindFixture{name: name, prefix: pythonFindNeutron, base: base, singular: singular, plural: plural, nameKey: "name", nameQuery: true, listQuery: true, find: find}
	}
	fixtures := []pythonFindFixture{
		neutron("find_tap_mirror", "/taas/tap_mirrors", "tap_mirror", "tap_mirrors", func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.IdentityFindOption) (string, error) {
			return pythonFindAccess(tapmirrors.New(c).FindIdentity, func(v *tapmirrors.TapMirror) string { return v.ID + "/" + v.Name })
		}),
		neutron("find_segment", "/segments", "segment", "segments", func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.IdentityFindOption) (string, error) {
			return pythonFindAccess(segments.New(c).FindIdentity, func(v *segments.Segment) string { return v.ID + "/" + v.Name })
		}),
		neutron("find_vpn_service", "/vpn/vpnservices", "vpnservice", "vpnservices", func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.IdentityFindOption) (string, error) {
			return pythonFindAccess(vpnservices.New(c).FindIdentity, func(v *vpnservices.Service) string { return v.ID + "/" + v.Name })
		}),
		neutron("find_vpn_endpoint_group", "/vpn/endpoint-groups", "endpoint_group", "endpoint_groups", func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.IdentityFindOption) (string, error) {
			return pythonFindAccess(endpointgroups.New(c).FindIdentity, func(v *endpointgroups.EndpointGroup) string { return v.ID + "/" + v.Name })
		}),
		neutron("find_vpn_ike_policy", "/vpn/ikepolicies", "ikepolicy", "ikepolicies", func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.IdentityFindOption) (string, error) {
			return pythonFindAccess(ikepolicies.New(c).FindIdentity, func(v *ikepolicies.Policy) string { return v.ID + "/" + v.Name })
		}),
		neutron("find_vpn_ipsec_policy", "/vpn/ipsecpolicies", "ipsecpolicy", "ipsecpolicies", func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.IdentityFindOption) (string, error) {
			return pythonFindAccess(ipsecpolicies.New(c).FindIdentity, func(v *ipsecpolicies.Policy) string { return v.ID + "/" + v.Name })
		}),
		neutron("find_vpn_ipsec_site_connection", "/vpn/ipsec-site-connections", "ipsec_site_connection", "ipsec_site_connections", func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.IdentityFindOption) (string, error) {
			return pythonFindAccess(siteconnections.New(c).FindIdentity, func(v *siteconnections.Connection) string { return v.ID + "/" + v.Name })
		}),
		neutron("find_firewall_rule", "/fwaas/firewall_rules", "firewall_rule", "firewall_rules", func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.IdentityFindOption) (string, error) {
			return pythonFindAccess(fwrules.New(c).FindIdentity, func(v *fwrules.Rule) string { return v.ID + "/" + v.Name })
		}),
		neutron("find_firewall_policy", "/fwaas/firewall_policies", "firewall_policy", "firewall_policies", func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.IdentityFindOption) (string, error) {
			return pythonFindAccess(fwpolicies.New(c).FindIdentity, func(v *fwpolicies.Policy) string { return v.ID + "/" + v.Name })
		}),
		neutron("find_firewall_group", "/fwaas/firewall_groups", "firewall_group", "firewall_groups", func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.IdentityFindOption) (string, error) {
			return pythonFindAccess(fwgroups.New(c).FindIdentity, func(v *fwgroups.Group) string { return v.ID + "/" + v.Name })
		}),
		neutron("find_bgpvpn", "/bgpvpn/bgpvpns", "bgpvpn", "bgpvpns", func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.IdentityFindOption) (string, error) {
			return pythonFindAccess(bgpvpns.New(c).FindIdentity, func(v *bgpvpns.BGPVPN) string { return v.ID + "/" + v.Name })
		}),
		{name: "find_bgp_peer", prefix: pythonFindNeutron, base: "/bgp-peers", singular: "bgp_peer", plural: "bgp_peers", nameKey: "name", find: func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.IdentityFindOption) (string, error) {
			return pythonFindAccess(peers.New(c).FindIdentity, func(v *peers.BGPPeer) string { return v.ID + "/" + v.Name })
		}},
		{name: "find_bgp_speaker", prefix: pythonFindNeutron, base: "/bgp-speakers", singular: "bgp_speaker", plural: "bgp_speakers", nameKey: "name", find: func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.IdentityFindOption) (string, error) {
			return pythonFindAccess(speakers.New(c).FindIdentity, func(v *speakers.BGPSpeaker) string { return v.ID + "/" + v.Name })
		}},
		{name: "find_ip", prefix: pythonFindNeutron, base: "/floatingips", singular: "floatingip", plural: "floatingips", nameKey: "floating_ip_address", listQuery: true, find: func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.IdentityFindOption) (string, error) {
			return pythonFindAccess(floatingips.New(c).FindIdentity, func(v *floatingips.FloatingIP) string { return v.ID + "/" + v.FloatingIP })
		}},
		{name: "find_policy", prefix: "/identity/v3", base: "/policies", singular: "policy", plural: "policies", listQuery: true, find: func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.IdentityFindOption) (string, error) {
			return pythonFindAccess(policies.New(c).FindIdentity, func(v *policies.Policy) string { return v.ID + "/" })
		}},
		{name: "find_region", prefix: "/identity/v3", base: "/regions", singular: "region", plural: "regions", listQuery: true, find: func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.IdentityFindOption) (string, error) {
			return pythonFindAccess(regions.New(c).FindIdentity, func(v *regions.Region) string { return v.ID + "/" })
		}},
		{name: "find_aggregate", prefix: "/compute/v2.1", base: "/os-aggregates", singular: "aggregate", plural: "aggregates", nameKey: "name", numericID: true, find: func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.IdentityFindOption) (string, error) {
			return pythonFindAccess(aggregates.New(c).FindIdentity, func(v *aggregates.Aggregate) string { return strconv.Itoa(v.ID) + "/" + v.Name })
		}},
	}
	return fixtures
}

func pythonFindClient(cloud *testcloud.Cloud, f pythonFindFixture) *gophercloud.ServiceClient {
	if f.prefix == pythonFindNeutron {
		c := cloud.Client("network", "/unused/catalog")
		c.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + pythonFindNeutron)
		return c
	}
	return cloud.Client(f.singular, f.prefix)
}

// pythonFindRow renders one row whose ID and Python name are given.
func pythonFindRow(f pythonFindFixture, id, name string) string {
	idValue := strconv.Quote(id)
	if f.numericID {
		idValue = id
	}
	switch f.nameKey {
	case "":
		// Keystone policies/regions have no name; a stray name key is ignored.
		return fmt.Sprintf(`{"id":%s,"name":%q}`, idValue, name)
	default:
		return fmt.Sprintf(`{"id":%s,%q:%q}`, idValue, f.nameKey, name)
	}
}

type pythonFindRecorder struct {
	mu    sync.Mutex
	calls []string
}

func (r *pythonFindRecorder) add(req *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, req.Method+" "+req.URL.Path+"?"+req.URL.RawQuery)
}

func (r *pythonFindRecorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...)
}

// serve answers the member GET with getStatus (200 returns getRow) and the
// list with rows.
func pythonFindServe(t *testing.T, f pythonFindFixture, getStatus int, getRow string, rows string) (*gophercloud.ServiceClient, *pythonFindRecorder) {
	t.Helper()
	cloud := testcloud.New(t)
	recorder := &pythonFindRecorder{}
	cloud.Mux.HandleFunc("GET "+f.prefix+f.base+"/{id}", func(w http.ResponseWriter, r *http.Request) {
		recorder.add(r)
		if getStatus == 200 {
			testcloud.JSON(w, 200, fmt.Sprintf(`{%q:%s}`, f.singular, getRow))
			return
		}
		testcloud.JSON(w, getStatus, `{"error":"native"}`)
	})
	cloud.Mux.HandleFunc("GET "+f.prefix+f.base, func(w http.ResponseWriter, r *http.Request) {
		recorder.add(r)
		testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[%s]}`, f.plural, rows))
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		recorder.add(r)
		w.WriteHeader(599)
	})
	return pythonFindClient(cloud, f), recorder
}

func pythonFindListQuery(f pythonFindFixture, identity string) string {
	if f.nameQuery {
		return url.Values{"name": {identity}}.Encode()
	}
	return ""
}

func TestPythonFindGETByIDReturnsWithoutList(t *testing.T) {
	for _, f := range pythonFindFixtures() {
		t.Run(f.name, func(t *testing.T) {
			id := "lookup-id"
			if f.numericID {
				id = "7"
			}
			c, recorder := pythonFindServe(t, f, 200, pythonFindRow(f, id, "server-name"), "")
			got, err := f.find(c)(context.Background(), id)
			wantName := "server-name"
			if f.nameKey == "" {
				wantName = ""
			}
			if err != nil || got != id+"/"+wantName {
				t.Fatal(got, err)
			}
			if calls := recorder.snapshot(); fmt.Sprint(calls) != fmt.Sprint([]string{"GET " + f.prefix + f.base + "/" + id + "?"}) {
				t.Fatal(calls)
			}
		})
	}
}

func TestPythonFindFallsBackAfter400403404AndMatchesIDOrPythonName(t *testing.T) {
	for _, f := range pythonFindFixtures() {
		for _, code := range []int{400, 403, 404} {
			t.Run(fmt.Sprint(f.name, "/", code), func(t *testing.T) {
				byID, other := "10", "11"
				if !f.numericID {
					byID, other = "row-id", "other-id"
				}
				rows := pythonFindRow(f, other, "unrelated") + "," + pythonFindRow(f, byID, "app")
				c, recorder := pythonFindServe(t, f, code, "", rows)
				got, err := f.find(c)(context.Background(), "app")
				if f.nameKey == "" {
					// Python compares Resource.name, which Keystone never populates.
					if err != nil || got != "" {
						t.Fatal("ID-only binding matched a name", got, err)
					}
				} else if err != nil || got != byID+"/app" {
					t.Fatal(got, err)
				}
				want := []string{"GET " + f.prefix + f.base + "/app?", "GET " + f.prefix + f.base + "?" + pythonFindListQuery(f, "app")}
				if calls := recorder.snapshot(); fmt.Sprint(calls) != fmt.Sprint(want) {
					t.Fatal(calls)
				}
				if f.numericID {
					// Python's integer id never equals the string name_or_id in
					// the list; Go compares IDs as text after a failed GET.
					return
				}
				c, _ = pythonFindServe(t, f, code, "", rows)
				if got, err := f.find(c)(context.Background(), byID); err != nil || got == "" || got[:len(byID)] != byID {
					t.Fatal("ID match in list fallback lost", got, err)
				}
			})
		}
	}
}

func TestPythonFindMissingDefaultsToNilAndStrictIsNotFound(t *testing.T) {
	for _, f := range pythonFindFixtures() {
		t.Run(f.name, func(t *testing.T) {
			c, recorder := pythonFindServe(t, f, 404, "", "")
			if got, err := f.find(c)(context.Background(), "absent"); err != nil || got != "" {
				t.Fatal(got, err)
			}
			if _, err := f.find(c)(context.Background(), "absent", resource.WithIdentityFindIgnoreMissing(false)); !errors.Is(err, resource.ErrNotFound) {
				t.Fatal(err)
			}
			if len(recorder.snapshot()) != 4 {
				t.Fatal(recorder.snapshot())
			}
			// Other GET failures are returned without a list.
			c, recorder = pythonFindServe(t, f, 500, "", "")
			if _, err := f.find(c)(context.Background(), "absent"); !gophercloud.ResponseCodeIs(err, 500) || len(recorder.snapshot()) != 1 {
				t.Fatal(err, recorder.snapshot())
			}
		})
	}
}

func TestPythonFindDuplicateMatchesAreAmbiguous(t *testing.T) {
	for _, f := range pythonFindFixtures() {
		if f.nameKey == "" {
			continue
		}
		t.Run(f.name, func(t *testing.T) {
			first, second := "1", "2"
			if !f.numericID {
				first, second = "first", "second"
			}
			c, _ := pythonFindServe(t, f, 404, "", pythonFindRow(f, first, "dup")+","+pythonFindRow(f, second, "dup"))
			if got, err := f.find(c)(context.Background(), "dup"); got != "" || !errors.Is(err, resource.ErrAmbiguous) {
				t.Fatal(got, err)
			}
		})
	}
}

func TestPythonFindForwardsQueryToGETAndList(t *testing.T) {
	for _, f := range pythonFindFixtures() {
		if f.numericID {
			// Pinned find_aggregate declares no **query.
			continue
		}
		t.Run(f.name, func(t *testing.T) {
			query := resource.WithIdentityFindQuery("project_id", "project-a")
			c, recorder := pythonFindServe(t, f, 404, "", pythonFindRow(f, "row-id", "app"))
			got, err := f.find(c)(context.Background(), "app", query)
			wantList := url.Values{"project_id": {"project-a"}}
			if f.nameQuery {
				wantList.Set("name", "app")
			}
			want := []string{"GET " + f.prefix + f.base + "/app?project_id=project-a"}
			if f.listQuery {
				want = append(want, "GET "+f.prefix+f.base+"?"+wantList.Encode())
				if f.nameKey != "" && (err != nil || got != "row-id/app") || f.nameKey == "" && (err != nil || got != "") {
					t.Fatal(got, err)
				}
			} else if got != "" || !errors.Is(err, resource.ErrUnsupported) {
				// The native BGP pager takes no query; Python accepts only
				// limit/marker there and rejects other keys.
				t.Fatal("query-less native list accepted caller query", got, err)
			}
			if calls := recorder.snapshot(); fmt.Sprint(calls) != fmt.Sprint(want) {
				t.Fatal(calls)
			}
		})
	}
}

// Python sends find_aggregate's name_or_id literally to os-aggregates/{x},
// although the native getter accepts only integers.
func TestPythonFindAggregateSendsLiteralIdentityToMemberRoute(t *testing.T) {
	var f pythonFindFixture
	for _, fixture := range pythonFindFixtures() {
		if fixture.numericID {
			f = fixture
		}
	}
	c, recorder := pythonFindServe(t, f, 404, "", pythonFindRow(f, "3", "rack-a"))
	if got, err := f.find(c)(context.Background(), "rack-a"); err != nil || got != "3/rack-a" {
		t.Fatal(got, err)
	}
	c2, recorder2 := pythonFindServe(t, f, 200, pythonFindRow(f, "3", "rack-a"), "")
	if got, err := f.find(c2)(context.Background(), "3"); err != nil || got != "3/rack-a" {
		t.Fatal(got, err)
	}
	if calls := append(recorder.snapshot(), recorder2.snapshot()...); fmt.Sprint(calls) != fmt.Sprint([]string{"GET /compute/v2.1/os-aggregates/rack-a?", "GET /compute/v2.1/os-aggregates?", "GET /compute/v2.1/os-aggregates/3?"}) {
		t.Fatal(calls)
	}
	// Explicit Ref ID lookup keeps its numeric validation before HTTP.
	if _, err := aggregates.New(c).Find(context.Background(), resource.ID("rack-a")); err == nil {
		t.Fatal("explicit numeric Ref accepted a name")
	}
}
