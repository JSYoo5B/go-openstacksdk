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

	"github.com/gophercloud/gophercloud/v2"
	tokens3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	"gophercloudsdk/containerinfra/v1/quotas"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

const magnumQuotaBody = `{"id":9007199254740993,"project_id":"wire-project","resource":"Cluster","hard_limit":0,"created_at":"2017-01-17T17:35:48+00:00","updated_at":null,"vendor":{"big":9007199254740993},"optional":null}`

func newMagnumQuotaScope(t *testing.T, cloud *testcloud.Cloud) *quotas.ResourceQuotaScope {
	t.Helper()
	project, err := quotas.New(cloud.Client("container-infrastructure-management", "/magnum/v1")).InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil {
		t.Fatal(err)
	}
	scope, err := project.ForResource(quotas.Cluster)
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func TestMagnumQuotaScopedCreatePreservesFixedIdentityExactJSONAndMetadata(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /magnum/v1/quotas", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body) != 4 || string(body["project_id"]) != `"project-fixed"` || string(body["resource"]) != `"Cluster"` || string(body["hard_limit"]) != "0" || string(body["vendor"]) != `{"tier":"requested"}` {
			t.Errorf("body=%s err=%v", body, err)
		}
		if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Configured") != "original" {
			t.Error(r.URL, r.Header)
		}
		w.Header().Set("X-Openstack-Request-Id", "quota-created")
		testcloud.JSON(w, 201, magnumQuotaBody)
	})
	client := cloud.Client("container-infrastructure-management", "/catalog")
	client.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + "/magnum/v1")
	client.MoreHeaders = map[string]string{"X-Configured": "original"}
	project, err := quotas.New(client).InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil || project.ProjectID() != "project-fixed" || calls.Load() != 0 {
		t.Fatal(project, err)
	}
	scope, err := project.ForResource(quotas.Cluster)
	if err != nil || scope.ProjectID() != "project-fixed" || scope.ResourceName() != quotas.Cluster || calls.Load() != 0 {
		t.Fatal(scope, err)
	}
	extension := map[string]string{"tier": "requested"}
	option := quotas.WithQuotaCreateField("vendor", extension)
	extension["tier"] = "caller-change"
	value, err := scope.Create(context.Background(), quotas.WithHardLimit(0), option)
	if err != nil || value == nil || value.RequestProjectID != "project-fixed" || value.RequestResource != quotas.Cluster || value.ProjectID != "wire-project" || value.ID != "9007199254740993" || value.HardLimit != 0 || value.Resource != "Cluster" || value.CreatedAt.IsZero() || !value.UpdatedAt.IsZero() || value.StatusCode != 201 || value.Header.Get("X-Openstack-Request-Id") != "quota-created" || string(value.Body["optional"]) != "null" || string(value.Body["vendor"]) != `{"big":9007199254740993}` || calls.Load() != 1 {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if _, exists := value.Body["future"]; exists {
		t.Fatal("omitted field invented")
	}
	value.Header.Set("X-Configured", "changed")
	if client.MoreHeaders["X-Configured"] != "original" || client.ResourceBase != gophercloud.NormalizeURL(cloud.Server.URL+"/magnum/v1") || client.ProviderClient != cloud.Provider {
		t.Fatal("scope changed the selected client")
	}
}

func TestMagnumQuotaNilScopesRemainSafeToInspectAndRejectCreate(t *testing.T) {
	var project *quotas.ProjectQuotaScope
	var scope *quotas.ResourceQuotaScope
	if project.ProjectID() != "" || scope.ProjectID() != "" || scope.ResourceName() != "" {
		t.Fatal("nil scope fabricated an identity")
	}
	if _, err := project.ForResource(quotas.Cluster); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := scope.Create(context.Background(), quotas.WithHardLimit(1)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestMagnumQuotaCreateOptionsSnapshotAndRetainIntegersAcrossReauth(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, reauths atomic.Int32
	limit := int(^uint(0) >> 1)
	option := quotas.WithQuotaCreateOptions(quotas.QuotaCreateOpts{HardLimit: &limit})
	limit = 88
	cloud.Provider.ReauthFunc = func(context.Context) error {
		reauths.Add(1)
		cloud.Provider.SetToken("fresh-token")
		return nil
	}
	cloud.Mux.HandleFunc("POST /magnum/v1/quotas", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || string(body["hard_limit"]) != strconv.Itoa(int(^uint(0)>>1)) {
			t.Errorf("exact integer snapshot=%s err=%v", body, err)
		}
		if calls.Add(1) == 1 {
			testcloud.JSON(w, 401, `{"error":"expired"}`)
			return
		}
		if r.Header.Get("X-Auth-Token") != "fresh-token" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 201, magnumQuotaBody)
	})
	var applied *int
	capture := func(config *request.Config[quotas.QuotaCreateOpts]) error {
		applied = config.Options.HardLimit
		return nil
	}
	scope := newMagnumQuotaScope(t, cloud)
	if _, err := scope.Create(context.Background(), option, capture); err != nil {
		t.Fatal(err)
	}
	*applied = 77
	if _, err := scope.Create(context.Background(), option); err != nil || calls.Load() != 3 || reauths.Load() != 1 || limit != 88 {
		t.Fatal(err, calls.Load(), reauths.Load(), limit)
	}
}

func TestMagnumQuotaScopeUsesSeparateIdentityAndRecordedAuth(t *testing.T) {
	cloud := testcloud.New(t)
	var lookups, creates atomic.Int32
	cloud.Mux.HandleFunc("GET /identity/v3/projects", func(w http.ResponseWriter, r *http.Request) {
		lookups.Add(1)
		if r.URL.Query().Get("name") != "tenant" || r.Header.Get("X-Magnum-Only") != "" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"projects":[{"id":"project-fixed","name":"tenant"}]}`)
	})
	cloud.Mux.HandleFunc("POST /magnum/v1/quotas", func(w http.ResponseWriter, r *http.Request) {
		creates.Add(1)
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || string(body["project_id"]) != `"project-fixed"` {
			t.Error(body, err)
		}
		testcloud.JSON(w, 201, magnumQuotaBody)
	})
	client := cloud.Client("container-infrastructure-management", "/magnum/v1")
	client.MoreHeaders = map[string]string{"X-Magnum-Only": "original"}
	api := quotas.New(client)
	project, err := api.InProject(context.Background(), resource.Name("tenant"), quotas.WithIdentityClient(cloud.Client("identity", "/identity/v3")))
	if err != nil || project.ProjectID() != "project-fixed" || lookups.Load() != 1 || creates.Load() != 0 {
		t.Fatal(project, err)
	}
	scope, err := project.ForResource(quotas.Cluster)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := scope.Create(context.Background(), quotas.WithHardLimit(0)); err != nil {
			t.Fatal(err)
		}
	}
	auth := tokens3.CreateResult{}
	auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": "project-fixed"}}}
	auth.Header = http.Header{"X-Subject-Token": []string{"project-token"}}
	if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
		t.Fatal(err)
	}
	current, err := api.CurrentProject(context.Background())
	if err != nil || current.ProjectID() != "project-fixed" {
		t.Fatal(current, err)
	}
	auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": "changed"}}}
	if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
		t.Fatal(err)
	}
	if current.ProjectID() != "project-fixed" || lookups.Load() != 1 || creates.Load() != 2 {
		t.Fatal("recorded project was not frozen")
	}
}

func TestMagnumQuotaCreatePreflightAndCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 201, magnumQuotaBody) })
	scope := newMagnumQuotaScope(t, cloud)
	for _, options := range [][]quotas.QuotaCreateOption{
		nil,
		{nil},
		{quotas.WithQuotaCreateOptions(quotas.QuotaCreateOpts{})},
		{quotas.WithHardLimit(1), quotas.WithQuotaCreateField("hard_limit", 2)},
		{quotas.WithHardLimit(1), quotas.WithQuotaCreateField("project_id", "changed")},
		{quotas.WithHardLimit(1), quotas.WithQuotaCreateField("resource", "changed")},
		{quotas.WithHardLimit(1), quotas.WithQuotaCreateField("id", 2)},
		{quotas.WithHardLimit(1), quotas.WithQuotaCreateField("vendor", make(chan bool))},
		{quotas.WithHardLimit(1), request.WithQuery[quotas.QuotaCreateOpts]("all_tenants", "true")},
		{quotas.WithHardLimit(1), request.WithHeader[quotas.QuotaCreateOpts]("X-Vendor", "value")},
		{quotas.WithHardLimit(1), request.WithArgument[quotas.QuotaCreateOpts]("unsupported", true)},
	} {
		if value, err := scope.Create(context.Background(), options...); value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal(value, err, calls.Load())
		}
	}
	api := quotas.New(cloud.Client("container-infrastructure-management", "/magnum/v1"))
	if _, err := api.CurrentProject(context.Background()); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	for _, id := range []string{"", "bad/path", "null\x00id"} {
		if _, err := api.InProject(context.Background(), resource.ID(id)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(id, err)
		}
	}
	project, err := api.InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []quotas.ResourceName{"", "bad/path", "bad\x00name"} {
		if _, err := project.ForResource(name); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(name, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := scope.Create(ctx, quotas.WithHardLimit(0)); !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatal(err)
	}
}

func TestMagnumQuotaCreateHTTPFailuresAndSuccessPolicy(t *testing.T) {
	for _, status := range []int{200, 201, 202, 204, 403, 404, 409, 500} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Openstack-Request-Id", "quota-policy")
				testcloud.JSON(w, status, magnumQuotaBody)
			})
			value, err := newMagnumQuotaScope(t, cloud).Create(context.Background(), quotas.WithHardLimit(0))
			if status == 201 {
				if err != nil || value.StatusCode != status {
					t.Fatal(value, err)
				}
				return
			}
			var response gophercloud.ErrUnexpectedResponseCode
			if value != nil || !errors.As(err, &response) || response.Actual != status || response.ResponseHeader.Get("X-Openstack-Request-Id") != "quota-policy" || errors.Is(err, resource.ErrNotFound) != (status == 404) {
				t.Fatal(value, response, err)
			}
		})
	}
}

func TestMagnumQuotaCreateDecodeFailuresRetainSuccessfulEvidence(t *testing.T) {
	for _, body := range []string{`{}`, `null`, `[]`, `{bad`, `{"hard_limit":null}`, `{"hard_limit":"1"}`, `{"hard_limit":1,"id":1.5}`, `{"hard_limit":1,"id":{}}`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Openstack-Request-Id", "quota-malformed")
				testcloud.JSON(w, 201, body)
			})
			value, err := newMagnumQuotaScope(t, cloud).Create(context.Background(), quotas.WithHardLimit(0))
			var decode *quotas.QuotaResponseError
			if value != nil || !errors.As(err, &decode) || decode.StatusCode != 201 || string(decode.Body) != body || decode.Header.Get("X-Openstack-Request-Id") != "quota-malformed" {
				t.Fatal(value, decode, err)
			}
		})
	}
}

func TestMagnumQuotaCreateRejectsScopeChangingRedirectAndPreservesCanceledCause(t *testing.T) {
	cloud := testcloud.New(t)
	var escaped atomic.Int32
	cloud.Mux.HandleFunc("POST /magnum/v1/quotas", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/elsewhere", http.StatusTemporaryRedirect)
	})
	cloud.Mux.HandleFunc("/elsewhere", func(w http.ResponseWriter, r *http.Request) { escaped.Add(1); testcloud.JSON(w, 201, magnumQuotaBody) })
	if _, err := newMagnumQuotaScope(t, cloud).Create(context.Background(), quotas.WithHardLimit(0)); !errors.Is(err, resource.ErrInvalidOption) || escaped.Load() != 0 {
		t.Fatal(err, escaped.Load())
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
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := newMagnumQuotaScope(t, blocked).Create(ctx, quotas.WithHardLimit(0)); done <- err }()
	select {
	case <-started:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("quota create did not reach server")
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestMagnumQuotaGeneratedCreateRetainsNativeContract(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /magnum/v1/quotas", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body) != 3 || string(body["project_id"]) != `"native-project"` || string(body["resource"]) != `"Cluster"` || string(body["hard_limit"]) != "0" {
			t.Error(body, err)
		}
		testcloud.JSON(w, 201, `{"id":26,"project_id":"native-project","resource":"Cluster","hard_limit":0}`)
	})
	value, err := quotas.New(cloud.Client("container-infrastructure-management", "/magnum/v1")).Create(context.Background(), quotas.CreateOpts{ProjectID: "native-project", Resource: "Cluster", HardLimit: 0})
	if err != nil || value == nil || value.ID != "26" || value.ProjectID != "native-project" {
		t.Fatal(fmt.Sprintf("%+v", value), err)
	}
}
