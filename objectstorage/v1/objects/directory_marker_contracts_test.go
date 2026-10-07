package objects_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func directoryMarkerExternalPayload(t *testing.T, r *http.Request) {
	t.Helper()
	var body []byte
	var readErr, closeErr error
	if r.Body != nil {
		body, readErr = io.ReadAll(r.Body)
		closeErr = r.Body.Close()
	}
	if readErr != nil || closeErr != nil || len(body) != 0 || r.ContentLength != 0 || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 || r.Method != "PUT" || r.URL.RawQuery != "" || r.Header.Get("Content-Type") != "application/directory" || len(r.Header.Values("Content-Type")) != 1 || r.Header.Get("X-Detect-Content-Type") != "" {
		t.Errorf("marker=%s %s bytes=%x length=%d headers=%v read=%v close=%v", r.Method, r.URL, body, r.ContentLength, r.Header, readErr, closeErr)
	}
}
func directoryMarkerExternalResult(t *testing.T, result *objects.CreateObjectResult, status int, body string) {
	t.Helper()
	if result == nil || result.Source != "bytes" || result.Mode != "ordinary" || result.Size != 0 || result.MD5 != "" || result.SHA256 != "" || result.Capabilities != nil || result.Discovery != nil || len(result.Segments) != 0 || result.Manifest != nil || len(result.Cleanup) != 0 || result.Ordinary == nil || result.Ordinary.Acknowledgement == nil {
		t.Fatalf("marker result=%+v", result)
	}
	response := result.Ordinary.Acknowledgement
	if response.StatusCode != status || string(response.Body) != body {
		t.Fatalf("ack=%+v expected status=%d body=%q", response, status, body)
	}
}

func TestDirectoryMarkerContractsLiteralWireAndRawAcknowledgements(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := calls.Add(1)
		name := objectMetadataKey
		if call == 2 {
			name = "/leading//trailing/"
		}
		path := "/reverse/a%20b/v1/AUTH_marker/" + url.PathEscape(objectMetadataContainer) + "/" + url.PathEscape(name)
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) != 0 || r.Method != "PUT" || r.RequestURI != path || r.ContentLength != 0 || len(r.TransferEncoding) != 0 || r.Header.Get("Content-Type") != "application/directory" || len(r.Header.Values("Content-Type")) != 1 || r.Header.Get("X-Auth-Token") != "marker-token" || r.Header.Get("X-Object-Meta-Book") != "literal %2F" || r.Header.Get("X-Source") != "source" {
			t.Errorf("wire=%s %s bytes=%x length=%d headers=%v err=%v", r.Method, r.RequestURI, body, r.ContentLength, r.Header, err)
		}
		if r.Header.Get("X-Object-Meta-X-Sdk-Md5") != "" || r.Header.Get("X-Object-Meta-X-Sdk-Sha256") != "" || r.Header.Get("X-Object-Manifest") != "" || r.Header.Get("ETag") != "" {
			t.Errorf("unexpected generated headers=%v", r.Header)
		}
		w.Header().Set("X-Trans-Id", "wire-marker")
		w.WriteHeader(201 + int(call-1))
		_, _ = w.Write([]byte("marker acknowledgement"))
	}))
	defer server.Close()
	provider := &gophercloud.ProviderClient{HTTPClient: *server.Client()}
	provider.UseTokenLock()
	provider.SetToken("marker-token")
	client := &gophercloud.ServiceClient{ProviderClient: provider, Type: "object-store", Endpoint: server.URL + "/catalogue/v1/AUTH_other/", ResourceBase: server.URL + "/reverse/a%20b/v1/AUTH_marker/", MoreHeaders: map[string]string{"Content-Type": "source/type", "X-Source": "source"}}
	for index, name := range []string{objectMetadataKey, "/leading//trailing/"} {
		result, err := objects.New(client).CreateDirectoryMarkerObject(context.Background(), objectMetadataContainer, name, objects.WithDirectoryMarkerHeader("content-type", "caller/type"), objects.WithDirectoryMarkerMetadataValue("Book", "literal %2F"))
		directoryMarkerExternalResult(t, result, 201+index, "marker acknowledgement")
		if err != nil || len(result.Ordinary.Attempts) != 1 || result.Ordinary.Attempts[0].PhysicalAttempt != 1 || result.Ordinary.Acknowledgement.Header.Get("X-Trans-Id") != "wire-marker" {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("expected only two object PUTs; HTTP=%d", calls.Load())
	}
}

func TestDirectoryMarkerContractsPreflightAndSnapshots(t *testing.T) {
	t.Run("factory maps and every callback own state", func(t *testing.T) {
		headers, metadata := map[string]string{"X-Call": "captured", "Content-Type": "caller/type"}, map[string]string{"Book": "captured"}
		factory := objects.WithDirectoryMarkerOpts(objects.DirectoryMarkerOpts{Headers: headers, Metadata: metadata})
		headers["X-Call"], metadata["Book"] = "mutated factory input", "mutated factory input"
		calls, callbacks := 0, 0
		var retained *objects.DirectoryMarkerOpts
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			directoryMarkerExternalPayload(t, r)
			if r.Header.Get("X-Call") != "captured" || r.Header.Get("X-Object-Meta-Book") != "captured" || r.Header.Get("X-Source") != "captured" {
				t.Errorf("snapshots=%v", r.Header)
			}
			return objectMetadataResponse(201, "snapshot proof"), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "captured", "Content-Type": "source/type"}
		result, err := objects.New(client).CreateDirectoryMarkerObject(context.Background(), "c", "o", factory, func(cfg *objects.DirectoryMarkerOpts) error {
			callbacks++
			retained = cfg
			client.MoreHeaders["X-Source"] = "valid drift"
			client.MoreHeaders["Content-Type"] = "valid source drift"
			return nil
		}, func(*objects.DirectoryMarkerOpts) error {
			callbacks++
			retained.Headers["X-Call"] = "poisoned retained option"
			retained.Metadata["Book"] = "poisoned retained option"
			return nil
		})
		directoryMarkerExternalResult(t, result, 201, "snapshot proof")
		if err != nil || calls != 1 || callbacks != 2 {
			t.Fatalf("result=%+v err=%v HTTP=%d callbacks=%d", result, err, calls, callbacks)
		}
		result.Ordinary.Acknowledgement.Header.Set("X-Trans-Id", "changed acknowledgement")
		result.Ordinary.Acknowledgement.Body[0] = 'X'
		if result.Ordinary.Attempts[0].Response.Header.Get("X-Trans-Id") != "actual-object" || string(result.Ordinary.Attempts[0].Response.Body) != "snapshot proof" {
			t.Fatalf("shared phase observations=%+v", result.Ordinary)
		}
	})
	t.Run("reserved or invalid input never reaches transport", func(t *testing.T) {
		for _, option := range []objects.DirectoryMarkerOption{
			objects.WithDirectoryMarkerHeader("X-Detect-Content-Type", "false"),
			objects.WithDirectoryMarkerHeader("Content-Length", "1"),
			objects.WithDirectoryMarkerHeader("Content-Type", "bad\x7f"),
			objects.WithDirectoryMarkerHeaders(map[string]string{"Content-Type": "a", "content-type": "a"}),
			nil,
		} {
			calls := 0
			client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
				calls++
				return objectMetadataResponse(201, "unexpected"), nil
			})
			result, err := objects.New(client).CreateDirectoryMarkerObject(context.Background(), "c", "o", option)
			if !errors.Is(err, resource.ErrInvalidOption) || result != nil || calls != 0 {
				t.Fatalf("result=%+v err=%v HTTP=%d", result, err, calls)
			}
		}
	})
	t.Run("original source rejects before user callbacks", func(t *testing.T) {
		for _, source := range []map[string]string{{"X-Detect-Content-Type": "false"}, {"Content-Type": "a", "content-type": "a"}, {"Content-Type": "bad\x7f"}} {
			calls, callbacks := 0, 0
			client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
				calls++
				return objectMetadataResponse(201, "unexpected"), nil
			})
			client.MoreHeaders = source
			result, err := objects.New(client).CreateDirectoryMarkerObject(context.Background(), "c", "o", func(*objects.DirectoryMarkerOpts) error { callbacks++; return nil })
			if !errors.Is(err, resource.ErrInvalidOption) || result != nil || calls != 0 || callbacks != 0 {
				t.Fatalf("source=%v result=%+v err=%v HTTP=%d callback=%d", source, result, err, calls, callbacks)
			}
		}
	})
	t.Run("callback and source causes join before PUT", func(t *testing.T) {
		cause := errors.New("marker option failed")
		calls := 0
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			return objectMetadataResponse(201, "unexpected"), nil
		})
		result, err := objects.New(client).CreateDirectoryMarkerObject(context.Background(), "c", "o", func(*objects.DirectoryMarkerOpts) error {
			client.Endpoint = "https://changed.invalid/v1/AUTH_other/"
			return cause
		})
		if !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || result != nil || calls != 0 {
			t.Fatalf("result=%+v err=%v HTTP=%d", result, err, calls)
		}
	})
}

func TestDirectoryMarkerContractsNativeReplayAndPhysicalEvidence(t *testing.T) {
	for _, status := range []int{401, 429, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls, hooks, callbacks := 0, 0, 0
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				directoryMarkerExternalPayload(t, r)
				token := "original-token"
				if calls == 2 {
					token = "refreshed"
				}
				if r.Header.Get("X-Auth-Token") != token || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Object-Meta-Book") != "owned" {
					t.Errorf("attempt=%d headers=%v", calls, r.Header)
				}
				if calls == 1 {
					return objectMetadataResponse(status, "native rejected proof"), nil
				}
				return objectMetadataResponse(202, "native marker acknowledgement"), nil
			})
			client.MoreHeaders = map[string]string{"X-Source": "captured", "Content-Type": "source/type"}
			if status == 401 {
				client.ReauthFunc = func(context.Context) error { hooks++; client.SetToken("refreshed"); return nil }
			} else if status == 429 {
				client.RetryBackoffFunc = func(_ context.Context, proof *gophercloud.ErrUnexpectedResponseCode, _ error, _ uint) error {
					hooks++
					client.SetToken("refreshed")
					proof.Body[0] = 'X'
					proof.ResponseHeader.Set("X-Trans-Id", "mutated callback")
					return nil
				}
			} else {
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					var proof gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &proof) || proof.Actual != 503 {
						return err
					}
					hooks++
					client.SetToken("refreshed")
					proof.Body[0] = 'X'
					proof.ResponseHeader.Set("X-Trans-Id", "mutated callback")
					return nil
				}
			}
			result, err := objects.New(client).CreateDirectoryMarkerObject(context.Background(), "c", "o", objects.WithDirectoryMarkerMetadataValue("Book", "owned"), func(*objects.DirectoryMarkerOpts) error {
				callbacks++
				client.MoreHeaders["X-Source"] = "valid drift"
				return nil
			})
			directoryMarkerExternalResult(t, result, 202, "native marker acknowledgement")
			if err != nil || calls != 2 || hooks != 1 || callbacks != 1 || len(result.Ordinary.Attempts) != 2 || result.Ordinary.Attempts[0].Response.StatusCode != status || string(result.Ordinary.Attempts[0].Response.Body) != "native rejected proof" || result.Ordinary.Attempts[0].Response.Header.Get("X-Trans-Id") != "actual-object" || result.Ordinary.Attempts[0].Error == nil || result.Ordinary.Attempts[1].PhysicalAttempt != 2 {
				t.Fatalf("result=%+v err=%v HTTP=%d hooks=%d callback=%d", result, err, calls, hooks, callbacks)
			}
		})
	}
	t.Run("307 follows fixed literal target with exact empty marker", func(t *testing.T) {
		calls, redirects := 0, 0
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			directoryMarkerExternalPayload(t, r)
			if r.URL.String() != objectMetadataEndpoint {
				t.Errorf("target=%s", r.URL)
			}
			if calls == 1 {
				response := objectMetadataResponse(307, "redirect proof")
				response.Header.Set("Location", objectMetadataEndpoint)
				return response, nil
			}
			return objectMetadataResponse(201, "after redirect"), nil
		})
		client.HTTPClient.CheckRedirect = func(r *http.Request, via []*http.Request) error {
			redirects++
			if r.Method != "PUT" || r.Header.Get("Content-Type") != "application/directory" || len(via) != 1 {
				t.Errorf("redirect=%s headers=%v via=%d", r.Method, r.Header, len(via))
			}
			return nil
		}
		result, err := objects.New(client).CreateDirectoryMarkerObject(context.Background(), objectMetadataContainer, objectMetadataKey)
		directoryMarkerExternalResult(t, result, 201, "after redirect")
		if err != nil || calls != 2 || redirects != 1 || len(result.Ordinary.Attempts) != 2 || result.Ordinary.Attempts[0].Response.StatusCode != 307 || string(result.Ordinary.Attempts[0].Response.Body) != "redirect proof" {
			t.Fatalf("result=%+v err=%v HTTP=%d redirects=%d", result, err, calls, redirects)
		}
	})
}

func TestDirectoryMarkerContractsNativeMediaAndDetectGuards(t *testing.T) {
	t.Run("retry cannot remove change alias or detect marker media", func(t *testing.T) {
		for _, mutation := range []string{"remove", "change", "alias", "detect", "body", "codes"} {
			calls := 0
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				directoryMarkerExternalPayload(t, r)
				return objectMetadataResponse(503, "before retry guard"), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, cfg *gophercloud.RequestOpts, _ error, _ uint) error {
				if cfg.MoreHeaders == nil {
					cfg.MoreHeaders = map[string]string{}
				}
				switch mutation {
				case "remove":
					delete(cfg.MoreHeaders, "Content-Type")
				case "change":
					cfg.MoreHeaders["Content-Type"] = "changed/type"
				case "alias":
					cfg.MoreHeaders["content-type"] = "application/directory"
				case "detect":
					cfg.MoreHeaders["X-Detect-Content-Type"] = "false"
				case "body":
					cfg.RawBody = strings.NewReader("injected")
				case "codes":
					cfg.OkCodes = append(cfg.OkCodes, 503)
				}
				return nil
			}
			result, err := objects.New(client).CreateDirectoryMarkerObject(context.Background(), "c", "o")
			if !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || result == nil || result.Ordinary == nil || result.Ordinary.Acknowledgement != nil || len(result.Ordinary.Attempts) != 1 || result.Ordinary.Attempts[0].Response.StatusCode != 503 || string(result.Ordinary.Attempts[0].Response.Body) != "before retry guard" {
				t.Fatalf("mutation=%s result=%+v err=%v HTTP=%d", mutation, result, err, calls)
			}
		}
	})
	t.Run("raw redirect media must stay one owned value", func(t *testing.T) {
		for _, mutation := range []string{"remove", "change", "alias", "multiple", "detect"} {
			calls, redirects := 0, 0
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				directoryMarkerExternalPayload(t, r)
				response := objectMetadataResponse(307, "before redirect guard")
				response.Header.Set("Location", objectMetadataEndpoint)
				return response, nil
			})
			client.HTTPClient.CheckRedirect = func(r *http.Request, _ []*http.Request) error {
				redirects++
				switch mutation {
				case "remove":
					r.Header.Del("Content-Type")
				case "change":
					r.Header.Set("Content-Type", "changed/type")
				case "alias":
					r.Header["content-type"] = []string{"application/directory"}
				case "multiple":
					r.Header["Content-Type"] = []string{"application/directory", "application/directory"}
				case "detect":
					r.Header.Set("X-Detect-Content-Type", "false")
				}
				return nil
			}
			result, err := objects.New(client).CreateDirectoryMarkerObject(context.Background(), objectMetadataContainer, objectMetadataKey)
			if !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || redirects != 1 || result == nil || result.Ordinary == nil || result.Ordinary.Acknowledgement != nil || len(result.Ordinary.Attempts) != 1 || result.Ordinary.Attempts[0].Response.StatusCode != 307 {
				t.Fatalf("mutation=%s result=%+v err=%v HTTP=%d redirects=%d", mutation, result, err, calls, redirects)
			}
		}
	})
}

func TestDirectoryMarkerContractsAcceptedFaultsAndSourceGuards(t *testing.T) {
	for _, status := range []int{201, 202} {
		for _, fault := range []string{"read", "close"} {
			t.Run(fmt.Sprintf("%d/%s", status, fault), func(t *testing.T) {
				sentinel := errors.New("marker response " + fault)
				proof := string([]byte{'a', 0, 0xff, 'z'})
				body := &objectMetadataBody{Reader: strings.NewReader(proof)}
				if fault == "read" {
					body.Reader = objectMetadataReader(func(p []byte) (int, error) { return copy(p, proof), sentinel })
				} else {
					body.closeErr = sentinel
				}
				calls, retries := 0, 0
				client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
					calls++
					directoryMarkerExternalPayload(t, r)
					return objectMetadataWire(status, body), nil
				})
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				result, err := objects.New(client).CreateDirectoryMarkerObject(context.Background(), "c", "o")
				directoryMarkerExternalResult(t, result, status, proof)
				if !errors.Is(err, sentinel) || calls != 1 || retries != 0 || len(result.Ordinary.Attempts) != 1 || body.closes.Load() != 1 {
					t.Fatalf("result=%+v err=%v HTTP=%d retries=%d closes=%d", result, err, calls, retries, body.closes.Load())
				}
				objectMetadataProof(t, err, status, proof)
			})
		}
	}
	t.Run("transient detect source fault remains observed after Close restores it", func(t *testing.T) {
		calls := 0
		var client *gophercloud.ServiceClient
		body := &objectMetadataBody{Reader: strings.NewReader("accepted source proof"), onClose: func() { delete(client.MoreHeaders, "X-Detect-Content-Type") }}
		client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			directoryMarkerExternalPayload(t, r)
			client.MoreHeaders["X-Detect-Content-Type"] = "false"
			return objectMetadataWire(201, body), nil
		})
		client.MoreHeaders = map[string]string{}
		result, err := objects.New(client).CreateDirectoryMarkerObject(context.Background(), "c", "o")
		directoryMarkerExternalResult(t, result, 201, "accepted source proof")
		if !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || body.closes.Load() != 1 {
			t.Fatalf("result=%+v err=%v HTTP=%d closes=%d", result, err, calls, body.closes.Load())
		}
		objectMetadataProof(t, err, 201, "accepted source proof")
	})
	t.Run("physical response and transport error never fabricate acknowledgement", func(t *testing.T) {
		sentinel := errors.New("physical marker transport failure")
		body := &objectMetadataBody{Reader: strings.NewReader("not consumed")}
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			directoryMarkerExternalPayload(t, r)
			return objectMetadataWire(201, body), sentinel
		})
		result, err := objects.New(client).CreateDirectoryMarkerObject(context.Background(), "c", "o")
		if !errors.Is(err, sentinel) || result == nil || result.Ordinary == nil || result.Ordinary.Acknowledgement != nil || len(result.Ordinary.Attempts) != 1 || result.Ordinary.Attempts[0].Response == nil || result.Ordinary.Attempts[0].Response.StatusCode != 201 || len(result.Ordinary.Attempts[0].Response.Body) != 0 || body.closes.Load() != 1 || len(result.Cleanup) != 0 {
			t.Fatalf("result=%+v err=%v closes=%d", result, err, body.closes.Load())
		}
	})
}
