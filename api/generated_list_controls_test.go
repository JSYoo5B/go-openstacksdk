package api_test

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/compute/v2/aggregates"
	"github.com/JSYoo5B/gophercloudsdk/compute/v2/servers"
	"github.com/JSYoo5B/gophercloudsdk/dns/v2/recordsets"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/loadbalancer/v2/pools"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type generatedListAccess struct {
	list func(context.Context, ...resource.ListOption) iter.Seq2[string, error]
	all  func(context.Context, ...resource.ListOption) ([]string, error)
}

func generatedListAccessFor[T any](collection *resource.Collection[T], id func(*T) string) generatedListAccess {
	return generatedListAccess{
		list: func(ctx context.Context, options ...resource.ListOption) iter.Seq2[string, error] {
			return func(yield func(string, error) bool) {
				for value, err := range collection.List(ctx, options...) {
					identity := ""
					if value != nil {
						identity = id(value)
					}
					if !yield(identity, err) {
						return
					}
				}
			}
		},
		all: func(ctx context.Context, options ...resource.ListOption) ([]string, error) {
			rows, err := collection.All(ctx, options...)
			if err != nil {
				return nil, err
			}
			ids := make([]string, 0, len(rows))
			for _, row := range rows {
				ids = append(ids, id(row))
			}
			return ids, nil
		},
	}
}

type generatedListFixture struct {
	service, endpoint, path, key string
	regexName, localStatus       bool
	new                          func(*testing.T, *gophercloud.ServiceClient) generatedListAccess
}

func generatedListFixtures() []generatedListFixture {
	return []generatedListFixture{
		{"compute", "/reverse/nova/v2.1/project", "/reverse/nova/v2.1/project/servers/detail", "servers", true, false,
			func(_ *testing.T, client *gophercloud.ServiceClient) generatedListAccess {
				return generatedListAccessFor(servers.New(client).Resources, func(value *servers.Server) string { return value.ID })
			}},
		{"dns", "/reverse/designate/v2", "/reverse/designate/v2/zones/parent/recordsets", "recordsets", false, false,
			func(t *testing.T, client *gophercloud.ServiceClient) generatedListAccess {
				scope, err := recordsets.New(client).InZone(context.Background(), resource.ID("parent"))
				if err != nil {
					t.Fatal(err)
				}
				return generatedListAccessFor(scope.Collection, func(value *recordsets.RecordSet) string { return value.ID })
			}},
		{"load-balancer", "/reverse/octavia/v2.0", "/reverse/octavia/v2.0/lbaas/pools/parent/members", "members", false, true,
			func(t *testing.T, client *gophercloud.ServiceClient) generatedListAccess {
				scope, err := pools.New(client).Members(context.Background(), resource.ID("parent"))
				if err != nil {
					t.Fatal(err)
				}
				return generatedListAccessFor(scope.Collection, func(value *pools.Member) string { return value.ID })
			}},
	}
}

func generatedListBody(fixture generatedListFixture, rows, next string) string {
	links := ""
	if next != "" {
		if fixture.key == "recordsets" {
			links = fmt.Sprintf(`,"links":{"next":%q}`, next)
		} else {
			links = fmt.Sprintf(`,%q:[{"rel":"next","href":%q}]`, fixture.key+"_links", next)
		}
	}
	return fmt.Sprintf(`{%q:[%s]%s}`, fixture.key, rows, links)
}

func generatedListRow(id, name, status string) string {
	return fmt.Sprintf(`{"id":%q,"name":%q,"status":%q,"provisioning_status":%q}`, id, name, status, status)
}

func TestGeneratedListControlsPreserveFiltersFixedParentAndExplicitSize(t *testing.T) {
	for _, fixture := range generatedListFixtures() {
		for _, size := range []int{0, 7} {
			t.Run(fmt.Sprint(fixture.key, "/size=", size), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Mux.HandleFunc("GET "+fixture.path, func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					wantName := "web[1]."
					if fixture.regexName {
						wantName = "^" + regexp.QuoteMeta(wantName) + "$"
					}
					query := r.URL.Query()
					wantLimit, wantStatus := "", "active"
					if size > 0 {
						wantLimit = fmt.Sprint(size)
					}
					if fixture.localStatus {
						wantStatus = ""
					}
					if query.Get("name") != wantName || query.Get("status") != wantStatus || query.Get("limit") != wantLimit || query.Get("vendor") != "retained" || query.Has("max_items") || query.Has("paginated") {
						t.Error("controlled path changed query remapping or invented a limit hint", r.URL)
					}
					rows := generatedListRow("other", "other", "ERROR") + "," + generatedListRow("match", "web[1].", "ACTIVE") + "," + generatedListRow("unused", "web[1].", "ACTIVE")
					testcloud.JSON(w, 200, generatedListBody(fixture, rows, cloud.Server.URL+r.URL.RequestURI()))
				})
				access := fixture.new(t, cloud.Client(fixture.service, fixture.endpoint))
				options := []resource.ListOption{resource.WithName("web[1]."), resource.WithStatus("active"), resource.WithQuery("vendor", "retained"), resource.WithMaxItems(2)}
				if size > 0 {
					options = append(options, resource.WithPageSize(size))
				}
				ids, err := access.all(context.Background(), options...)
				if err != nil || len(ids) != 1 || ids[0] != "match" || requests.Load() != 1 {
					t.Fatal("cap must count before common filters and stop before a cyclic next", ids, err, requests.Load())
				}
			})
		}
	}
}

func TestGeneratedListControlsSinglePageAndConsumerBreakSkipUnusedNext(t *testing.T) {
	for _, fixture := range generatedListFixtures() {
		for _, mode := range []string{"single-page", "break"} {
			t.Run(fixture.key+"/"+mode, func(t *testing.T) {
				cloud, nextCloud := testcloud.New(t), testcloud.New(t)
				var requests, nextRequests atomic.Int32
				nextCloud.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { nextRequests.Add(1); w.WriteHeader(500) })
				cloud.Mux.HandleFunc("GET "+fixture.path, func(w http.ResponseWriter, _ *http.Request) {
					requests.Add(1)
					testcloud.JSON(w, 200, generatedListBody(fixture, generatedListRow("one", "one", "ACTIVE")+","+generatedListRow("two", "two", "ACTIVE"), nextCloud.Server.URL+fixture.path))
				})
				access := fixture.new(t, cloud.Client(fixture.service, fixture.endpoint))
				options := []resource.ListOption(nil)
				if mode == "single-page" {
					options = append(options, resource.WithPaginated(false))
				}
				seen := 0
				for _, err := range access.list(context.Background(), options...) {
					if err != nil {
						t.Fatal(err)
					}
					seen++
					if mode == "break" {
						break
					}
				}
				want := 2
				if mode == "break" {
					want = 1
				}
				if seen != want || requests.Load() != 1 || nextRequests.Load() != 0 {
					t.Fatal("unused next was followed", seen, requests.Load(), nextRequests.Load())
				}
			})
		}
	}
}

func TestGeneratedListControlsAcrossPagesAndDefaultCycleGuard(t *testing.T) {
	for _, fixture := range generatedListFixtures() {
		for _, mode := range []string{"cross-page-cap", "zero-unbounded", "late-http", "cycle"} {
			t.Run(fixture.key+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Mux.HandleFunc("GET "+fixture.path, func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if r.URL.Query().Has("limit") {
						t.Error("native cap invented a limit", r.URL)
					}
					if mode == "cycle" {
						testcloud.JSON(w, 200, generatedListBody(fixture, generatedListRow("one", "one", "ACTIVE"), cloud.Server.URL+r.URL.RequestURI()))
						return
					}
					if r.URL.Query().Get("marker") == "" {
						testcloud.JSON(w, 200, generatedListBody(fixture, generatedListRow("one", "one", "ACTIVE"), cloud.Server.URL+fixture.path+"?marker=next"))
						return
					}
					if mode == "late-http" {
						testcloud.JSON(w, 503, `{"error":"second page"}`)
						return
					}
					testcloud.JSON(w, 200, generatedListBody(fixture, generatedListRow("two", "two", "ACTIVE")+","+generatedListRow("three", "three", "ACTIVE"), ""))
				})
				access := fixture.new(t, cloud.Client(fixture.service, fixture.endpoint))
				maximum := 0
				if mode == "cross-page-cap" || mode == "late-http" {
					maximum = 2
				}
				seen := 0
				var finalErr error
				for _, err := range access.list(context.Background(), resource.WithMaxItems(maximum)) {
					if err != nil {
						finalErr = err
					} else {
						seen++
					}
				}
				switch mode {
				case "cross-page-cap":
					if seen != 2 || finalErr != nil || requests.Load() != 2 {
						t.Fatal(seen, finalErr, requests.Load())
					}
				case "zero-unbounded":
					if seen != 3 || finalErr != nil || requests.Load() != 2 {
						t.Fatal(seen, finalErr, requests.Load())
					}
				case "late-http":
					if seen != 1 || !gophercloud.ResponseCodeIs(finalErr, 503) || requests.Load() != 2 {
						t.Fatal(seen, finalErr, requests.Load())
					}
				case "cycle":
					if seen != 1 || !errors.Is(finalErr, resource.ErrPaginationCycle) || requests.Load() != 1 {
						t.Fatal(seen, finalErr, requests.Load())
					}
				}
			})
		}
	}
}

func TestGeneratedListControlsRetainWholePageDecodeAndCancellation(t *testing.T) {
	for _, fixture := range generatedListFixtures() {
		for _, mode := range []string{"native-page-decode", "cancel-at-cap", "negative"} {
			t.Run(fixture.key+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Mux.HandleFunc("GET "+fixture.path, func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					rows := generatedListRow("one", "one", "ACTIVE")
					if mode == "native-page-decode" {
						rows += `,false`
					}
					testcloud.JSON(w, 200, generatedListBody(fixture, rows, cloud.Server.URL+r.URL.RequestURI()))
				})
				access := fixture.new(t, cloud.Client(fixture.service, fixture.endpoint))
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				maximum := 1
				if mode == "negative" {
					maximum = -1
				}
				seen := 0
				var finalErr error
				for _, err := range access.list(ctx, resource.WithMaxItems(maximum)) {
					if err != nil {
						finalErr = err
					} else {
						seen++
						cancel()
					}
				}
				if mode == "negative" {
					if seen != 0 || !errors.Is(finalErr, resource.ErrInvalidOption) || requests.Load() != 0 {
						t.Fatal(seen, finalErr, requests.Load())
					}
				} else if mode == "cancel-at-cap" {
					if seen != 1 || !errors.Is(finalErr, context.Canceled) || requests.Load() != 1 {
						t.Fatal(seen, finalErr, requests.Load())
					}
				} else if seen != 0 || finalErr == nil || requests.Load() != 1 {
					t.Fatal("whole-page native decode must retain an unused row's error", seen, finalErr, requests.Load())
				}
			})
		}
	}
}

func TestGeneratedListControlsOptionlessListKeepsQueryCapability(t *testing.T) {
	cloud := testcloud.New(t)
	var requests atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/nova/v2.1/project/os-aggregates", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if len(r.URL.Query()) != 0 {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"aggregates":[{"id":1,"name":"one"},{"id":2,"name":"two"}]}`)
	})
	collection := aggregates.New(cloud.Client("compute", "/reverse/nova/v2.1/project")).Resources
	rows, err := collection.All(context.Background(), resource.WithMaxItems(1), resource.WithPaginated(false))
	if err != nil || len(rows) != 1 || rows[0].ID != 1 || requests.Load() != 1 {
		t.Fatal(rows, err, requests.Load())
	}
	rows, err = collection.All(context.Background(), resource.WithQuery("vendor", "unsupported"), resource.WithPaginated(false))
	if rows != nil || !errors.Is(err, resource.ErrUnsupported) || requests.Load() != 1 {
		t.Fatal("local control must not invent query capability", rows, err, requests.Load())
	}
}

func TestGeneratedListPublicZeroControlKeepsDesignateHeaderBuilder(t *testing.T) {
	cloud := testcloud.New(t)
	var requests atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/designate/v2/zones/parent/recordsets", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("X-Auth-All-Projects") != "true" || r.Header.Get("X-Auth-Sudo-Tenant-ID") != "project" || r.Header.Get("X-Vendor") != "retained" || r.URL.Query().Get("limit") != "3" || r.URL.Query().Get("vendor") != "value" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"recordsets":[{"id":"one","name":"one"}]}`)
	})
	client := cloud.Client("dns", "/reverse/designate/v2")
	seen := 0
	for _, err := range recordsets.New(client).ListByZone(context.Background(), "parent",
		recordsets.WithListByZoneOptions(recordsets.ListOpts{Limit: 3, AllProjects: true, SudoTenantID: "project"}),
		recordsets.WithListByZoneHeader("X-Vendor", "retained"),
		recordsets.WithListByZoneQuery("vendor", "value")) {
		if err != nil {
			t.Fatal(err)
		}
		seen++
	}
	if seen != 1 || requests.Load() != 1 {
		t.Fatal(seen, requests.Load())
	}
}
