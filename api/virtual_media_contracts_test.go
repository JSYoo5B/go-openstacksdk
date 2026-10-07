package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/baremetal/v1/nodes"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestVirtualMediaHasTypedFieldsAndPreservesResponseMetadata(t *testing.T) {
	for _, inserted := range []bool{true, false} {
		t.Run(map[bool]string{true: "inserted", false: "detached"}[inserted], func(t *testing.T) {
			cloud := testcloud.New(t)
			image := ""
			if inserted {
				image = "https://image.invalid/disk.iso"
			}
			cloud.Mux.HandleFunc("/baremetal/nodes/node/vmedia", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.Header.Get("X-OpenStack-Ironic-API-Version") != "1.93" {
					t.Errorf("method=%s headers=%v", r.Method, r.Header)
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Openstack-Request-Id", "req-media")
				_ = json.NewEncoder(w).Encode(map[string]any{"image": image, "inserted": inserted, "media_types": []string{"CD", "DVD"}, "vendor:device": map[string]bool{"enabled": false}})
			})
			client := cloud.Client("baremetal", "/baremetal")
			client.Microversion = "1.93"
			media, err := nodes.New(client).GetVirtualMedia(context.Background(), "node")
			if err != nil {
				t.Fatal(err)
			}
			if media.Image != image || media.Inserted != inserted || len(media.MediaTypes) != 2 || media.MediaTypes[1] != "DVD" || media.Header.Get("X-Openstack-Request-Id") != "req-media" {
				t.Fatalf("media=%+v", media)
			}
			var body map[string]json.RawMessage
			if err := json.Unmarshal(media.Body, &body); err != nil || string(body["vendor:device"]) != `{"enabled":false}` {
				t.Fatalf("body=%s err=%v", media.Body, err)
			}
		})
	}
}

func TestVirtualMediaErrorsAreReturnedWithoutPartialModels(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"invalid-body", `{"inserted":"true"}`, 200},
		{"unsupported-microversion", `{"error":"unsupported version"}`, 406},
		{"forbidden", `{"error":"forbidden"}`, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/baremetal/nodes/node/vmedia", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, tc.status, tc.body) })
			media, err := nodes.New(cloud.Client("baremetal", "/baremetal")).GetVirtualMedia(context.Background(), "node")
			var operation *resource.OperationError
			if media != nil || err == nil || !errors.As(err, &operation) {
				t.Fatalf("media=%v err=%v", media, err)
			}
			if tc.status != 200 {
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) || response.Actual != tc.status {
					t.Fatal(err)
				}
			}
		})
	}
}
