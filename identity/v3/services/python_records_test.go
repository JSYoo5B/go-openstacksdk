package services_test

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
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/services"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func serviceRecordAPI(t *testing.T, reply func(*http.Request) *http.Response) (*services.API, *testcloud.Cloud) {
	t.Helper()
	fixture := testcloud.New(t)
	fixture.Provider.HTTPClient.Transport = pythonServiceTransport(func(r *http.Request) (*http.Response, error) { return reply(r), nil })
	client := fixture.Client("identity", "/catalog/")
	client.ResourceBase = fixture.Server.URL + "/keystone/v3/"
	return services.New(client), fixture
}
func serviceRecordWire(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"service-proof"}}, Body: io.NopCloser(strings.NewReader(body))}
}
func serviceRecordIDs(t *testing.T, rows []*services.ServiceRecord) []string {
	t.Helper()
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		var id string
		if err := json.Unmarshal(r.Resource.Body["id"], &id); err != nil {
			t.Fatal(r, err)
		}
		out = append(out, id)
	}
	return out
}
func TestPythonServiceRecordsDefaultProjectionPresenceAndIndependentPhysicalEvidence(t *testing.T) {
	body := `{"services":[{"id":"a","name":"nova","type":42,"description":{"owner":"infra"},"enabled":"false","links":{"self":"/v3/services/a"},"x_extension":{"n":9007199254740993},"location":{"cloud":"forged"},"project_id":"foreign"},{"id":null}]}`
	calls := 0
	api, _ := serviceRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.Method != http.MethodGet || r.URL.Path != pythonServiceBase || r.URL.RawQuery != "" || r.Body != nil || r.Header.Get("Accept") != "application/json" {
			t.Fatal(r.Method, r.URL, r.Header, r.Body)
		}
		return serviceRecordWire(199, body)
	})
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 2 || calls != 1 {
		t.Fatal(rows, err, calls)
	}
	for key, want := range map[string]string{"type": "42", "description": `{"owner":"infra"}`, "is_enabled": "true", "location": "null"} {
		if string(rows[0].Resource.Body[key]) != want {
			t.Fatal(key, rows[0])
		}
	}
	for _, key := range []string{"id", "name", "description", "type", "is_enabled", "links", "location"} {
		if string(rows[1].Resource.Body[key]) != "null" {
			t.Fatal(key, rows[1])
		}
	}
	for _, key := range []string{"enabled", "x_extension", "project_id"} {
		if _, ok := rows[0].Resource.Body[key]; ok {
			t.Fatal(key, rows[0])
		}
	}
	if string(rows[0].Wire.Body["x_extension"]) != `{"n":9007199254740993}` || rows[0].StatusCode != 199 || string(rows[0].Envelope) != body || rows[0].Header.Get("X-Request-Id") != "service-proof" {
		t.Fatal(rows[0])
	}
	clone := rows[0].Clone()
	clone.Resource.Body["name"][1] = 'X'
	clone.Wire.Body["x_extension"][0] = '['
	clone.Header.Set("X-Request-Id", "changed")
	clone.Envelope[0] = '['
	if string(rows[0].Resource.Body["name"]) != `"nova"` || string(rows[0].Wire.Body["x_extension"]) != `{"n":9007199254740993}` || rows[0].Header.Get("X-Request-Id") != "service-proof" || string(rows[0].Envelope) != body || rows[1].Header.Get("X-Request-Id") != "service-proof" {
		t.Fatal(rows)
	}
}
func TestPythonServiceRecordsDeclaredQueryAndOrderedLoadedLocalFilters(t *testing.T) {
	var query url.Values
	api, _ := serviceRecordAPI(t, func(r *http.Request) *http.Response {
		query = r.URL.Query()
		return serviceRecordWire(200, `{"services":[{"id":"a","description":0.10000000000000001,"enabled":true},{"id":"b","description":0.2,"enabled":false}]}`)
	})
	rows, err := api.AllRecords(context.Background(), services.WithRecordListQuery("type", []string{"compute", "image"}), services.WithRecordListQuery("name", false), services.WithRecordListQuery("description", json.RawMessage(`0.1`)), services.WithRecordListQuery("is_enabled", true), services.WithRecordListQuery("enabled", func() {}), services.WithRecordListQuery("unknown", func() {}))
	if err != nil || !reflect.DeepEqual(serviceRecordIDs(t, rows), []string{"a"}) || !reflect.DeepEqual(query, url.Values{"type": {"compute", "image"}, "name": {"False"}}) {
		t.Fatal(rows, err, query)
	}
	for _, raw := range []string{`"truthy"`, `[1]`, `1`, `true`, `{"k":1}`} {
		api, _ := serviceRecordAPI(t, func(*http.Request) *http.Response {
			return serviceRecordWire(200, `{"services":[{"id":"a","links":`+raw+`}]}`)
		})
		rows, err := api.AllRecords(context.Background(), services.WithRecordListQuery("links", map[string]any{}))
		if err != nil || len(rows) != 1 {
			t.Fatal(raw, rows, err)
		}
	}
}
func TestPythonServiceRecordsRawMarkerShortPagesCapBreakAndReusableIterator(t *testing.T) {
	calls := 0
	var markers []string
	api, _ := serviceRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		markers = append(markers, r.URL.Query().Get("marker"))
		if r.URL.Query().Get("limit") != "2" || r.URL.Query().Get("type") != "d" {
			t.Fatal(r.URL)
		}
		switch r.URL.Query().Get("marker") {
		case "":
			return serviceRecordWire(200, `{"services":[{"id":"a","description":"skip"},{"id":"b","description":"keep"}]}`)
		case "b":
			return serviceRecordWire(200, `{"services":[{"id":"c","description":"keep"}]}`)
		default:
			return serviceRecordWire(200, `{"services":[]}`)
		}
	})
	opts := []services.RecordListOption{services.WithRecordListLimit(2), services.WithRecordListQuery("type", "d"), services.WithRecordListQuery("description", "keep")}
	rows, err := api.AllRecords(context.Background(), opts...)
	if err != nil || !reflect.DeepEqual(serviceRecordIDs(t, rows), []string{"b", "c"}) || !reflect.DeepEqual(markers, []string{"", "b", "c"}) {
		t.Fatal(rows, err, markers)
	}
	calls = 0
	rows, err = api.AllRecords(context.Background(), append(opts, services.WithRecordListMaxItems(2))...)
	if err != nil || len(rows) != 1 || calls != 1 {
		t.Fatal(rows, err, calls)
	}
	seq := api.ListRecords(context.Background(), opts...)
	before := calls
	for i := 0; i < 2; i++ {
		for row, err := range seq {
			if err != nil || row == nil {
				t.Fatal(row, err)
			}
			break
		}
	}
	if calls-before != 2 {
		t.Fatal(calls, before)
	}
}
func TestPythonServiceRecordsSourceDictionaryRootPresenceAndActualContinuationGuards(t *testing.T) {
	for _, mode := range []string{"dictionary", "root masks plural", "foreign", "changed filter", "cycle", "null root"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			api, _ := serviceRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				if calls > 1 && mode != "cycle" {
					return serviceRecordWire(200, `{"services":[]}`)
				}
				suffix := `,"links":{"next":"?marker=a&type=d"}`
				if mode == "root masks plural" {
					suffix = `,"links":{},"services_links":[{"rel":"next","href":"https://foreign.invalid/v3/services"}]`
				}
				if mode == "foreign" {
					suffix = `,"next":"https://foreign.invalid/v3/services?marker=a&type=d"`
				}
				if mode == "changed filter" {
					suffix = `,"next":"?marker=a&type=changed"`
				}
				if mode == "cycle" {
					suffix = `,"next":"?marker=a&type=d"`
				}
				if mode == "null root" {
					suffix = `,"links":null`
				}
				return serviceRecordWire(200, `{"services":[{"id":"a"}]`+suffix+`}`)
			})
			rows, err := api.AllRecords(context.Background(), services.WithRecordListQuery("type", "d"))
			if mode == "dictionary" || mode == "root masks plural" {
				if err != nil || len(rows) != 1 || calls != 1 {
					t.Fatal(rows, err, calls)
				}
			} else {
				var proof *resource.ResponseError
				if rows != nil || err == nil || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-Id") != "service-proof" {
					t.Fatal(rows, err, proof)
				}
			}
		})
	}
}
func TestPythonServiceRecordsProxyGeneratorJMESPreservesUnreadAndNestedIdentity(t *testing.T) {
	calls := 0
	api, _ := serviceRecordAPI(t, func(*http.Request) *http.Response {
		calls++
		return serviceRecordWire(200, `{"services":[{"id":"a"}]}`)
	})
	for expression, want := range map[string]string{"[*].id": "null", "type(@)": "null", "`[1,false,null]`": "[1,false,null]", "@ == @": "true"} {
		result, err := api.ProxyListRecords(context.Background(), services.WithRecordListJMESPath(expression))
		if err != nil || result == nil || string(result.JSON) != want || calls != 0 {
			t.Fatal(expression, result, err, calls)
		}
	}
	result, err := api.ProxyListRecords(context.Background(), services.WithRecordListJMESPath("@"))
	generator, ok := result.Value.(*services.RecordListGenerator)
	if err != nil || !ok || result.JSON != nil || calls != 0 {
		t.Fatal(result, err, calls)
	}
	for row, err := range generator.Records {
		if err != nil || row == nil {
			t.Fatal(row, err)
		}
	}
	if calls != 1 {
		t.Fatal(calls)
	}
	result, err = api.ProxyListRecords(context.Background(), services.WithRecordListJMESPath("to_array(@)"))
	if err != nil || result.JSON != nil || calls != 1 {
		t.Fatal(result, err, calls)
	}
	if _, err = api.ProxyListRecords(context.Background(), services.WithRecordListJMESPath("length(@)")); !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
		t.Fatal(err, calls)
	}
}
func TestPythonServiceRecordsOptionSnapshotsFormalsAndPreflightSourceGuards(t *testing.T) {
	calls := 0
	var query url.Values
	api, _ := serviceRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		query = r.URL.Query()
		if r.Header.Get("X-Custom") != "owned" {
			t.Fatal(r.Header)
		}
		return serviceRecordWire(200, `{"services":[]}`)
	})
	headers := map[string]string{"X-Custom": "owned"}
	q := map[string]json.RawMessage{"type": json.RawMessage(`"owned"`), "unknown": json.RawMessage(`not valid`)}
	opt := services.WithRecordListOpts(services.RecordListOpts{Headers: headers, Query: q})
	headers["X-Custom"] = "changed"
	q["type"][1] = 'X'
	_, err := api.AllRecords(context.Background(), opt)
	if err != nil || calls != 1 || query.Get("type") != "owned" {
		t.Fatal(err, calls, query)
	}
	before := calls
	for _, options := range [][]services.RecordListOption{{nil}, {services.WithRecordListQuery("self", func() {})}, {services.WithRecordListQuery("__conflicting_attrs", map[string]any{"paginated": false})}, {services.WithRecordListBasePath("https://foreign.invalid/")}, {services.WithRecordListMaxItems(-1)}} {
		rows, err := api.AllRecords(context.Background(), options...)
		var proof *resource.ResponseError
		if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || calls != before {
			t.Fatal(rows, err, proof, calls)
		}
	}
	cause := errors.New("service callback canceled")
	ctx, cancel := context.WithCancelCause(context.Background())
	_, err = api.AllRecords(ctx, func(*services.RecordListOpts) error { cancel(cause); return nil })
	if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || calls != before {
		t.Fatal(err, calls)
	}
	old := api.RawClient().Endpoint
	_, err = api.AllRecords(context.Background(), func(*services.RecordListOpts) error {
		api.RawClient().Endpoint = "https://changed.invalid/"
		return nil
	})
	api.RawClient().Endpoint = old
	if !errors.Is(err, resource.ErrInvalidOption) || calls != before {
		t.Fatal(err, calls)
	}
}
func TestPythonServiceRecordsExplicitLocationOwnsCurrentScopeWithoutUnknownProjectInference(t *testing.T) {
	cloudName, projectName := "configured", "current"
	location := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &projectName}}
	option := services.WithRecordListLocation(location)
	cloudName = "changed"
	projectName = "changed"
	api, _ := serviceRecordAPI(t, func(*http.Request) *http.Response {
		return serviceRecordWire(200, `{"services":[{"id":"a","project_id":"foreign","location":{"cloud":"wire"}}]}`)
	})
	rows, err := api.AllRecords(context.Background(), option)
	if err != nil || len(rows) != 1 || string(rows[0].Wire.Body["project_id"]) != `"foreign"` {
		t.Fatal(rows, err)
	}
	var actual map[string]json.RawMessage
	_ = json.Unmarshal(rows[0].Resource.Body["location"], &actual)
	var project map[string]json.RawMessage
	_ = json.Unmarshal(actual["project"], &project)
	if string(actual["cloud"]) != `"configured"` || string(project["id"]) != `"scope"` || string(project["name"]) != `"current"` {
		t.Fatal(actual, project)
	}
}
func TestPythonServiceRecordsMalformedRowsAndLocalFilterFaultRetainAcceptedPage(t *testing.T) {
	for _, body := range []string{`{"services":null}`, `{"services":[null]}`, `{"services":[{"id":"a","connection":true}]}`, `{"services":[{"id":"a","links":3}]}`, `{"services":[]} trailing`} {
		api, _ := serviceRecordAPI(t, func(*http.Request) *http.Response { return serviceRecordWire(203, body) })
		opts := []services.RecordListOption{}
		if strings.Contains(body, `"links":3`) {
			opts = append(opts, services.WithRecordListQuery("links", map[string]any{"key": "value"}))
		}
		rows, err := api.AllRecords(context.Background(), opts...)
		var proof *resource.ResponseError
		if rows != nil || err == nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "service-proof" {
			t.Fatal(body, rows, err, proof)
		}
	}
}

type serviceRecordFaultBody struct {
	data            string
	read, close     error
	onRead, onClose func()
	used            bool
	closed          int
}

func (b *serviceRecordFaultBody) Read(p []byte) (int, error) {
	if b.used {
		return 0, io.EOF
	}
	b.used = true
	n := copy(p, b.data)
	if b.onRead != nil {
		b.onRead()
	}
	if b.read != nil {
		return n, b.read
	}
	return n, io.EOF
}
func (b *serviceRecordFaultBody) Close() error {
	b.closed++
	if b.onClose != nil {
		b.onClose()
	}
	return b.close
}
func TestPythonServiceRecordsPhysicalFaultsKeepNative401AndActualReceiptWithoutCallbacks(t *testing.T) {
	for _, status := range []int{200, 400, 403, 404, 401, 429, 498, 503} {
		for _, mode := range []string{"read", "close", "both", "restore cancel"} {
			t.Run(fmt.Sprintf("%d/%s", status, mode), func(t *testing.T) {
				calls, retries, reauth, backoff := 0, 0, 0, 0
				readCause := errors.New("service physical read")
				closeCause := errors.New("service physical close")
				cancelCause := errors.New("service caller canceled")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				body := &serviceRecordFaultBody{data: `{"services":[{"id":"a"}]}`}
				if mode == "read" || mode == "both" {
					body.read = readCause
				}
				if mode == "close" || mode == "both" {
					body.close = closeCause
				}
				api, fixture := serviceRecordAPI(t, func(r *http.Request) *http.Response {
					calls++
					response := serviceRecordWire(status, body.data)
					response.Body = body
					return response
				})
				if mode == "restore cancel" {
					old := api.RawClient().Endpoint
					body.onRead = func() { api.RawClient().Endpoint = "https://changed.invalid/"; cancel(cancelCause) }
					body.onClose = func() { api.RawClient().Endpoint = old }
					body.close = closeCause
				}
				fixture.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				fixture.Provider.ReauthFunc = func(context.Context) error { reauth++; return nil }
				fixture.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
					backoff++
					return nil
				}
				rows, err := api.AllRecords(ctx)
				if rows != nil || err == nil || calls != 1 || body.closed != 1 || retries != 0 || reauth != 0 || backoff != 0 {
					t.Fatal(rows, err, calls, body.closed, retries, reauth, backoff)
				}
				if status < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body.data || proof.Header.Get("X-Request-Id") != "service-proof" {
						t.Fatal(err, proof)
					}
				} else {
					original := err
					if status == 401 {
						var failure *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &failure) || failure.ErrOriginal == nil || failure.ErrReauth == nil {
							t.Fatal(err, failure)
						}
						original = failure.ErrOriginal
					}
					var proof gophercloud.ErrUnexpectedResponseCode
					if !errors.As(original, &proof) || proof.Actual != status || string(proof.Body) != body.data || proof.ResponseHeader.Get("X-Request-Id") != "service-proof" || proof.Method != http.MethodGet || !strings.Contains(proof.URL, pythonServiceBase) {
						t.Fatal(err, proof)
					}
				}
				if body.read != nil && !errors.Is(err, readCause) {
					t.Fatal(err)
				}
				if body.close != nil && !errors.Is(err, closeCause) {
					t.Fatal(err)
				}
				if mode == "restore cancel" && (!errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption)) {
					t.Fatal(err)
				}
			})
		}
	}
}
func TestPythonServiceRecordsCleanReauthRetryAndBackoffKeepFrozenIdentityRequest(t *testing.T) {
	for _, first := range []int{401, 503, 429, 498} {
		t.Run(fmt.Sprint(first), func(t *testing.T) {
			calls, reauth, retries, backoff := 0, 0, 0, 0
			api, fixture := serviceRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				if r.Header.Get("OpenStack-API-Version") != "identity 3.1" || r.Header.Get("X-Selected") != "fixed" || r.Header.Get("If-Match") != "" || r.URL.Query().Get("type") != "d" {
					t.Fatal(r.URL, r.Header)
				}
				if calls == 1 {
					return serviceRecordWire(first, `{"error":"retry"}`)
				}
				if first == 401 && r.Header.Get("X-Auth-Token") != "renewed" {
					t.Fatal(r.Header)
				}
				return serviceRecordWire(200, `{"services":[]}`)
			})
			fixture.Provider.ReauthFunc = func(context.Context) error {
				reauth++
				fixture.Provider.SetToken("renewed")
				api.RawClient().MoreHeaders["If-Match"] = "injected"
				return nil
			}
			fixture.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				api.RawClient().MoreHeaders["If-Match"] = "injected"
				return nil
			}
			fixture.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				backoff++
				api.RawClient().MoreHeaders["If-Match"] = "injected"
				return nil
			}
			api.RawClient().MoreHeaders = map[string]string{"X-Selected": "fixed"}
			_, err := api.AllRecords(context.Background(), services.WithRecordListQuery("type", "d"), services.WithRecordListMicroversion("3.1"))
			if err != nil || calls != 2 {
				t.Fatal(err, calls, reauth, retries, backoff)
			}
			if first == 401 && reauth != 1 {
				t.Fatal(reauth)
			}
			if first == 503 && retries != 1 {
				t.Fatal(retries)
			}
			if (first == 429 || first == 498) && backoff != 1 {
				t.Fatal(backoff)
			}
		})
	}
}

func TestPythonServiceRecordsBoolAliasResponseOrderAndLocalCanonicalKey(t *testing.T) {
	api, _ := serviceRecordAPI(t, func(*http.Request) *http.Response {
		return serviceRecordWire(200, `{"services":[{"id":"a","enabled":true,"is_enabled":false},{"id":"b","is_enabled":false,"enabled":"false"},{"id":"c","enabled":0},{"id":"d","enabled":null}]}`)
	})
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 4 {
		t.Fatal(rows, err)
	}
	for i, want := range []string{"false", "true", "false", "null"} {
		if string(rows[i].Resource.Body["is_enabled"]) != want {
			t.Fatal(rows[i])
		}
	}
	rows, err = api.AllRecords(context.Background(), services.WithRecordListQuery("is_enabled", false))
	if err != nil || !reflect.DeepEqual(serviceRecordIDs(t, rows), []string{"a", "c"}) {
		t.Fatal(rows, err)
	}
}
func TestPythonServiceRecordsFormalRestoreHeaderVersionUnknownAndConstructorReceipt(t *testing.T) {
	calls := 0
	api, _ := serviceRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.URL.RawQuery != "name=new" || r.Header.Get("X-Trace") != "restored" || r.Header.Get("Accept") != "custom/json" || r.Header.Get("OpenStack-API-Version") != "identity 3.99" {
			t.Fatal(r.URL, r.Header)
		}
		return serviceRecordWire(200, `{"services":{"id":"one","links":{"keep":1}}}`)
	})
	raw := json.RawMessage(`{"name":"new","links":{"keep":1},"headers":{"X-Trace":"restored","Accept":"custom/json"},"microversion":"3.99","jmespath_filters":"length(@)","location":false,"__conflicting_attrs":{"name":"must-not-recurse"}}`)
	restore := services.WithRecordListQuery("__conflicting_attrs", raw)
	raw[1] = 'X'
	rows, err := api.AllRecords(context.Background(), services.WithRecordListQuery("name", "old"), restore)
	if err != nil || len(rows) != 1 || calls != 1 {
		t.Fatal(rows, err, calls)
	}
	for _, payload := range []string{`{}`, `{"services":null}`, `{"services":[null]}`, `{"services":[4]}`, `{"services":[{"id":"one","cls":null}]}`, `{"services":[{"id":"one","connection":null}]}`, `{"services":[{"id":"one","microversion":null}]}`, `{"services":[{"id":"one","_synchronized":false}]}`, `{"services":[{"id":"one"}],"next":true}`, ``} {
		api, _ := serviceRecordAPI(t, func(*http.Request) *http.Response {
			status := 203
			if payload == "" {
				status = 204
			}
			return serviceRecordWire(status, payload)
		})
		rows, err := api.AllRecords(context.Background())
		var proof *resource.ResponseError
		if rows != nil || !errors.As(err, &proof) || string(proof.Body) != payload || proof.Header.Get("X-Request-Id") != "service-proof" {
			t.Fatal(payload, rows, err, proof)
		}
	}
}
func TestPythonServiceResolveRecordClosedFinderMemberAndCleanStatusFallback(t *testing.T) {
	for _, status := range []int{199, 203, 304, 399, 400, 403, 404, 429, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls := []string{}
			api, _ := serviceRecordAPI(t, func(r *http.Request) *http.Response {
				calls = append(calls, r.URL.RequestURI())
				if r.URL.Path == pythonServiceBase+"/nova" {
					return serviceRecordWire(status, `{"service":{"id":"member","name":"nova","enabled":false}}`)
				}
				if r.URL.Query().Get("name") != "nova" {
					t.Fatal(r.URL)
				}
				return serviceRecordWire(200, `{"services":[{"id":"found","name":"nova"}]}`)
			})
			row, err := api.ResolveRecord(context.Background(), "nova")
			if status < 400 {
				if err != nil || row == nil || row.StatusCode != status || len(calls) != 1 || string(row.Resource.Body["id"]) != `"member"` {
					t.Fatal(row, err, calls)
				}
			} else if status == 400 || status == 403 || status == 404 {
				if err != nil || row == nil || len(calls) != 2 || string(row.Resource.Body["id"]) != `"found"` {
					t.Fatal(row, err, calls)
				}
			} else {
				var proof gophercloud.ErrUnexpectedResponseCode
				if row != nil || !errors.As(err, &proof) || proof.Actual != status || len(calls) != 1 {
					t.Fatal(row, err, calls, proof)
				}
			}
		})
	}
}
func TestPythonServiceResolveRecordMissingDuplicatesAndGetOnlyVersion(t *testing.T) {
	for _, mode := range []string{"missing", "duplicate", "late failure", "single"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			api, _ := serviceRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				if r.Header.Get("X-Select") != "owned" {
					t.Fatal(r.Header)
				}
				if r.URL.Path == pythonServiceBase+"/nova" {
					if r.Header.Get("OpenStack-API-Version") != "identity 3.8" || r.URL.RawQuery != "" {
						t.Fatal(r.URL, r.Header)
					}
					return serviceRecordWire(404, `{"error":"missing member"}`)
				}
				if r.Header.Get("OpenStack-API-Version") != "identity 3.1" || r.URL.Query().Get("name") != "nova" {
					t.Fatal(r.URL, r.Header)
				}
				if calls == 2 {
					if mode == "missing" {
						return serviceRecordWire(200, `{"services":[]}`)
					}
					return serviceRecordWire(203, `{"services":[{"id":"a","name":"nova"}],"next":"?name=nova&marker=a"}`)
				}
				if mode == "late failure" {
					return serviceRecordWire(503, `{"error":"late"}`)
				}
				if mode == "duplicate" {
					return serviceRecordWire(200, `{"services":[{"id":"a","name":"nova"}]}`)
				}
				return serviceRecordWire(200, `{"services":[]}`)
			})
			api.RawClient().Microversion = "3.1"
			row, err := api.ResolveRecord(context.Background(), "nova", services.WithRecordResolveHeaders(map[string]string{"X-Select": "owned"}), services.WithRecordResolveMicroversion("3.8"))
			switch mode {
			case "missing":
				if err != nil || row != nil || calls != 2 {
					t.Fatal(row, err, calls)
				}
			case "duplicate":
				if row != nil || !errors.Is(err, resource.ErrAmbiguous) || calls != 3 {
					t.Fatal(row, err, calls)
				}
			case "late failure":
				var proof gophercloud.ErrUnexpectedResponseCode
				if row != nil || !errors.As(err, &proof) || proof.Actual != 503 || calls != 3 {
					t.Fatal(row, err, calls)
				}
			case "single":
				if err != nil || row == nil || row.StatusCode != 203 || calls != 3 {
					t.Fatal(row, err, calls)
				}
			}
		})
	}
}
func TestPythonServiceResolveRecordPhysicalForbiddenDoesNotFallThrough(t *testing.T) {
	calls, retries := 0, 0
	cause := errors.New("member close")
	api, fixture := serviceRecordAPI(t, func(*http.Request) *http.Response {
		calls++
		r := serviceRecordWire(403, `{"error":"actual"}`)
		r.Body = &serviceRecordFaultBody{data: `{"error":"actual"}`, close: cause}
		return r
	})
	fixture.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		retries++
		return nil
	}
	row, err := api.ResolveRecord(context.Background(), "nova")
	var proof gophercloud.ErrUnexpectedResponseCode
	if row != nil || !errors.Is(err, cause) || !errors.As(err, &proof) || proof.Actual != 403 || calls != 1 || retries != 0 {
		t.Fatal(row, err, proof, calls, retries)
	}
}
func TestPythonServiceResolveRecordOptionsSnapshotCancellationAndExplicitLocation(t *testing.T) {
	name, zone := "configured", json.RawMessage(`"az"`)
	location := resource.CloudLocation{Cloud: &name, Zone: zone, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`)}}
	headers := map[string]string{"X-Owned": "owned"}
	option := services.WithRecordResolveOpts(services.RecordResolveOpts{Location: &location, Headers: headers})
	name = "changed"
	zone[1] = 'X'
	headers["X-Owned"] = "changed"
	calls := 0
	api, _ := serviceRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.Header.Get("X-Owned") != "owned" {
			t.Fatal(r.Header)
		}
		return serviceRecordWire(200, `{"service":{"id":"a","project_id":"foreign","location":{"cloud":"wire"}}}`)
	})
	row, err := api.ResolveRecord(context.Background(), "a", option)
	if err != nil || row == nil || !strings.Contains(string(row.Resource.Body["location"]), `"cloud":"configured"`) || !strings.Contains(string(row.Resource.Body["location"]), `"zone":"az"`) || string(row.Wire.Body["project_id"]) != `"foreign"` {
		t.Fatal(row, err)
	}
	for _, mode := range []string{"nil", "source", "cancel", "location"} {
		t.Run(mode, func(t *testing.T) {
			before := calls
			seen := 0
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("resolve cancellation")
			old := api.RawClient().Endpoint
			defer func() { api.RawClient().Endpoint = old }()
			var first services.RecordResolveOption = func(*services.RecordResolveOpts) error {
				if mode == "source" {
					api.RawClient().Endpoint = "https://changed.invalid/"
				} else {
					cancel(cause)
				}
				return nil
			}
			if mode == "nil" {
				first = nil
			}
			if mode == "location" {
				first = services.WithRecordResolveLocation(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`bad`)}})
			}
			row, err := api.ResolveRecord(ctx, "a", first, func(*services.RecordResolveOpts) error { seen++; return nil })
			var proof *resource.ResponseError
			if row != nil || err == nil || calls != before || errors.As(err, &proof) || (mode != "location" && seen != 0) {
				t.Fatal(row, err, calls, seen, proof)
			}
			if mode == "cancel" && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
				t.Fatal(err)
			}
		})
	}
}
