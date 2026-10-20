package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func qosPolicyActionFixture(t *testing.T, handler func(http.ResponseWriter, *http.Request)) (*network.Service, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	count := &atomic.Int32{}
	cloud.Mux.HandleFunc("/neutron/", func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		if (r.Method != http.MethodGet && r.Method != http.MethodDelete) || r.Header.Get("X-Auth-Token") == "" {
			t.Error(r.Method, r.Header)
		}
		w.Header().Set("X-Request-Id", "qos-action-actual")
		handler(w, r)
	})
	client := cloud.Client("network", "/catalog/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return network.New(client), count
}
func qosPolicyActionGate(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/neutron/v2.0/extensions" {
		testcloud.JSON(w, 200, `{"extensions":[{"alias":"qos"},{"alias":"other"}]}`)
		return true
	}
	return false
}
func qosPolicyActionNativeProof(t *testing.T, err error, status int, body, method string) {
	t.Helper()
	receipt := err
	if status == 401 {
		var auth *gophercloud.ErrUnableToReauthenticate
		if !errors.As(err, &auth) || auth == nil || auth.ErrOriginal == nil || auth.ErrReauth == nil {
			t.Fatal(err, auth)
		}
		receipt = auth.ErrOriginal
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(receipt, &native) || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "qos-action-actual" || native.Method != method {
		t.Fatal(err, native)
	}
}

func TestCloudQoSPolicyGetDefaultMissingAndMemberEvidence(t *testing.T) {
	service, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if qosPolicyActionGate(w, r) {
			return
		}
		if r.URL.Path == "/neutron/v2.0/qos/policies" {
			if r.URL.Query().Get("name") != "missing" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, `{"policies":[]}`)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/missing") {
			testcloud.JSON(w, 404, `{"error":"member missing"}`)
			return
		}
		if r.URL.EscapedPath() != "/neutron/v2.0/qos/policies/literal%2Fid%20%252F" {
			t.Error(r.URL)
		}
		if !reflect.DeepEqual(r.URL.Query(), url.Values{"name": {"seed"}, "fields": {"id", "tags"}, "x_seed": {"nested"}, "jmespath_filters": {"[0]"}}) {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 203, `{"policy":{"id":null,"name":"actual","tenant_id":"owner","rules":{"max_kbps":9007199254740993},"x_new":null}}`)
	})
	result, err := service.GetQoSPolicyRecord(context.Background(), "missing")
	if err != nil || result == nil || result.Policy != nil || len(result.Extensions) != 2 || count.Load() != 3 {
		t.Fatal(result, err, count.Load())
	}
	result, err = service.GetQoSPolicyRecord(context.Background(), "literal/id %2F", network.WithQoSPolicyFilters(json.RawMessage(`{"name":"seed","fields":["id","tags"],"x_seed":{"nested":true},"jmespath_filters":"[0]"}`)))
	if err != nil || result == nil || result.Policy == nil || count.Load() != 5 {
		t.Fatal(result, err, count.Load())
	}
	row := result.Policy
	for key, expected := range map[string]string{"id": "null", "name": `"actual"`, "project_id": `"owner"`, "rules": `[{"max_kbps":9007199254740993}]`, "tags": "[]", "x_seed": `{"nested":true}`} {
		if string(row.Resource.Body[key]) != expected {
			t.Fatal(key, row.Resource.Body)
		}
	}
	if row.Wire.Body["x_seed"] != nil || row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "qos-action-actual" || string(row.Envelope) != `{"policy":{"id":null,"name":"actual","tenant_id":"owner","rules":{"max_kbps":9007199254740993},"x_new":null}}` {
		t.Fatal(row)
	}
	row.Resource.Body["rules"][1] = 'X'
	if string(row.Wire.Body["rules"]) != `{"max_kbps":9007199254740993}` {
		t.Fatal(row)
	}
}
func TestCloudQoSPolicyGetFalseyFiltersRawControlsAndSourceSnapshot(t *testing.T) {
	for _, raw := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(`false`), json.RawMessage(`0`), json.RawMessage(`1e-4000`), json.RawMessage(`""`), json.RawMessage(`[]`), json.RawMessage(`{}`)} {
		service, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if qosPolicyActionGate(w, r) {
				return
			}
			if r.URL.RawQuery != "" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
		})
		result, err := service.GetQoSPolicyRecord(context.Background(), "requested", network.WithQoSPolicyFilters(raw))
		if err != nil || result == nil || result.Policy == nil || count.Load() != 2 {
			t.Fatal(string(raw), result, err, count.Load())
		}
	}
	for _, key := range []string{"name_or_id", "ignore_missing"} {
		service, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if !qosPolicyActionGate(w, r) {
				t.Fatal("unsupported kwargs policy HTTP", r.URL)
			}
		})
		filters, _ := json.Marshal(map[string]any{key: nil})
		result, err := service.GetQoSPolicyRecord(context.Background(), "requested", network.WithQoSPolicyFilters(filters))
		if result == nil || len(result.Extensions) != 2 || result.Policy != nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != 1 {
			t.Fatal(key, result, err, count.Load())
		}
	}
	for _, raw := range []string{`"not JMES"`, `[1]`, `{bad`} {
		service, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if !qosPolicyActionGate(w, r) {
				t.Fatal(r.URL)
			}
		})
		result, err := service.GetQoSPolicyRecord(context.Background(), "requested", network.WithQoSPolicyFilters(json.RawMessage(raw)))
		if result == nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != 1 {
			t.Fatal(raw, result, err, count.Load())
		}
	}
	service, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if qosPolicyActionGate(w, r) {
			return
		}
		t.Fatal("mutated route policy HTTP")
	})
	result, err := service.GetQoSPolicyRecord(context.Background(), "requested", func(c *network.QoSPolicyReadOpts) error {
		if c.Headers == nil {
			t.Error("nil callback headers")
		}
		service.RawClient().ResourceBase += "changed/"
		return nil
	})
	if result == nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != 1 {
		t.Fatal(result, err, count.Load())
	}
}
func TestCloudQoSPolicyGetFallbackListPathVersionLocalFiltersAndLateAmbiguity(t *testing.T) {
	var paths, versions []string
	service, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if qosPolicyActionGate(w, r) {
			return
		}
		paths = append(paths, r.URL.Path)
		versions = append(versions, r.Header.Get("OpenStack-API-Version"))
		if r.URL.Path != "/neutron/v2.0/alternate-policies" {
			testcloud.JSON(w, 403, `{"error":"member denied"}`)
			return
		}
		if !reflect.DeepEqual(r.URL.Query(), url.Values{"name": {"Wanted"}, "shared": {"False"}}) {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 203, `{"policies":[{"id":"skip","name":"Wanted","tenant_id":"other","rules":[9007199254740992]},{"id":"actual","name":"Wanted","tenant_id":"owner","rules":[9007199254740992]}]}`)
	})
	service.RawClient().Microversion = "2.0"
	raw := json.RawMessage(`{"list_base_path":"/alternate-policies","microversion":"2.1","name":"Wanted","is_shared":false,"tenant_id":"owner","rules":[9007199254740993.0]}`)
	option := network.WithQoSPolicyFilters(raw)
	raw[2] = 'X'
	result, err := service.GetQoSPolicyRecord(context.Background(), "actual", option)
	if err != nil || result == nil || result.Policy == nil || count.Load() != 3 || !reflect.DeepEqual(paths, []string{"/neutron/v2.0/qos/policies/actual", "/neutron/v2.0/alternate-policies"}) || !reflect.DeepEqual(versions, []string{"network 2.1", "network 2.0"}) || string(result.Policy.Resource.Body["id"]) != `"actual"` {
		t.Fatal(result, err, count.Load(), paths, versions)
	}
	successful, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if qosPolicyActionGate(w, r) {
			return
		}
		testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
	})
	result, err = successful.GetQoSPolicyRecord(context.Background(), "requested", network.WithQoSPolicyFilters(json.RawMessage(`{"list_base_path":"https://unused.invalid"}`)))
	if err != nil || result == nil || result.Policy == nil || count.Load() != 2 {
		t.Fatal(result, err, count.Load())
	}
	duplicate, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if qosPolicyActionGate(w, r) {
			return
		}
		if r.URL.Path != "/neutron/v2.0/qos/policies" {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		if r.URL.Query().Get("marker") == "later" {
			testcloud.JSON(w, 200, `{"policies":[{"id":"later","name":"same"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"policies":[{"id":"first","name":"same"}],"next":"?name=same&marker=later"}`)
	})
	result, err = duplicate.GetQoSPolicyRecord(context.Background(), "same")
	if result == nil || result.Policy != nil || !errors.Is(err, resource.ErrAmbiguous) || count.Load() != 4 {
		t.Fatal(result, err, count.Load())
	}
}
func TestCloudQoSPolicyDeleteMissingFoundAndIgnored404HaveDifferentResults(t *testing.T) {
	for _, status := range []int{200, 203, 204, 299, 302, 399, 404} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var methods, paths []string
			body := `raw deletion receipt`
			service, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
				methods = append(methods, r.Method)
				paths = append(paths, r.URL.EscapedPath())
				if qosPolicyActionGate(w, r) {
					return
				}
				if r.Method == http.MethodGet {
					testcloud.JSON(w, 203, `{"policy":{"id":"actual/id %2F","name":"display"}}`)
					return
				}
				data, err := io.ReadAll(r.Body)
				if err != nil || len(data) != 0 || r.URL.RawQuery != "" || r.Header.Get("If-Match") != "" {
					t.Error(string(data), err, r.URL, r.Header)
				}
				if status == 204 {
					w.WriteHeader(status)
					return
				}
				testcloud.JSON(w, status, body)
			})
			result, err := service.DeleteQoSPolicyRecord(context.Background(), "requested")
			expectedBody := body
			if status == 204 {
				expectedBody = ""
			}
			if err != nil || result == nil || !result.Deleted || result.Policy == nil || result.Delete == nil || result.Delete.StatusCode != status || string(result.Delete.Envelope) != expectedBody || result.Delete.Header.Get("X-Request-Id") != "qos-action-actual" || result.Delete.IgnoredMissing != (status == 404) || count.Load() != 3 {
				t.Fatal(result, err, count.Load())
			}
			if !reflect.DeepEqual(methods, []string{"GET", "GET", "DELETE"}) || !reflect.DeepEqual(paths, []string{"/neutron/v2.0/extensions", "/neutron/v2.0/qos/policies/requested", "/neutron/v2.0/qos/policies/actual%2Fid%20%252F"}) {
				t.Fatal(methods, paths)
			}
		})
	}
	missing, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if qosPolicyActionGate(w, r) {
			return
		}
		if r.Method == http.MethodDelete {
			t.Fatal("missing lookup DELETE", r.URL)
		}
		if r.URL.Path == "/neutron/v2.0/qos/policies" {
			testcloud.JSON(w, 200, `{"policies":[]}`)
		} else {
			testcloud.JSON(w, 404, `{}`)
		}
	})
	result, err := missing.DeleteQoSPolicyRecord(context.Background(), "missing")
	if err != nil || result == nil || result.Deleted || result.Policy != nil || result.Delete != nil || count.Load() != 3 {
		t.Fatal(result, err, count.Load())
	}
}
func TestCloudQoSPolicyDeleteReturnedJSONIDAndIndependentActualReceipts(t *testing.T) {
	for _, tc := range []struct{ raw, member string }{{`9007199254740993`, "9007199254740993"}, {`true`, "True"}, {`1e-4000`, ""}, {`null`, ""}, {`false`, ""}, {`0`, ""}, {`""`, ""}} {
		service, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if qosPolicyActionGate(w, r) {
				return
			}
			if r.Method == http.MethodGet {
				testcloud.JSON(w, 200, `{"policy":{"id":`+tc.raw+`,"x_new":9007199254740993}}`)
				return
			}
			if r.URL.Path != "/neutron/v2.0/qos/policies/"+tc.member {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 203, `raw success`)
		})
		result, err := service.DeleteQoSPolicyRecord(context.Background(), "requested")
		if tc.member == "" {
			if result == nil || result.Deleted || result.Delete != nil || result.Policy == nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != 2 {
				t.Fatal(tc.raw, result, err, count.Load())
			}
			continue
		}
		if err != nil || result == nil || !result.Deleted || result.Delete == nil || count.Load() != 3 || string(result.Policy.Wire.Body["id"]) != tc.raw {
			t.Fatal(tc.raw, result, err, count.Load())
		}
		result.Delete.Envelope[0] = 'X'
		result.Delete.Header.Set("X-Request-Id", "changed")
		if result.Policy.Header.Get("X-Request-Id") != "qos-action-actual" || string(result.Policy.Envelope) != `{"policy":{"id":`+tc.raw+`,"x_new":9007199254740993}}` {
			t.Fatal(result)
		}
	}
}

type qosPolicyActionObservedBody struct {
	io.ReadCloser
	readErr, closeErr error
	closed            *atomic.Int32
	onClose           func()
}

func (b *qosPolicyActionObservedBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if n > 0 && b.readErr != nil {
		return n, b.readErr
	}
	return n, err
}
func (b *qosPolicyActionObservedBody) Close() error {
	b.closed.Add(1)
	err := b.ReadCloser.Close()
	if b.onClose != nil {
		b.onClose()
	}
	return errors.Join(err, b.closeErr)
}

type qosPolicyActionObservedTransport struct {
	base              http.RoundTripper
	method            string
	readErr, closeErr error
	closed            *atomic.Int32
	onClose           func()
	status            int
}

func (t qosPolicyActionObservedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(r)
	if response != nil && r.Method == t.method && r.URL.Path == "/neutron/v2.0/qos/policies/actual" {
		if t.status != 0 {
			response.StatusCode = t.status
			response.Status = fmt.Sprint(t.status)
		}
		response.Body = &qosPolicyActionObservedBody{ReadCloser: response.Body, readErr: t.readErr, closeErr: t.closeErr, closed: t.closed, onClose: t.onClose}
	}
	return response, err
}
func TestCloudQoSPolicyActionsPhysicalFaultsNeverFallbackMissingOrRetry(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		for _, status := range []int{200, 401, 404, 503} {
			for _, lane := range []string{"read-close", "cancel", "source"} {
				t.Run(fmt.Sprintf("%s/%d/%s", method, status, lane), func(t *testing.T) {
					body := `{"policy":{"id":"actual"}}`
					if method == http.MethodDelete {
						body = `actual rejected DELETE`
					}
					service, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
						if qosPolicyActionGate(w, r) {
							return
						}
						if method == http.MethodDelete && r.Method == http.MethodGet {
							testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
							return
						}
						testcloud.JSON(w, status, body)
					})
					readErr, closeErr, cancelCause := errors.New("read physical"), errors.New("close physical"), errors.New("caller cancellation")
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					closed := &atomic.Int32{}
					base := service.RawClient().HTTPClient.Transport
					if base == nil {
						base = http.DefaultTransport
					}
					observed := qosPolicyActionObservedTransport{base: base, method: method, closed: closed}
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
					var err error
					if method == http.MethodGet {
						result, cause := service.GetQoSPolicyRecord(ctx, "actual")
						err = cause
						if result == nil || result.Policy != nil {
							t.Fatal(result, err)
						}
					} else {
						result, cause := service.DeleteQoSPolicyRecord(ctx, "actual")
						err = cause
						if result == nil || result.Deleted || result.Policy == nil {
							t.Fatal(result, err)
						}
					}
					want := int32(2)
					if method == http.MethodDelete {
						want = 3
					}
					if err == nil || count.Load() != want || closed.Load() != 1 || retries != 0 || reauths != 0 {
						t.Fatal(err, count.Load(), closed.Load(), retries, reauths)
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
						if !errors.As(err, &receipt) || receipt.StatusCode != status || string(receipt.Body) != body || receipt.Header.Get("X-Request-Id") != "qos-action-actual" {
							t.Fatal(err, receipt)
						}
					} else {
						qosPolicyActionNativeProof(t, err, status, body, method)
						if status == 401 {
							var auth *gophercloud.ErrUnableToReauthenticate
							if !errors.As(err, &auth) || auth == nil {
								t.Fatal(err)
							}
							if lane == "read-close" && (!errors.Is(auth.ErrReauth, readErr) || !errors.Is(auth.ErrReauth, closeErr)) {
								t.Fatal(err, auth)
							}
						}
					}
				})
			}
		}
	}
}
func TestCloudQoSPolicyActionsCleanRetryRenewalAndRouteGuard(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		for _, lane := range []string{"retry", "reauth", "source"} {
			t.Run(method+"/"+lane, func(t *testing.T) {
				attempts := 0
				var tokens []string
				service, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
					if qosPolicyActionGate(w, r) {
						return
					}
					if method == http.MethodDelete && r.Method == http.MethodGet {
						testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
						return
					}
					attempts++
					tokens = append(tokens, r.Header.Get("X-Auth-Token"))
					if attempts == 1 {
						status := 503
						if lane == "reauth" {
							status = 401
						}
						testcloud.JSON(w, status, `{"error":"original"}`)
						return
					}
					if method == http.MethodGet {
						testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
					} else {
						w.WriteHeader(204)
					}
				})
				callbacks := 0
				if lane == "reauth" {
					service.RawClient().ProviderClient.ReauthFunc = func(context.Context) error {
						callbacks++
						service.RawClient().ProviderClient.SetToken("renewed")
						return nil
					}
				} else {
					service.RawClient().ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
						callbacks++
						if lane == "source" {
							service.RawClient().ResourceBase += "changed/"
						}
						return nil
					}
				}
				var err error
				if method == http.MethodGet {
					result, cause := service.GetQoSPolicyRecord(context.Background(), "actual")
					err = cause
					if result == nil {
						t.Fatal(result, err)
					}
				} else {
					result, cause := service.DeleteQoSPolicyRecord(context.Background(), "actual")
					err = cause
					if result == nil || result.Deleted != (lane != "source") {
						t.Fatal(result, err)
					}
				}
				baseCalls := int32(2)
				if method == http.MethodDelete {
					baseCalls = 3
				}
				if lane == "source" {
					if !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || count.Load() != baseCalls || callbacks != 1 {
						t.Fatal(err, count.Load(), callbacks)
					}
				} else {
					if err != nil || count.Load() != baseCalls+1 || callbacks != 1 {
						t.Fatal(err, count.Load(), callbacks)
					}
					if lane == "reauth" && !reflect.DeepEqual(tokens, []string{"test-token", "renewed"}) {
						t.Fatal(tokens)
					}
				}
			})
		}
	}
}
func TestCloudQoSPolicyActionsSourceStatusPolicyAndGateFailures(t *testing.T) {
	service, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if qosPolicyActionGate(w, r) {
			return
		}
		if r.Method == http.MethodGet {
			testcloud.JSON(w, 200, `{"policy":{"id":"actual"}}`)
		} else {
			testcloud.JSON(w, 200, `raw deletion199`)
		}
	})
	base := service.RawClient().HTTPClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	closed := &atomic.Int32{}
	service.RawClient().HTTPClient.Transport = qosPolicyActionObservedTransport{base: base, method: http.MethodDelete, closed: closed, status: 199}
	result, err := service.DeleteQoSPolicyRecord(context.Background(), "actual")
	if err != nil || result == nil || !result.Deleted || result.Delete == nil || result.Delete.StatusCode != 199 || string(result.Delete.Envelope) != `raw deletion199` || count.Load() != 3 || closed.Load() != 1 {
		t.Fatal(result, err, count.Load(), closed.Load())
	}
	for _, operation := range []string{"get", "delete"} {
		unavailable, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/neutron/v2.0/extensions" {
				t.Fatal("policy before unavailable gate", r.URL)
			}
			testcloud.JSON(w, 200, `{"extensions":[{"alias":"other"}]}`)
		})
		var err error
		if operation == "get" {
			result, cause := unavailable.GetQoSPolicyRecord(context.Background(), "actual")
			err = cause
			if result == nil || len(result.Extensions) != 1 {
				t.Fatal(result, err)
			}
		} else {
			result, cause := unavailable.DeleteQoSPolicyRecord(context.Background(), "actual")
			err = cause
			if result == nil || result.Deleted || len(result.Extensions) != 1 {
				t.Fatal(result, err)
			}
		}
		if !errors.Is(err, network.ErrUnavailableExtension) || count.Load() != 1 {
			t.Fatal(err, count.Load())
		}
	}
	var nilService *network.Service
	if result, err := nilService.GetQoSPolicyRecord(context.Background(), "actual"); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(result, err)
	}
	before := count.Load()
	if result, err := service.DeleteQoSPolicyRecord(nil, "actual"); result != nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != before {
		t.Fatal(result, err, count.Load())
	}
}
