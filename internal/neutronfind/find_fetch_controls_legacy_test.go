package neutronfind_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Adding an explicitly selected fetch-control entry point must not make raw
// controls valid on existing Find/Get bindings, even for Source-shaped profiles.
func TestFetchControlsLegacyEntryPointsRejectRawControls(t *testing.T) {
	for _, source := range []bool{false, true} {
		for _, method := range []string{"Find", "Get"} {
			for _, key := range []string{"requires_id", "error_message", "skip_cache", "resource_response_key"} {
				name := "baseline/"
				if source {
					name = "source/"
				}
				t.Run(name+method+"/"+key, func(t *testing.T) {
					cloud := testcloud.New(t)
					calls := 0
					cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
						calls++
						testcloud.JSON(w, http.StatusOK, `{"record":{"id":"one"}}`)
					})
					engine, err := neutronfind.New(context.Background(), cloud.Client("network", "/v2.0/"), policyProfile(source), nil)
					if err != nil {
						t.Fatal(err)
					}
					policy := neutronfind.Options{Query: map[string]json.RawMessage{key: json.RawMessage(`null`)}}
					var row *neutronfind.Record
					if method == "Find" {
						row, err = engine.Find(context.Background(), "one", policy)
					} else {
						row, err = engine.Get(context.Background(), "one", policy)
					}
					if !errors.Is(err, resource.ErrInvalidOption) || row != nil || calls != 0 {
						t.Fatalf("row=%v err=%v HTTP calls=%d", row, err, calls)
					}
				})
			}
		}
	}
}

// These two existing policy lanes intentionally differ: baseline overrides both
// requests, while SourceFindPolicy applies the override to member GET only.
func TestFetchControlsLegacyFindKeepsHeadersAndVersionPhases(t *testing.T) {
	for _, source := range []bool{false, true} {
		name := "baseline"
		if source {
			name = "source"
		}
		t.Run(name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests []*http.Request
			cloud.Mux.HandleFunc("/v2.0/records/target", func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.Clone(r.Context()))
				testcloud.JSON(w, http.StatusNotFound, `{"error":"fallback"}`)
			})
			cloud.Mux.HandleFunc("/v2.0/records", func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.Clone(r.Context()))
				testcloud.JSON(w, http.StatusOK, `{"records":[{"id":"one","name":"target"}]}`)
			})
			client := cloud.Client("network", "/v2.0/")
			client.Microversion = "2.0"
			client.MoreHeaders = map[string]string{"X-Client": "selected"}
			engine, err := neutronfind.New(context.Background(), client, policyProfile(source), nil)
			if err != nil {
				t.Fatal(err)
			}
			version := "2.1"
			headers := map[string]string{"X-Call": "owned", "If-Match": "revision-one"}
			row, err := engine.Find(context.Background(), "target", neutronfind.Options{Microversion: &version, Headers: headers})
			if err != nil || row == nil || !row.FromList() || len(requests) != 2 {
				t.Fatalf("row=%v err=%v requests=%d", row, err, len(requests))
			}
			wantVersions := []string{"network 2.1", "network 2.1"}
			if source {
				wantVersions[1] = "network 2.0"
			}
			for i, request := range requests {
				if request.Method != http.MethodGet || request.Header.Get("OpenStack-API-Version") != wantVersions[i] || request.Header.Get("X-Client") != "selected" || request.Header.Get("X-Call") != "owned" || request.Header.Get("If-Match") != "revision-one" {
					t.Fatalf("request %d: method=%s headers=%v", i, request.Method, request.Header)
				}
				if i == 0 && len(request.URL.Query()) != 0 || i == 1 && request.URL.Query().Get("name") != "target" {
					t.Fatalf("request %d query=%v", i, request.URL.Query())
				}
			}
			if client.Microversion != "2.0" || !reflect.DeepEqual(client.MoreHeaders, map[string]string{"X-Client": "selected"}) || !reflect.DeepEqual(headers, map[string]string{"X-Call": "owned", "If-Match": "revision-one"}) {
				t.Fatal("Find changed the selected client or caller header map", client, headers)
			}
		})
	}
}

func TestFetchControlsLegacyGetStillRequiresMemberIdentity(t *testing.T) {
	for _, source := range []bool{false, true} {
		cloud := testcloud.New(t)
		calls := 0
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls++
			testcloud.JSON(w, http.StatusOK, `{"record":{"id":"one"}}`)
		})
		engine, err := neutronfind.New(context.Background(), cloud.Client("network", "/v2.0/"), policyProfile(source), nil)
		if err != nil {
			t.Fatal(err)
		}
		row, err := engine.Get(context.Background(), "", neutronfind.Options{})
		if !errors.Is(err, resource.ErrInvalidOption) || row != nil || calls != 0 {
			t.Fatalf("source=%t row=%v err=%v HTTP calls=%d", source, row, err, calls)
		}
	}
}
