package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	placement "github.com/JSYoo5B/go-openstacksdk/placement/v1"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/traits"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// placementWaitFixture serves one GET route: `present` successful bodies,
// then a Placement 404. Any other method or path fails the test.
func placementWaitFixture(t *testing.T, path, body string, present int32) (*gophercloud.ServiceClient, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	var gets atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != path {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			testcloud.JSON(w, 500, `{}`)
			return
		}
		if gets.Add(1) <= present {
			if body == "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			testcloud.JSON(w, 200, body)
			return
		}
		testcloud.JSON(w, 404, `{"errors":[{"status":404,"title":"Not Found"}]}`)
	})
	return cloud.Client("placement", "/placement"), &gets
}

func TestPlacementProxyWaitsDeleteEveryFetchableResource(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fast := []resource.WaitOption{resource.WithPollInterval(time.Millisecond), resource.WithTimeout(5 * time.Second)}
	cases := []struct {
		name, path, body string
		wait             func(*gophercloud.ServiceClient) error
	}{
		{"resource provider", "/placement/resource_providers/rp-1", `{"uuid":"rp-1","name":"a","generation":1}`, func(c *gophercloud.ServiceClient) error {
			return placement.WaitForDelete(ctx, resourceproviders.New(c).Resources, resource.ID("rp-1"), fast...)
		}},
		{"resource class", "/placement/resource_classes/CUSTOM_A", `{"name":"CUSTOM_A"}`, func(c *gophercloud.ServiceClient) error {
			return placement.WaitForDelete(ctx, resourceclasses.New(c).Resources, resource.ID("CUSTOM_A"), fast...)
		}},
		{"inventory", "/placement/resource_providers/rp-1/inventories/VCPU", `{"resource_provider_generation":1,"total":4}`, func(c *gophercloud.ServiceClient) error {
			return placement.WaitForDelete(ctx, resourceproviders.New(c).InventoryResources("rp-1"), resource.ID("VCPU"), fast...)
		}},
		{"trait", "/placement/traits/CUSTOM_T", "", func(c *gophercloud.ServiceClient) error {
			return placement.WaitForDelete(ctx, traits.New(c).Resources(), resource.ID("CUSTOM_T"), fast...)
		}},
		{"provider traits", "/placement/resource_providers/rp-1/traits", `{"resource_provider_generation":1,"traits":["CUSTOM_T"]}`, func(c *gophercloud.ServiceClient) error {
			return placement.WaitForDelete(ctx, resourceproviders.New(c).TraitResources(), resource.ID("rp-1"), fast...)
		}},
		{"allocation", "/placement/allocations/consumer-1", `{"allocations":{"rp-1":{"resources":{"VCPU":1}}},"project_id":"p","user_id":"u"}`, func(c *gophercloud.ServiceClient) error {
			return placement.WaitForDelete(ctx, allocations.New(c).Resources(), resource.ID("consumer-1"), fast...)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client, gets := placementWaitFixture(t, tc.path, tc.body, 2)
			if err := tc.wait(client); err != nil || gets.Load() != 3 {
				t.Fatalf("err=%v gets=%d", err, gets.Load())
			}
		})
	}
}

func TestPlacementProxyWaitsStatusAttributesAndEmptyAllocations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fast := resource.WithPollInterval(time.Millisecond)

	// No Placement model declares status: the default fails before HTTP.
	client, gets := placementWaitFixture(t, "/placement/resource_providers/rp-1", `{"uuid":"rp-1","name":"ready","generation":1}`, 10)
	providers := resourceproviders.New(client).Resources
	if _, err := placement.WaitForStatus(ctx, providers, resource.ID("rp-1"), "ready", fast); !errors.Is(err, resource.ErrUnsupported) || gets.Load() != 0 {
		t.Fatalf("default attribute err=%v gets=%d", err, gets.Load())
	}
	provider, err := placement.WaitForStatus(ctx, providers, resource.ID("rp-1"), "READY", fast, resource.WithStatusAttribute("name"))
	if err != nil || provider.Name != "ready" || gets.Load() != 1 {
		t.Fatalf("provider=%v err=%v gets=%d", provider, err, gets.Load())
	}

	// Numeric inventory attributes are not string statuses, as Python's
	// lower() call on an integer also fails.
	invClient, invGets := placementWaitFixture(t, "/placement/resource_providers/rp-1/inventories/VCPU", `{"resource_provider_generation":1,"total":4}`, 10)
	if _, err := placement.WaitForStatus(ctx, resourceproviders.New(invClient).InventoryResources("rp-1"), resource.ID("VCPU"), "4", fast, resource.WithStatusAttribute("total")); !errors.Is(err, resource.ErrUnsupported) || invGets.Load() != 0 {
		t.Fatalf("numeric attribute err=%v gets=%d", err, invGets.Load())
	}

	// Allocation exposes the Python project_id attribute name. Its consumer is
	// the polled reference; ERROR in the selected attribute fails.
	allocClient, allocGets := placementWaitFixture(t, "/placement/allocations/consumer-1", `{"allocations":{},"project_id":"Project-A","user_id":"u"}`, 1<<20)
	allocs := allocations.New(allocClient).Resources()
	value, err := placement.WaitForStatus(ctx, allocs, resource.ID("consumer-1"), "project-a", fast, resource.WithStatusAttribute("project_id"))
	if err != nil || value.ConsumerUUID != "consumer-1" || value.ProjectID == nil || *value.ProjectID != "Project-A" {
		t.Fatalf("allocation=%v err=%v", value, err)
	}
	// An empty allocation set is a successful observation, not absence: the
	// deletion wait keeps polling until its deadline, without DELETE.
	before := allocGets.Load()
	err = placement.WaitForDelete(ctx, allocs, resource.ID("consumer-1"), fast, resource.WithTimeout(30*time.Millisecond))
	if !errors.Is(err, context.DeadlineExceeded) || allocGets.Load()-before < 2 {
		t.Fatalf("empty allocation delete err=%v gets=%d", err, allocGets.Load()-before)
	}

	// A provider-traits 403 is not absence.
	cloud := testcloud.New(t)
	var denied atomic.Int32
	cloud.Mux.HandleFunc("/placement/resource_providers/rp-2/traits", func(w http.ResponseWriter, r *http.Request) {
		denied.Add(1)
		testcloud.JSON(w, 403, `{"errors":[{"status":403}]}`)
	})
	if err := placement.WaitForDelete(ctx, resourceproviders.New(cloud.Client("placement", "/placement")).TraitResources(), resource.ID("rp-2"), fast); err == nil || denied.Load() != 1 {
		t.Fatalf("403 err=%v gets=%d", err, denied.Load())
	}
}
