package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/dns/v2/zones"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPyDNSZoneDeleteWithSharesSendsHeaderOnlyWhenTrue(t *testing.T) {
	cloud := testcloud.New(t)
	var headers []http.Header
	cloud.Mux.HandleFunc("DELETE /v2/zones/zone-1", func(w http.ResponseWriter, r *http.Request) {
		headers = append(headers, r.Header.Clone())
		testcloud.JSON(w, 202, designateZoneBody)
	})
	cloud.Mux.HandleFunc("DELETE /v2/zones/missing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	api := newDesignateZones(t, cloud)
	ctx := context.Background()
	for _, shares := range []bool{true, false} {
		value, err := api.DeleteWithShares(ctx, "zone-1", shares)
		if err != nil || value == nil || value.ID != "zone-1" || value.Status != "PENDING" {
			t.Fatalf("shares=%v value=%+v error=%v", shares, value, err)
		}
	}
	if len(headers) != 2 || headers[0].Get("X-Designate-Delete-Shares") != "true" {
		t.Fatalf("headers=%v", headers)
	}
	if _, ok := headers[1]["X-Designate-Delete-Shares"]; ok {
		t.Fatalf("false must omit the header: %v", headers[1])
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.DeleteWithShares(ctx, "missing", true); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native 404 stays an error: %v", err)
	}
	if err := resource.IgnoreMissing(func() error { _, err := api.DeleteWithShares(ctx, "missing", true); return err }()); err != nil {
		t.Fatalf("IgnoreMissing: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores a missing zone by default: %v", err)
	}
}

func TestPyDNSZoneDeleteWithSharesOnlyAccepts202AndHonorsCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	status := 200
	cloud.Mux.HandleFunc("DELETE /v2/zones/zone-1", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, status, designateZoneBody)
	})
	api := newDesignateZones(t, cloud)
	if _, err := api.DeleteWithShares(context.Background(), "zone-1", false); err == nil {
		t.Fatal("200 must not be accepted")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	before := calls.Load()
	if _, err := api.DeleteWithShares(canceled, "zone-1", true); !errors.Is(err, context.Canceled) || calls.Load() != before {
		t.Fatalf("error=%v calls=%d->%d", err, before, calls.Load())
	}
}

func TestPyDNSZoneAbandonAndXfrPostWithoutBody(t *testing.T) {
	cloud := testcloud.New(t)
	seen := map[string]int{}
	code := 202
	for _, task := range []string{"abandon", "xfr"} {
		cloud.Mux.HandleFunc("POST /v2/zones/zone-1/tasks/"+task, func(w http.ResponseWriter, r *http.Request) {
			data, _ := io.ReadAll(r.Body)
			if len(data) != 0 || r.Header.Get("Content-Type") != "" {
				t.Errorf("%s must send no body: %q %q", task, data, r.Header.Get("Content-Type"))
			}
			seen[task]++
			w.WriteHeader(code)
		})
	}
	api := newDesignateZones(t, cloud)
	ctx := context.Background()
	for _, c := range []int{202, 204} {
		code = c
		if err := api.Abandon(ctx, "zone-1"); err != nil {
			t.Fatalf("abandon %d: %v", c, err)
		}
		if err := api.Xfr(ctx, "zone-1"); err != nil {
			t.Fatalf("xfr %d: %v", c, err)
		}
	}
	for _, c := range []int{200, 400, 404, 500} {
		code = c
		var unexpected gophercloud.ErrUnexpectedResponseCode
		if err := api.Abandon(ctx, "zone-1"); err == nil {
			t.Fatalf("abandon %d accepted", c)
		} else if c == 404 && !errors.As(err, &unexpected) {
			t.Fatalf("native 404 lost: %v", err)
		}
		if err := api.Xfr(ctx, "zone-1"); err == nil {
			t.Fatalf("xfr %d accepted", c)
		}
	}
	if seen["abandon"] != 6 || seen["xfr"] != 6 {
		t.Fatalf("seen=%v", seen)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := api.Abandon(canceled, "zone-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("abandon: %v", err)
	}
	if err := api.Xfr(canceled, "zone-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("xfr: %v", err)
	}
	if seen["abandon"] != 6 || seen["xfr"] != 6 {
		t.Fatalf("canceled calls reached HTTP: %v", seen)
	}
}

func TestPyDNSZoneNameserversPreserveOrderAndFailures(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/zones/zone-1/nameservers", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"nameservers":[{"hostname":"ns2.example.org.","priority":2,"extra":1},{"hostname":"ns1.example.org.","priority":1}]}`)
	})
	cloud.Mux.HandleFunc("GET /v2/zones/empty/nameservers", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v2/zones/missing/nameservers", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v2/zones/bad/nameservers", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 202, `{"nameservers":[]}`)
	})
	api := newDesignateZones(t, cloud)
	ctx := context.Background()
	got, err := api.Nameservers(ctx, "zone-1")
	want := []zones.Nameserver{{Hostname: "ns2.example.org.", Priority: 2}, {Hostname: "ns1.example.org.", Priority: 1}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%v error=%v", got, err)
	}
	if got, err := api.Nameservers(ctx, "empty"); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty: %#v %v", got, err)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.Nameservers(ctx, "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("missing: %v", err)
	}
	if _, err := api.Nameservers(ctx, "bad"); err == nil {
		t.Fatal("only 200 is accepted")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Nameservers(canceled, "zone-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled: %v", err)
	}
}

func TestPyDNSZoneFindIDUsesOnlyGETAndNameUsesList(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, lists atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/zones/zone-1", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		testcloud.JSON(w, 200, designateZoneBody)
	})
	cloud.Mux.HandleFunc("GET /v2/zones/absent", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v2/zones", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if r.URL.Query().Get("name") == "dup." {
			testcloud.JSON(w, 200, `{"zones":[{"id":"a","name":"dup."},{"id":"b","name":"dup."}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"zones":[]}`)
	})
	api := newDesignateZones(t, cloud)
	ctx := context.Background()
	if v, err := api.Find(ctx, resource.ID("zone-1")); err != nil || v == nil || v.ID != "zone-1" || gets.Load() != 1 || lists.Load() != 0 {
		t.Fatalf("id find: %+v %v gets=%d lists=%d", v, err, gets.Load(), lists.Load())
	}
	// Python find_zone defaults to ignore_missing=True; Go defaults to an error.
	if _, err := api.Find(ctx, resource.ID("absent")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("strict default: %v", err)
	}
	if v, err := api.Find(ctx, resource.ID("absent"), resource.WithIgnoreMissing()); err != nil || v != nil {
		t.Fatalf("ignore missing: %v %v", v, err)
	}
	if _, err := api.Find(ctx, resource.Name("none.")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("name strict: %v", err)
	}
	if v, err := api.Find(ctx, resource.Name("none."), resource.WithIgnoreMissing()); err != nil || v != nil {
		t.Fatalf("name ignore missing: %v %v", v, err)
	}
	if _, err := api.Find(ctx, resource.Name("dup.")); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatalf("duplicate names must fail: %v", err)
	}
	if gets.Load() != 3 || lists.Load() != 3 {
		t.Fatalf("name lookups must not issue member GETs: gets=%d lists=%d", gets.Load(), lists.Load())
	}
}

func TestPyDNSZoneCreateSecondaryKeepsCallerFields(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /v2/zones", func(w http.ResponseWriter, r *http.Request) {
		want := map[string]string{"name": `"sec.example.org."`, "type": `"SECONDARY"`, "masters": `["192.0.2.1"]`, "email": `"a@example.org"`, "ttl": `600`}
		if got := designateStrings(designateFlatBody(t, r)); !reflect.DeepEqual(got, want) {
			t.Errorf("body=%v", got)
		}
		testcloud.JSON(w, 202, designateZoneBody)
	})
	_, err := newDesignateZones(t, cloud).Create(context.Background(), zones.CreateOpts{
		Name: "sec.example.org.", Type: "SECONDARY", Masters: []string{"192.0.2.1"}, Email: "a@example.org", TTL: 600})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPyDNSZoneWaitsUseStatusFailureAndAbsence(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var n atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/zones/zone-1", func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) < 3 {
			testcloud.JSON(w, 200, `{"id":"zone-1","status":"PENDING"}`)
			return
		}
		testcloud.JSON(w, 200, `{"id":"zone-1","status":"active"}`)
	})
	cloud.Mux.HandleFunc("GET /v2/zones/bad", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"id":"bad","status":"ERROR"}`)
	})
	var gone atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/zones/gone", func(w http.ResponseWriter, r *http.Request) {
		if gone.Add(1) < 2 {
			testcloud.JSON(w, 200, `{"id":"gone","status":"PENDING","action":"DELETE"}`)
			return
		}
		testcloud.JSON(w, 404, `{}`)
	})
	api := newDesignateZones(t, cloud)
	ctx := context.Background()
	fast := resource.WithPollInterval(5 * time.Millisecond)
	v, err := api.WaitFor(ctx, resource.ID("zone-1"), "ACTIVE", fast, resource.WithTimeout(5*time.Second))
	if err != nil || v == nil || v.Status != "active" || n.Load() < 3 {
		t.Fatalf("v=%+v err=%v polls=%d", v, err, n.Load())
	}
	if _, err := api.WaitFor(ctx, resource.ID("bad"), "ACTIVE", fast); !errors.Is(err, resource.ErrFailedState) {
		t.Fatalf("ERROR must fail the wait: %v", err)
	}
	if err := api.WaitForDeletion(ctx, resource.ID("gone"), fast, resource.WithTimeout(5*time.Second)); err != nil || gone.Load() != 2 {
		t.Fatalf("err=%v polls=%d", err, gone.Load())
	}
}
