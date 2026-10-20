package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const routerReadRows = `{"ports":[{"id":"gateway-first","device_owner":"network:router_gateway"},{"id":"internal","device_owner":"network:router_interface"},{"id":"ignored-ha","device_owner":"network:router_ha_interface"},{"id":"distributed","device_owner":"network:router_interface_distributed"},{"id":"replicated","device_owner":"network:ha_router_replicated_interface"},{"id":"unrelated","device_owner":"compute:nova"},{"id":"missing"}]}`

func routerReadIDs(t *testing.T, result *network.RouterInterfaceReadResult) []string {
	t.Helper()
	out := []string{}
	for _, row := range result.Ports {
		var id string
		if e := json.Unmarshal(row.Resource.Body["id"], &id); e != nil {
			t.Fatal(e)
		}
		out = append(out, id)
	}
	return out
}
func TestCloudRouterInterfacesCollectsEveryPageBeforeStableOwnerBuckets(t *testing.T) {
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/ports": {{203, routerReadRows[:len(routerReadRows)-1] + `,"next":"?marker=tail"}`}, {200, `{"ports":[{"id":"gateway-last","device_owner":"network:router_gateway"}]}`}}})
	result, err := network.New(client).ListRouterInterfaces(context.Background(), network.RouterInterfaceRouterID("literal/router"))
	want := []string{"internal", "distributed", "replicated", "gateway-first", "gateway-last"}
	if err != nil || result == nil || !reflect.DeepEqual(routerReadIDs(t, result), want) || len(result.Inventory) != 8 || len(calls()) != 2 || calls()[0].query.Get("device_id") != "literal/router" || calls()[1].query.Get("device_id") != "literal/router" || calls()[1].query.Get("marker") != "tail" {
		t.Fatal(result, err, calls())
	}
	if result.Ports[0].StatusCode != 203 || result.Ports[4].StatusCode != 200 || string(result.Inventory[0].Resource.Body["id"]) != `"gateway-first"` {
		t.Fatal(result)
	}
}
func TestCloudRouterInterfacesFalseyExactUnknownAndNonstringTypes(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		count int
	}{
		{"null", 4}, {"false", 4}, {"0", 4}, {"1e-400", 4}, {`""`, 4}, {"[]", 4}, {"{}", 4},
		{`"internal"`, 3}, {`"external"`, 1}, {`"Internal"`, 0}, {`"other"`, 0}, {"true", 0}, {"[1]", 0}, {`{"x":1}`, 0}, {`"\u0069nternal"`, 3},
	} {
		client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/ports": {{200, routerReadRows}}})
		result, err := network.New(client).ListRouterInterfaces(context.Background(), json.RawMessage(`{"id":null}`), network.WithRouterInterfaceTypeJSON(json.RawMessage(tc.raw)))
		if err != nil || len(result.Ports) != tc.count || len(result.Inventory) != 7 || len(calls()) != 1 || calls()[0].query.Has("device_id") {
			t.Fatal(tc, result, err, calls())
		}
	}
}
func TestCloudRouterInterfacesRouterDictionaryReadDoesNotResolveOrSeed(t *testing.T) {
	for _, tc := range []struct {
		router string
		query  []string
	}{
		{`{"id":true}`, []string{"True"}}, {`{"id":42}`, []string{"42"}}, {`{"id":["a","b"]}`, []string{"a", "b"}}, {`{"id":{"key":1}}`, []string{"key"}}, {`{"id":""}`, []string{""}},
	} {
		client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/ports": {{200, `{"ports":[{"id":"wire","device_owner":true}]}`}}})
		result, err := network.New(client).ListRouterInterfaces(context.Background(), json.RawMessage(tc.router))
		if err != nil || len(result.Ports) != 0 || len(result.Inventory) != 1 || len(calls()) != 1 || !reflect.DeepEqual(calls()[0].query["device_id"], tc.query) || string(result.Inventory[0].Resource.Body["device_id"]) != "null" {
			t.Fatal(tc, result, err, calls())
		}
	}
	for _, raw := range []string{`{}`, `null`, `"router"`, `[]`, `broken`} {
		client, calls := pythonCloudBClient(t, nil)
		result, err := network.New(client).ListRouterInterfaces(context.Background(), json.RawMessage(raw))
		if result != nil || err == nil || len(calls()) != 0 {
			t.Fatal(raw, result, err, calls())
		}
	}
}
func TestCloudRouterInterfacesOwnedInputsLocationHeadersAndRawReceipt(t *testing.T) {
	cloud := testcloud.New(t)
	var header http.Header
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		header = r.Header.Clone()
		w.Header().Set("X-Request-Id", "router-actual")
		testcloud.JSON(w, 203, `{"ports":[{"id":"one","device_owner":"network:router_interface","tenant_id":"foreign","availability_zone":"wire-only","location":{"fake":true},"shared":1e-400}]}`)
	})
	client := cloud.Client("network", "/")
	client.ResourceBase = cloud.Server.URL + "/v2.0/"
	name, project, zone := "cloud", json.RawMessage(`"current"`), json.RawMessage(`"configured-zone"`)
	location := resource.CloudLocation{Cloud: &name, Zone: zone, Project: resource.CloudProject{ID: project, Name: &name}}
	loc := network.WithRouterInterfaceReadCurrentLocation(location)
	name = "mutated"
	project[1] = 'X'
	zone[1] = 'X'
	raw := json.RawMessage(`"internal"`)
	filter := network.WithRouterInterfaceTypeJSON(raw)
	raw[1] = 'X'
	headers := map[string]string{"X-Trace": "owned"}
	h := network.WithRouterInterfaceReadHeaders(headers)
	headers["X-Trace"] = "mutated"
	result, err := network.New(client).ListRouterInterfaces(context.Background(), network.RouterInterfaceRouterID("router"), loc, filter, h, network.WithRouterInterfaceReadMicroversion("2.77"))
	if err != nil || len(result.Ports) != 1 || header.Get("X-Trace") != "owned" || header.Get("OpenStack-API-Version") != "network 2.77" {
		t.Fatal(result, err, header)
	}
	row := result.Ports[0]
	var current resource.CloudLocation
	if e := json.Unmarshal(row.Resource.Body["location"], &current); e != nil || current.Cloud == nil || *current.Cloud != "cloud" || string(current.Project.ID) != `"foreign"` || current.Project.Name != nil || string(current.Zone) != "null" {
		t.Fatal(current, e)
	}
	if row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "router-actual" || string(row.Wire.Body["location"]) != `{"fake":true}` {
		t.Fatal(row)
	}
	value := string(result.Value)
	row.Resource.Body["id"][1] = 'X'
	row.Header.Set("X-Request-Id", "consumer")
	row.Envelope[0] = 'X'
	row.Wire.Body["id"][1] = 'Y'
	if string(result.Value) != value || string(result.Inventory[0].Resource.Body["id"]) != `"one"` || result.LastPort.Header.Get("X-Request-Id") != "router-actual" || string(result.Inventory[0].Wire.Body["id"]) != `"one"` {
		t.Fatal(result)
	}
}
func TestCloudRouterInterfacesSourceGuardCancellationAndPreflight(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		client, calls := pythonCloudBClient(t, nil)
		service := network.New(client)
		ctx, cancel := context.WithCancelCause(context.Background())
		cause, optionErr := errors.New("router cancelled"), errors.New("option failure")
		result, err := service.ListRouterInterfaces(ctx, network.RouterInterfaceRouterID("router"), func(*network.RouterInterfaceReadOpts) error {
			if cancelled {
				cancel(cause)
			} else {
				client.ResourceBase += "changed/"
			}
			return optionErr
		})
		cancel(nil)
		if result != nil || !errors.Is(err, optionErr) || cancelled && !errors.Is(err, cause) || !cancelled && !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 0 {
			t.Fatal(result, err, calls())
		}
	}
	var service *network.Service
	result, err := service.ListRouterInterfaces(context.Background(), network.RouterInterfaceRouterID("router"))
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(result, err)
	}
	client, calls := pythonCloudBClient(t, nil)
	result, err = network.New(client).ListRouterInterfaces(nil, network.RouterInterfaceRouterID("router"))
	if result != nil || err == nil || len(calls()) != 0 {
		t.Fatal(result, err, calls())
	}
}
func TestCloudRouterInterfacesLatePagesPreventPartialSuccessAndKeepPhysicalFaults(t *testing.T) {
	for _, status := range []int{200, 203, 401, 404, 429, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			n := 0
			faultBody := `{"actual":"router page"}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n++
				if r.URL.Query().Get("marker") == "late" {
					w.Header().Set("X-Request-Id", "router-final")
					w.Header().Set("Content-Length", "1000")
					w.WriteHeader(status)
					_, _ = w.Write([]byte(faultBody))
					return
				}
				testcloud.JSON(w, 200, `{"ports":[{"id":"observed","device_owner":"network:router_interface"}],"next":"?marker=late"}`)
			})
			client := cloud.Client("network", "/")
			client.ResourceBase = cloud.Server.URL + "/v2.0/"
			retry, reauth, backoff := 0, 0, 0
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retry++
				return nil
			}
			client.ReauthFunc = func(context.Context) error { reauth++; return nil }
			client.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				backoff++
				return nil
			}
			result, err := network.New(client).ListRouterInterfaces(context.Background(), network.RouterInterfaceRouterID("router"), network.WithRouterInterfaceType("unknown"))
			if result == nil || len(result.Inventory) != 1 || result.Ports != nil || result.Value != nil || n != 2 || retry != 0 || reauth != 0 || backoff != 0 || !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatal(result, err, n, retry, reauth, backoff)
			}
			if status < 400 {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != faultBody || proof.Header.Get("X-Request-Id") != "router-final" {
					t.Fatal(proof, err)
				}
			} else {
				original := err
				if status == 401 {
					var unable *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &unable) || !errors.Is(unable.ErrReauth, io.ErrUnexpectedEOF) {
						t.Fatal(unable, err)
					}
					original = unable.ErrOriginal
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(original, &native) || native.Actual != status || string(native.Body) != faultBody || native.ResponseHeader.Get("X-Request-Id") != "router-final" {
					t.Fatal(native, err)
				}
			}
		})
	}
}
