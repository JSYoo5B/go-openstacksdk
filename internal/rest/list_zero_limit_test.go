package rest

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestListZeroLimitRequiresServiceOptIn(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "default rejects", true: "enabled accepts"}[enabled], func(t *testing.T) {
			calls := 0
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Query().Get("limit") != "0" {
					t.Errorf("zero was omitted: %s", r.URL)
				}
				writeList(w, `{"items":[{"id":"only"}]}`)
			})
			spec.Paging.AllowZeroLimit = enabled
			spec.Paging.MarkerFallback = true
			spec.Paging.Marker = func(*listItem) (string, error) {
				t.Error("zero enabled marker fallback")
				return "", errors.New("unexpected marker")
			}
			rows, err := collectList(context.Background(), spec, url.Values{"limit": {"0"}})
			if enabled {
				if err != nil || len(rows) != 1 || calls != 1 {
					t.Fatal(rows, err, calls)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(rows, err, calls)
			}
		})
	}
}

func TestListZeroLimitRetainsExplicitLinksAndFalseyCapHint(t *testing.T) {
	for _, capHint := range []bool{false, true} {
		t.Run(map[bool]string{false: "links retain zero", true: "positive cap replaces zero"}[capHint], func(t *testing.T) {
			calls := 0
			wantLimit := "0"
			control := ListControl{}
			if capHint {
				wantLimit = "1"
				control = ListControl{MaxItems: 1, LimitHint: true}
			}
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Query().Get("limit") != wantLimit || r.URL.Query().Get("project_id") != "fixed" {
					t.Errorf("query changed: %s", r.URL)
				}
				if calls == 1 {
					writeList(w, `{"items":[{"id":"first"}],"next":"?marker=second"}`)
					return
				}
				if calls != 2 || r.URL.Query().Get("marker") != "second" {
					t.Errorf("unexpected continuation: %s", r.URL)
				}
				writeList(w, `{"items":[{"id":"second"}]}`)
			})
			spec.Paging.AllowZeroLimit = true
			spec.Paging.MarkerFallback = true
			spec.Paging.Marker = func(*listItem) (string, error) {
				t.Error("unused marker was inspected")
				return "", errors.New("unexpected marker")
			}
			rows, err := collectControlledList(context.Background(), spec, url.Values{"limit": {"0"}, "project_id": {"fixed"}}, control)
			want := 2
			if capHint {
				want = 1
			}
			if err != nil || len(rows) != want || calls != want {
				t.Fatal(rows, err, calls, want)
			}
		})
	}
}
