package gophercloudsdk_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/compute"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestFloatingIPCreatePostNotFoundCanFallbackBeforeAcceptance(t *testing.T) {
	_, conn, state := floatingCreateFixture(t, compute.FloatingIPNeutron)
	state.reply = func(r *http.Request) (int, string) {
		if strings.Contains(r.URL.Path, "networks/net") {
			return 200, `{"network":{"id":"net"}}`
		}
		if strings.Contains(r.URL.Path, "network") {
			return 404, `{"error":"not implemented"}`
		}
		if r.Method == "POST" {
			return 200, `{"floating_ip":{"id":"nova"}}`
		}
		return 200, `{"floating_ip":{"id":"nova","pool":"net"}}`
	}
	result, err := conn.CreateFloatingIP(context.Background(), floatingCreateInput("net"))
	if err != nil || !result.Allocated || result.Backend != compute.FloatingIPNova || result.FallbackError == nil || len(state.events) != 4 || result.FloatingIP.NormalizationSource != compute.FloatingIPNeutron || !reflect.DeepEqual(state.locators, []string{"network", "compute"}) {
		t.Fatal(result, err, state)
	}
}

func TestFloatingIPCreateNovaCompatibilityAndUnsafeIDKeepAllocation(t *testing.T) {
	for _, mode := range []string{"GET403", "GET malformed", "POST null ID", "POST object ID"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := floatingCreateFixture(t, compute.FloatingIPNova)
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "POST" {
					if mode == "POST null ID" {
						return 200, `{"floating_ip":{"id":null}}`
					}
					if mode == "POST object ID" {
						return 200, `{"floating_ip":{"id":{"key":"value"}}}`
					}
					return 200, `{"floating_ip":{"id":"new","ip":"known"}}`
				}
				if mode == "GET403" {
					return 403, `{"error":"compat denied"}`
				}
				return 200, `{"floating_ip":false}`
			}
			result, err := conn.CreateFloatingIP(context.Background(), floatingCreateInput("literal"))
			var proof *resource.ResponseError
			if err == nil || !result.Allocated || result.Allocation == nil || result.Allocation.Wire == nil || result.AllocationResponse.StatusCode != 200 || result.Cleanup != nil {
				t.Fatal(result, err, state)
			}
			if mode == "GET403" {
				if !gophercloud.ResponseCodeIs(err, 403) || result.Compatibility.Failure.StatusCode != 403 {
					t.Fatal(result, err)
				}
			} else if !errors.As(err, &proof) {
				t.Fatal(err)
			}
			if strings.HasPrefix(mode, "POST") {
				if len(state.events) != 1 || result.Compatibility != nil || proof.StatusCode != 200 {
					t.Fatal(result, err, state)
				}
			} else if len(state.events) != 2 || result.Compatibility == nil || result.Compatibility.Failure == nil {
				t.Fatal(result, err, state)
			}
		})
	}
}

func TestFloatingIPCreateResponsePortOverridesSeed(t *testing.T) {
	for _, port := range []string{`null`, `"other"`, `false`} {
		t.Run(port, func(t *testing.T) {
			_, conn, state := floatingCreateFixture(t, compute.FloatingIPNeutron)
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "GET" {
					return 200, `{"network":{"id":"net"}}`
				}
				return 201, `{"floatingip":{"id":"new","port_id":` + port + `}}`
			}
			result, err := conn.CreateFloatingIP(context.Background(), floatingCreateInput("net"), compute.WithFloatingIPCreatePort("port"))
			var proof *resource.ResponseError
			if err == nil || !result.Allocated || !errors.As(err, &proof) || proof.StatusCode != 201 || string(result.FloatingIP.Resource.Body["port_id"]) != port || string(result.FloatingIP.Wire.Body["port_id"]) != port || result.Cleanup != nil || len(state.events) != 2 {
				t.Fatal(result, err, state)
			}
		})
	}
}

func TestFloatingIPCreateDirectWait404DoesNotAllocateAgain(t *testing.T) {
	_, conn, state := floatingCreateFixture(t, compute.FloatingIPNeutron)
	id := "12345678-1234-1234-1234-123456789abc"
	state.reply = func(r *http.Request) (int, string) {
		if strings.Contains(r.URL.Path, "networks/net") {
			return 200, `{"network":{"id":"net"}}`
		}
		if r.Method == "POST" {
			return 201, `{"floatingip":{"id":"` + id + `","port_id":"port"}}`
		}
		return 404, `{"error":"missing member"}`
	}
	result, err := conn.CreateFloatingIP(context.Background(), floatingCreateInput("net"), compute.WithFloatingIPCreatePort("port"), compute.WithFloatingIPCreateWait(true), compute.WithFloatingIPCreateDirectGet(true))
	if err == nil || !result.Allocated || result.Backend != compute.FloatingIPNeutron || result.FallbackError != nil || result.Cleanup != nil || result.Compatibility != nil || len(result.Observations) != 1 || len(state.events) != 3 || state.events[2] != "GET /create-network/v2.0/floatingips/"+id {
		t.Fatal(result, err, state)
	}
}

func TestFloatingIPCreateWholeBudgetSnapshotAndCachedService(t *testing.T) {
	_, conn, state := floatingCreateFixture(t, compute.FloatingIPNeutron)
	service, err := conn.Compute(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	name := "original"
	calls := 0
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	parentDeadline, _ := ctx.Deadline()
	state.reply = func(r *http.Request) (int, string) {
		if r.Method == "GET" {
			return 200, `{"network":{"id":"net"}}`
		}
		return 202, `{"floatingip":{"id":"new"}}`
	}
	result, err := service.CreateFloatingIP(ctx, compute.CreateFloatingIPRequest{Network: &name}, compute.WithFloatingIPCreateTimeout(time.Minute), func(o *compute.FloatingIPCreateOpts) error { calls++; name = "changed"; return nil })
	if err != nil || !result.Allocated || calls != 1 || state.events[0] != "GET /create-network/v2.0/networks/original" || len(state.events) != 2 || !reflect.DeepEqual(state.locators, []string{"compute", "network"}) {
		t.Fatal(result, err, state)
	}
	for _, deadline := range state.deadlines {
		if deadline != parentDeadline {
			t.Fatal(deadline, parentDeadline)
		}
	}
	_, err = conn.CreateFloatingIP(ctx, floatingCreateInput("net"))
	if err != nil || len(state.locators) != 2 {
		t.Fatal(err, state)
	}
}

func TestFloatingIPCreateParentDeadlineIsNotSDKWaitTimeout(t *testing.T) {
	_, conn, state := floatingCreateFixture(t, compute.FloatingIPNeutron)
	state.reply = func(r *http.Request) (int, string) {
		if strings.Contains(r.URL.Path, "networks/net") {
			return 200, `{"network":{"id":"net"}}`
		}
		if r.Method == "POST" {
			return 201, `{"floatingip":{"id":"new"}}`
		}
		<-r.Context().Done()
		return 200, `{"floatingips":[]}`
	}
	result, err := conn.CreateFloatingIP(context.Background(), floatingCreateInput("net"), compute.WithFloatingIPCreatePort("port"), compute.WithFloatingIPCreateWait(true), compute.WithFloatingIPCreateTimeout(20*time.Millisecond))
	var timeout *compute.FloatingIPCreateTimeoutError
	if !errors.Is(err, context.DeadlineExceeded) || errors.As(err, &timeout) || !result.Allocated || result.Cleanup != nil || len(state.events) != 3 {
		t.Fatal(result, err, state)
	}
}

func TestFloatingIPCreateNativeRetryCannotExpandCodesOrChangeSource(t *testing.T) {
	for _, mode := range []string{"expand", "source"} {
		t.Run(mode, func(t *testing.T) {
			cloud, conn, state := floatingCreateFixture(t, compute.FloatingIPNeutron)
			service, err := conn.Network(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			callbacks := 0
			cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				callbacks++
				if mode == "source" {
					service.RawClient().Endpoint += "changed/"
				} else {
					opts.OkCodes = append(opts.OkCodes, 404)
				}
				return nil
			}
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "GET" {
					return 200, `{"network":{"id":"net"}}`
				}
				return 404, `{"error":"not accepted"}`
			}
			result, err := conn.CreateFloatingIP(context.Background(), floatingCreateInput("net"))
			if err == nil || result.Allocated || result.FallbackError != nil || result.Compatibility != nil || callbacks != 1 {
				t.Fatal(result, err, state, callbacks)
			}
			want := 3
			if mode == "source" {
				want = 2
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			}
			if len(state.events) != want {
				t.Fatal(state.events, want)
			}
		})
	}
}

func TestFloatingIPCreateOptionsCannotReplaceCapturedService(t *testing.T) {
	_, conn, state := floatingCreateFixture(t, compute.FloatingIPNeutron)
	service, err := conn.Compute(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.CreateFloatingIP(context.Background(), floatingCreateInput("net"), func(*compute.FloatingIPCreateOpts) error { service.Servers = nil; return nil })
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(state.events) != 0 {
		t.Fatal(result, err, state)
	}
}
