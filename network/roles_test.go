package network_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func roleIDs(rows []*network.RoleNetwork) []string {
	result := make([]string, len(rows))
	for i, row := range rows {
		result[i] = row.ID
	}
	return result
}
func preparedRoles(t *testing.T, options ...network.NetworkRoleOption) network.NetworkRolePolicy {
	t.Helper()
	policy, err := network.PrepareNetworkRoleOptions(options...)
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages(t *testing.T) {
	for _, configured := range []bool{false, true} {
		t.Run(fmt.Sprint(configured), func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, subnetLists atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.Header.Get("X-Auth-Token") != "test-token" || r.URL.Query().Get("name") != "" || r.URL.Query().Get("router:external") != "" {
					t.Error(r.URL, r.Header)
				}
				switch r.URL.Query().Get("marker") {
				case "":
					testcloud.JSON(w, 200, fmt.Sprintf(`{"networks":[{"id":"ext","name":"public","router:external":true,"tags":["original"],"subnets":["s"],"created_at":"2026-10-07T01:02:03Z"},{"id":"provider","name":"physical","provider:physical_network":"phys"}],"networks_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2.0/networks?marker=empty"))
				case "empty":
					testcloud.JSON(w, 200, fmt.Sprintf(`{"networks":[],"networks_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2.0/networks?marker=last"))
				case "last":
					testcloud.JSON(w, 200, `{"networks":[{"id":"first","name":"private-first"},{"id":"last","name":"private-last"}]}`)
				default:
					t.Error(r.URL)
					http.Error(w, "unexpected", 500)
				}
			})
			cloud.Mux.HandleFunc("GET /v2.0/subnets", func(w http.ResponseWriter, r *http.Request) {
				subnetLists.Add(1)
				if configured {
					t.Error("configured NAT destination must skip subnet discovery")
				}
				if r.URL.Query().Get("marker") == "" {
					testcloud.JSON(w, 200, fmt.Sprintf(`{"subnets":[{"id":"s1","network_id":"first","gateway_ip":"10.0.0.1","ip_version":6}],"subnets_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2.0/subnets?marker=second"))
					return
				}
				if r.URL.Query().Get("marker") == "second" {
					testcloud.JSON(w, 200, fmt.Sprintf(`{"subnets":[],"subnets_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2.0/subnets?marker=third"))
					return
				}
				testcloud.JSON(w, 200, `{"subnets":[{"id":"s2","network_id":"last","gateway_ip":"10.1.0.1"}]}`)
			})
			policy := network.NetworkRolePolicy{}
			want4, wantInternal4 := []string{"ext", "provider"}, []string{"first", "last"}
			want6, wantInternal6 := []string{"ext"}, []string{"provider", "first", "last"}
			wantSource, wantDestination, wantDefault := "ext", "last", ""
			if configured {
				policy = preparedRoles(t, network.WithConfiguredNetworks(
					network.ConfiguredNetwork{Name: "provider", RoutesIPv6Externally: true, NATSource: true},
					network.ConfiguredNetwork{Name: "private-first", NATDestination: true},
					network.ConfiguredNetwork{Name: "last", DefaultInterface: true}))
				want4, wantInternal4 = []string{"ext"}, []string{"provider", "first", "last"}
				want6, wantInternal6 = []string{"ext", "provider"}, []string{"first", "last"}
				wantSource, wantDestination, wantDefault = "provider", "first", "last"
			}
			roles := network.NewWithDependencies(cloud.Client("network", "/v2.0"), network.Dependencies{NetworkRoles: policy}).Roles
			result, err := roles.Discover(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(roleIDs(result.ExternalIPv4), want4) || !reflect.DeepEqual(roleIDs(result.InternalIPv4), wantInternal4) || !reflect.DeepEqual(roleIDs(result.ExternalIPv6), want6) || !reflect.DeepEqual(roleIDs(result.InternalIPv6), wantInternal6) {
				t.Fatalf("roles=%+v", result)
			}
			if result.NATSource.ID != wantSource || result.NATDestination.ID != wantDestination || !reflect.DeepEqual(roleIDs(result.ExternalIPv4Floating), []string{wantSource}) {
				t.Fatalf("NAT=%+v", result)
			}
			if (result.DefaultNetwork == nil) != (wantDefault == "") || (result.DefaultNetwork != nil && result.DefaultNetwork.ID != wantDefault) {
				t.Fatalf("default=%+v", result.DefaultNetwork)
			}
			if !result.ExternalIPv4[0].RouterExternal || result.ExternalIPv4[0].CreatedAt.IsZero() {
				t.Fatal("lost extension/native timestamp")
			}
			provider := result.InternalIPv6[0]
			if configured {
				provider = result.ExternalIPv6[1]
			}
			if provider.ProviderPhysicalNetwork != "phys" {
				t.Fatal("lost provider extension")
			}
			result.ExternalIPv4[0].ID = "changed"
			result.ExternalIPv4[0].Tags[0] = "changed"
			result.ExternalIPv4[0].Subnets[0] = "changed"
			cached, err := roles.Discover(context.Background())
			if err != nil || cached.ExternalIPv4[0].ID != "ext" || cached.ExternalIPv4[0].Tags[0] != "original" || cached.ExternalIPv4[0].Subnets[0] != "s" || lists.Load() != 3 {
				t.Fatalf("cache=%+v err=%v requests=%d", cached, err, lists.Load())
			}
			wantSubnets := int32(3)
			if configured {
				wantSubnets = 0
			}
			if subnetLists.Load() != wantSubnets {
				t.Fatal(subnetLists.Load())
			}
		})
	}
}

func TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors(t *testing.T) {
	for _, scenario := range []string{"missing role", "multiple missing roles", "missing NAT source", "cross-kind default", "duplicate destination", "unsafe ID", "late network403", "subnet403", "cycle", "invalid extension", "204", "first NAT source"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				switch scenario {
				case "204":
					w.WriteHeader(204)
				case "cycle":
					testcloud.JSON(w, 200, fmt.Sprintf(`{"networks":[],"networks_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2.0/networks"))
				case "late network403":
					if r.URL.Query().Get("marker") != "" {
						testcloud.JSON(w, 403, `{}`)
					} else {
						testcloud.JSON(w, 200, fmt.Sprintf(`{"networks":[{"id":"one","name":"chosen"}],"networks_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2.0/networks?marker=last"))
					}
				case "cross-kind default":
					testcloud.JSON(w, 200, `{"networks":[{"id":"chosen","name":"other"},{"id":"two","name":"chosen"}]}`)
				case "duplicate destination":
					testcloud.JSON(w, 200, `{"networks":[{"id":"one","name":"chosen"},{"id":"two","name":"chosen"}]}`)
				case "unsafe ID":
					testcloud.JSON(w, 200, `{"networks":[{"id":"bad/id","name":"chosen"}]}`)
				case "invalid extension":
					testcloud.JSON(w, 200, `{"networks":[{"id":"one","name":"chosen","router:external":"invalid"}]}`)
				default:
					testcloud.JSON(w, 200, `{"networks":[{"id":"one","name":"chosen"},{"id":"two","name":"second"}]}`)
				}
			})
			cloud.Mux.HandleFunc("GET /v2.0/subnets", func(w http.ResponseWriter, r *http.Request) {
				if scenario == "subnet403" {
					testcloud.JSON(w, 403, `{}`)
				} else {
					testcloud.JSON(w, 200, `{"subnets":[]}`)
				}
			})
			var configured []network.ConfiguredNetwork
			switch scenario {
			case "missing role":
				configured = []network.ConfiguredNetwork{{Name: "absent", RoutesIPv4Externally: true}}
			case "multiple missing roles":
				configured = []network.ConfiguredNetwork{{Name: "first-missing", RoutesIPv4Externally: true}, {Name: "second-missing", RoutesIPv4Externally: true}}
			case "missing NAT source":
				configured = []network.ConfiguredNetwork{{Name: "absent", NATSource: true}}
			case "cross-kind default":
				configured = []network.ConfiguredNetwork{{Name: "chosen", DefaultInterface: true}}
			case "duplicate destination":
				configured = []network.ConfiguredNetwork{{Name: "chosen", NATDestination: true}}
			case "first NAT source":
				configured = []network.ConfiguredNetwork{{Name: "one", NATSource: true}, {Name: "two", NATSource: true}}
			}
			policy := preparedRoles(t, network.WithConfiguredNetworks(configured...))
			roles := network.NewWithDependencies(cloud.Client("network", "/v2.0"), network.Dependencies{NetworkRoles: policy}).Roles
			result, err := roles.Discover(context.Background())
			switch scenario {
			case "204":
				if err != nil || len(result.ExternalIPv4) != 0 {
					t.Fatal(result, err)
				}
			case "first NAT source":
				if err != nil || result.NATSource.ID != "one" || !reflect.DeepEqual(roleIDs(result.ExternalIPv4Floating), []string{"one"}) {
					t.Fatal(result, err)
				}
			case "missing role", "missing NAT source":
				if !errors.Is(err, resource.ErrNotFound) {
					t.Fatal(err)
				}
			case "multiple missing roles":
				var missing *resource.NotFoundError
				if !errors.As(err, &missing) || missing.Reference != "first-missing" {
					t.Fatal(err)
				}
			case "cross-kind default", "duplicate destination":
				if !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(err)
				}
			case "unsafe ID":
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			case "cycle":
				if err == nil || calls.Load() != 1 {
					t.Fatalf("err=%v calls=%d", err, calls.Load())
				}
			case "invalid extension":
				if err == nil {
					t.Fatal("invalid extension was accepted")
				}
			default:
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) || response.Actual != 403 {
					t.Fatal(err)
				}
			}
			if err != nil {
				if result != nil {
					t.Fatal("partial roles must not be cached/returned")
				}
				before := calls.Load()
				_, _ = roles.Discover(context.Background())
				if calls.Load() <= before {
					t.Fatal("failure was cached")
				}
			}
		})
	}
}

// This context observes entry into the follower's select without sleeps or
// inspecting the implementation's lock/cache fields.
type roleFollowerContext struct {
	context.Context
	entered chan struct{}
	once    sync.Once
}

func (c *roleFollowerContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.entered) })
	return c.Context.Done()
}

func TestNetworkRoleConcurrentDiscoveriesShareFlightAndOwnResults(t *testing.T) {
	cloud := testcloud.New(t)
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			close(started)
			<-release
		}
		testcloud.JSON(w, 200, `{"networks":[{"id":"one","name":"one"}]}`)
	})
	policy := preparedRoles(t, network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "one", NATDestination: true}))
	roles := network.NewWithDependencies(cloud.Client("network", "/v2.0"), network.Dependencies{NetworkRoles: policy}).Roles
	results := make(chan *network.NetworkRoleSnapshot, 3)
	run := func(ctx context.Context) {
		result, err := roles.Discover(ctx)
		if err != nil {
			t.Error(err)
		}
		results <- result
	}
	go run(context.Background())
	<-started
	for range 2 {
		ctx := &roleFollowerContext{Context: context.Background(), entered: make(chan struct{})}
		go run(ctx)
		<-ctx.entered
	}
	if calls.Load() != 1 {
		t.Fatal(calls.Load())
	}
	releaseOnce.Do(func() { close(release) })
	first, second, third := <-results, <-results, <-results
	if first == nil || second == nil || third == nil || calls.Load() != 1 {
		t.Fatal(first, second, third, calls.Load())
	}
	first.NATDestination.Name = "changed"
	if second.NATDestination.Name != "one" || third.NATDestination.Name != "one" {
		t.Fatal("shared mutable result")
	}
}

func TestNetworkRoleCacheFlightCancellationAndReset(t *testing.T) {
	cloud := testcloud.New(t)
	started, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			close(started)
			<-release
		}
		testcloud.JSON(w, 200, `{"networks":[{"id":"one","name":"one"}]}`)
	})
	policy := preparedRoles(t, network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "one", NATDestination: true}))
	roles := network.NewWithDependencies(cloud.Client("network", "/v2.0"), network.Dependencies{NetworkRoles: policy}).Roles
	first := make(chan error, 1)
	go func() { _, err := roles.Discover(context.Background()); first <- err }()
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := roles.Discover(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	roles.Reset()
	close(release)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	if _, err := roles.Discover(context.Background()); err != nil || calls.Load() != 2 {
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
	second := make(chan error, 2)
	for range 2 {
		go func() { _, err := roles.Discover(context.Background()); second <- err }()
	}
	for range 2 {
		if err := <-second; err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 2 {
		t.Fatal("cached concurrent getters repeated requests", calls.Load())
	}
}

func TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices(t *testing.T) {
	for _, client := range []*gophercloud.ServiceClient{nil, {}} {
		if _, err := network.New(client).Roles.Discover(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	for _, options := range [][]network.NetworkRoleOption{
		{nil}, {network.WithConfiguredNetworks(network.ConfiguredNetwork{})},
		{network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "a", DefaultInterface: true}, network.ConfiguredNetwork{Name: "b", DefaultInterface: true})},
		{network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "a", NATDestination: true}, network.ConfiguredNetwork{Name: "b", NATDestination: true})},
		{network.WithConfiguredNetworks(network.ConfiguredNetwork{}), network.WithConfiguredNetworks()},
	} {
		if _, err := network.PrepareNetworkRoleOptions(options...); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	rows := []network.ConfiguredNetwork{{Name: "one", NATDestination: true}}
	option := network.WithConfiguredNetworks(rows...)
	rows[0].Name = "changed"
	policy := preparedRoles(t, option, network.WithExternalNetworkDiscovery(false), network.WithInternalNetworkDiscovery(false))
	if policy.UseExternalNetwork() || policy.UseInternalNetwork() {
		t.Fatal("disabled flags lost")
	}
	roles := network.NewWithDependencies(nil, network.Dependencies{NetworkRoles: policy}).Roles
	result, err := roles.Discover(context.Background())
	if err != nil || result.NATDestination != nil {
		t.Fatal(result, err)
	}
	if _, err := roles.Discover(nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	// Restore either flag; successful discovery still validates the original one.
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"networks":[{"id":"one","name":"one"}]}`)
	})
	policy = preparedRoles(t, option, network.WithExternalNetworkDiscovery(false), network.WithInternalNetworkDiscovery(false), network.WithInternalNetworkDiscovery(true))
	result, err = network.NewWithDependencies(cloud.Client("network", "/v2.0"), network.Dependencies{NetworkRoles: policy}).Roles.Discover(context.Background())
	if err != nil || result.NATDestination.ID != "one" {
		t.Fatal(result, err)
	}
}
