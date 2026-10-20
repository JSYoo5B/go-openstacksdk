package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/instanceha/v1/hosts"
	"github.com/JSYoo5B/go-openstacksdk/instanceha/v1/notifications"
	"github.com/JSYoo5B/go-openstacksdk/instanceha/v1/segments"
	"github.com/JSYoo5B/go-openstacksdk/instanceha/v1/vmoves"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type instanceHAWaitAPI struct {
	kind, path, envelope, uuid string
	hasStatus                  bool
	status                     func(context.Context, resource.Ref, string, ...resource.WaitOption) (*resource.Metadata, error)
	deleted                    func(context.Context, resource.Ref, ...resource.WaitOption) error
}

func instanceHAWaitAPIs(t *testing.T, client *gophercloud.ServiceClient) []instanceHAWaitAPI {
	t.Helper()
	n := notifications.New(client)
	g := segments.New(client)
	h, err := hosts.New(client).InSegment(context.Background(), resource.ID(segmentUUID))
	if err != nil {
		t.Fatal(err)
	}
	v, err := vmoves.New(client).InNotification(context.Background(), resource.ID(notificationUUID))
	if err != nil {
		t.Fatal(err)
	}
	return []instanceHAWaitAPI{
		{"notifications", "notifications/" + notificationUUID, "notification", notificationUUID, true,
			func(ctx context.Context, ref resource.Ref, status string, opts ...resource.WaitOption) (*resource.Metadata, error) {
				value, err := n.WaitForStatus(ctx, ref, status, opts...)
				if value == nil {
					return nil, err
				}
				return &value.Metadata, err
			}, n.WaitForDelete},
		{"segments", "segments/" + segmentUUID, "segment", segmentUUID, false,
			func(ctx context.Context, ref resource.Ref, status string, opts ...resource.WaitOption) (*resource.Metadata, error) {
				value, err := g.WaitForStatus(ctx, ref, status, opts...)
				if value == nil {
					return nil, err
				}
				return &value.Metadata, err
			}, g.WaitForDelete},
		{"hosts", "segments/" + segmentUUID + "/hosts/" + hostUUID, "host", hostUUID, false,
			func(ctx context.Context, ref resource.Ref, status string, opts ...resource.WaitOption) (*resource.Metadata, error) {
				value, err := h.WaitForStatus(ctx, ref, status, opts...)
				if value == nil {
					return nil, err
				}
				return &value.Metadata, err
			}, h.WaitForDelete},
		{"vmoves", "notifications/" + notificationUUID + "/vmoves/" + vmoveUUID, "vmove", vmoveUUID, true,
			func(ctx context.Context, ref resource.Ref, status string, opts ...resource.WaitOption) (*resource.Metadata, error) {
				value, err := v.WaitForStatus(ctx, ref, status, opts...)
				if value == nil {
					return nil, err
				}
				return &value.Metadata, err
			}, v.WaitForDelete},
	}
}

func TestInstanceHAWaitFourFacadesPreserveFixedRoutesAndHTTPBody(t *testing.T) {
	for index := 0; index < 4; index++ {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("instance-ha", "/proxy/instance-ha/v1/project")
			client.Microversion = "1.3"
			api := instanceHAWaitAPIs(t, client)[index]
			var calls, callbacks atomic.Int32
			cloud.Mux.HandleFunc("/proxy/instance-ha/v1/project/"+api.path, func(w http.ResponseWriter, r *http.Request) {
				poll := calls.Add(1)
				if r.Method != "GET" || r.Header.Get("OpenStack-API-Version") != "instance-ha 1.3" || r.URL.RawQuery != "" {
					t.Errorf("request=%s %s headers=%v", r.Method, r.URL, r.Header)
				}
				state := "ongoing"
				if poll == 2 {
					state = "succeeded"
				}
				// Numeric database ID and a changed response UUID cannot redirect polling.
				w.Header().Set("X-Request-ID", "poll-evidence")
				w.Header().Set("Location", "https://foreign.invalid/do-not-follow")
				testcloud.JSON(w, 200, fmt.Sprintf(`{%q:{"id":9007199254740993,"uuid":"%s","notification_uuid":"%s","status":%q,"name":%q,"vendor":9007199254740995,"nullable":null}}`, api.envelope, otherSegmentUUID, otherSegmentUUID, state, state))
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("escaped route: %s %s", r.Method, r.URL) })
			opts := []resource.WaitOption{resource.WithPollInterval(time.Millisecond), resource.WithProgressCallback(func(progress int) {
				callbacks.Add(1)
				if progress != 0 {
					t.Errorf("unexpected progress=%d", progress)
				}
			})}
			if !api.hasStatus {
				opts = append(opts, resource.WithStatusAttribute("name"))
			}
			value, err := api.status(context.Background(), resource.ID(api.uuid), "SUCCEEDED", opts...)
			if err != nil || value == nil || calls.Load() != 2 || string(value.Body["id"]) != "9007199254740993" || string(value.Body["vendor"]) != "9007199254740995" || string(value.Body["nullable"]) != "null" || value.StatusCode != 200 || value.Header.Get("X-Request-ID") != "poll-evidence" || value.Header.Get("Location") != "https://foreign.invalid/do-not-follow" || callbacks.Load() != 1 {
				t.Fatalf("kind=%s value=%+v err=%v calls/callbacks=%d/%d", api.kind, value, err, calls.Load(), callbacks.Load())
			}
		})
	}
}

func TestInstanceHAWaitStatuslessModelsRejectDefaultBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("instance-ha", "/v1")
	client.Microversion = "1.3"
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("invalid HTTP %s", r.URL) })
	for _, api := range instanceHAWaitAPIs(t, client) {
		if !api.hasStatus {
			if value, err := api.status(context.Background(), resource.ID(api.uuid), "active"); value != nil || !errors.Is(err, resource.ErrUnsupported) {
				t.Fatalf("%s value=%v err=%v", api.kind, value, err)
			}
		}
		for _, opts := range [][]resource.WaitOption{{nil}, {resource.WithTimeout(0)}, {resource.WithPollInterval(-time.Second)}, {resource.WithStatusAttribute("unknown_vendor_status")}} {
			if _, err := api.status(context.Background(), resource.ID(api.uuid), "active", opts...); err == nil {
				t.Fatalf("%s admitted invalid wait options", api.kind)
			}
		}
		if _, err := api.status(context.Background(), resource.ID("9007199254740993"), "active"); err == nil {
			t.Fatalf("%s admitted database ID", api.kind)
		}
	}
}

func TestInstanceHAWaitDeletionUsesStatusOrAbsenceWithoutDELETE(t *testing.T) {
	for index := 0; index < 4; index++ {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("instance-ha", "/v1")
			client.Microversion = "1.3"
			api := instanceHAWaitAPIs(t, client)[index]
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/v1/"+api.path, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("deletion waiter submitted %s", r.Method)
				}
				if calls.Add(1) == 1 {
					testcloud.JSON(w, 200, fmt.Sprintf(`{%q:{"uuid":"%s","notification_uuid":"%s","status":"DELETED"}}`, api.envelope, api.uuid, api.uuid))
				} else {
					testcloud.JSON(w, 404, `{"error":"gone"}`)
				}
			})
			if err := api.deleted(context.Background(), resource.ID(api.uuid), resource.WithPollInterval(time.Millisecond)); err != nil {
				t.Fatal(err)
			}
			want := int32(2)
			if api.hasStatus {
				want = 1
			}
			if calls.Load() != want {
				t.Fatalf("%s calls=%d want=%d", api.kind, calls.Load(), want)
			}
		})
	}
}

type instanceHAWaitTransport func(*http.Request) (*http.Response, error)

func (fn instanceHAWaitTransport) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func TestInstanceHAWaitDeadlineDefaultsAndOverrides(t *testing.T) {
	for _, operation := range []string{"status-unlimited", "delete-default", "status-override", "delete-unlimited", "delete-parent", "common-wait"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("instance-ha", "/v1")
			client.Microversion = "1.3"
			api := instanceHAWaitAPIs(t, client)[0]
			original := client.ProviderClient.HTTPClient.Transport
			var calls atomic.Int32
			client.ProviderClient.HTTPClient.Transport = instanceHAWaitTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				deadline, has := r.Context().Deadline()
				want := time.Duration(0)
				switch operation {
				case "delete-default":
					want = 120 * time.Second
				case "status-override":
					want = 20 * time.Second
				case "delete-parent":
					want = 10 * time.Second
				case "common-wait":
					want = 5 * time.Minute
				}
				if (want > 0) != has || (has && (time.Until(deadline) > want || time.Until(deadline) < want-time.Second)) {
					t.Errorf("%s deadline exists=%v remaining=%v want=%v", operation, has, time.Until(deadline), want)
				}
				return original.RoundTrip(r)
			})
			cloud.Mux.HandleFunc("/v1/"+api.path, func(w http.ResponseWriter, r *http.Request) {
				if operation == "delete-default" || operation == "delete-unlimited" || operation == "delete-parent" {
					testcloud.JSON(w, 404, `{"error":"gone"}`)
					return
				}
				testcloud.JSON(w, 200, `{"notification":{"notification_uuid":"`+notificationUUID+`","status":"succeeded"}}`)
			})
			ctx := context.Background()
			if operation == "delete-parent" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
				defer cancel()
			}
			var err error
			switch operation {
			case "status-unlimited":
				_, err = api.status(ctx, resource.ID(api.uuid), "succeeded")
			case "status-override":
				_, err = api.status(ctx, resource.ID(api.uuid), "succeeded", resource.WithTimeout(20*time.Second))
			case "delete-default", "delete-parent":
				err = api.deleted(ctx, resource.ID(api.uuid))
			case "delete-unlimited":
				err = api.deleted(ctx, resource.ID(api.uuid), resource.WithUnlimitedWait())
			case "common-wait":
				_, err = notifications.New(client).Resources.Wait(ctx, resource.ID(api.uuid), "succeeded")
			}
			if err != nil || calls.Load() != 1 {
				t.Fatalf("calls=%d err=%v", calls.Load(), err)
			}
		})
	}
}

func TestInstanceHAWaitErrorOnlyDefaultsAndFailureReplacement(t *testing.T) {
	for _, initial := range []string{"ERROR", "failed"} {
		for _, override := range []string{"default", "failed-only", "disabled", "target"} {
			t.Run(initial+"/"+override, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("instance-ha", "/v1")
				client.Microversion = "1.3"
				api := instanceHAWaitAPIs(t, client)[3]
				var calls atomic.Int32
				cloud.Mux.HandleFunc("/v1/"+api.path, func(w http.ResponseWriter, r *http.Request) {
					state := initial
					if calls.Add(1) > 1 {
						state = "succeeded"
					}
					testcloud.JSON(w, 200, fmt.Sprintf(`{"vmove":{"uuid":"%s","status":%q}}`, vmoveUUID, state))
				})
				opts := []resource.WaitOption{resource.WithPollInterval(time.Millisecond)}
				target := "succeeded"
				switch override {
				case "failed-only":
					opts = append(opts, resource.WithFailureStates("failed"))
				case "disabled":
					opts = append(opts, resource.WithFailureStates())
				case "target":
					target = initial
				}
				value, err := api.status(context.Background(), resource.ID(api.uuid), target, opts...)
				failure := override == "default" && initial == "ERROR" || override == "failed-only" && initial == "failed"
				if failure {
					if value != nil || !errors.Is(err, resource.ErrFailedState) || calls.Load() != 1 {
						t.Fatalf("value=%v err=%v calls=%d", value, err, calls.Load())
					}
				} else if err != nil || value == nil {
					t.Fatalf("value=%v err=%v", value, err)
				}
			})
		}
	}
}

func TestInstanceHAWaitRevalidatesVersionAndKeepsNativeAndDecodeFailures(t *testing.T) {
	for _, scenario := range []string{"changed-version", "forbidden", "malformed", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("instance-ha", "/v1")
			client.Microversion = "1.3"
			api := instanceHAWaitAPIs(t, client)[3]
			var calls atomic.Int32
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cloud.Mux.HandleFunc("/v1/"+api.path, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Request-ID", "retained")
				switch scenario {
				case "changed-version":
					client.Microversion = "1.2"
					testcloud.JSON(w, 200, `{"vmove":{"uuid":"`+vmoveUUID+`","status":"ongoing"}}`)
				case "forbidden":
					testcloud.JSON(w, 403, `{"error":"denied"}`)
				case "malformed":
					testcloud.JSON(w, 200, `{"vmove":[]}`)
				case "cancelled":
					testcloud.JSON(w, 200, `{"vmove":{"uuid":"`+vmoveUUID+`","status":"ongoing"}}`)
				}
			})
			opts := []resource.WaitOption{resource.WithPollInterval(time.Millisecond)}
			if scenario == "cancelled" {
				opts = append(opts, resource.WithProgressCallback(func(int) { cancel() }))
			}
			value, err := api.status(ctx, resource.ID(api.uuid), "succeeded", opts...)
			if value != nil || err == nil || calls.Load() != 1 {
				t.Fatalf("value=%v error=%v calls=%d", value, err, calls.Load())
			}
			switch scenario {
			case "changed-version":
				if !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal(err)
				}
			case "forbidden":
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != `{"error":"denied"}` || native.ResponseHeader.Get("X-Request-ID") != "retained" {
					t.Fatalf("native=%+v err=%v", native, err)
				}
			case "malformed":
				var response *resource.ResponseError
				if !errors.As(err, &response) || response.StatusCode != 200 || string(response.Body) != `{"vmove":[]}` || response.Header.Get("X-Request-ID") != "retained" {
					t.Fatalf("response=%+v err=%v", response, err)
				}
			case "cancelled":
				if !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
			}
		})
	}
}
