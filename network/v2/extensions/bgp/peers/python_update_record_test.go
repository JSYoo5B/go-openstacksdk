package peers_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonBGPPeerUpdateRecordSendsEmptyName(t *testing.T) {
	ctx := context.Background()
	status, reply := http.StatusOK, `{"bgp_peer":`+pythonBGPPeerRow+`}`
	var traces []string
	api, calls, _ := pythonBGPPeerAPI(t, func(req *http.Request) *http.Response {
		traces = append(traces, req.Header.Get("X-Trace"))
		return pythonBGPPeerWire(status, reply)
	})
	// update_bgp_peer("bp-1", name="") sends the empty name.
	got, err := api.UpdateRecord(ctx, "bp-1", peers.WithUpdateRecordName(request.Present("")))
	if err != nil || !got.Sent || got.StatusCode != 200 || got.BGPPeer == nil || got.BGPPeer.ID != "bp-1" || got.BGPPeer.RemoteAS != 64512 || string(got.Fields["peer_ip"]) != `"192.0.2.10"` {
		t.Fatal(got, err)
	}
	// update_bgp_peer("bp-1", name=None, password="", hold_time=0) with a flat
	// 202 response; BgpPeer allows unknown body attributes and Python accepts
	// every status below 400.
	status, reply = http.StatusAccepted, `{"id":"bp-1","name":"","remote_as":64512}`
	got, err = api.UpdateRecord(ctx, "bp-1",
		peers.WithUpdateRecordOptions(peers.UpdateRecordOpts{Name: request.Null[string](), Password: request.Present("")}),
		peers.WithUpdateRecordField("hold_time", 0), peers.WithUpdateRecordHeader("X-Trace", "t"))
	if err != nil || got.StatusCode != 202 || got.BGPPeer.Name != "" || got.BGPPeer.RemoteAS != 64512 || len(got.Fields) != 3 {
		t.Fatal(got, err)
	}
	// update_bgp_peer("bp-1") has nothing dirty, so Python sends no request.
	if got, err := api.UpdateRecord(ctx, "bp-1"); err != nil || got.Sent || got.BGPPeer != nil {
		t.Fatal(got, err)
	}
	// An empty or non-JSON accepted body leaves only the status and headers.
	status, reply = http.StatusNoContent, ``
	if got, err := api.UpdateRecord(ctx, "bp-1", peers.WithUpdateRecordPassword(request.Present("pw"))); err != nil || !got.Sent || got.StatusCode != 204 || got.BGPPeer != nil || got.Fields != nil {
		t.Fatal(got, err)
	}
	want := []pythonBGPPeerCall{
		{http.MethodPut, pythonBGPPeerBase + "/bp-1", "", `{"bgp_peer":{"name":""}}`},
		{http.MethodPut, pythonBGPPeerBase + "/bp-1", "", `{"bgp_peer":{"hold_time":0,"name":null,"password":""}}`},
		{http.MethodPut, pythonBGPPeerBase + "/bp-1", "", `{"bgp_peer":{"password":"pw"}}`},
	}
	if !reflect.DeepEqual(*calls, want) || !reflect.DeepEqual(traces, []string{"", "t", ""}) {
		t.Fatalf("%+v %q", *calls, traces)
	}
}

func TestPythonBGPPeerUpdateRecordRejectsInputsAndErrorStatuses(t *testing.T) {
	ctx := context.Background()
	api, calls, _ := pythonBGPPeerAPI(t, func(*http.Request) *http.Response {
		return pythonBGPPeerWire(http.StatusConflict, `{"NeutronError":{"type":"Conflict","message":"busy"}}`)
	})
	for _, option := range []peers.UpdateRecordOption{
		peers.WithUpdateRecordField("name", ""),
		peers.WithUpdateRecordField("id", "bp-2"),
		peers.WithUpdateRecordField("", 1),
		nil,
	} {
		var operation *resource.OperationError
		if _, err := api.UpdateRecord(ctx, "bp-1", option); !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "UpdateRecord" {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"", " ", ".."} {
		if _, err := api.UpdateRecord(ctx, id, peers.WithUpdateRecordName(request.Present("x"))); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(id, err)
		}
	}
	if len(*calls) != 0 {
		t.Fatalf("%+v", *calls)
	}
	var operation *resource.OperationError
	if _, err := api.UpdateRecord(ctx, "bp-1", peers.WithUpdateRecordName(request.Present("x"))); !gophercloud.ResponseCodeIs(err, http.StatusConflict) || !errors.As(err, &operation) || operation.Resource != "peers" {
		t.Fatal(err)
	}
	var nilAPI *peers.API
	if _, err := nilAPI.UpdateRecord(ctx, "bp-1", peers.WithUpdateRecordName(request.Present("x"))); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
