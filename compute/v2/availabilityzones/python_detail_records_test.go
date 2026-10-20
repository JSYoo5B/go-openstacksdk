package availabilityzones_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/availabilityzones"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonZoneDetailCall struct{ method, path, query, body, nova, api, custom string }

const pythonZoneDetailPath = "/nova/v2.1/os-availability-zone/detail"

func pythonZoneDetailAPI(t *testing.T, microversion string, reply func(*http.Request) (int, string)) (*availabilityzones.API, *[]pythonZoneDetailCall, string) {
	t.Helper()
	calls := &[]pythonZoneDetailCall{}
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		*calls = append(*calls, pythonZoneDetailCall{r.Method, r.URL.Path, r.URL.RawQuery, string(data), r.Header.Get("X-OpenStack-Nova-API-Version"), r.Header.Get("OpenStack-API-Version"), r.Header.Get("X-Zone-Probe")})
		status, body := reply(r)
		w.Header().Set("X-Zone-Page", r.URL.RawQuery)
		testcloud.JSON(w, status, body)
	})
	client := cloud.Client("compute", "/nova/v2.1/")
	client.Microversion = microversion
	return availabilityzones.New(client), calls, cloud.Server.URL
}

const pythonZoneDetailRow = `{"zoneName":"internal","zoneState":{"available":true},"hosts":{"ctl":{"nova-scheduler":{"active":true,"available":true,"updated_at":"2026-10-01T01:02:03.000000"}}},"vendor":9007199254740993}`

func TestPythonZoneDetailRecordsUseDetailRouteAndKeepHosts(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name, client, nova, api string
		options                 []availabilityzones.AvailabilityZoneListOption
	}{
		{"availability_zones(details=True) with the client default", "2.55", "2.55", "compute 2.55", nil},
		{"availability_zones(details=True) with no default is versionless", "", "", "", nil},
		{"explicit empty microversion stays versionless", "2.55", "", "", []availabilityzones.AvailabilityZoneListOption{availabilityzones.WithAvailabilityZoneMicroversion("")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page := `{"availabilityZoneInfo":[` + pythonZoneDetailRow + `,{"zoneName":"nova","zoneState":{"available":false},"hosts":null}]}`
			api, calls, _ := pythonZoneDetailAPI(t, tc.client, func(*http.Request) (int, string) { return http.StatusOK, page })
			options := append(tc.options, availabilityzones.WithAvailabilityZoneHeader("X-Zone-Probe", "detail"))
			var rows []*availabilityzones.AvailabilityZoneRecord
			for record, err := range api.ListDetailRecords(ctx, options...) {
				if err != nil {
					t.Fatal(err)
				}
				rows = append(rows, record)
			}
			want := []pythonZoneDetailCall{{http.MethodGet, pythonZoneDetailPath, "", "", tc.nova, tc.api, "detail"}}
			if !reflect.DeepEqual(*calls, want) {
				t.Fatalf("%+v", *calls)
			}
			if len(rows) != 2 || string(rows[0].Envelope) != pythonZoneDetailRow || rows[0].StatusCode != 200 || rows[0].Header.Get("Content-Type") != "application/json" {
				t.Fatal(rows)
			}
			first, second := rows[0].Resource.Body, rows[1].Resource.Body
			if string(first["name"]) != `"internal"` || string(first["state"]) != `{"available":true}` || string(first["hosts"]) != `{"ctl":{"nova-scheduler":{"active":true,"available":true,"updated_at":"2026-10-01T01:02:03.000000"}}}` || string(first["id"]) != "null" || string(first["location"]) != "null" {
				t.Fatal(first)
			}
			if string(second["name"]) != `"nova"` || string(second["hosts"]) != "null" {
				t.Fatal(second)
			}
			if _, leaked := first["vendor"]; leaked || string(rows[0].Wire.Body["vendor"]) != "9007199254740993" || string(rows[0].Wire.Body["zoneName"]) != `"internal"` {
				t.Fatal(rows[0].Wire)
			}
		})
	}
}

func TestPythonZoneDetailRecordsFollowDetailLinks(t *testing.T) {
	ctx := context.Background()
	var next string
	api, calls, server := pythonZoneDetailAPI(t, "", func(r *http.Request) (int, string) {
		if r.URL.RawQuery == "" {
			return http.StatusOK, `{"availabilityZoneInfo":[{"zoneName":"a"}],"availabilityZoneInfo_links":[{"rel":"next","href":"` + next + `"}]}`
		}
		return http.StatusOK, `{"availabilityZoneInfo":[{"zoneName":"b"}]}`
	})
	next = server + pythonZoneDetailPath + "?marker=a"
	var names []string
	for record, err := range api.ListDetailRecords(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, string(record.Resource.Body["name"]))
	}
	want := []pythonZoneDetailCall{{http.MethodGet, pythonZoneDetailPath, "", "", "", "", ""}, {http.MethodGet, pythonZoneDetailPath, "marker=a", "", "", "", ""}}
	if !reflect.DeepEqual(names, []string{`"a"`, `"b"`}) || !reflect.DeepEqual(*calls, want) {
		t.Fatal(names, *calls)
	}
	// A continuation that leaves the detail route is rejected before HTTP.
	next = server + "/nova/v2.1/os-availability-zone?marker=a"
	*calls = nil
	rows, failed := 0, false
	for record, err := range api.ListDetailRecords(ctx) {
		if err != nil {
			failed = true
			continue
		}
		if record != nil {
			rows++
		}
	}
	if rows != 1 || !failed || len(*calls) != 1 {
		t.Fatal(rows, failed, *calls)
	}
}

func TestPythonZoneDetailRecordsStatuses(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		status int
		body   string
		rows   int
	}{
		{http.StatusOK, `{"availabilityZoneInfo":[]}`, 0},
		{http.StatusCreated, `{"availabilityZoneInfo":[{"zoneName":"z"}]}`, 1},
		{http.StatusMultipleChoices, `{"availabilityZoneInfo":{"zoneName":"z"}}`, 1},
		{http.StatusForbidden, `{"forbidden":{"message":"admin only"}}`, -1},
		{http.StatusNotFound, `{"itemNotFound":{"message":"gone"}}`, -1},
	} {
		api, calls, _ := pythonZoneDetailAPI(t, "2.55", func(*http.Request) (int, string) { return tc.status, tc.body })
		rows := 0
		var failure error
		for record, err := range api.ListDetailRecords(ctx) {
			if err != nil {
				failure = err
				continue
			}
			if record == nil {
				t.Fatal("nil record without error")
			}
			rows++
		}
		if len(*calls) != 1 || (*calls)[0].path != pythonZoneDetailPath {
			t.Fatal(*calls)
		}
		if tc.rows >= 0 {
			if failure != nil || rows != tc.rows {
				t.Fatal(tc.status, rows, failure)
			}
			continue
		}
		var operation *resource.OperationError
		var native gophercloud.ErrUnexpectedResponseCode
		if rows != 0 || !errors.As(failure, &operation) || operation.Operation != "ListDetailRecords" || operation.Resource != "availabilityzones" || !errors.As(failure, &native) || native.Actual != tc.status || string(native.Body) != tc.body {
			t.Fatal(tc.status, rows, failure)
		}
	}
}

func TestPythonZoneDetailRecordsLeaveOrdinaryRoute(t *testing.T) {
	api, calls, _ := pythonZoneDetailAPI(t, "", func(*http.Request) (int, string) {
		return http.StatusOK, `{"availabilityZoneInfo":[{"zoneName":"nova","zoneState":{"available":true},"hosts":null}]}`
	})
	for _, err := range api.ListRecords(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(*calls) != 1 || (*calls)[0].path != "/nova/v2.1/os-availability-zone" {
		t.Fatal(*calls)
	}
}
