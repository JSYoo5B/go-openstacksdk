package services_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

// pythonServiceCall records the wire request openstacksdk would also send.
type pythonServiceCall struct{ method, path, query, body, version string }

type pythonServiceTransport func(*http.Request) (*http.Response, error)

func (transport pythonServiceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonServiceWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonServiceAPI(t *testing.T, microversion string, reply func(*http.Request) *http.Response) (*services.API, *[]pythonServiceCall) {
	t.Helper()
	calls := &[]pythonServiceCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonServiceTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonServiceCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("OpenStack-API-Version")})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	client.Microversion = microversion
	return services.New(client), calls
}

const pythonServicePath = "/cinder/v3/project/os-services"

const pythonServiceRows = `{"services":[
	{"binary":"cinder-volume","host":"node@lvm","zone":"nova","status":"enabled","state":"up","updated_at":"2026-10-11T01:02:03.000000","disabled_reason":null,"frozen":true,"cluster":"c1","replication_status":"enabled","active_backend_id":"secondary","backend_state":"up"},
	{"binary":"cinder-scheduler","host":"node","zone":"nova","status":"disabled","state":"down","updated_at":null,"disabled_reason":"maintenance"}]}`

func pythonServiceCollect(t *testing.T, seq func(func(*services.Service, error) bool)) []services.Service {
	t.Helper()
	var rows []services.Service
	for value, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, *value)
	}
	return rows
}

func TestPythonServicesListMatchesProxyQuery(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		options []services.ListOption
		query   string
	}{
		// services() sends no query.
		{"default", nil, ""},
		// services(binary=..., host=...) uses the two declared QueryParameters.
		{"binary and host", []services.ListOption{services.WithListOptions(services.ListOpts{Binary: "cinder-volume", Host: "node@lvm"})}, "binary=cinder-volume&host=node%40lvm"},
		// services(limit=..., marker=...) uses the default pagination QueryParameters.
		{"pagination keys", []services.ListOption{services.WithListQuery("limit", "1"), services.WithListQuery("marker", "m")}, "limit=1&marker=m"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api, calls := pythonServiceAPI(t, "3.32", func(*http.Request) *http.Response { return pythonServiceWire(200, pythonServiceRows) })
			rows := pythonServiceCollect(t, api.List(ctx, tc.options...))
			want := []pythonServiceCall{{http.MethodGet, pythonServicePath, tc.query, "", "volume 3.32"}}
			if !reflect.DeepEqual(*calls, want) {
				t.Fatalf("%+v", *calls)
			}
			if len(rows) != 2 {
				t.Fatal(rows)
			}
			// Python exposes the same keys as binary/name, host, zone, status, state, frozen, cluster, replication_status, active_backend_id and disabled_reason.
			first, second := rows[0], rows[1]
			if !(first.Binary == "cinder-volume" && first.Host == "node@lvm" && first.Zone == "nova" && first.Status == "enabled" && first.State == "up" &&
				first.Frozen && first.Cluster == "c1" && first.ReplicationStatus == "enabled" && first.ActiveBackendID == "secondary" && first.DisabledReason == "" &&
				first.UpdatedAt.Equal(time.Date(2026, 10, 11, 1, 2, 3, 0, time.UTC))) {
				t.Fatalf("%+v", first)
			}
			if !(second.Binary == "cinder-scheduler" && second.Status == "disabled" && second.State == "down" && second.DisabledReason == "maintenance" && second.UpdatedAt.IsZero()) {
				t.Fatalf("%+v", second)
			}
		})
	}
}

// Python filters Body attributes that are not QueryParameters (zone, status, ...) locally; Go callers filter the rows.
func TestPythonServicesListLocalFilterIsCallerSide(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonServiceAPI(t, "", func(*http.Request) *http.Response { return pythonServiceWire(200, pythonServiceRows) })
	var enabled []string
	for _, row := range pythonServiceCollect(t, api.List(ctx)) {
		if row.Status == "enabled" {
			enabled = append(enabled, row.Binary)
		}
	}
	if !reflect.DeepEqual(enabled, []string{"cinder-volume"}) {
		t.Fatal(enabled)
	}
	if want := []pythonServiceCall{{http.MethodGet, pythonServicePath, "", "", ""}}; !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

// find_service lists with only the extra query (host) and matches name_or_id locally; Go has no matcher.
func TestPythonServiceFindListPhase(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonServiceAPI(t, "", func(*http.Request) *http.Response { return pythonServiceWire(200, pythonServiceRows) })
	rows := pythonServiceCollect(t, api.List(ctx, services.WithListOptions(services.ListOpts{Host: "node@lvm"})))
	if want := []pythonServiceCall{{http.MethodGet, pythonServicePath, "host=node%40lvm", "", ""}}; !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
	var matched []services.Service
	for _, row := range rows {
		if row.Binary == "cinder-volume" && row.Host == "node@lvm" {
			matched = append(matched, row)
		}
	}
	if len(matched) != 1 {
		t.Fatal(matched)
	}
}
