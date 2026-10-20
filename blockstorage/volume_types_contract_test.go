package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const volumeTypeContractPath = volumeReadContractBase + "types"
const volumeTypeContractMember = volumeTypeContractPath + "/target"

func volumeTypeContractPage(rows, next string) string {
	tail := ""
	if next != "" {
		encoded, _ := json.Marshal(next)
		tail = `,"volume_types_links":[{"rel":"next","href":` + string(encoded) + `}]`
	}
	return `{"volume_types":` + rows + tail + `}`
}

func volumeTypeContractOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var outer *resource.OperationError
	if !errors.As(err, &outer) || outer.Operation != operation || outer.Resource != "volume type" || outer.Cause == nil {
		t.Fatal("public type operation context lost", err)
	}
}

type volumeTypeContractOutcome struct {
	nilResult bool
	value     json.RawMessage
	types     []*resource.RawResource
	typ       *resource.RawResource
	observed  *blockstorage.VolumeTypesPage
	pages     []*blockstorage.VolumeTypesPage
	err       error
}

func volumeTypeContractCall(ctx context.Context, client *gophercloud.ServiceClient, operation string, read []blockstorage.VolumeTypeReadOption, search ...blockstorage.VolumeTypeSearchOption) volumeTypeContractOutcome {
	switch operation {
	case "ListVolumeTypes":
		result, err := blockstorage.ListVolumeTypes(ctx, client, read...)
		if result == nil {
			return volumeTypeContractOutcome{nilResult: true, err: err}
		}
		return volumeTypeContractOutcome{value: result.Value, types: result.Types, pages: result.Pages, err: err}
	case "SearchVolumeTypes":
		result, err := blockstorage.SearchVolumeTypes(ctx, client, blockstorage.SearchVolumeTypesRequest{}, search...)
		if result == nil {
			return volumeTypeContractOutcome{nilResult: true, err: err}
		}
		return volumeTypeContractOutcome{value: result.Value, types: result.Types, pages: result.Pages, err: err}
	case "GetVolumeType":
		result, err := blockstorage.GetVolumeType(ctx, client, blockstorage.GetVolumeTypeRequest{NameOrID: "target"}, search...)
		if result == nil {
			return volumeTypeContractOutcome{nilResult: true, err: err}
		}
		return volumeTypeContractOutcome{value: result.Value, typ: result.Type, observed: result.Observed, pages: result.Pages, err: err}
	default:
		panic("unknown public volume type operation")
	}
}

func TestListVolumeTypesCompletesQuerylessPagesAndOwnsSixFieldViewsAndWireProof(t *testing.T) {
	cloud := testcloud.New(t)
	client := volumeReadContractClient(cloud)
	var calls atomic.Int32
	first := volumeTypeContractPage(`[{"id":"duplicate","name":"first","description":[false],"extra_specs":{"precision":9007199254740993},"os-volume-type-access:is_public":"false","metadata":{"wire-only":true},"project_id":"foreign","availability_zone":"wire-zone"}]`, "?marker=next&limit=1")
	second := volumeTypeContractPage(`[{"id":"duplicate","name":"second","extra_specs":[["ignored",1]],"is_public":false},{"location":{"opaque":[null,9007199254740993]},"unknown":true}]`, "")
	zone := json.RawMessage(`"owned-zone"`)
	name := "owned-project"
	location := resource.CloudLocation{Zone: zone, Project: resource.CloudProject{ID: json.RawMessage(`"owned"`), Name: &name}}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		volumeReadContractWire(t, r, volumeTypeContractPath, "test-token")
		switch calls.Add(1) {
		case 1:
			if r.URL.RawQuery != "" {
				t.Error("list invented visibility/name/detail query", r.URL)
			}
			w.Header().Set("X-Proof", "first")
			testcloud.JSON(w, 200, first)
		case 2:
			if r.URL.Query().Get("marker") != "next" || r.URL.Query().Get("limit") != "1" || len(r.URL.Query()) != 2 {
				t.Error(r.URL)
			}
			w.Header().Set("X-Proof", "second")
			testcloud.JSON(w, 200, second)
		default:
			t.Error("enrichment/member/list restart", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.ListVolumeTypes(context.Background(), client, blockstorage.WithVolumeTypeReadLocation(location))
	if err != nil || result == nil || calls.Load() != 2 || len(result.Pages) != 2 || len(result.Types) != 3 || result.Types[0] == result.Types[1] {
		t.Fatal(result, err, calls.Load())
	}
	rows := volumeReadContractRows(t, result.Value)
	if len(rows) != 3 || len(rows[0]) != 6 || string(rows[0]["is_public"]) != "true" || string(rows[1]["extra_specs"]) != "{}" || string(rows[2]["location"]) != `{"opaque":[null,9007199254740993]}` {
		t.Fatal(string(result.Value))
	}
	computed := volumeReadContractFields(t, rows[0]["location"])
	project := volumeReadContractFields(t, computed["project"])
	if string(computed["zone"]) != `"owned-zone"` || string(project["id"]) != `"owned"` || string(project["name"]) != `"owned-project"` {
		t.Fatal("unknown wire owner/zone adopted", string(rows[0]["location"]))
	}
	if _, ok := rows[0]["metadata"]; ok {
		t.Fatal("Type acquired Volume MetadataMixin")
	}
	if string(result.Types[0].Body["metadata"]) != `{"wire-only":true}` || result.Types[0].Header.Get("X-Proof") != "first" || result.Types[2].StatusCode != 200 {
		t.Fatal(result)
	}
	result.Pages[0].Body[0] = '!'
	result.Pages[1].Header.Set("X-Proof", "caller-page")
	result.Types[0].Body["id"][0] = '!'
	result.Types[2].Body["location"][0] = '!'
	result.Types[1].Header.Set("X-Proof", "caller-row")
	if !json.Valid(result.Value) || result.Types[0].Header.Get("X-Proof") != "first" || result.Types[2].Header.Get("X-Proof") != "second" || string(result.Pages[1].Body) != second {
		t.Fatal("views/rows/pages alias", result)
	}
}

func TestVolumeTypesEmptyCanonicalListIgnoresUnusedLinksAndLocation(t *testing.T) {
	for _, operation := range []string{"ListVolumeTypes", "SearchVolumeTypes"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls atomic.Int32
			body := `{"volume_types":[],"links":false,"volume_types_links":{"invalid":true},"next":17}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				volumeReadContractWire(t, r, volumeTypeContractPath, "test-token")
				calls.Add(1)
				testcloud.JSON(w, 200, body)
			})
			location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`invalid-unused`)}}
			got := volumeTypeContractCall(context.Background(), client, operation, []blockstorage.VolumeTypeReadOption{blockstorage.WithVolumeTypeReadLocation(location)}, blockstorage.WithVolumeTypeSearchLocation(location))
			if got.err != nil || got.nilResult || string(got.value) != "[]" || got.types == nil || len(got.types) != 0 || calls.Load() != 1 || len(got.pages) != 1 || string(got.pages[0].Body) != body {
				t.Fatal(got, calls.Load())
			}
		})
	}
}

func TestGetVolumeTypeOmittedAndNullFiltersKeepMemberQueryAndFreshCanonicalIdentity(t *testing.T) {
	for _, null := range []bool{false, true} {
		for _, row := range []string{`{}`, `{"id":false,"name":[1],"location":false,"project_id":"foreign","availability_zone":"wire"}`, `{"id":"different","unknown":9007199254740993}`} {
			t.Run(row+map[bool]string{false: " omitted", true: " null"}[null], func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				var calls atomic.Int32
				body := `{"volume_type":` + row + `}`
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					volumeReadContractWire(t, r, volumeTypeContractMember, "test-token")
					if r.URL.RawQuery != "is_public=none" {
						t.Error("mandatory literal visibility changed", r.URL)
					}
					if calls.Add(1) != 1 {
						t.Error("successful member fell back", r.URL)
					}
					w.Header().Set("X-Proof", "member")
					testcloud.JSON(w, 200, body)
				})
				location := resource.CloudLocation{Zone: json.RawMessage(`"owned"`)}
				options := []blockstorage.VolumeTypeSearchOption{blockstorage.WithVolumeTypeSearchLocation(location)}
				if null {
					options = append(options, blockstorage.WithVolumeTypeSearchFilters(json.RawMessage(" null ")))
				}
				result, err := blockstorage.GetVolumeType(context.Background(), client, blockstorage.GetVolumeTypeRequest{NameOrID: "target"}, options...)
				if err != nil || result == nil || result.Type == nil || result.Observed == nil || len(result.Pages) != 0 || calls.Load() != 1 || string(result.Observed.Body) != body {
					t.Fatal(result, err, calls.Load())
				}
				fields := volumeReadContractFields(t, result.Value)
				owned := volumeReadContractFields(t, fields["location"])
				if len(fields) != 6 || string(owned["zone"]) != `"owned"` {
					t.Fatal("member wire location/zone used", string(result.Value))
				}
				if row == `{}` {
					if string(fields["id"]) != "null" || len(result.Type.Body) != 0 {
						t.Fatal("request id fabricated", result)
					}
				} else if row == `{"id":false,"name":[1],"location":false,"project_id":"foreign","availability_zone":"wire"}` {
					if string(fields["id"]) != "false" || string(fields["name"]) != "[1]" {
						t.Fatal(string(result.Value))
					}
				} else if string(fields["id"]) != `"different"` {
					t.Fatal(string(result.Value))
				}
				result.Observed.Body[0] = '!'
				result.Observed.Header.Set("X-Proof", "caller")
				result.Type.Body["name"] = json.RawMessage(`"caller"`)
				if !json.Valid(result.Value) || result.Type.Header.Get("X-Proof") != "member" {
					t.Fatal("member proof aliases raw/view", result)
				}
			})
		}
	}
}

func TestGetVolumeTypeDefaultFallbackUsesNoneWithoutNameHintAndWaitsForExactAbsence(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		for _, found := range []bool{false, true} {
			t.Run(http.StatusText(code)+map[bool]string{false: " absent", true: " found"}[found], func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					n := calls.Add(1)
					path := volumeTypeContractPath
					if n == 1 {
						path = volumeTypeContractMember
					}
					volumeReadContractWire(t, r, path, "test-token")
					if r.URL.Query().Get("is_public") != "none" || r.URL.Query().Has("name") {
						t.Error("find visibility/name policy", r.URL)
					}
					switch n {
					case 1:
						testcloud.JSON(w, code, `{"error":"missing member"}`)
					case 2:
						if len(r.URL.Query()) != 1 {
							t.Error(r.URL)
						}
						testcloud.JSON(w, 200, volumeTypeContractPage(`[{"id":"other","name":"Target"}]`, "?marker=last&limit=2"))
					case 3:
						if r.URL.Query().Get("marker") != "last" || r.URL.Query().Get("limit") != "2" {
							t.Error(r.URL)
						}
						rows := `[]`
						if found {
							rows = `[{"id":"actual","name":"target"}]`
						}
						testcloud.JSON(w, 200, volumeTypeContractPage(rows, ""))
					default:
						t.Error("extra fallback/refresh", r.URL)
						w.WriteHeader(500)
					}
				})
				result, err := blockstorage.GetVolumeType(context.Background(), client, blockstorage.GetVolumeTypeRequest{NameOrID: "target"})
				if err != nil || result == nil || calls.Load() != 3 || result.Observed != nil || len(result.Pages) != 2 {
					t.Fatal(result, err, calls.Load())
				}
				if found {
					if result.Type == nil || string(volumeReadContractFields(t, result.Value)["id"]) != `"actual"` {
						t.Fatal(result)
					}
				} else if result.Value != nil || result.Type != nil {
					t.Fatal("failed member treated as found", result)
				}
			})
		}
	}
}

func TestVolumeTypeExactFindStopsAtSecondMatchWhileSearchConsumesUnusedBadRowAndLink(t *testing.T) {
	for _, badRow := range []bool{false, true} {
		for _, fullSearch := range []bool{false, true} {
			t.Run(map[bool]string{false: "bad link", true: "bad row"}[badRow]+map[bool]string{false: " exact find", true: " full search"}[fullSearch], func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				var calls atomic.Int32
				identity := "literal / name*"
				tail := ""
				if badRow {
					tail = ",false"
				}
				body := `{"volume_types":[{"id":"one","name":"literal / name*"},{"id":"two","name":"literal / name*"}` + tail + `],"links":false}`
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					volumeReadContractWire(t, r, volumeTypeContractPath, "test-token")
					query := "is_public=none"
					if fullSearch {
						query = ""
					}
					if r.URL.RawQuery != query || calls.Add(1) != 1 {
						t.Error("unsafe name used member/hint/refresh", r.URL)
					}
					w.Header().Set("X-Proof", "current-origin")
					testcloud.JSON(w, 200, body)
				})
				var value json.RawMessage
				var pages []*blockstorage.VolumeTypesPage
				var err error
				if fullSearch {
					result, cause := blockstorage.SearchVolumeTypes(context.Background(), client, blockstorage.SearchVolumeTypesRequest{NameOrID: identity})
					if result == nil || result.Types != nil {
						t.Fatal(result, cause)
					}
					value, pages, err = result.Value, result.Pages, cause
				} else {
					result, cause := blockstorage.GetVolumeType(context.Background(), client, blockstorage.GetVolumeTypeRequest{NameOrID: identity})
					if result == nil || result.Type != nil || result.Observed != nil {
						t.Fatal(result, cause)
					}
					value, pages, err = result.Value, result.Pages, cause
				}
				if value != nil || len(pages) != 1 || calls.Load() != 1 || string(pages[0].Body) != body {
					t.Fatal(string(value), pages, err, calls.Load())
				}
				var accepted *resource.ResponseError
				if fullSearch {
					if !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != body {
						t.Fatal(err)
					}
					volumeTypeContractOperation(t, err, "SearchVolumeTypes")
				} else {
					if !errors.Is(err, resource.ErrAmbiguous) || errors.As(err, &accepted) {
						t.Fatal(err)
					}
					volumeTypeContractOperation(t, err, "GetVolumeType")
				}
			})
		}
	}
}

func TestSearchVolumeTypesCompletesAllPagesAndPreservesExactGlobDuplicateSelection(t *testing.T) {
	cloud := testcloud.New(t)
	client := volumeReadContractClient(cloud)
	var calls atomic.Int32
	first := volumeTypeContractPage(`[{"id":"fast*","name":"literal"},{"id":"same","name":"fast/a\n한글","extra_specs":{"nested":{"mode":true}}}]`, "?marker=next")
	second := volumeTypeContractPage(`[{"id":"same","name":"fast/a\n한글","extra_specs":{"nested":{"mode":1}}},{"id":"skip","name":"Fast/a\n한글"}]`, "")
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		volumeReadContractWire(t, r, volumeTypeContractPath, "test-token")
		switch calls.Add(1) {
		case 1:
			if r.URL.RawQuery != "" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, first)
		case 2:
			if r.URL.RawQuery != "marker=next" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, second)
		default:
			t.Error(r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.SearchVolumeTypes(context.Background(), client, blockstorage.SearchVolumeTypesRequest{NameOrID: "fast*"})
	if err != nil || result == nil || calls.Load() != 2 || len(result.Pages) != 2 || len(result.Types) != 3 || result.Types[1] == result.Types[2] {
		t.Fatal(result, err, calls.Load())
	}
	rows := volumeReadContractRows(t, result.Value)
	if string(rows[0]["id"]) != `"fast*"` || string(rows[1]["id"]) != `"same"` || string(rows[2]["id"]) != `"same"` {
		t.Fatal(string(result.Value))
	}
}

func TestGetVolumeTypeEveryNonnullFalseyFilterUsesFullQuerylessSearch(t *testing.T) {
	for _, filter := range []string{`{}`, `false`, `0`, `""`, `[]`} {
		t.Run(filter, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				volumeReadContractWire(t, r, volumeTypeContractPath, "test-token")
				switch calls.Add(1) {
				case 1:
					if r.URL.RawQuery != "" {
						t.Error("filtered getter leaked none/name", r.URL)
					}
					testcloud.JSON(w, 200, volumeTypeContractPage(`[{"id":"target"}]`, "?marker=second"))
				case 2:
					if r.URL.RawQuery != "marker=second" {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, volumeTypeContractPage(`[{"id":"unused"}]`, ""))
				default:
					t.Error(r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := blockstorage.GetVolumeType(context.Background(), client, blockstorage.GetVolumeTypeRequest{NameOrID: "target"}, blockstorage.WithVolumeTypeSearchFilters(json.RawMessage(filter)))
			if err != nil || result == nil || result.Type == nil || result.Observed != nil || len(result.Pages) != 2 || calls.Load() != 2 || string(result.Type.Body["id"]) != `"target"` {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestVolumeTypeExpressionsKeepDynamicSearchAndGetterFirstValueSemantics(t *testing.T) {
	for _, tc := range []struct {
		expression, search, get string
		ambiguous, invalid      bool
	}{
		{"`[false]`", "[false]", "false", false, false}, {"`[0]`", "[0]", "0", false, false}, {"`[\"\"]`", `[""]`, `""`, false, false}, {"`[null]`", "[null]", "", false, false}, {"`false`", "false", "", false, false}, {"`{}`", "{}", "", false, false}, {"`\"한\"`", `"한"`, `"한"`, false, false}, {"`[1,2]`", "[1,2]", "", true, false}, {"`true`", "true", "", false, true}, {"`{\"one\":1}`", `{"one":1}`, "", false, true},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				volumeReadContractWire(t, r, volumeTypeContractPath, "test-token")
				if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				calls.Add(1)
				testcloud.JSON(w, 200, volumeTypeContractPage(`[{"id":"target"}]`, ""))
			})
			search, err := blockstorage.SearchVolumeTypes(context.Background(), client, blockstorage.SearchVolumeTypesRequest{}, blockstorage.WithVolumeTypeSearchExpression(tc.expression))
			if err != nil || search == nil || string(search.Value) != tc.search || search.Types != nil || len(search.Pages) != 1 {
				t.Fatal(search, err)
			}
			get, err := blockstorage.GetVolumeType(context.Background(), client, blockstorage.GetVolumeTypeRequest{}, blockstorage.WithVolumeTypeSearchExpression(tc.expression))
			if get == nil || string(get.Value) != tc.get || get.Type != nil || get.Observed != nil || len(get.Pages) != 1 || calls.Load() != 2 {
				t.Fatal(get, err, calls.Load())
			}
			if tc.ambiguous {
				var multiple *blockstorage.VolumeTypeSelectionError
				if !errors.Is(err, resource.ErrAmbiguous) || !errors.As(err, &multiple) || multiple.Length != 2 {
					t.Fatal(err)
				}
			} else if tc.invalid {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestVolumeTypeLocalFiltersFollowCompleteEagerNormalizationAndKeepOnlyOriginPages(t *testing.T) {
	for _, kind := range []string{"bad row", "bad filter", "later rejection"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls atomic.Int32
			rows := `[{"id":"unused"}]`
			if kind == "bad row" {
				rows = `[{"id":"unused"},false]`
			}
			first := volumeTypeContractPage(rows, "?marker=later")
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				volumeReadContractWire(t, r, volumeTypeContractPath, "test-token")
				n := calls.Add(1)
				w.Header().Set("X-Proof", "origin")
				if n == 1 {
					testcloud.JSON(w, 200, first)
				} else if n == 2 {
					if kind == "later rejection" {
						testcloud.JSON(w, 403, `{"error":"later"}`)
					} else {
						testcloud.JSON(w, 200, volumeTypeContractPage(`[{"id":"target"}]`, ""))
					}
				} else {
					t.Error(r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := blockstorage.SearchVolumeTypes(context.Background(), client, blockstorage.SearchVolumeTypesRequest{NameOrID: "target"}, blockstorage.WithVolumeTypeSearchFilters(json.RawMessage(`not JSON`)))
			var physical *resource.ResponseError
			var native gophercloud.ErrUnexpectedResponseCode
			if result == nil || result.Value != nil || result.Types != nil || err == nil {
				t.Fatal(result, err)
			}
			switch kind {
			case "bad row":
				if calls.Load() != 1 || len(result.Pages) != 1 || !errors.As(err, &physical) || string(physical.Body) != first {
					t.Fatal(result, err, calls.Load())
				}
			case "bad filter":
				if calls.Load() != 2 || len(result.Pages) != 2 || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &physical) {
					t.Fatal(result, err)
				}
			case "later rejection":
				if calls.Load() != 2 || len(result.Pages) != 1 || !errors.As(err, &native) || native.Actual != 403 || errors.As(err, &physical) {
					t.Fatal(result, err)
				}
			}
			volumeTypeContractOperation(t, err, "SearchVolumeTypes")
		})
	}
}

func TestVolumeTypeSearchMappingUsesNormalizedSubsetsAndOrderedLazyErrors(t *testing.T) {
	for _, tc := range []struct {
		name, rows, identity, filter string
		matches                      int
		invalid                      bool
	}{
		{"empty identifier selection skips truthy shape", `[{"id":"other"}]`, "target", `true`, 0, false},
		{"selected truthy shape fails locally", `[{"id":"target"}]`, "target", `true`, 0, true},
		{"null actual skips nested missing field", `[{"id":"target","extra_specs":null}]`, "target", `{"extra_specs":{"missing":1}}`, 0, false},
		{"nested bool numeric equality", `[{"id":"target","extra_specs":{"nested":{"value":1,"unreferenced":false}}}]`, "target", `{"extra_specs":{"nested":{"value":true}}}`, 1, false},
		{"mismatch leaves unknown metadata unvisited", `[{"id":"target"}]`, "target", `{"name":"mismatch","metadata":{"missing":1}}`, 0, false},
		{"visited unknown metadata is not Volume mixin", `[{"id":"target"}]`, "target", `{"metadata":{"missing":1},"name":"mismatch"}`, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				volumeReadContractWire(t, r, volumeTypeContractPath, "test-token")
				calls.Add(1)
				w.Header().Set("X-Proof", "mapping-origin")
				testcloud.JSON(w, 200, volumeTypeContractPage(tc.rows, ""))
			})
			result, err := blockstorage.SearchVolumeTypes(context.Background(), client, blockstorage.SearchVolumeTypesRequest{NameOrID: tc.identity}, blockstorage.WithVolumeTypeSearchFilters(json.RawMessage(tc.filter)))
			var physical *resource.ResponseError
			if result == nil || len(result.Pages) != 1 || calls.Load() != 1 || errors.As(err, &physical) {
				t.Fatal("local selection borrowed physical error proof", result, err, calls.Load())
			}
			if tc.invalid {
				if !errors.Is(err, resource.ErrInvalidOption) || result.Value != nil || result.Types != nil {
					t.Fatal(result, err)
				}
			} else if err != nil || result.Value == nil || result.Types == nil || len(result.Types) != tc.matches {
				t.Fatal(result, err)
			}
		})
	}
}
