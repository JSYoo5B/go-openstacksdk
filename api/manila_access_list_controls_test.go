package api_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/JSYoo5B/gophercloudsdk/sharedfilesystems/v2/shareaccessrules"
	"github.com/gophercloud/gophercloud/v2"
)

func TestManilaAccessListControlsSnapshotAndSingleCollection(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc(shareAccessBase+"/share-access-rules", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		checkShareAccessHeaders(t, r, "2.82")
		want := map[string][]string{"share_id": {"share-id"}, "limit": {"7"}, "marker": {"wire-marker"}, "vendor": {"a&b"}}
		if r.Method != http.MethodGet || !reflect.DeepEqual(map[string][]string(r.URL.Query()), want) || r.Header.Get("X-Configured") != "source" || r.Header.Get("X-List") != "owned" {
			t.Errorf("method=%s query=%s header=%v", r.Method, r.URL.RawQuery, r.Header)
		}
		w.Header().Set("X-Request-Id", "owned-list")
		testcloud.JSON(w, 200, `{"access_list":[{"id":"first","metadata":{"large":9007199254740993},"lock_visibility":false},{"id":"second","share_id":"share-id"},{"id":"third"}],"links":[{"rel":"next","href":"https://foreign.invalid/steal-token"}]}`)
	})
	client := shareAccessClient(cloud, "2.82")
	client.MoreHeaders = map[string]string{"X-Configured": "source"}
	scope, err := shareaccessrules.New(client).InShare(context.Background(), resource.ID("share-id"))
	if err != nil {
		t.Fatal(err)
	}
	options := []shareaccessrules.ListOption{
		shareaccessrules.WithListMaxItems(1), shareaccessrules.WithListMaxItems(2),
		shareaccessrules.WithListPaginated(true), shareaccessrules.WithListPaginated(false),
		shareaccessrules.WithListQuery("limit", "7"), shareaccessrules.WithListQuery("marker", "wire-marker"),
		shareaccessrules.WithListQuery("vendor", "a&b"), shareaccessrules.WithListHeader("X-List", "owned"),
	}
	sequence := scope.List(context.Background(), options...)
	options[1] = shareaccessrules.WithListMaxItems(1)
	if calls.Load() != 0 {
		t.Fatal("List must remain lazy")
	}
	for range 2 {
		var values []*shareaccessrules.AccessRule
		for value, err := range sequence {
			if err != nil {
				t.Fatal(err)
			}
			values = append(values, value)
		}
		if len(values) != 2 || values[0].ID != "first" || values[1].ID != "second" || values[0].ParentShareID != "share-id" || values[0].ShareID != "" || string(values[0].Body["metadata"]) != `{"large":9007199254740993}` || values[0].LockVisibility == nil || *values[0].LockVisibility || values[0].Header.Get("X-Request-Id") != "owned-list" {
			t.Fatalf("owned values=%+v", values)
		}
	}
	values, err := scope.All(context.Background(), append(options[2:], shareaccessrules.WithListMaxItems(1), shareaccessrules.WithListMaxItems(0), shareaccessrules.WithListPaginated(true))...)
	if err != nil || len(values) != 3 || calls.Load() != 3 || client.Microversion != "2.82" || client.Endpoint != cloud.Server.URL+shareAccessBase+"/" || !reflect.DeepEqual(client.MoreHeaders, map[string]string{"X-Configured": "source"}) {
		t.Fatalf("all=%+v err=%v calls=%d source=%+v", values, err, calls.Load(), client)
	}
}

func TestManilaAccessListControlsCapBeforeLaterDecode(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	body := `{"access_list":[{"id":"first"},{"id":"later","created_at":"bad-date"}],"access_list_links":{"broken":"ignored"},"next":"https://foreign.invalid/next"}`
	cloud.Mux.HandleFunc(shareAccessBase+"/share-access-rules", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if len(r.URL.Query()) != 1 || r.URL.Query().Get("share_id") != "share-id" {
			t.Errorf("local controls must not add a wire hint: %s", r.URL.RawQuery)
		}
		w.Header().Set("X-Request-Id", "later-decode")
		testcloud.JSON(w, 200, body)
	})
	scope := shareAccessScope(t, cloud)
	for _, paginated := range []bool{false, true} {
		values, err := scope.All(context.Background(), shareaccessrules.WithListMaxItems(1), shareaccessrules.WithListPaginated(paginated))
		if err != nil || len(values) != 1 || values[0].ID != "first" {
			t.Fatalf("paginated=%v values=%+v err=%v", paginated, values, err)
		}
	}
	for _, options := range [][]shareaccessrules.ListOption{nil, {shareaccessrules.WithListPaginated(false)}, {shareaccessrules.WithListMaxItems(2)}} {
		values, err := scope.All(context.Background(), options...)
		var accepted *resource.ResponseError
		if values != nil || !errors.As(err, &accepted) || string(accepted.Body) != body || accepted.StatusCode != 200 || accepted.Header.Get("X-Request-Id") != "later-decode" {
			t.Fatalf("later row must fail with whole response: values=%v error=%v evidence=%+v", values, err, accepted)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for value, err := range scope.List(ctx, shareaccessrules.WithListMaxItems(1)) {
		if err != nil || value.ID != "first" {
			t.Fatalf("value=%+v err=%v", value, err)
		}
		cancel()
		break // Consumer break takes precedence over its own cancellation.
	}
	if calls.Load() != 6 {
		t.Fatalf("one request per consumption, got %d", calls.Load())
	}
}

func TestManilaAccessListControlsRejectBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 500, `{}`) })
	scope := shareAccessScope(t, cloud)
	for _, option := range []shareaccessrules.ListOption{
		nil, shareaccessrules.WithListMaxItems(-1), shareaccessrules.WithListQuery("max_items", "1"),
		shareaccessrules.WithListQuery("paginated", "false"), shareaccessrules.WithListQuery("share_id", "other"),
		shareaccessrules.WithListHeader("x-openstack-manila-api-version", "2.82"),
		request.WithField[shareaccessrules.ListOpts]("field", true), request.WithArgument[shareaccessrules.ListOpts]("foreign", true),
		request.WithArgument[shareaccessrules.ListOpts]("shareaccessrules.list_control", "wrong-type"),
	} {
		if _, err := scope.All(context.Background(), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("option error=%v", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := scope.All(ctx, shareaccessrules.WithListMaxItems(1)); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatalf("preflight emitted %d requests", calls.Load())
	}
}

func TestManilaAccessListControlsStrictSourceVersion(t *testing.T) {
	for _, tc := range []struct {
		name, version, service string
		headers                map[string]string
		ok                     bool
	}{
		{"native", "2.45", "shared-file-system", nil, true},
		{"sharev2", "2.82", "sharev2", nil, true},
		{"share", "2.100", "share", nil, true},
		{"manual legacy", "2.45", "", map[string]string{"x-openstack-manila-api-version": "2.45"}, true},
		{"matching headers", "2.82", "shared-file-system", map[string]string{"x-openstack-manila-api-version": "2.82", "openstack-api-version": "share 2.82"}, true},
		{"empty type", "2.45", "", nil, false},
		{"generic only", "2.45", "", map[string]string{"OpenStack-API-Version": "share 2.45"}, false},
		{"other service", "2.45", "compute", nil, false},
		{"leading major", "02.45", "shared-file-system", nil, false},
		{"leading minor", "2.045", "shared-file-system", nil, false},
		{"latest", "latest", "shared-file-system", nil, false},
		{"empty selected", "", "shared-file-system", nil, false},
		{"legacy override", "2.82", "shared-file-system", map[string]string{"x-openstack-manila-api-version": "2.44"}, false},
		{"generic override", "2.82", "shared-file-system", map[string]string{"openstack-api-version": "compute 2.82"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(shareAccessBase+"/share-access-rules", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Header.Get("X-OpenStack-Manila-API-Version") != tc.version || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Errorf("selected headers=%v", r.Header)
				}
				testcloud.JSON(w, 200, `{"access_list":[]}`)
			})
			client := shareAccessClient(cloud, tc.version)
			client.Type, client.MoreHeaders = tc.service, tc.headers
			scope, err := shareaccessrules.New(client).InShare(context.Background(), resource.ID("share-id"))
			if tc.ok {
				if err != nil {
					t.Fatal(err)
				}
				values, err := scope.All(context.Background(), shareaccessrules.WithListMaxItems(1))
				if err != nil || values == nil || len(values) != 0 || calls.Load() != 1 {
					t.Fatalf("values=%v err=%v calls=%d", values, err, calls.Load())
				}
			} else if err == nil || calls.Load() != 0 {
				t.Fatalf("invalid source scope=%v err=%v calls=%d", scope, err, calls.Load())
			}
		})
	}
}

func TestManilaAccessListControlsRecheckSourceAfterOptionsAndLookup(t *testing.T) {
	for _, method := range []string{"List", "Get", "Allow", "Deny"} {
		t.Run(method, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 500, `{}`) })
			client := shareAccessClient(cloud, "2.82")
			scope, err := shareaccessrules.New(client).InShare(context.Background(), resource.ID("share-id"))
			if err != nil {
				t.Fatal(err)
			}
			switch method {
			case "List":
				_, err = scope.All(context.Background(), func(*request.Config[shareaccessrules.ListOpts]) error { client.Microversion = "2.44"; return nil })
			case "Get":
				_, err = scope.Get(context.Background(), "rule-id", func(*request.Config[shareaccessrules.GetOpts]) error { client.Type = "compute"; return nil })
			case "Allow":
				_, err = scope.Allow(context.Background(), shareaccessrules.AllowOpts{AccessType: "ip", AccessTo: "192.0.2.1"}, func(*request.Config[shareaccessrules.AllowOpts]) error {
					client.MoreHeaders = map[string]string{"x-openstack-manila-api-version": "2.44"}
					return nil
				})
			case "Deny":
				err = scope.Deny(context.Background(), "rule-id", func(*request.Config[shareaccessrules.DenyOpts]) error { client.Microversion = "2.44"; return nil })
			}
			if err == nil || calls.Load() != 0 {
				t.Fatalf("post-options guard err=%v calls=%d", err, calls.Load())
			}
		})
	}
	for _, parentLookup := range []bool{true, false} {
		t.Run(map[bool]string{true: "parent name", false: "deny access ownership"}[parentLookup], func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if parentLookup && r.URL.Path == shareAccessBase+"/shares/detail" {
					testcloud.JSON(w, 200, `{"shares":[{"id":"share-id","name":"team.prod"}]}`)
				} else if !parentLookup && r.URL.Path == shareAccessBase+"/share-access-rules/rule-id" {
					testcloud.JSON(w, 200, `{"access":{"id":"rule-id","share_id":"share-id"}}`)
				} else {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
					testcloud.JSON(w, 500, `{}`)
				}
			})
			client := shareAccessClient(cloud, "2.82")
			transport := client.HTTPClient.Transport
			client.HTTPClient.Transport = manilaAccessControlTransport(func(r *http.Request) (*http.Response, error) {
				response, err := transport.RoundTrip(r)
				if parentLookup {
					client.Microversion = "2.44"
				} else {
					client.Microversion = "2.81"
				}
				return response, err
			})
			var err error
			if parentLookup {
				_, err = shareaccessrules.New(client).InShare(context.Background(), resource.Name("team.prod"))
			} else {
				scope, bindErr := shareaccessrules.New(client).InShare(context.Background(), resource.ID("share-id"))
				if bindErr != nil {
					t.Fatal(bindErr)
				}
				err = scope.Deny(context.Background(), "rule-id", shareaccessrules.WithUnrestrict(false))
			}
			if !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 1 {
				t.Fatalf("post-lookup guard err=%v calls=%d", err, calls.Load())
			}
		})
	}
}

func TestManilaAccessListControlsAcceptedAndNativeErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		parent     bool
	}{
		{"missing array", `{}`, 200, false}, {"null array", `{"access_list":null}`, 200, false},
		{"bad JSON", `{"access_list":[`, 200, false}, {"bad row", `{"access_list":[{"id":42}]}`, 200, false},
		{"wrong parent", `{"access_list":[{"id":"foreign","share_id":"other"},{"id":"good"}]}`, 200, true},
		{"native", `{"forbidden":{"message":"unchanged"}}`, 403, false},
		{"unexpected success", `{"access_list":[]}`, 201, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Request-Id", "original-error")
				testcloud.JSON(w, tc.status, tc.body)
			})
			values, err := shareAccessScope(t, cloud).All(context.Background(), shareaccessrules.WithListMaxItems(1))
			if err == nil || values != nil || calls.Load() != 1 {
				t.Fatalf("values=%v err=%v calls=%d", values, err, calls.Load())
			}
			if tc.status == 200 {
				var accepted *resource.ResponseError
				if !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != tc.body || accepted.Header.Get("X-Request-Id") != "original-error" || accepted.Cause == nil {
					t.Fatalf("accepted evidence=%+v err=%v", accepted, err)
				}
				if tc.parent && !errors.Is(err, shareaccessrules.ErrParentMismatch) {
					t.Fatal(err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				var accepted *resource.ResponseError
				if !errors.As(err, &native) || errors.As(err, &accepted) || native.Actual != tc.status || string(native.Body) != tc.body || native.ResponseHeader.Get("X-Request-Id") != "original-error" {
					t.Fatalf("native evidence=%+v err=%v", native, err)
				}
			}
		})
	}
}

func TestManilaAccessListControlsTerminalCancellation(t *testing.T) {
	for _, maximum := range []int{0, 1} {
		t.Run(map[int]string{0: "final row", 1: "cap row"}[maximum], func(t *testing.T) {
			cloud := testcloud.New(t)
			body := `{"access_list":[{"id":"one"}]}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, body) })
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var rows, failures int
			for value, err := range shareAccessScope(t, cloud).List(ctx, shareaccessrules.WithListMaxItems(maximum)) {
				if value != nil {
					rows++
					cancel()
					continue
				}
				failures++
				var accepted *resource.ResponseError
				if !errors.Is(err, context.Canceled) || !errors.As(err, &accepted) || string(accepted.Body) != body || accepted.StatusCode != 200 {
					t.Fatalf("terminal cancellation=%v evidence=%+v", err, accepted)
				}
			}
			if rows != 1 || failures != 1 {
				t.Fatalf("rows=%d failures=%d", rows, failures)
			}
		})
	}
	t.Run("in flight", func(t *testing.T) {
		cloud := testcloud.New(t)
		started := make(chan struct{})
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			close(started)
			<-r.Context().Done()
		})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		scope := shareAccessScope(t, cloud)
		result := make(chan error, 1)
		go func() { _, err := scope.All(ctx, shareaccessrules.WithListMaxItems(1)); result <- err }()
		<-started
		cancel()
		if err := <-result; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	})
}

type manilaAccessControlTransport func(*http.Request) (*http.Response, error)

func (f manilaAccessControlTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
