package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/instanceha/v1/hosts"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const hostUUID = "33333333-3333-4333-8333-333333333333"

func TestInstanceHAHostFixedSegmentCRUDAndExactName(t *testing.T) {
	cloud := testcloud.New(t)
	var parents, lists, creates, updates, deletes atomic.Int32
	client := cloud.Client("instance-ha", "/instance-ha/v1/catalog-tenant")
	client.Microversion = "1.2"
	cloud.Mux.HandleFunc("/instance-ha/v1/catalog-tenant/segments", func(w http.ResponseWriter, r *http.Request) {
		parents.Add(1)
		testcloud.JSON(w, 200, `{"segments":[{"uuid":"`+segmentUUID+`","id":123,"name":"compute-a"}]}`)
	})
	response := `{"host":{"uuid":"` + hostUUID + `","id":9007199254740993,"name":"compute.01","type":"COMPUTE","control_attributes":"SSH","reserved":false,"on_maintenance":false,"failover_segment_id":"` + segmentUUID + `","failover_segment":{"uuid":"` + segmentUUID + `","id":123,"name":"compute-a","description":null},"updated_at":null}}`
	cloud.Mux.HandleFunc("/instance-ha/v1/catalog-tenant/segments/"+segmentUUID+"/hosts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			lists.Add(1)
			if r.URL.Query().Has("name") {
				t.Error("unsupported name query")
			}
			testcloud.JSON(w, 200, `{"hosts":[{"uuid":"`+hostUUID+`","id":1,"name":"compute.01"},{"uuid":"`+otherSegmentUUID+`","id":2,"name":"computeX01"}]}`)
			return
		}
		creates.Add(1)
		if r.Method != "POST" {
			t.Error(r.Method)
		}
		var body map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if string(body["host"]["control_attributes"]) != `"SSH"` || string(body["host"]["reserved"]) != "false" || body["host"]["segment_id"] != nil {
			t.Errorf("body=%v", body)
		}
		w.Header().Set("X-Request-ID", "host-create")
		testcloud.JSON(w, 201, response)
	})
	cloud.Mux.HandleFunc("/instance-ha/v1/catalog-tenant/segments/"+segmentUUID+"/hosts/"+hostUUID, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			testcloud.JSON(w, 200, response)
		case "PUT":
			updates.Add(1)
			var body map[string]map[string]json.RawMessage
			_ = json.NewDecoder(r.Body).Decode(&body)
			if string(body["host"]["on_maintenance"]) != "false" || len(body["host"]) != 1 {
				t.Errorf("update=%v", body)
			}
			testcloud.JSON(w, 200, response)
		case "DELETE":
			deletes.Add(1)
			w.WriteHeader(204)
		default:
			t.Error(r.Method)
		}
	})
	scope, err := hosts.New(client).InSegment(context.Background(), resource.Name("compute-a"))
	if err != nil {
		t.Fatal(err)
	}
	if scope.SegmentID() != segmentUUID || parents.Load() != 1 {
		t.Fatal("parent was not resolved by UUID")
	}
	reserved := false
	opt := hosts.WithCreateOptions(hosts.CreateOpts{Name: "compute.01", Type: "COMPUTE", ControlAttributes: "SSH", Reserved: &reserved})
	reserved = true
	host, err := scope.Create(context.Background(), hosts.CreateOpts{}, opt)
	if err != nil {
		t.Fatal(err)
	}
	if host.UUID != hostUUID || host.SegmentID != segmentUUID || string(host.ID) != "9007199254740993" || host.StatusCode != 201 || host.Header.Get("X-Request-ID") != "host-create" || host.FailoverSegment == nil || host.FailoverSegment.UUID != segmentUUID {
		t.Fatalf("host=%+v", host)
	}
	if value, err := scope.Get(context.Background(), hostUUID); err != nil || value.SegmentID != segmentUUID {
		t.Fatalf("value=%+v err=%v", value, err)
	}
	maintenance := false
	if _, err := scope.Update(context.Background(), resource.Name("compute.01"), hosts.UpdateOpts{OnMaintenance: &maintenance}); err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(context.Background(), resource.ID(hostUUID)); err != nil {
		t.Fatal(err)
	}
	if parents.Load() != 1 || lists.Load() != 1 || creates.Load() != 1 || updates.Load() != 1 || deletes.Load() != 1 {
		t.Fatalf("parents=%d lists=%d create=%d update=%d delete=%d", parents.Load(), lists.Load(), creates.Load(), updates.Load(), deletes.Load())
	}
}

func TestInstanceHAHostScopeRejectsParentOverridesBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("invalid input made HTTP %s", r.URL) })
	client := cloud.Client("instance-ha", "/v1")
	scope, err := hosts.New(client).InSegment(context.Background(), resource.ID(segmentUUID))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := hosts.New(client).InSegment(context.Background(), resource.ID("123")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	other := otherSegmentUUID
	if _, err := scope.All(context.Background(), hosts.WithListOptions(hosts.ListOpts{FailoverSegmentID: &other})); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, key := range []string{"failover_segment_id", "segment_id", "segment"} {
		if _, err := scope.Resources.All(context.Background(), resource.WithQuery(key, otherSegmentUUID)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: %v", key, err)
		}
	}
	for _, key := range []string{"failover_segment_id", "segment_id", "uuid", "id", "reserved"} {
		if _, err := scope.Create(context.Background(), hosts.CreateOpts{Name: "h", Type: "COMPUTE", ControlAttributes: "SSH"}, hosts.WithCreateField(key, "override")); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: %v", key, err)
		}
	}
	if _, err := scope.Update(context.Background(), resource.Name("looked-up"), hosts.UpdateOpts{}, hosts.WithUpdateField("segment_id", segmentUUID)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestInstanceHAHostTypedListFiltersAndRawControlAttributes(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/v1/segments/"+segmentUUID+"/hosts", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("reserved") != "false" || q.Get("on_maintenance") != "false" || q.Get("failover_segment_id") != segmentUUID || q.Get("type") != "COMPUTE" {
			t.Errorf("query=%v", q)
		}
		testcloud.JSON(w, 200, `{"hosts":[{"uuid":"`+hostUUID+`","id":"9007199254740993","name":"h","control_attributes":{"mcastport":5405,"large":9007199254740995},"reserved":null,"on_maintenance":false}]}`)
	})
	scope, err := hosts.New(cloud.Client("instance-ha", "/v1")).InSegment(context.Background(), resource.ID(segmentUUID))
	if err != nil {
		t.Fatal(err)
	}
	fixed, kind, disabled := segmentUUID, "COMPUTE", false
	options := []hosts.ListOption{hosts.WithListOptions(hosts.ListOpts{Reserved: &disabled, OnMaintenance: &disabled, FailoverSegmentID: &fixed, Type: &kind})}
	iterator := scope.List(context.Background(), options...)
	options[0] = hosts.WithListOptions(hosts.ListOpts{})
	var values []*hosts.Host
	for value, err := range iterator {
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, value)
	}
	if len(values) != 1 || values[0].SegmentID != segmentUUID || string(values[0].ID) != `"9007199254740993"` || values[0].Reserved != nil || string(values[0].Body["reserved"]) != "null" || string(values[0].ControlAttributes) != `{"mcastport":5405,"large":9007199254740995}` {
		t.Fatalf("values=%+v", values)
	}
}

func TestInstanceHAHostChangedSourceVersionRejectsAllScopeRequests(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("conflict made HTTP %s", r.URL) })
	client := cloud.Client("instance-ha", "/v1")
	client.Microversion = "1.2"
	scope, err := hosts.New(client).InSegment(context.Background(), resource.ID(segmentUUID))
	if err != nil {
		t.Fatal(err)
	}
	client.MoreHeaders = map[string]string{"openstack-api-version": "instance-ha 1.0"}
	operations := []func() error{
		func() error { _, e := scope.Get(context.Background(), hostUUID); return e },
		func() error { _, e := scope.All(context.Background()); return e },
		func() error {
			_, e := scope.Create(context.Background(), hosts.CreateOpts{Name: "h", Type: "COMPUTE", ControlAttributes: "SSH"})
			return e
		},
		func() error {
			_, e := scope.Update(context.Background(), resource.Name("h"), hosts.UpdateOpts{})
			return e
		},
		func() error { return scope.Delete(context.Background(), resource.ID(hostUUID)) },
		func() error {
			_, e := hosts.New(client).InSegment(context.Background(), resource.Name("segment"))
			return e
		},
	}
	for _, run := range operations {
		if err := run(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
}

func TestInstanceHAHostDeletionWaitPreserves404AndOtherFailures(t *testing.T) {
	cloud := testcloud.New(t)
	var polls atomic.Int32
	cloud.Mux.HandleFunc("/v1/segments/"+segmentUUID+"/hosts/"+hostUUID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && polls.Add(1) == 1 {
			testcloud.JSON(w, 200, `{"host":{"uuid":"`+hostUUID+`","name":"h"}}`)
			return
		}
		testcloud.JSON(w, 404, `{"error":"missing"}`)
	})
	cloud.Mux.HandleFunc("/v1/segments/"+segmentUUID+"/hosts/"+otherSegmentUUID, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 409, `{"error":"in use"}`) })
	scope, err := hosts.New(cloud.Client("instance-ha", "/v1")).InSegment(context.Background(), resource.ID(segmentUUID))
	if err != nil {
		t.Fatal(err)
	}
	if err := scope.WaitDeleted(context.Background(), resource.ID(hostUUID), resource.WithPollInterval(time.Millisecond)); err != nil || polls.Load() != 2 {
		t.Fatalf("polls=%d err=%v", polls.Load(), err)
	}
	if err := scope.Delete(context.Background(), resource.ID(hostUUID)); err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(context.Background(), resource.ID(hostUUID), resource.WithMissingError()); !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatal(err)
	}
	if err := scope.Delete(context.Background(), resource.ID(otherSegmentUUID)); !gophercloud.ResponseCodeIs(err, 409) {
		t.Fatal(err)
	}
}
