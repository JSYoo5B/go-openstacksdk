package compute_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	computev2 "github.com/JSYoo5B/go-openstacksdk/compute/v2"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const flavorRAMListPath = "/nova/v2.1/flavors/detail"
const flavorRAMDefaultRows = `{"flavors":[{"id":"big","name":"m1.big","ram":8},{"id":"first","name":"m1.first","ram":4},{"id":"second","name":"m1.second","ram":4},{"id":"small","name":"m1.small","ram":2}]}`

// Reuse the existing public testhelper-backed Nova transport fixture and body
// observers; RAM selection needs neither a custom mock-server framework nor a
// second flavor reader.
func flavorRAMFixture(t *testing.T, rows string) (*compute.Service, *gophercloud.ServiceClient, *[]string) {
	t.Helper()
	s, c, calls := serverGroupCloudFixture(t, func(r *http.Request) *http.Response {
		if r.Method != http.MethodGet {
			t.Error(r.Method, r.URL)
		}
		if r.URL.Path == flavorRAMListPath {
			if r.URL.Query().Get("is_public") != "None" || len(r.URL.Query()) != 1 {
				t.Error(r.URL)
			}
			return backupServerWire(200, rows, "list")
		}
		if !strings.HasSuffix(r.URL.Path, "/os-extra_specs") || r.URL.RawQuery != "" {
			t.Error(r.URL)
		}
		return backupServerWire(201, `{"extra_specs":{"actual":"yes"},"vendor":9007199254740993}`, "extra")
	})
	c.Microversion = "2.55"
	return s, c, calls
}
func flavorRAMID(t *testing.T, result *compute.FlavorRAMResult) string {
	t.Helper()
	if result == nil || result.Flavor == nil || result.Flavor.Resource == nil {
		t.Fatal(result)
	}
	var id string
	if e := json.Unmarshal(result.Flavor.Resource.Body["id"], &id); e != nil {
		t.Fatal(e)
	}
	return id
}
func TestCloudFlavorByRAMStableSortAllPagesDefaultEnrichmentAndActualProof(t *testing.T) {
	s, c, calls := serverGroupCloudFixture(t, func(r *http.Request) *http.Response {
		if r.URL.Path == flavorRAMListPath {
			if r.URL.Query().Get("is_public") != "None" || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.55" {
				t.Error(r.URL, r.Header)
			}
			if r.URL.Query().Get("marker") == "second" {
				return backupServerWire(202, `{"flavors":[{"id":"tie","name":"other","ram":4},{"id":"small","name":"small","ram":2}]}`, "page2")
			}
			return backupServerWire(203, `{"flavors":[{"id":"big","name":"large","ram":8},{"id":"first","name":"first","ram":4}],"flavors_links":[{"rel":"next","href":"?is_public=None&marker=second"}]}`, "page1")
		}
		return backupServerWire(201, `{"extra_specs":{"actual":"yes"},"vendor":9007199254740993}`, "extra")
	})
	c.Microversion = "2.55"
	result, e := s.GetFlavorByRAM(context.Background(), 4)
	want := []string{"GET " + flavorRAMListPath + "?is_public=None", "GET /nova/v2.1/flavors/big/os-extra_specs?", "GET /nova/v2.1/flavors/first/os-extra_specs?", "GET " + flavorRAMListPath + "?is_public=None&marker=second", "GET /nova/v2.1/flavors/tie/os-extra_specs?", "GET /nova/v2.1/flavors/small/os-extra_specs?"}
	if e != nil || flavorRAMID(t, result) != "first" || len(result.Inventory) != 4 || !reflect.DeepEqual(*calls, want) {
		t.Fatal(result, e, *calls, want)
	}
	chosen := result.Flavor
	if chosen.StatusCode != 203 || chosen.Header.Get("X-Backup-Stage") != "page1" || string(chosen.Envelope) != `{"id":"first","name":"first","ram":4}` || string(chosen.Wire.Body["ram"]) != "4" || chosen.Wire.Body["location"] != nil || string(chosen.Resource.Body["location"]) != "null" || chosen.Enrichment == nil || chosen.Enrichment.StatusCode != 201 || string(chosen.Enrichment.Wire.Body["vendor"]) != "9007199254740993" {
		t.Fatal(chosen)
	}
	chosen.Resource.Body["id"] = json.RawMessage(`"changed"`)
	chosen.Wire.Body["ram"] = json.RawMessage("999")
	chosen.Enrichment.ExtraSpecs[2] = 'X'
	chosen.Header.Set("X-Backup-Stage", "changed")
	if string(result.Inventory[1].Resource.Body["id"]) != `"first"` || string(result.Inventory[1].Wire.Body["ram"]) != "4" || string(result.Inventory[1].Enrichment.ExtraSpecs) != `{"actual":"yes"}` || result.Inventory[1].Header.Get("X-Backup-Stage") != "page1" {
		t.Fatal(result.Inventory[1])
	}
}
func TestCloudFlavorByRAMThresholdJSONNumbersDescriptorsAndNoMatchOrder(t *testing.T) {
	for _, tc := range []struct {
		raw, want        string
		invalid, missing bool
	}{
		{"0", "small", false, false}, {"-1", "small", false, false}, {"true", "small", false, false}, {"false", "small", false, false}, {"4", "first", false, false}, {"4.5", "big", false, false}, {"9", "", false, true}, {"100000000000000000000000", "", false, true}, {"null", "", true, false}, {`"4"`, "", true, false}, {"[]", "", true, false}, {"{}", "", true, false}, {"1e-400", "small", false, false}, {"1e400", "", false, true},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			s, _, calls := flavorRAMFixture(t, flavorRAMDefaultRows)
			r, e := s.GetFlavorByRAM(context.Background(), json.RawMessage(tc.raw), compute.WithFlavorRAMGetExtra(false))
			if r == nil || len(r.Inventory) != 4 || len(*calls) != 1 {
				t.Fatal(r, e, *calls)
			}
			if tc.invalid {
				if !errors.Is(e, resource.ErrInvalidOption) || r.Flavor != nil {
					t.Fatal(r, e)
				}
				return
			}
			if tc.missing {
				if !errors.Is(e, compute.ErrNoFlavorByRAM) || r.Flavor != nil {
					t.Fatal(r, e)
				}
				var proof *resource.ResponseError
				if !errors.As(e, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Backup-Stage") != "list" {
					t.Fatal(e, proof)
				}
				return
			}
			if e != nil || flavorRAMID(t, r) != tc.want {
				t.Fatal(r, e, tc.want)
			}
		})
	}
	for _, tc := range []struct {
		rows, threshold, want string
		invalid, missing      bool
	}{
		{`{"flavors":[{"id":"bad","ram":null},{"id":"ok","ram":4}]}`, "4", "", true, false},
		{`{"flavors":[{"id":"signed","ram":"-1"},{"id":"ok","ram":2}]}`, "1", "ok", false, false},
		{`{"flavors":[{"id":"bool","ram":true},{"id":"int","ram":1}]}`, "1", "bool", false, false},
		{`{"flavors":[{"id":"float","ram":1e23},{"id":"int","ram":100000000000000000000000}]}`, "100000000000000000000000", "int", false, false},
		{`{"flavors":[{"id":"float","ram":1e23},{"id":"int","ram":100000000000000000000000}]}`, "1e23", "float", false, false},
		{`{"flavors":[]}`, "null", "", false, true},
	} {
		t.Run(tc.threshold+tc.rows, func(t *testing.T) {
			s, _, calls := flavorRAMFixture(t, tc.rows)
			r, e := s.GetFlavorByRAM(context.Background(), json.RawMessage(tc.threshold), compute.WithFlavorRAMGetExtra(false), compute.WithFlavorRAMInclude(false))
			if r == nil || len(*calls) != 1 {
				t.Fatal(r, e, *calls)
			}
			if tc.invalid {
				if !errors.Is(e, resource.ErrInvalidOption) {
					t.Fatal(r, e)
				}
				return
			}
			if tc.missing {
				if !errors.Is(e, compute.ErrNoFlavorByRAM) || errors.Is(e, resource.ErrInvalidOption) {
					t.Fatal(r, e)
				}
				return
			}
			if e != nil || flavorRAMID(t, r) != tc.want {
				t.Fatal(r, e)
			}
		})
	}
}
func TestCloudFlavorByRAMIncludeSubstringTruthinessContainersAndShortCircuit(t *testing.T) {
	for _, tc := range []struct {
		name, include, want string
		invalid, missing    bool
	}{
		{`"m1.second"`, `"second"`, "row", false, false}, {`"m1.second"`, `"Second"`, "", false, true}, {`"m1.second"`, `"s*"`, "", false, true},
		{`null`, `null`, "row", false, false}, {`null`, `false`, "row", false, false}, {`null`, `0`, "row", false, false}, {`null`, `[]`, "row", false, false}, {`null`, `{}`, "row", false, false}, {`null`, `""`, "row", false, false}, {`null`, `"x"`, "", true, false}, {`7`, `"x"`, "", true, false},
		{`["x","y"]`, `"x"`, "row", false, false}, {`[true]`, `1`, "row", false, false}, {`{"x":1}`, `"x"`, "row", false, false}, {`{"x":1}`, `true`, "", false, true}, {`{"x":1}`, `["x"]`, "", true, false},
	} {
		t.Run(tc.name+tc.include, func(t *testing.T) {
			s, _, calls := flavorRAMFixture(t, `{"flavors":[{"id":"row","ram":4,"name":`+tc.name+`}]}`)
			r, e := s.GetFlavorByRAM(context.Background(), 4, compute.WithFlavorRAMGetExtra(false), compute.WithFlavorRAMInclude(json.RawMessage(tc.include)))
			if r == nil || len(r.Inventory) != 1 || len(*calls) != 1 {
				t.Fatal(r, e, *calls)
			}
			if tc.invalid {
				if !errors.Is(e, resource.ErrInvalidOption) {
					t.Fatal(r, e)
				}
				return
			}
			if tc.missing {
				if !errors.Is(e, compute.ErrNoFlavorByRAM) {
					t.Fatal(r, e)
				}
				return
			}
			if e != nil || flavorRAMID(t, r) != tc.want {
				t.Fatal(r, e)
			}
		})
	}
	s, _, _ := flavorRAMFixture(t, `{"flavors":[{"id":"bad-name","ram":0,"name":null},{"id":"good","ram":4,"name":"match"}]}`)
	r, e := s.GetFlavorByRAM(context.Background(), 3, compute.WithFlavorRAMGetExtra(false), compute.WithFlavorRAMInclude("match"))
	if e != nil || flavorRAMID(t, r) != "good" {
		t.Fatal(r, e)
	}
}
func TestCloudFlavorByRAMExtraTruthinessInlineSkipAndLateFailureBeforeSelection(t *testing.T) {
	for _, raw := range []string{"false", "null", "0", `""`, "[]", "{}", "true", `"yes"`} {
		t.Run(raw, func(t *testing.T) {
			s, _, calls := flavorRAMFixture(t, flavorRAMDefaultRows)
			r, e := s.GetFlavorByRAM(context.Background(), 4, compute.WithFlavorRAMGetExtraValue(json.RawMessage(raw)))
			n := 1
			if raw == "true" || raw == `"yes"` {
				n = 5
			}
			if e != nil || flavorRAMID(t, r) != "first" || len(*calls) != n {
				t.Fatal(r, e, *calls)
			}
		})
	}
	s, _, calls := flavorRAMFixture(t, `{"flavors":[{"id":"inline","ram":4,"extra_specs":{"already":"there"}}]}`)
	r, e := s.GetFlavorByRAM(context.Background(), 4)
	if e != nil || flavorRAMID(t, r) != "inline" || len(*calls) != 1 || r.Flavor.Enrichment != nil {
		t.Fatal(r, e, *calls)
	}
	for _, lane := range []string{"extra", "page"} {
		t.Run(lane, func(t *testing.T) {
			service, c, observed := serverGroupCloudFixture(t, func(req *http.Request) *http.Response {
				if req.URL.Path == flavorRAMListPath {
					if req.URL.Query().Get("marker") == "late" {
						return backupServerWire(503, `{"error":"late page"}`, "late-page")
					}
					if lane == "page" {
						return backupServerWire(200, `{"flavors":[{"id":"match","ram":4,"extra_specs":{"inline":true}}],"flavors_links":[{"rel":"next","href":"?marker=late"}]}`, "head")
					}
					return backupServerWire(200, `{"flavors":[{"id":"match","ram":4},{"id":"late","ram":9}]}`, "head")
				}
				if strings.Contains(req.URL.Path, "/late/") {
					return backupServerWire(503, `{"error":"late extra"}`, "late-extra")
				}
				return backupServerWire(200, `{"extra_specs":{"actual":"yes"}}`, "extra")
			})
			c.Microversion = "2.55"
			got, err := service.GetFlavorByRAM(context.Background(), 4)
			var native gophercloud.ErrUnexpectedResponseCode
			if got == nil || got.Flavor != nil || len(got.Inventory) == 0 || !errors.As(err, &native) || native.Actual != 503 || !strings.Contains(string(native.Body), "late") || len(*observed) < 2 {
				t.Fatal(got, err, native, *observed)
			}
		})
	}
}
func TestCloudFlavorByRAMOwnsOptionsAndInheritedReadTransportOptions(t *testing.T) {
	s, c, calls := flavorRAMFixture(t, flavorRAMDefaultRows)
	raw := json.RawMessage(`"first"`)
	include := compute.WithFlavorRAMInclude(raw)
	raw[1] = 'X'
	query := []compute.FlavorQueryOption{compute.WithFlavorQueryExtraSpecs(false), compute.WithFlavorQueryHeader("X-Owned", "before"), compute.WithFlavorQueryMicroversion("2.55")}
	queryOption := compute.WithFlavorRAMQueryOptions(query...)
	query[0] = compute.WithFlavorQueryExtraSpecs(true)
	var retained *compute.FlavorRAMOpts
	original := c.ProviderClient.HTTPClient.Transport
	c.ProviderClient.HTTPClient.Transport = testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("X-Owned") != "before" {
			t.Error(req.Header)
		}
		return original.RoundTrip(req)
	})
	r, e := s.GetFlavorByRAM(context.Background(), 4, queryOption, include, func(v *compute.FlavorRAMOpts) error { retained = v; return nil }, func(*compute.FlavorRAMOpts) error {
		(*retained.Include)[1] = 'X'
		retained.QueryOptions[0] = compute.WithFlavorQueryExtraSpecs(true)
		return nil
	})
	if e != nil || flavorRAMID(t, r) != "first" || len(*calls) != 1 {
		t.Fatal(r, e, *calls)
	}
	c.ProviderClient.HTTPClient.Transport = original
	cleared := json.RawMessage(`"second"`)
	opts := compute.WithFlavorRAMOpts(compute.FlavorRAMOpts{Include: &cleared})
	cleared[1] = 'X'
	r, e = s.GetFlavorByRAM(context.Background(), 4, opts, compute.WithFlavorRAMGetExtra(false))
	if e != nil || flavorRAMID(t, r) != "second" {
		t.Fatal(r, e)
	}
}

type flavorRAMMarshalFunc func() ([]byte, error)

func (f flavorRAMMarshalFunc) MarshalJSON() ([]byte, error) { return f() }
func TestCloudFlavorByRAMPreflightSelectedBindingCancellationAndUnsupportedOptions(t *testing.T) {
	for _, lane := range []string{"nil-option", "nil-context", "bad-json", "filters", "binding", "cancel", "marshal-cancel"} {
		t.Run(lane, func(t *testing.T) {
			s, c, calls := flavorRAMFixture(t, flavorRAMDefaultRows)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("RAM custom cancel")
			var active context.Context = ctx
			var ram any = 4
			opts := []compute.FlavorRAMOption{}
			switch lane {
			case "nil-option":
				opts = append(opts, nil)
			case "nil-context":
				active = nil
			case "bad-json":
				opts = append(opts, compute.WithFlavorRAMInclude(json.RawMessage("[")))
			case "filters":
				opts = append(opts, compute.WithFlavorRAMQueryOptions(compute.WithFlavorQueryFilters(json.RawMessage("{}"))))
			case "binding":
				opts = append(opts, func(*compute.FlavorRAMOpts) error { s.API = computev2.New(c); return nil })
			case "cancel":
				opts = append(opts, func(*compute.FlavorRAMOpts) error { cancel(cause); return nil })
			case "marshal-cancel":
				ram = flavorRAMMarshalFunc(func() ([]byte, error) { cancel(cause); return []byte("4"), nil })
			}
			r, e := s.GetFlavorByRAM(active, ram, opts...)
			if r != nil || e == nil || len(*calls) != 0 {
				t.Fatal(r, e, *calls)
			}
			if strings.Contains(lane, "cancel") {
				if !errors.Is(e, context.Canceled) || !errors.Is(e, cause) {
					t.Fatal(e)
				}
			} else if !errors.Is(e, resource.ErrInvalidOption) {
				t.Fatal(e)
			}
		})
	}
}
func TestCloudFlavorByRAMPhysicalFaultsNative401AndCleanRetry(t *testing.T) {
	for _, lane := range []string{"list-read-close", "extra-read-close", "cancel", "restored-source", "native401", "clean-retry"} {
		t.Run(lane, func(t *testing.T) {
			readErr, closeErr, cancelErr := errors.New("RAM read"), errors.New("RAM close"), errors.New("RAM canceled")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			attempts := 0
			body := &testcloud.Body{Data: `{"flavors":[{"id":"first","ram":4}]}`}
			var c *gophercloud.ServiceClient
			s, client, calls := serverGroupCloudFixture(t, func(req *http.Request) *http.Response {
				attempts++
				if lane == "native401" {
					return backupServerWire(401, `{"error":"actual auth"}`, "auth")
				}
				if lane == "clean-retry" && attempts == 1 {
					return backupServerWire(503, `{"error":"retry"}`, "retry")
				}
				if lane == "extra-read-close" && req.URL.Path == flavorRAMListPath {
					return backupServerWire(200, `{"flavors":[{"id":"first","ram":4}]}`, "list")
				}
				if lane == "clean-retry" {
					return backupServerWire(200, `{"flavors":[{"id":"first","ram":4,"extra_specs":{"inline":true}}]}`, "complete")
				}
				return testcloud.StreamResponse(200, body, http.Header{"X-Actual": {"physical"}})
			})
			c = client
			c.Microversion = "2.55"
			original := c.ResourceBase
			switch lane {
			case "list-read-close":
				body.ReadErr, body.CloseErr = readErr, closeErr
			case "extra-read-close":
				body.Data = `{"extra_specs":{"actual":"yes"}}`
				body.ReadErr, body.CloseErr = readErr, closeErr
			case "cancel":
				body.OnClose = func() { cancel(cancelErr) }
			case "restored-source":
				body.OnRead = func() { c.ResourceBase = original + "changed/" }
				body.OnClose = func() { c.ResourceBase = original }
			}
			retries := 0
			if lane == "clean-retry" {
				c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					if retries > 1 {
						return fmt.Errorf("unexpected extra retry")
					}
					return nil
				}
			}
			r, e := s.GetFlavorByRAM(ctx, 4)
			if lane == "clean-retry" {
				if e != nil || flavorRAMID(t, r) != "first" || len(*calls) != 2 || retries != 1 {
					t.Fatal(r, e, *calls, retries)
				}
				return
			}
			if e == nil || r != nil && r.Flavor != nil {
				t.Fatal(r, e)
			}
			if lane == "native401" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(e, &native) || native.Actual != 401 || string(native.Body) != `{"error":"actual auth"}` || len(*calls) != 1 {
					t.Fatal(e, native, *calls)
				}
				return
			}
			if body.Closes() != 1 {
				t.Fatal(body.Closes(), e)
			}
			if strings.HasSuffix(lane, "read-close") && (!errors.Is(e, readErr) || !errors.Is(e, closeErr)) {
				t.Fatal(e)
			}
			if lane == "cancel" && (!errors.Is(e, context.Canceled) || !errors.Is(e, cancelErr)) {
				t.Fatal(e)
			}
			if lane == "restored-source" && !errors.Is(e, resource.ErrInvalidOption) {
				t.Fatal(e)
			}
			var proof *resource.ResponseError
			if !errors.As(e, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Actual") != "physical" || len(proof.Body) == 0 {
				t.Fatal(e, proof)
			}
		})
	}
}
