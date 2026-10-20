package endpoints_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/testhelper"
)

func endpointRecordsAPI(t *testing.T, handler func(http.ResponseWriter, *http.Request)) (*endpoints.API, *testcloud.Cloud, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := &atomic.Int32{}
	cloud.Mux.HandleFunc("/keystone/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testhelper.TestMethod(t, r, http.MethodGet)
		testhelper.TestHeader(t, r, "X-Auth-Token", "test-token")
		w.Header().Set("X-Request-Id", "endpoint-actual")
		handler(w, r)
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return endpoints.New(client), cloud, calls
}
func endpointRecordField(t *testing.T, value *endpoints.EndpointRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatal(key, want, value)
	}
}
func endpointRecordProof(t *testing.T, err error, status int, body string) {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "endpoint-actual" || proof.Cause == nil {
		t.Fatal(err, proof)
	}
}

func TestEndpointRecordsSourceMRODefaultsUntypedLinksAndActualEvidence(t *testing.T) {
	body := `{"endpoints":[{"id":null,"name":"nova","interface":"public","enabled":"false","links":[null,1,{"unknown":9007199254740993}],"region_id":"RegionOne","service_id":"svc","url":"literal","unknown":"wire","region":"unmodeled","self":"wire-only","location":{"spoof":true}},{}]}`
	api, _, calls := endpointRecordsAPI(t, func(w http.ResponseWriter, r *http.Request) {
		testhelper.AssertEquals(t, "/keystone/v3/endpoints", r.URL.Path)
		testhelper.AssertEquals(t, "", r.URL.RawQuery)
		testhelper.TestHeader(t, r, "Accept", "application/json")
		testhelper.AssertEquals(t, "", r.Header.Get("OpenStack-API-Version"))
		w.Header().Set("ETag", "actual")
		testcloud.JSON(w, 203, body)
	})
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 2 || calls.Load() != 1 {
		t.Fatal(rows, err, calls.Load())
	}
	for key, want := range map[string]string{"id": "null", "name": `"nova"`, "interface": `"public"`, "is_enabled": "true", "links": `[null,1,{"unknown":9007199254740993}]`, "region_id": `"RegionOne"`, "service_id": `"svc"`, "url": `"literal"`, "location": "null"} {
		endpointRecordField(t, rows[0], key, want)
	}
	if len(rows[0].Resource.Body) != 9 || string(rows[0].Envelope) != body || rows[0].StatusCode != 203 || rows[0].Header.Get("ETag") != "actual" || string(rows[0].Wire.Body["enabled"]) != `"false"` || string(rows[0].Wire.Body["self"]) != `"wire-only"` {
		t.Fatal(rows)
	}
	for _, key := range []string{"id", "name", "interface", "is_enabled", "links", "region_id", "service_id", "url", "location"} {
		endpointRecordField(t, rows[1], key, "null")
	}
	rows[0].Resource.Body["links"][0] = 'X'
	rows[0].Header.Set("ETag", "consumer")
	rows[0].Envelope[0] = 'X'
	if string(rows[0].Wire.Body["links"]) != `[null,1,{"unknown":9007199254740993}]` || rows[0].Wire.Header.Get("ETag") != "actual" || rows[0].Resource.Header.Get("ETag") != "actual" {
		t.Fatal(rows[0])
	}
}

func TestEndpointRecordsOrderedEnabledAliasesAndPythonBool(t *testing.T) {
	for _, tc := range []struct{ row, want, links string }{
		{`{"enabled":true,"is_enabled":false,"links":false}`, "false", "false"},
		{`{"is_enabled":false,"enabled":"no","links":{"next":"literal"}}`, "true", `{"next":"literal"}`},
		{`{"enabled":0e9999,"links":"literal"}`, "false", `"literal"`},
		{`{"enabled":1e-9999,"links":null}`, "false", "null"},
		{`{"enabled":[],"links":[]}`, "false", "[]"},
		{`{"enabled":null}`, "null", "null"},
	} {
		t.Run(tc.row, func(t *testing.T) {
			api, _, calls := endpointRecordsAPI(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, 200, `{"endpoints":[`+tc.row+`]}`) })
			rows, err := api.AllRecords(context.Background())
			if err != nil || len(rows) != 1 || calls.Load() != 1 {
				t.Fatal(rows, err, calls.Load())
			}
			endpointRecordField(t, rows[0], "is_enabled", tc.want)
			endpointRecordField(t, rows[0], "links", tc.links)
		})
	}
}

func TestEndpointRecordsQueriesLocalBodyFiltersAndUnknownKwargs(t *testing.T) {
	api, _, calls := endpointRecordsAPI(t, func(w http.ResponseWriter, r *http.Request) {
		want := url.Values{"interface": {"public"}, "region_id": {"RegionOne"}, "service_id": {"svc"}, "limit": {"0"}, "marker": {"start"}}
		if !reflect.DeepEqual(r.URL.Query(), want) {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"endpoints":[{"id":"keep","name":"nova","enabled":1,"links":{"nested":{"flag":1,"extra":true},"n":9007199254740992},"url":"literal"},{"id":"skip","name":"nova","enabled":false,"links":{},"url":"literal"}]}`)
	})
	rows, err := api.AllRecords(context.Background(), endpoints.WithRecordListQuery("interface", "public"), endpoints.WithRecordListQuery("region_id", "RegionOne"), endpoints.WithRecordListQuery("service_id", "svc"), endpoints.WithRecordListLimit(0), endpoints.WithRecordListMarker("start"), endpoints.WithRecordListQuery("id", "keep"), endpoints.WithRecordListQuery("name", "nova"), endpoints.WithRecordListQuery("is_enabled", true), endpoints.WithRecordListQuery("url", "literal"), endpoints.WithRecordListQuery("links", json.RawMessage(`{"nested":{"flag":true},"n":9007199254740993.0}`)), endpoints.WithRecordListQuery("enabled", make(chan int)), endpoints.WithRecordListQuery("region", make(chan int)), endpoints.WithRecordListQuery("unknown", make(chan int)))
	if err != nil || len(rows) != 1 || calls.Load() != 1 {
		t.Fatal(rows, err, calls.Load())
	}
	endpointRecordField(t, rows[0], "id", `"keep"`)
	empty, _, count := endpointRecordsAPI(t, func(w http.ResponseWriter, _ *http.Request) {
		testcloud.JSON(w, 200, `{"endpoints":[{"links":{}},{"links":{"nonempty":true}},{"links":null}]}`)
	})
	rows, err = empty.AllRecords(context.Background(), endpoints.WithRecordListQuery("links", map[string]any{}))
	if err != nil || len(rows) != 1 || count.Load() != 1 {
		t.Fatal(rows, err, count.Load())
	}
	endpointRecordField(t, rows[0], "links", `{"nonempty":true}`)
}

func TestEndpointRecordsSourceDictionaryLinksAndMarkerFallback(t *testing.T) {
	var queries []string
	api, _, calls := endpointRecordsAPI(t, func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		switch r.URL.Query().Get("marker") {
		case "one":
			testcloud.JSON(w, 200, `{"endpoints":[{"id":"two"}],"links":{"next":"https://foreign.invalid/ignored"}}`)
		case "two":
			testcloud.JSON(w, 200, `{"endpoints":[]}`)
		default:
			testcloud.JSON(w, 203, `{"endpoints":[{"id":"one"}],"links":{"next":"https://foreign.invalid/ignored"},"endpoints_links":[{"rel":"next","href":"?marker=wrong"}]}`)
		}
	})
	rows, err := api.AllRecords(context.Background(), endpoints.WithRecordListLimit(1))
	if err != nil || len(rows) != 2 || calls.Load() != 3 || !reflect.DeepEqual(queries, []string{"limit=1", "limit=1&marker=one", "limit=1&marker=two"}) {
		t.Fatal(rows, err, calls.Load(), queries)
	}
	first, _, count := endpointRecordsAPI(t, func(w http.ResponseWriter, _ *http.Request) {
		testcloud.JSON(w, 200, `{"endpoints":[{"id":"one"}],"links":{"next":"?marker=ignored"}}`)
	})
	rows, err = first.AllRecords(context.Background())
	if err != nil || len(rows) != 1 || count.Load() != 1 {
		t.Fatal(rows, err, count.Load())
	}
}

func TestEndpointRecordsSourceLinkPrecedenceAndMalformedPageReceipt(t *testing.T) {
	for _, tc := range []struct {
		name, fields, marker string
		fail                 bool
	}{
		{"root shadows plural", `"links":[],"endpoints_links":[{"rel":"next","href":"?marker=wrong"}]`, "", false},
		{"plural list", `"endpoints_links":[{"rel":"next","href":"/v3/endpoints?marker=plural"}]`, "plural", false},
		{"top next after ignored dict", `"links":{"next":"?marker=wrong"},"next":"?marker=top"`, "top", false},
		{"root first wins", `"links":[{"rel":"next","href":"?marker=first"},{"rel":"next","href":"?marker=second"}],"next":true`, "first", false},
		{"null iterable", `"links":null`, "", true},
		{"foreign continuation", `"next":"https://foreign.invalid/endpoints?marker=next"`, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"endpoints":[{"id":"one"}],` + tc.fields + `}`
			api, _, calls := endpointRecordsAPI(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("marker") == "" {
					testcloud.JSON(w, 203, body)
					return
				}
				testhelper.AssertEquals(t, tc.marker, r.URL.Query().Get("marker"))
				testcloud.JSON(w, 200, `{"endpoints":[]}`)
			})
			rows, err := api.AllRecords(context.Background())
			if tc.fail {
				if rows != nil || err == nil || calls.Load() != 1 {
					t.Fatal(rows, err, calls.Load())
				}
				endpointRecordProof(t, err, 203, body)
				return
			}
			want := int32(1)
			if tc.marker != "" {
				want = 2
			}
			if err != nil || len(rows) != 1 || calls.Load() != want {
				t.Fatal(rows, err, calls.Load())
			}
		})
	}
}

func TestEndpointRecordsRawCapLazyBreakAndCancellation(t *testing.T) {
	api, _, calls := endpointRecordsAPI(t, func(w http.ResponseWriter, r *http.Request) {
		testhelper.AssertEquals(t, "2", r.URL.Query().Get("limit"))
		testcloud.JSON(w, 203, `{"endpoints":[{"id":"skip"},{"id":"keep"},{"connection":null}],"next":"https://foreign.invalid/unused"}`)
	})
	rows, err := api.AllRecords(context.Background(), endpoints.WithRecordListMaxItems(2), endpoints.WithRecordListQuery("id", "keep"))
	if err != nil || len(rows) != 1 || calls.Load() != 1 {
		t.Fatal(rows, err, calls.Load())
	}
	for _, lane := range []string{"break", "paginated", "cancel"} {
		t.Run(lane, func(t *testing.T) {
			next, _, count := endpointRecordsAPI(t, func(w http.ResponseWriter, _ *http.Request) {
				testcloud.JSON(w, 200, `{"endpoints":[{"id":"one"}],"next":"?marker=next"}`)
			})
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("consumer cancel")
			options := []endpoints.RecordListOption{}
			if lane == "paginated" {
				options = append(options, endpoints.WithRecordListPaginated(false))
			}
			seq := next.ListRecords(ctx, options...)
			if count.Load() != 0 {
				t.Fatal("eager HTTP")
			}
			n := 0
			var failure error
			for _, err := range seq {
				if err != nil {
					failure = err
					break
				}
				n++
				if lane == "break" {
					break
				}
				if lane == "cancel" {
					cancel(cause)
				}
			}
			if n != 1 || count.Load() != 1 || (lane == "cancel" && !errors.Is(failure, cause)) || (lane != "cancel" && failure != nil) {
				t.Fatal(n, count.Load(), failure)
			}
		})
	}
}

func TestEndpointRecordsMalformedAndLaterErrorDiscardPartialCollection(t *testing.T) {
	for _, body := range []string{`{}`, `{"endpoints":null}`, `{"endpoints":[null]}`, `{"endpoints":[{"connection":null}]}`, `{"endpoints":[{"microversion":null}]}`, `{"endpoints":[{"id":null}]}`} {
		api, _, calls := endpointRecordsAPI(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, 203, body) })
		rows, err := api.AllRecords(context.Background(), endpoints.WithRecordListLimit(1))
		if rows != nil || err == nil || calls.Load() != 1 {
			t.Fatal(body, rows, err, calls.Load())
		}
		endpointRecordProof(t, err, 203, body)
	}
	late, _, calls := endpointRecordsAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("marker") == "late" {
			testcloud.JSON(w, 403, `{"error":"late"}`)
			return
		}
		testcloud.JSON(w, 200, `{"endpoints":[{"id":"first"}],"next":"?marker=late"}`)
	})
	rows, err := late.AllRecords(context.Background())
	if rows != nil || !gophercloud.ResponseCodeIs(err, 403) || calls.Load() != 2 {
		t.Fatal(rows, err, calls.Load())
	}
}

func TestEndpointRecordsOpaqueProxyJMESUsesActualGenerator(t *testing.T) {
	api, _, calls := endpointRecordsAPI(t, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"endpoints":[{"id":"one"}]}`) })
	for _, tc := range []struct{ expression, want string }{{"[].id", "null"}, {"[0]", "null"}, {"type(@)", "null"}, {"@ == @", "true"}, {"`{\"literal\":true}`", `{"literal":true}`}} {
		value, err := api.ProxyListRecords(context.Background(), endpoints.WithRecordListJMESPath(tc.expression))
		if err != nil || value == nil || string(value.JSON) != tc.want || calls.Load() != 0 {
			t.Fatal(tc, value, err, calls.Load())
		}
	}
	for _, expression := range []string{"length(@)", "to_string(@)"} {
		value, err := api.ProxyListRecords(context.Background(), endpoints.WithRecordListJMESPath(expression))
		if err == nil || value != nil || calls.Load() != 0 {
			t.Fatal(value, err, calls.Load())
		}
	}
	value, err := api.ProxyListRecords(context.Background(), endpoints.WithRecordListJMESPath("to_array(@)"))
	if err != nil || value == nil || value.JSON != nil || calls.Load() != 0 {
		t.Fatal(value, err, calls.Load())
	}
	items, ok := value.Value.([]any)
	if !ok || len(items) != 1 {
		t.Fatal(value.Value)
	}
	generator, ok := items[0].(*endpoints.RecordListGenerator)
	if !ok {
		t.Fatal(items)
	}
	if _, err := json.Marshal(generator); err == nil || calls.Load() != 0 {
		t.Fatal(err, calls.Load())
	}
	n := 0
	for row, err := range generator.Records {
		if err != nil {
			t.Fatal(err)
		}
		endpointRecordField(t, row, "id", `"one"`)
		n++
	}
	if n != 1 || calls.Load() != 1 {
		t.Fatal(n, calls.Load())
	}
	value, err = api.ProxyListRecords(context.Background(), endpoints.WithRecordListQuery("jmespath_filters", true))
	if err != nil || value == nil || value.JSON != nil || calls.Load() != 1 {
		t.Fatal(value, err, calls.Load())
	}
}

func TestEndpointRecordsOwnOptionsFormalControlsAndIdentityVersion(t *testing.T) {
	api, _, calls := endpointRecordsAPI(t, func(w http.ResponseWriter, r *http.Request) {
		testhelper.AssertEquals(t, "/keystone/v3/alternate-endpoints", r.URL.Path)
		testhelper.TestHeader(t, r, "X-Trace", "owned")
		testhelper.TestHeader(t, r, "OpenStack-API-Version", "identity 3.2")
		testhelper.AssertEquals(t, "", r.Header.Get("X-OpenStack-Nova-API-Version"))
		testcloud.JSON(w, 200, `{"endpoints":[{"id":"one"}]}`)
	})
	headers := map[string]string{"X-Trace": "owned"}
	option := endpoints.WithRecordListHeaders(headers)
	headers["X-Trace"] = "changed"
	var workers sync.WaitGroup
	for i := 0; i < 3; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			rows, err := api.AllRecords(context.Background(), option, endpoints.WithRecordListBasePath("alternate-endpoints"), endpoints.WithRecordListMicroversion("3.2"))
			if err != nil || len(rows) != 1 {
				t.Error(rows, err)
			}
		}()
	}
	workers.Wait()
	if calls.Load() != 3 || api.RawClient().Microversion != "" {
		t.Fatal(calls.Load(), api.RawClient().Microversion)
	}
	rows, err := api.AllRecords(context.Background(), endpoints.WithRecordListHeaders(nil), func(c *endpoints.RecordListOpts) error {
		c.Headers["X-Trace"] = "owned"
		c.Query["id"] = json.RawMessage(`"one"`)
		return nil
	}, endpoints.WithRecordListBasePath("alternate-endpoints"), endpoints.WithRecordListConflictingAttrs(json.RawMessage(`{"microversion":"3.2"}`)))
	if err != nil || len(rows) != 1 || calls.Load() != 4 {
		t.Fatal(rows, err, calls.Load())
	}
	for _, options := range [][]endpoints.RecordListOption{{nil}, {endpoints.WithRecordListQuery("self", make(chan int))}, {endpoints.WithRecordListQuery("session", make(chan int))}, {endpoints.WithRecordListConflictingAttrs(json.RawMessage(`{"paginated":false}`))}, {endpoints.WithRecordListBasePath("https://foreign.invalid")}, {endpoints.WithRecordListHeaders(map[string]string{"X-Auth-Token": "foreign"})}} {
		before := calls.Load()
		rows, err := api.AllRecords(context.Background(), options...)
		if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != before {
			t.Fatal(rows, err, calls.Load())
		}
	}
}

func TestEndpointRecordsSourceGuardBeforeOptionsAndAfterGeneratorCapture(t *testing.T) {
	for _, lane := range []string{"option", "generator", "cancel", "wrong-role"} {
		t.Run(lane, func(t *testing.T) {
			api, _, calls := endpointRecordsAPI(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, 200, `{"endpoints":[]}`) })
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("preflight cancellation")
			var failure error
			if lane == "generator" {
				value, err := api.ProxyListRecords(ctx)
				if err != nil {
					t.Fatal(err)
				}
				api.RawClient().ResourceBase += "changed/"
				for _, err := range value.Value.(*endpoints.RecordListGenerator).Records {
					failure = err
					break
				}
			} else {
				options := []endpoints.RecordListOption{}
				if lane == "option" {
					options = append(options, func(*endpoints.RecordListOpts) error { api.RawClient().ResourceBase += "changed/"; return nil })
				}
				if lane == "cancel" {
					cancel(cause)
				}
				if lane == "wrong-role" {
					api.RawClient().Type = "network"
				}
				_, failure = api.AllRecords(ctx, options...)
			}
			expected := resource.ErrInvalidOption
			if lane == "cancel" {
				expected = cause
			} else if lane == "wrong-role" {
				expected = resource.ErrUnsupported
			}
			if failure == nil || calls.Load() != 0 || !errors.Is(failure, expected) {
				t.Fatal(failure, calls.Load())
			}
		})
	}
}

type endpointRecordsFaultBody struct {
	io.ReadCloser
	readErr, closeErr error
	closes            *atomic.Int32
}

func (b *endpointRecordsFaultBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if n > 0 && b.readErr != nil {
		return n, b.readErr
	}
	return n, err
}
func (b *endpointRecordsFaultBody) Close() error {
	b.closes.Add(1)
	return errors.Join(b.ReadCloser.Close(), b.closeErr)
}

func TestEndpointRecordsPhysicalFaultsRetainActualProofWithoutHooks(t *testing.T) {
	for _, status := range []int{203, 401, 404, 429, 503} {
		for _, lane := range []string{"read", "close"} {
			t.Run(fmt.Sprintf("%d/%s", status, lane), func(t *testing.T) {
				body := `{"endpoints":[{"id":"one"}]}`
				api, cloud, calls := endpointRecordsAPI(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, status, body) })
				cause := errors.New(lane + " fault")
				closed := &atomic.Int32{}
				base := cloud.Provider.HTTPClient.Transport
				cloud.Provider.HTTPClient.Transport = nativeEndpointTransport(func(r *http.Request) (*http.Response, error) {
					response, err := base.RoundTrip(r)
					if response != nil {
						b := &endpointRecordsFaultBody{ReadCloser: response.Body, closes: closed}
						if lane == "read" {
							b.readErr = cause
						} else {
							b.closeErr = cause
						}
						response.Body = b
					}
					return response, err
				})
				hooks := 0
				cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks++
					return nil
				}
				cloud.Provider.ReauthFunc = func(context.Context) error { hooks++; return nil }
				cloud.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error { hooks++; return nil }
				rows, err := api.AllRecords(context.Background())
				if rows != nil || err == nil || calls.Load() != 1 || closed.Load() != 1 || hooks != 0 || !errors.Is(err, cause) {
					t.Fatal(rows, err, calls.Load(), closed.Load(), hooks)
				}
				if status < 400 {
					endpointRecordProof(t, err, status, body)
					return
				}
				receipt := err
				if status == 401 {
					var auth *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &auth) || auth.ErrOriginal == nil || !errors.Is(auth.ErrReauth, cause) {
						t.Fatal(err)
					}
					receipt = auth.ErrOriginal
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(receipt, &native) || native.Actual != status || native.Method != http.MethodGet || native.URL != cloud.Server.URL+"/keystone/v3/endpoints" || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "endpoint-actual" {
					t.Fatal(err, native)
				}
			})
		}
	}
}

func TestEndpointRecordsCleanNativeRetryKeepsScopeQueryAndHeaders(t *testing.T) {
	for _, status := range []int{401, 429, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			attempts := &atomic.Int32{}
			api, cloud, calls := endpointRecordsAPI(t, func(w http.ResponseWriter, r *http.Request) {
				testhelper.AssertEquals(t, "public", r.URL.Query().Get("interface"))
				testhelper.TestHeader(t, r, "X-Trace", "owned")
				testhelper.TestHeader(t, r, "OpenStack-API-Version", "identity 3.2")
				if attempts.Add(1) == 1 {
					testcloud.JSON(w, status, `{}`)
					return
				}
				testcloud.JSON(w, 200, `{"endpoints":[]}`)
			})
			hooks := 0
			switch status {
			case 401:
				cloud.Provider.ReauthFunc = func(context.Context) error { hooks++; return nil }
			case 429:
				cloud.Provider.MaxBackoffRetries = 1
				cloud.Provider.RetryBackoffFunc = func(_ context.Context, native *gophercloud.ErrUnexpectedResponseCode, _ error, count uint) error {
					hooks++
					if native.Actual != 429 || count != 1 {
						return errors.New("unexpected backoff")
					}
					return nil
				}
			case 503:
				cloud.Provider.RetryFunc = func(_ context.Context, method, target string, _ *gophercloud.RequestOpts, err error, count uint) error {
					hooks++
					if count != 1 {
						return err
					}
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != 503 {
						return err
					}
					testhelper.AssertEquals(t, http.MethodGet, method)
					testhelper.AssertEquals(t, cloud.Server.URL+"/keystone/v3/endpoints?interface=public", target)
					return nil
				}
			}
			rows, err := api.AllRecords(context.Background(), endpoints.WithRecordListQuery("interface", "public"), endpoints.WithRecordListHeaders(map[string]string{"X-Trace": "owned"}), endpoints.WithRecordListMicroversion("3.2"))
			if err != nil || rows == nil || len(rows) != 0 || calls.Load() != 2 || hooks != 1 {
				t.Fatal(rows, err, calls.Load(), hooks)
			}
		})
	}
}

func TestEndpointRecordsSourceStatusOptInPreservesNativeBindings(t *testing.T) {
	for _, status := range []int{101, 203, 302, 399} {
		var calls []nativeEndpointCall
		api, _ := nativeEndpointAPI(t, &calls, func(*http.Request) *http.Response { return nativeEndpointWire(status, `{"endpoints":[{"id":"one"}]}`) })
		rows, err := api.AllRecords(context.Background())
		if err != nil || len(rows) != 1 || rows[0].StatusCode != status || len(calls) != 1 {
			t.Fatal(status, rows, err, calls)
		}
	}
}
