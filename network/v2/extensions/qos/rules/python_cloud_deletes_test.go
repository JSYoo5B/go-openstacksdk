package rules_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type qosDeleteTestKind struct {
	name, path string
	delete     func(context.Context, *rules.API, string, string, ...rules.QoSRuleDeleteOption) (*rules.QoSRuleDeleteReceipt, error)
}

func qosDeleteTestKinds() []qosDeleteTestKind {
	return []qosDeleteTestKind{
		{"DeleteBandwidthLimitRecord", "bandwidth_limit_rules", func(ctx context.Context, a *rules.API, p, id string, o ...rules.QoSRuleDeleteOption) (*rules.QoSRuleDeleteReceipt, error) {
			return a.DeleteBandwidthLimitRecord(ctx, p, id, o...)
		}},
		{"DeleteDSCPMarkingRecord", "dscp_marking_rules", func(ctx context.Context, a *rules.API, p, id string, o ...rules.QoSRuleDeleteOption) (*rules.QoSRuleDeleteReceipt, error) {
			return a.DeleteDSCPMarkingRecord(ctx, p, id, o...)
		}},
		{"DeleteMinimumBandwidthRecord", "minimum_bandwidth_rules", func(ctx context.Context, a *rules.API, p, id string, o ...rules.QoSRuleDeleteOption) (*rules.QoSRuleDeleteReceipt, error) {
			return a.DeleteMinimumBandwidthRecord(ctx, p, id, o...)
		}},
	}
}
func TestQoSRuleDeleteSourceOpaqueACKAndStatusOptIn(t *testing.T) {
	for _, kind := range qosDeleteTestKinds() {
		for _, status := range []int{199, 204, 302, 399} {
			t.Run(fmt.Sprint(kind.name, status), func(t *testing.T) {
				var calls []nativeRuleCall
				api, _ := nativeRuleAPI(t, &calls, func(r *http.Request) *http.Response {
					if r.Method != http.MethodDelete || r.URL.RawQuery != "" || r.Body != nil {
						t.Fatal(r.Method, r.URL, r.Body)
					}
					response := nativeRuleWire(status, "opaque ACK")
					response.Header.Set("X-Request-Id", "ack")
					return response
				})
				receipt, err := kind.delete(context.Background(), api, "parent", "rule", rules.WithQoSRuleDeleteIgnoreMissing(false))
				if err != nil || receipt == nil || receipt.StatusCode != status || string(receipt.Envelope) != "opaque ACK" || receipt.Header.Get("X-Request-Id") != "ack" || receipt.IgnoredMissing || len(calls) != 1 || calls[0].path != "/neutron/v2.0/qos/policies/parent/"+kind.path+"/rule" || calls[0].body != "" {
					t.Fatal(receipt, err, calls)
				}
			})
		}
	}
}
func TestQoSRuleDeleteDefaultMissingAndStrictRejectionsKeepActualReceipt(t *testing.T) {
	for _, kind := range qosDeleteTestKinds() {
		for _, status := range []int{400, 403, 404, 409, 429, 500} {
			for _, strict := range []bool{false, true} {
				var calls []nativeRuleCall
				api, _ := nativeRuleAPI(t, &calls, func(*http.Request) *http.Response {
					response := nativeRuleWire(status, `{"error":"actual"}`)
					response.Header.Set("X-Request-Id", "reject")
					return response
				})
				var options []rules.QoSRuleDeleteOption
				if strict {
					options = append(options, rules.WithQoSRuleDeleteIgnoreMissing(false))
				}
				receipt, err := kind.delete(context.Background(), api, "parent", "rule", options...)
				if receipt == nil || receipt.StatusCode != status || string(receipt.Envelope) != `{"error":"actual"}` || receipt.Header.Get("X-Request-Id") != "reject" || len(calls) != 1 {
					t.Fatal(kind.name, status, strict, receipt, err, calls)
				}
				if status == 404 && !strict {
					if err != nil || !receipt.IgnoredMissing {
						t.Fatal(receipt, err)
					}
					continue
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if receipt.IgnoredMissing || !errors.As(err, &native) || native.Actual != status || native.Method != http.MethodDelete {
					t.Fatal(kind.name, status, receipt, err, native)
				}
				nativeRuleOperation(t, err, kind.name)
			}
		}
	}
}
func TestQoSRuleDeleteSafeLiteralIdentityAndPreflightNoReceipt(t *testing.T) {
	for _, kind := range qosDeleteTestKinds() {
		var calls []nativeRuleCall
		api, _ := nativeRuleAPI(t, &calls, func(r *http.Request) *http.Response {
			if r.URL.EscapedPath() != "/neutron/v2.0/qos/policies/p%2F1/"+kind.path+"/r%2F1%20%252F" {
				t.Fatal(r.URL, r.URL.EscapedPath())
			}
			return nativeRuleWire(204, "")
		})
		receipt, err := kind.delete(context.Background(), api, "p/1", "r/1 %2F")
		if err != nil || receipt == nil || len(calls) != 1 {
			t.Fatal(receipt, err, calls)
		}
		for _, pair := range [][2]string{{"", "r"}, {".", "r"}, {"p", ""}, {"p", "."}, {"p", ".."}, {"p", "\n"}} {
			receipt, err = kind.delete(context.Background(), api, pair[0], pair[1])
			if receipt != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 {
				t.Fatal(kind.name, pair, receipt, err, calls)
			}
		}
		receipt, err = kind.delete(context.Background(), api, "p", "r", nil)
		if receipt != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 {
			t.Fatal(receipt, err, calls)
		}
	}
}
func TestQoSRuleDeleteOwnedHeadersVersionCallbacksAndGuardCauses(t *testing.T) {
	headers := map[string]string{"X-Trace": "owned"}
	option := rules.WithQoSRuleDeleteHeaders(headers)
	headers["X-Trace"] = "changed"
	var calls []nativeRuleCall
	api, _ := nativeRuleAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.Header.Get("X-Trace") != "owned" || r.Header.Get("X-Callback") != "yes" || r.Header.Get("OpenStack-API-Version") != "network 2.1" {
			t.Fatal(r.Header)
		}
		return nativeRuleWire(204, "")
	})
	receipt, err := api.DeleteBandwidthLimitRecord(context.Background(), "parent", "rule", rules.WithQoSRuleDeleteHeaders(nil), option, rules.WithQoSRuleDeleteMicroversion("2.1"), func(c *rules.QoSRuleDeleteOpts) error { c.Headers["X-Callback"] = "yes"; return nil })
	if err != nil || receipt == nil || len(calls) != 1 {
		t.Fatal(receipt, err, calls)
	}
	for _, cancelled := range []bool{false, true} {
		var calls []nativeRuleCall
		api, _ := nativeRuleAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("guard escaped"); return nil })
		ctx, cancel := context.WithCancelCause(context.Background())
		cause, optionCause := errors.New("cancelled"), errors.New("option error")
		receipt, err := api.DeleteBandwidthLimitRecord(ctx, "parent", "rule", func(*rules.QoSRuleDeleteOpts) error {
			if cancelled {
				cancel(cause)
			} else {
				api.RawClient().ResourceBase += "changed/"
			}
			return optionCause
		})
		cancel(nil)
		if receipt != nil || !errors.Is(err, optionCause) || len(calls) != 0 || cancelled && !errors.Is(err, cause) || !cancelled && !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(receipt, err, calls)
		}
	}
}
func TestQoSRuleDeletePhysicalFaultsCannotBecomeMissing(t *testing.T) {
	for _, kind := range qosDeleteTestKinds() {
		for _, status := range []int{204, 404, 401, 500} {
			for _, fault := range []string{"read", "close", "mixed"} {
				t.Run(fmt.Sprint(kind.name, status, fault), func(t *testing.T) {
					readCause, closeCause := errors.New("delete read"), errors.New("delete close")
					body := &pythonBandwidthFaultBody{data: `{"error":"actual"}`}
					if fault != "close" {
						body.read = readCause
					}
					if fault != "read" {
						body.close = closeCause
					}
					var calls []nativeRuleCall
					api, _ := nativeRuleAPI(t, &calls, func(*http.Request) *http.Response {
						r := nativeRuleWire(status, body.data)
						r.Body = body
						r.Header.Set("X-Request-Id", "fault")
						return r
					})
					retry, reauth := 0, 0
					api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
						retry++
						return nil
					}
					api.RawClient().ReauthFunc = func(context.Context) error { reauth++; return nil }
					receipt, err := kind.delete(context.Background(), api, "parent", "rule")
					if err == nil || resource.IgnoreMissing(err) == nil || receipt == nil || receipt.StatusCode != status || receipt.IgnoredMissing || body.closes != 1 || len(calls) != 1 || retry != 0 || reauth != 0 {
						t.Fatal(receipt, err, body.closes, calls, retry, reauth)
					}
					if fault != "close" && !errors.Is(err, readCause) || fault != "read" && !errors.Is(err, closeCause) {
						t.Fatal(err)
					}
					if status == 401 {
						var auth *gophercloud.ErrUnableToReauthenticate
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(err, &auth) || !errors.As(auth.ErrOriginal, &native) || native.Actual != 401 || auth.ErrReauth == nil {
							t.Fatal(err, auth, native)
						}
					}
				})
			}
		}
	}
}
func TestQoSRuleDeleteAccepted404AndCallback404NeverEstablishMissing(t *testing.T) {
	for _, accepted := range []bool{false, true} {
		var calls []nativeRuleCall
		status := 503
		if accepted {
			status = 404
		}
		api, _ := nativeRuleAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(status, `{"error":"real"}`) })
		callbackCause := errors.New("callback404 cause")
		api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, c *gophercloud.RequestOpts, _ error, _ uint) error {
			if accepted {
				c.OkCodes = append(c.OkCodes, 404)
				return nil
			}
			return errors.Join(gophercloud.ErrUnexpectedResponseCode{Actual: 404}, callbackCause)
		}
		receipt, err := api.DeleteBandwidthLimitRecord(context.Background(), "parent", "rule")
		if err == nil || resource.IgnoreMissing(err) == nil || receipt == nil || receipt.StatusCode != status || receipt.IgnoredMissing || accepted && len(calls) != 2 || !accepted && (!errors.Is(err, callbackCause) || len(calls) != 1) {
			t.Fatal(accepted, receipt, err, calls)
		}
	}
}
func TestQoSRuleDeleteCleanRetryAndAuthenticationKeepFixedBodylessPolicy(t *testing.T) {
	for _, status := range []int{401, 503} {
		var calls []nativeRuleCall
		var tokens []string
		api, _ := nativeRuleAPI(t, &calls, func(r *http.Request) *http.Response {
			tokens = append(tokens, r.Header.Get("X-Auth-Token"))
			if r.Method != http.MethodDelete || r.Body != nil || r.Header.Get("If-Match") != "selected" || r.Header.Get("OpenStack-API-Version") != "network 2.1" {
				t.Fatal(r.Method, r.Body, r.Header)
			}
			if len(calls) == 1 {
				return nativeRuleWire(status, `{"error":"retry"}`)
			}
			return nativeRuleWire(204, "")
		})
		reauth, retry := 0, 0
		api.RawClient().ReauthFunc = func(context.Context) error { reauth++; api.RawClient().SetToken("renewed"); return nil }
		api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			retry++
			return nil
		}
		receipt, err := api.DeleteBandwidthLimitRecord(context.Background(), "parent", "rule", rules.WithQoSRuleDeleteIgnoreMissing(false), rules.WithQoSRuleDeleteHeaders(map[string]string{"If-Match": "selected"}), rules.WithQoSRuleDeleteMicroversion("2.1"))
		if err != nil || receipt == nil || receipt.StatusCode != 204 || len(calls) != 2 || status == 401 && (reauth != 1 || retry != 0 || !reflect.DeepEqual(tokens, []string{"test-token", "renewed"})) || status == 503 && (retry != 1 || reauth != 0) {
			t.Fatal(receipt, err, calls, tokens, reauth, retry)
		}
	}
}
