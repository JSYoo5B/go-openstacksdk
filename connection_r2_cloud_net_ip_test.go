package openstack_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type r2CloudNetCall struct{ method, path, query, body string }

func (c r2CloudNetCall) key() string { return c.method + " " + c.path }

type r2CloudNetFixture struct {
	mu      sync.Mutex
	calls   []r2CloudNetCall
	handle  func(r2CloudNetCall) (int, string)
	network bool
}

func (f *r2CloudNetFixture) keys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	keys := make([]string, len(f.calls))
	for i, call := range f.calls {
		keys[i] = call.key()
	}
	return keys
}

func (f *r2CloudNetFixture) find(key string) (r2CloudNetCall, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, call := range f.calls {
		if call.key() == key {
			return call, true
		}
	}
	return r2CloudNetCall{}, false
}

func (f *r2CloudNetFixture) count(key string) int {
	n := 0
	for _, k := range f.keys() {
		if k == key {
			n++
		}
	}
	return n
}

// newR2CloudNetFixture serves Neutron under /network/v2.0 and Nova under
// /compute for the named floating IP source. Unhandled requests fail the test.
func newR2CloudNetFixture(t *testing.T, source compute.FloatingIPSource, handle func(r2CloudNetCall) (int, string)) (*sdk.Connection, *r2CloudNetFixture) {
	t.Helper()
	cloud := testcloud.New(t)
	f := &r2CloudNetFixture{handle: handle}
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		switch opts.Type {
		case "network":
			return cloud.Server.URL + "/network/", nil
		case "compute":
			return cloud.Server.URL + "/compute/", nil
		}
		t.Error("unexpected service", opts.Type)
		return "", errors.New("unexpected service")
	}
	cloud.Provider.HTTPClient.Transport = serverWorkflowTransport(func(r *http.Request) (*http.Response, error) {
		var raw []byte
		if r.Body != nil {
			raw, _ = io.ReadAll(r.Body)
		}
		call := r2CloudNetCall{method: r.Method, path: r.URL.Path, query: r.URL.RawQuery, body: string(raw)}
		f.mu.Lock()
		f.calls = append(f.calls, call)
		f.mu.Unlock()
		code, body := f.handle(call)
		if code == 0 {
			t.Errorf("unexpected request %s?%s %s", call.key(), call.query, call.body)
			return nil, errors.New("unexpected request")
		}
		return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.Compute, "2.35"), sdk.WithServerAddressPolicy(compute.WithFloatingIPSource(source)))
	if err != nil {
		t.Fatal(err)
	}
	return conn, f
}

const r2CloudNetBoundIP = `{"id":"fip-1","floating_network_id":"external","floating_ip_address":"198.51.100.20","port_id":"port-1","fixed_ip_address":"10.0.0.8","status":"ACTIVE"}`

func r2CloudNetNeutronDetachHandler(list string, put int, putBody string) func(r2CloudNetCall) (int, string) {
	return func(c r2CloudNetCall) (int, string) {
		switch c.key() {
		case "GET /network/v2.0/floatingips":
			return 200, `{"floatingips":[` + list + `]}`
		case "PUT /network/v2.0/floatingips/fip-1":
			return put, putBody
		}
		return 0, ""
	}
}

func TestR2CloudNetDetachNeutronSendsOneNullPortUpdate(t *testing.T) {
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNeutron, r2CloudNetNeutronDetachHandler(r2CloudNetBoundIP, 200, `{"floatingip":{"id":"fip-1","port_id":null}}`))
	result, err := conn.DetachIPFromServer(context.Background(), compute.DetachFloatingIPRequest{ServerID: "srv-1", FloatingIPID: "fip-1"})
	if err != nil || result == nil || !result.Detached || result.Outcome != compute.DetachedNeutron || result.Backend != compute.FloatingIPNeutron || result.Response == nil || result.Response.StatusCode != 200 || result.Lookup == nil || result.Lookup.FloatingIP == nil {
		t.Fatal(result, err)
	}
	put, _ := f.find("PUT /network/v2.0/floatingips/fip-1")
	if put.body != `{"floatingip":{"port_id":null}}` || len(f.keys()) != 2 {
		t.Fatal(put, f.keys())
	}
}

func TestR2CloudNetDetachNeutronReturnsFalseWithoutMutation(t *testing.T) {
	for name, list := range map[string]string{
		"absent":     `{"id":"other","port_id":"p","floating_ip_address":"198.51.100.30"}`,
		"null port":  `{"id":"fip-1","port_id":null,"floating_ip_address":"198.51.100.20"}`,
		"empty port": `{"id":"fip-1","port_id":"","floating_ip_address":"198.51.100.20"}`,
	} {
		t.Run(name, func(t *testing.T) {
			conn, f := newR2CloudNetFixture(t, compute.FloatingIPNeutron, r2CloudNetNeutronDetachHandler(list, 200, `{}`))
			result, err := conn.DetachIPFromServer(context.Background(), compute.DetachFloatingIPRequest{ServerID: "srv-1", FloatingIPID: "fip-1"})
			if err != nil || result == nil || result.Detached || result.Outcome != compute.DetachNotAttached || result.Response != nil || f.count("PUT /network/v2.0/floatingips/fip-1") != 0 {
				t.Fatal(result, err, f.keys())
			}
		})
	}
}

func TestR2CloudNetDetachNeutronUpdateFailureNeverFallsBackToNova(t *testing.T) {
	for _, status := range []int{404, 409, 500} {
		conn, f := newR2CloudNetFixture(t, compute.FloatingIPNeutron, r2CloudNetNeutronDetachHandler(r2CloudNetBoundIP, status, `{"NeutronError":{"message":"nope"}}`))
		result, err := conn.DetachIPFromServer(context.Background(), compute.DetachFloatingIPRequest{ServerID: "srv-1", FloatingIPID: "fip-1"})
		var detach *compute.DetachFloatingIPError
		var proof *resource.ResponseError
		if !errors.As(err, &detach) || detach.FloatingIPID != "fip-1" || detach.ServerID != "srv-1" || result == nil || result.Detached || result.Failure == nil || result.Failure.StatusCode != status || !strings.Contains(string(result.Failure.Envelope), "nope") {
			t.Fatal(status, result, err, proof)
		}
		for _, key := range f.keys() {
			if strings.HasPrefix(key, "POST /compute") {
				t.Fatal("Nova fallback after Neutron failure", f.keys())
			}
		}
		if f.count("PUT /network/v2.0/floatingips/fip-1") != 1 {
			t.Fatal("mutation was replayed", f.keys())
		}
	}
}

func TestR2CloudNetDetachNovaSendsRemoveFloatingIPAction(t *testing.T) {
	handle := func(c r2CloudNetCall) (int, string) {
		switch c.key() {
		case "GET /compute/os-floating-ips":
			return 200, `{"floating_ips":[{"id":"12","ip":"198.51.100.21","pool":"public","instance_id":"srv-1","fixed_ip":"10.0.0.8"}]}`
		case "POST /compute/servers/srv-1/action":
			return 202, ``
		}
		return 0, ""
	}
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNova, handle)
	result, err := conn.DetachIPFromServer(context.Background(), compute.DetachFloatingIPRequest{ServerID: "srv-1", FloatingIPID: "12"})
	if err != nil || result == nil || result.Outcome != compute.DetachNovaActionAccepted || result.Detached || result.Backend != compute.FloatingIPNova || result.Response == nil || result.Response.StatusCode != 202 {
		t.Fatal(result, err)
	}
	post, _ := f.find("POST /compute/servers/srv-1/action")
	if post.body != `{"removeFloatingIp":{"address":"198.51.100.21"}}` {
		t.Fatal(post)
	}
}

func TestR2CloudNetDetachNovaMissingIPFailsBeforeAction(t *testing.T) {
	handle := func(c r2CloudNetCall) (int, string) {
		if c.key() == "GET /compute/os-floating-ips" {
			return 200, `{"floating_ips":[{"id":"99","ip":"198.51.100.22","pool":"public","instance_id":null}]}`
		}
		return 0, ""
	}
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNova, handle)
	result, err := conn.DetachIPFromServer(context.Background(), compute.DetachFloatingIPRequest{ServerID: "srv-1", FloatingIPID: "12"})
	if !errors.Is(err, resource.ErrInvalidOption) || result == nil || result.Outcome != "" || len(f.keys()) != 1 {
		t.Fatal(result, err, f.keys())
	}
}

func TestR2CloudNetDetachNovaActionRejectionIsReportedOnce(t *testing.T) {
	handle := func(c r2CloudNetCall) (int, string) {
		switch c.key() {
		case "GET /compute/os-floating-ips":
			return 200, `{"floating_ips":[{"id":"12","ip":"198.51.100.21","pool":"public","instance_id":"srv-1"}]}`
		case "POST /compute/servers/srv-1/action":
			return 409, `{"conflictingRequest":{"message":"busy"}}`
		}
		return 0, ""
	}
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNova, handle)
	result, err := conn.DetachIPFromServer(context.Background(), compute.DetachFloatingIPRequest{ServerID: "srv-1", FloatingIPID: "12"})
	var detach *compute.DetachFloatingIPError
	if !errors.As(err, &detach) || result == nil || result.Failure == nil || result.Failure.StatusCode != 409 || f.count("POST /compute/servers/srv-1/action") != 1 {
		t.Fatal(result, err, f.keys())
	}
}

func TestR2CloudNetDetachValidatesInputsAndContextBeforeHTTP(t *testing.T) {
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNeutron, func(r2CloudNetCall) (int, string) { return 0, "" })
	for _, input := range []compute.DetachFloatingIPRequest{{FloatingIPID: "fip-1"}, {ServerID: "srv-1"}} {
		if _, err := conn.DetachIPFromServer(context.Background(), input); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(input, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := conn.DetachIPFromServer(ctx, compute.DetachFloatingIPRequest{ServerID: "srv-1", FloatingIPID: "fip-1"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	var nilConn *sdk.Connection
	if _, err := nilConn.DetachIPFromServer(context.Background(), compute.DetachFloatingIPRequest{}); err == nil {
		t.Fatal("nil connection")
	}
	if len(f.keys()) != 0 {
		t.Fatal(f.keys())
	}
}

func TestR2CloudNetDetachCancellationDuringUpdateDoesNotReplay(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handle := func(c r2CloudNetCall) (int, string) {
		switch c.key() {
		case "GET /network/v2.0/floatingips":
			return 200, `{"floatingips":[` + r2CloudNetBoundIP + `]}`
		case "PUT /network/v2.0/floatingips/fip-1":
			cancel()
			return 200, `{"floatingip":{"id":"fip-1","port_id":null}}`
		}
		return 0, ""
	}
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNeutron, handle)
	result, err := conn.DetachIPFromServer(ctx, compute.DetachFloatingIPRequest{ServerID: "srv-1", FloatingIPID: "fip-1"})
	if err == nil || result == nil || f.count("PUT /network/v2.0/floatingips/fip-1") != 1 {
		t.Fatal(result, err, f.keys())
	}
	_ = time.Second
}

func r2CloudNetNovaAutoHandler(attached *bool, serverBody func() string) func(r2CloudNetCall) (int, string) {
	return func(c r2CloudNetCall) (int, string) {
		switch c.key() {
		case "GET /network/v2.0/networks":
			return 200, `{"networks":[{"id":"private","name":"private"}]}`
		case "GET /network/v2.0/subnets":
			return 200, `{"subnets":[]}`
		case "GET /compute/os-floating-ip-pools":
			return 200, `{"floating_ip_pools":[{"name":"public"}]}`
		case "GET /compute/os-floating-ips":
			inst := "null"
			if *attached {
				inst = `"srv-1"`
			}
			return 200, `{"floating_ips":[{"id":"12","ip":"198.51.100.21","pool":"public","instance_id":` + inst + `}]}`
		case "GET /compute/os-floating-ips/12":
			return 200, `{"floating_ip":{"id":"12","ip":"198.51.100.21","pool":"public","instance_id":null}}`
		case "POST /compute/servers/srv-1/action":
			*attached = true
			return 202, ``
		case "GET /compute/servers/srv-1":
			return 200, serverBody()
		}
		return 0, ""
	}
}

func TestR2CloudNetAddAutoIPNovaReuseAttachesAndWaits(t *testing.T) {
	attached := false
	body := func() string {
		return `{"server":{"id":"srv-1","status":"ACTIVE","addresses":{"private":[{"version":4,"addr":"10.0.0.8","OS-EXT-IPS:type":"fixed"},{"version":4,"addr":"198.51.100.21","OS-EXT-IPS:type":"floating"}]}}}`
	}
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNova, r2CloudNetNovaAutoHandler(&attached, body))
	server := &compute.Server{ID: "srv-1", Status: "ACTIVE"}
	result, err := conn.AddAutoIP(context.Background(), server, compute.WithAddAutoIPWait(true), compute.WithAddAutoIPPollInterval(time.Millisecond))
	if err != nil {
		t.Fatal(err, f.keys())
	}
	if !result.Reused || result.Created || result.FloatingIPAddr != "198.51.100.21" || !result.Observed || result.AttachSkipped || result.Backend != compute.FloatingIPNova {
		t.Fatal(result, f.keys())
	}
	post, _ := f.find("POST /compute/servers/srv-1/action")
	if !strings.Contains(post.body, `"addFloatingIp"`) || !strings.Contains(post.body, "198.51.100.21") || f.count("POST /compute/servers/srv-1/action") != 1 {
		t.Fatal(post, f.keys())
	}
	if result.InterfaceIP != "198.51.100.21" {
		t.Fatal("interface ip", result.InterfaceIP)
	}
	if server.Addresses != nil {
		t.Fatal("caller server mutated")
	}
}

func TestR2CloudNetAddAutoIPAlreadyAttachedSkipsAttach(t *testing.T) {
	attached := false
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNova, r2CloudNetNovaAutoHandler(&attached, func() string { return `{}` }))
	server := &compute.Server{ID: "srv-1", Status: "ACTIVE", Addresses: map[string]any{"private": []any{map[string]any{"version": 4, "addr": "198.51.100.21", "OS-EXT-IPS:type": "floating"}}}}
	result, err := conn.AddAutoIP(context.Background(), server, compute.WithAddAutoIPWait(true))
	if err != nil || !result.AlreadyAttached || f.count("POST /compute/servers/srv-1/action") != 0 || f.count("GET /compute/servers/srv-1") != 0 {
		t.Fatal(result, err, f.keys())
	}
}

func TestR2CloudNetAddAutoIPWaitTimeoutOnNovaReportsDeadlineWithoutCleanup(t *testing.T) {
	attached := false
	body := func() string {
		return `{"server":{"id":"srv-1","status":"ACTIVE","addresses":{"private":[{"version":4,"addr":"10.0.0.8","OS-EXT-IPS:type":"fixed"}]}}}`
	}
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNova, r2CloudNetNovaAutoHandler(&attached, body))
	result, err := conn.AddAutoIP(context.Background(), &compute.Server{ID: "srv-1"}, compute.WithAddAutoIPWait(true), compute.WithAddAutoIPTimeout(60*time.Millisecond), compute.WithAddAutoIPPollInterval(5*time.Millisecond))
	var timeout *compute.AddAutoIPTimeoutError
	if !errors.As(err, &timeout) || !errors.Is(err, context.DeadlineExceeded) || result.Observed || result.Cleanup != nil {
		t.Fatal(result, err)
	}
	for _, key := range f.keys() {
		if strings.HasPrefix(key, "DELETE") {
			t.Fatal("unexpected delete", f.keys())
		}
	}
	if f.count("POST /compute/servers/srv-1/action") != 1 {
		t.Fatal(f.keys())
	}
}

func TestR2CloudNetAddAutoIPValidation(t *testing.T) {
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNova, func(r2CloudNetCall) (int, string) { return 0, "" })
	if _, err := conn.AddAutoIP(context.Background(), nil); err == nil {
		t.Fatal("nil server")
	}
	if _, err := conn.AddAutoIP(context.Background(), &compute.Server{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := conn.AddAutoIP(context.Background(), &compute.Server{ID: "s"}, nil); err == nil {
		t.Fatal("nil option")
	}
	if _, err := conn.AddAutoIP(context.Background(), &compute.Server{ID: "s"}, compute.WithAddAutoIPPollInterval(0)); err == nil {
		t.Fatal("interval")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := conn.AddAutoIP(ctx, &compute.Server{ID: "s"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	var nilConn *sdk.Connection
	if _, err := nilConn.AddAutoIP(context.Background(), &compute.Server{ID: "s"}); err == nil {
		t.Fatal("nil connection")
	}
	if len(f.keys()) != 0 {
		t.Fatal(f.keys())
	}
}

func r2CloudNetNeutronCreateHandler(serverShowsIP *bool, deleted *bool) func(r2CloudNetCall) (int, string) {
	gets := 0
	return func(c r2CloudNetCall) (int, string) {
		switch c.key() {
		case "GET /network/v2.0/networks":
			return 200, `{"networks":[{"id":"external","name":"public","router:external":true,"subnets":["sub"]},{"id":"private","name":"private"}]}`
		case "GET /network/v2.0/subnets":
			return 200, `{"subnets":[{"id":"sub","network_id":"external","ip_version":4}]}`
		case "GET /network/v2.0/ports":
			return 200, `{"ports":[{"id":"port-1","device_id":"srv-1","network_id":"private","fixed_ips":[{"ip_address":"10.0.0.8"}]}]}`
		case "POST /network/v2.0/floatingips":
			return 201, `{"floatingip":{"id":"new-1","floating_network_id":"external","floating_ip_address":"198.51.100.40","port_id":"port-1","fixed_ip_address":"10.0.0.8","status":"ACTIVE"}}`
		case "GET /network/v2.0/floatingips":
			if *deleted {
				return 200, `{"floatingips":[]}`
			}
			return 200, `{"floatingips":[{"id":"new-1","floating_network_id":"external","floating_ip_address":"198.51.100.40","port_id":"port-1","status":"ACTIVE"}]}`
		case "GET /network/v2.0/floatingips/new-1":
			if *deleted {
				return 404, `{"NeutronError":{"message":"gone"}}`
			}
			return 200, `{"floatingip":{"id":"new-1","floating_network_id":"external","floating_ip_address":"198.51.100.40","port_id":"port-1","status":"ACTIVE"}}`
		case "DELETE /network/v2.0/floatingips/new-1":
			*deleted = true
			return 204, ``
		case "GET /compute/servers/srv-1":
			gets++
			if *serverShowsIP && gets > 1 {
				return 200, `{"server":{"id":"srv-1","status":"ACTIVE","addresses":{"private":[{"version":4,"addr":"198.51.100.40","OS-EXT-IPS:type":"floating"}]}}}`
			}
			return 200, `{"server":{"id":"srv-1","status":"ACTIVE","addresses":{"private":[{"version":4,"addr":"10.0.0.8","OS-EXT-IPS:type":"fixed"}]}}}`
		}
		return 0, ""
	}
}

func TestR2CloudNetAddAutoIPNeutronCreateSkipsAttachAndWaits(t *testing.T) {
	show, deleted := true, false
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNeutron, r2CloudNetNeutronCreateHandler(&show, &deleted))
	result, err := conn.AddAutoIP(context.Background(), &compute.Server{ID: "srv-1"}, compute.WithAddAutoIPReuse(false), compute.WithAddAutoIPWait(true), compute.WithAddAutoIPPollInterval(time.Millisecond))
	if err != nil || !result.Created || !result.AttachSkipped || !result.Observed || result.FloatingIPAddr != "198.51.100.40" || result.Attach != nil {
		t.Fatal(result, err, f.keys())
	}
	if f.count("POST /network/v2.0/floatingips") != 1 || f.count("PUT /network/v2.0/floatingips/new-1") != 0 || f.count("DELETE /network/v2.0/floatingips/new-1") != 0 {
		t.Fatal(f.keys())
	}
}

func TestR2CloudNetAddAutoIPNeutronCreateTimeoutDeletesOnceUnlessDisabled(t *testing.T) {
	for _, cleanup := range []bool{true, false} {
		show, deleted := false, false
		conn, f := newR2CloudNetFixture(t, compute.FloatingIPNeutron, r2CloudNetNeutronCreateHandler(&show, &deleted))
		result, err := conn.AddAutoIP(context.Background(), &compute.Server{ID: "srv-1"}, compute.WithAddAutoIPReuse(false), compute.WithAddAutoIPWait(true), compute.WithAddAutoIPTimeout(80*time.Millisecond), compute.WithAddAutoIPPollInterval(5*time.Millisecond), compute.WithAddAutoIPCleanup(cleanup))
		var timeout *compute.AddAutoIPTimeoutError
		if !errors.As(err, &timeout) {
			t.Fatal(cleanup, result, err, f.keys())
		}
		wantDelete := 0
		if cleanup {
			wantDelete = 1
			if result.Cleanup == nil || !result.Cleanup.Deleted {
				t.Fatal(result.Cleanup, result.CleanupError)
			}
		}
		if f.count("DELETE /network/v2.0/floatingips/new-1") != wantDelete {
			t.Fatal(cleanup, f.keys())
		}
	}
}
