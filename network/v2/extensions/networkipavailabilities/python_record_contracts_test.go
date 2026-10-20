package networkipavailabilities_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func availabilityField(t *testing.T, r *networkipavailabilities.IPAvailabilityRecord, key, want string) {
	t.Helper()
	if r == nil || r.Resource == nil || string(r.Resource.Body[key]) != want {
		t.Fatal(key, want, r)
	}
}
func TestIPAvailabilityRecordsOwnedProjectionAndRawReceipt(t *testing.T) {
	var calls []nativeIPAvailabilityCall
	body := `{"network_ip_availabilities":[{"network_id":"n1","network_name":"public","tenant_id":"p","total_ips":"340282366920938463463374607431768211456","used_ips":3.9,"subnet_ip_availability":{"used_ips":"raw"},"ip_availability_details":false,"revision_number":"raw","self":"drop","x_new":{"n":9007199254740993}},{}]}`
	api, _ := nativeIPAvailabilityAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.Header.Get("Accept") != "application/json" {
			t.Error(req.Header)
		}
		return nativeIPAvailabilityWire(203, body)
	})
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 2 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	for key, want := range map[string]string{"id": "null", "name": "null", "network_id": `"n1"`, "network_name": `"public"`, "project_id": `"p"`, "total_ips": "340282366920938463463374607431768211456", "used_ips": "3", "subnet_ip_availability": `[{"used_ips":"raw"}]`, "ip_availability_details": "{}", "revision_number": `"raw"`, "location": "null"} {
		availabilityField(t, rows[0], key, want)
	}
	if rows[0].Resource.Body["self"] != nil || string(rows[0].Wire.Body["self"]) != `"drop"` || string(rows[0].Wire.Body["total_ips"]) != `"340282366920938463463374607431768211456"` || string(rows[0].Envelope) != body || rows[0].StatusCode != 203 {
		t.Fatal(rows[0])
	}
	rows[0].Resource.Body["x_new"][0] = 'X'
	rows[0].Header.Set("Content-Type", "changed")
	rows[0].Envelope[0] = 'X'
	if string(rows[0].Wire.Body["x_new"]) != `{"n":9007199254740993}` || rows[0].Wire.Header.Get("Content-Type") != "application/json" {
		t.Fatal("aliased receipt")
	}
	for _, key := range []string{"id", "name", "network_id", "network_name", "project_id", "tenant_id", "total_ips", "used_ips", "subnet_ip_availability", "ip_availability_details", "location"} {
		availabilityField(t, rows[1], key, "null")
	}
}
func TestIPAvailabilityRecordsAllPagesServerQueriesAndLocalFilters(t *testing.T) {
	var calls []nativeIPAvailabilityCall
	api, _ := nativeIPAvailabilityAPI(t, &calls, func(req *http.Request) *http.Response {
		q := req.URL.Query()
		if q.Get("network_name") != "public" || q.Get("ip_version") != "6" || q.Get("fields") != "network_id" || q.Get("total_ips") != "" || q.Get("unknown") != "" {
			t.Error(req.URL)
		}
		if q.Get("marker") == "p2" {
			return nativeIPAvailabilityWire(200, `{"network_ip_availabilities":[{"id":"second","network_name":"public","total_ips":2,"used_ips":0}]}`)
		}
		return nativeIPAvailabilityWire(200, `{"network_ip_availabilities":[{"id":"first","total_ips":1}],"network_ip_availabilities_links":[{"rel":"next","href":"?marker=p2"}]}`)
	})
	seq := api.ListRecords(context.Background(), networkipavailabilities.WithRecordListQuery("network_name", "public"), networkipavailabilities.WithRecordListQuery("ip_version", 6), networkipavailabilities.WithRecordListQuery("fields", []string{"network_id"}), networkipavailabilities.WithRecordListQuery("total_ips", 2), networkipavailabilities.WithRecordListQuery("unknown", make(chan int)))
	if len(calls) != 0 {
		t.Fatal("not lazy")
	}
	var rows []*networkipavailabilities.IPAvailabilityRecord
	for row, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if len(rows) != 1 || len(calls) != 2 {
		t.Fatal(rows, calls)
	}
	availabilityField(t, rows[0], "id", `"second"`)
}
func TestIPAvailabilityFindActualInheritedIdentityAndNoNameHint(t *testing.T) {
	for _, status := range []int{400, 403, 404} {
		for _, identity := range []string{"literal-id", "literal-name", "net-only", "public"} {
			t.Run(identity+string(rune(status)), func(t *testing.T) {
				var calls []nativeIPAvailabilityCall
				api, _ := nativeIPAvailabilityAPI(t, &calls, func(req *http.Request) *http.Response {
					if strings.HasSuffix(req.URL.Path, "/"+identity) {
						return nativeIPAvailabilityWire(status, `{}`)
					}
					if req.URL.RawQuery != "" {
						t.Error("unexpected hint", req.URL)
					}
					return nativeIPAvailabilityWire(200, `{"network_ip_availabilities":[{"id":"literal-id","name":"literal-name","network_id":"net-only","network_name":"public"}]}`)
				})
				got, err := api.FindRecord(context.Background(), identity)
				if err != nil || len(calls) != 2 {
					t.Fatal(got, err, calls)
				}
				if identity == "net-only" || identity == "public" {
					if got != nil {
						t.Fatal("invented network identity", got)
					}
				} else {
					availabilityField(t, got, "id", `"literal-id"`)
				}
			})
		}
	}
}
func TestIPAvailabilityFindDirectSparseSeedAndNoFallback(t *testing.T) {
	var calls []nativeIPAvailabilityCall
	api, _ := nativeIPAvailabilityAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.EscapedPath() != "/neutron/v2.0/network-ip-availabilities/literal%2Fid" || req.URL.Query().Get("network_name") != "seed" {
			t.Error(req.URL)
		}
		return nativeIPAvailabilityWire(200, `{"network_ip_availability":{"network_id":"server","used_ips":null,"x_new":false}}`)
	})
	got, err := api.FindRecord(context.Background(), "literal/id", networkipavailabilities.WithRecordFindQuery("network_name", "seed"))
	if err != nil || len(calls) != 1 {
		t.Fatal(got, err, calls)
	}
	availabilityField(t, got, "id", `"literal/id"`)
	availabilityField(t, got, "network_name", `"seed"`)
	availabilityField(t, got, "used_ips", "null")
	availabilityField(t, got, "x_new", "false")
	if got.Wire.Body["id"] != nil || got.Wire.Body["network_name"] != nil {
		t.Fatal("seed leaked to Wire", got)
	}
}
func TestIPAvailabilityFindMissingAmbiguityAndLateFailure(t *testing.T) {
	for _, lane := range []string{"missing", "required", "duplicate", "late-error", "network-id-only"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativeIPAvailabilityCall
			api, _ := nativeIPAvailabilityAPI(t, &calls, func(req *http.Request) *http.Response {
				if strings.HasSuffix(req.URL.Path, "/wanted") {
					return nativeIPAvailabilityWire(404, `{}`)
				}
				if req.URL.Query().Get("marker") == "later" {
					if lane == "late-error" {
						return nativeIPAvailabilityWire(503, `{"error":"later"}`)
					}
					return nativeIPAvailabilityWire(200, `{"network_ip_availabilities":[{"id":"second","name":"wanted"}]}`)
				}
				switch lane {
				case "duplicate", "late-error":
					return nativeIPAvailabilityWire(200, `{"network_ip_availabilities":[{"id":"first","name":"wanted"}],"next":"?marker=later"}`)
				case "network-id-only":
					return nativeIPAvailabilityWire(200, `{"network_ip_availabilities":[{"network_id":"wanted","network_name":"wanted"}]}`)
				default:
					return nativeIPAvailabilityWire(200, `{"network_ip_availabilities":[]}`)
				}
			})
			var opts []networkipavailabilities.RecordFindOption
			if lane == "required" {
				opts = append(opts, networkipavailabilities.WithRecordFindIgnoreMissing(false))
			}
			got, err := api.FindRecord(context.Background(), "wanted", opts...)
			if got != nil {
				t.Fatal(got)
			}
			switch lane {
			case "required":
				if !errors.Is(err, resource.ErrNotFound) {
					t.Fatal(err)
				}
			case "duplicate":
				if !errors.Is(err, resource.ErrAmbiguous) || len(calls) != 3 {
					t.Fatal(err, calls)
				}
			case "late-error":
				if !gophercloud.ResponseCodeIs(err, 503) || len(calls) != 3 {
					t.Fatal(err, calls)
				}
			default:
				if err != nil || len(calls) != 2 {
					t.Fatal(err, calls)
				}
			}
		})
	}
}
func TestIPAvailabilityRecordsCapsVersionsAndOwnedOptions(t *testing.T) {
	var calls []nativeIPAvailabilityCall
	var versions []string
	api, _ := nativeIPAvailabilityAPI(t, &calls, func(req *http.Request) *http.Response {
		versions = append(versions, req.Header.Get("OpenStack-API-Version"))
		if req.Header.Get("X-Trace") != "owned" {
			t.Error(req.Header)
		}
		if strings.HasSuffix(req.URL.Path, "/wanted") {
			return nativeIPAvailabilityWire(404, `{}`)
		}
		if req.URL.Path != "/neutron/v2.0/custom" || req.URL.Query().Get("limit") != "2" {
			t.Error(req.URL)
		}
		return nativeIPAvailabilityWire(200, `{"network_ip_availabilities":[{"id":"skip","total_ips":0},{"id":"wanted","total_ips":2},{"cls":null}],"next":"https://foreign.invalid/never"}`)
	})
	api.RawClient().Microversion = "2.0"
	path := "/custom"
	headers := map[string]string{"X-Trace": "owned"}
	opts := networkipavailabilities.WithRecordFindOpts(networkipavailabilities.RecordFindOpts{ListBasePath: &path, Headers: headers})
	path = "/changed"
	headers["X-Trace"] = "changed"
	got, err := api.FindRecord(context.Background(), "wanted", opts, networkipavailabilities.WithRecordFindMicroversion("2.1"), networkipavailabilities.WithRecordFindQuery("max_items", 2), networkipavailabilities.WithRecordFindQuery("total_ips", 2))
	if err != nil || got == nil || len(calls) != 2 || !reflect.DeepEqual(versions, []string{"network 2.1", "network 2.0"}) || api.RawClient().Microversion != "2.0" {
		t.Fatal(got, err, calls, versions)
	}
}

type availabilityFaultBody struct {
	data  []byte
	cause error
}

func (b *availabilityFaultBody) Read(p []byte) (int, error) {
	n := copy(p, b.data)
	b.data = b.data[n:]
	return n, b.cause
}
func (b *availabilityFaultBody) Close() error { return b.cause }
func TestIPAvailabilityRecordsPhysicalFailuresAndPreflight(t *testing.T) {
	cause := errors.New("physical")
	for _, status := range []int{200, 403, 404} {
		var calls []nativeIPAvailabilityCall
		api, cloud := nativeIPAvailabilityAPI(t, &calls, func(*http.Request) *http.Response {
			r := nativeIPAvailabilityWire(status, `{}`)
			r.Body = &availabilityFaultBody{[]byte(`{"actual":true}`), cause}
			return r
		})
		callbacks := 0
		cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			callbacks++
			return nil
		}
		got, err := api.FindRecord(context.Background(), "wanted")
		if got != nil || !errors.Is(err, cause) || len(calls) != 1 || callbacks != 0 {
			t.Fatal(got, err, calls, callbacks)
		}
	}
	for _, lane := range []string{"nil-option", "source", "cancel", "constructor"} {
		var calls []nativeIPAvailabilityCall
		api, _ := nativeIPAvailabilityAPI(t, &calls, func(*http.Request) *http.Response {
			t.Error("preflight reached HTTP")
			return nativeIPAvailabilityWire(200, `{}`)
		})
		ctx := context.Background()
		var opts []networkipavailabilities.RecordFindOption
		switch lane {
		case "nil-option":
			opts = append(opts, nil)
		case "source":
			opts = append(opts, func(*networkipavailabilities.RecordFindOpts) error {
				api.RawClient().ResourceBase += "changed/"
				return nil
			})
		case "cancel":
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			cancel()
		case "constructor":
			opts = append(opts, networkipavailabilities.WithRecordFindQuery("id", "collision"))
		}
		_, err := api.FindRecord(ctx, "wanted", opts...)
		if err == nil || len(calls) != 0 {
			t.Fatal(lane, err, calls)
		}
	}
}
func TestIPAvailabilityProxyGeneratorAndJSONProjection(t *testing.T) {
	var calls []nativeIPAvailabilityCall
	api, _ := nativeIPAvailabilityAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeIPAvailabilityWire(200, `{"network_ip_availabilities":[]}`)
	})
	result, err := api.ProxyListRecords(context.Background(), networkipavailabilities.WithRecordListJMESPath("@"))
	if err != nil || len(calls) != 0 {
		t.Fatal(result, err, calls)
	}
	if _, ok := result.Value.(*networkipavailabilities.RecordListGenerator); !ok {
		t.Fatal(result)
	}
	result, err = api.ProxyListRecords(context.Background(), networkipavailabilities.WithRecordListJMESPath("[].network_id"))
	if err != nil || string(result.JSON) != "null" || len(calls) != 0 {
		t.Fatal(result, err, calls)
	}
	_, err = api.ProxyListRecords(context.Background(), networkipavailabilities.WithRecordListJMESPath("length(@)"))
	if err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}
