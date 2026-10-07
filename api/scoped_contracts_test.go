package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/compute/v2/attachinterfaces"
	"github.com/JSYoo5B/gophercloudsdk/compute/v2/volumeattach"
	"github.com/JSYoo5B/gophercloudsdk/containerinfra/v1/nodegroups"
	"github.com/JSYoo5B/gophercloudsdk/dns/v2/recordsets"
	"github.com/JSYoo5B/gophercloudsdk/identity/v3/applicationcredentials"
	"github.com/JSYoo5B/gophercloudsdk/identity/v3/ec2credentials"
	"github.com/JSYoo5B/gophercloudsdk/image/v2/members"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/loadbalancer/v2/l7policies"
	"github.com/JSYoo5B/gophercloudsdk/loadbalancer/v2/pools"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/layer3/portforwarding"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func checkScope[T any](t *testing.T, cloud *testcloud.Cloud, bind func(context.Context, resource.Ref) (*resource.Collection[T], error), parentBase, parentEnvelope, childBase, listEnvelope, getEnvelope string, parentModel, childModel map[string]any, parentName, childName, status bool) {
	t.Helper()
	var parentCalls, childCalls, waitPolls atomic.Int32
	var waitMode atomic.Bool
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected scoped URL: %s", r.URL.Path)
		testcloud.JSON(w, 404, "{}")
	})
	cloud.Mux.HandleFunc(parentBase, func(w http.ResponseWriter, r *http.Request) {
		parentCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{parentEnvelope: []any{parentModel}})
	})
	cloud.Mux.HandleFunc(childBase, func(w http.ResponseWriter, r *http.Request) {
		childCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{listEnvelope: []any{childModel}})
	})
	cloud.Mux.HandleFunc(childBase+"/child", func(w http.ResponseWriter, r *http.Request) {
		childCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete || waitMode.Load() && waitPolls.Add(1) > 1 {
			testcloud.JSON(w, 404, "{}")
			return
		}
		var body any = childModel
		if getEnvelope != "" {
			body = map[string]any{getEnvelope: childModel}
		}
		_ = json.NewEncoder(w).Encode(body)
	})
	ctx := context.Background()
	collection, err := bind(ctx, resource.ID("parent"))
	if err != nil || collection == nil || parentCalls.Load() != 0 || childCalls.Load() != 0 {
		t.Fatalf("scope=%v parent=%d child=%d err=%v", collection, parentCalls.Load(), childCalls.Load(), err)
	}
	if _, err := bind(ctx, resource.ID("..")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := bind(canceled, resource.ID("parent")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if parentName {
		collection, err = bind(ctx, resource.Name("parent-name"))
		if err != nil || parentCalls.Load() != 1 {
			t.Fatalf("parent=%d err=%v", parentCalls.Load(), err)
		}
	} else if _, err := bind(ctx, resource.Name("parent-name")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if values, err := collection.All(ctx); err != nil || len(values) != 1 {
		t.Fatalf("values=%v err=%v", values, err)
	}
	if value, err := collection.Get(ctx, "child"); err != nil || value == nil {
		t.Fatalf("value=%v err=%v", value, err)
	}
	if childName {
		if id, err := collection.ResolveID(ctx, resource.Name("worker")); err != nil || id != "child" {
			t.Fatalf("child ID=%q err=%v", id, err)
		}
	} else if _, err := collection.Find(ctx, resource.Name("worker")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if status {
		if _, err := collection.Wait(ctx, resource.ID("child"), "ACTIVE"); err != nil {
			t.Fatal(err)
		}
		if values, err := collection.All(ctx, resource.WithStatus("active")); err != nil || len(values) != 1 {
			t.Fatalf("values=%v err=%v", values, err)
		}
	} else if _, err := collection.Wait(ctx, resource.ID("child"), "ACTIVE"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if err := collection.Delete(ctx, resource.ID("child")); err != nil {
		t.Fatal(err)
	}
	if err := collection.Delete(ctx, resource.ID("child"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	waitMode.Store(true)
	if err := collection.WaitDeleted(ctx, resource.ID("child"), resource.WithPollInterval(time.Millisecond)); err != nil || waitPolls.Load() != 2 {
		t.Fatalf("polls=%d err=%v", waitPolls.Load(), err)
	}
	if parentName && parentCalls.Load() != 1 {
		t.Fatalf("parent was resolved more than once: %d", parentCalls.Load())
	}
}

func TestScopedBindingsKeepParentAndNativeIdentifiers(t *testing.T) {
	parent := map[string]any{"id": "parent", "uuid": "parent", "name": "parent-name"}
	child := map[string]any{"id": "child", "name": "worker", "status": "ACTIVE"}
	t.Run("DNS record sets", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := recordsets.New(cloud.Client("dns", "/dns/v2"))
		bind := func(ctx context.Context, ref resource.Ref) (*resource.Collection[recordsets.RecordSet], error) {
			s, err := api.InZone(ctx, ref)
			if err != nil {
				return nil, err
			}
			return s.Collection, nil
		}
		checkScope(t, cloud, bind, "/dns/v2/zones", "zones", "/dns/v2/zones/parent/recordsets", "recordsets", "", parent, child, true, true, true)
	})
	t.Run("Nova interfaces", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := attachinterfaces.New(cloud.Client("compute", "/compute/v2.1/project"))
		bind := func(ctx context.Context, ref resource.Ref) (*resource.Collection[attachinterfaces.Interface], error) {
			s, err := api.InServer(ctx, ref)
			if err != nil {
				return nil, err
			}
			return s.Collection, nil
		}
		checkScope(t, cloud, bind, "/compute/v2.1/project/servers/detail", "servers", "/compute/v2.1/project/servers/parent/os-interface", "interfaceAttachments", "interfaceAttachment", parent, map[string]any{"port_id": "child", "port_state": "ACTIVE"}, true, false, true)
	})
	t.Run("Nova volume attachment uses volume ID", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := volumeattach.New(cloud.Client("compute", "/compute/v2.1/project"))
		bind := func(ctx context.Context, ref resource.Ref) (*resource.Collection[volumeattach.VolumeAttachment], error) {
			s, err := api.InServer(ctx, ref)
			if err != nil {
				return nil, err
			}
			return s.Collection, nil
		}
		checkScope(t, cloud, bind, "/compute/v2.1/project/servers/detail", "servers", "/compute/v2.1/project/servers/parent/os-volume_attachments", "volumeAttachments", "volumeAttachment", parent, map[string]any{"id": "attachment-id", "volumeId": "child"}, true, false, false)
	})
	t.Run("Magnum node group uses UUID", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := nodegroups.New(cloud.Client("container-infrastructure-management", "/magnum/v1"))
		bind := func(ctx context.Context, ref resource.Ref) (*resource.Collection[nodegroups.NodeGroup], error) {
			s, err := api.InCluster(ctx, ref)
			if err != nil {
				return nil, err
			}
			return s.Collection, nil
		}
		checkScope(t, cloud, bind, "/magnum/v1/clusters/detail", "clusters", "/magnum/v1/clusters/parent/nodegroups", "nodegroups", "", parent, map[string]any{"id": 42, "uuid": "child", "name": "worker", "status": "ACTIVE"}, true, true, true)
	})
	t.Run("Keystone application credentials", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := applicationcredentials.New(cloud.Client("identity", "/identity/v3"))
		bind := func(ctx context.Context, ref resource.Ref) (*resource.Collection[applicationcredentials.ApplicationCredential], error) {
			s, err := api.InUser(ctx, ref)
			if err != nil {
				return nil, err
			}
			return s.Collection, nil
		}
		checkScope(t, cloud, bind, "/identity/v3/users", "users", "/identity/v3/users/parent/application_credentials", "application_credentials", "application_credential", parent, child, true, true, false)
	})
	t.Run("Keystone access rules", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := applicationcredentials.New(cloud.Client("identity", "/identity/v3"))
		bind := func(ctx context.Context, ref resource.Ref) (*resource.Collection[applicationcredentials.AccessRule], error) {
			s, err := api.AccessRules(ctx, ref)
			if err != nil {
				return nil, err
			}
			return s.Collection, nil
		}
		checkScope(t, cloud, bind, "/identity/v3/users", "users", "/identity/v3/users/parent/access_rules", "access_rules", "access_rule", parent, child, true, false, false)
	})
	t.Run("Keystone EC2 credentials use access ID", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := ec2credentials.New(cloud.Client("identity", "/identity/v3"))
		bind := func(ctx context.Context, ref resource.Ref) (*resource.Collection[ec2credentials.Credential], error) {
			s, err := api.InUser(ctx, ref)
			if err != nil {
				return nil, err
			}
			return s.Collection, nil
		}
		checkScope(t, cloud, bind, "/identity/v3/users", "users", "/identity/v3/users/parent/credentials/OS-EC2", "credentials", "credential", parent, map[string]any{"access": "child"}, true, false, false)
	})
	t.Run("Glance members use member ID", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := members.New(cloud.Client("image", "/image/v2"))
		bind := func(ctx context.Context, ref resource.Ref) (*resource.Collection[members.Member], error) {
			s, err := api.InImage(ctx, ref)
			if err != nil {
				return nil, err
			}
			return s.Collection, nil
		}
		checkScope(t, cloud, bind, "/image/v2/images", "images", "/image/v2/images/parent/members", "members", "", parent, map[string]any{"member_id": "child", "status": "ACTIVE"}, true, false, true)
	})
	t.Run("Octavia pool members", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := pools.New(cloud.Client("load-balancer", "/octavia/v2.0"))
		bind := func(ctx context.Context, ref resource.Ref) (*resource.Collection[pools.Member], error) {
			s, err := api.Members(ctx, ref)
			if err != nil {
				return nil, err
			}
			return s.Collection, nil
		}
		checkScope(t, cloud, bind, "/octavia/v2.0/lbaas/pools", "pools", "/octavia/v2.0/lbaas/pools/parent/members", "members", "member", parent, map[string]any{"id": "child", "name": "worker", "provisioning_status": "ACTIVE"}, true, true, true)
	})
	t.Run("Octavia L7 rules", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := l7policies.New(cloud.Client("load-balancer", "/octavia/v2.0"))
		bind := func(ctx context.Context, ref resource.Ref) (*resource.Collection[l7policies.Rule], error) {
			s, err := api.Rules(ctx, ref)
			if err != nil {
				return nil, err
			}
			return s.Collection, nil
		}
		checkScope(t, cloud, bind, "/octavia/v2.0/lbaas/l7policies", "l7policies", "/octavia/v2.0/lbaas/l7policies/parent/rules", "rules", "rule", parent, map[string]any{"id": "child", "provisioning_status": "ACTIVE"}, true, false, true)
	})
	t.Run("Neutron port forwarding", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := portforwarding.New(cloud.Client("network", "/neutron/v2.0"))
		bind := func(ctx context.Context, ref resource.Ref) (*resource.Collection[portforwarding.PortForwarding], error) {
			s, err := api.InFloatingIP(ctx, ref)
			if err != nil {
				return nil, err
			}
			return s.Collection, nil
		}
		checkScope(t, cloud, bind, "/neutron/v2.0/floatingips", "floatingips", "/neutron/v2.0/floatingips/parent/port_forwardings", "port_forwardings", "port_forwarding", parent, child, false, false, false)
	})
}

func TestScopedRecordSetCreateAndUpdateKeepExtensionsAndDefaultTTL(t *testing.T) {
	cloud := testcloud.New(t)
	var parentCalls, writes atomic.Int32
	cloud.Mux.HandleFunc("/v2/zones", func(w http.ResponseWriter, r *http.Request) {
		parentCalls.Add(1)
		testcloud.JSON(w, 200, "{\"zones\":[{\"id\":\"zone-id\",\"name\":\"example.org.\"}]}")
	})
	cloud.Mux.HandleFunc("/v2/zones/zone-id/recordsets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			testcloud.JSON(w, 200, "{\"recordsets\":[{\"id\":\"record-id\",\"name\":\"www.example.org.\"}]}")
			return
		}
		writes.Add(1)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if r.Method != http.MethodPost || body["name"] != "www.example.org." || body["vendor:enabled"] != false {
			t.Errorf("method=%s body=%v", r.Method, body)
		}
		testcloud.JSON(w, 201, "{\"id\":\"record-id\",\"name\":\"www.example.org.\"}")
	})
	cloud.Mux.HandleFunc("/v2/zones/zone-id/recordsets/record-id", func(w http.ResponseWriter, r *http.Request) {
		writes.Add(1)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		ttl, exists := body["ttl"]
		if r.Method != http.MethodPut || !exists || ttl != nil || body["vendor:enabled"] != false {
			t.Errorf("method=%s body=%v", r.Method, body)
		}
		testcloud.JSON(w, 200, "{\"id\":\"record-id\",\"name\":\"www.example.org.\"}")
	})
	ctx := context.Background()
	scope, err := recordsets.New(cloud.Client("dns", "/v2")).InZone(ctx, resource.Name("example.org."))
	if err != nil {
		t.Fatal(err)
	}
	value, err := scope.Create(ctx, recordsets.CreateOpts{Name: "www.example.org.", Type: "A", Records: []string{"192.0.2.1"}}, recordsets.WithCreateField("vendor:enabled", false))
	if err != nil || value.ID != "record-id" {
		t.Fatalf("value=%v err=%v", value, err)
	}
	ttl := 0
	value, err = scope.Update(ctx, resource.Name("www.example.org."), recordsets.UpdateOpts{TTL: &ttl}, recordsets.WithUpdateField("vendor:enabled", false))
	if err != nil || value.ID != "record-id" {
		t.Fatalf("value=%v err=%v", value, err)
	}
	if _, err := scope.Update(ctx, resource.ID(".."), recordsets.UpdateOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if parentCalls.Load() != 1 || writes.Load() != 2 {
		t.Fatalf("parent=%d writes=%d", parentCalls.Load(), writes.Load())
	}
}

func TestScopedNodeGroupPatchUsesUUIDAndRetainsZero(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/v1/clusters/cluster-id/nodegroups", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, "{\"nodegroups\":[{\"id\":42,\"uuid\":\"node-uuid\",\"name\":\"workers\"}]}")
	})
	cloud.Mux.HandleFunc("/v1/clusters/cluster-id/nodegroups/node-uuid", func(w http.ResponseWriter, r *http.Request) {
		var body []map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if r.Method != http.MethodPatch || len(body) != 1 || body[0]["op"] != "replace" || body[0]["path"] != "/min_node_count" || body[0]["value"] != float64(0) {
			t.Errorf("method=%s patch=%v", r.Method, body)
		}
		testcloud.JSON(w, 202, "{\"id\":42,\"uuid\":\"node-uuid\"}")
	})
	ctx := context.Background()
	scope, err := nodegroups.New(cloud.Client("container-infrastructure-management", "/v1")).InCluster(ctx, resource.ID("cluster-id"))
	if err != nil {
		t.Fatal(err)
	}
	value, err := scope.Update(ctx, resource.Name("workers"), []nodegroups.UpdateOpts{{Op: nodegroups.ReplaceOp, Path: "/min_node_count", Value: 0}})
	if err != nil || value.UUID != "node-uuid" {
		t.Fatalf("value=%v err=%v", value, err)
	}
}
