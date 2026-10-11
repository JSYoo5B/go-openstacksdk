package cindermetadata

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestRecordExchangeReusesFixedScopeAndUnparsedAcknowledgements(t *testing.T) {
	for _, collection := range []string{"volumes", "snapshots"} {
		t.Run(collection, func(t *testing.T) {
			fake := testcloud.New(t)
			client := fake.Client("volume", "/cinder/project/")
			client.Microversion = "3.15"
			source, err := cloudread.Capture(context.Background(), client, "volume")
			if err != nil {
				t.Fatal(err)
			}
			scope, err := New(context.Background(), &source.Client, collection, "literal-id")
			if err != nil {
				t.Fatal(err)
			}
			var methods []string
			fake.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("OpenStack-API-Version") != "volume 3.15" || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Fatal(r.Header)
				}
				expected := "/cinder/project/" + collection + "/literal-id/metadata"
				if r.Method == http.MethodDelete {
					expected += "/a%2Fb"
				}
				if r.URL.EscapedPath() != expected || r.URL.RawQuery != "" {
					t.Fatal(r.URL)
				}
				methods = append(methods, r.Method)
				testcloud.JSON(w, 202, "opaque ACK")
			})
			key := "a/b"
			for _, step := range []struct {
				method string
				key    *string
				body   any
			}{{"GET", nil, nil}, {"POST", nil, map[string]any{"metadata": nil}}, {"PUT", nil, map[string]any{"metadata": map[string]any{}}}, {"DELETE", &key, nil}} {
				response, err := scope.ExchangeRecord(context.Background(), step.method, step.key, step.body, source.Guard)
				if err != nil || response == nil || response.StatusCode != 202 || string(response.Body) != "opaque ACK" {
					t.Fatal(response, err)
				}
			}
			if !reflect.DeepEqual(methods, []string{"GET", "POST", "PUT", "DELETE"}) {
				t.Fatal(methods)
			}
		})
	}
}

func TestRecordExchangeRejectsUnsupportedTargetsBeforeHTTP(t *testing.T) {
	fake := testcloud.New(t)
	client := fake.Client("volume", "/cinder/project/")
	source, err := cloudread.Capture(context.Background(), client, "volume")
	if err != nil {
		t.Fatal(err)
	}
	scope, err := New(context.Background(), &source.Client, "volumes", "literal-id")
	if err != nil {
		t.Fatal(err)
	}
	fake.Mux.HandleFunc("/", func(http.ResponseWriter, *http.Request) { t.Error("unexpected HTTP") })
	key := "key"
	for _, step := range []struct {
		method string
		key    *string
		body   any
	}{{"PATCH", nil, nil}, {"GET", &key, nil}, {"GET", nil, map[string]any{}}, {"POST", &key, nil}, {"DELETE", nil, nil}, {"DELETE", &key, map[string]any{}}} {
		response, err := scope.ExchangeRecord(context.Background(), step.method, step.key, step.body, source.Guard)
		if response != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(response, err)
		}
	}
	if response, err := scope.ExchangeRecord(context.Background(), "GET", nil, nil, nil); response != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(response, err)
	}
}

func TestRecordExchangePreservesRejectedReadFaultAndPreventsReplay(t *testing.T) {
	fake := testcloud.New(t)
	client := fake.Client("volume", "/cinder/project/")
	source, err := cloudread.Capture(context.Background(), client, "volume")
	if err != nil {
		t.Fatal(err)
	}
	scope, err := New(context.Background(), &source.Client, "snapshots", "literal-id")
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("physical rejected read")
	calls := 0
	fake.Provider.HTTPClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 503, Header: http.Header{"X-Request-Id": {"actual503"}}, Body: &failingBody{data: "partial", cause: cause}, Request: r}, nil
	})
	// Refresh the captured transport after configuring the fixture.
	source, err = cloudread.Capture(context.Background(), client, "volume")
	if err != nil {
		t.Fatal(err)
	}
	scope, err = New(context.Background(), &source.Client, "snapshots", "literal-id")
	if err != nil {
		t.Fatal(err)
	}
	fake.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return nil }
	response, err := scope.ExchangeRecord(context.Background(), "POST", nil, map[string]any{"metadata": map[string]any{}}, source.Guard)
	var native gophercloud.ErrUnexpectedResponseCode
	if calls != 1 || !errors.Is(err, cause) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != "partial" || native.ResponseHeader.Get("X-Request-Id") != "actual503" || response == nil || response.StatusCode != 503 {
		t.Fatal(calls, response, err)
	}
}

func TestRecordExchangeSelectedVersionKeepsCleanNativeRetry(t *testing.T) {
	fake := testcloud.New(t)
	client := fake.Client("volume", "/cinder/project/")
	client.Microversion = "3.15"
	client.MoreHeaders = map[string]string{"openstack-api-version": "volume 3.15", "x-openstack-volume-api-version": "3.15"}
	source, err := cloudread.Capture(context.Background(), client, "volume")
	if err != nil {
		t.Fatal(err)
	}
	scope, err := New(context.Background(), &source.Client, "volumes", "literal-id")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	fake.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("OpenStack-API-Version") != "volume 3.15" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.15" {
			t.Error(r.Header)
		}
		if calls == 1 {
			testcloud.JSON(w, 503, `{"error":"temporary"}`)
		} else {
			testcloud.JSON(w, 200, `{"metadata":{}}`)
		}
	})
	fake.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return nil }
	response, err := scope.ExchangeRecord(context.Background(), "GET", nil, nil, source.Guard)
	if err != nil || calls != 2 || response == nil || response.StatusCode != 200 {
		t.Fatal(calls, response, err)
	}
}
