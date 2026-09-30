package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"gophercloudsdk/compute"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestCreateResolvesDependenciesAndSerializesOptions(t *testing.T) {
	cloud := testcloud.New(t)
	service, _ := connection(t, cloud).Compute(context.Background())
	cloud.Mux.HandleFunc("GET /image/v2/images", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"images":[{"id":"image-id","name":"ubuntu"}]}`)
	})
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/flavors/detail", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"flavors":[{"id":"flavor-id","name":"small"}]}`)
	})
	cloud.Mux.HandleFunc("GET /network/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"networks":[{"id":"network-id","name":"private"}]}`)
	})
	cloud.Mux.HandleFunc("POST /compute/v2.1/project/servers", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Server map[string]any `json:"server"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		for key, want := range map[string]any{"name": "web", "imageRef": "image-id", "flavorRef": "flavor-id", "key_name": "ssh-key", "config_drive": false, "user_data": "aGVsbG8="} {
			if body.Server[key] != want {
				t.Errorf("%s=%v want=%v", key, body.Server[key], want)
			}
		}
		if body.Server["metadata"].(map[string]any)["team"] != "infra" {
			t.Error("metadata snapshot lost")
		}
		if body.Server["vendor_hint"].(map[string]any)["zone"] != "fast" {
			t.Error("extension snapshot lost")
		}
		nets := body.Server["networks"].([]any)
		if len(nets) != 1 || nets[0].(map[string]any)["uuid"] != "network-id" {
			t.Error("network name was not resolved")
		}
		testcloud.JSON(w, 202, `{"server":{"id":"created","name":"web","status":"BUILD"}}`)
	})
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/created", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"server":{"id":"created","name":"web","status":"ACTIVE"}}`)
	})
	metadata := map[string]string{"team": "infra"}
	field := map[string]any{"zone": "fast"}
	metadataOption := compute.WithMetadata(metadata)
	fieldOption := compute.WithField("vendor_hint", field)
	metadata["team"] = "mutated"
	field["zone"] = "mutated"
	created, err := service.Servers.Create(context.Background(), compute.CreateServerRequest{Name: "web", Image: resource.Name("ubuntu"), Flavor: resource.Name("small")},
		metadataOption, fieldOption, compute.WithKeyName("ssh-key"), compute.WithConfigDrive(false), compute.WithUserData([]byte("hello")), compute.WithNetworks(resource.Name("private")), compute.WithWait())
	if err != nil || created.Status != "ACTIVE" {
		t.Fatalf("created=%v err=%v", created, err)
	}
}

func TestCreateValidationBeforeNetworkAndExplicitIDs(t *testing.T) {
	cloud := testcloud.New(t)
	service, _ := connection(t, cloud).Compute(context.Background())
	var requests atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != "POST" || r.URL.Path != "/compute/v2.1/project/servers" {
			t.Errorf("IDs must skip lookups: %s %s", r.Method, r.URL)
		}
		testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
	})
	request := compute.CreateServerRequest{Name: "web", Image: resource.ID("image-id"), Flavor: resource.ID("flavor-id")}
	for _, option := range []compute.CreateServerOption{
		compute.WithField("name", "override"), compute.WithField("key_name", "override"), compute.WithField("bad", make(chan int)),
		compute.WithNetworks(resource.Ref{}), compute.WithWait(resource.WithTimeout(-1)), nil,
	} {
		if _, err := service.Servers.Create(context.Background(), request, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("expected invalid option, got %v", err)
		}
	}
	if requests.Load() != 0 {
		t.Fatal("invalid options made requests")
	}
	if _, err := service.Servers.Create(context.Background(), request, compute.WithNetworks(resource.ID("network-id"))); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatalf("requests=%d", requests.Load())
	}
}

func TestCreateWaitFailureRetainsCreatedResource(t *testing.T) {
	cloud := testcloud.New(t)
	service, _ := connection(t, cloud).Compute(context.Background())
	cloud.Mux.HandleFunc("POST /compute/v2.1/project/servers", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
	})
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/created", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"server":{"id":"created","status":"BUILD"}}`)
	})
	v, err := service.Servers.Create(context.Background(), compute.CreateServerRequest{Name: "web", Image: resource.ID("image-id"), Flavor: resource.ID("flavor-id")}, compute.WithWait(resource.WithTimeout(10*time.Millisecond)))
	if v == nil || v.ID != "created" || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("created resource lost: v=%v err=%v", v, err)
	}
}
