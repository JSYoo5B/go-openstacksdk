package groups_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonCinderGroupReplicationStockDefaultsLiteralRoutesAndAck(t *testing.T) {
	for _, action := range []string{"enable", "disable", "failover"} {
		t.Run(action, func(t *testing.T) {
			body := string([]byte{0xff, '{'})
			api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(202, body) })
			var result *groups.ReplicationRecordResult
			var err error
			want := `{"enable_replication":{}}`
			switch action {
			case "enable":
				result, err = api.EnableReplicationRecord(context.Background(), "literal/id %2F")
			case "disable":
				result, err = api.DisableReplicationRecord(context.Background(), "literal/id %2F")
				want = `{"disable_replication":{}}`
			case "failover":
				result, err = api.FailoverReplicationRecord(context.Background(), "literal/id %2F")
				want = `{"failover_replication":{"allow_attached_volume":false,"secondary_backend_id":null}}`
			}
			if err != nil || result == nil || result.Reference != "literal/id %2F" || result.StatusCode != 202 || string(result.Envelope) != body || result.Header.Get("X-Request-Id") != "actual" || len(result.Discovery) != 0 || calls.Len() != 1 || calls.Calls()[0].Method != http.MethodPost || calls.Calls()[0].EscapedPath != groupBase+"/literal%2Fid%20%252F/action" || calls.Calls()[0].RawQuery != "" || calls.Calls()[0].Header.Get("Content-Type") != "application/json" || calls.Calls()[0].Header.Get("OpenStack-API-Version") != "volume 3.38" {
				t.Fatal(result, err, calls.Calls())
			}
			groupJSON(t, calls.Calls()[0].Body, want)
		})
	}
}

func TestPythonCinderGroupReplicationPinnedSourceWireKeepsSchemaRejection(t *testing.T) {
	for _, action := range []string{"enable", "disable", "failover"} {
		t.Run(action, func(t *testing.T) {
			var calls *testcloud.Recorder
			api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response {
				var body map[string]json.RawMessage
				if json.Unmarshal([]byte(calls.Calls()[calls.Len()-1].Body), &body) != nil {
					t.Fatal(calls.Calls())
				}
				// Fixed Cinder27 schema requires object-valued enable/disable and
				// the exact failover_replication top-level key, not Source's typo.
				if action == "failover" {
					if body["modify_body_for_action"] == nil || body["failover_replication"] != nil {
						t.Fatal(body)
					}
				} else if string(body[action+"_replication"]) != "null" {
					t.Fatal(body)
				}
				return groupWire(400, `{"schema":"actual-rejection"}`)
			})
			var result *groups.ReplicationRecordResult
			var err error
			want := `{"enable_replication":null}`
			switch action {
			case "enable":
				result, err = api.EnableReplicationRecord(context.Background(), "id", groups.WithRecordReplicationPinnedSourceWire(true))
			case "disable":
				result, err = api.DisableReplicationRecord(context.Background(), "id", groups.WithRecordReplicationPinnedSourceWire(true))
				want = `{"disable_replication":null}`
			case "failover":
				result, err = api.FailoverReplicationRecord(context.Background(), "id", groups.WithRecordFailoverPinnedSourceWire(true))
				want = `{"modify_body_for_action":{"allow_attached_volume":false,"secondary_backend_id":null}}`
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if result == nil || result.StatusCode != 400 || string(result.Envelope) != `{"schema":"actual-rejection"}` || calls.Len() != 1 || !errors.As(err, &native) || native.Actual != 400 || string(native.Body) != string(result.Envelope) || native.ResponseHeader.Get("X-Request-Id") != "actual" {
				t.Fatal(result, err, calls.Calls())
			}
			groupJSON(t, calls.Calls()[0].Body, want)
		})
	}
}

func TestPythonCinderGroupReplicationFailoverFalseNullEmptyAndOptsReset(t *testing.T) {
	for _, variant := range []string{"default", "explicitfalse_empty", "true_backend", "resetnull"} {
		api, calls := groupAPI(t, "3.90", func(*http.Request) *http.Response { return groupWire(202, "opaque-ack") })
		options := []groups.RecordFailoverOption{}
		want := `{"failover_replication":{"allow_attached_volume":false,"secondary_backend_id":null}}`
		switch variant {
		case "explicitfalse_empty":
			options = append(options, groups.WithRecordFailoverAllowedAttachedVolume(false), groups.WithRecordFailoverSecondaryBackendID(""))
			want = `{"failover_replication":{"allow_attached_volume":false,"secondary_backend_id":""}}`
		case "true_backend":
			options = append(options, groups.WithRecordFailoverAllowedAttachedVolume(true), groups.WithRecordFailoverSecondaryBackendID("backend/% snow"))
			want = `{"failover_replication":{"allow_attached_volume":true,"secondary_backend_id":"backend/% snow"}}`
		case "resetnull":
			options = append(options, groups.WithRecordFailoverSecondaryBackendID("old"), groups.WithRecordFailoverOpts(groups.RecordFailoverOpts{}))
		}
		value, err := api.FailoverReplicationRecord(context.Background(), "id", options...)
		if err != nil || value == nil || value.Reference != "id" || value.StatusCode != 202 || string(value.Envelope) != "opaque-ack" || calls.Len() != 1 || calls.Calls()[0].Header.Get("OpenStack-API-Version") != "volume 3.90" {
			t.Fatal(variant, value, err, calls.Calls())
		}
		groupJSON(t, calls.Calls()[0].Body, want)
	}
}

func TestPythonCinderGroupReplicationOwnedOptionsRetainedMutationAndSelectedAliases(t *testing.T) {
	allow, backend, version := true, "snapshot", "3.90"
	headers := map[string]string{"X-Policy": "owned"}
	option := groups.WithRecordFailoverOpts(groups.RecordFailoverOpts{AllowedAttachedVolume: &allow, SecondaryBackendID: &backend, Microversion: &version, Headers: headers})
	allow = false
	backend = "changed"
	version = "3.37"
	headers["X-Policy"] = "changed"
	var held *groups.RecordFailoverOpts
	api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response {
		if held != nil {
			held.Headers["X-Policy"] = "retained"
			*held.AllowedAttachedVolume = false
			*held.SecondaryBackendID = "retained"
			*held.Microversion = "3.37"
		}
		return groupWire(202, "accepted")
	})
	original := map[string]string{"OpenStack-API-Version": "volume 3.38", "X-OpenStack-Volume-API-Version": "3.38", "X-Source": "source"}
	expected := map[string]string{"OpenStack-API-Version": "volume 3.38", "X-OpenStack-Volume-API-Version": "3.38", "X-Source": "source"}
	api.RawClient().MoreHeaders = original
	for range 2 {
		value, err := api.FailoverReplicationRecord(context.Background(), "id", option, func(c *groups.RecordFailoverOpts) error { held = c; return nil })
		if err != nil || value == nil || !reflect.DeepEqual(api.RawClient().MoreHeaders, expected) || api.RawClient().Microversion != "3.38" {
			t.Fatal(value, err, calls.Calls(), api.RawClient())
		}
	}
	if calls.Len() != 2 {
		t.Fatal(calls.Calls())
	}
	for _, call := range calls.Calls() {
		if call.Header.Get("X-Policy") != "owned" || call.Header.Get("X-Source") != "source" || call.Header.Get("OpenStack-API-Version") != "volume 3.90" || call.Header.Get("X-OpenStack-Volume-API-Version") != "3.90" {
			t.Fatal(call)
		}
		groupJSON(t, call.Body, `{"failover_replication":{"allow_attached_volume":true,"secondary_backend_id":"snapshot"}}`)
	}
}

func TestPythonCinderGroupReplicationFirstCustomOptionsOwnNilHeaders(t *testing.T) {
	for _, failover := range []bool{false, true} {
		api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(202, "ack") })
		var value *groups.ReplicationRecordResult
		var err error
		if failover {
			value, err = api.FailoverReplicationRecord(context.Background(), "id", groups.WithRecordFailoverHeaders(nil), func(c *groups.RecordFailoverOpts) error {
				if c.Headers == nil || len(c.Headers) != 0 {
					t.Fatal(c)
				}
				c.Headers["X-First"] = "owned"
				return nil
			})
		} else {
			value, err = api.EnableReplicationRecord(context.Background(), "id", groups.WithRecordReplicationOpts(groups.RecordReplicationOpts{Headers: nil}), func(c *groups.RecordReplicationOpts) error {
				if c.Headers == nil || len(c.Headers) != 0 {
					t.Fatal(c)
				}
				c.Headers["X-First"] = "owned"
				return nil
			})
		}
		if err != nil || value == nil || calls.Len() != 1 || calls.Calls()[0].Header.Get("X-First") != "owned" || len(api.RawClient().MoreHeaders) != 0 {
			t.Fatal(value, err, calls.Calls(), api.RawClient().MoreHeaders)
		}
	}
}

func TestPythonCinderGroupReplicationMinimumRejectsBeforeMutationAndOverrideWorks(t *testing.T) {
	for _, selected := range []string{"3.37", "latest", "4.0"} {
		api, calls := groupAPI(t, selected, func(*http.Request) *http.Response { t.Fatal("unsupported action HTTP"); return nil })
		value, err := api.EnableReplicationRecord(context.Background(), "id")
		if value != nil || !errors.Is(err, resource.ErrUnsupported) || calls.Len() != 0 {
			t.Fatal(selected, value, err, calls.Calls())
		}
	}
	api, calls := groupAPI(t, "3.37", func(*http.Request) *http.Response { return groupWire(202, "ack") })
	value, err := api.DisableReplicationRecord(context.Background(), "id", groups.WithRecordReplicationMicroversion("3.38"))
	if err != nil || value == nil || calls.Len() != 1 || calls.Calls()[0].Header.Get("OpenStack-API-Version") != "volume 3.38" || api.RawClient().Microversion != "3.37" {
		t.Fatal(value, err, calls.Calls())
	}
}

func TestPythonCinderGroupReplicationDefaultDiscoveryCeilingAndUnsupportedProof(t *testing.T) {
	for _, bounds := range []struct {
		maximum, minimum string
		supported        bool
	}{{"3.90", "3.0", true}, {"3.37", "3.0", false}, {"3.90", "3.50", false}, {"3.90", "", false}} {
		body := `{"version":{"id":"v3.0","version":"` + bounds.maximum + `","min_version":"` + bounds.minimum + `"}}`
		api, calls := groupAPI(t, "", func(r *http.Request) *http.Response {
			if r.URL.Path == "/cinder/v3/" {
				return groupWire(200, body)
			}
			return groupWire(202, "action")
		})
		value, err := api.EnableReplicationRecord(context.Background(), "id")
		if value == nil || len(value.Discovery) != 1 || value.Discovery[0].StatusCode != 200 || string(value.Discovery[0].Envelope) != body || value.Discovery[0].Header.Get("X-Request-Id") != "actual" || api.RawClient().Microversion != "" {
			t.Fatal(bounds, value, err, calls.Calls())
		}
		if bounds.supported {
			if err != nil || calls.Len() != 2 || value.StatusCode != 202 || string(value.Envelope) != "action" || calls.Calls()[1].Header.Get("OpenStack-API-Version") != "volume 3.38" {
				t.Fatal(bounds, value, err, calls.Calls())
			}
		} else {
			var proof *resource.ResponseError
			if !errors.Is(err, resource.ErrUnsupported) || calls.Len() != 1 || value.StatusCode != 0 || value.Envelope != nil || value.Header != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body {
				t.Fatal(bounds, value, err, proof, calls.Calls())
			}
		}
	}
}

func TestPythonCinderGroupReplicationDiscoveryDenialIsSeparateFromAction(t *testing.T) {
	for _, code := range []int{400, 403, 503} {
		api, calls := groupAPI(t, "", func(*http.Request) *http.Response { return groupWire(code, `{"discovery":"denied"}`) })
		value, err := api.FailoverReplicationRecord(context.Background(), "id")
		var native gophercloud.ErrUnexpectedResponseCode
		if value == nil || value.Reference != "id" || value.StatusCode != 0 || value.Envelope != nil || value.Header != nil || len(value.Discovery) != 1 || value.Discovery[0].StatusCode != code || string(value.Discovery[0].Envelope) != `{"discovery":"denied"}` || calls.Len() != 1 || calls.Calls()[0].Method != http.MethodGet || !errors.As(err, &native) || native.Actual != code || native.ResponseHeader.Get("X-Request-Id") != "actual" {
			t.Fatal(value, err, calls.Calls())
		}
	}
}

func TestPythonCinderGroupReplicationAllSourceAdmittedCodesIgnoreResponseJSON(t *testing.T) {
	for _, code := range []int{100, 202, 299, 399} {
		api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(code, `{"group":null,"status":"not-completion"}`) })
		value, err := api.DisableReplicationRecord(context.Background(), "id")
		if err != nil || value == nil || value.Reference != "id" || value.StatusCode != code || string(value.Envelope) != `{"group":null,"status":"not-completion"}` || calls.Len() != 1 {
			t.Fatal(code, value, err, calls.Calls())
		}
	}
}

func TestPythonCinderGroupReplicationPhysicalCausesSourceRestoreAndCancelKeepActualReceipt(t *testing.T) {
	for _, code := range []int{202, 400, 403, 404, 503} {
		readCause, closeCause, cancelCause := errors.New("read actual"), errors.New("close actual"), errors.New("custom cancel")
		ctx, cancel := context.WithCancelCause(context.Background())
		faulty := &testcloud.Body{Data: `{"partial":`, ReadErr: readCause, CloseErr: closeCause}
		api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response {
			response := groupWire(code, "")
			response.Body = faulty
			return response
		})
		original := api.RawClient().ResourceBase
		faulty.OnRead = func() { api.RawClient().ResourceBase = original + "changed/" }
		faulty.OnClose = func() { api.RawClient().ResourceBase = original; cancel(cancelCause) }
		hooks := 0
		api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			hooks++
			return nil
		}
		value, err := api.EnableReplicationRecord(ctx, "id")
		if value == nil || value.StatusCode != code || value.Reference != "id" || string(value.Envelope) != faulty.Data || value.Header.Get("X-Request-Id") != "actual" || calls.Len() != 1 || faulty.Closes() != 1 || hooks != 0 {
			t.Fatal(code, value, err, calls.Calls(), faulty.Closes(), hooks)
		}
		for _, cause := range []error{readCause, closeCause, cancelCause, context.Canceled, resource.ErrInvalidOption} {
			if !errors.Is(err, cause) {
				t.Fatal(code, cause, err)
			}
		}
		if code < 400 {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != faulty.Data {
				t.Fatal(err, proof)
			}
		} else {
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || string(native.Body) != faulty.Data {
				t.Fatal(err, native)
			}
		}
		cancel(nil)
	}
}

func TestPythonCinderGroupReplicationRejectionsNeverIgnoreOrLookup(t *testing.T) {
	for _, code := range []int{400, 403, 404, 409, 500, 503} {
		api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(code, `{"action":"actual-failure"}`) })
		value, err := api.FailoverReplicationRecord(context.Background(), "id")
		var native gophercloud.ErrUnexpectedResponseCode
		if value == nil || value.StatusCode != code || calls.Len() != 1 || calls.Calls()[0].Method != http.MethodPost || !errors.As(err, &native) || native.Actual != code || string(native.Body) != `{"action":"actual-failure"}` || native.ResponseHeader.Get("X-Request-Id") != "actual" {
			t.Fatal(code, value, err, calls.Calls())
		}
	}
}

func TestPythonCinderGroupReplicationNativeRetryOwnsBodyHeadersAndPreservesLiveToken(t *testing.T) {
	for _, mode := range []string{"version", "nova", "authorization", "body", "content_type", "omit_version", "omit_content_type"} {
		t.Run(mode, func(t *testing.T) {
			api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(503, `{"original":"rejection"}`) })
			api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				switch mode {
				case "version":
					opts.MoreHeaders["OpenStack-API-Version"] = "volume 3.99"
				case "nova":
					opts.MoreHeaders["X-OpenStack-Nova-API-Version"] = "2.99"
				case "authorization":
					opts.MoreHeaders["Authorization"] = "forged"
				case "body":
					opts.JSONBody = map[string]any{"enable_replication": map[string]any{"extra": true}}
				case "content_type":
					opts.MoreHeaders["Content-Type"] = "text/plain"
				case "omit_version":
					opts.OmitHeaders = append(opts.OmitHeaders, "OpenStack-API-Version")
				case "omit_content_type":
					opts.OmitHeaders = append(opts.OmitHeaders, "Content-Type")
				}
				return nil
			}
			value, err := api.EnableReplicationRecord(context.Background(), "id")
			var native gophercloud.ErrUnexpectedResponseCode
			if value == nil || value.StatusCode != 503 || string(value.Envelope) != `{"original":"rejection"}` || calls.Len() != 1 || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != string(value.Envelope) {
				t.Fatal(mode, value, err, calls.Calls())
			}
		})
	}
	attempt := 0
	api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response {
		attempt++
		if attempt == 1 {
			return groupWire(503, "first")
		}
		return groupWire(202, "accepted")
	})
	api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
		opts.MoreHeaders["X-Retry"] = "ordinary"
		api.RawClient().SetToken("live-token")
		return nil
	}
	value, err := api.EnableReplicationRecord(context.Background(), "id")
	if err != nil || value == nil || value.StatusCode != 202 || calls.Len() != 2 || calls.Calls()[1].Header.Get("X-Retry") != "ordinary" || calls.Calls()[1].Header.Get("X-Auth-Token") != "live-token" {
		t.Fatal(value, err, calls.Calls())
	}
	for _, call := range calls.Calls() {
		groupJSON(t, call.Body, `{"enable_replication":{}}`)
	}
}

func TestPythonCinderGroupReplicationPreflightOptionsAndOriginalCauses(t *testing.T) {
	for _, invoke := range []func(*groups.API) error{
		func(api *groups.API) error { _, e := api.EnableReplicationRecord(nil, "id"); return e },
		func(api *groups.API) error {
			_, e := (*groups.API)(nil).EnableReplicationRecord(context.Background(), "id")
			return e
		},
		func(api *groups.API) error { _, e := api.DisableReplicationRecord(context.Background(), " "); return e },
		func(api *groups.API) error {
			_, e := api.EnableReplicationRecord(context.Background(), "id", nil)
			return e
		},
		func(api *groups.API) error {
			_, e := api.FailoverReplicationRecord(context.Background(), "id", nil)
			return e
		},
		func(api *groups.API) error {
			_, e := api.EnableReplicationRecord(context.Background(), "id", groups.WithRecordReplicationMicroversion("invalid"))
			return e
		},
		func(api *groups.API) error {
			_, e := api.EnableReplicationRecord(context.Background(), "id", groups.WithRecordReplicationHeaders(map[string]string{"Authorization": "forged"}))
			return e
		},
		func(api *groups.API) error {
			_, e := api.EnableReplicationRecord(context.Background(), "id", groups.WithRecordReplicationHeaders(map[string]string{"OpenStack-API-Version": "volume 3.38"}))
			return e
		},
		func(api *groups.API) error {
			_, e := api.FailoverReplicationRecord(context.Background(), "id", groups.WithRecordFailoverSecondaryBackendID(string([]byte{0xff})))
			return e
		},
	} {
		api, calls := groupAPI(t, "", func(*http.Request) *http.Response { t.Fatal("preflight HTTP"); return nil })
		if err := invoke(api); !errors.Is(err, resource.ErrInvalidOption) || calls.Len() != 0 {
			t.Fatal(err, calls.Calls())
		}
	}
	api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { t.Fatal("source option HTTP"); return nil })
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("option cause")
	value, err := api.EnableReplicationRecord(ctx, "id", func(*groups.RecordReplicationOpts) error {
		api.RawClient().ResourceBase = "https://other.example/v3/project/"
		cancel(cause)
		return cause
	})
	if value != nil || calls.Len() != 0 || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(value, err, calls.Calls())
	}
}

func TestPythonCinderGroupReplicationActualEvidenceOwnsDiscoveryAndAction(t *testing.T) {
	api, calls := groupAPI(t, "", func(r *http.Request) *http.Response {
		if strings.HasSuffix(r.URL.Path, "/action") {
			return groupWire(202, `{"receipt":"action"}`)
		}
		return groupWire(200, `{"version":{"id":"v3.0","version":"3.90","min_version":"3.0"}}`)
	})
	first, err := api.EnableReplicationRecord(context.Background(), "id")
	if err != nil || first == nil || len(first.Discovery) != 1 {
		t.Fatal(first, err, calls.Calls())
	}
	first.Envelope[0] = 'X'
	first.Header.Set("X-Request-Id", "changed")
	first.Discovery[0].Envelope[0] = 'X'
	first.Discovery[0].Header.Set("X-Request-Id", "changed")
	second, err := api.DisableReplicationRecord(context.Background(), "id")
	if err != nil || second == nil || calls.Len() != 4 || len(second.Discovery) != 1 || string(second.Envelope) != `{"receipt":"action"}` || second.Header.Get("X-Request-Id") != "actual" || second.Discovery[0].Envelope[0] != '{' || second.Discovery[0].Header.Get("X-Request-Id") != "actual" || api.RawClient().Microversion != "" {
		t.Fatal(first, second, err, calls.Calls())
	}
}

func TestPythonCinderGroupReplicationFaulty401StopsCallbacksAndClean401RenewsToken(t *testing.T) {
	for _, mode := range []string{"read", "close", "both"} {
		t.Run(mode, func(t *testing.T) {
			readCause, closeCause := errors.New("actual401 read"), errors.New("actual401 close")
			faulty := &testcloud.Body{Data: `{"expired":"actual401"}`}
			if mode != "close" {
				faulty.ReadErr = readCause
			}
			if mode != "read" {
				faulty.CloseErr = closeCause
			}
			api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response {
				response := groupWire(401, "")
				response.Body = faulty
				return response
			})
			retry, reauth := 0, 0
			api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retry++
				return nil
			}
			api.RawClient().ReauthFunc = func(context.Context) error { reauth++; api.RawClient().SetToken("renewed"); return nil }
			value, err := api.DisableReplicationRecord(context.Background(), "id")
			var authentication *gophercloud.ErrUnableToReauthenticate
			var native gophercloud.ErrUnexpectedResponseCode
			if value == nil || value.StatusCode != 401 || string(value.Envelope) != faulty.Data || value.Header.Get("X-Request-Id") != "actual" || calls.Len() != 1 || retry != 0 || reauth != 0 || faulty.Closes() != 1 || !errors.As(err, &authentication) || !errors.As(authentication.ErrOriginal, &native) || native.Actual != 401 || string(native.Body) != faulty.Data || native.ResponseHeader.Get("X-Request-Id") != "actual" || native.Method != http.MethodPost || !strings.HasSuffix(native.URL, groupBase+"/id/action") {
				t.Fatal(mode, value, err, calls.Calls(), retry, reauth, faulty.Closes())
			}
			if mode != "close" && (!errors.Is(err, readCause) || !errors.Is(authentication.ErrReauth, readCause)) || mode != "read" && (!errors.Is(err, closeCause) || !errors.Is(authentication.ErrReauth, closeCause)) {
				t.Fatal(mode, err)
			}
			groupJSON(t, calls.Calls()[0].Body, `{"disable_replication":{}}`)
		})
	}
	attempt := 0
	api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response {
		attempt++
		if attempt == 1 {
			return groupWire(401, `{"expired":"clean"}`)
		}
		return groupWire(202, "ack")
	})
	renewals := 0
	api.RawClient().ReauthFunc = func(context.Context) error { renewals++; api.RawClient().SetToken("renewed"); return nil }
	value, err := api.FailoverReplicationRecord(context.Background(), "id", groups.WithRecordFailoverSecondaryBackendID("backend"))
	if err != nil || value == nil || value.StatusCode != 202 || calls.Len() != 2 || renewals != 1 || calls.Calls()[1].Header.Get("X-Auth-Token") != "renewed" {
		t.Fatal(value, err, calls.Calls(), renewals)
	}
	for _, call := range calls.Calls() {
		if call.Method != http.MethodPost || call.Header.Get("OpenStack-API-Version") != "volume 3.38" {
			t.Fatal(call)
		}
		groupJSON(t, call.Body, `{"failover_replication":{"allow_attached_volume":false,"secondary_backend_id":"backend"}}`)
	}
}
