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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const dscpRecordPath = "/neutron/v2.0/qos/policies/policy/dscp_marking_rules"

func dscpRecordWire(status int, body string) *http.Response {
	r := nativeRuleWire(status, body)
	r.Header.Set("X-Request-Id", "actual-dscp")
	return r
}
func dscpRecordAPI(t *testing.T, reply func(*http.Request) *http.Response) (*rules.API, *[]nativeRuleCall) {
	t.Helper()
	calls := new([]nativeRuleCall)
	api, _ := nativeRuleAPI(t, calls, func(r *http.Request) *http.Response {
		if r.Method != "GET" || r.Body != nil || r.Header.Get("X-Auth-Token") == "" {
			t.Errorf("request=%s %s headers=%v body=%v", r.Method, r.URL, r.Header, r.Body)
		}
		return reply(r)
	})
	return api, calls
}
func dscpRecordField(t *testing.T, value *rules.DSCPMarkingRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatalf("record=%+v field=%s want=%s", value, key, want)
	}
}
func dscpRecordProof(t *testing.T, err error, status int, body string) {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "actual-dscp" {
		t.Fatalf("proof=%+v error=%v", proof, err)
	}
}
func collectDSCPRecords(seq iter.Seq2[*rules.DSCPMarkingRecord, error]) ([]*rules.DSCPMarkingRecord, error) {
	var values []*rules.DSCPMarkingRecord
	for value, err := range seq {
		if err != nil {
			return values, err
		}
		values = append(values, value)
	}
	return values, nil
}

func TestPythonDSCPMarkingGetLiteralPolicyRouteRawProjectionAndDefaults(t *testing.T) {
	parent, id := "policy /%2F?#", "rule /%2F?#"
	raw := `{"dscp_marking_rule":{"id":null,"name":false,"dscp_mark":false,"qos_policy_id":"server-foreign","x_new":900719925474099312345,"self":"wire-only"}}`
	api, calls := dscpRecordAPI(t, func(r *http.Request) *http.Response {
		want := "/neutron/v2.0/qos/policies/" + url.PathEscape(parent) + "/dscp_marking_rules/" + url.PathEscape(id)
		if r.URL.EscapedPath() != want || r.URL.RawQuery != "" {
			t.Fatal(r.URL, want)
		}
		response := dscpRecordWire(203, raw)
		response.Header.Set("If-Match", "account-observation")
		return response
	})
	value, err := api.GetDSCPMarkingRecord(context.Background(), parent, id)
	if err != nil || value == nil || value.PolicyID != parent || value.StatusCode != 203 || string(value.Envelope) != raw || len(*calls) != 1 {
		t.Fatal(err, value, *calls)
	}
	for key, want := range map[string]string{"id": "null", "name": "false", "dscp_mark": "false", "qos_policy_id": `"policy /%2F?#"`, "x_new": "900719925474099312345", "location": "null"} {
		dscpRecordField(t, value, key, want)
	}
	if value.Resource.Body["self"] != nil || value.Resource.Body["if_match"] != nil || value.Resource.Body["tags"] != nil || value.Resource.Body["revision_number"] != nil || string(value.Wire.Body["qos_policy_id"]) != `"server-foreign"` || string(value.Wire.Body["self"]) != `"wire-only"` {
		t.Fatal(value)
	}
	value.Resource.Body["qos_policy_id"][1] = 'X'
	value.Header.Set("X-Request-Id", "consumer")
	if value.PolicyID != parent || string(value.Wire.Body["qos_policy_id"]) != `"server-foreign"` || string(value.Envelope) != raw || value.Resource.Header.Get("X-Request-Id") != "actual-dscp" || value.Wire.Header.Get("X-Request-Id") != "actual-dscp" {
		t.Fatal("record carriers alias")
	}
}
func TestPythonDSCPMarkingGetPartialReceiptsWithoutQuerySeedsOrFallback(t *testing.T) {
	for _, body := range []string{"", "broken", `{}`, `{"dscp_marking_rule":{}}`} {
		t.Run(body, func(t *testing.T) {
			api, calls := dscpRecordAPI(t, func(r *http.Request) *http.Response {
				if r.URL.Query().Get("name") != "wire-only" || r.URL.Query().Get("dscp_mark") != "0" || r.URL.Query().Has("qos_policy_id") {
					t.Fatal(r.URL)
				}
				return dscpRecordWire(204, body)
			})
			value, err := api.GetDSCPMarkingRecord(context.Background(), "policy", "literal", rules.WithDSCPMarkingGetQuery("name", "wire-only"), rules.WithDSCPMarkingGetQuery("dscp_mark", 0))
			if err != nil || value == nil || value.PolicyID != "policy" || value.StatusCode != 204 || string(value.Envelope) != body || len(*calls) != 1 || len(value.Resource.Body) != 4 {
				t.Fatal(err, value, *calls)
			}
			dscpRecordField(t, value, "id", `"literal"`)
			dscpRecordField(t, value, "name", "null")
			dscpRecordField(t, value, "dscp_mark", "null")
			dscpRecordField(t, value, "location", "null")
		})
	}
	for _, body := range []string{`null`, `[]`, `{"dscp_marking_rule":null}`, `{"dscp_marking_rule":false}`, "{\"dscp_marking_rule\":{\"name\":\"\xff\"}}"} {
		api, calls := dscpRecordAPI(t, func(*http.Request) *http.Response { return dscpRecordWire(200, body) })
		_, err := api.GetDSCPMarkingRecord(context.Background(), "policy", "literal")
		if err == nil || len(*calls) != 1 {
			t.Fatal(err, *calls)
		}
		dscpRecordProof(t, err, 200, body)
	}
	for _, status := range []int{400, 403, 404, 409, 429, 500, 503, 599} {
		raw := `{"error":"member reject"}`
		api, calls := dscpRecordAPI(t, func(*http.Request) *http.Response { return dscpRecordWire(status, raw) })
		value, err := api.GetDSCPMarkingRecord(context.Background(), "policy", "literal")
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || !errors.As(err, &native) || native.Actual != status || string(native.Body) != raw || len(*calls) != 1 {
			t.Fatal(err, value, native, *calls)
		}
		nativeRuleOperation(t, err, "GetDSCPMarkingRecord")
	}
}
func TestPythonDSCPMarkingFindBoundRequiredKwargsAndUnknownConstructorSeeds(t *testing.T) {
	raw := `{"dscp_marking_rule":{"dscp_mark":null,"x_new":false}}`
	api, calls := dscpRecordAPI(t, func(r *http.Request) *http.Response {
		if r.URL.Path != dscpRecordPath+"/target" || r.URL.Query().Get("qos_policy_id") != "policy" || !reflect.DeepEqual(r.URL.Query()["x_seed"], []string{"0", "False"}) || r.URL.Query().Has("all_projects") {
			t.Fatal(r.URL)
		}
		return dscpRecordWire(200, raw)
	})
	value, err := api.FindDSCPMarkingRecord(context.Background(), "policy", "target", rules.WithDSCPMarkingFindQuery("name", "constructor"), rules.WithDSCPMarkingFindQuery("x_seed", []any{0, false, nil}), rules.WithDSCPMarkingFindQuery("all_projects", true))
	if err != nil || value == nil || len(*calls) != 1 || value.PolicyID != "policy" || value.Resource.Body["qos_policy_id"] != nil {
		t.Fatal(err, value, *calls)
	}
	dscpRecordField(t, value, "id", `"target"`)
	dscpRecordField(t, value, "name", `"constructor"`)
	dscpRecordField(t, value, "x_seed", `[0,false,null]`)
	dscpRecordField(t, value, "dscp_mark", "null")
}
func TestPythonDSCPMarkingFindCleanFallbackLocalFiltersAndNoNameHint(t *testing.T) {
	for _, status := range []int{400, 403, 404} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			n := 0
			var listed url.Values
			api, calls := dscpRecordAPI(t, func(r *http.Request) *http.Response {
				n++
				if n == 1 {
					return dscpRecordWire(status, `{"error":"member hidden"}`)
				}
				listed = r.URL.Query()
				return dscpRecordWire(200, `{"dscp_marking_rules":[{"id":"skip","name":"target","dscp_mark":1},{"id":"match","name":"target","dscp_mark":false,"qos_policy_id":"foreign","x_new":null}]}`)
			})
			value, err := api.FindDSCPMarkingRecord(context.Background(), "policy", "target", rules.WithDSCPMarkingFindQuery("dscp_mark", 0), rules.WithDSCPMarkingFindQuery("x_unknown", false), rules.WithDSCPMarkingFindQuery("fields", []string{"id"}), rules.WithDSCPMarkingFindQuery("all_projects", true))
			if err != nil || value == nil || value.PolicyID != "policy" || len(*calls) != 2 || len(listed) != 0 || (*calls)[1].path != dscpRecordPath || value.Resource.Body["qos_policy_id"] != nil {
				t.Fatal(err, value, *calls, listed)
			}
			dscpRecordField(t, value, "id", `"match"`)
			dscpRecordField(t, value, "dscp_mark", "false")
			dscpRecordField(t, value, "x_new", "null")
			if value.Resource.Body["x_unknown"] != nil {
				t.Fatal("GET seed leaked into list row", value)
			}
		})
	}
}
func TestPythonDSCPMarkingFindCompletesUniquenessAndMissingSearch(t *testing.T) {
	for _, mode := range []string{"unique linked", "duplicate linked", "missing default", "missing strict", "later page rejection"} {
		t.Run(mode, func(t *testing.T) {
			n := 0
			api, calls := dscpRecordAPI(t, func(r *http.Request) *http.Response {
				n++
				if n == 1 {
					return dscpRecordWire(404, `{"error":"missing"}`)
				}
				if mode == "missing default" || mode == "missing strict" {
					return dscpRecordWire(200, `{"dscp_marking_rules":[]}`)
				}
				if n == 2 {
					return dscpRecordWire(200, `{"dscp_marking_rules":[{"id":"one","name":"target"}],"dscp_marking_rules_links":[{"rel":"next","href":"?marker=one"}]}`)
				}
				if r.URL.Query().Get("marker") != "one" {
					t.Fatal(r.URL)
				}
				if mode == "duplicate linked" {
					return dscpRecordWire(200, `{"dscp_marking_rules":[{"id":"two","name":"target"}]}`)
				}
				if mode == "later page rejection" {
					return dscpRecordWire(403, `{"error":"later denied"}`)
				}
				return dscpRecordWire(200, `{"dscp_marking_rules":[]}`)
			})
			var options []rules.DSCPMarkingFindOption
			if mode == "missing strict" {
				options = append(options, rules.WithDSCPMarkingFindIgnoreMissing(false))
			}
			value, err := api.FindDSCPMarkingRecord(context.Background(), "policy", "target", options...)
			switch mode {
			case "unique linked":
				if err != nil || value == nil || len(*calls) != 3 {
					t.Fatal(err, value, *calls)
				}
			case "duplicate linked":
				if value != nil || !errors.Is(err, resource.ErrAmbiguous) || len(*calls) != 3 {
					t.Fatal(err, value, *calls)
				}
			case "missing default":
				if err != nil || value != nil || len(*calls) != 2 {
					t.Fatal(err, value, *calls)
				}
			case "missing strict":
				if value != nil || !errors.Is(err, resource.ErrNotFound) || len(*calls) != 2 {
					t.Fatal(err, value, *calls)
				}
			case "later page rejection":
				var native gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.As(err, &native) || native.Actual != 403 || len(*calls) != 3 {
					t.Fatal(err, value, *calls)
				}
			}
		})
	}
}
func TestPythonDSCPMarkingListServerClassificationRawCapAndOwnedURI(t *testing.T) {
	raw := `{"dscp_marking_rules":[{"id":"skip","name":"other","dscp_mark":0},{"id":"one","name":"wanted","dscp_mark":false,"qos_policy_id":"foreign","x_extension":900719925474099312345},null],"next":"https://foreign.invalid/"}`
	api, calls := dscpRecordAPI(t, func(r *http.Request) *http.Response {
		if r.URL.Path != dscpRecordPath || r.URL.Query().Get("limit") != "2" || r.URL.Query().Get("marker") != "start" || len(r.URL.Query()) != 2 || r.Header.Get("Accept") != "application/json" {
			t.Fatal(r.URL, r.Header)
		}
		return dscpRecordWire(200, raw)
	})
	seq := api.ListDSCPMarkingRecords(context.Background(), "policy", rules.WithDSCPMarkingListMaxItems(2), rules.WithDSCPMarkingListMarker("start"), rules.WithDSCPMarkingListQuery("name", "wanted"), rules.WithDSCPMarkingListQuery("dscp_mark", 0), rules.WithDSCPMarkingListQuery("fields", make(chan int)), rules.WithDSCPMarkingListQuery("unknown", make(chan int)))
	if len(*calls) != 0 {
		t.Fatal("list not lazy")
	}
	values, err := collectDSCPRecords(seq)
	if err != nil || len(values) != 1 || len(*calls) != 1 || values[0].PolicyID != "policy" || values[0].Resource.Body["qos_policy_id"] != nil || string(values[0].Wire.Body["qos_policy_id"]) != `"foreign"` || string(values[0].Envelope) != raw {
		t.Fatal(err, values, *calls)
	}
	dscpRecordField(t, values[0], "dscp_mark", "false")
	dscpRecordField(t, values[0], "x_extension", "900719925474099312345")
}
func TestPythonDSCPMarkingListOrderedDictionaryFiltersAndExactNumbers(t *testing.T) {
	raw := `{"dscp_marking_rules":[{"id":"truthy","name":"different","dscp_mark":9007199254740992}]}`
	for _, mode := range []string{"false first", "scalar fault first", "empty predicate", "exact decimal"} {
		t.Run(mode, func(t *testing.T) {
			api, calls := dscpRecordAPI(t, func(*http.Request) *http.Response { return dscpRecordWire(200, raw) })
			var options []rules.DSCPMarkingListOption
			want := 0
			failure := false
			switch mode {
			case "false first":
				options = []rules.DSCPMarkingListOption{rules.WithDSCPMarkingListQuery("name", "no"), rules.WithDSCPMarkingListQuery("id", map[string]any{"x": true})}
			case "scalar fault first":
				options = []rules.DSCPMarkingListOption{rules.WithDSCPMarkingListQuery("id", map[string]any{"x": true}), rules.WithDSCPMarkingListQuery("name", "no")}
				failure = true
			case "empty predicate":
				options = []rules.DSCPMarkingListOption{rules.WithDSCPMarkingListQuery("id", map[string]any{})}
				want = 1
			case "exact decimal":
				options = []rules.DSCPMarkingListOption{rules.WithDSCPMarkingListQuery("dscp_mark", json.RawMessage("9007199254740993.0"))}
			}
			values, err := collectDSCPRecords(api.ListDSCPMarkingRecords(context.Background(), "policy", options...))
			if (err != nil) != failure || len(values) != want || len(*calls) != 1 {
				t.Fatal(err, values, *calls)
			}
			if failure {
				dscpRecordProof(t, err, 200, raw)
			}
		})
	}
}
func TestPythonDSCPMarkingListPagingBoundsAndConstructorProof(t *testing.T) {
	for _, mode := range []string{"versioned link", "marker short page", "singleton", "single page", "cycle", "foreign policy", "constructor cls", "constructor connection", "wrong row", "null envelope", "missing envelope", "blank204", "malformed"} {
		t.Run(mode, func(t *testing.T) {
			n := 0
			lastBody := ""
			lastStatus := 200
			api, calls := dscpRecordAPI(t, func(r *http.Request) *http.Response {
				n++
				lastBody = `{"dscp_marking_rules":[]}`
				lastStatus = 200
				if n == 1 {
					switch mode {
					case "versioned link":
						lastBody = `{"dscp_marking_rules":[{"id":"one"}],"next":"/v2.0/qos/policies/policy/dscp_marking_rules?marker=one"}`
					case "marker short page", "single page":
						lastBody = `{"dscp_marking_rules":[{"id":"one"}]}`
					case "singleton":
						lastBody = `{"dscp_marking_rules":{"id":"one","dscp_mark":null}}`
					case "cycle":
						lastBody = `{"dscp_marking_rules":[{"id":"one"}],"next":"?marker=initial"}`
					case "foreign policy":
						lastBody = `{"dscp_marking_rules":[{"id":"one"}],"next":"../foreign/dscp_marking_rules?marker=one"}`
					case "constructor cls":
						lastBody = `{"dscp_marking_rules":[{"cls":false}]}`
					case "constructor connection":
						lastBody = `{"dscp_marking_rules":[{"connection":null}]}`
					case "wrong row":
						lastBody = `{"dscp_marking_rules":[null]}`
					case "null envelope":
						lastBody = `{"dscp_marking_rules":null}`
					case "missing envelope":
						lastBody = `{}`
					case "blank204":
						lastBody = ""
						lastStatus = 204
					case "malformed":
						lastBody = `{"dscp_marking_rules":[{"id":"one"},]}`
					}
				} else if r.URL.Query().Get("marker") != "one" {
					t.Fatal(r.URL)
				}
				return dscpRecordWire(lastStatus, lastBody)
			})
			options := []rules.DSCPMarkingListOption{rules.WithDSCPMarkingListMarker("initial")}
			if mode == "marker short page" || mode == "single page" {
				options = append(options, rules.WithDSCPMarkingListLimit(2))
			}
			if mode == "single page" {
				options = append(options, rules.WithDSCPMarkingListPaginated(false))
			}
			values, err := collectDSCPRecords(api.ListDSCPMarkingRecords(context.Background(), "policy", options...))
			success := mode == "versioned link" || mode == "marker short page" || mode == "single page" || mode == "singleton"
			if (err == nil) != success {
				t.Fatal(mode, err, values, *calls)
			}
			if success {
				if len(values) != 1 {
					t.Fatal(values)
				}
				want := 1
				if mode == "versioned link" || mode == "marker short page" {
					want = 2
				}
				if len(*calls) != want {
					t.Fatal(*calls)
				}
			} else {
				if len(*calls) != 1 {
					t.Fatal(*calls)
				}
				dscpRecordProof(t, err, lastStatus, lastBody)
			}
		})
	}
}

type dscpRecordFaultBody struct {
	io.Reader
	closed     atomic.Int32
	closeCause error
	onClose    func()
}

func (b *dscpRecordFaultBody) Close() error {
	b.closed.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeCause
}

type dscpRecordFaultReader func([]byte) (int, error)

func (f dscpRecordFaultReader) Read(p []byte) (int, error) { return f(p) }
func TestPythonDSCPMarkingPhysicalFaultsPreventFallbackAndKeepEvidence(t *testing.T) {
	for _, operation := range []string{"get", "find", "list"} {
		for _, status := range []int{200, 400, 403, 404, 500, 503} {
			for _, fault := range []string{"read", "close", "mixed", "source read restored close", "cancel close"} {
				t.Run(fmt.Sprintf("%s/%d/%s", operation, status, fault), func(t *testing.T) {
					raw := `{"dscp_marking_rule":{"id":"one"}}`
					if operation == "list" {
						raw = `{"dscp_marking_rules":[{"id":"one"}]}`
					}
					readCause, closeCause, cancelCause := errors.New("read cause"), errors.New("close cause"), errors.New("cancel cause")
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					var api *rules.API
					body := &dscpRecordFaultBody{Reader: strings.NewReader(raw)}
					if fault == "read" || fault == "mixed" {
						done := false
						body.Reader = dscpRecordFaultReader(func(p []byte) (int, error) {
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
					original := ""
					if fault == "source read restored close" {
						done := false
						body.Reader = dscpRecordFaultReader(func(p []byte) (int, error) {
							if done {
								return 0, io.EOF
							}
							done = true
							api.RawClient().ResourceBase = "https://foreign.invalid/v2/"
							return copy(p, raw), nil
						})
						body.onClose = func() { api.RawClient().ResourceBase = original }
					}
					if fault == "cancel close" {
						body.onClose = func() { cancel(cancelCause) }
					}
					var calls *[]nativeRuleCall
					api, calls = dscpRecordAPI(t, func(*http.Request) *http.Response {
						response := dscpRecordWire(status, raw)
						response.Body = body
						return response
					})
					original = api.RawClient().ResourceBase
					var err error
					var value *rules.DSCPMarkingRecord
					switch operation {
					case "get":
						value, err = api.GetDSCPMarkingRecord(ctx, "policy", "target")
					case "find":
						value, err = api.FindDSCPMarkingRecord(ctx, "policy", "target")
					case "list":
						var values []*rules.DSCPMarkingRecord
						values, err = collectDSCPRecords(api.ListDSCPMarkingRecords(ctx, "policy"))
						if len(values) != 0 {
							t.Fatal(values)
						}
					}
					if err == nil || value != nil || len(*calls) != 1 || body.closed.Load() != 1 {
						t.Fatal(err, value, *calls, body.closed.Load())
					}
					if status < 400 {
						dscpRecordProof(t, err, status, raw)
					} else {
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(err, &native) || native.Actual != status || string(native.Body) != raw || native.ResponseHeader.Get("X-Request-Id") != "actual-dscp" {
							t.Fatal(err, native)
						}
					}
					if (fault == "read" || fault == "mixed") && !errors.Is(err, readCause) || (fault == "close" || fault == "mixed") && !errors.Is(err, closeCause) || fault == "source read restored close" && !errors.Is(err, resource.ErrInvalidOption) || fault == "cancel close" && !errors.Is(err, cancelCause) {
						t.Fatal(err)
					}
				})
			}
		}
	}
}
func TestPythonDSCPMarkingVersionsAndRetryHeadersRemainSelected(t *testing.T) {
	for _, operation := range []string{"get", "find", "list"} {
		t.Run(operation, func(t *testing.T) {
			n := 0
			var versions, tokens, accepts []string
			api, calls := dscpRecordAPI(t, func(r *http.Request) *http.Response {
				n++
				versions = append(versions, r.Header.Get("OpenStack-API-Version"))
				tokens = append(tokens, r.Header.Get("X-Auth-Token"))
				accepts = append(accepts, r.Header.Get("Accept"))
				if n == 1 {
					return dscpRecordWire(503, `{"retry":true}`)
				}
				if operation == "find" && n == 2 {
					return dscpRecordWire(404, `{"missing":true}`)
				}
				if operation == "get" {
					return dscpRecordWire(200, `{"dscp_marking_rule":{"id":"target"}}`)
				}
				return dscpRecordWire(200, `{"dscp_marking_rules":[]}`)
			})
			api.RawClient().Microversion = "2.0"
			api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, original error, _ uint) error {
				if gophercloud.ResponseCodeIs(original, 404) {
					return original
				}
				opts.MoreHeaders["X-Retry-Probe"] = "ordinary"
				api.RawClient().SetToken("fresh")
				return nil
			}
			var err error
			switch operation {
			case "get":
				_, err = api.GetDSCPMarkingRecord(context.Background(), "policy", "target", rules.WithDSCPMarkingGetMicroversion("2.1"))
			case "find":
				_, err = api.FindDSCPMarkingRecord(context.Background(), "policy", "target", rules.WithDSCPMarkingFindMicroversion("2.1"))
			case "list":
				_, err = collectDSCPRecords(api.ListDSCPMarkingRecords(context.Background(), "policy", rules.WithDSCPMarkingListMicroversion("2.1")))
			}
			want := []string{"network 2.1", "network 2.1"}
			if operation == "find" {
				want = append(want, "network 2.0")
			}
			if err != nil || !reflect.DeepEqual(versions, want) || tokens[0] != "test-token" || tokens[1] != "fresh" || len(*calls) != len(want) || api.RawClient().Microversion != "2.0" || api.RawClient().MoreHeaders != nil {
				t.Fatal(err, versions, tokens, accepts, *calls)
			}
			if operation == "list" || operation == "find" {
				if accepts[len(accepts)-1] != "application/json" {
					t.Fatal(accepts)
				}
			}
		})
	}
	for _, operation := range []string{"get", "find", "list"} {
		for _, key := range []string{"Authorization", "X-Service-Token", "Content-Length", "X-OpenStack-Nova-API-Version", "X-OpenStack-Volume-API-Version", "OpenStack-API-Version"} {
			t.Run(operation+key, func(t *testing.T) {
				raw := `{"retry":"physical"}`
				body := &dscpRecordFaultBody{Reader: strings.NewReader(raw)}
				cause := errors.New("callback cause")
				api, calls := dscpRecordAPI(t, func(*http.Request) *http.Response {
					response := dscpRecordWire(503, raw)
					response.Body = body
					return response
				})
				api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
					opts.MoreHeaders[key] = "foreign"
					return cause
				}
				var err error
				switch operation {
				case "get":
					_, err = api.GetDSCPMarkingRecord(context.Background(), "policy", "target")
				case "find":
					_, err = api.FindDSCPMarkingRecord(context.Background(), "policy", "target")
				case "list":
					_, err = collectDSCPRecords(api.ListDSCPMarkingRecords(context.Background(), "policy"))
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != raw || len(*calls) != 1 || body.closed.Load() != 1 {
					t.Fatal(err, native, *calls, body.closed.Load())
				}
			})
		}
	}
}
func TestPythonDSCPMarkingPreflightBoundControlsCancellationAndSnapshots(t *testing.T) {
	api, calls := dscpRecordAPI(t, func(*http.Request) *http.Response { t.Fatal("preflight sent HTTP"); return nil })
	for _, parent := range []string{"", " ", ".", "..", "bad\n", "bad\xff"} {
		_, err := api.GetDSCPMarkingRecord(context.Background(), parent, "rule")
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	_, err := api.FindDSCPMarkingRecord(context.Background(), "policy", "target", rules.WithDSCPMarkingFindQuery("qos_policy_id", nil))
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	_, err = api.FindDSCPMarkingRecord(context.Background(), "policy", "target", rules.WithDSCPMarkingFindQuery("qos_rule_id", "target"))
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	_, err = collectDSCPRecords(api.ListDSCPMarkingRecords(context.Background(), "policy", rules.WithDSCPMarkingListQuery("qos_policy_id", false)))
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, key := range []string{"cls", "connection", "microversion", "base_path", "location", "id"} {
		_, err = api.FindDSCPMarkingRecord(context.Background(), "policy", "target", rules.WithDSCPMarkingFindQuery(key, nil))
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(key, err)
		}
	}
	_, err = api.GetDSCPMarkingRecord(nil, "policy", "target")
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	_, err = (*rules.API)(nil).FindDSCPMarkingRecord(context.Background(), "policy", "target")
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	_, err = api.GetDSCPMarkingRecord(context.Background(), "policy", "target", nil)
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	cause := errors.New("canceled option")
	ctx, cancel := context.WithCancelCause(context.Background())
	_, err = api.FindDSCPMarkingRecord(ctx, "policy", "target", func(*rules.DSCPMarkingFindOpts) error { cancel(cause); return nil })
	if !errors.Is(err, cause) {
		t.Fatal(err)
	}
	var proof *resource.ResponseError
	if len(*calls) != 0 || errors.As(err, &proof) {
		t.Fatal(err, *calls)
	}
	raw := json.RawMessage(`"original"`)
	headers := map[string]string{"X-Probe": "original"}
	version := "2.1"
	cfg := rules.DSCPMarkingFindOpts{Query: map[string]json.RawMessage{"x_seed": raw}, Headers: headers, Microversion: &version}
	owned := rules.WithDSCPMarkingFindOpts(cfg)
	raw[1] = 'X'
	headers["X-Probe"] = "changed"
	version = "bad\n"
	var retained *rules.DSCPMarkingFindOpts
	api, calls = dscpRecordAPI(t, func(r *http.Request) *http.Response {
		retained.Query["x_seed"][1] = 'Y'
		retained.Headers["X-Probe"] = "mutated"
		*retained.Microversion = "bad\n"
		if r.URL.Query().Get("x_seed") != "original" || r.Header.Get("X-Probe") != "original" || r.Header.Get("OpenStack-API-Version") != "network 2.1" {
			t.Fatal(r.URL, r.Header)
		}
		return dscpRecordWire(200, `{"dscp_marking_rule":{}}`)
	})
	value, err := api.FindDSCPMarkingRecord(context.Background(), "policy", "target", owned, func(v *rules.DSCPMarkingFindOpts) error { retained = v; return nil })
	if err != nil || value == nil || len(*calls) != 1 {
		t.Fatal(err, value, *calls)
	}
	dscpRecordField(t, value, "x_seed", `"original"`)
}
func TestPythonDSCPMarkingPublicFixtureAndConcurrentOwnedReuse(t *testing.T) {
	cloud := testcloud.New(t)
	var requests atomic.Int32
	cloud.Mux.HandleFunc("/neutron/", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != "GET" || r.URL.EscapedPath() != dscpRecordPath+"/literal%2Fid" || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Probe") != "owned" {
			t.Error(r.Method, r.RequestURI, r.Header)
		}
		testcloud.JSON(w, 200, `{"dscp_marking_rule":{"id":"literal/id","dscp_mark":0}}`)
	})
	client := cloud.Client("network", "/catalog/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	api := rules.New(client)
	headers := map[string]string{"X-Probe": "owned"}
	option := rules.WithDSCPMarkingGetHeaders(headers)
	headers["X-Probe"] = "changed"
	var wait sync.WaitGroup
	for i := 0; i < 8; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			value, err := api.GetDSCPMarkingRecord(context.Background(), "policy", "literal/id", option)
			if err != nil || value == nil || value.PolicyID != "policy" {
				t.Error(err, value)
			}
		}()
	}
	wait.Wait()
	if requests.Load() != 8 || client.MoreHeaders != nil {
		t.Fatal(requests.Load(), client)
	}
}
