package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

func heatScope(t *testing.T, api *stacks.API, name, id string) *stacks.StackScope {
	t.Helper()
	scope, err := api.ForStack(stacks.StackIdentity{Name: name, ID: id})
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func TestHeatStackScopeNameLookupUsesAllPagesAndDistinguishesDetails(t *testing.T) {
	cloud := testcloud.New(t)
	api := stacks.New(cloud.Client("orchestration", "/heat"))
	var lists, gets atomic.Int32
	cloud.Mux.HandleFunc("GET /heat/stacks", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if r.URL.Query().Get("name") != "" {
			t.Errorf("name lookup must use exact local matching: %s", r.URL)
		}
		if r.URL.Query().Get("marker") == "" {
			testcloud.JSON(w, 200, `{"stacks":[{"stack_name":"app-copy","id":"other"}],"links":[{"rel":"next","href":"?marker=next"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"stacks":[{"stack_name":"app","id":"fixed","stack_status":"CREATE_COMPLETE","tags":["ops"],"creation_time":"2026-10-01T01:02:03Z"}]}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed", func(w http.ResponseWriter, r *http.Request) {
		if gets.Add(1) == 1 && r.URL.RawQuery != "resolve_outputs=False" || gets.Load() == 2 && r.URL.RawQuery != "" {
			t.Errorf("resolve_outputs policy lost: %s", r.URL)
		}
		testcloud.JSON(w, 200, `{"stack":{"stack_name":"app","id":"fixed","stack_status":"CREATE_COMPLETE","disable_rollback":false,"parameters":{"size":"large"},"outputs":[{"output_key":"endpoint","output_value":{"port":8080}}],"template_description":"template","timeout_mins":30}}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected identity lookup: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	summary, err := api.Resources().Find(context.Background(), resource.Name("app"))
	if err != nil || summary == nil || summary.Detailed || summary.ID != "fixed" || len(summary.Tags) != 1 || summary.CreationTime.IsZero() {
		t.Fatalf("summary=%v err=%v", summary, err)
	}
	scope, err := api.InStack(context.Background(), resource.Name("app"))
	if err != nil {
		t.Fatal(err)
	}
	copy := scope.Identity()
	copy.ID = "changed"
	if scope.Identity().ID != "fixed" || lists.Load() != 4 || gets.Load() != 0 {
		t.Fatalf("scope did not fix list identity: identity=%v lists=%d gets=%d", scope.Identity(), lists.Load(), gets.Load())
	}
	for _, options := range [][]stacks.GetOption{{stacks.WithResolveOutputs(false)}, nil} {
		value, err := scope.Get(context.Background(), options...)
		if err != nil || value == nil || !value.Detailed || value.ID != "fixed" || value.Parameters["size"] != "large" || len(value.Outputs) != 1 || value.Timeout != 30 || value.DisableRollback {
			t.Fatalf("detail=%v err=%v", value, err)
		}
	}
}

func TestHeatStackExactDuplicateNamesAcrossPagesAndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	api := stacks.New(cloud.Client("orchestration", "/heat"))
	var duplicate atomic.Bool
	duplicate.Store(true)
	cloud.Mux.HandleFunc("GET /heat/stacks", func(w http.ResponseWriter, r *http.Request) {
		if !duplicate.Load() {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.URL.Query().Get("marker") == "" {
			testcloud.JSON(w, 200, `{"stacks":[{"stack_name":"app","id":"first"}],"links":[{"rel":"next","href":"`+cloud.Server.URL+`/heat/stacks?marker=next"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"stacks":[{"stack_name":"app","id":"second"}]}`)
	})
	if _, err := api.InStack(context.Background(), resource.Name("app")); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatal(err)
	}
	duplicate.Store(false)
	if _, err := api.Resources().Find(context.Background(), resource.Name("app")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	value, err := api.Resources().Find(context.Background(), resource.Name("app"), resource.WithIgnoreMissing())
	if err != nil || value != nil {
		t.Fatalf("ignored missing=%v err=%v", value, err)
	}
	if err := api.Resources().Delete(context.Background(), resource.Name("app")); err != nil {
		t.Fatal(err)
	}
	if err := api.Resources().Delete(context.Background(), resource.Name("app"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	all, err := api.Resources().All(context.Background())
	if err != nil || all == nil || len(all) != 0 {
		t.Fatalf("empty list=%v err=%v", all, err)
	}
}

func TestHeatStackIDRedirectPreservesCanonicalIDQueryAndHeaders(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("orchestration", "/heat")
	client.MoreHeaders = map[string]string{"X-Policy": "keep"}
	api := stacks.New(client)
	cloud.Mux.HandleFunc("GET /heat/stacks/fixed", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "resolve_outputs=False" {
			t.Errorf("identity query=%s", r.URL.RawQuery)
		}
		http.Redirect(w, r, "/heat/stacks/canonical/fixed?resolve_outputs=False", http.StatusFound)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/canonical/fixed", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "resolve_outputs=False" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Policy") != "keep" {
			t.Errorf("redirect changed query/header: %s headers=%v", r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"stack":{"stack_name":"canonical","id":"fixed","stack_status":"CREATE_COMPLETE"}}`)
	})
	value, err := api.Resources().Get(context.Background(), "fixed", stacks.WithResolveOutputs(false))
	if err != nil || value == nil || value.ID != "fixed" || value.Name != "canonical" || !value.Detailed {
		t.Fatalf("redirect result=%v err=%v", value, err)
	}
	identity, err := value.Identity()
	if err != nil || identity != (stacks.StackIdentity{Name: "canonical", ID: "fixed"}) {
		t.Fatalf("identity=%v err=%v", identity, err)
	}
}

func TestHeatStackIdentityMismatchesAndHTTPErrorsArePreserved(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"name mismatch", `{"stack":{"stack_name":"other","id":"fixed"}}`, 200},
		{"ID mismatch", `{"stack":{"stack_name":"app","id":"other"}}`, 200},
		{"missing object", `{"stack":null}`, 200},
		{"incomplete identity", `{"stack":{"stack_name":"app"}}`, 200},
		{"invalid timestamp", `{"stack":{"stack_name":"app","id":"fixed","creation_time":"invalid"}}`, 200},
		{"forbidden", `{"message":"denied"}`, 403},
		{"missing", `{"message":"missing"}`, 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			api := stacks.New(cloud.Client("orchestration", "/heat"))
			cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, tc.status, tc.body) })
			value, err := heatScope(t, api, "app", "fixed").Get(context.Background())
			if err == nil || value != nil {
				t.Fatalf("mismatched/malformed result=%v err=%v", value, err)
			}
			if tc.status >= 400 {
				var responseError gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &responseError) || responseError.Actual != tc.status {
					t.Fatalf("HTTP cause lost: %v", err)
				}
			}
			if tc.status == 404 && !errors.Is(err, resource.ErrNotFound) {
				t.Fatal(err)
			}
		})
	}
	cloud := testcloud.New(t)
	api := stacks.New(cloud.Client("orchestration", "/heat"))
	cloud.Mux.HandleFunc("GET /heat/stacks/fixed", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"stack":{"stack_name":"app","id":"other"}}`)
	})
	if value, err := api.Resources().Get(context.Background(), "fixed"); err == nil || value != nil {
		t.Fatalf("ID lookup must reject canonical ID replacement: value=%v err=%v", value, err)
	}
}

func TestHeatStackScopeWaitUsesFixedPairAndDetectsFailedStates(t *testing.T) {
	for _, finalStatus := range []string{"CREATE_COMPLETE", "CREATE_FAILED", "UPDATE_FAILED", "ROLLBACK_FAILED"} {
		t.Run(finalStatus, func(t *testing.T) {
			cloud := testcloud.New(t)
			api := stacks.New(cloud.Client("orchestration", "/heat"))
			var gets atomic.Int32
			cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed", func(w http.ResponseWriter, r *http.Request) {
				status := "CREATE_IN_PROGRESS"
				if gets.Add(1) == 2 {
					status = finalStatus
				}
				testcloud.JSON(w, 200, `{"stack":{"stack_name":"app","id":"fixed","stack_status":"`+status+`"}}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("wait must keep name and ID: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected request", 500)
			})
			value, err := heatScope(t, api, "app", "fixed").Wait(context.Background(), "CREATE_COMPLETE", resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second))
			if finalStatus == "CREATE_COMPLETE" {
				if err != nil || value == nil || value.ID != "fixed" || !value.Detailed {
					t.Fatalf("wait=%v err=%v", value, err)
				}
			} else {
				var failed *resource.FailedStateError
				if !errors.As(err, &failed) || failed.ID != "fixed" || failed.Status != finalStatus || !errors.Is(err, resource.ErrFailedState) {
					t.Fatalf("failure state lost: %v", err)
				}
			}
			if gets.Load() != 2 {
				t.Fatalf("unexpected extra poll: %d", gets.Load())
			}
		})
	}
}

func TestHeatStackScopeDeletionWaitRecognizesCompletionFailureAndHTTP(t *testing.T) {
	for _, tc := range []struct {
		name, status string
		code         int
		want         error
	}{
		{"HTTP disappearance", "", 404, nil},
		{"DELETE_COMPLETE", "DELETE_COMPLETE", 200, nil},
		{"DELETE_FAILED", "DELETE_FAILED", 200, resource.ErrFailedState},
		{"forbidden", "", 403, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			api := stacks.New(cloud.Client("orchestration", "/heat"))
			cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, tc.code, `{"stack":{"stack_name":"app","id":"fixed","stack_status":"`+tc.status+`"}}`)
			})
			err := heatScope(t, api, "app", "fixed").WaitDeleted(context.Background(), resource.WithTimeout(time.Second))
			if tc.code == 403 {
				var responseError gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &responseError) || responseError.Actual != 403 {
					t.Fatalf("HTTP cause lost: %v", err)
				}
			} else if tc.want != nil && !errors.Is(err, tc.want) || tc.want == nil && err != nil {
				t.Fatalf("wait error=%v want=%v", err, tc.want)
			}
		})
	}
}

func TestHeatStackScopeMutationsUseFixedIdentityAndMissingPolicy(t *testing.T) {
	cloud := testcloud.New(t)
	api := stacks.New(cloud.Client("orchestration", "/heat"))
	var requests atomic.Int32
	cloud.Mux.HandleFunc("PATCH /heat/stacks/app/fixed", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		want := map[string]any{"parameters": map[string]any{"size": "large"}, "tags": "ops,test", "vendor_hint": false}
		if !reflect.DeepEqual(body, want) {
			t.Errorf("patch=%#v want=%#v", body, want)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	cloud.Mux.HandleFunc("DELETE /heat/stacks/app/fixed", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		testcloud.JSON(w, 404, `{"message":"missing"}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("known identity must bypass lookups: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	scope := heatScope(t, api, "app", "fixed")
	if requests.Load() != 0 {
		t.Fatal("ForStack must not request")
	}
	if err := scope.UpdatePatch(context.Background(), stacks.UpdateOpts{Parameters: map[string]any{"size": "large"}, Tags: []string{"ops", "test"}}, stacks.WithUpdatePatchField("vendor_hint", false)); err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(context.Background(), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if requests.Load() != 3 {
		t.Fatalf("requests=%d", requests.Load())
	}
}

func TestHeatStackSDKPagerMarkerFallbackCycleAndBreak(t *testing.T) {
	cloud := testcloud.New(t)
	api := stacks.New(cloud.Client("orchestration", "/heat"))
	var requests atomic.Int32
	var repeat atomic.Bool
	cloud.Mux.HandleFunc("GET /heat/stacks", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Query().Get("limit") != "1" {
			t.Errorf("page size lost: %s", r.URL)
		}
		switch r.URL.Query().Get("marker") {
		case "":
			testcloud.JSON(w, 200, `{"stacks":[{"stack_name":"first","id":"first"}]}`)
		case "first":
			if repeat.Load() {
				testcloud.JSON(w, 200, `{"stacks":[{"stack_name":"first","id":"first"}]}`)
				return
			}
			testcloud.JSON(w, 200, `{"stacks":[{"stack_name":"second","id":"second"}]}`)
		case "second":
			testcloud.JSON(w, 200, `{"stacks":[]}`)
		default:
			t.Errorf("unexpected marker: %s", r.URL)
		}
	})
	all, err := api.Resources().All(context.Background(), resource.WithPageSize(1))
	if err != nil || len(all) != 2 || requests.Load() != 3 {
		t.Fatalf("pages=%v requests=%d err=%v", all, requests.Load(), err)
	}
	requests.Store(0)
	for _, err := range api.Resources().List(context.Background(), resource.WithPageSize(1)) {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if requests.Load() != 1 {
		t.Fatalf("break fetched additional pages: %d", requests.Load())
	}
	requests.Store(0)
	repeat.Store(true)
	if _, err := api.Resources().All(context.Background(), resource.WithPageSize(1)); !errors.Is(err, resource.ErrPaginationCycle) || requests.Load() != 2 {
		t.Fatalf("unchanged marker must stop: requests=%d err=%v", requests.Load(), err)
	}
}

func TestHeatStackScopeValidationAndCancellationBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	api := stacks.New(cloud.Client("orchestration", "/heat"))
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("invalid/canceled operation must not request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	for _, identity := range []stacks.StackIdentity{{}, {Name: "app"}, {Name: "../app", ID: "fixed"}, {Name: "app", ID: "fixed/other"}, {Name: "app", ID: "fixed\x00"}} {
		if _, err := api.ForStack(identity); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	scope := heatScope(t, api, "app", "fixed")
	if _, err := scope.Get(context.Background(), nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, id := range []string{"../fixed", "fixed\x00"} {
		if _, err := api.Resources().Get(context.Background(), id); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := scope.Wait(context.Background(), ""); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.InStack(ctx, resource.Name("app")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := scope.Get(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := scope.UpdatePatch(ctx, stacks.UpdateOpts{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := scope.WaitDeleted(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestHeatStackSDKPagerRepeatedLinkUsesCommonGuard(t *testing.T) {
	cloud := testcloud.New(t)
	api := stacks.New(cloud.Client("orchestration", "/heat"))
	var requests atomic.Int32
	cloud.Mux.HandleFunc("GET /heat/stacks", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		testcloud.JSON(w, 200, `{"stacks":[{"stack_name":"app","id":"fixed"}],"links":[{"rel":"next","href":"?limit=1"}]}`)
	})
	if _, err := api.Resources().All(context.Background(), resource.WithPageSize(1)); !errors.Is(err, resource.ErrPaginationCycle) || requests.Load() != 1 {
		t.Fatalf("repeat next link must not be requested twice: requests=%d err=%v", requests.Load(), err)
	}
	requests.Store(0)
	for _, err := range api.Resources().List(context.Background(), resource.WithPageSize(1)) {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if requests.Load() != 1 {
		t.Fatalf("break must avoid next-link processing: requests=%d", requests.Load())
	}
}
