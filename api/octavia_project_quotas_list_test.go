package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/loadbalancer/v2/quotas"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestOctaviaQuotaListProjectsQuerySnapshotPaginationAndRawValues(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/lbaas/quotas", func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			query := r.URL.Query()
			if query.Get("project_id") != "project-a" || query.Get("limit") != "2" || query.Get("marker") != "before" || query.Get("page_reverse") != "False" || !reflect.DeepEqual(query["fields"], []string{"loadbalancer", "project_id", "vendor"}) {
				t.Error(r.URL)
			}
			w.Header().Set("X-Openstack-Request-Id", "first-page")
			testcloud.JSON(w, 200, `{"quotas":[{"project_id":"project-a","loadbalancer":0,"listener":null,"vendor":9007199254740993},{"project_id":"project-b","load_balancer":-1}],"quotas_links":[{"rel":"next","href":"?marker=next"}]}`)
		} else if n == 2 {
			query := r.URL.Query()
			if query.Get("marker") != "next" || query.Get("project_id") != "project-a" || query.Get("limit") != "2" || query.Get("page_reverse") != "False" || !reflect.DeepEqual(query["fields"], []string{"loadbalancer", "project_id", "vendor"}) {
				t.Error(r.URL)
			}
			w.Header().Set("X-Openstack-Request-Id", "empty-page")
			testcloud.JSON(w, 200, `{"quotas":[],"quotas_links":[{"rel":"next","href":"?marker=last"}]}`)
		} else if n == 3 {
			query := r.URL.Query()
			if query.Get("marker") != "last" || query.Get("project_id") != "project-a" || query.Get("limit") != "2" || !reflect.DeepEqual(query["fields"], []string{"loadbalancer", "project_id", "vendor"}) {
				t.Error(r.URL)
			}
			w.Header().Set("X-Openstack-Request-Id", "last-page")
			// A fields projection may omit project_id. It must not be filled
			// from the original collection filter or authentication record.
			testcloud.JSON(w, 200, `{"quotas":[{"pool":4,"optional":null}]}`)
		} else {
			t.Error("unexpected additional page", n)
			http.Error(w, "unexpected", 500)
		}
	})
	fields, reverse := []string{"loadbalancer", "project_id", "vendor"}, false
	option := quotas.WithProjectListOptions(quotas.ProjectListOpts{ProjectID: "project-a", Fields: fields, Limit: 2, Marker: "before", PageReverse: &reverse})
	fields[0], reverse = "changed", true
	api := quotas.New(cloud.Client("load-balancer", "/v2"))
	options := []quotas.ProjectListOption{option}
	iterator := api.ListProjects(context.Background(), options...)
	options[0] = quotas.WithProjectListOptions(quotas.ProjectListOpts{ProjectID: "mutated"})
	if calls.Load() != 0 {
		t.Fatal("iterator performed eager HTTP")
	}
	var values []*quotas.QuotaResource
	for value, err := range iterator {
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, value)
	}
	if len(values) != 3 || calls.Load() != 3 || values[0].ProjectID != "project-a" || values[0].Loadbalancer != 0 || values[1].ProjectID != "project-b" || values[1].Loadbalancer != -1 || values[2].ProjectID != "" || values[2].Pool != 4 {
		t.Fatal(values, calls.Load())
	}
	if values[0].StatusCode != 200 || values[0].Header.Get("X-Openstack-Request-Id") != "first-page" || values[2].Header.Get("X-Openstack-Request-Id") != "last-page" || string(values[0].Body["vendor"]) != "9007199254740993" || string(values[0].Body["listener"]) != "null" || string(values[2].Body["optional"]) != "null" {
		t.Fatal(values)
	}
	values[0].Header.Set("X-Openstack-Request-Id", "changed")
	values[0].Body["vendor"][0] = 'X'
	if values[1].Header.Get("X-Openstack-Request-Id") != "first-page" || values[2].Header.Get("X-Openstack-Request-Id") != "last-page" {
		t.Fatal("collection items share mutable metadata")
	}
	// Applying a reusable option must also return independent private copies.
	first, err := request.Apply(quotas.ProjectListOpts{}, option)
	if err != nil {
		t.Fatal(err)
	}
	first.Options.Fields[0], *first.Options.PageReverse = "mutated-private", true
	second, err := request.Apply(quotas.ProjectListOpts{}, option)
	if err != nil || second.Options.Fields[0] != "loadbalancer" || *second.Options.PageReverse {
		t.Fatal(second, err)
	}
}

func TestOctaviaQuotaListReverseUsesServerBooleanAndCarriesQuery(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/lbaas/quotas", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("project_id") != "project-a" || query.Get("page_reverse") != "True" || !reflect.DeepEqual(query["fields"], []string{"project_id", "listener"}) {
			t.Errorf("filter, projection or reverse lost: %s", r.URL)
		}
		switch calls.Add(1) {
		case 1:
			if query.Get("limit") != "1" || query.Get("marker") != "" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, `{"quotas":[{"project_id":"project-a","listener":1}],"quotas_links":[{"rel":"next","href":"?marker=next&limit=2"}]}`)
		case 2:
			if query.Get("limit") != "2" || query.Get("marker") != "next" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, `{"quotas":[{"project_id":"project-a","listener":2}]}`)
		default:
			t.Error("unexpected page", r.URL)
		}
	})
	reverse := true
	values, err := quotas.New(cloud.Client("load-balancer", "/v2")).AllProjects(context.Background(), quotas.WithProjectListOptions(quotas.ProjectListOpts{ProjectID: "project-a", Fields: []string{"project_id", "listener"}, Limit: 1, PageReverse: &reverse}))
	if err != nil || len(values) != 2 || calls.Load() != 2 {
		t.Fatal(values, err, calls.Load())
	}
}

func TestOctaviaQuotaListBreakSkipsLaterDecodeAndNextHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/lbaas/quotas", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		// Both the second item and link envelope are invalid, but neither
		// belongs to the consumed part of a terminated iterator.
		testcloud.JSON(w, 200, `{"quotas":[{"project_id":"first","pool":2},{"pool":"invalid"}],"quotas_links":"invalid"}`)
	})
	count := 0
	for value, err := range quotas.New(cloud.Client("load-balancer", "/v2")).ListProjects(context.Background()) {
		if err != nil || value.ProjectID != "first" {
			t.Fatal(value, err)
		}
		count++
		break
	}
	if count != 1 || calls.Load() != 1 {
		t.Fatal(count, calls.Load())
	}
}

func TestOctaviaQuotaListCancellationStopsAdditionalHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/lbaas/quotas", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"quotas":[{"project_id":"first"},{"project_id":"second"}],"quotas_links":[{"rel":"next","href":"?marker=next"}]}`)
	})
	api := quotas.New(cloud.Client("load-balancer", "/v2"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if values, err := api.AllProjects(ctx); values != nil || !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatal(values, err, calls.Load())
	}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	values, failures := 0, 0
	for value, err := range api.ListProjects(ctx) {
		if err != nil {
			if value != nil || !errors.Is(err, context.Canceled) {
				t.Fatal(value, err)
			}
			failures++
			continue
		}
		if value.ProjectID != "first" {
			t.Fatal(value)
		}
		values++
		cancel()
	}
	if values != 1 || failures != 1 || calls.Load() != 1 {
		t.Fatal(values, failures, calls.Load())
	}
}

func TestOctaviaQuotaListCycleOriginAndConflictingLinks(t *testing.T) {
	for _, test := range []struct {
		name, link string
		want       error
		cycle      bool
	}{
		{"cycle canonical query", `{"rel":"next","href":"?page_reverse=False&limit=1"}`, nil, true},
		{"cross origin", `{"rel":"next","href":"https://other.invalid/v2/lbaas/quotas"}`, resource.ErrUnsupported, false},
		{"credentials", `{"rel":"next","href":"https://user:secret@other.invalid/v2/lbaas/quotas"}`, resource.ErrUnsupported, false},
		{"two next links", `{"rel":"next","href":"?marker=a"},{"rel":"next","href":"?marker=b"}`, resource.ErrInvalidOption, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			body := fmt.Sprintf(`{"quotas":[],"quotas_links":[%s]}`, test.link)
			cloud.Mux.HandleFunc("GET /v2/lbaas/quotas", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Openstack-Request-Id", "links")
				testcloud.JSON(w, 200, body)
			})
			reverse := false
			values, err := quotas.New(cloud.Client("load-balancer", "/v2")).AllProjects(context.Background(), quotas.WithProjectListOptions(quotas.ProjectListOpts{Limit: 1, PageReverse: &reverse}))
			if values != nil || err == nil || calls.Load() != 1 {
				t.Fatal(values, err, calls.Load())
			}
			if test.cycle {
				var cycle *resource.PaginationCycleError
				if !errors.As(err, &cycle) {
					t.Fatal(err)
				}
			} else {
				var decode *quotas.QuotaResponseError
				if !errors.Is(err, test.want) || !errors.As(err, &decode) || decode.StatusCode != 200 || decode.Header.Get("X-Openstack-Request-Id") != "links" || string(decode.Body) != body {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestOctaviaQuotaListMalformedResponsesRetainEvidence(t *testing.T) {
	for _, body := range []string{
		`{}`, `null`, `{"quotas":null}`, `{"quotas":{}}`, `{"quotas":[null]}`, `{"quotas":[3]}`, `{"quotas":[{"listener":"bad"}]}`,
		`{"quotas":[{"project_id":null}]}`, `{"quotas":[{"project_id":42}]}`, `{"quotas":[{"project_id":"bad/path"}]}`,
		`{"quotas":[],"quotas_links":"bad"}`, `{"quotas":`,
	} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries.Add(1)
				return errors.New("unexpected decode retry")
			}
			cloud.Mux.HandleFunc("GET /v2/lbaas/quotas", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Openstack-Request-Id", "malformed-list")
				testcloud.JSON(w, 200, body)
			})
			values, err := quotas.New(cloud.Client("load-balancer", "/v2")).AllProjects(context.Background())
			var response *quotas.QuotaResponseError
			if values != nil || !errors.As(err, &response) || response.StatusCode != 200 || response.Header.Get("X-Openstack-Request-Id") != "malformed-list" || string(response.Body) != body || calls.Load() != 1 || retries.Load() != 0 {
				t.Fatal(values, err, calls.Load(), retries.Load())
			}
		})
	}
}

func TestOctaviaQuotaListStatusPolicyAndEmptyCollection(t *testing.T) {
	for _, status := range []int{200, 201, 202, 204, 300, 403, 404, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("GET /v2/lbaas/quotas", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				w.Header().Set("X-Openstack-Request-Id", "status-list")
				testcloud.JSON(w, status, `{"quotas":[]}`)
			})
			values, err := quotas.New(cloud.Client("load-balancer", "/v2")).AllProjects(context.Background())
			if status == 200 {
				if err != nil || values == nil || len(values) != 0 {
					t.Fatal(values, err)
				}
				return
			}
			var response gophercloud.ErrUnexpectedResponseCode
			if values != nil || !errors.As(err, &response) || response.Actual != status || response.ResponseHeader.Get("X-Openstack-Request-Id") != "status-list" || errors.Is(err, resource.ErrNotFound) != (status == 404) {
				t.Fatal(values, err)
			}
		})
	}
}

func TestOctaviaQuotaListInvalidOptionsFailBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, `{"quotas":[]}`) })
	api := quotas.New(cloud.Client("load-balancer", "/v2"))
	for _, options := range [][]quotas.ProjectListOption{
		{nil},
		{quotas.WithProjectListOptions(quotas.ProjectListOpts{Limit: -1})},
		{quotas.WithProjectListOptions(quotas.ProjectListOpts{Marker: "bad/path"})},
		{quotas.WithProjectListOptions(quotas.ProjectListOpts{ProjectID: "bad/path"})},
		{quotas.WithProjectListOptions(quotas.ProjectListOpts{Fields: []string{" "}})},
		{request.WithQuery[quotas.ProjectListOpts]("vendor", "x")},
		{request.WithHeader[quotas.ProjectListOpts]("X-Vendor", "x")},
		{request.WithField[quotas.ProjectListOpts]("vendor", false)},
		{request.WithArgument[quotas.ProjectListOpts]("vendor", false)},
	} {
		if values, err := api.AllProjects(context.Background(), options...); values != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal(values, err, calls.Load())
		}
	}
}

func TestOctaviaQuotaDefaultInheritanceDuplicateAndForgedArgument(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("PUT /v2/lbaas/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body struct {
			Quota map[string]json.RawMessage `json:"quota"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Quota) != 1 || string(body.Quota["pool"]) != "null" {
			t.Fatal(body, err)
		}
		testcloud.JSON(w, 202, `{"quota":{"pool":null}}`)
	})
	scope := newOctaviaQuotaScope(t, cloud)
	value, err := scope.Update(context.Background(), quotas.UpdateOpts{}, quotas.WithDefaultLimit(quotas.LimitPools), quotas.WithDefaultLimit(quotas.LimitPools))
	if err != nil || value == nil || value.Pool != 0 || string(value.Body["pool"]) != "null" || calls.Load() != 1 {
		t.Fatal(value, err, calls.Load())
	}
	for _, forged := range []quotas.QuotaLimit{"unknown", "project_id", "id"} {
		option := request.WithArgument[quotas.UpdateOpts]("quota_default_limits", []quotas.QuotaLimit{forged})
		if value, err := scope.Update(context.Background(), quotas.UpdateOpts{}, option); value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
			t.Fatal(value, err, calls.Load())
		}
	}
}
