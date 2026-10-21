package api_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const r2icFast = time.Millisecond

func TestR2IronicCSetProvisionStateBodyVersionAndReadBack(t *testing.T) {
	cloud := testcloud.New(t)
	put := r2icServe(t, cloud, "PUT /v1/nodes/n1/states/provision", 202, "")
	get := r2icServe(t, cloud, "GET /v1/nodes/n1", 200, `{"uuid":"n1","provision_state":"cleaning"}`)
	api := nodes.New(r2icClient(cloud, "1.99"))
	ctx := context.Background()
	empty := ""
	node, err := api.SetProvisionState(ctx, "n1", nodes.ProvisionStateRequest{Target: "clean", CleanSteps: []map[string]any{}, RescuePassword: nil})
	if err != nil || node.UUID != "n1" || len(get.all()) != 1 {
		t.Fatal(node, err)
	}
	if _, err := api.SetProvisionState(ctx, "n1", nodes.ProvisionStateRequest{Target: "rescue", RescuePassword: &empty}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.SetProvisionState(ctx, "n1", nodes.ProvisionStateRequest{Target: "active", ConfigDrive: map[string]any{"meta_data": map[string]any{"a": 1}}, DeploySteps: []map[string]any{{"interface": "deploy"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.SetProvisionState(ctx, "n1", nodes.ProvisionStateRequest{Target: "active", ConfigDrive: []byte("QUJD")}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.SetProvisionState(ctx, "n1", nodes.ProvisionStateRequest{Target: "service", Runbook: "rb"}); err != nil {
		t.Fatal(err)
	}
	calls := put.all()
	want := []string{
		`{"clean_steps":[],"target":"clean"}`,
		`{"rescue_password":"","target":"rescue"}`,
		`{"configdrive":{"meta_data":{"a":1}},"deploy_steps":[{"interface":"deploy"}],"target":"active"}`,
		`{"configdrive":"QUJD","target":"active"}`,
		`{"runbook":"rb","target":"service"}`,
	}
	for i, w := range want {
		if calls[i].Body != w {
			t.Fatalf("body %d: %s", i, calls[i].Body)
		}
	}
}

func TestR2IronicCSetProvisionStateValidationBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	ironicNodeUnreachable(t, cloud)
	api := nodes.New(r2icClient(cloud, "1.99"))
	ctx := context.Background()
	pw := "x"
	for name, req := range map[string]nodes.ProvisionStateRequest{
		"drive target":      {Target: "clean", ConfigDrive: "x"},
		"clean steps":       {Target: "active", CleanSteps: []map[string]any{}},
		"deploy steps":      {Target: "clean", DeploySteps: []map[string]any{{}}},
		"service steps":     {Target: "clean", ServiceSteps: []map[string]any{}},
		"rescue password":   {Target: "clean", RescuePassword: &pw},
		"runbook+steps":     {Target: "clean", Runbook: "r", CleanSteps: []map[string]any{}},
		"runbook target":    {Target: "active", Runbook: "r"},
		"bad drive type":    {Target: "active", ConfigDrive: 7},
		"invalid utf8 byte": {Target: "active", ConfigDrive: []byte{0xff}},
	} {
		if _, err := api.SetProvisionState(ctx, "n1", req); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := api.SetProvisionState(ctx, "n1", nodes.ProvisionStateRequest{Target: "abort"}, nodes.WithProvisionStateWait()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Errorf("unknown expected state: %v", err)
	}
	// Microversion minimum: clean needs 1.15, a dict config drive 1.56.
	old := nodes.New(r2icClient(cloud, "1.14"))
	if _, err := old.SetProvisionState(ctx, "n1", nodes.ProvisionStateRequest{Target: "clean"}); !errors.Is(err, resource.ErrUnsupported) {
		t.Errorf("clean on 1.14: %v", err)
	}
	mid := nodes.New(r2icClient(cloud, "1.55"))
	if _, err := mid.SetProvisionState(ctx, "n1", nodes.ProvisionStateRequest{Target: "active", ConfigDrive: map[string]any{"a": 1}}); !errors.Is(err, resource.ErrUnsupported) {
		t.Errorf("dict drive on 1.55: %v", err)
	}
	if _, err := mid.SetProvisionState(ctx, "n1", nodes.ProvisionStateRequest{Target: "rebuild", ConfigDrive: "x", DeploySteps: []map[string]any{{"a": 1}}}); !errors.Is(err, resource.ErrUnsupported) {
		t.Errorf("deploy steps on 1.55: %v", err)
	}
	if _, err := nodes.New(r2icClient(cloud, "1.91")).SetProvisionState(ctx, "n1", nodes.ProvisionStateRequest{Target: "clean", Runbook: "r"}); !errors.Is(err, resource.ErrUnsupported) {
		t.Errorf("runbook on 1.91: %v", err)
	}
}

func TestR2IronicCSetProvisionStateDiscoversVersionAndKeepsConflict(t *testing.T) {
	cloud := testcloud.New(t)
	r2icDiscovery(t, cloud, "1.1", "1.120")
	put := r2icServe(t, cloud, "PUT /v1/nodes/n1/states/provision", 409, `{}`)
	r2icServe(t, cloud, "GET /v1/nodes/n1", 200, `{"uuid":"n1"}`)
	api := nodes.New(r2icClient(cloud, ""))
	_, err := api.SetProvisionState(context.Background(), "n1", nodes.ProvisionStateRequest{Target: "provide"})
	if !gophercloud.ResponseCodeIs(err, 409) || len(put.all()) != 1 || put.all()[0].Version != "1.109" {
		t.Fatalf("%v %+v", err, put.all())
	}
}

func TestR2IronicCSetProvisionStateWaitDetectsFailure(t *testing.T) {
	cloud := testcloud.New(t)
	r2icServe(t, cloud, "PUT /v1/nodes/n1/states/provision", 202, "")
	get := r2icServe(t, cloud, "GET /v1/nodes/n1", 200, `{"uuid":"n1","provision_state":"deploying"}`, 200, `{"uuid":"n1","provision_state":"deploy failed","last_error":"boom"}`)
	api := nodes.New(r2icClient(cloud, "1.99"))
	_, err := api.SetProvisionState(context.Background(), "n1", nodes.ProvisionStateRequest{Target: "active"}, nodes.WithProvisionStateWait(nodes.WithProvisionWaitInterval(r2icFast)))
	var fail *nodes.ProvisionFailureError
	if !errors.As(err, &fail) || !errors.Is(err, resource.ErrFailedState) || fail.LastError != "boom" || fail.State != "deploy failed" || len(get.all()) != 2 {
		t.Fatalf("%v calls=%d", err, len(get.all()))
	}
}

func TestR2IronicCStateReachedRules(t *testing.T) {
	cases := []struct {
		state, last, want string
		abort             bool
		reached, fails    bool
	}{
		{"active", "", "active", true, true, false},
		{"", "", "available", true, true, false},
		{"", "", "active", true, false, false},
		{"error", "e", "active", true, false, true},
		{"error", "e", "active", false, false, false},
		{"clean failed", "e", "available", true, false, true},
		{"enroll", "creds", "manageable", true, false, true},
		{"enroll", "", "manageable", true, false, false},
		{"enroll", "creds", "manageable", false, false, false},
		{"enroll", "creds", "active", true, false, false},
		{"manageable", "", "manageable", true, true, false},
	}
	for _, c := range cases {
		got, err := nodes.StateReached(&nodes.Node{UUID: "n", ProvisionState: c.state, LastError: c.last}, c.want, c.abort)
		if got != c.reached || (err != nil) != c.fails {
			t.Errorf("%+v: got=%v err=%v", c, got, err)
		}
	}
}

func TestR2IronicCAwaitProvisionStatesMultiNodeResults(t *testing.T) {
	cloud := testcloud.New(t)
	r2icServe(t, cloud, "GET /v1/nodes/ok", 200, `{"uuid":"ok","provision_state":"manageable"}`)
	slow := r2icServe(t, cloud, "GET /v1/nodes/slow", 200, `{"uuid":"slow","provision_state":"manageing"}`, 200, `{"uuid":"slow","provision_state":"manageable"}`)
	r2icServe(t, cloud, "GET /v1/nodes/bad", 200, `{"uuid":"bad","provision_state":"enroll","last_error":"creds"}`)
	r2icServe(t, cloud, "GET /v1/nodes/stuck", 200, `{"uuid":"stuck","provision_state":"verifying"}`)
	api := nodes.New(r2icClient(cloud, "1.99"))
	ctx := context.Background()
	res, err := api.AwaitProvisionStates(ctx, []string{"ok", "slow"}, "manageable", nodes.WithProvisionWaitInterval(r2icFast))
	if err != nil || len(res.Success) != 2 || res.Success[0].UUID != "ok" || res.Success[1].UUID != "slow" || len(slow.all()) != 2 {
		t.Fatal(res, err)
	}
	res, err = api.AwaitProvisionStates(ctx, []string{"ok", "bad"}, "manageable", nodes.WithProvisionWaitInterval(r2icFast))
	var fail *nodes.ProvisionFailureError
	if !errors.As(err, &fail) || !fail.Manage || len(res.Success) != 1 {
		t.Fatalf("fail-fast: %v %+v", err, res)
	}
	res, err = api.AwaitProvisionStates(ctx, []string{"ok", "bad"}, "manageable", nodes.WithContinueOnFailure(), nodes.WithProvisionWaitInterval(r2icFast))
	if err != nil || len(res.Success) != 1 || len(res.Failure) != 1 || res.Failure[0].UUID != "bad" || len(res.Timeout) != 0 {
		t.Fatalf("collect: %v %+v", err, res)
	}
	short, cancel := context.WithTimeout(ctx, 30*time.Millisecond)
	defer cancel()
	res, err = api.AwaitProvisionStates(short, []string{"ok", "stuck"}, "manageable", nodes.WithContinueOnFailure(), nodes.WithProvisionWaitInterval(5*time.Millisecond))
	if err != nil || len(res.Success) != 1 || len(res.Timeout) != 1 || res.Timeout[0].UUID != "stuck" {
		t.Fatalf("timeout: %v %+v", err, res)
	}
	short2, cancel2 := context.WithTimeout(ctx, 30*time.Millisecond)
	defer cancel2()
	if _, err = api.AwaitProvisionStates(short2, []string{"stuck"}, "manageable", nodes.WithProvisionWaitInterval(5*time.Millisecond)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline as error: %v", err)
	}
}

func TestR2IronicCSetPowerStateWaitAndSoftVersion(t *testing.T) {
	cloud := testcloud.New(t)
	put := r2icServe(t, cloud, "PUT /v1/nodes/n1/states/power", 202, "")
	r2icServe(t, cloud, "GET /v1/nodes/n1", 200, `{"uuid":"n1","power_state":"power off"}`, 200, `{"uuid":"n1","power_state":"power on"}`)
	api := nodes.New(r2icClient(cloud, "1.99"))
	ctx := context.Background()
	if err := api.SetPowerState(ctx, "n1", "rebooting", nodes.WithPowerStateWait(nodes.WithWaitInterval(r2icFast))); err != nil {
		t.Fatal(err)
	}
	if put.all()[0].Body != `{"target":"rebooting"}` {
		t.Fatal(put.all())
	}
	if err := api.SetPowerState(ctx, "n1", "hibernate", nodes.WithPowerStateWait()); !errors.Is(err, resource.ErrInvalidOption) || len(put.all()) != 1 {
		t.Fatalf("unknown target with wait: %v", err)
	}
	if err := api.SetPowerState(ctx, "n1", "hibernate"); err != nil {
		t.Fatalf("unknown target without wait is sent: %v", err)
	}
	if err := nodes.New(r2icClient(cloud, "1.26")).SetPowerState(ctx, "n1", "soft power off"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("soft needs 1.27: %v", err)
	}
	if err := nodes.New(r2icClient(cloud, "1.27")).SetPowerState(ctx, "n1", "soft power off"); err != nil {
		t.Fatal(err)
	}
}

func TestR2IronicCCreateWithProvisionStateMicroversionBodyAndManage(t *testing.T) {
	cloud := testcloud.New(t)
	post := r2icServe(t, cloud, "POST /v1/nodes", 201, `{"uuid":"n1","provision_state":"enroll"}`)
	put := r2icServe(t, cloud, "PUT /v1/nodes/n1/states/provision", 202, "")
	r2icServe(t, cloud, "GET /v1/nodes/n1", 200, `{"uuid":"n1","provision_state":"manageable"}`)
	ctx := context.Background()
	api := nodes.New(r2icClient(cloud, "1.99"))
	if _, err := api.CreateWithProvisionState(ctx, nodes.CreateEnroll, nodes.CreateOpts{Driver: "fake", Name: "x"}); err != nil {
		t.Fatal(err)
	}
	if c := post.all()[0]; c.Body != `{"driver":"fake","name":"x"}` || c.Version != "1.99" {
		t.Fatalf("provision_state is never sent: %+v", c)
	}
	node, err := api.CreateWithProvisionState(ctx, nodes.CreateManageable, nodes.CreateOpts{Driver: "fake"})
	if err != nil || node.ProvisionState != "manageable" || len(put.all()) != 1 || put.all()[0].Body != `{"target":"manage"}` {
		t.Fatalf("manage follow-up: %v %+v %+v", err, node, put.all())
	}
	if _, err := api.CreateWithProvisionState(ctx, nodes.CreateAvailable, nodes.CreateOpts{Driver: "fake"}); err != nil {
		t.Fatal(err)
	}
	if c := post.all()[len(post.all())-1]; c.Version != "1.10" {
		t.Fatalf("available lowers the version to 1.10: %+v", c)
	}
	if _, err := api.CreateWithProvisionState(ctx, "active", nodes.CreateOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	n := len(post.all())
	if _, err := nodes.New(r2icClient(cloud, "1.10")).CreateWithProvisionState(ctx, nodes.CreateEnroll, nodes.CreateOpts{}); !errors.Is(err, resource.ErrUnsupported) || len(post.all()) != n {
		t.Fatalf("enroll needs 1.11: %v", err)
	}
}
