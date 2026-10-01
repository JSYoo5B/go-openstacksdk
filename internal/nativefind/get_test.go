package nativefind_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"gophercloudsdk/internal/nativefind"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func TestNativeIdentityGetPreservesQueryNativeResultCodesAndEvidence(t *testing.T) {
	for _, status := range []int{200, 203, 201, 400, 403, 404, 409, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("compute", "/unused")
			client.ResourceBase = cloud.Server.URL + "/reverse/compute/"
			var calls atomic.Int32
			query := url.Values{"fields": {"id", "name"}, "name": {"a&b /한글"}, "empty": nil}
			cloud.Mux.HandleFunc("GET /reverse/compute/servers/requested", func(w http.ResponseWriter, request *http.Request) {
				calls.Add(1)
				if request.URL.RawQuery != query.Encode() || request.Header.Get("X-Auth-Token") != "test-token" {
					t.Error(request.URL, request.Header)
				}
				w.Header().Set("X-Evidence", "native")
				testcloud.JSON(w, status, `{"server":{"id":"canonical","name":"different"}}`)
			})
			var result servers.GetResult
			result.Header, result.Err = nativefind.Get(context.Background(), client, []string{"servers", "requested"}, query, []int{200, 203}, &result.Body)
			value, err := result.Extract()
			if status == 200 || status == 203 {
				if err != nil || value == nil || value.ID != "canonical" || result.Header.Get("X-Evidence") != "native" {
					t.Fatal(value, err, result.Header)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				// The native extractor returns a zero model pointer alongside its
				// error. Preserve that extractor's contract and inspect the cause.
				if !errors.As(err, &native) || native.Actual != status || native.URL != cloud.Server.URL+"/reverse/compute/servers/requested?"+query.Encode() || native.ResponseHeader.Get("X-Evidence") != "native" || string(native.Body) != `{"server":{"id":"canonical","name":"different"}}` {
					t.Fatal(value, err, native)
				}
			}
			if calls.Load() != 1 {
				t.Fatal("native getter resent the request", calls.Load())
			}
		})
	}
}

func TestNativeIdentityGetFreezesInputsAndUsesLiveProviderReauthentication(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("compute", "/reverse/compute")
	client.Microversion = "2.77"
	client.MoreHeaders = map[string]string{"X-Trace": "source"}
	query := url.Values{"fields": {"id", "name"}, "name": nil}
	segments := []string{"servers", "requested"}
	codes := []int{200, 203}
	var calls, reauth atomic.Int32
	cloud.Provider.ReauthFunc = func(ctx context.Context) error {
		reauth.Add(1)
		cloud.Provider.SetToken("renewed")
		return nil
	}
	cloud.Mux.HandleFunc("GET /reverse/compute/servers/requested", func(w http.ResponseWriter, request *http.Request) {
		call := calls.Add(1)
		if !reflect.DeepEqual(request.URL.Query()["fields"], []string{"id", "name"}) || request.URL.Query().Has("name") || request.Header.Get("X-Trace") != "source" || request.Header.Get("X-OpenStack-Nova-API-Version") != "2.77" {
			t.Error(request.URL, request.Header)
		}
		if call == 1 {
			if request.Header.Get("X-Auth-Token") != "test-token" {
				t.Error(request.Header)
			}
			testcloud.JSON(w, 401, `{"error":"expired"}`)
			return
		}
		if request.Header.Get("X-Auth-Token") != "renewed" {
			t.Error(request.Header)
		}
		testcloud.JSON(w, 203, `{"server":{"id":"canonical","name":"name"}}`)
	})
	transport := cloud.Provider.HTTPClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	cloud.Provider.HTTPClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		query["fields"][0] = "changed"
		query["name"] = []string{"changed"}
		segments[1] = "changed"
		codes[1] = 201
		return transport.RoundTrip(request)
	})
	var result servers.GetResult
	result.Header, result.Err = nativefind.Get(context.Background(), client, segments, query, codes, &result.Body)
	value, err := result.Extract()
	if err != nil || value == nil || value.ID != "canonical" || calls.Load() != 2 || reauth.Load() != 1 || client.Microversion != "2.77" || client.MoreHeaders["X-Trace"] != "source" {
		t.Fatal(value, err, calls.Load(), reauth.Load(), client)
	}
}

func TestNativeIdentityGetAcceptedDecodeAndContextRemainTerminal(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("compute", "/reverse")
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/servers/requested", func(w http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"server":{"id":true}}`)
	})
	var result servers.GetResult
	result.Header, result.Err = nativefind.Get(context.Background(), client, []string{"servers", "requested"}, url.Values{"domain_id": {"domain"}}, []int{200}, &result.Body)
	_, err := result.Extract()
	var typed *json.UnmarshalTypeError
	if !errors.As(err, &typed) || calls.Load() != 1 {
		t.Fatal(err, calls.Load())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := nativefind.Get(ctx, client, []string{"servers", "requested"}, nil, []int{200}, &result.Body); !errors.Is(err, context.Canceled) || calls.Load() != 1 {
		t.Fatal(err, calls.Load())
	}
}

func TestNativeIdentityGetRejectsUnsafeRoutesAndClientURLsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("compute", "/reverse")
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /", func(w http.ResponseWriter, request *http.Request) { calls.Add(1) })
	for _, part := range []string{"", ".", "..", "with space", "with\u00a0space", "x/y", "x%2fy", "x?query", "x#fragment", "x\\y", "x\x00", string([]byte{0xff})} {
		if _, err := nativefind.Get(context.Background(), client, []string{"servers", part}, nil, []int{200}, nil); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(part, err)
		}
	}
	for _, endpoint := range []string{"/relative/", cloud.Server.URL + "/reverse/?query=existing", cloud.Server.URL + "/reverse/#fragment", "http://user:password@host/reverse/", "ftp://host/reverse/"} {
		copy := *client
		copy.Endpoint = endpoint
		if _, err := nativefind.Get(context.Background(), &copy, []string{"servers", "id"}, nil, []int{200}, nil); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(endpoint, err)
		}
	}
	for _, source := range []*gophercloud.ServiceClient{nil, {Endpoint: client.Endpoint}} {
		if _, err := nativefind.Get(context.Background(), source, []string{"servers", "id"}, nil, []int{200}, nil); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(source, err)
		}
	}
	if _, err := nativefind.Get(nil, client, []string{"servers", "id"}, nil, []int{200}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, codes := range [][]int{nil, {99}, {600}} {
		if _, err := nativefind.Get(context.Background(), client, []string{"servers", "id"}, nil, codes, nil); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(codes, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("invalid input reached HTTP", calls.Load())
	}
}
