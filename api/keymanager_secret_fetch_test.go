package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/keymanager/v1/secrets"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// openstacksdk ef55d7d: _proxy.py:322–333 -> Secret.fetch:91–138.
// Barbican's published secret representation and /payload GET both accept 200.
const secretFetchPath = "/reverse/barbican/v1/secrets/secret-alpha"

func secretFetchClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("key-manager", "/catalog/v1")
	client.ResourceBase = cloud.Server.URL + "/reverse/barbican/v1/"
	return client
}

type secretFetchRoundTripFunc func(*http.Request) (*http.Response, error)

func (f secretFetchRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestKeyManagerSecretFetchCompositionFixedTargetsAndEvidence(t *testing.T) {
	cloud, foreign := testcloud.New(t), testcloud.New(t)
	var metadataCalls, payloadCalls, followed, middleware atomic.Int32
	foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { followed.Add(1); w.WriteHeader(500) })
	client := secretFetchClient(cloud)
	client.MoreHeaders = map[string]string{"X-Project-Id": "fixed-project", "x-trace": "source"}
	transport := cloud.Provider.HTTPClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	cloud.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		middleware.Add(1)
		response, err := transport.RoundTrip(r)
		if r.URL.Path == secretFetchPath {
			// Apply the source change on the calling transport boundary, before
			// metadata decoding, without concurrent unsynchronized client writes.
			client.ResourceBase = cloud.Server.URL + "/changed/v1/"
		}
		return response, err
	})
	api := secrets.New(client)
	cloud.Mux.HandleFunc(secretFetchPath, func(w http.ResponseWriter, r *http.Request) {
		metadataCalls.Add(1)
		if r.Method != "GET" || r.URL.RawQuery != "" || r.Header.Get("Accept") != "application/json" || r.Header.Get("X-Project-Id") != "fixed-project" || r.Header.Get("X-Trace") != "source" {
			t.Error(r.Method, r.URL, r.Header)
		}
		cloud.Provider.SetToken("payload-token")
		w.Header().Set("X-Request-ID", "metadata-evidence")
		testcloud.JSON(w, 200, `{"name":"canonical","NAME":false,"status":"ACTIVE","STATUS":{},"secret_ref":"`+foreign.Server.URL+`/secret/other","id":null,"content_types":{"default":"text/plain"},"CONTENT_TYPES":false,"bit_length":9007199254740993,"created":"original timestamp","updated":null,"vendor":{"n":9007199254740993},"payload":"server metadata only"}`)
	})
	cloud.Mux.HandleFunc(secretFetchPath+"/payload", func(w http.ResponseWriter, r *http.Request) {
		payloadCalls.Add(1)
		if r.URL.RawQuery != "" || r.Header.Get("Accept") != "text/plain" || r.Header.Get("X-Auth-Token") != "payload-token" || r.Header.Get("X-Trace") != "source" {
			t.Error(r.URL, r.Header)
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("X-Request-ID", "payload-evidence")
		_, _ = w.Write([]byte("secret text"))
	})
	value, err := api.Fetch(context.Background(), resource.ID("secret-alpha"), secrets.WithFetchHeader("X-Trace", "call"))
	if err != nil || value == nil || value.SecretID != "secret-alpha" || value.Name != "canonical" || value.Status != "ACTIVE" || value.SecretRef != foreign.Server.URL+"/secret/other" || value.StatusCode != 200 || value.Header.Get("X-Request-ID") != "metadata-evidence" || value.Payload == nil || value.Payload.Text == nil || *value.Payload.Text != "secret text" || value.Payload.Accept != "text/plain" || value.Payload.Header.Get("Content-Type") != "application/octet-stream" || value.Payload.Header.Get("X-Request-ID") != "payload-evidence" || value.Payload.StatusCode != 200 {
		t.Fatal(value, err)
	}
	if string(value.Body["id"]) != "null" || string(value.Body["vendor"]) != `{"n":9007199254740993}` || string(value.BitLength) != "9007199254740993" || string(value.Body["payload"]) != `"server metadata only"` || string(value.Body["NAME"]) != "false" || value.CreatedAt == nil || *value.CreatedAt != "original timestamp" || value.UpdatedAt != nil {
		t.Fatal(value)
	}
	// Mutable metadata and payload evidence are independent owned values.
	value.Header.Set("X-Request-ID", "caller")
	value.Body["vendor"][0] = 'x'
	value.Payload.Body[0] = 'x'
	if value.Payload.Header.Get("X-Request-ID") != "payload-evidence" || *value.Payload.Text != "secret text" || metadataCalls.Load() != 1 || payloadCalls.Load() != 1 || followed.Load() != 0 || middleware.Load() != 2 || api.RawClient() != client || client.ProviderClient != cloud.Provider {
		t.Fatal(value, metadataCalls.Load(), payloadCalls.Load(), followed.Load(), middleware.Load())
	}
}

func TestKeyManagerSecretFetchContentTypeSelectionPolicies(t *testing.T) {
	for _, test := range []struct {
		name, metadata, accept string
		options                []secrets.FetchOption
		payload, failure       bool
	}{
		{name: "absent", metadata: `{"name":"actual"}`},
		{name: "case alias ignored", metadata: `{"CONTENT_TYPES":{"default":"text/plain"}}`},
		{name: "null default", metadata: `{"content_types":{"default":null}}`},
		{name: "null container", metadata: `{"content_types":null}`, failure: true},
		{name: "nonobject container", metadata: `{"content_types":[]}`, failure: true},
		{name: "missing default", metadata: `{"content_types":{}}`, failure: true},
		{name: "case alias default", metadata: `{"content_types":{"DEFAULT":"text/plain"}}`, failure: true},
		{name: "nonstring default", metadata: `{"content_types":{"default":false}}`, failure: true},
		{name: "response payload type does not select", metadata: `{"payload_content_type":"text/plain"}`},
		{name: "metadata chooses", metadata: `{"content_types":{"default":"application/octet-stream"}}`, payload: true, accept: "application/octet-stream"},
		{name: "empty metadata choice", metadata: `{"content_types":{"default":""}}`, payload: true},
		{name: "seed bypasses malformed container", metadata: `{"content_types":null}`, options: []secrets.FetchOption{secrets.WithFetchContentType("custom/type")}, payload: true, accept: "custom/type"},
		{name: "seed bypasses missing default", metadata: `{"content_types":{}}`, options: []secrets.FetchOption{secrets.WithFetchContentType("")}, payload: true},
		{name: "disable bypasses malformed selection", metadata: `{"content_types":null}`, options: []secrets.FetchOption{secrets.WithFetchPayload(false)}},
		{name: "disable wins seed", metadata: `{}`, options: []secrets.FetchOption{secrets.WithFetchPayload(false), secrets.WithFetchContentType("text/plain")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var metadataCalls, payloadCalls atomic.Int32
			cloud.Mux.HandleFunc(secretFetchPath, func(w http.ResponseWriter, r *http.Request) {
				metadataCalls.Add(1)
				w.Header().Set("X-Evidence", "selection")
				testcloud.JSON(w, 200, test.metadata)
			})
			cloud.Mux.HandleFunc(secretFetchPath+"/payload", func(w http.ResponseWriter, r *http.Request) {
				payloadCalls.Add(1)
				if r.Header.Get("Accept") != test.accept {
					t.Error(r.Header)
				}
				if _, exists := r.Header["Accept"]; !exists { // Empty is still an explicit chosen header.
					t.Error("Accept was omitted")
				}
				_, _ = w.Write([]byte("bytes"))
			})
			value, err := secrets.New(secretFetchClient(cloud)).Fetch(context.Background(), resource.ID("secret-alpha"), test.options...)
			if value == nil || metadataCalls.Load() != 1 || (err != nil) != test.failure || (value.Payload != nil) != test.payload || (payloadCalls.Load() == 1) != test.payload || value.StatusCode != 200 || value.Header.Get("X-Evidence") != "selection" {
				t.Fatal(value, err, metadataCalls.Load(), payloadCalls.Load())
			}
			if test.failure {
				var accepted *resource.ResponseError
				if !errors.As(err, &accepted) || string(accepted.Body) != test.metadata || accepted.StatusCode != 200 || accepted.Header.Get("X-Evidence") != "selection" {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestKeyManagerSecretFetchExactTextAndBinary(t *testing.T) {
	for _, test := range []struct {
		contentType string
		body        []byte
		text, fail  bool
	}{
		{"text/plain", []byte("\xef\xbb\xbf한글\x00"), true, false},
		{"text/plain", []byte{0xff, 0x00}, false, true},
		{"text/plain;charset=utf-8", []byte{0xff, 0x00}, false, false},
		{"application/octet-stream", []byte{0xff, 0x00}, false, false},
	} {
		t.Run(test.contentType+fmt.Sprint(test.fail), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(secretFetchPath, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, `{}`) })
			cloud.Mux.HandleFunc(secretFetchPath+"/payload", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "text/plain")
				w.Header().Set("X-Bytes", "original")
				_, _ = w.Write(test.body)
			})
			value, err := secrets.New(secretFetchClient(cloud)).Fetch(context.Background(), resource.ID("secret-alpha"), secrets.WithFetchContentType(test.contentType))
			if value == nil || value.Payload == nil || (err != nil) != test.fail || (value.Payload.Text != nil) != test.text || !reflect.DeepEqual(value.Payload.Body, test.body) || value.Payload.Accept != test.contentType || calls.Load() != 2 {
				t.Fatal(value, err, calls.Load())
			}
			if test.text && *value.Payload.Text != string(test.body) {
				t.Fatal(value.Payload.Text)
			}
			if test.fail {
				var accepted *resource.ResponseError
				if !errors.As(err, &accepted) || !reflect.DeepEqual(accepted.Body, test.body) || accepted.Header.Get("X-Bytes") != "original" || accepted.StatusCode != 200 {
					t.Fatal(err)
				}
				value.Payload.Body[0] = 0
				value.Payload.Header.Set("X-Bytes", "caller")
				if accepted.Body[0] != 0xff || accepted.Header.Get("X-Bytes") != "original" {
					t.Fatal(accepted)
				}
			}
		})
	}
}

func TestKeyManagerSecretFetchOptionSnapshotsAndConcurrentReuse(t *testing.T) {
	cloud := testcloud.New(t)
	var metadataCalls, payloadCalls atomic.Int32
	contentType, enabled := "application/octet-stream", true
	option := secrets.WithFetchOptions(secrets.FetchOpts{ContentType: &contentType, Payload: &enabled})
	contentType, enabled = "caller changed", false
	cloud.Mux.HandleFunc(secretFetchPath, func(w http.ResponseWriter, r *http.Request) {
		metadataCalls.Add(1)
		if r.Header.Get("X-Trace") != "owned" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"name":"actual","content_types":null,"vendor":9007199254740993}`)
	})
	cloud.Mux.HandleFunc(secretFetchPath+"/payload", func(w http.ResponseWriter, r *http.Request) {
		payloadCalls.Add(1)
		if r.Header.Get("Accept") != "application/octet-stream" {
			t.Error(r.Header)
		}
		_, _ = w.Write([]byte("owned bytes"))
	})
	api := secrets.New(secretFetchClient(cloud))
	options := []secrets.FetchOption{option, secrets.WithFetchHeader("X-Trace", "owned")}
	var wg sync.WaitGroup
	results := make(chan *secrets.FetchedSecret, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, err := api.Fetch(context.Background(), resource.ID("secret-alpha"), options...)
			if err != nil {
				t.Error(err)
				return
			}
			results <- value
		}()
	}
	wg.Wait()
	close(results)
	for value := range results {
		if value.Payload == nil || string(value.Payload.Body) != "owned bytes" || string(value.Body["vendor"]) != "9007199254740993" {
			t.Fatal(value)
		}
		value.Payload.Body[0] = 'x'
		value.Body["vendor"][0] = '0'
	}
	if metadataCalls.Load() != 8 || payloadCalls.Load() != 8 {
		t.Fatal(metadataCalls.Load(), payloadCalls.Load())
	}
}

func TestKeyManagerSecretFetchNameLookupAndPreparedOwnership(t *testing.T) {
	cloud := testcloud.New(t)
	client := secretFetchClient(cloud)
	var listCalls, metadataCalls, payloadCalls atomic.Int32
	var retained *request.Config[secrets.FetchOpts]
	option := secrets.FetchOption(func(config *request.Config[secrets.FetchOpts]) error {
		contentType := "owned/type"
		config.Options.ContentType = &contentType
		config.Headers = map[string]string{"X-Trace": "owned"}
		retained = config
		return nil
	})
	cloud.Mux.HandleFunc("/reverse/barbican/v1/secrets", func(w http.ResponseWriter, r *http.Request) {
		listCalls.Add(1)
		if r.URL.Query().Get("name") != "exact name" {
			testcloud.JSON(w, 200, `{"secrets":[]}`)
			return
		}
		if r.URL.Query().Get("page") == "" {
			*retained.Options.ContentType = "changed/type"
			retained.Headers["X-Trace"] = "changed"
			testcloud.JSON(w, 200, `{"secrets":[{"name":"decoy","secret_ref":"`+cloud.Server.URL+`/secrets/decoy"}],"next":"`+cloud.Server.URL+`/reverse/barbican/v1/secrets?name=exact+name&page=2"}`)
			return
		}
		testcloud.JSON(w, 200, `{"secrets":[{"name":"exact name","secret_ref":"https://foreign.invalid/secrets/secret-alpha"}]}`)
	})
	cloud.Mux.HandleFunc(secretFetchPath, func(w http.ResponseWriter, r *http.Request) {
		metadataCalls.Add(1)
		if r.Header.Get("X-Trace") != "owned" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"name":"server changed","secret_ref":"https://foreign.invalid/other","content_types":null}`)
	})
	cloud.Mux.HandleFunc(secretFetchPath+"/payload", func(w http.ResponseWriter, r *http.Request) {
		payloadCalls.Add(1)
		if r.Header.Get("Accept") != "owned/type" || r.Header.Get("X-Trace") != "owned" {
			t.Error(r.Header)
		}
		_, _ = w.Write([]byte("bytes"))
	})
	value, err := secrets.New(client).Fetch(context.Background(), resource.Name("exact name"), option)
	if err != nil || value == nil || value.SecretID != "secret-alpha" || value.Name != "server changed" || listCalls.Load() != 2 || metadataCalls.Load() != 1 || payloadCalls.Load() != 1 {
		t.Fatal(value, err, listCalls.Load(), metadataCalls.Load(), payloadCalls.Load())
	}
	// A missing explicit Name is not a heuristic direct ID fetch.
	_, err = secrets.New(client).Fetch(context.Background(), resource.Name("missing"), secrets.WithFetchPayload(false))
	if !errors.Is(err, resource.ErrNotFound) || metadataCalls.Load() != 1 || payloadCalls.Load() != 1 {
		t.Fatal(err, metadataCalls.Load(), payloadCalls.Load())
	}
}

func TestKeyManagerSecretFetchNativeCodesAndDecodeFailures(t *testing.T) {
	for _, phase := range []string{"metadata", "payload"} {
		for _, code := range []int{203, 204, 400, 403, 404, 406} {
			t.Run(phase+fmt.Sprint(code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc(secretFetchPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if phase == "metadata" {
						testcloud.JSON(w, code, `{"error":"actual"}`)
						return
					}
					testcloud.JSON(w, 200, `{"name":"metadata","content_types":{"default":"text/plain"}}`)
				})
				cloud.Mux.HandleFunc(secretFetchPath+"/payload", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					testcloud.JSON(w, code, `{"error":"payload"}`)
				})
				value, err := secrets.New(secretFetchClient(cloud)).Fetch(context.Background(), resource.ID("secret-alpha"))
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || (value != nil) != (phase == "payload") || calls.Load() != map[string]int32{"metadata": 1, "payload": 2}[phase] {
					t.Fatal(value, err, calls.Load())
				}
				if phase == "payload" && (value.Payload != nil || value.Name != "metadata" || value.StatusCode != 200) {
					t.Fatal(value)
				}
			})
		}
	}
	for _, body := range []string{`null`, `[]`, `{"name":false}`, `{"name":"truncated"`, `{"content_types":{"default":"text/plain"},"updated":false}`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(secretFetchPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Evidence", "malformed")
				testcloud.JSON(w, 200, body)
			})
			value, err := secrets.New(secretFetchClient(cloud)).Fetch(context.Background(), resource.ID("secret-alpha"))
			var accepted *resource.ResponseError
			if value != nil || !errors.As(err, &accepted) || string(accepted.Body) != body || accepted.StatusCode != 200 || accepted.Header.Get("X-Evidence") != "malformed" || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
		})
	}
}

type secretFetchReadFailure struct {
	cause error
	first bool
}

func (r *secretFetchReadFailure) Read(p []byte) (int, error) {
	if !r.first {
		r.first = true
		return copy(p, "partial bytes"), nil
	}
	return 0, r.cause
}
func (r *secretFetchReadFailure) Close() error { return nil }

func TestKeyManagerSecretFetchReadTransportAndCancellation(t *testing.T) {
	for _, phase := range []string{"metadata", "payload"} {
		for _, mode := range []string{"read", "transport404", "canceled404", "canceled200"} {
			t.Run(phase+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := secretFetchClient(cloud)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				cause := errors.New("actual read failure")
				transportCause := &gophercloud.ErrUnexpectedResponseCode{Actual: 404}
				var calls atomic.Int32
				cloud.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					isPayload := strings.HasSuffix(r.URL.Path, "/payload")
					if phase == "payload" && !isPayload {
						return &http.Response{StatusCode: 200, Header: http.Header{"X-Evidence": {"metadata"}}, Body: io.NopCloser(strings.NewReader(`{"name":"actual","content_types":{"default":"application/octet-stream"}}`)), Request: r}, nil
					}
					if mode == "transport404" {
						return nil, transportCause
					}
					code, body := 200, io.ReadCloser(&secretFetchReadFailure{cause: cause})
					if strings.HasPrefix(mode, "canceled") {
						cancel()
						body = io.NopCloser(strings.NewReader(`{"error":"actual response"}`))
						if mode == "canceled404" {
							code = 404
						}
					}
					return &http.Response{StatusCode: code, Header: http.Header{"X-Evidence": {"actual"}}, Body: body, Request: r}, nil
				})
				value, err := secrets.New(client).Fetch(ctx, resource.ID("secret-alpha"))
				if err == nil || (value != nil) != (phase == "payload") || calls.Load() != map[string]int32{"metadata": 1, "payload": 2}[phase] {
					t.Fatal(value, err, calls.Load())
				}
				if phase == "payload" && (value.Name != "actual" || value.Header.Get("X-Evidence") != "metadata") {
					t.Fatal(value)
				}
				switch mode {
				case "read", "canceled200":
					var accepted *resource.ResponseError
					if !errors.As(err, &accepted) || accepted.StatusCode != 200 || accepted.Header.Get("X-Evidence") != "actual" {
						t.Fatal(err)
					}
					if mode == "read" && (!errors.Is(err, cause) || string(accepted.Body) != "partial bytes") {
						t.Fatal(err)
					}
					if phase == "payload" && (value.Payload == nil || !reflect.DeepEqual(value.Payload.Body, accepted.Body)) {
						t.Fatal(value, err)
					}
				case "canceled404":
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != `{"error":"actual response"}` {
						t.Fatal(err)
					}
				case "transport404":
					var transport *url.Error
					if !errors.As(err, &transport) || !errors.Is(err, transportCause) {
						t.Fatal(err)
					}
				}
				if strings.HasPrefix(mode, "canceled") && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestKeyManagerSecretFetchSourceAndOptionPreflight(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, `{}`) })
	for _, test := range []struct {
		name   string
		change func(*gophercloud.ServiceClient)
		option secrets.FetchOption
	}{
		{name: "Accept owned", option: secrets.WithFetchHeader("Accept", "other")},
		{name: "auth owned", option: secrets.WithFetchHeader("X-Auth-Token", "other")},
		{name: "source Accept owned", change: func(c *gophercloud.ServiceClient) { c.MoreHeaders = map[string]string{"Accept": "other"} }},
		{name: "source case conflict", change: func(c *gophercloud.ServiceClient) {
			c.MoreHeaders = map[string]string{"X-Trace": "one", "x-trace": "two"}
		}},
		{name: "option case conflict", option: func(c *request.Config[secrets.FetchOpts]) error {
			c.Headers = map[string]string{"X-Trace": "one", "x-trace": "two"}
			return nil
		}},
		{name: "query unsupported", option: func(c *request.Config[secrets.FetchOpts]) error { c.Query.Set("vendor", "value"); return nil }},
		{name: "body unsupported", option: func(c *request.Config[secrets.FetchOpts]) error {
			c.Fields["payload"] = json.RawMessage(`"value"`)
			return nil
		}},
		{name: "argument unsupported", option: func(c *request.Config[secrets.FetchOpts]) error { c.Arguments["base_path"] = "other"; return nil }},
		{name: "content type newline", option: secrets.WithFetchContentType("text/plain\r\nX-Other:value")},
		{name: "option changes type", option: func(c *request.Config[secrets.FetchOpts]) error { return nil }, change: func(c *gophercloud.ServiceClient) { c.Type = "compute" }},
		{name: "provider nil", change: func(c *gophercloud.ServiceClient) { c.ProviderClient = nil }},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := secretFetchClient(cloud)
			if test.change != nil {
				test.change(client)
			}
			var options []secrets.FetchOption
			if test.option != nil {
				options = append(options, test.option)
			}
			value, err := secrets.New(client).Fetch(context.Background(), resource.ID("secret-alpha"), options...)
			if value != nil || err == nil || calls.Load() != 0 {
				t.Fatal(value, err, calls.Load())
			}
		})
	}
	for _, ref := range []resource.Ref{resource.ID(""), resource.ID("a/b"), resource.ID("a%2Fb"), resource.ID("a\x00b"), resource.ID("a\u2003b")} {
		if _, err := secrets.New(secretFetchClient(cloud)).Fetch(context.Background(), ref); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(ref, err)
		}
	}
	var nilAPI *secrets.API
	if _, err := nilAPI.Fetch(context.Background(), resource.ID("secret-alpha")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := secrets.New(secretFetchClient(cloud)).Fetch(nil, resource.ID("secret-alpha")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := secrets.New(secretFetchClient(cloud)).Fetch(ctx, resource.ID("secret-alpha")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := secrets.New(secretFetchClient(cloud)).Fetch(context.Background(), resource.ID("secret-alpha"), nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
	t.Run("equal header case variants", func(t *testing.T) {
		fixture := testcloud.New(t)
		client := secretFetchClient(fixture)
		client.MoreHeaders = map[string]string{"X-Source": "equal", "x-source": "equal"}
		var count atomic.Int32
		fixture.Mux.HandleFunc(secretFetchPath, func(w http.ResponseWriter, r *http.Request) {
			count.Add(1)
			if r.Header.Get("X-Source") != "equal" || r.Header.Get("X-Extra") != "equal" {
				t.Error(r.Header)
			}
			testcloud.JSON(w, 200, `{}`)
		})
		value, err := secrets.New(client).Fetch(context.Background(), resource.ID("secret-alpha"), func(config *request.Config[secrets.FetchOpts]) error {
			config.Headers = map[string]string{"X-Extra": "equal", "x-extra": "equal"}
			return nil
		})
		if err != nil || value == nil || count.Load() != 1 || len(client.MoreHeaders) != 2 {
			t.Fatal(value, err, count.Load())
		}
	})
	t.Run("rechecks after options and between steps", func(t *testing.T) {
		fixture := testcloud.New(t)
		client := secretFetchClient(fixture)
		var metadata, payload atomic.Int32
		fixture.Mux.HandleFunc(secretFetchPath, func(w http.ResponseWriter, r *http.Request) {
			metadata.Add(1)
			testcloud.JSON(w, 200, `{"name":"actual","content_types":{"default":"text/plain"}}`)
		})
		fixture.Mux.HandleFunc(secretFetchPath+"/payload", func(w http.ResponseWriter, r *http.Request) { payload.Add(1) })
		transport := fixture.Provider.HTTPClient.Transport
		if transport == nil {
			transport = http.DefaultTransport
		}
		fixture.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
			response, err := transport.RoundTrip(r)
			client.Type = "compute"
			return response, err
		})
		api := secrets.New(client)
		value, err := api.Fetch(context.Background(), resource.ID("secret-alpha"))
		if value == nil || value.Name != "actual" || !errors.Is(err, resource.ErrUnsupported) || metadata.Load() != 1 || payload.Load() != 0 {
			t.Fatal(value, err, metadata.Load(), payload.Load())
		}
		client.Type = "key-manager"
		value, err = api.Fetch(context.Background(), resource.ID("secret-alpha"), func(c *request.Config[secrets.FetchOpts]) error { client.Type = "compute"; return nil })
		if value != nil || !errors.Is(err, resource.ErrUnsupported) || metadata.Load() != 1 {
			t.Fatal(value, err, metadata.Load())
		}
	})
}
