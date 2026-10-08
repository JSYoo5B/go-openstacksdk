package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func attachRequest() network.AttachFloatingIPRequest {
	return network.AttachFloatingIPRequest{Server: resource.ID("server"), IP: resource.ID("fip")}
}

func attachIP(port, fixed, status, revision string) string {
	return fmt.Sprintf(`{"id":"fip","project_id":"foreign","floating_network_id":"external","floating_ip_address":"198.51.100.10","port_id":%q,"fixed_ip_address":%q,"status":%q,"tags":["original"]%s}`, port, fixed, status, revision)
}

func attachUnexpected(t *testing.T, cloud *testcloud.Cloud) {
	t.Helper()
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected attach HTTP", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
}

func TestFloatingIPAttachMovesExactExistingIPWithoutOwnerOrAllocation(t *testing.T) {
	for _, revision := range []string{`,"revision_number":0`, `,"revision_number":null`, ""} {
		t.Run(revision, func(t *testing.T) {
			cloud := testcloud.New(t)
			ensurePortFixture(t, cloud)
			plannedPortRead(t, cloud)
			var reads, writes atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
				currentRevision := revision
				if reads.Add(1) > 1 {
					currentRevision = `,"revision_number":7`
				}
				testcloud.JSON(w, 200, `{"floatingip":`+attachIP("old-port", "10.1.0.20", "DOWN", currentRevision)+`}`)
			})
			cloud.Mux.HandleFunc("PUT /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
				writes.Add(1)
				wantHeader := ""
				if strings.Contains(revision, ":0") {
					wantHeader = "revision_number=0"
				}
				if r.Header.Get("If-Match") != wantHeader {
					t.Error("revision changed", r.Header)
				}
				if got := floatingIPBody(t, r); !reflect.DeepEqual(got, map[string]any{"port_id": "port", "fixed_ip_address": "10.0.0.10"}) {
					t.Error(got)
				}
				testcloud.JSON(w, 200, `{"floatingip":`+attachIP("port", "10.0.0.10", "DOWN", "")+`}`)
			})
			attachUnexpected(t, cloud)
			service := network.New(cloud.Client("network", "/v2.0"))
			planner, err := service.FloatingIPs.NewPlanner(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			plan, err := planner.PrepareAttach(context.Background(), attachRequest())
			want := network.FloatingIPAttachSelection{FloatingIPSelection: network.FloatingIPSelection{ServerID: "server", NetworkID: "external", PortNetworkID: "private", PortID: "port", FixedIPv4: "10.0.0.10"}, IPID: "fip", Address: "198.51.100.10"}
			if err != nil || plan.Selection() != want || reads.Load() != 1 || writes.Load() != 0 {
				t.Fatal(plan.Selection(), err, reads.Load(), writes.Load())
			}
			copy := plan.Selection()
			copy.Address, copy.PortID = "198.51.100.99", "replacement"
			result, err := service.FloatingIPs.AttachPrepared(context.Background(), plan)
			if err != nil || result == nil || !result.Reused || result.Allocated || result.FloatingIP.PortID != "port" || result.FloatingIP.ProjectID != "foreign" || reads.Load() != 2 || writes.Load() != 1 {
				t.Fatal(result, err, reads.Load(), writes.Load())
			}
		})
	}
}

func TestFloatingIPAttachAddressConsumesEveryPageBeforeSelection(t *testing.T) {
	for _, scenario := range []string{"last page", "empty HTTP page", "duplicate", "late403", "late malformed", "query drift", "missing", "terminal204"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			var reads, ports atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				reads.Add(1)
				if len(r.URL.Query()) != 1 && len(r.URL.Query()) != 2 || r.URL.Query().Get("floating_ip_address") != "198.51.100.10" {
					t.Error("unexpected scoped filter", r.URL)
				}
				if r.URL.Query().Get("marker") != "" {
					switch scenario {
					case "late403":
						http.Error(w, "late attach denied", 403)
						return
					case "late malformed":
						testcloud.JSON(w, 200, `{"floatingips":`)
						return
					case "terminal204":
						w.WriteHeader(204)
						return
					}
					row := attachIP("other", "10.1.0.1", "ERROR", "")
					if scenario == "missing" {
						row = strings.Replace(row, "198.51.100.10", "198.51.100.99", 1)
					}
					testcloud.JSON(w, 200, `{"floatingips":[`+row+`]}`)
					return
				}
				rows := ""
				if scenario == "duplicate" || strings.HasPrefix(scenario, "late") || scenario == "query drift" {
					rows = attachIP("", "", "DOWN", "")
				}
				next := cloud.Server.URL + "/v2.0/floatingips?marker=last"
				if scenario == "query drift" {
					next += "&floating_ip_address=198.51.100.99"
				}
				if scenario == "empty HTTP page" {
					w.Header().Set("Link", "<"+next+">; rel=\"next\"")
					testcloud.JSON(w, 200, `{"floatingips":[]}`)
				} else {
					testcloud.JSON(w, 200, fmt.Sprintf(`{"floatingips":[%s],"floatingips_links":[{"rel":"next","href":%q}]}`, rows, next))
				}
			})
			cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
				ports.Add(1)
				testcloud.JSON(w, 200, `{"ports":[`+plannedPort+`]}`)
			})
			attachUnexpected(t, cloud)
			request := attachRequest()
			request.IP = resource.Name("198.51.100.10")
			plan, err := network.New(cloud.Client("network", "/v2.0")).FloatingIPs.PrepareAttach(context.Background(), request)
			if scenario == "last page" || scenario == "empty HTTP page" {
				if err != nil || plan.Selection().IPID != "fip" || ports.Load() != 1 || reads.Load() != 2 {
					t.Fatal(plan.Selection(), err, ports.Load(), reads.Load())
				}
				return
			}
			if err == nil || plan.Selection() != (network.FloatingIPAttachSelection{}) || ports.Load() != 0 {
				t.Fatal(plan.Selection(), err, ports.Load())
			}
			if scenario == "duplicate" && !errors.Is(err, resource.ErrAmbiguous) || (scenario == "missing" || scenario == "terminal204") && !errors.Is(err, resource.ErrNotFound) {
				t.Fatal(err)
			}
			if scenario == "query drift" && reads.Load() != 1 {
				t.Fatal("queried drifted continuation", reads.Load())
			}
			if scenario == "late403" {
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &proof) || proof.Actual != 403 || !strings.Contains(string(proof.Body), "late attach denied") {
					t.Fatal(err, proof)
				}
			}
			if scenario == "late malformed" {
				var syntax *json.SyntaxError
				var proof *resource.ResponseError
				if !errors.As(err, &syntax) || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != `{"floatingips":` || reads.Load() != 2 {
					t.Fatal(err, proof, reads.Load())
				}
			}
		})
	}
}

func TestFloatingIPAttachRechecksPinnedIPAndPortBeforeMutation(t *testing.T) {
	for _, field := range []string{"id", "address", "network", "owner", "association", "port server", "port fixed", "412"} {
		t.Run(field, func(t *testing.T) {
			cloud := testcloud.New(t)
			ensurePortFixture(t, cloud)
			var gets, puts atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/ports/port", func(w http.ResponseWriter, r *http.Request) {
				row := plannedPort
				if field == "port server" {
					row = strings.Replace(row, `"server"`, `"foreign"`, 1)
				} else if field == "port fixed" {
					row = strings.Replace(row, "10.0.0.10", "10.0.0.99", 1)
				}
				testcloud.JSON(w, 200, `{"port":`+row+`}`)
			})
			cloud.Mux.HandleFunc("GET /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
				row := attachIP("old-port", "10.1.0.20", "DOWN", `,"revision_number":0`)
				if gets.Add(1) > 1 {
					fromTo := map[string][2]string{"id": {`"fip"`, `"wrong"`}, "address": {"198.51.100.10", "198.51.100.99"}, "network": {`"external"`, `"wrong"`}, "owner": {`"foreign"`, `"other"`}, "association": {`"old-port"`, `"concurrent-port"`}}
					if change, ok := fromTo[field]; ok {
						row = strings.Replace(row, change[0], change[1], 1)
					}
				}
				testcloud.JSON(w, 200, `{"floatingip":`+row+`}`)
			})
			cloud.Mux.HandleFunc("PUT /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
				puts.Add(1)
				if r.Header.Get("If-Match") != "revision_number=0" {
					t.Error(r.Header)
				}
				http.Error(w, "revision conflict", 412)
			})
			attachUnexpected(t, cloud)
			result, err := network.New(cloud.Client("network", "/v2.0")).FloatingIPs.Attach(context.Background(), attachRequest())
			if err == nil || result == nil || !result.Reused || result.Allocated || result.FloatingIP.ID != "fip" || result.FloatingIP.PortID != "old-port" || result.FloatingIP.FloatingIP != "198.51.100.10" {
				t.Fatal(result, err)
			}
			want := int32(0)
			if field == "412" {
				want = 1
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &proof) || proof.Actual != 412 {
					t.Fatal(err)
				}
			}
			if puts.Load() != want {
				t.Fatal("unexpected mutation", puts.Load())
			}
		})
	}
}

func TestFloatingIPAttachAcceptedPUTRetainsOnlyExactValidatedModel(t *testing.T) {
	for _, scenario := range []string{"matching close", "matching read", "source", "cancel", "wrong address", "wrong ID", "wrong owner", "wrong fixed", "malformed"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			ensurePortFixture(t, cloud)
			plannedPortRead(t, cloud)
			var gets, writes atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 200, `{"floatingip":`+attachIP("old-port", "10.1.0.20", "DOWN", "")+`}`)
			})
			attachUnexpected(t, cloud)
			client := cloud.Client("network", "/v2.0")
			service := network.New(client)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("accepted attach processing error")
			body := `{"floatingip":` + attachIP("port", "10.0.0.10", "DOWN", "") + `}`
			change := map[string][2]string{"wrong address": {"198.51.100.10", "198.51.100.99"}, "wrong ID": {`"fip"`, `"wrong"`}, "wrong owner": {`"foreign"`, `"other"`}, "wrong fixed": {"10.0.0.10", "10.0.0.99"}}
			if replacement, ok := change[scenario]; ok {
				body = strings.Replace(body, replacement[0], replacement[1], 1)
			}
			if scenario == "malformed" {
				body = `{"floatingip":`
			}
			base := client.ProviderClient.HTTPClient.Transport
			client.ProviderClient.HTTPClient.Transport = planTransport(func(r *http.Request) (*http.Response, error) {
				if r.Method != "PUT" {
					return base.RoundTrip(r)
				}
				writes.Add(1)
				var responseBody = planCloseBody{Reader: strings.NewReader(body), close: func() error {
					if scenario == "source" {
						service.API = nil
					} else if scenario == "cancel" {
						cancel(cause)
					}
					return cause
				}}
				if scenario == "matching read" {
					responseBody.Reader = &acceptedIPReader{data: []byte(body), cause: cause}
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"X-Attach-Proof": {"accepted"}}, Body: responseBody, Request: r}, nil
			})
			retries := 0
			client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return cause
			}
			result, err := service.FloatingIPs.Attach(ctx, attachRequest(), network.WithAttachActive())
			var proof *resource.ResponseError
			if result == nil || result.FloatingIP == nil || !result.Reused || result.Allocated || !errors.Is(err, cause) || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Attach-Proof") != "accepted" || string(proof.Body) != body || writes.Load() != 1 || gets.Load() != 2 || retries != 0 {
				t.Fatal(result, err, proof, gets.Load(), writes.Load(), retries)
			}
			wantPort := "port"
			if strings.HasPrefix(scenario, "wrong") || scenario == "malformed" {
				wantPort = "old-port"
				if scenario == "malformed" {
					var syntax *json.SyntaxError
					if !errors.As(err, &syntax) {
						t.Fatal(err)
					}
				} else if !strings.Contains(err.Error(), map[string]string{"wrong address": "address changed", "wrong ID": "does not match", "wrong owner": "different or inconsistent project", "wrong fixed": "destination is"}[scenario]) {
					t.Fatal("missing validation error", err)
				}
			}
			if result.FloatingIP.PortID != wantPort || result.FloatingIP.FloatingIP != "198.51.100.10" {
				t.Fatal(result.FloatingIP)
			}
			if scenario == "source" && !errors.Is(err, resource.ErrInvalidOption) || scenario == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}

func TestFloatingIPAttachPreparedPolicyAndResultAreOwned(t *testing.T) {
	cloud := testcloud.New(t)
	plannedPortRead(t, cloud)
	var gets atomic.Int32
	cloud.Mux.HandleFunc("GET /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
		status := "DOWN"
		if gets.Add(1) == 4 {
			status = "ACTIVE"
		}
		testcloud.JSON(w, 200, `{"floatingip":`+attachIP("port", "10.0.0.10", status, "")+`}`)
	})
	attachUnexpected(t, cloud)
	progress := 0
	ensure, err := network.PrepareEnsureFloatingIPOptions(context.Background(), network.WithEnsurePort(resource.ID("port")), network.WithEnsureProject("irrelevant"), network.WithEnsureReuse(false), network.WithEnsureWait(resource.WithTimeout(time.Second), resource.WithPollInterval(time.Millisecond), resource.WithProgressCallback(func(int) { progress++ })))
	if err != nil {
		t.Fatal(err)
	}
	policy, err := network.PrepareAttachFloatingIPOptions(context.Background(), network.WithAttachDestinationPolicy(ensure), network.WithAttachNoWait())
	if err != nil {
		t.Fatal(err)
	}
	ips := network.New(cloud.Client("network", "/v2.0")).FloatingIPs
	plan, err := ips.PrepareAttach(context.Background(), attachRequest(), network.WithAttachFloatingIPPolicy(policy), network.WithAttachActive())
	if err != nil {
		t.Fatal(err)
	}
	result, err := ips.AttachPrepared(context.Background(), plan)
	if err != nil || result.FloatingIP.Status != "ACTIVE" || progress != 1 || gets.Load() != 4 {
		t.Fatal(result, err, progress, gets.Load())
	}
	result.FloatingIP.Tags[0] = "caller mutation"
	result, err = ips.Attach(context.Background(), attachRequest(), network.WithAttachFloatingIPPolicy(policy))
	if err != nil || result.FloatingIP.Status != "DOWN" || result.FloatingIP.Tags[0] != "original" || gets.Load() != 6 || progress != 1 {
		t.Fatal(result, err, gets.Load(), progress)
	}
}

func TestFloatingIPAttachWaitRequiresActualStatusAndPinnedAddress(t *testing.T) {
	for _, scenario := range []string{"custom status", "changed address", "timeout", "ERROR"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			ensurePortFixture(t, cloud)
			plannedPortRead(t, cloud)
			var gets atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
				row := attachIP("port", "10.0.0.10", "DOWN", "")
				row = strings.Replace(row, `"tags":`, `"description":"ACTIVE","tags":`, 1)
				if gets.Add(1) > 2 {
					switch scenario {
					case "changed address":
						row = strings.Replace(row, "198.51.100.10", "198.51.100.99", 1)
					case "ERROR":
						row = strings.Replace(row, "DOWN", "ERROR", 1)
					}
				}
				testcloud.JSON(w, 200, `{"floatingip":`+row+`}`)
			})
			attachUnexpected(t, cloud)
			timeout := time.Second
			if scenario == "timeout" {
				timeout = 20 * time.Millisecond
			}
			wait := []resource.WaitOption{resource.WithTimeout(timeout), resource.WithPollInterval(time.Millisecond)}
			if scenario == "custom status" {
				wait = append(wait, resource.WithStatusAttribute("description"))
			}
			result, err := network.New(cloud.Client("network", "/v2.0")).FloatingIPs.Attach(context.Background(), attachRequest(), network.WithAttachWait(wait...))
			if err == nil || result == nil || result.FloatingIP.FloatingIP != "198.51.100.10" || result.FloatingIP.Status != "DOWN" || result.Allocated || !result.Reused || gets.Load() < 3 {
				t.Fatal(result, err, gets.Load())
			}
			if scenario == "timeout" && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal(err)
			}
			if scenario == "changed address" {
				var proof *resource.ResponseError
				if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || !strings.Contains(string(proof.Body), "198.51.100.99") || gets.Load() != 3 {
					t.Fatal(err, proof, gets.Load())
				}
			}
			if scenario == "ERROR" {
				var failed *resource.FailedStateError
				if !errors.As(err, &failed) || failed.Status != "ERROR" || gets.Load() != 3 {
					t.Fatal(err, failed, gets.Load())
				}
			}
			if scenario == "custom status" && (!errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(err.Error(), `status is "DOWN"`) || gets.Load() != 3) {
				t.Fatal(err, gets.Load())
			}
		})
	}
}
