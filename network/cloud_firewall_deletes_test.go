package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type firewallDeleteFamily struct {
	family, singular string
	call             func(*network.Service, context.Context, string, ...network.FirewallDeleteOption) (*network.FirewallDeletion, error)
}

var firewallDeleteFamilies = []firewallDeleteFamily{
	{"firewall_rules", "firewall_rule", (*network.Service).DeleteFirewallRuleRecord},
	{"firewall_policies", "firewall_policy", (*network.Service).DeleteFirewallPolicyRecord},
	{"firewall_groups", "firewall_group", (*network.Service).DeleteFirewallGroupRecord},
}

func TestCloudFirewallDeleteStrictFindThenActualIDAndDelete404IsFalse(t *testing.T) {
	for _, f := range firewallDeleteFamilies {
		for _, status := range []int{204, 202, 203, 404, 409} {
			t.Run(fmt.Sprintf("%s/%d", f.family, status), func(t *testing.T) {
				var methods []string
				service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
					methods = append(methods, r.Method+" "+r.URL.Path)
					if r.Method == http.MethodGet {
						if r.URL.Path != "/neutron/v2.0/fwaas/"+f.family+"/asked" {
							t.Error(r.URL)
						}
						testcloud.JSON(w, 203, `{"`+f.singular+`":{"id":"actual","name":"kept","project_id":"foreign","ports":false,"firewall_rules":"one","x_new":9007199254740993,"summary":{"forged":true}}}`)
						return
					}
					if r.Method != http.MethodDelete || r.URL.Path != "/neutron/v2.0/fwaas/"+f.family+"/actual" || r.URL.RawQuery != "" || r.ContentLength > 0 {
						t.Error(r.Method, r.URL, r.ContentLength)
					}
					if status == 204 {
						w.WriteHeader(status)
						return
					}
					testcloud.JSON(w, status, `actual deletion body`)
				})
				result, err := f.call(service, context.Background(), "asked")
				if result == nil || result.Lookup == nil || result.Lookup.Resource == nil || result.Lookup.StatusCode != 203 || string(result.Lookup.Resource.Body["x_new"]) != "9007199254740993" || count.Load() != 2 || len(methods) != 2 {
					t.Fatal(result, err, count.Load(), methods)
				}
				if result.Delete == nil || result.Delete.StatusCode != status || result.Delete.Header.Get("X-Request-Id") != "qos-mutation-actual" || result.Delete.Resource != nil || result.Delete.Wire != nil {
					t.Fatal(result, err)
				}
				if status == 409 {
					if err == nil || result.Deleted || !gophercloud.ResponseCodeIs(err, 409) {
						t.Fatal(result, err)
					}
					return
				}
				if err != nil || result.Deleted != (status != 404) {
					t.Fatal(result, err)
				}
				if f.family == "firewall_groups" && string(result.Lookup.Resource.Body["ports"]) != "[false]" {
					t.Fatal(result.Lookup)
				}
				if f.family == "firewall_policies" && string(result.Lookup.Resource.Body["firewall_rules"]) != `["one"]` {
					t.Fatal(result.Lookup)
				}
				if f.family == "firewall_rules" && (string(result.Lookup.Resource.Body["summary"]) != `""` || string(result.Lookup.Wire.Body["summary"]) != `{"forged":true}`) {
					t.Fatal(result.Lookup)
				}
			})
		}
	}
}
func TestCloudFirewallDeleteMissingStrictLookupAndFallbackErrors(t *testing.T) {
	for _, f := range firewallDeleteFamilies {
		for _, lane := range []string{"empty", "list404", "later-duplicate", "late503", "found"} {
			t.Run(f.family+"/"+lane, func(t *testing.T) {
				service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
					path := "/neutron/v2.0/fwaas/" + f.family
					if r.Method == http.MethodDelete {
						if lane != "found" {
							t.Fatal("delete before unique complete match", lane)
						}
						w.WriteHeader(204)
						return
					}
					if r.URL.Path == path+"/name" {
						testcloud.JSON(w, 403, `{"error":"member denied"}`)
						return
					}
					if r.URL.Path != path || r.URL.Query().Get("name") != "name" {
						t.Error(r.URL)
					}
					switch lane {
					case "empty":
						testcloud.JSON(w, 200, `{"`+f.family+`":[]}`)
					case "list404":
						testcloud.JSON(w, 404, `{"error":"missing list"}`)
					case "found":
						testcloud.JSON(w, 200, `{"`+f.family+`":[{"id":"actual","name":"name"}]}`)
					default:
						if r.URL.Query().Get("marker") == "later" {
							if lane == "late503" {
								testcloud.JSON(w, 503, `{"error":"late actual"}`)
							} else {
								testcloud.JSON(w, 200, `{"`+f.family+`":[{"id":"second","name":"name"}]}`)
							}
							return
						}
						testcloud.JSON(w, 200, `{"`+f.family+`":[{"id":"first","name":"name"}],"next":"?marker=later"}`)
					}
				})
				result, err := f.call(service, context.Background(), "name")
				if result == nil {
					t.Fatal(result, err)
				}
				switch lane {
				case "empty", "list404":
					if err != nil || result.Deleted || result.Delete != nil || count.Load() != 2 {
						t.Fatal(result, err, count.Load())
					}
				case "found":
					if err != nil || !result.Deleted || result.Lookup == nil || count.Load() != 3 {
						t.Fatal(result, err, count.Load())
					}
				case "later-duplicate":
					if !errors.Is(err, resource.ErrAmbiguous) || result.Deleted || result.Delete != nil || count.Load() != 3 {
						t.Fatal(result, err, count.Load())
					}
				case "late503":
					if !gophercloud.ResponseCodeIs(err, 503) || result.Deleted || result.Delete != nil || count.Load() != 3 {
						t.Fatal(result, err, count.Load())
					}
				}
			})
		}
	}
}
func TestCloudFirewallDeleteFiltersOwnJSONAndUntypedLocalProjection(t *testing.T) {
	for _, f := range firewallDeleteFamilies {
		for _, raw := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(`false`), json.RawMessage(`0`), json.RawMessage(`1e-4000`), json.RawMessage(`""`), json.RawMessage(`[]`), json.RawMessage(`{}`)} {
			service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				if r.Method == http.MethodDelete {
					w.WriteHeader(204)
					return
				}
				testcloud.JSON(w, 200, `{"`+f.singular+`":{"id":"actual"}}`)
			})
			result, err := f.call(service, context.Background(), "asked", network.WithFirewallDeleteFilters(raw))
			if err != nil || !result.Deleted || count.Load() != 2 {
				t.Fatal(f.family, string(raw), result, err, count.Load())
			}
		}
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				if r.Header.Get("X-Owned") != "before" {
					t.Error(r.Header)
				}
				w.WriteHeader(204)
				return
			}
			if strings.HasSuffix(r.URL.Path, "/asked") {
				if r.URL.Query().Get("name") != "override" || r.URL.Query().Get("x_seed") != "precise" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 404, `{"error":"fallback"}`)
				return
			}
			if r.URL.Query().Get("name") != "override" || r.URL.Query().Has("x_seed") {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, `{"`+f.family+`":[{"id":"actual","name":"asked","project_id":false,"shared":"false","enabled":0,"ip_version":"4","admin_state_up":1,"audited":true}]}`)
		})
		raw := json.RawMessage(`{"name":"override","x_seed":"precise"}`)
		headers := map[string]string{"X-Owned": "before"}
		opts := []network.FirewallDeleteOption{network.WithFirewallDeleteFilters(raw), network.WithFirewallDeleteHeaders(headers)}
		raw[2] = 'X'
		headers["X-Owned"] = "after"
		result, err := f.call(service, context.Background(), "asked", opts...)
		if err != nil || !result.Deleted || result.Lookup == nil || string(result.Lookup.Resource.Body["shared"]) != `"false"` || count.Load() != 3 {
			t.Fatal(f.family, result, err, count.Load())
		}
		for _, bad := range []json.RawMessage{json.RawMessage(`"[?name]"`), json.RawMessage(`[["name","x"]]`), json.RawMessage(`{"ignore_missing":null}`), json.RawMessage(`{"name_or_id":null}`), json.RawMessage(`{bad`)} {
			before := count.Load()
			result, err := f.call(service, context.Background(), "asked", network.WithFirewallDeleteFilters(bad))
			if !errors.Is(err, resource.ErrInvalidOption) || result.Deleted || count.Load() != before {
				t.Fatal(string(bad), result, err, count.Load())
			}
		}
	}
}
func TestCloudFirewallDeleteAcceptedEmptyLookupAndFalseyReturnedIDBoundaries(t *testing.T) {
	for _, f := range firewallDeleteFamilies {
		for _, lane := range []string{"empty204", "malformed", "null-id", "false-id"} {
			service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					if lane != "empty204" && lane != "malformed" {
						t.Fatal("falsey collection delete")
					}
					if !strings.HasSuffix(r.URL.Path, "/asked") {
						t.Error(r.URL)
					}
					w.WriteHeader(204)
					return
				}
				switch lane {
				case "empty204":
					w.WriteHeader(204)
				case "malformed":
					testcloud.JSON(w, 200, `{bad`)
				case "null-id":
					testcloud.JSON(w, 200, `{"`+f.singular+`":{"id":null}}`)
				case "false-id":
					testcloud.JSON(w, 200, `{"`+f.singular+`":{"id":false}}`)
				}
			})
			result, err := f.call(service, context.Background(), "asked")
			if lane == "null-id" || lane == "false-id" {
				if err == nil || result.Deleted || result.Delete != nil || count.Load() != 1 {
					t.Fatal(f.family, lane, result, err, count.Load())
				}
			} else if err != nil || !result.Deleted || result.Lookup == nil || count.Load() != 2 {
				t.Fatal(f.family, lane, result, err, count.Load())
			}
		}
	}
}

type firewallDeleteObservedTransport struct {
	base              http.RoundTripper
	method, family    string
	readErr, closeErr error
	closed            *atomic.Int32
	onClose           func()
}

func (t firewallDeleteObservedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	v, err := t.base.RoundTrip(r)
	if v != nil && r.Method == t.method && strings.Contains(r.URL.Path, "/fwaas/"+t.family+"/") {
		v.Body = &qosPolicyActionObservedBody{ReadCloser: v.Body, readErr: t.readErr, closeErr: t.closeErr, closed: t.closed, onClose: t.onClose}
	}
	return v, err
}
func TestCloudFirewallDeletePhysicalFaultsAreNeverMissingAndKeepActualProof(t *testing.T) {
	for _, f := range firewallDeleteFamilies {
		for _, method := range []string{http.MethodGet, http.MethodDelete} {
			for _, status := range []int{200, 401, 404, 503} {
				for _, lane := range []string{"read-close", "cancel", "source"} {
					t.Run(fmt.Sprintf("%s/%s/%d/%s", f.family, method, status, lane), func(t *testing.T) {
						body := `{"` + f.singular + `":{"id":"actual"}}`
						service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
							if method == http.MethodDelete && r.Method == http.MethodGet {
								testcloud.JSON(w, 200, body)
								return
							}
							testcloud.JSON(w, status, body)
						})
						readErr, closeErr, cancelCause := errors.New("read actual"), errors.New("close actual"), errors.New("caller cause")
						ctx, cancel := context.WithCancelCause(context.Background())
						defer cancel(nil)
						closed := &atomic.Int32{}
						base := service.RawClient().HTTPClient.Transport
						if base == nil {
							base = http.DefaultTransport
						}
						observer := firewallDeleteObservedTransport{base: base, method: method, family: f.family, closed: closed}
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
						unexpected := errors.New("unexpected native callback")
						service.RawClient().ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
							retries++
							return unexpected
						}
						service.RawClient().ProviderClient.ReauthFunc = func(context.Context) error { reauths++; return unexpected }
						result, err := f.call(service, ctx, "asked")
						want := int32(1)
						if method == http.MethodDelete {
							want = 2
						}
						if result == nil || result.Deleted || err == nil || count.Load() != want || closed.Load() != 1 || retries != 0 || reauths != 0 {
							t.Fatal(result, err, count.Load(), closed.Load(), retries, reauths)
						}
						switch lane {
						case "read-close":
							physical := err
							if status == http.StatusUnauthorized {
								// Pinned native carrier has no Unwrap. Keep any
								// outer joined fault and verify its independent
								// ErrReauth handling causes explicitly.
								var auth *gophercloud.ErrUnableToReauthenticate
								if !errors.As(err, &auth) || auth == nil || auth.ErrOriginal == nil || !errors.Is(auth.ErrReauth, readErr) || !errors.Is(auth.ErrReauth, closeErr) {
									t.Fatal(err, auth)
								}
								physical = errors.Join(err, auth.ErrReauth)
							}
							if !errors.Is(physical, readErr) || !errors.Is(physical, closeErr) {
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
						proof := result.Lookup
						if method == http.MethodDelete {
							proof = result.Delete
						}
						if proof == nil || proof.StatusCode != status || string(proof.Envelope) != body || proof.Header.Get("X-Request-Id") != "qos-mutation-actual" || proof.Resource != nil || proof.Wire != nil {
							t.Fatal(result, err, proof)
						}
						if status == 200 {
							var actual *resource.ResponseError
							if !errors.As(err, &actual) || actual.StatusCode != status || string(actual.Body) != body {
								t.Fatal(err, actual)
							}
						} else {
							nativeErr := err
							if status == 401 {
								var auth *gophercloud.ErrUnableToReauthenticate
								if !errors.As(err, &auth) || auth.ErrOriginal == nil || auth.ErrReauth == nil {
									t.Fatal(err, auth)
								}
								nativeErr = auth.ErrOriginal
							}
							var native gophercloud.ErrUnexpectedResponseCode
							if !errors.As(nativeErr, &native) || native.Actual != status || native.Method != method || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "qos-mutation-actual" {
								t.Fatal(err, native)
							}
						}
					})
				}
			}
		}
	}
}
func TestCloudFirewallDeleteOptionsSourceGuardsAndFabricated404NeverFalse(t *testing.T) {
	f := firewallDeleteFamilies[0]
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 503, `{"error":"actual503"}`) })
	fake := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{200}, Method: http.MethodGet, URL: "https://callback.invalid", Body: []byte("fabricated")}
	calls := 0
	service.RawClient().ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		calls++
		return fake
	}
	result, err := f.call(service, context.Background(), "asked")
	if err == nil || result.Deleted || count.Load() != 1 || calls != 1 || !gophercloud.ResponseCodeIs(err, 503) {
		t.Fatal(result, err, count.Load(), calls)
	}
	before := count.Load()
	result, err = f.call(service, context.Background(), "asked", func(*network.FirewallDeleteOpts) error { service.RawClient().ResourceBase += "changed/"; return nil })
	if err == nil || result.Deleted || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != before {
		t.Fatal(result, err, count.Load())
	}
}
