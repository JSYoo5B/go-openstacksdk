package regions_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/regions"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func regionRecordAPI(t *testing.T, reply func(*http.Request) *http.Response) (*regions.API, *testcloud.Cloud) {
	t.Helper()
	fixture := testcloud.New(t)
	fixture.Provider.HTTPClient.Transport = nativeRegionTransport(func(r *http.Request) (*http.Response, error) { return reply(r), nil })
	client := fixture.Client("identity", "/catalog/")
	client.ResourceBase = fixture.Server.URL + "/keystone/v3/"
	return regions.New(client), fixture
}
func regionRecordWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"region-proof"}}, Body: io.NopCloser(strings.NewReader(body))}
}
func regionRecordField(t *testing.T, r *regions.RegionRecord, key, want string) {
	t.Helper()
	if r == nil || r.Resource == nil || string(r.Resource.Body[key]) != want {
		t.Fatalf("%s want %s: %+v", key, want, r)
	}
}

type regionRecordMarshaler struct{ calls *int }

func (v regionRecordMarshaler) MarshalJSON() ([]byte, error) {
	*v.calls++
	return nil, errors.New("must not marshal unknown argument")
}

func TestPythonRegionRecordsSourceDictionaryLinksAndPresence(t *testing.T) {
	for _, links := range []string{`,"links":{"next":"https://untrusted.invalid/v3/regions"}`, `,"links":{"next":false},"regions_links":[{"rel":"next","href":"https://untrusted.invalid/regions"}]`, `,"links":null,"regions_links":[{"rel":"next","href":"https://untrusted.invalid/regions"}]`} {
		calls := 0
		api, _ := regionRecordAPI(t, func(*http.Request) *http.Response {
			calls++
			return regionRecordWire(200, `{"regions":[{"id":"a"}]`+links+`}`)
		})
		rows, err := api.AllRecords(context.Background())
		if strings.Contains(links, `"links":null`) {
			var proof *resource.ResponseError
			if err == nil || rows != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-Id") != "region-proof" || string(proof.Body) != `{"regions":[{"id":"a"}]`+links+`}` || calls != 1 {
				t.Fatal(links, rows, err, proof, calls)
			}
		} else if err != nil || len(rows) != 1 || calls != 1 {
			t.Fatal(links, rows, err, calls)
		}
	}
}
func TestPythonRegionRecordsFilteredMarkerRawCapAndBreak(t *testing.T) {
	for _, mode := range []string{"marker", "cap", "break"} {
		t.Run(mode, func(t *testing.T) {
			var markers []string
			api, _ := regionRecordAPI(t, func(r *http.Request) *http.Response {
				markers = append(markers, r.URL.Query().Get("marker"))
				if r.URL.Query().Get("limit") != "5" || r.URL.Query().Has("description") {
					t.Fatal(r.URL)
				}
				if len(markers) == 1 {
					return regionRecordWire(200, `{"regions":[{"id":"a","description":"keep"},{"id":"last-filtered","description":"skip"}],"links":{"next":"ignored.invalid"}}`)
				}
				if len(markers) == 2 {
					return regionRecordWire(200, `{"regions":[{"id":"short","description":"keep"}]}`)
				}
				return regionRecordWire(200, `{"regions":[]}`)
			})
			opts := []regions.RecordListOption{regions.WithRecordListLimit(5), regions.WithRecordListQuery("description", "keep")}
			if mode == "cap" {
				opts = append(opts, regions.WithRecordListMaxItems(2))
			}
			if mode == "break" {
				for row, err := range api.ListRecords(context.Background(), opts...) {
					if err != nil || row == nil {
						t.Fatal(row, err)
					}
					row.Resource.Body["id"] = json.RawMessage(`"caller-mutated"`)
					break
				}
				if !reflect.DeepEqual(markers, []string{""}) {
					t.Fatal(markers)
				}
				return
			}
			rows, err := api.AllRecords(context.Background(), opts...)
			if mode == "cap" {
				if err != nil || len(rows) != 1 || !reflect.DeepEqual(markers, []string{""}) {
					t.Fatal(rows, err, markers)
				}
				return
			}
			if err != nil || len(rows) != 2 || !reflect.DeepEqual(markers, []string{"", "last-filtered", "short"}) {
				t.Fatal(rows, err, markers)
			}
		})
	}
}
func TestPythonRegionRecordsOrderedLoadedLocalEquality(t *testing.T) {
	for _, tc := range []struct {
		actual, filter string
		want           int
	}{{"0.10000000000000001", "0.1", 1}, {"1e-4000", "false", 1}, {"9007199254740993.0", "9007199254740992", 1}, {"9007199254740993.0", "9007199254740993", 0}, {`{"nested":{"a":true,"b":2}}`, `{"nested":{"a":1}}`, 1}, {`"truthy"`, `{}`, 1}, {`0`, `{}`, 0}} {
		api, _ := regionRecordAPI(t, func(*http.Request) *http.Response {
			return regionRecordWire(200, `{"regions":[{"id":"a","description":`+tc.actual+`}]}`)
		})
		rows, err := api.AllRecords(context.Background(), regions.WithRecordListQuery("description", json.RawMessage(tc.filter)))
		if err != nil || len(rows) != tc.want {
			t.Fatal(tc, rows, err)
		}
	}
	api, _ := regionRecordAPI(t, func(*http.Request) *http.Response {
		return regionRecordWire(200, `{"regions":[{"id":"a","description":"skip","name":"d"}]}`)
	})
	rows, err := api.AllRecords(context.Background(), regions.WithRecordListQuery("description", "mismatch"), regions.WithRecordListQuery("id", map[string]any{"x": 1}))
	if err != nil || len(rows) != 0 {
		t.Fatal(rows, err)
	}
	rows, err = api.AllRecords(context.Background(), regions.WithRecordListQuery("id", map[string]any{"x": 1}), regions.WithRecordListQuery("description", "mismatch"))
	var proof *resource.ResponseError
	if err == nil || rows != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-Id") != "region-proof" || string(proof.Body) != `{"regions":[{"id":"a","description":"skip","name":"d"}]}` {
		t.Fatal(rows, err, proof)
	}
}
func TestPythonRegionRecordsProxyExpressionsRemainOpaqueUntilConsumed(t *testing.T) {
	calls := 0
	api, _ := regionRecordAPI(t, func(*http.Request) *http.Response { calls++; return regionRecordWire(200, `{"regions":[{"id":"a"}]}`) })
	for expr, want := range map[string]string{"[0].id": "null", "type(@)": "null", "`[]`": "[]", "`[1,false,null]`": "[1,false,null]"} {
		result, err := api.ProxyListRecords(context.Background(), regions.WithRecordListJMESPath(expr))
		if err != nil || result == nil || string(result.JSON) != want || calls != 0 {
			t.Fatal(expr, result, err, calls)
		}
	}
	result, err := api.ProxyListRecords(context.Background(), regions.WithRecordListJMESPath("@"))
	if err != nil || result.JSON != nil || calls != 0 {
		t.Fatal(result, err, calls)
	}
	generator, ok := result.Value.(*regions.RecordListGenerator)
	if !ok {
		t.Fatal(result)
	}
	for row, err := range generator.Records {
		if err != nil || row == nil {
			t.Fatal(row, err)
		}
	}
	if calls != 1 {
		t.Fatal(calls)
	}
	result, err = api.ProxyListRecords(context.Background(), regions.WithRecordListJMESPath("to_array(@)"))
	if err != nil || result.JSON != nil {
		t.Fatal(result, err)
	}
	array, ok := result.Value.([]any)
	if !ok || len(array) != 1 {
		t.Fatal(result)
	}
	if _, ok := array[0].(*regions.RecordListGenerator); !ok {
		t.Fatal(result)
	}
	before := calls
	for _, expr := range []string{"length(@)", "["} {
		_, err := api.ProxyListRecords(context.Background(), regions.WithRecordListJMESPath(expr))
		if !errors.Is(err, resource.ErrInvalidOption) || calls != before {
			t.Fatal(expr, err, calls)
		}
	}
	for _, raw := range []string{"null", "false", "0", `""`, "true", "[]"} {
		result, err := api.ProxyListRecords(context.Background(), regions.WithRecordListQuery("jmespath_filters", json.RawMessage(raw)))
		if err != nil {
			t.Fatal(raw, err)
		}
		if _, ok := result.Value.(*regions.RecordListGenerator); !ok {
			t.Fatal(raw, result)
		}
	}
	if calls != before {
		t.Fatal(calls)
	}
}
func TestPythonRegionRecordsOwnedOptionsStickySourceAndCancellation(t *testing.T) {
	calls := 0
	api, _ := regionRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.URL.Query().Get("parent_region_id") != "owned" || r.Header.Get("X-Owned") != "owned" {
			t.Fatal(r.URL, r.Header)
		}
		return regionRecordWire(200, `{"regions":[]}`)
	})
	raw := json.RawMessage(`"owned"`)
	headers := map[string]string{"X-Owned": "owned"}
	opts := regions.WithRecordListOpts(regions.RecordListOpts{Query: map[string]json.RawMessage{"parent_region_id": raw}, Headers: headers})
	raw[1] = 'X'
	headers["X-Owned"] = "changed"
	if _, err := api.AllRecords(context.Background(), opts); err != nil || calls != 1 {
		t.Fatal(err, calls)
	}
	for _, mode := range []string{"source", "cancel", "nil"} {
		t.Run(mode, func(t *testing.T) {
			before := calls
			callbacks := 0
			client := api.RawClient()
			endpoint := client.Endpoint
			defer func() { client.Endpoint = endpoint }()
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("custom region cancel")
			change := regions.RecordListOption(func(*regions.RecordListOpts) error {
				if mode == "source" {
					client.Endpoint += "changed"
				} else if mode == "cancel" {
					cancel(cause)
				}
				return nil
			})
			if mode == "nil" {
				change = nil
			}
			restore := regions.RecordListOption(func(*regions.RecordListOpts) error { callbacks++; client.Endpoint = endpoint; return nil })
			_, err := api.AllRecords(ctx, change, restore)
			var proof *resource.ResponseError
			if err == nil || errors.As(err, &proof) || calls != before || callbacks != 0 {
				t.Fatal(err, proof, calls, callbacks)
			}
			if mode == "cancel" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
		})
	}
}
func TestPythonRegionRecordsCleanNativeRetryUsesLiveAuthAndCapturedHeaders(t *testing.T) {
	for _, code := range []int{401, 503, 429, 498} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls, reauth, retry, backoff := 0, 0, 0, 0
			api, fixture := regionRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				if r.Method != "GET" || r.Body != nil || r.Header.Get("X-Snapshot") != "before" || r.Header.Get("X-Late") != "" {
					t.Fatal(r.Header, r.Body)
				}
				if calls == 1 {
					if r.Header.Get("X-Auth-Token") != "test-token" {
						t.Fatal(r.Header)
					}
					return regionRecordWire(code, `{"error":"native clean"}`)
				}
				if r.Header.Get("X-Auth-Token") != "refreshed" {
					t.Fatal(r.Header)
				}
				return regionRecordWire(203, `{"regions":[{"id":"a"}]}`)
			})
			api.RawClient().MoreHeaders = map[string]string{"X-Snapshot": "before"}
			update := func() {
				fixture.Provider.SetToken("refreshed")
				api.RawClient().MoreHeaders["X-Snapshot"] = "after"
				api.RawClient().MoreHeaders["X-Late"] = "after"
			}
			fixture.Provider.ReauthFunc = func(context.Context) error { reauth++; update(); return nil }
			fixture.Provider.RetryFunc = func(ctx context.Context, method, u string, opts *gophercloud.RequestOpts, _ error, count uint) error {
				retry++
				update()
				return nil
			}
			fixture.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				backoff++
				update()
				return nil
			}
			fixture.Provider.MaxBackoffRetries = 1
			rows, err := api.AllRecords(context.Background())
			if err != nil || len(rows) != 1 || rows[0].StatusCode != 203 || calls != 2 {
				t.Fatal(rows, err, calls)
			}
			if code == 401 && (reauth != 1 || retry != 0 || backoff != 0) || code == 503 && (retry != 1 || reauth != 0 || backoff != 0) || (code == 429 || code == 498) && (backoff != 1 || reauth != 0 || retry != 0) {
				t.Fatal(reauth, retry, backoff)
			}
		})
	}
}

func TestPythonRegionRecordsProxyFormalRestorationAndBoundCollision(t *testing.T) {
	calls := 0
	api, _ := regionRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.URL.RawQuery != "parent_region_id=restored" || r.Header.Get("X-Raw") != "restored" || r.Header.Get("OpenStack-API-Version") != "identity 3.7" || r.URL.Path != "/keystone/v3/regions" {
			t.Fatal(r.URL, r.Header)
		}
		return regionRecordWire(200, `{"regions":[{"id":"a"}]}`)
	})
	rows, err := api.AllRecords(context.Background(),
		regions.WithRecordListQuery("parent_region_id", "original"),
		regions.WithRecordListQuery("headers", map[string]string{"X-Raw": "original"}),
		regions.WithRecordListQuery("microversion", "3.1"),
		regions.WithRecordListQuery("__conflicting_attrs", json.RawMessage(`{"parent_region_id":"restored","headers":{"X-Raw":"restored"},"microversion":"3.7","x_unknown":{"n":1},"jmespath_filters":"length(@)"}`)),
	)
	if err != nil || len(rows) != 1 || calls != 1 {
		t.Fatal(rows, err, calls)
	}
	before := calls
	for _, raw := range []string{`{"paginated":false}`, `{"base_path":null}`, `{"resource_type":1}`, `true`} {
		_, err := api.AllRecords(context.Background(), regions.WithRecordListQuery("__conflicting_attrs", json.RawMessage(raw)))
		var proof *resource.ResponseError
		if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || calls != before {
			t.Fatal(raw, err, proof, calls)
		}
	}
	result, err := api.ProxyListRecords(context.Background(), regions.WithRecordListJMESPath("`[]`"), regions.WithRecordListQuery("__conflicting_attrs", json.RawMessage(`{"jmespath_filters":"length(@)","x_unknown":true}`)))
	if err != nil || result == nil || string(result.JSON) != "[]" || calls != before {
		t.Fatal(result, err, calls)
	}
}

func TestPythonRegionRecordsRawDefaultsLocalNameAndIndependentEvidence(t *testing.T) {
	calls, encoded := 0, 0
	body := `{"regions":[{"id":"a","name":"wanted","description":false,"parent_region_id":"root","links":{"self":"wire"},"location":{"cloud":"untrusted"},"x_extension":{"n":1}},{"id":null}]}`
	api, _ := regionRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.URL.Path != "/keystone/v3/regions" || r.URL.RawQuery != "parent_region_id=root" || r.Body != nil || r.Header.Get("Accept") != "application/json" || r.Header.Get("OpenStack-API-Version") != "" {
			t.Fatal(r.URL, r.Body, r.Header)
		}
		return regionRecordWire(203, body)
	})
	seq := api.ListRecords(context.Background(), regions.WithRecordListQuery("parent_region_id", "root"), regions.WithRecordListQuery("name", "wanted"), regions.WithRecordListQuery("description", false), regions.WithRecordListQuery("x_unknown", regionRecordMarshaler{&encoded}))
	if calls != 0 || encoded != 0 {
		t.Fatal(calls, encoded)
	}
	var rows []*regions.RegionRecord
	for row, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if len(rows) != 1 || calls != 1 || encoded != 0 {
		t.Fatal(rows, calls, encoded)
	}
	for key, want := range map[string]string{"id": `"a"`, "name": `"wanted"`, "description": "false", "parent_region_id": `"root"`, "links": `{"self":"wire"}`, "location": "null"} {
		regionRecordField(t, rows[0], key, want)
	}
	if string(rows[0].Wire.Body["x_extension"]) != `{"n":1}` || string(rows[0].Wire.Body["location"]) != `{"cloud":"untrusted"}` || string(rows[0].Envelope) != body || rows[0].StatusCode != 203 || rows[0].Header.Get("X-Request-Id") != "region-proof" {
		t.Fatal(rows[0])
	}
	clone := rows[0].Clone()
	clone.Resource.Body["id"][1] = 'X'
	clone.Header.Set("X-Request-Id", "caller")
	clone.Envelope[0] = '['
	if string(rows[0].Resource.Body["id"]) != `"a"` || rows[0].Header.Get("X-Request-Id") != "region-proof" || string(rows[0].Envelope) != body {
		t.Fatal(rows)
	}
	api, _ = regionRecordAPI(t, func(*http.Request) *http.Response { return regionRecordWire(200, `{"regions":[{}]}`) })
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	for _, key := range []string{"id", "name", "description", "parent_region_id", "links", "location"} {
		regionRecordField(t, rows[0], key, "null")
	}
}
func TestPythonRegionRecordsExplicitLocationStatusAndBooleanLimitBoundary(t *testing.T) {
	name, project := "configured", "current"
	location := resource.CloudLocation{Cloud: &name, Zone: json.RawMessage(`"caller-zone"`), Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &project}}
	option := regions.WithRecordListLocation(location)
	name = "changed"
	project = "changed"
	for _, code := range []int{100, 199, 203, 300, 399} {
		calls := 0
		api, _ := regionRecordAPI(t, func(*http.Request) *http.Response {
			calls++
			return regionRecordWire(code, `{"regions":[{"id":"a","project_id":"foreign","availability_zone":"forged","location":{"cloud":"wire"}}]}`)
		})
		rows, err := api.AllRecords(context.Background(), option)
		if err != nil || len(rows) != 1 || rows[0].StatusCode != code {
			t.Fatal(rows, err)
		}
		var value map[string]json.RawMessage
		_ = json.Unmarshal(rows[0].Resource.Body["location"], &value)
		var scope map[string]json.RawMessage
		_ = json.Unmarshal(value["project"], &scope)
		if string(value["cloud"]) != `"configured"` || string(value["zone"]) != "null" || string(scope["id"]) != `"scope"` || string(scope["name"]) != `"current"` || string(rows[0].Wire.Body["availability_zone"]) != `"forged"` {
			t.Fatal(rows)
		}
		for _, limit := range []bool{true, false} {
			before := calls
			_, err := api.AllRecords(context.Background(), regions.WithRecordListQuery("limit", limit))
			var proof *resource.ResponseError
			if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || calls != before {
				t.Fatal(limit, err, proof, calls)
			}
		}
	}
}

type regionFaultBody struct {
	first             bool
	closes            *int
	readErr, closeErr error
	read, close       func()
}

func (b *regionFaultBody) Read(p []byte) (int, error) {
	if b.first {
		return 0, io.EOF
	}
	b.first = true
	if b.read != nil {
		b.read()
	}
	n := copy(p, `{"regions":`)
	if b.readErr != nil {
		return n, b.readErr
	}
	return n, io.EOF
}
func (b *regionFaultBody) Close() error {
	*b.closes++
	if b.close != nil {
		b.close()
	}
	return b.closeErr
}
func TestPythonRegionRecordsPhysicalFaultsKeepActualProofAndStopRetry(t *testing.T) {
	for _, code := range []int{200, 400, 403, 404, 401, 429, 503} {
		for _, phase := range []string{"read", "close", "source", "cancel"} {
			t.Run(fmt.Sprintf("%d/%s", code, phase), func(t *testing.T) {
				calls, closes, hooks := 0, 0, 0
				cause := errors.New("region physical fault")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				var api *regions.API
				api, fixture := regionRecordAPI(t, func(*http.Request) *http.Response {
					calls++
					b := &regionFaultBody{closes: &closes}
					switch phase {
					case "read":
						b.readErr = cause
					case "close":
						b.closeErr = cause
					case "source":
						b.read = func() { api.RawClient().Endpoint += "changed" }
						b.close = func() { api.RawClient().Endpoint = strings.TrimSuffix(api.RawClient().Endpoint, "changed") }
					case "cancel":
						b.read = func() { cancel(cause) }
					}
					response := regionRecordWire(code, "")
					response.Body = b
					return response
				})
				fixture.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks++
					return nil
				}
				fixture.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error { hooks++; return nil }
				fixture.Provider.ReauthFunc = func(context.Context) error { hooks++; return nil }
				fixture.Provider.MaxBackoffRetries = 1
				row, err := api.AllRecords(ctx)
				if err == nil || row != nil || calls != 1 || closes != 1 || hooks != 0 {
					t.Fatal(row, err, calls, closes, hooks)
				}
				if code < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != code || proof.Header.Get("X-Request-Id") != "region-proof" || string(proof.Body) != `{"regions":` {
						t.Fatal(err, proof)
					}
				} else {
					original := err
					if code == 401 {
						var auth *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &auth) || auth.ErrOriginal == nil || auth.ErrReauth == nil {
							t.Fatal(err, auth)
						}
						original = auth.ErrOriginal
					}
					var proof gophercloud.ErrUnexpectedResponseCode
					if !errors.As(original, &proof) || proof.Actual != code || proof.ResponseHeader.Get("X-Request-Id") != "region-proof" || string(proof.Body) != `{"regions":` || proof.Method != "GET" || !strings.Contains(proof.URL, "/keystone/v3/regions") {
						t.Fatal(err, proof)
					}
				}
				if code != 401 && phase != "source" && !errors.Is(err, cause) {
					t.Fatal(err, cause)
				}
				if code == 401 {
					var reauth *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &reauth) || reauth.ErrReauth == nil {
						t.Fatal(err, reauth)
					}
					// The native 401 wrapper has no Unwrap: its original response
					// and failed reauthentication are independently inspectable.
					// Gophercloud stores ErrOriginal as a response-error value.
					var original gophercloud.ErrUnexpectedResponseCode
					if !errors.As(reauth.ErrOriginal, &original) || original.Actual != 401 || original.Method != "GET" || !strings.Contains(original.URL, "/regions") || string(original.Body) != `{"regions":` || original.ResponseHeader.Get("X-Request-Id") != "region-proof" {
						t.Fatal(err, original)
					}
					if phase == "source" {
						if !errors.Is(reauth.ErrReauth, resource.ErrInvalidOption) {
							t.Fatal(err, reauth.ErrReauth)
						}
					} else if !errors.Is(reauth.ErrReauth, cause) {
						t.Fatal(err, reauth.ErrReauth, cause)
					}
					if phase == "cancel" && !errors.Is(reauth.ErrReauth, context.Canceled) {
						t.Fatal(err, reauth.ErrReauth)
					}
				}
			})
		}
	}
}
