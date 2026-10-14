package subnets_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func subnetListResponse(code int, body string) *http.Response {
	response := nativeSubnetWire(code, body)
	response.Header.Set("X-Request-Id", "subnet-list-actual")
	return response
}
func subnetListProof(t *testing.T, err error, code int, body string) {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "subnet-list-actual" {
		t.Fatal(err, proof)
	}
}
func TestPythonSubnetListsOwnedSparseDefaultsAndOrderedAliases(t *testing.T) {
	var calls []nativeSubnetCall
	api := pythonCoreSubnetAPI(t, &calls, func(*http.Request) *http.Response { return subnetListResponse(199, `{"subnets":[{}]}`) })
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 1 || len(calls) != 1 || rows[0].StatusCode != 199 {
		t.Fatal(rows, err, calls)
	}
	for _, key := range []string{"id", "revision_number", "allocation_pools", "cidr", "created_at", "description", "dns_nameservers", "dns_publish_fixed_ip", "gateway_ip", "host_routes", "ip_version", "ipv6_address_mode", "ipv6_ra_mode", "is_dhcp_enabled", "name", "network_id", "prefix_length", "project_id", "tenant_id", "segment_id", "service_types", "subnet_pool_id", "updated_at", "use_default_subnet_pool", "if_match", "location"} {
		pythonFindField(t, rows[0], key, "null")
	}
	pythonFindField(t, rows[0], "tags", "[]")
	for _, tc := range []struct{ row, dhcp, prefix string }{{`{"enable_dhcp":"False","is_dhcp_enabled":false,"prefixlen":1,"prefix_length":2}`, "false", "2"}, {`{"is_dhcp_enabled":false,"enable_dhcp":"False","prefix_length":2,"prefixlen":1}`, "true", "1"}} {
		api = pythonCoreSubnetAPI(t, &calls, func(*http.Request) *http.Response { return subnetListResponse(203, `{"subnets":[`+tc.row+`]}`) })
		rows, err = api.AllRecords(context.Background())
		if err != nil || len(rows) != 1 {
			t.Fatal(rows, err)
		}
		pythonFindField(t, rows[0], "is_dhcp_enabled", tc.dhcp)
		pythonFindField(t, rows[0], "prefix_length", tc.prefix)
	}
}
func TestPythonSubnetListsDescriptorsRowHeadersAndIndependentActualEvidence(t *testing.T) {
	var calls []nativeSubnetCall
	body := `{"subnets":[{"id":"one","revision_number":"١٢","ip_version":4.9,"tags":"tag","allocation_pools":{"n":9007199254740993},"dns_nameservers":false,"enable_dhcp":"False","use_default_subnetpool":1e-400,"IF-MATCH":"row","self":"discard","location":{"forged":true},"x":9007199254740993}]}`
	api := pythonCoreSubnetAPI(t, &calls, func(*http.Request) *http.Response {
		response := subnetListResponse(203, body)
		response.Header.Set("If-Match", "page")
		return response
	})
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	v := rows[0]
	for k, w := range map[string]string{"revision_number": "12", "ip_version": "4", "tags": `["tag"]`, "allocation_pools": `[{"n":9007199254740993}]`, "dns_nameservers": "[false]", "is_dhcp_enabled": "true", "use_default_subnet_pool": "false", "if_match": `["row"]`, "location": "null", "x": "9007199254740993"} {
		pythonFindField(t, v, k, w)
	}
	if v.Resource.Body["self"] != nil || v.Header.Get("If-Match") != "page" || string(v.Envelope) != body {
		t.Fatal(v)
	}
	v.Resource.Body["x"][0] = 'X'
	v.Header.Set("X-Request-Id", "consumer")
	v.Envelope[0] = 'X'
	if string(v.Wire.Body["x"]) != "9007199254740993" || v.Wire.Header.Get("X-Request-Id") != "subnet-list-actual" || v.Resource.Header.Get("X-Request-Id") != "subnet-list-actual" {
		t.Fatal(v)
	}
}
func TestPythonSubnetListsPushdownLocalLoadedFiltersAndUnknownMarshalDiscard(t *testing.T) {
	var calls []nativeSubnetCall
	api := pythonCoreSubnetAPI(t, &calls, func(*http.Request) *http.Response {
		return subnetListResponse(200, `{"subnets":[{"id":"no","prefixlen":2},{"id":"yes","prefixlen":true,"created_at":9007199254740993}]}`)
	})
	rows, err := api.AllRecords(context.Background(), subnets.WithRecordListQuery("prefix_length", 1), subnets.WithRecordListQuery("created_at", json.RawMessage("9007199254740992.0")), subnets.WithRecordListQuery("is_dhcp_enabled", false), subnets.WithRecordListQuery("unknown_extension", make(chan int)), subnets.WithRecordListPaginated(false))
	if err != nil || len(rows) != 0 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	} // large int != rounded float, while true equals 1.
	rows, err = api.AllRecords(context.Background(), subnets.WithRecordListQuery("prefix_length", 1), subnets.WithRecordListQuery("is_dhcp_enabled", false), subnets.WithRecordListQuery("subnet_pool_id", "pool"), subnets.WithRecordListQuery("unknown_extension", "discard"), subnets.WithRecordListPaginated(false))
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err, calls)
	}
	q, _ := url.ParseQuery(calls[len(calls)-1].query)
	if q.Get("enable_dhcp") != "False" || q.Get("subnetpool_id") != "pool" || q.Has("prefix_length") || q.Has("tenant_id") {
		t.Fatal(q)
	}
}
func TestPythonSubnetListsProxyGeneratorJMESAndLazyResourceControls(t *testing.T) {
	var calls []nativeSubnetCall
	api := pythonCoreSubnetAPI(t, &calls, func(*http.Request) *http.Response { return subnetListResponse(200, `{"subnets":[{"id":"one"}]}`) })
	for _, raw := range []json.RawMessage{json.RawMessage(`"bad"`), json.RawMessage(`[1]`)} {
		result, err := api.ProxyListRecords(context.Background(), subnets.WithRecordListQuery("headers", raw), subnets.WithRecordListJMESPath("`[]`"))
		if err != nil || string(result.JSON) != "[]" || len(calls) != 0 {
			t.Fatal(result, err, calls)
		}
		result, err = api.ProxyListRecords(context.Background(), subnets.WithRecordListQuery("headers", raw))
		if err != nil {
			t.Fatal(err)
		}
		generator := result.Value.(*subnets.RecordListGenerator)
		for row, err := range generator.Records {
			if row != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(row, err)
			}
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	}
	for _, expr := range []string{"[0]", "[*].id", "[?id == 'one']", "type(@)"} {
		result, err := api.ProxyListRecords(context.Background(), subnets.WithRecordListJMESPath(expr))
		if err != nil || string(result.JSON) != "null" || len(calls) != 0 {
			t.Fatal(expr, result, err, calls)
		}
	}
	result, err := api.ProxyListRecords(context.Background(), subnets.WithRecordListJMESPath("to_array(@)"))
	if err != nil || result.JSON != nil {
		t.Fatal(result, err)
	}
	nested := result.Value.([]any)[0].(*subnets.RecordListGenerator)
	count := 0
	for row, err := range nested.Records {
		if err != nil || row == nil {
			t.Fatal(row, err)
		}
		count++
	}
	if count != 1 || len(calls) != 1 {
		t.Fatal(count, calls)
	}
	before := len(calls)
	if _, err := api.ProxyListRecords(context.Background(), subnets.WithRecordListJMESPath("length(@)")); err == nil || len(calls) != before {
		t.Fatal(err, calls)
	}
}
func TestPythonSubnetListsEmptyPathOnlyNullDefaultAndDelayedCancellation(t *testing.T) {
	for _, raw := range []string{`""`, `null`, `false`, `0`, `[]`, `{}`} {
		var calls []nativeSubnetCall
		api := pythonCoreSubnetAPI(t, &calls, func(*http.Request) *http.Response { return subnetListResponse(200, `{"subnets":[]}`) })
		result, err := api.ProxyListRecords(context.Background(), subnets.WithRecordListQuery("base_path", json.RawMessage(raw)))
		if err != nil {
			t.Fatal(raw, err)
		}
		var terminal error
		for _, err := range result.Value.(*subnets.RecordListGenerator).Records {
			terminal = err
		}
		if raw == `null` {
			if terminal != nil || len(calls) != 1 {
				t.Fatal(raw, terminal, calls)
			}
		} else if raw == `""` {
			if terminal != nil || len(calls) != 0 {
				t.Fatal(raw, terminal, calls)
			}
		} else if !errors.Is(terminal, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(raw, terminal, calls)
		}
		result, err = api.ProxyListRecords(context.Background(), subnets.WithRecordListQuery("base_path", json.RawMessage(raw)), subnets.WithRecordListJMESPath("`[]`"))
		if err != nil || string(result.JSON) != "[]" {
			t.Fatal(raw, result, err)
		}
	}
	var calls []nativeSubnetCall
	api := pythonCoreSubnetAPI(t, &calls, func(*http.Request) *http.Response { return subnetListResponse(200, `{"subnets":[]}`) })
	ctx, cancel := context.WithCancel(context.Background())
	result, err := api.ProxyListRecords(ctx, subnets.WithRecordListBasePath(""))
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	for row, err := range result.Value.(*subnets.RecordListGenerator).Records {
		if row != nil || !errors.Is(err, context.Canceled) {
			t.Fatal(row, err)
		}
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}
func TestPythonSubnetListsOneLevelRestorationBindingAndImmutableOptions(t *testing.T) {
	for _, key := range []string{"self", "resource_type", "cls", "session", "paginated", "base_path"} {
		var calls []nativeSubnetCall
		api := pythonCoreSubnetAPI(t, &calls, func(*http.Request) *http.Response { return subnetListResponse(200, `{"subnets":[]}`) })
		raw := json.RawMessage(fmt.Sprintf(`{%q:null}`, key))
		result, err := api.ProxyListRecords(context.Background(), subnets.WithRecordListConflictingAttrs(raw), subnets.WithRecordListJMESPath("`[]`"))
		if key == "self" || key == "resource_type" {
			if err != nil || string(result.JSON) != "[]" {
				t.Fatal(key, result, err)
			}
		} else if !errors.Is(err, resource.ErrInvalidOption) || result != nil {
			t.Fatal(key, result, err)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	}
	var calls []nativeSubnetCall
	api := pythonCoreSubnetAPI(t, &calls, func(*http.Request) *http.Response { return subnetListResponse(200, `{"subnets":[]}`) })
	raw := json.RawMessage(`{"name":"restored","__conflicting_attrs":{"name":"never"},"jmespath_filters":"length(@)"}`)
	opt := subnets.WithRecordListConflictingAttrs(raw)
	raw[2] = 'X'
	_, err := api.AllRecords(context.Background(), subnets.WithRecordListQuery("name", "first"), opt)
	if err != nil || len(calls) != 1 {
		t.Fatal(err, calls)
	}
	q, _ := url.ParseQuery(calls[0].query)
	if q.Get("name") != "restored" {
		t.Fatal(q)
	}
	query := map[string]json.RawMessage{"name": json.RawMessage(`"owned"`)}
	headers := map[string]string{"X-Trace": "owned"}
	option := subnets.WithRecordListOpts(subnets.RecordListOpts{Query: query, Headers: headers})
	query["name"] = json.RawMessage(`"changed"`)
	headers["X-Trace"] = "changed"
	_, err = api.AllRecords(context.Background(), option)
	if err != nil {
		t.Fatal(err)
	}
	q, _ = url.ParseQuery(calls[1].query)
	if q.Get("name") != "owned" {
		t.Fatal(q)
	}
}
func TestPythonSubnetListsMicroversionJSONDomainAndDefaultDisable(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{{`null`, "network 2.3"}, {`false`, ""}, {`0`, ""}, {`""`, ""}, {`[]`, ""}, {`{}`, ""}, {`true`, "network 1.0"}, {`1`, "network 1.0"}, {`[2,4]`, "network 2.4"}, {`"v2.4"`, "network 2.4"}, {`"latest"`, "network latest"}} {
		var calls []nativeSubnetCall
		api := pythonCoreSubnetAPI(t, &calls, func(r *http.Request) *http.Response {
			if r.Header.Get("OpenStack-API-Version") != tc.want {
				t.Fatal(tc, r.Header)
			}
			return subnetListResponse(200, `{"subnets":[]}`)
		})
		api.RawClient().Microversion = "2.3"
		rows, err := api.AllRecords(context.Background(), subnets.WithRecordListQuery("microversion", json.RawMessage(tc.raw)))
		if err != nil || len(rows) != 0 || len(calls) != 1 || api.RawClient().Microversion != "2.3" {
			t.Fatal(tc, rows, err, calls)
		}
	}
	for _, raw := range []string{`{"x":1}`, `"2.latest"`} {
		var calls []nativeSubnetCall
		api := pythonCoreSubnetAPI(t, &calls, func(*http.Request) *http.Response { return subnetListResponse(200, `{"subnets":[]}`) })
		result, err := api.ProxyListRecords(context.Background(), subnets.WithRecordListQuery("microversion", json.RawMessage(raw)))
		if err != nil {
			t.Fatal(err)
		}
		for _, err := range result.Value.(*subnets.RecordListGenerator).Records {
			if err == nil {
				t.Fatal(raw)
			}
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	}
}
func TestPythonSubnetListsSourceLinkPrecedenceAndRawMarkerMaxItems(t *testing.T) {
	var calls []nativeSubnetCall
	api := pythonCoreSubnetAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.URL.Query().Get("marker") != "" {
			return subnetListResponse(200, `{"subnets":[]}`)
		}
		return subnetListResponse(200, `{"subnets":[{"id":"filtered","prefixlen":2},{"id":"yes","prefixlen":1}],"links":{"next":"?marker=ignored"},"subnets_links":[{"rel":"next","href":"?marker=wrong"}]}`)
	})
	rows, err := api.AllRecords(context.Background(), subnets.WithRecordListQuery("prefix_length", 1), subnets.WithRecordListLimit(2))
	if err != nil || len(rows) != 1 || len(calls) != 2 {
		t.Fatal(rows, err, calls)
	}
	q, _ := url.ParseQuery(calls[1].query)
	if q.Get("marker") != "yes" {
		t.Fatal(q)
	}
	calls = nil
	api = pythonCoreSubnetAPI(t, &calls, func(*http.Request) *http.Response {
		return subnetListResponse(200, `{"subnets":[{"id":"one"},{"id":"two"}],"next":"?marker=late"}`)
	})
	rows, err = api.AllRecords(context.Background(), subnets.WithRecordListMaxItems(1))
	if err != nil || len(rows) != 1 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	q, _ = url.ParseQuery(calls[0].query)
	if q.Get("limit") != "1" {
		t.Fatal(q)
	}
}
func TestPythonSubnetListsMalformedAcceptedPagesKeepPhysicalProof(t *testing.T) {
	for _, tc := range []struct {
		code int
		body string
	}{{204, ""}, {200, "bad"}, {200, `{}`}, {200, `{"subnets":null}`}, {200, `{"subnets":false}`}, {203, `{"subnets":[{"ip_version":"²"}]}`}, {200, `{"subnets":[{"connection":null}]}`}} {
		var calls []nativeSubnetCall
		api := pythonCoreSubnetAPI(t, &calls, func(*http.Request) *http.Response { return subnetListResponse(tc.code, tc.body) })
		rows, err := api.AllRecords(context.Background())
		if rows != nil || err == nil || len(calls) != 1 {
			t.Fatal(tc, rows, err, calls)
		}
		subnetListProof(t, err, tc.code, tc.body)
	}
	var calls []nativeSubnetCall
	api := pythonCoreSubnetAPI(t, &calls, func(*http.Request) *http.Response { return subnetListResponse(203, `{"subnets":{"id":"one"}}`) })
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
}
func TestPythonSubnetListsLatePhysicalFaultsNative401AndNoFaultReplay(t *testing.T) {
	for _, status := range []int{200, 401, 429, 498} {
		var calls []nativeSubnetCall
		readCause, closeCause := errors.New("list read"), errors.New("list close")
		body := `{"subnets":[{"id":"late"}]}`
		if status >= 400 {
			body = `{"error":"physical late"}`
		}
		api := pythonCoreSubnetAPI(t, &calls, func(*http.Request) *http.Response {
			if len(calls) == 1 {
				return subnetListResponse(200, `{"subnets":[{"id":"first"}],"next":"?marker=late"}`)
			}
			response := subnetListResponse(status, body)
			response.Body = &pythonFindFaultBody{data: body, read: readCause, close: closeCause}
			return response
		})
		retries, backoff, auth := 0, 0, 0
		api.RawClient().ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			retries++
			return nil
		}
		api.RawClient().ProviderClient.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
			backoff++
			return nil
		}
		api.RawClient().ProviderClient.ReauthFunc = func(context.Context) error { auth++; return nil }
		seen := 0
		var terminal error
		for row, err := range api.ListRecords(context.Background()) {
			if err != nil {
				terminal = err
				break
			}
			if row != nil {
				seen++
			}
		}
		if seen != 1 || !errors.Is(terminal, readCause) || !errors.Is(terminal, closeCause) || len(calls) != 2 || retries != 0 || backoff != 0 || auth != 0 {
			t.Fatal(status, seen, terminal, calls, retries, backoff, auth)
		}
		if status >= 400 {
			original := terminal
			if status == 401 {
				// Gophercloud's native401 carrier does not unwrap either field.
				var noauth *gophercloud.ErrUnableToReauthenticate
				if !errors.As(terminal, &noauth) || noauth == nil || !errors.Is(noauth.ErrReauth, readCause) || !errors.Is(noauth.ErrReauth, closeCause) {
					t.Fatal(terminal, noauth)
				}
				original = noauth.ErrOriginal
				if _, ok := original.(gophercloud.ErrUnexpectedResponseCode); !ok {
					t.Fatalf("native401 original must keep the actual value carrier: %T %v", original, original)
				}
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(original, &native) || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "subnet-list-actual" || native.Method != http.MethodGet {
				t.Fatal(terminal, native)
			}
			actualURL, parseErr := url.Parse(native.URL)
			if parseErr != nil || actualURL.Path != "/neutron/v2.0/subnets" || actualURL.Query().Get("marker") != "late" {
				t.Fatal(terminal, native, parseErr)
			}
		} else {
			subnetListProof(t, terminal, status, body)
		}
	}
}
func TestPythonSubnetListsCleanReauthenticationAndCapturedHeaders(t *testing.T) {
	var calls []nativeSubnetCall
	var tokens, trace []string
	api := pythonCoreSubnetAPI(t, &calls, func(r *http.Request) *http.Response {
		tokens = append(tokens, r.Header.Get("X-Auth-Token"))
		trace = append(trace, r.Header.Get("X-Trace"))
		if len(calls) == 1 {
			return subnetListResponse(401, `{"error":"expired"}`)
		}
		return subnetListResponse(200, `{"subnets":[]}`)
	})
	client := api.RawClient()
	client.MoreHeaders = map[string]string{"X-Trace": "owned"}
	client.ProviderClient.ReauthFunc = func(context.Context) error {
		client.ProviderClient.SetToken("renewed")
		client.MoreHeaders["X-Trace"] = "caller-change"
		return nil
	}
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 0 || len(calls) != 2 || tokens[0] == tokens[1] || !reflect.DeepEqual(trace, []string{"owned", "owned"}) {
		t.Fatal(rows, err, calls, tokens, trace)
	}
}
