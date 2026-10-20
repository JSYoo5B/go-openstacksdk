package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type cloudQoSCreateKind struct {
	kind, key, required string
	create              func(context.Context, *network.Service, string, any, ...network.QoSRuleCreateOption) (*network.QoSRuleCreateResult, error)
}

func cloudQoSCreateKinds() []cloudQoSCreateKind {
	return []cloudQoSCreateKind{
		{"bandwidth_limit_rules", "bandwidth_limit_rule", "max_kbps", func(c context.Context, s *network.Service, p string, v any, o ...network.QoSRuleCreateOption) (*network.QoSRuleCreateResult, error) {
			return s.CreateQoSBandwidthLimitRuleRecord(c, p, v, o...)
		}},
		{"dscp_marking_rules", "dscp_marking_rule", "dscp_mark", func(c context.Context, s *network.Service, p string, v any, o ...network.QoSRuleCreateOption) (*network.QoSRuleCreateResult, error) {
			return s.CreateQoSDSCPMarkingRuleRecord(c, p, v, o...)
		}},
		{"minimum_bandwidth_rules", "minimum_bandwidth_rule", "min_kbps", func(c context.Context, s *network.Service, p string, v any, o ...network.QoSRuleCreateOption) (*network.QoSRuleCreateResult, error) {
			return s.CreateQoSMinimumBandwidthRuleRecord(c, p, v, o...)
		}},
	}
}
func qosCloudCreateGate(w http.ResponseWriter, r *http.Request, direction bool) bool {
	if r.Method != http.MethodGet || r.URL.Path != "/neutron/v2.0/extensions" {
		return false
	}
	body := `{"extensions":[{"alias":"qos"},{"alias":"other"}]}`
	if direction {
		body = `{"extensions":[{"alias":"qos"},{"alias":"qos-bw-limit-direction"}]}`
	}
	testcloud.JSON(w, 200, body)
	return true
}
func qosCloudCreateBody(t *testing.T, r *http.Request, key string) map[string]json.RawMessage {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	var envelope map[string]map[string]json.RawMessage
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(string(data), err)
	}
	return envelope[key]
}
func TestCloudQoSRuleCreateValidKwargsOwnedOptionsAndRequiredRawPresence(t *testing.T) {
	for _, kind := range cloudQoSCreateKinds() {
		t.Run(kind.kind, func(t *testing.T) {
			service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if qosCloudCreateGate(w, r, false) {
					return
				}
				if r.Method == http.MethodGet {
					testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
					return
				}
				if r.Method != http.MethodPost || r.URL.Path != "/neutron/v2.0/qos/policies/actual/"+kind.kind {
					t.Error(r.Method, r.URL)
				}
				body := qosCloudCreateBody(t, r, kind.key)
				if body[kind.required] == nil || len(body) != 1 {
					t.Error(body)
				}
				testcloud.JSON(w, 201, `{"`+kind.key+`":{"id":"new"}}`)
			})
			for _, options := range [][]network.QoSRuleCreateOption{{nil}, {network.WithQoSRuleCreateJSON(json.RawMessage(`[]`))}, {network.WithQoSRuleCreateJSON(json.RawMessage(`{bad`))}, {network.WithQoSRuleCreateAttribute(kind.required, 0)}, {network.WithQoSRuleCreateAttribute("policy_name_or_id", "other")}, {network.WithQoSRuleCreateAttribute("bad", nil)}} {
				result, err := kind.create(context.Background(), service, "actual", 0, options...)
				if result != nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != 0 {
					t.Fatal(result, err, count.Load())
				}
			}
			if kind.kind == "dscp_marking_rules" {
				result, err := kind.create(context.Background(), service, "actual", 0, network.WithQoSRuleCreateDirection("egress"))
				if result != nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != 0 {
					t.Fatal(result, err, count.Load())
				}
			}
			for _, value := range []any{0, nil, false, "0", json.Number("9007199254740993")} {
				result, err := kind.create(context.Background(), service, "actual", value)
				if err != nil || result == nil || result.Rule == nil {
					t.Fatal(result, err)
				}
				raw, _ := json.Marshal(value)
				if string(result.Rule.Body[kind.required]) != string(raw) || result.Wire.Body[kind.required] != nil {
					t.Fatal(result, string(raw))
				}
			}
			before := count.Load()
			if result, err := kind.create(context.Background(), service, "actual", make(chan int)); result != nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != before {
				t.Fatal(result, err, count.Load())
			}
		})
	}
	raw := json.RawMessage(`{"max_burst_kbps":3}`)
	option := network.WithQoSRuleCreateJSON(raw)
	raw[2] = 'X'
	var posted map[string]json.RawMessage
	service, _ := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if qosCloudCreateGate(w, r, false) {
			return
		}
		if r.Method == http.MethodGet {
			testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
			return
		}
		posted = qosCloudCreateBody(t, r, "bandwidth_limit_rule")
		testcloud.JSON(w, 201, `{"bandwidth_limit_rule":{"id":"new"}}`)
	})
	result, err := service.CreateQoSBandwidthLimitRuleRecord(context.Background(), "actual", 0, option, network.WithQoSRuleCreateMaxBurstKBPS(0))
	if err != nil || result == nil || string(posted["max_burst_kbps"]) != "0" || len(posted) != 2 {
		t.Fatal(result, err, posted)
	}
}
func TestCloudQoSRuleCreateBandwidthDirectionNonnullGateAfterPolicy(t *testing.T) {
	for _, raw := range []string{`{}`, `{"direction":null}`, `{"direction":false}`, `{"direction":0}`, `{"direction":""}`, `{"direction":[]}`, `{"direction":{}}`} {
		for _, available := range []bool{false, true} {
			var order []string
			var posted map[string]json.RawMessage
			service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if qosCloudCreateGate(w, r, available) {
					order = append(order, "gate")
					return
				}
				if r.Method == http.MethodGet {
					order = append(order, "policy")
					testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
					return
				}
				order = append(order, "create")
				posted = qosCloudCreateBody(t, r, "bandwidth_limit_rule")
				testcloud.JSON(w, 201, `{"bandwidth_limit_rule":{"id":"new"}}`)
			})
			var input map[string]json.RawMessage
			json.Unmarshal([]byte(raw), &input)
			direction, present := input["direction"]
			extra := present && string(direction) != "null"
			result, err := service.CreateQoSBandwidthLimitRuleRecord(context.Background(), "actual", 0, network.WithQoSRuleCreateJSON(json.RawMessage(raw)), network.WithQoSRuleCreateMaxBurstKBPS(0))
			expected := []string{"gate", "policy", "create"}
			if extra {
				expected = []string{"gate", "policy", "gate", "create"}
			}
			if err != nil || result == nil || result.DirectionDropped != (extra && !available) || !reflect.DeepEqual(order, expected) || count.Load() != int32(len(expected)) {
				t.Fatal(raw, available, result, err, order, count.Load())
			}
			if string(posted["max_kbps"]) != "0" || string(posted["max_burst_kbps"]) != "0" {
				t.Fatal(posted)
			}
			if extra && !available {
				if posted["direction"] != nil {
					t.Fatal(posted)
				}
			} else if string(posted["direction"]) != string(direction) {
				t.Fatal(posted, direction)
			}
		}
	}
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if qosCloudCreateGate(w, r, false) {
			return
		}
		if r.Method == http.MethodGet {
			testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
			return
		}
		body := qosCloudCreateBody(t, r, "minimum_bandwidth_rule")
		if string(body["direction"]) != `"ingress"` {
			t.Error(body)
		}
		testcloud.JSON(w, 201, `{"minimum_bandwidth_rule":{"id":"new"}}`)
	})
	result, err := service.CreateQoSMinimumBandwidthRuleRecord(context.Background(), "actual", 0, network.WithQoSRuleCreateDirection("ingress"))
	if err != nil || result == nil || result.DirectionDropped || count.Load() != 3 {
		t.Fatal(result, err, count.Load())
	}
}
func TestCloudQoSRuleCreateFindMissingAmbiguityAndListFallback(t *testing.T) {
	for _, kind := range cloudQoSCreateKinds() {
		for _, rows := range []string{`[]`, `[{"id":"actual","name":"requested"}]`, `[{"id":"one","name":"requested"},{"id":"two","name":"requested"}]`} {
			service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if qosCloudCreateGate(w, r, false) {
					return
				}
				if r.URL.Path == "/neutron/v2.0/qos/policies/requested" {
					testcloud.JSON(w, 404, `{"error":"lookup absent"}`)
					return
				}
				if r.URL.Path == "/neutron/v2.0/qos/policies" {
					if r.URL.Query().Get("name") != "requested" {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, `{"policies":`+rows+`}`)
					return
				}
				if r.Method != http.MethodPost || r.URL.Path != "/neutron/v2.0/qos/policies/actual/"+kind.kind {
					t.Error(r.Method, r.URL)
				}
				testcloud.JSON(w, 201, `{"`+kind.key+`":{"id":"new"}}`)
			})
			result, err := kind.create(context.Background(), service, "requested", 0)
			if result == nil || len(result.Extensions) != 2 {
				t.Fatal(result, err)
			}
			if rows == `[]` {
				if !errors.Is(err, resource.ErrNotFound) || count.Load() != 3 || result.Rule != nil {
					t.Fatal(result, err, count.Load())
				}
			} else if strings.Contains(rows, `"two"`) {
				if !errors.Is(err, resource.ErrAmbiguous) || count.Load() != 3 || result.Rule != nil {
					t.Fatal(result, err, count.Load())
				}
			} else if err != nil || result.Rule == nil || count.Load() != 4 || string(result.Policy.Resource.Body["id"]) != `"actual"` {
				t.Fatal(result, err, count.Load())
			}
		}
	}
}
func TestCloudQoSRuleCreateCompleteGatesAndOptionalFailuresKeepStages(t *testing.T) {
	for _, kind := range cloudQoSCreateKinds() {
		for _, lane := range []string{"unavailable", "late-alias", "late-page"} {
			service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "extensions") {
					t.Fatal("policy/create after failed gate", r.URL)
				}
				switch lane {
				case "unavailable":
					testcloud.JSON(w, 200, `{"extensions":[{"alias":"other"}]}`)
				case "late-alias":
					testcloud.JSON(w, 200, `{"extensions":[{"alias":"qos"},{"alias":[]}]}`)
				case "late-page":
					if r.URL.Query().Get("marker") == "next" {
						testcloud.JSON(w, 403, `{"error":"late extension"}`)
					} else {
						testcloud.JSON(w, 200, `{"extensions":[{"alias":"qos"}],"extensions_links":[{"rel":"next","href":"?marker=next"}]}`)
					}
				}
			})
			result, err := kind.create(context.Background(), service, "requested", 0)
			if result == nil || err == nil || result.Policy != nil || result.Rule != nil || result.StatusCode != 0 {
				t.Fatal(result, err)
			}
			if lane == "unavailable" && !errors.Is(err, network.ErrUnavailableExtension) {
				t.Fatal(err)
			}
			if lane == "late-alias" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if lane == "late-page" && (!gophercloud.ResponseCodeIs(err, 403) || count.Load() != 2) {
				t.Fatal(err, count.Load())
			}
		}
	}
	gates := 0
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/neutron/v2.0/extensions" {
			gates++
			if gates == 1 {
				testcloud.JSON(w, 200, `{"extensions":[{"alias":"qos"}]}`)
			} else {
				testcloud.JSON(w, 403, `{"error":"direction gate"}`)
			}
			return
		}
		if r.Method != http.MethodGet {
			t.Fatal("create after optional gate error")
		}
		testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
	})
	result, err := service.CreateQoSBandwidthLimitRuleRecord(context.Background(), "actual", 0, network.WithQoSRuleCreateDirection("ingress"))
	if result == nil || result.Policy == nil || result.Rule != nil || result.DirectionDropped || !gophercloud.ResponseCodeIs(err, 403) || count.Load() != 3 {
		t.Fatal(result, err, count.Load())
	}
}
func TestCloudQoSRuleCreateReturnedParentJSONURIAndActualViews(t *testing.T) {
	for _, kind := range cloudQoSCreateKinds() {
		for _, tc := range []struct{ raw, path string }{{"null", "None"}, {"false", "False"}, {"0", "0"}, {"true", "True"}, {"9007199254740993", "9007199254740993"}, {`"p/1"`, "p%2F1"}} {
			body := `{"` + kind.key + `":{"id":null,"qos_policy_id":"foreign","microversion":"foreign","location":{"forged":true},"x_new":9007199254740993}}`
			service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("OpenStack-API-Version") != "network 2.1" {
					t.Error(r.Header)
				}
				if qosCloudCreateGate(w, r, false) {
					return
				}
				if r.Method == http.MethodGet {
					testcloud.JSON(w, 203, `{"policy":{"id":`+tc.raw+`}}`)
					return
				}
				if r.URL.EscapedPath() != "/neutron/v2.0/qos/policies/"+tc.path+"/"+kind.kind {
					t.Error(r.URL.EscapedPath())
				}
				testcloud.JSON(w, 201, body)
			})
			service.RawClient().Microversion = "2.1"
			result, err := kind.create(context.Background(), service, "requested", 0)
			if err != nil || result == nil || result.Rule == nil || count.Load() != 3 || string(result.Rule.Body["qos_policy_id"]) != tc.raw || string(result.Rule.Body[kind.required]) != "0" || string(result.Rule.Body["microversion"]) != `"2.1"` || string(result.Rule.Body["location"]) != "null" || result.Wire.Body[kind.required] != nil || string(result.Wire.Body["qos_policy_id"]) != `"foreign"` || string(result.Envelope) != body {
				t.Fatal(tc, result, err, count.Load())
			}
			result.Envelope[0] = 'X'
			result.Header.Set("X-Request-Id", "modified")
			if result.Policy.Header.Get("X-Request-Id") != "qos-mutation-actual" || result.Policy.StatusCode != 203 {
				t.Fatal(result)
			}
		}
	}
}

// Existing action-suite response body observation is reused for rule POSTs.
type qosRuleCreateObservedTransport struct {
	base              http.RoundTripper
	kind              string
	readErr, closeErr error
	closed            *atomic.Int32
	onClose           func()
	status            int
	lookupStatus      int
}

func (t qosRuleCreateObservedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(r)
	if response != nil && r.Method == http.MethodGet && r.URL.Path == "/neutron/v2.0/qos/policies/actual" && t.lookupStatus != 0 {
		response.StatusCode = t.lookupStatus
		response.Status = fmt.Sprint(t.lookupStatus)
	}
	if response != nil && r.Method == http.MethodPost && r.URL.Path == "/neutron/v2.0/qos/policies/actual/"+t.kind {
		if t.status != 0 {
			response.StatusCode = t.status
			response.Status = fmt.Sprint(t.status)
		}
		response.Body = &qosPolicyActionObservedBody{ReadCloser: response.Body, readErr: t.readErr, closeErr: t.closeErr, closed: t.closed, onClose: t.onClose}
	}
	return response, err
}
func TestCloudQoSRuleCreatePhysicalFailuresPreserveActualProofWithoutHooks(t *testing.T) {
	for _, kind := range cloudQoSCreateKinds() {
		for _, status := range []int{200, 401, 404, 503} {
			for _, lane := range []string{"read-close", "cancel", "source"} {
				t.Run(fmt.Sprintf("%s/%d/%s", kind.kind, status, lane), func(t *testing.T) {
					body := `{"` + kind.key + `":{"id":"new"}}`
					service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
						if qosCloudCreateGate(w, r, false) {
							return
						}
						if r.Method == http.MethodGet {
							testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
							return
						}
						testcloud.JSON(w, status, body)
					})
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					readErr, closeErr, cancelCause := errors.New("create read physical"), errors.New("create close physical"), errors.New("create caller cancel")
					base := service.RawClient().HTTPClient.Transport
					if base == nil {
						base = http.DefaultTransport
					}
					closed := &atomic.Int32{}
					observer := qosRuleCreateObservedTransport{base: base, kind: kind.kind, closed: closed}
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
						return errors.New("unexpected bounded retry hook")
					}
					service.RawClient().ProviderClient.ReauthFunc = func(context.Context) error { reauths++; return errors.New("unexpected bounded reauth hook") }
					result, err := kind.create(ctx, service, "actual", 0)
					if err == nil || result == nil || result.Policy == nil || result.Rule != nil || result.Wire != nil || result.StatusCode != status || string(result.Envelope) != body || result.Header.Get("X-Request-Id") != "qos-mutation-actual" || count.Load() != 3 || closed.Load() != 1 || retries != 0 || reauths != 0 {
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
						if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "qos-mutation-actual" {
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
						if !errors.As(proof, &native) || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "qos-mutation-actual" || native.Method != http.MethodPost {
							t.Fatal(err, native)
						}
					}
				})
			}
		}
	}
}
func TestCloudQoSRuleCreateCleanRetryRenewalSource199AndHeaderCapture(t *testing.T) {
	for _, kind := range cloudQoSCreateKinds() {
		for _, lane := range []string{"source199", "retry", "reauth", "route"} {
			t.Run(kind.kind+"/"+lane, func(t *testing.T) {
				attempts := 0
				var tokens []string
				service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("X-Owned") != "before" {
						t.Error(r.Header)
					}
					if qosCloudCreateGate(w, r, false) {
						return
					}
					if r.Method == http.MethodGet {
						testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
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
					testcloud.JSON(w, 201, `{"`+kind.key+`":{"id":"new"}}`)
				})
				service.RawClient().MoreHeaders = map[string]string{"X-Owned": "before"}
				option := network.QoSRuleCreateOption(func(*network.QoSRuleCreateOpts) error {
					service.RawClient().MoreHeaders["X-Owned"] = "after"
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
					service.RawClient().HTTPClient.Transport = qosRuleCreateObservedTransport{base: base, kind: kind.kind, closed: closed, status: 199, lookupStatus: 199}
				}
				result, err := kind.create(context.Background(), service, "actual", 0, option)
				if lane == "route" {
					if result == nil || result.Rule != nil || result.StatusCode != 503 || !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || count.Load() != 3 || callbacks != 1 {
						t.Fatal(result, err, count.Load(), callbacks)
					}
					return
				}
				want := int32(4)
				if lane == "source199" {
					want = 3
				}
				if err != nil || result == nil || result.Rule == nil || count.Load() != want {
					t.Fatal(result, err, count.Load())
				}
				if lane == "source199" {
					if result.StatusCode != 199 || result.Policy.StatusCode != 199 || callbacks != 0 || closed.Load() != 1 {
						t.Fatal(result, callbacks, closed.Load())
					}
				} else if callbacks != 1 {
					t.Fatal(callbacks)
				}
				if lane == "reauth" && !reflect.DeepEqual(tokens, []string{"test-token", "renewed"}) {
					t.Fatal(tokens)
				}
			})
		}
	}
}
func TestCloudQoSRuleCreatePreflightCancellationAndSelectedAPIGuards(t *testing.T) {
	for _, kind := range cloudQoSCreateKinds() {
		for _, lane := range []string{"canceled", "nil-context", "changed-api", "changed-route"} {
			service, count := qosCloudMutationFixture(t, func(http.ResponseWriter, *http.Request) { t.Fatal("invalid source reached HTTP") })
			ctx := context.Background()
			var options []network.QoSRuleCreateOption
			switch lane {
			case "canceled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "nil-context":
				ctx = nil
			case "changed-api":
				options = append(options, func(*network.QoSRuleCreateOpts) error { service.API = networkv2.New(service.RawClient()); return nil })
			case "changed-route":
				options = append(options, func(*network.QoSRuleCreateOpts) error { service.RawClient().ResourceBase += "changed/"; return nil })
			}
			result, err := kind.create(ctx, service, "actual", 0, options...)
			if result != nil || err == nil || count.Load() != 0 {
				t.Fatal(lane, result, err, count.Load())
			}
			if lane == "canceled" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		}
	}
}
