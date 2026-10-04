package blockstorage_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

const vroType = "literal / % ?#"
const vroDefaultRetype = `{"os-retype":{"migration_policy":"never","new_type":"literal / % ?#"}}`
const vroEmptyRetype = `{"os-retype":{"new_type":"literal / % ?#"}}`
const vroFalseCompletion = `{"os-extend_volume_completion":{"error":false}}`
const vroTrueCompletion = `{"os-extend_volume_completion":{"error":true}}`

type vroAction func(context.Context, *gophercloud.ServiceClient) (*blockstorage.VolumeActionResult, error)

func vroWire(t *testing.T, req *http.Request, want, token string) {
	t.Helper()
	var body []byte
	var err error
	if req.Body != nil {
		body, err = io.ReadAll(req.Body)
	}
	if err != nil || string(body) != want || req.Method != http.MethodPost || req.URL.EscapedPath() != vsaBase+"volumes/literal/action" || req.URL.RawQuery != "" || req.Header.Get("X-Source") != "original" || req.Header.Get("X-Auth-Token") != token || req.Header.Get("OpenStack-API-Version") != "volume 3.80" || req.Header.Get("X-OpenStack-Volume-API-Version") != "3.80" || req.ContentLength != int64(len(want)) || len(req.TransferEncoding) != 0 {
		t.Error("resize option changed literal fields, scope, framing, captured header or version", req.Method, req.URL, string(body), req.Header, req.ContentLength, req.TransferEncoding, err)
	}
}
func vroHTTP(t *testing.T, body, token string, client *gophercloud.ServiceClient, cloud *testcloud.Cloud, call vroAction) {
	t.Helper()
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		vroWire(t, req, body, token)
		w.Header().Set("X-Proof", "resize option")
		w.WriteHeader(203)
		_, _ = w.Write([]byte("opaque resize acknowledgement"))
	})
	result, err := call(vsaContext(t), client)
	if err != nil || result == nil || !result.Completed || result.VolumeID != "literal" || result.Microversion != "3.80" || len(result.Discovery) != 0 || result.Applied == nil || result.Applied.StatusCode != 203 || result.Applied.Header.Get("X-Proof") != "resize option" || string(result.Applied.Body) != "opaque resize acknowledgement" || calls.Load() != 1 {
		t.Fatal(result, err, calls.Load())
	}
}

func TestVolumeResizeOptionsDefaultsEmptyReplacementAndFactoriesOwnValues(t *testing.T) {
	retypes := []struct {
		name, body string
		options    []blockstorage.VolumeRetypeOption
	}{
		{"default never", vroDefaultRetype, nil},
		{"explicit empty omitted", vroEmptyRetype, []blockstorage.VolumeRetypeOption{blockstorage.WithVolumeRetypeMigrationPolicy("")}},
		{"arbitrary policy", `{"os-retype":{"migration_policy":"not an enum / %?#","new_type":"literal / % ?#"}}`, []blockstorage.VolumeRetypeOption{blockstorage.WithVolumeRetypeMigrationPolicy("not an enum / %?#")}},
		{"empty then complete replacement", vroDefaultRetype, []blockstorage.VolumeRetypeOption{blockstorage.WithVolumeRetypeMigrationPolicy(""), blockstorage.WithVolumeRetypeOptions(blockstorage.VolumeRetypeOpts{})}},
		{"later empty wins", vroEmptyRetype, []blockstorage.VolumeRetypeOption{blockstorage.WithVolumeRetypeOptions(blockstorage.VolumeRetypeOpts{}), blockstorage.WithVolumeRetypeMigrationPolicy("")}},
		{"invalid earlier policy replaced", vroDefaultRetype, []blockstorage.VolumeRetypeOption{blockstorage.WithVolumeRetypeMigrationPolicy(string([]byte{0xff})), blockstorage.WithVolumeRetypeOptions(blockstorage.VolumeRetypeOpts{})}},
	}
	for _, tc := range retypes {
		t.Run("retype/"+tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			vroHTTP(t, tc.body, "test-token", vsaClient(cloud, "3.80"), cloud, func(ctx context.Context, client *gophercloud.ServiceClient) (*blockstorage.VolumeActionResult, error) {
				return blockstorage.RetypeVolume(ctx, client, blockstorage.VolumeActionRequest{VolumeID: "literal"}, vroType, tc.options...)
			})
		})
	}
	completions := []struct {
		name, body string
		options    []blockstorage.VolumeExtendCompletionOption
	}{
		{"default false", vroFalseCompletion, nil},
		{"explicit true", vroTrueCompletion, []blockstorage.VolumeExtendCompletionOption{blockstorage.WithVolumeExtendCompletionError(true)}},
		{"explicit false", vroFalseCompletion, []blockstorage.VolumeExtendCompletionOption{blockstorage.WithVolumeExtendCompletionError(false)}},
		{"true then replacement default", vroFalseCompletion, []blockstorage.VolumeExtendCompletionOption{blockstorage.WithVolumeExtendCompletionError(true), blockstorage.WithVolumeExtendCompletionOptions(blockstorage.VolumeExtendCompletionOpts{})}},
		{"later true wins", vroTrueCompletion, []blockstorage.VolumeExtendCompletionOption{blockstorage.WithVolumeExtendCompletionOptions(blockstorage.VolumeExtendCompletionOpts{}), blockstorage.WithVolumeExtendCompletionError(true)}},
	}
	for _, tc := range completions {
		t.Run("completion/"+tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			vroHTTP(t, tc.body, "test-token", vsaClient(cloud, "3.80"), cloud, func(ctx context.Context, client *gophercloud.ServiceClient) (*blockstorage.VolumeActionResult, error) {
				return blockstorage.CompleteVolumeExtend(ctx, client, blockstorage.VolumeActionRequest{VolumeID: "literal"}, tc.options...)
			})
		})
	}
	t.Run("factories own caller and every prepared result", func(t *testing.T) {
		policy, flag := "owned", true
		rvalue := blockstorage.VolumeRetypeOpts{MigrationPolicy: &policy}
		cvalue := blockstorage.VolumeExtendCompletionOpts{Error: &flag}
		rfactory := blockstorage.WithVolumeRetypeOptions(rvalue)
		cfactory := blockstorage.WithVolumeExtendCompletionOptions(cvalue)
		policy, flag = "caller changed", false
		rvalue.MigrationPolicy, cvalue.Error = nil, nil
		for i := 0; i < 2; i++ {
			r, err := blockstorage.PrepareVolumeRetypeOptions(vsaContext(t), rfactory)
			if err != nil || r.MigrationPolicy == nil || *r.MigrationPolicy != "owned" {
				t.Fatal(r, err)
			}
			c, err := blockstorage.PrepareVolumeExtendCompletionOptions(vsaContext(t), cfactory)
			if err != nil || c.Error == nil || !*c.Error {
				t.Fatal(c, err)
			}
			*r.MigrationPolicy, *c.Error = "returned changed", false
		}
		cloud := testcloud.New(t)
		vroHTTP(t, `{"os-retype":{"migration_policy":"owned","new_type":"literal / % ?#"}}`, "test-token", vsaClient(cloud, "3.80"), cloud, func(ctx context.Context, client *gophercloud.ServiceClient) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.RetypeVolume(ctx, client, blockstorage.VolumeActionRequest{VolumeID: "literal"}, vroType, rfactory)
		})
		cloud = testcloud.New(t)
		vroHTTP(t, vroTrueCompletion, "test-token", vsaClient(cloud, "3.80"), cloud, func(ctx context.Context, client *gophercloud.ServiceClient) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.CompleteVolumeExtend(ctx, client, blockstorage.VolumeActionRequest{VolumeID: "literal"}, cfactory)
		})
	})
}

func TestVolumeResizeOriginalsOwnOrderCallbackSliceAndRetainedPointersThroughHTTP(t *testing.T) {
	for _, operation := range []string{"RetypeVolume", "CompleteVolumeExtend"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.80")
			var order []int
			var replaced atomic.Int32
			var firstRetained, secondRetained func()
			var call vroAction
			body := vroTrueCompletion
			if operation == "RetypeVolume" {
				body = `{"os-retype":{"migration_policy":"owned","new_type":"literal / % ?#"}}`
				policy := "owned"
				var options []blockstorage.VolumeRetypeOption
				options = []blockstorage.VolumeRetypeOption{
					func(next *blockstorage.VolumeRetypeOpts) error {
						order = append(order, 1)
						next.MigrationPolicy = &policy
						firstRetained = func() { *next.MigrationPolicy = "retained changed" }
						options[1] = func(*blockstorage.VolumeRetypeOpts) error {
							replaced.Add(1)
							return errors.New("replaced retype callback")
						}
						client.MoreHeaders["x-source"] = "later ordinary header"
						cloud.Provider.SetToken("option-live")
						return nil
					},
					func(next *blockstorage.VolumeRetypeOpts) error {
						order = append(order, 2)
						policy = "outside changed"
						firstRetained()
						if next.MigrationPolicy == nil || *next.MigrationPolicy != "owned" {
							t.Error("retype callback borrowed retained input", next)
						}
						secondRetained = func() { *next.MigrationPolicy = "second retained changed" }
						return nil
					},
					func(next *blockstorage.VolumeRetypeOpts) error {
						order = append(order, 3)
						secondRetained()
						if next.MigrationPolicy == nil || *next.MigrationPolicy != "owned" {
							t.Error("retype callback borrowed previous policy", next)
						}
						return nil
					},
				}
				call = func(ctx context.Context, client *gophercloud.ServiceClient) (*blockstorage.VolumeActionResult, error) {
					return blockstorage.RetypeVolume(ctx, client, blockstorage.VolumeActionRequest{VolumeID: "literal"}, vroType, options...)
				}
			} else {
				flag := true
				var options []blockstorage.VolumeExtendCompletionOption
				options = []blockstorage.VolumeExtendCompletionOption{
					func(next *blockstorage.VolumeExtendCompletionOpts) error {
						order = append(order, 1)
						next.Error = &flag
						firstRetained = func() { *next.Error = false }
						options[1] = func(*blockstorage.VolumeExtendCompletionOpts) error {
							replaced.Add(1)
							return errors.New("replaced completion callback")
						}
						client.MoreHeaders["x-source"] = "later ordinary header"
						cloud.Provider.SetToken("option-live")
						return nil
					},
					func(next *blockstorage.VolumeExtendCompletionOpts) error {
						order = append(order, 2)
						flag = false
						firstRetained()
						if next.Error == nil || !*next.Error {
							t.Error("completion callback borrowed retained input", next)
						}
						secondRetained = func() { *next.Error = false }
						return nil
					},
					func(next *blockstorage.VolumeExtendCompletionOpts) error {
						order = append(order, 3)
						secondRetained()
						if next.Error == nil || !*next.Error {
							t.Error("completion callback borrowed previous policy", next)
						}
						return nil
					},
				}
				call = func(ctx context.Context, client *gophercloud.ServiceClient) (*blockstorage.VolumeActionResult, error) {
					return blockstorage.CompleteVolumeExtend(ctx, client, blockstorage.VolumeActionRequest{VolumeID: "literal"}, options...)
				}
			}
			vroHTTP(t, body, "option-live", client, cloud, call)
			if !reflect.DeepEqual(order, []int{1, 2, 3}) || replaced.Load() != 0 || client.MoreHeaders["x-source"] != "later ordinary header" {
				t.Fatal(order, replaced.Load(), client)
			}
		})
	}
}

func TestVolumeResizePreparePreservesCausesAndValidatesOnlyFinalRetypePolicy(t *testing.T) {
	for _, family := range []string{"retype", "completion"} {
		kinds := []string{"nil context", "already canceled", "nil callback", "callback error", "callback cancel", "callback error and cancel"}
		if family == "retype" {
			kinds = append(kinds, "invalid final policy", "invalid then repaired policy")
		}
		for _, kind := range kinds {
			t.Run(family+"/"+kind, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(vsaContext(t))
				defer cancel(nil)
				var selected context.Context = ctx
				callbackCause, cancelCause := errors.New("resize prepare callback"), errors.New("resize prepare custom cancellation")
				first, later := 0, 0
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
				wantFirst, wantLater := 1, 0
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
				if kind == "invalid final policy" || kind == "invalid then repaired policy" {
					wantLater = 1
				}
				var err error
				if family == "retype" {
					options := []blockstorage.VolumeRetypeOption{
						func(next *blockstorage.VolumeRetypeOpts) error {
							policy := "owned"
							if kind == "invalid final policy" || kind == "invalid then repaired policy" {
								policy = string([]byte{0xff})
							}
							next.MigrationPolicy = &policy
							return callback()
						},
						func(next *blockstorage.VolumeRetypeOpts) error {
							later++
							if kind == "invalid then repaired policy" {
								policy := "repaired arbitrary policy"
								next.MigrationPolicy = &policy
							}
							return nil
						},
					}
					if kind == "nil callback" {
						options[0] = nil
					}
					var prepared blockstorage.VolumeRetypeOpts
					prepared, err = blockstorage.PrepareVolumeRetypeOptions(selected, options...)
					if kind == "invalid then repaired policy" {
						if err != nil || prepared.MigrationPolicy == nil || *prepared.MigrationPolicy != "repaired arbitrary policy" || first != 1 || later != 1 {
							t.Fatal(prepared, err, first, later)
						}
						return
					}
					if prepared.MigrationPolicy != nil {
						t.Fatal("failed retype Prepare published policy", prepared, err)
					}
				} else {
					options := []blockstorage.VolumeExtendCompletionOption{
						func(next *blockstorage.VolumeExtendCompletionOpts) error {
							yes := true
							next.Error = &yes
							return callback()
						},
						func(*blockstorage.VolumeExtendCompletionOpts) error { later++; return nil },
					}
					if kind == "nil callback" {
						options[0] = nil
					}
					var prepared blockstorage.VolumeExtendCompletionOpts
					prepared, err = blockstorage.PrepareVolumeExtendCompletionOptions(selected, options...)
					if prepared.Error != nil {
						t.Fatal("failed completion Prepare published policy", prepared, err)
					}
				}
				if err == nil || first != wantFirst || later != wantLater {
					t.Fatal(err, first, later, wantFirst, wantLater)
				}
				if (kind == "nil context" || kind == "nil callback" || kind == "invalid final policy") && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				if (kind == "already canceled" || kind == "callback cancel" || kind == "callback error and cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal("prepare cancellation identity lost", err)
				}
				if (kind == "callback error" || kind == "callback error and cancel") && !errors.Is(err, callbackCause) {
					t.Fatal("prepare callback identity lost", err)
				}
			})
		}
	}
}

func TestVolumeResizeDirectPreflightAndStickySourcesStopOriginalsAndHTTP(t *testing.T) {
	for _, operation := range []string{"RetypeVolume", "CompleteVolumeExtend"} {
		kinds := []string{"nil client", "wrong role and unsafe ID", "unsafe ID", "nil context", "already canceled", "nil callback", "source change", "joined source callback and cancel"}
		if operation == "RetypeVolume" {
			kinds = append(kinds, "invalid new type", "invalid final migration policy")
		}
		for _, kind := range kinds {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.80")
				original := *client
				ctx, cancel := context.WithCancelCause(vsaContext(t))
				defer cancel(nil)
				var selected context.Context = ctx
				id, newType := "literal", vroType
				callbackCause, cancelCause := errors.New("resize original callback"), errors.New("resize original custom cancellation")
				var calls, first, later atomic.Int32
				want, wantFirst, wantLater := resource.ErrInvalidOption, int32(0), int32(0)
				switch kind {
				case "nil client":
					client = nil
				case "wrong role and unsafe ID":
					client.Type = "compute"
					id = "a/b"
					want = resource.ErrUnsupported
				case "unsafe ID":
					id = "a/b"
				case "nil context":
					selected = nil
				case "already canceled":
					cancel(cancelCause)
					want = context.Canceled
				case "source change", "joined source callback and cancel":
					wantFirst = 1
				case "invalid new type":
					newType = string([]byte{0xff})
				case "invalid final migration policy":
					wantFirst, wantLater = 1, 1
				}
				callback := func() error {
					first.Add(1)
					if kind == "source change" || kind == "joined source callback and cancel" {
						client.ResourceBase += "changed/"
					}
					if kind == "joined source callback and cancel" {
						cancel(cancelCause)
						return callbackCause
					}
					return nil
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
					calls.Add(1)
					t.Error("local resize option failure reached HTTP", req.URL)
					w.WriteHeader(500)
				})
				var result *blockstorage.VolumeActionResult
				var err error
				if operation == "RetypeVolume" {
					options := []blockstorage.VolumeRetypeOption{
						func(next *blockstorage.VolumeRetypeOpts) error {
							policy := "never"
							if kind == "invalid final migration policy" {
								policy = string([]byte{0xff})
							}
							next.MigrationPolicy = &policy
							return callback()
						},
						func(*blockstorage.VolumeRetypeOpts) error { later.Add(1); *client = original; return nil },
					}
					if kind == "nil callback" {
						options[0] = nil
					}
					result, err = blockstorage.RetypeVolume(selected, client, blockstorage.VolumeActionRequest{VolumeID: id}, newType, options...)
				} else {
					options := []blockstorage.VolumeExtendCompletionOption{
						func(next *blockstorage.VolumeExtendCompletionOpts) error {
							yes := true
							next.Error = &yes
							return callback()
						},
						func(*blockstorage.VolumeExtendCompletionOpts) error { later.Add(1); *client = original; return nil },
					}
					if kind == "nil callback" {
						options[0] = nil
					}
					result, err = blockstorage.CompleteVolumeExtend(selected, client, blockstorage.VolumeActionRequest{VolumeID: id}, options...)
				}
				var proof *resource.ResponseError
				vsaOperation(t, err, operation)
				if result != nil || !errors.Is(err, want) || errors.As(err, &proof) || calls.Load() != 0 || first.Load() != wantFirst || later.Load() != wantLater {
					t.Fatal(result, err, proof, calls.Load(), first.Load(), later.Load(), wantFirst, wantLater)
				}
				if (kind == "already canceled" || kind == "joined source callback and cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal("direct cancellation identity lost", err)
				}
				if kind == "joined source callback and cancel" && !errors.Is(err, callbackCause) {
					t.Fatal("direct callback identity lost", err)
				}
			})
		}
	}
}
