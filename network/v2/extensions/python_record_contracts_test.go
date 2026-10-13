package extensions_test

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

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func recordExtensionAPI(t *testing.T, handler func(http.ResponseWriter, *http.Request)) (*extensions.API, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	count := &atomic.Int32{}
	cloud.Mux.HandleFunc("/neutron/", func(w http.ResponseWriter, req *http.Request) {
		count.Add(1)
		if req.Method != http.MethodGet || req.Header.Get("X-Auth-Token") == "" {
			t.Error(req.Method, req.Header)
		}
		w.Header().Set("X-Request-Id", "extension-actual")
		handler(w, req)
	})
	client := cloud.Client("network", "/catalog/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return extensions.New(client), count
}
func extensionRecordField(t *testing.T, value *extensions.ExtensionRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatal(key, want, value)
	}
}
func extensionRecordProof(t *testing.T, err error, status int, body string) {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "extension-actual" || proof.Cause == nil {
		t.Fatal(err, proof)
	}
}

func TestExtensionRecordsDefaultMROAliasIDAndNestedLinksCast(t *testing.T) {
	body := `{"extensions":[{"alias":"qos","name":"Quality of Service","links":[null,false,1,"text",[1],{"href":"/qos","x_new":9007199254740993}],"updated":"literal","tags":false,"revision_number":"untouched","if_match":"body-only","self":"wire-only","location":{"spoof":true}},{}]}`
	api, count := recordExtensionAPI(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/neutron/v2.0/extensions" || req.URL.RawQuery != "" || req.Header.Get("Accept") != "application/json" {
			t.Error(req.URL, req.Header)
		}
		w.Header().Set("If-Match", "http-only")
		testcloud.JSON(w, 203, body)
	})
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 2 || count.Load() != 1 {
		t.Fatal(rows, err, count.Load())
	}
	for key, want := range map[string]string{"id": `"qos"`, "alias": `"qos"`, "name": `"Quality of Service"`, "links": `[{},{},{},{},{},{"href":"/qos","x_new":9007199254740993}]`, "updated_at": `"literal"`, "tags": "false", "revision_number": `"untouched"`, "if_match": `"body-only"`, "location": "null"} {
		extensionRecordField(t, rows[0], key, want)
	}
	if rows[0].Resource.Body["self"] != nil || rows[0].Resource.Body["updated"] != nil || string(rows[0].Wire.Body["self"]) != `"wire-only"` || string(rows[0].Wire.Body["links"]) != `[null,false,1,"text",[1],{"href":"/qos","x_new":9007199254740993}]` || rows[0].Header.Get("If-Match") != "http-only" || string(rows[0].Envelope) != body || rows[0].StatusCode != 203 || len(rows[1].Resource.Body) != 7 {
		t.Fatal(rows)
	}
	for _, key := range []string{"id", "name", "alias", "description", "links", "updated_at", "location"} {
		extensionRecordField(t, rows[1], key, "null")
	}
	for _, tc := range []struct{ row, wantID, wantLinks string }{
		{`{"id":null,"alias":"fallback","links":null}`, "null", "null"}, {`{"id":false,"alias":"fallback","links":{"href":"x"}}`, "false", `[{"href":"x"}]`}, {`{"id":0,"alias":"fallback","links":"text"}`, "0", `[{}]`}, {`{"id":"","alias":"fallback","links":[]}`, `""`, `[]`},
	} {
		item, calls := recordExtensionAPI(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, 200, `{"extensions":[`+tc.row+`]}`) })
		rows, err := item.AllRecords(context.Background())
		if err != nil || len(rows) != 1 || calls.Load() != 1 {
			t.Fatal(tc, rows, err)
		}
		extensionRecordField(t, rows[0], "id", tc.wantID)
		extensionRecordField(t, rows[0], "links", tc.wantLinks)
	}
}

func TestExtensionListRecordsAllBodyFiltersAndQueryControlsAreLibraryOwned(t *testing.T) {
	api, count := recordExtensionAPI(t, func(w http.ResponseWriter, req *http.Request) {
		if !reflect.DeepEqual(req.URL.Query(), mapQuery("limit", "0", "marker", "start")) {
			t.Error(req.URL)
		}
		testcloud.JSON(w, 200, `{"extensions":[{"alias":"qos","name":"Name","description":{"enabled":1,"extra":true},"updated_at":"old","updated":{"n":9007199254740992},"links":[null,{"href":"x"}]},{"alias":"skip","name":"Name","description":{},"links":[]}]}`)
	})
	rows, err := api.AllRecords(context.Background(), extensions.WithRecordListLimit(0), extensions.WithRecordListMarker("start"), extensions.WithRecordListQuery("id", "qos"), extensions.WithRecordListQuery("alias", "qos"), extensions.WithRecordListQuery("name", "Name"), extensions.WithRecordListQuery("description", map[string]any{"enabled": true}), extensions.WithRecordListQuery("updated_at", json.RawMessage(`{"n":9007199254740993.0}`)), extensions.WithRecordListQuery("links", []any{map[string]any{}, map[string]any{"href": "x"}}), extensions.WithRecordListQuery("fields", make(chan int)), extensions.WithRecordListQuery("unknown", make(chan int)))
	if err != nil || len(rows) != 1 || count.Load() != 1 {
		t.Fatal(rows, err, count.Load())
	}
	extensionRecordField(t, rows[0], "id", `"qos"`)
}
func mapQuery(values ...string) url.Values {
	result := url.Values{}
	for i := 0; i < len(values); i += 2 {
		result[values[i]] = append(result[values[i]], values[i+1])
	}
	return result
}

func TestExtensionFindMemberSeedExplicitNullAndPrivateActualEvidence(t *testing.T) {
	body := `{"extension":{"alias":"wire-alias","updated":"literal","x_new":{"n":9007199254740993}}}`
	api, count := recordExtensionAPI(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.EscapedPath() != "/neutron/v2.0/extensions/literal%2Fname%20%252F" || req.URL.Query().Get("name") != "seed" || req.URL.Query().Get("x_seed") != "key" {
			t.Error(req.URL)
		}
		testcloud.JSON(w, 203, body)
	})
	value, err := api.FindRecord(context.Background(), "literal/name %2F", extensions.WithRecordFindQuery("name", "seed"), extensions.WithRecordFindQuery("x_seed", map[string]any{"key": nil}))
	if err != nil || count.Load() != 1 || value == nil || string(value.Envelope) != body || value.StatusCode != 203 {
		t.Fatal(value, err, count.Load())
	}
	extensionRecordField(t, value, "id", `"literal/name %2F"`)
	extensionRecordField(t, value, "alias", `"wire-alias"`)
	extensionRecordField(t, value, "name", `"seed"`)
	extensionRecordField(t, value, "x_seed", `{"key":null}`)
	value.Resource.Body["x_new"][0] = 'X'
	value.Header.Set("X-Request-Id", "consumer")
	value.Envelope[0] = 'X'
	if string(value.Wire.Body["x_new"]) != `{"n":9007199254740993}` || value.Wire.Header.Get("X-Request-Id") != "extension-actual" || value.Resource.Header.Get("X-Request-Id") != "extension-actual" {
		t.Fatal(value)
	}
	null, calls := recordExtensionAPI(t, func(w http.ResponseWriter, _ *http.Request) {
		testcloud.JSON(w, 200, `{"extension":{"id":null,"alias":"wire","links":null}}`)
	})
	value, err = null.FindRecord(context.Background(), "requested", extensions.WithRecordFindQuery("name", nil))
	if err != nil || calls.Load() != 1 {
		t.Fatal(value, err, calls.Load())
	}
	extensionRecordField(t, value, "id", "null")
	extensionRecordField(t, value, "name", "null")
	extensionRecordField(t, value, "links", "null")
	for _, body := range []string{"", "not-json"} {
		empty, calls := recordExtensionAPI(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, 200, body) })
		value, err := empty.FindRecord(context.Background(), "requested")
		if err != nil || value == nil || value.Wire != nil || calls.Load() != 1 {
			t.Fatal(value, err, calls.Load())
		}
		extensionRecordField(t, value, "id", `"requested"`)
	}
}

func TestExtensionFindAliasNameFallbackHasNoServerNameHintAndDetectsLateAmbiguity(t *testing.T) {
	for _, status := range []int{400, 403, 404} {
		for _, identity := range []string{"qos", "Name"} {
			api, count := recordExtensionAPI(t, func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path != "/neutron/v2.0/extensions" {
					testcloud.JSON(w, status, `{"error":"member"}`)
					return
				}
				if req.URL.RawQuery != "" {
					t.Error("unexpected name hint", req.URL)
				}
				testcloud.JSON(w, 200, `{"extensions":[{"alias":"qos","name":"Name"},{"alias":"other","name":"Other"}]}`)
			})
			value, err := api.FindRecord(context.Background(), identity)
			if err != nil || value == nil || count.Load() != 2 {
				t.Fatal(status, identity, value, err, count.Load())
			}
			extensionRecordField(t, value, "id", `"qos"`)
		}
	}
	ambiguous, count := recordExtensionAPI(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.URL.Path != "/neutron/v2.0/extensions":
			testcloud.JSON(w, 404, `{}`)
		case req.URL.Query().Get("marker") == "later":
			testcloud.JSON(w, 200, `{"extensions":[{"alias":"qos","name":"Other"}]}`)
		default:
			testcloud.JSON(w, 200, `{"extensions":[{"alias":"first","name":"qos"}],"next":"/v2.0/extensions?marker=later"}`)
		}
	})
	value, err := ambiguous.FindRecord(context.Background(), "qos")
	if value != nil || !errors.Is(err, resource.ErrAmbiguous) || count.Load() != 3 {
		t.Fatal(value, err, count.Load())
	}
}

func TestExtensionFindMissingForbiddenAndLocalBodyFiltersPreserveTheirPhases(t *testing.T) {
	missing, count := recordExtensionAPI(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/neutron/v2.0/extensions" {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		testcloud.JSON(w, 200, `{"extensions":[]}`)
	})
	value, err := missing.FindRecord(context.Background(), "missing")
	if value != nil || err != nil || count.Load() != 2 {
		t.Fatal(value, err, count.Load())
	}
	value, err = missing.FindRecord(context.Background(), "missing", extensions.WithRecordFindIgnoreMissing(false))
	if value != nil || !errors.Is(err, resource.ErrNotFound) || count.Load() != 4 {
		t.Fatal(value, err, count.Load())
	}
	forbidden, calls := recordExtensionAPI(t, func(w http.ResponseWriter, req *http.Request) { testcloud.JSON(w, 403, `{"error":"denied"}`) })
	value, err = forbidden.FindRecord(context.Background(), "qos")
	if value != nil || !gophercloud.ResponseCodeIs(err, 403) || calls.Load() != 2 {
		t.Fatal(value, err, calls.Load())
	}
	filtered, calls := recordExtensionAPI(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/neutron/v2.0/extensions" {
			if req.URL.Query().Get("name") != "Wanted" || req.URL.Query().Get("updated_at") != "literal" {
				t.Error(req.URL)
			}
			testcloud.JSON(w, 404, `{}`)
			return
		}
		if req.URL.RawQuery != "" {
			t.Error(req.URL)
		}
		testcloud.JSON(w, 200, `{"extensions":[{"alias":"qos","name":"Other","updated":"literal"},{"alias":"qos","name":"Wanted","updated":"literal"}]}`)
	})
	value, err = filtered.FindRecord(context.Background(), "qos", extensions.WithRecordFindQuery("name", "Wanted"), extensions.WithRecordFindQuery("updated_at", "literal"))
	if err != nil || value == nil || calls.Load() != 2 {
		t.Fatal(value, err, calls.Load())
	}
	extensionRecordField(t, value, "name", `"Wanted"`)
}

func TestExtensionFindListBasePathAndGetOnlyVersionAreIndependent(t *testing.T) {
	var paths, versions []string
	api, count := recordExtensionAPI(t, func(w http.ResponseWriter, req *http.Request) {
		paths = append(paths, req.URL.Path)
		versions = append(versions, req.Header.Get("OpenStack-API-Version"))
		if req.Header.Get("X-Trace") != "owned" {
			t.Error(req.Header)
		}
		if req.URL.Path != "/neutron/v2.0/alternate-extensions" {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		if req.URL.RawQuery != "" {
			t.Error(req.URL)
		}
		testcloud.JSON(w, 200, `{"extensions":[{"alias":"qos"}]}`)
	})
	api.RawClient().Microversion = "2.0"
	path := "/alternate-extensions"
	options := extensions.WithRecordFindOpts(extensions.RecordFindOpts{ListBasePath: &path})
	path = "/changed"
	value, err := api.FindRecord(context.Background(), "qos", options, extensions.WithRecordFindMicroversion("2.1"), extensions.WithRecordFindHeaders(map[string]string{"X-Trace": "owned"}))
	if err != nil || value == nil || count.Load() != 2 || !reflect.DeepEqual(paths, []string{"/neutron/v2.0/extensions/qos", "/neutron/v2.0/alternate-extensions"}) || !reflect.DeepEqual(versions, []string{"network 2.1", "network 2.0"}) || api.RawClient().Microversion != "2.0" {
		t.Fatal(value, err, count.Load(), paths, versions)
	}
	callsBefore := count.Load()
	value, err = api.FindRecord(context.Background(), "qos", extensions.WithRecordFindQuery("list_base_path", "/alternate-extensions"), extensions.WithRecordFindQuery("microversion", "2.1"), extensions.WithRecordFindHeaders(map[string]string{"X-Trace": "owned"}))
	if err != nil || value == nil || count.Load() != callsBefore+2 {
		t.Fatal(value, err, count.Load())
	}
}

func TestExtensionListAlternateMarkerPrivateProofAndRawCaps(t *testing.T) {
	var queries []string
	api, count := recordExtensionAPI(t, func(w http.ResponseWriter, req *http.Request) {
		queries = append(queries, req.URL.RawQuery)
		switch req.URL.Query().Get("marker") {
		case "first":
			testcloud.JSON(w, 200, `{"extensions":[{"alias":"second"}]}`)
		case "second":
			testcloud.JSON(w, 200, `{"extensions":[]}`)
		default:
			testcloud.JSON(w, 200, `{"extensions":[{"alias":"first"}]}`)
		}
	})
	rows, err := api.AllRecords(context.Background(), extensions.WithRecordListLimit(1))
	if err != nil || len(rows) != 2 || count.Load() != 3 || !reflect.DeepEqual(queries, []string{"limit=1", "limit=1&marker=first", "limit=1&marker=second"}) {
		t.Fatal(rows, err, count.Load(), queries)
	}
	capped, calls := recordExtensionAPI(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get("limit") != "2" {
			t.Error(req.URL)
		}
		testcloud.JSON(w, 203, `{"extensions":[{"alias":"skip"},{"alias":"keep"},{"cls":null}],"next":"https://foreign.invalid"}`)
	})
	rows, err = capped.AllRecords(context.Background(), extensions.WithRecordListMaxItems(2), extensions.WithRecordListQuery("alias", "keep"))
	if err != nil || len(rows) != 1 || calls.Load() != 1 {
		t.Fatal(rows, err, calls.Load())
	}
	for _, rawID := range []string{`null`, `""`} {
		body := `{"extensions":[{"id":` + rawID + `,"alias":"fallback"}]}`
		invalid, calls := recordExtensionAPI(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, 203, body) })
		rows, err := invalid.AllRecords(context.Background(), extensions.WithRecordListLimit(1))
		if rows != nil || err == nil || calls.Load() != 1 {
			t.Fatal(rows, err, calls.Load())
		}
		extensionRecordProof(t, err, 203, body)
	}
}

func TestExtensionRecordsSourceLinkPrecedenceAndActualMalformedProof(t *testing.T) {
	for _, tc := range []struct {
		name, fields, marker string
		fail                 bool
	}{
		{"root empty shadows plural", `"links":[],"extensions_links":[{"rel":"next","href":"?marker=ignored"}]`, "", false},
		{"dictionary next ignored", `"links":{"next":"?marker=ignored"}`, "", false},
		{"plural list follows", `"extensions_links":[{"rel":"next","href":"?marker=plural"}]`, "plural", false},
		{"first literal wins", `"links":[{"rel":"next","href":"?marker=first"},{"rel":"next","href":"?marker=second"}],"next":true`, "first", false},
		{"case sensitive relation", `"links":[{"rel":"NEXT","href":"?marker=ignored"}],"next":"?marker=top"`, "top", false},
		{"actual null iterable error", `"links":null`, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"extensions":[{"alias":"first"}],` + tc.fields + `}`
			api, count := recordExtensionAPI(t, func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Query().Get("marker") == "" {
					testcloud.JSON(w, 203, body)
					return
				}
				if req.URL.Query().Get("marker") != tc.marker {
					t.Error(req.URL)
				}
				testcloud.JSON(w, 200, `{"extensions":[]}`)
			})
			rows, err := api.AllRecords(context.Background())
			if tc.fail {
				if rows != nil || err == nil || count.Load() != 1 {
					t.Fatal(rows, err, count.Load())
				}
				extensionRecordProof(t, err, 203, body)
				return
			}
			calls := int32(1)
			if tc.marker != "" {
				calls = 2
			}
			if err != nil || len(rows) != 1 || count.Load() != calls {
				t.Fatal(rows, err, count.Load())
			}
		})
	}
}

func TestExtensionProxyJMESGeneratorResultsAndSourceConflictControls(t *testing.T) {
	api, count := recordExtensionAPI(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/neutron/v2.0/alternate-extensions" || req.Header.Get("X-Trace") != "owned" || req.Header.Get("OpenStack-API-Version") != "network 2.1" || req.URL.RawQuery != "" {
			t.Error(req.URL, req.Header)
		}
		testcloud.JSON(w, 200, `{"extensions":[{"alias":"qos"}]}`)
	})
	for _, tc := range []struct{ expression, want string }{{"[*].alias", "null"}, {"@ == @", "true"}, {"`{\"ok\":true}`", `{"ok":true}`}} {
		result, err := api.ProxyListRecords(context.Background(), extensions.WithRecordListJMESPath(tc.expression))
		if err != nil || result == nil || string(result.JSON) != tc.want || count.Load() != 0 {
			t.Fatal(result, err, count.Load())
		}
	}
	if result, err := api.ProxyListRecords(context.Background(), extensions.WithRecordListJMESPath("length(@)")); result != nil || err == nil || count.Load() != 0 {
		t.Fatal(result, err, count.Load())
	}
	result, err := api.ProxyListRecords(context.Background(), extensions.WithRecordListJMESPath("to_array(@)"), extensions.WithRecordListBasePath("/alternate-extensions"), extensions.WithRecordListConflictingAttrs(json.RawMessage(`{"headers":{"X-Trace":"owned"},"microversion":"2.1","jmespath_filters":"[?"}`)))
	if err != nil || result == nil || result.JSON != nil || count.Load() != 0 {
		t.Fatal(result, err, count.Load())
	}
	items, ok := result.Value.([]any)
	if !ok || len(items) != 1 {
		t.Fatal(result.Value)
	}
	generator, ok := items[0].(*extensions.RecordListGenerator)
	if !ok {
		t.Fatal(items)
	}
	for value, err := range generator.Records {
		if err != nil {
			t.Fatal(err)
		}
		extensionRecordField(t, value, "id", `"qos"`)
	}
	if count.Load() != 1 {
		t.Fatal(count.Load())
	}
}

func TestExtensionRecordsSnapshotsNilMapsConcurrentReuseAndPreflight(t *testing.T) {
	api, count := recordExtensionAPI(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("X-Trace") != "owned" {
			t.Error(req.Header)
		}
		testcloud.JSON(w, 200, `{"extensions":[{"alias":"qos"}]}`)
	})
	header := map[string]string{"X-Trace": "owned"}
	option := extensions.WithRecordListHeaders(header)
	header["X-Trace"] = "changed"
	var workers sync.WaitGroup
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			rows, err := api.AllRecords(context.Background(), option)
			if err != nil || len(rows) != 1 {
				t.Error(rows, err)
			}
		}()
	}
	workers.Wait()
	if count.Load() != 4 {
		t.Fatal(count.Load())
	}
	rows, err := api.AllRecords(context.Background(), extensions.WithRecordListHeaders(nil), func(c *extensions.RecordListOpts) error {
		c.Headers["X-Trace"] = "owned"
		c.Query["alias"] = json.RawMessage(`"qos"`)
		return nil
	})
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	for _, options := range [][]extensions.RecordFindOption{{nil}, {extensions.WithRecordFindQuery("id", make(chan int))}, {extensions.WithRecordFindQuery("ignore_missing", make(chan int))}, {extensions.WithRecordFindQuery("name_or_id", make(chan int))}, {extensions.WithRecordFindHeaders(map[string]string{"X-Auth-Token": "foreign"})}} {
		before := count.Load()
		value, err := api.FindRecord(context.Background(), "qos", options...)
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != before {
			t.Fatal(value, err, count.Load())
		}
	}
	for _, options := range [][]extensions.RecordListOption{{nil}, {extensions.WithRecordListQuery("self", make(chan int))}, {extensions.WithRecordListConflictingAttrs(json.RawMessage(`{"paginated":false}`))}, {extensions.WithRecordListBasePath("https://foreign.invalid")}} {
		before := count.Load()
		rows, err := api.AllRecords(context.Background(), options...)
		if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != before {
			t.Fatal(rows, err, count.Load())
		}
	}
}

func TestExtensionRecordsMalformedEnvelopeAndLateErrorsKeepActualProof(t *testing.T) {
	for _, body := range []string{`{}`, `{"extensions":null}`, `{"extensions":[null]}`, `{"extensions":[{"connection":null}]}`, `{"extensions":[{"microversion":null}]}`} {
		invalid, count := recordExtensionAPI(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, 203, body) })
		rows, err := invalid.AllRecords(context.Background())
		if rows != nil || err == nil || count.Load() != 1 {
			t.Fatal(rows, err, count.Load())
		}
		extensionRecordProof(t, err, 203, body)
	}
	late, count := recordExtensionAPI(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get("marker") == "late" {
			testcloud.JSON(w, 403, `{"error":"late"}`)
			return
		}
		testcloud.JSON(w, 200, `{"extensions":[{"alias":"first"}],"next":"/v2.0/extensions?marker=late"}`)
	})
	rows, err := late.AllRecords(context.Background())
	if rows != nil || !gophercloud.ResponseCodeIs(err, 403) || count.Load() != 2 {
		t.Fatal(rows, err, count.Load())
	}
}

type extensionObservedBody struct {
	io.ReadCloser
	readErr, closeErr error
	closed            *atomic.Int32
	onClose           func()
}

func (b *extensionObservedBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if n > 0 && b.readErr != nil {
		return n, b.readErr
	}
	return n, err
}
func (b *extensionObservedBody) Close() error {
	b.closed.Add(1)
	err := b.ReadCloser.Close()
	if b.onClose != nil {
		b.onClose()
	}
	return errors.Join(err, b.closeErr)
}

type extensionObservedTransport struct {
	base              http.RoundTripper
	readErr, closeErr error
	closed            *atomic.Int32
	onClose           func()
}

func (t extensionObservedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(req)
	if response != nil {
		response.Body = &extensionObservedBody{ReadCloser: response.Body, readErr: t.readErr, closeErr: t.closeErr, closed: t.closed, onClose: t.onClose}
	}
	return response, err
}
func TestExtensionRecordsPhysicalFaultsNeverFallbackRetryOrReauth(t *testing.T) {
	for _, op := range []string{"find", "list"} {
		for _, status := range []int{200, 401, 404, 503} {
			for _, fault := range []string{"read-close", "cancel", "source"} {
				t.Run(fmt.Sprintf("%s/%d/%s", op, status, fault), func(t *testing.T) {
					body := `{"extension":{"alias":"qos"}}`
					if op == "list" {
						body = `{"extensions":[{"alias":"qos"}]}`
					}
					api, count := recordExtensionAPI(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, status, body) })
					readErr, closeErr, cancelErr := errors.New("read"), errors.New("close"), errors.New("cancel")
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					closed := &atomic.Int32{}
					base := api.RawClient().HTTPClient.Transport
					if base == nil {
						base = http.DefaultTransport
					}
					observed := extensionObservedTransport{base: base, closed: closed}
					switch fault {
					case "read-close":
						observed.readErr, observed.closeErr = readErr, closeErr
					case "cancel":
						observed.onClose = func() { cancel(cancelErr) }
					case "source":
						observed.onClose = func() { api.RawClient().ResourceBase += "changed/" }
					}
					api.RawClient().HTTPClient.Transport = observed
					retries, reauths := 0, 0
					api.RawClient().ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
						retries++
						return nil
					}
					api.RawClient().ProviderClient.ReauthFunc = func(context.Context) error { reauths++; return nil }
					var err error
					if op == "find" {
						_, err = api.FindRecord(ctx, "qos")
					} else {
						_, err = api.AllRecords(ctx)
					}
					if err == nil || count.Load() != 1 || closed.Load() != 1 || retries != 0 || reauths != 0 {
						t.Fatal(err, count.Load(), closed.Load(), retries, reauths)
					}
					switch fault {
					case "read-close":
						if !errors.Is(err, readErr) || !errors.Is(err, closeErr) {
							t.Fatal(err)
						}
					case "cancel":
						if !errors.Is(err, context.Canceled) || !errors.Is(err, cancelErr) {
							t.Fatal(err)
						}
					case "source":
						if !errors.Is(err, resource.ErrInvalidOption) {
							t.Fatal(err)
						}
					}
					if status < 400 {
						extensionRecordProof(t, err, status, body)
					} else {
						receipt := err
						if status == 401 {
							var auth *gophercloud.ErrUnableToReauthenticate
							if !errors.As(err, &auth) || auth.ErrOriginal == nil || auth.ErrReauth == nil {
								t.Fatal(err, auth)
							}
							receipt = auth.ErrOriginal
						}
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(receipt, &native) || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "extension-actual" {
							t.Fatal(err, native)
						}
					}
				})
			}
		}
	}
}

func TestExtensionRecordsCleanRetryTokenRenewalAndProtocolGuard(t *testing.T) {
	for _, op := range []string{"find", "list"} {
		for _, lane := range []string{"reauth", "retry", "auth-mutation"} {
			t.Run(op+"/"+lane, func(t *testing.T) {
				var tokens []string
				body := `{"extension":{"alias":"qos"}}`
				if op == "list" {
					body = `{"extensions":[{"alias":"qos"}]}`
				}
				attempt := 0
				api, count := recordExtensionAPI(t, func(w http.ResponseWriter, req *http.Request) {
					tokens = append(tokens, req.Header.Get("X-Auth-Token"))
					attempt++
					if attempt == 1 {
						status := 503
						if lane == "reauth" {
							status = 401
						}
						testcloud.JSON(w, status, `{"error":"original"}`)
						return
					}
					testcloud.JSON(w, 200, body)
				})
				callbacks := 0
				if lane == "reauth" {
					api.RawClient().ProviderClient.ReauthFunc = func(context.Context) error {
						callbacks++
						api.RawClient().ProviderClient.SetToken("renewed")
						return nil
					}
				} else {
					api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, options *gophercloud.RequestOpts, _ error, _ uint) error {
						callbacks++
						if lane == "auth-mutation" {
							options.MoreHeaders["Authorization"] = "foreign"
						}
						return nil
					}
				}
				var err error
				if op == "find" {
					_, err = api.FindRecord(context.Background(), "qos")
				} else {
					_, err = api.AllRecords(context.Background())
				}
				if lane == "auth-mutation" {
					if !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || count.Load() != 1 || callbacks != 1 {
						t.Fatal(err, count.Load(), callbacks)
					}
				} else {
					if err != nil || count.Load() != 2 || callbacks != 1 {
						t.Fatal(err, count.Load(), callbacks)
					}
					if lane == "reauth" && !reflect.DeepEqual(tokens, []string{"test-token", "renewed"}) {
						t.Fatal(tokens)
					}
				}
			})
		}
	}
}
