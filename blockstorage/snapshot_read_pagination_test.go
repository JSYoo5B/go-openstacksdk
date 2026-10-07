package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestVolumeSnapshotReadInitialLimitFollowsShortPageExcludedRawDictionaryMarker(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var calls atomic.Int32
	first := snapshotReadContractPage(`[{"id":"kept","description":"wanted","unknown":9007199254740993},{"id":{"second key":false,"first key":{"nested":null}},"description":"excluded"}]`, "")
	// Empty resources end the source loop before unusable links are consumed.
	second := `{"snapshots":[],"links":[null],"next":"https://unused.invalid/snapshots"}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		snapshotReadContractWire(t, r, snapshotReadContractDetail, "test-token")
		switch calls.Add(1) {
		case 1:
			want := url.Values{"limit": {"5"}}
			if !reflect.DeepEqual(r.URL.Query(), want) || r.URL.RawQuery != want.Encode() {
				t.Error("initial raw limit or local predicate changed", r.URL)
			}
			w.Header().Set("X-Proof", "short first page")
			testcloud.JSON(w, 203, first)
		case 2:
			// The last consumed row supplies the marker even when excluded.
			// Requests expands the untyped dictionary ID into ordered keys,
			// irrespective of its false/nonnull dictionary values.
			want := url.Values{"limit": {"5"}, "marker": {"second key", "first key"}}
			if !reflect.DeepEqual(r.URL.Query(), want) || r.URL.RawQuery != want.Encode() {
				t.Error("short-page continuation lost raw last ID or key order", r.URL, want)
			}
			w.Header().Set("X-Proof", "empty second page")
			testcloud.JSON(w, 200, second)
		default:
			t.Error("empty EOF links were consumed or listing restarted", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.ListVolumeSnapshots(ctx, client,
		blockstorage.WithVolumeSnapshotListFilters(json.RawMessage(`{"limit":5,"description":"wanted"}`)))
	if err != nil || result == nil || calls.Load() != 2 || len(result.Pages) != 2 || len(result.Snapshots) != 1 || string(result.Snapshots[0].Body["id"]) != `"kept"` {
		t.Fatal(result, err, calls.Load())
	}
	rows := snapshotReadContractRows(t, result.Value)
	if len(rows) != 1 || string(rows[0]["id"]) != `"kept"` || result.Snapshots[0].StatusCode != 203 || result.Snapshots[0].Header.Get("X-Proof") != "short first page" || string(result.Snapshots[0].Body["unknown"]) != "9007199254740993" {
		t.Fatal("logical result or selected physical origin changed", result, string(result.Value))
	}
	if string(result.Pages[0].Body) != first || result.Pages[0].StatusCode != 203 || result.Pages[0].Header.Get("X-Proof") != "short first page" || string(result.Pages[1].Body) != second || result.Pages[1].StatusCode != 200 || result.Pages[1].Header.Get("X-Proof") != "empty second page" {
		t.Fatal("short and empty physical pages were not independently retained", result)
	}
	// Proof ownership is independent of the committed normalized and raw row.
	result.Pages[0].Body[0] = '!'
	result.Pages[0].Header.Set("X-Proof", "caller page mutation")
	result.Pages[1].Header.Set("X-Proof", "caller EOF mutation")
	if !json.Valid(result.Value) || string(result.Snapshots[0].Body["id"]) != `"kept"` || result.Snapshots[0].Header.Get("X-Proof") != "short first page" || string(result.Pages[1].Body) != second {
		t.Fatal("pagination evidence aliases the completed row or other page")
	}
}

func TestVolumeSnapshotReadCanonicalURLCycleWithoutMarkerEqualityKeepsTwoPagesAtomically(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var calls atomic.Int32
	first := snapshotReadContractPage(`[{"id":"first","name":"one"}]`, "?status=only")
	second := snapshotReadContractPage(`[{"id":"second","name":"two"}]`, "?status=only")
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		snapshotReadContractWire(t, r, snapshotReadContractDetail, "test-token")
		switch calls.Add(1) {
		case 1:
			if r.URL.RawQuery != "" {
				t.Error("initial ordinary list query changed", r.URL)
			}
			w.Header().Set("X-Proof", "first cycle page")
			testcloud.JSON(w, 203, first)
		case 2:
			want := url.Values{"status": {"only"}}
			if !reflect.DeepEqual(r.URL.Query(), want) || r.URL.RawQuery != want.Encode() || r.URL.Query().Has("marker") {
				t.Error("marker was invented for advertised status-only continuation", r.URL)
			}
			w.Header().Set("X-Proof", "second cycle page")
			testcloud.JSON(w, 200, second)
		default:
			t.Error("canonical same URL received a third request", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.ListVolumeSnapshots(ctx, client)
	var cycle *resource.PaginationCycleError
	var physical *resource.ResponseError
	if result == nil || result.Value != nil || result.Snapshots != nil || calls.Load() != 2 || len(result.Pages) != 2 || !errors.Is(err, resource.ErrPaginationCycle) || !errors.As(err, &cycle) || cycle.URL != cloud.Server.URL+snapshotReadContractDetail+"?status=only" || errors.As(err, &physical) {
		t.Fatal("full URL cycle committed partial values or borrowed a physical error", result, err, calls.Load())
	}
	snapshotReadContractOperation(t, err, "ListVolumeSnapshots")
	if string(result.Pages[0].Body) != first || result.Pages[0].StatusCode != 203 || result.Pages[0].Header.Get("X-Proof") != "first cycle page" || string(result.Pages[1].Body) != second || result.Pages[1].StatusCode != 200 || result.Pages[1].Header.Get("X-Proof") != "second cycle page" {
		t.Fatal("cycle lost already admitted physical observations", result)
	}
	result.Pages[0].Header.Set("X-Proof", "caller first mutation")
	result.Pages[0].Body[0] = '!'
	if string(result.Pages[1].Body) != second || result.Pages[1].Header.Get("X-Proof") != "second cycle page" {
		t.Fatal("cycle pages share mutable proof storage")
	}
}
