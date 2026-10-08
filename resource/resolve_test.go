package resource_test

import (
	"context"
	"errors"
	"iter"
	"net/url"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestResolveIDUsesExactNamesAndNeverFetchesExplicitIDs(t *testing.T) {
	calls := 0
	items := []entry{{ID: "fixed", Name: "worker"}, {ID: "other", Name: "worker-suffix"}}
	var listError error
	collection := resource.NewCollection(resource.Adapter[entry]{
		Kind: "parent", ID: func(v *entry) string { return v.ID }, Name: func(v *entry) string { return v.Name },
		Iterate: func(ctx context.Context, q url.Values) iter.Seq2[*entry, error] {
			return func(yield func(*entry, error) bool) {
				calls++
				if listError != nil {
					yield(nil, listError)
					return
				}
				for i := range items {
					if !yield(&items[i], nil) {
						return
					}
				}
			}
		},
	})
	ctx := context.Background()
	if id, err := collection.ResolveID(ctx, resource.ID("worker")); err != nil || id != "worker" || calls != 0 {
		t.Fatalf("id=%q calls=%d err=%v", id, calls, err)
	}
	if id, err := collection.ResolveID(ctx, resource.Name("worker")); err != nil || id != "fixed" || calls != 1 {
		t.Fatalf("id=%q calls=%d err=%v", id, calls, err)
	}
	if _, err := collection.ResolveID(ctx, resource.Name("missing")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	items[1].Name = "worker"
	if _, err := collection.ResolveID(ctx, resource.Name("worker")); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatal(err)
	}
	items = []entry{{ID: "..", Name: "worker"}}
	if _, err := collection.ResolveID(ctx, resource.Name("worker")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	listError = errors.New("permission denied")
	if _, err := collection.ResolveID(ctx, resource.Name("worker")); !errors.Is(err, listError) {
		t.Fatal(err)
	}
	before := calls
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := collection.ResolveID(canceled, resource.ID("fixed")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	for _, id := range []string{"", ".", "..", "parent/child", "parent?query=1"} {
		if _, err := collection.ResolveID(ctx, resource.ID(id)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if calls != before {
		t.Fatalf("invalid references performed requests: %d -> %d", before, calls)
	}
}
