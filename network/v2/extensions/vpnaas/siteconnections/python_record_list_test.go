package siteconnections_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func siteListField(t *testing.T, row *siteconnections.SiteConnectionRecord, key, want string) {
	t.Helper()
	if row == nil || row.Resource == nil || string(row.Resource.Body[key]) != want {
		t.Fatal(key, want, row)
	}
}
func TestVPNSiteRecordsDescriptorsDefaultsAndOwnedProof(t *testing.T) {
	calls := []nativeVPNCall{}
	body := `{"ipsec_site_connections":[{"id":"x","admin_state_up":"false","interval":"030","mtu":1500.9,"timeout":120,"peer_cidrs":"10.0.0.0/24","dpd":false,"tenant_id":"p","self":"drop","extension":{"n":9007199254740993}},{}]}`
	api, _ := nativeVPNAPI(t, &calls, func(*http.Request) *http.Response { return nativeVPNWire(203, body) })
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 2 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	for k, v := range map[string]string{"is_admin_state_up": "true", "interval": "30", "mtu": "1500", "timeout": "120", "peer_cidrs": `["10.0.0.0/24"]`, "dpd": "{}", "project_id": `"p"`, "location": "null"} {
		siteListField(t, rows[0], k, v)
	}
	for _, k := range []string{"action", "interval", "timeout", "mtu", "dpd", "peer_cidrs", "is_admin_state_up", "name", "id"} {
		siteListField(t, rows[1], k, "null")
	}
	if rows[0].Resource.Body["self"] != nil || string(rows[0].Wire.Body["self"]) != `"drop"` || string(rows[0].Envelope) != body || rows[0].StatusCode != 203 || string(rows[0].Wire.Body["interval"]) != `"030"` {
		t.Fatal(rows[0])
	}
	rows[0].Resource.Body["extension"][0] = 'X'
	rows[0].Envelope[0] = 'X'
	rows[0].Header.Set("Content-Type", "changed")
	if string(rows[0].Wire.Body["extension"]) != `{"n":9007199254740993}` || rows[0].Wire.Header.Get("Content-Type") != "application/json" {
		t.Fatal("aliased proof")
	}
}
func TestVPNSiteRecordsServerQueryAndLocalFiltersAcrossPages(t *testing.T) {
	calls := []nativeVPNCall{}
	api, _ := nativeVPNAPI(t, &calls, func(req *http.Request) *http.Response {
		q := req.URL.Query()
		if q.Get("status") != "ACTIVE" || q.Get("admin_state_up") != "False" || q.Get("mtu") != "1500" || q.Get("fields") != "id" || q.Get("interval") != "" || q.Get("peer_cidrs") != "" || q.Get("tenant_id") != "" || q.Get("unknown") != "" {
			t.Error(q)
		}
		if q.Get("marker") == "p2" {
			return nativeVPNWire(200, `{"ipsec_site_connections":[{"id":"yes","interval":"30","peer_cidrs":"10.0.0.0/24"}]}`)
		}
		return nativeVPNWire(200, `{"ipsec_site_connections":[{"id":"no","interval":10}],"ipsec_site_connections_links":[{"rel":"next","href":"?marker=p2"}]}`)
	})
	seq := api.ListRecords(context.Background(), siteconnections.WithRecordListQuery("status", "ACTIVE"), siteconnections.WithRecordListQuery("is_admin_state_up", false), siteconnections.WithRecordListQuery("mtu", 1500), siteconnections.WithRecordListQuery("fields", []string{"id"}), siteconnections.WithRecordListQuery("interval", 30), siteconnections.WithRecordListQuery("peer_cidrs", []string{"10.0.0.0/24"}), siteconnections.WithRecordListQuery("tenant_id", make(chan int)), siteconnections.WithRecordListQuery("unknown", make(chan int)))
	if len(calls) != 0 {
		t.Fatal("eager HTTP")
	}
	rows := []*siteconnections.SiteConnectionRecord{}
	for r, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, r)
	}
	if len(rows) != 1 || len(calls) != 2 {
		t.Fatal(rows, calls)
	}
	siteListField(t, rows[0], "id", `"yes"`)
}
func TestVPNSiteRecordsMarkerCountsPhysicalRowsBeforeFiltering(t *testing.T) {
	calls := []nativeVPNCall{}
	api, _ := nativeVPNAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("marker") == "c" {
			return nativeVPNWire(200, `{"ipsec_site_connections":[]}`)
		}
		if req.URL.Query().Get("marker") == "b" {
			return nativeVPNWire(200, `{"ipsec_site_connections":[{"id":"c","action":"hold"}]}`)
		}
		return nativeVPNWire(200, `{"ipsec_site_connections":[{"id":"a","action":"clear"},{"id":"b","action":"clear"}]}`)
	})
	rows, err := api.AllRecords(context.Background(), siteconnections.WithRecordListLimit(2), siteconnections.WithRecordListQuery("action", "hold"))
	if err != nil || len(rows) != 1 || len(calls) != 3 || !strings.Contains(calls[1].query, "marker=b") || !strings.Contains(calls[2].query, "marker=c") {
		t.Fatal(rows, err, calls)
	}
	siteListField(t, rows[0], "id", `"c"`)
}
func TestVPNSiteRecordsControlsUnknownAndOwnedOptions(t *testing.T) {
	for _, key := range []string{"cls", "resource_type", "session", "self"} {
		t.Run(key, func(t *testing.T) {
			calls := []nativeVPNCall{}
			api, _ := nativeVPNAPI(t, &calls, func(*http.Request) *http.Response { t.Error("control HTTP"); return nativeVPNWire(200, `{}`) })
			_, err := api.AllRecords(context.Background(), siteconnections.WithRecordListQuery(key, make(chan int)))
			if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
				t.Fatal(err, calls)
			}
		})
	}
	calls := []nativeVPNCall{}
	headers := map[string]string{"X-Leaf": "owned"}
	raw := json.RawMessage(`"hold"`)
	query := map[string]json.RawMessage{"action": raw}
	opt := siteconnections.WithRecordListOpts(siteconnections.RecordListOpts{Headers: headers, Query: query})
	headers["X-Leaf"] = "changed"
	raw[1] = 'X'
	api, _ := nativeVPNAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.Header.Get("X-Leaf") != "owned" {
			t.Error(req.Header)
		}
		return nativeVPNWire(200, `{"ipsec_site_connections":[{"id":"x","action":"hold"}]}`)
	})
	for range 2 {
		rows, err := api.AllRecords(context.Background(), opt)
		if err != nil || len(rows) != 1 {
			t.Fatal(rows, err)
		}
	}
	if len(calls) != 2 {
		t.Fatal(calls)
	}
}
func TestVPNSiteRecordsProxyOpaqueGeneratorExpressions(t *testing.T) {
	calls := []nativeVPNCall{}
	api, _ := nativeVPNAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeVPNWire(200, `{"ipsec_site_connections":[{"id":"x"}]}`)
	})
	result, err := api.ProxyListRecords(context.Background(), siteconnections.WithRecordListJMESPath("@"))
	if err != nil || len(calls) != 0 {
		t.Fatal(result, err, calls)
	}
	generator, ok := result.Value.(*siteconnections.RecordListGenerator)
	if !ok {
		t.Fatal(result)
	}
	if _, err := json.Marshal(generator); err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	var ids []string
	for row, err := range generator.Records {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, string(row.Resource.Body["id"]))
	}
	if !reflect.DeepEqual(ids, []string{`"x"`}) || len(calls) != 1 {
		t.Fatal(ids, calls)
	}
	for _, expr := range []string{"[].id", "type(@)", "length(@)"} {
		t.Run(expr, func(t *testing.T) {
			before := len(calls)
			result, err := api.ProxyListRecords(context.Background(), siteconnections.WithRecordListJMESPath(expr))
			if expr == "length(@)" {
				if err == nil {
					t.Fatal(result)
				}
			} else {
				want := "null"

				if err != nil || string(result.JSON) != want {
					t.Fatal(result, err)
				}
			}
			if len(calls) != before {
				t.Fatal("expression consumed generator", calls)
			}
		})
	}
	if _, err := api.AllRecords(context.Background(), siteconnections.WithRecordListJMESPath("@")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
func TestVPNSiteRecordsFailureDropsAllPreservesYieldAndReceipt(t *testing.T) {
	calls := []nativeVPNCall{}
	api, _ := nativeVPNAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("marker") == "bad" {
			r := nativeVPNWire(409, `{"error":"late"}`)
			r.Header.Set("X-Failure", "actual")
			return r
		}
		return nativeVPNWire(200, `{"ipsec_site_connections":[{"id":"x"}],"ipsec_site_connections_links":[{"rel":"next","href":"?marker=bad"}]}`)
	})
	rows, err := api.AllRecords(context.Background())
	var native gophercloud.ErrUnexpectedResponseCode
	if rows != nil || !errors.As(err, &native) || native.Actual != 409 || string(native.Body) != `{"error":"late"}` || native.ResponseHeader.Get("X-Failure") != "actual" {
		t.Fatal(rows, err, native)
	}
	count := 0
	for row, err := range api.ListRecords(context.Background()) {
		if err != nil {
			if !errors.As(err, &native) {
				t.Fatal(err)
			}
			break
		}
		if row != nil {
			count++
		}
	}
	if count != 1 {
		t.Fatal(count)
	}
}
func TestVPNSiteRecordsCancellationAndRouteGuard(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("stop list")
	cancel(cause)
	var nilAPI *siteconnections.API
	if _, err := nilAPI.AllRecords(ctx); !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	calls := []nativeVPNCall{}
	api, _ := nativeVPNAPI(t, &calls, func(*http.Request) *http.Response { t.Error("preflight HTTP"); return nativeVPNWire(200, `{}`) })
	entered := false
	_, err := api.AllRecords(context.Background(), func(*siteconnections.RecordListOpts) error {
		entered = true
		api.RawClient().ResourceBase = "https://changed.invalid/v2.0/"
		return nil
	})
	if !entered || !errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(err.Error(), "source or route changed") || len(calls) != 0 {
		t.Fatal(fmt.Sprint(err), entered, calls)
	}
}
