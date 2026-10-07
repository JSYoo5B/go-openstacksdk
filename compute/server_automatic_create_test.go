package compute_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/compute"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func automaticCreateRequest() compute.CreateServerRequest {
	return compute.CreateServerRequest{Name: "web", Image: resource.ID("image"), Flavor: resource.ID("flavor")}
}

func automaticCreateOptions() compute.AutomaticServerCreateOptions {
	return compute.AutomaticServerCreateOptions{Server: []compute.CreateServerOption{compute.WithNetworks(resource.ID("private")), compute.WithWait(resource.WithPollInterval(time.Millisecond))}, AutomaticIP: automaticOptions()}
}

func addAutomaticCreate(f *automaticFixture, body string, code int, inspect func(map[string]any)) *atomic.Int32 {
	posts := &atomic.Int32{}
	f.cloud.Mux.HandleFunc("POST /v2.1/servers", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		var request struct{ Server map[string]any }
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			panic(err)
		}
		if inspect != nil {
			inspect(request.Server)
		}
		testcloud.JSON(w, code, body)
	})
	return posts
}

func TestAutomaticCreateWaitsForRawMetadataAndObservesAssignment(t *testing.T) {
	f := newAutomaticFixture(t)
	posts := addAutomaticCreate(f, `{"server":{"id":"server","status":"BUILD","adminPass":"initial-password"}}`, 202, func(body map[string]any) {
		if body["imageRef"] != "image" || body["flavorRef"] != "flavor" || !reflect.DeepEqual(body["networks"], []any{map[string]any{"uuid": "private"}}) {
			t.Error(body)
		}
	})
	f.rawBody = func(n int32) (int, string) {
		if n == 1 {
			return 200, `{"server":{"id":"server","status":"BUILD","progress":25}}`
		}
		if n == 2 {
			return 203, `{"server":{"id":"server","status":"ACTIVE","addresses":null}}`
		}
		if n == 3 {
			return 200, `{"server":{"id":"server","status":"ACTIVE","addresses":` + autoFixed + `}}`
		}
		return 200, `{"server":{"id":"server","status":"ACTIVE","addresses":` + autoFloating + `}}`
	}
	o := automaticCreateOptions()
	var progress []int
	o.Server = append(o.Server, compute.WithWait(resource.WithPollInterval(time.Millisecond), resource.WithProgressCallback(func(n int) { progress = append(progress, n) })))
	result, err := f.service.CreateWithAutomaticFloatingIP(context.Background(), automaticCreateRequest(), o)
	if err != nil || result == nil || result.Creation.AdminPass != "initial-password" || result.Server.AdminPass != "" || result.Server.Status != "ACTIVE" || result.Automatic == nil || !result.Automatic.Observed || result.Automatic.Assignment == nil || posts.Load() != 1 || f.posts.Load() != 1 || f.raw.Load() != 4 || !reflect.DeepEqual(progress, []int{25, 0}) {
		t.Fatal(result, err, posts.Load(), f.posts.Load(), f.raw.Load(), progress)
	}
}

func TestAutomaticCreateKnownSkipsDoNotLoadIPDependencies(t *testing.T) {
	for _, scenario := range []string{"disabled", "private", "source none", "floating", "public"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			service := compute.New(cloud.Client("compute", "/v2.1"), compute.Dependencies{AddressNetworks: func(context.Context) (*network.Service, error) {
				calls++
				return nil, errors.New("eager IP dependency")
			}})
			posts, gets := 0, 0
			cloud.Mux.HandleFunc("POST /v2.1/servers", func(w http.ResponseWriter, r *http.Request) {
				posts++
				testcloud.JSON(w, 202, `{"server":{"id":"server"}}`)
			})
			addresses, access := autoFixed, ""
			o := automaticCreateOptions()
			switch scenario {
			case "disabled":
				o.AutomaticIP = append(o.AutomaticIP, compute.WithAutomaticIPEnabled(false))
			case "private":
				o.AutomaticIP = append(o.AutomaticIP, compute.WithAutomaticAddressOptions(compute.WithPrivateCloud(true)))
			case "source none":
				o.AutomaticIP = append(o.AutomaticIP, compute.WithAutomaticAddressOptions(compute.WithFloatingIPSource(compute.FloatingIPNone)))
			case "floating":
				addresses = autoFloating
			case "public":
				access = `,"accessIPv4":"8.8.4.4"`
			}
			cloud.Mux.HandleFunc("GET /v2.1/servers/server", func(w http.ResponseWriter, r *http.Request) {
				gets++
				testcloud.JSON(w, 200, `{"server":{"id":"server","status":"ACTIVE"`+access+`,"addresses":`+addresses+`}}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Error("unexpected", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			result, err := service.CreateWithAutomaticFloatingIP(context.Background(), automaticCreateRequest(), o)
			if err != nil || result == nil || result.Automatic == nil || result.Automatic.Decision.Needed || result.Automatic.Assignment != nil || calls != 0 || posts != 1 || gets != 1 {
				t.Fatal(result, err, calls, posts, gets)
			}
		})
	}
}

func TestAutomaticCreateWaitErrorsKeepLastMatchingServer(t *testing.T) {
	for _, scenario := range []string{"empty", "nil timeout", "wrong ID", "ERROR", "HTTP", "progress source", "progress cancel"} {
		t.Run(scenario, func(t *testing.T) {
			f := newAutomaticFixture(t)
			posts := addAutomaticCreate(f, `{"server":{"id":"server","status":"BUILD","adminPass":"secret"}}`, 202, nil)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("progress canceled")
			o := automaticCreateOptions()
			o.AutomaticIP = append(o.AutomaticIP, compute.WithAutomaticIPEnabled(false))
			o.Server = append(o.Server, compute.WithWait(resource.WithPollInterval(time.Millisecond), resource.WithTimeout(20*time.Millisecond), resource.WithFailureStates(), resource.WithProgressCallback(func(int) {
				if scenario == "progress source" {
					f.service.API = nil
				}
				if scenario == "progress cancel" {
					cancel(cause)
				}
			})))
			f.rawBody = func(n int32) (int, string) {
				switch scenario {
				case "empty":
					return 200, `{"server":{"id":"server","status":"ACTIVE","addresses":{"private":[]}}}`
				case "nil timeout":
					return 200, `{"server":{"id":"server","status":"ACTIVE","addresses":null}}`
				case "wrong ID":
					return 200, `{"server":{"id":"other","status":"ACTIVE","addresses":` + autoFixed + `}}`
				case "ERROR":
					return 200, `{"server":{"id":"server","status":"ERROR","fault":{"message":"boot failed"}}}`
				case "HTTP":
					if n > 1 {
						return 403, `{"error":"denied"}`
					}
				}
				return 200, `{"server":{"id":"server","status":"BUILD","progress":25}}`
			}
			result, err := f.service.CreateWithAutomaticFloatingIP(ctx, automaticCreateRequest(), o)
			if err == nil || result == nil || result.Creation.AdminPass != "secret" || result.Server.ID != "server" || result.Automatic != nil || posts.Load() != 1 || f.posts.Load() != 0 {
				t.Fatal(result, err, posts.Load(), f.posts.Load())
			}
			switch scenario {
			case "empty":
				if !errors.Is(err, compute.ErrServerAddressesUnavailable) || result.Server.Status != "ACTIVE" {
					t.Fatal(result, err)
				}
			case "nil timeout":
				if !errors.Is(err, context.DeadlineExceeded) || result.Server.Status != "ACTIVE" {
					t.Fatal(result, err)
				}
			case "ERROR":
				if !errors.Is(err, resource.ErrFailedState) || result.Server.Fault.Message != "boot failed" {
					t.Fatal(result, err)
				}
			case "progress cancel":
				if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || f.raw.Load() != 1 {
					t.Fatal(result, err, f.raw.Load())
				}
			case "progress source":
				if !errors.Is(err, resource.ErrInvalidOption) || f.raw.Load() != 1 {
					t.Fatal(result, err, f.raw.Load())
				}
			case "HTTP":
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) || response.Actual != 403 || f.raw.Load() != 2 {
					t.Fatal(result, err, f.raw.Load())
				}
			}
		})
	}
}

func TestAutomaticCreateRejectsInvalidPolicyBeforePOST(t *testing.T) {
	for _, scenario := range []string{"status", "nil automatic", "nil server", "invalid network", "boot conflict"} {
		t.Run(scenario, func(t *testing.T) {
			f := newAutomaticFixture(t)
			posts := addAutomaticCreate(f, `{"server":{"id":"server"}}`, 202, nil)
			o := automaticCreateOptions()
			switch scenario {
			case "status":
				o.Server = append(o.Server, compute.WithWait(resource.WithStatusAttribute("TaskState")))
			case "nil automatic":
				o.AutomaticIP = append(o.AutomaticIP, nil)
			case "nil server":
				o.Server = append(o.Server, nil)
			case "invalid network":
				o.FloatingIPNetwork = resource.ID("../unsafe")
			case "boot conflict":
				o.Server = append(o.Server, compute.WithBootVolume(resource.ID("volume")))
			}
			result, err := f.service.CreateWithAutomaticFloatingIP(context.Background(), automaticCreateRequest(), o)
			if result != nil || err == nil || posts.Load() != 0 || f.raw.Load() != 0 || f.roles.Load() != 0 || f.ports.Load() != 0 {
				t.Fatal(result, err, posts.Load(), f.raw.Load(), f.roles.Load(), f.ports.Load())
			}
		})
	}
}

func TestAutomaticCreateAcceptedResponsePolicyAndDecodeDoNotRetry(t *testing.T) {
	for _, scenario := range []string{"200", "202", "201", "null", "missing", "malformed", "unsafe ID"} {
		t.Run(scenario, func(t *testing.T) {
			f := newAutomaticFixture(t)
			body, code := `{"server":{"id":"server","status":"BUILD"}}`, 202
			switch scenario {
			case "200":
				code = 200
			case "201":
				code = 201
			case "null":
				body = `{"server":null}`
			case "missing":
				body = `{"other":{}}`
			case "malformed":
				body = `{"server":`
			case "unsafe ID":
				body = `{"server":{"id":"../unsafe"}}`
			}
			posts := addAutomaticCreate(f, body, code, nil)
			retries := 0
			f.cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return errors.New("no retry")
			}
			o := automaticCreateOptions()
			o.AutomaticIP = append(o.AutomaticIP, compute.WithAutomaticIPEnabled(false))
			result, err := f.service.CreateWithAutomaticFloatingIP(context.Background(), automaticCreateRequest(), o)
			wantRetries := 0
			if scenario == "201" {
				wantRetries = 1
			}
			if posts.Load() != 1 || retries != wantRetries {
				t.Fatal(posts.Load(), retries, err)
			}
			if scenario == "200" || scenario == "202" {
				if err != nil || result == nil || result.Server.Status != "ACTIVE" {
					t.Fatal(result, err)
				}
				return
			}
			var response *resource.ResponseError
			if err == nil || f.raw.Load() != 0 || f.posts.Load() != 0 {
				t.Fatal(result, err, f.raw.Load(), f.posts.Load())
			}
			if scenario != "201" && (!errors.As(err, &response) || response.StatusCode != 202) {
				t.Fatal(result, err)
			}
			if scenario == "unsafe ID" && (result.Creation == nil || !strings.Contains(result.Creation.ID, "unsafe")) {
				t.Fatal(result, err)
			}
		})
	}
}
