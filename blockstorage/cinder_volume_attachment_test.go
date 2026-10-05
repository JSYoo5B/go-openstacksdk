package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/blockstorage/v3/volumes"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

type cdaCall func(context.Context, *gophercloud.ServiceClient, string) (*blockstorage.VolumeActionResult, error)
type cdaCase struct {
	name, operation, body string
	call                  cdaCall
}

func cdaActions() []cdaCase {
	return []cdaCase{
		{"attach", "AttachCinderVolume", `{"os-attach":{"instance_uuid":"instance","mountpoint":"/m"}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.AttachCinderVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "/m", blockstorage.WithCinderVolumeAttachInstance("instance"))
		}},
		{"detach", "DetachCinderVolume", `{"os-detach":{"attachment_id":"attachment"}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.DetachCinderVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "attachment")
		}},
		{"force detach", "DetachCinderVolume", `{"os-force_detach":{"attachment_id":"attachment","connector":{"host":"","large":9007199254740993,"multipath":false,"unknown":null}}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.DetachCinderVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "attachment", blockstorage.WithCinderVolumeDetachForce(true), blockstorage.WithCinderVolumeDetachConnector(map[string]json.RawMessage{"host": json.RawMessage(`""`), "large": json.RawMessage(`9007199254740993`), "multipath": json.RawMessage(`false`), "unknown": json.RawMessage(`null`)}))
		}},
	}
}
func cdaPost(t *testing.T, req *http.Request, id, body, version, token string) {
	t.Helper()
	var raw []byte
	var err error
	if req.Body != nil {
		raw, err = io.ReadAll(req.Body)
	}
	current := ""
	if version != "" {
		current = "volume " + version
	}
	if err != nil || string(raw) != body || req.Method != http.MethodPost || req.URL.EscapedPath() != vsaBase+"volumes/"+url.PathEscape(id)+"/action" || req.URL.RawQuery != "" || req.Header.Get("X-Source") != "original" || req.Header.Get("X-Auth-Token") != token || req.Header.Get("OpenStack-API-Version") != current || req.Header.Get("X-OpenStack-Volume-API-Version") != version || req.ContentLength != int64(len(body)) || len(req.TransferEncoding) != 0 {
		t.Error("direct attachment changed literal action body, fixed route, framing or captured policy", req.Method, req.URL, string(raw), req.Header, req.ContentLength, req.TransferEncoding, err)
	}
}
func cdaNoAdmitted(t *testing.T, result *blockstorage.VolumeActionResult, err error, operation string) {
	t.Helper()
	vsaOperation(t, err, operation)
	var proof *resource.ResponseError
	if result == nil || result.VolumeID != "id" || result.Completed || result.Applied != nil || len(result.Discovery) != 0 || errors.As(err, &proof) {
		t.Fatal("natively rejected action acquired accepted proof", result, err, proof)
	}
}

func TestCinderVolumeAttachmentLiteralBranchesAcceptOpaqueAcknowledgementsWithoutLookup(t *testing.T) {
	bad := string([]byte{0xff})
	cases := append(cdaActions(), []cdaCase{
		{"empty instance wins nonempty host", "AttachCinderVolume", `{"os-attach":{"instance_uuid":"","mountpoint":""}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.AttachCinderVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "", blockstorage.WithCinderVolumeAttachInstance(""), blockstorage.WithCinderVolumeAttachHostName("ignored"))
		}},
		{"inactive invalid host is not consumed", "AttachCinderVolume", `{"os-attach":{"instance_uuid":"instance","mountpoint":"/m"}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.AttachCinderVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "/m", blockstorage.WithCinderVolumeAttachInstance("instance"), blockstorage.WithCinderVolumeAttachHostName(bad))
		}},
		{"empty host and literal control mountpoint", "AttachCinderVolume", `{"os-attach":{"host_name":"","mountpoint":" /?%#\n\u0000한글"}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.AttachCinderVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, " /?%#\n\x00한글", blockstorage.WithCinderVolumeAttachHostName(""))
		}},
		{"normal detach ignores invalid connector", "DetachCinderVolume", `{"os-detach":{"attachment_id":""}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.DetachCinderVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "", blockstorage.WithCinderVolumeDetachConnector(map[string]json.RawMessage{bad: nil, "invalid": json.RawMessage(`not JSON`)}))
		}},
		{"explicit false ignores invalid connector", "DetachCinderVolume", `{"os-detach":{"attachment_id":" /?%#\n\u0000한글"}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.DetachCinderVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, " /?%#\n\x00한글", blockstorage.WithCinderVolumeDetachForce(false), blockstorage.WithCinderVolumeDetachConnector(map[string]json.RawMessage{"invalid": json.RawMessage{0xff}}))
		}},
		{"force nil connector omits connector", "DetachCinderVolume", `{"os-force_detach":{"attachment_id":""}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.DetachCinderVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "", blockstorage.WithCinderVolumeDetachForce(true))
		}},
		{"force empty connector omits connector", "DetachCinderVolume", `{"os-force_detach":{"attachment_id":""}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.DetachCinderVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "", blockstorage.WithCinderVolumeDetachForce(true), blockstorage.WithCinderVolumeDetachConnector(map[string]json.RawMessage{}))
		}},
	}...)
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.60")
			id := "한글-é"
			var calls atomic.Int32
			code := []int{200, 203, 302, 399}[index%4]
			reply := []byte{0xff, 0x00, '{', '!'}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				cdaPost(t, req, id, tc.body, "3.60", "test-token")
				w.Header().Set("X-Proof", "opaque action")
				w.WriteHeader(code)
				_, _ = w.Write(reply)
			})
			result, err := tc.call(vsaContext(t), client, id)
			if err != nil || result == nil || !result.Completed || result.VolumeID != id || result.Microversion != "3.60" || len(result.Discovery) != 0 || result.Applied == nil || result.Applied.StatusCode != code || result.Applied.Header.Get("X-Proof") != "opaque action" || !bytes.Equal(result.Applied.Body, reply) || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			result.Applied.Body[0] = '!'
			if reply[0] != 0xff {
				t.Fatal("caller-owned accepted proof aliases fixture body")
			}
		})
	}
}

func TestCinderVolumeAttachmentUsesOrdinarySelectedOrCappedMicroversionWithoutMinimumGate(t *testing.T) {
	cases := []struct {
		name, selected, min, max, want string
		discover                       bool
	}{
		{"selected below action caps", "3.1", "", "", "3.1", false},
		{"selected exceeds resource cap", "3.90", "", "", "3.90", false},
		{"selected latest is literal", "latest", "", "", "latest", false},
		{"unselected cap", "", "3.0", "3.99", "3.71", true},
		{"unselected lower server maximum", "", "3.0", "3.40", "3.40", true},
		{"unselected server minimum above cap", "", "3.99", "3.99", "", true},
	}
	for _, action := range cdaActions() {
		for _, tc := range cases {
			t.Run(action.name+" "+tc.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, tc.selected)
				var calls atomic.Int32
				if tc.selected != "" {
					client.MoreHeaders["OpenStack-API-Version"] = "volume " + tc.selected
					client.MoreHeaders["X-OpenStack-Volume-API-Version"] = tc.selected
				}
				advertisement := `{"version":{"id":"v3.0","min_version":"` + tc.min + `","max_version":"` + tc.max + `"}}`
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
					n := calls.Add(1)
					if tc.discover && n == 1 {
						vsaDiscoveryRequest(t, req, vsaVersionPath)
						w.Header().Set("X-Proof", "ordinary discovery")
						testcloud.JSON(w, 300, advertisement)
						return
					}
					cdaPost(t, req, "id", action.body, tc.want, "test-token")
					w.Header().Set("X-Proof", "ordinary action")
					testcloud.JSON(w, 203, "opaque")
				})
				result, err := action.call(vsaContext(t), client, "id")
				wantCalls := int32(1)
				wantPages := 0
				if tc.discover {
					wantCalls = 2
					wantPages = 1
				}
				if err != nil || result == nil || !result.Completed || result.Microversion != tc.want || client.Microversion != tc.selected || len(result.Discovery) != wantPages || result.Applied == nil || result.Applied.StatusCode != 203 || result.Applied.Header.Get("X-Proof") != "ordinary action" || string(result.Applied.Body) != "opaque" || calls.Load() != wantCalls {
					t.Fatal(result, err, calls.Load())
				}
				if tc.discover {
					if string(result.Discovery[0].Body) != advertisement || result.Discovery[0].StatusCode != 300 || result.Discovery[0].Header.Get("X-Proof") != "ordinary discovery" {
						t.Fatal(result.Discovery)
					}
					result.Discovery[0].Body[0] = '!'
					result.Discovery[0].Header.Set("X-Proof", "mutated")
					if string(result.Applied.Body) != "opaque" || result.Applied.Header.Get("X-Proof") != "ordinary action" {
						t.Fatal("discovery proof aliases action acknowledgement")
					}
				}
			})
		}
	}
}

func TestCinderVolumeAttachmentPreflightAndOwnedOriginalOptionsPrecedeHTTP(t *testing.T) {
	for _, action := range cdaActions() {
		for _, kind := range []string{"nil context", "canceled context", "nil client", "wrong service", "unsafe volume ID", "invalid required text", "source changed by original"} {
			t.Run(action.name+" "+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.60")
				ctx := vsaContext(t)
				id := "id"
				text := "/m"
				if action.operation == "DetachCinderVolume" {
					text = "attachment"
				}
				cause := errors.New("direct attachment caller cancellation")
				want := resource.ErrInvalidOption
				switch kind {
				case "nil context":
					ctx = nil
				case "canceled context":
					canceled, cancel := context.WithCancelCause(ctx)
					cancel(cause)
					ctx = canceled
					want = context.Canceled
				case "nil client":
					client = nil
				case "wrong service":
					client.Type = "compute"
					want = resource.ErrUnsupported
				case "unsafe volume ID":
					id = "id/other"
				case "invalid required text":
					text = string([]byte{0xff})
				}
				var calls, originals, later atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
					calls.Add(1)
					t.Error("invalid direct action reached HTTP", req.URL)
					w.WriteHeader(500)
				})
				var result *blockstorage.VolumeActionResult
				var err error
				if action.operation == "AttachCinderVolume" {
					result, err = blockstorage.AttachCinderVolume(ctx, client, blockstorage.VolumeActionRequest{VolumeID: id}, text, func(o *blockstorage.CinderVolumeAttachOpts) error {
						originals.Add(1)
						value := "instance"
						o.Instance = &value
						if kind == "source changed by original" {
							client.ResourceBase += "changed/"
						}
						return nil
					}, func(o *blockstorage.CinderVolumeAttachOpts) error {
						later.Add(1)
						if kind == "source changed by original" {
							client.ResourceBase = cloud.Server.URL + vsaBase
						}
						return nil
					})
				} else {
					result, err = blockstorage.DetachCinderVolume(ctx, client, blockstorage.VolumeActionRequest{VolumeID: id}, text, func(o *blockstorage.CinderVolumeDetachOpts) error {
						originals.Add(1)
						if kind == "source changed by original" {
							client.ResourceBase += "changed/"
						}
						return nil
					}, func(o *blockstorage.CinderVolumeDetachOpts) error {
						later.Add(1)
						if kind == "source changed by original" {
							client.ResourceBase = cloud.Server.URL + vsaBase
						}
						return nil
					})
				}
				vsaOperation(t, err, action.operation)
				var proof *resource.ResponseError
				wantOriginals := int32(0)
				if kind == "source changed by original" {
					wantOriginals = 1
				}
				if result != nil || !errors.Is(err, want) || errors.As(err, &proof) || calls.Load() != 0 || originals.Load() != wantOriginals || later.Load() != 0 || kind == "canceled context" && !errors.Is(err, cause) {
					t.Fatal(result, err, proof, calls.Load(), originals.Load(), later.Load())
				}
			})
		}
	}
	for _, tc := range []struct {
		name, operation string
		call            cdaCall
	}{
		{"missing selectors", "AttachCinderVolume", func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.AttachCinderVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "")
		}},
		{"invalid selected instance", "AttachCinderVolume", func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.AttachCinderVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "", blockstorage.WithCinderVolumeAttachInstance(string([]byte{0xff})), blockstorage.WithCinderVolumeAttachHostName("valid inactive host"))
		}},
		{"invalid selected host", "AttachCinderVolume", func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.AttachCinderVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "", blockstorage.WithCinderVolumeAttachHostName(string([]byte{0xff})))
		}},
		{"forced nil raw value", "DetachCinderVolume", func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.DetachCinderVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "", blockstorage.WithCinderVolumeDetachForce(true), blockstorage.WithCinderVolumeDetachConnector(map[string]json.RawMessage{"nil": nil}))
		}},
		{"forced trailing raw JSON", "DetachCinderVolume", func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.DetachCinderVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "", blockstorage.WithCinderVolumeDetachForce(true), blockstorage.WithCinderVolumeDetachConnector(map[string]json.RawMessage{"invalid": json.RawMessage(`null true`)}))
		}},
		{"forced invalid UTF8 key", "DetachCinderVolume", func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.DetachCinderVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "", blockstorage.WithCinderVolumeDetachForce(true), blockstorage.WithCinderVolumeDetachConnector(map[string]json.RawMessage{string([]byte{0xff}): json.RawMessage(`null`)}))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.60")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) { calls.Add(1); w.WriteHeader(500) })
			result, err := tc.call(vsaContext(t), client, "id")
			vsaOperation(t, err, tc.operation)
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
	t.Run("captured option slice and retained attach pointer", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := vsaClient(cloud, "3.60")
		value := "instance"
		var retained *blockstorage.CinderVolumeAttachOpts
		var originals, later, replaced, calls atomic.Int32
		var options []blockstorage.CinderVolumeAttachOption
		options = []blockstorage.CinderVolumeAttachOption{func(o *blockstorage.CinderVolumeAttachOpts) error {
			originals.Add(1)
			o.Instance = &value
			retained = o
			options[1] = func(*blockstorage.CinderVolumeAttachOpts) error {
				replaced.Add(1)
				return errors.New("replacement must not execute")
			}
			return nil
		}, func(*blockstorage.CinderVolumeAttachOpts) error {
			later.Add(1)
			*retained.Instance = "mutated retained instance"
			client.MoreHeaders["x-source"] = "ordinary later header"
			return nil
		}}
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			calls.Add(1)
			cdaPost(t, req, "id", `{"os-attach":{"instance_uuid":"instance","mountpoint":"/m"}}`, "3.60", "test-token")
			w.WriteHeader(203)
		})
		result, err := blockstorage.AttachCinderVolume(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, "/m", options...)
		if err != nil || result == nil || !result.Completed || calls.Load() != 1 || originals.Load() != 1 || later.Load() != 1 || replaced.Load() != 0 || value != "mutated retained instance" {
			t.Fatal(result, err, calls.Load(), originals.Load(), later.Load(), replaced.Load(), value)
		}
	})
	t.Run("connector factory and retained callback raw bytes are owned", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := vsaClient(cloud, "3.60")
		raw := json.RawMessage(`9007199254740993`)
		members := map[string]json.RawMessage{"large": raw}
		owned := blockstorage.WithCinderVolumeDetachConnector(members)
		raw[0] = '1'
		members["late"] = json.RawMessage(`true`)
		var retained *blockstorage.CinderVolumeDetachOpts
		var callbacks, calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			calls.Add(1)
			cdaPost(t, req, "id", `{"os-force_detach":{"attachment_id":"attachment","connector":{"large":9007199254740993}}}`, "3.60", "test-token")
			w.WriteHeader(203)
		})
		result, err := blockstorage.DetachCinderVolume(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, "attachment", owned, blockstorage.WithCinderVolumeDetachForce(true), func(o *blockstorage.CinderVolumeDetachOpts) error { callbacks.Add(1); retained = o; return nil }, func(*blockstorage.CinderVolumeDetachOpts) error {
			callbacks.Add(1)
			retained.Connector["large"][0] = '2'
			retained.Connector["late"] = json.RawMessage(`false`)
			*retained.Force = false
			return nil
		})
		if err != nil || result == nil || !result.Completed || calls.Load() != 1 || callbacks.Load() != 2 {
			t.Fatal(result, err, calls.Load(), callbacks.Load())
		}
	})
}

func TestCinderVolumeAttachmentServiceMethodsPreserveNativeAttachDetachContracts(t *testing.T) {
	cloud := testcloud.New(t)
	client := vsaClient(cloud, "3.60")
	api := volumes.New(client)
	var calls atomic.Int32
	nativeAttach := `{"os-attach":{"host_name":"host","instance_uuid":"instance","mode":"ro","mountpoint":"/m"}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		n := calls.Add(1)
		body := nativeAttach
		code := 202
		switch n {
		case 1:
		case 2:
			code = 203
		case 3:
			body = `{"os-detach":{}}`
		case 4:
			body = `{"os-detach":{}}`
			code = 203
		case 5:
			body = `{"os-attach":{"instance_uuid":"","mountpoint":""}}`
			code = 200
		case 6:
			body = `{"os-detach":{"attachment_id":""}}`
			code = 203
		case 7:
			body = `{"os-force_detach":{"attachment_id":"","connector":{"multipath":false}}}`
			code = 399
		default:
			t.Error("unexpected native/direct service HTTP", n)
			w.WriteHeader(500)
			return
		}
		cdaPost(t, req, "id", body, "3.60", "test-token")
		w.Header().Set("X-Proof", "service action")
		w.WriteHeader(code)
		_, _ = w.Write([]byte{0xff, 0x00})
	})
	old := volumes.AttachOpts{MountPoint: "/m", InstanceUUID: "instance", HostName: "host", Mode: volumes.ReadOnly}
	if err := api.Attach(vsaContext(t), "id", old); err != nil || calls.Load() != 1 {
		t.Fatal(err, calls.Load())
	}
	err := api.Attach(vsaContext(t), "id", old)
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != 203 || len(native.Expected) != 1 || native.Expected[0] != 202 || calls.Load() != 2 {
		t.Fatal("native Attach policy changed", err, native, calls.Load())
	}
	if err := api.Detach(vsaContext(t), "id", volumes.DetachOpts{}); err != nil || calls.Load() != 3 {
		t.Fatal(err, calls.Load())
	}
	err = api.Detach(vsaContext(t), "id", volumes.DetachOpts{})
	if !errors.As(err, &native) || native.Actual != 203 || len(native.Expected) != 1 || native.Expected[0] != 202 || calls.Load() != 4 {
		t.Fatal("native Detach policy changed", err, native, calls.Load())
	}
	attached, err := api.AttachCinderVolume(vsaContext(t), "id", "", volumes.WithCinderVolumeAttachInstance(""), volumes.WithCinderVolumeAttachHostName("ignored"))
	if err != nil || attached == nil || !attached.Completed || attached.Applied == nil || attached.Applied.StatusCode != 200 || len(attached.Discovery) != 0 || calls.Load() != 5 {
		t.Fatal(attached, err, calls.Load())
	}
	detached, err := api.DetachCinderVolume(vsaContext(t), "id", "")
	if err != nil || detached == nil || !detached.Completed || detached.Applied == nil || detached.Applied.StatusCode != 203 || len(detached.Discovery) != 0 || calls.Load() != 6 {
		t.Fatal(detached, err, calls.Load())
	}
	forced, err := api.DetachCinderVolume(vsaContext(t), "id", "", volumes.WithCinderVolumeDetachForce(true), volumes.WithCinderVolumeDetachConnector(map[string]json.RawMessage{"multipath": json.RawMessage(`false`)}))
	if err != nil || forced == nil || !forced.Completed || forced.Applied == nil || forced.Applied.StatusCode != 399 || !bytes.Equal(forced.Applied.Body, []byte{0xff, 0x00}) || len(forced.Discovery) != 0 || calls.Load() != 7 {
		t.Fatal(forced, err, calls.Load())
	}
	var missing *volumes.API
	absent, err := missing.AttachCinderVolume(vsaContext(t), "id", "", volumes.WithCinderVolumeAttachInstance("instance"))
	vsaOperation(t, err, "AttachCinderVolume")
	if absent != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 7 {
		t.Fatal(absent, err, calls.Load())
	}
	absent, err = missing.DetachCinderVolume(vsaContext(t), "id", "")
	vsaOperation(t, err, "DetachCinderVolume")
	if absent != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 7 {
		t.Fatal(absent, err, calls.Load())
	}
}
