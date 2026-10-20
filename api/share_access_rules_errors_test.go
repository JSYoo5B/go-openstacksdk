package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules"
	"github.com/gophercloud/gophercloud/v2"
)

func TestShareAccessScopeParentMismatchIsNeverIgnored(t *testing.T) {
	for _, actual := range []string{"other-share", ""} {
		t.Run(actual, func(t *testing.T) {
			cloud := testcloud.New(t)
			var posts atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					posts.Add(1)
				}
				testcloud.JSON(w, 200, `{"access":{"id":"rule-id","share_id":"`+actual+`","state":"active"}}`)
			})
			scope := shareAccessScope(t, cloud)
			ctx := context.Background()
			for _, run := range []func() error{
				func() error { _, err := scope.Get(ctx, "rule-id"); return err },
				func() error {
					_, err := scope.Find(ctx, resource.ID("rule-id"), resource.WithIgnoreMissing())
					return err
				},
				func() error { return scope.Delete(ctx, resource.ID("rule-id")) },
				func() error { return scope.Deny(ctx, "rule-id", shareaccessrules.WithDenyIgnoreMissing(true)) },
				func() error { _, err := scope.Wait(ctx, resource.ID("rule-id"), "active"); return err },
				func() error { return scope.WaitDeleted(ctx, resource.ID("rule-id")) },
			} {
				err := run()
				var mismatch *shareaccessrules.ParentMismatchError
				if !errors.Is(err, shareaccessrules.ErrParentMismatch) || !errors.As(err, &mismatch) || mismatch.ShareID != "share-id" || mismatch.AccessID != "rule-id" || mismatch.ActualShareID != actual || errors.Is(err, resource.ErrNotFound) {
					t.Errorf("parent mismatch was hidden or lost: %v", err)
				}
			}
			if posts.Load() != 0 {
				t.Fatalf("wrong share performed %d mutation requests", posts.Load())
			}
		})
	}
}

func TestShareAccessScopeMissingRulePoliciesRequireReadableParent(t *testing.T) {
	cloud := testcloud.New(t)
	var parentGets, mutations atomic.Int32
	cloud.Mux.HandleFunc(shareAccessBase+"/share-access-rules/rule-id", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{"itemNotFound":{"message":"rule missing"}}`)
	})
	cloud.Mux.HandleFunc(shareAccessBase+"/shares/share-id", func(w http.ResponseWriter, r *http.Request) {
		parentGets.Add(1)
		testcloud.JSON(w, 200, `{"share":{"id":"share-id"}}`)
	})
	cloud.Mux.HandleFunc(shareAccessBase+"/shares/share-id/action", func(w http.ResponseWriter, r *http.Request) {
		mutations.Add(1)
		w.WriteHeader(202)
	})
	scope := shareAccessScope(t, cloud)
	ctx := context.Background()
	if value, err := scope.Find(ctx, resource.ID("rule-id"), resource.WithIgnoreMissing()); value != nil || err != nil {
		t.Fatalf("ignored Find=%v %v", value, err)
	}
	for _, run := range []func() error{
		func() error { return scope.Deny(ctx, "rule-id") },
		func() error {
			return scope.Deny(ctx, "rule-id", shareaccessrules.WithDenyIgnoreMissing(false), shareaccessrules.WithDenyIgnoreMissing(true))
		},
		func() error { return scope.Delete(ctx, resource.ID("rule-id")) },
		func() error { return scope.WaitDeleted(ctx, resource.ID("rule-id")) },
	} {
		if err := run(); err != nil {
			t.Errorf("missing rule in an existing share should succeed: %v", err)
		}
	}
	for _, run := range []func() error{
		func() error { _, err := scope.Get(ctx, "rule-id"); return err },
		func() error { _, err := scope.Find(ctx, resource.ID("rule-id")); return err },
		func() error {
			return scope.Deny(ctx, "rule-id", shareaccessrules.WithDenyIgnoreMissing(true), shareaccessrules.WithDenyIgnoreMissing(false))
		},
		func() error { return scope.Delete(ctx, resource.ID("rule-id"), resource.WithMissingError()) },
		func() error { _, err := scope.Wait(ctx, resource.ID("rule-id"), "active"); return err },
	} {
		if err := run(); !errors.Is(err, resource.ErrNotFound) || !gophercloud.ResponseCodeIs(err, 404) || errors.Is(err, shareaccessrules.ErrParentUnavailable) {
			t.Errorf("strict missing rule must retain original 404: %v", err)
		}
	}
	if parentGets.Load() != 10 || mutations.Load() != 0 {
		t.Fatalf("parent checks=%d mutations=%d", parentGets.Load(), mutations.Load())
	}
}

func TestShareAccessScopeNeverHidesMissingOrForbiddenParent(t *testing.T) {
	for _, tc := range []struct {
		name string
		code int
		body string
	}{
		{"missing", 404, `{"itemNotFound":{"message":"share missing"}}`},
		{"forbidden", 403, `{"forbidden":{"message":"share inaccessible"}}`},
		{"broken response", 200, `{"share":{"id":"other-share"}}`},
		{"null response", 200, `{"share":null}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc(shareAccessBase+"/share-access-rules/rule-id", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 404, `{"itemNotFound":{"message":"rule or parent missing"}}`)
			})
			cloud.Mux.HandleFunc(shareAccessBase+"/shares/share-id", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, tc.code, tc.body)
			})
			scope := shareAccessScope(t, cloud)
			ctx := context.Background()
			for _, run := range []func() error{
				func() error { _, err := scope.Get(ctx, "rule-id"); return err },
				func() error {
					_, err := scope.Find(ctx, resource.ID("rule-id"), resource.WithIgnoreMissing())
					return err
				},
				func() error { return scope.Deny(ctx, "rule-id") },
				func() error { return scope.Delete(ctx, resource.ID("rule-id")) },
				func() error { _, err := scope.Wait(ctx, resource.ID("rule-id"), "active"); return err },
				func() error { return scope.WaitDeleted(ctx, resource.ID("rule-id")) },
			} {
				err := run()
				var parent *shareaccessrules.ParentError
				if !errors.Is(err, shareaccessrules.ErrParentUnavailable) || !errors.As(err, &parent) || parent.ShareID != "share-id" || errors.Is(err, resource.ErrNotFound) {
					t.Errorf("parent failure was hidden or misclassified: %v", err)
				}
				if tc.code != 200 && !gophercloud.ResponseCodeIs(err, tc.code) {
					t.Errorf("parent HTTP %d cause lost: %v", tc.code, err)
				}
			}
		})
	}
}

func TestShareAccessScopeDenyMissingAfterGetDoesNotHideParentFailure(t *testing.T) {
	for _, parentCode := range []int{200, 403, 404} {
		t.Run(http.StatusText(parentCode), func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc(shareAccessBase+"/share-access-rules/rule-id", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"access":`+shareAccessValue+`}`)
			})
			cloud.Mux.HandleFunc(shareAccessBase+"/shares/share-id/action", func(w http.ResponseWriter, r *http.Request) {
				shareAccessBody(t, r, "deny_access")
				testcloud.JSON(w, 404, `{"itemNotFound":{"message":"gone between lookup and action"}}`)
			})
			cloud.Mux.HandleFunc(shareAccessBase+"/shares/share-id", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, parentCode, `{"share":{"id":"share-id"}}`)
			})
			scope := shareAccessScope(t, cloud)
			ctx := context.Background()
			for _, run := range []func() error{
				func() error { return scope.Deny(ctx, "rule-id") },
				func() error { return scope.Delete(ctx, resource.ID("rule-id")) },
			} {
				err := run()
				if parentCode == 200 && err != nil {
					t.Fatalf("verified missing rule should be ignored: %v", err)
				}
				if parentCode != 200 && (!errors.Is(err, shareaccessrules.ErrParentUnavailable) || !gophercloud.ResponseCodeIs(err, parentCode)) {
					t.Fatalf("post-action parent error was hidden: %v", err)
				}
			}
			if parentCode == 200 {
				if err := scope.Deny(ctx, "rule-id", shareaccessrules.WithDenyIgnoreMissing(false)); !errors.Is(err, resource.ErrNotFound) || !gophercloud.ResponseCodeIs(err, 404) {
					t.Fatalf("strict post-action 404=%v", err)
				}
			}
		})
	}
}

func TestShareAccessScopePreservesHTTPFailuresAndNativeSuccessCodes(t *testing.T) {
	for _, status := range []int{401, 403, 409, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Request-Id", "failed-request")
				testcloud.JSON(w, status, `{"error":{"message":"denied"}}`)
			})
			scope := shareAccessScope(t, cloud)
			ctx := context.Background()
			for _, run := range []func() error{
				func() error { _, err := scope.Get(ctx, "rule-id"); return err },
				func() error { _, err := scope.All(ctx); return err },
				func() error {
					_, err := scope.Allow(ctx, shareaccessrules.AllowOpts{AccessType: "ip", AccessTo: "192.0.2.1"})
					return err
				},
				func() error { return scope.Deny(ctx, "rule-id") },
				func() error { return scope.Delete(ctx, resource.ID("rule-id")) },
				func() error { return scope.WaitDeleted(ctx, resource.ID("rule-id")) },
			} {
				err := run()
				var response gophercloud.ErrUnexpectedResponseCode
				if !gophercloud.ResponseCodeIs(err, status) || !errors.As(err, &response) || response.ResponseHeader.Get("X-Request-Id") != "failed-request" || len(response.Body) == 0 || errors.Is(err, resource.ErrNotFound) {
					t.Errorf("original HTTP error lost: %v", err)
				}
			}
		})
	}
	for _, status := range []int{200, 201, 202, 204, 403} {
		t.Run("deny "+http.StatusText(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc(shareAccessBase+"/share-access-rules/rule-id", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"access":`+shareAccessValue+`}`)
			})
			cloud.Mux.HandleFunc(shareAccessBase+"/shares/share-id/action", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) })
			err := shareAccessScope(t, cloud).Deny(context.Background(), "rule-id")
			if status == 200 || status == 202 {
				if err != nil {
					t.Fatal(err)
				}
			} else if !gophercloud.ResponseCodeIs(err, status) {
				t.Fatalf("unexpected status %d accepted: %v", status, err)
			}
		})
	}
	for _, tc := range []struct {
		name string
		code int
		body string
	}{
		{"wrong grant status", 201, `{"access":` + shareAccessValue + `}`},
		{"missing access", 200, `{}`},
		{"null access", 200, `{"access":null}`},
		{"missing id", 200, `{"access":{"share_id":"share-id"}}`},
		{"invalid access type", 200, `{"access":42}`},
		{"invalid locks", 200, `{"access":{"id":"created-id","lock_visibility":"false"}}`},
		{"wrong parent", 200, `{"access":{"id":"created-id","share_id":"other"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, tc.code, tc.body) })
			scope := shareAccessScope(t, cloud)
			value, err := scope.Allow(context.Background(), shareaccessrules.AllowOpts{AccessType: "ip", AccessTo: "192.0.2.1"})
			if err == nil {
				t.Fatalf("invalid response accepted: %+v", value)
			}
			if tc.name == "wrong parent" && (value == nil || value.ID != "created-id" || !errors.Is(err, shareaccessrules.ErrParentMismatch)) {
				t.Fatalf("created resource lost on parent error: %v %v", value, err)
			}
			if tc.name == "invalid locks" && (value == nil || value.ID != "created-id") {
				t.Fatalf("created resource lost on lock decode error: %v %v", value, err)
			}
			if got, err := scope.Get(context.Background(), "rule-id"); got != nil || err == nil {
				t.Fatalf("invalid get returned usable value: %v %v", got, err)
			}
		})
	}
}

func TestShareAccessScopeWaitFailureTimeoutAndInFlightCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	var state atomic.Value
	state.Store("error")
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"access":{"id":"rule-id","share_id":"share-id","state":"`+state.Load().(string)+`"}}`)
	})
	scope := shareAccessScope(t, cloud)
	if _, err := scope.Wait(context.Background(), resource.ID("rule-id"), "active"); !errors.Is(err, resource.ErrFailedState) {
		t.Fatalf("failed rule state=%v", err)
	}
	state.Store("applying")
	if _, err := scope.Wait(context.Background(), resource.ID("rule-id"), "active", resource.WithTimeout(10*time.Millisecond), resource.WithPollInterval(time.Millisecond)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("wait timeout=%v", err)
	}
	if err := scope.WaitDeleted(context.Background(), resource.ID("rule-id"), resource.WithTimeout(10*time.Millisecond), resource.WithPollInterval(time.Millisecond)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deletion timeout=%v", err)
	}
	blocking := testcloud.New(t)
	started := make(chan struct{})
	blocking.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := shareAccessScope(t, blocking).Get(ctx, "rule-id"); done <- err }()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("in-flight cancellation=%v", err)
	}
}

func TestShareAccessScopeParentNameReportsMissingAndAmbiguous(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want error
	}{
		{"missing", `{"shares":[]}`, resource.ErrNotFound},
		{"ambiguous", `{"shares":[{"id":"a","name":"team"},{"id":"b","name":"team"}]}`, resource.ErrAmbiguous},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != shareAccessBase+"/shares/detail" || r.URL.Query().Get("name") != "team" {
					t.Errorf("parent lookup=%s", r.URL)
				}
				testcloud.JSON(w, 200, tc.body)
			})
			if scope, err := shareaccessrules.New(shareAccessClient(cloud, "2.45")).InShare(context.Background(), resource.Name("team")); scope != nil || !errors.Is(err, tc.want) || calls.Load() != 1 {
				t.Fatalf("scope=%v err=%v calls=%d", scope, err, calls.Load())
			}
		})
	}
}
