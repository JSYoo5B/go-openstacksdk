package api_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	upcontainers "github.com/gophercloud/gophercloud/v2/openstack/keymanager/v1/containers"
	uporders "github.com/gophercloud/gophercloud/v2/openstack/keymanager/v1/orders"
	upsecrets "github.com/gophercloud/gophercloud/v2/openstack/keymanager/v1/secrets"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/keymanager/v1/containers"
	"gophercloudsdk/keymanager/v1/orders"
	"gophercloudsdk/keymanager/v1/secrets"
	"gophercloudsdk/resource"
)

// Reuse the existing leaf/cached-Connection fixtures and accepted-body wrappers.
// These are owned Remove/Resources.Delete receipts, not native alias changes.
func TestKeyManagerOwnedMetadataDeleteFixedTargetsAndPassiveBodies(t *testing.T) {
	for _, kind := range []string{"containers", "orders", "secrets"} {
		for fixtureIndex, fixtureName := range []string{"leaf", "cached connection"} {
			for _, lane := range []string{"Remove", "Resources.Delete"} {
				for _, status := range []int{200, 202, 204, 299, 399} {
					t.Run(fmt.Sprintf("%s/%s/%s/%d", kind, fixtureName, lane, status), func(t *testing.T) {
						c := testcloud.New(t)
						var calls atomic.Int32
						const id = "request-α+one"
						path := "/reverse/barbican/v1/" + kind + "/" + id
						c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
							calls.Add(1)
							th.TestMethod(t, r, http.MethodDelete)
							th.TestHeader(t, r, "X-Auth-Token", "owned-delete-token")
							th.TestHeader(t, r, "X-Source", "source")
							th.TestHeader(t, r, "OpenStack-API-Version", "key-manager 1.0")
							if r.URL.Path != path || r.URL.EscapedPath() != "/reverse/barbican/v1/"+kind+"/"+url.PathEscape(id) || r.URL.RawQuery != "" {
								t.Error("owned DELETE resolved another target", r.Method, r.URL)
							}
							if data, err := io.ReadAll(r.Body); err != nil || len(data) != 0 {
								t.Error("owned DELETE sent a body", string(data), err)
							}
							w.Header().Set("X-Delete-Proof", "passive")
							testcloud.JSON(w, status, "not JSON\x00{\"id\":\"https://foreign.invalid/ignored\"")
						})
						var call func(context.Context, resource.Ref, ...resource.LookupOption) error
						var client *gophercloud.ServiceClient
						if kind == "containers" {
							a := containerListFixtures()[fixtureIndex].open(t, c)
							client, call = a.RawClient(), a.Remove
							if lane == "Resources.Delete" {
								call = a.Resources.Delete
							}
						} else if kind == "orders" {
							a := orderListFixtures()[fixtureIndex].open(t, c)
							client, call = a.RawClient(), a.Remove
							if lane == "Resources.Delete" {
								call = a.Resources.Delete
							}
						} else {
							a := secretListFixtures()[fixtureIndex].open(t, c)
							client, call = a.RawClient(), a.Remove
							if lane == "Resources.Delete" {
								call = a.Resources.Delete
							}
						}
						client.MoreHeaders = map[string]string{"X-Source": "source"}
						client.Microversion = "1.0"
						c.Provider.SetToken("owned-delete-token")
						if err := call(context.Background(), resource.ID(id)); err != nil || calls.Load() != 1 || client.ProviderClient != c.Provider || client.ResourceBase != c.Server.URL+"/reverse/barbican/v1/" {
							t.Fatal("passive DELETE response caused decode, lookup or resend", err, calls.Load(), client)
						}
					})
				}
			}
		}
	}
}

func TestKeyManagerOwnedMetadataDeleteMissingAndNativeFailures(t *testing.T) {
	for _, kind := range []string{"containers", "orders", "secrets"} {
		for fixtureIndex, fixtureName := range []string{"leaf", "cached connection"} {
			for _, tc := range []struct {
				name   string
				code   int
				strict bool
			}{
				{"default missing", 404, false}, {"strict missing", 404, true}, {"forbidden", 403, false}, {"conflict", 409, false},
			} {
				t.Run(kind+"/"+fixtureName+"/"+tc.name, func(t *testing.T) {
					c := testcloud.New(t)
					path := "/reverse/barbican/v1/" + kind + "/id"
					const body = `{"message":"original delete failure"}`
					var calls atomic.Int32
					c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						th.TestMethod(t, r, http.MethodDelete)
						if r.URL.Path != path || r.URL.RawQuery != "" {
							t.Error(r.Method, r.URL)
						}
						w.Header().Set("X-Delete-Proof", tc.name)
						testcloud.JSON(w, tc.code, body)
					})
					var call func(context.Context, resource.Ref, ...resource.LookupOption) error
					if kind == "containers" {
						call = containerListFixtures()[fixtureIndex].open(t, c).Remove
					} else if kind == "orders" {
						call = orderListFixtures()[fixtureIndex].open(t, c).Remove
					} else {
						call = secretListFixtures()[fixtureIndex].open(t, c).Remove
					}
					var options []resource.LookupOption
					if tc.strict {
						options = []resource.LookupOption{resource.WithMissingError()}
					}
					err := call(context.Background(), resource.ID("id"), options...)
					if calls.Load() != 1 {
						t.Fatal("DELETE failure caused lookup/verification/resend", calls.Load(), err)
					}
					if tc.code == 404 && !tc.strict {
						if err != nil {
							t.Fatal("default missing was not ignored", err)
						}
						return
					}
					var outer, collection, inner *resource.OperationError
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &outer) || outer.Operation != "Remove" || outer.Resource != kind || !errors.As(outer.Cause, &collection) || collection.Operation != "delete" || collection.Resource != kind || !errors.As(err, &native) || native.Actual != tc.code || native.Method != http.MethodDelete || native.URL != c.Server.URL+path || string(native.Body) != body || native.ResponseHeader.Get("X-Delete-Proof") != tc.name {
						t.Fatal("owned/native DELETE operation or HTTP cause lost", err, outer, collection, native)
					}
					if tc.strict {
						var missing *resource.NotFoundError
						if !errors.Is(err, resource.ErrNotFound) || !errors.As(collection.Cause, &missing) || missing.Reference != "id" || !errors.As(missing.Cause, &inner) || inner.Operation != "Delete" || inner.Resource != kind {
							t.Fatal("strict DELETE missing/inner cause lost", err, missing, inner)
						}
					} else if errors.Is(err, resource.ErrNotFound) || !errors.As(collection.Cause, &inner) || inner.Operation != "Delete" || inner.Resource != kind {
						t.Fatal("non404 DELETE failure was hidden or reclassified", err, inner)
					}
				})
			}
		}
	}
}

func TestKeyManagerOwnedMetadataDeleteAcceptedFailuresStayTerminal(t *testing.T) {
	// Common failures are shared engine evidence; each public lane receives
	// representative failures instead of duplicating every fault per service.
	for _, tc := range []struct{ kind, mode string }{
		{"containers", "read404"}, {"orders", "close404"},
		{"containers", "read"}, {"containers", "close"},
		{"orders", "source"}, {"orders", "cancel"},
	} {
		kind, mode := tc.kind, tc.mode
		t.Run(kind+"/"+mode, func(t *testing.T) {
			c := testcloud.New(t)
			path := "/reverse/barbican/v1/" + kind + "/id"
			const body = "accepted-prefix-not-JSON"
			var calls, retries atomic.Int32
			c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodDelete)
				if r.URL.Path != path || r.URL.RawQuery != "" {
					t.Error(r.Method, r.URL)
				}
				w.Header().Set("X-Delete-Proof", mode)
				testcloud.JSON(w, 202, body)
			})
			var client *gophercloud.ServiceClient
			var call func(context.Context, resource.Ref, ...resource.LookupOption) error
			if kind == "containers" {
				a := containerListFixtures()[0].open(t, c)
				client, call = a.RawClient(), a.Resources.Delete
			} else {
				a := orderListFixtures()[1].open(t, c)
				client, call = a.RawClient(), a.Remove
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("accepted DELETE " + mode)
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{202, 204}, Method: http.MethodDelete, URL: c.Server.URL + path, Body: []byte("nested native404"), ResponseHeader: http.Header{"X-Nested-Proof": {"original"}}}
			var track *payloadContractTracking
			switch mode {
			case "read404":
				track = payloadContractTrack(c, nested, nil)
			case "close404":
				track = payloadContractTrack(c, nil, nested)
			case "read":
				track = payloadContractTrack(c, cause, nil)
			case "close":
				track = payloadContractTrack(c, nil, cause)
			default:
				base := c.Provider.HTTPClient.Transport
				if base == nil {
					base = http.DefaultTransport
				}
				c.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					response, err := base.RoundTrip(r)
					if err != nil {
						return response, err
					}
					data, readErr := io.ReadAll(response.Body)
					closeErr := response.Body.Close()
					if readErr != nil || closeErr != nil {
						return nil, errors.Join(readErr, closeErr)
					}
					response.Body = io.NopCloser(bytes.NewReader(data))
					if mode == "source" {
						client.ResourceBase = c.Server.URL + "/changed/v1/"
					} else {
						cancel(cause)
					}
					return response, nil
				})
			}
			c.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			err := call(ctx, resource.ID("id")) // default ignore-missing must not suppress nested404
			var proof *resource.ResponseError
			var operation *resource.OperationError
			if err == nil || errors.Is(err, resource.ErrNotFound) || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != body || proof.Header.Get("X-Delete-Proof") != mode || calls.Load() != 1 || retries.Load() != 0 || !errors.As(err, &operation) {
				t.Fatal("accepted DELETE proof was ignored/discarded/retried", err, proof, operation, calls.Load(), retries.Load())
			}
			if kind == "orders" {
				if operation.Operation != "Remove" {
					t.Fatal(operation)
				}
				if !errors.As(operation.Cause, &operation) {
					t.Fatal("collection delete error missing", err)
				}
			}
			if operation.Operation != "delete" || operation.Resource != kind {
				t.Fatal(operation)
			}
			var inner *resource.OperationError
			if !errors.As(operation.Cause, &inner) || inner.Operation != "Delete" || inner.Resource != kind {
				t.Fatal(inner, err)
			}
			if strings.HasSuffix(mode, "404") {
				var original gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &original) || original.Actual != 404 || string(original.Body) != "nested native404" || original.ResponseHeader.Get("X-Nested-Proof") != "original" {
					t.Fatal(original, err)
				}
			} else if mode == "source" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, cause) || mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if track != nil {
				physical := track.last(t)
				if track.calls.Load() != 1 || physical.reads.Load() == 0 || physical.closes.Load() != 1 {
					t.Fatal(track.calls.Load(), physical.reads.Load(), physical.closes.Load())
				}
			}
		})
	}
}

func TestKeyManagerOwnedMetadataDeletePreflightAndNativeRetrySources(t *testing.T) {
	for _, kind := range []string{"containers", "orders"} {
		for _, mode := range []string{"invalid ID", "full HREF ID", "nil context", "cancel", "nil API", "nil Resources", "nil provider", "wrong service", "bad endpoint", "nil option", "Order Name"} {
			if mode == "Order Name" && kind != "orders" {
				continue
			}
			t.Run(kind+"/"+mode, func(t *testing.T) {
				c := testcloud.New(t)
				var calls atomic.Int32
				c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
				client := secretFetchClient(c)
				ca, oa := containers.New(client), orders.New(client)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("delete preflight cancelled")
				ref := resource.ID("id")
				var options []resource.LookupOption
				want := resource.ErrInvalidOption
				switch mode {
				case "invalid ID":
					ref = resource.ID("bad/path")
				case "full HREF ID":
					ref = resource.ID("https://foreign.invalid/id")
				case "nil context":
					ctx = nil
				case "cancel":
					cancel(cause)
					want = context.Canceled
				case "nil API":
					ca, oa = nil, nil
				case "nil Resources":
					ca.Resources, oa.Resources = nil, nil
				case "nil provider":
					client.ProviderClient = nil
				case "wrong service":
					client.Type = "compute"
					want = resource.ErrUnsupported
				case "bad endpoint":
					client.Endpoint = "https://bad.invalid/?query=route"
				case "nil option":
					options = []resource.LookupOption{nil}
				case "Order Name":
					ref = resource.Name("meta-name")
					want = resource.ErrUnsupported
				}
				var err error
				if kind == "containers" {
					err = ca.Remove(ctx, ref, options...)
				} else {
					err = oa.Remove(ctx, ref, options...)
				}
				if !errors.Is(err, want) || calls.Load() != 0 || mode == "cancel" && !errors.Is(err, cause) {
					t.Fatal(err, want, calls.Load())
				}
			})
		}
		for _, mode := range []string{"source change", "expanded404"} {
			t.Run(kind+"/native retry "+mode, func(t *testing.T) {
				c := testcloud.New(t)
				path := "/reverse/barbican/v1/" + kind + "/id"
				var calls, retries atomic.Int32
				var client *gophercloud.ServiceClient
				var call func(context.Context, resource.Ref, ...resource.LookupOption) error
				if kind == "containers" {
					a := containerListFixtures()[0].open(t, c)
					client, call = a.RawClient(), a.Remove
				} else {
					a := orderListFixtures()[1].open(t, c)
					client, call = a.RawClient(), a.Remove
				}
				c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					th.TestMethod(t, r, http.MethodDelete)
					if r.URL.Path != path || r.URL.RawQuery != "" {
						t.Error(r.Method, r.URL)
					}
					status := 503
					if mode == "expanded404" {
						status = 404
					}
					w.Header().Set("X-Delete-Proof", "native retry")
					testcloud.JSON(w, status, `{"message":"retry original"}`)
				})
				c.Provider.RetryFunc = func(_ context.Context, method, target string, options *gophercloud.RequestOpts, original error, count uint) error {
					retries.Add(1)
					if method != http.MethodDelete || target != c.Server.URL+path || count != 1 {
						t.Error(method, target, count)
					}
					if mode == "source change" {
						client.ResourceBase = c.Server.URL + "/changed/v1/"
					} else {
						options.OkCodes = append(options.OkCodes, 404)
					}
					return nil
				}
				err := call(context.Background(), resource.ID("id"))
				var native gophercloud.ErrUnexpectedResponseCode
				if err == nil || errors.Is(err, resource.ErrNotFound) || !errors.As(err, &native) || string(native.Body) != `{"message":"retry original"}` || native.ResponseHeader.Get("X-Delete-Proof") != "native retry" || retries.Load() != 1 {
					t.Fatal(err, native, calls.Load(), retries.Load())
				}
				if mode == "source change" {
					if !errors.Is(err, resource.ErrInvalidOption) || native.Actual != 503 || calls.Load() != 1 {
						t.Fatal(err, native, calls.Load())
					}
				} else {
					var terminal interface{ TerminalSDKFailure() bool }
					if native.Actual != 404 || calls.Load() != 2 || !errors.As(err, &terminal) || !terminal.TerminalSDKFailure() {
						t.Fatal(err, native, calls.Load())
					}
				}
			})
		}
	}
}

func TestKeyManagerMetadataNativeDeleteCompatibility(t *testing.T) {
	// These assignments fail to compile if generated native aliases change.
	var _ containers.DeleteResult = upcontainers.DeleteResult{}
	var _ orders.DeleteResult = uporders.DeleteResult{}
	var _ *containers.Container = (*upcontainers.Container)(nil)
	var _ *orders.Order = (*uporders.Order)(nil)
	var _ secrets.DeleteResult = upsecrets.DeleteResult{}
	var _ *secrets.Secret = (*upsecrets.Secret)(nil)
	for _, kind := range []string{"containers", "orders", "secrets"} {
		for _, status := range []int{200, 202, 204, 399, 404} {
			t.Run(fmt.Sprintf("%s/%d", kind, status), func(t *testing.T) {
				c := testcloud.New(t)
				path := "/reverse/barbican/v1/" + kind + "/id"
				const body = "opaque-native-delete"
				var calls atomic.Int32
				c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					th.TestMethod(t, r, http.MethodDelete)
					if r.URL.Path != path || r.URL.RawQuery != "" {
						t.Error(r.Method, r.URL)
					}
					w.Header().Set("X-Native-Proof", "unchanged")
					testcloud.JSON(w, status, body)
				})
				var err error
				if kind == "containers" {
					err = containerListFixtures()[0].open(t, c).Delete(context.Background(), "id")
				} else if kind == "orders" {
					err = orderListFixtures()[0].open(t, c).Delete(context.Background(), "id")
				} else {
					err = secretListFixtures()[0].open(t, c).Delete(context.Background(), "id")
				}
				if calls.Load() != 1 {
					t.Fatal(err, calls.Load())
				}
				if status == 202 || status == 204 {
					if err != nil {
						t.Fatal("native accepted status changed", err)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					var op *resource.OperationError
					if !errors.As(err, &native) || native.Actual != status || !reflect.DeepEqual(native.Expected, []int{202, 204}) || string(native.Body) != body || native.ResponseHeader.Get("X-Native-Proof") != "unchanged" || errors.Is(err, resource.ErrNotFound) || !errors.As(err, &op) || op.Operation != "Delete" || op.Resource != kind {
						t.Fatal("native compatibility status/cause changed", err, native, op)
					}
				}
			})
		}
	}
}

func TestKeyManagerMetadataRemoveKeepsExistingExactNameLookup(t *testing.T) {
	for _, kind := range []string{"containers", "secrets"} {
		for fixtureIndex, fixtureName := range []string{"leaf", "cached connection"} {
			t.Run(kind+"/"+fixtureName, func(t *testing.T) {
				c := testcloud.New(t)
				path := containerListPath
				if kind == "secrets" {
					path = secretFindCollectionPath
				}
				var events []string
				c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					events = append(events, r.Method+" "+r.URL.Path)
					if r.Method == http.MethodGet && r.URL.Path == path {
						if r.URL.RawQuery != "name=chosen" {
							t.Error(r.URL)
						}
						var page string
						if kind == "containers" {
							rows := containerListRow(t, "other", map[string]any{"container_ref": "https://foreign.invalid/containers/decoy"}) + "," + containerListRow(t, "chosen", map[string]any{"container_ref": "https://foreign.invalid/containers/actual"})
							page = containerListPage(rows, "")
						} else {
							rows := secretListRow(t, "other", map[string]any{"secret_ref": "https://foreign.invalid/secrets/decoy"}) + "," + secretListRow(t, "chosen", map[string]any{"secret_ref": "https://foreign.invalid/secrets/actual", "content_types": map[string]any{"default": "text/plain"}})
							page = secretFindPage(rows, "")
						}
						testcloud.JSON(w, 200, page)
						return
					}
					if r.Method != http.MethodDelete || r.URL.Path != path+"/actual" || r.URL.RawQuery != "" {
						t.Error("Name lookup changed or introduced payload/metadata GET", r.Method, r.URL)
					}
					testcloud.JSON(w, 200, "opaque-success")
				})
				var err error
				if kind == "containers" {
					err = containerListFixtures()[fixtureIndex].open(t, c).Remove(context.Background(), resource.Name("chosen"))
				} else {
					err = secretListFixtures()[fixtureIndex].open(t, c).Remove(context.Background(), resource.Name("chosen"))
				}
				if err != nil || !reflect.DeepEqual(events, []string{http.MethodGet + " " + path, http.MethodDelete + " " + path + "/actual"}) {
					t.Fatal(err, events)
				}
			})
		}
	}
}
