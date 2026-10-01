package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"gophercloudsdk/clustering/v1/profiles"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

func TestClusteringProfilesUpdateSnapshotsHeadersBeforeLookupAndRechecksVersion(t *testing.T) {
	for _, invalidVersion := range []bool{false, true} {
		t.Run(map[bool]string{false: "input_snapshot", true: "changed_version"}[invalidVersion], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("clustering", "/senlin/v1")
			client.Microversion = "1.2"
			var captured *request.Config[profiles.UpdateOpts]
			lookups, patches := 0, 0
			cloud.Mux.HandleFunc("GET /senlin/v1/profiles", func(w http.ResponseWriter, r *http.Request) {
				lookups++
				if r.URL.Query().Get("name") != "match" || captured == nil {
					t.Error(r.URL, captured)
				}
				captured.Headers["X-Vendor"] = "changed during lookup"
				changed := "changed"
				captured.Options.Name = &changed
				captured.Fields["vendor"] = json.RawMessage(`{"enabled":true}`)
				if invalidVersion {
					client.Microversion = "latest"
				}
				testcloud.JSON(w, 200, `{"profiles":[{"id":"fixed-id","name":"match"}]}`)
			})
			cloud.Mux.HandleFunc("PATCH /senlin/v1/profiles/fixed-id", func(w http.ResponseWriter, r *http.Request) {
				patches++
				if r.Header.Get("X-Vendor") != "original" || r.Header.Get("OpenStack-API-Version") != "clustering 1.2" {
					t.Error(r.Header)
				}
				var body map[string]map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || string(body["profile"]["name"]) != `"updated"` || string(body["profile"]["vendor"]) != `{"enabled":false}` {
					t.Errorf("lookup changed prepared body: %v %v", body, err)
				}
				testcloud.JSON(w, 200, `{"profile":{"id":"fixed-id","name":"updated"}}`)
			})
			name := "updated"
			value, err := profiles.New(client).Update(context.Background(), resource.Name("match"), profiles.UpdateOpts{Name: &name},
				profiles.WithUpdateHeader("X-Vendor", "original"), profiles.WithUpdateField("vendor", map[string]any{"enabled": false}),
				func(config *request.Config[profiles.UpdateOpts]) error { captured = config; return nil })
			if invalidVersion {
				if !errors.Is(err, resource.ErrUnsupported) || patches != 0 || value != nil {
					t.Fatalf("changed version sent mutation: value=%v patches=%d err=%v", value, patches, err)
				}
			} else if err != nil || value == nil || value.Name != "updated" || patches != 1 {
				t.Fatalf("prepared mutation changed: value=%v patches=%d err=%v", value, patches, err)
			}
			if lookups != 1 {
				t.Fatalf("name resolved %d times", lookups)
			}
		})
	}
}
