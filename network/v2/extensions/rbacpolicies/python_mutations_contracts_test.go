package rbacpolicies_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonRBACMutationField(t *testing.T, v *rbacpolicies.RBACPolicyRecord, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		t.Fatalf("field %s want %s, got %#v", key, want, v)
	}
}
func pythonRBACMutationBody(t *testing.T, raw string, want map[string]any) {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal([]byte(raw), &v); err != nil || !reflect.DeepEqual(v, map[string]any{"rbac_policy": want}) {
		t.Fatal(raw, want, err)
	}
}
func pythonRBACMutationRun(ctx context.Context, api *rbacpolicies.API, op string) error {
	switch op {
	case "create":
		_, err := api.CreateRecord(ctx, rbacpolicies.WithRecordCreateField("target_project_id", "project"))
		return err
	case "update":
		_, err := api.UpdateRecord(ctx, "target", rbacpolicies.WithRecordUpdateField("target_project_id", "project"))
		return err
	default:
		_, err := api.DeleteRecord(ctx, "target", rbacpolicies.WithRecordDeleteIgnoreMissing(false))
		return err
	}
}

func TestPythonRBACMutationCreateProvidedAttrsAndEmptyEnvelope(t *testing.T) {
	var calls []nativeRBACCall
	body := `{"rbac_policy":{"id":"created","target_tenant":"server-target","project_id":null,"location":"forged","x_response":false}}`
	api, _ := nativeRBACAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.Method != http.MethodPost || req.URL.Path != "/neutron/v2.0/rbac-policies" || req.URL.RawQuery != "" {
			t.Fatal(req.Method, req.URL)
		}
		r := nativeRBACWire(203, body)
		r.Header.Set("X-Request-Id", "create-proof")
		return r
	})
	value, err := api.CreateRecord(context.Background(), rbacpolicies.WithRecordCreateField("target_project_id", nil), rbacpolicies.WithRecordCreateField("action", false), rbacpolicies.WithRecordCreateField("object_id", 0), rbacpolicies.WithRecordCreateField("object_type", map[string]any{"extension": true}), rbacpolicies.WithRecordCreateField("x_ext", []any{nil, true, 0}), rbacpolicies.WithRecordCreateField("if_none_match", "body-value"))
	if err != nil || value == nil || len(calls) != 1 || value.StatusCode != 203 || string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "create-proof" {
		t.Fatal(value, err, calls)
	}
	pythonRBACMutationBody(t, calls[0].body, map[string]any{"target_tenant": nil, "action": false, "object_id": float64(0), "object_type": map[string]any{"extension": true}, "x_ext": []any{nil, true, float64(0)}, "if_none_match": "body-value"})
	pythonRBACMutationField(t, value, "target_project_id", `"server-target"`)
	pythonRBACMutationField(t, value, "action", `false`)
	pythonRBACMutationField(t, value, "object_id", `0`)
	pythonRBACMutationField(t, value, "name", `null`)
	pythonRBACMutationField(t, value, "location", `null`)
	if value.Resource.Body["target_tenant"] != nil || value.Resource.Body["tags"] != nil || value.Resource.Body["if_match"] != nil || string(value.Wire.Body["location"]) != `"forged"` {
		t.Fatal(value)
	}
	value.Resource.Body["target_project_id"][1] = 'x'
	value.Header.Set("X-Request-Id", "changed")
	if string(value.Wire.Body["target_tenant"]) != `"server-target"` || value.Wire.Header.Get("X-Request-Id") != "create-proof" || string(value.Envelope) != body {
		t.Fatal("aliased receipt", value)
	}
	calls = nil
	value, err = api.CreateRecord(context.Background())
	if err != nil || value == nil || len(calls) != 1 {
		t.Fatal(value, err, calls)
	}
	pythonRBACMutationBody(t, calls[0].body, map[string]any{})
}

func TestPythonRBACMutationOrderedAliasesAndTenantFallback(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		var calls []nativeRBACCall
		api, _ := nativeRBACAPI(t, &calls, func(*http.Request) *http.Response { return nativeRBACWire(204, "") })
		first, second := rbacpolicies.WithRecordCreateField("target_project_id", "canonical"), rbacpolicies.WithRecordCreateField("target_tenant", "wire")
		want := "wire"
		if reverse {
			first, second = second, first
			want = "canonical"
		}
		value, err := api.CreateRecord(context.Background(), first, second, rbacpolicies.WithRecordCreateField("tenant_id", "owner"))
		if err != nil || value.Wire != nil || len(calls) != 1 {
			t.Fatal(value, err, calls)
		}
		pythonRBACMutationBody(t, calls[0].body, map[string]any{"target_tenant": want, "tenant_id": "owner"})
		pythonRBACMutationField(t, value, "target_project_id", fmt.Sprintf("%q", want))
		pythonRBACMutationField(t, value, "tenant_id", `"owner"`)
		pythonRBACMutationField(t, value, "project_id", `"owner"`)
	}
	var calls []nativeRBACCall
	api, _ := nativeRBACAPI(t, &calls, func(*http.Request) *http.Response { return nativeRBACWire(202, "{not-json") })
	value, err := api.CreateRecord(context.Background(), rbacpolicies.WithRecordCreateField("project_id", nil), rbacpolicies.WithRecordCreateField("tenant_id", "tenant"), rbacpolicies.WithRecordCreateField("id", false))
	if err != nil || value.Wire != nil || string(value.Envelope) != "{not-json" {
		t.Fatal(value, err)
	}
	pythonRBACMutationField(t, value, "project_id", `null`)
	pythonRBACMutationField(t, value, "tenant_id", `"tenant"`)
	pythonRBACMutationField(t, value, "id", `false`)
}

func TestPythonRBACMutationUpdateNoopAndExplicitNull(t *testing.T) {
	var calls []nativeRBACCall
	api, _ := nativeRBACAPI(t, &calls, func(*http.Request) *http.Response { return nativeRBACWire(204, "") })
	for _, options := range [][]rbacpolicies.RecordUpdateOption{nil, {rbacpolicies.WithRecordUpdateField("microversion", map[string]any{"ignored": true})}, {rbacpolicies.WithRecordUpdateHeaders(map[string]string{"If-Match": "condition"}), rbacpolicies.WithRecordUpdateMicroversion("2.0")}} {
		value, err := api.UpdateRecord(context.Background(), "literal/id %2F", options...)
		if err != nil || value == nil || value.Wire != nil || value.StatusCode != 0 || value.Header != nil && len(value.Header) != 0 || len(value.Envelope) != 0 || len(calls) != 0 {
			t.Fatal(value, err, calls)
		}
		pythonRBACMutationField(t, value, "id", `"literal/id %2F"`)
		pythonRBACMutationField(t, value, "target_project_id", `null`)
		pythonRBACMutationField(t, value, "location", `null`)
		if value.Resource.Body["microversion"] != nil {
			t.Fatal(value.Resource.Body)
		}
	}
	value, err := api.UpdateRecord(context.Background(), "literal/id %2F", rbacpolicies.WithRecordUpdateField("target_project_id", nil), rbacpolicies.WithRecordUpdateField("x_changed", false))
	if err != nil || value == nil || len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].query != "" {
		t.Fatal(value, err, calls)
	}
	pythonRBACMutationBody(t, calls[0].body, map[string]any{"target_tenant": nil, "x_changed": false})
	pythonRBACMutationField(t, value, "target_project_id", `null`)
	pythonRBACMutationField(t, value, "x_changed", `false`)
}

func TestPythonRBACMutationLiteralIdentityAndServerOwnedSchema(t *testing.T) {
	var calls []nativeRBACCall
	identity := "政策/id %2F"
	api, _ := nativeRBACAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.EscapedPath() != "/neutron/v2.0/rbac-policies/"+url.PathEscape(identity) || req.URL.RawQuery != "" {
			t.Fatal(req.URL)
		}
		r := nativeRBACWire(400, `{"error":"stock immutable object_type"}`)
		r.Header.Set("X-Request-Id", "stock-schema")
		return r
	})
	value, err := api.UpdateRecord(context.Background(), identity, rbacpolicies.WithRecordUpdateField("object_type", "network"), rbacpolicies.WithRecordUpdateField("action", "extension-action"))
	var native gophercloud.ErrUnexpectedResponseCode
	if value != nil || !errors.As(err, &native) || native.Actual != 400 || native.Method != http.MethodPut || native.ResponseHeader.Get("X-Request-Id") != "stock-schema" || len(calls) != 1 {
		t.Fatal(value, err, native, calls)
	}
	pythonRBACMutationBody(t, calls[0].body, map[string]any{"object_type": "network", "action": "extension-action"})
	for _, id := range []string{"", " ", ".", "..", "bad\n", string([]byte{255})} {
		before := len(calls)
		_, err := api.UpdateRecord(context.Background(), id)
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
			t.Fatal(id, err, calls)
		}
		_, err = api.DeleteRecord(context.Background(), id)
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
			t.Fatal(id, err, calls)
		}
	}
}

func TestPythonRBACMutationBodyHeaderAndVersionOwnership(t *testing.T) {
	for _, op := range []string{"create", "update"} {
		var calls []nativeRBACCall
		api, _ := nativeRBACAPI(t, &calls, func(req *http.Request) *http.Response {
			if req.Header.Get("If-None-Match") != "header-condition" || req.Header.Get("If-Match") != "header-match" || req.Header.Get("OpenStack-API-Version") != "network 2.1" {
				t.Fatal(req.Header)
			}
			return nativeRBACWire(200, `{"rbac_policy":{"microversion":"forged","if_match":"received-body"}}`)
		})
		source := api.RawClient()
		source.Microversion = "2.0"
		source.MoreHeaders = map[string]string{"Openstack-Api-Version": "network 2.0"}
		var value *rbacpolicies.RBACPolicyRecord
		var err error
		headers := map[string]string{"If-None-Match": "header-condition", "If-Match": "header-match"}
		if op == "create" {
			value, err = api.CreateRecord(context.Background(), rbacpolicies.WithRecordCreateField("if_none_match", nil), rbacpolicies.WithRecordCreateField("if_match", "body-condition"), rbacpolicies.WithRecordCreateField("microversion", false), rbacpolicies.WithRecordCreateHeaders(headers), rbacpolicies.WithRecordCreateMicroversion("2.1"))
		} else {
			value, err = api.UpdateRecord(context.Background(), "target", rbacpolicies.WithRecordUpdateField("if_none_match", nil), rbacpolicies.WithRecordUpdateField("if_match", "body-condition"), rbacpolicies.WithRecordUpdateField("microversion", false), rbacpolicies.WithRecordUpdateHeaders(headers), rbacpolicies.WithRecordUpdateMicroversion("2.1"))
		}
		if err != nil || value == nil || len(calls) != 1 || source.Microversion != "2.0" || source.MoreHeaders["Openstack-Api-Version"] != "network 2.0" {
			t.Fatal(op, value, err, calls, source)
		}
		pythonRBACMutationBody(t, calls[0].body, map[string]any{"if_none_match": nil, "if_match": "body-condition"})
		pythonRBACMutationField(t, value, "microversion", `"2.1"`)
		pythonRBACMutationField(t, value, "if_match", `"received-body"`)
		if string(value.Wire.Body["microversion"]) != `"forged"` {
			t.Fatal(value)
		}
	}
}

func TestPythonRBACMutationDeleteMissingDefaultsAndReceipt(t *testing.T) {
	for _, status := range []int{200, 204, 302, 404} {
		var calls []nativeRBACCall
		body := `{"error":"actual delete receipt"}`
		api, _ := nativeRBACAPI(t, &calls, func(req *http.Request) *http.Response {
			if req.Method != http.MethodDelete || req.Body != nil || req.URL.RawQuery != "" {
				t.Fatal(req.Method, req.Body, req.URL)
			}
			r := nativeRBACWire(status, body)
			r.Header.Set("X-Request-Id", "delete-proof")
			return r
		})
		value, err := api.DeleteRecord(context.Background(), "literal/id %2F")
		if err != nil || value == nil || value.StatusCode != status || value.IgnoredMissing != (status == 404) || string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "delete-proof" || len(calls) != 1 || calls[0].body != "" {
			t.Fatal(value, err, calls)
		}
		calls = nil
		value, err = api.DeleteRecord(context.Background(), "literal/id %2F", rbacpolicies.WithRecordDeleteIgnoreMissing(false))
		if status == 404 {
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != 404 || native.ResponseHeader.Get("X-Request-Id") != "delete-proof" {
				t.Fatal(value, err, native)
			}
		} else if err != nil {
			t.Fatal(value, err)
		}
		if len(calls) != 1 {
			t.Fatal(calls)
		}
	}
}

func TestPythonRBACMutationAllRejectedStatusAndMalformedProof(t *testing.T) {
	for _, op := range []string{"create", "update", "delete"} {
		var calls []nativeRBACCall
		status := 400
		body := `{"error":"native receipt"}`
		api, _ := nativeRBACAPI(t, &calls, func(*http.Request) *http.Response {
			r := nativeRBACWire(status, body)
			r.Header.Set("X-Request-Id", "all-status")
			return r
		})
		for code := 400; code <= 599; code++ {
			status = code
			calls = nil
			err := pythonRBACMutationRun(context.Background(), api, op)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "all-status" || len(calls) != 1 {
				t.Fatal(op, code, err, native, calls)
			}
		}
		if op == "delete" {
			continue
		}
		status = 299
		for _, bad := range []string{`null`, `[]`, `false`, `{"rbac_policy":null}`, `{"rbac_policy":[]}`, "{\"rbac_policy\":{\"id\":\"\xff\"}}"} {
			body = bad
			calls = nil
			err := pythonRBACMutationRun(context.Background(), api, op)
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 299 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "all-status" || len(calls) != 1 {
				t.Fatal(op, bad, err, proof, calls)
			}
		}
	}
}

type pythonRBACMutationFaultBody struct {
	data            string
	used            bool
	closes          int
	read, close     error
	onRead, onClose func()
}

func (b *pythonRBACMutationFaultBody) Read(p []byte) (int, error) {
	if b.used {
		return 0, io.EOF
	}
	b.used = true
	if b.onRead != nil {
		b.onRead()
	}
	return copy(p, b.data), b.read
}
func (b *pythonRBACMutationFaultBody) Close() error {
	b.closes++
	if b.onClose != nil {
		b.onClose()
	}
	return b.close
}

func TestPythonRBACMutationMixedPhysicalFaultsNeverRetryOrIgnore(t *testing.T) {
	for _, op := range []string{"create", "update", "delete"} {
		for _, status := range []int{200, 299, 399, 401, 404, 503, 599} {
			t.Run(fmt.Sprintf("%s/%d", op, status), func(t *testing.T) {
				var calls []nativeRBACCall
				readCause, closeCause, cancelCause := errors.New("read fault"), errors.New("close fault"), errors.New("custom cancel")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				body := `{"rbac_policy":{"id":"target"}}`
				physical := &pythonRBACMutationFaultBody{data: body, read: readCause, close: closeCause}
				api, cloud := nativeRBACAPI(t, &calls, func(*http.Request) *http.Response {
					r := nativeRBACWire(status, body)
					r.Body = physical
					r.Header.Set("X-Request-Id", "mixed-proof")
					return r
				})
				source := api.RawClient()
				original := source.ResourceBase
				physical.onRead = func() { source.ResourceBase = original + "changed/" }
				physical.onClose = func() { source.ResourceBase = original; cancel(cancelCause) }
				retries, reauths := 0, 0
				cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				cloud.Provider.ReauthFunc = func(context.Context) error { reauths++; return nil }
				var err error
				if op == "delete" {
					_, err = api.DeleteRecord(ctx, "target")
				} else {
					err = pythonRBACMutationRun(ctx, api, op)
				}
				if err == nil || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 || physical.closes != 1 || retries != 0 || reauths != 0 {
					t.Fatal(err, calls, physical.closes, retries, reauths)
				}
				method := http.MethodPost
				target := original + "rbac-policies"
				if op == "update" {
					method = http.MethodPut
					target += "/target"
				}
				if op == "delete" {
					method = http.MethodDelete
					target += "/target"
				}
				if status < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "mixed-proof" {
						t.Fatal(err, proof)
					}
				} else {
					receiptErr := err
					if status == 401 {
						var nativeReauth *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &nativeReauth) || !errors.Is(nativeReauth.ErrReauth, readCause) || !errors.Is(nativeReauth.ErrReauth, closeCause) || !errors.Is(nativeReauth.ErrReauth, cancelCause) || !errors.Is(nativeReauth.ErrReauth, resource.ErrInvalidOption) {
							t.Fatal(err, nativeReauth)
						}
						receiptErr = nativeReauth.ErrOriginal
					}
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(receiptErr, &native) || native.Actual != status || native.Method != method || native.URL != target || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "mixed-proof" {
						t.Fatal(err, native, target)
					}
				}
			})
		}
	}
}

func TestPythonRBACMutationCleanRetriesKeepBodyRouteAndLiveToken(t *testing.T) {
	for _, op := range []string{"create", "update", "delete"} {
		for _, mode := range []string{"retry", "reauth"} {
			var calls []nativeRBACCall
			var provider *gophercloud.ProviderClient
			api, cloud := nativeRBACAPI(t, &calls, func(req *http.Request) *http.Response {
				if len(calls) == 1 {
					status := 503
					if mode == "reauth" {
						status = 401
					}
					return nativeRBACWire(status, `{"error":"first"}`)
				}
				if req.Header.Get("X-Auth-Token") != "refreshed" || req.Header.Get("OpenStack-API-Version") != "" || calls[0] != calls[1] {
					t.Fatal(req.Header, calls)
				}
				if op != "delete" {
					pythonRBACMutationBody(t, calls[1].body, map[string]any{"target_tenant": "project"})
				}
				return nativeRBACWire(203, `{"rbac_policy":{"id":"target"}}`)
			})
			provider = cloud.Provider
			provider.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, count uint) error {
				if count > 1 {
					return original
				}
				provider.SetToken("refreshed")
				return nil
			}
			provider.ReauthFunc = func(context.Context) error { provider.SetToken("refreshed"); return nil }
			err := pythonRBACMutationRun(context.Background(), api, op)
			if err != nil || len(calls) != 2 {
				t.Fatal(op, mode, err, calls)
			}
		}
	}
}

func TestPythonRBACMutationRetryCannotChangeOwnedProtocol(t *testing.T) {
	for _, op := range []string{"create", "update", "delete"} {
		for _, mutation := range []string{"body", "version", "foreign-version", "authorization", "condition", "omit"} {
			var calls []nativeRBACCall
			api, cloud := nativeRBACAPI(t, &calls, func(*http.Request) *http.Response {
				r := nativeRBACWire(503, `{"error":"first"}`)
				r.Header.Set("X-Request-Id", "retry-proof")
				return r
			})
			api.RawClient().MoreHeaders = map[string]string{"X-Trace": "owned"}
			cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				if o.MoreHeaders == nil {
					o.MoreHeaders = make(map[string]string)
				}
				switch mutation {
				case "body":
					o.JSONBody = map[string]any{"forged": true}
				case "version":
					o.MoreHeaders["Openstack-Api-Version"] = "network 9.9"
				case "foreign-version":
					o.MoreHeaders["X-OpenStack-Volume-API-Version"] = "3.99"
				case "authorization":
					o.MoreHeaders["Authorization"] = "forged"
				case "condition":
					o.MoreHeaders["If-Match"] = "new"
				case "omit":
					o.OmitHeaders = append(o.OmitHeaders, "X-Trace")
				}
				return nil
			}
			err := pythonRBACMutationRun(context.Background(), api, op)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"error":"first"}` || native.ResponseHeader.Get("X-Request-Id") != "retry-proof" || len(calls) != 1 {
				t.Fatal(op, mutation, err, native, calls)
			}
		}
	}
}

type pythonRBACMutationMarshalFailure struct{ cause error }

func (v pythonRBACMutationMarshalFailure) MarshalJSON() ([]byte, error) { return nil, v.cause }

func TestPythonRBACMutationOptionsSnapshotAndMarshalCause(t *testing.T) {
	raw := json.RawMessage(`"owned"`)
	attrs := map[string]json.RawMessage{"target_project_id": raw}
	headers := map[string]string{"X-Trace": "owned"}
	version := "2.0"
	option := rbacpolicies.WithRecordCreateOpts(rbacpolicies.RecordCreateOpts{Attrs: attrs, Headers: headers, Microversion: &version})
	raw[1] = 'x'
	attrs["target_project_id"] = json.RawMessage(`"changed"`)
	headers["X-Trace"] = "changed"
	version = "changed"
	var calls []nativeRBACCall
	var escaped *rbacpolicies.RecordCreateOpts
	api, _ := nativeRBACAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.Header.Get("X-Trace") != "owned" || req.Header.Get("OpenStack-API-Version") != "network 2.0" {
			t.Fatal(req.Header)
		}
		escaped.Attrs["target_project_id"][1] = 'z'
		escaped.Headers["X-Trace"] = "late"
		return nativeRBACWire(204, "")
	})
	value, err := api.CreateRecord(context.Background(), option, func(c *rbacpolicies.RecordCreateOpts) error { escaped = c; return nil })
	if err != nil || len(calls) != 1 {
		t.Fatal(value, err, calls)
	}
	pythonRBACMutationBody(t, calls[0].body, map[string]any{"target_tenant": "owned"})
	pythonRBACMutationField(t, value, "target_project_id", `"owned"`)
	cause := errors.New("marshal cause")
	for _, op := range []string{"create", "update"} {
		before := len(calls)
		var err error
		if op == "create" {
			_, err = api.CreateRecord(context.Background(), rbacpolicies.WithRecordCreateField("x_ext", pythonRBACMutationMarshalFailure{cause}))
		} else {
			_, err = api.UpdateRecord(context.Background(), "target", rbacpolicies.WithRecordUpdateField("x_ext", pythonRBACMutationMarshalFailure{cause}))
		}
		var marshal *json.MarshalerError
		if !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &marshal) || len(calls) != before {
			t.Fatal(op, err, calls)
		}
	}
}

func TestPythonRBACMutationPreflightAndCallbackSourceGuard(t *testing.T) {
	var calls []nativeRBACCall
	api, _ := nativeRBACAPI(t, &calls, func(*http.Request) *http.Response { return nativeRBACWire(204, "") })
	for _, key := range []string{"", "bad\n", string([]byte{255}), "base_path", "cls", "connection", "_synchronized", "self", "__conflicting_attrs", "location", "requires_id", "create_opts", "resource_key"} {
		_, err := api.CreateRecord(context.Background(), rbacpolicies.WithRecordCreateField(key, nil))
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(key, err, calls)
		}
		_, err = api.UpdateRecord(context.Background(), "target", rbacpolicies.WithRecordUpdateField(key, nil))
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(key, err, calls)
		}
	}
	for _, key := range []string{"id", "rbac_policy"} {
		_, err := api.UpdateRecord(context.Background(), "target", rbacpolicies.WithRecordUpdateField(key, nil))
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(key, err, calls)
		}
	}
	_, err := api.CreateRecord(context.Background(), rbacpolicies.WithRecordCreateOpts(rbacpolicies.RecordCreateOpts{Attrs: map[string]json.RawMessage{"x_ext": json.RawMessage(`[`)}}))
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	_, err = api.UpdateRecord(context.Background(), "target", rbacpolicies.WithRecordUpdateHeaders(map[string]string{"Authorization": "forged"}))
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	_, err = api.CreateRecord(context.Background(), nil)
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	_, err = api.UpdateRecord(context.Background(), "target", nil)
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	_, err = api.DeleteRecord(context.Background(), "target", nil)
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	cause := errors.New("callback cause")
	source := api.RawClient()
	original := source.ResourceBase
	_, err = api.CreateRecord(context.Background(), func(c *rbacpolicies.RecordCreateOpts) error {
		c.Attrs["target_project_id"] = json.RawMessage(`"project"`)
		source.ResourceBase = original + "changed/"
		return cause
	})
	if !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonRBACMutationResetOptionsAndDeterministicBulkAliases(t *testing.T) {
	var calls []nativeRBACCall
	api, _ := nativeRBACAPI(t, &calls, func(*http.Request) *http.Response { return nativeRBACWire(204, "") })
	value, err := api.CreateRecord(context.Background(), func(c *rbacpolicies.RecordCreateOpts) error { c.Attrs = nil; c.Headers = nil; return nil }, func(c *rbacpolicies.RecordCreateOpts) error {
		c.Attrs["target_project_id"] = json.RawMessage(`"canonical"`)
		c.Attrs["target_tenant"] = json.RawMessage(`"wire"`)
		return nil
	})
	if err != nil || len(calls) != 1 {
		t.Fatal(value, err, calls)
	}
	pythonRBACMutationBody(t, calls[0].body, map[string]any{"target_tenant": "wire"})
	pythonRBACMutationField(t, value, "target_project_id", `"wire"`)
	calls = nil
	value, err = api.UpdateRecord(context.Background(), "target", func(c *rbacpolicies.RecordUpdateOpts) error { c.Attrs = nil; c.Headers = nil; return nil }, func(c *rbacpolicies.RecordUpdateOpts) error {
		c.Attrs["target_project_id"] = json.RawMessage(`false`)
		return nil
	})
	if err != nil || len(calls) != 1 {
		t.Fatal(value, err, calls)
	}
	pythonRBACMutationBody(t, calls[0].body, map[string]any{"target_tenant": false})
}

func TestPythonRBACMutationAcceptedSparseOverlayAndRawNull(t *testing.T) {
	for _, status := range []int{200, 203, 299, 302, 399} {
		var calls []nativeRBACCall
		body := `{"rbac_policy":{"target_tenant":false,"tenant_id":"received-owner","self":"raw","x_float":1e-400,"x_integer":9007199254740993,"x_nested":{"on":null}}}`
		api, _ := nativeRBACAPI(t, &calls, func(*http.Request) *http.Response { return nativeRBACWire(status, body) })
		value, err := api.CreateRecord(context.Background(), rbacpolicies.WithRecordCreateOpts(rbacpolicies.RecordCreateOpts{Attrs: map[string]json.RawMessage{"action": nil, "project_id": json.RawMessage(`"seed-owner"`)}}))
		if err != nil || value == nil || len(calls) != 1 || value.StatusCode != status || string(value.Envelope) != body {
			t.Fatal(status, value, err, calls)
		}
		pythonRBACMutationBody(t, calls[0].body, map[string]any{"action": nil, "project_id": "seed-owner"})
		pythonRBACMutationField(t, value, "action", `null`)
		pythonRBACMutationField(t, value, "target_project_id", `false`)
		pythonRBACMutationField(t, value, "project_id", `"seed-owner"`)
		pythonRBACMutationField(t, value, "tenant_id", `"received-owner"`)
		pythonRBACMutationField(t, value, "x_integer", `9007199254740993`)
		pythonRBACMutationField(t, value, "x_float", `1e-400`)
		if value.Resource.Body["self"] != nil || string(value.Wire.Body["self"]) != `"raw"` {
			t.Fatal(value)
		}
	}
}

func TestPythonRBACMutationVersionDefaultConstructorAndEmptyOverride(t *testing.T) {
	for _, op := range []string{"create", "update", "delete"} {
		for _, override := range []bool{false, true} {
			var calls []nativeRBACCall
			api, _ := nativeRBACAPI(t, &calls, func(req *http.Request) *http.Response {
				want := "network 2.0"
				if override {
					want = ""
				}
				if req.Header.Get("OpenStack-API-Version") != want || len(req.Header.Values("OpenStack-API-Version")) > 1 {
					t.Fatal(req.Header)
				}
				return nativeRBACWire(204, "")
			})
			source := api.RawClient()
			source.Microversion = "2.0"
			source.MoreHeaders = map[string]string{"Openstack-Api-Version": "network 2.0"}
			var err error
			switch op {
			case "create":
				options := []rbacpolicies.RecordCreateOption{rbacpolicies.WithRecordCreateField("microversion", "9.9")}
				if override {
					options = append(options, rbacpolicies.WithRecordCreateMicroversion(""))
				}
				_, err = api.CreateRecord(context.Background(), options...)
			case "update":
				options := []rbacpolicies.RecordUpdateOption{rbacpolicies.WithRecordUpdateField("microversion", nil), rbacpolicies.WithRecordUpdateField("target_project_id", "project")}
				if override {
					options = append(options, rbacpolicies.WithRecordUpdateMicroversion(""))
				}
				_, err = api.UpdateRecord(context.Background(), "target", options...)
			default:
				options := []rbacpolicies.RecordDeleteOption{}
				if override {
					options = append(options, rbacpolicies.WithRecordDeleteMicroversion(""))
				}
				_, err = api.DeleteRecord(context.Background(), "target", options...)
			}
			if err != nil || len(calls) != 1 || source.Microversion != "2.0" || source.MoreHeaders["Openstack-Api-Version"] != "network 2.0" {
				t.Fatal(op, override, err, calls, source)
			}
			if op == "create" {
				pythonRBACMutationBody(t, calls[0].body, map[string]any{})
			}
			if op == "update" {
				pythonRBACMutationBody(t, calls[0].body, map[string]any{"target_tenant": "project"})
			}
		}
	}
}

func TestPythonRBACMutationDeleteCallbackAndExpandedStatusRemainTerminal(t *testing.T) {
	cause := errors.New("retry callback failure")
	var calls []nativeRBACCall
	api, cloud := nativeRBACAPI(t, &calls, func(*http.Request) *http.Response {
		r := nativeRBACWire(404, `{"error":"actual missing"}`)
		r.Header.Set("X-Request-Id", "callback-proof")
		return r
	})
	cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return cause }
	value, err := api.DeleteRecord(context.Background(), "target")
	var native gophercloud.ErrUnexpectedResponseCode
	if value != nil || !errors.Is(err, cause) || !errors.As(err, &native) || native.Actual != 404 || native.ResponseHeader.Get("X-Request-Id") != "callback-proof" || len(calls) != 1 {
		t.Fatal(value, err, native, calls)
	}
	calls = nil
	api, cloud = nativeRBACAPI(t, &calls, func(*http.Request) *http.Response {
		status := 503
		if len(calls) > 1 {
			status = 404
		}
		r := nativeRBACWire(status, `{"error":"expanded status"}`)
		r.Header.Set("X-Request-Id", "expanded-proof")
		return r
	})
	cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, original error, count uint) error {
		if count > 1 {
			return original
		}
		o.OkCodes = append(o.OkCodes, 404)
		return nil
	}
	value, err = api.DeleteRecord(context.Background(), "target")
	if value != nil || err == nil || !errors.As(err, &native) || native.Actual != 404 || native.ResponseHeader.Get("X-Request-Id") != "expanded-proof" || len(calls) != 2 {
		t.Fatal(value, err, native, calls)
	}
}
