package api_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func pyCloudIronicVIFConnection(t *testing.T, cloud *testcloud.Cloud, rec *pyCloudIronicRecorder) *sdk.Connection {
	t.Helper()
	conn, err := sdk.FromProvider(cloud.Provider,
		sdk.WithEndpoint(sdk.BareMetal, cloud.Server.URL+"/baremetal/v1"),
		sdk.WithMicroversion(sdk.BareMetal, "1.80"),
		sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network/v2.0/"))
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("GET /baremetal/v1/nodes/{id}", func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		switch r.PathValue("id") {
		case "node-a", "node-uuid":
			testcloud.JSON(w, 200, `{"uuid":"node-uuid","name":"node-a"}`)
		default:
			testcloud.JSON(w, 404, `{}`)
		}
	})
	cloud.Mux.HandleFunc("GET /baremetal/v1/nodes/detail", func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		testcloud.JSON(w, 200, `{"nodes":[]}`)
	})
	cloud.Mux.HandleFunc("GET /network/v2.0/ports/{id}", func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		if r.PathValue("id") == "port-id" {
			testcloud.JSON(w, 200, `{"port":{"id":"port-id","name":"p"}}`)
			return
		}
		testcloud.JSON(w, 404, `{"NeutronError":{"message":"none"}}`)
	})
	cloud.Mux.HandleFunc("GET /network/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		if r.URL.Query().Get("name") == "named" {
			testcloud.JSON(w, 200, `{"ports":[{"id":"port-named","name":"named"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"ports":[]}`)
	})
	return conn
}

func TestPyCloudIronicAttachPortFindsMachineAndPortThenPostsOnceWithoutRetry(t *testing.T) {
	cloud := testcloud.New(t)
	rec := &pyCloudIronicRecorder{}
	conn := pyCloudIronicVIFConnection(t, cloud, rec)
	cloud.Mux.HandleFunc("POST /baremetal/v1/nodes/{id}/vifs", func(w http.ResponseWriter, r *http.Request) {
		body := rec.add(r)
		if r.PathValue("id") != "node-uuid" || body != `{"id":"port-named"}` && body != `{"id":"port-id"}` {
			t.Errorf("node=%s body=%s", r.PathValue("id"), body)
		}
		if body == `{"id":"port-id"}` {
			testcloud.JSON(w, 409, `{"error":"locked"}`)
			return
		}
		w.WriteHeader(204)
	})
	ctx := context.Background()
	if err := conn.AttachPortToMachine(ctx, "node-a", "named"); err != nil {
		t.Fatal(err)
	}
	pyCloudIronicRequireLines(t, rec,
		"GET /baremetal/v1/nodes/node-a",
		"GET /network/v2.0/ports/named",
		"GET /network/v2.0/ports?name=named",
		"POST /baremetal/v1/nodes/node-uuid/vifs")
	ironicRequireStatus(t, conn.AttachPortToMachine(ctx, "node-uuid", "port-id"), 409)
	if n := len(rec.snapshot()); n != 4+3 {
		t.Fatalf("a 409 attach must not be retried: %d requests", n)
	}
}

func TestPyCloudIronicAttachPortMissingMachineOrPortStopsBeforeMutation(t *testing.T) {
	cloud := testcloud.New(t)
	rec := &pyCloudIronicRecorder{}
	conn := pyCloudIronicVIFConnection(t, cloud, rec)
	cloud.Mux.HandleFunc("/baremetal/v1/nodes/{id}/vifs", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected VIF request %s %s", r.Method, r.URL)
		w.WriteHeader(500)
	})
	ctx := context.Background()
	if err := conn.AttachPortToMachine(ctx, "ghost", "port-id"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing machine err=%v", err)
	}
	if err := conn.AttachPortToMachine(ctx, "node-a", "no-such-port"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing port err=%v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	before := len(rec.snapshot())
	if err := conn.AttachPortToMachine(canceled, "node-a", "port-id"); !errors.Is(err, context.Canceled) || len(rec.snapshot()) != before {
		t.Fatalf("canceled err=%v", err)
	}
}

func TestPyCloudIronicDetachPortReportsAlreadyDetachedOn400AndKeepsOtherErrors(t *testing.T) {
	cloud := testcloud.New(t)
	rec := &pyCloudIronicRecorder{}
	conn := pyCloudIronicVIFConnection(t, cloud, rec)
	cloud.Mux.HandleFunc("DELETE /baremetal/v1/nodes/{id}/vifs/{vif}", func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		switch r.PathValue("vif") {
		case "port-id":
			w.WriteHeader(204)
		case "port-named":
			testcloud.JSON(w, 400, `{"error":"not attached"}`)
		}
	})
	ctx := context.Background()
	detached, err := conn.DetachPortFromMachine(ctx, "node-a", "port-id")
	if err != nil || !detached {
		t.Fatalf("detached=%v err=%v", detached, err)
	}
	detached, err = conn.DetachPortFromMachine(ctx, "node-a", "named")
	if err != nil || detached {
		t.Fatalf("400 must report not detached without error: %v %v", detached, err)
	}
	if got := rec.snapshot(); got[len(got)-1] != "DELETE /baremetal/v1/nodes/node-uuid/vifs/port-named" {
		t.Fatalf("requests=%q", got)
	}
	if _, err := conn.DetachPortFromMachine(ctx, "ghost", "port-id"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing machine err=%v", err)
	}
}

func TestPyCloudIronicListPortsAttachedFindsEveryVIFAndFailsWholeOnMissing(t *testing.T) {
	cloud := testcloud.New(t)
	rec := &pyCloudIronicRecorder{}
	conn := pyCloudIronicVIFConnection(t, cloud, rec)
	vifs := `{"vifs":[{"id":"port-id"},{"id":"named"}]}`
	cloud.Mux.HandleFunc("GET /baremetal/v1/nodes/{id}/vifs", func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		testcloud.JSON(w, 200, vifs)
	})
	ctx := context.Background()
	ports, err := conn.ListPortsAttachedToMachine(ctx, "node-a")
	if err != nil || len(ports) != 2 || ports[0].ID != "port-id" || ports[1].ID != "port-named" {
		t.Fatalf("ports=%v err=%v", ports, err)
	}
	pyCloudIronicRequireLines(t, rec,
		"GET /baremetal/v1/nodes/node-a",
		"GET /baremetal/v1/nodes/node-uuid/vifs",
		"GET /network/v2.0/ports/port-id",
		"GET /network/v2.0/ports/named",
		"GET /network/v2.0/ports?name=named")
	vifs = `{"vifs":[{"id":"port-id"},{"id":"vanished"}]}`
	ports, err = conn.ListPortsAttachedToMachine(ctx, "node-a")
	if !errors.Is(err, resource.ErrNotFound) || ports != nil {
		t.Fatalf("ports=%v err=%v", ports, err)
	}
	if !reflect.DeepEqual(len(rec.snapshot()) > 5, true) {
		t.Fatal("expected additional requests")
	}
}
