package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path"
	"testing"

	"gophercloudsdk/compute/v2/aggregates"
	"gophercloudsdk/identity/v3/users"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/keymanager/v1/secrets"
	"gophercloudsdk/loadbalancer/v2/loadbalancers"
	"gophercloudsdk/resource"
)

func checkResourceBinding[T any](t *testing.T, cloud *testcloud.Cloud, base, id, listEnvelope, getEnvelope string, model map[string]any, collection *resource.Collection[T], idOf func(*T) string, hasStatus bool) {
	t.Helper()
	duplicate, deleted := false, false
	cloud.Mux.HandleFunc(base, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method=%s", r.Method)
		}
		other := map[string]any{}
		for key, value := range model {
			other[key] = value
		}
		other["name"] = "worker-suffix"
		if duplicate {
			other["name"] = "worker"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{listEnvelope: []any{model, other}})
	})
	cloud.Mux.HandleFunc(base+"/"+id, func(w http.ResponseWriter, r *http.Request) {
		if deleted || r.Method == http.MethodDelete {
			deleted = true
			testcloud.JSON(w, 404, `{"error":"missing"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		var body any = model
		if getEnvelope != "" {
			body = map[string]any{getEnvelope: model}
		}
		_ = json.NewEncoder(w).Encode(body)
	})
	ctx := context.Background()
	value, err := collection.Find(ctx, resource.Name("worker"))
	if err != nil || idOf(value) != id {
		t.Fatalf("value=%v err=%v", value, err)
	}
	duplicate = true
	if _, err := collection.Find(ctx, resource.Name("worker")); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatal(err)
	}
	duplicate = false
	value, err = collection.Find(ctx, resource.ID(id))
	if err != nil || idOf(value) != id {
		t.Fatalf("value=%v err=%v", value, err)
	}
	if hasStatus {
		if _, err := collection.Wait(ctx, resource.ID(id), "ACTIVE"); err != nil {
			t.Fatal(err)
		}
		values, err := collection.All(ctx, resource.WithName("worker"), resource.WithStatus("ACTIVE"))
		if err != nil || len(values) != 1 {
			t.Fatalf("values=%v err=%v", values, err)
		}
	} else if _, err := collection.Wait(ctx, resource.ID(id), "ACTIVE"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if err := collection.Delete(ctx, resource.Name("worker")); err != nil {
		t.Fatal(err)
	}
	if err := collection.WaitDeleted(ctx, resource.ID(id)); err != nil {
		t.Fatal(err)
	}
	if err := collection.Delete(ctx, resource.ID(id), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
}

func TestGeneratedBindingsHandleNamesNumericIDsRefsAndStateFields(t *testing.T) {
	t.Run("identity", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := users.New(cloud.Client("identity", "/identity/v3"))
		checkResourceBinding(t, cloud, "/identity/v3/users", "fixed", "users", "user", map[string]any{"id": "fixed", "name": "worker"}, api.Resources, func(v *users.User) string { return v.ID }, false)
	})
	t.Run("aggregate numeric ID", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := aggregates.New(cloud.Client("compute", "/compute/v2.1/project"))
		checkResourceBinding(t, cloud, "/compute/v2.1/project/os-aggregates", "1", "aggregates", "aggregate", map[string]any{"id": 1, "name": "worker"}, api.Resources, func(v *aggregates.Aggregate) string {
			if v.ID != 1 {
				t.Fatal(v.ID)
			}
			return "1"
		}, false)
	})
	t.Run("secret URL reference", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := secrets.New(cloud.Client("key-manager", "/keymanager/v1"))
		checkResourceBinding(t, cloud, "/keymanager/v1/secrets", "fixed", "secrets", "", map[string]any{"secret_ref": cloud.Server.URL + "/keymanager/v1/secrets/fixed", "name": "worker", "status": "ACTIVE"}, api.Resources, func(v *secrets.Secret) string { return path.Base(v.SecretRef) }, true)
	})
	t.Run("load balancer provisioning status", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := loadbalancers.New(cloud.Client("load-balancer", "/loadbalancer/v2.0"))
		checkResourceBinding(t, cloud, "/loadbalancer/v2.0/lbaas/loadbalancers", "fixed", "loadbalancers", "loadbalancer", map[string]any{"id": "fixed", "name": "worker", "provisioning_status": "ACTIVE"}, api.Resources, func(v *loadbalancers.LoadBalancer) string { return v.ID }, true)
	})
}
