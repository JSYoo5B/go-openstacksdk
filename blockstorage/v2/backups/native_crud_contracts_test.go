package backups_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/backups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeBackupTransport func(*http.Request) (*http.Response, error)

func (transport nativeBackupTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeBackupWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeBackupCall struct{ method, path, query, body string }

func nativeBackupAPI(t *testing.T, calls *[]nativeBackupCall, reply func(*http.Request) *http.Response) (*backups.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeBackupTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeBackupCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v2/project/"
	return backups.New(client), cloud
}

func nativeBackupOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "backups" {
		t.Fatal("generated backups context", err, wrapped)
	}
}

const nativeBackupRow = `{"id":"bk-1","name":"nightly","volume_id":"vol-1","status":"available","size":10,"object_count":2,"is_incremental":true,"has_dependent_backups":false,"os-backup-project-attr:project_id":"project","metadata":{"k":"v"},"availability_zone":null,"created_at":"2026-10-11T01:02:03.000000","data_timestamp":"2026-10-11T01:02:00"}`

func TestNativeBackupRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeBackupCall
	var cloud *testcloud.Cloud
	api, cloud := nativeBackupAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeBackupWire(202, "")
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/restore"):
			return nativeBackupWire(202, `{"restore":{"backup_id":"bk-1","volume_id":"vol-2","volume_name":"restored"}}`)
		case req.Method == http.MethodPost:
			return nativeBackupWire(202, `{"backup":`+nativeBackupRow+`}`)
		case req.URL.Path == "/cinder/v2/project/backups" || req.URL.Path == "/cinder/v2/project/backups/detail":
			return nativeBackupWire(200, `{"backups":[`+nativeBackupRow+`],"backups_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/backups?marker=x"}]}`)
		case req.URL.Path == "/other/backups":
			return nativeBackupWire(200, `{"backups":[{"id":"bk-2","metadata":null}]}`)
		}
		return nativeBackupWire(200, `{"backup":`+nativeBackupRow+`}`)
	})
	created, err := api.Create(ctx, backups.CreateOpts{VolumeID: "vol-1", Name: "nightly", Incremental: true, Container: "c", Metadata: map[string]string{"k": "v"}}, backups.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "bk-1" && created.IsIncremental && created.ProjectID == "project" && (*created.Metadata)["k"] == "v" && created.AvailabilityZone == nil && created.DataTimestamp.Second() == 0 && created.CreatedAt.Second() == 3) {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "bk-1")
	if err != nil || got.ObjectCount != 2 {
		t.Fatal(got, err)
	}
	restored, err := api.RestoreFromBackup(ctx, "bk-1", backups.RestoreOpts{Name: "restored"}, backups.WithRestoreFromBackupField("x_extension", 1))
	if err != nil || *restored != (backups.Restore{BackupID: "bk-1", VolumeID: "vol-2", VolumeName: "restored"}) {
		t.Fatal(restored, err)
	}
	// Empty restore options leave an empty restore object.
	if _, err := api.RestoreFromBackup(ctx, "bk-1", backups.RestoreOpts{}); err != nil {
		t.Fatal(err)
	}
	collect := func(seq func(func(*backups.Backup, error) bool)) []string {
		var ids []string
		for value, err := range seq {
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, value.ID)
		}
		return ids
	}
	listed := collect(api.List(ctx, backups.WithListOptions(backups.ListOpts{VolumeID: "vol-1", AllTenants: true, Limit: 1}), backups.WithListQuery("extra", "1")))
	detailed := collect(api.ListDetail(ctx, backups.WithListDetailOptions(backups.ListDetailOpts{WithCount: true, Sort: "created_at:desc"}), backups.WithListDetailQuery("extra", "1")))
	if !reflect.DeepEqual(listed, []string{"bk-1", "bk-2"}) || !reflect.DeepEqual(detailed, []string{"bk-1", "bk-2"}) {
		t.Fatal(listed, detailed)
	}
	if err := api.Delete(ctx, "bk-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 9 {
		t.Fatalf("%+v", calls)
	}
	listQuery, _ := url.ParseQuery(calls[4].query)
	detailQuery, _ := url.ParseQuery(calls[6].query)
	base := "/cinder/v2/project/backups"
	want := []nativeBackupCall{
		{http.MethodPost, base, "", `{"backup":{"container":"c","incremental":true,"metadata":{"k":"v"},"name":"nightly","volume_id":"vol-1","x_extension":1}}`},
		{http.MethodGet, base + "/bk-1", "", ""},
		{http.MethodPost, base + "/bk-1/restore", "", `{"restore":{"name":"restored","x_extension":1}}`},
		{http.MethodPost, base + "/bk-1/restore", "", `{"restore":{}}`},
		// List and ListDetail take different option types and routes.
		{http.MethodGet, base, calls[4].query, ""},
		{http.MethodGet, "/other/backups", "marker=x", ""},
		{http.MethodGet, base + "/detail", calls[6].query, ""},
		{http.MethodGet, "/other/backups", "marker=x", ""},
		{http.MethodDelete, base + "/bk-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(listQuery, url.Values{"volume_id": {"vol-1"}, "all_tenants": {"true"}, "limit": {"1"}, "extra": {"1"}}) || !reflect.DeepEqual(detailQuery, url.Values{"with_count": {"true"}, "sort": {"created_at:desc"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v %v", calls, listQuery, detailQuery)
	}
}

// Update is excluded: Cinder added backup update in v3.9 and the v2 adapter keeps the flat native body.
func TestNativeBackupStrictStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*backups.API) error
	}{
		// Create and restore accept only 202.
		{"Create", []int{202}, func(api *backups.API) error {
			_, err := api.Create(ctx, backups.CreateOpts{VolumeID: "vol-1"})
			return err
		}},
		{"Get", []int{200}, func(api *backups.API) error { _, err := api.Get(ctx, "bk-1"); return err }},
		{"RestoreFromBackup", []int{202}, func(api *backups.API) error {
			_, err := api.RestoreFromBackup(ctx, "bk-1", backups.RestoreOpts{})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *backups.API) error { return api.Delete(ctx, "bk-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeBackupCall
				api, _ := nativeBackupAPI(t, &calls, func(*http.Request) *http.Response { return nativeBackupWire(code, `{"backup":{}}`) })
				err := call.call(api)
				nativeBackupOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope and timestamp decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			// An empty object or null envelope yields a zero backup.
			`{}`:              false,
			`{"backup":null}`: false,
			`{"other":{}}`:    true,
			`{"backup":{"data_timestamp":"2026-10-11T01:02:03Z"}}`: true,
		} {
			var calls []nativeBackupCall
			api, _ := nativeBackupAPI(t, &calls, func(*http.Request) *http.Response { return nativeBackupWire(200, body) })
			got, err := api.Get(ctx, "bk-1")
			if wantErr {
				nativeBackupOperation(t, err, "Get")
			} else if err != nil || got == nil || got.ID != "" || got.Metadata != nil {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"backups":[]}`}, {204, ""}} {
			for name, list := range map[string]func(*backups.API) []error{
				"List": func(api *backups.API) (errs []error) {
					for _, err := range api.List(ctx) {
						errs = append(errs, err)
					}
					return errs
				},
				"ListDetail": func(api *backups.API) (errs []error) {
					for _, err := range api.ListDetail(ctx) {
						errs = append(errs, err)
					}
					return errs
				},
			} {
				var calls []nativeBackupCall
				api, _ := nativeBackupAPI(t, &calls, func(*http.Request) *http.Response { return nativeBackupWire(tc.code, tc.body) })
				errs := list(api)
				var native gophercloud.ErrUnexpectedResponseCode
				switch {
				case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
				case tc.code == 200 && len(errs) == 0:
				case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
				default:
					t.Fatal(name, tc.code, errs)
				}
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeBackupCall
		api, _ := nativeBackupAPI(t, &calls, func(*http.Request) *http.Response { return nativeBackupWire(202, `{}`) })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"volume id": {"Create", func() error { _, err := api.Create(ctx, backups.CreateOpts{}); return err }()},
			"core extension": {"Create", func() error {
				_, err := api.Create(ctx, backups.CreateOpts{VolumeID: "vol-1"}, backups.WithCreateField("incremental", true))
				return err
			}()},
			"restore nil option": {"RestoreFromBackup", func() error {
				_, err := api.RestoreFromBackup(ctx, "bk-1", backups.RestoreOpts{}, nil)
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeBackupOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeBackupOperation(t, err, "List")
		}
		for _, err := range api.ListDetail(ctx, nil) {
			nativeBackupOperation(t, err, "ListDetail")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
