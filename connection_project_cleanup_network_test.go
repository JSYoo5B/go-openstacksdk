package openstack_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/projectcleanup"
)

// networkFixture serves Networking lists keyed by "collection?query", router
// reads and mutations. Every request is recorded; an unlisted read fails.
// status maps a recorded mutation event to its response code (-1 drops the
// connection without a response); the default is success.
type networkFixture struct {
	*cleanupCloud
	lists   map[string]string
	routers map[string]string
	status  map[string]int
}

func newNetworkFixture(t *testing.T, servers string, lists map[string]string) *networkFixture {
	t.Helper()
	f := &networkFixture{cleanupCloud: newCleanupCloud(t), lists: lists, routers: map[string]string{}, status: map[string]int{}}
	list := func(w http.ResponseWriter, r *http.Request, collection string) {
		if r.URL.Query().Has("all_tenants") || r.URL.Query().Has("all_projects") {
			t.Error("all-projects listing", r.URL)
		}
		key := collection
		if query := r.URL.Query().Encode(); query != "" {
			key += "?" + query
		}
		f.record("GET " + key)
		rows, ok := f.lists[key]
		if !ok {
			t.Error("unexpected list", key)
			http.Error(w, "unexpected", 500)
			return
		}
		testcloud.JSON(w, 200, `{"`+strings.ReplaceAll(path.Base(collection), "-", "_")+`":[`+rows+`]}`)
	}
	f.Mux.HandleFunc("GET /network/v2.0/{collection}", func(w http.ResponseWriter, r *http.Request) {
		list(w, r, r.PathValue("collection"))
	})
	f.Mux.HandleFunc("GET /network/v2.0/vpn/{collection}", func(w http.ResponseWriter, r *http.Request) {
		list(w, r, "vpn/"+r.PathValue("collection"))
	})
	f.Mux.HandleFunc("GET /network/v2.0/routers/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		f.record("GET routers/" + id)
		body, ok := f.routers[id]
		switch {
		case !ok:
			testcloud.JSON(w, 404, `{"NeutronError":{"type":"RouterNotFound"}}`)
		case strings.HasPrefix(body, "status:"):
			testcloud.JSON(w, 500, `{"NeutronError":{"message":"down"}}`)
		default:
			testcloud.JSON(w, 200, `{"router":`+body+`}`)
		}
	})
	mutate := func(w http.ResponseWriter, event string, success int, body string) {
		f.record(event)
		switch code := f.status[event]; code {
		case 0:
			if body == "" {
				w.WriteHeader(success)
				return
			}
			testcloud.JSON(w, success, body)
		case -1:
			conn, _, err := http.NewResponseController(w).Hijack()
			if err == nil {
				_ = conn.Close()
			}
		default:
			testcloud.JSON(w, code, `{"NeutronError":{"message":"rejected"}}`)
		}
	}
	f.Mux.HandleFunc("DELETE /network/v2.0/{collection}/{id}", func(w http.ResponseWriter, r *http.Request) {
		mutate(w, "DELETE "+r.PathValue("collection")+"/"+r.PathValue("id"), 204, "")
	})
	f.Mux.HandleFunc("DELETE /network/v2.0/vpn/{collection}/{id}", func(w http.ResponseWriter, r *http.Request) {
		mutate(w, "DELETE vpn/"+r.PathValue("collection")+"/"+r.PathValue("id"), 204, "")
	})
	f.Mux.HandleFunc("PUT /network/v2.0/routers/{id}/{action}", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mutate(w, "PUT routers/"+r.PathValue("id")+"/"+r.PathValue("action")+" "+string(body), 200, `{"id":"`+r.PathValue("id")+`"}`)
	})
	f.Mux.HandleFunc("DELETE /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.record("delete " + r.PathValue("id"))
		w.WriteHeader(204)
	})
	f.Mux.HandleFunc("GET /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.record("get " + r.PathValue("id"))
		testcloud.JSON(w, 404, `{"itemNotFound":{"code":404}}`)
	})
	f.serveCompute(t, servers, ``)
	return f
}

func networkRef(kind, id string) projectcleanup.Ref {
	return projectcleanup.Ref{Service: "network", Kind: kind, ID: id}
}

const vpnProvider = `{"service_type":"VPN","name":"strongswan","default":true}`

// fullNetworkLists covers every network kind and Python's conditions.
func fullNetworkLists() map[string]string {
	return map[string]string{
		"service-providers?service_type=VPN": vpnProvider,
		"vpn/ipsec-site-connections?project_id=project": `{"id":"c1","project_id":"project","vpnservice_id":"v1","ikepolicy_id":"i1","ipsecpolicy_id":"p1","local_ep_group_id":"e1","peer_ep_group_id":"e2"},` +
			`{"id":"c2","project_id":"other","vpnservice_id":"v2"}`,
		"vpn/vpnservices?project_id=project":     `{"id":"v1","project_id":"project"},{"id":"v2","project_id":"project"}`,
		"vpn/endpoint-groups?project_id=project": `{"id":"e1"},{"id":"e2"}`,
		"vpn/ikepolicies?project_id=project":     `{"id":"i1"}`,
		"vpn/ipsecpolicies?project_id=project":   `{"id":"p1"},{"id":"p2"}`,
		"floatingips?project_id=project": `{"id":"f1","floating_ip_address":"203.0.113.1","port_id":"p-vm","project_id":"project"},` +
			`{"id":"f2","port_id":null,"project_id":"project"},{"id":"f3","port_id":null,"tenant_id":"other"}`,
		"security-groups?project_id=project": `{"id":"sg-default","name":"default"},{"id":"sg1","name":"web"}`,
		"networks?project_id=project": `{"id":"net-ok","name":"app","project_id":"project","shared":false,"router:external":false},` +
			`{"id":"net-busy","project_id":"project"},{"id":"net-shared","project_id":"project","shared":true},` +
			`{"id":"net-ext","project_id":"project","router:external":true},{"id":"net-other","project_id":"other"}`,
		"ports?network_id=net-ok&project_id=project": `{"id":"p-dhcp","device_owner":"network:dhcp","device_id":"dhcp1"},{"id":"p-free","device_owner":""},` +
			`{"id":"p-null","device_owner":null},{"id":"p-if","device_owner":"network:router_interface","device_id":"r1"},` +
			`{"id":"p-vm","device_owner":"compute:nova","device_id":"s1"}`,
		"subnets?network_id=net-ok&project_id=project": `{"id":"sn1","name":"app-v4"}`,
		"ports?network_id=net-busy&project_id=project": `{"id":"p-busy","device_owner":"compute:nova","device_id":"s-kept"}`,
		"ports?device_id=r1":                           `{"id":"p-if","device_owner":"network:router_interface"},{"id":"p-gw","device_owner":"network:router_gateway","project_id":""}`,
		"routers?project_id=project":                   `{"id":"r1","project_id":"project"},{"id":"r2","name":"spare","project_id":"project"},{"id":"r3","project_id":"project"}`,
		"ports?device_id=r2":                           ``,
		"ports?device_id=r3":                           `{"id":"p-r3","device_owner":"network:router_interface"}`,
	}
}

const routerR1 = `{"id":"r1","name":"edge","project_id":"project","routes":[{"destination":"10.1.0.0/24","nexthop":"10.0.0.5"}]}`

func newFullNetworkFixture(t *testing.T) *networkFixture {
	f := newNetworkFixture(t, `{"id":"s1","tenant_id":"project"}`, fullNetworkLists())
	f.routers["r1"] = routerR1
	return f
}

var fullNetworkDiscovery = []string{
	"list servers", "list groups",
	"GET service-providers?service_type=VPN",
	"GET vpn/ipsec-site-connections?project_id=project",
	"GET vpn/vpnservices?project_id=project",
	"GET vpn/endpoint-groups?project_id=project",
	"GET vpn/ikepolicies?project_id=project",
	"GET vpn/ipsecpolicies?project_id=project",
	"GET floatingips?project_id=project",
	"GET security-groups?project_id=project",
	"GET networks?project_id=project",
	"GET ports?network_id=net-ok&project_id=project",
	"GET routers/r1",
	"GET subnets?network_id=net-ok&project_id=project",
	"GET ports?network_id=net-busy&project_id=project",
	"GET ports?device_id=r1",
	"GET routers?project_id=project",
	"GET ports?device_id=r2",
	"GET ports?device_id=r3",
}

func TestProjectCleanupNetworkPlanFollowsPythonConditions(t *testing.T) {
	f := newFullNetworkFixture(t)
	plan, err := f.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"server:s1",
		"vpn_ipsec_site_connection:c1", "vpn_service:v1", "vpn_endpoint_group:e1", "vpn_endpoint_group:e2", "vpn_ike_policy:i1",
		"vpn_ipsec_policy:p1", "vpn_ipsec_policy:p2", "floating_ip:f1", "floating_ip:f2", "security_group:sg1",
		"router_routes:r1", "router_interface:p-if", "router:r1", "router:r2",
		"port:p-free", "port:p-null", "port:p-vm", "subnet:sn1", "network:net-ok"}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, want) {
		t.Fatalf("candidates\n got %v\nwant %v", got, want)
	}
	for _, stage := range plan.Stages {
		if stage.Service == "network" && stage.WaitForAbsence {
			t.Errorf("network stage %s waits", stage.Kind)
		}
	}
	c1 := networkRef("vpn_ipsec_site_connection", "c1")
	iface := projectcleanup.Ref{Service: "network", Kind: "router_interface", ID: "p-if", Parent: "r1"}
	routes := networkRef("router_routes", "r1")
	ports := []projectcleanup.Ref{iface, networkRef("port", "p-free"), networkRef("port", "p-null"), networkRef("port", "p-vm")}
	deps := map[string][]projectcleanup.Ref{
		"vpn_service:v1": {c1}, "vpn_endpoint_group:e1": {c1}, "vpn_endpoint_group:e2": {c1}, "vpn_ike_policy:i1": {c1}, "vpn_ipsec_policy:p1": {c1},
		"vpn_ipsec_policy:p2": nil, "floating_ip:f1": nil, "router_routes:r1": nil,
		"router_interface:p-if": {routes}, "router:r1": {routes, iface}, "router:r2": nil,
		"port:p-free": nil, "port:p-vm": {{Service: "compute", Kind: "server", ID: "s1"}},
		"subnet:sn1": ports, "network:net-ok": append(slices.Clone(ports), networkRef("subnet", "sn1")),
	}
	for key, want := range deps {
		kind, id, _ := strings.Cut(key, ":")
		if got := candidateByID(plan, kind, id).DependsOn; !slices.Equal(got, want) {
			t.Errorf("%s depends on %v, want %v", key, got, want)
		}
	}
	if c := candidateByID(plan, "router_interface", "p-if"); c.Parent != "r1" || !strings.Contains(c.Reason, "net-ok") {
		t.Errorf("interface %+v", c)
	}
	if c := candidateByID(plan, "router_routes", "r1"); c.Name != "edge" || !strings.Contains(string(c.Raw), `"nexthop":"10.0.0.5"`) {
		t.Errorf("routes %+v", c)
	}
	if c := candidateByID(plan, "router", "r2"); !strings.Contains(c.Reason, "no ports") {
		t.Errorf("unattached router %+v", c)
	}
	if c := candidateByID(plan, "floating_ip", "f1"); c.Name != "203.0.113.1" {
		t.Errorf("floating ip name %q", c.Name)
	}
	reasons := exclusionReasons(plan)
	for key, want := range map[string]string{
		"vpn_ipsec_site_connection:c2": "owned by project other",
		"vpn_service:v2":               "not cleanup candidates: c2",
		"floating_ip:f3":               "owned by project other (tenant_id)",
		"security_group:sg-default":    "default security group",
		"network:net-busy":             "p-busy (compute:nova s-kept)",
		"network:net-shared":           "network is shared",
		"network:net-ext":              "network is external",
		"network:net-other":            "owned by project other",
		"router:r3":                    "router has 1 ports",
	} {
		if !strings.Contains(reasons[key], want) {
			t.Errorf("%s: %q, want %q", key, reasons[key], want)
		}
	}
	if len(reasons) != 9 {
		t.Errorf("exclusions %v", reasons)
	}
	for _, u := range plan.Unavailable {
		if u.Service == "network" {
			t.Errorf("network reported unavailable: %+v", u)
		}
	}
	if got := f.events(); !slices.Equal(got, fullNetworkDiscovery) {
		t.Fatalf("requests\n got %v\nwant %v", got, fullNetworkDiscovery)
	}
}

func TestProjectCleanupNetworkExecutesInPythonOrder(t *testing.T) {
	f := newFullNetworkFixture(t)
	plan, err := f.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	result, err := f.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithDeletionWait(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	want := append(slices.Clone(fullNetworkDiscovery),
		"delete s1", "get s1",
		"DELETE vpn/ipsec-site-connections/c1", "DELETE vpn/vpnservices/v1", "DELETE vpn/endpoint-groups/e1", "DELETE vpn/endpoint-groups/e2",
		"DELETE vpn/ikepolicies/i1", "DELETE vpn/ipsecpolicies/p1", "DELETE vpn/ipsecpolicies/p2",
		"DELETE floatingips/f1", "DELETE floatingips/f2", "DELETE security-groups/sg1",
		`PUT routers/r1/remove_extraroutes {"router":{"routes":[{"destination":"10.1.0.0/24","nexthop":"10.0.0.5"}]}}`,
		`PUT routers/r1/remove_router_interface {"port_id":"p-if"}`,
		"DELETE routers/r1", "DELETE routers/r2",
		"DELETE ports/p-free", "DELETE ports/p-null", "DELETE ports/p-vm", "DELETE subnets/sn1", "DELETE networks/net-ok")
	if got := f.events(); !slices.Equal(got, want) {
		t.Fatalf("requests\n got %v\nwant %v", got, want)
	}
	if result.Count(projectcleanup.Deleted) != 1 || result.Count(projectcleanup.Accepted) != len(result.Outcomes)-1 {
		t.Fatalf("outcomes %+v", result.Outcomes)
	}
}

func TestProjectCleanupNetworkOutcomesBlockDependents(t *testing.T) {
	f := newFullNetworkFixture(t)
	f.status["DELETE vpn/ipsec-site-connections/c1"] = 500
	f.status["DELETE security-groups/sg1"] = -1
	f.status[`PUT routers/r1/remove_router_interface {"port_id":"p-if"}`] = 409
	f.status["DELETE ports/p-free"] = 404
	f.status[`PUT routers/r1/remove_extraroutes {"router":{"routes":[{"destination":"10.1.0.0/24","nexthop":"10.0.0.5"}]}}`] = 0
	plan, err := f.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	result, err := f.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithDeletionWait(10*time.Second))
	if !errors.Is(err, projectcleanup.ErrIncomplete) {
		t.Fatalf("err %v", err)
	}
	states := map[string]projectcleanup.Outcome{}
	for _, o := range result.Outcomes {
		states[o.Candidate.Kind+":"+o.Candidate.ID] = o
	}
	for key, state := range map[string]projectcleanup.State{
		"vpn_ipsec_site_connection:c1": projectcleanup.Failed, "vpn_service:v1": projectcleanup.NotAttempted,
		"vpn_endpoint_group:e1": projectcleanup.NotAttempted, "vpn_ike_policy:i1": projectcleanup.NotAttempted,
		"vpn_ipsec_policy:p1": projectcleanup.NotAttempted, "vpn_ipsec_policy:p2": projectcleanup.Accepted,
		"security_group:sg1": projectcleanup.Uncertain, "router_routes:r1": projectcleanup.Accepted,
		"router_interface:p-if": projectcleanup.Failed, "router:r1": projectcleanup.NotAttempted, "router:r2": projectcleanup.Accepted,
		"port:p-free": projectcleanup.AlreadyMissing, "port:p-vm": projectcleanup.Accepted,
		"subnet:sn1": projectcleanup.NotAttempted, "network:net-ok": projectcleanup.NotAttempted,
	} {
		if states[key].State != state {
			t.Errorf("%s: %s, want %s (%s %v)", key, states[key].State, state, states[key].Reason, states[key].Err)
		}
	}
	if iface := states["router_interface:p-if"]; iface.StatusCode != 409 || !strings.Contains(string(iface.Body), "rejected") {
		t.Errorf("interface evidence %+v", iface)
	}
	if !strings.Contains(states["subnet:sn1"].Reason, "router_interface") {
		t.Errorf("subnet reason %q", states["subnet:sn1"].Reason)
	}
	for _, event := range f.events() {
		for _, never := range []string{"vpnservices/v1", "ipsecpolicies/p1", "routers/r1", "subnets/sn1", "networks/net-ok"} {
			if strings.HasPrefix(event, "DELETE") && strings.Contains(event, never) {
				t.Errorf("dependent sent: %s", event)
			}
		}
	}
}

func TestProjectCleanupNetworkTimeFiltersFollowPython(t *testing.T) {
	f := newNetworkFixture(t, ``, map[string]string{
		"service-providers?service_type=VPN":            vpnProvider,
		"vpn/ipsec-site-connections?project_id=project": `{"id":"c1","vpnservice_id":"v1"}`,
		"vpn/vpnservices?project_id=project":            `{"id":"v1"}`,
		"vpn/endpoint-groups?project_id=project":        ``,
		"vpn/ikepolicies?project_id=project":            `{"id":"i1"}`,
		"vpn/ipsecpolicies?project_id=project":          ``,
		"floatingips?project_id=project":                `{"id":"f-att","port_id":"p1","created_at":"2025-01-01T00:00:00Z"},{"id":"f-new","port_id":null,"created_at":"2026-06-01T00:00:00Z"}`,
		"security-groups?project_id=project":            `{"id":"sg-old","created_at":"2025-01-01T00:00:00Z"},{"id":"sg-new","created_at":"2026-06-01T00:00:00Z"}`,
		"networks?project_id=project":                   `{"id":"net-old","created_at":"2025-01-01T00:00:00Z"},{"id":"net-new","created_at":"2026-06-01T00:00:00Z"}`,
		"ports?network_id=net-old&project_id=project":   ``,
		"ports?network_id=net-new&project_id=project":   ``,
		"subnets?network_id=net-old&project_id=project": `{"id":"sn","created_at":"2026-06-01T00:00:00Z"}`,
		"routers?project_id=project":                    `{"id":"r-new","created_at":"2026-06-01T00:00:00Z"}`,
		"ports?device_id=r-new":                         ``,
	})
	plan, err := f.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithCreatedBefore(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"floating_ip:f-new", "security_group:sg-old", "router:r-new", "subnet:sn", "network:net-old"}) {
		t.Fatalf("candidates %v", got)
	}
	reasons := exclusionReasons(plan)
	for key, want := range map[string]string{
		"vpn_ipsec_site_connection:c1": "created_at is not available for network.vpn_ipsec_site_connection",
		"vpn_service:v1":               "not cleanup candidates: c1",
		"vpn_ike_policy:i1":            "created_at is not available for network.vpn_ike_policy",
		"floating_ip:f-att":            "attached to port p1",
		"security_group:sg-new":        "after the cutoff",
		"network:net-new":              "after the cutoff",
	} {
		if !strings.Contains(reasons[key], want) {
			t.Errorf("%s: %q, want %q", key, reasons[key], want)
		}
	}
	if c := candidateByID(plan, "floating_ip", "f-new"); !strings.Contains(c.Reason, "time filters do not apply") {
		t.Errorf("floating ip reason %q", c.Reason)
	}
	if slices.Contains(f.events(), "GET subnets?network_id=net-new&project_id=project") {
		t.Error("subnets of an unselected network were listed")
	}
}

func TestProjectCleanupNetworkSelectorAndSkipNames(t *testing.T) {
	lists := map[string]string{
		"service-providers?service_type=VPN":        ``,
		"floatingips?project_id=project":            `{"id":"f-keep","port_id":"p1"},{"id":"f-drop"}`,
		"security-groups?project_id=project":        ``,
		"networks?project_id=project":               `{"id":"net-x"}`,
		"ports?network_id=net-x&project_id=project": ``,
		"routers?project_id=project":                `{"id":"r0"}`,
		"ports?device_id=r0":                        ``,
	}
	f := newNetworkFixture(t, ``, lists)
	ctx := context.Background()
	var seen []string
	plan, err := f.conn.PlanProjectCleanup(ctx, projectcleanup.WithSelector(func(c projectcleanup.Candidate) (bool, error) {
		seen = append(seen, c.Kind+":"+c.ID)
		return c.ID == "f-drop", nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	// Without time filters an attached floating IP is not kept by Python's
	// rule; the selector still decides. Routers without ports are not
	// evaluated, as in Python.
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"floating_ip:f-drop", "router:r0"}) {
		t.Fatalf("candidates %v", got)
	}
	if !slices.Equal(seen, []string{"floating_ip:f-keep", "floating_ip:f-drop", "network:net-x"}) {
		t.Fatalf("selector calls %v", seen)
	}
	reasons := exclusionReasons(plan)
	if reasons["floating_ip:f-keep"] != "rejected by selector" || reasons["network:net-x"] != "rejected by selector" {
		t.Fatalf("exclusions %v", reasons)
	}
	if slices.Contains(f.events(), "GET vpn/ipsec-site-connections?project_id=project") {
		t.Fatal("VPN listed without a VPN service provider")
	}
	boom := errors.New("boom")
	if _, err := f.conn.PlanProjectCleanup(ctx, projectcleanup.WithSelector(func(c projectcleanup.Candidate) (bool, error) {
		if c.Kind == "floating_ip" {
			return false, boom
		}
		return true, nil
	})); !errors.Is(err, boom) {
		t.Fatalf("selector error %v", err)
	}

	before := len(f.events())
	plan, err = f.conn.PlanProjectCleanup(ctx, projectcleanup.WithAllResources(), projectcleanup.WithSkip("network.port"))
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"floating_ip:f-keep", "floating_ip:f-drop", "router:r0"}) {
		t.Fatalf("skip port: %v", got)
	}
	wantEvents := []string{"list servers", "list groups", "GET service-providers?service_type=VPN", "GET floatingips?project_id=project",
		"GET security-groups?project_id=project", "GET routers?project_id=project", "GET ports?device_id=r0"}
	if got := f.events()[before:]; !slices.Equal(got, wantEvents) {
		t.Fatalf("skip port requests %v", got)
	}
	before = len(f.events())
	plan, err = f.conn.PlanProjectCleanup(ctx, projectcleanup.WithAllResources(),
		projectcleanup.WithSkip("network.router", "network.floating_ip", "network.security_group"))
	if err != nil || len(plan.Candidates()) != 0 {
		t.Fatalf("skip all: %v %v", refIDs(plan.Candidates()), err)
	}
	if got := f.events()[before:]; !slices.Equal(got, []string{"list servers", "list groups", "GET service-providers?service_type=VPN"}) {
		t.Fatalf("skip all requests %v", got)
	}
}

func TestProjectCleanupNetworkRouterConditions(t *testing.T) {
	f := newNetworkFixture(t, ``, map[string]string{
		"service-providers?service_type=VPN":          ``,
		"floatingips?project_id=project":              ``,
		"security-groups?project_id=project":          ``,
		"networks?project_id=project":                 `{"id":"net-a"},{"id":"net-b"},{"id":"net-c"}`,
		"ports?network_id=net-a&project_id=project":   `{"id":"pa","device_owner":"network:router_interface_distributed","device_id":"r-foreign"}`,
		"ports?network_id=net-b&project_id=project":   `{"id":"pb","device_owner":"network:ha_router_replicated_interface","device_id":"r-missing"}`,
		"ports?network_id=net-c&project_id=project":   `{"id":"pc","device_owner":"network:router_interface","device_id":"r-multi"}`,
		"subnets?network_id=net-c&project_id=project": ``,
		"ports?device_id=r-multi":                     `{"id":"pc","device_owner":"network:router_interface"},{"id":"px","device_owner":"network:router_interface"},{"id":"ph","device_owner":"network:router_ha_interface"}`,
		"routers?project_id=project":                  `{"id":"r-multi"}`,
	})
	f.routers["r-foreign"] = `{"id":"r-foreign","project_id":"other"}`
	f.routers["r-multi"] = `{"id":"r-multi","routes":[]}`
	plan, err := f.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	// The interface and network of net-c are cleaned; the router keeps an
	// interface on another network and has no routes to remove.
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"router_interface:pc", "network:net-c"}) {
		t.Fatalf("candidates %v", got)
	}
	reasons := exclusionReasons(plan)
	for key, want := range map[string]string{
		"network:net-a":  "router r-foreign of interface port pa is owned by project other",
		"network:net-b":  "router r-missing of interface port pb was not found",
		"router:r-multi": "router keeps ports that are not removed by cleanup: px (network:router_interface)",
	} {
		if !strings.Contains(reasons[key], want) {
			t.Errorf("%s: %q, want %q", key, reasons[key], want)
		}
	}
	if len(reasons) != 3 {
		t.Errorf("exclusions %v", reasons)
	}

	// A failed router read is a discovery error of the network service.
	f.routers["r-multi"] = "status:500"
	plan, err = f.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	var serviceErr *projectcleanup.ServiceError
	if !errors.As(err, &serviceErr) || serviceErr.Service != "network" || !strings.Contains(err.Error(), "get router r-multi") {
		t.Fatalf("router read failure: %v", err)
	}
	if len(plan.Candidates()) != 0 || len(plan.Excluded) != 0 {
		t.Fatalf("partial network plan %+v", plan)
	}
}
