package api_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/limits"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type novaLimitsTransport func(*http.Request) (*http.Response, error)

func (f novaLimitsTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestNovaLimitsRetriesAndReauthKeepSingleTenantQueryAndSourceTransport(t *testing.T) {
	for _, mode := range []string{"reauth", "backoff", "retry"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, callbacks, transports atomic.Int32
			const projectID = "project+tag&scope=chosen"
			query := (url.Values{"tenant_id": []string{projectID}, "reserved": []string{"0"}}).Encode()
			cloud.Mux.HandleFunc("/nova/limits", func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				if r.Method != http.MethodGet || r.URL.RawQuery != query || len(r.URL.Query()["tenant_id"]) != 1 {
					t.Error(r.Method, r.URL)
				}
				if n == 1 {
					code := map[string]int{"reauth": 401, "backoff": 429, "retry": 503}[mode]
					testcloud.JSON(w, code, `{"error":"retry limits"}`)
					return
				}
				if mode == "reauth" && r.Header.Get("X-Auth-Token") != "project-token" {
					t.Error(r.Header)
				}
				testcloud.JSON(w, 200, novaLimitsBody)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("retry changed route=%s", r.URL) })
			original := cloud.Provider.HTTPClient.Transport
			cloud.Provider.HTTPClient.Transport = novaLimitsTransport(func(r *http.Request) (*http.Response, error) { transports.Add(1); return original.RoundTrip(r) })
			switch mode {
			case "reauth":
				cloud.Provider.ReauthFunc = func(context.Context) error {
					callbacks.Add(1)
					setQuotaProjectAuth(t, cloud.Provider, "changed-project")
					return nil
				}
			case "backoff":
				cloud.Provider.RetryBackoffFunc = func(ctx context.Context, response *gophercloud.ErrUnexpectedResponseCode, err error, count uint) error {
					callbacks.Add(1)
					if response.Actual != 429 || response.URL != cloud.Server.URL+"/nova/limits?"+query || count != 1 {
						t.Error(response, count)
					}
					return nil
				}
			case "retry":
				cloud.Provider.RetryFunc = func(ctx context.Context, method, endpoint string, opts *gophercloud.RequestOpts, err error, count uint) error {
					callbacks.Add(1)
					if method != http.MethodGet || endpoint != cloud.Server.URL+"/nova/limits?"+query || count != 1 {
						t.Error(method, endpoint, count)
					}
					return nil
				}
			}
			scope, err := limits.New(cloud.Client("compute", "/nova")).InProject(context.Background(), resource.ID(projectID))
			if err != nil {
				t.Fatal(err)
			}
			value, err := scope.Get(context.Background(), limits.WithGetReserved(false))
			if err != nil || value.ProjectID != projectID || scope.ProjectID() != projectID || calls.Load() != 2 || callbacks.Load() != 1 || transports.Load() != 2 {
				t.Fatal(value, err, calls.Load(), callbacks.Load(), transports.Load())
			}
		})
	}
}

func TestNovaLimitsRedirectCannotDropProjectChangeOriginOrMethod(t *testing.T) {
	for _, target := range []string{"current project", "other project", "external origin", "policy method"} {
		t.Run(target, func(t *testing.T) {
			cloud, other := testcloud.New(t), testcloud.New(t)
			var calls, unsafe, policy, transports atomic.Int32
			other.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { unsafe.Add(1); testcloud.JSON(w, 200, novaLimitsBody) })
			location := "/nova/limits"
			switch target {
			case "other project":
				location += "?tenant_id=other-project"
			case "external origin":
				location = other.Server.URL + "/nova/limits?tenant_id=project-fixed"
			case "policy method":
				location += "?tenant_id=project-fixed"
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					w.Header().Set("Location", location)
					w.WriteHeader(307)
				} else {
					unsafe.Add(1)
					testcloud.JSON(w, 200, novaLimitsBody)
				}
			})
			original := cloud.Provider.HTTPClient.Transport
			cloud.Provider.HTTPClient.Transport = novaLimitsTransport(func(r *http.Request) (*http.Response, error) { transports.Add(1); return original.RoundTrip(r) })
			cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
				policy.Add(1)
				if target == "policy method" {
					next.Method = http.MethodPost
				}
				return nil
			}
			value, err := newNovaLimitsScope(t, cloud).Get(context.Background())
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || unsafe.Load() != 0 || calls.Load() != 1 || transports.Load() != 1 || policy.Load() != 1 {
				t.Fatal(value, err, calls.Load(), unsafe.Load(), transports.Load(), policy.Load())
			}
		})
	}
}

func TestNovaLimitsSafeRedirectAndCallerPolicyRejectionArePreserved(t *testing.T) {
	for _, reject := range []bool{false, true} {
		cloud := testcloud.New(t)
		var calls, policy atomic.Int32
		sentinel := errors.New("caller denied limits redirect")
		cloud.Mux.HandleFunc("/nova/limits", func(w http.ResponseWriter, r *http.Request) {
			if !reflect.DeepEqual(r.URL.Query()["tenant_id"], []string{"project-fixed"}) {
				t.Error(r.URL)
			}
			if calls.Add(1) == 1 {
				w.Header().Set("Location", r.URL.String())
				w.WriteHeader(307)
			} else {
				testcloud.JSON(w, 200, novaLimitsBody)
			}
		})
		cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
			policy.Add(1)
			if reject {
				return sentinel
			}
			return nil
		}
		value, err := newNovaLimitsScope(t, cloud).Get(context.Background())
		if reject {
			if value != nil || !errors.Is(err, sentinel) || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
		} else if err != nil || value == nil || value.ProjectID != "project-fixed" || calls.Load() != 2 {
			t.Fatal(value, err, calls.Load())
		}
		if policy.Load() != 1 {
			t.Fatal("source redirect policy lost", policy.Load())
		}
	}
}

func TestNovaLimitsRetryAndTransportFailuresPreserveSourceErrorsWithoutFallback(t *testing.T) {
	t.Run("retry rejection", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls, callbacks atomic.Int32
		sentinel := errors.New("caller stopped limits retry")
		cloud.Mux.HandleFunc("/nova/limits", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			w.Header().Set("X-Openstack-Request-Id", "retry-error")
			testcloud.JSON(w, 503, `{"error":"limits unavailable"}`)
		})
		cloud.Provider.RetryFunc = func(ctx context.Context, method, endpoint string, opts *gophercloud.RequestOpts, err error, count uint) error {
			callbacks.Add(1)
			var response gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &response) || response.Actual != 503 || string(response.Body) != `{"error":"limits unavailable"}` || response.ResponseHeader.Get("X-Openstack-Request-Id") != "retry-error" || count != 1 {
				t.Error(response, err, count)
			}
			return sentinel
		}
		if value, err := newNovaLimitsScope(t, cloud).Get(context.Background()); value != nil || !errors.Is(err, sentinel) || calls.Load() != 1 || callbacks.Load() != 1 {
			t.Fatal(value, err, calls.Load(), callbacks.Load())
		}
	})
	t.Run("transport rejection", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		sentinel := errors.New("source transport rejected limits")
		cloud.Provider.HTTPClient.Transport = novaLimitsTransport(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.URL.Query().Get("tenant_id") != "project-fixed" {
				t.Error(r.URL)
			}
			return nil, sentinel
		})
		if value, err := newNovaLimitsScope(t, cloud).Get(context.Background()); value != nil || !errors.Is(err, sentinel) || calls.Load() != 1 {
			t.Fatal(value, err, calls.Load())
		}
	})
}

func TestNovaLimitsCanceledCallerDoesNotWaitForSharedReauth(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	started, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, novaLimitsBody) })
	cloud.Provider.ReauthFunc = func(context.Context) error { close(started); <-release; return nil }
	go func() { done <- cloud.Provider.Reauthenticate(context.Background(), "test-token") }()
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	value, err := newNovaLimitsScope(t, cloud).Get(ctx)
	close(release)
	if cause := <-done; cause != nil {
		t.Fatal(cause)
	}
	if value != nil || !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 0 {
		t.Fatal(value, err, calls.Load())
	}
}
