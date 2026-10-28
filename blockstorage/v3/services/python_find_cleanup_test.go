package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestPythonCinderFindServiceListsOnlyAndMatchesAliasIDHost(t *testing.T) {
	for _, identity := range []string{"7", "cinder-volume"} {
		t.Run(identity, func(t *testing.T) {
			api, calls := svcActionAPI(t, "3.32", "3.70", func(r *http.Request) (int, string) {
				if r.Method != http.MethodGet || r.URL.Path != svcActionBase || r.URL.Query().Get("host") != "wanted" || r.URL.Query().Has("name") {
					t.Fatal(r.Method, r.URL)
				}
				return 203, `{"services":[{"id":7,"binary":"cinder-volume","host":"wrong"},{"id":7,"binary":"cinder-volume","host":"wanted","zone":"az1","x_unknown":false}]}`
			})
			row, err := api.FindRecord(context.Background(), identity, services.WithFindRecordQuery("host", "wanted"), services.WithFindRecordQuery("availability_zone", "az1"))
			if err != nil || row == nil || calls.Len() != 1 || string(row.Body["name"]) != `"cinder-volume"` || string(row.Body["host"]) != `"wanted"` || row.StatusCode != 203 || string(row.Wire["x_unknown"]) != "false" {
				t.Fatal(row, err, svcActionCalls(calls))
			}
		})
	}
}
func TestPythonCinderFindServiceMissingDuplicateAndLatePage(t *testing.T) {
	for _, mode := range []string{"missing", "strict", "duplicate", "late"} {
		t.Run(mode, func(t *testing.T) {
			reads := 0
			api, _ := svcActionAPI(t, "3.32", "3.70", func(r *http.Request) (int, string) {
				reads++
				if mode == "missing" || mode == "strict" {
					return 200, `{"services":[]}`
				}
				if reads == 1 {
					return 200, `{"services":[{"id":"one","binary":"match"}],"next":"?marker=later"}`
				}
				if mode == "late" {
					return 503, `{"error":"late proof"}`
				}
				return 200, `{"services":[{"id":"two","binary":"match"}]}`
			})
			row, err := api.FindRecord(context.Background(), "match", services.WithFindRecordIgnoreMissing(mode != "strict"))
			if row != nil {
				t.Fatal(row)
			}
			switch mode {
			case "missing":
				if err != nil {
					t.Fatal(err)
				}
			case "strict":
				if !errors.Is(err, resource.ErrNotFound) {
					t.Fatal(err)
				}
			case "duplicate":
				if !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(err)
				}
			case "late":
				svcActionOperation(t, err, "FindRecord", 503)
			}
		})
	}
}
func TestPythonCinderFindServiceDiscoveryAndOwnedOptions(t *testing.T) {
	query := map[string]json.RawMessage{"host": json.RawMessage(`"original"`)}
	option := services.WithFindRecordOpts(services.FindRecordOpts{Query: query})
	query["host"][1] = 'X'
	api, calls := svcActionAPI(t, "", "3.70", func(r *http.Request) (int, string) {
		if r.URL.Query().Get("host") != "original" {
			t.Fatal(r.URL)
		}
		return 200, `{"services":[{"id":"1","binary":"match","host":"original"}]}`
	})
	row, err := api.FindRecord(context.Background(), "match", option)
	if err != nil || row == nil || calls.Len() != 2 || len(row.Discovery) != 1 {
		t.Fatal(row, err, svcActionCalls(calls))
	}
	svcActionVersion(t, svcActionCalls(calls)[1], "3.32")
}
func TestPythonCinderCleanupWorkersExactDefaultsAndRawResult(t *testing.T) {
	const body = `{"binary":null,"cluster_name":null,"disabled":null,"host":null,"is_up":false,"resource_id":null,"resource_type":null,"service_id":null}`
	api, calls := svcActionAPI(t, "3.24", "3.70", func(r *http.Request) (int, string) {
		if r.Method != http.MethodPut || r.URL.Path != "/cinder/v3/project/workers/cleanup" {
			t.Fatal(r.Method, r.URL)
		}
		return 203, `{"cleaning":[{"id":7,"host":"host"}],"unavailable":[],"x_extension":9007199254740993}`
	})
	row, err := api.CleanupServiceWorkers(context.Background())
	if err != nil || row == nil || calls.Len() != 1 || svcActionCalls(calls)[0].body != body || row.StatusCode != 203 || !strings.Contains(string(row.JSON), "9007199254740993") {
		t.Fatal(row, err, svcActionCalls(calls))
	}
}
func TestPythonCinderCleanupWorkersVersionAndOwnedFilters(t *testing.T) {
	filters := map[string]json.RawMessage{"host": json.RawMessage(`"owned"`), "is_up": json.RawMessage("null"), "service_id": json.RawMessage("7")}
	option := services.WithCleanupWorkersOpts(services.CleanupWorkersOpts{Filters: filters})
	filters["host"][1] = 'X'
	api, calls := svcActionAPI(t, "", "3.70", func(*http.Request) (int, string) { return 200, `{"cleaning":[],"unavailable":[]}` })
	row, err := api.CleanupServiceWorkers(context.Background(), option)
	if err != nil || row == nil || calls.Len() != 2 || len(row.Discovery) != 1 || !strings.Contains(svcActionCalls(calls)[1].body, `"host":"owned"`) || !strings.Contains(svcActionCalls(calls)[1].body, `"is_up":null`) {
		t.Fatal(row, err, svcActionCalls(calls))
	}
	svcActionVersion(t, svcActionCalls(calls)[1], "3.32")
	low, lowCalls := svcActionAPI(t, "3.23", "3.70", func(*http.Request) (int, string) { t.Fatal("unsupported worker request"); return 0, "" })
	if _, err := low.CleanupServiceWorkers(context.Background()); !errors.Is(err, resource.ErrUnsupported) || lowCalls.Len() != 0 {
		t.Fatal(err, svcActionCalls(lowCalls))
	}
}
func TestPythonCinderCleanupWorkersPreflightAndFailureProof(t *testing.T) {
	api, calls := svcActionAPI(t, "3.24", "3.70", func(*http.Request) (int, string) { return 404, `{"error":"cleanup actual"}` })
	for _, options := range [][]services.CleanupWorkersOption{{nil}, {services.WithCleanupWorkerFilter("unknown", true)}} {
		if _, err := api.CleanupServiceWorkers(context.Background(), options...); err == nil || calls.Len() != 0 {
			t.Fatal(err, svcActionCalls(calls))
		}
	}
	row, err := api.CleanupServiceWorkers(context.Background())
	if row == nil || row.JSON != nil || calls.Len() != 1 {
		t.Fatal(row, err, svcActionCalls(calls))
	}
	svcActionOperation(t, err, "CleanupServiceWorkers", 404)
}

func TestPythonCinderFindServiceNamePresenceAndListRouteControls(t *testing.T) {
	api, calls := svcActionAPI(t, "3.32", "3.70", func(r *http.Request) (int, string) {
		if r.URL.Path != "/cinder/v3/project/alternate-services" || r.URL.Query().Get("limit") != "2" {
			t.Fatal(r.URL)
		}
		return 200, `{"services":[{"id":"one","binary":"match","name":null},{"id":"two","binary":"match"}]}`
	})
	row, err := api.FindRecord(context.Background(), "match", services.WithFindRecordQuery("list_base_path", "/alternate-services"), services.WithFindRecordQuery("limit", 2), services.WithFindRecordQuery("paginated", false))
	if err != nil || row == nil || string(row.Body["id"]) != `"two"` || calls.Len() != 1 {
		t.Fatal(row, err, svcActionCalls(calls))
	}
}

func TestPythonCinderFindServiceUsesOwnedResourceListControls(t *testing.T) {
	api, calls := svcActionAPI(t, "3.32", "3.70", func(r *http.Request) (int, string) {
		if r.Method != http.MethodGet || r.URL.Path != "/cinder/v3/project/source-services" || r.Header.Get("X-Raw") != "owned" || r.Header.Get("Accept") != "application/vnd.services+json" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.32" || r.URL.Query().Get("host") != "host" || r.URL.Query().Get("limit") != "2" || len(r.URL.Query()) != 2 {
			t.Fatal(r.Method, r.URL, r.Header)
		}
		return 200, `{"services":[{"id":7,"binary":"cinder-volume","host":"host"}]}`
	})
	row, err := api.FindRecord(context.Background(), "7",
		services.WithFindRecordQuery("list_base_path", "/source-services"),
		services.WithFindRecordQuery("base_path", "/overridden"),
		services.WithFindRecordQuery("headers", map[string]string{"X-Raw": "owned", "Accept": "application/vnd.services+json"}),
		services.WithFindRecordQuery("microversion", "9.99"),
		services.WithFindRecordQuery("__conflicting_attrs", map[string]any{"host": "wrong", "base_path": "https://foreign.invalid"}),
		services.WithFindRecordQuery("jmespath_filters", "[?"),
		services.WithFindRecordQuery("all_projects", true),
		services.WithFindRecordQuery("host", "host"), services.WithFindRecordQuery("limit", 2), services.WithFindRecordQuery("paginated", false))
	if err != nil || row == nil || calls.Len() != 1 {
		t.Fatal(row, err, svcActionCalls(calls))
	}
}

func TestPythonCinderFindServiceRawHeadersValidateAndFalseyControls(t *testing.T) {
	for _, raw := range []json.RawMessage{json.RawMessage("null"), json.RawMessage("false"), json.RawMessage(`""`), json.RawMessage(`{"X-Raw":3}`)} {
		t.Run(string(raw), func(t *testing.T) {
			api, calls := svcActionAPI(t, "3.32", "3.70", func(r *http.Request) (int, string) {
				if r.URL.Path != svcActionBase || r.URL.RawQuery != "" {
					t.Fatal(r.URL)
				}
				return 200, `{"services":[{"id":"1","binary":"match"}]}`
			})
			row, err := api.FindRecord(context.Background(), "match", services.WithFindRecordOpts(services.FindRecordOpts{Query: map[string]json.RawMessage{"headers": raw, "list_base_path": json.RawMessage("false")}}))
			if string(raw) == `{"X-Raw":3}` {
				if err == nil || row != nil || calls.Len() != 0 {
					t.Fatal(row, err, svcActionCalls(calls))
				}
			} else if err != nil || row == nil || calls.Len() != 1 {
				t.Fatal(row, err, svcActionCalls(calls))
			}
		})
	}
}
