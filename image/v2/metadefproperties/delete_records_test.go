package metadefproperties

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

var propertyRecordDeletions = []struct {
	name string
	all  bool
}{{"member", false}, {"all", true}}

type propertyRecordDeleteOptions struct {
	member []DeleteOption
	all    []DeleteAllOption
}

func propertyRecordDeleteCall(scope *NamespaceScope, ctx context.Context, all bool, input RecordRequest, options propertyRecordDeleteOptions) (*Acknowledgement, error) {
	if all {
		return scope.DeleteAllRecords(ctx, options.all...)
	}
	return scope.DeleteRecord(ctx, input, options.member...)
}

func TestMetadefPropertyDeleteRecordsFixedRoutesAndOpaqueStatuses(t *testing.T) {
	for _, operation := range propertyRecordDeletions {
		for _, code := range []int{200, 201, 202, 203, 204, 299, 300, 304, 399} {
			t.Run(fmt.Sprintf("%s/%d", operation.name, code), func(t *testing.T) {
				calls := 0
				body := "opaque acknowledgement\xff\x00"
				client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					want := "/reverse/glance/v2/metadefs/namespaces/" + url.PathEscape(propertyRecordParent) + "/properties"
					if !operation.all {
						want += "/" + url.PathEscape(propertyRecordChild)
					}
					if req.Method != http.MethodDelete || req.URL.EscapedPath() != want || req.URL.RawQuery != "" || req.Body != nil || req.Header.Get("X-Auth-Token") != "before" || req.Header.Get("X-Option") != "owned" {
						t.Fatal("fixed single bodyless deletion", req.Method, req.URL, req.Header)
					}
					response := propertyCoreJSON(req, code, body)
					response.Header.Set("Link", "broken and passive")
					return response, nil
				})
				scope := propertyRecordScope(t, client, Dependencies{})
				options := propertyRecordDeleteOptions{member: []DeleteOption{WithDeleteHeader("X-Option", "owned")}, all: []DeleteAllOption{WithDeleteAllHeader("X-Option", "owned")}}
				ack, err := propertyRecordDeleteCall(scope, context.Background(), operation.all, RecordRequest{ID: propertyRecordChild}, options)
				if err != nil || ack == nil || ack.Namespace != propertyRecordParent || ack.StatusCode != code || string(ack.Body) != body || ack.Header.Get("X-Proof") != "original" || calls != 1 {
					t.Fatal("opaque actual receipt lost", ack, err, calls)
				}
				if operation.all {
					if ack.Name != nil {
						t.Fatal("bulk receipt invented a child")
					}
				} else if ack.Name == nil || *ack.Name != propertyRecordChild {
					t.Fatal("member response changed selected child", ack.Name)
				}
			})
		}
	}
}

func TestMetadefPropertyDeleteRecordResourceIdentityIsSnapshotAndOnlySelection(t *testing.T) {
	for _, check := range []struct{ name, body, identity string }{
		{"explicit id", `{"id":"selected-id","name":"alias"}`, "selected-id"},
		{"alternate name", `{"name":"selected-name"}`, "selected-name"},
		{"opaque invalid definitions ignored", `{"id":"selected-id","minimum":"²","title":{"opaque":true},"namespace_name":"foreign","location":[1]}`, "selected-id"},
	} {
		t.Run(check.name, func(t *testing.T) {
			calls := 0
			seed := propertyRecordSeed(t, check.body)
			seed.Body["unused"] = json.RawMessage(`not valid JSON`)
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.EscapedPath() != "/reverse/glance/v2/metadefs/namespaces/"+url.PathEscape(propertyRecordParent)+"/properties/"+url.PathEscape(check.identity) || req.URL.RawQuery != "" || req.Body != nil {
					t.Fatal("callback or definition retargeted deletion", req.URL)
				}
				return propertyCoreJSON(req, 202, `{"name":"response name","namespace_name":"response parent","deleted":900719925474099312345}`), nil
			})
			ack, err := propertyRecordScope(t, client, Dependencies{}).DeleteRecord(context.Background(), RecordRequest{Resource: seed}, func(*DeleteOpts) error {
				seed.Body["id"] = json.RawMessage(`"caller changed"`)
				seed.Body["name"] = json.RawMessage(`"caller changed"`)
				return nil
			})
			if err != nil || ack == nil || ack.Name == nil || *ack.Name != check.identity || ack.Namespace != propertyRecordParent || calls != 1 || ack.StatusCode != 202 {
				t.Fatal(ack, err, calls)
			}
		})
	}
}

func TestMetadefPropertyDeleteRecordsMissingReceiptsAndNativeErrors(t *testing.T) {
	for _, mode := range []string{"default missing", "explicit handled missing", "handled missing close error", "strict member missing", "bulk missing", "transport named404"} {
		t.Run(mode, func(t *testing.T) {
			calls, retries := 0, 0
			closeErr := errors.New("missing response close")
			body := &propertyCoreBody{reader: strings.NewReader("missing opaque\xff")}
			if mode == "handled missing close error" {
				body.closeErr = closeErr
			}
			transport404 := &gophercloud.ErrUnexpectedResponseCode{Actual: 404, Body: []byte("transport claim")}
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if mode == "transport named404" {
					return nil, transport404
				}
				return propertyCoreHTTP(req, 404, body), nil
			})
			stop := errors.New("native retry stop")
			client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return stop
			}
			scope := propertyRecordScope(t, client, Dependencies{})
			options := propertyRecordDeleteOptions{}
			if mode == "strict member missing" {
				options.member = []DeleteOption{WithDeleteIgnoreMissing(false)}
			}
			if mode == "explicit handled missing" {
				options.member = []DeleteOption{WithDeleteIgnoreMissing(true)}
			}
			ack, err := propertyRecordDeleteCall(scope, context.Background(), mode == "bulk missing", RecordRequest{ID: propertyRecordChild}, options)
			if calls != 1 {
				t.Fatal(calls)
			}
			switch mode {
			case "default missing", "explicit handled missing", "handled missing close error":
				if ack == nil || ack.StatusCode != 404 || ack.Name == nil || *ack.Name != propertyRecordChild || ack.Namespace != propertyRecordParent || string(ack.Body) != "missing opaque\xff" || body.closes != 1 || retries != 0 {
					t.Fatal("handled physical missing discarded or resent actual receipt", ack, err, body.closes, retries)
				}
				if mode == "handled missing close error" {
					if !errors.Is(err, closeErr) {
						t.Fatal(err)
					}
					propertyCoreProof(t, err, 404, "missing opaque\xff")
				} else if err != nil {
					t.Fatal(err)
				}
			case "transport named404":
				if ack != nil || !errors.Is(err, transport404) || !errors.Is(err, stop) || retries != 1 || body.closes != 0 {
					t.Fatal("transport status claim became handled physical absence", ack, err, retries, body.closes)
				}
			default:
				var native gophercloud.ErrUnexpectedResponseCode
				if ack != nil || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "missing opaque\xff" || native.ResponseHeader.Get("X-Proof") != "original" || !errors.Is(err, stop) || body.closes != 1 || retries != 1 {
					t.Fatal("native missing policy lost", ack, err, native, retries, body.closes)
				}
			}
		})
	}
}

func TestMetadefPropertyDeleteRecordCompleteInputPreflight(t *testing.T) {
	for _, check := range []struct {
		name  string
		input RecordRequest
	}{
		{"missing", RecordRequest{}},
		{"both forms", RecordRequest{ID: "selected", Resource: &resource.RawResource{}}},
		{"missing resource identity", RecordRequest{Resource: &resource.RawResource{}}},
		{"explicit null id", RecordRequest{Resource: propertyRecordSeed(t, `{"id":null,"name":"alias"}`)}},
		{"explicit empty id", RecordRequest{Resource: propertyRecordSeed(t, `{"id":"","name":"alias"}`)}},
		{"explicit nonstring id", RecordRequest{Resource: propertyRecordSeed(t, `{"id":false,"name":"alias"}`)}},
		{"unsafe id", RecordRequest{ID: "bad/name"}},
		{"control id", RecordRequest{ID: "bad\nname"}},
		{"overlong id", RecordRequest{ID: strings.Repeat("界", 81)}},
		{"invalid UTF8 raw id", RecordRequest{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": {'"', 0xff, '"'}, "name": json.RawMessage(`"alias"`)}}}}},
	} {
		t.Run(check.name, func(t *testing.T) {
			calls, callbacks := 0, 0
			client := propertyCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			ack, err := propertyRecordScope(t, client, Dependencies{}).DeleteRecord(context.Background(), check.input, func(*DeleteOpts) error { callbacks++; return nil })
			if ack != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 {
				t.Fatal("invalid selection invoked callback or HTTP", ack, err, calls, callbacks)
			}
		})
	}
}

func TestMetadefPropertyDeleteRecordsSourceFirstAndPerCallbackGuard(t *testing.T) {
	for _, operation := range propertyRecordDeletions {
		for _, mode := range []string{"nil context", "canceled context", "source lifetime", "outer preflight", "nil option", "callback error", "source callback cannot be restored", "namespace callback cannot be restored"} {
			t.Run(operation.name+"/"+mode, func(t *testing.T) {
				calls, first, second := 0, 0, 0
				cause := errors.New("guard cause")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				var use context.Context = ctx
				client := propertyCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
				scope := propertyRecordScope(t, client, Dependencies{})
				switch mode {
				case "nil context":
					use = nil
				case "canceled context":
					cancel(cause)
				case "source lifetime":
					client.Endpoint = "https://changed.test/"
				case "outer preflight":
					use = rest.WithOperationGuard(use, func(context.Context) error { return cause })
				}
				mutate := func() error {
					first++
					if mode == "callback error" {
						return cause
					}
					if mode == "source callback cannot be restored" {
						client.Endpoint = "https://changed.test/"
					}
					if mode == "namespace callback cannot be restored" {
						scope.namespace = "changed parent"
					}
					return nil
				}
				restore := func() error {
					second++
					client.Endpoint = "https://example.test/catalog/"
					scope.namespace = propertyRecordParent
					return nil
				}
				options := propertyRecordDeleteOptions{member: []DeleteOption{func(*DeleteOpts) error { return mutate() }, func(*DeleteOpts) error { return restore() }}, all: []DeleteAllOption{func(*DeleteAllOpts) error { return mutate() }, func(*DeleteAllOpts) error { return restore() }}}
				if mode == "nil option" {
					options = propertyRecordDeleteOptions{member: []DeleteOption{nil}, all: []DeleteAllOption{nil}}
				}
				ack, err := propertyRecordDeleteCall(scope, use, operation.all, RecordRequest{ID: propertyRecordChild}, options)
				if ack != nil || err == nil || calls != 0 || second != 0 {
					t.Fatal("source/option failure was restored or reached HTTP", ack, err, calls, first, second)
				}
				wantFirst := 0
				if strings.HasPrefix(mode, "callback") || strings.Contains(mode, "callback cannot") {
					wantFirst = 1
				}
				if first != wantFirst {
					t.Fatal("source preflight did not precede callbacks", first)
				}
				if mode == "canceled context" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) || mode == "outer preflight" && !errors.Is(err, cause) || mode == "callback error" && !errors.Is(err, cause) {
					t.Fatal("preflight cause lost", err)
				}
			})
		}
	}
}

func TestMetadefPropertyDeleteRecordOptionsOwnPointersHeadersAndIdentity(t *testing.T) {
	calls := 0
	seed := propertyRecordSeed(t, `{"id":"snapshot-id","name":"alias"}`)
	client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.EscapedPath() != "/reverse/glance/v2/metadefs/namespaces/"+url.PathEscape(propertyRecordParent)+"/properties/snapshot-id" || req.URL.RawQuery != "" || req.Body != nil || req.Header.Get("X-Source") != "before" || req.Header.Get("X-Full") != "owned" || req.Header.Get("X-Bulk") != "owned" || req.Header.Get("X-Replaced") != "" || req.Header.Get("X-Final") != "last" {
			t.Fatal("member option snapshot lost", req.URL, req.Header)
		}
		return propertyCoreJSON(req, 404, "missing"), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "before"}
	scope := propertyRecordScope(t, client, Dependencies{})
	ignore := false
	headers := map[string]string{"X-Full": "owned"}
	bulk := map[string]string{"X-Bulk": "owned"}
	full := WithDeleteOpts(DeleteOpts{Headers: headers, IgnoreMissing: &ignore})
	bulkOption := WithDeleteHeaders(bulk)
	var retained *DeleteOpts
	options := []DeleteOption{WithDeleteHeader("X-Replaced", "old"), full, bulkOption,
		func(config *DeleteOpts) error {
			retained = config
			seed.Body["id"] = json.RawMessage(`"caller changed"`)
			client.MoreHeaders["X-Source"] = "caller changed"
			return nil
		},
		func(*DeleteOpts) error {
			retained.Headers["X-Full"] = "retained changed"
			*retained.IgnoreMissing = true
			return nil
		},
		WithDeleteHeader("x-final", "last"),
	}
	ignore = true
	headers["X-Full"] = "caller changed"
	bulk["X-Bulk"] = "caller changed"
	ack, err := scope.DeleteRecord(context.Background(), RecordRequest{Resource: seed}, options...)
	if ack != nil || !gophercloud.ResponseCodeIs(err, 404) || calls != 1 {
		t.Fatal("retained/factory pointer changed strict missing policy", ack, err, calls)
	}
}

func TestMetadefPropertyDeleteRecordsAcceptedFaultsKeepAckAndFrozenTargets(t *testing.T) {
	for _, operation := range propertyRecordDeletions {
		for _, mode := range []string{"read close cancel", "namespace close mutation", "observed source restored"} {
			t.Run(operation.name+"/"+mode, func(t *testing.T) {
				calls, retries := 0, 0
				readErr, closeErr, cause := errors.New("read"), errors.New("close"), errors.New("cancel")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				bodyText := "opaque receipt\xff"
				var client *gophercloud.ServiceClient
				var scope *NamespaceScope
				var body *propertyCoreBody
				client = propertyCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					body = &propertyCoreBody{reader: strings.NewReader(bodyText)}
					if mode == "read close cancel" {
						body.reader = &propertyCoreReader{data: bodyText, err: readErr, after: func() { cancel(cause) }}
						body.closeErr = closeErr
					}
					if mode == "namespace close mutation" {
						return propertyCoreHTTP(req, 203, &propertyRecordCloseBody{propertyCoreBody: body, after: func() { scope.namespace = "changed parent" }}), nil
					}
					if mode == "observed source restored" {
						body.reader = &propertyCoreReader{data: bodyText, err: io.EOF, after: func() {
							client.Microversion = "2.3"
							if observed := rest.CheckOperationGuard(req.Context()); !errors.Is(observed, resource.ErrInvalidOption) {
								t.Error("source change not observed", observed)
							}
						}}
						return propertyCoreHTTP(req, 203, &propertyRecordCloseBody{propertyCoreBody: body, after: func() { client.Microversion = "" }}), nil
					}
					return propertyCoreHTTP(req, 203, body), nil
				})
				client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				scope = propertyRecordScope(t, client, Dependencies{})
				ack, err := propertyRecordDeleteCall(scope, ctx, operation.all, RecordRequest{ID: propertyRecordChild}, propertyRecordDeleteOptions{})
				if ack == nil || err == nil || ack.StatusCode != 203 || ack.Namespace != propertyRecordParent || string(ack.Body) != bodyText || ack.Header.Get("X-Proof") != "original" || calls != 1 || retries != 0 || body.closes != 1 {
					t.Fatal("accepted fault lost frozen actual receipt", ack, err, calls, retries, body.closes)
				}
				if operation.all {
					if ack.Name != nil {
						t.Fatal("bulk fault invented child")
					}
				} else if ack.Name == nil || *ack.Name != propertyRecordChild {
					t.Fatal("fault changed chosen child", ack.Name)
				}
				proof := propertyCoreProof(t, err, 203, bodyText)
				if mode == "read close cancel" {
					for _, expected := range []error{readErr, closeErr, cause, context.Canceled} {
						if !errors.Is(err, expected) {
							t.Fatal("accepted cause lost", expected, err)
						}
					}
				} else if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal("guard cause lost", err)
				}
				ack.Body[0] = 'X'
				ack.Header.Set("X-Proof", "caller changed")
				if ack.Name != nil {
					*ack.Name = "caller changed"
				}
				if string(proof.Body) != bodyText || proof.Header.Get("X-Proof") != "original" {
					t.Fatal("Ack and error proof alias")
				}
			})
		}
	}
}

func TestMetadefPropertyDeleteRecordsNativeStatusExpansionAndLegacySeparation(t *testing.T) {
	for _, operation := range propertyRecordDeletions {
		t.Run(operation.name+" native expansion", func(t *testing.T) {
			calls, retries := 0, 0
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return propertyCoreJSON(req, 503, "busy"), nil
				}
				return propertyCoreJSON(req, 400, "outside original policy"), nil
			})
			client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				retries++
				if retries > 1 {
					return errors.New("extra retry")
				}
				opts.OkCodes = append(opts.OkCodes, 400)
				return nil
			}
			ack, err := propertyRecordDeleteCall(propertyRecordScope(t, client, Dependencies{}), context.Background(), operation.all, RecordRequest{ID: propertyRecordChild}, propertyRecordDeleteOptions{})
			var native gophercloud.ErrUnexpectedResponseCode
			if ack != nil || !errors.As(err, &native) || native.Actual != 400 || string(native.Body) != "outside original policy" || calls != 2 || retries != 1 {
				t.Fatal("retry expansion established SDK deletion success", ack, err, native, calls, retries)
			}
		})
		for _, code := range []int{202, 404} {
			t.Run(fmt.Sprintf("%s legacy %d", operation.name, code), func(t *testing.T) {
				client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
					return propertyCoreJSON(req, code, "legacy response"), nil
				})
				scope := propertyRecordScope(t, client, Dependencies{})
				var ack *Acknowledgement
				var err error
				if operation.all {
					ack, err = scope.DeleteAll(context.Background())
				} else {
					ack, err = scope.Delete(context.Background(), propertyRecordChild)
				}
				if !operation.all && code == 404 {
					if ack != nil || err != nil {
						t.Fatal("legacy handled missing acquired a receipt", ack, err)
					}
				} else if ack != nil || !gophercloud.ResponseCodeIs(err, code) {
					t.Fatal("legacy strict204 policy changed", ack, err)
				}
			})
		}
	}
}
