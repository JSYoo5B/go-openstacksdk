package compute_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/compute"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

func TestAutomaticCreatePreservesBootAndNativeBodyOptions(t *testing.T) {
	for _, scenario := range []string{"image", "existing volume", "new volume"} {
		t.Run(scenario, func(t *testing.T) {
			f := newAutomaticFixture(t)
			f.service.RawClient().Microversion = "2.67"
			o := automaticCreateOptions()
			o.AutomaticIP = append(o.AutomaticIP, compute.WithAutomaticIPEnabled(false))
			o.Server = append(o.Server, compute.WithUserData([]byte("hello")), compute.WithMetadata(map[string]string{"role": "web"}), compute.WithSecurityGroups("default"), compute.WithField("vendor", map[string]any{"value": false}), compute.WithNetworkInterfaces(compute.ServerNetworkInterface{Network: resource.ID("private"), Port: resource.ID("port"), FixedIP: "10.0.0.10"}))
			request := automaticCreateRequest()
			if scenario == "existing volume" {
				request.Image = resource.Ref{}
				o.Server = append(o.Server, compute.WithBootVolume(resource.ID("volume")))
			}
			if scenario == "new volume" {
				o.Server = append(o.Server, compute.WithBootVolumeSize(20), compute.WithBootVolumeType("fast"))
			}
			posts := addAutomaticCreate(f, `{"server":{"id":"server"}}`, 202, func(body map[string]any) {
				if body["user_data"] != "aGVsbG8=" || !reflect.DeepEqual(body["metadata"], map[string]any{"role": "web"}) || !reflect.DeepEqual(body["security_groups"], []any{map[string]any{"name": "default"}}) || !reflect.DeepEqual(body["vendor"], map[string]any{"value": false}) || !reflect.DeepEqual(body["networks"], []any{map[string]any{"uuid": "private", "port": "port", "fixed_ip": "10.0.0.10"}}) {
					t.Error(body)
				}
				if scenario == "image" {
					if body["imageRef"] != "image" || body["block_device_mapping_v2"] != nil {
						t.Error(body)
					}
					return
				}
				if body["imageRef"] != "" {
					t.Error(body)
				}
				rows, ok := body["block_device_mapping_v2"].([]any)
				if !ok || len(rows) != 1 {
					t.Error(body)
					return
				}
				row := rows[0].(map[string]any)
				if row["destination_type"] != "volume" || row["boot_index"] != float64(0) || row["delete_on_termination"] != false {
					t.Error(row)
				}
				if scenario == "existing volume" {
					if row["source_type"] != "volume" || row["uuid"] != "volume" {
						t.Error(row)
					}
				} else if row["source_type"] != "image" || row["uuid"] != "image" || row["volume_size"] != float64(20) || row["volume_type"] != "fast" {
					t.Error(row)
				}
			})
			result, err := f.service.CreateWithAutomaticFloatingIP(context.Background(), request, o)
			if err != nil || result == nil || posts.Load() != 1 || f.posts.Load() != 0 || f.roles.Load() != 0 {
				t.Fatal(result, err, posts.Load(), f.posts.Load(), f.roles.Load())
			}
		})
	}
}

func TestAutomaticCreateAcceptedPOSTAndGETCloseKeepResources(t *testing.T) {
	for _, scenario := range []string{"POST error", "POST cancel", "POST source", "GET error", "ancestor POST source"} {
		t.Run(scenario, func(t *testing.T) {
			f := newAutomaticFixture(t)
			posts := addAutomaticCreate(f, `{"server":{"id":"server","status":"BUILD","adminPass":"creation-secret"}}`, 202, nil)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("accepted pipeline stopped")
			outerChanged := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerChanged {
					return cause
				}
				return nil
			})
			base := f.cloud.Provider.HTTPClient.Transport
			f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
				response, err := base.RoundTrip(r)
				matches := r.Method == "POST" && r.URL.Path == "/v2.1/servers"
				if scenario == "GET error" {
					matches = r.URL.Path == "/v2.1/servers/server"
				}
				if err == nil && matches {
					response.Header.Set("X-Creation-Proof", "accepted")
					response.Body = automaticCloseBody{ReadCloser: response.Body, close: func() error {
						switch scenario {
						case "POST cancel":
							cancel(cause)
							return nil
						case "POST source":
							f.service.API = nil
							return nil
						case "ancestor POST source":
							outerChanged = true
							return nil
						default:
							return cause
						}
					}}
				}
				return response, err
			})
			retries := 0
			f.cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return cause
			}
			o := automaticCreateOptions()
			o.AutomaticIP = append(o.AutomaticIP, compute.WithAutomaticIPEnabled(false))
			result, err := f.service.CreateWithAutomaticFloatingIP(ctx, automaticCreateRequest(), o)
			var response *resource.ResponseError
			if result == nil || result.Creation == nil || result.Creation.AdminPass != "creation-secret" || result.Server == nil || !errors.As(err, &response) || response.Header.Get("X-Creation-Proof") != "accepted" || retries != 0 || posts.Load() != 1 || f.posts.Load() != 0 {
				t.Fatal(result, err, response, retries, posts.Load(), f.posts.Load())
			}
			if scenario == "GET error" {
				if result.Server.Status != "ACTIVE" || response.StatusCode != 200 || f.raw.Load() != 1 {
					t.Fatal(result, err, f.raw.Load())
				}
			} else if result.Server.Status != "BUILD" || response.StatusCode != 202 || f.raw.Load() != 0 {
				t.Fatal(result, err, f.raw.Load())
			}
			if scenario == "POST cancel" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if scenario == "ancestor POST source" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
		})
	}
}

func TestAutomaticCreateSharesDeadlineAndKeepsAssignmentOnConvergenceCancel(t *testing.T) {
	f := newAutomaticFixture(t)
	posts := addAutomaticCreate(f, `{"server":{"id":"server"}}`, 202, nil)
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("stop convergence")
	o := automaticCreateOptions()
	o.AutomaticIP = append(o.AutomaticIP, compute.WithAutomaticIPTimeout(250*time.Millisecond), compute.WithAutomaticIPProgress(func(*compute.Server) error { cancel(cause); return nil }))
	f.rawBody = func(int32) (int, string) {
		return 200, `{"server":{"id":"server","status":"ACTIVE","addresses":` + autoFixed + `}}`
	}
	var deadlines []time.Time
	base := f.cloud.Provider.HTTPClient.Transport
	f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
		deadline, ok := r.Context().Deadline()
		if !ok {
			t.Error("missing deadline", r.URL)
		}
		deadlines = append(deadlines, deadline)
		return base.RoundTrip(r)
	})
	result, err := f.service.CreateWithAutomaticFloatingIP(ctx, automaticCreateRequest(), o)
	if !errors.Is(err, cause) || result == nil || result.Automatic == nil || result.Automatic.Assignment == nil || !result.Automatic.Assignment.Allocated || result.Automatic.Observed || posts.Load() != 1 || f.posts.Load() != 1 || f.raw.Load() != 2 {
		t.Fatal(result, err, posts.Load(), f.posts.Load(), f.raw.Load())
	}
	if len(deadlines) < 5 {
		t.Fatal(deadlines)
	}
	for _, deadline := range deadlines {
		if !deadline.Equal(deadlines[0]) {
			t.Fatal("budget restarted", deadlines)
		}
	}
}
