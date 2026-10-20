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

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const neutronQuotaLimits = `{"quota":{"project_id":"wire-project","network":-1,"port":0,"vendor":{"big":9007199254740993},"optional":null}}`

func newNeutronQuotaScope(t *testing.T, cloud *testcloud.Cloud) *quotas.ProjectQuotaScope {
	t.Helper()
	scope, err := quotas.New(cloud.Client("network", "/neutron/v2.0")).InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func TestNeutronProjectQuotaOperationsKeepTargetRawJSONAndMetadata(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, updates, deletes atomic.Int32
	cloud.Mux.HandleFunc("/neutron/v2.0/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" || r.Header.Get("X-Configured") != "original" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Errorf("request URL/header=%s/%v", r.URL, r.Header)
		}
		switch r.Method {
		case http.MethodGet:
			gets.Add(1)
			w.Header().Set("X-Openstack-Request-Id", "get-quota")
			testcloud.JSON(w, 200, neutronQuotaLimits)
		case http.MethodPut:
			updates.Add(1)
			var body struct {
				Quota map[string]json.RawMessage `json:"quota"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Quota) != 4 || string(body.Quota["network"]) != "-1" || string(body.Quota["port"]) != "0" || string(body.Quota["check_limit"]) != "false" || string(body.Quota["vendor"]) != `{"tier":"requested"}` {
				t.Errorf("PUT quota=%s error=%v", body.Quota, err)
			}
			w.Header().Set("X-Openstack-Request-Id", "update-quota")
			testcloud.JSON(w, 200, neutronQuotaLimits)
		case http.MethodDelete:
			deletes.Add(1)
			if r.ContentLength > 0 {
				t.Error("DELETE has an unexpected body")
			}
			w.Header().Set("X-Openstack-Request-Id", "delete-quota")
			w.WriteHeader(204)
		default:
			t.Errorf("unsupported quota operation %s", r.Method)
		}
	})
	cloud.Mux.HandleFunc("/neutron/v2.0/quotas/project-fixed/details.json", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.RawQuery != "" {
			t.Errorf("detail URL/method=%s/%s", r.URL, r.Method)
		}
		w.Header().Set("X-Openstack-Request-Id", "detail-quota")
		testcloud.JSON(w, 200, `{"quota":{"network":{"limit":-1,"used":2,"reserved":"3","future":9007199254740993},"port":{"limit":0,"used":0,"reserved":0},"project_id":"wire-project","vendor":{"limit":1,"used":0,"reserved":0,"null":null}}}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("request escaped the selected resource base/target: %s", r.URL.Path)
		testcloud.JSON(w, 404, "{}")
	})
	client := cloud.Client("network", "/catalog")
	client.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + "/neutron/v2.0")
	client.MoreHeaders = map[string]string{"X-Configured": "original"}
	scope, err := quotas.New(client).InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil || scope.ProjectID() != "project-fixed" || gets.Load() != 0 {
		t.Fatalf("scope=%v error=%v", scope, err)
	}
	value, err := scope.Get(context.Background())
	if err != nil || value == nil || value.ProjectID != "project-fixed" || value.Network != -1 || value.Port != 0 || value.StatusCode != 200 || string(value.Body["project_id"]) != `"wire-project"` || string(value.Body["vendor"]) != `{"big":9007199254740993}` || string(value.Body["optional"]) != "null" || value.Header.Get("X-Openstack-Request-Id") != "get-quota" {
		t.Fatalf("GET value=%+v error=%v", value, err)
	}
	if _, exists := value.Body["subnet"]; exists {
		t.Fatal("omitted quota field was invented")
	}
	detail, err := scope.Detail(context.Background())
	if err != nil || detail == nil || detail.ProjectID != "project-fixed" || detail.Network.Limit != -1 || detail.Network.Used != 2 || detail.Network.Reserved != 3 || detail.Port.Limit != 0 || detail.StatusCode != 200 || string(detail.Body["network"]) != `{"limit":-1,"used":2,"reserved":"3","future":9007199254740993}` || detail.Header.Get("X-Openstack-Request-Id") != "detail-quota" {
		t.Fatalf("detail=%+v error=%v", detail, err)
	}
	zero, unlimited := 0, -1
	extension := map[string]string{"tier": "requested"}
	field := quotas.WithUpdateField("vendor", extension)
	extension["tier"] = "caller-change"
	updated, err := scope.Update(context.Background(), quotas.UpdateOpts{Network: &unlimited, Port: &zero}, field, quotas.WithUpdateCheckLimit(false))
	if err != nil || updated == nil || updated.ProjectID != "project-fixed" || updated.Header.Get("X-Openstack-Request-Id") != "update-quota" || updated.StatusCode != 200 || string(updated.Body["vendor"]) != `{"big":9007199254740993}` {
		t.Fatalf("updated=%+v error=%v", updated, err)
	}
	deleted, err := scope.Delete(context.Background())
	if err != nil || deleted == nil || deleted.ProjectID != "project-fixed" || deleted.StatusCode != 204 || deleted.Header.Get("X-Openstack-Request-Id") != "delete-quota" || gets.Load() != 1 || updates.Load() != 1 || deletes.Load() != 1 {
		t.Fatalf("DELETE=%+v counts=%d/%d/%d error=%v", deleted, gets.Load(), updates.Load(), deletes.Load(), err)
	}
	value.Header.Set("X-Openstack-Request-Id", "caller-change")
	if updated.Header.Get("X-Openstack-Request-Id") != "update-quota" || detail.Header.Get("X-Openstack-Request-Id") != "detail-quota" || deleted.Header.Get("X-Openstack-Request-Id") != "delete-quota" || client.MoreHeaders["X-Configured"] != "original" || client.ResourceBase != gophercloud.NormalizeURL(cloud.Server.URL+"/neutron/v2.0") || client.Microversion != "" {
		t.Fatal("scope changed selected client configuration or shared mutable response headers")
	}
}

func TestNeutronProjectQuotaUpdateOmittedFalseZeroAndSnapshotAcrossReauth(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, reauths atomic.Int32
	port, ignored := 0, 17
	limit := int(^uint(0) >> 1)
	cloud.Provider.ReauthFunc = func(context.Context) error {
		port, limit = 999, 888
		reauths.Add(1)
		cloud.Provider.SetToken("fresh-token")
		return nil
	}
	cloud.Mux.HandleFunc("/neutron/v2.0/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Quota map[string]json.RawMessage `json:"quota"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || r.Method != http.MethodPut {
			t.Fatalf("PUT=%s body=%s error=%v", r.Method, body.Quota, err)
		}
		switch calls.Add(1) {
		case 1:
			if len(body.Quota) != 0 {
				t.Errorf("nil limits/check flag must be omitted: %s", body.Quota)
			}
		case 2:
			if len(body.Quota) != 1 || string(body.Quota["check_limit"]) != "true" {
				t.Errorf("explicit true=%s", body.Quota)
			}
		case 3, 4:
			if len(body.Quota) != 3 || string(body.Quota["check_limit"]) != "false" || string(body.Quota["port"]) != "0" || string(body.Quota["network"]) != strconv.Itoa(int(^uint(0)>>1)) {
				t.Errorf("snapshot/exact integer/last option=%s", body.Quota)
			}
			if calls.Load() == 3 {
				testcloud.JSON(w, 401, `{"error":"expired"}`)
				return
			}
			if r.Header.Get("X-Auth-Token") != "fresh-token" {
				t.Error(r.Header)
			}
		}
		testcloud.JSON(w, 200, neutronQuotaLimits)
	})
	scope := newNeutronQuotaScope(t, cloud)
	if _, err := scope.Update(context.Background(), quotas.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	if _, err := scope.Update(context.Background(), quotas.UpdateOpts{}, quotas.WithUpdateCheckLimit(true)); err != nil {
		t.Fatal(err)
	}
	if _, err := scope.Update(context.Background(), quotas.UpdateOpts{Port: &ignored}, quotas.WithUpdateOptions(quotas.UpdateOpts{Port: &port, Network: &limit}), quotas.WithUpdateCheckLimit(true), quotas.WithUpdateCheckLimit(false)); err != nil || calls.Load() != 4 || reauths.Load() != 1 || port != 999 || limit != 888 {
		t.Fatalf("calls/reauths=%d/%d pointers=%d/%d error=%v", calls.Load(), reauths.Load(), port, limit, err)
	}
}

func TestNeutronProjectQuotaOptionsSnapshotAtCreationAndRemainReusable(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/neutron/v2.0/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body struct {
			Quota map[string]json.RawMessage `json:"quota"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || r.Method != http.MethodPut || len(body.Quota) != 2 || string(body.Quota["port"]) != "0" || string(body.Quota["network"]) != "-1" {
			t.Errorf("typed option snapshot=%s error=%v", body.Quota, err)
		}
		testcloud.JSON(w, 200, neutronQuotaLimits)
	})
	port, network := 0, -1
	option := quotas.WithQuotaOptions(quotas.UpdateOpts{Port: &port, Network: &network})
	port, network = 99, 88
	var appliedPort *int
	capture := func(config *request.Config[quotas.UpdateOpts]) error {
		appliedPort = config.Options.Port
		return nil
	}
	scope := newNeutronQuotaScope(t, cloud)
	if _, err := scope.Update(context.Background(), quotas.UpdateOpts{}, option, capture); err != nil {
		t.Fatal(err)
	}
	// Mutating one application's private pointer must not change the stored
	// snapshot that is used when the same option is applied again.
	*appliedPort = 77
	if _, err := scope.Update(context.Background(), quotas.UpdateOpts{}, option); err != nil || calls.Load() != 2 || port != 99 || network != 88 {
		t.Fatalf("calls=%d caller=%d/%d error=%v", calls.Load(), port, network, err)
	}
}

func TestNeutronProjectQuotaOptionsFailBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, neutronQuotaLimits) })
	scope := newNeutronQuotaScope(t, cloud)
	bad := -2
	for _, tc := range []struct {
		name string
		opts quotas.UpdateOpts
		with []quotas.UpdateOption
	}{
		{"invalid limit", quotas.UpdateOpts{FloatingIP: &bad}, nil},
		{"nil option", quotas.UpdateOpts{}, []quotas.UpdateOption{nil}},
		{"core collision", quotas.UpdateOpts{}, []quotas.UpdateOption{quotas.WithUpdateField("port", 0)}},
		{"check limit collision", quotas.UpdateOpts{}, []quotas.UpdateOption{quotas.WithUpdateField("check_limit", false)}},
		{"unserializable field", quotas.UpdateOpts{}, []quotas.UpdateOption{quotas.WithUpdateField("vendor", make(chan bool))}},
		{"query", quotas.UpdateOpts{}, []quotas.UpdateOption{request.WithQuery[quotas.UpdateOpts]("fields", "port")}},
		{"header", quotas.UpdateOpts{}, []quotas.UpdateOption{request.WithHeader[quotas.UpdateOpts]("X-Vendor", "value")}},
		{"unknown argument", quotas.UpdateOpts{}, []quotas.UpdateOption{request.WithArgument[quotas.UpdateOpts]("unknown", true)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if value, err := scope.Update(context.Background(), tc.opts, tc.with...); value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatalf("value=%+v calls=%d error=%v", value, calls.Load(), err)
			}
		})
	}
	if value, err := scope.Delete(context.Background(), nil); value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
		t.Fatalf("DELETE=%+v calls=%d error=%v", value, calls.Load(), err)
	}
	if value, err := scope.Detail(context.Background()); value != nil || err == nil || calls.Load() != 1 {
		t.Fatal("detail was incorrectly decoded as ordinary integer limits")
	}
}

func TestNeutronProjectQuotaHTTPFailuresAndStrictMissingPolicy(t *testing.T) {
	for _, status := range []int{404, 403, 500} {
		for _, operation := range []string{"Get", "Detail", "Update", "Delete"} {
			t.Run(fmt.Sprintf("%s-%d", operation, status), func(t *testing.T) {
				cloud := testcloud.New(t)
				const body = `{"error":{"message":"quota denied"}}`
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Openstack-Request-Id", "quota-denied")
					testcloud.JSON(w, status, body)
				})
				scope := newNeutronQuotaScope(t, cloud)
				err := neutronQuotaOperation(scope, operation, context.Background())
				var response gophercloud.ErrUnexpectedResponseCode
				var sdk *resource.OperationError
				if !errors.As(err, &response) || response.Actual != status || string(response.Body) != body || response.ResponseHeader.Get("X-Openstack-Request-Id") != "quota-denied" || !errors.As(err, &sdk) || sdk.Operation != operation || errors.Is(err, resource.ErrNotFound) != (status == 404) {
					t.Fatalf("response=%+v sdk=%+v error=%v", response, sdk, err)
				}
				if operation == "Delete" {
					value, err := scope.Delete(context.Background(), quotas.WithDeleteIgnoreMissing(true))
					if value != nil || (err == nil) != (status == 404) {
						t.Fatalf("ignored DELETE=%+v error=%v", value, err)
					}
					_, err = scope.Delete(context.Background(), quotas.WithDeleteIgnoreMissing(true), quotas.WithDeleteIgnoreMissing(false))
					if !gophercloud.ResponseCodeIs(err, status) {
						t.Fatalf("last false option did not restore strict errors: %v", err)
					}
				}
			})
		}
	}
}

func neutronQuotaOperation(scope *quotas.ProjectQuotaScope, operation string, ctx context.Context) error {
	switch operation {
	case "Get":
		_, err := scope.Get(ctx)
		return err
	case "Detail":
		_, err := scope.Detail(ctx)
		return err
	case "Update":
		_, err := scope.Update(ctx, quotas.UpdateOpts{})
		return err
	default:
		_, err := scope.Delete(ctx)
		return err
	}
}

func TestNeutronProjectQuotaSuccessCodesMatchNativePolicy(t *testing.T) {
	for _, operation := range []string{"Get", "Detail", "Update", "Delete"} {
		for _, status := range []int{200, 201, 202, 204} {
			t.Run(fmt.Sprintf("%s-%d", operation, status), func(t *testing.T) {
				cloud := testcloud.New(t)
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					if operation == "Delete" {
						w.Header().Set("X-Openstack-Request-Id", "delete-success")
						w.WriteHeader(status)
						return
					}
					if operation == "Detail" {
						testcloud.JSON(w, status, `{"quota":{"network":{"limit":1,"used":0,"reserved":0}}}`)
						return
					}
					testcloud.JSON(w, status, neutronQuotaLimits)
				})
				scope := newNeutronQuotaScope(t, cloud)
				if operation == "Delete" {
					value, err := scope.Delete(context.Background())
					if status == 202 || status == 204 {
						if err != nil || value == nil || value.StatusCode != status || value.Header.Get("X-Openstack-Request-Id") != "delete-success" {
							t.Fatalf("DELETE=%+v error=%v", value, err)
						}
					} else if value != nil || !gophercloud.ResponseCodeIs(err, status) {
						t.Fatalf("wrong DELETE success policy=%+v/%v", value, err)
					}
				} else if err := neutronQuotaOperation(scope, operation, context.Background()); (err == nil) != (status == 200) || status != 200 && !gophercloud.ResponseCodeIs(err, status) {
					t.Fatalf("success policy error=%v", err)
				}
			})
		}
	}
}

func TestNeutronProjectQuotaRejectsMalformedObjects(t *testing.T) {
	for _, operation := range []string{"Get", "Detail", "Update"} {
		for _, body := range []string{`{}`, `{"quota":null}`, `{"quota":[]}`, `{"quota":"scalar"}`, `{"quota":{"network":"invalid"}}`} {
			t.Run(operation+body, func(t *testing.T) {
				cloud := testcloud.New(t)
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, body) })
				if err := neutronQuotaOperation(newNeutronQuotaScope(t, cloud), operation, context.Background()); err == nil {
					t.Fatalf("malformed response accepted: %s", body)
				}
			})
		}
	}
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"quota":{"network":{"limit":1,"used":0,"reserved":"not-a-number"}}}`)
	})
	if _, err := newNeutronQuotaScope(t, cloud).Detail(context.Background()); err == nil {
		t.Fatal("native reserved string compatibility accepted an invalid integer")
	}
}

func TestNeutronProjectQuotaCancellationPreservesCause(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	defer close(release)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		started <- struct{}{}
		// Client cancellation is asserted below. Server cleanup must not depend
		// on when its transport observes that a request connection was closed.
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})
	scope := newNeutronQuotaScope(t, cloud)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, operation := range []string{"Get", "Detail", "Update", "Delete"} {
		if err := neutronQuotaOperation(scope, operation, ctx); !errors.Is(err, context.Canceled) || calls.Load() != 0 {
			t.Fatalf("%s calls=%d error=%v", operation, calls.Load(), err)
		}
	}
	ctx, cancel = context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { _, err := scope.Update(ctx, quotas.UpdateOpts{}); result <- err }()
	select {
	case <-started:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("update request did not reach server")
	}
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := scope.Detail(ctx); !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 2 {
		t.Fatalf("detail calls=%d error=%v", calls.Load(), err)
	}
}
