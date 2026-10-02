package gophercloudsdk_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/keymanager/v1/secrets"
	"gophercloudsdk/resource"
)

func TestConnectionKeyManagerSecretFetchSharedClientAndLiveToken(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, lookups atomic.Int32
	cloud.Provider.EndpointLocator = func(options gophercloud.EndpointOpts) (string, error) {
		lookups.Add(1)
		if options.Type != "key-manager" || options.Region != "region" || options.Availability != gophercloud.AvailabilityInternal {
			t.Errorf("catalog selection: %#v", options)
		}
		return cloud.Server.URL + "/catalog/barbican/v1/", nil
	}
	metadata := `{"id":null,"secret_ref":"https://passive.test/secrets/other","name":"owned","content_types":{"default":"text/plain"},"bit_length":9007199254740993,"extension":{"present":null}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.URL.RawQuery != "" || r.Header.Get("X-SDK-Source") != "shared" || r.Header.Get("X-Trace") != "fetch" || r.Header.Get("OpenStack-API-Version") != "" {
			t.Error("shared fetch request policy", r.Method, r.URL, r.Header)
		}
		switch r.URL.Path {
		case "/reverse/barbican/v1/secrets/requested":
			if r.Header.Get("X-Auth-Token") != "metadata-token" || r.Header.Get("Accept") != "application/json" {
				t.Error("metadata request", r.Header)
			}
			cloud.Provider.SetToken("payload-token")
			w.Header().Set("X-Request-ID", "metadata-response")
			testcloud.JSON(w, http.StatusOK, metadata)
		case "/reverse/barbican/v1/secrets/requested/payload":
			if r.Header.Get("X-Auth-Token") != "payload-token" || r.Header.Get("Accept") != "text/plain" {
				t.Error("payload request lost live token or selected type", r.Header)
			}
			w.Header().Set("X-Request-ID", "payload-response")
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("비밀"))
		default:
			t.Error("fetch followed a response reference or performed an ID lookup", r.URL)
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithRegion("region"), sdk.WithInterface(gophercloud.AvailabilityInternal))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.KeyManagerV1(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	again, err := conn.KeyManager(context.Background())
	if err != nil || again != service || lookups.Load() != 1 {
		t.Fatal("cached service alias", again, err, lookups.Load())
	}
	client := service.RawClient()
	if service.Secrets.RawClient() != client || client.ProviderClient != cloud.Provider {
		t.Fatal("Fetch must use the existing authenticated Secrets API")
	}
	client.ResourceBase = cloud.Server.URL + "/reverse/barbican/v1/"
	client.MoreHeaders = map[string]string{"X-SDK-Source": "shared"}
	cloud.Provider.SetToken("metadata-token")
	value, err := service.Secrets.Fetch(context.Background(), resource.ID("requested"), secrets.WithFetchHeader("X-Trace", "fetch"))
	if err != nil || value == nil {
		t.Fatal(value, err)
	}
	if value.SecretID != "requested" || value.StatusCode != http.StatusOK || value.Header.Get("X-Request-ID") != "metadata-response" || value.Payload == nil || value.Payload.Text == nil || *value.Payload.Text != "비밀" || string(value.Payload.Body) != "비밀" || value.Payload.Accept != "text/plain" || value.Payload.StatusCode != http.StatusOK || value.Payload.Header.Get("X-Request-ID") != "payload-response" {
		t.Fatalf("combined result did not retain both actual responses: %#v", value)
	}
	body := value.Body
	if string(body["id"]) != "null" || string(body["bit_length"]) != "9007199254740993" || body["payload"] != nil || string(body["extension"]) != `{"present":null}` {
		t.Fatal("actual metadata body was replaced by request/payload fields", body)
	}
	if calls.Load() != 2 || lookups.Load() != 1 {
		t.Fatal("unexpected HTTP or catalog work", calls.Load(), lookups.Load())
	}
}

func TestConnectionKeyManagerSecretFetchCancellationAndStrictMissing(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.URL.Path {
		case "/barbican/v1/secrets/missing":
			testcloud.JSON(w, http.StatusNotFound, `{"error":"missing metadata"}`)
		case "/barbican/v1/secrets/payload-missing":
			w.Header().Set("X-Request-ID", "accepted-metadata")
			testcloud.JSON(w, http.StatusOK, `{"name":"visible","content_types":{"default":"text/plain"}}`)
		case "/barbican/v1/secrets/payload-missing/payload":
			testcloud.JSON(w, http.StatusNotFound, `{"error":"missing payload"}`)
		default:
			t.Error(r.URL)
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.KeyManager, cloud.Server.URL+"/barbican/v1/"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.KeyManager(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if value, err := service.Secrets.Fetch(ctx, resource.ID("missing")); value != nil || !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatal("canceled Fetch started HTTP", value, err, calls.Load())
	}
	if value, err := service.Secrets.Fetch(context.Background(), resource.ID("missing")); value != nil || !gophercloud.ResponseCodeIs(err, http.StatusNotFound) || calls.Load() != 1 {
		t.Fatal("metadata 404 was suppressed or caused another request", value, err, calls.Load())
	}
	value, err := service.Secrets.Fetch(context.Background(), resource.ID("payload-missing"))
	if value == nil || !gophercloud.ResponseCodeIs(err, http.StatusNotFound) || value.StatusCode != http.StatusOK || value.Header.Get("X-Request-ID") != "accepted-metadata" || string(value.Body["name"]) != `"visible"` || string(value.Body["content_types"]) != `{"default":"text/plain"}` || len(value.Body) != 2 || calls.Load() != 3 {
		t.Fatal("payload failure lost metadata or caused a resend", value, err, calls.Load())
	}
}
