package connectiontest_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/network"
)

// _has_secgroups: a secgroup_source other than neutron or nova (any casing),
// including null, raises OpenStackCloudUnavailableFeature before the server
// lookup. Go returns network.ErrUnavailableSecurityGroups without any request.
func TestCloudServerSecurityGroupsUnavailableSourceFailsBeforeHTTP(t *testing.T) {
	for _, source := range []string{"none", "", "bogus"} {
		cloud := r2CloudComputeSecGroupFixture(t, nil)
		conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.Compute, "2.96"), sdk.WithSecurityGroupSource(source))
		if err != nil {
			t.Fatal(err)
		}
		ctx := context.Background()
		if ok, err := conn.AddServerSecurityGroups(ctx, "s1", "alpha"); ok || !errors.Is(err, network.ErrUnavailableSecurityGroups) {
			t.Fatalf("%q add: %v %v", source, ok, err)
		}
		if ok, err := conn.RemoveServerSecurityGroups(ctx, "s1", "alpha"); ok || !errors.Is(err, network.ErrUnavailableSecurityGroups) {
			t.Fatalf("%q remove: %v %v", source, ok, err)
		}
		if _, err := conn.GetSecurityGroupRecord(ctx, "alpha"); !errors.Is(err, network.ErrUnavailableSecurityGroups) {
			t.Fatalf("%q get: %v", source, err)
		}
		if calls := cloud.Calls(); len(calls) != 0 {
			t.Fatalf("%q: %+v", source, calls)
		}
	}
}

func cloudSecGroupNovaFixture(t *testing.T, options ...sdk.ConnectionOption) *r2CloudComputeCloud {
	t.Helper()
	// The legacy os-security-groups proxy needs a microversion below 2.36.
	cloud := r2CloudComputeNew(t, append([]sdk.ConnectionOption{sdk.WithMicroversion(sdk.Compute, "2.1")}, options...)...)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s1", 200, `{"server":{"id":"s1","name":"web-1","addresses":{}}}`)
	cloud.JSON("GET "+r2CloudComputeBase+"/os-security-groups", 200,
		`{"security_groups":[{"id":"1","name":"alpha","description":"","tenant_id":"p","rules":[]},{"id":"2","name":"gamma","description":"","tenant_id":"p","rules":[]}]}`)
	cloud.Handle("POST "+r2CloudComputeBase+"/servers/s1/action", func(w http.ResponseWriter, _ *http.Request, _ string) {
		w.WriteHeader(202)
	})
	cloud.JSON("GET /network/v2.0/security-groups", 500, `{}`)
	return cloud
}

func cloudSecGroupAssertNovaOnly(t *testing.T, cloud *r2CloudComputeCloud, want string) {
	t.Helper()
	for _, call := range cloud.Calls() {
		if strings.HasPrefix(call.Path, "/network/") {
			t.Fatalf("Neutron contacted: %+v", call)
		}
	}
	if got := strings.Join(r2CloudComputeActionBodies(cloud), "|"); got != want {
		t.Fatal(got)
	}
}

// _use_neutron_secgroups requires has_service('network') and the exact source
// "neutron": source nova, a differently cased "Neutron" and a disabled
// has_network flag all resolve groups through Nova while Neutron is in the
// catalog.
func TestCloudServerSecurityGroupsNovaSelection(t *testing.T) {
	cases := map[string][]sdk.ConnectionOption{
		"source nova":     {sdk.WithSecurityGroupSource("nova")},
		"source Neutron":  {sdk.WithSecurityGroupSource("Neutron")},
		"has_network off": {sdk.WithServiceEnabled("network", false)},
	}
	for name, options := range cases {
		t.Run(name, func(t *testing.T) {
			cloud := cloudSecGroupNovaFixture(t, options...)
			ok, err := cloud.Conn.AddServerSecurityGroups(context.Background(), "s1", "gamma", "alpha")
			if err != nil || !ok {
				t.Fatal(ok, err, cloud.Calls())
			}
			cloudSecGroupAssertNovaOnly(t, cloud, `{"addSecurityGroup":{"name":"gamma"}}|{"addSecurityGroup":{"name":"alpha"}}`)
			ok, err = cloud.Conn.RemoveServerSecurityGroups(context.Background(), "s1", "alpha")
			if err != nil || !ok {
				t.Fatal(ok, err)
			}
		})
	}
}
