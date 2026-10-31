package agents_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonL3AgentWire(code int, body string) *http.Response {
	return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}})
}

type pythonL3AgentCall struct{ method, path, query, body string }

type pythonL3AgentRecorder struct {
	mu    sync.Mutex
	calls []pythonL3AgentCall
}

func (r *pythonL3AgentRecorder) snapshot() []pythonL3AgentCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]pythonL3AgentCall(nil), r.calls...)
}

func pythonL3AgentAPI(t *testing.T, recorder *pythonL3AgentRecorder, reply func(*http.Request) *http.Response) *agents.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		recorder.mu.Lock()
		recorder.calls = append(recorder.calls, pythonL3AgentCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		recorder.mu.Unlock()
		return reply(req), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return agents.New(client)
}

func pythonL3AgentExpect(t *testing.T, got, want []pythonL3AgentCall) {
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
		if g.method != w.method || g.path != w.path || g.query != w.query || !same {
			t.Fatalf("call %d\n got %+v\nwant %+v", i, g, w)
		}
	}
}

const pythonL3AgentBase = "/neutron/v2.0/agents/a-1/l3-routers"

// Agent.add_router_to_agent and remove_router_from_agent use raw session calls on /agents/{id}/l3-routers.
func TestPythonL3AgentRouterScheduling(t *testing.T) {
	ctx := context.Background()
	t.Run("add_router_to_agent", func(t *testing.T) {
		var recorder pythonL3AgentRecorder
		api := pythonL3AgentAPI(t, &recorder, func(req *http.Request) *http.Response {
			if strings.Contains(req.URL.Path, "/a-x/") {
				return pythonL3AgentWire(404, `{"NeutronError":{"type":"AgentNotFound"}}`)
			}
			return pythonL3AgentWire(201, `null`)
		})
		// add_router_to_agent("a-1", "r-1")
		if err := api.ScheduleL3Router(ctx, "a-1", agents.ScheduleL3RouterOpts{RouterID: "r-1"}); err != nil {
			t.Fatal(err)
		}
		// add_router_to_agent("a-1", "r-1", ha_chassis_priority=5)
		if err := api.ScheduleL3Router(ctx, "a-1", agents.ScheduleL3RouterOpts{RouterID: "r-1"}, agents.WithScheduleL3RouterField("ha_chassis_priority", 5)); err != nil {
			t.Fatal(err)
		}
		// Python returns the error body as a dict because the proxy does not raise; Go returns the 404.
		if err := api.ScheduleL3Router(ctx, "a-x", agents.ScheduleL3RouterOpts{RouterID: "r-1"}); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			t.Fatal(err)
		}
		pythonL3AgentExpect(t, recorder.snapshot(), []pythonL3AgentCall{
			{http.MethodPost, pythonL3AgentBase, "", `{"router_id":"r-1"}`},
			{http.MethodPost, pythonL3AgentBase, "", `{"router_id":"r-1","ha_chassis_priority":5}`},
			{http.MethodPost, "/neutron/v2.0/agents/a-x/l3-routers", "", `{"router_id":"r-1"}`},
		})
	})
	t.Run("remove_router_from_agent", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			code  int
			check func(error) bool
		}{
			{"removed", 204, func(err error) bool { return err == nil }},
			// Python never raises here; Go reports every non-2xx status.
			{"missing", 404, func(err error) bool { return gophercloud.ResponseCodeIs(err, http.StatusNotFound) }},
			{"conflict", 409, func(err error) bool { return gophercloud.ResponseCodeIs(err, http.StatusConflict) }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var recorder pythonL3AgentRecorder
				api := pythonL3AgentAPI(t, &recorder, func(*http.Request) *http.Response { return pythonL3AgentWire(tc.code, ``) })
				// remove_router_from_agent("a-1", "r-1"); Python also sends {"router_id":"r-1"} as a DELETE body.
				if err := api.RemoveL3Router(ctx, "a-1", "r-1"); !tc.check(err) {
					t.Fatal(err)
				}
				pythonL3AgentExpect(t, recorder.snapshot(), []pythonL3AgentCall{{http.MethodDelete, pythonL3AgentBase + "/r-1", "", ""}})
			})
		}
	})
	t.Run("agent_hosted_routers", func(t *testing.T) {
		var recorder pythonL3AgentRecorder
		api := pythonL3AgentAPI(t, &recorder, func(*http.Request) *http.Response {
			return pythonL3AgentWire(200, `{"routers":[{"id":"r-1","name":"edge","status":"ACTIVE","admin_state_up":true,"distributed":false,"ha":true,"project_id":"proj"},{"id":"r-2","name":"core"}]}`)
		})
		// agent_hosted_routers("a-1") lists L3AgentRouter (Router fields) with no query.
		routers, err := api.ListL3Routers(ctx, "a-1")
		if err != nil || len(routers) != 2 || routers[0].ID != "r-1" || routers[0].Name != "edge" || routers[0].Status != "ACTIVE" || !routers[0].AdminStateUp || routers[0].Distributed || routers[0].ProjectID != "proj" || routers[1].ID != "r-2" {
			t.Fatal(routers, err)
		}
		pythonL3AgentExpect(t, recorder.snapshot(), []pythonL3AgentCall{{http.MethodGet, pythonL3AgentBase, "", ""}})
	})
	t.Run("agent_hosted_routers error", func(t *testing.T) {
		var recorder pythonL3AgentRecorder
		api := pythonL3AgentAPI(t, &recorder, func(*http.Request) *http.Response { return pythonL3AgentWire(404, `{}`) })
		if _, err := api.ListL3Routers(ctx, "a-1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			t.Fatal(err)
		}
	})
}
