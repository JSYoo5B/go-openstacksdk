package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/messages"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/replicas"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharenetworks"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shares"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharetransfers"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharetypes"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/snapshots"
	"github.com/gophercloud/gophercloud/v2"
)

type pyManilaARemover struct {
	name   string
	route  string
	remove func(c *gophercloud.ServiceClient) func(context.Context, resource.Ref, ...resource.LookupOption) error
}

func pyManilaARemovers() []pyManilaARemover {
	return []pyManilaARemover{
		{"delete_share", "/v2/shares/", func(c *gophercloud.ServiceClient) func(context.Context, resource.Ref, ...resource.LookupOption) error {
			return shares.New(c).Remove
		}},
		{"delete_share_snapshot", "/v2/snapshots/", func(c *gophercloud.ServiceClient) func(context.Context, resource.Ref, ...resource.LookupOption) error {
			return snapshots.New(c).Remove
		}},
		{"delete_share_replica", "/v2/share-replicas/", func(c *gophercloud.ServiceClient) func(context.Context, resource.Ref, ...resource.LookupOption) error {
			return replicas.New(c).Remove
		}},
		{"delete_share_network", "/v2/share-networks/", func(c *gophercloud.ServiceClient) func(context.Context, resource.Ref, ...resource.LookupOption) error {
			return sharenetworks.New(c).Remove
		}},
		{"delete_share_transfers", "/v2/share-transfers/", func(c *gophercloud.ServiceClient) func(context.Context, resource.Ref, ...resource.LookupOption) error {
			return sharetransfers.New(c).Remove
		}},
		{"delete_share_type", "/v2/types/", func(c *gophercloud.ServiceClient) func(context.Context, resource.Ref, ...resource.LookupOption) error {
			return sharetypes.New(c).Remove
		}},
		{"delete_user_message", "/v2/messages/", func(c *gophercloud.ServiceClient) func(context.Context, resource.Ref, ...resource.LookupOption) error {
			return messages.New(c).Remove
		}},
	}
}

// Python delete_*(x, ignore_missing=True) issues a single DELETE for an ID,
// swallows only 404 and does not look the ID up first.
func TestPyManilaADeleteByIDIsOneDeleteAndIgnoresMissingByDefault(t *testing.T) {
	for _, tc := range pyManilaARemovers() {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			status := 202
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != http.MethodDelete || r.URL.Path != tc.route+"item-1" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL)
				}
				testcloud.JSON(w, status, `{"message":"x"}`)
			})
			remove := tc.remove(cloud.Client("sharev2", "/v2"))
			if err := remove(context.Background(), resource.ID("item-1")); err != nil {
				t.Fatal(err)
			}
			status = 404
			if err := remove(context.Background(), resource.ID("item-1")); err != nil {
				t.Fatalf("404 must be ignored by default: %v", err)
			}
			if calls.Load() != 2 {
				t.Fatalf("calls=%d", calls.Load())
			}
		})
	}
}

// ignore_missing=False raises NotFound; non-404 failures are never swallowed.
func TestPyManilaADeleteMissingErrorAndOtherFailuresPropagateWithoutRetry(t *testing.T) {
	for _, tc := range pyManilaARemovers() {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			status := 404
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, status, `{"error":"x"}`)
			})
			remove := tc.remove(cloud.Client("sharev2", "/v2"))
			err := remove(context.Background(), resource.ID("item-1"), resource.WithMissingError())
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.Is(err, resource.ErrNotFound) || !errors.As(err, &native) || native.Actual != 404 {
				t.Fatalf("want ErrNotFound wrapping native 404, got %v", err)
			}
			for _, code := range []int{403, 500} {
				status = code
				err := remove(context.Background(), resource.ID("item-1"))
				if err == nil || errors.Is(err, resource.ErrNotFound) || !errors.As(err, &native) || native.Actual != code {
					t.Fatalf("status %d error=%v", code, err)
				}
			}
			if calls.Load() != 3 {
				t.Fatalf("mutation replayed or skipped: calls=%d", calls.Load())
			}
		})
	}
}

func TestPyManilaADeleteCancelledContextSendsNothing(t *testing.T) {
	for _, tc := range pyManilaARemovers() {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected request %s", r.URL)
			})
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			err := tc.remove(cloud.Client("sharev2", "/v2"))(ctx, resource.ID("item-1"))
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func pyManilaAReadBody(t *testing.T, r *http.Request) string {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
	}
	return string(data)
}

// delete_share/delete_share_snapshot/delete_share_replica with force=True post
// {"force_delete": null} to the action route instead of DELETE.
func TestPyManilaAForceRemovePostsForceDeleteAction(t *testing.T) {
	cases := []struct {
		name, route string
		force       func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.LookupOption) error
	}{
		{"share", "/v2/shares/item-1/action", func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.LookupOption) error {
			return shares.New(c).ForceRemove
		}},
		{"snapshot", "/v2/snapshots/item-1/action", func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.LookupOption) error {
			return snapshots.New(c).ForceRemove
		}},
		{"replica", "/v2/share-replicas/item-1/action", func(c *gophercloud.ServiceClient) func(context.Context, string, ...resource.LookupOption) error {
			return replicas.New(c).ForceRemove
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			status := 202
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != tc.route {
					t.Errorf("unexpected request %s %s", r.Method, r.URL)
				}
				if got := pyManilaAReadBody(t, r); got != `{"force_delete":null}` {
					t.Errorf("body=%s", got)
				}
				testcloud.JSON(w, status, `{}`)
			})
			force := tc.force(cloud.Client("sharev2", "/v2"))
			if err := force(context.Background(), "item-1"); err != nil {
				t.Fatal(err)
			}
			status = 404
			if err := force(context.Background(), "item-1"); err != nil {
				t.Fatalf("404 ignored by default: %v", err)
			}
			err := force(context.Background(), "item-1", resource.WithMissingError())
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.Is(err, resource.ErrNotFound) || !errors.As(err, &native) {
				t.Fatalf("err=%v", err)
			}
			status = 500
			if err := force(context.Background(), "item-1"); err == nil || errors.Is(err, resource.ErrNotFound) {
				t.Fatalf("500 must fail: %v", err)
			}
			if calls.Load() != 4 {
				t.Fatalf("calls=%d", calls.Load())
			}
		})
	}
}

func TestPyManilaAForceRemoveRejectsInvalidInputsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("unexpected request %s", r.URL) })
	api := shares.New(cloud.Client("sharev2", "/v2"))
	for _, id := range []string{"", "a/b", "a b", ".."} {
		if err := api.ForceRemove(context.Background(), id); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("id %q err=%v", id, err)
		}
	}
	if err := api.ForceRemove(context.Background(), "ok", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option err=%v", err)
	}
}
