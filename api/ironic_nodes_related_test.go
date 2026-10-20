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
)

func TestIronicNodeSuccessCodesAreExactForRelatedReadsAndWrites(t *testing.T) {
	subscription := nodes.CreateSubscriptionOpts{Destination: "https://events.example.test/hook"}
	ironicNodeRunStatusCases(t, []ironicNodeStatusCase{
		{name: "ListBIOSSettings", pattern: "GET /v1/nodes/n-1/bios", ok: []int{200}, bad: []int{202, 204}, body: `{"bios":[]}`,
			call: func(ctx context.Context, api *nodes.API) error {
				_, err := api.ListBIOSSettings(ctx, "n-1")
				return err
			}},
		{name: "GetBIOSSetting", pattern: "GET /v1/nodes/n-1/bios/Boot", ok: []int{200}, bad: []int{202, 204}, body: `{"setting":{"name":"Boot","value":"x"}}`,
			call: func(ctx context.Context, api *nodes.API) error {
				_, err := api.GetBIOSSetting(ctx, "n-1", "Boot")
				return err
			}},
		{name: "GetVendorPassthruMethods", pattern: "GET /v1/nodes/n-1/vendor_passthru/methods", ok: []int{200}, bad: []int{202, 204}, body: `{}`,
			call: func(ctx context.Context, api *nodes.API) error {
				_, err := api.GetVendorPassthruMethods(ctx, "n-1")
				return err
			}},
		{name: "ListVendorPassthruMethods", pattern: "GET /v1/nodes/n-1/vendor_passthru/methods", ok: []int{200}, bad: []int{202, 204}, body: `{}`,
			call: func(ctx context.Context, api *nodes.API) error {
				_, err := api.ListVendorPassthruMethods(ctx, "n-1")
				return err
			}},
		{name: "CallVendorPassthru", pattern: "POST /v1/nodes/n-1/vendor_passthru", ok: []int{200, 202, 204}, bad: []int{201}, body: `{"done":true}`,
			call: func(ctx context.Context, api *nodes.API) error {
				_, err := api.CallVendorPassthru(ctx, "n-1", "POST", nodes.VendorPassthruCallOpts{Method: "reset"})
				return err
			}},
		{name: "GetAllSubscriptions", pattern: "GET /v1/nodes/n-1/vendor_passthru", ok: []int{200}, bad: []int{202, 204}, body: `{}`,
			call: func(ctx context.Context, api *nodes.API) error {
				_, err := api.GetAllSubscriptions(ctx, "n-1")
				return err
			}},
		{name: "GetSubscription", pattern: "GET /v1/nodes/n-1/vendor_passthru", ok: []int{200}, bad: []int{202, 204}, body: `{"Id":"s1"}`,
			call: func(ctx context.Context, api *nodes.API) error {
				_, err := api.GetSubscription(ctx, "n-1", nodes.GetSubscriptionOpts{Id: "s1"})
				return err
			}},
		{name: "CreateSubscription", pattern: "POST /v1/nodes/n-1/vendor_passthru", ok: []int{200}, bad: []int{201, 202}, body: `{"Id":"s1"}`,
			call: func(ctx context.Context, api *nodes.API) error {
				_, err := api.CreateSubscription(ctx, "n-1", nodes.CallVendorPassthruOpts{Method: "create_subscription"}, subscription)
				return err
			}},
		{name: "DeleteSubscription", pattern: "DELETE /v1/nodes/n-1/vendor_passthru", ok: []int{200, 202, 204}, bad: []int{201},
			call: func(ctx context.Context, api *nodes.API) error {
				return api.DeleteSubscription(ctx, "n-1", nodes.DeleteSubscriptionOpts{Id: "s1"})
			}},
		{name: "GetInventory", pattern: "GET /v1/nodes/n-1/inventory", ok: []int{200}, bad: []int{202, 204}, body: `{"inventory":{},"plugin_data":{}}`,
			call: func(ctx context.Context, api *nodes.API) error { _, err := api.GetInventory(ctx, "n-1"); return err }},
		{name: "ListFirmware", pattern: "GET /v1/nodes/n-1/firmware", ok: []int{200}, bad: []int{202, 204}, body: `{"firmware":[]}`,
			call: func(ctx context.Context, api *nodes.API) error { _, err := api.ListFirmware(ctx, "n-1"); return err }},
		{name: "AttachVirtualMedia", pattern: "POST /v1/nodes/n-1/vmedia", ok: []int{204}, bad: []int{200, 201, 202},
			call: func(ctx context.Context, api *nodes.API) error {
				return api.AttachVirtualMedia(ctx, "n-1", nodes.AttachVirtualMediaOpts{DeviceType: nodes.VirtualMediaCD, ImageURL: "http://img.example.test/a.iso"})
			}},
		{name: "DetachVirtualMedia", pattern: "DELETE /v1/nodes/n-1/vmedia", ok: []int{204}, bad: []int{200, 202},
			call: func(ctx context.Context, api *nodes.API) error {
				return api.DetachVirtualMedia(ctx, "n-1", nodes.DetachVirtualMediaOpts{})
			}},
		{name: "GetVirtualMedia", pattern: "GET /v1/nodes/n-1/vmedia", ok: []int{200}, bad: []int{202, 204}, body: `{"inserted":false}`,
			call: func(ctx context.Context, api *nodes.API) error { _, err := api.GetVirtualMedia(ctx, "n-1"); return err }},
		{name: "ListVirtualInterfaces", pattern: "GET /v1/nodes/n-1/vifs", ok: []int{200}, bad: []int{202, 204}, body: `{"vifs":[]}`,
			call: func(ctx context.Context, api *nodes.API) error {
				_, err := api.ListVirtualInterfaces(ctx, "n-1")
				return err
			}},
		{name: "AttachVirtualInterface", pattern: "POST /v1/nodes/n-1/vifs", ok: []int{204}, bad: []int{200, 202},
			call: func(ctx context.Context, api *nodes.API) error {
				return api.AttachVirtualInterface(ctx, "n-1", nodes.VirtualInterfaceOpts{ID: "vif-1"})
			}},
		{name: "DetachVirtualInterface", pattern: "DELETE /v1/nodes/n-1/vifs/vif-1", ok: []int{204}, bad: []int{200, 202},
			call: func(ctx context.Context, api *nodes.API) error {
				return api.DetachVirtualInterface(ctx, "n-1", "vif-1")
			}},
	})
}

func TestIronicNodeBIOSQueriesAndPointerFieldsDecode(t *testing.T) {
	cloud := testcloud.New(t)
	list := ironicNodeServe(t, cloud, "GET /v1/nodes/n-1/bios", 200, `{"bios":[{"name":"Boot","value":"uefi","read_only":false,"lower_bound":0,"unique":null,"allowable_values":[]}]}`)
	one := ironicNodeServe(t, cloud, "GET /v1/nodes/n-1/bios/Boot", 200, `{"setting":{"name":"Boot","value":"uefi","read_only":false}}`)
	byName := ironicNodeServe(t, cloud, "GET /v1/nodes/n-1/bios/Named", 200, `{"Named":{"name":"Named","value":"v"}}`)
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	settings, err := api.ListBIOSSettings(ctx, "n-1", nodes.WithListBIOSSettingsOptions(nodes.ListBIOSSettingsOpts{Detail: true}), nodes.WithListBIOSSettingsQuery("vendor", "x"))
	if err != nil || len(settings) != 1 {
		t.Fatalf("settings=%+v error=%v", settings, err)
	}
	got := settings[0]
	if got.Name != "Boot" || got.ReadOnly == nil || *got.ReadOnly || got.LowerBound == nil || *got.LowerBound != 0 || got.Unique != nil || got.AllowableValues == nil || len(got.AllowableValues) != 0 {
		t.Fatalf("setting=%+v", got)
	}
	if query := list.only(t).Query; query.Get("detail") != "true" || query.Get("vendor") != "x" || query.Has("fields") {
		t.Fatalf("query=%v", query)
	}
	if _, err := api.ListBIOSSettings(ctx, "n-1", nodes.WithListBIOSSettingsOptions(nodes.ListBIOSSettingsOpts{Fields: []string{"Boot", "Proc"}})); err != nil {
		t.Fatal(err)
	}
	if query := list.all()[1].Query; query.Get("fields") != "Boot,Proc" || query.Has("detail") {
		t.Fatalf("fields query=%v", query)
	}
	setting, err := api.GetBIOSSetting(ctx, "n-1", "Boot")
	if err != nil || setting.Value != "uefi" || setting.ReadOnly == nil || *setting.ReadOnly {
		t.Fatalf("setting=%+v error=%v", setting, err)
	}
	if call := one.only(t); call.Path != "/v1/nodes/n-1/bios/Boot" {
		t.Fatalf("path=%s", call.Path)
	}
	// Gophercloud decodes the single-setting body from a "setting" key; a body
	// keyed by the setting name decodes to the zero BIOSSetting.
	named, err := api.GetBIOSSetting(ctx, "n-1", "Named")
	if err != nil || named == nil || !reflect.DeepEqual(*named, nodes.BIOSSetting{}) || byName.count() != 1 {
		t.Fatalf("named=%+v error=%v", named, err)
	}
}

func TestIronicNodeBIOSRejectsConflictingOptionsAndNilOptionBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	ironicNodeUnreachable(t, cloud)
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	if _, err := api.ListBIOSSettings(ctx, "n-1", nodes.WithListBIOSSettingsOptions(nodes.ListBIOSSettingsOpts{Detail: true, Fields: []string{"Boot"}})); err == nil {
		t.Fatal("detail with fields must fail before HTTP")
	}
	if _, err := api.ListBIOSSettings(ctx, "n-1", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option: %v", err)
	}
}

func TestIronicNodeVendorPassthruMethodsAndCallBodyAndStatus(t *testing.T) {
	cloud := testcloud.New(t)
	ironicNodeServe(t, cloud, "GET /v1/nodes/n-1/vendor_passthru/methods", 200, `{"reset":{"http_methods":["POST"],"async":false,"description":"","attach":false,"require_exclusive_lock":true}}`)
	call := ironicNodeServe(t, cloud, "POST /v1/nodes/n-1/vendor_passthru", 202, `{"queued":true}`)
	empty := ironicNodeServe(t, cloud, "PUT /v1/nodes/n-1/vendor_passthru", 204, "")
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	methods, err := api.ListVendorPassthruMethods(ctx, "n-1")
	if err != nil || methods["reset"].Async || methods["reset"].HTTPMethods[0] != "POST" || !methods["reset"].RequireExclusiveLock {
		t.Fatalf("methods=%+v error=%v", methods, err)
	}
	value, err := api.CallVendorPassthru(ctx, "n-1", "POST", nodes.VendorPassthruCallOpts{Method: "reset", Body: map[string]any{"force": false, "note": nil}})
	if err != nil || value["queued"] != true {
		t.Fatalf("value=%v error=%v", value, err)
	}
	recorded := call.only(t)
	if recorded.Query.Get("method") != "reset" || !reflect.DeepEqual(ironicNodeFields(t, recorded.Body), map[string]string{"force": `false`, "note": `null`}) {
		t.Fatalf("call=%+v", recorded)
	}
	value, err = api.CallVendorPassthru(ctx, "n-1", "PUT", nodes.VendorPassthruCallOpts{Method: "reset"})
	if err != nil || len(value) != 0 {
		t.Fatalf("no-content value=%v error=%v", value, err)
	}
	if got := empty.only(t); got.Body != "" || got.Query.Get("method") != "reset" {
		t.Fatalf("empty call=%+v", got)
	}
}

func TestIronicNodeVendorPassthruRequiresMethodAndRejectsNilOptionBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	ironicNodeUnreachable(t, cloud)
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	if _, err := api.CallVendorPassthru(ctx, "n-1", "POST", nodes.VendorPassthruCallOpts{}); err == nil {
		t.Fatal("an empty vendor passthru method must fail before HTTP")
	}
	if _, err := api.CallVendorPassthru(ctx, "n-1", "POST", nodes.VendorPassthruCallOpts{Method: "x"}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option: %v", err)
	}
}

func TestIronicNodeSubscriptionsUseMethodQueryAndRequestBodies(t *testing.T) {
	cloud := testcloud.New(t)
	create := ironicNodeServe(t, cloud, "POST /v1/nodes/n-1/vendor_passthru", 200, `{"Id":"s1","Destination":"https://events.example.test/hook","EventTypes":["Alert"],"Protocol":"Redfish"}`)
	get := ironicNodeServe(t, cloud, "GET /v1/nodes/n-1/vendor_passthru", 200, `{"Id":"s1","Context":"","EventTypes":[]}`)
	remove := ironicNodeServe(t, cloud, "DELETE /v1/nodes/n-1/vendor_passthru", 204, "")
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	created, err := api.CreateSubscription(ctx, "n-1", nodes.CallVendorPassthruOpts{Method: "create_subscription"},
		nodes.CreateSubscriptionOpts{Destination: "https://events.example.test/hook", EventTypes: []string{"Alert"}})
	if err != nil || created.Id != "s1" || created.Protocol != "Redfish" {
		t.Fatalf("created=%+v error=%v", created, err)
	}
	if call := create.only(t); call.Query.Get("method") != "create_subscription" ||
		!reflect.DeepEqual(ironicNodeFields(t, call.Body), map[string]string{"Destination": `"https://events.example.test/hook"`, "EventTypes": `["Alert"]`}) {
		t.Fatalf("create call=%+v", call)
	}
	found, err := api.GetSubscription(ctx, "n-1", nodes.GetSubscriptionOpts{Id: "s1"}, nodes.WithGetSubscriptionOptions(nodes.CallVendorPassthruOpts{Method: "get_subscription"}))
	if err != nil || found.Id != "s1" || found.EventTypes == nil || len(found.EventTypes) != 0 {
		t.Fatalf("found=%+v error=%v", found, err)
	}
	if call := get.only(t); call.Query.Get("method") != "get_subscription" || call.Body != `{"id":"s1"}` {
		t.Fatalf("get call=%+v", call)
	}
	if err := api.DeleteSubscription(ctx, "n-1", nodes.DeleteSubscriptionOpts{Id: "s1"}, nodes.WithDeleteSubscriptionOptions(nodes.CallVendorPassthruOpts{Method: "delete_subscription"})); err != nil {
		t.Fatal(err)
	}
	if call := remove.only(t); call.Query.Get("method") != "delete_subscription" || call.Body != `{"id":"s1"}` {
		t.Fatalf("delete call=%+v", call)
	}
}

func TestIronicNodeGetAllSubscriptionsAndDeleteWithoutMethodOmitQuery(t *testing.T) {
	cloud := testcloud.New(t)
	all := ironicNodeServe(t, cloud, "GET /v1/nodes/n-1/vendor_passthru", 200, `{"Name":"Event Subscriptions","Members":[{"@odata.id":"/s/1"}],"Members@odata.count":1}`)
	remove := ironicNodeServe(t, cloud, "DELETE /v1/nodes/n-1/vendor_passthru", 200, "")
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	value, err := api.GetAllSubscriptions(ctx, "n-1", nodes.WithGetAllSubscriptionsOptions(nodes.CallVendorPassthruOpts{Method: "get_all_subscriptions"}))
	if err != nil || value.MembersCount != 1 || len(value.Members) != 1 {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if query := all.only(t).Query; query.Get("method") != "get_all_subscriptions" {
		t.Fatalf("query=%v", query)
	}
	if err := api.DeleteSubscription(ctx, "n-1", nodes.DeleteSubscriptionOpts{}); err != nil {
		t.Fatal(err)
	}
	if call := remove.only(t); call.Query.Has("method") || call.Body != `{"id":""}` {
		t.Fatalf("a missing method is not validated for subscriptions: %+v", call)
	}
	blocked := testcloud.New(t)
	ironicNodeUnreachable(t, blocked)
	if _, err := newIronicNodes(t, blocked).GetAllSubscriptions(ctx, "n-1", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option: %v", err)
	}
	if err := newIronicNodes(t, blocked).DeleteSubscription(ctx, "n-1", nodes.DeleteSubscriptionOpts{Id: "s1"}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option: %v", err)
	}
}

func TestIronicNodeVirtualInterfacesListAttachDetach(t *testing.T) {
	cloud := testcloud.New(t)
	ironicNodeServe(t, cloud, "GET /v1/nodes/n-1/vifs", 200, `{"vifs":[{"id":"vif-1"},{"id":"vif-2"}]}`)
	attach := ironicNodeServe(t, cloud, "POST /v1/nodes/n-1/vifs", 204, "")
	detach := ironicNodeServe(t, cloud, "DELETE /v1/nodes/n-1/vifs/vif-1", 204, "")
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	vifs, err := api.ListVirtualInterfaces(ctx, "n-1")
	if err != nil || len(vifs) != 2 || vifs[1].ID != "vif-2" {
		t.Fatalf("vifs=%+v error=%v", vifs, err)
	}
	if err := api.AttachVirtualInterface(ctx, "n-1", nodes.VirtualInterfaceOpts{ID: "vif-1", PortUUID: "port-1"}, nodes.WithAttachVirtualInterfaceField("vendor", false)); err != nil {
		t.Fatal(err)
	}
	if got := ironicNodeFields(t, attach.only(t).Body); !reflect.DeepEqual(got, map[string]string{"id": `"vif-1"`, "port_uuid": `"port-1"`, "vendor": `false`}) {
		t.Fatalf("attach body=%v", got)
	}
	if err := api.DetachVirtualInterface(ctx, "n-1", "vif-1"); err != nil {
		t.Fatal(err)
	}
	if call := detach.only(t); call.Body != "" {
		t.Fatalf("detach body=%q", call.Body)
	}
}

func TestIronicNodeVirtualInterfaceRejectsConflictsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	ironicNodeUnreachable(t, cloud)
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	if err := api.AttachVirtualInterface(ctx, "n-1", nodes.VirtualInterfaceOpts{ID: "v", PortUUID: "p", PortgroupUUID: "g"}); err == nil {
		t.Fatal("port_uuid with portgroup_uuid must fail before HTTP")
	}
	if err := api.AttachVirtualInterface(ctx, "n-1", nodes.VirtualInterfaceOpts{ID: "v"}, nodes.WithAttachVirtualInterfaceField("id", "x")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected field: %v", err)
	}
	if err := api.AttachVirtualInterface(ctx, "n-1", nodes.VirtualInterfaceOpts{}); err == nil {
		t.Fatal("missing id must fail before HTTP")
	}
	if err := api.AttachVirtualInterface(ctx, "n-1", nodes.VirtualInterfaceOpts{ID: "v"}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option: %v", err)
	}
}

func TestIronicNodeVirtualMediaAttachDetachAndGetKeepsRawBody(t *testing.T) {
	cloud := testcloud.New(t)
	attach := ironicNodeServe(t, cloud, "POST /v1/nodes/n-1/vmedia", 204, "")
	detach := ironicNodeServe(t, cloud, "DELETE /v1/nodes/n-1/vmedia", 204, "")
	ironicNodeServe(t, cloud, "GET /v1/nodes/n-1/vmedia", 200, `{"image":"","inserted":false,"media_types":[],"vendor_extra":1}`)
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	if err := api.AttachVirtualMedia(ctx, "n-1", nodes.AttachVirtualMediaOpts{DeviceType: nodes.VirtualMediaCD, ImageURL: "http://img.example.test/a.iso"}, nodes.WithAttachVirtualMediaField("vendor", false)); err != nil {
		t.Fatal(err)
	}
	if err := api.AttachVirtualMedia(ctx, "n-1", nodes.AttachVirtualMediaOpts{DeviceType: nodes.VirtualMediaDisk, ImageURL: "swift://c/o", ImageDownloadSource: nodes.ImageDownloadSourceSwift}); err != nil {
		t.Fatal(err)
	}
	calls := attach.all()
	if got := ironicNodeFields(t, calls[0].Body); !reflect.DeepEqual(got, map[string]string{"device_type": `"cdrom"`, "image_url": `"http://img.example.test/a.iso"`, "vendor": `false`}) {
		t.Fatalf("first attach body=%v", got)
	}
	if got := ironicNodeFields(t, calls[1].Body); got["image_download_source"] != `"swift"` {
		t.Fatalf("second attach body=%v", got)
	}
	if err := api.DetachVirtualMedia(ctx, "n-1", nodes.DetachVirtualMediaOpts{DeviceTypes: []nodes.VirtualMediaDeviceType{nodes.VirtualMediaCD, nodes.VirtualMediaFloppy}}, nodes.WithDetachVirtualMediaQuery("vendor", "x")); err != nil {
		t.Fatal(err)
	}
	if err := api.DetachVirtualMedia(ctx, "n-1", nodes.DetachVirtualMediaOpts{}); err != nil {
		t.Fatal(err)
	}
	detached := detach.all()
	if detached[0].Query.Get("device_types") != "cdrom,floppy" || detached[0].Query.Get("vendor") != "x" || len(detached[1].Query) != 0 {
		t.Fatalf("detach queries=%v %v", detached[0].Query, detached[1].Query)
	}
	media, err := api.GetVirtualMedia(ctx, "n-1")
	if err != nil || media.Inserted || media.Image != "" || media.MediaTypes == nil || len(media.MediaTypes) != 0 || string(media.Body) != `{"image":"","inserted":false,"media_types":[],"vendor_extra":1}` {
		t.Fatalf("media=%+v error=%v", media, err)
	}
}

func TestIronicNodeVirtualMediaRejectsProtectedAndNilOptionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	ironicNodeUnreachable(t, cloud)
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	if err := api.AttachVirtualMedia(ctx, "n-1", nodes.AttachVirtualMediaOpts{}, nodes.WithAttachVirtualMediaField("device_type", "x")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected field: %v", err)
	}
	if err := api.AttachVirtualMedia(ctx, "n-1", nodes.AttachVirtualMediaOpts{}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil attach option: %v", err)
	}
	if err := api.DetachVirtualMedia(ctx, "n-1", nodes.DetachVirtualMediaOpts{}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil detach option: %v", err)
	}
}

func TestIronicNodeInventoryAndFirmwareDecodeNullableFields(t *testing.T) {
	cloud := testcloud.New(t)
	ironicNodeServe(t, cloud, "GET /v1/nodes/n-1/inventory", 200, `{"inventory":{"bmc_address":"192.0.2.9"},"plugin_data":{"vendor_note":false}}`)
	ironicNodeServe(t, cloud, "GET /v1/nodes/n-1/firmware", 200, `{"firmware":[{"component":"bios","initial_version":"1.0","current_version":"1.1","created_at":"2026-01-02T03:04:05+00:00","updated_at":null,"vendor":null,"serial_number":""}]}`)
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	data, err := api.GetInventory(ctx, "n-1")
	if err != nil || data.Inventory.BmcAddress != "192.0.2.9" {
		t.Fatalf("inventory=%+v error=%v", data, err)
	}
	firmware, err := api.ListFirmware(ctx, "n-1")
	if err != nil || len(firmware) != 1 || firmware[0].Component != "bios" || firmware[0].UpdatedAt != nil || firmware[0].Vendor != nil || firmware[0].SerialNumber == nil || *firmware[0].SerialNumber != "" {
		t.Fatalf("firmware=%+v error=%v", firmware, err)
	}
}

func TestIronicNodeRelatedCallsHonorPreCanceledContextAndDeadlineCause(t *testing.T) {
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
	var errs []error
	collect := func(_ any, err error) { errs = append(errs, err) }
	collect(api.ListBIOSSettings(canceled, "n-1"))
	collect(api.GetBIOSSetting(canceled, "n-1", "Boot"))
	collect(api.GetVendorPassthruMethods(canceled, "n-1"))
	collect(api.ListVendorPassthruMethods(canceled, "n-1"))
	collect(api.CallVendorPassthru(canceled, "n-1", "POST", nodes.VendorPassthruCallOpts{Method: "reset"}))
	collect(api.GetAllSubscriptions(canceled, "n-1"))
	collect(api.GetSubscription(canceled, "n-1", nodes.GetSubscriptionOpts{Id: "s"}))
	collect(api.CreateSubscription(canceled, "n-1", nodes.CallVendorPassthruOpts{Method: "m"}, nodes.CreateSubscriptionOpts{}))
	errs = append(errs, api.DeleteSubscription(canceled, "n-1", nodes.DeleteSubscriptionOpts{}))
	collect(api.GetInventory(canceled, "n-1"))
	collect(api.ListFirmware(canceled, "n-1"))
	errs = append(errs, api.AttachVirtualMedia(canceled, "n-1", nodes.AttachVirtualMediaOpts{}))
	errs = append(errs, api.DetachVirtualMedia(canceled, "n-1", nodes.DetachVirtualMediaOpts{}))
	collect(api.GetVirtualMedia(canceled, "n-1"))
	collect(api.ListVirtualInterfaces(canceled, "n-1"))
	errs = append(errs, api.AttachVirtualInterface(canceled, "n-1", nodes.VirtualInterfaceOpts{ID: "v"}))
	errs = append(errs, api.DetachVirtualInterface(canceled, "n-1", "v"))
	collect(api.GetBootDevice(canceled, "n-1"))
	collect(api.GetSupportedBootDevices(canceled, "n-1"))
	for i, err := range errs {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("pre-canceled call %d: %v", i, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if err := api.AttachVirtualInterface(deadline, "n-1", nodes.VirtualInterfaceOpts{ID: "v"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
