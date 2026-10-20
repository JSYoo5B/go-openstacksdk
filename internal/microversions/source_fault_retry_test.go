package microversions

import (
	"context"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
	"net/http"
	"strings"
	"testing"
)

func TestSourceDiscoveryStopsFaultRetryHooksWithoutChangingCleanRetry(t *testing.T) {
	for _, status := range []int{401, 429, 503} {
		for _, physical := range []bool{false, true} {
			cloud := testcloud.New(t)
			client := cloud.Client("compute", "/v2.1/")
			readErr, closeErr := errors.New("discovery partial read"), errors.New("discovery close")
			body := &discoveryFaultBody{reader: strings.NewReader(`{"error":"wire"}`)}
			if physical {
				body.readErr = readErr
				body.closeErr = closeErr
			}
			calls, hooks := 0, 0
			hookErr := errors.New("clean discovery hook")
			cloud.Provider.HTTPClient.Transport = transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: status, Header: http.Header{"X-Proof": []string{"discovery"}}, Body: body, Request: r}, nil
			})
			cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				hooks++
				return hookErr
			}
			cloud.Provider.ReauthFunc = func(context.Context) error { hooks++; return hookErr }
			cloud.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				hooks++
				return hookErr
			}
			source, err := cloudread.Capture(context.Background(), client, "compute")
			if err != nil {
				t.Fatal(err)
			}
			profile := NovaProfile
			profile.StopFaultRetryCallbacks = true
			_, err = Read(context.Background(), source, profile)
			if err == nil || calls != 1 || body.closes != 1 {
				t.Fatal(status, physical, err, calls, body.closes)
			}
			if physical {
				original := err
				if status == 401 {
					var auth *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &auth) || !errors.Is(auth.ErrReauth, readErr) || !errors.Is(auth.ErrReauth, closeErr) {
						t.Fatal(err, auth)
					}
					original = auth.ErrOriginal
				}
				var proof gophercloud.ErrUnexpectedResponseCode
				if hooks != 0 || !errors.Is(err, readErr) || !errors.Is(err, closeErr) || !errors.As(original, &proof) || proof.Actual != status || string(proof.Body) != `{"error":"wire"}` || proof.ResponseHeader.Get("X-Proof") != "discovery" || proof.Method != "GET" {
					t.Fatal(status, err, hooks, proof)
				}
			} else {
				cause := err
				var auth *gophercloud.ErrUnableToReauthenticate
				if errors.As(err, &auth) && auth != nil {
					cause = auth.ErrReauth
				}
				if hooks == 0 || !errors.Is(cause, hookErr) {
					t.Fatal(status, err, hooks)
				}
			}
		}
	}
}
