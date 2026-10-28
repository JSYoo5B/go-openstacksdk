package compute_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type consoleCloudFixture struct {
	service       *compute.Service
	cloud         *testcloud.Cloud
	client        *gophercloud.ServiceClient
	calls, bodies []string
}

func newConsoleCloudFixture(t *testing.T, reply func(*http.Request) *http.Response) *consoleCloudFixture {
	t.Helper()
	cloud := testcloud.New(t)
	client := cloud.Client("compute", "/nova/v2.1/")
	client.Microversion = "2.96"
	f := &consoleCloudFixture{cloud: cloud, client: client}
	f.service = compute.New(client, compute.Dependencies{})
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		f.calls = append(f.calls, req.Method+" "+req.URL.EscapedPath()+"?"+req.URL.RawQuery)
		var raw []byte
		if req.Body != nil {
			raw, _ = io.ReadAll(req.Body)
		}
		f.bodies = append(f.bodies, string(raw))
		if req.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(req.Header)
		}
		return reply(req), nil
	}))
	return f
}
func consoleCloudAction(body string) *http.Response { return backupServerWire(200, body, "console") }
func TestCloudServerConsoleBareLookupAndSeedBypass(t *testing.T) {
	for _, seed := range []bool{false, true} {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			f := newConsoleCloudFixture(t, func(req *http.Request) *http.Response {
				if req.Method == "GET" {
					return backupServerWire(200, `{"server":{"id":"s1","name":"vm","addresses":{"net":[{"addr":"10.0.0.2","version":4}]}}}`, "lookup")
				}
				return consoleCloudAction(`{"output":"console"}`)
			})
			var opts []compute.ServerConsoleOption
			if seed {
				opts = append(opts, compute.WithServerConsoleServerJSON(json.RawMessage(`{"id":"s1","name":"seed"}`)))
			}
			result, err := f.service.GetServerConsole(context.Background(), "ignored-or-vm", opts...)
			n := 2
			if seed {
				n = 1
			}
			if err != nil || result == nil || result.Action == nil || string(result.Value) != `"console"` || !result.HasOutput || result.Unsupported || len(f.calls) != n || f.calls[n-1] != "POST /nova/v2.1/servers/s1/action?" || f.bodies[n-1] != `{"os-getConsoleOutput":{}}` {
				t.Fatal(result, err, f.calls, f.bodies)
			}
			if seed {
				if result.Server != nil || string(result.Seed) != `{"id":"s1","name":"seed"}` {
					t.Fatal(result)
				}
			} else if result.Server == nil || result.Server.StatusCode != 200 || result.Server.Header.Get("X-Backup-Stage") != "lookup" || result.Seed != nil {
				t.Fatal(result)
			}
		})
	}
}
func TestCloudServerConsoleNameFallbackUsesDetailsLocalScopeAndAmbiguity(t *testing.T) {
	for _, lane := range []string{"found", "missing", "duplicate", "member400", "lookupfault"} {
		t.Run(lane, func(t *testing.T) {
			fault := errors.New("lookup read")
			f := newConsoleCloudFixture(t, func(req *http.Request) *http.Response {
				if req.Method == "POST" {
					return consoleCloudAction(`{"output":false}`)
				}
				if req.URL.Path == "/nova/v2.1/servers/detail" {
					if req.URL.Query().Has("all_tenants") || req.URL.Query().Has("all_projects") || req.URL.Query().Get("name") != "vm" {
						t.Error(req.URL)
					}
					switch lane {
					case "missing":
						return backupServerWire(200, `{"servers":[]}`, "list")
					case "duplicate":
						return backupServerWire(200, `{"servers":[{"id":"s1","name":"vm"},{"id":"s2","name":"vm"}]}`, "list")
					}
					return backupServerWire(200, `{"servers":[{"id":"s1","name":"vm"}]}`, "list")
				}
				if lane == "lookupfault" {
					return testcloud.StreamResponse(400, &testcloud.Body{Data: `{"bad":"lookup"}`, ReadErr: fault}, http.Header{"X-Backup-Stage": {"lookup"}})
				}
				if lane == "member400" {
					return backupServerWire(400, `{"bad":"lookup"}`, "member")
				}
				return backupServerWire(404, `{}`, "member")
			})
			result, err := f.service.GetServerConsole(context.Background(), "vm")
			if lane == "missing" || lane == "duplicate" || lane == "lookupfault" {
				if err == nil || result == nil || result.Action != nil {
					t.Fatal(result, err, f.calls)
				}
				for _, call := range f.calls {
					if len(call) > 4 && call[:4] == "POST" {
						t.Fatal(f.calls)
					}
				}
				if lane == "missing" && !errors.Is(err, resource.ErrNotFound) || lane == "duplicate" && !errors.Is(err, resource.ErrAmbiguous) || lane == "lookupfault" && !errors.Is(err, fault) {
					t.Fatal(err)
				}
			} else if err != nil || string(result.Value) != "false" || len(f.calls) != 3 || result.Server == nil {
				t.Fatal(result, err, f.calls)
			}
		})
	}
}
func TestCloudServerConsoleCleanAction400OnlyAndRawPresence(t *testing.T) {
	for _, tc := range []struct {
		status                      int
		body, want                  string
		presence, unsupported, fail bool
	}{
		{400, `{"bad":"unsupported"}`, `""`, false, true, false}, {403, `{}`, "", false, false, true}, {404, `{}`, "", false, false, true}, {500, `{}`, "", false, false, true},
		{200, `{"output":null}`, "null", true, false, false}, {200, `{"output":0}`, "0", true, false, false}, {200, `{"output":{}}`, "{}", true, false, false}, {200, `{"Output":"wrong"}`, "null", false, false, false}, {200, `[]`, "null", false, false, false}, {200, `null`, "", false, false, true},
	} {
		t.Run(fmt.Sprintf("%d/%s", tc.status, tc.body), func(t *testing.T) {
			f := newConsoleCloudFixture(t, func(*http.Request) *http.Response { return backupServerWire(tc.status, tc.body, "console") })
			result, err := f.service.GetServerConsole(context.Background(), "ignored", compute.WithServerConsoleServerJSON(json.RawMessage(`{"id":"s1"}`)))
			if result == nil || result.Action == nil || result.Action.StatusCode != tc.status || result.Action.Header.Get("X-Backup-Stage") != "console" || string(result.Action.Envelope) != tc.body || len(f.calls) != 1 {
				t.Fatal(result, err, f.calls)
			}
			if tc.fail {
				if err == nil || result.Unsupported {
					t.Fatal(result, err)
				}
			} else if err != nil || string(result.Value) != tc.want || result.HasOutput != tc.presence || result.Unsupported != tc.unsupported {
				t.Fatal(result, err)
			}
		})
	}
}
func TestCloudServerConsoleLengthServerSnapshotsRecordAndReset(t *testing.T) {
	length := json.RawMessage(`{"raw":false}`)
	seed := json.RawMessage(`{"id":"s1"}`)
	option := compute.WithServerConsoleOpts(compute.ServerConsoleOpts{Length: &length, Server: &seed})
	length[2] = 'X'
	seed[2] = 'X'
	f := newConsoleCloudFixture(t, func(req *http.Request) *http.Response {
		if req.Method == "GET" {
			return backupServerWire(200, `{"server":{"id":"s2"}}`, "lookup")
		}
		return consoleCloudAction(`{"output":"ok"}`)
	})
	result, err := f.service.GetServerConsole(context.Background(), "vm", option)
	if err != nil || result.Server != nil || f.bodies[0] != `{"os-getConsoleOutput":{"length":{"raw":false}}}` || f.calls[0] != "POST /nova/v2.1/servers/s1/action?" {
		t.Fatal(result, err, f.calls, f.bodies)
	}
	_, err = f.service.GetServerConsole(context.Background(), "vm", option, compute.WithoutServerConsoleServer(), compute.WithoutServerConsoleLength())
	if err != nil || len(f.calls) != 3 || f.calls[2] != "POST /nova/v2.1/servers/s2/action?" || f.bodies[2] != `{"os-getConsoleOutput":{}}` {
		t.Fatal(err, f.calls, f.bodies)
	}
	record := &servers.ServerFindRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"s3"`)}}}}
	owned := compute.WithServerConsoleServerRecord(record)
	record.Resource.Body["id"] = json.RawMessage(`"changed"`)
	_, err = f.service.GetServerConsole(context.Background(), "ignored", owned, compute.WithServerConsoleLength(0))
	if err != nil || f.calls[3] != "POST /nova/v2.1/servers/s3/action?" || f.bodies[3] != `{"os-getConsoleOutput":{"length":0}}` {
		t.Fatal(err, f.calls, f.bodies)
	}
}
func TestCloudServerConsoleDictionaryFalseyIDsAndMissingKeys(t *testing.T) {
	for _, tc := range []struct {
		seed, path string
		fail       bool
	}{
		{`{"id":null}`, "/nova/v2.1/servers//action", false}, {`{"id":false}`, "/nova/v2.1/servers//action", false}, {`{"id":0}`, "/nova/v2.1/servers//action", false}, {`{"id":""}`, "/nova/v2.1/servers//action", false}, {`{"id":42}`, "/nova/v2.1/servers/42/action", false}, {`{"id":true}`, "/nova/v2.1/servers/True/action", false},
		{`{"id":{"id":"nested"}}`, "/nova/v2.1/servers/nested/action", false},
		{`{"id":{"name":"without-id"}}`, "/nova/v2.1/servers//action", false},
		{`{}`, "", true}, {`{"name":"no-id"}`, "", true}, {`null`, "", true}, {`[]`, "", true}, {string([]byte{'{', '"', 'i', 'd', '"', ':', '"', 0xff, '"', '}'}), "", true},
	} {
		t.Run(tc.seed, func(t *testing.T) {
			f := newConsoleCloudFixture(t, func(*http.Request) *http.Response { return consoleCloudAction(`{}`) })
			got, err := f.service.GetServerConsole(context.Background(), "ignored", compute.WithServerConsoleServerJSON(json.RawMessage(tc.seed)))
			if tc.fail {
				if err == nil || len(f.calls) != 0 {
					t.Fatal(got, err, f.calls)
				}
			} else if err != nil || !reflect.DeepEqual(f.calls, []string{"POST " + tc.path + "?"}) {
				t.Fatal(got, err, f.calls)
			}
		})
	}
}
func TestCloudServerConsoleNilCancelledAndSourceBindingsPreflight(t *testing.T) {
	for _, lane := range []string{"nil-service", "nil-context", "cancel", "nil-option", "component", "source", "nil-record"} {
		t.Run(lane, func(t *testing.T) {
			f := newConsoleCloudFixture(t, func(*http.Request) *http.Response { return consoleCloudAction(`{}`) })
			service := f.service
			ctx := context.Background()
			var options []compute.ServerConsoleOption
			switch lane {
			case "nil-service":
				service = nil
			case "nil-context":
				ctx = nil
			case "cancel":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "nil-option":
				options = []compute.ServerConsoleOption{nil}
			case "component":
				service.API.Servers = servers.New(f.cloud.Client("compute", "/other/v2.1/"))
			case "source":
				options = []compute.ServerConsoleOption{func(*compute.ServerConsoleOpts) error { service.API.Servers = servers.New(f.client); return nil }}
			case "nil-record":
				options = []compute.ServerConsoleOption{compute.WithServerConsoleServerRecord(nil)}
			}
			result, err := service.GetServerConsole(ctx, "vm", options...)
			if err == nil || len(f.calls) != 0 {
				t.Fatal(result, err, f.calls)
			}
		})
	}
}
func TestCloudServerConsoleConnectionLocationSnapshotAndGuard(t *testing.T) {
	for _, mutate := range []bool{false, true} {
		t.Run(fmt.Sprint(mutate), func(t *testing.T) {
			f := newConsoleCloudFixture(t, func(req *http.Request) *http.Response {
				if req.Method == "GET" {
					return backupServerWire(200, `{"server":{"id":"s1","tenant_id":"foreign","OS-EXT-AZ:availability_zone":"zone"}}`, "lookup")
				}
				return consoleCloudAction(`{}`)
			})
			reads := 0
			f.service = compute.New(f.client, compute.Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
				reads++
				if mutate {
					f.client.ResourceBase += "changed/"
				}
				cloudName, regionName := "owned", "region"
				return resource.CloudLocation{Cloud: &cloudName, RegionName: &regionName}, nil
			}})
			got, err := f.service.GetServerConsole(context.Background(), "vm")
			if mutate {
				if err == nil || len(f.calls) != 0 || reads != 1 {
					t.Fatal(got, err, reads, f.calls)
				}
				return
			}
			if err != nil || reads != 1 || got.Server == nil || got.Server.Resource == nil {
				t.Fatal(got, err, reads)
			}
			var location map[string]json.RawMessage
			if json.Unmarshal(got.Server.Resource.Body["location"], &location) != nil || string(location["cloud"]) != `"owned"` || string(location["region_name"]) != `"region"` || string(location["zone"]) != `"zone"` {
				t.Fatal(location)
			}
		})
	}
}

func TestCloudServerConsoleDirty400NeverBecomesUnsupported(t *testing.T) {
	for _, lane := range []string{"read", "close", "cancel", "source", "retry", "expanded-codes"} {
		t.Run(lane, func(t *testing.T) {
			readErr, closeErr, cancelErr, retryErr := errors.New("read"), errors.New("close"), errors.New("cancel cause"), errors.New("retry")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			body := &testcloud.Body{Data: `{"bad":"actual"}`}
			f := newConsoleCloudFixture(t, func(*http.Request) *http.Response {
				return testcloud.StreamResponse(400, body, http.Header{"X-Actual": {"physical"}})
			})
			base := f.client.ResourceBase
			switch lane {
			case "read":
				body.ReadErr = readErr
			case "close":
				body.CloseErr = closeErr
			case "cancel":
				body.OnRead = func() { cancel(cancelErr) }
			case "source":
				body.OnRead = func() { f.client.ResourceBase += "wrong/" }
				body.OnClose = func() { f.client.ResourceBase = base }
			case "retry":
				f.cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return retryErr }
			case "expanded-codes":
				f.cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
					o.OkCodes = append(o.OkCodes, 400)
					return nil
				}
			}
			got, err := f.service.GetServerConsole(ctx, "ignored", compute.WithServerConsoleServerJSON(json.RawMessage(`{"id":"s1"}`)))
			var native gophercloud.ErrUnexpectedResponseCode
			if err == nil || got == nil || got.Unsupported || got.Action == nil || got.Action.StatusCode != 400 || got.Action.Header.Get("X-Actual") != "physical" || !errors.As(err, &native) || native.Actual != 400 {
				t.Fatal(got, err, native, f.calls)
			}
			if lane == "read" && !errors.Is(err, readErr) || lane == "close" && !errors.Is(err, closeErr) || lane == "cancel" && !errors.Is(err, cancelErr) || lane == "source" && !errors.Is(err, resource.ErrInvalidOption) || lane == "retry" && !errors.Is(err, retryErr) {
				t.Fatal(err)
			}
		})
	}
}
func TestCloudServerConsoleCleanRetryStopsAndAcceptedReadKeepsSeparateLookup(t *testing.T) {
	for _, lane := range []string{"clean-retry", "read-failure"} {
		t.Run(lane, func(t *testing.T) {
			cause := errors.New("action physicalread")
			f := newConsoleCloudFixture(t, func(req *http.Request) *http.Response {
				if req.Method == "GET" {
					return backupServerWire(200, `{"server":{"id":"s1"}}`, "lookup")
				}
				if lane == "read-failure" {
					return testcloud.StreamResponse(200, &testcloud.Body{Data: `{"output":"partial"}`, ReadErr: cause}, http.Header{"X-Backup-Stage": {"action"}})
				}
				return backupServerWire(400, `{}`, "action")
			})
			if lane == "clean-retry" {
				f.cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
					return original
				}
			}
			got, err := f.service.GetServerConsole(context.Background(), "vm")
			if got == nil || got.Server == nil || got.Server.Header.Get("X-Backup-Stage") != "lookup" || got.Action == nil || got.Action.Header.Get("X-Backup-Stage") != "action" || len(f.calls) != 2 {
				t.Fatal(got, err, f.calls)
			}
			if lane == "clean-retry" {
				if err != nil || !got.Unsupported || string(got.Value) != `""` {
					t.Fatal(got, err)
				}
			} else if !errors.Is(err, cause) || got.Unsupported {
				t.Fatal(got, err)
			}
		})
	}
}

func TestCloudServerConsoleCapturesHeadersBeforeOptionsAndAcrossLookupAction(t *testing.T) {
	for _, supplied := range []bool{false, true} {
		t.Run(fmt.Sprint(supplied), func(t *testing.T) {
			f := newConsoleCloudFixture(t, func(req *http.Request) *http.Response { return consoleCloudAction(`{"output":"ok"}`) })
			f.client.MoreHeaders = map[string]string{"X-Source": "captured"}
			var headers []http.Header
			f.cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
				headers = append(headers, req.Header.Clone())
				if req.Method == "GET" {
					f.client.MoreHeaders["X-Source"] = "after-lookup"
					return backupServerWire(200, `{"server":{"id":"s1"}}`, "lookup"), nil
				}
				return consoleCloudAction(`{"output":"ok"}`), nil
			}))
			opts := []compute.ServerConsoleOption{func(*compute.ServerConsoleOpts) error { f.client.MoreHeaders["X-Source"] = "after-option"; return nil }}
			if supplied {
				opts = append(opts, compute.WithServerConsoleServerJSON(json.RawMessage(`{"id":"s1"}`)))
			}
			got, err := f.service.GetServerConsole(context.Background(), "vm", opts...)
			wantCalls := 2
			if supplied {
				wantCalls = 1
			}
			if err != nil || got == nil || string(got.Value) != `"ok"` || len(headers) != wantCalls {
				t.Fatal(got, err, headers)
			}
			for _, h := range headers {
				if h.Get("X-Source") != "captured" || h.Get("X-Auth-Token") != "test-token" {
					t.Fatal(headers)
				}
			}
		})
	}
}
