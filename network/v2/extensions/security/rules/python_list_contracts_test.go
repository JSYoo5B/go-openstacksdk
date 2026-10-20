package rules_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const securityRecordListPath = "/neutron/v2.0/security-group-rules"

func securityRecordListWire(status int, body string) *http.Response {
	r := nativeRuleWire(status, body)
	r.Header.Set("X-Request-Id", "actual-security-list")
	return r
}
func securityRecordListAPI(t *testing.T, reply func(*http.Request) *http.Response) (*rules.API, *[]nativeRuleCall) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := new([]nativeRuleCall)
	nativeRuleRecorder(cloud, calls, func(r *http.Request) *http.Response {
		if r.Method != "GET" || r.Body != nil || r.Header.Get("X-Auth-Token") == "" {
			t.Errorf("unexpected request %s %s %v", r.Method, r.URL, r.Header)
		}
		return reply(r)
	})
	return rules.New(nativeRuleClient(cloud)), calls
}
func securityRecordListField(t *testing.T, v *rules.SecurityGroupRuleRecord, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		t.Fatalf("%s want=%s record=%+v", key, want, v)
	}
}
func securityRecordListProof(t *testing.T, err error, status int, body string) {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "actual-security-list" {
		t.Fatal(err, proof)
	}
}
func collectSecurityRecordList(seq iter.Seq2[*rules.SecurityGroupRuleRecord, error]) ([]*rules.SecurityGroupRuleRecord, error) {
	var values []*rules.SecurityGroupRuleRecord
	for v, err := range seq {
		if err != nil {
			return values, err
		}
		values = append(values, v)
	}
	return values, nil
}

func TestPythonSecurityRuleListSourceDescriptorsHeadersAndOwnedEvidence(t *testing.T) {
	raw := `{"security_group_rules":[{"id":"one","ethertype":"IPv4","ether_type":"IPv6","project_id":null,"tenant_id":"tenant","port_range_min":false,"port_range_max":-3.9,"revision_number":"١٢","tags":{"a":false,"b":0},"if-match":"row","if_match":false,"IF_MATCH":"unknown-body","location":{"project":{"id":"foreign"}},"self":"wire-only","x_new":900719925474099312345},{"tenant_id":"legacy"},{}]}`
	api, calls := securityRecordListAPI(t, func(r *http.Request) *http.Response {
		if r.URL.Path != securityRecordListPath || r.URL.RawQuery != "" || r.Header.Get("Accept") != "application/json" {
			t.Fatal(r.URL, r.Header)
		}
		response := securityRecordListWire(203, raw)
		response.Header.Set("If-Match", "whole-page")
		return response
	})
	seq := api.ListRecords(context.Background())
	if len(*calls) != 0 {
		t.Fatal("not lazy")
	}
	values, err := collectSecurityRecordList(seq)
	if err != nil || len(values) != 3 || len(*calls) != 1 {
		t.Fatal(err, values, *calls)
	}
	for key, want := range map[string]string{"ether_type": `"IPv6"`, "project_id": "null", "tenant_id": `"tenant"`, "port_range_min": "false", "port_range_max": "-3", "revision_number": "12", "tags": `[{"a":false,"b":0}]`, "if_match": "[false]", "IF_MATCH": `"unknown-body"`, "location": "null", "x_new": "900719925474099312345"} {
		securityRecordListField(t, values[0], key, want)
	}
	if values[0].Resource.Body["self"] != nil || values[0].Resource.Body["ethertype"] != nil || string(values[0].Wire.Body["ethertype"]) != `"IPv4"` || string(values[0].Wire.Body["port_range_max"]) != "-3.9" || string(values[0].Envelope) != raw || values[0].StatusCode != 203 || values[0].Header.Get("If-Match") != "whole-page" {
		t.Fatal(values[0])
	}
	securityRecordListField(t, values[1], "project_id", `"legacy"`)
	securityRecordListField(t, values[1], "if_match", "null")
	securityRecordListField(t, values[2], "tags", "[]")
	if len(values[2].Resource.Body) != 20 {
		t.Fatal(values[2].Resource.Body)
	}
	clone := values[0].Resource.Clone()
	clone.Body["location"][0] = 'X'
	values[0].Resource.Body["ether_type"][1] = 'X'
	values[0].Wire.Body["location"][0] = 'X'
	values[0].Header.Set("If-Match", "consumer")
	if string(values[0].Resource.Body["location"]) != "null" || string(values[0].Envelope) != raw || string(values[0].Wire.Body["ether_type"]) != `"IPv6"` || values[0].Resource.Header.Get("If-Match") != "whole-page" || values[0].Wire.Header.Get("If-Match") != "whole-page" || string(values[1].Resource.Body["tags"]) != "[]" {
		t.Fatal("owned carriers alias", values)
	}
}
func TestPythonSecurityRuleListServerAliasesLocalFieldsAndUnknownDiscard(t *testing.T) {
	var query url.Values
	api, calls := securityRecordListAPI(t, func(r *http.Request) *http.Response {
		query = r.URL.Query()
		return securityRecordListWire(200, `{"security_group_rules":[{"id":"one","name":"wanted","created_at":{"enabled":1},"direction":"server-unchecked"},{"id":"two","name":"other","created_at":{"enabled":1}}]}`)
	})
	values, err := collectSecurityRecordList(api.ListRecords(context.Background(), rules.WithRecordListQuery("id", []any{"one", 0, false, nil}), rules.WithRecordListQuery("ether_type", "IPv6"), rules.WithRecordListQuery("ethertype", "IPv4"), rules.WithRecordListQuery("any_tags", []string{"red", "blue"}), rules.WithRecordListQuery("tags-any", "wrong"), rules.WithRecordListQuery("fields", []string{"id", "name"}), rules.WithRecordListQuery("project_id", "project"), rules.WithRecordListQuery("tenant_id", "tenant"), rules.WithRecordListQuery("direction", "ingress"), rules.WithRecordListQuery("name", "wanted"), rules.WithRecordListQuery("created_at", map[string]any{"enabled": true}), rules.WithRecordListQuery("x_unknown", make(chan int)), rules.WithRecordListQuery("if_match", make(chan int)), rules.WithRecordListQuery("all_projects", make(chan int))))
	want := url.Values{"id": {"one", "0", "False"}, "ethertype": {"IPv6"}, "tags-any": {"red", "blue"}, "fields": {"id", "name"}, "project_id": {"project"}, "tenant_id": {"tenant"}, "direction": {"ingress"}}
	if err != nil || len(values) != 1 || len(*calls) != 1 || !reflect.DeepEqual(query, want) {
		t.Fatal(err, values, *calls, query)
	}
	securityRecordListField(t, values[0], "direction", `"server-unchecked"`)
}
func TestPythonSecurityRuleListOrderedDictionaryFiltersAndExactEquality(t *testing.T) {
	for _, mode := range []string{"false first", "mapping fault first", "empty dict", "float underflow", "exact equality"} {
		t.Run(mode, func(t *testing.T) {
			raw := `{"security_group_rules":[{"id":"one","name":"truthy","created_at":9007199254740992}]}`
			if mode == "float underflow" {
				raw = `{"security_group_rules":[{"id":"one","created_at":1e-400}]}`
			}
			api, calls := securityRecordListAPI(t, func(*http.Request) *http.Response { return securityRecordListWire(200, raw) })
			var options []rules.RecordListOption
			want := 0
			bad := false
			switch mode {
			case "false first":
				options = []rules.RecordListOption{rules.WithRecordListQuery("name", "no"), rules.WithRecordListQuery("created_at", map[string]any{"x": true})}
			case "mapping fault first":
				options = []rules.RecordListOption{rules.WithRecordListQuery("created_at", map[string]any{"x": true}), rules.WithRecordListQuery("name", "no")}
				bad = true
			case "empty dict":
				options = []rules.RecordListOption{rules.WithRecordListQuery("name", map[string]any{})}
				want = 1
			case "float underflow":
				options = []rules.RecordListOption{rules.WithRecordListQuery("created_at", map[string]any{})}
			case "exact equality":
				options = []rules.RecordListOption{rules.WithRecordListQuery("created_at", json.RawMessage("9007199254740993.0"))}
			}
			values, err := collectSecurityRecordList(api.ListRecords(context.Background(), options...))
			if (err != nil) != bad || len(values) != want || len(*calls) != 1 {
				t.Fatal(err, values, *calls)
			}
			if bad {
				securityRecordListProof(t, err, 200, raw)
			}
		})
	}
}
func TestPythonSecurityRuleListRawCapAndBreakAvoidUnconsumedWork(t *testing.T) {
	raw := `{"security_group_rules":[{"id":"skip","name":"other"},{"id":"one","name":"wanted"},{"cls":null,"revision_number":"²"}],"next":"https://foreign.invalid/"}`
	api, calls := securityRecordListAPI(t, func(r *http.Request) *http.Response {
		if r.URL.Query().Get("limit") != "2" {
			t.Fatal(r.URL)
		}
		return securityRecordListWire(200, raw)
	})
	values, err := api.AllRecords(context.Background(), rules.WithRecordListQuery("name", "wanted"), rules.WithRecordListLimit(0), rules.WithRecordListMaxItems(2))
	if err != nil || len(values) != 1 || len(*calls) != 1 {
		t.Fatal(err, values, *calls)
	}
	securityRecordListField(t, values[0], "id", `"one"`)
	api, calls = securityRecordListAPI(t, func(*http.Request) *http.Response { return securityRecordListWire(200, raw) })
	values, err = api.AllRecords(context.Background(), rules.WithRecordListQuery("name", "wanted"), rules.WithRecordListMaxItems(1))
	if err != nil || values == nil || len(values) != 0 || len(*calls) != 1 {
		t.Fatal(err, values, *calls)
	}
	api, calls = securityRecordListAPI(t, func(*http.Request) *http.Response { return securityRecordListWire(200, raw) })
	for _, err := range api.ListRecords(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if len(*calls) != 1 {
		t.Fatal(*calls)
	}
}
func TestPythonSecurityRuleListPagesEnvelopeAndConstructorProof(t *testing.T) {
	modes := []string{"body next", "plural links", "http link", "versioned", "short page", "single page", "singleton", "empty", "falsey next", "cycle", "foreign", "filter change", "wrong row", "null envelope", "missing envelope", "malformed", "blank204", "cls false", "connection null", "microversion false", "synchronized zero", "digit error"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			n := 0
			last := `{"security_group_rules":[]}`
			status := 200
			api, calls := securityRecordListAPI(t, func(r *http.Request) *http.Response {
				n++
				last = `{"security_group_rules":[]}`
				status = 200
				if n == 1 {
					switch mode {
					case "body next", "single page":
						last = `{"security_group_rules":[{"id":"one"}],"next":"?marker=one"}`
					case "plural links":
						last = `{"security_group_rules":[{"id":"one"}],"security_group_rules_links":[{"rel":"next","href":"?marker=one"}]}`
					case "http link", "short page":
						last = `{"security_group_rules":[{"id":"one"}]}`
					case "versioned":
						last = `{"security_group_rules":[{"id":"one"}],"next":"/v2.0/security-group-rules?marker=one"}`
					case "singleton":
						last = `{"security_group_rules":{"id":"one"}}`
					case "empty":
						last = `{"security_group_rules":[],"next":"https://foreign.invalid/"}`
					case "falsey next":
						last = `{"security_group_rules":[{"id":"one"}],"next":1e-400}`
					case "cycle":
						last = `{"security_group_rules":[{"id":"one"}],"next":"?marker=start"}`
					case "foreign":
						last = `{"security_group_rules":[{"id":"one"}],"next":"/v2.0/networks?marker=one"}`
					case "filter change":
						last = `{"security_group_rules":[{"id":"one"}],"next":"?marker=one&direction=egress"}`
					case "wrong row":
						last = `{"security_group_rules":[false]}`
					case "null envelope":
						last = `{"security_group_rules":null}`
					case "missing envelope":
						last = `{}`
					case "malformed":
						last = `{"security_group_rules":[]`
					case "blank204":
						last = ""
						status = 204
					case "cls false":
						last = `{"security_group_rules":[{"cls":false}]}`
					case "connection null":
						last = `{"security_group_rules":[{"connection":null}]}`
					case "microversion false":
						last = `{"security_group_rules":[{"microversion":false}]}`
					case "synchronized zero":
						last = `{"security_group_rules":[{"_synchronized":0}]}`
					case "digit error":
						last = `{"security_group_rules":[{"revision_number":"²"}]}`
					}
				} else if r.URL.Query().Get("marker") != "one" {
					t.Fatal(r.URL)
				}
				response := securityRecordListWire(status, last)
				if mode == "http link" && n == 1 {
					response.Header.Set("Link", "<?marker=one>; rel=\"next\"")
				}
				return response
			})
			options := []rules.RecordListOption{rules.WithRecordListMarker("start"), rules.WithRecordListQuery("direction", "ingress")}
			if mode == "short page" {
				options = append(options, rules.WithRecordListLimit(5))
			}
			if mode == "single page" {
				options = append(options, rules.WithRecordListPaginated(false))
			}
			values, err := collectSecurityRecordList(api.ListRecords(context.Background(), options...))
			success := mode == "body next" || mode == "plural links" || mode == "http link" || mode == "versioned" || mode == "short page" || mode == "single page" || mode == "singleton" || mode == "empty" || mode == "falsey next"
			if (err == nil) != success {
				t.Fatal(err, values, *calls)
			}
			if success {
				want := 1
				if mode == "empty" {
					want = 0
				}
				if len(values) != want {
					t.Fatal(values)
				}
				count := 1
				if mode == "body next" || mode == "plural links" || mode == "http link" || mode == "versioned" || mode == "short page" {
					count = 2
				}
				if len(*calls) != count {
					t.Fatal(*calls)
				}
			} else {
				if len(*calls) != 1 {
					t.Fatal(*calls)
				}
				securityRecordListProof(t, err, status, last)
			}
		})
	}
}
func TestPythonSecurityRuleListStatusPoliciesAndLateDenial(t *testing.T) {
	for _, status := range []int{199, 200, 203, 204, 299, 300, 399, 400, 403, 404, 409, 429, 500, 503, 599} {
		raw := `{"security_group_rules":[{"id":"one"}]}`
		api, calls := securityRecordListAPI(t, func(*http.Request) *http.Response { return securityRecordListWire(status, raw) })
		values, err := api.AllRecords(context.Background(), rules.WithRecordListQuery("ignore_missing", true))
		if status >= 200 && status < 400 {
			if err != nil || len(values) != 1 || values[0].StatusCode != status {
				t.Fatal(err, values)
			}
		} else {
			var native gophercloud.ErrUnexpectedResponseCode
			if values != nil || !errors.As(err, &native) || native.Actual != status || string(native.Body) != raw || native.ResponseHeader.Get("X-Request-Id") != "actual-security-list" {
				t.Fatal(err, values, native)
			}
			nativeRuleOperation(t, err, "ListRecords")
		}
		if len(*calls) != 1 {
			t.Fatal(*calls)
		}
	}
	n := 0
	api, calls := securityRecordListAPI(t, func(*http.Request) *http.Response {
		n++
		if n == 1 {
			return securityRecordListWire(200, `{"security_group_rules":[{"id":"one"}],"next":"?marker=one"}`)
		}
		return securityRecordListWire(403, `{"denied":"page two"}`)
	})
	values, err := api.AllRecords(context.Background())
	if values != nil || !gophercloud.ResponseCodeIs(err, 403) || len(*calls) != 2 {
		t.Fatal(err, values, *calls)
	}
}

type securityListFaultBody struct {
	io.Reader
	closed     atomic.Int32
	closeCause error
	onClose    func()
}

func (b *securityListFaultBody) Close() error {
	b.closed.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeCause
}

type securityListFaultReader func([]byte) (int, error)

func (f securityListFaultReader) Read(p []byte) (int, error) { return f(p) }
func TestPythonSecurityRuleListPhysicalFaultsStopRetryCallbacks(t *testing.T) {
	for _, status := range []int{200, 203, 400, 401, 403, 404, 409, 500, 503, 599} {
		for _, fault := range []string{"read", "close", "mixed", "source read restored close", "cancel read", "cancel close", "all mixed"} {
			t.Run(fmt.Sprintf("%d/%s", status, fault), func(t *testing.T) {
				raw := `{"security_group_rules":[{"id":"one"}]}`
				readCause, closeCause, cancelCause, callbackCause := errors.New("read cause"), errors.New("close cause"), errors.New("cancel cause"), errors.New("callback cause")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				body := &securityListFaultBody{Reader: strings.NewReader(raw)}
				var api *rules.API
				original := ""
				done := false
				if fault == "read" || fault == "mixed" {
					body.Reader = securityListFaultReader(func(p []byte) (int, error) {
						if done {
							return 0, io.EOF
						}
						done = true
						return copy(p, raw), readCause
					})
				}
				if fault == "close" || fault == "mixed" {
					body.closeCause = closeCause
				}
				if fault == "source read restored close" {
					body.Reader = securityListFaultReader(func(p []byte) (int, error) {
						if done {
							return 0, io.EOF
						}
						done = true
						api.RawClient().ResourceBase = "https://foreign.invalid/v2/"
						return copy(p, raw), nil
					})
					body.onClose = func() { api.RawClient().ResourceBase = original }
				}
				if fault == "cancel read" {
					body.Reader = securityListFaultReader(func(p []byte) (int, error) {
						if done {
							return 0, io.EOF
						}
						done = true
						cancel(cancelCause)
						return copy(p, raw), nil
					})
				}
				if fault == "cancel close" {
					body.onClose = func() { cancel(cancelCause) }
				}
				if fault == "all mixed" {
					body.Reader = securityListFaultReader(func(p []byte) (int, error) {
						if done {
							return 0, io.EOF
						}
						done = true
						api.RawClient().ResourceBase = "https://foreign.invalid/v2/"
						return copy(p, raw), readCause
					})
					body.closeCause = closeCause
					body.onClose = func() { api.RawClient().ResourceBase = original; cancel(cancelCause) }
				}
				var calls *[]nativeRuleCall
				api, calls = securityRecordListAPI(t, func(*http.Request) *http.Response { r := securityRecordListWire(status, raw); r.Body = body; return r })
				original = api.RawClient().ResourceBase
				retries, reauth := 0, 0
				api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return callbackCause
				}
				api.RawClient().ReauthFunc = func(context.Context) error { reauth++; return nil }
				originalURL := api.RawClient().ServiceURL("security-group-rules")
				values, err := api.AllRecords(ctx)
				if values != nil || err == nil || len(*calls) != 1 || body.closed.Load() != 1 || retries != 0 || reauth != 0 || errors.Is(err, callbackCause) {
					t.Fatal(err, values, *calls, body.closed.Load(), retries, reauth)
				}
				assertFault := func(cause error) {
					t.Helper()
					if (fault == "read" || fault == "mixed" || fault == "all mixed") && !errors.Is(cause, readCause) || (fault == "close" || fault == "mixed" || fault == "all mixed") && !errors.Is(cause, closeCause) || (fault == "source read restored close" || fault == "all mixed") && !errors.Is(cause, resource.ErrInvalidOption) || (strings.HasPrefix(fault, "cancel") || fault == "all mixed") && (!errors.Is(cause, cancelCause) || !errors.Is(cause, context.Canceled)) {
						t.Fatal(cause)
					}
				}
				assertFault(err)
				if status < 400 {
					securityRecordListProof(t, err, status, raw)
				} else {
					receiptErr := err
					if status == 401 {
						// The native pointer intentionally keeps two independent errors in public
						// fields rather than Unwrap. Inspect each original receipt and guard cause.
						var auth *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &auth) || auth.ErrOriginal == nil || auth.ErrReauth == nil {
							t.Fatal(err, auth)
						}
						assertFault(auth.ErrReauth)
						receiptErr = auth.ErrOriginal
					}
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(receiptErr, &native) || native.Actual != status || native.Method != http.MethodGet || native.URL != originalURL || string(native.Body) != raw || native.ResponseHeader.Get("X-Request-Id") != "actual-security-list" {
						t.Fatal(err, native)
					}
				}
			})
		}
	}
}
func TestPythonSecurityRuleListVersionsHeadersRetryAndReauthentication(t *testing.T) {
	for _, mode := range []string{"default", "explicit", "empty", "live retry", "reauth", "expanded codes"} {
		t.Run(mode, func(t *testing.T) {
			var versions, tokens, accepts []string
			n := 0
			api, calls := securityRecordListAPI(t, func(r *http.Request) *http.Response {
				n++
				versions = append(versions, r.Header.Get("OpenStack-API-Version"))
				tokens = append(tokens, r.Header.Get("X-Auth-Token"))
				accepts = append(accepts, r.Header.Get("Accept"))
				if n == 1 && (mode == "live retry" || mode == "expanded codes") {
					return securityRecordListWire(503, `{"retry":true}`)
				}
				if mode == "reauth" && n == 1 {
					return securityRecordListWire(401, `{"expired":true}`)
				}
				if mode == "expanded codes" {
					return securityRecordListWire(400, `{"outside":"original"}`)
				}
				return securityRecordListWire(200, `{"security_group_rules":[]}`)
			})
			api.RawClient().Microversion = "2.0"
			api.RawClient().MoreHeaders = map[string]string{"Accept": "captured"}
			var options []rules.RecordListOption
			want := "network 2.0"
			if mode == "explicit" {
				want = "network 2.1"
				options = append(options, rules.WithRecordListMicroversion("2.1"), rules.WithRecordListHeaders(map[string]string{"aCcEpT": "owned"}))
			}
			if mode == "empty" {
				want = ""
				options = append(options, rules.WithRecordListMicroversion(""))
			}
			if mode == "live retry" || mode == "expanded codes" {
				api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
					o.MoreHeaders["X-Retry-Probe"] = "ordinary"
					if mode == "expanded codes" {
						o.OkCodes = append(o.OkCodes, 400)
					}
					api.RawClient().SetToken("renewed")
					return nil
				}
			}
			if mode == "reauth" {
				api.RawClient().ReauthFunc = func(context.Context) error { api.RawClient().SetToken("renewed"); return nil }
			}
			values, err := api.AllRecords(context.Background(), options...)
			count := 1
			if mode == "live retry" || mode == "reauth" || mode == "expanded codes" {
				count = 2
			}
			if mode == "expanded codes" {
				if values != nil || !gophercloud.ResponseCodeIs(err, 400) {
					t.Fatal(err, values)
				}
			} else if err != nil || values == nil || len(values) != 0 {
				t.Fatal(err, values)
			}
			if len(*calls) != count || api.RawClient().Microversion != "2.0" || api.RawClient().MoreHeaders["Accept"] != "captured" {
				t.Fatal(*calls, api.RawClient())
			}
			for _, actual := range versions {
				if actual != want {
					t.Fatal(versions, want)
				}
			}
			expectedAccept := "application/json"
			if mode == "explicit" {
				expectedAccept = "owned"
			}
			for _, actual := range accepts {
				if actual != expectedAccept {
					t.Fatal(accepts)
				}
			}
			if count == 2 && !reflect.DeepEqual(tokens, []string{"test-token", "renewed"}) {
				t.Fatal(tokens)
			}
		})
	}
	for _, key := range []string{"Authorization", "X-Service-Token", "Content-Length", "X-OpenStack-Nova-API-Version", "X-OpenStack-Volume-API-Version", "OpenStack-API-Version", "omit-version", "JSONBody"} {
		t.Run(key, func(t *testing.T) {
			raw := `{"retry":"actual"}`
			body := &securityListFaultBody{Reader: strings.NewReader(raw)}
			api, calls := securityRecordListAPI(t, func(*http.Request) *http.Response { r := securityRecordListWire(503, raw); r.Body = body; return r })
			api.RawClient().Microversion = "2.0"
			cause := errors.New("retry callback")
			api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				switch key {
				case "JSONBody":
					o.JSONBody = map[string]any{"body": true}
				case "omit-version":
					o.OmitHeaders = []string{"OpenStack-API-Version"}
				default:
					o.MoreHeaders[key] = "foreign"
				}
				return cause
			}
			values, err := api.AllRecords(context.Background())
			var native gophercloud.ErrUnexpectedResponseCode
			if values != nil || !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != raw || len(*calls) != 1 || body.closed.Load() != 1 {
				t.Fatal(err, values, native, *calls, body.closed.Load())
			}
		})
	}
}
func TestPythonSecurityRuleListPreflightAndOwnedOptionSnapshots(t *testing.T) {
	api, calls := securityRecordListAPI(t, func(*http.Request) *http.Response { t.Fatal("unexpected preflight HTTP"); return nil })
	for _, key := range []string{"cls", "connection", "microversion", "location", "headers", "base_path", "session", "__conflicting_attrs", "bad\n", " "} {
		_, err := api.AllRecords(context.Background(), rules.WithRecordListQuery(key, nil))
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(key, err)
		}
	}
	for _, opts := range [][]rules.RecordListOption{{nil}, {rules.WithRecordListMaxItems(-1)}, {rules.WithRecordListLimit(-1)}, {rules.WithRecordListHeaders(map[string]string{"Authorization": "foreign"})}, {rules.WithRecordListQuery("name", json.RawMessage(`broken`))}} {
		_, err := api.AllRecords(context.Background(), opts...)
		if err == nil {
			t.Fatal(opts)
		}
	}
	_, err := api.AllRecords(nil)
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	_, err = (*rules.API)(nil).AllRecords(context.Background())
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	cause := errors.New("canceled option")
	ctx, cancel := context.WithCancelCause(context.Background())
	_, err = api.AllRecords(ctx, func(*rules.RecordListOpts) error { cancel(cause); return nil })
	if !errors.Is(err, cause) || len(*calls) != 0 {
		t.Fatal(err, *calls)
	}
	var proof *resource.ResponseError
	if errors.As(err, &proof) {
		t.Fatal("invented receipt", proof)
	}
	raw := json.RawMessage(`"original"`)
	headers := map[string]string{"X-Probe": "original"}
	version := "2.1"
	cfg := rules.RecordListOpts{Query: map[string]json.RawMessage{"name": raw}, Headers: headers, Microversion: &version}
	owned := rules.WithRecordListOpts(cfg)
	raw[1] = 'X'
	headers["X-Probe"] = "changed"
	version = "bad\n"
	var retained *rules.RecordListOpts
	api, calls = securityRecordListAPI(t, func(r *http.Request) *http.Response {
		retained.Query["name"][1] = 'Y'
		retained.Headers["X-Probe"] = "changed"
		*retained.Microversion = "bad\n"
		if r.Header.Get("X-Probe") != "original" || r.Header.Get("OpenStack-API-Version") != "network 2.1" || r.URL.RawQuery != "" {
			t.Fatal(r.URL, r.Header)
		}
		return securityRecordListWire(200, `{"security_group_rules":[{"name":"original"}]}`)
	})
	seq := api.ListRecords(context.Background(), owned, func(v *rules.RecordListOpts) error { retained = v; return nil })
	if len(*calls) != 0 {
		t.Fatal("not lazy")
	}
	values, err := collectSecurityRecordList(seq)
	if err != nil || len(values) != 1 || len(*calls) != 1 {
		t.Fatal(err, values, *calls)
	}
}
func TestPythonSecurityRuleListPublicFixtureConcurrentReuse(t *testing.T) {
	cloud := testcloud.New(t)
	var count atomic.Int32
	cloud.Mux.HandleFunc("/neutron/", func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		if r.Method != "GET" || r.URL.Path != securityRecordListPath || r.URL.Query().Get("security_group_id") != "group /%?#" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Probe") != "owned" {
			t.Error(r.Method, r.RequestURI, r.Header)
		}
		testcloud.JSON(w, 200, `{"security_group_rules":[{"id":"one","port_range_min":0}]}`)
	})
	api := rules.New(nativeRuleClient(cloud))
	headers := map[string]string{"X-Probe": "owned"}
	options := []rules.RecordListOption{rules.WithRecordListHeaders(headers), rules.WithRecordListQuery("security_group_id", "group /%?#")}
	headers["X-Probe"] = "changed"
	var wait sync.WaitGroup
	for i := 0; i < 8; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			values, err := api.AllRecords(context.Background(), options...)
			if err != nil || len(values) != 1 {
				t.Error(err, values)
			}
		}()
	}
	wait.Wait()
	if count.Load() != 8 || api.RawClient().MoreHeaders != nil {
		t.Fatal(count.Load(), api.RawClient())
	}
}

func TestPythonSecurityRuleListConditionalHeadersRemainSelected(t *testing.T) {
	for _, condition := range []string{"If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since"} {
		for _, mode := range []string{"absent injection", "selected retained", "selected replaced", "selected omitted"} {
			t.Run(condition+"/"+mode, func(t *testing.T) {
				n := 0
				raw := `{"rejection":"conditional"}`
				api, calls := securityRecordListAPI(t, func(r *http.Request) *http.Response {
					n++
					want := "selected"
					if mode == "absent injection" {
						want = ""
					}
					if r.Header.Get(condition) != want {
						t.Fatal(condition, r.Header)
					}
					if n == 1 {
						return securityRecordListWire(503, raw)
					}
					return securityRecordListWire(200, `{"security_group_rules":[]}`)
				})
				var options []rules.RecordListOption
				if mode != "absent injection" {
					options = append(options, rules.WithRecordListHeaders(map[string]string{strings.ToLower(condition): "selected"}))
				}
				cause := errors.New("conditional callback cause")
				api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
					if mode == "selected retained" {
						opts.MoreHeaders["X-Retry-Probe"] = "ordinary"
						return nil
					}
					if mode == "selected omitted" {
						opts.OmitHeaders = []string{strings.ToUpper(condition)}
					} else {
						for key := range opts.MoreHeaders {
							if strings.EqualFold(key, condition) {
								delete(opts.MoreHeaders, key)
							}
						}
						opts.MoreHeaders[strings.ToUpper(condition)] = "foreign"
					}
					return cause
				}
				values, err := api.AllRecords(context.Background(), options...)
				if mode == "selected retained" {
					if err != nil || values == nil || len(values) != 0 || len(*calls) != 2 {
						t.Fatal(err, values, *calls)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if values != nil || !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != raw || native.ResponseHeader.Get("X-Request-Id") != "actual-security-list" || len(*calls) != 1 {
						t.Fatal(err, values, native, *calls)
					}
				}
				if api.RawClient().MoreHeaders != nil {
					t.Fatal("original client Header mutated", api.RawClient())
				}
			})
		}
	}
}

func TestPythonSecurityRuleListCustomCallbacksReceiveOwnedMaps(t *testing.T) {
	for _, mode := range []string{"first callback", "nil reset", "empty opts factory"} {
		t.Run(mode, func(t *testing.T) {
			var retained *rules.RecordListOpts
			api, calls := securityRecordListAPI(t, func(r *http.Request) *http.Response {
				if r.URL.RawQuery != "" || r.Header.Get("X-Probe") != "owned" {
					t.Fatal(r.URL, r.Header)
				}
				retained.Query["name"][1] = 'X'
				retained.Headers["X-Probe"] = "consumer"
				return securityRecordListWire(200, `{"security_group_rules":[{"name":"owned"}]}`)
			})
			var options []rules.RecordListOption
			switch mode {
			case "nil reset":
				options = append(options, func(o *rules.RecordListOpts) error { o.Query = nil; o.Headers = nil; return nil })
			case "empty opts factory":
				options = append(options, rules.WithRecordListOpts(rules.RecordListOpts{}))
			}
			options = append(options, func(o *rules.RecordListOpts) error {
				if o.Query == nil || o.Headers == nil {
					t.Fatal("SDK option maps were not initialized", mode, o)
				}
				o.Query["name"] = json.RawMessage(`"owned"`)
				o.Headers["X-Probe"] = "owned"
				retained = o
				return nil
			})
			seq := api.ListRecords(context.Background(), options...)
			if retained != nil || len(*calls) != 0 {
				t.Fatal("callback evaluated eagerly", retained, *calls)
			}
			values, err := collectSecurityRecordList(seq)
			if err != nil || len(values) != 1 || len(*calls) != 1 || api.RawClient().MoreHeaders != nil {
				t.Fatal(err, values, *calls, api.RawClient())
			}
			securityRecordListField(t, values[0], "name", `"owned"`)
		})
	}
}

type securityListTransformMarshaler struct{ called *atomic.Int32 }

func (v securityListTransformMarshaler) MarshalJSON() ([]byte, error) {
	v.called.Add(1)
	return nil, errors.New("unsupported transform value must not be marshaled")
}
func TestPythonSecurityRuleListJMESPathTransformPresenceIsExplicitlyUnsupported(t *testing.T) {
	var marshaled atomic.Int32
	values := []any{nil, false, "", "[?direction == 'ingress'].id", make(chan int), json.RawMessage(`malformed ignored value`), securityListTransformMarshaler{called: &marshaled}}
	for i, value := range values {
		t.Run(fmt.Sprintf("factory/%d", i), func(t *testing.T) {
			api, calls := securityRecordListAPI(t, func(*http.Request) *http.Response { t.Fatal("unsupported transform reached HTTP"); return nil })
			option := rules.WithRecordListQuery("jmespath_filters", value)
			seq := api.ListRecords(context.Background(), option)
			if len(*calls) != 0 || marshaled.Load() != 0 {
				t.Fatal("eager unsupported transform evaluation", *calls, marshaled.Load())
			}
			rows, err := collectSecurityRecordList(seq)
			var proof *resource.ResponseError
			if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(err.Error(), "jmespath_filters") || errors.As(err, &proof) || len(*calls) != 0 || marshaled.Load() != 0 {
				t.Fatal(rows, err, proof, *calls, marshaled.Load())
			}
		})
	}
	for _, raw := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(`false`), json.RawMessage(`"[].id"`), json.RawMessage(`malformed`)} {
		for _, mode := range []string{"DTO", "custom callback"} {
			t.Run(mode+"/"+string(raw), func(t *testing.T) {
				api, calls := securityRecordListAPI(t, func(*http.Request) *http.Response { t.Fatal("unsupported owned transform reached HTTP"); return nil })
				option := rules.WithRecordListOpts(rules.RecordListOpts{Query: map[string]json.RawMessage{"jmespath_filters": raw}})
				if mode == "custom callback" {
					option = func(o *rules.RecordListOpts) error { o.Query["jmespath_filters"] = raw; return nil }
				}
				rows, err := api.AllRecords(context.Background(), option)
				var proof *resource.ResponseError
				if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(err.Error(), "jmespath_filters") || errors.As(err, &proof) || len(*calls) != 0 {
					t.Fatal(rows, err, proof, *calls)
				}
			})
		}
	}
}
