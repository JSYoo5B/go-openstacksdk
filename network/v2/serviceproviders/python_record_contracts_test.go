package serviceproviders_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func providerAPI(t *testing.T, handler func(http.ResponseWriter, *http.Request)) (*serviceproviders.API, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	count := &atomic.Int32{}
	cloud.Mux.HandleFunc("/neutron/", func(w http.ResponseWriter, req *http.Request) {
		count.Add(1)
		if req.Method != "GET" || req.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(req.Method, req.Header)
		}
		w.Header().Set("X-Request-Id", "provider-actual")
		handler(w, req)
	})
	client := cloud.Client("network", "/catalog/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return serviceproviders.New(client), count
}
func providerProof(t *testing.T, err error, code int, body string) {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "provider-actual" {
		t.Fatal(err, proof)
	}
}
func TestServiceProviderListOwnedDefaultsAliasesAndNoNetworkMixins(t *testing.T) {
	body := `{"service_providers":[{"name":"p","service_type":"QOS","default":"false","x_extension":{"big":9007199254740993},"tags":false,"revision_number":"untyped","self":"wire","location":{"fake":true}},{}]}`
	api, count := providerAPI(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/neutron/v2.0/service-providers" || req.URL.RawQuery != "" || req.Header.Get("Accept") != "application/json" {
			t.Error(req.URL, req.Header)
		}
		testcloud.JSON(w, 203, body)
	})
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 2 || count.Load() != 1 {
		t.Fatal(rows, err, count.Load())
	}
	for key, want := range map[string]string{"id": "null", "name": `"p"`, "service_type": `"QOS"`, "is_default": "true", "tags": "false", "revision_number": `"untyped"`, "location": "null"} {
		if string(rows[0].Resource.Body[key]) != want {
			t.Fatal(key, rows[0])
		}
	}
	if rows[0].Resource.Body["self"] != nil || rows[0].Resource.Body["default"] != nil || string(rows[0].Wire.Body["default"]) != `"false"` || string(rows[0].Wire.Body["self"]) != `"wire"` || rows[0].StatusCode != 203 || string(rows[0].Envelope) != body || len(rows[1].Resource.Body) != 5 {
		t.Fatal(rows)
	}
	for _, key := range []string{"id", "name", "service_type", "is_default", "location"} {
		if string(rows[1].Resource.Body[key]) != "null" {
			t.Fatal(rows[1])
		}
	}
	rows[0].Resource.Body["x_extension"][0] = 'X'
	rows[0].Header.Set("X-Request-Id", "consumer")
	rows[0].Envelope[0] = 'X'
	if string(rows[0].Wire.Body["x_extension"]) != `{"big":9007199254740993}` || rows[0].Wire.Header.Get("X-Request-Id") != "provider-actual" || rows[0].Resource.Header.Get("X-Request-Id") != "provider-actual" {
		t.Fatal(rows[0])
	}
}
func TestServiceProviderListFullQueriesDefaultPresenceAndLocalID(t *testing.T) {
	var query map[string][]string
	api, count := providerAPI(t, func(w http.ResponseWriter, req *http.Request) {
		query = req.URL.Query()
		testcloud.JSON(w, 200, `{"service_providers":[{"id":"keep","name":"other","default":false},{"id":"skip","name":"name","default":true}]}`)
	})
	rows, err := api.AllRecords(context.Background(), serviceproviders.WithRecordListLimit(0), serviceproviders.WithRecordListMarker("start"), serviceproviders.WithRecordListQuery("fields", []any{"name", nil, "default"}), serviceproviders.WithRecordListQuery("service_type", []string{"QOS", "FIREWALL"}), serviceproviders.WithRecordListQuery("name", "server-only"), serviceproviders.WithRecordListQuery("is_default", false), serviceproviders.WithRecordListQuery("default", true), serviceproviders.WithRecordListQuery("id", "keep"), serviceproviders.WithRecordListQuery("unknown", make(chan int)))
	want := map[string][]string{"limit": {"0"}, "marker": {"start"}, "fields": {"name", "default"}, "service_type": {"QOS", "FIREWALL"}, "name": {"server-only"}, "default": {"False"}}
	if err != nil || len(rows) != 1 || count.Load() != 1 || !reflect.DeepEqual(query, want) || string(rows[0].Resource.Body["id"]) != `"keep"` {
		t.Fatal(rows, err, query, count.Load())
	}
	rows, err = api.AllRecords(context.Background(), serviceproviders.WithRecordListQuery("is_default", nil), serviceproviders.WithRecordListQuery("default", true))
	if err != nil || len(rows) != 2 || query["default"] != nil {
		t.Fatal(rows, err, query)
	}
}
func TestServiceProviderProxyJMESPreservesOpaqueGeneratorBeforeHTTP(t *testing.T) {
	api, count := providerAPI(t, func(w http.ResponseWriter, req *http.Request) {
		testcloud.JSON(w, 200, `{"service_providers":[{"name":"p"}]}`)
	})
	for _, tc := range []struct{ expression, want string }{{"[*].name", "null"}, {"@ == @", "true"}, {"`{\"ok\":true}`", `{"ok":true}`}} {
		result, err := api.ProxyListRecords(context.Background(), serviceproviders.WithRecordListJMESPath(tc.expression))
		if err != nil || result == nil || string(result.JSON) != tc.want || count.Load() != 0 {
			t.Fatal(result, err, count.Load())
		}
	}
	if result, err := api.ProxyListRecords(context.Background(), serviceproviders.WithRecordListJMESPath("length(@)")); result != nil || err == nil || count.Load() != 0 {
		t.Fatal(result, err, count.Load())
	}
	result, err := api.ProxyListRecords(context.Background(), serviceproviders.WithRecordListJMESPath("to_array(@)"))
	if err != nil || result == nil || result.JSON != nil || count.Load() != 0 {
		t.Fatal(result, err, count.Load())
	}
	values, ok := result.Value.([]any)
	if !ok || len(values) != 1 {
		t.Fatal(result.Value)
	}
	generator, ok := values[0].(*serviceproviders.RecordListGenerator)
	if !ok {
		t.Fatal(values)
	}
	for value, err := range generator.Records {
		if err != nil || string(value.Resource.Body["name"]) != `"p"` {
			t.Fatal(value, err)
		}
	}
	if count.Load() != 1 {
		t.Fatal(count.Load())
	}
}
func TestServiceProviderListSourceControlsPagingAndActualErrors(t *testing.T) {
	api, count := providerAPI(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/neutron/v2.0/other-providers" || req.URL.Query().Get("name") != "restored" || req.Header.Get("X-Trace") != "owned" || req.Header.Get("OpenStack-API-Version") != "network 2.1" {
			t.Error(req.URL, req.Header)
		}
		if req.URL.Query().Get("marker") == "second" {
			testcloud.JSON(w, 200, `{"service_providers":[]}`)
			return
		}
		testcloud.JSON(w, 200, `{"service_providers":[{"name":"first","default":false}],"next":"/neutron/v2.0/other-providers?marker=second"}`)
	})
	rows, err := api.AllRecords(context.Background(), serviceproviders.WithRecordListBasePath("/other-providers"), serviceproviders.WithRecordListQuery("name", "old"), serviceproviders.WithRecordListConflictingAttrs(json.RawMessage(`{"name":"restored","headers":{"X-Trace":"owned"},"microversion":"2.1"}`)))
	if err != nil || len(rows) != 1 || count.Load() != 2 || api.RawClient().Microversion != "" {
		t.Fatal(rows, err, count.Load())
	}
	for _, body := range []string{`{}`, `{"service_providers":null}`, `{"service_providers":[{"cls":null}]}`, `{"service_providers":[null]}`} {
		invalid, calls := providerAPI(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, 203, body) })
		rows, err := invalid.AllRecords(context.Background())
		if rows != nil || err == nil || calls.Load() != 1 {
			t.Fatal(rows, err, calls.Load())
		}
		providerProof(t, err, 203, body)
	}
}
func TestServiceProviderListCapsAndOwnedCallbackMaps(t *testing.T) {
	api, count := providerAPI(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get("limit") != "2" || req.Header.Get("X-Trace") != "owned" {
			t.Error(req.URL, req.Header)
		}
		testcloud.JSON(w, 200, `{"service_providers":[{"id":"skip"},{"id":"keep"},{"cls":null}],"next":"https://foreign.invalid"}`)
	})
	var retained *serviceproviders.RecordListOpts
	rows, err := api.AllRecords(context.Background(), serviceproviders.WithRecordListHeaders(nil), func(c *serviceproviders.RecordListOpts) error {
		c.Headers["X-Trace"] = "owned"
		retained = c
		return nil
	}, func(*serviceproviders.RecordListOpts) error { retained.Headers["X-Trace"] = "changed"; return nil }, serviceproviders.WithRecordListQuery("id", "keep"), serviceproviders.WithRecordListMaxItems(2))
	if err != nil || len(rows) != 1 || count.Load() != 1 {
		t.Fatal(rows, err, count.Load())
	}
}

type providerObservedBody struct {
	io.ReadCloser
	readErr, closeErr error
	closed            *atomic.Int32
}

func (b *providerObservedBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if n > 0 && b.readErr != nil {
		return n, b.readErr
	}
	return n, err
}
func (b *providerObservedBody) Close() error {
	b.closed.Add(1)
	return errors.Join(b.ReadCloser.Close(), b.closeErr)
}

type providerObservedTransport struct {
	base              http.RoundTripper
	readErr, closeErr error
	closed            *atomic.Int32
}

func (t providerObservedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(req)
	if response != nil {
		response.Body = &providerObservedBody{ReadCloser: response.Body, readErr: t.readErr, closeErr: t.closeErr, closed: t.closed}
	}
	return response, err
}
func TestServiceProviderListPhysicalProofNeverReauthsOrRetries(t *testing.T) {
	for _, code := range []int{200, 401, 503} {
		body := `{"service_providers":[{"name":"p"}]}`
		api, count := providerAPI(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, code, body) })
		readErr, closeErr := errors.New("read"), errors.New("close")
		closed := &atomic.Int32{}
		retries, reauths := 0, 0
		base := api.RawClient().HTTPClient.Transport
		if base == nil {
			base = http.DefaultTransport
		}
		api.RawClient().HTTPClient.Transport = providerObservedTransport{base: base, readErr: readErr, closeErr: closeErr, closed: closed}
		api.RawClient().ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			retries++
			return nil
		}
		api.RawClient().ProviderClient.ReauthFunc = func(context.Context) error { reauths++; return nil }
		rows, err := api.AllRecords(context.Background())
		if rows != nil || !errors.Is(err, readErr) || !errors.Is(err, closeErr) || count.Load() != 1 || closed.Load() != 1 || retries != 0 || reauths != 0 {
			t.Fatal(rows, err, count.Load(), closed.Load(), retries, reauths)
		}
		if code < 400 {
			providerProof(t, err, code, body)
		} else {
			receipt := err
			if code == 401 {
				var auth *gophercloud.ErrUnableToReauthenticate
				if !errors.As(err, &auth) || !errors.Is(auth.ErrReauth, readErr) || !errors.Is(auth.ErrReauth, closeErr) {
					t.Fatal(err, auth)
				}
				receipt = auth.ErrOriginal
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(receipt, &native) || native.Actual != code || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "provider-actual" {
				t.Fatal(err, native)
			}
		}
	}
}
