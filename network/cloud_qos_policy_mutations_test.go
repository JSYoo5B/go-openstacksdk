package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func qosCloudMutationFixture(t *testing.T, handler func(http.ResponseWriter, *http.Request)) (*network.Service, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	count := &atomic.Int32{}
	cloud.Mux.HandleFunc("/neutron/", func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		if r.Header.Get("X-Auth-Token") == "" {
			t.Error(r.Header)
		}
		w.Header().Set("X-Request-Id", "qos-mutation-actual")
		handler(w, r)
	})
	client := cloud.Client("network", "/catalog/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return network.New(client), count
}
func qosCloudMutationGate(w http.ResponseWriter, r *http.Request, withDefault bool) bool {
	if r.Method == http.MethodGet && r.URL.Path == "/neutron/v2.0/extensions" {
		body := `{"extensions":[{"alias":"qos"}]}`
		if withDefault {
			body = `{"extensions":[{"alias":"qos"},{"alias":"qos-default"}]}`
		}
		testcloud.JSON(w, 200, body)
		return true
	}
	return false
}
func qosCloudMutationBody(t *testing.T, r *http.Request) map[string]json.RawMessage {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	var envelope map[string]json.RawMessage
	if json.Unmarshal(data, &envelope) != nil {
		t.Fatal(string(data))
	}
	var body map[string]json.RawMessage
	if json.Unmarshal(envelope["policy"], &body) != nil {
		t.Fatal(string(data))
	}
	return body
}
func TestCloudQoSPolicyMutationValidKwargsBeforeGateAndOwnedOptions(t *testing.T) {
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if qosCloudMutationGate(w, r, false) {
			return
		}
		body := qosCloudMutationBody(t, r)
		if r.Method != http.MethodPost || string(body["name"]) != `"last"` || len(body) != 1 {
			t.Error(r.Method, body)
		}
		testcloud.JSON(w, 201, `{"policy":{"id":"new","name":"last"}}`)
	})
	raw := json.RawMessage(`{"name":"first"}`)
	option := network.WithQoSPolicyMutationJSON(raw)
	raw[2] = 'X'
	result, err := service.CreateQoSPolicyRecord(context.Background(), option, network.WithQoSPolicyName("last"))
	if err != nil || result == nil || result.Policy == nil || count.Load() != 2 {
		t.Fatal(result, err, count.Load())
	}
	for _, options := range [][]network.QoSPolicyMutationOption{{nil}, {network.WithQoSPolicyAttribute("is_shared", make(chan int))}, {network.WithQoSPolicyMutationJSON(json.RawMessage(`{"is_default":false}`))}, {network.WithQoSPolicyMutationJSON(json.RawMessage(`{"name":1,"bad":null}`))}, {network.WithQoSPolicyMutationJSON(json.RawMessage(`[]`))}, {network.WithQoSPolicyMutationJSON(json.RawMessage(`{bad`))}} {
		before := count.Load()
		result, err := service.CreateQoSPolicyRecord(context.Background(), options...)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != before {
			t.Fatal(result, err, count.Load())
		}
		result, err = service.UpdateQoSPolicyRecord(context.Background(), "id", options...)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != before {
			t.Fatal(result, err, count.Load())
		}
	}
}
func TestCloudQoSPolicyCreateDefaultNonnullFreshGatesAndEmptyPOST(t *testing.T) {
	for _, tc := range []struct {
		raw       string
		available bool
		gates     int
		expected  string
		dropped   bool
	}{
		{`{}`, false, 1, "", false}, {`{"default":null}`, true, 1, "", false}, {`{"default":false}`, true, 2, "false", false}, {`{"default":0}`, true, 2, "0", false}, {`{"default":[]}`, true, 2, "[]", false}, {`{"default":{}}`, true, 2, "{}", false}, {`{"default":false}`, false, 2, "", true},
	} {
		gates := 0
		var posted map[string]json.RawMessage
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if qosCloudMutationGate(w, r, tc.available) {
				gates++
				return
			}
			posted = qosCloudMutationBody(t, r)
			if r.Method != http.MethodPost {
				t.Error(r.Method)
			}
			testcloud.JSON(w, 201, `{"policy":{"id":"new"}}`)
		})
		result, err := service.CreateQoSPolicyRecord(context.Background(), network.WithQoSPolicyMutationJSON(json.RawMessage(tc.raw)))
		if err != nil || result == nil || result.Policy == nil || result.NoOp || result.DefaultDropped != tc.dropped || gates != tc.gates || count.Load() != int32(tc.gates+1) || string(posted["is_default"]) != tc.expected || posted["default"] != nil {
			t.Fatal(tc.raw, result, err, posted, gates, count.Load())
		}
		if tc.expected == "" && len(posted) != 0 {
			t.Fatal(posted)
		}
	}
}
func TestCloudQoSPolicyUpdateEmptyAfterDefaultSkipsLookupAndMissingRaises(t *testing.T) {
	for _, tc := range []struct {
		raw       string
		available bool
		gates     int
		dropped   bool
	}{{`{}`, true, 1, false}, {`{"default":null}`, true, 1, false}, {`{"default":false}`, false, 2, true}} {
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if !qosCloudMutationGate(w, r, tc.available) {
				t.Fatal("empty update lookup/mutation", r.URL)
			}
		})
		result, err := service.UpdateQoSPolicyRecord(context.Background(), "", network.WithQoSPolicyMutationJSON(json.RawMessage(tc.raw)))
		if err != nil || result == nil || !result.NoOp || result.Policy != nil || result.Lookup != nil || result.DefaultDropped != tc.dropped || count.Load() != int32(tc.gates) {
			t.Fatal(tc.raw, result, err, count.Load())
		}
	}
	missing, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if qosCloudMutationGate(w, r, false) {
			return
		}
		if r.Method != http.MethodGet {
			t.Fatal("missing PUT", r.URL)
		}
		if r.URL.Path == "/neutron/v2.0/qos/policies" {
			testcloud.JSON(w, 200, `{"policies":[]}`)
		} else {
			testcloud.JSON(w, 404, `{}`)
		}
	})
	result, err := missing.UpdateQoSPolicyRecord(context.Background(), "missing", network.WithQoSPolicyName("new"))
	if result == nil || result.Policy != nil || !errors.Is(err, resource.ErrNotFound) || count.Load() != 3 {
		t.Fatal(result, err, count.Load())
	}
}
func TestCloudQoSPolicyUpdateRawDirtyMissingAliasAndOnlyChangedPayload(t *testing.T) {
	for _, tc := range []struct {
		current, attrs, expected string
		put                      bool
	}{
		{`{"id":"actual","name":"gold","shared":true}`, `{"name":"gold","shared":1}`, `{}`, false},
		{`{"id":"actual"}`, `{"name":null}`, `{"name":null}`, true},
		{`{"id":"actual","tenant_id":"owner"}`, `{"project_id":"owner"}`, `{"project_id":"owner"}`, true},
		{`{"id":"actual","shared":"false"}`, `{"shared":true}`, `{"shared":true}`, true},
		{`{"id":"actual","name":"old","description":"same","shared":false}`, `{"name":"new","description":"same","shared":0}`, `{"name":"new"}`, true},
	} {
		puts := 0
		var payload map[string]json.RawMessage
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if qosCloudMutationGate(w, r, false) {
				return
			}
			if r.Method == http.MethodGet {
				testcloud.JSON(w, 203, `{"policy":`+tc.current+`}`)
				return
			}
			puts++
			if r.Method != http.MethodPut || r.URL.Path != "/neutron/v2.0/qos/policies/actual" {
				t.Error(r.Method, r.URL)
			}
			payload = qosCloudMutationBody(t, r)
			testcloud.JSON(w, 200, `{"policy":{"id":"actual","x_new":9007199254740993}}`)
		})
		result, err := service.UpdateQoSPolicyRecord(context.Background(), "requested", network.WithQoSPolicyMutationJSON(json.RawMessage(tc.attrs)))
		wantCalls := int32(2)
		if tc.put {
			wantCalls = 3
		}
		if err != nil || result == nil || result.Policy == nil || result.Lookup == nil || result.NoOp == tc.put || count.Load() != wantCalls || puts != qosMutationBoolToInt(tc.put) {
			t.Fatal(tc.attrs, result, err, count.Load(), puts)
		}
		if tc.put {
			var expected map[string]json.RawMessage
			json.Unmarshal([]byte(tc.expected), &expected)
			if !reflect.DeepEqual(payload, expected) {
				t.Fatal(payload, expected)
			}
			if result.Policy.Wire.Body["name"] != nil && string(result.Policy.Wire.Body["name"]) != `"new"` {
				t.Fatal(result.Policy)
			}
		} else if result.Policy.StatusCode != 203 || string(result.Policy.Envelope) != `{"policy":`+tc.current+`}` {
			t.Fatal(result.Policy)
		}
	}
}
func qosMutationBoolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// Ensure rejected transport handling retains a native value status when no
// actual resource result exists. Larger physical matrices reuse this fixture.
func TestCloudQoSPolicyMutationRejectedStatusPreservesActualEvidence(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		for _, status := range []int{400, 403, 404, 409, 503} {
			body := `{"error":"actual mutation"}`
			service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if qosCloudMutationGate(w, r, false) {
					return
				}
				if r.Method == http.MethodGet {
					testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
					return
				}
				testcloud.JSON(w, status, body)
			})
			var result *network.QoSPolicyMutation
			var err error
			if method == http.MethodPost {
				result, err = service.CreateQoSPolicyRecord(context.Background(), network.WithQoSPolicyName("new"))
			} else {
				result, err = service.UpdateQoSPolicyRecord(context.Background(), "actual", network.WithQoSPolicyName("new"))
			}
			expected := int32(2)
			if method == http.MethodPut {
				expected = 3
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if result == nil || result.Policy == nil || result.Policy.Resource != nil || result.Policy.StatusCode != status || string(result.Policy.Envelope) != body || result.Policy.Header.Get("X-Request-Id") != "qos-mutation-actual" || !errors.As(err, &native) || native.Actual != status || string(native.Body) != body || count.Load() != expected {
				t.Fatal(result, err, native, count.Load())
			}
		}
	}
}

// Reuse the action suite's observed real response body; only the POST/PUT
// selection differs. The HTTP fixture remains Gophercloud testhelper-backed.
type qosPolicyMutationObservedTransport struct {
	base              http.RoundTripper
	method            string
	readErr, closeErr error
	closed            *atomic.Int32
	onClose           func()
	status            int
}

func (t qosPolicyMutationObservedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(r)
	if response != nil && r.Method == t.method && (r.URL.Path == "/neutron/v2.0/qos/policies" || r.URL.Path == "/neutron/v2.0/qos/policies/actual") {
		if t.status != 0 {
			response.StatusCode = t.status
			response.Status = fmt.Sprint(t.status)
		}
		response.Body = &qosPolicyActionObservedBody{ReadCloser: response.Body, readErr: t.readErr, closeErr: t.closeErr, closed: t.closed, onClose: t.onClose}
	}
	return response, err
}
func TestCloudQoSPolicyMutationPhysicalFaultsPreserveActualEvidenceWithoutHooks(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		for _, status := range []int{200, 401, 404, 503} {
			for _, lane := range []string{"read-close", "cancel", "source"} {
				t.Run(fmt.Sprintf("%s/%d/%s", method, status, lane), func(t *testing.T) {
					body := `{"policy":{"id":"actual","name":"new"}}`
					service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
						if qosCloudMutationGate(w, r, false) {
							return
						}
						if r.Method == http.MethodGet {
							testcloud.JSON(w, 200, `{"policy":{"id":"actual","name":"old"}}`)
							return
						}
						testcloud.JSON(w, status, body)
					})
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					readErr, closeErr, cancelCause := errors.New("mutation read physical"), errors.New("mutation close physical"), errors.New("mutation caller cancellation")
					base := service.RawClient().HTTPClient.Transport
					if base == nil {
						base = http.DefaultTransport
					}
					closed := &atomic.Int32{}
					observed := qosPolicyMutationObservedTransport{base: base, method: method, closed: closed}
					switch lane {
					case "read-close":
						observed.readErr, observed.closeErr = readErr, closeErr
					case "cancel":
						observed.onClose = func() { cancel(cancelCause) }
					case "source":
						observed.onClose = func() { service.RawClient().ResourceBase += "changed/" }
					}
					service.RawClient().HTTPClient.Transport = observed
					retries, reauths := 0, 0
					service.RawClient().ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
						retries++
						return nil
					}
					service.RawClient().ProviderClient.ReauthFunc = func(context.Context) error { reauths++; return nil }
					var result *network.QoSPolicyMutation
					var err error
					if method == http.MethodPost {
						result, err = service.CreateQoSPolicyRecord(ctx, network.WithQoSPolicyName("new"))
					} else {
						result, err = service.UpdateQoSPolicyRecord(ctx, "actual", network.WithQoSPolicyName("new"))
					}
					want := int32(2)
					if method == http.MethodPut {
						want = 3
					}
					if err == nil || result == nil || result.NoOp || result.Policy == nil || result.Policy.Resource != nil || result.Policy.Wire != nil || result.Policy.StatusCode != status || string(result.Policy.Envelope) != body || result.Policy.Header.Get("X-Request-Id") != "qos-mutation-actual" || count.Load() != want || closed.Load() != 1 || retries != 0 || reauths != 0 {
						t.Fatal(result, err, count.Load(), closed.Load(), retries, reauths)
					}
					if method == http.MethodPut && (result.Lookup == nil || result.Lookup.StatusCode != 200 || string(result.Lookup.Resource.Body["name"]) != `"old"`) {
						t.Fatal(result)
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
						var receipt *resource.ResponseError
						if !errors.As(err, &receipt) || receipt.StatusCode != status || string(receipt.Body) != body || receipt.Header.Get("X-Request-Id") != "qos-mutation-actual" {
							t.Fatal(err, receipt)
						}
					} else {
						receipt := err
						if status == 401 {
							var auth *gophercloud.ErrUnableToReauthenticate
							if !errors.As(err, &auth) || auth == nil || auth.ErrOriginal == nil || auth.ErrReauth == nil {
								t.Fatal(err, auth)
							}
							receipt = auth.ErrOriginal
							if lane == "read-close" && (!errors.Is(auth.ErrReauth, readErr) || !errors.Is(auth.ErrReauth, closeErr)) {
								t.Fatal(err, auth)
							}
						}
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(receipt, &native) || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "qos-mutation-actual" || native.Method != method {
							t.Fatal(err, native)
						}
					}
				})
			}
		}
	}
}
func TestCloudQoSPolicyMutationSourceStatusPolicyCleanRetryAndImmutableHeaders(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		for _, lane := range []string{"source199", "retry", "reauth", "route"} {
			t.Run(method+"/"+lane, func(t *testing.T) {
				attempts := 0
				var tokens []string
				service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("X-Owned") != "before" {
						t.Error(r.Header)
					}
					if qosCloudMutationGate(w, r, false) {
						return
					}
					if r.Method == http.MethodGet {
						testcloud.JSON(w, 200, `{"policy":{"id":"actual","name":"old"}}`)
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
					testcloud.JSON(w, 200, `{"policy":{"id":"actual","name":"new"}}`)
				})
				service.RawClient().MoreHeaders = map[string]string{"X-Owned": "before"}
				option := network.QoSPolicyMutationOption(func(opts *network.QoSPolicyMutationOpts) error {
					service.RawClient().MoreHeaders["X-Owned"] = "after"
					opts.Attributes = map[string]json.RawMessage{"name": json.RawMessage(`"new"`)}
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
					service.RawClient().HTTPClient.Transport = qosPolicyMutationObservedTransport{base: base, method: method, closed: closed, status: 199}
				}
				var result *network.QoSPolicyMutation
				var err error
				if method == http.MethodPost {
					result, err = service.CreateQoSPolicyRecord(context.Background(), option)
				} else {
					result, err = service.UpdateQoSPolicyRecord(context.Background(), "actual", option)
				}
				want := int32(2)
				if method == http.MethodPut {
					want = 3
				}
				if lane == "route" {
					if !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || count.Load() != want || callbacks != 1 || result == nil || result.Policy == nil || result.Policy.StatusCode != 503 {
						t.Fatal(result, err, count.Load(), callbacks)
					}
					return
				}
				if lane != "source199" {
					want++
				}
				if err != nil || result == nil || result.Policy == nil || result.NoOp || count.Load() != want {
					t.Fatal(result, err, count.Load(), callbacks)
				}
				if lane == "source199" {
					if result.Policy.StatusCode != 199 || closed.Load() != 1 || callbacks != 0 {
						t.Fatal(result, closed.Load(), callbacks)
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
