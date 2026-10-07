package resource

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type identitySpecsItem struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	ExtraSpecs map[string]string `json:"extra_specs"`
}

type identitySpecsPage struct{ pagination.LinkedPageBase }

func (page identitySpecsPage) IsEmpty() (bool, error) {
	items, err := identitySpecsItems(page)
	return len(items) == 0, err
}

func identitySpecsItems(page pagination.Page) ([]identitySpecsItem, error) {
	var body struct {
		Items []identitySpecsItem `json:"items"`
	}
	err := page.(identitySpecsPage).ExtractInto(&body)
	return body.Items, err
}

const identitySpecsPath = "/extra/v1/items"

func identitySpecsBody(rows, next string) string {
	return fmt.Sprintf(`{"items":[%s],"links":{"next":%q}}`, rows, next)
}

func identitySpecsAdapter(cloud *testcloud.Cloud) Adapter[identitySpecsItem] {
	client := cloud.Client("test", "/extra/v1")
	get := func(ctx context.Context, id string, query url.Values) (*identitySpecsItem, error) {
		var body struct {
			Item identitySpecsItem `json:"item"`
		}
		_, err := client.Get(ctx, client.ServiceURL("items", id)+"?"+query.Encode(), &body, &gophercloud.RequestOpts{OkCodes: []int{200}})
		return &body.Item, err
	}
	return Adapter[identitySpecsItem]{
		Kind: "spec-items", IdentityFind: true,
		Get:                       func(ctx context.Context, id string) (*identitySpecsItem, error) { return get(ctx, id, nil) },
		GetIdentityQuery:          get,
		IdentityListQueryDefaults: url.Values{"is_public": {"None"}},
		IdentityExtraSpecs: func(ctx context.Context, value *identitySpecsItem) (*identitySpecsItem, error) {
			owned := *value
			owned.ExtraSpecs = maps.Clone(value.ExtraSpecs)
			var body struct {
				ExtraSpecs map[string]string `json:"extra_specs"`
			}
			_, err := client.Get(ctx, client.ServiceURL("items", value.ID, "os-extra-specs"), &body, &gophercloud.RequestOpts{OkCodes: []int{200}})
			if err != nil {
				return nil, err
			}
			owned.ExtraSpecs = body.ExtraSpecs
			return &owned, nil
		},
		List: func(query url.Values) pagination.Pager {
			return pagination.NewPager(client, client.ServiceURL("items")+"?"+query.Encode(), func(result pagination.PageResult) pagination.Page {
				return identitySpecsPage{pagination.LinkedPageBase{PageResult: result}}
			})
		},
		Extract: identitySpecsItems,
		ID:      func(value *identitySpecsItem) string { return value.ID },
		Name:    func(value *identitySpecsItem) string { return value.Name },
	}
}

func TestCollectionFindIdentityExtraSpecsDefaultsOnlyAffectIdentityLists(t *testing.T) {
	for _, check := range []struct {
		name  string
		query url.Values
	}{
		{"default", nil},
		{"false", url.Values{"is_public": {"false"}}},
		{"repeated", url.Values{"is_public": {"true", "false"}, "tag": {"one", "two"}}},
		{"nil", url.Values{"is_public": nil}},
		{"empty-slice", url.Values{"is_public": {}}},
		{"empty-string", url.Values{"is_public": {""}}},
		{"caller-name", url.Values{"name": {"one", "two"}}},
	} {
		t.Run(check.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, lists, extras atomic.Int32
			cloud.Mux.HandleFunc("GET "+identitySpecsPath+"/identity", func(w http.ResponseWriter, request *http.Request) {
				gets.Add(1)
				if request.URL.RawQuery != check.query.Encode() {
					t.Error("list defaults leaked into direct GET", request.URL)
				}
				w.WriteHeader(404)
			})
			cloud.Mux.HandleFunc("GET "+identitySpecsPath, func(w http.ResponseWriter, request *http.Request) {
				lists.Add(1)
				testcloud.JSON(w, 200, identitySpecsBody(`{"id":"canonical","name":"identity"}`, ""))
			})
			cloud.Mux.HandleFunc("GET "+identitySpecsPath+"/canonical/os-extra-specs", func(w http.ResponseWriter, request *http.Request) {
				extras.Add(1)
				w.WriteHeader(500)
			})
			adapter := identitySpecsAdapter(cloud)
			nativeList := adapter.List
			adapter.List = func(query url.Values) pagination.Pager {
				wanted := cloneIdentityFindOptions(IdentityFindOpts{Query: check.query}).Query
				if !wanted.Has("is_public") {
					wanted.Set("is_public", "None")
				}
				if !reflect.DeepEqual(query, wanted) {
					t.Error("default replaced caller key presence or values", query, wanted)
				}
				return nativeList(query)
			}
			value, err := NewCollection(adapter).FindIdentity(context.Background(), "identity", WithIdentityFindOptions(IdentityFindOpts{Query: check.query}))
			if err != nil || value == nil || value.ID != "canonical" || gets.Load() != 1 || lists.Load() != 1 || extras.Load() != 0 {
				t.Fatal(value, err, gets.Load(), lists.Load(), extras.Load())
			}
		})
	}
}

func TestCollectionFindIdentityExtraSpecsCapabilityAndQueryPreflight(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, request *http.Request) { calls.Add(1); w.WriteHeader(500) })
	for _, enabled := range []bool{true, false} {
		adapter := identitySpecsAdapter(cloud)
		adapter.IdentityExtraSpecs = nil
		_, err := NewCollection(adapter).FindIdentity(context.Background(), "identity", WithIdentityFindExtraSpecs(enabled), WithIdentityFindFallback(FindFallbackNever))
		if !errors.Is(err, ErrUnsupported) {
			t.Fatal("unsupported explicit extra-spec option reached GET", enabled, err)
		}
	}
	for _, key := range []string{"get_extra_specs", "GET_EXTRA_SPECS", "max_items", "base_path", " "} {
		adapter := identitySpecsAdapter(cloud)
		adapter.IdentityListQueryDefaults = url.Values{key: {"bad"}}
		if _, err := NewCollection(adapter).FindIdentity(context.Background(), "identity", WithIdentityFindFallback(FindFallbackNever)); !errors.Is(err, ErrInvalidOption) {
			t.Fatal("invalid SDK default accepted before GET", key, err)
		}
	}
	for _, key := range []string{"get_extra_specs", "GET_EXTRA_SPECS", "Get_Extra_Specs"} {
		for _, option := range []IdentityFindOption{WithIdentityFindQuery(key, "true"), WithIdentityFindOptions(IdentityFindOpts{Query: url.Values{key: nil}})} {
			if _, err := NewCollection(identitySpecsAdapter(cloud)).FindIdentity(context.Background(), "identity", option); !errors.Is(err, ErrInvalidOption) {
				t.Fatal("raw SDK control accepted", key, err)
			}
		}
	}
	if _, err := NewCollection(identitySpecsAdapter(cloud)).FindIdentity(context.Background(), "identity", WithIdentityFindDetails(false)); !errors.Is(err, ErrUnsupported) {
		t.Fatal("extra-spec capability invented details support", err)
	}
	if calls.Load() != 0 {
		t.Fatal("invalid options performed HTTP", calls.Load())
	}
}

func TestCollectionFindIdentityExtraSpecsEnrichesValidatedUniqueSuccess(t *testing.T) {
	for _, check := range []struct {
		name     string
		fallback bool
		options  []IdentityFindOption
		enrich   bool
	}{
		{"default-false", false, nil, false},
		{"explicit-false", false, []IdentityFindOption{WithIdentityFindExtraSpecs(false)}, false},
		{"last-false", false, []IdentityFindOption{WithIdentityFindExtraSpecs(true), WithIdentityFindExtraSpecs(false)}, false},
		{"bulk-reset", false, []IdentityFindOption{WithIdentityFindExtraSpecs(true), WithIdentityFindOptions(IdentityFindOpts{})}, false},
		{"get-true", false, []IdentityFindOption{WithIdentityFindExtraSpecs(true)}, true},
		{"never-get-true", false, []IdentityFindOption{WithIdentityFindExtraSpecs(true), WithIdentityFindFallback(FindFallbackNever)}, true},
		{"list-after-all-pages", true, []IdentityFindOption{WithIdentityFindExtraSpecs(false), WithIdentityFindExtraSpecs(true)}, true},
	} {
		t.Run(check.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, lists, extras atomic.Int32
			cloud.Mux.HandleFunc("GET "+identitySpecsPath+"/identity", func(w http.ResponseWriter, request *http.Request) {
				gets.Add(1)
				if check.fallback {
					w.WriteHeader(404)
					return
				}
				testcloud.JSON(w, 200, `{"item":{"id":"canonical","name":"different"}}`)
			})
			cloud.Mux.HandleFunc("GET "+identitySpecsPath, func(w http.ResponseWriter, request *http.Request) {
				if lists.Add(1) == 1 {
					testcloud.JSON(w, 200, identitySpecsBody(`{"id":"canonical","name":"identity"}`, cloud.Server.URL+identitySpecsPath+"?marker=second&is_public=None"))
				} else {
					testcloud.JSON(w, 200, identitySpecsBody(`{"id":"other","name":"unrelated"}`, ""))
				}
			})
			cloud.Mux.HandleFunc("GET "+identitySpecsPath+"/canonical/os-extra-specs", func(w http.ResponseWriter, request *http.Request) {
				extras.Add(1)
				if check.fallback && lists.Load() != 2 || request.URL.RawQuery != "" {
					t.Error("enrichment preceded uniqueness or inherited lookup query", lists.Load(), request.URL)
				}
				testcloud.JSON(w, 200, `{"extra_specs":{"hw:arch":"x86_64"}}`)
			})
			value, err := NewCollection(identitySpecsAdapter(cloud)).FindIdentity(context.Background(), "identity", check.options...)
			wantedExtras, wantedLists := int32(0), int32(0)
			if check.enrich {
				wantedExtras = 1
			}
			if check.fallback {
				wantedLists = 2
			}
			if err != nil || value == nil || value.ID != "canonical" || check.enrich && value.ExtraSpecs["hw:arch"] != "x86_64" || gets.Load() != 1 || lists.Load() != wantedLists || extras.Load() != wantedExtras {
				t.Fatal(value, err, gets.Load(), lists.Load(), extras.Load())
			}
		})
	}
}

func TestCollectionFindIdentityExtraSpecsListFailuresAndAbsenceNeverEnrich(t *testing.T) {
	for _, mode := range []string{"late-http", "duplicate", "cycle", "decode", "invalid-id", "empty", "strict-empty", "never-404", "strict-never-404", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var lists, extras atomic.Int32
			cloud.Mux.HandleFunc("GET "+identitySpecsPath+"/identity", func(w http.ResponseWriter, request *http.Request) { w.WriteHeader(404) })
			cloud.Mux.HandleFunc("GET "+identitySpecsPath, func(w http.ResponseWriter, request *http.Request) {
				page := lists.Add(1)
				rows, next := `{"id":"canonical","name":"identity"}`, ""
				switch mode {
				case "late-http", "duplicate":
					if page == 1 {
						next = cloud.Server.URL + identitySpecsPath + "?marker=second"
					} else if mode == "late-http" {
						w.WriteHeader(503)
						return
					}
				case "cycle":
					rows = `{"id":"other","name":"unrelated"}`
					next = cloud.Server.URL + identitySpecsPath + "?marker=cycle"
				case "decode":
					rows += `,{"id":"other","extra_specs":false}`
				case "invalid-id":
					rows += `,{"id":"","name":"unrelated"}`
				case "empty", "strict-empty":
					rows = ""
				case "cancel":
					cancel()
				}
				testcloud.JSON(w, 200, identitySpecsBody(rows, next))
			})
			cloud.Mux.HandleFunc("GET "+identitySpecsPath+"/canonical/os-extra-specs", func(w http.ResponseWriter, request *http.Request) { extras.Add(1); w.WriteHeader(500) })
			options := []IdentityFindOption{WithIdentityFindExtraSpecs(true)}
			if mode == "never-404" || mode == "strict-never-404" {
				options = append(options, WithIdentityFindFallback(FindFallbackNever))
			}
			if mode == "strict-empty" || mode == "strict-never-404" {
				options = append(options, WithIdentityFindIgnoreMissing(false))
			}
			value, err := NewCollection(identitySpecsAdapter(cloud)).FindIdentity(ctx, "identity", options...)
			if value != nil || extras.Load() != 0 {
				t.Fatal("absence or incomplete list was enriched", value, err, extras.Load())
			}
			if mode == "empty" || mode == "never-404" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || mode == "duplicate" && !errors.Is(err, ErrAmbiguous) || mode == "cycle" && !errors.Is(err, ErrPaginationCycle) || mode == "invalid-id" && !errors.Is(err, ErrInvalidOption) || mode == "cancel" && !errors.Is(err, context.Canceled) || mode == "late-http" && !gophercloud.ResponseCodeIs(err, 503) {
				t.Fatal(err)
			}
			if mode == "strict-empty" || mode == "strict-never-404" {
				if !errors.Is(err, ErrNotFound) || gophercloud.ResponseCodeIs(err, 404) != (mode == "strict-never-404") {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestCollectionFindIdentityExtraSpecsErrorsAndInvalidResultsAreTerminal(t *testing.T) {
	for _, phase := range []string{"get", "list"} {
		for _, mode := range []string{"403", "404", "500", "decode", "nil-result", "unsafe-result", "cancel"} {
			t.Run(phase+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var lists, extras atomic.Int32
				cloud.Mux.HandleFunc("GET "+identitySpecsPath+"/identity", func(w http.ResponseWriter, request *http.Request) {
					if phase == "list" {
						w.WriteHeader(404)
						return
					}
					testcloud.JSON(w, 200, `{"item":{"id":"canonical","name":"identity"}}`)
				})
				cloud.Mux.HandleFunc("GET "+identitySpecsPath, func(w http.ResponseWriter, request *http.Request) {
					lists.Add(1)
					testcloud.JSON(w, 200, identitySpecsBody(`{"id":"canonical","name":"identity"}`, ""))
				})
				cloud.Mux.HandleFunc("GET "+identitySpecsPath+"/canonical/os-extra-specs", func(w http.ResponseWriter, request *http.Request) {
					extras.Add(1)
					w.Header().Set("X-Spec-Evidence", "terminal")
					switch mode {
					case "403", "404", "500":
						code := map[string]int{"403": 403, "404": 404, "500": 500}[mode]
						testcloud.JSON(w, code, `{"error":"extra specs denied"}`)
					case "decode":
						testcloud.JSON(w, 200, `{"extra_specs":false}`)
					default:
						testcloud.JSON(w, 200, `{"extra_specs":{"hw:arch":"x86_64"}}`)
					}
				})
				adapter := identitySpecsAdapter(cloud)
				nativeExtra := adapter.IdentityExtraSpecs
				adapter.IdentityExtraSpecs = func(ctx context.Context, value *identitySpecsItem) (*identitySpecsItem, error) {
					value, err := nativeExtra(ctx, value)
					if err != nil {
						return nil, err
					}
					switch mode {
					case "nil-result":
						return nil, nil
					case "unsafe-result":
						value.ID = "unsafe/id"
					case "cancel":
						cancel()
					}
					return value, nil
				}
				value, err := NewCollection(adapter).FindIdentity(ctx, "identity", WithIdentityFindExtraSpecs(true))
				wantedLists := int32(0)
				if phase == "list" {
					wantedLists = 1
				}
				var operation *OperationError
				if value != nil || err == nil || extras.Load() != 1 || lists.Load() != wantedLists || !errors.As(err, &operation) || operation.Operation != "find_identity" {
					t.Fatal("enrichment failure started another search or lost operation", value, err, lists.Load(), extras.Load())
				}
				if mode == "403" || mode == "404" || mode == "500" {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.ResponseHeader.Get("X-Spec-Evidence") != "terminal" || string(native.Body) != `{"error":"extra specs denied"}` {
						t.Fatal("native enrichment evidence lost", err, native)
					}
				} else if (mode == "nil-result" || mode == "unsafe-result") && !errors.Is(err, ErrInvalidOption) || mode == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestCollectionFindIdentityExtraSpecsSnapshotsMetadataOptionsAndCallback(t *testing.T) {
	for _, custom := range []bool{false, true} {
		t.Run(fmt.Sprint("custom-", custom), func(t *testing.T) {
			cloud := testcloud.New(t)
			defaults := url.Values{"is_public": {"None"}}
			adapter := identitySpecsAdapter(cloud)
			adapter.IdentityListQueryDefaults = defaults
			collection := NewCollection(adapter)
			defaults["is_public"][0] = "constructor mutation"
			getExtra := true
			input := IdentityFindOpts{GetExtraSpecs: &getExtra, Query: url.Values{"tag": {"one", "two"}, "name": nil}}
			option := WithIdentityFindOptions(input)
			var captured *IdentityFindOpts
			if custom {
				option = func(options *IdentityFindOpts) error { *options = input; captured = options; return nil }
			} else {
				getExtra = false
				input.Query["tag"][0] = "constructor mutation"
			}
			var extras, replacements atomic.Int32
			cloud.Mux.HandleFunc("GET "+identitySpecsPath+"/identity", func(w http.ResponseWriter, request *http.Request) {
				if request.URL.Query().Has("is_public") || !reflect.DeepEqual(request.URL.Query()["tag"], []string{"one", "two"}) {
					t.Error(request.URL)
				}
				if captured != nil {
					*captured.GetExtraSpecs = false
					captured.Query["tag"][0] = "GET mutation"
					captured.Query.Set("name", "GET mutation")
				}
				collection.binding.IdentityListQueryDefaults.Set("is_public", "GET mutation")
				collection.binding.IdentityExtraSpecs = func(context.Context, *identitySpecsItem) (*identitySpecsItem, error) {
					replacements.Add(1)
					return nil, errors.New("replacement callback")
				}
				w.WriteHeader(404)
			})
			cloud.Mux.HandleFunc("GET "+identitySpecsPath, func(w http.ResponseWriter, request *http.Request) {
				if request.URL.Query().Get("is_public") != "None" || request.URL.Query().Has("name") || !reflect.DeepEqual(request.URL.Query()["tag"], []string{"one", "two"}) {
					t.Error("metadata or captured option changed list policy", request.URL)
				}
				testcloud.JSON(w, 200, identitySpecsBody(`{"id":"canonical","name":"identity"}`, ""))
			})
			cloud.Mux.HandleFunc("GET "+identitySpecsPath+"/canonical/os-extra-specs", func(w http.ResponseWriter, request *http.Request) {
				extras.Add(1)
				testcloud.JSON(w, 200, `{"extra_specs":{"hw:arch":"x86_64"}}`)
			})
			value, err := collection.FindIdentity(context.Background(), "identity", option)
			if err != nil || value == nil || value.ExtraSpecs["hw:arch"] != "x86_64" || extras.Load() != 1 || replacements.Load() != 0 {
				t.Fatal(value, err, extras.Load(), replacements.Load())
			}
		})
	}
}

func TestCollectionFindIdentityExtraSpecsConcurrentOwnedOptionReuse(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, extras atomic.Int32
	cloud.Mux.HandleFunc("GET "+identitySpecsPath+"/identity", func(w http.ResponseWriter, request *http.Request) {
		gets.Add(1)
		if !reflect.DeepEqual(request.URL.Query()["tag"], []string{"one", "two"}) || request.URL.Query().Has("is_public") {
			t.Error(request.URL)
		}
		testcloud.JSON(w, 200, `{"item":{"id":"canonical","name":"identity"}}`)
	})
	cloud.Mux.HandleFunc("GET "+identitySpecsPath+"/canonical/os-extra-specs", func(w http.ResponseWriter, request *http.Request) {
		extras.Add(1)
		testcloud.JSON(w, 200, `{"extra_specs":{"hw:arch":"x86_64"}}`)
	})
	getExtra := true
	input := IdentityFindOpts{GetExtraSpecs: &getExtra, Query: url.Values{"tag": {"one", "two"}}}
	option := WithIdentityFindOptions(input)
	getExtra, input.Query["tag"][0] = false, "changed"
	collection := NewCollection(identitySpecsAdapter(cloud))
	var group sync.WaitGroup
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			value, err := collection.FindIdentity(context.Background(), "identity", option)
			if err != nil || value == nil || value.ExtraSpecs["hw:arch"] != "x86_64" {
				t.Error(value, err)
			}
		}()
	}
	group.Wait()
	if gets.Load() != 8 || extras.Load() != 8 {
		t.Fatal(gets.Load(), extras.Load())
	}
}

func TestCollectionFindIdentityExtraSpecsDefaultsPreserveOrdinaryAndSecondarySearches(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, lists, extras atomic.Int32
	cloud.Mux.HandleFunc("GET "+identitySpecsPath+"/identity", func(w http.ResponseWriter, request *http.Request) {
		gets.Add(1)
		if request.URL.RawQuery != "" {
			t.Error(request.URL)
		}
		if gets.Load() > 1 {
			w.WriteHeader(404)
			return
		}
		testcloud.JSON(w, 200, `{"item":{"id":"canonical","name":"identity"}}`)
	})
	cloud.Mux.HandleFunc("GET "+identitySpecsPath, func(w http.ResponseWriter, request *http.Request) {
		page := lists.Add(1)
		if page <= 2 {
			if request.URL.RawQuery != "" {
				t.Error("identity policy changed ordinary List/RefFind", request.URL)
			}
			testcloud.JSON(w, 200, identitySpecsBody(`{"id":"canonical","name":"identity"}`, ""))
			return
		}
		if request.URL.Query().Get("is_public") != "None" || request.URL.Query().Has("name") {
			t.Error("list defaults lost or name hint invented", request.URL)
		}
		if request.URL.Query().Get("os_hidden") == "true" {
			testcloud.JSON(w, 200, identitySpecsBody(`{"id":"canonical","name":"identity"}`, ""))
			return
		}
		testcloud.JSON(w, 200, identitySpecsBody("", ""))
	})
	cloud.Mux.HandleFunc("GET "+identitySpecsPath+"/canonical/os-extra-specs", func(w http.ResponseWriter, request *http.Request) { extras.Add(1); w.WriteHeader(500) })
	adapter := identitySpecsAdapter(cloud)
	adapter.IdentityMissingListQuery = url.Values{"os_hidden": {"true"}}
	collection := NewCollection(adapter)
	if value, err := collection.Get(context.Background(), "identity"); err != nil || value == nil {
		t.Fatal(value, err)
	}
	if values, err := collection.All(context.Background()); err != nil || len(values) != 1 {
		t.Fatal(values, err)
	}
	if value, err := collection.Find(context.Background(), Name("identity")); err != nil || value == nil {
		t.Fatal(value, err)
	}
	if value, err := collection.FindIdentity(context.Background(), "identity"); err != nil || value == nil || value.ID != "canonical" {
		t.Fatal(value, err)
	}
	if gets.Load() != 2 || lists.Load() != 4 || extras.Load() != 0 {
		t.Fatal(gets.Load(), lists.Load(), extras.Load())
	}
}
