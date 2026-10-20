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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func qosPolicyCloudFixture(t *testing.T, handler func(http.ResponseWriter, *http.Request)) (*network.Service, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := &atomic.Int32{}
	cloud.Mux.HandleFunc("/neutron/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.Header.Get("X-Auth-Token") == "" {
			t.Error(r.Method, r.Header)
		}
		w.Header().Set("X-Request-Id", "qos-cloud-actual")
		handler(w, r)
	})
	client := cloud.Client("network", "/catalog/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return network.New(client), calls
}
func qosPolicyCloudGate(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/neutron/v2.0/extensions" {
		testcloud.JSON(w, 200, `{"extensions":[{"alias":"qos"},{"alias":"other"}]}`)
		return true
	}
	return false
}
func qosPolicyCloudField(t *testing.T, row *policies.PolicyRecord, key, expected string) {
	t.Helper()
	if row == nil || row.Resource == nil || string(row.Resource.Body[key]) != expected {
		t.Fatal(row, key, expected)
	}
}
func TestCloudQoSPolicyListAndSearchGateEveryCallWithoutEarlyExit(t *testing.T) {
	var paths []string
	service, calls := qosPolicyCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path+"?"+r.URL.RawQuery)
		if r.URL.Path == "/neutron/v2.0/extensions" {
			if r.URL.Query().Get("marker") == "later" {
				testcloud.JSON(w, 200, `{"extensions":[{"alias":"other"}]}`)
				return
			}
			testcloud.JSON(w, 203, `{"extensions":[{"alias":"qos"}],"next":"?marker=later"}`)
			return
		}
		testcloud.JSON(w, 203, `{"policies":[{"id":"one","name":"gold","tenant_id":"owner","shared":0,"rules":{"max_kbps":9007199254740993},"x_new":null}]}`)
	})
	for _, search := range []bool{false, true} {
		var result *network.QoSPolicyRecords
		var err error
		if search {
			result, err = service.SearchQoSPolicyRecords(context.Background(), "literal/id*")
		} else {
			result, err = service.ListQoSPolicyRecords(context.Background())
		}
		if err != nil || result == nil || len(result.Extensions) != 2 || len(result.Policies) != 1 || len(result.Value) != 1 || result.JSON == nil {
			t.Fatal(result, err)
		}
		row := result.Policies[0]
		qosPolicyCloudField(t, row, "project_id", `"owner"`)
		qosPolicyCloudField(t, row, "is_shared", `false`)
		qosPolicyCloudField(t, row, "rules", `[{"max_kbps":9007199254740993}]`)
		if row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "qos-cloud-actual" || result.Extensions[0].StatusCode != 203 || row.Wire.Body["project_id"] != nil {
			t.Fatal(result)
		}
	}
	expected := []string{"/neutron/v2.0/extensions?", "/neutron/v2.0/extensions?marker=later", "/neutron/v2.0/qos/policies?", "/neutron/v2.0/extensions?", "/neutron/v2.0/extensions?marker=later", "/neutron/v2.0/qos/policies?name=literal%2Fid%2A"}
	if calls.Load() != 6 || !reflect.DeepEqual(paths, expected) {
		t.Fatal(calls.Load(), paths)
	}
	for _, body := range []string{`{"extensions":[{"alias":"unrelated"}]}`, `{"extensions":[{"alias":"qos"}],"next":"?marker=later"}`} {
		missing, count := qosPolicyCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/neutron/v2.0/extensions" {
				t.Fatal("policy before complete gate", r.URL)
			}
			if r.URL.Query().Get("marker") == "later" {
				testcloud.JSON(w, 403, `{"error":"late extension permission"}`)
				return
			}
			testcloud.JSON(w, 200, body)
		})
		result, err := missing.ListQoSPolicyRecords(context.Background())
		if result == nil || len(result.Extensions) == 0 || result.Policies != nil || err == nil {
			t.Fatal(result, err)
		}
		if count.Load() == 1 && !errors.Is(err, network.ErrUnavailableExtension) {
			t.Fatal(err)
		}
		if count.Load() == 2 && !gophercloud.ResponseCodeIs(err, 403) {
			t.Fatal(err)
		}
	}
}

func TestCloudQoSPolicyFiltersFalseyAndMappingErrorsHappenAfterGate(t *testing.T) {
	for _, raw := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(`false`), json.RawMessage(`0`), json.RawMessage(`-0.0`), json.RawMessage(`1e-4000`), json.RawMessage(`""`), json.RawMessage(`[]`), json.RawMessage(`{}`)} {
		service, count := qosPolicyCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if qosPolicyCloudGate(w, r) {
				return
			}
			if r.URL.RawQuery != "" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, `{"policies":[]}`)
		})
		result, err := service.ListQoSPolicyRecords(context.Background(), network.WithQoSPolicyFilters(raw))
		if err != nil || result == nil || result.Policies == nil || result.Value == nil || string(result.JSON) != "[]" || count.Load() != 2 {
			t.Fatal(string(raw), result, err, count.Load())
		}
	}
	for _, raw := range []string{`"not JMES"`, `true`, `1`, `[1]`, `{malformed`} {
		service, count := qosPolicyCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if !qosPolicyCloudGate(w, r) {
				t.Fatal("invalid filters policy HTTP", r.URL)
			}
		})
		result, err := service.ListQoSPolicyRecords(context.Background(), network.WithQoSPolicyFilters(json.RawMessage(raw)))
		if result == nil || len(result.Extensions) != 2 || result.Policies != nil || result.Value != nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != 1 {
			t.Fatal(raw, result, err, count.Load())
		}
	}
}
func TestCloudQoSPolicySearchLiteralNameAndOrderedUpdateAreNotLocalSearch(t *testing.T) {
	for _, tc := range []struct {
		filters  string
		expected url.Values
	}{
		{`{"name":"override","is_shared":false,"fields":["id","x_new"],"unknown":1}`, url.Values{"name": {"override"}, "shared": {"False"}, "fields": {"id", "x_new"}}},
		{`{"name":null}`, url.Values{}},
		{`[["name","old"],["name","override"],["is_default",false]]`, url.Values{"name": {"override"}, "is_default": {"False"}}},
		{`["xy"]`, url.Values{"name": {"literal/id*"}}},
		{`[{"a":"unused","b":"unused"}]`, url.Values{"name": {"literal/id*"}}},
	} {
		service, count := qosPolicyCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if qosPolicyCloudGate(w, r) {
				return
			}
			if !reflect.DeepEqual(r.URL.Query(), tc.expected) {
				t.Error(r.URL, tc.expected)
			}
			// Deliberately a different ID/name: cloud Search must trust server filtering.
			testcloud.JSON(w, 200, `{"policies":[{"id":"unrelated","name":"unrelated"}]}`)
		})
		result, err := service.SearchQoSPolicyRecords(context.Background(), "literal/id*", network.WithQoSPolicyFilters(json.RawMessage(tc.filters)))
		if err != nil || result == nil || len(result.Policies) != 1 || count.Load() != 2 {
			t.Fatal(tc.filters, result, err, count.Load())
		}
	}
	for _, raw := range []string{`"ab"`, `[[1,"value"]]`, `[["name"]]`, `[true]`} {
		service, count := qosPolicyCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if !qosPolicyCloudGate(w, r) {
				t.Fatal(r.URL)
			}
		})
		result, err := service.SearchQoSPolicyRecords(context.Background(), "gold", network.WithQoSPolicyFilters(json.RawMessage(raw)))
		if result == nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != 1 {
			t.Fatal(raw, result, err, count.Load())
		}
	}
}
func TestCloudQoSPolicyRawFormalsAliasesAndLocalJSONAreOwned(t *testing.T) {
	var policyCalls int
	service, count := qosPolicyCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if qosPolicyCloudGate(w, r) {
			return
		}
		policyCalls++
		if r.URL.Path != "/neutron/v2.0/alternate-policies" || r.Header.Get("X-Trace") != "owned" || r.Header.Get("OpenStack-API-Version") != "network 2.1" || !reflect.DeepEqual(r.URL.Query(), url.Values{"project_id": {"owner"}, "shared": {"False"}, "limit": {"2"}}) {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 203, `{"policies":[{"id":"skip","tenant_id":"other","rules":[{"max_kbps":9007199254740993}]},{"id":"keep","tenant_id":"owner","rules":[{"max_kbps":9007199254740993}]}],"next":"https://foreign.invalid"}`)
	})
	raw := json.RawMessage(`{"base_path":"/alternate-policies","headers":{"X-Trace":"owned"},"microversion":"2.1","project_id":"owner","is_shared":false,"tenant_id":"owner","rules":[{"max_kbps":9007199254740993}],"max_items":2,"__conflicting_attrs":{"jmespath_filters":"[?"}}`)
	option := network.WithQoSPolicyFilters(raw)
	raw[2] = 'X'
	result, err := service.ListQoSPolicyRecords(context.Background(), option)
	if err != nil || result == nil || len(result.Policies) != 1 || policyCalls != 1 || count.Load() != 2 || string(result.Policies[0].Resource.Body["id"]) != `"keep"` {
		t.Fatal(result, err, count.Load())
	}
	for _, filters := range []string{`{"cls":null}`, `{"__conflicting_attrs":{"base_path":"x"}}`, `{"base_path":"https://foreign.invalid"}`} {
		before := count.Load()
		result, err := service.ListQoSPolicyRecords(context.Background(), network.WithQoSPolicyFilters(json.RawMessage(filters)))
		if result == nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != before+1 {
			t.Fatal(filters, result, err, count.Load())
		}
	}
}
func TestCloudQoSPolicyProxyIterableAndOpaqueResultsPreserveHTTPPhases(t *testing.T) {
	for _, tc := range []struct {
		expr, expected             string
		policyHTTP, failed, opaque bool
	}{
		{"@", "", true, false, false},
		{"`[1,2]`", "[1,2]", false, false, false},
		{"`\"é🙂\"`", `["é","🙂"]`, false, false, false},
		{"`{\"z\":1,\"a\":2}`", `["a","z"]`, false, false, false},
		{"to_array(@)", "", false, false, true},
		{"[0]", "", false, true, false},
		{"length(@)", "", false, true, false},
		{"`1`", "", false, true, false},
	} {
		t.Run(tc.expr, func(t *testing.T) {
			service, count := qosPolicyCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if qosPolicyCloudGate(w, r) {
					return
				}
				testcloud.JSON(w, 200, `{"policies":[{"id":"one"}]}`)
			})
			filters, _ := json.Marshal(map[string]any{"jmespath_filters": tc.expr})
			result, err := service.ListQoSPolicyRecords(context.Background(), network.WithQoSPolicyFilters(filters))
			expectedCalls := int32(1)
			if tc.policyHTTP {
				expectedCalls = 2
			}
			if (err != nil) != tc.failed || result == nil || count.Load() != expectedCalls {
				t.Fatal(result, err, count.Load())
			}
			if tc.failed {
				if result.Policies != nil || result.Value != nil {
					t.Fatal(result)
				}
				return
			}
			if tc.expected != "" && string(result.JSON) != tc.expected {
				t.Fatal(string(result.JSON), tc.expected)
			}
			if tc.opaque {
				if result.JSON != nil || len(result.Value) != 1 {
					t.Fatal(result)
				}
				if _, ok := result.Value[0].(*policies.RecordListGenerator); !ok {
					t.Fatal(result.Value)
				}
			}
		})
	}
}
func TestCloudQoSPolicyReadOptionsSnapshotsCallbacksAndLateFailures(t *testing.T) {
	raw := json.RawMessage(`{"fields":["id"]}`)
	headers := map[string]string{"X-Trace": "owned"}
	version := "2.1"
	option := network.WithQoSPolicyReadOpts(network.QoSPolicyReadOpts{Filters: raw, Headers: headers, Microversion: &version})
	raw[2] = 'X'
	headers["X-Trace"] = "changed"
	version = "changed"
	var lock sync.Mutex
	service, _ := qosPolicyCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if qosPolicyCloudGate(w, r) {
			return
		}
		if r.Header.Get("X-Trace") != "owned" || r.Header.Get("OpenStack-API-Version") != "network 2.1" || r.URL.Query().Get("fields") != "id" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"policies":[]}`)
	})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := service.ListQoSPolicyRecords(context.Background(), option, func(c *network.QoSPolicyReadOpts) error {
				if c.Headers == nil {
					t.Error("nil callback headers")
				}
				return nil
			})
			if err != nil || result == nil {
				lock.Lock()
				t.Error(result, err)
				lock.Unlock()
			}
		}()
	}
	wg.Wait()
	late, count := qosPolicyCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if qosPolicyCloudGate(w, r) {
			return
		}
		if r.URL.Query().Get("marker") == "later" {
			testcloud.JSON(w, 403, `{"error":"late policy permission"}`)
			return
		}
		testcloud.JSON(w, 200, `{"policies":[{"id":"first"}],"next":"?marker=later"}`)
	})
	result, err := late.ListQoSPolicyRecords(context.Background())
	if result == nil || len(result.Extensions) != 2 || result.Policies != nil || result.Value != nil || !gophercloud.ResponseCodeIs(err, 403) || count.Load() != 3 {
		t.Fatal(result, err, count.Load())
	}
	before := count.Load()
	result, err = late.ListQoSPolicyRecords(context.Background(), func(c *network.QoSPolicyReadOpts) error {
		late.RawClient().ResourceBase += "changed/"
		return nil
	})
	if result == nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != before+1 {
		t.Fatal(result, err, count.Load())
	}
}

type qosPolicyCloudObservedBody struct {
	io.ReadCloser
	closed  *atomic.Int32
	fault   error
	onClose func()
}

func (b *qosPolicyCloudObservedBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if n > 0 && b.fault != nil {
		return n, b.fault
	}
	return n, err
}
func (b *qosPolicyCloudObservedBody) Close() error {
	b.closed.Add(1)
	err := b.ReadCloser.Close()
	if b.onClose != nil {
		b.onClose()
	}
	return errors.Join(err, b.fault)
}

type qosPolicyCloudObservedTransport struct {
	base    http.RoundTripper
	closed  *atomic.Int32
	fault   error
	onClose func()
}

func (t qosPolicyCloudObservedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(r)
	if response != nil && r.URL.Path == "/neutron/v2.0/qos/policies" {
		response.Body = &qosPolicyCloudObservedBody{ReadCloser: response.Body, closed: t.closed, fault: t.fault, onClose: t.onClose}
	}
	return response, err
}
func TestCloudQoSPolicyPhysicalFaultsAndCleanRetryKeepActualReceipt(t *testing.T) {
	for _, status := range []int{200, 401, 404, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			service, count := qosPolicyCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if qosPolicyCloudGate(w, r) {
					return
				}
				testcloud.JSON(w, status, `{"policies":[{"id":"actual"}]}`)
			})
			fault := errors.New("policy body and close")
			closed := &atomic.Int32{}
			base := service.RawClient().HTTPClient.Transport
			if base == nil {
				base = http.DefaultTransport
			}
			service.RawClient().HTTPClient.Transport = qosPolicyCloudObservedTransport{base: base, closed: closed, fault: fault}
			retries, reauths := 0, 0
			service.RawClient().ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			service.RawClient().ProviderClient.ReauthFunc = func(context.Context) error { reauths++; return nil }
			result, err := service.ListQoSPolicyRecords(context.Background())
			if result == nil || result.Policies != nil || !errors.Is(err, fault) || count.Load() != 2 || closed.Load() != 1 || retries != 0 || reauths != 0 {
				t.Fatal(result, err, count.Load(), closed.Load(), retries, reauths)
			}
			if status == 200 {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != `{"policies":[{"id":"actual"}]}` || proof.Header.Get("X-Request-Id") != "qos-cloud-actual" || !errors.Is(proof.Cause, fault) {
					t.Fatal(err, proof)
				}
			} else {
				receipt := err
				if status == 401 {
					// Native ErrUnableToReauthenticate deliberately does not
					// unwrap its independent original/handling errors.
					var auth *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &auth) || auth == nil || auth.ErrOriginal == nil || !errors.Is(auth.ErrReauth, fault) {
						t.Fatal(err, auth)
					}
					receipt = auth.ErrOriginal
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(receipt, &native) || native.Actual != status || native.Method != http.MethodGet || !strings.HasSuffix(native.URL, "/neutron/v2.0/qos/policies") || string(native.Body) != `{"policies":[{"id":"actual"}]}` || native.ResponseHeader.Get("X-Request-Id") != "qos-cloud-actual" {
					t.Fatal(err, native)
				}
			}
		})
	}
	for _, mutate := range []bool{false, true} {
		attempts := 0
		service, count := qosPolicyCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if qosPolicyCloudGate(w, r) {
				return
			}
			attempts++
			if attempts == 1 {
				testcloud.JSON(w, 503, `{"error":"original"}`)
				return
			}
			testcloud.JSON(w, 200, `{"policies":[]}`)
		})
		callbacks := 0
		service.RawClient().ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			callbacks++
			if mutate {
				service.RawClient().ResourceBase += "changed/"
			}
			return nil
		}
		result, err := service.ListQoSPolicyRecords(context.Background())
		if mutate {
			if result == nil || !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || count.Load() != 2 || callbacks != 1 {
				t.Fatal(result, err, count.Load(), callbacks)
			}
		} else if err != nil || result == nil || count.Load() != 3 || callbacks != 1 {
			t.Fatal(result, err, count.Load(), callbacks)
		}
	}
}
