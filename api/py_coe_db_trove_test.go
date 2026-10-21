package api_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/db/v1/flavors"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func pyCOEDBInstanceSequence(cloud *testcloud.Cloud, statuses ...string) *atomic.Int32 {
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /trove/instances/inst-1", func(w http.ResponseWriter, r *http.Request) {
		index := int(calls.Add(1)) - 1
		if index >= len(statuses) {
			index = len(statuses) - 1
		}
		if statuses[index] == "404" {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		testcloud.JSON(w, 200, `{"instance":{"id":"inst-1","name":"db","status":"`+statuses[index]+`"}}`)
	})
	return &calls
}

func TestPyCOEDBInstanceFindByIDUsesMemberGetAndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	var lists atomic.Int32
	cloud.Mux.HandleFunc("GET /trove/instances/inst-1", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, troveInstanceBody) })
	cloud.Mux.HandleFunc("GET /trove/instances/gone", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /trove/instances", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		testcloud.JSON(w, 200, `{"instances":[{"id":"inst-1","name":"db"},{"id":"inst-2","name":"dup"},{"id":"inst-3","name":"dup"}]}`)
	})
	api := troveInstanceAPI(cloud)
	ctx := context.Background()
	if value, err := api.Find(ctx, resource.ID("inst-1")); err != nil || value == nil || value.ID != "inst-1" || lists.Load() != 0 {
		t.Fatalf("by id: %+v %v lists=%d", value, err, lists.Load())
	}
	if _, err := api.Find(ctx, resource.ID("gone")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing id: %v", err)
	}
	if value, err := api.Find(ctx, resource.ID("gone"), resource.WithIgnoreMissing()); err != nil || value != nil {
		t.Fatalf("ignore missing id: %+v %v", value, err)
	}
	if value, err := api.Find(ctx, resource.Name("db")); err != nil || value == nil || value.ID != "inst-1" {
		t.Fatalf("by name: %+v %v", value, err)
	}
	var ambiguous *resource.AmbiguousError
	if _, err := api.Find(ctx, resource.Name("dup")); !errors.As(err, &ambiguous) || !reflect.DeepEqual(ambiguous.IDs, []string{"inst-2", "inst-3"}) {
		t.Fatalf("duplicate names: %v", err)
	}
}

func TestPyCOEDBInstanceWaitForStatusMatchesCaseInsensitivelyWithZeroProgressAndFailureOverride(t *testing.T) {
	cloud := testcloud.New(t)
	calls := pyCOEDBInstanceSequence(cloud, "BUILD", "BUILD", "ACTIVE")
	var progress []int
	value, err := troveInstanceAPI(cloud).WaitFor(context.Background(), resource.ID("inst-1"), "active",
		resource.WithPollInterval(time.Millisecond), resource.WithProgressCallback(func(p int) { progress = append(progress, p) }))
	if err != nil || value == nil || value.Status != "ACTIVE" || calls.Load() != 3 || !reflect.DeepEqual(progress, []int{0, 0}) {
		t.Fatalf("value=%+v err=%v calls=%d progress=%v", value, err, calls.Load(), progress)
	}
	// Go's service default stops at ERROR and FAILED.
	cloud2 := testcloud.New(t)
	pyCOEDBInstanceSequence(cloud2, "BUILD", "FAILED")
	var failed *resource.FailedStateError
	if _, err := troveInstanceAPI(cloud2).WaitFor(context.Background(), resource.ID("inst-1"), "ACTIVE", resource.WithPollInterval(time.Millisecond)); !errors.As(err, &failed) || failed.Status != "FAILED" {
		t.Fatalf("default failure: %v", err)
	}
	// The Python default failure list is exactly ["ERROR"], so FAILED does not stop that wait.
	cloud3 := testcloud.New(t)
	calls3 := pyCOEDBInstanceSequence(cloud3, "FAILED", "ACTIVE")
	if value, err := troveInstanceAPI(cloud3).WaitFor(context.Background(), resource.ID("inst-1"), "ACTIVE",
		resource.WithPollInterval(time.Millisecond), resource.WithFailureStates("ERROR")); err != nil || value == nil || calls3.Load() != 2 {
		t.Fatalf("python-style failures: %+v %v calls=%d", value, err, calls3.Load())
	}
	cloud4 := testcloud.New(t)
	pyCOEDBInstanceSequence(cloud4, "ERROR")
	if _, err := troveInstanceAPI(cloud4).WaitFor(context.Background(), resource.ID("inst-1"), "ACTIVE",
		resource.WithPollInterval(time.Millisecond), resource.WithFailureStates("ERROR")); !errors.Is(err, resource.ErrFailedState) {
		t.Fatalf("ERROR failure: %v", err)
	}
}

func TestPyCOEDBInstanceWaitTimeoutAndUnlimitedKeepContextCause(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	pyCOEDBInstanceSequence(cloud, "BUILD")
	_, err := troveInstanceAPI(cloud).WaitFor(context.Background(), resource.ID("inst-1"), "ACTIVE",
		resource.WithPollInterval(time.Millisecond), resource.WithTimeout(30*time.Millisecond))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, err := troveInstanceAPI(cloud).WaitFor(ctx, resource.ID("inst-1"), "ACTIVE",
		resource.WithPollInterval(time.Millisecond), resource.WithUnlimitedWait()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unlimited with context deadline: %v", err)
	}
}

func TestPyCOEDBInstanceWaitForDeletionCompletesOnNotFoundAndNeverDeletes(t *testing.T) {
	cloud := testcloud.New(t)
	var deletes atomic.Int32
	cloud.Mux.HandleFunc("DELETE /trove/instances/inst-1", func(w http.ResponseWriter, r *http.Request) { deletes.Add(1) })
	calls := pyCOEDBInstanceSequence(cloud, "SHUTDOWN", "SHUTDOWN", "404")
	var progress []int
	if err := troveInstanceAPI(cloud).WaitForDeletion(context.Background(), resource.ID("inst-1"),
		resource.WithPollInterval(time.Millisecond), resource.WithProgressCallback(func(p int) { progress = append(progress, p) })); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 || deletes.Load() != 0 || !reflect.DeepEqual(progress, []int{0, 0}) {
		t.Fatalf("calls=%d deletes=%d progress=%v", calls.Load(), deletes.Load(), progress)
	}
	cloud2 := testcloud.New(t)
	pyCOEDBInstanceSequence(cloud2, "SHUTDOWN", "deleted")
	if err := troveInstanceAPI(cloud2).WaitForDeletion(context.Background(), resource.ID("inst-1"), resource.WithPollInterval(time.Millisecond)); err != nil {
		t.Fatalf("deleted status: %v", err)
	}
	cloud3 := testcloud.New(t)
	cloud3.Mux.HandleFunc("GET /trove/instances/inst-1", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 403, `{}`) })
	if err := troveInstanceAPI(cloud3).WaitForDeletion(context.Background(), resource.ID("inst-1"), resource.WithPollInterval(time.Millisecond)); err == nil || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("forbidden must not complete the wait: %v", err)
	}
	cloud4 := testcloud.New(t)
	var cancelled atomic.Int32
	cloud4.Mux.HandleFunc("GET /trove/instances/inst-1", func(w http.ResponseWriter, r *http.Request) { cancelled.Add(1) })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := troveInstanceAPI(cloud4).WaitForDeletion(ctx, resource.ID("inst-1")); !errors.Is(err, context.Canceled) || cancelled.Load() != 0 {
		t.Fatalf("canceled: err=%v calls=%d", err, cancelled.Load())
	}
}

func TestPyCOEDBFlavorFindByIDUsesMemberGetAndNamesAreExact(t *testing.T) {
	cloud := testcloud.New(t)
	var lists atomic.Int32
	cloud.Mux.HandleFunc("GET /trove/flavors/1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"flavor":{"id":1,"name":"m1.tiny","ram":0,"str_id":"1"}}`)
	})
	cloud.Mux.HandleFunc("GET /trove/flavors/9", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /trove/flavors", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		testcloud.JSON(w, 200, `{"flavors":[{"id":1,"name":"m1.tiny","ram":0},{"id":2,"name":"twin","ram":1},{"id":3,"name":"twin","ram":2}]}`)
	})
	api := flavors.New(cloud.Client("database", "/trove"))
	ctx := context.Background()
	if value, err := api.Find(ctx, resource.ID("1")); err != nil || value == nil || value.Name != "m1.tiny" || value.RAM != 0 || lists.Load() != 0 {
		t.Fatalf("by id: %+v %v lists=%d", value, err, lists.Load())
	}
	if _, err := api.Find(ctx, resource.ID("9")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing id: %v", err)
	}
	if value, err := api.Find(ctx, resource.ID("9"), resource.WithIgnoreMissing()); err != nil || value != nil {
		t.Fatalf("ignore missing: %+v %v", value, err)
	}
	if _, err := api.Find(ctx, resource.Name("twin")); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatalf("duplicate names: %v", err)
	}
	if value, err := api.Find(ctx, resource.Name("none"), resource.WithIgnoreMissing()); err != nil || value != nil {
		t.Fatalf("ignore missing name: %+v %v", value, err)
	}
}
