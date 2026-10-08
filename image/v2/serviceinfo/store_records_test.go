package serviceinfo

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

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestStoreRecordsProjectionAndOwnedReceipts(t *testing.T) {
	const row = `{"id":900719925474099312345,"name":["passive",null],"description":{"future":true},"default":"false","properties":{"number":1.00000000000000000001},"type":[1],"read-only":false,"weight":1e400,"location":{"cloud":"wire"},"self":"https://foreign.test/","vendor":false}`
	const raw = `{"stores":[` + row + `,{}],"next":false}`
	calls := 0
	var actualHeader http.Header
	client := infoClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodGet || req.URL.String() != "https://example.test/reverse/glance/v2/info/stores" || req.Body != nil || req.Header.Get("Accept") != "application/json" {
			t.Fatal("basic bodyless discovery", req.Method, req.URL, req.Body, req.Header)
		}
		wire := infoRecordJSON(req, 203, raw)
		actualHeader = wire.Header
		return wire, nil
	})
	rows, err := New(client).AllStoreRecords(context.Background())
	if err != nil || len(rows) != 2 || calls != 1 {
		t.Fatal(rows, err, calls)
	}
	want := map[string]string{"id": "900719925474099312345", "name": `["passive",null]`, "description": `{"future":true}`, "is_default": "true", "properties": `{"number":1.00000000000000000001}`, "location": "null"}
	th.CheckDeepEquals(t, want, infoRecordValues(rows[0].Resource))
	for _, record := range rows {
		if record.Resource == nil || record.Wire == nil || record.StatusCode != 203 || record.Resource.StatusCode != 203 || record.Wire.StatusCode != 203 || record.Header.Get("X-Actual") != "owned" || record.Resource.Header.Get("X-Actual") != "owned" || record.Wire.Header.Get("X-Actual") != "owned" || string(record.Envelope) != raw {
			t.Fatal("page receipt", record)
		}
	}
	for _, key := range []string{"id", "name", "description", "is_default", "properties", "location"} {
		th.AssertEquals(t, "null", string(rows[1].Resource.Body[key]))
	}
	th.AssertEquals(t, `[1]`, string(rows[0].Wire.Body["type"]))
	th.AssertEquals(t, "1e400", string(rows[0].Wire.Body["weight"]))
	th.AssertEquals(t, `"https://foreign.test/"`, string(rows[0].Wire.Body["self"]))
	for _, key := range []string{"type", "read-only", "weight", "self", "vendor"} {
		if _, exists := rows[0].Resource.Body[key]; exists {
			t.Fatal("unknown field became declared descriptor", key)
		}
	}
	rows[0].Resource.Body["properties"][0] = '!'
	rows[0].Resource.Header.Set("X-Actual", "view changed")
	rows[0].Wire.Body["id"][0] = '0'
	rows[0].Wire.Header.Set("X-Actual", "wire changed")
	rows[0].Header.Set("X-Actual", "record changed")
	rows[0].Envelope[0] = '!'
	if string(rows[0].Resource.Body["id"]) != "900719925474099312345" || string(rows[0].Wire.Body["properties"]) != want["properties"] || rows[1].Header.Get("X-Actual") != "owned" || string(rows[1].Envelope) != raw || actualHeader.Get("X-Actual") != "owned" {
		t.Fatal("record channel alias")
	}
}

func TestStoreRecordsDescriptorsAliasesAndLocations(t *testing.T) {
	for _, test := range []struct{ name, row, id, flag, props, location string }{
		{"name is not alternate id", `{"name":"store name"}`, "null", "null", "null", `"captured"`},
		{"present null id", `{"id":null,"name":"store name"}`, "null", "null", "null", `"captured"`},
		{"untyped identity and description", `{"id":[1],"description":false}`, `[1]`, "null", "null", `"captured"`},
		{"false lexical string is truthy", `{"default":"false"}`, "null", "true", "null", `"captured"`},
		{"empty text false", `{"default":""}`, "null", "false", "null", `"captured"`},
		{"null remains null", `{"default":null,"properties":null}`, "null", "null", "null", `"captured"`},
		{"numeric zero false", `{"default":0}`, "null", "false", "null", `"captured"`},
		{"empty containers false", `{"default":{},"properties":[]}`, "null", "false", `{}`, `"captured"`},
		{"nonempty containers true", `{"default":[null],"properties":false}`, "null", "true", `{}`, `"captured"`},
		{"properties arbitrary object", `{"properties":{"value":[null,true,1e400]}}`, "null", "null", `{"value":[null,true,1e400]}`, `"captured"`},
		{"properties nonobject string", `{"properties":"opaque"}`, "null", "null", `{}`, `"captured"`},
		{"wire alias later", `{"is_default":false,"default":"false"}`, "null", "true", "null", `"captured"`},
		{"canonical alias later", `{"default":"false","is_default":false}`, "null", "false", "null", `"captured"`},
		{"duplicate keeps first insertion position", `{"default":false,"is_default":true,"default":null}`, "null", "true", "null", `"captured"`},
		{"only wire location", `{"location":{"cloud":"wire"}}`, "null", "null", "null", `"wire"`},
		{"unknown plus wire location", `{"type":false,"location":{"cloud":"wire"}}`, "null", "null", "null", `"wire"`},
		{"recognized null recomputes location", `{"description":null,"location":{"cloud":"wire"}}`, "null", "null", "null", `"captured"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := "captured"
			locations := 0
			client := infoClient(func(req *http.Request) (*http.Response, error) {
				return infoRecordJSON(req, 200, `{"stores":[`+test.row+`]}`), nil
			})
			api := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return resource.CloudLocation{Cloud: &cloud}, nil }})
			rows, err := api.AllStoreRecords(context.Background(), func(*StoreRecordListOpts) error { cloud = "after option"; return nil })
			if err != nil || len(rows) != 1 || locations != 1 || len(rows[0].Resource.Body) != 6 {
				t.Fatal(rows, err, locations)
			}
			th.AssertEquals(t, test.id, string(rows[0].Resource.Body["id"]))
			th.AssertEquals(t, test.flag, string(rows[0].Resource.Body["is_default"]))
			th.AssertEquals(t, test.props, string(rows[0].Resource.Body["properties"]))
			var captured resource.CloudLocation
			if err := json.Unmarshal(rows[0].Resource.Body["location"], &captured); err != nil || captured.Cloud == nil {
				t.Fatal(captured, err)
			}
			encoded, _ := json.Marshal(*captured.Cloud)
			th.AssertEquals(t, test.location, string(encoded))
		})
	}
	for _, location := range []string{"null", "false", "[1]", "42", `"literal"`} {
		t.Run("location only "+location, func(t *testing.T) {
			client := infoClient(func(req *http.Request) (*http.Response, error) {
				return infoRecordJSON(req, 200, `{"stores":[{"location":`+location+`}]}`), nil
			})
			rows, err := New(client).AllStoreRecords(context.Background())
			if err != nil || len(rows) != 1 {
				t.Fatal(rows, err)
			}
			th.AssertEquals(t, location, string(rows[0].Resource.Body["location"]))
		})
	}
}

func TestStoreRecordsLazyRepeatedCapturedOptions(t *testing.T) {
	calls, callbacks, locations := 0, 0, 0
	cloud := "first"
	limit := 2
	paginated := false
	headers := map[string]string{"X-Option": "factory snapshot"}
	filters := map[string]json.RawMessage{"id": json.RawMessage(`"selected"`)}
	factory := WithStoreRecordListOpts(StoreRecordListOpts{Headers: headers, Filters: filters, Limit: &limit, Marker: "start", Paginated: &paginated})
	var retained *StoreRecordListOpts
	client := infoClient(func(req *http.Request) (*http.Response, error) {
		calls++
		wantSource := "first"
		if calls == 2 {
			wantSource = "second"
		}
		if req.Header.Get("X-Source") != wantSource || req.Header.Get("X-Option") != "factory snapshot" || req.URL.Query().Get("limit") != "2" || req.URL.Query().Get("marker") != "start" || len(req.URL.Query()) != 2 {
			t.Fatal("owned snapshot", req.URL, req.Header)
		}
		retained.Headers["X-Option"] = "retained mutation"
		retained.Filters["id"][1] = 'X'
		*retained.Limit = 1
		*retained.Paginated = true
		return infoRecordJSON(req, 200, `{"stores":[{"id":"selected"}],"next":"https://foreign.test/never"}`), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "first"}
	api := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
		locations++
		client.MoreHeaders["X-Source"] = "changed by location"
		return resource.CloudLocation{Cloud: &cloud}, nil
	}})
	stream := api.ListStoreRecords(context.Background(), factory, func(value *StoreRecordListOpts) error {
		callbacks++
		retained = value
		client.MoreHeaders["X-Source"] = "changed by option"
		cloud = "changed by option"
		return nil
	})
	headers["X-Option"] = "caller changed"
	filters["id"][1] = 'X'
	limit = 1
	paginated = true
	if calls != 0 || callbacks != 0 || locations != 0 {
		t.Fatal("eager discovery", calls, callbacks, locations)
	}
	for iteration := 0; iteration < 2; iteration++ {
		wantCloud := "first"
		if iteration == 1 {
			client.MoreHeaders["X-Source"] = "second"
			cloud = "second"
			wantCloud = "second"
		}
		count := 0
		for row, err := range stream {
			if err != nil {
				t.Fatal(err)
			}
			count++
			var got resource.CloudLocation
			if err := json.Unmarshal(row.Resource.Body["location"], &got); err != nil || got.Cloud == nil || *got.Cloud != wantCloud {
				t.Fatal("per-iteration location", got, err)
			}
		}
		if count != 1 || calls != iteration+1 || callbacks != iteration+1 || locations != iteration+1 {
			t.Fatal(count, calls, callbacks, locations)
		}
	}
}

func TestStoreRecordsSourcePagingAndRawCaps(t *testing.T) {
	for _, mode := range []string{"next", "plural links", "links", "HTTP Link", "positive marker fallback", "explicit zero", "first server limit only", "single page", "raw cap filter", "zero plus cap hint", "early break"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			const next = "/v2/info/stores?marker=second"
			first := `{"stores":[{"id":"first"},{"id":"last"}]`
			suffix := `,"next":"` + next + `"`
			options := []StoreRecordListOption{}
			switch mode {
			case "plural links":
				suffix = `,"stores_links":[{"rel":"next","href":"` + next + `"}]`
			case "links":
				suffix = `,"links":[{"rel":"next","href":"` + next + `"}]`
			case "HTTP Link":
				suffix = ""
			case "positive marker fallback":
				suffix = ""
				options = append(options, WithStoreRecordListLimit(2))
			case "explicit zero":
				suffix = ""
				options = append(options, WithStoreRecordListLimit(0))
			case "first server limit only":
				suffix = `,"limit":2`
			case "single page":
				options = append(options, WithStoreRecordListPaginated(false))
			case "raw cap filter":
				options = append(options, WithStoreRecordListMaxItems(2), WithStoreRecordListFilter("id", "last"))
			case "zero plus cap hint":
				options = append(options, WithStoreRecordListLimit(0), WithStoreRecordListMaxItems(2), WithStoreRecordListFilter("id", "last"))
			}
			first += suffix + `}`
			client := infoClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.Path != "/reverse/glance/v2/info/stores" {
					t.Fatal("basic collection scope", req.URL)
				}
				if calls == 1 {
					switch mode {
					case "positive marker fallback", "raw cap filter", "zero plus cap hint":
						th.AssertEquals(t, "2", req.URL.Query().Get("limit"))
					case "explicit zero":
						th.AssertEquals(t, "0", req.URL.Query().Get("limit"))
					default:
						th.AssertEquals(t, "", req.URL.RawQuery)
					}
					reply := infoRecordJSON(req, 200, first)
					if mode == "HTTP Link" {
						reply.Header.Set("Link", "<"+next+">; rel=\"next\"")
					}
					return reply, nil
				}
				if calls != 2 {
					t.Fatal("unexpected third page", req.URL)
				}
				if mode == "positive marker fallback" {
					if req.URL.Query().Get("marker") != "last" || req.URL.Query().Get("limit") != "2" {
						t.Fatal("original wire last marker", req.URL)
					}
				} else if req.URL.Query().Get("marker") != "second" {
					t.Fatal("versioned continuation", req.URL)
				}
				return infoRecordJSON(req, 200, `{"stores":[]}`), nil
			})
			count := 0
			for row, err := range New(client).ListStoreRecords(context.Background(), options...) {
				if err != nil {
					t.Fatal(err)
				}
				count++
				if mode == "positive marker fallback" {
					row.Resource.Body["id"] = json.RawMessage(`"caller mutation"`)
					row.Wire.Body["id"] = json.RawMessage(`"wire caller mutation"`)
				}
				if mode == "early break" {
					break
				}
			}
			wantCalls, wantCount := 2, 2
			switch mode {
			case "explicit zero", "first server limit only", "single page":
				wantCalls = 1
			case "raw cap filter", "zero plus cap hint", "early break":
				wantCalls = 1
				wantCount = 1
			}
			if calls != wantCalls || count != wantCount {
				t.Fatal("paging/caps", calls, count, wantCalls, wantCount)
			}
		})
	}
	t.Run("server ignores positive limit and repeated cursor is stopped", func(t *testing.T) {
		calls := 0
		client := infoClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.URL.Query().Get("limit") != "1" || calls > 2 || calls == 2 && req.URL.Query().Get("marker") != "same" {
				t.Fatal("unexpected synthetic page", calls, req.URL)
			}
			return infoRecordJSON(req, 200, `{"stores":[{"id":"same"}]}`), nil
		})
		rows, err := New(client).AllStoreRecords(context.Background(), WithStoreRecordListLimit(1))
		if len(rows) != 2 || !errors.Is(err, resource.ErrPaginationCycle) || calls != 2 {
			t.Fatal("repeat cycle/partial evidence", rows, err, calls)
		}
	})
	t.Run("partial collector retains earlier page", func(t *testing.T) {
		calls := 0
		client := infoClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return infoRecordJSON(req, 200, `{"stores":[{"id":"kept"}],"next":"?marker=second"}`), nil
			}
			return infoRecordJSON(req, 200, `{"stores":[null]}`), nil
		})
		rows, err := New(client).AllStoreRecords(context.Background())
		if len(rows) != 1 || err == nil || calls != 2 || string(rows[0].Resource.Body["id"]) != `"kept"` {
			t.Fatal(rows, err, calls)
		}
		infoRecordProof(t, err, 200, `{"stores":[null]}`)
	})
	t.Run("unused bad row and foreign link ignored on break", func(t *testing.T) {
		calls := 0
		client := infoClient(func(req *http.Request) (*http.Response, error) {
			calls++
			return infoRecordJSON(req, 200, `{"stores":[{"id":"kept"},null],"next":"https://foreign.test/unsafe"}`), nil
		})
		count := 0
		for _, err := range New(client).ListStoreRecords(context.Background()) {
			if err != nil {
				t.Fatal(err)
			}
			count++
			break
		}
		if calls != 1 || count != 1 {
			t.Fatal(calls, count)
		}
	})
	t.Run("falsey next uses existing source policy", func(t *testing.T) {
		for _, next := range []string{"null", "false", "0", `""`, `[]`, `{}`} {
			t.Run(next, func(t *testing.T) {
				calls := 0
				client := infoClient(func(req *http.Request) (*http.Response, error) {
					calls++
					return infoRecordJSON(req, 200, `{"stores":[{}],"next":`+next+`}`), nil
				})
				rows, err := New(client).AllStoreRecords(context.Background())
				if len(rows) != 1 || err != nil || calls != 1 {
					t.Fatal(rows, err, calls)
				}
			})
		}
	})
}

func TestStoreRecordsCanonicalFiltersAndUnknownQueryPolicy(t *testing.T) {
	for _, test := range []struct {
		name, rows, field string
		value             any
		want              int
	}{
		{"id", `[{"id":"selected"},{"id":"other"}]`, "id", "selected", 1},
		{"name", `[{"name":"selected"},{"id":"selected"}]`, "name", "selected", 1},
		{"description untyped", `[{"description":[1,null]},{"description":"other"}]`, "description", []any{1, nil}, 1},
		{"canonical truthy default", `[{"default":"false"},{"default":false},{"default":null}]`, "is_default", true, 1},
		{"default remote query ignored", `[{"default":true},{"default":false}]`, "default", make(chan int), 2},
		{"unknown query ignored", `[{},{}]`, "vendor", json.RawMessage(`{`), 2},
		{"detail field ignored", `[{"type":"one"},{"type":"two"}]`, "type", "one", 2},
		{"properties subset", `[{"properties":{"outer":{"kept":1,"extra":2}}},{"properties":{"outer":{}}}]`, "properties", map[string]any{"outer": map[string]any{"kept": 1}}, 1},
		{"empty properties filter needs nonempty actual", `[{"properties":{}},{"properties":{"present":null}},{"properties":null}]`, "properties", map[string]any{}, 1},
		{"nullable default", `[{}, {"default":null}, {"default":false}]`, "is_default", nil, 2},
		{"exact huge id", `[{"id":900719925474099312345},{"id":900719925474099312346}]`, "id", json.RawMessage(`900719925474099312345`), 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := infoClient(func(req *http.Request) (*http.Response, error) {
				if req.URL.RawQuery != "" {
					t.Fatal("body/unknown filter became server query", req.URL)
				}
				return infoRecordJSON(req, 200, `{"stores":`+test.rows+`}`), nil
			})
			rows, err := New(client).AllStoreRecords(context.Background(), WithStoreRecordListFilter(test.field, test.value))
			if err != nil || len(rows) != test.want {
				t.Fatal(rows, err)
			}
		})
	}
	t.Run("bulk filter factory owns captured nested values", func(t *testing.T) {
		values := map[string]any{"properties": map[string]any{"number": json.RawMessage(`900719925474099312345`)}, "default": make(chan int)}
		option := WithStoreRecordListFilters(values)
		values["properties"].(map[string]any)["number"] = 0
		client := infoClient(func(req *http.Request) (*http.Response, error) {
			return infoRecordJSON(req, 200, `{"stores":[{"properties":{"number":900719925474099312345}},{"properties":{"number":0}}]}`), nil
		})
		rows, err := New(client).AllStoreRecords(context.Background(), option)
		if err != nil || len(rows) != 1 {
			t.Fatal(rows, err)
		}
	})
}

func TestStoreRecordsEnvelopeStatusesAndConstructorFailures(t *testing.T) {
	for _, test := range []struct {
		code  int
		raw   string
		good  bool
		count int
	}{
		{200, `{"stores":[]}`, true, 0}, {201, `{"stores":{}}`, true, 1}, {299, `{"stores":[{}]}`, true, 1}, {300, `{"stores":[{}]}`, true, 1}, {399, `{"stores":[]}`, true, 0},
		{204, "", false, 0}, {200, "not JSON", false, 0}, {200, `{}`, false, 0}, {200, `null`, false, 0}, {200, `[]`, false, 0}, {200, `{"stores":null}`, false, 0}, {200, `{"stores":false}`, false, 0}, {200, `{"stores":[null]}`, false, 0}, {200, `{"stores":[1]}`, false, 0}, {200, `{"stores":[{"connection":null}]}`, false, 0}, {200, `{"stores":[{"microversion":"2"}]}`, false, 0}, {200, `{"stores":[{"_synchronized":false}]}`, false, 0}, {200, "{\"stores\":[{\"unknown\":\"\xff\"}]}", false, 0},
	} {
		t.Run(fmt.Sprintf("%d/%q", test.code, test.raw), func(t *testing.T) {
			calls, retries := 0, 0
			body := &infoBody{reader: strings.NewReader(test.raw)}
			client := infoClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return infoRecordHTTP(req, test.code, body), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			rows, err := New(client).AllStoreRecords(context.Background())
			if test.good {
				if err != nil || len(rows) != test.count {
					t.Fatal(rows, err)
				}
			} else {
				if err == nil || len(rows) != 0 {
					t.Fatal(rows, err)
				}
				infoRecordProof(t, err, test.code, test.raw)
			}
			if calls != 1 || retries != 0 || body.closes != 1 {
				t.Fatal("accepted body replay", calls, retries, body.closes)
			}
		})
	}
}

func TestStoreRecordsBodyFailuresAndStickySourceGuards(t *testing.T) {
	const raw = `{"stores":[{"id":"actual"}]}`
	for _, mode := range []string{"read", "close", "cancel", "source drift", "source restored at Close", "outer restored at Close"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New("store record body failure")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			calls, retries := 0, 0
			var client *gophercloud.ServiceClient
			body := &infoBody{reader: strings.NewReader(raw)}
			outerInvalid := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerInvalid {
					return cause
				}
				return nil
			})
			action := func() {}
			switch mode {
			case "read":
				body.reader = infoReader(func(buf []byte) (int, error) { return copy(buf, raw), cause })
			case "close":
				body.closeErr = errors.Join(cause, gophercloud.ErrUnexpectedResponseCode{Actual: 404})
			case "cancel":
				action = func() { cancel(cause) }
			case "source drift", "source restored at Close":
				action = func() { client.Endpoint = "https://example.test/changed/" }
			case "outer restored at Close":
				action = func() { outerInvalid = true }
			}
			if mode != "read" {
				body.reader = infoReader(func(buf []byte) (int, error) { action(); return copy(buf, raw), io.EOF })
			}
			selected := io.ReadCloser(body)
			if strings.Contains(mode, "restored") {
				selected = &infoRecordCloseBody{infoBody: body, after: func() { client.Endpoint = "https://example.test/"; outerInvalid = false }}
			}
			client = infoClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return infoRecordHTTP(req, 201, selected), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			rows, err := New(client).AllStoreRecords(ctx)
			if len(rows) != 0 || err == nil || calls != 1 || retries != 0 || body.closes != 1 {
				t.Fatal(rows, err, calls, retries, body.closes)
			}
			if strings.HasPrefix(mode, "source") {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, cause) {
				t.Fatal("cause lost", err)
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if errors.Is(err, resource.ErrNotFound) {
				t.Fatal("nested404 became absence", err)
			}
			infoRecordProof(t, err, 201, raw)
		})
	}
}

func TestStoreRecordsPreflightAndLegacyDetailCompatibility(t *testing.T) {
	for _, test := range []struct {
		name   string
		option StoreRecordListOption
	}{
		{"nil option", nil}, {"negative limit", WithStoreRecordListLimit(-1)}, {"negative cap", WithStoreRecordListMaxItems(-1)}, {"bad marker", WithStoreRecordListMarker("\n")}, {"owned auth", WithStoreRecordListHeader("X-Auth-Token", "foreign")}, {"bad header", WithStoreRecordListHeader("X-Extra", "\n")}, {"broken canonical JSON", WithStoreRecordListOpts(StoreRecordListOpts{Filters: map[string]json.RawMessage{"properties": json.RawMessage(`{`)}})},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := infoClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil })
			rows, err := New(client).AllStoreRecords(context.Background(), test.option)
			if len(rows) != 0 || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(rows, err, calls)
			}
		})
	}
	t.Run("unknown map filter discarded before JSON validation", func(t *testing.T) {
		client := infoClient(func(req *http.Request) (*http.Response, error) {
			return infoRecordJSON(req, 200, `{"stores":[{}]}`), nil
		})
		rows, err := New(client).AllStoreRecords(context.Background(), WithStoreRecordListOpts(StoreRecordListOpts{Filters: map[string]json.RawMessage{"default": json.RawMessage(`{`), "vendor": json.RawMessage(`{`)}}))
		if len(rows) != 1 || err != nil {
			t.Fatal(rows, err)
		}
	})
	t.Run("option source drift stops later restoration", func(t *testing.T) {
		calls, callbacks := 0, 0
		client := infoClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil })
		rows, err := New(client).AllStoreRecords(context.Background(), func(*StoreRecordListOpts) error { client.ProviderClient = &gophercloud.ProviderClient{}; return nil }, func(*StoreRecordListOpts) error { callbacks++; return nil })
		if len(rows) != 0 || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 {
			t.Fatal(rows, err, calls, callbacks)
		}
	})
	t.Run("empty collector allocated", func(t *testing.T) {
		client := infoClient(func(req *http.Request) (*http.Response, error) { return infoRecordJSON(req, 200, `{"stores":[]}`), nil })
		rows, err := New(client).AllStoreRecords(context.Background())
		if rows == nil || err != nil || len(rows) != 0 {
			t.Fatal(rows, err)
		}
	})
	t.Run("owned controls expose no admin detail selector", func(t *testing.T) {
		if _, found := reflect.TypeFor[StoreRecordListOpts]().FieldByName("Details"); found {
			t.Fatal("detail selector entered core user owned controls")
		}
	})
	t.Run("legacy detail route and lexical flag remain", func(t *testing.T) {
		calls := 0
		client := infoClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.URL.Path != "/reverse/glance/v2/info/stores/detail" {
				t.Fatal(req.URL)
			}
			return infoRecordJSON(req, 200, `{"stores":[{"id":"legacy","default":"false","type":"file","read-only":true,"weight":0}]}`), nil
		})
		rows, err := New(client).AllStores(context.Background(), WithListStoresDetails(true))
		if err != nil || len(rows) != 1 || calls != 1 || rows[0].IsDefault == nil || *rows[0].IsDefault || rows[0].Type == nil || *rows[0].Type != "file" || rows[0].ReadOnly == nil || !*rows[0].ReadOnly || rows[0].Weight == nil || *rows[0].Weight != 0 {
			t.Fatal(rows, err, calls)
		}
	})
	t.Run("legacy remains strict200", func(t *testing.T) {
		calls := 0
		client := infoClient(func(req *http.Request) (*http.Response, error) {
			calls++
			return infoRecordJSON(req, 201, `{"stores":[]}`), nil
		})
		rows, err := New(client).AllStores(context.Background())
		var native gophercloud.ErrUnexpectedResponseCode
		if rows != nil || !errors.As(err, &native) || native.Actual != 201 || calls != 1 {
			t.Fatal(rows, err, native, calls)
		}
	})
}
