package networksegmentranges_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networksegmentranges"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonSegmentRangeTransport func(*http.Request) (*http.Response, error)

func (transport pythonSegmentRangeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonSegmentRangeCall struct {
	method, path, query, body string
	header                    http.Header
}

func pythonSegmentRangeAPI(t *testing.T, calls *[]pythonSegmentRangeCall, reply func(*http.Request) (int, string)) *networksegmentranges.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonSegmentRangeTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonSegmentRangeCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Clone()})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return networksegmentranges.New(client)
}

func pythonSegmentRangeField(t *testing.T, v *networksegmentranges.Record, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		var got json.RawMessage
		if v != nil && v.Resource != nil {
			got = v.Resource.Body[key]
		}
		t.Fatalf("%s: got %s want %s", key, got, want)
	}
}

func pythonSegmentRangeBody(t *testing.T, raw string, want map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal([]byte(raw), &got); err != nil || !reflect.DeepEqual(got, map[string]any{"network_segment_range": want}) {
		t.Fatal(raw, want, err)
	}
}

func pythonSegmentRangeOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "networksegmentranges" {
		t.Fatal("operation context", err)
	}
}

func pythonSegmentRangeStatus(t *testing.T, err error, code int) {
	t.Helper()
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatal("native status", code, err)
	}
}

const pythonSegmentRangeBase = "/neutron/v2.0/network_segment_ranges"

func TestPythonSegmentRangeCreateSendsProvidedAttrs(t *testing.T) {
	var calls []pythonSegmentRangeCall
	api := pythonSegmentRangeAPI(t, &calls, func(*http.Request) (int, string) {
		return 201, `{"network_segment_range":{"id":"r1","name":"vlan","network_type":"vlan","minimum":"100","maximum":200,"shared":1,"default":false,"used":{},"available":[100],"tenant_id":"p","revision_number":2}}`
	})
	v, err := api.Create(context.Background(), networksegmentranges.WithCreateField("name", "vlan"), networksegmentranges.WithCreateField("network_type", "vlan"),
		networksegmentranges.WithCreateField("minimum", 100), networksegmentranges.WithCreateField("maximum", 200), networksegmentranges.WithCreateField("shared", false),
		networksegmentranges.WithCreateField("project_id", nil), networksegmentranges.WithCreateField("x_extension", true), networksegmentranges.WithCreateField("microversion", "ignored"))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != pythonSegmentRangeBase || calls[0].query != "" || calls[0].header.Get("Content-Type") != "application/json" {
		t.Fatal(calls)
	}
	pythonSegmentRangeBody(t, calls[0].body, map[string]any{"name": "vlan", "network_type": "vlan", "minimum": float64(100), "maximum": float64(200), "shared": false, "project_id": nil, "x_extension": true})
	pythonSegmentRangeField(t, v, "id", `"r1"`)
	pythonSegmentRangeField(t, v, "minimum", `100`)
	pythonSegmentRangeField(t, v, "shared", `true`)
	pythonSegmentRangeField(t, v, "default", `false`)
	pythonSegmentRangeField(t, v, "project_id", `null`)
	pythonSegmentRangeField(t, v, "physical_network", `null`)
	pythonSegmentRangeField(t, v, "tenant_id", `"p"`)
	pythonSegmentRangeField(t, v, "x_extension", `true`)
	pythonSegmentRangeField(t, v, "location", `null`)
	if v.StatusCode != 201 || string(v.Wire.Body["minimum"]) != `"100"` || v.Text("network_type") != "vlan" {
		t.Fatal(v)
	}

	calls = nil
	if _, err := api.Create(context.Background()); err != nil || len(calls) != 1 || calls[0].body != `{"network_segment_range":{}}` {
		t.Fatal("Resource.create has no required-field precheck", err, calls)
	}
	calls = nil
	if _, err := api.Create(context.Background(), networksegmentranges.WithCreateField("base_path", "/x")); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal("route control fails before HTTP", err, calls)
	}
	api = pythonSegmentRangeAPI(t, &calls, func(*http.Request) (int, string) { return 409, `{"NeutronError":{}}` })
	_, err = api.Create(context.Background(), networksegmentranges.WithCreateField("name", "vlan"))
	pythonSegmentRangeStatus(t, err, 409)
	pythonSegmentRangeOperation(t, err, "Create")
}

func TestPythonSegmentRangeGetFieldsStatusesAndEscapedID(t *testing.T) {
	for _, code := range []int{200, 203, 400, 403, 404, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []pythonSegmentRangeCall
			api := pythonSegmentRangeAPI(t, &calls, func(*http.Request) (int, string) {
				return code, `{"network_segment_range":{"name":"vlan","available":5}}`
			})
			v, err := api.Get(context.Background(), "r1")
			if len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != pythonSegmentRangeBase+"/r1" || calls[0].query != "" || calls[0].body != "" {
				t.Fatal(calls)
			}
			if code < 400 {
				if err != nil || v.StatusCode != code {
					t.Fatal(v, err)
				}
				pythonSegmentRangeField(t, v, "id", `"r1"`)
				pythonSegmentRangeField(t, v, "available", `[5]`)
				return
			}
			if v != nil {
				t.Fatal(v)
			}
			pythonSegmentRangeStatus(t, err, code)
			pythonSegmentRangeOperation(t, err, "Get")
		})
	}
	var calls []pythonSegmentRangeCall
	api := pythonSegmentRangeAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{}` })
	if _, err := api.Get(context.Background(), "a/b c", networksegmentranges.WithGetFields("name", "minimum", "x_extension")); err != nil || calls[0].path != pythonSegmentRangeBase+"/a%2Fb%20c" || calls[0].query != "fields=name&fields=minimum&fields=x_extension" {
		t.Fatal(err, calls)
	}
	if _, err := api.Get(context.Background(), " "); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 {
		t.Fatal("blank ID fails before HTTP", err, calls)
	}
}

func TestPythonSegmentRangeUpdatePutsChangedAttrs(t *testing.T) {
	var calls []pythonSegmentRangeCall
	api := pythonSegmentRangeAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"network_segment_range":{"id":"r1","name":"new","maximum":300}}`
	})
	v, err := api.Update(context.Background(), "r1", networksegmentranges.WithUpdateField("name", "new"), networksegmentranges.WithUpdateField("maximum", 300))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != pythonSegmentRangeBase+"/r1" || calls[0].query != "" {
		t.Fatal(calls)
	}
	pythonSegmentRangeBody(t, calls[0].body, map[string]any{"name": "new", "maximum": float64(300)})
	pythonSegmentRangeField(t, v, "maximum", `300`)
	pythonSegmentRangeField(t, v, "minimum", `null`)

	calls = nil
	v, err = api.Update(context.Background(), "r1")
	if err != nil || len(calls) != 0 || v.StatusCode != 0 || v.Wire != nil {
		t.Fatal("no changed attribute sends nothing", v, err, calls)
	}
	pythonSegmentRangeField(t, v, "id", `"r1"`)
	if _, err := api.Update(context.Background(), "r1", networksegmentranges.WithUpdateField("id", "other")); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal("id duplicates the update identity", err, calls)
	}
	api = pythonSegmentRangeAPI(t, &calls, func(*http.Request) (int, string) { return 404, `missing` })
	_, err = api.Update(context.Background(), "r1", networksegmentranges.WithUpdateField("name", "x"))
	pythonSegmentRangeStatus(t, err, 404)
	pythonSegmentRangeOperation(t, err, "Update")
}

func TestPythonSegmentRangeDeleteIgnoresMissingByDefault(t *testing.T) {
	for _, tc := range []struct {
		name    string
		code    int
		options []networksegmentranges.DeleteOption
		ignored bool
		fails   bool
	}{
		{"deleted", 204, nil, false, false},
		{"missing ignored", 404, nil, true, false},
		{"missing reported", 404, []networksegmentranges.DeleteOption{networksegmentranges.WithDeleteIgnoreMissing(false)}, false, true},
		{"conflict", 409, nil, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonSegmentRangeCall
			api := pythonSegmentRangeAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, "" })
			result, err := api.Delete(context.Background(), "r1", tc.options...)
			if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != pythonSegmentRangeBase+"/r1" || calls[0].body != "" {
				t.Fatal(calls)
			}
			if tc.fails {
				pythonSegmentRangeStatus(t, err, tc.code)
				pythonSegmentRangeOperation(t, err, "Delete")
				return
			}
			if err != nil || result == nil || result.StatusCode != tc.code || result.IgnoredMissing != tc.ignored {
				t.Fatal(result, err)
			}
		})
	}
}

func TestPythonSegmentRangeListQueryFiltersAndLinksPaging(t *testing.T) {
	var calls []pythonSegmentRangeCall
	api := pythonSegmentRangeAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.URL.Query().Get("marker") {
		case "":
			return 200, `{"network_segment_ranges":[{"id":"a","network_type":"vlan"},{"id":"b","network_type":"vlan"}],"network_segment_ranges_links":[{"rel":"next","href":"http://` + req.Host + `/neutron/v2.0/network_segment_ranges?network_type=vlan&limit=2&marker=b"}]}`
		case "b":
			return 200, `{"network_segment_ranges":[{"id":"c","network_type":"vlan"}]}`
		}
		return 200, `{"network_segment_ranges":[]}`
	})
	var ids []string
	for v, err := range api.List(context.Background(), networksegmentranges.WithListQuery("network_type", "vlan"), networksegmentranges.WithListLimit(2),
		networksegmentranges.WithListQuery("id", "c"), networksegmentranges.WithListQuery("unknown", "dropped")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, v.Text("id"))
	}
	// The short page still continues by raw marker until an empty page.
	if !reflect.DeepEqual(ids, []string{"c"}) || len(calls) != 3 {
		t.Fatal(ids, calls)
	}
	if calls[0].method != http.MethodGet || calls[0].path != pythonSegmentRangeBase || calls[0].query != "limit=2&network_type=vlan" || calls[0].header.Get("Accept") != "application/json" {
		t.Fatal(calls[0])
	}
	if calls[1].query != "limit=2&marker=b&network_type=vlan" || calls[2].query != "limit=2&marker=c&network_type=vlan" {
		t.Fatal(calls[1:])
	}

	calls = nil
	values, err := api.All(context.Background(), networksegmentranges.WithListPaginated(false))
	if err != nil || len(values) != 2 || len(calls) != 1 || calls[0].query != "" {
		t.Fatal("paginated=False reads one page", values, err, calls)
	}
	api = pythonSegmentRangeAPI(t, &calls, func(*http.Request) (int, string) { return 403, `{}` })
	_, err = api.All(context.Background())
	pythonSegmentRangeStatus(t, err, 403)
	pythonSegmentRangeOperation(t, err, "List")
}

func TestPythonSegmentRangeFindFallsBackToNameList(t *testing.T) {
	t.Run("id", func(t *testing.T) {
		var calls []pythonSegmentRangeCall
		api := pythonSegmentRangeAPI(t, &calls, func(*http.Request) (int, string) {
			return 200, `{"network_segment_range":{"id":"r1","name":"vlan"}}`
		})
		v, err := api.Find(context.Background(), "r1")
		if err != nil || v.Text("name") != "vlan" || len(calls) != 1 || calls[0].path != pythonSegmentRangeBase+"/r1" {
			t.Fatal(v, err, calls)
		}
	})
	t.Run("name", func(t *testing.T) {
		var calls []pythonSegmentRangeCall
		api := pythonSegmentRangeAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.Path != pythonSegmentRangeBase {
				return 404, `{}`
			}
			return 200, `{"network_segment_ranges":[{"id":"r1","name":"vlan","shared":true},{"id":"r2","name":"other"}]}`
		})
		v, err := api.Find(context.Background(), "vlan", networksegmentranges.WithFindQuery("shared", true))
		if err != nil || v.Text("id") != "r1" || len(calls) != 2 {
			t.Fatal(v, err, calls)
		}
		if calls[0].path != pythonSegmentRangeBase+"/vlan" || calls[0].query != "shared=True" || calls[1].path != pythonSegmentRangeBase || calls[1].query != "name=vlan&shared=True" {
			t.Fatal(calls)
		}
	})
	t.Run("missing", func(t *testing.T) {
		var calls []pythonSegmentRangeCall
		api := pythonSegmentRangeAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.Path != pythonSegmentRangeBase {
				return 404, `{}`
			}
			return 200, `{"network_segment_ranges":[]}`
		})
		if v, err := api.Find(context.Background(), "none"); v != nil || err != nil {
			t.Fatal(v, err)
		}
		_, err := api.Find(context.Background(), "none", networksegmentranges.WithFindIgnoreMissing(false))
		if !errors.Is(err, resource.ErrNotFound) {
			t.Fatal(err)
		}
		pythonSegmentRangeOperation(t, err, "Find")
	})
	t.Run("ambiguous", func(t *testing.T) {
		var calls []pythonSegmentRangeCall
		api := pythonSegmentRangeAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.Path != pythonSegmentRangeBase {
				return 400, `{}`
			}
			return 200, `{"network_segment_ranges":[{"id":"r1","name":"dup"},{"id":"r2","name":"dup"}]}`
		})
		if _, err := api.Find(context.Background(), "dup"); !errors.Is(err, resource.ErrAmbiguous) {
			t.Fatal(err)
		}
	})
	t.Run("server error does not fall back", func(t *testing.T) {
		var calls []pythonSegmentRangeCall
		api := pythonSegmentRangeAPI(t, &calls, func(*http.Request) (int, string) { return 500, `{}` })
		_, err := api.Find(context.Background(), "vlan")
		pythonSegmentRangeStatus(t, err, 500)
		if len(calls) != 1 {
			t.Fatal(calls)
		}
	})
}
