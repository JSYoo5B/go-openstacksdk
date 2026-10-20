package resource_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// IdentityName replaces Name for list matching only; nil Name stays an
// unsupported explicit Ref name lookup.
func TestCollectionFindIdentityNameOverrideMatchesAlternateFieldOrIDOnly(t *testing.T) {
	for _, mode := range []string{"alternate", "id-only"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/v1/items/{id}", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
			cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.RawQuery != "" {
					t.Error("name hint added without NameQuery", r.URL)
				}
				testcloud.JSON(w, 200, identityFindBody(`{"id":"first","name":"203.0.113.5"},{"id":"second","name":"other"}`, ""))
			})
			adapter := identityFindAdapter(cloud)
			adapter.Name, adapter.NameQuery = nil, nil
			adapter.IdentityName = func(value *identityFindItem) string { return value.Name }
			if mode == "id-only" {
				adapter.IdentityName = func(*identityFindItem) string { return "" }
			}
			collection := resource.NewCollection(adapter)
			value, err := collection.FindIdentity(context.Background(), "203.0.113.5")
			if mode == "alternate" && (err != nil || value == nil || value.ID != "first") || mode == "id-only" && (err != nil || value != nil) {
				t.Fatal(value, err)
			}
			if value, err := collection.FindIdentity(context.Background(), "second"); err != nil || value == nil || value.ID != "second" {
				t.Fatal("ID match lost", value, err)
			}
			if _, err := collection.Find(context.Background(), resource.Name("203.0.113.5")); !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal("explicit name lookup gained IdentityName", err)
			}
			if lists.Load() != 2 {
				t.Fatal(lists.Load())
			}
		})
	}
	cloud := testcloud.New(t)
	adapter := identityFindAdapter(cloud)
	adapter.Name = nil
	if _, err := resource.NewCollection(adapter).FindIdentity(context.Background(), "x"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal("binding without any name policy was accepted", err)
	}
}

// IdentityLiteralGet sends the original identity through the query getter
// even without query, so numeric native getters do not reject names first.
func TestCollectionFindIdentityLiteralGetUsesQueryHookWithoutQuery(t *testing.T) {
	cloud := testcloud.New(t)
	var hooks, native atomic.Int32
	var mu sync.Mutex
	var paths []string
	record := func(r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		paths = append(paths, r.URL.Path+"?"+r.URL.RawQuery)
	}
	cloud.Mux.HandleFunc("GET /reverse/v1/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		record(r)
		if r.PathValue("id") == "7" {
			testcloud.JSON(w, 200, `{"item":{"id":"7","name":"seven"}}`)
			return
		}
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) {
		record(r)
		testcloud.JSON(w, 200, identityFindBody(`{"id":"7","name":"seven"}`, ""))
	})
	adapter := identityFindAdapter(cloud)
	query := adapter.GetIdentityQuery
	adapter.GetIdentityQuery = func(ctx context.Context, id string, q url.Values) (*identityFindItem, error) {
		hooks.Add(1)
		return query(ctx, id, q)
	}
	adapter.Get = func(context.Context, string) (*identityFindItem, error) {
		native.Add(1)
		return nil, fmt.Errorf("numeric getter rejected the identity: %w", resource.ErrInvalidOption)
	}
	adapter.IdentityLiteralGet = true
	collection := resource.NewCollection(adapter)
	for _, identity := range []string{"7", "seven"} {
		value, err := collection.FindIdentity(context.Background(), identity)
		if err != nil || value == nil || value.ID != "7" {
			t.Fatal(identity, value, err)
		}
	}
	want := []string{"/reverse/v1/items/7?", "/reverse/v1/items/seven?", "/reverse/v1/items?name=%5Eseven%24"}
	mu.Lock()
	defer mu.Unlock()
	if fmt.Sprint(paths) != fmt.Sprint(want) || hooks.Load() != 2 || native.Load() != 0 {
		t.Fatal(paths, hooks.Load(), native.Load())
	}
	adapter.GetIdentityQuery = nil
	if _, err := resource.NewCollection(adapter).FindIdentity(context.Background(), "7"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal("literal GET without query hook was accepted", err)
	}
}
