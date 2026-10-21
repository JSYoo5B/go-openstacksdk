package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"

	"github.com/gophercloud/gophercloud/v2"
)

// r2MiscKeystone serves token requests with a catalog and records each body.
type r2MiscKeystone struct {
	mu     sync.Mutex
	bodies []map[string]any
}

func (k *r2MiscKeystone) count() int {
	k.mu.Lock()
	defer k.mu.Unlock()
	return len(k.bodies)
}

func (k *r2MiscKeystone) body(i int) map[string]any {
	k.mu.Lock()
	defer k.mu.Unlock()
	return k.bodies[i]
}

func r2MiscNested(t *testing.T, value any, path ...string) any {
	t.Helper()
	for _, key := range path {
		m, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		value = m[key]
	}
	return value
}

func newR2MiscKeystone(t *testing.T, cloud *testcloud.Cloud) *r2MiscKeystone {
	t.Helper()
	k := &r2MiscKeystone{}
	cloud.Mux.HandleFunc("POST /v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		k.mu.Lock()
		k.bodies = append(k.bodies, body)
		n := len(k.bodies)
		k.mu.Unlock()
		w.Header().Set("X-Subject-Token", fmt.Sprintf("token-%d", n))
		testcloud.JSON(w, http.StatusCreated, fmt.Sprintf(`{"token":{"expires_at":"2099-01-01T00:00:00Z","user":{"id":"user-%d","name":"alice"},"catalog":[
{"id":"s1","type":"compute","name":"nova","endpoints":[
 {"id":"e1","interface":"public","region":"RegionOne","region_id":"RegionOne","url":"%s/compute/v2.1/pub"},
 {"id":"e2","interface":"internal","region":"RegionOne","region_id":"RegionOne","url":"%s/compute/v2.1/int"},
 {"id":"e3","interface":"public","region":"RegionTwo","region_id":"RegionTwo","url":"%s/compute/v2.1/pub2"}]},
{"id":"s2","type":"network","name":"neutron","endpoints":[
 {"id":"e4","interface":"public","region":"RegionOne","region_id":"RegionOne","url":"%s/network"}]}]}}`, n, cloud.Server.URL, cloud.Server.URL, cloud.Server.URL, cloud.Server.URL))
	})
	return k
}

func r2MiscConnect(t *testing.T, cloud *testcloud.Cloud, auth gophercloud.AuthOptions, opts ...sdk.ConnectionOption) *sdk.Connection {
	t.Helper()
	auth.IdentityEndpoint = cloud.Server.URL + "/v3"
	opts = append([]sdk.ConnectionOption{sdk.WithAuth(auth), sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute/v2.1/project")}, opts...)
	c, err := sdk.Connect(context.Background(), opts...)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestR2MiscGlobalRequestHeaderOnDerivedConnectionOnly(t *testing.T) {
	cloud := testcloud.New(t)
	var mu sync.Mutex
	seen := map[string][]string{}
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen[r.PathValue("id")] = []string{r.Header.Get("X-Openstack-Request-Id"), r.Header.Get("X-Auth-Token")}
		mu.Unlock()
		testcloud.JSON(w, 200, `{"server":{"id":"s","name":"n"}}`)
	})
	cloud.Mux.HandleFunc("GET /network/v2.0/networks/{id}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen["net-"+r.PathValue("id")] = []string{r.Header.Get("X-Openstack-Request-Id")}
		mu.Unlock()
		testcloud.JSON(w, 200, `{"network":{"id":"n","name":"n"}}`)
	})
	parent, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute/v2.1/project"), sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	derived, err := parent.GlobalRequest("req-0123")
	if err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]*sdk.Connection{"parent": parent, "derived": derived} {
		compute, err := c.Compute(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compute.Servers.Get(ctx, name); err != nil {
			t.Fatal(err)
		}
		net, err := c.Network(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := net.Networks.Get(ctx, name); err != nil {
			t.Fatal(err)
		}
	}
	if got := seen["parent"]; got[0] != "" || got[1] != "test-token" {
		t.Fatalf("parent request headers %v", got)
	}
	if got := seen["derived"]; got[0] != "req-0123" || got[1] != "test-token" {
		t.Fatalf("derived request headers %v", got)
	}
	if seen["net-parent"][0] != "" || seen["net-derived"][0] != "req-0123" {
		t.Fatalf("network headers %v %v", seen["net-parent"], seen["net-derived"])
	}
	again, err := derived.GlobalRequest("req-other")
	if err != nil {
		t.Fatal(err)
	}
	compute, _ := again.Compute(ctx)
	if _, err := compute.Servers.Get(ctx, "again"); err != nil {
		t.Fatal(err)
	}
	if seen["again"][0] != "req-other" {
		t.Fatalf("re-derived header %v", seen["again"])
	}
}

func TestR2MiscGlobalRequestRejectsUnsafeIDs(t *testing.T) {
	cloud := testcloud.New(t)
	c, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"", "req 1", "req-1\r\nX-Evil: 1", "req-é", "req-\t1"} {
		if _, err := c.GlobalRequest(id); err == nil {
			t.Errorf("id %q accepted", id)
		}
	}
	var nilConn *sdk.Connection
	if _, err := nilConn.GlobalRequest("req-1"); err == nil {
		t.Error("nil connection accepted")
	}
}

func TestR2MiscConnectAsOverridesAuthAndSharesNothing(t *testing.T) {
	cloud := testcloud.New(t)
	keystone := newR2MiscKeystone(t, cloud)
	var mu sync.Mutex
	tokens := []string{}
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		tokens = append(tokens, r.Header.Get("X-Auth-Token")+"|"+r.Header.Get("X-Openstack-Request-Id"))
		mu.Unlock()
		testcloud.JSON(w, 200, `{"server":{"id":"s"}}`)
	})
	parent := r2MiscConnect(t, cloud, gophercloud.AuthOptions{Username: "alice", Password: "pw", DomainName: "Default", Scope: &gophercloud.AuthScope{ProjectName: "p1", DomainName: "Default"}}, sdk.WithRegion("RegionOne"))
	ctx := context.Background()

	child, err := parent.ConnectAs(ctx, sdk.AsProjectName("p2"))
	if err != nil {
		t.Fatal(err)
	}
	body := keystone.body(1)
	if got := r2MiscNested(t, body, "auth", "scope", "project", "name"); got != "p2" {
		t.Fatalf("scope project %v", got)
	}
	if got := r2MiscNested(t, body, "auth", "scope", "project", "domain", "name"); got != "Default" {
		t.Fatalf("project domain inherited as %v", got)
	}
	if got := r2MiscNested(t, body, "auth", "identity", "password", "user", "name"); got != "alice" {
		t.Fatalf("user %v", got)
	}

	if _, err := parent.ConnectAs(ctx, sdk.AsUserID("uid"), sdk.AsPassword("pw2")); err != nil {
		t.Fatal(err)
	}
	user := r2MiscNested(t, keystone.body(2), "auth", "identity", "password", "user").(map[string]any)
	if user["id"] != "uid" || user["name"] != nil || user["password"] != "pw2" {
		t.Fatalf("user override %v", user)
	}

	if _, err := parent.ConnectAs(ctx, sdk.AsProjectID("pid"), sdk.AsProjectName("ignored")); err != nil {
		t.Fatal(err)
	}
	if got := r2MiscNested(t, keystone.body(3), "auth", "scope", "project", "id"); got != "pid" {
		t.Fatalf("project id scope %v", got)
	}
	if _, err := parent.ConnectAsProject(ctx, sdk.ProjectRef{Name: "p3", DomainID: "dom"}); err != nil {
		t.Fatal(err)
	}
	if got := r2MiscNested(t, keystone.body(4), "auth", "scope", "project", "domain", "id"); got != "dom" {
		t.Fatalf("project domain id %v", got)
	}
	if got := r2MiscNested(t, keystone.body(4), "auth", "scope", "project", "domain", "name"); got != nil {
		t.Fatalf("inherited project domain name kept: %v", got)
	}

	// The derived Connection authenticated separately, kept the endpoint
	// override and region, and sends its own token without request IDs.
	for name, c := range map[string]*sdk.Connection{"parent": parent, "child": child} {
		compute, err := c.Compute(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := compute.Servers.Get(ctx, name); err != nil {
			t.Fatal(err)
		}
	}
	if len(tokens) != 2 || tokens[0] != "token-1|" || tokens[1] != "token-2|" {
		t.Fatalf("tokens %v", tokens)
	}
	if parent.AuthToken() != "token-1" || child.AuthToken() != "token-2" {
		t.Fatalf("auth tokens %q %q", parent.AuthToken(), child.AuthToken())
	}
	location, err := child.CurrentLocation()
	if err != nil {
		t.Fatal(err)
	}
	if location.RegionName == nil || *location.RegionName != "RegionOne" || location.Project.Name == nil || *location.Project.Name != "p2" {
		t.Fatalf("location %+v", location)
	}
	if _, ok, err := child.EndpointFor("network"); err != nil || !ok {
		t.Fatalf("child catalog %v %v", ok, err)
	}
}

func TestR2MiscConnectAsInvalidInputsMakeNoRequest(t *testing.T) {
	cloud := testcloud.New(t)
	keystone := newR2MiscKeystone(t, cloud)
	parent := r2MiscConnect(t, cloud, gophercloud.AuthOptions{Username: "alice", Password: "pw", DomainName: "Default", Scope: &gophercloud.AuthScope{ProjectID: "pid"}})
	ctx := context.Background()
	if _, err := parent.ConnectAs(ctx, sdk.AsProjectDomainName("d")); err == nil {
		t.Error("project domain without a project by name accepted")
	}
	if _, err := parent.ConnectAs(ctx, sdk.AsProjectName("")); err == nil {
		t.Error("empty project accepted")
	}
	if _, err := parent.ConnectAs(ctx, nil); err == nil {
		t.Error("nil option accepted")
	}
	if _, err := parent.ConnectAs(ctx, sdk.AsAuthOptions(nil)); err == nil {
		t.Error("nil edit accepted")
	}
	if _, err := parent.ConnectAsProject(ctx, sdk.ProjectRef{}); err == nil {
		t.Error("empty project ref accepted")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := parent.ConnectAs(canceled, sdk.AsPassword("x")); !errors.Is(err, context.Canceled) {
		t.Errorf("canceled: %v", err)
	}
	if keystone.count() != 1 {
		t.Fatalf("token requests %d", keystone.count())
	}
	adopted, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adopted.ConnectAs(ctx, sdk.AsPassword("x")); !errors.Is(err, sdk.ErrNoRecordedAuth) {
		t.Errorf("adopted provider: %v", err)
	}
	// An authentication failure wraps the native cause and is not retried.
	cloud.Mux.HandleFunc("POST /v3/other/auth/tokens", func(w http.ResponseWriter, r *http.Request) {})
	edited := sdk.AsAuthOptions(func(a *gophercloud.AuthOptions) error {
		a.IdentityEndpoint = cloud.Server.URL + "/missing/"
		return nil
	})
	if _, err := parent.ConnectAs(ctx, edited); err == nil {
		t.Error("authentication against a missing endpoint succeeded")
	}
}

func TestR2MiscConnectAsDoesNotInheritGlobalRequestID(t *testing.T) {
	cloud := testcloud.New(t)
	newR2MiscKeystone(t, cloud)
	var mu sync.Mutex
	var ids []string
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		ids = append(ids, r.Header.Get("X-Openstack-Request-Id"))
		mu.Unlock()
		testcloud.JSON(w, 200, `{"server":{"id":"s"}}`)
	})
	parent := r2MiscConnect(t, cloud, gophercloud.AuthOptions{Username: "alice", Password: "pw", DomainName: "Default"})
	global, err := parent.GlobalRequest("req-keep")
	if err != nil {
		t.Fatal(err)
	}
	// A GlobalRequest Connection keeps its recorded authentication.
	child, err := global.ConnectAs(context.Background(), sdk.AsPassword("pw2"))
	if err != nil {
		t.Fatal(err)
	}
	compute, _ := child.Compute(context.Background())
	if _, err := compute.Servers.Get(context.Background(), "x"); err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "" {
		t.Fatalf("request ids %v", ids)
	}
}
