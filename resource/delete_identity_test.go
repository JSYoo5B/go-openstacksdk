package resource_test

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/url"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type deleteIdentityItem struct {
	ID, Name string
}

func TestDeleteNameValidatesResolvedIdentityBeforeMutation(t *testing.T) {
	for _, id := range []string{"", " ", "database-id", "fixed-uuid"} {
		t.Run(fmt.Sprintf("id=%q", id), func(t *testing.T) {
			lists, deletes := 0, 0
			collection := resource.NewCollection(resource.Adapter[deleteIdentityItem]{
				Kind: "items", ID: func(item *deleteIdentityItem) string { return item.ID },
				Name: func(item *deleteIdentityItem) string { return item.Name },
				ValidateID: func(value string) error {
					if value != "fixed-uuid" {
						return fmt.Errorf("%w: item route requires its UUID", resource.ErrInvalidOption)
					}
					return nil
				},
				Iterate: func(context.Context, url.Values) iter.Seq2[*deleteIdentityItem, error] {
					return func(yield func(*deleteIdentityItem, error) bool) {
						lists++
						yield(&deleteIdentityItem{ID: id, Name: "match"}, nil)
					}
				},
				Delete: func(_ context.Context, value string) error {
					deletes++
					if value != "fixed-uuid" {
						t.Errorf("mutation received an invalid response identity: %q", value)
					}
					return nil
				},
			})
			err := collection.Delete(context.Background(), resource.Name("match"))
			if id == "fixed-uuid" {
				if err != nil || deletes != 1 {
					t.Fatalf("valid route rejected: deletes=%d err=%v", deletes, err)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) || deletes != 0 {
				t.Fatalf("invalid route mutated: deletes=%d err=%v", deletes, err)
			}
			if lists != 1 {
				t.Fatalf("name resolved %d times", lists)
			}
		})
	}
}
