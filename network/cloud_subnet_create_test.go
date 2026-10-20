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
	networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func subnetCloudGET(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	switch {
	case strings.Contains(r.URL.Path, "/networks/"):
		testcloud.JSON(w, 200, `{"network":{"id":"actual-network"}}`)
	case strings.Contains(r.URL.Path, "/subnetpools/"):
		testcloud.JSON(w, 200, `{"subnetpool":{"id":"actual-pool"}}`)
	default:
		return false
	}
	return true
}
func subnetCloudAssertField(t *testing.T, fields map[string]json.RawMessage, key, want string) {
	t.Helper()
	if string(fields[key]) != want {
		t.Fatalf("%s=%s, want %s", key, fields[key], want)
	}
}
func TestCloudSubnetCreateDefaultsOwnedKeywordsRawBodyAndIndependentEvidence(t *testing.T) {
	var posted map[string]json.RawMessage
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if subnetCloudGET(w, r) {
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/neutron/v2.0/subnets" {
			t.Error(r.Method, r.URL)
		}
		posted = qosCloudCreateBody(t, r, "subnet")
		testcloud.JSON(w, 201, `{"subnet":{"id":null,"x_wire":7}}`)
	})
	raw := json.RawMessage(`{"cidr":"10.0.0.0/24","x_seed":{"n":9007199254740993}}`)
	option := network.WithSubnetCreateJSON(raw)
	raw[2] = 'X'
	result, err := service.CreateSubnetRecord(context.Background(), "requested", option)
	if err != nil || result == nil || result.Resource == nil || result.Wire == nil || result.Network == nil || result.SubnetPool != nil || result.StatusCode != 201 || count.Load() != 2 {
		t.Fatal(result, err, count.Load())
	}
	for key, want := range map[string]string{"network_id": `"actual-network"`, "ip_version": "4", "enable_dhcp": "false", "cidr": `"10.0.0.0/24"`, "x_seed": `{"n":9007199254740993}`} {
		subnetCloudAssertField(t, posted, key, want)
	}
	for key, want := range map[string]string{"id": "null", "ip_version": "4", "is_dhcp_enabled": "false", "location": "null", "tags": "[]", "gateway_ip": "null", "x_seed": `{"n":9007199254740993}`} {
		subnetCloudAssertField(t, result.Resource.Body, key, want)
	}
	if result.Wire.Body["ip_version"] != nil || result.Wire.Body["x_seed"] != nil || string(result.Envelope) != `{"subnet":{"id":null,"x_wire":7}}` {
		t.Fatal(result)
	}
	result.Header.Set("X-Request-Id", "poison")
	result.Envelope[0] = 'X'
	result.Resource.Body["x_wire"] = json.RawMessage("9")
	if result.Wire.Header.Get("X-Request-Id") != "qos-mutation-actual" || result.Network.Header.Get("X-Request-Id") != "qos-mutation-actual" || string(result.Wire.Body["x_wire"]) != "7" {
		t.Fatal("aliased observations", result)
	}
}
func TestCloudSubnetCreateNetworkLookupPrecedesAllSemanticValidation(t *testing.T) {
	for _, raw := range []string{`{}`, `{"disable_gateway_ip":true,"gateway_ip":"1"}`, `{"cidr":"x","subnetpool_name_or_id":"pool"}`, `{"use_default_subnetpool":true,"subnetpool_name_or_id":"pool"}`, `{"ip_version":"bad"}`} {
		var paths []string
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			paths = append(paths, r.URL.Path)
			if r.Method != http.MethodGet {
				t.Fatal(r.Method)
			}
			if strings.HasSuffix(r.URL.Path, "/networks") {
				testcloud.JSON(w, 200, `{"networks":[]}`)
			} else {
				testcloud.JSON(w, 404, `{}`)
			}
		})
		result, err := service.CreateSubnetRecord(context.Background(), "missing", network.WithSubnetCreateJSON(json.RawMessage(raw)))
		if result == nil || !errors.Is(err, resource.ErrNotFound) || count.Load() != 2 || !reflect.DeepEqual(paths, []string{"/neutron/v2.0/networks/missing", "/neutron/v2.0/networks"}) {
			t.Fatal(raw, result, err, count.Load(), paths)
		}
	}
	for _, tc := range []struct{ raw, message string }{{`{}`, "cidr is required"}, {`{"disable_gateway_ip":true,"gateway_ip":"1"}`, "disable_gateway_ip"}, {`{"cidr":"x","use_default_subnetpool":true}`, "cidr and subnetpool"}, {`{"use_default_subnetpool":true,"subnetpool_name_or_id":"pool"}`, "use_default_subnetpool and"}} {
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if !subnetCloudGET(w, r) {
				t.Fatal("validation reached POST", r.URL)
			}
		})
		result, err := service.CreateSubnetRecord(context.Background(), "requested", network.WithSubnetCreateJSON(json.RawMessage(tc.raw)))
		if result == nil || result.Network == nil || !errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(err.Error(), tc.message) || count.Load() != 1 {
			t.Fatal(tc, result, err, count.Load())
		}
	}
}
func TestCloudSubnetCreateFalseyFormalsTenantPresenceAndGatewayNull(t *testing.T) {
	for _, raw := range []string{`null`, `false`, `0`, `""`, `[]`, `{}`} {
		var posted map[string]json.RawMessage
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				q := r.URL.Query()
				if raw == "null" && q.Has("tenant_id") {
					t.Error(q)
				}
				if raw == "false" && q.Get("tenant_id") != "False" {
					t.Error(q)
				}
				if raw == "0" && q.Get("tenant_id") != "0" {
					t.Error(q)
				}
				if raw == `""` && (!q.Has("tenant_id") || q.Get("tenant_id") != "") {
					t.Error(q)
				}
				testcloud.JSON(w, 200, `{"network":{"id":"actual-network"}}`)
				return
			}
			posted = qosCloudCreateBody(t, r, "subnet")
			testcloud.JSON(w, 201, `{"subnet":{"id":"new"}}`)
		})
		kwargs := json.RawMessage(`{"cidr":"x","tenant_id":` + raw + `,"subnet_name":` + raw + `,"allocation_pools":` + raw + `,"gateway_ip":` + raw + `,"dns_nameservers":` + raw + `,"host_routes":` + raw + `,"ipv6_ra_mode":` + raw + `,"ipv6_address_mode":` + raw + `,"prefixlen":` + raw + `}`)
		result, err := service.CreateSubnetRecord(context.Background(), "requested", network.WithSubnetCreateJSON(kwargs))
		if err != nil || result == nil || count.Load() != 2 || len(posted) != 4 {
			t.Fatal(raw, result, err, count.Load(), posted)
		}
	}
	var posted map[string]json.RawMessage
	service, _ := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if subnetCloudGET(w, r) {
			return
		}
		posted = qosCloudCreateBody(t, r, "subnet")
		testcloud.JSON(w, 201, `{"subnet":{}}`)
	})
	result, err := service.CreateSubnetRecord(context.Background(), "requested", network.WithSubnetCreateCIDR("x"), network.WithSubnetCreateGatewayIP(""), network.WithSubnetCreateDisableGateway(true))
	if err != nil || result == nil || string(posted["gateway_ip"]) != "null" || string(result.Resource.Body["gateway_ip"]) != "null" {
		t.Fatal(result, err, posted)
	}
}
func TestCloudSubnetCreateSubnetPoolLookupThenPythonStringIPConversion(t *testing.T) {
	for _, tc := range []struct{ raw, body, view string }{{`" +٤ "`, "4", "4"}, {`"4_0"`, "40", "40"}, {`"４"`, "4", "4"}, {`"-4"`, "-4", "-4"}, {`null`, "null", "null"}, {`false`, "false", "false"}, {`[]`, "[]", "0"}, {`4.9`, "4.9", "4"}} {
		var paths []string
		var posted map[string]json.RawMessage
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			paths = append(paths, r.URL.Path)
			if subnetCloudGET(w, r) {
				return
			}
			posted = qosCloudCreateBody(t, r, "subnet")
			testcloud.JSON(w, 201, `{"subnet":{}}`)
		})
		result, err := service.CreateSubnetRecord(context.Background(), "requested", network.WithSubnetCreateSubnetPool("pool"), network.WithSubnetCreateIPVersion(json.RawMessage(tc.raw)))
		if err != nil || result == nil || result.SubnetPool == nil || count.Load() != 3 || paths[1] != "/neutron/v2.0/subnetpools/pool" {
			t.Fatal(tc, result, err, paths)
		}
		subnetCloudAssertField(t, posted, "ip_version", tc.body)
		subnetCloudAssertField(t, posted, "subnetpool_id", `"actual-pool"`)
		subnetCloudAssertField(t, result.Resource.Body, "ip_version", tc.view)
	}
	for _, raw := range []string{`"4.0"`, `"²"`} {
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if !subnetCloudGET(w, r) {
				t.Fatal("bad IP posted", r.URL)
			}
		})
		result, err := service.CreateSubnetRecord(context.Background(), "requested", network.WithSubnetCreateSubnetPool("pool"), network.WithSubnetCreateIPVersion(json.RawMessage(raw)))
		if result == nil || result.SubnetPool == nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != 2 {
			t.Fatal(raw, result, err, count.Load())
		}
	}
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/networks/") {
			subnetCloudGET(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/subnetpools") {
			testcloud.JSON(w, 200, `{"subnetpools":[]}`)
		} else {
			testcloud.JSON(w, 404, `{}`)
		}
	})
	result, err := service.CreateSubnetRecord(context.Background(), "requested", network.WithSubnetCreateSubnetPool("missing"), network.WithSubnetCreateIPVersion("bad"))
	if result == nil || !errors.Is(err, resource.ErrNotFound) || strings.Contains(err.Error(), "ip_version must") || count.Load() != 3 {
		t.Fatal(result, err, count.Load())
	}
}
func TestCloudSubnetCreateOrderedAliasesConflictRestorationAndRawDescriptorValues(t *testing.T) {
	for _, tc := range []struct{ raw, body, view string }{
		{`{"cidr":"x","is_dhcp_enabled":"False","allocation_pools":"ab","dns_nameservers":{"one":true},"prefix_length":"canonical","prefixlen":"formal","network_id":false}`, `{"network_id":false,"ip_version":4,"enable_dhcp":"False","allocation_pools":"ab","dns_nameservers":{"one":true},"prefixlen":"formal","cidr":"x"}`, `true`},
		{`{"cidr":"x","__conflicting_attrs":{"ip_version":"bad","network_id":null,"enable_dhcp":false,"x_extension":0}}`, `{"network_id":null,"ip_version":"bad","enable_dhcp":false,"cidr":"x","x_extension":0}`, `false`},
		{`{"cidr":"x","__conflicting_attrs":false}`, `{"network_id":"actual-network","ip_version":4,"enable_dhcp":false,"cidr":"x","__conflicting_attrs":false}`, `false`},
	} {
		var posted map[string]json.RawMessage
		service, _ := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if subnetCloudGET(w, r) {
				return
			}
			posted = qosCloudCreateBody(t, r, "subnet")
			testcloud.JSON(w, 201, `{"subnet":{}}`)
		})
		result, err := service.CreateSubnetRecord(context.Background(), "requested", network.WithSubnetCreateJSON(json.RawMessage(tc.raw)))
		var want map[string]json.RawMessage
		_ = json.Unmarshal([]byte(tc.body), &want)
		if err != nil || result == nil || !reflect.DeepEqual(posted, want) || string(result.Resource.Body["is_dhcp_enabled"]) != tc.view {
			t.Fatal(tc, result, err, posted, want)
		}
	}
}
func TestCloudSubnetCreateSourceProxyControlsHeadersAndOriginalBasePath(t *testing.T) {
	for _, tc := range []struct{ raw, path, header string }{{`{"cidr":"x","base_path":"/alternate","if_match":["a","b"],"microversion":"3.7"}`, "/neutron/v2.0/alternate", "a, b"}, {`{"cidr":"x","base_path":"","if_match":null}`, "/neutron/v2.0/", "None"}, {`{"cidr":"x","__conflicting_attrs":{"base_path":"/body-only"},"if_match":false}`, "/neutron/v2.0/subnets", "False"}} {
		var posted map[string]json.RawMessage
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if subnetCloudGET(w, r) {
				return
			}
			if r.URL.Path != tc.path || r.Header.Get("If-Match") != tc.header || r.Header.Get("OpenStack-API-Version") != "network 2.1" {
				t.Error(r.URL, r.Header)
			}
			posted = qosCloudCreateBody(t, r, "subnet")
			w.Header().Set("If-Match", "actual-response")
			testcloud.JSON(w, 201, `{"subnet":{"microversion":"foreign"}}`)
		})
		service.RawClient().Microversion = "2.1"
		result, err := service.CreateSubnetRecord(context.Background(), "requested", network.WithSubnetCreateJSON(json.RawMessage(tc.raw)))
		if err != nil || result == nil || count.Load() != 2 || posted["microversion"] != nil || posted["if_match"] != nil || string(result.Resource.Body["if_match"]) != `["actual-response"]` || string(result.Resource.Body["microversion"]) != `"2.1"` {
			t.Fatal(tc, result, err, posted, count.Load())
		}
		if strings.Contains(tc.raw, "body-only") {
			subnetCloudAssertField(t, posted, "base_path", `"/body-only"`)
			subnetCloudAssertField(t, result.Resource.Body, "base_path", `"/subnets"`)
		}
	}
	for _, raw := range []string{`{"cidr":"x","resource_type":null}`, `{"cidr":"x","cls":null}`, `{"cidr":"x","connection":null}`, `{"cidr":"x","_synchronized":null}`, `{"cidr":"x","if_match":[1]}`, `{"cidr":"x","__conflicting_attrs":true}`} {
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if !subnetCloudGET(w, r) {
				t.Fatal("control reached POST", r.URL)
			}
		})
		result, err := service.CreateSubnetRecord(context.Background(), "requested", network.WithSubnetCreateJSON(json.RawMessage(raw)))
		if result == nil || result.Network == nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != 1 {
			t.Fatal(raw, result, err, count.Load())
		}
	}
}
func TestCloudSubnetCreateResponseCanonicalAliasesDoNotLoseActualHeaderOrWire(t *testing.T) {
	service, _ := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if subnetCloudGET(w, r) {
			return
		}
		w.Header().Set("If-Match", "server")
		testcloud.JSON(w, 201, `{"subnet":{"id":null,"enable_dhcp":"False","is_dhcp_enabled":false,"network_id":null,"location":{"fake":true},"microversion":"foreign"}}`)
	})
	service.RawClient().Microversion = "2.1"
	result, err := service.CreateSubnetRecord(context.Background(), "requested", network.WithSubnetCreateCIDR("x"), network.WithSubnetCreateAttribute("enable_dhcp", "False"), network.WithSubnetCreateAttribute("if_match", []string{"seed"}))
	if err != nil || result == nil {
		t.Fatal(result, err)
	}
	for key, want := range map[string]string{"id": "null", "network_id": "null", "is_dhcp_enabled": "false", "if_match": `["server"]`, "location": "null", "microversion": `"2.1"`} {
		subnetCloudAssertField(t, result.Resource.Body, key, want)
	}
	subnetCloudAssertField(t, result.Wire.Body, "enable_dhcp", `"False"`)
	subnetCloudAssertField(t, result.Wire.Body, "is_dhcp_enabled", "false")
	subnetCloudAssertField(t, result.Wire.Body, "location", `{"fake":true}`)
}
func TestCloudSubnetCreateDefaultLookupsConsumeWholePagesAndKeepPartialFailure(t *testing.T) {
	for _, stage := range []string{"network", "pool"} {
		for _, late := range []string{"duplicate", "error"} {
			var posts atomic.Int32
			service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					posts.Add(1)
					t.Error("partial lookup posted")
					return
				}
				resourceName := "networks"
				member := "network"
				target := "requested"
				if stage == "pool" {
					resourceName = "subnetpools"
					member = "subnetpool"
					target = "pool"
					if strings.Contains(r.URL.Path, "/networks/") {
						subnetCloudGET(w, r)
						return
					}
				}
				if strings.Contains(r.URL.Path, "/"+resourceName+"/") {
					testcloud.JSON(w, 404, `{}`)
					return
				}
				if r.URL.Query().Get("marker") == "late" {
					if late == "error" {
						w.Header().Set("X-Request-Id", "late-lookup")
						testcloud.JSON(w, 503, `{"late":"failed"}`)
					} else {
						testcloud.JSON(w, 203, `{"`+resourceName+`":[{"id":"second","name":"`+target+`"}],"links":[]}`)
					}
					return
				}
				next := r.URL.Path + "?marker=late"
				testcloud.JSON(w, 203, fmt.Sprintf(`{"%s":[{"id":"first","name":%q}],"links":[{"rel":"next","href":%q}]}`, resourceName, target, next))
				_ = member
			})
			options := []network.SubnetCreateOption{network.WithSubnetCreateCIDR("x")}
			if stage == "pool" {
				options = []network.SubnetCreateOption{network.WithSubnetCreateSubnetPool("pool")}
			}
			result, err := service.CreateSubnetRecord(context.Background(), "requested", options...)
			if result == nil || err == nil || posts.Load() != 0 || count.Load() < 3 {
				t.Fatal(stage, late, result, err, count.Load())
			}
			if late == "duplicate" && !errors.Is(err, resource.ErrAmbiguous) {
				t.Fatal(err)
			}
			if late == "error" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 503 || native.ResponseHeader.Get("X-Request-Id") != "late-lookup" {
					t.Fatal(err, native)
				}
			}
		}
	}
}
func TestCloudSubnetCreateOwnedLocationUsesActualProjectsWithoutForgedWire(t *testing.T) {
	name := "dev"
	location := resource.CloudLocation{Cloud: &name, Project: resource.CloudProject{ID: json.RawMessage(`"current"`)}}
	option := network.WithSubnetCreateCurrentLocation(location)
	name = "changed"
	location.Project.ID[1] = 'X'
	service, _ := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			testcloud.JSON(w, 200, `{"network":{"id":"actual-network","project_id":"current","location":{"forged":true}}}`)
			return
		}
		testcloud.JSON(w, 201, `{"subnet":{"id":"new","project_id":{"owner":9007199254740993},"location":{"forged":true}}}`)
	})
	result, err := service.CreateSubnetRecord(context.Background(), "requested", option, network.WithSubnetCreateOpts(network.SubnetCreateOpts{Fields: map[string]json.RawMessage{"cidr": json.RawMessage(`"x"`)}}))
	if err != nil || result == nil {
		t.Fatal(result, err)
	}
	var selected resource.CloudLocation
	if json.Unmarshal(result.Resource.Body["location"], &selected) != nil || selected.Cloud == nil || *selected.Cloud != "dev" || string(selected.Project.ID) != `{"owner":9007199254740993}` {
		t.Fatal(string(result.Resource.Body["location"]))
	}
	subnetCloudAssertField(t, result.Wire.Body, "location", `{"forged":true}`)
	if !strings.Contains(string(result.Network.Resource.Body["location"]), `"current"`) {
		t.Fatal(result.Network.Resource)
	}
}

type subnetCreateObservedTransport struct {
	base              http.RoundTripper
	method            string
	status            int
	readErr, closeErr error
	closed            *atomic.Int32
	onClose           func()
}

func (o subnetCreateObservedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := o.base.RoundTrip(r)
	if response != nil && r.Method == o.method && ((o.method == http.MethodPost) || strings.HasSuffix(r.URL.Path, "/networks/requested")) {
		if o.status != 0 {
			response.StatusCode = o.status
			response.Status = fmt.Sprint(o.status)
		}
		response.Body = &qosPolicyActionObservedBody{ReadCloser: response.Body, readErr: o.readErr, closeErr: o.closeErr, closed: o.closed, onClose: o.onClose}
	}
	return response, err
}
func TestCloudSubnetCreatePhysicalFaultsKeepActualLookupOrMutationAndStopHooks(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		for _, status := range []int{200, 401, 404, 503} {
			for _, lane := range []string{"read-close", "cancel", "source"} {
				t.Run(fmt.Sprintf("%s/%d/%s", method, status, lane), func(t *testing.T) {
					body := `{"subnet":{"id":"new"}}`
					if method == http.MethodGet {
						body = `{"network":{"id":"actual-network"}}`
					}
					service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
						if method == http.MethodPost && subnetCloudGET(w, r) {
							return
						}
						testcloud.JSON(w, status, body)
					})
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					readErr, closeErr, cancelCause := errors.New("subnet actual read"), errors.New("subnet actual close"), errors.New("subnet caller cancellation")
					base := service.RawClient().HTTPClient.Transport
					if base == nil {
						base = http.DefaultTransport
					}
					closed := &atomic.Int32{}
					observer := subnetCreateObservedTransport{base: base, method: method, closed: closed}
					if lane == "read-close" {
						observer.readErr, observer.closeErr = readErr, closeErr
					}
					if lane == "cancel" {
						observer.onClose = func() { cancel(cancelCause) }
					}
					if lane == "source" {
						observer.onClose = func() { service.RawClient().ResourceBase += "drift/" }
					}
					service.RawClient().HTTPClient.Transport = observer
					hooks, reauth := 0, 0
					service.RawClient().ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
						hooks++
						return errors.New("unexpected physical retry")
					}
					service.RawClient().ProviderClient.ReauthFunc = func(context.Context) error { reauth++; return errors.New("unexpected physical reauth") }
					result, err := service.CreateSubnetRecord(ctx, "requested", network.WithSubnetCreateCIDR("x"))
					wantCalls := int32(2)
					if method == http.MethodGet {
						wantCalls = 1
					}
					if err == nil || result == nil || result.Resource != nil || result.Wire != nil || count.Load() != wantCalls || closed.Load() != 1 || hooks != 0 || reauth != 0 {
						t.Fatal(result, err, count.Load(), closed.Load(), hooks, reauth)
					}
					receiptStatus, receiptBody, receiptHeader := result.StatusCode, result.Envelope, result.Header
					if method == http.MethodGet {
						if result.Network == nil {
							t.Fatal(result, err)
						}
						receiptStatus, receiptBody, receiptHeader = result.Network.StatusCode, result.Network.Envelope, result.Network.Header
					}
					if receiptStatus != status || string(receiptBody) != body || receiptHeader.Get("X-Request-Id") != "qos-mutation-actual" {
						t.Fatal(receiptStatus, string(receiptBody), receiptHeader, err)
					}
					if lane == "read-close" && (!errors.Is(err, readErr) || !errors.Is(err, closeErr)) {
						t.Fatal(err)
					}
					if lane == "cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
						t.Fatal(err)
					}
					if lane == "source" && !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
					if status == 401 {
						var auth *gophercloud.ErrUnableToReauthenticate
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(err, &auth) || auth == nil || auth.ErrReauth == nil || !errors.As(auth.ErrOriginal, &native) || native.Actual != 401 || string(native.Body) != body {
							t.Fatal(err, auth, native)
						}
					}
				})
			}
		}
	}
}
func TestCloudSubnetCreateCleanRetryRenewalSource199AndFixedHeaderCapture(t *testing.T) {
	for _, lane := range []string{"retry", "renewal", "source199", "route"} {
		t.Run(lane, func(t *testing.T) {
			attempts := 0
			var tokens []string
			service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if subnetCloudGET(w, r) {
					return
				}
				attempts++
				tokens = append(tokens, r.Header.Get("X-Auth-Token"))
				if r.Header.Get("X-Trace") != "selected" {
					t.Error(r.Header)
				}
				if attempts == 1 && lane != "source199" {
					status := 503
					if lane == "renewal" {
						status = 401
					}
					testcloud.JSON(w, status, `{"rejected":true}`)
					return
				}
				testcloud.JSON(w, 201, `{"subnet":{"id":"new"}}`)
			})
			client := service.RawClient()
			client.MoreHeaders = map[string]string{"X-Trace": "selected"}
			hooks, renewals := 0, 0
			if lane == "retry" || lane == "route" {
				client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks++
					if lane == "route" {
						client.ResourceBase += "drift/"
					}
					return nil
				}
			}
			if lane == "renewal" {
				client.ProviderClient.ReauthFunc = func(context.Context) error { renewals++; client.ProviderClient.SetToken("renewed"); return nil }
			}
			if lane == "source199" {
				base := client.HTTPClient.Transport
				if base == nil {
					base = http.DefaultTransport
				}
				client.HTTPClient.Transport = subnetCreateObservedTransport{base: base, method: http.MethodPost, status: 199, closed: &atomic.Int32{}}
			}
			result, err := service.CreateSubnetRecord(context.Background(), "requested", network.WithSubnetCreateCIDR("x"), func(*network.SubnetCreateOpts) error { client.MoreHeaders["X-Trace"] = "later valid value"; return nil })
			if lane == "route" {
				if result == nil || !errors.Is(err, resource.ErrInvalidOption) || attempts != 1 || hooks != 1 || result.StatusCode != 503 {
					t.Fatal(result, err, attempts, hooks)
				}
				return
			}
			expectedAttempts := 2
			if lane == "source199" {
				expectedAttempts = 1
			}
			if err != nil || result == nil || result.Resource == nil || attempts != expectedAttempts || count.Load() != int32(expectedAttempts+1) {
				t.Fatal(result, err, attempts, count.Load())
			}
			if lane == "retry" && hooks != 1 {
				t.Fatal(hooks)
			}
			if lane == "renewal" && (renewals != 1 || tokens[1] != "renewed") {
				t.Fatal(renewals, tokens)
			}
			if lane == "source199" && result.StatusCode != 199 {
				t.Fatal(result.StatusCode)
			}
		})
	}
}
func TestCloudSubnetCreatePreflightBoundArgumentsCancellationAndImmutableService(t *testing.T) {
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) { t.Fatal("invalid preflight HTTP", r.URL) })
	for _, opts := range [][]network.SubnetCreateOption{{nil}, {network.WithSubnetCreateJSON(json.RawMessage(`[]`))}, {network.WithSubnetCreateAttribute("network_name_or_id", "other")}, {network.WithSubnetCreateAttribute("self", nil)}, {network.WithSubnetCreateAttribute("cidr", make(chan int))}, {network.WithSubnetCreateOpts(network.SubnetCreateOpts{Fields: map[string]json.RawMessage{"cidr": json.RawMessage(`{bad`)}})}} {
		result, err := service.CreateSubnetRecord(context.Background(), "requested", opts...)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != 0 {
			t.Fatal(result, err, count.Load())
		}
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("early subnet cancel")
	cancel(cause)
	result, err := service.CreateSubnetRecord(ctx, "requested", network.WithSubnetCreateCIDR("x"))
	if result != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || count.Load() != 0 {
		t.Fatal(result, err)
	}
	original := service.API
	result, err = service.CreateSubnetRecord(context.Background(), "requested", func(*network.SubnetCreateOpts) error { service.API = networkv2.New(service.RawClient()); return nil })
	service.API = original
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != 0 {
		t.Fatal(result, err, count.Load())
	}
	var missing *network.Service
	result, err = missing.CreateSubnetRecord(context.Background(), "requested")
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(result, err)
	}
}
func TestCloudSubnetCreateAcceptedMutationInvalidatesOnlyPrivateSuccessfulRoleCache(t *testing.T) {
	for _, lane := range []string{"success", "decode", "physical", "cancel", "rejected"} {
		t.Run(lane, func(t *testing.T) {
			var lists atomic.Int32
			service, _ := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					if strings.HasSuffix(r.URL.Path, "/networks") {
						lists.Add(1)
						testcloud.JSON(w, 200, `{"networks":[{"id":"actual-network","name":"private","subnets":[]}]}`)
					} else if strings.HasSuffix(r.URL.Path, "/subnets") {
						testcloud.JSON(w, 200, `{"subnets":[]}`)
					} else {
						subnetCloudGET(w, r)
					}
					return
				}
				if lane == "rejected" {
					testcloud.JSON(w, 403, `{"denied":true}`)
				} else if lane == "decode" {
					testcloud.JSON(w, 201, `{"subnet":null}`)
				} else {
					testcloud.JSON(w, 201, `{"subnet":{"id":"new"}}`)
				}
			})
			if _, err := service.Roles.Discover(context.Background()); err != nil {
				t.Fatal(err)
			}
			if _, err := service.Roles.Discover(context.Background()); err != nil || lists.Load() != 1 {
				t.Fatal(err, lists.Load())
			}
			originalRoles := service.Roles
			service.Roles = network.New(service.RawClient()).Roles // The private cached target must still be invalidated.
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			if lane == "physical" || lane == "cancel" {
				base := service.RawClient().HTTPClient.Transport
				if base == nil {
					base = http.DefaultTransport
				}
				observer := subnetCreateObservedTransport{base: base, method: http.MethodPost, closed: &atomic.Int32{}}
				if lane == "physical" {
					observer.readErr = errors.New("accepted physical failure")
				} else {
					observer.onClose = func() { cancel(errors.New("accepted cancellation")) }
				}
				service.RawClient().HTTPClient.Transport = observer
			}
			result, err := service.CreateSubnetRecord(ctx, "requested", network.WithSubnetCreateCIDR("x"))
			if result == nil || (lane == "success" && err != nil) || (lane != "success" && err == nil) {
				t.Fatal(result, err)
			}
			if _, err := originalRoles.Discover(context.Background()); err != nil {
				t.Fatal(err)
			}
			expected := int32(2)
			if lane == "rejected" {
				expected = 1
			}
			if lists.Load() != expected {
				t.Fatal(lane, lists.Load(), expected)
			}
		})
	}
}
