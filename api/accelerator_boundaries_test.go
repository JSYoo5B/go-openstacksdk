package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/accelerator/v2/deployables"
	"gophercloudsdk/accelerator/v2/devices"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestAcceleratorEmptyPageContinuesAndCycles(t *testing.T) {
	for _, cycle := range []bool{false, true} {
		t.Run(fmt.Sprint(cycle), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/v2/deployables", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("page") == "2" && !cycle {
					testcloud.JSON(w, 200, `{"deployables":[{"uuid":"after-empty","name":"target"}]}`)
					return
				}
				testcloud.JSON(w, 200, `{"deployables":[],"next":"?page=2"}`)
			})
			values, err := deployables.New(cloud.Client("accelerator", "/v2")).All(context.Background())
			if cycle {
				if !errors.Is(err, resource.ErrPaginationCycle) {
					t.Fatal(err)
				}
			} else if err != nil || len(values) != 1 || values[0].UUID != "after-empty" {
				t.Fatalf("values: %+v/%v", values, err)
			}
			if calls.Load() != 2 {
				t.Fatalf("calls=%d", calls.Load())
			}
		})
	}
}

func TestAcceleratorRedirectOriginAndCallerPolicy(t *testing.T) {
	foreign := testcloud.New(t)
	var foreignCalls atomic.Int32
	foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		foreignCalls.Add(1)
		t.Errorf("foreign request received token %q", r.Header.Get("X-Auth-Token"))
		testcloud.JSON(w, 200, `{"deployables":[]}`)
	})
	cloud := testcloud.New(t)
	var policyCalls atomic.Int32
	cloud.Provider.HTTPClient.CheckRedirect = func(r *http.Request, via []*http.Request) error { policyCalls.Add(1); return nil }
	cloud.Mux.HandleFunc("/v2/deployables", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"deployables":[{"uuid":"one"}],"next":"/v2/page2"}`)
	})
	cloud.Mux.HandleFunc("/v2/page2", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, foreign.Server.URL+"/v2/foreign", 302)
	})
	a := deployables.New(cloud.Client("accelerator", "/v2"))
	if _, err := a.All(context.Background()); err == nil {
		t.Fatal("foreign redirect accepted")
	}
	if foreignCalls.Load() != 0 || policyCalls.Load() != 1 || a.RawClient().ProviderClient != cloud.Provider {
		t.Fatalf("foreign/policy/shared: %d/%d", foreignCalls.Load(), policyCalls.Load())
	}
	cloud.Provider.HTTPClient.CheckRedirect = func(r *http.Request, via []*http.Request) error { policyCalls.Add(1); return http.ErrUseLastResponse }
	if _, err := a.All(context.Background()); !gophercloud.ResponseCodeIs(err, 302) {
		t.Fatalf("caller policy ignored: %v", err)
	}
	if foreignCalls.Load() != 0 || policyCalls.Load() != 2 {
		t.Fatal("caller policy/origin changed")
	}
}

func TestAcceleratorGuardSharesReauthenticationAndRetries(t *testing.T) {
	cloud := testcloud.New(t)
	var attempts, reauths, backoffs atomic.Int32
	cloud.Provider.ReauthFunc = func(ctx context.Context) error {
		reauths.Add(1)
		cloud.Provider.SetToken("refreshed-token")
		return nil
	}
	cloud.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
		backoffs.Add(1)
		return nil
	}
	cloud.Mux.HandleFunc("/v2/deployables", func(w http.ResponseWriter, r *http.Request) {
		switch attempts.Add(1) {
		case 1:
			if r.Header.Get("X-Auth-Token") != "test-token" {
				t.Error(r.Header)
			}
			testcloud.JSON(w, 401, `{"error":"expired"}`)
		case 2:
			if r.Header.Get("X-Auth-Token") != "refreshed-token" {
				t.Error(r.Header)
			}
			testcloud.JSON(w, 429, `{"error":"limited"}`)
		default:
			if r.Header.Get("X-Auth-Token") != "refreshed-token" {
				t.Error(r.Header)
			}
			testcloud.JSON(w, 200, `{"deployables":[{"uuid":"one"}]}`)
		}
	})
	cloud.Mux.HandleFunc("/v2/deployables/one", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Token") != "refreshed-token" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"uuid":"one"}`)
	})
	a := deployables.New(cloud.Client("accelerator", "/v2"))
	values, err := a.All(context.Background())
	if err != nil || len(values) != 1 {
		t.Fatalf("list: %v/%v", values, err)
	}
	if _, err := a.Get(context.Background(), "one"); err != nil {
		t.Fatal(err)
	}
	if attempts.Load() != 3 || reauths.Load() != 1 || backoffs.Load() != 1 || cloud.Provider.Token() != "refreshed-token" || cloud.Provider.HTTPClient.CheckRedirect != nil {
		t.Fatalf("shared policy: %d/%d/%d", attempts.Load(), reauths.Load(), backoffs.Load())
	}
}

func TestAcceleratorActionAndStatusVersionsAreSeparate(t *testing.T) {
	cloud := testcloud.New(t)
	var posts, gets atomic.Int32
	cloud.Mux.HandleFunc("/v2/devices/d1/enable", func(w http.ResponseWriter, r *http.Request) { posts.Add(1); w.WriteHeader(200) })
	cloud.Mux.HandleFunc("/v2/devices/d1", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); testcloud.JSON(w, 200, `{"uuid":"d1"}`) })
	for _, version := range []string{"", "2.0", "2.2"} {
		client := cloud.Client("accelerator", "/v2")
		client.Microversion = version
		a := devices.New(client)
		if _, err := a.Enable(context.Background(), resource.ID("d1")); err != nil {
			t.Fatalf("action version %s: %v", version, err)
		}
		if _, err := a.Wait(context.Background(), resource.ID("d1"), "enabled"); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatalf("wait version %s: %v", version, err)
		}
		if _, err := a.Resources.Wait(context.Background(), resource.ID("d1"), "enabled"); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(err)
		}
	}
	if posts.Load() != 3 || gets.Load() != 0 {
		t.Fatalf("posts/gets=%d/%d", posts.Load(), gets.Load())
	}
}

func TestAcceleratorPaginationUsesRotatedTokenAndReauthenticatesActualToken(t *testing.T) {
	for _, expired := range []bool{false, true} {
		t.Run(fmt.Sprint(expired), func(t *testing.T) {
			cloud := testcloud.New(t)
			var pageCalls, reauths atomic.Int32
			cloud.Provider.ReauthFunc = func(context.Context) error { reauths.Add(1); cloud.Provider.SetToken("reauth-token"); return nil }
			cloud.Mux.HandleFunc("/v2/deployables", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("page") == "2" {
					pageCalls.Add(1)
					token := r.Header.Get("X-Auth-Token")
					if expired && pageCalls.Load() == 1 {
						if token != "rotated-token" {
							t.Errorf("stale token: %q", token)
						}
						testcloud.JSON(w, 401, `{"error":"rotated token expired"}`)
						return
					}
					expected := "rotated-token"
					if expired {
						expected = "reauth-token"
					}
					if token != expected {
						t.Errorf("page2 token: %q", token)
						testcloud.JSON(w, 403, `{}`)
						return
					}
					testcloud.JSON(w, 200, `{"deployables":[{"uuid":"two"}]}`)
					return
				}
				testcloud.JSON(w, 200, `{"deployables":[{"uuid":"one"}],"next":"?page=2"}`)
			})
			count := 0
			for value, err := range deployables.New(cloud.Client("accelerator", "/v2")).List(context.Background()) {
				if err != nil {
					t.Fatal(err)
				}
				count++
				if value.UUID == "one" {
					cloud.Provider.SetToken("rotated-token")
				}
			}
			if count != 2 || (expired && reauths.Load() != 1) || (!expired && reauths.Load() != 0) {
				t.Fatalf("count/reauth=%d/%d", count, reauths.Load())
			}
		})
	}
}

func TestAcceleratorSharedOngoingReauthWaitHonorsCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	started, release, finished := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	cloud.Provider.ReauthFunc = func(context.Context) error {
		close(started)
		<-release
		cloud.Provider.SetToken("after-shared-reauth")
		return nil
	}
	go func() { finished <- cloud.Provider.Reauthenticate(context.Background(), "test-token") }()
	<-started
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/v2/deployables/one", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("X-Auth-Token") != "after-shared-reauth" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"uuid":"one"}`)
	})
	a := deployables.New(cloud.Client("accelerator", "/v2"))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	_, err := a.Get(ctx, "one")
	cancel()
	// Always release the original owner's operation before reporting failure.
	close(release)
	if reauthErr := <-finished; reauthErr != nil {
		t.Fatal(reauthErr)
	}
	if !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 0 {
		t.Fatalf("waiting request: %v/%d", err, calls.Load())
	}
	if _, err := a.Get(context.Background(), "one"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("requests=%d", calls.Load())
	}
}
