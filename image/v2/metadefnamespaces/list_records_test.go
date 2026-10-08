package metadefnamespaces

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

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func namespaceRecordRows(t *testing.T, body string, options ...RecordListOption) ([]*Record, error) {
	t.Helper()
	api := New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) { return namespaceCoreJSON(req, 203, body), nil }))
	return api.AllRecords(context.Background(), append(options, WithRecordListPaginated(false))...)
}
func namespaceRecordField(t *testing.T, row *Record, key, want string) {
	t.Helper()
	got := row.Resource.Body[key]
	var left, right any
	if err := json.Unmarshal(got, &left); err != nil {
		t.Fatalf("field %s invalid: %q", key, got)
	}
	if err := json.Unmarshal([]byte(want), &right); err != nil {
		t.Fatal(err)
	}
	if right == nil && left == nil {
		return
	}
	th.CheckDeepEquals(t, right, left)
}

func TestMetadefNamespaceRecordsDeclaredProjectionAndOwnedChannels(t *testing.T) {
	body := `{"namespaces":[{"namespace":{"raw":9007199254740993},"name":false,"created_at":[1],"description":{"any":null},"display_name":7,"protected":"false","owner":null,"resource_type_associations":[{"name":"OS::A","future":9007199254740993},"literal",null,[],false],"updated_at":true,"visibility":{},"tags":"one","location":{"foreign":true},"self":"https://foreign.test/x","properties":{"extension":1},"objects":[false],"unknown":9007199254740993},{}],"next":false,"receipt":9007199254740993}`
	rows, err := namespaceRecordRows(t, body)
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows %d: %v", len(rows), err)
	}
	row := rows[0]
	th.CheckEquals(t, 13, len(row.Resource.Body))
	for key, want := range map[string]string{"id": `{"raw":9007199254740993}`, "namespace": `{"raw":9007199254740993}`, "name": `false`, "created_at": `[1]`, "description": `{"any":null}`, "display_name": `7`, "is_protected": `true`, "owner": `null`, "updated_at": `true`, "visibility": `{}`, "tags": `["one"]`, "resource_type_associations": `[{"name":"OS::A","future":9007199254740993},{},{},{},{}]`, "location": `null`} {
		namespaceRecordField(t, row, key, want)
	}
	th.CheckEquals(t, "9007199254740993", string(row.Wire.Body["unknown"]))
	th.CheckEquals(t, `"false"`, string(row.Wire.Body["protected"]))
	if row.Resource.Body["self"] != nil || row.Resource.Body["properties"] != nil || row.Resource.Body["objects"] != nil || row.Resource.Body["protected"] != nil {
		t.Fatal("undeclared wire leaked into declared view")
	}
	namespaceRecordField(t, rows[1], "tags", `[]`)
	namespaceRecordField(t, rows[1], "resource_type_associations", `null`)
	if string(row.Envelope) != body || row.StatusCode != 203 || row.Resource.StatusCode != 203 || row.Wire.StatusCode != 203 {
		t.Fatal("lost whole page/status")
	}
	row.Resource.Body["namespace"][2] = 'X'
	row.Wire.Body["unknown"][0] = '1'
	row.Envelope[2] = 'X'
	row.Header.Set("X-Proof", "row")
	row.Resource.Header.Set("X-Proof", "resource")
	row.Wire.Header.Set("X-Proof", "wire")
	if string(row.Wire.Body["namespace"]) != `{"raw":9007199254740993}` || rows[1].Header.Get("X-Proof") != "original" || rows[1].Resource.Header.Get("X-Proof") != "original" || string(rows[1].Envelope) != body {
		t.Fatal("result channels or sibling rows alias")
	}
}

func TestMetadefNamespaceRecordsDescriptorNullDefaultAndAliasOrder(t *testing.T) {
	cases := []struct{ name, wire, field, want string }{
		{"bool missing", `{}`, "is_protected", `null`}, {"bool null", `{"protected":null}`, "is_protected", `null`},
		{"bool false", `{"protected":false}`, "is_protected", `false`}, {"bool zero", `{"protected":0}`, "is_protected", `false`},
		{"bool empty string", `{"protected":""}`, "is_protected", `false`}, {"bool empty array", `{"protected":[]}`, "is_protected", `false`},
		{"bool empty object", `{"protected":{}}`, "is_protected", `false`}, {"bool negative", `{"protected":-1}`, "is_protected", `true`},
		{"bool string", `{"protected":"false"}`, "is_protected", `true`}, {"bool nonempty array", `{"protected":[null]}`, "is_protected", `true`},
		{"bool canonical later", `{"protected":false,"is_protected":"truthy"}`, "is_protected", `true`},
		{"bool remote later", `{"is_protected":"truthy","protected":false}`, "is_protected", `false`},
		{"bool duplicate first position", `{"protected":false,"is_protected":false,"protected":true}`, "is_protected", `false`},
		{"tags missing", `{}`, "tags", `[]`}, {"tags null", `{"tags":null}`, "tags", `null`},
		{"tags empty", `{"tags":[]}`, "tags", `[]`}, {"tags object", `{"tags":{"name":"n"}}`, "tags", `[{"name":"n"}]`},
		{"tags array untyped", `{"tags":[1,false,null,{"name":"n"}]}`, "tags", `[1,false,null,{"name":"n"}]`},
		{"tags emptystring", `{"tags":""}`, "tags", `[""]`},
		{"associations missing", `{}`, "resource_type_associations", `null`}, {"associations null", `{"resource_type_associations":null}`, "resource_type_associations", `null`},
		{"associations empty", `{"resource_type_associations":[]}`, "resource_type_associations", `[]`},
		{"associations singleton dict", `{"resource_type_associations":{"name":"OS::A"}}`, "resource_type_associations", `[{"name":"OS::A"}]`},
		{"associations scalar", `{"resource_type_associations":0}`, "resource_type_associations", `[{}]`},
		{"id missing", `{"namespace":"OS::A"}`, "id", `"OS::A"`}, {"id null", `{"id":null,"namespace":"OS::A"}`, "id", `null`},
		{"id empty", `{"id":"","namespace":"OS::A"}`, "id", `""`}, {"id arbitrary", `{"id":false,"namespace":"OS::A"}`, "id", `false`},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := namespaceRecordRows(t, `{"namespaces":`+tt.wire+`}`)
			if err != nil || len(rows) != 1 {
				t.Fatalf("%v", err)
			}
			namespaceRecordField(t, rows[0], tt.field, tt.want)
		})
	}
}

func TestMetadefNamespaceRecordsActualStatusesAndMandatoryRepresentation(t *testing.T) {
	for _, status := range []int{200, 201, 202, 203, 204, 206, 226, 299, 300, 302, 304, 307, 399} {
		t.Run(fmt.Sprintf("status %d", status), func(t *testing.T) {
			body := &namespaceCoreBody{reader: strings.NewReader(`{"namespaces":{"namespace":"one"}}`)}
			client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) { return namespaceCoreHTTP(req, status, body), nil })
			client.ProviderClient.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
			rows, err := New(client).AllRecords(context.Background(), WithRecordListPaginated(false))
			if err != nil || len(rows) != 1 || rows[0].StatusCode != status || body.closes != 1 {
				t.Fatalf("status %d rows %d closes %d: %v", status, len(rows), body.closes, err)
			}
		})
	}
	bad := []string{"", `null`, `[]`, `{}`, `{"namespaces":null}`, `{"namespaces":false}`, `{"namespaces":1}`, `{"namespaces":"x"}`, `{"namespaces":[null]}`, `{"namespaces":[false]}`, `{"namespaces":[]} {}`, string(append([]byte(`{"namespaces":[],"unknown":"`), append([]byte{0xff}, []byte(`"}`)...)...))}
	for i, body := range bad {
		t.Run(fmt.Sprintf("malformed %d", i), func(t *testing.T) {
			rows, err := namespaceRecordRows(t, body)
			if len(rows) != 0 || err == nil {
				t.Fatalf("accepted %q: %v", body, err)
			}
			namespaceCoreProof(t, err, 203, body)
		})
	}
	for _, status := range []int{200, 204, 304} {
		t.Run(fmt.Sprintf("empty status %d", status), func(t *testing.T) {
			api := New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) { return namespaceCoreJSON(req, status, ""), nil }))
			rows, err := api.AllRecords(context.Background())
			if err == nil || len(rows) != 0 {
				t.Fatal("missing representation accepted")
			}
			namespaceCoreProof(t, err, status, "")
		})
	}
}

func TestMetadefNamespaceRecordsSourceQueriesAndNormalizedLocalFilters(t *testing.T) {
	calls := 0
	client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		want := url.Values{"limit": {"3"}, "marker": {"OS::starting/+?"}, "resource_types": {"OS::A", "OS::B"}, "sort_dir": {"DESC"}, "sort_key": {"vendor_future"}, "visibility": {"unexpected-server-owned"}}
		if req.Method != http.MethodGet || req.Body != nil || req.URL.EscapedPath() != "/reverse/glance/v2/metadefs/namespaces" || !reflect.DeepEqual(req.URL.Query(), want) || req.Header.Get("Accept") != "application/json" || req.Header.Get("X-Custom") != "owned" {
			t.Fatalf("request: %s %s headers %v", req.Method, req.URL, req.Header)
		}
		return namespaceCoreJSON(req, 206, `{"namespaces":[{"namespace":"one","protected":"false","tags":"one","resource_type_associations":{"name":"OS::A","extra":1}},{"namespace":"two","protected":false,"tags":[]},{"namespace":"three","protected":true,"tags":"one","resource_type_associations":[{"name":"OS::A","extra":1}]}]}`), nil
	})
	rows, err := New(client).AllRecords(context.Background(), WithRecordListLimit(3), WithRecordListMarker("OS::starting/+?"), WithRecordListPaginated(false), WithRecordListHeader("X-Custom", "owned"), WithRecordListFilters(map[string]any{
		"resource_types": []string{"OS::A", "OS::B"}, "sort_dir": "DESC", "sort_key": "vendor_future", "visibility": "unexpected-server-owned", "is_protected": true, "tags": []string{"one"}, "resource_type_associations": []any{map[string]any{"name": "OS::A", "extra": 1}}, "protected": make(chan int), "any_tags": make(chan int), "vendor": make(chan int),
	}))
	if err != nil || len(rows) != 2 || calls != 1 {
		t.Fatalf("rows %d calls %d: %v", len(rows), calls, err)
	}
	// Local dictionary filters use subset matching after descriptor conversion.
	rows, err = namespaceRecordRows(t, `{"namespaces":[{"description":{"a":1,"b":2}},{"description":{}},{"description":false}]}`, WithRecordListFilter("description", map[string]any{"a": 1}))
	if err != nil || len(rows) != 1 {
		t.Fatalf("subset filter %d: %v", len(rows), err)
	}
	rows, err = namespaceRecordRows(t, `{"namespaces":[{"protected":"false"},{"protected":true}]}`, WithRecordListFilter("is_protected", "false"))
	if err != nil || len(rows) != 0 {
		t.Fatalf("filter input coerced: %d %v", len(rows), err)
	}
	rows, err = namespaceRecordRows(t, `{"namespaces":[{}, {"tags":null}]}`, WithRecordListFilter("tags", nil))
	if err != nil || len(rows) != 1 {
		t.Fatalf("tags missing default confused with null: %d %v", len(rows), err)
	}
}

func TestMetadefNamespaceRecordsGenericPaginationAndRawMarkers(t *testing.T) {
	forms := []struct{ name, fields, header string }{
		{"next", `,"next":"/v2/metadefs/namespaces?marker=next"`, ""},
		{"list links", `,"links":[{"rel":"next","href":"?marker=next"}]`, ""},
		{"dict links", `,"links":{"next":"?marker=next"}`, ""},
		{"plural links", `,"namespaces_links":[{"rel":"next","href":"?marker=next"}]`, ""},
		{"http link", ``, `<?marker=next>; rel="next"`},
	}
	for _, tt := range forms {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			api := New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				th.CheckEquals(t, "public", req.URL.Query().Get("visibility"))
				body := `{"namespaces":[{"namespace":"first"}]` + tt.fields + `}`
				if calls == 2 {
					th.CheckEquals(t, "next", req.URL.Query().Get("marker"))
					body = `{"namespaces":[],"next":"https://foreign.test/ignored"}`
				}
				resp := namespaceCoreJSON(req, 203, body)
				if calls == 1 && tt.header != "" {
					resp.Header.Set("Link", tt.header)
				}
				return resp, nil
			}))
			rows, err := api.AllRecords(context.Background(), WithRecordListVisibility("public"))
			if err != nil || len(rows) != 1 || calls != 2 {
				t.Fatalf("calls %d rows %d: %v", calls, len(rows), err)
			}
		})
	}
	for _, tt := range []struct{ name, wire, marker string }{
		{"namespace fallback", `{"namespace":"OS::last/+?"}`, "OS::last/+?"},
		{"explicit id", `{"namespace":"ignored","id":"explicit-id"}`, "explicit-id"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			api := New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				th.CheckEquals(t, "2", req.URL.Query().Get("limit"))
				if calls == 2 {
					th.CheckEquals(t, tt.marker, req.URL.Query().Get("marker"))
					return namespaceCoreJSON(req, 200, `{"namespaces":[]}`), nil
				}
				return namespaceCoreJSON(req, 200, `{"namespaces":[`+tt.wire+`]}`), nil
			}))
			stream := api.ListRecords(context.Background(), WithRecordListLimit(2))
			rows := 0
			for row, err := range stream {
				if err != nil {
					t.Fatal(err)
				}
				rows++
				row.Resource.Body["id"] = json.RawMessage(`"caller-mutation"`)
				row.Wire.Body["namespace"] = json.RawMessage(`"caller-mutation"`)
			}
			if rows != 1 || calls != 2 {
				t.Fatalf("calls %d rows %d", calls, rows)
			}
		})
	}
	for _, wire := range []string{`{"namespace":"fallback","id":null}`, `{"namespace":"fallback","id":false}`, `{"namespace":""}`, `{}`} {
		t.Run("bad marker "+wire, func(t *testing.T) {
			body := `{"namespaces":[` + wire + `]}`
			api := New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) { return namespaceCoreJSON(req, 200, body), nil }))
			rows, err := api.AllRecords(context.Background(), WithRecordListLimit(2))
			if len(rows) != 1 || err == nil {
				t.Fatalf("rows %d: %v", len(rows), err)
			}
			namespaceCoreProof(t, err, 200, body)
		})
	}
	for _, next := range []string{"https://foreign.test/metadefs/namespaces?marker=n", "/v2/images?marker=n", "?marker=n&visibility=private", "?marker=n&vendor=x", "?marker=n&marker=m", "?marker=starting"} {
		t.Run("guard "+next, func(t *testing.T) {
			calls := 0
			body := fmt.Sprintf(`{"namespaces":[{"namespace":"one"}],"next":%q}`, next)
			api := New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return namespaceCoreJSON(req, 200, body), nil
			}))
			rows, err := api.AllRecords(context.Background(), WithRecordListMarker("starting"), WithRecordListVisibility("public"))
			if len(rows) != 1 || err == nil || calls != 1 {
				t.Fatalf("calls %d rows %d: %v", calls, len(rows), err)
			}
			namespaceCoreProof(t, err, 200, body)
		})
	}
}

func TestMetadefNamespaceRecordsCapsPartialsAndIteratorOwnership(t *testing.T) {
	body := `{"namespaces":[{"namespace":"filtered","protected":false},false],"next":false}`
	calls := 0
	api := New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		th.CheckEquals(t, "1", req.URL.Query().Get("limit"))
		return namespaceCoreJSON(req, 203, body), nil
	}))
	rows, err := api.AllRecords(context.Background(), WithRecordListMaxItems(1), WithRecordListFilter("is_protected", true))
	if err != nil || len(rows) != 0 || calls != 1 {
		t.Fatalf("raw cap filtered rows %d calls %d: %v", len(rows), calls, err)
	}
	api = New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) { return namespaceCoreJSON(req, 203, body), nil }))
	rows, err = api.AllRecords(context.Background(), WithRecordListPaginated(false))
	if err == nil || len(rows) != 1 {
		t.Fatalf("partial rows %d: %v", len(rows), err)
	}
	namespaceCoreProof(t, err, 203, body)
	consumed := 0
	for row, err := range api.ListRecords(context.Background()) {
		if err != nil || row == nil {
			t.Fatal(err)
		}
		consumed++
		break
	}
	th.CheckEquals(t, 1, consumed)
	laterCalls := 0
	first := `{"namespaces":[{"namespace":"one"}],"next":"?marker=next"}`
	api = New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
		laterCalls++
		if laterCalls == 1 {
			return namespaceCoreJSON(req, 200, first), nil
		}
		return namespaceCoreJSON(req, 200, `{"namespaces":null}`), nil
	}))
	rows, err = api.AllRecords(context.Background())
	if err == nil || len(rows) != 1 || laterCalls != 2 {
		t.Fatalf("later rows %d: %v", len(rows), err)
	}
	namespaceCoreProof(t, err, 200, `{"namespaces":null}`)
	// Reusing one stream recaptures mutable source/header state per iteration.
	headers := map[string]string{"X-Custom": "owned"}
	filters := map[string]any{"namespace": "one"}
	option := WithRecordListOpts(RecordListOpts{Headers: headers, Paginated: copyPointer(new(bool)), Filters: []resource.ListOption{resource.WithFilters(filters)}})
	headers["X-Custom"] = "caller"
	filters["namespace"] = "caller"
	var retained *RecordListOpts
	client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
		th.CheckEquals(t, "owned", req.Header.Get("X-Custom"))
		return namespaceCoreJSON(req, 200, `{"namespaces":[{"namespace":"one"}]}`), nil
	})
	stream := New(client).ListRecords(context.Background(), option, func(value *RecordListOpts) error { retained = value; return nil })
	for run := 0; run < 2; run++ {
		rows, err = collectRecords(stream)
		if err != nil || len(rows) != 1 {
			t.Fatal(err)
		}
		retained.Headers["X-Custom"] = "retained"
		retained.Filters = nil
	}
}

func TestMetadefNamespaceRecordsCurrentLocationAndStickyGuards(t *testing.T) {
	cloud := "initial"
	region := "RegionOne"
	locationCalls := 0
	httpCalls := 0
	client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
		httpCalls++
		return namespaceCoreJSON(req, 200, `{"namespaces":[{"namespace":"declared","location":{"foreign":true}},{"location":{"foreign":true}},{"tags":null,"location":{"foreign":true}},{}]}`), nil
	})
	api := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
		locationCalls++
		return resource.CloudLocation{Cloud: &cloud, RegionName: &region, Project: resource.CloudProject{ID: json.RawMessage(`"project"`)}, Zone: json.RawMessage(`"zone"`)}, nil
	}})
	rows, err := api.AllRecords(context.Background(), WithRecordListPaginated(false), func(*RecordListOpts) error { cloud = "changed-after-capture"; return nil })
	if err != nil || len(rows) != 4 || locationCalls != 1 || httpCalls != 1 {
		t.Fatalf("locations %d calls %d: %v", locationCalls, httpCalls, err)
	}
	for _, i := range []int{0, 2, 3} {
		namespaceRecordField(t, rows[i], "location", `{"cloud":"initial","region_name":"RegionOne","zone":"zone","project":{"id":"project","name":null,"domain_id":null,"domain_name":null}}`)
	}
	namespaceRecordField(t, rows[1], "location", `{"foreign":true}`)
	// Observe source mutation inside an option, then restore it: sticky failure
	// prevents the next callback and every physical request.
	calls := 0
	client = namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		return namespaceCoreJSON(req, 200, `{"namespaces":[]}`), nil
	})
	api = New(client)
	original := client.Endpoint
	ctx := rest.WithOperationGuard(context.Background(), func(context.Context) error {
		if client.Endpoint != original {
			return resource.ErrInvalidOption
		}
		return nil
	})
	ran := 0
	rows, err = api.AllRecords(ctx, func(*RecordListOpts) error { client.Endpoint = "https://changed.test/"; return nil }, func(*RecordListOpts) error { ran++; client.Endpoint = original; return nil })
	client.Endpoint = original
	if err == nil || len(rows) != 0 || ran != 0 || calls != 0 {
		t.Fatalf("sticky options ran%d calls%d: %v", ran, calls, err)
	}
	// A guard error during body read remains proved after Close restores source.
	fault := errors.New("read guard")
	bodyText := `{"namespaces":[{"namespace":"one"}]}`
	var failing bool
	reader := &namespaceCoreReader{data: bodyText, err: io.EOF, after: func() { failing = true }}
	body := &namespaceRecordRestoringBody{namespaceCoreBody: &namespaceCoreBody{reader: reader}, after: func() { failing = false }}
	client = namespaceCoreClient(func(req *http.Request) (*http.Response, error) { return namespaceCoreHTTP(req, 206, body), nil })
	ctx = rest.WithOperationGuard(context.Background(), func(context.Context) error {
		if failing {
			return fault
		}
		return nil
	})
	rows, err = New(client).AllRecords(ctx)
	if len(rows) != 0 || !errors.Is(err, fault) || body.closes != 1 || failing {
		t.Fatalf("guard receipt closes%d: %v", body.closes, err)
	}
	namespaceCoreProof(t, err, 206, bodyText)
}

type namespaceRecordRestoringBody struct {
	*namespaceCoreBody
	after func()
}

func (b *namespaceRecordRestoringBody) Close() error {
	err := b.namespaceCoreBody.Close()
	if b.after != nil {
		b.after()
	}
	return err
}

func TestMetadefNamespaceRecordsCompletePreflightAndFaultReceipts(t *testing.T) {
	cases := []struct {
		name   string
		option RecordListOption
	}{
		{"nil", nil}, {"negative limit", WithRecordListLimit(-1)}, {"negative cap", WithRecordListMaxItems(-1)},
		{"empty marker", WithRecordListFilter("marker", "")}, {"marker controls", WithRecordListMarker("line\n")},
		{"limit negative semantic", WithRecordListFilter("limit", -1)}, {"bad limit", WithRecordListFilter("limit", []int{1, 2})},
		{"query controls", WithRecordListFilter("resource_types", "bad\n")}, {"known encoding", WithRecordListFilter("namespace", make(chan int))},
		{"raw option wrong namespace", func(c *RecordListOpts) error {
			c.Filters = []resource.ListOption{resource.WithQuery("vendor", "x")}
			return nil
		}},
		{"managed header", WithRecordListHeader("Authorization", "x")}, {"invalid header", WithRecordListHeader("X-Valid", "bad\n")},
		{"header aliases", WithRecordListHeaders(map[string]string{"x-h": "one", "X-H": "two"})},
	}
	for _, control := range []string{"resource_type", "session", "paginated", "base_path", "microversion", "headers", "max_items", "allow_unknown_params", "jmespath_filters", "__conflicting_attrs"} {
		cases = append(cases, struct {
			name   string
			option RecordListOption
		}{"reserved " + control, WithRecordListFilter(control, true)})
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return namespaceCoreJSON(req, 200, `{"namespaces":[]}`), nil
			})
			rows, err := New(client).AllRecords(context.Background(), tt.option)
			if err == nil || calls != 0 || len(rows) != 0 {
				t.Fatalf("calls %d: %v", calls, err)
			}
		})
	}
	for _, name := range []string{"nil API", "nil context", "canceled", "wrong type", "source token header", "location error", "location JSON", "outer guard"} {
		t.Run(name, func(t *testing.T) {
			calls, callbacks := 0, 0
			client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return namespaceCoreJSON(req, 200, `{"namespaces":[]}`), nil
			})
			api := New(client)
			ctx := context.Background()
			sentinel := errors.New(name)
			switch name {
			case "nil API":
				api = nil
			case "nil context":
				ctx = nil
			case "canceled":
				var cancel context.CancelCauseFunc
				ctx, cancel = context.WithCancelCause(ctx)
				cancel(sentinel)
			case "wrong type":
				client.Type = "compute"
			case "source token header":
				client.MoreHeaders = map[string]string{"X-Auth-Token": "foreign"}
			case "location error":
				api = NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { return resource.CloudLocation{}, sentinel }})
			case "location JSON":
				api = NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { return resource.CloudLocation{Zone: json.RawMessage(`{`)}, nil }})
			case "outer guard":
				ctx = rest.WithOperationGuard(ctx, func(context.Context) error { return sentinel })
			}
			rows, err := api.AllRecords(ctx, func(*RecordListOpts) error { callbacks++; return nil })
			if err == nil || calls != 0 || callbacks != 0 || len(rows) != 0 {
				t.Fatalf("callbacks %d calls %d: %v", callbacks, calls, err)
			}
			if name == "canceled" && (!errors.Is(err, context.Canceled) || !errors.Is(err, sentinel)) {
				t.Fatal("lost cancel cause")
			}
		})
	}
	for _, name := range []string{"read", "close", "cancel read", "source read", "source close"} {
		t.Run(name, func(t *testing.T) {
			bodyText := `{"namespaces":[{"namespace":"one"}]}`
			fault := errors.New(name)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			reader := &namespaceCoreReader{data: bodyText, err: io.EOF}
			base := &namespaceCoreBody{reader: reader}
			body := &namespaceRecordRestoringBody{namespaceCoreBody: base}
			var client *gophercloud.ServiceClient
			client = namespaceCoreClient(func(req *http.Request) (*http.Response, error) { return namespaceCoreHTTP(req, 299, body), nil })
			switch name {
			case "read":
				reader.err = fault
			case "close":
				base.closeErr = fault
			case "cancel read":
				reader.after = func() { cancel(fault) }
			case "source read":
				reader.after = func() { client.Endpoint = "https://changed.test/" }
			case "source close":
				body.after = func() { client.Endpoint = "https://changed.test/" }
			}
			rows, err := New(client).AllRecords(ctx)
			if len(rows) != 0 || err == nil || base.closes != 1 {
				t.Fatalf("rows%d closes%d: %v", len(rows), base.closes, err)
			}
			if (name == "read" || name == "close" || name == "cancel read") && !errors.Is(err, fault) {
				t.Fatal("lost fault")
			}
			namespaceCoreProof(t, err, 299, bodyText)
		})
	}
	// Explicit semantic/typed collisions are rejected even when both are valid.
	calls := 0
	client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		return namespaceCoreJSON(req, 200, `{"namespaces":[]}`), nil
	})
	rows, err := New(client).AllRecords(context.Background(), WithRecordListLimit(1), WithRecordListFilter("limit", 1))
	if err == nil || calls != 0 || len(rows) != 0 {
		t.Fatal("semantic control collision sent HTTP")
	}
}

func TestMetadefNamespaceRecordsExplicitZeroLimitAndBindingCollisions(t *testing.T) {
	for _, option := range []RecordListOption{WithRecordListLimit(0), WithRecordListFilter("limit", 0)} {
		t.Run("zero no fallback", func(t *testing.T) {
			calls := 0
			api := New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if !req.URL.Query().Has("limit") || req.URL.Query().Get("limit") != "0" {
					t.Fatalf("missing explicit zero: %s", req.URL)
				}
				return namespaceCoreJSON(req, 200, `{"namespaces":[{"namespace":"one"}]}`), nil
			}))
			rows, err := api.AllRecords(context.Background(), option)
			if err != nil || len(rows) != 1 || calls != 1 {
				t.Fatalf("calls%d rows%d: %v", calls, len(rows), err)
			}
		})
		t.Run("zero advertised versioned continuation", func(t *testing.T) {
			calls := 0
			api := New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				th.CheckEquals(t, "0", req.URL.Query().Get("limit"))
				th.CheckEquals(t, "/reverse/glance/v2/metadefs/namespaces", req.URL.EscapedPath())
				if calls == 1 {
					return namespaceCoreJSON(req, 200, `{"namespaces":[{}],"next":"/v2/metadefs/namespaces?limit=0&marker=next"}`), nil
				}
				th.CheckEquals(t, "next", req.URL.Query().Get("marker"))
				return namespaceCoreJSON(req, 200, `{"namespaces":[]}`), nil
			}))
			rows, err := api.AllRecords(context.Background(), option)
			if err != nil || len(rows) != 1 || calls != 2 {
				t.Fatalf("calls%d rows%d: %v", calls, len(rows), err)
			}
		})
	}
	for _, tt := range []struct {
		name   string
		option RecordListOption
		want   string
	}{
		{"typed falsey zero hint", WithRecordListLimit(0), "2"},
		{"semantic falsey zero hint", WithRecordListFilter("limit", 0), "2"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			api := New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
				th.CheckEquals(t, tt.want, req.URL.Query().Get("limit"))
				return namespaceCoreJSON(req, 200, `{"namespaces":[{},{}],"next":false}`), nil
			}))
			rows, err := api.AllRecords(context.Background(), tt.option, WithRecordListMaxItems(2))
			if err != nil || len(rows) != 2 {
				t.Fatalf("rows%d: %v", len(rows), err)
			}
		})
	}
	// The pointer-valued option snapshots explicit zero independently.
	zero := 0
	headers := map[string]string{"X-Owned": "true"}
	option := WithRecordListOpts(RecordListOpts{Limit: &zero, Headers: headers})
	zero = 8
	headers["X-Owned"] = "caller"
	api := New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
		th.CheckEquals(t, "0", req.URL.Query().Get("limit"))
		th.CheckEquals(t, "true", req.Header.Get("X-Owned"))
		return namespaceCoreJSON(req, 200, `{"namespaces":[]}`), nil
	}))
	if _, err := api.AllRecords(context.Background(), option); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"connection", "_synchronized", "microversion"} {
		t.Run("row binding "+key, func(t *testing.T) {
			body := fmt.Sprintf(`{"namespaces":[{"namespace":"one"},{%q:null}]}`, key)
			rows, err := namespaceRecordRows(t, body)
			if len(rows) != 1 || err == nil {
				t.Fatalf("rows%d: %v", len(rows), err)
			}
			namespaceCoreProof(t, err, 203, body)
		})
	}
	body := `{"namespaces":[{"namespace":"one","base_path":"https://foreign.test","resource_type":false,"session":{},"__conflicting_attrs":{"namespace":"bad"},"self":"foreign"}]}`
	rows, err := namespaceRecordRows(t, body)
	if err != nil || len(rows) != 1 {
		t.Fatal(err)
	}
	namespaceRecordField(t, rows[0], "namespace", `"one"`)
	if len(rows[0].Resource.Body) != 13 || string(rows[0].Wire.Body["base_path"]) != `"https://foreign.test"` {
		t.Fatal("unknown constructor fields became policy")
	}
}
