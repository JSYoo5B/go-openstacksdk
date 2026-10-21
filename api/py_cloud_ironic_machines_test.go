package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	baremetal "github.com/JSYoo5B/go-openstacksdk/baremetal/v1"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pyCloudIronicService(cloud *testcloud.Cloud) *baremetal.Service {
	return baremetal.New(ironicClient(cloud, "baremetal", "1.80"))
}

type pyCloudIronicRecorder struct {
	mu     sync.Mutex
	lines  []string
	paths  []string
	bodies []string
}

func (r *pyCloudIronicRecorder) add(req *http.Request) string {
	body, _ := io.ReadAll(req.Body)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lines = append(r.lines, req.Method+" "+req.URL.RequestURI())
	r.bodies = append(r.bodies, string(body))
	return string(body)
}

func (r *pyCloudIronicRecorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.lines...)
}

func pyCloudIronicRequireLines(t *testing.T, rec *pyCloudIronicRecorder, want ...string) {
	t.Helper()
	if got := rec.snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("requests=%q want %q", got, want)
	}
}

func TestPyCloudIronicListMachinesUsesBasicRouteAndFollowsPages(t *testing.T) {
	cloud := testcloud.New(t)
	rec := &pyCloudIronicRecorder{}
	cloud.Mux.HandleFunc("GET /v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		if got := r.Header.Get(ironicAPIVersionHeader); got != "1.80" {
			t.Errorf("microversion header=%q", got)
		}
		if r.URL.Query().Get("marker") == "n1" {
			testcloud.JSON(w, 200, `{"nodes":[{"uuid":"n2","name":"two"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"nodes":[{"uuid":"n1","name":null}],"next":"`+cloud.Server.URL+`/v1/nodes?marker=n1"}`)
	})
	rows, err := pyCloudIronicService(cloud).ListMachines(context.Background())
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows=%v err=%v", rows, err)
	}
	if string(rows[0].Resource.Body["id"]) != `"n1"` || string(rows[1].Resource.Body["name"]) != `"two"` {
		t.Fatalf("rows=%+v %+v", rows[0].Resource, rows[1].Resource)
	}
	pyCloudIronicRequireLines(t, rec, "GET /v1/nodes", "GET /v1/nodes?marker=n1")
}

func TestPyCloudIronicListMachinesPreservesHTTPFailureAndCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 503, `{"error":"busy"}`)
	})
	service := pyCloudIronicService(cloud)
	_, err := service.ListMachines(context.Background())
	ironicRequireStatus(t, err, 503)
	if calls.Load() != 1 {
		t.Fatalf("a failed list must not be replayed: %d", calls.Load())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.ListMachines(ctx); !errors.Is(err, context.Canceled) || calls.Load() != 1 {
		t.Fatalf("canceled list: err=%v calls=%d", err, calls.Load())
	}
}

func TestPyCloudIronicGetMachineFindsByDirectReadOrMissingNil(t *testing.T) {
	cloud := testcloud.New(t)
	rec := &pyCloudIronicRecorder{}
	cloud.Mux.HandleFunc("GET /v1/nodes/{id}", func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		if r.PathValue("id") == "found" {
			testcloud.JSON(w, 200, `{"uuid":"u1","name":"found","provision_state":"manageable","instance_uuid":null}`)
			return
		}
		testcloud.JSON(w, 404, `{"error":"gone"}`)
	})
	cloud.Mux.HandleFunc("GET /v1/nodes/detail", func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		testcloud.JSON(w, 200, `{"nodes":[]}`)
	})
	service := pyCloudIronicService(cloud)
	row, err := service.GetMachine(context.Background(), "found")
	if err != nil || row == nil || string(row.Resource.Body["id"]) != `"u1"` || string(row.Resource.Body["provision_state"]) != `"manageable"` {
		t.Fatalf("row=%+v err=%v", row, err)
	}
	pyCloudIronicRequireLines(t, rec, "GET /v1/nodes/found")
	missing, err := service.GetMachine(context.Background(), "ghost")
	if err != nil || missing != nil {
		t.Fatalf("missing node must be nil without error: row=%v err=%v", missing, err)
	}
	pyCloudIronicRequireLines(t, rec, "GET /v1/nodes/found", "GET /v1/nodes/ghost", "GET /v1/nodes/detail")
}

func TestPyCloudIronicMachineHelpersRejectEmptyIdentityBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	ironicForbidHTTP(t, cloud)
	service := pyCloudIronicService(cloud)
	ctx := context.Background()
	calls := map[string]error{}
	_, calls["get"] = service.GetMachine(ctx, "")
	_, calls["nics"] = service.ListNICsForMachine(ctx, "")
	_, calls["mac"] = service.GetNICByMAC(ctx, "")
	_, calls["bymac"] = service.GetMachineByMAC(ctx, "")
	calls["on"] = service.SetMachinePowerOn(ctx, "")
	calls["off"] = service.SetMachinePowerOff(ctx, "")
	calls["reboot"] = service.SetMachinePowerReboot(ctx, "")
	calls["maint"] = service.SetMachineMaintenance(ctx, "", nil)
	calls["unmaint"] = service.RemoveMachineFromMaintenance(ctx, "")
	_, calls["patch"] = service.PatchMachine(ctx, "", nodes.UpdateOpts{})
	_, calls["purge"] = service.PurgeMachineInstanceInfo(ctx, "")
	calls["validate"] = service.ValidateMachine(ctx, "", true)
	_, calls["wait"] = service.WaitForMachineReservation(ctx, "")
	for name, err := range calls {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: want invalid option, got %v", name, err)
		}
	}
}

func TestPyCloudIronicNICListsUseDetailRouteNodeUUIDAndAddressQueries(t *testing.T) {
	cloud := testcloud.New(t)
	rec := &pyCloudIronicRecorder{}
	cloud.Mux.HandleFunc("GET /v1/ports/detail", func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		q := r.URL.Query()
		switch {
		case q.Get("address") == "aa:bb" && q.Get("marker") == "p1":
			testcloud.JSON(w, 200, `{"ports":[{"uuid":"p2","address":"aa:bb","node_uuid":"other"}]}`)
		case q.Get("address") == "aa:bb":
			testcloud.JSON(w, 200, `{"ports":[{"uuid":"p1","address":"aa:bb","node_uuid":"node-1"}],"ports_links":[{"rel":"next","href":"`+cloud.Server.URL+`/v1/ports/detail?address=aa%3Abb&marker=p1"}]}`)
		case q.Get("address") == "none":
			testcloud.JSON(w, 200, `{"ports":[]}`)
		default:
			testcloud.JSON(w, 200, `{"ports":[{"uuid":"p1","address":"aa:bb","node_uuid":"node-1"}]}`)
		}
	})
	service := pyCloudIronicService(cloud)
	ctx := context.Background()
	all, err := service.ListNICs(ctx)
	if err != nil || len(all) != 1 || all[0].UUID != "p1" {
		t.Fatalf("all=%v err=%v", all, err)
	}
	forNode, err := service.ListNICsForMachine(ctx, "node-1")
	if err != nil || len(forNode) != 1 {
		t.Fatalf("forNode=%v err=%v", forNode, err)
	}
	first, err := service.GetNICByMAC(ctx, "aa:bb")
	if err != nil || first == nil || first.UUID != "p1" {
		t.Fatalf("first=%v err=%v", first, err)
	}
	none, err := service.GetNICByMAC(ctx, "none")
	if err != nil || none != nil {
		t.Fatalf("none=%v err=%v", none, err)
	}
	pyCloudIronicRequireLines(t, rec,
		"GET /v1/ports/detail",
		"GET /v1/ports/detail?node_uuid=node-1",
		"GET /v1/ports/detail?address=aa%3Abb",
		"GET /v1/ports/detail?address=aa%3Abb&marker=p1",
		"GET /v1/ports/detail?address=none")
}

func TestPyCloudIronicGetMachineByMACResolvesNodeThroughPortNodeUUID(t *testing.T) {
	cloud := testcloud.New(t)
	rec := &pyCloudIronicRecorder{}
	cloud.Mux.HandleFunc("GET /v1/ports/detail", func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		if r.URL.Query().Get("address") == "aa:bb" {
			testcloud.JSON(w, 200, `{"ports":[{"uuid":"p1","address":"aa:bb","node_uuid":"node-9"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"ports":[]}`)
	})
	cloud.Mux.HandleFunc("GET /v1/nodes/{id}", func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		testcloud.JSON(w, 200, `{"uuid":"node-9","name":"nine"}`)
	})
	service := pyCloudIronicService(cloud)
	row, err := service.GetMachineByMAC(context.Background(), "aa:bb")
	if err != nil || row == nil || string(row.Resource.Body["id"]) != `"node-9"` {
		t.Fatalf("row=%+v err=%v", row, err)
	}
	pyCloudIronicRequireLines(t, rec, "GET /v1/ports/detail?address=aa%3Abb", "GET /v1/nodes/node-9")
	missing, err := service.GetMachineByMAC(context.Background(), "zz:zz")
	if err != nil || missing != nil {
		t.Fatalf("unknown MAC must yield nil: %v %v", missing, err)
	}
	if got := len(rec.snapshot()); got != 3 {
		t.Fatalf("unknown MAC must not read a node: %d requests", got)
	}
}

func TestPyCloudIronicPowerHelpersSendTargetAndNeverReplay(t *testing.T) {
	cloud := testcloud.New(t)
	rec := &pyCloudIronicRecorder{}
	cloud.Mux.HandleFunc("PUT /v1/nodes/{id}/states/power", func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		switch r.PathValue("id") {
		case "locked":
			testcloud.JSON(w, 409, `{"error":"locked"}`)
		case "wrong-code":
			testcloud.JSON(w, 200, `{}`)
		default:
			w.WriteHeader(202)
		}
	})
	service := pyCloudIronicService(cloud)
	ctx := context.Background()
	for _, call := range []struct {
		run  func(string) error
		want string
	}{
		{func(id string) error { return service.SetMachinePowerOn(ctx, id) }, `{"target":"power on"}`},
		{func(id string) error { return service.SetMachinePowerOff(ctx, id) }, `{"target":"power off"}`},
		{func(id string) error { return service.SetMachinePowerReboot(ctx, id) }, `{"target":"rebooting"}`},
	} {
		if err := call.run("node-a"); err != nil {
			t.Fatalf("%s: %v", call.want, err)
		}
		rec.mu.Lock()
		got := rec.bodies[len(rec.bodies)-1]
		rec.mu.Unlock()
		if got != call.want {
			t.Fatalf("body=%s want %s", got, call.want)
		}
	}
	ironicRequireStatus(t, service.SetMachinePowerOn(ctx, "locked"), 409)
	ironicRequireStatus(t, service.SetMachinePowerOff(ctx, "wrong-code"), 200)
	if n := len(rec.snapshot()); n != 5 {
		t.Fatalf("expected one request per call, got %d", n)
	}
}

func TestPyCloudIronicMaintenanceHelpersOmitNilAndEmptyReasonWithoutFollowUpRead(t *testing.T) {
	cloud := testcloud.New(t)
	rec := &pyCloudIronicRecorder{}
	cloud.Mux.HandleFunc("/v1/nodes/{id}/maintenance", func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		w.WriteHeader(202)
	})
	cloud.Mux.HandleFunc("GET /v1/nodes/{id}", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("the SDK helper must not add a follow-up read: %s", r.URL)
		w.WriteHeader(500)
	})
	service := pyCloudIronicService(cloud)
	ctx := context.Background()
	empty, reason := "", "disk swap"
	for _, r := range []*string{nil, &empty, &reason} {
		if err := service.SetMachineMaintenance(ctx, "node-a", r); err != nil {
			t.Fatal(err)
		}
	}
	if err := service.RemoveMachineFromMaintenance(ctx, "node-a"); err != nil {
		t.Fatal(err)
	}
	pyCloudIronicRequireLines(t, rec, "PUT /v1/nodes/node-a/maintenance", "PUT /v1/nodes/node-a/maintenance", "PUT /v1/nodes/node-a/maintenance", "DELETE /v1/nodes/node-a/maintenance")
	rec.mu.Lock()
	bodies := append([]string(nil), rec.bodies...)
	rec.mu.Unlock()
	if !reflect.DeepEqual(bodies, []string{`{}`, `{}`, `{"reason":"disk swap"}`, ""}) {
		t.Fatalf("bodies=%q", bodies)
	}
}

func TestPyCloudIronicPatchAndPurgeSendJSONPatchWithoutConflictRetry(t *testing.T) {
	cloud := testcloud.New(t)
	rec := &pyCloudIronicRecorder{}
	cloud.Mux.HandleFunc("PATCH /v1/nodes/{id}", func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		if r.PathValue("id") == "busy" {
			testcloud.JSON(w, 409, `{"error":"locked"}`)
			return
		}
		testcloud.JSON(w, 200, `{"uuid":"u1","name":"renamed","instance_info":{}}`)
	})
	service := pyCloudIronicService(cloud)
	ctx := context.Background()
	node, err := service.PatchMachine(ctx, "node-a", nodes.UpdateOpts{
		nodes.UpdateOperation{Op: nodes.ReplaceOp, Path: "/name", Value: "renamed"},
	})
	if err != nil || node.Name != "renamed" {
		t.Fatalf("node=%+v err=%v", node, err)
	}
	if _, err := service.PurgeMachineInstanceInfo(ctx, "node-a"); err != nil {
		t.Fatal(err)
	}
	rec.mu.Lock()
	bodies := append([]string(nil), rec.bodies...)
	rec.mu.Unlock()
	if !reflect.DeepEqual(bodies, []string{`[{"op":"replace","path":"/name","value":"renamed"}]`, `[{"op":"remove","path":"/instance_info"}]`}) {
		t.Fatalf("bodies=%q", bodies)
	}
	_, err = service.PatchMachine(ctx, "busy", nodes.UpdateOpts{nodes.UpdateOperation{Op: nodes.RemoveOp, Path: "/extra"}})
	ironicRequireStatus(t, err, 409)
	if n := len(rec.snapshot()); n != 3 {
		t.Fatalf("a 409 must not be retried automatically: %d requests", n)
	}
}

func TestPyCloudIronicValidateMachineChecksOnlyRequiredPresentInterfaces(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/nodes/{id}/validate", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.PathValue("id") {
		case "ok":
			testcloud.JSON(w, 200, `{"boot":{"result":true},"deploy":{"result":true},"management":{"result":true},"power":{"result":true},"console":{"result":false,"reason":"unused"}}`)
		case "missing-iface":
			testcloud.JSON(w, 200, `{"power":{"result":true},"deploy":{"result":true}}`)
		case "bad":
			testcloud.JSON(w, 200, `{"boot":{"result":false,"reason":"no image"},"deploy":{"result":null},"management":{"result":true},"power":{"result":true}}`)
		case "power-bad":
			testcloud.JSON(w, 200, `{"boot":{"result":false},"power":{"result":false,"reason":"ipmi"}}`)
		default:
			testcloud.JSON(w, 404, `{}`)
		}
	})
	service := pyCloudIronicService(cloud)
	ctx := context.Background()
	if err := service.ValidateMachine(ctx, "ok", true); err != nil {
		t.Fatal(err)
	}
	if err := service.ValidateMachine(ctx, "missing-iface", true); err != nil {
		t.Fatalf("interfaces absent from the response are ignored: %v", err)
	}
	var vErr *baremetal.MachineValidationError
	err := service.ValidateMachine(ctx, "bad", true)
	if !errors.As(err, &vErr) || !reflect.DeepEqual(vErr.Failures, []string{"boot (no image)", "deploy (no reason)"}) || vErr.Machine != "bad" {
		t.Fatalf("err=%v", err)
	}
	if err := service.ValidateMachine(ctx, "bad", false); err != nil {
		t.Fatalf("power-only validation ignores other interfaces: %v", err)
	}
	err = service.ValidateMachine(ctx, "power-bad", false)
	if !errors.As(err, &vErr) || !reflect.DeepEqual(vErr.Failures, []string{"power (ipmi)"}) {
		t.Fatalf("err=%v", err)
	}
	if err := service.ValidateMachineInterfaces(ctx, "bad"); err != nil {
		t.Fatalf("empty required list checks only HTTP: %v", err)
	}
	ironicRequireStatus(t, service.ValidateMachine(ctx, "gone", true), 404)
	if calls.Load() != 7 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestPyCloudIronicWaitForReservationPollsUntilUnlockedAndHonorsSharedOptions(t *testing.T) {
	cloud := testcloud.New(t)
	var reads atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/nodes/{id}", func(w http.ResponseWriter, r *http.Request) {
		n := reads.Add(1)
		switch r.PathValue("id") {
		case "stuck":
			testcloud.JSON(w, 200, `{"uuid":"stuck","reservation":"conductor-1"}`)
		default:
			if n < 3 {
				testcloud.JSON(w, 200, `{"uuid":"n","reservation":"conductor-1"}`)
				return
			}
			testcloud.JSON(w, 200, `{"uuid":"n","reservation":null}`)
		}
	})
	service := pyCloudIronicService(cloud)
	ctx := context.Background()
	node, err := service.WaitForMachineReservation(ctx, "n", resource.WithPollInterval(time.Millisecond))
	if err != nil || node.Reservation != "" || reads.Load() != 3 {
		t.Fatalf("node=%+v reads=%d err=%v", node, reads.Load(), err)
	}
	_, err = service.WaitForMachineReservation(ctx, "stuck", resource.WithPollInterval(time.Millisecond), resource.WithTimeout(30*time.Millisecond))
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout err=%v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	before := reads.Load()
	if _, err := service.WaitForMachineReservation(canceled, "n"); !errors.Is(err, context.Canceled) || reads.Load() != before {
		t.Fatalf("canceled wait: err=%v reads=%d->%d", err, before, reads.Load())
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	_ = unexpected
}

func TestPyCloudIronicValidateMachineInterfacesUsesCallerListLikeDeprecatedValidateNode(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v1/nodes/{id}/validate", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"boot":{"result":false,"reason":"no image"},"deploy":{"result":true},"management":{"result":false,"reason":"ignored"},"power":{"result":false}}`)
	})
	service := pyCloudIronicService(cloud)
	var vErr *baremetal.MachineValidationError
	err := service.ValidateMachineInterfaces(context.Background(), "n", "boot", "deploy", "power")
	if !errors.As(err, &vErr) || !reflect.DeepEqual(vErr.Failures, []string{"boot (no image)", "power (no reason)"}) {
		t.Fatalf("err=%v", err)
	}
}
