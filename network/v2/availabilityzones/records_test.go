package availabilityzones_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/availabilityzones"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func zoneWire(code int, raw string) *http.Response {
	return testcloud.Response(code, raw, http.Header{"X-Request-Id": {"zone-actual"}})
}
func zoneAPI(t *testing.T, reply testcloud.RoundTripperFunc) (*availabilityzones.API, *gophercloud.ServiceClient) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(reply)
	client := cloud.Client("network", "/neutron/v2.0/")
	return availabilityzones.New(client), client
}

type unmarshalIgnored struct{ calls *int }

func (v unmarshalIgnored) MarshalJSON() ([]byte, error) {
	*v.calls++
	return nil, errors.New("ignored proxy value marshaled")
}
func TestAvailabilityZonesProxyDiscardsAllQueryKwargsWithoutMarshalOrFiltering(t *testing.T) {
	calls, marshaled := 0, 0
	api, _ := zoneAPI(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != "GET" || req.URL.Path != "/neutron/v2.0/availability_zones" || req.URL.RawQuery != "" || req.Body != nil {
			t.Fatal(req.Method, req.URL, req.Body)
		}
		return zoneWire(200, `{"availability_zones":[{"name":"az-a","resource":"network","state":"available"},{"name":"az-b","resource":"router","state":"unavailable"}]}`), nil
	})
	opts := []availabilityzones.ListOption{}
	for _, key := range []string{"name", "resource", "state", "limit", "marker", "fields", "jmespath_filters", "microversion", "headers", "cls", "session"} {
		opts = append(opts, availabilityzones.WithProxyQuery(key, unmarshalIgnored{&marshaled}))
	}
	rows, err := api.AllRecords(context.Background(), opts...)
	if err != nil || len(rows) != 2 || calls != 1 || marshaled != 0 || string(rows[1].Resource.Body["name"]) != `"az-b"` {
		t.Fatal(rows, err, calls, marshaled)
	}
}
func TestAvailabilityZonesDeclaredDefaultsUnknownFieldsAndActualReceiptOwnBytes(t *testing.T) {
	raw := `{"availability_zones":[{"name":"az","x":{"large":9007199254740993},"resource":null,"state":false,"location":{"spoof":true}}]}`
	api, _ := zoneAPI(t, func(*http.Request) (*http.Response, error) { return zoneWire(203, raw), nil })
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 1 || rows[0].StatusCode != 203 || string(rows[0].Envelope) != raw || string(rows[0].Resource.Body["id"]) != "null" || string(rows[0].Resource.Body["state"]) != "false" || string(rows[0].Resource.Body["location"]) != "null" {
		t.Fatal(rows, err)
	}
	row := rows[0]
	row.Resource.Body["name"][1] = 'X'
	row.Header.Set("X-Request-Id", "late")
	if string(row.Wire.Body["name"]) != `"az"` || row.Wire.Header.Get("X-Request-Id") != "zone-actual" || row.Resource.Header.Get("X-Request-Id") != "zone-actual" {
		t.Fatal(row)
	}
}
func TestAvailabilityZonesPagingAndAllRecordsDiscardPartialErrors(t *testing.T) {
	calls := 0
	api, client := zoneAPI(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return zoneWire(200, `{"availability_zones":[{"id":"one","name":"first"}],"availability_zones_links":[{"rel":"next","href":"?marker=one"}]}`), nil
		}
		if req.URL.Query().Get("marker") != "one" {
			t.Fatal(req.URL)
		}
		return zoneWire(200, `{"availability_zones":[false]}`), nil
	})
	rows, err := api.AllRecords(context.Background())
	var proof *resource.ResponseError
	if rows != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || calls != 2 {
		t.Fatal(rows, err, calls, client)
	}
}
func TestAvailabilityZonesLazyStopDoesNotFollowNextLink(t *testing.T) {
	calls := 0
	api, _ := zoneAPI(t, func(*http.Request) (*http.Response, error) {
		calls++
		return zoneWire(200, `{"availability_zones":[{"name":"az"}],"availability_zones_links":[{"rel":"next","href":"?marker=next"}]}`), nil
	})
	for row, err := range api.ListRecords(context.Background()) {
		if err != nil || row == nil {
			t.Fatal(row, err)
		}
		break
	}
	if calls != 1 {
		t.Fatal(calls)
	}
}
func TestAvailabilityZonesCallbacksCannotHideSourceChangeOrCancellation(t *testing.T) {
	calls, next := 0, 0
	api, client := zoneAPI(t, func(*http.Request) (*http.Response, error) {
		calls++
		return zoneWire(200, `{"availability_zones":[]}`), nil
	})
	initial := client.Endpoint
	change := func(*availabilityzones.ListOpts) error { client.Endpoint = "https://foreign.invalid/v2/"; return nil }
	restore := func(*availabilityzones.ListOpts) error { next++; client.Endpoint = initial; return nil }
	_, err := api.AllRecords(context.Background(), change, restore)
	if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || next != 0 {
		t.Fatal(err, calls, next)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("zone cancel")
	cancel(cause)
	_, err = api.AllRecords(ctx)
	if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestAvailabilityZonesEmptyMalformedAndBoundSelfPolicy(t *testing.T) {
	for _, raw := range []string{`{"availability_zones":null}`, `{"availability_zones":[{"cls":true}]}`, `{"availability_zones":[null]}`} {
		api, _ := zoneAPI(t, func(*http.Request) (*http.Response, error) { return zoneWire(200, raw), nil })
		rows, err := api.AllRecords(context.Background())
		if rows != nil || err == nil {
			t.Fatal(raw, rows, err)
		}
	}
	calls := 0
	api, _ := zoneAPI(t, func(*http.Request) (*http.Response, error) {
		calls++
		return zoneWire(200, `{"availability_zones":[]}`), nil
	})
	rows, err := api.AllRecords(context.Background())
	if err != nil || rows == nil || len(rows) != 0 {
		t.Fatal(rows, err)
	}
	_, err = api.AllRecords(context.Background(), availabilityzones.WithProxyQuery("self", json.RawMessage(`null`)))
	if !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
		t.Fatal(err, calls)
	}
	_, err = api.AllRecords(context.Background(), nil)
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
func TestAvailabilityZonesNative401KeepsActualOriginalErrorAndCleanReauth(t *testing.T) {
	calls, reauth := 0, 0
	api, client := zoneAPI(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return zoneWire(401, `{"error":"renew"}`), nil
		}
		if req.Header.Get("X-Auth-Token") != "renewed" {
			t.Fatal(req.Header)
		}
		return zoneWire(200, `{"availability_zones":[]}`), nil
	})
	client.ReauthFunc = func(context.Context) error { reauth++; client.SetToken("renewed"); return nil }
	rows, err := api.AllRecords(context.Background())
	if err != nil || rows == nil || calls != 2 || reauth != 1 {
		t.Fatal(rows, err, calls, reauth)
	}
}
