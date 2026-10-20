package nativefind_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/nativefind"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
)

func TestNativeFlavorExtraSpecsUsesResolvedIDAndOwnsReturnedModel(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("compute", "/unused")
	client.ResourceBase = cloud.Server.URL + "/reverse/compute/"
	input := &flavors.Flavor{ID: "1", Name: "canonical-name", Disk: 12, RAM: 2048, VCPUs: 3, Swap: 7, RxTxFactor: 0.5, IsPublic: true, Ephemeral: 5, Description: "detail", ExtraSpecs: map[string]string{}}
	original := *input
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/compute/flavors/1/os-extra_specs", func(w http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		if request.URL.RawQuery != "" {
			t.Error("lookup query leaked to extra specs", request.URL)
		}
		testcloud.JSON(w, 200, `{"extra_specs":{"hw:cpu_policy":"dedicated","vendor":"value"}}`)
	})
	value, err := nativefind.FlavorExtraSpecs(context.Background(), client, input)
	if err != nil || value == nil || value == input || calls.Load() != 1 {
		t.Fatal(value, err, calls.Load())
	}
	copy := *value
	copy.ExtraSpecs = original.ExtraSpecs
	if !reflect.DeepEqual(copy, original) || !reflect.DeepEqual(*input, original) || !reflect.DeepEqual(value.ExtraSpecs, map[string]string{"hw:cpu_policy": "dedicated", "vendor": "value"}) {
		t.Fatal("enrichment replaced other fields or mutated input", value, input, original)
	}
	value.ExtraSpecs["vendor"] = "caller-change"
	if len(input.ExtraSpecs) != 0 {
		t.Fatal("returned map aliases caller input", input)
	}
}

func TestNativeFlavorExtraSpecsInlineAndNativeNilEmptyDefaults(t *testing.T) {
	for _, entry := range []struct {
		name    string
		inline  map[string]string
		body    string
		nilMap  bool
		calls   int32
		entries map[string]string
	}{
		{"inline", map[string]string{"existing": "value"}, "", false, 0, map[string]string{"existing": "value"}},
		{"omitted", nil, `{}`, true, 1, nil},
		{"null", map[string]string{}, `{"extra_specs":null}`, true, 1, nil},
		{"empty", nil, `{"extra_specs":{}}`, false, 1, map[string]string{}},
		{"filled", map[string]string{}, `{"extra_specs":{"new":"value"}}`, false, 1, map[string]string{"new": "value"}},
	} {
		t.Run(entry.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("compute", "/reverse")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/flavors/canonical/os-extra_specs", func(w http.ResponseWriter, request *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, entry.body)
			})
			input := &flavors.Flavor{ID: "canonical", ExtraSpecs: entry.inline}
			value, err := nativefind.FlavorExtraSpecs(context.Background(), client, input)
			if err != nil || value == nil || value == input || calls.Load() != entry.calls || (value.ExtraSpecs == nil) != entry.nilMap || !reflect.DeepEqual(value.ExtraSpecs, entry.entries) {
				t.Fatal(value, err, calls.Load())
			}
			if entry.calls == 0 {
				value.ExtraSpecs["existing"] = "changed"
				if input.ExtraSpecs["existing"] != "value" {
					t.Fatal("inline map was not copied", input)
				}
			}
		})
	}
}

func TestNativeFlavorExtraSpecsNativeFailuresRemainTerminal(t *testing.T) {
	for _, status := range []int{203, 204, 400, 403, 404, 409, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("compute", "/reverse")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/flavors/canonical/os-extra_specs", func(w http.ResponseWriter, request *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Evidence", "original")
				testcloud.JSON(w, status, `{"error":"extra-specs-failed"}`)
			})
			input := &flavors.Flavor{ID: "canonical", ExtraSpecs: map[string]string{}}
			value, err := nativefind.FlavorExtraSpecs(context.Background(), client, input)
			var native gophercloud.ErrUnexpectedResponseCode
			if value != nil || !errors.As(err, &native) || native.Actual != status || native.URL != cloud.Server.URL+"/reverse/flavors/canonical/os-extra_specs" || native.ResponseHeader.Get("X-Evidence") != "original" || calls.Load() != 1 || len(input.ExtraSpecs) != 0 {
				t.Fatal(value, err, native, calls.Load())
			}
		})
	}
	for _, body := range []string{`{"extra_specs":{"numeric":1}}`, `{"extra_specs":{"boolean":true}}`, `{"extra_specs":[]}`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/flavors/canonical/os-extra_specs", func(w http.ResponseWriter, request *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, body)
			})
			value, err := nativefind.FlavorExtraSpecs(context.Background(), cloud.Client("compute", "/reverse"), &flavors.Flavor{ID: "canonical"})
			var typed *json.UnmarshalTypeError
			if value != nil || !errors.As(err, &typed) || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
		})
	}
	for _, read := range []bool{false, true} {
		t.Run(fmt.Sprintf("read=%t", read), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			sentinel := errors.New("native-extra-specs-cause")
			cloud.Provider.HTTPClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
				calls.Add(1)
				if !read {
					return nil, sentinel
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Request: request, Body: nativeListReadError{sentinel}}, nil
			})
			value, err := nativefind.FlavorExtraSpecs(context.Background(), cloud.Client("compute", "/reverse"), &flavors.Flavor{ID: "canonical"})
			if value != nil || !errors.Is(err, sentinel) || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
		})
	}
}

func TestNativeFlavorExtraSpecsSharesLiveSourceAndFreezesInput(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("compute", "/unused")
	client.ResourceBase = cloud.Server.URL + "/reverse/compute/"
	client.Microversion = "2.7"
	client.MoreHeaders = map[string]string{"X-Source": "unchanged"}
	originalEndpoint, originalBase := client.Endpoint, client.ResourceBase
	provider := client.ProviderClient
	input := &flavors.Flavor{ID: "canonical", Name: "original", ExtraSpecs: map[string]string{}}
	ctx := context.WithValue(context.Background(), struct{ value string }{"native"}, "request-context")
	var calls, middleware atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/compute/flavors/canonical/os-extra_specs", func(w http.ResponseWriter, request *http.Request) {
		call := calls.Add(1)
		expectedToken := "test-token"
		if call == 2 {
			expectedToken = "renewed"
		}
		if request.Header.Get("X-Auth-Token") != expectedToken || request.Header.Get("X-Source") != "unchanged" || request.Header.Get("X-OpenStack-Nova-API-Version") != "2.7" || request.URL.RawQuery != "" {
			t.Error(request.Header, request.URL)
		}
		testcloud.JSON(w, 200, `{"extra_specs":{"from":"native"}}`)
	})
	transport := provider.HTTPClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	provider.HTTPClient.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		middleware.Add(1)
		if request.Context().Value(struct{ value string }{"native"}) != "request-context" {
			t.Error("caller context was replaced")
		}
		input.ID = "caller-change"
		input.Name = "changed"
		input.ExtraSpecs["late"] = "input"
		return transport.RoundTrip(request)
	})
	value, err := nativefind.FlavorExtraSpecs(ctx, client, input)
	if err != nil || value == nil || value.ID != "canonical" || value.Name != "original" || !reflect.DeepEqual(value.ExtraSpecs, map[string]string{"from": "native"}) {
		t.Fatal(value, err)
	}
	provider.SetToken("renewed")
	if _, err := nativefind.FlavorExtraSpecs(ctx, client, &flavors.Flavor{ID: "canonical"}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 || middleware.Load() != 2 || client.ProviderClient != provider || client.Endpoint != originalEndpoint || client.ResourceBase != originalBase || client.Microversion != "2.7" || client.MoreHeaders["X-Source"] != "unchanged" {
		t.Fatal("source changed or request was replayed", calls.Load(), middleware.Load(), client)
	}
}

func TestNativeFlavorExtraSpecsRejectsInvalidInputAndCancellationBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("compute", "/reverse")
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /", func(w http.ResponseWriter, request *http.Request) { calls.Add(1) })
	for _, id := range []string{"", ".", "..", "a/b", "a%2fb", "a?query", "a#fragment", "a\\b", "a b", "a\u00a0b", "a\x00", string([]byte{0xff})} {
		for _, inline := range []bool{false, true} {
			value := &flavors.Flavor{ID: id}
			if inline {
				value.ExtraSpecs = map[string]string{"inline": "value"}
			}
			if result, err := nativefind.FlavorExtraSpecs(context.Background(), client, value); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(id, inline, result, err)
			}
		}
	}
	for _, source := range []*gophercloud.ServiceClient{nil, {Endpoint: client.Endpoint}} {
		if _, err := nativefind.FlavorExtraSpecs(context.Background(), source, &flavors.Flavor{ID: "canonical"}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(source, err)
		}
	}
	for _, endpoint := range []string{"/relative/", "ftp://host/reverse/", "http://user:password@host/reverse/", cloud.Server.URL + "/reverse/?query=existing", cloud.Server.URL + "/reverse/#fragment"} {
		copy := *client
		copy.Endpoint = endpoint
		if _, err := nativefind.FlavorExtraSpecs(context.Background(), &copy, &flavors.Flavor{ID: "canonical"}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(endpoint, err)
		}
	}
	if _, err := nativefind.FlavorExtraSpecs(nil, client, &flavors.Flavor{ID: "canonical"}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := nativefind.FlavorExtraSpecs(context.Background(), client, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := nativefind.FlavorExtraSpecs(ctx, client, &flavors.Flavor{ID: "canonical", ExtraSpecs: map[string]string{"inline": "value"}}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("preflight reached HTTP", calls.Load())
	}
}

func TestNativeFlavorExtraSpecsConcurrentCallsReturnIndependentMaps(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("compute", "/reverse")
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/flavors/canonical/os-extra_specs", func(w http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"extra_specs":{"shared":"native"}}`)
	})
	input := &flavors.Flavor{ID: "canonical", Name: "name"}
	values := make([]*flavors.Flavor, 8)
	var group sync.WaitGroup
	for i := range values {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			value, err := nativefind.FlavorExtraSpecs(context.Background(), client, input)
			if err != nil || value == nil || value.Name != "name" {
				t.Error(value, err)
				return
			}
			values[i] = value
		}(i)
	}
	group.Wait()
	if calls.Load() != 8 || input.ExtraSpecs != nil {
		t.Fatal(calls.Load(), input)
	}
	for i, value := range values {
		if value == nil {
			t.Fatal("missing concurrent result", i)
		}
		value.ExtraSpecs["shared"] = fmt.Sprint(i)
	}
	for i, value := range values {
		if value.ExtraSpecs["shared"] != fmt.Sprint(i) {
			t.Fatal("sibling results share maps", i, value)
		}
	}
}
