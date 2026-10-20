package api_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestComputeKeypairDeleteOwnedDefaultsOwnerAndNativeCompatibility(t *testing.T) {
	for _, check := range []struct {
		name                              string
		status                            int
		owner, extension                  string
		strict, explicitIgnore, wantError bool
	}{
		{"default current owner", 202, "", "", false, false, false},
		{"escaped explicit owner", 200, "owner+team?&한글", "", false, false, false},
		{"empty owner omitted", 204, "", "", false, false, false},
		{"passive expanded success", 203, "owner", "value+one", false, false, false},
		{"upper success", 399, "", "", false, false, false},
		{"default missing", 404, "", "", false, false, false},
		{"explicit ignored missing", 404, "owner", "", false, true, false},
		{"strict missing", 404, "owner", "", true, false, true},
		{"strict success", 202, "owner", "", true, false, false},
		{"foreign owner forbidden", 403, "foreign-owner", "", false, false, true},
	} {
		t.Run(check.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.1" // The owner is passed literally without a new local gate.
			client.MoreHeaders = map[string]string{"X-Source": "selected"}
			api := keypairs.New(client)
			const name = "key-α+one"
			path := "/reverse/nova/v2.1/project/os-keypairs/" + name
			const body = "passive-not-JSON\x00{"
			query := make(url.Values)
			if check.owner != "" {
				query.Set("user_id", check.owner)
			}
			if check.extension != "" {
				query.Set("vendor", check.extension)
			}
			target := cloud.Server.URL + "/reverse/nova/v2.1/project/os-keypairs/" + url.PathEscape(name)
			if query.Encode() != "" {
				target += "?" + query.Encode()
			}
			var calls, callbacks atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodDelete)
				th.TestHeader(t, r, "X-Auth-Token", "delete-live")
				th.TestHeader(t, r, "X-Source", "selected")
				th.TestHeader(t, r, "X-Extension", "owned")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.1")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.1")
				if r.URL.Path != path || r.URL.EscapedPath() != "/reverse/nova/v2.1/project/os-keypairs/"+url.PathEscape(name) || r.URL.RawQuery != query.Encode() {
					t.Error("DELETE performed lookup or changed fixed member/owner", r.Method, r.URL, query)
				}
				if data, err := io.ReadAll(r.Body); err != nil || len(data) != 0 {
					t.Error("DELETE sent body", string(data), err)
				}
				w.Header().Set("X-Delete-Proof", "actual")
				testcloud.JSON(w, check.status, body)
			})
			input := keypairs.KeypairDeleteOpts{UserID: check.owner}
			if check.strict || check.explicitIgnore {
				input.IgnoreMissing = request.Present(!check.strict)
			}
			captured := keypairs.WithKeypairDeleteOptions(input)
			input.UserID = "caller-changed"
			input.IgnoreMissing = request.Present(true)
			options := []keypairs.KeypairDeleteOption{
				captured,
				keypairs.WithKeypairDeleteHeader("X-Extension", "owned"),
				func(_ *request.Config[keypairs.KeypairDeleteOpts]) error {
					callbacks.Add(1)
					cloud.Provider.SetToken("delete-live")
					return nil
				},
			}
			if check.extension != "" {
				options = append(options, keypairs.WithKeypairDeleteQuery("vendor", check.extension))
			}
			err := api.DeleteKeypair(context.Background(), name, options...)
			if (err != nil) != check.wantError || calls.Load() != 1 || callbacks.Load() != 1 || client.ProviderClient != cloud.Provider || client.Microversion != "2.1" || client.MoreHeaders["X-Source"] != "selected" || len(client.MoreHeaders) != 1 {
				t.Fatal(err, calls.Load(), callbacks.Load(), client)
			}
			if check.wantError {
				var native gophercloud.ErrUnexpectedResponseCode
				var operation *resource.OperationError
				if !errors.As(err, &native) || native.Actual != check.status || len(native.Expected) != 200 || native.Expected[0] != 200 || native.Expected[199] != 399 || native.Method != http.MethodDelete || native.URL != target || string(native.Body) != body || native.ResponseHeader.Get("X-Delete-Proof") != "actual" || !errors.As(err, &operation) || operation.Operation != "DeleteKeypair" || operation.Resource != "keypairs" {
					t.Fatal("native DELETE proof/cause or operation context lost", err, native, operation)
				}
				if errors.Is(err, resource.ErrNotFound) != (check.status == 404) {
					t.Fatal("nonmissing failure reclassified", err)
				}
			}
		})
	}
	for _, check := range []struct {
		lane      string
		status    int
		wantError bool
	}{
		{"native Delete", 202, false}, {"native Delete", 204, false}, {"native Delete", 200, true}, {"native Delete", 404, true},
		{"native Delete owner and query", 202, false},
		{"native Remove default", 404, false}, {"native Remove strict", 404, true},
	} {
		t.Run(fmt.Sprintf("compatibility/%s/%d", check.lane, check.status), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			client.MoreHeaders = map[string]string{"X-Source": "native selected"}
			cloud.Provider.SetToken("native-delete-live")
			const path = "/reverse/nova/v2.1/project/os-keypairs/name"
			const body = "native passive body"
			query := make(url.Values)
			if check.lane == "native Delete owner and query" {
				query = url.Values{"user_id": {"owner+team"}, "vendor": {"value+one"}}
			}
			target := cloud.Server.URL + path
			if query.Encode() != "" {
				target += "?" + query.Encode()
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodDelete)
				th.TestHeader(t, r, "X-Auth-Token", "native-delete-live")
				th.TestHeader(t, r, "X-Source", "native selected")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.55")
				if r.URL.Path != path || r.URL.EscapedPath() != path || r.URL.RawQuery != query.Encode() {
					t.Error("native DELETE changed member or escaped owner/query", r.URL, query)
				}
				if data, err := io.ReadAll(r.Body); err != nil || len(data) != 0 {
					t.Error("native DELETE sent body", string(data), err)
				}
				w.Header().Set("X-Delete-Proof", "native actual")
				testcloud.JSON(w, check.status, body)
			})
			api := keypairs.New(client)
			var err error
			switch check.lane {
			case "native Delete":
				err = api.Delete(context.Background(), "name")
			case "native Delete owner and query":
				err = api.Delete(context.Background(), "name", keypairs.WithDeleteOptions(keypairs.DeleteOpts{UserID: "owner+team"}), keypairs.WithDeleteQuery("vendor", "value+one"))
			case "native Remove default":
				err = api.Remove(context.Background(), resource.ID("name"))
			default:
				err = api.Remove(context.Background(), resource.ID("name"), resource.WithMissingError())
			}
			if (err != nil) != check.wantError || calls.Load() != 1 {
				t.Fatal(err, calls.Load())
			}
			if check.wantError {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != check.status || !reflect.DeepEqual(native.Expected, []int{202, 204}) || native.Method != http.MethodDelete || native.URL != target || string(native.Body) != body || native.ResponseHeader.Get("X-Delete-Proof") != "native actual" {
					t.Fatal("native ABI/status policy or original HTTP proof changed", err, native)
				}
			}
		})
	}
}

func TestComputeKeypairDeleteOwnedPreflightAndOptionOwnership(t *testing.T) {
	for _, mode := range []string{"nil API", "nil client", "nil context", "invalid name", "pre-canceled", "nil option", "null missing policy", "owner query", "missing query", "body field", "auth header", "option cancellation", "option source", "reassigned API"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			api := keypairs.New(client)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause, original := errors.New("delete cancellation"), errors.New("option original")
			name := "name"
			var options []keypairs.KeypairDeleteOption
			var calls, later atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			switch mode {
			case "nil API":
				api = nil
			case "nil client":
				api = keypairs.New(nil)
			case "nil context":
				ctx = nil
			case "invalid name":
				name = "."
			case "pre-canceled":
				cancel(cause)
			case "nil option":
				options = append(options, nil)
			case "null missing policy":
				options = append(options, keypairs.WithKeypairDeleteOptions(keypairs.KeypairDeleteOpts{IgnoreMissing: request.Null[bool]()}))
			case "owner query":
				options = append(options, keypairs.WithKeypairDeleteQuery("user_id", "extension-owner"))
			case "missing query":
				options = append(options, keypairs.WithKeypairDeleteQuery("ignore_missing", "false"))
			case "body field":
				options = append(options, request.WithField[keypairs.KeypairDeleteOpts]("vendor", true))
			case "auth header":
				options = append(options, keypairs.WithKeypairDeleteHeader("X-Auth-Token", "extension-token"))
			case "option cancellation":
				options = append(options, func(_ *request.Config[keypairs.KeypairDeleteOpts]) error { cancel(cause); return original })
			case "option source":
				options = append(options, func(_ *request.Config[keypairs.KeypairDeleteOpts]) error {
					client.ResourceBase = cloud.Server.URL + "/changed/"
					return nil
				})
			case "reassigned API":
				options = append(options, func(_ *request.Config[keypairs.KeypairDeleteOpts]) error {
					*api = *keypairs.New(flavorIdentityClient(cloud))
					return nil
				})
			}
			options = append(options, func(_ *request.Config[keypairs.KeypairDeleteOpts]) error { later.Add(1); return nil })
			err := api.DeleteKeypair(ctx, name, options...)
			if err == nil || calls.Load() != 0 {
				t.Fatal("invalid preflight reached HTTP", err, calls.Load())
			}
			if mode == "pre-canceled" || mode == "option cancellation" {
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || mode == "option cancellation" && !errors.Is(err, original) {
					t.Fatal("context/option cause lost", err)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			switch mode {
			case "null missing policy", "owner query", "missing query", "body field", "auth header":
				if later.Load() != 1 {
					t.Fatal("valid callbacks not applied exactly once before final validation", later.Load())
				}
			default:
				if later.Load() != 0 {
					t.Fatal("guard failure reached later callback", later.Load())
				}
			}
		})
	}
	for _, mode := range []string{"last wins", "bulk resets typed options"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			want := url.Values{"vendor": {"kept"}}
			if mode == "last wins" {
				want.Set("user_id", "last-owner")
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodDelete)
				th.TestHeader(t, r, "X-Extension", "kept")
				if r.URL.Path != "/reverse/nova/v2.1/project/os-keypairs/name" || !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error(r.URL, want)
				}
				w.WriteHeader(404)
			})
			sharedQuery := url.Values{"vendor": {"kept"}}
			sharedHeaders := map[string]string{"X-Extension": "kept"}
			var callbacks atomic.Int32
			options := []keypairs.KeypairDeleteOption{
				keypairs.WithKeypairDeleteIgnoreMissing(false), keypairs.WithKeypairDeleteUserID("first-owner"),
				func(config *request.Config[keypairs.KeypairDeleteOpts]) error {
					callbacks.Add(1)
					config.Query = sharedQuery
					config.Headers = sharedHeaders
					return nil
				},
				func(_ *request.Config[keypairs.KeypairDeleteOpts]) error {
					callbacks.Add(1)
					sharedQuery.Set("vendor", "caller-changed")
					sharedHeaders["X-Extension"] = "caller-changed"
					return nil
				},
			}
			if mode == "last wins" {
				options = append(options, keypairs.WithKeypairDeleteUserID("last-owner"), keypairs.WithKeypairDeleteIgnoreMissing(true))
			} else {
				options = append(options, keypairs.WithKeypairDeleteOptions(keypairs.KeypairDeleteOpts{}))
			}
			if err := keypairs.New(flavorIdentityClient(cloud)).DeleteKeypair(context.Background(), "name", options...); err != nil || calls.Load() != 1 || callbacks.Load() != 2 {
				t.Fatal("options replayed, caller alias retained, or typed clear lost extensions", err, calls.Load(), callbacks.Load())
			}
		})
	}
}

func TestComputeKeypairDeleteAcceptedFailuresAndNativeRetryGuards(t *testing.T) {
	for _, mode := range []string{"read", "read404", "close404", "source", "cancel"} {
		t.Run("accepted "+mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("accepted delete " + mode)
			const body = "accepted-passive-not-JSON"
			var calls, retries atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodDelete)
				if r.URL.Path != "/reverse/nova/v2.1/project/os-keypairs/name" || r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				w.Header().Set("X-Delete-Proof", mode)
				testcloud.JSON(w, 202, body)
			})
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{202, 204}, Method: http.MethodDelete, URL: "nested", Body: []byte("nested native404"), ResponseHeader: http.Header{"X-Nested-Proof": {"original"}}}
			var track *payloadContractTracking
			switch mode {
			case "read":
				track = payloadContractTrack(cloud, cause, nil)
			case "read404":
				track = payloadContractTrack(cloud, nested, nil)
			case "close404":
				track = payloadContractTrack(cloud, nil, nested)
			default:
				base := cloud.Provider.HTTPClient.Transport
				if base == nil {
					base = http.DefaultTransport
				}
				cloud.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					response, err := base.RoundTrip(r)
					if err != nil {
						return response, err
					}
					data, readErr := io.ReadAll(response.Body)
					closeErr := response.Body.Close()
					if readErr != nil || closeErr != nil {
						return nil, errors.Join(readErr, closeErr)
					}
					response.Body = io.NopCloser(bytes.NewReader(data))
					if mode == "source" {
						client.ResourceBase = cloud.Server.URL + "/changed/"
					} else {
						cancel(cause)
					}
					return response, nil
				})
			}
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			err := keypairs.New(client).DeleteKeypair(ctx, "name")
			var proof *resource.ResponseError
			var operation *resource.OperationError
			if err == nil || errors.Is(err, resource.ErrNotFound) || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != body || proof.Header.Get("X-Delete-Proof") != mode || calls.Load() != 1 || retries.Load() != 0 || !errors.As(err, &operation) || operation.Operation != "DeleteKeypair" || operation.Resource != "keypairs" {
				t.Fatal("accepted failure ignored/retried or receipt lost", err, proof, operation, calls.Load(), retries.Load())
			}
			if mode == "read404" || mode == "close404" {
				var original gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &original) || original.Actual != 404 || original.URL != "nested" || string(original.Body) != "nested native404" || original.ResponseHeader.Get("X-Nested-Proof") != "original" {
					t.Fatal("nested original cause lost", err, original)
				}
			} else if mode == "source" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, cause) || mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal("accepted context/cause lost", err)
			}
			if track != nil {
				physical := track.last(t)
				if track.calls.Load() != 1 || physical.reads.Load() == 0 || physical.closes.Load() != 1 {
					t.Fatal(track.calls.Load(), physical.reads.Load(), physical.closes.Load())
				}
			}
		})
	}
	t.Run("native retry source change cannot resend", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		var calls, retries, callbacks atomic.Int32
		const body = "original native503"
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestMethod(t, r, http.MethodDelete)
			w.Header().Set("X-Delete-Proof", "retry")
			testcloud.JSON(w, 503, body)
		})
		cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, _ error, _ uint) error {
			retries.Add(1)
			client.ResourceBase = cloud.Server.URL + "/changed/"
			return nil
		}
		err := keypairs.New(client).DeleteKeypair(context.Background(), "name", func(_ *request.Config[keypairs.KeypairDeleteOpts]) error { callbacks.Add(1); return nil })
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != body || native.ResponseHeader.Get("X-Delete-Proof") != "retry" || calls.Load() != 1 || retries.Load() != 1 || callbacks.Load() != 1 {
			t.Fatal("retry source change resent/hidden original proof", err, native, calls.Load(), retries.Load(), callbacks.Load())
		}
	})
}
