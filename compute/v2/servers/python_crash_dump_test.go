package servers_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonTriggerServerCrashDumpOwnsPayloadRouteAndSelectedVersion(t *testing.T) {
	// Source trigger_crash_dump delegates to _action without an action-local
	// 2.17 check. Nova's server-side minimum is not a new local rejection gate.
	for _, version := range []string{"", "2.1", "2.16", "2.17", "2.100", "latest"} {
		t.Run("version="+version, func(t *testing.T) {
			client, _, calls := pythonServerCloud(t, func(req *http.Request) *http.Response {
				values, exists := req.Header["Accept"]
				if !exists || len(values) != 1 || values[0] != "" || req.Header.Get("X-Owned") != "captured" || req.Header.Get("X-Auth-Token") != "test-token" {
					t.Fatal(req.Header)
				}
				return nativeServerWire(202, `accepted without JSON`)
			})
			client.Microversion = version
			client.MoreHeaders = map[string]string{"Accept": "application/foreign"}
			err := servers.New(client).TriggerServerCrashDump(context.Background(), "folder/name%", servers.WithUserActionHeader("X-Owned", "captured"), servers.WithUserActionHeader("Accept", "caller"))
			if err != nil || calls.Len() != 1 {
				t.Fatal(err, pythonServerCalls(calls))
			}
			call := (pythonServerCalls(calls))[0]
			if call.method != http.MethodPost || !strings.HasSuffix(call.path, "/servers/folder/name%/action") || call.query != "" || call.body != `{"trigger_crash_dump":null}` {
				t.Fatal(call)
			}
			// The fixture's path captures decoded URL.Path; escaping is checked below.
			if call.version != version || client.Microversion != version || client.MoreHeaders["Accept"] != "application/foreign" {
				t.Fatal(call, client)
			}
		})
	}
	t.Run("literal escaped path", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := nativeServerClient(cloud)
		calls := 0
		cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.URL.EscapedPath() != "/nova/v2.1/servers/folder%2Fname%25/action" {
				t.Fatal(req.URL)
			}
			return nativeServerWire(202, ""), nil
		}))
		if err := servers.New(client).TriggerServerCrashDump(context.Background(), "folder/name%"); err != nil || calls != 1 {
			t.Fatal(err, calls)
		}
	})
}

func TestPythonTriggerServerCrashDumpOptionsAndSourceGuardsPrecedeHTTP(t *testing.T) {
	client, _, calls := pythonServerCloud(t, func(*http.Request) *http.Response { t.Fatal("invalid crash dump sent HTTP"); return nil })
	api := servers.New(client)
	for _, opts := range [][]servers.UserActionOption{
		{nil}, {servers.WithUserActionField("x", true)}, {servers.WithUserActionHeader("X-Auth-Token", "foreign")},
		{func(c *request.Config[servers.UserActionOpts]) error { c.Query.Set("name", "forbidden"); return nil }},
	} {
		if err := api.TriggerServerCrashDump(context.Background(), "s1", opts...); err == nil || calls.Len() != 0 {
			t.Fatal(err, pythonServerCalls(calls))
		}
	}
	for _, id := range []string{"", " ", ".", "..", "bad\n", "\xff"} {
		if err := api.TriggerServerCrashDump(context.Background(), id); !errors.Is(err, resource.ErrInvalidOption) || calls.Len() != 0 {
			t.Fatal(id, err, pythonServerCalls(calls))
		}
	}
	if err := api.TriggerServerCrashDump(nil, "s1"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	var nilAPI *servers.API
	if err := nilAPI.TriggerServerCrashDump(context.Background(), "s1"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	original := client.ResourceBase
	later := 0
	err := api.TriggerServerCrashDump(context.Background(), "s1", func(*request.Config[servers.UserActionOpts]) error { client.ResourceBase += "changed/"; return nil }, func(*request.Config[servers.UserActionOpts]) error {
		later++
		client.ResourceBase = original
		return nil
	})
	client.ResourceBase = original
	if !errors.Is(err, resource.ErrInvalidOption) || later != 0 || calls.Len() != 0 {
		t.Fatal(err, later, pythonServerCalls(calls))
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("crash dump canceled")
	cancel(cause)
	if err := api.TriggerServerCrashDump(ctx, "s1"); !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls.Len() != 0 {
		t.Fatal(err, pythonServerCalls(calls))
	}
}

func TestPythonTriggerServerCrashDumpStatusesKeepNativeOperationProof(t *testing.T) {
	for _, status := range []int{200, 202, 204, 299, 302, 399, 400, 404, 409, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			const raw = `actual action receipt, not JSON`
			client, _, calls := pythonServerCloud(t, func(*http.Request) *http.Response { return nativeServerWire(status, raw) })
			err := servers.New(client).TriggerServerCrashDump(context.Background(), "s1")
			if calls.Len() != 1 {
				t.Fatal(pythonServerCalls(calls))
			}
			if status < 400 {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var native gophercloud.ErrUnexpectedResponseCode
			var operation *resource.OperationError
			if !errors.As(err, &native) || native.Actual != status || native.Method != http.MethodPost || string(native.Body) != raw || native.ResponseHeader.Get("X-Server-Proof") != "actual" || !errors.As(err, &operation) || operation.Operation != "TriggerServerCrashDump" || operation.Resource != "servers" {
				t.Fatal(err, native, operation)
			}
		})
	}
}

func TestPythonTriggerServerCrashDumpPhysicalFaultNeverReplays(t *testing.T) {
	for _, status := range []int{202, 404, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := nativeServerClient(cloud)
			api := servers.New(client)
			readCause, closeCause, cancelCause := errors.New("crash dump read"), errors.New("crash dump close"), errors.New("crash dump cancel")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			original := client.ResourceBase
			calls, retries, reauths, closed := 0, 0, 0, 0
			cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodPost {
					t.Fatal(req.Method)
				}
				response := nativeServerWire(status, "")
				response.Body = &userActionFaultBody{reader: strings.NewReader("physical crash dump receipt"), readErr: readCause, closeErr: closeCause, afterRead: func() { client.ResourceBase = "https://foreign.invalid/"; cancel(cancelCause) }, afterClose: func() { closed++; client.ResourceBase = original }}
				return response, nil
			}))
			cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			cloud.Provider.ReauthFunc = func(context.Context) error { reauths++; return nil }
			err := api.TriggerServerCrashDump(ctx, "s1")
			for _, cause := range []error{readCause, closeCause, cancelCause, context.Canceled, resource.ErrInvalidOption} {
				if !errors.Is(err, cause) {
					t.Fatal(cause, err)
				}
			}
			if calls != 1 || closed != 1 || retries != 0 || reauths != 0 {
				t.Fatal(err, calls, closed, retries, reauths)
			}
			if status < 400 {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != "physical crash dump receipt" || proof.Header.Get("X-Server-Proof") != "actual" {
					t.Fatal(err, proof)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != status || native.Method != http.MethodPost || string(native.Body) != "physical crash dump receipt" || native.ResponseHeader.Get("X-Server-Proof") != "actual" {
					t.Fatal(err, native)
				}
			}
		})
	}
}
