package api_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/allocations"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/conductors"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/ports"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestR2IronicCUpdateAttributesAddOpsSortedAndMaintenanceRouting(t *testing.T) {
	cloud := testcloud.New(t)
	patch := r2icServe(t, cloud, "PATCH /v1/nodes/n1", 200, `{"uuid":"n1","name":"new"}`)
	put := r2icServe(t, cloud, "PUT /v1/nodes/n1/maintenance", 202, "")
	del := r2icServe(t, cloud, "DELETE /v1/nodes/n1/maintenance", 202, "")
	get := r2icServe(t, cloud, "GET /v1/nodes/n1", 200, `{"uuid":"n1","maintenance":true}`)
	api := nodes.New(r2icClient(cloud, "1.99"))
	ctx := context.Background()
	node, err := api.UpdateAttributes(ctx, "n1", map[string]any{"name": "new", "instance_id": nil, "extra": map[string]any{}, "is_protected": false})
	if err != nil || node.Name != "new" {
		t.Fatal(node, err)
	}
	want := `[{"op":"add","path":"/extra","value":{}},{"op":"add","path":"/instance_uuid","value":null},{"op":"add","path":"/name","value":"new"},{"op":"add","path":"/protected","value":false}]`
	if got := patch.all()[0].Body; got != want {
		t.Fatalf("patch %s", got)
	}
	// maintenance only: PUT with reason then refetch, no PATCH.
	if _, err := api.UpdateAttributes(ctx, "n1", map[string]any{"is_maintenance": true}); err != nil {
		t.Fatal(err)
	}
	if put.all()[0].Body != `{"reason":null}` || len(get.all()) != 1 || len(patch.all()) != 1 {
		t.Fatalf("%+v", put.all())
	}
	if _, err := api.UpdateAttributes(ctx, "n1", map[string]any{"maintenance": true, "maintenance_reason": "rack", "name": "x"}); err != nil {
		t.Fatal(err)
	}
	if put.all()[1].Body != `{"reason":"rack"}` || len(patch.all()) != 2 || patch.all()[1].Body != `[{"op":"add","path":"/name","value":"x"}]` {
		t.Fatalf("%+v %+v", put.all(), patch.all())
	}
	// false maintenance deletes; a truthy reason alongside it is dropped.
	if _, err := api.UpdateAttributes(ctx, "n1", map[string]any{"is_maintenance": false, "maintenance_reason": "ignored"}); err != nil || len(del.all()) != 1 {
		t.Fatal(err)
	}
	// a reason without maintenance is refused before HTTP.
	n := len(put.all()) + len(del.all()) + len(patch.all())
	if _, err := api.UpdateAttributes(ctx, "n1", map[string]any{"maintenance_reason": "x"}); !errors.Is(err, resource.ErrInvalidOption) || n != len(put.all())+len(del.all())+len(patch.all()) {
		t.Fatalf("reason only: %v", err)
	}
	for _, bad := range []map[string]any{{"bogus": 1}, {"uuid": "z"}, {"id": "z"}, {"instance_id": 1, "instance_uuid": 2}} {
		if _, err := api.UpdateAttributes(ctx, "n1", bad); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%v: %v", bad, err)
		}
	}
}

func TestR2IronicCUpdateAttributesConflictIsNotRetried(t *testing.T) {
	cloud := testcloud.New(t)
	patch := r2icServe(t, cloud, "PATCH /v1/nodes/n1", 409, `{}`)
	_, err := nodes.New(r2icClient(cloud, "1.99")).UpdateAttributes(context.Background(), "n1", map[string]any{"name": "x"})
	if !gophercloud.ResponseCodeIs(err, 409) || len(patch.all()) != 1 {
		t.Fatalf("%v %d", err, len(patch.all()))
	}
}

func TestR2IronicCPatchNodeConvertsPathsResetInterfacesAndEmptyPatch(t *testing.T) {
	cloud := testcloud.New(t)
	patch := r2icServe(t, cloud, "PATCH /v1/nodes/n1", 200, `{"uuid":"n1"}`)
	ironicNodeUnreachable(t, testcloud.New(t))
	api := nodes.New(r2icClient(cloud, "1.99"))
	ctx := context.Background()
	res, err := api.PatchNode(ctx, "n1", []nodes.PatchOp{
		{Op: "replace", Path: "/instance_id", Value: []byte(`"i"`)},
		{Op: "remove", Path: "/driver_info/password"},
		{Op: "add", Path: "/is_maintenance", Value: []byte(`false`)},
		{Op: "add", Path: "/instance_info", Value: []byte(`null`)},
	}, nodes.WithResetInterfaces(false))
	if err != nil || !res.Sent || res.Node.UUID != "n1" {
		t.Fatal(res, err)
	}
	c := patch.all()[0]
	if c.RawQuery != "reset_interfaces=False" || c.Body != `[{"op":"replace","path":"/instance_uuid","value":"i"},{"op":"remove","path":"/driver_info/password"},{"op":"add","path":"/maintenance","value":false},{"op":"add","path":"/instance_info","value":null}]` {
		t.Fatalf("%+v", c)
	}
	if _, err := api.PatchNode(ctx, "n1", []nodes.PatchOp{{Op: "add", Path: "/name", Value: []byte(`"a"`)}}, nodes.WithResetInterfaces(true)); err != nil || patch.all()[1].RawQuery != "reset_interfaces=True" {
		t.Fatal(err)
	}
	empty, err := api.PatchNode(ctx, "n1", nil)
	if err != nil || empty.Sent || empty.Node != nil || len(patch.all()) != 2 {
		t.Fatalf("empty patch sends nothing: %+v %v", empty, err)
	}
	if _, err := api.PatchNode(ctx, "n1", []nodes.PatchOp{{Op: "add", Path: "/"}}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.PatchNode(ctx, "n1", []nodes.PatchOp{{Path: "/name"}}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := nodes.New(r2icClient(cloud, "1.44")).PatchNode(ctx, "n1", []nodes.PatchOp{{Op: "add", Path: "/name", Value: []byte(`"a"`)}}, nodes.WithResetInterfaces(true)); !errors.Is(err, resource.ErrUnsupported) || len(patch.all()) != 2 {
		t.Fatalf("reset_interfaces needs 1.45: %v", err)
	}
}

func TestR2IronicCUpdateFromOriginalSendsStructuralDiffOnly(t *testing.T) {
	cloud := testcloud.New(t)
	patch := r2icServe(t, cloud, "PATCH /v1/nodes/n1", 200, `{"uuid":"n1"}`)
	r2icServe(t, cloud, "GET /v1/nodes/n1", 200, `{"uuid":"n1"}`)
	api := nodes.New(r2icClient(cloud, "1.99"))
	orig := map[string]rawJSON{"driver_info": `{"a":1,"b":2}`, "name": `"x"`, "traits": `["A","B","C"]`}
	got, err := api.UpdateFromOriginal(context.Background(), "n1", r2icRaw(orig), map[string]any{
		"driver_info": map[string]any{"a": 1, "b": 3, "c": 4}, "name": "x", "traits": []string{"A", "D"}, "description": "d"})
	if err != nil || got == nil {
		t.Fatal(got, err)
	}
	want := `[{"op":"add","path":"/description","value":"d"},{"op":"replace","path":"/driver_info/b","value":3},{"op":"add","path":"/driver_info/c","value":4},{"op":"replace","path":"/traits/1","value":"D"},{"op":"remove","path":"/traits/2"}]`
	if b := patch.all()[0].Body; b != want {
		t.Fatalf("%s", b)
	}
	// no difference: no write, node is read.
	if _, err := api.UpdateFromOriginal(context.Background(), "n1", r2icRaw(orig), map[string]any{"name": "x"}); err != nil || len(patch.all()) != 1 {
		t.Fatalf("%v %d", err, len(patch.all()))
	}
}

func TestR2IronicCPortAndAllocationPatchAndUpdate(t *testing.T) {
	cloud := testcloud.New(t)
	pp := r2icServe(t, cloud, "PATCH /v1/ports/p1", 200, `{"uuid":"p1","address":"aa"}`)
	ap := r2icServe(t, cloud, "PATCH /v1/allocations/a1", 200, `{"uuid":"a1","name":"n"}`)
	papi := ports.New(r2icClient(cloud, "1.99"))
	aapi := allocations.New(r2icClient(cloud, "1.99"))
	ctx := context.Background()
	res, err := papi.UpdateAttributes(ctx, "p1", map[string]any{"is_pxe_enabled": false, "node_id": "n", "extra": map[string]any{}, "address": "aa"})
	if err != nil || !res.Sent || res.Port.UUID != "p1" {
		t.Fatal(res, err)
	}
	if want := `[{"op":"add","path":"/address","value":"aa"},{"op":"add","path":"/extra","value":{}},{"op":"add","path":"/node_uuid","value":"n"},{"op":"add","path":"/pxe_enabled","value":false}]`; pp.all()[0].Body != want {
		t.Fatalf("%s", pp.all()[0].Body)
	}
	if _, err := papi.PatchPort(ctx, "p1", []ports.PatchOp{{Op: "replace", Path: "/port_group_id/x", Value: []byte(`1`)}, {Op: "remove", Path: "/extra/k"}}); err != nil {
		t.Fatal(err)
	}
	if pp.all()[1].Body != `[{"op":"replace","path":"/portgroup_uuid/x","value":1},{"op":"remove","path":"/extra/k"}]` {
		t.Fatal(pp.all()[1].Body)
	}
	if r, err := papi.PatchPort(ctx, "p1", nil); err != nil || r.Sent || len(pp.all()) != 2 {
		t.Fatal(r, err)
	}
	if _, err := papi.UpdateAttributes(ctx, "p1", map[string]any{"nope": 1}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ares, err := aapi.UpdateAttributes(ctx, "a1", map[string]any{"name": "n", "node_id": nil, "candidate_nodes": []string{}})
	if err != nil || ares.Allocation.UUID != "a1" {
		t.Fatal(ares, err)
	}
	if want := `[{"op":"add","path":"/candidate_nodes","value":[]},{"op":"add","path":"/name","value":"n"},{"op":"add","path":"/node_uuid","value":null}]`; ap.all()[0].Body != want {
		t.Fatalf("%s", ap.all()[0].Body)
	}
	if _, err := aapi.PatchAllocation(ctx, "a1", []allocations.PatchOp{{Op: "replace", Path: "/node_id", Value: []byte(`"z"`)}}); err != nil || ap.all()[1].Body != `[{"op":"replace","path":"/node_uuid","value":"z"}]` {
		t.Fatal(err)
	}
	if r, err := aapi.PatchAllocation(ctx, "a1", nil); err != nil || r.Sent || len(ap.all()) != 2 {
		t.Fatal(r, err)
	}
}

func TestR2IronicCWaitForActiveAllocation(t *testing.T) {
	cloud := testcloud.New(t)
	r2icServe(t, cloud, "GET /v1/allocations/ok", 200, `{"uuid":"ok","state":"allocating"}`, 200, `{"uuid":"ok","state":"active"}`)
	r2icServe(t, cloud, "GET /v1/allocations/bad", 200, `{"uuid":"bad","state":"allocating"}`, 200, `{"uuid":"bad","state":"error","last_error":"no node"}`)
	r2icServe(t, cloud, "GET /v1/allocations/slow", 200, `{"uuid":"slow","state":"allocating"}`)
	api := allocations.New(r2icClient(cloud, "1.99"))
	ctx := context.Background()
	if a, err := api.WaitForActive(ctx, "ok", allocations.WithWaitInterval(r2icFast)); err != nil || a.State != "active" {
		t.Fatal(a, err)
	}
	_, err := api.WaitForActive(ctx, "bad", allocations.WithWaitInterval(r2icFast))
	var failed *allocations.AllocationFailedError
	if !errors.As(err, &failed) || !errors.Is(err, resource.ErrFailedState) || failed.LastError != "no node" {
		t.Fatal(err)
	}
	if a, err := api.WaitForActive(ctx, "bad", allocations.WithWaitInterval(r2icFast), allocations.WithIgnoreError(true)); err != nil || a.State != "error" {
		t.Fatalf("ignore error returns the allocation: %v %v", a, err)
	}
	short, cancel := context.WithTimeout(ctx, 30*time.Millisecond)
	defer cancel()
	if _, err := api.WaitForActive(short, "slow", allocations.WithWaitInterval(5*time.Millisecond)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}

func TestR2IronicCPortRecordGetFindListFollowSource(t *testing.T) {
	cloud := testcloud.New(t)
	get := r2icServe(t, cloud, "GET /v1/ports/p1", 200, `{"uuid":"p1","node_uuid":"n","pxe_enabled":0,"x_extra":1}`)
	r2icServe(t, cloud, "GET /v1/ports/by-name", 404, `{}`)
	detail := r2icServe(t, cloud, "GET /v1/ports/detail", 200, `{"ports":[{"uuid":"p2","name":"by-name"},{"uuid":"p3","name":"other"}]}`)
	basic := r2icServe(t, cloud, "GET /v1/ports", 200, `{"ports":[{"uuid":"p1","node_uuid":"n","extra":{}}]}`)
	api := ports.New(r2icClient(cloud, "1.101"))
	ctx := context.Background()
	rec, err := api.GetRecord(ctx, "p1", ports.WithRecordGetFields("id", "node_id", "is_pxe_enabled", "unknown"))
	if err != nil || get.all()[0].RawQuery != "fields=uuid%2Cnode_uuid%2Cpxe_enabled%2Cunknown" {
		t.Fatalf("%v %+v", err, get.all())
	}
	if string(rec.Resource.Body["is_pxe_enabled"]) != "false" || string(rec.Resource.Body["node_id"]) != `"n"` || rec.Resource.Body["x_extra"] != nil || rec.Wire.Body["x_extra"] == nil {
		t.Fatalf("%+v", rec)
	}
	found, err := api.FindRecord(ctx, "by-name")
	if err != nil || string(found.Resource.Body["id"]) != `"p2"` || detail.all()[0].RawQuery != "" {
		t.Fatalf("fallback is the detailed list without a name hint: %v %+v", err, detail.all())
	}
	if gone, err := api.FindRecord(ctx, "by-name", ports.WithRecordFindDetails(false)); err != nil || gone != nil || len(basic.all()) != 1 {
		t.Fatalf("basic fallback: %v %v", gone, err)
	}
	rows, err := api.AllRecords(ctx, ports.WithRecordListQuery("node_id", "n"), ports.WithRecordListQuery("address", "aa"), ports.WithRecordListQuery("sort_key", "name"), ports.WithRecordListQuery("fields", []string{"id", "extra"}))
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	q := r2icQuery(basic.all()[1].RawQuery)
	if q.Get("node_uuid") != "n" || q.Get("address") != "aa" || q.Get("fields") != "uuid,extra" || q.Has("sort_key") || q.Has("node_id") {
		t.Fatalf("aliases and csv fields: %v", q)
	}
}

func TestR2IronicCPortFindDuplicateNames(t *testing.T) {
	cloud := testcloud.New(t)
	r2icServe(t, cloud, "GET /v1/ports/dup", 404, `{}`)
	r2icServe(t, cloud, "GET /v1/ports/detail", 200, `{"ports":[{"uuid":"a","name":"dup"},{"uuid":"b","name":"dup"}]}`)
	_, err := ports.New(r2icClient(cloud, "1.101")).FindRecord(context.Background(), "dup")
	if err == nil {
		t.Fatal("duplicate names must fail")
	}
}

func TestR2IronicCAllocationAndConductorRecordGetFields(t *testing.T) {
	cloud := testcloud.New(t)
	a := r2icServe(t, cloud, "GET /v1/allocations/a1", 200, `{"uuid":"a1","node_uuid":"n","candidate_nodes":"x"}`)
	c := r2icServe(t, cloud, "GET /v1/conductors/host-1", 200, `{"hostname":"host-1","alive":1,"drivers":["fake"]}`)
	r2icServe(t, cloud, "GET /v1/conductors/missing", 404, `{}`)
	ctx := context.Background()
	arec, err := allocations.New(r2icClient(cloud, "1.60")).GetRecord(ctx, "a1", allocations.WithRecordGetFields("id", "node_id", "candidate_nodes"))
	if err != nil || a.all()[0].RawQuery != "fields=uuid%2Cnode_uuid%2Ccandidate_nodes" || string(arec.Resource.Body["candidate_nodes"]) != `["x"]` {
		t.Fatalf("%v %+v", err, a.all())
	}
	crec, err := conductors.New(r2icClient(cloud, "1.49")).GetRecord(ctx, "host-1", conductors.WithRecordGetFields("hostname", "alive"))
	if err != nil || c.all()[0].RawQuery != "fields=hostname%2Calive" || string(crec.Resource.Body["alive"]) != "true" {
		t.Fatalf("%v %+v", err, c.all())
	}
	if _, err := conductors.New(r2icClient(cloud, "1.49")).GetRecord(ctx, "missing"); !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatalf("a missing conductor stays strict: %v", err)
	}
	ctxc, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := allocations.New(r2icClient(cloud, "1.60")).GetRecord(ctxc, "a1"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestR2IronicCPortRecordDiscoveryCapsAtClassMaximum(t *testing.T) {
	cloud := testcloud.New(t)
	r2icDiscovery(t, cloud, "1.1", "1.120")
	get := r2icServe(t, cloud, "GET /v1/ports/p1", 200, `{"uuid":"p1"}`)
	if _, err := ports.New(r2icClient(cloud, "")).GetRecord(context.Background(), "p1"); err != nil || get.all()[0].Version != "1.101" {
		t.Fatalf("%v %+v", err, get.all())
	}
}

func TestR2IronicCPortListRecordsIsLazyAndStopsAfterBreak(t *testing.T) {
	cloud := testcloud.New(t)
	page := r2icServe(t, cloud, "GET /v1/ports/detail", 200, `{"ports":[{"uuid":"a"},{"uuid":"b"}],"ports_links":[{"rel":"next","href":"`+cloud.Server.URL+`/v1/ports/detail?marker=b"}]}`)
	api := ports.New(r2icClient(cloud, "1.101"))
	seq := api.ListRecords(context.Background(), ports.WithRecordListDetails(true), ports.WithRecordListQuery("sort_key", "x"))
	if len(page.all()) != 0 {
		t.Fatal("building the sequence must not send a request")
	}
	for row, err := range seq {
		if err != nil || string(row.Resource.Body["id"]) != `"a"` {
			t.Fatal(row, err)
		}
		break
	}
	if len(page.all()) != 1 || r2icQuery(page.all()[0].RawQuery).Has("sort_key") {
		t.Fatalf("%+v", page.all())
	}
}
