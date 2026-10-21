package api_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPyIronicBValidateRequiredChecksOnlyRequiredPresentInterfaces(t *testing.T) {
	cloud := testcloud.New(t)
	body := `{"boot":{"result":true},"deploy":{"result":false,"reason":"no image"},"power":{"result":null,"reason":null},"console":{"result":false,"reason":"off"}}`
	rec := ironicNodeServe(t, cloud, "GET /v1/nodes/n-1/validate", 200, body)
	api := newIronicNodes(t, cloud)
	ctx := context.Background()

	got, err := api.ValidateRequired(ctx, "n-1", nodes.DefaultValidationInterfaces())
	var verr *nodes.ValidationError
	if !errors.Is(err, nodes.ErrValidationFailed) || !errors.As(err, &verr) {
		t.Fatalf("err=%v", err)
	}
	want := []nodes.InterfaceFailure{{Interface: "deploy", Reason: "no image"}, {Interface: "power", Reason: "None"}}
	if !reflect.DeepEqual(verr.Failures, want) || !strings.Contains(verr.Error(), "deploy (no image), power (None)") {
		t.Fatalf("failures=%v msg=%s", verr.Failures, verr.Error())
	}
	if len(got) != 4 || got["power"].Result != nil || got["power"].Reason != nil || got["boot"].Result == nil || !*got["boot"].Result {
		t.Fatalf("results=%v", got)
	}

	// Nil required checks nothing; an absent required interface is not a failure.
	if _, err := api.ValidateRequired(ctx, "n-1", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := api.ValidateRequired(ctx, "n-1", []string{"boot", "storage", "console-missing"}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.ValidateRequired(ctx, "n-1", []string{"console"}); !errors.Is(err, nodes.ErrValidationFailed) {
		t.Fatalf("console: %v", err)
	}
	if rec.count() != 4 {
		t.Fatalf("calls=%d", rec.count())
	}
}

func TestPyIronicBValidateRequiredKeepsHTTPFailuresAndCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	ironicNodeServe(t, cloud, "GET /v1/nodes/gone/validate", 404, `{}`)
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := newIronicNodes(t, cloud).ValidateRequired(context.Background(), "gone", nil); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("err=%v", err)
	}
	blocked := testcloud.New(t)
	ironicNodeUnreachable(t, blocked)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := newIronicNodes(t, blocked).ValidateRequired(ctx, "n-1", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

func TestPyIronicBMaintenanceSendsNullOrExplicitReasonThenReadsNode(t *testing.T) {
	cloud := testcloud.New(t)
	put := ironicNodeServe(t, cloud, "PUT /v1/nodes/n-1/maintenance", 202, "")
	del := ironicNodeServe(t, cloud, "DELETE /v1/nodes/n-1/maintenance", 202, "")
	get := ironicNodeServe(t, cloud, "GET /v1/nodes/n-1", 200, `{"uuid":"n-1","maintenance":true,"maintenance_reason":"x"}`)
	api := newIronicNodes(t, cloud)
	ctx := context.Background()

	node, err := api.SetMaintenanceAndGet(ctx, "n-1", nil)
	if err != nil || node.UUID != "n-1" || !node.Maintenance {
		t.Fatalf("node=%v err=%v", node, err)
	}
	empty := ""
	if _, err := api.SetMaintenanceAndGet(ctx, "n-1", &empty); err != nil {
		t.Fatal(err)
	}
	reason := "rack work"
	if _, err := api.SetMaintenanceAndGet(ctx, "n-1", &reason); err != nil {
		t.Fatal(err)
	}
	calls := put.all()
	for i, want := range []string{`null`, `""`, `"rack work"`} {
		if got := ironicNodeFields(t, calls[i].Body); !reflect.DeepEqual(got, map[string]string{"reason": want}) {
			t.Fatalf("body %d=%v", i, got)
		}
	}
	if get.count() != 3 {
		t.Fatalf("reads=%d", get.count())
	}
	if _, err := api.UnsetMaintenanceAndGet(ctx, "n-1"); err != nil {
		t.Fatal(err)
	}
	if c := del.only(t); c.Body != "" || get.count() != 4 {
		t.Fatalf("body=%q reads=%d", c.Body, get.count())
	}
}

func TestPyIronicBMaintenanceFailedWriteSkipsReadAndFailedReadKeepsCause(t *testing.T) {
	cloud := testcloud.New(t)
	ironicNodeServe(t, cloud, "PUT /v1/nodes/bad/maintenance", 200, "")
	ironicNodeServe(t, cloud, "DELETE /v1/nodes/bad/maintenance", 409, `{}`)
	badGet := ironicNodeServe(t, cloud, "GET /v1/nodes/bad", 200, `{"uuid":"bad"}`)
	ironicNodeServe(t, cloud, "PUT /v1/nodes/n-1/maintenance", 202, "")
	missing := ironicNodeServe(t, cloud, "GET /v1/nodes/n-1", 404, `{}`)
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.SetMaintenanceAndGet(ctx, "bad", nil); !errors.As(err, &unexpected) || unexpected.Actual != 200 {
		t.Fatalf("only 202 is accepted: %v", err)
	}
	if _, err := api.UnsetMaintenanceAndGet(ctx, "bad"); !errors.As(err, &unexpected) || unexpected.Actual != 409 {
		t.Fatalf("err=%v", err)
	}
	if badGet.count() != 0 {
		t.Fatal("a failed write must not read the node")
	}
	if node, err := api.SetMaintenanceAndGet(ctx, "n-1", nil); node != nil || !errors.As(err, &unexpected) || unexpected.Actual != 404 || missing.count() != 1 {
		t.Fatalf("node=%v err=%v", node, err)
	}
}

func TestPyIronicBWaitForPowerStatePollsImmediatelyThenAtInterval(t *testing.T) {
	cloud := testcloud.New(t)
	var polls atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/nodes/n-1", func(w http.ResponseWriter, r *http.Request) {
		state := "power off"
		if polls.Add(1) >= 3 {
			state = "power on"
		}
		testcloud.JSON(w, 200, `{"uuid":"n-1","power_state":"`+state+`"}`)
	})
	node, err := newIronicNodes(t, cloud).WaitForPowerState(context.Background(), "n-1", "power on", nodes.WithWaitInterval(time.Millisecond))
	if err != nil || node.PowerState != "power on" || polls.Load() != 3 {
		t.Fatalf("node=%v err=%v polls=%d", node, err, polls.Load())
	}
}

func TestPyIronicBWaitForPowerStateStopsOnDeadlineAndReadErrors(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	stuck := ironicNodeServe(t, cloud, "GET /v1/nodes/stuck", 200, `{"uuid":"stuck","power_state":null}`)
	ironicNodeServe(t, cloud, "GET /v1/nodes/gone", 404, `{}`)
	api := newIronicNodes(t, cloud)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	if _, err := api.WaitForPowerState(ctx, "stuck", "power on", nodes.WithWaitInterval(5*time.Millisecond)); !errors.Is(err, context.DeadlineExceeded) || stuck.count() < 2 {
		t.Fatalf("err=%v polls=%d", err, stuck.count())
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.WaitForPowerState(context.Background(), "gone", "power on"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("err=%v", err)
	}
}

func TestPyIronicBWaitForReservationUsesSnapshotAndPollsUntilReleased(t *testing.T) {
	cloud := testcloud.New(t)
	var polls atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/nodes/n-1", func(w http.ResponseWriter, r *http.Request) {
		body := `{"uuid":"n-1","reservation":"conductor-1"}`
		if polls.Add(1) >= 2 {
			body = `{"uuid":"n-1","reservation":null}`
		}
		testcloud.JSON(w, 200, body)
	})
	api := newIronicNodes(t, cloud)
	ctx := context.Background()

	free := &nodes.Node{UUID: "n-1"}
	if got, err := api.WaitForReservation(ctx, free); err != nil || got != free || polls.Load() != 0 {
		t.Fatalf("an unreserved snapshot must return without HTTP: err=%v polls=%d", err, polls.Load())
	}
	got, err := api.WaitForReservation(ctx, &nodes.Node{UUID: "n-1", Reservation: "conductor-1"}, nodes.WithWaitInterval(time.Millisecond))
	if err != nil || got.Reservation != "" || polls.Load() != 2 {
		t.Fatalf("got=%v err=%v polls=%d", got, err, polls.Load())
	}
	if _, err := api.WaitForReservation(ctx, nil); err == nil {
		t.Fatal("nil snapshot must be rejected")
	}
}
