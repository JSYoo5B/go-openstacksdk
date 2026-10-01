package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/dns/v2/quotas"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

type designateQuotaTransport func(*http.Request) (*http.Response, error)

func (f designateQuotaTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDesignateQuotaRetriesAndReauthKeepFixedProjectAndSnapshottedBody(t *testing.T) {
	for _, mode := range []string{"reauth", "backoff", "retry"} {
		for _, operation := range []string{"Get", "Update", "Reset"} {
			t.Run(mode+operation, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls, callbacks, transports atomic.Int32
				zero, large := 0, int(^uint(0)>>1)
				option := quotas.WithQuotaOptions(quotas.UpdateOpts{Zones: &zero, ZoneRecords: &large})
				zero, large = 77, 88
				cloud.Mux.HandleFunc("/dns/v2/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) {
					n := calls.Add(1)
					if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Sudo-Project-ID") != "project-fixed" {
						t.Error(r.URL, r.Header)
					}
					if operation == "Update" {
						var body map[string]json.RawMessage
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Error(err)
						}
						if len(body) != 2 || string(body["zones"]) != "0" || string(body["zone_records"]) != strconv.Itoa(int(^uint(0)>>1)) {
							t.Errorf("retry PATCH body=%s", body)
						}
					}
					if n == 1 {
						code := map[string]int{"reauth": 401, "backoff": 429, "retry": 503}[mode]
						testcloud.JSON(w, code, `{"error":"try again"}`)
						return
					}
					if mode == "reauth" && r.Header.Get("X-Auth-Token") != "fresh-token" {
						t.Error(r.Header)
					}
					if operation == "Reset" {
						w.WriteHeader(204)
					} else {
						testcloud.JSON(w, 200, `{"zones":`+strconv.Itoa(int(^uint(0)>>1))+`}`)
					}
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("retry escaped target: %s %s", r.Method, r.URL) })
				original := cloud.Provider.HTTPClient.Transport
				cloud.Provider.HTTPClient.Transport = designateQuotaTransport(func(r *http.Request) (*http.Response, error) {
					transports.Add(1)
					return original.RoundTrip(r)
				})
				mutateInputs := func() {
					callbacks.Add(1)
					zero, large = 555, 666
				}
				switch mode {
				case "reauth":
					cloud.Provider.ReauthFunc = func(ctx context.Context) error { mutateInputs(); cloud.Provider.SetToken("fresh-token"); return nil }
				case "backoff":
					cloud.Provider.RetryBackoffFunc = func(ctx context.Context, response *gophercloud.ErrUnexpectedResponseCode, err error, count uint) error {
						mutateInputs()
						if response.Actual != 429 || !strings.HasSuffix(response.URL, "/quotas/project-fixed") || count != 1 {
							t.Error(response, count)
						}
						return nil
					}
				case "retry":
					cloud.Provider.RetryFunc = func(ctx context.Context, method, endpoint string, opts *gophercloud.RequestOpts, err error, count uint) error {
						mutateInputs()
						if !strings.HasSuffix(endpoint, "/quotas/project-fixed") || count != 1 {
							t.Error(method, endpoint, count)
						}
						return nil
					}
				}
				scope := newDesignateQuotaScope(t, cloud)
				var err error
				if operation == "Update" {
					var value *quotas.QuotaResource
					value, err = scope.Update(context.Background(), quotas.UpdateOpts{}, option)
					if err == nil && (value.Zones != int(^uint(0)>>1) || string(value.Body["zones"]) != strconv.Itoa(int(^uint(0)>>1))) {
						t.Fatal("response integer rounded", value)
					}
				} else {
					err = callDesignateQuota(scope, operation, context.Background())
				}
				if err != nil || calls.Load() != 2 || callbacks.Load() != 1 || transports.Load() != 2 || scope.ProjectID() != "project-fixed" {
					t.Fatal(err, calls.Load(), callbacks.Load(), transports.Load())
				}
			})
		}
	}
}

func TestDesignateQuotaRedirectCannotResetAnotherProjectOrCurrentProject(t *testing.T) {
	for _, tc := range []struct {
		name, location string
		status         int
		mutate         func(*http.Request)
	}{
		{"other project", "/dns/v2/quotas/other-project", 307, nil},
		{"current project collection", "/dns/v2/quotas/", 307, nil},
		{"change method", "/dns/v2/quotas/project-fixed", 303, nil},
		{"opaque policy", "/dns/v2/quotas/project-fixed", 307, func(r *http.Request) { r.URL.Opaque = "/dns/v2/quotas/other-project" }},
		{"host policy", "/dns/v2/quotas/project-fixed", 307, func(r *http.Request) { r.Host = "other-project.invalid" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, policy, transports atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					w.Header().Set("Location", tc.location)
					w.WriteHeader(tc.status)
				} else {
					t.Errorf("unsafe redirect reached HTTP: %s %s", r.Method, r.URL)
					w.WriteHeader(204)
				}
			})
			original := cloud.Provider.HTTPClient.Transport
			cloud.Provider.HTTPClient.Transport = designateQuotaTransport(func(r *http.Request) (*http.Response, error) { transports.Add(1); return original.RoundTrip(r) })
			cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
				policy.Add(1)
				if tc.mutate != nil {
					tc.mutate(next)
				}
				return nil
			}
			value, err := newDesignateQuotaScope(t, cloud).Reset(context.Background())
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 || policy.Load() != 1 || transports.Load() != 1 {
				t.Fatal(value, err, calls.Load(), policy.Load(), transports.Load())
			}
		})
	}
}

func TestDesignateQuotaPreservesSafeRedirectAndCallerRejection(t *testing.T) {
	for _, reject := range []bool{false, true} {
		cloud := testcloud.New(t)
		var calls, policy atomic.Int32
		sentinel := errors.New("caller redirect rejected")
		cloud.Mux.HandleFunc("/dns/v2/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) {
			if calls.Add(1) == 1 {
				w.Header().Set("Location", r.URL.String())
				w.WriteHeader(307)
			} else {
				if r.Method != http.MethodDelete || r.Header.Get("X-Auth-Sudo-Project-ID") != "project-fixed" {
					t.Error(r.Method, r.Header)
				}
				w.WriteHeader(204)
			}
		})
		cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
			policy.Add(1)
			if reject {
				return sentinel
			}
			return nil
		}
		value, err := newDesignateQuotaScope(t, cloud).Reset(context.Background())
		if reject {
			if value != nil || !errors.Is(err, sentinel) || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
		} else if value == nil || err != nil || calls.Load() != 2 {
			t.Fatal(value, err, calls.Load())
		}
		if policy.Load() != 1 {
			t.Fatal("source redirect policy was not retained", policy.Load())
		}
	}
}

func TestDesignateQuotaCancellationDoesNotWaitForSharedReauth(t *testing.T) {
	cloud := testcloud.New(t)
	started, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	var requests atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { requests.Add(1); testcloud.JSON(w, 200, `{}`) })
	cloud.Provider.ReauthFunc = func(context.Context) error { close(started); <-release; return nil }
	go func() { done <- cloud.Provider.Reauthenticate(context.Background(), "test-token") }()
	<-started
	scope := newDesignateQuotaScope(t, cloud)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	_, err := scope.Get(ctx)
	close(release)
	if cause := <-done; cause != nil {
		t.Fatal(cause)
	}
	if !errors.Is(err, context.DeadlineExceeded) || requests.Load() != 0 {
		t.Fatal(err, requests.Load())
	}
}
