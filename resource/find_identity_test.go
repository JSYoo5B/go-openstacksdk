package resource_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type identityFindItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type identityFindPage struct{ pagination.LinkedPageBase }

func (page identityFindPage) IsEmpty() (bool, error) {
	values, err := identityFindItems(page)
	return len(values) == 0, err
}

func identityFindItems(page pagination.Page) ([]identityFindItem, error) {
	var body struct {
		Items []identityFindItem `json:"items"`
	}
	err := page.(identityFindPage).ExtractInto(&body)
	return body.Items, err
}

func identityFindAdapter(cloud *testcloud.Cloud) resource.Adapter[identityFindItem] {
	client := cloud.Client("test", "/reverse/v1")
	get := func(ctx context.Context, id string, query url.Values) (*identityFindItem, error) {
		var body struct {
			Item identityFindItem `json:"item"`
		}
		target := client.ServiceURL("items", id)
		if encoded := query.Encode(); encoded != "" {
			target += "?" + encoded
		}
		_, err := client.Get(ctx, target, &body, &gophercloud.RequestOpts{OkCodes: []int{200, 203}})
		return &body.Item, err
	}
	return resource.Adapter[identityFindItem]{
		Kind: "identity-items", IdentityFind: true,
		Get: func(ctx context.Context, id string) (*identityFindItem, error) {
			return get(ctx, id, nil)
		},
		GetIdentityQuery: get,
		List: func(query url.Values) pagination.Pager {
			return pagination.NewPager(client, client.ServiceURL("items")+"?"+query.Encode(), func(result pagination.PageResult) pagination.Page {
				return identityFindPage{pagination.LinkedPageBase{PageResult: result}}
			})
		},
		Extract: identityFindItems,
		ID:      func(value *identityFindItem) string { return value.ID },
		Name:    func(value *identityFindItem) string { return value.Name },
		NameQuery: func(name string) string {
			return "^" + regexp.QuoteMeta(name) + "$"
		},
	}
}

func identityFindBody(rows, next string) string {
	return fmt.Sprintf(`{"items":[%s],"links":{"next":%q}}`, rows, next)
}

func TestCollectionFindIdentitySafeGETKeepsNativeSuccessAndErrors(t *testing.T) {
	for _, status := range []int{200, 203, 201, 401, 409, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, lists atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/v1/items/short-id", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Error(r.URL, r.Header)
				}
				w.Header().Set("X-Evidence", "native")
				testcloud.JSON(w, status, `{"item":{"id":"canonical","name":"different"}}`)
			})
			cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) { lists.Add(1) })
			collection := resource.NewCollection(identityFindAdapter(cloud))
			value, err := collection.FindIdentity(context.Background(), "short-id")
			if status == 200 || status == 203 {
				if err != nil || value == nil || value.ID != "canonical" {
					t.Fatal(value, err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.As(err, &native) || native.Actual != status || native.ResponseHeader.Get("X-Evidence") != "native" || len(native.Body) == 0 {
					t.Fatal(value, err, native)
				}
			}
			if gets.Load() != 1 || lists.Load() != 0 {
				t.Fatal(gets.Load(), lists.Load())
			}
		})
	}
}

func TestCollectionFindIdentityFallbackPoliciesAndLogicalMissing(t *testing.T) {
	for _, status := range []int{400, 403, 404} {
		for _, policy := range []resource.FindFallbackPolicy{resource.FindFallbackCompatible, resource.FindFallbackNotFoundOnly, resource.FindFallbackNever} {
			t.Run(fmt.Sprint(status, "/", policy), func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets, lists atomic.Int32
				cloud.Mux.HandleFunc("GET /reverse/v1/items/identity", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					testcloud.JSON(w, status, `{"error":"native"}`)
				})
				cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					if r.URL.Query().Get("name") != "^identity$" {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, identityFindBody(`{"id":"identity","name":"another-name"}`, ""))
				})
				collection := resource.NewCollection(identityFindAdapter(cloud))
				value, err := collection.FindIdentity(context.Background(), "identity", resource.WithIdentityFindFallback(policy))
				fallback := policy == resource.FindFallbackCompatible || (policy == resource.FindFallbackNotFoundOnly && status == 404)
				if fallback {
					if err != nil || value == nil || value.ID != "identity" || lists.Load() != 1 {
						t.Fatal(value, err, lists.Load())
					}
				} else if status == 404 {
					if value != nil || err != nil || lists.Load() != 0 {
						t.Fatal(value, err, lists.Load())
					}
					_, err = collection.FindIdentity(context.Background(), "identity", resource.WithIdentityFindFallback(policy), resource.WithIdentityFindIgnoreMissing(false))
					if !errors.Is(err, resource.ErrNotFound) || !gophercloud.ResponseCodeIs(err, 404) {
						t.Fatal(err)
					}
				} else if value != nil || !gophercloud.ResponseCodeIs(err, status) || lists.Load() != 0 {
					t.Fatal(value, err, lists.Load())
				}
			})
		}
	}
	for _, listStatus := range []int{200, 403, 404} {
		t.Run(fmt.Sprint("empty-or-error/", listStatus), func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("GET /reverse/v1/items/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 403, `{}`) })
			cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, listStatus, identityFindBody("", "")) })
			collection := resource.NewCollection(identityFindAdapter(cloud))
			for _, ignore := range []bool{true, false} {
				value, err := collection.FindIdentity(context.Background(), "missing", resource.WithIdentityFindIgnoreMissing(ignore))
				if listStatus != 200 {
					if value != nil || !gophercloud.ResponseCodeIs(err, listStatus) {
						t.Fatal(value, err)
					}
				} else if ignore {
					if value != nil || err != nil {
						t.Fatal(value, err)
					}
				} else {
					var missing *resource.NotFoundError
					if value != nil || !errors.As(err, &missing) || missing.Cause != nil || gophercloud.ResponseCodeIs(err, 403) {
						t.Fatal(value, err, missing)
					}
				}
			}
		})
	}
}

func TestCollectionFindIdentityUnsafeLiteralNamesNeverUseIDRoutes(t *testing.T) {
	for _, name := range []string{"my vm", "folder/item", "100%", "%2F", "literal?x=1#part", ".", "..", "\u00a0vm\u00a0", "한글 이름", "https://foreign.invalid/path"} {
		t.Run(name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, other atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("name") != "^"+regexp.QuoteMeta(name)+"$" {
					t.Error(r.URL)
				}
				row, _ := json.Marshal(identityFindItem{ID: "canonical", Name: name})
				testcloud.JSON(w, 200, identityFindBody(string(row), ""))
			})
			cloud.Mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) { other.Add(1) })
			collection := resource.NewCollection(identityFindAdapter(cloud))
			for _, policy := range []resource.FindFallbackPolicy{resource.FindFallbackCompatible, resource.FindFallbackNotFoundOnly} {
				value, err := collection.FindIdentity(context.Background(), name, resource.WithIdentityFindFallback(policy))
				if err != nil || value == nil || value.Name != name {
					t.Fatal(value, err)
				}
			}
			value, err := collection.FindIdentity(context.Background(), name, resource.WithIdentityFindFallback(resource.FindFallbackNever))
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || lists.Load() != 2 || other.Load() != 0 {
				t.Fatal(value, err, lists.Load(), other.Load())
			}
		})
	}
}

func TestCollectionFindIdentitySnapshotsBulkAndCapturedCustomOptions(t *testing.T) {
	for _, custom := range []bool{false, true} {
		t.Run(fmt.Sprint("custom=", custom), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			ignored := false
			query := url.Values{"vendor": {"one", "two"}, "name": nil}
			original := resource.IdentityFindOpts{IgnoreMissing: &ignored, Query: query}
			var captured *resource.IdentityFindOpts
			option := resource.WithIdentityFindOptions(original)
			if custom {
				option = func(config *resource.IdentityFindOpts) error { *config = original; captured = config; return nil }
			} else {
				ignored = true
				query["vendor"][0] = "before"
				query["name"] = []string{"wrong"}
			}
			cloud.Mux.HandleFunc("GET /reverse/v1/items/input", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !reflect.DeepEqual(r.URL.Query()["vendor"], []string{"one", "two"}) || r.URL.Query().Has("name") {
					t.Error("direct GET lost the frozen query", r.URL)
				}
				if custom {
					*captured.IgnoreMissing = true
					captured.Fallback = resource.FindFallbackNever
					captured.Query["vendor"][0] = "during"
					captured.Query["name"] = []string{"wrong"}
				}
				testcloud.JSON(w, 404, `{}`)
			})
			cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !reflect.DeepEqual(r.URL.Query()["vendor"], []string{"one", "two"}) || r.URL.Query().Has("name") {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, identityFindBody("", ""))
			})
			collection := resource.NewCollection(identityFindAdapter(cloud))
			value, err := collection.FindIdentity(context.Background(), "input", option)
			if value != nil || !errors.Is(err, resource.ErrNotFound) || calls.Load() != 2 {
				t.Fatal(value, err, calls.Load())
			}
			if !custom {
				value, err = collection.FindIdentity(context.Background(), "input", option, resource.WithIdentityFindIgnoreMissing(true))
				if value != nil || err != nil || calls.Load() != 4 {
					t.Fatal("reused option or last-wins changed", value, err, calls.Load())
				}
			}
		})
	}
}

func TestCollectionFindIdentityCallerQueryPrecedenceAndBulkReset(t *testing.T) {
	cloud := testcloud.New(t)
	var mode atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/v1/items/input", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if mode.Load() == 0 {
			if !query.Has("name") || query.Get("name") != "" || query.Get("vendor") != "a&b" {
				t.Error("direct GET query", query)
			}
		} else if r.URL.RawQuery != "" {
			t.Error("bulk reset added a list-only name hint to GET", r.URL)
		}
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if mode.Load() == 0 {
			if !query.Has("name") || query.Get("name") != "" || query.Get("vendor") != "a&b" {
				t.Error(query)
			}
		} else if query.Get("name") != "^input$" || query.Has("vendor") {
			t.Error(query)
		}
		testcloud.JSON(w, 200, identityFindBody("", ""))
	})
	collection := resource.NewCollection(identityFindAdapter(cloud))
	options := []resource.IdentityFindOption{resource.WithIdentityFindIgnoreMissing(false), resource.WithIdentityFindFallback(99), resource.WithIdentityFindFallback(resource.FindFallbackCompatible), resource.WithIdentityFindQuery("name", ""), resource.WithIdentityFindQuery("vendor", "a&b")}
	if value, err := collection.FindIdentity(context.Background(), "input", options...); value != nil || !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(value, err)
	}
	mode.Store(1)
	options = append(options, resource.WithIdentityFindOptions(resource.IdentityFindOpts{}))
	if value, err := collection.FindIdentity(context.Background(), "input", options...); value != nil || err != nil {
		t.Fatal(value, err)
	}
}

func TestCollectionFindIdentityContinuesAfterMatchAndRejectsDuplicates(t *testing.T) {
	for _, outcome := range []string{"id-match", "duplicate-name", "duplicate-same-id", "late-http", "late-decode", "invalid-row", "cycle"} {
		t.Run(outcome, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/v1/items/input", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 404, `{}`) })
			cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("marker") == "next" {
					switch outcome {
					case "id-match":
						testcloud.JSON(w, 200, identityFindBody(`{"id":"input","name":"not-input"}`, ""))
					case "duplicate-name":
						testcloud.JSON(w, 200, identityFindBody(`{"id":"second","name":"input"}`, ""))
					case "duplicate-same-id":
						testcloud.JSON(w, 200, identityFindBody(`{"id":"first","name":"input"}`, ""))
					case "late-http":
						testcloud.JSON(w, 403, `{"error":"late"}`)
					case "late-decode":
						testcloud.JSON(w, 200, identityFindBody(`{"id":true,"name":"unmatched"}`, ""))
					}
					return
				}
				rows := `{"id":"first","name":"input"}`
				if outcome == "id-match" {
					rows = `{"id":"other","name":"unmatched"}`
				}
				next := cloud.Server.URL + "/reverse/v1/items?marker=next"
				if outcome == "cycle" {
					next = cloud.Server.URL + r.URL.String()
				}
				if outcome == "invalid-row" {
					rows += `,{"id":"","name":"unmatched"}`
					next = ""
				}
				testcloud.JSON(w, 200, identityFindBody(rows, next))
			})
			collection := resource.NewCollection(identityFindAdapter(cloud))
			value, err := collection.FindIdentity(context.Background(), "input")
			switch outcome {
			case "id-match":
				if value == nil || value.ID != "input" || err != nil {
					t.Fatal(value, err)
				}
			case "duplicate-name", "duplicate-same-id":
				var duplicate *resource.AmbiguousError
				if value != nil || !errors.As(err, &duplicate) || len(duplicate.IDs) != 2 {
					t.Fatal(value, err, duplicate)
				}
			case "late-http":
				if value != nil || !gophercloud.ResponseCodeIs(err, 403) {
					t.Fatal(value, err)
				}
			case "late-decode":
				var typed *json.UnmarshalTypeError
				if value != nil || !errors.As(err, &typed) {
					t.Fatal(value, err)
				}
			case "invalid-row":
				if value != nil || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(value, err)
				}
			case "cycle":
				if value != nil || !errors.Is(err, resource.ErrPaginationCycle) {
					t.Fatal(value, err)
				}
			}
			want := int32(3)
			if outcome == "invalid-row" || outcome == "cycle" {
				want = 2
			}
			if calls.Load() != want {
				t.Fatal(calls.Load(), want)
			}
		})
	}
}

func TestCollectionFindIdentityAcceptedAndNestedTerminalErrorsNeverFallback(t *testing.T) {
	for _, outcome := range []string{"malformed", "missing-id", "nil-resource", "accepted", "transport", "decode", "read"} {
		t.Run(outcome, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, lists atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/v1/items/input", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if outcome == "malformed" {
					testcloud.JSON(w, 200, `{"item":`)
				} else if outcome == "missing-id" {
					testcloud.JSON(w, 200, `{"item":{"name":"input"}}`)
				} else {
					testcloud.JSON(w, 404, `{}`)
				}
			})
			cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) { lists.Add(1) })
			adapter := identityFindAdapter(cloud)
			get := adapter.Get
			adapter.Get = func(ctx context.Context, identity string) (*identityFindItem, error) {
				value, err := get(ctx, identity)
				switch outcome {
				case "nil-resource":
					return nil, nil
				case "accepted":
					return nil, &resource.ResponseError{Body: []byte(`{"accepted":true}`), Header: http.Header{"X-Evidence": {"owned"}}, StatusCode: 200, Cause: err}
				case "transport":
					return nil, &url.Error{Op: "GET", URL: "original", Err: err}
				case "decode":
					return nil, errors.Join(err, &json.UnmarshalTypeError{Value: "bool", Type: reflect.TypeFor[string]()})
				case "read":
					return nil, errors.Join(err, io.ErrUnexpectedEOF)
				}
				return value, err
			}
			value, err := resource.NewCollection(adapter).FindIdentity(context.Background(), "input")
			if value != nil || err == nil || gets.Load() != 1 || lists.Load() != 0 {
				t.Fatal(value, err, gets.Load(), lists.Load())
			}
			if outcome == "accepted" {
				var accepted *resource.ResponseError
				if !errors.As(err, &accepted) || accepted.StatusCode != 200 || accepted.Header.Get("X-Evidence") != "owned" || len(accepted.Body) == 0 {
					t.Fatal(err, accepted)
				}
			}
		})
	}
}

func TestCollectionFindIdentityRespectsBindingIDPolicyAndConsumedRows(t *testing.T) {
	cloud := testcloud.New(t)
	var lists, gets atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		testcloud.JSON(w, 200, identityFindBody(`{"id":"native-canonical","name":"ordinary"}`, ""))
	})
	adapter := identityFindAdapter(cloud)
	adapter.Get = func(context.Context, string) (*identityFindItem, error) { gets.Add(1); return nil, nil }
	adapter.ValidateID = func(id string) error {
		if !strings.HasPrefix(id, "native-") {
			return fmt.Errorf("%w: binding requires native ID", resource.ErrInvalidOption)
		}
		return nil
	}
	collection := resource.NewCollection(adapter)
	if value, err := collection.FindIdentity(context.Background(), "ordinary"); err != nil || value == nil || value.ID != "native-canonical" || lists.Load() != 1 || gets.Load() != 0 {
		t.Fatal(value, err, lists.Load(), gets.Load())
	}
	if value, err := collection.FindIdentity(context.Background(), "ordinary", resource.WithIdentityFindFallback(resource.FindFallbackNever)); value != nil || !errors.Is(err, resource.ErrInvalidOption) || lists.Load() != 1 || gets.Load() != 0 {
		t.Fatal(value, err, lists.Load(), gets.Load())
	}
}

func TestCollectionFindIdentityPreflightAndExplicitCapabilities(t *testing.T) {
	var calls int
	adapter := resource.Adapter[identityFindItem]{Kind: "owned", IdentityFind: true,
		Get: func(context.Context, string) (*identityFindItem, error) { calls++; return nil, nil },
		ID:  func(value *identityFindItem) string { return value.ID }, Name: func(value *identityFindItem) string { return value.Name },
	}
	collection := resource.NewCollection(adapter)
	for _, identity := range []string{"", " \u00a0 ", "x\n", "x\x00", string([]byte{0xff})} {
		if value, err := collection.FindIdentity(context.Background(), identity); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(identity, value, err)
		}
	}
	for _, option := range []resource.IdentityFindOption{nil, resource.WithIdentityFindFallback(99), resource.WithIdentityFindQuery("", "x"), resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: url.Values{"\n": {"x"}}})} {
		if value, err := collection.FindIdentity(context.Background(), "safe", option); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(value, err)
		}
	}
	for _, key := range []string{"max_items", "paginated", "base_path", "list_base_path", "jmespath_filters", "headers", "microversion", "allow_unknown_params", "ignore_missing", "fallback", "HEADERS"} {
		for _, option := range []resource.IdentityFindOption{resource.WithIdentityFindQuery(key, "x"), resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: url.Values{key: {"x"}}})} {
			if _, err := collection.FindIdentity(context.Background(), "safe", option); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(key, err)
			}
		}
	}
	if _, err := collection.FindIdentity(nil, "safe"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := collection.FindIdentity(ctx, "safe"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := collection.FindIdentity(context.Background(), "unsafe name"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	var nilCollection *resource.Collection[identityFindItem]
	zero := &resource.Collection[identityFindItem]{}
	adapter.IdentityFind = false
	for _, unsupported := range []*resource.Collection[identityFindItem]{nilCollection, zero, resource.NewCollection(adapter)} {
		if _, err := unsupported.FindIdentity(context.Background(), "safe"); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(err)
		}
	}
	if calls != 0 {
		t.Fatal("preflight reached GET", calls)
	}
}

func TestCollectionFindIdentityChecksCancellationBetweenPhasesAndAfterEmptyList(t *testing.T) {
	for _, phase := range []string{"get", "empty-list", "matched-list"} {
		t.Run(phase, func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var lists atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/v1/items/input", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
			cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				rows := ""
				if phase == "matched-list" {
					rows = `{"id":"input","name":"input"}`
				}
				testcloud.JSON(w, 200, identityFindBody(rows, ""))
			})
			adapter := identityFindAdapter(cloud)
			get := adapter.Get
			adapter.Get = func(ctx context.Context, id string) (*identityFindItem, error) {
				value, err := get(ctx, id)
				if phase == "get" {
					cancel()
				}
				return value, err
			}
			extract := adapter.Extract
			adapter.Extract = func(page pagination.Page) ([]identityFindItem, error) {
				values, err := extract(page)
				cancel()
				return values, err
			}
			// Native EachPage does not call Extract for an empty page. Use a
			// nonempty response while extracting zero rows to cover terminal ctx.
			if phase == "empty-list" {
				adapter.List = func(query url.Values) pagination.Pager {
					client := cloud.Client("test", "/reverse/v1")
					return pagination.NewPager(client, client.ServiceURL("items")+"?"+query.Encode(), func(result pagination.PageResult) pagination.Page {
						return valuePage{pagination.LinkedPageBase{PageResult: result}}
					})
				}
				adapter.Extract = func(pagination.Page) ([]identityFindItem, error) { cancel(); return nil, nil }
			}
			value, err := resource.NewCollection(adapter).FindIdentity(ctx, "input")
			if value != nil || !errors.Is(err, context.Canceled) {
				t.Fatal(value, err)
			}
			if phase == "get" && lists.Load() != 0 {
				t.Fatal("fallback continued after cancellation", lists.Load())
			}
		})
	}
}

func TestCollectionFindIdentityQueryHookCapabilityBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, lists atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/v1/items/safe", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if r.URL.RawQuery != "" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"item":{"id":"safe","name":"safe"}}`)
	})
	cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if r.URL.Query().Get("domain_id") != "domain" || r.URL.Query().Get("name") != "^unsafe name$" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, identityFindBody(`{"id":"canonical","name":"unsafe name"}`, ""))
	})
	adapter := identityFindAdapter(cloud)
	adapter.GetIdentityQuery = nil
	collection := resource.NewCollection(adapter)
	for _, query := range []url.Values{{"domain_id": {"domain"}}, {"name": nil}, {"name": {}}} {
		for _, policy := range []resource.FindFallbackPolicy{resource.FindFallbackCompatible, resource.FindFallbackNotFoundOnly, resource.FindFallbackNever} {
			value, err := collection.FindIdentity(context.Background(), "safe", resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: query, Fallback: policy}))
			if value != nil || !errors.Is(err, resource.ErrUnsupported) || gets.Load() != 0 || lists.Load() != 0 {
				t.Fatal(query, policy, value, err, gets.Load(), lists.Load())
			}
		}
	}
	if value, err := collection.FindIdentity(context.Background(), "safe", resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: url.Values{}})); err != nil || value == nil || gets.Load() != 1 {
		t.Fatal(value, err, gets.Load())
	}
	if value, err := collection.FindIdentity(context.Background(), "unsafe name", resource.WithIdentityFindQuery("domain_id", "domain")); err != nil || value == nil || gets.Load() != 1 || lists.Load() != 1 {
		t.Fatal(value, err, gets.Load(), lists.Load())
	}
	adapter.GetIdentityQuery = func(context.Context, string, url.Values) (*identityFindItem, error) {
		t.Error("empty query selected the query hook")
		return nil, nil
	}
	if value, err := resource.NewCollection(adapter).FindIdentity(context.Background(), "safe"); err != nil || value == nil || gets.Load() != 2 {
		t.Fatal(value, err, gets.Load())
	}
}

func TestCollectionFindIdentityQueryHookMutationsCannotChangeFallback(t *testing.T) {
	for _, nameValues := range [][]string{nil, {}, {""}, {"caller", "second"}} {
		t.Run(fmt.Sprint(nameValues), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			query := url.Values{"vendor": {"one", "two"}, "name": append([]string(nil), nameValues...)}
			frozen := url.Values{"vendor": {"one", "two"}, "name": append([]string(nil), nameValues...)}
			ignored := false
			option := resource.WithIdentityFindOptions(resource.IdentityFindOpts{IgnoreMissing: &ignored, Query: query})
			query["vendor"][0] = "before"
			query["name"] = []string{"before"}
			ignored = true
			cloud.Mux.HandleFunc("GET /reverse/v1/items/input", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != frozen.Encode() {
					t.Error("GET", r.URL.RawQuery, frozen.Encode())
				}
				testcloud.JSON(w, 403, `{"error":"private"}`)
			})
			cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != frozen.Encode() {
					t.Error("LIST", r.URL.RawQuery, frozen.Encode())
				}
				testcloud.JSON(w, 200, identityFindBody("", ""))
			})
			adapter := identityFindAdapter(cloud)
			get, list := adapter.GetIdentityQuery, adapter.List
			adapter.GetIdentityQuery = func(ctx context.Context, id string, received url.Values) (*identityFindItem, error) {
				if !reflect.DeepEqual(received, frozen) {
					t.Error("hook query", received, frozen)
				}
				value, err := get(ctx, id, received)
				received["vendor"][0] = "hook-mutated"
				received["name"] = []string{"hook-mutated"}
				received["extra"] = []string{"hook-mutated"}
				return value, err
			}
			adapter.List = func(received url.Values) pagination.Pager {
				if !reflect.DeepEqual(received, frozen) {
					t.Error("fallback map", received, frozen)
				}
				return list(received)
			}
			collection := resource.NewCollection(adapter)
			value, err := collection.FindIdentity(context.Background(), "input", option)
			var missing *resource.NotFoundError
			if value != nil || !errors.As(err, &missing) || missing.Cause != nil || gophercloud.ResponseCodeIs(err, 403) || calls.Load() != 2 {
				t.Fatal(value, err, missing, calls.Load())
			}
			if value, err := collection.FindIdentity(context.Background(), "input", option, resource.WithIdentityFindIgnoreMissing(true)); value != nil || err != nil || calls.Load() != 4 {
				t.Fatal("reused query hook or option was not independent", value, err, calls.Load())
			}
		})
	}
}

func TestCollectionFindIdentityQueryHookKeepsNativeAndTerminalCauses(t *testing.T) {
	for _, outcome := range []string{"not-found", "accepted", "transport", "decode", "read", "canceled"} {
		t.Run(outcome, func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var gets, lists atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/v1/items/input", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if r.URL.Query().Get("domain_id") != "domain" {
					t.Error(r.URL)
				}
				w.Header().Set("X-Evidence", "query-get")
				testcloud.JSON(w, 404, `{"error":"original"}`)
			})
			cloud.Mux.HandleFunc("GET /reverse/v1/items", func(w http.ResponseWriter, r *http.Request) { lists.Add(1) })
			adapter := identityFindAdapter(cloud)
			get := adapter.GetIdentityQuery
			adapter.GetIdentityQuery = func(ctx context.Context, id string, query url.Values) (*identityFindItem, error) {
				value, err := get(ctx, id, query)
				switch outcome {
				case "accepted":
					return nil, &resource.ResponseError{Body: []byte(`{"accepted":true}`), Header: http.Header{"X-Evidence": {"accepted"}}, StatusCode: 200, Cause: err}
				case "transport":
					return nil, &url.Error{Op: "GET", URL: "original", Err: err}
				case "decode":
					return nil, errors.Join(err, &json.UnmarshalTypeError{Value: "bool", Type: reflect.TypeFor[string]()})
				case "read":
					return nil, errors.Join(err, io.ErrUnexpectedEOF)
				case "canceled":
					cancel()
				}
				return value, err
			}
			options := []resource.IdentityFindOption{resource.WithIdentityFindQuery("domain_id", "domain")}
			if outcome == "not-found" {
				options = append(options, resource.WithIdentityFindFallback(resource.FindFallbackNever), resource.WithIdentityFindIgnoreMissing(false))
			}
			value, err := resource.NewCollection(adapter).FindIdentity(ctx, "input", options...)
			if value != nil || err == nil || gets.Load() != 1 || lists.Load() != 0 {
				t.Fatal(value, err, gets.Load(), lists.Load())
			}
			if outcome == "not-found" {
				var missing *resource.NotFoundError
				var native gophercloud.ErrUnexpectedResponseCode
				var operation, getOperation *resource.OperationError
				if !errors.As(err, &missing) || !errors.As(missing.Cause, &native) || native.ResponseHeader.Get("X-Evidence") != "query-get" || string(native.Body) != `{"error":"original"}` || !errors.As(err, &operation) || operation.Operation != "find_identity" || !errors.As(operation.Cause, &getOperation) || getOperation.Operation != "get" {
					t.Fatal(err, missing, native)
				}
			} else if outcome == "accepted" {
				var accepted *resource.ResponseError
				if !errors.As(err, &accepted) || accepted.StatusCode != 200 || accepted.Header.Get("X-Evidence") != "accepted" {
					t.Fatal(err, accepted)
				}
			} else if outcome == "canceled" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}
