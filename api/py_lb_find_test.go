package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/amphorae"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/flavorprofiles"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/flavors"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/l7policies"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/listeners"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/loadbalancers"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/monitors"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/pools"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pyLBFindCase struct {
	label   string
	base    string // path below /v2
	listKey string
	itemKey string
	byName  bool
	find    func(ctx context.Context, cloud *testcloud.Cloud, ref resource.Ref, opts ...resource.LookupOption) (string, error)
}

func pyLBFindCases() []pyLBFindCase {
	return []pyLBFindCase{
		{label: "load balancer", base: "/lbaas/loadbalancers", listKey: "loadbalancers", itemKey: "loadbalancer", byName: true,
			find: func(ctx context.Context, c *testcloud.Cloud, ref resource.Ref, o ...resource.LookupOption) (string, error) {
				v, err := loadbalancers.New(c.Client("load-balancer", "/v2")).Find(ctx, ref, o...)
				return pyLBID(v, err, func() string { return v.ID })
			}},
		{label: "listener", base: "/lbaas/listeners", listKey: "listeners", itemKey: "listener", byName: true,
			find: func(ctx context.Context, c *testcloud.Cloud, ref resource.Ref, o ...resource.LookupOption) (string, error) {
				v, err := listeners.New(c.Client("load-balancer", "/v2")).Find(ctx, ref, o...)
				return pyLBID(v, err, func() string { return v.ID })
			}},
		{label: "pool", base: "/lbaas/pools", listKey: "pools", itemKey: "pool", byName: true,
			find: func(ctx context.Context, c *testcloud.Cloud, ref resource.Ref, o ...resource.LookupOption) (string, error) {
				v, err := pools.New(c.Client("load-balancer", "/v2")).Find(ctx, ref, o...)
				return pyLBID(v, err, func() string { return v.ID })
			}},
		{label: "health monitor", base: "/lbaas/healthmonitors", listKey: "healthmonitors", itemKey: "healthmonitor", byName: true,
			find: func(ctx context.Context, c *testcloud.Cloud, ref resource.Ref, o ...resource.LookupOption) (string, error) {
				v, err := monitors.New(c.Client("load-balancer", "/v2")).Find(ctx, ref, o...)
				return pyLBID(v, err, func() string { return v.ID })
			}},
		{label: "l7 policy", base: "/lbaas/l7policies", listKey: "l7policies", itemKey: "l7policy", byName: true,
			find: func(ctx context.Context, c *testcloud.Cloud, ref resource.Ref, o ...resource.LookupOption) (string, error) {
				v, err := l7policies.New(c.Client("load-balancer", "/v2")).Find(ctx, ref, o...)
				return pyLBID(v, err, func() string { return v.ID })
			}},
		{label: "flavor", base: "/lbaas/flavors", listKey: "flavors", itemKey: "flavor", byName: true,
			find: func(ctx context.Context, c *testcloud.Cloud, ref resource.Ref, o ...resource.LookupOption) (string, error) {
				v, err := flavors.New(c.Client("load-balancer", "/v2")).Find(ctx, ref, o...)
				return pyLBID(v, err, func() string { return v.ID })
			}},
		{label: "flavor profile", base: "/lbaas/flavorprofiles", listKey: "flavorprofiles", itemKey: "flavorprofile", byName: true,
			find: func(ctx context.Context, c *testcloud.Cloud, ref resource.Ref, o ...resource.LookupOption) (string, error) {
				v, err := flavorprofiles.New(c.Client("load-balancer", "/v2")).Find(ctx, ref, o...)
				return pyLBID(v, err, func() string { return v.ID })
			}},
		{label: "amphora", base: "/octavia/amphorae", listKey: "amphorae", itemKey: "amphora",
			find: func(ctx context.Context, c *testcloud.Cloud, ref resource.Ref, o ...resource.LookupOption) (string, error) {
				v, err := amphorae.New(c.Client("load-balancer", "/v2")).Find(ctx, ref, o...)
				return pyLBID(v, err, func() string { return v.ID })
			}},
		{label: "member", base: "/lbaas/pools/pool-1/members", listKey: "members", itemKey: "member", byName: true,
			find: func(ctx context.Context, c *testcloud.Cloud, ref resource.Ref, o ...resource.LookupOption) (string, error) {
				scope, err := pools.New(c.Client("load-balancer", "/v2")).Members(ctx, resource.ID("pool-1"))
				if err != nil {
					return "", err
				}
				v, err := scope.Find(ctx, ref, o...)
				return pyLBID(v, err, func() string { return v.ID })
			}},
		{label: "l7 rule", base: "/lbaas/l7policies/pol-1/rules", listKey: "rules", itemKey: "rule",
			find: func(ctx context.Context, c *testcloud.Cloud, ref resource.Ref, o ...resource.LookupOption) (string, error) {
				scope, err := l7policies.New(c.Client("load-balancer", "/v2")).Rules(ctx, resource.ID("pol-1"))
				if err != nil {
					return "", err
				}
				v, err := scope.Find(ctx, ref, o...)
				return pyLBID(v, err, func() string { return v.ID })
			}},
	}
}

// pyLBID returns the found ID, or an empty string for a nil result.
func pyLBID[T any](v *T, err error, id func() string) (string, error) {
	if v == nil {
		return "", err
	}
	return id(), err
}

func pyLBItem(key, id, name string) json.RawMessage {
	data, _ := json.Marshal(map[string]any{key: map[string]any{"id": id, "name": name}})
	return data
}

type pyLBRecorder struct {
	mu    sync.Mutex
	calls []string
	names []url.Values
}

func (r *pyLBRecorder) add(req *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, req.Method+" "+req.URL.Path)
	r.names = append(r.names, req.URL.Query())
}

// Python find_* tries the ID, then an exact name list, returns None for a
// missing object by default and raises on duplicates. Go splits the reference
// kind explicitly and defaults to strict; WithIgnoreMissing restores the
// Python default.
func TestPyLBFindExplicitIDAndNameKeepIgnoreMissingAndDuplicateSemantics(t *testing.T) {
	for _, tc := range pyLBFindCases() {
		t.Run(tc.label, func(t *testing.T) {
			cloud := testcloud.New(t)
			rec := &pyLBRecorder{}
			cloud.Mux.HandleFunc("GET /v2"+tc.base+"/{id}", func(w http.ResponseWriter, r *http.Request) {
				rec.add(r)
				if r.PathValue("id") == "id-1" {
					testcloud.JSON(w, 200, string(pyLBItem(tc.itemKey, "id-1", "n1")))
					return
				}
				testcloud.JSON(w, 404, `{}`)
			})
			cloud.Mux.HandleFunc("GET /v2"+tc.base, func(w http.ResponseWriter, r *http.Request) {
				rec.add(r)
				var rows []json.RawMessage
				add := func(id, name string) {
					var wrapper map[string]json.RawMessage
					_ = json.Unmarshal(pyLBItem(tc.itemKey, id, name), &wrapper)
					rows = append(rows, wrapper[tc.itemKey])
				}
				switch r.URL.Query().Get("name") {
				case "n1":
					add("id-1", "n1")
				case "dup":
					add("d1", "dup")
					add("d2", "dup")
				}
				data, _ := json.Marshal(map[string]any{tc.listKey: rows})
				testcloud.JSON(w, 200, string(data))
			})
			ctx := context.Background()

			id, err := tc.find(ctx, cloud, resource.ID("id-1"))
			if err != nil || id != "id-1" {
				t.Fatalf("id hit id=%q err=%v", id, err)
			}
			if id, err = tc.find(ctx, cloud, resource.ID("nope"), resource.WithIgnoreMissing()); err != nil || id != "" {
				t.Fatalf("ignored missing id=%q err=%v", id, err)
			}
			if _, err = tc.find(ctx, cloud, resource.ID("nope")); err == nil {
				t.Fatal("Go ID lookup is strict unless WithIgnoreMissing is supplied")
			}
			if !tc.byName {
				return
			}
			if id, err = tc.find(ctx, cloud, resource.Name("n1")); err != nil || id != "id-1" {
				t.Fatalf("name hit id=%q err=%v", id, err)
			}
			if id, err = tc.find(ctx, cloud, resource.Name("absent"), resource.WithIgnoreMissing()); err != nil || id != "" {
				t.Fatalf("ignored missing name id=%q err=%v", id, err)
			}
			if _, err = tc.find(ctx, cloud, resource.Name("absent")); !errors.Is(err, resource.ErrNotFound) {
				t.Fatalf("strict missing name: %v", err)
			}
			var ambiguous *resource.AmbiguousError
			if _, err = tc.find(ctx, cloud, resource.Name("dup")); !errors.As(err, &ambiguous) {
				t.Fatalf("duplicate name: %v", err)
			}
			rec.mu.Lock()
			defer rec.mu.Unlock()
			for i, call := range rec.calls {
				if call == "GET /v2"+tc.base && rec.names[i].Get("name") == "" {
					t.Fatalf("name lookup sent no name filter: %v", rec.names[i])
				}
			}
		})
	}
}
