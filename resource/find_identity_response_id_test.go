package resource_test

import (
	"context"
	"errors"
	"iter"
	"net/url"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type passiveIdentityValue struct{ id, name string }

func TestPrepareIdentityFindOptionsFreezesCustomAndBulkValues(t *testing.T) {
	missing, details, projects, specs := false, true, false, true
	query := url.Values{"name": {"", "second"}}
	bulk := resource.WithIdentityFindOptions(resource.IdentityFindOpts{IgnoreMissing: &missing, Details: &details, AllProjects: &projects, GetExtraSpecs: &specs, Query: query})
	missing, details, projects, specs = true, false, true, false
	query["name"][0] = "changed"
	var retained *resource.IdentityFindOpts
	prepared, err := resource.PrepareIdentityFindOptions(bulk, func(value *resource.IdentityFindOpts) error {
		retained = value
		value.Fallback = resource.FindFallbackNever
		return nil
	})
	if err != nil || *prepared.IgnoreMissing || !*prepared.Details || *prepared.AllProjects || !*prepared.GetExtraSpecs || prepared.Fallback != resource.FindFallbackNever || prepared.Query["name"][0] != "" || prepared.Query["name"][1] != "second" {
		t.Fatal(prepared, err)
	}
	*retained.IgnoreMissing = true
	*retained.Details = false
	*retained.AllProjects = true
	*retained.GetExtraSpecs = false
	retained.Query["name"][0] = "retained mutation"
	if *prepared.IgnoreMissing || !*prepared.Details || *prepared.AllProjects || !*prepared.GetExtraSpecs || prepared.Query["name"][0] != "" {
		t.Fatal("prepared options alias a retained callback", prepared)
	}
	prepared.Query["name"][0] = "local"
	again, err := resource.PrepareIdentityFindOptions(bulk)
	if err != nil || again.Query["name"][0] != "" || *again.IgnoreMissing {
		t.Fatal("reusing bulk options aliases previous preparation", again, err)
	}
}

func TestPrepareIdentityFindOptionsRejectsInvalidFinalConfiguration(t *testing.T) {
	for _, options := range [][]resource.IdentityFindOption{
		{nil},
		{resource.WithIdentityFindFallback(resource.FindFallbackPolicy(-1))},
		{resource.WithIdentityFindFallback(resource.FindFallbackPolicy(100))},
		{func(value *resource.IdentityFindOpts) error {
			value.Query = url.Values{"headers": {"value"}}
			return nil
		}},
		{func(value *resource.IdentityFindOpts) error {
			value.Query = url.Values{"bad\nkey": {"value"}}
			return nil
		}},
	} {
		if _, err := resource.PrepareIdentityFindOptions(options...); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	cause := errors.New("custom option failure")
	if _, err := resource.PrepareIdentityFindOptions(func(*resource.IdentityFindOpts) error { return cause }); !errors.Is(err, cause) {
		t.Fatal(err)
	}
	value, err := resource.PrepareIdentityFindOptions(resource.WithIdentityFindFallback(resource.FindFallbackPolicy(100)), resource.WithIdentityFindOptions(resource.IdentityFindOpts{}))
	if err != nil || value.IgnoreMissing != nil || value.Fallback != resource.FindFallbackCompatible || len(value.Query) != 0 {
		t.Fatal("later bulk replacement must determine the final configuration", value, err)
	}
}

func passiveIdentityAdapter(rows []*passiveIdentityValue) resource.Adapter[passiveIdentityValue] {
	return resource.Adapter[passiveIdentityValue]{
		Kind:         "passive_identity",
		IdentityFind: true,
		Get: func(context.Context, string) (*passiveIdentityValue, error) {
			return nil, gophercloud.ErrUnexpectedResponseCode{Actual: 404}
		},
		Iterate: func(context.Context, url.Values) iter.Seq2[*passiveIdentityValue, error] {
			return func(yield func(*passiveIdentityValue, error) bool) {
				for _, row := range rows {
					if !yield(row, nil) {
						return
					}
				}
			}
		},
		ID:                 func(value *passiveIdentityValue) string { return value.id },
		Name:               func(value *passiveIdentityValue) string { return value.name },
		NameQuery:          func(identity string) string { return identity },
		IdentityResponseID: func(value *passiveIdentityValue) (string, error) { return value.id, nil },
	}
}

func TestCollectionFindIdentityPassiveIDsAreOnlyCompared(t *testing.T) {
	full := "https://different.test/secrets/shared-component"
	opaque := "one key/%?#"
	rows := []*passiveIdentityValue{{id: "https://first.test/secrets/shared-component", name: "other"}, {id: full, name: "chosen"}, {id: "", name: "anonymous"}, {name: opaque}}
	for _, test := range []struct {
		identity string
		index    int
		gets     int
		escaped  bool
	}{
		{full, 1, 0, false}, {"chosen", 1, 1, false}, {"anonymous", 2, 1, false},
		{opaque, 3, 0, false}, {opaque, 3, 1, true},
	} {
		adapter := passiveIdentityAdapter(rows)
		if test.escaped {
			adapter.IdentityDirectGet = func(string) error { return nil }
			adapter.ValidateID = func(string) error { return nil }
		}
		var gets, lists, comparisons int
		adapter.Get = func(context.Context, string) (*passiveIdentityValue, error) {
			gets++
			return nil, gophercloud.ErrUnexpectedResponseCode{Actual: 404}
		}
		original := adapter.Iterate
		adapter.Iterate = func(ctx context.Context, query url.Values) iter.Seq2[*passiveIdentityValue, error] {
			lists++
			if query.Get("name") != test.identity {
				t.Fatal(query)
			}
			return original(ctx, query)
		}
		adapter.IdentityResponseID = func(value *passiveIdentityValue) (string, error) { comparisons++; return value.id, nil }
		value, err := resource.NewCollection(adapter).FindIdentity(context.Background(), test.identity)
		if err != nil || value != rows[test.index] || gets != test.gets || lists != 1 || comparisons != len(rows) {
			t.Fatal(test.identity, value, err, gets, lists, comparisons)
		}
	}
	// Opt-in comparison permits a direct response with no addressable identity.
	adapter := passiveIdentityAdapter(nil)
	adapter.ID = nil
	returned := &passiveIdentityValue{name: "actual"}
	adapter.Get = func(context.Context, string) (*passiveIdentityValue, error) { return returned, nil }
	if value, err := resource.NewCollection(adapter).FindIdentity(context.Background(), "requested"); err != nil || value != returned {
		t.Fatal(value, err)
	}
	// An audited route validator can reject before either GET or list fallback.
	cause := errors.New("escaped member validation failed")
	adapter.IdentityDirectGet = func(string) error { return cause }
	adapter.Get = func(context.Context, string) (*passiveIdentityValue, error) {
		t.Fatal("rejected member reached GET")
		return nil, nil
	}
	adapter.Iterate = func(context.Context, url.Values) iter.Seq2[*passiveIdentityValue, error] {
		t.Fatal("rejected member reached list")
		return nil
	}
	if value, err := resource.NewCollection(adapter).FindIdentity(context.Background(), "requested"); value != nil || !errors.Is(err, cause) {
		t.Fatal(value, err)
	}
}

func TestCollectionFindIdentityPassiveIDsKeepLateFailuresAndDuplicates(t *testing.T) {
	matched := &passiveIdentityValue{name: "target"}
	for _, mode := range []string{"duplicate", "late error", "callback error", "nil direct", "nil row"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New("observed failure")
			adapter := passiveIdentityAdapter([]*passiveIdentityValue{matched})
			var callbacks int
			adapter.IdentityResponseID = func(value *passiveIdentityValue) (string, error) {
				callbacks++
				if mode == "callback error" {
					return "", cause
				}
				return value.id, nil
			}
			switch mode {
			case "duplicate":
				adapter = passiveIdentityAdapter([]*passiveIdentityValue{matched, {name: "target"}})
			case "late error":
				adapter.Iterate = func(context.Context, url.Values) iter.Seq2[*passiveIdentityValue, error] {
					return func(yield func(*passiveIdentityValue, error) bool) {
						if yield(matched, nil) {
							yield(nil, cause)
						}
					}
				}
			case "nil direct":
				adapter.Get = func(context.Context, string) (*passiveIdentityValue, error) { return nil, nil }
			case "nil row":
				adapter.Iterate = func(context.Context, url.Values) iter.Seq2[*passiveIdentityValue, error] {
					return func(yield func(*passiveIdentityValue, error) bool) { yield(nil, nil) }
				}
			}
			value, err := resource.NewCollection(adapter).FindIdentity(context.Background(), "target")
			if value != nil || err == nil {
				t.Fatal(value, err)
			}
			switch mode {
			case "duplicate":
				if !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(err)
				}
			case "late error", "callback error":
				if !errors.Is(err, cause) {
					t.Fatal(err)
				}
			case "nil direct", "nil row":
				if !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
					t.Fatal(err, callbacks)
				}
			}
		})
	}
}

func TestCollectionFindIdentityPassiveIDsLeaveRouteGuardsAndDefaultBindings(t *testing.T) {
	for _, id := range []string{"", "https://passive.test/secrets/one"} {
		adapter := passiveIdentityAdapter(nil)
		adapter.IdentityResponseID = nil
		adapter.Get = func(context.Context, string) (*passiveIdentityValue, error) {
			return &passiveIdentityValue{id: id}, nil
		}
		if value, err := resource.NewCollection(adapter).FindIdentity(context.Background(), "requested"); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("existing response-ID policy changed", value, err)
		}
	}
	adapter := passiveIdentityAdapter(nil)
	var calls int
	adapter.Get = func(context.Context, string) (*passiveIdentityValue, error) {
		calls++
		return &passiveIdentityValue{}, nil
	}
	adapter.Delete = func(context.Context, string) error { calls++; return nil }
	adapter.IdentityResponseID = func(value *passiveIdentityValue) (string, error) { calls++; return value.id, nil }
	collection := resource.NewCollection(adapter)
	for _, id := range []string{"a/b", "https://passive.test/secrets/one"} {
		if _, err := collection.Get(context.Background(), id); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
		if err := collection.Delete(context.Background(), resource.ID(id)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if calls != 0 {
		t.Fatal("a passive response callback weakened request routing", calls)
	}
}

func TestCollectionFindIdentityPassiveIDCancellationIsTerminal(t *testing.T) {
	for _, direct := range []bool{true, false} {
		ctx, cancel := context.WithCancel(context.Background())
		adapter := passiveIdentityAdapter([]*passiveIdentityValue{{name: "target"}})
		if direct {
			adapter.Get = func(context.Context, string) (*passiveIdentityValue, error) { return &passiveIdentityValue{}, nil }
		}
		adapter.IdentityResponseID = func(*passiveIdentityValue) (string, error) { cancel(); return "", nil }
		value, err := resource.NewCollection(adapter).FindIdentity(ctx, "target")
		if value != nil || !errors.Is(err, context.Canceled) {
			t.Fatal(value, err)
		}
	}
}
