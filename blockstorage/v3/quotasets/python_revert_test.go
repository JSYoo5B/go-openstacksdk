package quotasets_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonCinderRevertUsesSourceURLencodeWithoutDoseq(t *testing.T) {
	api, _, calls := quotaClassFixture(t, func(r *http.Request) *http.Response {
		want := url.Values{"none": {"None"}, "flag": {"False"}, "list": {"['a', 1, True, None]"}, "mapping": {"{'a': 1, 'b': False}"}, "empty": {"[]"}, "big": {"9007199254740993"}}
		if r.Method != http.MethodDelete || r.URL.EscapedPath() != "/cinder/v3/os-quota-sets/project" || r.Body != nil || !reflect.DeepEqual(r.URL.Query(), want) {
			t.Fatal(r.Method, r.URL, r.Body)
		}
		return quotaClassWire(203, "opaque reset acknowledgement")
	})
	scope, err := api.InProject(context.Background(), resource.ID("project"))
	if err != nil {
		t.Fatal(err)
	}
	row, err := scope.RevertRecord(context.Background(), quotasets.WithQuotaRevertQuery("none", nil), quotasets.WithQuotaRevertQuery("flag", false), quotasets.WithQuotaRevertQuery("list", []any{"a", 1, true, nil}), quotasets.WithQuotaRevertQuery("mapping", json.RawMessage(`{"a":1,"b":false}`)), quotasets.WithQuotaRevertQuery("empty", []string{}), quotasets.WithQuotaRevertQuery("big", json.RawMessage(`9007199254740993`)))
	if err != nil || row == nil || row.ProjectID != "project" || row.StatusCode != 203 || string(row.Envelope) != "opaque reset acknowledgement" || row.Header.Get("X-Request-Id") != "quota-class-actual" || *calls != 1 {
		t.Fatal(row, err, *calls)
	}
}

func TestPythonCinderRevertRawControlNamesStayDeletionParams(t *testing.T) {
	api, client, calls := quotaClassFixture(t, func(r *http.Request) *http.Response {
		if r.URL.Path != "/cinder/v3/os-quota-sets/project" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.42" || r.Header.Get("X-Query") != "" || r.URL.Query().Get("microversion") != "9.99" || r.URL.Query().Get("base_path") != "https://foreign.invalid/ignored" || r.URL.Query().Get("headers") != "{'X-Query': 'literal'}" || r.URL.Query().Get("project_id") != "other" || r.URL.Query().Get("error_message") != "literal" {
			t.Fatal(r.URL, r.Header)
		}
		return quotaClassWire(200, "")
	})
	client.Microversion = "3.42"
	scope, err := api.InProject(context.Background(), resource.ID("project"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = scope.RevertRecord(context.Background(), quotasets.WithQuotaRevertQuery("microversion", "9.99"), quotasets.WithQuotaRevertQuery("base_path", "https://foreign.invalid/ignored"), quotasets.WithQuotaRevertQuery("headers", map[string]string{"X-Query": "literal"}), quotasets.WithQuotaRevertQuery("project_id", "other"), quotasets.WithQuotaRevertQuery("error_message", "literal"))
	if err != nil || *calls != 1 || client.Microversion != "3.42" {
		t.Fatal(err, *calls, client)
	}
}

func TestPythonCinderRevertOwnsOptionsOrderAndTransportHeaders(t *testing.T) {
	query := map[string]json.RawMessage{"value": json.RawMessage(`"owned"`)}
	headers := map[string]string{"X-Owned": "original"}
	version := "3.50"
	option := quotasets.WithQuotaRevertOpts(quotasets.QuotaRevertOpts{Query: query, Headers: headers, Microversion: &version})
	query["value"][1] = 'X'
	headers["X-Owned"] = "changed"
	version = "bad\n"
	api, client, calls := quotaClassFixture(t, func(r *http.Request) *http.Response {
		if r.URL.RawQuery != "value=owned&second=last&first=1" || r.Header.Get("X-Owned") != "original" || r.Header.Get("OpenStack-API-Version") != "volume 3.50" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.50" {
			t.Fatal(r.URL, r.Header)
		}
		return quotaClassWire(200, `not JSON`)
	})
	scope, err := api.InProject(context.Background(), resource.ID("project"))
	if err != nil {
		t.Fatal(err)
	}
	// Bulk map keys acquire a deterministic initial order before subsequent
	// single-query options append keys; replacement retains its first position.
	for range 2 {
		row, err := scope.RevertRecord(context.Background(), option, quotasets.WithQuotaRevertQuery("second", "old"), quotasets.WithQuotaRevertQuery("first", 1), quotasets.WithQuotaRevertQuery("second", "last"))
		if err != nil || row == nil || row.Microversion != "3.50" {
			t.Fatal(row, err)
		}
	}
	if *calls != 2 || client.Microversion != "" {
		t.Fatal(*calls, client)
	}
}

func TestPythonCinderRevertAcceptsBelow400AndKeeps404Strict(t *testing.T) {
	for _, code := range []int{199, 200, 202, 204, 302, 399, 400, 404, 503} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			const raw = "not JSON reset evidence"
			api, _, calls := quotaClassFixture(t, func(*http.Request) *http.Response { return quotaClassWire(code, raw) })
			scope, err := api.InProject(context.Background(), resource.ID("project"))
			if err != nil {
				t.Fatal(err)
			}
			row, err := scope.RevertRecord(context.Background())
			if *calls != 1 {
				t.Fatal(*calls)
			}
			if code < 400 {
				if err != nil || row == nil || row.StatusCode != code || string(row.Envelope) != raw {
					t.Fatal(row, err)
				}
				return
			}
			var native gophercloud.ErrUnexpectedResponseCode
			var operation *resource.OperationError
			if row != nil || !errors.As(err, &native) || native.Actual != code || native.Method != http.MethodDelete || string(native.Body) != raw || native.ResponseHeader.Get("X-Request-Id") != "quota-class-actual" || !errors.As(err, &operation) || operation.Operation != "RevertRecord" {
				t.Fatal(row, err, native, operation)
			}
			if code == 404 && !errors.Is(err, resource.ErrNotFound) {
				t.Fatal(err)
			}
		})
	}
}

func TestPythonCinderRevertPreflightGuardsAndCancellation(t *testing.T) {
	api, client, calls := quotaClassFixture(t, func(*http.Request) *http.Response { t.Fatal("preflight sent HTTP"); return nil })
	scope, err := api.InProject(context.Background(), resource.ID("project"))
	if err != nil {
		t.Fatal(err)
	}
	for _, opts := range [][]quotasets.QuotaRevertOption{{nil}, {quotasets.WithQuotaRevertQuery("bad", make(chan int))}, {quotasets.WithQuotaRevertOpts(quotasets.QuotaRevertOpts{Query: map[string]json.RawMessage{"bad": json.RawMessage(`{broken`)}})}, {quotasets.WithQuotaRevertHeaders(map[string]string{"X-Auth-Token": "foreign"})}} {
		if row, err := scope.RevertRecord(context.Background(), opts...); row != nil || err == nil || *calls != 0 {
			t.Fatal(row, err, *calls)
		}
	}
	original := client.ResourceBase
	later := 0
	row, err := scope.RevertRecord(context.Background(), func(*quotasets.QuotaRevertOpts) error { client.ResourceBase += "changed/"; return nil }, func(*quotasets.QuotaRevertOpts) error { later++; client.ResourceBase = original; return nil })
	client.ResourceBase = original
	if row != nil || !errors.Is(err, resource.ErrInvalidOption) || later != 0 || *calls != 0 {
		t.Fatal(row, err, later, *calls)
	}
	cause := errors.New("stop revert")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if row, err := scope.RevertRecord(ctx); row != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal(row, err)
	}
	var nilScope *quotasets.ProjectQuotaScope
	if row, err := nilScope.RevertRecord(context.Background()); row != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(row, err)
	}
}

func TestPythonCinderRevertPhysicalErrorsNeverReplayOrIgnore404(t *testing.T) {
	for _, code := range []int{200, 404} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			const raw = "physical reset evidence"
			readErr, closeErr := errors.New("revert read"), errors.New("revert close")
			physical := &quotaClassFaultBody{Reader: strings.NewReader(raw), readCause: readErr, closeCause: closeErr}
			api, client, calls := quotaClassFixture(t, func(*http.Request) *http.Response {
				response := quotaClassWire(code, raw)
				response.Body = physical
				return response
			})
			scope, err := api.InProject(context.Background(), resource.ID("project"))
			if err != nil {
				t.Fatal(err)
			}
			retries, reauths := 0, 0
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			client.ReauthFunc = func(context.Context) error { reauths++; return nil }
			row, err := scope.RevertRecord(context.Background())
			if !errors.Is(err, readErr) || !errors.Is(err, closeErr) || *calls != 1 || physical.closed != 1 || retries != 0 || reauths != 0 {
				t.Fatal(row, err, *calls, physical.closed, retries, reauths)
			}
			if code < 400 {
				var proof *resource.ResponseError
				if row == nil || !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != raw || string(row.Envelope) != raw {
					t.Fatal(row, err, proof)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if row != nil || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != raw {
					t.Fatal(row, err, native)
				}
			}
		})
	}
}

func TestPythonCinderRevertKeepsNativeResetSeparate(t *testing.T) {
	api, _, calls := quotaClassFixture(t, func(*http.Request) *http.Response { return quotaClassWire(204, "") })
	scope, err := api.InProject(context.Background(), resource.ID("project"))
	if err != nil {
		t.Fatal(err)
	}
	if row, err := scope.Reset(context.Background()); row != nil || err == nil {
		t.Fatal(row, err)
	}
	if row, err := scope.RevertRecord(context.Background()); row == nil || err != nil {
		t.Fatal(row, err)
	}
	if *calls != 2 {
		t.Fatal(*calls)
	}
}

func TestPythonCinderRevertCleanRetryKeepsQueryAndRejectsBodyReplacement(t *testing.T) {
	for _, tamper := range []bool{false, true} {
		t.Run(fmt.Sprint(tamper), func(t *testing.T) {
			attempts := 0
			api, client, calls := quotaClassFixture(t, func(r *http.Request) *http.Response {
				attempts++
				if r.Method != http.MethodDelete || r.URL.RawQuery != "flag=False" || r.Body != nil {
					t.Fatal(r.Method, r.URL, r.Body)
				}
				if attempts == 1 {
					return quotaClassWire(503, "first failure")
				}
				return quotaClassWire(200, "reset accepted")
			})
			scope, err := api.InProject(context.Background(), resource.ID("project"))
			if err != nil {
				t.Fatal(err)
			}
			retries := 0
			client.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				retries++
				if tamper {
					opts.JSONBody = map[string]any{"forbidden": "body"}
				}
				return nil
			}
			row, err := scope.RevertRecord(context.Background(), quotasets.WithQuotaRevertQuery("flag", false))
			if tamper {
				var native gophercloud.ErrUnexpectedResponseCode
				if row != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || *calls != 1 || retries != 1 {
					t.Fatal(row, err, *calls, retries)
				}
			} else if err != nil || row == nil || row.StatusCode != 200 || *calls != 2 || retries != 1 {
				t.Fatal(row, err, *calls, retries)
			}
		})
	}
}
