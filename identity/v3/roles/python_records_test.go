package roles_test

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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func roleRecordAPI(t *testing.T, reply func(*http.Request) *http.Response) (*roles.API, *testcloud.Cloud) {
	t.Helper()
	fixture := testcloud.New(t)
	fixture.Provider.HTTPClient.Transport = pythonRoleTransport(func(r *http.Request) (*http.Response, error) { return reply(r), nil })
	client := fixture.Client("identity", "/catalog/")
	client.ResourceBase = fixture.Server.URL + "/keystone/v3/"
	return roles.New(client), fixture
}
func roleRecordWire(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"role-proof"}}, Body: io.NopCloser(strings.NewReader(body))}
}
func roleRecordIDs(t *testing.T, rows []*roles.RoleRecord) []string {
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
func TestPythonRoleRecordsDefaultProjectionPresenceAndIndependentPhysicalEvidence(t *testing.T) {
	body := `{"roles":[{"id":"a","name":"reader","description":null,"domain_id":"d","options":3,"links":{"self":"/v3/roles/a"},"x_extension":{"n":9007199254740993},"location":{"cloud":"forged"},"project_id":"foreign"},{"id":null}]}`
	calls := 0
	api, _ := roleRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.Method != http.MethodGet || r.URL.Path != pythonRoleBase || r.URL.RawQuery != "" || r.Body != nil || r.Header.Get("Accept") != "application/json" {
			t.Fatal(r.Method, r.URL, r.Header, r.Body)
		}
		return roleRecordWire(199, body)
	})
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 2 || calls != 1 {
		t.Fatal(rows, err, calls)
	}
	if string(rows[0].Resource.Body["options"]) != "{}" || string(rows[0].Resource.Body["location"]) != "null" || string(rows[0].Resource.Body["description"]) != "null" || string(rows[1].Resource.Body["options"]) != "null" || string(rows[1].Resource.Body["id"]) != "null" {
		t.Fatal(rows)
	}
	if _, ok := rows[0].Resource.Body["x_extension"]; ok {
		t.Fatal(rows[0].Resource.Body)
	}
	if _, ok := rows[0].Resource.Body["project_id"]; ok {
		t.Fatal(rows[0].Resource.Body)
	}
	if string(rows[0].Wire.Body["x_extension"]) != `{"n":9007199254740993}` || rows[0].StatusCode != 199 || string(rows[0].Envelope) != body || rows[0].Header.Get("X-Request-Id") != "role-proof" {
		t.Fatal(rows[0])
	}
	clone := rows[0].Clone()
	clone.Resource.Body["name"][1] = 'X'
	clone.Wire.Body["x_extension"][0] = '['
	clone.Header.Set("X-Request-Id", "changed")
	clone.Envelope[0] = '['
	if string(rows[0].Resource.Body["name"]) != `"reader"` || string(rows[0].Wire.Body["x_extension"]) != `{"n":9007199254740993}` || rows[0].Header.Get("X-Request-Id") != "role-proof" || string(rows[0].Envelope) != body || rows[1].Header.Get("X-Request-Id") != "role-proof" {
		t.Fatal(rows)
	}
}
func TestPythonRoleRecordsDeclaredQueryAndOrderedLoadedLocalFilters(t *testing.T) {
	var query url.Values
	api, _ := roleRecordAPI(t, func(r *http.Request) *http.Response {
		query = r.URL.Query()
		return roleRecordWire(200, `{"roles":[{"id":"a","name":"server name","description":0.10000000000000001,"options":{"flag":1.0}},{"id":"b","description":0.2,"options":{"flag":false}}]}`)
	})
	rows, err := api.AllRecords(context.Background(), roles.WithRecordListQuery("domain_id", []string{"d1", "d2"}), roles.WithRecordListQuery("name", false), roles.WithRecordListQuery("description", json.RawMessage(`0.1`)), roles.WithRecordListQuery("options", map[string]any{"flag": true}), roles.WithRecordListQuery("unknown", func() {}))
	if err != nil || !reflect.DeepEqual(roleRecordIDs(t, rows), []string{"a"}) || !reflect.DeepEqual(query, url.Values{"domain_id": {"d1", "d2"}, "name": {"False"}}) {
		t.Fatal(rows, err, query)
	}
	for _, raw := range []string{`"truthy"`, `[1]`, `1`, `true`, `{"k":1}`} {
		api, _ := roleRecordAPI(t, func(*http.Request) *http.Response {
			return roleRecordWire(200, `{"roles":[{"id":"a","links":`+raw+`}]}`)
		})
		rows, err := api.AllRecords(context.Background(), roles.WithRecordListQuery("links", map[string]any{}))
		if err != nil || len(rows) != 1 {
			t.Fatal(raw, rows, err)
		}
	}
}
func TestPythonRoleRecordsRawMarkerShortPagesCapBreakAndReusableIterator(t *testing.T) {
	calls := 0
	var markers []string
	api, _ := roleRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		markers = append(markers, r.URL.Query().Get("marker"))
		if r.URL.Query().Get("limit") != "2" || r.URL.Query().Get("domain_id") != "d" {
			t.Fatal(r.URL)
		}
		switch r.URL.Query().Get("marker") {
		case "":
			return roleRecordWire(200, `{"roles":[{"id":"a","description":"skip"},{"id":"b","description":"keep"}]}`)
		case "b":
			return roleRecordWire(200, `{"roles":[{"id":"c","description":"keep"}]}`)
		default:
			return roleRecordWire(200, `{"roles":[]}`)
		}
	})
	opts := []roles.RecordListOption{roles.WithRecordListLimit(2), roles.WithRecordListQuery("domain_id", "d"), roles.WithRecordListQuery("description", "keep")}
	rows, err := api.AllRecords(context.Background(), opts...)
	if err != nil || !reflect.DeepEqual(roleRecordIDs(t, rows), []string{"b", "c"}) || !reflect.DeepEqual(markers, []string{"", "b", "c"}) {
		t.Fatal(rows, err, markers)
	}
	calls = 0
	rows, err = api.AllRecords(context.Background(), append(opts, roles.WithRecordListMaxItems(2))...)
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
func TestPythonRoleRecordsSourceDictionaryRootPresenceAndActualContinuationGuards(t *testing.T) {
	for _, mode := range []string{"dictionary", "root masks plural", "foreign", "changed filter", "cycle", "null root"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			api, _ := roleRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				if calls > 1 && mode != "cycle" {
					return roleRecordWire(200, `{"roles":[]}`)
				}
				suffix := `,"links":{"next":"?marker=a&domain_id=d"}`
				if mode == "root masks plural" {
					suffix = `,"links":{},"roles_links":[{"rel":"next","href":"https://foreign.invalid/v3/roles"}]`
				}
				if mode == "foreign" {
					suffix = `,"next":"https://foreign.invalid/v3/roles?marker=a&domain_id=d"`
				}
				if mode == "changed filter" {
					suffix = `,"next":"?marker=a&domain_id=changed"`
				}
				if mode == "cycle" {
					suffix = `,"next":"?marker=a&domain_id=d"`
				}
				if mode == "null root" {
					suffix = `,"links":null`
				}
				return roleRecordWire(200, `{"roles":[{"id":"a"}]`+suffix+`}`)
			})
			rows, err := api.AllRecords(context.Background(), roles.WithRecordListQuery("domain_id", "d"))
			if mode == "dictionary" || mode == "root masks plural" {
				if err != nil || len(rows) != 1 || calls != 1 {
					t.Fatal(rows, err, calls)
				}
			} else {
				var proof *resource.ResponseError
				if rows != nil || err == nil || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-Id") != "role-proof" {
					t.Fatal(rows, err, proof)
				}
			}
		})
	}
}
func TestPythonRoleRecordsProxyGeneratorJMESPreservesUnreadAndNestedIdentity(t *testing.T) {
	calls := 0
	api, _ := roleRecordAPI(t, func(*http.Request) *http.Response { calls++; return roleRecordWire(200, `{"roles":[{"id":"a"}]}`) })
	for expression, want := range map[string]string{"[*].id": "null", "type(@)": "null", "`[1,false,null]`": "[1,false,null]", "@ == @": "true"} {
		result, err := api.ProxyListRecords(context.Background(), roles.WithRecordListJMESPath(expression))
		if err != nil || result == nil || string(result.JSON) != want || calls != 0 {
			t.Fatal(expression, result, err, calls)
		}
	}
	result, err := api.ProxyListRecords(context.Background(), roles.WithRecordListJMESPath("@"))
	generator, ok := result.Value.(*roles.RecordListGenerator)
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
	result, err = api.ProxyListRecords(context.Background(), roles.WithRecordListJMESPath("to_array(@)"))
	if err != nil || result.JSON != nil || calls != 1 {
		t.Fatal(result, err, calls)
	}
	if _, err = api.ProxyListRecords(context.Background(), roles.WithRecordListJMESPath("length(@)")); !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
		t.Fatal(err, calls)
	}
}
func TestPythonRoleRecordsOptionSnapshotsFormalsAndPreflightSourceGuards(t *testing.T) {
	calls := 0
	var query url.Values
	api, _ := roleRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		query = r.URL.Query()
		if r.Header.Get("X-Custom") != "owned" {
			t.Fatal(r.Header)
		}
		return roleRecordWire(200, `{"roles":[]}`)
	})
	headers := map[string]string{"X-Custom": "owned"}
	q := map[string]json.RawMessage{"domain_id": json.RawMessage(`"owned"`), "unknown": json.RawMessage(`not valid`)}
	opt := roles.WithRecordListOpts(roles.RecordListOpts{Headers: headers, Query: q})
	headers["X-Custom"] = "changed"
	q["domain_id"][1] = 'X'
	_, err := api.AllRecords(context.Background(), opt)
	if err != nil || calls != 1 || query.Get("domain_id") != "owned" {
		t.Fatal(err, calls, query)
	}
	before := calls
	for _, options := range [][]roles.RecordListOption{{nil}, {roles.WithRecordListQuery("self", func() {})}, {roles.WithRecordListQuery("__conflicting_attrs", map[string]any{"paginated": false})}, {roles.WithRecordListBasePath("https://foreign.invalid/")}, {roles.WithRecordListMaxItems(-1)}} {
		rows, err := api.AllRecords(context.Background(), options...)
		var proof *resource.ResponseError
		if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || calls != before {
			t.Fatal(rows, err, proof, calls)
		}
	}
	cause := errors.New("role callback canceled")
	ctx, cancel := context.WithCancelCause(context.Background())
	_, err = api.AllRecords(ctx, func(*roles.RecordListOpts) error { cancel(cause); return nil })
	if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || calls != before {
		t.Fatal(err, calls)
	}
	old := api.RawClient().Endpoint
	_, err = api.AllRecords(context.Background(), func(*roles.RecordListOpts) error { api.RawClient().Endpoint = "https://changed.invalid/"; return nil })
	api.RawClient().Endpoint = old
	if !errors.Is(err, resource.ErrInvalidOption) || calls != before {
		t.Fatal(err, calls)
	}
}
func TestPythonRoleRecordsExplicitLocationOwnsCurrentScopeWithoutUnknownProjectInference(t *testing.T) {
	cloudName, projectName := "configured", "current"
	location := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &projectName}}
	option := roles.WithRecordListLocation(location)
	cloudName = "changed"
	projectName = "changed"
	api, _ := roleRecordAPI(t, func(*http.Request) *http.Response {
		return roleRecordWire(200, `{"roles":[{"id":"a","project_id":"foreign","location":{"cloud":"wire"}}]}`)
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
func TestPythonRoleRecordsMalformedRowsAndLocalFilterFaultRetainAcceptedPage(t *testing.T) {
	for _, body := range []string{`{"roles":null}`, `{"roles":[null]}`, `{"roles":[{"id":"a","connection":true}]}`, `{"roles":[{"id":"a","links":3}]}`, `{"roles":[]} trailing`} {
		api, _ := roleRecordAPI(t, func(*http.Request) *http.Response { return roleRecordWire(203, body) })
		opts := []roles.RecordListOption{}
		if strings.Contains(body, `"links":3`) {
			opts = append(opts, roles.WithRecordListQuery("links", map[string]any{"key": "value"}))
		}
		rows, err := api.AllRecords(context.Background(), opts...)
		var proof *resource.ResponseError
		if rows != nil || err == nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "role-proof" {
			t.Fatal(body, rows, err, proof)
		}
	}
}

type roleRecordFaultBody struct {
	data            string
	read, close     error
	onRead, onClose func()
	used            bool
	closed          int
}

func (b *roleRecordFaultBody) Read(p []byte) (int, error) {
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
func (b *roleRecordFaultBody) Close() error {
	b.closed++
	if b.onClose != nil {
		b.onClose()
	}
	return b.close
}
func TestPythonRoleRecordsPhysicalFaultsKeepNative401AndActualReceiptWithoutCallbacks(t *testing.T) {
	for _, status := range []int{200, 400, 403, 404, 401, 429, 498, 503} {
		for _, mode := range []string{"read", "close", "both", "restore cancel"} {
			t.Run(fmt.Sprintf("%d/%s", status, mode), func(t *testing.T) {
				calls, retries, reauth, backoff := 0, 0, 0, 0
				readCause := errors.New("role physical read")
				closeCause := errors.New("role physical close")
				cancelCause := errors.New("role caller canceled")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				body := &roleRecordFaultBody{data: `{"roles":[{"id":"a"}]}`}
				if mode == "read" || mode == "both" {
					body.read = readCause
				}
				if mode == "close" || mode == "both" {
					body.close = closeCause
				}
				api, fixture := roleRecordAPI(t, func(r *http.Request) *http.Response {
					calls++
					response := roleRecordWire(status, body.data)
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
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body.data || proof.Header.Get("X-Request-Id") != "role-proof" {
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
					if !errors.As(original, &proof) || proof.Actual != status || string(proof.Body) != body.data || proof.ResponseHeader.Get("X-Request-Id") != "role-proof" || proof.Method != http.MethodGet || !strings.Contains(proof.URL, pythonRoleBase) {
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
func TestPythonRoleRecordsCleanReauthRetryAndBackoffKeepFrozenIdentityRequest(t *testing.T) {
	for _, first := range []int{401, 503, 429, 498} {
		t.Run(fmt.Sprint(first), func(t *testing.T) {
			calls, reauth, retries, backoff := 0, 0, 0, 0
			api, fixture := roleRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				if r.Header.Get("OpenStack-API-Version") != "identity 3.1" || r.Header.Get("X-Selected") != "fixed" || r.Header.Get("If-Match") != "" || r.URL.Query().Get("domain_id") != "d" {
					t.Fatal(r.URL, r.Header)
				}
				if calls == 1 {
					return roleRecordWire(first, `{"error":"retry"}`)
				}
				if first == 401 && r.Header.Get("X-Auth-Token") != "renewed" {
					t.Fatal(r.Header)
				}
				return roleRecordWire(200, `{"roles":[]}`)
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
			_, err := api.AllRecords(context.Background(), roles.WithRecordListQuery("domain_id", "d"), roles.WithRecordListMicroversion("3.1"))
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
