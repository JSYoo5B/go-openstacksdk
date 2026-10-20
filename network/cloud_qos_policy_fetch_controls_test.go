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

func TestCloudQoSPolicyFetchControlsFreshFullGateAndPolicyProfile(t *testing.T) {
	var paths []string
	const body = `{"selected":{"id":"actual","tenant_id":"owner","shared":0,"rules":{"rate":9007199254740993},"location":{"cloud":"wire"},"x_unknown":null},"policy":{"id":"wrong"}}`
	service, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path+"?"+r.URL.RawQuery)
		if r.Method != http.MethodGet {
			t.Error(r.Method)
		}
		if r.URL.Path == "/neutron/v2.0/extensions" {
			if r.Header.Get("X-Typed") != "" {
				t.Error("policy options leaked into discovery", r.Header)
			}
			if r.URL.Query().Get("marker") == "tail" {
				testcloud.JSON(w, 200, `{"extensions":[{"alias":"other"}]}`)
			} else {
				testcloud.JSON(w, 203, `{"extensions":[{"alias":"qos"}],"next":"?marker=tail"}`)
			}
			return
		}
		if r.URL.Path != "/neutron/v2.0/owned-policies" || r.URL.RawQuery != "" || r.Header.Get("X-Typed") != "owned" || r.Header.Get("Cache-Control") != "" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 203, body)
	})
	option := network.WithQoSPolicyFilters(json.RawMessage(`{"requires_id":false,"base_path":"/owned-policies","skip_cache":true,"error_message":"custom lookup","resource_response_key":"selected"}`))
	for i := 0; i < 2; i++ {
		result, err := service.GetQoSPolicyRecord(context.Background(), "requested", option, network.WithQoSPolicyHeaders(map[string]string{"X-Typed": "owned"}))
		if err != nil || result == nil || result.Policy == nil || len(result.Extensions) != 2 {
			t.Fatal(result, err)
		}
		for index, code := range []int{203, 200} {
			ext := result.Extensions[index]
			if ext.StatusCode != code || ext.Header.Get("X-Request-Id") != "qos-action-actual" || len(ext.Envelope) == 0 {
				t.Fatal(ext)
			}
		}
		row := result.Policy
		for key, want := range map[string]string{"id": `"actual"`, "project_id": `"owner"`, "is_shared": `false`, "rules": `[{"rate":9007199254740993}]`, "tags": `[]`, "location": `null`, "x_unknown": `null`, "requires_id": `false`, "skip_cache": `true`} {
			if string(row.Resource.Body[key]) != want {
				t.Fatal(key, row.Resource.Body)
			}
		}
		if row.StatusCode != 203 || string(row.Envelope) != body || row.Header.Get("X-Request-Id") != "qos-action-actual" || row.Wire.Body["skip_cache"] != nil || string(row.Wire.Body["location"]) != `{"cloud":"wire"}` || string(row.Wire.Body["rules"]) != `{"rate":9007199254740993}` {
			t.Fatal(row)
		}
	}
	if count.Load() != 6 || !reflect.DeepEqual(paths, []string{"/neutron/v2.0/extensions?", "/neutron/v2.0/extensions?marker=tail", "/neutron/v2.0/owned-policies?", "/neutron/v2.0/extensions?", "/neutron/v2.0/extensions?marker=tail", "/neutron/v2.0/owned-policies?"}) {
		t.Fatal(count.Load(), paths)
	}
}

func TestCloudQoSPolicyFetchControlsRawHeadersHaveSeparatePhases(t *testing.T) {
	var paths, versions []string
	service, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if qosPolicyActionGate(w, r) {
			return
		}
		paths = append(paths, r.URL.Path)
		versions = append(versions, r.Header.Get("OpenStack-API-Version"))
		if r.Header.Get("X-Owned") != "typed" {
			t.Error(r.Header)
		}
		if len(paths) == 1 {
			if !reflect.DeepEqual(r.URL.Query()["headers"], []string{"Accept", "X-Phase"}) || r.Header.Get("X-Phase") != "typed" {
				t.Error(r.URL, r.Header)
			}
			testcloud.JSON(w, 404, `{"error":"missing"}`)
			return
		}
		if r.URL.Query().Has("headers") || r.URL.Query().Get("name") != "target" || r.Header.Get("X-Phase") != "raw" || r.Header.Get("Accept") != "application/vnd.policy+json" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"policies":[{"id":"found","name":"target"}]}`)
	})
	service.RawClient().Microversion = "2.0"
	headers := map[string]string{"X-Phase": "typed", "X-Owned": "typed"}
	result, err := service.GetQoSPolicyRecord(context.Background(), "target", network.WithQoSPolicyHeaders(headers), network.WithQoSPolicyFilters(json.RawMessage(`{"base_path":"/owned-policies","list_base_path":"/fallback-policies","microversion":"2.1","headers":{"Accept":"application/vnd.policy+json","X-Phase":"raw"}}`)))
	if err != nil || result == nil || result.Policy == nil || count.Load() != 3 || !reflect.DeepEqual(paths, []string{"/neutron/v2.0/owned-policies/target", "/neutron/v2.0/fallback-policies"}) || !reflect.DeepEqual(versions, []string{"network 2.1", "network 2.0"}) || service.RawClient().Microversion != "2.0" || headers["X-Phase"] != "typed" {
		t.Fatal(result, err, count.Load(), paths, versions)
	}
}

func TestCloudQoSPolicyFetchControlsInvalidHeadersWaitForFallback(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		t.Run(fmt.Sprint(fallback), func(t *testing.T) {
			service, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if qosPolicyActionGate(w, r) {
					return
				}
				if !strings.HasSuffix(r.URL.Path, "/target") || r.URL.Query().Get("headers") != "X-Raw" {
					t.Error(r.URL)
				}
				if fallback {
					testcloud.JSON(w, 404, `{}`)
				} else {
					testcloud.JSON(w, 200, `{"policy":{"id":"found"}}`)
				}
			})
			result, err := service.GetQoSPolicyRecord(context.Background(), "target", network.WithQoSPolicyFilters(json.RawMessage(`{"headers":{"X-Raw":2}}`)))
			if result == nil || len(result.Extensions) != 2 || count.Load() != 2 {
				t.Fatal(result, err, count.Load())
			}
			if fallback {
				if err == nil || result.Policy != nil {
					t.Fatal(result, err)
				}
			} else if err != nil || result.Policy == nil {
				t.Fatal(result, err)
			}
		})
	}
}

func TestCloudQoSPolicyFetchControlsOwnedOptionsAndFalseyEnvelope(t *testing.T) {
	raw := json.RawMessage(`{"base_path":"/owned-policies","requires_id":false,"resource_response_key":false,"skip_cache":false}`)
	headers := map[string]string{"X-Owned": "original"}
	option := network.WithQoSPolicyReadOpts(network.QoSPolicyReadOpts{Filters: raw, Headers: headers})
	copy(raw, []byte(`{"base_path":"/other-policies"}`))
	headers["X-Owned"] = "changed"
	calls := 0
	service, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if qosPolicyActionGate(w, r) {
			return
		}
		calls++
		if r.URL.Path != "/neutron/v2.0/owned-policies" || r.URL.RawQuery != "" || r.Header.Get("X-Owned") != "original" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"policy":{"id":"policy-%d"},"false":{"id":"wrong"}}`, calls))
	})
	for i := 1; i <= 2; i++ {
		result, err := service.GetQoSPolicyRecord(context.Background(), "target", option)
		if err != nil || result == nil || result.Policy == nil || string(result.Policy.Resource.Body["id"]) != fmt.Sprintf(`"policy-%d"`, i) {
			t.Fatal(result, err)
		}
	}
	if count.Load() != 4 {
		t.Fatal(count.Load())
	}
}

func TestCloudQoSPolicyFetchControlsCustomErrorKeepsNativeProof(t *testing.T) {
	for _, status := range []int{404, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			const body = `{"error":"actual member failure"}`
			service, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if qosPolicyActionGate(w, r) {
					return
				}
				if r.URL.Query().Has("error_message") {
					t.Error(r.URL)
				}
				if strings.HasSuffix(r.URL.Path, "/target") {
					testcloud.JSON(w, status, body)
				} else {
					testcloud.JSON(w, 200, `{"policies":[]}`)
				}
			})
			result, err := service.GetQoSPolicyRecord(context.Background(), "target", network.WithQoSPolicyFilters(json.RawMessage(`{"error_message":"custom policy lookup failed"}`)))
			if result == nil || result.Policy != nil || len(result.Extensions) != 2 {
				t.Fatal(result, err)
			}
			if status == 404 {
				if err != nil || count.Load() != 3 {
					t.Fatal(err, count.Load())
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), "custom policy lookup failed") || count.Load() != 2 {
					t.Fatal(err, count.Load())
				}
				qosPolicyActionNativeProof(t, err, status, body, http.MethodGet)
			}
		})
	}
}

func TestCloudQoSPolicyFetchControlsPhysicalFaultNeverReplaysOrDeletes(t *testing.T) {
	for _, status := range []int{200, 404} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			const body = `{"policy":{"id":"actual"}}`
			service, count := qosPolicyActionFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if qosPolicyActionGate(w, r) {
					return
				}
				if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/actual") {
					t.Error(r.Method, r.URL)
				}
				testcloud.JSON(w, status, body)
			})
			readErr, closeErr := errors.New("controlled read failure"), errors.New("controlled close failure")
			closed := &atomic.Int32{}
			base := service.RawClient().HTTPClient.Transport
			if base == nil {
				base = http.DefaultTransport
			}
			service.RawClient().HTTPClient.Transport = qosPolicyActionObservedTransport{base: base, method: http.MethodGet, readErr: readErr, closeErr: closeErr, closed: closed}
			retries, reauths := 0, 0
			service.RawClient().ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			service.RawClient().ProviderClient.ReauthFunc = func(context.Context) error { reauths++; return nil }
			result, err := service.GetQoSPolicyRecord(context.Background(), "actual", network.WithQoSPolicyFilters(json.RawMessage(`{"requires_id":true,"skip_cache":true,"error_message":"controlled policy lookup"}`)))
			if result == nil || result.Policy != nil || len(result.Extensions) != 2 || !errors.Is(err, readErr) || !errors.Is(err, closeErr) || count.Load() != 2 || closed.Load() != 1 || retries != 0 || reauths != 0 {
				t.Fatal(result, err, count.Load(), closed.Load(), retries, reauths)
			}
			if status < 400 {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "qos-action-actual" {
					t.Fatal(err, proof)
				}
			} else {
				qosPolicyActionNativeProof(t, err, status, body, http.MethodGet)
			}
		})
	}
}
