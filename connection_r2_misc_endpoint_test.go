package openstack_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"

	"github.com/gophercloud/gophercloud/v2"
)

func TestR2MiscEndpointForCatalogSelectionAndOverride(t *testing.T) {
	cloud := testcloud.New(t)
	newR2MiscKeystone(t, cloud)
	c := r2MiscConnect(t, cloud, gophercloud.AuthOptions{Username: "alice", Password: "pw", DomainName: "Default"}, sdk.WithRegion("RegionOne"))
	base := cloud.Server.URL
	cases := []struct {
		name    string
		service string
		options []sdk.EndpointOption
		want    string
		found   bool
	}{
		{"network default", "network", nil, base + "/network", true},
		{"internal interface", "network", []sdk.EndpointOption{sdk.WithEndpointInterface(gophercloud.AvailabilityInternal)}, "", false},
		{"override ignores interface", "compute", []sdk.EndpointOption{sdk.WithEndpointInterface(gophercloud.AvailabilityInternal)}, base + "/compute/v2.1/project", true},
		{"URL suffix ignored", "network", []sdk.EndpointOption{sdk.WithEndpointInterface("publicURL")}, base + "/network", true},
		{"other region", "network", []sdk.EndpointOption{sdk.WithEndpointRegion("RegionTwo")}, "", false},
		{"any region", "network", []sdk.EndpointOption{sdk.WithEndpointRegion("")}, base + "/network", true},
		{"service name", "network", []sdk.EndpointOption{sdk.WithEndpointServiceName("other")}, "", false},
		{"missing type", "volumev3", nil, "", false},
		{"override wins over catalog", "compute", nil, base + "/compute/v2.1/project", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, found, err := c.EndpointFor(tc.service, tc.options...)
			if err != nil || found != tc.found || got != tc.want {
				t.Fatalf("got %q %v %v", got, found, err)
			}
			session, sessionFound, err := c.SessionEndpoint(tc.service, tc.options...)
			if err != nil || session != got || sessionFound != found {
				t.Fatalf("session endpoint %q %v %v", session, sessionFound, err)
			}
			has, err := c.HasService(tc.service, tc.options...)
			if err != nil || has != tc.found {
				t.Fatalf("has service %v %v", has, err)
			}
		})
	}
	// Catalog order and the first matching endpoint decide, as in keystoneauth.
	other, err := sdk.Connect(context.Background(), sdk.WithAuth(gophercloud.AuthOptions{IdentityEndpoint: base + "/v3", Username: "a", Password: "p", DomainName: "Default"}))
	if err != nil {
		t.Fatal(err)
	}
	if got, ok, err := other.EndpointFor("compute"); err != nil || !ok || got != base+"/compute/v2.1/pub" {
		t.Fatalf("first public endpoint %q %v %v", got, ok, err)
	}
	if got, ok, err := other.EndpointFor("compute", sdk.WithEndpointRegion("RegionTwo")); err != nil || !ok || got != base+"/compute/v2.1/pub2" {
		t.Fatalf("region endpoint %q %v %v", got, ok, err)
	}
}

func TestR2MiscEndpointForInvalidAndUnrecordedCatalog(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	var seen gophercloud.EndpointOpts
	provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		seen = opts
		if opts.Type == "absent" {
			return "", gophercloud.ErrEndpointNotFound{}
		}
		if opts.Type == "broken" {
			return "", errors.New("boom")
		}
		return "https://svc.example/v1/", nil
	}
	c, err := sdk.FromProvider(provider, sdk.WithRegion("r1"), sdk.WithInterface(gophercloud.AvailabilityInternal))
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := c.EndpointFor("thing", sdk.WithEndpointServiceName("svc"))
	if err != nil || !ok || got != "https://svc.example/v1/" {
		t.Fatalf("%q %v %v", got, ok, err)
	}
	if seen.Type != "thing" || seen.Region != "r1" || seen.Availability != gophercloud.AvailabilityInternal || seen.Name != "svc" {
		t.Fatalf("locator options %+v", seen)
	}
	if _, ok, err := c.EndpointFor("absent"); err != nil || ok {
		t.Fatalf("absent %v %v", ok, err)
	}
	if _, ok, err := c.EndpointFor("broken"); err == nil || ok {
		t.Fatalf("broken lookup %v %v", ok, err)
	}
	if _, _, err := c.EndpointFor(""); err == nil {
		t.Error("empty type accepted")
	}
	if _, _, err := c.EndpointFor("x", nil); err == nil {
		t.Error("nil option accepted")
	}
	if _, _, err := c.EndpointFor("x", sdk.WithEndpointInterface("")); err == nil {
		t.Error("empty interface accepted")
	}
	bare, err := sdk.FromProvider(&gophercloud.ProviderClient{})
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := bare.HasService("thing"); err == nil || ok || !errors.Is(err, sdk.ErrNoAuthResult) {
		t.Fatalf("no catalog: %v %v", ok, err)
	}
}

func TestR2MiscAddServiceBuildsLazyClientOncePerConnection(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /custom/v1/things", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"id":"`+r.Header.Get("X-Openstack-Request-Id")+"|"+r.Header.Get("X-Auth-Token")+`"}`)
	})
	var factoryCalls int
	factory := func(p *gophercloud.ProviderClient, eo gophercloud.EndpointOpts) (*gophercloud.ServiceClient, error) {
		factoryCalls++
		endpoint, err := p.EndpointLocator(eo)
		if err != nil {
			return nil, err
		}
		return &gophercloud.ServiceClient{ProviderClient: p, Type: "custom", Endpoint: gophercloud.NormalizeURL(endpoint)}, nil
	}
	c, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := c.AddService(sdk.Service("custom"), factory, cloud.Server.URL+"/custom/v1"); err != nil {
		t.Fatal(err)
	}
	if factoryCalls != 0 {
		t.Fatal("factory ran at registration")
	}
	if err := c.AddService(sdk.Service("custom"), factory); err == nil {
		t.Error("duplicate registration accepted")
	}
	if err := c.AddService(sdk.Compute, factory); err == nil {
		t.Error("built-in service replaced")
	}
	if err := c.AddService("", factory); err == nil {
		t.Error("empty name accepted")
	}
	if err := c.AddService("x", nil); err == nil {
		t.Error("nil factory accepted")
	}
	if err := c.AddService("x", factory, "not a url"); err == nil {
		t.Error("bad endpoint accepted")
	}
	if _, err := c.CustomServiceClient(ctx, "unknown"); err == nil {
		t.Error("unregistered service returned a client")
	}
	first, err := c.CustomServiceClient(ctx, "custom")
	if err != nil {
		t.Fatal(err)
	}
	second, _ := c.CustomServiceClient(ctx, "custom")
	if first != second || factoryCalls != 1 || first.ProviderClient != cloud.Provider {
		t.Fatalf("cache: calls=%d same=%v", factoryCalls, first == second)
	}
	var body struct{ ID string }
	if _, err := first.Get(ctx, first.ServiceURL("things"), &body, nil); err != nil || body.ID != "|test-token" {
		t.Fatalf("request %v %q", err, body.ID)
	}
	// A derived GlobalRequest Connection inherits the registration with its own
	// client carrying the header; the parent client stays untouched.
	derived, err := c.GlobalRequest("req-custom")
	if err != nil {
		t.Fatal(err)
	}
	child, err := derived.CustomServiceClient(ctx, "custom")
	if err != nil {
		t.Fatal(err)
	}
	if child == first || factoryCalls != 2 {
		t.Fatalf("derived client shared: calls=%d", factoryCalls)
	}
	if _, err := child.Get(ctx, child.ServiceURL("things"), &body, nil); err != nil || body.ID != "req-custom|test-token" {
		t.Fatalf("derived request %v %q", err, body.ID)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := c.CustomServiceClient(canceled, "custom"); !errors.Is(err, context.Canceled) {
		t.Errorf("canceled: %v", err)
	}
}

func TestR2MiscAddServiceWithoutEndpointUsesLocatorAndReportsFactoryErrors(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		return "https://" + opts.Type + ".example/", nil
	}
	c, err := sdk.FromProvider(provider)
	if err != nil {
		t.Fatal(err)
	}
	factory := func(p *gophercloud.ProviderClient, eo gophercloud.EndpointOpts) (*gophercloud.ServiceClient, error) {
		eo.Type = "widgets"
		endpoint, err := p.EndpointLocator(eo)
		return &gophercloud.ServiceClient{Endpoint: endpoint}, err
	}
	if err := c.AddService("widgets", factory); err != nil {
		t.Fatal(err)
	}
	client, err := c.CustomServiceClient(context.Background(), "widgets")
	if err != nil || client.Endpoint != "https://widgets.example/" || client.ProviderClient != provider {
		t.Fatalf("%+v %v", client, err)
	}
	failing := errors.New("factory failed")
	if err := c.AddService("broken", func(*gophercloud.ProviderClient, gophercloud.EndpointOpts) (*gophercloud.ServiceClient, error) {
		return nil, failing
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.CustomServiceClient(context.Background(), "broken"); !errors.Is(err, failing) {
		t.Fatalf("factory error %v", err)
	}
	if err := c.AddService("empty", func(*gophercloud.ProviderClient, gophercloud.EndpointOpts) (*gophercloud.ServiceClient, error) {
		return nil, nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.CustomServiceClient(context.Background(), "empty"); err == nil {
		t.Fatal("nil client accepted")
	}
	noLocator, err := sdk.FromProvider(&gophercloud.ProviderClient{})
	if err != nil {
		t.Fatal(err)
	}
	_ = noLocator.AddService("widgets", factory)
	if _, err := noLocator.CustomServiceClient(context.Background(), "widgets"); err == nil {
		t.Fatal("missing locator accepted")
	}
}
