package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pyCloudNetReply struct {
	status int
	body   string
}

type pyCloudNetFixture struct {
	cloud   *testcloud.Cloud
	mu      sync.Mutex
	calls   []string
	bodies  []string
	replies map[string][]pyCloudNetReply
}

func newPyCloudNetFixture(t *testing.T, replies map[string][]pyCloudNetReply) *pyCloudNetFixture {
	t.Helper()
	f := &pyCloudNetFixture{cloud: testcloud.New(t), replies: replies}
	f.cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		key := r.Method + " " + r.URL.Path
		f.mu.Lock()
		f.calls = append(f.calls, key)
		f.bodies = append(f.bodies, string(raw))
		queue := f.replies[key]
		if len(queue) == 0 {
			f.mu.Unlock()
			t.Errorf("unexpected request %s", key)
			testcloud.JSON(w, 500, `{}`)
			return
		}
		reply := queue[0]
		f.replies[key] = queue[1:]
		f.mu.Unlock()
		if reply.body == "" {
			w.WriteHeader(reply.status)
			return
		}
		testcloud.JSON(w, reply.status, reply.body)
	})
	return f
}

func (f *pyCloudNetFixture) neutron() *network.SecurityGroupCloud {
	client := f.cloud.Client("network", "/")
	client.ResourceBase = f.cloud.Server.URL + "/v2.0/"
	return network.NewSecurityGroupCloud(network.SecurityGroupReadDependencies{Neutron: network.New(client)})
}

func (f *pyCloudNetFixture) nova() *network.SecurityGroupCloud {
	absent := false
	source := "nova"
	return network.NewSecurityGroupCloud(network.SecurityGroupReadDependencies{Nova: f.cloud.Client("compute", "/"), Source: &source, NetworkAvailable: &absent})
}

func (f *pyCloudNetFixture) snapshot() ([]string, []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...), append([]string(nil), f.bodies...)
}

func pyCloudNetJSONEqual(t *testing.T, got, want string) {
	t.Helper()
	var a, b any
	if json.Unmarshal([]byte(got), &a) != nil || json.Unmarshal([]byte(want), &b) != nil || !reflect.DeepEqual(a, b) {
		t.Fatalf("body %s want %s", got, want)
	}
}

const pyCloudNetListGroups = `{"security_groups":[{"id":"sg-1","name":"web","description":"d","security_group_rules":[]}]}`

func TestPyCloudNetSecurityGroupDeleteNeutronListsThenDeletesOnceAndMissingIsFalse(t *testing.T) {
	f := newPyCloudNetFixture(t, map[string][]pyCloudNetReply{
		"GET /v2.0/security-groups":         {{200, pyCloudNetListGroups}, {200, pyCloudNetListGroups}},
		"DELETE /v2.0/security-groups/sg-1": {{204, ""}},
	})
	ok, err := f.neutron().DeleteSecurityGroup(context.Background(), "web")
	calls, _ := f.snapshot()
	if err != nil || !ok || !reflect.DeepEqual(calls, []string{"GET /v2.0/security-groups", "DELETE /v2.0/security-groups/sg-1"}) {
		t.Fatal(ok, err, calls)
	}
	ok, err = f.neutron().DeleteSecurityGroup(context.Background(), "absent")
	calls, _ = f.snapshot()
	if err != nil || ok || len(calls) != 3 {
		t.Fatal(ok, err, calls)
	}
}

func TestPyCloudNetSecurityGroupDeleteStrictAfterLookupAndAmbiguous(t *testing.T) {
	f := newPyCloudNetFixture(t, map[string][]pyCloudNetReply{
		"GET /v2.0/security-groups":         {{200, pyCloudNetListGroups}},
		"DELETE /v2.0/security-groups/sg-1": {{404, `{"NeutronError":{"message":"gone"}}`}},
	})
	ok, err := f.neutron().DeleteSecurityGroup(context.Background(), "sg-1")
	if err == nil || ok {
		t.Fatal("404 after lookup must fail", ok, err)
	}
	dup := `{"security_groups":[{"id":"a","name":"web"},{"id":"b","name":"web"}]}`
	f = newPyCloudNetFixture(t, map[string][]pyCloudNetReply{"GET /v2.0/security-groups": {{200, dup}}})
	ok, err = f.neutron().DeleteSecurityGroup(context.Background(), "web")
	calls, _ := f.snapshot()
	if err == nil || ok || !errors.Is(err, resource.ErrAmbiguous) || len(calls) != 1 {
		t.Fatal(ok, err, calls)
	}
}

func TestPyCloudNetSecurityGroupDeleteNovaUsesOSSecurityGroups(t *testing.T) {
	f := newPyCloudNetFixture(t, map[string][]pyCloudNetReply{
		"GET /os-security-groups":      {{200, `{"security_groups":[{"id":"7","name":"web","description":"d","rules":[],"tenant_id":"p"}]}`}},
		"DELETE /os-security-groups/7": {{202, ""}},
	})
	ok, err := f.nova().DeleteSecurityGroup(context.Background(), "web")
	calls, _ := f.snapshot()
	if err != nil || !ok || !reflect.DeepEqual(calls, []string{"GET /os-security-groups", "DELETE /os-security-groups/7"}) {
		t.Fatal(ok, err, calls)
	}
}

func TestPyCloudNetSecurityGroupDeleteFeatureDisabledSendsNothing(t *testing.T) {
	f := newPyCloudNetFixture(t, map[string][]pyCloudNetReply{})
	disabled := ""
	cloud := network.NewSecurityGroupCloud(network.SecurityGroupReadDependencies{Source: &disabled})
	ok, err := cloud.DeleteSecurityGroup(context.Background(), "web")
	calls, _ := f.snapshot()
	if ok || !errors.Is(err, network.ErrUnavailableSecurityGroups) || len(calls) != 0 {
		t.Fatal(ok, err, calls)
	}
	if _, err := cloud.UpdateSecurityGroup(context.Background(), "web", network.SecurityGroupUpdate{}); !errors.Is(err, network.ErrUnavailableSecurityGroups) {
		t.Fatal(err)
	}
	if _, err := cloud.CreateSecurityGroupRule(context.Background(), "web", network.SecurityGroupRuleSpec{}); !errors.Is(err, network.ErrUnavailableSecurityGroups) {
		t.Fatal(err)
	}
}

func TestPyCloudNetSecurityGroupUpdateNeutronSendsOnlySuppliedIncludingFalseAndEmpty(t *testing.T) {
	f := newPyCloudNetFixture(t, map[string][]pyCloudNetReply{
		"GET /v2.0/security-groups":      {{200, pyCloudNetListGroups}, {200, pyCloudNetListGroups}, {200, pyCloudNetListGroups}},
		"PUT /v2.0/security-groups/sg-1": {{200, `{"security_group":{"id":"sg-1","name":"","description":"d","stateful":false}}`}},
	})
	empty, no := "", false
	res, err := f.neutron().UpdateSecurityGroup(context.Background(), "web", network.SecurityGroupUpdate{Name: &empty, Stateful: &no})
	_, bodies := f.snapshot()
	if err != nil || res == nil || res.Backend != "neutron" || res.Group == nil || len(bodies) != 2 {
		t.Fatal(res, err, bodies)
	}
	pyCloudNetJSONEqual(t, bodies[1], `{"security_group":{"name":"","stateful":false}}`)
	// No fields: Resource has no dirty state, so no PUT is sent.
	res, err = f.neutron().UpdateSecurityGroup(context.Background(), "web", network.SecurityGroupUpdate{})
	calls, _ := f.snapshot()
	if err != nil || res.Group == nil || res.Group.ID != "sg-1" || len(calls) != 3 {
		t.Fatal(res, err, calls)
	}
	// Missing group fails before mutation.
	f.replies["GET /v2.0/security-groups"] = []pyCloudNetReply{{200, `{"security_groups":[]}`}}
	_, err = f.neutron().UpdateSecurityGroup(context.Background(), "nope", network.SecurityGroupUpdate{Name: &empty})
	calls, _ = f.snapshot()
	if !errors.Is(err, resource.ErrNotFound) || len(calls) != 4 {
		t.Fatal(err, calls)
	}
}

func TestPyCloudNetSecurityGroupUpdateNovaFillsNameAndDescription(t *testing.T) {
	f := newPyCloudNetFixture(t, map[string][]pyCloudNetReply{
		"GET /os-security-groups":   {{200, `{"security_groups":[{"id":"7","name":"web","description":"old","rules":[]}]}`}},
		"PUT /os-security-groups/7": {{200, `{"security_group":{"id":"7","name":"web","description":"new"}}`}},
	})
	desc := "new"
	res, err := f.nova().UpdateSecurityGroup(context.Background(), "web", network.SecurityGroupUpdate{Description: &desc})
	_, bodies := f.snapshot()
	if err != nil || res == nil || res.Backend != "nova" || len(res.NovaBody) == 0 || len(bodies) != 2 {
		t.Fatal(res, err, bodies)
	}
	pyCloudNetJSONEqual(t, bodies[1], `{"security_group":{"name":"web","description":"new"}}`)
}

func TestPyCloudNetSecurityGroupRuleNeutronBodyKeepsNullsAndMapsMinusOne(t *testing.T) {
	f := newPyCloudNetFixture(t, map[string][]pyCloudNetReply{
		"GET /v2.0/security-groups":       {{200, pyCloudNetListGroups}},
		"POST /v2.0/security-group-rules": {{201, `{"security_group_rule":{"id":"r1","security_group_id":"sg-1"}}`}},
	})
	min, max, proto := -1, 0, "icmp"
	res, err := f.neutron().CreateSecurityGroupRule(context.Background(), "web", network.SecurityGroupRuleSpec{PortRangeMin: &min, PortRangeMax: &max, Protocol: &proto})
	_, bodies := f.snapshot()
	if err != nil || res == nil || res.Rule == nil || res.Rule.ID != "r1" || len(bodies) != 2 {
		t.Fatal(res, err, bodies)
	}
	pyCloudNetJSONEqual(t, bodies[1], `{"security_group_rule":{"security_group_id":"sg-1","port_range_min":null,"port_range_max":0,"protocol":"icmp","remote_ip_prefix":null,"remote_group_id":null,"direction":"ingress","ethertype":"IPv4"}}`)
}

func TestPyCloudNetSecurityGroupRuleNeutronProjectAndDescriptionOnlyWhenGiven(t *testing.T) {
	f := newPyCloudNetFixture(t, map[string][]pyCloudNetReply{
		"GET /v2.0/security-groups":       {{200, pyCloudNetListGroups}},
		"POST /v2.0/security-group-rules": {{201, `{"security_group_rule":{"id":"r2"}}`}},
	})
	proj, desc, empty := "p", "", ""
	_, err := f.neutron().CreateSecurityGroupRule(context.Background(), "sg-1", network.SecurityGroupRuleSpec{ProjectID: &proj, Description: &desc, RemoteAddressGroupID: &empty, Direction: "egress", Ethertype: "IPv6"})
	_, bodies := f.snapshot()
	if err != nil || len(bodies) != 2 {
		t.Fatal(err, bodies)
	}
	pyCloudNetJSONEqual(t, bodies[1], `{"security_group_rule":{"security_group_id":"sg-1","port_range_min":null,"port_range_max":null,"protocol":null,"remote_ip_prefix":null,"remote_group_id":null,"direction":"egress","ethertype":"IPv6","tenant_id":"p","description":""}}`)
}

func TestPyCloudNetSecurityGroupRuleNovaRulesAndValidation(t *testing.T) {
	list := `{"security_groups":[{"id":"7","name":"web","description":"d","rules":[]}]}`
	f := newPyCloudNetFixture(t, map[string][]pyCloudNetReply{
		"GET /os-security-groups":       {{200, list}, {200, list}, {200, list}, {200, list}},
		"POST /os-security-group-rules": {{200, `{"security_group_rule":{"id":"9"}}`}, {200, `{"security_group_rule":{"id":"10"}}`}},
	})
	ctx := context.Background()
	tcp, icmp := "tcp", "icmp"
	if _, err := f.nova().CreateSecurityGroupRule(ctx, "web", network.SecurityGroupRuleSpec{Protocol: &tcp}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.nova().CreateSecurityGroupRule(ctx, "web", network.SecurityGroupRuleSpec{Protocol: &icmp}); err != nil {
		t.Fatal(err)
	}
	_, bodies := f.snapshot()
	pyCloudNetJSONEqual(t, bodies[1], `{"security_group_rule":{"parent_group_id":"7","ip_protocol":"tcp","from_port":1,"to_port":65535,"cidr":null,"group_id":null}}`)
	pyCloudNetJSONEqual(t, bodies[3], `{"security_group_rule":{"parent_group_id":"7","ip_protocol":"icmp","from_port":-1,"to_port":-1,"cidr":null,"group_id":null}}`)
	before, _ := f.snapshot()
	if _, err := f.nova().CreateSecurityGroupRule(ctx, "web", network.SecurityGroupRuleSpec{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("nil protocol must fail", err)
	}
	if _, err := f.nova().CreateSecurityGroupRule(ctx, "web", network.SecurityGroupRuleSpec{Protocol: &tcp, Direction: "egress"}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("egress must fail", err)
	}
	after, _ := f.snapshot()
	for _, c := range after[len(before):] {
		if c != "GET /os-security-groups" {
			t.Fatal("validation failure must not mutate", after)
		}
	}
}
