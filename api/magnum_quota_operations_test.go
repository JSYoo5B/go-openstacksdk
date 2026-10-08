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

	"github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/quotas"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestMagnumQuotaScopedOperationsKeepProjectResourceBodyAndMetadata(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, updates, deletes atomic.Int32
	cloud.Mux.HandleFunc("/magnum/v1/quotas/project-fixed/Cluster", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(r.URL, r.Header)
		}
		switch r.Method {
		case http.MethodGet:
			gets.Add(1)
			if r.ContentLength > 0 {
				t.Error("GET has a request body")
			}
			w.Header().Set("X-Openstack-Request-Id", "quota-get")
			testcloud.JSON(w, 200, magnumQuotaBody)
		case http.MethodPatch:
			updates.Add(1)
			var body map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body) != 4 || string(body["project_id"]) != `"project-fixed"` || string(body["resource"]) != `"Cluster"` || string(body["hard_limit"]) != "0" || string(body["vendor"]) != `{"tier":"requested"}` {
				t.Errorf("PATCH flat body=%s err=%v", body, err)
			}
			w.Header().Set("X-Openstack-Request-Id", "quota-update")
			testcloud.JSON(w, 202, magnumQuotaBody)
		case http.MethodDelete:
			deletes.Add(1)
			if r.ContentLength > 0 || r.Header.Get("Content-Type") != "" {
				t.Error("DELETE has request body/content type")
			}
			w.Header().Set("X-Openstack-Request-Id", "quota-delete")
			w.WriteHeader(204)
		default:
			t.Error("unexpected method", r.Method)
		}
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("escaped compound identity", r.URL)
		w.WriteHeader(404)
	})
	scope := newMagnumQuotaScope(t, cloud)
	value, err := scope.Get(context.Background())
	if err != nil || value == nil || value.RequestProjectID != "project-fixed" || value.RequestResource != quotas.Cluster || value.ProjectID != "wire-project" || value.ID != "9007199254740993" || value.StatusCode != 200 || value.Header.Get("X-Openstack-Request-Id") != "quota-get" || string(value.Body["vendor"]) != `{"big":9007199254740993}` {
		t.Fatal(value, err)
	}
	extension := map[string]string{"tier": "requested"}
	field := quotas.WithQuotaUpdateField("vendor", extension)
	extension["tier"] = "caller-change"
	updated, err := scope.Update(context.Background(), quotas.WithHardLimit(0), field)
	if err != nil || updated == nil || updated.RequestProjectID != "project-fixed" || updated.RequestResource != quotas.Cluster || updated.StatusCode != 202 || updated.Header.Get("X-Openstack-Request-Id") != "quota-update" || updated.ID != "9007199254740993" || string(updated.Body["optional"]) != "null" {
		t.Fatal(updated, err)
	}
	deleted, err := scope.Delete(context.Background())
	if err != nil || deleted == nil || deleted.ProjectID != "project-fixed" || deleted.ResourceName != quotas.Cluster || deleted.StatusCode != 204 || deleted.Header.Get("X-Openstack-Request-Id") != "quota-delete" || gets.Load() != 1 || updates.Load() != 1 || deletes.Load() != 1 {
		t.Fatal(deleted, err, gets.Load(), updates.Load(), deletes.Load())
	}
	value.Header.Set("X-Openstack-Request-Id", "caller-change")
	if updated.Header.Get("X-Openstack-Request-Id") != "quota-update" || deleted.Header.Get("X-Openstack-Request-Id") != "quota-delete" {
		t.Fatal("response headers share mutable state")
	}
}

func TestMagnumQuotaGetDeploymentFallbackPreservesOmittedIdentity(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /magnum/v1/quotas/project-fixed/Cluster", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"project_id":"project-fixed","hard_limit":7}`)
	})
	value, err := newMagnumQuotaScope(t, cloud).Get(context.Background())
	if err != nil || value == nil || value.HardLimit != 7 || value.ID != "" || value.Resource != "" || value.RequestResource != quotas.Cluster || value.RequestProjectID != "project-fixed" {
		t.Fatal(value, err)
	}
	for _, key := range []string{"id", "resource"} {
		if _, exists := value.Body[key]; exists {
			t.Fatal("fallback identity invented", key)
		}
	}
}

func TestMagnumQuotaUpdateSnapshotRemainsExactAcrossReauthAndReuse(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, reauths atomic.Int32
	limit := int(^uint(0) >> 1)
	option := quotas.WithQuotaUpdateOptions(quotas.QuotaUpdateOpts{HardLimit: &limit})
	limit = 88
	cloud.Provider.ReauthFunc = func(context.Context) error { reauths.Add(1); cloud.Provider.SetToken("fresh-token"); return nil }
	cloud.Mux.HandleFunc("PATCH /magnum/v1/quotas/project-fixed/Cluster", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || string(body["hard_limit"]) != strconv.Itoa(int(^uint(0)>>1)) || string(body["project_id"]) != `"project-fixed"` || string(body["resource"]) != `"Cluster"` {
			t.Error(body, err)
		}
		if calls.Add(1) == 1 {
			testcloud.JSON(w, 401, `{"error":"expired"}`)
			return
		}
		if r.Header.Get("X-Auth-Token") != "fresh-token" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 202, magnumQuotaBody)
	})
	scope := newMagnumQuotaScope(t, cloud)
	var applied *int
	capture := func(config *request.Config[quotas.QuotaUpdateOpts]) error {
		applied = config.Options.HardLimit
		return nil
	}
	if _, err := scope.Update(context.Background(), option, capture); err != nil {
		t.Fatal(err)
	}
	*applied = 77
	if _, err := scope.Update(context.Background(), option); err != nil || calls.Load() != 3 || reauths.Load() != 1 || limit != 88 {
		t.Fatal(err, calls.Load(), reauths.Load(), limit)
	}
}

func TestMagnumQuotaOperationsPreflightRejectsInvalidOptionsAndNilScopes(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 202, magnumQuotaBody) })
	scope := newMagnumQuotaScope(t, cloud)
	for _, options := range [][]quotas.QuotaUpdateOption{
		nil, {nil}, {quotas.WithQuotaUpdateOptions(quotas.QuotaUpdateOpts{})},
		{quotas.WithHardLimit(0), quotas.WithQuotaUpdateField("hard_limit", 1)},
		{quotas.WithHardLimit(0), quotas.WithQuotaUpdateField("project_id", "changed")},
		{quotas.WithHardLimit(0), quotas.WithQuotaUpdateField("resource", "changed")},
		{quotas.WithHardLimit(0), quotas.WithQuotaUpdateField("id", 1)},
		{quotas.WithHardLimit(0), quotas.WithQuotaUpdateField("vendor", make(chan bool))},
		{quotas.WithHardLimit(0), request.WithQuery[quotas.QuotaUpdateOpts]("all_tenants", "true")},
		{quotas.WithHardLimit(0), request.WithHeader[quotas.QuotaUpdateOpts]("X-Vendor", "value")},
		{quotas.WithHardLimit(0), request.WithArgument[quotas.QuotaUpdateOpts]("unsupported", true)},
	} {
		if value, err := scope.Update(context.Background(), options...); value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal(value, err, calls.Load())
		}
	}
	if _, err := scope.Delete(context.Background(), nil); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
		t.Fatal(err)
	}
	for _, nilScope := range []*quotas.ResourceQuotaScope{nil, {}} {
		for _, operation := range []string{"Get", "Update", "Delete"} {
			if err := magnumQuotaOperation(nilScope, operation, context.Background()); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(operation, err)
			}
		}
	}
}

func magnumQuotaOperation(scope *quotas.ResourceQuotaScope, operation string, ctx context.Context) error {
	switch operation {
	case "Get":
		_, err := scope.Get(ctx)
		return err
	case "Update":
		_, err := scope.Update(ctx, quotas.WithHardLimit(0))
		return err
	default:
		_, err := scope.Delete(ctx)
		return err
	}
}

func TestMagnumQuotaOperationsHTTPPolicyAndStrictMissing(t *testing.T) {
	for _, operation := range []string{"Get", "Update", "Delete"} {
		for _, status := range []int{200, 201, 202, 204, 403, 404, 500} {
			t.Run(fmt.Sprintf("%s-%d", operation, status), func(t *testing.T) {
				cloud := testcloud.New(t)
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Openstack-Request-Id", "quota-policy")
					if operation == "Delete" {
						w.WriteHeader(status)
						return
					}
					testcloud.JSON(w, status, magnumQuotaBody)
				})
				scope := newMagnumQuotaScope(t, cloud)
				err := magnumQuotaOperation(scope, operation, context.Background())
				accepted := operation == "Get" && status == 200 || operation == "Update" && status == 202 || operation == "Delete" && status == 204
				if accepted {
					if err != nil {
						t.Fatal(err)
					}
					return
				}
				var response gophercloud.ErrUnexpectedResponseCode
				var sdk *resource.OperationError
				if !errors.As(err, &response) || response.Actual != status || response.ResponseHeader.Get("X-Openstack-Request-Id") != "quota-policy" || !errors.As(err, &sdk) || sdk.Operation != operation || errors.Is(err, resource.ErrNotFound) != (status == 404) {
					t.Fatal(response, sdk, err)
				}
				if operation == "Delete" {
					value, err := scope.Delete(context.Background(), quotas.WithDeleteIgnoreMissing(true))
					if value != nil || (err == nil) != (status == 404) {
						t.Fatal(value, err)
					}
					_, err = scope.Delete(context.Background(), quotas.WithDeleteIgnoreMissing(true), quotas.WithDeleteIgnoreMissing(false))
					if !gophercloud.ResponseCodeIs(err, status) {
						t.Fatal(err)
					}
				}
			})
		}
	}
}

func TestMagnumQuotaGetAndUpdateDecodeFailuresKeepSuccessfulEvidence(t *testing.T) {
	for _, operation := range []string{"Get", "Update"} {
		for _, body := range []string{`{}`, `null`, `[]`, `{"hard_limit":null}`, `{"hard_limit":"1"}`, `{"hard_limit":1,"id":1.5}`} {
			t.Run(operation+body, func(t *testing.T) {
				cloud := testcloud.New(t)
				status := 200
				if operation == "Update" {
					status = 202
				}
				var calls atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("X-Openstack-Request-Id", "quota-malformed")
					testcloud.JSON(w, status, body)
				})
				err := magnumQuotaOperation(newMagnumQuotaScope(t, cloud), operation, context.Background())
				var decode *quotas.QuotaResponseError
				if !errors.As(err, &decode) || decode.StatusCode != status || string(decode.Body) != body || decode.Header.Get("X-Openstack-Request-Id") != "quota-malformed" || calls.Load() != 1 {
					t.Fatal(decode, err, calls.Load())
				}
			})
		}
	}
}

func TestMagnumQuotaOperationsPreventScopeChangingRedirects(t *testing.T) {
	for _, operation := range []string{"Get", "Update", "Delete"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			var escaped atomic.Int32
			cloud.Mux.HandleFunc("/magnum/v1/quotas/project-fixed/Cluster", func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/magnum/v1/quotas/other/Cluster", http.StatusTemporaryRedirect)
			})
			cloud.Mux.HandleFunc("/magnum/v1/quotas/other/Cluster", func(w http.ResponseWriter, r *http.Request) { escaped.Add(1); w.WriteHeader(204) })
			if err := magnumQuotaOperation(newMagnumQuotaScope(t, cloud), operation, context.Background()); !errors.Is(err, resource.ErrInvalidOption) || escaped.Load() != 0 {
				t.Fatal(err, escaped.Load())
			}
		})
	}
}

func TestMagnumQuotaOperationsCancellationPreservesCause(t *testing.T) {
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
	scope := newMagnumQuotaScope(t, cloud)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, operation := range []string{"Get", "Update", "Delete"} {
		if err := magnumQuotaOperation(scope, operation, ctx); !errors.Is(err, context.Canceled) || calls.Load() != 0 {
			t.Fatal(operation, err, calls.Load())
		}
	}
	ctx, cancel = context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- magnumQuotaOperation(scope, "Update", ctx) }()
	select {
	case <-started:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("PATCH did not reach server")
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := magnumQuotaOperation(scope, "Get", ctx); !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 2 {
		t.Fatal(err, calls.Load())
	}
}
