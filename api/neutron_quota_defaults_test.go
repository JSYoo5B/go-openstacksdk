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

func TestNeutronQuotaDefaultsUsesDistinctEndpointAndKeepsRawResponse(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/neutron/v2.0/quotas/project-fixed/default", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Configured") != "original" {
			t.Errorf("default request=%s %s headers=%v", r.Method, r.URL, r.Header)
		}
		w.Header().Set("X-Openstack-Request-Id", "default-quota")
		testcloud.JSON(w, 200, `{"quota":{"network":-1,"port":0,"floatingip":50,"vendor":{"big":9007199254740993},"optional":null}}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("defaults used a different endpoint: %s", r.URL)
	})
	client := cloud.Client("network", "/catalog")
	client.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + "/neutron/v2.0")
	client.MoreHeaders = map[string]string{"X-Configured": "original"}
	scope, err := quotas.New(client).InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := scope.Defaults(context.Background())
	if err != nil || first == nil || first.ProjectID != "project-fixed" || first.Network != -1 || first.Port != 0 || first.FloatingIP != 50 || first.StatusCode != 200 || first.Header.Get("X-Openstack-Request-Id") != "default-quota" || string(first.Body["vendor"]) != `{"big":9007199254740993}` || string(first.Body["optional"]) != "null" {
		t.Fatalf("defaults=%+v error=%v", first, err)
	}
	if _, exists := first.Body["project_id"]; exists {
		t.Fatal("missing response identity was invented")
	}
	first.Body["vendor"][0] = '['
	first.Header.Set("X-Openstack-Request-Id", "caller-change")
	second, err := scope.Defaults(context.Background())
	if err != nil || string(second.Body["vendor"]) != `{"big":9007199254740993}` || second.Header.Get("X-Openstack-Request-Id") != "default-quota" || calls.Load() != 2 {
		t.Fatalf("second defaults=%+v calls=%d error=%v", second, calls.Load(), err)
	}
	if client.MoreHeaders["X-Configured"] != "original" || client.ResourceBase != gophercloud.NormalizeURL(cloud.Server.URL+"/neutron/v2.0") || client.Microversion != "" {
		t.Fatal("Defaults changed the client")
	}
}

func TestNeutronQuotaDefaultsPreservesHTTPAndDecodeErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		missing    bool
	}{
		{"not found", `{"error":"missing default"}`, 404, true},
		{"forbidden", `{"error":"forbidden"}`, 403, false},
		{"unexpected success", `{"quota":{}}`, 201, false},
		{"missing object", `{}`, 200, false},
		{"null object", `{"quota":null}`, 200, false},
		{"array object", `{"quota":[]}`, 200, false},
		{"scalar object", `{"quota":"bad"}`, 200, false},
		{"bad limit", `{"quota":{"network":"bad"}}`, 200, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/neutron/v2.0/quotas/project-fixed/default", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Openstack-Request-Id", "default-error")
				testcloud.JSON(w, tc.status, tc.body)
			})
			value, err := newNeutronQuotaScope(t, cloud).Defaults(context.Background())
			if value != nil || err == nil || errors.Is(err, resource.ErrNotFound) != tc.missing {
				t.Fatalf("value=%v error=%v", value, err)
			}
			if tc.status != 200 {
				var cause gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &cause) || cause.Actual != tc.status || string(cause.Body) != tc.body || cause.ResponseHeader.Get("X-Openstack-Request-Id") != "default-error" {
					t.Fatalf("HTTP cause=%+v error=%v", cause, err)
				}
			}
			if tc.name == "bad limit" {
				var cause *json.UnmarshalTypeError
				if !errors.As(err, &cause) {
					t.Fatalf("decode cause lost: %v", err)
				}
			}
		})
	}
}

func TestNeutronQuotaDefaultsCancellationAndPreflight(t *testing.T) {
	cloud := testcloud.New(t)
	started, release := make(chan struct{}, 1), make(chan struct{})
	defer close(release)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/neutron/v2.0/quotas/project-fixed/default", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		started <- struct{}{}
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})
	scope := newNeutronQuotaScope(t, cloud)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if value, err := scope.Defaults(cancelled); value != nil || !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatalf("preflight=%v calls=%d error=%v", value, calls.Load(), err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := scope.Defaults(ctx); done <- err }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancel cause=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("request did not cancel")
	}
	timeout, stop := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer stop()
	if value, err := scope.Defaults(timeout); value != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout value=%v error=%v", value, err)
	}
}
