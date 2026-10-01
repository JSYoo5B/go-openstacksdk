package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/compute/v2/limits"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

const novaLimitsBody = `{"limits":{"absolute":{"maxTotalCores":-1,"maxTotalInstances":0,"totalCoresUsed":null,"vendor":9007199254740993},"rate":[{"uri":"*","regex":".*","limit":[{"next-available":9007199254740993,"remaining":0,"unit":"MINUTE","value":120,"verb":"POST","vendor":{"tier":"legacy"}}],"vendor":null}],"id":"wire-project","optional":null}}`

func newNovaLimitsScope(t *testing.T, cloud *testcloud.Cloud) *limits.ProjectLimitsScope {
	t.Helper()
	scope, err := limits.New(cloud.Client("compute", "/nova")).InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func TestNovaLimitsFetchAndProjectGetPreserveLegacyRateAndRawResponse(t *testing.T) {
	cloud := testcloud.New(t)
	const projectID = "project+tag&scope=chosen"
	var current, project atomic.Int32
	cloud.Mux.HandleFunc("/nova/limits", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Configured") != "keep" || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.57" {
			t.Error(r.Method, r.URL, r.Header)
		}
		if r.URL.Query().Has("tenant_id") {
			project.Add(1)
			if r.URL.RawQuery != (url.Values{"tenant_id": []string{projectID}, "reserved": []string{"1"}}).Encode() || len(r.URL.Query()["tenant_id"]) != 1 {
				t.Error(r.URL)
			}
		} else {
			current.Add(1)
			if r.URL.RawQuery != "" {
				t.Error(r.URL)
			}
		}
		w.Header().Set("X-Openstack-Request-Id", "limits-response")
		w.Header().Add("X-Vendor", "one")
		w.Header().Add("X-Vendor", "two")
		testcloud.JSON(w, 200, novaLimitsBody)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("limits invented an ID route: %s", r.URL)
		http.Error(w, "wrong route", 404)
	})
	client := cloud.Client("compute", "/catalog")
	client.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + "/nova")
	client.Microversion = "2.57"
	client.MoreHeaders = map[string]string{"X-Configured": "keep"}
	api := limits.New(client)
	plain, err := api.Fetch(context.Background())
	if err != nil || plain == nil || plain.ProjectID != "" || plain.Absolute.MaxTotalCores != -1 || plain.Absolute.MaxTotalInstances != 0 || plain.Absolute.TotalCoresUsed != 0 || string(plain.AbsoluteBody["totalCoresUsed"]) != "null" || string(plain.AbsoluteBody["vendor"]) != "9007199254740993" || string(plain.Body["optional"]) != "null" || string(plain.Body["id"]) != `"wire-project"` || plain.StatusCode != 200 || plain.Header.Get("X-Openstack-Request-Id") != "limits-response" || len(plain.Header.Values("X-Vendor")) != 2 {
		t.Fatalf("limits=%+v err=%v", plain, err)
	}
	if len(plain.Rate) != 1 || plain.Rate[0].URI != "*" || plain.Rate[0].Regex != ".*" || string(plain.Rate[0].Body["vendor"]) != "null" || len(plain.Rate[0].Limits) != 1 {
		t.Fatal("legacy rate groups lost", plain.Rate)
	}
	rule := plain.Rate[0].Limits[0]
	if string(rule.NextAvailable) != "9007199254740993" || rule.Remaining != 0 || rule.Value != 120 || rule.Unit != "MINUTE" || rule.Verb != "POST" || string(rule.Body["vendor"]) != `{"tier":"legacy"}` {
		t.Fatal("legacy rate rule lost", rule)
	}
	scope, err := api.InProject(context.Background(), resource.ID(projectID))
	if err != nil || scope.ProjectID() != projectID || project.Load() != 0 {
		t.Fatal(scope, err)
	}
	for _, unsupported := range []string{"List", "Find", "Wait", "Update", "Reset", "Defaults", "InUser", "InClass"} {
		if _, exists := reflect.TypeOf(scope).MethodByName(unsupported); exists {
			t.Errorf("limits scope invents %s", unsupported)
		}
	}
	bound, err := scope.Get(context.Background(), limits.WithGetReserved(true))
	if err != nil || bound.ProjectID != projectID || bound.StatusCode != 200 || project.Load() != 1 {
		t.Fatal(bound, err)
	}
	plain.Header["X-Vendor"][0] = "changed"
	plain.AbsoluteBody["vendor"][0] = '0'
	plain.Body["rate"][0] = 'X'
	plain.Rate[0].Body["vendor"][0] = 'X'
	plain.Rate[0].Limits[0].NextAvailable[0] = '0'
	plain.Rate[0].Limits[0].Body["vendor"][0] = 'X'
	if bound.Header.Values("X-Vendor")[0] != "one" || string(bound.AbsoluteBody["vendor"]) != "9007199254740993" || string(bound.Rate[0].Limits[0].NextAvailable) != "9007199254740993" || string(bound.Rate[0].Limits[0].Body["vendor"]) != `{"tier":"legacy"}` || string(bound.Rate[0].Body["vendor"]) != "null" || current.Load() != 1 || client.MoreHeaders["X-Configured"] != "keep" || client.ProviderClient != cloud.Provider || api.RawClient() != client {
		t.Fatal("responses or source client share mutable state")
	}
}

func TestNovaLimitsReservedHelperUsesIntegersAndRawQueryKeepsNativeValues(t *testing.T) {
	for _, tc := range []struct {
		name, wire string
		option     limits.GetOption
	}{
		{"include", "1", limits.WithGetReserved(true)},
		{"exclude", "0", limits.WithGetReserved(false)},
		{"negative includes", "-7", limits.WithGetQuery("reserved", "-7")},
		{"other nonzero includes", "2", limits.WithGetQuery("reserved", "2")},
		{"noninteger preserved", "true", limits.WithGetQuery("reserved", "true")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/nova/limits", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if got := r.URL.Query()["reserved"]; !reflect.DeepEqual(got, []string{tc.wire}) {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"limits":{"absolute":{},"rate":[]}}`)
			})
			api := limits.New(cloud.Client("compute", "/nova"))
			if _, err := api.Fetch(context.Background(), tc.option); err != nil {
				t.Fatal(err)
			}
			scope := newNovaLimitsScope(t, cloud)
			if _, err := scope.Get(context.Background(), tc.option); err != nil {
				t.Fatal(err)
			}
			if _, err := api.Get(context.Background(), tc.option); err != nil || calls.Load() != 3 {
				t.Fatal("native Get changed or helper incompatible", err, calls.Load())
			}
		})
	}
}

func TestNovaLimitsFetchExplicitTenantAndLargeTypedIntegers(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/nova/limits", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if !reflect.DeepEqual(r.URL.Query()["tenant_id"], []string{"project-fixed"}) || r.URL.Query().Get("reserved") != "0" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"limits":{"absolute":{"maxTotalCores":`+strconv.Itoa(int(^uint(0)>>1))+`},"rate":[{"limit":[{"next-available":"2026-10-01T00:00:00Z","value":`+strconv.Itoa(int(^uint(0)>>1))+`}]}]}}`)
	})
	api := limits.New(cloud.Client("compute", "/nova"))
	for _, option := range []limits.GetOption{limits.WithGetOptions(limits.GetOpts{TenantID: "project-fixed"}), limits.WithGetQuery("tenant_id", "project-fixed")} {
		value, err := api.Fetch(context.Background(), option, limits.WithGetReserved(false))
		if err != nil || value.ProjectID != "project-fixed" || value.Absolute.MaxTotalCores != int(^uint(0)>>1) || value.Rate[0].Limits[0].Value != int(^uint(0)>>1) || string(value.Rate[0].Limits[0].NextAvailable) != `"2026-10-01T00:00:00Z"` || string(value.AbsoluteBody["maxTotalCores"]) != strconv.Itoa(int(^uint(0)>>1)) {
			t.Fatal("integer/timestamp changed", value, err)
		}
	}
	if calls.Load() != 2 {
		t.Fatal(calls.Load())
	}
}

func TestNovaLimitsNullEmptyAndOmittedFieldsRemainDistinct(t *testing.T) {
	for _, tc := range []struct {
		body          string
		ratePresent   bool
		rateNonNil    bool
		absoluteField string
	}{
		{`{"limits":{}}`, false, false, ""},
		{`{"limits":{"absolute":null,"rate":null}}`, true, false, "null"},
		{`{"limits":{"absolute":{},"rate":[]}}`, true, true, "{}"},
	} {
		t.Run(tc.body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, tc.body) })
			value, err := limits.New(cloud.Client("compute", "/nova")).Fetch(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			_, hasRate := value.Body["rate"]
			if hasRate != tc.ratePresent || (value.Rate != nil) != tc.rateNonNil || string(value.Body["absolute"]) != tc.absoluteField || value.Absolute.MaxTotalCores != 0 {
				t.Fatal("null/omission/empty list conflated", value)
			}
		})
	}
}

func TestNovaLimitsFixedScopeRejectsIdentityOverridesBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("invalid option reached HTTP=%s", r.URL) })
	scope := newNovaLimitsScope(t, cloud)
	for _, option := range []limits.GetOption{
		nil, limits.WithGetOptions(limits.GetOpts{TenantID: "other-project"}), limits.WithGetOptions(limits.GetOpts{TenantID: "project-fixed"}),
		limits.WithGetQuery("tenant_id", ""), limits.WithGetQuery("tenant_id", "project-fixed"), limits.WithGetQuery("project_id", "other-project"),
		request.WithField[limits.GetOpts]("vendor", true), request.WithHeader[limits.GetOpts]("X-Vendor", "x"), request.WithArgument[limits.GetOpts]("unknown", true),
	} {
		if value, err := scope.Get(context.Background(), option); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(value, err)
		}
	}
	api := limits.New(cloud.Client("compute", "/nova"))
	for _, with := range [][]limits.GetOption{
		{nil}, {limits.WithGetQuery("project_id", "project-fixed")}, {limits.WithGetQuery("tenant_id", "")},
		{limits.WithGetOptions(limits.GetOpts{TenantID: "bad/path"})},
		{limits.WithGetOptions(limits.GetOpts{TenantID: "project-fixed"}), limits.WithGetQuery("tenant_id", "other")},
		{func(config *request.Config[limits.GetOpts]) error {
			config.Query["tenant_id"] = []string{"one", "two"}
			return nil
		}},
	} {
		if value, err := api.Fetch(context.Background(), with...); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(value, err)
		}
	}
}

func TestNovaLimitsHTTPFailuresRetainOriginalEvidenceAndOperation(t *testing.T) {
	for _, scoped := range []bool{false, true} {
		for _, status := range []int{403, 404, 203} {
			cloud := testcloud.New(t)
			const body = `{"error":{"message":"limits denied"}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Openstack-Request-Id", "denied")
				testcloud.JSON(w, status, body)
			})
			var err error
			operation := "Fetch"
			if scoped {
				_, err = newNovaLimitsScope(t, cloud).Get(context.Background())
				operation = "Get"
			} else {
				_, err = limits.New(cloud.Client("compute", "/nova")).Fetch(context.Background())
			}
			var response gophercloud.ErrUnexpectedResponseCode
			var sdk *resource.OperationError
			if !errors.As(err, &response) || response.Actual != status || string(response.Body) != body || response.ResponseHeader.Get("X-Openstack-Request-Id") != "denied" || !errors.As(err, &sdk) || sdk.Operation != operation || errors.Is(err, resource.ErrNotFound) != (status == 404) {
				t.Fatal(operation, response, sdk, err)
			}
		}
	}
}

func TestNovaLimitsDecodeFailuresRetainAcceptedHTTPBodyHeaderAndStatus(t *testing.T) {
	for _, body := range []string{`{}`, `null`, `[]`, `{"limits":null}`, `{"limits":[]}`, `{"limits":{"absolute":[]}}`, `{"limits":{"absolute":{"maxTotalCores":"bad"}}}`, `{"limits":{"rate":"bad"}}`, `{"limits":{"rate":[null]}}`, `{"limits":{"rate":[{"limit":[null]}]}}`, `{"limits":{"rate":[{"limit":[{"remaining":"bad"}]}]}}`, `{"limits":`, `{"limits":{}} {"limits":{}}`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("X-Vendor", "one")
				w.Header().Add("X-Vendor", "two")
				testcloud.JSON(w, 200, body)
			})
			value, err := limits.New(cloud.Client("compute", "/nova")).Fetch(context.Background())
			var wire *limits.LimitsResponseError
			if value != nil || !errors.As(err, &wire) || wire.StatusCode != 200 || string(wire.Body) != body || !reflect.DeepEqual(wire.Header.Values("X-Vendor"), []string{"one", "two"}) || errors.Unwrap(wire) == nil {
				t.Fatal(value, wire, err)
			}
			if body == `{"limits":{"absolute":{"maxTotalCores":"bad"}}}` {
				var cause *json.UnmarshalTypeError
				if !errors.As(err, &cause) {
					t.Fatal("JSON cause lost", err)
				}
			}
		})
	}
}

func TestNovaLimitsContextOwnsHeaderWaitAndPartialBodyRead(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("X-Openstack-Request-Id", "partial")
		_, _ = io.WriteString(w, `{"limits":`)
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})
	scope := newNovaLimitsScope(t, cloud)
	canceled, stop := context.WithCancel(context.Background())
	stop()
	if value, err := scope.Get(canceled); value != nil || !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatal(value, err, calls.Load())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	value, err := scope.Get(ctx)
	var wire *limits.LimitsResponseError
	if value != nil || !errors.Is(err, context.DeadlineExceeded) || !errors.As(err, &wire) || wire.StatusCode != 200 || string(wire.Body) != `{"limits":` || wire.Header.Get("X-Openstack-Request-Id") != "partial" || calls.Load() != 1 {
		t.Fatal(value, wire, err, calls.Load())
	}
}
