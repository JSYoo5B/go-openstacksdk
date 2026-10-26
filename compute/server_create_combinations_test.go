package compute_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// createCapture records the single Nova POST of a cloud create_server path.
type createCapture struct {
	mu      sync.Mutex
	posts   int
	body    map[string]any
	version string
}

func captureServerCreate(t *testing.T, cloud *testcloud.Cloud, path, reply string) *createCapture {
	t.Helper()
	capture := &createCapture{}
	cloud.Mux.HandleFunc("POST "+path, func(w http.ResponseWriter, r *http.Request) {
		var request struct{ Server map[string]any }
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		capture.mu.Lock()
		capture.posts++
		capture.body, capture.version = request.Server, r.Header.Get("X-OpenStack-Nova-API-Version")
		capture.mu.Unlock()
		testcloud.JSON(w, 202, reply)
	})
	return capture
}

func (c *createCapture) snapshot() (int, map[string]any, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.posts, c.body, c.version
}

// The cloud create_server path combines boot source, data volumes and NIC
// selection in one POST: explicit NICs and a default network become networks
// rows, and without either Python replaces networks with auto only when
// supports_microversion(compute, "2.37") holds for the advertised range.
func TestAutomaticCreateCombinesBootVolumesAndNetworkSelections(t *testing.T) {
	imageMapping := map[string]any{"source_type": "image", "destination_type": "local", "uuid": "image", "boot_index": float64(0), "delete_on_termination": true}
	dataMapping := map[string]any{"source_type": "volume", "destination_type": "volume", "uuid": "data-volume", "boot_index": float64(-1), "delete_on_termination": false}
	for _, tc := range []struct {
		name           string
		maximum        string
		withoutImage   bool
		defaultNetwork resource.Ref
		options        []compute.CreateServerOption
		networks       any
		mappings       []any
		imageRef       string
		version        string
		discoveries    int32
	}{
		{
			name: "image with explicit NIC and data volume", maximum: "2.100",
			options:  []compute.CreateServerOption{compute.WithNetworks(resource.ID("private")), compute.WithDataVolumes(resource.ID("data-volume"))},
			networks: []any{map[string]any{"uuid": "private"}}, mappings: []any{imageMapping, dataMapping}, imageRef: "image",
		},
		{
			name: "existing boot volume with fixed IP and port NICs", maximum: "2.100", withoutImage: true,
			options: []compute.CreateServerOption{compute.WithBootVolume(resource.ID("boot-volume")), compute.WithNetworkInterfaces(
				compute.ServerNetworkInterface{Network: resource.ID("private"), FixedIP: "10.0.0.10"}, compute.ServerNetworkInterface{Port: resource.ID("port")})},
			networks: []any{map[string]any{"uuid": "private", "fixed_ip": "10.0.0.10"}, map[string]any{"port": "port"}},
			mappings: []any{map[string]any{"source_type": "volume", "destination_type": "volume", "uuid": "boot-volume", "boot_index": float64(0), "delete_on_termination": false}},
		},
		{
			name: "new boot volume with default network", maximum: "2.100", defaultNetwork: resource.ID("default-network"),
			options:  []compute.CreateServerOption{compute.WithBootVolumeSize(10), compute.WithDataVolumes(resource.ID("data-volume"))},
			networks: []any{map[string]any{"uuid": "default-network"}},
			mappings: []any{map[string]any{"source_type": "image", "destination_type": "volume", "uuid": "image", "boot_index": float64(0), "delete_on_termination": false, "volume_size": float64(10)}, dataMapping},
		},
		{
			name: "new boot volume without NICs at advertised 2.37", maximum: "2.100",
			options:  []compute.CreateServerOption{compute.WithBootVolumeSize(10)},
			networks: "auto", version: "2.37", discoveries: 1,
			mappings: []any{map[string]any{"source_type": "image", "destination_type": "volume", "uuid": "image", "boot_index": float64(0), "delete_on_termination": false, "volume_size": float64(10)}},
		},
		{
			name: "image without NICs below advertised 2.37", maximum: "2.36",
			imageRef: "image", discoveries: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newDispatchFixture(t, "")
			discoveries := f.cloud.AdvertiseNova("/v2.1/", "2.1", tc.maximum)
			dependencies := compute.Dependencies{AddressNetworks: func(context.Context) (*network.Service, error) { return f.network, nil }}
			if tc.defaultNetwork != (resource.Ref{}) {
				dependencies.DefaultNetwork = func(context.Context) (resource.Ref, error) { return tc.defaultNetwork, nil }
			}
			f.service = compute.New(f.cloud.Client("compute", "/v2.1"), dependencies)
			capture := captureServerCreate(t, f.cloud, "/v2.1/servers", `{"server":{"id":"server","adminPass":"initial"}}`)
			request := automaticCreateRequest()
			if tc.withoutImage {
				request.Image = resource.Ref{}
			}
			options := append(slices.Clone(tc.options), compute.WithWait(resource.WithPollInterval(time.Millisecond)))
			result, err := f.service.CreateWithAutomaticFloatingIP(context.Background(), request, compute.AutomaticServerCreateOptions{
				Server: options, AutomaticIP: append(dispatchOptions(), compute.WithAutomaticIPEnabled(false)),
			})
			if err != nil || result == nil || result.Automatic == nil || result.Automatic.Decision.Reason != compute.AutomaticIPDisabled || result.Creation.AdminPass != "initial" {
				t.Fatal(result, err, f.trace())
			}
			posts, body, version := capture.snapshot()
			networks, present := body["networks"]
			if posts != 1 || present != (tc.networks != nil) || !reflect.DeepEqual(networks, tc.networks) || version != tc.version || body["imageRef"] != tc.imageRef {
				t.Fatalf("posts=%d networks=%#v version=%q body=%#v", posts, networks, version, body)
			}
			mappings, present := body["block_device_mapping_v2"]
			if present != (tc.mappings != nil) || tc.mappings != nil && !reflect.DeepEqual(mappings, tc.mappings) {
				t.Fatalf("mappings=%#v want=%#v", mappings, tc.mappings)
			}
			if discoveries.Load() != tc.discoveries || len(f.trace()) != 1 || f.trace()[0] != "raw:" || f.posts.Load() != 0 {
				t.Fatal(discoveries.Load(), f.trace())
			}
		})
	}
}

// Python's add_ips_to_server checks ip_pool, then ips, then auto_ip. The
// same precedence holds through the creation path, after a networks=auto POST
// (Neutron fixture) or a POST without networks at a pinned pre-2.37 client
// (legacy Nova fixture).
func TestAutomaticCreateIPLanesFollowPythonPrecedence(t *testing.T) {
	t.Run("Neutron pool wins over addresses and auto", func(t *testing.T) {
		f := newDispatchFixture(t, "")
		f.cloud.AdvertiseNova("/v2.1/", "2.1", "2.100")
		capture := captureServerCreate(t, f.cloud, "/v2.1/servers", `{"server":{"id":"server"}}`)
		ip := append(dispatchOptions(), compute.WithFloatingIPAddresses(dispatchAddresses["a"]), compute.WithFloatingIPPool(resource.ID("pool-network")), compute.WithAutomaticEnsureOptions(network.WithEnsureReuse(false)))
		result, err := f.service.CreateWithAutomaticFloatingIP(context.Background(), automaticCreateRequest(), compute.AutomaticServerCreateOptions{
			Server: []compute.CreateServerOption{compute.WithWait(resource.WithPollInterval(time.Millisecond))}, AutomaticIP: ip,
		})
		if err != nil || result == nil || result.Automatic.Mode != compute.ServerIPPool || result.Automatic.Decision.Reason != compute.AutomaticIPPoolRequested || result.Automatic.Assignment == nil || !result.Automatic.Assignment.Allocated || result.Automatic.Assignment.FloatingIP.ID != "ip-pool" || !result.Automatic.Observed || f.posts.Load() != 1 || slices.Contains(f.trace(), "list:a") {
			t.Fatal(result, err, f.trace())
		}
		if posts, body, version := capture.snapshot(); posts != 1 || body["networks"] != "auto" || version != "2.37" {
			t.Fatal(posts, body, version)
		}
	})
	t.Run("Neutron addresses win over auto", func(t *testing.T) {
		f := newDispatchFixture(t, "")
		f.cloud.AdvertiseNova("/v2.1/", "2.1", "2.100")
		captureServerCreate(t, f.cloud, "/v2.1/servers", `{"server":{"id":"server"}}`)
		ip := append(dispatchOptions(), compute.WithFloatingIPAddresses(dispatchAddresses["a"]), compute.WithAutomaticIPEnabled(true))
		result, err := f.service.CreateWithAutomaticFloatingIP(context.Background(), automaticCreateRequest(), compute.AutomaticServerCreateOptions{
			Server: []compute.CreateServerOption{compute.WithWait(resource.WithPollInterval(time.Millisecond))}, AutomaticIP: ip,
		})
		if err != nil || result == nil || result.Automatic.Mode != compute.ServerIPExplicit || result.Automatic.Decision.Reason != compute.AutomaticIPAddressesRequested || len(result.Automatic.Attempts) != 1 || result.Automatic.Assignment.FloatingIP.ID != "ip-a" || result.Automatic.Assignment.Allocated || f.posts.Load() != 0 || slices.Contains(f.trace(), "list:pool") {
			t.Fatal(result, err, f.trace())
		}
	})
	t.Run("Neutron auto default allocates on the external network", func(t *testing.T) {
		f := newAutomaticFixture(t)
		discoveries := f.cloud.AdvertiseNova("/v2.1/", "2.1", "2.100")
		posts := addAutomaticCreate(f, `{"server":{"id":"server","status":"BUILD"}}`, 202, func(body map[string]any) {
			if body["networks"] != "auto" {
				t.Error(body)
			}
		})
		f.rawBody = func(n int32) (int, string) {
			if n == 1 {
				return 200, `{"server":{"id":"server","status":"ACTIVE","addresses":` + autoFixed + `}}`
			}
			return 200, `{"server":{"id":"server","status":"ACTIVE","addresses":` + autoFloating + `}}`
		}
		result, err := f.service.CreateWithAutomaticFloatingIP(context.Background(), automaticCreateRequest(), compute.AutomaticServerCreateOptions{
			Server: []compute.CreateServerOption{compute.WithWait(resource.WithPollInterval(time.Millisecond))}, AutomaticIP: automaticOptions(),
		})
		if err != nil || result == nil || result.Automatic.Mode != compute.ServerIPAutomatic || result.Automatic.Decision.Reason != compute.AutomaticIPNeeded || result.Automatic.Assignment == nil || !result.Automatic.Assignment.Allocated || !result.Automatic.Observed || posts.Load() != 1 || f.posts.Load() != 1 || discoveries.Load() != 1 {
			t.Fatal(result, err, posts.Load(), f.posts.Load(), discoveries.Load())
		}
	})
	t.Run("source none disables auto", func(t *testing.T) {
		f := newDispatchFixture(t, "")
		f.cloud.AdvertiseNova("/v2.1/", "2.1", "2.100")
		captureServerCreate(t, f.cloud, "/v2.1/servers", `{"server":{"id":"server"}}`)
		ip := append(dispatchOptions(), compute.WithAutomaticAddressOptions(compute.WithFloatingIPSource(compute.FloatingIPNone)))
		result, err := f.service.CreateWithAutomaticFloatingIP(context.Background(), automaticCreateRequest(), compute.AutomaticServerCreateOptions{
			Server: []compute.CreateServerOption{compute.WithWait(resource.WithPollInterval(time.Millisecond))}, AutomaticIP: ip,
		})
		if err != nil || result == nil || result.Automatic.Mode != compute.ServerIPAutomatic || result.Automatic.Decision.Reason != compute.AutomaticIPDisabled || result.Automatic.Decision.Needed || f.posts.Load() != 0 || len(f.trace()) != 1 {
			t.Fatal(result, err, f.trace())
		}
	})
	for _, lane := range []string{"Nova auto default pool", "Nova pool wins over addresses", "Nova addresses"} {
		t.Run(lane, func(t *testing.T) {
			f := newNovaIPFixture(t, "")
			f.status = "ACTIVE"
			capture := captureServerCreate(t, f.cloud, "/v2.1/servers", `{"server":{"id":"server"}}`)
			ip := []compute.AutomaticFloatingIPOption{compute.WithAutomaticAddressOptions(compute.WithFloatingIPSource(compute.FloatingIPNova), compute.WithAddressReachability(false)), compute.WithAutomaticEnsureOptions(network.WithEnsureFixedAddress("10.0.0.10")), compute.WithAutomaticIPPollInterval(time.Millisecond)}
			mode, want := compute.ServerIPAutomatic, []string{"pools", "list", "get:2", "attach:" + dispatchAddresses["b"]}
			switch lane {
			case "Nova pool wins over addresses":
				ip = append(ip, compute.WithFloatingIPAddresses(dispatchAddresses["c"]), compute.WithFloatingIPPool(resource.Name("public")))
				mode, want = compute.ServerIPPool, []string{"list", "get:2", "attach:" + dispatchAddresses["b"]}
			case "Nova addresses":
				ip = append(ip, compute.WithFloatingIPAddresses(dispatchAddresses["c"]))
				mode, want = compute.ServerIPExplicit, []string{"list", "get:3", "attach:" + dispatchAddresses["c"]}
			}
			result, err := f.service.CreateWithAutomaticFloatingIP(context.Background(), automaticCreateRequest(), compute.AutomaticServerCreateOptions{
				Server: []compute.CreateServerOption{compute.WithWait(resource.WithPollInterval(time.Millisecond))}, AutomaticIP: ip,
			})
			if err != nil || result == nil || result.Automatic.Mode != mode || result.Automatic.Assignment != nil || result.Automatic.NovaAssignment == nil || !result.Automatic.NovaAssignment.ActionAccepted || !result.Automatic.Observed {
				t.Fatal(result, err, f.trace())
			}
			var actions []string
			for _, event := range f.trace() {
				if event != "raw" {
					actions = append(actions, event)
				}
			}
			if !reflect.DeepEqual(actions, want) {
				t.Fatalf("actions=%v want=%v", actions, want)
			}
			// The pinned 2.35 client cannot send networks=auto, so the
			// missing NIC selection is omitted without discovery.
			if posts, body, version := capture.snapshot(); posts != 1 || body["networks"] != nil || version != "2.35" {
				t.Fatal(posts, body, version)
			}
		})
	}
}
