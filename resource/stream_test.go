package resource_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2/pagination"
)

type valuePage struct{ pagination.LinkedPageBase }

func (p valuePage) NextPageURL() (string, error) {
	next, _ := p.Body.(map[string]any)["next"].(string)
	return next, nil
}
func (p valuePage) IsEmpty() (bool, error) { return false, nil }

func TestTypedPageValuesAndEarlyStop(t *testing.T) {
	cloud := testcloud.New(t)
	var nextCalls atomic.Int32
	cloud.Mux.HandleFunc("GET /values", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, fmt.Sprintf(`{"values":{"one":false},"next":%q}`, cloud.Server.URL+"/next"))
	})
	cloud.Mux.HandleFunc("GET /next", func(w http.ResponseWriter, r *http.Request) {
		nextCalls.Add(1)
		testcloud.JSON(w, 200, `{"values":{"two":true}}`)
	})
	client := cloud.Client("test", "/")
	pager := func() pagination.Pager {
		return pagination.NewPager(client, cloud.Server.URL+"/values", func(r pagination.PageResult) pagination.Page {
			return valuePage{pagination.LinkedPageBase{PageResult: r}}
		})
	}
	extract := func(page pagination.Page) (map[string]any, error) {
		p := page.(valuePage)
		v := p.Body.(map[string]any)["values"].(map[string]any)
		return v, nil
	}
	for _, err := range resource.StreamValues(context.Background(), pager(), extract) {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if nextCalls.Load() != 0 {
		t.Fatal("pagination continued after break")
	}
	count := 0
	for value, err := range resource.StreamValues(context.Background(), pager(), extract) {
		if err != nil {
			t.Fatal(err)
		}
		if len(value) != 1 {
			t.Fatal(value)
		}
		count++
	}
	if count != 2 || nextCalls.Load() != 1 {
		t.Fatalf("pages=%d next=%d", count, nextCalls.Load())
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	count = 0
	for value, err := range resource.StreamValues(canceled, pager(), extract) {
		count++
		if value != nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("value=%v err=%v", value, err)
		}
	}
	if count != 1 {
		t.Fatalf("error yields=%d", count)
	}
}
