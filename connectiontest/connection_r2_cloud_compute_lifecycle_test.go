package connectiontest_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func r2CloudComputeMethods(calls []r2CloudComputeCall, method string) []r2CloudComputeCall {
	var out []r2CloudComputeCall
	for _, call := range calls {
		if call.Method == method {
			out = append(out, call)
		}
	}
	return out
}

func TestR2CloudComputeDeleteServerMissingServerSendsNoDelete(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/absent", 404, `{"itemNotFound":{"message":"nf","code":404}}`)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/detail", 200, `{"servers":[]}`)
	deleted, err := cloud.Conn.DeleteServer(context.Background(), "absent")
	if err != nil || deleted || len(r2CloudComputeMethods(cloud.Calls(), http.MethodDelete)) != 0 {
		t.Fatal(deleted, err, cloud.Calls())
	}
}

func TestR2CloudComputeDeleteServerFoundSendsOneDeleteAndMaps404ToFalse(t *testing.T) {
	for _, status := range []int{204, 404} {
		cloud := r2CloudComputeNew(t)
		cloud.JSON("GET "+r2CloudComputeBase+"/servers/web-1", 404, `{"itemNotFound":{"message":"nf","code":404}}`)
		cloud.JSON("GET "+r2CloudComputeBase+"/servers/detail", 200, `{"servers":[{"id":"s1","name":"web-1","status":"ACTIVE","addresses":{}}]}`)
		cloud.Handle("DELETE "+r2CloudComputeBase+"/servers/s1", func(w http.ResponseWriter, _ *http.Request, _ string) {
			w.WriteHeader(status)
		})
		deleted, err := cloud.Conn.DeleteServer(context.Background(), "web-1")
		if err != nil || deleted != (status == 204) || len(r2CloudComputeMethods(cloud.Calls(), http.MethodDelete)) != 1 {
			t.Fatal(status, deleted, err, cloud.Calls())
		}
	}
}

func TestR2CloudComputeDeleteServerDeleteErrorIsNotReplayed(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s1", 200, `{"server":{"id":"s1","name":"web-1","addresses":{}}}`)
	cloud.JSON("DELETE "+r2CloudComputeBase+"/servers/s1", 500, `{"computeFault":{"message":"x","code":500}}`)
	deleted, err := cloud.Conn.DeleteServer(context.Background(), "s1", sdk.WithCloudServerWait(true))
	if err == nil || deleted || len(r2CloudComputeMethods(cloud.Calls(), http.MethodDelete)) != 1 {
		t.Fatal(deleted, err, cloud.Calls())
	}
}

func TestR2CloudComputeDeleteServerWaitObservesAbsence(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	var gets atomic.Int32
	var deleted atomic.Bool
	cloud.Handle("GET "+r2CloudComputeBase+"/servers/s1", func(w http.ResponseWriter, _ *http.Request, _ string) {
		if deleted.Load() && gets.Add(1) > 1 {
			testcloud.JSON(w, 404, `{"itemNotFound":{"message":"nf","code":404}}`)
			return
		}
		testcloud.JSON(w, 200, `{"server":{"id":"s1","name":"web-1","status":"ACTIVE","addresses":{}}}`)
	})
	cloud.Handle("DELETE "+r2CloudComputeBase+"/servers/s1", func(w http.ResponseWriter, _ *http.Request, _ string) {
		deleted.Store(true)
		w.WriteHeader(204)
	})
	ok, err := cloud.Conn.DeleteServer(context.Background(), "s1", sdk.WithCloudServerWait(true), sdk.WithCloudServerWaitOptions(resource.WithPollInterval(time.Millisecond)))
	if err != nil || !ok || gets.Load() < 2 || len(r2CloudComputeMethods(cloud.Calls(), http.MethodDelete)) != 1 {
		t.Fatal(ok, err, gets.Load())
	}
}

func TestR2CloudComputeDeleteServerWaitTimeoutKeepsAcceptedDelete(t *testing.T) {
	t.Parallel()
	cloud := r2CloudComputeNew(t)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s1", 200, `{"server":{"id":"s1","name":"web-1","status":"ACTIVE","addresses":{}}}`)
	cloud.JSON("DELETE "+r2CloudComputeBase+"/servers/s1", 204, ``)
	ok, err := cloud.Conn.DeleteServer(context.Background(), "s1", sdk.WithCloudServerWait(true), sdk.WithCloudServerTimeout(30*time.Millisecond), sdk.WithCloudServerWaitOptions(resource.WithPollInterval(5*time.Millisecond)))
	if !ok || !errors.Is(err, context.DeadlineExceeded) || len(r2CloudComputeMethods(cloud.Calls(), http.MethodDelete)) != 1 {
		t.Fatal(ok, err)
	}
}

func TestR2CloudComputeDeleteServerRejectsNonPositiveTimeoutBeforeHTTP(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	_, err := cloud.Conn.DeleteServer(context.Background(), "s1", sdk.WithCloudServerWait(true), sdk.WithCloudServerTimeout(0))
	if !errors.Is(err, resource.ErrInvalidOption) || len(cloud.Calls()) != 0 {
		t.Fatal(err, cloud.Calls())
	}
}

const r2CloudComputeFloatingServer = `{"server":{"id":"s1","name":"web-1","status":"ACTIVE","addresses":{` +
	`"private":[{"addr":"10.0.0.5","version":4,"OS-EXT-IPS:type":"fixed"},{"addr":"198.51.100.7","version":4,"OS-EXT-IPS:type":"floating"},{"addr":"untagged","version":4}]}}}`

func r2CloudComputeFloatingFixture(t *testing.T, remains bool, fipDelete int) *r2CloudComputeCloud {
	t.Helper()
	cloud := r2CloudComputeNew(t, sdk.WithServerAddressPolicy(compute.WithFloatingIPSource(compute.FloatingIPNeutron)))
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s1", 200, r2CloudComputeFloatingServer)
	cloud.Handle("GET /network/v2.0/floatingips", func(w http.ResponseWriter, r *http.Request, _ string) {
		q := r.URL.Query()
		if q.Get("floating_ip_address") == "198.51.100.7" || remains && q.Get("id") == "fip1" {
			testcloud.JSON(w, 200, `{"floatingips":[{"id":"fip1","floating_ip_address":"198.51.100.7","status":"ACTIVE"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"floatingips":[]}`)
	})
	cloud.JSON("DELETE /network/v2.0/floatingips/fip1", fipDelete, `{"NeutronError":{"message":"x"}}`)
	cloud.JSON("DELETE "+r2CloudComputeBase+"/servers/s1", 204, ``)
	return cloud
}

func TestR2CloudComputeDeleteServerDeleteIPsRemovesFloatingBeforeServer(t *testing.T) {
	cloud := r2CloudComputeFloatingFixture(t, false, 204)
	ok, err := cloud.Conn.DeleteServer(context.Background(), "s1", sdk.WithCloudServerDeleteIPs(true))
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	var order []string
	for _, call := range cloud.Calls() {
		if call.Method == http.MethodDelete {
			order = append(order, call.Path)
		}
	}
	if strings.Join(order, ",") != "/network/v2.0/floatingips/fip1,"+r2CloudComputeBase+"/servers/s1" {
		t.Fatal(order)
	}
}

func TestR2CloudComputeDeleteServerDeleteIPsFailureAbortsServerDelete(t *testing.T) {
	cloud := r2CloudComputeFloatingFixture(t, true, 500)
	ok, err := cloud.Conn.DeleteServer(context.Background(), "s1", sdk.WithCloudServerDeleteIPs(true))
	if err == nil || ok {
		t.Fatal(ok, err)
	}
	for _, call := range cloud.Calls() {
		if call.Method == http.MethodDelete && strings.Contains(call.Path, "/servers/") {
			t.Fatalf("server must not be deleted: %+v", call)
		}
	}
}

func TestR2CloudComputeDeleteServerWithoutDeleteIPsLeavesFloatingAlone(t *testing.T) {
	cloud := r2CloudComputeFloatingFixture(t, false, 204)
	ok, err := cloud.Conn.DeleteServer(context.Background(), "s1")
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	for _, call := range cloud.Calls() {
		if strings.HasPrefix(call.Path, "/network/") {
			t.Fatalf("%+v", call)
		}
	}
}

func TestR2CloudComputeRebuildServerSendsImageAndPasswordAndRejectsEmptyImage(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	cloud.JSON("POST "+r2CloudComputeBase+"/servers/s1/action", 202, `{"server":{"id":"s1","status":"REBUILD","adminPass":"fresh","name":"web-1","addresses":{}}}`)
	ctx := context.Background()
	if _, err := cloud.Conn.RebuildServer(ctx, "s1", ""); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if len(cloud.Calls()) != 0 {
		t.Fatal(cloud.Calls())
	}
	got, err := cloud.Conn.RebuildServer(ctx, "s1", "img1", sdk.WithCloudServerActionBare(true))
	if err != nil || got == nil || !got.Sent || got.Server == nil || got.AdminPassword != "fresh" {
		t.Fatalf("%+v %v", got, err)
	}
	if body := cloud.Calls()[0].Body; body != `{"rebuild":{"imageRef":"img1"}}` {
		t.Fatal(body)
	}
	if _, err = cloud.Conn.RebuildServer(ctx, "s1", "img1", sdk.WithCloudServerAdminPassword("pw"), sdk.WithCloudServerActionBare(true)); err != nil {
		t.Fatal(err)
	}
	if body := cloud.Calls()[1].Body; body != `{"rebuild":{"adminPass":"pw","imageRef":"img1"}}` {
		t.Fatal(body)
	}
}

func TestR2CloudComputeRebuildServerWaitRestoresAdminPasswordOnActiveServer(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	var gets atomic.Int32
	cloud.JSON("POST "+r2CloudComputeBase+"/servers/s1/action", 202, `{"server":{"id":"s1","status":"REBUILD","name":"web-1","addresses":{}}}`)
	cloud.Handle("GET "+r2CloudComputeBase+"/servers/s1", func(w http.ResponseWriter, _ *http.Request, _ string) {
		status := "REBUILD"
		if gets.Add(1) > 1 {
			status = "ACTIVE"
		}
		testcloud.JSON(w, 200, `{"server":{"id":"s1","status":"`+status+`","name":"web-1","addresses":{}}}`)
	})
	got, err := cloud.Conn.RebuildServer(context.Background(), "s1", "img1", sdk.WithCloudServerAdminPassword("supplied"), sdk.WithCloudServerWait(true),
		sdk.WithCloudServerWaitOptions(resource.WithPollInterval(time.Millisecond)), sdk.WithCloudServerActionBare(true))
	if err != nil || got == nil || got.Server == nil || got.Server.Status != "ACTIVE" || got.Server.AdminPass != "supplied" || got.AdminPassword != "supplied" || gets.Load() < 2 {
		t.Fatalf("%+v %v", got, err)
	}
	if posts := r2CloudComputeMethods(cloud.Calls(), http.MethodPost); len(posts) != 1 {
		t.Fatal(posts)
	}
}

func TestR2CloudComputeRebuildServerWaitFailureKeepsActionResultWithoutReplay(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	cloud.JSON("POST "+r2CloudComputeBase+"/servers/s1/action", 202, `{"server":{"id":"s1","status":"REBUILD","adminPass":"fresh","addresses":{}}}`)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s1", 200, `{"server":{"id":"s1","status":"ERROR","addresses":{}}}`)
	got, err := cloud.Conn.RebuildServer(context.Background(), "s1", "img1", sdk.WithCloudServerWait(true), sdk.WithCloudServerActionBare(true))
	if !errors.Is(err, resource.ErrFailedState) || got == nil || got.Server == nil || got.AdminPassword != "fresh" {
		t.Fatalf("%+v %v", got, err)
	}
	if posts := r2CloudComputeMethods(cloud.Calls(), http.MethodPost); len(posts) != 1 {
		t.Fatal(posts)
	}
}
