package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// updateReceiptHelper adapts one cloud update helper to a common shape: it
// changes the name to "new" and reports the receipt plus the native model's
// name when a native model was projected.
type updateReceiptHelper struct {
	name, path, key string
	call            func(context.Context, *network.Service) (*network.UpdateReceipt, *string, bool, error)
}

func updateReceiptHelpers() []updateReceiptHelper {
	return []updateReceiptHelper{
		{"router", "/v2.0/routers/p", "router", func(ctx context.Context, s *network.Service) (*network.UpdateReceipt, *string, bool, error) {
			r, err := s.UpdateRouterRecord(ctx, "p", network.WithRouterUpdateJSON(json.RawMessage(`{"name":"new"}`)))
			if r == nil {
				return nil, nil, false, err
			}
			if r.Router == nil {
				return r.Response, nil, r.NoOp, err
			}
			return r.Response, &r.Router.Name, r.NoOp, err
		}},
		{"subnet", "/v2.0/subnets/p", "subnet", func(ctx context.Context, s *network.Service) (*network.UpdateReceipt, *string, bool, error) {
			r, err := s.UpdateSubnetRecord(ctx, "p", network.WithSubnetUpdateJSON(json.RawMessage(`{"subnet_name":"new"}`)))
			if r == nil {
				return nil, nil, false, err
			}
			if r.Subnet == nil {
				return r.Response, nil, r.NoOp, err
			}
			return r.Response, &r.Subnet.Name, r.NoOp, err
		}},
		{"port", "/v2.0/ports/p", "port", func(ctx context.Context, s *network.Service) (*network.UpdateReceipt, *string, bool, error) {
			r, err := s.UpdatePortRecord(ctx, "p", network.WithPortUpdateJSON(json.RawMessage(`{"name":"new"}`)))
			if r == nil {
				return nil, nil, false, err
			}
			if r.Port == nil {
				return r.Response, nil, r.NoOp, err
			}
			return r.Response, &r.Port.Name, r.NoOp, err
		}},
		{"firewall-group", "/v2.0/fwaas/firewall_groups/p", "firewall_group", func(ctx context.Context, s *network.Service) (*network.UpdateReceipt, *string, bool, error) {
			r, err := s.UpdateFirewallGroupRecord(ctx, "p", network.WithFirewallGroupUpdateJSON(json.RawMessage(`{"name":"new"}`)))
			if r == nil {
				return nil, nil, false, err
			}
			if r.Group == nil {
				return r.Response, nil, r.NoOp, err
			}
			return r.Response, &r.Group.Name, r.NoOp, err
		}},
		{"firewall-policy", "/v2.0/fwaas/firewall_policies/p", "firewall_policy", func(ctx context.Context, s *network.Service) (*network.UpdateReceipt, *string, bool, error) {
			r, err := s.UpdateFirewallPolicyRecord(ctx, "p", network.WithFirewallPolicyUpdateJSON(json.RawMessage(`{"name":"new"}`)))
			if r == nil {
				return nil, nil, false, err
			}
			if r.Policy == nil {
				return r.Response, nil, r.NoOp, err
			}
			return r.Response, &r.Policy.Name, r.NoOp, err
		}},
		{"firewall-rule", "/v2.0/fwaas/firewall_rules/p", "firewall_rule", func(ctx context.Context, s *network.Service) (*network.UpdateReceipt, *string, bool, error) {
			r, err := s.UpdateFirewallRuleRecord(ctx, "p", network.WithFirewallRuleUpdateJSON(json.RawMessage(`{"name":"new"}`)))
			if r == nil {
				return nil, nil, false, err
			}
			if r.Rule == nil {
				return r.Response, nil, r.NoOp, err
			}
			return r.Response, &r.Rule.Name, r.NoOp, err
		}},
	}
}

type receiptReply struct {
	status      int
	contentType string
	body        string
}

// receiptClient serves one lookup GET and one PUT reply for path and counts
// every request, so tests can prove no PUT is replayed.
func receiptClient(t *testing.T, path, key, lookup string, put receiptReply) (*gophercloud.ServiceClient, func() []string) {
	t.Helper()
	cloud := testcloud.New(t)
	var mu sync.Mutex
	var calls []string
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		mu.Lock()
		calls = append(calls, r.Method+" "+r.URL.Path+" "+string(raw))
		mu.Unlock()
		switch {
		case r.Method == http.MethodGet && r.URL.Path == path:
			testcloud.JSON(w, http.StatusOK, `{"`+key+`":`+lookup+`}`)
		case r.Method == http.MethodPut && r.URL.Path == path:
			if put.contentType != "" {
				w.Header().Set("Content-Type", put.contentType)
			}
			w.Header().Set("X-Openstack-Request-Id", "req-put")
			w.WriteHeader(put.status)
			_, _ = io.WriteString(w, put.body)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			testcloud.JSON(w, http.StatusInternalServerError, `{}`)
		}
	})
	client := cloud.Client("network", "/")
	client.ResourceBase = cloud.Server.URL + "/v2.0/"
	return client, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), calls...)
	}
}

func rawText(raw json.RawMessage) string { return string(raw) }

// TestCloudUpdateReceiptMergesPythonTranslateResponse covers the accepted PUT
// bodies Python Resource._translate_response merges into the found resource:
// complete and partial resource-key bodies, top-level bodies without the key,
// and ignored empty/malformed bodies. Unknown members survive in Body/Merged.
func TestCloudUpdateReceiptMergesPythonTranslateResponse(t *testing.T) {
	const lookup = `{"id":"p","name":"old","status":"ACTIVE","x_lookup_only":{"a":[1,null]}}`
	for _, helper := range updateReceiptHelpers() {
		for _, tc := range []struct {
			name        string
			reply       receiptReply
			ignored     bool
			wantName    string
			wantStatus  string
			wantUnknown string
			wantBody    map[string]string
		}{
			{"complete", receiptReply{200, "application/json", `{"` + helper.key + `":{"id":"p","name":"resp","status":"DOWN","x_new":false}}`}, false, "resp", `"DOWN"`, "false", map[string]string{"id": `"p"`, "name": `"resp"`, "status": `"DOWN"`, "x_new": "false"}},
			{"partial", receiptReply{200, "application/json", `{"` + helper.key + `":{"name":"resp","self":"dropped"}}`}, false, "resp", `"ACTIVE"`, "", map[string]string{"name": `"resp"`}},
			{"top-level-without-key", receiptReply{202, "application/json", `{"name":"top","x_new":0,"self":"dropped"}`}, false, "top", `"ACTIVE"`, "0", map[string]string{"name": `"top"`, "x_new": "0"}},
			{"empty-200", receiptReply{200, "", ``}, true, "new", `"ACTIVE"`, "", nil},
			{"no-content", receiptReply{204, "", ``}, true, "new", `"ACTIVE"`, "", nil},
			{"malformed", receiptReply{200, "application/json", `{"` + helper.key + `":`}, true, "new", `"ACTIVE"`, "", nil},
			{"nonfinite", receiptReply{200, "application/json", `{"` + helper.key + `":{"name":NaN}}`}, true, "new", `"ACTIVE"`, "", nil},
		} {
			t.Run(helper.name+"/"+tc.name, func(t *testing.T) {
				client, calls := receiptClient(t, helper.path, helper.key, lookup, tc.reply)
				receipt, name, noOp, err := helper.call(context.Background(), network.New(client))
				if err != nil || noOp || receipt == nil || receipt.Merged == nil || receipt.ProjectionError != nil {
					t.Fatal(receipt, err)
				}
				if got := calls(); len(got) != 2 || !strings.HasPrefix(got[1], "PUT "+helper.path+` {"`+helper.key+`":{"name":"new"}}`) {
					t.Fatal(got)
				}
				if receipt.StatusCode != tc.reply.status || receipt.Header.Get("X-Openstack-Request-Id") != "req-put" || string(receipt.Envelope) != tc.reply.body || receipt.BodyIgnored != tc.ignored {
					t.Fatal(receipt)
				}
				if receipt.Merged.StatusCode != tc.reply.status || receipt.Merged.Header.Get("X-Openstack-Request-Id") != "req-put" {
					t.Fatal(receipt.Merged)
				}
				if tc.ignored != (receipt.Body == nil) || len(receipt.Body) != len(tc.wantBody) {
					t.Fatal(receipt.Body)
				}
				for k, v := range tc.wantBody {
					if rawText(receipt.Body[k]) != v {
						t.Fatal(k, receipt.Body)
					}
				}
				merged := receipt.Merged.Body
				if rawText(merged["id"]) != `"p"` || rawText(merged["status"]) != tc.wantStatus || rawText(merged["x_lookup_only"]) != `{"a":[1,null]}` || rawText(merged["x_new"]) != tc.wantUnknown {
					t.Fatal(merged)
				}
				if _, present := merged["self"]; present {
					t.Fatal("self member must be dropped", merged)
				}
				if name == nil || *name != tc.wantName || rawText(merged["name"]) != `"`+tc.wantName+`"` {
					t.Fatal(name, merged)
				}
			})
		}
	}
}

// TestCloudUpdateReceiptNonObjectBodiesFailAfterRemoteSuccess covers the JSON
// bodies on which Python raises after the PUT already succeeded. The receipt is
// retained, no merged view or native model is fabricated, and nothing replays.
func TestCloudUpdateReceiptNonObjectBodiesFailAfterRemoteSuccess(t *testing.T) {
	for _, helper := range updateReceiptHelpers() {
		for _, body := range []string{`[]`, `["` + helper.key + `"]`, `"text"`, `7`, `true`, `null`, `{"` + helper.key + `":null}`, `{"` + helper.key + `":[]}`, `{"` + helper.key + `":"x"}`} {
			t.Run(helper.name+"/"+body, func(t *testing.T) {
				client, calls := receiptClient(t, helper.path, helper.key, `{"id":"p","name":"old"}`, receiptReply{200, "application/json", body})
				receipt, name, _, err := helper.call(context.Background(), network.New(client))
				var responseErr *resource.ResponseError
				if err == nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &responseErr) || responseErr.StatusCode != 200 {
					t.Fatal(err)
				}
				if receipt == nil || receipt.Merged != nil || receipt.Body != nil || string(receipt.Envelope) != body || name != nil || len(calls()) != 2 {
					t.Fatal(receipt, name, calls())
				}
			})
		}
	}
}

// TestCloudUpdateReceiptRejectedPutHasNoReceipt keeps the lookup evidence but
// does not synthesize a receipt or a native model for a rejected PUT.
func TestCloudUpdateReceiptRejectedPutHasNoReceipt(t *testing.T) {
	for _, helper := range updateReceiptHelpers() {
		for _, status := range []int{http.StatusBadRequest, http.StatusConflict, http.StatusPreconditionFailed} {
			t.Run(helper.name+"/"+http.StatusText(status), func(t *testing.T) {
				client, calls := receiptClient(t, helper.path, helper.key, `{"id":"p","name":"old"}`, receiptReply{status, "application/json", `{"NeutronError":{}}`})
				receipt, name, _, err := helper.call(context.Background(), network.New(client))
				var unexpected gophercloud.ErrUnexpectedResponseCode
				if err == nil || !errors.As(err, &unexpected) || unexpected.Actual != status || receipt != nil || name != nil || len(calls()) != 2 {
					t.Fatal(receipt, name, err, calls())
				}
			})
		}
	}
}

// TestCloudUpdateReceiptProjectionKeepsMergedAuthority shows that a merged
// value the native schema cannot hold is reported without failing the
// accepted update, and the remaining members still project.
func TestCloudUpdateReceiptProjectionKeepsMergedAuthority(t *testing.T) {
	client, calls := receiptClient(t, "/v2.0/subnets/p", "subnet", `{"id":"p","name":"old","enable_dhcp":true}`, receiptReply{200, "application/json", `{"subnet":{"name":"resp"}}`})
	result, err := network.New(client).UpdateSubnetRecord(context.Background(), "p", network.WithSubnetUpdateJSON(json.RawMessage(`{"enable_dhcp":"false"}`)))
	if err != nil || result == nil || result.Response == nil || result.Response.ProjectionError == nil || !errors.Is(result.Response.ProjectionError, resource.ErrInvalidOption) || !strings.Contains(result.Response.ProjectionError.Error(), "enable_dhcp") {
		t.Fatal(result, err)
	}
	if result.Subnet == nil || result.Subnet.Name != "resp" || result.Subnet.EnableDHCP || rawText(result.Response.Merged.Body["enable_dhcp"]) != `"false"` || len(calls()) != 2 {
		t.Fatal(result.Subnet, result.Response.Merged.Body, calls())
	}
}
