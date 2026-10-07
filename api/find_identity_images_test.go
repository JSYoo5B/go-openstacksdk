package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	image "github.com/JSYoo5B/gophercloudsdk/image"
	"github.com/JSYoo5B/gophercloudsdk/image/v2/images"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Pinned image.find adds one hidden-image list only after Resource.find reports
// complete logical absence. It uses the original query plus os_hidden=True,
// without repeating GET or adding Resource.find's automatic name hint. Native
// Glance keeps root-object GET results and ImagePage's {images,next} contract.
type imageIdentityFind func(context.Context, string, ...resource.IdentityFindOption) (*images.Image, error)

type imageIdentityFixture struct {
	name string
	new  func(*gophercloud.ServiceClient) imageIdentityFind
}

func imageIdentityFixtures() []imageIdentityFixture {
	return []imageIdentityFixture{
		{"generated-leaf", func(c *gophercloud.ServiceClient) imageIdentityFind { return images.New(c).FindIdentity }},
		{"handwritten-collection", func(c *gophercloud.ServiceClient) imageIdentityFind { return image.New(c).Images.FindIdentity }},
	}
}

const imageIdentityPath = "/reverse/glance/v2/images"

func imageIdentityClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("image", "/unused/catalog")
	client.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + "/reverse/glance/v2")
	return client
}

func imageIdentityPage(rows, next string) string {
	return fmt.Sprintf(`{"images":[%s],"next":%q}`, rows, next)
}

func imageIdentityNext(query string) string {
	// Glance's native nextPageURL retains the service's reverse prefix while
	// resolving the advertised version-relative path.
	return "/v2/images?" + query
}

func TestNativeFindIdentityImagesGETContractsAndNativeModel(t *testing.T) {
	for _, f := range imageIdentityFixtures() {
		for _, query := range []bool{false, true} {
			for _, response := range []struct {
				name   string
				status int
				body   string
				ok     bool
			}{
				{"root-object", 200, `{"id":"returned","name":"Different","status":"active","os_hidden":true,"size":12345,"virtual_size":45678,"tags":["one","two"],"created_at":"2025-01-02T03:04:05Z","vendor_property":"kept"}`, true},
				{"unexpected-203", 203, `{"id":"returned","name":"lookup"}`, false},
				{"unexpected-204", 204, "", false},
				{"malformed-typed-200", 200, `{"id":"returned","name":"lookup","size":"not-a-number"}`, false},
			} {
				t.Run(fmt.Sprintf("%s/%s/query-%t", f.name, response.name, query), func(t *testing.T) {
					cloud := testcloud.New(t)
					var gets, lists atomic.Int32
					cloud.Mux.HandleFunc("GET "+imageIdentityPath+"/lookup", func(w http.ResponseWriter, r *http.Request) {
						gets.Add(1)
						if query && !reflect.DeepEqual(r.URL.Query()["tag"], []string{"first", "second"}) || !query && r.URL.RawQuery != "" {
							t.Error("native GET query changed", r.URL)
						}
						w.Header().Set("Openstack-Image-Import-Methods", "glance-direct,web-download")
						w.Header().Set("Openstack-Image-Store-Ids", "fast,archive")
						if response.status == 204 {
							w.WriteHeader(204)
						} else {
							testcloud.JSON(w, response.status, response.body)
						}
					})
					cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { lists.Add(1); w.WriteHeader(500) })
					var opts []resource.IdentityFindOption
					if query {
						opts = append(opts, resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: url.Values{"tag": {"first", "second"}}}))
					}
					value, err := f.new(imageIdentityClient(cloud))(context.Background(), "lookup", opts...)
					if gets.Load() != 1 || lists.Load() != 0 {
						t.Fatal("GET was retried or followed by a search", gets.Load(), lists.Load(), err)
					}
					if !response.ok {
						if value != nil || err == nil {
							t.Fatal(value, err)
						}
						if response.status != 200 && !gophercloud.ResponseCodeIs(err, response.status) {
							t.Fatal("native status lost", err)
						}
						return
					}
					if err != nil || value == nil || value.ID != "returned" || value.Name != "Different" || !value.Hidden || value.SizeBytes != 12345 || value.VirtualSize != 45678 || value.Properties["vendor_property"] != "kept" || value.CreatedAt.IsZero() || !reflect.DeepEqual(value.Tags, []string{"one", "two"}) || !reflect.DeepEqual(value.OpenStackImageImportMethods, []string{"glance-direct", "web-download"}) || !reflect.DeepEqual(value.OpenStackImageStoreIDs, []string{"fast", "archive"}) {
						t.Fatal("native image extraction changed", value, err)
					}
				})
			}
		}
	}
}

func TestNativeFindIdentityImagesHiddenPassUsesOriginalCallerQuery(t *testing.T) {
	for _, f := range imageIdentityFixtures() {
		for _, test := range []struct {
			name  string
			query url.Values
		}{
			{"automatic-name-first-only", url.Values{"tag": {"first", "second"}}},
			{"explicit-name", url.Values{"name": {"caller-pattern"}, "tag": {"first", "second"}}},
			{"repeated-name", url.Values{"name": {"one", "two"}}},
			{"nil-name", url.Values{"name": nil}},
			{"empty-name-slice", url.Values{"name": {}}},
			{"empty-name-value", url.Values{"name": {""}}},
			{"explicit-hidden-false-overridden", url.Values{"os_hidden": {"false"}, "vendor": {"first", "second"}}},
		} {
			t.Run(f.name+"/"+test.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets, normal, hidden atomic.Int32
				first := make(url.Values)
				second := make(url.Values)
				for key, values := range test.query {
					first[key] = append([]string(nil), values...)
					second[key] = append([]string(nil), values...)
				}
				if !first.Has("name") {
					first.Set("name", "lookup")
				}
				second.Set("os_hidden", "true")
				cloud.Mux.HandleFunc("GET "+imageIdentityPath+"/lookup", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					if r.URL.RawQuery != test.query.Encode() {
						t.Error("caller query not forwarded to GET", r.URL)
					}
					w.WriteHeader(404)
				})
				cloud.Mux.HandleFunc("GET "+imageIdentityPath, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Query().Get("os_hidden") != "true" {
						normal.Add(1)
						if r.URL.RawQuery != first.Encode() {
							t.Error("first search hint or caller query changed", r.URL, first)
						}
						testcloud.JSON(w, 200, imageIdentityPage("", ""))
						return
					}
					hidden.Add(1)
					if r.URL.RawQuery != second.Encode() {
						t.Error("hidden search reused automatic hint or lost caller query", r.URL, second)
					}
					// Match only the ID: hidden retry must not impose a local name
					// predicate or send a second direct GET.
					testcloud.JSON(w, 200, imageIdentityPage(`{"id":"lookup","name":"Different","os_hidden":true}`, ""))
				})
				value, err := f.new(imageIdentityClient(cloud))(context.Background(), "lookup", resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: test.query}))
				if err != nil || value == nil || value.ID != "lookup" || value.Name != "Different" || !value.Hidden || gets.Load() != 1 || normal.Load() != 1 || hidden.Load() != 1 {
					t.Fatal(value, err, gets.Load(), normal.Load(), hidden.Load())
				}
			})
		}
	}
}

func TestNativeFindIdentityImagesMissingDefaultsAndFallbackBoundaries(t *testing.T) {
	for _, f := range imageIdentityFixtures() {
		for _, code := range []int{400, 403, 404} {
			for _, policy := range []resource.FindFallbackPolicy{resource.FindFallbackCompatible, resource.FindFallbackNotFoundOnly, resource.FindFallbackNever} {
				for _, strict := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%d/policy-%d/strict-%t", f.name, code, policy, strict), func(t *testing.T) {
						cloud := testcloud.New(t)
						var gets, lists atomic.Int32
						cloud.Mux.HandleFunc("GET "+imageIdentityPath+"/lookup", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); w.WriteHeader(code) })
						cloud.Mux.HandleFunc("GET "+imageIdentityPath, func(w http.ResponseWriter, r *http.Request) {
							lists.Add(1)
							if lists.Load() == 1 && r.URL.Query().Get("name") != "lookup" || lists.Load() == 2 && (r.URL.Query().Has("name") || r.URL.Query().Get("os_hidden") != "true") {
								t.Error("absence search phases changed", r.URL)
							}
							testcloud.JSON(w, 200, imageIdentityPage("", ""))
						})
						value, err := f.new(imageIdentityClient(cloud))(context.Background(), "lookup", resource.WithIdentityFindFallback(policy), resource.WithIdentityFindIgnoreMissing(!strict))
						fallback := policy == resource.FindFallbackCompatible || policy == resource.FindFallbackNotFoundOnly && code == 404
						if value != nil || gets.Load() != 1 {
							t.Fatal(value, err, gets.Load())
						}
						if fallback {
							if lists.Load() != 2 || strict && !errors.Is(err, resource.ErrNotFound) || !strict && err != nil {
								t.Fatal("logical absence was decided before both searches", err, lists.Load())
							}
							if strict {
								var missing *resource.NotFoundError
								if !errors.As(err, &missing) || missing.Cause != nil {
									t.Fatal("logical absence retained suppressed GET failure", err)
								}
							}
						} else if lists.Load() != 0 || code == 404 && !strict && err != nil || (code != 404 || strict) && !gophercloud.ResponseCodeIs(err, code) {
							t.Fatal("fallback policy widened", err, lists.Load())
						}
					})
				}
			}
		}
	}
}

func TestNativeFindIdentityImagesNormalSearchTerminalNeverRetriesHidden(t *testing.T) {
	for _, f := range imageIdentityFixtures() {
		for _, mode := range []string{"match", "duplicate", "late-http", "full-page-decode", "invalid-row", "cycle", "cancel"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var gets, normal, hidden atomic.Int32
				cloud.Mux.HandleFunc("GET "+imageIdentityPath+"/lookup", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); w.WriteHeader(404) })
				cloud.Mux.HandleFunc("GET "+imageIdentityPath, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Query().Get("os_hidden") == "true" {
						hidden.Add(1)
						w.WriteHeader(500)
						return
					}
					page := normal.Add(1)
					rows, next := `{"id":"found","name":"lookup"}`, ""
					switch mode {
					case "duplicate", "late-http":
						if page == 1 {
							next = imageIdentityNext("marker=second&name=lookup")
						} else if mode == "late-http" {
							w.WriteHeader(503)
							return
						}
					case "full-page-decode":
						rows += `,{"id":"other","name":"irrelevant","virtual_size":"bad"}`
					case "invalid-row":
						rows += `,{"id":"","name":"irrelevant"}`
					case "cycle":
						rows = `{"id":"other","name":"irrelevant"}`
						next = imageIdentityNext("marker=cycle&name=lookup")
					case "cancel":
						cancel()
					}
					testcloud.JSON(w, 200, imageIdentityPage(rows, next))
				})
				value, err := f.new(imageIdentityClient(cloud))(ctx, "lookup")
				if gets.Load() != 1 || hidden.Load() != 0 {
					t.Fatal("terminal normal observation triggered hidden retry", err, gets.Load(), normal.Load(), hidden.Load())
				}
				if mode == "match" {
					if err != nil || value == nil || value.ID != "found" || normal.Load() != 1 {
						t.Fatal(value, err, normal.Load())
					}
					return
				}
				if value != nil || err == nil {
					t.Fatal(value, err)
				}
				switch mode {
				case "duplicate":
					if !errors.Is(err, resource.ErrAmbiguous) || normal.Load() != 2 {
						t.Fatal(err, normal.Load())
					}
				case "late-http":
					if !gophercloud.ResponseCodeIs(err, 503) || normal.Load() != 2 {
						t.Fatal(err, normal.Load())
					}
				case "invalid-row":
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				case "cancel":
					if !errors.Is(err, context.Canceled) {
						t.Fatal(err)
					}
				}
			})
		}
	}
}

func TestNativeFindIdentityImagesHiddenPagesAndTerminalFailures(t *testing.T) {
	for _, f := range imageIdentityFixtures() {
		for _, mode := range []string{"match-300", "duplicate", "late-http", "full-page-decode", "invalid-row", "cycle", "list-403", "list-404", "empty-204", "strict-empty-204"} {
			t.Run(f.name+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets, normal, hidden atomic.Int32
				cloud.Mux.HandleFunc("GET "+imageIdentityPath+"/lookup", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); w.WriteHeader(404) })
				cloud.Mux.HandleFunc("GET "+imageIdentityPath, func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Query().Get("os_hidden") != "true" {
						page := normal.Add(1)
						if mode == "match-300" && page == 1 {
							testcloud.JSON(w, 200, imageIdentityPage(`{"id":"normal-other","name":"irrelevant"}`, imageIdentityNext("marker=normal-next&name=lookup")))
							return
						}
						testcloud.JSON(w, 200, imageIdentityPage("", ""))
						return
					}
					page := hidden.Add(1)
					if r.URL.Query().Has("name") {
						t.Error("automatic name leaked to hidden pages", r.URL)
					}
					if mode == "empty-204" || mode == "strict-empty-204" {
						w.WriteHeader(204)
						return
					}
					if mode == "list-403" || mode == "list-404" {
						status := 403
						if mode == "list-404" {
							status = 404
						}
						w.WriteHeader(status)
						return
					}
					rows, next, code := `{"id":"other","name":"irrelevant"}`, imageIdentityNext("marker=hidden-next&os_hidden=true"), 200
					if page > 1 {
						rows, next = `{"id":"lookup","name":"Different","os_hidden":true,"size":321}`, ""
					}
					switch mode {
					case "match-300":
						if page > 1 {
							code = 300
						}
					case "duplicate":
						rows = `{"id":"lookup","name":"Different"}`
					case "late-http":
						if page == 1 {
							rows = `{"id":"lookup","name":"Different"}`
						} else {
							w.WriteHeader(502)
							return
						}
					case "full-page-decode":
						rows, next = `{"id":"lookup","name":"Different"},{"id":"other","tags":false}`, ""
					case "invalid-row":
						rows, next = `{"id":"lookup","name":"Different"},{"name":"unrelated"}`, ""
					case "cycle":
						rows = `{"id":"other","name":"irrelevant"}`
						next = imageIdentityNext("marker=hidden-next&os_hidden=true")
					}
					testcloud.JSON(w, code, imageIdentityPage(rows, next))
				})
				value, err := f.new(imageIdentityClient(cloud))(context.Background(), "lookup", resource.WithIdentityFindIgnoreMissing(mode != "strict-empty-204"))
				wantNormal := int32(1)
				if mode == "match-300" {
					wantNormal = 2
				}
				if gets.Load() != 1 || normal.Load() != wantNormal || hidden.Load() == 0 {
					t.Fatal("hidden search phases changed", value, err, gets.Load(), normal.Load(), hidden.Load())
				}
				if mode == "match-300" {
					if err != nil || value == nil || value.ID != "lookup" || value.Name != "Different" || value.SizeBytes != 321 || !value.Hidden || hidden.Load() != 2 {
						t.Fatal(value, err, hidden.Load())
					}
					return
				}
				if mode == "empty-204" {
					if value != nil || err != nil || hidden.Load() != 1 {
						t.Fatal(value, err, hidden.Load())
					}
					return
				}
				if value != nil || err == nil {
					t.Fatal(value, err)
				}
				switch mode {
				case "duplicate":
					if !errors.Is(err, resource.ErrAmbiguous) || hidden.Load() != 2 {
						t.Fatal(err, hidden.Load())
					}
				case "late-http":
					if !gophercloud.ResponseCodeIs(err, 502) || hidden.Load() != 2 {
						t.Fatal(err, hidden.Load())
					}
				case "list-403", "list-404":
					status := 403
					if mode == "list-404" {
						status = 404
					}
					if !gophercloud.ResponseCodeIs(err, status) {
						t.Fatal("list absence was hidden", err)
					}
				case "strict-empty-204":
					if !errors.Is(err, resource.ErrNotFound) {
						t.Fatal(err)
					}
				}
			})
		}
	}
}

func TestNativeFindIdentityImagesQuerySnapshotsAndConcurrentReuse(t *testing.T) {
	for _, f := range imageIdentityFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, normal, hidden atomic.Int32
			query := url.Values{"name": nil, "tag": {"first", "second"}, "os_hidden": {"false"}}
			ignore := true
			option := resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: query, IgnoreMissing: &ignore})
			query["tag"][0], query["name"], ignore = "mutated", []string{"mutated"}, false
			var capturedMu sync.Mutex
			var captured *resource.IdentityFindOpts
			cloud.Mux.HandleFunc("GET "+imageIdentityPath+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if !reflect.DeepEqual(r.URL.Query()["tag"], []string{"first", "second"}) || r.URL.Query().Has("name") || r.URL.Query().Get("os_hidden") != "false" {
					t.Error("captured query escaped snapshot", r.URL)
				}
				capturedMu.Lock()
				if captured != nil {
					captured.Query["tag"][0] = "mutated-during-GET"
					captured.Query.Set("name", "mutated-during-GET")
					*captured.IgnoreMissing = false
				}
				capturedMu.Unlock()
				w.WriteHeader(404)
			})
			cloud.Mux.HandleFunc("GET "+imageIdentityPath, func(w http.ResponseWriter, r *http.Request) {
				if !reflect.DeepEqual(r.URL.Query()["tag"], []string{"first", "second"}) || r.URL.Query().Has("name") {
					t.Error("reused query lost values/presence", r.URL)
				}
				if r.URL.Query().Get("os_hidden") == "true" {
					hidden.Add(1)
				} else {
					normal.Add(1)
					if r.URL.Query().Get("os_hidden") != "false" {
						t.Error(r.URL)
					}
				}
				testcloud.JSON(w, 200, imageIdentityPage("", ""))
			})
			find := f.new(imageIdentityClient(cloud))
			var group sync.WaitGroup
			for range 8 {
				group.Add(1)
				go func() {
					defer group.Done()
					if value, err := find(context.Background(), "lookup", option); value != nil || err != nil {
						t.Error(value, err)
					}
				}()
			}
			group.Wait()
			if gets.Load() != 8 || normal.Load() != 8 || hidden.Load() != 8 {
				t.Fatal(gets.Load(), normal.Load(), hidden.Load())
			}
			// A custom option may retain its mutable configuration. The parsed
			// final snapshot must remain independent across both list phases.
			custom := func(options *resource.IdentityFindOpts) error {
				options.Query = url.Values{"name": nil, "tag": {"first", "second"}, "os_hidden": {"false"}}
				value := true
				options.IgnoreMissing = &value
				capturedMu.Lock()
				captured = options
				capturedMu.Unlock()
				return nil
			}
			if value, err := find(context.Background(), "lookup", custom); value != nil || err != nil {
				t.Fatal("captured custom configuration changed parsed search", value, err)
			}
			if gets.Load() != 9 || normal.Load() != 9 || hidden.Load() != 9 {
				t.Fatal(gets.Load(), normal.Load(), hidden.Load())
			}
		})
	}
}

type imageIdentityContextKey struct{}

func TestNativeFindIdentityImagesKeepsLiveProviderHeadersAndCallerContext(t *testing.T) {
	for _, f := range imageIdentityFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := imageIdentityClient(cloud)
			client.MoreHeaders = map[string]string{"X-Configured": "source"}
			endpoint, base := client.Endpoint, client.ResourceBase
			transport := cloud.Provider.HTTPClient.Transport
			if transport == nil {
				transport = http.DefaultTransport
			}
			var calls, lists atomic.Int32
			cloud.Provider.HTTPClient.Transport = nativeIdentityFindTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Context().Value(imageIdentityContextKey{}) != "caller" {
					t.Error("caller context lost")
				}
				copy := r.Clone(r.Context())
				copy.Header.Set("X-Image-Middleware", "source")
				return transport.RoundTrip(copy)
			})
			check := func(r *http.Request, token string) {
				if r.Header.Get("X-Auth-Token") != token || r.Header.Get("X-Configured") != "source" || r.Header.Get("X-Image-Middleware") != "source" || r.URL.Query().Get("vendor") != "kept" {
					t.Error(r.URL, r.Header)
				}
			}
			cloud.Mux.HandleFunc("GET "+imageIdentityPath+"/lookup", func(w http.ResponseWriter, r *http.Request) {
				check(r, "test-token")
				cloud.Provider.SetToken("normal-token")
				w.WriteHeader(404)
			})
			cloud.Mux.HandleFunc("GET "+imageIdentityPath, func(w http.ResponseWriter, r *http.Request) {
				if lists.Add(1) == 1 {
					check(r, "normal-token")
					cloud.Provider.SetToken("hidden-token")
					testcloud.JSON(w, 200, imageIdentityPage("", ""))
					return
				}
				check(r, "hidden-token")
				if r.URL.Query().Has("name") || r.URL.Query().Get("os_hidden") != "true" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, imageIdentityPage(`{"id":"found","name":"lookup"}`, ""))
			})
			ctx := context.WithValue(context.Background(), imageIdentityContextKey{}, "caller")
			value, err := f.new(client)(ctx, "lookup", resource.WithIdentityFindQuery("vendor", "kept"))
			if err != nil || value == nil || value.ID != "found" || calls.Load() != 3 || lists.Load() != 2 {
				t.Fatal(value, err, calls.Load(), lists.Load())
			}
			if client.Endpoint != endpoint || client.ResourceBase != base || !reflect.DeepEqual(client.MoreHeaders, map[string]string{"X-Configured": "source"}) {
				t.Fatal("lookup mutated the original service client", client)
			}
		})
	}
}

func TestNativeFindIdentityImagesPreflightUnsafeNamesAndCancellation(t *testing.T) {
	for _, f := range imageIdentityFixtures() {
		t.Run(f.name+"/preflight", func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			find := f.new(imageIdentityClient(cloud))
			canceled, cancel := context.WithCancel(context.Background())
			cancel()
			for _, test := range []struct {
				ctx      context.Context
				identity string
				opts     []resource.IdentityFindOption
				want     error
			}{
				{nil, "lookup", nil, resource.ErrInvalidOption},
				{canceled, "lookup", nil, context.Canceled},
				{context.Background(), " ", nil, resource.ErrInvalidOption},
				{context.Background(), "lookup", []resource.IdentityFindOption{nil}, resource.ErrInvalidOption},
				{context.Background(), "lookup", []resource.IdentityFindOption{resource.WithIdentityFindQuery("max_items", "1")}, resource.ErrInvalidOption},
				{context.Background(), "lookup", []resource.IdentityFindOption{resource.WithIdentityFindDetails(false)}, resource.ErrUnsupported},
				{context.Background(), "lookup", []resource.IdentityFindOption{resource.WithIdentityFindAllProjects(false)}, resource.ErrUnsupported},
				{context.Background(), "unsafe/name", []resource.IdentityFindOption{resource.WithIdentityFindFallback(resource.FindFallbackNever)}, resource.ErrInvalidOption},
			} {
				if value, err := find(test.ctx, test.identity, test.opts...); value != nil || !errors.Is(err, test.want) {
					t.Fatal(value, err, test.want)
				}
			}
			if calls.Load() != 0 {
				t.Fatal("invalid input performed HTTP", calls.Load())
			}
		})
		for _, identity := range []string{"unsafe/name", "literal%2Fname", " space name "} {
			t.Run(f.name+"/unsafe/"+identity, func(t *testing.T) {
				cloud := testcloud.New(t)
				var lists, gets atomic.Int32
				cloud.Mux.HandleFunc("GET "+imageIdentityPath, func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					if lists.Load() == 1 {
						if r.URL.Query().Get("name") != identity || r.URL.Query().Has("os_hidden") {
							t.Error("unsafe name changed", r.URL)
						}
						testcloud.JSON(w, 200, imageIdentityPage("", ""))
					} else {
						if r.URL.Query().Has("name") || r.URL.Query().Get("os_hidden") != "true" {
							t.Error(r.URL)
						}
						testcloud.JSON(w, 200, imageIdentityPage(nativeIdentityFindRow("found", identity), ""))
					}
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); w.WriteHeader(500) })
				value, err := f.new(imageIdentityClient(cloud))(context.Background(), identity)
				if err != nil || value == nil || value.Name != identity || lists.Load() != 2 || gets.Load() != 0 {
					t.Fatal(value, err, lists.Load(), gets.Load())
				}
			})
		}
		t.Run(f.name+"/cancel-after-normal-empty", func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var hidden atomic.Int32
			cloud.Mux.HandleFunc("GET "+imageIdentityPath+"/lookup", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) })
			cloud.Mux.HandleFunc("GET "+imageIdentityPath, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("os_hidden") == "true" {
					hidden.Add(1)
				}
				cancel()
				testcloud.JSON(w, 200, imageIdentityPage("", ""))
			})
			value, err := f.new(imageIdentityClient(cloud))(ctx, "lookup")
			if value != nil || !errors.Is(err, context.Canceled) || hidden.Load() != 0 {
				t.Fatal(value, err, hidden.Load())
			}
		})
	}
}
