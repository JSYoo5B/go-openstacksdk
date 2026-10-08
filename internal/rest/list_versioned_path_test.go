package rest

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestListVersionedPathAliasRequiresExactRelativeCollection(t *testing.T) {
	for _, next := range []string{"/v2/items?marker=second", "https://foreign.test/v2/items?marker=second", "/v2/other?marker=second", "/v2/%69tems?marker=second"} {
		t.Run(next, func(t *testing.T) {
			calls := 0
			spec := listSpec(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls == 1 {
					writeList(w, `{"items":[{"id":"first"}],"next":"`+next+`"}`)
					return
				}
				if calls != 2 || r.URL.Query().Get("marker") != "second" {
					t.Errorf("wrong next: %s", r.URL)
				}
				writeList(w, `{"items":[]}`)
			})
			spec.Paging.VersionedPath = "/v2/items"
			rows, err := collectList(context.Background(), spec, nil)
			if next == "/v2/items?marker=second" {
				if err != nil || len(rows) != 1 || calls != 2 {
					t.Fatal(rows, err, calls)
				}
			} else {
				var proof *resource.ResponseError
				if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || len(rows) != 1 || calls != 1 {
					t.Fatal(rows, err, calls)
				}
			}
		})
	}
}
