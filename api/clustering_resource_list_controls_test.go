package api_test

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/events"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/policytypes"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/services"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const resourceListPrefix = "/reverse/senlin/v1"

type resourceListControlAccess struct {
	list func(context.Context, ...resource.ListOption) iter.Seq2[*resource.Metadata, error]
	all  func(context.Context, ...resource.ListOption) ([]*resource.Metadata, error)
}

func resourceListControlAdapter[T any](collection *resource.Collection[T], metadata func(*T) *resource.Metadata) resourceListControlAccess {
	return resourceListControlAccess{
		list: func(ctx context.Context, options ...resource.ListOption) iter.Seq2[*resource.Metadata, error] {
			return typedListMetadata(collection.List(ctx, options...), metadata)
		},
		all: func(ctx context.Context, options ...resource.ListOption) ([]*resource.Metadata, error) {
			values, err := collection.All(ctx, options...)
			if err != nil {
				return nil, err
			}
			result := make([]*resource.Metadata, 0, len(values))
			for _, value := range values {
				result = append(result, metadata(value))
			}
			return result, nil
		},
	}
}

type resourceListControlCase struct {
	path, plural       string
	hasName, hasStatus bool
	localStatus        bool
	scoped             bool
	new                func(*testing.T, *gophercloud.ServiceClient) resourceListControlAccess
}

func resourceListControlCases() []resourceListControlCase {
	return []resourceListControlCase{
		{"profiles", "profiles", true, false, false, false, func(_ *testing.T, client *gophercloud.ServiceClient) resourceListControlAccess {
			return resourceListControlAdapter(profiles.New(client).Resources, func(value *profiles.Profile) *resource.Metadata { return &value.Metadata })
		}},
		{"policies", "policies", true, false, false, false, func(_ *testing.T, client *gophercloud.ServiceClient) resourceListControlAccess {
			return resourceListControlAdapter(policies.New(client).Resources, func(value *policies.Policy) *resource.Metadata { return &value.Metadata })
		}},
		{"clusters", "clusters", true, true, false, false, func(_ *testing.T, client *gophercloud.ServiceClient) resourceListControlAccess {
			return resourceListControlAdapter(clusters.New(client).Resources, func(value *clusters.Cluster) *resource.Metadata { return &value.Metadata })
		}},
		{"nodes", "nodes", true, true, false, false, func(_ *testing.T, client *gophercloud.ServiceClient) resourceListControlAccess {
			return resourceListControlAdapter(nodes.New(client).Resources, func(value *nodes.Node) *resource.Metadata { return &value.Metadata })
		}},
		{"receivers", "receivers", true, false, false, false, func(_ *testing.T, client *gophercloud.ServiceClient) resourceListControlAccess {
			return resourceListControlAdapter(receivers.New(client).Resources, func(value *receivers.Receiver) *resource.Metadata { return &value.Metadata })
		}},
		{"actions", "actions", true, true, false, false, func(_ *testing.T, client *gophercloud.ServiceClient) resourceListControlAccess {
			return resourceListControlAdapter(actions.New(client).Resources, func(value *actions.Action) *resource.Metadata { return &value.Metadata })
		}},
		{"events", "events", false, false, false, false, func(_ *testing.T, client *gophercloud.ServiceClient) resourceListControlAccess {
			return resourceListControlAdapter(events.New(client).Resources, func(value *events.Event) *resource.Metadata { return &value.Metadata })
		}},
		{"profile-types", "profile_types", true, false, false, false, func(_ *testing.T, client *gophercloud.ServiceClient) resourceListControlAccess {
			return resourceListControlAdapter(profiletypes.New(client).Resources, func(value *profiletypes.ProfileType) *resource.Metadata { return &value.Metadata })
		}},
		{"policy-types", "policy_types", true, false, false, false, func(_ *testing.T, client *gophercloud.ServiceClient) resourceListControlAccess {
			return resourceListControlAdapter(policytypes.New(client).Resources, func(value *policytypes.PolicyType) *resource.Metadata { return &value.Metadata })
		}},
		{"services", "services", false, true, true, false, func(_ *testing.T, client *gophercloud.ServiceClient) resourceListControlAccess {
			return resourceListControlAdapter(services.New(client).Resources, func(value *services.Service) *resource.Metadata { return &value.Metadata })
		}},
		{"clusters/canonical-parent/policies", "cluster_policies", true, false, false, true, func(t *testing.T, client *gophercloud.ServiceClient) resourceListControlAccess {
			t.Helper()
			scope, err := clusterpolicies.New(client).InCluster(context.Background(), resource.ID("input-parent"))
			if err != nil {
				t.Fatal(err)
			}
			if scope.ClusterID() != "canonical-parent" {
				t.Fatal("parent request identity replaced canonical response identity", scope.ClusterID())
			}
			return resourceListControlAdapter(scope.Resources, func(value *clusterpolicies.ClusterPolicy) *resource.Metadata { return &value.Metadata })
		}},
	}
}

func newResourceListControlAccess(t *testing.T, cloud *testcloud.Cloud, facade resourceListControlCase) resourceListControlAccess {
	t.Helper()
	if facade.scoped {
		var parents atomic.Int32
		cloud.Mux.HandleFunc("GET "+resourceListPrefix+"/clusters/input-parent", func(w http.ResponseWriter, _ *http.Request) {
			parents.Add(1)
			testcloud.JSON(w, 200, `{"cluster":{"id":"canonical-parent","name":"parent"}}`)
		})
		t.Cleanup(func() {
			if parents.Load() != 1 {
				t.Error("scope must prepare canonical parent once", parents.Load())
			}
		})
	}
	client := cloud.Client("clustering", resourceListPrefix)
	if facade.plural == "services" {
		client.Microversion = "1.7"
	}
	return facade.new(t, client)
}

func resourceListControlRow(identity, name, status string) string {
	return fmt.Sprintf(`{"id":%q,"name":%q,"status":%q,"policy_id":%q,"policy_name":%q,"cluster_id":"canonical-parent","vendor":9007199254740993}`, identity, name, status, "policy-"+identity, name)
}

func TestClusteringResourceListControlsElevenBindingsCapRawRowsBeforeLocalFilters(t *testing.T) {
	for _, facade := range resourceListControlCases() {
		t.Run(facade.plural, func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests atomic.Int32
			cloud.Mux.HandleFunc("GET "+resourceListPrefix+"/"+facade.path, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				wantLimit := "2"
				if facade.scoped {
					wantLimit = ""
				}
				if r.URL.Query().Get("limit") != wantLimit || r.URL.Query().Has("max_items") || r.URL.Query().Has("paginated") {
					t.Error("local controls or wrong limit hint on wire", r.URL)
				}
				if facade.hasStatus {
					wantStatus := "active"
					if facade.localStatus {
						wantStatus = ""
					}
					if r.URL.Query().Get("status") != wantStatus {
						t.Error("status query ownership", r.URL)
					}
				}
				w.Header().Set("X-Request-ID", "raw-cap-page")
				body := fmt.Sprintf(`{%q:[%s,%s,{"id":[],"name":[],"policy_id":[],"cluster_id":[]}],"next":"https://foreign.invalid/next"}`, facade.plural, resourceListControlRow("one", "other", "ERROR"), resourceListControlRow("two", "wanted", "ACTIVE"))
				testcloud.JSON(w, 200, body)
			})
			access := newResourceListControlAccess(t, cloud, facade)
			options := []resource.ListOption{resource.WithMaxItems(2)}
			if facade.hasName {
				options = append(options, resource.WithName("wanted"))
			}
			if facade.hasStatus {
				options = append(options, resource.WithStatus("active"))
			}
			values, err := access.all(context.Background(), options...)
			want := 2
			if facade.hasName || facade.hasStatus {
				want = 1
			}
			if err != nil || len(values) != want || requests.Load() != 1 {
				t.Fatal("raw cap applied after filter or failed to skip trailing row/foreign next", err, len(values), requests.Load())
			}
			for _, value := range values {
				if value.StatusCode != 200 || value.Header.Get("X-Request-ID") != "raw-cap-page" || string(value.Body["vendor"]) != "9007199254740993" {
					t.Fatal("response evidence lost through common Resources.All", value)
				}
			}
		})
	}
}

func TestClusteringResourceListControlsElevenBindingsSinglePage(t *testing.T) {
	for _, facade := range resourceListControlCases() {
		t.Run(facade.plural, func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests atomic.Int32
			cloud.Mux.HandleFunc("GET "+resourceListPrefix+"/"+facade.path, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.URL.Query().Has("limit") || r.URL.Query().Has("paginated") {
					t.Error(r.URL)
				}
				w.Header().Set("Link", "malformed continuation ignored for one page")
				testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[%s,%s],"next":"https://foreign.invalid/next"}`, facade.plural, resourceListControlRow("one", "one", "ACTIVE"), resourceListControlRow("two", "two", "ACTIVE")))
			})
			access := newResourceListControlAccess(t, cloud, facade)
			values, err := typedListCollect(access.list(context.Background(), resource.WithPaginated(false)))
			if err != nil || len(values) != 2 || requests.Load() != 1 {
				t.Fatal("public Resources.List did not stop after one page", err, len(values), requests.Load())
			}
		})
	}
}

func TestClusteringResourceListControlsElevenBindingsEmptyIgnoresContinuation(t *testing.T) {
	for _, facade := range resourceListControlCases() {
		t.Run(facade.plural, func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests atomic.Int32
			cloud.Mux.HandleFunc("GET "+resourceListPrefix+"/"+facade.path, func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				w.Header().Set("Link", "malformed continuation ignored on empty page")
				testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[],"next":false}`, facade.plural))
			})
			access := newResourceListControlAccess(t, cloud, facade)
			values, err := access.all(context.Background(), resource.WithMaxItems(10))
			if err != nil || values == nil || len(values) != 0 || requests.Load() != 1 {
				t.Fatal("empty Senlin page inspected or followed continuation", err, values, requests.Load())
			}
		})
	}
}

func TestClusteringResourceListControlsElevenBindingsLimitHintAndExplicitPageSize(t *testing.T) {
	for _, facade := range resourceListControlCases() {
		for _, pageSize := range []int{0, 3} {
			t.Run(facade.plural+"/size="+strconv.Itoa(pageSize), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Mux.HandleFunc("GET "+resourceListPrefix+"/"+facade.path, func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					want := "10"
					if pageSize != 0 {
						want = strconv.Itoa(pageSize)
					} else if facade.scoped {
						want = ""
					}
					if r.URL.Query().Get("limit") != want {
						t.Error("maximum overwrote explicit size or unsupported hint", r.URL)
					}
					testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[%s]}`, facade.plural, resourceListControlRow("one", "one", "ACTIVE")))
				})
				access := newResourceListControlAccess(t, cloud, facade)
				options := []resource.ListOption{resource.WithMaxItems(10), resource.WithPaginated(false)}
				if pageSize != 0 {
					options = append(options, resource.WithPageSize(pageSize))
				}
				values, err := access.all(context.Background(), options...)
				if facade.scoped && pageSize != 0 {
					if !errors.Is(err, resource.ErrUnsupported) || requests.Load() != 0 {
						t.Fatal("initial cluster policy page size must fail before child HTTP", err, requests.Load())
					}
					return
				}
				if err != nil || len(values) != 1 || requests.Load() != 1 {
					t.Fatal(err, len(values), requests.Load())
				}
			})
		}
	}
}

func TestClusteringResourceListControlsElevenBindingsDefaultForeignGuard(t *testing.T) {
	for _, facade := range resourceListControlCases() {
		t.Run(facade.plural, func(t *testing.T) {
			cloud, foreign := testcloud.New(t), testcloud.New(t)
			var requests, foreignRequests atomic.Int32
			foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { foreignRequests.Add(1); w.WriteHeader(500) })
			body := fmt.Sprintf(`{%q:[%s],"next":%q}`, facade.plural, resourceListControlRow("one", "one", "ACTIVE"), foreign.Server.URL+resourceListPrefix+"/"+facade.path)
			cloud.Mux.HandleFunc("GET "+resourceListPrefix+"/"+facade.path, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if len(r.URL.Query()) != 0 {
					t.Error("default list invented pagination query", r.URL)
				}
				w.Header().Set("X-Request-ID", "guard-page")
				testcloud.JSON(w, 200, body)
			})
			access := newResourceListControlAccess(t, cloud, facade)
			var terminal error
			var yielded int
			for _, err := range access.list(context.Background()) {
				if err != nil {
					terminal = err
					break
				}
				yielded++
			}
			var responseError *resource.ResponseError
			if !errors.As(terminal, &responseError) || responseError.StatusCode != 200 || string(responseError.Body) != body || responseError.Header.Get("X-Request-ID") != "guard-page" || requests.Load() != 1 || foreignRequests.Load() != 0 || yielded != 1 {
				t.Fatal("uncontrolled list lost foreign guard or accepted evidence", terminal, requests.Load(), foreignRequests.Load(), yielded)
			}
		})
	}
}

func TestClusteringResourceListControlsServicesVersionGateBeforeHTTP(t *testing.T) {
	for _, version := range []string{"", "1.6"} {
		t.Run("version="+version, func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { requests.Add(1); w.WriteHeader(500) })
			client := cloud.Client("clustering", resourceListPrefix)
			client.Microversion = version
			values, err := services.New(client).Resources.All(context.Background(), resource.WithMaxItems(10), resource.WithPaginated(false))
			if !errors.Is(err, resource.ErrUnsupported) || values != nil || requests.Load() != 0 {
				t.Fatal("list controls bypassed Services >=1.7 gate", err, values, requests.Load())
			}
		})
	}
}
