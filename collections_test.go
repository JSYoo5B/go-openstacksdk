package gophercloudsdk_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

func TestSharedPolicyAcrossServices(t *testing.T) {
	cloud := testcloud.New(t)
	c := connection(t, cloud)
	ctx := context.Background()
	compute, _ := c.Compute(ctx)
	network, _ := c.Network(ctx)
	image, _ := c.Image(ctx)
	storage, _ := c.BlockStorage(ctx)
	checks := []struct {
		name, path, body string
		lookup           func(context.Context, resource.Ref) (string, error)
	}{
		{"server", "/compute/v2.1/project/servers/detail", `{"servers":[{"id":"s1","name":"same"}]}`, func(ctx context.Context, r resource.Ref) (string, error) {
			v, e := compute.Servers.Find(ctx, r)
			if e != nil {
				return "", e
			}
			return v.ID, nil
		}},
		{"flavor", "/compute/v2.1/project/flavors/detail", `{"flavors":[{"id":"f1","name":"same"}]}`, func(ctx context.Context, r resource.Ref) (string, error) {
			v, e := compute.Flavors.Find(ctx, r)
			if e != nil {
				return "", e
			}
			return v.ID, nil
		}},
		{"network", "/network/v2.0/networks", `{"networks":[{"id":"n1","name":"same"}]}`, func(ctx context.Context, r resource.Ref) (string, error) {
			v, e := network.Networks.Find(ctx, r)
			if e != nil {
				return "", e
			}
			return v.ID, nil
		}},
		{"image", "/image/v2/images", `{"images":[{"id":"i1","name":"same"}]}`, func(ctx context.Context, r resource.Ref) (string, error) {
			v, e := image.Images.Find(ctx, r)
			if e != nil {
				return "", e
			}
			return v.ID, nil
		}},
		{"volume", "/volume/v3/project/volumes/detail", `{"volumes":[{"id":"v1","name":"same"}]}`, func(ctx context.Context, r resource.Ref) (string, error) {
			v, e := storage.Volumes.Find(ctx, r)
			if e != nil {
				return "", e
			}
			return v.ID, nil
		}},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			cloud.Mux.HandleFunc("GET "+check.path, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, check.body) })
			if id, err := check.lookup(ctx, resource.Name("same")); err != nil || id == "" {
				t.Fatalf("id=%q err=%v", id, err)
			}
			if _, err := check.lookup(ctx, resource.Name("absent")); !errors.Is(err, resource.ErrNotFound) {
				t.Fatalf("missing: %v", err)
			}
		})
	}
	if err := compute.Flavors.Delete(ctx, resource.ID("id")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := compute.Flavors.Wait(ctx, resource.ID("id"), "ACTIVE"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
}

func TestPaginationExactNamesAndEarlyStop(t *testing.T) {
	cloud := testcloud.New(t)
	service, _ := connection(t, cloud).Compute(context.Background())
	var nextCalls atomic.Int32
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/detail", func(w http.ResponseWriter, r *http.Request) {
		if name := r.URL.Query().Get("name"); name != "" && name != "^"+regexp.QuoteMeta("web[1].*")+"$" {
			t.Errorf("name regexp=%q", name)
		}
		if r.URL.Query().Get("limit") != "2" {
			t.Error("page size missing")
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"servers":[{"id":"s1","name":"web[1].*"},{"id":"other","name":"web-anything"}],"servers_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/compute/v2.1/project/servers/page2"))
	})
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/page2", func(w http.ResponseWriter, r *http.Request) {
		nextCalls.Add(1)
		testcloud.JSON(w, 200, `{"servers":[{"id":"s2","name":"web[1].*"}]}`)
	})
	ctx := context.Background()
	for _, err := range service.Servers.List(ctx, resource.WithPageSize(2)) {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if nextCalls.Load() != 0 {
		t.Fatal("fetched next page after break")
	}
	all, err := service.Servers.All(ctx, resource.WithPageSize(2), resource.WithName("web[1].*"))
	if err != nil || len(all) != 2 || all[0].ID != "s1" || all[1].ID != "s2" {
		t.Fatalf("all=%v err=%v", all, err)
	}
}

func TestDuplicateNamesAcrossPages(t *testing.T) {
	cloud := testcloud.New(t)
	service, _ := connection(t, cloud).Compute(context.Background())
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/detail", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, fmt.Sprintf(`{"servers":[{"id":"s1","name":"duplicate"}],"servers_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/compute/v2.1/project/servers/page2"))
	})
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/page2", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"servers":[{"id":"s2","name":"duplicate"}]}`)
	})
	for _, ignore := range []bool{false, true} {
		var opts []resource.LookupOption
		if ignore {
			opts = append(opts, resource.WithIgnoreMissing())
		}
		_, err := service.Servers.Find(context.Background(), resource.Name("duplicate"), opts...)
		var ambiguous *resource.AmbiguousError
		if !errors.Is(err, resource.ErrAmbiguous) || !errors.As(err, &ambiguous) || len(ambiguous.IDs) != 2 {
			t.Fatalf("ambiguity lost: %v", err)
		}
	}
	if err := service.Servers.Delete(context.Background(), resource.Name("duplicate")); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatal(err)
	}
}

func TestMissingAndHTTPFailures(t *testing.T) {
	cloud := testcloud.New(t)
	service, _ := connection(t, cloud).Compute(context.Background())
	cloud.Mux.HandleFunc("/compute/v2.1/project/servers/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{"message":"missing"}`) })
	cloud.Mux.HandleFunc("/compute/v2.1/project/servers/forbidden", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 403, `{"message":"forbidden"}`) })
	ctx := context.Background()
	_, err := service.Servers.Get(ctx, "missing")
	if !errors.Is(err, resource.ErrNotFound) || !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatalf("underlying HTTP error lost: %v", err)
	}
	if v, err := service.Servers.Find(ctx, resource.ID("missing"), resource.WithIgnoreMissing()); v != nil || err != nil {
		t.Fatalf("v=%v err=%v", v, err)
	}
	if err := service.Servers.Delete(ctx, resource.ID("missing")); err != nil {
		t.Fatal(err)
	}
	if err := service.Servers.Delete(ctx, resource.ID("missing"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := service.Servers.Find(ctx, resource.ID("forbidden"), resource.WithIgnoreMissing()); !gophercloud.ResponseCodeIs(err, 403) {
		t.Fatal(err)
	}
	if err := service.Servers.Delete(ctx, resource.ID("forbidden")); !gophercloud.ResponseCodeIs(err, 403) {
		t.Fatal(err)
	}
}

func TestWaitPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	service, _ := connection(t, cloud).Compute(context.Background())
	var polls atomic.Int32
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/ready", func(w http.ResponseWriter, r *http.Request) {
		status := "BUILD"
		if polls.Add(1) > 1 {
			status = "ACTIVE"
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"server":{"id":"ready","status":%q}}`, status))
	})
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/failed", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"server":{"id":"failed","status":"ERROR"}}`)
	})
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/build", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"server":{"id":"build","status":"BUILD"}}`)
	})
	ctx := context.Background()
	if v, err := service.Servers.Wait(ctx, resource.ID("ready"), "active", resource.WithPollInterval(time.Millisecond)); err != nil || v.Status != "ACTIVE" {
		t.Fatalf("v=%v err=%v", v, err)
	}
	if _, err := service.Servers.Wait(ctx, resource.ID("failed"), "ACTIVE"); !errors.Is(err, resource.ErrFailedState) {
		t.Fatal(err)
	}
	if _, err := service.Servers.Wait(ctx, resource.ID("build"), "ACTIVE", resource.WithTimeout(10*time.Millisecond)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := service.Servers.Wait(canceled, resource.ID("build"), "ACTIVE"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := service.Servers.Wait(ctx, resource.ID("build"), "ACTIVE", resource.WithPollInterval(0)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
