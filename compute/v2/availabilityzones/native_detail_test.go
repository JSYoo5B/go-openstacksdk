package availabilityzones_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/availabilityzones"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeZoneDetailTransport func(*http.Request) (*http.Response, error)

func (transport nativeZoneDetailTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeZoneDetailWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeZoneDetailCall struct{ method, path, query string }

func nativeZoneDetailAPI(t *testing.T, calls *[]nativeZoneDetailCall, reply func(*http.Request) *http.Response) *availabilityzones.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeZoneDetailTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, nativeZoneDetailCall{req.Method, req.URL.Path, req.URL.RawQuery})
		return reply(req), nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	return availabilityzones.New(client)
}

func TestNativeAvailabilityZoneListDetailObjectPage(t *testing.T) {
	ctx := context.Background()
	var calls []nativeZoneDetailCall
	// The object body would fail SinglePageBase's array-only IsEmpty, so the facade reads it as one native page.
	api := nativeZoneDetailAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeZoneDetailWire(200, `{"availabilityZoneInfo":[{"zoneName":"internal","zoneState":{"available":true},"hosts":{"ctl":{"nova-scheduler":{"active":true,"available":true,"updated_at":"2026-10-01T01:02:03.000000"}}}},{"zoneName":"nova","zoneState":{"available":false},"hosts":null}],"links":{"next":"http://ignored/"}}`)
	})
	var rows []*availabilityzones.AvailabilityZone
	for value, err := range api.ListDetail(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if len(rows) != 2 || rows[0].ZoneName != "internal" || !rows[0].ZoneState.Available || rows[1].ZoneState.Available || rows[1].Hosts != nil {
		t.Fatalf("%+v", rows)
	}
	state := rows[0].Hosts["ctl"]["nova-scheduler"]
	if !state.Active || !state.Available || !state.UpdatedAt.Equal(time.Date(2026, 10, 1, 1, 2, 3, 0, time.UTC)) {
		t.Fatalf("%+v", state)
	}
	if !reflect.DeepEqual(calls, []nativeZoneDetailCall{{http.MethodGet, "/nova/v2.1/os-availability-zone/detail", ""}}) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeAvailabilityZoneListDetailStatusesAndDecode(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		code int
		body string
		want string
	}{
		{404, `{}`, "status"},
		{201, `{}`, "status"},
		{200, `{}`, "empty"},
		{200, `{"availabilityZoneInfo":null}`, "empty"},
		{200, `{"availabilityZoneInfo":[]}`, "empty"},
		// A bad row fails the whole page before any row is published.
		{200, `{"availabilityZoneInfo":[{"zoneName":"ok"},{"zoneName":1}]}`, "decode"},
		{200, `{"availabilityZoneInfo":[{"hosts":{"h":{"s":{"updated_at":"2026-10-01T01:02:03Z"}}}}]}`, "decode"},
		{204, "", "eof"},
	} {
		var calls []nativeZoneDetailCall
		api := nativeZoneDetailAPI(t, &calls, func(*http.Request) *http.Response { return nativeZoneDetailWire(tc.code, tc.body) })
		var rows int
		var errs []error
		for value, err := range api.ListDetail(ctx) {
			if value != nil {
				rows++
			}
			if err != nil {
				errs = append(errs, err)
			}
		}
		var native gophercloud.ErrUnexpectedResponseCode
		var wrapped *resource.OperationError
		ok := rows == 0 && len(calls) == 1
		switch tc.want {
		case "status":
			ok = ok && len(errs) == 1 && errors.As(errs[0], &native) && native.Actual == tc.code && reflect.DeepEqual(native.Expected, []int{200, 204, 300}) && !errors.As(errs[0], &wrapped)
		case "empty":
			ok = ok && len(errs) == 0
		case "decode":
			ok = ok && len(errs) == 1 && !errors.As(errs[0], &wrapped)
		case "eof":
			ok = ok && len(errs) == 1 && errors.Is(errs[0], io.EOF)
		}
		if !ok {
			t.Fatal(tc.code, tc.body, rows, errs)
		}
	}
}
