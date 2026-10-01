package api_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/compute/v2/flavors"
	"gophercloudsdk/image/v2/images"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/loadbalancer/v2/pools"
	"gophercloudsdk/network/v2/extensions/layer3/routers"
	securitygroups "gophercloudsdk/network/v2/extensions/security/groups"
	"gophercloudsdk/resource"
)

// Wire status is an explicit service query. It must not be dropped merely
// because a native model has no Status, or mistaken for WithStatus's local
// filter on an Octavia member. This matrix uses the sixteen audited bindings.
func nativeIdentityWireStatusFixtures() []nativeIdentityFindFixture {
	return append(nativeIdentityFindFixtures(),
		nativeIdentityFindFixture{"compute", "/reverse/nova/v2.1/project", "/reverse/nova/v2.1/project/flavors", "/reverse/nova/v2.1/project/flavors/detail", "flavor", "flavors", false,
			func(_ *testing.T, c *gophercloud.ServiceClient) nativeIdentityFindAccess {
				return nativeIdentityFindAccessFor(flavors.New(c).FindIdentity, func(v *flavors.Flavor) string { return v.ID }, func(v *flavors.Flavor) string { return v.Name })
			}},
		nativeIdentityFindFixture{"image", "/reverse/glance/v2", "/reverse/glance/v2/images", "/reverse/glance/v2/images", "image", "images", false,
			func(_ *testing.T, c *gophercloud.ServiceClient) nativeIdentityFindAccess {
				return nativeIdentityFindAccessFor(images.New(c).FindIdentity, func(v *images.Image) string { return v.ID }, func(v *images.Image) string { return v.Name })
			}},
		nativeIdentityFindFixture{"network", "/reverse/neutron/v2.0", "/reverse/neutron/v2.0/routers", "/reverse/neutron/v2.0/routers", "router", "routers", false,
			func(_ *testing.T, c *gophercloud.ServiceClient) nativeIdentityFindAccess {
				return nativeIdentityFindAccessFor(routers.New(c).FindIdentity, func(v *routers.Router) string { return v.ID }, func(v *routers.Router) string { return v.Name })
			}},
		nativeIdentityFindFixture{"network", "/reverse/neutron/v2.0", "/reverse/neutron/v2.0/security-groups", "/reverse/neutron/v2.0/security-groups", "security_group", "security_groups", false,
			func(_ *testing.T, c *gophercloud.ServiceClient) nativeIdentityFindAccess {
				return nativeIdentityFindAccessFor(securitygroups.New(c).FindIdentity, func(v *securitygroups.SecGroup) string { return v.ID }, func(v *securitygroups.SecGroup) string { return v.Name })
			}},
	)
}

func TestNativeFindIdentityWireStatusSurvivesBothPhases(t *testing.T) {
	for _, f := range nativeIdentityWireStatusFixtures() {
		for _, values := range [][]string{nil, {}, {"", "first", "second"}} {
			t.Run(f.plural+"/"+url.Values{"status": values}.Encode(), func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets, lists atomic.Int32
				query := url.Values{"status": values, "name": {"caller-filter"}, "vendor": {"first", "second"}}
				if f.plural == "flavors" {
					query["is_public"] = []string{"true"}
				}
				want := query.Encode()
				option := resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: query})
				query["status"] = []string{"mutated"}
				cloud.Mux.HandleFunc("GET "+f.base+"/lookup", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					if r.URL.RawQuery != want {
						t.Error("GET changed raw wire query", r.URL, want)
					}
					w.WriteHeader(403)
				})
				cloud.Mux.HandleFunc("GET "+f.path, func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					if r.URL.RawQuery != want {
						t.Error("LIST dropped or coerced raw status", r.URL, want)
					}
					// A wire query alone must not become a local status filter.
					testcloud.JSON(w, 200, nativeIdentityFindListBody(f, `{"id":"found","name":"lookup","status":"ACTIVE","provisioning_status":"ERROR"}`, ""))
				})
				value, err := f.new(t, nativeIdentityFindClient(cloud, f))(context.Background(), "lookup", option)
				if err != nil || value == nil || value.id != "found" || gets.Load() != 1 || lists.Load() != 1 {
					t.Fatal(value, err, gets.Load(), lists.Load())
				}
			})
		}
	}
}

func TestNativeFindIdentityWireStatusServerRejectionIsTerminal(t *testing.T) {
	for _, f := range nativeIdentityWireStatusFixtures() {
		t.Run(f.plural, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, lists atomic.Int32
			cloud.Mux.HandleFunc("GET "+f.base+"/lookup", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); w.WriteHeader(404) })
			cloud.Mux.HandleFunc("GET "+f.path, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if !reflect.DeepEqual(r.URL.Query()["status"], []string{"vendor-status"}) {
					t.Error("server never received explicit status", r.URL)
				}
				w.WriteHeader(400)
			})
			value, err := f.new(t, nativeIdentityFindClient(cloud, f))(context.Background(), "lookup", resource.WithIdentityFindQuery("status", "vendor-status"))
			if value != nil || !gophercloud.ResponseCodeIs(err, 400) || errors.Is(err, resource.ErrNotFound) || gets.Load() != 1 || lists.Load() != 1 {
				t.Fatal("server query rejection was hidden", value, err, gets.Load(), lists.Load())
			}
		})
	}
}

func TestNativeFindIdentityMemberWireAndTypedStatusRemainSeparate(t *testing.T) {
	for _, test := range []struct {
		name    string
		options []resource.ListOption
		ids     []string
		wire    string
	}{
		{"raw", []resource.ListOption{resource.WithQuery("status", "wire-filter")}, []string{"inactive", "active"}, "wire-filter"},
		{"local", []resource.ListOption{resource.WithStatus("ACTIVE")}, []string{"active"}, ""},
		{"local-cap", []resource.ListOption{resource.WithStatus("ACTIVE"), resource.WithMaxItems(1)}, []string{}, ""},
		{"typed-then-raw", []resource.ListOption{resource.WithStatus("ACTIVE"), resource.WithQuery("status", "ERROR")}, []string{"inactive"}, ""},
		{"raw-then-typed", []resource.ListOption{resource.WithQuery("status", "ERROR"), resource.WithStatus("ACTIVE")}, []string{"active"}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/lbaas/pools/parent/members", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if test.wire == "" && r.URL.Query().Has("status") || test.wire != "" && r.URL.Query().Get("status") != test.wire || r.URL.Query().Get("vendor") != "kept" || r.URL.Query().Has("limit") {
					t.Error("local/raw policy changed wire query", r.URL)
				}
				testcloud.JSON(w, 200, `{"members":[{"id":"inactive","name":"same","provisioning_status":"ERROR"},{"id":"active","name":"same","provisioning_status":"ACTIVE"}]}`)
			})
			scope, err := pools.New(cloud.Client("load-balancer", "/v2.0")).Members(context.Background(), resource.ID("parent"))
			if err != nil {
				t.Fatal(err)
			}
			options := append(append([]resource.ListOption(nil), test.options...), resource.WithQuery("vendor", "kept"))
			values, err := scope.All(context.Background(), options...)
			ids := make([]string, 0, len(values))
			for _, v := range values {
				ids = append(ids, v.ID)
			}
			if err != nil || !reflect.DeepEqual(ids, test.ids) || calls.Load() != 1 {
				t.Fatal(ids, err, calls.Load())
			}
		})
	}
}
