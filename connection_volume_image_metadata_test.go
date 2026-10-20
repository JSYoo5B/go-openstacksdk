package openstack_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func vimDeletedKeys(result *blockstorage.VolumeImageMetadataDeleteResult) []string {
	keys := make([]string, len(result.Deleted))
	for i, entry := range result.Deleted {
		keys[i] = entry.Key
	}
	return keys
}

func TestConnectionVolumeImageMetadataUsesCachedCinderAndPreservesAllEmptyAndDuplicateIntents(t *testing.T) {
	cloud := testcloud.New(t)
	var getters, gets, posts atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		getters.Add(1)
		if opts.Type != "block-storage" || opts.Version != 3 {
			t.Error("unexpected Compute or other service", opts)
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
	client.MoreHeaders = map[string]string{"X-Image-Caller": "captured", "OpenStack-API-Version": "volume 3.90", "X-OpenStack-Volume-API-Version": "3.90"}
	bodies := []string{
		`{"os-set_image_metadata":{"metadata":{}}}`,
		`{"os-unset_image_metadata":{"key":"b"}}`,
		`{"os-unset_image_metadata":{"key":""}}`,
		`{"os-unset_image_metadata":{"key":"b"}}`,
		`{"os-unset_image_metadata":{"key":"a"}}`,
		`{"os-unset_image_metadata":{"key":"empty"}}`,
		`{"os-unset_image_metadata":{"key":"z"}}`,
	}
	member := `{"volume":{"id":"other identity","metadata":{"wrong":true},"volume_image_metadata":{"z":null,"a":false,"empty":{}}}}`
	ack := []byte{0xff, 0x00, '{'}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			gets.Add(1)
			if r.URL.EscapedPath() != vcmBase+"volumes/literal-name" || r.URL.RawQuery != "" || r.ContentLength != 0 || len(r.TransferEncoding) != 0 || r.Header.Get("X-Image-Caller") != "captured" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != "volume 3.90" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.90" {
				t.Error("member observation changed target or policy", r.URL, r.Header)
			}
			w.Header().Set("X-Image-Stage", "member")
			testcloud.JSON(w, 200, member)
			return
		}
		n := posts.Add(1)
		if n > int32(len(bodies)) {
			t.Error("extra image action or wait", r.URL)
			w.WriteHeader(500)
			return
		}
		vcmWire(t, r, bodies[n-1], "3.90")
		if r.Header.Get("X-Image-Caller") != "captured" {
			t.Error(r.Header)
		}
		w.Header().Set("X-Image-Stage", "action")
		w.WriteHeader(203)
		_, _ = w.Write(ack)
	})
	input := blockstorage.VolumeActionRequest{VolumeID: "literal-name"}
	set, err := conn.SetVolumeImageMetadata(vcmContext(t), input)
	if err != nil || set == nil || !set.Completed || set.VolumeID != input.VolumeID || set.Microversion != "3.90" || len(set.Discovery) != 0 || set.Applied == nil || set.Applied.StatusCode != 203 || !bytes.Equal(set.Applied.Body, ack) {
		t.Fatal(set, err)
	}
	empty, err := conn.DeleteVolumeImageMetadata(vcmContext(t), input, blockstorage.WithVolumeImageMetadataDeleteKeys())
	if err != nil || empty == nil || !empty.Completed || empty.VolumeID != input.VolumeID || empty.Microversion != "3.90" || len(empty.Discovery) != 0 || empty.Observed != nil || len(empty.Deleted) != 0 || empty.Failed != nil || posts.Load() != 1 || gets.Load() != 0 {
		t.Fatal("explicit none performed HTTP or became all", empty, err, posts.Load(), gets.Load())
	}
	explicit, err := conn.DeleteVolumeImageMetadata(vcmContext(t), input, blockstorage.WithVolumeImageMetadataDeleteKeys("b", "", "b"))
	if err != nil || explicit == nil || !explicit.Completed || explicit.Observed != nil || explicit.Failed != nil || len(explicit.Discovery) != 0 || !reflect.DeepEqual(vimDeletedKeys(explicit), []string{"b", "", "b"}) || posts.Load() != 4 || gets.Load() != 0 {
		t.Fatal(explicit, err, posts.Load(), gets.Load())
	}
	all, err := conn.DeleteVolumeImageMetadata(vcmContext(t), input)
	if err != nil || all == nil || !all.Completed || all.VolumeID != input.VolumeID || all.Microversion != "3.90" || len(all.Discovery) != 0 || all.Failed != nil || all.Observed == nil || all.Observed.StatusCode != 200 || all.Observed.Header.Get("X-Image-Stage") != "member" || string(all.Observed.Body) != member || !reflect.DeepEqual(vimDeletedKeys(all), []string{"a", "empty", "z"}) {
		t.Fatal(all, err)
	}
	for _, result := range []*blockstorage.VolumeImageMetadataDeleteResult{explicit, all} {
		for _, entry := range result.Deleted {
			if entry.Response == nil || entry.Response == result.Observed || entry.Response.StatusCode != 203 || entry.Response.Header.Get("X-Image-Stage") != "action" || !bytes.Equal(entry.Response.Body, ack) {
				t.Fatal("member proof borrowed as key acknowledgement", entry, result)
			}
		}
	}
	if getters.Load() != 1 || gets.Load() != 1 || posts.Load() != 7 || client.Microversion != "3.90" || client.MoreHeaders["OpenStack-API-Version"] != "volume 3.90" || client.MoreHeaders["X-OpenStack-Volume-API-Version"] != "3.90" {
		t.Fatal(getters.Load(), gets.Load(), posts.Load(), client)
	}
}

func TestConnectionVolumeImageMetadataOriginalsRunOnceBeforeGetterAndRetainedOptionsStayDetached(t *testing.T) {
	for _, family := range []string{"set", "delete"} {
		t.Run(family, func(t *testing.T) {
			cloud := testcloud.New(t)
			var originals, getters, posts atomic.Int32
			var retainedSet *blockstorage.VolumeImageMetadataOpts
			var retainedDelete *blockstorage.VolumeImageMetadataDeleteOpts
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				getters.Add(1)
				if originals.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 {
					t.Error("getter preceded originals or selected another service", originals.Load(), opts)
				}
				if retainedSet != nil {
					retainedSet.Metadata["payload"][0] = '['
					delete(retainedSet.Metadata, "payload")
				}
				if retainedDelete != nil {
					(*retainedDelete.Keys)[0] = "getter changed"
					*retainedDelete.Keys = nil
				}
				return cloud.Server.URL + vcmBase, nil
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.90"))
			if err != nil {
				t.Fatal(err)
			}
			bodies := []string{`{"os-set_image_metadata":{"metadata":{"payload":{"nested":[false,null,9007199254740993]}}}}`}
			if family == "delete" {
				bodies = []string{`{"os-unset_image_metadata":{"key":"b"}}`, `{"os-unset_image_metadata":{"key":"a"}}`, `{"os-unset_image_metadata":{"key":"b"}}`}
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := posts.Add(1)
				if n > int32(len(bodies)) {
					t.Error("extra lookup or action", r.URL)
					w.WriteHeader(500)
					return
				}
				vcmWire(t, r, bodies[n-1], "3.90")
				w.WriteHeader(204)
			})
			input := blockstorage.VolumeActionRequest{VolumeID: "literal-name"}
			if family == "set" {
				result, cause := conn.SetVolumeImageMetadata(vcmContext(t), input, func(next *blockstorage.VolumeImageMetadataOpts) error {
					originals.Add(1)
					next.Metadata = map[string]json.RawMessage{"payload": json.RawMessage(`{"nested":[false,null,9007199254740993]}`)}
					retainedSet = next
					return nil
				})
				if cause != nil || result == nil || !result.Completed || result.Applied == nil || result.Applied.StatusCode != 204 || len(result.Discovery) != 0 {
					t.Fatal(result, cause)
				}
			} else {
				result, cause := conn.DeleteVolumeImageMetadata(vcmContext(t), input, func(next *blockstorage.VolumeImageMetadataDeleteOpts) error {
					originals.Add(1)
					keys := []string{"b", "a", "b"}
					next.Keys = &keys
					retainedDelete = next
					return nil
				})
				if cause != nil || result == nil || !result.Completed || result.Observed != nil || result.Failed != nil || len(result.Discovery) != 0 || !reflect.DeepEqual(vimDeletedKeys(result), []string{"b", "a", "b"}) {
					t.Fatal(result, cause)
				}
			}
			if originals.Load() != 1 || getters.Load() != 1 || posts.Load() != int32(len(bodies)) {
				t.Fatal(originals.Load(), getters.Load(), posts.Load())
			}
		})
	}
}

func TestConnectionVolumeImageMetadataPreflightOptionsAndGetterFailuresKeepEveryCause(t *testing.T) {
	for _, family := range []string{"set", "delete"} {
		for _, kind := range []string{"nil connection", "nil context", "already canceled", "nil option", "callback error and cancel", "invalid final option", "unsafe ID", "getter error", "getter error and cancel", "cached wrong source"} {
			t.Run(family+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancelCause(vcmContext(t))
				defer cancel(nil)
				var selected context.Context = ctx
				callbackCause, cancelCause, getterCause := errors.New("image Connection callback"), errors.New("image Connection custom cancel"), errors.New("image Connection getter")
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
				input := blockstorage.VolumeActionRequest{VolumeID: "literal-name"}
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
				case "unsafe ID":
					input.VolumeID = "a/b"
				case "getter error", "getter error and cancel":
					want = getterCause
					wantGetters = 1
				case "cached wrong source":
					service, cause := conn.BlockStorageV3(vcmContext(t))
					if cause != nil {
						t.Fatal(cause)
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
				operation := "SetVolumeImageMetadata"
				if family == "set" {
					options := []blockstorage.VolumeImageMetadataOption{
						func(next *blockstorage.VolumeImageMetadataOpts) error {
							next.Metadata = map[string]json.RawMessage{"key": json.RawMessage(`null`)}
							if kind == "invalid final option" {
								next.Metadata["key"] = json.RawMessage(`{]`)
							}
							return callback()
						},
						func(*blockstorage.VolumeImageMetadataOpts) error { later++; return nil },
					}
					if kind == "nil option" {
						options[0] = nil
					}
					result, cause := conn.SetVolumeImageMetadata(selected, input, options...)
					err = cause
					if result != nil {
						t.Fatal("set preflight invented result", result, err)
					}
				} else {
					operation = "DeleteVolumeImageMetadata"
					options := []blockstorage.VolumeImageMetadataDeleteOption{
						func(next *blockstorage.VolumeImageMetadataDeleteOpts) error {
							keys := []string{"key"}
							if kind == "invalid final option" {
								keys[0] = string([]byte{0xff})
							}
							next.Keys = &keys
							return callback()
						},
						func(*blockstorage.VolumeImageMetadataDeleteOpts) error { later++; return nil },
					}
					if kind == "nil option" {
						options[0] = nil
					}
					result, cause := conn.DeleteVolumeImageMetadata(selected, input, options...)
					err = cause
					if result != nil {
						t.Fatal("delete preflight invented result", result, err)
					}
				}
				vscError(t, operation, err)
				var proof *resource.ResponseError
				if !errors.Is(err, want) || errors.As(err, &proof) || first != wantFirst || later != wantLater || getters.Load() != wantGetters || requests.Load() != 0 {
					t.Fatal(err, first, later, getters.Load(), requests.Load())
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
