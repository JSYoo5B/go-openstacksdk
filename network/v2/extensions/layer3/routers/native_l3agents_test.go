package routers_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeL3AgentTransport func(*http.Request) (*http.Response, error)

func (transport nativeL3AgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeL3AgentWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeL3AgentCall struct{ method, path, query, body string }

func nativeL3AgentAPI(t *testing.T, calls *[]nativeL3AgentCall, reply func(*http.Request) *http.Response) (*routers.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeL3AgentTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeL3AgentCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return routers.New(client), cloud
}

func nativeL3AgentCollect(seq func(func(*routers.L3Agent, error) bool)) ([]*routers.L3Agent, []error) {
	var rows []*routers.L3Agent
	var errs []error
	for value, err := range seq {
		if err != nil {
			errs = append(errs, err)
			continue
		}
		rows = append(rows, value)
	}
	return rows, errs
}

func TestNativeRouterL3AgentsSinglePageDecodeAndStatuses(t *testing.T) {
	ctx := context.Background()
	var calls []nativeL3AgentCall
	var cloud *testcloud.Cloud
	api, cloud := nativeL3AgentAPI(t, &calls, func(*http.Request) *http.Response {
		// The list is a single page; neither links form is followed.
		return nativeL3AgentWire(200, `{"agents":[{"id":"ag-1","agent_type":"L3 agent","alive":true,"host":"net-1","configurations":{"agent_mode":"dvr_snat"},"created_at":"2026-10-10 01:02:03","heartbeat_timestamp":""}],"agents_links":[{"rel":"next","href":"`+cloud.Server.URL+`/never"}],"links":{"next":"`+cloud.Server.URL+`/never"}}`)
	})
	rows, errs := nativeL3AgentCollect(api.ListL3Agents(ctx, "r-1"))
	if len(errs) != 0 || len(rows) != 1 || !(rows[0].ID == "ag-1" && rows[0].Alive && rows[0].Configurations["agent_mode"] == "dvr_snat" &&
		rows[0].CreatedAt.Equal(time.Date(2026, 10, 10, 1, 2, 3, 0, time.UTC)) && rows[0].HeartbeatTimestamp.IsZero()) {
		t.Fatal(rows, errs)
	}
	if !reflect.DeepEqual(calls, []nativeL3AgentCall{{http.MethodGet, "/neutron/v2.0/routers/r-1/l3-agents", "", ""}}) {
		t.Fatalf("%+v", calls)
	}
	for _, tc := range []struct {
		name string
		code int
		body string
	}{
		{"not found", 404, `{}`},
		{"empty", 200, `{"agents":[]}`},
		{"missing key", 200, `{}`},
		{"bodyless 204", 204, ""},
		// Agent timestamps accept only the space-separated form without a zone.
		{"RFC3339 timestamp", 200, `{"agents":[{"id":"ag-1","created_at":"2026-10-10T01:02:03Z"}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []nativeL3AgentCall
			api, _ := nativeL3AgentAPI(t, &calls, func(*http.Request) *http.Response { return nativeL3AgentWire(tc.code, tc.body) })
			rows, errs := nativeL3AgentCollect(api.ListL3Agents(ctx, "r-1"))
			var native gophercloud.ErrUnexpectedResponseCode
			var wrapped *resource.OperationError
			for _, err := range errs {
				if errors.As(err, &wrapped) {
					t.Fatal("list streams carry no operation context", err)
				}
			}
			switch tc.name {
			case "not found":
				if len(errs) != 1 || !errors.As(errs[0], &native) || !reflect.DeepEqual(native.Expected, []int{200, 204, 300}) {
					t.Fatal(errs)
				}
			case "empty", "missing key":
				if len(rows) != 0 || len(errs) != 0 {
					t.Fatal(rows, errs)
				}
			case "bodyless 204":
				if len(errs) != 1 || !errors.Is(errs[0], io.EOF) {
					t.Fatal(errs)
				}
			default:
				if len(rows) != 0 || len(errs) != 1 {
					t.Fatal(rows, errs)
				}
			}
			if len(calls) != 1 {
				t.Fatal(calls)
			}
		})
	}
}
