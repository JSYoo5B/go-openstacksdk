package bgpvpns_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonBGPVPNCreateRecordSendsZeroLocalPref(t *testing.T) {
	ctx := context.Background()
	zeroRow := strings.Replace(pythonBGPVPNRow, `"local_pref":100`, `"local_pref":0`, 1)
	status, reply := http.StatusCreated, `{"bgpvpn":`+zeroRow+`}`
	var traces []string
	api, calls := pythonBGPVPNAPI(t, func(req *http.Request) *http.Response {
		traces = append(traces, req.Header.Get("X-Trace"))
		return pythonBGPVPNWire(status, reply)
	})
	// create_bgpvpn(name="vpn", local_pref=0).
	got, err := api.CreateRecord(ctx,
		bgpvpns.WithCreateRecordOptions(bgpvpns.CreateRecordOpts{Name: request.Present("vpn")}),
		bgpvpns.WithCreateRecordLocalPref(request.Present(0)))
	if err != nil || got.StatusCode != 201 || got.BGPVPN == nil || got.BGPVPN.ID != "vpn-1" || got.BGPVPN.LocalPref == nil || *got.BGPVPN.LocalPref != 0 || string(got.Fields["networks"]) != `["n-1"]` {
		t.Fatal(got, err)
	}
	// create_bgpvpn(name="", type="l2", vni=0, route_targets=[],
	// import_targets=None, project_id="p", tenant_id="p") with a flat 200
	// response; BgpVpn allows unknown body attributes and Python accepts every
	// status below 400.
	status, reply = http.StatusOK, `{"id":"vpn-2","name":"","type":"l2","vni":0,"route_targets":[]}`
	got, err = api.CreateRecord(ctx,
		bgpvpns.WithCreateRecordOptions(bgpvpns.CreateRecordOpts{
			Name: request.Present(""), Type: request.Present("l2"), VNI: request.Present(0), RouteTargets: request.Present([]string{}),
			ImportTargets: request.Null[[]string](), ProjectID: request.Present("p"),
		}),
		bgpvpns.WithCreateRecordField("tenant_id", "p"), bgpvpns.WithCreateRecordHeader("X-Trace", "t"))
	if err != nil || got.StatusCode != 200 || got.BGPVPN.ID != "vpn-2" || got.BGPVPN.Type != "l2" || got.BGPVPN.LocalPref != nil || len(got.Fields) != 5 {
		t.Fatal(got, err)
	}
	// create_bgpvpn() still sends POST with an empty envelope, and an empty
	// accepted body leaves only the status and headers.
	status, reply = http.StatusAccepted, ``
	if got, err := api.CreateRecord(ctx); err != nil || got.StatusCode != 202 || got.BGPVPN != nil || got.Fields != nil {
		t.Fatal(got, err)
	}
	want := []pythonBGPVPNCall{
		{http.MethodPost, pythonBGPVPNBase, "", `{"bgpvpn":{"local_pref":0,"name":"vpn"}}`},
		{http.MethodPost, pythonBGPVPNBase, "", `{"bgpvpn":{"import_targets":null,"name":"","project_id":"p","route_targets":[],"tenant_id":"p","type":"l2","vni":0}}`},
		{http.MethodPost, pythonBGPVPNBase, "", `{"bgpvpn":{}}`},
	}
	if !reflect.DeepEqual(*calls, want) || !reflect.DeepEqual(traces, []string{"", "t", ""}) {
		t.Fatalf("%+v %q", *calls, traces)
	}
}

func TestPythonBGPVPNCreateRecordRejectsInputsAndErrorStatuses(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonBGPVPNAPI(t, func(*http.Request) *http.Response {
		return pythonBGPVPNWire(http.StatusBadRequest, `{"NeutronError":{"type":"HTTPBadRequest","message":"bad"}}`)
	})
	for _, option := range []bgpvpns.CreateRecordOption{
		bgpvpns.WithCreateRecordField("local_pref", 0),
		bgpvpns.WithCreateRecordField(" ", 0),
		nil,
	} {
		var operation *resource.OperationError
		if _, err := api.CreateRecord(ctx, option); !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "CreateRecord" {
			t.Fatal(err)
		}
	}
	if len(*calls) != 0 {
		t.Fatalf("%+v", *calls)
	}
	var operation *resource.OperationError
	if _, err := api.CreateRecord(ctx, bgpvpns.WithCreateRecordLocalPref(request.Present(0))); !gophercloud.ResponseCodeIs(err, http.StatusBadRequest) || !errors.As(err, &operation) || operation.Resource != "bgpvpns" {
		t.Fatal(err)
	}
	var nilAPI *bgpvpns.API
	if _, err := nilAPI.CreateRecord(ctx); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
