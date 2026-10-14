package networks_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestPythonNetworkCloudListSourceStatusDefaultsAndConnectedLocation(t *testing.T) {
	var calls []nativeNetworkCall
	body := `{"networks":{"id":"actual","project_id":"foreign","tenant_id":"ignored","availability_zone":false,"mtu":1500.9,"router:external":1,"x_new":9007199254740993,"location":{"spoofed":true}}}`
	api := pythonCoreNetworkAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.Method != http.MethodGet || r.URL.Path != "/neutron/v2.0/networks" {
			t.Error(r.Method, r.URL)
		}
		v := nativeNetworkWire(199, body)
		v.Header.Set("X-Request-Id", "actual-cloud-list")
		return v
	})
	name := "configured"
	region := "region"
	loc := resource.CloudLocation{Cloud: &name, RegionName: &region, Project: resource.CloudProject{ID: json.RawMessage(`"current"`), Name: &name}}
	proxy, err := api.ProxyCloudListRecords(context.Background(), &loc)
	if err != nil {
		t.Fatal(err)
	}
	generator := proxy.Value.(*networks.RecordListGenerator)
	rows := 0
	for row, err := range generator.Records {
		if err != nil || row == nil {
			t.Fatal(row, err)
		}
		rows++
		if string(row.Resource.Body["mtu"]) != "1500" || string(row.Resource.Body["is_router_external"]) != "true" || string(row.Resource.Body["x_new"]) != "9007199254740993" || row.StatusCode != 199 || string(row.Envelope) != body || row.Header.Get("X-Request-Id") != "actual-cloud-list" || string(row.Wire.Body["location"]) != `{"spoofed":true}` {
			t.Fatal(row)
		}
		var value resource.CloudLocation
		json.Unmarshal(row.Resource.Body["location"], &value)
		if string(value.Project.ID) != `"foreign"` || value.Project.Name != nil || string(value.Zone) != "false" {
			t.Fatal(value)
		}
	}
	if rows != 1 || len(calls) != 1 {
		t.Fatal(rows, calls)
	}
}
func TestPythonNetworkCloudListEmptyPathSkipsHTTPAndKeepsIterationGuards(t *testing.T) {
	for _, lane := range []string{"empty", "cancel", "source"} {
		var calls []nativeNetworkCall
		api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("empty path HTTP"); return nil })
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		proxy, err := api.ProxyCloudListRecords(ctx, nil, networks.WithRecordListBasePath(""))
		if err != nil {
			t.Fatal(err)
		}
		cause := errors.New("caller cancel")
		if lane == "cancel" {
			cancel(cause)
		} else if lane == "source" {
			api.RawClient().ResourceBase += "changed/"
		}
		yielded := 0
		for row, err := range proxy.Value.(*networks.RecordListGenerator).Records {
			yielded++
			if row != nil || err == nil {
				t.Fatal(row, err)
			}
			if lane == "cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
				t.Fatal(err)
			}
			if lane == "source" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		}
		want := 0
		if lane != "empty" {
			want = 1
		}
		if yielded != want || len(calls) != 0 {
			t.Fatal(yielded, calls)
		}
	}
}
func TestPythonNetworkCloudListOpaqueProxyStaysLazyWithOwnedLocation(t *testing.T) {
	var calls []nativeNetworkCall
	api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeNetworkWire(200, `{"networks":[{"id":"one","project_id":false,"tenant_id":"foreign"}]}`)
	})
	name := "before"
	loc := resource.CloudLocation{Cloud: &name, Project: resource.CloudProject{ID: json.RawMessage(`"current"`), Name: &name}}
	proxy, err := api.ProxyCloudListRecords(context.Background(), &loc, networks.WithRecordListJMESPath("to_array(@)"))
	if err != nil || len(calls) != 0 || proxy.JSON != nil {
		t.Fatal(proxy, err, calls)
	}
	name = "after"
	loc.Project.ID[1] = 'X'
	values := proxy.Value.([]any)
	generator := values[0].(*networks.RecordListGenerator)
	for row, err := range generator.Records {
		if err != nil {
			t.Fatal(err)
		}
		var value resource.CloudLocation
		json.Unmarshal(row.Resource.Body["location"], &value)
		if value.Cloud == nil || *value.Cloud != "before" || string(value.Project.ID) != `"current"` || value.Project.Name == nil || *value.Project.Name != "before" {
			t.Fatal(value)
		}
	}
	if len(calls) != 1 {
		t.Fatal(calls)
	}
}

func TestPythonNetworkCloudListRawFormalsRemainLazyAcrossProxyPhases(t *testing.T) {
	for _, key := range []string{"headers", "base_path", "microversion"} {
		var calls []nativeNetworkCall
		api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("invalid raw formal HTTP"); return nil })
		raw := json.RawMessage(`{"x":1}`)
		if key == "headers" {
			raw = json.RawMessage(`"bad"`)
		}
		opts := []networks.RecordListOption{networks.WithCloudRecordListQuery(key, raw), networks.WithCloudRecordListQuery("jmespath_filters", json.RawMessage("\"`[]`\""))}
		proxy, err := api.ProxyCloudListRecords(context.Background(), nil, opts...)
		if err != nil || proxy == nil || string(proxy.JSON) != "[]" || len(calls) != 0 {
			t.Fatal(key, proxy, err, calls)
		}
		proxy, err = api.ProxyCloudListRecords(context.Background(), nil, networks.WithCloudRecordListQuery(key, raw))
		if err != nil || proxy == nil || len(calls) != 0 {
			t.Fatal(key, proxy, err, calls)
		}
		yielded := 0
		for row, cause := range proxy.Value.(*networks.RecordListGenerator).Records {
			yielded++
			if row != nil || !errors.Is(cause, resource.ErrInvalidOption) {
				t.Fatal(row, cause)
			}
		}
		if yielded != 1 || len(calls) != 0 {
			t.Fatal(key, yielded, calls)
		}
	}
	for _, key := range []string{"base_path", "paginated", "cls", "session"} {
		var calls []nativeNetworkCall
		api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("binding error HTTP"); return nil })
		raw := json.RawMessage(`{"` + key + `":null}`)
		proxy, err := api.ProxyCloudListRecords(context.Background(), nil, networks.WithCloudRecordListQuery("__conflicting_attrs", raw), networks.WithCloudRecordListQuery("jmespath_filters", json.RawMessage("\"`[]`\"")))
		if proxy != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(key, proxy, err, calls)
		}
	}
}

func TestPythonNetworkCloudListOriginalAndRestoredBindingHaveDifferentPhases(t *testing.T) {
	for _, key := range []string{"self", "resource_type"} {
		for _, expression := range []string{"", "`[]`"} {
			var calls []nativeNetworkCall
			api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { return nativeNetworkWire(200, `{"networks":[{"id":"one"}]}`) })
			opts := []networks.RecordListOption{networks.WithCloudRecordListQuery(key, json.RawMessage(`null`))}
			if expression != "" {
				raw, _ := json.Marshal(expression)
				opts = append(opts, networks.WithCloudRecordListQuery("jmespath_filters", raw))
			}
			proxy, err := api.ProxyCloudListRecords(context.Background(), nil, opts...)
			if proxy != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
				t.Fatal(key, expression, proxy, err, calls)
			}
			raw := json.RawMessage(`{"` + key + `":null}`)
			opts[0] = networks.WithCloudRecordListQuery("__conflicting_attrs", raw)
			proxy, err = api.ProxyCloudListRecords(context.Background(), nil, opts...)
			if err != nil || proxy == nil || len(calls) != 0 {
				t.Fatal(key, expression, proxy, err, calls)
			}
			if expression != "" {
				if string(proxy.JSON) != "[]" {
					t.Fatal(proxy)
				}
				continue
			}
			rows := 0
			for row, cause := range proxy.Value.(*networks.RecordListGenerator).Records {
				rows++
				if row == nil || cause != nil || string(row.Resource.Body["id"]) != `"one"` {
					t.Fatal(row, cause)
				}
			}
			if rows != 1 || len(calls) != 1 {
				t.Fatal(key, rows, calls)
			}
		}
	}
}
