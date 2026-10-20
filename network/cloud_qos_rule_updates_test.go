package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type cloudQoSUpdateKind struct {
	kind, key, field string
	update           func(context.Context, *network.Service, string, string, ...network.QoSRuleUpdateOption) (*network.QoSRuleUpdateResult, error)
}

func cloudQoSUpdateKinds() []cloudQoSUpdateKind {
	return []cloudQoSUpdateKind{
		{"bandwidth_limit_rules", "bandwidth_limit_rule", "max_kbps", func(c context.Context, s *network.Service, p, id string, o ...network.QoSRuleUpdateOption) (*network.QoSRuleUpdateResult, error) {
			return s.UpdateQoSBandwidthLimitRuleRecord(c, p, id, o...)
		}},
		{"dscp_marking_rules", "dscp_marking_rule", "dscp_mark", func(c context.Context, s *network.Service, p, id string, o ...network.QoSRuleUpdateOption) (*network.QoSRuleUpdateResult, error) {
			return s.UpdateQoSDSCPMarkingRuleRecord(c, p, id, o...)
		}},
		{"minimum_bandwidth_rules", "minimum_bandwidth_rule", "min_kbps", func(c context.Context, s *network.Service, p, id string, o ...network.QoSRuleUpdateOption) (*network.QoSRuleUpdateResult, error) {
			return s.UpdateQoSMinimumBandwidthRuleRecord(c, p, id, o...)
		}},
	}
}
func TestCloudQoSRuleUpdateValidKwargsBeforeHTTPAndOwnedOptions(t *testing.T) {
	for _, kind := range cloudQoSUpdateKinds() {
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if qosCloudCreateGate(w, r, false) {
				return
			}
			if r.URL.Path == "/neutron/v2.0/qos/policies/actual" {
				testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
				return
			}
			if r.Method == http.MethodGet {
				testcloud.JSON(w, 200, `{"`+kind.key+`":{"id":"actual","`+kind.field+`":1}}`)
				return
			}
			body := qosCloudCreateBody(t, r, kind.key)
			if string(body[kind.field]) != "3" || len(body) != 1 {
				t.Error(body)
			}
			testcloud.JSON(w, 200, `{"`+kind.key+`":{"id":"actual"}}`)
		})
		for _, options := range [][]network.QoSRuleUpdateOption{{nil}, {network.WithQoSRuleUpdateJSON(json.RawMessage(`[]`))}, {network.WithQoSRuleUpdateJSON(json.RawMessage(`{bad`))}, {network.WithQoSRuleUpdateAttribute("bad", nil)}, {network.WithQoSRuleUpdateAttribute("rule_id", "other")}, {network.WithQoSRuleUpdateAttribute("policy_name_or_id", "other")}} {
			result, err := kind.update(context.Background(), service, "actual", "requested", options...)
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != 0 {
				t.Fatal(result, err, count.Load())
			}
		}
		raw := json.RawMessage(`{"` + kind.field + `":2}`)
		option := network.WithQoSRuleUpdateJSON(raw)
		raw[2] = 'X'
		result, err := kind.update(context.Background(), service, "actual", "requested", option, network.WithQoSRuleUpdateAttribute(kind.field, 3))
		if err != nil || result == nil || result.NoOp || result.Rule == nil || count.Load() != 4 {
			t.Fatal(result, err, count.Load())
		}
	}
}
func TestCloudQoSRuleUpdateEmptyAfterPolicyAndDirectionGateSkipsChild(t *testing.T) {
	for _, kind := range cloudQoSUpdateKinds() {
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if qosCloudCreateGate(w, r, false) {
				return
			}
			if r.URL.Path != "/neutron/v2.0/qos/policies/actual" || r.Method != http.MethodGet {
				t.Fatal("empty child request", r.Method, r.URL)
			}
			testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
		})
		result, err := kind.update(context.Background(), service, "actual", "")
		if err != nil || result == nil || !result.NoOp || result.Policy == nil || result.Current != nil || result.Rule != nil || count.Load() != 2 {
			t.Fatal(result, err, count.Load())
		}
		missing, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if qosCloudCreateGate(w, r, false) {
				return
			}
			if r.URL.Path == "/neutron/v2.0/qos/policies" {
				testcloud.JSON(w, 200, `{"policies":[]}`)
			} else {
				testcloud.JSON(w, 404, `{}`)
			}
		})
		result, err = kind.update(context.Background(), missing, "missing", "")
		if result == nil || result.NoOp || !errors.Is(err, resource.ErrNotFound) || count.Load() != 3 {
			t.Fatal(result, err, count.Load())
		}
	}
	for _, raw := range []string{`{"direction":null}`, `{"direction":false}`, `{"direction":0}`, `{"direction":""}`, `{"direction":[]}`, `{"direction":{}}`} {
		for _, available := range []bool{false, true} {
			var order []string
			service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if qosCloudCreateGate(w, r, available) {
					order = append(order, "gate")
					return
				}
				if r.URL.Path == "/neutron/v2.0/qos/policies/actual" {
					order = append(order, "policy")
					testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
					return
				}
				if r.Method == http.MethodGet {
					order = append(order, "child")
					testcloud.JSON(w, 200, `{"bandwidth_limit_rule":{"id":"actual","direction":"egress"}}`)
					return
				}
				order = append(order, "put")
				testcloud.JSON(w, 200, `{"bandwidth_limit_rule":{"id":"actual"}}`)
			})
			nonnull := raw != `{"direction":null}`
			drop := nonnull && !available
			result, err := service.UpdateQoSBandwidthLimitRuleRecord(context.Background(), "actual", "requested", network.WithQoSRuleUpdateJSON(json.RawMessage(raw)))
			want := []string{"gate", "policy"}
			if nonnull {
				want = append(want, "gate")
			}
			if !drop {
				want = append(want, "child", "put")
			}
			if err != nil || result == nil || result.DirectionDropped != drop || result.NoOp != drop || !reflect.DeepEqual(order, want) || count.Load() != int32(len(want)) {
				t.Fatal(raw, available, result, err, order, count.Load())
			}
			if drop && result.Current != nil {
				t.Fatal(result)
			}
		}
	}
}
func TestCloudQoSRuleUpdateRawDeltaEqualAndPartialResponse(t *testing.T) {
	for _, kind := range cloudQoSUpdateKinds() {
		for _, tc := range []struct {
			before, after string
			put           bool
		}{{"missing", "null", true}, {"true", "1", false}, {`"false"`, "true", true}, {"9007199254740992.0", "9007199254740993.0", false}, {"1e-4000", "0.0", false}, {"1", "2", true}} {
			var payload map[string]json.RawMessage
			service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if qosCloudCreateGate(w, r, false) {
					return
				}
				if r.URL.Path == "/neutron/v2.0/qos/policies/actual" {
					testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
					return
				}
				if r.Method == http.MethodGet {
					field := ""
					if tc.before != "missing" {
						field = `,"` + kind.field + `":` + tc.before
					}
					testcloud.JSON(w, 203, `{"`+kind.key+`":{"id":"actual","microversion":"foreign","qos_policy_id":"foreign"`+field+`}}`)
					return
				}
				if r.URL.Path != "/neutron/v2.0/qos/policies/actual/"+kind.kind+"/actual" || r.Method != http.MethodPut {
					t.Error(r.Method, r.URL)
				}
				payload = qosCloudCreateBody(t, r, kind.key)
				testcloud.JSON(w, 200, `{"`+kind.key+`":{"id":"actual","x_new":9007199254740993,"microversion":"foreign","qos_policy_id":"foreign"}}`)
			})
			service.RawClient().Microversion = "2.1"
			result, err := kind.update(context.Background(), service, "actual", "requested", network.WithQoSRuleUpdateJSON(json.RawMessage(`{"`+kind.field+`":`+tc.after+`}`)))
			calls := int32(3)
			want := tc.before
			version := "null"
			if tc.put {
				calls = 4
				want = tc.after
				version = `"2.1"`
			}
			if err != nil || result == nil || result.Current == nil || result.Rule == nil || result.NoOp == tc.put || count.Load() != calls || string(result.Rule.Resource.Body[kind.field]) != want || string(result.Rule.Resource.Body["microversion"]) != version || string(result.Rule.Resource.Body["qos_policy_id"]) != `"actual"` {
				t.Fatal(tc, result, err, count.Load())
			}
			if tc.put {
				if len(payload) != 1 || string(payload[kind.field]) != tc.after || result.Rule.Wire.Body[kind.field] != nil {
					t.Fatal(payload, result)
				}
			} else if result.Rule.StatusCode != 203 || !reflect.DeepEqual(result.Rule.Envelope, result.Current.Envelope) {
				t.Fatal(result)
			}
		}
	}
}
func TestCloudQoSRuleUpdateStrictChildGETAndFailedPUTKeepIndependentReceipts(t *testing.T) {
	for _, kind := range cloudQoSUpdateKinds() {
		for _, method := range []string{http.MethodGet, http.MethodPut} {
			for _, status := range []int{400, 403, 404, 409, 503} {
				body := `{"error":"actual rule stage"}`
				service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
					if qosCloudCreateGate(w, r, false) {
						return
					}
					if r.URL.Path == "/neutron/v2.0/qos/policies/actual" {
						testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
						return
					}
					if method == http.MethodPut && r.Method == http.MethodGet {
						testcloud.JSON(w, 203, `{"`+kind.key+`":{"id":"actual","`+kind.field+`":1}}`)
						return
					}
					testcloud.JSON(w, status, body)
				})
				result, err := kind.update(context.Background(), service, "actual", "requested", network.WithQoSRuleUpdateAttribute(kind.field, 2))
				want := int32(3)
				var receipt = result.Current
				if method == http.MethodPut {
					want = 4
					receipt = result.Rule
				}
				if err == nil || result == nil || result.NoOp || receipt == nil || receipt.Resource != nil || receipt.StatusCode != status || string(receipt.Envelope) != body || receipt.Header.Get("X-Request-Id") != "qos-mutation-actual" || count.Load() != want {
					t.Fatal(method, status, result, err, count.Load())
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Method != method || native.Actual != status || string(native.Body) != body {
					t.Fatal(err, native)
				}
				if method == http.MethodGet && result.Rule != nil {
					t.Fatal(result)
				}
			}
		}
	}
}

// Observe actual HTTP GET/PUT bodies using the existing action-suite wrapper.
type qosRuleUpdateObservedTransport struct {
	base              http.RoundTripper
	kind, method      string
	readErr, closeErr error
	closed            *atomic.Int32
	onClose           func()
	status            int
}

func (t qosRuleUpdateObservedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	v, err := t.base.RoundTrip(r)
	if v != nil && r.Method == t.method && strings.Contains(r.URL.Path, "/"+t.kind+"/") {
		if t.status != 0 {
			v.StatusCode = t.status
			v.Status = fmt.Sprint(t.status)
		}
		v.Body = &qosPolicyActionObservedBody{ReadCloser: v.Body, readErr: t.readErr, closeErr: t.closeErr, closed: t.closed, onClose: t.onClose}
	}
	return v, err
}
func TestCloudQoSRuleUpdatePhysicalGETPUTFaultsKeepNative401AndNoHooks(t *testing.T) {
	for _, kind := range cloudQoSUpdateKinds() {
		for _, method := range []string{http.MethodGet, http.MethodPut} {
			for _, status := range []int{200, 401, 404, 503} {
				for _, lane := range []string{"read-close", "cancel", "source"} {
					t.Run(fmt.Sprintf("%s/%s/%d/%s", kind.kind, method, status, lane), func(t *testing.T) {
						body := `{"` + kind.key + `":{"id":"actual","` + kind.field + `":1}}`
						service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
							if qosCloudCreateGate(w, r, false) {
								return
							}
							if r.URL.Path == "/neutron/v2.0/qos/policies/actual" {
								testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
								return
							}
							if method == http.MethodPut && r.Method == http.MethodGet {
								testcloud.JSON(w, 200, body)
								return
							}
							testcloud.JSON(w, status, body)
						})
						ctx, cancel := context.WithCancelCause(context.Background())
						defer cancel(nil)
						readErr, closeErr, cancelCause := errors.New("update read physical"), errors.New("update close physical"), errors.New("update cancel")
						base := service.RawClient().HTTPClient.Transport
						if base == nil {
							base = http.DefaultTransport
						}
						closed := &atomic.Int32{}
						observer := qosRuleUpdateObservedTransport{base: base, kind: kind.kind, method: method, closed: closed}
						switch lane {
						case "read-close":
							observer.readErr, observer.closeErr = readErr, closeErr
						case "cancel":
							observer.onClose = func() { cancel(cancelCause) }
						case "source":
							observer.onClose = func() { service.RawClient().ResourceBase += "changed/" }
						}
						service.RawClient().HTTPClient.Transport = observer
						retries, reauths := 0, 0
						service.RawClient().ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
							retries++
							return errors.New("unexpected bounded retry")
						}
						service.RawClient().ProviderClient.ReauthFunc = func(context.Context) error { reauths++; return errors.New("unexpected bounded reauth") }
						result, err := kind.update(ctx, service, "actual", "requested", network.WithQoSRuleUpdateAttribute(kind.field, 2))
						want := int32(3)
						receipt := result.Current
						if method == http.MethodPut {
							want = 4
							receipt = result.Rule
						}
						if err == nil || result == nil || result.NoOp || result.Policy == nil || receipt == nil || receipt.Resource != nil || receipt.Wire != nil || receipt.StatusCode != status || string(receipt.Envelope) != body || receipt.Header.Get("X-Request-Id") != "qos-mutation-actual" || count.Load() != want || closed.Load() != 1 || retries != 0 || reauths != 0 {
							t.Fatal(result, err, count.Load(), closed.Load(), retries, reauths)
						}
						switch lane {
						case "read-close":
							if !errors.Is(err, readErr) || !errors.Is(err, closeErr) {
								t.Fatal(err)
							}
						case "cancel":
							if !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) {
								t.Fatal(err)
							}
						case "source":
							if !errors.Is(err, resource.ErrInvalidOption) {
								t.Fatal(err)
							}
						}
						if status < 400 {
							var proof *resource.ResponseError
							if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body {
								t.Fatal(err, proof)
							}
						} else {
							proof := err
							if status == 401 {
								var auth *gophercloud.ErrUnableToReauthenticate
								if !errors.As(err, &auth) || auth == nil || auth.ErrOriginal == nil || auth.ErrReauth == nil {
									t.Fatal(err, auth)
								}
								proof = auth.ErrOriginal
								if lane == "read-close" && (!errors.Is(auth.ErrReauth, readErr) || !errors.Is(auth.ErrReauth, closeErr)) {
									t.Fatal(auth)
								}
							}
							var native gophercloud.ErrUnexpectedResponseCode
							if !errors.As(proof, &native) || native.Method != method || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "qos-mutation-actual" {
								t.Fatal(err, native)
							}
						}
					})
				}
			}
		}
	}
}
func TestCloudQoSRuleUpdateCleanRetryRenewal199AndHeaderCapture(t *testing.T) {
	for _, kind := range cloudQoSUpdateKinds() {
		for _, lane := range []string{"retry", "reauth", "source199", "route"} {
			attempts := 0
			var tokens []string
			service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Owned") != "before" {
					t.Error(r.Header)
				}
				if qosCloudCreateGate(w, r, false) {
					return
				}
				if r.URL.Path == "/neutron/v2.0/qos/policies/actual" {
					testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
					return
				}
				if r.Method == http.MethodGet {
					testcloud.JSON(w, 200, `{"`+kind.key+`":{"id":"actual","`+kind.field+`":1}}`)
					return
				}
				attempts++
				tokens = append(tokens, r.Header.Get("X-Auth-Token"))
				if attempts == 1 && lane != "source199" {
					status := 503
					if lane == "reauth" {
						status = 401
					}
					testcloud.JSON(w, status, `{"error":"original"}`)
					return
				}
				testcloud.JSON(w, 200, `{"`+kind.key+`":{"id":"actual"}}`)
			})
			service.RawClient().MoreHeaders = map[string]string{"X-Owned": "before"}
			option := network.QoSRuleUpdateOption(func(c *network.QoSRuleUpdateOpts) error {
				service.RawClient().MoreHeaders["X-Owned"] = "after"
				c.Attributes = map[string]json.RawMessage{kind.field: json.RawMessage("2")}
				return nil
			})
			callbacks := 0
			if lane == "reauth" {
				service.RawClient().ProviderClient.ReauthFunc = func(context.Context) error {
					callbacks++
					service.RawClient().ProviderClient.SetToken("renewed")
					return nil
				}
			} else if lane != "source199" {
				service.RawClient().ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					callbacks++
					if lane == "route" {
						service.RawClient().ResourceBase += "changed/"
					}
					return nil
				}
			}
			closed := &atomic.Int32{}
			if lane == "source199" {
				base := service.RawClient().HTTPClient.Transport
				if base == nil {
					base = http.DefaultTransport
				}
				service.RawClient().HTTPClient.Transport = qosRuleUpdateObservedTransport{base: base, kind: kind.kind, method: http.MethodPut, closed: closed, status: 199}
			}
			result, err := kind.update(context.Background(), service, "actual", "requested", option)
			if lane == "route" {
				if result == nil || result.Rule == nil || result.Rule.StatusCode != 503 || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != 4 || callbacks != 1 {
					t.Fatal(result, err, count.Load(), callbacks)
				}
				continue
			}
			want := int32(5)
			if lane == "source199" {
				want = 4
			}
			if err != nil || result == nil || result.Rule == nil || result.NoOp || count.Load() != want {
				t.Fatal(result, err, count.Load())
			}
			if lane == "source199" {
				if result.Rule.StatusCode != 199 || closed.Load() != 1 || callbacks != 0 {
					t.Fatal(result, closed.Load(), callbacks)
				}
			} else if callbacks != 1 {
				t.Fatal(callbacks)
			}
			if lane == "reauth" && !reflect.DeepEqual(tokens, []string{"test-token", "renewed"}) {
				t.Fatal(tokens)
			}
		}
	}
}
