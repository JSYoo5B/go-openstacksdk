package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const designateQuotaBody = `{"id":"wire-other","project_id":"wire-project","api_export_size":1000,"recordset_records":20,"zone_records":null,"zone_recordsets":0,"zones":-1,"vendor":{"counter":9007199254740993},"optional":null}`

func newDesignateQuotaScope(t *testing.T, cloud *testcloud.Cloud) *quotas.ProjectQuotaScope {
	t.Helper()
	scope, err := quotas.New(cloud.Client("dns", "/dns/v2")).InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func TestDesignateProjectQuotasUseRootObjectAndOfficialMethods(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, updates, resets atomic.Int32
	cloud.Mux.HandleFunc("/dns/v2/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Sudo-Project-ID") != "project-fixed" || r.Header.Get("X-Auth-All-Projects") != "true" || r.Header.Get("X-Client") != "preserved" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != "dns 2.0" {
			t.Errorf("request=%s %s headers=%s", r.Method, r.URL, r.Header)
		}
		w.Header().Add("X-Vendor", "one")
		w.Header().Add("X-Vendor", "two")
		w.Header().Set("X-Openstack-Request-Id", "quota-"+r.Method)
		switch r.Method {
		case http.MethodGet:
			gets.Add(1)
			testcloud.JSON(w, 200, designateQuotaBody)
		case http.MethodPatch:
			updates.Add(1)
			var fields map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
				t.Error(err)
			}
			if len(fields) != 4 || string(fields["zones"]) != "0" || string(fields["zone_records"]) != strconv.Itoa(int(^uint(0)>>1)) || string(fields["vendor"]) != `{"counter":9007199254740993,"enabled":false}` || string(fields["optional"]) != "null" {
				t.Errorf("root PATCH body=%s", fields)
			}
			testcloud.JSON(w, 200, designateQuotaBody)
		case http.MethodDelete:
			resets.Add(1)
			if r.ContentLength != 0 {
				t.Errorf("reset body length=%d", r.ContentLength)
			}
			w.WriteHeader(204)
		default:
			t.Errorf("unexpected method=%s", r.Method)
		}
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("quota scope used a collection, defaults or different project route: %s", r.URL)
		http.Error(w, "unexpected route", 404)
	})
	client := cloud.Client("dns", "/catalog")
	client.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + "/dns/v2")
	client.Microversion = "2.0"
	client.MoreHeaders = map[string]string{"X-Client": "preserved", "x-auth-sudo-project-id": "wrong-project", "x-auth-all-projects": "original"}
	api := quotas.New(client)
	scope, err := api.InProject(context.Background(), resource.ID("project-fixed"), quotas.WithAllProjects(true))
	if err != nil || scope.ProjectID() != "project-fixed" || gets.Load()+updates.Load()+resets.Load() != 0 {
		t.Fatal(scope, err)
	}
	for _, method := range []string{"List", "Find", "Wait", "Defaults", "InUser"} {
		if _, exists := reflect.TypeOf(scope).MethodByName(method); exists {
			t.Errorf("scope invents unsupported %s", method)
		}
	}
	value, err := scope.Get(context.Background())
	if err != nil || value == nil || value.ProjectID != "project-fixed" || value.APIExporterSize != 1000 || value.RecordsetRecords != 20 || value.ZoneRecords != 0 || value.ZoneRecordsets != 0 || value.Zones != -1 || value.StatusCode != 200 || string(value.Body["id"]) != `"wire-other"` || string(value.Body["project_id"]) != `"wire-project"` || string(value.Body["zone_records"]) != "null" || string(value.Body["optional"]) != "null" || string(value.Body["vendor"]) != `{"counter":9007199254740993}` || len(value.Header.Values("X-Vendor")) != 2 {
		t.Fatalf("quota=%+v err=%v", value, err)
	}
	zero, large := 0, int(^uint(0)>>1)
	option := quotas.WithQuotaOptions(quotas.UpdateOpts{Zones: &zero, ZoneRecords: &large})
	zero, large = 77, 88
	vendor := map[string]any{"enabled": false, "counter": json.Number("9007199254740993")}
	extension := quotas.WithUpdateField("vendor", vendor)
	vendor["enabled"] = true
	updated, err := scope.Update(context.Background(), quotas.UpdateOpts{}, option, extension, quotas.WithUpdateField("optional", nil))
	if err != nil || updated == nil || updated.ProjectID != "project-fixed" || updated.StatusCode != 200 || updated.Header.Get("X-Openstack-Request-Id") != "quota-PATCH" {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
	reset, err := scope.Reset(context.Background())
	if err != nil || reset == nil || reset.ProjectID != "project-fixed" || reset.StatusCode != 204 || reset.Header.Get("X-Openstack-Request-Id") != "quota-DELETE" || gets.Load() != 1 || updates.Load() != 1 || resets.Load() != 1 {
		t.Fatalf("reset=%+v counts=%d/%d/%d err=%v", reset, gets.Load(), updates.Load(), resets.Load(), err)
	}
	value.Body["vendor"][0] = 'X'
	value.Header["X-Vendor"][0] = "changed"
	if string(updated.Body["vendor"]) != `{"counter":9007199254740993}` || updated.Header.Values("X-Vendor")[0] != "one" || client.MoreHeaders["x-auth-sudo-project-id"] != "wrong-project" || client.MoreHeaders["x-auth-all-projects"] != "original" || client.ProviderClient != cloud.Provider || api.RawClient() != client {
		t.Fatal("response/header handling mutated source client or another response")
	}
}

func TestDesignateQuotaAllProjectsHeaderIsExplicitAndLastOptionWins(t *testing.T) {
	for _, tc := range []struct {
		name string
		with []quotas.ProjectOption
		want string
	}{
		{"preserved", nil, "True"},
		{"explicit false", []quotas.ProjectOption{quotas.WithAllProjects(false)}, "false"},
		{"last wins", []quotas.ProjectOption{quotas.WithAllProjects(true), quotas.WithAllProjects(false)}, "false"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/dns/v2/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) {
				if len(r.Header.Values("X-Auth-All-Projects")) != 1 || r.Header.Get("X-Auth-All-Projects") != tc.want || r.Header.Get("X-Auth-Sudo-Project-ID") != "project-fixed" || r.Header.Get("X-Unrelated") != "keep" {
					t.Error(r.Header)
				}
				testcloud.JSON(w, 200, `{}`)
			})
			client := cloud.Client("dns", "/dns/v2")
			client.MoreHeaders = map[string]string{"x-auth-all-projects": "True", "X-Unrelated": "keep"}
			scope, err := quotas.New(client).InProject(context.Background(), resource.ID("project-fixed"), tc.with...)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := scope.Get(context.Background()); err != nil || client.MoreHeaders["x-auth-all-projects"] != "True" || len(client.MoreHeaders) != 2 {
				t.Fatal(err, client.MoreHeaders)
			}
		})
	}
}

func TestDesignateQuotaGeneratedNativeAPIRemainsUnchanged(t *testing.T) {
	cloud := testcloud.New(t)
	var native, scoped atomic.Int32
	cloud.Mux.HandleFunc("/dns/v2/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Sudo-Project-ID") == "" {
			native.Add(1)
		} else if r.Header.Get("X-Auth-Sudo-Project-ID") == "project-fixed" {
			scoped.Add(1)
		} else {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"zones":4}`)
	})
	api := quotas.New(cloud.Client("dns", "/dns/v2"))
	if api.URL(context.Background(), "project-fixed") != cloud.Server.URL+"/dns/v2/quotas/project-fixed" {
		t.Fatal("native URL changed")
	}
	if value, err := api.Get(context.Background(), "project-fixed"); err != nil || value.Zones != 4 {
		t.Fatal(value, err)
	}
	zero := 0
	if value, err := api.Update(context.Background(), "project-fixed", quotas.UpdateOpts{Zones: &zero}); err != nil || value.Zones != 4 {
		t.Fatal(value, err)
	}
	if value, err := newDesignateQuotaScope(t, cloud).Get(context.Background()); err != nil || value.Zones != 4 || native.Load() != 2 || scoped.Load() != 1 {
		t.Fatal(value, err, native.Load(), scoped.Load())
	}
}

func TestDesignateQuotaInvalidOptionsAndZeroScopeFailBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, `{}`) })
	scope := newDesignateQuotaScope(t, cloud)
	for _, with := range [][]quotas.UpdateOption{
		{nil}, {quotas.WithUpdateField("zones", 0)}, {quotas.WithUpdateField("id", "other")},
		{quotas.WithUpdateField("project", "other")}, {quotas.WithUpdateField("project_id", "other")},
		{quotas.WithUpdateField("vendor", make(chan bool))},
		{request.WithQuery[quotas.UpdateOpts]("project", "other")},
		{request.WithHeader[quotas.UpdateOpts]("X-Auth-Sudo-Project-ID", "other")},
		{request.WithArgument[quotas.UpdateOpts]("unknown", true)},
	} {
		if value, err := scope.Update(context.Background(), quotas.UpdateOpts{}, with...); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(value, err)
		}
	}
	if value, err := scope.Reset(context.Background(), nil); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(value, err)
	}
	for _, zero := range []*quotas.ProjectQuotaScope{nil, {}} {
		for _, operation := range []string{"Get", "Update", "Reset"} {
			if err := callDesignateQuota(zero, operation, context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(operation, err)
			}
		}
	}
	if calls.Load() != 0 {
		t.Fatal("invalid options made HTTP requests")
	}
}

func callDesignateQuota(scope *quotas.ProjectQuotaScope, operation string, ctx context.Context) error {
	var err error
	switch operation {
	case "Get":
		_, err = scope.Get(ctx)
	case "Update":
		_, err = scope.Update(ctx, quotas.UpdateOpts{})
	case "Reset":
		_, err = scope.Reset(ctx)
	}
	return err
}

func TestDesignateQuotaErrorsPreserveNativeResponseAndMissingPolicy(t *testing.T) {
	for _, operation := range []string{"Get", "Update", "Reset"} {
		for _, status := range []int{403, 404, 409} {
			t.Run(operation+http.StatusText(status), func(t *testing.T) {
				cloud := testcloud.New(t)
				const body = `{"error":{"message":"quota denied"}}`
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Openstack-Request-Id", "quota-denied")
					testcloud.JSON(w, status, body)
				})
				scope := newDesignateQuotaScope(t, cloud)
				err := callDesignateQuota(scope, operation, context.Background())
				var response gophercloud.ErrUnexpectedResponseCode
				var sdk *resource.OperationError
				if !errors.As(err, &response) || response.Actual != status || string(response.Body) != body || response.ResponseHeader.Get("X-Openstack-Request-Id") != "quota-denied" || !errors.As(err, &sdk) || sdk.Operation != operation || errors.Is(err, resource.ErrNotFound) != (status == 404) {
					t.Fatalf("response=%+v operation=%+v err=%v", response, sdk, err)
				}
				if status == 404 {
					var missing *resource.NotFoundError
					if !errors.As(err, &missing) || missing.Reference != "project-fixed" {
						t.Fatal(missing, err)
					}
				}
				if operation == "Reset" {
					value, err := scope.Reset(context.Background(), quotas.WithResetIgnoreMissing(true))
					if value != nil || (err == nil) != (status == 404) {
						t.Fatal(value, err)
					}
					if _, err := scope.Reset(context.Background(), quotas.WithResetIgnoreMissing(true), quotas.WithResetIgnoreMissing(false)); !gophercloud.ResponseCodeIs(err, status) {
						t.Fatal(err)
					}
				}
			})
		}
	}
}

func TestDesignateQuotaDecodeErrorsRetainSuccessfulHTTPEvidence(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `"bad"`, `{"zones":"bad"}`, `{"zones":`, `{"zones":1} {"zones":2}`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("X-Vendor", "one")
				w.Header().Add("X-Vendor", "two")
				testcloud.JSON(w, 200, body)
			})
			value, err := newDesignateQuotaScope(t, cloud).Get(context.Background())
			var wire *quotas.QuotaResponseError
			if value != nil || !errors.As(err, &wire) || wire.StatusCode != 200 || string(wire.Body) != body || !reflect.DeepEqual(wire.Header.Values("X-Vendor"), []string{"one", "two"}) {
				t.Fatalf("value=%+v wire=%+v err=%v", value, wire, err)
			}
			if body == `{"zones":"bad"}` {
				var decode *json.UnmarshalTypeError
				if !errors.As(err, &decode) {
					t.Fatal("decode cause lost", err)
				}
			}
		})
	}
}

func TestDesignateQuotaSuccessfulCodeContractDoesNotBroaden(t *testing.T) {
	for _, operation := range []string{"Get", "Update", "Reset"} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 202, `{}`) })
		err := callDesignateQuota(newDesignateQuotaScope(t, cloud), operation, context.Background())
		var response gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &response) || response.Actual != 202 {
			t.Fatal(operation, err)
		}
	}
}

func TestDesignateQuotaContextOwnsTransportAndBodyRead(t *testing.T) {
	for _, operation := range []string{"Get", "Update", "Reset"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			release := make(chan struct{})
			t.Cleanup(func() { close(release) })
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				_, _ = io.Copy(io.Discard, r.Body)
				if operation != "Reset" {
					w.Header().Set("X-Openstack-Request-Id", "partial")
					_, _ = io.WriteString(w, `{"zones":`)
					w.(http.Flusher).Flush()
				}
				select {
				case <-r.Context().Done():
				case <-release:
				}
			})
			scope := newDesignateQuotaScope(t, cloud)
			canceled, stop := context.WithCancel(context.Background())
			stop()
			if err := callDesignateQuota(scope, operation, canceled); !errors.Is(err, context.Canceled) || calls.Load() != 0 {
				t.Fatal(err, calls.Load())
			}
			deadline, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
			defer cancel()
			err := callDesignateQuota(scope, operation, deadline)
			if !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 1 {
				t.Fatal(err, calls.Load())
			}
			if operation != "Reset" {
				var wire *quotas.QuotaResponseError
				if !errors.As(err, &wire) || wire.StatusCode != 200 || !strings.HasPrefix(string(wire.Body), `{"zones":`) || wire.Header.Get("X-Openstack-Request-Id") != "partial" {
					t.Fatal(wire, err)
				}
			}
		})
	}
}
