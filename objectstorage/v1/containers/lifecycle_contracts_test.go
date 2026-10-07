package containers_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	swift "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1"
)

// These fixtures reuse only the independently frozen metadata_contracts_test.go
// transport, body, provider and response helpers owned by this external package.
func containerLifecycleRequest(t *testing.T, r *http.Request, method string) {
	t.Helper()
	containerMetadataNoBody(t, r)
	if r.Method != method {
		t.Errorf("method=%s want=%s", r.Method, method)
	}
}

func containerLifecycleResult(t *testing.T, result *containers.ContainerResponse, status int, body string, ignored bool) {
	t.Helper()
	if result == nil || result.StatusCode != status || string(result.Body) != body || result.Header.Get("X-Trans-Id") != "actual-container" || result.IgnoredMissing != ignored {
		t.Fatalf("result=%+v want status=%d body=%q ignored=%t", result, status, body, ignored)
	}
}

func containerLifecycleNativeError(t *testing.T, err error, method string, status int, body string) gophercloud.ErrUnexpectedResponseCode {
	t.Helper()
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Method != method || native.URL != containerMetadataEndpoint || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Trans-Id") != "actual-container" {
		t.Fatalf("native proof=%+v err=%v", native, err)
	}
	return native
}

func TestContainerLifecycleContractsWireAndIdentity(t *testing.T) {
	t.Run("create actual 201 and 202 bodyless without hidden read", func(t *testing.T) {
		for _, status := range []int{201, 202} {
			for _, metadata := range []map[string]string{nil, {}, {"Book": "  한글 %2F\t", "Empty": "", "Read_acl": "literal custom"}} {
				calls := 0
				client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
					calls++
					containerLifecycleRequest(t, r, "PUT")
					if r.Header.Get("X-Container-Read") != ".r:*" || r.Header.Get("X-Detect-Content-Type") != "false" || len(r.Header.Values("X-Versions-Location")) != 1 || r.Header.Get("X-Versions-Location") != "" {
						t.Errorf("system fields=%v", r.Header)
					}
					for key, value := range metadata {
						if got := r.Header.Values("X-Container-Meta-" + key); !reflect.DeepEqual(got, []string{value}) {
							t.Errorf("metadata %s=%v want=%q", key, got, value)
						}
					}
					for key := range r.Header {
						if strings.HasPrefix(strings.ToLower(key), "x-container-meta-") && len(metadata) == 0 {
							t.Errorf("nil/empty metadata sent %s", key)
						}
					}
					return containerMetadataResponse(status, "opaque acknowledgement"), nil
				})
				result, err := containers.New(client).CreateContainer(context.Background(), containerMetadataName,
					containers.WithCreateContainerMetadata(metadata),
					containers.WithCreateContainerHeaders(map[string]string{"X-Container-Read": ".r:*", "X-Detect-Content-Type": "false", "X-Versions-Location": ""}))
				if err != nil || calls != 1 {
					t.Fatalf("err=%v requests=%d", err, calls)
				}
				containerLifecycleResult(t, result, status, "opaque acknowledgement", false)
			}
		}
	})

	t.Run("delete default and explicit missing states", func(t *testing.T) {
		for _, tc := range []struct {
			status int
			flag   *bool
		}{{204, nil}, {404, nil}, {404, lifecycleBool(true)}, {204, lifecycleBool(false)}, {404, lifecycleBool(false)}} {
			calls, hooks := 0, 0
			client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				containerLifecycleRequest(t, r, "DELETE")
				return containerMetadataResponse(tc.status, "delete proof"), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				hooks++
				return original
			}
			result, err := containers.New(client).DeleteContainer(context.Background(), containerMetadataName, containers.WithDeleteContainerOpts(containers.DeleteContainerOpts{IgnoreMissing: tc.flag}))
			if tc.status == 404 && tc.flag != nil && !*tc.flag {
				if result != nil || err == nil || hooks != 1 {
					t.Fatalf("strict result=%+v err=%v hooks=%d", result, err, hooks)
				}
				containerLifecycleNativeError(t, err, "DELETE", 404, "delete proof")
			} else {
				if err != nil || hooks != 0 {
					t.Fatalf("result=%+v err=%v hooks=%d", result, err, hooks)
				}
				containerLifecycleResult(t, result, tc.status, "delete proof", tc.status == 404)
			}
			if calls != 1 {
				t.Fatalf("requests=%d", calls)
			}
		}
	})

	t.Run("literal name and encoded effective base once on actual wire", func(t *testing.T) {
		for _, name := range []string{containerMetadataName, " %2F?# ", "x:y", ".hidden", strings.Repeat("x", 300)} {
			for _, method := range []string{"PUT", "DELETE"} {
				calls := 0
				client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
					calls++
					expected := "https://swift.invalid/reverse/p%2Fq/v1/AUTH_other/" + url.PathEscape(name)
					parsed, parseErr := url.Parse(expected)
					if parseErr != nil || r.Method != method || r.URL.String() != expected || r.URL.EscapedPath() != parsed.EscapedPath() || r.URL.RequestURI() != parsed.RequestURI() || r.URL.RawQuery != "" || r.Body != nil {
						t.Errorf("request=%s %s URI=%s body=%v", r.Method, r.URL, r.URL.RequestURI(), r.Body)
					}
					status := 204
					if method == "PUT" {
						status = 201
					}
					return containerMetadataResponse(status, ""), nil
				})
				client.Endpoint, client.ResourceBase = "https://swift.invalid/endpoint", "https://swift.invalid/reverse/p%2Fq/v1/AUTH_other/"
				api := containers.New(client)
				var err error
				if method == "PUT" {
					_, err = api.CreateContainer(context.Background(), name)
				} else {
					_, err = api.DeleteContainer(context.Background(), name)
				}
				if err != nil || calls != 1 {
					t.Fatalf("name=%q err=%v requests=%d", name, err, calls)
				}
			}
		}
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			n := calls.Add(1)
			if r.RequestURI != "/reverse/p%2Fq/v1/AUTH_account/"+url.PathEscape(containerMetadataName) || r.Header.Get("X-Auth-Token") != "original-token" {
				t.Errorf("request=%s headers=%v", r.RequestURI, r.Header)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil || len(body) != 0 || n > 2 || (n == 1 && r.Method != "PUT") || (n == 2 && r.Method != "DELETE") {
				t.Errorf("request %d %s body=%q err=%v", n, r.Method, body, err)
			}
			w.Header().Set("X-Trans-Id", "wire")
			if n == 1 {
				w.WriteHeader(201)
			} else {
				w.WriteHeader(204)
			}
		}))
		defer server.Close()
		client := containerMetadataClient(nil)
		client.HTTPClient = *server.Client()
		client.Endpoint, client.ResourceBase = server.URL+"/endpoint/", server.URL+"/reverse/p%2Fq/v1/AUTH_account/"
		api := containers.New(client)
		if result, err := api.CreateContainer(context.Background(), containerMetadataName); err != nil || result == nil || result.StatusCode != 201 {
			t.Fatalf("PUT=%+v err=%v", result, err)
		}
		if result, err := api.DeleteContainer(context.Background(), containerMetadataName); err != nil || result == nil || result.StatusCode != 204 {
			t.Fatalf("DELETE=%+v err=%v", result, err)
		}
		if calls.Load() != 2 {
			t.Fatalf("requests=%d", calls.Load())
		}
	})

	t.Run("exact ordinary system headers no aliases or value coercion", func(t *testing.T) {
		headers := map[string]string{"Content-Type": "application/custom", "X-Container-Sync-To": "https://other.invalid/v1/a/b", "X-Container-Sync-Key": "", "X-Container-Write": "project:role", "X-Storage-Policy": "literal", "X-History-Location": "old%2Fvalue"}
		client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
			containerLifecycleRequest(t, r, "PUT")
			for key, value := range headers {
				if !reflect.DeepEqual(r.Header.Values(key), []string{value}) {
					t.Errorf("%s=%v", key, r.Header.Values(key))
				}
			}
			if r.Header.Get("X-Container-Meta-Temp-Url-Key") != "secret" || r.Header.Get("X-Container-Meta-Sync_key") != "custom" || len(r.Header.Values("X-Container-Sync-Keys")) != 0 {
				t.Errorf("headers=%v", r.Header)
			}
			return containerMetadataResponse(202, ""), nil
		})
		result, err := containers.New(client).CreateContainer(context.Background(), containerMetadataName, containers.WithCreateContainerHeaders(headers), containers.WithCreateContainerMetadata(map[string]string{"Temp-URL-Key": "secret", "Sync_key": "custom"}))
		if err != nil {
			t.Fatal(err)
		}
		containerLifecycleResult(t, result, 202, "", false)
	})
}

func lifecycleBool(value bool) *bool { return &value }

func TestContainerLifecycleContractsPreflightAndOptions(t *testing.T) {
	t.Run("source context and literal identity preflight before callbacks", func(t *testing.T) {
		for _, name := range []string{"", ".", "..", "a/b", "a\\b", "a\n", "a\x00", "a\x7f", string([]byte{0xff})} {
			for _, create := range []bool{true, false} {
				calls, callbacks := 0, 0
				client := containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
				api := containers.New(client)
				var err error
				if create {
					_, err = api.CreateContainer(context.Background(), name, func(*containers.CreateContainerOpts) error { callbacks++; return nil })
				} else {
					_, err = api.DeleteContainer(context.Background(), name, func(*containers.DeleteContainerOpts) error { callbacks++; return nil })
				}
				if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 {
					t.Fatalf("name=%q create=%t err=%v calls=%d callbacks=%d", name, create, err, calls, callbacks)
				}
				if name == "" {
					var native swift.ErrEmptyContainerName
					if !errors.As(err, &native) {
						t.Fatalf("lost native empty cause: %v", err)
					}
				}
				if name == "a/b" {
					var native swift.ErrInvalidContainerName
					if !errors.As(err, &native) {
						t.Fatalf("lost native slash cause: %v", err)
					}
				}
			}
		}
		for _, tc := range []struct {
			name   string
			change func(*gophercloud.ServiceClient)
		}{
			{"nil provider", func(c *gophercloud.ServiceClient) { c.ProviderClient = nil }},
			{"wrong service", func(c *gophercloud.ServiceClient) { c.Type = "compute" }},
			{"query endpoint", func(c *gophercloud.ServiceClient) { c.Endpoint += "?x=1" }},
			{"force query", func(c *gophercloud.ServiceClient) { c.Endpoint += "?" }},
			{"user info", func(c *gophercloud.ServiceClient) { c.Endpoint = "https://user@swift.invalid/" }},
			{"foreign base", func(c *gophercloud.ServiceClient) { c.ResourceBase = "https://else.invalid/" }},
			{"base without slash", func(c *gophercloud.ServiceClient) { c.ResourceBase = "https://swift.invalid/base" }},
			{"base fragment", func(c *gophercloud.ServiceClient) { c.ResourceBase = "https://swift.invalid/base/#x" }},
			{"bad version", func(c *gophercloud.ServiceClient) { c.Microversion = "x\n" }},
		} {
			calls, callbacks := 0, 0
			client := containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
			tc.change(client)
			_, err := containers.New(client).CreateContainer(context.Background(), containerMetadataName, func(*containers.CreateContainerOpts) error { callbacks++; return nil })
			want := resource.ErrInvalidOption
			if tc.name == "wrong service" {
				want = resource.ErrUnsupported
			}
			if !errors.Is(err, want) || calls != 0 || callbacks != 0 {
				t.Fatalf("%s err=%v calls=%d callbacks=%d", tc.name, err, calls, callbacks)
			}
		}
		var api *containers.API
		if result, err := api.DeleteContainer(context.Background(), containerMetadataName); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil API=%+v err=%v", result, err)
		}
		for _, ctx := range []context.Context{nil, func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }()} {
			calls, callbacks := 0, 0
			client := containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
			_, err := containers.New(client).DeleteContainer(ctx, containerMetadataName, func(*containers.DeleteContainerOpts) error { callbacks++; return nil })
			if err == nil || calls != 0 || callbacks != 0 {
				t.Fatalf("context err=%v calls=%d callbacks=%d", err, calls, callbacks)
			}
		}
	})

	t.Run("original tokens field values aliases and protected headers", func(t *testing.T) {
		for _, metadata := range []map[string]string{{"": "x"}, {"K": "x"}, {"a b": "x"}, {"X-Container-Meta-Book": "x"}, {"Book": "x", "book": "y"}, {"Book": "x\r\nInjected: yes"}, {"Book": "x\x7f"}, {"Book": string([]byte{0xff})}} {
			calls := 0
			client := containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
			if result, err := containers.New(client).CreateContainer(context.Background(), containerMetadataName, containers.WithCreateContainerMetadata(metadata)); result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatalf("metadata=%v result=%+v err=%v calls=%d", metadata, result, err, calls)
			}
		}
		for _, key := range []string{"Authorization", "X-Auth-Token", "Host", "Content-Length", "Transfer-Encoding", "Connection", "Proxy-Connection", "Proxy-Authorization", "Upgrade", "Trailer", "Te", "X-Newest", "X-Container-Meta-Book", "X-Remove-Container-Meta-Book", "K", "bad key"} {
			for _, source := range []bool{true, false} {
				calls := 0
				client := containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
				var options []containers.DeleteContainerOption
				if source {
					client.MoreHeaders = map[string]string{key: "value"}
				} else {
					options = []containers.DeleteContainerOption{containers.WithDeleteContainerHeader(key, "value")}
				}
				if result, err := containers.New(client).DeleteContainer(context.Background(), containerMetadataName, options...); result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
					t.Fatalf("key=%s source=%t result=%+v err=%v calls=%d", key, source, result, err, calls)
				}
			}
		}
		client := containerMetadataClient(func(*http.Request) (*http.Response, error) {
			t.Error("nil option reached HTTP")
			return nil, errors.New("HTTP")
		})
		api := containers.New(client)
		if _, err := api.CreateContainer(context.Background(), containerMetadataName, nil); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil create option: %v", err)
		}
		if _, err := api.DeleteContainer(context.Background(), containerMetadataName, nil); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil delete option: %v", err)
		}
		for _, headers := range []map[string]string{{"x-call": "one", "X-Call": "two"}, {"X-Call": "bad\nvalue"}} {
			client := containerMetadataClient(func(*http.Request) (*http.Response, error) {
				t.Error("unexpected HTTP")
				return nil, errors.New("HTTP")
			})
			if _, err := containers.New(client).CreateContainer(context.Background(), containerMetadataName, containers.WithCreateContainerHeaders(headers)); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("headers=%v err=%v", headers, err)
			}
		}
	})

	t.Run("all options final replacement canonical override and snapshots", func(t *testing.T) {
		headers, metadata := map[string]string{"x-call": "factory", "X-Gone": "gone"}, map[string]string{"Old": "gone"}
		full := containers.WithCreateContainerOpts(containers.CreateContainerOpts{Headers: headers, Metadata: metadata})
		headers["x-call"], metadata["Old"] = "caller mutation", "caller mutation"
		replacement := map[string]string{"Book": "snapshot", "Empty": ""}
		set := containers.WithCreateContainerMetadata(replacement)
		replacement["Book"] = "caller mutation"
		var retained *containers.CreateContainerOpts
		calls, callbacks := 0, 0
		client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			containerLifecycleRequest(t, r, "PUT")
			if r.Header.Get("X-Call") != "final" || len(r.Header.Values("X-Call")) != 1 || r.Header.Get("X-Plural") != "plural" || r.Header.Get("X-Container-Meta-Book") != "snapshot" || len(r.Header.Values("X-Container-Meta-Old")) != 0 {
				t.Errorf("headers=%v", r.Header)
			}
			return containerMetadataResponse(201, ""), nil
		})
		result, err := containers.New(client).CreateContainer(context.Background(), containerMetadataName, full, set,
			func(o *containers.CreateContainerOpts) error { callbacks++; retained = o; return nil },
			func(*containers.CreateContainerOpts) error {
				callbacks++
				retained.Headers["X-Call"] = "retained mutation"
				retained.Metadata["Book"] = "retained mutation"
				return nil
			},
			containers.WithCreateContainerHeaders(map[string]string{"x-call": "plural", "X-Plural": "plural"}), containers.WithCreateContainerHeader("X-Call", "final"))
		if err != nil || result == nil || calls != 1 || callbacks != 2 {
			t.Fatalf("result=%+v err=%v calls=%d callbacks=%d", result, err, calls, callbacks)
		}
		flag := true
		original := map[string]string{"x-call": "full"}
		deleteFull := containers.WithDeleteContainerOpts(containers.DeleteContainerOpts{Headers: original, IgnoreMissing: &flag})
		flag = false
		original["x-call"] = "mutated"
		client.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) {
			containerLifecycleRequest(t, r, "DELETE")
			if r.Header.Get("X-Call") != "final" || len(r.Header.Values("X-Call")) != 1 {
				t.Errorf("headers=%v", r.Header)
			}
			return containerMetadataResponse(404, "missing"), nil
		})
		result, err = containers.New(client).DeleteContainer(context.Background(), containerMetadataName, containers.WithDeleteContainerIgnoreMissing(false), deleteFull, containers.WithDeleteContainerHeaders(map[string]string{"x-call": "plural"}), containers.WithDeleteContainerHeader("X-Call", "final"))
		if err != nil {
			t.Fatal(err)
		}
		containerLifecycleResult(t, result, 404, "missing", true)
		for _, create := range []bool{true, false} {
			client.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) {
				if r.Header.Get("X-Gone") != "" || len(r.Header.Values("X-Container-Meta-Old")) != 0 {
					t.Errorf("full reset retained previous fields=%v", r.Header)
				}
				status := 204
				if create {
					status = 201
				}
				return containerMetadataResponse(status, ""), nil
			})
			if create {
				_, err = containers.New(client).CreateContainer(context.Background(), containerMetadataName, full, containers.WithCreateContainerOpts(containers.CreateContainerOpts{}))
			} else {
				_, err = containers.New(client).DeleteContainer(context.Background(), containerMetadataName, deleteFull, containers.WithDeleteContainerOpts(containers.DeleteContainerOpts{}))
			}
			if err != nil {
				t.Fatal(err)
			}
		}
	})

	t.Run("callback errors cancellation source drift and parallel reuse", func(t *testing.T) {
		cause := errors.New("callback cause")
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		calls, callbacks := 0, 0
		client := containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
		_, err := containers.New(client).CreateContainer(ctx, containerMetadataName, func(*containers.CreateContainerOpts) error {
			callbacks++
			client.Endpoint = "https://swift.invalid/drift/"
			cancel(cause)
			return cause
		})
		if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 1 {
			t.Fatalf("err=%v calls=%d callbacks=%d", err, calls, callbacks)
		}
		client = containerMetadataClient(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Call") != "parallel" || r.Header.Get("X-Container-Meta-Book") != "same" || r.Header.Get("X-Auth-Token") != "latest" {
				t.Errorf("parallel headers=%v", r.Header)
			}
			return containerMetadataResponse(201, ""), nil
		})
		client.SetToken("latest")
		api := containers.New(client)
		options := []containers.CreateContainerOption{containers.WithCreateContainerHeaders(map[string]string{"x-call": "parallel"}), containers.WithCreateContainerMetadata(map[string]string{"Book": "same"})}
		var wg sync.WaitGroup
		for i := 0; i < 12; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if result, err := api.CreateContainer(context.Background(), containerMetadataName, options...); err != nil || result == nil {
					t.Errorf("parallel=%+v err=%v", result, err)
				}
			}()
		}
		wg.Wait()
	})
}

func TestContainerLifecycleContractsAcknowledgementOwnership(t *testing.T) {
	t.Run("opaque acknowledgement no date numeric or JSON decode", func(t *testing.T) {
		for _, status := range []int{201, 202, 204, 404} {
			body := &containerMetadataBody{Reader: strings.NewReader(string([]byte{0xff, 0x00, 'x'}))}
			client := containerMetadataClient(func(*http.Request) (*http.Response, error) {
				response := containerMetadataWire(status, body)
				response.Header["Date"] = []string{"not a date"}
				response.Header["Content-Length"] = []string{"not numeric"}
				return response, nil
			})
			api := containers.New(client)
			var result *containers.ContainerResponse
			var err error
			if status < 204 {
				result, err = api.CreateContainer(context.Background(), containerMetadataName)
			} else {
				result, err = api.DeleteContainer(context.Background(), containerMetadataName)
			}
			if err != nil {
				t.Fatal(err)
			}
			containerLifecycleResult(t, result, status, string([]byte{0xff, 0x00, 'x'}), status == 404)
			if result.Header.Get("Date") != "not a date" || body.closes.Load() != 1 {
				t.Fatalf("result=%+v closes=%d", result, body.closes.Load())
			}
		}
	})

	t.Run("accepted partial read close and custom context retained once", func(t *testing.T) {
		for _, status := range []int{201, 202, 204} {
			readCause, closeCause, cancelCause := errors.New("read failure"), errors.New("close failure"), errors.New("cancel cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			body := &containerMetadataBody{Reader: containerMetadataReader(func(p []byte) (int, error) { return copy(p, "partial"), readCause }), closeErr: closeCause, onClose: func() { cancel(cancelCause) }}
			calls, hooks := 0, 0
			client := containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return containerMetadataWire(status, body), nil })
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				hooks++
				return original
			}
			api := containers.New(client)
			var result *containers.ContainerResponse
			var err error
			if status == 204 {
				result, err = api.DeleteContainer(ctx, containerMetadataName)
			} else {
				result, err = api.CreateContainer(ctx, containerMetadataName)
			}
			containerLifecycleResult(t, result, status, "partial", false)
			containerMetadataProof(t, err, status, "partial")
			for _, cause := range []error{readCause, closeCause, cancelCause, context.Canceled} {
				if !errors.Is(err, cause) {
					t.Fatalf("lost %v: %v", cause, err)
				}
			}
			if calls != 1 || hooks != 0 || body.closes.Load() != 1 {
				t.Fatalf("calls=%d hooks=%d closes=%d", calls, hooks, body.closes.Load())
			}
		}
	})

	t.Run("proof headers body and error snapshots independent", func(t *testing.T) {
		closeCause := errors.New("close")
		body := &containerMetadataBody{Reader: strings.NewReader("proof"), closeErr: closeCause}
		response := containerMetadataWire(201, body)
		client := containerMetadataClient(func(*http.Request) (*http.Response, error) { return response, nil })
		result, err := containers.New(client).CreateContainer(context.Background(), containerMetadataName)
		containerLifecycleResult(t, result, 201, "proof", false)
		proof := containerMetadataProof(t, err, 201, "proof")
		response.Header.Set("X-Trans-Id", "wire mutation")
		result.Header.Set("X-Trans-Id", "caller mutation")
		result.Body[0] = 'X'
		if proof.Header.Get("X-Trans-Id") != "actual-container" || string(proof.Body) != "proof" || !errors.Is(err, closeCause) {
			t.Fatalf("proof alias=%+v err=%v", proof, err)
		}
		proof.Body[1] = 'Y'
		proof.Header.Set("X-Proof", "error only")
		if string(result.Body) != "Xroof" || result.Header.Get("X-Proof") != "" {
			t.Fatalf("result alias=%+v", result)
		}
	})

	t.Run("expanded native codes cannot change actual success evidence", func(t *testing.T) {
		for _, tc := range []struct {
			create   bool
			status   int
			expected []int
		}{{true, 204, []int{201, 202}}, {true, 200, []int{201, 202}}, {false, 202, []int{204, 404}}} {
			closeCause := errors.New("unexpected close")
			accepted := &containerMetadataBody{Reader: strings.NewReader("unexpected proof"), closeErr: closeCause}
			rejected := &containerMetadataBody{Reader: strings.NewReader("busy")}
			calls, hooks := 0, 0
			client := containerMetadataClient(func(*http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return containerMetadataWire(503, rejected), nil
				}
				return containerMetadataWire(tc.status, accepted), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				hooks++
				opts.OkCodes = append(opts.OkCodes, tc.status)
				return nil
			}
			api := containers.New(client)
			var result *containers.ContainerResponse
			var err error
			method := "DELETE"
			if tc.create {
				method = "PUT"
				result, err = api.CreateContainer(context.Background(), containerMetadataName)
			} else {
				result, err = api.DeleteContainer(context.Background(), containerMetadataName)
			}
			if result != nil || !errors.Is(err, closeCause) || calls != 2 || hooks != 1 || accepted.closes.Load() != 1 || rejected.closes.Load() != 1 {
				t.Fatalf("result=%+v err=%v calls=%d hooks=%d closes=%d/%d", result, err, calls, hooks, accepted.closes.Load(), rejected.closes.Load())
			}
			native := containerLifecycleNativeError(t, err, method, tc.status, "unexpected proof")
			if !reflect.DeepEqual(native.Expected, tc.expected) {
				t.Fatalf("Expected=%v want=%v", native.Expected, tc.expected)
			}
		}
	})
}

func TestContainerLifecycleContractsMissingAndFailures(t *testing.T) {
	t.Run("owned 404 handling failures raw result never marked ignored", func(t *testing.T) {
		for _, causeKind := range []string{"read", "close", "context", "source"} {
			cause := errors.New(causeKind)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			body := &containerMetadataBody{Reader: strings.NewReader("missing proof")}
			calls, hooks := 0, 0
			var client *gophercloud.ServiceClient
			client = containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return containerMetadataWire(404, body), nil })
			switch causeKind {
			case "read":
				body.Reader = containerMetadataReader(func(p []byte) (int, error) { return copy(p, "missing proof"), cause })
			case "close":
				body.closeErr = cause
			case "context":
				body.onClose = func() { cancel(cause) }
			case "source":
				body.onClose = func() { client.ResourceBase = "https://swift.invalid/drift/" }
			}
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				hooks++
				return original
			}
			result, err := containers.New(client).DeleteContainer(ctx, containerMetadataName)
			containerLifecycleResult(t, result, 404, "missing proof", false)
			containerMetadataProof(t, err, 404, "missing proof")
			if causeKind == "source" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatalf("source err=%v", err)
				}
			} else if !errors.Is(err, cause) {
				t.Fatalf("lost %v err=%v", cause, err)
			}
			if calls != 1 || hooks != 0 || body.closes.Load() != 1 {
				t.Fatalf("calls=%d hooks=%d closes=%d", calls, hooks, body.closes.Load())
			}
		}
	})

	t.Run("transport callback and context nested 404 remain errors", func(t *testing.T) {
		nested := &gophercloud.ErrUnexpectedResponseCode{Actual: 404, Method: "DELETE", URL: containerMetadataEndpoint, Body: []byte("nested")}
		for _, fromHook := range []bool{false, true} {
			calls, hooks := 0, 0
			client := containerMetadataClient(func(*http.Request) (*http.Response, error) {
				calls++
				if !fromHook {
					return nil, nested
				}
				return containerMetadataResponse(503, "busy"), nil
			})
			if fromHook {
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, _ error, _ uint) error {
					hooks++
					return nested
				}
			}
			result, err := containers.New(client).DeleteContainer(context.Background(), containerMetadataName)
			if result != nil || !errors.Is(err, nested) || calls != 1 || (fromHook && hooks != 1) {
				t.Fatalf("result=%+v err=%v calls=%d hooks=%d", result, err, calls, hooks)
			}
		}
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		body := &containerMetadataBody{Reader: strings.NewReader("missing"), onClose: func() { cancel(nested) }}
		client := containerMetadataClient(func(*http.Request) (*http.Response, error) { return containerMetadataWire(404, body), nil })
		result, err := containers.New(client).DeleteContainer(ctx, containerMetadataName)
		containerLifecycleResult(t, result, 404, "missing", false)
		if !errors.Is(err, nested) || !errors.Is(err, context.Canceled) {
			t.Fatalf("context err=%v", err)
		}
	})

	t.Run("server missing forbidden conflict and quota no purge or fallback", func(t *testing.T) {
		for _, tc := range []struct {
			create bool
			status int
		}{{true, 400}, {true, 404}, {true, 409}, {true, 507}, {false, 403}, {false, 409}, {false, 202}} {
			calls := 0
			client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if tc.create {
					containerLifecycleRequest(t, r, "PUT")
				} else {
					containerLifecycleRequest(t, r, "DELETE")
				}
				return containerMetadataResponse(tc.status, "server failure"), nil
			})
			api := containers.New(client)
			var result *containers.ContainerResponse
			var err error
			method := "DELETE"
			if tc.create {
				method = "PUT"
				result, err = api.CreateContainer(context.Background(), containerMetadataName)
			} else {
				result, err = api.DeleteContainer(context.Background(), containerMetadataName)
			}
			if result != nil || err == nil || calls != 1 {
				t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
			}
			containerLifecycleNativeError(t, err, method, tc.status, "server failure")
		}
	})

	t.Run("ordinary prebody retries and terminal hook causes", func(t *testing.T) {
		for _, transport := range []bool{false, true} {
			cause := errors.New("transport temporary")
			calls, hooks := 0, 0
			client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				containerLifecycleRequest(t, r, "PUT")
				if calls == 1 {
					if transport {
						return nil, cause
					}
					return containerMetadataResponse(503, "busy"), nil
				}
				return containerMetadataResponse(202, "retried"), nil
			})
			client.RetryFunc = func(_ context.Context, method, endpoint string, opts *gophercloud.RequestOpts, original error, _ uint) error {
				hooks++
				if method != "PUT" || endpoint != containerMetadataEndpoint || opts.JSONBody != nil || opts.RawBody != nil || !opts.KeepResponseBody {
					t.Errorf("hook=%s %s opts=%+v", method, endpoint, opts)
				}
				if transport && !errors.Is(original, cause) {
					t.Errorf("original=%v", original)
				}
				return nil
			}
			result, err := containers.New(client).CreateContainer(context.Background(), containerMetadataName)
			if err != nil || calls != 2 || hooks != 1 {
				t.Fatalf("result=%+v err=%v calls=%d hooks=%d", result, err, calls, hooks)
			}
			containerLifecycleResult(t, result, 202, "retried", false)
		}
		cause := errors.New("hook terminal")
		calls := 0
		client := containerMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			return containerMetadataResponse(503, "busy"), nil
		})
		client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, _ error, _ uint) error { return cause }
		result, err := containers.New(client).DeleteContainer(context.Background(), containerMetadataName)
		if result != nil || !errors.Is(err, cause) || calls != 1 {
			t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
		}
		containerLifecycleNativeError(t, err, "DELETE", 503, "busy")

		calls, authCalls := 0, 0
		client = containerMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return containerMetadataResponse(401, "unauthorized"), nil
			}
			if r.Header.Get("X-Auth-Token") != "refreshed" || r.Method != "PUT" || r.URL.String() != containerMetadataEndpoint || r.Body != nil {
				t.Errorf("reauth request=%s %s headers=%v", r.Method, r.URL, r.Header)
			}
			return containerMetadataResponse(201, "reauthenticated"), nil
		})
		client.ReauthFunc = func(context.Context) error { authCalls++; client.SetToken("refreshed"); return nil }
		result, err = containers.New(client).CreateContainer(context.Background(), containerMetadataName)
		if err != nil || calls != 2 || authCalls != 1 {
			t.Fatalf("reauth result=%+v err=%v calls=%d auth=%d", result, err, calls, authCalls)
		}
		containerLifecycleResult(t, result, 201, "reauthenticated", false)
		reauthCause := errors.New("reauth failure")
		client = containerMetadataClient(func(*http.Request) (*http.Response, error) {
			return containerMetadataResponse(401, "unauthorized"), nil
		})
		client.ReauthFunc = func(context.Context) error { return reauthCause }
		result, err = containers.New(client).DeleteContainer(context.Background(), containerMetadataName)
		var authFailure *gophercloud.ErrUnableToReauthenticate
		if result != nil || !errors.As(err, &authFailure) || authFailure.ErrReauth != reauthCause {
			t.Fatalf("reauth failure=%+v err=%v", authFailure, err)
		}
		containerLifecycleNativeError(t, authFailure.ErrOriginal, "DELETE", 401, "unauthorized")
	})
}

func TestContainerLifecycleContractsSourceAndNativePolicy(t *testing.T) {
	t.Run("captured source target and late drift cannot retarget", func(t *testing.T) {
		for _, change := range []func(*gophercloud.ServiceClient){func(c *gophercloud.ServiceClient) { c.Endpoint += "drift/" }, func(c *gophercloud.ServiceClient) { c.ResourceBase = "https://swift.invalid/new/" }, func(c *gophercloud.ServiceClient) { c.Type = "compute" }, func(c *gophercloud.ServiceClient) { c.Microversion = "changed" }, func(c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} }} {
			calls := 0
			client := containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
			result, err := containers.New(client).DeleteContainer(context.Background(), containerMetadataName, func(*containers.DeleteContainerOpts) error { change(client); return nil })
			if result != nil || err == nil || calls != 0 {
				t.Fatalf("preflight result=%+v err=%v calls=%d", result, err, calls)
			}
			client = containerMetadataClient(nil)
			body := &containerMetadataBody{Reader: strings.NewReader("accepted"), onClose: func() { change(client) }}
			client.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) {
				containerLifecycleRequest(t, r, "PUT")
				return containerMetadataWire(201, body), nil
			})
			result, err = containers.New(client).CreateContainer(context.Background(), containerMetadataName)
			containerLifecycleResult(t, result, 201, "accepted", false)
			containerMetadataProof(t, err, 201, "accepted")
			if err == nil || body.closes.Load() != 1 {
				t.Fatalf("err=%v closes=%d", err, body.closes.Load())
			}
		}
		for _, location := range []string{"https://else.invalid/target", containerMetadataEndpoint + "?retarget=1", containerMetadataBase + "other"} {
			calls := 0
			client := containerMetadataClient(func(*http.Request) (*http.Response, error) {
				calls++
				response := containerMetadataResponse(307, "")
				response.Header.Set("Location", location)
				return response, nil
			})
			result, err := containers.New(client).CreateContainer(context.Background(), containerMetadataName)
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
				t.Fatalf("redirect result=%+v err=%v calls=%d", result, err, calls)
			}
		}

		calls := 0
		client := containerMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				response := containerMetadataResponse(307, "")
				response.Header.Set("Location", containerMetadataEndpoint)
				return response, nil
			}
			return containerMetadataResponse(201, "same target"), nil
		})
		result, err := containers.New(client).CreateContainer(context.Background(), containerMetadataName)
		if err != nil || result == nil || calls != 2 {
			t.Fatalf("same-target result=%+v err=%v calls=%d", result, err, calls)
		}
	})

	t.Run("live auth captured ordinary headers and advanced native header policy", func(t *testing.T) {
		original := map[string]string{"x-call": "source", "Content-Type": "source/type"}
		var client *gophercloud.ServiceClient
		calls := 0
		client = containerMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("X-Auth-Token") != "latest" || r.Header.Get("X-Call") != "option" || r.Header.Get("Content-Type") != "option/type" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Container-Meta-Book") != "metadata" {
				t.Errorf("headers=%v", r.Header)
			}
			return containerMetadataResponse(201, ""), nil
		})
		original["X-Source"] = "captured"
		client.MoreHeaders = original
		result, err := containers.New(client).CreateContainer(context.Background(), containerMetadataName,
			func(*containers.CreateContainerOpts) error {
				client.SetToken("latest")
				client.MoreHeaders["X-Source"] = "changed after capture"
				return nil
			},
			containers.WithCreateContainerHeaders(map[string]string{"X-Call": "option", "Content-Type": "option/type"}), containers.WithCreateContainerMetadata(map[string]string{"Book": "metadata"}))
		if err != nil || result == nil || calls != 1 || original["x-call"] != "source" || original["Content-Type"] != "source/type" || original["X-Source"] != "changed after capture" {
			t.Fatalf("result=%+v err=%v source=%v calls=%d", result, err, original, calls)
		}
		calls, hooks := 0, 0
		client = containerMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return containerMetadataResponse(503, "busy"), nil
			}
			if r.Header.Get("X-Container-Meta-Book") != "advanced" || r.Header.Get("X-Call") != "advanced" {
				t.Errorf("advanced=%v", r.Header)
			}
			return containerMetadataResponse(202, ""), nil
		})
		client.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
			hooks++
			opts.MoreHeaders = map[string]string{"X-Container-Meta-Book": "advanced", "X-Call": "advanced"}
			return nil
		}
		result, err = containers.New(client).CreateContainer(context.Background(), containerMetadataName)
		if err != nil || result == nil || calls != 2 || hooks != 1 || client.MoreHeaders != nil {
			t.Fatalf("advanced result=%+v err=%v calls=%d hooks=%d", result, err, calls, hooks)
		}
	})

	t.Run("bodyless retry response ownership guard no second mutation", func(t *testing.T) {
		for _, alter := range []struct {
			name   string
			change func(*gophercloud.RequestOpts)
		}{
			{"KeepResponseBody", func(o *gophercloud.RequestOpts) { o.KeepResponseBody = false }},
			{"JSONResponse", func(o *gophercloud.RequestOpts) { o.JSONResponse = &map[string]any{} }},
			{"RawBody", func(o *gophercloud.RequestOpts) { o.RawBody = strings.NewReader("injected") }},
			{"JSON null", func(o *gophercloud.RequestOpts) { o.JSONBody = json.RawMessage("null") }},
		} {
			calls, hooks := 0, 0
			body := &containerMetadataBody{Reader: strings.NewReader("busy")}
			cause := errors.New("callback cause")
			client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				containerLifecycleRequest(t, r, "DELETE")
				return containerMetadataWire(503, body), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				hooks++
				alter.change(opts)
				return cause
			}
			result, err := containers.New(client).DeleteContainer(context.Background(), containerMetadataName)
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, cause) || calls != 1 || hooks != 1 || body.closes.Load() != 1 {
				t.Fatalf("%s result=%+v err=%v calls=%d hooks=%d closes=%d", alter.name, result, err, calls, hooks, body.closes.Load())
			}
			containerLifecycleNativeError(t, err, "DELETE", 503, "busy")
		}
	})

	t.Run("native generated and resource collection compatibility", func(t *testing.T) {
		for _, status := range []int{201, 202, 204} {
			client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
				containerLifecycleRequest(t, r, "PUT")
				return containerMetadataResponse(status, ""), nil
			})
			if result, err := containers.New(client).Create(context.Background(), containerMetadataName, containers.CreateOpts{Metadata: map[string]string{"Book": "native"}}); err != nil || result == nil {
				t.Fatalf("nativeCreate%d=%+v err=%v", status, result, err)
			}
		}
		for _, status := range []int{202, 204} {
			client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
				containerLifecycleRequest(t, r, "DELETE")
				return containerMetadataResponse(status, ""), nil
			})
			if result, err := containers.New(client).Delete(context.Background(), containerMetadataName); err != nil || result == nil {
				t.Fatalf("nativeDelete%d=%+v err=%v", status, result, err)
			}
		}
		calls := 0
		client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method != "DELETE" || r.URL.String() != containerMetadataBase+"missing" {
				t.Errorf("resource request=%s %s", r.Method, r.URL)
			}
			return containerMetadataResponse(404, "missing"), nil
		})
		if err := containers.New(client).Resources.Delete(context.Background(), resource.ID("missing")); err != nil || calls != 1 {
			t.Fatalf("resourceDelete err=%v calls=%d", err, calls)
		}
	})
}
