package cloudread

import (
	"context"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func cloudReadPolicyString(value string) *string { return &value }

func cloudReadPolicyClient(endpoint string) *gophercloud.ServiceClient {
	provider := &gophercloud.ProviderClient{HTTPClient: http.Client{}}
	provider.UseTokenLock()
	provider.SetToken("captured-token")
	return &gophercloud.ServiceClient{ProviderClient: provider, Endpoint: endpoint, Microversion: "3.60", MoreHeaders: map[string]string{"accept": "application/original", "X-Captured": "original"}}
}

func cloudReadPolicySource(t *testing.T, client *gophercloud.ServiceClient) *Source {
	t.Helper()
	value, err := Capture(context.Background(), client, "volume")
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func cloudReadPolicySnapshot(value gophercloud.ServiceClient) gophercloud.ServiceClient {
	value.MoreHeaders = maps.Clone(value.MoreHeaders)
	return value
}

func cloudReadPolicyNoResponse(t *testing.T, err error) {
	t.Helper()
	var proof *resource.ResponseError
	if errors.As(err, &proof) {
		t.Fatal("local policy failure borrowed a physical response", proof)
	}
}

func TestCloudReadPolicyCanonicalOverridesOwnInputsAndKeepSelectedSource(t *testing.T) {
	client := cloudReadPolicyClient("https://policy.test/reverse/v3/")
	before := cloudReadPolicySnapshot(*client)
	value := cloudReadPolicySource(t, client)
	headers := map[string]string{"ACCEPT": "application/operation", "Content-Type": "application/custom", "x-operation": "owned", "X-Operation": "owned"}
	version := "3.7"
	if err := value.WithPolicy(context.Background(), &version, headers); err != nil {
		t.Fatal(err)
	}
	headers["ACCEPT"], headers["x-operation"], version = "changed", "changed", "latest"
	delete(headers, "Content-Type")
	if value.Client.Microversion != "3.7" || value.Client.Type != "volumev3" || value.Client.ProviderClient != client.ProviderClient || value.Client.Endpoint != client.Endpoint || value.Client.ResourceBase != client.ResourceBase {
		t.Fatal(value.Client)
	}
	want := map[string]string{"Accept": "application/operation", "Content-Type": "application/custom", "X-Operation": "owned", "X-Captured": "original"}
	if !reflect.DeepEqual(value.Client.MoreHeaders, want) || !reflect.DeepEqual(*client, before) {
		t.Fatal(value.Client.MoreHeaders, *client, before)
	}
	client.MoreHeaders["X-Captured"] = "later ordinary source setting"
	if err := value.Guard(context.Background()); err != nil {
		t.Fatal(err)
	}
	if value.Client.MoreHeaders["X-Captured"] != "original" {
		t.Fatal("captured source header aliased original", value.Client.MoreHeaders)
	}
}

func TestCloudReadPolicyExplicitMicroversionsKeepEmptyAndSymbolicValues(t *testing.T) {
	for _, tc := range []struct {
		name    string
		version *string
		want    string
	}{
		{"omitted", nil, "3.60"}, {"explicit-empty", cloudReadPolicyString(""), ""},
		{"symbolic", cloudReadPolicyString("latest"), "latest"}, {"numeric", cloudReadPolicyString("3.7"), "3.7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := cloudReadPolicyClient("https://policy.test/reverse/v3/")
			value := cloudReadPolicySource(t, client)
			if err := value.WithPolicy(context.Background(), tc.version, nil); err != nil {
				t.Fatal(err)
			}
			if tc.version != nil {
				*tc.version = "caller changed retained pointer"
			}
			if value.Client.Microversion != tc.want || client.Microversion != "3.60" || client.Type != "" {
				t.Fatal(value.Client.Microversion, client)
			}
			if err := value.WithPolicy(context.Background(), nil, map[string]string{"X-Later": "later"}); err != nil {
				t.Fatal(err)
			}
			if value.Client.Microversion != tc.want || value.Client.MoreHeaders["X-Later"] != "later" {
				t.Fatal("nil override reset current owned policy", value.Client)
			}
		})
	}
}

func TestCloudReadPolicyRejectsInvalidHeadersAndVersionsAtomically(t *testing.T) {
	for _, tc := range []struct {
		name    string
		version *string
		headers map[string]string
	}{
		{"auth-token", nil, map[string]string{"X-Auth-Token": "caller"}},
		{"service-token", nil, map[string]string{"X-Service-Token": "caller"}},
		{"authorization", nil, map[string]string{"Authorization": "caller"}},
		{"host-route", nil, map[string]string{"Host": "changed.test"}},
		{"body-framing", nil, map[string]string{"Content-Length": "1"}},
		{"protocol-upgrade", nil, map[string]string{"Upgrade": "websocket"}},
		{"header-injection", nil, map[string]string{"X-Operation": "value\r\nInjected: yes"}},
		{"header-token", nil, map[string]string{"bad name": "value"}},
		{"conflicting-case-aliases", nil, map[string]string{"X-Operation": "first", "x-operation": "second"}},
		{"generic-version-mismatch", nil, map[string]string{"OpenStack-API-Version": "volume 3.60"}},
		{"legacy-version-mismatch", nil, map[string]string{"X-OpenStack-Volume-API-Version": "3.60"}},
		{"cross-role-version", nil, map[string]string{"X-OpenStack-Nova-API-Version": "3.7"}},
		{"version-control", cloudReadPolicyString("3.7\n"), map[string]string{"X-Operation": "would otherwise change"}},
		{"version-invalid-UTF8", cloudReadPolicyString(string([]byte{0xff})), nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := cloudReadPolicyClient("https://policy.test/reverse/v3/")
			value := cloudReadPolicySource(t, client)
			if err := value.WithPolicy(context.Background(), cloudReadPolicyString("3.7"), map[string]string{"X-Operation": "prior valid"}); err != nil {
				t.Fatal(err)
			}
			before, original := cloudReadPolicySnapshot(value.Client), cloudReadPolicySnapshot(*client)
			err := value.WithPolicy(context.Background(), tc.version, tc.headers)
			if !errors.Is(err, resource.ErrInvalidOption) || !reflect.DeepEqual(value.Client, before) || !reflect.DeepEqual(*client, original) {
				t.Fatal(err, value.Client, before, *client)
			}
			cloudReadPolicyNoResponse(t, err)
			if err := value.WithPolicy(context.Background(), nil, map[string]string{"X-Recovered": "valid"}); err != nil {
				t.Fatal("a local policy error poisoned the unchanged source", err)
			}
		})
	}
}

func TestCloudReadPolicyVersionOverrideRequiresConsistentCapturedHeaders(t *testing.T) {
	client := cloudReadPolicyClient("https://policy.test/reverse/v3/")
	client.MoreHeaders["openstack-api-version"] = "volume 3.60"
	client.MoreHeaders["x-openstack-volume-api-version"] = "3.60"
	value := cloudReadPolicySource(t, client)
	before := cloudReadPolicySnapshot(value.Client)
	for _, version := range []string{"3.7", ""} {
		err := value.WithPolicy(context.Background(), &version, nil)
		if !errors.Is(err, resource.ErrInvalidOption) || !reflect.DeepEqual(value.Client, before) {
			t.Fatal("stale captured version headers were silently discarded", err, value.Client)
		}
	}
	headers := map[string]string{"OPENSTACK-API-VERSION": "volume latest", "X-OpenStack-Volume-API-Version": "latest"}
	if err := value.WithPolicy(context.Background(), cloudReadPolicyString("latest"), headers); err != nil {
		t.Fatal(err)
	}
	if value.Client.Microversion != "latest" || value.Client.MoreHeaders["Openstack-Api-Version"] != "volume latest" || value.Client.MoreHeaders["X-Openstack-Volume-Api-Version"] != "latest" {
		t.Fatal(value.Client)
	}
	if client.Microversion != "3.60" || client.MoreHeaders["openstack-api-version"] != "volume 3.60" {
		t.Fatal("source settings changed", client)
	}
}

func TestCloudReadPolicySourceChangeStaysStickyBeforeAnyPolicyCommit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*gophercloud.ServiceClient)
	}{
		{"endpoint", func(c *gophercloud.ServiceClient) { c.Endpoint = "https://changed.test/v3/" }},
		{"resource-base", func(c *gophercloud.ServiceClient) { c.ResourceBase = c.Endpoint + "changed/" }},
		{"role", func(c *gophercloud.ServiceClient) { c.Type = "compute" }},
		{"version", func(c *gophercloud.ServiceClient) { c.Microversion = "3.61" }},
		{"provider", func(c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := cloudReadPolicyClient("https://policy.test/reverse/v3/")
			value := cloudReadPolicySource(t, client)
			original, before := cloudReadPolicySnapshot(*client), cloudReadPolicySnapshot(value.Client)
			tc.mutate(client)
			err := value.WithPolicy(context.Background(), cloudReadPolicyString("latest"), map[string]string{"X-Operation": "must not commit"})
			if !errors.Is(err, resource.ErrInvalidOption) || !reflect.DeepEqual(value.Client, before) {
				t.Fatal(err, value.Client)
			}
			*client = original
			err = value.WithPolicy(context.Background(), nil, nil)
			if !errors.Is(err, resource.ErrInvalidOption) || !reflect.DeepEqual(value.Client, before) {
				t.Fatal("observed source change was forgotten after restoration", err)
			}
			cloudReadPolicyNoResponse(t, err)
		})
	}
}

func TestCloudReadPolicyCancellationRetainsCauseAndIsAtomic(t *testing.T) {
	client := cloudReadPolicyClient("https://policy.test/reverse/v3/")
	value := cloudReadPolicySource(t, client)
	before := cloudReadPolicySnapshot(value.Client)
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("caller cancelled policy preparation")
	cancel(cause)
	err := value.WithPolicy(ctx, cloudReadPolicyString("latest"), map[string]string{"X-Operation": "must not commit"})
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !reflect.DeepEqual(value.Client, before) {
		t.Fatal(err, value.Client)
	}
	cloudReadPolicyNoResponse(t, err)
	if err := value.WithPolicy(context.Background(), nil, nil); err != nil {
		t.Fatal("context cancellation was incorrectly stored as source drift", err)
	}
	if err := value.WithPolicy(nil, nil, nil); !errors.Is(err, resource.ErrInvalidOption) || !reflect.DeepEqual(value.Client, before) {
		t.Fatal("nil context policy committed", err)
	}
	var absent *Source
	if err := absent.WithPolicy(context.Background(), nil, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestCloudReadPolicyOwnedWirePolicyKeepsLiveTokenAcrossNativeRetry(t *testing.T) {
	type observation struct {
		method, path, query, token, accept, captured, operation, version, legacy string
		body                                                                     []byte
	}
	var mu sync.Mutex
	var observations []observation
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		mu.Lock()
		observations = append(observations, observation{r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("X-Auth-Token"), r.Header.Get("Accept"), r.Header.Get("X-Captured"), r.Header.Get("X-Operation"), r.Header.Get("OpenStack-API-Version"), r.Header.Get("X-OpenStack-Volume-API-Version"), body})
		mu.Unlock()
		if requests.Add(1) == 1 {
			w.WriteHeader(503)
			_, _ = io.WriteString(w, "first rejected")
			return
		}
		w.Header().Set("X-Request-Id", "actual-accepted")
		_, _ = io.WriteString(w, `{"snapshots":[]}`)
	}))
	defer server.Close()
	client := cloudReadPolicyClient(server.URL + "/reverse/v3/")
	var callbacks atomic.Int32
	target := client.ServiceURL("snapshots", "detail") + "?name=literal"
	client.RetryFunc = func(_ context.Context, method, endpoint string, opts *gophercloud.RequestOpts, original error, count uint) error {
		callbacks.Add(1)
		if count != 1 {
			return original
		}
		if method != http.MethodGet || endpoint != target || !gophercloud.ResponseCodeIs(original, 503) || !opts.KeepResponseBody || opts.JSONResponse != nil || opts.JSONBody != nil || opts.RawBody != nil {
			t.Error("native retry no longer owns the same bodyless request", method, endpoint, opts, original)
		}
		client.SetToken("retry-live-token")
		client.MoreHeaders["X-Captured"] = "changed ordinary source header"
		return nil
	}
	originalHook := reflect.ValueOf(client.RetryFunc).Pointer()
	value := cloudReadPolicySource(t, client)
	headers := map[string]string{"Accept": "application/operation", "X-Operation": "owned"}
	version := "latest"
	if err := value.WithPolicy(context.Background(), &version, headers); err != nil {
		t.Fatal(err)
	}
	headers["X-Operation"], version = "retained input changed", "3.9"
	client.SetToken("before-send-live-token")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	response, err := value.Get(ctx, target, 200)
	if err != nil || response == nil || response.StatusCode != 200 || string(response.Body) != `{"snapshots":[]}` || response.Header.Get("X-Request-Id") != "actual-accepted" || requests.Load() != 2 || callbacks.Load() != 1 {
		t.Fatal(response, err, requests.Load(), callbacks.Load())
	}
	mu.Lock()
	got := append([]observation(nil), observations...)
	mu.Unlock()
	if len(got) != 2 {
		t.Fatal(got)
	}
	for i, entry := range got {
		wantToken := "before-send-live-token"
		if i == 1 {
			wantToken = "retry-live-token"
		}
		if entry.method != "GET" || entry.path != "/reverse/v3/snapshots/detail" || entry.query != "name=literal" || len(entry.body) != 0 || entry.token != wantToken || entry.accept != "application/operation" || entry.captured != "original" || entry.operation != "owned" || entry.version != "volume latest" || entry.legacy != "latest" {
			t.Fatal(i, entry)
		}
	}
	if client.Microversion != "3.60" || client.Type != "" || reflect.ValueOf(client.RetryFunc).Pointer() != originalHook || value.Client.Microversion != "latest" {
		t.Fatal("wire operation altered selected source or caller hook", client, value.Client)
	}
}

func TestCloudReadPolicySourceMutationInNativeRetryStopsBeforeResend(t *testing.T) {
	var requests, callbacks atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("X-Request-Id", "actual-rejected")
		w.WriteHeader(503)
		_, _ = io.WriteString(w, "one physical rejected response")
	}))
	defer server.Close()
	client := cloudReadPolicyClient(server.URL + "/v3/")
	client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, count uint) error {
		callbacks.Add(1)
		if count != 1 {
			return original
		}
		client.Microversion = "changed-source-version"
		return nil
	}
	value := cloudReadPolicySource(t, client)
	if err := value.WithPolicy(context.Background(), cloudReadPolicyString("latest"), map[string]string{"X-Operation": "owned"}); err != nil {
		t.Fatal(err)
	}
	before := cloudReadPolicySnapshot(value.Client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	response, err := value.Get(ctx, client.Endpoint+"snapshots", 200)
	var native gophercloud.ErrUnexpectedResponseCode
	if response != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != "one physical rejected response" || native.ResponseHeader.Get("X-Request-Id") != "actual-rejected" || requests.Load() != 1 || callbacks.Load() != 1 {
		t.Fatal(response, err, native, requests.Load(), callbacks.Load())
	}
	cloudReadPolicyNoResponse(t, err)
	client.Microversion = "3.60"
	err = value.WithPolicy(context.Background(), nil, map[string]string{"X-New": "must not commit"})
	if !errors.Is(err, resource.ErrInvalidOption) || !reflect.DeepEqual(value.Client, before) || requests.Load() != 1 {
		t.Fatal("source restoration revived a known violation", err, value.Client)
	}
}
