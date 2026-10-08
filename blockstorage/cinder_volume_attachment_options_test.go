package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type vdaoCall func(context.Context, *gophercloud.ServiceClient) (*blockstorage.VolumeActionResult, error)

func vdaoHTTP(t *testing.T, cloud *testcloud.Cloud, client *gophercloud.ServiceClient, want string, call vdaoCall) {
	t.Helper()
	var requests atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		raw, err := io.ReadAll(r.Body)
		if err != nil || string(raw) != want || r.Method != http.MethodPost || r.URL.EscapedPath() != vsaBase+"volumes/literal/action" || r.URL.RawQuery != "" || r.ContentLength != int64(len(want)) || len(r.TransferEncoding) != 0 || r.Header.Get("X-Source") != "original" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != "volume 3.90" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.90" {
			t.Error(r.Method, r.URL, string(raw), want, r.Header, err)
		}
		w.Header().Set("X-Option-Proof", "actual action")
		w.WriteHeader(203)
		_, _ = w.Write([]byte("opaque attachment acknowledgement"))
	})
	result, err := call(vsaContext(t), client)
	if err != nil || result == nil || !result.Completed || result.VolumeID != "literal" || result.Microversion != "3.90" || len(result.Discovery) != 0 || result.Applied == nil || result.Applied.StatusCode != 203 || result.Applied.Header.Get("X-Option-Proof") != "actual action" || string(result.Applied.Body) != "opaque attachment acknowledgement" || requests.Load() != 1 {
		t.Fatal(result, err, requests.Load())
	}
}

func TestCinderVolumeAttachmentOptionsFactoriesDefaultsAndReplacementOwnership(t *testing.T) {
	ctx := vsaContext(t)
	host, instance, yes := "owned host", "owned instance", true
	raw := json.RawMessage(`{"nested":[false,null,9007199254740993]}`)
	original := string(raw)
	connector := map[string]json.RawMessage{"payload": raw}
	attachFactory := blockstorage.WithCinderVolumeAttachOptions(blockstorage.CinderVolumeAttachOpts{Instance: &instance, HostName: &host})
	detachFactory := blockstorage.WithCinderVolumeDetachOptions(blockstorage.CinderVolumeDetachOpts{Force: &yes, Connector: connector})
	connectorFactory := blockstorage.WithCinderVolumeDetachConnector(connector)
	host, instance, yes = "caller changed host", "caller changed instance", false
	raw[0] = '['
	delete(connector, "payload")
	for i := 0; i < 2; i++ {
		attach, err := blockstorage.PrepareCinderVolumeAttachOptions(ctx, attachFactory)
		if err != nil || attach.Instance == nil || *attach.Instance != "owned instance" || attach.HostName == nil || *attach.HostName != "owned host" {
			t.Fatal(attach, err)
		}
		detach, err := blockstorage.PrepareCinderVolumeDetachOptions(ctx, detachFactory)
		if err != nil || detach.Force == nil || !*detach.Force || string(detach.Connector["payload"]) != original {
			t.Fatal(detach, err)
		}
		fromConnector, err := blockstorage.PrepareCinderVolumeDetachOptions(ctx, connectorFactory, blockstorage.WithCinderVolumeDetachForce(true))
		if err != nil || string(fromConnector.Connector["payload"]) != original {
			t.Fatal(fromConnector, err)
		}
		*attach.Instance, *attach.HostName, *detach.Force = "returned instance", "returned host", false
		detach.Connector["payload"][0] = '['
		delete(detach.Connector, "payload")
		fromConnector.Connector["payload"][0] = '['
	}
	attach, err := blockstorage.PrepareCinderVolumeAttachOptions(ctx,
		blockstorage.WithCinderVolumeAttachInstance("discarded"),
		blockstorage.WithCinderVolumeAttachOptions(blockstorage.CinderVolumeAttachOpts{HostName: &host}),
		blockstorage.WithCinderVolumeAttachInstance(""))
	if err != nil || attach.Instance == nil || *attach.Instance != "" || attach.HostName == nil || *attach.HostName != "caller changed host" {
		t.Fatal(attach, err)
	}
	detach, err := blockstorage.PrepareCinderVolumeDetachOptions(ctx,
		blockstorage.WithCinderVolumeDetachForce(true), connectorFactory,
		blockstorage.WithCinderVolumeDetachOptions(blockstorage.CinderVolumeDetachOpts{}))
	if err != nil || detach.Force == nil || *detach.Force || detach.Connector != nil {
		t.Fatal("full replacement must remove connector and restore owned false default", detach, err)
	}
	empty, err := blockstorage.PrepareCinderVolumeDetachOptions(ctx)
	if err != nil || empty.Force == nil || *empty.Force {
		t.Fatal(empty, err)
	}
	*empty.Force = true
	empty, err = blockstorage.PrepareCinderVolumeDetachOptions(ctx)
	if err != nil || empty.Force == nil || *empty.Force {
		t.Fatal("default pointer borrowed a previous result", empty, err)
	}
}

func TestCinderVolumeAttachmentOriginalsOwnSlicePointersAndConnectorThroughHTTP(t *testing.T) {
	for _, family := range []string{"attach", "detach"} {
		t.Run(family, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.90")
			var order []int
			replaced := 0
			var call vdaoCall
			body := `{"os-attach":{"instance_uuid":"owned","mountpoint":""}}`
			if family == "attach" {
				instance := "owned"
				var first, second *blockstorage.CinderVolumeAttachOpts
				var options []blockstorage.CinderVolumeAttachOption
				options = []blockstorage.CinderVolumeAttachOption{
					func(next *blockstorage.CinderVolumeAttachOpts) error {
						order = append(order, 1)
						next.Instance = &instance
						first = next
						options[1] = func(*blockstorage.CinderVolumeAttachOpts) error {
							replaced++
							return errors.New("replacement must not run")
						}
						return nil
					},
					func(next *blockstorage.CinderVolumeAttachOpts) error {
						order = append(order, 2)
						instance = "caller changed"
						*first.Instance = "retained first changed"
						if next.Instance == nil || *next.Instance != "owned" {
							t.Fatal("borrowed first callback pointer", next)
						}
						second = next
						return nil
					},
					func(next *blockstorage.CinderVolumeAttachOpts) error {
						order = append(order, 3)
						*second.Instance = "retained second changed"
						if next.Instance == nil || *next.Instance != "owned" {
							t.Fatal("borrowed intermediate callback pointer", next)
						}
						return nil
					},
				}
				call = func(ctx context.Context, c *gophercloud.ServiceClient) (*blockstorage.VolumeActionResult, error) {
					return blockstorage.AttachCinderVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: "literal"}, "", options...)
				}
			} else {
				body = `{"os-force_detach":{"attachment_id":"","connector":{"payload":{"nested":[false,null,9007199254740993]}}}}`
				yes := true
				raw := json.RawMessage(`{"nested":[false,null,9007199254740993]}`)
				var first, second *blockstorage.CinderVolumeDetachOpts
				var options []blockstorage.CinderVolumeDetachOption
				options = []blockstorage.CinderVolumeDetachOption{
					func(next *blockstorage.CinderVolumeDetachOpts) error {
						order = append(order, 1)
						next.Force = &yes
						next.Connector = map[string]json.RawMessage{"payload": raw}
						first = next
						options[1] = func(*blockstorage.CinderVolumeDetachOpts) error {
							replaced++
							return errors.New("replacement must not run")
						}
						return nil
					},
					func(next *blockstorage.CinderVolumeDetachOpts) error {
						order = append(order, 2)
						yes = false
						*first.Force = false
						raw[0] = '['
						delete(first.Connector, "payload")
						if next.Force == nil || !*next.Force || string(next.Connector["payload"]) != `{"nested":[false,null,9007199254740993]}` {
							t.Fatal("borrowed first callback pointer/map/member", next)
						}
						second = next
						return nil
					},
					func(next *blockstorage.CinderVolumeDetachOpts) error {
						order = append(order, 3)
						*second.Force = false
						second.Connector["payload"][0] = '['
						delete(second.Connector, "payload")
						if next.Force == nil || !*next.Force || string(next.Connector["payload"]) != `{"nested":[false,null,9007199254740993]}` {
							t.Fatal("borrowed intermediate callback pointer/map/member", next)
						}
						return nil
					},
				}
				call = func(ctx context.Context, c *gophercloud.ServiceClient) (*blockstorage.VolumeActionResult, error) {
					return blockstorage.DetachCinderVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: "literal"}, "", options...)
				}
			}
			vdaoHTTP(t, cloud, client, body, call)
			if replaced != 0 || !reflect.DeepEqual(order, []int{1, 2, 3}) {
				t.Fatal(order, replaced)
			}
		})
	}
}

func TestCinderVolumeAttachmentPrepareValidatesFinalActiveValuesAndJoinsCauses(t *testing.T) {
	bad := string([]byte{0xff})
	attachCases := []struct {
		name    string
		options []blockstorage.CinderVolumeAttachOption
		valid   bool
	}{
		{"no selector", nil, false},
		{"empty instance ignores invalid host", []blockstorage.CinderVolumeAttachOption{blockstorage.WithCinderVolumeAttachHostName(bad), blockstorage.WithCinderVolumeAttachInstance("")}, true},
		{"empty host valid", []blockstorage.CinderVolumeAttachOption{blockstorage.WithCinderVolumeAttachHostName("")}, true},
		{"active host invalid", []blockstorage.CinderVolumeAttachOption{blockstorage.WithCinderVolumeAttachHostName(bad)}, false},
		{"active instance invalid despite host", []blockstorage.CinderVolumeAttachOption{blockstorage.WithCinderVolumeAttachInstance(bad), blockstorage.WithCinderVolumeAttachHostName("valid host")}, false},
		{"invalid instance replaced", []blockstorage.CinderVolumeAttachOption{blockstorage.WithCinderVolumeAttachInstance(bad), blockstorage.WithCinderVolumeAttachInstance("repaired")}, true},
		{"replacement removes both selectors", []blockstorage.CinderVolumeAttachOption{blockstorage.WithCinderVolumeAttachHostName("host"), blockstorage.WithCinderVolumeAttachOptions(blockstorage.CinderVolumeAttachOpts{})}, false},
	}
	for _, tc := range attachCases {
		t.Run("attach/"+tc.name, func(t *testing.T) {
			got, err := blockstorage.PrepareCinderVolumeAttachOptions(vsaContext(t), tc.options...)
			if tc.valid {
				if err != nil {
					t.Fatal(got, err)
				}
				return
			}
			if !errors.Is(err, resource.ErrInvalidOption) || got.Instance != nil || got.HostName != nil {
				t.Fatal(got, err)
			}
		})
	}
	badKey := map[string]json.RawMessage{bad: json.RawMessage(`true`)}
	malformed := map[string]json.RawMessage{"payload": json.RawMessage(`{]`)}
	badUTF8 := map[string]json.RawMessage{"payload": json.RawMessage([]byte{'"', 0xff, '"'})}
	detachCases := []struct {
		name    string
		options []blockstorage.CinderVolumeDetachOption
		valid   bool
	}{
		{"inactive invalid key and malformed member", []blockstorage.CinderVolumeDetachOption{blockstorage.WithCinderVolumeDetachConnector(map[string]json.RawMessage{bad: json.RawMessage(`{]`)})}, true},
		{"active empty object", []blockstorage.CinderVolumeDetachOption{blockstorage.WithCinderVolumeDetachForce(true), blockstorage.WithCinderVolumeDetachConnector(map[string]json.RawMessage{})}, true},
		{"active invalid key", []blockstorage.CinderVolumeDetachOption{blockstorage.WithCinderVolumeDetachForce(true), blockstorage.WithCinderVolumeDetachConnector(badKey)}, false},
		{"active malformed JSON", []blockstorage.CinderVolumeDetachOption{blockstorage.WithCinderVolumeDetachForce(true), blockstorage.WithCinderVolumeDetachConnector(malformed)}, false},
		{"active invalid UTF8", []blockstorage.CinderVolumeDetachOption{blockstorage.WithCinderVolumeDetachForce(true), blockstorage.WithCinderVolumeDetachConnector(badUTF8)}, false},
		{"bad connector replaced", []blockstorage.CinderVolumeDetachOption{blockstorage.WithCinderVolumeDetachForce(true), blockstorage.WithCinderVolumeDetachConnector(malformed), blockstorage.WithCinderVolumeDetachConnector(map[string]json.RawMessage{"payload": json.RawMessage(`null`)})}, true},
		{"later false makes connector inactive", []blockstorage.CinderVolumeDetachOption{blockstorage.WithCinderVolumeDetachForce(true), blockstorage.WithCinderVolumeDetachConnector(malformed), blockstorage.WithCinderVolumeDetachForce(false)}, true},
	}
	for _, tc := range detachCases {
		t.Run("detach/"+tc.name, func(t *testing.T) {
			got, err := blockstorage.PrepareCinderVolumeDetachOptions(vsaContext(t), tc.options...)
			if tc.valid {
				if err != nil || got.Force == nil {
					t.Fatal(got, err)
				}
				return
			}
			if !errors.Is(err, resource.ErrInvalidOption) || got.Force != nil || got.Connector != nil {
				t.Fatal(got, err)
			}
		})
	}
	for _, family := range []string{"attach", "detach"} {
		for _, kind := range []string{"nil context", "already canceled", "nil callback", "callback error", "callback cancel", "callback error and cancel"} {
			t.Run(family+"/"+kind, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(vsaContext(t))
				defer cancel(nil)
				var selected context.Context = ctx
				callbackCause, cancelCause := errors.New("attachment prepare callback"), errors.New("attachment prepare custom cancel")
				first, later, wantFirst := 0, 0, 1
				if kind == "nil context" {
					selected = nil
					wantFirst = 0
				}
				if kind == "already canceled" {
					cancel(cancelCause)
					wantFirst = 0
				}
				if kind == "nil callback" {
					wantFirst = 0
				}
				callback := func() error {
					first++
					if kind == "callback cancel" || kind == "callback error and cancel" {
						cancel(cancelCause)
					}
					if kind == "callback error" || kind == "callback error and cancel" {
						return callbackCause
					}
					return nil
				}
				var err error
				if family == "attach" {
					options := []blockstorage.CinderVolumeAttachOption{
						func(next *blockstorage.CinderVolumeAttachOpts) error {
							value := "instance"
							next.Instance = &value
							return callback()
						},
						func(*blockstorage.CinderVolumeAttachOpts) error { later++; return nil },
					}
					if kind == "nil callback" {
						options[0] = nil
					}
					got, cause := blockstorage.PrepareCinderVolumeAttachOptions(selected, options...)
					err = cause
					if got.Instance != nil || got.HostName != nil {
						t.Fatal("failed Prepare published attach policy", got, err)
					}
				} else {
					options := []blockstorage.CinderVolumeDetachOption{
						func(next *blockstorage.CinderVolumeDetachOpts) error {
							yes := true
							next.Force = &yes
							next.Connector = map[string]json.RawMessage{"payload": json.RawMessage(`null`)}
							return callback()
						},
						func(*blockstorage.CinderVolumeDetachOpts) error { later++; return nil },
					}
					if kind == "nil callback" {
						options[0] = nil
					}
					got, cause := blockstorage.PrepareCinderVolumeDetachOptions(selected, options...)
					err = cause
					if got.Force != nil || got.Connector != nil {
						t.Fatal("failed Prepare published detach policy", got, err)
					}
				}
				if err == nil || first != wantFirst || later != 0 {
					t.Fatal(err, first, later)
				}
				if (kind == "nil context" || kind == "nil callback") && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				if (kind == "already canceled" || kind == "callback cancel" || kind == "callback error and cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal("lost context identities", err)
				}
				if (kind == "callback error" || kind == "callback error and cancel") && !errors.Is(err, callbackCause) {
					t.Fatal("lost callback identity", err)
				}
			})
		}
	}
}

func TestCinderVolumeAttachmentDirectPreflightAndStickySourceStopOriginals(t *testing.T) {
	for _, family := range []string{"attach", "detach"} {
		for _, kind := range []string{"nil client", "wrong role before unsafe ID", "unsafe ID", "invalid required literal", "nil context", "already canceled", "source change", "source callback and cancellation"} {
			t.Run(family+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.90")
				original := *client
				ctx, cancel := context.WithCancelCause(vsaContext(t))
				defer cancel(nil)
				var selected context.Context = ctx
				id, literal := "literal", ""
				first, later := 0, 0
				var requests atomic.Int32
				callbackCause, cancelCause := errors.New("attachment direct callback"), errors.New("attachment direct custom cancel")
				want, wantFirst := resource.ErrInvalidOption, 0
				switch kind {
				case "nil client":
					client = nil
				case "wrong role before unsafe ID":
					client.Type = "compute"
					id = "a/b"
					want = resource.ErrUnsupported
				case "unsafe ID":
					id = "a/b"
				case "invalid required literal":
					literal = string([]byte{0xff})
				case "nil context":
					selected = nil
				case "already canceled":
					cancel(cancelCause)
					want = context.Canceled
				default:
					wantFirst = 1
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(500) })
				callback := func() error {
					first++
					client.Microversion = "3.91"
					if kind == "source callback and cancellation" {
						cancel(cancelCause)
						return callbackCause
					}
					return nil
				}
				var result *blockstorage.VolumeActionResult
				var err error
				operation := "AttachCinderVolume"
				if family == "attach" {
					result, err = blockstorage.AttachCinderVolume(selected, client, blockstorage.VolumeActionRequest{VolumeID: id}, literal,
						func(next *blockstorage.CinderVolumeAttachOpts) error {
							value := "instance"
							next.Instance = &value
							return callback()
						},
						func(*blockstorage.CinderVolumeAttachOpts) error { later++; *client = original; return nil })
				} else {
					operation = "DetachCinderVolume"
					result, err = blockstorage.DetachCinderVolume(selected, client, blockstorage.VolumeActionRequest{VolumeID: id}, literal,
						func(next *blockstorage.CinderVolumeDetachOpts) error {
							yes := true
							next.Force = &yes
							return callback()
						},
						func(*blockstorage.CinderVolumeDetachOpts) error { later++; *client = original; return nil })
				}
				vsaOperation(t, err, operation)
				var proof *resource.ResponseError
				if result != nil || !errors.Is(err, want) || errors.As(err, &proof) || first != wantFirst || later != 0 || requests.Load() != 0 {
					t.Fatal(result, err, first, later, requests.Load())
				}
				if (kind == "already canceled" || kind == "source callback and cancellation") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal("lost cancellation identities", err)
				}
				if kind == "source callback and cancellation" && !errors.Is(err, callbackCause) {
					t.Fatal("lost callback identity", err)
				}
			})
		}
	}
}
