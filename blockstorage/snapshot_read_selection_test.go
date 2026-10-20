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

func TestVolumeSnapshotReadMemberSeparatesSeededLogicalIDFromActualWireBody(t *testing.T) {
	for _, row := range []string{`{}`, `{"id":null}`, `{"id":false,"name":[1]}`, `{"id":"different","unknown":9007199254740993}`, `{"id":[],"connection":null,"microversion":null,"_synchronized":null,"self":null}`} {
		t.Run(row, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			body := `{"snapshot":` + row + `}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				snapshotReadContractWire(t, r, snapshotReadContractBasic+"/literal-ID", "test-token")
				if r.URL.RawQuery != "" {
					t.Error("member query leaked", r.URL)
				}
				w.Header().Set("X-Proof", "member")
				testcloud.JSON(w, 203, body)
			})
			result, err := blockstorage.GetVolumeSnapshotByID(context.Background(), client, blockstorage.GetVolumeSnapshotByIDRequest{ID: "literal-ID"})
			if err != nil || result == nil || result.Snapshot == nil || result.Observed == nil || len(result.Pages) != 0 || result.RequestedID != "literal-ID" || string(result.Observed.Body) != body || result.Snapshot.StatusCode != 203 || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			fields := snapshotReadContractFields(t, result.Value)
			wire := snapshotReadContractFields(t, json.RawMessage(row))
			id, present := wire["id"]
			if !present {
				if !result.SeededID || string(fields["id"]) != `"literal-ID"` {
					t.Fatal("missing source seed lost", result, string(result.Value))
				}
				if _, ok := result.Snapshot.Body["id"]; ok {
					t.Fatal("logical seed fabricated physical field")
				}
			} else if result.SeededID || string(fields["id"]) != string(id) || string(result.Snapshot.Body["id"]) != string(id) {
				t.Fatal("literal response ID changed", result, string(result.Value))
			}
			result.Observed.Body[0] = '!'
			result.Observed.Header.Set("X-Proof", "changed")
			result.Snapshot.Body["name"] = json.RawMessage(`"caller"`)
			if !json.Valid(result.Value) || result.Snapshot.Header.Get("X-Proof") != "member" {
				t.Fatal("member components share mutable storage")
			}
		})
	}
}

func TestVolumeSnapshotReadMemberAcceptsFlatPartialEmptyAndMalformedUTF8JSON(t *testing.T) {
	for _, body := range []string{`{"id":"flat","force":"FALSE","size":3.9}`, `{"snapshot":{"description":"partial"}}`, "", " \n\t", `{"snapshot":`, `not JSON`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Proof", "physical")
				testcloud.JSON(w, 203, body)
			})
			result, err := blockstorage.GetVolumeSnapshotByID(context.Background(), client, blockstorage.GetVolumeSnapshotByIDRequest{ID: "seed"})
			if err != nil || result == nil || result.Snapshot == nil || result.Observed == nil || string(result.Observed.Body) != body || result.RequestedID != "seed" || len(result.Pages) != 0 || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			fields := snapshotReadContractFields(t, result.Value)
			if body == `{"id":"flat","force":"FALSE","size":3.9}` {
				if result.SeededID || string(fields["id"]) != `"flat"` || string(fields["is_forced"]) != "false" || string(fields["size"]) != "3" {
					t.Fatal(result, string(result.Value))
				}
			} else {
				if !result.SeededID || string(fields["id"]) != `"seed"` {
					t.Fatal(result, string(result.Value))
				}
				if _, present := result.Snapshot.Body["id"]; present {
					t.Fatal("seed entered physical body")
				}
				if !json.Valid([]byte(body)) && len(result.Snapshot.Body) != 0 {
					t.Fatal("malformed body fabricated physical fields", result)
				}
			}
		})
	}
}

func TestVolumeSnapshotReadMemberSchemaAndUTF8ErrorsRemainPhysicalAndTerminal(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `false`, `{"snapshot":null}`, `{"snapshot":[]}`, `{"snapshot":false}`, `{"snapshot":{"force":"bad"}}`, `{"snapshot":{"size":"²"}}`, string([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'})} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Proof", "invalid")
				testcloud.JSON(w, 203, body)
			})
			result, err := blockstorage.GetVolumeSnapshot(context.Background(), client, blockstorage.GetVolumeSnapshotRequest{NameOrID: "wanted"})
			var physical *resource.ResponseError
			if result == nil || result.Value != nil || result.Snapshot != nil || result.Observed == nil || len(result.Pages) != 0 || calls.Load() != 1 || !errors.As(err, &physical) || physical.StatusCode != 203 || string(physical.Body) != body || physical.Header.Get("X-Proof") != "invalid" {
				t.Fatal(result, err, calls.Load())
			}
			snapshotReadContractOperation(t, err, "GetVolumeSnapshot")
		})
	}
}

func TestVolumeSnapshotReadByIDNeverFallsBackOnCleanNativeRejections(t *testing.T) {
	for _, code := range []int{400, 403, 404, 429} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				snapshotReadContractWire(t, r, snapshotReadContractBasic+"/missing", "test-token")
				w.Header().Set("X-Proof", "rejected")
				testcloud.JSON(w, code, `{"error":"direct member rejection"}`)
			})
			result, err := blockstorage.GetVolumeSnapshotByID(context.Background(), client, blockstorage.GetVolumeSnapshotByIDRequest{ID: "missing"})
			var native gophercloud.ErrUnexpectedResponseCode
			var physical *resource.ResponseError
			if result == nil || result.Value != nil || result.Snapshot != nil || result.Observed != nil || len(result.Pages) != 0 || calls.Load() != 1 || !errors.As(err, &native) || native.Actual != code || native.ResponseHeader.Get("X-Proof") != "rejected" || errors.As(err, &physical) {
				t.Fatal(result, err, calls.Load())
			}
			snapshotReadContractOperation(t, err, "GetVolumeSnapshotByID")
		})
	}
}

func TestVolumeSnapshotReadFindCleanFallbackUsesExactNameAndActualResponseIdentity(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					snapshotReadContractWire(t, r, snapshotReadContractBasic+"/wanted", "test-token")
					testcloud.JSON(w, code, `{"error":"compatible member rejection"}`)
				case 2:
					snapshotReadContractWire(t, r, snapshotReadContractDetail, "test-token")
					if r.URL.Query().Get("name") != "wanted" || len(r.URL.Query()) != 1 {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 203, snapshotReadContractPage(`[{"id":"wrong","name":"Wanted"},{"id":false,"name":"wanted"},{"id":"unrelated","name":"other"}]`, ""))
				default:
					t.Error("extra fallback/restart", r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := blockstorage.GetVolumeSnapshot(context.Background(), client, blockstorage.GetVolumeSnapshotRequest{NameOrID: "wanted"})
			if err != nil || result == nil || result.Snapshot == nil || result.Observed != nil || len(result.Pages) != 1 || result.SeededID || string(result.Snapshot.Body["id"]) != "false" || string(snapshotReadContractFields(t, result.Value)["id"]) != "false" || calls.Load() != 2 {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestVolumeSnapshotReadFindUnsafeAndWildcardNamesKeepExactLiteralSelection(t *testing.T) {
	for _, name := range []string{"wanted/name", "wanted%2Fname", "wanted?query", "wanted\\name", "wanted\u00a0name", "wanted*"} {
		t.Run(name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var member, list atomic.Int32
			raw, _ := json.Marshal(name)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != snapshotReadContractDetail {
					member.Add(1)
					testcloud.JSON(w, 404, `{"error":"missing"}`)
					return
				}
				list.Add(1)
				if r.URL.Query().Get("name") != name || len(r.URL.Query()) != 1 {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, snapshotReadContractPage(`[{"id":"decoy","name":"wanted-other"},{"id":"exact","name":`+string(raw)+`}]`, ""))
			})
			result, err := blockstorage.GetVolumeSnapshot(context.Background(), client, blockstorage.GetVolumeSnapshotRequest{NameOrID: name})
			wantMember := int32(0)
			if name == "wanted*" {
				wantMember = 1
			}
			if err != nil || result == nil || result.Snapshot == nil || string(result.Snapshot.Body["id"]) != `"exact"` || member.Load() != wantMember || list.Load() != 1 {
				t.Fatal(result, err, member.Load(), list.Load())
			}
		})
	}
}

func TestVolumeSnapshotReadFindAmbiguityStopsBeforeUnusedBadTailButUniqueMustExhaust(t *testing.T) {
	for _, ambiguous := range []bool{true, false} {
		t.Run(map[bool]string{true: "duplicate early", false: "unique consumes tail"}[ambiguous], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			rows := `[{"id":"first","name":"name/only"},`
			if ambiguous {
				rows += `{"id":"second","name":"name/only"},`
			}
			rows += `{"id":"bad","force":"bad"}]`
			body := snapshotReadContractPage(rows, "https://unused.invalid/")
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, body) })
			result, err := blockstorage.GetVolumeSnapshot(context.Background(), client, blockstorage.GetVolumeSnapshotRequest{NameOrID: "name/only"})
			var physical *resource.ResponseError
			if result == nil || result.Value != nil || result.Snapshot != nil || result.Observed != nil || len(result.Pages) != 1 || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			if ambiguous {
				if !errors.Is(err, resource.ErrAmbiguous) || errors.As(err, &physical) {
					t.Fatal("unused bad tail masked earlier ambiguity", err)
				}
			} else if !errors.As(err, &physical) || string(physical.Body) != body {
				t.Fatal("unique early result escaped unsuccessful exhaustion", err)
			}
		})
	}
}

func TestVolumeSnapshotReadSearchCompletesQuerylessListBeforeSelectionAndReturnsArbitraryExpression(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		snapshotReadContractWire(t, r, snapshotReadContractDetail, "test-token")
		n := calls.Add(1)
		if n == 1 {
			if r.URL.RawQuery != "" {
				t.Error("search selection entered server query", r.URL)
			}
			testcloud.JSON(w, 200, snapshotReadContractPage(`[{"id":"first","name":"disk-one"}]`, "?marker=next"))
			return
		}
		if n != 2 {
			t.Error("search refresh/restart", r.URL)
		}
		testcloud.JSON(w, 200, snapshotReadContractPage(`[{"id":"second","name":"disk-two"},{"id":"third","name":"other"}]`, ""))
	})
	result, err := blockstorage.SearchVolumeSnapshots(context.Background(), client, blockstorage.SearchVolumeSnapshotsRequest{NameOrID: "disk-*"}, blockstorage.WithVolumeSnapshotSearchExpression("{ids: [].id, count: length(@)}"))
	if err != nil || result == nil || result.Snapshots != nil || len(result.Pages) != 2 || calls.Load() != 2 {
		t.Fatal(result, err, calls.Load())
	}
	fields := snapshotReadContractFields(t, result.Value)
	if string(fields["ids"]) != `["first","second"]` || string(fields["count"]) != "2" {
		t.Fatal(string(result.Value))
	}
}

func TestVolumeSnapshotReadSearchDoesNotHideExcludedMalformedLaterRows(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls atomic.Int32
	first := snapshotReadContractPage(`[{"id":"match","name":"wanted"}]`, "?marker=next")
	second := snapshotReadContractPage(`[{"id":"excluded","name":"other","force":"bad"}]`, "")
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			testcloud.JSON(w, 200, first)
		} else if n == 2 {
			w.Header().Set("X-Proof", "late")
			testcloud.JSON(w, 203, second)
		} else {
			t.Error("search replay", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.SearchVolumeSnapshots(context.Background(), client, blockstorage.SearchVolumeSnapshotsRequest{NameOrID: "wanted"})
	var physical *resource.ResponseError
	if result == nil || result.Value != nil || result.Snapshots != nil || len(result.Pages) != 2 || calls.Load() != 2 || !errors.As(err, &physical) || physical.StatusCode != 203 || string(physical.Body) != second || physical.Header.Get("X-Proof") != "late" {
		t.Fatal(result, err, calls.Load())
	}
}

func TestVolumeSnapshotReadGetNonNullFalseyFiltersDispatchFullSearchButNullUsesFind(t *testing.T) {
	for _, filter := range []string{"null", "{}", "false", "0", "[]", `""`} {
		t.Run(filter, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if filter == "null" {
					snapshotReadContractWire(t, r, snapshotReadContractBasic+"/wanted", "test-token")
					testcloud.JSON(w, 200, `{"snapshot":{}}`)
					return
				}
				snapshotReadContractWire(t, r, snapshotReadContractDetail, "test-token")
				if r.URL.RawQuery != "" {
					t.Error("falsey explicit filter incorrectly used find/name hint", r.URL)
				}
				testcloud.JSON(w, 200, snapshotReadContractPage(`[{"id":"found","name":"wanted"},{"id":"ignored","name":"other"}]`, ""))
			})
			result, err := blockstorage.GetVolumeSnapshot(context.Background(), client, blockstorage.GetVolumeSnapshotRequest{NameOrID: "wanted"}, blockstorage.WithVolumeSnapshotSearchFilters(json.RawMessage(filter)))
			if err != nil || result == nil || result.Snapshot == nil || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			if filter == "null" {
				if !result.SeededID || result.Observed == nil || len(result.Pages) != 0 || string(snapshotReadContractFields(t, result.Value)["id"]) != `"wanted"` {
					t.Fatal(result)
				}
			} else if result.SeededID || result.Observed != nil || len(result.Pages) != 1 || string(result.Snapshot.Body["id"]) != `"found"` {
				t.Fatal(result)
			}
		})
	}
}

func TestVolumeSnapshotReadGetArbitraryExpressionDoesNotInventSnapshotAssociation(t *testing.T) {
	for _, tc := range []struct {
		expression, want string
		wantError        bool
	}{{"`[1]`", "1", false}, {"`[false]`", "false", false}, {"`[1,2]`", "", true}, {"`false`", "", false}, {"`null`", "", false}, {"`{\"key\":1}`", "", true}} {
		t.Run(tc.expression, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, snapshotReadContractPage(`[{"id":"source"}]`, ""))
			})
			result, err := blockstorage.GetVolumeSnapshot(context.Background(), client, blockstorage.GetVolumeSnapshotRequest{}, blockstorage.WithVolumeSnapshotSearchExpression(tc.expression))
			if result == nil || result.Snapshot != nil || result.Observed != nil || len(result.Pages) != 1 || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			if tc.wantError {
				var physical *resource.ResponseError
				if err == nil || result.Value != nil || errors.As(err, &physical) {
					t.Fatal(result, err)
				}
			} else if err != nil || string(result.Value) != tc.want {
				t.Fatal(result, err, string(result.Value))
			}
		})
	}
}

func TestVolumeSnapshotReadLookupFailureRetainsPagesButNeverSuppressedMemberProof(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			w.Header().Set("X-Proof", "old suppressed")
			testcloud.JSON(w, 404, `{"error":"old"}`)
		case 2:
			w.Header().Set("X-Proof", "first page")
			testcloud.JSON(w, 203, snapshotReadContractPage(`[{"id":"match","name":"wanted"}]`, "?marker=late"))
		case 3:
			w.Header().Set("X-Proof", "current native")
			testcloud.JSON(w, 429, `{"error":"current"}`)
		default:
			t.Error("lookup replay", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.GetVolumeSnapshot(context.Background(), client, blockstorage.GetVolumeSnapshotRequest{NameOrID: "wanted"})
	var physical *resource.ResponseError
	var native gophercloud.ErrUnexpectedResponseCode
	if result == nil || result.Value != nil || result.Snapshot != nil || result.Observed != nil || len(result.Pages) != 1 || result.Pages[0].Header.Get("X-Proof") != "first page" || calls.Load() != 3 || errors.As(err, &physical) || !errors.As(err, &native) || native.Actual != 429 || native.ResponseHeader.Get("X-Proof") != "current native" {
		t.Fatal(result, err, calls.Load())
	}
}

func TestVolumeSnapshotReadMemberUsesSourceSub400PolicyIncludingOpaqueEmpty204(t *testing.T) {
	for _, code := range []int{201, 203, 204, 300} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Proof", "sub400")
				testcloud.JSON(w, code, `{"snapshot":{"id":"wire"}}`)
			})
			result, err := blockstorage.GetVolumeSnapshotByID(context.Background(), client, blockstorage.GetVolumeSnapshotByIDRequest{ID: "requested"})
			if err != nil || result == nil || result.Snapshot == nil || result.Observed == nil || result.Observed.StatusCode != code || result.Snapshot.StatusCode != code || result.Observed.Header.Get("X-Proof") != "sub400" || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			if code == 204 {
				if !result.SeededID || len(result.Observed.Body) != 0 || len(result.Snapshot.Body) != 0 || string(snapshotReadContractFields(t, result.Value)["id"]) != `"requested"` {
					t.Fatal(result)
				}
			} else if result.SeededID || string(result.Snapshot.Body["id"]) != `"wire"` {
				t.Fatal(result)
			}
		})
	}
}

func TestVolumeSnapshotReadSearchLocalFilterErrorsDoNotBorrowAdmittedHTTPStatus(t *testing.T) {
	for _, tc := range []struct {
		filter, rows string
		fail         bool
	}{{`{"metadata":{"missing":1},"unknown_field":1}`, `[{"id":"mismatch","metadata":{}}]`, false}, {`{"unknown_field":1,"metadata":{"missing":1}}`, `[{"id":"ordered","metadata":{}}]`, true}, {`"length(1)"`, `[]`, true}} {
		t.Run(tc.filter, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			body := snapshotReadContractPage(tc.rows, "")
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Proof", "admitted")
				testcloud.JSON(w, 203, body)
			})
			result, err := blockstorage.SearchVolumeSnapshots(context.Background(), client, blockstorage.SearchVolumeSnapshotsRequest{}, blockstorage.WithVolumeSnapshotSearchFilters(json.RawMessage(tc.filter)))
			var physical *resource.ResponseError
			if result == nil || len(result.Pages) != 1 || result.Pages[0].StatusCode != 203 || string(result.Pages[0].Body) != body || calls.Load() != 1 || errors.As(err, &physical) {
				t.Fatal(result, err, calls.Load())
			}
			if tc.fail {
				if !errors.Is(err, resource.ErrInvalidOption) || result.Value != nil || result.Snapshots != nil {
					t.Fatal(result, err)
				}
			} else if err != nil || string(result.Value) != "[]" {
				t.Fatal(result, err)
			}
		})
	}
}
