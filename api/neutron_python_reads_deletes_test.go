package api_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const neutronPythonMemberBase = "/proxy/neutron/v2.0/"

func neutronPythonMemberClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + neutronPythonMemberBase
	return client
}

type neutronPythonReadFixture struct {
	name, path, envelope string
	fields, wireFields   []string
	get                  func(context.Context, *gophercloud.ServiceClient, string, ...string) (any, error)
}

// These expectations are independent pinned Resource.Body aliases. Query-only
// aliases are deliberately not transposed when selecting response fields.
func neutronPythonReadFixtures() []neutronPythonReadFixture {
	return []neutronPythonReadFixture{
		{"networks", "networks", "network", []string{"is_admin_state_up", "subnet_ids", "provider_network_type"}, []string{"admin_state_up", "subnets", "provider:network_type"}, func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
			return networks.New(c).GetWithFields(ctx, id, fields...)
		}},
		{"subnets", "subnets", "subnet", []string{"prefix_length", "is_dhcp_enabled", "subnet_pool_id"}, []string{"prefixlen", "enable_dhcp", "subnetpool_id"}, func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
			return subnets.New(c).GetWithFields(ctx, id, fields...)
		}},
		{"ports", "ports", "port", []string{"binding_host_id", "is_admin_state_up", "security_group_ids"}, []string{"binding:host_id", "admin_state_up", "security_groups"}, func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
			return ports.New(c).GetWithFields(ctx, id, fields...)
		}},
		{"floatingips", "floatingips", "floatingip", []string{"any_tags", "not_any_tags", "floating_ip_address"}, []string{"any_tags", "not_any_tags", "floating_ip_address"}, func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
			return floatingips.New(c).GetWithFields(ctx, id, fields...)
		}},
		{"routers", "routers", "router", []string{"revision_number", "is_admin_state_up", "is_distributed"}, []string{"revision", "admin_state_up", "distributed"}, func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
			return routers.New(c).GetWithFields(ctx, id, fields...)
		}},
		{"security-groups", "security-groups", "security_group", []string{"is_shared", "security_group_rules", "any_tags"}, []string{"shared", "security_group_rules", "any_tags"}, func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
			return groups.New(c).GetWithFields(ctx, id, fields...)
		}},
		{"security-group-rules", "security-group-rules", "security_group_rule", []string{"ether_type", "security_group_id", "remote_group_id"}, []string{"ethertype", "security_group_id", "remote_group_id"}, func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
			return rules.New(c).GetWithFields(ctx, id, fields...)
		}},
		{"subnetpools", "subnetpools", "subnetpool", []string{"maximum_prefix_length", "minimum_prefix_length", "default_prefix_length"}, []string{"max_prefixlen", "min_prefixlen", "default_prefixlen"}, func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
			return subnetpools.New(c).GetWithFields(ctx, id, fields...)
		}},
		{"trunks", "trunks", "trunk", []string{"is_admin_state_up", "sub_ports", "any_tags"}, []string{"admin_state_up", "sub_ports", "any_tags"}, func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
			return trunks.New(c).GetWithFields(ctx, id, fields...)
		}},
		{"qos-policies", "qos/policies", "policy", []string{"is_shared", "is_default", "any_tags"}, []string{"shared", "is_default", "any_tags"}, func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
			return policies.New(c).GetWithFields(ctx, id, fields...)
		}},
		{"address-groups", "address-groups", "address_group", []string{"addresses", "any_tags", "not_any_tags"}, []string{"addresses", "any_tags", "not_any_tags"}, func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
			return addressgroups.New(c).GetWithFields(ctx, id, fields...)
		}},
	}
}

func neutronPythonReadID(t *testing.T, value any) string {
	t.Helper()
	v := reflect.ValueOf(value)
	if !v.IsValid() || v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		t.Fatalf("expected the native model pointer, got %#v", value)
	}
	id := v.Elem().FieldByName("ID")
	if !id.IsValid() || id.Kind() != reflect.String {
		t.Fatalf("model has no native string ID: %T", value)
	}
	return id.String()
}

func TestNeutronPythonGetFieldsPreservesAliasesOrderAndNativeModels(t *testing.T) {
	for _, fixture := range neutronPythonReadFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			fields := append([]string{"id", fixture.fields[0], "id"}, fixture.fields[1:]...)
			fields = append(fields, "unknown vendor:field /?# 한글", "", "any_tags")
			wire := append([]string{"id", fixture.wireFields[0], "id"}, fixture.wireFields[1:]...)
			wire = append(wire, "unknown vendor:field /?# 한글", "", "any_tags")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+neutronPythonMemberBase+fixture.path+"/target", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), url.Values{"fields": wire}) || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Vendor") != "configured" {
					t.Errorf("member GET lost ordered fields or source headers: %s %#v", r.URL, r.Header)
				}
				testcloud.JSON(w, 200, `{"`+fixture.envelope+`":{"id":"wire-id","project_id":"p","vendor":{"ignored":true}}}`)
			})
			client := neutronPythonMemberClient(cloud)
			client.MoreHeaders = map[string]string{"X-Vendor": "configured"}
			original := append([]string(nil), fields...)
			value, err := fixture.get(context.Background(), client, "target", fields...)
			if err != nil || neutronPythonReadID(t, value) != "wire-id" || calls.Load() != 1 || !reflect.DeepEqual(fields, original) {
				t.Fatalf("value=%#v calls=%d fields=%q err=%v", value, calls.Load(), fields, err)
			}
		})
	}
}

func TestNeutronPythonGetFieldsEmptyAndSparseResponse(t *testing.T) {
	for _, fixture := range neutronPythonReadFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+neutronPythonMemberBase+fixture.path+"/target", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Errorf("nil/empty fields created a query: %s", r.URL)
				}
				testcloud.JSON(w, 200, `{"`+fixture.envelope+`":{}}`)
			})
			client := neutronPythonMemberClient(cloud)
			for _, fields := range [][]string{nil, {}} {
				value, err := fixture.get(context.Background(), client, "target", fields...)
				if err != nil || neutronPythonReadID(t, value) != "" {
					t.Fatalf("sparse native response %#v: %v", value, err)
				}
			}
			if calls.Load() != 2 {
				t.Fatal("empty field calls were not independent", calls.Load())
			}
			if fixture.name == "subnetpools" {
				for _, check := range []struct {
					name, row string
					want      [3]int
					bad       bool
				}{
					{"id-only", `{"id":"wire-id"}`, [3]int{}, false},
					{"number-prefixes", `{"id":"wire-id","default_prefixlen":24,"min_prefixlen":8,"max_prefixlen":32}`, [3]int{24, 8, 32}, false},
					{"string-prefixes", `{"id":"wire-id","default_prefixlen":"24","min_prefixlen":"8","max_prefixlen":"32"}`, [3]int{24, 8, 32}, false},
					{"explicit-null-default", `{"id":"wire-id","default_prefixlen":null}`, [3]int{}, true},
					{"explicit-null-min", `{"id":"wire-id","min_prefixlen":null}`, [3]int{}, true},
					{"explicit-null-max", `{"id":"wire-id","max_prefixlen":null}`, [3]int{}, true},
					{"invalid-prefix-type", `{"id":"wire-id","default_prefixlen":false}`, [3]int{}, true},
				} {
					t.Run(check.name, func(t *testing.T) {
						cloud := testcloud.New(t)
						cloud.Mux.HandleFunc("GET "+neutronPythonMemberBase+fixture.path+"/target", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"subnetpool":`+check.row+`}`) })
						value, err := fixture.get(context.Background(), neutronPythonMemberClient(cloud), "target", "id")
						if check.bad {
							if err == nil {
								t.Fatal("explicit null/invalid native prefix became a default", value)
							}
							return
						}
						if err != nil {
							t.Fatal("missing/native prefix conversion failed", value, err)
						}
						pool, ok := value.(*subnetpools.SubnetPool)
						if !ok || pool == nil || pool.ID != "wire-id" || [3]int{pool.DefaultPrefixLen, pool.MinPrefixLen, pool.MaxPrefixLen} != check.want {
							t.Fatal("typed native prefix result changed", value, check.want)
						}
					})
				}
			}
		})
	}
}

func TestNeutronPythonGetFieldsAcceptsSourceResponseStatuses(t *testing.T) {
	for _, fixture := range neutronPythonReadFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			for _, code := range []int{200, 201, 203, 299, 300, 399} {
				t.Run(fmt.Sprint(code), func(t *testing.T) {
					cloud := testcloud.New(t)
					cloud.Mux.HandleFunc("GET "+neutronPythonMemberBase+fixture.path+"/target", func(w http.ResponseWriter, r *http.Request) {
						if !reflect.DeepEqual(r.URL.Query(), url.Values{"fields": {"id"}}) {
							t.Errorf("field query drifted: %s", r.URL)
						}
						testcloud.JSON(w, code, `{"`+fixture.envelope+`":{"id":"wire-id"}}`)
					})
					value, err := fixture.get(context.Background(), neutronPythonMemberClient(cloud), "target", "id")
					if err != nil || neutronPythonReadID(t, value) != "wire-id" {
						t.Fatal("dedicated Python-profile GET rejected Source success", code, value, err)
					}
				})
			}
		})
	}
}

func TestNeutronPythonGetFieldsKeepsNativeDecodeStatusAndWireCauses(t *testing.T) {
	for _, fixture := range neutronPythonReadFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			for _, code := range []int{404, 412, 500} {
				t.Run(fmt.Sprint(code), func(t *testing.T) {
					cloud := testcloud.New(t)
					cloud.Mux.HandleFunc("GET "+neutronPythonMemberBase+fixture.path+"/target", func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("X-Request-Proof", "get-failure")
						testcloud.JSON(w, code, `{"error":"get-denied"}`)
					})
					_, err := fixture.get(context.Background(), neutronPythonMemberClient(cloud), "target", "id")
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != code || string(native.Body) != `{"error":"get-denied"}` || native.ResponseHeader.Get("X-Request-Proof") != "get-failure" || native.Method != http.MethodGet || !strings.Contains(native.URL, "/target?fields=id") {
						t.Fatalf("native GET evidence lost: %+v err=%v", native, err)
					}
				})
			}
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("GET "+neutronPythonMemberBase+fixture.path+"/target", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"`+fixture.envelope+`":{"id":{"wrong":"type"}}}`)
			})
			_, err := fixture.get(context.Background(), neutronPythonMemberClient(cloud), "target", "id")
			var decode *json.UnmarshalTypeError
			if !errors.As(err, &decode) {
				t.Fatal("native typed decode failure was lost", err)
			}
			wireFailure := errors.New("member GET wire failure")
			cloud.SetTransport(testcloud.RoundTripperFunc(func(*http.Request) (*http.Response, error) { return nil, wireFailure }))
			_, err = fixture.get(context.Background(), neutronPythonMemberClient(cloud), "target", "id")
			if !errors.Is(err, wireFailure) {
				t.Fatal("GET transport cause was lost", err)
			}
		})
	}
}

func TestNeutronPythonGetFieldsCancellationStopsBeforeHTTP(t *testing.T) {
	for _, fixture := range neutronPythonReadFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			for _, withCause := range []bool{false, true} {
				t.Run(fmt.Sprintf("precanceled/cause=%t", withCause), func(t *testing.T) {
					cloud := testcloud.New(t)
					var calls atomic.Int32
					base := cloud.Provider.HTTPClient.Transport
					cloud.SetTransport(testcloud.RoundTripperFunc(func(r *http.Request) (*http.Response, error) { calls.Add(1); return base.RoundTrip(r) }))
					ctx, cancel := context.WithCancelCause(context.Background())
					cause := errors.New("member GET custom cancellation")
					if withCause {
						cancel(cause)
					} else {
						cancel(nil)
					}
					_, err := fixture.get(ctx, neutronPythonMemberClient(cloud), "target", fixture.fields...)
					if !errors.Is(err, context.Canceled) || withCause && !errors.Is(err, cause) || calls.Load() != 0 {
						t.Fatal("precanceled GET lost cause or reached HTTP", calls.Load(), err)
					}
				})
			}
			for _, code := range []int{200, 404} {
				for _, mode := range []string{"body", "transport", "wire"} {
					t.Run(fmt.Sprintf("%d/cancel-%s", code, mode), func(t *testing.T) {
						cloud := testcloud.New(t)
						body := `{"` + fixture.envelope + `":{"id":"wire-id"}}`
						if code == 404 {
							body = `{"error":"gone"}`
						}
						var calls, closes atomic.Int32
						cloud.Mux.HandleFunc("GET "+neutronPythonMemberBase+fixture.path+"/target", func(w http.ResponseWriter, r *http.Request) {
							calls.Add(1)
							w.Header().Set("X-Request-Proof", "canceled-get")
							testcloud.JSON(w, code, body)
						})
						ctx, cancel := context.WithCancelCause(context.Background())
						defer cancel(nil)
						cause, wireFailure := errors.New("member GET canceled after response"), errors.New("member GET concurrent wire failure")
						base := cloud.Provider.HTTPClient.Transport
						cloud.SetTransport(testcloud.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
							if mode == "wire" {
								cancel(cause)
								return nil, wireFailure
							}
							response, err := base.RoundTrip(r)
							if err != nil {
								return response, err
							}
							if mode == "body" {
								response.Body = &acceleratorResponseBoundaryBody{ReadCloser: response.Body, cancel: func() { cancel(cause) }, closes: &closes}
							} else {
								cancel(cause)
							}
							return response, nil
						}))
						value, err := fixture.get(ctx, neutronPythonMemberClient(cloud), "target", "id")
						if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
							t.Fatal("GET response/transport cancellation lost cause", value, err)
						}
						if mode == "wire" {
							if calls.Load() != 0 || !errors.Is(err, wireFailure) {
								t.Fatal("cancel + wire failure was flattened", calls.Load(), err)
							}
							return
						}
						if calls.Load() != 1 || mode == "body" && closes.Load() != 1 {
							t.Fatal("GET canceled physical response was replayed or not closed", calls.Load(), closes.Load(), err)
						}
						if code == 404 {
							var native gophercloud.ErrUnexpectedResponseCode
							if !errors.As(err, &native) || native.Actual != 404 || native.ResponseHeader.Get("X-Request-Proof") != "canceled-get" || string(native.Body) != body {
								t.Fatal("canceled physical404 evidence lost", native, err)
							}
						}
					})
				}
			}
		})
	}
}

func TestNeutronPythonGetFieldsSnapshotAndConcurrentReuse(t *testing.T) {
	for _, fixture := range neutronPythonReadFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			input := []string{"id", fixture.fields[0], "id"}
			want := []string{"id", fixture.wireFields[0], "id"}
			cloud.Mux.HandleFunc("GET "+neutronPythonMemberBase+fixture.path+"/target", func(w http.ResponseWriter, r *http.Request) {
				if !reflect.DeepEqual(r.URL.Query()["fields"], want) {
					t.Errorf("request field snapshot drifted: %s", r.URL)
				}
				testcloud.JSON(w, 200, `{"`+fixture.envelope+`":{"id":"wire-id"}}`)
			})
			base := cloud.Provider.HTTPClient.Transport
			cloud.SetTransport(testcloud.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				// Mutate caller memory only after request construction. A successful
				// response must not trigger another evaluation or overwrite that input.
				input[0] = "caller-after-request"
				return base.RoundTrip(r)
			}))
			value, err := fixture.get(context.Background(), neutronPythonMemberClient(cloud), "target", input...)
			if err != nil || neutronPythonReadID(t, value) != "wire-id" || input[0] != "caller-after-request" {
				t.Fatal(value, input, err)
			}
			cloud.SetTransport(base)
			input[0] = "id"
			client := neutronPythonMemberClient(cloud)
			var wait sync.WaitGroup
			for range 8 {
				wait.Add(1)
				go func() {
					defer wait.Done()
					value, err := fixture.get(context.Background(), client, "target", input...)
					if err != nil {
						t.Error(err)
						return
					}
					if neutronPythonReadID(t, value) != "wire-id" {
						t.Error(value)
					}
				}()
			}
			wait.Wait()
			if !reflect.DeepEqual(input, []string{"id", fixture.fields[0], "id"}) {
				t.Fatal("concurrent reads mutated caller fields", input)
			}
		})
	}
}

type neutronPythonDeleteFixture struct {
	name, path string
	remove     func(context.Context, *gophercloud.ServiceClient, string, ...request.NeutronDeleteOption) error
	revision   func(int64) request.NeutronDeleteOption
	missing    func(bool) request.NeutronDeleteOption
	bulk       func(request.NeutronDeleteOpts) request.NeutronDeleteOption
}

func neutronPythonDeleteFixtures() []neutronPythonDeleteFixture {
	return []neutronPythonDeleteFixture{
		{"networks", "networks", func(ctx context.Context, c *gophercloud.ServiceClient, id string, opts ...request.NeutronDeleteOption) error {
			return networks.New(c).DeleteWithOptions(ctx, id, opts...)
		}, networks.WithDeleteRevision, networks.WithDeleteIgnoreMissing, networks.WithDeleteOptions},
		{"subnets", "subnets", func(ctx context.Context, c *gophercloud.ServiceClient, id string, opts ...request.NeutronDeleteOption) error {
			return subnets.New(c).DeleteWithOptions(ctx, id, opts...)
		}, subnets.WithDeleteRevision, subnets.WithDeleteIgnoreMissing, subnets.WithDeleteOptions},
		{"ports", "ports", func(ctx context.Context, c *gophercloud.ServiceClient, id string, opts ...request.NeutronDeleteOption) error {
			return ports.New(c).DeleteWithOptions(ctx, id, opts...)
		}, ports.WithDeleteRevision, ports.WithDeleteIgnoreMissing, ports.WithDeleteOptions},
		{"floatingips", "floatingips", func(ctx context.Context, c *gophercloud.ServiceClient, id string, opts ...request.NeutronDeleteOption) error {
			return floatingips.New(c).DeleteWithOptions(ctx, id, opts...)
		}, floatingips.WithDeleteRevision, floatingips.WithDeleteIgnoreMissing, floatingips.WithDeleteOptions},
		{"routers", "routers", func(ctx context.Context, c *gophercloud.ServiceClient, id string, opts ...request.NeutronDeleteOption) error {
			return routers.New(c).DeleteWithOptions(ctx, id, opts...)
		}, routers.WithDeleteRevision, routers.WithDeleteIgnoreMissing, routers.WithDeleteOptions},
		{"security-groups", "security-groups", func(ctx context.Context, c *gophercloud.ServiceClient, id string, opts ...request.NeutronDeleteOption) error {
			return groups.New(c).DeleteWithOptions(ctx, id, opts...)
		}, groups.WithDeleteRevision, groups.WithDeleteIgnoreMissing, groups.WithDeleteOptions},
		{"security-group-rules", "security-group-rules", func(ctx context.Context, c *gophercloud.ServiceClient, id string, opts ...request.NeutronDeleteOption) error {
			return rules.New(c).DeleteWithOptions(ctx, id, opts...)
		}, rules.WithDeleteRevision, rules.WithDeleteIgnoreMissing, rules.WithDeleteOptions},
	}
}

func TestNeutronPythonDeleteRevisionMissingPolicyAndNativeRoutes(t *testing.T) {
	for _, fixture := range neutronPythonDeleteFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			for _, check := range []struct {
				name, condition string
				revision        *int64
			}{
				{"omitted", "configured-condition", nil}, {"zero", "revision_number=0", neutronPythonInt64(0)}, {"negative", "revision_number=-7", neutronPythonInt64(-7)}, {"max-int64", "revision_number=9223372036854775807", neutronPythonInt64(9223372036854775807)},
			} {
				t.Run(check.name, func(t *testing.T) {
					cloud := testcloud.New(t)
					var calls atomic.Int32
					cloud.Mux.HandleFunc("DELETE "+neutronPythonMemberBase+fixture.path+"/target", func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						body, _ := io.ReadAll(r.Body)
						if r.URL.RawQuery != "" || len(body) != 0 || r.Header.Get("If-Match") != check.condition || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Vendor") != "configured" {
							t.Errorf("DELETE changed target/body/revision: %s %s %#v", r.URL, body, r.Header)
						}
						w.WriteHeader(202)
						_, _ = w.Write([]byte("acknowledgement is not JSON"))
					})
					var options []request.NeutronDeleteOption
					if check.revision != nil {
						options = append(options, fixture.revision(*check.revision))
					}
					client := neutronPythonMemberClient(cloud)
					client.MoreHeaders = map[string]string{"iF-mAtCh": "configured-condition", "X-Vendor": "configured"}
					if err := fixture.remove(context.Background(), client, "target", options...); err != nil || calls.Load() != 1 {
						t.Fatal(calls.Load(), err)
					}
					if client.MoreHeaders["iF-mAtCh"] != "configured-condition" || len(client.MoreHeaders) != 2 {
						t.Fatal("delete mutated configured client headers", client.MoreHeaders)
					}
				})
			}
		})
	}
}

func neutronPythonInt64(value int64) *int64 { return &value }

func TestNeutronPythonDeleteResponseStatusesAndCauses(t *testing.T) {
	for _, fixture := range neutronPythonDeleteFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			for _, check := range []struct {
				code   int
				strict bool
			}{
				{200, false}, {201, false}, {202, false}, {204, false}, {299, false}, {300, false}, {399, false}, {404, false}, {404, true}, {403, false}, {412, false}, {500, false},
			} {
				t.Run(fmt.Sprintf("%d/strict=%t", check.code, check.strict), func(t *testing.T) {
					cloud := testcloud.New(t)
					body := `{"error":"delete-denied","revision":7}`
					cloud.Mux.HandleFunc("DELETE "+neutronPythonMemberBase+fixture.path+"/target", func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("X-Request-Proof", "delete-failure")
						testcloud.JSON(w, check.code, body)
					})
					options := []request.NeutronDeleteOption{fixture.revision(7)}
					if check.strict {
						options = append(options, fixture.missing(false))
					}
					err := fixture.remove(context.Background(), neutronPythonMemberClient(cloud), "target", options...)
					if check.code < 400 || check.code == 404 && !check.strict {
						if err != nil {
							t.Fatal("Python DELETE success/default missing policy was lost", err)
						}
						return
					}
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != check.code || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Proof") != "delete-failure" || native.Method != http.MethodDelete || !strings.HasSuffix(native.URL, neutronPythonMemberBase+fixture.path+"/target") {
						t.Fatalf("native DELETE evidence lost: %+v err=%v", native, err)
					}
				})
			}
			for _, strict := range []bool{false, true} {
				for _, mode := range []string{"read", "close", "read-and-close"} {
					t.Run(fmt.Sprintf("404/%s/strict=%t", mode, strict), func(t *testing.T) {
						cloud := testcloud.New(t)
						body := `{"error":"gone","revision":7}`
						cloud.Mux.HandleFunc("DELETE "+neutronPythonMemberBase+fixture.path+"/target", func(w http.ResponseWriter, r *http.Request) {
							w.Header().Set("X-Request-Proof", "faulted-delete")
							testcloud.JSON(w, 404, body)
						})
						readFailure, closeFailure := error(nil), error(nil)
						if mode != "close" {
							readFailure = io.ErrUnexpectedEOF
						}
						if mode != "read" {
							closeFailure = errors.New("404 DELETE Close failure")
						}
						track := payloadContractTrack(cloud, readFailure, closeFailure)
						var options []request.NeutronDeleteOption
						if strict {
							options = append(options, fixture.missing(false))
						}
						err := fixture.remove(context.Background(), neutronPythonMemberClient(cloud), "target", options...)
						var native gophercloud.ErrUnexpectedResponseCode
						physical := track.last(t)
						if err == nil || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Proof") != "faulted-delete" || readFailure != nil && !errors.Is(err, readFailure) || closeFailure != nil && !errors.Is(err, closeFailure) || physical.reads.Load() == 0 || physical.closes.Load() != 1 || track.calls.Load() != 1 {
							t.Fatal("mixed404 read/Close failure became absence or lost proof", native, err, physical, track.calls.Load())
						}
					})
				}
			}
			t.Run("accepted-close-containing404", func(t *testing.T) {
				cloud := testcloud.New(t)
				body := "accepted opaque acknowledgement"
				cloud.Mux.HandleFunc("DELETE "+neutronPythonMemberBase+fixture.path+"/target", func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Request-Proof", "accepted-delete")
					w.WriteHeader(202)
					_, _ = w.Write([]byte(body))
				})
				nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Method: http.MethodDelete, URL: "nested-body-close", Body: []byte("callback404")}
				track := payloadContractTrack(cloud, nil, nested)
				err := fixture.remove(context.Background(), neutronPythonMemberClient(cloud), "target")
				var proof *resource.ResponseError
				physical := track.last(t)
				if err == nil || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != body || proof.Header.Get("X-Request-Proof") != "accepted-delete" || !gophercloud.ResponseCodeIs(err, 404) || physical.closes.Load() != 1 || track.calls.Load() != 1 {
					t.Fatal("accepted callback404 became missing or lost202 proof", proof, err, physical)
				}
			})
		})
	}
}

func TestNeutronPythonDeleteOptionsFreezeCallbacksAndReusableBulk(t *testing.T) {
	for _, fixture := range neutronPythonDeleteFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var condition atomic.Value
			condition.Store("revision_number=4")
			cloud.Mux.HandleFunc("DELETE "+neutronPythonMemberBase+fixture.path+"/target", func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("If-Match") != condition.Load().(string) {
					t.Errorf("option snapshot/later value lost: %#v", r.Header)
				}
				testcloud.JSON(w, 404, `{"error":"gone"}`)
			})
			client := neutronPythonMemberClient(cloud)
			var retained *request.NeutronDeleteOpts
			custom := request.NeutronDeleteOption(func(opts *request.NeutronDeleteOpts) error {
				revision, ignored := int64(4), true
				opts.Revision, opts.IgnoreMissing = &revision, &ignored
				retained = opts
				return nil
			})
			base := cloud.Provider.HTTPClient.Transport
			cloud.SetTransport(testcloud.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				*retained.Revision = 99
				*retained.IgnoreMissing = false
				return base.RoundTrip(r)
			}))
			if err := fixture.remove(context.Background(), client, "target", custom); err != nil {
				t.Fatal("retained custom option changed in-flight missing policy", err)
			}
			cloud.SetTransport(base)
			revision, ignored := int64(4), true
			bulk := fixture.bulk(request.NeutronDeleteOpts{Revision: &revision, IgnoreMissing: &ignored})
			revision, ignored = 99, false
			var wait sync.WaitGroup
			for range 8 {
				wait.Add(1)
				go func() {
					defer wait.Done()
					if err := fixture.remove(context.Background(), client, "target", bulk); err != nil {
						t.Error("reused owned bulk option drifted", err)
					}
				}()
			}
			wait.Wait()
			condition.Store("")
			if err := fixture.remove(context.Background(), client, "target", fixture.revision(9), fixture.missing(false), fixture.bulk(request.NeutronDeleteOpts{})); err != nil {
				t.Fatal("later bulk did not replace previous fields/defaults", err)
			}
			condition.Store("revision_number=4")
			if err := fixture.remove(context.Background(), client, "target", fixture.revision(0), fixture.revision(4), fixture.missing(false), fixture.missing(true)); err != nil {
				t.Fatal("later scalar options did not win", err)
			}
		})
	}
}

func TestNeutronPythonDeleteCancellationWireFailuresAndInvalidOptions(t *testing.T) {
	for _, fixture := range neutronPythonDeleteFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("DELETE "+neutronPythonMemberBase+fixture.path+"/target", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(204) })
			client := neutronPythonMemberClient(cloud)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := fixture.remove(ctx, client, "target"); !errors.Is(err, context.Canceled) {
				t.Fatal("DELETE cancellation was lost", err)
			}
			optionFailure := errors.New("delete option rejected")
			for _, option := range []request.NeutronDeleteOption{nil, func(*request.NeutronDeleteOpts) error { return optionFailure }} {
				err := fixture.remove(context.Background(), client, "target", option)
				if err == nil || option != nil && !errors.Is(err, optionFailure) {
					t.Fatal("invalid option cause was lost", err)
				}
			}
			if calls.Load() != 0 {
				t.Fatal("canceled/invalid calls reached HTTP", calls.Load())
			}
			wireFailure := errors.New("DELETE wire failure")
			cloud.SetTransport(testcloud.RoundTripperFunc(func(*http.Request) (*http.Response, error) { return nil, wireFailure }))
			if err := fixture.remove(context.Background(), client, "target"); !errors.Is(err, wireFailure) {
				t.Fatal("DELETE transport failure was lost", err)
			}
			ctx, cancel = context.WithCancel(context.Background())
			cloud.SetTransport(testcloud.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				cancel()
				return &http.Response{StatusCode: 404, Header: http.Header{"X-Request-Proof": {"cancel-and-404"}}, Body: io.NopCloser(strings.NewReader(`{"error":"gone"}`)), Request: r}, nil
			}))
			if err := fixture.remove(ctx, client, "target"); !errors.Is(err, context.Canceled) {
				t.Fatal("default ignore-missing suppressed caller cancellation", err)
			}
		})
	}
}
