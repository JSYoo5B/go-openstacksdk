package resource_test

import (
	"context"
	"errors"
	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
	"iter"
	"net/url"
	"testing"
	"time"
)

func TestWaitDeletedUsesStableIDsAndPreservesFailures(t *testing.T) {
	ctx := context.Background()
	gets, lists := 0, 0
	collection := resource.NewCollection(resource.Adapter[entry]{
		Kind: "entry", ID: func(v *entry) string { return v.ID }, Name: func(v *entry) string { return v.Name },
		Iterate: func(ctx context.Context, q url.Values) iter.Seq2[*entry, error] {
			return func(yield func(*entry, error) bool) { lists++; yield(&entry{ID: "fixed", Name: "worker"}, nil) }
		},
		Get: func(ctx context.Context, id string) (*entry, error) {
			gets++
			if id != "fixed" {
				t.Fatal(id)
			}
			if gets > 1 {
				return nil, gophercloud.ErrUnexpectedResponseCode{Actual: 404}
			}
			return &entry{ID: id}, nil
		},
	})
	if err := collection.WaitDeleted(ctx, resource.Name("worker"), resource.WithPollInterval(time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if gets != 2 || lists != 1 {
		t.Fatalf("gets=%d lists=%d", gets, lists)
	}
	if err := collection.WaitDeleted(ctx, resource.ID("fixed")); err != nil {
		t.Fatal(err)
	}
	for _, code := range []int{403, 500} {
		failure := resource.NewCollection(resource.Adapter[entry]{Kind: "entry", Get: func(context.Context, string) (*entry, error) {
			return nil, gophercloud.ErrUnexpectedResponseCode{Actual: code}
		}})
		var response gophercloud.ErrUnexpectedResponseCode
		if err := failure.WaitDeleted(ctx, resource.ID("id")); !errors.As(err, &response) || response.Actual != code {
			t.Fatal(err)
		}
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := collection.WaitDeleted(cancelled, resource.ID("fixed")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
