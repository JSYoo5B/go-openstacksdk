package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Octavia's QuotaBase request type (octavia/api/v2/types/quotas.py) declares
// both loadbalancer/healthmonitor and the deprecated load_balancer/
// health_monitor spellings. These fixtures pin the exact request keys sent for
// each spelling choice; server-side acceptance is source-reviewed, not
// deployment-verified.
func TestOctaviaQuotaLegacyKeysOptInSpelling(t *testing.T) {
	five, zero, unlimited := 5, 0, -1
	for _, tc := range []struct {
		name    string
		options []quotas.UpdateOption
		want    map[string]string
	}{
		{
			name:    "default documented keys",
			options: []quotas.UpdateOption{quotas.WithDefaultLimit(quotas.LimitHealthMonitors)},
			want:    map[string]string{"loadbalancer": "5", "healthmonitor": "null", "listener": "0", "l7policy": "-1"},
		},
		{
			name:    "python spelling",
			options: []quotas.UpdateOption{quotas.WithLegacyQuotaKeys(true), quotas.WithDefaultLimit(quotas.LimitHealthMonitors)},
			want:    map[string]string{"load_balancer": "5", "health_monitor": "null", "listener": "0", "l7policy": "-1"},
		},
		{
			name:    "last option wins",
			options: []quotas.UpdateOption{quotas.WithLegacyQuotaKeys(true), quotas.WithLegacyQuotaKeys(false), quotas.WithDefaultLimit(quotas.LimitHealthMonitors)},
			want:    map[string]string{"loadbalancer": "5", "healthmonitor": "null", "listener": "0", "l7policy": "-1"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var puts atomic.Int32
			cloud.Mux.HandleFunc("PUT /v2/lbaas/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) {
				puts.Add(1)
				var body struct {
					Quota map[string]json.RawMessage `json:"quota"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				got := map[string]string{}
				for key, value := range body.Quota {
					got[key] = string(value)
				}
				if !reflect.DeepEqual(got, tc.want) {
					t.Errorf("body=%v want=%v", got, tc.want)
				}
				testcloud.JSON(w, 202, `{"quota":{"load_balancer":5,"loadbalancer":5,"health_monitor":-1,"healthmonitor":-1,"listener":0,"l7policy":-1}}`)
			})
			value, err := newOctaviaQuotaScope(t, cloud).Update(context.Background(), quotas.UpdateOpts{Loadbalancer: &five, Listener: &zero, L7Policy: &unlimited}, tc.options...)
			if err != nil || value == nil || value.Loadbalancer != 5 || value.Healthmonitor != -1 || value.StatusCode != 202 || puts.Load() != 1 {
				t.Fatalf("value=%+v error=%v puts=%d", value, err, puts.Load())
			}
		})
	}
}

func TestOctaviaQuotaLegacyKeysRejectAliasesAndForgedArgumentBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected HTTP %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	scope, err := quotas.New(cloud.Client("load-balancer", "/v2")).InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil {
		t.Fatal(err)
	}
	five := 5
	for name, options := range map[string][]quotas.UpdateOption{
		"legacy alias extension":    {quotas.WithLegacyQuotaKeys(true), quotas.WithUpdateField("load_balancer", 1)},
		"legacy monitor extension":  {quotas.WithLegacyQuotaKeys(true), quotas.WithUpdateField("health_monitor", 1)},
		"canonical alias extension": {quotas.WithLegacyQuotaKeys(true), quotas.WithUpdateField("loadbalancer", 1)},
		"forged argument type":      {request.WithArgument[quotas.UpdateOpts]("quota_legacy_keys", "yes")},
		"value and default":         {quotas.WithLegacyQuotaKeys(true), quotas.WithDefaultLimit(quotas.LimitLoadBalancers)},
	} {
		value, err := scope.Update(context.Background(), quotas.UpdateOpts{Loadbalancer: &five}, options...)
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: value=%+v error=%v", name, value, err)
		}
	}
}
