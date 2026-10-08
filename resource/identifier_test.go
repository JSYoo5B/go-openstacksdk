package resource_test

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/url"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestBindingIdentifierPolicyAppliesToLookupResolveDeleteAndWait(t *testing.T) {
	type model struct{ id, name, status string }
	const key = "folder/my file?#%.txt"
	ctx := context.Background()
	gets, deleted := 0, false
	collection := resource.NewCollection(resource.Adapter[model]{
		Kind: "opaque key",
		ValidateID: func(id string) error {
			if id == "" || id == "forbidden" {
				return fmt.Errorf("%w: invalid opaque key", resource.ErrInvalidOption)
			}
			return nil
		},
		Get: func(_ context.Context, id string) (*model, error) {
			gets++
			if id != key {
				t.Errorf("get key=%q", id)
			}
			if deleted {
				return nil, gophercloud.ErrUnexpectedResponseCode{Actual: 404}
			}
			return &model{id: key, name: "artifact", status: "ready"}, nil
		},
		Iterate: func(_ context.Context, query url.Values) iter.Seq2[*model, error] {
			if query.Get("prefix") != "artifact" || query.Has("name") {
				t.Errorf("name query=%v", query)
			}
			return func(yield func(*model, error) bool) { yield(&model{id: key, name: "artifact", status: "pending"}, nil) }
		},
		Delete: func(_ context.Context, id string) error {
			if id != key {
				t.Errorf("delete key=%q", id)
			}
			deleted = true
			return nil
		},
		ID: func(item *model) string { return item.id }, Name: func(item *model) string { return item.name },
		Status:       func(item *model) string { return item.status },
		NameQueryKey: "prefix", NameQuery: func(name string) string { return name },
	})
	if value, err := collection.Get(ctx, key); err != nil || value.id != key {
		t.Fatalf("get=%v err=%v", value, err)
	}
	if value, err := collection.Find(ctx, resource.ID(key)); err != nil || value.id != key {
		t.Fatalf("find=%v err=%v", value, err)
	}
	before := gets
	for _, ref := range []resource.Ref{resource.ID(key), resource.Name("artifact")} {
		if id, err := collection.ResolveID(ctx, ref); err != nil || id != key {
			t.Fatalf("resolved=%q err=%v", id, err)
		}
	}
	if gets != before {
		t.Fatal("resolution made a get request")
	}
	if value, err := collection.Wait(ctx, resource.Name("artifact"), "ready", resource.WithPollInterval(time.Millisecond)); err != nil || value.id != key {
		t.Fatalf("wait=%v err=%v", value, err)
	}
	if err := collection.Delete(ctx, resource.ID(key)); err != nil || !deleted {
		t.Fatalf("delete=%v deleted=%v", err, deleted)
	}
	if err := collection.WaitDeleted(ctx, resource.ID(key)); err != nil {
		t.Fatal(err)
	}
	before = gets
	if _, err := collection.Get(ctx, "forbidden"); !errors.Is(err, resource.ErrInvalidOption) || gets != before {
		t.Fatalf("err=%v gets=%d", err, gets)
	}
	ordinary := resource.NewCollection(resource.Adapter[model]{Get: func(_ context.Context, _ string) (*model, error) {
		t.Fatal("ordinary ID policy bypassed")
		return nil, nil
	}})
	if _, err := ordinary.Get(ctx, key); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
