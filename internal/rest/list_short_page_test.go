package rest

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestListShortPageMarkerPolicyUsesWireRowsAndStopsOnEmptyPage(t *testing.T) {
	for _, shortPage := range []bool{false, true} {
		t.Run(map[bool]string{false: "full_page_only", true: "every_nonempty_page"}[shortPage], func(t *testing.T) {
			var calls atomic.Int32
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("limit") != "2" || r.URL.Query().Get("project_id") != "fixed" {
					t.Errorf("request lost original filters: %s", r.URL)
				}
				switch r.URL.Query().Get("marker") {
				case "":
					writeList(w, `{"items":[{"id":"first","wire_marker":"wire-first"},{"id":"second","wire_marker":"wire-second"}]}`)
				case "wire-second":
					writeList(w, `{"items":[{"id":"short","wire_marker":"wire-short"}]}`)
				case "wire-short":
					writeList(w, `{"items":[]}`)
				default:
					t.Errorf("marker came from a mutable model: %s", r.URL)
				}
			})
			spec.Paging.MarkerFallback, spec.Paging.MarkerOnShortPage = true, shortPage
			spec.Paging.Marker = func(item *listItem) (string, error) { return item.WireMarker, nil }
			var count int
			for value, err := range List(context.Background(), spec, url.Values{"limit": {"2"}, "project_id": {"fixed"}}) {
				if err != nil {
					t.Fatal(err)
				}
				count++
				value.WireMarker = "consumer mutation"
			}
			want := int32(2)
			if shortPage {
				want = 3
			}
			if count != 3 || calls.Load() != want {
				t.Fatalf("rows=%d calls=%d want=%d", count, calls.Load(), want)
			}
		})
	}
}

func TestListShortPageMarkerPolicyRequiresExplicitLimitAndHonorsBreak(t *testing.T) {
	for _, limit := range []string{"", "5"} {
		var calls atomic.Int32
		spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			writeList(w, `{"items":[{"id":"first","wire_marker":"wire-first"}]}`)
		})
		spec.Paging.MarkerFallback, spec.Paging.MarkerOnShortPage = true, true
		spec.Paging.Marker = func(item *listItem) (string, error) { return item.WireMarker, nil }
		query := make(url.Values)
		if limit != "" {
			query.Set("limit", limit)
		}
		for _, err := range List(context.Background(), spec, query) {
			if err != nil {
				t.Fatal(err)
			}
			if limit != "" {
				break
			}
		}
		if calls.Load() != 1 {
			t.Fatalf("limit=%q fetched %d pages", limit, calls.Load())
		}
	}
}

func TestListShortPageMarkerPolicyKeepsCycleEvidence(t *testing.T) {
	var calls atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		writeList(w, `{"items":[{"id":"first","wire_marker":"same"}]}`)
	})
	spec.Paging.MarkerFallback, spec.Paging.MarkerOnShortPage = true, true
	spec.Paging.Marker = func(item *listItem) (string, error) { return item.WireMarker, nil }
	_, err := collectList(context.Background(), spec, url.Values{"limit": {"5"}})
	var response *resource.ResponseError
	if !errors.Is(err, resource.ErrPaginationCycle) || !errors.As(err, &response) || response.StatusCode != 200 || response.Header.Get("X-Page") != "kept" || calls.Load() != 2 {
		t.Fatalf("calls=%d response=%v err=%v", calls.Load(), response, err)
	}
}
