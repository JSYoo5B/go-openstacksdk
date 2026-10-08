package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func connectionAutomaticCreateOptions() compute.AutomaticServerCreateOptions {
	return compute.AutomaticServerCreateOptions{Server: []compute.CreateServerOption{compute.WithNetworks(resource.ID("private")), compute.WithWait(resource.WithPollInterval(time.Millisecond))}, AutomaticIP: []compute.AutomaticFloatingIPOption{compute.WithAutomaticIPEnabled(false), compute.WithAutomaticIPTimeout(time.Second)}}
}

func TestConnectionAutomaticCreateOwnsNamedDependencyPagesAndLifetime(t *testing.T) {
	for _, scenario := range []string{"image", "network", "port", "volume", "flavor", "retry compute source", "later image source", "nil image API"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute"), sdk.WithEndpoint(sdk.Image, cloud.Server.URL+"/image/v2"), sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network"), sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/volume/v3"))
			if err != nil {
				t.Fatal(err)
			}
			request := compute.CreateServerRequest{Name: "web", Image: resource.ID("image"), Flavor: resource.ID("flavor")}
			o := connectionAutomaticCreateOptions()
			path, plural, id := "/image/v2/images", "images", "resolved-image"
			switch scenario {
			case "network":
				path, plural, id = "/network/v2.0/networks", "networks", "resolved-network"
				o.Server[0] = compute.WithNetworks(resource.Name("wanted"))
			case "port":
				path, plural, id = "/network/v2.0/ports", "ports", "resolved-port"
				o.Server[0] = compute.WithNetworkInterfaces(compute.ServerNetworkInterface{Port: resource.Name("wanted")})
			case "volume":
				path, plural, id = "/volume/v3/volumes/detail", "volumes", "resolved-volume"
				request.Image = resource.Ref{}
				o.Server = append(o.Server, compute.WithBootVolume(resource.Name("wanted")))
			case "flavor":
				path, plural, id = "/compute/flavors/detail", "flavors", "resolved-flavor"
				request.Flavor = resource.Name("wanted")
			default:
				request.Image = resource.Name("wanted")
			}
			var lists, posts, gets atomic.Int32
			service, _ := conn.Compute(context.Background())
			if scenario == "nil image API" {
				image, _ := conn.Image(context.Background())
				image.API = nil
			}
			if scenario == "later image source" {
				request.Flavor = resource.Name("small")
				cloud.Mux.HandleFunc("GET /compute/flavors/detail", func(w http.ResponseWriter, r *http.Request) {
					image, _ := conn.Image(context.Background())
					image.API = nil
					testcloud.JSON(w, 200, `{"flavors":[{"id":"flavor","name":"small"}]}`)
				})
			}
			cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if scenario == "retry compute source" {
					testcloud.JSON(w, 503, `{"error":"unavailable"}`)
					return
				}
				if plural == "flavors" {
					if r.URL.Query().Has("name") || r.URL.Query().Has("is_public") {
						t.Error(r.URL)
					}
				} else if r.URL.Query().Get("name") != "wanted" {
					t.Error(r.URL)
				}
				if r.URL.Query().Get("marker") != "" {
					testcloud.JSON(w, 200, `{"`+plural+`":[]}`)
					return
				}
				testcloud.JSON(w, 200, fmt.Sprintf(`{"%s":[{"id":%q,"name":"wanted"}],"%s_links":[{"rel":"next","href":%q}]}`, plural, id, plural, cloud.Server.URL+path+"?marker=next"))
			})
			if scenario == "retry compute source" {
				cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					service.RawClient().Endpoint = cloud.Server.URL + "/changed/"
					return nil
				}
			}
			cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				var body struct{ Server map[string]any }
				_ = json.NewDecoder(r.Body).Decode(&body)
				if plural == "images" && body.Server["imageRef"] != id {
					t.Error(body)
				}
				if plural == "flavors" && body.Server["flavorRef"] != id {
					t.Error(body)
				}
				testcloud.JSON(w, 202, `{"server":{"id":"created"}}`)
			})
			cloud.Mux.HandleFunc("GET /compute/servers/created", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 200, `{"server":{"id":"created","status":"ACTIVE","addresses":{"private":[{"version":4,"addr":"10.0.0.10"}]}}}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Error("unexpected", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			result, err := conn.CreateWithAutomaticFloatingIP(context.Background(), request, o)
			if scenario == "retry compute source" || scenario == "later image source" || scenario == "nil image API" {
				if result != nil || !errors.Is(err, resource.ErrInvalidOption) || posts.Load() != 0 || gets.Load() != 0 {
					t.Fatal(result, err, posts.Load(), gets.Load())
				}
				if scenario == "retry compute source" && lists.Load() != 1 {
					t.Fatal(lists.Load())
				}
				return
			}
			if err != nil || result == nil || result.Automatic == nil || result.Automatic.Decision.Reason != compute.AutomaticIPDisabled || lists.Load() != 2 || posts.Load() != 1 || gets.Load() != 1 {
				t.Fatal(result, err, lists.Load(), posts.Load(), gets.Load())
			}
		})
	}
}

func TestConnectionAutomaticCreateSharesConfiguredDefaultRoleSnapshot(t *testing.T) {
	cloud := testcloud.New(t)
	var roles, posts, gets atomic.Int32
	cloud.Mux.HandleFunc("GET /network/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		roles.Add(1)
		if posts.Load() != 0 {
			t.Error("role snapshot replaced after creation")
		}
		testcloud.JSON(w, 200, `{"networks":[{"id":"private","name":"private"}]}`)
	})
	cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		var body struct {
			Server struct{ Networks []map[string]any }
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if len(body.Server.Networks) != 1 || body.Server.Networks[0]["uuid"] != "private" {
			t.Error(body)
		}
		testcloud.JSON(w, 202, `{"server":{"id":"created"}}`)
	})
	cloud.Mux.HandleFunc("GET /compute/servers/created", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		testcloud.JSON(w, 200, `{"server":{"id":"created","status":"ACTIVE","addresses":{"private":[{"version":4,"addr":"8.8.8.8","OS-EXT-IPS:type":"floating"}]}}}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute"), sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network"), sdk.WithNetworkRoles(network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "private", DefaultInterface: true, NATDestination: true})))
	if err != nil {
		t.Fatal(err)
	}
	o := connectionAutomaticCreateOptions()
	o.Server = o.Server[1:]
	o.AutomaticIP = []compute.AutomaticFloatingIPOption{compute.WithAutomaticIPTimeout(time.Second)}
	result, err := conn.CreateWithAutomaticFloatingIP(context.Background(), compute.CreateServerRequest{Name: "web", Image: resource.ID("image"), Flavor: resource.ID("flavor")}, o)
	if err != nil || result == nil || result.Automatic.Decision.Reason != compute.AutomaticIPExistingFloating || roles.Load() != 1 || posts.Load() != 1 || gets.Load() != 1 {
		t.Fatal(result, err, roles.Load(), posts.Load(), gets.Load())
	}
}
