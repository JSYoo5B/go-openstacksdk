package openstack_test

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/projectcleanup"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const cleanupLoadBalancerPath = "/load-balancer/v2.0/lbaas/loadbalancers"

func init() {
	cleanupEmptyLists[cleanupLoadBalancerPath] = "loadbalancers"
}

// serveLoadBalancers lists the given rows. Python sends no query, so any
// query (including project_id or an all-projects flag) fails the test.
func (c *cleanupCloud) serveLoadBalancers(t *testing.T, rows string) {
	c.Mux.HandleFunc("GET "+cleanupLoadBalancerPath, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Error("load balancer list query", r.URL)
		}
		c.record("list load balancers")
		testcloud.JSON(w, 200, `{"loadbalancers":[`+rows+`],"loadbalancers_links":[]}`)
	})
}

// serveLoadBalancerMutations answers DELETE and GET on loadbalancers/{id}.
// deletes maps an ID to the DELETE status (-1 drops the connection); states
// is the sequence of provisioning_status values returned by GET, 404 when
// exhausted, with the last state repeating.
func (c *cleanupCloud) serveLoadBalancerMutations(t *testing.T, deletes map[string]int, states map[string][]string) {
	var mu sync.Mutex
	sent := map[string]bool{}
	c.Mux.HandleFunc("DELETE "+cleanupLoadBalancerPath+"/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		c.record("DELETE " + id + "?" + r.URL.RawQuery)
		mu.Lock()
		if sent[id] {
			t.Error("DELETE sent twice", id)
		}
		sent[id] = true
		mu.Unlock()
		switch code := deletes[id]; code {
		case 0, 204:
			w.WriteHeader(204)
		case -1:
			conn, _, err := http.NewResponseController(w).Hijack()
			if err == nil {
				_ = conn.Close()
			}
		default:
			w.Header().Set("X-Openstack-Request-Id", "req-"+id)
			testcloud.JSON(w, code, `{"faultcode":"Client","faultstring":"load balancer `+id+`","code":`+strconv.Itoa(code)+`}`)
		}
	})
	c.Mux.HandleFunc("GET "+cleanupLoadBalancerPath+"/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		c.record("GET " + id)
		mu.Lock()
		queue := states[id]
		var state string
		if len(queue) > 0 {
			state, states[id] = queue[0], queue[1:]
			if len(queue) == 1 {
				states[id] = queue
			}
		}
		mu.Unlock()
		if state == "" {
			testcloud.JSON(w, 404, `{"faultcode":"Client","faultstring":"Load Balancer `+id+` not found.","code":404}`)
			return
		}
		testcloud.JSON(w, 200, `{"loadbalancer":{"id":"`+id+`","project_id":"project","provisioning_status":"`+state+`"}}`)
	})
}

const cleanupLoadBalancerRows = `{"id":"lb-a","name":"web","project_id":"project","provisioning_status":"ACTIVE","created_at":"2025-01-01T00:00:00","updated_at":null},` +
	`{"id":"lb-b","name":"db","project_id":"project","tenant_id":"project","provisioning_status":"ERROR","created_at":"2025-02-01T00:00:00","updated_at":"2025-03-01T00:00:00"},` +
	`{"id":"lb-foreign","name":"x","project_id":"other"},` +
	`{"id":"lb-tenant","tenant_id":"other"},` +
	`{"id":"lb-unowned","name":"y"},` +
	`{"name":"no-id","project_id":"project"}`

func TestProjectCleanupLoadBalancerPlansBeforeComputeAndNetwork(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveStacks(t, `{"id":"st-a","stack_name":"web"}`)
	cloud.serveLoadBalancers(t, cleanupLoadBalancerRows)
	cloud.serveCompute(t, `{"id":"s1"}`, ``)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"stack:st-a", "load_balancer:lb-a", "load_balancer:lb-b", "server:s1"}) {
		t.Fatalf("candidates %v", got)
	}
	if got := stageSummary(plan); !slices.Equal(got, []string{"stack+wait", "load_balancer+wait", "server+wait"}) {
		t.Fatalf("stages %v", got)
	}
	web := candidateByID(plan, "load_balancer", "lb-a")
	if web.Service != "load_balancer" || web.Name != "web" || string(web.CreatedAt) != `"2025-01-01T00:00:00"` || string(web.UpdatedAt) != "null" || len(web.DependsOn) != 0 || web.Reason != "all resources" {
		t.Fatalf("load balancer %+v", web)
	}
	if !strings.Contains(string(web.Raw), `"provisioning_status":"ACTIVE"`) {
		t.Fatalf("raw row %s", web.Raw)
	}
	if got := candidateByID(plan, "server", "s1").DependsOn; len(got) != 0 {
		t.Fatalf("server depends on %v", got)
	}
	reasons := exclusionReasons(plan)
	if !strings.Contains(reasons["load_balancer:lb-foreign"], "owned by project other (project_id)") ||
		!strings.Contains(reasons["load_balancer:lb-tenant"], "owned by project other (tenant_id)") ||
		!strings.Contains(reasons["load_balancer:lb-unowned"], "project_id is not reported") ||
		!strings.Contains(reasons["load_balancer:"], "no usable id") || len(reasons) != 4 {
		t.Fatalf("exclusions %v", reasons)
	}
	for _, u := range plan.Unavailable {
		if u.Service == "load_balancer" {
			t.Fatalf("load balancer unavailable: %+v", plan.Unavailable)
		}
	}
	if got := cloud.events(); !slices.Equal(got, []string{"list stacks", "list load balancers", "list servers", "list groups"}) {
		t.Fatalf("requests %v", got)
	}
}

func TestProjectCleanupLoadBalancerFiltersSelectorAndSkip(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveLoadBalancers(t, cleanupLoadBalancerRows)
	cloud.serveCompute(t, ``, ``)
	ctx := context.Background()
	plan, err := cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithUpdatedBefore(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"load_balancer:lb-b"}) {
		t.Fatalf("updated filter candidates %v", got)
	}
	if reason := exclusionReasons(plan)["load_balancer:lb-a"]; !strings.Contains(reason, "updated_at") {
		t.Fatalf("null updated_at: %q", reason)
	}
	plan, err = cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithCreatedBefore(time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, []string{"load_balancer:lb-a"}) {
		t.Fatalf("created filter candidates %v", got)
	}
	var seen []string
	plan, err = cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithSelector(func(c projectcleanup.Candidate) (bool, error) {
		seen = append(seen, c.Kind+":"+c.Name)
		return c.Name == "db", nil
	}))
	if err != nil || !slices.Equal(refIDs(plan.Candidates()), []string{"load_balancer:lb-b"}) || !slices.Equal(seen, []string{"load_balancer:web", "load_balancer:db"}) {
		t.Fatalf("selector %v %v %v", plan, seen, err)
	}
	before := len(cloud.events())
	plan, err = cloud.conn.PlanProjectCleanup(ctx, projectcleanup.WithAllResources(), projectcleanup.WithSkip("load_balancer.load_balancer"))
	if err != nil || len(plan.Candidates()) != 0 {
		t.Fatalf("skip %v %v", plan, err)
	}
	if got := cloud.events()[before:]; !slices.Equal(got, []string{"list servers", "list groups"}) {
		t.Fatalf("skipped load balancers were listed: %v", got)
	}
}

func TestProjectCleanupLoadBalancerExecutesCascadeDeletesThenCompute(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveLoadBalancers(t, `{"id":"ok","project_id":"project"},{"id":"soft","project_id":"project"},{"id":"broken","project_id":"project"},`+
		`{"id":"gone","project_id":"project"},{"id":"locked","project_id":"project"},{"id":"lost","project_id":"project"}`)
	cloud.serveLoadBalancerMutations(t, map[string]int{"gone": 404, "locked": 409, "lost": -1},
		map[string][]string{"soft": {"PENDING_DELETE", "DELETED"}, "broken": {"PENDING_DELETE", "ERROR"}})
	cloud.serveCompute(t, `{"id":"s1"}`, ``)
	cloud.Mux.HandleFunc("DELETE /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		cloud.record("delete " + r.PathValue("id"))
		w.WriteHeader(204)
	})
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		cloud.record("get " + r.PathValue("id"))
		testcloud.JSON(w, 404, `{"itemNotFound":{"code":404}}`)
	})
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	result, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithDeletionWait(30*time.Second))
	if !errors.Is(err, projectcleanup.ErrIncomplete) || result == nil {
		t.Fatalf("result %v err %v", result, err)
	}
	outcomes := map[string]projectcleanup.Outcome{}
	for _, o := range result.Outcomes {
		outcomes[o.Candidate.ID] = o
	}
	want := map[string]projectcleanup.State{"ok": projectcleanup.Deleted, "soft": projectcleanup.Deleted, "broken": projectcleanup.Accepted,
		"gone": projectcleanup.AlreadyMissing, "locked": projectcleanup.Failed, "lost": projectcleanup.Uncertain, "s1": projectcleanup.Deleted}
	for id, state := range want {
		if outcomes[id].State != state {
			t.Errorf("%s: %s, want %s (%v / %v)", id, outcomes[id].State, state, outcomes[id].Err, outcomes[id].WaitErr)
		}
	}
	if len(result.Outcomes) != len(want) {
		t.Fatalf("outcomes %+v", result.Outcomes)
	}
	if broken := outcomes["broken"]; !errors.Is(broken.WaitErr, resource.ErrFailedState) || !strings.Contains(broken.WaitErr.Error(), "ERROR") {
		t.Fatalf("broken wait %v", broken.WaitErr)
	}
	if locked := outcomes["locked"]; locked.StatusCode != 409 || locked.Header.Get("X-Openstack-Request-Id") != "req-locked" || !strings.Contains(string(locked.Body), "load balancer locked") {
		t.Fatalf("locked evidence %+v", locked)
	}
	if gone := outcomes["gone"]; gone.StatusCode != 404 || gone.WaitErr != nil {
		t.Fatalf("gone %+v", gone)
	}
	// Each load balancer gets one cascade DELETE; only accepted ones are
	// polled, after the whole stage. Compute runs after the waits even when
	// a load balancer failed, as Python's dependency walk does.
	wantEvents := []string{"list load balancers", "list servers", "list groups",
		"DELETE ok?cascade=true", "DELETE soft?cascade=true", "DELETE broken?cascade=true",
		"DELETE gone?cascade=true", "DELETE locked?cascade=true", "DELETE lost?cascade=true",
		"GET ok", "GET soft", "GET soft", "GET broken", "GET broken",
		"delete s1", "get s1"}
	if got := cloud.events(); !slices.Equal(got, wantEvents) {
		t.Fatalf("events\n got %v\nwant %v", got, wantEvents)
	}
}

func TestProjectCleanupLoadBalancerWaitTimeoutKeepsAcceptedLoadBalancer(t *testing.T) {
	cloud := newCleanupCloud(t)
	cloud.serveLoadBalancers(t, `{"id":"slow","project_id":"project"}`)
	cloud.serveLoadBalancerMutations(t, nil, map[string][]string{"slow": {"PENDING_DELETE"}})
	cloud.serveCompute(t, ``, ``)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	result, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithDeletionWait(100*time.Millisecond))
	if !errors.Is(err, projectcleanup.ErrIncomplete) || len(result.Outcomes) != 1 {
		t.Fatalf("result %v err %v", result, err)
	}
	slow := result.Outcomes[0]
	if slow.State != projectcleanup.Accepted || slow.WaitErr == nil || errors.Is(slow.WaitErr, resource.ErrFailedState) {
		t.Fatalf("slow %+v", slow)
	}
	if got := cloud.events(); !slices.Equal(got, []string{"list load balancers", "list servers", "list groups", "DELETE slow?cascade=true", "GET slow"}) {
		t.Fatalf("events %v", got)
	}
}

// newVIPPortFixture serves networks that hold Octavia VIP ports (device_owner
// Octavia, device_id lb-<load balancer ID>): net-a holds the VIP port of
// candidate lb-a plus an unowned port and a subnet, net-b the VIP port of
// candidate lb-b and a subnet, and net-x the VIP port of lb-x, which belongs
// to another project and is not a candidate.
func newVIPPortFixture(t *testing.T) *networkFixture {
	f := newNetworkFixture(t, ``, map[string]string{
		"service-providers?service_type=VPN": ``,
		"floatingips?project_id=project":     ``,
		"security-groups?project_id=project": ``,
		"networks?project_id=project": `{"id":"net-a","project_id":"project"},{"id":"net-b","project_id":"project"},` +
			`{"id":"net-x","project_id":"project"}`,
		"ports?network_id=net-a&project_id=project": `{"id":"p-vip-a","project_id":"project","device_owner":"Octavia","device_id":"lb-lb-a"},` +
			`{"id":"p-free","project_id":"project","device_owner":""}`,
		"ports?network_id=net-b&project_id=project":   `{"id":"p-vip-b","project_id":"project","device_owner":"Octavia","device_id":"lb-lb-b"}`,
		"ports?network_id=net-x&project_id=project":   `{"id":"p-vip-x","project_id":"project","device_owner":"Octavia","device_id":"lb-lb-x"}`,
		"subnets?network_id=net-a&project_id=project": `{"id":"sn-a","project_id":"project"}`,
		"subnets?network_id=net-b&project_id=project": `{"id":"sn-b","project_id":"project"}`,
		"routers?project_id=project":                  ``,
	})
	f.serveLoadBalancers(t, `{"id":"lb-a","project_id":"project","vip_port_id":"p-vip-a","vip_network_id":"net-a"},`+
		`{"id":"lb-b","project_id":"project","vip_port_id":"p-vip-b","vip_network_id":"net-b"},`+
		`{"id":"lb-x","project_id":"other","vip_port_id":"p-vip-x","vip_network_id":"net-x"}`)
	return f
}

// The Octavia VIP port of a load balancer candidate does not keep its
// network: the port is not planned (Octavia deletes it with the load
// balancer), and the network and its subnets depend on the load balancer.
// Python's non-dry run reaches the same state by listing networks only after
// its load balancers are gone. A VIP port of a load balancer that is not a
// candidate still keeps its network.
func TestProjectCleanupLoadBalancerVIPPortFollowsCandidate(t *testing.T) {
	f := newVIPPortFixture(t)
	plan, err := f.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"load_balancer:lb-a", "load_balancer:lb-b", "port:p-free", "subnet:sn-a", "subnet:sn-b", "network:net-a", "network:net-b"}
	if got := refIDs(plan.Candidates()); !slices.Equal(got, want) {
		t.Fatalf("candidates %v", got)
	}
	lbA := projectcleanup.Ref{Service: "load_balancer", Kind: "load_balancer", ID: "lb-a"}
	lbB := projectcleanup.Ref{Service: "load_balancer", Kind: "load_balancer", ID: "lb-b"}
	free := projectcleanup.Ref{Service: "network", Kind: "port", ID: "p-free"}
	snA := projectcleanup.Ref{Service: "network", Kind: "subnet", ID: "sn-a"}
	snB := projectcleanup.Ref{Service: "network", Kind: "subnet", ID: "sn-b"}
	for id, deps := range map[string][]projectcleanup.Ref{
		"subnet:sn-a": {free, lbA}, "network:net-a": {free, lbA, snA},
		"subnet:sn-b": {lbB}, "network:net-b": {lbB, snB},
		"port:p-free": nil,
	} {
		kind, ref, _ := strings.Cut(id, ":")
		if got := candidateByID(plan, kind, ref).DependsOn; !slices.Equal(got, deps) {
			t.Errorf("%s depends on %v, want %v", id, got, deps)
		}
	}
	reasons := exclusionReasons(plan)
	if reason := reasons["network:net-x"]; !strings.Contains(reason, "p-vip-x (Octavia lb-lb-x)") {
		t.Fatalf("network exclusion %q", reason)
	}
	if !strings.Contains(reasons["load_balancer:lb-x"], "owned by project other") {
		t.Fatalf("exclusions %v", reasons)
	}
	for _, c := range plan.Candidates() {
		if strings.HasPrefix(c.ID, "p-vip") {
			t.Fatalf("VIP port planned: %+v", c)
		}
	}

	// Skipping load balancers leaves no candidate, so the VIP ports keep
	// their networks again.
	plan, err = f.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources(), projectcleanup.WithSkip("load_balancer.load_balancer"))
	if err != nil {
		t.Fatal(err)
	}
	if got := refIDs(plan.Candidates()); len(got) != 0 {
		t.Fatalf("candidates with skipped load balancers %v", got)
	}
	reasons = exclusionReasons(plan)
	if !strings.Contains(reasons["network:net-a"], "p-vip-a (Octavia lb-lb-a)") || !strings.Contains(reasons["network:net-b"], "p-vip-b (Octavia lb-lb-b)") {
		t.Fatalf("exclusions %v", reasons)
	}
}

// A network whose load balancer was deleted and waited for is cleaned; one
// whose load balancer DELETE failed is NotAttempted together with its subnet.
func TestProjectCleanupLoadBalancerFailureBlocksVIPNetwork(t *testing.T) {
	f := newVIPPortFixture(t)
	f.serveLoadBalancerMutations(t, map[string]int{"lb-b": 409}, map[string][]string{"lb-a": {"PENDING_DELETE", "DELETED"}})
	plan, err := f.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	before := len(f.events())
	result, err := f.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithDeletionWait(10*time.Second))
	if !errors.Is(err, projectcleanup.ErrIncomplete) {
		t.Fatalf("err %v", err)
	}
	outcomes := map[string]projectcleanup.Outcome{}
	for _, o := range result.Outcomes {
		outcomes[o.Candidate.Kind+":"+o.Candidate.ID] = o
	}
	for key, state := range map[string]projectcleanup.State{
		"load_balancer:lb-a": projectcleanup.Deleted, "load_balancer:lb-b": projectcleanup.Failed,
		"port:p-free": projectcleanup.Accepted, "subnet:sn-a": projectcleanup.Accepted, "network:net-a": projectcleanup.Accepted,
		"subnet:sn-b": projectcleanup.NotAttempted, "network:net-b": projectcleanup.NotAttempted,
	} {
		if outcomes[key].State != state {
			t.Errorf("%s: %s, want %s (%s %v)", key, outcomes[key].State, state, outcomes[key].Reason, outcomes[key].Err)
		}
	}
	if len(result.Outcomes) != 7 {
		t.Fatalf("outcomes %+v", result.Outcomes)
	}
	if reason := outcomes["network:net-b"].Reason; !strings.Contains(reason, "lb-b") {
		t.Fatalf("network reason %q", reason)
	}
	want := []string{"DELETE lb-a?cascade=true", "DELETE lb-b?cascade=true", "GET lb-a", "GET lb-a",
		"DELETE ports/p-free", "DELETE subnets/sn-a", "DELETE networks/net-a"}
	if got := f.events()[before:]; !slices.Equal(got, want) {
		t.Fatalf("requests\n got %v\nwant %v", got, want)
	}
}
