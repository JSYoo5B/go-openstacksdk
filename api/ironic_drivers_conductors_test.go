package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/baremetal/apiversions"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/conductors"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/drivers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ---- API versions ----

const ironicAPIVersionsBody = `{"default_version":{"id":"v1","min_version":"1.1","status":"CURRENT","version":"1.99"},"versions":[{"id":"v1","min_version":"1.1","status":"CURRENT","version":"1.99"}]}`

func TestIronicAPIVersionListAndGetUseRootRoutesAndHeader(t *testing.T) {
	cloud := testcloud.New(t)
	type seen struct{ method, path, header string }
	var requests []seen
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, seen{r.Method, r.URL.Path, r.Header.Get(ironicAPIVersionHeader)})
		switch r.URL.Path {
		case "/":
			testcloud.JSON(w, 200, ironicAPIVersionsBody)
		case "/v1":
			testcloud.JSON(w, 200, `{"version":{"id":"v1","min_version":"1.1","status":"CURRENT","version":"1.99"}}`)
		default:
			testcloud.JSON(w, 404, `{}`)
		}
	})
	client := cloud.Client("baremetal", "/")
	client.Microversion = "1.50"
	api := apiversions.New(client)
	list, err := api.List(context.Background())
	if err != nil || list.DefaultVersion.Version != "1.99" || len(list.Versions) != 1 || list.Versions[0].MinVersion != "1.1" || list.Versions[0].Status != "CURRENT" {
		t.Fatalf("list=%+v error=%v", list, err)
	}
	one, err := api.Get(context.Background(), "v1")
	if err != nil || one.ID != "v1" || one.Version != "1.99" {
		t.Fatalf("version=%+v error=%v", one, err)
	}
	want := []seen{{"GET", "/", "1.50"}, {"GET", "/v1", "1.50"}}
	if !reflect.DeepEqual(requests, want) {
		t.Fatalf("requests=%v", requests)
	}
}

func TestIronicAPIVersionAcceptsOnly200AndKeepsNativeErrors(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(ironicAPIVersionHeader); got != "" {
			t.Errorf("no microversion header expected, got %q", got)
		}
		switch r.URL.Path {
		case "/":
			w.WriteHeader(204)
		case "/v1":
			testcloud.JSON(w, 202, `{"version":{}}`)
		default:
			testcloud.JSON(w, 404, `{}`)
		}
	})
	api := apiversions.New(cloud.Client("baremetal", "/"))
	_, err := api.List(context.Background())
	ironicRequireStatus(t, err, 204)
	_, err = api.Get(context.Background(), "v1")
	ironicRequireStatus(t, err, 202)
	_, err = api.Get(context.Background(), "v9")
	ironicRequireStatus(t, err, 404)
}

func TestIronicAPIVersionCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud, calls := ironicBlockingCloud(t)
	api := apiversions.New(cloud.Client("baremetal", "/"))
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.List(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled list: %v", err)
	}
	if _, err := api.Get(canceled, "v1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.List(deadline); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}

// ---- Drivers ----

const ironicDriverBody = `{"name":"redfish","type":"dynamic","hosts":["conductor-01"],"default_power_interface":"redfish","enabled_power_interfaces":["redfish","fake"],"links":[],"properties":[]}`

func newIronicDrivers(t *testing.T, cloud *testcloud.Cloud, version string) *drivers.API {
	t.Helper()
	return drivers.New(ironicClient(cloud, "baremetal", version))
}

func TestIronicDriverListFiltersFollowLinksAndExtensionQuery(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v1/drivers", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "redfish" {
			testcloud.JSON(w, 200, `{"drivers":[{"name":"ipmi"}]}`)
			return
		}
		next := cloud.Server.URL + "/v1/drivers?marker=redfish"
		testcloud.JSON(w, 200, `{"drivers":[`+ironicDriverBody+`],"drivers_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	var names []string
	for value, err := range newIronicDrivers(t, cloud, "").ListDrivers(context.Background(),
		drivers.WithListDriversOptions(drivers.ListDriversOpts{Detail: true, Type: "dynamic"}),
		drivers.WithListDriversQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, value.Name)
	}
	if !reflect.DeepEqual(names, []string{"redfish", "ipmi"}) || len(requests) != 2 {
		t.Fatalf("names=%v requests=%v", names, requests)
	}
	want := url.Values{"detail": {"true"}, "type": {"dynamic"}, "vendor": {"x"}}
	if !reflect.DeepEqual(requests[0], want) || requests[1].Get("marker") != "redfish" {
		t.Fatalf("queries=%v", requests)
	}
}

func TestIronicDriverListOmitsFalseDetailAndRejectsInvalidOptionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []string
	cloud.Mux.HandleFunc("GET /v1/drivers", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		testcloud.JSON(w, 200, `{"drivers":[]}`)
	})
	api := newIronicDrivers(t, cloud, "")
	for _, err := range api.ListDrivers(context.Background(), drivers.WithListDriversOptions(drivers.ListDriversOpts{Detail: false})) {
		if err != nil {
			t.Fatal(err)
		}
	}
	// The native Detail bool is not a pointer, so false is indistinguishable from omitted.
	if !reflect.DeepEqual(queries, []string{""}) {
		t.Fatalf("queries=%q", queries)
	}
	before := len(queries)
	for _, err := range api.ListDrivers(context.Background(), nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil option: %v", err)
		}
	}
	for _, err := range api.ListDrivers(context.Background(), drivers.WithListDriversQuery(" ", "x")) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("empty query key: %v", err)
		}
	}
	if len(queries) != before {
		t.Fatalf("invalid options reached HTTP: %q", queries)
	}
}

func TestIronicDriverReadsDetailsPropertiesDiskPropertiesAndPassthruMethods(t *testing.T) {
	cloud := testcloud.New(t)
	var headers []string
	record := func(w http.ResponseWriter, r *http.Request, body string) {
		headers = append(headers, r.Method+" "+r.URL.Path+" "+r.Header.Get(ironicAPIVersionHeader))
		testcloud.JSON(w, 200, body)
	}
	cloud.Mux.HandleFunc("GET /v1/drivers/redfish", func(w http.ResponseWriter, r *http.Request) { record(w, r, ironicDriverBody) })
	cloud.Mux.HandleFunc("GET /v1/drivers/redfish/properties", func(w http.ResponseWriter, r *http.Request) {
		record(w, r, `{"redfish_address":"required","redfish_verify_ca":"optional"}`)
	})
	cloud.Mux.HandleFunc("GET /v1/drivers/redfish/raid/logical_disk_properties", func(w http.ResponseWriter, r *http.Request) {
		record(w, r, `{"size_gb":"Size in GiB."}`)
	})
	cloud.Mux.HandleFunc("GET /v1/drivers/redfish/vendor_passthru/methods", func(w http.ResponseWriter, r *http.Request) {
		record(w, r, `{"reset":{"http_methods":["POST"],"async":false,"description":"Reset","attach":false}}`)
	})
	api := newIronicDrivers(t, cloud, "1.45")
	ctx := context.Background()
	detail, err := api.GetDriverDetails(ctx, "redfish")
	if err != nil || detail.Name != "redfish" || detail.Type != "dynamic" || !reflect.DeepEqual(detail.EnabledPowerInterfaces, []string{"redfish", "fake"}) {
		t.Fatalf("detail=%+v error=%v", detail, err)
	}
	props, err := api.GetDriverProperties(ctx, "redfish")
	if err != nil || (*props)["redfish_address"] != "required" {
		t.Fatalf("properties=%v error=%v", props, err)
	}
	disk, err := api.GetDriverDiskProperties(ctx, "redfish")
	if err != nil || (*disk)["size_gb"] != "Size in GiB." {
		t.Fatalf("disk properties=%v error=%v", disk, err)
	}
	methods, err := api.ListVendorPassthruMethods(ctx, "redfish")
	if err != nil || !reflect.DeepEqual(methods["reset"], drivers.VendorPassthruMethod{HTTPMethods: []string{"POST"}, Async: false, Description: "Reset", Attach: false}) {
		t.Fatalf("methods=%v error=%v", methods, err)
	}
	if len(headers) != 4 {
		t.Fatalf("requests=%v", headers)
	}
	for _, line := range headers {
		if line[len(line)-4:] != "1.45" {
			t.Fatalf("microversion header missing: %q", line)
		}
	}
}

func TestIronicDriverReadsRequireStatus200AndKeepNative404(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/v1/drivers/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/drivers/missing" || r.URL.Path == "/v1/drivers/missing/properties" {
			testcloud.JSON(w, 404, `{"error_message":"gone"}`)
			return
		}
		testcloud.JSON(w, 202, `{}`)
	})
	api := newIronicDrivers(t, cloud, "")
	ctx := context.Background()
	_, err := api.GetDriverDetails(ctx, "missing")
	ironicRequireStatus(t, err, 404)
	_, err = api.GetDriverProperties(ctx, "missing")
	ironicRequireStatus(t, err, 404)
	_, err = api.GetDriverDetails(ctx, "redfish")
	ironicRequireStatus(t, err, 202)
	_, err = api.GetDriverProperties(ctx, "redfish")
	ironicRequireStatus(t, err, 202)
	_, err = api.GetDriverDiskProperties(ctx, "redfish")
	ironicRequireStatus(t, err, 202)
	_, err = api.ListVendorPassthruMethods(ctx, "redfish")
	ironicRequireStatus(t, err, 202)
}

func TestIronicDriverCallVendorPassthruSendsMethodQueryAndBody(t *testing.T) {
	cloud := testcloud.New(t)
	type seen struct{ method, query, body string }
	var requests []seen
	cloud.Mux.HandleFunc("/v1/drivers/redfish/vendor_passthru", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		requests = append(requests, seen{r.Method, r.URL.RawQuery, string(data)})
		switch r.URL.Query().Get("method") {
		case "sync":
			testcloud.JSON(w, 200, `{"ok":true}`)
		case "async":
			w.WriteHeader(202)
		case "none":
			w.WriteHeader(204)
		default:
			testcloud.JSON(w, 201, `{}`)
		}
	})
	api := newIronicDrivers(t, cloud, "")
	ctx := context.Background()
	got, err := api.CallVendorPassthru(ctx, "redfish", http.MethodPost, drivers.VendorPassthruCallOpts{Method: "sync", Body: map[string]any{"enabled": false, "tags": []string{}, "none": nil}})
	if err != nil || got["ok"] != true {
		t.Fatalf("sync result=%v error=%v", got, err)
	}
	// An accepted asynchronous call has no body; acknowledgement is not completion.
	got, err = api.CallVendorPassthru(ctx, "redfish", http.MethodPost, drivers.VendorPassthruCallOpts{Method: "async"})
	if err != nil || got != nil {
		t.Fatalf("async result=%v error=%v", got, err)
	}
	got, err = api.CallVendorPassthru(ctx, "redfish", http.MethodGet, drivers.VendorPassthruCallOpts{Method: "none"})
	if err != nil || got != nil {
		t.Fatalf("204 result=%v error=%v", got, err)
	}
	_, err = api.CallVendorPassthru(ctx, "redfish", http.MethodPut, drivers.VendorPassthruCallOpts{Method: "a b&c"}, drivers.WithCallVendorPassthruOptions(drivers.VendorPassthruCallOpts{Method: "other"}))
	ironicRequireStatus(t, err, 201)
	want := []seen{
		{"POST", "method=sync", `{"enabled":false,"none":null,"tags":[]}`},
		{"POST", "method=async", ""},
		{"GET", "method=none", ""},
		{"PUT", "method=other", ""},
	}
	if !reflect.DeepEqual(requests, want) {
		t.Fatalf("requests=%v", requests)
	}
}

func TestIronicDriverCallVendorPassthruEscapesMethodAndRejectsInvalidInputBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []string
	cloud.Mux.HandleFunc("/v1/drivers/redfish/vendor_passthru", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		testcloud.JSON(w, 200, `{}`)
	})
	api := newIronicDrivers(t, cloud, "")
	if _, err := api.CallVendorPassthru(context.Background(), "redfish", http.MethodPost, drivers.VendorPassthruCallOpts{Method: "a b&c"}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(queries, []string{"method=a+b%26c"}) {
		t.Fatalf("queries=%q", queries)
	}
	if _, err := api.CallVendorPassthru(context.Background(), "redfish", http.MethodPost, drivers.VendorPassthruCallOpts{}); err == nil {
		t.Fatal("empty passthru method must fail before HTTP")
	}
	if _, err := api.CallVendorPassthru(context.Background(), "redfish", http.MethodPost, drivers.VendorPassthruCallOpts{Method: "x"}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option: %v", err)
	}
	if len(queries) != 1 {
		t.Fatalf("invalid input reached HTTP: %q", queries)
	}
}

func TestIronicDriverCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud, calls := ironicBlockingCloud(t)
	api := newIronicDrivers(t, cloud, "")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.GetDriverDetails(canceled, "redfish"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled details: %v", err)
	}
	if _, err := api.CallVendorPassthru(canceled, "redfish", http.MethodPost, drivers.VendorPassthruCallOpts{Method: "x"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled passthru: %v", err)
	}
	for _, err := range api.ListDrivers(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled list: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.CallVendorPassthru(deadline, "redfish", http.MethodPost, drivers.VendorPassthruCallOpts{Method: "x"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}

// ---- Conductors ----

const ironicConductorBody = `{"hostname":"conductor-01","alive":false,"drivers":["redfish"],"conductor_group":"","created_at":"2026-01-02T03:04:05+00:00","updated_at":null}`

func newIronicConductors(t *testing.T, cloud *testcloud.Cloud, version string) *conductors.API {
	t.Helper()
	return conductors.New(ironicClient(cloud, "baremetal", version))
}

func TestIronicConductorGetKeepsFalseAndStatusPolicy(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v1/conductors/{name}", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get(ironicAPIVersionHeader); got != "1.49" {
			t.Errorf("microversion header=%q", got)
		}
		switch r.PathValue("name") {
		case "conductor-01":
			testcloud.JSON(w, 200, ironicConductorBody)
		case "accepted":
			testcloud.JSON(w, 202, ironicConductorBody)
		default:
			testcloud.JSON(w, 404, `{"error_message":"gone"}`)
		}
	})
	api := newIronicConductors(t, cloud, "1.49")
	value, err := api.Get(context.Background(), "conductor-01")
	if err != nil || value.Hostname != "conductor-01" || value.Alive || value.ConductorGroup != "" || !reflect.DeepEqual(value.Drivers, []string{"redfish"}) || value.CreatedAt.IsZero() || !value.UpdatedAt.IsZero() {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	_, err = api.Get(context.Background(), "accepted")
	ironicRequireStatus(t, err, 202)
	_, err = api.Get(context.Background(), "missing")
	ironicRequireStatus(t, err, 404)
	if _, err := api.Resources.Find(context.Background(), resource.ID("missing")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("Collection lookup adds ErrNotFound: %v", err)
	}
}

func TestIronicConductorListFiltersFollowLinksAndRejectDetailWithFields(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v1/conductors", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "conductor-01" {
			testcloud.JSON(w, 200, `{"conductors":[{"hostname":"conductor-02"}]}`)
			return
		}
		next := cloud.Server.URL + "/v1/conductors?marker=conductor-01&limit=1"
		// Gophercloud 2.15.0 reads the singular conductor_links key.
		testcloud.JSON(w, 200, `{"conductors":[`+ironicConductorBody+`],"conductor_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	api := newIronicConductors(t, cloud, "1.49")
	var names []string
	for value, err := range api.List(context.Background(),
		conductors.WithListOptions(conductors.ListOpts{Fields: []string{"hostname", "alive"}, Limit: 1, SortKey: "hostname", SortDir: "asc"}),
		conductors.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, value.Hostname)
	}
	if !reflect.DeepEqual(names, []string{"conductor-01", "conductor-02"}) || len(requests) != 2 {
		t.Fatalf("names=%v requests=%v", names, requests)
	}
	want := url.Values{"fields": {"hostname,alive"}, "limit": {"1"}, "sort_key": {"hostname"}, "sort_dir": {"asc"}, "vendor": {"x"}}
	if !reflect.DeepEqual(requests[0], want) || requests[1].Get("marker") != "conductor-01" {
		t.Fatalf("queries=%v", requests)
	}
	requests = nil
	for _, err := range api.List(context.Background(), conductors.WithListOptions(conductors.ListOpts{Detail: true})) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(requests) == 0 || requests[0].Get("detail") != "true" {
		t.Fatalf("detail requests=%v", requests)
	}
	before := len(requests)
	for _, err := range api.List(context.Background(), conductors.WithListOptions(conductors.ListOpts{Detail: true, Fields: []string{"hostname"}})) {
		if err == nil {
			t.Fatal("detail with fields must fail before HTTP")
		}
	}
	for _, err := range api.List(context.Background(), nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil option: %v", err)
		}
	}
	if len(requests) != before {
		t.Fatalf("invalid options reached HTTP: %v", requests)
	}
}

func TestIronicConductorCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud, calls := ironicBlockingCloud(t)
	api := newIronicConductors(t, cloud, "1.49")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(canceled, "conductor-01"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	for _, err := range api.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled list: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Get(deadline, "conductor-01"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
