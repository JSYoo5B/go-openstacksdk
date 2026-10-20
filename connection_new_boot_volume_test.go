package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestNegotiatedMicroversionControlsBootVolumeType(t *testing.T) {
	for _, maximum := range []string{"2.9", "2.100"} {
		t.Run(maximum, func(t *testing.T) {
			cloud := testcloud.New(t)
			var discoveries, creates atomic.Int32
			cloud.Mux.HandleFunc("GET /nova/v2.1/{$}", func(w http.ResponseWriter, r *http.Request) {
				discoveries.Add(1)
				testcloud.JSON(w, 200, `{"version":{"id":"v2.1","min_version":"2.1","version":"`+maximum+`"}}`)
			})
			cloud.Mux.HandleFunc("POST /nova/v2.1/project/servers", func(w http.ResponseWriter, r *http.Request) {
				creates.Add(1)
				if r.Header.Get("OpenStack-API-Version") != "compute "+maximum {
					t.Error(r.Header)
				}
				var body struct {
					Server struct {
						Mapping []struct {
							VolumeType string `json:"volume_type"`
							Size       int    `json:"volume_size"`
						} `json:"block_device_mapping_v2"`
					} `json:"server"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if len(body.Server.Mapping) != 1 || body.Server.Mapping[0].VolumeType != "fast" || body.Server.Mapping[0].Size != 20 {
					t.Errorf("body=%+v", body)
				}
				testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
			})
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/nova/v2.1/project"), sdk.WithLatestMicroversion(sdk.Compute))
			if err != nil {
				t.Fatal(err)
			}
			service, err := conn.Compute(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			server, err := service.Servers.Create(context.Background(), compute.CreateServerRequest{Name: "server", Image: resource.ID("image"), Flavor: resource.ID("flavor")}, compute.WithBootVolumeSize(20), compute.WithBootVolumeType("fast"))
			if maximum == "2.9" {
				if server != nil || !errors.Is(err, resource.ErrUnsupported) || creates.Load() != 0 {
					t.Fatalf("server=%v err=%v creates=%d", server, err, creates.Load())
				}
			} else if err != nil || server == nil || server.ID != "created" || creates.Load() != 1 {
				t.Fatalf("server=%v err=%v creates=%d", server, err, creates.Load())
			}
			if discoveries.Load() != 1 {
				t.Fatalf("discoveries=%d", discoveries.Load())
			}
		})
	}
}
