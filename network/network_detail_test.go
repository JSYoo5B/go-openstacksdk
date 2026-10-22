package network_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const networkDetailBody = `{"network":{"id":"n1","name":"ext","admin_state_up":true,"router:external":true,"port_security_enabled":false,` +
	`"provider:network_type":"vlan","provider:physical_network":"physnet1","provider:segmentation_id":"vlan1","mtu":1450,` +
	`"dns_domain":"example.org.","is_default":null,"qos_policy_id":"q1","availability_zones":["az1"],` +
	`"segments":[{"provider:network_type":"flat"}],"vlan_transparent":false,"x_vendor":{"k":[1]}}}`

func requireNetworkDetail(t *testing.T, detail *network.NetworkDetail, status int) {
	t.Helper()
	if detail == nil || detail.ID != "n1" || detail.Name != "ext" || !detail.AdminStateUp {
		t.Fatalf("native fields: %+v", detail)
	}
	if detail.RouterExternal == nil || !*detail.RouterExternal || detail.PortSecurityEnabled == nil || *detail.PortSecurityEnabled ||
		detail.ProviderNetworkType == nil || *detail.ProviderNetworkType != "vlan" || detail.ProviderPhysicalNetwork == nil || *detail.ProviderPhysicalNetwork != "physnet1" ||
		string(detail.ProviderSegmentationID) != `"vlan1"` || detail.MTU == nil || *detail.MTU != 1450 || detail.DNSDomain == nil || *detail.DNSDomain != "example.org." ||
		detail.IsDefault != nil || detail.QoSPolicyID == nil || *detail.QoSPolicyID != "q1" || len(detail.AvailabilityZones) != 1 ||
		string(detail.Segments) != `[{"provider:network_type":"flat"}]` || detail.VLANTransparent == nil || *detail.VLANTransparent || detail.VLANQinQ != nil {
		t.Fatalf("extension fields: %+v", detail)
	}
	if string(detail.Metadata.Body["x_vendor"]) != `{"k":[1]}` || string(detail.Metadata.Body["is_default"]) != "null" || detail.Metadata.StatusCode != status || detail.Metadata.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("metadata: %+v", detail.Metadata)
	}
}

// TestNetworkDetailCreateExposesExtensionProperties shows that the owned model
// carries router:external/provider/MTU/DNS/port-security values from the single
// accepted create response, with unknown members and response metadata.
func TestNetworkDetailCreateExposesExtensionProperties(t *testing.T) {
	cloud := testcloud.New(t)
	var inventory, posts atomic.Int32
	mutationInventory(cloud, &inventory)
	cloud.Mux.HandleFunc("POST /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		body := networkMutationBody(t, r)
		if body["router:external"] != true || body["mtu"] != float64(1450) || body["dns_domain"] != "example.org." || body["port_security_enabled"] != false {
			t.Errorf("body=%v", body)
		}
		testcloud.JSON(w, http.StatusCreated, networkDetailBody)
	})
	s := mutationService(t, cloud)
	requireMutationCache(t, s, &inventory, 1)
	detail, err := s.CreateNetworkDetail(context.Background(), network.CreateNetworkRequest{Name: "ext"},
		network.WithNetworkExternal(true), network.WithNetworkMTU(1450), network.WithNetworkDNSDomain("example.org."), network.WithNetworkPortSecurity(false))
	if err != nil || posts.Load() != 1 {
		t.Fatal(detail, err, posts.Load())
	}
	requireNetworkDetail(t, detail, http.StatusCreated)
	requireMutationCache(t, s, &inventory, 2)
}

// TestNetworkDetailPythonDefaultsAndDecodeBoundary keeps Python's absent=false
// defaults for router:external/port_security_enabled, and reports a known
// extension type mismatch after the accepted create without a resend.
func TestNetworkDetailPythonDefaultsAndDecodeBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		ok         bool
	}{
		{"absent-defaults", `{"network":{"id":"n1","name":"x"}}`, true},
		{"null-stays-nil", `{"network":{"id":"n1","name":"x","router:external":null,"port_security_enabled":null,"mtu":null}}`, true},
		{"mtu-string", `{"network":{"id":"n1","name":"x","mtu":"1500"}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var inventory, posts atomic.Int32
			mutationInventory(cloud, &inventory)
			cloud.Mux.HandleFunc("POST /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				testcloud.JSON(w, http.StatusCreated, tc.body)
			})
			s := mutationService(t, cloud)
			detail, err := s.CreateNetworkDetail(context.Background(), network.CreateNetworkRequest{Name: "x"})
			if posts.Load() != 1 {
				t.Fatal(posts.Load())
			}
			if !tc.ok {
				var responseErr *resource.ResponseError
				if err == nil || detail != nil || !errors.As(err, &responseErr) || responseErr.StatusCode != http.StatusCreated {
					t.Fatal(detail, err)
				}
				// The native facade keeps its established contract for the same body.
				native, nativeErr := s.CreateNetwork(context.Background(), network.CreateNetworkRequest{Name: "x"})
				if nativeErr != nil || native == nil || native.ID != "n1" {
					t.Fatal(native, nativeErr)
				}
				return
			}
			if err != nil || detail == nil || detail.MTU != nil {
				t.Fatal(detail, err)
			}
			absent := tc.name == "absent-defaults"
			if absent != (detail.RouterExternal != nil && !*detail.RouterExternal) || absent != (detail.PortSecurityEnabled != nil && !*detail.PortSecurityEnabled) {
				t.Fatalf("%+v", detail)
			}
			if !absent && (detail.RouterExternal != nil || detail.PortSecurityEnabled != nil) {
				t.Fatalf("%+v", detail)
			}
		})
	}
}

// TestNetworkDetailUpdateMutationAndOptionlessLookup covers the PUT projection
// and the optionless branch, which returns the GET row with every member.
func TestNetworkDetailUpdateMutationAndOptionlessLookup(t *testing.T) {
	cloud := testcloud.New(t)
	var inventory, puts, gets atomic.Int32
	mutationInventory(cloud, &inventory)
	cloud.Mux.HandleFunc("PUT /v2.0/networks/n1", func(w http.ResponseWriter, r *http.Request) {
		puts.Add(1)
		if body := networkMutationBody(t, r); body["router:external"] != true || body["mtu"] != float64(1450) {
			t.Errorf("body=%v", body)
		}
		testcloud.JSON(w, http.StatusOK, networkDetailBody)
	})
	cloud.Mux.HandleFunc("GET /v2.0/networks/n1", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		testcloud.JSON(w, http.StatusOK, networkDetailBody)
	})
	s := mutationService(t, cloud)
	requireMutationCache(t, s, &inventory, 1)
	detail, err := s.UpdateNetworkDetail(context.Background(), resource.ID("n1"), network.WithNetworkExternal(true), network.WithNetworkMTU(1450))
	if err != nil || puts.Load() != 1 || gets.Load() != 0 {
		t.Fatal(detail, err, puts.Load(), gets.Load())
	}
	requireNetworkDetail(t, detail, http.StatusOK)
	requireMutationCache(t, s, &inventory, 2)

	detail, err = s.UpdateNetworkDetail(context.Background(), resource.ID("n1"))
	if err != nil || puts.Load() != 1 || gets.Load() != 1 {
		t.Fatal(detail, err, puts.Load(), gets.Load())
	}
	requireNetworkDetail(t, detail, http.StatusOK)
	requireMutationCache(t, s, &inventory, 3)

	cloud.Mux.HandleFunc("PUT /v2.0/networks/gone", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, http.StatusNotFound, `{"NeutronError":{}}`)
	})
	var missing *resource.NotFoundError
	if detail, err = s.UpdateNetworkDetail(context.Background(), resource.ID("gone"), network.WithNetworkName("x")); detail != nil || !errors.As(err, &missing) {
		t.Fatal(detail, err)
	}
}
