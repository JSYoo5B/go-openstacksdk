package ports_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonSubnetPortsLiteralAndOwnedTargetsScanWithoutSubnetLookup(t *testing.T) {
	native := &subnets.Subnet{ID: "sub"}
	record := &subnets.SubnetRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"sub"`)}}}}
	for _, target := range []ports.SubnetPortTarget{ports.SubnetPortID("sub"), ports.SubnetPortNative(native), ports.SubnetPortRecord(record), ports.SubnetPortFields(map[string]json.RawMessage{"id": json.RawMessage(`"sub"`)}), ports.SubnetPortValue(json.RawMessage(`"sub"`))} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(r *http.Request) *http.Response {
			if r.URL.Path != "/neutron/v2.0/ports" || r.URL.RawQuery != "" {
				t.Fatal("lookup or filter", r.URL)
			}
			return nativePortWire(200, `{"ports":[{"id":"a","fixed_ips":[{"subnet_id":"sub"}]},{"id":"b","fixed_ips":[{"subnet_id":"other"}]},{"id":"c","fixed_ips":[]}]}`)
		})
		result, err := api.GetSubnetPorts(context.Background(), target)
		if err != nil || result == nil || len(result.Ports) != 1 || len(result.Inventory) != 3 || len(calls) != 1 || !reflect.DeepEqual(pythonPortListIDs(t, result.Ports), []string{"a"}) {
			t.Fatal(result, err, calls)
		}
	}
}

func TestPythonSubnetPortsDuplicateMatchesKeepIdenticalPointerAndPagination(t *testing.T) {
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.URL.Query().Get("marker") == "next" {
			return nativePortWire(200, `{"ports":[{"id":"b","fixed_ips":[{"subnet_id":"sub"}]}]}`)
		}
		return nativePortWire(200, `{"ports":[{"id":"a","fixed_ips":[{"subnet_id":"sub"},{"subnet_id":"sub"}]}],"ports_links":[{"rel":"next","href":"?marker=next"}]}`)
	})
	result, err := api.GetSubnetPorts(context.Background(), ports.SubnetPortID("sub"))
	if err != nil || result == nil || len(result.Ports) != 3 || result.Ports[0] != result.Ports[1] || len(result.Inventory) != 2 || len(calls) != 2 {
		t.Fatal(result, err, calls)
	}
	if result.Ports[0].Wire == nil || result.Ports[0].StatusCode != 200 {
		t.Fatal(result)
	}
}

func TestPythonSubnetPortsRawIdentifierUsesSourceLoadedEquality(t *testing.T) {
	for _, id := range []json.RawMessage{json.RawMessage(`true`), json.RawMessage(`9007199254740993.0`), json.RawMessage(`null`), json.RawMessage(`{"k":[1]}`)} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
			return nativePortWire(200, `{"ports":[{"id":"p","fixed_ips":[{"subnet_id":1},{"subnet_id":9007199254740992},{"subnet_id":null},{"subnet_id":{"k":[true]}}]}]}`)
		})
		result, err := api.GetSubnetPorts(context.Background(), ports.SubnetPortValue(id))
		if err != nil || result == nil || len(result.Ports) != 1 || len(calls) != 1 {
			t.Fatal(string(id), result, err, calls)
		}
	}
}

func TestPythonSubnetPortsTargetsAndOptionsAreOwnedAndGuarded(t *testing.T) {
	fields := map[string]json.RawMessage{"id": json.RawMessage(`"sub"`)}
	target := ports.SubnetPortFields(fields)
	fields["id"][2] = 'x'
	headers := map[string]string{"X-Trace": "owned"}
	option := ports.WithSubnetPortHeaders(headers)
	headers["X-Trace"] = "mutated"
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.Header.Get("X-Trace") != "owned" || r.Header.Get("X-Callback") != "yes" || r.Header.Get("OpenStack-API-Version") != "network 2.1" {
			t.Fatal(r.Header)
		}
		return nativePortWire(200, `{"ports":[{"id":"p","fixed_ips":[{"subnet_id":"sub"}]}]}`)
	})
	result, err := api.GetSubnetPorts(context.Background(), target, ports.WithSubnetPortHeaders(nil), option, ports.WithSubnetPortMicroversion("2.1"), func(c *ports.SubnetPortOpts) error { c.Headers["X-Callback"] = "yes"; return nil })
	if err != nil || result == nil || len(result.Ports) != 1 || len(calls) != 1 {
		t.Fatal(result, err, calls)
	}
	n := len(calls)
	for _, target := range []ports.SubnetPortTarget{{}, ports.SubnetPortRecord(nil), ports.SubnetPortNative(nil), ports.SubnetPortValue(json.RawMessage(`[`))} {
		result, err = api.GetSubnetPorts(context.Background(), target)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != n {
			t.Fatal(result, err, calls)
		}
	}
	result, err = api.GetSubnetPorts(context.Background(), ports.SubnetPortID("sub"), nil)
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != n {
		t.Fatal(result, err, calls)
	}
}

func TestPythonSubnetPortsMalformedFixedIPsRetainActualPageAndObservedRows(t *testing.T) {
	for _, fixed := range []string{"null", `[{}]`, `[null]`, `[4]`, `["sub"]`} {
		var calls []nativePortCall
		payload := `{"ports":[{"id":"ok","fixed_ips":[{"subnet_id":"sub"}]},{"id":"bad","fixed_ips":` + fixed + `}]}`
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
			r := nativePortWire(203, payload)
			r.Header.Set("X-Request-Id", "subnet-selection")
			return r
		})
		result, err := api.GetSubnetPorts(context.Background(), ports.SubnetPortID("sub"))
		var proof *resource.ResponseError
		if result == nil || len(result.Ports) != 1 || len(result.Inventory) != 2 || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != payload || proof.Header.Get("X-Request-Id") != "subnet-selection" || len(calls) != 1 {
			t.Fatal(fixed, result, err, proof, calls)
		}
		pythonPortListOperation(t, err, "GetSubnetPorts")
	}
}

func TestPythonSubnetPortsPhysical401NeverRetriesSelection(t *testing.T) {
	var calls []nativePortCall
	readCause, closeCause := errors.New("subnet read"), errors.New("subnet close")
	closed := 0
	body := &pythonFindFaultBody{data: `{"error":"actual"}`, read: readCause, close: closeCause, onClose: func() { closed++ }}
	api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { r := nativePortWire(401, body.data); r.Body = body; return r })
	reauth, retry := 0, 0
	api.RawClient().ReauthFunc = func(context.Context) error { reauth++; return nil }
	api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		retry++
		return nil
	}
	result, err := api.GetSubnetPorts(context.Background(), ports.SubnetPortID("sub"))
	var auth *gophercloud.ErrUnableToReauthenticate
	if result == nil || len(result.Inventory) != 0 || !errors.As(err, &auth) || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || closed != 1 || reauth != 0 || retry != 0 || len(calls) != 1 {
		t.Fatal(result, err, auth, closed, reauth, retry, calls)
	}
}

func TestPythonSubnetPortsCallbackCancellationAndSourceMutationPreserveCauses(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("unexpected HTTP"); return nil })
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		cause := errors.New("callback cancelled")
		optionCause := errors.New("option failed")
		result, err := api.GetSubnetPorts(ctx, ports.SubnetPortID("sub"), func(*ports.SubnetPortOpts) error {
			if cancelled {
				cancel(cause)
			} else {
				api.RawClient().ResourceBase += "changed/"
			}
			return optionCause
		})
		if result != nil || !errors.Is(err, optionCause) || len(calls) != 0 {
			t.Fatal(result, err, calls)
		}
		if cancelled && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) || !cancelled && !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
}
