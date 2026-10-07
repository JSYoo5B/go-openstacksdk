package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/JSYoo5B/gophercloudsdk/sharedfilesystems/v2/shareaccessrules"
)

func TestManilaAccessIdentityCanonicalFieldsAndRawAliases(t *testing.T) {
	for _, rule := range []string{
		`{"id":"rule-id","ID":"alias-id","share_id":"share-id","SHARE_ID":"foreign","state":"active","STATE":"error"}`,
		`{"id":"rule-id","ID":42,"share_id":"share-id","SHARE_ID":false,"state":"active","STATE":"error"}`,
	} {
		t.Run(rule, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, lists, posts atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				checkShareAccessHeaders(t, r, "2.82")
				w.Header().Set("X-Request-Id", "canonical-identity")
				switch {
				case r.Method == http.MethodGet && r.URL.Path == shareAccessBase+"/share-access-rules/rule-id":
					gets.Add(1)
					testcloud.JSON(w, 200, `{"access":`+rule+`}`)
				case r.Method == http.MethodGet && r.URL.Path == shareAccessBase+"/share-access-rules" && r.URL.Query().Get("share_id") == "share-id":
					lists.Add(1)
					testcloud.JSON(w, 200, `{"access_list":[`+rule+`]}`)
				case r.Method == http.MethodPost && r.URL.Path == shareAccessBase+"/shares/share-id/action":
					posts.Add(1)
					shareAccessBody(t, r, "allow_access")
					testcloud.JSON(w, 200, `{"access":`+rule+`}`)
				default:
					t.Errorf("unexpected request %s %s", r.Method, r.URL)
					testcloud.JSON(w, 500, `{}`)
				}
			})
			scope := shareAccessScope(t, cloud)
			check := func(value *shareaccessrules.AccessRule, err error) {
				t.Helper()
				if err != nil || value == nil || value.ID != "rule-id" || value.ShareID != "share-id" || value.ParentShareID != "share-id" || value.State != "error" || value.Header.Get("X-Request-Id") != "canonical-identity" || len(value.Body["ID"]) == 0 || len(value.Body["SHARE_ID"]) == 0 {
					t.Fatalf("canonical identity/raw aliases/native field order lost: value=%+v err=%v", value, err)
				}
			}
			value, err := scope.Get(context.Background(), "rule-id")
			check(value, err)
			values, err := scope.All(context.Background(), shareaccessrules.WithListMaxItems(1))
			if err != nil || len(values) != 1 {
				t.Fatalf("list=%v err=%v", values, err)
			}
			check(values[0], nil)
			value, err = scope.Allow(context.Background(), shareaccessrules.AllowOpts{AccessType: "ip", AccessTo: "192.0.2.1"})
			check(value, err)
			if gets.Load() != 1 || lists.Load() != 1 || posts.Load() != 1 {
				t.Fatalf("unexpected repeat/follow: get=%d list=%d post=%d", gets.Load(), lists.Load(), posts.Load())
			}
		})
	}
}

func TestManilaAccessIdentityShadowCannotAuthorizeDeny(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, posts atomic.Int32
	rule := `{"id":"rule-id","ID":"spoof-id","share_id":"foreign","SHARE_ID":"share-id"}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts.Add(1)
			testcloud.JSON(w, 500, `{}`)
			return
		}
		gets.Add(1)
		if r.URL.Path == shareAccessBase+"/share-access-rules" {
			testcloud.JSON(w, 200, `{"access_list":[`+rule+`]}`)
		} else if r.URL.Path == shareAccessBase+"/share-access-rules/rule-id" {
			testcloud.JSON(w, 200, `{"access":`+rule+`}`)
		} else {
			t.Errorf("identity mismatch must not trigger parent/fallback GET: %s", r.URL)
			testcloud.JSON(w, 500, `{}`)
		}
	})
	scope := shareAccessScope(t, cloud)
	for _, run := range []func() error{
		func() error { _, err := scope.Get(context.Background(), "rule-id"); return err },
		func() error {
			_, err := scope.All(context.Background(), shareaccessrules.WithListMaxItems(1))
			return err
		},
		func() error {
			return scope.Deny(context.Background(), "rule-id", shareaccessrules.WithDenyIgnoreMissing(true))
		},
	} {
		err := run()
		var mismatch *shareaccessrules.ParentMismatchError
		if !errors.Is(err, shareaccessrules.ErrParentMismatch) || !errors.As(err, &mismatch) || mismatch.ShareID != "share-id" || mismatch.ActualShareID != "foreign" || mismatch.AccessID != "rule-id" {
			t.Fatalf("raw parent mismatch lost: err=%v mismatch=%+v", err, mismatch)
		}
	}
	if gets.Load() != 3 || posts.Load() != 0 {
		t.Fatalf("foreign rule requests get=%d mutation=%d", gets.Load(), posts.Load())
	}
}

func TestManilaAccessIdentityOmittedParentAndFixedMutationRoute(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == shareAccessBase+"/share-access-rules":
			testcloud.JSON(w, 200, `{"access_list":[{"id":"wire-id","ID":"alias-id","SHARE_ID":"foreign"}]}`)
		case r.Method == http.MethodGet && r.URL.Path == shareAccessBase+"/share-access-rules/route-id":
			testcloud.JSON(w, 200, `{"access":{"id":"wire-id","ID":"alias-id","share_id":"share-id"}}`)
		case r.Method == http.MethodPost && r.URL.Path == shareAccessBase+"/shares/share-id/action":
			body := shareAccessBody(t, r, "deny_access")
			if body["access_id"] != "route-id" || len(body) != 1 {
				t.Errorf("response identity widened mutation: %v", body)
			}
			w.WriteHeader(202)
		default:
			t.Errorf("response identity widened route: %s %s", r.Method, r.URL)
			testcloud.JSON(w, 500, `{}`)
		}
	})
	scope := shareAccessScope(t, cloud)
	values, err := scope.All(context.Background())
	if err != nil || len(values) != 1 || values[0].ID != "wire-id" || values[0].ShareID != "" || values[0].ParentShareID != "share-id" || len(values[0].Body["share_id"]) != 0 || string(values[0].Body["SHARE_ID"]) != `"foreign"` {
		t.Fatalf("omitted canonical parent was synthesized: values=%+v err=%v", values, err)
	}
	if err := scope.Deny(context.Background(), "route-id"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 {
		t.Fatalf("requests=%d", calls.Load())
	}
}

func TestManilaAccessIdentityRejectsMissingInvalidCanonicalIDs(t *testing.T) {
	for _, rule := range []string{
		`{"ID":"alias-id","share_id":"share-id"}`,
		`{"id":null,"ID":"alias-id","share_id":"share-id"}`,
		`{"id":"","ID":"alias-id","share_id":"share-id"}`,
		`{"id":42,"ID":"alias-id","share_id":"share-id"}`,
		`{"id":"bad/id","ID":"alias-id","share_id":"share-id"}`,
		`{"id":"bad%2fid","ID":"alias-id","share_id":"share-id"}`,
		`{"id":"rule-id","share_id":null,"SHARE_ID":"share-id"}`,
		`{"id":"rule-id","share_id":42,"SHARE_ID":"share-id"}`,
		`{"id":"rule-id","share_id":"bad/id","SHARE_ID":"share-id"}`,
	} {
		t.Run(rule, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			body := `{"access_list":[` + rule + `]}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Request-Id", "invalid-canonical-id")
				if r.URL.Path == shareAccessBase+"/share-access-rules" {
					testcloud.JSON(w, 200, body)
				} else {
					testcloud.JSON(w, 200, `{"access":`+rule+`}`)
				}
			})
			scope := shareAccessScope(t, cloud)
			if value, err := scope.Get(context.Background(), "route-id"); err == nil || value != nil {
				t.Fatalf("invalid canonical GET identity accepted: value=%+v err=%v", value, err)
			}
			values, err := scope.All(context.Background(), shareaccessrules.WithListMaxItems(1))
			var accepted *resource.ResponseError
			if err == nil || values != nil || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != body || accepted.Header.Get("X-Request-Id") != "invalid-canonical-id" || accepted.Cause == nil || calls.Load() != 2 {
				t.Fatalf("invalid list identity evidence lost: values=%v err=%v evidence=%+v calls=%d", values, err, accepted, calls.Load())
			}
			if err := scope.Deny(context.Background(), "route-id"); err == nil || calls.Load() != 3 {
				t.Fatalf("invalid identity must block POST: err=%v calls=%d", err, calls.Load())
			}
		})
	}
}

func TestManilaAccessIdentityKnownCreatedIDSurvivesDecodeFailure(t *testing.T) {
	for _, rule := range []string{
		`{"id":"created-id","ID":"alias-id","share_id":"share-id","created_at":"bad-date"}`,
		`{"id":"created-id","ID":42,"share_id":"share-id","lock_visibility":"bad-bool"}`,
		`{"id":"created-id","ID":"alias-id","share_id":false,"SHARE_ID":"share-id"}`,
	} {
		t.Run(rule, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(shareAccessBase+"/shares/share-id/action", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				shareAccessBody(t, r, "allow_access")
				w.Header().Set("X-Request-Id", "created-then-failed")
				testcloud.JSON(w, 200, `{"access":`+rule+`}`)
			})
			value, err := shareAccessScope(t, cloud).Allow(context.Background(), shareaccessrules.AllowOpts{AccessType: "ip", AccessTo: "192.0.2.1"})
			if err == nil || value == nil || value.ID != "created-id" || value.ParentShareID != "share-id" || value.Header.Get("X-Request-Id") != "created-then-failed" || string(value.Body["id"]) != `"created-id"` || len(value.Body["ID"]) == 0 || calls.Load() != 1 {
				t.Fatalf("known created identity lost or operation resent: value=%+v err=%v calls=%d", value, err, calls.Load())
			}
		})
	}
}
