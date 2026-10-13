package networks_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestPythonNetworkProxyListRecordsJSONResultsDoNotReadGenerator(t *testing.T) {
	for _, tc := range []struct{ expression, expected string }{
		{"[?status==`ACTIVE`]", "null"}, {"[*].id", "null"}, {"[0]", "null"}, {"type(@)", "null"}, {"`true`", "true"}, {"`17`", "17"}, {"`{\"answer\":42}`", `{"answer":42}`}, {"`[1,false,null]`", `[1,false,null]`}, {"length(to_array(@))", "1"}, {"@ == @", "true"}, {"contains(to_array(@), @)", "true"}, {"[@,@] == [@,@]", "true"},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			var calls []nativeNetworkCall
			api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("expression consumed generator"); return nil })
			result, err := api.ProxyListRecords(context.Background(), networks.WithRecordListJMESPath(tc.expression))
			if err != nil || result == nil || string(result.JSON) != tc.expected || len(calls) != 0 {
				t.Fatal(result, err, calls)
			}
		})
	}
}

func TestPythonNetworkProxyListRecordsJMESRuntimeFailuresBeforeHTTP(t *testing.T) {
	for _, expression := range []string{"length(@)", "sort_by(@,&id)", "contains(@,`x`)", "to_string(@)", "[?", "unknown_function(@)"} {
		t.Run(expression, func(t *testing.T) {
			var calls []nativeNetworkCall
			api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("expression failure consumed generator"); return nil })
			result, err := api.ProxyListRecords(context.Background(), networks.WithRecordListJMESPath(expression))
			var operation *resource.OperationError
			if result != nil || err == nil || !errors.As(err, &operation) || operation.Operation != "ProxyListRecords" || operation.Resource != "networks" || len(calls) != 0 {
				t.Fatal(result, err, operation, calls)
			}
		})
	}
}

func TestPythonNetworkProxyListRecordsGeneratorIdentityAndNestedResultsStayLazy(t *testing.T) {
	for _, expression := range []string{"", "@", "not_null(@,`[]`)", "to_array(@)", "{records:@}"} {
		t.Run(expression, func(t *testing.T) {
			var calls []nativeNetworkCall
			api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Query().Get("name") != "name" {
					t.Fatal(req.URL)
				}
				return networkListResponse(200, `{"networks":[{"id":"one"}]}`)
			})
			result, err := api.ProxyListRecords(context.Background(), networks.WithRecordListJMESPath(expression), networks.WithRecordListQuery("name", "name"))
			if err != nil || result == nil || result.JSON != nil || len(calls) != 0 {
				t.Fatal(result, err, calls)
			}
			var generator *networks.RecordListGenerator
			switch value := result.Value.(type) {
			case *networks.RecordListGenerator:
				generator = value
			case []any:
				if len(value) != 1 {
					t.Fatal(value)
				}
				generator, _ = value[0].(*networks.RecordListGenerator)
			case map[string]any:
				generator, _ = value["records"].(*networks.RecordListGenerator)
			default:
				t.Fatal(value)
			}
			if generator == nil || generator.Records == nil || generator.JMESPathIdentity() != generator {
				t.Fatal(generator)
			}
			if _, err := json.Marshal(generator); err == nil || len(calls) != 0 {
				t.Fatal(err, calls)
			}
			ids := []string{}
			for value, err := range generator.Records {
				if err != nil {
					t.Fatal(err)
				}
				ids = append(ids, pythonNetworkListIDs(t, []*networks.NetworkRecord{value})[0])
			}
			if !reflect.DeepEqual(ids, []string{"one"}) || len(calls) != 1 {
				t.Fatal(ids, calls)
			}
		})
	}
}

func TestPythonNetworkListRecordsFalseyAndNonstringJMESAreIgnored(t *testing.T) {
	for index, value := range []any{nil, false, 0, 1, []any{}, []string{"truthy"}, map[string]any{"truthy": true}, "", json.RawMessage(`null`), json.RawMessage(`false`), json.RawMessage(`1`)} {
		t.Run(fmt.Sprintf("%d/%T", index, value), func(t *testing.T) {
			var calls []nativeNetworkCall
			var marshals atomic.Int32
			api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { return networkListResponse(200, `{"networks":[{"id":"one"}]}`) })
			result, err := api.ProxyListRecords(context.Background(), networks.WithRecordListQuery("jmespath_filters", value))
			if err != nil || result == nil || result.JSON != nil || len(calls) != 0 {
				t.Fatal(result, err, calls)
			}
			if _, ok := result.Value.(*networks.RecordListGenerator); !ok {
				t.Fatal(result.Value)
			}
			rows, err := api.AllRecords(context.Background(), networks.WithRecordListQuery("jmespath_filters", value), networks.WithRecordListQuery("jmespath_filters", networkListJSONCall{&marshals}))
			if err != nil || len(rows) != 1 || len(calls) != 1 || marshals.Load() != 0 {
				t.Fatal(rows, err, calls, marshals.Load())
			}
		})
	}
}
