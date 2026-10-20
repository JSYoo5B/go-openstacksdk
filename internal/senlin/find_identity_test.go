package senlin_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/senlin"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type findIdentityRow struct {
	resource.Metadata
	ID   string `json:"id"`
	Name string `json:"name"`
}

func findIdentitySpec(client *gophercloud.ServiceClient) rest.CollectionSpec[findIdentityRow] {
	return rest.CollectionSpec[findIdentityRow]{Client: client, Path: "items", Kind: "find-test", SingleKey: "item", PluralKey: "items", Get: true,
		ID: func(value *findIdentityRow) string { return value.ID }, Name: func(value *findIdentityRow) string { return value.Name },
		Metadata: func(value *findIdentityRow) *resource.Metadata { return &value.Metadata },
		Validate: func(ctx context.Context) error { return senlin.RequireVersion(ctx, client, 7) }, ValidateID: senlin.Identifier,
		Paging: rest.PagePolicy[findIdentityRow]{MaxItemsLimitHint: true, StopOnEmptyPage: true}}
}

func normalizeFindIdentity(value *findIdentityRow, id, name string) { value.ID, value.Name = id, name }

func findIdentityClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("clustering", "/reverse/v1/project")
	client.Microversion = "1.7"
	return client
}

func TestFindIdentityDirectGetRestoresCanonicalRawFieldsWithoutListing(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, lists atomic.Int32
	identity := "wanted.+[prod]"
	cloud.Mux.HandleFunc("GET /reverse/v1/project/items/{identity}", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if r.PathValue("identity") != identity || r.URL.RawQuery != "" {
			t.Error("identity escaped outside its fixed route", r.URL)
		}
		w.Header().Set("X-Request-ID", "get-proof")
		testcloud.JSON(w, 200, `{"item":{"id":"canonical","ID":"shadow","name":"wanted.+[prod]","NAME":"shadow-name","large":9007199254740993,"unknown":null}}`)
	})
	cloud.Mux.HandleFunc("GET /reverse/v1/project/items", func(w http.ResponseWriter, r *http.Request) { lists.Add(1); testcloud.JSON(w, 200, `{"items":[]}`) })
	value, err := senlin.FindIdentity(context.Background(), findIdentityClient(cloud), findIdentitySpec, normalizeFindIdentity, identity)
	if err != nil || value == nil || value.ID != "canonical" || value.Name != identity || value.StatusCode != 200 || value.Header.Get("X-Request-ID") != "get-proof" || string(value.Body["ID"]) != `"shadow"` || string(value.Body["NAME"]) != `"shadow-name"` || string(value.Body["large"]) != "9007199254740993" || string(value.Body["unknown"]) != "null" || gets.Load() != 1 || lists.Load() != 0 {
		t.Fatal("direct lookup lost canonical authority or evidence", value, err, gets.Load(), lists.Load())
	}
}

func TestFindIdentityFallbackPoliciesRetainDisallowedHTTPFailures(t *testing.T) {
	for _, status := range []int{400, 403, 404, 401, 409, 500, 503} {
		for _, policy := range []resource.FindFallbackPolicy{resource.FindFallbackCompatible, resource.FindFallbackNotFoundOnly, resource.FindFallbackNever} {
			t.Run(http.StatusText(status)+"/"+string(rune('0'+policy)), func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets, lists atomic.Int32
				cloud.Mux.HandleFunc("GET /reverse/v1/project/items/{identity}", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					w.Header().Set("X-Request-ID", "get-failure")
					testcloud.JSON(w, status, `{"error":"direct failure"}`)
				})
				cloud.Mux.HandleFunc("GET /reverse/v1/project/items", func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					if !reflect.DeepEqual(r.URL.Query(), url.Values{"name": {"wanted"}}) {
						t.Error("fallback name query is not owned and literal", r.URL)
					}
					testcloud.JSON(w, 200, `{"items":[{"id":"real","name":"wanted"}]}`)
				})
				value, err := senlin.FindIdentity(context.Background(), findIdentityClient(cloud), findIdentitySpec, normalizeFindIdentity, "wanted", senlin.WithFindFallback(policy), senlin.WithFindIgnoreMissing(false))
				allowed := (policy == resource.FindFallbackCompatible && (status == 400 || status == 403 || status == 404)) || (policy == resource.FindFallbackNotFoundOnly && status == 404)
				if allowed {
					if err != nil || value == nil || value.ID != "real" || gets.Load() != 1 || lists.Load() != 1 {
						t.Fatal(value, err, gets.Load(), lists.Load())
					}
					return
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.As(err, &native) || native.Actual != status || native.ResponseHeader.Get("X-Request-ID") != "get-failure" || string(native.Body) != `{"error":"direct failure"}` || gets.Load() != 1 || lists.Load() != 0 {
					t.Fatal("disallowed status triggered fallback or lost proof", value, err, native, gets.Load(), lists.Load())
				}
				if status == 404 && !errors.Is(err, resource.ErrNotFound) {
					t.Fatal("strict direct absence is not NotFound", err)
				}
			})
		}
	}
}

func TestFindIdentityAcceptedGETDecodeFailuresNeverFallBack(t *testing.T) {
	for _, body := range []string{
		`{"item":null}`, `{"items":[]}`, `{"item":[]}`, `{"item":{"name":"wanted"}}`,
		`{"item":{"id":null}}`, `{"item":{"id":""}}`, `{"item":{"id":"bad/id"}}`,
		`{"item":{"id":"real","name":false}}`, `{"item":{"id":"real","ID":123}}`, `{"item":`,
	} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, lists atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/v1/project/items/{identity}", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				w.Header().Set("X-Request-ID", "accepted-get")
				testcloud.JSON(w, 200, body)
			})
			cloud.Mux.HandleFunc("GET /reverse/v1/project/items", func(w http.ResponseWriter, r *http.Request) { lists.Add(1); testcloud.JSON(w, 200, `{"items":[]}`) })
			value, err := senlin.FindIdentity(context.Background(), findIdentityClient(cloud), findIdentitySpec, normalizeFindIdentity, "wanted")
			var proof *resource.ResponseError
			if value != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-ID") != "accepted-get" || string(proof.Body) != body || gets.Load() != 1 || lists.Load() != 0 {
				t.Fatal("accepted malformed GET was suppressed or resent", value, err, proof, gets.Load(), lists.Load())
			}
		})
	}
}

func TestFindIdentityScansRemainingPagesForExactIDDuplicatesAndLateFailures(t *testing.T) {
	for _, mode := range []string{"exact-id", "duplicate", "late-http", "malformed-row", "foreign-link", "validator", "canonical-shadow"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, lists atomic.Int32
			body := `{"items":[{"id":"real","name":"wanted"}],"links":[{"rel":"next","href":"?marker=real"}]}`
			if mode == "malformed-row" {
				body = `{"items":[{"id":"real","name":"wanted"},{"name":"other"}],"links":false}`
			} else if mode == "foreign-link" {
				body = `{"items":[{"id":"real","name":"wanted"}],"links":[{"rel":"next","href":"https://foreign.invalid/items"}]}`
			} else if mode == "validator" {
				body = `{"items":[{"id":"real","name":"wanted"}]}`
			} else if mode == "canonical-shadow" {
				body = `{"items":[{"id":"other","ID":"wanted","name":"other","NAME":"wanted"}]}`
			}
			cloud.Mux.HandleFunc("GET /reverse/v1/project/items/{identity}", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 404, `{"error":"missing"}`)
			})
			second := `{"items":[{"id":"wanted","name":"different","ID":"shadow","NAME":"shadow-name","large":9007199254740993}]}`
			cloud.Mux.HandleFunc("GET /reverse/v1/project/items", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("label") != "literal:wanted" || r.URL.Query().Has("name") {
					t.Error("factory-owned name query lost", r.URL)
				}
				w.Header().Set("X-Request-ID", "list-proof")
				if r.URL.Query().Get("marker") == "" {
					if mode == "exact-id" {
						testcloud.JSON(w, 200, `{"items":[{"id":"other","name":"different"}],"links":[{"rel":"next","href":"?marker=real"}]}`)
					} else {
						testcloud.JSON(w, 200, body)
					}
				} else if mode == "late-http" {
					testcloud.JSON(w, 503, `{"error":"late failure"}`)
				} else {
					testcloud.JSON(w, 200, second)
				}
			})
			validation := errors.New("wrong fixed scope")
			factory := func(client *gophercloud.ServiceClient) rest.CollectionSpec[findIdentityRow] {
				spec := findIdentitySpec(client)
				spec.NameQueryKey = "label"
				spec.NameQuery = func(identity string) string { return "literal:" + identity }
				if mode == "validator" {
					spec.ValidateItem = func(*findIdentityRow) error { return validation }
				}
				return spec
			}
			value, err := senlin.FindIdentity(context.Background(), findIdentityClient(cloud), factory, normalizeFindIdentity, "wanted", senlin.WithFindIgnoreMissing(false))
			if mode == "exact-id" {
				if err != nil || value == nil || value.ID != "wanted" || value.Name != "different" || string(value.Body["ID"]) != `"shadow"` || string(value.Body["large"]) != "9007199254740993" || gets.Load() != 1 || lists.Load() != 2 {
					t.Fatal("ID match was removed by a name-only filter", value, err, lists.Load())
				}
				return
			}
			if mode == "canonical-shadow" {
				if value != nil || !errors.Is(err, resource.ErrNotFound) || lists.Load() != 1 {
					t.Fatal("case-variant shadow produced a false match", value, err, lists.Load())
				}
				return
			}
			if mode == "late-http" {
				if value != nil || !gophercloud.ResponseCodeIs(err, 503) || lists.Load() != 2 {
					t.Fatal("first match hid a later HTTP failure", value, err, lists.Load())
				}
				return
			}
			var proof *resource.ResponseError
			wantBody, wantLists := body, int32(1)
			if mode == "duplicate" {
				wantBody, wantLists = second, 2
				var ambiguous *resource.AmbiguousError
				if !errors.As(err, &ambiguous) || !reflect.DeepEqual(ambiguous.IDs, []string{"real", "wanted"}) {
					t.Fatal("exact ID/name duplicates were not detected", err, ambiguous)
				}
			}
			if value != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-ID") != "list-proof" || string(proof.Body) != wantBody || gets.Load() != 1 || lists.Load() != wantLists {
				t.Fatal("list failure lost whole-page evidence", value, err, proof, lists.Load())
			}
			if mode == "validator" && !errors.Is(err, validation) {
				t.Fatal("original row validator was bypassed", err)
			}
		})
	}
}

func TestFindIdentityMissingPolicyNeverHidesFallbackHTTPOrAddsSuppressedCause(t *testing.T) {
	for _, mode := range []string{"default-empty", "strict-empty", "fallback-forbidden", "fallback-not-found", "never-default", "last-wins", "input-snapshot"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, lists atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/v1/project/items/{identity}", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				status := 403
				if mode == "never-default" {
					status = 404
				}
				testcloud.JSON(w, status, `{"error":"direct failure"}`)
			})
			cloud.Mux.HandleFunc("GET /reverse/v1/project/items", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				status, body := 200, `{"items":[]}`
				if mode == "fallback-forbidden" {
					status, body = 403, `{"error":"list forbidden"}`
				} else if mode == "fallback-not-found" {
					status, body = 404, `{"error":"list missing"}`
				}
				testcloud.JSON(w, status, body)
			})
			var options []senlin.FindOption
			ignored := true
			var captured *request.Config[senlin.FindOpts]
			if mode == "strict-empty" {
				options = append(options, senlin.WithFindIgnoreMissing(false))
			} else if mode == "never-default" {
				options = append(options, senlin.WithFindFallback(resource.FindFallbackNever))
			} else if mode == "last-wins" {
				options = append(options, senlin.WithFindIgnoreMissing(false), senlin.WithFindIgnoreMissing(true), senlin.WithFindFallback(-1), senlin.WithFindFallback(resource.FindFallbackCompatible), senlin.WithFindMicroversion("1.07"), senlin.WithFindMicroversion("1.7"))
			} else if mode == "input-snapshot" {
				options = append(options, request.WithOptions(senlin.FindOpts{IgnoreMissing: &ignored}), func(config *request.Config[senlin.FindOpts]) error { captured = config; return nil })
				native := cloud.Provider.HTTPClient.Transport
				cloud.Provider.HTTPClient.Transport = findIdentityTransport(func(r *http.Request) (*http.Response, error) {
					response, err := native.RoundTrip(r)
					if strings.HasSuffix(r.URL.Path, "/items/wanted") {
						ignored = false
						captured.Options.Fallback = resource.FindFallbackNever
					}
					return response, err
				})
			}
			value, err := senlin.FindIdentity(context.Background(), findIdentityClient(cloud), findIdentitySpec, normalizeFindIdentity, "wanted", options...)
			if mode == "strict-empty" {
				var missing *resource.NotFoundError
				if value != nil || !errors.As(err, &missing) || missing.Cause != nil || gophercloud.ResponseCodeIs(err, 403) || lists.Load() != 1 {
					t.Fatal("logical absence retained suppressed forbidden cause", value, err, missing, lists.Load())
				}
			} else if mode == "fallback-forbidden" || mode == "fallback-not-found" {
				status := 403
				if mode == "fallback-not-found" {
					status = 404
				}
				if value != nil || !gophercloud.ResponseCodeIs(err, status) || lists.Load() != 1 {
					t.Fatal("ignoreMissing hid a fallback list failure", value, err, lists.Load())
				}
			} else if value != nil || err != nil || (mode == "never-default" && lists.Load() != 0) || (mode != "never-default" && lists.Load() != 1) {
				t.Fatal("default or final missing policy changed", value, err, lists.Load())
			}
			if gets.Load() != 1 {
				t.Fatal("direct GET was retried implicitly", gets.Load())
			}
		})
	}
}

type findIdentityTransport func(*http.Request) (*http.Response, error)

func (f findIdentityTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFindIdentityOwnsRequestSnapshotAcrossReauthAndBothPhases(t *testing.T) {
	cloud := testcloud.New(t)
	source := findIdentityClient(cloud)
	source.Microversion = "1.2"
	source.MoreHeaders = map[string]string{"x-find": "source", "X-Source": "kept"}
	var calls, reauth atomic.Int32
	var captured request.Config[senlin.FindOpts]
	ignored := true
	check := func(r *http.Request, wantToken string) {
		if r.Header.Get("X-Find") != "request" || r.Header.Get("X-Source") != "kept" || r.Header.Get("Accept") != "application/vnd.find+json" || r.Header.Get("OpenStack-API-Version") != "clustering 1.7" || r.Header.Get("X-Auth-Token") != wantToken {
			t.Error("request settings changed across lookup phases", r.URL, r.Header)
		}
	}
	cloud.Mux.HandleFunc("GET /reverse/v1/project/items/{identity}", func(w http.ResponseWriter, r *http.Request) {
		attempt := calls.Add(1)
		if r.PathValue("identity") != "wanted" || r.URL.RawQuery != "" {
			t.Error("direct identity route changed", r.URL)
		}
		if attempt == 1 {
			check(r, "test-token")
			testcloud.JSON(w, 401, `{"error":"expired"}`)
		} else {
			check(r, "fresh-token")
			testcloud.JSON(w, 403, `{"error":"controller name lookup forbidden"}`)
		}
	})
	cloud.Mux.HandleFunc("GET /reverse/v1/project/items", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("name") != "wanted" || r.URL.Query().Has("limit") || r.URL.Query().Has("microversion") || r.URL.Query().Has("headers") {
			t.Error("Find controls leaked or list was capped", r.URL)
		}
		if r.URL.Query().Get("marker") == "" {
			check(r, "fallback-token")
			w.Header().Set("X-Request-ID", "first-page")
			testcloud.JSON(w, 200, `{"items":[{"id":"real","name":"wanted","large":9007199254740993}],"links":[{"rel":"next","href":"?marker=real"}]}`)
		} else {
			check(r, "page-token")
			testcloud.JSON(w, 200, `{"items":[{"id":"other","name":null}]}`)
		}
	})
	cloud.Provider.ReauthFunc = func(ctx context.Context) error {
		reauth.Add(1)
		captured.Headers["X-Find"] = "changed-input"
		captured.Arguments["senlin.find.microversion"] = "1.1"
		ignored = false
		cloud.Provider.SetToken("fresh-token")
		return nil
	}
	native := cloud.Provider.HTTPClient.Transport
	cloud.Provider.HTTPClient.Transport = findIdentityTransport(func(r *http.Request) (*http.Response, error) {
		response, err := native.RoundTrip(r)
		if err == nil && strings.HasSuffix(r.URL.Path, "/items/wanted") && response.StatusCode == 403 {
			source.MoreHeaders["x-find"] = "changed-source"
			cloud.Provider.SetToken("fallback-token")
		} else if err == nil && strings.HasSuffix(r.URL.Path, "/items") && r.URL.Query().Get("marker") == "" {
			cloud.Provider.SetToken("page-token")
		}
		return response, err
	})
	value, err := senlin.FindIdentity(context.Background(), source, findIdentitySpec, normalizeFindIdentity, "wanted",
		request.WithOptions(senlin.FindOpts{IgnoreMissing: &ignored}),
		senlin.WithFindHeader("x-find", "earlier"), senlin.WithFindHeader("X-FIND", "request"), senlin.WithFindHeader("Accept", "application/vnd.find+json"),
		senlin.WithFindMicroversion("1.7"), func(config *request.Config[senlin.FindOpts]) error { captured = *config; return nil })
	if err != nil || value == nil || value.ID != "real" || value.Name != "wanted" || string(value.Body["large"]) != "9007199254740993" || value.Header.Get("X-Request-ID") != "first-page" || calls.Load() != 4 || reauth.Load() != 1 || source.Microversion != "1.2" || len(source.MoreHeaders) != 2 || cloud.Provider.Token() != "page-token" {
		t.Fatal("shared client or request snapshot changed", value, err, calls.Load(), reauth.Load(), source)
	}
}

func TestFindIdentityPreflightRejectsForeignControlsAndInvalidFactories(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/v1/project/items/{identity}", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"item":{"id":"real"}}`)
	})
	for _, option := range []senlin.FindOption{
		nil, senlin.WithFindFallback(-1), senlin.WithFindFallback(3), senlin.WithFindMicroversion("latest"), senlin.WithFindMicroversion("1.07"),
		request.WithArgument[senlin.FindOpts]("senlin.find.microversion", 7),
		request.WithArgument[senlin.FindOpts]("senlin.list.microversion", "1.7"),
		request.WithArgument[senlin.FindOpts]("items.local_filters", map[string]any{}),
		request.WithField[senlin.FindOpts]("vendor", true), request.WithQuery[senlin.FindOpts]("name", "other"),
		request.WithQuery[senlin.FindOpts]("base_path", "/foreign"), request.WithQuery[senlin.FindOpts]("max_items", "1"),
		senlin.WithFindHeader("X-Auth-Token", "foreign"), senlin.WithFindHeader("OpenStack-API-Version", "clustering 1.7"), senlin.WithFindHeader("Cookie", "foreign"),
		senlin.WithFindHeader("X-Vendor", "bad\nvalue"),
	} {
		value, err := senlin.FindIdentity(context.Background(), findIdentityClient(cloud), findIdentitySpec, normalizeFindIdentity, "wanted", option)
		var operation *resource.OperationError
		if value != nil || err == nil || (!errors.Is(err, resource.ErrInvalidOption) && !errors.Is(err, resource.ErrUnsupported)) || !errors.As(err, &operation) || operation.Operation != "FindIdentity" || operation.Resource != "find-test" || calls.Load() != 0 {
			t.Fatal("foreign controls crossed Find boundary", value, err, calls.Load())
		}
	}
	for _, identity := range []string{"", " ", ".", "..", "bad/id", "bad\\id", "bad?query", "bad#fragment", "bad%2Fid", "bad name"} {
		if _, err := senlin.FindIdentity(context.Background(), findIdentityClient(cloud), findIdentitySpec, normalizeFindIdentity, identity); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal("invalid identity was sent", identity, err, calls.Load())
		}
	}
	for _, source := range []*gophercloud.ServiceClient{nil, {Type: "clustering"}, {Type: "compute", ProviderClient: cloud.Provider}} {
		if _, err := senlin.FindIdentity(context.Background(), source, findIdentitySpec, normalizeFindIdentity, "wanted"); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal("invalid source was sent", err, calls.Load())
		}
	}
	if _, err := senlin.FindIdentity[findIdentityRow](context.Background(), findIdentityClient(cloud), nil, normalizeFindIdentity, "wanted"); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
		t.Fatal("nil factory was sent", err, calls.Load())
	}
	if _, err := senlin.FindIdentity(context.Background(), findIdentityClient(cloud), findIdentitySpec, nil, "wanted"); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
		t.Fatal("nil normalizer was sent", err, calls.Load())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := senlin.FindIdentity(ctx, findIdentityClient(cloud), findIdentitySpec, normalizeFindIdentity, "wanted"); !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatal("canceled lookup was sent", err, calls.Load())
	}
}

type findIdentityReadFailure struct {
	first bool
	cause error
}

func (body *findIdentityReadFailure) Read(buffer []byte) (int, error) {
	if !body.first {
		body.first = true
		return copy(buffer, `{"item":`), nil
	}
	return 0, body.cause
}

func (*findIdentityReadFailure) Close() error { return nil }

func TestFindIdentityAcceptedReadAndTransportErrorsNeverTriggerFallback(t *testing.T) {
	for _, mode := range []string{"accepted-read", "accepted-read-native-cause", "transport", "transport-native-cause"} {
		t.Run(mode, func(t *testing.T) {
			var calls int
			cause := error(io.ErrUnexpectedEOF)
			if strings.HasSuffix(mode, "native-cause") {
				cause = gophercloud.ErrUnexpectedResponseCode{Actual: 404, Body: []byte(`{"error":"nested cause"}`)}
			}
			provider := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: findIdentityTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Path != "/v1/project/items/wanted" {
					t.Error("failure unexpectedly invoked fallback", r.URL)
				}
				if strings.HasPrefix(mode, "transport") {
					return nil, cause
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"X-Request-Id": {"read-proof"}}, Body: &findIdentityReadFailure{cause: cause}}, nil
			})}}
			provider.UseTokenLock()
			provider.SetToken("token")
			source := &gophercloud.ServiceClient{ProviderClient: provider, Endpoint: "https://service.test/v1/project/", Type: "clustering", Microversion: "1.7"}
			value, err := senlin.FindIdentity(context.Background(), source, findIdentitySpec, normalizeFindIdentity, "wanted")
			if value != nil || !errors.Is(err, cause) || calls != 1 {
				// Native error values contain slices and do not implement equality;
				// their exact status remains available through errors.As instead.
				var native gophercloud.ErrUnexpectedResponseCode
				if value != nil || calls != 1 || !strings.HasSuffix(mode, "native-cause") || !errors.As(err, &native) || native.Actual != 404 {
					t.Fatal("failure was suppressed or retried", value, err, calls)
				}
			}
			if strings.HasPrefix(mode, "accepted-read") {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-ID") != "read-proof" || string(proof.Body) != `{"item":` {
					t.Fatal("accepted read lost its response evidence", err, proof)
				}
			}
		})
	}
}

func TestFindIdentityRechecksOriginalSourceEffectiveGatesAndCancellation(t *testing.T) {
	for _, mode := range []string{"phase-source", "phase-cancel", "page-source", "page-cancel", "effective-upgrade", "uncontrolled-downshift"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			source := findIdentityClient(cloud)
			var calls atomic.Int32
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cloud.Mux.HandleFunc("GET /reverse/v1/project/items/{identity}", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 403, `{"error":"direct failure"}`)
			})
			cloud.Mux.HandleFunc("GET /reverse/v1/project/items", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("marker") == "" {
					w.Header().Set("X-Request-ID", "cancel-proof")
					testcloud.JSON(w, 200, `{"items":[{"id":"real","name":"wanted"}],"links":[{"rel":"next","href":"?marker=real"}]}`)
				} else {
					testcloud.JSON(w, 200, `{"items":[{"id":"other","name":"different"}]}`)
				}
			})
			native := cloud.Provider.HTTPClient.Transport
			cloud.Provider.HTTPClient.Transport = findIdentityTransport(func(r *http.Request) (*http.Response, error) {
				response, err := native.RoundTrip(r)
				if err != nil {
					return response, err
				}
				if strings.HasSuffix(r.URL.Path, "/items/wanted") {
					switch mode {
					case "phase-source":
						source.Type = "compute"
					case "phase-cancel":
						cancel()
					case "effective-upgrade", "uncontrolled-downshift":
						source.Microversion = "1.0"
					}
				} else if mode == "page-source" && r.URL.Query().Get("marker") == "" {
					source.Type = "compute"
				}
				return response, nil
			})
			var options []senlin.FindOption
			if mode != "uncontrolled-downshift" {
				options = append(options, senlin.WithFindMicroversion("1.7"))
			}
			normalize := normalizeFindIdentity
			if mode == "page-cancel" {
				normalize = func(value *findIdentityRow, id, name string) { normalizeFindIdentity(value, id, name); cancel() }
			}
			value, err := senlin.FindIdentity(ctx, source, findIdentitySpec, normalize, "wanted", options...)
			if mode == "effective-upgrade" {
				if err != nil || value == nil || value.ID != "real" || calls.Load() != 3 {
					t.Fatal("valid source lower version denied effective gate", value, err, calls.Load())
				}
				return
			}
			wantError, wantCalls := resource.ErrInvalidOption, int32(1)
			if mode == "phase-cancel" || mode == "page-cancel" {
				wantError = context.Canceled
			} else if mode == "uncontrolled-downshift" {
				wantError = resource.ErrUnsupported
			}
			if strings.HasPrefix(mode, "page-") {
				wantCalls = 2
			}
			if value != nil || !errors.Is(err, wantError) || calls.Load() != wantCalls {
				t.Fatal("phase/page preflight failed to stop lookup", value, err, calls.Load())
			}
			if mode == "page-cancel" {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-ID") != "cancel-proof" || string(proof.Body) != `{"items":[{"id":"real","name":"wanted"}],"links":[{"rel":"next","href":"?marker=real"}]}` {
					t.Fatal("accepted terminal cancellation lost whole-page proof", err, proof)
				}
			}
		})
	}
}
