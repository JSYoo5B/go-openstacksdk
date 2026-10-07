package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/containerinfra/v1/quotas"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const magnumQuotaListRow = `{"id":9007199254740993,"project_id":"row-project","resource":"Cluster","hard_limit":0,"vendor":{"big":9007199254740993},"optional":null}`

func TestMagnumQuotaListPreservesAllTenantsExtensionsAndExactRowsAcrossPages(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /magnum/v1/quotas", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("all_tenants") != "true" || query.Get("vendor_filter") != "requested" || query.Get("sort_key") != "id" || query.Get("sort_dir") != "asc" || query.Get("limit") != "1" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(r.URL, r.Header)
		}
		switch calls.Add(1) {
		case 1:
			if query.Has("marker") {
				t.Error(query)
			}
			w.Header().Set("X-Openstack-Request-Id", "page-one")
			testcloud.JSON(w, 200, `{"quotas":[`+magnumQuotaListRow+`],"next":"?limit=1&marker=9007199254740993&sort_key=id&sort_dir=asc"}`)
		case 2:
			if query.Get("marker") != "9007199254740993" {
				t.Error(query)
			}
			w.Header().Set("X-Openstack-Request-Id", "page-two")
			testcloud.JSON(w, 200, `{"quotas":[{"id":"9007199254740994","project_id":"other-project","resource":"Cluster","hard_limit":4}],"next":"?limit=1&marker=9007199254740994&sort_key=id&sort_dir=asc"}`)
		case 3:
			if query.Get("marker") != "9007199254740994" {
				t.Error(query)
			}
			testcloud.JSON(w, 200, `{"quotas":[]}`)
		default:
			t.Error("unexpected page fetch", calls.Load())
		}
	})
	api := quotas.New(cloud.Client("container-infrastructure-management", "/catalog"))
	api.RawClient().ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + "/magnum/v1")
	values, err := api.All(context.Background(), quotas.WithListOptions(quotas.ListOpts{Limit: 1}), quotas.WithListAllTenants(true), quotas.WithListQuery("vendor_filter", "requested"))
	if err != nil || len(values) != 2 || calls.Load() != 3 {
		t.Fatal(values, err, calls.Load())
	}
	first, second := values[0], values[1]
	if first.ID != "9007199254740993" || first.ProjectID != "row-project" || first.Resource != "Cluster" || first.RequestProjectID != "" || first.RequestResource != "" || first.StatusCode != 200 || first.Header.Get("X-Openstack-Request-Id") != "page-one" || string(first.Body["vendor"]) != `{"big":9007199254740993}` || string(first.Body["optional"]) != "null" || second.ID != "9007199254740994" || second.ProjectID != "other-project" || second.Header.Get("X-Openstack-Request-Id") != "page-two" {
		t.Fatal(first, second)
	}
	first.Header.Set("X-Openstack-Request-Id", "caller-change")
	if second.Header.Get("X-Openstack-Request-Id") != "page-two" {
		t.Fatal("page headers share mutable state")
	}
}

func TestMagnumQuotaListOptionsSnapshotLazyAndReusable(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /magnum/v1/quotas", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("all_tenants") != "true" || r.URL.Query().Get("marker") != "9007199254740993" || r.URL.Query().Get("sort_key") != "resource" || r.URL.Query().Get("sort_dir") != "desc" || r.URL.Query().Has("limit") {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"quotas":[]}`)
	})
	all := true
	option := quotas.WithListOptions(quotas.ListOpts{Marker: "9007199254740993", SortKey: "resource", SortDir: "desc", AllTenants: &all})
	all = false
	var applied *bool
	capture := func(config *request.Config[quotas.ListOpts]) error { applied = config.Options.AllTenants; return nil }
	options := []quotas.ListOption{option, capture}
	api := quotas.New(cloud.Client("container-infrastructure-management", "/magnum/v1"))
	iterator := api.List(context.Background(), options...)
	options[0] = quotas.WithListAllTenants(false)
	if calls.Load() != 0 {
		t.Fatal("iterator was eager")
	}
	for range 2 {
		count := 0
		for value, err := range iterator {
			count++
			t.Fatal("empty iterator yielded", value, err)
		}
		if count != 0 || applied == nil {
			t.Fatal(count, applied)
		}
		*applied = false
	}
	if calls.Load() != 2 || all != false {
		t.Fatal(calls.Load(), all)
	}
}

func TestMagnumQuotaListAcceptsInitialServerCapAndFixesEffectiveLimit(t *testing.T) {
	for _, laterLimit := range []int{1, 2} {
		t.Run(strconv.Itoa(laterLimit), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /magnum/v1/quotas", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					if r.URL.Query().Get("limit") != "10000" {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, `{"quotas":[`+magnumQuotaListRow+`],"next":"?limit=1&marker=9007199254740993"}`)
				case 2:
					if r.URL.Query().Get("limit") != "1" {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, fmt.Sprintf(`{"quotas":[`+magnumQuotaListRow+`],"next":"?limit=%d&marker=9007199254740994"}`, laterLimit))
				default:
					if r.URL.Query().Get("limit") != "1" {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, `{"quotas":[]}`)
				}
			})
			values, err := quotas.New(cloud.Client("container-infrastructure-management", "/magnum/v1")).All(context.Background(), quotas.WithListOptions(quotas.ListOpts{Limit: 10000}))
			if laterLimit == 1 {
				if err != nil || len(values) != 2 || calls.Load() != 3 {
					t.Fatal(values, err, calls.Load())
				}
			} else if values != nil || !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 2 {
				t.Fatal(values, err, calls.Load())
			}
		})
	}
}

func TestMagnumQuotaListDefaultsFalseAndServerEffectiveLimit(t *testing.T) {
	for _, input := range []string{"omitted", "false", "last-false"} {
		t.Run(input, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /magnum/v1/quotas", func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				if q.Get("sort_key") != "id" || q.Get("sort_dir") != "asc" || q.Has("all_tenants") != (input != "omitted") || q.Get("all_tenants") != "" && q.Get("all_tenants") != "false" {
					t.Error(q)
				}
				if calls.Add(1) == 1 {
					if q.Has("limit") {
						t.Error("zero limit must be omitted", q)
					}
					testcloud.JSON(w, 200, `{"quotas":[`+magnumQuotaListRow+`],"next":"?limit=1&marker=9007199254740993&sort_dir=asc&sort_key=id"}`)
					return
				}
				if q.Get("limit") != "1" {
					t.Error("server effective page limit missing", q)
				}
				testcloud.JSON(w, 200, `{"quotas":[],"next":null}`)
			})
			var options []quotas.ListOption
			if input == "false" {
				options = []quotas.ListOption{quotas.WithListAllTenants(false)}
			}
			if input == "last-false" {
				options = []quotas.ListOption{quotas.WithListAllTenants(true), quotas.WithListAllTenants(false)}
			}
			values, err := quotas.New(cloud.Client("container-infrastructure-management", "/magnum/v1")).All(context.Background(), options...)
			if err != nil || len(values) != 1 || calls.Load() != 2 {
				t.Fatal(values, err, calls.Load())
			}
		})
	}
}

func TestMagnumQuotaListBreakSkipsBadRowsBadNextAndFurtherRequests(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /magnum/v1/quotas", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"quotas":[`+magnumQuotaListRow+`,{"hard_limit":"bad"}],"next":{}}`)
	})
	api := quotas.New(cloud.Client("container-infrastructure-management", "/magnum/v1"))
	for value, err := range api.List(context.Background()) {
		if err != nil || value.ID != "9007199254740993" {
			t.Fatal(value, err)
		}
		break
	}
	if calls.Load() != 1 {
		t.Fatal(calls.Load())
	}
	count, failures := 0, 0
	for value, err := range api.List(context.Background()) {
		count++
		if err != nil {
			failures++
			if value != nil {
				t.Fatal(value)
			}
			continue
		}
		if value.ID != "9007199254740993" {
			t.Fatal(value)
		}
	}
	if count != 2 || failures != 1 || calls.Load() != 2 {
		t.Fatal(count, failures, calls.Load())
	}
}

func TestMagnumQuotaListPreflightAndMalformedResponsesPreserveEvidence(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, `{"quotas":[]}`) })
	api := quotas.New(cloud.Client("container-infrastructure-management", "/magnum/v1"))
	for _, options := range [][]quotas.ListOption{
		{nil}, {quotas.WithListOptions(quotas.ListOpts{Limit: -1})},
		{quotas.WithListOptions(quotas.ListOpts{Marker: "0"})}, {quotas.WithListOptions(quotas.ListOpts{Marker: "project-id"})},
		{quotas.WithListOptions(quotas.ListOpts{SortKey: " "})}, {quotas.WithListOptions(quotas.ListOpts{SortDir: "ASC"})},
		{quotas.WithListQuery("all_tenants", "true")}, {quotas.WithListQuery("limit", "2")}, {quotas.WithListQuery("", "bad")},
		{request.WithField[quotas.ListOpts]("field", 1)}, {request.WithHeader[quotas.ListOpts]("X-Test", "value")}, {request.WithArgument[quotas.ListOpts]("unsupported", true)},
	} {
		if values, err := api.All(context.Background(), options...); values != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal(values, err, calls.Load())
		}
	}
	for _, body := range []string{`{}`, `null`, `[]`, `{"quotas":null}`, `{"quotas":{}}`, `{"quotas":[null]}`, `{"quotas":[{"hard_limit":1}]}`, `{"quotas":[{"id":1,"project_id":"bad/path","resource":"Cluster","hard_limit":1}]}`, `{"quotas":[{"id":1,"project_id":"p","resource":null,"hard_limit":1}]}`, `{"quotas":[],"next":{}}`} {
		t.Run(body, func(t *testing.T) {
			broken := testcloud.New(t)
			broken.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Openstack-Request-Id", "bad-page")
				testcloud.JSON(w, 200, body)
			})
			values, err := quotas.New(broken.Client("container-infrastructure-management", "/magnum/v1")).All(context.Background())
			var evidence *quotas.QuotaResponseError
			if values != nil || !errors.As(err, &evidence) || string(evidence.Body) != body || evidence.StatusCode != 200 || evidence.Header.Get("X-Openstack-Request-Id") != "bad-page" {
				t.Fatal(values, evidence, err)
			}
		})
	}
}

func TestMagnumQuotaListRejectsScopeChangingLinksAndCycles(t *testing.T) {
	for _, link := range []string{"https://foreign.invalid/magnum/v1/quotas?marker=1", "/magnum/v1/clusters?marker=1", "?marker=1&all_tenants=false", "?marker=1&sort_key=resource", "?marker=1&sort_dir=desc", "?marker=1&limit=2", "?marker=1&vendor_filter=changed", "?marker=1&unrequested=query", "?marker=1&marker=2", "?marker=1&limit=1&limit=1", "?marker=row-id", "?sort_key=id", "?marker=1#fragment", "?marker=1&bad=%zz"} {
		t.Run(link, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, fmt.Sprintf(`{"quotas":[],"next":%q}`, link))
			})
			values, err := quotas.New(cloud.Client("container-infrastructure-management", "/magnum/v1")).All(context.Background(), quotas.WithListOptions(quotas.ListOpts{Limit: 1}), quotas.WithListAllTenants(true), quotas.WithListQuery("vendor_filter", "requested"))
			if values != nil || err == nil || calls.Load() != 1 {
				t.Fatal(values, err, calls.Load())
			}
		})
	}
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /magnum/v1/quotas", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"quotas":[],"next":"?sort_dir=asc&marker=01&sort_key=id"}`)
	})
	if values, err := quotas.New(cloud.Client("container-infrastructure-management", "/magnum/v1")).All(context.Background()); values != nil || !errors.Is(err, resource.ErrPaginationCycle) || calls.Load() != 2 {
		t.Fatal(values, err, calls.Load())
	}
}

func TestMagnumQuotaListHTTPPolicyAndPageErrors(t *testing.T) {
	for _, status := range []int{201, 202, 204, 403, 404, 500} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Openstack-Request-Id", "http-page")
				testcloud.JSON(w, status, `{"error":"denied"}`)
			})
			values, err := quotas.New(cloud.Client("container-infrastructure-management", "/magnum/v1")).All(context.Background())
			var httpError gophercloud.ErrUnexpectedResponseCode
			var sdk *resource.OperationError
			if values != nil || !errors.As(err, &httpError) || httpError.Actual != status || httpError.ResponseHeader.Get("X-Openstack-Request-Id") != "http-page" || !errors.As(err, &sdk) || sdk.Operation != "List" || errors.Is(err, resource.ErrNotFound) != (status == 404) {
				t.Fatal(values, httpError, sdk, err)
			}
		})
	}
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /magnum/v1/quotas", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("marker") {
			testcloud.JSON(w, 403, `{"error":"page2"}`)
			return
		}
		testcloud.JSON(w, 200, `{"quotas":[`+magnumQuotaListRow+`],"next":"?marker=9007199254740993"}`)
	})
	if values, err := quotas.New(cloud.Client("container-infrastructure-management", "/magnum/v1")).All(context.Background()); values != nil || !gophercloud.ResponseCodeIs(err, 403) {
		t.Fatal(values, err)
	}
}

func TestMagnumQuotaListCancellationAndRedirectCannotFetchAnotherPage(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /magnum/v1/quotas", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"quotas":[`+magnumQuotaListRow+`],"next":"?marker=9007199254740993"}`)
	})
	api := quotas.New(cloud.Client("container-infrastructure-management", "/magnum/v1"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if values, err := api.All(ctx); values != nil || !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatal(values, err, calls.Load())
	}
	ctx, cancel = context.WithCancel(context.Background())
	count := 0
	for value, err := range api.List(ctx) {
		count++
		if value != nil {
			cancel()
			continue
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	}
	if count != 2 || calls.Load() != 1 {
		t.Fatal(count, calls.Load())
	}
	blocked := testcloud.New(t)
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	defer close(release)
	blocked.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})
	ctx, cancel = context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := quotas.New(blocked.Client("container-infrastructure-management", "/magnum/v1")).All(ctx)
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("list did not reach server")
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	redirect := testcloud.New(t)
	var escaped atomic.Int32
	redirect.Mux.HandleFunc("GET /magnum/v1/quotas", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("marker") {
			escaped.Add(1)
			testcloud.JSON(w, 200, `{"quotas":[]}`)
			return
		}
		http.Redirect(w, r, "?marker=1", http.StatusTemporaryRedirect)
	})
	if values, err := quotas.New(redirect.Client("container-infrastructure-management", "/magnum/v1")).All(context.Background()); values != nil || !errors.Is(err, resource.ErrInvalidOption) || escaped.Load() != 0 {
		t.Fatal(values, err, escaped.Load())
	}
}

func TestMagnumQuotaListRetainsBodyRawMessages(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /magnum/v1/quotas", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"quotas":[`+magnumQuotaListRow+`,`+magnumQuotaListRow+`]}`)
	})
	values, err := quotas.New(cloud.Client("container-infrastructure-management", "/magnum/v1")).All(context.Background())
	if err != nil || len(values) != 2 {
		t.Fatal(values, err)
	}
	values[0].Body["optional"] = json.RawMessage("1")
	values[0].Body["vendor"][0] = '['
	if string(values[1].Body["optional"]) != "null" || string(values[1].Body["vendor"]) != `{"big":9007199254740993}` {
		t.Fatal("rows share mutable raw fields")
	}
}
