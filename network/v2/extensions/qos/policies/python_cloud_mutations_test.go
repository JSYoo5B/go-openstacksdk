package policies_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func qosMutationLeafFixture(t *testing.T, handler func(http.ResponseWriter, *http.Request)) (*policies.API, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	count := &atomic.Int32{}
	cloud.Mux.HandleFunc("/neutron/", func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.Header().Set("X-Request-Id", "leaf-actual")
		handler(w, r)
	})
	client := cloud.Client("network", "/catalog/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return policies.New(client), count
}
func TestPolicyCloudMutationBorrowedLedgerIgnoresExportedTamperingAndBindsOwner(t *testing.T) {
	api, count := qosMutationLeafFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatal("tampered raw ledger caused PUT", r.Method)
		}
		testcloud.JSON(w, 203, `{"policy":{"id":"actual","name":"gold","shared":true,"x_new":9007199254740993}}`)
	})
	current, err := api.FindCloudMutationRecord(context.Background(), "actual")
	if err != nil || current == nil {
		t.Fatal(current, err)
	}
	clone := current.Clone()
	current.Resource.Body["name"] = json.RawMessage(`"evil"`)
	current.Wire.Body["name"] = json.RawMessage(`"evil"`)
	current.Header.Set("X-Request-Id", "evil")
	current.Envelope[0] = 'X'
	clone.Resource.Body["is_shared"] = json.RawMessage(`false`)
	for _, seed := range []*policies.PolicyRecord{current, clone} {
		result, changed, err := api.UpdateCloudRecord(context.Background(), seed, map[string]json.RawMessage{"name": json.RawMessage(`"gold"`), "shared": json.RawMessage(`1`)})
		if err != nil || changed || result == nil || string(result.Resource.Body["name"]) != `"gold"` || string(result.Resource.Body["is_shared"]) != `true` || string(result.Wire.Body["name"]) != `"gold"` || result.Header.Get("X-Request-Id") != "leaf-actual" || count.Load() != 1 {
			t.Fatal(result, changed, err, count.Load())
		}
	}
	foreign := policies.New(api.RawClient())
	if result, changed, err := foreign.UpdateCloudRecord(context.Background(), clone, map[string]json.RawMessage{"name": json.RawMessage(`"new"`)}); result != nil || changed || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != 1 {
		t.Fatal(result, changed, err, count.Load())
	}
	api.RawClient().ResourceBase += "changed/"
	if result, _, err := api.UpdateCloudRecord(context.Background(), clone, map[string]json.RawMessage{"name": json.RawMessage(`"new"`)}); result != nil || !errors.Is(err, resource.ErrInvalidOption) || count.Load() != 1 {
		t.Fatal(result, err, count.Load())
	}
}
func TestPolicyCloudMutationListConstructorRawLedgerAndPartialResponseSeed(t *testing.T) {
	api, count := qosMutationLeafFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			data, _ := io.ReadAll(r.Body)
			if string(data) != `{"policy":{"name":"new"}}` {
				t.Error(string(data))
			}
			testcloud.JSON(w, 200, `{"policy":{"id":"actual","x_reply":null}}`)
			return
		}
		if r.URL.Path == "/neutron/v2.0/qos/policies" {
			testcloud.JSON(w, 203, `{"policies":[{"id":"actual","name":"gold","is_shared":"false","tenant_id":"owner"}]}`)
		} else {
			testcloud.JSON(w, 404, `{}`)
		}
	})
	current, err := api.FindCloudMutationRecord(context.Background(), "gold")
	if err != nil || current == nil || count.Load() != 2 {
		t.Fatal(current, err, count.Load())
	}
	result, changed, err := api.UpdateCloudRecord(context.Background(), current, map[string]json.RawMessage{"shared": json.RawMessage(`"false"`)})
	if err != nil || changed || result == nil || string(result.Resource.Body["is_shared"]) != `true` || count.Load() != 2 {
		t.Fatal(result, changed, err, count.Load())
	}
	result, changed, err = api.UpdateCloudRecord(context.Background(), current, map[string]json.RawMessage{"name": json.RawMessage(`"new"`)})
	if err != nil || !changed || result == nil || string(result.Resource.Body["name"]) != `"new"` || string(result.Resource.Body["project_id"]) != `"owner"` || string(result.Resource.Body["is_shared"]) != `true` || result.Wire.Body["name"] != nil || result.StatusCode != 200 || count.Load() != 3 {
		t.Fatal(result, changed, err, count.Load())
	}
	if string(current.Resource.Body["name"]) != `"gold"` || string(current.Wire.Body["name"]) != `"gold"` {
		t.Fatal(current)
	}
}

func TestPolicyCloudMutationSelectedVersionAndEqualUpdateReset(t *testing.T) {
	for _, lane := range []string{"create", "dirty", "equal"} {
		t.Run(lane, func(t *testing.T) {
			api, count := qosMutationLeafFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("OpenStack-API-Version") != "network 2.1" {
					t.Error(r.Header)
				}
				if r.Method == http.MethodGet {
					testcloud.JSON(w, 200, `{"policy":{"id":"actual","name":"same","microversion":"foreign"}}`)
					return
				}
				testcloud.JSON(w, 200, `{"policy":{"id":"actual","microversion":"foreign"}}`)
			})
			api.RawClient().Microversion = "2.1"
			var result *policies.PolicyRecord
			var err error
			var changed bool
			wantVersion := `"2.1"`
			wantCalls := int32(1)
			if lane == "create" {
				result, err = api.CreateCloudRecord(context.Background(), map[string]json.RawMessage{"name": json.RawMessage(`"same"`)})
			} else {
				var current *policies.PolicyRecord
				current, err = api.FindCloudMutationRecord(context.Background(), "actual")
				if err != nil {
					t.Fatal(err)
				}
				name := `"new"`
				if lane == "equal" {
					name = `"same"`
					wantVersion = "null"
				} else {
					wantCalls = 2
				}
				result, changed, err = api.UpdateCloudRecord(context.Background(), current, map[string]json.RawMessage{"name": json.RawMessage(name)})
				if changed != (lane == "dirty") {
					t.Fatal(changed, lane)
				}
			}
			if err != nil || result == nil || string(result.Resource.Body["microversion"]) != wantVersion || string(result.Wire.Body["microversion"]) != `"foreign"` || count.Load() != wantCalls {
				t.Fatal(result, err, count.Load(), wantVersion)
			}
		})
	}
}
