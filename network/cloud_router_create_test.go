package network_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Existing public-testhelper fixture and response-body observer are shared with
// cloud QoS tests; this wrapper only selects the router/extension observation.
type routerCloudObservedTransport struct {
	base              http.RoundTripper
	path              string
	status            int
	readErr, closeErr error
	closed            *atomic.Int32
	onClose           func()
}

func (o routerCloudObservedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := o.base.RoundTrip(r)
	if response != nil && r.URL.Path == o.path {
		if o.status != 0 {
			response.StatusCode = o.status
			response.Status = fmt.Sprint(o.status)
		}
		if o.closed != nil {
			response.Body = &qosPolicyActionObservedBody{ReadCloser: response.Body, readErr: o.readErr, closeErr: o.closeErr, closed: o.closed, onClose: o.onClose}
		}
	}
	return response, err
}
func routerCloudReply(t *testing.T, s *network.Service, count *atomic.Int32, options ...network.RouterCreateOption) (*network.RouterCreateResult, error) {
	t.Helper()
	result, err := s.CreateRouterRecord(context.Background(), options...)
	if err != nil || result == nil || result.Router == nil || count.Load() != 1 {
		t.Fatal(result, err, count.Load())
	}
	return result, err
}
func TestCloudRouterCreateDefaultAndSourceJSONPresenceTruthiness(t *testing.T) {
	for _, key := range []string{"name", "admin_state_up", "ext_gateway_net_id", "enable_snat", "ext_fixed_ips", "project_id"} {
		for _, raw := range []string{"null", "false", "0", `""`, "[]", "{}", `"False"`, "1e-400", "9007199254740993"} {
			t.Run(key+"/"+raw, func(t *testing.T) {
				var posted map[string]json.RawMessage
				service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodPost || r.URL.Path != "/neutron/v2.0/routers" || r.URL.RawQuery != "" {
						t.Error(r.Method, r.URL)
					}
					posted = qosCloudCreateBody(t, r, "router")
					testcloud.JSON(w, 201, `{"router":{}}`)
				})
				result, _ := routerCloudReply(t, service, count, network.WithRouterCreateJSON(json.RawMessage(`{"`+key+`":`+raw+`}`)))
				falsey := raw == "null" || raw == "false" || raw == "0" || raw == `""` || raw == "[]" || raw == "{}" || raw == "1e-400"
				switch key {
				case "name":
					if (!falsey) != (posted["name"] != nil) || !falsey && string(posted["name"]) != raw {
						t.Fatal(posted)
					}
				case "admin_state_up":
					if string(posted["admin_state_up"]) != raw {
						t.Fatal(posted)
					}
					want := "true"
					if falsey {
						want = "false"
					}
					if raw == "null" {
						want = "null"
					}
					if string(result.Router.Body["is_admin_state_up"]) != want {
						t.Fatal(result.Router)
					}
				case "project_id":
					if (raw != "null") != (posted["project_id"] != nil) || raw != "null" && string(posted["project_id"]) != raw {
						t.Fatal(posted)
					}
				default:
					var gateway map[string]json.RawMessage
					if posted["external_gateway_info"] != nil {
						if e := json.Unmarshal(posted["external_gateway_info"], &gateway); e != nil {
							t.Fatal(e)
						}
					}
					nested := map[string]string{"ext_gateway_net_id": "network_id", "enable_snat": "enable_snat", "ext_fixed_ips": "external_fixed_ips"}[key]
					include := !falsey
					if key == "enable_snat" {
						include = raw != "null"
					}
					if include != (gateway[nested] != nil) || include && string(gateway[nested]) != raw {
						t.Fatal(posted, gateway)
					}
				}
				if key != "admin_state_up" && string(posted["admin_state_up"]) != "true" {
					t.Fatal(posted)
				}
				if result.Wire == nil || len(result.Wire.Body) != 0 || string(result.Router.Body["location"]) != "null" || result.Router.Body["admin_state_up"] != nil {
					t.Fatal(result)
				}
			})
		}
	}
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		posted := qosCloudCreateBody(t, r, "router")
		if len(posted) != 1 || string(posted["admin_state_up"]) != "true" {
			t.Error(posted)
		}
		testcloud.JSON(w, 201, `{"router":{"id":"created"}}`)
	})
	routerCloudReply(t, service, count)
}
func TestCloudRouterCreateGatewayAssemblyAndOwnedConcreteOptions(t *testing.T) {
	raw := json.RawMessage(`{"name":"old","ext_gateway_net_id":"ext","enable_snat":false,"ext_fixed_ips":[],"project_id":""}`)
	option := network.WithRouterCreateJSON(raw)
	raw[2] = 'X'
	ips := []map[string]any{{"subnet_id": "sub", "ip_address": "192.0.2.8"}}
	fixed := network.WithRouterCreateExternalFixedIPs(ips)
	ips[0]["subnet_id"] = "modified"
	var posted map[string]json.RawMessage
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		posted = qosCloudCreateBody(t, r, "router")
		testcloud.JSON(w, 201, `{"router":{"id":"new"}}`)
	})
	result, _ := routerCloudReply(t, service, count, option, network.WithRouterCreateName("new"), network.WithRouterCreateAdminStateUp(false), fixed)
	if string(posted["name"]) != `"new"` || string(posted["admin_state_up"]) != "false" || string(posted["project_id"]) != `""` {
		t.Fatal(posted)
	}
	var gateway map[string]json.RawMessage
	if e := json.Unmarshal(posted["external_gateway_info"], &gateway); e != nil || string(gateway["network_id"]) != `"ext"` || string(gateway["enable_snat"]) != "false" || string(gateway["external_fixed_ips"]) != `[{"ip_address":"192.0.2.8","subnet_id":"sub"}]` {
		t.Fatal(gateway, e)
	}
	if result.Router.Body["name"] == nil || result.Wire.Body["name"] != nil || result.StatusCode != 201 {
		t.Fatal(result)
	}
	attrs := map[string]json.RawMessage{"name": json.RawMessage(`"owned"`)}
	snapshot := network.WithRouterCreateOpts(network.RouterCreateOpts{Attributes: attrs})
	attrs["name"][1] = 'X'
	count.Store(0)
	routerCloudReply(t, service, count, snapshot)
	if string(posted["name"]) != `"owned"` {
		t.Fatal(posted)
	}
}
func TestCloudRouterCreateAZListValidationThenFreshCompleteGate(t *testing.T) {
	for _, raw := range []string{"false", "0", `""`, "{}", `"zone"`, "1e-400"} {
		service, count := qosCloudMutationFixture(t, func(http.ResponseWriter, *http.Request) { t.Error("HTTP before invalid hints") })
		result, e := service.CreateRouterRecord(context.Background(), network.WithRouterCreateAttribute("availability_zone_hints", json.RawMessage(raw)))
		if result != nil || !errors.Is(e, resource.ErrInvalidOption) || count.Load() != 0 {
			t.Fatal(result, e, count.Load())
		}
	}
	for _, input := range []string{"null", "[]", `["zone",null,0]`} {
		var order []string
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				order = append(order, "gate")
				testcloud.JSON(w, 200, `{"extensions":[{"alias":"router_availability_zone"},{"alias":"other"}]}`)
				return
			}
			order = append(order, "post")
			posted := qosCloudCreateBody(t, r, "router")
			if input == "null" && posted["availability_zone_hints"] != nil || input != "null" && string(posted["availability_zone_hints"]) != input {
				t.Error(posted)
			}
			testcloud.JSON(w, 201, `{"router":{"id":"new"}}`)
		})
		for repeat := 0; repeat < 2; repeat++ {
			result, e := service.CreateRouterRecord(context.Background(), network.WithRouterCreateAttribute("availability_zone_hints", json.RawMessage(input)))
			want := []string{"post"}
			extCount := 0
			if input != "null" {
				want = []string{"gate", "post"}
				extCount = 2
			}
			if e != nil || result == nil || result.Router == nil || len(result.Extensions) != extCount || !reflect.DeepEqual(order, want) || count.Load() != int32(len(want)) {
				t.Fatal(result, e, order, count.Load())
			}
			order = nil
			count.Store(0)
		}
	}
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			testcloud.JSON(w, 200, `{"extensions":[{"alias":"router_availability_zone"}]}`)
			return
		}
		posted := qosCloudCreateBody(t, r, "router")
		if string(posted["availability_zone_hints"]) != "[]" {
			t.Error(posted)
		}
		testcloud.JSON(w, 201, `{"router":{}}`)
	})
	result, e := service.CreateRouterRecord(context.Background(), network.WithRouterCreateAvailabilityZoneHints())
	if e != nil || result == nil || count.Load() != 2 {
		t.Fatal(result, e, count.Load())
	}
}
func TestCloudRouterCreateAZMissingLateAliasAndPageFailuresPreserveOnlyGates(t *testing.T) {
	for _, mode := range []string{"missing", "late alias", "late page"} {
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/neutron/v2.0/extensions" {
				t.Fatal("POST after failed gate", r.Method, r.URL)
			}
			switch mode {
			case "missing":
				testcloud.JSON(w, 200, `{"extensions":[{"alias":"other"}]}`)
			case "late alias":
				testcloud.JSON(w, 200, `{"extensions":[{"alias":"router_availability_zone"},{"alias":[]}]}`)
			default:
				if r.URL.Query().Get("marker") == "tail" {
					testcloud.JSON(w, 403, `{"error":"late page"}`)
				} else {
					testcloud.JSON(w, 200, `{"extensions":[{"alias":"router_availability_zone"}],"extensions_links":[{"rel":"next","href":"?marker=tail"}]}`)
				}
			}
		})
		result, e := service.CreateRouterRecord(context.Background(), network.WithRouterCreateAvailabilityZoneHints())
		if result == nil || result.Router != nil || result.Wire != nil || result.StatusCode != 0 || len(result.Envelope) != 0 || len(result.Extensions) == 0 || e == nil {
			t.Fatal(result, e)
		}
		if mode == "missing" && !errors.Is(e, network.ErrUnavailableExtension) || mode == "late alias" && !errors.Is(e, resource.ErrInvalidOption) || mode == "late page" && (!gophercloud.ResponseCodeIs(e, 403) || count.Load() != 2) {
			t.Fatal(result, e, count.Load())
		}
	}
}
func TestCloudRouterCreateResponseSourceDefaultsAliasesAndOwnedLocation(t *testing.T) {
	body := `{"router":{"id":null,"admin_state_up":null,"tenant_id":"foreign","external_gateway_info":false,"availability_zone_hints":"z","tags":"tag","revision":"7","microversion":"foreign","location":{"forged":true},"x_new":9007199254740993}}`
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header()["If-Match"] = []string{"first", "second"}
		testcloud.JSON(w, 203, body)
	})
	service.RawClient().Microversion = "2.1"
	name, project := "configured", json.RawMessage(`"current"`)
	location := resource.CloudLocation{Cloud: &name, Project: resource.CloudProject{ID: project, Name: &name}}
	current := network.WithRouterCreateCurrentLocation(location)
	name = "changed"
	project[1] = 'X'
	result, _ := routerCloudReply(t, service, count, network.WithRouterCreateName("seed"), current)
	for key, want := range map[string]string{"id": "null", "name": `"seed"`, "is_admin_state_up": "null", "project_id": `"foreign"`, "tenant_id": `"foreign"`, "external_gateway_info": "{}", "availability_zone_hints": `["z"]`, "tags": `["tag"]`, "revision_number": "7", "microversion": `"2.1"`, "if_match": `["first, second"]`, "x_new": "9007199254740993"} {
		if string(result.Router.Body[key]) != want {
			t.Fatal(key, string(result.Router.Body[key]), want)
		}
	}
	var actual resource.CloudLocation
	if e := json.Unmarshal(result.Router.Body["location"], &actual); e != nil || actual.Cloud == nil || *actual.Cloud != "configured" || string(actual.Project.ID) != `"foreign"` || actual.Project.Name != nil {
		t.Fatal(actual, e)
	}
	if string(result.Wire.Body["location"]) != `{"forged":true}` || string(result.Wire.Body["microversion"]) != `"foreign"` || string(result.Envelope) != body || result.Router.Body["admin_state_up"] != nil {
		t.Fatal(result)
	}
	result.Router.Body["x_new"][0] = '1'
	result.Router.Header.Set("X-Request-Id", "changed")
	result.Header.Set("X-Request-Id", "changed")
	result.Envelope[0] = 'X'
	if string(result.Wire.Body["x_new"]) != "9007199254740993" || result.Wire.Header.Get("X-Request-Id") != "qos-mutation-actual" {
		t.Fatal(result)
	}
}
func TestCloudRouterCreatePreflightCancellationAndSelectedSourceGuards(t *testing.T) {
	service, count := qosCloudMutationFixture(t, func(http.ResponseWriter, *http.Request) { t.Error("invalid preflight sent HTTP") })
	for _, options := range [][]network.RouterCreateOption{{nil}, {network.WithRouterCreateJSON(json.RawMessage(`[]`))}, {network.WithRouterCreateJSON(json.RawMessage(`{bad`))}, {network.WithRouterCreateAttribute("tenant_id", make(chan int))}, {network.WithRouterCreateAttribute("is_admin_state_up", false)}, {network.WithRouterCreateAttribute("admin_state_up", make(chan int))}, {network.WithRouterCreateOpts(network.RouterCreateOpts{Attributes: map[string]json.RawMessage{"name": json.RawMessage(`bad`)}})}, {network.WithRouterCreateCurrentLocation(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`bad`)}})}} {
		result, e := service.CreateRouterRecord(context.Background(), options...)
		if result != nil || !errors.Is(e, resource.ErrInvalidOption) || count.Load() != 0 {
			t.Fatal(result, e, count.Load())
		}
	}
	var nilService *network.Service
	if result, e := nilService.CreateRouterRecord(context.Background()); result != nil || !errors.Is(e, resource.ErrInvalidOption) {
		t.Fatal(result, e)
	}
	if result, e := service.CreateRouterRecord(nil); result != nil || !errors.Is(e, resource.ErrInvalidOption) {
		t.Fatal(result, e)
	}
	cause := errors.New("caller cancel")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if result, e := service.CreateRouterRecord(ctx); result != nil || !errors.Is(e, context.Canceled) || !errors.Is(e, cause) || count.Load() != 0 {
		t.Fatal(result, e, count.Load())
	}
	originalAPI, originalEndpoint := service.API, service.RawClient().Endpoint
	for _, mode := range []string{"API", "route", "cancel"} {
		ctx, cancel := context.WithCancelCause(context.Background())
		option := network.RouterCreateOption(func(*network.RouterCreateOpts) error {
			switch mode {
			case "API":
				service.API = networkv2.New(service.RawClient())
			case "route":
				service.RawClient().Endpoint += "changed/"
			default:
				cancel(cause)
			}
			return nil
		})
		result, e := service.CreateRouterRecord(ctx, option)
		if result != nil || e == nil || count.Load() != 0 || mode == "cancel" && (!errors.Is(e, context.Canceled) || !errors.Is(e, cause)) || mode != "cancel" && !errors.Is(e, resource.ErrInvalidOption) {
			t.Fatal(result, e, count.Load())
		}
		service.API = originalAPI
		service.RawClient().Endpoint = originalEndpoint
		cancel(nil)
	}
}
func TestCloudRouterCreateRejectedStatusesAndInvalidResponsesKeepActualReceipt(t *testing.T) {
	for _, status := range []int{400, 403, 404, 409, 503} {
		body := `{"error":"router create rejected"}`
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, status, body) })
		result, e := service.CreateRouterRecord(context.Background())
		var native gophercloud.ErrUnexpectedResponseCode
		if result == nil || result.Router != nil || result.Wire != nil || result.StatusCode != status || string(result.Envelope) != body || result.Header.Get("X-Request-Id") != "qos-mutation-actual" || !errors.As(e, &native) || native.Method != http.MethodPost || native.Actual != status || count.Load() != 1 {
			t.Fatal(result, e, native, count.Load())
		}
	}
	for _, body := range []string{"", "broken", "{}", "null", "[]", `{"router":null}`, `{"router":[]}`} {
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 202, body) })
		result, e := service.CreateRouterRecord(context.Background(), network.WithRouterCreateName("seed"), network.WithRouterCreateAdminStateUp(false))
		bad := body == "null" || body == "[]" || body == `{"router":null}` || body == `{"router":[]}`
		if result == nil || result.StatusCode != 202 || string(result.Envelope) != body || count.Load() != 1 || bad && (e == nil || result.Router != nil) || !bad && (e != nil || result.Router == nil || string(result.Router.Body["name"]) != `"seed"` || string(result.Router.Body["is_admin_state_up"]) != "false") {
			t.Fatal(body, result, e, count.Load())
		}
	}
}
func TestCloudRouterCreatePhysicalReadCloseCancellationSourceAndNative401NoHooks(t *testing.T) {
	for _, status := range []int{200, 401, 404, 503} {
		for _, mode := range []string{"read-close", "cancel", "source"} {
			t.Run(fmt.Sprint(status, "/", mode), func(t *testing.T) {
				body := `{"router":{"id":"physical"}}`
				service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, status, body) })
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readErr, closeErr, cancelCause := errors.New("router physical read"), errors.New("router physical close"), errors.New("router cancellation")
				closed := &atomic.Int32{}
				base := service.RawClient().HTTPClient.Transport
				if base == nil {
					base = http.DefaultTransport
				}
				observe := routerCloudObservedTransport{base: base, path: "/neutron/v2.0/routers", closed: closed}
				switch mode {
				case "read-close":
					observe.readErr, observe.closeErr = readErr, closeErr
				case "cancel":
					observe.onClose = func() { cancel(cancelCause) }
				default:
					observe.onClose = func() { service.RawClient().ResourceBase += "changed/" }
				}
				service.RawClient().HTTPClient.Transport = observe
				retries, reauths := 0, 0
				service.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return errors.New("unexpected bounded retry")
				}
				service.RawClient().ReauthFunc = func(context.Context) error { reauths++; return errors.New("unexpected bounded reauth") }
				result, e := service.CreateRouterRecord(ctx)
				if e == nil || result == nil || result.Router != nil || result.Wire != nil || result.StatusCode != status || string(result.Envelope) != body || result.Header.Get("X-Request-Id") != "qos-mutation-actual" || count.Load() != 1 || closed.Load() != 1 || retries != 0 || reauths != 0 {
					t.Fatal(result, e, count.Load(), closed.Load(), retries, reauths)
				}
				if mode == "read-close" && (!errors.Is(e, readErr) || !errors.Is(e, closeErr)) || mode == "cancel" && (!errors.Is(e, context.Canceled) || !errors.Is(e, cancelCause)) || mode == "source" && !errors.Is(e, resource.ErrInvalidOption) {
					t.Fatal(e)
				}
				if status < 400 {
					var proof *resource.ResponseError
					if !errors.As(e, &proof) || proof.StatusCode != status || string(proof.Body) != body {
						t.Fatal(e, proof)
					}
				} else {
					actual := e
					if status == 401 {
						var auth *gophercloud.ErrUnableToReauthenticate
						if !errors.As(e, &auth) || auth.ErrOriginal == nil || auth.ErrReauth == nil {
							t.Fatal(e, auth)
						}
						actual = auth.ErrOriginal
					}
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(actual, &native) || native.Actual != status || string(native.Body) != body || native.Method != http.MethodPost {
						t.Fatal(e, native)
					}
				}
			})
		}
	}
}
func TestCloudRouterCreateCleanRetryTokenRenewalAndHeaderVersionCapture(t *testing.T) {
	for _, mode := range []string{"retry", "reauth", "route", "source199"} {
		t.Run(mode, func(t *testing.T) {
			attempts := 0
			tokens := []string{}
			posted := [][]byte{}
			service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Owned") != "before" || r.Header.Get("OpenStack-API-Version") != "network 2.1" {
					t.Error(r.Header)
				}
				if r.Method == http.MethodGet {
					testcloud.JSON(w, 200, `{"extensions":[{"alias":"router_availability_zone"}]}`)
					return
				}
				attempts++
				tokens = append(tokens, r.Header.Get("X-Auth-Token"))
				v := qosCloudCreateBody(t, r, "router")
				encoded, _ := json.Marshal(v)
				posted = append(posted, encoded)
				if attempts == 1 && mode != "source199" {
					status := 503
					if mode == "reauth" {
						status = 401
					}
					testcloud.JSON(w, status, `{"error":"original"}`)
					return
				}
				testcloud.JSON(w, 201, `{"router":{"id":"new"}}`)
			})
			service.RawClient().MoreHeaders = map[string]string{"X-Owned": "before"}
			service.RawClient().Microversion = "2.1"
			callbacks := 0
			if mode == "reauth" {
				service.RawClient().ReauthFunc = func(context.Context) error { callbacks++; service.RawClient().SetToken("renewed"); return nil }
			} else if mode != "source199" {
				service.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					callbacks++
					if mode == "route" {
						service.RawClient().ResourceBase += "changed/"
					}
					return nil
				}
			}
			if mode == "source199" {
				base := service.RawClient().HTTPClient.Transport
				if base == nil {
					base = http.DefaultTransport
				}
				extension := routerCloudObservedTransport{base: base, path: "/neutron/v2.0/extensions", status: 199}
				service.RawClient().HTTPClient.Transport = routerCloudObservedTransport{base: extension, path: "/neutron/v2.0/routers", status: 199}
			}
			option := network.RouterCreateOption(func(*network.RouterCreateOpts) error {
				service.RawClient().MoreHeaders["X-Owned"] = "after"
				return nil
			})
			result, e := service.CreateRouterRecord(context.Background(), network.WithRouterCreateAvailabilityZoneHints(), network.WithRouterCreateName("owned"), option)
			if mode == "route" {
				if result == nil || result.Router != nil || result.StatusCode != 503 || !errors.Is(e, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(e, 503) || count.Load() != 2 || callbacks != 1 {
					t.Fatal(result, e, count.Load(), callbacks)
				}
				return
			}
			want := int32(3)
			if mode == "source199" {
				want = 2
			}
			if e != nil || result == nil || result.Router == nil || count.Load() != want || len(result.Extensions) != 1 {
				t.Fatal(result, e, count.Load())
			}
			if mode == "source199" && (result.StatusCode != 199 || result.Extensions[0].StatusCode != 199 || callbacks != 0) || mode != "source199" && (callbacks != 1 || len(posted) != 2 || !bytes.Equal(posted[0], posted[1])) {
				t.Fatal(result, callbacks, posted)
			}
			if mode == "reauth" && !reflect.DeepEqual(tokens, []string{"test-token", "renewed"}) {
				t.Fatal(tokens)
			}
		})
	}
}

func TestCloudRouterCreateAZPhysicalFailureHasExtensionProofAndNoPostReceipt(t *testing.T) {
	for _, status := range []int{200, 401, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			body := `{"extensions":[{"alias":"router_availability_zone"}]}`
			service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/neutron/v2.0/extensions" {
					t.Error("POST after failed physical gate", r.Method, r.URL)
				}
				testcloud.JSON(w, status, body)
			})
			readErr, closeErr := errors.New("extension physical read"), errors.New("extension physical close")
			closed := &atomic.Int32{}
			base := service.RawClient().HTTPClient.Transport
			if base == nil {
				base = http.DefaultTransport
			}
			service.RawClient().HTTPClient.Transport = routerCloudObservedTransport{base: base, path: "/neutron/v2.0/extensions", readErr: readErr, closeErr: closeErr, closed: closed}
			retries, reauths := 0, 0
			service.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return errors.New("unexpected bounded gate retry")
			}
			service.RawClient().ReauthFunc = func(context.Context) error { reauths++; return errors.New("unexpected bounded gate reauth") }
			result, e := service.CreateRouterRecord(context.Background(), network.WithRouterCreateAvailabilityZoneHints())
			if result == nil || result.Router != nil || result.Wire != nil || result.StatusCode != 0 || len(result.Envelope) != 0 || e == nil || count.Load() != 1 || closed.Load() != 1 || retries != 0 || reauths != 0 {
				t.Fatal(result, e, count.Load(), closed.Load(), retries, reauths)
			}
			if status < 400 {
				var proof *resource.ResponseError
				if !errors.As(e, &proof) || proof.StatusCode != status || string(proof.Body) != body || !errors.Is(e, readErr) || !errors.Is(e, closeErr) {
					t.Fatal(e, proof)
				}
			} else {
				actual := e
				if status == 401 {
					var auth *gophercloud.ErrUnableToReauthenticate
					if !errors.As(e, &auth) || auth.ErrOriginal == nil || auth.ErrReauth == nil || !errors.Is(auth.ErrReauth, readErr) || !errors.Is(auth.ErrReauth, closeErr) {
						t.Fatal(e, auth)
					}
					actual = auth.ErrOriginal
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(actual, &native) || native.Actual != status || native.Method != http.MethodGet || string(native.Body) != body {
					t.Fatal(e, native)
				}
			}
		})
	}
}
