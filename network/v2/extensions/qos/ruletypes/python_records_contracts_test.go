package ruletypes_test

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

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// The existing nativeRuleTypeAPI creates the public testhelper fixture. This
// only replaces its fixed response using the existing native transport type.
func recordAPI(t *testing.T, calls *[]nativeRuleTypeCall, reply func(*http.Request, int) *http.Response) *ruletypes.API {
	t.Helper()
	api := nativeRuleTypeAPI(t, calls, 200, `{}`)
	api.RawClient().ProviderClient.HTTPClient.Transport = nativeRuleTypeTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, nativeRuleTypeCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery})
		if req.Method != "GET" || req.Header.Get("X-Auth-Token") == "" {
			t.Fatal(req)
		}
		return reply(req, len(*calls)), nil
	})
	return api
}
func recordWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"X-Request-Id": []string{"rule-type-proof"}}}
}
func recordField(t *testing.T, v *ruletypes.QoSRuleTypeRecord, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		t.Fatalf("%s want %s got %#v", key, want, v)
	}
}

func TestPythonQoSRuleTypeGetFieldsDefaultsAndOwnedActualEvidence(t *testing.T) {
	for _, fields := range [][]string{nil, {}, {"type", "drivers", "id", "unknown", "type"}} {
		t.Run(fmt.Sprint(fields), func(t *testing.T) {
			var calls []nativeRuleTypeCall
			body := `{"rule_type":{"type":null,"drivers":false,"name":0,"cls":false,"self":"drop","microversion":{"untrusted":true},"location":{"untrusted":"location"},"x_new":9007199254740993}}`
			api := recordAPI(t, &calls, func(req *http.Request, _ int) *http.Response {
				want := url.Values{}
				if len(fields) != 0 {
					want["fields"] = []string{"type", "drivers", "id", "unknown", "type"}
				}
				if req.URL.EscapedPath() != "/neutron/v2.0/qos/rule-types/literal%2Fname%20%252F" || !reflect.DeepEqual(req.URL.Query(), want) || req.Header.Get("OpenStack-API-Version") != "network 2.2" {
					t.Fatal(req.URL, req.Header, want)
				}
				return recordWire(203, body)
			})
			option := ruletypes.WithRecordGetFields(fields...)
			if len(fields) > 0 {
				fields[0] = "changed"
			}
			v, err := api.GetRecord(context.Background(), "literal/name %2F", option, ruletypes.WithRecordGetMicroversion("2.2"))
			if err != nil || len(calls) != 1 || v.StatusCode != 203 || string(v.Envelope) != body || v.Header.Get("X-Request-Id") != "rule-type-proof" {
				t.Fatal(v, err, calls)
			}
			recordField(t, v, "id", `"literal/name %2F"`)
			recordField(t, v, "type", `null`)
			recordField(t, v, "drivers", `[false]`)
			recordField(t, v, "name", `0`)
			recordField(t, v, "location", `null`)
			recordField(t, v, "microversion", `"2.2"`)
			recordField(t, v, "x_new", `9007199254740993`)
			recordField(t, v, "cls", `false`)
			if v.Resource.Body["self"] != nil || v.Resource.Body["fields"] != nil || v.Wire.Body["id"] != nil || string(v.Wire.Body["location"]) != `{"untrusted":"location"}` || string(v.Wire.Body["microversion"]) != `{"untrusted":true}` {
				t.Fatal(v)
			}
			v.Resource.Body["x_new"][0] = 'x'
			v.Envelope[0] = 'x'
			v.Header.Set("X-Request-Id", "changed")
			if string(v.Wire.Body["x_new"]) != `9007199254740993` || v.Wire.Header.Get("X-Request-Id") != "rule-type-proof" || v.Resource.Header.Get("X-Request-Id") != "rule-type-proof" {
				t.Fatal(v)
			}
		})
	}
	for _, body := range []string{"", `not JSON`, `{"rule_type":{}}`} {
		var calls []nativeRuleTypeCall
		api := recordAPI(t, &calls, func(*http.Request, int) *http.Response { return recordWire(200, body) })
		v, err := api.GetRecord(context.Background(), "bandwidth_limit")
		if err != nil || len(calls) != 1 || string(v.Envelope) != body {
			t.Fatal(v, err, calls)
		}
		recordField(t, v, "id", `"bandwidth_limit"`)
		recordField(t, v, "type", `null`)
		recordField(t, v, "drivers", `null`)
		recordField(t, v, "name", `null`)
	}
}

func TestPythonQoSRuleTypeLiteralIDPriorityAndAlternateListID(t *testing.T) {
	for _, body := range []string{`{"type":"bandwidth_limit"}`, `{"id":null,"type":"bandwidth_limit"}`, `{"id":false,"type":0}`, `{"id":"literal","type":"alternate"}`, `{"type":false}`, `{"drivers":{}}`} {
		var calls []nativeRuleTypeCall
		api := recordAPI(t, &calls, func(*http.Request, int) *http.Response { return recordWire(200, `{"rule_types":[`+body+`]}`) })
		values, err := api.AllRecords(context.Background(), ruletypes.WithRecordListPaginated(false))
		if err != nil || len(values) != 1 || len(calls) != 1 {
			t.Fatal(values, err, calls)
		}
		var literal map[string]json.RawMessage
		_ = json.Unmarshal([]byte(body), &literal)
		want := "null"
		if raw, ok := literal["id"]; ok {
			want = string(raw)
		} else if raw, ok := literal["type"]; ok {
			want = string(raw)
		}
		recordField(t, values[0], "id", want)
		if _, present := literal["id"]; !present && values[0].Wire.Body["id"] != nil {
			t.Fatal(values)
		}
		if raw, ok := literal["drivers"]; ok && string(raw) == `{}` {
			recordField(t, values[0], "drivers", `[{}]`)
		} else {
			recordField(t, values[0], "drivers", `null`)
		}
	}
	var calls []nativeRuleTypeCall
	api := recordAPI(t, &calls, func(*http.Request, int) *http.Response {
		return recordWire(200, `{"rule_type":{"id":null,"type":"server"}}`)
	})
	v, err := api.GetRecord(context.Background(), "requested")
	if err != nil {
		t.Fatal(err)
	}
	recordField(t, v, "id", `null`)
	recordField(t, v, "type", `"server"`)
}

func TestPythonQoSRuleTypeFindCleanFallbackDefaultsAndCompleteUniqueness(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		var calls []nativeRuleTypeCall
		api := recordAPI(t, &calls, func(req *http.Request, n int) *http.Response {
			if req.URL.RawQuery != "" {
				t.Fatal("find accepted arbitrary query/name hint", req.URL)
			}
			if n == 1 {
				return recordWire(code, `{"error":"hidden"}`)
			}
			return recordWire(200, `{"rule_types":[{"type":"other"},{"type":"bandwidth_limit","drivers":null}]}`)
		})
		value, err := api.FindRecord(context.Background(), "bandwidth_limit")
		if err != nil || len(calls) != 2 || calls[1].path != "/neutron/v2.0/qos/rule-types" {
			t.Fatal(value, err, calls)
		}
		recordField(t, value, "id", `"bandwidth_limit"`)
		recordField(t, value, "type", `"bandwidth_limit"`)
	}
	for _, strict := range []bool{false, true} {
		var calls []nativeRuleTypeCall
		api := recordAPI(t, &calls, func(*http.Request, int) *http.Response {
			if len(calls) == 1 {
				return recordWire(404, `{}`)
			}
			return recordWire(200, `{"rule_types":[]}`)
		})
		value, err := api.FindRecord(context.Background(), "missing", ruletypes.WithRecordFindIgnoreMissing(!strict))
		if value != nil || len(calls) != 2 || (strict && !errors.Is(err, resource.ErrNotFound)) || (!strict && err != nil) {
			t.Fatal(value, err, calls)
		}
	}
	var calls []nativeRuleTypeCall
	api := recordAPI(t, &calls, func(req *http.Request, n int) *http.Response {
		if n == 1 {
			return recordWire(404, `{}`)
		}
		if n == 2 {
			return recordWire(200, `{"rule_types":[{"type":"duplicate"}],"rule_types_links":[{"rel":"next","href":"?marker=next"}]}`)
		}
		if req.URL.Query().Get("marker") != "next" {
			t.Fatal(req.URL)
		}
		return recordWire(200, `{"rule_types":[{"id":"second","name":"duplicate"}]}`)
	})
	value, err := api.FindRecord(context.Background(), "duplicate")
	if value != nil || !errors.Is(err, resource.ErrAmbiguous) || len(calls) != 3 {
		t.Fatal(value, err, calls)
	}
}

func TestPythonQoSRuleTypeFindGetOnlyVersionOverrideAndLiteralMemberSeed(t *testing.T) {
	var calls []nativeRuleTypeCall
	api := recordAPI(t, &calls, func(req *http.Request, n int) *http.Response {
		want := "network 2.2"
		if n > 1 {
			want = "network 2.1"
		}
		if req.Header.Get("OpenStack-API-Version") != want || req.URL.RawQuery != "" {
			t.Fatal(req.Header, req.URL)
		}
		if n == 1 {
			return recordWire(403, `{}`)
		}
		return recordWire(200, `{"rule_types":[{"type":"bandwidth_limit"}]}`)
	})
	api.RawClient().Microversion = "2.1"
	value, err := api.FindRecord(context.Background(), "bandwidth_limit", ruletypes.WithRecordFindMicroversion("2.2"))
	if err != nil || value == nil || len(calls) != 2 || api.RawClient().Microversion != "2.1" {
		t.Fatal(value, err, calls)
	}
	recordField(t, value, "id", `"bandwidth_limit"`)
	calls = nil
	api = recordAPI(t, &calls, func(*http.Request, int) *http.Response {
		return recordWire(200, `{"rule_type":{"type":"server-type","microversion":false}}`)
	})
	value, err = api.FindRecord(context.Background(), "requested")
	if err != nil || len(calls) != 1 {
		t.Fatal(value, err, calls)
	}
	recordField(t, value, "id", `"requested"`)
	recordField(t, value, "type", `"server-type"`)
	recordField(t, value, "microversion", `null`)
	if string(value.Wire.Body["microversion"]) != `false` {
		t.Fatal(value)
	}
}

func TestPythonQoSRuleTypeListQueryClassificationRawCapAndUnknownDiscard(t *testing.T) {
	var calls []nativeRuleTypeCall
	api := recordAPI(t, &calls, func(req *http.Request, _ int) *http.Response {
		want := url.Values{"type": {"wanted"}, "drivers": {"False"}, "fields": {"type", "drivers"}, "all_rules": {"True"}, "all_supported": {"0"}, "limit": {"2"}, "marker": {"start"}}
		if !reflect.DeepEqual(req.URL.Query(), want) || req.Header.Get("Accept") != "application/json" {
			t.Fatal(req.URL, req.Header, want)
		}
		return recordWire(200, `{"rule_types":[{"type":"first","name":"other"},{"type":"second","name":"wanted","drivers":false},{"type":"third","name":"wanted"}]}`)
	})
	query := map[string]json.RawMessage{"type": json.RawMessage(`"wanted"`), "drivers": json.RawMessage(`false`), "fields": json.RawMessage(`["type","drivers"]`), "all_rules": json.RawMessage(`true`), "all_supported": json.RawMessage(`0`), "marker": json.RawMessage(`"start"`), "name": json.RawMessage(`"wanted"`), "unknown": json.RawMessage(`invalid ignored`)}
	option := ruletypes.WithRecordListOpts(ruletypes.RecordListOpts{Query: query})
	query["type"] = json.RawMessage(`"changed"`)
	values, err := api.AllRecords(context.Background(), option, ruletypes.WithRecordListMaxItems(2))
	if err != nil || len(values) != 1 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
	recordField(t, values[0], "id", `"second"`)
	recordField(t, values[0], "drivers", `[false]`)
}

func TestPythonQoSRuleTypeListLocalIdentityUsesAlternateTypeAndExactJSONPolicy(t *testing.T) {
	for _, tc := range []struct {
		condition json.RawMessage
		body      string
		want      int
	}{
		{json.RawMessage(`true`), `{"rule_types":[{"type":1},{"type":false}]}`, 1},
		{json.RawMessage(`9007199254740992`), `{"rule_types":[{"type":9007199254740993.0}]}`, 0},
		{json.RawMessage(`0`), `{"rule_types":[{"type":1e-4000}]}`, 0},
	} {
		var calls []nativeRuleTypeCall
		api := recordAPI(t, &calls, func(req *http.Request, _ int) *http.Response {
			if req.URL.RawQuery != "" {
				t.Fatal(req.URL)
			}
			return recordWire(200, tc.body)
		})
		values, err := api.AllRecords(context.Background(), ruletypes.WithRecordListOpts(ruletypes.RecordListOpts{Query: map[string]json.RawMessage{"id": tc.condition}}), ruletypes.WithRecordListPaginated(false))
		if err != nil || len(values) != tc.want || len(calls) != 1 {
			t.Fatal(values, err, calls)
		}
	}
}

func TestPythonQoSRuleTypeListPagingUsesAlternateAndLiteralWireIDPriority(t *testing.T) {
	for _, row := range []string{`{"type":"alternate/id %"}`, `{"id":"literal/id %","type":"ignored"}`, `{"id":false,"type":"ignored"}`} {
		var calls []nativeRuleTypeCall
		api := recordAPI(t, &calls, func(req *http.Request, n int) *http.Response {
			if n == 1 {
				return recordWire(200, `{"rule_types":[`+row+`]}`)
			}
			want := "alternate/id %"
			if strings.Contains(row, "literal/id") {
				want = "literal/id %"
			}
			if strings.Contains(row, `"id":false`) {
				want = "False"
			}
			if req.URL.Query().Get("marker") != want || req.URL.Query().Get("limit") != "2" {
				t.Fatal(req.URL, want)
			}
			return recordWire(200, `{"rule_types":[]}`)
		})
		values, err := api.AllRecords(context.Background(), ruletypes.WithRecordListLimit(2))
		if err != nil || len(values) != 1 || len(calls) != 2 {
			t.Fatal(values, err, calls)
		}
	}
	// A present null id never silently falls back to type in the common marker.
	var calls []nativeRuleTypeCall
	body := `{"rule_types":[{"id":null,"type":"must-not-route"}]}`
	api := recordAPI(t, &calls, func(*http.Request, int) *http.Response { return recordWire(200, body) })
	values, err := api.AllRecords(context.Background(), ruletypes.WithRecordListLimit(2))
	var receipt *resource.ResponseError
	if values != nil || !errors.As(err, &receipt) || string(receipt.Body) != body || len(calls) != 1 {
		t.Fatal(values, err, receipt, calls)
	}
}

func TestPythonQoSRuleTypeListLinkFormsBreakAndConstructorControlCollisions(t *testing.T) {
	for _, kind := range []string{"body", "http", "dictionary"} {
		var calls []nativeRuleTypeCall
		api := recordAPI(t, &calls, func(req *http.Request, n int) *http.Response {
			if n > 1 {
				if req.URL.Query().Get("marker") != "next" {
					t.Fatal(req.URL)
				}
				return recordWire(200, `{"rule_types":[]}`)
			}
			suffix := ""
			switch kind {
			case "body":
				suffix = `,"rule_types_links":[{"rel":"next","href":"?marker=next"}]`
			case "dictionary":
				suffix = `,"links":{"next":"?marker=next"}`
			}
			response := recordWire(200, `{"rule_types":[{"type":"one"}]`+suffix+`}`)
			if kind == "http" {
				response.Header.Set("Link", `<?marker=next>; rel="next"`)
			}
			return response
		})
		rows, err := api.AllRecords(context.Background())
		if err != nil || len(rows) != 1 || len(calls) != 2 {
			t.Fatal(rows, err, calls)
		}
		calls = nil
		for row, err := range api.ListRecords(context.Background()) {
			if err != nil || row == nil {
				t.Fatal(row, err)
			}
			break
		}
		if len(calls) != 1 {
			t.Fatal(calls)
		}
	}
	for _, key := range []string{"connection", "microversion", "_synchronized", "cls"} {
		for _, value := range []string{"null", "false"} {
			var calls []nativeRuleTypeCall
			body := `{"rule_types":[{"type":"one","` + key + `":` + value + `}]}`
			api := recordAPI(t, &calls, func(*http.Request, int) *http.Response { return recordWire(200, body) })
			rows, err := api.AllRecords(context.Background())
			var receipt *resource.ResponseError
			if rows != nil || !errors.As(err, &receipt) || string(receipt.Body) != body || len(calls) != 1 {
				t.Fatal(rows, err, receipt, calls)
			}
		}
	}
}

func TestPythonQoSRuleTypeErrorsNeverBecomeFalseFoundOrFallback(t *testing.T) {
	for _, lane := range []string{"get", "find", "list"} {
		for _, code := range []int{200, 404, 503} {
			if lane == "find" && code == 404 {
				continue
			}
			var calls []nativeRuleTypeCall
			body := `{"rule_type":null}`
			if lane == "list" {
				body = `{"rule_types":[null]}`
			}
			api := recordAPI(t, &calls, func(*http.Request, int) *http.Response { return recordWire(code, body) })
			var err error
			switch lane {
			case "get":
				_, err = api.GetRecord(context.Background(), "one")
			case "find":
				_, err = api.FindRecord(context.Background(), "one")
			case "list":
				_, err = api.AllRecords(context.Background())
			}
			if err == nil || len(calls) != 1 {
				t.Fatal(lane, code, err, calls)
			}
			if code == 200 {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || string(proof.Body) != body {
					t.Fatal(err, proof)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code {
					t.Fatal(err, native)
				}
			}
		}
	}
}

func TestPythonQoSRuleTypeRecordOptionsPreflightAndLazyCloneIsolation(t *testing.T) {
	var calls []nativeRuleTypeCall
	api := recordAPI(t, &calls, func(*http.Request, int) *http.Response { return recordWire(200, `{"rule_types":[]}`) })
	for _, key := range []string{"cls", "location", "__conflicting_attrs", "connection", "microversion", "base_path"} {
		rows, err := api.AllRecords(context.Background(), ruletypes.WithRecordListQuery(key, false))
		if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(key, rows, err, calls)
		}
	}
	for _, opt := range []ruletypes.RecordFindOption{nil, ruletypes.WithRecordFindHeaders(map[string]string{"Authorization": "forged"}), ruletypes.WithRecordFindMicroversion("bad\nversion")} {
		value, err := api.FindRecord(context.Background(), "one", opt)
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(value, err, calls)
		}
	}
	_, err := api.GetRecord(context.Background(), "one", ruletypes.WithRecordGetFields(string([]byte{255})))
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	called := 0
	option := func(o *ruletypes.RecordListOpts) error {
		called++
		o.Query = map[string]json.RawMessage{"type": json.RawMessage(`"original"`)}
		return nil
	}
	seq := api.ListRecords(context.Background(), option)
	if called != 0 || len(calls) != 0 {
		t.Fatal("list evaluated eagerly", called, calls)
	}
	for _, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
	}
	if called != 1 || len(calls) != 1 || calls[0].query != "type=original" {
		t.Fatal(called, calls)
	}
}

// Existing native transport and public fixture remain the HTTP engine. These
// readers compose only real Read/Close/source/cancel boundary failures.
type recordClosingBody struct {
	io.Reader
	close func() error
}

func (b *recordClosingBody) Close() error { return b.close() }

type recordReadHook struct {
	io.ReadCloser
	after func()
	cause error
	done  bool
}

func (b *recordReadHook) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err == io.EOF && !b.done {
		b.done = true
		b.after()
		return n, b.cause
	}
	return n, err
}

func TestPythonQoSRuleTypeCompoundPhysicalFaultsAreTerminalWithNoRetryCallback(t *testing.T) {
	for _, lane := range []string{"get", "find", "list"} {
		for _, code := range []int{200, 401, 404, 429, 503} {
			t.Run(fmt.Sprintf("%s/%d", lane, code), func(t *testing.T) {
				var calls []nativeRuleTypeCall
				readCause, closeCause, cancelCause := errors.New("physical read"), errors.New("physical close"), errors.New("caller cancel in Close")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				var source *gophercloud.ServiceClient
				originalBase := ""
				closes, retries, reauth := 0, 0, 0
				body := `{"rule_type":{"type":"one"}}`
				if lane == "list" {
					body = `{"rule_types":[{"type":"one"}]}`
				}
				api := recordAPI(t, &calls, func(*http.Request, int) *http.Response {
					r := recordWire(code, body)
					r.Body = &recordReadHook{ReadCloser: &recordClosingBody{Reader: strings.NewReader(body), close: func() error { closes++; source.ResourceBase = originalBase; cancel(cancelCause); return closeCause }}, after: func() { source.ResourceBase = originalBase + "changed/" }, cause: readCause}
					return r
				})
				source = api.RawClient()
				originalBase = source.ResourceBase
				source.ProviderClient.ReauthFunc = func(context.Context) error { reauth++; return nil }
				originalURL := source.ServiceURL("qos/rule-types")
				if lane != "list" {
					originalURL = source.ServiceURL("qos/rule-types", "one")
				}
				source.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				var err error
				switch lane {
				case "get":
					_, err = api.GetRecord(ctx, "one")
				case "find":
					_, err = api.FindRecord(ctx, "one")
				case "list":
					_, err = api.AllRecords(ctx)
				}
				if !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) || len(calls) != 1 || closes != 1 || retries != 0 || reauth != 0 {
					t.Fatal(err, calls, closes, retries)
				}
				if code < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "rule-type-proof" {
						t.Fatal(err, proof)
					}
				} else {
					receiptError := err
					if code == 401 {
						// Native reauth failures keep independent public fields,
						// rather than Unwrap. A willing original callback must
						// never run for physical/source/cancellation faults.
						var auth *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &auth) || auth.ErrOriginal == nil || auth.ErrReauth == nil ||
							!errors.Is(auth.ErrReauth, readCause) || !errors.Is(auth.ErrReauth, closeCause) ||
							!errors.Is(auth.ErrReauth, resource.ErrInvalidOption) || !errors.Is(auth.ErrReauth, context.Canceled) || !errors.Is(auth.ErrReauth, cancelCause) {
							t.Fatal(err, auth)
						}
						receiptError = auth.ErrOriginal
					}
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(receiptError, &native) || native.Actual != code || native.Method != http.MethodGet || native.URL != originalURL || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "rule-type-proof" {
						t.Fatal(err, native)
					}
				}
			})
		}
	}
}

func TestPythonQoSRuleTypeNativeCleanRetryAndOwnedHeaderMutation(t *testing.T) {
	for _, mutate := range []bool{false, true} {
		var calls []nativeRuleTypeCall
		api := recordAPI(t, &calls, func(req *http.Request, n int) *http.Response {
			if req.Header.Get("OpenStack-API-Version") != "network 2.2" {
				t.Fatal(req.Header)
			}
			if n == 1 {
				return recordWire(503, `{"error":"temporary"}`)
			}
			if req.Header.Get("X-Retry") != "native" || req.Header.Get("X-Auth-Token") != "live-token" {
				t.Fatal(req.Header)
			}
			return recordWire(200, `{"rule_type":{"type":"one"}}`)
		})
		retries := 0
		api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
			retries++
			if mutate {
				opts.MoreHeaders["OpenStack-API-Version"] = "network changed"
			} else {
				opts.MoreHeaders["X-Retry"] = "native"
				api.RawClient().ProviderClient.SetToken("live-token")
			}
			return nil
		}
		value, err := api.GetRecord(context.Background(), "one", ruletypes.WithRecordGetMicroversion("2.2"))
		if mutate {
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 || retries != 1 {
				t.Fatal(value, err, calls, retries)
			}
		} else if err != nil || value == nil || len(calls) != 2 || retries != 1 {
			t.Fatal(value, err, calls, retries)
		}
	}
}

func TestPythonQoSRuleTypeNestedDriverParametersRemainExactRawJSON(t *testing.T) {
	drivers := `[{"name":null,"supported_parameters":[{"parameter_type":"range","parameter_values":{"start":0,"end":9007199254740993,"x_new":false}},{"parameter_type":"choices","parameter_values":[null,true,0,"ingress"]}],"extension":{"nested":null}}]`
	row := `{"type":"bandwidth_limit","drivers":` + drivers + `}`
	for _, lane := range []string{"get", "list"} {
		var calls []nativeRuleTypeCall
		api := recordAPI(t, &calls, func(*http.Request, int) *http.Response {
			body := `{"rule_type":` + row + `}`
			if lane == "list" {
				body = `{"rule_types":[` + row + `]}`
			}
			return recordWire(200, body)
		})
		var value *ruletypes.QoSRuleTypeRecord
		var err error
		if lane == "get" {
			value, err = api.GetRecord(context.Background(), "bandwidth_limit")
		} else {
			var rows []*ruletypes.QoSRuleTypeRecord
			rows, err = api.AllRecords(context.Background(), ruletypes.WithRecordListPaginated(false))
			if len(rows) == 1 {
				value = rows[0]
			}
		}
		if err != nil || value == nil || len(calls) != 1 {
			t.Fatal(value, err, calls)
		}
		recordField(t, value, "drivers", drivers)
		value.Resource.Body["drivers"][0] = 'x'
		if string(value.Wire.Body["drivers"]) != drivers {
			t.Fatal("nested drivers evidence aliases", value)
		}
	}
}

func TestPythonQoSRuleTypeCustomCallbacksReceiveInitializedOwnedMaps(t *testing.T) {
	for _, lane := range []string{"get", "find", "list"} {
		for _, mode := range []string{"first", "nil reset", "empty factory", "nil header factory"} {
			t.Run(lane+"/"+mode, func(t *testing.T) {
				var calls []nativeRuleTypeCall
				var retainedHeaders map[string]string
				var retainedQuery map[string]json.RawMessage
				applied := 0
				api := recordAPI(t, &calls, func(req *http.Request, _ int) *http.Response {
					if req.Header.Get("X-Owned") != "selected" || req.URL.RawQuery != "" {
						t.Fatal(req.URL, req.Header)
					}
					retainedHeaders["X-Owned"] = "consumer mutation"
					if retainedQuery != nil {
						retainedQuery["name"][1] = 'X'
					}
					raw := `{"rule_type":{"type":"one","name":"selected"}}`
					if lane == "list" {
						raw = `{"rule_types":[{"type":"one","name":"selected"}]}`
					}
					return recordWire(200, raw)
				})
				switch lane {
				case "get":
					var options []ruletypes.RecordGetOption
					switch mode {
					case "nil reset":
						options = append(options, func(o *ruletypes.RecordGetOpts) error { o.Headers = nil; return nil })
					case "empty factory":
						options = append(options, ruletypes.WithRecordGetOpts(ruletypes.RecordGetOpts{}))
					case "nil header factory":
						options = append(options, ruletypes.WithRecordGetHeaders(nil))
					}
					options = append(options, func(o *ruletypes.RecordGetOpts) error {
						if o.Headers == nil {
							t.Fatal("nil callback Header map")
						}
						o.Headers["X-Owned"] = "selected"
						retainedHeaders = o.Headers
						applied++
						return nil
					})
					value, err := api.GetRecord(context.Background(), "one", options...)
					if err != nil || value == nil {
						t.Fatal(err, value)
					}
				case "find":
					var options []ruletypes.RecordFindOption
					switch mode {
					case "nil reset":
						options = append(options, func(o *ruletypes.RecordFindOpts) error { o.Headers = nil; return nil })
					case "empty factory":
						options = append(options, ruletypes.WithRecordFindOpts(ruletypes.RecordFindOpts{}))
					case "nil header factory":
						options = append(options, ruletypes.WithRecordFindHeaders(nil))
					}
					options = append(options, func(o *ruletypes.RecordFindOpts) error {
						if o.Headers == nil {
							t.Fatal("nil callback Header map")
						}
						o.Headers["X-Owned"] = "selected"
						retainedHeaders = o.Headers
						applied++
						return nil
					})
					value, err := api.FindRecord(context.Background(), "one", options...)
					if err != nil || value == nil {
						t.Fatal(err, value)
					}
				case "list":
					var options []ruletypes.RecordListOption
					switch mode {
					case "nil reset":
						options = append(options, func(o *ruletypes.RecordListOpts) error { o.Headers = nil; o.Query = nil; return nil })
					case "empty factory":
						options = append(options, ruletypes.WithRecordListOpts(ruletypes.RecordListOpts{}))
					case "nil header factory":
						options = append(options, ruletypes.WithRecordListHeaders(nil))
					}
					options = append(options, func(o *ruletypes.RecordListOpts) error {
						if o.Headers == nil || o.Query == nil {
							t.Fatal("nil callback maps")
						}
						o.Headers["X-Owned"] = "selected"
						o.Query["name"] = json.RawMessage(`"selected"`)
						retainedHeaders = o.Headers
						retainedQuery = o.Query
						applied++
						return nil
					})
					seq := api.ListRecords(context.Background(), options...)
					if applied != 0 || len(calls) != 0 {
						t.Fatal("eager option callback", applied, calls)
					}
					count := 0
					for value, err := range seq {
						if err != nil || value == nil {
							t.Fatal(err, value)
						}
						recordField(t, value, "name", `"selected"`)
						count++
					}
					if count != 1 {
						t.Fatal(count)
					}
				}
				if applied != 1 || len(calls) != 1 || api.RawClient().MoreHeaders != nil {
					t.Fatal(applied, calls, api.RawClient())
				}
			})
		}
	}
}
