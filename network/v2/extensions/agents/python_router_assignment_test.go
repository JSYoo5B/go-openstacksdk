package agents_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPythonRouterAssignmentPriorityPayloadAndRawJSONStatus(t *testing.T) {
	for _, tc := range []struct {
		name, priority, body string
		status               int
	}{{"omitted", "", "{}", 200}, {"null", "null", "{}", 202}, {"zero", "0", `{"ha_chassis_priority":0}`, 200}, {"negative", "-1", `{"ha_chassis_priority":-1}`, 200}, {"array", "[]", `{"ha_chassis_priority":[]}`, 404}} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonParityCoreAgentCall
			api, _ := pythonParityCoreAgentAPI(t, &calls, func(r *http.Request) (int, string) {
				if r.Method != "PUT" || r.URL.EscapedPath() != "/neutron/v2.0/agents/agent%2Fid/l3-routers/router%2Fid" {
					t.Error(r.Method, r.URL)
				}
				return tc.status, `[null,false,9007199254740993]`
			})
			var opts []agents.RouterAssignmentOption
			if tc.priority != "" {
				opts = append(opts, agents.WithRouterAssignmentPriorityJSON(json.RawMessage(tc.priority)))
			}
			result, err := api.UpdateRouterInAgent(context.Background(), "agent/id", "router/id", opts...)
			if err != nil || result == nil || len(calls) != 1 || calls[0].body != tc.body || result.StatusCode != tc.status || string(result.Value) != `[null,false,9007199254740993]` {
				t.Fatal(result, err, calls)
			}
			clone := result.Clone()
			clone.Value[1] = 'X'
			if strings.Contains(string(result.Value), "X") || string(result.Envelope) != `[null,false,9007199254740993]` {
				t.Fatal("result proof aliases")
			}
		})
	}
}
func TestPythonRouterAssignmentInvalidJSONAndPhysicalEvidence(t *testing.T) {
	var calls []pythonParityCoreAgentCall
	api, cloud := pythonParityCoreAgentAPI(t, &calls, func(*http.Request) (int, string) { return 409, "opaque non JSON" })
	result, err := api.UpdateRouterInAgent(context.Background(), "a", "r")
	var proof *resource.ResponseError
	if result == nil || result.StatusCode != 409 || !errors.As(err, &proof) || proof.StatusCode != 409 || string(proof.Body) != "opaque non JSON" {
		t.Fatal(result, err)
	}
	fault := errors.New("router assignment body failed")
	cloud.SetTransport(testcloud.LinkRequest(func(*http.Request) (*http.Response, error) {
		return testcloud.StreamResponse(200, &routerAssignmentFault{Reader: strings.NewReader(`{"ok":true}`), cause: fault}, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	_, err = api.UpdateRouterInAgent(context.Background(), "a", "r")
	if !errors.Is(err, fault) || !errors.As(err, &proof) {
		t.Fatal(err)
	}
}

type routerAssignmentFault struct {
	io.Reader
	cause error
}

func (f *routerAssignmentFault) Read(p []byte) (int, error) {
	n, _ := f.Reader.Read(p)
	return n, f.cause
}
func (f *routerAssignmentFault) Close() error { return nil }
func TestPythonRouterAssignmentOptionsOwnershipAndPreflight(t *testing.T) {
	var calls []pythonParityCoreAgentCall
	api, cloud := pythonParityCoreAgentAPI(t, &calls, func(r *http.Request) (int, string) {
		if r.Header.Get("X-Owned") != "original" {
			t.Error(r.Header)
		}
		return 200, `{"actual":true}`
	})
	priority := json.RawMessage(`12`)
	headers := map[string]string{"X-Owned": "original"}
	option := agents.WithRouterAssignmentOpts(agents.RouterAssignmentOpts{Priority: priority, Headers: headers})
	priority[0] = '9'
	headers["X-Owned"] = "changed"
	_, err := api.UpdateRouterInAgent(context.Background(), "a", "r", option)
	if err != nil || len(calls) != 1 || calls[0].body != `{"ha_chassis_priority":12}` {
		t.Fatal(err, calls)
	}
	before := len(calls)
	for _, opts := range [][]agents.RouterAssignmentOption{{nil}, {agents.WithRouterAssignmentPriorityJSON(json.RawMessage(`{bad`))}, {agents.WithRouterAssignmentHeaders(map[string]string{"X-Auth-Token": "foreign"})}} {
		_, err = api.UpdateRouterInAgent(context.Background(), "a", "r", opts...)
		if err == nil || len(calls) != before {
			t.Fatal(err, calls)
		}
	}
	_, err = api.UpdateRouterInAgent(context.Background(), "a", "r", func(*agents.RouterAssignmentOpts) error {
		*api = *agents.New(cloud.Client("network", "/changed/"))
		return nil
	})
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
		t.Fatal(err, calls)
	}
}

func TestPythonRouterAssignmentVersionOverrideAndCancellation(t *testing.T) {
	var calls []pythonParityCoreAgentCall
	api, _ := pythonParityCoreAgentAPI(t, &calls, func(r *http.Request) (int, string) {
		if r.Header.Get("OpenStack-API-Version") != "network 2.5" {
			t.Error(r.Header)
		}
		return 200, `null`
	})
	client := api.RawClient()
	client.Microversion = "2.0"
	client.MoreHeaders = map[string]string{"OpenStack-API-Version": "network 2.0"}
	result, err := api.UpdateRouterInAgent(context.Background(), "a", "r", agents.WithRouterAssignmentMicroversion("2.5"))
	if err != nil || result == nil || string(result.Value) != "null" || len(calls) != 1 || client.Microversion != "2.0" || client.MoreHeaders["OpenStack-API-Version"] != "network 2.0" {
		t.Fatal(result, err, calls, client)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = api.UpdateRouterInAgent(ctx, "a", "r")
	if !errors.Is(err, context.Canceled) || len(calls) != 1 {
		t.Fatal(err, calls)
	}
}
