package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func nicConnectionRequest() compute.CreateServerRequest {
	return compute.CreateServerRequest{
		Name: "nic-server", Image: resource.ID("image-id"), Flavor: resource.ID("flavor-id"),
	}
}

func TestConnectionServerNICsResolveExactNamesAndOwnInputs(t *testing.T) {
	cloud := testcloud.New(t)
	var networks, ports, creates atomic.Int32
	cloud.Mux.HandleFunc("GET /network/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		networks.Add(1)
		if r.URL.Query().Get("name") != "private[1].*" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Errorf("network query=%q headers=%v", r.URL.RawQuery, r.Header)
		}
		testcloud.JSON(w, 200, `{"networks":[{"id":"wrong-network","name":"private[1].*copy"},{"id":"network-id","name":"private[1].*"}]}`)
	})
	cloud.Mux.HandleFunc("GET /network/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		ports.Add(1)
		if r.URL.Query().Get("name") != "nic[1].*" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Errorf("port query=%q headers=%v", r.URL.RawQuery, r.Header)
		}
		testcloud.JSON(w, 200, `{"ports":[{"id":"wrong-port","name":"nic[1].*copy"},{"id":"port-id","name":"nic[1].*"}]}`)
	})
	cloud.Mux.HandleFunc("POST /compute/v2.1/project/servers", func(w http.ResponseWriter, r *http.Request) {
		creates.Add(1)
		var body struct {
			Server map[string]any `json:"server"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		want := []any{
			map[string]any{"uuid": "network-id", "fixed_ip": "2001:db8::10", "tag": ""},
			map[string]any{"port": "port-id"},
			map[string]any{"uuid": "other-network-id", "port": "other-port-id"},
		}
		if !reflect.DeepEqual(body.Server["networks"], want) {
			t.Errorf("networks=%#v want=%#v", body.Server["networks"], want)
		}
		for key, want := range map[string]any{"name": "nic-server", "imageRef": "image-id", "flavorRef": "flavor-id"} {
			if body.Server[key] != want {
				t.Errorf("%s=%#v want=%#v", key, body.Server[key], want)
			}
		}
		if r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != "compute 2.42" || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.42" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected lookup/wait: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	conn, err := sdk.FromProvider(cloud.Provider,
		sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute/v2.1/project"),
		sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network"),
		sdk.WithMicroversion(sdk.Compute, "2.42"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.Compute(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	tag := request.Present("")
	input := []compute.ServerNetworkInterface{
		{Network: resource.Name("private[1].*"), FixedIP: "2001:db8::10", Tag: tag},
		{Port: resource.Name("nic[1].*")},
		{Network: resource.ID("other-network-id"), Port: resource.ID("other-port-id")},
	}
	option := compute.WithNetworkInterfaces(input...)
	input[0].Network = resource.Name("mutated-network")
	input[1].Port = resource.Name("mutated-port")
	tag = request.Present("mutated-tag")
	for range 2 {
		row, err := service.Servers.Create(context.Background(), nicConnectionRequest(),
			compute.WithNetworks(resource.Name("discarded-network")), option)
		if err != nil || row == nil || row.ID != "created" || row.Status != "BUILD" {
			t.Fatalf("row=%+v err=%v", row, err)
		}
	}
	if networks.Load() != 2 || ports.Load() != 2 || creates.Load() != 2 {
		t.Fatalf("networks=%d ports=%d creates=%d", networks.Load(), ports.Load(), creates.Load())
	}
	cached, err := conn.Compute(context.Background())
	if err != nil || cached != service || cached.RawClient().ProviderClient != cloud.Provider || cached.RawClient().Microversion != "2.42" {
		t.Fatalf("cached=%v err=%v", cached, err)
	}
}

func TestConnectionServerNICNameFailuresPreventNovaCreation(t *testing.T) {
	for _, kind := range []string{"networks", "ports"} {
		for _, scenario := range []string{"missing", "ambiguous", "forbidden"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				cloud := testcloud.New(t)
				var lists, creates atomic.Int32
				name := "selected[1].*"
				path := "/network/v2.0/" + kind
				cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					if r.URL.Query().Get("name") != name || r.Header.Get("X-Auth-Token") != "test-token" {
						t.Errorf("query=%q headers=%v", r.URL.RawQuery, r.Header)
					}
					if scenario == "forbidden" {
						testcloud.JSON(w, 403, `{"error":{"message":"NIC lookup denied"}}`)
						return
					}
					rows := []any{}
					if scenario == "ambiguous" {
						rows = []any{
							map[string]any{"id": "one", "name": name},
							map[string]any{"id": "two", "name": name},
						}
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]any{kind: rows})
				})
				cloud.Mux.HandleFunc("POST /compute/v2.1/project/servers", func(w http.ResponseWriter, r *http.Request) {
					creates.Add(1)
					testcloud.JSON(w, 202, `{"server":{"id":"must-not-create"}}`)
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
					http.Error(w, "unexpected request", 500)
				})
				service, err := connection(t, cloud).Compute(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				nic := compute.ServerNetworkInterface{}
				if kind == "networks" {
					nic.Network = resource.Name(name)
				} else {
					nic.Port = resource.Name(name)
				}
				row, err := service.Servers.Create(context.Background(), nicConnectionRequest(), compute.WithNetworkInterfaces(nic))
				if err == nil || row != nil || lists.Load() != 1 || creates.Load() != 0 {
					t.Fatalf("row=%v err=%v lists=%d creates=%d", row, err, lists.Load(), creates.Load())
				}
				switch scenario {
				case "missing":
					if !errors.Is(err, resource.ErrNotFound) {
						t.Fatal(err)
					}
				case "ambiguous":
					if !errors.Is(err, resource.ErrAmbiguous) {
						t.Fatal(err)
					}
				case "forbidden":
					var response gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &response) || response.Actual != 403 {
						t.Fatal(err)
					}
				}
			})
		}
	}
}

func TestConnectionServerNICIDsNeedNoNetworkEndpoint(t *testing.T) {
	cloud := testcloud.New(t)
	var locates, creates atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		return "", errors.New("catalog intentionally unavailable")
	}
	cloud.Mux.HandleFunc("POST /compute/v2.1/project/servers", func(w http.ResponseWriter, r *http.Request) {
		creates.Add(1)
		var body struct {
			Server map[string]any `json:"server"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		want := []any{
			map[string]any{"uuid": "network-id", "port": "port-id", "fixed_ip": "192.0.2.10"},
			map[string]any{"port": "other-port-id"},
		}
		if !reflect.DeepEqual(body.Server["networks"], want) {
			t.Errorf("networks=%#v", body.Server["networks"])
		}
		if r.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("IDs must avoid lookup: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute/v2.1/project"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.Compute(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	row, err := service.Servers.Create(context.Background(), nicConnectionRequest(), compute.WithNetworkInterfaces(
		compute.ServerNetworkInterface{Network: resource.ID("network-id"), Port: resource.ID("port-id"), FixedIP: "192.0.2.10"},
		compute.ServerNetworkInterface{Port: resource.ID("other-port-id")}))
	if err != nil || row == nil || row.ID != "created" || creates.Load() != 1 || locates.Load() != 0 {
		t.Fatalf("row=%v err=%v creates=%d locates=%d", row, err, creates.Load(), locates.Load())
	}
}

func TestConnectionNegotiatedVersionControlsDefaultNetworkAuto(t *testing.T) {
	for _, maximum := range []string{"2.36", "2.37", "2.100"} {
		t.Run(maximum, func(t *testing.T) {
			cloud := testcloud.New(t)
			var discoveries, creates atomic.Int32
			cloud.Mux.HandleFunc("GET /nova/v2.1/{$}", func(w http.ResponseWriter, r *http.Request) {
				discoveries.Add(1)
				if r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != "" || r.Header.Get("X-OpenStack-Nova-API-Version") != "" {
					t.Error(r.Header)
				}
				testcloud.JSON(w, 200, `{"version":{"id":"v2.1","status":"CURRENT","min_version":"2.1","version":"`+maximum+`"}}`)
			})
			cloud.Mux.HandleFunc("POST /nova/v2.1/project/servers", func(w http.ResponseWriter, r *http.Request) {
				creates.Add(1)
				var body struct {
					Server map[string]any `json:"server"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				network, present := body.Server["networks"]
				if maximum == "2.36" {
					if present {
						t.Errorf("pre-2.37 networks=%#v", network)
					}
				} else if !present || network != "auto" {
					t.Errorf("default networks=%#v present=%v", network, present)
				}
				if r.Header.Get("OpenStack-API-Version") != "compute "+maximum || r.Header.Get("X-OpenStack-Nova-API-Version") != maximum || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Error(r.Header)
				}
				testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("default auto must not query Neutron: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected request", 500)
			})
			conn, err := sdk.FromProvider(cloud.Provider,
				sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/nova/v2.1/project"), sdk.WithLatestMicroversion(sdk.Compute))
			if err != nil {
				t.Fatal(err)
			}
			service, err := conn.Compute(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			client := service.RawClient()
			for range 2 {
				row, err := service.Servers.Create(context.Background(), nicConnectionRequest())
				if err != nil || row == nil || row.ID != "created" {
					t.Fatalf("row=%v err=%v", row, err)
				}
			}
			generated, err := conn.ComputeV2(context.Background())
			if err != nil || generated.RawClient() != client || client.Microversion != maximum || client.ProviderClient != cloud.Provider {
				t.Fatalf("cached client/version changed: generated=%v err=%v", generated, err)
			}
			selection, err := conn.Microversion(context.Background(), sdk.Compute)
			if err != nil || !selection.Negotiated || selection.Selected != maximum || selection.DiscoveryURL != cloud.Server.URL+"/nova/v2.1/" {
				t.Fatalf("selection=%+v err=%v", selection, err)
			}
			if discoveries.Load() != 1 || creates.Load() != 2 {
				t.Fatalf("discoveries=%d creates=%d", discoveries.Load(), creates.Load())
			}
		})
	}
}
