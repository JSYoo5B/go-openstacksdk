package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/compute"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type floatingCreateState struct {
	locators, events []string
	bodies           []string
	deadlines        []time.Time
	reply            func(*http.Request) (int, string)
	afterClose       func(*http.Request) error
	catalogError     error
}

func floatingCreateFixture(t *testing.T, source compute.FloatingIPSource) (*testcloud.Cloud, *sdk.Connection, *floatingCreateState) {
	t.Helper()
	cloud := testcloud.New(t)
	state := &floatingCreateState{}
	cloud.Provider.EndpointLocator = func(o gophercloud.EndpointOpts) (string, error) {
		state.locators = append(state.locators, o.Type)
		if o.Type == "network" && state.catalogError != nil {
			return "", state.catalogError
		}
		return cloud.Server.URL + "/create-" + o.Type + "/", nil
	}
	cloud.Provider.HTTPClient.Transport = serverWorkflowTransport(func(r *http.Request) (*http.Response, error) {
		state.events = append(state.events, r.Method+" "+r.URL.Path)
		var body []byte
		var err error
		if r.Body != nil {
			body, err = io.ReadAll(r.Body)
		}
		if err != nil {
			t.Error(err)
		}
		state.bodies = append(state.bodies, string(body))
		deadline, _ := r.Context().Deadline()
		state.deadlines = append(state.deadlines, deadline)
		if strings.Contains(r.URL.Path, "compute") && r.Header.Get("X-OpenStack-Nova-API-Version") != "2.35" {
			t.Error(r.Header)
		}
		if state.reply == nil {
			return nil, errors.New("missing create fixture")
		}
		code, text := state.reply(r)
		var reader io.ReadCloser = io.NopCloser(strings.NewReader(text))
		if state.afterClose != nil {
			reader = readyConnectionCloseBody{ReadCloser: reader, close: func() error { return state.afterClose(r) }}
		}
		return &http.Response{StatusCode: code, Body: reader, Header: http.Header{"X-Create-Proof": {r.Method + " " + r.URL.Path}}, Request: r}, nil
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.Compute, "2.35"), sdk.WithServerAddressPolicy(compute.WithFloatingIPSource(source)), sdk.WithCloudLocation(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"scope"`)}}))
	if err != nil {
		t.Fatal(err)
	}
	return cloud, conn, state
}
func floatingCreateInput(name string) compute.CreateFloatingIPRequest {
	return compute.CreateFloatingIPRequest{Network: &name}
}

func TestFloatingIPCreateNeutronFreshAndRequestSeedOwnership(t *testing.T) {
	for _, port := range []string{"", "port"} {
		t.Run(port, func(t *testing.T) {
			_, conn, state := floatingCreateFixture(t, compute.FloatingIPNeutron)
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "GET" && strings.Contains(r.URL.Path, "networks/net") {
					return 200, `{"network":{"id":"ordinary","router:external":false}}`
				}
				if r.Method != "POST" {
					t.Error(r.URL)
				}
				return 201, `{"id":"new","vendor":9007199254740993,"port_details":null}`
			}
			result, err := conn.CreateFloatingIP(context.Background(), floatingCreateInput("net"), compute.WithFloatingIPCreatePort(port), compute.WithFloatingIPCreateServer(resource.ID("ignored/path")), compute.WithFloatingIPCreateFixedAddress("ignored"), compute.WithFloatingIPCreateNATDestination("ignored"))
			if port == "" { // No server means no destination lookup in the unattached case.
				if !errors.Is(err, resource.ErrInvalidOption) || result == nil || result.Allocated || len(state.events) != 1 {
					t.Fatal(result, err, state)
				}
				return
			}
			if err != nil || !result.Allocated || result.Waited || result.FloatingIP.Resource == nil || string(result.FloatingIP.Resource.Body["port_id"]) != `"port"` || string(result.FloatingIP.Resource.Body["floating_network_id"]) != `"ordinary"` || len(result.FloatingIP.Wire.Body["port_id"]) != 0 || result.AllocationResponse.StatusCode != 201 || len(state.events) != 2 || !reflect.DeepEqual(state.locators, []string{"network"}) {
				t.Fatal(result, err, state)
			}
			var body map[string]map[string]string
			_ = json.Unmarshal([]byte(state.bodies[1]), &body)
			if !reflect.DeepEqual(body, map[string]map[string]string{"floatingip": {"floating_network_id": "ordinary", "port_id": "port"}}) {
				t.Fatal(body)
			}
			result.FloatingIP.Resource.Body["port_id"][1] = 'X'
			if len(result.FloatingIP.Wire.Body["port_id"]) != 0 || string(result.Allocation.Resource.Body["port_id"]) != `"port"` {
				t.Fatal(result)
			}
		})
	}
}

func TestFloatingIPCreateUnattachedIgnoresWaitAndFixedNAT(t *testing.T) {
	_, conn, state := floatingCreateFixture(t, compute.FloatingIPNeutron)
	state.reply = func(r *http.Request) (int, string) {
		if r.Method == "GET" {
			return 200, `{"network":{"id":"net"}}`
		}
		return 202, `{"floatingip":{"floating_ip_address":null}}`
	}
	result, err := conn.CreateFloatingIP(context.Background(), floatingCreateInput("net"), compute.WithFloatingIPCreateFixedAddress("ignored"), compute.WithFloatingIPCreateNATDestination("ignored"), compute.WithFloatingIPCreateWait(true), compute.WithFloatingIPCreateWaitTimeout(-time.Second))
	if err != nil || !result.Allocated || result.Waited || result.Cleanup != nil || len(result.Observations) != 0 || len(state.events) != 2 || string(result.FloatingIP.Resource.Body["id"]) != "null" || len(result.FloatingIP.Wire.Body["id"]) != 0 {
		t.Fatal(result, err, state)
	}
	if state.bodies[1] != `{"floatingip":{"floating_network_id":"net"}}` {
		t.Fatal(state.bodies)
	}
}

func TestFloatingIPCreateNovaFreshCompatibilityAndPoolPresence(t *testing.T) {
	for _, source := range []compute.FloatingIPSource{compute.FloatingIPNova, compute.FloatingIPNone} {
		for _, mode := range []string{"default", "literal empty", "literal named"} {
			t.Run(string(source)+mode, func(t *testing.T) {
				_, conn, state := floatingCreateFixture(t, source)
				input := compute.CreateFloatingIPRequest{}
				pool := "first"
				if mode == "literal empty" {
					pool = ""
					input = floatingCreateInput(pool)
				} else if mode == "literal named" {
					pool = "literal"
					input = floatingCreateInput(pool)
				}
				state.reply = func(r *http.Request) (int, string) {
					if strings.HasSuffix(r.URL.Path, "os-floating-ip-pools") {
						return 200, `{"floating_ip_pools":[{"name":"first"},{"name":"second"}]}`
					}
					if r.Method == "POST" {
						return 200, `{"floating_ip":{"id":7,"ip":"old"}}`
					}
					if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "os-floating-ips/7") {
						return 201, `{"floating_ip":{"id":7,"ip":null,"pool":"other","instance_id":"attached","status":"ERROR","vendor":9007199254740993}}`
					}
					t.Error(r.URL)
					return 500, `{}`
				}
				result, err := conn.CreateFloatingIP(context.Background(), input, compute.WithFloatingIPCreateServer(resource.ID("ignored/path")), compute.WithFloatingIPCreateFixedAddress("invalid"), compute.WithFloatingIPCreateNATDestination("ignored"), compute.WithFloatingIPCreateWait(true), compute.WithFloatingIPCreateWaitTimeout(-1))
				if err != nil || !result.Allocated || result.Backend != compute.FloatingIPNova || result.Waited || result.Cleanup != nil || result.Compatibility == nil || result.Compatibility.Observed.StatusCode != 201 || string(result.FloatingIP.Resource.Body["status"]) != `"ACTIVE"` || string(result.FloatingIP.Wire.Body["status"]) != `"ERROR"` || string(result.FloatingIP.Wire.Body["vendor"]) != "9007199254740993" {
					t.Fatal(result, err, state)
				}
				index := 0
				if mode == "default" {
					index = 1
					if len(state.events) != 3 || result.PoolQuery == nil {
						t.Fatal(result, state)
					}
				} else if len(state.events) != 2 || result.PoolQuery != nil {
					t.Fatal(result, state)
				}
				var body map[string]string
				_ = json.Unmarshal([]byte(state.bodies[index]), &body)
				if !reflect.DeepEqual(body, map[string]string{"pool": pool}) {
					t.Fatal(body, pool)
				}
			})
		}
	}
}

func TestFloatingIPCreatePreacceptNotFoundFallbackAndPortPolicy(t *testing.T) {
	for _, port := range []string{"", "port"} {
		t.Run(port, func(t *testing.T) {
			_, conn, state := floatingCreateFixture(t, compute.FloatingIPNeutron)
			state.reply = func(r *http.Request) (int, string) {
				if strings.Contains(r.URL.Path, "networks/") {
					return 404, `{"error":"missing"}`
				}
				if strings.HasSuffix(r.URL.Path, "networks") {
					return 200, `{"networks":[]}`
				}
				if r.Method == "POST" {
					return 200, `{"floating_ip":{"id":"new"}}`
				}
				return 200, `{"floating_ip":{"id":"new","ip":"198.51.100.3","status":"DOWN"}}`
			}
			result, err := conn.CreateFloatingIP(context.Background(), floatingCreateInput("literal"), compute.WithFloatingIPCreatePort(port))
			if result.FallbackError == nil || result.Backend != compute.FloatingIPNova {
				t.Fatal(result, err)
			}
			if port != "" {
				if !errors.Is(err, resource.ErrUnsupported) || len(state.events) != 2 || result.Allocated {
					t.Fatal(result, err, state)
				}
			} else if err != nil || !result.Allocated || len(state.events) != 4 || result.FloatingIP.NormalizationSource != compute.FloatingIPNeutron || string(result.FloatingIP.Resource.Body["status"]) != `"DOWN"` {
				t.Fatal(result, err, state)
			}
		})
	}
}

func TestFloatingIPCreateAcceptedDecodeCloseCancelAndSourceFailures(t *testing.T) {
	for _, mode := range []string{"malformed", "close", "cancel", "source"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := floatingCreateFixture(t, compute.FloatingIPNeutron)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("after accepted create")
			service, err := conn.Network(ctx)
			if err != nil {
				t.Fatal(err)
			}
			state.reply = func(r *http.Request) (int, string) {
				if r.Method == "GET" {
					return 200, `{"network":{"id":"net"}}`
				}
				if mode == "malformed" {
					return 202, `not-json`
				}
				return 202, `{"floatingip":{"id":"new","port_id":"port"}}`
			}
			state.afterClose = func(r *http.Request) error {
				if r.Method != "POST" {
					return nil
				}
				switch mode {
				case "close":
					return cause
				case "cancel":
					cancel(cause)
				case "source":
					service.RawClient().Endpoint += "changed/"
				}
				return nil
			}
			result, err := conn.CreateFloatingIP(ctx, floatingCreateInput("net"), compute.WithFloatingIPCreatePort("port"), compute.WithFloatingIPCreateWait(true))
			var proof *resource.ResponseError
			if err == nil || !result.Allocated || result.AllocationResponse == nil || result.AllocationResponse.StatusCode != 202 || result.AllocationResponse.Header.Get("X-Create-Proof") == "" || !errors.As(err, &proof) || proof.StatusCode != 202 || len(state.events) != 2 || result.Cleanup != nil || result.Compatibility != nil || len(result.Observations) != 0 {
				t.Fatal(result, err, state)
			}
			if (mode == "close" || mode == "cancel") && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if mode == "source" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestFloatingIPCreateWaitPublicGetSkipsErrorAndChecksFinalPort(t *testing.T) {
	for _, mode := range []string{"active", "port mismatch", "lookup denied"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := floatingCreateFixture(t, compute.FloatingIPNeutron)
			gets := 0
			state.reply = func(r *http.Request) (int, string) {
				if strings.Contains(r.URL.Path, "networks/net") {
					return 200, `{"network":{"id":"net"}}`
				}
				if r.Method == "POST" {
					return 201, `{"floatingip":{"id":"new"}}`
				}
				if r.Method == "DELETE" {
					t.Error("unexpected cleanup")
					return 204, ""
				}
				gets++
				if mode == "lookup denied" {
					return 403, `{}`
				}
				if gets == 1 {
					return 200, `{"floatingips":[]}`
				}
				if gets == 2 {
					return 200, `{"floatingips":[{"id":"new","status":"ERROR","port_id":"port"}]}`
				}
				port := "port"
				if mode == "port mismatch" {
					port = "other"
				}
				return 200, fmt.Sprintf(`{"floatingips":[{"id":"new","status":"\u0041CTIVE","port_id":%q}]}`, port)
			}
			result, err := conn.CreateFloatingIP(context.Background(), floatingCreateInput("net"), compute.WithFloatingIPCreatePort("port"), compute.WithFloatingIPCreateWait(true), compute.WithFloatingIPCreateWaitTimeout(time.Second), compute.WithFloatingIPCreateWaitInterval(time.Millisecond))
			if !result.Allocated || result.Cleanup != nil || result.Backend != compute.FloatingIPNeutron {
				t.Fatal(result, err)
			}
			if mode == "active" {
				if err != nil || !result.Waited || len(result.Observations) != 3 || len(state.events) != 5 {
					t.Fatal(result, err, state)
				}
			} else if err == nil || result.Waited || result.Compatibility != nil {
				t.Fatal(result, err, state)
			}
		})
	}
}

func TestFloatingIPCreateSDKWaitTimeoutCleansWithLiveParent(t *testing.T) {
	for _, mode := range []string{"immediate", "negative", "real", "cleanup denied"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := floatingCreateFixture(t, compute.FloatingIPNeutron)
			deletes := 0
			state.reply = func(r *http.Request) (int, string) {
				if strings.Contains(r.URL.Path, "networks/net") {
					return 200, `{"network":{"id":"net"}}`
				}
				if r.Method == "POST" {
					return 202, `{"floatingip":{"id":"new","port_id":"port"}}`
				}
				if r.Method == "DELETE" {
					deletes++
					if mode == "cleanup denied" {
						return 403, `{"error":"cleanup denied"}`
					}
					return 204, ""
				}
				if deletes > 0 {
					return 200, `{"floatingips":[]}`
				}
				return 200, `{"floatingips":[{"id":"new","status":"ERROR","port_id":"port"}]}`
			}
			duration := time.Duration(0)
			if mode == "negative" {
				duration = -time.Second
			}
			if mode == "real" {
				duration = 20 * time.Millisecond
			}
			result, err := conn.CreateFloatingIP(context.Background(), floatingCreateInput("net"), compute.WithFloatingIPCreatePort("port"), compute.WithFloatingIPCreateWait(true), compute.WithFloatingIPCreateWaitTimeout(duration), compute.WithFloatingIPCreateWaitInterval(time.Millisecond))
			var timeout *compute.FloatingIPCreateTimeoutError
			if !errors.As(err, &timeout) || !errors.Is(err, context.DeadlineExceeded) || !result.Allocated || result.Waited || result.Cleanup == nil || deletes != 1 || timeout.ID != "new" {
				t.Fatal(result, err, state)
			}
			if mode == "cleanup denied" {
				if result.CleanupError == nil || result.Cleanup.Deleted {
					t.Fatal(result, err)
				}
			} else if result.CleanupError != nil || !result.Cleanup.Deleted || !result.Cleanup.Absent {
				t.Fatal(result, err)
			}
			if !state.deadlines[0].IsZero() || !state.deadlines[1].IsZero() {
				t.Fatal(state.deadlines)
			}
			for i, event := range state.events {
				if strings.HasPrefix(event, "DELETE") && !state.deadlines[i].IsZero() {
					t.Fatal("cleanup used expired wait budget", state)
				}
			}
		})
	}
}

func TestFloatingIPCreateCallerCancellationNeverDetachesCleanup(t *testing.T) {
	_, conn, state := floatingCreateFixture(t, compute.FloatingIPNeutron)
	cause := errors.New("caller stopped wait")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	state.reply = func(r *http.Request) (int, string) {
		if strings.Contains(r.URL.Path, "networks/net") {
			return 200, `{"network":{"id":"net"}}`
		}
		if r.Method == "POST" {
			return 201, `{"floatingip":{"id":"new"}}`
		}
		cancel(cause)
		return 200, `{"floatingips":[]}`
	}
	result, err := conn.CreateFloatingIP(ctx, floatingCreateInput("net"), compute.WithFloatingIPCreatePort("port"), compute.WithFloatingIPCreateWait(true))
	if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || !result.Allocated || result.Cleanup != nil || len(state.events) != 3 {
		t.Fatal(result, err, state)
	}
}
