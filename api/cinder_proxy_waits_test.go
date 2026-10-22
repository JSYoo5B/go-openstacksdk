package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/capabilities"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/defaulttypes"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// cinderWaitFixture answers version discovery at the endpoint root and one
// member GET route: `present` successful bodies, then a Cinder 404. Any other
// method or path fails the test.
func cinderWaitFixture(t *testing.T, path, body string, present int32) (*gophercloud.ServiceClient, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	var gets atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/cinder/v3/" {
			testcloud.JSON(w, 200, `{"version":{"id":"v3.0","status":"CURRENT","version":"3.70","min_version":"3.0"}}`)
			return
		}
		if r.Method != http.MethodGet || r.URL.Path != path {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			testcloud.JSON(w, 500, `{}`)
			return
		}
		if gets.Add(1) <= present {
			testcloud.JSON(w, 200, body)
			return
		}
		testcloud.JSON(w, 404, `{"itemNotFound":{"code":404,"message":"missing"}}`)
	})
	client := cloud.Client("volumev3", "/cinder/v3/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	return client, &gets
}

// Every pinned block_storage/v3 Resource with allow_fetch and a member route
// has an SDK collection that the shared deletion waiter polls until 404.
func TestCinderProxyWaitsDeleteEveryFetchableResource(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fast := []resource.WaitOption{resource.WithPollInterval(time.Millisecond), resource.WithTimeout(5 * time.Second)}
	cases := []struct {
		name, path, body string
		wait             func(*gophercloud.ServiceClient) error
	}{
		{"volume", "/cinder/v3/project/volumes/v-1", `{"volume":{"id":"v-1","status":"available"}}`, func(c *gophercloud.ServiceClient) error {
			return blockstorage.WaitForDelete(ctx, volumes.New(c).Resources, resource.ID("v-1"), fast...)
		}},
		{"snapshot", "/cinder/v3/project/snapshots/s-1", `{"snapshot":{"id":"s-1","status":"available"}}`, func(c *gophercloud.ServiceClient) error {
			return blockstorage.WaitForDelete(ctx, snapshots.New(c).Resources, resource.ID("s-1"), fast...)
		}},
		{"backup", "/cinder/v3/project/backups/b-1", `{"backup":{"id":"b-1","status":"available"}}`, func(c *gophercloud.ServiceClient) error {
			return blockstorage.WaitForDelete(ctx, backups.New(c).Resources, resource.ID("b-1"), fast...)
		}},
		{"attachment", "/cinder/v3/project/attachments/a-1", `{"attachment":{"id":"a-1","status":"attached"}}`, func(c *gophercloud.ServiceClient) error {
			return blockstorage.WaitForDelete(ctx, attachments.New(c).Resources, resource.ID("a-1"), fast...)
		}},
		{"transfer", "/cinder/v3/project/os-volume-transfer/t-1", `{"transfer":{"id":"t-1","volume_id":"v-1"}}`, func(c *gophercloud.ServiceClient) error {
			return blockstorage.WaitForDelete(ctx, transfers.New(c).Resources, resource.ID("t-1"), fast...)
		}},
		{"QoS spec", "/cinder/v3/project/qos-specs/q-1", `{"qos_specs":{"id":"q-1","name":"a","specs":{}}}`, func(c *gophercloud.ServiceClient) error {
			return blockstorage.WaitForDelete(ctx, qos.New(c).Resources, resource.ID("q-1"), fast...)
		}},
		{"volume type", "/cinder/v3/project/types/vt-1", `{"volume_type":{"id":"vt-1","name":"a"}}`, func(c *gophercloud.ServiceClient) error {
			return blockstorage.WaitForDelete(ctx, volumetypes.New(c).Resources, resource.ID("vt-1"), fast...)
		}},
		{"group", "/cinder/v3/project/groups/g-1", `{"group":{"id":"g-1","status":"available"}}`, func(c *gophercloud.ServiceClient) error {
			return blockstorage.WaitForDelete(ctx, groups.New(c).Resources(), resource.ID("g-1"), fast...)
		}},
		{"group snapshot", "/cinder/v3/project/group_snapshots/gs-1", `{"group_snapshot":{"id":"gs-1","status":"available"}}`, func(c *gophercloud.ServiceClient) error {
			return blockstorage.WaitForDelete(ctx, groupsnapshots.New(c).Resources(), resource.ID("gs-1"), fast...)
		}},
		{"consistency group", "/cinder/v3/project/consistencygroups/cg-1", `{"consistencygroup":{"id":"cg-1","status":"available"}}`, func(c *gophercloud.ServiceClient) error {
			return blockstorage.WaitForDelete(ctx, consistencygroups.New(c).Resources(), resource.ID("cg-1"), fast...)
		}},
		{"consistency group snapshot", "/cinder/v3/project/cgsnapshots/cs-1", `{"cgsnapshot":{"id":"cs-1","status":"available"}}`, func(c *gophercloud.ServiceClient) error {
			return blockstorage.WaitForDelete(ctx, consistencygroupsnapshots.New(c).Resources(), resource.ID("cs-1"), fast...)
		}},
		{"cluster", "/cinder/v3/project/clusters/c1@lvm", `{"cluster":{"name":"c1@lvm","status":"enabled"}}`, func(c *gophercloud.ServiceClient) error {
			return blockstorage.WaitForDelete(ctx, clusters.New(c).Resources(), resource.ID("c1@lvm"), fast...)
		}},
		{"default type", "/cinder/v3/project/default-types/p-1", `{"default_type":{"project_id":"p-1","volume_type_id":"vt-1"}}`, func(c *gophercloud.ServiceClient) error {
			return blockstorage.WaitForDelete(ctx, defaulttypes.New(c).Resources(), resource.ID("p-1"), fast...)
		}},
		{"group type", "/cinder/v3/project/group_types/gt-1", `{"group_type":{"id":"gt-1","name":"a"}}`, func(c *gophercloud.ServiceClient) error {
			return blockstorage.WaitForDelete(ctx, grouptypes.New(c).Resources(), resource.ID("gt-1"), fast...)
		}},
		{"message", "/cinder/v3/project/messages/m-1", `{"message":{"id":"m-1","event_id":"VOLUME_000001"}}`, func(c *gophercloud.ServiceClient) error {
			return blockstorage.WaitForDelete(ctx, messages.New(c).Resources(), resource.ID("m-1"), fast...)
		}},
		{"capabilities", "/cinder/v3/project/capabilities/host@lvm", `{"namespace":"OS::Storage::Capabilities::host@lvm","volume_backend_name":"lvm"}`, func(c *gophercloud.ServiceClient) error {
			return blockstorage.WaitForDelete(ctx, capabilities.New(c).Resources(), resource.ID("host@lvm"), fast...)
		}},
		{"quota class set", "/cinder/v3/project/os-quota-class-sets/gold", `{"quota_class_set":{"id":"gold","volumes":10}}`, func(c *gophercloud.ServiceClient) error {
			return blockstorage.WaitForDelete(ctx, quotasets.New(c).QuotaClassResources(), resource.ID("gold"), fast...)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client, gets := cinderWaitFixture(t, tc.path, tc.body, 2)
			if err := tc.wait(client); err != nil || gets.Load() != 3 {
				t.Fatalf("err=%v gets=%d", err, gets.Load())
			}
		})
	}
}

func TestCinderProxyWaitsRecordAndClusterStatus(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fast := resource.WithPollInterval(time.Millisecond)

	// Record collections read the declared status attribute.
	cloud := testcloud.New(t)
	var groupGets atomic.Int32
	cloud.Mux.HandleFunc("/cinder/v3/", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"version":{"id":"v3.0","status":"CURRENT","version":"3.70","min_version":"3.0"}}`)
	})
	cloud.Mux.HandleFunc("/cinder/v3/project/groups/g-1", func(w http.ResponseWriter, r *http.Request) {
		status := "creating"
		if groupGets.Add(1) >= 2 {
			status = "Available"
		}
		testcloud.JSON(w, 200, `{"group":{"id":"g-1","status":"`+status+`"}}`)
	})
	cloud.Mux.HandleFunc("/cinder/v3/project/groups/g-2", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"group":{"id":"g-2","status":"error"}}`)
	})
	cloud.Mux.HandleFunc("/cinder/v3/project/groups/g-3", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"group":{"id":"g-3","status":"deleted"}}`)
	})
	cloud.Mux.HandleFunc("/cinder/v3/project/clusters/c1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"cluster":{"name":"c1","status":"enabled"}}`)
	})
	client := cloud.Client("volumev3", "/cinder/v3/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"

	collection := groups.New(client).Resources()
	group, err := blockstorage.WaitForAvailable(ctx, collection, resource.ID("g-1"), fast)
	if err != nil || groupGets.Load() != 2 || group.Resource == nil || string(group.Resource.Body["status"]) != `"Available"` {
		t.Fatalf("group=%v err=%v gets=%d", group, err, groupGets.Load())
	}
	if _, err := blockstorage.WaitForAvailable(ctx, collection, resource.ID("g-2"), fast); !errors.Is(err, resource.ErrFailedState) {
		t.Fatalf("error err=%v", err)
	}
	if err := blockstorage.WaitForDelete(ctx, collection, resource.ID("g-3"), fast); err != nil {
		t.Fatalf("deleted status err=%v", err)
	}
	cluster, err := blockstorage.WaitForState(ctx, clusters.New(client).Resources(), resource.ID("c1"), "ENABLED", fast)
	if err != nil || cluster.Status == nil || *cluster.Status != "enabled" {
		t.Fatalf("cluster=%v err=%v", cluster, err)
	}
	// A status-less model fails before HTTP without a selected attribute.
	if _, err := blockstorage.WaitForState(ctx, grouptypes.New(client).Resources(), resource.ID("gt-1"), "x", fast); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("status-less err=%v", err)
	}
}
