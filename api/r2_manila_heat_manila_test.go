package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharenetworks"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shares"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharetransfers"
)

const r2ManilaID = "33333333-3333-4333-8333-333333333333"

func TestR2ManilaHeatShareNetworkUpdateMembersDistinguishesOmissionEmptyAndNull(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var puts atomic.Int32
	cloud.Mux.HandleFunc("/manila/v2/share-networks/"+r2ManilaID, func(w http.ResponseWriter, r *http.Request) {
		puts.Add(1)
		body := r2ManilaHeatBody(t, r)
		inner := map[string]string{}
		for k, v := range r2ManilaHeatInner(t, body["share_network"]) {
			inner[k] = string(v)
		}
		if r.Method != "PUT" || len(body) != 1 || len(inner) != 4 || inner["neutron_net_id"] != "null" || inner["neutron_subnet_id"] != `""` || inner["name"] != `""` || inner["extra"] != "1" {
			t.Errorf("%s %v", r.Method, inner)
		}
		testcloud.JSON(w, 200, `{"share_network":{"id":"`+r2ManilaID+`","neutron_net_id":null,"unknown":9007199254740993}}`)
	})
	api := sharenetworks.New(cloud.Client("sharev2", "/manila/v2"))
	value, err := api.UpdateMembers(context.Background(), r2ManilaID, sharenetworks.UpdateMembersOpts{
		NeutronNetID: request.Null[string](), NeutronSubnetID: request.Present(""), Name: request.Present(""),
	}, sharenetworks.WithUpdateMembersField("extra", 1))
	if err != nil || string(value.Body["unknown"]) != "9007199254740993" || value.StatusCode != 200 {
		t.Fatal(value, err)
	}
	for _, option := range []sharenetworks.UpdateMembersOption{sharenetworks.WithUpdateMembersField("name", "x"), sharenetworks.WithUpdateMembersField("nova_net_id", "x")} {
		if _, err := api.UpdateMembers(context.Background(), r2ManilaID, sharenetworks.UpdateMembersOpts{Name: request.Present("a")}, option); !errors.Is(err, resource.ErrInvalidOption) && err == nil {
			t.Fatal("collision accepted")
		}
	}
	if _, err := api.UpdateMembers(context.Background(), r2ManilaID, sharenetworks.UpdateMembersOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.UpdateMembers(context.Background(), "a/b", sharenetworks.UpdateMembersOpts{Name: request.Present("a")}); err == nil {
		t.Fatal("bad id accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	before := puts.Load()
	if _, err := api.UpdateMembers(ctx, r2ManilaID, sharenetworks.UpdateMembersOpts{Name: request.Present("a")}); !errors.Is(err, context.Canceled) || puts.Load() != before {
		t.Fatal(err)
	}
}

func TestR2ManilaHeatFindByNameOrIDFollowsPythonFindFallbackAndDuplicates(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var lists []string
	cloud.Mux.HandleFunc("/manila/v2/shares/"+r2ManilaID, func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"share":{"id":"`+r2ManilaID+`","name":"direct"}}`)
	})
	for _, code := range []int{400, 403, 404} {
		name := map[int]string{400: "bad", 403: "forbid", 404: "gone"}[code]
		code := code
		cloud.Mux.HandleFunc("/manila/v2/shares/"+name, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, code, `{}`) })
	}
	cloud.Mux.HandleFunc("/manila/v2/shares/boom", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 500, `{}`) })
	cloud.Mux.HandleFunc("/manila/v2/shares", func(w http.ResponseWriter, r *http.Request) {
		lists = append(lists, r.URL.RawQuery)
		switch r.URL.Query().Get("name") {
		case "dup":
			testcloud.JSON(w, 200, `{"shares":[{"id":"a","name":"dup"},{"id":"b","name":"dup"}]}`)
		case "gone", "bad", "forbid", "has space":
			testcloud.JSON(w, 200, `{"shares":[{"id":"x","name":"`+r.URL.Query().Get("name")+`"},{"id":"y","name":"other"}]}`)
		default:
			testcloud.JSON(w, 200, `{"shares":[]}`)
		}
	})
	api := shares.New(cloud.Client("sharev2", "/manila/v2"))
	ctx := context.Background()
	if v, err := api.FindByNameOrID(ctx, r2ManilaID); err != nil || v.Name != "direct" || len(lists) != 0 {
		t.Fatal(v, err, lists)
	}
	for _, name := range []string{"gone", "bad", "forbid", "has space"} {
		if v, err := api.FindByNameOrID(ctx, name); err != nil || v == nil || v.ID != "x" {
			t.Fatalf("%s: %v %v", name, v, err)
		}
	}
	if _, err := api.FindByNameOrID(ctx, "boom"); err == nil {
		t.Fatal("500 swallowed")
	}
	if _, err := api.FindByNameOrID(ctx, "dup"); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatal(err)
	}
	if v, err := api.FindByNameOrID(ctx, "nothing"); err != nil || v != nil {
		t.Fatal(v, err)
	}
	if _, err := api.FindByNameOrID(ctx, "nothing", shares.WithFindIgnoreMissing(false)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	lists = nil
	if _, err := api.FindByNameOrID(ctx, "nothing", shares.WithFindAllProjects(true), shares.WithFindQuery("project_id", "p")); err != nil || len(lists) != 1 || lists[0] != "all_tenants=true&name=nothing&project_id=p" {
		t.Fatal(err, lists)
	}
	lists = nil
	if _, err := api.FindByNameOrID(ctx, "nothing", shares.WithFindQuery("name", "other")); err != nil || lists[0] != "name=other" {
		t.Fatal(err, lists)
	}
}

func TestR2ManilaHeatFindShareTransferByNameOrID(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/manila/v2/share-transfers/"+r2ManilaID, func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"transfer":{"id":"`+r2ManilaID+`","name":"direct"}}`)
	})
	cloud.Mux.HandleFunc("/manila/v2/share-transfers/named", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("/manila/v2/share-transfers", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("name") != "named" {
			t.Errorf("query=%s", r.URL.RawQuery)
		}
		testcloud.JSON(w, 200, `{"transfers":[{"id":"t1","name":"named"}]}`)
	})
	api := sharetransfers.New(cloud.Client("sharev2", "/manila/v2"))
	if v, err := api.FindByNameOrID(context.Background(), r2ManilaID); err != nil || v.Name != "direct" {
		t.Fatal(v, err)
	}
	if v, err := api.FindByNameOrID(context.Background(), "named"); err != nil || v.ID != "t1" {
		t.Fatal(v, err)
	}
}

func TestR2ManilaHeatAccessActionsSkipVersionFloorAndLookups(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var posts atomic.Int32
	cloud.Mux.HandleFunc("/manila/v2/shares/"+r2ManilaID+"/action", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		body := r2ManilaHeatBody(t, r)
		if allow, ok := body["allow_access"]; ok {
			inner := r2ManilaHeatInner(t, allow)
			if len(body) != 1 || string(inner["access_type"]) != `"ip"` || string(inner["lock_deletion"]) != "false" || string(inner["metadata"]) != "null" || string(inner["lock_reason"]) != `""` || inner["access_level"] != nil {
				t.Errorf("allow=%v", inner)
			}
			testcloud.JSON(w, 200, `{"access":{"id":"acc","state":"queued_to_apply","extra":null}}`)
			return
		}
		inner := r2ManilaHeatInner(t, body["deny_access"])
		switch string(inner["access_id"]) {
		case `"acc"`:
			if len(inner) != 2 || string(inner["unrestrict"]) != "true" {
				t.Errorf("deny=%v", inner)
			}
			w.Header().Set("X-Request-Id", "deny")
			w.WriteHeader(202)
		case `"gone"`:
			testcloud.JSON(w, 404, `{}`)
		default:
			testcloud.JSON(w, 500, `{}`)
		}
	})
	api := shareaccessrules.New(cloud.Client("sharev2", "/manila/v2"))
	ctx := context.Background()
	value, err := api.AllowAccessAction(ctx, r2ManilaID, shareaccessrules.AllowActionOpts{
		AccessType: request.Present("ip"), LockDeletion: request.Present(false), Metadata: request.Null[json.RawMessage](), LockReason: request.Present(""),
	})
	if err != nil || string(value.Body["extra"]) != "null" || value.StatusCode != 200 {
		t.Fatal(value, err)
	}
	for _, option := range []shareaccessrules.AllowActionOption{shareaccessrules.WithAllowActionField("access_type", "x"), shareaccessrules.WithAllowActionField("access_level", "rw")} {
		if _, err := api.AllowAccessAction(ctx, r2ManilaID, shareaccessrules.AllowActionOpts{}, option); err == nil {
			t.Fatal("collision accepted")
		}
	}
	result, err := api.DenyAccessAction(ctx, r2ManilaID, "acc", shareaccessrules.WithDenyActionUnrestrict(true))
	if err != nil || result.StatusCode != 202 || result.Header.Get("X-Request-Id") != "deny" {
		t.Fatal(result, err)
	}
	if result, err := api.DenyAccessAction(ctx, r2ManilaID, "gone"); err != nil || result != nil {
		t.Fatal(result, err)
	}
	if _, err := api.DenyAccessAction(ctx, r2ManilaID, "gone", shareaccessrules.WithDenyActionIgnoreMissing(false)); err == nil {
		t.Fatal("missing ignored")
	}
	before := posts.Load()
	if _, err := api.DenyAccessAction(ctx, r2ManilaID, "boom"); err == nil || posts.Load() != before+1 {
		t.Fatalf("err=%v posts=%d", err, posts.Load())
	}
	for _, bad := range []string{"", "a/b"} {
		if _, err := api.DenyAccessAction(ctx, bad, "acc"); err == nil {
			t.Fatal("bad share accepted")
		}
		if _, err := api.DenyAccessAction(ctx, r2ManilaID, bad); err == nil {
			t.Fatal("bad access accepted")
		}
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	before = posts.Load()
	if _, err := api.AllowAccessAction(cancelled, r2ManilaID, shareaccessrules.AllowActionOpts{}); !errors.Is(err, context.Canceled) || posts.Load() != before {
		t.Fatal(err)
	}
}

func r2ManilaHeatInner(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var inner map[string]json.RawMessage
	if err := json.Unmarshal(raw, &inner); err != nil {
		t.Fatal(err)
	}
	return inner
}
