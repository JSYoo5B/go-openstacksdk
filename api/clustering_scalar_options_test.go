package api_test

import (
	"context"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestClusteringScalarOptionsPreserveExplicitValuesAndReuse(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/v1")
	client.Microversion = "1.13"
	var calls atomic.Int32
	want := map[string]string{
		"POST /v1/clusters":           `{"cluster":{"desired_capacity":0,"max_size":-1,"min_size":0,"name":"workers","profile_id":"profile","timeout":null}}`,
		"PATCH /v1/clusters/selected": `{"cluster":{"name":null,"profile_id":"changed","timeout":0}}`,
		"POST /v1/nodes":              `{"node":{"cluster_id":null,"name":"worker","profile_id":"profile","role":""}}`,
		"PATCH /v1/nodes/selected":    `{"node":{"name":"renamed","profile_id":null,"role":null,"tainted":false}}`,
	}
	for route, expected := range want {
		cloud.Mux.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != expected {
				t.Error(string(body), expected, err)
			}
			w.Header().Set("Location", "actions/accepted")
			code, response := http.StatusAccepted, `{"node":{"id":"accepted-node"}}`
			if r.URL.Path == "/v1/clusters" {
				code, response = http.StatusCreated, `{"cluster":{"id":"accepted-cluster"}}`
			}
			if r.URL.Path == "/v1/clusters/selected" {
				response = `{"cluster":{"id":"accepted-cluster"}}`
			}
			testcloud.JSON(w, code, response)
		})
	}
	clusterAPI, nodeAPI := clusters.New(client), nodes.New(client)
	var captured *request.Config[clusters.CreateOpts]
	createCluster := []clusters.CreateOption{clusters.WithCreateMinSize(0), clusters.WithCreateMaxSize(-1), clusters.WithCreateDesiredCapacity(0), clusters.WithCreateTimeoutNull(), func(config *request.Config[clusters.CreateOpts]) error { captured = config; return nil }}
	updateCluster := []clusters.UpdateOption{clusters.WithUpdateNameNull(), clusters.WithUpdateProfileID("changed"), clusters.WithUpdateTimeout(0)}
	createNode := []nodes.CreateOption{nodes.WithCreateClusterIDNull(), nodes.WithCreateRole("")}
	updateNode := []nodes.UpdateOption{nodes.WithUpdateName("renamed"), nodes.WithUpdateProfileIDNull(), nodes.WithUpdateRoleNull(), nodes.WithUpdateTainted(false)}
	for range 2 {
		if _, err := clusterAPI.Create(context.Background(), clusters.CreateOpts{Name: "workers", ProfileID: "profile"}, createCluster...); err != nil {
			t.Fatal(err)
		}
		// A retained custom-option config must not change the next use of a setter.
		*captured.Options.MinSize, *captured.Options.MaxSize, *captured.Options.DesiredCapacity = 9, 9, 9
		if _, err := clusterAPI.Update(context.Background(), resource.ID("selected"), clusters.UpdateOpts{}, updateCluster...); err != nil {
			t.Fatal(err)
		}
		if _, err := nodeAPI.Create(context.Background(), nodes.CreateOpts{Name: "worker", ProfileID: "profile"}, createNode...); err != nil {
			t.Fatal(err)
		}
		if _, err := nodeAPI.Update(context.Background(), resource.ID("selected"), nodes.UpdateOpts{}, updateNode...); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 8 {
		t.Fatal(calls.Load())
	}
}
