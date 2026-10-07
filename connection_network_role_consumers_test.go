package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

func TestConnectionDefaultNICConsumesRoleCacheAndReset(t *testing.T) {
	cloud := testcloud.New(t)
	var lists, subnets, creates atomic.Int32
	cloud.Mux.HandleFunc("GET /network/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		generation := lists.Add(1)
		if r.URL.RawQuery != "" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"networks":[{"id":"private-%d","name":"private"}]}`, generation))
	})
	cloud.Mux.HandleFunc("GET /network/v2.0/subnets", func(w http.ResponseWriter, r *http.Request) {
		subnets.Add(1)
		testcloud.JSON(w, 200, `{"subnets":[]}`)
	})
	cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
		call := creates.Add(1)
		want := "private-1"
		if call == 3 {
			want = "private-2"
		}
		var body struct{ Server map[string]any }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(body.Server["networks"], []any{map[string]any{"uuid": want}}) {
			t.Errorf("body=%#v want=%s", body.Server, want)
		}
		testcloud.JSON(w, 202, `{"server":{"id":"created"}}`)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute"), sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network"),
		sdk.WithNetworkRoles(network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "private", DefaultInterface: true})))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	roles, err := conn.GetNetworkRoles(ctx)
	if err != nil {
		t.Fatal(err)
	}
	roles.DefaultNetwork.ID = "caller-modified"
	service, err := conn.Compute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := service.Servers.Create(ctx, nicConnectionRequest()); err != nil {
			t.Fatal(err)
		}
	}
	if lists.Load() != 1 || subnets.Load() != 1 {
		t.Fatal(lists.Load(), subnets.Load())
	}
	conn.ResetNetworkRoles()
	if _, err := service.Servers.Create(ctx, nicConnectionRequest()); err != nil {
		t.Fatal(err)
	}
	if lists.Load() != 2 || subnets.Load() != 2 || creates.Load() != 3 {
		t.Fatal(lists.Load(), subnets.Load(), creates.Load())
	}
}

func TestConnectionDefaultNICRoleErrorsStopCreationAndRetry(t *testing.T) {
	for _, scenario := range []string{"subnet forbidden", "other configured role missing"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, subnets, creates atomic.Int32
			cloud.Mux.HandleFunc("GET /network/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				call := lists.Add(1)
				rows := `[{"id":"private-id","name":"private"}]`
				if call > 1 || scenario == "subnet forbidden" {
					rows = `[{"id":"private-id","name":"private"},{"id":"public-id","name":"public"}]`
				}
				testcloud.JSON(w, 200, `{"networks":`+rows+`}`)
			})
			cloud.Mux.HandleFunc("GET /network/v2.0/subnets", func(w http.ResponseWriter, r *http.Request) {
				call := subnets.Add(1)
				if call == 1 && scenario == "subnet forbidden" {
					testcloud.JSON(w, 403, `{"error":{"message":"subnets denied"}}`)
					return
				}
				testcloud.JSON(w, 200, `{"subnets":[]}`)
			})
			cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
				creates.Add(1)
				var body struct{ Server map[string]any }
				_ = json.NewDecoder(r.Body).Decode(&body)
				if !reflect.DeepEqual(body.Server["networks"], []any{map[string]any{"uuid": "private-id"}}) {
					t.Error(body.Server)
				}
				testcloud.JSON(w, 202, `{"server":{"id":"created"}}`)
			})
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute"), sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network"),
				sdk.WithNetworkRoles(network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "private", DefaultInterface: true}, network.ConfiguredNetwork{Name: "public", NATSource: true, RoutesIPv4Externally: true})))
			if err != nil {
				t.Fatal(err)
			}
			service, err := conn.Compute(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			row, err := service.Servers.Create(context.Background(), nicConnectionRequest())
			if row != nil || err == nil || creates.Load() != 0 {
				t.Fatal(row, err, creates.Load())
			}
			if scenario == "subnet forbidden" {
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) || response.Actual != 403 {
					t.Fatal(err)
				}
			} else if !errors.Is(err, resource.ErrNotFound) {
				t.Fatal(err)
			}
			for range 2 {
				if _, err := service.Servers.Create(context.Background(), nicConnectionRequest()); err != nil {
					t.Fatal(err)
				}
			}
			if lists.Load() != 2 || subnets.Load() != 2 || creates.Load() != 2 {
				t.Fatal(lists.Load(), subnets.Load(), creates.Load())
			}
		})
	}
}

func TestConnectionDefaultNICEmptyRolesUseNovaNetworkDefault(t *testing.T) {
	for _, scenario := range []string{"disabled", "missing endpoint", "missing endpoint value"} {
		for _, version := range []string{"", "2.37"} {
			t.Run(scenario+"/"+version, func(t *testing.T) {
				cloud := testcloud.New(t)
				var locates, creates atomic.Int32
				cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
					locates.Add(1)
					if scenario == "missing endpoint value" {
						return "", gophercloud.ErrEndpointNotFound{}
					}
					return "", &gophercloud.ErrEndpointNotFound{}
				}
				roleOpts := []network.NetworkRoleOption{network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "missing", DefaultInterface: true})}
				if scenario == "disabled" {
					roleOpts = append(roleOpts, network.WithExternalNetworkDiscovery(false), network.WithInternalNetworkDiscovery(false))
				}
				cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
					creates.Add(1)
					var body struct{ Server map[string]any }
					_ = json.NewDecoder(r.Body).Decode(&body)
					value, present := body.Server["networks"]
					if version == "2.37" && value != "auto" || version == "" && present {
						t.Error(body.Server)
					}
					testcloud.JSON(w, 202, `{"server":{"id":"created"}}`)
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("unexpected request %s %s", r.Method, r.URL)
					http.Error(w, "unexpected", 500)
				})
				opts := []sdk.ConnectionOption{sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute"), sdk.WithNetworkRoles(roleOpts...)}
				if version != "" {
					opts = append(opts, sdk.WithMicroversion(sdk.Compute, version))
				}
				conn, err := sdk.FromProvider(cloud.Provider, opts...)
				if err != nil {
					t.Fatal(err)
				}
				service, err := conn.Compute(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				if _, err := service.Servers.Create(context.Background(), nicConnectionRequest()); err != nil || creates.Load() != 1 {
					t.Fatal(err, creates.Load())
				}
				wantLocates := int32(1)
				if scenario == "disabled" {
					wantLocates = 0
				}
				if locates.Load() != wantLocates {
					t.Fatal(locates.Load(), wantLocates)
				}
			})
		}
	}
}
