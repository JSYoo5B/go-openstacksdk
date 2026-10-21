package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/chassis"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/portgroups"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/volumeconnectors"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/volumetargets"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// r2IronicAKind erases the four concrete APIs so one contract test covers every
// route. Every closure calls the real public API; no behavior is mocked.
type r2IronicAKind struct {
	name       string // test label
	path       string // route below /v1
	plural     string // list envelope key
	detailPath bool   // detail list is <path>/detail instead of ?detail=True
	filter     string // typed list filter query key accepted by every list call
	// item renders one wire object whose uuid is id and whose optional name is name.
	item func(id, name string) string
	// create runs Create with explicit false/zero/empty values and one extension field.
	create     func(context.Context, *gophercloud.ServiceClient) (string, error)
	createBody string // exact expected JSON body of create
	// createCollision tries each reserved extension name through WithCreateField.
	createCollision func(context.Context, *gophercloud.ServiceClient, string) error
	reserved        []string
	get             func(context.Context, *gophercloud.ServiceClient, string, ...string) (string, error)
	update          func(context.Context, *gophercloud.ServiceClient, string, ...r2IronicAOp) (string, error)
	del             func(context.Context, *gophercloud.ServiceClient, string) error
	remove          func(context.Context, *gophercloud.ServiceClient, string, ...resource.LookupOption) error
	// list returns uuids. limit and the typed filter are always set; extra adds WithListQuery.
	list     func(ctx context.Context, c *gophercloud.ServiceClient, detail bool, limit int, filter string, fields []string, extra [][2]string) ([]string, error)
	find     func(context.Context, *gophercloud.ServiceClient, string, ...resource.IdentityFindOption) (string, error)
	rawKeep  func(context.Context, *gophercloud.ServiceClient, string) (bool, error) // Body/Header/StatusCode retained
	protect  []string                                                                // server-owned patch targets
	hasNames bool
}

// r2IronicAOp is the test-side description of one patch operation.
type r2IronicAOp struct {
	Op, Path string
	Value    any
	NoValue  bool
}

func r2ironicAItem(fields string) func(id, name string) string {
	return func(id, name string) string {
		out := `{"uuid":"` + id + `"`
		if name != "" {
			out += `,"name":"` + name + `"`
		}
		return out + fields + `,"created_at":"2026-01-02T03:04:05+00:00","updated_at":null,"links":[{"href":"http://x/` + id + `","rel":"self"}],"vendor_extra":12345678901234567890}`
	}
}

func r2ironicAStr(v string) *string { return &v }
func r2ironicABool(v bool) *bool    { return &v }
func r2ironicAInt(v int) *int       { return &v }

func r2ironicAPatchOps[T any](ops []r2IronicAOp, mk func(op, path string, value any) T) []T {
	out := make([]T, 0, len(ops))
	for _, o := range ops {
		if o.NoValue {
			out = append(out, mk("remove", o.Path, nil))
			continue
		}
		out = append(out, mk(o.Op, o.Path, o.Value))
	}
	return out
}

func r2ironicAKinds() []r2IronicAKind {
	chassisAPI := func(c *gophercloud.ServiceClient) *chassis.API { return chassis.New(c) }
	pgAPI := func(c *gophercloud.ServiceClient) *portgroups.API { return portgroups.New(c) }
	vcAPI := func(c *gophercloud.ServiceClient) *volumeconnectors.API { return volumeconnectors.New(c) }
	vtAPI := func(c *gophercloud.ServiceClient) *volumetargets.API { return volumetargets.New(c) }
	return []r2IronicAKind{
		{
			name: "chassis", path: "chassis", plural: "chassis", detailPath: true, filter: "",
			item:       r2ironicAItem(`,"description":null,"extra":{"a":false},"nodes":[{"href":"http://x/n","rel":"self"}]`),
			createBody: `{"description":"","extra":{},"uuid":"c-1","vendor":{"enabled":false}}`,
			reserved:   []string{"uuid", "description", "extra", "created_at", "updated_at", "links", "nodes"},
			protect:    []string{"uuid", "created_at", "updated_at", "links", "nodes"},
			create: func(ctx context.Context, c *gophercloud.ServiceClient) (string, error) {
				v, err := chassisAPI(c).Create(ctx, chassis.CreateOpts{UUID: r2ironicAStr("c-1"), Description: r2ironicAStr(""), Extra: map[string]any{}},
					chassis.WithCreateField("vendor", map[string]bool{"enabled": false}))
				return r2ironicAUUID(v, err, func() string { return v.UUID })
			},
			createCollision: func(ctx context.Context, c *gophercloud.ServiceClient, key string) error {
				_, err := chassisAPI(c).Create(ctx, chassis.CreateOpts{}, chassis.WithCreateField(key, 1))
				return err
			},
			get: func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (string, error) {
				var opts []chassis.GetOption
				if fields != nil {
					opts = append(opts, chassis.WithGetFields(fields...))
				}
				v, err := chassisAPI(c).Get(ctx, id, opts...)
				return r2ironicAUUID(v, err, func() string { return v.UUID })
			},
			update: func(ctx context.Context, c *gophercloud.ServiceClient, id string, ops ...r2IronicAOp) (string, error) {
				patch := r2ironicAPatchOps(ops, func(op, path string, value any) chassis.PatchOperation {
					return chassis.PatchOperation{Op: op, Path: path, Value: value}
				})
				v, err := chassisAPI(c).Update(ctx, id, chassis.UpdateOpts{}, chassis.WithUpdateOperations(patch...))
				return r2ironicAUUID(v, err, func() string { return v.UUID })
			},
			del: func(ctx context.Context, c *gophercloud.ServiceClient, id string) error {
				return chassisAPI(c).Delete(ctx, id)
			},
			remove: func(ctx context.Context, c *gophercloud.ServiceClient, id string, o ...resource.LookupOption) error {
				return chassisAPI(c).Remove(ctx, resource.ID(id), o...)
			},
			list: func(ctx context.Context, c *gophercloud.ServiceClient, detail bool, limit int, filter string, fields []string, extra [][2]string) ([]string, error) {
				opts := []chassis.ListOption{chassis.WithListOptions(chassis.ListOpts{Limit: limit, Fields: fields, SortKey: "uuid", SortDir: "desc"})}
				for _, e := range extra {
					opts = append(opts, chassis.WithListQuery(e[0], e[1]))
				}
				var out []string
				seq := chassisAPI(c).List(ctx, opts...)
				if detail {
					seq = chassisAPI(c).ListDetail(ctx, opts...)
				}
				for v, err := range seq {
					if err != nil {
						return out, err
					}
					out = append(out, v.UUID)
				}
				return out, nil
			},
			find: func(ctx context.Context, c *gophercloud.ServiceClient, id string, o ...resource.IdentityFindOption) (string, error) {
				v, err := chassisAPI(c).FindIdentity(ctx, id, o...)
				return r2ironicAUUID(v, err, func() string { return v.UUID })
			},
			rawKeep: func(ctx context.Context, c *gophercloud.ServiceClient, id string) (bool, error) {
				v, err := chassisAPI(c).Get(ctx, id)
				if err != nil {
					return false, err
				}
				return string(v.Body["vendor_extra"]) == "12345678901234567890" && v.StatusCode == 200 && v.Header.Get("Content-Type") != "" && v.Description == nil && v.UpdatedAt == nil && len(v.Nodes) == 1 && v.Extra["a"] == false, nil
			},
		},
		{
			name: "portgroups", path: "portgroups", plural: "portgroups", detailPath: true, filter: "node", hasNames: true,
			item:       r2ironicAItem(`,"address":null,"node_uuid":"n-1","standalone_ports_supported":false,"mode":null,"properties":{},"extra":{},"internal_info":{},"ports":[]`),
			createBody: `{"address":"","extra":{},"mode":"","name":"","node_uuid":"n-1","properties":{},"standalone_ports_supported":false,"uuid":"pg-1","vendor":0}`,
			reserved:   []string{"uuid", "node_uuid", "name", "address", "standalone_ports_supported", "mode", "properties", "extra", "created_at", "updated_at", "links", "ports", "internal_info"},
			protect:    []string{"uuid", "created_at", "updated_at", "links", "ports", "internal_info"},
			create: func(ctx context.Context, c *gophercloud.ServiceClient) (string, error) {
				v, err := pgAPI(c).Create(ctx, portgroups.CreateOpts{UUID: r2ironicAStr("pg-1"), NodeUUID: "n-1", Name: r2ironicAStr(""), Address: r2ironicAStr(""),
					StandalonePortsSupported: r2ironicABool(false), Mode: r2ironicAStr(""), Properties: map[string]any{}, Extra: map[string]any{}},
					portgroups.WithCreateField("vendor", 0))
				return r2ironicAUUID(v, err, func() string { return v.UUID })
			},
			createCollision: func(ctx context.Context, c *gophercloud.ServiceClient, key string) error {
				_, err := pgAPI(c).Create(ctx, portgroups.CreateOpts{}, portgroups.WithCreateField(key, 1))
				return err
			},
			get: func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (string, error) {
				var opts []portgroups.GetOption
				if fields != nil {
					opts = append(opts, portgroups.WithGetFields(fields...))
				}
				v, err := pgAPI(c).Get(ctx, id, opts...)
				return r2ironicAUUID(v, err, func() string { return v.UUID })
			},
			update: func(ctx context.Context, c *gophercloud.ServiceClient, id string, ops ...r2IronicAOp) (string, error) {
				patch := r2ironicAPatchOps(ops, func(op, path string, value any) portgroups.PatchOperation {
					return portgroups.PatchOperation{Op: op, Path: path, Value: value}
				})
				v, err := pgAPI(c).Update(ctx, id, portgroups.UpdateOpts{}, portgroups.WithUpdateOperations(patch...))
				return r2ironicAUUID(v, err, func() string { return v.UUID })
			},
			del: func(ctx context.Context, c *gophercloud.ServiceClient, id string) error {
				return pgAPI(c).Delete(ctx, id)
			},
			remove: func(ctx context.Context, c *gophercloud.ServiceClient, id string, o ...resource.LookupOption) error {
				return pgAPI(c).Remove(ctx, resource.ID(id), o...)
			},
			list: func(ctx context.Context, c *gophercloud.ServiceClient, detail bool, limit int, filter string, fields []string, extra [][2]string) ([]string, error) {
				opts := []portgroups.ListOption{portgroups.WithListOptions(portgroups.ListOpts{Limit: limit, Fields: fields, Node: filter, Address: "aa:bb"})}
				for _, e := range extra {
					opts = append(opts, portgroups.WithListQuery(e[0], e[1]))
				}
				var out []string
				seq := pgAPI(c).List(ctx, opts...)
				if detail {
					seq = pgAPI(c).ListDetail(ctx, opts...)
				}
				for v, err := range seq {
					if err != nil {
						return out, err
					}
					out = append(out, v.UUID)
				}
				return out, nil
			},
			find: func(ctx context.Context, c *gophercloud.ServiceClient, id string, o ...resource.IdentityFindOption) (string, error) {
				v, err := pgAPI(c).FindIdentity(ctx, id, o...)
				return r2ironicAUUID(v, err, func() string { return v.UUID })
			},
			rawKeep: func(ctx context.Context, c *gophercloud.ServiceClient, id string) (bool, error) {
				v, err := pgAPI(c).Get(ctx, id)
				if err != nil {
					return false, err
				}
				return string(v.Body["vendor_extra"]) == "12345678901234567890" && v.StatusCode == 200 && v.Address == nil && v.Mode == nil && v.StandalonePortsSupported != nil && !*v.StandalonePortsSupported && v.NodeUUID == "n-1", nil
			},
		},
		{
			name: "volumeconnectors", path: "volume/connectors", plural: "connectors", filter: "node",
			item:       r2ironicAItem(`,"node_uuid":"n-1","type":"iqn","connector_id":"iqn.x","extra":{}`),
			createBody: `{"connector_id":"iqn.x","extra":{},"node_uuid":"n-1","type":"iqn","uuid":"vc-1","vendor":false}`,
			reserved:   []string{"uuid", "node_uuid", "type", "connector_id", "extra", "created_at", "updated_at", "links"},
			protect:    []string{"uuid", "created_at", "updated_at", "links"},
			create: func(ctx context.Context, c *gophercloud.ServiceClient) (string, error) {
				v, err := vcAPI(c).Create(ctx, volumeconnectors.CreateOpts{UUID: r2ironicAStr("vc-1"), NodeUUID: "n-1", Type: "iqn", ConnectorID: "iqn.x", Extra: map[string]any{}},
					volumeconnectors.WithCreateField("vendor", false))
				return r2ironicAUUID(v, err, func() string { return v.UUID })
			},
			createCollision: func(ctx context.Context, c *gophercloud.ServiceClient, key string) error {
				_, err := vcAPI(c).Create(ctx, volumeconnectors.CreateOpts{}, volumeconnectors.WithCreateField(key, 1))
				return err
			},
			get: func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (string, error) {
				var opts []volumeconnectors.GetOption
				if fields != nil {
					opts = append(opts, volumeconnectors.WithGetFields(fields...))
				}
				v, err := vcAPI(c).Get(ctx, id, opts...)
				return r2ironicAUUID(v, err, func() string { return v.UUID })
			},
			update: func(ctx context.Context, c *gophercloud.ServiceClient, id string, ops ...r2IronicAOp) (string, error) {
				patch := r2ironicAPatchOps(ops, func(op, path string, value any) volumeconnectors.PatchOperation {
					return volumeconnectors.PatchOperation{Op: op, Path: path, Value: value}
				})
				v, err := vcAPI(c).Update(ctx, id, volumeconnectors.UpdateOpts{}, volumeconnectors.WithUpdateOperations(patch...))
				return r2ironicAUUID(v, err, func() string { return v.UUID })
			},
			del: func(ctx context.Context, c *gophercloud.ServiceClient, id string) error {
				return vcAPI(c).Delete(ctx, id)
			},
			remove: func(ctx context.Context, c *gophercloud.ServiceClient, id string, o ...resource.LookupOption) error {
				return vcAPI(c).Remove(ctx, resource.ID(id), o...)
			},
			list: func(ctx context.Context, c *gophercloud.ServiceClient, detail bool, limit int, filter string, fields []string, extra [][2]string) ([]string, error) {
				opts := []volumeconnectors.ListOption{volumeconnectors.WithListOptions(volumeconnectors.ListOpts{Limit: limit, Fields: fields, Node: filter})}
				for _, e := range extra {
					opts = append(opts, volumeconnectors.WithListQuery(e[0], e[1]))
				}
				var out []string
				seq := vcAPI(c).List(ctx, opts...)
				if detail {
					seq = vcAPI(c).ListDetail(ctx, opts...)
				}
				for v, err := range seq {
					if err != nil {
						return out, err
					}
					out = append(out, v.UUID)
				}
				return out, nil
			},
			find: func(ctx context.Context, c *gophercloud.ServiceClient, id string, o ...resource.IdentityFindOption) (string, error) {
				v, err := vcAPI(c).FindIdentity(ctx, id, o...)
				return r2ironicAUUID(v, err, func() string { return v.UUID })
			},
			rawKeep: func(ctx context.Context, c *gophercloud.ServiceClient, id string) (bool, error) {
				v, err := vcAPI(c).Get(ctx, id)
				if err != nil {
					return false, err
				}
				return string(v.Body["vendor_extra"]) == "12345678901234567890" && v.StatusCode == 200 && v.ConnectorID == "iqn.x" && v.Type == "iqn", nil
			},
		},
		{
			name: "volumetargets", path: "volume/targets", plural: "targets", filter: "node",
			item:       r2ironicAItem(`,"node_uuid":"n-1","volume_type":"iscsi","volume_id":"vol-1","boot_index":0,"properties":{},"extra":{}`),
			createBody: `{"boot_index":0,"extra":{},"node_uuid":"n-1","properties":{},"uuid":"vt-1","vendor":null,"volume_id":"vol-1","volume_type":"iscsi"}`,
			reserved:   []string{"uuid", "node_uuid", "volume_type", "volume_id", "boot_index", "properties", "extra", "created_at", "updated_at", "links"},
			protect:    []string{"uuid", "created_at", "updated_at", "links"},
			create: func(ctx context.Context, c *gophercloud.ServiceClient) (string, error) {
				v, err := vtAPI(c).Create(ctx, volumetargets.CreateOpts{UUID: r2ironicAStr("vt-1"), NodeUUID: "n-1", VolumeType: "iscsi", VolumeID: "vol-1", BootIndex: r2ironicAInt(0),
					Properties: map[string]any{}, Extra: map[string]any{}}, volumetargets.WithCreateField("vendor", nil))
				return r2ironicAUUID(v, err, func() string { return v.UUID })
			},
			createCollision: func(ctx context.Context, c *gophercloud.ServiceClient, key string) error {
				_, err := vtAPI(c).Create(ctx, volumetargets.CreateOpts{}, volumetargets.WithCreateField(key, 1))
				return err
			},
			get: func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (string, error) {
				var opts []volumetargets.GetOption
				if fields != nil {
					opts = append(opts, volumetargets.WithGetFields(fields...))
				}
				v, err := vtAPI(c).Get(ctx, id, opts...)
				return r2ironicAUUID(v, err, func() string { return v.UUID })
			},
			update: func(ctx context.Context, c *gophercloud.ServiceClient, id string, ops ...r2IronicAOp) (string, error) {
				patch := r2ironicAPatchOps(ops, func(op, path string, value any) volumetargets.PatchOperation {
					return volumetargets.PatchOperation{Op: op, Path: path, Value: value}
				})
				v, err := vtAPI(c).Update(ctx, id, volumetargets.UpdateOpts{}, volumetargets.WithUpdateOperations(patch...))
				return r2ironicAUUID(v, err, func() string { return v.UUID })
			},
			del: func(ctx context.Context, c *gophercloud.ServiceClient, id string) error {
				return vtAPI(c).Delete(ctx, id)
			},
			remove: func(ctx context.Context, c *gophercloud.ServiceClient, id string, o ...resource.LookupOption) error {
				return vtAPI(c).Remove(ctx, resource.ID(id), o...)
			},
			list: func(ctx context.Context, c *gophercloud.ServiceClient, detail bool, limit int, filter string, fields []string, extra [][2]string) ([]string, error) {
				opts := []volumetargets.ListOption{volumetargets.WithListOptions(volumetargets.ListOpts{Limit: limit, Fields: fields, Node: filter})}
				for _, e := range extra {
					opts = append(opts, volumetargets.WithListQuery(e[0], e[1]))
				}
				var out []string
				seq := vtAPI(c).List(ctx, opts...)
				if detail {
					seq = vtAPI(c).ListDetail(ctx, opts...)
				}
				for v, err := range seq {
					if err != nil {
						return out, err
					}
					out = append(out, v.UUID)
				}
				return out, nil
			},
			find: func(ctx context.Context, c *gophercloud.ServiceClient, id string, o ...resource.IdentityFindOption) (string, error) {
				v, err := vtAPI(c).FindIdentity(ctx, id, o...)
				return r2ironicAUUID(v, err, func() string { return v.UUID })
			},
			rawKeep: func(ctx context.Context, c *gophercloud.ServiceClient, id string) (bool, error) {
				v, err := vtAPI(c).Get(ctx, id)
				if err != nil {
					return false, err
				}
				return string(v.Body["vendor_extra"]) == "12345678901234567890" && v.StatusCode == 200 && v.BootIndex == 0 && v.VolumeID == "vol-1", nil
			},
		},
	}
}

func r2ironicAUUID[T any](v *T, err error, uuid func() string) (string, error) {
	if err != nil || v == nil {
		return "", err
	}
	return uuid(), nil
}

// r2ironicARecorder captures requests for one route family.
type r2ironicARecorder struct {
	mu       sync.Mutex
	requests []r2ironicAReq
}

type r2ironicAReq struct {
	Method, Path string
	Query        url.Values
	Header       http.Header
	Body         string
}

func (r *r2ironicARecorder) add(req *http.Request) r2ironicAReq {
	b, _ := io.ReadAll(req.Body)
	got := r2ironicAReq{Method: req.Method, Path: req.URL.Path, Query: req.URL.Query(), Header: req.Header.Clone(), Body: string(b)}
	r.mu.Lock()
	r.requests = append(r.requests, got)
	r.mu.Unlock()
	return got
}

func (r *r2ironicARecorder) all() []r2ironicAReq {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]r2ironicAReq(nil), r.requests...)
}

// r2ironicAServe routes every request below /v1/<k.path> through handle.
func r2ironicAServe(t *testing.T, cloud *testcloud.Cloud, k r2IronicAKind, handle func(w http.ResponseWriter, req r2ironicAReq)) *r2ironicARecorder {
	t.Helper()
	rec := &r2ironicARecorder{}
	cloud.Mux.HandleFunc("/v1/"+k.path, func(w http.ResponseWriter, req *http.Request) { handle(w, rec.add(req)) })
	cloud.Mux.HandleFunc("/v1/"+k.path+"/", func(w http.ResponseWriter, req *http.Request) { handle(w, rec.add(req)) })
	return rec
}

func r2ironicAJSONEqual(t *testing.T, got, want string) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal([]byte(got), &g); err != nil {
		t.Fatalf("got is not JSON: %q: %v", got, err)
	}
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("want is not JSON: %v", err)
	}
	gb, _ := json.Marshal(g)
	wb, _ := json.Marshal(w)
	if string(gb) != string(wb) {
		t.Fatalf("body mismatch\n got %s\nwant %s", gb, wb)
	}
}

func r2ironicAIs404(err error) bool { return err != nil && gophercloud.ResponseCodeIs(err, 404) }

func r2ironicAHas(list []string, v string) bool {
	for _, s := range list {
		if strings.EqualFold(s, v) {
			return true
		}
	}
	return false
}
