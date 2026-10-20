package volumes_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func connectorFixture(t *testing.T, reply func(*http.Request) *http.Response) (*volumes.API, *[]pythonVolumeCall) {
	api, calls := pythonVolumeAPI(t, reply)
	api.RawClient().Microversion = "3.71"
	return api, calls
}
func connectorOperations() []struct {
	name, action string
	call         func(context.Context, *volumes.API, map[string]any, ...volumes.ConnectorOption) (*volumes.VolumeConnectorResult, error)
} {
	return []struct {
		name, action string
		call         func(context.Context, *volumes.API, map[string]any, ...volumes.ConnectorOption) (*volumes.VolumeConnectorResult, error)
	}{
		{"InitializeWithConnector", "os-initialize_connection", func(ctx context.Context, a *volumes.API, c map[string]any, o ...volumes.ConnectorOption) (*volumes.VolumeConnectorResult, error) {
			return a.InitializeWithConnector(ctx, "vol-1", c, o...)
		}},
		{"TerminateWithConnector", "os-terminate_connection", func(ctx context.Context, a *volumes.API, c map[string]any, o ...volumes.ConnectorOption) (*volumes.VolumeConnectorResult, error) {
			return a.TerminateWithConnector(ctx, "vol-1", c, o...)
		}},
	}
}
func connectorWrapped(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "volumes" {
		t.Fatalf("err=%v wrapper=%+v", err, wrapped)
	}
}

func TestPythonVolumeConnectorNestedInputAndOptionSnapshots(t *testing.T) {
	for _, operation := range connectorOperations() {
		t.Run(operation.name, func(t *testing.T) {
			connector := map[string]any{"wwnns": []string{"node-1", "node-2"}, "nqn": "nqn.example", "extensions": map[string]any{"false": false, "zero": 0, "null": nil, "empty": []any{}, "huge": json.RawMessage("900719925474099312345")}}
			original, _ := json.Marshal(connector)
			headers := map[string]string{"X-Connector": "before"}
			factory := volumes.WithConnectorHeaders(headers)
			headers["X-Connector"] = "after"
			var physical http.Header
			api, calls := connectorFixture(t, func(req *http.Request) *http.Response {
				physical = req.Header.Clone()
				response := pythonVolumeWire(203, `{"connection_info":{"driver_volume_type":"nvmeof","data":{"unknown":[false,0,null],"huge":900719925474099312345}},"unmodeled":true}`)
				response.Header.Set("X-Request-Id", "actual-action")
				return response
			})
			retained := map[string]string{"X-Snapshot": "owned"}
			result, err := operation.call(context.Background(), api, connector, factory, func(cfg *request.Config[volumes.ConnectorOpts]) error {
				connector["wwnns"].([]string)[0] = "changed"
				connector["extensions"].(map[string]any)["false"] = true
				cfg.Headers["X-Snapshot"] = retained["X-Snapshot"]
				return nil
			}, func(*request.Config[volumes.ConnectorOpts]) error { retained["X-Snapshot"] = "changed"; return nil })
			expected, _ := json.Marshal(map[string]any{operation.action: map[string]any{"connector": json.RawMessage(original)}})
			if err != nil || result == nil || result.Response == nil || result.Response.StatusCode != 203 || result.Response.Header.Get("X-Request-Id") != "actual-action" || result.Microversion != "3.71" || len(result.Discovery) != 0 || len(*calls) != 1 || (*calls)[0].method != "POST" || (*calls)[0].path != pythonVolumeBase+"/vol-1/action" || (*calls)[0].query != "" || (*calls)[0].body != string(expected) {
				t.Fatalf("result=%+v err=%v calls=%+v", result, err, *calls)
			}
			if physical.Get("X-Connector") != "before" || physical.Get("X-Snapshot") != "owned" || physical.Get("Openstack-Api-Version") != "volume 3.71" || physical.Get("X-Openstack-Volume-Api-Version") != "3.71" || physical.Get("X-Auth-Token") != "test-token" {
				t.Fatal(physical)
			}
			if api.RawClient().MoreHeaders != nil || api.RawClient().Microversion != "3.71" {
				t.Fatal("selected source mutated", api.RawClient())
			}
			if operation.name == "InitializeWithConnector" {
				if string(result.ConnectionInfo) != `{"driver_volume_type":"nvmeof","data":{"unknown":[false,0,null],"huge":900719925474099312345}}` {
					t.Fatal(string(result.ConnectionInfo))
				}
				result.ConnectionInfo[0] = 'X'
				if result.Response.Body[0] != '{' {
					t.Fatal("decoded value aliases response")
				}
			} else if result.ConnectionInfo != nil {
				t.Fatal(result.ConnectionInfo)
			}
		})
	}
}

func TestPythonVolumeConnectorNullEmptyAndRawConnectionInfo(t *testing.T) {
	for _, operation := range connectorOperations() {
		for _, connector := range []map[string]any{nil, {}, {"nil": nil, "empty": "", "false": false, "zero": 0}} {
			for _, info := range []string{`null`, `false`, `0`, `"text"`, `[]`, `{"extra":[null,false,0]}`} {
				api, calls := connectorFixture(t, func(*http.Request) *http.Response { return pythonVolumeWire(200, `{"connection_info":`+info+`}`) })
				result, err := operation.call(context.Background(), api, connector)
				raw, _ := json.Marshal(map[string]any{operation.action: map[string]any{"connector": connector}})
				if err != nil || result == nil || result.Response == nil || len(*calls) != 1 || (*calls)[0].body != string(raw) {
					t.Fatalf("result=%+v err=%v calls=%v", result, err, *calls)
				}
				if operation.name == "InitializeWithConnector" && string(result.ConnectionInfo) != info {
					t.Fatal(string(result.ConnectionInfo), info)
				}
				if operation.name == "TerminateWithConnector" && result.ConnectionInfo != nil {
					t.Fatal(result.ConnectionInfo)
				}
			}
		}
	}
}

func TestPythonVolumeConnectorResponseShapeAndSourceStatuses(t *testing.T) {
	for _, operation := range connectorOperations() {
		for _, status := range []int{200, 201, 202, 203, 204, 299, 300, 399, 400, 403, 404, 409, 500, 599} {
			t.Run(fmt.Sprintf("%s/status%d", operation.name, status), func(t *testing.T) {
				api, calls := connectorFixture(t, func(*http.Request) *http.Response { return pythonVolumeWire(status, `{"connection_info":null}`) })
				result, err := operation.call(context.Background(), api, map[string]any{})
				if len(*calls) != 1 {
					t.Fatal(*calls)
				}
				if status < 400 {
					if err != nil || result == nil || result.Response.StatusCode != status {
						t.Fatalf("result=%+v err=%v", result, err)
					}
					return
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != status || string(native.Body) != `{"connection_info":null}` {
					t.Fatal(err, native)
				}
				connectorWrapped(t, err, operation.name)
			})
		}
	}
	for _, body := range []string{"", `null`, `[]`, `false`, `{"other":true}`, `{broken`, string([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'})} {
		api, calls := connectorFixture(t, func(*http.Request) *http.Response { return pythonVolumeWire(202, body) })
		result, err := api.InitializeWithConnector(context.Background(), "vol-1", map[string]any{})
		var proof *resource.ResponseError
		if result == nil || result.ConnectionInfo != nil || result.Response == nil || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != body || len(*calls) != 1 {
			t.Fatalf("result=%+v err=%v proof=%+v calls=%v", result, err, proof, *calls)
		}
		api, calls = connectorFixture(t, func(*http.Request) *http.Response { return pythonVolumeWire(202, body) })
		result, err = api.TerminateWithConnector(context.Background(), "vol-1", map[string]any{})
		if err != nil || result == nil || string(result.Response.Body) != body || len(*calls) != 1 {
			t.Fatalf("opaque result=%+v err=%v", result, err)
		}
	}
}

type connectorMarshalFunc func() ([]byte, error)

func (f connectorMarshalFunc) MarshalJSON() ([]byte, error) { return f() }

func TestPythonVolumeConnectorCapturesSourceBeforeMarshalAndOptions(t *testing.T) {
	marshalCause := errors.New("connector MarshalJSON cause")
	cancelCause := errors.New("connector canceled")
	for _, operation := range connectorOperations() {
		for _, mode := range []string{"marshal source", "marshal cancel", "marshal cause", "option source", "option cancel", "invalid connector", "nil option", "header auth", "query forbidden", "field forbidden"} {
			t.Run(operation.name+"/"+mode, func(t *testing.T) {
				api, calls := connectorFixture(t, func(*http.Request) *http.Response { return pythonVolumeWire(200, `{"connection_info":null}`) })
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				connector := map[string]any{}
				var options []volumes.ConnectorOption
				switch mode {
				case "marshal source":
					connector["value"] = connectorMarshalFunc(func() ([]byte, error) { api.RawClient().ResourceBase += "changed/"; return []byte("null"), nil })
				case "marshal cancel":
					connector["value"] = connectorMarshalFunc(func() ([]byte, error) { cancel(cancelCause); return nil, marshalCause })
				case "marshal cause":
					connector["value"] = connectorMarshalFunc(func() ([]byte, error) { return nil, marshalCause })
				case "option source":
					options = []volumes.ConnectorOption{func(*request.Config[volumes.ConnectorOpts]) error {
						api.RawClient().Microversion = "3.70"
						return marshalCause
					}}
				case "option cancel":
					options = []volumes.ConnectorOption{func(*request.Config[volumes.ConnectorOpts]) error { cancel(cancelCause); return marshalCause }}
				case "invalid connector":
					connector["value"] = make(chan int)
				case "nil option":
					options = []volumes.ConnectorOption{nil}
				case "header auth":
					options = []volumes.ConnectorOption{volumes.WithConnectorHeader("X-Auth-Token", "caller")}
				case "query forbidden":
					options = []volumes.ConnectorOption{request.WithQuery[volumes.ConnectorOpts]("x", "y")}
				case "field forbidden":
					options = []volumes.ConnectorOption{request.WithField[volumes.ConnectorOpts]("x", true)}
				}
				_, err := operation.call(ctx, api, connector, options...)
				if err == nil || len(*calls) != 0 {
					t.Fatalf("err=%v calls=%v", err, *calls)
				}
				connectorWrapped(t, err, operation.name)
				if strings.Contains(mode, "cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) || !errors.Is(err, marshalCause)) {
					t.Fatal(err)
				}
				if mode == "marshal cause" && !errors.Is(err, marshalCause) {
					t.Fatal(err)
				}
				if mode == "option source" && (!errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, marshalCause)) {
					t.Fatal(err)
				}
			})
		}
	}
	for _, id := range []string{"", ".", "..", "a/b", "a?b", "a b"} {
		api, calls := connectorFixture(t, func(*http.Request) *http.Response { return pythonVolumeWire(200, `{"connection_info":null}`) })
		_, err := api.InitializeWithConnector(context.Background(), id, nil)
		if !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatalf("id=%q err=%v calls=%v", id, err, *calls)
		}
	}
	var nilAPI *volumes.API
	if _, err := nilAPI.InitializeWithConnector(context.Background(), "v", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

type connectorFaultBody struct {
	io.ReadCloser
	readErr, closeErr     error
	afterRead, afterClose func()
	closes                *atomic.Int32
}

func (b *connectorFaultBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if b.afterRead != nil {
		b.afterRead()
	}
	if b.readErr != nil {
		return n, b.readErr
	}
	return n, err
}
func (b *connectorFaultBody) Close() error {
	if b.closes != nil {
		b.closes.Add(1)
	}
	if b.afterClose != nil {
		b.afterClose()
	}
	return errors.Join(b.ReadCloser.Close(), b.closeErr)
}

func TestPythonVolumeConnectorPhysicalFaultsKeepReceiptsAndCauses(t *testing.T) {
	for _, operation := range connectorOperations() {
		for _, status := range []int{203, 403} {
			for _, mode := range []string{"read close", "cancel read", "sticky source"} {
				t.Run(fmt.Sprintf("%s/%d/%s", operation.name, status, mode), func(t *testing.T) {
					readCause, closeCause, cancelCause := io.ErrUnexpectedEOF, errors.New("connector Close cause"), errors.New("connector cancellation cause")
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					var closes atomic.Int32
					var api *volumes.API
					var original string
					api, calls := connectorFixture(t, func(*http.Request) *http.Response {
						response := pythonVolumeWire(status, `{"connection_info":{"actual":true}}`)
						response.Header.Set("X-Proof", "physical-action")
						body := &connectorFaultBody{ReadCloser: response.Body, closes: &closes}
						switch mode {
						case "read close":
							body.readErr, body.closeErr = readCause, closeCause
						case "cancel read":
							body.afterRead = func() { cancel(cancelCause) }
						case "sticky source":
							body.afterRead = func() { api.RawClient().ResourceBase = original + "changed/" }
							body.afterClose = func() { api.RawClient().ResourceBase = original }
						}
						response.Body = body
						return response
					})
					original = api.RawClient().ResourceBase
					hooks := 0
					api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
						hooks++
						return nil
					}
					result, err := operation.call(ctx, api, map[string]any{"null": nil})
					if err == nil || len(*calls) != 1 || closes.Load() != 1 || hooks > 1 || (status < 400 && hooks != 0) {
						t.Fatalf("result=%+v err=%v calls=%v closes=%d hooks=%d", result, err, *calls, closes.Load(), hooks)
					}
					connectorWrapped(t, err, operation.name)
					if mode == "read close" && (!errors.Is(err, readCause) || !errors.Is(err, closeCause)) {
						t.Fatal(err)
					}
					if mode == "cancel read" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
						t.Fatal(err)
					}
					if mode == "sticky source" && !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
					if status < 400 {
						var proof *resource.ResponseError
						if result == nil || result.ConnectionInfo != nil || result.Response == nil || !errors.As(err, &proof) || proof.StatusCode != status || proof.Header.Get("X-Proof") != "physical-action" || string(proof.Body) != `{"connection_info":{"actual":true}}` {
							t.Fatalf("result=%+v err=%v proof=%+v", result, err, proof)
						}
					} else {
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(err, &native) || native.Actual != status || native.ResponseHeader.Get("X-Proof") != "physical-action" || string(native.Body) != `{"connection_info":{"actual":true}}` {
							t.Fatal(err, native)
						}
					}
				})
			}
		}
	}
}

func TestPythonVolumeConnectorNativeRetryOwnsBodyAndVersion(t *testing.T) {
	for _, operation := range connectorOperations() {
		for _, mode := range []string{"clean", "body changed", "version changed"} {
			t.Run(operation.name+"/"+mode, func(t *testing.T) {
				var requests int
				api, calls := connectorFixture(t, func(*http.Request) *http.Response {
					requests++
					if requests == 1 {
						return pythonVolumeWire(503, `{"first":"rejected"}`)
					}
					return pythonVolumeWire(201, `{"connection_info":false}`)
				})
				hooks := 0
				api.RawClient().RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
					hooks++
					if mode == "body changed" {
						opts.JSONBody = map[string]any{"os-terminate_connection": nil}
					}
					if mode == "version changed" {
						opts.MoreHeaders["Openstack-Api-Version"] = "volume 3.70"
					}
					return nil
				}
				result, err := operation.call(context.Background(), api, map[string]any{"nqn": "owned"})
				if mode == "clean" {
					if err != nil || result == nil || result.Response.StatusCode != 201 || len(*calls) != 2 || hooks != 1 || (*calls)[0].body != (*calls)[1].body {
						t.Fatalf("result=%+v err=%v calls=%v hooks=%d", result, err, *calls, hooks)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || len(*calls) != 1 || hooks != 1 {
						t.Fatalf("result=%+v err=%v calls=%v hooks=%d", result, err, *calls, hooks)
					}
				}
			})
		}
	}
}

func TestPythonVolumeConnectorNegotiationPreservesActualDiscovery(t *testing.T) {
	var headers []http.Header
	api, calls := pythonVolumeAPI(t, func(req *http.Request) *http.Response {
		headers = append(headers, req.Header.Clone())
		if req.Method == "GET" {
			return pythonVolumeWire(200, `{"versions":[{"id":"v3.0","status":"CURRENT","version":"3.80","min_version":"3.0"}]}`)
		}
		return pythonVolumeWire(202, `{"connection_info":null}`)
	})
	result, err := api.InitializeWithConnector(context.Background(), "vol-1", nil)
	if err != nil || result == nil || result.Microversion != "3.71" || len(result.Discovery) != 1 || result.Discovery[0].StatusCode != 200 || len(*calls) != 2 || (*calls)[0].method != "GET" || (*calls)[1].method != "POST" || !reflect.DeepEqual([]string{headers[0].Get("Openstack-Api-Version"), headers[1].Get("Openstack-Api-Version")}, []string{"", "volume 3.71"}) {
		t.Fatalf("result=%+v err=%v calls=%v headers=%v", result, err, *calls, headers)
	}
	if api.RawClient().Microversion != "" {
		t.Fatal("negotiation mutated shared client")
	}
	before := bytes.Clone(result.Response.Body)
	result.Discovery[0].Body[0] = 'X'
	if !bytes.Equal(before, result.Response.Body) {
		t.Fatal("discovery aliases action")
	}
}
