package compute_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"gophercloudsdk/compute"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

func TestServerAddressExpansionDefaultPrivateIPv6AndOwnedSnapshot(t *testing.T) {
	for _, scenario := range []struct {
		name                                              string
		options                                           []compute.ServerAddressOption
		defaultNetwork, wantInterface, wantAccess, wantV6 string
	}{
		{"default6", []compute.ServerAddressOption{compute.WithLocalIPv6(true)}, "default", "fd00::1", "8.8.8.8", "2001:db8::2"},
		{"default4 forced", []compute.ServerAddressOption{compute.WithLocalIPv6(true), compute.WithForceIPv4(true)}, "default", "10.1.0.1", "8.8.8.8", ""},
		{"private after missing default", []compute.ServerAddressOption{compute.WithPrivateCloud(true), compute.WithLocalIPv6(true)}, "", "10.0.0.1", "10.0.0.1", "2001:db8::2"},
		{"public6 locally usable", []compute.ServerAddressOption{compute.WithLocalIPv6(true)}, "", "2001:db8::2", "8.8.8.8", "2001:db8::2"},
		{"public4 locally disabled", []compute.ServerAddressOption{compute.WithLocalIPv6(false)}, "", "8.8.8.8", "8.8.8.8", "2001:db8::2"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			row := addressRow(4, "10.0.0.1", "fixed", "mac")
			row["vendor:field"] = map[string]any{"value": []any{1, 2}}
			server := &compute.Server{ID: "server", Status: "BUILD", AccessIPv4: "8.8.8.8", AccessIPv6: "2001:db8::2", Addresses: map[string]any{
				"inside": []any{row}, "default": []any{addressRow(4, "10.1.0.1", "fixed"), addressRow(6, "fd00::1", "fixed")}, "null": nil, "empty": []any{},
			}}
			before, _ := json.Marshal(server.Addresses)
			calls := 0
			service := compute.New(nil, compute.Dependencies{NetworkRoles: func(context.Context) (*network.NetworkRoleSnapshot, error) {
				calls++
				roles := &network.NetworkRoleSnapshot{InternalIPv4: []*network.RoleNetwork{addressRole("inside")}}
				if scenario.defaultNetwork != "" {
					roles.DefaultNetwork = addressRole(scenario.defaultNetwork)
				}
				return roles, nil
			}})
			options := append(scenario.options, compute.WithAddressReachability(false))
			view, err := service.ExpandServerInterfaces(context.Background(), server, options...)
			if err != nil || view.InterfaceIP != scenario.wantInterface || view.AccessIPv4 != scenario.wantAccess || view.PublicIPv6 != scenario.wantV6 || view.PrivateIPv4 != "10.0.0.1" || calls != 1 {
				t.Fatalf("view=%+v err=%v calls=%d", view, err, calls)
			}
			if view.Addresses["null"] != nil || view.Addresses["empty"] == nil {
				t.Fatalf("null/empty lost: %#v", view.Addresses)
			}
			view.Addresses["inside"][0].Fields["vendor:field"][0] = 'X'
			view.Addresses["inside"][0].Address = "changed"
			after, _ := json.Marshal(server.Addresses)
			if string(before) != string(after) || server.AccessIPv4 != "8.8.8.8" {
				t.Fatal("view mutated native input")
			}
		})
	}
}

func TestServerAddressExpansionEmptyDefaultStopsFamilyBeforeInterfaceFallback(t *testing.T) {
	service := compute.New(nil, compute.Dependencies{NetworkRoles: func(context.Context) (*network.NetworkRoleSnapshot, error) {
		return &network.NetworkRoleSnapshot{DefaultNetwork: addressRole("")}, nil
	}})
	server := &compute.Server{AccessIPv4: "8.8.8.8", Addresses: map[string]any{"": []any{addressRow(6, "", "fixed"), addressRow(4, "10.0.0.1", "fixed")}}}
	view, err := service.ExpandServerInterfaces(context.Background(), server, compute.WithLocalIPv6(true), compute.WithAddressReachability(false), compute.WithAddressNetworkOrder(""))
	if err != nil || view.InterfaceIP != "8.8.8.8" {
		t.Fatalf("empty IPv6 must stop default4 and use public fallback: view=%+v err=%v", view, err)
	}
	server.Addresses = nil
	view, err = service.ExpandServerInterfaces(context.Background(), server)
	if err != nil || view.Addresses != nil {
		t.Fatalf("nil addresses lost: %+v %v", view, err)
	}
	server.Addresses = map[string]any{}
	view, err = service.ExpandServerInterfaces(context.Background(), server)
	if err != nil || view.Addresses == nil {
		t.Fatalf("empty addresses lost: %+v %v", view, err)
	}
}

func TestServerAddressSupplementGatesSkipNetworkAndOnlyNonIPv6Floating(t *testing.T) {
	for _, scenario := range []struct {
		name, status string
		row          map[string]any
		options      []compute.ServerAddressOption
		wantCalls    int
	}{
		{"BUILD", "BUILD", addressRow(4, "10.0.0.1", "fixed"), nil, 0},
		{"case sensitive ACTIVE", "active", addressRow(4, "10.0.0.1", "fixed"), nil, 0},
		{"disabled", "ACTIVE", addressRow(4, "10.0.0.1", "fixed"), []compute.ServerAddressOption{compute.WithFloatingIPSource(compute.FloatingIPNone)}, 0},
		{"existing floating4", "ACTIVE", addressRow(4, "8.8.8.8", "floating"), nil, 0},
		{"existing floating non6", "ACTIVE", addressRow(5, "8.8.8.8", "floating"), nil, 0},
		{"floating6 still supplements", "ACTIVE", addressRow(6, "2001:db8::1", "floating"), nil, 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			calls := 0
			service := compute.New(nil, compute.Dependencies{AddressNetworks: func(context.Context) (*network.Service, error) { calls++; return nil, nil }})
			view, err := service.ExpandServerInterfaces(context.Background(), &compute.Server{ID: "server", Status: scenario.status, Addresses: map[string]any{"private": []any{scenario.row}}}, append(scenario.options, compute.WithAddressReachability(false))...)
			if view == nil || err != nil || calls != scenario.wantCalls {
				t.Fatalf("view=%+v err=%v calls=%d", view, err, calls)
			}
		})
	}
}

func addressSupplementService(cloud *testcloud.Cloud) (*compute.Service, *network.Service) {
	netService := network.New(cloud.Client("network", "/network"))
	return compute.New(cloud.Client("compute", "/compute"), compute.Dependencies{AddressNetworks: func(context.Context) (*network.Service, error) { return netService, nil }}), netService
}

func TestServerAddressSupplementPagesScopeMACAndDuplicateFixedNetwork(t *testing.T) {
	cloud := testcloud.New(t)
	var paths []string
	cloud.Mux.HandleFunc("GET /network/ports", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		if r.URL.Query().Get("device_id") != "server" {
			t.Error(r.URL)
		}
		if r.URL.Query().Get("marker") == "" {
			testcloud.JSON(w, 200, `{"ports":[],"ports_links":[{"rel":"next","href":"?device_id=server&marker=next"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"ports":[{"id":"ignored","device_id":"another"},{"id":"port","device_id":"server","mac_address":"mac"}]}`)
	})
	cloud.Mux.HandleFunc("GET /network/floatingips", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		if r.URL.Query().Get("port_id") != "port" {
			t.Error(r.URL)
		}
		if r.URL.Query().Get("marker") == "" {
			testcloud.JSON(w, 200, `{"floatingips":[],"floatingips_links":[{"rel":"next","href":"?port_id=port&marker=next"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"floatingips":[{"id":"wrong-port","port_id":"another","fixed_ip_address":"10.0.0.1","floating_ip_address":"9.9.9.9"},{"id":"missing-fixed","port_id":"port","fixed_ip_address":"10.9.0.1","floating_ip_address":"9.9.9.9"},{"id":"good","port_id":"port","fixed_ip_address":"10.0.0.1","floating_ip_address":"8.8.8.8","status":"DOWN"}]}`)
	})
	service, _ := addressSupplementService(cloud)
	server := &compute.Server{ID: "server", Status: "ACTIVE", Addresses: map[string]any{"a": []any{addressRow(4, "10.0.0.1", "fixed")}, "b": []any{addressRow(4, "10.0.0.1", "fixed", "mac")}}}
	view, err := service.ExpandServerInterfaces(context.Background(), server, compute.WithAddressReachability(false))
	if err != nil || view.SupplementalError != nil || view.PublicIPv4 != "8.8.8.8" || len(view.Addresses["a"]) != 1 || len(view.Addresses["b"]) != 2 {
		t.Fatalf("view=%+v err=%v", view, err)
	}
	row := view.Addresses["b"][1]
	if row.Version != 4 || row.Type != "floating" || row.MACAddress != "mac" || !row.MACPresent || !row.Supplemental {
		t.Fatalf("synthetic row=%+v", row)
	}
	if len(server.Addresses["b"].([]any)) != 1 || len(paths) != 4 {
		t.Fatalf("input/request scope changed: paths=%v", paths)
	}
}

func TestServerAddressSupplementNovaSourceLocalFilterAliasesAnd404(t *testing.T) {
	for _, scenario := range []string{"extensions and canonical null", "missing Nova API"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("GET /network/ports", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"ports":[{"id":"port1","device_id":"server","mac_address":"mac1"},{"id":"port2","device_id":"server","mac_address":"mac2"}]}`)
			})
			calls := 0
			cloud.Mux.HandleFunc("GET /compute/os-floating-ips", func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.RawQuery != "" {
					t.Error("Nova received Neutron filters", r.URL)
				}
				if scenario == "missing Nova API" {
					http.Error(w, "missing", 404)
					return
				}
				testcloud.JSON(w, 200, `{"floating_ips":[{"id":1,"port_id":"port1","fixed_ip":"10.0.0.1","ip":"8.8.8.8"},{"id":2,"port_id":"port2","fixed_ip":"10.0.0.1","fixed_ip_address":null,"ip":"9.9.9.9"},{"id":3,"ip":"1.1.1.1","fixed_ip":"10.0.0.1"}]}`)
			})
			cloud.Mux.HandleFunc("GET /network/floatingips", func(w http.ResponseWriter, r *http.Request) { t.Fatal("Nova source used Neutron floating IPs") })
			service, _ := addressSupplementService(cloud)
			server := &compute.Server{ID: "server", Status: "ACTIVE", Addresses: map[string]any{"private": []any{addressRow(4, "10.0.0.1", "fixed", "mac1")}}}
			view, err := service.ExpandServerInterfaces(context.Background(), server, compute.WithFloatingIPSource(compute.FloatingIPNova), compute.WithAddressReachability(false))
			wantIP, wantRows := "8.8.8.8", 2
			if scenario == "missing Nova API" {
				wantIP, wantRows = "", 1
			}
			if err != nil || view.SupplementalError != nil || view.PublicIPv4 != wantIP || len(view.Addresses["private"]) != wantRows || calls != 2 {
				t.Fatalf("view=%+v err=%v calls=%d", view, err, calls)
			}
		})
	}
}

func TestServerAddressSupplementPartialCleanHTTPFailureAndTerminal204(t *testing.T) {
	for _, status := range []int{403, 204} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("GET /network/ports", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"ports":[{"id":"one","device_id":"server"},{"id":"two","device_id":"server"}]}`)
			})
			cloud.Mux.HandleFunc("GET /network/floatingips", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("port_id") == "one" {
					testcloud.JSON(w, 200, `{"floatingips":[{"port_id":"one","fixed_ip_address":"10.0.0.1","floating_ip_address":"8.8.8.8"}]}`)
					return
				}
				if status == 204 {
					w.Header().Set("Link", `</outside>; rel="next"`)
					w.WriteHeader(204)
					return
				}
				http.Error(w, "optional lookup denied", status)
			})
			service, _ := addressSupplementService(cloud)
			view, err := service.ExpandServerInterfaces(context.Background(), &compute.Server{ID: "server", Status: "ACTIVE", Addresses: map[string]any{"private": []any{addressRow(4, "10.0.0.1", "fixed")}}}, compute.WithAddressReachability(false))
			if err != nil || view.PublicIPv4 != "8.8.8.8" || len(view.Addresses["private"]) != 2 || (view.SupplementalError != nil) != (status == 403) {
				t.Fatalf("view=%+v err=%v", view, err)
			}
		})
	}
}

type addressRoundTripper func(*http.Request) (*http.Response, error)

func (f addressRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type addressCloseBody struct {
	io.ReadCloser
	close func() error
}

func (b addressCloseBody) Close() error { _ = b.ReadCloser.Close(); return b.close() }

func TestServerAddressSupplementAcceptedFailuresKeepPartialAndGuardComputeOwner(t *testing.T) {
	for _, scenario := range []string{"malformed", "close error", "network source", "compute collection", "canceled"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			service, netService := addressSupplementService(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("accepted close failed")
			calls := 0
			cloud.Provider.HTTPClient.Transport = addressRoundTripper(func(r *http.Request) (*http.Response, error) {
				calls++
				body := `{"ports":[{"id":"one","device_id":"server"},{"id":"two","device_id":"server"}]}`
				if strings.HasSuffix(r.URL.Path, "floatingips") {
					body = `{"floatingips":[{"port_id":"one","fixed_ip_address":"10.0.0.1","floating_ip_address":"8.8.8.8"}]}`
					if r.URL.Query().Get("port_id") == "two" {
						body = `{"floatingips":[]}`
						if scenario == "malformed" {
							body = `{"floatingips":`
						}
					}
				}
				response := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}
				if r.URL.Query().Get("port_id") == "two" && scenario != "malformed" {
					response.Body = addressCloseBody{ReadCloser: response.Body, close: func() error {
						switch scenario {
						case "close error":
							return cause
						case "network source":
							netService.RawClient().Endpoint += "changed/"
						case "compute collection":
							service.Servers = nil
						case "canceled":
							cancel(cause)
						}
						return nil
					}}
				}
				return response, nil
			})
			server := &compute.Server{ID: "server", Status: "ACTIVE", Addresses: map[string]any{"private": []any{addressRow(4, "10.0.0.1", "fixed")}}}
			view, err := service.ExpandServerInterfaces(ctx, server, compute.WithAddressReachability(false))
			if err == nil || view == nil || view.SupplementalError != nil || len(view.Addresses["private"]) != 2 || calls != 3 || len(server.Addresses["private"].([]any)) != 1 {
				t.Fatalf("view=%+v err=%v calls=%d", view, err, calls)
			}
			if scenario == "close error" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if scenario == "canceled" && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
				t.Fatal(err)
			}
			if (scenario == "network source" || scenario == "compute collection") && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestServerAddressEmptyCandidateDoesNotFallThroughRoleOrFloatingPriority(t *testing.T) {
	service := compute.New(nil, compute.Dependencies{NetworkRoles: func(context.Context) (*network.NetworkRoleSnapshot, error) {
		return &network.NetworkRoleSnapshot{ExternalIPv4: []*network.RoleNetwork{addressRole("first"), addressRole("second")}}, nil
	}})
	server := &compute.Server{Addresses: map[string]any{"first": []any{addressRow(4, "", "fixed")}, "second": []any{addressRow(4, "8.8.8.8", "fixed")}}}
	got, err := service.GetServerPublicIP(context.Background(), server)
	if err != nil || got != "" {
		t.Fatalf("empty first role fell through: %q %v", got, err)
	}
	server.Addresses = map[string]any{"unknown": []any{addressRow(4, "", "floating")}, "public": []any{addressRow(4, "8.8.8.8", "fixed")}}
	got, err = compute.New(nil, compute.Dependencies{}).GetServerPublicIP(context.Background(), server)
	if err != nil || got != "" {
		t.Fatalf("empty floating fell through: %q %v", got, err)
	}
	if reflect.DeepEqual(server.Addresses, map[string]any{}) {
		t.Fatal("input cleared")
	}
}

func TestServerAddressSupplementPortMACMissingNullAndEmptyAreDistinct(t *testing.T) {
	for _, scenario := range []struct {
		name, macJSON, wantPrivate string
		present                    bool
	}{
		{"missing", "", "10.0.0.1", false}, {"null", `,"mac_address":null`, "10.0.0.1", false}, {"empty", `,"mac_address":""`, "8.8.8.8", true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("GET /network/ports", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"ports":[{"id":"port","device_id":"server"`+scenario.macJSON+`}]}`)
			})
			cloud.Mux.HandleFunc("GET /network/floatingips", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"floatingips":[{"port_id":"port","fixed_ip_address":"10.0.0.1","floating_ip_address":"8.8.8.8"}]}`)
			})
			netService := network.New(cloud.Client("network", "/network"))
			service := compute.New(cloud.Client("compute", "/compute"), compute.Dependencies{AddressNetworks: func(context.Context) (*network.Service, error) { return netService, nil }, NetworkRoles: func(context.Context) (*network.NetworkRoleSnapshot, error) {
				return &network.NetworkRoleSnapshot{InternalIPv4: []*network.RoleNetwork{addressRole("inside")}}, nil
			}})
			view, err := service.ExpandServerInterfaces(context.Background(), &compute.Server{ID: "server", Status: "ACTIVE", Addresses: map[string]any{"inside": []any{addressRow(4, "10.0.0.1", "fixed")}}}, compute.WithAddressReachability(false))
			if err != nil || view.PrivateIPv4 != scenario.wantPrivate || view.Addresses["inside"][1].MACPresent != scenario.present {
				t.Fatalf("view=%+v err=%v", view, err)
			}
		})
	}
}
