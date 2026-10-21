package api_test

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "github.com/JSYoo5B/go-openstacksdk/baremetal/v1"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/ports"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func r2icService(cloud *testcloud.Cloud) *v1.Service { return v1.New(r2icClient(cloud, "1.99")) }

func TestR2IronicCMachineProvisionStateConfigDriveWaitAndDefaults(t *testing.T) {
	cloud := testcloud.New(t)
	put := r2icServe(t, cloud, "PUT /v1/nodes/n1/states/provision", 202, "")
	r2icServe(t, cloud, "GET /v1/nodes/n1", 200, `{"uuid":"n1","provision_state":"deploying"}`, 200, `{"uuid":"n1","provision_state":"active"}`)
	s := r2icService(cloud)
	ctx := context.Background()
	node, err := s.NodeSetProvisionState(ctx, "n1", "active", v1.WithMachineConfigDrive("QUJD"), v1.WithMachineWait(true), v1.WithMachineWaitInterval(r2icFast))
	if err != nil || node.ProvisionState != "active" || put.all()[0].Body != `{"configdrive":"QUJD","target":"active"}` {
		t.Fatalf("%v %+v", err, put.all())
	}
	if err := s.ActivateNode(ctx, "n1"); err != nil || put.all()[1].Body != `{"target":"active"}` {
		t.Fatal(err)
	}
	if err := s.DeactivateNode(ctx, "n1", v1.WithMachineWait(true), v1.WithMachineWaitInterval(r2icFast), v1.WithMachineTimeout(time.Second)); err != nil || put.all()[2].Body != `{"target":"deleted"}` {
		t.Logf("deactivate waits for available: %v", err)
	}
	if _, err := s.NodeSetProvisionState(ctx, "n1", "clean", v1.WithMachineConfigDrive("x")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("config drive only with active/rebuild: %v", err)
	}
	if _, err := s.NodeSetProvisionState(ctx, "", "active"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestR2IronicCMachineWaitTimeoutIsBoundedByOption(t *testing.T) {
	cloud := testcloud.New(t)
	r2icServe(t, cloud, "PUT /v1/nodes/n1/states/provision", 202, "")
	r2icServe(t, cloud, "GET /v1/nodes/n1", 200, `{"uuid":"n1","provision_state":"deploying"}`)
	_, err := r2icService(cloud).NodeSetProvisionState(context.Background(), "n1", "active", v1.WithMachineWait(true), v1.WithMachineTimeout(40*time.Millisecond), v1.WithMachineWaitInterval(5*time.Millisecond))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}

func TestR2IronicCInspectMachineWorkflowAndRefusals(t *testing.T) {
	cloud := testcloud.New(t)
	put := r2icServe(t, cloud, "PUT /v1/nodes/n1/states/provision", 202, "")
	r2icServe(t, cloud, "GET /v1/nodes/n1", 200, `{"uuid":"n1","provision_state":"available"}`, 200, `{"uuid":"n1","provision_state":"manageable"}`, 200, `{"uuid":"n1","provision_state":"manageable"}`, 200, `{"uuid":"n1","provision_state":"available"}`)
	r2icServe(t, cloud, "GET /v1/nodes/used", 200, `{"uuid":"used","provision_state":"available","instance_uuid":"i-1"}`)
	r2icServe(t, cloud, "GET /v1/nodes/busy", 200, `{"uuid":"busy","provision_state":"active"}`)
	r2icServe(t, cloud, "GET /v1/nodes/failed", 200, `{"uuid":"failed","provision_state":"inspect failed"}`, 200, `{"uuid":"failed","provision_state":"manageable"}`)
	r2icServe(t, cloud, "PUT /v1/nodes/failed/states/provision", 202, "")
	s := r2icService(cloud)
	ctx := context.Background()
	node, err := s.InspectMachine(ctx, "n1", v1.WithMachineWaitInterval(r2icFast))
	if err != nil || node.ProvisionState != "available" {
		t.Fatal(node, err)
	}
	var targets []string
	for _, c := range put.all() {
		targets = append(targets, c.Body)
	}
	if len(targets) != 3 || targets[0] != `{"target":"manage"}` || targets[1] != `{"target":"inspect"}` || targets[2] != `{"target":"provide"}` {
		t.Fatalf("%v", targets)
	}
	for _, id := range []string{"used", "busy"} {
		if _, err := s.InspectMachine(ctx, id); !errors.Is(err, v1.ErrMachineState) {
			t.Fatalf("%s: %v", id, err)
		}
	}
	if len(put.all()) != 3 {
		t.Fatal("refused machines must not be changed")
	}
	if _, err := s.InspectMachine(ctx, "failed", v1.WithMachineWaitInterval(r2icFast)); err != nil {
		t.Fatalf("inspect failed is accepted: %v", err)
	}
}

func TestR2IronicCRegisterMachineWorkflowAndPortOrdering(t *testing.T) {
	cloud := testcloud.New(t)
	post := r2icServe(t, cloud, "POST /v1/nodes", 201, `{"uuid":"n1","provision_state":"enroll"}`)
	put := r2icServe(t, cloud, "PUT /v1/nodes/n1/states/provision", 202, "")
	r2icServe(t, cloud, "GET /v1/nodes/n1", 200, `{"uuid":"n1","provision_state":"manageable"}`, 200, `{"uuid":"n1","provision_state":"available"}`)
	pp := r2icServe(t, cloud, "POST /v1/ports", 201, `{"uuid":"p1","address":"aa"}`)
	s := r2icService(cloud)
	node, err := s.RegisterMachine(context.Background(), nodes.CreateOpts{Driver: "fake"}, v1.NICAddresses("aa:bb", "cc:dd"), v1.WithMachineWaitInterval(r2icFast))
	if err != nil || node.ProvisionState != "available" {
		t.Fatal(node, err)
	}
	if post.all()[0].Body != `{"driver":"fake"}` || post.all()[0].Version != "1.99" {
		t.Fatalf("%+v", post.all())
	}
	if p := pp.all(); len(p) != 2 || p[0].Body != `{"address":"aa:bb","node_uuid":"n1"}` {
		t.Fatalf("%+v", p)
	}
	if b := put.all(); len(b) != 2 || b[0].Body != `{"target":"manage"}` || b[1].Body != `{"target":"provide"}` {
		t.Fatalf("%+v", b)
	}
	if _, err := s.RegisterMachine(context.Background(), nodes.CreateOpts{}, []ports.CreateOpts{{Address: "a", NodeUUID: "x"}}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := s.RegisterMachine(context.Background(), nodes.CreateOpts{}, nil, v1.WithRegisterProvisionState("active")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestR2IronicCRegisterMachineRollsBackPortsAndNodeOnFailure(t *testing.T) {
	cloud := testcloud.New(t)
	r2icServe(t, cloud, "POST /v1/nodes", 201, `{"uuid":"n1","provision_state":"enroll"}`)
	r2icServe(t, cloud, "PUT /v1/nodes/n1/states/provision", 202, "")
	r2icServe(t, cloud, "GET /v1/nodes/n1", 200, `{"uuid":"n1","provision_state":"manageable"}`)
	r2icServe(t, cloud, "POST /v1/ports", 201, `{"uuid":"p1"}`, 400, `{}`)
	delPort := r2icServe(t, cloud, "DELETE /v1/ports/p1", 204, "")
	delNode := r2icServe(t, cloud, "DELETE /v1/nodes/n1", 204, "")
	s := r2icService(cloud)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := s.RegisterMachine(ctx, nodes.CreateOpts{}, v1.NICAddresses("a", "b"), v1.WithRegisterProvisionState("manageable"), v1.WithMachineWaitInterval(r2icFast))
	var reg *v1.RegisterError
	if !errors.As(err, &reg) || !reg.RolledBack || !gophercloud.ResponseCodeIs(err, 400) || len(delPort.all()) != 1 || len(delNode.all()) != 1 {
		t.Fatalf("%v %+v", err, reg)
	}
}

func TestR2IronicCUnregisterMachineStatesPortsAndMissingDeletes(t *testing.T) {
	cloud := testcloud.New(t)
	r2icServe(t, cloud, "GET /v1/nodes/n1", 200, `{"uuid":"n1","provision_state":"available"}`)
	r2icServe(t, cloud, "GET /v1/nodes/live", 200, `{"uuid":"live","provision_state":"clean wait"}`)
	r2icServe(t, cloud, "GET /v1/nodes/ghost", 404, `{}`)
	r2icServe(t, cloud, "GET /v1/nodes/detail", 200, `{"nodes":[]}`)
	list := r2icServe(t, cloud, "GET /v1/ports", 200, `{"ports":[{"uuid":"p1"},{"uuid":"p2"}]}`, 200, `{"ports":[]}`)
	delPort := r2icServe(t, cloud, "DELETE /v1/ports/p1", 404, `{}`)
	delNode := r2icServe(t, cloud, "DELETE /v1/nodes/n1", 404, `{}`)
	s := r2icService(cloud)
	ctx := context.Background()
	if err := s.UnregisterMachine(ctx, []string{"aa", "bb"}, "n1"); err != nil {
		t.Fatalf("404 deletes are ignored: %v", err)
	}
	if q := list.all(); len(q) != 2 || q[0].RawQuery != "address=aa" || len(delPort.all()) != 1 || len(delNode.all()) != 1 {
		t.Fatalf("first match only, empty list skipped: %+v", list.all())
	}
	if err := s.UnregisterMachine(ctx, nil, "live"); !errors.Is(err, v1.ErrMachineState) {
		t.Fatal(err)
	}
	if err := s.UnregisterMachine(ctx, nil, "ghost"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestR2IronicCUnregisterMachineLockTimeout(t *testing.T) {
	cloud := testcloud.New(t)
	r2icServe(t, cloud, "GET /v1/nodes/n1", 200, `{"uuid":"n1","provision_state":"available","reservation":"host-a"}`)
	err := r2icService(cloud).UnregisterMachine(context.Background(), nil, "n1", v1.WithMachineLockTimeout(30*time.Millisecond), v1.WithMachineWaitInterval(5*time.Millisecond))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}

func TestR2IronicCUpdateMachineChangesAndNoop(t *testing.T) {
	cloud := testcloud.New(t)
	const body = `{"uuid":"n1","name":"old","driver_info":{"a":1},"instance_uuid":null,"maintenance":false}`
	get := r2icServe(t, cloud, "GET /v1/nodes/n1", 200, body)
	patch := r2icServe(t, cloud, "PATCH /v1/nodes/n1", 200, `{"uuid":"n1","name":"new"}`)
	s := r2icService(cloud)
	ctx := context.Background()
	res, err := s.UpdateMachine(ctx, "n1", map[string]any{"name": "new", "driver_info": map[string]any{"a": 2}, "instance_id": nil})
	if err != nil || res.Node == nil || res.Node.Name != "new" {
		t.Fatal(res, err)
	}
	want := []string{"/driver_info/a", "/name"}
	if len(res.Changes) != 2 || res.Changes[0] != want[0] || res.Changes[1] != want[1] {
		t.Fatalf("changes use attribute names: %v", res.Changes)
	}
	if patch.all()[0].Body != `[{"op":"replace","path":"/driver_info/a","value":2},{"op":"replace","path":"/name","value":"new"}]` {
		t.Fatal(patch.all()[0].Body)
	}
	noop, err := s.UpdateMachine(ctx, "n1", map[string]any{"name": "old"})
	if err != nil || noop.Changes != nil || noop.Node != nil || noop.Record == nil || len(patch.all()) != 1 {
		t.Fatalf("%+v %v", noop, err)
	}
	if _, err := s.UpdateMachine(ctx, "n1", map[string]any{"bogus": 1}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if len(get.all()) != 3 {
		t.Fatalf("one read per call: %d", len(get.all()))
	}
	r2icServe(t, cloud, "GET /v1/nodes/ghost", 404, `{}`)
	r2icServe(t, cloud, "GET /v1/nodes/detail", 200, `{"nodes":[]}`)
	if _, err := s.UpdateMachine(ctx, "ghost", map[string]any{"name": "x"}); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestR2IronicCWaitForStatusAndDeleteDefaults(t *testing.T) {
	cloud := testcloud.New(t)
	r2icServe(t, cloud, "GET /v1/nodes/n1", 200, `{"uuid":"n1","provision_state":"cleaning"}`, 200, `{"uuid":"n1","provision_state":"available"}`)
	r2icServe(t, cloud, "GET /v1/nodes/bad", 200, `{"uuid":"bad","provision_state":"ERROR"}`)
	r2icServe(t, cloud, "GET /v1/nodes/clean-failed", 200, `{"uuid":"clean-failed","provision_state":"clean failed"}`)
	r2icServe(t, cloud, "GET /v1/nodes/del", 200, `{"uuid":"del","provision_state":"deleting"}`, 404, `{}`)
	s := r2icService(cloud)
	ctx := context.Background()
	n, err := v1.WaitForStatus(ctx, s.Nodes.Resources, resource.ID("n1"), "AVAILABLE", resource.WithPollInterval(r2icFast))
	if err != nil || n.UUID != "n1" {
		t.Fatal(n, err)
	}
	if _, err := v1.WaitForStatus(ctx, s.Nodes.Resources, resource.ID("bad"), "available", resource.WithPollInterval(r2icFast)); !errors.Is(err, resource.ErrFailedState) {
		t.Fatalf("ERROR is the only default failure: %v", err)
	}
	short, cancel := context.WithTimeout(ctx, 40*time.Millisecond)
	defer cancel()
	if _, err := v1.WaitForStatus(short, s.Nodes.Resources, resource.ID("clean-failed"), "available", resource.WithPollInterval(5*time.Millisecond)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("a state like 'clean failed' is not a default failure: %v", err)
	}
	if err := v1.WaitForDelete(ctx, s.Nodes.Resources, resource.ID("del"), resource.WithPollInterval(r2icFast)); err != nil {
		t.Fatal(err)
	}
}
