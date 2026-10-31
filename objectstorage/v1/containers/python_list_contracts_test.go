package containers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type containerListCall struct {
	method, path, escaped, body string
	query                       url.Values
	header                      http.Header
}

func containerListAPI(t *testing.T, reply func(*http.Request) *http.Response) (*containers.API, *[]containerListCall) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := new([]containerListCall)
	cloud.SetTransport(testcloud.LinkRequest(func(r *http.Request) (*http.Response, error) {
		body := ""
		if r.Body != nil {
			data, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			body = string(data)
		}
		*calls = append(*calls, containerListCall{r.Method, r.URL.Path, r.URL.EscapedPath(), body, r.URL.Query(), r.Header.Clone()})
		return reply(r), nil
	}))
	client := cloud.Client("object-store", "/catalog/v1/")
	client.ResourceBase = cloud.Server.URL + "/reverse/a%20b/v1/AUTH_owner/"
	return containers.New(client), calls
}
func collectContainerList(seq iter.Seq2[*containers.ContainerRecord, error]) ([]*containers.ContainerRecord, *containers.ContainerRecord, error) {
	var values []*containers.ContainerRecord
	for value, err := range seq {
		if err != nil {
			return values, value, err
		}
		values = append(values, value)
	}
	return values, nil, nil
}
func containerListProof(t *testing.T, receipt *containers.ContainerRecord, err error, status int, body string) {
	t.Helper()
	containerMetadataProof(t, err, status, body)
	recordOperation(t, err, "ListRecords")
	if receipt == nil || receipt.Resource != nil || receipt.Wire != nil || receipt.StatusCode != status || string(receipt.Envelope) != body || receipt.Header.Get("X-Trans-Id") != "actual-container" {
		t.Fatalf("receipt=%+v error=%v", receipt, err)
	}
}
func TestPythonContainerListDefaultProjectionAndIndependentReceipts(t *testing.T) {
	body := `[{"name":"alpha","count":false,"bytes":9007199254740993123456789,"object_count":"۱۲","is_newest":"false","read_ACL":false,"metadata":{"Book":false,"x-container-meta-exact":0},"unknown":{"future":true}},{"id":null,"name":"beta","count":null,"bytes":0},{"id":{"opaque":true}},{"name":""}]`
	api, calls := containerListAPI(t, func(*http.Request) *http.Response {
		response := containerMetadataResponse(200, body)
		response.Header.Set("X-Container-Object-Count", "invalid account observation")
		response.Header.Set("Content-Type", "application/json")
		response.Header.Set("X-Container-Meta-Page", "must-not-project")
		return response
	})
	seq := api.ListRecords(context.Background())
	if len(*calls) != 0 {
		t.Fatal("listing was not lazy")
	}
	values, _, err := collectContainerList(seq)
	if err != nil || len(values) != 4 || len(*calls) != 1 {
		t.Fatal(err, values, *calls)
	}
	call := (*calls)[0]
	if call.method != "GET" || call.escaped != "/reverse/a%20b/v1/AUTH_owner/" || len(call.query) != 0 || call.body != "" || call.header.Get("Accept") != "application/json" || call.header.Get("X-Auth-Token") != "test-token" || call.header.Get("OpenStack-API-Version") != "" {
		t.Fatal(call)
	}
	for key, want := range map[string]string{"name": `"alpha"`, "id": `"alpha"`, "count": "false", "bytes": "9007199254740993123456789", "object_count": "12", "bytes_used": "9007199254740993123456789", "is_newest": "true", "read_ACL": "false", "content_type": "null", "location": "null"} {
		recordView(t, values[0], key, want)
	}
	if len(values[0].Resource.Body) != 21 || len(values[0].Metadata) != 2 || values[0].Metadata["X-Container-Meta-Book"] != "False" || values[0].Metadata["x-container-meta-exact"] != "0" || string(values[0].Wire.Body["unknown"]) != `{"future":true}` || string(values[0].Wire.Body["count"]) != "false" || string(values[0].Envelope) != body || values[0].StatusCode != 200 {
		t.Fatal(values[0])
	}
	recordView(t, values[1], "id", "null")
	recordView(t, values[1], "name", `"beta"`)
	recordView(t, values[1], "count", "null")
	recordView(t, values[2], "id", `{"opaque":true}`)
	recordView(t, values[2], "name", `{"opaque":true}`)
	recordView(t, values[3], "id", `""`)
	values[0].Resource.Body["name"][1] = 'X'
	values[0].Wire.Body["name"][1] = 'Y'
	values[0].Header.Set("X-Trans-Id", "consumer")
	if string(values[0].SeedBody["name"]) != `"alpha"` || string(values[0].Envelope) != body || values[0].Resource.Header.Get("X-Trans-Id") != "actual-container" || values[1].Header.Get("X-Trans-Id") != "actual-container" || api.RawClient().MoreHeaders != nil {
		t.Fatal("record or client carrier aliases")
	}
}
func TestPythonContainerListQueryClassificationAndOptionOwnership(t *testing.T) {
	for _, tc := range []struct {
		label string
		raw   any
		want  []string
	}{
		{"boolean", false, []string{"False"}}, {"exact number", json.Number("900719925474099312345"), []string{"900719925474099312345"}}, {"repeated", []any{"a", false, nil, 0}, []string{"a", "False", "0"}}, {"null", nil, nil}, {"object keys", map[string]any{"two": false, "one": 0}, []string{"one", "two"}},
	} {
		t.Run(tc.label, func(t *testing.T) {
			api, calls := containerListAPI(t, func(*http.Request) *http.Response { return containerMetadataResponse(200, "[]") })
			_, _, err := collectContainerList(api.ListRecords(context.Background(), containers.WithRecordListQuery("prefix", tc.raw), containers.WithRecordListQuery("end_marker", make(chan int)), containers.WithRecordListQuery("full_listing", false), containers.WithRecordListQuery("unknown", make(chan int))))
			if err != nil || len(*calls) != 1 || !reflect.DeepEqual((*calls)[0].query["prefix"], tc.want) || len((*calls)[0].query) > 1 {
				t.Fatal(err, *calls, tc.want)
			}
		})
	}
	raw := json.RawMessage(`"orig"`)
	limit := 0
	format := "json"
	header := map[string]string{"Accept": "application/custom", "X-Probe": "original"}
	cfg := containers.RecordListOpts{Query: map[string]json.RawMessage{"prefix": raw, "unknown": json.RawMessage("broken"), "name": json.RawMessage(`"b"`), "is_newest": json.RawMessage("broken")}, Headers: header, Limit: &limit, Format: &format}
	option := containers.WithRecordListOpts(cfg)
	raw[1] = 'X'
	cfg.Query["name"] = json.RawMessage(`"a"`)
	limit = 4
	format = "plain"
	header["X-Probe"] = "changed"
	api, calls := containerListAPI(t, func(*http.Request) *http.Response {
		return containerMetadataResponse(200, `[{"name":"a"},{"name":"b"}]`)
	})
	values, _, err := collectContainerList(api.ListRecords(context.Background(), option))
	if err != nil || len(values) != 1 || len(*calls) != 1 || (*calls)[0].query.Get("prefix") != "orig" || (*calls)[0].query.Get("limit") != "0" || (*calls)[0].query.Get("format") != "json" || (*calls)[0].header.Get("Accept") != "application/custom" || (*calls)[0].header.Get("X-Probe") != "original" || len((*calls)[0].query) != 3 {
		t.Fatal(err, values, *calls)
	}
	recordView(t, values[0], "name", `"b"`)
}
func TestPythonContainerListLocalEqualityAndDictionaryShortCircuit(t *testing.T) {
	for _, tc := range []struct {
		label, body, key string
		filter           any
		want             int
		failure          bool
	}{
		{"typed bool equals zero", `[{"count":false},{"count":true}]`, "count", false, 1, false},
		{"null missing equality", `[{"name":null},{}]`, "name", nil, 2, false},
		{"empty predicate truthful scalar", `[{"name":"truthy"},{"name":""},{"name":0}]`, "name", map[string]any{}, 1, false},
		{"nested ordered short circuit", `[{"name":{"a":0,"b":"scalar"}}]`, "name", json.RawMessage(`{"a":1,"b":{"x":1}}`), 0, false},
		{"missing nested null", `[{"name":{"other":true}}]`, "name", map[string]any{"missing": nil}, 1, false},
		{"mapping get error", `[{"name":"scalar"}]`, "name", map[string]any{"x": 1}, 0, true},
	} {
		t.Run(tc.label, func(t *testing.T) {
			api, calls := containerListAPI(t, func(*http.Request) *http.Response { return containerMetadataResponse(200, tc.body) })
			values, receipt, err := collectContainerList(api.ListRecords(context.Background(), containers.WithRecordListQuery(tc.key, tc.filter)))
			if len(values) != tc.want || len(*calls) != 1 || (err != nil) != tc.failure {
				t.Fatal(err, values, *calls)
			}
			if tc.failure {
				containerListProof(t, receipt, err, 200, tc.body)
			}
		})
	}
}
func TestPythonContainerListOrderedLocalPredicatesAvoidLaterMappingFault(t *testing.T) {
	body := `[{"name":"different","id":"truthy scalar"}]`
	for _, reverse := range []bool{false, true} {
		api, calls := containerListAPI(t, func(*http.Request) *http.Response { return containerMetadataResponse(200, body) })
		options := []containers.RecordListOption{containers.WithRecordListQuery("name", "wanted"), containers.WithRecordListQuery("id", map[string]any{"child": true})}
		if reverse {
			options[0], options[1] = options[1], options[0]
		}
		values, receipt, err := collectContainerList(api.ListRecords(context.Background(), options...))
		if len(values) != 0 || len(*calls) != 1 || (err != nil) != reverse {
			t.Fatal(err, values, *calls)
		}
		if reverse {
			containerListProof(t, receipt, err, 200, body)
		}
	}
}

func TestPythonContainerListCountPagingUsesRawRowsAndRetainedMarker(t *testing.T) {
	first := `[{"name":"skip","count":0},{"name":"cursor%2F space","count":1}]`
	second := `[{"name":"last","count":1}]`
	api, calls := containerListAPI(t, func(r *http.Request) *http.Response {
		if r.URL.Query().Get("marker") == "" {
			response := containerMetadataResponse(200, first)
			response.Header.Set("X-Account-Container-Count", " +٣ ")
			return response
		}
		if r.URL.Query().Get("marker") != "cursor%2F space" || r.URL.Query().Get("prefix") != "fixed" {
			t.Fatal(r.URL)
		}
		response := containerMetadataResponse(200, second)
		response.Header.Set("X-Account-Container-Count", "3")
		return response
	})
	var values []*containers.ContainerRecord
	for value, err := range api.ListRecords(context.Background(), containers.WithRecordListPrefix("fixed"), containers.WithRecordListQuery("count", 1)) {
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, value)
		value.Resource.Body["id"] = json.RawMessage(`"consumer tampered"`)
	}
	if len(values) != 2 || len(*calls) != 2 || (*calls)[1].query.Get("marker") != "cursor%2F space" {
		t.Fatal(values, *calls)
	}
}
func TestPythonContainerListLimitTruthinessAndIgnoredFullListing(t *testing.T) {
	for _, tc := range []struct {
		label    string
		value    any
		want     string
		requests int
	}{
		{"integer zero", 0, "0", 1}, {"boolean false", false, "False", 1}, {"string zero", "0", "0", 3}, {"positive", 2, "2", 3}, {"negative Source raw", -1, "-1", 3},
	} {
		t.Run(tc.label, func(t *testing.T) {
			api, calls := containerListAPI(t, func(r *http.Request) *http.Response {
				switch r.URL.Query().Get("marker") {
				case "":
					return containerMetadataResponse(200, `[{"name":"a"}]`)
				case "a":
					return containerMetadataResponse(200, `[{"name":"b"}]`)
				default:
					return containerMetadataResponse(200, "[]")
				}
			})
			values, _, err := collectContainerList(api.ListRecords(context.Background(), containers.WithRecordListQuery("limit", tc.value), containers.WithRecordListQuery("full_listing", false), containers.WithRecordListQuery("end_marker", "a")))
			if err != nil || len(*calls) != tc.requests || len(values) != (tc.requests+1)/2 || (*calls)[0].query.Get("limit") != tc.want {
				t.Fatal(err, values, *calls)
			}
			for _, call := range *calls {
				if call.query.Get("limit") != tc.want || len(call.query["full_listing"]) != 0 || len(call.query["end_marker"]) != 0 {
					t.Fatal(call)
				}
			}
		})
	}
	t.Run("empty page ignores continuation and malformed count", func(t *testing.T) {
		api, calls := containerListAPI(t, func(*http.Request) *http.Response {
			r := containerMetadataResponse(200, "[]")
			r.Header.Set("Link", `<https://foreign.invalid/>; rel="next"`)
			r.Header.Set("X-Account-Container-Count", "invalid")
			return r
		})
		values, _, err := collectContainerList(api.ListRecords(context.Background(), containers.WithRecordListLimit(2)))
		if err != nil || len(values) != 0 || len(*calls) != 1 {
			t.Fatal(err, values, *calls)
		}
	})
}
func TestPythonContainerListAdvertisedContinuationAndAccountGuards(t *testing.T) {
	for _, kind := range []string{"header", "body next", "body count key links", "false href top fallback"} {
		t.Run(kind, func(t *testing.T) {
			api, calls := containerListAPI(t, func(r *http.Request) *http.Response {
				if r.URL.Query().Get("marker") == "a" {
					return containerMetadataResponse(200, "[]")
				}
				body := `[{"name":"a"}]`
				if kind == "body next" {
					body = `{"name":"a","next":"?marker=a"}`
				}
				if kind == "body count key links" {
					body = `{"name":"a","X-Account-Container-Count":[{"rel":"next","href":"?marker=a"}]}`
				}
				if kind == "false href top fallback" {
					body = `{"name":"a","X-Account-Container-Count":[{"rel":"next","href":null},{"rel":"next","href":"https://foreign.invalid/"}],"next":"?marker=a"}`
				}
				response := containerMetadataResponse(200, body)
				response.Header.Set("X-Account-Container-Count", "invalid")
				if kind == "header" {
					response.Header.Set("Link", `<?marker=a>; rel="next", <ignored?title=a,b>; rel="previous"; title="comma, quote"`)
				}
				return response
			})
			values, _, err := collectContainerList(api.ListRecords(context.Background(), containers.WithRecordListPrefix("held")))
			if err != nil || len(values) != 1 || len(*calls) != 2 || (*calls)[1].query.Get("prefix") != "held" {
				t.Fatal(err, values, *calls)
			}
		})
	}
	for _, link := range []string{`<https://foreign.invalid/>; rel="next"`, `</different-account/>; rel="next"`, `<?prefix=changed&marker=a>; rel="next"`, `<?limit=3&marker=a>; rel="next"`, `<?marker=first>; rel="next"`, `<?marker=a>; rel="next", <?marker=b>; rel="next"`, `malformed`, `<?marker=a>; rel="next`} {
		t.Run(link, func(t *testing.T) {
			body := `[{"name":"a"}]`
			api, calls := containerListAPI(t, func(*http.Request) *http.Response {
				r := containerMetadataResponse(200, body)
				r.Header.Set("Link", link)
				return r
			})
			values, receipt, err := collectContainerList(api.ListRecords(context.Background(), containers.WithRecordListPrefix("held"), containers.WithRecordListMarker("first"), containers.WithRecordListLimit(2)))
			if err == nil || len(values) != 1 || len(*calls) != 1 {
				t.Fatal(err, values, *calls)
			}
			containerListProof(t, receipt, err, 200, body)
		})
	}
}
func TestPythonContainerListRawCapsAndPageBoundsStopBeforeUnusedRows(t *testing.T) {
	for _, options := range [][]containers.RecordListOption{
		{containers.WithRecordListMaxItems(1)}, {containers.WithRecordListSinglePage(true)},
	} {
		body := `[{"name":"a"}]`
		api, calls := containerListAPI(t, func(*http.Request) *http.Response {
			r := containerMetadataResponse(200, body)
			r.Header.Set("Link", `<https://foreign.invalid/>; rel="next"`)
			return r
		})
		values, _, err := collectContainerList(api.ListRecords(context.Background(), options...))
		if err != nil || len(values) != 1 || len(*calls) != 1 {
			t.Fatal(err, values, *calls)
		}
	}
	body := `[{"name":"skip"},null]`
	api, calls := containerListAPI(t, func(*http.Request) *http.Response { return containerMetadataResponse(200, body) })
	values, _, err := collectContainerList(api.ListRecords(context.Background(), containers.WithRecordListMaxItems(1), containers.WithRecordListQuery("name", "wanted")))
	if err != nil || len(values) != 0 || len(*calls) != 1 || (*calls)[0].query.Get("limit") != "1" {
		t.Fatal(err, values, *calls)
	}
	api, calls = containerListAPI(t, func(*http.Request) *http.Response {
		r := containerMetadataResponse(200, `[{"name":"a"}]`)
		r.Header.Set("X-Account-Container-Count", "900719925474099312345")
		return r
	})
	values, receipt, err := collectContainerList(api.ListRecords(context.Background(), containers.WithRecordListMaxPages(1)))
	if err == nil || len(values) != 1 || len(*calls) != 1 {
		t.Fatal(err, values, *calls)
	}
	containerListProof(t, receipt, err, 200, `[{"name":"a"}]`)
	api, calls = containerListAPI(t, func(*http.Request) *http.Response { return containerMetadataResponse(200, `[{"name":"a"},null]`) })
	for value, err := range api.ListRecords(context.Background()) {
		if err != nil || value.Resource == nil {
			t.Fatal(err, value)
		}
		break
	}
	if len(*calls) != 1 {
		t.Fatal(*calls)
	}
}
func TestPythonContainerListWrongShapesAndConstructorCollisionsKeepPageProof(t *testing.T) {
	for _, tc := range []struct {
		body         string
		status, want int
	}{
		{"", 204, 0}, {"broken", 200, 0}, {"null", 200, 0}, {"false", 200, 0}, {"12", 200, 0}, {`[null]`, 200, 0}, {`["name"]`, 200, 0},
		{`[{"name":"a"},null]`, 200, 1}, {`[{"cls":null}]`, 200, 0}, {`[{"cls":false}]`, 200, 0}, {`[{"connection":null}]`, 200, 0}, {`[{"microversion":null}]`, 200, 0}, {`[{"_synchronized":false}]`, 200, 0},
		{`[{"metadata":false}]`, 200, 0}, {`[{"count":"²"}]`, 200, 0}, {`[{"name":"a"},]`, 200, 0}, {"[{\"name\":\"\xff\"}]", 200, 0},
	} {
		t.Run(fmt.Sprintf("%d/%q", tc.status, tc.body), func(t *testing.T) {
			api, calls := containerListAPI(t, func(*http.Request) *http.Response { return containerMetadataResponse(tc.status, tc.body) })
			values, receipt, err := collectContainerList(api.ListRecords(context.Background()))
			if err == nil || len(values) != tc.want || len(*calls) != 1 {
				t.Fatal(err, values, *calls)
			}
			containerListProof(t, receipt, err, tc.status, tc.body)
		})
	}
	for _, body := range []string{`{"self":"discard","name":"singleton"}`, `[{"metadata":null}]`, `[{"bytes":"not-int"}]`} {
		api, _ := containerListAPI(t, func(*http.Request) *http.Response { return containerMetadataResponse(199, body) })
		values, _, err := collectContainerList(api.ListRecords(context.Background()))
		if err != nil || len(values) != 1 || values[0].StatusCode != 199 || string(values[0].Envelope) != body {
			t.Fatal(err, values)
		}
		if body == `[{"bytes":"not-int"}]` {
			recordView(t, values[0], "bytes", "0")
			recordView(t, values[0], "bytes_used", "0")
			if string(values[0].Wire.Body["bytes"]) != `"not-int"` {
				t.Fatal("descriptor erased actual wire value", values[0])
			}
		}
	}
}
func TestPythonContainerListPhysicalReadCloseFailuresAndStickySource(t *testing.T) {
	for _, status := range []int{199, 200, 304, 400, 403, 404, 429, 500, 503, 599} {
		for _, fault := range []string{"read", "close", "mixed", "source read restore close", "cancel close"} {
			t.Run(fmt.Sprintf("%d/%s", status, fault), func(t *testing.T) {
				raw := `[{"name":"actual"}]`
				readCause, closeCause := errors.New("read cause"), errors.New("close cause")
				cancelCause := errors.New("cancel cause")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				var client *gophercloud.ServiceClient
				body := &containerMetadataBody{Reader: strings.NewReader(raw)}
				if fault == "read" || fault == "mixed" {
					done := false
					body.Reader = containerMetadataReader(func(p []byte) (int, error) {
						if done {
							return 0, io.EOF
						}
						done = true
						return copy(p, raw), readCause
					})
				}
				if fault == "close" || fault == "mixed" {
					body.closeErr = closeCause
				}
				if fault == "source read restore close" {
					done := false
					body.Reader = containerMetadataReader(func(p []byte) (int, error) {
						if done {
							return 0, io.EOF
						}
						done = true
						client.ResourceBase = "https://changed.invalid/account/"
						return copy(p, raw), nil
					})
					body.onClose = func() { client.ResourceBase = "" }
				}
				if fault == "cancel close" {
					body.onClose = func() { cancel(cancelCause) }
				}
				client = containerMetadataClient(func(*http.Request) (*http.Response, error) { return containerMetadataWire(status, body), nil })
				calls, retries := 0, 0
				originalTransport := client.HTTPClient.Transport
				client.HTTPClient.Transport = testcloud.LinkRequest(func(r *http.Request) (*http.Response, error) { calls++; return originalTransport.RoundTrip(r) })
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				values, receipt, err := collectContainerList(containers.New(client).ListRecords(ctx))
				if err == nil || len(values) != 0 || calls != 1 || retries != 0 || body.closes.Load() != 1 {
					t.Fatal(err, values, calls, retries, body.closes.Load())
				}
				containerListProof(t, receipt, err, status, raw)
				if (fault == "read" || fault == "mixed") && !errors.Is(err, readCause) || (fault == "close" || fault == "mixed") && !errors.Is(err, closeCause) || fault == "source read restore close" && !errors.Is(err, resource.ErrInvalidOption) || fault == "cancel close" && !errors.Is(err, cancelCause) {
					t.Fatal(err)
				}
				if status >= 400 {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != status || string(native.Body) != raw || native.Method != "GET" || native.URL != containerMetadataBase {
						t.Fatal(err, native)
					}
				}
			})
		}
	}
}
func TestPythonContainerListRejectedStatusesAreNeverMissingSuppressed(t *testing.T) {
	for _, status := range []int{400, 403, 404, 409, 429, 500, 503, 599} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			raw := `{"physical":"reject"}`
			body := &containerMetadataBody{Reader: strings.NewReader(raw)}
			api, calls := containerListAPI(t, func(*http.Request) *http.Response { return containerMetadataWire(status, body) })
			values, receipt, err := collectContainerList(api.ListRecords(context.Background()))
			var native gophercloud.ErrUnexpectedResponseCode
			if err == nil || len(values) != 0 || len(*calls) != 1 || body.closes.Load() != 1 || !errors.As(err, &native) || native.Actual != status || string(native.Body) != raw {
				t.Fatal(err, values, *calls, native)
			}
			containerListProof(t, receipt, err, status, raw)
		})
	}
}
func TestPythonContainerListRetryHeaderOwnershipAndActualReceipt(t *testing.T) {
	cases := []struct {
		key, value string
		omit       bool
	}{{"Authorization", "Bearer foreign", false}, {"X-Service-Token", "foreign", false}, {"Content-Length", "5", false}, {"Transfer-Encoding", "chunked", false}, {"Connection", "close", false}, {"Host", "foreign.invalid", false}, {"X-OpenStack-Nova-API-Version", "2.80", false}, {"X-OpenStack-Volume-API-Version", "3.80", false}, {"Accept", "text/plain", false}, {"OpenStack-API-Version", "object-store 9.0", false}, {"OpenStack-API-Version", "", true}}
	for _, tc := range cases {
		t.Run(tc.key+fmt.Sprint(tc.omit), func(t *testing.T) {
			raw := `{"retry":"original"}`
			body := &containerMetadataBody{Reader: strings.NewReader(raw)}
			callbacks := 0
			callbackCause := errors.New("callback cause")
			api, calls := containerListAPI(t, func(*http.Request) *http.Response { return containerMetadataWire(503, body) })
			api.RawClient().Microversion = "1.2"
			api.RawClient().MoreHeaders = map[string]string{"openstack-api-version": "object-store 1.2"}
			api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				callbacks++
				if tc.omit {
					opts.OmitHeaders = append(opts.OmitHeaders, tc.key)
				} else {
					opts.MoreHeaders[tc.key] = tc.value
				}
				return callbackCause
			}
			values, receipt, err := collectContainerList(api.ListRecords(context.Background()))
			var native gophercloud.ErrUnexpectedResponseCode
			if len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, callbackCause) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != raw || native.Method != "GET" || len(*calls) != 1 || callbacks != 1 || body.closes.Load() != 1 || api.RawClient().Microversion != "1.2" || len(api.RawClient().MoreHeaders) != 1 {
				t.Fatal(err, native, values, *calls, callbacks, body.closes.Load())
			}
			containerListProof(t, receipt, err, 503, raw)
		})
	}
	t.Run("ordinary retry header live token and per operation version", func(t *testing.T) {
		number := 0
		api, calls := containerListAPI(t, func(*http.Request) *http.Response {
			number++
			if number == 1 {
				return containerMetadataResponse(503, `{"retry":true}`)
			}
			return containerMetadataResponse(200, "[]")
		})
		api.RawClient().Type = ""
		api.RawClient().Microversion = "1.2"
		api.RawClient().MoreHeaders = map[string]string{"openstack-api-version": "object-store 1.2", "X-Newest": "True"}
		api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
			opts.MoreHeaders["X-Retry-Probe"] = "ordinary"
			api.RawClient().SetToken("fresh")
			return nil
		}
		_, _, err := collectContainerList(api.ListRecords(context.Background(), containers.WithRecordListMicroversion("1.3")))
		if err != nil || len(*calls) != 2 || (*calls)[0].header.Get("OpenStack-API-Version") != "object-store 1.3" || (*calls)[1].header.Get("OpenStack-API-Version") != "object-store 1.3" || (*calls)[0].header.Get("X-Auth-Token") != "test-token" || (*calls)[1].header.Get("X-Auth-Token") != "fresh" || (*calls)[1].header.Get("X-Retry-Probe") != "ordinary" || api.RawClient().Microversion != "1.2" || api.RawClient().Type != "" || api.RawClient().MoreHeaders["openstack-api-version"] != "object-store 1.2" {
			t.Fatal(err, *calls, api.RawClient())
		}
	})
}
func TestPythonContainerListPreflightCancellationAndRetainedOptionOwnership(t *testing.T) {
	for _, option := range []containers.RecordListOption{
		nil, containers.WithRecordListLimit(-1), containers.WithRecordListMaxItems(-1), containers.WithRecordListMaxPages(-1), containers.WithRecordListHeaders(map[string]string{"Authorization": "bad"}), containers.WithRecordListHeaders(map[string]string{"OpenStack-API-Version": "object-store 1.2"}), containers.WithRecordListPrefix("bad\xff"), containers.WithRecordListQuery("prefix", json.RawMessage("bad")), containers.WithRecordListQuery("paginated", false),
	} {
		api, calls := containerListAPI(t, func(*http.Request) *http.Response { t.Fatal("preflight sent HTTP"); return nil })
		_, receipt, err := collectContainerList(api.ListRecords(context.Background(), option))
		var proof *resource.ResponseError
		if err == nil || receipt != nil || len(*calls) != 0 || errors.As(err, &proof) {
			t.Fatal(err, receipt, *calls)
		}
	}
	api, calls := containerListAPI(t, func(*http.Request) *http.Response { return containerMetadataResponse(200, "[]") })
	for _, seq := range []iter.Seq2[*containers.ContainerRecord, error]{(*containers.API)(nil).ListRecords(context.Background()), api.ListRecords(nil)} {
		_, receipt, err := collectContainerList(seq)
		if err == nil || receipt != nil || len(*calls) != 0 {
			t.Fatal(err, receipt, *calls)
		}
	}
	cause := errors.New("option canceled")
	ctx, cancel := context.WithCancelCause(context.Background())
	_, receipt, err := collectContainerList(api.ListRecords(ctx, func(*containers.RecordListOpts) error { cancel(cause); return nil }))
	if !errors.Is(err, cause) || receipt != nil || len(*calls) != 0 {
		t.Fatal(err, receipt, *calls)
	}
	ctx = context.Background()
	var retained *containers.RecordListOpts
	api, calls = containerListAPI(t, func(*http.Request) *http.Response {
		retained.Query["name"] = json.RawMessage(`"wrong"`)
		*retained.Prefix = "changed"
		retained.Headers["X-Probe"] = "changed"
		return containerMetadataResponse(200, `[{"name":"wanted"}]`)
	})
	values, _, err := collectContainerList(api.ListRecords(ctx, containers.WithRecordListPrefix("fixed"), containers.WithRecordListQuery("name", "wanted"), containers.WithRecordListHeaders(map[string]string{"X-Probe": "fixed"}), func(cfg *containers.RecordListOpts) error { retained = cfg; return nil }))
	if err != nil || len(values) != 1 || (*calls)[0].query.Get("prefix") != "fixed" || (*calls)[0].header.Get("X-Probe") != "fixed" {
		t.Fatal(err, values, *calls)
	}
}
func TestPythonContainerListConcurrentOwnedOptionReuse(t *testing.T) {
	cloud := testcloud.New(t)
	var requests atomic.Int32
	cloud.SetTransport(testcloud.LinkRequest(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		if r.URL.Query().Get("prefix") != "owned" || r.Header.Get("X-Probe") != "owned" {
			t.Error(r.URL, r.Header)
		}
		return containerMetadataResponse(200, `[{"name":"a","count":0}]`), nil
	}))
	api := containers.New(cloud.Client("object-store", "/v1/AUTH_owner/"))
	option := containers.WithRecordListOpts(containers.RecordListOpts{Query: map[string]json.RawMessage{"prefix": json.RawMessage(`"owned"`)}, Headers: map[string]string{"X-Probe": "owned"}})
	var wait sync.WaitGroup
	for i := 0; i < 8; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			values, _, err := collectContainerList(api.ListRecords(context.Background(), option))
			if err != nil || len(values) != 1 {
				t.Error(err, values)
			}
		}()
	}
	wait.Wait()
	if requests.Load() != 8 || api.RawClient().MoreHeaders != nil {
		t.Fatal(requests.Load(), api.RawClient())
	}
}

func TestPythonContainerListPublicFixtureWireAndSourceCapture(t *testing.T) {
	cloud := testcloud.New(t)
	requests := 0
	cloud.Mux.HandleFunc("/reverse/", func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != "GET" || r.URL.EscapedPath() != "/reverse/a%20b/v1/AUTH_owner/" || r.Header.Get("Accept") != "application/json" || r.Header.Get("X-Auth-Token") != "test-token" || r.URL.Query().Get("prefix") != "a%2Fb space" {
			t.Error(r.Method, r.RequestURI, r.Header)
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil || len(raw) != 0 {
			t.Error(err, string(raw))
		}
		if r.URL.Query().Get("marker") == "" {
			w.Header().Set("X-Account-Container-Count", "2")
			testcloud.JSON(w, 200, `[{"name":"a%2Fb space","count":0}]`)
		} else {
			if r.URL.Query().Get("marker") != "a%2Fb space" {
				t.Error(r.URL)
			}
			w.Header().Set("X-Account-Container-Count", "2")
			testcloud.JSON(w, 200, `[{"name":"last","count":1}]`)
		}
	})
	client := cloud.Client("object-store", "/endpoint/")
	client.ResourceBase = cloud.Server.URL + "/reverse/a%20b/v1/AUTH_owner/"
	api := containers.New(client)
	values, _, err := collectContainerList(api.ListRecords(context.Background(), containers.WithRecordListPrefix("a%2Fb space")))
	if err != nil || len(values) != 2 || requests != 2 {
		t.Fatal(err, values, requests)
	}
	original := client.ResourceBase
	_, receipt, err := collectContainerList(api.ListRecords(context.Background(), func(*containers.RecordListOpts) error { client.ResourceBase = cloud.Server.URL + "/other/"; return nil }))
	client.ResourceBase = original
	if !errors.Is(err, resource.ErrInvalidOption) || receipt != nil || requests != 2 {
		t.Fatal(err, receipt, requests)
	}
}

func TestPythonContainerListOperationAcceptAndConditionalRetryOwnership(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		api, calls := containerListAPI(t, func(*http.Request) *http.Response { return containerMetadataResponse(200, "[]") })
		api.RawClient().MoreHeaders = map[string]string{"accept": "application/source"}
		var options []containers.RecordListOption
		want := "application/json"
		if explicit {
			options = append(options, containers.WithRecordListHeaders(map[string]string{"accept": "application/operation"}))
			want = "application/operation"
		}
		_, _, err := collectContainerList(api.ListRecords(context.Background(), options...))
		if err != nil || len(*calls) != 1 || (*calls)[0].header.Get("Accept") != want || api.RawClient().MoreHeaders["accept"] != "application/source" {
			t.Fatal(err, *calls, api.RawClient())
		}
	}
	for _, key := range []string{"If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since"} {
		for _, configured := range []bool{false, true} {
			for _, omit := range []bool{false, true} {
				if omit && !configured {
					continue
				}
				t.Run(key+fmt.Sprint(configured, omit), func(t *testing.T) {
					raw := `{"retry":"conditional"}`
					body := &containerMetadataBody{Reader: strings.NewReader(raw)}
					callbackCause := errors.New("conditional retry cause")
					api, calls := containerListAPI(t, func(*http.Request) *http.Response { return containerMetadataWire(503, body) })
					var options []containers.RecordListOption
					if configured {
						options = append(options, containers.WithRecordListHeaders(map[string]string{key: "owned-condition"}))
					}
					api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
						if omit {
							opts.OmitHeaders = append(opts.OmitHeaders, key)
						} else {
							opts.MoreHeaders[key] = "changed-condition"
						}
						return callbackCause
					}
					_, receipt, err := collectContainerList(api.ListRecords(context.Background(), options...))
					if !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, callbackCause) || len(*calls) != 1 || body.closes.Load() != 1 {
						t.Fatal(err, *calls, body.closes.Load())
					}
					containerListProof(t, receipt, err, 503, raw)
				})
			}
		}
	}
}
func TestPythonContainerListExactDecimalEqualityPolicy(t *testing.T) {
	body := `[{"name":"large","count":9007199254740992}]`
	api, calls := containerListAPI(t, func(*http.Request) *http.Response { return containerMetadataResponse(200, body) })
	values, _, err := collectContainerList(api.ListRecords(context.Background(), containers.WithRecordListQuery("count", json.RawMessage("9007199254740993.0"))))
	if err != nil || len(values) != 0 || len(*calls) != 1 {
		t.Fatal("shared exact-decimal policy rounded raw filter", err, values, *calls)
	}
}
