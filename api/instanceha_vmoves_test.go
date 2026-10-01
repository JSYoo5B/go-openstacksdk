package api_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"gophercloudsdk/instanceha/v1/vmoves"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

const vmoveUUID = "55555555-5555-4555-8555-555555555555"

func TestInstanceHAVMoveFixedNotificationGetAndTypedList(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, lists atomic.Int32
	client := cloud.Client("instance-ha", "/instance-ha/v1/catalog-tenant")
	client.Microversion = "1.3"
	response := `{"uuid":"` + vmoveUUID + `","id":9007199254740993,"notification_uuid":"` + notificationUUID + `","instance_uuid":"` + hostUUID + `","instance_name":"vm-01","source_host":"compute-01","dest_host":null,"start_time":"2026-10-01 09:10:11.123456","end_time":null,"status":"ongoing","type":"evacuation","message":null,"custom":9007199254740995}`
	path := "/instance-ha/v1/catalog-tenant/notifications/" + notificationUUID + "/vmoves"
	cloud.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if r.Method != "GET" || r.URL.Query().Get("type") != "evacuation" || r.URL.Query().Get("status") != "ongoing" {
			t.Errorf("request=%s %v", r.Method, r.URL)
		}
		testcloud.JSON(w, 200, `{"vmoves":[`+response+`]}`)
	})
	cloud.Mux.HandleFunc(path+"/"+vmoveUUID, func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if r.Header.Get("OpenStack-API-Version") != "instance-ha 1.3" {
			t.Error(r.Header)
		}
		w.Header().Set("X-Request-ID", "vmove-get")
		testcloud.JSON(w, 200, `{"vmove":`+response+`}`)
	})
	scope, err := vmoves.New(client).InNotification(context.Background(), resource.ID(notificationUUID))
	if err != nil {
		t.Fatal(err)
	}
	if gets.Load() != 0 || lists.Load() != 0 || scope.NotificationID() != notificationUUID {
		t.Fatal("scope made parent HTTP")
	}
	state, kind := "ongoing", "evacuation"
	values, err := scope.All(context.Background(), vmoves.WithListOptions(vmoves.ListOpts{Status: &state, Type: &kind}))
	if err != nil || len(values) != 1 {
		t.Fatalf("values=%+v err=%v", values, err)
	}
	value, err := scope.Get(context.Background(), values[0].UUID)
	if err != nil {
		t.Fatal(err)
	}
	if value.NotificationID != notificationUUID || value.NotificationUUID != notificationUUID || value.ServerID != hostUUID || value.ServerName != "vm-01" || string(value.ID) != "9007199254740993" || string(value.Body["custom"]) != "9007199254740995" || value.DestHost != nil || value.EndTime != nil || value.Message != nil || value.StatusCode != 200 || value.Header.Get("X-Request-ID") != "vmove-get" {
		t.Fatalf("value=%+v", value)
	}
	if gets.Load() != 1 || lists.Load() != 1 {
		t.Fatalf("get=%d list=%d", gets.Load(), lists.Load())
	}
}

func TestInstanceHAVMoveMinimumVersionAndNamesFailBeforeHTTP(t *testing.T) {
	for _, version := range []string{"", "1.0", "1.2", "latest"} {
		t.Run(version, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("below-minimum HTTP %s", r.URL) })
			client := cloud.Client("instance-ha", "/v1")
			client.Microversion = version
			for _, ref := range []resource.Ref{resource.ID(notificationUUID), resource.Name("notification-name")} {
				if _, err := vmoves.New(client).InNotification(context.Background(), ref); !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal(err)
				}
			}
		})
	}
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("unsupported HTTP %s", r.URL) })
	client := cloud.Client("instance-ha", "/v1")
	client.Microversion = "1.3"
	if _, err := vmoves.New(client).InNotification(context.Background(), resource.Name("name")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	scope, err := vmoves.New(client).InNotification(context.Background(), resource.ID(notificationUUID))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scope.Find(context.Background(), resource.Name("vm-01")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if err := scope.Delete(context.Background(), resource.ID(vmoveUUID)); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := scope.Get(context.Background(), "1"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestInstanceHAVMoveChangedVersionAndParentQueryCannotEscapeScope(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("invalid request %s", r.URL) })
	client := cloud.Client("instance-ha", "/v1")
	client.Microversion = "1.3"
	scope, err := vmoves.New(client).InNotification(context.Background(), resource.ID(notificationUUID))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"notification_id", "notification_uuid", "notification"} {
		if _, err := scope.Resources.All(context.Background(), resource.WithQuery(key, otherSegmentUUID)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: %v", key, err)
		}
		if _, err := scope.All(context.Background(), vmoves.WithListQuery(key, otherSegmentUUID)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: %v", key, err)
		}
	}
	client.Microversion = "1.2"
	operations := []func() error{func() error { _, e := scope.Get(context.Background(), vmoveUUID); return e }, func() error { _, e := scope.All(context.Background()); return e }, func() error { _, e := scope.Resources.All(context.Background()); return e }, func() error { _, e := scope.Wait(context.Background(), resource.ID(vmoveUUID), "succeeded"); return e }}
	for _, run := range operations {
		if err := run(); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(err)
		}
	}
	client.Microversion = "1.3"
	client.MoreHeaders = map[string]string{"openstack-api-version": "instance-ha 1.2"}
	if _, err := scope.Get(context.Background(), vmoveUUID); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestInstanceHAVMovePaginationRejectsAnotherParentAndRechecksVersion(t *testing.T) {
	for _, changeVersion := range []bool{false, true} {
		t.Run(map[bool]string{false: "parent", true: "version"}[changeVersion], func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			path := "/v1/notifications/" + notificationUUID + "/vmoves"
			cloud.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				next := path + "?marker=next-page"
				if !changeVersion {
					next = "/v1/notifications/" + otherSegmentUUID + "/vmoves?marker=next-page"
				}
				testcloud.JSON(w, 200, `{"vmoves":[{"uuid":"`+vmoveUUID+`","status":"ongoing"}],"next":"`+next+`"}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("pagination escaped parent %s", r.URL) })
			client := cloud.Client("instance-ha", "/v1")
			client.Microversion = "1.3"
			scope, err := vmoves.New(client).InNotification(context.Background(), resource.ID(notificationUUID))
			if err != nil {
				t.Fatal(err)
			}
			var final error
			var values int
			for _, err := range scope.List(context.Background()) {
				if err != nil {
					final = err
					break
				}
				values++
				if changeVersion {
					client.Microversion = "1.2"
				}
			}
			want := resource.ErrInvalidOption
			if changeVersion {
				want = resource.ErrUnsupported
			}
			if !errors.Is(final, want) || calls.Load() != 1 || values != 1 {
				t.Fatalf("values=%d calls=%d err=%v", values, calls.Load(), final)
			}
		})
	}
}

func TestInstanceHAVMoveMissingUUIDRemainsMissingAndWaitUsesRealStatus(t *testing.T) {
	cloud := testcloud.New(t)
	var polls atomic.Int32
	client := cloud.Client("instance-ha", "/v1")
	client.Microversion = "1.3"
	path := "/v1/notifications/" + notificationUUID + "/vmoves"
	cloud.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"vmoves":[{"id":1,"instance_uuid":"`+hostUUID+`","instance_name":"vm-01","status":"ongoing"}]}`)
	})
	cloud.Mux.HandleFunc(path+"/"+vmoveUUID, func(w http.ResponseWriter, r *http.Request) {
		state := "ongoing"
		if polls.Add(1) > 1 {
			state = "succeeded"
		}
		testcloud.JSON(w, 200, `{"vmove":{"uuid":"`+vmoveUUID+`","status":"`+state+`"}}`)
	})
	scope, err := vmoves.New(client).InNotification(context.Background(), resource.ID(notificationUUID))
	if err != nil {
		t.Fatal(err)
	}
	values, err := scope.All(context.Background())
	if err != nil || len(values) != 1 || values[0].UUID != "" || string(values[0].ID) != "1" {
		t.Fatalf("values=%+v err=%v", values, err)
	}
	var progress []int
	value, err := scope.Wait(context.Background(), resource.ID(vmoveUUID), "succeeded", resource.WithPollInterval(time.Millisecond), resource.WithProgressCallback(func(p int) { progress = append(progress, p) }))
	if err != nil || value.Status != "succeeded" || polls.Load() != 2 || !reflect.DeepEqual(progress, []int{0}) {
		t.Fatalf("value=%+v polls=%d progress=%v err=%v", value, polls.Load(), progress, err)
	}
}

func TestInstanceHAVMoveIteratorSnapshotsOptionSlice(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/v1/notifications/"+notificationUUID+"/vmoves", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("status") != "ongoing" {
			t.Errorf("query=%v", r.URL.Query())
		}
		testcloud.JSON(w, 200, `{"vmoves":[{"uuid":"`+vmoveUUID+`","status":"ongoing"}]}`)
	})
	client := cloud.Client("instance-ha", "/v1")
	client.Microversion = "1.3"
	scope, err := vmoves.New(client).InNotification(context.Background(), resource.ID(notificationUUID))
	if err != nil {
		t.Fatal(err)
	}
	status := "ongoing"
	options := []vmoves.ListOption{vmoves.WithListOptions(vmoves.ListOpts{Status: &status})}
	iterator := scope.List(context.Background(), options...)
	status = "mutated"
	options[0] = vmoves.WithListQuery("status", "mutated")
	for value, err := range iterator {
		if err != nil || value.Status != "ongoing" {
			t.Fatalf("value=%+v err=%v", value, err)
		}
	}
}
