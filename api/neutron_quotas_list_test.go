package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/quotas"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestNeutronQuotasListIsLazySinglePageAndKeepsProjectRawMetadata(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/neutron/v2.0/quotas", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.URL.RawQuery != "" || r.Header.Get("X-Configured") != "original" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Errorf("list request=%s %s headers=%v", r.Method, r.URL, r.Header)
		}
		w.Header().Set("X-Openstack-Request-Id", "list-quotas")
		testcloud.JSON(w, 200, `{"quotas":[{"project_id":"one","tenant_id":"one","network":-1,"port":null,"vendor":{"big":9007199254740993},"optional":null},{"tenant_id":"two","network":5,"port":0,"vendor":{"big":9007199254740995}}],"metadata":{"future":true},"links":{"self":"ignored","next":null},"quotas_links":[{"rel":"self","href":"ignored"}]}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("list requested another endpoint: %s", r.URL) })
	client := cloud.Client("network", "/catalog")
	client.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + "/neutron/v2.0")
	client.MoreHeaders = map[string]string{"X-Configured": "original"}
	api := quotas.New(client)
	sequence := api.ListProjects(context.Background())
	if calls.Load() != 0 {
		t.Fatal("creating a sequence made an HTTP request")
	}
	var values []*quotas.QuotaResource
	for value, err := range sequence {
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, value)
	}
	if len(values) != 2 || calls.Load() != 1 {
		t.Fatalf("values=%+v calls=%d", values, calls.Load())
	}
	one, two := values[0], values[1]
	if one.ProjectID != "one" || one.Network != -1 || one.Port != 0 || string(one.Body["port"]) != "null" || string(one.Body["vendor"]) != `{"big":9007199254740993}` || string(one.Body["optional"]) != "null" || one.Header.Get("X-Openstack-Request-Id") != "list-quotas" || one.StatusCode != 200 || two.ProjectID != "two" || two.Network != 5 || two.StatusCode != 200 {
		t.Fatalf("one=%+v two=%+v", one, two)
	}
	if _, exists := two.Body["project_id"]; exists {
		t.Fatal("legacy identity was copied into the raw response")
	}
	one.Header.Set("X-Openstack-Request-Id", "caller-change")
	one.Body["vendor"][0] = '['
	if two.Header.Get("X-Openstack-Request-Id") != "list-quotas" || string(two.Body["vendor"]) != `{"big":9007199254740995}` {
		t.Fatal("rows shared mutable metadata")
	}
	again, err := api.AllProjects(context.Background())
	if err != nil || len(again) != 2 || string(again[0].Body["vendor"]) != `{"big":9007199254740993}` || again[0].Header.Get("X-Openstack-Request-Id") != "list-quotas" || calls.Load() != 2 {
		t.Fatalf("again=%+v calls=%d error=%v", again, calls.Load(), err)
	}
	if client.MoreHeaders["X-Configured"] != "original" || client.ResourceBase != gophercloud.NormalizeURL(cloud.Server.URL+"/neutron/v2.0") || client.Microversion != "" {
		t.Fatal("ListProjects changed selected client configuration")
	}
}

func TestNeutronQuotasListBreakAndLocalOptionsDoNotSendServerPaging(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/neutron/v2.0/quotas", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.RawQuery != "" {
			t.Errorf("unsupported query/filter/paging reached Neutron: %s", r.URL)
		}
		testcloud.JSON(w, 200, `{"quotas":[{"project_id":"one","network":10},{"project_id":"two","network":20},null]}`)
	})
	api := quotas.New(cloud.Client("network", "/neutron/v2.0"))
	count := 0
	for value, err := range api.ListProjects(context.Background()) {
		if err != nil || value.ProjectID != "one" {
			t.Fatalf("first=%+v error=%v", value, err)
		}
		count++
		break
	}
	if count != 1 || calls.Load() != 1 {
		t.Fatalf("break count=%d calls=%d", count, calls.Load())
	}
	option := quotas.WithListProjectID("two")
	for range 2 {
		values, err := api.AllProjects(context.Background(), quotas.WithListProjectID("one"), option, quotas.WithListMaxItems(2), quotas.WithListMaxItems(1))
		if err != nil || len(values) != 1 || values[0].ProjectID != "two" || values[0].Network != 20 {
			t.Fatalf("filtered values=%+v error=%v", values, err)
		}
	}
	if calls.Load() != 3 {
		t.Fatalf("local filter caused extra requests: %d", calls.Load())
	}
	options := []quotas.ListOption{quotas.WithListProjectID("one"), quotas.WithListMaxItems(1)}
	sequence := api.ListProjects(context.Background(), options...)
	options[0] = quotas.WithListProjectID("two")
	for value, err := range sequence {
		if err != nil || value.ProjectID != "one" {
			t.Fatalf("sequence borrowed caller option slice: value=%+v error=%v", value, err)
		}
	}
	// A local filter does not silently discard malformed project rows it visits.
	values, err := api.AllProjects(context.Background(), quotas.WithListProjectID("absent"))
	if values != nil || err == nil {
		t.Fatalf("malformed filtered row ignored: values=%v error=%v", values, err)
	}
}

func TestNeutronQuotasListRejectsMalformedEnvelopeRowsAndIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		typedCause bool
	}{
		{"missing collection", `{}`, false},
		{"null collection", `{"quotas":null}`, false},
		{"object collection", `{"quotas":{}}`, true},
		{"scalar collection", `{"quotas":3}`, true},
		{"array envelope", `[]`, true},
		{"null row", `{"quotas":[null]}`, false},
		{"array row", `{"quotas":[[]]}`, true},
		{"scalar row", `{"quotas":[false]}`, true},
		{"missing identity", `{"quotas":[{"network":1}]}`, false},
		{"empty identity", `{"quotas":[{"project_id":""}]}`, false},
		{"null identity", `{"quotas":[{"project_id":null,"tenant_id":"legacy"}]}`, false},
		{"numeric identity", `{"quotas":[{"project_id":1}]}`, true},
		{"object identity", `{"quotas":[{"project_id":{}}]}`, true},
		{"unsafe identity", `{"quotas":[{"project_id":"bad/path"}]}`, false},
		{"encoded identity", `{"quotas":[{"project_id":"bad%2Fpath"}]}`, false},
		{"conflicting identities", `{"quotas":[{"project_id":"one","tenant_id":"two"}]}`, false},
		{"invalid legacy identity", `{"quotas":[{"project_id":"one","tenant_id":null}]}`, false},
		{"bad native limit", `{"quotas":[{"project_id":"one","port":"bad"}]}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/neutron/v2.0/quotas", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, tc.body) })
			values, err := quotas.New(cloud.Client("network", "/neutron/v2.0")).AllProjects(context.Background())
			if values != nil || err == nil {
				t.Fatalf("values=%+v error=%v", values, err)
			}
			if tc.typedCause {
				var cause *json.UnmarshalTypeError
				if !errors.As(err, &cause) {
					t.Fatalf("decode cause lost: %v", err)
				}
			}
		})
	}
}

func TestNeutronQuotasListSinglePageContinuationPolicyPreservesOtherMetadata(t *testing.T) {
	for _, tc := range []struct {
		name, extra string
		unsupported bool
	}{
		{"no metadata", ``, false},
		{"empty next", `,"quotas_links":[{"rel":"next","href":""}]`, false},
		{"previous link", `,"quotas_links":[{"rel":"previous","href":"https://invalid.example/prev"}]`, false},
		{"unknown links schema", `,"quotas_links":{"vendor":true},"links":[{"vendor":true}]`, false},
		{"unknown root schema", `,"links":{"next":false},"vendor":{"next":"not a continuation"}`, false},
		{"Neutron next", `,"quotas_links":[{"rel":"next","href":"https://invalid.example/next"}]`, true},
		{"next with unknown neighbor", `,"quotas_links":[{"rel":42},null,false,{"rel":"next","href":"https://invalid.example/next"}]`, true},
		{"root next", `,"links":{"next":"https://invalid.example/next"}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/neutron/v2.0/quotas", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, `{"quotas":[]`+tc.extra+`}`)
			})
			values, err := quotas.New(cloud.Client("network", "/neutron/v2.0")).AllProjects(context.Background())
			if tc.unsupported {
				if values != nil || !errors.Is(err, resource.ErrUnsupported) {
					t.Fatalf("values=%v error=%v", values, err)
				}
			} else if err != nil || values == nil || len(values) != 0 {
				t.Fatalf("empty list=%v error=%v", values, err)
			}
			if calls.Load() != 1 {
				t.Fatalf("continuation caused %d requests", calls.Load())
			}
		})
	}
}

func TestNeutronQuotasListPreflightOptionsAndHTTPFailures(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Errorf("invalid configuration made HTTP request: %s", r.URL)
	})
	api := quotas.New(cloud.Client("network", "/neutron/v2.0"))
	for _, option := range []quotas.ListOption{nil, quotas.WithListProjectID(""), quotas.WithListProjectID("../path"), quotas.WithListMaxItems(0), quotas.WithListMaxItems(-1)} {
		sequence := api.ListProjects(context.Background(), option)
		if calls.Load() != 0 {
			t.Fatal("creating an invalid sequence made HTTP request")
		}
		seen := 0
		for value, err := range sequence {
			seen++
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("value=%v error=%v", value, err)
			}
		}
		if seen != 1 {
			t.Fatalf("invalid option yielded %d results", seen)
		}
	}
	for _, invalid := range []*quotas.API{nil, quotas.New(nil), quotas.New(&gophercloud.ServiceClient{})} {
		if values, err := invalid.AllProjects(context.Background()); values != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("invalid API values=%v error=%v", values, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("preflight reached HTTP")
	}
	for _, status := range []int{401, 403, 404, 204, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			failureCloud := testcloud.New(t)
			failureCloud.Mux.HandleFunc("/neutron/v2.0/quotas", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Openstack-Request-Id", "list-denied")
				testcloud.JSON(w, status, `{"error":"denied"}`)
			})
			values, err := quotas.New(failureCloud.Client("network", "/neutron/v2.0")).AllProjects(context.Background())
			var cause gophercloud.ErrUnexpectedResponseCode
			if values != nil || !errors.As(err, &cause) || cause.Actual != status || cause.ResponseHeader.Get("X-Openstack-Request-Id") != "list-denied" {
				t.Fatalf("values=%v HTTP cause=%+v error=%v", values, cause, err)
			}
			if status != 204 && string(cause.Body) != `{"error":"denied"}` {
				t.Fatalf("HTTP body lost: %s", cause.Body)
			}
		})
	}
}

func TestNeutronQuotasListCancellationAndPartialIteration(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/neutron/v2.0/quotas", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"quotas":[{"project_id":"one"},{"project_id":"two"}]}`)
	})
	api := quotas.New(cloud.Client("network", "/neutron/v2.0"))
	ctx, cancel := context.WithCancel(context.Background())
	sequence := api.ListProjects(ctx)
	cancel()
	if values, err := api.AllProjects(ctx); values != nil || !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatalf("preflight values=%v calls=%d error=%v", values, calls.Load(), err)
	}
	for value, err := range sequence {
		if value != nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("delayed consumption=%v error=%v", value, err)
		}
	}
	ctx, cancel = context.WithCancel(context.Background())
	seen := 0
	for value, err := range api.ListProjects(ctx) {
		seen++
		if seen == 1 {
			if err != nil || value.ProjectID != "one" {
				t.Fatalf("first=%v error=%v", value, err)
			}
			cancel()
		} else if value != nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("second=%v error=%v", value, err)
		}
	}
	if seen != 2 || calls.Load() != 1 {
		t.Fatalf("seen=%d calls=%d", seen, calls.Load())
	}
	// AllProjects does not return a misleading partial success slice.
	badCloud := testcloud.New(t)
	badCloud.Mux.HandleFunc("/neutron/v2.0/quotas", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"quotas":[{"project_id":"one"},null]}`)
	})
	if values, err := quotas.New(badCloud.Client("network", "/neutron/v2.0")).AllProjects(context.Background()); values != nil || err == nil {
		t.Fatalf("partial result=%v error=%v", values, err)
	}
	blocked := testcloud.New(t)
	started, release := make(chan struct{}, 1), make(chan struct{})
	defer close(release)
	blocked.Mux.HandleFunc("/neutron/v2.0/quotas", func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})
	ctx, cancel = context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := quotas.New(blocked.Client("network", "/neutron/v2.0")).AllProjects(ctx); done <- err }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("list request did not start")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel cause=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("list request did not cancel")
	}
	timeout, stop := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer stop()
	if values, err := quotas.New(blocked.Client("network", "/neutron/v2.0")).AllProjects(timeout); values != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout values=%v error=%v", values, err)
	}
}
