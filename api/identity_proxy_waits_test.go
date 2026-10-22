package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/identity"
	appcreds "github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/federation"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/federationprotocols"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/identityproviders"
	identitylimits "github.com/JSYoo5B/go-openstacksdk/identity/v3/limits"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/policies"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/regions"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/serviceproviders"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/services"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// identityWaitFixture serves one GET route: `present` successful bodies, then
// a Keystone 404. Any other method or path fails the test.
func identityWaitFixture(t *testing.T, path, body string, present int32) (*gophercloud.ServiceClient, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	var gets atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != path {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			testcloud.JSON(w, 500, `{}`)
			return
		}
		if gets.Add(1) <= present {
			testcloud.JSON(w, 200, body)
			return
		}
		testcloud.JSON(w, 404, `{"error":{"code":404,"title":"Not Found","message":"missing"}}`)
	})
	return cloud.Client("identity", "/v3/"), &gets
}

func appCredentialScope(t *testing.T, c *gophercloud.ServiceClient) *resource.Collection[appcreds.ApplicationCredential] {
	t.Helper()
	scope, err := appcreds.New(c).InUser(context.Background(), resource.ID("u-1"))
	if err != nil {
		t.Fatalf("scope: %v", err)
	}
	return scope.Collection
}

// Every pinned identity/v3 Resource with allow_fetch has an SDK collection
// that the shared deletion waiter polls by fixed ID until 404.
func TestIdentityProxyWaitsDeleteEveryFetchableResource(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fast := []resource.WaitOption{resource.WithPollInterval(time.Millisecond), resource.WithTimeout(5 * time.Second)}
	cases := []struct {
		name, path, body string
		wait             func(*testing.T, *gophercloud.ServiceClient) error
	}{
		{"access rule", "/v3/users/u-1/access_rules/ar-1", `{"access_rule":{"id":"ar-1","path":"/v2.1/servers","method":"GET","service":"compute"}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, appcreds.New(c).AccessRuleResources("u-1"), resource.ID("ar-1"), fast...)
		}},
		{"application credential", "/v3/users/u-1/application_credentials/ac-1", `{"application_credential":{"id":"ac-1","name":"a"}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, appCredentialScope(t, c), resource.ID("ac-1"), fast...)
		}},
		{"credential", "/v3/credentials/cr-1", `{"credential":{"id":"cr-1","type":"ec2","blob":"{}"}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, credentials.New(c).Resources, resource.ID("cr-1"), fast...)
		}},
		{"domain", "/v3/domains/d-1", `{"domain":{"id":"d-1","name":"a"}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, domains.New(c).Resources, resource.ID("d-1"), fast...)
		}},
		{"domain config", "/v3/domains/d-1/config", `{"config":{"identity":{"driver":"ldap"}}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, domains.New(c).ConfigResources(), resource.ID("d-1"), fast...)
		}},
		{"endpoint", "/v3/endpoints/e-1", `{"endpoint":{"id":"e-1","interface":"public","url":"https://example.invalid"}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, endpoints.New(c).Resources, resource.ID("e-1"), fast...)
		}},
		{"endpoint group", "/v3/OS-EP-FILTER/endpoint_groups/eg-1", `{"endpoint_group":{"id":"eg-1","name":"a","filters":{}}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, endpointgroups.New(c).Resources(), resource.ID("eg-1"), fast...)
		}},
		{"federation protocol", "/v3/OS-FEDERATION/identity_providers/idp-1/protocols/saml2", `{"protocol":{"id":"saml2","mapping_id":"m-1"}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, federationprotocols.New(c).Resources("idp-1"), resource.ID("saml2"), fast...)
		}},
		{"group", "/v3/groups/g-1", `{"group":{"id":"g-1","name":"a"}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, groups.New(c).Resources, resource.ID("g-1"), fast...)
		}},
		{"identity provider", "/v3/OS-FEDERATION/identity_providers/idp-1", `{"identity_provider":{"id":"idp-1","enabled":true}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, identityproviders.New(c).Resources(), resource.ID("idp-1"), fast...)
		}},
		{"limit", "/v3/limits/l-1", `{"limit":{"id":"l-1","resource_name":"cores","resource_limit":1}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, identitylimits.New(c).Resources, resource.ID("l-1"), fast...)
		}},
		{"mapping", "/v3/OS-FEDERATION/mappings/m-1", `{"mapping":{"id":"m-1","rules":[]}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, federation.New(c).MappingResources(), resource.ID("m-1"), fast...)
		}},
		{"policy", "/v3/policies/po-1", `{"policy":{"id":"po-1","type":"application/json","blob":"{}"}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, policies.New(c).Resources, resource.ID("po-1"), fast...)
		}},
		{"project", "/v3/projects/p-1", `{"project":{"id":"p-1","name":"a"}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, projects.New(c).Resources, resource.ID("p-1"), fast...)
		}},
		{"region", "/v3/regions/r-1", `{"region":{"id":"r-1"}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, regions.New(c).Resources, resource.ID("r-1"), fast...)
		}},
		{"registered limit", "/v3/registered_limits/rl-1", `{"registered_limit":{"id":"rl-1","resource_name":"cores","default_limit":1}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, registeredlimits.New(c).Resources, resource.ID("rl-1"), fast...)
		}},
		{"role", "/v3/roles/ro-1", `{"role":{"id":"ro-1","name":"a"}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, roles.New(c).Resources, resource.ID("ro-1"), fast...)
		}},
		{"service", "/v3/services/s-1", `{"service":{"id":"s-1","type":"compute"}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, services.New(c).Resources, resource.ID("s-1"), fast...)
		}},
		{"service provider", "/v3/OS-FEDERATION/service_providers/sp-1", `{"service_provider":{"id":"sp-1","enabled":true}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, serviceproviders.New(c).Resources(), resource.ID("sp-1"), fast...)
		}},
		{"trust", "/v3/OS-TRUST/trusts/t-1", `{"trust":{"id":"t-1","impersonation":false}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, trusts.New(c).Resources, resource.ID("t-1"), fast...)
		}},
		{"user", "/v3/users/u-1", `{"user":{"id":"u-1","name":"a"}}`, func(t *testing.T, c *gophercloud.ServiceClient) error {
			return identity.WaitForDelete(ctx, users.New(c).Resources, resource.ID("u-1"), fast...)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client, gets := identityWaitFixture(t, tc.path, tc.body, 2)
			if err := tc.wait(t, client); err != nil || gets.Load() != 3 {
				t.Fatalf("err=%v gets=%d", err, gets.Load())
			}
		})
	}
}

func TestIdentityProxyWaitsStatusAttributeDefaultsAndTerminalErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fast := resource.WithPollInterval(time.Millisecond)

	// No Keystone v3 model declares status: the default fails before HTTP.
	client, gets := identityWaitFixture(t, "/v3/projects/p-1", `{"project":{"id":"p-1","name":"a","description":"Ready"}}`, 1<<20)
	collection := projects.New(client).Resources
	if _, err := identity.WaitForStatus(ctx, collection, resource.ID("p-1"), "ready", fast); !errors.Is(err, resource.ErrUnsupported) || gets.Load() != 0 {
		t.Fatalf("default attribute err=%v gets=%d", err, gets.Load())
	}
	project, err := identity.WaitForStatus(ctx, collection, resource.ID("p-1"), "READY", fast, resource.WithStatusAttribute("description"))
	if err != nil || project.Description != "Ready" || gets.Load() != 1 {
		t.Fatalf("project=%v err=%v gets=%d", project, err, gets.Load())
	}
	// ERROR in the selected attribute fails with the default failure list.
	failClient, _ := identityWaitFixture(t, "/v3/OS-FEDERATION/identity_providers/idp-1", `{"identity_provider":{"id":"idp-1","description":"error"}}`, 1<<20)
	if _, err := identity.WaitForStatus(ctx, identityproviders.New(failClient).Resources(), resource.ID("idp-1"), "ready", fast, resource.WithStatusAttribute("Description")); !errors.Is(err, resource.ErrFailedState) {
		t.Fatalf("ERROR err=%v", err)
	}
	// An existing project keeps the deletion wait polling until its deadline.
	before := gets.Load()
	if err := identity.WaitForDelete(ctx, collection, resource.ID("p-1"), fast, resource.WithTimeout(30*time.Millisecond)); !errors.Is(err, context.DeadlineExceeded) || gets.Load()-before < 2 {
		t.Fatalf("present delete err=%v gets=%d", err, gets.Load()-before)
	}
	// A 403 is not absence.
	cloud := testcloud.New(t)
	var denied atomic.Int32
	cloud.Mux.HandleFunc("/v3/regions/r-2", func(w http.ResponseWriter, r *http.Request) {
		denied.Add(1)
		testcloud.JSON(w, 403, `{"error":{"code":403}}`)
	})
	if err := identity.WaitForDelete(ctx, regions.New(cloud.Client("identity", "/v3/")).Resources, resource.ID("r-2"), fast); err == nil || denied.Load() != 1 {
		t.Fatalf("403 err=%v gets=%d", err, denied.Load())
	}
}
