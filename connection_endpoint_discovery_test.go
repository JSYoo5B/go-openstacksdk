package openstack_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

type discoveryCall struct{ Path, Token string }

type discoveryFixture struct {
	cloud *testcloud.Cloud
	base  string
	mu    sync.Mutex
	calls []discoveryCall
}

func (f *discoveryFixture) Calls() []discoveryCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]discoveryCall(nil), f.calls...)
}

func (f *discoveryFixture) reset() {
	f.mu.Lock()
	f.calls = nil
	f.mu.Unlock()
}

func (f *discoveryFixture) handle(pattern string, handler func(w http.ResponseWriter, r *http.Request)) {
	f.cloud.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.calls = append(f.calls, discoveryCall{r.URL.Path, r.Header.Get("X-Auth-Token")})
		f.mu.Unlock()
		handler(w, r)
	})
}

// newDiscoveryFixture serves a project-scoped Keystone token whose catalog
// lists versioned, unversioned and project-suffixed endpoints, and version
// documents beneath them.
func newDiscoveryFixture(t *testing.T) *discoveryFixture {
	t.Helper()
	cloud := testcloud.New(t)
	f := &discoveryFixture{cloud: cloud, base: cloud.Server.URL}
	endpoint := func(kind, url string) string {
		return fmt.Sprintf(`{"type":%q,"name":%q,"id":%q,"endpoints":[{"id":"e-%s","interface":"public","region":"RegionOne","region_id":"RegionOne","url":%q}]}`, kind, kind, kind, kind, f.base+url)
	}
	catalog := strings.Join([]string{
		endpoint("compute", "/compute/v2.1/proj1"),
		endpoint("network", "/network"),
		endpoint("block-storage", "/volume/v3/proj1"),
		endpoint("image", "/image"),
		endpoint("placement", "/placement"),
		endpoint("dns", "/dns"),
		endpoint("identity", "/identity/v3"),
		endpoint("volumev3", "/legacy/v3/proj1"),
	}, ",")
	cloud.Mux.HandleFunc("POST /v3/auth/tokens", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Subject-Token", "test-token")
		testcloud.JSON(w, http.StatusCreated, `{"token":{"expires_at":"2099-01-01T00:00:00Z","user":{"id":"u1","name":"alice"},`+
			`"project":{"id":"proj1","name":"demo","domain":{"id":"default","name":"Default"}},"catalog":[`+catalog+`]}}`)
	})
	f.handle("GET /compute/{$}", func(w http.ResponseWriter, _ *http.Request) {
		testcloud.JSON(w, 200, `{"versions":[{"id":"v2.0","status":"SUPPORTED","links":[{"rel":"self","href":"`+f.base+`/compute/v2/"}]},`+
			`{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"`+f.base+`/compute/v2.1/"}]}]}`)
	})
	f.handle("GET /network", func(w http.ResponseWriter, _ *http.Request) {
		testcloud.JSON(w, 200, `{"versions":[{"id":"v2.0","status":"CURRENT","links":[{"rel":"self","href":"`+f.base+`/network/v2.0/"}]}]}`)
	})
	f.handle("GET /volume/{$}", func(w http.ResponseWriter, _ *http.Request) {
		testcloud.JSON(w, 300, `{"versions":[`+
			`{"id":"v3.0","status":"CURRENT","min_version":"3.0","version":"3.71","links":[{"rel":"self","href":"`+f.base+`/volume/v3/"}]},`+
			`{"id":"v2.0","status":"DEPRECATED","links":[{"rel":"describedby","href":"https://docs.example/"},{"rel":"self","href":"`+f.base+`/volume/v2/"}]}]}`)
	})
	f.handle("GET /image", func(w http.ResponseWriter, _ *http.Request) {
		// Relative links resolve against the discovery URL; a foreign host is
		// repaired onto the discovery host, as keystoneauth does.
		testcloud.JSON(w, 300, `{"versions":[`+
			`{"id":"v3.0","status":"EXPERIMENTAL","links":[{"rel":"self","href":"v3/"}]},`+
			`{"id":"v2.15","status":"CURRENT","links":[{"rel":"self","href":"https://proxy.invalid/v2.15/"}]},`+
			`{"id":"v2.1","status":"SUPPORTED","links":[{"rel":"self","href":"v2/"}]},`+
			`{"id":"v2.2","status":"SUPPORTED"},`+
			`{"id":"v1.1","links":[{"rel":"self","href":"v1/"}]},`+
			`{"id":"v1.0","status":"DEPRECATED","links":[{"rel":"self","href":"v1/"}]}]}`)
	})
	f.handle("GET /placement", func(w http.ResponseWriter, _ *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	f.handle("GET /dns", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Token") == "" {
			testcloud.JSON(w, 401, `{}`)
			return
		}
		testcloud.JSON(w, 200, `{"versions":{"values":[{"id":"v2","status":"stable","links":[{"rel":"self","href":"`+f.base+`/dns/v2"}]}]}}`)
	})
	return f
}

func (f *discoveryFixture) connect(t *testing.T, options ...sdk.ConnectionOption) *sdk.Connection {
	t.Helper()
	auth := gophercloud.AuthOptions{IdentityEndpoint: f.base + "/v3", Username: "alice", Password: "pw", DomainName: "Default"}
	c, err := sdk.Connect(context.Background(), append([]sdk.ConnectionOption{sdk.WithAuth(auth), sdk.WithRegion("RegionOne")}, options...)...)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestSessionEndpointContextVersionDiscovery(t *testing.T) {
	f := newDiscoveryFixture(t)
	c := f.connect(t)
	ctx := context.Background()
	base := f.base
	cases := []struct {
		name, service, min, max string
		want                    string
		found                   bool
		calls                   []string
	}{
		{"no bounds equals catalog", "network", "", "", base + "/network", true, nil},
		{"catalog version within major needs no request", "compute", "2", "2.latest", base + "/compute/v2.1/proj1", true, nil},
		{"project suffix kept on exact catalog match", "block-storage", "3.0", "3.latest", base + "/volume/v3/proj1", true, nil},
		{"unversioned catalog discovers versioned URL", "network", "2", "", base + "/network/v2.0/", true, []string{"/network"}},
		{"no matching version is not found", "network", "3", "", "", false, []string{"/network"}},
		{"older major from root document with project restored", "block-storage", "2", "2.latest", base + "/volume/v2/proj1", true, []string{"/volume/"}},
		{"newest version in range and host repair", "image", "2.1", "2.latest", base + "/image/v2.15/", true, []string{"/image"}},
		{"bounded maximum picks older minor", "image", "2.0", "2.10", base + "/image/v2/", true, []string{"/image"}},
		{"deprecated kept and experimental dropped", "image", "1", "1.latest", base + "/image/v1/", true, []string{"/image"}},
		{"minimum without maximum means latest", "image", "1", "", base + "/image/v2.15/", true, []string{"/image"}},
		{"maximum without minimum finds nothing", "image", "", "2", "", false, []string{"/image"}},
		{"latest picks the newest", "image", "latest", "", base + "/image/v2.15/", true, []string{"/image"}},
		{"unreachable discovery falls back to catalog", "placement", "1", "", base + "/placement", true, []string{"/placement"}},
		{"401 retried with token", "dns", "2", "", base + "/dns/v2", true, []string{"/dns", "/dns"}},
		{"identity catalog version matches", "identity", "3", "3.latest", base + "/identity/v3", true, nil},
		{"alias implies its major", "volumev3", "", "", base + "/legacy/v3/proj1", true, nil},
		{"absent service", "object-store", "1", "", "", false, nil},
		{"other major consults the root document", "compute", "3", "", "", false, []string{"/compute/"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f.reset()
			got, found, err := c.SessionEndpointContext(ctx, tc.service, sdk.WithEndpointMinVersion(tc.min), sdk.WithEndpointMaxVersion(tc.max))
			if err != nil || found != tc.found || got != tc.want {
				t.Fatalf("got %q %v %v", got, found, err)
			}
			var paths []string
			for _, call := range f.Calls() {
				paths = append(paths, call.Path)
			}
			if strings.Join(paths, ",") != strings.Join(tc.calls, ",") {
				t.Fatalf("discovery requests %v", paths)
			}
		})
	}
	f.reset()
	if _, _, err := c.SessionEndpointContext(ctx, "dns", sdk.WithEndpointMinVersion("2")); err != nil {
		t.Fatal(err)
	}
	if calls := f.Calls(); len(calls) != 2 || calls[0].Token != "" || calls[1].Token != "test-token" {
		t.Fatalf("token handling %+v", calls)
	}
}

func TestSessionEndpointContextOverrideInvalidAndCanceled(t *testing.T) {
	f := newDiscoveryFixture(t)
	c := f.connect(t, sdk.WithEndpoint(sdk.Network, f.base+"/override"))
	ctx := context.Background()
	if got, found, err := c.SessionEndpointContext(ctx, "network", sdk.WithEndpointMinVersion("9")); err != nil || !found || got != f.base+"/override" {
		t.Fatalf("override %q %v %v", got, found, err)
	}
	for name, options := range map[string][]sdk.EndpointOption{
		"invalid minimum":       {sdk.WithEndpointMinVersion("two")},
		"minimum above maximum": {sdk.WithEndpointMinVersion("2.5"), sdk.WithEndpointMaxVersion("2.1")},
		"latest with bound max": {sdk.WithEndpointMinVersion("latest"), sdk.WithEndpointMaxVersion("2")},
	} {
		if _, _, err := c.SessionEndpointContext(ctx, "image", options...); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if _, _, err := c.SessionEndpointContext(ctx, "volumev3", sdk.WithEndpointMinVersion("2")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("implied version mismatch: %v", err)
	}
	if _, _, err := c.SessionEndpoint("image", sdk.WithEndpointMinVersion("2")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("bounds without context: %v", err)
	}
	if _, _, err := c.SessionEndpointContext(ctx, ""); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("empty type: %v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	f.reset()
	if _, _, err := c.SessionEndpointContext(canceled, "image", sdk.WithEndpointMinVersion("2")); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled: %v", err)
	}
	if calls := f.Calls(); len(calls) != 0 {
		t.Fatal(calls)
	}
}

// Without a recorded token project, a path segment after a version segment is
// treated as a project ID, removed for discovery and appended again.
func TestSessionEndpointContextProjectPeekWithoutTokenProject(t *testing.T) {
	f := newDiscoveryFixture(t)
	f.cloud.Provider.EndpointLocator = func(o gophercloud.EndpointOpts) (string, error) {
		if o.Type == "block-storage" {
			return f.base + "/volume/v3/tenant9", nil
		}
		return "", gophercloud.ErrEndpointNotFound{}
	}
	c, err := sdk.FromProvider(f.cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := c.HasServiceVersion(context.Background(), "block-storage", "2")
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	got, found, err := c.SessionEndpointContext(context.Background(), "block-storage", sdk.WithEndpointMinVersion("2"), sdk.WithEndpointMaxVersion("2.latest"))
	if err != nil || !found || got != f.base+"/volume/v2/tenant9" {
		t.Fatalf("%q %v %v", got, found, err)
	}
}

func TestHasServiceVersionAndServiceFlags(t *testing.T) {
	f := newDiscoveryFixture(t)
	ctx := context.Background()
	c := f.connect(t)
	for _, tc := range []struct {
		service, version string
		want             bool
	}{
		{"compute", "2", true},
		{"compute", "3", false},
		{"network", "v2.0", true},
		{"network", "", true},
		{"block-storage", "2", true},
		{"placement", "1", true},
		{"object-store", "", false},
		// Aliases are not official service types: has_service is false.
		{"volumev3", "", false},
	} {
		f.reset()
		got, err := c.HasServiceVersion(ctx, tc.service, tc.version)
		if err != nil || got != tc.want {
			t.Fatalf("%s %s: %v %v", tc.service, tc.version, got, err)
		}
	}
	if ok, err := c.HasService("volumev3"); err != nil || ok {
		t.Fatalf("alias HasService %v %v", ok, err)
	}
	if !c.ServiceEnabled("load_balancer") || !c.ServiceEnabled("Load-Balancer") || c.ServiceEnabled("volumev3") || c.ServiceEnabled("custom") {
		t.Fatal("official defaults")
	}
	if _, err := c.HasServiceVersion(ctx, "network", "x"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("invalid version: %v", err)
	}

	flagged := f.connect(t, sdk.WithServiceEnabled("network", false), sdk.WithServiceEnabled("volumev3", true), sdk.WithServiceEnabled("custom-thing", true))
	f.reset()
	if ok, err := flagged.HasServiceVersion(ctx, "network", "2"); err != nil || ok {
		t.Fatalf("disabled network %v %v", ok, err)
	}
	if ok, err := flagged.HasService("network"); err != nil || ok {
		t.Fatalf("disabled network HasService %v %v", ok, err)
	}
	if calls := f.Calls(); len(calls) != 0 {
		t.Fatalf("disabled service looked up: %+v", calls)
	}
	if ok, err := flagged.HasServiceVersion(ctx, "volumev3", ""); err != nil || !ok {
		t.Fatalf("enabled alias %v %v", ok, err)
	}
	if !flagged.ServiceEnabled("custom_thing") {
		t.Fatal("custom flag")
	}
	if _, err := sdk.FromProvider(f.cloud.Provider, sdk.WithServiceEnabled(" ", true)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("empty type flag: %v", err)
	}
}

func writeServiceConfigCloud(t *testing.T, base, settings string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "clouds.yaml")
	body := "clouds:\n  dev:\n    region_name: RegionOne\n    auth:\n      auth_url: " + base + "/v3\n      username: alice\n      password: pw\n      user_domain_name: Default\n" + settings
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// clouds.yaml has_<service> flags and secgroup_source are read per cloud.
func TestCloudsYAMLServiceFlagsAndSecurityGroupSource(t *testing.T) {
	f := newDiscoveryFixture(t)
	ctx := context.Background()
	path := writeServiceConfigCloud(t, f.base, "    has_network: false\n    has_volumev3: true\n    secgroup_source: null\n")
	c, err := sdk.Connect(ctx, sdk.WithCloud("dev"), sdk.WithCloudFiles(path))
	if err != nil {
		t.Fatal(err)
	}
	if c.ServiceEnabled("network") || !c.ServiceEnabled("volumev3") || !c.ServiceEnabled("compute") {
		t.Fatal("flags not applied")
	}
	if ok, err := c.HasService("network"); err != nil || ok {
		t.Fatalf("has_network false: %v %v", ok, err)
	}
	if ok, err := c.AddServerSecurityGroups(ctx, "s1", "alpha"); ok || !errors.Is(err, network.ErrUnavailableSecurityGroups) {
		t.Fatalf("secgroup_source null: %v %v", ok, err)
	}
	// Explicit options win over clouds.yaml.
	c, err = sdk.Connect(ctx, sdk.WithCloud("dev"), sdk.WithCloudFiles(path), sdk.WithServiceEnabled("network", true), sdk.WithSecurityGroupSource("nova"))
	if err != nil {
		t.Fatal(err)
	}
	if !c.ServiceEnabled("network") {
		t.Fatal("option did not override has_network")
	}
	if _, err := c.SecurityGroupCloud(ctx); err != nil {
		t.Fatalf("nova source: %v", err)
	}
	for _, settings := range []string{"    has_network: 'false'\n", "    secgroup_source: 3\n"} {
		bad := writeServiceConfigCloud(t, f.base, settings)
		if _, err := sdk.Connect(ctx, sdk.WithCloud("dev"), sdk.WithCloudFiles(bad)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%q accepted: %v", settings, err)
		}
	}
}
