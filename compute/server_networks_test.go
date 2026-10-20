package compute_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func nicCreateRequest() compute.CreateServerRequest {
	return compute.CreateServerRequest{Name: "nic-server", Image: resource.ID("image"), Flavor: resource.ID("flavor")}
}

func checkNICServerBody(t *testing.T, r *http.Request, want any) {
	t.Helper()
	var body struct {
		Server map[string]any `json:"server"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Error(err)
		return
	}
	value, present := body.Server["networks"]
	if present != (want != nil) {
		t.Errorf("networks present=%v want=%v", present, want != nil)
	}
	if !reflect.DeepEqual(value, want) {
		t.Errorf("networks=%#v want=%#v", body.Server["networks"], want)
	}
	if r.Header.Get("X-Auth-Token") != "test-token" {
		t.Errorf("token=%q", r.Header.Get("X-Auth-Token"))
	}
}

func TestServerNICWirePreservesCombinationsOrderAndExplicitEmptyTag(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("compute", "/prefix/v2.1/project")
	client.Microversion = "2.42"
	service := compute.New(client, compute.Dependencies{
		Network: func(context.Context, resource.Ref) (string, error) { t.Error("ID network lookup"); return "", nil },
		Port:    func(context.Context, resource.Ref) (string, error) { t.Error("ID port lookup"); return "", nil },
	})
	tag, empty := request.Present("app"), request.Present("")
	nics := []compute.ServerNetworkInterface{
		{Network: resource.ID("network"), Port: resource.ID("port"), FixedIP: "2001:db8::10", Tag: tag},
		{Port: resource.ID("second-port")},
		{FixedIP: "literal-address-without-local-parsing"},
		{Tag: empty},
		{Tag: request.Null[string]()},
		{},
	}
	want := []any{
		map[string]any{"uuid": "network", "port": "port", "fixed_ip": "2001:db8::10", "tag": "app"},
		map[string]any{"port": "second-port"},
		map[string]any{"fixed_ip": "literal-address-without-local-parsing"},
		map[string]any{"tag": ""},
		map[string]any{"tag": nil},
		map[string]any{},
	}
	cloud.Mux.HandleFunc("POST /prefix/v2.1/project/servers", func(w http.ResponseWriter, r *http.Request) {
		checkNICServerBody(t, r, want)
		if r.Header.Get("OpenStack-API-Version") != "compute 2.42" || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.42" {
			t.Errorf("headers=%v", r.Header)
		}
		testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
	})
	server, err := service.Servers.Create(context.Background(), nicCreateRequest(), compute.WithNetworkInterfaces(nics...))
	if err != nil || server == nil || server.ID != "created" || client.Microversion != "2.42" {
		t.Fatalf("server=%v err=%v version=%q", server, err, client.Microversion)
	}
}

func TestServerNICOptionsOwnSlicesTagsAndConcurrentReuse(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("compute", "/compute")
	client.Microversion = "2.100"
	service := compute.New(client, compute.Dependencies{})
	tag := request.Present("original")
	input := []compute.ServerNetworkInterface{{Network: resource.ID("network"), Tag: tag}}
	option := compute.WithNetworkInterfaces(input...)
	input[0].Network = resource.ID("changed")
	tag = request.Present("changed")
	input[0].Tag = request.Null[string]()
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		checkNICServerBody(t, r, []any{map[string]any{"uuid": "network", "tag": "original"}})
		testcloud.JSON(w, 202, `{"server":{"id":"created"}}`)
	})
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if _, err := service.Servers.Create(context.Background(), nicCreateRequest(), option); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if calls.Load() != 8 || tag != request.Present("changed") || !input[0].Tag.IsNull() {
		t.Fatalf("calls=%d input=%+v tag=%v", calls.Load(), input, tag)
	}
}

func TestServerNICSelectionUsesLastValidOption(t *testing.T) {
	tag := request.Present("old")
	for _, tc := range []struct {
		name, version string
		opts          []compute.CreateServerOption
		want          any
	}{
		{"NIC replaces networks", "", []compute.CreateServerOption{compute.WithNetworks(resource.Name("old")), compute.WithNetworkInterfaces(compute.ServerNetworkInterface{Port: resource.ID("port")})}, []any{map[string]any{"port": "port"}}},
		{"networks replace tagged NIC", "", []compute.CreateServerOption{compute.WithNetworkInterfaces(compute.ServerNetworkInterface{Port: resource.Name("old"), Tag: tag}), compute.WithNetworks(resource.ID("network"))}, []any{map[string]any{"uuid": "network"}}},
		{"mode replaces NIC", "2.37", []compute.CreateServerOption{compute.WithNetworkInterfaces(compute.ServerNetworkInterface{Port: resource.Name("old"), Tag: tag}), compute.WithNetworkMode("none")}, "none"},
		{"NIC replaces mode", "", []compute.CreateServerOption{compute.WithNetworkMode("auto"), compute.WithNetworkInterfaces(compute.ServerNetworkInterface{Network: resource.ID("network")})}, []any{map[string]any{"uuid": "network"}}},
		{"mode replaces networks", "2.37", []compute.CreateServerOption{compute.WithNetworks(resource.Name("old")), compute.WithNetworkMode("auto")}, "auto"},
		{"networks replace mode", "", []compute.CreateServerOption{compute.WithNetworkMode("none"), compute.WithNetworks(resource.ID("network"))}, []any{map[string]any{"uuid": "network"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("compute", "/compute")
			client.Microversion = tc.version
			failResolve := func(context.Context, resource.Ref) (string, error) {
				t.Error("replaced selection resolved")
				return "", nil
			}
			service := compute.New(client, compute.Dependencies{Network: failResolve, Port: failResolve})
			cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
				checkNICServerBody(t, r, tc.want)
				testcloud.JSON(w, 202, `{"server":{"id":"created"}}`)
			})
			if _, err := service.Servers.Create(context.Background(), nicCreateRequest(), tc.opts...); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestServerNICDefaultsModesAndVersionBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, version string
		opts          []compute.CreateServerOption
		want          any
	}{
		{"unselected version omits networks", "", nil, nil},
		{"older version omits networks", "2.36", nil, nil},
		{"modern version defaults auto", "2.37", nil, "auto"},
		{"three digit minor defaults auto", "2.100", nil, "auto"},
		{"explicit none", "2.37", []compute.CreateServerOption{compute.WithNetworkMode("none")}, "none"},
		{"explicit auto", "2.37", []compute.CreateServerOption{compute.WithNetworkMode("auto")}, "auto"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("compute", "/compute")
			client.Microversion = tc.version
			service := compute.New(client, compute.Dependencies{})
			cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
				checkNICServerBody(t, r, tc.want)
				testcloud.JSON(w, 202, `{"server":{"id":"created"}}`)
			})
			if _, err := service.Servers.Create(context.Background(), nicCreateRequest(), tc.opts...); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestServerNICInvalidOptionsFailBeforeDependencyLookup(t *testing.T) {
	empty := request.Present("")
	for _, tc := range []struct {
		name, version string
		opts          []compute.CreateServerOption
		want          error
	}{
		{"empty NIC list", "2.42", []compute.CreateServerOption{compute.WithNetworkInterfaces()}, resource.ErrInvalidOption},
		{"empty networks still fail", "2.42", []compute.CreateServerOption{compute.WithNetworks(), compute.WithNetworkMode("auto")}, resource.ErrInvalidOption},
		{"invalid later NIC", "2.42", []compute.CreateServerOption{compute.WithNetworkInterfaces(compute.ServerNetworkInterface{Network: resource.Name("valid")}, compute.ServerNetworkInterface{Port: resource.ID("bad/port")})}, resource.ErrInvalidOption},
		{"invalid network name", "2.42", []compute.CreateServerOption{compute.WithNetworkInterfaces(compute.ServerNetworkInterface{Network: resource.Name("")})}, resource.ErrInvalidOption},
		{"unknown mode", "2.42", []compute.CreateServerOption{compute.WithNetworkMode("other")}, resource.ErrInvalidOption},
		{"none before 2.37", "2.36", []compute.CreateServerOption{compute.WithNetworkMode("none")}, resource.ErrUnsupported},
		{"auto with no version", "", []compute.CreateServerOption{compute.WithNetworkMode("auto")}, resource.ErrUnsupported},
		{"empty tag before 2.42", "2.41", []compute.CreateServerOption{compute.WithNetworkInterfaces(compute.ServerNetworkInterface{Tag: empty})}, resource.ErrUnsupported},
		{"tag old native range", "2.32", []compute.CreateServerOption{compute.WithNetworkInterfaces(compute.ServerNetworkInterface{Tag: empty})}, resource.ErrUnsupported},
		{"null tag before 2.42", "2.41", []compute.CreateServerOption{compute.WithNetworkInterfaces(compute.ServerNetworkInterface{Tag: request.Null[string]()})}, resource.ErrUnsupported},
		{"tag malformed version", "2.42.0", []compute.CreateServerOption{compute.WithNetworkInterfaces(compute.ServerNetworkInterface{Tag: empty})}, resource.ErrUnsupported},
		{"mode invalid major", "3.0", []compute.CreateServerOption{compute.WithNetworkMode("auto")}, resource.ErrUnsupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("compute", "/compute")
			client.Microversion = tc.version
			resolve := func(context.Context, resource.Ref) (string, error) {
				t.Error("preflight must precede lookup")
				return "", nil
			}
			service := compute.New(client, compute.Dependencies{Image: resolve, Volume: resolve, Network: resolve, Port: resolve})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Error("preflight made HTTP request")
				http.Error(w, "unexpected", 500)
			})
			request := nicCreateRequest()
			request.Image, request.Flavor = resource.Name("image"), resource.Name("flavor")
			if _, err := service.Servers.Create(context.Background(), request, tc.opts...); !errors.Is(err, tc.want) {
				t.Fatalf("error=%v want=%v", err, tc.want)
			}
		})
	}
}

func TestServerNICResolverFailuresAndCancellationStopBeforeNova(t *testing.T) {
	for _, kind := range []string{"network", "port"} {
		for _, scenario := range []string{"missing resolver", "empty ID", "unsafe ID", "cause", "cancelled after resolve"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				cause := errors.New("resolver failed")
				want := resource.ErrInvalidOption
				resolve := func(context.Context, resource.Ref) (string, error) {
					switch scenario {
					case "empty ID":
						return "", nil
					case "unsafe ID":
						return "bad/id", nil
					case "cause":
						return "", cause
					default:
						cancel()
						return "id", nil
					}
				}
				if scenario == "missing resolver" {
					resolve = nil
					want = resource.ErrUnsupported
				}
				if scenario == "cause" {
					want = cause
				}
				if scenario == "cancelled after resolve" {
					want = context.Canceled
				}
				nic := compute.ServerNetworkInterface{}
				deps := compute.Dependencies{}
				if kind == "network" {
					nic.Network = resource.Name("name")
					deps.Network = resolve
				} else {
					nic.Port = resource.Name("name")
					deps.Port = resolve
				}
				service := compute.New(cloud.Client("compute", "/compute"), deps)
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					t.Error("resolver failure made HTTP request")
					http.Error(w, "unexpected", 500)
				})
				if _, err := service.Servers.Create(ctx, nicCreateRequest(), compute.WithNetworkInterfaces(nic)); !errors.Is(err, want) {
					t.Fatalf("error=%v want=%v", err, want)
				}
			})
		}
	}
}

func TestServerNICsPreserveVolumeBootAndCreatedServerOnWaitFailure(t *testing.T) {
	for _, newVolume := range []bool{false, true} {
		t.Run(map[bool]string{false: "existing volume", true: "new image volume"}[newVolume], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("compute", "/compute")
			client.Microversion = "2.67"
			service := compute.New(client, compute.Dependencies{})
			cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Server map[string]any `json:"server"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				mapping := map[string]any{"source_type": "volume", "destination_type": "volume", "uuid": "root-volume", "boot_index": float64(0), "delete_on_termination": false}
				if newVolume {
					mapping["source_type"], mapping["uuid"], mapping["volume_size"], mapping["volume_type"] = "image", "image", float64(20), "fast"
				}
				want := map[string]any{"name": "nic-server", "imageRef": "", "flavorRef": "flavor", "networks": []any{map[string]any{"port": "port"}}, "block_device_mapping_v2": []any{mapping}}
				if !reflect.DeepEqual(body.Server, want) {
					t.Errorf("body=%#v want=%#v", body.Server, want)
				}
				testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
			})
			cloud.Mux.HandleFunc("GET /compute/servers/created", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"server":{"id":"created","status":"ERROR"}}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected lookup/cleanup: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			request := nicCreateRequest()
			opts := []compute.CreateServerOption{compute.WithNetworkInterfaces(compute.ServerNetworkInterface{Port: resource.ID("port")}), compute.WithWait(resource.WithTimeout(time.Second))}
			if newVolume {
				opts = append(opts, compute.WithBootVolumeSize(20), compute.WithBootVolumeType("fast"))
			} else {
				request.Image = resource.Ref{}
				opts = append(opts, compute.WithBootVolume(resource.ID("root-volume")))
			}
			row, err := service.Servers.Create(context.Background(), request, opts...)
			if !errors.Is(err, resource.ErrFailedState) || row == nil || row.ID != "created" || row.Status != "BUILD" {
				t.Fatalf("row=%v err=%v", row, err)
			}
		})
	}
}
