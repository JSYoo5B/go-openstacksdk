package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type acceptedIPReader struct {
	data  []byte
	cause error
}

func (r *acceptedIPReader) Read(dst []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(dst, r.data)
	r.data = r.data[n:]
	if len(r.data) == 0 {
		return n, r.cause
	}
	return n, nil
}

func TestFloatingIPAcceptedAllocationKeepsDecodedModelAndProcessingCauses(t *testing.T) {
	for _, code := range []int{201, 202} {
		for _, scenario := range []string{"close", "read", "source", "cancel", "owner close"} {
			t.Run(fmt.Sprintf("%d/%s", code, scenario), func(t *testing.T) {
				cloud := testcloud.New(t)
				ensurePortFixture(t, cloud)
				plannedPortRead(t, cloud)
				client := cloud.Client("network", "/v2.0")
				service := network.New(client)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("accepted allocation processing failed")
				var writes atomic.Int32
				body := `{"floatingip":{"id":"fip","project_id":"owner","floating_network_id":"external","floating_ip_address":"198.51.100.10","port_id":"port","fixed_ip_address":"10.0.0.10","status":"DOWN"}}`
				if scenario == "owner close" {
					body = strings.Replace(body, `"project_id":"owner"`, `"project_id":"foreign"`, 1)
				}
				base := client.ProviderClient.HTTPClient.Transport
				client.ProviderClient.HTTPClient.Transport = planTransport(func(r *http.Request) (*http.Response, error) {
					if r.Method != "POST" {
						return base.RoundTrip(r)
					}
					writes.Add(1)
					var reader io.Reader = strings.NewReader(body)
					if scenario == "read" {
						reader = &acceptedIPReader{data: []byte(body), cause: cause}
					}
					closer := planCloseBody{Reader: reader, close: func() error {
						switch scenario {
						case "close", "owner close":
							return cause
						case "source":
							service.API = nil
						case "cancel":
							cancel(cause)
						}
						return nil
					}}
					return &http.Response{StatusCode: code, Header: http.Header{"X-Allocation-Proof": {"accepted"}}, Body: closer, Request: r}, nil
				})
				retries := 0
				client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return cause
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					t.Error("unexpected followup", r.Method, r.URL)
					http.Error(w, "unexpected", 500)
				})
				plan, err := service.FloatingIPs.PrepareEnsure(ctx, ensureFloatingRequest(), network.WithEnsureReuse(false), network.WithEnsureProject("owner"), network.WithEnsureActive())
				if err != nil {
					t.Fatal(err)
				}
				result, err := service.FloatingIPs.EnsurePrepared(ctx, plan)
				var proof *resource.ResponseError
				if result == nil || !result.Allocated || result.Reused || result.FloatingIP == nil || result.FloatingIP.ID != "fip" || result.FloatingIP.PortID != "port" || result.FloatingIP.Status != "DOWN" || !errors.As(err, &proof) || proof.StatusCode != code || proof.Header.Get("X-Allocation-Proof") != "accepted" || string(proof.Body) != body || writes.Load() != 1 || retries != 0 {
					t.Fatal(result, err, proof, writes.Load(), retries)
				}
				if scenario == "source" {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, cause) {
					t.Fatal(err)
				}
				if scenario == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				if scenario == "owner close" && (result.FloatingIP.ProjectID != "foreign" || !strings.Contains(err.Error(), "different or inconsistent project")) {
					t.Fatal("lost decoded allocation evidence or joined owner validation", result, err)
				}
			})
		}
	}
}

func TestFloatingIPAcceptedAssociationAdoptsOnlyMatchingModelOnCloseError(t *testing.T) {
	for _, scenario := range []string{"matching", "wrong ID", "wrong fixed"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			ensurePortFixture(t, cloud)
			plannedPortRead(t, cloud)
			cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"floatingips":[{"id":"free","project_id":"owner","floating_network_id":"external","floating_ip_address":"198.51.100.10","revision_number":0}]}`)
			})
			client := cloud.Client("network", "/v2.0")
			cause := errors.New("association Close failed")
			var writes atomic.Int32
			base := client.ProviderClient.HTTPClient.Transport
			client.ProviderClient.HTTPClient.Transport = planTransport(func(r *http.Request) (*http.Response, error) {
				if r.Method != "PUT" {
					return base.RoundTrip(r)
				}
				writes.Add(1)
				if r.URL.Path != "/v2.0/floatingips/free" || r.Header.Get("If-Match") != "revision_number=0" {
					t.Error(r.URL, r.Header)
				}
				id, fixed := "free", "10.0.0.10"
				if scenario == "wrong ID" {
					id = "other"
				}
				if scenario == "wrong fixed" {
					fixed = "10.0.0.99"
				}
				body := fmt.Sprintf(`{"floatingip":{"id":%q,"project_id":"owner","floating_network_id":"external","floating_ip_address":"198.51.100.10","port_id":"port","fixed_ip_address":%q,"status":"DOWN"}}`, id, fixed)
				return &http.Response{StatusCode: 200, Header: http.Header{"X-Association-Proof": {"accepted"}}, Body: planCloseBody{Reader: strings.NewReader(body), close: func() error { return cause }}, Request: r}, nil
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Error("unexpected allocation/wait/cleanup", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			ips := network.New(client).FloatingIPs
			plan, err := ips.PrepareEnsure(context.Background(), ensureFloatingRequest(), network.WithEnsureProject("owner"), network.WithEnsureActive())
			if err != nil {
				t.Fatal(err)
			}
			result, err := ips.EnsurePrepared(context.Background(), plan)
			var proof *resource.ResponseError
			if !errors.Is(err, cause) || result == nil || !result.Reused || result.Allocated || result.FloatingIP == nil || result.FloatingIP.ID != "free" || writes.Load() != 1 || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Association-Proof") != "accepted" {
				t.Fatal(result, err, proof, writes.Load())
			}
			want := ""
			if scenario == "matching" {
				want = "port"
			}
			if result.FloatingIP.PortID != want {
				t.Fatal("adopted invalid response or lost matching body", result, err)
			}
			if scenario == "wrong ID" && !strings.Contains(err.Error(), "does not match floating IP") || scenario == "wrong fixed" && !strings.Contains(err.Error(), "destination is") {
				t.Fatal("lost joined assignment validation", err)
			}
		})
	}
}

func TestFloatingIPAcceptedReadAndDecodeErrorsRemainJoined(t *testing.T) {
	cloud := testcloud.New(t)
	ensurePortFixture(t, cloud)
	plannedPortRead(t, cloud)
	client := cloud.Client("network", "/v2.0")
	readCause, closeCause := errors.New("truncated read"), errors.New("failed Close")
	var writes atomic.Int32
	base := client.ProviderClient.HTTPClient.Transport
	client.ProviderClient.HTTPClient.Transport = planTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method != "POST" {
			return base.RoundTrip(r)
		}
		writes.Add(1)
		return &http.Response{StatusCode: 202, Header: http.Header{"X-Allocation-Proof": {"partial"}}, Body: planCloseBody{Reader: &acceptedIPReader{data: []byte(`{"floatingip":`), cause: readCause}, close: func() error { return closeCause }}, Request: r}, nil
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	ips := network.New(client).FloatingIPs
	plan, err := ips.PrepareEnsure(context.Background(), ensureFloatingRequest(), network.WithEnsureReuse(false))
	if err != nil {
		t.Fatal(err)
	}
	result, err := ips.EnsurePrepared(context.Background(), plan)
	var proof *resource.ResponseError
	var decode *json.SyntaxError
	if result == nil || !result.Allocated || result.FloatingIP != nil || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.As(err, &decode) || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != `{"floatingip":` || writes.Load() != 1 {
		t.Fatal(result, err, proof, writes.Load())
	}
}
