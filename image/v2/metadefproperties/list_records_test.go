package metadefproperties

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
)

func TestMetadefPropertyListRecordsOrderedDictionaryProjectionAndReceipts(t *testing.T) {
	calls := 0
	body := `{"properties":{"z first":{"name":"old"},"a second":{},"z first":{"name":"response name","id":null,"minLength":"03","readonly":"false","namespace_name":"wire parent","location":{"cloud":"wire"},"future":900719925474099312345,"Key":"wire key"}},"next":false,"links":"opaque"}`
	client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		want := "/reverse/glance/v2/metadefs/namespaces/" + url.PathEscape(propertyRecordParent) + "/properties"
		if req.Method != http.MethodGet || req.URL.EscapedPath() != want || req.URL.RawQuery != "" || req.Body != nil || req.Header.Get("Accept") != "application/json" || req.Header.Get("X-Auth-Token") != "before" {
			t.Fatal("fixed finite collection request", req.Method, req.URL, req.Header)
		}
		response := propertyCoreJSON(req, 203, body)
		response.Header.Set("Link", "broken and unused")
		return response, nil
	})
	rows, err := propertyRecordScope(t, client, Dependencies{}).AllRecords(context.Background())
	if err != nil || len(rows) != 2 || calls != 1 || rows[0].Key == nil || *rows[0].Key != "z first" || rows[1].Key == nil || *rows[1].Key != "a second" {
		t.Fatal("dictionary first slots/last values", rows, err, calls)
	}
	want := propertyRecordDefaults("a second")
	want["name"] = `"a second"`
	if !reflect.DeepEqual(propertyRecordValues(rows[1]), want) {
		t.Fatal("dictionary key name/id defaults", rows[1].Resource.Body, want)
	}
	first := rows[0]
	if first.Namespace != propertyRecordParent || string(first.Resource.Body["name"]) != `"response name"` || string(first.Resource.Body["id"]) != "null" || string(first.Resource.Body["min_length"]) != "3" || string(first.Resource.Body["is_readonly"]) != "true" || string(first.Resource.Body["location"]) != "null" || len(first.Resource.Body) != 21 {
		t.Fatal("declared record projection", first)
	}
	if string(first.Wire.Body["minLength"]) != `"03"` || string(first.Wire.Body["readonly"]) != `"false"` || string(first.Wire.Body["future"]) != "900719925474099312345" || string(first.Wire.Body["namespace_name"]) != `"wire parent"` || string(first.Wire.Body["Key"]) != `"wire key"` || len(rows[1].Wire.Body) != 0 {
		t.Fatal("row Wire was seeded or narrowed", first.Wire, rows[1].Wire)
	}
	for _, row := range rows {
		if row.StatusCode != 203 || row.Resource.StatusCode != 203 || row.Wire.StatusCode != 203 || row.Header.Get("X-Proof") != "original" || row.Resource.Header.Get("X-Proof") != "original" || row.Wire.Header.Get("X-Proof") != "original" || string(row.Envelope) != body {
			t.Fatal("physical-page receipt missing", row)
		}
		if row.Resource.CreatedAt != nil || row.Resource.UpdatedAt != nil || row.Resource.Links != nil {
			t.Fatal("projected metadata acquired undeclared DTO fields", row.Resource.Metadata)
		}
	}
	first.Resource.Body["name"][1] = 'X'
	first.Resource.Header.Set("X-Proof", "view changed")
	first.Header.Set("X-Proof", "record changed")
	first.Envelope[0] = 'X'
	*first.Key = "caller key"
	if string(first.Wire.Body["name"]) != `"response name"` || first.Wire.Header.Get("X-Proof") != "original" || rows[1].Header.Get("X-Proof") != "original" || string(rows[1].Envelope) != body || *rows[1].Key != "a second" {
		t.Fatal("owned view/wire/page/key channels alias")
	}
}

func TestMetadefPropertyListRecordsNamesAndKeysRemainPassive(t *testing.T) {
	for _, check := range []struct{ key, raw, name, id string }{
		{"", `{}`, `""`, `""`},
		{"key/with%?#slash", `{}`, `"key/with%?#slash"`, `"key/with%?#slash"`},
		{strings.Repeat("界", 81), `{}`, "", ""},
		{"outer", `{"name":null}`, "null", "null"},
		{"outer", `{"name":false}`, "false", "false"},
		{"outer", `{"id":[],"name":"response"}`, `"response"`, "[]"},
		{"outer", `{"id":null,"name":"response"}`, `"response"`, "null"},
	} {
		t.Run(fmt.Sprintf("%q/%s", check.key, check.raw), func(t *testing.T) {
			key, _ := json.Marshal(check.key)
			body := `{"properties":{` + string(key) + `:` + check.raw + `}}`
			calls := 0
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.RawQuery != "" {
					t.Fatal(req.URL)
				}
				return propertyCoreJSON(req, 200, body), nil
			})
			rows, err := propertyRecordScope(t, client, Dependencies{}).AllRecords(context.Background())
			if err != nil || len(rows) != 1 || calls != 1 || rows[0].Key == nil || *rows[0].Key != check.key {
				t.Fatal(rows, err, calls)
			}
			wantName, wantID := check.name, check.id
			if wantName == "" {
				wantName = string(key)
				wantID = string(key)
			}
			if string(rows[0].Resource.Body["name"]) != wantName || string(rows[0].Resource.Body["id"]) != wantID {
				t.Fatal("passive dictionary name/id", rows[0].Resource.Body)
			}
			var raw map[string]json.RawMessage
			if err := json.Unmarshal([]byte(check.raw), &raw); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(rows[0].Wire.Body, raw) {
				t.Fatal("passive key was injected into Wire", rows[0].Wire.Body, raw)
			}
		})
	}
}

func TestMetadefPropertyListRecordsCanonicalFiltersUseProjectedValues(t *testing.T) {
	other := `{"id":"different","name":"different","type":"different","title":"different","description":"different","operators":["different"],"default":"different","readonly":false,"minimum":9,"maximum":9,"enum":["different"],"pattern":"different","minLength":9,"maxLength":9,"items":{},"uniqueItems":true,"minItems":9,"maxItems":9,"additionalItems":true}`
	for _, check := range []struct {
		field, row string
		filter     any
	}{
		{"id", `{"id":"selected"}`, "selected"},
		{"name", `{}`, "selected"},
		{"type", `{"type":{"kind":"custom","extra":1}}`, map[string]any{"kind": "custom"}},
		{"title", `{"title":"selected"}`, "selected"},
		{"description", `{"description":{"nested":{"kept":1,"extra":2}}}`, map[string]any{"nested": map[string]any{"kept": 1}}},
		{"operators", `{"operators":"and"}`, []string{"and"}},
		{"default", `{"default":{"number":900719925474099312345}}`, map[string]any{"number": json.RawMessage(`900719925474099312345`)}},
		{"is_readonly", `{"readonly":"false"}`, true},
		{"minimum", `{"minimum":2.9}`, 2},
		{"maximum", `{"maximum":900719925474099312345}`, json.RawMessage(`900719925474099312345`)},
		{"enum", `{"enum":"one"}`, []string{"one"}},
		{"pattern", `{"pattern":"selected"}`, "selected"},
		{"min_length", `{}`, 0},
		{"max_length", `{"maxLength":"٠٣"}`, 3},
		{"items", `{"items":{"kept":1,"extra":2}}`, map[string]any{"kept": 1}},
		{"require_unique_items", `{}`, false},
		{"min_items", `{}`, 0},
		{"max_items", `{"maxItems":"03"}`, 3},
		{"allow_additional_items", `{"additionalItems":[]}`, false},
	} {
		t.Run(check.field, func(t *testing.T) {
			calls := 0
			body := `{"properties":{"selected":` + check.row + `,"other":` + other + `}}`
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.RawQuery != "" || req.Body != nil {
					t.Fatal("local descriptor filter leaked into request", req.URL)
				}
				return propertyCoreJSON(req, 200, body), nil
			})
			rows, err := propertyRecordScope(t, client, Dependencies{}).AllRecords(context.Background(), WithRecordListFilter(check.field, check.filter))
			if err != nil || len(rows) != 1 || calls != 1 || rows[0].Key == nil || *rows[0].Key != "selected" {
				t.Fatal("filter did not read projected descriptor value", rows, err, calls)
			}
		})
	}
	for _, check := range []struct {
		name, rows string
		filters    []RecordListOption
		want       int
	}{
		{"no request-filter coercion", `{"one":{"minimum":"03"}}`, []RecordListOption{WithRecordListFilter("minimum", "03")}, 0},
		{"null known field", `{"one":{},"two":{"title":null},"three":{"title":"value"}}`, []RecordListOption{WithRecordListFilter("title", nil)}, 2},
		{"empty dict needs nonempty actual", `{"one":{"items":{}},"two":{"items":{"value":1}}}`, []RecordListOption{WithRecordListFilter("items", map[string]any{})}, 1},
		{"Go bool differs from number", `{"one":{"readonly":"false"}}`, []RecordListOption{WithRecordListFilter("is_readonly", 1)}, 0},
		{"max_items descriptor is not cap", `{"one":{"maxItems":1},"two":{"maxItems":1},"three":{"maxItems":1}}`, []RecordListOption{WithRecordListFilter("max_items", 1)}, 3},
		{"negative max_items descriptor allowed", `{"one":{"maxItems":-1}}`, []RecordListOption{WithRecordListFilter("max_items", -1)}, 1},
		{"bulk replaces prior field", `{"one":{"name":"match","minimum":2}}`, []RecordListOption{WithRecordListFilter("minimum", 9), WithRecordListFilters(map[string]any{"name": "match"})}, 1},
		{"later single overlays bulk", `{"one":{"name":"final"}}`, []RecordListOption{WithRecordListFilters(map[string]any{"name": "old"}), WithRecordListFilter("name", "final")}, 1},
		{"replacement clears encoding error", `{"one":{"name":"final"}}`, []RecordListOption{WithRecordListFilter("name", make(chan int)), WithRecordListFilter("name", "final")}, 1},
	} {
		t.Run(check.name, func(t *testing.T) {
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				if req.URL.RawQuery != "" {
					t.Fatal(req.URL)
				}
				return propertyCoreJSON(req, 200, `{"properties":`+check.rows+`}`), nil
			})
			rows, err := propertyRecordScope(t, client, Dependencies{}).AllRecords(context.Background(), check.filters...)
			if err != nil || len(rows) != check.want {
				t.Fatal(rows, err)
			}
		})
	}
	for _, ignored := range []string{"readonly", "minLength", "maxLength", "uniqueItems", "minItems", "maxItems", "additionalItems", "limit", "marker", "location", "created_at", "future"} {
		t.Run("ignored before encoding "+ignored, func(t *testing.T) {
			calls := 0
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.RawQuery != "" {
					t.Fatal("ignored parameter leaked into URL", req.URL)
				}
				return propertyCoreJSON(req, 200, `{"properties":{"one":{}}}`), nil
			})
			rows, err := propertyRecordScope(t, client, Dependencies{}).AllRecords(context.Background(), WithRecordListFilter(ignored, make(chan int)))
			if err != nil || len(rows) != 1 || calls != 1 {
				t.Fatal("ignored field exposed its encoding failure", rows, err, calls)
			}
		})
	}
}

func TestMetadefPropertyListRecordsCapsBreakAndPartialResults(t *testing.T) {
	for _, mode := range []string{"cap", "break", "filtered cap", "late bad value", "late descriptor error", "duplicate last value"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			body := `{"properties":{"one":{"title":"first"},"two":null},"next":false,"links":"bad"}`
			if mode == "late descriptor error" {
				body = `{"properties":{"one":{"title":"first"},"two":{"minimum":"²"}},"next":false}`
			}
			if mode == "duplicate last value" {
				body = `{"properties":{"one":{"title":"first"},"two":{},"two":null}}`
			}
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.RawQuery != "" {
					t.Fatal("raw cap sent a limit hint", req.URL)
				}
				response := propertyCoreJSON(req, 203, body)
				response.Header.Set("Link", "unusable")
				return response, nil
			})
			scope := propertyRecordScope(t, client, Dependencies{})
			options := []RecordListOption{}
			if mode == "cap" || mode == "filtered cap" {
				options = append(options, WithRecordListMaxItems(1))
			}
			if mode == "filtered cap" {
				options = append(options, WithRecordListFilter("title", "excluded"))
			}
			var rows []*Record
			var err error
			if mode == "break" {
				for row, readErr := range scope.ListRecords(context.Background()) {
					rows = append(rows, row)
					err = readErr
					break
				}
			} else {
				rows, err = scope.AllRecords(context.Background(), options...)
			}
			want := 1
			if mode == "filtered cap" {
				want = 0
			}
			if len(rows) != want || calls != 1 {
				t.Fatal(rows, err, calls)
			}
			if strings.HasPrefix(mode, "late") || mode == "duplicate last value" {
				if err == nil {
					t.Fatal("consumed invalid row was accepted")
				}
				proof := propertyCoreProof(t, err, 203, body)
				rows[0].Envelope[0] = 'X'
				rows[0].Header.Set("X-Proof", "caller changed")
				if string(proof.Body) != body || proof.Header.Get("X-Proof") != "original" {
					t.Fatal("partial row aliases failure receipt")
				}
			} else if err != nil {
				t.Fatal("unconsumed value or passive pagination was decoded", err)
			}
		})
	}
	for _, bad := range []string{`null`, `false`, `[]`, `{"minimum":"²"}`} {
		t.Run("cap skips "+bad, func(t *testing.T) {
			body := `{"properties":{"one":{},"unused":` + bad + `}}`
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) { return propertyCoreJSON(req, 200, body), nil })
			rows, err := propertyRecordScope(t, client, Dependencies{}).AllRecords(context.Background(), WithRecordListMaxItems(1))
			if err != nil || len(rows) != 1 {
				t.Fatal(rows, err)
			}
		})
	}
}

func TestMetadefPropertyListRecordsStatusAndMandatoryWholePageValidation(t *testing.T) {
	for _, code := range []int{200, 201, 203, 204, 299, 300, 304, 399} {
		t.Run(fmt.Sprintf("accepted %d", code), func(t *testing.T) {
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				return propertyCoreJSON(req, code, `{"properties":{"one":{}}}`), nil
			})
			rows, err := propertyRecordScope(t, client, Dependencies{}).AllRecords(context.Background())
			if err != nil || len(rows) != 1 || rows[0].StatusCode != code || rows[0].Wire.StatusCode != code || rows[0].Resource.StatusCode != code {
				t.Fatal(rows, err)
			}
		})
	}
	for _, body := range []string{"", `not JSON`, `null`, `[]`, `{}`, `{"properties":null}`, `{"properties":[]}`, `{"properties":false}`, `{"properties":{"one":{}}} trailing`, `{"properties":{"one":{},"unused":}}`, "{\"properties\":{\"one\":{}},\"unused\":\"\xff\"}"} {
		t.Run("invalid page "+fmt.Sprintf("%q", body), func(t *testing.T) {
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) { return propertyCoreJSON(req, 204, body), nil })
			rows, err := propertyRecordScope(t, client, Dependencies{}).AllRecords(context.Background(), WithRecordListMaxItems(1))
			if err == nil || len(rows) != 0 {
				t.Fatal("list reused member invalid-JSON tolerance", rows, err)
			}
			propertyCoreProof(t, err, 204, body)
		})
	}
	for _, code := range []int{400, 404, 500} {
		t.Run(fmt.Sprintf("rejected %d", code), func(t *testing.T) {
			calls := 0
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return propertyCoreJSON(req, code, "native rejection"), nil
			})
			rows, err := propertyRecordScope(t, client, Dependencies{}).AllRecords(context.Background())
			var native gophercloud.ErrUnexpectedResponseCode
			if len(rows) != 0 || !errors.As(err, &native) || native.Actual != code || string(native.Body) != "native rejection" || calls != 1 {
				t.Fatal(rows, err, native, calls)
			}
		})
	}
	t.Run("successful empty ignores next", func(t *testing.T) {
		calls := 0
		client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			response := propertyCoreJSON(req, 200, `{"properties":{},"next":false}`)
			response.Header.Set("Link", "broken")
			return response, nil
		})
		rows, err := propertyRecordScope(t, client, Dependencies{}).AllRecords(context.Background())
		if err != nil || rows == nil || len(rows) != 0 || calls != 1 {
			t.Fatal(rows, err, calls)
		}
	})
}

func TestMetadefPropertyListRecordsLazySnapshotsAndReiteration(t *testing.T) {
	calls, callbacks, locations := 0, 0, 0
	cloud := "first"
	location := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"owned project"`)}}
	client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		wantSource := "first"
		if calls == 2 {
			wantSource = "second"
		}
		if req.Header.Get("X-Source") != wantSource || req.Header.Get("X-Full") != "factory" || req.Header.Get("X-Bulk") != "factory" || req.Header.Get("X-Replaced") != "" || req.Header.Get("X-Final") != "last" || req.URL.RawQuery != "" {
			t.Fatal("lazy snapshot ownership", req.URL, req.Header)
		}
		return propertyCoreJSON(req, 200, `{"properties":{"one":{"name":"match","location":{"cloud":"wire"}},"unused":null}}`), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "first"}
	scope := propertyRecordScope(t, client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return location, nil }})
	fullHeaders := map[string]string{"X-Full": "factory"}
	bulkHeaders := map[string]string{"X-Bulk": "factory"}
	filters := map[string]any{"name": "match"}
	filterSlice := []resource.ListOption{resource.WithFilters(filters)}
	full := WithRecordListOpts(RecordListOpts{Headers: fullHeaders, Filters: filterSlice, MaxItems: 1})
	bulk := WithRecordListHeaders(bulkHeaders)
	var retained *RecordListOpts
	options := []RecordListOption{WithRecordListHeader("X-Replaced", "old"), full, bulk,
		func(config *RecordListOpts) error {
			callbacks++
			retained = config
			cloud = "callback changed"
			location.Project.ID[1] = 'X'
			client.MoreHeaders["X-Source"] = "callback changed"
			return nil
		},
		func(*RecordListOpts) error {
			retained.Headers["X-Full"] = "retained changed"
			retained.Filters[0] = resource.WithFilter("name", "excluded")
			retained.MaxItems = 0
			return nil
		},
		WithRecordListHeader("x-final", "last"),
	}
	seq := scope.ListRecords(context.Background(), options...)
	options[0] = nil
	fullHeaders["X-Full"] = "caller changed"
	bulkHeaders["X-Bulk"] = "caller changed"
	filters["name"] = "caller changed"
	filterSlice[0] = resource.WithFilter("name", "excluded")
	if calls != 0 || callbacks != 0 || locations != 0 {
		t.Fatal("finite records iterator was eager")
	}
	for iteration := 0; iteration < 2; iteration++ {
		wantCloud := "first"
		if iteration == 1 {
			cloud = "second"
			location.Project.ID = json.RawMessage(`"owned project"`)
			client.MoreHeaders["X-Source"] = "second"
			wantCloud = "second"
		}
		count := 0
		for row, err := range seq {
			if err != nil {
				t.Fatal(err)
			}
			count++
			var actual resource.CloudLocation
			if err := json.Unmarshal(row.Resource.Body["location"], &actual); err != nil || actual.Cloud == nil || *actual.Cloud != wantCloud || string(actual.Project.ID) != `"owned project"` {
				t.Fatal("location not captured before options", actual, err)
			}
			if string(row.Wire.Body["location"]) != `{"cloud":"wire"}` {
				t.Fatal("location projection changed Wire")
			}
		}
		if count != 1 || calls != iteration+1 || callbacks != iteration+1 || locations != iteration+1 {
			t.Fatal(count, calls, callbacks, locations)
		}
	}
}

func TestMetadefPropertyListRecordsCompletePreflightAndSourceLifetime(t *testing.T) {
	for _, check := range []struct {
		name    string
		options []RecordListOption
	}{
		{"nil option", []RecordListOption{nil}},
		{"negative raw cap", []RecordListOption{WithRecordListMaxItems(-1)}},
		{"nil semantic option", []RecordListOption{WithRecordListOpts(RecordListOpts{Filters: []resource.ListOption{nil}})}},
		{"nonsemantic option", []RecordListOption{WithRecordListOpts(RecordListOpts{Filters: []resource.ListOption{resource.WithName("other")}})}},
		{"known encoding failure", []RecordListOption{WithRecordListFilter("minimum", make(chan int))}},
		{"protected header", []RecordListOption{WithRecordListHeader("Authorization", "foreign")}},
		{"accept header", []RecordListOption{WithRecordListHeader("Accept", "text/plain")}},
		{"header aliases", []RecordListOption{WithRecordListHeaders(map[string]string{"X-Alias": "one", "x-alias": "two"})}},
		{"header newline", []RecordListOption{WithRecordListHeader("X-Option", "bad\nvalue")}},
	} {
		t.Run(check.name, func(t *testing.T) {
			calls := 0
			client := propertyCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
			rows, err := propertyRecordScope(t, client, Dependencies{}).AllRecords(context.Background(), check.options...)
			if len(rows) != 0 || err == nil || calls != 0 {
				t.Fatal(rows, err, calls)
			}
		})
	}
	for _, key := range []string{"namespace_name", "base_path", "microversion", "session", "headers"} {
		t.Run("reserved control "+key, func(t *testing.T) {
			calls := 0
			client := propertyCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			rows, err := propertyRecordScope(t, client, Dependencies{}).AllRecords(context.Background(), WithRecordListFilter(key, "unsafe"))
			if len(rows) != 0 || err == nil || calls != 0 {
				t.Fatal("control replaced fixed namespace/session/headers", rows, err, calls)
			}
		})
	}
	for _, mode := range []string{"nil context", "canceled context", "source lifetime", "outer failure", "location failure", "callback failure", "callback retarget"} {
		t.Run(mode, func(t *testing.T) {
			calls, callbacks, locations := 0, 0, 0
			cause := errors.New("preflight cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var use context.Context = ctx
			client := propertyCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			scope := propertyRecordScope(t, client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
				locations++
				if mode == "location failure" {
					return resource.CloudLocation{}, cause
				}
				return resource.CloudLocation{}, nil
			}})
			switch mode {
			case "nil context":
				use = nil
			case "canceled context":
				cancel(cause)
			case "source lifetime":
				client.Endpoint = "https://changed.test/"
			case "outer failure":
				use = rest.WithOperationGuard(use, func(context.Context) error { return cause })
			}
			rows, err := scope.AllRecords(use, func(*RecordListOpts) error {
				callbacks++
				if mode == "callback failure" {
					return cause
				}
				if mode == "callback retarget" {
					client.Type = "compute"
				}
				return nil
			})
			if len(rows) != 0 || err == nil || calls != 0 {
				t.Fatal(rows, err, calls)
			}
			if mode == "callback failure" || mode == "callback retarget" {
				if callbacks != 1 || locations != 1 {
					t.Fatal(callbacks, locations)
				}
			} else if callbacks != 0 {
				t.Fatal("invalid preflight ran options", callbacks)
			}
			if mode == "nil context" || mode == "canceled context" || mode == "source lifetime" || mode == "outer failure" {
				if locations != 0 {
					t.Fatal("source guard invoked location", locations)
				}
			}
			if mode == "callback failure" || mode == "location failure" || mode == "outer failure" || mode == "canceled context" {
				if !errors.Is(err, cause) {
					t.Fatal("preflight cause lost", err)
				}
			}
		})
	}
}

func TestMetadefPropertyListRecordsGuardBeforeNextDecodeAndEmptyCompletion(t *testing.T) {
	for _, mode := range []string{"source after yield", "outer after yield", "canceled after yield", "cap completion guard", "empty callback guard", "observed restored read"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			outerFailed := false
			cause := errors.New("guard cause")
			base, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			ctx := rest.WithOperationGuard(base, func(context.Context) error {
				if outerFailed {
					return cause
				}
				return nil
			})
			bodyText := `{"properties":{"one":{},"unused":null}}`
			if mode == "empty callback guard" {
				bodyText = `{"properties":{}}`
			}
			var client *gophercloud.ServiceClient
			client = propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				body := &propertyCoreBody{reader: strings.NewReader(bodyText)}
				if mode == "empty callback guard" {
					return propertyCoreHTTP(req, 200, &propertyRecordCloseBody{propertyCoreBody: body, after: func() { outerFailed = true }}), nil
				}
				if mode == "observed restored read" {
					body.reader = &propertyCoreReader{data: bodyText, err: io.EOF, after: func() {
						client.Microversion = "2.3"
						if err := rest.CheckOperationGuard(req.Context()); !errors.Is(err, resource.ErrInvalidOption) {
							t.Error("source failure not observed", err)
						}
					}}
					return propertyCoreHTTP(req, 200, &propertyRecordCloseBody{propertyCoreBody: body, after: func() { client.Microversion = "" }}), nil
				}
				return propertyCoreHTTP(req, 200, body), nil
			})
			scope := propertyRecordScope(t, client, Dependencies{})
			options := []RecordListOption{}
			if mode == "cap completion guard" {
				options = append(options, WithRecordListMaxItems(1))
			}
			rows, failures := 0, 0
			for _, err := range scope.ListRecords(ctx, options...) {
				if err != nil {
					failures++
					propertyCoreProof(t, err, 200, bodyText)
					if mode == "outer after yield" || mode == "empty callback guard" {
						if !errors.Is(err, cause) {
							t.Fatal("outer cause lost", err)
						}
					} else if mode == "canceled after yield" {
						if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
							t.Fatal("context cause lost", err)
						}
					} else if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal("source cause lost", err)
					}
					continue
				}
				rows++
				switch mode {
				case "source after yield", "cap completion guard":
					client.ResourceBase += "changed/"
				case "outer after yield":
					outerFailed = true
				case "canceled after yield":
					cancel(cause)
				}
			}
			want := 1
			if mode == "empty callback guard" || mode == "observed restored read" {
				want = 0
			}
			if calls != 1 || rows != want || failures != 1 {
				t.Fatal("guard ran after later row decode or missed completion", calls, rows, failures)
			}
		})
	}
}

func TestMetadefPropertyListRecordsAcceptedFaultsAndNativeOwnership(t *testing.T) {
	t.Run("accepted read close cancel", func(t *testing.T) {
		calls, retries := 0, 0
		readErr, closeErr, cause := errors.New("read"), errors.New("close"), errors.New("cancel")
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		bodyText := `{"properties":{"one":{}}}`
		body := &propertyCoreBody{reader: &propertyCoreReader{data: bodyText, err: readErr, after: func() { cancel(cause) }}, closeErr: closeErr}
		client := propertyCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return propertyCoreHTTP(req, 203, body), nil })
		client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			retries++
			return nil
		}
		rows, err := propertyRecordScope(t, client, Dependencies{}).AllRecords(ctx)
		if len(rows) != 0 || err == nil || calls != 1 || retries != 0 || body.closes != 1 {
			t.Fatal(rows, err, calls, retries, body.closes)
		}
		propertyCoreProof(t, err, 203, bodyText)
		for _, expected := range []error{readErr, closeErr, cause, context.Canceled} {
			if !errors.Is(err, expected) {
				t.Fatal("accepted cause lost", expected, err)
			}
		}
	})
	for _, mutation := range []bool{false, true} {
		t.Run(fmt.Sprintf("native mutation %t", mutation), func(t *testing.T) {
			calls, retries := 0, 0
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.RawQuery != "" || req.Body != nil {
					t.Fatal(req.URL, req.Body)
				}
				if calls == 1 {
					return propertyCoreJSON(req, 503, "busy"), nil
				}
				if req.Header.Get("X-Native") != "retry" {
					t.Fatal("native retry header lost")
				}
				return propertyCoreJSON(req, 200, `{"properties":{"one":{}}}`), nil
			})
			client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				retries++
				if retries > 1 {
					return errors.New("extra retry")
				}
				if mutation {
					opts.JSONBody = map[string]string{"injected": "body"}
				} else {
					if opts.MoreHeaders == nil {
						opts.MoreHeaders = map[string]string{}
					}
					opts.MoreHeaders["X-Native"] = "retry"
				}
				return nil
			}
			rows, err := propertyRecordScope(t, client, Dependencies{}).AllRecords(context.Background())
			if mutation {
				if len(rows) != 0 || !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || calls != 1 || retries != 1 {
					t.Fatal(rows, err, calls, retries)
				}
			} else if err != nil || len(rows) != 1 || calls != 2 || retries != 1 {
				t.Fatal(rows, err, calls, retries)
			}
		})
	}
}

func TestMetadefPropertyListRecordsLegacyFiniteDTOIsSeparate(t *testing.T) {
	calls := 0
	client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.RawQuery != "" {
			t.Fatal("legacy local cap became wire query")
		}
		return propertyCoreJSON(req, 200, `{"properties":{"literal":{"minLength":"03","readonly":"false"}},"next":false}`), nil
	})
	rows, err := propertyCoreScope(t, client, propertyRecordParent).All(context.Background(), WithListMaxItems(1))
	if err != nil || len(rows) != 1 || calls != 1 || rows[0].Key == nil || *rows[0].Key != "literal" || rows[0].Name != nil || string(rows[0].Body["minLength"]) != `"03"` || string(rows[0].Body["readonly"]) != `"false"` {
		t.Fatal(rows, err, calls)
	}
}

func TestMetadefPropertyListRecordsOwnExplicitJSONAcceptAcrossSourceAndRetry(t *testing.T) {
	for _, mode := range []string{"source inherited representation", "retry cannot change representation"} {
		t.Run(mode, func(t *testing.T) {
			calls, retries := 0, 0
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Header.Get("Accept") != "application/json" {
					t.Fatal("explicit list representation lost", req.Header)
				}
				if mode == "retry cannot change representation" {
					return propertyCoreJSON(req, 503, "retry trigger"), nil
				}
				return propertyCoreJSON(req, 200, `{"properties":{}}`), nil
			})
			client.MoreHeaders = map[string]string{"Accept": "text/plain"}
			if mode == "retry cannot change representation" {
				client.ProviderClient.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
					retries++
					opts.MoreHeaders["Accept"] = "text/plain"
					return nil
				}
			}
			rows, err := propertyRecordScope(t, client, Dependencies{}).AllRecords(context.Background())
			if calls != 1 || len(rows) != 0 || client.MoreHeaders["Accept"] != "text/plain" {
				t.Fatal("representation ownership changed caller source or replayed", rows, err, calls)
			}
			if mode == "retry cannot change representation" {
				if err == nil || retries != 1 {
					t.Fatal("native retry changed owned Accept", err, retries)
				}
			} else if err != nil || retries != 0 || rows == nil {
				t.Fatal(rows, err, retries)
			}
		})
	}
}
