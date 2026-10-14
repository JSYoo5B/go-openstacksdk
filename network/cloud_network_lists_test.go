package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestCloudNetworkListAbsentServicePrecedesFiltersAndSearchHasNoAbsenceGate(t *testing.T) {
	for _, raw := range []string{`{bad`, `[1]`, `{"jmespath_filters":"length(@)"}`, `{"headers":"bad"}`} {
		var service *network.Service
		result, err := service.ListNetworkRecords(context.Background(), network.WithNetworkCloudServiceAvailable(false), network.WithNetworkCloudFilters(json.RawMessage(raw)), network.WithNetworkCloudCurrentLocation(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`bad`)}}))
		if err != nil || result == nil || result.Networks == nil || result.Values == nil || string(result.JSON) != "[]" {
			t.Fatal(raw, result, err)
		}
		result, err = service.SearchNetworkRecords(context.Background(), "literal", network.WithNetworkCloudServiceAvailable(false))
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(result, err)
		}
	}
	service, count := qosCloudMutationFixture(t, func(http.ResponseWriter, *http.Request) { t.Fatal("absent service HTTP") })
	service.RawClient().Type = "compute"
	result, err := service.ListNetworkRecords(context.Background(), network.WithNetworkCloudServiceAvailable(false), network.WithNetworkCloudFilters(json.RawMessage(`bad`)))
	if err != nil || string(result.JSON) != "[]" || count.Load() != 0 {
		t.Fatal(result, err, count.Load())
	}
}

func TestCloudNetworkListFalseyFiltersAndSearchOrderedNameUpdate(t *testing.T) {
	for _, raw := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(`false`), json.RawMessage(`0`), json.RawMessage(`1e-4000`), json.RawMessage(`""`), json.RawMessage(`[]`), json.RawMessage(`{}`)} {
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/neutron/v2.0/networks" || r.URL.RawQuery != "" {
				t.Error(r.Method, r.URL)
			}
			testcloud.JSON(w, 200, `{"networks":[]}`)
		})
		result, err := service.ListNetworkRecords(context.Background(), network.WithNetworkCloudFilters(raw))
		if err != nil || result == nil || result.Networks == nil || string(result.JSON) != "[]" || count.Load() != 1 {
			t.Fatal(string(raw), result, err, count.Load())
		}
	}
	for _, tc := range []struct {
		raw   string
		query url.Values
	}{
		{`{"name":"override","is_shared":false,"fields":["id","x_new"],"tenant_id":"ignored"}`, url.Values{"name": {"override"}, "shared": {"False"}, "fields": {"id", "x_new"}}},
		{`{"name":null}`, url.Values{}},
		{`[["name","old"],["name","last"],["is_admin_state_up",false]]`, url.Values{"name": {"last"}, "admin_state_up": {"False"}}},
		{`["xy"]`, url.Values{"name": {"literal/id*"}}},
	} {
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if !reflect.DeepEqual(r.URL.Query(), tc.query) {
				t.Error(r.URL, tc.query)
			}
			testcloud.JSON(w, 200, `{"networks":[{"id":"unrelated","name":"unrelated"}]}`)
		})
		result, err := service.SearchNetworkRecords(context.Background(), "literal/id*", network.WithNetworkCloudFilters(json.RawMessage(tc.raw)), network.WithNetworkCloudServiceAvailable(false))
		if err != nil || result == nil || len(result.Networks) != 1 || count.Load() != 1 {
			t.Fatal(tc.raw, result, err, count.Load())
		}
	}
	for _, raw := range []string{`"bad"`, `true`, `1`, `[1]`, `[["name","value"]]`, `{bad`} {
		service, count := qosCloudMutationFixture(t, func(http.ResponseWriter, *http.Request) { t.Fatal("invalid mapping HTTP") })
		result, err := service.ListNetworkRecords(context.Background(), network.WithNetworkCloudFilters(json.RawMessage(raw)))
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != 0 {
			t.Fatal(raw, result, err, count.Load())
		}
	}
}

func TestCloudNetworkListRawControlsLocalLoadedEqualityPagingAndReceipts(t *testing.T) {
	var paths []string
	body := `{"networks":[{"id":"skip","provider:segmentation_id":2,"segments":[{"x":2}],"mtu":1500},{"id":"keep","provider:segmentation_id":1,"segments":[{"x":true}],"mtu":1500.9,"router:external":1,"x_new":9007199254740993}],"next":"?marker=later"}`
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path+"?"+r.URL.RawQuery)
		if r.URL.Path != "/neutron/v2.0/alternate" || r.Header.Get("X-Trace") != "owned" || r.Header.Get("OpenStack-API-Version") != "network 2.1" {
			t.Error(r.URL, r.Header)
		}
		if r.URL.Query().Get("marker") == "later" {
			testcloud.JSON(w, 203, `{"networks":[{"id":"last","provider:segmentation_id":true,"segments":[{"x":1}],"mtu":1500}]}`)
			return
		}
		if r.URL.Query().Get("limit") != "3" || r.URL.Query().Get("shared") != "False" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 203, body)
	})
	raw := json.RawMessage(`{"base_path":"/alternate","headers":{"X-Trace":"owned"},"microversion":"2.1","is_shared":false,"segments":[{"x":1}],"mtu":1500,"max_items":3,"__conflicting_attrs":{"jmespath_filters":"[?"}}`)
	option := network.WithNetworkCloudFilters(raw)
	raw[2] = 'X'
	result, err := service.ListNetworkRecords(context.Background(), option)
	if err != nil || result == nil || len(result.Networks) != 2 || count.Load() != 2 || len(paths) != 2 {
		t.Fatal(result, err, count.Load(), paths)
	}
	row := result.Networks[0]
	if string(row.Resource.Body["id"]) != `"keep"` || string(row.Resource.Body["is_router_external"]) != "true" || string(row.Resource.Body["mtu"]) != "1500" || string(row.Resource.Body["x_new"]) != "9007199254740993" || string(row.Envelope) != body || row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "qos-mutation-actual" || row.Wire.Body["mtu"] == nil || string(row.Resource.Body["location"]) != "null" {
		t.Fatal(row)
	}
	if string(result.Networks[1].Resource.Body["id"]) != `"last"` {
		t.Fatal(result)
	}
	for _, raw := range []string{`{"cls":null}`, `{"__conflicting_attrs":{"base_path":"x"}}`, `{"base_path":"https://foreign.invalid"}`} {
		before := count.Load()
		value, cause := service.ListNetworkRecords(context.Background(), network.WithNetworkCloudFilters(json.RawMessage(raw)))
		if value != nil || !errors.Is(cause, resource.ErrInvalidOption) || count.Load() != before {
			t.Fatal(raw, value, cause, count.Load())
		}
	}
}

func TestCloudNetworkListOpaqueJMESListConversionAndDeferredFormals(t *testing.T) {
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"networks":[{"id":"one","availability_zone":false}]}`)
	})
	for _, tc := range []struct{ expr, want string }{
		{"`[]`", "[]"}, {"`[1,null,true]`", "[1,null,true]"}, {"`\"ab\"`", `["a","b"]`}, {"`{\"z\":0,\"a\":1}`", `["a","z"]`},
	} {
		raw, _ := json.Marshal(map[string]any{"jmespath_filters": tc.expr, "headers": "bad", "base_path": false, "microversion": map[string]int{"x": 1}})
		value, err := service.ListNetworkRecords(context.Background(), network.WithNetworkCloudFilters(raw))
		if err != nil || value == nil || string(value.JSON) != tc.want || count.Load() != 0 {
			t.Fatal(tc, value, err, count.Load())
		}
	}
	for _, expr := range []string{"[0]", "[*]", "type(@)", "length(@)", "sort_by(@,&id)"} {
		raw, _ := json.Marshal(map[string]any{"jmespath_filters": expr})
		value, err := service.ListNetworkRecords(context.Background(), network.WithNetworkCloudFilters(raw))
		if value != nil || err == nil || count.Load() != 0 {
			t.Fatal(expr, value, err, count.Load())
		}
	}
	raw := json.RawMessage(`{"headers":"bad","jmespath_filters":"to_array(@)"}`)
	nested, err := service.ListNetworkRecords(context.Background(), network.WithNetworkCloudFilters(raw))
	if err != nil || nested == nil || nested.JSON != nil || len(nested.Values) != 1 || count.Load() != 0 {
		t.Fatal(nested, err, count.Load())
	}
	for row, cause := range nested.Values[0].(*networks.RecordListGenerator).Records {
		if row != nil || !errors.Is(cause, resource.ErrInvalidOption) {
			t.Fatal(row, cause)
		}
	}
	if count.Load() != 0 {
		t.Fatal(count.Load())
	}
	value, err := service.ListNetworkRecords(context.Background(), network.WithNetworkCloudFilters(json.RawMessage(`{"base_path":""}`)))
	if err != nil || string(value.JSON) != "[]" || count.Load() != 0 {
		t.Fatal(value, err, count.Load())
	}
	for _, raw := range []string{`{"headers":"bad"}`, `{"base_path":false}`, `{"microversion":{"x":1}}`} {
		value, err := service.ListNetworkRecords(context.Background(), network.WithNetworkCloudFilters(json.RawMessage(raw)))
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != 0 {
			t.Fatal(raw, value, err, count.Load())
		}
	}
	value, err = service.ListNetworkRecords(context.Background(), network.WithNetworkCloudFilters(json.RawMessage(`{"jmespath_filters":"@"}`)))
	if err != nil || len(value.Networks) != 1 || count.Load() != 1 {
		t.Fatal(value, err, count.Load())
	}
}

func TestCloudNetworkListRawMicroversionTransportAndUnknownLocationZone(t *testing.T) {
	for _, tc := range []struct{ raw, header string }{
		{`false`, ""}, {`0`, ""}, {`""`, ""}, {`[]`, ""}, {`{}`, ""}, {`true`, "network 1.0"}, {`1`, "network 1.0"}, {`[2,4]`, "network 2.4"}, {`{"2":null,"4":false}`, "network 2.4"}, {`"v2.3"`, "network 2.3"}, {`null`, "network 2.7"},
	} {
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("OpenStack-API-Version") != tc.header {
				t.Error(tc.raw, r.Header)
			}
			testcloud.JSON(w, 200, `{"networks":[{"id":"one","project_id":1,"tenant_id":"ignored","availability_zone":{"raw":false},"location":{"forged":true}}]}`)
		})
		service.RawClient().Microversion = "2.7"
		n := "configured"
		loc := resource.CloudLocation{Cloud: &n, Project: resource.CloudProject{ID: json.RawMessage(`true`), Name: &n}}
		result, err := service.ListNetworkRecords(context.Background(), network.WithNetworkCloudFilters(json.RawMessage(`{"microversion":`+tc.raw+`}`)), network.WithNetworkCloudCurrentLocation(loc))
		if err != nil || result == nil || len(result.Networks) != 1 || count.Load() != 1 {
			t.Fatal(tc, result, err, count.Load())
		}
		var actual resource.CloudLocation
		json.Unmarshal(result.Networks[0].Resource.Body["location"], &actual)
		if string(actual.Project.ID) != "true" || actual.Project.Name == nil || *actual.Project.Name != "configured" || string(actual.Zone) != `{"raw":false}` || string(result.Networks[0].Wire.Body["location"]) != `{"forged":true}` {
			t.Fatal(actual, result.Networks[0])
		}
	}
}

func TestCloudNetworkListLateFailureIsAtomicAndOptionsOwnSource(t *testing.T) {
	service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Owned") != "before" {
			t.Error(r.Header)
		}
		if r.URL.Query().Get("marker") == "later" {
			testcloud.JSON(w, 503, `{"error":"late actual"}`)
			return
		}
		testcloud.JSON(w, 200, `{"networks":[{"id":"first"}],"next":"?marker=later"}`)
	})
	headers := map[string]string{"X-Owned": "before"}
	option := network.WithNetworkCloudHeaders(headers)
	headers["X-Owned"] = "after"
	result, err := service.ListNetworkRecords(context.Background(), option)
	var native gophercloud.ErrUnexpectedResponseCode
	if result != nil || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"error":"late actual"}` || count.Load() != 2 {
		t.Fatal(result, err, count.Load())
	}
	before := count.Load()
	result, err = service.ListNetworkRecords(context.Background(), func(*network.NetworkCloudListOpts) error { service.RawClient().ResourceBase += "changed/"; return nil })
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != before {
		t.Fatal(result, err, count.Load())
	}
}

type networkListObservedTransport struct {
	base              http.RoundTripper
	readErr, closeErr error
	closed            *atomic.Int32
	onClose           func()
}

func (t networkListObservedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	v, err := t.base.RoundTrip(r)
	if v != nil && r.URL.Path == "/neutron/v2.0/networks" {
		v.Body = &qosPolicyActionObservedBody{ReadCloser: v.Body, readErr: t.readErr, closeErr: t.closeErr, closed: t.closed, onClose: t.onClose}
	}
	return v, err
}
func TestCloudNetworkListPhysicalFaultsKeepActualReceiptAndNoNativeHooks(t *testing.T) {
	for _, status := range []int{200, 401, 404, 503} {
		for _, lane := range []string{"read-close", "cancel", "source"} {
			t.Run(fmt.Sprintf("%d/%s", status, lane), func(t *testing.T) {
				body := `{"networks":[{"id":"actual"}]}`
				service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, status, body) })
				readErr, closeErr, cancelCause := errors.New("read actual"), errors.New("close actual"), errors.New("caller cause")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				closed := &atomic.Int32{}
				base := service.RawClient().HTTPClient.Transport
				if base == nil {
					base = http.DefaultTransport
				}
				observer := networkListObservedTransport{base: base, closed: closed}
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
				result, err := service.ListNetworkRecords(ctx)
				if result != nil || err == nil || count.Load() != 1 || closed.Load() != 1 || retries != 0 || reauths != 0 || errors.Is(err, unexpected) {
					t.Fatal(result, err, count.Load(), closed.Load(), retries, reauths)
				}
				switch lane {
				case "read-close":
					if !errors.Is(err, readErr) || !errors.Is(err, closeErr) {
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
				if status == 200 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "qos-mutation-actual" {
						t.Fatal(err, proof)
					}
				} else {
					receipt := err
					if status == 401 {
						var auth *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &auth) || auth.ErrOriginal == nil || auth.ErrReauth == nil {
							t.Fatal(err, auth)
						}
						receipt = auth.ErrOriginal
						if lane == "read-close" && (!errors.Is(auth.ErrReauth, readErr) || !errors.Is(auth.ErrReauth, closeErr)) {
							t.Fatal(err, auth)
						}
					}
					var proof gophercloud.ErrUnexpectedResponseCode
					if !errors.As(receipt, &proof) || proof.Actual != status || proof.Method != http.MethodGet || !strings.HasSuffix(proof.URL, "/neutron/v2.0/networks") || string(proof.Body) != body || proof.ResponseHeader.Get("X-Request-Id") != "qos-mutation-actual" {
						t.Fatal(err, proof)
					}
				}
			})
		}
	}
}

func TestCloudNetworkListCleanRetryTokenRenewalAndGuardedRetry(t *testing.T) {
	for _, lane := range []string{"retry", "reauth", "source"} {
		attempts := 0
		var tokens []string
		service, count := qosCloudMutationFixture(t, func(w http.ResponseWriter, r *http.Request) {
			attempts++
			tokens = append(tokens, r.Header.Get("X-Auth-Token"))
			if attempts == 1 {
				status := 503
				if lane == "reauth" {
					status = 401
				}
				testcloud.JSON(w, status, `{"error":"original"}`)
				return
			}
			testcloud.JSON(w, 200, `{"networks":[]}`)
		})
		callbacks := 0
		unexpected := errors.New("unexpected repeat callback")
		service.RawClient().ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			callbacks++
			if callbacks > 1 {
				return unexpected
			}
			if lane == "source" {
				service.RawClient().ResourceBase += "changed/"
			}
			return nil
		}
		service.RawClient().ProviderClient.ReauthFunc = func(context.Context) error {
			callbacks++
			if callbacks > 1 {
				return unexpected
			}
			service.RawClient().ProviderClient.SetToken("renewed")
			return nil
		}
		result, err := service.ListNetworkRecords(context.Background())
		if lane == "source" {
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || count.Load() != 1 || callbacks != 1 {
				t.Fatal(result, err, count.Load(), callbacks)
			}
		} else if err != nil || result == nil || count.Load() != 2 || callbacks != 1 {
			t.Fatal(result, err, count.Load(), callbacks)
		}
		if lane == "reauth" && (len(tokens) != 2 || tokens[1] != "renewed") {
			t.Fatal(tokens)
		}
	}
}
