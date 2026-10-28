package servergroups_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestSourceCloudGroupReadsListDescriptorsAliasesUnknownStatusAndVersion(t *testing.T) {
	for _, status := range []int{199, 200, 302, 399} {
		api, calls := pythonServerGroupAPI(t, func(r *http.Request) *http.Response {
			if r.URL.Path != pythonServerGroupBase || r.URL.RawQuery != "" || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.64" {
				t.Error(r.URL, r.Header)
			}
			return pythonServerGroupWire(status, `{"server_groups":[{"id":null,"members":["first"],"member_ids":false,"policies":"untyped","metadata":false,"rules":false,"unknown":9007199254740993}]}`)
		})
		rows := 0
		for row, e := range api.ListCloudRecords(context.Background()) {
			if e != nil || row == nil || row.Resource == nil || row.StatusCode != status {
				t.Fatal(row, e)
			}
			for key, want := range map[string]string{"id": "null", "name": "null", "member_ids": "false", "policies": `"untyped"`, "metadata": "false", "rules": "{}", "location": "null"} {
				if string(row.Resource.Body[key]) != want {
					t.Fatal(key, row.Resource.Body)
				}
			}
			if row.Resource.Body["unknown"] != nil || row.Resource.Body["members"] != nil || string(row.Wire.Body["unknown"]) != "9007199254740993" {
				t.Fatal(row)
			}
			rows++
		}
		if rows != 1 || calls.Len() != 1 {
			t.Fatal(rows, pythonServerGroupCalls(calls))
		}
	}
}
func TestSourceCloudGroupReadsFindEarlyDuplicateEmptyTrailingAndLateFailure(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		api, calls := pythonServerGroupAPI(t, func(r *http.Request) *http.Response {
			if r.URL.Path != pythonServerGroupBase {
				return pythonServerGroupWire(404, `{"error":"member absent"}`)
			}
			if r.URL.Query().Get("marker") == "next" {
				return pythonServerGroupWire(503, `{"error":"late"}`)
			}
			rows := `[{"id":null,"name":"same"}]`
			if duplicate {
				rows = `[{"id":null,"name":"same"},{"id":0,"name":"same"}]`
			}
			return pythonServerGroupWire(200, `{"server_groups":`+rows+`,"server_groups_links":[{"rel":"next","href":"?marker=next"}]}`)
		})
		result, e := api.FindCloudRecord(context.Background(), "same")
		if result == nil || result.Group != nil || result.Observed.StatusCode != 404 {
			t.Fatal(result, e)
		}
		if duplicate {
			if !errors.Is(e, resource.ErrAmbiguous) || calls.Len() != 2 || len(result.Inventory) != 2 {
				t.Fatal(result, e, pythonServerGroupCalls(calls))
			}
		} else if !gophercloud.ResponseCodeIs(e, 503) || calls.Len() != 3 || len(result.Inventory) != 1 {
			t.Fatal(result, e, pythonServerGroupCalls(calls))
		}
	}
	api, calls := pythonServerGroupAPI(t, func(r *http.Request) *http.Response {
		if r.URL.Path == pythonServerGroupBase+"/" {
			return pythonServerGroupWire(200, `{"server_group":{"name":"empty-direct"}}`)
		}
		t.Error("not empty trailing member", r.URL)
		return pythonServerGroupWire(500, `{}`)
	})
	result, e := api.FindCloudRecord(context.Background(), "")
	if e != nil || result == nil || result.Group == nil || string(result.Group.Resource.Body["id"]) != `""` || calls.Len() != 1 || calls.Calls()[0].Path != pythonServerGroupBase+"/" {
		t.Fatal(result, e, pythonServerGroupCalls(calls))
	}
}
func TestSourceCloudGroupReadsDiscoveryCeilingSelectedSessionAndLiteralSegments(t *testing.T) {
	api, calls := pythonServerGroupAPI(t, func(*http.Request) *http.Response { return pythonServerGroupWire(200, `{}`) })
	client := api.RawClient()
	client.Endpoint = client.ResourceBase
	client.Microversion = ""
	versions := []string{}
	client.HTTPClient.Transport = testcloud.LinkRequest(func(r *http.Request) (*http.Response, error) {
		versions = append(versions, r.Header.Get("X-OpenStack-Nova-API-Version"))
		if r.URL.Path == "/nova/v2.1" || r.URL.Path == "/nova/v2.1/" {
			return pythonServerGroupWire(200, `{"version":{"id":"v2.1","status":"CURRENT","min_version":"2.1","version":"2.100","links":[{"rel":"self","href":"`+client.ResourceBase+`"}]}}`), nil
		}
		if r.URL.EscapedPath() != pythonServerGroupBase+"/one%2Ftwo%20%252F" {
			t.Error(r.URL)
		}
		return pythonServerGroupWire(200, `{"server_group":{"id":"actual"}}`), nil
	})
	result, e := api.FindCloudRecord(context.Background(), "one/two %2F")
	if e != nil || result == nil || result.Group == nil || !reflect.DeepEqual(versions, []string{"", "2.64"}) || client.Microversion != "" {
		t.Fatal(result, e, versions)
	}
	client.Microversion = "2.100"
	versions = nil
	if _, e := api.FindCloudRecord(context.Background(), "one/two %2F"); e != nil || !reflect.DeepEqual(versions, []string{"2.100"}) {
		t.Fatal(e, versions)
	}
	for _, identity := range []string{"..", ".", " ", "bad\nvalue"} {
		before := len(versions)
		if r, e := api.FindCloudRecord(context.Background(), identity); r != nil || !errors.Is(e, resource.ErrInvalidOption) || len(versions) != before || calls.Len() != 0 {
			t.Fatal(r, e, versions, pythonServerGroupCalls(calls))
		}
	}
}
func TestSourceCloudGroupReadsExpandedRejected404AndPhysicalFailureDoNotFallback(t *testing.T) {
	api, calls := pythonServerGroupAPI(t, func(*http.Request) *http.Response { return pythonServerGroupWire(404, `{"error":"actual"}`) })
	callbacks := 0
	api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
		callbacks++
		o.OkCodes = append(o.OkCodes, 404)
		return nil
	}
	result, e := api.FindCloudRecord(context.Background(), "g1")
	var native gophercloud.ErrUnexpectedResponseCode
	if result == nil || result.Group != nil || result.Observed == nil || !errors.As(e, &native) || native.Actual != 404 || calls.Len() != 2 || callbacks != 1 {
		t.Fatal(result, e, native, pythonServerGroupCalls(calls), callbacks)
	}
}

func TestSourceCloudGroupReadsEmptyExactFallbackNeverMatchesNullAsEmptyString(t *testing.T) {
	for _, raw := range []string{`{"id":null,"name":"not-empty"}`, `{"id":"not-empty","name":null}`, `{"id":"actual-empty","name":""}`} {
		api, calls := pythonServerGroupAPI(t, func(r *http.Request) *http.Response {
			if r.URL.Path == pythonServerGroupBase+"/" {
				return pythonServerGroupWire(404, `{"error":"absent"}`)
			}
			return pythonServerGroupWire(200, `{"server_groups":[`+raw+`]}`)
		})
		result, e := api.FindCloudRecord(context.Background(), "")
		match := raw == `{"id":"actual-empty","name":""}`
		if e != nil || result == nil || (result.Group != nil) != match || calls.Len() != 2 || len(result.Inventory) != 1 {
			t.Fatal(raw, result, e, pythonServerGroupCalls(calls))
		}
	}
}
