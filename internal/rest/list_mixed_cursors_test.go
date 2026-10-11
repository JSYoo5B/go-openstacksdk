package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func markerOrOffsetSpec(t *testing.T, reply http.HandlerFunc) CollectionSpec[listItem] {
	spec := listSpec(t, reply)
	spec.Paging.MarkerOrOffsetPagination = true
	spec.Paging.MarkerFallback = true
	spec.Paging.MarkerOnShortPage = true
	spec.Paging.StopOnEmptyPage = true
	spec.Paging.Marker = func(item *listItem) (string, error) { return item.ID, nil }
	return spec
}

func TestListMarkerOrOffsetSwitchesFamiliesAndUsesRetainedWireFallback(t *testing.T) {
	var calls atomic.Int32
	spec := markerOrOffsetSpec(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("project_id") != "fixed" || q.Get("limit") != "2" || q.Has("marker") && q.Has("offset") {
			t.Error("request ownership", r.URL)
		}
		switch n := calls.Add(1); n {
		case 1:
			if q.Get("marker") != "seed" || q.Has("offset") {
				t.Error(r.URL)
			}
			writeList(w, `{"items":[{"id":"one"}],"next":"?offset=1"}`)
		case 2:
			if q.Get("offset") != "1" || q.Has("marker") {
				t.Error(r.URL)
			}
			writeList(w, `{"items":[{"id":"two"}]}`)
		case 3:
			if q.Get("marker") != "two" || q.Has("offset") {
				t.Error(r.URL)
			}
			writeList(w, `{"items":[{"id":"three"}],"next":"?offset=2"}`)
		case 4:
			if q.Get("offset") != "2" || q.Has("marker") {
				t.Error(r.URL)
			}
			writeList(w, `{"items":[]}`)
		default:
			t.Error("unexpected HTTP", n)
		}
	})
	var ids []string
	for value, err := range List(context.Background(), spec, url.Values{"marker": {"seed"}, "limit": {"2"}, "project_id": {"fixed"}}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
		value.ID = "caller-mutated"
	}
	if fmt.Sprint(ids) != "[one two three]" || calls.Load() != 4 {
		t.Fatal(ids, calls.Load())
	}
}

func TestListMarkerOrOffsetRetainsOffsetFloorAcrossMarkerPages(t *testing.T) {
	for _, last := range []string{"1", "2"} {
		t.Run(last, func(t *testing.T) {
			var calls atomic.Int32
			body := `{"items":[{"id":"three"}],"next":"?offset=` + last + `"}`
			spec := markerOrOffsetSpec(t, func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					writeList(w, `{"items":[{"id":"one"}],"next":"?offset=2"}`)
				case 2:
					writeList(w, `{"items":[{"id":"two"}]}`)
				case 3:
					writeList(w, body)
				default:
					t.Error("offset rewind performed HTTP", r.URL)
				}
			})
			values, err := collectList(context.Background(), spec, url.Values{"limit": {"2"}})
			var proof *resource.ResponseError
			var cycle *resource.PaginationCycleError
			expected := errors.Is(err, resource.ErrInvalidOption)
			if last == "2" {
				expected = errors.As(err, &cycle)
			}
			if !expected || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || proof.Header.Get("X-Page") != "kept" || len(values) != 3 || calls.Load() != 3 {
				t.Fatal(values, err, proof, calls.Load())
			}
		})
	}
}

func TestListMarkerOrOffsetRejectsMalformedAndRetargetedLinks(t *testing.T) {
	for name, next := range map[string]string{
		"both": "?marker=new&offset=1", "duplicate": "?offset=1&offset=2", "negative": "?offset=-1", "repeat": "?offset=0", "filter": "?offset=1&project_id=other", "path": "/v1/other?offset=1", "origin": "http://other.invalid/v1/items?offset=1", "extra": "?offset=1&all_projects=true", "limit": "?offset=1&limit=3",
	} {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			body := fmt.Sprintf(`{"items":[{"id":"one"}],"next":%q}`, next)
			spec := markerOrOffsetSpec(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); writeList(w, body) })
			values, err := collectList(context.Background(), spec, url.Values{"project_id": {"fixed"}, "limit": {"2"}})
			var proof *resource.ResponseError
			if err == nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || len(values) != 1 || calls.Load() != 1 {
				t.Fatal(values, err, proof, calls.Load())
			}
		})
	}
}

func TestListMarkerOrOffsetRequiresOneValidInitialCursorAndPolicy(t *testing.T) {
	for name, query := range map[string]url.Values{"both": {"marker": {"start"}, "offset": {"1"}}, "negative": {"offset": {"-1"}}, "duplicate": {"offset": {"1", "2"}}, "blank": {"marker": {""}}, "policy": nil} {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			spec := markerOrOffsetSpec(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) })
			if name == "policy" {
				spec.Paging.OffsetPagination = true
			}
			_, err := collectList(context.Background(), spec, query)
			if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(err, calls.Load())
			}
		})
	}
}
