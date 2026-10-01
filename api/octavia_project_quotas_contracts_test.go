package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/loadbalancer/v2/quotas"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

const octaviaQuotaBody = `{"quota":{"project_id":"wire-project","load_balancer":7,"loadbalancer":-1,"health_monitor":8,"healthmonitor":0,"listener":null,"member":0,"pool":2,"l7policy":3,"l7rule":4,"vendor":{"counter":9007199254740993},"optional":null}}`

func newOctaviaQuotaScope(t *testing.T, cloud *testcloud.Cloud) *quotas.ProjectQuotaScope {
	t.Helper()
	scope, err := quotas.New(cloud.Client("load-balancer", "/v2")).InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func TestOctaviaQuotaAliasesKeepNativeNullFallbackAndRawFields(t *testing.T) {
	for _, tc := range []struct {
		name, body                  string
		loadbalancer, healthmonitor int
	}{
		{"canonical values", `{"quota":{"load_balancer":7,"loadbalancer":0,"health_monitor":8,"healthmonitor":-1}}`, 0, -1},
		{"canonical null fallback", `{"quota":{"load_balancer":7,"loadbalancer":null,"health_monitor":8,"healthmonitor":null}}`, 7, 8},
		{"both null", `{"quota":{"load_balancer":null,"loadbalancer":null,"health_monitor":null,"healthmonitor":null}}`, 0, 0},
		{"missing aliases", `{"quota":{}}`, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("GET /v2/lbaas/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, tc.body)
			})
			value, err := newOctaviaQuotaScope(t, cloud).Get(context.Background())
			if err != nil || value == nil || value.Loadbalancer != tc.loadbalancer || value.Healthmonitor != tc.healthmonitor {
				t.Fatalf("quota=%+v error=%v", value, err)
			}
			var envelope struct {
				Quota map[string]json.RawMessage `json:"quota"`
			}
			if err := json.Unmarshal([]byte(tc.body), &envelope); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(value.Body, envelope.Quota) {
				t.Fatalf("raw fields changed: %v", value.Body)
			}
		})
	}
}

func TestOctaviaProjectQuotasUseOfficialPathsAndGlobalDefaults(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, defaults, updates, resets atomic.Int32
	cloud.Mux.HandleFunc("/v2/lbaas/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Configured") != "original" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(r.Header)
		}
		switch r.Method {
		case http.MethodGet:
			gets.Add(1)
			if !reflect.DeepEqual(r.URL.Query()["fields"], []string{"loadbalancer", "vendor"}) {
				t.Error(r.URL)
			}
			w.Header().Set("X-Openstack-Request-Id", "get-quota")
			testcloud.JSON(w, 200, octaviaQuotaBody)
		case http.MethodPut:
			updates.Add(1)
			var body struct {
				Quota map[string]json.RawMessage `json:"quota"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if len(body.Quota) != 4 || string(body.Quota["loadbalancer"]) != "-1" || string(body.Quota["listener"]) != "0" || string(body.Quota["member"]) != "null" || string(body.Quota["vendor"]) != `{"enabled":false}` {
				t.Errorf("body=%s", body.Quota)
			}
			w.Header().Set("X-Openstack-Request-Id", "update-quota")
			testcloud.JSON(w, 202, octaviaQuotaBody)
		case http.MethodDelete:
			resets.Add(1)
			if r.ContentLength > 0 {
				t.Fatal("unexpected reset body")
			}
			w.Header().Set("X-Openstack-Request-Id", "reset-quota")
			w.WriteHeader(204)
		default:
			t.Error(r.Method)
		}
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/quotas/defaults", func(w http.ResponseWriter, r *http.Request) {
		defaults.Add(1)
		if r.URL.RawQuery != "" {
			t.Error(r.URL)
		}
		w.Header().Set("X-Openstack-Request-Id", "global-defaults")
		testcloud.JSON(w, 200, `{"quota":{"loadbalancer":50,"healthmonitor":-1,"listener":null,"vendor":9007199254740993}}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("incorrect quota route: %s", r.URL)
		http.Error(w, "wrong route", 404)
	})
	client := cloud.Client("load-balancer", "/catalog")
	client.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + "/v2")
	client.MoreHeaders = map[string]string{"X-Configured": "original"}
	api := quotas.New(client)
	scope, err := api.InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil || gets.Load() != 0 {
		t.Fatal(scope, err)
	}
	fields := []string{"loadbalancer", "vendor"}
	getOption := quotas.WithGetFields(fields...)
	fields[0] = "mutated"
	value, err := scope.Get(context.Background(), getOption)
	if err != nil || value == nil || value.ProjectID != "project-fixed" || value.Loadbalancer != -1 || value.Healthmonitor != 0 || value.Listener != 0 || value.StatusCode != 200 || string(value.Body["listener"]) != "null" || string(value.Body["project_id"]) != `"wire-project"` || string(value.Body["vendor"]) != `{"counter":9007199254740993}` || value.Header.Get("X-Openstack-Request-Id") != "get-quota" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	baseline, err := scope.Defaults(context.Background())
	if err != nil || baseline == nil || baseline.Loadbalancer != 50 || baseline.Healthmonitor != -1 || string(baseline.Body["vendor"]) != "9007199254740993" || baseline.Header.Get("X-Openstack-Request-Id") != "global-defaults" {
		t.Fatal(baseline, err)
	}
	zero, unlimited := 0, -1
	vendor := map[string]bool{"enabled": false}
	extension := quotas.WithUpdateField("vendor", vendor)
	vendor["enabled"] = true
	updated, err := scope.Update(context.Background(), quotas.UpdateOpts{Loadbalancer: &unlimited, Listener: &zero}, quotas.WithDefaultLimit(quotas.LimitMembers), extension)
	if err != nil || updated == nil || updated.StatusCode != 202 || updated.ProjectID != "project-fixed" || updated.Header.Get("X-Openstack-Request-Id") != "update-quota" {
		t.Fatal(updated, err)
	}
	reset, err := scope.Reset(context.Background())
	if err != nil || reset == nil || reset.ProjectID != "project-fixed" || reset.StatusCode != 204 || reset.Header.Get("X-Openstack-Request-Id") != "reset-quota" || gets.Load() != 1 || defaults.Load() != 1 || updates.Load() != 1 || resets.Load() != 1 {
		t.Fatalf("reset=%+v counts=%d/%d/%d/%d error=%v", reset, gets.Load(), defaults.Load(), updates.Load(), resets.Load(), err)
	}
	value.Header.Set("X-Openstack-Request-Id", "changed")
	value.Body["vendor"][0] = 'X'
	if updated.Header.Get("X-Openstack-Request-Id") != "update-quota" || string(updated.Body["vendor"]) != `{"counter":9007199254740993}` || baseline.Header.Get("X-Openstack-Request-Id") != "global-defaults" || client.MoreHeaders["X-Configured"] != "original" || client.ResourceBase != gophercloud.NormalizeURL(cloud.Server.URL+"/v2") {
		t.Fatal("responses/configuration share mutable state")
	}
}

func TestOctaviaQuotaScopesCorrectNativeURLWithoutChangingGeneratedAPI(t *testing.T) {
	cloud := testcloud.New(t)
	var native, scoped atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) { native.Add(1); testcloud.JSON(w, 200, octaviaQuotaBody) })
	cloud.Mux.HandleFunc("GET /v2/lbaas/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) { scoped.Add(1); testcloud.JSON(w, 200, octaviaQuotaBody) })
	api := quotas.New(cloud.Client("load-balancer", "/v2"))
	if _, err := api.Get(context.Background(), "project-fixed"); err != nil {
		t.Fatal(err)
	}
	scope, err := api.InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = scope.Get(context.Background()); err != nil {
		t.Fatal(err)
	}
	if native.Load() != 1 || scoped.Load() != 1 {
		t.Fatal(native.Load(), scoped.Load())
	}
	base := quotas.New(cloud.Client("load-balancer", "/v2/lbaas"))
	scope, err = base.InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = scope.Get(context.Background()); err != nil || scoped.Load() != 2 {
		t.Fatal(err, scoped.Load())
	}
}

func TestOctaviaQuotaSnapshotOmissionExactIntegersAndRetry(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, reauths atomic.Int32
	zero, large := 0, int(^uint(0)>>1)
	option := quotas.WithQuotaOptions(quotas.UpdateOpts{Listener: &zero, Pool: &large})
	zero, large = 99, 88
	cloud.Provider.ReauthFunc = func(context.Context) error {
		zero, large = 777, 888
		reauths.Add(1)
		cloud.Provider.SetToken("fresh")
		return nil
	}
	cloud.Mux.HandleFunc("PUT /v2/lbaas/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Quota map[string]json.RawMessage `json:"quota"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		n := calls.Add(1)
		if n == 1 {
			if len(body.Quota) != 0 {
				t.Error(body.Quota)
			}
		} else {
			if len(body.Quota) != 2 || string(body.Quota["listener"]) != "0" || string(body.Quota["pool"]) != strconv.Itoa(int(^uint(0)>>1)) {
				t.Errorf("snapshot=%s", body.Quota)
			}
		}
		if n == 2 {
			testcloud.JSON(w, 401, `{"error":"expired"}`)
			return
		}
		testcloud.JSON(w, 202, octaviaQuotaBody)
	})
	scope := newOctaviaQuotaScope(t, cloud)
	if _, err := scope.Update(context.Background(), quotas.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	var prior *int
	capture := func(config *request.Config[quotas.UpdateOpts]) error { prior = config.Options.Listener; return nil }
	if _, err := scope.Update(context.Background(), quotas.UpdateOpts{}, option, capture); err != nil {
		t.Fatal(err)
	}
	*prior = 100
	if _, err := scope.Update(context.Background(), quotas.UpdateOpts{}, option); err != nil || calls.Load() != 4 || reauths.Load() != 1 {
		t.Fatal(err, calls.Load(), reauths.Load())
	}
}

func TestOctaviaQuotaInvalidOptionsFailBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 202, octaviaQuotaBody) })
	scope := newOctaviaQuotaScope(t, cloud)
	bad := -2
	zero := 0
	for _, test := range []struct {
		name    string
		opts    quotas.UpdateOpts
		options []quotas.UpdateOption
	}{
		{"negative", quotas.UpdateOpts{Pool: &bad}, nil}, {"nil", quotas.UpdateOpts{}, []quotas.UpdateOption{nil}},
		{"core", quotas.UpdateOpts{}, []quotas.UpdateOption{quotas.WithUpdateField("listener", 0)}}, {"legacy core", quotas.UpdateOpts{}, []quotas.UpdateOption{quotas.WithUpdateField("load_balancer", 0)}},
		{"project", quotas.UpdateOpts{}, []quotas.UpdateOption{quotas.WithUpdateField("project_id", "other")}},
		{"nonserializable", quotas.UpdateOpts{}, []quotas.UpdateOption{quotas.WithUpdateField("vendor", make(chan bool))}},
		{"query", quotas.UpdateOpts{}, []quotas.UpdateOption{request.WithQuery[quotas.UpdateOpts]("vendor", "x")}},
		{"header", quotas.UpdateOpts{}, []quotas.UpdateOption{request.WithHeader[quotas.UpdateOpts]("X-Vendor", "x")}},
		{"argument", quotas.UpdateOpts{}, []quotas.UpdateOption{request.WithArgument[quotas.UpdateOpts]("unknown", true)}},
		{"unknown default", quotas.UpdateOpts{}, []quotas.UpdateOption{quotas.WithDefaultLimit("unknown")}},
		{"value/default conflict", quotas.UpdateOpts{Listener: &zero}, []quotas.UpdateOption{quotas.WithDefaultLimit(quotas.LimitListeners)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if value, err := scope.Update(context.Background(), test.opts, test.options...); value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(value, err, calls.Load())
			}
		})
	}
	if _, err := scope.Reset(context.Background(), nil); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
		t.Fatal(err)
	}
	for _, option := range []quotas.GetOption{nil, quotas.WithGetFields(" ")} {
		if _, err := scope.Get(context.Background(), option); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal(err)
		}
	}
}

func TestOctaviaQuotaSuccessCodesAndStrictMissingPolicy(t *testing.T) {
	for _, operation := range []string{"Get", "Defaults", "Update", "Reset"} {
		for _, status := range []int{200, 201, 202, 204, 403, 404, 500} {
			t.Run(fmt.Sprintf("%s/%d", operation, status), func(t *testing.T) {
				cloud := testcloud.New(t)
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Openstack-Request-Id", "response-evidence")
					if operation == "Reset" {
						w.WriteHeader(status)
					} else {
						testcloud.JSON(w, status, octaviaQuotaBody)
					}
				})
				scope := newOctaviaQuotaScope(t, cloud)
				var err error
				code := 0
				switch operation {
				case "Get":
					v, e := scope.Get(context.Background())
					err = e
					if v != nil {
						code = v.StatusCode
					}
				case "Defaults":
					v, e := scope.Defaults(context.Background())
					err = e
					if v != nil {
						code = v.StatusCode
					}
				case "Update":
					v, e := scope.Update(context.Background(), quotas.UpdateOpts{})
					err = e
					if v != nil {
						code = v.StatusCode
					}
				case "Reset":
					v, e := scope.Reset(context.Background())
					err = e
					if v != nil {
						code = v.StatusCode
					}
				}
				accepted := status == 200
				if operation == "Update" {
					accepted = status == 200 || status == 202
				}
				if operation == "Reset" {
					accepted = status == 202 || status == 204
				}
				if accepted {
					if err != nil || code != status {
						t.Fatal(code, err)
					}
					return
				}
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) || response.Actual != status || response.ResponseHeader.Get("X-Openstack-Request-Id") != "response-evidence" || errors.Is(err, resource.ErrNotFound) != (status == 404) {
					t.Fatalf("response=%+v error=%v", response, err)
				}
				if operation == "Reset" {
					value, err := scope.Reset(context.Background(), quotas.WithResetIgnoreMissing(true))
					if value != nil || (err == nil) != (status == 404) {
						t.Fatal(value, err)
					}
					_, err = scope.Reset(context.Background(), quotas.WithResetIgnoreMissing(true), quotas.WithResetIgnoreMissing(false))
					if !gophercloud.ResponseCodeIs(err, status) {
						t.Fatal(err)
					}
				}
			})
		}
	}
}

func TestOctaviaQuotaMalformedAcceptedResponsesRetainEvidenceWithoutRetry(t *testing.T) {
	for _, body := range []string{`{}`, `{"quota":null}`, `{"quota":[]}`, `{"quota":"scalar"}`, `{"quota":{"listener":"invalid"}}`, `{"quota":`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries.Add(1)
				return nil
			}
			cloud.Mux.HandleFunc("PUT /v2/lbaas/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Openstack-Request-Id", "accepted-error")
				testcloud.JSON(w, 202, body)
			})
			value, err := newOctaviaQuotaScope(t, cloud).Update(context.Background(), quotas.UpdateOpts{})
			var evidence *quotas.QuotaResponseError
			if value != nil || !errors.As(err, &evidence) || evidence.StatusCode != 202 || string(evidence.Body) != body || evidence.Header.Get("X-Openstack-Request-Id") != "accepted-error" || calls.Load() != 1 || retries.Load() != 0 {
				t.Fatalf("value=%v evidence=%+v calls/retries=%d/%d error=%v", value, evidence, calls.Load(), retries.Load(), err)
			}
		})
	}
}

func TestOctaviaQuotaCancellationAndDeadlinePreserveCause(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	defer close(release)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		started <- struct{}{}
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})
	scope := newOctaviaQuotaScope(t, cloud)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, operation := range []func(context.Context) error{func(ctx context.Context) error { _, e := scope.Get(ctx); return e }, func(ctx context.Context) error { _, e := scope.Defaults(ctx); return e }, func(ctx context.Context) error { _, e := scope.Update(ctx, quotas.UpdateOpts{}); return e }, func(ctx context.Context) error { _, e := scope.Reset(ctx); return e }} {
		if err := operation(ctx); !errors.Is(err, context.Canceled) || calls.Load() != 0 {
			t.Fatal(err, calls.Load())
		}
	}
	ctx, cancel = context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { _, err := scope.Update(ctx, quotas.UpdateOpts{}); result <- err }()
	select {
	case <-started:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("request never reached server")
	}
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := scope.Get(ctx); !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 2 {
		t.Fatal(err, calls.Load())
	}
}
