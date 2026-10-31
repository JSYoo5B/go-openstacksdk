package users_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Existing Gophercloud/testcloud fixture and public transport carrier are
// reused; only operation-specific physical response decoration lives here.
func recordAPI(t *testing.T, calls *[]pythonUserCall, reply func(*http.Request) *http.Response) *sdk.API {
	t.Helper()
	api := pythonUserAPI(t, calls, func(*http.Request) (int, string) { t.Fatal("base fixture unexpectedly used"); return 500, "" })
	api.RawClient().HTTPClient.Transport = testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonUserCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	return api
}
func recordUserWire(status int, body string) *http.Response {
	return testcloud.Response(status, body, http.Header{"Content-Type": {"application/json"}})
}
func recordListOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "users" {
		t.Fatal(err, wrapped)
	}
}
func recordIDs(t *testing.T, rows []*sdk.UserRecord) []string {
	t.Helper()
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		var id string
		if err := json.Unmarshal(row.Resource.Body["id"], &id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids
}
func recordField(t *testing.T, row *sdk.UserRecord, key, want string) {
	t.Helper()
	if !bytes.Equal(bytes.TrimSpace(row.Resource.Body[key]), []byte(want)) {
		t.Fatal(key, string(row.Resource.Body[key]), want)
	}
}

func TestPythonUserListRecordsLazyQueryAndUntypedView(t *testing.T) {
	var calls []pythonUserCall
	body := `{"users":[{"id":"one","name":0,"description":false,"email":"10","links":{"self":"x","nested":{"flag":true}},"unique_id":"svc","region_id":"r","resource_name":"cores","project_id":"p","domain_id":"d","self":"ignored","x_extension":{"keep":1},"location":{"cloud":"untrusted"}},{"id":"two"}]}`
	api := recordAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.Method != "GET" || r.URL.Path != "/keystone/v3/users" || r.URL.RawQuery != "domain_id=d&name=alice&password_expires_at=never" || r.Header.Get("Accept") != "application/json" || r.Header.Get("OpenStack-API-Version") != "" {
			t.Fatal(r.Method, r.URL, r.Header)
		}
		response := recordUserWire(203, body)
		response.Header.Set("X-Request-Id", "actual-page")
		return response
	})
	encoded := 0
	seq := api.ListRecords(context.Background(), sdk.WithRecordListQuery("domain_id", "d"), sdk.WithRecordListQuery("name", "alice"), sdk.WithRecordListQuery("password_expires_at", "never"), sdk.WithRecordListQuery("x_extension", recordMarshaler{&encoded}))
	if len(calls) != 0 {
		t.Fatal(calls)
	}
	rows := make([]*sdk.UserRecord, 0)
	for row, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if len(rows) != 2 || len(calls) != 1 || calls[0].body != "" || encoded != 0 {
		t.Fatal(rows, calls, encoded)
	}
	recordField(t, rows[0], "email", `"10"`)
	recordField(t, rows[0], "name", "0")
	recordField(t, rows[0], "description", "false")
	for _, key := range []string{"x_extension", "self", "project_id"} {
		if _, exists := rows[0].Resource.Body[key]; exists {
			t.Fatal(key, rows[0].Resource.Body)
		}
	}
	if string(rows[0].Wire.Body["x_extension"]) != `{"keep":1}` || string(rows[0].Wire.Body["location"]) != `{"cloud":"untrusted"}` || string(rows[0].Envelope) != body || rows[0].Header.Get("X-Request-Id") != "actual-page" || rows[0].StatusCode != 203 {
		t.Fatal(rows[0])
	}
	recordField(t, rows[1], "email", "null")
	recordField(t, rows[1], "links", "null")
	rows[0].Resource.Body["id"][1] = 'X'
	rows[0].Header.Set("X-Request-Id", "changed")
	if string(rows[0].Wire.Body["id"]) != `"one"` || rows[0].Wire.Header.Get("X-Request-Id") != "actual-page" || string(rows[0].Envelope) != body {
		t.Fatal(rows[0])
	}
}

type recordMarshaler struct{ calls *int }

func (v recordMarshaler) MarshalJSON() ([]byte, error) {
	*v.calls++
	return nil, errors.New("must not marshal unknown kwarg")
}

func TestPythonUserListRecordsDictionaryLinksIgnoredWithSourcePrecedence(t *testing.T) {
	for _, extra := range []string{`,"links":{"next":"https://untrusted.invalid/v3/users"}`, `,"links":{"next":false},"users_links":[{"rel":"next","href":"https://untrusted.invalid/users"}]`} {
		var calls []pythonUserCall
		api := recordAPI(t, &calls, func(*http.Request) *http.Response { return recordUserWire(200, `{"users":[{"id":"one"}]`+extra+`}`) })
		rows, err := api.AllRecords(context.Background())
		if err != nil || len(rows) != 1 || len(calls) != 1 {
			t.Fatal(extra, rows, err, calls)
		}
	}
}

func TestPythonUserListRecordsFilteredMarkerAndRawMaxItems(t *testing.T) {
	for _, mode := range []string{"marker", "max_items", "single_page"} {
		t.Run(mode, func(t *testing.T) {
			var calls []pythonUserCall
			api := recordAPI(t, &calls, func(r *http.Request) *http.Response {
				if r.URL.Query().Get("limit") != "5" || r.URL.Query().Get("unique_id") != "svc" || r.URL.Query().Has("description") {
					t.Fatal(r.URL)
				}
				if r.URL.Query().Get("marker") == "last-filtered" {
					return recordUserWire(200, `{"users":[]}`)
				}
				return recordUserWire(200, `{"users":[{"id":"kept","description":"yes"},{"id":"last-filtered","description":"no"}],"links":{"next":"ignored.invalid"}}`)
			})
			opts := []sdk.RecordListOption{sdk.WithRecordListLimit(5), sdk.WithRecordListQuery("unique_id", "svc"), sdk.WithRecordListQuery("description", "yes")}
			if mode == "max_items" {
				opts = append(opts, sdk.WithRecordListMaxItems(2))
			}
			if mode == "single_page" {
				opts = append(opts, sdk.WithRecordListPaginated(false))
			}
			rows, err := api.AllRecords(context.Background(), opts...)
			if err != nil || !reflect.DeepEqual(recordIDs(t, rows), []string{"kept"}) {
				t.Fatal(rows, err)
			}
			want := 1
			if mode == "marker" {
				want = 2
			}
			if len(calls) != want || want == 2 && !strings.Contains(calls[1].query, "marker=last-filtered") {
				t.Fatal(calls)
			}
		})
	}
	// max_items introduces a limit; zero remains unbounded and no default limit.
	for _, max := range []int{0, 1} {
		var calls []pythonUserCall
		api := recordAPI(t, &calls, func(r *http.Request) *http.Response {
			want := ""
			if max == 1 {
				want = "1"
			}
			if r.URL.Query().Get("limit") != want {
				t.Fatal(r.URL)
			}
			return recordUserWire(200, `{"users":[{"id":"one"}]}`)
		})
		rows, err := api.AllRecords(context.Background(), sdk.WithRecordListMaxItems(max))
		if err != nil || len(rows) != 1 || len(calls) != 1 {
			t.Fatal(rows, err, calls)
		}
	}
}

func TestPythonUserListRecordsLoadedLocalFilterAndShortCircuit(t *testing.T) {
	for _, tc := range []struct {
		filter json.RawMessage
		want   []string
		fail   bool
	}{
		{json.RawMessage(`{"count":9007199254740993.0,"flag":true}`), []string{"one"}, false},
		{json.RawMessage(`{"count":1,"nested":{"ok":true}}`), []string{}, false},
		{json.RawMessage(`{"nested":{"ok":true},"count":1}`), nil, true},
		{json.RawMessage(`{}`), []string{"one"}, false},
	} {
		var calls []pythonUserCall
		raw := `{"users":[{"id":"one","links":{"count":9007199254740992,"flag":1,"nested":true}},{"id":"two","links":null}]}`
		api := recordAPI(t, &calls, func(*http.Request) *http.Response { return recordUserWire(200, raw) })
		rows, err := api.AllRecords(context.Background(), sdk.WithRecordListQuery("links", tc.filter))
		if tc.fail {
			var evidence *resource.ResponseError
			if rows != nil || !errors.As(err, &evidence) || string(evidence.Body) != raw {
				t.Fatal(rows, err, evidence)
			}
			continue
		}
		if err != nil || !reflect.DeepEqual(recordIDs(t, rows), tc.want) || len(calls) != 1 || calls[0].query != "" {
			t.Fatal(rows, err, calls)
		}
	}
	// Scalar numeric comparisons also use CPython's loaded float equality.
	var calls []pythonUserCall
	api := recordAPI(t, &calls, func(*http.Request) *http.Response {
		return recordUserWire(200, `{"users":[{"id":"one","email":9007199254740992},{"id":"two","email":"9007199254740992"}]}`)
	})
	rows, err := api.AllRecords(context.Background(), sdk.WithRecordListQuery("email", json.RawMessage(`9007199254740993.0`)))
	if err != nil || !reflect.DeepEqual(recordIDs(t, rows), []string{"one"}) || calls[0].query != "" {
		t.Fatal(rows, err, calls)
	}
}

func TestPythonUserListRecordsSingletonMalformedAndConstructorEvidence(t *testing.T) {
	var calls []pythonUserCall
	api := recordAPI(t, &calls, func(*http.Request) *http.Response {
		return recordUserWire(200, `{"users":{"id":"one","email":false}}`)
	})
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	recordField(t, rows[0], "email", "false")
	for _, payload := range []string{`{}`, `{"users":null}`, `{"users":[null]}`, `{"users":[4]}`, `{"users":[{"id":"one","cls":null}]}`, `{"users":[{"id":"one","connection":null}]}`, `{"users":[{"id":"one","microversion":null}]}`, `{"users":[{"id":"one","_synchronized":false}]}`, `{"users":[{"id":"one"}],"next":true}`, ``} {
		var calls []pythonUserCall
		status := 203
		if payload == "" {
			status = 204
		}
		api := recordAPI(t, &calls, func(*http.Request) *http.Response {
			r := recordUserWire(status, payload)
			r.Header.Set("X-Request-Id", "malformed")
			return r
		})
		rows, err := api.AllRecords(context.Background())
		var proof *resource.ResponseError
		if rows != nil || !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != payload || proof.Header.Get("X-Request-Id") != "malformed" || len(calls) != 1 {
			t.Fatal(payload, rows, err, proof, calls)
		}
		recordListOperation(t, err, "ListRecords")
	}
}

func TestPythonUserListRecordsOptionsSnapshotFormalRestorationAndHeaders(t *testing.T) {
	var calls []pythonUserCall
	api := recordAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.Header.Get("X-Trace") != "restored" || r.Header.Get("Accept") != "custom/json" || r.Header.Get("OpenStack-API-Version") != "identity 3.99" || r.URL.RawQuery != "unique_id=new" {
			t.Fatal(r.Header, r.URL)
		}
		return recordUserWire(200, `{"users":[{"id":"one","links":{"keep":1}}]}`)
	})
	opts := sdk.RecordListOpts{Query: map[string]json.RawMessage{"unique_id": json.RawMessage(`"old"`)}}
	snapshot := sdk.WithRecordListOpts(opts)
	opts.Query["unique_id"] = json.RawMessage(`"mutated"`)
	raw := json.RawMessage(`{"unique_id":"new","links":{"keep":1},"headers":{"X-Trace":"restored","Accept":"custom/json"},"microversion":"3.99","jmespath_filters":"length(@)","location":false,"__conflicting_attrs":{"unique_id":"must-not-recurse"}}`)
	restore := sdk.WithRecordListQuery("__conflicting_attrs", raw)
	raw[1] = 'X'
	rows, err := api.AllRecords(context.Background(), snapshot, restore)
	if err != nil || len(rows) != 1 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	n := len(calls)
	for _, v := range []json.RawMessage{json.RawMessage(`{"paginated":false}`), json.RawMessage(`{"base_path":"/users"}`), json.RawMessage(`{"cls":null}`), json.RawMessage(`true`)} {
		rows, err = api.AllRecords(context.Background(), sdk.WithRecordListQuery("__conflicting_attrs", v))
		if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != n {
			t.Fatal(v, rows, err, calls)
		}
	}
	for _, key := range []string{"cls", "session", "self", "resource_type"} {
		rows, err = api.AllRecords(context.Background(), sdk.WithRecordListQuery(key, nil))
		if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != n {
			t.Fatal(key, rows, err, calls)
		}
	}
}

func TestPythonUserListRecordsUnknownDiscardBeforeEncodingAndTypedJMESBoundary(t *testing.T) {
	var calls []pythonUserCall
	encoded := 0
	api := recordAPI(t, &calls, func(*http.Request) *http.Response { return recordUserWire(200, `{"users":[]}`) })
	options := []sdk.RecordListOption{}
	for _, key := range []string{"if_match", "revision_number", "tags", "project_id_UNKNOWN", "unknown", "connection", "_synchronized", "location", "allow_unknown_params", "list_base_path"} {
		options = append(options, sdk.WithRecordListQuery(key, recordMarshaler{&encoded}))
	}
	rows, err := api.AllRecords(context.Background(), options...)
	if err != nil || rows == nil || len(rows) != 0 || encoded != 0 || len(calls) != 1 || calls[0].query != "" {
		t.Fatal(rows, err, encoded, calls)
	}
	rows, err = api.AllRecords(context.Background(), sdk.WithRecordListJMESPath("[*].id"))
	if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
}

func TestPythonUserProxyListOpaqueProjectionAndRetainedGenerators(t *testing.T) {
	for _, expression := range []string{"[*].id", "[0]", "type(@)", "`true`"} {
		var calls []pythonUserCall
		api := recordAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("expression consumed generator"); return nil })
		result, err := api.ProxyListRecords(context.Background(), sdk.WithRecordListJMESPath(expression))
		want := "null"
		if expression == "`true`" {
			want = "true"
		}
		if err != nil || result == nil || string(result.JSON) != want || len(calls) != 0 {
			t.Fatal(expression, result, err, calls)
		}
	}
	for _, expression := range []string{"length(@)", "to_string(@)", "sort_by(@,&id)", "["} {
		var calls []pythonUserCall
		api := recordAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("type error consumed generator"); return nil })
		result, err := api.ProxyListRecords(context.Background(), sdk.WithRecordListJMESPath(expression))
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(expression, result, err, calls)
		}
	}
	for _, filter := range []any{nil, false, 0, []string{}, map[string]any{}, true, 1, 1e-300, "", "@", "to_array(@)", "not_null(@,`[]`)"} {
		var calls []pythonUserCall
		api := recordAPI(t, &calls, func(*http.Request) *http.Response { return recordUserWire(200, `{"users":[{"id":"one"}]}`) })
		result, err := api.ProxyListRecords(context.Background(), sdk.WithRecordListQuery("jmespath_filters", filter))
		if err != nil || result == nil || result.JSON != nil || len(calls) != 0 {
			t.Fatal(filter, result, err, calls)
		}
		value := result.Value
		if array, ok := value.([]any); ok {
			if len(array) != 1 {
				t.Fatal(array)
			}
			value = array[0]
		}
		generator, ok := value.(*sdk.RecordListGenerator)
		if !ok {
			t.Fatal(filter, value)
		}
		n := 0
		for row, err := range generator.Records {
			if err != nil || row == nil {
				t.Fatal(row, err)
			}
			n++
		}
		if n != 1 || len(calls) != 1 {
			t.Fatal(filter, n, calls)
		}
	}
}

func TestPythonUserListRecordsGuardsOptionCancellationAndContextCauses(t *testing.T) {
	for _, mode := range []string{"context", "source"} {
		var calls []pythonUserCall
		api := recordAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("invalid operation reached HTTP"); return nil })
		ctx, cancel := context.WithCancelCause(context.Background())
		cause := errors.New("caller canceled")
		later := false
		options := []sdk.RecordListOption{func(*sdk.RecordListOpts) error {
			if mode == "context" {
				cancel(cause)
			} else {
				api.RawClient().ResourceBase = "https://changed.invalid/v3/"
			}
			return nil
		}, func(*sdk.RecordListOpts) error { later = true; return nil }}
		rows, err := api.AllRecords(ctx, options...)
		if rows != nil || err == nil || len(calls) != 0 || later {
			t.Fatal(mode, rows, err, calls, later)
		}
		if mode == "context" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
			t.Fatal(err)
		}
		cancel(nil)
	}
	var calls []pythonUserCall
	api := recordAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("invalid context reached HTTP"); return nil })
	for _, ctx := range []context.Context{nil} {
		rows, err := api.AllRecords(ctx)
		if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(rows, err)
		}
	}
	for _, options := range [][]sdk.RecordListOption{{nil}, {sdk.WithRecordListHeaders(map[string]string{"Authorization": "forbidden"})}, {sdk.WithRecordListBasePath("https://changed.invalid/users")}} {
		rows, err := api.AllRecords(context.Background(), options...)
		if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(rows, err, calls)
		}
	}
}

func TestPythonUserListRecordsIteratorKeepsYieldedRowsAndAllDiscards(t *testing.T) {
	for _, collect := range []bool{false, true} {
		var calls []pythonUserCall
		raw := `{"users":[{"id":"one"}],"next":"https://changed.invalid/users?page=2"}`
		api := recordAPI(t, &calls, func(*http.Request) *http.Response { return recordUserWire(200, raw) })
		var err error
		n := 0
		if collect {
			rows, e := api.AllRecords(context.Background())
			err = e
			if rows != nil {
				t.Fatal(rows)
			}
		} else {
			for row, e := range api.ListRecords(context.Background()) {
				if e != nil {
					err = e
					break
				}
				if row == nil {
					t.Fatal(row)
				}
				n++
			}
		}
		var evidence *resource.ResponseError
		if !errors.As(err, &evidence) || string(evidence.Body) != raw || len(calls) != 1 || !collect && n != 1 {
			t.Fatal(err, evidence, n, calls)
		}
	}
}

func TestPythonUserListRecordsPhysicalFailuresPreserveNative401AndStopCallbacks(t *testing.T) {
	for _, status := range []int{200, 203, 401, 404, 429, 503} {
		for _, fault := range []string{"read", "close", "mixed"} {
			t.Run(fmt.Sprint(status, fault), func(t *testing.T) {
				readCause, closeCause, hookCause := errors.New("physical read"), errors.New("physical close"), errors.New("callback must not run")
				raw := `{"users":[{"id":"p"}]}`
				if status >= 400 {
					raw = `{"error":{"message":"physical rejection"}}`
				}
				closed := 0
				body := &testcloud.Body{Data: raw, OnClose: func() { closed++ }}
				if fault != "close" {
					body.ReadErr = readCause
				}
				if fault != "read" {
					body.CloseErr = closeCause
				}
				var calls []pythonUserCall
				api := recordAPI(t, &calls, func(*http.Request) *http.Response {
					r := recordUserWire(status, raw)
					r.Body = body
					r.Header.Set("X-Request-Id", "actual-identity-list")
					return r
				})
				retries, reauth := 0, 0
				api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return hookCause
				}
				api.RawClient().ReauthFunc = func(context.Context) error { reauth++; return nil }
				rows, err := api.AllRecords(context.Background())
				if rows != nil || err == nil || len(calls) != 1 || closed != 1 || retries != 0 || reauth != 0 || errors.Is(err, hookCause) {
					t.Fatal(rows, err, calls, closed, retries, reauth)
				}
				assertCauses := func(e error) {
					t.Helper()
					if fault != "close" && !errors.Is(e, readCause) || fault != "read" && !errors.Is(e, closeCause) {
						t.Fatal(e)
					}
				}
				assertCauses(err)
				if status < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != raw || proof.Header.Get("X-Request-Id") != "actual-identity-list" {
						t.Fatal(err, proof)
					}
					return
				}
				original := err
				if status == 401 {
					var native *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &native) || native.ErrOriginal == nil || native.ErrReauth == nil {
						t.Fatal(err, native)
					}
					assertCauses(native.ErrReauth)
					original = native.ErrOriginal
				}
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(original, &proof) || proof.Actual != status || proof.Method != http.MethodGet || proof.URL != api.RawClient().ServiceURL("users") || string(proof.Body) != raw || proof.ResponseHeader.Get("X-Request-Id") != "actual-identity-list" {
					t.Fatal(err, proof)
				}
			})
		}
	}
}

func TestPythonUserListRecordsCleanReauthenticationAndRetryKeepPolicy(t *testing.T) {
	for _, status := range []int{401, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls []pythonUserCall
			var tokens []string
			api := recordAPI(t, &calls, func(r *http.Request) *http.Response {
				tokens = append(tokens, r.Header.Get("X-Auth-Token"))
				if r.Header.Get("If-Match") != "configured" || r.Header.Get("Accept") != "application/json" || r.Header.Get("OpenStack-API-Version") != "identity 2.1" || r.URL.RawQuery != "unique_id=n" {
					t.Fatal(r.Header, r.URL)
				}
				if len(calls) == 1 {
					return recordUserWire(status, `{"error":{"message":"retry"}}`)
				}
				return recordUserWire(200, `{"users":[{"id":"p"}]}`)
			})
			api.RawClient().Microversion = "9.9"
			api.RawClient().MoreHeaders = map[string]string{"Accept": "configured", "If-Match": "configured"}
			retries, reauth := 0, 0
			api.RawClient().ReauthFunc = func(context.Context) error { reauth++; api.RawClient().SetToken("renewed"); return nil }
			api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, c *gophercloud.RequestOpts, _ error, _ uint) error {
				retries++
				c.OkCodes = append(c.OkCodes, 503)
				c.MoreHeaders["X-Retry-Probe"] = "ordinary"
				return nil
			}
			rows, err := api.AllRecords(context.Background(), sdk.WithRecordListMicroversion("2.1"), sdk.WithRecordListQuery("unique_id", "n"))
			if err != nil || len(rows) != 1 || len(calls) != 2 || status == 401 && (reauth != 1 || retries != 0 || !reflect.DeepEqual(tokens, []string{"test-token", "renewed"})) || status == 503 && (retries != 1 || reauth != 0) {
				t.Fatal(rows, err, calls, tokens, retries, reauth)
			}
			if api.RawClient().Microversion != "9.9" || api.RawClient().MoreHeaders["Accept"] != "configured" {
				t.Fatal(api.RawClient())
			}
		})
	}
}

func TestPythonUserListRecordsLocationIsExplicitImmutableAndDeclared(t *testing.T) {
	var calls []pythonUserCall
	raw := `{"users":[{"id":"one","project_id":"foreign","location":{"cloud":"server"}},{"id":"two","project_id":false}]}`
	api := recordAPI(t, &calls, func(*http.Request) *http.Response { return recordUserWire(200, raw) })
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 2 {
		t.Fatal(rows, err)
	}
	recordField(t, rows[0], "location", "null")
	cloud, project, region := "recorded", "configured", "RegionOne"
	location := resource.CloudLocation{Cloud: &cloud, RegionName: &region, Project: resource.CloudProject{ID: json.RawMessage(`"current"`), Name: &project}}
	option := sdk.WithRecordListLocation(location)
	cloud = "mutated"
	location.Project.ID[1] = 'X'
	rows, err = api.AllRecords(context.Background(), option)
	if err != nil || len(rows) != 2 {
		t.Fatal(rows, err)
	}
	var computed map[string]json.RawMessage
	if err := json.Unmarshal(rows[0].Resource.Body["location"], &computed); err != nil {
		t.Fatal(err)
	}
	if string(computed["cloud"]) != `"recorded"` || string(computed["region_name"]) != `"RegionOne"` || string(rows[0].Wire.Body["location"]) != `{"cloud":"server"}` {
		t.Fatal(computed, rows[0].Wire.Body)
	}
	var p map[string]json.RawMessage
	if err := json.Unmarshal(computed["project"], &p); err != nil {
		t.Fatal(err)
	}
	wantID, wantName := `"current"`, `"configured"`

	if string(p["id"]) != wantID || string(p["name"]) != wantName {
		t.Fatal(p)
	}
	if err := json.Unmarshal(rows[1].Resource.Body["location"], &computed); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(computed["project"], &p); err != nil {
		t.Fatal(err)
	}
	if string(p["id"]) != `"current"` || string(p["name"]) != `"configured"` {
		t.Fatal(p)
	}
	n := len(calls)
	rows, err = api.AllRecords(context.Background(), sdk.WithRecordListLocation(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`not JSON`)}}))
	if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != n {
		t.Fatal(rows, err, calls)
	}
}

func TestPythonUserListRecordsCloseCancellationKeepsReceiptAndCause(t *testing.T) {
	cause := errors.New("canceled in physical Close")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	var calls []pythonUserCall
	raw := `{"users":[{"id":"one"}]}`
	closed := 0
	api := recordAPI(t, &calls, func(*http.Request) *http.Response {
		response := recordUserWire(203, raw)
		response.Header.Set("X-Request-Id", "cancel-close")
		response.Body = &testcloud.Body{Data: raw, OnClose: func() { closed++; cancel(cause) }}
		return response
	})
	rows, err := api.AllRecords(ctx)
	var receipt *resource.ResponseError
	if rows != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.As(err, &receipt) || receipt.StatusCode != 203 || string(receipt.Body) != raw || receipt.Header.Get("X-Request-Id") != "cancel-close" || closed != 1 || len(calls) != 1 {
		t.Fatal(rows, err, receipt, closed, calls)
	}
}

func TestPythonUserListRecordsSourceSuccessStatusAndHeaderNoDiscovery(t *testing.T) {
	for _, status := range []int{101, 200, 203, 301, 304, 399} {
		var calls []pythonUserCall
		api := recordAPI(t, &calls, func(r *http.Request) *http.Response {
			if r.Header.Get("OpenStack-API-Version") != "identity 3.5" || r.Header.Get("Accept") != "application/json" || r.URL.Path != "/keystone/v3/users" {
				t.Fatal(r.Header, r.URL)
			}
			return recordUserWire(status, `{"users":[{"id":"one"}]}`)
		})
		rows, err := api.AllRecords(context.Background(), sdk.WithRecordListMicroversion("3.5"))
		if err != nil || len(rows) != 1 || rows[0].StatusCode != status || len(calls) != 1 {
			t.Fatal(status, rows, err, calls)
		}
	}
}

func TestPythonUserListRecordsExactDescriptorDefaultsAndResponseAliasOrder(t *testing.T) {
	var calls []pythonUserCall
	raw := `{"users":[{"id":"one","enabled":true,"is_enabled":false,"federated":{"idp_id":"idp"},"options":false,"default_project_id":null,"password_expires_at":9007199254740993,"description":0,"x_unknown":{"keep":1}},{"id":"two","is_enabled":false,"enabled":"false","federated":null,"options":null},{"id":"three"},{"id":"four","enabled":1e-400,"federated":0}]}`
	api := recordAPI(t, &calls, func(*http.Request) *http.Response { return recordUserWire(200, raw) })
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 4 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	for _, tc := range []struct {
		index     int
		key, want string
	}{
		{0, "is_enabled", "false"}, {0, "federated", `[{"idp_id":"idp"}]`}, {0, "options", `{}`}, {0, "password_expires_at", "9007199254740993"}, {0, "description", "0"},
		{1, "is_enabled", "true"}, {1, "federated", "null"}, {1, "options", "null"},
		{2, "is_enabled", "null"}, {2, "options", `{}`}, {2, "federated", "null"}, {2, "password", "null"}, {2, "default_project_id", "null"},
		{3, "is_enabled", "false"}, {3, "federated", `[0]`},
	} {
		recordField(t, rows[tc.index], tc.key, tc.want)
	}
	if _, ok := rows[0].Resource.Body["enabled"]; ok {
		t.Fatal(rows[0].Resource.Body)
	}
	if _, ok := rows[0].Resource.Body["x_unknown"]; ok {
		t.Fatal(rows[0].Resource.Body)
	}
	if string(rows[0].Wire.Body["enabled"]) != "true" || string(rows[0].Wire.Body["is_enabled"]) != "false" || string(rows[0].Wire.Body["options"]) != "false" {
		t.Fatal(rows[0].Wire.Body)
	}
	clone := rows[0].Clone()
	clone.Resource.Body["id"][1] = 'X'
	clone.Wire.Body["id"][1] = 'Y'
	clone.Envelope[0] = 'X'
	if string(rows[0].Resource.Body["id"]) != `"one"` || string(rows[0].Wire.Body["id"]) != `"one"` || string(rows[0].Envelope) != raw {
		t.Fatal(rows[0])
	}
}

func TestPythonUserListRecordsDeclaredQueriesCanonicalWinsRequestsValues(t *testing.T) {
	var calls []pythonUserCall
	api := recordAPI(t, &calls, func(r *http.Request) *http.Response {
		q := r.URL.Query()
		if !reflect.DeepEqual(q["name"], []string{"a", "False"}) || q.Get("domain_id") != "" || !q.Has("domain_id") || q.Get("enabled") != "False" || q.Has("password_expires_at") || q.Get("idp_id") != "idp" || q.Get("protocol_id") != "saml" || q.Get("unique_id") != "subject" || q.Has("email") {
			t.Fatal(q)
		}
		return recordUserWire(200, `{"users":[{"id":"one","email":"keep","enabled":true}]}`)
	})
	rows, err := api.AllRecords(context.Background(), sdk.WithRecordListQuery("name", []any{"a", nil, false}), sdk.WithRecordListQuery("domain_id", ""), sdk.WithRecordListQuery("enabled", true), sdk.WithRecordListQuery("is_enabled", false), sdk.WithRecordListQuery("password_expires_at", nil), sdk.WithRecordListQuery("idp_id", "idp"), sdk.WithRecordListQuery("protocol_id", "saml"), sdk.WithRecordListQuery("unique_id", "subject"), sdk.WithRecordListQuery("email", "keep"))
	if err != nil || len(rows) != 1 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	// enabled's mapped filter is remote-only; it does not filter this true row.
	recordField(t, rows[0], "is_enabled", "true")
}

func TestPythonUserListRecordsOptionsDefaultNullAndNestedLoadedLocal(t *testing.T) {
	raw := `{"users":[{"id":"missing"},{"id":"explicit-null","options":null},{"id":"kept","options":{"n":9007199254740992,"nested":{"ok":1},"extra":1}},{"id":"scalar","options":true}]}`
	for _, tc := range []struct {
		filter json.RawMessage
		want   []string
	}{
		{json.RawMessage(`null`), []string{"explicit-null"}},
		{json.RawMessage(`{}`), []string{"kept"}},
		{json.RawMessage(`{"n":9007199254740993.0,"nested":{"ok":true}}`), []string{"kept"}},
	} {
		var calls []pythonUserCall
		api := recordAPI(t, &calls, func(*http.Request) *http.Response { return recordUserWire(200, raw) })
		rows, err := api.AllRecords(context.Background(), sdk.WithRecordListQuery("options", tc.filter))
		if err != nil || !reflect.DeepEqual(recordIDs(t, rows), tc.want) || len(calls) != 1 || calls[0].query != "" {
			t.Fatal(rows, err, calls)
		}
	}
}
