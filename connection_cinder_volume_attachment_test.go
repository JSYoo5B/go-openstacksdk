package openstack_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionCinderVolumeAttachmentsUseCachedCinderAndIgnoreLocation(t *testing.T) {
	cloud := testcloud.New(t)
	var getters, posts atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		getters.Add(1)
		if opts.Type != "block-storage" || opts.Version != 3 {
			t.Error("unexpected Nova or other service selection", opts)
		}
		return cloud.Server.URL + vcmBase, nil
	}
	bad := string([]byte{0xff})
	location := resource.CloudLocation{Cloud: &bad, Zone: json.RawMessage("invalid"), Project: resource.CloudProject{ID: json.RawMessage("invalid")}}
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.90"), sdk.WithCloudLocation(location))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.BlockStorageV3(vcmContext(t))
	if err != nil {
		t.Fatal(err)
	}
	client := service.RawClient()
	client.MoreHeaders = map[string]string{"X-Direct-Caller": "captured", "OpenStack-API-Version": "volume 3.90", "X-OpenStack-Volume-API-Version": "3.90"}
	bodies := []string{
		`{"os-attach":{"instance_uuid":"","mountpoint":""}}`,
		`{"os-detach":{"attachment_id":""}}`,
		`{"os-force_detach":{"attachment_id":"attachment / ?#","connector":{"host":"","multipath":false,"opaque":null}}}`,
	}
	raw := []byte{0xff, 0x00, '{'}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		n := posts.Add(1)
		if n > int32(len(bodies)) {
			t.Error("extra lookup, discovery or wait", r.URL)
			w.WriteHeader(500)
			return
		}
		vcmWire(t, r, bodies[n-1], "3.90")
		if r.Header.Get("X-Direct-Caller") != "captured" {
			t.Error(r.Header)
		}
		w.Header().Set("X-Direct-Proof", "action")
		w.WriteHeader(203)
		_, _ = w.Write(raw)
	})
	input := blockstorage.VolumeActionRequest{VolumeID: "literal-name"}
	calls := []func() (*blockstorage.VolumeActionResult, error){
		func() (*blockstorage.VolumeActionResult, error) {
			return conn.AttachCinderVolume(vcmContext(t), input, "", blockstorage.WithCinderVolumeAttachHostName(bad), blockstorage.WithCinderVolumeAttachInstance(""))
		},
		func() (*blockstorage.VolumeActionResult, error) {
			return conn.DetachCinderVolume(vcmContext(t), input, "", blockstorage.WithCinderVolumeDetachConnector(map[string]json.RawMessage{bad: json.RawMessage(`{]`)}))
		},
		func() (*blockstorage.VolumeActionResult, error) {
			return conn.DetachCinderVolume(vcmContext(t), input, "attachment / ?#", blockstorage.WithCinderVolumeDetachForce(true), blockstorage.WithCinderVolumeDetachConnector(map[string]json.RawMessage{"host": json.RawMessage(`""`), "multipath": json.RawMessage(`false`), "opaque": json.RawMessage(`null`)}))
		},
	}
	for _, call := range calls {
		result, err := call()
		if err != nil || result == nil || !result.Completed || result.VolumeID != input.VolumeID || result.Microversion != "3.90" || len(result.Discovery) != 0 || result.Applied == nil || result.Applied.StatusCode != 203 || result.Applied.Header.Get("X-Direct-Proof") != "action" || !bytes.Equal(result.Applied.Body, raw) {
			t.Fatal(result, err)
		}
	}
	if getters.Load() != 1 || posts.Load() != 3 || client.Microversion != "3.90" || client.MoreHeaders["OpenStack-API-Version"] != "volume 3.90" || client.MoreHeaders["X-OpenStack-Volume-API-Version"] != "3.90" {
		t.Fatal(getters.Load(), posts.Load(), client)
	}
}

func TestConnectionCinderVolumeAttachmentOriginalsRunOnceBeforeGetterAndRetainedValuesStayDetached(t *testing.T) {
	for _, family := range []string{"attach", "detach"} {
		t.Run(family, func(t *testing.T) {
			cloud := testcloud.New(t)
			var originals, getters, posts atomic.Int32
			var retainedAttach *blockstorage.CinderVolumeAttachOpts
			var retainedDetach *blockstorage.CinderVolumeDetachOpts
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				getters.Add(1)
				if originals.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 {
					t.Error("getter preceded originals or selected another service", originals.Load(), opts)
				}
				if retainedAttach != nil {
					*retainedAttach.Instance = "getter changed caller instance"
					retainedAttach.HostName = nil
				}
				if retainedDetach != nil {
					*retainedDetach.Force = false
					retainedDetach.Connector["payload"][0] = '['
					delete(retainedDetach.Connector, "payload")
				}
				return cloud.Server.URL + vcmBase, nil
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.90"))
			if err != nil {
				t.Fatal(err)
			}
			body := `{"os-attach":{"instance_uuid":"owned","mountpoint":"mount / ?#"}}`
			if family == "detach" {
				body = `{"os-force_detach":{"attachment_id":"attachment / ?#","connector":{"payload":{"nested":[false,null,9007199254740993]}}}}`
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				vcmWire(t, r, body, "3.90")
				w.WriteHeader(204)
			})
			input := blockstorage.VolumeActionRequest{VolumeID: "literal-name"}
			var result *blockstorage.VolumeActionResult
			if family == "attach" {
				result, err = conn.AttachCinderVolume(vcmContext(t), input, "mount / ?#", func(next *blockstorage.CinderVolumeAttachOpts) error {
					originals.Add(1)
					value := "owned"
					next.Instance = &value
					retainedAttach = next
					return nil
				})
			} else {
				result, err = conn.DetachCinderVolume(vcmContext(t), input, "attachment / ?#", func(next *blockstorage.CinderVolumeDetachOpts) error {
					originals.Add(1)
					yes := true
					next.Force = &yes
					next.Connector = map[string]json.RawMessage{"payload": json.RawMessage(`{"nested":[false,null,9007199254740993]}`)}
					retainedDetach = next
					return nil
				})
			}
			if err != nil || result == nil || !result.Completed || len(result.Discovery) != 0 || result.Applied == nil || result.Applied.StatusCode != 204 || originals.Load() != 1 || getters.Load() != 1 || posts.Load() != 1 {
				t.Fatal(result, err, originals.Load(), getters.Load(), posts.Load())
			}
		})
	}
}

func TestConnectionCinderVolumeAttachmentPreflightOptionsAndGetterFailuresKeepCauses(t *testing.T) {
	for _, family := range []string{"attach", "detach"} {
		kinds := []string{"nil connection", "nil context", "already canceled", "nil option", "callback error and cancel", "invalid literal", "unsafe ID", "invalid active option", "getter error", "getter error and cancel", "cached wrong source"}
		for _, kind := range kinds {
			t.Run(family+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancelCause(vcmContext(t))
				defer cancel(nil)
				var selected context.Context = ctx
				callbackCause, cancelCause, getterCause := errors.New("direct Connection callback"), errors.New("direct Connection custom cancel"), errors.New("direct Connection getter")
				var getters, requests atomic.Int32
				first, later := 0, 0
				cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
					getters.Add(1)
					if kind == "getter error and cancel" {
						cancel(cancelCause)
						return "", getterCause
					}
					if kind == "getter error" {
						return "", getterCause
					}
					return cloud.Server.URL + vcmBase, nil
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(500) })
				conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.90"))
				if err != nil {
					t.Fatal(err)
				}
				input, literal := blockstorage.VolumeActionRequest{VolumeID: "literal-name"}, ""
				want, wantFirst, wantLater, wantGetters := resource.ErrInvalidOption, 1, 1, int32(0)
				switch kind {
				case "nil connection":
					conn = nil
					wantFirst, wantLater = 0, 0
				case "nil context":
					selected = nil
					wantFirst, wantLater = 0, 0
				case "already canceled":
					cancel(cancelCause)
					want = context.Canceled
					wantFirst, wantLater = 0, 0
				case "nil option":
					wantFirst, wantLater = 0, 0
				case "callback error and cancel":
					want = callbackCause
					wantLater = 0
				case "invalid literal":
					literal = string([]byte{0xff})
				case "unsafe ID":
					input.VolumeID = "a/b"
				case "getter error", "getter error and cancel":
					want = getterCause
					wantGetters = 1
				case "cached wrong source":
					service, e := conn.BlockStorageV3(vcmContext(t))
					if e != nil {
						t.Fatal(e)
					}
					service.RawClient().Type = "compute"
					want, wantGetters = resource.ErrUnsupported, 1
				}
				callback := func() error {
					first++
					if kind == "callback error and cancel" {
						cancel(cancelCause)
						return callbackCause
					}
					return nil
				}
				var result *blockstorage.VolumeActionResult
				operation := "AttachCinderVolume"
				if family == "attach" {
					options := []blockstorage.CinderVolumeAttachOption{
						func(next *blockstorage.CinderVolumeAttachOpts) error {
							value := "instance"
							if kind == "invalid active option" {
								value = string([]byte{0xff})
							}
							next.Instance = &value
							return callback()
						},
						func(*blockstorage.CinderVolumeAttachOpts) error { later++; return nil },
					}
					if kind == "nil option" {
						options[0] = nil
					}
					result, err = conn.AttachCinderVolume(selected, input, literal, options...)
				} else {
					operation = "DetachCinderVolume"
					options := []blockstorage.CinderVolumeDetachOption{
						func(next *blockstorage.CinderVolumeDetachOpts) error {
							yes := true
							next.Force = &yes
							next.Connector = map[string]json.RawMessage{"payload": json.RawMessage(`null`)}
							if kind == "invalid active option" {
								next.Connector["payload"] = json.RawMessage(`{]`)
							}
							return callback()
						},
						func(*blockstorage.CinderVolumeDetachOpts) error { later++; return nil },
					}
					if kind == "nil option" {
						options[0] = nil
					}
					result, err = conn.DetachCinderVolume(selected, input, literal, options...)
				}
				vscError(t, operation, err)
				var proof *resource.ResponseError
				if result != nil || !errors.Is(err, want) || errors.As(err, &proof) || first != wantFirst || later != wantLater || getters.Load() != wantGetters || requests.Load() != 0 {
					t.Fatal(result, err, first, later, getters.Load(), requests.Load())
				}
				if (kind == "already canceled" || kind == "callback error and cancel" || kind == "getter error and cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal("lost custom context identities", err)
				}
				if kind == "callback error and cancel" && !errors.Is(err, callbackCause) {
					t.Fatal("lost callback identity", err)
				}
				if (kind == "getter error" || kind == "getter error and cancel") && !errors.Is(err, getterCause) {
					t.Fatal("lost getter identity", err)
				}
			})
		}
	}
}
