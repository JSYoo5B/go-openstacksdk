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

	"github.com/JSYoo5B/gophercloudsdk/compute"
	"github.com/JSYoo5B/gophercloudsdk/compute/v2/flavors"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestComputeFindFlavorMemberSeedAliasesAndLiteralRoutes(t *testing.T) {
	for _, tc := range []struct {
		name, body, id, nameValue, description, ram string
		status                                      int
		wire, failure                               bool
	}{
		{"wrapped nullable response overrides seed", `{"flavor":{"id":42,"name":null,"description":null,"ram":"8","vendor":9007199254740993}}`, `42`, `null`, `null`, `8`, 200, true, false},
		{"flat response preserves omitted seed", `{"name":"actual","original_name":"original"}`, `"target"`, `"actual"`, `"seed"`, `4`, 201, true, false},
		{"empty member preserves seed", `{"flavor":{}}`, `"target"`, `null`, `"seed"`, `4`, 200, true, false},
		{"null response ID falls through original name", `{"flavor":{"id":null,"original_name":"7"}}`, `"7"`, `"7"`, `"seed"`, `4`, 200, true, false},
		{"opaque accepted response preserves seed", `opaque response`, `"target"`, `null`, `"seed"`, `4`, 200, false, false},
		{"malformed accepted response preserves seed", `{"flavor":`, `"target"`, `null`, `"seed"`, `4`, 200, false, false},
		{"empty204 accepted response preserves seed", "", `"target"`, `null`, `"seed"`, `4`, 204, false, false},
		{"parsed null root is an error", `null`, "", "", "", "", 200, false, true},
		{"parsed array member is an error", `{"flavor":[]}`, "", "", "", "", 200, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			var calls atomic.Int32
			wantQuery := url.Values{"description": {"seed"}, "ram": {"4"}, "min_ram": {"2"}, "vendor": {"literal"}}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "test-token")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.55")
				th.TestHeader(t, r, "Accept", "application/json")
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 || r.ContentLength != 0 || r.URL.Path != flavorIdentityPath+"/target" || !reflect.DeepEqual(r.URL.Query(), wantQuery) || r.URL.RawQuery != wantQuery.Encode() {
					t.Error("member query was transposed or list default added", r.URL, string(body), err)
				}
				w.Header().Set("X-Flavor-Find", tc.name)
				testcloud.JSON(w, tc.status, tc.body)
			})
			value, err := flavors.New(client).FindFlavor(context.Background(), "target",
				flavors.WithFlavorFindFilters(map[string]any{"description": "seed", "ram": 4, "min_ram": 2}),
				flavors.WithFlavorFindQuery("vendor", "literal"))
			if value == nil || (err != nil) != tc.failure || calls.Load() != 1 || value.StatusCode != tc.status || value.Header.Get("X-Flavor-Find") != tc.name || string(value.Envelope) != tc.body || client.Microversion != "2.55" {
				t.Fatal(value, err, calls.Load())
			}
			if tc.failure {
				var receipt *resource.ResponseError
				if value.Resource != nil || value.Wire != nil || !errors.As(err, &receipt) || receipt.StatusCode != tc.status || string(receipt.Body) != tc.body || receipt.Header.Get("X-Flavor-Find") != tc.name || errors.Is(err, resource.ErrNotFound) {
					t.Fatal("accepted member parse error became fallback/missing", value, err, receipt)
				}
				return
			}
			if value.Resource == nil || len(value.Resource.Body) != 14 || string(value.Resource.Body["id"]) != tc.id || string(value.Resource.Body["name"]) != tc.nameValue || string(value.Resource.Body["description"]) != tc.description || string(value.Resource.Body["ram"]) != tc.ram || string(value.Resource.Body["is_public"]) != `true` || string(value.Resource.Body["location"]) != `null` || (value.Wire != nil) != tc.wire || value.Enrichment != nil {
				t.Fatal("member descriptor seed differs", value)
			}
			if _, present := value.Resource.Body["vendor"]; present {
				t.Fatal("unknown field became declared member attribute", value)
			}
			if value.Wire != nil {
				if _, present := value.Wire.Body["flavor"]; present {
					t.Fatal("member Wire retained wrapper", value.Wire)
				}
				if tc.name == "wrapped nullable response overrides seed" && string(value.Wire.Body["vendor"]) != `9007199254740993` {
					t.Fatal(value.Wire)
				}
				if tc.name == "empty member preserves seed" && len(value.Wire.Body) != 0 {
					t.Fatal("input seed leaked into actual Wire", value.Wire)
				}
			}
			value.Resource.Body["id"][0] = 'x'
			if string(value.Resource.Body["name"]) != tc.nameValue {
				t.Fatal("member logical identity aliases name", value)
			}
			value.Header.Set("X-Flavor-Find", "changed")
			if len(value.Envelope) != 0 {
				value.Envelope[0] = 'x'
			}
			if value.Wire != nil {
				if value.Wire.Header.Get("X-Flavor-Find") != tc.name {
					t.Fatal("member receipt header aliases Wire", value)
				}
				for _, raw := range value.Wire.Body {
					if !json.Valid(raw) {
						t.Fatal("member view/envelope aliases Wire", value)
					}
				}
			}
		})
	}
	for index, tc := range []struct {
		identity, selected string
		empty              bool
	}{
		{"flavor with space", "2.100", false},
		{"flavor/+?#%한글", "2.100", true},
		{"plain", "latest", false},
	} {
		t.Run("literal member route "+strconv.Itoa(index), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = tc.selected
			var options []flavors.FlavorFindOption
			if tc.empty {
				options = append(options, flavors.WithFlavorFindMicroversion(""))
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				if r.URL.EscapedPath() != flavorIdentityPath+"/"+url.PathEscape(tc.identity) || r.URL.RawQuery != "" || r.ContentLength != 0 {
					t.Error("find literal did not remain one member", r.URL)
				}
				if tc.empty {
					th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
					th.TestHeaderUnset(t, r, "OpenStack-API-Version")
				} else {
					th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", tc.selected)
					th.TestHeader(t, r, "OpenStack-API-Version", "compute "+tc.selected)
				}
				testcloud.JSON(w, 200, `{"flavor":{}}`)
			})
			value, err := flavors.New(client).FindFlavor(context.Background(), tc.identity, options...)
			seed, _ := json.Marshal(tc.identity)
			if err != nil || value == nil || value.Resource == nil || string(value.Resource.Body["id"]) != string(seed) || calls.Load() != 1 || client.Microversion != tc.selected {
				t.Fatal(value, err, calls.Load(), client)
			}
		})
	}
	t.Run("semantic id conflicts before discovery or member", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			t.Error("conflicting seed ID touched HTTP", r.URL)
			w.WriteHeader(500)
		})
		value, err := flavors.New(client).FindFlavor(context.Background(), "target", flavors.WithFlavorFindFilter("id", "other"))
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal(value, err, calls.Load())
		}
	})
}

func TestComputeFindFlavorFallbackTerminalErrorsAndConnectedLocation(t *testing.T) {
	for _, mode := range []string{"400", "403", "404", "default missing", "strict missing", "duplicate next page", "late403 after match", "numeric ID does not match string", "original-name alias matches", "unique waits for full list", "unique enriches after full list", "409 no fallback", "500 no fallback", "auto2.61 shared with fallback", "explicit empty shared with fallback"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Endpoint = cloud.Server.URL + "/reverse/nova/v2.1/project/"
			client.Microversion = "2.55"
			identity, status, selected := "target", 404, "2.55"
			if mode == "400" {
				status = 400
			} else if mode == "403" {
				status = 403
			} else if mode == "409 no fallback" {
				status = 409
			} else if mode == "500 no fallback" {
				status = 500
			} else if mode == "numeric ID does not match string" {
				identity = "7"
			} else if mode == "auto2.61 shared with fallback" {
				client.Microversion, selected = "", "2.61"
			} else if mode == "explicit empty shared with fallback" {
				selected = ""
			}
			options := []flavors.FlavorFindOption{flavors.WithFlavorFindFilter("min_ram", 2), flavors.WithFlavorFindQuery("vendor", "literal")}
			if mode == "strict missing" {
				options = append(options, flavors.WithFlavorFindIgnoreMissing(false))
			}
			if mode == "explicit empty shared with fallback" {
				options = append(options, flavors.WithFlavorFindMicroversion(""))
			}
			if mode == "unique enriches after full list" || mode == "duplicate next page" || mode == "late403 after match" {
				options = append(options, flavors.WithFlavorFindExtraSpecs(true))
			}
			var gets, lists, specs, discoveries, callbacks atomic.Int32
			options = append(options, func(_ *request.Config[flavors.FlavorFindOpts]) error { callbacks.Add(1); return nil })
			const rejected = `{"error":"original flavor member rejection"}`
			const row = `{"id":"7","name":"target"}`
			linked := mode == "duplicate next page" || mode == "late403 after match" || mode == "unique waits for full list" || mode == "unique enriches after full list"
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				th.TestMethod(t, r, http.MethodGet)
				payload, err := io.ReadAll(r.Body)
				if err != nil || len(payload) != 0 || r.ContentLength != 0 || callbacks.Load() != 1 {
					t.Error(r.URL, string(payload), err, callbacks.Load())
				}
				if r.URL.Path == computeConsoleDiscoveryPath {
					discoveries.Add(1)
					th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
					th.TestHeaderUnset(t, r, "OpenStack-API-Version")
					testcloud.JSON(w, 200, `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.110"}}`)
					return
				}
				if selected == "" {
					th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
					th.TestHeaderUnset(t, r, "OpenStack-API-Version")
				} else {
					th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", selected)
					th.TestHeader(t, r, "OpenStack-API-Version", "compute "+selected)
				}
				if r.URL.Path == flavorIdentityPath+"/"+identity {
					gets.Add(1)
					th.TestHeader(t, r, "X-Auth-Token", "test-token")
					if !reflect.DeepEqual(r.URL.Query(), url.Values{"min_ram": {"2"}, "vendor": {"literal"}}) {
						t.Error("member gained list default/aliases/name hint", r.URL)
					}
					cloud.Provider.SetToken("fallback-flavor-live")
					w.Header().Set("X-Flavor-Find", "member original")
					testcloud.JSON(w, status, rejected)
					return
				}
				th.TestHeader(t, r, "X-Auth-Token", "fallback-flavor-live")
				if r.URL.Path == flavorExtraSpecsFetchPath {
					specs.Add(1)
					if mode != "unique enriches after full list" || lists.Load() != 2 || r.URL.RawQuery != "" {
						t.Error("unresolved/missing/duplicate flavor enriched", r.URL, lists.Load())
					}
					w.Header().Set("X-Spec-Proof", "find child")
					testcloud.JSON(w, 201, `{"extra_specs":{"cpu":"fresh"}}`)
					return
				}
				if r.URL.Path != flavorIdentityPath+"/detail" {
					t.Error("find performed unrelated lookup", r.URL)
					w.WriteHeader(500)
					return
				}
				page := lists.Add(1)
				want := url.Values{"minRam": {"2"}, "vendor": {"literal"}, "is_public": {"None"}}
				if page == 2 {
					want.Set("marker", "next")
				}
				if !reflect.DeepEqual(r.URL.Query(), want) || page > 2 {
					t.Error("fallback query/alias/default changed", r.URL, want, page)
				}
				w.Header().Set("X-Flavor-Find", "list actual")
				if page == 2 {
					if mode == "late403 after match" {
						w.Header().Set("X-Flavor-Find", "late original")
						testcloud.JSON(w, 403, `{"error":"late flavor forbidden"}`)
					} else if mode == "duplicate next page" {
						testcloud.JSON(w, 200, flavorIdentityPage(`{"id":"target"}`, ""))
					} else {
						testcloud.JSON(w, 200, flavorIdentityPage(`{"id":"later","name":"miss"}`, ""))
					}
					return
				}
				rows := row
				switch mode {
				case "default missing", "strict missing":
					rows = `{"id":"miss","name":"miss"}`
				case "numeric ID does not match string":
					rows = `{"id":7}`
				case "original-name alias matches":
					rows = `{"id":null,"original_name":"target"}`
				}
				next := ""
				if linked {
					next = cloud.Server.URL + flavorIdentityPath + "/detail?is_public=None&minRam=2&vendor=literal&marker=next"
				}
				testcloud.JSON(w, 200, flavorIdentityPage(rows, next))
			})
			originalVersion := client.Microversion
			value, err := flavors.New(client).FindFlavor(context.Background(), identity, options...)
			wantLists, wantSpecs, wantDiscovery := int32(1), int32(0), int32(0)
			if linked {
				wantLists = 2
			}
			if status == 409 || status == 500 {
				wantLists = 0
			}
			if mode == "unique enriches after full list" {
				wantSpecs = 1
			}
			if mode == "auto2.61 shared with fallback" {
				wantDiscovery = 1
			}
			if gets.Load() != 1 || lists.Load() != wantLists || specs.Load() != wantSpecs || discoveries.Load() != wantDiscovery || callbacks.Load() != 1 || client.Microversion != originalVersion {
				t.Fatal(value, err, gets.Load(), lists.Load(), specs.Load(), discoveries.Load(), callbacks.Load())
			}
			if status == 409 || status == 500 {
				var proof gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.As(err, &proof) || proof.Actual != status || len(proof.Expected) != 200 || proof.Method != http.MethodGet || proof.URL != cloud.Server.URL+flavorIdentityPath+"/target?min_ram=2&vendor=literal" || string(proof.Body) != rejected || proof.ResponseHeader.Get("X-Flavor-Find") != "member original" {
					t.Fatal("unrelated HTTP became fallback/missing", value, err, proof)
				}
				return
			}
			switch mode {
			case "default missing", "numeric ID does not match string":
				if value != nil || err != nil {
					t.Fatal(value, err)
				}
			case "strict missing":
				var proof gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.Is(err, resource.ErrNotFound) || errors.As(err, &proof) {
					t.Fatal("logical absence retained suppressed HTTP", value, err, proof)
				}
			case "duplicate next page":
				if value != nil || !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(value, err)
				}
			case "late403 after match":
				var proof gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.As(err, &proof) || proof.Actual != 403 || string(proof.Body) != `{"error":"late flavor forbidden"}` || proof.ResponseHeader.Get("X-Flavor-Find") != "late original" {
					t.Fatal(value, err, proof)
				}
			default:
				wantID := `"7"`
				if mode == "original-name alias matches" {
					wantID = `"target"`
				}
				if err != nil || value == nil || value.Resource == nil || value.Wire == nil || string(value.Resource.Body["id"]) != wantID || string(value.Resource.Body["name"]) != `"target"` || value.Header.Get("X-Flavor-Find") != "list actual" || value.StatusCode != 200 {
					t.Fatal(value, err)
				}
				if mode == "unique enriches after full list" && (value.Enrichment == nil || value.Enrichment.StatusCode != 201 || string(value.Resource.Body["extra_specs"]) != `{"cpu":"fresh"}` || string(value.Envelope) != row || string(value.Wire.Body["id"]) != `"7"`) {
					t.Fatal("unique enrichment changed list evidence", value)
				}
			}
		})
	}
	for _, mode := range []string{"accepted member read404", "accepted member Close404", "source after accepted member"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			const body = `{"flavor":{"id":"7"}}`
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Method: http.MethodGet, URL: "nested-flavor", Body: []byte("physical flavor404")}
			var track *payloadContractTracking
			if mode == "accepted member read404" {
				track = payloadContractTrack(cloud, nested, nil)
			} else if mode == "accepted member Close404" {
				track = payloadContractTrack(cloud, nil, nested)
			} else {
				base := cloud.Provider.HTTPClient.Transport
				if base == nil {
					base = http.DefaultTransport
				}
				cloud.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					response, err := base.RoundTrip(r)
					if err == nil {
						client.Microversion = "2.99"
					}
					return response, err
				})
			}
			var calls, retries atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != flavorIdentityPath+"/target" || r.URL.RawQuery != "" {
					t.Error("accepted member fault performed fallback", r.URL)
				}
				w.Header().Set("X-Flavor-Find", mode)
				testcloud.JSON(w, 200, body)
			})
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			value, err := flavors.New(client).FindFlavor(context.Background(), "target")
			var receipt *resource.ResponseError
			if value == nil || value.Resource != nil || value.Wire != nil || !errors.As(err, &receipt) || receipt.StatusCode != 200 || string(receipt.Body) != body || receipt.Header.Get("X-Flavor-Find") != mode || string(value.Envelope) != body || value.StatusCode != 200 || calls.Load() != 1 || retries.Load() != 0 || errors.Is(err, resource.ErrNotFound) {
				t.Fatal("accepted member physical cause became fallback/missing", value, err, receipt, calls.Load(), retries.Load())
			}
			if track != nil {
				var proof gophercloud.ErrUnexpectedResponseCode
				physical := track.last(t)
				if !errors.As(err, &proof) || proof.Actual != 404 || proof.URL != "nested-flavor" || string(proof.Body) != "physical flavor404" || track.calls.Load() != 1 || physical.closes.Load() != 1 || physical.reads.Load() == 0 {
					t.Fatal(err, proof, track.calls.Load(), physical)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
	t.Run("service list and find capture independent location before options", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.55"
		cloudName, region := "recorded-cloud", "recorded-region"
		project := json.RawMessage(`"scope"`)
		location := resource.CloudLocation{Cloud: &cloudName, RegionName: &region, Project: resource.CloudProject{ID: project}}
		want, err := location.ForResource(nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		var locations, calls, callbacks atomic.Int32
		service := compute.New(client, compute.Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations.Add(1); return location, nil }})
		const row = `{"id":"7","location":{"cloud":"wire-cloud"},"project_id":"wire-project","zone":"wire-zone"}`
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestMethod(t, r, http.MethodGet)
			if r.URL.Path == flavorIdentityPath+"/detail" {
				if r.URL.RawQuery != "is_public=None" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, flavorIdentityPage(row+`,{"id":"8","location":false}`, ""))
			} else if r.URL.Path == flavorIdentityPath+"/target" && r.URL.RawQuery == "" {
				testcloud.JSON(w, 200, `{"flavor":{"id":"target","location":false}}`)
			} else {
				t.Error("service location invented another lookup", r.URL)
				w.WriteHeader(500)
			}
		})
		sequence := service.ListFlavors(context.Background(), func(_ *request.Config[flavors.FlavorListOpts]) error {
			callbacks.Add(1)
			if locations.Load() != 1 || calls.Load() != 0 {
				t.Error("list location was not captured before options", locations.Load(), calls.Load())
			}
			cloudName, region = "caller-changed", "caller-changed"
			project[1] = 'x'
			return nil
		})
		if locations.Load() != 0 || calls.Load() != 0 || callbacks.Load() != 0 {
			t.Fatal("service iterator eagerly captured location")
		}
		var records []*flavors.FlavorRecord
		for value, err := range sequence {
			if err != nil || value == nil || value.Resource == nil || string(value.Resource.Body["location"]) != string(want) {
				t.Fatal(value, err, string(want))
			}
			records = append(records, value)
		}
		if len(records) != 2 || calls.Load() != 1 || locations.Load() != 1 || callbacks.Load() != 1 || string(records[0].Envelope) != row || string(records[0].Wire.Body["location"]) != `{"cloud":"wire-cloud"}` || string(records[0].Wire.Body["project_id"]) != `"wire-project"` || string(records[0].Wire.Body["zone"]) != `"wire-zone"` {
			t.Fatal(records, calls.Load(), locations.Load(), callbacks.Load())
		}
		records[0].Resource.Body["location"][0] = 'x'
		if string(records[1].Resource.Body["location"]) != string(want) {
			t.Fatal("location bytes alias across rows", records)
		}
		cloudName, region = "next-cloud", "next-region"
		project = json.RawMessage(`"next-scope"`)
		location.Project.ID = project
		wantFind, err := location.ForResource(nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		value, err := service.FindFlavor(context.Background(), "target", func(_ *request.Config[flavors.FlavorFindOpts]) error {
			callbacks.Add(1)
			if locations.Load() != 2 || calls.Load() != 1 {
				t.Error("find location was not captured before options", locations.Load(), calls.Load())
			}
			cloudName = "find-caller-changed"
			return nil
		})
		if err != nil || value == nil || value.Resource == nil || string(value.Resource.Body["location"]) != string(wantFind) || string(value.Wire.Body["location"]) != `false` || calls.Load() != 2 || locations.Load() != 2 || callbacks.Load() != 2 || string(records[1].Resource.Body["location"]) != string(want) {
			t.Fatal(value, err, calls.Load(), locations.Load(), callbacks.Load())
		}
	})
	for _, mode := range []string{"location callback changes source", "option cancellation stops later callback"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var calls, locations, later atomic.Int32
			service := compute.New(client, compute.Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
				locations.Add(1)
				if mode == "location callback changes source" {
					client.ResourceBase = cloud.Server.URL + "/changed/"
				}
				return resource.CloudLocation{}, nil
			}})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				t.Error("stopped service flavor operation touched HTTP", r.URL)
				w.WriteHeader(500)
			})
			value, err := service.FindFlavor(ctx, "target", func(_ *request.Config[flavors.FlavorFindOpts]) error {
				if mode == "option cancellation stops later callback" {
					cancel()
				}
				return nil
			}, func(_ *request.Config[flavors.FlavorFindOpts]) error { later.Add(1); return nil })
			want := error(resource.ErrInvalidOption)
			if mode == "option cancellation stops later callback" {
				want = context.Canceled
			}
			if value != nil || !errors.Is(err, want) || calls.Load() != 0 || locations.Load() != 1 || later.Load() != 0 {
				t.Fatal(value, err, calls.Load(), locations.Load(), later.Load())
			}
		})
	}
}
