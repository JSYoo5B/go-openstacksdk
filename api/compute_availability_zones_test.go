package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/compute"
	"github.com/JSYoo5B/gophercloudsdk/compute/v2/availabilityzones"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

const computeAvailabilityZonesPath = "/reverse/nova/v2.1/project/os-availability-zone"

// Ordinary Proxy availability_zones and Cloud names use the existing owned
// reader and public HTTP fixture. Detailed admin inventory is a separate unit.
func TestComputeAvailabilityZoneRecordsOrdinaryViewsLazyPagingAndOwnership(t *testing.T) {
	for _, tc := range []struct {
		name, row, id, zone, state, hosts string
		status                            int
	}{
		{"raw descriptors and unknown precision", `{"id":"wire-id","zoneName":"zone-a","zoneState":{"available":null,"vendor":9007199254740993},"hosts":{"node":{"vendor":[null,false,9007199254740993]}},"vendor":9007199254740993}`, `"wire-id"`, `"zone-a"`, `{"available":null,"vendor":9007199254740993}`, `{"node":{"vendor":[null,false,9007199254740993]}}`, 200},
		{"missing descriptors default to raw null", `{}`, `null`, `null`, `null`, `null`, 201},
		{"explicit null descriptors remain null", `{"id":null,"zoneName":null,"zoneState":null,"hosts":null}`, `null`, `null`, `null`, `null`, 300},
		{"numeric name does not become inherited ID", `{"zoneName":17,"zoneState":"opaque","hosts":false}`, `null`, `17`, `"opaque"`, `false`, 200},
		{"wire descriptor aliases later in dictionary win", `{"name":"canonical","zoneName":"wire","state":{"available":false},"zoneState":{"available":true}}`, `null`, `"wire"`, `{"available":true}`, `null`, 200},
		{"canonical descriptor aliases later in dictionary win", `{"zoneName":"wire","name":"canonical","zoneState":{"available":true},"state":{"available":false}}`, `null`, `"canonical"`, `{"available":false}`, `null`, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "test-token")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.55")
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 || r.ContentLength != 0 || r.URL.Path != computeAvailabilityZonesPath || r.URL.RawQuery != "" {
					t.Error("ordinary AZ reader changed route or sent a body/query", r.URL, string(body), err)
				}
				w.Header().Set("X-AZ-Page", "actual")
				testcloud.JSON(w, tc.status, `{"availabilityZoneInfo":[`+tc.row+`]}`)
			})
			var record *availabilityzones.AvailabilityZoneRecord
			count := 0
			for value, err := range availabilityzones.New(client).ListRecords(context.Background()) {
				if err != nil {
					t.Fatal(value, err)
				}
				record, count = value, count+1
			}
			if count != 1 || calls.Load() != 1 || record == nil || record.Resource == nil || record.Wire == nil || len(record.Resource.Body) != 5 || string(record.Resource.Body["id"]) != tc.id || string(record.Resource.Body["name"]) != tc.zone || string(record.Resource.Body["state"]) != tc.state || string(record.Resource.Body["hosts"]) != tc.hosts || string(record.Resource.Body["location"]) != `null` || string(record.Envelope) != tc.row || record.StatusCode != tc.status || record.Header.Get("X-AZ-Page") != "actual" || record.Wire.Header.Get("X-AZ-Page") != "actual" || client.Microversion != "2.55" {
				t.Fatal("owned AZ descriptors/defaults or receipt differ", record, count, calls.Load())
			}
			if _, exists := record.Resource.Body["vendor"]; exists {
				t.Fatal("unknown wire attribute became declared AZ field", record.Resource)
			}
			if tc.name == "raw descriptors and unknown precision" && string(record.Wire.Body["vendor"]) != `9007199254740993` {
				t.Fatal(record.Wire)
			}
			record.Resource.Body["name"][0] = 'x'
			record.Header.Set("X-AZ-Page", "changed")
			record.Envelope[0] = 'x'
			if record.Wire.Header.Get("X-AZ-Page") != "actual" {
				t.Fatal("record receipt aliases actual Wire header", record)
			}
			for _, raw := range record.Wire.Body {
				if !json.Valid(raw) {
					t.Fatal("view/Envelope aliases Wire JSON", record.Wire)
				}
			}
		})
	}
	for _, mode := range []string{"ordinary linked pages preserve duplicates", "empty ordinary page ignores advertised next", "consumer break stops next page", "late HTTP403 remains an iterator error"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			var calls atomic.Int32
			next := cloud.Server.URL + computeAvailabilityZonesPath + "?marker=page-2"
			encoded, _ := json.Marshal(next)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				index := calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "test-token")
				wantQuery := ""
				if index == 2 {
					wantQuery = "marker=page-2"
				}
				if r.URL.Path != computeAvailabilityZonesPath || r.URL.RawQuery != wantQuery {
					t.Error("owned AZ continuation changed route", r.URL)
				}
				w.Header().Set("X-AZ-Page", mode)
				if index == 2 && mode == "late HTTP403 remains an iterator error" {
					testcloud.JSON(w, 403, `{"error":"late AZ page forbidden"}`)
					return
				}
				rows, links := `{"zoneName":"same","zoneState":{"available":true}}`, ""
				if index == 1 {
					links = `,"links":[{"rel":"next","href":` + string(encoded) + `}]`
					if mode == "empty ordinary page ignores advertised next" {
						rows = ""
					}
				}
				testcloud.JSON(w, 200, `{"availabilityZoneInfo":[`+rows+`]`+links+`}`)
			})
			count, failures := 0, 0
			for record, err := range availabilityzones.New(client).ListRecords(context.Background()) {
				if err != nil {
					failures++
					var proof gophercloud.ErrUnexpectedResponseCode
					if record != nil || !errors.As(err, &proof) || proof.Actual != 403 || proof.URL != next || string(proof.Body) != `{"error":"late AZ page forbidden"}` || proof.ResponseHeader.Get("X-AZ-Page") != mode {
						t.Fatal(record, err, proof)
					}
					continue
				}
				count++
				if record == nil || string(record.Resource.Body["name"]) != `"same"` {
					t.Fatal(record)
				}
				if mode == "consumer break stops next page" {
					break
				}
			}
			wantCalls, wantRows, wantErrors := int32(2), 2, 0
			switch mode {
			case "empty ordinary page ignores advertised next":
				wantCalls, wantRows = 1, 0
			case "consumer break stops next page":
				wantCalls, wantRows = 1, 1
			case "late HTTP403 remains an iterator error":
				wantRows, wantErrors = 1, 1
			}
			if calls.Load() != wantCalls || count != wantRows || failures != wantErrors {
				t.Fatal(calls.Load(), count, failures)
			}
		})
	}
	t.Run("service iterator lazily owns options location and raw receipt separately", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.1"
		version := "2.55"
		captured := availabilityzones.WithAvailabilityZoneListOptions(availabilityzones.AvailabilityZoneListOpts{Microversion: &version})
		version = "2.1"
		cloudName, project := "az-cloud", json.RawMessage(`"scope"`)
		facts := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: project}}
		wantLocation, err := facts.ForResource(nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		var calls, locations, callbacks atomic.Int32
		service := compute.New(client, compute.Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations.Add(1); return facts, nil }})
		const row = `{"zoneName":"first","zoneState":{"available":false},"location":{"cloud":"wire-cloud"}}`
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestMethod(t, r, http.MethodGet)
			th.TestHeader(t, r, "X-AZ-Owned", "owned")
			th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
			th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.55")
			if r.URL.Path != computeAvailabilityZonesPath || r.URL.RawQuery != "" || locations.Load() != 1 || callbacks.Load() != 2 {
				t.Error(r.URL, locations.Load(), callbacks.Load())
			}
			testcloud.JSON(w, 200, `{"availabilityZoneInfo":[`+row+`,{"zoneName":"second","zoneState":{"available":true},"location":false}]}`)
		})
		headers := map[string]string{"X-AZ-Owned": "owned"}
		sequence := service.ListAvailabilityZones(context.Background(), captured,
			func(c *request.Config[availabilityzones.AvailabilityZoneListOpts]) error {
				callbacks.Add(1)
				if locations.Load() != 1 || calls.Load() != 0 {
					t.Error("AZ location was not captured before options", locations.Load(), calls.Load())
				}
				c.Headers = headers
				return nil
			}, func(_ *request.Config[availabilityzones.AvailabilityZoneListOpts]) error {
				callbacks.Add(1)
				headers["X-AZ-Owned"] = "caller-changed"
				cloudName = "caller-changed"
				project[1] = 'x'
				return nil
			})
		if calls.Load() != 0 || locations.Load() != 0 || callbacks.Load() != 0 {
			t.Fatal("AZ iterator creation eagerly ran source/location/options", calls.Load(), locations.Load(), callbacks.Load())
		}
		var records []*availabilityzones.AvailabilityZoneRecord
		for record, err := range sequence {
			if err != nil || record == nil || string(record.Resource.Body["location"]) != string(wantLocation) {
				t.Fatal(record, err, string(wantLocation))
			}
			records = append(records, record)
		}
		if len(records) != 2 || calls.Load() != 1 || locations.Load() != 1 || callbacks.Load() != 2 || client.Microversion != "2.1" || string(records[0].Envelope) != row || string(records[0].Wire.Body["location"]) != `{"cloud":"wire-cloud"}` || string(records[1].Wire.Body["location"]) != `false` {
			t.Fatal(records, calls.Load(), locations.Load(), callbacks.Load())
		}
		records[0].Resource.Body["location"][0] = 'x'
		if string(records[1].Resource.Body["location"]) != string(wantLocation) || string(records[0].Envelope) != row || !json.Valid(records[0].Wire.Body["location"]) {
			t.Fatal("computed AZ locations alias each other or physical evidence", records)
		}
	})
	t.Run("explicit empty version owns a versionless ordinary GET", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.55"
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
			th.TestHeaderUnset(t, r, "OpenStack-API-Version")
			th.TestHeader(t, r, "X-AZ-Version", "empty")
			if r.URL.Path != computeAvailabilityZonesPath || r.URL.RawQuery != "" {
				t.Error("ordinary AZ operation invented discovery", r.URL)
			}
			testcloud.JSON(w, 200, `{"availabilityZoneInfo":[]}`)
		})
		count := 0
		for record, err := range availabilityzones.New(client).ListRecords(context.Background(), availabilityzones.WithAvailabilityZoneMicroversion(""), availabilityzones.WithAvailabilityZoneHeader("X-AZ-Version", "empty")) {
			count++
			t.Fatal(record, err)
		}
		if count != 0 || calls.Load() != 1 || client.Microversion != "2.55" {
			t.Fatal(count, calls.Load(), client.Microversion)
		}
	})
	t.Run("service option source mutation stops later callback before HTTP", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.55"
		var calls, later atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
		count := 0
		for record, err := range compute.New(client, compute.Dependencies{}).ListAvailabilityZones(context.Background(),
			func(_ *request.Config[availabilityzones.AvailabilityZoneListOpts]) error {
				client.ResourceBase = cloud.Server.URL + "/changed/"
				return nil
			},
			func(_ *request.Config[availabilityzones.AvailabilityZoneListOpts]) error { later.Add(1); return nil }) {
			count++
			if record != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(record, err)
			}
		}
		if count != 1 || calls.Load() != 0 || later.Load() != 0 {
			t.Fatal(count, calls.Load(), later.Load())
		}
	})
}

func TestComputeAvailabilityZoneNamesTruthinessSuppressionAndPartialInventory(t *testing.T) {
	for _, tc := range []struct {
		name, rows           string
		unavailable, invalid bool
		wantNames            []string
		inventory            int
	}{
		{"default excludes unavailable and retains duplicate order", `{"zoneName":"same","zoneState":{"available":true}},{"zoneName":"down","zoneState":{"available":false}},{"zoneName":"same","zoneState":{"available":true}}`, false, false, []string{`"same"`, `"same"`}, 3},
		{"unavailable true includes both states in source order", `{"zoneName":"up","zoneState":{"available":true}},{"zoneName":"down","zoneState":{"available":false}}`, true, false, []string{`"up"`, `"down"`}, 2},
		{"available uses Python JSON truthiness", `{"zoneName":"zero","zoneState":{"available":0}},{"zoneName":"one","zoneState":{"available":1}},{"zoneName":"empty string","zoneState":{"available":""}},{"zoneName":"text","zoneState":{"available":"yes"}},{"zoneName":"empty array","zoneState":{"available":[]}},{"zoneName":"array","zoneState":{"available":[0]}},{"zoneName":"empty object","zoneState":{"available":{}}},{"zoneName":"object","zoneState":{"available":{"value":0}}},{"zoneName":"null","zoneState":{"available":null}}`, false, false, []string{`"one"`, `"text"`, `"array"`, `"object"`}, 9},
		{"names remain passive JSON including null and unknown precision", `{"zoneName":null,"zoneState":{"available":true}},{"zoneName":17,"zoneState":{"available":true}},{"zoneName":false,"zoneState":{"available":true}},{"zoneName":{"vendor":9007199254740993},"zoneState":{"available":true}},{"zoneName":[],"zoneState":{"available":true}},{"zoneState":{"available":true}}`, false, false, []string{`null`, `17`, `false`, `{"vendor":9007199254740993}`, `[]`, `null`}, 6},
		{"empty inventory completes an empty names array", "", false, false, []string{}, 0},
		{"canonical name and state aliases feed Cloud names", `{"name":"canonical","state":{"available":true}}`, false, false, []string{`"canonical"`}, 1},
		{"missing state fails before availability policy", `{"zoneName":"bad"}`, false, true, nil, 1},
		{"unavailable true does not bypass null state", `{"zoneName":"bad","zoneState":null}`, true, true, nil, 1},
		{"unavailable true does not bypass missing available key", `{"zoneName":"bad","zoneState":{}}`, true, true, nil, 1},
		{"unavailable true does not bypass string state", `{"zoneName":"bad","zoneState":"up"}`, true, true, nil, 1},
		{"unavailable true does not bypass array state", `{"zoneName":"bad","zoneState":[]}`, true, true, nil, 1},
		{"unavailable true does not bypass boolean state", `{"zoneName":"bad","zoneState":true}`, true, true, nil, 1},
		{"unavailable true does not bypass number state", `{"zoneName":"bad","zoneState":0}`, true, true, nil, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "test-token")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				if r.URL.Path != computeAvailabilityZonesPath || r.URL.RawQuery != "" {
					t.Error("Cloud names requested admin detail or query filtering", r.URL)
				}
				w.Header().Set("X-AZ-Names", "actual")
				testcloud.JSON(w, 200, `{"availabilityZoneInfo":[`+tc.rows+`]}`)
			})
			var options []compute.AvailabilityZoneNamesOption
			if tc.unavailable {
				options = append(options, compute.WithUnavailableZones(true))
			}
			result, err := compute.New(client, compute.Dependencies{}).ListAvailabilityZoneNames(context.Background(), options...)
			if result == nil || (err != nil) != tc.invalid || calls.Load() != 1 || len(result.Inventory) != tc.inventory || result.SuppressedError != nil {
				t.Fatal(result, err, calls.Load())
			}
			if tc.invalid {
				if !errors.Is(err, resource.ErrInvalidOption) || result.Value != nil || result.Names != nil || result.Inventory[0].StatusCode != 200 || result.Inventory[0].Header.Get("X-AZ-Names") != "actual" {
					t.Fatal("malformed state was swallowed or published as completed names", result, err)
				}
				return
			}
			names := make([]string, len(result.Names))
			for i, name := range result.Names {
				names[i] = string(name)
			}
			var values []json.RawMessage
			if !reflect.DeepEqual(names, tc.wantNames) || result.Value == nil || json.Unmarshal(result.Value, &values) != nil || len(values) != len(names) {
				t.Fatal("Cloud names lost source truthiness/order/passive values", result, names, tc.wantNames)
			}
			for i, name := range result.Names {
				if string(values[i]) != string(name) {
					t.Fatal(string(result.Value), result.Names)
				}
			}
			if len(result.Names) != 0 {
				result.Names[0][0] = 'x'
				if !json.Valid(result.Value) {
					t.Fatal("Names aliases completed JSON value", result)
				}
				for _, row := range result.Inventory {
					if !json.Valid(row.Resource.Body["name"]) || !json.Valid(row.Envelope) {
						t.Fatal("Names mutation changed inventory evidence", row)
					}
				}
			}
		})
	}
	for _, mode := range []string{"initial actual403 is suppressed", "initial actual500 is suppressed", "late actual404 discards completed names prefix", "repeated marker SDK cycle is suppressed", "accepted malformed JSON is not suppressed", "accepted Close404 is not suppressed", "source failure beside actual500 is not suppressed", "cancellation beside actual403 is not suppressed", "different retry cause beside actual500 is not suppressed"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("AZ Cloud names cancelled")
			retryCause := errors.New("AZ retry callback failed independently")
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Method: http.MethodGet, URL: "nested-AZ-close", Body: []byte("original AZ Close404")}
			var track *payloadContractTracking
			if mode == "accepted Close404 is not suppressed" {
				track = payloadContractTrack(cloud, nil, nested)
			}
			if mode == "source failure beside actual500 is not suppressed" || mode == "cancellation beside actual403 is not suppressed" {
				base := cloud.Provider.HTTPClient.Transport
				if base == nil {
					base = http.DefaultTransport
				}
				cloud.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					response, err := base.RoundTrip(r)
					if err == nil {
						if mode == "source failure beside actual500 is not suppressed" {
							client.Microversion = "2.99"
						} else {
							cancel(cause)
						}
					}
					return response, err
				})
			}
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				if mode == "different retry cause beside actual500 is not suppressed" {
					return retryCause
				}
				return err
			}
			next := cloud.Server.URL + computeAvailabilityZonesPath + "?marker=repeated"
			encoded, _ := json.Marshal(next)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				index := calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				if r.URL.Path != computeAvailabilityZonesPath || index == 1 && r.URL.RawQuery != "" || index == 2 && r.URL.RawQuery != "marker=repeated" {
					t.Error(r.URL)
				}
				w.Header().Set("X-AZ-Fault", "original")
				switch mode {
				case "initial actual403 is suppressed", "cancellation beside actual403 is not suppressed":
					testcloud.JSON(w, 403, `{"error":"AZ names denied"}`)
				case "initial actual500 is suppressed", "source failure beside actual500 is not suppressed", "different retry cause beside actual500 is not suppressed":
					testcloud.JSON(w, 500, `{"error":"AZ names failed"}`)
				case "accepted malformed JSON is not suppressed":
					testcloud.JSON(w, 200, `{"availabilityZoneInfo":!}`)
				case "late actual404 discards completed names prefix":
					if index == 2 {
						testcloud.JSON(w, 404, `{"error":"AZ names next missing"}`)
					} else {
						testcloud.JSON(w, 200, `{"availabilityZoneInfo":[{"zoneName":"kept","zoneState":{"available":true}}],"links":[{"rel":"next","href":`+string(encoded)+`}]}`)
					}
				case "repeated marker SDK cycle is suppressed":
					testcloud.JSON(w, 200, `{"availabilityZoneInfo":[{"zoneName":"kept","zoneState":{"available":true}}],"links":[{"rel":"next","href":`+string(encoded)+`}]}`)
				default:
					testcloud.JSON(w, 200, `{"availabilityZoneInfo":[{"zoneName":"kept","zoneState":{"available":true}}]}`)
				}
			})
			result, err := compute.New(client, compute.Dependencies{}).ListAvailabilityZoneNames(ctx)
			suppressed := mode == "initial actual403 is suppressed" || mode == "initial actual500 is suppressed" || mode == "late actual404 discards completed names prefix" || mode == "repeated marker SDK cycle is suppressed"
			wantCalls, wantInventory := int32(1), 0
			if mode == "late actual404 discards completed names prefix" {
				wantCalls, wantInventory = 2, 1
			} else if mode == "repeated marker SDK cycle is suppressed" {
				wantCalls, wantInventory = 2, 2
			}
			if result == nil || calls.Load() != wantCalls || len(result.Inventory) != wantInventory || (err == nil) != suppressed || (result.SuppressedError != nil) != suppressed {
				t.Fatal("Cloud AZ exception boundary differs", result, err, calls.Load(), wantInventory)
			}
			if suppressed {
				if result.Names == nil || len(result.Names) != 0 || string(result.Value) != `[]` {
					t.Fatal("suppressed late failure retained a completed name prefix", result)
				}
				if mode == "repeated marker SDK cycle is suppressed" {
					var cycle *resource.PaginationCycleError
					if !errors.As(result.SuppressedError, &cycle) {
						t.Fatal(result.SuppressedError)
					}
				} else {
					var proof gophercloud.ErrUnexpectedResponseCode
					wantStatus, wantURL, wantBody := 403, cloud.Server.URL+computeAvailabilityZonesPath, `{"error":"AZ names denied"}`
					if mode == "initial actual500 is suppressed" {
						wantStatus, wantBody = 500, `{"error":"AZ names failed"}`
					} else if mode == "late actual404 discards completed names prefix" {
						wantStatus, wantURL, wantBody = 404, next, `{"error":"AZ names next missing"}`
					}
					if !errors.As(result.SuppressedError, &proof) || proof.Actual != wantStatus || proof.URL != wantURL || string(proof.Body) != wantBody || proof.ResponseHeader.Get("X-AZ-Fault") != "original" {
						t.Fatal("suppressed cause lost original physical evidence", result.SuppressedError, proof)
					}
				}
				return
			}
			if result.Value != nil || result.Names != nil {
				t.Fatal("terminal fault published completed AZ names", result, err)
			}
			switch mode {
			case "different retry cause beside actual500 is not suppressed":
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.Is(err, retryCause) || !errors.As(err, &proof) || proof.Actual != 500 || retries.Load() != 1 {
					t.Fatal("mixed callback cause was suppressed", err, proof, retries.Load())
				}
			case "source failure beside actual500 is not suppressed":
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			case "cancellation beside actual403 is not suppressed":
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
					t.Fatal(err)
				}
			default:
				var receipt *resource.ResponseError
				if !errors.As(err, &receipt) || receipt.StatusCode != 200 || receipt.Header.Get("X-AZ-Fault") != "original" || retries.Load() != 0 {
					t.Fatal("accepted decode/physical fault became an HTTP suppression", err, receipt, retries.Load())
				}
				if track != nil {
					var proof gophercloud.ErrUnexpectedResponseCode
					physical := track.last(t)
					if !errors.As(err, &proof) || proof.URL != nested.URL || string(proof.Body) != string(nested.Body) || track.calls.Load() != 1 || physical.reads.Load() == 0 || physical.closes.Load() != 1 {
						t.Fatal(err, proof, physical)
					}
				}
			}
		})
	}
	t.Run("native cause from an option is not a fetched SDK exception", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.55"
		var calls, later atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
		original := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Method: http.MethodGet, URL: "option-native-cause", Body: []byte("option cause")}
		result, err := compute.New(client, compute.Dependencies{}).ListAvailabilityZoneNames(context.Background(),
			func(_ *request.Config[compute.AvailabilityZoneNamesOpts]) error { return original },
			func(_ *request.Config[compute.AvailabilityZoneNamesOpts]) error { later.Add(1); return nil })
		var proof gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &proof) || proof.URL != original.URL || calls.Load() != 0 || later.Load() != 0 || result != nil && (result.SuppressedError != nil || result.Value != nil || result.Names != nil) {
			t.Fatal("option native cause was swallowed as an actual fetch failure", result, err, calls.Load(), later.Load())
		}
	})
	t.Run("Cloud typed options own unavailable version headers and location once", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.1"
		version := "2.55"
		captured := compute.WithAvailabilityZoneNamesOptions(compute.AvailabilityZoneNamesOpts{Unavailable: true, Microversion: &version})
		version = "2.1"
		cloudName := "names-cloud"
		var locations, calls, callbacks atomic.Int32
		service := compute.New(client, compute.Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
			locations.Add(1)
			return resource.CloudLocation{Cloud: &cloudName}, nil
		}})
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestMethod(t, r, http.MethodGet)
			th.TestHeader(t, r, "X-AZ-Names", "owned")
			th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
			th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.55")
			if r.URL.Path != computeAvailabilityZonesPath || r.URL.RawQuery != "" || callbacks.Load() != 1 || locations.Load() != 1 {
				t.Error(r.URL, callbacks.Load(), locations.Load())
			}
			testcloud.JSON(w, 200, `{"availabilityZoneInfo":[{"zoneName":"down","zoneState":{"available":false},"location":false}]}`)
		})
		result, err := service.ListAvailabilityZoneNames(context.Background(), captured, compute.WithAvailabilityZoneNamesHeader("X-AZ-Names", "owned"),
			func(_ *request.Config[compute.AvailabilityZoneNamesOpts]) error {
				callbacks.Add(1)
				if locations.Load() != 1 || calls.Load() != 0 {
					t.Error("Cloud names options precede location snapshot", locations.Load(), calls.Load())
				}
				cloudName = "caller-changed"
				return nil
			})
		var location resource.CloudLocation
		if err != nil || result == nil || len(result.Names) != 1 || string(result.Names[0]) != `"down"` || string(result.Value) != `["down"]` || len(result.Inventory) != 1 || locations.Load() != 1 || callbacks.Load() != 1 || calls.Load() != 1 || client.Microversion != "2.1" || json.Unmarshal(result.Inventory[0].Resource.Body["location"], &location) != nil || location.Cloud == nil || *location.Cloud != "names-cloud" || string(result.Inventory[0].Wire.Body["location"]) != `false` {
			t.Fatal("Cloud names changed owned input/location or native source", result, err, locations.Load(), callbacks.Load(), calls.Load())
		}
	})
}

func TestComputeAvailabilityZoneNativeListTypedSinglePageABI(t *testing.T) {
	const native = `{"zoneName":"zone-a","zoneState":{"available":true},"hosts":{"node":{"nova-compute":{"active":true,"available":false,"updated_at":"2024-07-01T12:30:45.123"}}},"vendor":9007199254740993}`
	for _, tc := range []struct {
		name, body, failure string
		status, wantRows    int
	}{
		{"ordinary object envelope decodes typed hosts and ignores next", `{"availabilityZoneInfo":[` + native + `,{"zoneName":"zone-b","zoneState":{"available":false}}]}`, "", 200, 2},
		{"native300 retains typed ordinary fields", `{"availabilityZoneInfo":[` + native + `]}`, "", 300, 1},
		{"nullable and missing native fields use zero values", `{"availabilityZoneInfo":[{"zoneName":null,"zoneState":null,"hosts":null},{}]}`, "", 200, 2},
		{"null native plural is empty", `{"availabilityZoneInfo":null}`, "", 200, 0},
		{"missing native plural is empty", `{}`, "", 200, 0},
		{"native204 is accepted but empty JSON body returns EOF", "", "EOF", 204, 0},
		{"native201 preserves strict success codes and original receipt", `{"error":"native AZ unexpected201"}`, "HTTP", 201, 0},
		{"native403 preserves original HTTP evidence", `{"error":"native AZ forbidden"}`, "HTTP", 403, 0},
		{"later numeric native name discards the whole decoded page", `{"availabilityZoneInfo":[` + native + `,{"zoneName":17}]}`, "type", 200, 0},
		{"native available field requires a bool", `{"availabilityZoneInfo":[{"zoneName":"bad","zoneState":{"available":"yes"}}]}`, "type", 200, 0},
		{"native malformed JSON preserves syntax error", `{"availabilityZoneInfo":!}`, "syntax", 200, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "test-token")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				if r.URL.Path != computeAvailabilityZonesPath || r.URL.RawQuery != "" {
					t.Error("native single-page List followed advertised next or requested detail", r.URL)
				}
				w.Header().Set("X-Native-AZ", "actual")
				body := tc.body
				if tc.failure == "" && tc.wantRows != 0 {
					body = body[:len(body)-1] + `,"availabilityZoneInfo_links":[{"rel":"next","href":"` + cloud.Server.URL + computeAvailabilityZonesPath + `?marker=unused"}]}`
				}
				testcloud.JSON(w, tc.status, body)
			})
			sequence := availabilityzones.New(client).List(context.Background())
			if calls.Load() != 0 {
				t.Fatal("native AZ iterator performed eager HTTP")
			}
			var zones []*availabilityzones.AvailabilityZone
			var failure error
			errorsCount := 0
			for zone, err := range sequence {
				if err != nil {
					failure, errorsCount = err, errorsCount+1
					if zone != nil {
						t.Fatal("native whole-page decode returned a partial row", zone, err)
					}
				} else {
					zones = append(zones, zone)
				}
			}
			if calls.Load() != 1 || len(zones) != tc.wantRows || (failure != nil) != (tc.failure != "") || errorsCount > 1 || client.Microversion != "2.55" {
				t.Fatal(zones, failure, calls.Load(), errorsCount)
			}
			switch tc.failure {
			case "HTTP":
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(failure, &proof) || proof.Actual != tc.status || proof.Method != http.MethodGet || proof.URL != cloud.Server.URL+computeAvailabilityZonesPath || string(proof.Body) != tc.body || proof.ResponseHeader.Get("X-Native-AZ") != "actual" {
					t.Fatal("native HTTP cause lost original evidence", failure, proof)
				}
			case "EOF":
				if !errors.Is(failure, io.EOF) {
					t.Fatal(failure)
				}
			case "type":
				var mismatch *json.UnmarshalTypeError
				if !errors.As(failure, &mismatch) {
					t.Fatal(failure)
				}
			case "syntax":
				var syntax *json.SyntaxError
				if !errors.As(failure, &syntax) {
					t.Fatal(failure)
				}
			default:
				if tc.wantRows != 0 && tc.name != "nullable and missing native fields use zero values" {
					zone := zones[0]
					state := zone.Hosts["node"]["nova-compute"]
					if zone.ZoneName != "zone-a" || !zone.ZoneState.Available || !state.Active || state.Available || !state.UpdatedAt.Equal(time.Date(2024, 7, 1, 12, 30, 45, 123000000, time.UTC)) {
						t.Fatal("native known DTO/timestamp changed", zone, state)
					}
				} else {
					for _, zone := range zones {
						if zone == nil || zone.ZoneName != "" || zone.ZoneState.Available || zone.Hosts != nil {
							t.Fatal("nullable native fields acquired source raw defaults", zone)
						}
					}
				}
			}
		})
	}
}
