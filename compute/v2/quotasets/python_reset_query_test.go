package quotasets_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonQuotaResetCall struct{ method, path, query, version string }

func pythonQuotaResetScope(t *testing.T, status *int) (*quotasets.ProjectQuotaScope, *[]pythonQuotaResetCall) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := &[]pythonQuotaResetCall{}
	cloud.Mux.HandleFunc("/compute/v2.1/", func(w http.ResponseWriter, r *http.Request) {
		*calls = append(*calls, pythonQuotaResetCall{r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("X-OpenStack-Nova-API-Version")})
		w.Header().Set("X-Reset", "proof")
		if *status == http.StatusAccepted || *status == http.StatusNoContent {
			w.WriteHeader(*status)
			return
		}
		testcloud.JSON(w, *status, `{"itemNotFound":{"message":"missing"}}`)
	})
	client := cloud.Client("compute", "/compute/v2.1/")
	client.Microversion = "2.56"
	scope, err := quotasets.New(client).InProject(context.Background(), resource.ID("project"))
	if err != nil {
		t.Fatal(err)
	}
	return scope, calls
}

func TestPythonQuotaResetForwardsQueryAndMissingPolicy(t *testing.T) {
	ctx := context.Background()
	status := http.StatusAccepted
	scope, calls := pythonQuotaResetScope(t, &status)
	// revert_quota_set(project, force="True", region="r1"); urlencode sends
	// str(value), so the literal text reaches the server.
	got, err := scope.Reset(ctx, quotasets.WithResetQuery("region", "old"), quotasets.WithResetQuery("force", "True"), quotasets.WithResetQuery("region", "r1"))
	if err != nil || got.ProjectID != "project" || got.UserID != "" || got.Header.Get("X-Reset") != "proof" {
		t.Fatal(got, err)
	}
	user, err := scope.InUser(ctx, resource.ID("user"))
	if err != nil {
		t.Fatal(err)
	}
	// revert_quota_set(project, user_id="user", force="True")
	status = http.StatusNoContent
	got, err = user.Reset(ctx, quotasets.WithResetQuery("force", "True"))
	if err != nil || got.ProjectID != "project" || got.UserID != "user" {
		t.Fatal(got, err)
	}
	// revert_quota_set(project) without query keeps the native bodyless DELETE.
	if _, err := scope.Reset(ctx); err != nil {
		t.Fatal(err)
	}
	// Python raises NotFoundException (ignore_missing=False); the Go default
	// keeps the 404 and WithResetIgnoreMissing(true) suppresses only it.
	status = http.StatusNotFound
	if _, err := scope.Reset(ctx, quotasets.WithResetQuery("force", "True")); !errors.Is(err, resource.ErrNotFound) || !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	if got, err := scope.Reset(ctx, quotasets.WithResetQuery("force", "True"), quotasets.WithResetIgnoreMissing(true)); got != nil || err != nil {
		t.Fatal(got, err)
	}
	status = http.StatusInternalServerError
	var operation *resource.OperationError
	if _, err := user.Reset(ctx, quotasets.WithResetQuery("force", "True"), quotasets.WithResetIgnoreMissing(true)); !gophercloud.ResponseCodeIs(err, status) || !errors.As(err, &operation) || operation.Operation != "Reset" {
		t.Fatal(err)
	}
	want := []pythonQuotaResetCall{
		{http.MethodDelete, "/compute/v2.1/os-quota-sets/project", "force=True&region=r1", "2.56"},
		{http.MethodDelete, "/compute/v2.1/os-quota-sets/project", "force=True&user_id=user", "2.56"},
		{http.MethodDelete, "/compute/v2.1/os-quota-sets/project", "", "2.56"},
		{http.MethodDelete, "/compute/v2.1/os-quota-sets/project", "force=True", "2.56"},
		{http.MethodDelete, "/compute/v2.1/os-quota-sets/project", "force=True", "2.56"},
		{http.MethodDelete, "/compute/v2.1/os-quota-sets/project", "force=True&user_id=user", "2.56"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonQuotaResetQueryRejectsScopeOwnedAndInvalidKeys(t *testing.T) {
	ctx := context.Background()
	status := http.StatusAccepted
	scope, calls := pythonQuotaResetScope(t, &status)
	user, err := scope.InUser(ctx, resource.ID("user"))
	if err != nil {
		t.Fatal(err)
	}
	for _, option := range []quotasets.ResetOption{
		quotasets.WithResetQuery("user_id", "other"),
		quotasets.WithResetQuery(" ", "v"),
		quotasets.WithResetQuery("bad\nkey", "v"),
		quotasets.WithResetQuery("key", "\xff"),
		nil,
	} {
		for _, reset := range []func() error{
			func() error { _, err := scope.Reset(ctx, option); return err },
			func() error { _, err := user.Reset(ctx, option); return err },
		} {
			var operation *resource.OperationError
			if err := reset(); !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "Reset" {
				t.Fatal(err)
			}
		}
	}
	if len(*calls) != 0 {
		t.Fatalf("%+v", *calls)
	}
}
