package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/compute"
	"github.com/JSYoo5B/gophercloudsdk/compute/v2/flavors"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

// Pinned Cloud _compute.py:165-181, 278-289, 524-561. These tables verify
// composition of existing owned Flavor readers and Cloud selection, not a
// second paging, discovery, descriptor, matcher, or physical-fault matrix.
func TestComputeCloudFlavorEagerInventoryDefaultsAndPartialEvidence(t *testing.T) {
	const first = `{"id":"7","name":"blue-2","extra_specs":{}},{"id":"8","name":"red","extra_specs":{"cpu":"inline"}}`
	const second = `{"id":"9","name":"blue-1","extra_specs":null},{"id":"10","name":"blue-2","extra_specs":{"cpu":"inline"}}`
	const partial = `{"id":"6","name":"red","extra_specs":{"cpu":"inline"}},{"id":"7","name":"blue-1"},{"id":"8","name":"unconsumed"}`
	for _, tc := range []struct {
		name, operation, identity, filters, rows, nextRows, failure string
		extra, inventory                                            int
		wantNames, wantSpecs                                        []string
	}{
		{"all default false eagerly collects detail pages and duplicates", "all", "", "", first, second, "", -1, 4, []string{`"blue-2"`, `"red"`, `"blue-1"`, `"blue-2"`}, nil},
		{"all explicit true enriches falsey specs", "all", "", "", first, second, "", 1, 4, []string{`"blue-2"`, `"red"`, `"blue-1"`, `"blue-2"`}, []string{"7", "9"}},
		{"all explicit false skips specs", "all", "", "", `{"id":"7","name":"blue-1"}`, "", "", 0, 1, []string{`"blue-1"`}, nil},
		{"search default true enriches the entire inventory before glob", "search", "blue-*", "", first, second, "", -1, 4, []string{`"blue-2"`, `"blue-1"`, `"blue-2"`}, []string{"7", "9"}},
		{"search explicit false skips specs", "search", "blue-*", "", first, second, "", 0, 4, []string{`"blue-2"`, `"blue-1"`, `"blue-2"`}, nil},
		{"search enriches unmatched rows before returning empty selection", "search", "missing", "", `{"id":"7","name":"other"}`, "", "", -1, 1, []string{}, []string{"7"}},
		{"search dictionary sees fetched specs after complete inventory", "search", "", `{"extra_specs":{"cpu":"fresh"}}`, first, second, "", -1, 4, []string{`"blue-2"`, `"blue-1"`}, []string{"7", "9"}},
		{"all empty inventory is a completed JSON array", "all", "", "", "", "", "", -1, 0, []string{}, nil},
		{"search empty inventory is a completed JSON array", "search", "", "", "", "", "", -1, 0, []string{}, nil},
		{"all late page403 retains inventory without completed list", "all", "", "", `{"id":"7","name":"blue-1","extra_specs":{"cpu":"inline"}}`, `{"id":"8","name":"unused"}`, "page403", -1, 1, nil, nil},
		{"late page403 retains inventory without completed selection", "search", "blue-*", "", `{"id":"7","name":"blue-1","extra_specs":{"cpu":"inline"}}`, `{"id":"8","name":"unused"}`, "page403", -1, 1, nil, nil},
		{"child actual404 includes the failing row in partial inventory", "search", "blue-*", "", partial, "", "child404", -1, 2, nil, []string{"7"}},
		{"accepted child Close404 retains both row and child receipts", "search", "blue-*", "", partial, "", "childClose", -1, 2, nil, []string{"7"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			service := compute.New(client, compute.Dependencies{})
			var pages, calls, callbacks atomic.Int32
			var mu sync.Mutex
			var specIDs []string
			next := cloud.Server.URL + flavorIdentityPath + "/detail?is_public=None&marker=page-2"
			const childBody = `{"extra_specs":{"cpu":"fresh"},"vendor":9007199254740993}`
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Method: http.MethodGet, URL: "nested-cloud-flavor-specs", Body: []byte("original child Close404")}
			var track *payloadContractTracking
			if tc.failure == "childClose" {
				track = payloadContractTrack(cloud, nil, nil)
				base := cloud.Provider.HTTPClient.Transport
				cloud.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					response, err := base.RoundTrip(r)
					if err == nil && r.URL.Path == flavorIdentityPath+"/7/os-extra_specs" {
						response.Body.(*payloadContractBody).closeErr = nested
					}
					return response, err
				})
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.55")
				th.TestHeader(t, r, "Accept", "application/json")
				th.TestHeader(t, r, "X-Cloud-Flavor", "owned")
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 || r.ContentLength != 0 || callbacks.Load() != 1 {
					t.Error("Cloud reader changed body or reapplied callbacks", r.URL, string(body), err, callbacks.Load())
				}
				if r.URL.Path == flavorIdentityPath+"/detail" {
					page := pages.Add(1)
					wantQuery := url.Values{"is_public": {"None"}}
					if page == 2 {
						wantQuery.Set("marker", "page-2")
					}
					if !reflect.DeepEqual(r.URL.Query(), wantQuery) || r.URL.RawQuery != wantQuery.Encode() {
						t.Error("Cloud search filters leaked into the server list", r.URL, wantQuery)
					}
					if page == 1 {
						th.TestHeader(t, r, "X-Auth-Token", "test-token")
						cloud.Provider.SetToken("cloud-flavor-live")
					} else {
						th.TestHeader(t, r, "X-Auth-Token", "cloud-flavor-live")
					}
					w.Header().Set("X-Cloud-Page", tc.name)
					if page == 2 && tc.failure == "page403" {
						testcloud.JSON(w, 403, `{"error":"late Cloud flavor inventory denied"}`)
						return
					}
					rows, advertised := tc.rows, ""
					if page == 2 {
						rows = tc.nextRows
					} else if tc.nextRows != "" {
						advertised = next
					}
					testcloud.JSON(w, 200, flavorIdentityPage(rows, advertised))
					return
				}
				th.TestHeader(t, r, "X-Auth-Token", "cloud-flavor-live")
				id := ""
				if r.URL.Path == flavorIdentityPath+"/7/os-extra_specs" {
					id = "7"
				} else if r.URL.Path == flavorIdentityPath+"/9/os-extra_specs" {
					id = "9"
				} else {
					t.Error("Cloud composition added another lookup", r.URL)
				}
				mu.Lock()
				specIDs = append(specIDs, id)
				mu.Unlock()
				if r.URL.RawQuery != "" {
					t.Error("Cloud selection leaked into specs GET", r.URL)
				}
				w.Header().Set("X-Cloud-Specs", "child")
				if tc.failure == "child404" {
					testcloud.JSON(w, 404, `{"error":"Cloud flavor specs missing"}`)
					return
				}
				testcloud.JSON(w, 201, childBody)
			})
			options := []compute.FlavorQueryOption{
				compute.WithFlavorQueryHeader("X-Cloud-Flavor", "owned"),
				func(_ *request.Config[compute.FlavorQueryOpts]) error { callbacks.Add(1); return nil },
			}
			if tc.extra >= 0 {
				options = append(options, compute.WithFlavorQueryExtraSpecs(tc.extra == 1))
			}
			if tc.filters != "" {
				options = append(options, compute.WithFlavorQueryFilters(json.RawMessage(tc.filters)))
			}
			var result *compute.FlavorQueryResult
			var err error
			if tc.operation == "all" {
				result, err = service.AllFlavors(context.Background(), options...)
			} else {
				result, err = service.SearchFlavors(context.Background(), tc.identity, options...)
			}
			mu.Lock()
			gotSpecs := append([]string(nil), specIDs...)
			mu.Unlock()
			wantPages := int32(1)
			if tc.nextRows != "" {
				wantPages = 2
			}
			if result == nil || (err != nil) != (tc.failure != "") || len(result.Inventory) != tc.inventory || pages.Load() != wantPages || calls.Load() != wantPages+int32(len(tc.wantSpecs)) || callbacks.Load() != 1 || !reflect.DeepEqual(gotSpecs, tc.wantSpecs) || client.Microversion != "2.55" {
				t.Fatal("Cloud eager/default/enrichment composition differs", result, err, pages.Load(), calls.Load(), callbacks.Load(), gotSpecs)
			}
			if tc.failure != "" {
				if result.Value != nil || result.Flavors != nil || errors.Is(err, resource.ErrNotFound) {
					t.Fatal("partial inventory became a completed selection or missing", result, err)
				}
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &proof) {
					t.Fatal("original HTTP cause lost", err)
				}
				if tc.failure == "page403" {
					if proof.Actual != 403 || proof.URL != next || string(proof.Body) != `{"error":"late Cloud flavor inventory denied"}` || proof.ResponseHeader.Get("X-Cloud-Page") != tc.name {
						t.Fatal(proof)
					}
					return
				}
				row := result.Inventory[1]
				if row == nil || row.Resource == nil || row.Wire == nil || string(row.Resource.Body["id"]) != `"7"` || string(row.Envelope) != `{"id":"7","name":"blue-1"}` || row.StatusCode != 200 || row.Header.Get("X-Cloud-Page") != tc.name {
					t.Fatal("failing child discarded the original inventory row", row, err)
				}
				if track == nil {
					if proof.Actual != 404 || proof.URL != cloud.Server.URL+flavorIdentityPath+"/7/os-extra_specs" || string(proof.Body) != `{"error":"Cloud flavor specs missing"}` || proof.ResponseHeader.Get("X-Cloud-Specs") != "child" || row.Enrichment != nil {
						t.Fatal(row, proof)
					}
				} else {
					var receipt *resource.ResponseError
					physical := track.last(t)
					if proof.URL != nested.URL || string(proof.Body) != string(nested.Body) || !errors.As(err, &receipt) || receipt.StatusCode != 201 || string(receipt.Body) != childBody || row.Enrichment == nil || row.Enrichment.Resource != nil || row.Enrichment.Wire != nil || row.Enrichment.StatusCode != 201 || string(row.Enrichment.Envelope) != childBody || row.Enrichment.Header.Get("X-Cloud-Specs") != "child" || track.calls.Load() != 2 || physical.reads.Load() == 0 || physical.closes.Load() != 1 {
						t.Fatal("accepted child terminal receipt was discarded", row, err, receipt, proof, physical)
					}
				}
				return
			}
			names := make([]string, len(result.Flavors))
			for i, row := range result.Flavors {
				if row == nil || row.Resource == nil || row.Wire == nil || len(row.Resource.Body) != 14 || string(row.Resource.Body["location"]) != `null` {
					t.Fatal("Cloud inventory lost owned source view", row)
				}
				names[i] = string(row.Resource.Body["name"])
			}
			var valueRows []json.RawMessage
			if !reflect.DeepEqual(names, tc.wantNames) || result.Value == nil || json.Unmarshal(result.Value, &valueRows) != nil || len(valueRows) != len(tc.wantNames) {
				t.Fatal("Cloud selection lost page order/duplicates or completed empty array", names, tc.wantNames, string(result.Value))
			}
			for _, row := range result.Inventory {
				if row.Enrichment != nil && (row.Enrichment.StatusCode != 201 || string(row.Enrichment.ExtraSpecs) != `{"cpu":"fresh"}` || string(row.Enrichment.Wire.Body["vendor"]) != `9007199254740993`) {
					t.Fatal("inventory enrichment receipt differs", row.Enrichment)
				}
			}
		})
	}
}

func TestComputeCloudFlavorSelectionOwnedOptionsAndLocation(t *testing.T) {
	const rows = `{"id":"7","name":"blue-2","ram":4},{"id":"8","name":"red","ram":8},{"id":"9","name":"blue-1","ram":4},{"id":"10","name":"blue-2","ram":4}`
	for _, tc := range []struct {
		name, identity, filters, inventoryRows, rawValue string
		wantNames                                        []string
		inventory                                        int
		failure, expression                              bool
	}{
		{"glob preserves inventory order and duplicates", "blue-*", "", rows, "", []string{`"blue-2"`, `"blue-1"`, `"blue-2"`}, 4, false, false},
		{"numeric logical ID participates in Cloud string matching", "17", "", `{"id":17,"name":"number"},{"id":"18","name":"other"}`, "", []string{`"number"`}, 2, false, false},
		{"dictionary is applied locally after all detail rows", "", `{"ram":4}`, rows, "", []string{`"blue-2"`, `"blue-1"`, `"blue-2"`}, 4, false, false},
		{"unknown first predicate is terminal after inventory", "", `{"unknown":true,"ram":0}`, rows, "", nil, 4, true, false},
		{"earlier mismatch leaves unknown predicate unconsumed", "", `{"ram":0,"unknown":true}`, rows, "", []string{}, 4, false, false},
		{"identifier phase empties inventory selection before unknown predicate", "missing", `{"unknown":true}`, rows, "", []string{}, 4, false, false},
		{"expression scalar has no invented Flavor association", "blue-*", "length(@)", rows, `3`, nil, 4, false, true},
		{"expression object preserves arbitrary JSON", "blue-*", "{count:length(@)}", rows, `{"count":3}`, nil, 4, false, true},
		{"expression array preserves values without record association", "blue-*", "[].name", rows, `["blue-2","blue-1","blue-2"]`, nil, 4, false, true},
		{"expression false remains a completed JSON value", "", "`false`", rows, `false`, nil, 4, false, true},
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
				if r.URL.Path != flavorIdentityPath+"/detail" || r.URL.RawQuery != "is_public=None" {
					t.Error("Cloud filters/name changed eager inventory query", r.URL)
				}
				testcloud.JSON(w, 200, flavorIdentityPage(tc.inventoryRows, ""))
			})
			options := []compute.FlavorQueryOption{compute.WithFlavorQueryExtraSpecs(false)}
			if tc.expression {
				options = append(options, compute.WithFlavorQueryExpression(tc.filters))
			} else if tc.filters != "" {
				options = append(options, compute.WithFlavorQueryFilters(json.RawMessage(tc.filters)))
			}
			result, err := compute.New(client, compute.Dependencies{}).SearchFlavors(context.Background(), tc.identity, options...)
			if result == nil || (err != nil) != tc.failure || calls.Load() != 1 || len(result.Inventory) != tc.inventory {
				t.Fatal(result, err, calls.Load())
			}
			if tc.failure {
				if result.Value != nil || result.Flavors != nil {
					t.Fatal("failed Cloud selection published a completed result", result, err)
				}
				return
			}
			if tc.expression {
				if string(result.Value) != tc.rawValue || result.Flavors != nil {
					t.Fatal("expression invented Flavor association or changed arbitrary JSON", result, string(result.Value))
				}
				return
			}
			names := make([]string, len(result.Flavors))
			for i, row := range result.Flavors {
				names[i] = string(row.Resource.Body["name"])
			}
			if !reflect.DeepEqual(names, tc.wantNames) || result.Value == nil || !json.Valid(result.Value) {
				t.Fatal(names, tc.wantNames, string(result.Value))
			}
		})
	}
	t.Run("captured pointer options and callback headers are reusable with one owned location", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.1"
		filters, version, extra := json.RawMessage(`{"ram":4}`), "2.55", false
		captured := compute.WithFlavorQueryOptions(compute.FlavorQueryOpts{Filters: &filters, Microversion: &version, GetExtra: &extra})
		filters[7], version, extra = '8', "2.1", true
		cloudName, project := "owned-cloud", json.RawMessage(`"scope"`)
		var locations, calls, callbacks atomic.Int32
		service := compute.New(client, compute.Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
			locations.Add(1)
			return resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: project}}, nil
		}})
		const first = `{"id":"7","name":"owned-a","ram":4,"location":{"cloud":"wire-cloud"}}`
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestMethod(t, r, http.MethodGet)
			th.TestHeader(t, r, "X-Auth-Token", "owned-live")
			th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
			th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.55")
			th.TestHeader(t, r, "X-Cloud-Flavor", "owned")
			if r.URL.Path != flavorIdentityPath+"/detail" || r.URL.RawQuery != "is_public=None" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, flavorIdentityPage(first+`,{"id":"8","name":"owned-b","ram":4,"location":false}`, ""))
		})
		for repeat := 0; repeat < 2; repeat++ {
			cloudName, project = "owned-cloud", json.RawMessage(`"scope"`)
			facts := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: project}}
			want, err := facts.ForResource(nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			headers := map[string]string{"X-Cloud-Flavor": "owned"}
			result, err := service.SearchFlavors(context.Background(), "owned-*", captured,
				func(c *request.Config[compute.FlavorQueryOpts]) error {
					callbacks.Add(1)
					if locations.Load() != int32(repeat+1) || calls.Load() != int32(repeat) {
						t.Error("Cloud location was not captured once before options", locations.Load(), calls.Load())
					}
					c.Headers = headers
					return nil
				}, func(_ *request.Config[compute.FlavorQueryOpts]) error {
					callbacks.Add(1)
					headers["X-Cloud-Flavor"] = "caller-changed"
					cloudName = "caller-changed"
					project[1] = 'x'
					cloud.Provider.SetToken("owned-live")
					return nil
				})
			if err != nil || result == nil || len(result.Flavors) != 2 || len(result.Inventory) != 2 || calls.Load() != int32(repeat+1) || locations.Load() != int32(repeat+1) || callbacks.Load() != int32(2*(repeat+1)) || client.Microversion != "2.1" {
				t.Fatal("Cloud snapshot/options were reapplied or aliased", result, err, calls.Load(), locations.Load(), callbacks.Load())
			}
			for _, row := range result.Flavors {
				if row.Enrichment != nil || string(row.Resource.Body["location"]) != string(want) || string(row.Resource.Body["ram"]) != `4` {
					t.Fatal(row, string(want))
				}
			}
			if string(result.Inventory[0].Envelope) != first || string(result.Inventory[0].Wire.Body["location"]) != `{"cloud":"wire-cloud"}` || string(result.Inventory[1].Wire.Body["location"]) != `false` {
				t.Fatal("computed location overwrote actual row evidence", result.Inventory)
			}
			result.Flavors[0].Resource.Body["location"][0] = 'x'
			if string(result.Flavors[1].Resource.Body["location"]) != string(want) || !json.Valid(result.Value) || string(result.Inventory[0].Envelope) != first {
				t.Fatal("view mutation changed another row or normalized/physical evidence", result)
			}
		}
	})
	t.Run("nil filters clears presence for AllFlavors", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.55"
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.URL.Path != flavorIdentityPath+"/detail" || r.URL.RawQuery != "is_public=None" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, flavorIdentityPage("", ""))
		})
		result, err := compute.New(client, compute.Dependencies{}).AllFlavors(context.Background(), compute.WithFlavorQueryFilters(json.RawMessage(`null`)), compute.WithFlavorQueryFilters(nil))
		if err != nil || result == nil || string(result.Value) != `[]` || len(result.Flavors) != 0 || len(result.Inventory) != 0 || calls.Load() != 1 {
			t.Fatal(result, err, calls.Load())
		}
	})
	for _, mode := range []string{"location callback changes source", "option cancellation stops later callbacks"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("Cloud flavor option cancellation")
			var calls, locations, later atomic.Int32
			service := compute.New(client, compute.Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
				locations.Add(1)
				if mode == "location callback changes source" {
					client.ResourceBase = cloud.Server.URL + "/changed/"
				}
				return resource.CloudLocation{}, nil
			}})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			result, err := service.SearchFlavors(ctx, "target", func(_ *request.Config[compute.FlavorQueryOpts]) error {
				if mode == "option cancellation stops later callbacks" {
					cancel(cause)
				}
				return nil
			}, func(_ *request.Config[compute.FlavorQueryOpts]) error { later.Add(1); return nil })
			if err == nil || calls.Load() != 0 || locations.Load() != 1 || later.Load() != 0 || result != nil && (result.Value != nil || result.Flavors != nil || len(result.Inventory) != 0) {
				t.Fatal("Cloud preflight guard allowed later callbacks or HTTP", result, err, calls.Load(), locations.Load(), later.Load())
			}
			if mode == "option cancellation stops later callbacks" {
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestComputeCloudFlavorGetFindBindingAndFilterPreflight(t *testing.T) {
	for _, tc := range []struct {
		name, filters, memberBody, query, fallback, rows string
		extra, wantCalls                                 int
		fetch, missing, ambiguous, clear                 bool
	}{
		{"default directly finds then fetches absent specs", "", `{"flavor":{"id":"7","name":"target"}}`, "", "", "", -1, 2, true, false, false, false},
		{"explicit empty microversion owns member and child without discovery", "", `{"flavor":{"id":"7","name":"target"}}`, "", "", "", -1, 2, true, false, false, false},
		{"nil clear discards a prior truthy string filter", "", `{"flavor":{}}`, "", "", "", 0, 1, false, false, false, true},
		{"explicit null keeps member lookup and inline specs", `null`, `{"flavor":{"id":"7","extra_specs":{"cpu":"inline"}}}`, "", "", "", -1, 1, false, false, false, false},
		{"falsey empty object is an unfiltered Find", `{}`, `{"flavor":{}}`, "", "", "", 0, 1, false, false, false, false},
		{"falsey false is an unfiltered Find", `false`, `{"flavor":{}}`, "", "", "", 0, 1, false, false, false, false},
		{"falsey zero is an unfiltered Find", `0`, `{"flavor":{}}`, "", "", "", 0, 1, false, false, false, false},
		{"falsey empty string is an unfiltered Find", `""`, `{"flavor":{}}`, "", "", "", 0, 1, false, false, false, false},
		{"falsey empty array is an unfiltered Find", `[]`, `{"flavor":{}}`, "", "", "", 0, 1, false, false, false, false},
		{"truthy dictionary preserves original member query and seed", `{"ram":4,"min_ram":2}`, `{"flavor":{}}`, "min_ram=2&ram=4", "", "", 0, 1, false, false, false, false},
		{"nonempty inline specs skip default true enrichment", "", `{"flavor":{"id":"7","extra_specs":{"cpu":"inline"}}}`, "", "", "", -1, 1, false, false, false, false},
		{"clean403 reuses Find fallback and enriches only unique match", "", "", "", "403", `{"id":"7","name":"target"},{"id":"8","name":"other"}`, -1, 3, true, false, false, false},
		{"default missing returns nil without Cloud search", "", "", "", "404", "", -1, 2, false, true, false, false},
		{"duplicate fallback is ambiguous before enrichment", "", "", "", "400", `{"id":"7","name":"target"},{"id":"8","name":"target"}`, -1, 2, false, false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			cloudName := "get-cloud"
			var calls, locations, callbacks, specs atomic.Int32
			service := compute.New(client, compute.Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
				locations.Add(1)
				return resource.CloudLocation{Cloud: &cloudName}, nil
			}})
			facts := resource.CloudLocation{Cloud: &cloudName}
			wantLocation, err := facts.ForResource(nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				index := calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				if tc.name == "explicit empty microversion owns member and child without discovery" {
					th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
					th.TestHeaderUnset(t, r, "OpenStack-API-Version")
				} else {
					th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
					th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.55")
				}
				th.TestHeader(t, r, "X-Cloud-Flavor", "get")
				body, readErr := io.ReadAll(r.Body)
				if readErr != nil || len(body) != 0 || callbacks.Load() != 1 || locations.Load() != 1 {
					t.Error("GetFlavor recaptured state or sent a body", r.URL, string(body), readErr, callbacks.Load(), locations.Load())
				}
				if index == 1 {
					th.TestHeader(t, r, "X-Auth-Token", "test-token")
					if r.URL.Path != flavorIdentityPath+"/target" || r.URL.RawQuery != tc.query {
						t.Error("GetFlavor routed through Cloud inventory instead of member Find", r.URL, tc.query)
					}
					cloud.Provider.SetToken("get-flavor-live")
					w.Header().Set("X-Cloud-Get", "member")
					switch tc.fallback {
					case "400":
						testcloud.JSON(w, 400, `{"error":"member bad request"}`)
					case "403":
						testcloud.JSON(w, 403, `{"error":"member denied"}`)
					case "404":
						testcloud.JSON(w, 404, `{"error":"member missing"}`)
					default:
						testcloud.JSON(w, 200, tc.memberBody)
					}
					return
				}
				th.TestHeader(t, r, "X-Auth-Token", "get-flavor-live")
				if tc.fallback != "" && index == 2 {
					if r.URL.Path != flavorIdentityPath+"/detail" || r.URL.RawQuery != "is_public=None" {
						t.Error("GetFlavor fallback changed the owned Find inventory", r.URL)
					}
					w.Header().Set("X-Cloud-Get", "fallback")
					testcloud.JSON(w, 200, flavorIdentityPage(tc.rows, ""))
					return
				}
				specs.Add(1)
				if r.URL.Path != flavorIdentityPath+"/7/os-extra_specs" || r.URL.RawQuery != "" {
					t.Error("GetFlavor looked up a different specs owner", r.URL)
				}
				w.Header().Set("X-Cloud-Get", "specs")
				testcloud.JSON(w, 201, `{"extra_specs":{"cpu":"fresh"}}`)
			})
			options := []compute.FlavorQueryOption{compute.WithFlavorQueryHeader("X-Cloud-Flavor", "get"), func(_ *request.Config[compute.FlavorQueryOpts]) error {
				callbacks.Add(1)
				if locations.Load() != 1 || calls.Load() != 0 {
					t.Error("GetFlavor location/options order differs", locations.Load(), calls.Load())
				}
				cloudName = "caller-changed"
				return nil
			}}
			if tc.filters != "" {
				options = append(options, compute.WithFlavorQueryFilters(json.RawMessage(tc.filters)))
			}
			if tc.clear {
				options = append(options, compute.WithFlavorQueryExpression("length(@)"), compute.WithFlavorQueryFilters(nil))
			}
			if tc.extra >= 0 {
				options = append(options, compute.WithFlavorQueryExtraSpecs(tc.extra == 1))
			}
			if tc.name == "explicit empty microversion owns member and child without discovery" {
				options = append(options, compute.WithFlavorQueryMicroversion(""))
			}
			var record *flavors.FlavorRecord
			record, err = service.GetFlavor(context.Background(), "target", options...)
			wantSpecs := int32(0)
			if tc.fetch {
				wantSpecs = 1
			}
			if calls.Load() != int32(tc.wantCalls) || specs.Load() != wantSpecs || locations.Load() != 1 || callbacks.Load() != 1 || client.Microversion != "2.55" {
				t.Fatal(record, err, calls.Load(), specs.Load(), locations.Load(), callbacks.Load())
			}
			if tc.ambiguous {
				if record != nil || !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal("GetFlavor discarded Find ambiguity", record, err)
				}
				return
			}
			if tc.missing {
				if record != nil || err != nil {
					t.Fatal("missing GetFlavor became strict missing or Cloud search", record, err)
				}
				return
			}
			if err != nil || record == nil || record.Resource == nil || record.Wire == nil || string(record.Resource.Body["location"]) != string(wantLocation) || record.StatusCode != 200 {
				t.Fatal(record, err, string(wantLocation))
			}
			if tc.query != "" && (string(record.Resource.Body["ram"]) != `4` || string(record.Resource.Body["id"]) != `"target"`) {
				t.Fatal("Cloud dictionary did not seed the member Find", record.Resource)
			}
			if tc.fetch {
				child := record.Enrichment
				if child == nil || child.Resource == nil || string(record.Resource.Body["extra_specs"]) != `{"cpu":"fresh"}` || string(child.Resource.Body["location"]) != string(wantLocation) || child.StatusCode != 201 || child.Header.Get("X-Cloud-Get") != "specs" || record.Header.Get("X-Cloud-Get") == "specs" || string(child.Envelope) != `{"extra_specs":{"cpu":"fresh"}}` {
					t.Fatal("GetFlavor lost separate child/location evidence", record)
				}
			} else if record.Enrichment != nil {
				t.Fatal("GetFlavor enriched an inline or disabled result", record)
			}
		})
	}
	for _, tc := range []struct{ name, operation, filters string }{
		{"get truthy expression is not a Cloud search", "get", `"length(@)"`},
		{"get truthy list cannot become keyword filters", "get", `[1]`},
		{"get truthy number cannot become keyword filters", "get", `1`},
		{"get truthy boolean cannot become keyword filters", "get", `true`},
		{"get reserved name_or_id conflicts before HTTP", "get", `{"name_or_id":"other"}`},
		{"get reserved ignore_missing conflicts before HTTP", "get", `{"ignore_missing":false}`},
		{"get reserved get_extra_specs conflicts before HTTP", "get", `{"get_extra_specs":false}`},
		{"get dictionary-valued member query remains unsupported", "get", `{"extra_specs":{"cpu":"nested"}}`},
		{"all explicit null filter presence is unsupported", "all", `null`},
		{"all empty object filter presence is unsupported", "all", `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			service := compute.New(client, compute.Dependencies{})
			option := compute.WithFlavorQueryFilters(json.RawMessage(tc.filters))
			var err error
			if tc.operation == "all" {
				var result *compute.FlavorQueryResult
				result, err = service.AllFlavors(context.Background(), option)
				if result != nil && (result.Value != nil || result.Flavors != nil || len(result.Inventory) != 0) {
					t.Fatal("invalid AllFlavors published a result", result)
				}
			} else {
				var record *flavors.FlavorRecord
				record, err = service.GetFlavor(context.Background(), "target", option)
				if record != nil {
					t.Fatal("invalid GetFlavor published a result", record)
				}
			}
			if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal("Cloud unsupported filter touched HTTP", err, calls.Load())
			}
		})
	}
}
