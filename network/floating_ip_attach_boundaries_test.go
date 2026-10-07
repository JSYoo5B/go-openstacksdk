package network_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestFloatingIPAttachPreflightRejectsInvalidInputAndPlans(t *testing.T) {
	cloud := testcloud.New(t)
	attachUnexpected(t, cloud)
	ips := network.New(cloud.Client("network", "/v2.0")).FloatingIPs
	for _, options := range [][]network.AttachFloatingIPOption{
		{nil}, {network.WithAttachFloatingIPPolicy(network.AttachFloatingIPPolicy{})},
		{network.WithAttachDestinationPolicy(network.EnsureFloatingIPPolicy{})},
		{network.WithAttachWait(resource.WithTimeout(-time.Second)), network.WithAttachNoWait()},
		{network.WithAttachFixedAddress("2001:db8::1")}, {network.WithAttachPort(resource.ID("../port"))},
	} {
		result, err := ips.Attach(context.Background(), attachRequest(), options...)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(result, err)
		}
	}
	if result, err := ips.Attach(context.Background(), attachRequest(), network.WithAttachWait(resource.WithStatusAttribute("unknown_field"))); result != nil || !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(result, err)
	}
	for _, request := range []network.AttachFloatingIPRequest{
		{}, {Server: resource.ID("server"), IP: resource.Name("2001:db8::1")},
		{Server: resource.ID("server"), IP: resource.Name("fip")}, {Server: resource.ID("bad/id"), IP: resource.ID("fip")},
	} {
		plan, err := ips.PrepareAttach(context.Background(), request)
		if !errors.Is(err, resource.ErrInvalidOption) || plan.Selection() != (network.FloatingIPAttachSelection{}) {
			t.Fatal(plan.Selection(), err)
		}
		if result, err := ips.AttachPrepared(context.Background(), plan); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(result, err)
		}
	}
	if result, err := ips.Attach(nil, attachRequest()); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(result, err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("explicit attachment canceled")
	cancel(cause)
	if result, err := ips.Attach(ctx, attachRequest()); result != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal(result, err)
	}
	if result, err := ips.AttachPrepared(context.Background(), network.FloatingIPAttachPlan{}); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(result, err)
	}
}

func TestFloatingIPAttachInvalidSelectedModelsRetainHTTPProof(t *testing.T) {
	for _, byAddress := range []bool{false, true} {
		for _, field := range []string{"revision", "owner aliases", "network", "ID"} {
			t.Run(fmt.Sprintf("address=%v/%s", byAddress, field), func(t *testing.T) {
				cloud := testcloud.New(t)
				row := attachIP("", "", "DOWN", `,"revision_number":-1`)
				if field != "revision" {
					row = attachIP("", "", "DOWN", "")
				}
				switch field {
				case "owner aliases":
					row = strings.Replace(row, `"project_id":"foreign"`, `"project_id":"foreign","tenant_id":"other"`, 1)
				case "network":
					row = strings.Replace(row, `"external"`, `"bad/network"`, 1)
				case "ID":
					row = strings.Replace(row, `"fip"`, `"bad/id"`, 1)
				}
				path, key := "/v2.0/floatingips/fip", "floatingip"
				if byAddress {
					path, key = "/v2.0/floatingips", "floatingips"
					row = "[" + row + "]"
				}
				body := fmt.Sprintf(`{%q:%s}`, key, row)
				cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Selection-Proof", "invalid-existing")
					testcloud.JSON(w, 200, body)
				})
				attachUnexpected(t, cloud)
				request := attachRequest()
				if byAddress {
					request.IP = resource.Name("198.51.100.10")
				}
				plan, err := network.New(cloud.Client("network", "/v2.0")).FloatingIPs.PrepareAttach(context.Background(), request)
				var proof *resource.ResponseError
				if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Selection-Proof") != "invalid-existing" || string(proof.Body) != body || plan.Selection() != (network.FloatingIPAttachSelection{}) {
					t.Fatal(plan.Selection(), err, proof)
				}
			})
		}
	}
}

func TestFloatingIPAttachPlanSnapshotSurvivesCallerMutationAndRejectsForeignService(t *testing.T) {
	for _, tags := range []string{`["original"]`, `[]`, `null`} {
		t.Run(tags, func(t *testing.T) {
			cloud := testcloud.New(t)
			ensurePortFixture(t, cloud)
			cloud.Mux.HandleFunc("GET /v2.0/ports/port", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"port":`+strings.Replace(plannedPort, `"server"`, `"changed"`, 1)+`}`)
			})
			var gets atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				row := strings.Replace(attachIP("", "", "DOWN", `,"revision_number":0`), `["original"]`, tags, 1)
				testcloud.JSON(w, 200, `{"floatingip":`+row+`}`)
			})
			attachUnexpected(t, cloud)
			client := cloud.Client("network", "/v2.0")
			ips := network.New(client).FloatingIPs
			plan, err := ips.PrepareAttach(context.Background(), attachRequest())
			if err != nil {
				t.Fatal(err)
			}
			if result, err := network.New(client).FloatingIPs.AttachPrepared(context.Background(), plan); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(result, err)
			}
			first, err := ips.AttachPrepared(context.Background(), plan)
			if err == nil || first == nil || first.FloatingIP == nil {
				t.Fatal(first, err)
			}
			first.FloatingIP.ID, first.FloatingIP.FloatingIP = "caller", "198.51.100.99"
			if len(first.FloatingIP.Tags) > 0 {
				first.FloatingIP.Tags[0] = "caller"
			}
			second, err := ips.AttachPrepared(context.Background(), plan)
			if err == nil || second == nil || second.FloatingIP.ID != "fip" || second.FloatingIP.FloatingIP != "198.51.100.10" || gets.Load() != 1 {
				t.Fatal(second, err, gets.Load())
			}
			if tags == `["original"]` && second.FloatingIP.Tags[0] != "original" || tags == `[]` && second.FloatingIP.Tags == nil || tags == `null` && second.FloatingIP.Tags != nil {
				t.Fatal("plan tags changed", second.FloatingIP.Tags)
			}
		})
	}
}

func TestFloatingIPAttachGuardsSourceCloseAndPhysicalRetries(t *testing.T) {
	for _, scenario := range []string{"source before execution", "selection close", "retry revision", "outer retry", "header override"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			ensurePortFixture(t, cloud)
			plannedPortRead(t, cloud)
			var puts atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"floatingip":`+attachIP("", "", "DOWN", `,"revision_number":0`)+`}`)
			})
			cloud.Mux.HandleFunc("PUT /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
				puts.Add(1)
				http.Error(w, "original attachment PUT503", 503)
			})
			attachUnexpected(t, cloud)
			client := cloud.Client("network", "/v2.0")
			service := network.New(client)
			changed := false
			ctx := rest.WithOperationGuard(context.Background(), func(context.Context) error {
				if changed {
					return fmt.Errorf("%w: outer source changed", resource.ErrInvalidOption)
				}
				return nil
			})
			if scenario == "selection close" {
				base := client.ProviderClient.HTTPClient.Transport
				client.ProviderClient.HTTPClient.Transport = planTransport(func(r *http.Request) (*http.Response, error) {
					response, err := base.RoundTrip(r)
					if err == nil && r.URL.Path == "/v2.0/floatingips/fip" {
						original := response.Body
						response.Body = planCloseBody{Reader: original, close: func() error { service.API = nil; return original.Close() }}
					}
					return response, err
				})
			}
			plan, err := service.FloatingIPs.PrepareAttach(ctx, attachRequest())
			if scenario == "selection close" {
				var proof *resource.ResponseError
				if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || plan.Selection() != (network.FloatingIPAttachSelection{}) || puts.Load() != 0 {
					t.Fatal(plan.Selection(), err, proof, puts.Load())
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "source before execution" {
				original := client.Endpoint
				client.Endpoint = cloud.Server.URL + "/changed/"
				result, err := service.FloatingIPs.AttachPrepared(ctx, plan)
				client.Endpoint = original
				_, sticky := service.FloatingIPs.AttachPrepared(ctx, plan)
				if result == nil || result.FloatingIP.ID != "fip" || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(sticky, resource.ErrInvalidOption) || puts.Load() != 0 {
					t.Fatal(result, err, sticky, puts.Load())
				}
				return
			}
			if scenario == "header override" {
				client.MoreHeaders = map[string]string{"If-Match": "revision_number=7"}
			} else {
				client.ProviderClient.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, _ error, _ uint) error {
					if scenario == "outer retry" {
						changed = true
					} else {
						options.MoreHeaders["If-Match"] = "revision_number=7"
					}
					return nil
				}
			}
			result, err := service.FloatingIPs.AttachPrepared(ctx, plan)
			wantPuts := int32(1)
			if scenario == "header override" {
				wantPuts = 0
			}
			if !errors.Is(err, resource.ErrInvalidOption) || result == nil || result.FloatingIP.PortID != "" || result.Allocated || !result.Reused || puts.Load() != wantPuts {
				t.Fatal(result, err, puts.Load())
			}
			if scenario != "header override" {
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &proof) || proof.Actual != 503 || !strings.Contains(string(proof.Body), "original attachment PUT503") {
					t.Fatal(err, proof)
				}
			}
		})
	}
}
