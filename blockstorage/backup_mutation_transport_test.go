package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestVolumeBackupMutationAcceptedFailuresRetainOnlyCurrentProofAndAllCauses(t *testing.T) {
	for _, phase := range []string{"create", "create poll", "delete", "force delete", "delete poll"} {
		for _, fault := range []string{"read", "close", "cancel", "joined read close cancel source"} {
			t.Run(phase+"/"+fault, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := bmwClient(cloud)
				ctx, cancel := context.WithCancelCause(bmwContext(t))
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("backup Read"), errors.New("backup Close"), errors.New("backup cancellation")
				step, code := int32(1), 203
				reply := `{"backup":{"id":"actual","status":"available"}}`
				if phase == "create poll" {
					step, code = 2, 200
				}
				if phase == "delete" || phase == "force delete" {
					step, code, reply = 2, 202, string([]byte{0xff, 0, 'x'})
				}
				if phase == "delete poll" {
					step, code, reply = 3, 200, `{"backup":{"status":"deleted"}}`
				}
				broken := &snapshotReadTransportBody{data: strings.NewReader(reply)}
				switch fault {
				case "read":
					broken.readError = readCause
				case "close":
					broken.closeError = closeCause
				case "cancel":
					broken.onClose = func() { cancel(cancelCause) }
				default:
					broken.readError = readCause
					broken.closeError = closeCause
					broken.onClose = func() { client.Microversion = "3.99"; cancel(cancelCause) }
				}
				var calls, retries atomic.Int32
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
					retries.Add(1)
					return original
				}
				cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
					n := calls.Add(1)
					if n == step {
						method, path, version := "POST", bmwCollection, "3.60"
						if phase == "create poll" {
							method, path = "GET", bmwCollection+"/actual"
						}
						if phase == "delete" {
							method, path = "DELETE", bmwCollection+"/actual"
						}
						if phase == "force delete" {
							path, version = bmwCollection+"/actual/action", "3.64"
						}
						if phase == "delete poll" {
							method, path = "GET", bmwCollection+"/actual"
						}
						bmwRequest(t, r, method, path, version, "test-token")
						return snapshotReadTransportResponse(r, code, broken, "current"), nil
					}
					if n == 1 {
						if phase == "create poll" {
							bmwRequest(t, r, "POST", bmwCollection, "3.60", "test-token")
						} else {
							bmwRequest(t, r, "GET", bmwCollection+"/requested", "3.60", "test-token")
						}
						return snapshotReadTransportResponse(r, 202, io.NopCloser(strings.NewReader(`{"backup":{"id":"actual","status":"creating"}}`)), "prior"), nil
					}
					if n == 2 && phase == "delete poll" {
						bmwRequest(t, r, "DELETE", bmwCollection+"/actual", "3.60", "test-token")
						return snapshotReadTransportResponse(r, 204, io.NopCloser(strings.NewReader("")), "applied"), nil
					}
					t.Error("accepted failure replayed", r.Method, r.URL)
					return nil, errors.New("unexpected request")
				})
				var proof *blockstorage.VolumeBackupMutationPage
				var err error
				if strings.HasPrefix(phase, "create") {
					result, failed := blockstorage.CreateVolumeBackup(ctx, client, blockstorage.CreateVolumeBackupRequest{VolumeID: "volume"})
					err = failed
					if result == nil || result.Value != nil || result.Ready != nil || result.Created == nil {
						t.Fatal(result, err)
					}
					proof = result.LastAccepted
					if phase == "create poll" && (result.CreatedValue == nil || result.Created.Header.Get("X-Proof") != "prior") {
						t.Fatal(result)
					}
				} else {
					result, failed := blockstorage.DeleteVolumeBackup(ctx, client, blockstorage.DeleteVolumeBackupRequest{NameOrID: "requested"}, blockstorage.WithDeleteVolumeBackupForce(phase == "force delete"), blockstorage.WithDeleteVolumeBackupWait(true))
					err = failed
					if result == nil || result.Deleted != nil || result.Ready != nil || result.Resolved == nil || result.Resolved.Value == nil || result.Applied == nil {
						t.Fatal(result, err)
					}
					proof = result.LastAccepted
					if phase == "delete poll" && result.Applied.Header.Get("X-Proof") != "applied" {
						t.Fatal(result)
					}
				}
				var physical *resource.ResponseError
				if err == nil || !errors.As(err, &physical) || physical.StatusCode != code || string(physical.Body) != reply || physical.Header.Get("X-Proof") != "current" || calls.Load() != step || retries.Load() != 0 || broken.closes.Load() != 1 {
					t.Fatal(err, proof, calls.Load(), retries.Load(), broken.closes.Load())
				}
				bmwPage(t, proof, code, reply, "current")
				if (fault == "read" || strings.HasPrefix(fault, "joined")) && !errors.Is(err, readCause) {
					t.Fatal(err)
				}
				if (fault == "close" || strings.HasPrefix(fault, "joined")) && !errors.Is(err, closeCause) {
					t.Fatal(err)
				}
				if (fault == "cancel" || strings.HasPrefix(fault, "joined")) && (!errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled)) {
					t.Fatal(err)
				}
				if strings.HasPrefix(fault, "joined") && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				proof.Body[0] = '!'
				proof.Header.Set("X-Proof", "changed")
				if !bytes.Equal(physical.Body, []byte(reply)) || physical.Header.Get("X-Proof") != "current" {
					t.Fatal("result proof aliases error proof", physical)
				}
			})
		}
	}
}

func TestDeleteVolumeBackupPolling404RequiresCleanBodyAndOriginalNativePolicy(t *testing.T) {
	for _, fault := range []string{"clean", "read", "close", "read and close", "Close EOF", "callback wrapper", "callback swallows fault", "expanded404"} {
		t.Run(fault, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			readCause, closeCause := errors.New("poll404 Read"), errors.New("poll404 Close")
			body := `{"error":"gone"}`
			rejected := &snapshotReadTransportBody{data: strings.NewReader(body)}
			if fault == "read" || fault == "read and close" || fault == "callback swallows fault" {
				rejected.readError = readCause
			}
			if fault == "close" || fault == "read and close" {
				rejected.closeError = closeCause
			}
			if fault == "Close EOF" {
				rejected.closeError = io.EOF
			}
			var calls, retries atomic.Int32
			if strings.HasPrefix(fault, "callback") || fault == "expanded404" {
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, original error, count uint) error {
					n := retries.Add(1)
					if n > 1 || count != 1 {
						return errors.Join(original, errors.New("bounded retry exceeded"))
					}
					var native gophercloud.ErrUnexpectedResponseCode
					want := 404
					if fault == "expanded404" {
						want = 500
					}
					if !errors.As(original, &native) || native.Actual != want {
						t.Error(original)
					}
					if fault == "callback wrapper" {
						return fmt.Errorf("caller wraps native: %w", original)
					}
					if fault == "expanded404" {
						o.OkCodes = append(o.OkCodes, 404)
					}
					return nil
				}
			}
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
				switch n := calls.Add(1); n {
				case 1:
					return snapshotReadTransportResponse(r, 200, io.NopCloser(strings.NewReader(`{"backup":{"id":"actual","status":"available"}}`)), "resolved"), nil
				case 2:
					bmwRequest(t, r, "DELETE", bmwCollection+"/actual", "3.60", "test-token")
					return snapshotReadTransportResponse(r, 204, io.NopCloser(strings.NewReader("")), "applied"), nil
				case 3:
					bmwRequest(t, r, "GET", bmwCollection+"/actual", "3.60", "test-token")
					if fault == "expanded404" {
						return snapshotReadTransportResponse(r, 500, io.NopCloser(strings.NewReader(`{"error":"retry"}`)), "retry"), nil
					}
					return snapshotReadTransportResponse(r, 404, rejected, "absent"), nil
				case 4:
					if fault == "expanded404" {
						return snapshotReadTransportResponse(r, 404, rejected, "absent"), nil
					}
					t.Error("faulty native poll was replayed")
					return snapshotReadTransportResponse(r, 200, io.NopCloser(strings.NewReader(`{"backup":{"status":"deleted"}}`)), "unexpected replay"), nil
				default:
					return nil, errors.New("unexpected poll")
				}
			})
			result, err := bmwDelete(t, client, blockstorage.WithDeleteVolumeBackupWait(true))
			expectedCalls := int32(3)
			if fault == "expanded404" {
				expectedCalls = 4
			}
			if result == nil || result.Resolved == nil || result.Applied == nil || calls.Load() != expectedCalls || rejected.closes.Load() != 1 {
				t.Fatal(result, err, calls.Load(), rejected.closes.Load())
			}
			bmwPage(t, result.LastAccepted, 204, "", "applied")
			if fault == "clean" {
				if err != nil || result.Deleted == nil || !*result.Deleted || result.Absent == nil || result.Ready != nil || result.ReadyBackup != nil {
					t.Fatal(result, err)
				}
				bmwPage(t, result.Absent, 404, body, "absent")
				return
			}
			var native gophercloud.ErrUnexpectedResponseCode
			var physical *resource.ResponseError
			if err == nil || result.Deleted != nil || result.Absent != nil || result.Ready != nil || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != body || errors.As(err, &physical) {
				t.Fatal(result, err)
			}
			if (fault == "read" || fault == "read and close" || fault == "callback swallows fault") && !errors.Is(err, readCause) {
				t.Fatal(err)
			}
			if (fault == "close" || fault == "read and close") && !errors.Is(err, closeCause) {
				t.Fatal(err)
			}
			if fault == "Close EOF" && !errors.Is(err, io.EOF) {
				t.Fatal(err)
			}
			if (strings.HasPrefix(fault, "callback") || fault == "expanded404") && retries.Load() != 1 {
				t.Fatal(retries.Load())
			}
		})
	}
}

func TestVolumeBackupMutationNativeAuthAndRetryUseLiveTokenWithOwnedHeadersAndVersion(t *testing.T) {
	for _, entry := range []string{"create", "force action", "delete poll"} {
		for _, mode := range []string{"reauth", "retry"} {
			t.Run(entry+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := bmwClient(cloud)
				var calls, retries, reauths atomic.Int32
				cloud.Provider.ReauthFunc = func(context.Context) error { reauths.Add(1); cloud.Provider.SetToken("reauth-token"); return nil }
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, count uint) error {
					if retries.Add(1) > 1 || count != 1 {
						return errors.Join(original, errors.New("bounded backup retry"))
					}
					cloud.Provider.SetToken("retry-token")
					return nil
				}
				failingStep := int32(1)
				if entry == "force action" {
					failingStep = 2
				}
				if entry == "delete poll" {
					failingStep = 3
				}
				cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
					n := calls.Add(1)
					method, path, version := "POST", bmwCollection, "3.60"
					if entry != "create" {
						method, path = "GET", bmwCollection+"/requested"
						if n >= 2 {
							if entry == "force action" {
								method, path, version = "POST", bmwCollection+"/actual/action", "3.64"
							} else if n == 2 {
								method, path = "DELETE", bmwCollection+"/actual"
							} else {
								method, path = "GET", bmwCollection+"/actual"
							}
						}
					}
					token := "test-token"
					if n == failingStep+1 {
						token = map[string]string{"reauth": "reauth-token", "retry": "retry-token"}[mode]
					}
					raw := bmwRequest(t, r, method, path, version, token)
					if method == "POST" {
						if entry == "force action" {
							if string(raw) != `{"os-force_delete":null}` {
								t.Error(string(raw))
							}
						} else {
							payload := smwFields(t, smwFields(t, raw)["backup"])
							if len(payload) != 6 || string(payload["name"]) != `"owned"` || string(payload["force"]) != "false" {
								t.Error(payload)
							}
						}
					}
					if n == failingStep {
						client.MoreHeaders["x-source"] = "later ordinary mutation"
						code := 503
						if mode == "reauth" {
							code = 401
						}
						return snapshotReadTransportResponse(r, code, io.NopCloser(strings.NewReader(`{"error":"native retry"}`)), "rejected"), nil
					}
					if n == failingStep+1 {
						if entry == "create" {
							return snapshotReadTransportResponse(r, 202, io.NopCloser(strings.NewReader(`{"backup":{"id":"created","status":"available"}}`)), "ready"), nil
						}
						if entry == "force action" {
							return snapshotReadTransportResponse(r, 203, io.NopCloser(strings.NewReader(string([]byte{0xff, 0, 'x'}))), "applied"), nil
						}
						return snapshotReadTransportResponse(r, 200, io.NopCloser(strings.NewReader(`{"backup":{"status":"deleted"}}`)), "ready"), nil
					}
					if n == 1 {
						return snapshotReadTransportResponse(r, 200, io.NopCloser(strings.NewReader(`{"backup":{"id":"actual","status":"available"}}`)), "resolved"), nil
					}
					if n == 2 && entry == "delete poll" {
						return snapshotReadTransportResponse(r, 204, io.NopCloser(strings.NewReader("")), "applied"), nil
					}
					return nil, errors.New("unexpected auth request")
				})
				if entry == "create" {
					result, err := bmwCreate(t, client, blockstorage.WithCreateVolumeBackupName("owned"))
					if err != nil || result == nil || result.Ready == nil {
						t.Fatal(result, err)
					}
				} else {
					result, err := bmwDelete(t, client, blockstorage.WithDeleteVolumeBackupForce(entry == "force action"), blockstorage.WithDeleteVolumeBackupWait(entry == "delete poll"))
					if err != nil || result == nil || result.Deleted == nil || !*result.Deleted || result.Applied == nil {
						t.Fatal(result, err)
					}
				}
				if calls.Load() != failingStep+1 || (mode == "reauth" && (reauths.Load() != 1 || retries.Load() != 0)) || (mode == "retry" && (reauths.Load() != 0 || retries.Load() != 1)) || client.Microversion != "3.60" || client.MoreHeaders["x-source"] != "later ordinary mutation" {
					t.Fatal(calls.Load(), retries.Load(), reauths.Load(), client)
				}
			})
		}
	}
}

func TestVolumeBackupMutationRetryCannotReplaceOwnedBodyDecoderOrFixedScope(t *testing.T) {
	for _, entry := range []string{"create", "force action"} {
		for _, change := range []string{"body", "raw body", "decoder", "retention", "source", "generic version header", "legacy version header", "omit generic version", "omit legacy version", "delete generic version", "delete legacy version"} {
			if entry == "create" && strings.Contains(change, "version") {
				continue
			}
			t.Run(entry+"/"+change, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := bmwClient(cloud)
				var calls, retries atomic.Int32
				step := int32(1)
				rejectedCode := 500
				if entry == "force action" {
					step = 2
					rejectedCode = 503
				}
				cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
					n := calls.Add(1)
					if n < step {
						return snapshotReadTransportResponse(r, 200, io.NopCloser(strings.NewReader(`{"backup":{"id":"actual"}}`)), "resolved"), nil
					}
					path, version := bmwCollection, "3.60"
					if entry == "force action" {
						path, version = bmwCollection+"/actual/action", "3.64"
					}
					bmwRequest(t, r, "POST", path, version, "test-token")
					return snapshotReadTransportResponse(r, rejectedCode, io.NopCloser(strings.NewReader(`{"error":"original"}`)), "rejected"), nil
				})
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, original error, count uint) error {
					if retries.Add(1) > 1 || count != 1 {
						return errors.Join(original, errors.New("bounded ownership retry"))
					}
					switch change {
					case "body":
						o.JSONBody = json.RawMessage(`{"replacement":true}`)
					case "raw body":
						o.RawBody = strings.NewReader("changed")
					case "decoder":
						o.JSONResponse = &map[string]any{}
					case "retention":
						o.KeepResponseBody = false
					case "source":
						client.Endpoint += "changed/"
					case "generic version header":
						o.MoreHeaders = map[string]string{"openstack-api-version": "volume 3.80"}
					case "legacy version header":
						o.MoreHeaders = map[string]string{"X-OpenStack-Volume-API-Version": "3.80"}
					case "omit generic version":
						o.OmitHeaders = []string{"OpenStack-API-Version"}
					case "omit legacy version":
						o.OmitHeaders = []string{"x-openstack-volume-api-version"}
					case "delete generic version":
						for key := range o.MoreHeaders {
							if strings.EqualFold(key, "OpenStack-API-Version") {
								delete(o.MoreHeaders, key)
							}
						}
					case "delete legacy version":
						for key := range o.MoreHeaders {
							if strings.EqualFold(key, "X-OpenStack-Volume-API-Version") {
								delete(o.MoreHeaders, key)
							}
						}
					}
					return nil
				}
				var err error
				if entry == "create" {
					result, failed := bmwCreate(t, client)
					err = failed
					if result == nil || result.Created != nil || result.Value != nil {
						t.Fatal(result, err)
					}
				} else {
					result, failed := bmwDelete(t, client, blockstorage.WithDeleteVolumeBackupForce(true))
					err = failed
					if result == nil || result.Resolved == nil || result.Applied != nil || result.LastAccepted != nil || result.Deleted != nil {
						t.Fatal(result, err)
					}
				}
				var native gophercloud.ErrUnexpectedResponseCode
				var physical *resource.ResponseError
				if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != rejectedCode || native.ResponseHeader.Get("X-Proof") != "rejected" || errors.As(err, &physical) || calls.Load() != step || retries.Load() != 1 {
					t.Fatal(err, calls.Load(), retries.Load())
				}
			})
		}
	}
	for _, change := range []string{"generic version", "legacy version", "missing version", "duplicate version"} {
		t.Run("force physical redirect/"+change, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var calls, redirects atomic.Int32
			cloud.Provider.HTTPClient.CheckRedirect = func(r *http.Request, _ []*http.Request) error {
				redirects.Add(1)
				switch change {
				case "generic version":
					r.Header.Set("OpenStack-API-Version", "volume 3.80")
				case "legacy version":
					r.Header.Set("X-OpenStack-Volume-API-Version", "3.80")
				case "missing version":
					r.Header.Del("OpenStack-API-Version")
				case "duplicate version":
					r.Header.Add("X-OpenStack-Volume-API-Version", "3.64")
				}
				return nil
			}
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
				switch calls.Add(1) {
				case 1:
					return snapshotReadTransportResponse(r, 200, io.NopCloser(strings.NewReader(`{"backup":{"id":"actual"}}`)), "resolved"), nil
				case 2:
					raw := bmwRequest(t, r, "POST", bmwCollection+"/actual/action", "3.64", "test-token")
					if string(raw) != `{"os-force_delete":null}` || len(r.Header.Values("OpenStack-API-Version")) != 1 || len(r.Header.Values("X-OpenStack-Volume-API-Version")) != 1 || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.64" {
						t.Error(string(raw), r.Header)
					}
					response := snapshotReadTransportResponse(r, 307, io.NopCloser(strings.NewReader("redirect")), "intermediate redirect")
					response.Header.Set("Location", client.ServiceURL("backups", "actual", "action"))
					return response, nil
				default:
					t.Error("header-mutated redirect physically resent")
					return snapshotReadTransportResponse(r, 204, io.NopCloser(strings.NewReader("")), "unexpected resend"), nil
				}
			})
			result, err := bmwDelete(t, client, blockstorage.WithDeleteVolumeBackupForce(true))
			var physical *resource.ResponseError
			if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &physical) || result == nil || result.Resolved == nil || result.Resolved.Value == nil || result.Applied != nil || result.LastAccepted != nil || result.Deleted != nil || calls.Load() != 2 || redirects.Load() != 1 || client.Microversion != "3.60" {
				t.Fatal(result, err, calls.Load(), redirects.Load())
			}
		})
	}
}

func TestVolumeBackupMutationExpandedNativeStatusNeverCreatesSuccessOrAbsenceProof(t *testing.T) {
	for _, entry := range []string{"create", "normal delete", "force delete"} {
		t.Run(entry, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var calls, retries atomic.Int32
			step := int32(1)
			if entry != "create" {
				step = 2
			}
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, original error, count uint) error {
				if retries.Add(1) > 1 || count != 1 {
					return errors.Join(original, errors.New("bounded status expansion"))
				}
				o.OkCodes = append(o.OkCodes, 404)
				return nil
			}
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if n < step {
					return snapshotReadTransportResponse(r, 200, io.NopCloser(strings.NewReader(`{"backup":{"id":"actual","status":"available"}}`)), "resolved"), nil
				}
				if n == step {
					return snapshotReadTransportResponse(r, 500, io.NopCloser(strings.NewReader(`{"error":"retry"}`)), "retry"), nil
				}
				if n == step+1 {
					return snapshotReadTransportResponse(r, 404, io.NopCloser(strings.NewReader(`{"error":"expanded"}`)), "expanded"), nil
				}
				return nil, errors.New("unexpected expanded request")
			})
			var err error
			if entry == "create" {
				result, failed := bmwCreate(t, client)
				err = failed
				if result == nil || result.Created != nil || result.LastAccepted != nil || result.Value != nil || result.Ready != nil {
					t.Fatal(result, err)
				}
			} else {
				result, failed := bmwDelete(t, client, blockstorage.WithDeleteVolumeBackupForce(entry == "force delete"), blockstorage.WithDeleteVolumeBackupWait(true))
				err = failed
				if result == nil || result.Resolved == nil || result.Applied != nil || result.LastAccepted != nil || result.Absent != nil || result.Ready != nil || result.Deleted != nil {
					t.Fatal(result, err)
				}
			}
			var native gophercloud.ErrUnexpectedResponseCode
			var physical *resource.ResponseError
			if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != `{"error":"expanded"}` || native.ResponseHeader.Get("X-Proof") != "expanded" || errors.As(err, &physical) || calls.Load() != step+1 || retries.Load() != 1 {
				t.Fatal(err, calls.Load(), retries.Load())
			}
		})
	}
}

func TestVolumeBackupMutationOriginalSourceChangesAreTerminalBeforeLaterRestoration(t *testing.T) {
	for _, entry := range []string{"create", "delete"} {
		for _, fact := range []string{"provider", "endpoint", "resource base", "type", "microversion"} {
			t.Run(entry+"/"+fact, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := bmwClient(cloud)
				saved := *client
				var calls atomic.Int32
				callbacks := 0
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
				mutate := func() { callbacks++; snapshotReadTransportMutate(client, fact) }
				restore := func() { callbacks++; *client = saved }
				var err error
				if entry == "create" {
					result, failed := bmwCreate(t, client, func(*blockstorage.CreateVolumeBackupOpts) error { mutate(); return nil }, func(*blockstorage.CreateVolumeBackupOpts) error { restore(); return nil })
					err = failed
					if result != nil {
						t.Fatal(result, err)
					}
				} else {
					result, failed := bmwDelete(t, client, func(*blockstorage.DeleteVolumeBackupOpts) error { mutate(); return nil }, func(*blockstorage.DeleteVolumeBackupOpts) error { restore(); return nil })
					err = failed
					if result != nil {
						t.Fatal(result, err)
					}
				}
				var physical *resource.ResponseError
				if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &physical) || callbacks != 1 || calls.Load() != 0 {
					t.Fatal(err, callbacks, calls.Load())
				}
			})
		}
	}
}
