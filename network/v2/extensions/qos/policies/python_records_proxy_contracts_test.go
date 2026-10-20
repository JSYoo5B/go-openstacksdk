package policies_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func qosProxyAPI(t *testing.T, handler func(http.ResponseWriter, *http.Request)) (*policies.API, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	count := &atomic.Int32{}
	cloud.Mux.HandleFunc("/neutron/", func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.Header().Set("X-Request-Id", "proxy-actual")
		handler(w, r)
	})
	client := cloud.Client("network", "/catalog/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return policies.New(client), count
}
func TestQoSPolicyProxyListRawFormalsGeneratorAndOpaqueJSON(t *testing.T) {
	api, count := qosProxyAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/neutron/v2.0/alternate-policies" || r.Header.Get("X-Trace") != "owned" || r.Header.Get("OpenStack-API-Version") != "network 2.1" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 203, `{"policies":[{"id":"actual","rules":[9007199254740993]}]}`)
	})
	for _, tc := range []struct{ expression, expected string }{{"[0]", "null"}, {"[*].name", "null"}, {"@ == @", "true"}, {"`[1,2]`", "[1,2]"}} {
		result, err := api.ProxyListRecords(context.Background(), policies.WithRecordListJMESPath(tc.expression))
		if err != nil || result == nil || string(result.JSON) != tc.expected || count.Load() != 0 {
			t.Fatal(result, err, count.Load())
		}
	}
	result, err := api.ProxyListRecords(context.Background(), policies.WithRecordListJMESPath("to_array(@)"), policies.WithRecordListQuery("base_path", "/alternate-policies"), policies.WithRecordListQuery("__conflicting_attrs", json.RawMessage(`{"headers":{"X-Trace":"owned"},"microversion":"2.1","jmespath_filters":"[?"}`)))
	if err != nil || result == nil || result.JSON != nil || count.Load() != 0 {
		t.Fatal(result, err, count.Load())
	}
	array, ok := result.Value.([]any)
	if !ok || len(array) != 1 {
		t.Fatal(result.Value)
	}
	generator, ok := array[0].(*policies.RecordListGenerator)
	if !ok {
		t.Fatal(array)
	}
	if _, err := json.Marshal(generator); err == nil || count.Load() != 0 {
		t.Fatal(err, count.Load())
	}
	for row, err := range generator.Records {
		if err != nil || row.StatusCode != 203 || string(row.Wire.Body["rules"]) != `[9007199254740993]` || row.Header.Get("X-Request-Id") != "proxy-actual" {
			t.Fatal(row, err)
		}
	}
	if count.Load() != 1 {
		t.Fatal(count.Load())
	}
}
func TestQoSPolicyProxySourceLinksLoadedEqualityAndGuardAreOptIn(t *testing.T) {
	api, count := qosProxyAPI(t, func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 203, `{"policies":[{"id":"one","rules":[9007199254740992],"tenant_id":1e-4000}],"links":[],"policies_links":[{"rel":"next","href":"?marker=ignored"}]}`)
	})
	result, err := api.ProxyListRecords(context.Background(), policies.WithRecordListQuery("rules", json.RawMessage(`[9007199254740993.0]`)), policies.WithRecordListQuery("tenant_id", 0))
	if err != nil || result == nil {
		t.Fatal(result, err)
	}
	generator, ok := result.Value.(*policies.RecordListGenerator)
	if !ok {
		t.Fatal(result.Value)
	}
	var ids []string
	for row, err := range generator.Records {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, string(row.Resource.Body["id"]))
	}
	if !reflect.DeepEqual(ids, []string{`"one"`}) || count.Load() != 1 {
		t.Fatal(ids, count.Load())
	}
	guardFault := errors.New("outer selected source changed")
	before := count.Load()
	result, err = api.ProxyListRecords(context.Background(), policies.WithOperationGuard(func(context.Context) error { return guardFault }))
	if result != nil || !errors.Is(err, guardFault) || count.Load() != before {
		t.Fatal(result, err, count.Load())
	}
}
func TestQoSPolicyProxyOptsSnapshotsFormalErrorsAndTypedBasePath(t *testing.T) {
	api, count := qosProxyAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/neutron/v2.0/alternate-policies" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"policies":[]}`)
	})
	path := "/alternate-policies"
	option := policies.WithRecordListOpts(policies.RecordListOpts{BasePath: &path})
	path = "/changed"
	rows, err := api.AllRecords(context.Background(), option)
	if err != nil || len(rows) != 0 || count.Load() != 1 {
		t.Fatal(rows, err, count.Load())
	}
	for _, options := range [][]policies.RecordListOption{{nil}, {policies.WithRecordListQuery("cls", nil)}, {policies.WithRecordListQuery("__conflicting_attrs", map[string]any{"paginated": false})}, {policies.WithRecordListQuery("headers", map[string]any{"X-Trace": 1})}, {policies.WithRecordListQuery("base_path", "https://foreign.invalid")}} {
		before := count.Load()
		result, err := api.ProxyListRecords(context.Background(), options...)
		// Paths are consumed by lazy Resource.list. Bind/formal errors are immediate.
		if result != nil && err == nil {
			generator := result.Value.(*policies.RecordListGenerator)
			for _, failure := range generator.Records {
				err = failure
				if failure != nil {
					break
				}
			}
		}
		if !errors.Is(err, resource.ErrInvalidOption) || count.Load() != before {
			t.Fatal(result, err, count.Load())
		}
	}
}
