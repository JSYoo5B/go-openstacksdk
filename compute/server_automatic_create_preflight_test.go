package compute

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestAutomaticCreatePreparesClosuresOnceAndCapturesSourceBeforeOptions(t *testing.T) {
	for _, scenario := range []string{"once", "automatic endpoint", "server client", "resolver source", "resolver budget"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			service := New(cloud.Client("compute", "/v2.1"), Dependencies{})
			posts := 0
			cloud.Mux.HandleFunc("POST /v2.1/servers", func(w http.ResponseWriter, r *http.Request) {
				posts++
				testcloud.JSON(w, 202, `{"server":{"id":"server"}}`)
			})
			cloud.Mux.HandleFunc("GET /v2.1/servers/server", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"server":{"id":"server","status":"ACTIVE","addresses":{"net":[{"version":4,"addr":"10.0.0.1"}]}}}`)
			})
			count := make(map[string]int)
			o := AutomaticServerCreateOptions{Server: []CreateServerOption{WithNetworks(resource.ID("private")), func(o *createServerOptions) error {
				count["server"]++
				if scenario == "server client" {
					service.Servers.client = nil
				}
				return nil
			}}}
			// WaitOption's private destination is checked by the shared resource
			// tests; this fixture counts the SDK-owned create/address/IP options.
			o.Server = o.Server[:2]
			o.AutomaticIP = []AutomaticFloatingIPOption{func(o *automaticFloatingIPOptions) error {
				count["automatic"]++
				if scenario == "automatic endpoint" {
					service.client.Endpoint = cloud.Server.URL + "/changed/"
				}
				return nil
			}, WithAutomaticIPEnabled(false), WithAutomaticAddressOptions(func(o *serverAddressOptions) error { count["address"]++; return nil })}
			o.AutomaticIP = o.AutomaticIP[:3]
			request := CreateServerRequest{Name: "web", Image: resource.ID("image"), Flavor: resource.ID("flavor")}
			if scenario == "resolver source" || scenario == "resolver budget" {
				request.Image = resource.Name("image")
				service.Servers.dependencies.Image = func(ctx context.Context, ref resource.Ref) (string, error) {
					if scenario == "resolver source" {
						service.API = nil
						return "image", nil
					}
					<-ctx.Done()
					return "", ctx.Err()
				}
				o.AutomaticIP = append(o.AutomaticIP, WithAutomaticIPTimeout(10*time.Millisecond))
			}
			result, err := service.CreateWithAutomaticFloatingIP(context.Background(), request, o)
			if scenario == "once" {
				if err != nil || result == nil || posts != 1 || count["server"] != 1 || count["automatic"] != 1 || count["address"] != 1 {
					t.Fatal(result, err, posts, count)
				}
				return
			}
			if result != nil || err == nil || posts != 0 {
				t.Fatal(result, err, posts)
			}
			if scenario == "resolver budget" {
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}
