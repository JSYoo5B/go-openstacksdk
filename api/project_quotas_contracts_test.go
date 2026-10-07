package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/compute/v2/quotasets"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	tokens2 "github.com/gophercloud/gophercloud/v2/openstack/identity/v2/tokens"
	tokens3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

const projectQuotaLimits = `{"quota_set":{"id":"wire-project","cores":-1,"instances":0,"networks":8,"vendor":{"tier":"custom"},"optional":null}}`

func newProjectQuotaScope(t *testing.T, cloud *testcloud.Cloud) *quotasets.ProjectQuotaScope {
	t.Helper()
	client := cloud.Client("compute", "/nova")
	client.Microversion = "2.56"
	scope, err := quotasets.New(client).InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func TestProjectQuotaLimitsDetailUpdateAndResetPreserveTargetAndResponse(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, updates, resets atomic.Int32
	cloud.Mux.HandleFunc("/nova/os-quota-sets/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-OpenStack-Nova-API-Version") != "2.56" || r.URL.RawQuery != "" {
			t.Errorf("version=%q query=%q", r.Header.Get("X-OpenStack-Nova-API-Version"), r.URL.RawQuery)
		}
		switch r.Method {
		case http.MethodGet:
			gets.Add(1)
			w.Header().Set("X-Openstack-Request-Id", "req-get")
			testcloud.JSON(w, 200, projectQuotaLimits)
		case http.MethodPut:
			updates.Add(1)
			var body struct {
				Quota map[string]json.RawMessage `json:"quota_set"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if string(body.Quota["cores"]) != "-1" || string(body.Quota["instances"]) != "0" || string(body.Quota["force"]) != "false" || string(body.Quota["vendor"]) != `{"tier":"requested"}` || len(body.Quota) != 4 {
				t.Errorf("request quota=%s", body.Quota)
			}
			w.Header().Set("X-Openstack-Request-Id", "req-update")
			testcloud.JSON(w, 200, projectQuotaLimits)
		case http.MethodDelete:
			resets.Add(1)
			w.Header().Set("X-Openstack-Request-Id", "req-reset")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method=%s", r.Method)
		}
	})
	cloud.Mux.HandleFunc("/nova/os-quota-sets/project-fixed/detail", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Error(r.Method)
		}
		w.Header().Set("X-Openstack-Request-Id", "req-detail")
		testcloud.JSON(w, 200, `{"quota_set":{"id":"wire-project","cores":{"limit":-1,"in_use":2,"reserved":0,"vendor":"retained"},"instances":{"limit":0,"in_use":0,"reserved":0},"networks":{"limit":8,"in_use":1,"reserved":2}}}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("request escaped fixed target: %s", r.URL.Path)
		testcloud.JSON(w, 404, "{}")
	})
	scope := newProjectQuotaScope(t, cloud)
	if scope.ProjectID() != "project-fixed" || gets.Load() != 0 {
		t.Fatal("explicit ID performed a lookup")
	}
	value, err := scope.Get(context.Background())
	if err != nil || value == nil || value.ProjectID != "project-fixed" || value.ID != "wire-project" || value.Cores != -1 || value.Instances != 0 || string(value.Body["networks"]) != "8" || string(value.Body["vendor"]) != `{"tier":"custom"}` || string(value.Body["optional"]) != "null" || value.Header.Get("X-Openstack-Request-Id") != "req-get" {
		t.Fatalf("quota=%+v err=%v", value, err)
	}
	detail, err := scope.Detail(context.Background())
	if err != nil || detail == nil || detail.ProjectID != "project-fixed" || detail.ID != "wire-project" || detail.Cores.Limit != -1 || detail.Cores.InUse != 2 || detail.Cores.Reserved != 0 || detail.Instances.Limit != 0 || string(detail.Body["cores"]) != `{"in_use":2,"limit":-1,"reserved":0,"vendor":"retained"}` || string(detail.Body["networks"]) != `{"in_use":1,"limit":8,"reserved":2}` || detail.Header.Get("X-Openstack-Request-Id") != "req-detail" {
		t.Fatalf("detail=%+v err=%v", detail, err)
	}
	zero, unlimited := 0, -1
	extension := map[string]string{"tier": "requested"}
	option := quotasets.WithUpdateField("vendor", extension)
	extension["tier"] = "caller-change"
	updated, err := scope.Update(context.Background(), quotasets.UpdateOpts{Cores: &unlimited, Instances: &zero, Force: true}, option, quotasets.WithUpdateForce(false))
	if err != nil || updated == nil || updated.ProjectID != "project-fixed" || updated.ID != "wire-project" || updated.Cores != -1 || updated.Header.Get("X-Openstack-Request-Id") != "req-update" || string(updated.Body["networks"]) != "8" {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
	reset, err := scope.Reset(context.Background())
	if err != nil || reset == nil || reset.ProjectID != "project-fixed" || reset.Header.Get("X-Openstack-Request-Id") != "req-reset" || resets.Load() != 1 || updates.Load() != 1 || gets.Load() != 1 {
		t.Fatalf("reset=%+v gets=%d updates=%d deletes=%d err=%v", reset, gets.Load(), updates.Load(), resets.Load(), err)
	}
	value.Header.Set("X-Openstack-Request-Id", "caller-change")
	if updated.Header.Get("X-Openstack-Request-Id") != "req-update" || detail.Header.Get("X-Openstack-Request-Id") != "req-detail" || reset.Header.Get("X-Openstack-Request-Id") != "req-reset" {
		t.Fatal("operations share mutable response headers")
	}
}

func TestProjectQuotaResolvesExactProjectNameWithSeparateKeystoneClientOnce(t *testing.T) {
	cloud := testcloud.New(t)
	var identityCalls, computeCalls atomic.Int32
	cloud.Mux.HandleFunc("/identity/v3/projects", func(w http.ResponseWriter, r *http.Request) {
		identityCalls.Add(1)
		if r.Method != http.MethodGet || r.URL.Query().Get("name") != "tenant" || r.Header.Get("X-OpenStack-Nova-API-Version") != "" {
			t.Errorf("identity request method=%s query=%s nova-version=%s", r.Method, r.URL.RawQuery, r.Header.Get("X-OpenStack-Nova-API-Version"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"projects": []map[string]string{{"id": "wrong", "name": "tenant-extra"}}, "links": map[string]string{"next": cloud.Server.URL + "/identity/v3/projects/page2"}})
	})
	cloud.Mux.HandleFunc("/identity/v3/projects/page2", func(w http.ResponseWriter, r *http.Request) {
		identityCalls.Add(1)
		testcloud.JSON(w, 200, `{"projects":[{"id":"project-fixed","name":"tenant"}]}`)
	})
	cloud.Mux.HandleFunc("/nova/os-quota-sets/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		computeCalls.Add(1)
		testcloud.JSON(w, 200, projectQuotaLimits)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("wrong service/target for lookup: %s", r.URL.Path)
		testcloud.JSON(w, 404, "{}")
	})
	api := quotasets.New(cloud.Client("compute", "/nova"))
	scope, err := api.InProject(context.Background(), resource.Name("tenant"), quotasets.WithIdentityClient(cloud.Client("identity", "/identity/v3")))
	if err != nil || scope == nil || scope.ProjectID() != "project-fixed" || identityCalls.Load() != 2 || computeCalls.Load() != 0 {
		t.Fatalf("scope=%v identity=%d compute=%d err=%v", scope, identityCalls.Load(), computeCalls.Load(), err)
	}
	for range 2 {
		if value, err := scope.Get(context.Background()); err != nil || value.ProjectID != "project-fixed" {
			t.Fatalf("value=%+v err=%v", value, err)
		}
	}
	if identityCalls.Load() != 2 || computeCalls.Load() != 2 {
		t.Fatal("quota operation re-resolved its project")
	}
}

func TestProjectQuotaProjectLookupFailuresPreventNovaRequests(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		cause      error
	}{
		{"missing", `{"projects":[]}`, 200, resource.ErrNotFound},
		{"ambiguous", `{"projects":[{"id":"one","name":"tenant"},{"id":"two","name":"tenant"}]}`, 200, resource.ErrAmbiguous},
		{"invalid returned ID", `{"projects":[{"id":"bad/id","name":"tenant"}]}`, 200, resource.ErrInvalidOption},
		{"forbidden", `{"error":{"message":"identity denied"}}`, 403, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var identityCalls atomic.Int32
			cloud.Mux.HandleFunc("/identity/v3/projects", func(w http.ResponseWriter, r *http.Request) {
				identityCalls.Add(1)
				testcloud.JSON(w, tc.status, tc.body)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("failed project lookup called Nova: %s", r.URL.Path)
				testcloud.JSON(w, 404, "{}")
			})
			scope, err := quotasets.New(cloud.Client("compute", "/nova")).InProject(context.Background(), resource.Name("tenant"), quotasets.WithIdentityClient(cloud.Client("identity", "/identity/v3")))
			if scope != nil || err == nil || identityCalls.Load() != 1 || tc.cause != nil && !errors.Is(err, tc.cause) {
				t.Fatalf("scope=%v identity=%d err=%v", scope, identityCalls.Load(), err)
			}
			if tc.status == 403 {
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) || response.Actual != 403 || string(response.Body) != tc.body {
					t.Fatalf("response=%+v err=%v", response, err)
				}
			}
		})
	}
}

func setQuotaProjectAuth(t *testing.T, provider *gophercloud.ProviderClient, id string) {
	t.Helper()
	var auth tokens3.CreateResult
	auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": id}}}
	auth.Header = http.Header{"X-Subject-Token": []string{"project-token"}}
	if err := provider.SetTokenAndAuthResult(auth); err != nil {
		t.Fatal(err)
	}
}

func TestProjectQuotaCurrentProjectUsesRecordedV3AndV2AuthenticationAndFreezesID(t *testing.T) {
	for _, version := range []string{"v3", "v2"} {
		t.Run(version, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/nova/os-quota-sets/project-fixed", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, projectQuotaLimits)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("current project guessed/refreshed a target: %s", r.URL.Path)
				testcloud.JSON(w, 404, "{}")
			})
			if version == "v3" {
				setQuotaProjectAuth(t, cloud.Provider, "project-fixed")
			} else {
				var auth tokens2.CreateResult
				auth.Body = map[string]any{"access": map[string]any{"token": map[string]any{"id": "v2-token", "expires": "2026-10-01T00:00:00.000Z", "tenant": map[string]any{"id": "project-fixed"}}}}
				if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
					t.Fatal(err)
				}
			}
			scope, err := quotasets.New(cloud.Client("compute", "/nova")).CurrentProject(context.Background())
			if err != nil || scope == nil || scope.ProjectID() != "project-fixed" || calls.Load() != 0 {
				t.Fatalf("scope=%v calls=%d err=%v", scope, calls.Load(), err)
			}
			setQuotaProjectAuth(t, cloud.Provider, "changed-project")
			value, err := scope.Get(context.Background())
			if err != nil || value == nil || value.ProjectID != "project-fixed" || calls.Load() != 1 {
				t.Fatalf("quota=%+v calls=%d err=%v", value, calls.Load(), err)
			}
		})
	}
}

type quotaUnsupportedAuth struct{}

func (*quotaUnsupportedAuth) ExtractTokenID() (string, error) { return "custom-token", nil }

func TestProjectQuotaCurrentProjectRejectsUnscopedManualAndInvalidAuthResults(t *testing.T) {
	for _, tc := range []struct {
		name string
		auth gophercloud.AuthResult
		body any
		id   string
		want error
	}{
		{name: "manual token", want: resource.ErrUnsupported},
		{name: "unsupported auth", auth: &quotaUnsupportedAuth{}, want: resource.ErrUnsupported},
		{name: "typed nil auth", auth: (*quotaUnsupportedAuth)(nil), want: resource.ErrUnsupported},
		{name: "system", body: map[string]any{"token": map[string]any{"system": map[string]any{"all": true}}}, want: resource.ErrUnsupported},
		{name: "domain", body: map[string]any{"token": map[string]any{"domain": map[string]any{"id": "domain"}}}, want: resource.ErrUnsupported},
		{name: "empty project", body: map[string]any{"token": map[string]any{"project": map[string]any{"id": ""}}}, want: resource.ErrUnsupported},
		{name: "bad project ID", id: "project/path", want: resource.ErrInvalidOption},
		{name: "malformed project", body: map[string]any{"token": map[string]any{"project": "bad"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("invalid auth made HTTP request: %s", r.URL.Path)
			})
			if tc.auth != nil {
				if err := cloud.Provider.SetTokenAndAuthResult(tc.auth); err != nil {
					t.Fatal(err)
				}
			} else if tc.id != "" {
				setQuotaProjectAuth(t, cloud.Provider, tc.id)
			} else if tc.body != nil {
				var auth tokens3.CreateResult
				auth.Body, auth.Header = tc.body, http.Header{"X-Subject-Token": []string{"private-token"}}
				if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
					t.Fatal(err)
				}
			}
			scope, err := quotasets.New(cloud.Client("compute", "/nova/project-fixed")).CurrentProject(context.Background())
			if scope != nil || err == nil || tc.want != nil && !errors.Is(err, tc.want) || strings.Contains(err.Error(), "private-token") || strings.Contains(err.Error(), "custom-token") {
				t.Fatalf("scope=%v err=%v", scope, err)
			}
			if tc.name == "malformed project" {
				var decode *json.UnmarshalTypeError
				if !errors.As(err, &decode) {
					t.Fatalf("project decode cause lost: %v", err)
				}
			}
		})
	}
}

func TestProjectQuotaUpdateDistinguishesOmittedFalseZeroAndUnlimited(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/nova/os-quota-sets/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var envelope struct {
			Quota map[string]json.RawMessage `json:"quota_set"`
		}
		if r.Method != http.MethodPut {
			t.Error(r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Error(err)
		}
		switch calls.Load() {
		case 1:
			if len(envelope.Quota) != 0 {
				t.Errorf("nil inputs must be omitted: %s", envelope.Quota)
			}
		case 2:
			if len(envelope.Quota) != 1 || string(envelope.Quota["force"]) != "true" {
				t.Errorf("native force=true: %s", envelope.Quota)
			}
		case 3:
			if len(envelope.Quota) != 3 || string(envelope.Quota["force"]) != "false" || string(envelope.Quota["ram"]) != "-1" || string(envelope.Quota["instances"]) != "0" {
				t.Errorf("replacement/last option semantics: %s", envelope.Quota)
			}
		}
		testcloud.JSON(w, 200, projectQuotaLimits)
	})
	scope := newProjectQuotaScope(t, cloud)
	if _, err := scope.Update(context.Background(), quotasets.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	if _, err := scope.Update(context.Background(), quotasets.UpdateOpts{Force: true}); err != nil {
		t.Fatal(err)
	}
	zero, unlimited, ignored := 0, -1, 17
	if _, err := scope.Update(context.Background(), quotasets.UpdateOpts{Cores: &ignored}, quotasets.WithUpdateOptions(quotasets.UpdateOpts{RAM: &unlimited, Instances: &zero}), quotasets.WithUpdateForce(true), quotasets.WithUpdateForce(false)); err != nil || calls.Load() != 3 {
		t.Fatalf("calls=%d err=%v", calls.Load(), err)
	}
}

func TestProjectQuotaInvalidOptionsFailBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, projectQuotaLimits) })
	scope := newProjectQuotaScope(t, cloud)
	negative := -2
	for _, tc := range []struct {
		name string
		opts quotasets.UpdateOpts
		with []quotasets.UpdateOption
	}{
		{name: "invalid limit", opts: quotasets.UpdateOpts{Cores: &negative}},
		{name: "nil update option", with: []quotasets.UpdateOption{nil}},
		{name: "core collision", with: []quotasets.UpdateOption{quotasets.WithUpdateField("instances", 0)}},
		{name: "force collision", with: []quotasets.UpdateOption{quotasets.WithUpdateField("force", false)}},
		{name: "unserializable extension", with: []quotasets.UpdateOption{quotasets.WithUpdateField("vendor", make(chan bool))}},
		{name: "unsupported query", with: []quotasets.UpdateOption{request.WithQuery[quotasets.UpdateOpts]("user_id", "user")}},
		{name: "unsupported header", with: []quotasets.UpdateOption{request.WithHeader[quotasets.UpdateOpts]("X-Vendor", "value")}},
		{name: "unknown argument", with: []quotasets.UpdateOption{request.WithArgument[quotasets.UpdateOpts]("unknown", false)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, err := scope.Update(context.Background(), tc.opts, tc.with...)
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatalf("value=%v calls=%d err=%v", value, calls.Load(), err)
			}
		})
	}
	if value, err := scope.Reset(context.Background(), nil); value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
		t.Fatalf("reset=%v calls=%d err=%v", value, calls.Load(), err)
	}
	api := quotasets.New(cloud.Client("compute", "/nova"))
	for _, ref := range []resource.Ref{resource.ID("bad/path"), resource.ID(""), resource.Name("tenant")} {
		value, err := api.InProject(context.Background(), ref)
		if value != nil || err == nil || calls.Load() != 0 {
			t.Fatalf("scope=%v calls=%d err=%v", value, calls.Load(), err)
		}
	}
	for _, option := range []quotasets.ProjectOption{nil, quotasets.WithIdentityClient(nil), quotasets.WithIdentityClient(api.RawClient())} {
		value, err := api.InProject(context.Background(), resource.Name("tenant"), option)
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatalf("scope=%v calls=%d err=%v", value, calls.Load(), err)
		}
	}
}

func TestProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound(t *testing.T) {
	for _, status := range []int{404, 403} {
		for _, operation := range []string{"Get", "Detail", "Update", "Reset"} {
			t.Run(operation+http.StatusText(status), func(t *testing.T) {
				cloud := testcloud.New(t)
				body := `{"error":{"message":"quota denied"}}`
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Openstack-Request-Id", "req-denied")
					testcloud.JSON(w, status, body)
				})
				scope := newProjectQuotaScope(t, cloud)
				var err error
				switch operation {
				case "Get":
					value, cause := scope.Get(context.Background())
					err = cause
					if value != nil {
						t.Fatal(value)
					}
				case "Detail":
					value, cause := scope.Detail(context.Background())
					err = cause
					if value != nil {
						t.Fatal(value)
					}
				case "Update":
					value, cause := scope.Update(context.Background(), quotasets.UpdateOpts{})
					err = cause
					if value != nil {
						t.Fatal(value)
					}
				case "Reset":
					value, cause := scope.Reset(context.Background())
					err = cause
					if value != nil {
						t.Fatal(value)
					}
				}
				var response gophercloud.ErrUnexpectedResponseCode
				var sdk *resource.OperationError
				if !errors.As(err, &response) || response.Actual != status || string(response.Body) != body || response.ResponseHeader.Get("X-Openstack-Request-Id") != "req-denied" || !errors.As(err, &sdk) || sdk.Operation != operation || errors.Is(err, resource.ErrNotFound) != (status == 404) {
					t.Fatalf("response=%+v sdk=%+v err=%v", response, sdk, err)
				}
				if operation == "Reset" {
					value, err := scope.Reset(context.Background(), quotasets.WithResetIgnoreMissing(true))
					if value != nil || (err == nil) != (status == 404) {
						t.Fatalf("ignore reset=%v err=%v", value, err)
					}
					_, err = scope.Reset(context.Background(), quotasets.WithResetIgnoreMissing(true), quotasets.WithResetIgnoreMissing(false))
					if !gophercloud.ResponseCodeIs(err, status) {
						t.Fatalf("last reset option did not win: %v", err)
					}
				}
			})
		}
	}
}

func TestProjectQuotaRejectsMalformedQuotaObjects(t *testing.T) {
	for _, body := range []string{`{}`, `{"quota_set":null}`, `{"quota_set":[]}`, `{"quota_set":"scalar"}`, `{"quota_set":{"cores":"invalid"}}`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, body) })
			scope := newProjectQuotaScope(t, cloud)
			if value, err := scope.Get(context.Background()); value != nil || err == nil {
				t.Fatalf("get=%v err=%v", value, err)
			}
			if value, err := scope.Detail(context.Background()); value != nil || err == nil {
				t.Fatalf("detail=%v err=%v", value, err)
			}
			if value, err := scope.Update(context.Background(), quotasets.UpdateOpts{}); value != nil || err == nil {
				t.Fatalf("update=%v err=%v", value, err)
			}
		})
	}
}

func TestProjectQuotaCancellationAndTimeoutPreserveCause(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	started := make(chan struct{}, 1)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		started <- struct{}{}
		<-r.Context().Done()
	})
	scope := newProjectQuotaScope(t, cloud)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	api := quotasets.New(cloud.Client("compute", "/nova"))
	if value, err := api.InProject(ctx, resource.ID("project-fixed")); value != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("scope=%v err=%v", value, err)
	}
	if value, err := api.CurrentProject(ctx); value != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("current=%v err=%v", value, err)
	}
	if value, err := scope.Get(ctx); value != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("get=%v err=%v", value, err)
	}
	if value, err := scope.Detail(ctx); value != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("detail=%v err=%v", value, err)
	}
	if value, err := scope.Update(ctx, quotasets.UpdateOpts{}); value != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("update=%v err=%v", value, err)
	}
	if value, err := scope.Reset(ctx); value != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("reset=%v err=%v", value, err)
	}
	if calls.Load() != 0 {
		t.Fatalf("pre-canceled calls=%d", calls.Load())
	}
	ctx, cancel = context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { _, err := scope.Get(ctx); result <- err }()
	select {
	case <-started:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("quota request never reached server")
	}
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if value, err := scope.Detail(ctx); value != nil || !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 2 {
		t.Fatalf("detail=%v calls=%d err=%v", value, calls.Load(), err)
	}
}
