package resource_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2/pagination"
)

type cycleItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func cycleItems(page pagination.Page) ([]cycleItem, error) {
	var body struct {
		Items []cycleItem `json:"items"`
	}
	err := page.(valuePage).ExtractInto(&body)
	return body.Items, err
}

func TestPaginationCycleStopsBeforeRefetch(t *testing.T) {
	for _, cycle := range []string{"self", "two pages", "reordered query"} {
		for _, consumer := range []string{"Stream", "StreamValues", "Pages", "Collection"} {
			t.Run(cycle+"/"+consumer, func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				firstURL := cloud.Server.URL + "/first?a=1&b=2"
				cloud.Mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if r.Header.Get("X-Page-Header") != "preserved" {
						t.Error("pager headers lost")
					}
					next := firstURL
					if cycle == "two pages" && r.URL.Path == "/first" {
						next = cloud.Server.URL + "/second"
					}
					if cycle == "reordered query" {
						next = cloud.Server.URL + "/first?b=2&a=1#fragment"
					}
					testcloud.JSON(w, 200, fmt.Sprintf(`{"items":[{"id":%q,"name":"same"}],"next":%q}`, r.URL.Path[1:], next))
				})
				pager := pagination.NewPager(cloud.Client("test", "/"), firstURL, func(r pagination.PageResult) pagination.Page {
					return valuePage{pagination.LinkedPageBase{PageResult: r}}
				})
				pager.Headers = map[string]string{"X-Page-Header": "preserved"}
				count, failures := 0, 0
				var cycleErr error
				observe := func(err error) {
					if err != nil {
						failures++
						cycleErr = err
					} else {
						count++
					}
				}
				ctx := context.Background() // Must terminate without a timeout.
				switch consumer {
				case "Stream":
					for _, err := range resource.Stream(ctx, pager, cycleItems) {
						observe(err)
					}
				case "StreamValues":
					for _, err := range resource.StreamValues(ctx, pager, cycleItems) {
						observe(err)
					}
				case "Pages":
					for _, err := range resource.Pages(ctx, pager) {
						observe(err)
					}
				case "Collection":
					collection := resource.NewCollection(resource.Adapter[cycleItem]{Kind: "cycle", List: func(url.Values) pagination.Pager { return pager }, Extract: cycleItems, ID: func(v *cycleItem) string { return v.ID }, Name: func(v *cycleItem) string { return v.Name }})
					for _, err := range collection.List(ctx) {
						observe(err)
					}
					all, err := collection.All(ctx)
					if all != nil || !errors.Is(err, resource.ErrPaginationCycle) {
						t.Fatalf("all=%v err=%v", all, err)
					}
					_, err = collection.Find(ctx, resource.Name("absent"))
					if !errors.Is(err, resource.ErrPaginationCycle) {
						t.Fatalf("Find err=%v", err)
					}
				}
				want := 1
				if cycle == "two pages" {
					want = 2
				}
				if count != want || failures != 1 || !errors.Is(cycleErr, resource.ErrPaginationCycle) {
					t.Fatalf("count=%d failures=%d err=%v", count, failures, cycleErr)
				}
				var details *resource.PaginationCycleError
				if !errors.As(cycleErr, &details) || details.URL == "" {
					t.Fatalf("missing details: %v", cycleErr)
				}
				iterations := 1
				if consumer == "Collection" {
					iterations = 3
				}
				if requests.Load() != int32(want*iterations) {
					t.Fatalf("requests=%d want=%d", requests.Load(), want*iterations)
				}
			})
		}
	}
}

func TestBreakingStreamDoesNotEvaluateNextLink(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /first", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"items":[{"id":"one"}]}`) })
	var nextCalls atomic.Int32
	pager := pagination.NewPager(cloud.Client("test", "/"), cloud.Server.URL+"/first", func(r pagination.PageResult) pagination.Page {
		return failingNextPage{PageResult: r, calls: &nextCalls}
	})
	for _, err := range resource.Pages(context.Background(), pager) {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if nextCalls.Load() != 0 {
		t.Fatalf("next calls=%d", nextCalls.Load())
	}
}

type failingNextPage struct {
	pagination.PageResult
	calls *atomic.Int32
}

func (p failingNextPage) NextPageURL() (string, error) {
	p.calls.Add(1)
	return "", errors.New("invalid next link")
}
func (p failingNextPage) IsEmpty() (bool, error) { return false, nil }
func (p failingNextPage) GetBody() any           { return p.Body }
