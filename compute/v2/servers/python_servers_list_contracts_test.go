package servers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// These fixtures compare servers.API.ListCloudRecords with the pinned
// openstack/compute/v2/_proxy.py Proxy.servers listing, which delegates to
// openstack/proxy.py Proxy._list and openstack/resource.py Resource.list over
// the openstack/compute/v2/server.py Server resource.

func serverListCollect(t *testing.T, api *servers.API, options ...servers.RecordFindOption) ([]*servers.ServerFindRecord, error) {
	t.Helper()
	var rows []*servers.ServerFindRecord
	for row, err := range api.ListCloudRecords(context.Background(), options...) {
		if err != nil {
			return rows, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func serverListIDs(rows []*servers.ServerFindRecord) string {
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = strings.Trim(string(row.Resource.Body["id"]), `"`)
	}
	return strings.Join(ids, ",")
}

func TestPythonServersListDetailsAndAllProjectsAlternatives(t *testing.T) {
	var calls []nativeServerCall
	var headers []http.Header
	api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
		headers = append(headers, req.Header.Clone())
		return nativeServerWire(200, `{"servers":[{"id":"s1","name":"one"}]}`)
	})
	for _, item := range []struct {
		name    string
		options []servers.RecordFindOption
		path    string
		query   string
	}{
		{"default details", nil, serverFindDetail, ""},
		{"details true", []servers.RecordFindOption{servers.WithRecordFindDetails(true)}, serverFindDetail, ""},
		{"details false", []servers.RecordFindOption{servers.WithRecordFindDetails(false)}, serverFindBase, ""},
		{"all projects", []servers.RecordFindOption{servers.WithRecordFindAllProjects(true)}, serverFindDetail, "all_tenants=True"},
		{"all projects false", []servers.RecordFindOption{servers.WithRecordFindAllProjects(false)}, serverFindDetail, ""},
		{"summary all projects", []servers.RecordFindOption{servers.WithRecordFindDetails(false), servers.WithRecordFindAllProjects(true)}, serverFindBase, "all_tenants=True"},
		{"raw all_tenants", []servers.RecordFindOption{servers.WithRecordFindQuery("all_tenants", 1)}, serverFindDetail, "all_tenants=1"},
	} {
		calls, headers = nil, nil
		rows, err := serverListCollect(t, api, item.options...)
		if err != nil || len(rows) != 1 || len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != item.path || calls[0].query != item.query {
			t.Fatal(item.name, rows, err, calls)
		}
		if rows[0].Microversion != "2.96" || rows[0].StatusCode != 200 || headers[0].Get("Accept") != "application/json" || headers[0].Get("X-OpenStack-Nova-API-Version") != "2.96" {
			t.Fatal(item.name, rows[0], headers[0])
		}
		serverFindValue(t, rows[0], "name", `"one"`)
	}
	calls = nil
	for _, conflict := range [][]servers.RecordFindOption{
		{servers.WithRecordFindAllProjects(true), servers.WithRecordFindQuery("all_tenants", true)},
		{servers.WithRecordFindAllProjects(false), servers.WithRecordFindQuery("all_tenants", true)},
	} {
		if _, err := serverListCollect(t, api, conflict...); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}

// serverListPythonQueryMapping is the pinned Server._query_mapping (client
// name to Nova name), including the implicit limit/marker and TagMixin pairs.
var serverListPythonQueryMapping = map[string]string{
	"auto_disk_config": "auto_disk_config", "availability_zone": "availability_zone", "created_at": "created_at",
	"description": "description", "flavor": "flavor", "hostname": "hostname", "image": "image", "kernel_id": "kernel_id",
	"key_name": "key_name", "launch_index": "launch_index", "launched_at": "launched_at", "locked": "locked",
	"locked_by": "locked_by", "name": "name", "node": "node", "power_state": "power_state", "progress": "progress",
	"project_id": "project_id", "ramdisk_id": "ramdisk_id", "reservation_id": "reservation_id",
	"root_device_name": "root_device_name", "status": "status", "task_state": "task_state",
	"terminated_at": "terminated_at", "user_id": "user_id", "vm_state": "vm_state", "sort_key": "sort_key",
	"sort_dir": "sort_dir", "pinned_availability_zone": "pinned_availability_zone",
	"access_ipv4": "access_ip_v4", "access_ipv6": "access_ip_v6", "has_config_drive": "config_drive",
	"deleted_only": "deleted", "compute_host": "host", "is_soft_deleted": "soft_deleted", "ipv4_address": "ip",
	"ipv6_address": "ip6", "changes_since": "changes-since", "changes_before": "changes-before", "id": "uuid",
	"tags": "tags", "any_tags": "tags-any", "not_tags": "not-tags", "not_any_tags": "not-tags-any",
	"limit": "limit", "marker": "marker",
}

func TestPythonServersListEveryTypedFilterTransposes(t *testing.T) {
	var calls []nativeServerCall
	api, _ := serverFindAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeServerWire(200, `{"servers":[]}`)
	})
	for client, wire := range serverListPythonQueryMapping {
		for _, key := range []string{client, wire} {
			calls = nil
			rows, err := serverListCollect(t, api, servers.WithRecordFindQuery(key, "7"))
			if err != nil || len(rows) != 0 || len(calls) != 1 {
				t.Fatal(key, rows, err, calls)
			}
			q, _ := url.ParseQuery(calls[0].query)
			if !reflect.DeepEqual(q, url.Values{wire: {"7"}}) {
				t.Fatal(key, q)
			}
		}
		if client == wire {
			continue
		}
		// The client-side name wins over the Nova name, and an explicit
		// client-side null suppresses the Nova name (requests drops None).
		calls = nil
		if _, err := serverListCollect(t, api, servers.WithRecordFindQuery(wire, "server"), servers.WithRecordFindQuery(client, "client")); err != nil {
			t.Fatal(err)
		}
		calls2 := calls
		calls = nil
		if _, err := serverListCollect(t, api, servers.WithRecordFindQuery(wire, "server"), servers.WithRecordFindQuery(client, nil)); err != nil {
			t.Fatal(err)
		}
		if want := (url.Values{wire: {"client"}}).Encode(); calls2[0].query != want || calls[0].query != "" {
			t.Fatal(client, calls2, calls)
		}
	}
	calls = nil
	if _, err := serverListCollect(t, api,
		servers.WithRecordFindQuery("locked", true),
		servers.WithRecordFindQuery("deleted_only", false),
		servers.WithRecordFindQuery("any_tags", []any{"a", nil, "b"}),
		servers.WithRecordFindQuery("progress", 0),
		servers.WithRecordFindQuery("x_vendor_extension", "dropped"),
		servers.WithRecordFindQuery("ipv4_address", nil)); err != nil {
		t.Fatal(err)
	}
	q, _ := url.ParseQuery(calls[0].query)
	if !reflect.DeepEqual(q, url.Values{"locked": {"True"}, "deleted": {"False"}, "tags-any": {"a", "b"}, "progress": {"0"}}) {
		t.Fatal(q)
	}
}

func TestPythonServersListClientBodyFiltersAndRawCap(t *testing.T) {
	var calls []nativeServerCall
	page := `{"servers":[` +
		`{"id":"a","host_status":"UP","locked":true,"metadata":{"role":"web","tier":"1"}},` +
		`{"id":"b","host_status":"DOWN","locked":false,"metadata":{"role":"web"}},` +
		`{"id":"c","host_status":"UP","locked":1,"metadata":{"role":"db"}},` +
		`{"id":"d","host_status":"UP","locked":true,"metadata":{"role":"web"}}]}`
	api, _ := serverFindAPI(t, &calls, func(*http.Request) *http.Response { return nativeServerWire(200, page) })
	for _, item := range []struct {
		name    string
		options []servers.RecordFindOption
		want    string
	}{
		{"scalar", []servers.RecordFindOption{servers.WithRecordFindQuery("host_status", "UP")}, "a,c,d"},
		{"converted bool equals int", []servers.RecordFindOption{servers.WithRecordFindQuery("is_locked", true)}, "a,c,d"},
		{"nested dictionary subset", []servers.RecordFindOption{servers.WithRecordFindQuery("metadata", map[string]any{"role": "web"})}, "a,b,d"},
		{"combined", []servers.RecordFindOption{servers.WithRecordFindQuery("host_status", "UP"), servers.WithRecordFindQuery("metadata", map[string]any{"role": "web"})}, "a,d"},
		// Resource.list counts every received row toward max_items before
		// the client filter, so the third row ends the stream unmatched.
		{"raw cap before filter", []servers.RecordFindOption{servers.WithRecordFindQuery("host_status", "DOWN"), servers.WithRecordFindMaxItems(1)}, ""},
		{"raw cap counts skipped rows", []servers.RecordFindOption{servers.WithRecordFindQuery("host_status", "UP"), servers.WithRecordFindMaxItems(3)}, "a,c"},
	} {
		calls = nil
		rows, err := serverListCollect(t, api, item.options...)
		if err != nil || serverListIDs(rows) != item.want || len(calls) != 1 {
			t.Fatal(item.name, serverListIDs(rows), err, calls)
		}
		q, _ := url.ParseQuery(calls[0].query)
		if q.Has("host_status") || q.Has("is_locked") || q.Has("metadata") || q.Has("locked") {
			t.Fatal(item.name, q)
		}
	}
	calls = nil
	if _, err := serverListCollect(t, api, servers.WithRecordFindQuery("metadata", map[string]any{"role": "web"}), servers.WithRecordFindMaxItems(2)); err != nil || calls[0].query != "limit=2" {
		t.Fatal(err, calls)
	}
}

func TestPythonServersListNextLinkAndMarkerFallbackPaging(t *testing.T) {
	for _, lane := range []string{"plural", "link", "next", "marker", "maxitems", "boundary", "boundaryfail", "single", "empty", "cycle"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativeServerCall
			api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
				q := req.URL.Query()
				switch q.Get("marker") {
				case "":
				case "s2":
					if lane == "cycle" {
						return nativeServerWire(200, `{"servers":[{"id":"s3"}],"servers_links":[{"rel":"next","href":"?marker=s2"}]}`)
					}
					if lane == "boundaryfail" {
						return nativeServerWire(503, `{}`)
					}
					if lane == "marker" && q.Get("limit") != "2" {
						return nativeServerWire(400, `{}`)
					}
					return nativeServerWire(200, `{"servers":[{"id":"s3"}]}`)
				case "s3":
					return nativeServerWire(200, `{"servers":[]}`)
				default:
					return nativeServerWire(400, `{}`)
				}
				switch lane {
				case "plural", "single", "cycle", "boundary", "boundaryfail":
					return nativeServerWire(200, `{"servers":[{"id":"s1"},{"id":"s2"}],"servers_links":[{"rel":"next","href":"?marker=s2"}]}`)
				case "link":
					r := nativeServerWire(200, `{"servers":[{"id":"s1"},{"id":"s2"}]}`)
					r.Header.Set("Link", `<?marker=s2>; rel="next"`)
					return r
				case "next":
					return nativeServerWire(200, `{"servers":[{"id":"s1"},{"id":"s2"}],"next":"/v2.1/servers/detail?marker=s2"}`)
				case "empty":
					return nativeServerWire(200, `{"servers":[],"servers_links":[{"rel":"next","href":"?marker=s2"}]}`)
				}
				return nativeServerWire(200, `{"servers":[{"id":"s1"},{"id":"s2"}]}`)
			})
			var options []servers.RecordFindOption
			want, requests := "s1,s2,s3", 2
			switch lane {
			case "marker":
				// No continuation link: a caller limit keeps requesting with the
				// last received ID until an empty page.
				options, requests = []servers.RecordFindOption{servers.WithRecordFindLimit(2)}, 3
			case "maxitems":
				// max_items without a limit sends it as the wire limit hint; the
				// marker fallback still requests the empty page after the cap.
				options, want, requests = []servers.RecordFindOption{servers.WithRecordFindMaxItems(3)}, "s1,s2,s3", 3
			case "boundary":
				// The cap is checked before each further row, so a cap reached at a
				// page boundary still requests the next page and discards its row.
				options, want = []servers.RecordFindOption{servers.WithRecordFindMaxItems(2)}, "s1,s2"
			case "boundaryfail":
				options = []servers.RecordFindOption{servers.WithRecordFindMaxItems(2)}
			case "single":
				options, want, requests = []servers.RecordFindOption{servers.WithRecordFindPaginated(false)}, "s1,s2", 1
			case "empty":
				want, requests = "", 1
			case "plain":
			}
			rows, err := serverListCollect(t, api, options...)
			if lane == "boundaryfail" {
				if serverListIDs(rows) != "s1,s2" || !gophercloud.ResponseCodeIs(err, http.StatusServiceUnavailable) || len(calls) != 2 {
					t.Fatal(serverListIDs(rows), err, calls)
				}
				return
			}
			if lane == "cycle" {
				var receipt *resource.ResponseError
				if serverListIDs(rows) != "s1,s2,s3" || !errors.As(err, &receipt) || receipt.StatusCode != 200 {
					t.Fatal(serverListIDs(rows), err, calls)
				}
				return
			}
			if err != nil || serverListIDs(rows) != want || len(calls) != requests {
				t.Fatal(serverListIDs(rows), err, calls)
			}
			if lane == "maxitems" {
				first, _ := url.ParseQuery(calls[0].query)
				if first.Get("limit") != "3" {
					t.Fatal(calls)
				}
			}
			if lane == "marker" && (calls[0].query != "limit=2" || !strings.Contains(calls[2].query, "marker=s3")) {
				t.Fatal(calls)
			}
		})
	}
}

func TestPythonServersListPerCallMicroversionHeadersAndControls(t *testing.T) {
	var calls []nativeServerCall
	var headers []http.Header
	api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
		headers = append(headers, req.Header.Clone())
		return nativeServerWire(200, `{"servers":[{"id":"s1"}]}`)
	})
	rows, err := serverListCollect(t, api, servers.WithRecordFindMicroversion("2.79"), servers.WithRecordFindHeaders(map[string]string{"X-Trace": "owned"}))
	if err != nil || len(rows) != 1 || rows[0].Microversion != "2.79" || len(headers) != 1 {
		t.Fatal(rows, err, headers)
	}
	h := headers[0]
	if h.Get("X-OpenStack-Nova-API-Version") != "2.79" || h.Get("OpenStack-API-Version") != "compute 2.79" || h.Get("X-Trace") != "owned" || h.Get("Accept") != "application/json" || h.Get("X-Auth-Token") != "test-token" {
		t.Fatal(h)
	}
	if api.RawClient().Microversion != "2.96" {
		t.Fatal("per-call microversion leaked into the shared client")
	}
	headers = nil
	if _, err := serverListCollect(t, api, servers.WithRecordFindHeaders(map[string]string{"Accept": "application/vnd.custom"})); err != nil || headers[0].Get("Accept") != "application/vnd.custom" || headers[0].Get("X-Trace") != "" {
		t.Fatal(err, headers)
	}
	calls = nil
	for _, invalid := range [][]servers.RecordFindOption{
		{servers.WithRecordFindHeaders(map[string]string{"OpenStack-API-Version": "compute 2.1"})},
		{servers.WithRecordFindHeaders(map[string]string{"X-OpenStack-Nova-API-Version": "2.1"})},
		{servers.WithRecordFindQuery("base_path", "/other")},
		{servers.WithRecordFindQuery("paginated", false)},
		{servers.WithRecordFindQuery("max_items", 1)},
		{servers.WithRecordFindQuery("microversion", "2.1")},
		{servers.WithRecordFindQuery("headers", map[string]string{})},
		{servers.WithRecordFindQuery("details", false)},
		{servers.WithRecordFindQuery("all_projects", true)},
		{servers.WithRecordFindQuery("jmespath_filters", "[?status=='ACTIVE']")},
		{servers.WithRecordFindOpts(servers.RecordFindOpts{Query: map[string]json.RawMessage{"id": json.RawMessage("{")}})},
		{servers.WithRecordFindMaxItems(-1)},
		{servers.WithRecordFindLimit(-1)},
		{nil},
	} {
		if _, err := serverListCollect(t, api, invalid...); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, err := range api.ListCloudRecords(ctx) {
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}

func TestPythonServersListRowDescriptorsAndReceipts(t *testing.T) {
	var calls []nativeServerCall
	fixtureCloud := "fixture"
	body := `{"servers":[{"id":"s1","accessIPv4":"192.0.2.10","OS-EXT-STS:power_state":1,"locked":"yes","tenant_id":"p1","OS-EXT-SRV-ATTR:host":"compute-a","os-extended-volumes:volumes_attached":[{"id":"v1"}],"flavor":{"original_name":"small","vcpus":"2"},"unknown":9007199254740993}],"servers_links":[]}`
	api, _ := serverFindAPI(t, &calls, func(*http.Request) *http.Response { return nativeServerWire(200, body) })
	rows, err := serverListCollect(t, api, servers.WithRecordFindLocation(resource.CloudLocation{Cloud: &fixtureCloud, Project: resource.CloudProject{ID: json.RawMessage(`"p1"`)}}))
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	row := rows[0]
	for key, want := range map[string]string{"access_ipv4": `"192.0.2.10"`, "power_state": "1", "is_locked": "true", "project_id": `"p1"`, "compute_host": `"compute-a"`, "tags": "[]", "status": "null"} {
		serverFindValue(t, row, key, want)
	}
	flavor := serverFindObject(t, row, "flavor")
	if string(flavor["name"]) != `"small"` || string(flavor["vcpus"]) != "2" {
		t.Fatal(flavor)
	}
	if string(row.Resource.Body["attached_volumes"]) != string(row.Resource.Body["volumes"]) || row.Resource.Body["unknown"] != nil {
		t.Fatal(row.Resource.Body)
	}
	if string(row.Wire.Body["unknown"]) != "9007199254740993" || string(row.Wire.Body["locked"]) != `"yes"` || string(row.Envelope) != body || row.Header.Get("X-Server-Proof") != "actual" {
		t.Fatal(row.Wire, string(row.Envelope), row.Header)
	}
	location := serverFindObject(t, row, "location")
	if string(location["cloud"]) != `"fixture"` {
		t.Fatal(location)
	}
	row.Resource.Body["project_id"][1] = 'X'
	if string(row.Wire.Body["tenant_id"]) != `"p1"` {
		t.Fatal("resource view aliases the received row")
	}
}
