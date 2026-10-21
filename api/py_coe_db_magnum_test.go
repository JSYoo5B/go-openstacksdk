package api_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/clustertemplates"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func pyCOEDBTemplates(cloud *testcloud.Cloud) *clustertemplates.API {
	return clustertemplates.New(cloud.Client("container-infra", "/v1"))
}

func TestPyCOEDBFindClusterByIDUsesMemberGetAndByNameUsesDetailList(t *testing.T) {
	cloud := testcloud.New(t)
	var members, lists atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/clusters/{id}", func(w http.ResponseWriter, r *http.Request) {
		members.Add(1)
		if r.PathValue("id") == "detail" {
			lists.Add(1)
			testcloud.JSON(w, 200, `{"clusters":[{"uuid":"cl-1","name":"prod"},{"uuid":"cl-2","name":"dup"},{"uuid":"cl-3","name":"dup"}]}`)
			return
		}
		if r.PathValue("id") == "cl-1" {
			testcloud.JSON(w, 200, magnumClusterBody)
			return
		}
		testcloud.JSON(w, 404, `{}`)
	})
	api := newMagnumClusters(t, cloud, "")
	ctx := context.Background()
	value, err := api.Find(ctx, resource.ID("cl-1"))
	if err != nil || value == nil || value.UUID != "cl-1" || lists.Load() != 0 {
		t.Fatalf("by id: %+v err=%v lists=%d", value, err, lists.Load())
	}
	// An ID reference is never retried as a name: a missing ID is a lookup error.
	if _, err := api.Find(ctx, resource.ID("nope")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing id: %v", err)
	}
	if value, err := api.Find(ctx, resource.ID("nope"), resource.WithIgnoreMissing()); err != nil || value != nil {
		t.Fatalf("ignore missing id: %+v %v", value, err)
	}
	value, err = api.Find(ctx, resource.Name("prod"))
	if err != nil || value == nil || value.UUID != "cl-1" {
		t.Fatalf("by name: %+v %v", value, err)
	}
	var ambiguous *resource.AmbiguousError
	if _, err := api.Find(ctx, resource.Name("dup")); !errors.As(err, &ambiguous) || !errors.Is(err, resource.ErrAmbiguous) || !reflect.DeepEqual(ambiguous.IDs, []string{"cl-2", "cl-3"}) {
		t.Fatalf("duplicate names: %v", err)
	}
	if _, err := api.Find(ctx, resource.Name("none")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing name: %v", err)
	}
	if value, err := api.Find(ctx, resource.Name("none"), resource.WithIgnoreMissing()); err != nil || value != nil {
		t.Fatalf("ignore missing name: %+v %v", value, err)
	}
}

func TestPyCOEDBFindClusterTemplateByIDUsesMemberGetAndByNameUsesList(t *testing.T) {
	cloud := testcloud.New(t)
	var lists atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/clustertemplates/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "tpl-1" {
			testcloud.JSON(w, 200, magnumTemplateBody)
			return
		}
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v1/clustertemplates", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		testcloud.JSON(w, 200, `{"clustertemplates":[{"uuid":"tpl-1","name":"k8s"},{"uuid":"tpl-2","name":"dup"},{"uuid":"tpl-3","name":"dup"}]}`)
	})
	api := pyCOEDBTemplates(cloud)
	ctx := context.Background()
	if value, err := api.Find(ctx, resource.ID("tpl-1")); err != nil || value == nil || value.UUID != "tpl-1" || lists.Load() != 0 {
		t.Fatalf("by id: %+v %v lists=%d", value, err, lists.Load())
	}
	if _, err := api.Find(ctx, resource.ID("nope")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing id: %v", err)
	}
	if value, err := api.Find(ctx, resource.Name("k8s")); err != nil || value == nil || value.UUID != "tpl-1" {
		t.Fatalf("by name: %+v %v", value, err)
	}
	if _, err := api.Find(ctx, resource.Name("dup")); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatalf("duplicate names: %v", err)
	}
	if value, err := api.Find(ctx, resource.Name("none"), resource.WithIgnoreMissing()); err != nil || value != nil {
		t.Fatalf("ignore missing: %+v %v", value, err)
	}
	if _, err := api.Find(ctx, resource.Name("none")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing name: %v", err)
	}
}

func pyCOEDBClusterSequence(cloud *testcloud.Cloud, statuses ...string) *atomic.Int32 {
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/clusters/cl-1", func(w http.ResponseWriter, r *http.Request) {
		index := int(calls.Add(1)) - 1
		if index >= len(statuses) {
			index = len(statuses) - 1
		}
		if statuses[index] == "404" {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		testcloud.JSON(w, 200, `{"uuid":"cl-1","name":"prod","status":"`+statuses[index]+`"}`)
	})
	return &calls
}

func TestPyCOEDBWaitForClusterStatusComparesCaseInsensitivelyAndReportsZeroProgress(t *testing.T) {
	cloud := testcloud.New(t)
	calls := pyCOEDBClusterSequence(cloud, "CREATE_IN_PROGRESS", "CREATE_IN_PROGRESS", "CREATE_COMPLETE")
	var progress []int
	value, err := newMagnumClusters(t, cloud, "").WaitFor(context.Background(), resource.ID("cl-1"), "create_complete",
		resource.WithPollInterval(time.Millisecond), resource.WithProgressCallback(func(p int) { progress = append(progress, p) }))
	if err != nil || value == nil || value.Status != "CREATE_COMPLETE" {
		t.Fatalf("value=%+v err=%v", value, err)
	}
	if !reflect.DeepEqual(progress, []int{0, 0}) || calls.Load() != 3 {
		t.Fatalf("progress=%v calls=%d", progress, calls.Load())
	}
}

func TestPyCOEDBWaitForClusterFailureStatesDefaultAndPythonStyleOverride(t *testing.T) {
	cloud := testcloud.New(t)
	pyCOEDBClusterSequence(cloud, "CREATE_FAILED", "CREATE_COMPLETE")
	api := newMagnumClusters(t, cloud, "")
	// Go's service default treats Magnum *_FAILED states as failures at once.
	_, err := api.WaitFor(context.Background(), resource.ID("cl-1"), "CREATE_COMPLETE", resource.WithPollInterval(time.Millisecond))
	var failed *resource.FailedStateError
	if !errors.As(err, &failed) || failed.Status != "CREATE_FAILED" || !errors.Is(err, resource.ErrFailedState) {
		t.Fatalf("default failure: %v", err)
	}
	// The Python default failure list is exactly ["ERROR"]; selecting it keeps polling past CREATE_FAILED.
	cloud2 := testcloud.New(t)
	calls := pyCOEDBClusterSequence(cloud2, "CREATE_FAILED", "CREATE_COMPLETE")
	value, err := newMagnumClusters(t, cloud2, "").WaitFor(context.Background(), resource.ID("cl-1"), "CREATE_COMPLETE",
		resource.WithPollInterval(time.Millisecond), resource.WithFailureStates("ERROR"))
	if err != nil || value == nil || value.Status != "CREATE_COMPLETE" || calls.Load() != 2 {
		t.Fatalf("python-style failures: %+v err=%v calls=%d", value, err, calls.Load())
	}
	cloud3 := testcloud.New(t)
	pyCOEDBClusterSequence(cloud3, "ERROR")
	if _, err := newMagnumClusters(t, cloud3, "").WaitFor(context.Background(), resource.ID("cl-1"), "CREATE_COMPLETE",
		resource.WithPollInterval(time.Millisecond), resource.WithFailureStates("error")); !errors.Is(err, resource.ErrFailedState) {
		t.Fatalf("explicit ERROR failure is case-insensitive: %v", err)
	}
}

func TestPyCOEDBWaitForClusterTimeoutAndUnlimitedWaitKeepContextCause(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	pyCOEDBClusterSequence(cloud, "CREATE_IN_PROGRESS")
	_, err := newMagnumClusters(t, cloud, "").WaitFor(context.Background(), resource.ID("cl-1"), "CREATE_COMPLETE",
		resource.WithPollInterval(time.Millisecond), resource.WithTimeout(30*time.Millisecond))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout: %v", err)
	}
	// An unlimited wait is bounded only by the caller's context.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err = newMagnumClusters(t, cloud, "").WaitFor(ctx, resource.ID("cl-1"), "CREATE_COMPLETE",
		resource.WithPollInterval(time.Millisecond), resource.WithUnlimitedWait())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unlimited with context deadline: %v", err)
	}
}

func TestPyCOEDBWaitForClusterDeletionCompletesOnNotFoundOrDeletedStatus(t *testing.T) {
	cloud := testcloud.New(t)
	calls := pyCOEDBClusterSequence(cloud, "DELETE_IN_PROGRESS", "DELETE_IN_PROGRESS", "404")
	var progress []int
	if err := newMagnumClusters(t, cloud, "").WaitForDeletion(context.Background(), resource.ID("cl-1"),
		resource.WithPollInterval(time.Millisecond), resource.WithProgressCallback(func(p int) { progress = append(progress, p) })); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 || !reflect.DeepEqual(progress, []int{0, 0}) {
		t.Fatalf("calls=%d progress=%v", calls.Load(), progress)
	}
	cloud2 := testcloud.New(t)
	pyCOEDBClusterSequence(cloud2, "DELETE_IN_PROGRESS", "deleted")
	if err := newMagnumClusters(t, cloud2, "").WaitForDeletion(context.Background(), resource.ID("cl-1"), resource.WithPollInterval(time.Millisecond)); err != nil {
		t.Fatalf("deleted status: %v", err)
	}
	// An already missing cluster completes at the first observation, and no DELETE is ever sent.
	cloud3 := testcloud.New(t)
	var deletes atomic.Int32
	cloud3.Mux.HandleFunc("DELETE /v1/clusters/cl-1", func(w http.ResponseWriter, r *http.Request) { deletes.Add(1) })
	calls3 := pyCOEDBClusterSequence(cloud3, "404")
	if err := newMagnumClusters(t, cloud3, "").WaitForDeletion(context.Background(), resource.ID("cl-1")); err != nil || calls3.Load() != 1 || deletes.Load() != 0 {
		t.Fatalf("already missing: err=%v calls=%d deletes=%d", err, calls3.Load(), deletes.Load())
	}
}

func TestPyCOEDBWaitForClusterDeletionSurfacesNonNotFoundErrorsAndCancellation(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v1/clusters/cl-1", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 403, `{}`) })
	err := newMagnumClusters(t, cloud, "").WaitForDeletion(context.Background(), resource.ID("cl-1"), resource.WithPollInterval(time.Millisecond))
	if err == nil || errors.Is(err, resource.ErrNotFound) || magnumUnexpected(err) != 403 {
		t.Fatalf("forbidden must not complete the wait: %v", err)
	}
	cloud2 := testcloud.New(t)
	var calls atomic.Int32
	cloud2.Mux.HandleFunc("GET /v1/clusters/cl-1", func(w http.ResponseWriter, r *http.Request) { calls.Add(1) })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := newMagnumClusters(t, cloud2, "").WaitForDeletion(ctx, resource.ID("cl-1")); !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatalf("canceled: err=%v calls=%d", err, calls.Load())
	}
	// Template collections share the deletion waiter.
	cloud3 := testcloud.New(t)
	var tplCalls atomic.Int32
	cloud3.Mux.HandleFunc("GET /v1/clustertemplates/tpl-1", func(w http.ResponseWriter, r *http.Request) {
		if tplCalls.Add(1) == 1 {
			testcloud.JSON(w, 200, magnumTemplateBody)
			return
		}
		testcloud.JSON(w, 404, `{}`)
	})
	if err := pyCOEDBTemplates(cloud3).WaitForDeletion(context.Background(), resource.ID("tpl-1"), resource.WithPollInterval(time.Millisecond)); err != nil || tplCalls.Load() != 2 {
		t.Fatalf("template deletion: err=%v calls=%d", err, tplCalls.Load())
	}
}

func TestPyCOEDBClusterAndTemplateRemoveAreStrictOrIgnoreMissingByPolicy(t *testing.T) {
	cloud := testcloud.New(t)
	var deletes atomic.Int32
	cloud.Mux.HandleFunc("DELETE /v1/clusters/{id}", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("DELETE /v1/clustertemplates/{id}", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		testcloud.JSON(w, 404, `{}`)
	})
	ctx := context.Background()
	// Python delete_* defaults to ignore_missing=True and sends one DELETE.
	if err := newMagnumClusters(t, cloud, "").Remove(ctx, resource.ID("gone")); err != nil {
		t.Fatalf("cluster default: %v", err)
	}
	if err := pyCOEDBTemplates(cloud).Remove(ctx, resource.ID("gone")); err != nil {
		t.Fatalf("template default: %v", err)
	}
	if err := newMagnumClusters(t, cloud, "").Remove(ctx, resource.ID("gone"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("cluster strict: %v", err)
	}
	if err := pyCOEDBTemplates(cloud).Remove(ctx, resource.ID("gone"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("template strict: %v", err)
	}
	if deletes.Load() != 4 {
		t.Fatalf("each Remove sends exactly one DELETE, got %d", deletes.Load())
	}
}
