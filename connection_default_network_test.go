package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/compute"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestConnectionDefaultNetworkIDsOptionsAndBootSources(t *testing.T) {
	for _, version := range []string{"", "2.37"} {
		for _, source := range []string{"image", "existing-volume", "new-volume"} {
			t.Run(version+"/"+source, func(t *testing.T) {
				cloud := testcloud.New(t)
				var locates, creates atomic.Int32
				cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
					locates.Add(1)
					return "", errors.New("no Neutron endpoint")
				}
				cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
					creates.Add(1)
					var body struct{ Server map[string]any }
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					want := []any{map[string]any{"uuid": "default-id"}}
					if !reflect.DeepEqual(body.Server["networks"], want) || body.Server["flavorRef"] != "flavor-id" || body.Server["name"] != "nic-server" {
						t.Errorf("body=%#v", body.Server)
					}
					if source == "image" {
						if body.Server["imageRef"] != "image-id" || body.Server["block_device_mapping_v2"] != nil {
							t.Errorf("image boot=%#v", body.Server)
						}
					} else {
						rows, ok := body.Server["block_device_mapping_v2"].([]any)
						if !ok || len(rows) != 1 {
							t.Errorf("block mappings=%#v", body.Server)
						} else {
							row := rows[0].(map[string]any)
							wantID, wantSource := "volume-id", "volume"
							if source == "new-volume" {
								wantID, wantSource = "image-id", "image"
								if row["volume_size"] != float64(10) {
									t.Errorf("size=%#v", row)
								}
							}
							if row["uuid"] != wantID || row["source_type"] != wantSource || row["destination_type"] != "volume" || row["delete_on_termination"] != true || body.Server["imageRef"] != "" {
								t.Errorf("volume boot=%#v", body.Server)
							}
						}
					}
					if r.Header.Get("X-Auth-Token") != "test-token" {
						t.Error(r.Header)
					}
					testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("unexpected lookup: %s %s", r.Method, r.URL)
					http.Error(w, "unexpected", 500)
				})
				opts := []sdk.ConnectionOption{sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute"),
					sdk.WithDefaultNetwork(resource.Name("discarded")), sdk.WithoutDefaultNetwork(), sdk.WithDefaultNetwork(resource.ID("default-id"))}
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
				req := nicConnectionRequest()
				var createOpts []compute.CreateServerOption
				if source == "existing-volume" {
					req.Image = resource.Ref{}
					createOpts = append(createOpts, compute.WithBootVolume(resource.ID("volume-id")))
				} else if source == "new-volume" {
					createOpts = append(createOpts, compute.WithBootVolumeSize(10))
				}
				if source != "image" {
					createOpts = append(createOpts, compute.WithDeleteBootVolumeOnTermination(true))
				}
				for range 2 {
					row, err := service.Servers.Create(context.Background(), req, createOpts...)
					if err != nil || row == nil || row.ID != "created" {
						t.Fatalf("row=%v err=%v", row, err)
					}
				}
				if locates.Load() != 0 || creates.Load() != 2 || service.RawClient().Microversion != version {
					t.Fatalf("locates=%d creates=%d version=%q", locates.Load(), creates.Load(), service.RawClient().Microversion)
				}
			})
		}
	}
}

func TestConnectionDefaultNetworkNameFailuresAndRetry(t *testing.T) {
	for _, scenario := range []string{"success", "missing", "ambiguous", "forbidden"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, creates atomic.Int32
			cloud.Mux.HandleFunc("GET /network/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("name") != "private[1].*" || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Errorf("query=%q headers=%v", r.URL.RawQuery, r.Header)
				}
				switch scenario {
				case "success":
					testcloud.JSON(w, 200, `{"networks":[{"id":"wrong","name":"private[1].*copy"},{"id":"selected","name":"private[1].*"}]}`)
				case "missing":
					testcloud.JSON(w, 200, `{"networks":[]}`)
				case "ambiguous":
					testcloud.JSON(w, 200, `{"networks":[{"id":"one","name":"private[1].*"},{"id":"two","name":"private[1].*"}]}`)
				case "forbidden":
					testcloud.JSON(w, 403, `{"error":{"message":"denied"}}`)
				}
			})
			cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
				creates.Add(1)
				var body struct{ Server map[string]any }
				_ = json.NewDecoder(r.Body).Decode(&body)
				if !reflect.DeepEqual(body.Server["networks"], []any{map[string]any{"uuid": "selected"}}) {
					t.Errorf("networks=%#v", body.Server)
				}
				testcloud.JSON(w, 202, `{"server":{"id":"created"}}`)
			})
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute"),
				sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network"), sdk.WithDefaultNetwork(resource.Name("private[1].*")))
			if err != nil {
				t.Fatal(err)
			}
			service, err := conn.Compute(context.Background())
			if err != nil || lists.Load() != 0 {
				t.Fatalf("lazy construction err=%v lists=%d", err, lists.Load())
			}
			for range 2 {
				row, err := service.Servers.Create(context.Background(), nicConnectionRequest())
				switch scenario {
				case "success":
					if err != nil || row == nil || row.ID != "created" {
						t.Fatalf("row=%v err=%v", row, err)
					}
				case "missing":
					if row != nil || !errors.Is(err, resource.ErrNotFound) {
						t.Fatalf("row=%v err=%v", row, err)
					}
				case "ambiguous":
					if row != nil || !errors.Is(err, resource.ErrAmbiguous) {
						t.Fatalf("row=%v err=%v", row, err)
					}
				case "forbidden":
					var response gophercloud.ErrUnexpectedResponseCode
					if row != nil || !errors.As(err, &response) || response.Actual != 403 {
						t.Fatalf("row=%v err=%v", row, err)
					}
				}
			}
			wantCreates := int32(0)
			if scenario == "success" {
				wantCreates = 2
			}
			if lists.Load() != 2 || creates.Load() != wantCreates {
				t.Fatalf("lists=%d creates=%d", lists.Load(), creates.Load())
			}
		})
	}
}

func TestConnectionDefaultNetworkDisableAndExplicitSelections(t *testing.T) {
	for _, tc := range []struct {
		name    string
		disable bool
		option  compute.CreateServerOption
		want    any
	}{
		{"disabled", true, nil, "auto"},
		{"network", false, compute.WithNetworks(resource.ID("explicit")), []any{map[string]any{"uuid": "explicit"}}},
		{"empty NIC", false, compute.WithNetworkInterfaces(compute.ServerNetworkInterface{}), []any{map[string]any{}}},
		{"fixed IP only", false, compute.WithNetworkInterfaces(compute.ServerNetworkInterface{FixedIP: "192.0.2.1"}), []any{map[string]any{"fixed_ip": "192.0.2.1"}}},
		{"none", false, compute.WithNetworkMode("none"), "none"},
		{"auto", false, compute.WithNetworkMode("auto"), "auto"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var locates atomic.Int32
			cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				return "", errors.New("default network must not be resolved")
			}
			cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
				var body struct{ Server map[string]any }
				_ = json.NewDecoder(r.Body).Decode(&body)
				if !reflect.DeepEqual(body.Server["networks"], tc.want) {
					t.Errorf("networks=%#v want=%#v", body.Server["networks"], tc.want)
				}
				testcloud.JSON(w, 202, `{"server":{"id":"created"}}`)
			})
			opts := []sdk.ConnectionOption{sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute"), sdk.WithMicroversion(sdk.Compute, "2.42"), sdk.WithDefaultNetwork(resource.Name("missing"))}
			if tc.disable {
				opts = append(opts, sdk.WithoutDefaultNetwork())
			}
			conn, err := sdk.FromProvider(cloud.Provider, opts...)
			if err != nil {
				t.Fatal(err)
			}
			service, err := conn.Compute(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			var createOpts []compute.CreateServerOption
			if tc.option != nil {
				createOpts = append(createOpts, tc.option)
			}
			row, err := service.Servers.Create(context.Background(), nicConnectionRequest(), createOpts...)
			if err != nil || row == nil || locates.Load() != 0 {
				t.Fatalf("row=%v err=%v locates=%d", row, err, locates.Load())
			}
		})
	}
}

func TestConnectionDefaultNetworkOptionValidationAndConcurrentReuse(t *testing.T) {
	cloud := testcloud.New(t)
	for _, ref := range []resource.Ref{{}, resource.Name(""), resource.ID("bad/path")} {
		_, err := sdk.FromProvider(cloud.Provider, sdk.WithDefaultNetwork(ref), sdk.WithoutDefaultNetwork())
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	var creates atomic.Int32
	cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
		creates.Add(1)
		var body struct{ Server map[string]any }
		_ = json.NewDecoder(r.Body).Decode(&body)
		if !reflect.DeepEqual(body.Server["networks"], []any{map[string]any{"uuid": "default-id"}}) {
			t.Errorf("body=%#v", body.Server)
		}
		testcloud.JSON(w, 202, `{"server":{"id":"created"}}`)
	})
	option := sdk.WithDefaultNetwork(resource.ID("default-id"))
	services := make([]*compute.Service, 2)
	for i := range services {
		conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute"), option)
		if err != nil {
			t.Fatal(err)
		}
		services[i], err = conn.Compute(context.Background())
		if err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			row, err := services[i%len(services)].Servers.Create(context.Background(), nicConnectionRequest())
			if err != nil || row == nil || row.ID != "created" {
				t.Errorf("row=%v err=%v", row, err)
			}
		})
	}
	wg.Wait()
	if creates.Load() != 8 {
		t.Fatal(creates.Load())
	}
}
