package migrations_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/migrations"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeMigrationTransport func(*http.Request) (*http.Response, error)

func (transport nativeMigrationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeMigrationWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeMigrationCall struct{ method, path, query string }

func nativeMigrationAPI(t *testing.T, calls *[]nativeMigrationCall, reply func(*http.Request) *http.Response) *migrations.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeMigrationTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, nativeMigrationCall{req.Method, req.URL.Path, req.URL.RawQuery})
		return reply(req), nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	return migrations.New(client)
}

func TestNativeMigrationListQueryAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeMigrationCall
	api := nativeMigrationAPI(t, &calls, func(*http.Request) *http.Response {
		// Nova 2.59 adds migrations_links, but the native single page never follows it.
		return nativeMigrationWire(200, `{"migrations":[{"id":42,"uuid":"mig-uuid","instance_uuid":"s-1","status":"finished","migration_type":"live-migration","new_instance_type_id":2,"old_instance_type_id":1,"source_compute":"cmp1","dest_compute":"cmp2","user_id":"u","project_id":"p","created_at":"2026-10-01T01:02:03.000000","updated_at":null}],"migrations_links":[{"rel":"next","href":"http://ignored/?marker=mig-uuid"}]}`)
	})
	host, instance, kind, source, status, marker, user, project := "cmp1", "s-1", "live-migration", "cmp1", "finished", "m", "u", "p"
	limit := 0
	since := time.Date(2026, 10, 1, 9, 0, 0, 0, time.FixedZone("KST", 9*3600))
	before := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	var rows []*migrations.Migration
	for value, err := range api.List(ctx, migrations.WithListOptions(migrations.ListOpts{Host: &host, InstanceID: &instance, MigrationType: &kind, SourceCompute: &source, Status: &status, Limit: &limit, Marker: &marker, ChangesSince: &since, ChangesBefore: &before, UserID: &user, ProjectID: &project}), migrations.WithListQuery("hidden", "0")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if len(rows) != 1 {
		t.Fatal(rows)
	}
	row := rows[0]
	if row.ID != 42 || row.UUID != "mig-uuid" || row.InstanceID != "s-1" || row.NewInstanceTypeID != 2 || !row.CreatedAt.Equal(time.Date(2026, 10, 1, 1, 2, 3, 0, time.UTC)) || !row.UpdatedAt.IsZero() {
		t.Fatalf("%+v", row)
	}
	// changes-since/before keep the caller's RFC3339 offset; a non-nil zero limit is sent.
	want := []nativeMigrationCall{{http.MethodGet, "/nova/v2.1/os-migrations", "changes-before=2026-10-02T00%3A00%3A00Z&changes-since=2026-10-01T09%3A00%3A00%2B09%3A00&hidden=0&host=cmp1&instance_uuid=s-1&limit=0&marker=m&migration_type=live-migration&project_id=p&source_compute=cmp1&status=finished&user_id=u"}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeMigrationListStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		code int
		body string
	}{{404, `{}`}, {200, `{"migrations":[]}`}, {200, `{}`}, {200, `{"migrations":[{"id":"x"}]}`}, {200, `{"migrations":[{"created_at":"2026-10-01T01:02:03Z"}]}`}, {204, ""}} {
		var calls []nativeMigrationCall
		api := nativeMigrationAPI(t, &calls, func(*http.Request) *http.Response { return nativeMigrationWire(tc.code, tc.body) })
		var errs []error
		for _, err := range api.List(ctx) {
			errs = append(errs, err)
		}
		var native gophercloud.ErrUnexpectedResponseCode
		var wrapped *resource.OperationError
		bad := strings.Contains(tc.body, `"x"`) || strings.Contains(tc.body, "Z\"")
		switch {
		case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}) && !errors.As(errs[0], &wrapped):
		case tc.code == 200 && !bad && len(errs) == 0:
		case tc.code == 200 && bad && len(errs) == 1 && errs[0] != nil && !errors.As(errs[0], &wrapped):
		case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
		default:
			t.Fatal(tc.code, tc.body, errs)
		}
	}
	var calls []nativeMigrationCall
	api := nativeMigrationAPI(t, &calls, func(*http.Request) *http.Response { return nativeMigrationWire(200, `{}`) })
	for _, err := range api.List(ctx, nil) {
		var wrapped *resource.OperationError
		if !errors.As(err, &wrapped) || wrapped.Operation != "List" || wrapped.Resource != "migrations" {
			t.Fatal(err)
		}
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}
