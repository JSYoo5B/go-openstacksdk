package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/capabilities"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots"
	cindergroups "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	conntrackhelpers "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers"
	ndpproxies "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networksegmentranges"
	qosrules "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	defaultrules "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultrules"
	defaultstatefulness "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// rawWaitRoutes serves a fixed sequence of GET bodies per path from one test
// server, repeating the last body. Other methods and unknown paths fail the
// test, so a waiter can neither mutate nor retarget. Counts are per path.
type rawWaitRoutes struct {
	t      *testing.T
	mu     sync.Mutex
	bodies map[string][]string
	gets   map[string]int
}

func newRawWaitRoutes(t *testing.T, cloud *testcloud.Cloud, discovery map[string]string) *rawWaitRoutes {
	t.Helper()
	routes := &rawWaitRoutes{t: t, bodies: map[string][]string{}, gets: map[string]int{}}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if body, ok := discovery[r.URL.Path]; ok && r.Method == http.MethodGet {
			testcloud.JSON(w, 200, body)
			return
		}
		routes.mu.Lock()
		bodies, ok := routes.bodies[r.URL.Path]
		n := routes.gets[r.URL.Path]
		routes.gets[r.URL.Path] = n + 1
		routes.mu.Unlock()
		if r.Method != http.MethodGet || !ok {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			testcloud.JSON(w, 500, `{}`)
			return
		}
		testcloud.JSON(w, 200, bodies[min(n, len(bodies)-1)])
	})
	return routes
}

func (r *rawWaitRoutes) serve(path string, bodies ...string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bodies[path] = bodies
	r.gets[path] = 0
	return path
}

func (r *rawWaitRoutes) count(path string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.gets[path]
}

func rawWaitFast(attribute string, extra ...resource.WaitOption) []resource.WaitOption {
	return append([]resource.WaitOption{resource.WithPollInterval(time.Millisecond), resource.WithTimeout(5 * time.Second), resource.WithStatusAttribute(attribute)}, extra...)
}

func TestRawRecordWaitAttributesNeutron(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	routes := newRawWaitRoutes(t, cloud, nil)
	client := cloud.Client("network", "/neutron/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"

	// NetworkSegmentRange allows unknown Body attributes: an undeclared status
	// member is a Python attribute; the default ERROR failure applies.
	ranges := networksegmentranges.New(client).Resources()
	p := routes.serve("/neutron/v2.0/network_segment_ranges/r-1", `{"network_segment_range":{"id":"r-1","status":"BUILD"}}`, `{"network_segment_range":{"id":"r-1","status":"active"}}`)
	if value, err := network.WaitForStatus(ctx, ranges, resource.ID("r-1"), "ACTIVE", rawWaitFast("status")...); err != nil || value.Text("status") != "active" || routes.count(p) != 2 {
		t.Fatalf("segment range value=%v err=%v gets=%d", value, err, routes.count(p))
	}
	p = routes.serve("/neutron/v2.0/network_segment_ranges/r-2", `{"network_segment_range":{"id":"r-2","status":"error"}}`)
	if _, err := network.WaitForStatus(ctx, ranges, resource.ID("r-2"), "ACTIVE", rawWaitFast("status")...); !errors.Is(err, resource.ErrFailedState) || routes.count(p) != 1 {
		t.Fatalf("segment range failure err=%v gets=%d", err, routes.count(p))
	}
	p = routes.serve("/neutron/v2.0/network_segment_ranges/r-3", `{"network_segment_range":{"id":"r-3","status":"ACTIVE"}}`, `{"network_segment_range":{"id":"r-3","status":"deleted"}}`)
	if err := network.WaitForDelete(ctx, ranges, resource.ID("r-3"), rawWaitFast("status")...); err != nil || routes.count(p) != 2 {
		t.Fatalf("segment range deleted err=%v gets=%d", err, routes.count(p))
	}
	// Integer descriptors are not strings.
	p = routes.serve("/neutron/v2.0/network_segment_ranges/r-4", `{"network_segment_range":{"id":"r-4","minimum":100}}`)
	if _, err := network.WaitForStatus(ctx, ranges, resource.ID("r-4"), "100", rawWaitFast("minimum")...); !errors.Is(err, resource.ErrUnsupported) || routes.count(p) != 1 {
		t.Fatalf("segment range integer err=%v gets=%d", err, routes.count(p))
	}

	// DefaultSecurityGroupRule maps the ethertype wire key to ether_type; the
	// wire spelling is not an attribute, and a boolean is not a string.
	rules := defaultrules.New(client).Resources()
	p = routes.serve("/neutron/v2.0/default-security-group-rules/dr-1", `{"default_security_group_rule":{"id":"dr-1","ethertype":"IPv4","used_in_default_sg":true}}`, `{"default_security_group_rule":{"id":"dr-1","ethertype":"IPv6","used_in_default_sg":true}}`)
	if _, err := network.WaitForStatus(ctx, rules, resource.ID("dr-1"), "ipv6", rawWaitFast("ether_type")...); err != nil || routes.count(p) != 2 {
		t.Fatalf("default rule err=%v gets=%d", err, routes.count(p))
	}
	for _, attribute := range []string{"ethertype", "used_in_default_sg"} {
		routes.serve(p, `{"default_security_group_rule":{"id":"dr-1","ethertype":"IPv4","used_in_default_sg":true}}`)
		if _, err := network.WaitForStatus(ctx, rules, resource.ID("dr-1"), "IPv4", rawWaitFast(attribute)...); !errors.Is(err, resource.ErrUnsupported) || routes.count(p) != 1 {
			t.Fatalf("default rule %s err=%v gets=%d", attribute, err, routes.count(p))
		}
	}

	statefulness := defaultstatefulness.New(client).Resources()
	p = routes.serve("/neutron/v2.0/security-groups-default-statefulness/ds-1", `{"security_group_default_statefulness":{"id":"ds-1","project_id":"p-1","stateful":true}}`)
	if _, err := network.WaitForStatus(ctx, statefulness, resource.ID("ds-1"), "P-1", rawWaitFast("project_id")...); err != nil || routes.count(p) != 1 {
		t.Fatalf("statefulness err=%v gets=%d", err, routes.count(p))
	}
	routes.serve(p, `{"security_group_default_statefulness":{"id":"ds-1","project_id":"p-1","stateful":true}}`)
	if _, err := network.WaitForStatus(ctx, statefulness, resource.ID("ds-1"), "true", rawWaitFast("stateful")...); !errors.Is(err, resource.ErrUnsupported) || routes.count(p) != 1 {
		t.Fatalf("statefulness boolean err=%v gets=%d", err, routes.count(p))
	}

	// NDP proxy: caller failure states replace the default, and a declared
	// attribute that is absent from the response is null.
	proxies := ndpproxies.New(client).Resources()
	p = routes.serve("/neutron/v2.0/ndp_proxies/np-1", `{"ndp_proxy":{"id":"np-1","description":"pending"}}`, `{"ndp_proxy":{"id":"np-1","description":"BROKEN"}}`)
	if _, err := network.WaitForStatus(ctx, proxies, resource.ID("np-1"), "ready", rawWaitFast("description", resource.WithFailureStates("broken"))...); !errors.Is(err, resource.ErrFailedState) || routes.count(p) != 2 {
		t.Fatalf("NDP proxy failure err=%v gets=%d", err, routes.count(p))
	}
	routes.serve(p, `{"ndp_proxy":{"id":"np-1"}}`)
	if _, err := network.WaitForStatus(ctx, proxies, resource.ID("np-1"), "ready", rawWaitFast("description")...); !errors.Is(err, resource.ErrUnsupported) || routes.count(p) != 1 {
		t.Fatalf("NDP proxy null err=%v gets=%d", err, routes.count(p))
	}

	// ConntrackHelper does not allow unknown Body attributes: an undeclared
	// response member is discarded and cannot be selected. The router_id URI
	// attribute stays outside the view (Record.RouterID).
	helpers := conntrackhelpers.New(client).Resources("r-1")
	p = routes.serve("/neutron/v2.0/routers/r-1/conntrack_helpers/ch-1", `{"conntrack_helper":{"id":"ch-1","helper":"FTP","port":21,"status":"ACTIVE"}}`)
	if _, err := network.WaitForStatus(ctx, helpers, resource.ID("ch-1"), "ftp", rawWaitFast("helper")...); err != nil || routes.count(p) != 1 {
		t.Fatalf("conntrack err=%v gets=%d", err, routes.count(p))
	}
	for _, attribute := range []string{"status", "port", "router_id"} {
		routes.serve(p, `{"conntrack_helper":{"id":"ch-1","helper":"FTP","port":21,"status":"ACTIVE"}}`)
		if _, err := network.WaitForStatus(ctx, helpers, resource.ID("ch-1"), "ACTIVE", rawWaitFast(attribute)...); !errors.Is(err, resource.ErrUnsupported) || routes.count(p) != 1 {
			t.Fatalf("conntrack %s err=%v gets=%d", attribute, err, routes.count(p))
		}
	}

	qos := qosrules.New(client)
	p = routes.serve("/neutron/v2.0/qos/policies/qp-1/minimum_packet_rate_rules/mp-1", `{"minimum_packet_rate_rule":{"id":"mp-1","direction":"egress","min_kpps":1}}`, `{"minimum_packet_rate_rule":{"id":"mp-1","direction":"any","min_kpps":1}}`)
	if _, err := network.WaitForStatus(ctx, qos.MinimumPacketRateResources("qp-1"), resource.ID("mp-1"), "ANY", rawWaitFast("direction")...); err != nil || routes.count(p) != 2 {
		t.Fatalf("minimum packet rate err=%v gets=%d", err, routes.count(p))
	}
	p = routes.serve("/neutron/v2.0/qos/policies/qp-1/packet_rate_limit_rules/pr-1", `{"packet_rate_limit_rule":{"id":"pr-1","direction":"egress","max_kpps":5}}`)
	if _, err := network.WaitForStatus(ctx, qos.PacketRateLimitResources("qp-1"), resource.ID("pr-1"), "EGRESS", rawWaitFast("direction")...); err != nil || routes.count(p) != 1 {
		t.Fatalf("packet rate limit err=%v gets=%d", err, routes.count(p))
	}
	routes.serve(p, `{"packet_rate_limit_rule":{"id":"pr-1","direction":"egress","max_kpps":5}}`)
	if _, err := network.WaitForStatus(ctx, qos.PacketRateLimitResources("qp-1"), resource.ID("pr-1"), "5", rawWaitFast("max_kpps")...); !errors.Is(err, resource.ErrUnsupported) || routes.count(p) != 1 {
		t.Fatalf("packet rate limit integer err=%v gets=%d", err, routes.count(p))
	}

	// Without a selected attribute these statusless records still fail
	// before any request.
	p = routes.serve("/neutron/v2.0/ndp_proxies/np-2", `{"ndp_proxy":{"id":"np-2","status":"ACTIVE"}}`)
	if _, err := network.WaitForStatus(ctx, proxies, resource.ID("np-2"), "ACTIVE", resource.WithPollInterval(time.Millisecond)); !errors.Is(err, resource.ErrUnsupported) || routes.count(p) != 0 {
		t.Fatalf("NDP proxy default err=%v gets=%d", err, routes.count(p))
	}
}

func TestRawRecordWaitAttributesKeystone(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	routes := newRawWaitRoutes(t, cloud, nil)
	client := cloud.Client("identity", "/v3/")

	groups := endpointgroups.New(client).Resources()
	p := routes.serve("/v3/OS-EP-FILTER/endpoint_groups/eg-1", `{"endpoint_group":{"id":"eg-1","name":"pending","filters":{}}}`, `{"endpoint_group":{"id":"eg-1","name":"Ready","filters":{}}}`)
	if value, err := identity.WaitForStatus(ctx, groups, resource.ID("eg-1"), "ready", rawWaitFast("name")...); err != nil || value.Text("name") != "Ready" || routes.count(p) != 2 {
		t.Fatalf("endpoint group value=%v err=%v gets=%d", value, err, routes.count(p))
	}
	routes.serve(p, `{"endpoint_group":{"id":"eg-1","name":"ERROR","filters":{}}}`)
	if _, err := identity.WaitForStatus(ctx, groups, resource.ID("eg-1"), "ready", rawWaitFast("name")...); !errors.Is(err, resource.ErrFailedState) || routes.count(p) != 1 {
		t.Fatalf("endpoint group failure err=%v gets=%d", err, routes.count(p))
	}
	routes.serve(p, `{"endpoint_group":{"id":"eg-1","name":"a","filters":{}}}`)
	if _, err := identity.WaitForStatus(ctx, groups, resource.ID("eg-1"), "a", rawWaitFast("filters")...); !errors.Is(err, resource.ErrUnsupported) || routes.count(p) != 1 {
		t.Fatalf("endpoint group object err=%v gets=%d", err, routes.count(p))
	}

	// DomainConfig option groups are nested Resources, not strings, and its
	// computed location is null.
	configs := domains.New(client).ConfigResources()
	for _, attribute := range []string{"identity", "location", "domain_id"} {
		p = routes.serve("/v3/domains/d-1/config", `{"config":{"identity":{"driver":"ldap"}}}`)
		if _, err := identity.WaitForStatus(ctx, configs, resource.ID("d-1"), "ldap", rawWaitFast(attribute)...); !errors.Is(err, resource.ErrUnsupported) || routes.count(p) != 1 {
			t.Fatalf("domain config %s err=%v gets=%d", attribute, err, routes.count(p))
		}
	}
}

func TestRawRecordWaitAttributesCinder(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	routes := newRawWaitRoutes(t, cloud, map[string]string{
		"/cinder/v3/": `{"version":{"id":"v3.0","status":"CURRENT","version":"3.70","min_version":"3.0"}}`,
	})
	client := cloud.Client("volumev3", "/cinder/v3/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"

	// Group-family records keep the default status accessor and also accept a
	// selected attribute for waiting and the deleted check.
	groups := cindergroups.New(client).Resources()
	p := routes.serve("/cinder/v3/project/groups/g-1", `{"group":{"id":"g-1","status":"available","replication_status":"enabling"}}`, `{"group":{"id":"g-1","status":"available","replication_status":"enabled"}}`)
	if _, err := blockstorage.WaitForState(ctx, groups, resource.ID("g-1"), "ENABLED", rawWaitFast("replication_status")...); err != nil || routes.count(p) != 2 {
		t.Fatalf("group err=%v gets=%d", err, routes.count(p))
	}
	routes.serve(p, `{"group":{"id":"g-1","status":"available","replication_status":"Error"}}`)
	if _, err := blockstorage.WaitForState(ctx, groups, resource.ID("g-1"), "enabled", rawWaitFast("replication_status")...); !errors.Is(err, resource.ErrFailedState) || routes.count(p) != 1 {
		t.Fatalf("group failure err=%v gets=%d", err, routes.count(p))
	}
	routes.serve(p, `{"group":{"id":"g-1","status":"creating"}}`, `{"group":{"id":"g-1","status":"available"}}`)
	if _, err := blockstorage.WaitForAvailable(ctx, groups, resource.ID("g-1"), resource.WithPollInterval(time.Millisecond)); err != nil || routes.count(p) != 2 {
		t.Fatalf("group default err=%v gets=%d", err, routes.count(p))
	}
	routes.serve(p, `{"group":{"id":"g-1","status":"available","group_type":"DELETED"}}`)
	if err := blockstorage.WaitForDelete(ctx, groups, resource.ID("g-1"), rawWaitFast("group_type")...); err != nil || routes.count(p) != 1 {
		t.Fatalf("group deleted err=%v gets=%d", err, routes.count(p))
	}

	p = routes.serve("/cinder/v3/project/group_snapshots/gs-1", `{"group_snapshot":{"id":"gs-1","status":"available","name":"a"}}`, `{"group_snapshot":{"id":"gs-1","status":"available","name":"b"}}`)
	if _, err := blockstorage.WaitForState(ctx, groupsnapshots.New(client).Resources(), resource.ID("gs-1"), "B", rawWaitFast("name")...); err != nil || routes.count(p) != 2 {
		t.Fatalf("group snapshot err=%v gets=%d", err, routes.count(p))
	}
	p = routes.serve("/cinder/v3/project/consistencygroups/cg-1", `{"consistencygroup":{"id":"cg-1","status":"available","availability_zone":"nova"}}`)
	if _, err := blockstorage.WaitForState(ctx, consistencygroups.New(client).Resources(), resource.ID("cg-1"), "NOVA", rawWaitFast("availability_zone")...); err != nil || routes.count(p) != 1 {
		t.Fatalf("consistency group err=%v gets=%d", err, routes.count(p))
	}
	p = routes.serve("/cinder/v3/project/cgsnapshots/cs-1", `{"cgsnapshot":{"id":"cs-1","status":"available","consistencygroup_id":"cg-1"}}`)
	if _, err := blockstorage.WaitForState(ctx, consistencygroupsnapshots.New(client).Resources(), resource.ID("cs-1"), "CG-1", rawWaitFast("consistencygroup_id")...); err != nil || routes.count(p) != 1 {
		t.Fatalf("consistency group snapshot err=%v gets=%d", err, routes.count(p))
	}

	// Message, capabilities and quota class records declare no status, so a
	// selected attribute is required.
	msgs := messages.New(client).Resources()
	p = routes.serve("/cinder/v3/project/messages/m-1", `{"message":{"id":"m-1","event_id":"VOLUME_000001","message_level":"ERROR"}}`)
	if _, err := blockstorage.WaitForState(ctx, msgs, resource.ID("m-1"), "volume_000001", rawWaitFast("event_id")...); err != nil || routes.count(p) != 1 {
		t.Fatalf("message err=%v gets=%d", err, routes.count(p))
	}
	routes.serve(p, `{"message":{"id":"m-1","event_id":"VOLUME_000001","message_level":"ERROR"}}`)
	if _, err := blockstorage.WaitForState(ctx, msgs, resource.ID("m-1"), "info", rawWaitFast("message_level")...); !errors.Is(err, resource.ErrFailedState) || routes.count(p) != 1 {
		t.Fatalf("message failure err=%v gets=%d", err, routes.count(p))
	}
	routes.serve(p, `{"message":{"id":"m-1"}}`)
	if _, err := blockstorage.WaitForState(ctx, msgs, resource.ID("m-1"), "x", resource.WithPollInterval(time.Millisecond)); !errors.Is(err, resource.ErrUnsupported) || routes.count(p) != 0 {
		t.Fatalf("message default err=%v gets=%d", err, routes.count(p))
	}

	p = routes.serve("/cinder/v3/project/capabilities/host@lvm", `{"namespace":"OS::Storage::Capabilities::host@lvm","vendor_name":"Open Source","volume_backend_name":"lvm"}`)
	if _, err := blockstorage.WaitForState(ctx, capabilities.New(client).Resources(), resource.ID("host@lvm"), "open source", rawWaitFast("vendor_name")...); err != nil || routes.count(p) != 1 {
		t.Fatalf("capabilities err=%v gets=%d", err, routes.count(p))
	}

	quotaClasses := quotasets.New(client).QuotaClassResources()
	p = routes.serve("/cinder/v3/project/os-quota-class-sets/gold", `{"quota_class_set":{"id":"gold","volumes":10}}`)
	if _, err := blockstorage.WaitForState(ctx, quotaClasses, resource.ID("gold"), "GOLD", rawWaitFast("id")...); err != nil || routes.count(p) != 1 {
		t.Fatalf("quota class err=%v gets=%d", err, routes.count(p))
	}
	routes.serve(p, `{"quota_class_set":{"id":"gold","volumes":10}}`)
	if _, err := blockstorage.WaitForState(ctx, quotaClasses, resource.ID("gold"), "10", rawWaitFast("volumes")...); !errors.Is(err, resource.ErrUnsupported) || routes.count(p) != 1 {
		t.Fatalf("quota class integer err=%v gets=%d", err, routes.count(p))
	}
}
