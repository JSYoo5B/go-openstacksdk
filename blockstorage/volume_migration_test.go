package blockstorage_test

import (
	"bytes"
	"context"
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

type vmrCase struct {
	name, operation, body string
	cluster               bool
	call                  func(context.Context, *gophercloud.ServiceClient, string) (*blockstorage.VolumeActionResult, error)
}

func vmrCases() []vmrCase {
	return []vmrCase{
		{"reset defaults", "ResetVolumeStatus", `{"os-reset_status":{}}`, false, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.ResetVolumeStatus(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id})
		}},
		{"reset explicit empty", "ResetVolumeStatus", `{"os-reset_status":{}}`, false, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.ResetVolumeStatus(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, blockstorage.WithVolumeStatusResetStatus(""), blockstorage.WithVolumeStatusResetAttachStatus(""), blockstorage.WithVolumeStatusResetMigrationStatus(""))
		}},
		{"reset all named statuses", "ResetVolumeStatus", `{"os-reset_status":{"attach_status":"2","migration_status":"3","status":"1"}}`, false, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.ResetVolumeStatus(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, blockstorage.WithVolumeStatusResetStatus("1"), blockstorage.WithVolumeStatusResetAttachStatus("2"), blockstorage.WithVolumeStatusResetMigrationStatus("3"))
		}},
		{"reset mixed omission", "ResetVolumeStatus", `{"os-reset_status":{"attach_status":"server-defined"}}`, false, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.ResetVolumeStatus(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, blockstorage.WithVolumeStatusResetStatus(""), blockstorage.WithVolumeStatusResetAttachStatus("server-defined"))
		}},
		{"reset literal whitespace controls", "ResetVolumeStatus", `{"os-reset_status":{"status":" \n\u0000"}}`, false, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.ResetVolumeStatus(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, blockstorage.WithVolumeStatusResetStatus(" \n\x00"))
		}},
		{"reset replacement clears", "ResetVolumeStatus", `{"os-reset_status":{}}`, false, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.ResetVolumeStatus(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, blockstorage.WithVolumeStatusResetStatus("old"), blockstorage.WithVolumeStatusResetOptions(blockstorage.VolumeStatusResetOpts{}))
		}},
		{"migration defaults", "MigrateVolume", `{"os-migrate_volume":{}}`, false, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.MigrateVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id})
		}},
		{"migration host empty and false flags", "MigrateVolume", `{"os-migrate_volume":{"host":""}}`, false, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.MigrateVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, blockstorage.WithVolumeMigrationHost(""), blockstorage.WithVolumeMigrationForceHostCopy(false), blockstorage.WithVolumeMigrationLockVolume(false))
		}},
		{"migration literal host and true flags", "MigrateVolume", `{"os-migrate_volume":{"force_host_copy":true,"host":"host /?%#\n\u0000","lock_volume":true}}`, false, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.MigrateVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, blockstorage.WithVolumeMigrationHost("host /?%#\n\x00"), blockstorage.WithVolumeMigrationForceHostCopy(true), blockstorage.WithVolumeMigrationLockVolume(true))
		}},
		{"migration empty cluster", "MigrateVolume", `{"os-migrate_volume":{"cluster":""}}`, true, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.MigrateVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, blockstorage.WithVolumeMigrationCluster(""))
		}},
		{"migration both targets and flags", "MigrateVolume", `{"os-migrate_volume":{"cluster":"cluster@backend","force_host_copy":true,"host":"host@backend","lock_volume":true}}`, true, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.MigrateVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, blockstorage.WithVolumeMigrationHost("host@backend"), blockstorage.WithVolumeMigrationCluster("cluster@backend"), blockstorage.WithVolumeMigrationForceHostCopy(true), blockstorage.WithVolumeMigrationLockVolume(true))
		}},
		{"migration replacement clears cluster and flags", "MigrateVolume", `{"os-migrate_volume":{}}`, false, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.MigrateVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, blockstorage.WithVolumeMigrationCluster("old"), blockstorage.WithVolumeMigrationForceHostCopy(true), blockstorage.WithVolumeMigrationOptions(blockstorage.VolumeMigrationOpts{}))
		}},
		{"completion default false and literal empty target", "CompleteVolumeMigration", `{"os-migrate_volume_completion":{"error":false,"new_volume":""}}`, false, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.CompleteVolumeMigration(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "")
		}},
		{"completion explicit false", "CompleteVolumeMigration", `{"os-migrate_volume_completion":{"error":false,"new_volume":"new"}}`, false, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.CompleteVolumeMigration(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "new", blockstorage.WithVolumeMigrationCompletionError(false))
		}},
		{"completion true and literal body target", "CompleteVolumeMigration", `{"os-migrate_volume_completion":{"error":true,"new_volume":"new /?%#\n\u0000"}}`, false, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.CompleteVolumeMigration(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "new /?%#\n\x00", blockstorage.WithVolumeMigrationCompletionError(true))
		}},
		{"completion replacement restores false", "CompleteVolumeMigration", `{"os-migrate_volume_completion":{"error":false,"new_volume":"new"}}`, false, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.CompleteVolumeMigration(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "new", blockstorage.WithVolumeMigrationCompletionError(true), blockstorage.WithVolumeMigrationCompletionOptions(blockstorage.VolumeMigrationCompletionOpts{}))
		}},
	}
}
func vmrRequest(t *testing.T, req *http.Request, id, body, version string) {
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
	if err != nil || string(raw) != body || req.Method != http.MethodPost || req.URL.EscapedPath() != vsaBase+"volumes/"+url.PathEscape(id)+"/action" || req.URL.RawQuery != "" || req.Header.Get("X-Source") != "original" || req.Header.Get("X-Auth-Token") != "test-token" || req.Header.Get("OpenStack-API-Version") != current || req.Header.Get("X-OpenStack-Volume-API-Version") != version || req.ContentLength != int64(len(body)) || len(req.TransferEncoding) != 0 {
		t.Error("migration action changed literal body, fixed route or selected policy", req.Method, req.URL, string(raw), req.Header, req.ContentLength, req.TransferEncoding, err)
	}
}

func TestVolumeMigrationActionsSerializeTypedDefaultsAndPresenceWithoutLookups(t *testing.T) {
	for index, tc := range vmrCases() {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.60")
			var calls atomic.Int32
			id := "한글-ID"
			raw := []byte{0xff, 0x00, '{', '!'}
			code := []int{200, 203, 302, 399}[index%4]
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				call := calls.Add(1)
				if tc.cluster && call == 1 {
					vsaDiscoveryRequest(t, req, vsaVersionPath)
					w.Header().Set("X-Proof", "required support")
					testcloud.JSON(w, 200, `{"version":{"id":"v3.0","min_version":"3.0","max_version":"3.99"}}`)
					return
				}
				vmrRequest(t, req, id, tc.body, "3.60")
				w.Header().Set("X-Proof", "opaque action")
				w.WriteHeader(code)
				_, _ = w.Write(raw)
			})
			result, err := tc.call(vsaContext(t), client, id)
			expected := int32(1)
			if tc.cluster {
				expected = 2
			}
			if err != nil || result == nil || !result.Completed || result.VolumeID != id || result.Microversion != "3.60" || len(result.Discovery) != int(expected-1) || result.Applied == nil || result.Applied.StatusCode != code || !bytes.Equal(result.Applied.Body, raw) || result.Applied.Header.Get("X-Proof") != "opaque action" || calls.Load() != expected {
				t.Fatal(result, err, calls.Load())
			}
			if tc.cluster && (result.Discovery[0] == nil || result.Discovery[0].StatusCode != 200 || result.Discovery[0].Header.Get("X-Proof") != "required support") {
				t.Fatal("lost actual support proof", result)
			}
		})
	}
}

func TestMigrateVolumeRequiresAdvertisedSupportBeforeSelectedOrComputedVersion(t *testing.T) {
	for _, tc := range []struct {
		name, selected, reply, version string
		unsupported                    bool
	}{
		{"selected low default fails", "3.15", `{"id":"v3.0","min_version":"3.0","max_version":"3.99"}`, "", true},
		{"selected equal required", "3.16", `{"id":"v3.0","min_version":"3.16","max_version":"3.16"}`, "3.16", false},
		{"selected above advertised maximum kept raw", "3.90", `{"id":"v3.0","min_version":"3.0","max_version":"3.20"}`, "3.90", false},
		{"required above advertised maximum", "3.90", `{"id":"v3.0","min_version":"3.0","max_version":"3.15"}`, "", true},
		{"required below advertised minimum", "3.90", `{"id":"v3.0","min_version":"3.17","max_version":"3.99"}`, "", true},
		{"missing advertised minimum", "3.90", `{"id":"v3.0","max_version":"3.99"}`, "", true},
		{"missing advertised maximum", "3.90", `{"id":"v3.0","min_version":"3.0"}`, "", true},
		{"global latest major does not match required", "latest", `{"id":"v3.0","min_version":"3.0","max_version":"3.99"}`, "", true},
		{"different selected major", "4.0", `{"id":"v3.0","min_version":"3.0","max_version":"3.99"}`, "", true},
		{"same major latest remains literal", "3.latest", `{"id":"v3.0","min_version":"3.0","max_version":"3.99"}`, "3.latest", false},
		{"unselected support reuses probe for cap", "", `{"id":"v3.0","min_version":"3.0","max_version":"3.99"}`, "3.71", false},
		{"unselected maximum below cap", "", `{"id":"v3.0","min_version":"3.0","max_version":"3.40"}`, "3.40", false},
		{"cross major advertised bounds contain requirement", "", `{"id":"v3.0","min_version":"3.0","max_version":"4.1"}`, "3.71", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, tc.selected)
			if tc.selected != "" {
				client.MoreHeaders["OpenStack-API-Version"] = "volume " + tc.selected
				client.MoreHeaders["X-OpenStack-Volume-API-Version"] = tc.selected
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				call := calls.Add(1)
				if call == 1 {
					vsaDiscoveryRequest(t, req, vsaVersionPath)
					w.Header().Set("X-Proof", "current support")
					testcloud.JSON(w, 300, tc.reply)
					return
				}
				if tc.unsupported || call != 2 {
					t.Error("support failure posted or selected twice", call, req.URL)
					w.WriteHeader(500)
					return
				}
				vmrRequest(t, req, "id", `{"os-migrate_volume":{"cluster":""}}`, tc.version)
				w.Header().Set("X-Proof", "current action")
				w.WriteHeader(203)
				_, _ = w.Write([]byte{0xff, 0x00})
			})
			result, err := blockstorage.MigrateVolume(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, blockstorage.WithVolumeMigrationCluster(""))
			if result == nil || result.VolumeID != "id" || len(result.Discovery) != 1 || result.Discovery[0] == nil || result.Discovery[0].StatusCode != 300 || string(result.Discovery[0].Body) != tc.reply || result.Discovery[0].Header.Get("X-Proof") != "current support" || client.Microversion != tc.selected {
				t.Fatal(result, err, calls.Load(), client)
			}
			if client.MoreHeaders["x-source"] != "original" || (tc.selected != "" && (len(client.MoreHeaders) != 3 || client.MoreHeaders["OpenStack-API-Version"] != "volume "+tc.selected || client.MoreHeaders["X-OpenStack-Volume-API-Version"] != tc.selected)) {
				t.Fatal("versionless discovery mutated caller headers", client.MoreHeaders)
			}
			if tc.unsupported {
				vsaOperation(t, err, "MigrateVolume")
				var proof *resource.ResponseError
				if !errors.Is(err, resource.ErrUnsupported) || !errors.As(err, &proof) || string(proof.Body) != tc.reply || proof.StatusCode != 300 || proof.Header.Get("X-Proof") != "current support" || result.Completed || result.Applied != nil || calls.Load() != 1 {
					t.Fatal("requirement failure lost current proof or performed POST", result, err, proof, calls.Load())
				}
			} else if err != nil || !result.Completed || result.Microversion != tc.version || result.Applied == nil || result.Applied.StatusCode != 203 || !bytes.Equal(result.Applied.Body, []byte{0xff, 0x00}) || result.Applied.Header.Get("X-Proof") != "current action" || calls.Load() != 2 {
				t.Fatal("gate changed selected policy or renegotiated", result, err, calls.Load())
			}
		})
	}
}

func TestMigrateVolumeFiniteDiscoveryKeepsCurrentProofAndNeverBorrowsRejectedPages(t *testing.T) {
	good := `{"version":{"id":"v3.0","min_version":"3.0","max_version":"3.99"}}`
	for _, tc := range []struct {
		name                 string
		codes                []int
		replies              []string
		success, unsupported bool
		native               int
	}{
		{"clean 404 version fallback", []int{404, 200}, []string{"native first", good}, true, false, 0},
		{"clean 405 version fallback", []int{405, 300}, []string{"native first", good}, true, false, 0},
		{"accepted unusable then supported", []int{200, 200}, []string{`{}`, good}, true, false, 0},
		{"two accepted unusable advertisements", []int{200, 300}, []string{`{}`, `{"versions":[]}`}, false, true, 0},
		{"accepted page then clean rejected root", []int{200, 405}, []string{`{}`, "native root"}, false, true, 0},
		{"only clean rejected candidates", []int{404, 405}, []string{"native first", "native root"}, false, true, 405},
		{"malformed accepted discovery", []int{200}, []string{`{`}, false, false, 0},
		{"invalid advertised version", []int{300}, []string{`{"id":"v3.0","min_version":"3.0","max_version":"bad"}`}, false, false, 0},
		{"valid JSON wrong advertisement shape", []int{200}, []string{`{"versions":false}`}, false, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.60")
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = vsaTransport(func(req *http.Request) (*http.Response, error) {
				n := int(calls.Add(1))
				if n <= len(tc.codes) {
					path := vsaVersionPath
					if n == 2 {
						path = vsaRootPath
					}
					vsaDiscoveryRequest(t, req, path)
					return vsaResponse(req, tc.codes[n-1], []byte(tc.replies[n-1]), []string{"first discovery", "root discovery"}[n-1]), nil
				}
				if !tc.success || n != len(tc.codes)+1 {
					t.Error("finite support workflow sent an unexpected request", n, req.URL)
					return vsaResponse(req, 500, nil, "unexpected"), nil
				}
				vmrRequest(t, req, "id", `{"os-migrate_volume":{"cluster":""}}`, "3.60")
				return vsaResponse(req, 399, []byte{0xff, 0x00}, "action only"), nil
			})
			result, err := blockstorage.MigrateVolume(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, blockstorage.WithVolumeMigrationCluster(""))
			expectedPages := 0
			for i, code := range tc.codes {
				if code == 200 || code == 300 {
					if result == nil || expectedPages >= len(result.Discovery) || result.Discovery[expectedPages] == nil || result.Discovery[expectedPages].StatusCode != code || string(result.Discovery[expectedPages].Body) != tc.replies[i] || result.Discovery[expectedPages].Header.Get("X-Proof") != []string{"first discovery", "root discovery"}[i] {
						t.Fatal("only actual admitted discovery can be recorded", result, err, i)
					}
					expectedPages++
				}
			}
			if result == nil || result.VolumeID != "id" || len(result.Discovery) != expectedPages {
				t.Fatal(result, err)
			}
			if tc.success {
				if err != nil || !result.Completed || result.Microversion != "3.60" || result.Applied == nil || result.Applied.StatusCode != 399 || !bytes.Equal(result.Applied.Body, []byte{0xff, 0x00}) || result.Applied.Header.Get("X-Proof") != "action only" || calls.Load() != int32(len(tc.codes)+1) {
					t.Fatal(result, err, calls.Load())
				}
				return
			}
			vsaOperation(t, err, "MigrateVolume")
			var proof *resource.ResponseError
			var native gophercloud.ErrUnexpectedResponseCode
			if result.Completed || result.Applied != nil || calls.Load() != int32(len(tc.codes)) || errors.Is(err, resource.ErrUnsupported) != tc.unsupported {
				t.Fatal(result, err, calls.Load())
			}
			if tc.native != 0 {
				if !errors.As(err, &native) || native.Actual != tc.native || string(native.Body) != "native root" || native.ResponseHeader.Get("X-Proof") != "root discovery" || errors.As(err, &proof) || expectedPages != 0 {
					t.Fatal("terminal native rejection was fabricated as admission", result, err, native, proof)
				}
			} else {
				last := result.Discovery[len(result.Discovery)-1]
				if !errors.As(err, &proof) || proof.StatusCode != last.StatusCode || !bytes.Equal(proof.Body, last.Body) || proof.Header.Get("X-Proof") != last.Header.Get("X-Proof") {
					t.Fatal("terminal discovery did not retain its actual accepted proof", result, err, proof)
				}
				if len(proof.Body) != 0 {
					first := last.Body[0]
					proof.Body[0] = '!'
					if last.Body[0] != first {
						t.Fatal("discovery result aliases its ResponseError")
					}
				}
				proof.Header.Set("X-Proof", "caller error mutation")
				if last.Header.Get("X-Proof") == "caller error mutation" {
					t.Fatal("discovery error headers alias recorded page")
				}
			}
		})
	}
	for _, code := range []int{300, 404} {
		for _, kind := range []string{"read", "close", "both"} {
			t.Run("discovery fault "+http.StatusText(code)+" "+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.60")
				readCause, closeCause := errors.New("required discovery Read"), errors.New("required discovery Close")
				raw := []byte(good)
				var calls atomic.Int32
				cloud.Provider.HTTPClient.Transport = vsaTransport(func(req *http.Request) (*http.Response, error) {
					calls.Add(1)
					vsaDiscoveryRequest(t, req, vsaVersionPath)
					broken := &vsaFaultBody{reader: bytes.NewReader(raw)}
					if kind != "close" {
						broken.readErr = readCause
					}
					if kind != "read" {
						broken.closeErr = closeCause
					}
					response := vsaResponse(req, code, nil, "faulty support")
					response.Body = broken
					response.ContentLength = int64(len(raw))
					return response, nil
				})
				result, err := blockstorage.MigrateVolume(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, blockstorage.WithVolumeMigrationCluster(""))
				vsaOperation(t, err, "MigrateVolume")
				var proof *resource.ResponseError
				var native gophercloud.ErrUnexpectedResponseCode
				if result == nil || result.Completed || result.Applied != nil || calls.Load() != 1 || (kind != "close" && !errors.Is(err, readCause)) || (kind != "read" && !errors.Is(err, closeCause)) {
					t.Fatal("faulty support response was retried, posted or swallowed", result, err, calls.Load())
				}
				if code == 300 {
					if len(result.Discovery) != 1 || !errors.As(err, &proof) || proof.StatusCode != 300 || !bytes.Equal(proof.Body, raw) || proof.Header.Get("X-Proof") != "faulty support" || !bytes.Equal(result.Discovery[0].Body, raw) {
						t.Fatal(result, err, proof)
					}
				} else if len(result.Discovery) != 0 || errors.As(err, &proof) || !errors.As(err, &native) || native.Actual != 404 || !bytes.Equal(native.Body, raw) || native.ResponseHeader.Get("X-Proof") != "faulty support" {
					t.Fatal("faulty native 404 authorized fallback or acquired invented admission", result, err, native, proof)
				}
			})
		}
	}
}

type vmrOriginalAction struct {
	operation, body string
	call            func(context.Context, *gophercloud.ServiceClient, string, func() error, func() error) (*blockstorage.VolumeActionResult, error)
}

func vmrOriginalActions() []vmrOriginalAction {
	return []vmrOriginalAction{
		{"ResetVolumeStatus", `{"os-reset_status":{}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string, first, later func() error) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.ResetVolumeStatus(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, func(*blockstorage.VolumeStatusResetOpts) error { return first() }, func(*blockstorage.VolumeStatusResetOpts) error { return later() })
		}},
		{"MigrateVolume", `{"os-migrate_volume":{}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string, first, later func() error) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.MigrateVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, func(*blockstorage.VolumeMigrationOpts) error { return first() }, func(*blockstorage.VolumeMigrationOpts) error { return later() })
		}},
		{"CompleteVolumeMigration", `{"os-migrate_volume_completion":{"error":false,"new_volume":"new"}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string, first, later func() error) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.CompleteVolumeMigration(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "new", func(*blockstorage.VolumeMigrationCompletionOpts) error { return first() }, func(*blockstorage.VolumeMigrationCompletionOpts) error { return later() })
		}},
	}
}

func TestVolumeMigrationOriginalOptionsOwnFactoryPointersRetainedConfigsAndCallbackOrder(t *testing.T) {
	for _, kind := range []string{"reset", "migration", "completion"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.60")
			var calls, callbacks atomic.Int32
			var order []int32
			record := func(label int32) { callbacks.Add(1); order = append(order, label) }
			want := map[string]string{
				"reset":      `{"os-reset_status":{"attach_status":"original attach","migration_status":"original migration","status":"original status"}}`,
				"migration":  `{"os-migrate_volume":{"cluster":"","force_host_copy":true,"host":"original host","lock_volume":true}}`,
				"completion": `{"os-migrate_volume_completion":{"error":true,"new_volume":"new"}}`,
			}[kind]
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				n := calls.Add(1)
				if kind == "migration" && n == 1 {
					vsaDiscoveryRequest(t, req, vsaVersionPath)
					testcloud.JSON(w, 200, `{"id":"v3.0","min_version":"3.0","max_version":"3.99"}`)
					return
				}
				vmrRequest(t, req, "id", want, "3.60")
				testcloud.JSON(w, 203, "opaque")
			})
			ctx := vsaContext(t)
			var result *blockstorage.VolumeActionResult
			var err error
			switch kind {
			case "reset":
				status, attach, migration := "original status", "original attach", "original migration"
				factory := blockstorage.WithVolumeStatusResetOptions(blockstorage.VolumeStatusResetOpts{Status: &status, AttachStatus: &attach, MigrationStatus: &migration})
				status, attach, migration = string([]byte{0xff}), "changed attach", "changed migration"
				var retained *blockstorage.VolumeStatusResetOpts
				var opts []blockstorage.VolumeStatusResetOption
				opts = []blockstorage.VolumeStatusResetOption{
					func(*blockstorage.VolumeStatusResetOpts) error {
						record(1)
						opts[3] = func(*blockstorage.VolumeStatusResetOpts) error { return errors.New("replaced later original") }
						return nil
					},
					func(o *blockstorage.VolumeStatusResetOpts) error { record(2); return factory(o) },
					func(o *blockstorage.VolumeStatusResetOpts) error { record(3); retained = o; return nil },
					func(*blockstorage.VolumeStatusResetOpts) error {
						record(4)
						*retained.Status = string([]byte{0xff})
						*retained.AttachStatus = "late attach"
						*retained.MigrationStatus = "late migration"
						client.MoreHeaders["x-source"] = "later ordinary header"
						return nil
					},
				}
				result, err = blockstorage.ResetVolumeStatus(ctx, client, blockstorage.VolumeActionRequest{VolumeID: "id"}, opts...)
			case "migration":
				host, cluster := "original host", ""
				force, lock := true, true
				factory := blockstorage.WithVolumeMigrationOptions(blockstorage.VolumeMigrationOpts{Host: &host, Cluster: &cluster, ForceHostCopy: &force, LockVolume: &lock})
				host, cluster, force, lock = string([]byte{0xff}), "late outer cluster", false, false
				var retained *blockstorage.VolumeMigrationOpts
				var opts []blockstorage.VolumeMigrationOption
				opts = []blockstorage.VolumeMigrationOption{
					func(*blockstorage.VolumeMigrationOpts) error {
						record(1)
						opts[3] = func(*blockstorage.VolumeMigrationOpts) error { return errors.New("replaced later original") }
						return nil
					},
					func(o *blockstorage.VolumeMigrationOpts) error { record(2); return factory(o) },
					func(o *blockstorage.VolumeMigrationOpts) error { record(3); retained = o; return nil },
					func(*blockstorage.VolumeMigrationOpts) error {
						record(4)
						*retained.Host = string([]byte{0xff})
						*retained.Cluster = "late retained cluster"
						*retained.ForceHostCopy = false
						*retained.LockVolume = false
						client.MoreHeaders["x-source"] = "later ordinary header"
						return nil
					},
				}
				result, err = blockstorage.MigrateVolume(ctx, client, blockstorage.VolumeActionRequest{VolumeID: "id"}, opts...)
			case "completion":
				failed := true
				factory := blockstorage.WithVolumeMigrationCompletionOptions(blockstorage.VolumeMigrationCompletionOpts{Error: &failed})
				failed = false
				var retained *blockstorage.VolumeMigrationCompletionOpts
				var opts []blockstorage.VolumeMigrationCompletionOption
				opts = []blockstorage.VolumeMigrationCompletionOption{
					func(*blockstorage.VolumeMigrationCompletionOpts) error {
						record(1)
						opts[3] = func(*blockstorage.VolumeMigrationCompletionOpts) error { return errors.New("replaced later original") }
						return nil
					},
					func(o *blockstorage.VolumeMigrationCompletionOpts) error { record(2); return factory(o) },
					func(o *blockstorage.VolumeMigrationCompletionOpts) error { record(3); retained = o; return nil },
					func(*blockstorage.VolumeMigrationCompletionOpts) error {
						record(4)
						*retained.Error = false
						client.MoreHeaders["x-source"] = "later ordinary header"
						return nil
					},
				}
				result, err = blockstorage.CompleteVolumeMigration(ctx, client, blockstorage.VolumeActionRequest{VolumeID: "id"}, "new", opts...)
			}
			count := int32(1)
			if kind == "migration" {
				count = 2
			}
			if err != nil || result == nil || !result.Completed || result.Applied == nil || callbacks.Load() != 4 || len(order) != 4 || order[0] != 1 || order[1] != 2 || order[2] != 3 || order[3] != 4 || calls.Load() != count || client.MoreHeaders["x-source"] != "later ordinary header" {
				t.Fatal("original callbacks, retained config or factory inputs changed owned request policy", result, err, calls.Load(), callbacks.Load(), order)
			}
		})
	}
}

func TestVolumeMigrationActionsCaptureSourcesBeforeOriginalsAndStopBeforeRestoration(t *testing.T) {
	for _, action := range vmrOriginalActions() {
		for _, fact := range []string{"provider", "endpoint", "resource base", "service type", "microversion"} {
			t.Run(action.operation+" "+fact, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.60")
				var calls, firstCalls, laterCalls atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
					calls.Add(1)
					t.Error("changed captured source sent HTTP", req.URL)
					w.WriteHeader(500)
				})
				saved := *client
				first := func() error {
					firstCalls.Add(1)
					switch fact {
					case "provider":
						client.ProviderClient = &gophercloud.ProviderClient{}
					case "endpoint":
						client.Endpoint += "changed/"
					case "resource base":
						client.ResourceBase += "changed/"
					case "service type":
						client.Type = "volumev3"
					case "microversion":
						client.Microversion = "3.61"
					}
					return nil
				}
				later := func() error { laterCalls.Add(1); *client = saved; return nil }
				result, err := action.call(vsaContext(t), client, "id", first, later)
				vsaOperation(t, err, action.operation)
				var proof *resource.ResponseError
				if result != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || firstCalls.Load() != 1 || laterCalls.Load() != 0 || calls.Load() != 0 {
					t.Fatal("later original repaired an already changed source", result, err, firstCalls.Load(), laterCalls.Load(), calls.Load())
				}
			})
		}
	}
}

func TestVolumeMigrationActionsPreflightAndCallbackFailuresPreservePriorityAndCauses(t *testing.T) {
	for _, action := range vmrOriginalActions() {
		for _, kind := range []string{"nil context", "canceled context", "nil client", "nil provider", "wrong service", "reserved auth", "empty ID", "path ID", "escaped ID", "space ID", "control ID", "invalid UTF8 ID"} {
			t.Run(action.operation+" "+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.60")
				ctx := vsaContext(t)
				id := "id"
				callerCause := errors.New("migration preflight custom cancellation")
				expected := resource.ErrInvalidOption
				switch kind {
				case "nil context":
					ctx = nil
				case "canceled context":
					c, cancel := context.WithCancelCause(ctx)
					cancel(callerCause)
					ctx = c
					expected = context.Canceled
				case "nil client":
					client = nil
				case "nil provider":
					client.ProviderClient = nil
				case "wrong service":
					client.Type = "compute"
					expected = resource.ErrUnsupported
				case "reserved auth":
					client.MoreHeaders["Authorization"] = "caller auth"
				case "empty ID":
					id = ""
				case "path ID":
					id = "a/b"
				case "escaped ID":
					id = "%2f"
				case "space ID":
					id = "a b"
				case "control ID":
					id = "a\x00b"
				case "invalid UTF8 ID":
					id = string([]byte{0xff})
				}
				var calls, originals atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
					calls.Add(1)
					t.Error("invalid source or route sent HTTP", req.URL)
					w.WriteHeader(500)
				})
				callback := func() error { originals.Add(1); return nil }
				result, err := action.call(ctx, client, id, callback, callback)
				vsaOperation(t, err, action.operation)
				var proof *resource.ResponseError
				if result != nil || !errors.Is(err, expected) || errors.As(err, &proof) || calls.Load() != 0 || originals.Load() != 0 || (kind == "canceled context" && !errors.Is(err, callerCause)) {
					t.Fatal("source/ID preflight invoked originals or lost cause", result, err, calls.Load(), originals.Load())
				}
			})
		}
		for _, kind := range []string{"callback error", "callback error and cancellation and source"} {
			t.Run(action.operation+" "+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.60")
				ctx, cancel := context.WithCancelCause(vsaContext(t))
				defer cancel(nil)
				callbackCause, callerCause := errors.New("migration original callback"), errors.New("migration callback canceled caller")
				var calls, firstCalls, laterCalls atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
					calls.Add(1)
					t.Error("failed original sent HTTP", req.URL)
					w.WriteHeader(500)
				})
				first := func() error {
					firstCalls.Add(1)
					if kind != "callback error" {
						client.ResourceBase += "changed/"
						cancel(callerCause)
					}
					return callbackCause
				}
				later := func() error { laterCalls.Add(1); return nil }
				result, err := action.call(ctx, client, "id", first, later)
				vsaOperation(t, err, action.operation)
				var proof *resource.ResponseError
				if result != nil || !errors.Is(err, callbackCause) || errors.As(err, &proof) || calls.Load() != 0 || firstCalls.Load() != 1 || laterCalls.Load() != 0 {
					t.Fatal(result, err, calls.Load(), firstCalls.Load(), laterCalls.Load())
				}
				if kind != "callback error" && (!errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, context.Canceled) || !errors.Is(err, callerCause)) {
					t.Fatal("source/callback/custom cancellation were not all retained", err)
				}
			})
		}
	}
	t.Run("required new volume invalid UTF8 precedes originals", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := vsaClient(cloud, "3.60")
		var calls, originals atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) { calls.Add(1); w.WriteHeader(500) })
		result, err := blockstorage.CompleteVolumeMigration(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, string([]byte{0xff}), func(*blockstorage.VolumeMigrationCompletionOpts) error { originals.Add(1); return nil })
		vsaOperation(t, err, "CompleteVolumeMigration")
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || originals.Load() != 0 || calls.Load() != 0 {
			t.Fatal(result, err, originals.Load(), calls.Load())
		}
	})
	t.Run("final optional text invalid UTF8 does not reach HTTP", func(t *testing.T) {
		for _, kind := range []string{"status", "host", "cluster"} {
			t.Run(kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.60")
				var calls, originals atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) { calls.Add(1); w.WriteHeader(500) })
				bad := string([]byte{0xff})
				var result *blockstorage.VolumeActionResult
				var err error
				operation := "MigrateVolume"
				if kind == "status" {
					operation = "ResetVolumeStatus"
					result, err = blockstorage.ResetVolumeStatus(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, func(o *blockstorage.VolumeStatusResetOpts) error { originals.Add(1); o.Status = &bad; return nil })
				} else {
					result, err = blockstorage.MigrateVolume(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, func(o *blockstorage.VolumeMigrationOpts) error {
						originals.Add(1)
						if kind == "host" {
							o.Host = &bad
						} else {
							o.Cluster = &bad
						}
						return nil
					})
				}
				vsaOperation(t, err, operation)
				if result != nil || !errors.Is(err, resource.ErrInvalidOption) || originals.Load() != 1 || calls.Load() != 0 {
					t.Fatal(result, err, originals.Load(), calls.Load())
				}
			})
		}
	})
	t.Run("nil originals are local errors", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := vsaClient(cloud, "3.60")
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) { calls.Add(1); w.WriteHeader(500) })
		results := make([]*blockstorage.VolumeActionResult, 3)
		errs := make([]error, 3)
		results[0], errs[0] = blockstorage.ResetVolumeStatus(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, nil)
		results[1], errs[1] = blockstorage.MigrateVolume(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, nil)
		results[2], errs[2] = blockstorage.CompleteVolumeMigration(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, "new", nil)
		for i, operation := range []string{"ResetVolumeStatus", "MigrateVolume", "CompleteVolumeMigration"} {
			vsaOperation(t, errs[i], operation)
			if results[i] != nil || !errors.Is(errs[i], resource.ErrInvalidOption) {
				t.Fatal(results[i], errs[i])
			}
		}
		if calls.Load() != 0 {
			t.Fatal(calls.Load())
		}
	})
}

func TestVolumeMigrationActionsKeepOriginalNativePolicyWhenRetryExpandsHTTPAdmission(t *testing.T) {
	for _, action := range vmrOriginalActions() {
		for _, expanded := range []bool{false, true} {
			t.Run(action.operation+map[bool]string{false: " native 400", true: " expanded retry 400"}[expanded], func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.60")
				var calls, retries atomic.Int32
				if expanded {
					cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, original error, _ uint) error {
						if !gophercloud.ResponseCodeIs(original, 503) || retries.Add(1) > 1 {
							return original
						}
						opts.OkCodes = append(opts.OkCodes, 400)
						return nil
					}
				}
				raw := []byte{0xff, 0x00, 'r'}
				cloud.Provider.HTTPClient.Transport = vsaTransport(func(req *http.Request) (*http.Response, error) {
					n := calls.Add(1)
					vmrRequest(t, req, "id", action.body, "3.60")
					code := 400
					if expanded && n == 1 {
						code = 503
					}
					return vsaResponse(req, code, raw, "current rejection"), nil
				})
				result, err := action.call(vsaContext(t), client, "id", func() error { return nil }, func() error { return nil })
				vsaOperation(t, err, action.operation)
				var native gophercloud.ErrUnexpectedResponseCode
				var proof *resource.ResponseError
				expectedCalls := int32(1)
				if expanded {
					expectedCalls = 2
				}
				if result == nil || result.VolumeID != "id" || result.Microversion != "3.60" || result.Completed || result.Applied != nil || len(result.Discovery) != 0 || !errors.As(err, &native) || native.Actual != 400 || len(native.Expected) != 300 || native.Expected[0] != 100 || native.Expected[299] != 399 || !bytes.Equal(native.Body, raw) || native.ResponseHeader.Get("X-Proof") != "current rejection" || errors.As(err, &proof) || calls.Load() != expectedCalls || retries.Load() != expectedCalls-1 {
					t.Fatal("native status expansion invented SDK acknowledgement", result, err, native, proof, calls.Load(), retries.Load())
				}
				native.Body[0] = '!'
				native.ResponseHeader.Set("X-Proof", "caller changed")
				if raw[0] != 0xff {
					t.Fatal("native error aliases fixture response body")
				}
			})
		}
	}
}

func TestVolumeMigrationServiceMethodsKeepSDKContractsAndNilAPIPriority(t *testing.T) {
	for _, tc := range []struct {
		operation, body string
		cluster         bool
		call            func(context.Context, *volumes.API, *atomic.Int32) (*volumes.VolumeActionResult, error)
	}{
		{"ResetVolumeStatus", `{"os-reset_status":{"status":"server-defined"}}`, false, func(ctx context.Context, api *volumes.API, count *atomic.Int32) (*volumes.VolumeActionResult, error) {
			return api.ResetVolumeStatus(ctx, "id", volumes.WithVolumeStatusResetStatus("server-defined"), func(*volumes.VolumeStatusResetOpts) error { count.Add(1); return nil })
		}},
		{"MigrateVolume", `{"os-migrate_volume":{"cluster":"","host":"host@backend","lock_volume":true}}`, true, func(ctx context.Context, api *volumes.API, count *atomic.Int32) (*volumes.VolumeActionResult, error) {
			return api.MigrateVolume(ctx, "id", volumes.WithVolumeMigrationCluster(""), volumes.WithVolumeMigrationHost("host@backend"), volumes.WithVolumeMigrationLockVolume(true), func(*volumes.VolumeMigrationOpts) error { count.Add(1); return nil })
		}},
		{"CompleteVolumeMigration", `{"os-migrate_volume_completion":{"error":true,"new_volume":"new /?%#"}}`, false, func(ctx context.Context, api *volumes.API, count *atomic.Int32) (*volumes.VolumeActionResult, error) {
			return api.CompleteVolumeMigration(ctx, "id", "new /?%#", volumes.WithVolumeMigrationCompletionError(true), func(*volumes.VolumeMigrationCompletionOpts) error { count.Add(1); return nil })
		}},
	} {
		t.Run(tc.operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.90")
			api := volumes.New(client)
			var calls, originals atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				n := calls.Add(1)
				if tc.cluster && n == 1 {
					vsaDiscoveryRequest(t, req, vsaVersionPath)
					w.Header().Set("X-Proof", "service support")
					testcloud.JSON(w, 200, `{"id":"v3.0","min_version":"3.0","max_version":"3.20"}`)
					return
				}
				vmrRequest(t, req, "id", tc.body, "3.90")
				w.Header().Set("X-Proof", "service action")
				w.WriteHeader(399)
				_, _ = w.Write([]byte{0xff, 0x00})
			})
			result, err := tc.call(vsaContext(t), api, &originals)
			count := int32(1)
			if tc.cluster {
				count = 2
			}
			if err != nil || result == nil || result.VolumeID != "id" || !result.Completed || result.Microversion != "3.90" || result.Applied == nil || result.Applied.StatusCode != 399 || !bytes.Equal(result.Applied.Body, []byte{0xff, 0x00}) || result.Applied.Header.Get("X-Proof") != "service action" || len(result.Discovery) != int(count-1) || calls.Load() != count || originals.Load() != 1 {
				t.Fatal(result, err, calls.Load(), originals.Load())
			}
			var absent *volumes.API
			before := calls.Load()
			originals.Store(0)
			missing, err := tc.call(vsaContext(t), absent, &originals)
			vsaOperation(t, err, tc.operation)
			var proof *resource.ResponseError
			if missing != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || originals.Load() != 0 || calls.Load() != before {
				t.Fatal("nil service API invoked originals or HTTP", missing, err, proof, originals.Load(), calls.Load())
			}
		})
	}
}
