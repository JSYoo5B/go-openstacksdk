package compute_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestServerCreateExplicitLegacyMappingsAndTrustedCertificates(t *testing.T) {
	f := newCreateBodyFixture(t, "2.67")
	create := compute.CreateServerRequest{Name: "web", Image: resource.ID("image"), Flavor: resource.ID("flavor")}
	keep := false
	explicit := compute.ServerBlockDevice{SourceType: "blank", DestinationType: "local", BootIndex: request.Null[int](), DeleteOnTermination: &keep,
		DeviceName: "/dev/vdb", DeviceType: "disk", DiskBus: "virtio", GuestFormat: "swap", VolumeSize: 2, VolumeType: "fast", Tag: "swap", NoDevice: true}
	option := compute.WithBlockDeviceMappings(explicit)
	keep = true // the option snapshots pointer fields
	if _, err := f.service.Servers.Create(context.Background(), create, option, compute.WithDataVolumes(resource.ID("vol"))); err != nil {
		t.Fatal(err)
	}
	want := []any{
		map[string]any{"source_type": "blank", "destination_type": "local", "boot_index": nil, "delete_on_termination": false, "device_name": "/dev/vdb", "device_type": "disk",
			"disk_bus": "virtio", "guest_format": "swap", "volume_size": float64(2), "volume_type": "fast", "tag": "swap", "no_device": true},
		map[string]any{"source_type": "image", "destination_type": "local", "uuid": "image", "boot_index": float64(0), "delete_on_termination": true},
		map[string]any{"source_type": "volume", "destination_type": "volume", "uuid": "vol", "boot_index": float64(-1), "delete_on_termination": false},
	}
	if !reflect.DeepEqual(f.body["block_device_mapping_v2"], want) {
		t.Fatalf("%#v", f.body["block_device_mapping_v2"])
	}
	f.reset("2.63")
	legacy := compute.LegacyBlockDevice{DeviceName: "vdb", VolumeID: "vol", VolumeSize: 1, DeleteOnTermination: &keep}
	if _, err := f.service.Servers.Create(context.Background(), create, compute.WithLegacyBlockDeviceMappings(legacy), compute.WithTrustedImageCertificates("cert-a", "cert-b")); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.body["block_device_mapping"], []any{map[string]any{"device_name": "vdb", "volume_id": "vol", "volume_size": float64(1), "delete_on_termination": true}}) ||
		!reflect.DeepEqual(f.body["trusted_image_certificates"], []any{"cert-a", "cert-b"}) || f.body["block_device_mapping_v2"] != nil {
		t.Fatal(f.body)
	}
	for name, scenario := range map[string]struct {
		version string
		options []compute.CreateServerOption
		want    error
	}{
		"legacy and v2":     {"2.90", []compute.CreateServerOption{compute.WithLegacyBlockDeviceMappings(legacy), compute.WithDataVolumes(resource.ID("vol"))}, resource.ErrInvalidOption},
		"legacy no device":  {"2.90", []compute.CreateServerOption{compute.WithLegacyBlockDeviceMappings(compute.LegacyBlockDevice{})}, resource.ErrInvalidOption},
		"source":            {"2.90", []compute.CreateServerOption{compute.WithBlockDeviceMappings(compute.ServerBlockDevice{SourceType: "url"})}, resource.ErrInvalidOption},
		"destination":       {"2.90", []compute.CreateServerOption{compute.WithBlockDeviceMappings(compute.ServerBlockDevice{SourceType: "image", DestinationType: "nfs"})}, resource.ErrInvalidOption},
		"empty":             {"2.90", []compute.CreateServerOption{compute.WithBlockDeviceMappings()}, resource.ErrInvalidOption},
		"empty certs":       {"2.90", []compute.CreateServerOption{compute.WithTrustedImageCertificates()}, resource.ErrInvalidOption},
		"tag version":       {"2.41", []compute.CreateServerOption{compute.WithBlockDeviceMappings(compute.ServerBlockDevice{SourceType: "blank", Tag: "t"})}, resource.ErrUnsupported},
		"type version":      {"2.66", []compute.CreateServerOption{compute.WithBlockDeviceMappings(compute.ServerBlockDevice{SourceType: "blank", VolumeType: "v"})}, resource.ErrUnsupported},
		"certificates 2.62": {"2.62", []compute.CreateServerOption{compute.WithTrustedImageCertificates("c")}, resource.ErrUnsupported},
	} {
		t.Run(name, func(t *testing.T) {
			f.reset(scenario.version)
			if _, err := f.service.Servers.Create(context.Background(), create, scenario.options...); !errors.Is(err, scenario.want) || f.posts.Load() != 0 {
				t.Fatal(err, f.posts.Load())
			}
		})
	}
}

func TestServerCreateReservationReturnsOnlyReservationID(t *testing.T) {
	f := newCreateBodyFixture(t, "2.90")
	create := compute.CreateServerRequest{Name: "web", Image: resource.ID("image"), Flavor: resource.ID("flavor")}
	f.postCode, f.postBody = 202, `{"reservation_id":"r-abc"}`
	reservation, err := f.service.Servers.CreateReservation(context.Background(), create, compute.WithServerCount(2, 4))
	if err != nil || reservation.ReservationID != "r-abc" || reservation.StatusCode != 202 || f.body["return_reservation_id"] != true || f.body["min_count"] != float64(2) || f.body["max_count"] != float64(4) || f.posts.Load() != 1 {
		t.Fatal(reservation, err, f.body)
	}
	for name, body := range map[string]string{"server shape": `{"server":{"id":"created"}}`, "empty": `{"reservation_id":""}`} {
		f.reset("2.90")
		f.postCode, f.postBody = 202, body
		if reservation, err := f.service.Servers.CreateReservation(context.Background(), create); err == nil || reservation != nil || f.posts.Load() != 1 {
			t.Fatal(name, reservation, err, f.posts.Load())
		}
	}
	f.reset("2.90")
	for _, option := range []compute.CreateServerOption{compute.WithWait(), compute.WithCreateRefresh()} {
		if _, err := f.service.Servers.CreateReservation(context.Background(), create, option); !errors.Is(err, resource.ErrInvalidOption) || f.posts.Load() != 0 {
			t.Fatal(err, f.posts.Load())
		}
	}
}

func TestCreateRefreshedServerExpandsAndCopiesAdminPass(t *testing.T) {
	f := newCreateBodyFixture(t, "2.90")
	create := compute.CreateServerRequest{Name: "web", Image: resource.ID("image"), Flavor: resource.ID("flavor")}
	f.postCode, f.postBody = 202, `{"server":{"id":"created","status":"BUILD","adminPass":"generated"}}`
	f.getCode, f.getBody = 200, `{"server":{"id":"created","status":"ACTIVE","addresses":`+autoFloating+`}}`
	result, err := f.service.CreateRefreshedServer(context.Background(), create, nil, compute.WithAddressReachability(false))
	if err != nil || result.Creation.AdminPass != "generated" || result.Server.AdminPass != "generated" || result.Server.Status != "ACTIVE" || result.Addresses == nil || result.Addresses.PublicIPv4 != "8.8.8.8" || f.posts.Load() != 1 || f.gets.Load() != 1 {
		t.Fatal(result, err)
	}
	f.reset("2.90")
	f.postCode, f.postBody = 202, `{"server":{"id":"created","status":"BUILD","adminPass":"generated"}}`
	f.getCode, f.getBody = 200, `{"server":{"id":"created","status":"ERROR","fault":{"code":500}}}`
	result, err = f.service.CreateRefreshedServer(context.Background(), create, nil)
	var fault *compute.ServerFaultError
	if !errors.As(err, &fault) || fault.HasFaultMessage || fault.Server != result.Server || result.Server.AdminPass != "generated" || result.Addresses != nil {
		t.Fatal(result, err)
	}
	f.reset("2.90")
	if _, err := f.service.CreateRefreshedServer(context.Background(), create, []compute.CreateServerOption{compute.WithWait()}); !errors.Is(err, resource.ErrInvalidOption) || f.posts.Load() != 0 {
		t.Fatal(err)
	}
}

func TestServerWithoutAddressesDeleteSkipsTruthyEmptyRowMaps(t *testing.T) {
	f := newAutomaticFixture(t)
	var deletes atomic.Int32
	f.cloud.Mux.HandleFunc("DELETE /v2.1/servers/server", func(w http.ResponseWriter, r *http.Request) { deletes.Add(1); w.WriteHeader(204) })
	opts := append(serverReadyOptions(), compute.WithServerReadyAutomaticIPOptions(compute.WithAutomaticIPEnabled(false)), compute.WithServerWithoutAddressesPolicy(compute.DeleteServerWithoutAddresses))
	for addresses, emptyRows := range map[string]bool{`{"private":[]}`: true, `{"a":null,"b":[]}`: true, `{}`: false, `null`: false} {
		deletes.Store(0)
		_, err := f.service.GetActiveServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, addresses)}, opts...)
		var unavailable *compute.ServerAddressesUnavailableError
		wantDeletes := int32(1)
		if emptyRows {
			wantDeletes = 0
		}
		if !errors.As(err, &unavailable) || unavailable.EmptyRows != emptyRows || (unavailable.Deletion == nil) != emptyRows || deletes.Load() != wantDeletes {
			t.Fatal(addresses, err, deletes.Load())
		}
	}
	f.rawBody = func(int32) (int, string) {
		return 200, `{"server":{"id":"server","status":"ACTIVE","addresses":{"private":[]}}}`
	}
	deletes.Store(0)
	_, err := f.service.WaitForServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, opts...)
	var unavailable *compute.ServerAddressesUnavailableError
	if !errors.As(err, &unavailable) || !unavailable.EmptyRows || unavailable.Deletion != nil || deletes.Load() != 0 {
		t.Fatal(err, deletes.Load())
	}
}

func TestWaitForServerForwardsPoolSelectorToDispatch(t *testing.T) {
	f := newDispatchFixture(t, "")
	opts := append(dispatchOptions(), compute.WithFloatingIPPool(resource.ID("pool-network")), compute.WithAutomaticEnsureOptions(network.WithEnsureReuse(false)), compute.WithAutomaticIPEnabled(false))
	result, err := f.service.WaitForServer(context.Background(), dispatchInput(t), compute.WithServerReadyAutomaticIPOptions(opts...), compute.WithServerReadyWaitOptions(resource.WithPollInterval(time.Millisecond)))
	if err != nil || result == nil || result.Mode != compute.ServerIPPool || result.Decision.Selection.NetworkID != "pool-network" || result.Assignment == nil || !result.Assignment.Allocated || result.Assignment.FloatingIP.ID != "ip-pool" || !result.Observed || f.posts.Load() != 1 {
		t.Fatal(result, err, f.trace())
	}
}
