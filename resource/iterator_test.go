package resource_test

import (
	"context"
	"errors"
	"iter"
	"net/url"
	"testing"

	"gophercloudsdk/resource"
)

type entry struct{ ID, Name, Status string }

func TestCollectionAcceptsTypedIteratorsAndKeepsPolicies(t *testing.T) {
	yielded := 0
	collection := resource.NewCollection(resource.Adapter[entry]{
		Kind: "entry", ID: func(v *entry) string { return v.ID }, Name: func(v *entry) string { return v.Name }, Status: func(v *entry) string { return v.Status },
		Iterate: func(ctx context.Context, q url.Values) iter.Seq2[*entry, error] {
			return func(yield func(*entry, error) bool) {
				for _, v := range []entry{{"a", "worker", "BUILD"}, {"b", "worker", "ACTIVE"}, {"c", "other", "ACTIVE"}} {
					yielded++
					if !yield(&v, nil) {
						return
					}
				}
			}
		},
	})
	for v, err := range collection.List(context.Background(), resource.WithName("worker"), resource.WithStatus("active")) {
		if err != nil || v.ID != "b" {
			t.Fatalf("value=%v err=%v", v, err)
		}
		break
	}
	if yielded != 2 {
		t.Fatalf("yielded=%d", yielded)
	}
	if _, err := collection.Find(context.Background(), resource.Name("worker")); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatal(err)
	}
	if _, err := collection.Get(context.Background(), "id"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
}
