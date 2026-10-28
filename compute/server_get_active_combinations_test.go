package compute_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func decodedServer(t *testing.T, body string) *compute.Server {
	t.Helper()
	var server compute.Server
	if err := json.Unmarshal([]byte(body), &server); err != nil {
		t.Fatal(err)
	}
	return &server
}

// Python's get_active_server compares server['status'] with ERROR and ACTIVE
// and returns None otherwise, so a null or missing status is not ready. ERROR
// is checked before addresses, so the cleanup policy never deletes an ERROR
// server, and the message form follows the fault model. Null model fields of
// an ACTIVE server do not block a disabled automatic decision.
func TestGetActiveServerNullableStatusFaultAndModelCases(t *testing.T) {
	cases := []struct {
		name, body, message string
		ready, failed       bool
	}{
		{name: "null-status", body: `{"id":"server","status":null,"addresses":{"bad":true}}`},
		{name: "missing-status", body: `{"id":"server","addresses":null}`},
		{name: "error-null-fault", body: `{"id":"server","status":"ERROR","fault":null,"addresses":null}`, failed: true, message: "no further information available"},
		{name: "error-fault-without-message", body: `{"id":"server","status":"ERROR","fault":{"code":500},"addresses":{}}`, failed: true, message: "no further information available"},
		{name: "error-empty-message", body: `{"id":"server","status":"ERROR","fault":{"message":""}}`, failed: true, message: "no further information available"},
		{name: "error-message", body: `{"id":"server","status":"ERROR","fault":{"message":"No valid host","code":500},"addresses":` + autoFixed + `}`, failed: true, message: "Compute service reports fault: No valid host"},
		{name: "active-null-model", body: `{"id":"server","status":"ACTIVE","name":null,"accessIPv4":null,"accessIPv6":null,"image":"","flavor":null,"metadata":null,"fault":null,"addresses":` + autoFixed + `}`, ready: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newAutomaticFixture(t)
			deletes := 0
			f.cloud.Mux.HandleFunc("DELETE /v2.1/servers/server", func(w http.ResponseWriter, r *http.Request) { deletes++; w.WriteHeader(204) })
			opts := append(serverReadyOptions(), compute.WithServerWithoutAddressesPolicy(compute.DeleteServerWithoutAddresses), compute.WithServerReadyAutomaticIPOptions(compute.WithAutomaticIPEnabled(false)))
			result, err := f.service.GetActiveServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: decodedServer(t, tc.body)}, opts...)
			if deletes != 0 || f.raw.Load() != 0 || f.posts.Load() != 0 || f.ports.Load() != 0 || f.roles.Load() != 0 {
				t.Fatal("unexpected request", deletes, f.raw.Load(), f.posts.Load(), f.ports.Load(), f.roles.Load())
			}
			switch {
			case tc.failed:
				var fault *compute.ServerFaultError
				if !errors.As(err, &fault) || !errors.Is(err, resource.ErrFailedState) || !strings.Contains(err.Error(), tc.message) || result == nil || result.Server == nil || result.Server.ID != "server" || result.Assignment != nil {
					t.Fatal(result, err)
				}
			case tc.ready:
				if err != nil || result == nil || result.Decision.Reason != compute.AutomaticIPDisabled || result.Decision.Needed || result.Server.Name != "" {
					t.Fatal(result, err)
				}
			default:
				if result != nil || err != nil {
					t.Fatal(result, err)
				}
			}
		})
	}
}

// Python's get_active_server passes ips to add_ips_to_server, whose ips
// branch is add_ip_list. GetActiveServer's address lane therefore follows the
// same _attach_ip_to_server steps as AddIPList: the supplied model's only
// floating address short-circuits, a Neutron 404 from the association PUT is
// recorded before the next address, and without a floating address on the
// supplied model an already associated IP refreshes the server once even in
// the asynchronous lane.
func TestGetActiveServerAddressLaneFollowsSourceAttachSteps(t *testing.T) {
	common := []compute.AutomaticFloatingIPOption{compute.WithAutomaticIPPollInterval(time.Millisecond), compute.WithAutomaticAddressOptions(compute.WithAddressReachability(false))}
	ready := func(extra ...compute.AutomaticFloatingIPOption) compute.ServerReadyOption {
		return compute.WithServerReadyAutomaticIPOptions(append(append([]compute.AutomaticFloatingIPOption(nil), common...), extra...)...)
	}
	t.Run("short-circuit-and-not-found", func(t *testing.T) {
		f := newSourceIPFixture(t)
		f.ports = []string{sourcePortPrivate}
		f.notFound["put:b"] = true
		server := automaticServer(t, sourceAddresses(dispatchAddresses["a"]))
		result, err := f.service.GetActiveServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, ready(compute.WithFloatingIPAddresses(dispatchAddresses["a"], dispatchAddresses["b"], dispatchAddresses["c"])))
		trace := f.trace()
		if err != nil || result == nil || result.Mode != compute.ServerIPExplicit || len(result.Attempts) != 3 || result.Observed || trace != "list:a,list:b,ports,port:port-p,get:b,put:b,list:c,ports,port:port-p,get:c,put:c" {
			t.Fatal(result, err, trace)
		}
		first, second, third := result.Attempts[0], result.Attempts[1], result.Attempts[2]
		var native gophercloud.ErrUnexpectedResponseCode
		if !first.ShortCircuited || !first.Completed || second.Completed || second.Error != nil || !errors.As(second.AttachmentNotFound, &native) || native.Actual != 404 || !third.Completed || third.ShortCircuited || f.rawCalls.Load() != 0 {
			t.Fatal(result.Attempts)
		}
	})
	for _, wait := range []bool{false, true} {
		t.Run(fmt.Sprintf("refresh/wait=%v", wait), func(t *testing.T) {
			f := newSourceIPFixture(t)
			f.ports = []string{sourcePortPrivate}
			f.bound["a"] = [2]string{"port-p", "10.0.0.10"}
			f.raw = func(int32) string { return sourceAddresses(dispatchAddresses["a"]) }
			result, err := f.service.GetActiveServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, ready(compute.WithFloatingIPAddresses(dispatchAddresses["a"]), compute.WithAutomaticIPTimeout(5*time.Second)), compute.WithActiveServerWait(wait))
			if err != nil || f.trace() != "list:a,raw" || len(result.Attempts) != 1 || !result.Attempts[0].ShortCircuited || result.Observed || result.Server.Name != "raw-1" || len(f.puts) != 0 {
				t.Fatal(result, err, f.trace())
			}
		})
	}
	t.Run("unassociated-no-refresh", func(t *testing.T) {
		f := newSourceIPFixture(t)
		f.ports = []string{sourcePortPrivate}
		result, err := f.service.GetActiveServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, ready(compute.WithFloatingIPAddresses(dispatchAddresses["a"])))
		if err != nil || f.trace() != "list:a,ports,port:port-p,get:a,put:a" || !result.Attempts[0].Completed || result.Attempts[0].ShortCircuited || f.rawCalls.Load() != 0 {
			t.Fatal(result, err, f.trace())
		}
	})
	t.Run("nova-short-circuit", func(t *testing.T) {
		f := newNovaIPFixture(t, "")
		server := automaticServer(t, sourceAddresses(dispatchAddresses["b"]))
		nova := compute.WithAutomaticAddressOptions(compute.WithFloatingIPSource(compute.FloatingIPNova), compute.WithAddressReachability(false))
		result, err := f.service.GetActiveServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, compute.WithServerReadyAutomaticIPOptions(nova, compute.WithAutomaticIPPollInterval(time.Millisecond), compute.WithFloatingIPAddresses(dispatchAddresses["b"], dispatchAddresses["c"])), compute.WithServerReadyAutomaticIPOptions(compute.WithAutomaticEnsureOptions(network.WithEnsureFixedAddress("10.0.0.10"))))
		trace := strings.Join(f.trace(), ",")
		if err != nil || result == nil || len(result.Attempts) != 2 || !result.Attempts[0].ShortCircuited || result.Attempts[1].ShortCircuited || !result.Attempts[1].Completed || result.NovaAssignment == nil || !result.NovaAssignment.ActionAccepted || trace != "list,list,get:3,attach:"+dispatchAddresses["c"] {
			t.Fatal(result, err, trace)
		}
	})
}

// requestDeadlines records the deadline of every request sent by the cloud.
// Requests whose path is in stage report their deadline separately.
func requestDeadlines(t *testing.T, cloud *testcloud.Cloud, hook func(*http.Request), stage ...string) func() ([]time.Time, int) {
	var mu sync.Mutex
	var deadlines []time.Time
	missing := 0
	staged := map[string]bool{}
	for _, path := range stage {
		staged[path] = true
	}
	base := cloud.Provider.HTTPClient.Transport
	cloud.SetTransport(testcloud.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		if deadline, ok := r.Context().Deadline(); staged[r.URL.Path] {
			if !ok {
				t.Error("stage request without its own deadline", r.URL)
			}
		} else if ok {
			deadlines = append(deadlines, deadline)
		} else {
			missing++
		}
		mu.Unlock()
		response, err := base.RoundTrip(r)
		if hook != nil {
			hook(r)
		}
		return response, err
	}))
	return func() ([]time.Time, int) {
		mu.Lock()
		defer mu.Unlock()
		return append([]time.Time(nil), deadlines...), missing
	}
}

// One deadline bounds the whole GetActiveServer IP stage: 180 seconds by
// default, an override, an earlier parent, the parent alone when unlimited,
// and no deadline at all when unlimited under a parent without one. Python
// forwards timeout=180 to add_ips_to_server; Go never renews the budget
// between allocation, IP readiness and raw observation.
func TestGetActiveServerTimeoutUnlimitedAndParentCombinations(t *testing.T) {
	for _, wait := range []bool{false, true} {
		for _, mode := range []string{"default", "override", "parent", "unlimited", "unlimited-no-parent"} {
			t.Run(fmt.Sprintf("wait=%v/%s", wait, mode), func(t *testing.T) {
				f := newAutomaticFixture(t)
				// The Neutron IP ACTIVE wait keeps its own prepared stage
				// timeout inside the overall budget, so its GET is checked
				// separately when no overall deadline exists.
				stage := []string{}
				if mode == "unlimited-no-parent" {
					stage = append(stage, "/v2.0/floatingips/ip")
				}
				deadlines := requestDeadlines(t, f.cloud, nil, stage...)
				options := []compute.AutomaticFloatingIPOption{
					compute.WithAutomaticAddressOptions(compute.WithAddressReachability(false)),
					compute.WithAutomaticEnsureOptions(network.WithEnsureReuse(false), network.WithEnsureWait(resource.WithPollInterval(time.Millisecond))),
					compute.WithAutomaticIPPollInterval(time.Millisecond),
				}
				ctx, length := context.Background(), 4*time.Minute
				switch mode {
				case "override":
					options = append(options, compute.WithAutomaticIPTimeout(17*time.Second))
				case "parent":
					length = 30 * time.Second
				case "unlimited", "unlimited-no-parent":
					options = append(options, compute.WithUnlimitedAutomaticIPTimeout())
				}
				if mode != "unlimited-no-parent" {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, length)
					defer cancel()
				}
				parent, _ := ctx.Deadline()
				start := time.Now()
				result, err := f.service.GetActiveServer(ctx, compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, compute.WithServerReadyAutomaticIPOptions(options...), compute.WithActiveServerWait(wait))
				if err != nil || result == nil || result.Assignment == nil || result.Observed != wait || f.posts.Load() != 1 || (f.raw.Load() == 1) != wait {
					t.Fatal(result, err, f.raw.Load())
				}
				seen, missing := deadlines()
				if mode == "unlimited-no-parent" {
					if len(seen) != 0 || missing == 0 {
						t.Fatal("unexpected deadline", seen, missing)
					}
					return
				}
				if len(seen) == 0 || missing != 0 {
					t.Fatal("missing deadline", seen, missing)
				}
				for _, deadline := range seen {
					if !deadline.Equal(seen[0]) {
						t.Fatal("renewed budget", seen)
					}
				}
				want := map[string]time.Duration{"default": 180 * time.Second, "override": 17 * time.Second}[mode]
				if want == 0 {
					if !seen[0].Equal(parent) {
						t.Fatal(mode, seen[0], parent)
					}
				} else if remaining := seen[0].Sub(start); remaining > want+time.Second || remaining < want-time.Second {
					t.Fatal(mode, remaining)
				}
			})
		}
	}
}

// A parent canceled before the call stops before any request. A parent
// canceled with a cause after the accepted allocation keeps the known Server
// and Assignment beside the cause, without IP readiness, observation, retry
// or cleanup.
func TestGetActiveServerParentContextCancellation(t *testing.T) {
	t.Run("before", func(t *testing.T) {
		f := newAutomaticFixture(t)
		ctx, cancel := context.WithCancelCause(context.Background())
		stop := errors.New("caller stopped")
		cancel(stop)
		result, err := f.service.GetActiveServer(ctx, compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, serverReadyOptions()...)
		if result != nil || !errors.Is(err, stop) || f.posts.Load() != 0 || f.roles.Load() != 0 || f.raw.Load() != 0 {
			t.Fatal(result, err)
		}
	})
	for _, wait := range []bool{false, true} {
		t.Run(fmt.Sprintf("after-allocation/wait=%v", wait), func(t *testing.T) {
			f := newAutomaticFixture(t)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			stop := errors.New("caller stopped after allocation")
			ipGets := 0
			requestDeadlines(t, f.cloud, func(r *http.Request) {
				if r.Method == http.MethodPost && r.URL.Path == "/v2.0/floatingips" {
					cancel(stop)
				}
				if r.URL.Path == "/v2.0/floatingips/ip" {
					ipGets++
				}
			})
			opts := append(serverReadyOptions(), compute.WithActiveServerWait(wait))
			result, err := f.service.GetActiveServer(ctx, compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, opts...)
			if !errors.Is(err, stop) || result == nil || result.Server == nil || result.Server.ID != "server" || result.Assignment == nil || !result.Assignment.Allocated || result.Observed || f.posts.Load() != 1 || f.raw.Load() != 0 || ipGets != 0 {
				t.Fatal(result, err, f.posts.Load(), f.raw.Load(), ipGets)
			}
		})
	}
}
