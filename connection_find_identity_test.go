package gophercloudsdk_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"regexp"
	"sync/atomic"
	"testing"

	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestConnectionFindIdentitySharesPolicyAcrossHighLevelResources(t *testing.T) {
	cloud := testcloud.New(t)
	conn := connection(t, cloud)
	ctx := context.Background()
	compute, err := conn.Compute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	storage, err := conn.BlockStorage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	network, err := conn.Network(ctx)
	if err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		kind, base, envelope, identity, hint string
		find                                 func(context.Context, string, ...resource.IdentityFindOption) (string, error)
	}{
		{"server", "/compute/v2.1/project/servers", "servers", "web[1]", "^" + regexp.QuoteMeta("web[1]") + "$",
			func(ctx context.Context, value string, options ...resource.IdentityFindOption) (string, error) {
				server, err := compute.Servers.FindIdentity(ctx, value, options...)
				if server == nil {
					return "", err
				}
				return server.ID, err
			}},
		{"volume", "/volume/v3/project/volumes", "volumes", "data[1]", "data[1]",
			func(ctx context.Context, value string, options ...resource.IdentityFindOption) (string, error) {
				volume, err := storage.Volumes.FindIdentity(ctx, value, options...)
				if volume == nil {
					return "", err
				}
				return volume.ID, err
			}},
		{"port", "/network/v2.0/ports", "ports", "web-port", "web-port",
			func(ctx context.Context, value string, options ...resource.IdentityFindOption) (string, error) {
				port, err := network.Ports.FindIdentity(ctx, value, options...)
				if port == nil {
					return "", err
				}
				return port.ID, err
			}},
		{"network", "/network/v2.0/networks", "networks", "web-net", "web-net",
			func(ctx context.Context, value string, options ...resource.IdentityFindOption) (string, error) {
				valueNetwork, err := network.Networks.FindIdentity(ctx, value, options...)
				if valueNetwork == nil {
					return "", err
				}
				return valueNetwork.ID, err
			}},
	}
	for _, check := range checks {
		t.Run(check.kind, func(t *testing.T) {
			var gets, lists atomic.Int32
			cloud.Mux.HandleFunc("GET "+check.base+"/"+check.identity, func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if !reflect.DeepEqual(r.URL.Query()["tag"], []string{"first", "second"}) || r.URL.Query().Has("name") {
					t.Errorf("caller query or list-only hint on direct GET: %v", r.URL.Query())
				}
				testcloud.JSON(w, http.StatusForbidden, `{"error":"ID lookup denied"}`)
			})
			listPath := check.base
			if check.kind == "server" || check.kind == "volume" {
				listPath += "/detail"
			}
			cloud.Mux.HandleFunc("GET "+listPath, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				query := r.URL.Query()
				if query.Get("name") != check.hint || !reflect.DeepEqual(query["tag"], []string{"first", "second"}) {
					t.Errorf("query=%v", query)
				}
				if query.Get("project_id") == "empty" {
					testcloud.JSON(w, http.StatusOK, `{"`+check.envelope+`":[]}`)
					return
				}
				testcloud.JSON(w, http.StatusOK, `{"`+check.envelope+`":[{"id":"stable-id","name":"`+check.identity+`"}]}`)
			})
			query := resource.IdentityFindOpts{Query: map[string][]string{"tag": {"first", "second"}}}
			frozen := resource.WithIdentityFindOptions(query)
			query.Query["tag"][0] = "caller changed"
			id, err := check.find(ctx, check.identity, frozen)
			if err != nil || id != "stable-id" {
				t.Fatalf("id=%q err=%v", id, err)
			}
			id, err = check.find(ctx, check.identity, frozen, resource.WithIdentityFindQuery("project_id", "empty"))
			if err != nil || id != "" {
				t.Fatalf("default missing: id=%q err=%v", id, err)
			}
			_, err = check.find(ctx, check.identity, frozen, resource.WithIdentityFindIgnoreMissing(false), resource.WithIdentityFindQuery("project_id", "empty"))
			if !errors.Is(err, resource.ErrNotFound) {
				t.Fatalf("strict missing: %v", err)
			}
			if gets.Load() != 3 || lists.Load() != 3 {
				t.Fatalf("gets=%d lists=%d", gets.Load(), lists.Load())
			}
		})
	}
	if _, err := compute.Servers.FindIdentity(ctx, "server", resource.WithIdentityFindExtraSpecs(false)); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("flavor-only extra specs capability on server: %v", err)
	}
}
