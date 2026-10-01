package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/clustering/v1/clusters"
	"gophercloudsdk/clustering/v1/nodes"
	"gophercloudsdk/clustering/v1/policies"
	"gophercloudsdk/clustering/v1/profiles"
	"gophercloudsdk/clustering/v1/receivers"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// Source: pinned ef55d7d Resource.find resource.py:2536-2554 tries GET before
// falling back on 400/403/404; :2439-2456 matches ID OR name and detects a
// second match; :2571-2582 scans the list before applying ignore_missing.
// Senlin declares find_profile/cluster/node/policy/receiver, each with a True
// ignore_missing default (_proxy.py:164,308,756,974,1162). Per-call controls on
// both phases and strict canonical response identities are owned Go policies.
const clusteringFindIdentityPrefix = "/proxy/project/senlin/v1"

type clusteringFindIdentityInput struct {
	missing   *bool
	fallbacks []resource.FindFallbackPolicy
	headers   [][2]string
	versions  []string
}

type clusteringFindIdentityValue struct {
	id, name string
	metadata *resource.Metadata
}

type clusteringFindIdentityAccess struct {
	find     func(context.Context, string, clusteringFindIdentityInput) (*clusteringFindIdentityValue, error)
	explicit func(context.Context, resource.Ref, ...resource.LookupOption) (*clusteringFindIdentityValue, error)
}

func clusteringFindIdentityAdapter[O, M any](
	find func(context.Context, string, ...request.Option[O]) (*M, error),
	explicit func(context.Context, resource.Ref, ...resource.LookupOption) (*M, error),
	missing func(bool) request.Option[O], fallback func(resource.FindFallbackPolicy) request.Option[O],
	header func(string, string) request.Option[O], version func(string) request.Option[O],
	value func(*M) *clusteringFindIdentityValue,
) clusteringFindIdentityAccess {
	return clusteringFindIdentityAccess{
		find: func(ctx context.Context, identity string, input clusteringFindIdentityInput) (*clusteringFindIdentityValue, error) {
			options := make([]request.Option[O], 0)
			if input.missing != nil {
				options = append(options, missing(*input.missing))
			}
			for _, policy := range input.fallbacks {
				options = append(options, fallback(policy))
			}
			for _, pair := range input.headers {
				options = append(options, header(pair[0], pair[1]))
			}
			for _, selected := range input.versions {
				options = append(options, version(selected))
			}
			result, err := find(ctx, identity, options...)
			if result == nil {
				return nil, err
			}
			return value(result), err
		},
		explicit: func(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*clusteringFindIdentityValue, error) {
			result, err := explicit(ctx, ref, options...)
			if result == nil {
				return nil, err
			}
			return value(result), err
		},
	}
}

type clusteringFindIdentityFixture struct {
	path, singular, plural string
	new                    func(*gophercloud.ServiceClient) clusteringFindIdentityAccess
}

func clusteringFindIdentityFixtures() []clusteringFindIdentityFixture {
	return []clusteringFindIdentityFixture{
		{"profiles", "profile", "profiles", func(client *gophercloud.ServiceClient) clusteringFindIdentityAccess {
			api := profiles.New(client)
			return clusteringFindIdentityAdapter(api.FindIdentity, api.Find, profiles.WithFindIgnoreMissing, profiles.WithFindFallback, profiles.WithFindHeader, profiles.WithFindMicroversion, func(v *profiles.Profile) *clusteringFindIdentityValue {
				return &clusteringFindIdentityValue{v.ID, v.Name, &v.Metadata}
			})
		}},
		{"policies", "policy", "policies", func(client *gophercloud.ServiceClient) clusteringFindIdentityAccess {
			api := policies.New(client)
			return clusteringFindIdentityAdapter(api.FindIdentity, api.Find, policies.WithFindIgnoreMissing, policies.WithFindFallback, policies.WithFindHeader, policies.WithFindMicroversion, func(v *policies.Policy) *clusteringFindIdentityValue {
				return &clusteringFindIdentityValue{v.ID, v.Name, &v.Metadata}
			})
		}},
		{"clusters", "cluster", "clusters", func(client *gophercloud.ServiceClient) clusteringFindIdentityAccess {
			api := clusters.New(client)
			return clusteringFindIdentityAdapter(api.FindIdentity, api.Find, clusters.WithFindIgnoreMissing, clusters.WithFindFallback, clusters.WithFindHeader, clusters.WithFindMicroversion, func(v *clusters.Cluster) *clusteringFindIdentityValue {
				return &clusteringFindIdentityValue{v.ID, v.Name, &v.Metadata}
			})
		}},
		{"nodes", "node", "nodes", func(client *gophercloud.ServiceClient) clusteringFindIdentityAccess {
			api := nodes.New(client)
			return clusteringFindIdentityAdapter(api.FindIdentity, api.Find, nodes.WithFindIgnoreMissing, nodes.WithFindFallback, nodes.WithFindHeader, nodes.WithFindMicroversion, func(v *nodes.Node) *clusteringFindIdentityValue {
				return &clusteringFindIdentityValue{v.ID, v.Name, &v.Metadata}
			})
		}},
		{"receivers", "receiver", "receivers", func(client *gophercloud.ServiceClient) clusteringFindIdentityAccess {
			api := receivers.New(client)
			return clusteringFindIdentityAdapter(api.FindIdentity, api.Find, receivers.WithFindIgnoreMissing, receivers.WithFindFallback, receivers.WithFindHeader, receivers.WithFindMicroversion, func(v *receivers.Receiver) *clusteringFindIdentityValue {
				return &clusteringFindIdentityValue{v.ID, v.Name, &v.Metadata}
			})
		}},
	}
}

func clusteringFindIdentityCloud(t *testing.T) (*testcloud.Cloud, *gophercloud.ServiceClient) {
	t.Helper()
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/catalog/v1")
	client.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + clusteringFindIdentityPrefix)
	return cloud, client
}

func clusteringFindIdentityObject(fixture clusteringFindIdentityFixture, body string) string {
	return fmt.Sprintf(`{%q:%s}`, fixture.singular, body)
}

func clusteringFindIdentityPage(fixture clusteringFindIdentityFixture, rows, next string) string {
	if next == "" {
		return fmt.Sprintf(`{%q:%s}`, fixture.plural, rows)
	}
	return fmt.Sprintf(`{%q:%s,%q:[{"rel":"next","href":%q}]}`, fixture.plural, rows, fixture.plural+"_links", next)
}

func clusteringFindIdentityAssertGET(t *testing.T, r *http.Request) {
	t.Helper()
	if r.Method != http.MethodGet || r.ContentLength > 0 || r.Header.Get("X-Auth-Token") == "" {
		t.Error("find changed method/body or lost shared authentication", r.Method, r.ContentLength, r.Header)
	}
}

func TestClusteringFindIdentityFiveFacadesDirectGETAndCanonicalFields(t *testing.T) {
	for _, fixture := range clusteringFindIdentityFixtures() {
		t.Run(fixture.path, func(t *testing.T) {
			cloud, client := clusteringFindIdentityCloud(t)
			var gets, lists atomic.Int32
			path := clusteringFindIdentityPrefix + "/" + fixture.path
			cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				t.Error("successful identity GET must not list", r.URL)
				testcloud.JSON(w, 200, clusteringFindIdentityPage(fixture, `[]`, ""))
			})
			cloud.Mux.HandleFunc("GET "+path+"/", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				clusteringFindIdentityAssertGET(t, r)
				if r.URL.RawQuery != "" || r.Header.Get("X-Find") != "owned" || r.Header.Get("OpenStack-API-Version") != "clustering 1.7" {
					t.Error(r.URL, r.Header)
				}
				name := `,"name":"canonical-name","Name":"case-shadow"`
				if strings.HasSuffix(r.URL.Path, "/null-name") {
					name = `,"name":null,"Name":"case-shadow"`
				} else if strings.HasSuffix(r.URL.Path, "/omitted-name") {
					name = `,"Name":"case-shadow"`
				}
				w.Header().Set("X-Request-ID", "direct-find")
				testcloud.JSON(w, 200, clusteringFindIdentityObject(fixture, `{"id":"canonical-id","ID":"case-shadow"`+name+`,"future":9007199254740993}`))
			})
			access := fixture.new(client)
			for _, identity := range []string{"short-id", "12345678-1234-1234-1234-123456789012", "null-name", "omitted-name"} {
				value, err := access.find(context.Background(), identity, clusteringFindIdentityInput{headers: [][2]string{{"X-Find", "owned"}}, versions: []string{"1.7"}})
				wantName := "canonical-name"
				if identity == "null-name" || identity == "omitted-name" {
					wantName = ""
				}
				if err != nil || value == nil || value.id != "canonical-id" || value.name != wantName || value.metadata.StatusCode != 200 || value.metadata.Header.Get("X-Request-ID") != "direct-find" || string(value.metadata.Body["future"]) != "9007199254740993" || string(value.metadata.Body["ID"]) != `"case-shadow"` {
					t.Fatal("canonical response fields or original evidence lost", identity, value, err)
				}
			}
			if gets.Load() != 4 || lists.Load() != 0 || client.Microversion != "" || len(client.MoreHeaders) != 0 {
				t.Fatal(gets.Load(), lists.Load(), client.Microversion, client.MoreHeaders)
			}
		})
	}
}

func TestClusteringFindIdentityFallbackPoliciesAndNativeStatuses(t *testing.T) {
	for _, fixture := range clusteringFindIdentityFixtures() {
		for _, policy := range []resource.FindFallbackPolicy{resource.FindFallbackCompatible, resource.FindFallbackNotFoundOnly, resource.FindFallbackNever} {
			for _, code := range []int{400, 403, 404, 401, 409, 503, 201} {
				t.Run(fmt.Sprintf("%s/policy=%v/code=%d", fixture.path, policy, code), func(t *testing.T) {
					cloud, client := clusteringFindIdentityCloud(t)
					var gets, lists atomic.Int32
					path := clusteringFindIdentityPrefix + "/" + fixture.path
					cloud.Mux.HandleFunc("GET "+path+"/input", func(w http.ResponseWriter, r *http.Request) {
						gets.Add(1)
						clusteringFindIdentityAssertGET(t, r)
						w.Header().Set("X-Native", "direct-error")
						testcloud.JSON(w, code, `{"error":{"message":"direct-error"}}`)
					})
					cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
						lists.Add(1)
						clusteringFindIdentityAssertGET(t, r)
						if !reflect.DeepEqual(r.URL.Query()["name"], []string{"input"}) || len(r.URL.Query()) != 1 {
							t.Error("fallback did not preserve literal name query", r.URL)
						}
						testcloud.JSON(w, 200, clusteringFindIdentityPage(fixture, `[{"id":"selected-id","name":"input"}]`, ""))
					})
					value, err := fixture.new(client).find(context.Background(), "input", clusteringFindIdentityInput{fallbacks: []resource.FindFallbackPolicy{policy}})
					fallback := policy == resource.FindFallbackCompatible && (code == 400 || code == 403 || code == 404) || policy == resource.FindFallbackNotFoundOnly && code == 404
					wantGets := int32(1)
					if fallback {
						if err != nil || value == nil || value.id != "selected-id" || lists.Load() != 1 {
							t.Fatal("permitted fallback failed", value, err, lists.Load())
						}
					} else if policy == resource.FindFallbackNever && code == 404 {
						if value != nil || err != nil || lists.Load() != 0 {
							t.Fatal("disabled fallback did not apply default ignore_missing", value, err, lists.Load())
						}
						strict := false
						value, err = fixture.new(client).find(context.Background(), "input", clusteringFindIdentityInput{missing: &strict, fallbacks: []resource.FindFallbackPolicy{policy}})
						if value != nil || !errors.Is(err, resource.ErrNotFound) || !gophercloud.ResponseCodeIs(err, 404) || lists.Load() != 0 {
							t.Fatal("strict disabled fallback lost direct 404 cause", value, err, lists.Load())
						}
						wantGets = 2
					} else if value != nil || !gophercloud.ResponseCodeIs(err, code) || lists.Load() != 0 {
						t.Fatal("native direct error was swallowed or listed", value, err, lists.Load())
					}
					if gets.Load() != wantGets {
						t.Fatal("direct lookup retried", gets.Load())
					}
				})
			}
		}
	}
}

func TestClusteringFindIdentityRawIDOrNameMatchingAndMissingDefaults(t *testing.T) {
	for _, fixture := range clusteringFindIdentityFixtures() {
		t.Run(fixture.path, func(t *testing.T) {
			cloud, client := clusteringFindIdentityCloud(t)
			var gets, lists atomic.Int32
			path := clusteringFindIdentityPrefix + "/" + fixture.path
			cloud.Mux.HandleFunc("GET "+path+"/", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 404, `{"error":{"message":"not-an-id"}}`)
			})
			cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				rows := `[{"id":"prefix-only","name":"missing-extra"},{"id":"different-case","name":"MISSING"}]`
				switch r.URL.Query().Get("name") {
				case "id-match":
					rows = `[{"id":"id-match","ID":"shadow","name":"another-name","Name":"id-match"}]`
				case "name-match":
					rows = `[{"id":"different-id","ID":"shadow","name":"name-match","Name":"other"}]`
				}
				testcloud.JSON(w, 200, clusteringFindIdentityPage(fixture, rows, ""))
			})
			access := fixture.new(client)
			for _, tc := range []struct{ input, id, name string }{{"id-match", "id-match", "another-name"}, {"name-match", "different-id", "name-match"}} {
				value, err := access.find(context.Background(), tc.input, clusteringFindIdentityInput{})
				if err != nil || value == nil || value.id != tc.id || value.name != tc.name || string(value.metadata.Body["ID"]) != `"shadow"` {
					t.Fatal("raw identity OR exact name matching failed", tc, value, err)
				}
			}
			if value, err := access.find(context.Background(), "missing", clusteringFindIdentityInput{}); value != nil || err != nil {
				t.Fatal("default missing must be nil,nil", value, err)
			}
			strict := false
			if value, err := access.find(context.Background(), "missing", clusteringFindIdentityInput{missing: &strict}); value != nil || !errors.Is(err, resource.ErrNotFound) {
				t.Fatal("strict missing option ignored", value, err)
			}
			if gets.Load() != 4 || lists.Load() != 4 {
				t.Fatal(gets.Load(), lists.Load())
			}
		})
	}
	for _, code := range []int{403, 404, 201} {
		t.Run(fmt.Sprintf("fallback-error=%d", code), func(t *testing.T) {
			cloud, client := clusteringFindIdentityCloud(t)
			var gets, lists atomic.Int32
			cloud.Mux.HandleFunc("GET "+clusteringFindIdentityPrefix+"/profiles/input", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 404, `{"error":{"message":"direct-missing"}}`)
			})
			cloud.Mux.HandleFunc("GET "+clusteringFindIdentityPrefix+"/profiles", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				testcloud.JSON(w, code, `{"error":{"message":"list-denied-or-missing"}}`)
			})
			value, err := profiles.New(client).FindIdentity(context.Background(), "input")
			if value != nil || !gophercloud.ResponseCodeIs(err, code) || gets.Load() != 1 || lists.Load() != 1 {
				t.Fatal("ignore_missing swallowed fallback HTTP error", value, err, gets.Load(), lists.Load())
			}
		})
	}
	for _, fixture := range clusteringFindIdentityFixtures() {
		t.Run(fixture.path+"/empty-list-after-forbidden", func(t *testing.T) {
			cloud, client := clusteringFindIdentityCloud(t)
			var gets, lists atomic.Int32
			path := clusteringFindIdentityPrefix + "/" + fixture.path
			cloud.Mux.HandleFunc("GET "+path+"/input", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 403, `{"error":{"message":"direct-forbidden"}}`)
			})
			cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				testcloud.JSON(w, 200, clusteringFindIdentityPage(fixture, `[]`, ""))
			})
			access := fixture.new(client)
			if value, err := access.find(context.Background(), "input", clusteringFindIdentityInput{}); value != nil || err != nil {
				t.Fatal("compatible default did not apply final empty-list absence", value, err)
			}
			strict := false
			value, err := access.find(context.Background(), "input", clusteringFindIdentityInput{missing: &strict})
			var absent *resource.NotFoundError
			if value != nil || !errors.As(err, &absent) || !errors.Is(err, resource.ErrNotFound) || absent.Cause != nil || gophercloud.ResponseCodeIs(err, 403) || gets.Load() != 2 || lists.Load() != 2 {
				t.Fatal("strict empty fallback retained suppressed GET403 or lost absence", value, err, absent, gets.Load(), lists.Load())
			}
		})
	}
}

func TestClusteringFindIdentityScansLaterPagesForDuplicatesAndFailures(t *testing.T) {
	for _, fixture := range clusteringFindIdentityFixtures() {
		for _, outcome := range []string{"duplicate", "duplicate-same-id-later", "duplicate-same-id-same-page", "later-http", "later-malformed", "foreign-next", "unique"} {
			t.Run(fixture.path+"/"+outcome, func(t *testing.T) {
				cloud, client := clusteringFindIdentityCloud(t)
				var gets, lists, foreign atomic.Int32
				path := clusteringFindIdentityPrefix + "/" + fixture.path
				cloud.Mux.HandleFunc("GET "+path+"/input", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					testcloud.JSON(w, 404, `{"error":{"message":"missing"}}`)
				})
				cloud.Mux.HandleFunc("GET /foreign", func(w http.ResponseWriter, r *http.Request) { foreign.Add(1); w.WriteHeader(500) })
				cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
					page := lists.Add(1)
					if r.URL.Query().Get("name") != "input" {
						t.Error("continuation lost name query", r.URL)
					}
					if page == 1 {
						if outcome == "duplicate-same-id-same-page" {
							testcloud.JSON(w, 200, clusteringFindIdentityPage(fixture, `[{"id":"first-id","name":"input"},{"id":"first-id","name":"input"}]`, ""))
							return
						}
						next := path + "?marker=second"
						if outcome == "foreign-next" {
							next = "/foreign"
						}
						testcloud.JSON(w, 200, clusteringFindIdentityPage(fixture, `[{"id":"first-id","name":"input"}]`, next))
						return
					}
					if page != 2 || r.URL.Query().Get("marker") != "second" {
						t.Error("unexpected fallback page", page, r.URL)
					}
					switch outcome {
					case "duplicate":
						testcloud.JSON(w, 200, clusteringFindIdentityPage(fixture, `[{"id":"input","name":"another-name"}]`, ""))
					case "duplicate-same-id-later":
						testcloud.JSON(w, 200, clusteringFindIdentityPage(fixture, `[{"id":"first-id","name":"input"}]`, ""))
					case "later-http":
						testcloud.JSON(w, 403, `{"error":{"message":"later-denied"}}`)
					case "later-malformed":
						w.Header().Set("X-Request-ID", "later-malformed")
						testcloud.JSON(w, 200, `{"wrong":[]}`)
					default:
						testcloud.JSON(w, 200, clusteringFindIdentityPage(fixture, `[{"id":"unrelated","name":"not-input"}]`, ""))
					}
				})
				value, err := fixture.new(client).find(context.Background(), "input", clusteringFindIdentityInput{})
				switch outcome {
				case "duplicate", "duplicate-same-id-later", "duplicate-same-id-same-page":
					var ambiguous *resource.AmbiguousError
					wantIDs := []string{"first-id", "input"}
					if outcome != "duplicate" {
						wantIDs[1] = "first-id"
					}
					if value != nil || !errors.As(err, &ambiguous) || !reflect.DeepEqual(ambiguous.IDs, wantIDs) {
						t.Fatal("second ID OR name candidate was not ambiguous", value, err, ambiguous)
					}
				case "later-http":
					if value != nil || !gophercloud.ResponseCodeIs(err, 403) {
						t.Fatal("earlier match hid later HTTP failure", value, err)
					}
				case "later-malformed":
					var evidence *resource.ResponseError
					if value != nil || !errors.As(err, &evidence) || string(evidence.Body) != `{"wrong":[]}` || evidence.Header.Get("X-Request-ID") != "later-malformed" {
						t.Fatal(value, err, evidence)
					}
				case "foreign-next":
					if value != nil || !errors.Is(err, resource.ErrInvalidOption) || foreign.Load() != 0 {
						t.Fatal(value, err, foreign.Load())
					}
				case "unique":
					if err != nil || value == nil || value.id != "first-id" {
						t.Fatal(value, err)
					}
				}
				wantLists := int32(2)
				if outcome == "foreign-next" || outcome == "duplicate-same-id-same-page" {
					wantLists = 1
				}
				if gets.Load() != 1 || lists.Load() != wantLists {
					t.Fatal("lookup returned early or repeated requests", gets.Load(), lists.Load())
				}
			})
		}
	}
}

func TestClusteringFindIdentityAcceptedMalformedAndConsumedRowEvidence(t *testing.T) {
	for _, fixture := range clusteringFindIdentityFixtures() {
		for _, body := range []string{`{`, `{"wrong":{}}`, clusteringFindIdentityObject(fixture, `null`), clusteringFindIdentityObject(fixture, `[]`), clusteringFindIdentityObject(fixture, `{}`), clusteringFindIdentityObject(fixture, `{"id":null}`), clusteringFindIdentityObject(fixture, `{"id":7}`), clusteringFindIdentityObject(fixture, `{"id":"../bad"}`), clusteringFindIdentityObject(fixture, `{"ID":"only-case-alias"}`), clusteringFindIdentityObject(fixture, `{"id":"valid","ID":7}`), clusteringFindIdentityObject(fixture, `{"id":"valid","name":false}`)} {
			t.Run(fixture.path+"/direct="+body, func(t *testing.T) {
				cloud, client := clusteringFindIdentityCloud(t)
				var gets, lists atomic.Int32
				path := clusteringFindIdentityPrefix + "/" + fixture.path
				cloud.Mux.HandleFunc("GET "+path+"/input", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					w.Header().Set("X-Request-ID", "malformed-find")
					testcloud.JSON(w, 200, body)
				})
				cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					testcloud.JSON(w, 200, clusteringFindIdentityPage(fixture, `[]`, ""))
				})
				value, err := fixture.new(client).find(context.Background(), "input", clusteringFindIdentityInput{})
				var evidence *resource.ResponseError
				if value != nil || !errors.As(err, &evidence) || evidence.StatusCode != 200 || string(evidence.Body) != body || evidence.Header.Get("X-Request-ID") != "malformed-find" || gets.Load() != 1 || lists.Load() != 0 {
					t.Fatal("accepted malformed GET fell back or lost response evidence", value, err, evidence, gets.Load(), lists.Load())
				}
			})
		}
		for _, row := range []string{`{"name":"not-input"}`, `{"id":null,"name":"not-input"}`, `{"ID":"alias-only","name":"not-input"}`, `{"id":"valid","name":7}`} {
			t.Run(fixture.path+"/consumed-row="+row, func(t *testing.T) {
				cloud, client := clusteringFindIdentityCloud(t)
				var gets, lists atomic.Int32
				path := clusteringFindIdentityPrefix + "/" + fixture.path
				body := clusteringFindIdentityPage(fixture, `[`+row+`,{"id":"selected","name":"input"}]`, "")
				cloud.Mux.HandleFunc("GET "+path+"/input", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); testcloud.JSON(w, 404, `{}`) })
				cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					w.Header().Set("X-Request-ID", "invalid-list-row")
					testcloud.JSON(w, 200, body)
				})
				value, err := fixture.new(client).find(context.Background(), "input", clusteringFindIdentityInput{})
				var evidence *resource.ResponseError
				if value != nil || !errors.As(err, &evidence) || evidence.StatusCode != 200 || string(evidence.Body) != body || evidence.Header.Get("X-Request-ID") != "invalid-list-row" || gets.Load() != 1 || lists.Load() != 1 {
					t.Fatal("nonmatching malformed row was silently skipped", value, err, evidence, gets.Load(), lists.Load())
				}
			})
		}
	}
	t.Run("read-and-transport-failures", clusteringFindIdentityReadAndTransportFailures)
}

type clusteringFindIdentityTransport func(*http.Request) (*http.Response, error)

func (run clusteringFindIdentityTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return run(r)
}

type clusteringFindIdentityReadFailure struct {
	body  []byte
	cause error
}

func (r *clusteringFindIdentityReadFailure) Read(buffer []byte) (int, error) {
	n := copy(buffer, r.body)
	r.body = r.body[n:]
	return n, r.cause
}
func (*clusteringFindIdentityReadFailure) Close() error { return nil }

func clusteringFindIdentityReadAndTransportFailures(t *testing.T) {
	for _, readFailure := range []bool{false, true} {
		t.Run(fmt.Sprintf("read=%t", readFailure), func(t *testing.T) {
			cloud, client := clusteringFindIdentityCloud(t)
			cause := errors.New("native-find-transport-or-read-failure")
			var calls atomic.Int32
			body := `{"profile":{"id":"valid"}}`
			cloud.Provider.HTTPClient.Transport = clusteringFindIdentityTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.URL.Path != clusteringFindIdentityPrefix+"/profiles/input" || r.Method != http.MethodGet {
					t.Error("failure restarted lookup", r.Method, r.URL)
				}
				if !readFailure {
					return nil, cause
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"X-Request-Id": {"read-failed"}}, Body: &clusteringFindIdentityReadFailure{[]byte(body), cause}, Request: r}, nil
			})
			value, err := profiles.New(client).FindIdentity(context.Background(), "input")
			if value != nil || !errors.Is(err, cause) || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
			if readFailure {
				var evidence *resource.ResponseError
				if !errors.As(err, &evidence) || string(evidence.Body) != body || evidence.StatusCode != 200 || evidence.Header.Get("X-Request-ID") != "read-failed" {
					t.Fatal("read evidence lost", err, evidence)
				}
			}
		})
	}
}

func TestClusteringFindIdentityOptionsSnapshotsLastWinsAndConcurrentReuse(t *testing.T) {
	t.Run("snapshots-and-reuse", func(t *testing.T) {
		cloud, client := clusteringFindIdentityCloud(t)
		client.Microversion = "1.2"
		client.MoreHeaders = map[string]string{"x-trace": "source", "X-Shared": "kept"}
		var gets, lists atomic.Int32
		path := clusteringFindIdentityPrefix + "/profiles"
		check := func(r *http.Request) {
			if r.Header.Get("X-Trace") != "last" || r.Header.Get("X-Shared") != "kept" || r.Header.Get("OpenStack-API-Version") != "clustering 1.7" || r.URL.Query().Has("headers") || r.URL.Query().Has("microversion") {
				t.Error("request selection was not owned", r.URL, r.Header)
			}
		}
		cloud.Mux.HandleFunc("GET "+path+"/missing", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); check(r); testcloud.JSON(w, 404, `{}`) })
		cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
			lists.Add(1)
			check(r)
			testcloud.JSON(w, 200, `{"profiles":[]}`)
		})
		strict := false
		input := profiles.FindOpts{IgnoreMissing: &strict, Fallback: resource.FindFallbackNever}
		snapshot := profiles.WithFindOptions(input)
		strict, input.Fallback = true, resource.FindFallbackCompatible
		var captured map[string]string
		options := []profiles.FindOption{snapshot, profiles.WithFindFallback(resource.FindFallbackNever), profiles.WithFindFallback(resource.FindFallbackCompatible), profiles.WithFindHeader("x-trace", "first"), profiles.WithFindHeader("X-TRACE", "last"), profiles.WithFindMicroversion("1.2"), profiles.WithFindMicroversion("1.7"), func(config *request.Config[profiles.FindOpts]) error { captured = config.Headers; return nil }}
		native := cloud.Provider.HTTPClient.Transport
		if native == nil {
			native = http.DefaultTransport
		}
		cloud.Provider.HTTPClient.Transport = clusteringFindIdentityTransport(func(r *http.Request) (*http.Response, error) {
			response, err := native.RoundTrip(r)
			if r.URL.Path == path+"/missing" {
				captured["X-Trace"] = "changed-config"
				client.Microversion = "1.13"
				client.MoreHeaders["x-trace"] = "changed-source"
			}
			return response, err
		})
		api := profiles.New(client)
		for attempt := 0; attempt < 2; attempt++ {
			value, err := api.FindIdentity(context.Background(), "missing", options...)
			if value != nil || !errors.Is(err, resource.ErrNotFound) {
				t.Fatal("WithFindOptions pointer snapshot or reuse lost", value, err)
			}
		}
		value, err := api.FindIdentity(context.Background(), "missing", append(options, profiles.WithFindIgnoreMissing(true))...)
		if value != nil || err != nil || gets.Load() != 3 || lists.Load() != 3 || client.Microversion != "1.13" || client.MoreHeaders["x-trace"] != "changed-source" {
			t.Fatal(value, err, gets.Load(), lists.Load(), client.MoreHeaders)
		}
	})
	t.Run("concurrent-siblings", func(t *testing.T) {
		cloud, client := clusteringFindIdentityCloud(t)
		client.Microversion = "1.2"
		client.MoreHeaders = map[string]string{"X-Shared": "source"}
		original := maps.Clone(client.MoreHeaders)
		arrived := make(chan struct{}, 2)
		release := make(chan struct{})
		var gets, lists atomic.Int32
		check := func(r *http.Request) {
			caller := r.Header.Get("X-Caller")
			if caller != "one" && caller != "two" || r.Header.Get("OpenStack-API-Version") != map[string]string{"one": "clustering 1.7", "two": "clustering 1.13"}[caller] || r.Header.Get("X-Shared") != "source" {
				t.Error("sibling request selections leaked", r.Header)
			}
		}
		cloud.Mux.HandleFunc("GET "+clusteringFindIdentityPrefix+"/profiles/input", func(w http.ResponseWriter, r *http.Request) {
			gets.Add(1)
			check(r)
			arrived <- struct{}{}
			<-release
			testcloud.JSON(w, 404, `{}`)
		})
		cloud.Mux.HandleFunc("GET "+clusteringFindIdentityPrefix+"/profiles", func(w http.ResponseWriter, r *http.Request) {
			lists.Add(1)
			check(r)
			testcloud.JSON(w, 200, `{"profiles":[{"id":"canonical","name":"input"}]}`)
		})
		var done sync.WaitGroup
		for _, caller := range []string{"one", "two"} {
			done.Add(1)
			go func() {
				defer done.Done()
				selected := map[string]string{"one": "1.7", "two": "1.13"}[caller]
				value, err := profiles.New(client).FindIdentity(context.Background(), "input", profiles.WithFindHeader("X-Caller", caller), profiles.WithFindMicroversion(selected))
				if err != nil || value == nil || value.ID != "canonical" {
					t.Error(value, err)
				}
			}()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		for count := 0; count < 2; count++ {
			select {
			case <-arrived:
			case <-ctx.Done():
				close(release)
				t.Fatal("siblings did not enter independent GETs", ctx.Err())
			}
		}
		close(release)
		done.Wait()
		if gets.Load() != 2 || lists.Load() != 2 || client.Microversion != "1.2" || !reflect.DeepEqual(client.MoreHeaders, original) {
			t.Fatal(gets.Load(), lists.Load(), client.Microversion, client.MoreHeaders)
		}
	})
}

func TestClusteringFindIdentityPreflightProtectsIdentityAndRequestControls(t *testing.T) {
	for _, fixture := range clusteringFindIdentityFixtures() {
		t.Run(fixture.path, func(t *testing.T) {
			cloud, client := clusteringFindIdentityCloud(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			access := fixture.new(client)
			for _, identity := range []string{"", " ", ".", "..", "name/with/slash", "name with space", "escaped%2Fsegment", "name?query", "name#fragment"} {
				if value, err := access.find(context.Background(), identity, clusteringFindIdentityInput{}); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal("invalid route identity reached HTTP", identity, value, err)
				}
			}
			for _, pair := range [][2]string{{"X-Auth-Token", "other"}, {"Authorization", "other"}, {"OpenStack-API-Version", "clustering 1.7"}, {"Content-Type", "other/json"}, {"Host", "other"}, {"Cookie", "session=other"}, {"Content-Length", "1"}, {"X-Invalid\nKey", "value"}, {"X-Invalid", "line\nbreak"}} {
				if value, err := access.find(context.Background(), "input", clusteringFindIdentityInput{headers: [][2]string{pair}}); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal("protected or malformed header reached HTTP", pair, value, err)
				}
			}
			for _, selected := range []string{"latest", "2.0", "1.-1", "1.01", "1.x"} {
				if value, err := access.find(context.Background(), "input", clusteringFindIdentityInput{versions: []string{selected}}); value != nil || err == nil {
					t.Fatal("invalid version reached HTTP", selected, value, err)
				}
			}
			canceled, cancel := context.WithCancel(context.Background())
			cancel()
			if value, err := access.find(canceled, "input", clusteringFindIdentityInput{}); value != nil || !errors.Is(err, context.Canceled) {
				t.Fatal(value, err)
			}
			if calls.Load() != 0 {
				t.Fatal("preflight sent a request", calls.Load())
			}
		})
	}
	cloud, client := clusteringFindIdentityCloud(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	for _, option := range []profiles.FindOption{profiles.WithFindFallback(resource.FindFallbackPolicy(99)), request.WithQuery[profiles.FindOpts]("headers", "value"), request.WithQuery[profiles.FindOpts]("microversion", "1.7"), request.WithQuery[profiles.FindOpts]("base_path", "/other"), request.WithField[profiles.FindOpts]("name", "other"), request.WithArgument[profiles.FindOpts]("foreign.find", true)} {
		if value, err := profiles.New(client).FindIdentity(context.Background(), "input", option); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("generic option bypass reached HTTP", value, err)
		}
	}
	client.Microversion = "1.2"
	client.MoreHeaders = map[string]string{"openstack-api-version": "clustering 1.2"}
	if value, err := profiles.New(client).FindIdentity(context.Background(), "input", profiles.WithFindMicroversion("1.7")); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("source version header silently removed", value, err)
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringFindIdentityPhasesFreezeSelectionAndReadLiveToken(t *testing.T) {
	for _, selection := range []string{"inherited", "1.7", ""} {
		t.Run("selection="+selection, func(t *testing.T) {
			cloud, client := clusteringFindIdentityCloud(t)
			client.Microversion = "1.2"
			path := clusteringFindIdentityPrefix + "/profiles"
			var gets, lists atomic.Int32
			cloud.Mux.HandleFunc("GET "+path+"/input", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				want := "clustering 1.2"
				if selection == "1.7" {
					want = "clustering 1.7"
				} else if selection == "" {
					want = ""
				}
				if r.Header.Get("OpenStack-API-Version") != want || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Error(r.Header)
				}
				testcloud.JSON(w, 404, `{}`)
			})
			cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
				page := lists.Add(1)
				wantVersion, wantToken := "clustering 1.13", "rotated-first"
				if page == 2 {
					wantVersion, wantToken = "clustering 1.0", "rotated-second"
				}
				if selection == "1.7" {
					wantVersion = "clustering 1.7"
				} else if selection == "" {
					wantVersion = ""
				}
				if r.Header.Get("OpenStack-API-Version") != wantVersion || r.Header.Get("X-Auth-Token") != wantToken || r.URL.Query().Get("name") != "input" {
					t.Error(page, r.URL, r.Header)
				}
				if page == 1 {
					testcloud.JSON(w, 200, `{"profiles":[{"id":"selected","name":"input"}],"profiles_links":[{"rel":"next","href":"`+path+`?marker=second"}]}`)
				} else {
					testcloud.JSON(w, 200, `{"profiles":[{"id":"unrelated","name":"other"}]}`)
				}
			})
			native := cloud.Provider.HTTPClient.Transport
			if native == nil {
				native = http.DefaultTransport
			}
			cloud.Provider.HTTPClient.Transport = clusteringFindIdentityTransport(func(r *http.Request) (*http.Response, error) {
				response, err := native.RoundTrip(r)
				if r.URL.Path == path+"/input" {
					cloud.Provider.SetToken("rotated-first")
					client.Microversion = "1.13"
				} else if r.URL.Query().Get("marker") == "" {
					cloud.Provider.SetToken("rotated-second")
					client.Microversion = "1.0"
				}
				return response, err
			})
			var options []profiles.FindOption
			if selection != "inherited" {
				options = []profiles.FindOption{profiles.WithFindHeader("X-Selected", "stable"), profiles.WithFindMicroversion(selection)}
			}
			value, err := profiles.New(client).FindIdentity(context.Background(), "input", options...)
			if err != nil || value == nil || value.ID != "selected" || gets.Load() != 1 || lists.Load() != 2 || cloud.Provider.Token() != "rotated-second" || client.Microversion != "1.0" {
				t.Fatal(value, err, gets.Load(), lists.Load(), client.Microversion)
			}
		})
	}
}

func TestClusteringFindIdentitySourceAndContextRecheckedBeforeFallbackAndContinuation(t *testing.T) {
	for _, boundary := range []string{"fallback", "continuation"} {
		for _, invalid := range []string{"type", "provider", "version-header", "cancel"} {
			t.Run(boundary+"/"+invalid, func(t *testing.T) {
				cloud, client := clusteringFindIdentityCloud(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				path := clusteringFindIdentityPrefix + "/profiles"
				var gets, lists atomic.Int32
				cloud.Mux.HandleFunc("GET "+path+"/input", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); testcloud.JSON(w, 404, `{}`) })
				cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					testcloud.JSON(w, 200, `{"profiles":[{"id":"selected","name":"input"}],"profiles_links":[{"rel":"next","href":"`+path+`?marker=second"}]}`)
				})
				native := cloud.Provider.HTTPClient.Transport
				if native == nil {
					native = http.DefaultTransport
				}
				cloud.Provider.HTTPClient.Transport = clusteringFindIdentityTransport(func(r *http.Request) (*http.Response, error) {
					response, err := native.RoundTrip(r)
					if boundary == "fallback" && r.URL.Path == path+"/input" || boundary == "continuation" && r.URL.Path == path {
						switch invalid {
						case "type":
							client.Type = "compute"
						case "provider":
							client.ProviderClient = nil
						case "version-header":
							client.MoreHeaders = map[string]string{"OpenStack-API-Version": "clustering 1.99"}
						case "cancel":
							cancel()
						}
					}
					return response, err
				})
				value, err := profiles.New(client).FindIdentity(ctx, "input", profiles.WithFindHeader("X-Selected", "owned"), profiles.WithFindMicroversion("1.7"))
				want := resource.ErrInvalidOption
				if invalid == "cancel" {
					want = context.Canceled
				}
				wantLists := int32(0)
				if boundary == "continuation" {
					wantLists = 1
				}
				if value != nil || !errors.Is(err, want) || gets.Load() != 1 || lists.Load() != wantLists {
					t.Fatal("phase boundary reused an invalid source or earlier candidate", value, err, gets.Load(), lists.Load())
				}
			})
		}
	}
}

func TestClusteringFindIdentityExistingExplicitReferencesRemainDeterministic(t *testing.T) {
	for _, fixture := range clusteringFindIdentityFixtures() {
		t.Run(fixture.path, func(t *testing.T) {
			cloud, client := clusteringFindIdentityCloud(t)
			path := clusteringFindIdentityPrefix + "/" + fixture.path
			var gets, lists atomic.Int32
			cloud.Mux.HandleFunc("GET "+path+"/missing", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); testcloud.JSON(w, 404, `{}`) })
			cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				name := r.URL.Query().Get("name")
				quoted, _ := json.Marshal(name)
				testcloud.JSON(w, 200, clusteringFindIdentityPage(fixture, `[{"id":"canonical","name":`+string(quoted)+`},{"id":"not-matched","name":"prefix-only"}]`, ""))
			})
			access := fixture.new(client)
			if value, err := access.explicit(context.Background(), resource.ID("missing")); value != nil || err != nil || gets.Load() != 1 || lists.Load() != 0 {
				t.Fatal("explicit ID started a name fallback", value, err, gets.Load(), lists.Load())
			}
			if value, err := access.explicit(context.Background(), resource.ID("missing"), resource.WithMissingError()); value != nil || !errors.Is(err, resource.ErrNotFound) || gets.Load() != 2 || lists.Load() != 0 {
				t.Fatal(value, err, gets.Load(), lists.Load())
			}
			for _, name := range []string{"12345678-1234-1234-1234-123456789012", "name/with space"} {
				value, err := access.explicit(context.Background(), resource.Name(name))
				if err != nil || value == nil || value.id != "canonical" || value.name != name || gets.Load() != 2 {
					t.Fatal("explicit Name attempted a direct GET or changed exact matching", name, value, err, gets.Load())
				}
			}
			if lists.Load() != 2 {
				t.Fatal(lists.Load())
			}
		})
	}
}

var _ io.ReadCloser = (*clusteringFindIdentityReadFailure)(nil)
