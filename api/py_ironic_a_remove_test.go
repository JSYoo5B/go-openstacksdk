package api_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Python delete_node, delete_port and delete_allocation default to
// ignore_missing=True and raise NotFoundException when it is False. Other
// failures are never ignored.
func TestPyIronicARemoveMissingPolicyMatchesIgnoreMissing(t *testing.T) {
	cases := []struct {
		name   string
		prefix string
		remove func(ctx context.Context, c *testcloud.Cloud, id string, opts ...resource.LookupOption) error
	}{
		{"nodes", "/v1/nodes/", func(ctx context.Context, c *testcloud.Cloud, id string, opts ...resource.LookupOption) error {
			return newIronicNodes(t, c).Remove(ctx, resource.ID(id), opts...)
		}},
		{"ports", "/v1/ports/", func(ctx context.Context, c *testcloud.Cloud, id string, opts ...resource.LookupOption) error {
			return newIronicPorts(t, c, "").Remove(ctx, resource.ID(id), opts...)
		}},
		{"allocations", "/v1/allocations/", func(ctx context.Context, c *testcloud.Cloud, id string, opts ...resource.LookupOption) error {
			return newIronicAllocations(t, c, "").Remove(ctx, resource.ID(id), opts...)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var deletes int
			cloud.Mux.HandleFunc(tc.prefix, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("only DELETE is expected for an ID reference: %s %s", r.Method, r.URL)
				}
				deletes++
				switch r.URL.Path[len(tc.prefix):] {
				case "gone":
					testcloud.JSON(w, 404, `{"error_message":"gone"}`)
				case "busy":
					testcloud.JSON(w, 409, `{"error_message":"locked"}`)
				default:
					w.WriteHeader(204)
				}
			})
			ctx := context.Background()
			if err := tc.remove(ctx, cloud, "ok"); err != nil {
				t.Fatal(err)
			}
			if err := tc.remove(ctx, cloud, "gone"); err != nil {
				t.Fatalf("default ignore_missing: %v", err)
			}
			if err := tc.remove(ctx, cloud, "gone", resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
				t.Fatalf("ignore_missing=false: %v", err)
			}
			before := deletes
			if err := tc.remove(ctx, cloud, "busy"); err == nil || errors.Is(err, resource.ErrNotFound) {
				t.Fatalf("409 must not be ignored: %v", err)
			}
			if deletes != before+1 {
				t.Fatalf("a rejected delete must not be replayed: %d calls", deletes-before)
			}
		})
	}
}
