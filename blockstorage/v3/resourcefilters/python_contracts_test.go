package resourcefilters_test

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
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type filterCall struct {
	method, path, query string
	headers             http.Header
}

// All ordinary contracts use the public Gophercloud HTTP fixture through the
// existing testcloud adapter. This decorator only injects physical body faults.
type filterTransport struct {
	base  http.RoundTripper
	after func(*http.Request, *http.Response)
}

func (v filterTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := v.base.RoundTrip(r)
	if response != nil && v.after != nil {
		v.after(r, response)
	}
	return response, err
}
func filterFixture(t *testing.T, version string, handler func(http.ResponseWriter, *http.Request)) (*resourcefilters.API, *[]filterCall) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := &[]filterCall{}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if string(raw) != "" {
			t.Errorf("string(raw) = %v, want %v", string(raw), "")
			return
		}
		if r.Method != http.MethodGet {
			t.Errorf("r.Method = %v, want %v", r.Method, http.MethodGet)
			return
		}
		if got := r.Header.Get("X-Auth-Token"); got != "test-token" {
			t.Errorf("r.Header.Get(\"X-Auth-Token\") = %v, want %v", got, "test-token")
			return
		}
		*calls = append(*calls, filterCall{r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Clone()})
		w.Header().Set("X-Request-Id", "actual-filter")
		handler(w, r)
	})
	client := cloud.Client("volumev3", "/cinder/v3/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	client.Microversion = version
	return resourcefilters.New(client), calls
}
func filterJSON(w http.ResponseWriter, status int, body string) { testcloud.JSON(w, status, body) }
func filterField(t *testing.T, v *resourcefilters.ResourceFilterRecord, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		t.Fatal(key, want, v)
	}
}
func filterWrapper(t *testing.T, err error) {
	t.Helper()
	var op *resource.OperationError
	if !errors.As(err, &op) || op.Operation != "ListRecords" || op.Resource != "resource_filters" {
		t.Fatal(err, op)
	}
}
func TestPythonCinderResourceFiltersDeclaredProjectionOwnsLiteralPageEvidence(t *testing.T) {
	body := `{"resource_filters":[{"id":false,"name":900719925474099312345,"resource":false,"filters":{"field":null},"unknown":[null,0],"self":"passive","location":{"cloud":"untrusted"}}]}`
	api, calls := filterFixture(t, "3.33", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("OpenStack-API-Version"); got != "volume 3.33" {
			t.Errorf("r.Header.Get(\"OpenStack-API-Version\") = %v, want %v", got, "volume 3.33")
			return
		}
		if got := r.Header.Get("X-OpenStack-Volume-API-Version"); got != "3.33" {
			t.Errorf("r.Header.Get(\"X-OpenStack-Volume-API-Version\") = %v, want %v", got, "3.33")
			return
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("r.Header.Get(\"Accept\") = %v, want %v", got, "application/json")
			return
		}
		filterJSON(w, 203, body)
	})
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 1 || len(*calls) != 1 || (*calls)[0].path != "/cinder/v3/project/resource_filters" || (*calls)[0].query != "" {
		t.Fatal(rows, err, *calls)
	}
	v := rows[0]
	for key, want := range map[string]string{"id": "false", "name": "900719925474099312345", "resource": `"False"`, "filters": `[{"field":null}]`, "location": "null"} {
		filterField(t, v, key, want)
	}
	if len(v.Resource.Body) != 5 || v.Resource.Body["unknown"] != nil || string(v.Wire.Body["unknown"]) != "[null,0]" || string(v.Wire.Body["resource"]) != "false" || string(v.Envelope) != body || v.StatusCode != 203 || v.Microversion != "3.33" || len(v.Discovery) != 0 {
		t.Fatal(v)
	}
	v.Resource.Body["filters"][0] = 'X'
	v.Header.Set("X-Request-Id", "changed")
	v.Envelope[0] = 'X'
	if v.Wire.Header.Get("X-Request-Id") != "actual-filter" || v.Resource.Header.Get("X-Request-Id") != "actual-filter" || string(v.Wire.Body["filters"]) != `{"field":null}` {
		t.Fatal(v)
	}
}
func TestPythonCinderResourceFiltersSourceQueryExpansionLocalPredicatesAndUnknownDiscard(t *testing.T) {
	for _, query := range []map[string]any{{"resource": "volume", "filters": []any{true}, "id": 1}, {"resource": []any{nil, "volume", false}, "filters": []any{true}, "id": true}, {"resource": nil, "filters": []any{true}, "id": true}} {
		marshaled := 0
		query["unknown"] = filterMarshaler{&marshaled, `broken`}
		query["limit"] = filterMarshaler{&marshaled, `broken`}
		query["marker"] = filterMarshaler{&marshaled, `broken`}
		api, calls := filterFixture(t, "3.33", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Has("id") || r.URL.Query().Has("filters") || r.URL.Query().Has("limit") || r.URL.Query().Has("marker") {
				t.Fatal(r.URL)
			}
			filterJSON(w, 200, `{"resource_filters":[{"id":true,"resource":"volume","filters":[1]},{"id":false,"resource":"snapshot","filters":[0]}]}`)
		})
		rows, err := api.AllRecords(context.Background(), resourcefilters.WithRecordListQuery(query))
		if err != nil || len(rows) != 1 || len(*calls) != 1 || marshaled != 0 {
			t.Fatal(rows, err, *calls, marshaled)
		}
		filterField(t, rows[0], "id", "true")
		raw, _ := json.Marshal(query["resource"])
		var expected []string
		switch string(raw) {
		case `"volume"`:
			expected = []string{"volume"}
		case `[null,"volume",false]`:
			expected = []string{"volume", "False"}
		}
		parsed, _ := url.ParseQuery((*calls)[0].query)
		if !reflect.DeepEqual(parsed["resource"], expected) {
			t.Fatal(parsed, expected)
		}
	}
}
func TestPythonCinderResourceFiltersDefaultsNullableListAndPythonStrings(t *testing.T) {
	for _, row := range []string{`{}`, `{"resource":null,"filters":null}`, `{"resource":900719925474099312345,"filters":false}`, `{"resource":{"ordered":true},"filters":"whole"}`} {
		api, calls := filterFixture(t, "3.33", func(w http.ResponseWriter, _ *http.Request) { filterJSON(w, 200, `{"resource_filters":`+row+`}`) })
		values, err := api.AllRecords(context.Background())
		if err != nil || len(values) != 1 || len(*calls) != 1 {
			t.Fatal(values, err, *calls)
		}
		if row == `{}` || row == `{"resource":null,"filters":null}` {
			filterField(t, values[0], "resource", "null")
			filterField(t, values[0], "filters", "null")
		}
		if strings.Contains(row, "900719") {
			filterField(t, values[0], "resource", `"900719925474099312345"`)
			filterField(t, values[0], "filters", "[false]")
		}
		if strings.Contains(row, "ordered") {
			filterField(t, values[0], "resource", `"{'ordered': True}"`)
			filterField(t, values[0], "filters", `["whole"]`)
		}
	}
}
func TestPythonCinderResourceFiltersDiscoveryCeilingAndExplicitVersionOmission(t *testing.T) {
	for _, version := range []string{"", "3.90"} {
		api, calls := filterFixture(t, version, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/cinder/v3/" {
				if r.Header.Get("OpenStack-API-Version") != "" {
					t.Fatal(r.Header)
				}
				filterJSON(w, 200, `{"version":{"id":"v3.0","min_version":"3.0","version":"3.71"}}`)
				return
			}
			filterJSON(w, 200, `{"resource_filters":[{"resource":"volume"}]}`)
		})
		values, err := api.AllRecords(context.Background())
		n := 1
		d := 0
		expected := version
		if version == "" {
			n = 2
			d = 1
			expected = "3.33"
		}
		if err != nil || len(values) != 1 || len(*calls) != n || len(values[0].Discovery) != d || values[0].Microversion != expected || api.RawClient().Microversion != version {
			t.Fatal(values, err, *calls)
		}
	}
	api, calls := filterFixture(t, "3.90", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("OpenStack-API-Version") != "" || r.Header.Get("X-OpenStack-Volume-API-Version") != "" {
			t.Fatal(r.Header)
		}
		filterJSON(w, 200, `{"resource_filters":[{}]}`)
	})
	values, err := api.AllRecords(context.Background(), resourcefilters.WithRecordListMicroversion(""))
	if err != nil || len(values) != 1 || len(*calls) != 1 || len(values[0].Discovery) != 0 || api.RawClient().Microversion != "3.90" {
		t.Fatal(values, err, *calls)
	}
	api, calls = filterFixture(t, "3.32", func(w http.ResponseWriter, _ *http.Request) { filterJSON(w, 404, `{"version":"not registered"}`) })
	values, err = api.AllRecords(context.Background())
	if values != nil || len(*calls) != 1 || !gophercloud.ResponseCodeIs(err, 404) || errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(values, err, *calls)
	}
}
func TestPythonCinderResourceFiltersLocalCapStopsBeforeInvalidRowsAndNext(t *testing.T) {
	api, calls := filterFixture(t, "3.33", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "1" {
			t.Fatal(r.URL)
		}
		filterJSON(w, 200, `{"resource_filters":[{"resource":"volume","filters":[false]},null],"next":{"malformed":true}}`)
	})
	rows, err := api.AllRecords(context.Background(), resourcefilters.WithRecordListQuery(map[string]any{"filters": []any{true}}), resourcefilters.WithRecordListControl(resource.ListControl{MaxItems: 1}))
	if err != nil || len(rows) != 0 || len(*calls) != 1 {
		t.Fatal(rows, err, *calls)
	}
	api, calls = filterFixture(t, "3.33", func(w http.ResponseWriter, _ *http.Request) {
		filterJSON(w, 200, `{"resource_filters":[{}],"next":false}`)
	})
	seq := api.ListRecords(context.Background())
	if len(*calls) != 0 {
		t.Fatal(*calls)
	}
	for range seq {
		break
	}
	for range seq {
		break
	}
	if len(*calls) != 2 {
		t.Fatal(*calls)
	}
}
func TestPythonCinderResourceFiltersPaginationPreservesFiltersAndBlocksForeignOrCycle(t *testing.T) {
	for _, mode := range []string{"valid", "foreign", "cycle", "empty", "underflow"} {
		api, calls := filterFixture(t, "3.33", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("resource") != "volume" {
				t.Fatal(r.URL)
			}
			if r.URL.Query().Has("marker") {
				filterJSON(w, 200, `{"resource_filters":[{"id":"second","resource":"volume"}]}`)
				return
			}
			next := `"?marker=after"`
			row := `[{"id":"first","resource":"volume"}]`
			switch mode {
			case "foreign":
				next = `"https://foreign.invalid/resource_filters?marker=after"`
			case "cycle":
				next = `"?resource=volume"`
			case "empty":
				row = `[]`
			case "underflow":
				next = `1e-400`
			}
			filterJSON(w, 200, `{"resource_filters":`+row+`,"next":`+next+`}`)
		})
		rows, err := api.AllRecords(context.Background(), resourcefilters.WithRecordListResource("volume"))
		switch mode {
		case "valid":
			if err != nil || len(rows) != 2 || len(*calls) != 2 {
				t.Fatal(rows, err, *calls)
			}
		case "empty":
			if err != nil || len(rows) != 0 || len(*calls) != 1 {
				t.Fatal(rows, err, *calls)
			}
		case "underflow":
			if err != nil || len(rows) != 1 || len(*calls) != 1 {
				t.Fatal(rows, err, *calls)
			}
		default:
			if rows != nil || err == nil || len(*calls) != 1 {
				t.Fatal(mode, rows, err, *calls)
			}
		}
	}
	api, calls := filterFixture(t, "3.33", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("marker") {
			filterJSON(w, 200, `{"resource_filters":[{"resource":"snapshot"}]}`)
		} else {
			filterJSON(w, 200, `{"resource_filters":[{"resource":"volume"}],"resource_filters_links":[{"rel":"next","href":"?marker=after"}]}`)
		}
	})
	rows, err := api.AllRecords(context.Background(), resourcefilters.WithRecordListControl(resource.ListControl{SinglePage: true}))
	if err != nil || len(rows) != 1 || len(*calls) != 1 {
		t.Fatal(rows, err, *calls)
	}
}
func TestPythonCinderResourceFiltersBadEnvelopeAndConstructorCollisionKeepAcceptedPage(t *testing.T) {
	for _, body := range []string{"", `null`, `{}`, `{"resource_filters":null}`, `{"resource_filters":[null]}`, `{"resource_filters":[{"connection":false}]}`, `{"resource_filters":[{"microversion":"foreign"}]}`} {
		api, calls := filterFixture(t, "3.33", func(w http.ResponseWriter, _ *http.Request) { filterJSON(w, 203, body) })
		rows, err := api.AllRecords(context.Background())
		var proof *resource.ResponseError
		if rows != nil || err == nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "actual-filter" || len(*calls) != 1 {
			t.Fatal(body, rows, err, proof, *calls)
		}
		filterWrapper(t, err)
	}
}

type filterMarshaler struct {
	called *int
	body   string
}

func (v filterMarshaler) MarshalJSON() ([]byte, error) { *v.called++; return []byte(v.body), nil }
func TestPythonCinderResourceFiltersControlsBeforeMarshalerAndOptionSnapshots(t *testing.T) {
	for _, key := range []string{"cls", "self", "session", "base_path", "list_base_path", "microversion", "headers", "__conflicting_attrs", "max_items", "paginated", "allow_unknown_params"} {
		calls := 0
		api, wire := filterFixture(t, "3.33", func(http.ResponseWriter, *http.Request) { t.Fatal("HTTP") })
		_, err := api.AllRecords(context.Background(), resourcefilters.WithRecordListQuery(map[string]any{key: filterMarshaler{&calls, `null`}, "resource": filterMarshaler{&calls, `"volume"`}}))
		if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || len(*wire) != 0 {
			t.Fatal(key, err, calls, *wire)
		}
	}
	query := map[string]any{"resource": []any{"volume", nil, false}}
	option := resourcefilters.WithRecordListQuery(query)
	query["resource"].([]any)[0] = "changed"
	var retained map[string]string
	api, calls := filterFixture(t, "3.33", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Owned") != "before" || r.URL.RawQuery != "resource=volume&resource=False" {
			t.Fatal(r.Header, r.URL)
		}
		filterJSON(w, 200, `{"resource_filters":[]}`)
	})
	_, err := api.AllRecords(context.Background(), option, func(c *resourcefilters.RecordListOpts) error { c.Headers = nil; return nil }, func(c *resourcefilters.RecordListOpts) error {
		c.Headers["X-Owned"] = "before"
		retained = c.Headers
		return nil
	}, func(*resourcefilters.RecordListOpts) error { retained["X-Owned"] = "after"; return nil })
	if err != nil || len(*calls) != 1 || api.RawClient().MoreHeaders != nil {
		t.Fatal(err, *calls)
	}
	for _, option := range []resourcefilters.RecordListOption{nil, resourcefilters.WithRecordListMicroversion("bad"), resourcefilters.WithRecordListHeaders(map[string]string{"Authorization": "forged"}), resourcefilters.WithRecordListControl(resource.ListControl{MaxItems: -1})} {
		api, calls := filterFixture(t, "3.33", func(http.ResponseWriter, *http.Request) { t.Fatal("HTTP") })
		_, err := api.AllRecords(context.Background(), option)
		if !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(err, *calls)
		}
	}
}
func TestPythonCinderResourceFiltersListResultSupportsArbitraryJMESPathWithoutFakeProof(t *testing.T) {
	for _, expression := range []string{"length(@)", "[].filters[]", "[0].resource", "`null`", ""} {
		api, calls := filterFixture(t, "3.33", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Has("marker") {
				filterJSON(w, 200, `{"resource_filters":[{"resource":"snapshot","filters":["status"]}]}`)
			} else {
				filterJSON(w, 200, `{"resource_filters":[{"resource":"volume","filters":["name"]}],"next":"?marker=next"}`)
			}
		})
		result, err := api.ListResult(context.Background(), resourcefilters.WithRecordListJMESPath(expression))
		if err != nil || len(result.Records) != 2 || len(*calls) != 2 || result.Expression != (expression != "") {
			t.Fatal(result, err, *calls)
		}
		want := ""
		switch expression {
		case "length(@)":
			want = "2"
		case "[].filters[]":
			want = `["name","status"]`
		case "[0].resource":
			want = `"volume"`
		case "`null`":
			want = "null"
		}
		if want != "" && string(result.Value) != want {
			t.Fatal(expression, result)
		}
		if result.Records[0].StatusCode != 200 || result.Records[1].Header.Get("X-Request-Id") != "actual-filter" {
			t.Fatal(result)
		}
	}
	api, calls := filterFixture(t, "3.33", func(w http.ResponseWriter, _ *http.Request) { filterJSON(w, 200, `{"resource_filters":[]}`) })
	_, err := api.AllRecords(context.Background(), resourcefilters.WithRecordListJMESPath("length(@)"))
	if !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
		t.Fatal(err, *calls)
	}
	result, err := api.ListResult(context.Background(), resourcefilters.WithRecordListJMESPath("["))
	if err == nil || result.Value != nil || len(*calls) != 1 {
		t.Fatal(result, err, *calls)
	}
}

type filterFaultBody struct {
	io.ReadCloser
	readErr, closeErr     error
	afterRead, afterClose func()
	closes                *atomic.Int32
}

func (b *filterFaultBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if b.afterRead != nil {
		b.afterRead()
	}
	if b.readErr != nil {
		return n, b.readErr
	}
	return n, err
}
func (b *filterFaultBody) Close() error {
	if b.closes != nil {
		b.closes.Add(1)
	}
	if b.afterClose != nil {
		b.afterClose()
	}
	return errors.Join(b.ReadCloser.Close(), b.closeErr)
}
func TestPythonCinderResourceFiltersPhysicalFaults401FieldsAndHooksStayTerminal(t *testing.T) {
	for _, status := range []int{200, 203, 401, 404, 429, 503} {
		for _, mode := range []string{"read", "close", "both"} {
			t.Run(fmt.Sprintf("%d/%s", status, mode), func(t *testing.T) {
				body := `{"resource_filters":[{"resource":"volume"}]}`
				readCause, closeCause := errors.New("read actual"), errors.New("close actual")
				var closes atomic.Int32
				api, calls := filterFixture(t, "3.33", func(w http.ResponseWriter, _ *http.Request) { filterJSON(w, status, body) })
				provider := api.RawClient().ProviderClient
				provider.HTTPClient.Transport = filterTransport{base: provider.HTTPClient.Transport, after: func(_ *http.Request, r *http.Response) {
					b := &filterFaultBody{ReadCloser: r.Body, closes: &closes}
					if mode != "close" {
						b.readErr = readCause
					}
					if mode != "read" {
						b.closeErr = closeCause
					}
					r.Body = b
				}}
				retries, backoff, renew := 0, 0, 0
				provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
					backoff++
					return nil
				}
				provider.ReauthFunc = func(context.Context) error { renew++; return nil }
				rows, err := api.AllRecords(context.Background())
				cause := err
				if status == 401 {
					var auth *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &auth) || auth.ErrOriginal == nil || auth.ErrReauth == nil {
						t.Fatal(err, auth)
					}
					cause = errors.Join(auth.ErrOriginal, auth.ErrReauth)
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(auth.ErrOriginal, &native) || native.Actual != 401 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "actual-filter" {
						t.Fatal(auth, native)
					}
				}
				if err == nil || rows != nil || len(*calls) != 1 || closes.Load() != 1 || retries != 0 || backoff != 0 || renew != 0 || mode != "close" && !errors.Is(cause, readCause) || mode != "read" && !errors.Is(cause, closeCause) {
					t.Fatal(rows, err, cause, *calls, closes.Load(), retries, backoff, renew)
				}
				if status < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body {
						t.Fatal(err, proof)
					}
				} else if status != 401 {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != status || string(native.Body) != body {
						t.Fatal(err, native)
					}
				}
				filterWrapper(t, err)
			})
		}
	}
}
func TestPythonCinderResourceFiltersCleanRenewalAndRetryPreserveQueryAndProtocol(t *testing.T) {
	for _, status := range []int{401, 503} {
		attempts := 0
		api, calls := filterFixture(t, "3.33", func(w http.ResponseWriter, _ *http.Request) {
			attempts++
			if attempts == 1 {
				filterJSON(w, status, `{"retry":"actual"}`)
			} else {
				filterJSON(w, 200, `{"resource_filters":[{}]}`)
			}
		})
		hooks := 0
		provider := api.RawClient().ProviderClient
		if status == 401 {
			provider.ReauthFunc = func(context.Context) error { hooks++; return nil }
		} else {
			provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				hooks++
				return nil
			}
		}
		rows, err := api.AllRecords(context.Background(), resourcefilters.WithRecordListResource("volume"))
		if err != nil || len(rows) != 1 || len(*calls) != 2 || hooks != 1 || (*calls)[0].query != "resource=volume" || (*calls)[1].query != (*calls)[0].query || (*calls)[1].headers.Get("OpenStack-API-Version") != "volume 3.33" {
			t.Fatal(rows, err, *calls, hooks)
		}
	}
}
func TestPythonCinderResourceFiltersReadGuardAndCancellationSurviveCloseRestore(t *testing.T) {
	for _, mode := range []string{"source", "cancel"} {
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		cause := errors.New("read cancellation")
		var closes atomic.Int32
		api, calls := filterFixture(t, "3.33", func(w http.ResponseWriter, _ *http.Request) { filterJSON(w, 203, `{"resource_filters":[{}]}`) })
		provider := api.RawClient().ProviderClient
		provider.HTTPClient.Transport = filterTransport{base: provider.HTTPClient.Transport, after: func(_ *http.Request, r *http.Response) {
			r.Body = &filterFaultBody{ReadCloser: r.Body, closes: &closes, afterRead: func() {
				if mode == "source" {
					api.RawClient().Microversion = "3.12"
				} else {
					cancel(cause)
				}
			}, afterClose: func() { api.RawClient().Microversion = "3.33" }}
		}}
		rows, err := api.AllRecords(ctx)
		var proof *resource.ResponseError
		if rows != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || len(*calls) != 1 || closes.Load() != 1 || mode == "source" && !errors.Is(err, resource.ErrInvalidOption) || mode == "cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
			t.Fatal(rows, err, proof, *calls)
		}
	}
}
func TestPythonCinderResourceFiltersRetryCannotChangeBodyVersionOrHeaderAbsence(t *testing.T) {
	for _, mode := range []string{"body", "version", "foreign", "auth", "omit"} {
		api, calls := filterFixture(t, "3.33", func(w http.ResponseWriter, _ *http.Request) { filterJSON(w, 503, `{"retry":"actual"}`) })
		api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			switch mode {
			case "body":
				o.JSONBody = map[string]bool{"forged": true}
			case "version":
				o.MoreHeaders["OpenStack-API-Version"] = "volume 3.12"
			case "foreign":
				o.MoreHeaders["X-OpenStack-Nova-API-Version"] = "2.90"
			case "auth":
				o.MoreHeaders["Authorization"] = "forged"
			case "omit":
				o.OmitHeaders = append(o.OmitHeaders, "X-OpenStack-Volume-API-Version")
			}
			return nil
		}
		rows, err := api.AllRecords(context.Background())
		if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 1 || !gophercloud.ResponseCodeIs(err, 503) {
			t.Fatal(mode, rows, err, *calls)
		}
	}
}
