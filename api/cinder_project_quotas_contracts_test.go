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

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	tokens2 "github.com/gophercloud/gophercloud/v2/openstack/identity/v2/tokens"
	tokens3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

const cinderProjectQuotaLimits = `{"quota_set":{"id":"wire-project","volumes":-1,"snapshots":0,"volumes_SSD":8,"counter":9007199254740993,"vendor":{"tier":"custom"},"optional":null}}`

func newCinderProjectQuotaScope(t *testing.T, cloud *testcloud.Cloud) *quotasets.ProjectQuotaScope {
	t.Helper()
	client := cloud.Client("block-storage", "/cinder")
	client.Microversion = "3.70"
	scope, err := quotasets.New(client).InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func TestCinderProjectQuotaResolvesExactProjectNameWithSeparateKeystoneClientOnce(t *testing.T) {
	cloud := testcloud.New(t)
	var identityCalls, cinderCalls atomic.Int32
	cloud.Mux.HandleFunc("/identity/v3/projects", func(w http.ResponseWriter, r *http.Request) {
		identityCalls.Add(1)
		if r.Method != http.MethodGet || r.URL.Query().Get("name") != "tenant" || r.Header.Get("X-OpenStack-Volume-API-Version") != "" {
			t.Errorf("identity request method=%s query=%s cinder-version=%s", r.Method, r.URL.RawQuery, r.Header.Get("X-OpenStack-Volume-API-Version"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"projects": []map[string]string{{"id": "wrong", "name": "tenant-extra"}}, "links": map[string]string{"next": cloud.Server.URL + "/identity/v3/projects/page2"}})
	})
	cloud.Mux.HandleFunc("/identity/v3/projects/page2", func(w http.ResponseWriter, r *http.Request) {
		identityCalls.Add(1)
		testcloud.JSON(w, 200, `{"projects":[{"id":"project-fixed","name":"tenant"}]}`)
	})
	cloud.Mux.HandleFunc("/cinder/os-quota-sets/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		cinderCalls.Add(1)
		testcloud.JSON(w, 200, cinderProjectQuotaLimits)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("wrong service/target for lookup: %s", r.URL.Path)
		testcloud.JSON(w, 404, "{}")
	})
	client := cloud.Client("block-storage", "/cinder")
	client.Microversion = "3.70"
	api := quotasets.New(client)
	scope, err := api.InProject(context.Background(), resource.Name("tenant"), quotasets.WithIdentityClient(cloud.Client("identity", "/identity/v3")))
	if err != nil || scope == nil || scope.ProjectID() != "project-fixed" || identityCalls.Load() != 2 || cinderCalls.Load() != 0 {
		t.Fatalf("scope=%v identity=%d cinder=%d err=%v", scope, identityCalls.Load(), cinderCalls.Load(), err)
	}
	for range 2 {
		if value, err := scope.Get(context.Background()); err != nil || value.ProjectID != "project-fixed" {
			t.Fatalf("value=%+v err=%v", value, err)
		}
	}
	if identityCalls.Load() != 2 || cinderCalls.Load() != 2 {
		t.Fatal("quota operation re-resolved its project")
	}
}

func TestCinderProjectQuotaProjectLookupFailuresPreventCinderRequests(t *testing.T) {
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
				t.Errorf("failed project lookup called Cinder: %s", r.URL.Path)
				testcloud.JSON(w, 404, "{}")
			})
			scope, err := quotasets.New(cloud.Client("block-storage", "/cinder")).InProject(context.Background(), resource.Name("tenant"), quotasets.WithIdentityClient(cloud.Client("identity", "/identity/v3")))
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

func setCinderQuotaProjectAuth(t *testing.T, provider *gophercloud.ProviderClient, id string) {
	t.Helper()
	var auth tokens3.CreateResult
	auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": id}}}
	auth.Header = http.Header{"X-Subject-Token": []string{"project-token"}}
	if err := provider.SetTokenAndAuthResult(auth); err != nil {
		t.Fatal(err)
	}
}

func TestCinderProjectQuotaCurrentProjectUsesRecordedV3AndV2AuthenticationAndFreezesID(t *testing.T) {
	for _, version := range []string{"v3", "v2"} {
		t.Run(version, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/cinder/os-quota-sets/project-fixed", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, cinderProjectQuotaLimits)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("current project guessed/refreshed a target: %s", r.URL.Path)
				testcloud.JSON(w, 404, "{}")
			})
			if version == "v3" {
				setCinderQuotaProjectAuth(t, cloud.Provider, "project-fixed")
			} else {
				var auth tokens2.CreateResult
				auth.Body = map[string]any{"access": map[string]any{"token": map[string]any{"id": "v2-token", "expires": "2026-10-01T00:00:00.000Z", "tenant": map[string]any{"id": "project-fixed"}}}}
				if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
					t.Fatal(err)
				}
			}
			scope, err := quotasets.New(cloud.Client("block-storage", "/cinder")).CurrentProject(context.Background())
			if err != nil || scope == nil || scope.ProjectID() != "project-fixed" || calls.Load() != 0 {
				t.Fatalf("scope=%v calls=%d err=%v", scope, calls.Load(), err)
			}
			setCinderQuotaProjectAuth(t, cloud.Provider, "changed-project")
			value, err := scope.Get(context.Background())
			if err != nil || value == nil || value.ProjectID != "project-fixed" || calls.Load() != 1 {
				t.Fatalf("quota=%+v calls=%d err=%v", value, calls.Load(), err)
			}
		})
	}
}

type cinderQuotaUnsupportedAuth struct{}

func (*cinderQuotaUnsupportedAuth) ExtractTokenID() (string, error) { return "custom-token", nil }

func TestCinderProjectQuotaCurrentProjectRejectsUnscopedManualAndInvalidAuthResults(t *testing.T) {
	for _, tc := range []struct {
		name string
		auth gophercloud.AuthResult
		body any
		id   string
		want error
	}{
		{name: "manual token", want: resource.ErrUnsupported},
		{name: "unsupported auth", auth: &cinderQuotaUnsupportedAuth{}, want: resource.ErrUnsupported},
		{name: "typed nil auth", auth: (*cinderQuotaUnsupportedAuth)(nil), want: resource.ErrUnsupported},
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
				setCinderQuotaProjectAuth(t, cloud.Provider, tc.id)
			} else if tc.body != nil {
				var auth tokens3.CreateResult
				auth.Body, auth.Header = tc.body, http.Header{"X-Subject-Token": []string{"private-token"}}
				if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
					t.Fatal(err)
				}
			}
			scope, err := quotasets.New(cloud.Client("block-storage", "/cinder/project-fixed")).CurrentProject(context.Background())
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

func TestCinderProjectQuotaUpdateDistinguishesOmittedFalseZeroAndUnlimited(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/cinder/os-quota-sets/project-fixed", func(w http.ResponseWriter, r *http.Request) {
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
			if len(envelope.Quota) != 3 || string(envelope.Quota["force"]) != "false" || string(envelope.Quota["gigabytes"]) != "-1" || string(envelope.Quota["snapshots"]) != "0" {
				t.Errorf("replacement/last option semantics: %s", envelope.Quota)
			}
		}
		testcloud.JSON(w, 200, cinderProjectQuotaLimits)
	})
	scope := newCinderProjectQuotaScope(t, cloud)
	if _, err := scope.Update(context.Background(), quotasets.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	if _, err := scope.Update(context.Background(), quotasets.UpdateOpts{Force: true}); err != nil {
		t.Fatal(err)
	}
	zero, unlimited, ignored := 0, -1, 17
	if _, err := scope.Update(context.Background(), quotasets.UpdateOpts{Volumes: &ignored}, quotasets.WithUpdateOptions(quotasets.UpdateOpts{Gigabytes: &unlimited, Snapshots: &zero}), quotasets.WithUpdateForce(true), quotasets.WithUpdateForce(false)); err != nil || calls.Load() != 3 {
		t.Fatalf("calls=%d err=%v", calls.Load(), err)
	}
}

func TestCinderProjectQuotaInvalidOptionsFailBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, cinderProjectQuotaLimits)
	})
	scope := newCinderProjectQuotaScope(t, cloud)
	negative := -2
	for _, tc := range []struct {
		name string
		opts quotasets.UpdateOpts
		with []quotasets.UpdateOption
	}{
		{name: "invalid limit", opts: quotasets.UpdateOpts{Volumes: &negative}},
		{name: "unserializable Extra", opts: quotasets.UpdateOpts{Extra: map[string]any{"vendor": make(chan bool)}}},
		{name: "unserializable quota snapshot", with: []quotasets.UpdateOption{quotasets.WithQuotaOptions(quotasets.UpdateOpts{Extra: map[string]any{"vendor": make(chan bool)}})}},
		{name: "empty Extra key", opts: quotasets.UpdateOpts{Extra: map[string]any{" ": 0}}},
		{name: "Extra field collision", opts: quotasets.UpdateOpts{Extra: map[string]any{"volumes_SSD": 1}}, with: []quotasets.UpdateOption{quotasets.WithUpdateField("volumes_SSD", 2)}},
		{name: "unknown volume type quota", with: []quotasets.UpdateOption{quotasets.WithVolumeTypeQuota("backups", "SSD", 1)}},
		{name: "empty volume type", with: []quotasets.UpdateOption{quotasets.WithVolumeTypeQuota(quotasets.VolumeTypeVolumes, " ", 1)}},
		{name: "invalid volume type limit", with: []quotasets.UpdateOption{quotasets.WithVolumeTypeQuota(quotasets.VolumeTypeSnapshots, "SSD", -2)}},
		{name: "nil update option", with: []quotasets.UpdateOption{nil}},
		{name: "core collision", with: []quotasets.UpdateOption{quotasets.WithUpdateField("snapshots", 0)}},
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
	api := quotasets.New(cloud.Client("block-storage", "/cinder"))
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

func TestCinderProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound(t *testing.T) {
	for _, status := range []int{404, 403} {
		for _, operation := range []string{"Get", "Defaults", "Usage", "Update", "Reset"} {
			t.Run(operation+http.StatusText(status), func(t *testing.T) {
				cloud := testcloud.New(t)
				body := `{"error":{"message":"quota denied"}}`
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Openstack-Request-Id", "req-denied")
					testcloud.JSON(w, status, body)
				})
				scope := newCinderProjectQuotaScope(t, cloud)
				var err error
				switch operation {
				case "Get":
					value, cause := scope.Get(context.Background())
					err = cause
					if value != nil {
						t.Fatal(value)
					}
				case "Defaults":
					value, cause := scope.Defaults(context.Background())
					err = cause
					if value != nil {
						t.Fatal(value)
					}
				case "Usage":
					value, cause := scope.Usage(context.Background())
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

func TestCinderProjectQuotaRejectsMalformedQuotaObjects(t *testing.T) {
	for _, body := range []string{`{}`, `{"quota_set":null}`, `{"quota_set":[]}`, `{"quota_set":"scalar"}`, `{"quota_set":{"volumes":"invalid"}}`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, body) })
			scope := newCinderProjectQuotaScope(t, cloud)
			if value, err := scope.Get(context.Background()); value != nil || err == nil {
				t.Fatalf("get=%v err=%v", value, err)
			}
			if value, err := scope.Defaults(context.Background()); value != nil || err == nil {
				t.Fatalf("defaults=%v err=%v", value, err)
			}
			if value, err := scope.Usage(context.Background()); value != nil || err == nil {
				t.Fatalf("usage=%v err=%v", value, err)
			}
			if value, err := scope.Update(context.Background(), quotasets.UpdateOpts{}); value != nil || err == nil {
				t.Fatalf("update=%v err=%v", value, err)
			}
		})
	}
}

func TestCinderProjectQuotaCancellationAndTimeoutPreserveCause(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	started := make(chan struct{}, 1)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		started <- struct{}{}
		<-r.Context().Done()
	})
	scope := newCinderProjectQuotaScope(t, cloud)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	api := quotasets.New(cloud.Client("block-storage", "/cinder"))
	if value, err := api.InProject(ctx, resource.ID("project-fixed")); value != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("scope=%v err=%v", value, err)
	}
	if value, err := api.CurrentProject(ctx); value != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("current=%v err=%v", value, err)
	}
	if value, err := scope.Get(ctx); value != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("get=%v err=%v", value, err)
	}
	if value, err := scope.Defaults(ctx); value != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("defaults=%v err=%v", value, err)
	}
	if value, err := scope.Usage(ctx); value != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("usage=%v err=%v", value, err)
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
	if value, err := scope.Usage(ctx); value != nil || !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 2 {
		t.Fatalf("usage=%v calls=%d err=%v", value, calls.Load(), err)
	}
}
