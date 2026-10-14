package subnets_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonCloudSubnetCreateOwnedTranslatedBodySeedAndSourceStatuses(t *testing.T) {
	for _, status := range []int{199, 204, 302, 399} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var calls []nativeSubnetCall
			api := pythonCoreSubnetAPI(t, &calls, func(r *http.Request) *http.Response {
				if r.Method != http.MethodPost || r.URL.Path != "/neutron/v2.0/subnets" {
					t.Fatal(r.Method, r.URL)
				}
				body := `{"subnet":{"id":null,"is_dhcp_enabled":false,"enable_dhcp":"False","x_wire":2}}`
				if status == 204 {
					body = ""
				}
				return nativeSubnetWire(status, body)
			})
			attrs := map[string]json.RawMessage{"network_id": json.RawMessage(`"n"`), "ip_version": json.RawMessage("4"), "is_dhcp_enabled": json.RawMessage(`"False"`), "x_seed": json.RawMessage("9007199254740993")}
			value, err := api.CreateCloudRecord(context.Background(), attrs, func(c *subnets.CloudCreateOpts) error { attrs["network_id"] = json.RawMessage(`"mutated"`); return nil })
			if err != nil || value == nil || value.Resource == nil || value.StatusCode != status || len(calls) != 1 {
				t.Fatal(value, err, calls)
			}
			var posted map[string]map[string]json.RawMessage
			_ = json.Unmarshal([]byte(calls[0].body), &posted)
			if string(posted["subnet"]["network_id"]) != `"n"` || string(posted["subnet"]["enable_dhcp"]) != `"False"` || posted["subnet"]["is_dhcp_enabled"] != nil {
				t.Fatal(posted)
			}
			pythonFindField(t, value, "network_id", `"n"`)
			pythonFindField(t, value, "ip_version", "4")
			pythonFindField(t, value, "location", "null")
			if status == 204 {
				if value.Wire != nil {
					t.Fatal(value.Wire)
				}
				pythonFindField(t, value, "is_dhcp_enabled", "true")
			} else {
				pythonFindField(t, value, "id", "null")
				pythonFindField(t, value, "is_dhcp_enabled", "true")
				pythonFindWireField(t, value, "x_wire", "2")
			}
		})
	}
}
func TestPythonCloudSubnetCreateHeaderSeedAndActualResponsePrecedence(t *testing.T) {
	for _, withSeed := range []bool{false, true} {
		for _, withHeader := range []bool{false, true} {
			var calls []nativeSubnetCall
			api := pythonCoreSubnetAPI(t, &calls, func(*http.Request) *http.Response {
				response := nativeSubnetWire(201, `{"subnet":{}}`)
				if withHeader {
					response.Header.Set("If-Match", "actual-http")
				}
				return response
			})
			attrs := map[string]json.RawMessage{"network_id": json.RawMessage(`"n"`)}
			if withSeed {
				attrs["if_match"] = json.RawMessage(`["seed1","seed2"]`)
			}
			value, err := api.CreateCloudRecord(context.Background(), attrs)
			if err != nil || value == nil {
				t.Fatal(value, err)
			}
			expected := "null"
			if withSeed {
				expected = `["seed1","seed2"]`
			}
			if withHeader {
				expected = `["actual-http"]`
			}
			pythonFindField(t, value, "if_match", expected)
		}
	}
}
func TestPythonCloudSubnetCreateConstructorGetterFailuresAndNativeRejectedReceipts(t *testing.T) {
	var calls []nativeSubnetCall
	api := pythonCoreSubnetAPI(t, &calls, func(*http.Request) *http.Response { return nativeSubnetWire(201, `{"subnet":{}}`) })
	for _, attrs := range []map[string]json.RawMessage{{"cls": nil}, {"connection": nil}, {"_synchronized": nil}, {"resource_type": nil}, {"if_match": json.RawMessage(`[1]`)}, {"ip_version": json.RawMessage(`"²"`)}, {"__conflicting_attrs": json.RawMessage(`[1]`)}} {
		value, err := api.CreateCloudRecord(context.Background(), attrs)
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) && err == nil || len(calls) != 0 {
			t.Fatal(attrs, value, err, calls)
		}
	}
	for _, status := range []int{400, 403, 404, 409, 503} {
		calls = nil
		api = pythonCoreSubnetAPI(t, &calls, func(*http.Request) *http.Response { return nativeSubnetWire(status, `{"rejected":true}`) })
		value, err := api.CreateCloudRecord(context.Background(), map[string]json.RawMessage{})
		var native gophercloud.ErrUnexpectedResponseCode
		if value == nil || value.Resource != nil || value.Wire != nil || value.StatusCode != status || string(value.Envelope) != `{"rejected":true}` || !errors.As(err, &native) || native.Actual != status || len(calls) != 1 {
			t.Fatal(value, err, native, calls)
		}
	}
}
func TestPythonCloudSubnetCreateOnlyWinningHeaderAliasIsValidated(t *testing.T) {
	for _, keys := range [][]string{{"if_match", "if-match"}, {"if_match", "IF-MATCH"}} {
		var calls []nativeSubnetCall
		api := pythonCoreSubnetAPI(t, &calls, func(r *http.Request) *http.Response {
			if r.Header.Get("If-Match") != "safe" {
				t.Fatal(r.Header)
			}
			return nativeSubnetWire(201, `{"subnet":{}}`)
		})
		attrs := map[string]json.RawMessage{keys[0]: json.RawMessage(`[1]`), keys[1]: json.RawMessage(`"safe"`)}
		value, err := api.CreateCloudRecord(context.Background(), attrs, subnets.WithCloudCreateOrder(keys...))
		if err != nil || value == nil || len(calls) != 1 {
			t.Fatal(value, err, calls)
		}
		pythonFindField(t, value, "if_match", `["safe"]`)
		calls = nil
		value, err = api.CreateCloudRecord(context.Background(), attrs, subnets.WithCloudCreateOrder(keys[1], keys[0]))
		if value != nil || err == nil || len(calls) != 0 {
			t.Fatal(value, err, calls)
		}
	}
}
func TestPythonCloudSubnetCreateBasePathAndRestoredUnknownClassGetter(t *testing.T) {
	var calls []nativeSubnetCall
	api := pythonCoreSubnetAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.URL.Path != "/neutron/v2.0/alternate" || r.Header.Get("If-Match") != "None" {
			t.Fatal(r.URL, r.Header)
		}
		return nativeSubnetWire(201, `{"subnet":{}}`)
	})
	attrs := map[string]json.RawMessage{"base_path": json.RawMessage(`"/alternate"`), "if_match": nil, "__conflicting_attrs": json.RawMessage(`{"base_path":"/body-only"}`)}
	value, err := api.CreateCloudRecord(context.Background(), attrs)
	if err != nil || value == nil || len(calls) != 1 || !strings.Contains(calls[0].body, `"base_path":"/body-only"`) {
		t.Fatal(value, err, calls)
	}
	pythonFindField(t, value, "base_path", `"/subnets"`)
}
