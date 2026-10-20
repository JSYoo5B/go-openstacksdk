package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestQoSRuleScopesKeepPolicyAndTypedRuleIdentity(t *testing.T) {
	parent := map[string]any{"id": "parent", "name": "parent-name"}
	child := map[string]any{"id": "child"}
	t.Run("bandwidth limit", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := rules.New(cloud.Client("network", "/v2.0"))
		bind := func(ctx context.Context, ref resource.Ref) (*resource.Collection[rules.BandwidthLimitRule], error) {
			s, err := api.BandwidthLimitRules(ctx, ref)
			if err != nil {
				return nil, err
			}
			return s.Collection, nil
		}
		checkScope(t, cloud, bind, "/v2.0/qos/policies", "policies", "/v2.0/qos/policies/parent/bandwidth_limit_rules", "bandwidth_limit_rules", "bandwidth_limit_rule", parent, child, true, false, false)
	})
	t.Run("DSCP marking", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := rules.New(cloud.Client("network", "/v2.0"))
		bind := func(ctx context.Context, ref resource.Ref) (*resource.Collection[rules.DSCPMarkingRule], error) {
			s, err := api.DSCPMarkingRules(ctx, ref)
			if err != nil {
				return nil, err
			}
			return s.Collection, nil
		}
		checkScope(t, cloud, bind, "/v2.0/qos/policies", "policies", "/v2.0/qos/policies/parent/dscp_marking_rules", "dscp_marking_rules", "dscp_marking_rule", parent, child, true, false, false)
	})
	t.Run("minimum bandwidth", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := rules.New(cloud.Client("network", "/v2.0"))
		bind := func(ctx context.Context, ref resource.Ref) (*resource.Collection[rules.MinimumBandwidthRule], error) {
			s, err := api.MinimumBandwidthRules(ctx, ref)
			if err != nil {
				return nil, err
			}
			return s.Collection, nil
		}
		checkScope(t, cloud, bind, "/v2.0/qos/policies", "policies", "/v2.0/qos/policies/parent/minimum_bandwidth_rules", "minimum_bandwidth_rules", "minimum_bandwidth_rule", parent, child, true, false, false)
	})
}

func TestQoSRuleCRUDExtractsCorrectEnvelopeAndPreservesHTTPError(t *testing.T) {
	cloud := testcloud.New(t)
	for _, name := range []string{"bandwidth_limit", "dscp_marking", "minimum_bandwidth"} {
		cloud.Mux.HandleFunc("/v2.0/qos/policies/policy/"+name+"_rules", func(w http.ResponseWriter, r *http.Request) {
			var body map[string]map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if r.Method != http.MethodPost || body[name+"_rule"]["vendor:enabled"] != false {
				t.Errorf("method=%s body=%v", r.Method, body)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(201)
			_ = json.NewEncoder(w).Encode(map[string]any{name + "_rule": map[string]any{"id": "rule", "max_kbps": 123, "dscp_mark": 16, "min_kbps": 99}})
		})
		cloud.Mux.HandleFunc("/v2.0/qos/policies/policy/"+name+"_rules/rule", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPut {
				var body map[string]map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body[name+"_rule"]["vendor:enabled"] != false {
					t.Errorf("body=%v", body)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{name + "_rule": map[string]any{"id": "rule", "max_kbps": 123, "dscp_mark": 16, "min_kbps": 99}})
		})
	}
	cloud.Mux.HandleFunc("/v2.0/qos/policies/policy/bandwidth_limit_rules/forbidden", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 403, "{}") })
	api := rules.New(cloud.Client("network", "/v2.0"))
	ctx := context.Background()
	bandwidth, err := api.CreateBandwidthLimitRule(ctx, "policy", rules.CreateBandwidthLimitRuleOpts{MaxKBps: 123}, rules.WithCreateBandwidthLimitRuleField("vendor:enabled", false))
	if err != nil || bandwidth.ID != "rule" || bandwidth.MaxKBps != 123 {
		t.Fatalf("rule=%v err=%v", bandwidth, err)
	}
	bandwidth, err = api.GetBandwidthLimitRule(ctx, "policy", "rule")
	if err != nil || bandwidth.MaxKBps != 123 {
		t.Fatalf("rule=%v err=%v", bandwidth, err)
	}
	scope, err := api.BandwidthLimitRules(ctx, resource.ID("policy"))
	if err != nil {
		t.Fatal(err)
	}
	bandwidth, err = scope.Update(ctx, resource.ID("rule"), rules.UpdateBandwidthLimitRuleOpts{}, rules.WithUpdateBandwidthLimitRuleField("vendor:enabled", false))
	if err != nil || bandwidth.MaxKBps != 123 {
		t.Fatalf("rule=%v err=%v", bandwidth, err)
	}
	dscp, err := api.CreateDSCPMarkingRule(ctx, "policy", rules.CreateDSCPMarkingRuleOpts{DSCPMark: 16}, rules.WithCreateDSCPMarkingRuleField("vendor:enabled", false))
	if err != nil || dscp.ID != "rule" || dscp.DSCPMark != 16 {
		t.Fatalf("rule=%v err=%v", dscp, err)
	}
	dscp, err = api.GetDSCPMarkingRule(ctx, "policy", "rule")
	if err != nil || dscp.DSCPMark != 16 {
		t.Fatalf("rule=%v err=%v", dscp, err)
	}
	dscp, err = api.UpdateDSCPMarkingRule(ctx, "policy", "rule", rules.UpdateDSCPMarkingRuleOpts{}, rules.WithUpdateDSCPMarkingRuleField("vendor:enabled", false))
	if err != nil || dscp.DSCPMark != 16 {
		t.Fatalf("rule=%v err=%v", dscp, err)
	}
	minimum, err := api.CreateMinimumBandwidthRule(ctx, "policy", rules.CreateMinimumBandwidthRuleOpts{MinKBps: 99}, rules.WithCreateMinimumBandwidthRuleField("vendor:enabled", false))
	if err != nil || minimum.ID != "rule" || minimum.MinKBps != 99 {
		t.Fatalf("rule=%v err=%v", minimum, err)
	}
	minimum, err = api.GetMinimumBandwidthRule(ctx, "policy", "rule")
	if err != nil || minimum.MinKBps != 99 {
		t.Fatalf("rule=%v err=%v", minimum, err)
	}
	minimum, err = api.UpdateMinimumBandwidthRule(ctx, "policy", "rule", rules.UpdateMinimumBandwidthRuleOpts{}, rules.WithUpdateMinimumBandwidthRuleField("vendor:enabled", false))
	if err != nil || minimum.MinKBps != 99 {
		t.Fatalf("rule=%v err=%v", minimum, err)
	}
	_, err = api.GetBandwidthLimitRule(ctx, "policy", "forbidden")
	var response gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &response) || response.Actual != 403 {
		t.Fatal(err)
	}
}

func TestNovaNamedResultExtractorsReturnImageIDAndAdminPassword(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/v2.1/project/servers/server/action", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if _, ok := body["createImage"]; ok {
			w.Header().Set("X-OpenStack-Nova-API-Version", "2.1")
			w.Header().Set("Location", cloud.Server.URL+"/v2.1/project/images/image-id")
			w.WriteHeader(202)
			return
		}
		if _, ok := body["evacuate"]; !ok {
			t.Errorf("body=%v", body)
		}
		testcloud.JSON(w, 200, "{\"adminPass\":\"secret\"}")
	})
	api := servers.New(cloud.Client("compute", "/v2.1/project"))
	id, err := api.CreateImage(context.Background(), "server", servers.CreateImageOpts{Name: "snapshot"})
	if err != nil || id != "image-id" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	password, err := api.Evacuate(context.Background(), "server", servers.EvacuateOpts{})
	if err != nil || password != "secret" {
		t.Fatalf("password=%q err=%v", password, err)
	}
}
