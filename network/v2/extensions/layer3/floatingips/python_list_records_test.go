package floatingips_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonIPRecordsQueryClassificationAndAliasOrder(t *testing.T) {
	var calls []pythonIPsCall
	api := pythonIPsAPI(t, &calls, func(r *http.Request) (int, string) {
		q := r.URL.Query()
		if q.Get("status") != "ACTIVE" || q.Get("project_id") != "legacy" || q.Get("tags-any") != "red,blue" || q.Get("fields") != "id" || q.Get("dns_name") != "" || q.Get("name") != "" || q.Get("unknown") != "" {
			t.Fatalf("query %v", q)
		}
		return 200, `{"floatingips":[{"id":"one","floating_ip_address":"old","name":"new","dns_name":"dns","dns_domain":"example.","qos_policy_id":"gold","port_details":{"name":"p","nested":{"alive":true,"extra":1}},"status":"DOWN"}]}`
	})
	rows, err := api.AllRecords(context.Background(),
		floatingips.WithRecordListQuery("project_id", "canonical"), floatingips.WithRecordListQuery("tenant_id", "legacy"),
		floatingips.WithRecordListQuery("status", "ACTIVE"), floatingips.WithRecordListQuery("any_tags", "red,blue"), floatingips.WithRecordListQuery("fields", []string{"id"}),
		floatingips.WithRecordListQuery("name", "new"), floatingips.WithRecordListQuery("dns_name", "dns"), floatingips.WithRecordListQuery("dns_domain", "example."), floatingips.WithRecordListQuery("qos_policy_id", "gold"),
		floatingips.WithRecordListQuery("port_details", map[string]any{"nested": map[string]any{"alive": 1}}), floatingips.WithRecordListQuery("unknown", func() {}))
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	for _, key := range []string{"name", "floating_ip_address"} {
		if string(rows[0].Resource.Body[key]) != `"new"` {
			t.Fatal(key, rows[0].Resource.Body)
		}
	}
	if string(rows[0].Resource.Body["status"]) != `"DOWN"` || string(rows[0].Wire.Body["floating_ip_address"]) != `"old"` {
		t.Fatal(rows[0])
	}
	if len(calls) != 1 {
		t.Fatal(calls)
	}
}

func TestPythonIPRecordsProjectionProofAndClone(t *testing.T) {
	var calls []pythonIPsCall
	page := `{"floatingips":[{"id":"one","floating_ip_address":"203.0.113.1","revision_number":"٢","port_details":[["a",1]],"tags":"xy","if-match":"ab","extension":{"large":9007199254740993}}]}`
	api := pythonIPsAPI(t, &calls, func(*http.Request) (int, string) { return 202, page })
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	row := rows[0]
	for k, want := range map[string]string{"name": `"203.0.113.1"`, "revision_number": "2", "tags": `["xy"]`, "port_details": `{}`, "dns_name": "null"} {
		if string(row.Resource.Body[k]) != want {
			t.Fatalf("%s=%s want %s", k, row.Resource.Body[k], want)
		}
	}
	if string(row.Resource.Body["if_match"]) != `["ab"]` || row.StatusCode != 202 || string(row.Envelope) != page || row.Header.Get("Content-Type") != "application/json" {
		t.Fatal(row)
	}
	if string(row.Wire.Body["revision_number"]) != `"٢"` || string(row.Wire.Body["extension"]) != `{"large":9007199254740993}` {
		t.Fatal(row.Wire)
	}
	clone := row.Clone()
	clone.Resource.Body["name"][0] = 'x'
	clone.Wire.Body["id"][0] = 'x'
	clone.Envelope[0] = 'x'
	clone.Header.Set("X-Clone", "changed")
	if string(row.Resource.Body["name"]) != `"203.0.113.1"` || string(row.Wire.Body["id"]) != `"one"` || row.Envelope[0] != '{' || row.Header.Get("X-Clone") != "" {
		t.Fatal("clone aliases original")
	}
	if (*floatingips.ListRecord)(nil).Clone() != nil {
		t.Fatal("nil clone")
	}
}

func TestPythonIPRecordsCapsBeforeFiltersAndLazyBreak(t *testing.T) {
	var calls []pythonIPsCall
	api := pythonIPsAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"floatingips":[{"id":"one","dns_name":"other"},{"id":"two","dns_name":"match"},false],"floatingips_links":[{"rel":"next","href":false}]}`
	})
	rows, err := api.AllRecords(context.Background(), floatingips.WithRecordListMaxItems(1), floatingips.WithRecordListQuery("dns_name", "match"))
	if err != nil || len(rows) != 0 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	calls = nil
	for row, err := range api.ListRecords(context.Background()) {
		if err != nil || string(row.Resource.Body["id"]) != `"one"` {
			t.Fatal(row, err)
		}
		break
	}
	if len(calls) != 1 {
		t.Fatal(calls)
	}
	calls = nil
	rows, err = api.AllRecords(context.Background(), floatingips.WithRecordListQuery("max_items", -1))
	if err != nil || len(rows) != 0 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
}

func TestPythonIPRecordsPaginationLateErrorsAndResponseProof(t *testing.T) {
	var calls []pythonIPsCall
	api := pythonIPsAPI(t, &calls, func(r *http.Request) (int, string) {
		if r.URL.Query().Get("marker") == "one" {
			return 503, `{"error":"later"}`
		}
		return 200, `{"floatingips":[{"id":"one"}],"floatingips_links":[{"rel":"next","href":"http://` + r.Host + `/neutron/v2.0/floatingips?marker=one"}]}`
	})
	count := 0
	for row, err := range api.ListRecords(context.Background()) {
		if err != nil {
			var proof gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &proof) || proof.Actual != 503 || !strings.Contains(string(proof.Body), "later") {
				t.Fatal(err)
			}
			continue
		}
		if row == nil {
			t.Fatal("nil row")
		}
		count++
	}
	if count != 1 || len(calls) != 2 {
		t.Fatal(count, calls)
	}
	rows, err := api.AllRecords(context.Background())
	if err == nil || rows != nil {
		t.Fatal(rows, err)
	}
}

func TestPythonIPRecordsOwnedOptionsHeadersAndCancellation(t *testing.T) {
	var calls []pythonIPsCall
	api := pythonIPsAPI(t, &calls, func(r *http.Request) (int, string) {
		if r.Header.Get("X-Owned") != "old" || r.Header.Get("OpenStack-API-Version") != "network 2.1" || r.URL.Query().Get("status") != "ACTIVE" {
			t.Fatal(r.Header, r.URL)
		}
		return 200, `{"floatingips":[]}`
	})
	raw := json.RawMessage(`"ACTIVE"`)
	headers := map[string]string{"X-Owned": "old"}
	version := "2.1"
	option := floatingips.WithRecordListOpts(floatingips.RecordListOpts{Query: map[string]json.RawMessage{"status": raw}, Headers: headers, Microversion: &version})
	raw[1] = 'X'
	headers["X-Owned"] = "changed"
	version = "2.9"
	if _, err := api.AllRecords(context.Background(), option); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls = nil
	if _, err := api.AllRecords(ctx, option); !errors.Is(err, context.Canceled) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	var missing *floatingips.API
	if _, err := missing.AllRecords(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.AllRecords(context.Background(), nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestPythonIPRecordsProxyGeneratorAndJMESPath(t *testing.T) {
	var calls []pythonIPsCall
	api := pythonIPsAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"floatingips":[{"id":"one"}]}` })
	got, err := api.ProxyListRecords(context.Background(), floatingips.WithRecordListJMESPath("type(@)"))
	if err != nil || string(got.JSON) != "null" || len(calls) != 0 {
		t.Fatal(got, err, calls)
	}
	got, err = api.ProxyListRecords(context.Background(), floatingips.WithRecordListJMESPath("@"))
	if err != nil || got.JSON != nil {
		t.Fatal(got, err)
	}
	generator, ok := got.Value.(*floatingips.RecordListGenerator)
	if !ok {
		t.Fatal(got.Value)
	}
	count := 0
	for _, err := range generator.Records {
		if err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count != 1 {
		t.Fatal(count)
	}
	calls = nil
	if _, err := api.ProxyListRecords(context.Background(), floatingips.WithRecordListJMESPath("length(@)")); err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	if _, err := api.AllRecords(context.Background(), floatingips.WithRecordListJMESPath("@")); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonIPRecordsDescriptorFaultKeepsPhysicalPage(t *testing.T) {
	for _, page := range []string{`{"floatingips":[{"revision_number":"²"}]}`, `{"floatingips":[false]}`, `{"floatingips":[{"connection":{}}]}`} {
		t.Run(page, func(t *testing.T) {
			var calls []pythonIPsCall
			api := pythonIPsAPI(t, &calls, func(*http.Request) (int, string) { return 202, page })
			rows, err := api.AllRecords(context.Background())
			var proof *resource.ResponseError
			if rows != nil || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != page || proof.Header.Get("Content-Type") != "application/json" {
				t.Fatal(rows, err, proof)
			}
		})
	}
}
