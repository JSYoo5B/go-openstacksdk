package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"gophercloudsdk/compute"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestConnectionBootVolumeResolution(t *testing.T) {
	for _, tc := range []struct {
		name      string
		ref       resource.Ref
		volumes   string
		wantLists int32
		wantErr   error
	}{
		{name: "named Cinder volume", ref: resource.Name("root[1]"), volumes: `{"volumes":[{"id":"other","name":"root[1]-copy"},{"id":"boot-id","name":"root[1]"}]}`, wantLists: 1},
		{name: "explicit ID bypasses Cinder", ref: resource.ID("boot-id")},
		{name: "duplicate name prevents creation", ref: resource.Name("root[1]"), volumes: `{"volumes":[{"id":"one","name":"root[1]"},{"id":"two","name":"root[1]"}]}`, wantLists: 1, wantErr: resource.ErrAmbiguous},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, creates atomic.Int32
			cloud.Mux.HandleFunc("GET /volume/v3/project/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("name") != "root[1]" {
					t.Errorf("query=%s", r.URL.RawQuery)
				}
				testcloud.JSON(w, 200, tc.volumes)
			})
			cloud.Mux.HandleFunc("POST /compute/v2.1/project/servers", func(w http.ResponseWriter, r *http.Request) {
				creates.Add(1)
				var body struct {
					Server struct {
						Mapping []struct {
							UUID string `json:"uuid"`
						} `json:"block_device_mapping_v2"`
					} `json:"server"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if len(body.Server.Mapping) != 1 || body.Server.Mapping[0].UUID != "boot-id" {
					t.Errorf("body=%+v", body)
				}
				testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
			})
			service, err := connection(t, cloud).Compute(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			server, err := service.Servers.Create(context.Background(), compute.CreateServerRequest{Name: "server", Flavor: resource.ID("flavor-id")}, compute.WithBootVolume(tc.ref))
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("server=%v err=%v", server, err)
			}
			if lists.Load() != tc.wantLists {
				t.Fatalf("lists=%d", lists.Load())
			}
			if tc.wantErr != nil {
				if server != nil || creates.Load() != 0 {
					t.Fatalf("server=%v creates=%d", server, creates.Load())
				}
			} else if server == nil || server.ID != "created" || creates.Load() != 1 {
				t.Fatalf("server=%v creates=%d", server, creates.Load())
			}
		})
	}
}
