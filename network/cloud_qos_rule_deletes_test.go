package network_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type cloudQoSDeleteKind struct {
	name, path string
	delete     func(context.Context, *network.Service, string, string, ...network.QoSRuleDeleteOption) (*network.QoSRuleDeleteResult, error)
}

func cloudQoSDeleteKinds() []cloudQoSDeleteKind {
	return []cloudQoSDeleteKind{
		{"DeleteQoSBandwidthLimitRule", "bandwidth_limit_rules", func(ctx context.Context, s *network.Service, p, id string, o ...network.QoSRuleDeleteOption) (*network.QoSRuleDeleteResult, error) {
			return s.DeleteQoSBandwidthLimitRule(ctx, p, id, o...)
		}},
		{"DeleteQoSDSCPMarkingRule", "dscp_marking_rules", func(ctx context.Context, s *network.Service, p, id string, o ...network.QoSRuleDeleteOption) (*network.QoSRuleDeleteResult, error) {
			return s.DeleteQoSDSCPMarkingRule(ctx, p, id, o...)
		}},
		{"DeleteQoSMinimumBandwidthRule", "minimum_bandwidth_rules", func(ctx context.Context, s *network.Service, p, id string, o ...network.QoSRuleDeleteOption) (*network.QoSRuleDeleteResult, error) {
			return s.DeleteQoSMinimumBandwidthRule(ctx, p, id, o...)
		}},
	}
}
func cloudQoSDeleteReplies(kind string, status int, body string) map[string][]pythonCloudBReply {
	return map[string][]pythonCloudBReply{
		"GET /v2.0/extensions":                               {{200, `{"extensions":[{"alias":"qos"}]}`}},
		"GET /v2.0/qos/policies/parent":                      {{200, `{"policy":{"id":"parent","name":"selected"}}`}},
		"DELETE /v2.0/qos/policies/parent/" + kind + "/rule": {{status, body}},
	}
}
func TestCloudQoSRuleDeleteSourceTrueFalseAndOtherStatuses(t *testing.T) {
	for _, kind := range cloudQoSDeleteKinds() {
		for _, status := range []int{204, 202, 302, 399, 400, 403, 404, 408, 409, 500} {
			body := "opaque ACK"
			if status == 204 {
				body = ""
			}
			client, calls := pythonCloudBClient(t, cloudQoSDeleteReplies(kind.path, status, body))
			result, err := kind.delete(context.Background(), network.New(client), "parent", "rule")
			if result == nil || len(result.Extensions) != 1 || result.Policy == nil || result.Receipt == nil || result.Receipt.StatusCode != status || string(result.Receipt.Envelope) != body || len(calls()) != 3 {
				t.Fatal(kind.name, status, result, err, calls())
			}
			if status == 404 {
				if err != nil || !result.Completed || result.Deleted || result.Receipt.IgnoredMissing {
					t.Fatal(result, err)
				}
			} else if status < 400 {
				if err != nil || !result.Completed || !result.Deleted {
					t.Fatal(result, err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if result.Completed || result.Deleted || !errors.As(err, &native) || native.Actual != status {
					t.Fatal(result, err, native)
				}
			}
			pythonCloudBRequests(t, calls(), "GET /v2.0/extensions", "GET /v2.0/qos/policies/parent", "DELETE /v2.0/qos/policies/parent/"+kind.path+"/rule")
		}
	}
}
func TestCloudQoSRuleDeleteGateAndPolicyErrorsNeverBecomeFalse(t *testing.T) {
	for _, kind := range cloudQoSDeleteKinds() {
		client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/extensions": {{200, `{"extensions":[{"alias":"other"}]}`}}})
		result, err := kind.delete(context.Background(), network.New(client), "parent", "rule")
		if result == nil || result.Completed || result.Receipt != nil || result.Policy != nil || len(result.Extensions) != 1 || !errors.Is(err, network.ErrUnavailableExtension) || len(calls()) != 1 {
			t.Fatal(result, err, calls())
		}
		for _, body := range []string{`{"policies":[]}`, `{"policies":[{"id":"a","name":"missing"},{"id":"b","name":"missing"}]}`} {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
				"GET /v2.0/extensions":           {{200, `{"extensions":[{"alias":"qos"}]}`}},
				"GET /v2.0/qos/policies/missing": {{404, `{"error":"policy missing"}`}},
				"GET /v2.0/qos/policies":         {{200, body}},
			})
			result, err := kind.delete(context.Background(), network.New(client), "missing", "rule")
			if result == nil || result.Completed || result.Receipt != nil || len(result.Extensions) != 1 || err == nil || len(calls()) != 3 {
				t.Fatal(kind.name, result, err, calls())
			}
			if body == `{"policies":[]}` && !errors.Is(err, resource.ErrNotFound) || body != `{"policies":[]}` && !errors.Is(err, resource.ErrAmbiguous) {
				t.Fatal(err)
			}
		}
	}
}
func TestCloudQoSRuleDeleteConsumesEntireExtensionGateBeforeMutation(t *testing.T) {
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/extensions": {
		{200, `{"extensions":[{"alias":"qos"}],"extensions_links":[{"rel":"next","href":"?marker=later"}]}`},
		{503, `{"error":"actual later page"}`},
	}})
	result, err := network.New(client).DeleteQoSBandwidthLimitRule(context.Background(), "parent", "rule")
	var native gophercloud.ErrUnexpectedResponseCode
	if result == nil || len(result.Extensions) != 1 || result.Policy != nil || result.Receipt != nil || result.Completed || !errors.As(err, &native) || native.Actual != 503 || len(calls()) != 2 {
		t.Fatal(result, err, native, calls())
	}
}
func TestCloudQoSRuleDeleteOwnedHeadersVersionAndSafeIDs(t *testing.T) {
	cloud := testcloud.New(t)
	var paths []string
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/v2.0/extensions":
			testcloud.JSON(w, 200, `{"extensions":[{"alias":"qos"}]}`)
		case "/v2.0/qos/policies/parent":
			testcloud.JSON(w, 200, `{"policy":{"id":"parent"}}`)
		default:
			if r.Method != http.MethodDelete || r.URL.RawQuery != "" || r.Header.Get("X-Trace") != "owned" || r.Header.Get("X-Callback") != "yes" || r.Header.Get("OpenStack-API-Version") != "network 2.1" {
				t.Error(r.Method, r.URL, r.Header)
			}
			w.Header().Set("X-Request-Id", "delete-only")
			w.WriteHeader(204)
		}
	})
	client := cloud.Client("network", "/")
	client.ResourceBase = cloud.Server.URL + "/v2.0/"
	service := network.New(client)
	headers := map[string]string{"X-Trace": "owned"}
	option := network.WithQoSRuleDeleteHeaders(headers)
	headers["X-Trace"] = "changed"
	result, err := service.DeleteQoSBandwidthLimitRule(context.Background(), "parent", "rule", network.WithQoSRuleDeleteHeaders(nil), option, network.WithQoSRuleDeleteMicroversion("2.1"), func(c *network.QoSRuleDeleteOpts) error { c.Headers["X-Callback"] = "yes"; return nil })
	if err != nil || result == nil || !result.Deleted || result.Receipt.Header.Get("X-Request-Id") != "delete-only" || len(paths) != 3 {
		t.Fatal(result, err, paths)
	}
	for _, id := range []string{"", ".", "..", "\n"} {
		result, err = service.DeleteQoSBandwidthLimitRule(context.Background(), "parent", id)
		if result == nil || result.Completed || result.Receipt != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(id, result, err, paths)
		}
	}
	if len(paths) != 11 {
		t.Fatal("unsafe rule IDs issued DELETE", paths)
	}
}
func TestCloudQoSRuleDeleteCallbackSourceAndCancellationFailuresKeepParents(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		client, calls := pythonCloudBClient(t, cloudQoSDeleteReplies("bandwidth_limit_rules", 204, ""))
		service := network.New(client)
		ctx, cancel := context.WithCancelCause(context.Background())
		cause, optionCause := errors.New("cancelled cloud"), errors.New("delete option")
		result, err := service.DeleteQoSBandwidthLimitRule(ctx, "parent", "rule", func(*network.QoSRuleDeleteOpts) error {
			if cancelled {
				cancel(cause)
			} else {
				client.ResourceBase += "changed/"
			}
			return optionCause
		})
		cancel(nil)
		if result == nil || result.Completed || result.Receipt != nil || result.Policy == nil || len(result.Extensions) != 1 || len(calls()) != 2 || !errors.Is(err, optionCause) || cancelled && !errors.Is(err, cause) || !cancelled && !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(result, err, calls())
		}
	}
	client, calls := pythonCloudBClient(t, cloudQoSDeleteReplies("bandwidth_limit_rules", 204, ""))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := network.New(client).DeleteQoSBandwidthLimitRule(ctx, "parent", "rule")
	if result == nil || result.Completed || len(result.Extensions) != 0 || result.Receipt != nil || !errors.Is(err, context.Canceled) || len(calls()) != 0 {
		t.Fatal(result, err, calls())
	}
}
func TestCloudQoSRuleDeleteFaulty404And401KeepRealReceiptNotFalse(t *testing.T) {
	for _, status := range []int{404, 401} {
		cloud := testcloud.New(t)
		count := 0
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			count++
			switch r.URL.Path {
			case "/v2.0/extensions":
				testcloud.JSON(w, 200, `{"extensions":[{"alias":"qos"}]}`)
			case "/v2.0/qos/policies/parent":
				testcloud.JSON(w, 200, `{"policy":{"id":"parent"}}`)
			default:
				w.Header().Set("Content-Length", "1000")
				w.Header().Set("X-Request-Id", "actual-delete-fault")
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"error":"partial"}`))
			}
		})
		client := cloud.Client("network", "/")
		client.ResourceBase = cloud.Server.URL + "/v2.0/"
		retry, reauth := 0, 0
		client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			retry++
			return nil
		}
		client.ReauthFunc = func(context.Context) error { reauth++; return nil }
		result, err := network.New(client).DeleteQoSDSCPMarkingRule(context.Background(), "parent", "rule")
		if result == nil || result.Completed || result.Deleted || result.Receipt == nil || result.Receipt.StatusCode != status || result.Receipt.Header.Get("X-Request-Id") != "actual-delete-fault" || string(result.Receipt.Envelope) != `{"error":"partial"}` || !errors.Is(err, io.ErrUnexpectedEOF) || retry != 0 || reauth != 0 || count != 3 {
			t.Fatal(result, err, retry, reauth, count)
		}
	}
}
func TestCloudQoSRuleDeleteExpanded404AndCallback404NeverCompleteFalse(t *testing.T) {
	for _, expanded := range []bool{false, true} {
		status := 503
		if expanded {
			status = 404
		}
		replies := cloudQoSDeleteReplies("minimum_bandwidth_rules", status, `{"error":"actual"}`)
		if expanded {
			key := "DELETE /v2.0/qos/policies/parent/minimum_bandwidth_rules/rule"
			replies[key] = append(replies[key], replies[key][0])
		}
		client, calls := pythonCloudBClient(t, replies)
		marker := errors.New("retry injected404")
		client.RetryFunc = func(_ context.Context, _ string, _ string, c *gophercloud.RequestOpts, _ error, _ uint) error {
			if expanded {
				c.OkCodes = append(c.OkCodes, 404)
				return nil
			}
			return errors.Join(gophercloud.ErrUnexpectedResponseCode{Actual: 404}, marker)
		}
		result, err := network.New(client).DeleteQoSMinimumBandwidthRule(context.Background(), "parent", "rule")
		if result == nil || result.Completed || result.Deleted || result.Receipt == nil || result.Receipt.StatusCode != status || err == nil || expanded && len(calls()) != 4 || !expanded && (!errors.Is(err, marker) || len(calls()) != 3) {
			t.Fatal(expanded, result, err, calls())
		}
	}
}
