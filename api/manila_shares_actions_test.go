package api_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shares"
)

type manilaShareActionCase struct {
	name   string
	body   string
	accept []int
	call   func(api *shares.API) error
}

func manilaShareActionCases() []manilaShareActionCase {
	ctx := context.Background()
	return []manilaShareActionCase{
		{"Extend", `{"extend":{"new_size":0}}`, []int{202}, func(a *shares.API) error { return a.Extend(ctx, "share-1", shares.ExtendOpts{}) }},
		{"ExtendExtension", `{"extend":{"new_size":5,"vendor":false}}`, []int{202}, func(a *shares.API) error {
			return a.Extend(ctx, "share-1", shares.ExtendOpts{NewSize: 5}, shares.WithExtendField("vendor", false))
		}},
		{"Shrink", `{"shrink":{"new_size":0}}`, []int{202}, func(a *shares.API) error { return a.Shrink(ctx, "share-1", shares.ShrinkOpts{}) }},
		{"ShrinkExtension", `{"shrink":{"new_size":1,"vendor":[]}}`, []int{202}, func(a *shares.API) error {
			return a.Shrink(ctx, "share-1", shares.ShrinkOpts{NewSize: 1}, shares.WithShrinkField("vendor", []string{}))
		}},
		{"Revert", `{"revert":{"snapshot_id":"snap-1"}}`, []int{202}, func(a *shares.API) error {
			return a.Revert(ctx, "share-1", shares.RevertOpts{SnapshotID: "snap-1"})
		}},
		{"RevertEmptySnapshot", `{"revert":{"snapshot_id":""}}`, []int{202}, func(a *shares.API) error { return a.Revert(ctx, "share-1", shares.RevertOpts{}) }},
		{"ResetStatus", `{"reset_status":{"status":"error"}}`, []int{202}, func(a *shares.API) error {
			return a.ResetStatus(ctx, "share-1", shares.ResetStatusOpts{Status: "error"})
		}},
		{"ResetStatusExtension", `{"reset_status":{"status":"","vendor":null}}`, []int{202}, func(a *shares.API) error {
			return a.ResetStatus(ctx, "share-1", shares.ResetStatusOpts{}, shares.WithResetStatusField("vendor", nil))
		}},
		{"ForceDelete", `{"force_delete":null}`, []int{202}, func(a *shares.API) error { return a.ForceDelete(ctx, "share-1") }},
		{"Unmanage", `{"unmanage":null}`, []int{202}, func(a *shares.API) error { return a.Unmanage(ctx, "share-1") }},
		{"RevokeAccess", `{"deny_access":{"access_id":"rule-1"}}`, []int{200, 202}, func(a *shares.API) error {
			return a.RevokeAccess(ctx, "share-1", shares.RevokeAccessOpts{AccessID: "rule-1"})
		}},
		{"RevokeAccessExtension", `{"deny_access":{"access_id":"","vendor":{"a":false}}}`, []int{200, 202}, func(a *shares.API) error {
			return a.RevokeAccess(ctx, "share-1", shares.RevokeAccessOpts{}, shares.WithRevokeAccessField("vendor", map[string]bool{"a": false}))
		}},
	}
}

func TestManilaShareActionsSendExactBodiesAndAcceptDocumentedCodes(t *testing.T) {
	for _, tc := range manilaShareActionCases() {
		for _, status := range []int{200, 201, 202, 204} {
			accepted := false
			for _, code := range tc.accept {
				accepted = accepted || code == status
			}
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("POST /v2/shares/share-1/action", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				manilaShareVersion(t, r, "2.82")
				if got := manilaShareRead(t, r); got != tc.body {
					t.Errorf("%s body=%s want %s", tc.name, got, tc.body)
				}
				w.WriteHeader(status)
			})
			err := tc.call(newManilaShares(cloud, "2.82"))
			if accepted != (err == nil) || calls.Load() != 1 {
				t.Fatalf("%s status %d error=%v calls=%d", tc.name, status, err, calls.Load())
			}
			if !accepted {
				manilaShareCode(t, err, status)
			}
		}
	}
}

func TestManilaShareActionsDoNotValidateMicroversionLocally(t *testing.T) {
	for _, tc := range manilaShareActionCases() {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /v2/shares/share-1/action", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			manilaShareVersion(t, r, "")
			w.WriteHeader(202)
		})
		if err := tc.call(newManilaShares(cloud, "")); err != nil || calls.Load() != 1 {
			t.Fatalf("%s error=%v calls=%d", tc.name, err, calls.Load())
		}
	}
}

func TestManilaShareActionServerErrorsKeepNativeResponseAndAreNotReplayed(t *testing.T) {
	for _, tc := range manilaShareActionCases() {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /v2/shares/share-1/action", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			testcloud.JSON(w, 404, `{"itemNotFound":{"code":404}}`)
		})
		manilaShareCode(t, tc.call(newManilaShares(cloud, "2.82")), 404)
		if calls.Load() != 1 {
			t.Fatalf("%s calls=%d", tc.name, calls.Load())
		}
	}
}
