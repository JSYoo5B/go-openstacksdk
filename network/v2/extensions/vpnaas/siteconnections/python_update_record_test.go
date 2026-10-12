package siteconnections_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonSiteConnectionUpdateRecordSendsEmptyValues(t *testing.T) {
	ctx := context.Background()
	status, reply := http.StatusOK, `{"ipsec_site_connection":`+pythonSiteConnectionRow+`}`
	var traces []string
	api, calls, _ := pythonSiteConnectionAPI(t, func(req *http.Request) *http.Response {
		traces = append(traces, req.Header.Get("X-Trace"))
		return pythonSiteConnectionWire(status, reply)
	})
	// update_vpn_ipsec_site_connection("sc-1", local_id="", peer_cidrs=[]).
	got, err := api.UpdateRecord(ctx, "sc-1", siteconnections.WithUpdateRecordOptions(siteconnections.UpdateRecordOpts{
		LocalID: request.Present(""), PeerCIDRs: request.Present([]string{}),
	}))
	if err != nil || !got.Sent || got.StatusCode != 200 || got.Connection == nil || got.Connection.ID != "sc-1" || got.Connection.MTU != 1500 || got.Connection.DPD.Interval != 30 || string(got.Fields["auth_mode"]) != `"psk"` {
		t.Fatal(got, err)
	}
	// update_vpn_ipsec_site_connection("sc-1", name="", description="",
	// peer_address="", peer_id="", local_ep_group_id=None, peer_ep_group_id="",
	// mtu=0, initiator="", psk="", dpd={}, is_admin_state_up=False,
	// route_mode="static") with a flat 202 response; Python accepts every
	// status below 400.
	status, reply = http.StatusAccepted, `{"id":"sc-1","local_id":"","peer_cidrs":[],"mtu":0,"admin_state_up":false}`
	got, err = api.UpdateRecord(ctx, "sc-1",
		siteconnections.WithUpdateRecordOptions(siteconnections.UpdateRecordOpts{
			Name: request.Present(""), Description: request.Present(""), PeerAddress: request.Present(""), PeerID: request.Present(""),
			LocalEPGroupID: request.Null[string](), PeerEPGroupID: request.Present(""), MTU: request.Present(0), Initiator: request.Present(""),
			PSK: request.Present(""), DPD: request.Present(map[string]any{}), AdminStateUp: request.Present(false),
		}),
		siteconnections.WithUpdateRecordField("route_mode", "static"), siteconnections.WithUpdateRecordHeader("X-Trace", "t"))
	if err != nil || got.StatusCode != 202 || got.Connection.MTU != 0 || got.Connection.AdminStateUp || got.Connection.PeerCIDRs == nil || len(got.Fields) != 5 {
		t.Fatal(got, err)
	}
	// update_vpn_ipsec_site_connection("sc-1") has nothing dirty, so Python
	// sends no request.
	if got, err := api.UpdateRecord(ctx, "sc-1"); err != nil || got.Sent || got.Connection != nil {
		t.Fatal(got, err)
	}
	want := []pythonSiteConnectionCall{
		{http.MethodPut, pythonSiteConnectionBase + "/sc-1", "", `{"ipsec_site_connection":{"local_id":"","peer_cidrs":[]}}`},
		{http.MethodPut, pythonSiteConnectionBase + "/sc-1", "", `{"ipsec_site_connection":{"admin_state_up":false,"description":"","dpd":{},"initiator":"","local_ep_group_id":null,"mtu":0,"name":"","peer_address":"","peer_ep_group_id":"","peer_id":"","psk":"","route_mode":"static"}}`},
	}
	if !reflect.DeepEqual(*calls, want) || !reflect.DeepEqual(traces, []string{"", "t"}) {
		t.Fatalf("%+v %q", *calls, traces)
	}
}

func TestPythonSiteConnectionUpdateRecordRejectsInputsAndErrorStatuses(t *testing.T) {
	ctx := context.Background()
	api, calls, _ := pythonSiteConnectionAPI(t, func(*http.Request) *http.Response {
		return pythonSiteConnectionWire(http.StatusNotFound, `{"NeutronError":{"type":"IPsecSiteConnectionNotFound","message":"missing"}}`)
	})
	for _, option := range []siteconnections.UpdateRecordOption{
		siteconnections.WithUpdateRecordField("local_id", ""),
		siteconnections.WithUpdateRecordField("peer_cidrs", []string{}),
		siteconnections.WithUpdateRecordField("id", "sc-2"),
		nil,
	} {
		var operation *resource.OperationError
		if _, err := api.UpdateRecord(ctx, "sc-1", option); !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "UpdateRecord" {
			t.Fatal(err)
		}
	}
	if _, err := api.UpdateRecord(ctx, ".", siteconnections.WithUpdateRecordOptions(siteconnections.UpdateRecordOpts{MTU: request.Present(0)})); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if len(*calls) != 0 {
		t.Fatalf("%+v", *calls)
	}
	var operation *resource.OperationError
	if _, err := api.UpdateRecord(ctx, "sc-1", siteconnections.WithUpdateRecordOptions(siteconnections.UpdateRecordOpts{MTU: request.Present(0)})); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) || !errors.As(err, &operation) || operation.Resource != "siteconnections" {
		t.Fatal(err)
	}
	var nilAPI *siteconnections.API
	if _, err := nilAPI.UpdateRecord(ctx, "sc-1", siteconnections.WithUpdateRecordOptions(siteconnections.UpdateRecordOpts{MTU: request.Present(0)})); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
