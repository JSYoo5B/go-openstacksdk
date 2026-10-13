package ports_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func sourceCloudExpression(expression string) ports.SourceCloudListOption {
	raw, _ := json.Marshal(map[string]any{"jmespath_filters": expression})
	return ports.WithSourceCloudListFilters(raw)
}

func TestSourceCloudPortListLiteralIterablesDoNotFabricateResources(t *testing.T) {
	for _, test := range []struct{ expression, want string }{
		{"`[1,null,true,{\"id\":\"literal\"}]`", `[1,null,true,{"id":"literal"}]`},
		{"`{\"z\":1,\"a\":2}`", `["a","z"]`}, {"'한😀'", `["한","😀"]`}, {"`[]`", "[]"}, {"`{}`", "[]"}, {"''", "[]"},
	} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("literal performed HTTP"); return nil })
		result, err := api.ListSourceValues(context.Background(), sourceCloudExpression(test.expression))
		if err != nil || result == nil || string(result.JSON) != test.want || result.Ports != nil || len(result.Inventory) != 0 || len(calls) != 0 {
			t.Fatal(test, result, err, calls)
		}
		for _, value := range result.Values {
			if _, fake := value.(*ports.PortRecord); fake {
				t.Fatal("literal became record", value)
			}
		}
	}
}

func TestSourceCloudPortListNonIterablesAndGeneratorErrorsDoNoHTTP(t *testing.T) {
	for _, expression := range []string{"[*].id", "length(@)", "`null`", "`true`", "`42`", "["} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("noniterable performed HTTP"); return nil })
		result, err := api.ListSourceValues(context.Background(), sourceCloudExpression(expression))
		if err == nil || len(calls) != 0 || result != nil && result.JSON != nil {
			t.Fatal(expression, result, err, calls)
		}
		pythonPortListOperation(t, err, "ListSourceValues")
	}
}

func TestSourceCloudPortListReturnedGeneratorUsesActualRowsAndNoSeed(t *testing.T) {
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.URL.RawQuery != "network_id=n" || r.Header.Get("X-Trace") != "source" {
			t.Fatal(r.URL, r.Header)
		}
		return nativePortWire(203, `{"ports":[{"id":"p","project_id":"other","availability_zone":"zone"}]}`)
	})
	cloud, name := "fixture", "project"
	loc := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &name}}
	result, err := api.ListSourceValues(context.Background(), ports.WithSourceCloudListFilters(json.RawMessage(`{"jmespath_filters":"@","network_id":"n","headers":{"X-Trace":"source"}}`)), ports.WithSourceCloudListLocation(loc))
	if err != nil || result == nil || len(result.Ports) != 1 || len(result.Inventory) != 1 || len(result.Values) != 1 || result.Values[0] != result.Ports[0] || result.Ports[0] != result.Inventory[0] || len(calls) != 1 {
		t.Fatal(result, err, calls)
	}
	if result.Ports[0].Wire == nil || result.Ports[0].StatusCode != 203 {
		t.Fatal(result)
	}
	var location resource.CloudLocation
	if err := json.Unmarshal(result.Ports[0].Resource.Body["location"], &location); err != nil {
		t.Fatal(err)
	}
	if string(location.Project.ID) != `"other"` || location.Project.Name != nil || string(location.Zone) != `"zone"` {
		t.Fatal(location)
	}
	if _, fake := result.Ports[0].Resource.Body["network_id"]; fake && string(result.Ports[0].Resource.Body["network_id"]) != "null" {
		t.Fatal("query seed fabricated", result.Ports[0])
	}
}

func TestSourceCloudPortListNestedGeneratorRemainsOpaqueAndLazy(t *testing.T) {
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { return nativePortWire(200, `{"ports":[{"id":"p"}]}`) })
	result, err := api.ListSourceValues(context.Background(), sourceCloudExpression("to_array(@)"))
	if err != nil || result == nil || result.JSON != nil || result.Ports != nil || len(result.Inventory) != 0 || len(result.Values) != 1 || len(calls) != 0 {
		t.Fatal(result, err, calls)
	}
	generator, ok := result.Values[0].(*ports.RecordListGenerator)
	if !ok {
		t.Fatal(result.Values)
	}
	if _, err := json.Marshal(generator); err == nil {
		t.Fatal("opaque generator serialized")
	}
	n := 0
	for record, err := range generator.Records {
		if err != nil || record == nil || record.Wire == nil {
			t.Fatal(record, err)
		}
		n++
	}
	if n != 1 || len(calls) != 1 {
		t.Fatal(n, calls)
	}
}

func TestSourceCloudPortListFiltersFalseyAndTruthyNonMappings(t *testing.T) {
	for _, raw := range []json.RawMessage{nil, json.RawMessage("null"), json.RawMessage("false"), json.RawMessage("0"), json.RawMessage("1e-400"), json.RawMessage("[]"), json.RawMessage(`""`), json.RawMessage("{}")} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(r *http.Request) *http.Response {
			if r.URL.RawQuery != "" {
				t.Fatal(r.URL)
			}
			return nativePortWire(200, `{"ports":[]}`)
		})
		result, err := api.ListSourceValues(context.Background(), ports.WithSourceCloudListFilters(raw))
		if err != nil || result == nil || result.Ports == nil || string(result.JSON) != "[]" || len(calls) != 1 {
			t.Fatal(string(raw), result, err, calls)
		}
	}
	for _, raw := range []json.RawMessage{json.RawMessage("true"), json.RawMessage("1"), json.RawMessage("[1]"), json.RawMessage(`"x"`)} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("nonmapping filters performed HTTP"); return nil })
		result, err := api.ListSourceValues(context.Background(), ports.WithSourceCloudListFilters(raw))
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(string(raw), result, err, calls)
		}
	}
}

func TestSourceCloudPortListHeadersInputAndCallbackSnapshots(t *testing.T) {
	filters := json.RawMessage(`{"headers":{"X-Trace":"raw"},"fields":["id"]}`)
	option := ports.WithSourceCloudListFilters(filters)
	filters[2] = 'x'
	var calls []nativePortCall
	var escaped *ports.SourceCloudListOpts
	api := pythonCorePortAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.Header.Get("X-Trace") != "typed" || r.Header.Get("X-Callback") != "yes" || !reflect.DeepEqual(r.URL.Query()["fields"], []string{"id"}) {
			t.Fatal(r.Header, r.URL)
		}
		escaped.Headers["X-Trace"] = "mutated"
		return nativePortWire(200, `{"ports":[]}`)
	})
	result, err := api.ListSourceValues(context.Background(), option, ports.WithSourceCloudListHeaders(map[string]string{"X-Trace": "typed"}), func(c *ports.SourceCloudListOpts) error { c.Headers["X-Callback"] = "yes"; escaped = c; return nil })
	if err != nil || result == nil || len(calls) != 1 {
		t.Fatal(result, err, calls)
	}
}

func TestSourceCloudPortListLateFaultRetainsOnlyActualInventory(t *testing.T) {
	var calls []nativePortCall
	cause := errors.New("late page read")
	api := pythonCorePortAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.URL.Query().Get("marker") == "next" {
			res := nativePortWire(200, `{"ports":[{"id":"bad"}]}`)
			res.Body = &pythonFindFaultBody{data: `{"ports":[{"id":"bad"}]}`, read: cause}
			return res
		}
		return nativePortWire(200, `{"ports":[{"id":"p"}],"ports_links":[{"rel":"next","href":"?marker=next"}]}`)
	})
	result, err := api.ListSourceValues(context.Background())
	if result == nil || !errors.Is(err, cause) || len(result.Inventory) != 1 || result.JSON != nil || result.Ports != nil || result.Values != nil || len(calls) != 2 {
		t.Fatal(result, err, calls)
	}
}
