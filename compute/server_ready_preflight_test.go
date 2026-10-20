package compute

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestServerReadyPreparesOptionsOnceAndCapturesIdentityAndSource(t *testing.T) {
	for _, scenario := range []string{"once", "ready source", "automatic source", "address source"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			service := New(cloud.Client("compute", "/v2.1"), Dependencies{})
			reads := 0
			cloud.Mux.HandleFunc("GET /v2.1/servers/original", func(w http.ResponseWriter, r *http.Request) {
				reads++
				testcloud.JSON(w, 200, `{"server":{"id":"original","status":"ACTIVE","addresses":{"net":[{"version":4,"addr":"10.0.0.1"}]}}}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Error("unexpected", r.URL)
				http.Error(w, "unexpected", 500)
			})
			server := &Server{ID: "original", Status: "BUILD"}
			counts := map[string]int{}
			opts := []ServerReadyOption{func(o *serverReadyOptions) error {
				counts["ready"]++
				server.ID, server.Status = "changed", "ERROR"
				if scenario == "ready source" {
					service.API = nil
				}
				return nil
			}, WithServerReadyAutomaticIPOptions(WithAutomaticIPEnabled(false), func(o *automaticFloatingIPOptions) error {
				counts["automatic"]++
				if scenario == "automatic source" {
					service.client.Endpoint = cloud.Server.URL + "/replacement/"
				}
				return nil
			}, WithAutomaticAddressOptions(func(o *serverAddressOptions) error {
				counts["address"]++
				if scenario == "address source" {
					service.Servers.Collection = nil
				}
				return nil
			}))}
			result, err := service.WaitForServer(context.Background(), AutomaticFloatingIPRequest{Server: server}, opts...)
			if scenario == "once" {
				if err != nil || result == nil || result.Server.ID != "original" || result.Server.Status != "ACTIVE" || reads != 1 || counts["ready"] != 1 || counts["automatic"] != 1 || counts["address"] != 1 {
					t.Fatal(result, err, reads, counts)
				}
			} else if result != nil || !errors.Is(err, resource.ErrInvalidOption) || reads != 0 {
				t.Fatal(result, err, reads, counts)
			}
		})
	}
}
