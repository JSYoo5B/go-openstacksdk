package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"gophercloudsdk/resource"
)

func TestListOffsetContinuationEstablishesServerLimitAndKeepsFilters(t *testing.T) {
	var calls atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("project_id") != "fixed" || r.URL.Query().Has("marker") {
			t.Errorf("offset request changed scope: %s", r.URL)
		}
		switch r.URL.Query().Get("offset") {
		case "":
			if r.URL.Query().Has("limit") {
				t.Errorf("invented first-page limit: %s", r.URL)
			}
			writeList(w, `{"items":[{"id":"one"}],"next":"?offset=1&limit=2"}`)
		case "1":
			if r.URL.Query().Get("limit") != "2" {
				t.Errorf("server limit missing: %s", r.URL)
			}
			writeList(w, `{"items":[],"next":"?offset=2"}`)
		case "2":
			if r.URL.Query().Get("limit") != "2" {
				t.Errorf("server limit not retained: %s", r.URL)
			}
			writeList(w, `{"items":[{"id":"two"}]}`)
		default:
			t.Errorf("unexpected offset: %s", r.URL)
		}
	})
	spec.Paging.OffsetPagination = true
	query := url.Values{"project_id": {"fixed"}}
	stream := List(context.Background(), spec, query)
	query.Set("project_id", "changed later")
	var values []*listItem
	for value, err := range stream {
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, value)
	}
	if len(values) != 2 || calls.Load() != 3 {
		t.Fatalf("values=%v calls=%d", values, calls.Load())
	}
}

func TestListOffsetPolicyDoesNotEnableOtherCollectionCursors(t *testing.T) {
	var calls atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		writeList(w, `{"items":[{"id":"one"}],"next":"?offset=1&limit=2"}`)
	})
	values, err := collectList(context.Background(), spec, nil)
	if !errors.Is(err, resource.ErrInvalidOption) || len(values) != 1 || calls.Load() != 1 {
		t.Fatalf("default policy followed offset: values=%v calls=%d err=%v", values, calls.Load(), err)
	}
}

func TestListOffsetRejectsMalformedInitialCursorBeforeHTTP(t *testing.T) {
	for name, query := range map[string]url.Values{
		"negative": {"offset": {"-1"}}, "empty": {"offset": {""}},
		"duplicate": {"offset": {"1", "2"}}, "float": {"offset": {"1.5"}},
		"overflow": {"offset": {"18446744073709551616"}}, "signed": {"offset": {"+1"}},
		"marker": {"marker": {"other"}}, "mixed": {"marker": {"other"}, "offset": {"1"}},
	} {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1) })
			spec.Paging.OffsetPagination = true
			_, err := collectList(context.Background(), spec, query)
			if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatalf("calls=%d err=%v", calls.Load(), err)
			}
		})
	}
}

func TestListOffsetRejectsScopeChangesAndNonforwardCursors(t *testing.T) {
	for name, next := range map[string]string{
		"backwards": "?offset=1", "repeat": "?offset=2", "omitted": "?limit=2",
		"negative": "?offset=-1", "duplicate": "?offset=3&offset=4", "float": "?offset=3.5",
		"overflow": "?offset=18446744073709551616", "marker": "?offset=3&marker=other",
		"filter": "?offset=3&project_id=other", "newfilter": "?offset=3&all_projects=true",
		"increase": "?offset=3&limit=3", "reduce": "?offset=3&limit=1",
		"origin": "https://foreign.invalid/v1/items?offset=3", "path": "/v1/other?offset=3",
		"fragment": "?offset=3#fragment", "badquery": "?offset=3&vendor=%zz",
	} {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				writeList(w, fmt.Sprintf(`{"items":[{"id":"one"}],"next":%q}`, next))
			})
			spec.Paging.OffsetPagination = true
			values, err := collectList(context.Background(), spec, url.Values{"offset": {"2"}, "limit": {"2"}, "project_id": {"fixed"}})
			var response *resource.ResponseError
			if !errors.As(err, &response) || response.StatusCode != 200 || response.Header.Get("X-Page") != "kept" || len(values) != 1 || calls.Load() != 1 {
				t.Fatalf("values=%v calls=%d err=%v response=%v", values, calls.Load(), err, response)
			}
			if name == "repeat" || name == "omitted" {
				if !errors.Is(err, resource.ErrPaginationCycle) {
					t.Fatalf("cursor failed to progress without cycle error: %v", err)
				}
			}
		})
	}
}

func TestListOffsetCannotChangeEstablishedServerLimit(t *testing.T) {
	for name, initialNext := range map[string]string{"established": "?offset=1&limit=2", "notestablished": "?offset=1"} {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !r.URL.Query().Has("offset") {
					writeList(w, fmt.Sprintf(`{"items":[{"id":"one"}],"next":%q}`, initialNext))
				} else {
					writeList(w, `{"items":[{"id":"two"}],"next":"?offset=2&limit=3"}`)
				}
			})
			spec.Paging.OffsetPagination = true
			values, err := collectList(context.Background(), spec, nil)
			if !errors.Is(err, resource.ErrInvalidOption) || len(values) != 2 || calls.Load() != 2 {
				t.Fatalf("values=%v calls=%d err=%v", values, calls.Load(), err)
			}
		})
	}
}

func TestListOffsetCapAndBreakDoNotInspectUnconsumedContinuation(t *testing.T) {
	var calls atomic.Int32
	spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		writeList(w, `{"items":[{"id":"one"},null],"next":"https://foreign.invalid/items?offset=2"}`)
	})
	spec.Paging.OffsetPagination = true
	for value, err := range ListWithControl(context.Background(), spec, nil, ListControl{MaxItems: 1}) {
		if err != nil || value.ID != "one" {
			t.Fatalf("value=%v err=%v", value, err)
		}
	}
	for value, err := range List(context.Background(), spec, nil) {
		if err != nil || value.ID != "one" {
			t.Fatalf("value=%v err=%v", value, err)
		}
		break
	}
	if calls.Load() != 2 {
		t.Fatalf("unconsumed row/link triggered another request: %d", calls.Load())
	}
}
