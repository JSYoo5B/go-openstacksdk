package servers_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonUserActionsRoutesDefaultsAndOptionalValues(t *testing.T) {
	ctx := context.Background()
	client, _, calls := pythonServerCloud(t, func(req *http.Request) *http.Response {
		if req.Method == http.MethodPost {
			values, exists := req.Header["Accept"]
			if !exists || len(values) != 1 || values[0] != "" {
				t.Error("action must retain Source empty Accept header", req.Header)
			}
		}
		if req.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(req.Header)
		}
		return nativeServerWire(202, "accepted, not JSON")
	})
	client.Microversion = "2.96"
	api := servers.New(client)
	steps := []struct {
		body string
		call func() error
	}{
		{``, func() error { return api.ClearPassword(ctx, "s1") }},
		{`{"restore":null}`, func() error { return api.Restore(ctx, "s1") }},
		{`{"lock":null}`, func() error { return api.LockWithOptions(ctx, "s1") }},
		{`{"lock":{"locked_reason":"because"}}`, func() error { return api.LockWithOptions(ctx, "s1", servers.WithLockedReason("because")) }},
		{`{"lock":{"locked_reason":""}}`, func() error { return api.LockWithOptions(ctx, "s1", servers.WithLockedReason("")) }},
		{`{"lock":null}`, func() error {
			return api.LockWithOptions(ctx, "s1", servers.WithLockedReason("because"), servers.WithoutLockedReason())
		}},
		{`{"unshelve":null}`, func() error { return api.UnshelveWithOptions(ctx, "s1") }},
		{`{"unshelve":null}`, func() error { return api.UnshelveWithOptions(ctx, "s1", servers.WithUnshelveHost("")) }},
		{`{"unshelve":{"host":"host1"}}`, func() error { return api.UnshelveWithOptions(ctx, "s1", servers.WithUnshelveHost("host1")) }},
		{`{"unshelve":{"availability_zone":"az1"}}`, func() error { return api.UnshelveWithOptions(ctx, "s1", servers.WithUnshelveAvailabilityZone("az1")) }},
		{`{"unshelve":{"availability_zone":""}}`, func() error { return api.UnshelveWithOptions(ctx, "s1", servers.WithUnshelveAvailabilityZone("")) }},
		{`{"unshelve":{"availability_zone":null}}`, func() error {
			return api.UnshelveWithOptions(ctx, "s1", servers.WithUnshelveUnpinnedAvailabilityZone())
		}},
		{`{"unshelve":{"availability_zone":null,"host":"host1"}}`, func() error {
			return api.UnshelveWithOptions(ctx, "s1", servers.WithUnshelveUnpinnedAvailabilityZone(), servers.WithUnshelveHost("host1"))
		}},
		{`{"unshelve":null}`, func() error {
			return api.UnshelveWithOptions(ctx, "s1", servers.WithUnshelveUnpinnedAvailabilityZone(), servers.WithUnshelveDefaultAvailabilityZone())
		}},
	}
	for i, step := range steps {
		if err := step.call(); err != nil {
			t.Fatal(step.body, err)
		}
		want := pythonServerCall{http.MethodPost, pythonServerBase + "/action", "", step.body, "2.96"}
		if i == 0 {
			want.method = http.MethodDelete
			want.path = pythonServerBase + "/os-server-password"
		}
		if calls.Len() != i+1 || (pythonServerCalls(calls))[i] != want {
			t.Fatalf("step %d: %+v", i, pythonServerCalls(calls))
		}
	}
}

func TestPythonUserActionsDirectIPsNeedNoInventory(t *testing.T) {
	ctx := context.Background()
	client, _, calls := pythonServerCloud(t, func(*http.Request) *http.Response { return nativeServerWire(202, "") })
	// These direct actions remain through 2.43, although os-floating-ips
	// inventory routes were removed at 2.36.
	client.Microversion = "2.43"
	api := servers.New(client)
	steps := []struct {
		body string
		call func() error
	}{
		{`{"addFixedIp":{"networkId":"network1"}}`, func() error { return api.AddFixedIP(ctx, "s1", "network1") }},
		{`{"removeFixedIp":{"address":"10.0.0.1"}}`, func() error { return api.RemoveFixedIP(ctx, "s1", "10.0.0.1") }},
		{`{"addFloatingIp":{"address":"203.0.113.1"}}`, func() error { return api.AddFloatingIP(ctx, "s1", "203.0.113.1") }},
		{`{"addFloatingIp":{"address":"203.0.113.1","fixed_address":"10.0.0.1"}}`, func() error {
			return api.AddFloatingIP(ctx, "s1", "203.0.113.1", servers.WithFloatingIPFixedAddress("10.0.0.1"))
		}},
		{`{"addFloatingIp":{"address":"203.0.113.1","fixed_address":""}}`, func() error {
			return api.AddFloatingIP(ctx, "s1", "203.0.113.1", servers.WithFloatingIPFixedAddress(""))
		}},
		{`{"addFloatingIp":{"address":"203.0.113.1"}}`, func() error {
			return api.AddFloatingIP(ctx, "s1", "203.0.113.1", servers.WithFloatingIPFixedAddress("10.0.0.1"), servers.WithoutFloatingIPFixedAddress())
		}},
		{`{"removeFloatingIp":{"address":"203.0.113.1"}}`, func() error { return api.RemoveFloatingIP(ctx, "s1", "203.0.113.1") }},
		{`{"addFixedIp":{"networkId":""}}`, func() error { return api.AddFixedIP(ctx, "s1", "") }},
		{`{"removeFixedIp":{"address":""}}`, func() error { return api.RemoveFixedIP(ctx, "s1", "") }},
		{`{"addFloatingIp":{"address":""}}`, func() error { return api.AddFloatingIP(ctx, "s1", "") }},
		{`{"removeFloatingIp":{"address":""}}`, func() error { return api.RemoveFloatingIP(ctx, "s1", "") }},
	}
	for i, step := range steps {
		if err := step.call(); err != nil {
			t.Fatal(step.body, err)
		}
		want := pythonServerCall{http.MethodPost, pythonServerBase + "/action", "", step.body, "2.43"}
		if calls.Len() != i+1 || (pythonServerCalls(calls))[i] != want {
			t.Fatal(i, pythonServerCalls(calls))
		}
	}
}

func TestPythonUserActionsVersionBoundaries(t *testing.T) {
	cases := []struct {
		name, version string
		unsupported   bool
		call          func(*servers.API) error
	}{
		{"lock reason before", "2.72", true, func(a *servers.API) error {
			return a.LockWithOptions(context.Background(), "s1", servers.WithLockedReason(""))
		}},
		{"lock reason supported", "2.73", false, func(a *servers.API) error {
			return a.LockWithOptions(context.Background(), "s1", servers.WithLockedReason(""))
		}},
		{"lock omitted default", "", false, func(a *servers.API) error { return a.LockWithOptions(context.Background(), "s1") }},
		{"zone before", "2.76", true, func(a *servers.API) error {
			return a.UnshelveWithOptions(context.Background(), "s1", servers.WithUnshelveAvailabilityZone("az"))
		}},
		{"zone supported", "2.77", false, func(a *servers.API) error {
			return a.UnshelveWithOptions(context.Background(), "s1", servers.WithUnshelveAvailabilityZone("az"))
		}},
		{"host before", "2.90", true, func(a *servers.API) error {
			return a.UnshelveWithOptions(context.Background(), "s1", servers.WithUnshelveHost("host"))
		}},
		{"host supported", "2.91", false, func(a *servers.API) error {
			return a.UnshelveWithOptions(context.Background(), "s1", servers.WithUnshelveHost("host"))
		}},
		{"null zone before", "2.90", true, func(a *servers.API) error {
			return a.UnshelveWithOptions(context.Background(), "s1", servers.WithUnshelveUnpinnedAvailabilityZone())
		}},
		{"null zone supported", "2.91", false, func(a *servers.API) error {
			return a.UnshelveWithOptions(context.Background(), "s1", servers.WithUnshelveUnpinnedAvailabilityZone())
		}},
		{"add fixed removed", "2.44", true, func(a *servers.API) error { return a.AddFixedIP(context.Background(), "s1", "net") }},
		{"remove fixed removed", "2.44", true, func(a *servers.API) error { return a.RemoveFixedIP(context.Background(), "s1", "10.0.0.1") }},
		{"add floating removed", "2.44", true, func(a *servers.API) error { return a.AddFloatingIP(context.Background(), "s1", "203.0.113.1") }},
		{"remove floating removed", "2.44", true, func(a *servers.API) error { return a.RemoveFloatingIP(context.Background(), "s1", "203.0.113.1") }},
		{"floating latest removed", "latest", true, func(a *servers.API) error { return a.AddFloatingIP(context.Background(), "s1", "203.0.113.1") }},
		{"direct floating still available", "2.36", false, func(a *servers.API) error { return a.AddFloatingIP(context.Background(), "s1", "203.0.113.1") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, _, calls := pythonServerCloud(t, func(*http.Request) *http.Response { return nativeServerWire(202, "") })
			client.Microversion = tc.version
			err := tc.call(servers.New(client))
			if tc.unsupported {
				if !errors.Is(err, resource.ErrUnsupported) || calls.Len() != 0 {
					t.Fatal(err, pythonServerCalls(calls))
				}
			} else if err != nil || calls.Len() != 1 {
				t.Fatal(err, pythonServerCalls(calls))
			}
			if client.Microversion != tc.version {
				t.Fatal("operation changed selected version", client.Microversion)
			}
		})
	}
}

func TestPythonUserActionsExtensionsSnapshotAndPreflight(t *testing.T) {
	client, _, calls := pythonServerCloud(t, func(req *http.Request) *http.Response {
		if req.Header.Get("X-Test") != "value" {
			t.Error(req.Header)
		}
		return nativeServerWire(202, "")
	})
	client.Microversion = "2.43"
	api := servers.New(client)
	extension := map[string]any{"null": nil, "zero": 0, "false": false, "empty": []string{}}
	option := servers.WithFloatingIPServerField("x_extension", extension)
	extension["zero"] = 5
	if err := api.AddFloatingIP(context.Background(), "s1", "203.0.113.1", option, servers.WithFloatingIPServerHeader("X-Test", "value")); err != nil {
		t.Fatal(err)
	}
	want := `{"addFloatingIp":{"address":"203.0.113.1","x_extension":{"empty":[],"false":false,"null":null,"zero":0}}}`
	if calls.Len() != 1 || calls.Calls()[0].Body != want {
		t.Fatal(pythonServerCalls(calls))
	}
	before := calls.Len()
	cases := []func() error{
		func() error {
			return api.ClearPassword(context.Background(), "s1", servers.WithUserActionField("x", 1))
		},
		func() error { return api.Restore(context.Background(), "s1", nil) },
		func() error {
			return api.LockWithOptions(context.Background(), "s1", servers.WithLockServerField("locked_reason", nil))
		},
		func() error {
			return api.UnshelveWithOptions(context.Background(), "s1", servers.WithUnshelveServerField("availability_zone", nil))
		},
		func() error {
			return api.UnshelveWithOptions(context.Background(), "s1", servers.WithUnshelveServerField("host", "bad"))
		},
		func() error {
			return api.AddFixedIP(context.Background(), "s1", "net", servers.WithUserActionField("networkId", "bad"))
		},
		func() error {
			return api.RemoveFixedIP(context.Background(), "s1", "address", servers.WithUserActionField("address", "bad"))
		},
		func() error {
			return api.AddFloatingIP(context.Background(), "s1", "address", servers.WithFloatingIPServerField("fixed_address", "bad"))
		},
		func() error {
			return api.AddFloatingIP(context.Background(), "s1", "address", servers.WithFloatingIPServerHeader("X-Auth-Token", "bad"))
		},
		func() error {
			return api.RemoveFloatingIP(context.Background(), "s1", "address", request.WithQuery[servers.UserActionOpts]("x", "1"))
		},
		func() error {
			return api.AddFloatingIP(context.Background(), "s1", "address", request.WithArgument[servers.FloatingIPServerOpts]("x", 1))
		},
		func() error { return api.Restore(nil, "s1") },
		func() error { return api.Restore(context.Background(), "..") },
	}
	for i, call := range cases {
		if err := call(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(i, err)
		}
	}
	if calls.Len() != before {
		t.Fatal(pythonServerCalls(calls))
	}
}

func TestPythonUserActionsOptionsCaptureSourceBeforeCallbacks(t *testing.T) {
	client, _, calls := pythonServerCloud(t, func(*http.Request) *http.Response { return nativeServerWire(202, "") })
	api := servers.New(client)
	err := api.LockWithOptions(context.Background(), "s1", func(c *request.Config[servers.LockServerOpts]) error {
		client.ResourceBase = "https://elsewhere.invalid/v2.1/"
		return nil
	})
	if !errors.Is(err, resource.ErrInvalidOption) || calls.Len() != 0 {
		t.Fatal(err, pythonServerCalls(calls))
	}
	// Nil APIs and cancelled contexts fail before user option invocation.
	var nilAPI *servers.API
	optionCalls := 0
	option := func(c *request.Config[servers.LockServerOpts]) error { optionCalls++; return nil }
	if err := nilAPI.LockWithOptions(context.Background(), "s1", option); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	cause := errors.New("cancel reason")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if err := api.LockWithOptions(ctx, "s1", option); !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal(err)
	}
	if optionCalls != 0 {
		t.Fatal(optionCalls)
	}
}

func TestPythonUserActionsFinalStatusesAndMissingErrors(t *testing.T) {
	actions := map[string]func(*servers.API) error{
		"ClearPassword":       func(a *servers.API) error { return a.ClearPassword(context.Background(), "s1") },
		"Restore":             func(a *servers.API) error { return a.Restore(context.Background(), "s1") },
		"LockWithOptions":     func(a *servers.API) error { return a.LockWithOptions(context.Background(), "s1") },
		"UnshelveWithOptions": func(a *servers.API) error { return a.UnshelveWithOptions(context.Background(), "s1") },
		"AddFixedIP":          func(a *servers.API) error { return a.AddFixedIP(context.Background(), "s1", "net") },
		"RemoveFixedIP":       func(a *servers.API) error { return a.RemoveFixedIP(context.Background(), "s1", "addr") },
		"AddFloatingIP":       func(a *servers.API) error { return a.AddFloatingIP(context.Background(), "s1", "addr") },
		"RemoveFloatingIP":    func(a *servers.API) error { return a.RemoveFloatingIP(context.Background(), "s1", "addr") },
	}
	for name, call := range actions {
		for _, status := range []int{200, 202, 204, 302, 399, 400, 404, 500} {
			t.Run(fmt.Sprintf("%s/%d", name, status), func(t *testing.T) {
				client, _, calls := pythonServerCloud(t, func(*http.Request) *http.Response { return nativeServerWire(status, `not JSON`) })
				client.Microversion = "2.43"
				err := call(servers.New(client))
				if calls.Len() != 1 {
					t.Fatal(pythonServerCalls(calls))
				}
				if status < 400 {
					if err != nil {
						t.Fatal(err)
					}
					return
				}
				var operation *resource.OperationError
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &operation) || operation.Operation != name || operation.Resource != "servers" || !errors.As(err, &native) || native.Actual != status || string(native.Body) != `not JSON` || native.ResponseHeader.Get("X-Server-Proof") != "actual" {
					t.Fatal(err, operation, native)
				}
			})
		}
	}
}

type userActionFaultBody struct {
	reader                io.Reader
	readErr, closeErr     error
	afterRead, afterClose func()
}

func (b *userActionFaultBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	if b.afterRead != nil {
		b.afterRead()
		b.afterRead = nil
	}
	if b.readErr != nil {
		err = b.readErr
		b.readErr = nil
	}
	return n, err
}
func (b *userActionFaultBody) Close() error {
	if b.afterClose != nil {
		b.afterClose()
	}
	return b.closeErr
}

func TestPythonUserActionsPhysicalReadCloseCancellationAndSourceFaults(t *testing.T) {
	for _, status := range []int{202, 404} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := nativeServerClient(cloud)
			api := servers.New(client)
			readErr := errors.New("physical read")
			closeErr := errors.New("physical close")
			cause := errors.New("physical cancel")
			ctx, cancel := context.WithCancelCause(context.Background())
			count := 0
			original := client.ResourceBase
			cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
				count++
				response := nativeServerWire(status, "")
				response.Body = &userActionFaultBody{reader: strings.NewReader(`partial-wire`), readErr: readErr, closeErr: closeErr, afterRead: func() { client.ResourceBase = "https://bad.invalid/"; cancel(cause) }, afterClose: func() { client.ResourceBase = original }}
				return response, nil
			}))
			err := api.Restore(ctx, "s1")
			for _, target := range []error{readErr, closeErr, context.Canceled, cause, resource.ErrInvalidOption} {
				if !errors.Is(err, target) {
					t.Fatal(target, err)
				}
			}
			if count != 1 {
				t.Fatal(count)
			}
			if status < 400 {
				var receipt *resource.ResponseError
				if !errors.As(err, &receipt) || receipt.StatusCode != 202 || string(receipt.Body) != `partial-wire` || receipt.Header.Get("X-Server-Proof") != "actual" {
					t.Fatal(err, receipt)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != `partial-wire` {
					t.Fatal(err, native)
				}
			}
		})
	}
}

func TestPythonUserActionsRetryOwnershipAndCleanRetry(t *testing.T) {
	for _, tamper := range []bool{false, true} {
		t.Run(fmt.Sprint(tamper), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := nativeServerClient(cloud)
			count := 0
			cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
				count++
				if count == 1 {
					return nativeServerWire(503, "first-attempt"), nil
				}
				return nativeServerWire(202, ""), nil
			}))
			cloud.Provider.RetryFunc = func(ctx context.Context, method, url string, opts *gophercloud.RequestOpts, original error, count uint) error {
				if tamper {
					opts.JSONBody = map[string]any{"unlock": nil}
				}
				return nil
			}
			err := servers.New(client).LockWithOptions(context.Background(), "s1")
			if tamper {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || count != 1 {
					t.Fatal(err, native, count)
				}
			} else if err != nil || count != 2 {
				t.Fatal(err, count)
			}
		})
	}
	cloud := testcloud.New(t)
	client := nativeServerClient(cloud)
	cloud.SetTransport(testcloud.LinkRequest(func(*http.Request) (*http.Response, error) { return nativeServerWire(404, "gone"), nil }))
	retryErr := errors.New("retry denied")
	cloud.Provider.RetryFunc = func(ctx context.Context, method, url string, opts *gophercloud.RequestOpts, original error, count uint) error {
		return retryErr
	}
	err := servers.New(client).ClearPassword(context.Background(), "s1")
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.Is(err, retryErr) || !errors.As(err, &native) || native.Actual != 404 {
		t.Fatal(err, native)
	}
}

func TestPythonUserActionsLiteralEscapedServerID(t *testing.T) {
	var escaped string
	client, _, calls := pythonServerCloud(t, func(req *http.Request) *http.Response {
		escaped = req.URL.EscapedPath()
		return nativeServerWire(202, "")
	})
	if err := servers.New(client).Restore(context.Background(), "a/b%2Fz"); err != nil {
		t.Fatal(err)
	}
	if escaped != "/nova/v2.1/servers/a%2Fb%252Fz/action" || calls.Len() != 1 {
		t.Fatal(escaped, pythonServerCalls(calls))
	}
	if !reflect.DeepEqual(client.MoreHeaders, map[string]string(nil)) {
		t.Fatal(client.MoreHeaders)
	}
}
