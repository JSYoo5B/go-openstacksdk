package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

const volumeSearchTestBase = "/search/cinder/v3/p/"
const volumeSearchTestPath = volumeSearchTestBase + "volumes/detail"

func volumeSearchTestClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	return cloud.Client("volumev3", volumeSearchTestBase)
}
func volumeSearchTestPage(rows, next string) string {
	tail := ""
	if next != "" {
		raw, _ := json.Marshal(next)
		tail = `,"volumes_links":[{"rel":"next","href":` + string(raw) + `}]`
	}
	return `{"volumes":` + rows + tail + `}`
}
func volumeSearchTestFields(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(string(raw), err)
	}
	return fields
}
func volumeSearchTestRows(t *testing.T, raw json.RawMessage) []map[string]json.RawMessage {
	t.Helper()
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(string(raw), err)
	}
	return rows
}
func volumeSearchTestOperation(t *testing.T, err error, want string) {
	t.Helper()
	var operation *resource.OperationError
	if !errors.As(err, &operation) || operation.Operation != want || operation.Resource != "volume" {
		t.Fatal("operation context missing", err)
	}
}

func TestSearchVolumesCompletePagesExactOrGlobPreservesDuplicatesAndWireOwnership(t *testing.T) {
	cloud := testcloud.New(t)
	client := volumeSearchTestClient(cloud)
	var calls atomic.Int32
	first := volumeSearchTestPage(`[{"id":"disk*","name":"literal","unknown":{"keep":9007199254740993}},{"id":"duplicate","name":"disk-a","row":1}]`, `?marker=next&limit=2`)
	second := volumeSearchTestPage(`[{"id":"duplicate","name":"disk-a","row":2},{"id":"skip","name":"Disk-a"}]`, "")
	cloud.Mux.HandleFunc("GET "+volumeSearchTestPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && r.ContentLength > 0 {
			t.Error("request body sent")
		}
		w.Header().Set("X-Proof", r.URL.RawQuery)
		switch calls.Add(1) {
		case 1:
			if r.URL.RawQuery != "" {
				t.Error("local identity leaked to server query", r.URL)
			}
			testcloud.JSON(w, 200, first)
		case 2:
			if r.URL.RawQuery != "marker=next&limit=2" {
				t.Error("advertised query reordered", r.URL)
			}
			testcloud.JSON(w, 200, second)
		default:
			t.Error("unexpected list restart/refresh", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.SearchVolumes(context.Background(), client, blockstorage.SearchVolumesRequest{NameOrID: "disk*"})
	if err != nil || result == nil || calls.Load() != 2 || len(result.Pages) != 2 || len(result.Volumes) != 3 {
		t.Fatal(result, err, calls.Load())
	}
	rows := volumeSearchTestRows(t, result.Value)
	if len(rows) != 3 || string(rows[0]["id"]) != `"disk*"` || string(rows[1]["id"]) != `"duplicate"` || string(rows[2]["id"]) != `"duplicate"` {
		t.Fatal(string(result.Value))
	}
	if len(rows[0]) != 38 || string(rows[0]["status"]) != "null" {
		t.Fatal("normalized source view incomplete", string(result.Value))
	}
	if _, ok := rows[0]["unknown"]; ok {
		t.Fatal("wire-only member entered normalized view")
	}
	if string(result.Volumes[0].Body["unknown"]) != `{"keep":9007199254740993}` || result.Volumes[1] == result.Volumes[2] || result.Volumes[0].StatusCode != 200 {
		t.Fatal("wire/row evidence lost", result)
	}
	result.Pages[0].Body[0] = '!'
	result.Volumes[0].Body["id"] = json.RawMessage(`"caller"`)
	result.Volumes[2].Header.Set("X-Proof", "caller")
	if !json.Valid(result.Value) || string(rows[0]["id"]) != `"disk*"` || string(result.Pages[1].Body) != second || result.Volumes[1].Header.Get("X-Proof") == "caller" {
		t.Fatal("result components alias", result)
	}
}

func TestSearchVolumesNormalizesEveryRowBeforeNextAndConsumesFiltersAfterFullList(t *testing.T) {
	for _, eager := range []bool{true, false} {
		t.Run(map[bool]string{true: "malformed response", false: "malformed caller filter"}[eager], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeSearchTestClient(cloud)
			var calls atomic.Int32
			firstRows := `[{"id":"not-selected","bootable":false}]`
			if eager {
				firstRows = `[{"id":"not-selected","bootable":"bad"}]`
			}
			cloud.Mux.HandleFunc("GET "+volumeSearchTestPath, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Proof", "page")
				switch calls.Add(1) {
				case 1:
					testcloud.JSON(w, 200, volumeSearchTestPage(firstRows, `?marker=2`))
				case 2:
					testcloud.JSON(w, 200, volumeSearchTestPage(`[{"id":"selected"}]`, ""))
				default:
					t.Error(r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := blockstorage.SearchVolumes(context.Background(), client, blockstorage.SearchVolumesRequest{NameOrID: "selected"}, blockstorage.WithVolumeSearchFilters(json.RawMessage(`not JSON`)))
			want := int32(2)
			if eager {
				want = 1
			}
			if err == nil || result == nil || calls.Load() != want || len(result.Pages) != int(want) || result.Value != nil || result.Volumes != nil {
				t.Fatal(result, err, calls.Load())
			}
			volumeSearchTestOperation(t, err, "SearchVolumes")
			var response *resource.ResponseError
			if eager {
				if !errors.As(err, &response) || response.StatusCode != 200 || response.Header.Get("X-Proof") != "page" || !strings.Contains(err.Error(), "is_bootable") {
					t.Fatal("eager error lost originating response", err)
				}
			} else if errors.As(err, &response) || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("local filter borrowed physical response", err)
			}
		})
	}
}

func TestGetVolumeOmittedNullGETFirstAndCompatibleListFallback(t *testing.T) {
	for _, explicitNull := range []bool{false, true} {
		t.Run(map[bool]string{false: "omitted", true: "null"}[explicitNull], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeSearchTestClient(cloud)
			var members, lists atomic.Int32
			cloud.Mux.HandleFunc("GET "+volumeSearchTestBase+"volumes/id", func(w http.ResponseWriter, r *http.Request) {
				members.Add(1)
				w.Header().Set("X-Proof", "member")
				testcloud.JSON(w, 200, `{"volume":{"id":"id","name":"found","unknown":[1]}}`)
			})
			cloud.Mux.HandleFunc("GET "+volumeSearchTestPath, func(w http.ResponseWriter, r *http.Request) { lists.Add(1); w.WriteHeader(500) })
			var options []blockstorage.VolumeSearchOption
			if explicitNull {
				options = []blockstorage.VolumeSearchOption{blockstorage.WithVolumeSearchFilters(json.RawMessage(" null "))}
			}
			result, err := blockstorage.GetVolume(context.Background(), client, blockstorage.GetVolumeRequest{NameOrID: "id"}, options...)
			if err != nil || result == nil || members.Load() != 1 || lists.Load() != 0 || result.Observed == nil || result.Observed.StatusCode != 200 || len(result.Pages) != 0 || result.Volume == nil || string(result.Volume.Body["unknown"]) != "[1]" || string(volumeSearchTestFields(t, result.Value)["name"]) != `"found"` {
				t.Fatal(result, err, members.Load(), lists.Load())
			}
		})
	}
	for _, code := range []int{400, 403, 404} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeSearchTestClient(cloud)
			var members, lists atomic.Int32
			cloud.Mux.HandleFunc("GET "+volumeSearchTestBase+"volumes/human", func(w http.ResponseWriter, r *http.Request) {
				members.Add(1)
				testcloud.JSON(w, code, `{"member":"rejected"}`)
			})
			cloud.Mux.HandleFunc("GET "+volumeSearchTestPath, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("name") != "human" {
					t.Error("shared exact-name hint missing", r.URL)
				}
				testcloud.JSON(w, 200, `{"volumes":[{"id":"resolved","name":"human"}]}`)
			})
			result, err := blockstorage.GetVolume(context.Background(), client, blockstorage.GetVolumeRequest{NameOrID: "human"})
			if err != nil || result == nil || result.Volume == nil || members.Load() != 1 || lists.Load() != 1 || len(result.Pages) != 1 || result.Observed != nil || string(result.Volume.Body["id"]) != `"resolved"` {
				t.Fatal(result, err, members.Load(), lists.Load())
			}
		})
	}
}

func TestGetVolumeEveryFalseyNonnullFilterUsesFullListOnly(t *testing.T) {
	for _, filter := range []string{"false", "0", "-0.0", `""`, "[]", "{}"} {
		t.Run(filter, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeSearchTestClient(cloud)
			var lists, members atomic.Int32
			cloud.Mux.HandleFunc("GET "+volumeSearchTestPath, func(w http.ResponseWriter, r *http.Request) {
				n := lists.Add(1)
				if n == 1 {
					testcloud.JSON(w, 200, volumeSearchTestPage(`[{"id":"wanted"}]`, `?marker=2`))
				} else {
					testcloud.JSON(w, 200, `{"volumes":[{"id":"other"}]}`)
				}
			})
			cloud.Mux.HandleFunc("GET "+volumeSearchTestBase+"volumes/wanted", func(w http.ResponseWriter, r *http.Request) { members.Add(1); w.WriteHeader(500) })
			result, err := blockstorage.GetVolume(context.Background(), client, blockstorage.GetVolumeRequest{NameOrID: "wanted"}, blockstorage.WithVolumeSearchFilters(json.RawMessage(filter)))
			if err != nil || result == nil || result.Volume == nil || lists.Load() != 2 || members.Load() != 0 || result.Observed != nil || len(result.Pages) != 2 || string(result.Volume.Body["id"]) != `"wanted"` {
				t.Fatal(result, err, lists.Load(), members.Load())
			}
		})
	}
}

func TestSearchAndGetVolumeExpressionsKeepDynamicValuesAndSingleFalseyElement(t *testing.T) {
	for _, tc := range []struct {
		expression, want   string
		ambiguous, invalid bool
	}{
		{"`[false]`", "false", false, false}, {"`[0]`", "0", false, false}, {"`[\"\"]`", `""`, false, false}, {"`[null]`", "", false, false}, {"`false`", "", false, false}, {"`0`", "", false, false}, {"`\"\"`", "", false, false}, {"`{}`", "", false, false}, {"`\"가\"`", `"가"`, false, false}, {"`[1,2]`", "", true, false}, {"`\"ab\"`", "", true, false}, {"`true`", "", false, true}, {"`1`", "", false, true}, {"`{\"one\":1}`", "", false, true},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeSearchTestClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+volumeSearchTestPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, `{"volumes":[{"id":"id"}]}`)
			})
			search, err := blockstorage.SearchVolumes(context.Background(), client, blockstorage.SearchVolumesRequest{}, blockstorage.WithVolumeSearchExpression(tc.expression))
			if err != nil || search == nil || search.Value == nil || search.Volumes != nil || len(search.Pages) != 1 {
				t.Fatal("dynamic Search result", search, err)
			}
			result, err := blockstorage.GetVolume(context.Background(), client, blockstorage.GetVolumeRequest{}, blockstorage.WithVolumeSearchExpression(tc.expression))
			if result == nil || calls.Load() != 2 || len(result.Pages) != 1 || result.Volume != nil || result.Observed != nil || string(result.Value) != tc.want {
				t.Fatal(result, err, calls.Load())
			}
			if tc.ambiguous {
				var multiple *blockstorage.VolumeSelectionError
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

func TestSearchVolumesExpressionErrorsEvenWithoutIdentifierMatchesKeepOnlyPageEvidence(t *testing.T) {
	for _, expression := range []string{"[", "unknown_function(@)", "length(`1`)"} {
		t.Run(expression, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeSearchTestClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+volumeSearchTestPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, `{"volumes":[{"id":"other"}]}`)
			})
			result, err := blockstorage.SearchVolumes(context.Background(), client, blockstorage.SearchVolumesRequest{NameOrID: "missing"}, blockstorage.WithVolumeSearchExpression(expression))
			var response *resource.ResponseError
			if result == nil || calls.Load() != 1 || len(result.Pages) != 1 || result.Value != nil || result.Volumes != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &response) {
				t.Fatal(result, err)
			}
		})
	}
}

func TestSearchVolumesOwnedOptionsOriginalCallbacksAndSliceSnapshot(t *testing.T) {
	cloud := testcloud.New(t)
	client := volumeSearchTestClient(cloud)
	client.MoreHeaders = map[string]string{"X-Owned": "before"}
	name := "owned-name"
	location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"p"`), Name: &name}}
	filters := json.RawMessage(`{"status":"available"}`)
	bulk := blockstorage.WithVolumeSearchOptions(blockstorage.VolumeSearchOpts{Filters: &filters, Location: &location})
	filters[0] = '!'
	location.Project.ID[0] = '!'
	name = "caller"
	var callbacks []int
	var retained *blockstorage.VolumeSearchOpts
	options := []blockstorage.VolumeSearchOption{bulk, nil, func(*blockstorage.VolumeSearchOpts) error { callbacks = append(callbacks, 2); return nil }}
	options[1] = func(value *blockstorage.VolumeSearchOpts) error {
		callbacks = append(callbacks, 1)
		retained = value
		options[2] = func(*blockstorage.VolumeSearchOpts) error { return errors.New("mutated original slice") }
		client.MoreHeaders = map[string]string{"X-Owned": "after"}
		cloud.Provider.SetToken("live")
		return nil
	}
	cloud.Mux.HandleFunc("GET "+volumeSearchTestPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Owned") != "before" || r.Header.Get("X-Auth-Token") != "live" {
			t.Error("captured headers/live provider mismatch", r.Header)
		}
		(*retained.Filters)[0] = '!'
		*retained.Location.Project.Name = "retained caller"
		testcloud.JSON(w, 200, `{"volumes":[{"id":"id","status":"available","project_id":"p"}]}`)
	})
	result, err := blockstorage.SearchVolumes(context.Background(), client, blockstorage.SearchVolumesRequest{}, options...)
	if err != nil || result == nil || len(result.Volumes) != 1 || !reflect.DeepEqual(callbacks, []int{1, 2}) {
		t.Fatal(result, err, callbacks)
	}
	rows := volumeSearchTestRows(t, result.Value)
	project := volumeSearchTestFields(t, volumeSearchTestFields(t, rows[0]["location"])["project"])
	if string(project["name"]) != `"owned-name"` {
		t.Fatal(string(result.Value))
	}
	// Prepare must retain lazy filter syntax and return a separately owned snapshot.
	invalid := json.RawMessage(`not JSON`)
	prepared, err := blockstorage.PrepareVolumeSearchOptions(context.Background(), blockstorage.WithVolumeSearchFilters(invalid))
	if err != nil || prepared.Filters == nil || string(*prepared.Filters) != "not JSON" {
		t.Fatal(prepared, err)
	}
	invalid[0] = '!'
	if string(*prepared.Filters) != "not JSON" {
		t.Fatal("Prepare aliases factory input")
	}
}

func TestSearchVolumesPreflightContextOptionsAndFixedSourceRejectBeforeHTTP(t *testing.T) {
	for _, kind := range []string{"nil context", "canceled", "nil client", "wrong role", "nil option", "callback cancel", "source mutation"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeSearchTestClient(cloud)
			ctx := context.Background()
			callbacks := 0
			var sends atomic.Int32
			cause := errors.New("caller canceled")
			options := []blockstorage.VolumeSearchOption{func(*blockstorage.VolumeSearchOpts) error { callbacks++; return nil }}
			want := resource.ErrInvalidOption
			wantCallbacks := 0
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { sends.Add(1); w.WriteHeader(500) })
			switch kind {
			case "nil context":
				ctx = nil
			case "canceled":
				next, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx = next
				want = context.Canceled
			case "nil client":
				client = nil
			case "wrong role":
				client.Type = "compute"
				want = resource.ErrUnsupported
			case "nil option":
				options = append(options, nil)
				wantCallbacks = 1
			case "callback cancel":
				next, cancel := context.WithCancelCause(ctx)
				ctx = next
				options = []blockstorage.VolumeSearchOption{func(*blockstorage.VolumeSearchOpts) error { callbacks++; cancel(cause); return nil }, func(*blockstorage.VolumeSearchOpts) error { callbacks++; return nil }}
				want = context.Canceled
				wantCallbacks = 1
			case "source mutation":
				options = []blockstorage.VolumeSearchOption{func(*blockstorage.VolumeSearchOpts) error { callbacks++; client.Microversion = "3.60"; return nil }, func(*blockstorage.VolumeSearchOpts) error { callbacks++; return nil }}
				wantCallbacks = 1
			}
			result, err := blockstorage.SearchVolumes(ctx, client, blockstorage.SearchVolumesRequest{}, options...)
			if result != nil || !errors.Is(err, want) || callbacks != wantCallbacks || sends.Load() != 0 {
				t.Fatal(result, err, callbacks, sends.Load())
			}
			if want == context.Canceled && !errors.Is(err, cause) {
				t.Fatal(err)
			}
		})
	}
}

func TestGetVolumeDefaultExactListMissingAndAmbiguityKeepSharedIteratorPolicy(t *testing.T) {
	for _, multiple := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing", true: "duplicate"}[multiple], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeSearchTestClient(cloud)
			var calls atomic.Int32
			identity := "unsafe / human"
			rows := `[{"id":"other","name":"unsafe* / human"}]`
			next := ""
			if multiple {
				rows = `[{"id":"a","name":"unsafe / human"},{"id":"b","name":"unsafe / human"}]`
				next = "?unused=must-not-fetch"
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) != 1 || r.URL.Path != volumeSearchTestPath || r.URL.Query().Get("name") != identity {
					t.Error("default shared identity policy changed", r.URL)
				}
				testcloud.JSON(w, 200, volumeSearchTestPage(rows, next))
			})
			result, err := blockstorage.GetVolume(context.Background(), client, blockstorage.GetVolumeRequest{NameOrID: identity})
			if result == nil || calls.Load() != 1 || len(result.Pages) != 1 || result.Observed != nil || result.Volume != nil || result.Value != nil {
				t.Fatal(result, err, calls.Load())
			}
			if multiple {
				if !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
