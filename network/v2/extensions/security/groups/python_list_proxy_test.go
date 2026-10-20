package groups_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestPythonSecurityGroupSourceProxyOpaqueJMESAndLegacyTypedBoundary(t *testing.T) {
	for _, tc := range []struct {
		expression, json string
		failure          bool
	}{{"`[1,null]`", "[1,null]", false}, {"`{\"z\":1,\"a\":2}`", `{"a":2,"z":1}`, false}, {"to_array(@)", "", false}, {"[*].id", "null", false}, {"length(@)", "", true}} {
		var calls []nativeSecGroupCall
		api, _ := findAPI(t, &calls, func(*http.Request) *http.Response { t.Error("unexpected HTTP"); return listRecordWire(200, `{}`) })
		result, err := api.ProxyListRecords(context.Background(), groups.WithRecordProxyQuery("jmespath_filters", tc.expression))
		if tc.failure != (err != nil) || len(calls) != 0 || !tc.failure && (result == nil || string(result.JSON) != tc.json) {
			t.Fatal(tc, result, err, calls)
		}
	}
	var calls []nativeSecGroupCall
	api, _ := findAPI(t, &calls, func(*http.Request) *http.Response { return listRecordWire(200, `{"security_groups":[]}`) })
	_, err := api.AllRecords(context.Background(), groups.WithRecordListQuery("jmespath_filters", "@"))
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}
func TestPythonSecurityGroupSourceProxyFormalsBasePathAndSnapshot(t *testing.T) {
	var calls []nativeSecGroupCall
	api, _ := findAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.URL.Path != "/neutron/v2.0/alternate/path" || r.Header.Get("X-Trace") != "owned" || r.Header.Get("OpenStack-API-Version") != "network 2.77" || r.URL.Query().Get("limit") != "1" {
			t.Error(r.URL, r.Header)
		}
		return listRecordWire(203, `{"security_groups":[{"id":"actual"}],"next":"?marker=ignored"}`)
	})
	raw := json.RawMessage(`{"X-Trace":"owned"}`)
	option := groups.WithRecordProxyQuery("headers", raw)
	raw[2] = 'X'
	result, err := api.ProxyListRecords(context.Background(), groups.WithRecordProxyQuery("base_path", "alternate/%(fragment)s"), groups.WithRecordProxyQuery("fragment", "path"), option, groups.WithRecordProxyQuery("microversion", "2.77"), groups.WithRecordProxyQuery("paginated", false), groups.WithRecordProxyQuery("max_items", true))
	if err != nil {
		t.Fatal(err)
	}
	generator, ok := result.Value.(*groups.RecordListGenerator)
	if !ok {
		t.Fatal(result)
	}
	n := 0
	for row, err := range generator.Records {
		if err != nil || row == nil || row.StatusCode != 203 {
			t.Fatal(row, err)
		}
		n++
	}
	if n != 1 || len(calls) != 1 {
		t.Fatal(n, calls)
	}
}
func TestPythonSecurityGroupSourceProxyDuplicateControlsAndLazyContext(t *testing.T) {
	for _, raw := range []json.RawMessage{json.RawMessage(`{"paginated":false}`), json.RawMessage(`{"base_path":null}`), json.RawMessage(`{"session":null}`)} {
		var calls []nativeSecGroupCall
		api, _ := findAPI(t, &calls, func(*http.Request) *http.Response { return listRecordWire(200, `{}`) })
		result, err := api.ProxyListRecords(context.Background(), groups.WithRecordProxyQuery("__conflicting_attrs", raw))
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(result, err, calls)
		}
	}
	var calls []nativeSecGroupCall
	api, _ := findAPI(t, &calls, func(*http.Request) *http.Response { return listRecordWire(200, `{"security_groups":[]}`) })
	ctx, cancel := context.WithCancelCause(context.Background())
	result, err := api.ProxyListRecords(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("cancel generator")
	cancel(cause)
	n := 0
	for row, err := range result.Value.(*groups.RecordListGenerator).Records {
		n++
		if row != nil || !errors.Is(err, cause) {
			t.Fatal(row, err)
		}
	}
	if n != 1 || len(calls) != 0 {
		t.Fatal(n, calls)
	}
}
