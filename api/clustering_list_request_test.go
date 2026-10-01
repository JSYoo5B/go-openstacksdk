package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/clustering/v1/actions"
	"gophercloudsdk/clustering/v1/clusterpolicies"
	"gophercloudsdk/clustering/v1/clusters"
	"gophercloudsdk/clustering/v1/events"
	"gophercloudsdk/clustering/v1/nodes"
	"gophercloudsdk/clustering/v1/policies"
	"gophercloudsdk/clustering/v1/policytypes"
	"gophercloudsdk/clustering/v1/profiles"
	"gophercloudsdk/clustering/v1/profiletypes"
	"gophercloudsdk/clustering/v1/receivers"
	"gophercloudsdk/clustering/v1/services"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// Source: pinned Resource.list resource.py:2155-2165 consumes headers and
// microversion as controls; :2268-2282 merges headers and keeps both selections
// on all pages. Proxy._list proxy.py:866-906 consumes base_path separately.
// These fixtures exercise the concrete Go facades, not a synthetic adapter.
const clusteringListRequestPrefix = "/proxy/project/senlin/v1"

type clusteringListRequestInput struct {
	headers     [][2]string
	versions    []string
	query       [][2]string
	injectQuery bool
	capture     func(map[string]string)
	foreignArg  bool
	bodyField   bool
}

type clusteringListRequestAccess struct {
	list func(context.Context, clusteringListRequestInput) iter.Seq2[*resource.Metadata, error]
}

func clusteringListRequestAdapter[O, M any](
	list func(context.Context, ...request.Option[O]) iter.Seq2[*M, error],
	header func(string, string) request.Option[O], version func(string) request.Option[O],
	query func(string, string) request.Option[O], metadata func(*M) *resource.Metadata,
) clusteringListRequestAccess {
	return clusteringListRequestAccess{list: func(ctx context.Context, input clusteringListRequestInput) iter.Seq2[*resource.Metadata, error] {
		options := make([]request.Option[O], 0)
		for _, pair := range input.headers {
			options = append(options, header(pair[0], pair[1]))
		}
		for _, value := range input.versions {
			options = append(options, version(value))
		}
		for _, pair := range input.query {
			if !input.injectQuery {
				options = append(options, query(pair[0], pair[1]))
			}
		}
		options = append(options, func(config *request.Config[O]) error {
			if input.injectQuery {
				for _, pair := range input.query {
					config.Query[pair[0]] = []string{pair[1]}
				}
			}
			if input.capture != nil {
				input.capture(config.Headers)
			}
			if input.foreignArg {
				config.Arguments["foreign.list_request"] = "unused"
			}
			if input.bodyField {
				config.Fields["vendor"] = json.RawMessage(`false`)
			}
			return nil
		})
		values := list(ctx, options...)
		return func(yield func(*resource.Metadata, error) bool) {
			for value, err := range values {
				var result *resource.Metadata
				if value != nil {
					result = metadata(value)
				}
				if !yield(result, err) {
					return
				}
			}
		}
	}}
}

type clusteringListRequestFixture struct {
	path, plural string
	binding      bool
	new          func(*testing.T, *gophercloud.ServiceClient) clusteringListRequestAccess
}

func clusteringListRequestFixtures() []clusteringListRequestFixture {
	return []clusteringListRequestFixture{
		{"profiles", "profiles", false, func(_ *testing.T, client *gophercloud.ServiceClient) clusteringListRequestAccess {
			return clusteringListRequestAdapter(profiles.New(client).List, profiles.WithListHeader, profiles.WithListMicroversion, profiles.WithListQuery, func(v *profiles.Profile) *resource.Metadata { return &v.Metadata })
		}},
		{"policies", "policies", false, func(_ *testing.T, client *gophercloud.ServiceClient) clusteringListRequestAccess {
			return clusteringListRequestAdapter(policies.New(client).List, policies.WithListHeader, policies.WithListMicroversion, policies.WithListQuery, func(v *policies.Policy) *resource.Metadata { return &v.Metadata })
		}},
		{"clusters", "clusters", false, func(_ *testing.T, client *gophercloud.ServiceClient) clusteringListRequestAccess {
			return clusteringListRequestAdapter(clusters.New(client).List, clusters.WithListHeader, clusters.WithListMicroversion, clusters.WithListQuery, func(v *clusters.Cluster) *resource.Metadata { return &v.Metadata })
		}},
		{"nodes", "nodes", false, func(_ *testing.T, client *gophercloud.ServiceClient) clusteringListRequestAccess {
			return clusteringListRequestAdapter(nodes.New(client).List, nodes.WithListHeader, nodes.WithListMicroversion, nodes.WithListQuery, func(v *nodes.Node) *resource.Metadata { return &v.Metadata })
		}},
		{"receivers", "receivers", false, func(_ *testing.T, client *gophercloud.ServiceClient) clusteringListRequestAccess {
			return clusteringListRequestAdapter(receivers.New(client).List, receivers.WithListHeader, receivers.WithListMicroversion, receivers.WithListQuery, func(v *receivers.Receiver) *resource.Metadata { return &v.Metadata })
		}},
		{"actions", "actions", false, func(_ *testing.T, client *gophercloud.ServiceClient) clusteringListRequestAccess {
			return clusteringListRequestAdapter(actions.New(client).List, actions.WithListHeader, actions.WithListMicroversion, actions.WithListQuery, func(v *actions.Action) *resource.Metadata { return &v.Metadata })
		}},
		{"events", "events", false, func(_ *testing.T, client *gophercloud.ServiceClient) clusteringListRequestAccess {
			return clusteringListRequestAdapter(events.New(client).List, events.WithListHeader, events.WithListMicroversion, events.WithListQuery, func(v *events.Event) *resource.Metadata { return &v.Metadata })
		}},
		{"services", "services", false, func(_ *testing.T, client *gophercloud.ServiceClient) clusteringListRequestAccess {
			return clusteringListRequestAdapter(services.New(client).List, services.WithListHeader, services.WithListMicroversion, services.WithListQuery, func(v *services.Service) *resource.Metadata { return &v.Metadata })
		}},
		{"profile-types", "profile_types", false, func(_ *testing.T, client *gophercloud.ServiceClient) clusteringListRequestAccess {
			return clusteringListRequestAdapter(profiletypes.New(client).List, profiletypes.WithListHeader, profiletypes.WithListMicroversion, profiletypes.WithListQuery, func(v *profiletypes.ProfileType) *resource.Metadata { return &v.Metadata })
		}},
		{"policy-types", "policy_types", false, func(_ *testing.T, client *gophercloud.ServiceClient) clusteringListRequestAccess {
			return clusteringListRequestAdapter(policytypes.New(client).List, policytypes.WithListHeader, policytypes.WithListMicroversion, policytypes.WithListQuery, func(v *policytypes.PolicyType) *resource.Metadata { return &v.Metadata })
		}},
		{"clusters/canonical-parent/policies", "cluster_policies", true, func(t *testing.T, client *gophercloud.ServiceClient) clusteringListRequestAccess {
			scope, err := clusterpolicies.New(client).InCluster(context.Background(), resource.ID("input-parent"))
			if err != nil {
				t.Fatal(err)
			}
			return clusteringListRequestAdapter(scope.List, clusterpolicies.WithListHeader, clusterpolicies.WithListMicroversion, clusterpolicies.WithListQuery, func(v *clusterpolicies.ClusterPolicy) *resource.Metadata { return &v.Metadata })
		}},
	}
}

func clusteringListRequestCloud(t *testing.T, fixture clusteringListRequestFixture) (*testcloud.Cloud, *gophercloud.ServiceClient) {
	t.Helper()
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/catalog/v1")
	client.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + clusteringListRequestPrefix)
	client.Microversion = "1.7"
	if fixture.binding {
		var parents atomic.Int32
		cloud.Mux.HandleFunc("GET "+clusteringListRequestPrefix+"/clusters/input-parent", func(w http.ResponseWriter, r *http.Request) {
			parents.Add(1)
			if r.Header.Get("X-List-Only") != "" || r.Header.Get("OpenStack-API-Version") != "clustering 1.7" {
				t.Error("list options leaked into parent preparation", r.Header)
			}
			testcloud.JSON(w, 200, `{"cluster":{"id":"canonical-parent","name":"parent"}}`)
		})
		t.Cleanup(func() {
			if parents.Load() != 1 {
				t.Error("canonical parent should be prepared once", parents.Load())
			}
		})
	}
	return cloud, client
}

func clusteringListRequestBody(fixture clusteringListRequestFixture, id, next string) string {
	row := fmt.Sprintf(`{"id":%q,"name":%q,"policy_id":%q,"policy_name":%q,"cluster_id":"canonical-parent","vendor":9007199254740993}`, id, id, "policy-"+id, id)
	if next != "" {
		return fmt.Sprintf(`{%q:[%s],"next":%q}`, fixture.plural, row, next)
	}
	return fmt.Sprintf(`{%q:[%s]}`, fixture.plural, row)
}

func clusteringListRequestCollect(values iter.Seq2[*resource.Metadata, error]) ([]*resource.Metadata, error) {
	var rows []*resource.Metadata
	for value, err := range values {
		if err != nil {
			return rows, err
		}
		rows = append(rows, value)
	}
	return rows, nil
}

func TestClusteringListRequestElevenFacadesWireAndSourceOwnership(t *testing.T) {
	for _, fixture := range clusteringListRequestFixtures() {
		t.Run(fixture.plural, func(t *testing.T) {
			cloud, client := clusteringListRequestCloud(t, fixture)
			client.MoreHeaders = map[string]string{"X-Shared": "source"}
			originalHeaders := maps.Clone(client.MoreHeaders)
			originalEndpoint, originalBase := client.Endpoint, client.ResourceBase
			var gets atomic.Int32
			cloud.Mux.HandleFunc("GET "+clusteringListRequestPrefix+"/"+fixture.path, func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if r.URL.RawQuery != "" || r.Header.Get("X-List-Only") != "request" || r.Header.Get("X-Shared") != "source" || r.Header.Get("OpenStack-API-Version") != "clustering 1.13" || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Error("control did not reach actual scoped wire request", r.URL, r.Header)
				}
				w.Header().Set("X-Request-ID", "owned-list")
				testcloud.JSON(w, 200, clusteringListRequestBody(fixture, "one", ""))
			})
			access := fixture.new(t, client)
			rows, err := clusteringListRequestCollect(access.list(context.Background(), clusteringListRequestInput{
				headers: [][2]string{{"X-List-Only", "request"}}, versions: []string{"1.13"},
			}))
			if err != nil || len(rows) != 1 || gets.Load() != 1 {
				t.Fatal(rows, err, gets.Load())
			}
			if rows[0].Header.Get("X-Request-ID") != "owned-list" || rows[0].StatusCode != 200 || string(rows[0].Body["vendor"]) != "9007199254740993" {
				t.Fatal("response evidence changed", rows[0])
			}
			if client.Microversion != "1.7" || client.Type != "clustering" || client.Endpoint != originalEndpoint || client.ResourceBase != originalBase || client.ProviderClient != cloud.Provider || !reflect.DeepEqual(client.MoreHeaders, originalHeaders) {
				t.Fatal("per-call selection mutated the shared source", client)
			}
		})
	}
}

func TestClusteringListRequestHeadersCasePrecedenceAndConfigSnapshot(t *testing.T) {
	fixture := clusteringListRequestFixtures()[0]
	cloud, client := clusteringListRequestCloud(t, fixture)
	client.MoreHeaders = map[string]string{"x-shared": "source", "x-custom": "source-default", "Accept": "application/json"}
	originalHeaders := maps.Clone(client.MoreHeaders)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET "+clusteringListRequestPrefix+"/profiles", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		for key, want := range map[string]string{"X-Shared": "caller", "X-Custom": "last", "Accept": "application/vnd.test+json"} {
			if got := r.Header.Values(key); len(got) != 1 || got[0] != want {
				t.Error("case-insensitive caller precedence or snapshot lost", key, got)
			}
		}
		if r.Header.Get("OpenStack-API-Version") != "clustering 1.13" {
			t.Error("last microversion selection lost", r.Header)
		}
		next := ""
		if r.URL.Query().Get("marker") == "" {
			next = clusteringListRequestPrefix + "/profiles?marker=two"
		}
		testcloud.JSON(w, 200, clusteringListRequestBody(fixture, "row", next))
	})
	var captured map[string]string
	values := fixture.new(t, client).list(context.Background(), clusteringListRequestInput{
		headers:  [][2]string{{"X-Shared", "caller"}, {"x-custom", "first"}, {"X-Custom", "last"}, {"Accept", "application/vnd.test+json"}},
		versions: []string{"1.12", "1.13"}, capture: func(value map[string]string) { captured = value },
	})
	count := 0
	for _, err := range values {
		if err != nil {
			t.Fatal(err)
		}
		count++
		captured["X-Custom"] = "changed-after-prepare"
		captured["X-Shared"] = "changed-after-prepare"
	}
	if count != 2 || calls.Load() != 2 || !reflect.DeepEqual(client.MoreHeaders, originalHeaders) {
		t.Fatal("headers escaped their request ownership", count, calls.Load(), client.MoreHeaders)
	}

	// List retains its caller option slice at construction. Later slice changes
	// cannot replace controls before its lazy first request.
	cloud2 := testcloud.New(t)
	client2 := cloud2.Client("clustering", clusteringListRequestPrefix)
	cloud2.Mux.HandleFunc("GET "+clusteringListRequestPrefix+"/profiles", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Snapshot") != "original" {
			t.Error("caller option slice replaced constructed iterator", r.Header)
		}
		testcloud.JSON(w, 200, `{"profiles":[{"id":"one"}]}`)
	})
	options := []profiles.ListOption{profiles.WithListHeader("X-Snapshot", "original")}
	iterator := profiles.New(client2).List(context.Background(), options...)
	options[0] = profiles.WithListHeader("X-Snapshot", "replaced")
	for _, err := range iterator {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestClusteringListRequestIteratorsReuseAndConcurrentSiblings(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", clusteringListRequestPrefix)
	client.Microversion = "1.7"
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET "+clusteringListRequestPrefix+"/profiles", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		identity, version := r.Header.Get("X-Sibling"), r.Header.Get("OpenStack-API-Version")
		want := map[string]string{"one": "clustering 1.12", "two": "clustering 1.13"}[identity]
		if want == "" || version != want {
			t.Error("independent iterators contaminated each other", identity, version)
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"profiles":[{"id":%q}]}`, identity))
	})
	api := profiles.New(client)
	one := api.List(context.Background(), profiles.WithListHeader("X-Sibling", "one"), profiles.WithListMicroversion("1.12"))
	two := api.List(context.Background(), profiles.WithListHeader("X-Sibling", "two"), profiles.WithListMicroversion("1.13"))
	var wg sync.WaitGroup
	errorsFound := make(chan error, 2)
	for _, iterator := range []iter.Seq2[*profiles.Profile, error]{one, two} {
		wg.Add(1)
		go func(values iter.Seq2[*profiles.Profile, error]) {
			defer wg.Done()
			for _, err := range values {
				if err != nil {
					errorsFound <- err
					return
				}
			}
		}(iterator)
	}
	wg.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Fatal(err)
	}
	for _, err := range one {
		if err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 3 || client.Microversion != "1.7" || len(client.MoreHeaders) != 0 {
		t.Fatal("reuse or siblings mutated source", calls.Load(), client)
	}
}

func TestClusteringListRequestPagesFreezeSelectionAndReadLiveProviderToken(t *testing.T) {
	fixture := clusteringListRequestFixtures()[0]
	cloud, client := clusteringListRequestCloud(t, fixture)
	client.Microversion = "1.0"
	var gets atomic.Int32
	cloud.Mux.HandleFunc("GET "+clusteringListRequestPrefix+"/profiles", func(w http.ResponseWriter, r *http.Request) {
		page := gets.Add(1)
		wantToken := "test-token"
		if page == 2 {
			wantToken = "rotated-token"
		}
		if r.Header.Get("X-Page") != "stable" || r.Header.Get("OpenStack-API-Version") != "clustering 1.7" || r.Header.Get("X-Auth-Token") != wantToken {
			t.Error("page clone detached provider or changed request selection", page, r.Header)
		}
		if page == 1 {
			testcloud.JSON(w, 200, clusteringListRequestBody(fixture, "one", clusteringListRequestPrefix+"/profiles?marker=two"))
		} else {
			testcloud.JSON(w, 200, clusteringListRequestBody(fixture, "two", ""))
		}
	})
	count := 0
	for _, err := range profiles.New(client).List(context.Background(), profiles.WithListHeader("X-Page", "stable"), profiles.WithListMicroversion("1.7")) {
		if err != nil {
			t.Fatal(err)
		}
		count++
		if count == 1 {
			cloud.Provider.SetToken("rotated-token")
			client.Microversion = "1.13" // Valid source changes do not retarget this iteration.
		}
	}
	if count != 2 || gets.Load() != 2 || client.Microversion != "1.13" {
		t.Fatal(count, gets.Load(), client.Microversion)
	}
}

func TestClusteringListRequestEffectiveVersionGates(t *testing.T) {
	for _, kind := range []string{"services", "receiver-user"} {
		minimum, below := "1.7", "1.6"
		if kind == "receiver-user" {
			minimum, below = "1.4", "1.3"
		}
		for _, tc := range []struct {
			name, source, selected string
			allowed                bool
		}{
			{"upgrade", "1.0", minimum, true},
			{"downgrade", "1.13", below, false},
			{"server-default", "1.13", "", false},
		} {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("clustering", clusteringListRequestPrefix)
				client.Microversion = tc.source
				path, plural := "services", "services"
				if kind == "receiver-user" {
					path, plural = "receivers", "receivers"
				}
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET "+clusteringListRequestPrefix+"/"+path, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Header.Get("OpenStack-API-Version") != "clustering "+tc.selected || r.URL.Query().Has("microversion") {
						t.Error("effective version gate did not own wire selection", r.URL, r.Header)
					}
					if kind == "receiver-user" && r.URL.Query().Get("user") != "owner" {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[{"id":"one"}]}`, plural))
				})
				var err error
				if kind == "services" {
					_, err = services.New(client).All(context.Background(), services.WithListMicroversion(tc.selected))
				} else {
					_, err = receivers.New(client).All(context.Background(), receivers.WithListUserID("owner"), receivers.WithListMicroversion(tc.selected))
				}
				if tc.allowed {
					if err != nil || calls.Load() != 1 {
						t.Fatal("effective upgrade should pass", err, calls.Load())
					}
				} else if !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 0 {
					t.Fatal("effective downgrade/default must fail before HTTP", err, calls.Load())
				}
				if client.Microversion != tc.source {
					t.Fatal("gate changed shared version", client.Microversion)
				}
			})
		}
	}
	// An explicit empty selection means server default 1.0 on unversioned reads.
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", clusteringListRequestPrefix)
	client.Microversion = "1.13"
	cloud.Mux.HandleFunc("GET "+clusteringListRequestPrefix+"/profiles", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("OpenStack-API-Version") != "" {
			t.Error("explicit server default kept source version", r.Header)
		}
		testcloud.JSON(w, 200, `{"profiles":[]}`)
	})
	if _, err := profiles.New(client).All(context.Background(), profiles.WithListMicroversion("")); err != nil || client.Microversion != "1.13" {
		t.Fatal(err, client.Microversion)
	}
}

func TestClusteringListRequestBindingScopeAndOtherOperationsUnchanged(t *testing.T) {
	fixture := clusteringListRequestFixtures()[10]
	cloud, client := clusteringListRequestCloud(t, fixture)
	var lists, details atomic.Int32
	cloud.Mux.HandleFunc("GET "+clusteringListRequestPrefix+"/clusters/canonical-parent/policies", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if r.Header.Get("OpenStack-API-Version") != "clustering 1.13" || r.Header.Get("X-List-Only") != "scoped" || r.URL.RawQuery != "" {
			t.Error("scoped list did not keep canonical parent and owned selection", r.URL, r.Header)
		}
		testcloud.JSON(w, 200, clusteringListRequestBody(fixture, "binding-one", ""))
	})
	cloud.Mux.HandleFunc("GET "+clusteringListRequestPrefix+"/clusters/canonical-parent/policies/policy-one", func(w http.ResponseWriter, r *http.Request) {
		details.Add(1)
		if r.Header.Get("OpenStack-API-Version") != "clustering 1.7" || r.Header.Get("X-List-Only") != "" {
			t.Error("per-list selection leaked to Get", r.Header)
		}
		testcloud.JSON(w, 200, `{"cluster_policy":{"id":"detail-binding","policy_id":"policy-one","cluster_id":"canonical-parent"}}`)
	})
	scope, err := clusterpolicies.New(client).InCluster(context.Background(), resource.ID("input-parent"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := scope.All(context.Background(), clusterpolicies.WithListHeader("X-List-Only", "scoped"), clusterpolicies.WithListMicroversion("1.13"))
	if err != nil || len(rows) != 1 || rows[0].URIClusterID != "canonical-parent" || scope.ClusterID() != "canonical-parent" {
		t.Fatal(rows, err, scope.ClusterID())
	}
	row, err := scope.Get(context.Background(), "policy-one")
	if err != nil || row == nil || row.PolicyID != "policy-one" || lists.Load() != 1 || details.Load() != 1 || client.Microversion != "1.7" {
		t.Fatal(row, err, lists.Load(), details.Load(), client.Microversion)
	}
}

func TestClusteringListRequestReservedQueriesAndUnsupportedCapabilities(t *testing.T) {
	// Python consumes these names before QueryParameters._validate. They must
	// never become generic wire params in Go, including custom Option injection.
	for _, fixture := range clusteringListRequestFixtures() {
		t.Run(fixture.plural, func(t *testing.T) {
			cloud, client := clusteringListRequestCloud(t, fixture)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+clusteringListRequestPrefix+"/"+fixture.path, func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[]}`, fixture.plural))
			})
			access := fixture.new(t, client)
			for _, key := range []string{"headers", "microversion", "base_path"} {
				for _, injected := range []bool{false, true} {
					rows, err := clusteringListRequestCollect(access.list(context.Background(), clusteringListRequestInput{query: [][2]string{{key, "literal"}}, injectQuery: injected}))
					if !errors.Is(err, resource.ErrInvalidOption) || len(rows) != 0 || calls.Load() != 0 {
						t.Fatal("reserved control reached child HTTP", key, injected, rows, err, calls.Load())
					}
				}
			}
			for _, input := range []clusteringListRequestInput{{foreignArg: true}, {bodyField: true}} {
				_, err := clusteringListRequestCollect(access.list(context.Background(), input))
				if err == nil || calls.Load() != 0 {
					t.Fatal("unsupported option capability reached HTTP", input, err, calls.Load())
				}
			}
		})
	}
}

func TestClusteringListRequestProtectedHeadersInvalidVersionsAndSourceConflicts(t *testing.T) {
	fixture := clusteringListRequestFixtures()[0]
	cloud, client := clusteringListRequestCloud(t, fixture)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET "+clusteringListRequestPrefix+"/profiles", func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"profiles":[]}`)
	})
	access := fixture.new(t, client)
	for _, pair := range [][2]string{
		{"x-auth-token", "other"}, {"X-Service-Token", "other"}, {"authorization", "other"},
		{"OpenStack-API-Version", "clustering 1.13"}, {"content-type", "text/plain"},
		{"Content-Length", "10"}, {"Host", "foreign.invalid"}, {"Cookie", "other"},
		{"", "empty"}, {"X Invalid", "bad"}, {"X-Extra", "bad\r\nvalue"},
	} {
		_, err := clusteringListRequestCollect(access.list(context.Background(), clusteringListRequestInput{headers: [][2]string{pair}}))
		if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal("protected or malformed header reached HTTP", pair[0], err, calls.Load())
		}
	}
	for _, version := range []string{"latest", "2.1", "1", "1.01", "1.-1", " 1.7", "1.7\n"} {
		_, err := clusteringListRequestCollect(access.list(context.Background(), clusteringListRequestInput{versions: []string{version}}))
		if err == nil || calls.Load() != 0 {
			t.Fatal("invalid version reached HTTP", version, err, calls.Load())
		}
	}
	for _, conflict := range []struct {
		name string
		set  func()
	}{
		{"source-type", func() { client.Type = "compute" }},
		{"source-version", func() { client.Microversion = "latest" }},
		{"source-version-header", func() { client.MoreHeaders = map[string]string{"openstack-api-version": "clustering 1.99"} }},
		{"stale-explicit-version-header", func() { client.MoreHeaders = map[string]string{"OpenStack-API-Version": "clustering 1.7"} }},
	} {
		client.Type, client.Microversion, client.MoreHeaders = "clustering", "1.7", nil
		conflict.set()
		_, err := clusteringListRequestCollect(access.list(context.Background(), clusteringListRequestInput{versions: []string{"1.13"}}))
		if err == nil || calls.Load() != 0 {
			t.Fatal("invalid source or stale source header hidden by clone", conflict.name, err, calls.Load())
		}
	}
	client.Type, client.Microversion, client.MoreHeaders = "clustering", "1.7", map[string]string{"OpenStack-API-Version": "clustering 1.7"}
	_, err := clusteringListRequestCollect(access.list(context.Background(), clusteringListRequestInput{versions: []string{""}}))
	if err == nil || calls.Load() != 0 {
		t.Fatal("server default erased explicit source version header", err, calls.Load())
	}
}

func TestClusteringListRequestSourceRecheckCancellationAndErrorEvidence(t *testing.T) {
	fixture := clusteringListRequestFixtures()[0]
	for _, mutation := range []string{"type", "provider", "version-header", "cancel"} {
		t.Run(mutation, func(t *testing.T) {
			cloud, client := clusteringListRequestCloud(t, fixture)
			var gets atomic.Int32
			cloud.Mux.HandleFunc("GET "+clusteringListRequestPrefix+"/profiles", func(w http.ResponseWriter, _ *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 200, clusteringListRequestBody(fixture, "one", clusteringListRequestPrefix+"/profiles?marker=two"))
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			rows := 0
			var failure error
			for value, err := range profiles.New(client).List(ctx, profiles.WithListMicroversion("1.13"), profiles.WithListHeader("X-List", "owned")) {
				if err != nil {
					failure = err
					continue
				}
				if value == nil {
					t.Fatal("nil successful row")
				}
				rows++
				switch mutation {
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
			if rows != 1 || failure == nil || gets.Load() != 1 {
				t.Fatal("clone hid original source invalidation or canceled parent", rows, failure, gets.Load())
			}
			if mutation == "cancel" && !errors.Is(failure, context.Canceled) {
				t.Fatal("parent cancellation cause lost", failure)
			}
		})
	}
	for _, malformed := range []bool{false, true} {
		t.Run(fmt.Sprintf("error-evidence/%v", malformed), func(t *testing.T) {
			cloud, client := clusteringListRequestCloud(t, fixture)
			var gets atomic.Int32
			body, code := `{"error":{"message":"forbidden"}}`, 403
			if malformed {
				body, code = `{"wrong":[]}`, 200
			}
			cloud.Mux.HandleFunc("GET "+clusteringListRequestPrefix+"/profiles", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if r.Header.Get("OpenStack-API-Version") != "clustering 1.13" || r.Header.Get("X-List") != "owned" {
					t.Error("owned controls missing on failed request", r.Header)
				}
				w.Header().Set("X-Request-ID", "list-failure")
				testcloud.JSON(w, code, body)
			})
			_, err := profiles.New(client).All(context.Background(), profiles.WithListMicroversion("1.13"), profiles.WithListHeader("X-List", "owned"))
			if err == nil || gets.Load() != 1 {
				t.Fatal("failure missing or request resent", err, gets.Load())
			}
			if malformed {
				var response *resource.ResponseError
				if !errors.As(err, &response) || response.StatusCode != code || string(response.Body) != body || response.Header.Get("X-Request-ID") != "list-failure" {
					t.Fatal("accepted response evidence lost", err)
				}
			} else {
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) || response.Actual != code || string(response.Body) != body || response.ResponseHeader.Get("X-Request-ID") != "list-failure" {
					t.Fatal("native HTTP evidence lost", err)
				}
			}
		})
	}
	// With no explicit per-call controls, retain the existing live source
	// selection between pages rather than implicitly creating an owned clone.
	cloud, client := clusteringListRequestCloud(t, fixture)
	var gets atomic.Int32
	cloud.Mux.HandleFunc("GET "+clusteringListRequestPrefix+"/profiles", func(w http.ResponseWriter, r *http.Request) {
		page := gets.Add(1)
		want := "clustering 1.7"
		next := clusteringListRequestPrefix + "/profiles?marker=two"
		if page == 2 {
			want, next = "clustering 1.13", ""
		}
		if r.Header.Get("OpenStack-API-Version") != want {
			t.Error("zero controls changed live source behavior", page, r.Header)
		}
		testcloud.JSON(w, 200, clusteringListRequestBody(fixture, "row", next))
	})
	for _, err := range profiles.New(client).List(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		client.Microversion = "1.13"
	}
	if gets.Load() != 2 {
		t.Fatal("zero-control continuation changed", gets.Load())
	}
}
