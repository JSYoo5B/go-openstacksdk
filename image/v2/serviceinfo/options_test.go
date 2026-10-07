package serviceinfo

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestServiceInfoOptionsSnapshotReplacementAndRetainedHandles(t *testing.T) {
	paginated := false
	option := WithListStoresOptions(ListStoresOpts{Details: true, Limit: 2, Marker: "before", MaxItems: 3, Paginated: &paginated})
	paginated = true
	var retained *request.Config[ListStoresOpts]
	policy, query, headers, control, err := prepareListStores([]ListStoresOption{
		WithListStoresOptions(ListStoresOpts{Limit: 99}), option,
		func(config *request.Config[ListStoresOpts]) error {
			retained = config
			config.Query.Set("extension", "owned")
			config.Headers["X-Extra"] = "owned"
			return nil
		},
		func(*request.Config[ListStoresOpts]) error {
			*retained.Options.Paginated = true
			retained.Options.Details = false
			retained.Query.Set("extension", "changed")
			retained.Headers["X-Extra"] = "changed"
			return nil
		},
	})
	if err != nil || !policy.Details || policy.Limit != 2 || policy.MaxItems != 3 || policy.Paginated == nil || *policy.Paginated || query.Get("marker") != "before" || query.Get("extension") != "owned" || headers["X-Extra"] != "owned" || !control.SinglePage || control.LimitHint {
		t.Fatalf("policy=%+v query=%v headers=%v control=%+v err=%v", policy, query, headers, control, err)
	}
	retained.Query["extension"][0] = "later"
	if query.Get("extension") != "owned" {
		t.Fatal("prepared query aliases retained callback")
	}
	configA, configB := request.Config[ListStoresOpts]{}, request.Config[ListStoresOpts]{}
	if err := option(&configA); err != nil {
		t.Fatal(err)
	}
	if err := option(&configB); err != nil {
		t.Fatal(err)
	}
	*configA.Options.Paginated = true
	if *configB.Options.Paginated {
		t.Fatal("helper applications share pointer")
	}
}

func TestServiceInfoOptionsRejectUnsupportedInputsBeforeHTTP(t *testing.T) {
	custom := errors.New("option callback cause")
	for _, option := range []ListStoresOption{
		nil, func(*request.Config[ListStoresOpts]) error { return custom },
		WithListStoresOptions(ListStoresOpts{Limit: -1}), WithListStoresMaxItems(-1),
		WithListStoresQuery("DETAILS", "true"), WithListStoresQuery("max_items", "1"), WithListStoresQuery("base_path", "/other"),
		WithListStoresQuery("limit", "0"), WithListStoresQuery("marker", " "),
		WithListStoresQuery("invalid\xff", "value"), WithListStoresQuery("extension", "invalid\xff"),
		WithListStoresHeader("X-Auth-Token", "replacement"), WithListStoresHeader("OpenStack-API-Version", "image 2.8"), WithListStoresHeader("Content-Type", "text/plain"),
		WithListStoresHeader("Bad Name", "value"), WithListStoresHeader("X-Extra", "bad\x00value"),
		request.WithField[ListStoresOpts]("unknown", true), request.WithArgument[ListStoresOpts]("microversion", "2.8"),
	} {
		var requests int
		client := infoClient(func(req *http.Request) (*http.Response, error) {
			requests++
			return infoResponse(req, http.NoBody), nil
		})
		values, err := New(client).AllStores(context.Background(), option)
		if values != nil || err == nil || requests != 0 {
			t.Fatalf("option accepted values=%+v err=%v requests=%d", values, err, requests)
		}
		if !errors.Is(err, custom) && !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("validation or callback cause lost", err)
		}
	}
	for _, option := range []GetImportInfoOption{
		nil, request.WithQuery[GetImportInfoOpts]("unknown", "value"), request.WithField[GetImportInfoOpts]("unknown", true), request.WithArgument[GetImportInfoOpts]("microversion", "2.8"), WithGetImportInfoHeader("Authorization", "replacement"),
	} {
		var requests int
		client := infoClient(func(req *http.Request) (*http.Response, error) {
			requests++
			return infoResponse(req, http.NoBody), nil
		})
		value, err := New(client).GetImportInfo(context.Background(), option)
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || requests != 0 {
			t.Fatalf("value=%+v err=%v requests=%d", value, err, requests)
		}
	}
}

func TestServiceInfoOptionsConcurrentReusableSnapshots(t *testing.T) {
	paginated := false
	options := []ListStoresOption{WithListStoresOptions(ListStoresOpts{Paginated: &paginated}), WithListStoresDetails(true), WithListStoresMaxItems(5), WithListStoresQuery("extension", "owned"), WithListStoresHeader("X-Extra", "owned")}
	var group sync.WaitGroup
	for range 12 {
		group.Add(1)
		go func() {
			defer group.Done()
			for range 10 {
				policy, query, headers, control, err := prepareListStores(options)
				if err != nil || !policy.Details || policy.Paginated == nil || *policy.Paginated || !control.SinglePage || control.MaxItems != 5 || control.LimitHint || query.Get("extension") != "owned" || headers["X-Extra"] != "owned" {
					t.Errorf("reusable options changed: policy=%+v err=%v", policy, err)
					return
				}
				*policy.Paginated = true
				query["extension"][0] = "caller mutation"
				headers["X-Extra"] = "caller mutation"
			}
		}()
	}
	group.Wait()
}
