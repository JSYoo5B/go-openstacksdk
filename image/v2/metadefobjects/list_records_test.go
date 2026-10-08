package metadefobjects

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

const objectRecordParent = "OS::空 白"

// Only Close needs a callback missing from the existing objectCoreBody fixture.
// All byte delivery, faults, close counting and receipts still use that fixture.
type objectRecordCloseBody struct {
	*objectCoreBody
	after func()
}

func (body *objectRecordCloseBody) Close() error {
	err := body.objectCoreBody.Close()
	if body.after != nil {
		body.after()
	}
	return err
}

func objectRecordBindings(t *testing.T, client *gophercloud.ServiceClient, deps Dependencies) *NamespaceScope {
	t.Helper()
	scope, err := NewWithDependencies(client, deps).InNamespace(context.Background(), objectRecordParent)
	if err != nil {
		t.Fatal(err)
	}
	return scope
}
func objectRecordEnvelope(key, rows, extra string) string {
	return `{"` + key + `":` + rows + extra + `}`
}

func TestMetadefObjectRecordsFixedRoutesProjectionAndReceipts(t *testing.T) {
	t.Run("namespace", func(t *testing.T) {
		calls := 0
		row := `{"name":false,"created_at":900719925474099312345,"updated_at":{"literal":true},"description":[1],"properties":null,"required":false,"namespace_name":"wire parent","location":{"cloud":"wire"},"vendor":1e400}`
		body := objectRecordEnvelope("objects", `[`+row+`,{}]`, "")
		client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			path := "/reverse/glance/v2/metadefs/namespaces/" + url.PathEscape(objectRecordParent) + "/objects"
			if req.Method != http.MethodGet || req.URL.EscapedPath() != path || req.URL.RawQuery != "" || req.Body != nil || req.Header.Get("X-Auth-Token") != "before" || req.Header.Get("Accept") != "application/json" {
				t.Fatalf("fixed list request %s %s %+v", req.Method, req.URL, req.Header)
			}
			return objectCoreJSON(req, 203, body), nil
		})
		scope := objectRecordBindings(t, client, Dependencies{})
		rows, err := scope.AllRecords(context.Background())
		if err != nil || len(rows) != 2 || calls != 1 {
			t.Fatalf("record rows %v %+v calls %d", err, rows, calls)
		}
		first := rows[0]
		want := map[string]string{"id": "false", "name": "false", "created_at": "900719925474099312345", "updated_at": `{"literal":true}`, "location": "null"}
		parent, _ := json.Marshal(objectRecordParent)
		want["namespace_name"] = string(parent)
		want["description"] = "[1]"
		want["properties"] = "null"
		want["required"] = "false"
		if first.Namespace == nil || *first.Namespace != objectRecordParent || rows[1].Namespace == nil || *rows[1].Namespace != objectRecordParent {
			t.Fatal("missing captured namespace", first.Namespace, rows[1].Namespace)
		}
		actual := make(map[string]string)
		for field, raw := range first.Resource.Body {
			actual[field] = string(raw)
		}
		if !reflect.DeepEqual(want, actual) || first.Wire == nil || string(first.Wire.Body["vendor"]) != "1e400" || string(first.Wire.Body["namespace_name"]) != `"wire parent"` {
			t.Fatalf("projection/raw separation %+v %+v", actual, first.Wire)
		}
		for _, record := range rows {
			if record.StatusCode != 203 || record.Resource.StatusCode != 203 || record.Wire.StatusCode != 203 || record.Header.Get("X-Proof") != "original" || record.Resource.Header.Get("X-Proof") != "original" || record.Wire.Header.Get("X-Proof") != "original" || string(record.Envelope) != body {
				t.Fatalf("actual physical-page receipt %+v", record)
			}
		}
		for _, field := range []string{"id", "name", "created_at", "updated_at", "description", "properties", "required", "location"} {
			if string(rows[1].Resource.Body[field]) != "null" {
				t.Fatalf("missing %s acquired value %s", field, rows[1].Resource.Body[field])
			}
		}
		first.Resource.Body["name"][0] = '0'
		first.Resource.Header.Set("X-Proof", "view changed")
		first.Header.Set("X-Proof", "record changed")
		first.Envelope[0] = 'X'
		if string(first.Wire.Body["name"]) != "false" || first.Wire.Header.Get("X-Proof") != "original" || string(rows[1].Envelope) != body || rows[1].Header.Get("X-Proof") != "original" {
			t.Fatal("caller mutations crossed projection, wire or sibling receipt ownership")
		}
	})
}

func TestMetadefObjectRecordsIdentityPresenceAndLocation(t *testing.T) {
	for _, check := range []struct{ name, row, id string }{
		{"name alias", `{"name":"alias"}`, `"alias"`},
		{"present null id", `{"id":null,"name":"alias"}`, "null"},
		{"present empty id", `{"id":"","name":"alias"}`, `""`},
		{"untyped id", `{"id":[900719925474099312345],"name":null}`, "[900719925474099312345]"},
		{"wire location discarded with known field", `{"name":"alias","location":null}`, `"alias"`},
		{"only wire null location", `{"location":null}`, "null"},
		{"only wire untyped location", `{"location":[1]}`, "null"},
	} {
		t.Run("namespace/"+check.name, func(t *testing.T) {
			cloud := "captured"
			location := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"token project"`)}}
			locationCalls := 0
			client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
				return objectCoreJSON(req, 200, objectRecordEnvelope("objects", `[`+check.row+`]`, "")), nil
			})
			scope := objectRecordBindings(t, client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locationCalls++; return location, nil }})
			rows, err := scope.AllRecords(context.Background(), func(*RecordListOpts) error { cloud = "after option"; location.Project.ID[1] = 'X'; return nil })
			if err != nil || len(rows) != 1 || locationCalls != 1 || string(rows[0].Resource.Body["id"]) != check.id {
				t.Fatal(rows, err, locationCalls)
			}
			var decoded resource.CloudLocation
			if err := json.Unmarshal(rows[0].Resource.Body["location"], &decoded); err != nil || decoded.Cloud == nil || *decoded.Cloud != "captured" || string(decoded.Project.ID) != `"token project"` {
				t.Fatal("location was not captured before options", decoded, err)
			}
			var actualWire map[string]json.RawMessage
			if err := json.Unmarshal([]byte(check.row), &actualWire); err != nil {
				t.Fatal(err)
			}
			wantWire, supplied := actualWire["location"]
			gotWire, retained := rows[0].Wire.Body["location"]
			if supplied != retained || string(wantWire) != string(gotWire) {
				t.Fatal("collector projection changed actual wire location", rows[0].Wire.Body)
			}
		})
	}
}

func TestMetadefObjectRecordsLazyRepeatAndOptionSnapshots(t *testing.T) {
	t.Run("namespace", func(t *testing.T) {
		calls, callbacks, locations := 0, 0, 0
		cloud := "first"
		client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			wantSource := "first"
			if calls == 2 {
				wantSource = "second"
			}
			if req.Header.Get("X-Source") != wantSource || req.Header.Get("X-Option") != "factory snapshot" || req.URL.Query().Get("limit") != "2" || req.URL.Query().Get("marker") != "start" || len(req.URL.Query()) != 2 {
				t.Fatal("owned option/source snapshot", req.URL, req.Header)
			}
			return objectCoreJSON(req, 200, objectRecordEnvelope("objects", `[{"name":"match"}]`, `,"next":false`)), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "first"}
		scope := objectRecordBindings(t, client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return resource.CloudLocation{Cloud: &cloud}, nil }})
		paginated := false
		headers := map[string]string{"X-Option": "factory snapshot"}
		filters := map[string]any{"name": "match"}
		owned := WithRecordListOpts(RecordListOpts{Headers: headers, Limit: 2, Marker: "start", Paginated: &paginated, Filters: []resource.ListOption{resource.WithFilters(filters)}})
		seq := scope.ListRecords(context.Background(), owned, func(*RecordListOpts) error {
			callbacks++
			client.MoreHeaders["X-Source"] = "changed by callback"
			cloud = "changed by callback"
			return nil
		})
		headers["X-Option"] = "caller changed"
		filters["name"] = "caller changed"
		paginated = true
		if calls != 0 || callbacks != 0 || locations != 0 {
			t.Fatal("record iterator was eager")
		}
		for iteration := 0; iteration < 2; iteration++ {
			wantCloud := "first"
			if iteration == 1 {
				client.MoreHeaders["X-Source"] = "second"
				cloud = "second"
				wantCloud = "second"
			}
			count := 0
			for row, err := range seq {
				if err != nil {
					t.Fatal(err)
				}
				count++
				var location resource.CloudLocation
				if err := json.Unmarshal(row.Resource.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != wantCloud {
					t.Fatal("repeat did not capture its own location", location, err)
				}
			}
			if count != 1 || calls != iteration+1 || callbacks != iteration+1 || locations != iteration+1 {
				t.Fatal(count, calls, callbacks, locations)
			}
		}
	})
}

func TestMetadefObjectRecordsSemanticFilters(t *testing.T) {
	for _, check := range []struct {
		name, rows string
		options    []RecordListOption
		want       int
	}{
		{"unknown before encoding", `[{}]`, []RecordListOption{WithRecordListFilter("vendor", make(chan int))}, 1},
		{"declared null matches missing", `[{}, {"name":null}, {"name":"named"}]`, []RecordListOption{WithRecordListFilter("name", nil)}, 2},
		{"name alias id", `[{"name":"alias"},{"id":"other","name":"alias"},{"id":null,"name":"alias"}]`, []RecordListOption{WithRecordListFilter("id", "alias")}, 1},
		{"nested subset", `[{"properties":{"nested":{"kept":1,"extra":2}}},{"properties":{"nested":{}}},{"properties":null}]`, []RecordListOption{WithRecordListFilter("properties", map[string]any{"nested": map[string]any{"kept": 1}})}, 1},
		{"required untyped scalar", `[{"required":false},{"required":null},{"required":[]}]`, []RecordListOption{WithRecordListFilter("required", false)}, 1},
		{"properties arbitrary JSON", `[{"properties":[1,{"value":true}]},{"properties":{"value":true}}]`, []RecordListOption{WithRecordListFilter("properties", []any{1, map[string]any{"value": true}})}, 1},
		{"boolean differs from number", `[{"name":true},{"name":1},{"name":1.0}]`, []RecordListOption{WithRecordListFilter("name", true)}, 1},
		{"exact huge numeric equality", `[{"updated_at":900719925474099312345},{"updated_at":900719925474099312346}]`, []RecordListOption{WithRecordListFilter("updated_at", json.RawMessage(`900719925474099312345`))}, 1},
		{"later scalar overrides", `[{"name":"old"},{"name":"new"}]`, []RecordListOption{WithRecordListFilter("name", "old"), WithRecordListFilter("name", "new")}, 1},
		{"bulk replaces prior filters", `[{"name":"new","created_at":"other"}]`, []RecordListOption{WithRecordListFilter("created_at", "excluded"), WithRecordListFilters(map[string]any{"name": "new"})}, 1},
		{"replacement clears ignored encoding failure", `[{"name":"new"}]`, []RecordListOption{WithRecordListFilter("name", make(chan int)), WithRecordListFilter("name", "new")}, 1},
	} {
		t.Run("namespace/"+check.name, func(t *testing.T) {
			calls := 0
			client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.RawQuery != "" {
					t.Fatal("body/unknown filter leaked to wire query", req.URL)
				}
				return objectCoreJSON(req, 200, objectRecordEnvelope("objects", check.rows, "")), nil
			})
			scope := objectRecordBindings(t, client, Dependencies{})
			rows, err := scope.AllRecords(context.Background(), check.options...)
			if err != nil || len(rows) != check.want || calls != 1 {
				t.Fatal(rows, err, calls)
			}
		})
	}
	t.Run("namespace"+"/description filter", func(t *testing.T) {
		client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
			return objectCoreJSON(req, 200, objectRecordEnvelope("objects", `[{"description":"wanted"},{"description":"other"}]`, "")), nil
		})
		scope := objectRecordBindings(t, client, Dependencies{})
		rows, err := scope.AllRecords(context.Background(), WithRecordListFilter("description", "wanted"))
		want := 1
		if err != nil || len(rows) != want {
			t.Fatal(rows, err)
		}
	})
}

func TestMetadefObjectRecordsCompletePreflight(t *testing.T) {
	for _, check := range []struct {
		name    string
		options []RecordListOption
	}{
		{"nil option", []RecordListOption{nil}},
		{"negative limit", []RecordListOption{WithRecordListLimit(-1)}},
		{"negative cap", []RecordListOption{WithRecordListMaxItems(-1)}},
		{"whitespace marker", []RecordListOption{WithRecordListMarker(" \t ")}},
		{"multiple semantic limit", []RecordListOption{WithRecordListFilter("limit", []int{1, 2})}},
		{"semantic zero limit", []RecordListOption{WithRecordListFilter("limit", 0)}},
		{"conflicting typed semantic limit", []RecordListOption{WithRecordListLimit(2), WithRecordListFilter("limit", 2)}},
		{"semantic marker array", []RecordListOption{WithRecordListFilter("marker", []string{"one", "two"})}},
		{"nil semantic option", []RecordListOption{WithRecordListOpts(RecordListOpts{Filters: []resource.ListOption{nil}})}},
		{"non-semantic collection option", []RecordListOption{WithRecordListOpts(RecordListOpts{Filters: []resource.ListOption{resource.WithName("other")}})}},
		{"known encoding failure", []RecordListOption{WithRecordListFilter("name", make(chan int))}},
		{"protected header", []RecordListOption{WithRecordListHeader("X-Auth-Token", "foreign")}},
		{"accept header", []RecordListOption{WithRecordListHeader("Accept", "text/plain")}},
		{"header aliases", []RecordListOption{WithRecordListHeaders(map[string]string{"X-Duplicate": "one", "x-duplicate": "two"})}},
		{"header newline", []RecordListOption{WithRecordListHeader("X-Option", "bad\nvalue")}},
		{"visibility control", []RecordListOption{WithRecordListVisibility("pub\nlic")}},
		{"resource types control", []RecordListOption{WithRecordListResourceTypes("bad\x7f")}},
		{"sort key UTF8", []RecordListOption{WithRecordListSortKey("bad\xff")}},
		{"sort dir control", []RecordListOption{WithRecordListSortDir("bad\rdir")}},
		{"semantic query control", []RecordListOption{WithRecordListFilter("visibility", "bad\nvalue")}},
		{"typed semantic query collision", []RecordListOption{WithRecordListVisibility("public"), WithRecordListFilter("visibility", "public")}},
	} {
		t.Run("namespace/"+check.name, func(t *testing.T) {
			calls := 0
			client := objectCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
			scope := objectRecordBindings(t, client, Dependencies{})
			rows, err := scope.AllRecords(context.Background(), check.options...)
			if err == nil || len(rows) != 0 || calls != 0 {
				t.Fatal(rows, err, calls)
			}
		})
	}
	for _, key := range []string{"base_path", "list_base_path", "headers", "microversion", "session", "namespace_name", "allow_unknown_params", "paginated", "max_items", "resource_type", "namespace", "jmespath_filters"} {
		t.Run("namespace"+"/reserved "+key, func(t *testing.T) {
			calls := 0
			client := objectCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			scope := objectRecordBindings(t, client, Dependencies{})
			rows, err := scope.AllRecords(context.Background(), WithRecordListFilter(key, "unsafe"))
			if err == nil || len(rows) != 0 || calls != 0 {
				t.Fatal(rows, err, calls)
			}
		})
	}
}

func TestMetadefObjectRecordsPreflightOrderAndLocationFailures(t *testing.T) {
	for _, mode := range []string{"nil context", "canceled context", "protected source", "scope lifetime", "option retarget", "location error", "invalid location"} {

		t.Run("namespace/"+mode, func(t *testing.T) {
			calls, callbacks, locations := 0, 0, 0
			cause := errors.New("captured failure")
			ctx := context.Background()
			client := objectCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			scope := objectRecordBindings(t, client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
				locations++
				if mode == "location error" {
					return resource.CloudLocation{}, cause
				}
				if mode == "invalid location" {
					return resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`{`)}}, nil
				}
				return resource.CloudLocation{}, nil
			}})
			switch mode {
			case "nil context":
				ctx = nil
			case "canceled context":
				next, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx = next
			case "protected source":
				client.MoreHeaders = map[string]string{"Authorization": "foreign"}
			case "scope lifetime":
				client.ResourceBase += "changed/"
			}
			rows, err := scope.AllRecords(ctx, func(*RecordListOpts) error {
				callbacks++
				if mode == "option retarget" {
					client.Endpoint = "https://changed.test/"
				}
				return nil
			})
			if err == nil || len(rows) != 0 || calls != 0 {
				t.Fatal(rows, err, calls)
			}
			if mode == "location error" && !errors.Is(err, cause) || mode == "canceled context" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
				t.Fatal("missing cause", err)
			}
			if mode == "option retarget" {
				if callbacks != 1 || locations != 1 {
					t.Fatal(callbacks, locations)
				}
			} else if callbacks != 0 {
				t.Fatal("invalid preflight ran user option", callbacks)
			}
			if mode == "nil context" || mode == "canceled context" || mode == "protected source" || mode == "scope lifetime" {
				if locations != 0 {
					t.Fatal("source preflight invoked location", locations)
				}
			}
		})
	}
}

func TestMetadefObjectRecordsRawCapAndEarlyStop(t *testing.T) {
	for _, mode := range []string{"cap", "break", "single page", "filtered cap", "uncapped bad row"} {
		t.Run("namespace/"+mode, func(t *testing.T) {
			calls := 0
			rowsJSON := `[{"name":"first"},null]`
			if mode == "single page" {
				rowsJSON = `[{"name":"first"},{"name":"second"}]`
			}
			body := objectRecordEnvelope("objects", rowsJSON, `,"next":false,"links":"bad"`)
			client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				want := ""
				if mode == "cap" || mode == "filtered cap" {
					want = "1"
				}
				if req.URL.Query().Get("limit") != want {
					t.Fatal("max-items wire hint", req.URL)
				}
				response := objectCoreJSON(req, 200, body)
				response.Header.Set("Link", "malformed")
				return response, nil
			})
			scope := objectRecordBindings(t, client, Dependencies{})
			options := []RecordListOption{}
			if mode == "cap" || mode == "filtered cap" {
				options = append(options, WithRecordListMaxItems(1))
			}
			if mode == "filtered cap" {
				options = append(options, WithRecordListFilter("name", "excluded"))
			}
			if mode == "single page" {
				options = append(options, WithRecordListPaginated(false))
			}
			count := 0
			var failure error
			for row, err := range scope.ListRecords(context.Background(), options...) {
				if err != nil {
					failure = err
					break
				}
				count++
				if row == nil {
					t.Fatal("nil successful row")
				}
				if mode == "break" {
					break
				}
			}
			want := 1
			if mode == "filtered cap" {
				want = 0
			}
			if mode == "single page" {
				want = 2
			}
			if count != want || calls != 1 {
				t.Fatal(count, calls, failure)
			}
			if mode == "uncapped bad row" {
				if failure == nil {
					t.Fatal("consumed null row accepted")
				}
				objectCoreProof(t, failure, 200, body)
			} else if failure != nil {
				t.Fatal("unconsumed row or continuation decoded", failure)
			}
		})
	}
}

func TestMetadefObjectRecordsMarkerFallbackUsesWireIdentity(t *testing.T) {
	for _, check := range []struct{ name, row, marker string }{{"alternate name", `{"name":" a/b+% "}`, " a/b+% "}, {"explicit id", `{"id":"wire-id","name":"alias"}`, "wire-id"}} {
		t.Run("namespace/"+check.name, func(t *testing.T) {
			calls := 0
			client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.Query().Get("limit") != "5" {
					t.Fatal(req.URL)
				}
				if calls == 1 {
					if req.URL.Query().Has("marker") {
						t.Fatal(req.URL)
					}
					return objectCoreJSON(req, 200, objectRecordEnvelope("objects", `[`+check.row+`]`, "")), nil
				}
				if calls != 2 || req.URL.Query().Get("marker") != check.marker {
					t.Fatal("short-page wire marker changed", req.URL, calls)
				}
				return objectCoreJSON(req, 200, objectRecordEnvelope("objects", `[]`, `,"next":false`)), nil
			})
			scope := objectRecordBindings(t, client, Dependencies{})
			count := 0
			for row, err := range scope.ListRecords(context.Background(), WithRecordListLimit(5)) {
				if err != nil {
					t.Fatal(err)
				}
				count++
				row.Resource.Body["id"] = json.RawMessage(`"caller"`)
				row.Wire.Body["id"] = json.RawMessage(`"caller"`)
				row.Wire.Body["name"] = json.RawMessage(`"caller"`)
				row.Envelope[0] = 'X'
				row.Header.Set("Link", "bad")
			}
			if count != 1 || calls != 2 {
				t.Fatal(count, calls)
			}
		})
	}
	for _, row := range []string{`{}`, `{"id":null,"name":"alias"}`, `{"id":false,"name":"alias"}`, `{"id":""}`, `{"name":"  "}`, `{"name":42}`} {
		t.Run("namespace"+"/invalid marker "+row, func(t *testing.T) {
			calls := 0
			body := objectRecordEnvelope("objects", `[`+row+`]`, "")
			client := objectCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return objectCoreJSON(req, 200, body), nil })
			scope := objectRecordBindings(t, client, Dependencies{})
			rows, err := scope.AllRecords(context.Background(), WithRecordListLimit(5))
			if len(rows) != 1 || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
				t.Fatal(rows, err, calls)
			}
			objectCoreProof(t, err, 200, body)
		})
	}
}

func TestMetadefObjectRecordsContinuationRepresentations(t *testing.T) {
	for _, mode := range []string{"links array", "plural links", "links dictionary", "next field", "HTTP Link", "equivalent all forms"} {
		t.Run("namespace/"+mode, func(t *testing.T) {
			calls := 0
			client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.Query().Get("limit") != "4" {
					t.Fatal("omitted initial limit was not inherited", req.URL)
				}
				if calls == 2 {
					if req.URL.Query().Get("marker") != "next" {
						t.Fatal(req.URL)
					}
					return objectCoreJSON(req, 200, objectRecordEnvelope("objects", `[]`, "")), nil
				}
				if calls != 1 {
					t.Fatal("extra page", calls)
				}
				extra := ""
				switch mode {
				case "links array":
					extra = `,"links":[{"rel":"next","href":"?marker=next"}]`
				case "plural links":
					extra = `,"` + "objects" + `_links":[{"rel":"next","href":"?marker=next"}]`
				case "links dictionary":
					extra = `,"links":{"next":"?marker=next"}`
				case "next field":
					extra = `,"next":"?marker=next"`
				case "equivalent all forms":
					extra = `,"links":[{"rel":"next","href":"?marker=next"}],"` + "objects" + `_links":[{"rel":"next","href":"?limit=4&marker=next"}],"next":"?marker=next"`
				}
				response := objectCoreJSON(req, 200, objectRecordEnvelope("objects", `[{"name":"first"}]`, extra))
				if mode == "HTTP Link" || mode == "equivalent all forms" {
					response.Header.Set("Link", `<?marker=next>; rel="next"`)
				}
				return response, nil
			})
			scope := objectRecordBindings(t, client, Dependencies{})
			rows, err := scope.AllRecords(context.Background(), WithRecordListLimit(4))
			if err != nil || len(rows) != 1 || calls != 2 {
				t.Fatal(rows, err, calls)
			}
		})
	}
}

func TestMetadefObjectRecordsRejectedContinuationKeepsPartialReceipt(t *testing.T) {
	for _, check := range []struct {
		name, extra, header string
		cycle               bool
	}{
		{"foreign origin", `,"next":"https://foreign.test/metadefs/objects?marker=next"`, "", false},
		{"different collection", `,"next":"?marker=next&vendor=added"`, "", false},
		{"changed path", `,"next":"other?marker=next"`, "", false},
		{"fragment", `,"next":"?marker=next#fragment"`, "", false},
		{"changed limit", `,"next":"?marker=next&limit=6"`, "", false},
		{"multiple marker", `,"next":"?marker=one&marker=two"`, "", false},
		{"malformed next", `,"next":false`, "", false},
		{"conflicting representations", `,"next":"?marker=one","links":[{"rel":"next","href":"?marker=two"}]`, "", false},
		{"malformed HTTP Link", "", "malformed", false},
		{"repeated initial marker", `,"next":"?marker=seed"`, "", true},
	} {
		t.Run("namespace/"+check.name, func(t *testing.T) {
			calls := 0
			body := objectRecordEnvelope("objects", `[{"name":"first"}]`, check.extra)
			client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				response := objectCoreJSON(req, 200, body)
				if check.header != "" {
					response.Header.Set("Link", check.header)
				}
				return response, nil
			})
			scope := objectRecordBindings(t, client, Dependencies{})
			rows, err := scope.AllRecords(context.Background(), WithRecordListLimit(4), WithRecordListMarker("seed"))
			if len(rows) != 1 || err == nil || calls != 1 {
				t.Fatal(rows, err, calls)
			}
			proof := objectCoreProof(t, err, 200, body)
			if check.cycle {
				var cycle *resource.PaginationCycleError
				if !errors.As(err, &cycle) {
					t.Fatal("cycle cause missing", err)
				}
			}
			rows[0].Envelope[0] = 'X'
			rows[0].Header.Set("X-Proof", "changed")
			if string(proof.Body) != body || proof.Header.Get("X-Proof") != "original" {
				t.Fatal("partial row aliases failure receipt")
			}
		})
	}
}

func TestMetadefObjectRecordsStatusEnvelopeAndEmptyPolicy(t *testing.T) {
	for _, check := range []struct {
		name, rows string
		code       int
		good       bool
	}{
		{"200 array", `[{"name":"one"}]`, 200, true}, {"201 singleton", `{"name":"one"}`, 201, true}, {"299 array", `[{"name":"one"}]`, 299, true}, {"300 body", `[{"name":"one"}]`, 300, true}, {"399 body", `[{"name":"one"}]`, 399, true},
		{"null plural", `null`, 200, false}, {"scalar plural", `false`, 200, false}, {"bad row", `[null]`, 200, false},
	} {
		t.Run("namespace/"+check.name, func(t *testing.T) {
			body := objectRecordEnvelope("objects", check.rows, "")
			client := objectCoreClient(func(req *http.Request) (*http.Response, error) { return objectCoreJSON(req, check.code, body), nil })
			scope := objectRecordBindings(t, client, Dependencies{})
			rows, err := scope.AllRecords(context.Background())
			if check.good {
				if err != nil || len(rows) != 1 || rows[0].StatusCode != check.code {
					t.Fatal(rows, err)
				}
			} else {
				if err == nil || len(rows) != 0 {
					t.Fatal(rows, err)
				}
				objectCoreProof(t, err, check.code, body)
			}
		})
	}
	for _, body := range []string{"", `null`, `[]`, `{}`, `{"broken":`, "{\"" + "objects" + "\":[],\"bad\":\"\xff\"}"} {
		t.Run("namespace"+"/invalid root "+fmt.Sprintf("%q", body), func(t *testing.T) {
			client := objectCoreClient(func(req *http.Request) (*http.Response, error) { return objectCoreJSON(req, 204, body), nil })
			scope := objectRecordBindings(t, client, Dependencies{})
			rows, err := scope.AllRecords(context.Background())
			if err == nil || len(rows) != 0 {
				t.Fatal(rows, err)
			}
			objectCoreProof(t, err, 204, body)
		})
	}
	t.Run("namespace"+"/empty ignores next", func(t *testing.T) {
		calls := 0
		client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			return objectCoreJSON(req, 200, objectRecordEnvelope("objects", `[]`, `,"next":false`)), nil
		})
		scope := objectRecordBindings(t, client, Dependencies{})
		rows, err := scope.AllRecords(context.Background())
		if err != nil || rows == nil || len(rows) != 0 || calls != 1 {
			t.Fatal("successful empty materialization", rows, err, calls)
		}
	})
}

func TestMetadefObjectRecordsAcceptedReadCloseAndCancelCauses(t *testing.T) {
	for _, mode := range []string{"read", "close", "read close cancel"} {
		t.Run("namespace/"+mode, func(t *testing.T) {
			calls, retries := 0, 0
			readErr, closeErr, cancelCause := errors.New("accepted read"), errors.New("accepted close"), errors.New("accepted cancel")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			bodyText := objectRecordEnvelope("objects", `[{"name":"one"}]`, "")
			body := &objectCoreBody{reader: strings.NewReader(bodyText)}
			if mode != "close" {
				body.reader = &objectCoreReader{data: bodyText, err: readErr}
			}
			if mode != "read" {
				body.closeErr = closeErr
			}
			if mode == "read close cancel" {
				body.reader = &objectCoreReader{data: bodyText, err: readErr, after: func() { cancel(cancelCause) }}
			}
			client := objectCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return objectCoreHTTP(req, 203, body), nil })
			client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			scope := objectRecordBindings(t, client, Dependencies{})
			rows, err := scope.AllRecords(ctx)
			if len(rows) != 0 || err == nil || calls != 1 || retries != 0 || body.closes != 1 {
				t.Fatal(rows, err, calls, retries, body.closes)
			}
			objectCoreProof(t, err, 203, bodyText)
			if mode != "close" && !errors.Is(err, readErr) || mode != "read" && !errors.Is(err, closeErr) {
				t.Fatal("accepted body cause missing", err)
			}
			if mode == "read close cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
				t.Fatal("cancel cause missing", err)
			}
		})
	}
}

func TestMetadefObjectRecordsNativeRetryAndOwnership(t *testing.T) {
	for _, mode := range []string{"ordinary retry", "request body mutation", "source retarget", "status policy expansion"} {
		t.Run("namespace/"+mode, func(t *testing.T) {
			calls, retries := 0, 0
			client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Header.Get("X-Auth-Token") != "live" || req.URL.Query().Get("limit") != "3" || req.Body != nil {
					t.Fatal(req.URL, req.Header, req.Body)
				}
				if calls == 1 {
					return objectCoreJSON(req, 503, "busy"), nil
				}
				if mode == "status policy expansion" {
					return objectCoreJSON(req, 400, "outside original policy"), nil
				}
				if req.Header.Get("X-Native") != "retry" {
					t.Fatal("native unrelated retry header lost")
				}
				return objectCoreJSON(req, 200, objectRecordEnvelope("objects", `[]`, "")), nil
			})
			scope := objectRecordBindings(t, client, Dependencies{})
			client.ProviderClient.TokenID = "live"
			client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				retries++
				if retries > 1 {
					return errors.New("unexpected additional retry")
				}
				switch mode {
				case "ordinary retry":
					if opts.MoreHeaders == nil {
						opts.MoreHeaders = map[string]string{}
					}
					opts.MoreHeaders["X-Native"] = "retry"
				case "request body mutation":
					opts.JSONBody = map[string]string{"injected": "body"}
				case "source retarget":
					client.ResourceBase += "changed/"
				case "status policy expansion":
					opts.OkCodes = append(opts.OkCodes, 400)
				}
				return nil
			}
			rows, err := scope.AllRecords(context.Background(), WithRecordListLimit(3))
			if len(rows) != 0 || retries != 1 {
				t.Fatal(rows, err, retries)
			}
			switch mode {
			case "ordinary retry":
				if err != nil || rows == nil || calls != 2 {
					t.Fatal(rows, err, calls)
				}
			case "status policy expansion":
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 400 || string(native.Body) != "outside original policy" || calls != 2 {
					t.Fatal(native, err, calls)
				}
			default:
				if !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || calls != 1 {
					t.Fatal(err, calls)
				}
			}
		})
	}
}

func TestMetadefObjectRecordsSourceAndOuterGuardAcrossPhysicalBoundaries(t *testing.T) {
	for _, boundary := range []string{"round trip", "read", "close", "after yield", "outer read", "outer close", "outer after yield"} {
		t.Run("namespace/"+boundary, func(t *testing.T) {
			calls := 0
			bodyText := objectRecordEnvelope("objects", `[{"name":"one"},{"name":"two"}]`, "")
			outerCause := errors.New("outer invariant changed")
			outerFailed := false
			ctx := rest.WithOperationGuard(context.Background(), func(context.Context) error {
				if outerFailed {
					return outerCause
				}
				return nil
			})
			var client *gophercloud.ServiceClient
			client = objectCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if boundary == "round trip" {
					client.Endpoint = "https://changed.test/"
				}
				body := &objectCoreBody{reader: strings.NewReader(bodyText)}
				if boundary == "read" || boundary == "outer read" {
					body.reader = &objectCoreReader{data: bodyText, err: io.EOF, after: func() {
						if boundary == "read" {
							client.Microversion = "2.3"
						} else {
							outerFailed = true
						}
					}}
				}
				if boundary == "close" || boundary == "outer close" {
					return objectCoreHTTP(req, 200, &objectRecordCloseBody{objectCoreBody: body, after: func() {
						if boundary == "close" {
							client.ResourceBase += "changed/"
						} else {
							outerFailed = true
						}
					}}), nil
				}
				return objectCoreHTTP(req, 200, body), nil
			})
			scope := objectRecordBindings(t, client, Dependencies{})
			count, failures := 0, 0
			for _, err := range scope.ListRecords(ctx) {
				if err != nil {
					failures++
					objectCoreProof(t, err, 200, bodyText)
					if strings.HasPrefix(boundary, "outer") {
						if !errors.Is(err, outerCause) {
							t.Fatal(err)
						}
					} else if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
					continue
				}
				count++
				if boundary == "after yield" {
					client.ProviderClient = &gophercloud.ProviderClient{}
				}
				if boundary == "outer after yield" {
					outerFailed = true
				}
			}
			want := 0
			if strings.Contains(boundary, "after yield") {
				want = 1
			}
			if calls != 1 || count != want || failures != 1 {
				t.Fatal(calls, count, failures)
			}
		})
	}
}

func TestMetadefObjectRecordsLegacyFiniteListRemainsSeparate(t *testing.T) {
	calls := 0
	client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.RawQuery != "" {
			t.Fatal("legacy local cap became a wire query", req.URL)
		}
		return objectCoreJSON(req, 200, `{"objects":[{"name":"one"}],"next":"https://foreign.test/"}`), nil
	})
	rows, err := objectCoreScope(t, client, objectRecordParent).All(context.Background(), WithListMaxItems(2))
	if err != nil || len(rows) != 1 || calls != 1 || rows[0].Name == nil || *rows[0].Name != "one" {
		t.Fatal(rows, err, calls)
	}
}

func TestMetadefObjectRecordsFullOptionsAndRetainedCallbacksOwnStorage(t *testing.T) {
	t.Run("namespace", func(t *testing.T) {
		calls := 0
		client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Header.Get("X-Full") != "original" || req.Header.Get("X-Bulk") != "factory" || req.Header.Get("X-Final") != "after replacement" || req.Header.Get("X-Replaced") != "" || req.URL.Query().Get("limit") != "7" || req.URL.Query().Get("marker") != "snapshot" {
				t.Fatal("full replacement/retained option lost ownership", req.URL, req.Header)
			}
			return objectCoreJSON(req, 200, objectRecordEnvelope("objects", `[{"name":"kept"},{"name":"other"}]`, `,"next":false`)), nil
		})
		scope := objectRecordBindings(t, client, Dependencies{})
		paginated := false
		filters := []resource.ListOption{resource.WithFilter("name", "kept")}
		fullHeaders := map[string]string{"X-Full": "original"}
		bulkHeaders := map[string]string{"X-Bulk": "factory"}
		full := WithRecordListOpts(RecordListOpts{Headers: fullHeaders, Limit: 7, Marker: "snapshot", Paginated: &paginated, Filters: filters})
		bulk := WithRecordListHeaders(bulkHeaders)
		var retained *RecordListOpts
		options := []RecordListOption{
			WithRecordListHeader("X-Replaced", "old"), WithRecordListLimit(99), full, bulk,
			func(config *RecordListOpts) error { retained = config; return nil },
			func(config *RecordListOpts) error {
				retained.Headers["X-Full"] = "retained mutation"
				retained.Filters[0] = resource.WithFilter("name", "excluded")
				*retained.Paginated = true
				retained.Limit = 100
				return nil
			},
			WithRecordListHeader("x-final", "after replacement"),
		}
		seq := scope.ListRecords(context.Background(), options...)
		options[0] = WithRecordListHeader("X-Replaced", "caller changed")
		filters[0] = resource.WithFilter("name", "caller changed")
		fullHeaders["X-Full"] = "caller changed"
		bulkHeaders["X-Bulk"] = "caller changed"
		paginated = true
		rows := make([]*Record, 0)
		for row, err := range seq {
			if err != nil {
				t.Fatal(err)
			}
			rows = append(rows, row)
		}
		if len(rows) != 1 || calls != 1 || string(rows[0].Resource.Body["name"]) != `"kept"` {
			t.Fatal(rows, calls)
		}
	})
}

func TestMetadefObjectRecordsObservedGuardFailureSurvivesRestoration(t *testing.T) {
	for _, outer := range []bool{false, true} {
		t.Run(fmt.Sprintf("%s/outer %t", "namespace", outer), func(t *testing.T) {
			calls := 0
			outerFailed := false
			cause := errors.New("observed outer failure")
			ctx := rest.WithOperationGuard(context.Background(), func(context.Context) error {
				if outerFailed {
					return cause
				}
				return nil
			})
			bodyText := objectRecordEnvelope("objects", `[{"name":"one"}]`, "")
			var client *gophercloud.ServiceClient
			var body *objectCoreBody
			client = objectCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				body = &objectCoreBody{reader: &objectCoreReader{data: bodyText, err: io.EOF, after: func() {
					if outer {
						outerFailed = true
					} else {
						client.Type = "compute"
					}
					observed := rest.CheckOperationGuard(req.Context())
					if outer && !errors.Is(observed, cause) || !outer && !errors.Is(observed, resource.ErrInvalidOption) {
						t.Errorf("mutation was not observed by the operation guard: %v", observed)
					}
				}}}
				return objectCoreHTTP(req, 200, &objectRecordCloseBody{objectCoreBody: body, after: func() { outerFailed = false; client.Type = "image" }}), nil
			})
			scope := objectRecordBindings(t, client, Dependencies{})
			rows, err := scope.AllRecords(ctx)
			if len(rows) != 0 || err == nil || calls != 1 || body.closes != 1 || outerFailed || client.Type != "image" {
				t.Fatal(rows, err, calls, body.closes)
			}
			if outer && !errors.Is(err, cause) || !outer && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("restoration cleared an observed failure", err)
			}
			objectCoreProof(t, err, 200, bodyText)
		})
	}
}

func TestMetadefObjectRecordsQueryExtensionsAndFixedContinuation(t *testing.T) {
	for _, semantic := range []bool{false, true} {
		t.Run(fmt.Sprintf("semantic %t", semantic), func(t *testing.T) {
			calls := 0
			want := url.Values{"visibility": {"public"}, "resource_types": {"OS::Compute::Server"}, "sort_key": {"created_at"}, "sort_dir": {"desc"}}
			client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				actual := req.URL.Query()
				if calls == 2 {
					if actual.Get("marker") != "second" || actual.Get("limit") != "4" {
						t.Fatal(req.URL)
					}
					actual.Del("marker")
					actual.Del("limit")
				}
				if !reflect.DeepEqual(actual, want) || req.Body != nil {
					t.Fatal("query extension/continuation changed", req.URL, req.Body)
				}
				if calls == 1 {
					return objectCoreJSON(req, 203, `{"objects":[{"name":"first"}],"next":"?marker=second&limit=4"}`), nil
				}
				if calls != 2 {
					t.Fatal(calls)
				}
				return objectCoreJSON(req, 299, `{"objects":[{"name":"second"}]}`), nil
			})
			scope := objectCoreScope(t, client, objectRecordParent)
			options := []RecordListOption{WithRecordListVisibility("public"), WithRecordListResourceTypes("OS::Compute::Server"), WithRecordListSortKey("created_at"), WithRecordListSortDir("desc")}
			if semantic {
				options = []RecordListOption{WithRecordListFilters(map[string]any{"visibility": "public", "resource_types": "OS::Compute::Server", "sort_key": "created_at", "sort_dir": "desc"})}
			}
			rows, err := scope.AllRecords(context.Background(), options...)
			th.AssertNoErr(t, err)
			th.AssertEquals(t, 2, len(rows))
			th.AssertEquals(t, 2, calls)
			th.AssertEquals(t, 299, rows[1].StatusCode)
		})
	}
}

func TestMetadefObjectRecordsNamespaceGuardAndRowProvenance(t *testing.T) {
	for _, mode := range []string{"option", "read restored", "after yield"} {
		t.Run(mode, func(t *testing.T) {
			calls, count := 0, 0
			text := `{"objects":[{"name":"first","namespace_name":"wire parent"},{"name":"second"}]}`
			var scope *NamespaceScope
			client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				body := &objectCoreBody{reader: strings.NewReader(text)}
				if mode == "read restored" {
					body.reader = &objectCoreReader{data: text, err: io.EOF, after: func() {
						scope.namespace = "changed"
						if err := rest.CheckOperationGuard(req.Context()); !errors.Is(err, resource.ErrInvalidOption) {
							t.Errorf("namespace mutation was not observed: %v", err)
						}
					}}
					return objectCoreHTTP(req, 203, &objectRecordCloseBody{objectCoreBody: body, after: func() { scope.namespace = objectRecordParent }}), nil
				}
				return objectCoreHTTP(req, 203, body), nil
			})
			scope = objectCoreScope(t, client, objectRecordParent)
			var failure error
			options := []RecordListOption{}
			if mode == "option" {
				options = append(options, func(*RecordListOpts) error { scope.namespace = "changed"; return nil })
			}
			for row, err := range scope.ListRecords(context.Background(), options...) {
				if err != nil {
					failure = err
					break
				}
				count++
				if row.Namespace == nil || *row.Namespace != objectRecordParent || string(row.Resource.Body["namespace_name"]) != `"OS::空 白"` {
					t.Fatal(row)
				}
				row.Namespace = new(string)
				row.Resource.Body["namespace_name"] = json.RawMessage(`"caller parent"`)
				if mode == "after yield" {
					scope.namespace = "changed"
				}
			}
			if !errors.Is(failure, resource.ErrInvalidOption) {
				t.Fatal(failure)
			}
			switch mode {
			case "option":
				if calls != 0 || count != 0 {
					t.Fatal(calls, count)
				}
			case "read restored":
				if calls != 1 || count != 0 || scope.namespace != objectRecordParent {
					t.Fatal(calls, count, scope.namespace)
				}
				objectCoreProof(t, failure, 203, text)
			case "after yield":
				if calls != 1 || count != 1 {
					t.Fatal(calls, count)
				}
				objectCoreProof(t, failure, 203, text)
			}
		})
	}
}

func TestMetadefObjectRecordsConstructorBindingFailuresKeepPartialReceipt(t *testing.T) {
	for _, key := range []string{"connection", "_synchronized", "microversion"} {
		for _, status := range []int{200, 203, 399} {
			t.Run(fmt.Sprintf("%s/status %d", key, status), func(t *testing.T) {
				calls := 0
				text := `{"objects":[{"name":"kept"},{"name":"failed","` + key + `":null},{"name":"unconsumed"}]}`
				client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					return objectCoreJSON(req, status, text), nil
				})
				rows, err := objectCoreScope(t, client, objectRecordParent).AllRecords(context.Background())
				if len(rows) != 1 || calls != 1 || !errors.Is(err, resource.ErrInvalidOption) || string(rows[0].Resource.Body["name"]) != `"kept"` || rows[0].StatusCode != status {
					t.Fatal(rows, err, calls)
				}
				proof := objectCoreProof(t, err, status, text)
				rows[0].Envelope[0] = 'X'
				rows[0].Header.Set("X-Proof", "caller changed")
				if string(proof.Body) != text || proof.Header.Get("X-Proof") != "original" {
					t.Fatal("partial record changed constructor failure receipt", proof)
				}
			})
		}
	}
	t.Run("self and other unknown fields remain passive", func(t *testing.T) {
		text := `{"objects":[{"name":"kept","self":{"connection":"ignored"},"base_path":"https://foreign.test/","headers":{"Authorization":"foreign"},"vendor":1e400}]}`
		client := objectCoreClient(func(req *http.Request) (*http.Response, error) { return objectCoreJSON(req, 203, text), nil })
		rows, err := objectCoreScope(t, client, objectRecordParent).AllRecords(context.Background())
		th.AssertNoErr(t, err)
		th.AssertEquals(t, 1, len(rows))
		th.AssertEquals(t, 9, len(rows[0].Resource.Body))
		if string(rows[0].Wire.Body["self"]) != `{"connection":"ignored"}` || string(rows[0].Wire.Body["vendor"]) != "1e400" {
			t.Fatal(rows[0].Wire)
		}
		for _, key := range []string{"self", "base_path", "headers", "vendor"} {
			if _, present := rows[0].Resource.Body[key]; present {
				t.Fatal("unknown row field became a declared attribute", key)
			}
		}
	})
}
