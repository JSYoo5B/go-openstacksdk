package api_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

func TestPyIronicAConsoleReadKeepsInfoAndRequires200(t *testing.T) {
	cloud := testcloud.New(t)
	get := ironicNodeServe(t, cloud, "GET /v1/nodes/n-1/states/console", 200, `{"console_enabled":false,"console_info":{"type":"shellinabox","url":null},"extra":1}`)
	api := newIronicNodes(t, cloud)
	state, err := api.GetConsole(context.Background(), "n-1")
	if err != nil || state.Enabled || string(state.Info) != `{"type":"shellinabox","url":null}` {
		t.Fatalf("state=%+v error=%v", state, err)
	}
	if string(state.Body) != `{"console_enabled":false,"console_info":{"type":"shellinabox","url":null},"extra":1}` {
		t.Fatalf("body=%s", state.Body)
	}
	if call := get.only(t); call.Body != "" || len(call.Query) != 0 {
		t.Fatalf("call=%+v", call)
	}
	ironicNodeRunStatusCases(t, []ironicNodeStatusCase{{
		name: "GetConsole", pattern: "GET /v1/nodes/n-1/states/console", ok: []int{200}, bad: []int{202, 204}, body: `{"console_enabled":true,"console_info":null}`,
		call: func(ctx context.Context, api *nodes.API) error { _, err := api.GetConsole(ctx, "n-1"); return err },
	}})
}

func TestPyIronicAConsoleModeSendsExplicitBooleanAndAcceptsSuccessCodes(t *testing.T) {
	cloud := testcloud.New(t)
	put := ironicNodeServe(t, cloud, "PUT /v1/nodes/n-1/states/console", 202, "")
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	for _, enabled := range []bool{true, false} {
		if err := api.SetConsoleMode(ctx, "n-1", enabled); err != nil {
			t.Fatal(err)
		}
	}
	calls := put.all()
	if len(calls) != 2 || calls[0].Body != `{"enabled":true}` || calls[1].Body != `{"enabled":false}` {
		t.Fatalf("calls=%+v", calls)
	}
	ironicNodeRunStatusCases(t, []ironicNodeStatusCase{{
		name: "SetConsoleMode", pattern: "PUT /v1/nodes/n-1/states/console", ok: []int{200, 202, 204}, bad: []int{201},
		call: func(ctx context.Context, api *nodes.API) error { return api.SetConsoleMode(ctx, "n-1", true) },
	}})
}

func TestPyIronicATraitAddSendsBodylessPutAndEscapesTrait(t *testing.T) {
	cloud := testcloud.New(t)
	put := ironicNodeServe(t, cloud, "PUT /v1/nodes/{id}/traits/{trait}", 204, "")
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	if err := api.AddTrait(ctx, "n-1", "CUSTOM_GPU"); err != nil {
		t.Fatal(err)
	}
	if err := api.AddTrait(ctx, "n 2", "CUSTOM/A B"); err != nil {
		t.Fatal(err)
	}
	calls := put.all()
	if len(calls) != 2 || calls[0].Path != "/v1/nodes/n-1/traits/CUSTOM_GPU" || calls[0].Body != "" || calls[1].Path != "/v1/nodes/n 2/traits/CUSTOM/A B" {
		t.Fatalf("calls=%+v", calls)
	}
	ironicNodeRunStatusCases(t, []ironicNodeStatusCase{{
		name: "AddTrait", pattern: "PUT /v1/nodes/n-1/traits/CUSTOM_GPU", ok: []int{204}, bad: []int{200, 202},
		call: func(ctx context.Context, api *nodes.API) error { return api.AddTrait(ctx, "n-1", "CUSTOM_GPU") },
	}})
}

func TestPyIronicAConsoleAndTraitRejectEmptyInputAndHonorCancellation(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	if _, err := api.GetConsole(ctx, ""); err == nil {
		t.Error("empty node must fail before HTTP")
	}
	if err := api.SetConsoleMode(ctx, "", true); err == nil {
		t.Error("empty node must fail before HTTP")
	}
	if err := api.AddTrait(ctx, "", "T"); err == nil {
		t.Error("empty node must fail before HTTP")
	}
	if err := api.AddTrait(ctx, "n-1", ""); err == nil {
		t.Error("empty trait must fail before HTTP")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	_, errGet := api.GetConsole(canceled, "n-1")
	errSet := api.SetConsoleMode(canceled, "n-1", false)
	errTrait := api.AddTrait(canceled, "n-1", "T")
	for i, err := range []error{errGet, errSet, errTrait} {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("pre-canceled call %d: %v", i, err)
		}
	}
	deadline, stop := context.WithTimeout(ctx, 30*time.Millisecond)
	defer stop()
	if err := api.AddTrait(deadline, "n-1", "T"); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("deadline: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("only the deadline call may reach HTTP, got %d", got)
	}
}

func TestPyIronicAVendorPassthruForwardsCallerVerbWithoutReplay(t *testing.T) {
	cloud := testcloud.New(t)
	rec := ironicNodeServe(t, cloud, "/v1/nodes/n-1/vendor_passthru", 503, `{"error_message":"busy"}`)
	api := newIronicNodes(t, cloud)
	_, err := api.CallVendorPassthru(context.Background(), "n-1", http.MethodPost, nodes.VendorPassthruCallOpts{Method: "reset"})
	ironicRequireStatus(t, err, 503)
	_, err = api.CallVendorPassthru(context.Background(), "n-1", "PATCH", nodes.VendorPassthruCallOpts{Method: "reset"})
	ironicRequireStatus(t, err, 503)
	var methods []string
	for _, call := range rec.all() {
		methods = append(methods, call.Method)
	}
	if !reflect.DeepEqual(methods, []string{"POST", "PATCH"}) {
		t.Fatalf("a 503 must not be retried and the verb is forwarded as given: %v", methods)
	}
}
