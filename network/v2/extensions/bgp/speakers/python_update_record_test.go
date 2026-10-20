package speakers_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonBGPSpeakerUpdateRecordSendsOnlySuppliedFlags(t *testing.T) {
	ctx := context.Background()
	status, reply := http.StatusOK, `{"bgp_speaker":`+pythonBGPSpeakerRow+`}`
	var traces []string
	api, _, calls, _ := pythonBGPSpeakerAPI(t, func(req *http.Request) *http.Response {
		traces = append(traces, req.Header.Get("X-Trace"))
		return pythonBGPSpeakerWire(status, reply)
	})
	// update_bgp_speaker("bs-1", name="edge") leaves both advertise flags alone.
	got, err := api.UpdateRecord(ctx, "bs-1", speakers.WithUpdateRecordName(request.Present("edge")))
	if err != nil || !got.Sent || got.StatusCode != 200 || got.BGPSpeaker == nil || got.BGPSpeaker.ID != "bs-1" || !got.BGPSpeaker.AdvertiseTenantNetworks || got.BGPSpeaker.LocalAS != 65000 || string(got.Fields["peers"]) != `["bp-1"]` {
		t.Fatal(got, err)
	}
	// update_bgp_speaker("bs-1", advertise_tenant_networks=False).
	if _, err := api.UpdateRecord(ctx, "bs-1", speakers.WithUpdateRecordAdvertiseTenantNetworks(request.Present(false))); err != nil {
		t.Fatal(err)
	}
	// update_bgp_speaker("bs-1", name="", advertise_floating_ip_host_routes=False,
	// advertise_tenant_networks=None, networks=[]) with a flat 202 response;
	// BgpSpeaker allows unknown body attributes and Python accepts every status
	// below 400.
	status, reply = http.StatusAccepted, `{"id":"bs-1","name":"","advertise_floating_ip_host_routes":false}`
	got, err = api.UpdateRecord(ctx, "bs-1",
		speakers.WithUpdateRecordOptions(speakers.UpdateRecordOpts{Name: request.Present(""), AdvertiseTenantNetworks: request.Null[bool]()}),
		speakers.WithUpdateRecordAdvertiseFloatingIPHostRoutes(request.Present(false)),
		speakers.WithUpdateRecordField("networks", []string{}), speakers.WithUpdateRecordHeader("X-Trace", "t"))
	if err != nil || got.StatusCode != 202 || got.BGPSpeaker.Name != "" || got.BGPSpeaker.AdvertiseFloatingIPHostRoutes || len(got.Fields) != 3 {
		t.Fatal(got, err)
	}
	// update_bgp_speaker("bs-1") has nothing dirty, so Python sends no request.
	if got, err := api.UpdateRecord(ctx, "bs-1"); err != nil || got.Sent || got.BGPSpeaker != nil {
		t.Fatal(got, err)
	}
	want := []pythonBGPSpeakerCall{
		{http.MethodPut, pythonBGPSpeakerBase + "/bs-1", "", `{"bgp_speaker":{"name":"edge"}}`},
		{http.MethodPut, pythonBGPSpeakerBase + "/bs-1", "", `{"bgp_speaker":{"advertise_tenant_networks":false}}`},
		{http.MethodPut, pythonBGPSpeakerBase + "/bs-1", "", `{"bgp_speaker":{"advertise_floating_ip_host_routes":false,"advertise_tenant_networks":null,"name":"","networks":[]}}`},
	}
	if !reflect.DeepEqual(*calls, want) || !reflect.DeepEqual(traces, []string{"", "", "t"}) {
		t.Fatalf("%+v %q", *calls, traces)
	}
}

func TestPythonBGPSpeakerUpdateRecordRejectsInputsAndErrorStatuses(t *testing.T) {
	ctx := context.Background()
	api, _, calls, _ := pythonBGPSpeakerAPI(t, func(*http.Request) *http.Response {
		return pythonBGPSpeakerWire(http.StatusBadRequest, `{"NeutronError":{"type":"HTTPBadRequest","message":"bad"}}`)
	})
	for _, option := range []speakers.UpdateRecordOption{
		speakers.WithUpdateRecordField("advertise_tenant_networks", false),
		speakers.WithUpdateRecordField("id", "bs-2"),
		nil,
	} {
		var operation *resource.OperationError
		if _, err := api.UpdateRecord(ctx, "bs-1", option); !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "UpdateRecord" {
			t.Fatal(err)
		}
	}
	if _, err := api.UpdateRecord(ctx, "", speakers.WithUpdateRecordName(request.Present("x"))); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if len(*calls) != 0 {
		t.Fatalf("%+v", *calls)
	}
	var operation *resource.OperationError
	if _, err := api.UpdateRecord(ctx, "bs-1", speakers.WithUpdateRecordName(request.Present("x"))); !gophercloud.ResponseCodeIs(err, http.StatusBadRequest) || !errors.As(err, &operation) || operation.Resource != "speakers" {
		t.Fatal(err)
	}
	var nilAPI *speakers.API
	if _, err := nilAPI.UpdateRecord(ctx, "bs-1", speakers.WithUpdateRecordName(request.Present("x"))); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
