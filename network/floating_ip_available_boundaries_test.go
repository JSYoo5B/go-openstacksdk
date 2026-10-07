package network_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func beforeAvailableOption[T any](before func(), next func(*T) error) func(*T) error {
	return func(o *T) error { before(); return next(o) }
}

func TestFloatingIPAvailableOptionsCannotReplaceCapturedClient(t *testing.T) {
	for _, scenario := range []string{"endpoint", "provider", "version", "fixed IPv6", "fixed invalid"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("network", "/v2.0/")
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = planTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				return nil, errors.New("must stop before HTTP")
			})
			service := network.New(client)
			option := beforeAvailableOption(func() {
				switch scenario {
				case "endpoint":
					client.Endpoint += "changed/"
				case "provider":
					client.ProviderClient = nil
				case "version":
					client.Microversion = "changed"
				}
			}, network.WithAvailableProject("owner"))
			if scenario == "fixed IPv6" {
				option = network.WithAvailableFixedAddress("2001:db8::1")
			}
			if scenario == "fixed invalid" {
				option = network.WithAvailableFixedAddress("not an IP")
			}
			result, err := service.FloatingIPs.Available(context.Background(), availableRequest(resource.Ref{}), option)
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestFloatingIPAvailableAllocationMismatchKeepsKnownModel(t *testing.T) {
	cloud := testcloud.New(t)
	availableRoles(t, cloud)
	var posts atomic.Int32
	cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"floatingips":[]}`) })
	cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		w.Header().Set("X-Known", "wrong-network")
		testcloud.JSON(w, 201, `{"floatingip":{"id":"allocated","floating_network_id":"unexpected","floating_ip_address":"198.51.100.10","port_id":null}}`)
	})
	result, err := network.New(cloud.Client("network", "/v2.0/")).FloatingIPs.Available(context.Background(), availableRequest(resource.Ref{}))
	var proof *resource.ResponseError
	if err == nil || result == nil || !result.Allocated || result.Reused || result.FloatingIP.ID != "allocated" || result.FloatingIP.FloatingNetworkID != "unexpected" || string(result.Metadata.Body["floating_network_id"]) != `"unexpected"` || !errors.As(err, &proof) || proof.StatusCode != 201 || proof.Header.Get("X-Known") != "wrong-network" || result.AllocationResponse.StatusCode != 201 || posts.Load() != 1 {
		t.Fatal(result, err, proof, posts.Load())
	}
}

func TestFloatingIPAvailableMultiplePortsRequireNATAndIPv4(t *testing.T) {
	for _, scenario := range []string{"no NAT", "no IPv4"} {
		cloud := testcloud.New(t)
		availableRoles(t, cloud)
		var posts atomic.Int32
		cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"floatingips":[]}`) })
		cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
			body := `{"ports":[{"id":"a","device_id":"server","network_id":"private","fixed_ips":[{"ip_address":"2001:db8::1"}]}]}`
			if scenario == "no NAT" {
				body = `{"ports":[{"id":"a","device_id":"server","network_id":"private","fixed_ips":[{"ip_address":"10.0.0.10"}]},{"id":"b","device_id":"server","network_id":"private","fixed_ips":[{"ip_address":"10.0.0.11"}]}]}`
			}
			testcloud.JSON(w, 200, body)
		})
		cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
			posts.Add(1)
			t.Error("ambiguous/invalid allocation")
			w.WriteHeader(500)
		})
		result, err := network.New(cloud.Client("network", "/v2.0/")).FloatingIPs.Available(context.Background(), availableRequest(resource.ID("server")))
		if result != nil || err == nil || posts.Load() != 0 || (scenario == "no NAT" && !errors.Is(err, resource.ErrAmbiguous)) {
			t.Fatal(result, err, posts.Load())
		}
	}
}
