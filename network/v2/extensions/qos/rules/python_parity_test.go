package rules_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonQoSRuleParityTransport func(*http.Request) (*http.Response, error)

func (transport pythonQoSRuleParityTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonQoSRuleParityWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type pythonQoSRuleParityCall struct{ method, path, body string }

type pythonQoSRuleParityRecorder struct {
	mu    sync.Mutex
	calls []pythonQoSRuleParityCall
}

func (r *pythonQoSRuleParityRecorder) snapshot() []pythonQoSRuleParityCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]pythonQoSRuleParityCall(nil), r.calls...)
}

func pythonQoSRuleParityAPI(t *testing.T, recorder *pythonQoSRuleParityRecorder, reply func(*http.Request) *http.Response) *rules.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonQoSRuleParityTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		recorder.mu.Lock()
		recorder.calls = append(recorder.calls, pythonQoSRuleParityCall{req.Method, req.URL.Path + "?" + req.URL.RawQuery, raw})
		recorder.mu.Unlock()
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return rules.New(client)
}

func pythonQoSRuleParityExpect(t *testing.T, got, want []pythonQoSRuleParityCall) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("calls\n got %+v\nwant %+v", got, want)
	}
	for i := range got {
		g, w := got[i], want[i]
		same := g.body == w.body
		if !same && g.body != "" && w.body != "" {
			var x, y any
			same = json.Unmarshal([]byte(g.body), &x) == nil && json.Unmarshal([]byte(w.body), &y) == nil && reflect.DeepEqual(x, y)
		}
		if g.method != w.method || g.path != w.path || !same {
			t.Fatalf("call %d\n got %+v\nwant %+v", i, g, w)
		}
	}
}

func pythonQoSRuleParityInt(v int) *int { return &v }

const pythonQoSRuleParityBase = "/neutron/v2.0/qos/policies/qp-1/"

// The rule Resources are plain Resources under /qos/policies/%(qos_policy_id)s; the
// policy ID is a URI attribute and never enters the body.
func TestPythonQoSRuleParityCreateUpdate(t *testing.T) {
	ctx := context.Background()
	t.Run("bandwidth_limit_rule", func(t *testing.T) {
		var recorder pythonQoSRuleParityRecorder
		api := pythonQoSRuleParityAPI(t, &recorder, func(req *http.Request) *http.Response {
			code := 200
			if req.Method == http.MethodPost {
				code = 201
			}
			return pythonQoSRuleParityWire(code, `{"bandwidth_limit_rule":{"id":"r-1","max_kbps":1000,"max_burst_kbps":100,"direction":"egress","qos_policy_id":"qp-1"}}`)
		})
		// create_qos_bandwidth_limit_rule("qp-1", max_kbps=1000, max_burst_kbps=100, direction="egress")
		created, err := api.CreateBandwidthLimitRule(ctx, "qp-1", rules.CreateBandwidthLimitRuleOpts{MaxKBps: 1000, MaxBurstKBps: 100, Direction: "egress"})
		if err != nil || created.ID != "r-1" || created.MaxKBps != 1000 || created.MaxBurstKBps != 100 || created.Direction != "egress" {
			t.Fatal(created, err)
		}
		// max_kbps=0 is sent; Go always sends max_kbps, even where Python would omit it.
		if _, err := api.CreateBandwidthLimitRule(ctx, "qp-1", rules.CreateBandwidthLimitRuleOpts{}); err != nil {
			t.Fatal(err)
		}
		// update_qos_bandwidth_limit_rule("r-1", "qp-1", max_kbps=0, direction="ingress")
		if _, err := api.UpdateBandwidthLimitRule(ctx, "qp-1", "r-1", rules.UpdateBandwidthLimitRuleOpts{MaxKBps: pythonQoSRuleParityInt(0), Direction: "ingress"}); err != nil {
			t.Fatal(err)
		}
		// update_qos_bandwidth_limit_rule("r-1", "qp-1") sends nothing in Python; Go always sends the PUT.
		if _, err := api.UpdateBandwidthLimitRule(ctx, "qp-1", "r-1", rules.UpdateBandwidthLimitRuleOpts{}); err != nil {
			t.Fatal(err)
		}
		base := pythonQoSRuleParityBase + "bandwidth_limit_rules"
		pythonQoSRuleParityExpect(t, recorder.snapshot(), []pythonQoSRuleParityCall{
			{http.MethodPost, base + "?", `{"bandwidth_limit_rule":{"max_kbps":1000,"max_burst_kbps":100,"direction":"egress"}}`},
			{http.MethodPost, base + "?", `{"bandwidth_limit_rule":{"max_kbps":0}}`},
			{http.MethodPut, base + "/r-1?", `{"bandwidth_limit_rule":{"max_kbps":0,"direction":"ingress"}}`},
			{http.MethodPut, base + "/r-1?", `{"bandwidth_limit_rule":{}}`},
		})
	})
	t.Run("dscp_marking_rule", func(t *testing.T) {
		var recorder pythonQoSRuleParityRecorder
		api := pythonQoSRuleParityAPI(t, &recorder, func(req *http.Request) *http.Response {
			code := 200
			if req.Method == http.MethodPost {
				code = 201
			}
			return pythonQoSRuleParityWire(code, `{"dscp_marking_rule":{"id":"r-2","dscp_mark":26}}`)
		})
		// create_qos_dscp_marking_rule("qp-1", dscp_mark=26)
		created, err := api.CreateDSCPMarkingRule(ctx, "qp-1", rules.CreateDSCPMarkingRuleOpts{DSCPMark: 26})
		if err != nil || created.ID != "r-2" || created.DSCPMark != 26 {
			t.Fatal(created, err)
		}
		// update_qos_dscp_marking_rule("r-2", "qp-1", dscp_mark=0)
		if _, err := api.UpdateDSCPMarkingRule(ctx, "qp-1", "r-2", rules.UpdateDSCPMarkingRuleOpts{DSCPMark: pythonQoSRuleParityInt(0)}); err != nil {
			t.Fatal(err)
		}
		base := pythonQoSRuleParityBase + "dscp_marking_rules"
		pythonQoSRuleParityExpect(t, recorder.snapshot(), []pythonQoSRuleParityCall{
			{http.MethodPost, base + "?", `{"dscp_marking_rule":{"dscp_mark":26}}`},
			{http.MethodPut, base + "/r-2?", `{"dscp_marking_rule":{"dscp_mark":0}}`},
		})
	})
	t.Run("minimum_bandwidth_rule", func(t *testing.T) {
		var recorder pythonQoSRuleParityRecorder
		api := pythonQoSRuleParityAPI(t, &recorder, func(req *http.Request) *http.Response {
			code := 200
			if req.Method == http.MethodPost {
				code = 201
			}
			return pythonQoSRuleParityWire(code, `{"minimum_bandwidth_rule":{"id":"r-3","min_kbps":500,"direction":"egress"}}`)
		})
		// create_qos_minimum_bandwidth_rule("qp-1", min_kbps=500, direction="egress", x_extension=1)
		created, err := api.CreateMinimumBandwidthRule(ctx, "qp-1", rules.CreateMinimumBandwidthRuleOpts{MinKBps: 500, Direction: "egress"}, rules.WithCreateMinimumBandwidthRuleField("x_extension", 1))
		if err != nil || created.ID != "r-3" || created.MinKBps != 500 || created.Direction != "egress" {
			t.Fatal(created, err)
		}
		// update_qos_minimum_bandwidth_rule("r-3", "qp-1", min_kbps=1000)
		if _, err := api.UpdateMinimumBandwidthRule(ctx, "qp-1", "r-3", rules.UpdateMinimumBandwidthRuleOpts{MinKBps: pythonQoSRuleParityInt(1000)}); err != nil {
			t.Fatal(err)
		}
		base := pythonQoSRuleParityBase + "minimum_bandwidth_rules"
		pythonQoSRuleParityExpect(t, recorder.snapshot(), []pythonQoSRuleParityCall{
			{http.MethodPost, base + "?", `{"minimum_bandwidth_rule":{"min_kbps":500,"direction":"egress","x_extension":1}}`},
			{http.MethodPut, base + "/r-3?", `{"minimum_bandwidth_rule":{"min_kbps":1000}}`},
		})
	})
	t.Run("errors are returned", func(t *testing.T) {
		var recorder pythonQoSRuleParityRecorder
		api := pythonQoSRuleParityAPI(t, &recorder, func(*http.Request) *http.Response { return pythonQoSRuleParityWire(409, `{}`) })
		if _, err := api.CreateBandwidthLimitRule(ctx, "qp-1", rules.CreateBandwidthLimitRuleOpts{MaxKBps: 1}); !gophercloud.ResponseCodeIs(err, http.StatusConflict) {
			t.Fatal(err)
		}
		if _, err := api.UpdateMinimumBandwidthRule(ctx, "qp-1", "r-3", rules.UpdateMinimumBandwidthRuleOpts{MinKBps: pythonQoSRuleParityInt(1)}); !gophercloud.ResponseCodeIs(err, http.StatusConflict) {
			t.Fatal(err)
		}
	})
}

// delete_qos_*_rule defaults to ignore_missing=True; resource.IgnoreMissing gives the direct delete that default.
func TestPythonQoSRuleParityDeleteIgnoreMissing(t *testing.T) {
	ctx := context.Background()
	deletes := map[string]func(*rules.API) error{
		"bandwidth_limit_rules":   func(api *rules.API) error { return api.DeleteBandwidthLimitRule(ctx, "qp-1", "r-1") },
		"dscp_marking_rules":      func(api *rules.API) error { return api.DeleteDSCPMarkingRule(ctx, "qp-1", "r-1") },
		"minimum_bandwidth_rules": func(api *rules.API) error { return api.DeleteMinimumBandwidthRule(ctx, "qp-1", "r-1") },
	}
	for collection, call := range deletes {
		for _, tc := range []struct {
			name   string
			code   int
			ignore bool
			check  func(error) bool
		}{
			{"deleted", 204, true, func(err error) bool { return err == nil }},
			{"ignore_missing=True", 404, true, func(err error) bool { return err == nil }},
			{"ignore_missing=False", 404, false, func(err error) bool { return gophercloud.ResponseCodeIs(err, http.StatusNotFound) }},
			{"conflict is not hidden", 409, true, func(err error) bool { return gophercloud.ResponseCodeIs(err, http.StatusConflict) }},
		} {
			t.Run(collection+"/"+tc.name, func(t *testing.T) {
				var recorder pythonQoSRuleParityRecorder
				api := pythonQoSRuleParityAPI(t, &recorder, func(*http.Request) *http.Response { return pythonQoSRuleParityWire(tc.code, `{}`) })
				err := call(api)
				if tc.ignore {
					err = resource.IgnoreMissing(err)
				}
				if !tc.check(err) {
					t.Fatal(err)
				}
				pythonQoSRuleParityExpect(t, recorder.snapshot(), []pythonQoSRuleParityCall{{http.MethodDelete, pythonQoSRuleParityBase + collection + "/r-1?", ""}})
			})
		}
	}
}
