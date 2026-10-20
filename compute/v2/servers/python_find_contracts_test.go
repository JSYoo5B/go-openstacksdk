package servers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const serverFindBase = "/nova/v2.1/servers"
const serverFindDetail = serverFindBase + "/detail"

func serverFindAPI(t *testing.T, calls *[]nativeServerCall, reply func(*http.Request) *http.Response) (*servers.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	client := nativeServerClient(cloud)
	client.Microversion = "2.96"
	cloud.Provider.HTTPClient.Transport = nativeServerTransport(func(req *http.Request) (*http.Response, error) {
		body := ""
		if req.Body != nil {
			raw, _ := io.ReadAll(req.Body)
			body = string(raw)
		}
		*calls = append(*calls, nativeServerCall{req.Method, req.URL.Path, req.URL.RawQuery, body})
		return reply(req), nil
	})
	return servers.New(client), cloud
}
func serverFindValue(t *testing.T, v *servers.ServerFindRecord, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		t.Fatalf("%s want %s got %+v", key, want, v)
	}
}
func serverFindObject(t *testing.T, v *servers.ServerFindRecord, key string) map[string]json.RawMessage {
	t.Helper()
	var out map[string]json.RawMessage
	if v == nil || v.Resource == nil || json.Unmarshal(v.Resource.Body[key], &out) != nil {
		t.Fatalf("%s %+v", key, v)
	}
	return out
}
func TestPythonServerFindRecordDirectDefaultsAliasesAndWire(t *testing.T) {
	var calls []nativeServerCall
	var header http.Header
	var escaped string
	body := `{"server":{"id":null,"name":false,"tenant_id":"foreign","OS-EXT-AZ:availability_zone":"az","progress":"١٢","OS-EXT-SRV-ATTR:launch_index":true,"locked":"false-string","created":"not-a-native-time","config_drive":false,"image":"","metadata":0,"tags":null,"unknown":9007199254740993,"if_match":"body","self":"reserved"}}`
	api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
		header = req.Header.Clone()
		escaped = req.URL.EscapedPath()
		r := nativeServerWire(203, body)
		r.Header.Set("If-Match", "actual")
		return r
	})
	v, err := api.FindRecord(context.Background(), "folder/name%", servers.WithRecordFindDetails(false), servers.WithRecordFindAllProjects(true), servers.WithRecordFindQuery("x_raw", map[string]any{"false": false, "zero": 0}), servers.WithRecordFindQuery("name", "seed"), servers.WithRecordFindHeaders(map[string]string{"X-Trace": "owned"}))
	if err != nil || len(calls) != 1 || calls[0].method != "GET" || calls[0].body != "" || escaped != serverFindBase+"/folder%2Fname%25" || header.Get("OpenStack-API-Version") != "compute 2.96" || header.Get("X-OpenStack-Nova-API-Version") != "2.96" || header.Get("X-Trace") != "owned" || header.Get("X-Auth-Token") != "test-token" {
		t.Fatal(v, err, calls, header, escaped)
	}
	q, _ := url.ParseQuery(calls[0].query)
	if q.Has("all_projects") || q.Has("all_tenants") || q.Get("name") != "seed" || !reflect.DeepEqual(q["x_raw"], []string{"false", "zero"}) {
		t.Fatal(q)
	}
	for key, want := range map[string]string{"id": "null", "name": "false", "project_id": `"foreign"`, "progress": "12", "launch_index": "true", "is_locked": "true", "created_at": `"not-a-native-time"`, "config_drive": "false", "has_config_drive": "false", "image": "{}", "metadata": "{}", "tags": "null", "attached_volumes": "[]", "volumes": "[]", "interface_ip": `""`} {
		serverFindValue(t, v, key, want)
	}
	if v.Resource.Body["unknown"] != nil || v.Resource.Body["x_raw"] != nil || v.Resource.Body["self"] != nil || v.Resource.Body["if_match"] != nil || v.Wire == nil || string(v.Wire.Body["progress"]) != `"١٢"` || string(v.Wire.Body["unknown"]) != "9007199254740993" || string(v.Envelope) != body || v.StatusCode != 203 || v.Microversion != "2.96" {
		t.Fatal(v)
	}
	location := serverFindObject(t, v, "location")
	if string(location["zone"]) != `"az"` {
		t.Fatal(location)
	}
	v.Resource.Body["progress"][0] = 'X'
	v.Header.Set("If-Match", "changed")
	if string(v.Wire.Body["progress"]) != `"١٢"` || v.Wire.Header.Get("If-Match") != "actual" || v.Resource.Header.Get("If-Match") != "actual" {
		t.Fatal(v)
	}
}
func TestPythonServerFindRecordEmptyMalformedShapeAndDescriptorProof(t *testing.T) {
	for _, item := range []struct {
		body  string
		valid bool
	}{{"", true}, {"{bad", true}, {`{}`, true}, {`{"server":{}}`, true}, {`null`, false}, {`[]`, false}, {`false`, false}, {`{"server":null}`, false}, {`{"server":[]}`, false}, {`{"server":{"progress":"²"}}`, false}, {"{\"server\":{\"name\":\"\xff\"}}", false}} {
		t.Run(fmt.Sprintf("%q", item.body), func(t *testing.T) {
			var calls []nativeServerCall
			api, _ := serverFindAPI(t, &calls, func(*http.Request) *http.Response { return nativeServerWire(202, item.body) })
			v, err := api.FindRecord(context.Background(), "seed", servers.WithRecordFindQuery("progress", "12"))
			if (err == nil) != item.valid || v == nil || v.StatusCode != 202 || string(v.Envelope) != item.body || len(calls) != 1 {
				t.Fatal(item, v, err, calls)
			}
			if item.valid {
				serverFindValue(t, v, "id", `"seed"`)
				serverFindValue(t, v, "progress", "12")
				serverFindValue(t, v, "tags", "[]")
			} else {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != item.body {
					t.Fatal(err, proof)
				}
			}
		})
	}
}
func TestPythonServerFindRecordNestedSourceDescriptors(t *testing.T) {
	var calls []nativeServerCall
	api, _ := serverFindAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeServerWire(200, `{"server":{"flavor":{"original_name":"flavor-name","ram":"٢","vcpus":true,"is_public":0,"extra_specs":false,"x_unknown":7},"image":{"id":"img","owner":false,"min_disk":"٣","protected":"false","visibility":"private","hw_qemu_guest_agent":false,"properties":"wrapped","x_ext":0,"OpenStack-image-import-methods":"web-download"},"security_groups":[null,false,{"name":"default","x_ext":0}],"os-extended-volumes:volumes_attached":[{"volumeId":"vol","bdm_uuid":"bdm","delete_on_termination":false,"unknown":0},null],"personality":false,"addresses":false}}`)
	})
	v, err := api.FindRecord(context.Background(), "s1")
	if err != nil {
		t.Fatal(v, err)
	}
	flavor := serverFindObject(t, v, "flavor")
	for key, want := range map[string]string{"id": `"flavor-name"`, "name": `"flavor-name"`, "ram": "2", "vcpus": "true", "is_public": "false", "disk": "0", "extra_specs": "{}", "location": "null"} {
		if string(flavor[key]) != want {
			t.Fatal(key, string(flavor[key]), want)
		}
	}
	if flavor["x_unknown"] != nil {
		t.Fatal(flavor)
	}
	image := serverFindObject(t, v, "image")
	for key, want := range map[string]string{"id": `"img"`, "owner": "false", "owner_id": "false", "min_disk": `"٣"`, "is_protected": "true", "hw_qemu_guest_agent": `"False"`, "tags": "[]", "location": "null"} {
		if string(image[key]) != want {
			t.Fatal(key, string(image[key]), want)
		}
	}
	var properties map[string]json.RawMessage
	if json.Unmarshal(image["properties"], &properties) != nil || string(properties["properties"]) != `"wrapped"` || string(properties["x_ext"]) != "0" || properties["OpenStack-image-import-methods"] != nil {
		t.Fatal(properties)
	}
	serverFindValue(t, v, "security_groups", `[{},{},{"name":"default","x_ext":0}]`)
	serverFindValue(t, v, "addresses", `{}`)
	serverFindValue(t, v, "personality", `[false]`)
	var attached []map[string]json.RawMessage
	if json.Unmarshal(v.Resource.Body["attached_volumes"], &attached) != nil || len(attached) != 2 || string(attached[0]["id"]) != `"vol"` || string(attached[0]["volume_id"]) != `"vol"` || string(attached[0]["bdm_id"]) != `"bdm"` || attached[0]["unknown"] != nil || len(attached[1]) != 0 || !reflect.DeepEqual(v.Resource.Body["volumes"], v.Resource.Body["attached_volumes"]) {
		t.Fatal(attached, v)
	}
}
func TestPythonServerFindRecordNestedConversionsAndConstructorProfile(t *testing.T) {
	for _, row := range []string{`{"flavor":{"self":null}}`, `{"image":{"connection":false}}`, `{"image":{"hw_vif_multiqueue_enabled":"invalid"}}`, `{"image":{"OpenStack-image-import-methods":true}}`, `{"flavor":{"ram":"²"}}`} {
		var calls []nativeServerCall
		api, _ := serverFindAPI(t, &calls, func(*http.Request) *http.Response { return nativeServerWire(200, `{"server":`+row+`}`) })
		v, err := api.FindRecord(context.Background(), "wanted")
		var proof *resource.ResponseError
		if v == nil || err == nil || !errors.As(err, &proof) || proof.StatusCode != 200 {
			t.Fatal(row, v, err, proof)
		}
	}
}
func TestPythonServerFindRecordFallbackStatusMissingAndAuthority(t *testing.T) {
	for _, code := range []int{400, 403, 404, 401, 409} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []nativeServerCall
			api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Path == serverFindDetail {
					return nativeServerWire(200, `{"servers":[{"id":"wanted","name":"different"}]}`)
				}
				return nativeServerWire(code, `{"missing":"actual"}`)
			})
			api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				return original
			}
			v, err := api.FindRecord(context.Background(), "wanted")
			if code == 401 || code == 409 {
				if v != nil || !gophercloud.ResponseCodeIs(err, code) || len(calls) != 1 {
					t.Fatal(v, err, calls)
				}
				return
			}
			if err != nil || v == nil || len(calls) != 2 {
				t.Fatal(v, err, calls)
			}
			q, _ := url.ParseQuery(calls[1].query)
			if q.Get("name") != "wanted" || q.Has("all_tenants") {
				t.Fatal(q)
			}
		})
	}
	var calls []nativeServerCall
	api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path == serverFindDetail {
			return nativeServerWire(200, `{"servers":[]}`)
		}
		return nativeServerWire(404, `{}`)
	})
	v, err := api.FindRecord(context.Background(), "missing")
	if v != nil || err != nil || len(calls) != 2 {
		t.Fatal(v, err, calls)
	}
	v, err = api.FindRecord(context.Background(), "missing", servers.WithRecordFindIgnoreMissing(false))
	if v != nil || !errors.Is(err, resource.ErrNotFound) || gophercloud.ResponseCodeIs(err, 404) {
		t.Fatal(v, err)
	}
	calls = nil
	api, _ = serverFindAPI(t, &calls, func(*http.Request) *http.Response { return nativeServerWire(403, `{"forbidden":"actual"}`) })
	v, err = api.FindRecord(context.Background(), "wanted", servers.WithRecordFindAllProjects(true))
	if v != nil || !gophercloud.ResponseCodeIs(err, 403) || len(calls) != 2 {
		t.Fatal(v, err, calls)
	}
	q, _ := url.ParseQuery(calls[1].query)
	if q.Get("all_tenants") != "True" {
		t.Fatal(q)
	}
}
func TestPythonServerFindRecordDetailsScopeQueryPriorityAndOrderedSeed(t *testing.T) {
	var calls []nativeServerCall
	var accept []string
	api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
		accept = append(accept, req.Header.Get("Accept"))
		if req.URL.Path != serverFindBase {
			return nativeServerWire(404, `{}`)
		}
		return nativeServerWire(200, `{"servers":[{"id":"wanted","name":"wanted","fault":{"enabled":1,"extra":false}}]}`)
	})
	api.RawClient().MoreHeaders["Accept"] = "member-custom"
	v, err := api.FindRecord(context.Background(), "wanted", servers.WithRecordFindDetails(false), servers.WithRecordFindAllProjects(true), servers.WithRecordFindQuery("changes-since", "wire"), servers.WithRecordFindQuery("changes_since", nil), servers.WithRecordFindQuery("name", nil), servers.WithRecordFindQuery("any_tags", []any{"one", nil, "two"}), servers.WithRecordFindQuery("fault", map[string]any{"enabled": true}), servers.WithRecordFindQuery("x_extra", false))
	if err != nil || v == nil || len(calls) != 2 || !reflect.DeepEqual(accept, []string{"member-custom", "application/json"}) {
		t.Fatal(v, err, calls, accept)
	}
	q, _ := url.ParseQuery(calls[1].query)
	if q.Has("name") || q.Has("changes-since") || q.Has("fault") || q.Has("x_extra") || q.Get("all_tenants") != "True" || !reflect.DeepEqual(q["tags-any"], []string{"one", "two"}) {
		t.Fatal(q)
	}
	calls = nil
	api, _ = serverFindAPI(t, &calls, func(*http.Request) *http.Response { return nativeServerWire(200, `{}`) })
	v, err = api.FindRecord(context.Background(), "wanted", servers.WithRecordFindQuery("is_locked", false), servers.WithRecordFindQuery("locked", true))
	if err != nil {
		t.Fatal(err)
	}
	serverFindValue(t, v, "is_locked", "true")
	calls = nil
	v, err = api.FindRecord(context.Background(), "wanted", servers.WithRecordFindQuery("locked", true), servers.WithRecordFindQuery("is_locked", false))
	if err != nil {
		t.Fatal(err)
	}
	serverFindValue(t, v, "is_locked", "false")
}
func TestPythonServerFindRecordDictionaryTruthinessAndNumericEquality(t *testing.T) {
	for _, item := range []struct {
		actual, filter string
		match, valid   bool
	}{{`"truthy"`, `{}`, true, true}, {`"truthy"`, `{"k":true}`, false, false}, {`""`, `{}`, false, true}, {`1e-4000`, `{}`, false, true}, {`1e4000`, `{}`, true, true}, {`null`, `null`, true, true}, {`1`, `true`, true, true}, {`{"nested":{"enabled":1,"extra":false}}`, `{"nested":{"enabled":true}}`, true, true}, {`{"nested":{}}`, `{"nested":{}}`, false, true}} {
		t.Run(item.actual+item.filter, func(t *testing.T) {
			var calls []nativeServerCall
			api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Path != serverFindDetail {
					return nativeServerWire(404, `{}`)
				}
				return nativeServerWire(200, `{"servers":[{"id":"wanted","fault":`+item.actual+`}]}`)
			})
			v, err := api.FindRecord(context.Background(), "wanted", servers.WithRecordFindQuery("fault", json.RawMessage(item.filter)))
			if (v != nil) != item.match || (err == nil) != item.valid {
				t.Fatal(item, v, err)
			}
			if !item.valid {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 {
					t.Fatal(err, proof)
				}
			}
		})
	}
}
func TestPythonServerFindRecordDiscoveryCeilingAndSelectedVersions(t *testing.T) {
	for _, item := range []struct{ selected, maximum, minimum, want string }{{"", "2.999", "2.1", "2.100"}, {"", "2.80", "2.1", "2.80"}, {"", "2.110", "2.101", ""}, {"2.999", "2.80", "2.1", "2.999"}, {"latest", "2.80", "2.1", "latest"}} {
		t.Run(item.selected+item.maximum+item.minimum, func(t *testing.T) {
			var headers []http.Header
			client, _, calls := pythonServerCloud(t, func(req *http.Request) *http.Response {
				headers = append(headers, req.Header.Clone())
				if req.URL.Path == "/nova/v2.1/" {
					return nativeServerWire(200, pythonBackupAdvertisement(item.maximum, item.minimum))
				}
				if req.URL.Path == serverFindDetail {
					return nativeServerWire(200, `{"servers":[{"id":"wanted"}]}`)
				}
				return nativeServerWire(404, `{}`)
			})
			client.Endpoint = client.ResourceBase
			client.Microversion = item.selected
			v, err := servers.New(client).FindRecord(context.Background(), "wanted")
			wantCount := 2
			observations := 0
			if item.selected == "" {
				wantCount = 3
				observations = 1
			}
			if err != nil || v == nil || len(*calls) != wantCount || v.Microversion != item.want || len(v.Discovery) != observations || client.Microversion != item.selected {
				t.Fatal(item, v, err, *calls)
			}
			for i, h := range headers {
				want := item.want
				if item.selected == "" && i == 0 {
					want = ""
				}
				generic := ""
				if want != "" {
					generic = "compute " + want
				}
				if h.Get("X-OpenStack-Nova-API-Version") != want || h.Get("OpenStack-API-Version") != generic {
					t.Fatal(item, i, h)
				}
			}
			if observations > 0 {
				if v.Discovery[0].StatusCode != 200 || v.Discovery[0].Header.Get("X-Server-Proof") != "actual" || !strings.Contains(string(v.Discovery[0].Envelope), item.maximum) {
					t.Fatal(v.Discovery)
				}
				v.Discovery[0].Header.Set("X-Server-Proof", "changed")
				if v.Header.Get("X-Server-Proof") != "actual" {
					t.Fatal(v)
				}
			}
		})
	}
}
func TestPythonServerFindRecordExplicitVersionIsGETOnly(t *testing.T) {
	for _, version := range []string{"", "2.60", "latest"} {
		var calls []nativeServerCall
		var headers []http.Header
		api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
			headers = append(headers, req.Header.Clone())
			if req.URL.Path == serverFindDetail {
				return nativeServerWire(200, `{"servers":[{"id":"wanted"}]}`)
			}
			return nativeServerWire(404, `{}`)
		})
		v, err := api.FindRecord(context.Background(), "wanted", servers.WithRecordFindMicroversion(version))
		if err != nil || v == nil || v.Microversion != "2.96" || len(calls) != 2 || api.RawClient().Microversion != "2.96" || headers[0].Get("X-OpenStack-Nova-API-Version") != version || headers[1].Get("X-OpenStack-Nova-API-Version") != "2.96" {
			t.Fatal(version, v, err, calls, headers)
		}
	}
	client, _, calls := pythonServerCloud(t, func(req *http.Request) *http.Response {
		if req.URL.Path == "/nova/v2.1/" {
			return nativeServerWire(200, pythonBackupAdvertisement("2.90", "2.1"))
		}
		if req.URL.Path == serverFindDetail {
			return nativeServerWire(200, `{"servers":[{"id":"wanted"}]}`)
		}
		return nativeServerWire(404, `{}`)
	})
	client.Endpoint = client.ResourceBase
	client.Microversion = ""
	v, err := servers.New(client).FindRecord(context.Background(), "wanted", servers.WithRecordFindMicroversion("2.50"))
	if err != nil || v == nil || len(*calls) != 3 || (*calls)[0].version != "2.50" || (*calls)[1].path != "/nova/v2.1/" || (*calls)[2].version != "2.90" || len(v.Discovery) != 1 {
		t.Fatal(v, err, *calls)
	}
}
func TestPythonServerFindRecordDiscoveryTerminalProofAndSourceGuard(t *testing.T) {
	for _, lane := range []string{"malformed", "forbidden", "source"} {
		var client *gophercloud.ServiceClient
		client, _, calls := pythonServerCloud(t, func(req *http.Request) *http.Response {
			if lane == "source" {
				client.ResourceBase += "changed/"
				return nativeServerWire(200, pythonBackupAdvertisement("2.100", "2.1"))
			}
			if lane == "forbidden" {
				return nativeServerWire(403, `{"discovery":"forbidden"}`)
			}
			return nativeServerWire(200, `{bad`)
		})
		client.Endpoint = client.ResourceBase
		client.Microversion = ""
		original := client.ResourceBase
		v, err := servers.New(client).FindRecord(context.Background(), "wanted")
		client.ResourceBase = original
		if v != nil || err == nil || len(*calls) != 1 {
			t.Fatal(lane, v, err, *calls)
		}
		if lane == "forbidden" {
			if !gophercloud.ResponseCodeIs(err, 403) {
				t.Fatal(err)
			}
		} else {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 200 {
				t.Fatal(err, proof)
			}
			if lane == "source" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		}
	}
}
func TestPythonServerFindRecordInvalidControlsBeforeHTTP(t *testing.T) {
	var calls []nativeServerCall
	api, _ := serverFindAPI(t, &calls, func(*http.Request) *http.Response { return nativeServerWire(200, `{}`) })
	options := [][]servers.RecordFindOption{{nil}, {servers.WithRecordFindLimit(-1)}, {servers.WithRecordFindMaxItems(-1)}, {servers.WithRecordFindQuery("progress", "²")}, {servers.WithRecordFindQuery("x", json.RawMessage(`bad`))}, {servers.WithRecordFindAllProjects(false), servers.WithRecordFindQuery("all_tenants", nil)}, {servers.WithRecordFindHeaders(map[string]string{"X-Auth-Token": "bad"})}, {servers.WithRecordFindHeaders(map[string]string{"OpenStack-API-Version": "compute 2.5"})}, {servers.WithRecordFindHeaders(map[string]string{"X-OpenStack-Volume-API-Version": "3.5"})}}
	for _, key := range []string{"ignore_missing", "name_or_id", "session", "cls", "id", "self", "connection", "_synchronized", "details", "all_projects", "base_path", "list_base_path", "resource_type", "requires_id", "resource_response_key", "skip_cache", "error_message", "headers", "microversion", "paginated", "max_items", "location", "interface_ip"} {
		options = append(options, []servers.RecordFindOption{servers.WithRecordFindQuery(key, nil)})
	}
	for _, opts := range options {
		_, err := api.FindRecord(context.Background(), "wanted", opts...)
		if err == nil || len(calls) != 0 {
			t.Fatal(err, calls)
		}
	}
	for _, identity := range []string{"", "..", "a\n", "\xff"} {
		_, err := api.FindRecord(context.Background(), identity)
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(err, calls)
		}
	}
	api.RawClient().Type = "network"
	_, err := api.FindRecord(context.Background(), "wanted")
	if !errors.Is(err, resource.ErrUnsupported) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}
func TestPythonServerFindRecordSnapshotCancellationAndOptionSource(t *testing.T) {
	var calls []nativeServerCall
	var header http.Header
	api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response { header = req.Header.Clone(); return nativeServerWire(204, "") })
	query := map[string]json.RawMessage{"name": json.RawMessage(`"original"`)}
	version := "2.70"
	option := servers.WithRecordFindOpts(servers.RecordFindOpts{Query: query, Headers: map[string]string{"X-Trace": "original"}, Microversion: &version})
	query["name"][1] = 'X'
	version = "changed"
	var retained *servers.RecordFindOpts
	v, err := api.FindRecord(context.Background(), "wanted", option, func(p *servers.RecordFindOpts) error { retained = p; return nil }, func(*servers.RecordFindOpts) error {
		retained.Headers["X-Auth-Token"] = "bad"
		retained.Query["name"] = json.RawMessage(`false`)
		*retained.Microversion = "changed"
		return nil
	})
	if err != nil || v == nil || len(calls) != 1 || header.Get("X-Trace") != "original" || header.Get("X-OpenStack-Nova-API-Version") != "2.70" {
		t.Fatal(v, err, calls, header)
	}
	serverFindValue(t, v, "name", `"original"`)
	calls = nil
	ctx, cancel := context.WithCancelCause(context.Background())
	cause, callback := errors.New("cancel cause"), errors.New("callback")
	_, err = api.FindRecord(ctx, "wanted", func(*servers.RecordFindOpts) error { cancel(cause); return callback })
	if len(calls) != 0 || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, callback) {
		t.Fatal(err, calls)
	}
	original := api.RawClient().ResourceBase
	_, err = api.FindRecord(context.Background(), "wanted", func(*servers.RecordFindOpts) error { api.RawClient().ResourceBase += "changed/"; return callback })
	api.RawClient().ResourceBase = original
	if len(calls) != 0 || !errors.Is(err, callback) || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err, calls)
	}
}
func TestPythonServerFindRecordConcurrentOwnedOptions(t *testing.T) {
	var calls []nativeServerCall
	api, cloud := serverFindAPI(t, &calls, func(*http.Request) *http.Response { return nativeServerWire(200, `{}`) })
	var attempts atomic.Int64
	cloud.Provider.HTTPClient.Transport = nativeServerTransport(func(req *http.Request) (*http.Response, error) {
		attempts.Add(1)
		if req.Header.Get("X-Trace") != "owned" {
			t.Error(req.Header)
		}
		return nativeServerWire(200, `{"server":{"id":"wanted","metadata":{"x":false}}}`), nil
	})
	option := servers.WithRecordFindOpts(servers.RecordFindOpts{Query: map[string]json.RawMessage{"name": json.RawMessage(`"seed"`)}, Headers: map[string]string{"X-Trace": "owned"}})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := api.FindRecord(context.Background(), "wanted", option)
			if err != nil || v == nil {
				t.Error(v, err)
				return
			}
			v.Resource.Body["metadata"][0] = 'X'
			if string(v.Wire.Body["metadata"]) != `{"x":false}` {
				t.Error(v)
			}
		}()
	}
	wg.Wait()
	if attempts.Load() != 8 {
		t.Fatal(attempts.Load())
	}
}

type serverFindFaultBody struct {
	data              []byte
	readErr, closeErr error
	onRead, onClose   func()
	closed            int
}

func (b *serverFindFaultBody) Read(p []byte) (int, error) {
	if len(b.data) > 0 {
		n := copy(p, b.data)
		b.data = b.data[n:]
		if b.onRead != nil {
			b.onRead()
			b.onRead = nil
		}
		return n, nil
	}
	if b.readErr != nil {
		return 0, b.readErr
	}
	return 0, io.EOF
}
func (b *serverFindFaultBody) Close() error {
	b.closed++
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

func TestPythonServerFindRecordPhysicalAcceptedRejectedAndListErrors(t *testing.T) {
	for _, phase := range []string{"member", "list"} {
		for _, status := range []int{200, 400, 403, 404, 409, 503} {
			for _, lane := range []string{"read", "close", "read+close", "cancel", "source"} {
				t.Run(fmt.Sprintf("%s/%d/%s", phase, status, lane), func(t *testing.T) {
					var calls []nativeServerCall
					api, cloud := serverFindAPI(t, &calls, func(*http.Request) *http.Response { return nativeServerWire(200, `{}`) })
					readCause, closeCause, cancelCause := errors.New("physical read"), errors.New("physical close"), errors.New("custom cancel")
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					client := api.RawClient()
					original := client.ResourceBase
					defer func() { client.ResourceBase = original }()
					physical := &serverFindFaultBody{data: []byte(`{"actual":"physical"}`)}
					switch lane {
					case "read":
						physical.readErr = readCause
					case "close":
						physical.closeErr = closeCause
					case "read+close":
						physical.readErr = readCause
						physical.closeErr = closeCause
					case "cancel":
						physical.onRead = func() { cancel(cancelCause) }
					case "source":
						physical.onRead = func() { client.ResourceBase += "changed/" }
						physical.onClose = func() { client.ResourceBase = original }
					}
					attempts := 0
					cloud.Provider.HTTPClient.Transport = nativeServerTransport(func(req *http.Request) (*http.Response, error) {
						attempts++
						if phase == "list" && req.URL.Path != serverFindDetail {
							return nativeServerWire(404, `{}`), nil
						}
						return &http.Response{StatusCode: status, Header: http.Header{"X-Request-Id": {"physical"}}, Body: physical}, nil
					})
					record, err := api.FindRecord(ctx, "wanted")
					want := 1
					if phase == "list" {
						want = 2
					}
					if err == nil || attempts != want || physical.closed != 1 {
						t.Fatal(record, err, attempts, physical.closed)
					}
					if status < 400 {
						var receipt *resource.ResponseError
						if !errors.As(err, &receipt) || receipt.StatusCode != status || string(receipt.Body) != `{"actual":"physical"}` || receipt.Header.Get("X-Request-Id") != "physical" || phase == "member" && (record == nil || record.Resource != nil) || phase == "list" && record != nil {
							t.Fatal(record, err, receipt)
						}
					} else {
						var native gophercloud.ErrUnexpectedResponseCode
						if record != nil || !errors.As(err, &native) || native.Actual != status || string(native.Body) != `{"actual":"physical"}` || native.ResponseHeader.Get("X-Request-Id") != "physical" {
							t.Fatal(record, err, native)
						}
					}
					if (lane == "read" || lane == "read+close") && !errors.Is(err, readCause) || (lane == "close" || lane == "read+close") && !errors.Is(err, closeCause) || lane == "cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) || lane == "source" && !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				})
			}
		}
	}
}

func TestPythonServerFindRecordRetryKeepsOriginalRejectionAndRequest(t *testing.T) {
	for _, lane := range []string{"body", "version", "custom", "codes", "retry", "reauth"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativeServerCall
			var tokens []string
			api, cloud := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
				tokens = append(tokens, req.Header.Get("X-Auth-Token"))
				if len(calls) == 1 {
					if lane == "reauth" {
						return nativeServerWire(401, `{"expired":true}`)
					}
					return nativeServerWire(503, `{"retry":"actual"}`)
				}
				if lane == "codes" {
					return nativeServerWire(400, `{"outside":"original policy"}`)
				}
				return nativeServerWire(200, `{"server":{"id":"wanted"}}`)
			})
			api.RawClient().Microversion = "3.4"
			callback := errors.New("retry callback")
			if lane == "reauth" {
				cloud.Provider.ReauthFunc = func(context.Context) error { cloud.Provider.SetToken("renewed"); return nil }
			} else {
				api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
					switch lane {
					case "body":
						opts.JSONBody = map[string]any{"changed": true}
					case "version":
						opts.MoreHeaders["OpenStack-API-Version"] = "compute 3.5"
					case "custom":
						return callback
					case "codes":
						opts.OkCodes = append(opts.OkCodes, 400)
					}
					return nil
				}
			}
			record, err := api.FindRecord(context.Background(), "wanted", servers.WithRecordFindQuery("x_query", false))
			if lane == "retry" || lane == "reauth" {
				if err != nil || record == nil || len(calls) != 2 || calls[0] != calls[1] {
					t.Fatal(record, err, calls)
				}
				if lane == "reauth" && !reflect.DeepEqual(tokens, []string{"test-token", "renewed"}) {
					t.Fatal(tokens)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				wantCode, wantCalls := 503, 1
				if lane == "codes" {
					wantCode, wantCalls = 400, 2
				}
				if record != nil || !errors.As(err, &native) || native.Actual != wantCode || len(calls) != wantCalls {
					t.Fatal(record, err, native, calls)
				}
				if lane == "custom" {
					if !errors.Is(err, callback) {
						t.Fatal(err)
					}
				} else if lane != "codes" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestPythonServerFindRecordLateDuplicateAndPageFailure(t *testing.T) {
	for _, lane := range []string{"unique", "duplicate", "error"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativeServerCall
			api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Path != serverFindDetail {
					return nativeServerWire(404, `{}`)
				}
				if req.URL.Query().Get("marker") == "next" {
					if lane == "error" {
						return nativeServerWire(500, `{"late":"actual"}`)
					}
					if lane == "duplicate" {
						return nativeServerWire(200, `{"servers":[{"id":"another","name":"wanted"}]}`)
					}
					return nativeServerWire(200, `{"servers":[{"id":"other","name":"other"}]}`)
				}
				return nativeServerWire(200, `{"servers":[{"id":"wanted","name":"different"}],"servers_links":[{"rel":"next","href":"?marker=next"}]}`)
			})
			record, err := api.FindRecord(context.Background(), "wanted")
			if len(calls) != 3 {
				t.Fatal(calls)
			}
			if lane == "unique" {
				if err != nil || record == nil {
					t.Fatal(record, err)
				}
				serverFindValue(t, record, "id", `"wanted"`)
			} else {
				if record != nil || err == nil || lane == "duplicate" && !errors.Is(err, resource.ErrAmbiguous) || lane == "error" && !gophercloud.ResponseCodeIs(err, 500) {
					t.Fatal(record, err)
				}
			}
		})
	}
}
func TestPythonServerFindRecordMarkerCapAndSinglePage(t *testing.T) {
	var calls []nativeServerCall
	api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path != serverFindDetail {
			return nativeServerWire(404, `{}`)
		}
		if req.URL.Query().Get("marker") == "wanted" {
			return nativeServerWire(200, `{"servers":[]}`)
		}
		return nativeServerWire(200, `{"servers":{"id":"wanted","name":"wanted"}}`)
	})
	record, err := api.FindRecord(context.Background(), "wanted", servers.WithRecordFindLimit(5))
	if err != nil || record == nil || len(calls) != 3 || !strings.Contains(calls[2].query, "marker=wanted") || calls[0].query != "" {
		t.Fatal(record, err, calls)
	}
	calls = nil
	record, err = api.FindRecord(context.Background(), "wanted", servers.WithRecordFindLimit(5), servers.WithRecordFindPaginated(false))
	if err != nil || record == nil || len(calls) != 2 {
		t.Fatal(record, err, calls)
	}
	calls = nil
	api, _ = serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path != serverFindDetail {
			return nativeServerWire(404, `{}`)
		}
		return nativeServerWire(200, `{"servers":[{"id":"skip","name":"wanted","fault":false},{"id":"wanted","name":"wanted","fault":1},{"id":"bad","name":"wanted","connection":null}],"next":"https://bad.example/servers/detail"}`)
	})
	record, err = api.FindRecord(context.Background(), "wanted", servers.WithRecordFindQuery("fault", true), servers.WithRecordFindMaxItems(2), servers.WithRecordFindLimit(0))
	if err != nil || record == nil || len(calls) != 2 {
		t.Fatal(record, err, calls)
	}
	serverFindValue(t, record, "id", `"wanted"`)
	q, _ := url.ParseQuery(calls[1].query)
	if q.Get("limit") != "2" {
		t.Fatal(q)
	}
}
func TestPythonServerFindRecordPaginationRepresentationsAndGuards(t *testing.T) {
	for _, lane := range []string{"plural", "next", "http", "dictionary", "versioned", "empty", "filter", "path", "cycle"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativeServerCall
			body := `{"servers":[{"id":"first","name":"other"}]}`
			api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Path != serverFindDetail {
					return nativeServerWire(404, `{}`)
				}
				if req.URL.Query().Get("marker") != "" {
					if lane == "cycle" {
						return nativeServerWire(200, `{"servers":[{"id":"again"}],"next":"?marker=next"}`)
					}
					return nativeServerWire(200, `{"servers":[{"id":"wanted","name":"wanted"}]}`)
				}
				response := nativeServerWire(200, body)
				switch lane {
				case "plural":
					response = nativeServerWire(200, `{"servers":[{"id":"first"}],"servers_links":[{"rel":"next","href":"?marker=next"}]}`)
				case "next", "cycle":
					response = nativeServerWire(200, `{"servers":[{"id":"first"}],"next":"?marker=next"}`)
				case "http":
					response.Header.Set("Link", "<?marker=next>; rel=\"next\"")
				case "dictionary":
					response = nativeServerWire(200, `{"servers":[{"id":"first"}],"links":{"next":"?marker=next"}}`)
				case "versioned":
					response = nativeServerWire(200, `{"servers":[{"id":"first"}],"next":"/v2.1/servers/detail?marker=next"}`)
				case "empty":
					response = nativeServerWire(200, `{"servers":[],"next":"https://bad.example/servers/detail"}`)
				case "filter":
					response = nativeServerWire(200, `{"servers":[{"id":"first"}],"next":"?marker=next&project_id=changed"}`)
				case "path":
					response = nativeServerWire(200, `{"servers":[{"id":"first"}],"next":"/other/servers/detail?marker=next"}`)
				}
				return response
			})
			record, err := api.FindRecord(context.Background(), "wanted", servers.WithRecordFindQuery("project_id", "fixed"))
			if lane == "filter" || lane == "path" || lane == "cycle" {
				var receipt *resource.ResponseError
				if record != nil || err == nil || !errors.As(err, &receipt) || receipt.StatusCode != 200 {
					t.Fatal(record, err, receipt, calls)
				}
			} else if lane == "empty" {
				if record != nil || err != nil || len(calls) != 2 {
					t.Fatal(record, err, calls)
				}
			} else if record == nil || err != nil || len(calls) != 3 {
				t.Fatal(record, err, calls)
			}
		})
	}
}

func TestPythonServerFindRecordRetryForbiddenHeadersAndOmittedVersion(t *testing.T) {
	for _, list := range []bool{false, true} {
		for _, key := range []string{"aUtHoRiZaTiOn", "x-SeRvIcE-ToKeN", "hOsT", "Cookie", "content-length", "TRANSFER-ENCODING", "Connection", "Trailer", "Te", "Upgrade", "x-OpenStack-Nova-Api-Version", "x-OpenStack-Volume-Api-Version", "x-OpenStack-Manila-Api-Version", "X-OpenStack-Ironic-API-Version", "omit-selected-version", "omit-legacy-version"} {
			for _, custom := range []bool{false, true} {
				t.Run(fmt.Sprintf("%t/%s/%t", list, key, custom), func(t *testing.T) {
					var calls []nativeServerCall
					callback := errors.New("callback cause")
					body := `{"error":"original rejection"}`
					api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
						if list && req.URL.Path != serverFindDetail {
							return nativeServerWire(404, `{}`)
						}
						response := nativeServerWire(503, body)
						response.Header.Set("X-Request-Id", "fixed-policy-rejection")
						return response
					})
					api.RawClient().Microversion = "2.96"
					api.RawClient().RetryFunc = func(_ context.Context, _ string, endpoint string, opts *gophercloud.RequestOpts, original error, _ uint) error {
						parsed, err := url.Parse(endpoint)
						if err != nil {
							t.Fatal(err)
						}
						if list && parsed.Path != serverFindDetail {
							return original
						}
						if key == "omit-legacy-version" {
							opts.OmitHeaders = []string{"x-openstack-nova-api-version"}
						} else if key == "omit-selected-version" {
							opts.OmitHeaders = []string{"oPeNsTaCk-ApI-vErSiOn"}
						} else {
							opts.MoreHeaders[key] = "forbidden"
						}
						if custom {
							return callback
						}
						return nil
					}
					record, err := api.FindRecord(context.Background(), "wanted", servers.WithRecordFindMicroversion("2.70"))
					var native gophercloud.ErrUnexpectedResponseCode
					wantCalls := 1
					if list {
						wantCalls = 2
					}
					if record != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.Is(err, callback) != custom || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "fixed-policy-rejection" || len(calls) != wantCalls {
						t.Fatal(record, err, native, calls)
					}
				})
			}
		}
	}
}

func TestPythonServerFindRecordListConstructorAndHeaderBoundary(t *testing.T) {
	for _, key := range []string{"cls", "connection", "microversion", "_synchronized"} {
		var calls []nativeServerCall
		body := `{"servers":[{"id":"wanted","` + key + `":null}]}`
		api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
			if req.URL.Path != serverFindDetail {
				return nativeServerWire(404, `{}`)
			}
			r := nativeServerWire(203, body)
			r.Header.Set("X-Request-Id", "constructor")
			return r
		})
		v, err := api.FindRecord(context.Background(), "wanted")
		var proof *resource.ResponseError
		if v != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "constructor" || len(calls) != 2 {
			t.Fatal(key, v, err, proof, calls)
		}
	}
	var calls []nativeServerCall
	api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path != serverFindDetail {
			return nativeServerWire(404, `{}`)
		}
		r := nativeServerWire(299, `{"servers":[{"id":null,"name":"wanted","if_match":"body","self":"reserved","unknown":false}]}`)
		r.Header.Set("If-Match", "actual")
		return r
	})
	v, err := api.FindRecord(context.Background(), "wanted")
	if err != nil || v == nil || v.StatusCode != 299 || v.Header.Get("If-Match") != "actual" || v.Resource.Body["if_match"] != nil || v.Resource.Body["unknown"] != nil || v.Resource.Body["self"] != nil || v.Wire.Body["self"] == nil || string(v.Wire.Body["if_match"]) != `"body"` {
		t.Fatal(v, err, calls)
	}
	serverFindValue(t, v, "id", "null")
}
func TestPythonServerFindRecordPassiveIdentityAndOriginalStatusPolicy(t *testing.T) {
	for _, row := range []string{`{"id":null,"name":"wanted"}`, `{"id":42,"name":"wanted"}`, `{"name":"wanted"}`} {
		var calls []nativeServerCall
		api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
			if req.URL.Path != serverFindDetail {
				return nativeServerWire(404, `{}`)
			}
			return nativeServerWire(200, `{"servers":[`+row+`]}`)
		})
		v, err := api.FindRecord(context.Background(), "wanted")
		if err != nil || v == nil || len(calls) != 2 {
			t.Fatal(row, v, err, calls)
		}
	}
	for _, code := range []int{200, 201, 202, 204, 299, 300, 301, 399} {
		var calls []nativeServerCall
		api, _ := serverFindAPI(t, &calls, func(*http.Request) *http.Response { return nativeServerWire(code, `{"server":{"id":"wanted"}}`) })
		v, err := api.FindRecord(context.Background(), "wanted")
		if err != nil || v == nil || v.StatusCode != code || len(calls) != 1 {
			t.Fatal(code, v, err, calls)
		}
	}
}
func TestPythonServerFindRecordLocationScopeAndFalseyProject(t *testing.T) {
	for _, item := range []struct{ project, want string }{{`"current"`, `"current"`}, {`"foreign"`, `"foreign"`}, {`null`, `"current"`}, {`false`, `"current"`}, {`1e-4000`, `"current"`}} {
		var calls []nativeServerCall
		name, cloudName, region := "project-name", "cloud", "region"
		location := resource.CloudLocation{Cloud: &cloudName, RegionName: &region, Project: resource.CloudProject{ID: json.RawMessage(`"current"`), Name: &name}}
		option := servers.WithRecordFindLocation(location)
		*location.Project.Name = "mutated"
		api, _ := serverFindAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeServerWire(200, `{"server":{"tenant_id":`+item.project+`,"OS-EXT-AZ:availability_zone":false}}`)
		})
		v, err := api.FindRecord(context.Background(), "wanted", option)
		if err != nil {
			t.Fatal(item, v, err)
		}
		owned := serverFindObject(t, v, "location")
		var project map[string]json.RawMessage
		if json.Unmarshal(owned["project"], &project) != nil || string(project["id"]) != item.want || string(owned["zone"]) != "false" || string(owned["cloud"]) != `"cloud"` {
			t.Fatal(item, owned, project)
		}
		wantName := `"project-name"`
		if item.project == `"foreign"` {
			wantName = "null"
		}
		if string(project["name"]) != wantName {
			t.Fatal(item, project)
		}
	}
}
func TestPythonServerFindRecordAcceptExplicitAndSelectedAPIChange(t *testing.T) {
	var calls []nativeServerCall
	var accept []string
	api, cloud := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
		accept = append(accept, req.Header.Get("Accept"))
		if req.URL.Path != serverFindDetail {
			return nativeServerWire(404, `{}`)
		}
		return nativeServerWire(200, `{"servers":[{"id":"wanted"}]}`)
	})
	api.RawClient().MoreHeaders["Accept"] = "configured"
	v, err := api.FindRecord(context.Background(), "wanted", servers.WithRecordFindHeaders(map[string]string{"aCcEpT": "owned"}))
	if err != nil || v == nil || !reflect.DeepEqual(accept, []string{"owned", "owned"}) {
		t.Fatal(v, err, calls, accept)
	}
	calls = nil
	original := *api
	replacement := nativeServerClient(cloud)
	replacement.Microversion = "2.96"
	_, err = api.FindRecord(context.Background(), "wanted", func(*servers.RecordFindOpts) error { *api = *servers.New(replacement); return nil })
	*api = original
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonServerFindRecordSourceIntegerDescriptorsAndWireProof(t *testing.T) {
	// fields._convert_type returns existing ints unchanged, including bool;
	// JSON floats truncate through binary64, Unicode isdigit strings convert,
	// and nonnumeric/signed/padded strings and ordinary containers become zero.
	for _, list := range []bool{false, true} {
		for _, item := range []struct {
			raw, want string
			valid     bool
		}{
			{`true`, `true`, true}, {`false`, `false`, true},
			{`null`, `null`, true}, {`9007199254740993`, `9007199254740993`, true},
			{`"9007199254740993"`, `9007199254740993`, true},
			{`"١٢"`, `12`, true}, {`"１２"`, `12`, true},
			{`2.9`, `2`, true}, {`-2.9`, `-2`, true},
			{`9007199254740993.0`, `9007199254740992`, true},
			{`"-12"`, `0`, true}, {`" 12 "`, `0`, true},
			{`"not numeric"`, `0`, true}, {`""`, `0`, true},
			{`[]`, `0`, true}, {`{}`, `0`, true},
			{`"²"`, ``, false}, {`1e4000`, ``, false},
		} {
			t.Run(fmt.Sprintf("list=%t/%s", list, item.raw), func(t *testing.T) {
				var calls []nativeServerCall
				row := `{"id":"wanted","progress":` + item.raw + `,"flavor":{"ram":` + item.raw + `}}`
				body := `{"server":` + row + `}`
				if list {
					body = `{"servers":[` + row + `]}`
				}
				api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
					if list && req.URL.Path != serverFindDetail {
						return nativeServerWire(404, `{}`)
					}
					r := nativeServerWire(203, body)
					r.Header.Set("X-Request-Id", "integer-proof")
					return r
				})
				record, err := api.FindRecord(context.Background(), "wanted")
				wantCalls := 1
				if list {
					wantCalls = 2
				}
				if (err == nil) != item.valid || len(calls) != wantCalls {
					t.Fatal(item, record, err, calls)
				}
				if item.valid {
					serverFindValue(t, record, "progress", item.want)
					flavor := serverFindObject(t, record, "flavor")
					if string(flavor["ram"]) != item.want || record.Wire == nil || string(record.Wire.Body["progress"]) != item.raw || string(record.Envelope) != body || record.StatusCode != 203 || record.Header.Get("X-Request-Id") != "integer-proof" {
						t.Fatal(item, record, flavor)
					}
				} else {
					var proof *resource.ResponseError
					if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "integer-proof" || list && record != nil || !list && record == nil {
						t.Fatal(item, record, err, proof)
					}
				}
			})
		}
	}
}

func TestPythonServerFindRecordNextBinary64TruthinessAndWholePageProof(t *testing.T) {
	for _, item := range []struct {
		next  string
		valid bool
	}{
		{`1e-400`, true}, {`0.0`, true}, {`null`, true}, {`[]`, true}, {`{}`, true},
		{`1e-300`, false}, {`1e400`, false},
	} {
		t.Run(item.next, func(t *testing.T) {
			var calls []nativeServerCall
			body := `{"servers":[{"id":"wanted","metadata":{"decimal":1e-400},"extension":1e-400}],"next":` + item.next + `}`
			api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Path != serverFindDetail {
					return nativeServerWire(404, `{}`)
				}
				r := nativeServerWire(203, body)
				r.Header.Set("X-Request-Id", "next-truthiness")
				return r
			})
			record, err := api.FindRecord(context.Background(), "wanted")
			if (err == nil) != item.valid || len(calls) != 2 {
				t.Fatal(item, record, err, calls)
			}
			if item.valid {
				if record == nil || record.Wire == nil || string(record.Envelope) != body || string(record.Wire.Body["extension"]) != `1e-400` || record.StatusCode != 203 {
					t.Fatal(item, record)
				}
				serverFindValue(t, record, "metadata", `{"decimal":1e-400}`)
				record.Resource.Body["metadata"][0] = 'X'
				record.Header.Set("X-Request-Id", "changed")
				if string(record.Wire.Body["metadata"]) != `{"decimal":1e-400}` || string(record.Envelope) != body || record.Wire.Header.Get("X-Request-Id") != "next-truthiness" {
					t.Fatal(item, record)
				}
			} else {
				var proof *resource.ResponseError
				if record != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "next-truthiness" {
					t.Fatal(item, record, err, proof)
				}
			}
		})
	}
	// A falsey continuation must never hide the physical source's read/Close
	// failure or replace its original whole-page receipt with a decoded row.
	var calls []nativeServerCall
	body := `{"servers":[{"id":"wanted"}],"next":1e-400}`
	readCause, closeCause := errors.New("next page read"), errors.New("next page close")
	physical := &serverFindFaultBody{data: []byte(body), readErr: readCause, closeErr: closeCause}
	api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path != serverFindDetail {
			return nativeServerWire(404, `{}`)
		}
		return &http.Response{StatusCode: 203, Header: http.Header{"X-Request-Id": {"next-physical"}}, Body: physical}
	})
	record, err := api.FindRecord(context.Background(), "wanted")
	var proof *resource.ResponseError
	if record != nil || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "next-physical" || physical.closed != 1 || len(calls) != 2 {
		t.Fatal(record, err, proof, physical.closed, calls)
	}
}
func TestPythonServerFindRecordInformationalStatusIsBoundedGoDifference(t *testing.T) {
	// Source raise_from_response returns for <400. Go intentionally accepts
	// final 200..399 representations only; interim 100..199 is not an ACK.
	for _, list := range []bool{false, true} {
		var calls []nativeServerCall
		body := `{"server":{"id":"wanted"}}`
		if list {
			body = `{"servers":[{"id":"wanted"}]}`
		}
		api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
			if list && req.URL.Path != serverFindDetail {
				return nativeServerWire(404, `{}`)
			}
			r := nativeServerWire(199, body)
			r.Header.Set("X-Request-Id", "informational")
			return r
		})
		record, err := api.FindRecord(context.Background(), "wanted")
		var native gophercloud.ErrUnexpectedResponseCode
		wantCalls := 1
		if list {
			wantCalls = 2
		}
		if record != nil || !errors.As(err, &native) || native.Actual != 199 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "informational" || len(calls) != wantCalls {
			t.Fatal(list, record, err, native, calls)
		}
	}
}
