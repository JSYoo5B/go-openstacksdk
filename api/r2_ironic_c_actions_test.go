package api_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/drivers"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestR2IronicCBootModeSecureBootAndTraitsWire(t *testing.T) {
	cloud := testcloud.New(t)
	mode := r2icServe(t, cloud, "PUT /v1/nodes/n1/states/boot_mode", 202, "")
	secure := r2icServe(t, cloud, "PUT /v1/nodes/n1/states/secure_boot", 202, "")
	traits := r2icServe(t, cloud, "PUT /v1/nodes/n1/traits", 204, "")
	api := nodes.New(r2icClient(cloud, "1.80"))
	ctx := context.Background()
	r2icDiscovery(t, cloud, "1.1", "1.100")
	if err := api.SetBootMode(ctx, "n1", "uefi"); err != nil {
		t.Fatal(err)
	}
	if err := api.SetSecureBoot(ctx, "n1", false); err != nil {
		t.Fatal(err)
	}
	if err := api.SetTraits(ctx, "n1", nil); err != nil {
		t.Fatal(err)
	}
	if err := api.SetTraits(ctx, "n1", []string{}); err != nil {
		t.Fatal(err)
	}
	if c := mode.all()[0]; c.Body != `{"target":"uefi"}` || c.Version != "1.80" {
		t.Fatalf("%+v", c)
	}
	if c := secure.all()[0]; c.Body != `{"target":false}` {
		t.Fatalf("%+v", c)
	}
	if c := traits.all(); c[0].Body != `{"traits":null}` || c[1].Body != `{"traits":[]}` {
		t.Fatalf("%+v", c)
	}
	if err := api.SetBootMode(ctx, "n1", "legacy"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("invalid target: %v", err)
	}
	if len(mode.all()) != 1 {
		t.Fatal("invalid target must not send a request")
	}
}

func TestR2IronicCPickVersionRaisesUnsetAndRejectsLowConfigured(t *testing.T) {
	cloud := testcloud.New(t)
	r2icDiscovery(t, cloud, "1.1", "1.100")
	put := r2icServe(t, cloud, "PUT /v1/nodes/n1/traits", 204, "")
	ctx := context.Background()
	if err := nodes.New(r2icClient(cloud, "")).SetTraits(ctx, "n1", []string{"A"}); err != nil {
		t.Fatal(err)
	}
	if got := put.all()[0].Version; got != "1.37" {
		t.Fatalf("unset client raises to the required version, got %q", got)
	}
	if err := nodes.New(r2icClient(cloud, "1.30")).SetTraits(ctx, "n1", []string{"A"}); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("low configured version: %v", err)
	}
	if len(put.all()) != 1 {
		t.Fatal("unsupported version must not send the PUT")
	}
	old := testcloud.New(t)
	r2icDiscovery(t, old, "1.1", "1.30")
	nopt := r2icServe(t, old, "PUT /v1/nodes/n1/traits", 204, "")
	if err := nodes.New(r2icClient(old, "")).SetTraits(ctx, "n1", nil); !errors.Is(err, resource.ErrUnsupported) || len(nopt.all()) != 0 {
		t.Fatalf("endpoint range: %v", err)
	}
}

func TestR2IronicCRemoveTraitIgnoreMissingOnly400(t *testing.T) {
	cloud := testcloud.New(t)
	r2icDiscovery(t, cloud, "1.1", "1.100")
	del := r2icServe(t, cloud, "DELETE /v1/nodes/n1/traits/T%2FX", 204, "", 400, `{}`, 400, `{}`, 404, `{}`, 409, `{}`)
	api := nodes.New(r2icClient(cloud, "1.99"))
	ctx := context.Background()
	if ok, err := api.RemoveTrait(ctx, "n1", "T/X"); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if ok, err := api.RemoveTrait(ctx, "n1", "T/X"); err != nil || ok {
		t.Fatalf("400 is a no-op: %v %v", ok, err)
	}
	if _, err := api.RemoveTrait(ctx, "n1", "T/X", nodes.WithRemoveTraitIgnoreMissing(false)); !gophercloud.ResponseCodeIs(err, 400) {
		t.Fatalf("strict 400: %v", err)
	}
	if _, err := api.RemoveTrait(ctx, "n1", "T/X"); !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatalf("404 stays an error: %v", err)
	}
	if _, err := api.RemoveTrait(ctx, "n1", "T/X"); !gophercloud.ResponseCodeIs(err, 409) || len(del.all()) != 5 {
		t.Fatalf("409 once, no replay: %v calls=%d", err, len(del.all()))
	}
}

func TestR2IronicCInjectNMIVIFFirmwareAssertMicroversionBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	testcloud.New(t)
	ironicNodeUnreachable(t, cloud)
	low := nodes.New(r2icClient(cloud, "1.20"))
	ctx := context.Background()
	if err := low.InjectNMIChecked(ctx, "n1"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := low.ListVIFIDs(ctx, "n1"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := low.DetachVIF(ctx, "n1", "v"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := low.ListFirmwareRaw(ctx, "n1"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if err := low.DetachVirtualMediaJSON(ctx, "n1", []string{"cdrom"}); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
}

func TestR2IronicCInjectNMIListVIFsDetachVIFWire(t *testing.T) {
	cloud := testcloud.New(t)
	nmi := r2icServe(t, cloud, "PUT /v1/nodes/n1/management/inject_nmi", 204, "")
	vifs := r2icServe(t, cloud, "GET /v1/nodes/n1/vifs", 200, `{"vifs":[{"id":"a","extra":1},{"id":"b"}]}`)
	del := r2icServe(t, cloud, "DELETE /v1/nodes/n1/vifs/a", 204, "", 400, `{}`, 400, `{}`, 409, `{}`)
	api := nodes.New(r2icClient(cloud, "1.99"))
	ctx := context.Background()
	if err := api.InjectNMIChecked(ctx, "n1"); err != nil || nmi.all()[0].Body != `{}` {
		t.Fatalf("%v %+v", err, nmi.all())
	}
	ids, err := api.ListVIFIDs(ctx, "n1")
	if err != nil || len(ids) != 2 || ids[0] != "a" || ids[1] != "b" || vifs.all()[0].Version != "1.99" {
		t.Fatal(ids, err)
	}
	if ok, err := api.DetachVIF(ctx, "n1", "a"); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if ok, err := api.DetachVIF(ctx, "n1", "a"); err != nil || ok {
		t.Fatalf("already detached: %v %v", ok, err)
	}
	if _, err := api.DetachVIF(ctx, "n1", "a", nodes.WithDetachVIFIgnoreMissing(false)); !gophercloud.ResponseCodeIs(err, 400) {
		t.Fatal(err)
	}
	if _, err := api.DetachVIF(ctx, "n1", "a"); !gophercloud.ResponseCodeIs(err, 409) || len(del.all()) != 4 {
		t.Fatalf("no replay: %v %d", err, len(del.all()))
	}
}

func TestR2IronicCDetachVirtualMediaJSONUsesPythonBodyForm(t *testing.T) {
	cloud := testcloud.New(t)
	del := r2icServe(t, cloud, "DELETE /v1/nodes/n1/vmedia", 204, "", 204, "", 400, `{}`)
	api := nodes.New(r2icClient(cloud, "1.99"))
	ctx := context.Background()
	if err := api.DetachVirtualMediaJSON(ctx, "n1", []string{"cdrom", "floppy"}); err != nil {
		t.Fatal(err)
	}
	if err := api.DetachVirtualMediaJSON(ctx, "n1", nil); err != nil {
		t.Fatal(err)
	}
	calls := del.all()
	if calls[0].Body != `{"device_types":"cdrom,floppy"}` || calls[0].RawQuery != "" || calls[1].Body != "" {
		t.Fatalf("%+v", calls)
	}
	if err := api.DetachVirtualMediaJSON(ctx, "n1", []string{"disk"}); !gophercloud.ResponseCodeIs(err, 400) {
		t.Fatalf("400 stays an error: %v", err)
	}
}

func TestR2IronicCRawListsKeepUnknownFieldsAndDriverReadsDriverFirst(t *testing.T) {
	cloud := testcloud.New(t)
	fw := r2icServe(t, cloud, "GET /v1/nodes/n1/firmware", 200, `{"firmware":[{"component":"bios","extra":{"x":1}}]}`)
	pt := r2icServe(t, cloud, "GET /v1/nodes/n1/vendor_passthru/methods", 200, `{"m":{"async":true,"future":[1]}}`)
	drv := r2icServe(t, cloud, "GET /v1/drivers/fake", 200, `{"name":"fake"}`)
	dpt := r2icServe(t, cloud, "GET /v1/drivers/fake/vendor_passthru/methods", 200, `{"d":{"description":"x","new":null}}`)
	r2icServe(t, cloud, "GET /v1/drivers/gone", 404, `{}`)
	dpt2 := r2icServe(t, cloud, "GET /v1/drivers/gone/vendor_passthru/methods", 200, `{}`)
	api := nodes.New(r2icClient(cloud, "1.99"))
	ctx := context.Background()
	raw, err := api.ListFirmwareRaw(ctx, "n1")
	if err != nil || string(raw) != `{"firmware":[{"component":"bios","extra":{"x":1}}]}` || len(fw.all()) != 1 {
		t.Fatal(string(raw), err)
	}
	methods, err := api.ListVendorPassthruRaw(ctx, "n1")
	if err != nil || string(methods["m"]) != `{"async":true,"future":[1]}` || len(pt.all()) != 1 {
		t.Fatal(methods, err)
	}
	dapi := drivers.New(r2icClient(cloud, "1.99"))
	dm, err := dapi.ListVendorPassthruRaw(ctx, "fake")
	if err != nil || string(dm["d"]) != `{"description":"x","new":null}` || len(drv.all()) != 1 || len(dpt.all()) != 1 {
		t.Fatal(dm, err)
	}
	if _, err := dapi.ListVendorPassthruRaw(ctx, "gone"); !gophercloud.ResponseCodeIs(err, 404) || len(dpt2.all()) != 0 {
		t.Fatalf("missing driver stops before the listing: %v", err)
	}
}
