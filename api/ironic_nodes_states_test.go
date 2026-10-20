package api_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestIronicNodeSuccessCodesAreExactForStateAndManagementCalls(t *testing.T) {
	ironicNodeRunStatusCases(t, []ironicNodeStatusCase{
		{name: "ChangeProvisionState", pattern: "PUT /v1/nodes/n-1/states/provision", ok: []int{202}, bad: []int{200, 204},
			call: func(ctx context.Context, api *nodes.API) error {
				return api.ChangeProvisionState(ctx, "n-1", nodes.ProvisionStateOpts{Target: nodes.TargetProvide})
			}},
		{name: "ChangePowerState", pattern: "PUT /v1/nodes/n-1/states/power", ok: []int{202}, bad: []int{200, 204},
			call: func(ctx context.Context, api *nodes.API) error {
				return api.ChangePowerState(ctx, "n-1", nodes.PowerStateOpts{Target: nodes.PowerOn})
			}},
		{name: "SetMaintenance", pattern: "PUT /v1/nodes/n-1/maintenance", ok: []int{202}, bad: []int{200, 204},
			call: func(ctx context.Context, api *nodes.API) error {
				return api.SetMaintenance(ctx, "n-1", nodes.MaintenanceOpts{})
			}},
		{name: "UnsetMaintenance", pattern: "DELETE /v1/nodes/n-1/maintenance", ok: []int{202}, bad: []int{200, 204},
			call: func(ctx context.Context, api *nodes.API) error { return api.UnsetMaintenance(ctx, "n-1") }},
		{name: "SetBootDevice", pattern: "PUT /v1/nodes/n-1/management/boot_device", ok: []int{204}, bad: []int{200, 202},
			call: func(ctx context.Context, api *nodes.API) error {
				return api.SetBootDevice(ctx, "n-1", nodes.BootDeviceOpts{BootDevice: "pxe"})
			}},
		{name: "GetBootDevice", pattern: "GET /v1/nodes/n-1/management/boot_device", ok: []int{200}, bad: []int{202, 204}, body: `{"boot_device":"pxe","persistent":false}`,
			call: func(ctx context.Context, api *nodes.API) error { _, err := api.GetBootDevice(ctx, "n-1"); return err }},
		{name: "GetSupportedBootDevices", pattern: "GET /v1/nodes/n-1/management/boot_device/supported", ok: []int{200}, bad: []int{202, 204}, body: `{"supported_boot_devices":["pxe"]}`,
			call: func(ctx context.Context, api *nodes.API) error {
				_, err := api.GetSupportedBootDevices(ctx, "n-1")
				return err
			}},
		{name: "InjectNMI", pattern: "PUT /v1/nodes/n-1/management/inject_nmi", ok: []int{204}, bad: []int{200, 202},
			call: func(ctx context.Context, api *nodes.API) error { return api.InjectNMI(ctx, "n-1") }},
		{name: "SetRAIDConfig", pattern: "PUT /v1/nodes/n-1/states/raid", ok: []int{204}, bad: []int{200, 202},
			call: func(ctx context.Context, api *nodes.API) error {
				return api.SetRAIDConfig(ctx, "n-1", nodes.RAIDConfigOpts{LogicalDisks: []nodes.LogicalDisk{{RAIDLevel: nodes.RAID1}}})
			}},
	})
}

func TestIronicNodeChangeProvisionStateSendsStepsConfigDriveAndExtensions(t *testing.T) {
	cloud := testcloud.New(t)
	recorder := ironicNodeServe(t, cloud, "PUT /v1/nodes/n-1/states/provision", 202, "")
	err := newIronicNodes(t, cloud).ChangeProvisionState(context.Background(), "n-1", nodes.ProvisionStateOpts{
		Target:      nodes.TargetClean,
		CleanSteps:  []nodes.CleanStep{{Interface: nodes.InterfaceDeploy, Step: "erase_devices", Args: map[string]any{"verify": false}}},
		DeploySteps: []nodes.DeployStep{{Interface: nodes.InterfaceDeploy, Step: "deploy", Args: map[string]any{}, Priority: 0}},
		ConfigDrive: nodes.ConfigDrive{UserData: "#cloud-config"},
	}, nodes.WithChangeProvisionStateField("vendor", map[string]bool{"fast": false}))
	if err != nil {
		t.Fatal(err)
	}
	call := recorder.only(t)
	want := map[string]string{
		"target":       `"clean"`,
		"clean_steps":  `[{"args":{"verify":false},"interface":"deploy","step":"erase_devices"}]`,
		"deploy_steps": `[{"args":{},"interface":"deploy","priority":0,"step":"deploy"}]`,
		"configdrive":  `{"user_data":"#cloud-config"}`,
		"vendor":       `{"fast":false}`,
	}
	if got := ironicNodeFields(t, call.Body); !reflect.DeepEqual(got, want) {
		t.Fatalf("body=%v", got)
	}
}

func TestIronicNodeChangeProvisionStateOmitsUnsetValuesAndRejectsBadInputBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	recorder := ironicNodeServe(t, cloud, "PUT /v1/nodes/n-1/states/provision", 202, "")
	api := newIronicNodes(t, cloud)
	if err := api.ChangeProvisionState(context.Background(), "n-1", nodes.ProvisionStateOpts{Target: nodes.TargetRescue, RescuePassword: "pw"}); err != nil {
		t.Fatal(err)
	}
	if got := ironicNodeFields(t, recorder.only(t).Body); !reflect.DeepEqual(got, map[string]string{"target": `"rescue"`, "rescue_password": `"pw"`}) {
		t.Fatalf("body=%v", got)
	}

	blocked := testcloud.New(t)
	ironicNodeUnreachable(t, blocked)
	guarded := newIronicNodes(t, blocked)
	ctx := context.Background()
	if err := guarded.ChangeProvisionState(ctx, "n-1", nodes.ProvisionStateOpts{}); err == nil {
		t.Fatal("missing target must fail before HTTP")
	}
	if err := guarded.ChangeProvisionState(ctx, "n-1", nodes.ProvisionStateOpts{Target: nodes.TargetActive}, nodes.WithChangeProvisionStateField("target", "deleted")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected field: %v", err)
	}
	if err := guarded.ChangeProvisionState(ctx, "n-1", nodes.ProvisionStateOpts{Target: nodes.TargetActive}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option: %v", err)
	}
}

func TestIronicNodeProvisionAndPowerAcknowledgementIsNotCompletionAndNeverReplays(t *testing.T) {
	cloud := testcloud.New(t)
	provision := ironicNodeServe(t, cloud, "PUT /v1/nodes/n-1/states/provision", 202, "")
	power := ironicNodeServe(t, cloud, "PUT /v1/nodes/n-1/states/power", 202, "")
	conflict := ironicNodeServe(t, cloud, "PUT /v1/nodes/n-2/states/provision", 409, `{"error_message":"locked"}`)
	unavailable := ironicNodeServe(t, cloud, "PUT /v1/nodes/n-3/states/power", 503, `{"error_message":"busy"}`)
	other := ironicNodeServe(t, cloud, "GET /v1/nodes/n-1", 200, ironicNodeBody)
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	if err := api.ChangeProvisionState(ctx, "n-1", nodes.ProvisionStateOpts{Target: nodes.TargetActive}); err != nil {
		t.Fatal(err)
	}
	if err := api.ChangePowerState(ctx, "n-1", nodes.PowerStateOpts{Target: nodes.Rebooting}); err != nil {
		t.Fatal(err)
	}
	if provision.count() != 1 || power.count() != 1 || other.count() != 0 {
		t.Fatalf("an acknowledgement must not poll: provision=%d power=%d get=%d", provision.count(), power.count(), other.count())
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if err := api.ChangeProvisionState(ctx, "n-2", nodes.ProvisionStateOpts{Target: nodes.TargetActive}); !errors.As(err, &unexpected) || unexpected.Actual != 409 {
		t.Fatalf("conflict: %v", err)
	}
	if err := api.ChangePowerState(ctx, "n-3", nodes.PowerStateOpts{Target: nodes.PowerOff}); !errors.As(err, &unexpected) || unexpected.Actual != 503 {
		t.Fatalf("unavailable: %v", err)
	}
	if conflict.count() != 1 || unavailable.count() != 1 {
		t.Fatalf("a failed state change was replayed: %d %d", conflict.count(), unavailable.count())
	}
}

func TestIronicNodeChangePowerStateBodyKeepsTargetAndOmitsZeroTimeout(t *testing.T) {
	cloud := testcloud.New(t)
	recorder := ironicNodeServe(t, cloud, "PUT /v1/nodes/n-1/states/power", 202, "")
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	if err := api.ChangePowerState(ctx, "n-1", nodes.PowerStateOpts{Target: nodes.SoftRebooting, Timeout: 30}, nodes.WithChangePowerStateField("vendor", nil)); err != nil {
		t.Fatal(err)
	}
	if err := api.ChangePowerState(ctx, "n-1", nodes.PowerStateOpts{Target: nodes.PowerOff}); err != nil {
		t.Fatal(err)
	}
	calls := recorder.all()
	if len(calls) != 2 {
		t.Fatalf("calls=%+v", calls)
	}
	if got := ironicNodeFields(t, calls[0].Body); !reflect.DeepEqual(got, map[string]string{"target": `"soft rebooting"`, "timeout": `30`, "vendor": `null`}) {
		t.Fatalf("first body=%v", got)
	}
	if got := ironicNodeFields(t, calls[1].Body); !reflect.DeepEqual(got, map[string]string{"target": `"power off"`}) {
		t.Fatalf("second body=%v", got)
	}
	blocked := testcloud.New(t)
	ironicNodeUnreachable(t, blocked)
	guarded := newIronicNodes(t, blocked)
	if err := guarded.ChangePowerState(ctx, "n-1", nodes.PowerStateOpts{}); err == nil {
		t.Fatal("missing target must fail before HTTP")
	}
	if err := guarded.ChangePowerState(ctx, "n-1", nodes.PowerStateOpts{Target: nodes.PowerOn}, nodes.WithChangePowerStateField("target", "x")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected field: %v", err)
	}
}

func TestIronicNodeMaintenanceBodyAndNoBodyUnset(t *testing.T) {
	cloud := testcloud.New(t)
	set := ironicNodeServe(t, cloud, "PUT /v1/nodes/n-1/maintenance", 202, "")
	unset := ironicNodeServe(t, cloud, "DELETE /v1/nodes/n-1/maintenance", 202, "")
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	if err := api.SetMaintenance(ctx, "n-1", nodes.MaintenanceOpts{Reason: "rack work"}, nodes.WithSetMaintenanceField("ticket", 0)); err != nil {
		t.Fatal(err)
	}
	if err := api.SetMaintenance(ctx, "n-1", nodes.MaintenanceOpts{}); err != nil {
		t.Fatal(err)
	}
	calls := set.all()
	if got := ironicNodeFields(t, calls[0].Body); !reflect.DeepEqual(got, map[string]string{"reason": `"rack work"`, "ticket": `0`}) {
		t.Fatalf("body=%v", got)
	}
	if got := ironicNodeFields(t, calls[1].Body); len(got) != 0 {
		t.Fatalf("empty reason must be omitted: %v", got)
	}
	if err := api.UnsetMaintenance(ctx, "n-1"); err != nil {
		t.Fatal(err)
	}
	if call := unset.only(t); call.Body != "" {
		t.Fatalf("unset body=%q", call.Body)
	}
	blocked := testcloud.New(t)
	ironicNodeUnreachable(t, blocked)
	if err := newIronicNodes(t, blocked).SetMaintenance(ctx, "n-1", nodes.MaintenanceOpts{}, nodes.WithSetMaintenanceField("reason", "x")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected field: %v", err)
	}
}

func TestIronicNodeBootDeviceKeepsFalsePersistentAndDecodesReads(t *testing.T) {
	cloud := testcloud.New(t)
	set := ironicNodeServe(t, cloud, "PUT /v1/nodes/n-1/management/boot_device", 204, "")
	ironicNodeServe(t, cloud, "GET /v1/nodes/n-1/management/boot_device", 200, `{"boot_device":"disk","persistent":true}`)
	ironicNodeServe(t, cloud, "GET /v1/nodes/n-1/management/boot_device/supported", 200, `{"supported_boot_devices":["pxe","disk"]}`)
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	if err := api.SetBootDevice(ctx, "n-1", nodes.BootDeviceOpts{BootDevice: "pxe"}, nodes.WithSetBootDeviceField("vendor", false)); err != nil {
		t.Fatal(err)
	}
	if got := ironicNodeFields(t, set.only(t).Body); !reflect.DeepEqual(got, map[string]string{"boot_device": `"pxe"`, "persistent": `false`, "vendor": `false`}) {
		t.Fatalf("body=%v", got)
	}
	device, err := api.GetBootDevice(ctx, "n-1")
	if err != nil || device.BootDevice != "disk" || !device.Persistent {
		t.Fatalf("device=%+v error=%v", device, err)
	}
	supported, err := api.GetSupportedBootDevices(ctx, "n-1")
	if err != nil || !reflect.DeepEqual(supported, []string{"pxe", "disk"}) {
		t.Fatalf("supported=%v error=%v", supported, err)
	}
	blocked := testcloud.New(t)
	ironicNodeUnreachable(t, blocked)
	if err := newIronicNodes(t, blocked).SetBootDevice(ctx, "n-1", nodes.BootDeviceOpts{}, nodes.WithSetBootDeviceField("persistent", true)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected field: %v", err)
	}
}

func TestIronicNodeInjectNMISendsEmptyJSONObject(t *testing.T) {
	cloud := testcloud.New(t)
	recorder := ironicNodeServe(t, cloud, "PUT /v1/nodes/n-1/management/inject_nmi", 204, "")
	if err := newIronicNodes(t, cloud).InjectNMI(context.Background(), "n-1"); err != nil {
		t.Fatal(err)
	}
	if call := recorder.only(t); call.Body != `{}` {
		t.Fatalf("inject_nmi body=%q", call.Body)
	}
}

func TestIronicNodeSetRAIDConfigDefaultsSizeToMAXAndKeepsFalse(t *testing.T) {
	cloud := testcloud.New(t)
	recorder := ironicNodeServe(t, cloud, "PUT /v1/nodes/n-1/states/raid", 204, "")
	size := 100
	err := newIronicNodes(t, cloud).SetRAIDConfig(context.Background(), "n-1", nodes.RAIDConfigOpts{LogicalDisks: []nodes.LogicalDisk{
		{RAIDLevel: nodes.RAID1, IsRootVolume: ironicNodeFalse()},
		{RAIDLevel: nodes.RAID10, SizeGB: &size, SharePhysicalDisks: ironicNodeFalse(), VolumeName: "data"},
	}}, nodes.WithSetRAIDConfigField("vendor", []string{}))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"logical_disks": `[{"is_root_volume":false,"raid_level":"1","size_gb":"MAX"},{"raid_level":"1+0","share_physical_disks":false,"size_gb":100,"volume_name":"data"}]`,
		"vendor":        `[]`,
	}
	got := ironicNodeFields(t, recorder.only(t).Body)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body=%v", got)
	}
	blocked := testcloud.New(t)
	ironicNodeUnreachable(t, blocked)
	if err := newIronicNodes(t, blocked).SetRAIDConfig(context.Background(), "n-1", nodes.RAIDConfigOpts{}, nodes.WithSetRAIDConfigField("logical_disks", []int{})); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected field: %v", err)
	}
}

func TestIronicNodeWaitForProvisionStatePollsGetUntilStateMatches(t *testing.T) {
	cloud := testcloud.New(t)
	var polls atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/nodes/n-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-OpenStack-Ironic-API-Version") != ironicNodeVersion {
			t.Errorf("microversion header=%q", r.Header.Get("X-OpenStack-Ironic-API-Version"))
		}
		state := "deploying"
		if polls.Add(1) >= 2 {
			state = "active"
		}
		testcloud.JSON(w, 200, `{"uuid":"n-1","provision_state":"`+state+`"}`)
	})
	if err := newIronicNodes(t, cloud).WaitForProvisionState(context.Background(), "n-1", nodes.Active); err != nil {
		t.Fatal(err)
	}
	if polls.Load() != 2 {
		t.Fatalf("polls=%d", polls.Load())
	}
}

func TestIronicNodeWaitForProvisionStateDoesNotDetectFailedStatesAndStopsOnGetErrors(t *testing.T) {
	cloud := testcloud.New(t)
	failed := ironicNodeServe(t, cloud, "GET /v1/nodes/failed", 200, `{"uuid":"failed","provision_state":"deploy failed"}`)
	missing := ironicNodeServe(t, cloud, "GET /v1/nodes/missing", 404, `{}`)
	api := newIronicNodes(t, cloud)
	deadline, stop := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer stop()
	if err := api.WaitForProvisionState(deadline, "failed", nodes.Active); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("a failed state is not terminal for the native wait: %v", err)
	}
	if failed.count() < 1 {
		t.Fatal("the native wait must read the node")
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if err := api.WaitForProvisionState(context.Background(), "missing", nodes.Active); !errors.As(err, &unexpected) || unexpected.Actual != 404 || missing.count() != 1 {
		t.Fatalf("get error must stop the wait: %v calls=%d", err, missing.count())
	}
}

func TestIronicNodeStateCallsHonorPreCanceledContextAndDeadlineCause(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	api := newIronicNodes(t, cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	checks := map[string]error{
		"provision":   api.ChangeProvisionState(canceled, "n-1", nodes.ProvisionStateOpts{Target: nodes.TargetActive}),
		"power":       api.ChangePowerState(canceled, "n-1", nodes.PowerStateOpts{Target: nodes.PowerOn}),
		"maintenance": api.SetMaintenance(canceled, "n-1", nodes.MaintenanceOpts{}),
		"unset":       api.UnsetMaintenance(canceled, "n-1"),
		"bootdevice":  api.SetBootDevice(canceled, "n-1", nodes.BootDeviceOpts{BootDevice: "pxe"}),
		"nmi":         api.InjectNMI(canceled, "n-1"),
		"raid":        api.SetRAIDConfig(canceled, "n-1", nodes.RAIDConfigOpts{}),
		"wait":        api.WaitForProvisionState(canceled, "n-1", nodes.Active),
	}
	for name, err := range checks {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("pre-canceled %s: %v", name, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if err := api.ChangeProvisionState(deadline, "n-1", nodes.ProvisionStateOpts{Target: nodes.TargetActive}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
