package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const snapshotReadContractBase = "/proxy/cinder/v3/snapshot-project/"
const snapshotReadContractDetail = snapshotReadContractBase + "snapshots/detail"
const snapshotReadContractBasic = snapshotReadContractBase + "snapshots"

func snapshotReadContractClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("volumev3", "/catalog/cinder/")
	client.ResourceBase = cloud.Server.URL + snapshotReadContractBase
	client.Microversion = "3.60"
	client.MoreHeaders = map[string]string{"x-source": "entry"}
	return client
}
func snapshotReadContractWire(t *testing.T, r *http.Request, path, token string) {
	t.Helper()
	if r.Method != http.MethodGet || r.URL.Path != path || r.Header.Get("X-Source") != "entry" || r.Header.Get("X-Auth-Token") != token || r.Header.Get("OpenStack-API-Version") != "volume 3.60" {
		t.Error("fixed snapshot source request changed", r.Method, r.URL, r.Header)
	}
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) != 0 {
			t.Error("snapshot GET has a body", string(body), err)
		}
	}
}
func snapshotReadContractPage(rows, next string) string {
	if next == "" {
		return `{"snapshots":` + rows + `}`
	}
	href, _ := json.Marshal(next)
	return `{"snapshots":` + rows + `,"snapshots_links":[{"rel":"next","href":` + string(href) + `}]}`
}
func snapshotReadContractFields(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(string(raw), err)
	}
	return fields
}
func snapshotReadContractRows(t *testing.T, raw json.RawMessage) []map[string]json.RawMessage {
	t.Helper()
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(string(raw), err)
	}
	return rows
}
func snapshotReadContractOperation(t *testing.T, err error, name string) {
	t.Helper()
	var operation *resource.OperationError
	if !errors.As(err, &operation) || operation.Operation != name || operation.Cause == nil {
		t.Fatal("public operation context lost", err)
	}
}

func TestVolumeSnapshotReadListDefaultDetailedBasicAndCanonicalEmptyEOF(t *testing.T) {
	for _, basic := range []bool{false, true} {
		t.Run(map[bool]string{false: "default detailed", true: "explicit basic"}[basic], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			body := `{"snapshots":[],"links":[null],"next":"https://unused.invalid/"}`
			path := snapshotReadContractDetail
			var options []blockstorage.VolumeSnapshotListOption
			if basic {
				path = snapshotReadContractBasic
				options = append(options, blockstorage.WithVolumeSnapshotListDetailed(false))
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				snapshotReadContractWire(t, r, path, "test-token")
				if r.URL.RawQuery != "" {
					t.Error("initial cloud list query must be empty", r.URL)
				}
				w.Header().Set("X-Proof", "empty")
				testcloud.JSON(w, 203, body)
			})
			result, err := blockstorage.ListVolumeSnapshots(context.Background(), client, options...)
			if err != nil || result == nil || string(result.Value) != "[]" || result.Snapshots == nil || len(result.Snapshots) != 0 || len(result.Pages) != 1 || result.Pages[0].StatusCode != 203 || string(result.Pages[0].Body) != body || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestVolumeSnapshotReadListMaterializesAllPagesAndOwnsRawAndNormalizedRows(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls atomic.Int32
	first := snapshotReadContractPage(`[{"id":"duplicate","name":"first","force":"TrUe","size":"003","metadata":[1],"unknown":9007199254740993}]`, "?marker=second")
	second := snapshotReadContractPage(`[{"id":"duplicate","name":"second","size":true,"metadata":null,"location":{"ignored":true},"availability_zone":"ignored"}]`, "")
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		snapshotReadContractWire(t, r, snapshotReadContractDetail, "test-token")
		switch calls.Add(1) {
		case 1:
			if r.URL.RawQuery != "" {
				t.Error(r.URL)
			}
			w.Header().Set("X-Proof", "first")
			testcloud.JSON(w, 200, first)
		case 2:
			if r.URL.Query().Get("marker") != "second" {
				t.Error(r.URL)
			}
			w.Header().Set("X-Proof", "second")
			testcloud.JSON(w, 203, second)
		default:
			t.Error("list restarted or fetched a member", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.ListVolumeSnapshots(context.Background(), client)
	if err != nil || result == nil || len(result.Pages) != 2 || len(result.Snapshots) != 2 || result.Snapshots[0] == result.Snapshots[1] || calls.Load() != 2 {
		t.Fatal(result, err, calls.Load())
	}
	rows := snapshotReadContractRows(t, result.Value)
	if len(rows) != 2 || len(rows[0]) != 16 || string(rows[0]["is_forced"]) != "true" || string(rows[0]["size"]) != "3" || string(rows[0]["metadata"]) != "{}" || string(rows[1]["size"]) != "true" || string(rows[1]["metadata"]) != "null" || string(rows[0]["name"]) != `"first"` || string(rows[1]["name"]) != `"second"` {
		t.Fatal(string(result.Value))
	}
	if _, exists := rows[0]["unknown"]; exists {
		t.Fatal("wire extension entered normalized view")
	}
	if string(result.Snapshots[0].Body["unknown"]) != "9007199254740993" || result.Snapshots[1].StatusCode != 203 || result.Snapshots[1].Header.Get("X-Proof") != "second" {
		t.Fatal(result)
	}
	location := snapshotReadContractFields(t, rows[1]["location"])
	if string(location["zone"]) != "null" {
		t.Fatal("wire zone entered connected Snapshot location", string(rows[1]["location"]))
	}
	result.Snapshots[0].Body["id"][0] = '!'
	result.Snapshots[0].Header.Set("X-Proof", "raw mutated")
	result.Pages[0].Body[0] = '!'
	result.Pages[1].Header.Set("X-Proof", "page mutated")
	if !json.Valid(result.Value) || string(result.Snapshots[1].Body["id"]) != `"duplicate"` || result.Snapshots[1].Header.Get("X-Proof") != "second" || result.Pages[0].Header.Get("X-Proof") != "first" || string(result.Pages[1].Body) != second {
		t.Fatal("result views share mutable storage")
	}
}

func TestVolumeSnapshotReadListSeparatesServerFiltersLocalSubsetsAndUnknownFields(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls atomic.Int32
	filters := json.RawMessage(`{"name":"wire name","status":"available","volume_id":"parent","all_projects":true,"all_tenants":false,"project_id":false,"limit":0,"metadata":{"flag":1,"nested":{"one":true}},"description":"wanted","force":"bad ignored alias","unknown_route":"drop"}`)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		snapshotReadContractWire(t, r, snapshotReadContractDetail, "test-token")
		want := map[string][]string{"name": {"wire name"}, "status": {"available"}, "volume_id": {"parent"}, "all_tenants": {"True"}, "project_id": {"False"}, "limit": {"0"}}
		if !reflect.DeepEqual(map[string][]string(r.URL.Query()), want) {
			t.Error("query/local/control separation changed", r.URL.Query(), want)
		}
		testcloud.JSON(w, 200, snapshotReadContractPage(`[{"id":"keep","description":"wanted","metadata":{"flag":true,"nested":{"one":1,"extra":2},"extra":3}},{"id":"falsey","description":"wanted","metadata":{}},{"id":"exclude","description":"other","metadata":{"flag":1,"nested":{"one":true}}}]`, ""))
	})
	result, err := blockstorage.ListVolumeSnapshots(context.Background(), client, blockstorage.WithVolumeSnapshotListFilters(filters))
	if err != nil || result == nil || len(result.Snapshots) != 1 || string(result.Snapshots[0].Body["id"]) != `"keep"` || len(snapshotReadContractRows(t, result.Value)) != 1 || len(result.Pages) != 1 || calls.Load() != 1 {
		t.Fatal(result, err, calls.Load())
	}
}

func TestVolumeSnapshotReadListEagerDescriptorsFailBeforeLocalExclusionAndContinuation(t *testing.T) {
	for _, bad := range []string{`{"id":"bad","description":"excluded","force":"bad"}`, `{"id":"bad","description":"excluded","size":"²"}`, `false`, `null`} {
		t.Run(bad, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			body := snapshotReadContractPage(`[{"id":"good","description":"wanted"},`+bad+`]`, "?marker=must-not-fetch")
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Proof", "eager")
				testcloud.JSON(w, 203, body)
			})
			result, err := blockstorage.ListVolumeSnapshots(context.Background(), client, blockstorage.WithVolumeSnapshotListFilters(json.RawMessage(`{"description":"wanted"}`)))
			var physical *resource.ResponseError
			if result == nil || result.Value != nil || result.Snapshots != nil || len(result.Pages) != 1 || calls.Load() != 1 || !errors.As(err, &physical) || physical.StatusCode != 203 || string(physical.Body) != body || physical.Header.Get("X-Proof") != "eager" {
				t.Fatal(result, err, calls.Load())
			}
			snapshotReadContractOperation(t, err, "ListVolumeSnapshots")
			physical.Body[0] = '!'
			physical.Header.Set("X-Proof", "error mutation")
			if string(result.Pages[0].Body) != body || result.Pages[0].Header.Get("X-Proof") != "eager" {
				t.Fatal("physical error shares page bytes")
			}
		})
	}
}

func TestVolumeSnapshotReadListConstructorControlsArePresenceSensitiveAndSelfIsIgnored(t *testing.T) {
	for _, key := range []string{"connection", "microversion", "_synchronized", "self"} {
		t.Run(key, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			body := snapshotReadContractPage(`[{"id":"row","`+key+`":null}]`, "")
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, body) })
			result, err := blockstorage.ListVolumeSnapshots(context.Background(), client)
			if key == "self" {
				if err != nil || result == nil || len(result.Snapshots) != 1 || string(result.Snapshots[0].Body[key]) != "null" {
					t.Fatal(result, err)
				}
			} else {
				var physical *resource.ResponseError
				if result == nil || result.Value != nil || result.Snapshots != nil || len(result.Pages) != 1 || !errors.As(err, &physical) || string(physical.Body) != body {
					t.Fatal(result, err)
				}
			}
			if calls.Load() != 1 {
				t.Fatal(calls.Load())
			}
		})
	}
}

func TestVolumeSnapshotReadListWrapsCanonicalSingletonAndRejectsInvalidCollections(t *testing.T) {
	for _, body := range []string{`{"snapshots":{"id":"singleton"}}`, `{"snapshots":null}`, `{"snapshots":false}`, `{"Snapshots":[]}`, `null`, `[]`, `{"snapshots":`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, body) })
			result, err := blockstorage.ListVolumeSnapshots(context.Background(), client)
			if body == `{"snapshots":{"id":"singleton"}}` {
				if err != nil || result == nil || len(result.Snapshots) != 1 || string(result.Snapshots[0].Body["id"]) != `"singleton"` {
					t.Fatal(result, err)
				}
			} else {
				var physical *resource.ResponseError
				if result == nil || result.Value != nil || result.Snapshots != nil || len(result.Pages) != 1 || !errors.As(err, &physical) || string(physical.Body) != body {
					t.Fatal(result, err)
				}
			}
			if calls.Load() != 1 {
				t.Fatal(calls.Load())
			}
		})
	}
}

func TestVolumeSnapshotReadMaxItemsCountsExcludedRowsAndStopsBeforeUnusedTail(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls atomic.Int32
	body := snapshotReadContractPage(`[{"id":"excluded","description":"other"},false]`, "https://unused.invalid/")
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("limit") != "1" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, body)
	})
	result, err := blockstorage.ListVolumeSnapshots(context.Background(), client, blockstorage.WithVolumeSnapshotListMaxItems(1), blockstorage.WithVolumeSnapshotListFilters(json.RawMessage(`{"description":"wanted"}`)))
	if err != nil || result == nil || string(result.Value) != "[]" || result.Snapshots == nil || len(result.Snapshots) != 0 || len(result.Pages) != 1 || calls.Load() != 1 {
		t.Fatal(result, err, calls.Load())
	}
}

func TestVolumeSnapshotReadMaxItemsAtPageBoundaryStillFetchesAndParsesNextPage(t *testing.T) {
	for _, nextBody := range []string{`{"snapshots":[false]}`, `{"snapshots":`} {
		t.Run(nextBody, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				snapshotReadContractWire(t, r, snapshotReadContractDetail, "test-token")
				if calls.Add(1) == 1 {
					testcloud.JSON(w, 200, snapshotReadContractPage(`[{"id":"first"}]`, "?marker=second"))
					return
				}
				if calls.Load() != 2 || r.URL.Query().Get("marker") != "second" {
					t.Error("extra boundary request changed", r.URL, calls.Load())
				}
				testcloud.JSON(w, 200, nextBody)
			})
			result, err := blockstorage.ListVolumeSnapshots(context.Background(), client, blockstorage.WithVolumeSnapshotListMaxItems(1))
			if nextBody == `{"snapshots":[false]}` {
				if err != nil || result == nil || len(result.Snapshots) != 1 || len(result.Pages) != 2 {
					t.Fatal(result, err)
				}
			} else {
				var physical *resource.ResponseError
				if result == nil || result.Value != nil || result.Snapshots != nil || len(result.Pages) != 2 || !errors.As(err, &physical) || string(physical.Body) != nextBody {
					t.Fatal(result, err)
				}
			}
			if calls.Load() != 2 {
				t.Fatal(calls.Load())
			}
		})
	}
}

func TestVolumeSnapshotReadNegativeMaximumParsesResponseWithoutConsumingRows(t *testing.T) {
	for _, body := range []string{`{"snapshots":[{"force":"bad"}],"links":[null]}`, `{"snapshots":`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("limit") != "-1" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, body)
			})
			result, err := blockstorage.ListVolumeSnapshots(context.Background(), client, blockstorage.WithVolumeSnapshotListMaxItems(-1))
			if json.Valid([]byte(body)) {
				if err != nil || result == nil || string(result.Value) != "[]" || result.Snapshots == nil || len(result.Snapshots) != 0 || len(result.Pages) != 1 {
					t.Fatal(result, err)
				}
			} else {
				var physical *resource.ResponseError
				if result == nil || result.Value != nil || len(result.Pages) != 1 || !errors.As(err, &physical) {
					t.Fatal(result, err)
				}
			}
			if calls.Load() != 1 {
				t.Fatal(calls.Load())
			}
		})
	}
}

func TestVolumeSnapshotReadZeroMaximumIsUnlimitedAndPaginationFalseStopsFirstPage(t *testing.T) {
	for _, paginated := range []bool{true, false} {
		t.Run(map[bool]string{true: "zero unlimited", false: "first page"}[paginated], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					if r.URL.Query().Has("limit") {
						t.Error("zero invented limit", r.URL)
					}
					testcloud.JSON(w, 200, snapshotReadContractPage(`[{"id":"first"},{"id":"second"}]`, "?marker=next"))
					return
				}
				if calls.Load() != 2 {
					t.Error("unbounded fixture", calls.Load())
				}
				testcloud.JSON(w, 200, snapshotReadContractPage(`[{"id":"third"}]`, ""))
			})
			result, err := blockstorage.ListVolumeSnapshots(context.Background(), client, blockstorage.WithVolumeSnapshotListMaxItems(0), blockstorage.WithVolumeSnapshotListPagination(paginated))
			want := 2
			pages := 1
			if paginated {
				want = 3
				pages = 2
			}
			if err != nil || result == nil || len(result.Snapshots) != want || len(result.Pages) != pages || calls.Load() != int32(pages) {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestVolumeSnapshotReadListExpressionRepairsGeneratorInputAfterCompleteMaterialization(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			testcloud.JSON(w, 200, snapshotReadContractPage(`[{"id":"first"}]`, "?marker=next"))
			return
		}
		if n != 2 {
			t.Error("list expression caused extra HTTP", r.URL)
		}
		testcloud.JSON(w, 200, snapshotReadContractPage(`[{"id":"second"}]`, ""))
	})
	result, err := blockstorage.ListVolumeSnapshots(context.Background(), client, blockstorage.WithVolumeSnapshotListExpression("[].id"))
	if err != nil || result == nil || string(result.Value) != `["first","second"]` || result.Snapshots != nil || len(result.Pages) != 2 || calls.Load() != 2 {
		t.Fatal(result, err, calls.Load())
	}
}

func TestVolumeSnapshotReadListSelectedAliasConvertsOnlyFinalParsedDictionaryValue(t *testing.T) {
	for _, tc := range []struct {
		row  string
		fail bool
	}{{`{"id":"good","is_forced":"bad discarded","force":"TRUE"}`, false}, {`{"id":"bad","force":"TRUE","is_forced":"bad selected"}`, true}} {
		t.Run(tc.row, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			body := snapshotReadContractPage(`[`+tc.row+`]`, "")
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, body) })
			result, err := blockstorage.ListVolumeSnapshots(context.Background(), client)
			if tc.fail {
				var physical *resource.ResponseError
				if result == nil || result.Value != nil || result.Snapshots != nil || len(result.Pages) != 1 || !errors.As(err, &physical) || string(physical.Body) != body {
					t.Fatal(result, err)
				}
			} else {
				if err != nil || result == nil || len(result.Snapshots) != 1 || string(snapshotReadContractRows(t, result.Value)[0]["is_forced"]) != "true" || string(result.Snapshots[0].Body["is_forced"]) != `"bad discarded"` {
					t.Fatal(result, err)
				}
			}
			if calls.Load() != 1 {
				t.Fatal(calls.Load())
			}
		})
	}
}

func TestVolumeSnapshotReadTypedEmptyListExpressionEvaluatesAfterCompleteList(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls atomic.Int32
	body := snapshotReadContractPage(`[{"id":"wire"}]`, "")
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("X-Proof", "list before expression")
		testcloud.JSON(w, 203, body)
	})
	result, err := blockstorage.ListVolumeSnapshots(context.Background(), client, blockstorage.WithVolumeSnapshotListExpression(""))
	var physical *resource.ResponseError
	if result == nil || result.Value != nil || result.Snapshots != nil || len(result.Pages) != 1 || string(result.Pages[0].Body) != body || calls.Load() != 1 || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &physical) {
		t.Fatal(result, err, calls.Load())
	}
}
