package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/compute/v2/flavors"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestComputeFlavorRecordsQueryProjectionAndOwnedViews(t *testing.T) {
	for _, tc := range []struct {
		name, row string
		want      map[string]string
		singleton bool
	}{
		{"declared defaults", `{}`, nil, false},
		{"raw descriptor conversions and original-name identity", `{"id":0,"name":"","original_name":"7","description":{"raw":17},"disk":true,"ram":"0008","vcpus":3.9,"swap":"-9","OS-FLV-EXT-DATA:ephemeral":"٤","os-flavor-access:is_public":"false","OS-FLV-DISABLED:disabled":[],"rxtx_factor":" 2.5 ","extra_specs":[1],"vendor":9007199254740993}`, map[string]string{"id": `"7"`, "name": `""`, "original_name": `"7"`, "description": `{"raw":17}`, "disk": `true`, "ram": `8`, "vcpus": `3`, "swap": `0`, "ephemeral": `4`, "is_public": `true`, "is_disabled": `false`, "rxtx_factor": `2.5`}, false},
		{"explicit nullable attributes", `{"id":null,"name":null,"original_name":null,"description":null,"disk":null,"ram":null,"vcpus":null,"swap":null,"ephemeral":null,"is_public":null,"is_disabled":null,"rxtx_factor":null,"extra_specs":null}`, map[string]string{"disk": `null`, "ram": `null`, "vcpus": `null`, "swap": `null`, "ephemeral": `null`, "is_public": `null`, "extra_specs": `null`}, false},
		{"wire aliases last in response win", `{"id":"7","is_public":true,"os-flavor-access:is_public":false,"ephemeral":7,"OS-FLV-EXT-DATA:ephemeral":3}`, map[string]string{"id": `"7"`, "is_public": `false`, "ephemeral": `3`}, false},
		{"canonical aliases last in response win", `{"id":"7","os-flavor-access:is_public":false,"is_public":true,"OS-FLV-EXT-DATA:ephemeral":3,"ephemeral":7}`, map[string]string{"id": `"7"`, "is_public": `true`, "ephemeral": `7`}, false},
		{"nested flavor remains an unknown flat row field", `{"id":"outer","flavor":{"id":"inner","name":"inner","ram":99}}`, map[string]string{"id": `"outer"`}, false},
		{"omitted name aliases original_name", `{"original_name":"7"}`, map[string]string{"id": `"7"`, "name": `"7"`, "original_name": `"7"`}, false},
		{"numeric identity remains passive and bool float converts", `{"id":9007199254740993,"name":"other","rxtx_factor":true}`, map[string]string{"id": `9007199254740993`, "name": `"other"`, "rxtx_factor": `1`}, false},
		{"plural object is a singleton", `{"id":"single"}`, map[string]string{"id": `"single"`}, true},
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
				th.TestHeader(t, r, "Accept", "application/json")
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 || r.ContentLength != 0 || r.URL.Path != flavorIdentityPath+"/detail" || !reflect.DeepEqual(r.URL.Query(), url.Values{"is_public": {"None"}}) {
					t.Error("owned flavor default changed", r.URL, string(body), err)
				}
				w.Header().Set("X-Flavor-Page", tc.name)
				if tc.singleton {
					testcloud.JSON(w, 201, `{"flavors":`+tc.row+`}`)
				} else {
					testcloud.JSON(w, 201, flavorIdentityPage(tc.row, ""))
				}
			})
			var records []*flavors.FlavorRecord
			for value, err := range flavors.New(client).ListRecords(context.Background()) {
				if err != nil {
					t.Fatal(err)
				}
				records = append(records, value)
			}
			if len(records) != 1 || calls.Load() != 1 || records[0] == nil || records[0].Resource == nil || records[0].Wire == nil {
				t.Fatal(records, calls.Load())
			}
			value := records[0]
			want := map[string]string{"id": `null`, "name": `null`, "original_name": `null`, "description": `null`, "disk": `0`, "ram": `0`, "vcpus": `0`, "swap": `0`, "ephemeral": `0`, "is_public": `true`, "is_disabled": `null`, "rxtx_factor": `null`, "extra_specs": `{}`, "location": `null`}
			for key, raw := range tc.want {
				want[key] = raw
			}
			if len(value.Resource.Body) != len(want) || string(value.Envelope) != tc.row || value.StatusCode != 201 || value.Header.Get("X-Flavor-Page") != tc.name || value.Enrichment != nil {
				t.Fatal("row evidence or declared view differs", value)
			}
			for key, raw := range want {
				if string(value.Resource.Body[key]) != raw {
					t.Fatal("flavor descriptor differs", key, string(value.Resource.Body[key]), raw, value.Resource)
				}
			}
			if _, exists := value.Resource.Body["vendor"]; exists {
				t.Fatal("unknown actual field became declared view", value.Resource)
			}
			if tc.name == "raw descriptor conversions and original-name identity" && (string(value.Wire.Body["id"]) != `0` || string(value.Wire.Body["ram"]) != `"0008"` || string(value.Wire.Body["vendor"]) != `9007199254740993` || string(value.Wire.Body["extra_specs"]) != `[1]`) {
				t.Fatal("actual row was normalized", value.Wire)
			}
			if tc.name == "nested flavor remains an unknown flat row field" && string(value.Wire.Body["flavor"]) != `{"id":"inner","name":"inner","ram":99}` {
				t.Fatal(value.Wire)
			}
			value.Resource.Body["id"][0] = 'x'
			if string(value.Resource.Body["name"]) != want["name"] {
				t.Fatal("logical identity aliases another view field", value.Resource)
			}
			value.Envelope[0] = 'x'
			value.Header.Set("X-Flavor-Page", "changed")
			value.Resource.Header.Set("X-Flavor-Page", "changed")
			if value.Wire.Header.Get("X-Flavor-Page") != tc.name || value.Wire.StatusCode != 201 {
				t.Fatal("owned headers alias actual row", value)
			}
			for _, raw := range value.Wire.Body {
				if !json.Valid(raw) {
					t.Fatal("view or envelope aliases Wire bytes", value.Wire)
				}
			}
		})
	}
	for _, mode := range []string{"default detail", "summary null public", "empty public", "false public", "canonical seven queries win aliases", "raw overrides typed query"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			path, want := flavorIdentityPath+"/detail", url.Values{"is_public": {"None"}}
			var options []flavors.FlavorListOption
			switch mode {
			case "summary null public":
				path = flavorIdentityPath
				options = append(options, flavors.WithFlavorListDetails(false), flavors.WithFlavorListFilter("is_public", nil))
			case "empty public":
				options = append(options, flavors.WithFlavorListFilter("is_public", ""))
				want.Set("is_public", "")
			case "false public":
				options = append(options, flavors.WithFlavorListFilter("is_public", false))
				want.Set("is_public", "false")
			case "canonical seven queries win aliases":
				options = append(options, flavors.WithFlavorListFilters(map[string]any{"limit": 2, "marker": "start", "sort_key": "name", "sort_dir": "asc", "is_public": true, "min_disk": 4, "minDisk": 99, "min_ram": 3, "minRam": 88}))
				want = url.Values{"limit": {"2"}, "marker": {"start"}, "sort_key": {"name"}, "sort_dir": {"asc"}, "is_public": {"true"}, "minDisk": {"4"}, "minRam": {"3"}}
			case "raw overrides typed query":
				options = append(options, flavors.WithFlavorListOptions(flavors.FlavorListOpts{Limit: 2, Marker: "start"}), flavors.WithFlavorListQuery("limit", "1"), flavors.WithFlavorListQuery("vendor", "literal"))
				want.Set("limit", "1")
				want.Set("marker", "start")
				want.Set("vendor", "literal")
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				if r.URL.Path != path || !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error(r.URL, want)
				}
				testcloud.JSON(w, 200, flavorIdentityPage("", ""))
			})
			for value, err := range flavors.New(client).ListRecords(context.Background(), options...) {
				t.Fatal("empty list yielded", value, err)
			}
			if calls.Load() != 1 {
				t.Fatal(calls.Load())
			}
		})
	}
	for _, mode := range []string{"all twelve local attributes", "bool and number remain distinct", "unknown semantic function ignored"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			const hit = `{"id":"7","name":"small","original_name":"original","description":{"text":"detail"},"disk":"8","ram":true,"vcpus":3,"swap":0,"OS-FLV-EXT-DATA:ephemeral":4,"OS-FLV-DISABLED:disabled":"false","os-flavor-access:is_public":false,"rxtx_factor":"2.5","extra_specs":{"cpu":"dedicated"}}`
			filters := map[string]any{"id": "7", "name": "small", "original_name": "original", "description": map[string]any{"text": "detail"}, "disk": 8, "ram": true, "vcpus": 3, "swap": 0, "ephemeral": 4, "is_disabled": true, "rxtx_factor": 2.5, "extra_specs": map[string]any{"cpu": "dedicated"}}
			wantCount := 1
			if mode == "bool and number remain distinct" {
				filters, wantCount = map[string]any{"ram": 1}, 0
			} else if mode == "unknown semantic function ignored" {
				filters = map[string]any{"vendor": func() {}, "os-flavor-access:is_public": true}
				wantCount = 2
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != flavorIdentityPath+"/detail" || r.URL.RawQuery != "is_public=None" {
					t.Error("local attribute leaked into query", r.URL)
				}
				testcloud.JSON(w, 200, flavorIdentityPage(hit+`,{"id":"miss","name":"miss","ram":false}`, ""))
			})
			count := 0
			for value, err := range flavors.New(client).ListRecords(context.Background(), flavors.WithFlavorListFilters(filters)) {
				if err != nil || value == nil {
					t.Fatal(value, err)
				}
				count++
			}
			if count != wantCount || calls.Load() != 1 {
				t.Fatal(count, wantCount, calls.Load())
			}
		})
	}
	for _, mode := range []string{"semantic collides typed", "semantic collides raw", "unsupported base path"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			options := []flavors.FlavorListOption{flavors.WithFlavorListFilter("limit", 2), flavors.WithFlavorListLimit(2)}
			want := resource.ErrInvalidOption
			if mode == "semantic collides raw" {
				options = []flavors.FlavorListOption{flavors.WithFlavorListFilter("min_ram", 2), flavors.WithFlavorListQuery("minRam", "2")}
			} else if mode == "unsupported base path" {
				options = []flavors.FlavorListOption{flavors.WithFlavorListQuery("base_path", "/other")}
				want = resource.ErrUnsupported
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				t.Error("invalid flavor query touched HTTP", r.URL)
				w.WriteHeader(500)
			})
			failures := 0
			for value, err := range flavors.New(client).ListRecords(context.Background(), options...) {
				failures++
				if value != nil || !errors.Is(err, want) {
					t.Fatal(value, err)
				}
			}
			if failures != 1 || calls.Load() != 0 {
				t.Fatal(failures, calls.Load())
			}
		})
	}
}

func TestComputeFlavorRecordsPagingRawCapsAndLazyOwnership(t *testing.T) {
	for _, mode := range []string{"links", "flavors_links", "next", "HTTP Link", "dictionary compatibility", "initial limit short-page fallback", "server limit only", "single page", "empty page stops", "late403", "break", "accepted empty204 is an error"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			var calls, callbacks atomic.Int32
			options := []flavors.FlavorListOption{flavors.WithFlavorListFilter("disk", 1), func(_ *request.Config[flavors.FlavorListOpts]) error { callbacks.Add(1); return nil }}
			if mode == "initial limit short-page fallback" {
				options = append(options, flavors.WithFlavorListLimit(5))
			}
			if mode == "single page" {
				options = append(options, flavors.WithFlavorListPaginated(false))
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				call := calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				if r.URL.Path != flavorIdentityPath+"/detail" || r.URL.Query().Get("is_public") != "None" || callbacks.Load() != 1 || call > 3 {
					t.Error(r.URL, call, callbacks.Load())
				}
				if call > 1 {
					th.TestHeader(t, r, "X-Auth-Token", "next-flavor-live")
					wantMarker := "wire-last"
					if call == 3 {
						wantMarker = "second"
					}
					if r.URL.Query().Get("marker") != wantMarker {
						t.Error("marker did not use last consumed raw identity", r.URL)
					}
					if mode == "server limit only" && r.URL.Query().Get("limit") != "25" {
						t.Error(r.URL)
					}
					if mode == "late403" {
						w.Header().Set("X-Flavor-Page", "late")
						testcloud.JSON(w, 403, `{"error":"late flavor forbidden"}`)
					} else if call == 3 {
						testcloud.JSON(w, 200, flavorIdentityPage("", ""))
					} else {
						testcloud.JSON(w, 200, flavorIdentityPage(`{"id":"second","disk":1}`, ""))
					}
					return
				}
				th.TestHeader(t, r, "X-Auth-Token", "test-token")
				cloud.Provider.SetToken("next-flavor-live")
				if mode == "accepted empty204 is an error" {
					w.Header().Set("X-Flavor-Page", "accepted204")
					w.WriteHeader(204)
					return
				}
				next := cloud.Server.URL + flavorIdentityPath + "/detail?is_public=None&marker=wire-last"
				if mode == "server limit only" {
					next += "&limit=25"
				}
				continuation := ""
				switch mode {
				case "links", "server limit only", "empty page stops", "single page", "break", "late403":
					continuation = `,"links":[{"rel":"next","href":"` + next + `"}]`
				case "flavors_links":
					continuation = `,"flavors_links":[{"rel":"next","href":"` + next + `"}]`
				case "next":
					continuation = `,"next":"` + next + `"`
				case "HTTP Link":
					w.Header().Set("Link", "<"+next+">; rel=\"next\"")
				case "dictionary compatibility":
					// This dictionary representation is a Go compatibility extension.
					continuation = `,"links":{"next":"` + next + `"}`
				}
				rows := `{"id":"first","disk":1},{"id":null,"name":"","original_name":"wire-last","disk":2}`
				if mode == "empty page stops" {
					rows = ""
				}
				testcloud.JSON(w, 300, `{"flavors":[`+rows+`]`+continuation+`}`)
			})
			sequence := flavors.New(client).ListRecords(context.Background(), options...)
			if calls.Load() != 0 || callbacks.Load() != 0 {
				t.Fatal("owned iterator was eager", calls.Load(), callbacks.Load())
			}
			var ids []string
			var failure error
			for value, err := range sequence {
				if err != nil {
					failure = err
					break
				}
				var id string
				if err := json.Unmarshal(value.Resource.Body["id"], &id); err != nil {
					t.Fatal(err)
				}
				ids = append(ids, id)
				value.Resource.Body["id"][0] = 'x'
				if mode == "break" {
					break
				}
			}
			wantIDs, wantCalls := []string{"first", "second"}, int32(2)
			switch mode {
			case "initial limit short-page fallback":
				wantCalls = 3
			case "single page", "break":
				wantIDs, wantCalls = []string{"first"}, 1
			case "late403":
				wantIDs = []string{"first"}
			case "empty page stops", "accepted empty204 is an error":
				wantIDs, wantCalls = nil, 1
			}
			if !reflect.DeepEqual(ids, wantIDs) || calls.Load() != wantCalls || callbacks.Load() != 1 {
				t.Fatal(ids, wantIDs, failure, calls.Load(), wantCalls, callbacks.Load())
			}
			if mode == "late403" {
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(failure, &proof) || proof.Actual != 403 || string(proof.Body) != `{"error":"late flavor forbidden"}` || proof.ResponseHeader.Get("X-Flavor-Page") != "late" {
					t.Fatal(failure, proof)
				}
			} else if mode == "accepted empty204 is an error" {
				var receipt *resource.ResponseError
				if !errors.As(failure, &receipt) || receipt.StatusCode != 204 || len(receipt.Body) != 0 || receipt.Header.Get("X-Flavor-Page") != "accepted204" {
					t.Fatal(failure, receipt)
				}
			} else if failure != nil {
				t.Fatal(failure)
			}
		})
	}
	for _, maximum := range []int{1, 2, 0} {
		t.Run("physical raw cap "+strconv.Itoa(maximum), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			const body = `{"flavors":[{"id":"drop","disk":2},{"id":"hit","disk":1},false]}`
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if maximum > 0 && r.URL.Query().Get("limit") != strconv.Itoa(maximum) {
					t.Error(r.URL)
				}
				w.Header().Set("X-Flavor-Cap", "actual")
				testcloud.JSON(w, 200, body)
			})
			count := 0
			var failure error
			for _, err := range flavors.New(client).ListRecords(context.Background(), flavors.WithFlavorListMaxItems(maximum), flavors.WithFlavorListFilter("disk", 1)) {
				if err != nil {
					failure = err
					break
				}
				count++
			}
			want := 1
			if maximum == 1 {
				want = 0
			}
			if count != want || calls.Load() != 1 {
				t.Fatal(count, want, failure, calls.Load())
			}
			if maximum == 0 {
				var receipt *resource.ResponseError
				if !errors.As(failure, &receipt) || receipt.StatusCode != 200 || string(receipt.Body) != body || receipt.Header.Get("X-Flavor-Cap") != "actual" {
					t.Fatal(failure, receipt)
				}
			} else if failure != nil {
				t.Fatal("flavor list decoded beyond cap", failure)
			}
		})
	}
	t.Run("lazy callbacks own pointer header query and filter carriers", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.55"
		details, version := false, "2.99"
		bulk := flavors.WithFlavorListOptions(flavors.FlavorListOpts{Details: &details, Microversion: &version})
		details, version = true, "changed"
		filters := map[string]any{"name": "hit"}
		filter := flavors.WithFlavorListFilters(filters)
		filters["name"] = "caller-changed"
		headers := map[string]string{"X-Flavor-Trace": "owned"}
		query := url.Values{"vendor": {"owned"}}
		var calls, first, later atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestHeader(t, r, "X-Flavor-Trace", "owned")
			th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.99")
			if r.URL.Path != flavorIdentityPath || !reflect.DeepEqual(r.URL.Query(), url.Values{"is_public": {"None"}, "vendor": {"owned"}}) {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, flavorIdentityPage(`{"id":"7","name":"hit"},{"id":"8","name":"miss"}`, ""))
		})
		sequence := flavors.New(client).ListRecords(context.Background(), bulk, filter,
			func(c *request.Config[flavors.FlavorListOpts]) error {
				first.Add(1)
				c.Headers, c.Query = headers, query
				return nil
			},
			func(_ *request.Config[flavors.FlavorListOpts]) error {
				later.Add(1)
				headers["X-Flavor-Trace"] = "changed"
				query["vendor"][0] = "changed"
				return nil
			})
		if calls.Load() != 0 || first.Load() != 0 || later.Load() != 0 {
			t.Fatal("owned options executed before iteration")
		}
		count := 0
		for value, err := range sequence {
			if err != nil || value == nil || string(value.Resource.Body["name"]) != `"hit"` {
				t.Fatal(value, err)
			}
			count++
		}
		if count != 1 || calls.Load() != 1 || first.Load() != 1 || later.Load() != 1 || client.Microversion != "2.55" {
			t.Fatal(count, calls.Load(), first.Load(), later.Load(), client)
		}
	})
}

func TestComputeFlavorRecordsConditionalExtraSpecsEnrichment(t *testing.T) {
	for _, tc := range []struct {
		name, row, want string
		enabled, fetch  bool
	}{
		{"default leaves missing specs without a GET", `{"id":"7"}`, `{}`, false, false},
		{"nonempty inline dict skips separate GET", `{"id":"7","extra_specs":{"cpu":"inline"}}`, `{"cpu":"inline"}`, true, false},
		{"missing dict requires GET", `{"id":"7"}`, `{"cpu":"fresh"}`, true, true},
		{"null dict requires GET", `{"id":"7","extra_specs":null}`, `{"cpu":"fresh"}`, true, true},
		{"empty dict requires GET", `{"id":"7","extra_specs":{}}`, `{"cpu":"fresh"}`, true, true},
		{"scalar becomes empty dict and uses original-name identity", `{"id":null,"name":"","original_name":"7","extra_specs":1}`, `{"cpu":"fresh"}`, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			var lists, specs, callbacks atomic.Int32
			const specsBody = `{"extra_specs":{"cpu":"fresh"},"vendor":9007199254740993}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.55")
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 || r.ContentLength != 0 || callbacks.Load() != 1 {
					t.Error(r.URL, string(body), err, callbacks.Load())
				}
				switch r.URL.Path {
				case flavorIdentityPath + "/detail":
					lists.Add(1)
					th.TestHeader(t, r, "X-Auth-Token", "test-token")
					if r.URL.RawQuery != "is_public=None" {
						t.Error(r.URL)
					}
					cloud.Provider.SetToken("enrichment-live")
					w.Header().Set("X-Flavor-Page", "row")
					testcloud.JSON(w, 200, flavorIdentityPage(tc.row, ""))
				case flavorExtraSpecsFetchPath:
					specs.Add(1)
					th.TestHeader(t, r, "X-Auth-Token", "enrichment-live")
					if r.URL.RawQuery != "" {
						t.Error("list query leaked into specs GET", r.URL)
					}
					w.Header().Set("X-Spec-Proof", "child")
					testcloud.JSON(w, 201, specsBody)
				default:
					t.Error("enrichment invented another lookup", r.URL)
					w.WriteHeader(500)
				}
			})
			var result *flavors.FlavorRecord
			count := 0
			for value, err := range flavors.New(client).ListRecords(context.Background(), flavors.WithFlavorListExtraSpecs(tc.enabled), func(_ *request.Config[flavors.FlavorListOpts]) error { callbacks.Add(1); return nil }) {
				if err != nil {
					t.Fatal(value, err)
				}
				result, count = value, count+1
			}
			wantSpecs := int32(0)
			if tc.fetch {
				wantSpecs = 1
			}
			if count != 1 || result == nil || result.Resource == nil || result.Wire == nil || string(result.Resource.Body["extra_specs"]) != tc.want || string(result.Envelope) != tc.row || result.StatusCode != 200 || result.Header.Get("X-Flavor-Page") != "row" || lists.Load() != 1 || specs.Load() != wantSpecs || callbacks.Load() != 1 || client.Microversion != "2.55" {
				t.Fatal(result, count, lists.Load(), specs.Load(), callbacks.Load())
			}
			if tc.fetch {
				child := result.Enrichment
				if child == nil || child.Resource == nil || child.Wire == nil || string(child.ExtraSpecs) != tc.want || string(child.Envelope) != specsBody || child.StatusCode != 201 || child.Header.Get("X-Spec-Proof") != "child" || string(child.Wire.Body["vendor"]) != `9007199254740993` {
					t.Fatal("separate enrichment receipt was lost", result)
				}
				if raw, exists := result.Wire.Body["extra_specs"]; exists && string(raw) == tc.want {
					t.Fatal("enrichment rewrote actual row Wire", result)
				}
				result.Resource.Body["extra_specs"][0] = 'x'
				if string(child.ExtraSpecs) != tc.want || string(child.Resource.Body["extra_specs"]) != tc.want || string(child.Wire.Body["extra_specs"]) != tc.want {
					t.Fatal("enriched view aliases child carriers", result)
				}
			} else if result.Enrichment != nil {
				t.Fatal("inline/default operation fabricated enrichment", result)
			}
		})
	}
	for _, mode := range []string{"filtered row is not enriched", "cap excludes later row", "break excludes later row"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			var lists, specs atomic.Int32
			options := []flavors.FlavorListOption{flavors.WithFlavorListExtraSpecs(true)}
			rows := `{"id":"7","disk":2},{"id":"8","disk":1}`
			if mode == "filtered row is not enriched" {
				options = append(options, flavors.WithFlavorListFilter("disk", 1))
			} else if mode == "cap excludes later row" {
				options = append(options, flavors.WithFlavorListMaxItems(1))
			}
			wantID := "7"
			if mode == "filtered row is not enriched" {
				wantID = "8"
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == flavorIdentityPath+"/detail" {
					lists.Add(1)
					testcloud.JSON(w, 200, flavorIdentityPage(rows, ""))
					return
				}
				specs.Add(1)
				if r.URL.Path != flavorIdentityPath+"/"+wantID+"/os-extra_specs" || r.URL.RawQuery != "" {
					t.Error("filtered/capped/broken row was enriched", r.URL, wantID)
				}
				testcloud.JSON(w, 200, `{"extra_specs":{"cpu":"fresh"}}`)
			})
			count := 0
			for value, err := range flavors.New(client).ListRecords(context.Background(), options...) {
				if err != nil || value == nil || string(value.Resource.Body["id"]) != `"`+wantID+`"` || value.Enrichment == nil {
					t.Fatal(value, err)
				}
				count++
				if mode == "break excludes later row" {
					break
				}
			}
			if count != 1 || lists.Load() != 1 || specs.Load() != 1 {
				t.Fatal(count, lists.Load(), specs.Load())
			}
		})
	}
	t.Run("numeric logical identity is not coerced into a specs route", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.55"
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.URL.Path != flavorIdentityPath+"/detail" {
				t.Error("numeric response identity became executable path", r.URL)
			}
			testcloud.JSON(w, 200, flavorIdentityPage(`{"id":7}`, ""))
		})
		failures := 0
		for value, err := range flavors.New(client).ListRecords(context.Background(), flavors.WithFlavorListExtraSpecs(true)) {
			failures++
			if value == nil || value.Resource == nil || string(value.Resource.Body["id"]) != `7` || !errors.Is(err, resource.ErrInvalidOption) || value.Enrichment != nil {
				t.Fatal(value, err)
			}
		}
		if failures != 1 || calls.Load() != 1 {
			t.Fatal(failures, calls.Load())
		}
	})
	for _, mode := range []string{"specs actual404 is terminal", "accepted specs Close404 is terminal"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Method: http.MethodGet, URL: "nested-enrichment", Body: []byte("physical specs404")}
			var track *payloadContractTracking
			if mode == "accepted specs Close404 is terminal" {
				track = payloadContractTrack(cloud, nil, nil)
				base := cloud.Provider.HTTPClient.Transport
				cloud.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					response, err := base.RoundTrip(r)
					if err == nil && r.URL.Path == flavorExtraSpecsFetchPath {
						response.Body.(*payloadContractBody).closeErr = nested
					}
					return response, err
				})
			}
			var lists, specs atomic.Int32
			const row = `{"id":"7"}`
			body, status := `{"error":"specs missing"}`, 404
			if track != nil {
				body, status = `{"extra_specs":{"cpu":"actual"}}`, 200
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == flavorIdentityPath+"/detail" {
					lists.Add(1)
					testcloud.JSON(w, 200, flavorIdentityPage(row, ""))
					return
				}
				specs.Add(1)
				if r.URL.Path != flavorExtraSpecsFetchPath {
					t.Error(r.URL)
				}
				w.Header().Set("X-Spec-Proof", "actual")
				testcloud.JSON(w, status, body)
			})
			failures := 0
			for value, err := range flavors.New(client).ListRecords(context.Background(), flavors.WithFlavorListExtraSpecs(true)) {
				failures++
				if err == nil || value == nil || value.Wire == nil || string(value.Envelope) != row || string(value.Wire.Body["id"]) != `"7"` || errors.Is(err, resource.ErrNotFound) {
					t.Fatal("enrichment error lost original row or became missing", value, err)
				}
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &proof) || proof.Actual != 404 {
					t.Fatal(err, proof)
				}
				if track == nil {
					if proof.URL != cloud.Server.URL+flavorExtraSpecsFetchPath || string(proof.Body) != body || proof.ResponseHeader.Get("X-Spec-Proof") != "actual" {
						t.Fatal(err, proof)
					}
				} else {
					var receipt *resource.ResponseError
					if !errors.As(err, &receipt) || receipt.StatusCode != 200 || string(receipt.Body) != body || value.Enrichment == nil || value.Enrichment.Resource != nil || value.Enrichment.StatusCode != 200 || string(value.Enrichment.Envelope) != body || proof.URL != "nested-enrichment" || string(proof.Body) != "physical specs404" {
						t.Fatal(value, err, receipt, proof)
					}
					physical := track.last(t)
					if track.calls.Load() != 2 || physical.closes.Load() != 1 || physical.reads.Load() == 0 {
						t.Fatal(track.calls.Load(), physical)
					}
				}
			}
			if failures != 1 || lists.Load() != 1 || specs.Load() != 1 {
				t.Fatal(failures, lists.Load(), specs.Load())
			}
		})
	}
	t.Run("one discovery and options prepare list plus child at2.61", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Endpoint = cloud.Server.URL + "/reverse/nova/v2.1/project/"
		var discovery, lists, specs, callbacks atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			th.TestMethod(t, r, http.MethodGet)
			if callbacks.Load() != 1 {
				t.Error("option was reapplied during enrichment", callbacks.Load())
			}
			switch r.URL.Path {
			case computeConsoleDiscoveryPath:
				discovery.Add(1)
				th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
				th.TestHeaderUnset(t, r, "OpenStack-API-Version")
				testcloud.JSON(w, 201, `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.110"}}`)
			case flavorIdentityPath + "/detail":
				lists.Add(1)
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.61")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.61")
				testcloud.JSON(w, 200, flavorIdentityPage(`{"id":"7"}`, ""))
			case flavorExtraSpecsFetchPath:
				specs.Add(1)
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.61")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.61")
				if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"extra_specs":{}}`)
			default:
				t.Error("enrichment performed unrelated discovery/lookup", r.URL)
				w.WriteHeader(500)
			}
		})
		count := 0
		for value, err := range flavors.New(client).ListRecords(context.Background(), flavors.WithFlavorListExtraSpecs(true), func(_ *request.Config[flavors.FlavorListOpts]) error { callbacks.Add(1); return nil }) {
			if err != nil || value == nil || value.Enrichment == nil {
				t.Fatal(value, err)
			}
			count++
		}
		if count != 1 || discovery.Load() != 1 || lists.Load() != 1 || specs.Load() != 1 || callbacks.Load() != 1 || client.Microversion != "" {
			t.Fatal(count, discovery.Load(), lists.Load(), specs.Load(), callbacks.Load(), client)
		}
	})
}
