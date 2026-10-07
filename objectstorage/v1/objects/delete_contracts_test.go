package objects_test

import (
	"context"
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

	"github.com/JSYoo5B/gophercloudsdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const deleteContractBulkOK = `{"Response Status":"200 OK","Response Body":"","Number Deleted":2,"Number Not Found":1,"Errors":[],"plugin":{"precise":9007199254740993},"links":42,"created_at":false}`

func deleteContractWire(status int, body string, flag ...string) *http.Response {
	r := objectMetadataResponse(status, body)
	if flag != nil {
		r.Header["X-Static-Large-Object"] = flag
	}
	return r
}

func deleteContractNoPhase(t *testing.T, result *objects.DeleteObjectResult) {
	t.Helper()
	if result != nil && (result.Discovery != nil || result.Deletion != nil || result.Bulk != nil || result.IgnoredMissing) {
		t.Fatalf("fabricated accepted phase: %+v", result)
	}
}

func TestObjectDeleteContractsRoutesAndSelection(t *testing.T) {
	t.Run("literal reverse prefix and version reach HEAD and SLO DELETE", func(t *testing.T) {
		var calls atomic.Int32
		const version = "opaque/ +%?#한"
		path := "/reverse/a%2Fb/v1/A/" + url.PathEscape(objectMetadataContainer) + "/" + url.PathEscape(objectMetadataKey)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			query := url.Values{"version-id": {version}}
			if r.Method == http.MethodDelete {
				query.Set("multipart-manifest", "delete")
				if r.Header.Get("Accept") != "application/json" {
					t.Errorf("SLO Accept=%q", r.Header.Get("Accept"))
				}
			}
			if r.URL.EscapedPath() != path || r.URL.RawQuery != query.Encode() || r.Header.Get("X-Newest") != "false" || r.Header.Get("X-Call") != "owned" {
				t.Errorf("wire=%s %s headers=%v body=%v", r.Method, r.RequestURI, r.Header, r.Body)
			}
			if body, err := io.ReadAll(r.Body); err != nil || len(body) != 0 {
				t.Errorf("request body=%q err=%v", body, err)
			}
			w.Header().Set("X-Trans-Id", "wire-delete")
			if r.Method == http.MethodHead {
				w.Header().Set("X-Static-Large-Object", "true")
				w.WriteHeader(200)
				return
			}
			if r.Method != http.MethodDelete {
				t.Errorf("unexpected method=%s", r.Method)
			}
			_, _ = io.WriteString(w, deleteContractBulkOK)
		}))
		defer server.Close()
		client := &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{HTTPClient: *server.Client()}, Type: "object-store", Endpoint: server.URL + "/catalog/v1/A/", ResourceBase: server.URL + "/reverse/a%2Fb/v1/A/"}
		result, err := objects.New(client).DeleteObject(context.Background(), objectMetadataContainer, objectMetadataKey, objects.WithDeleteObjectVersionID(version), objects.WithDeleteObjectNewest(false), objects.WithDeleteObjectHeader("X-Call", "owned"))
		if err != nil || result == nil || result.Discovery == nil || result.Deletion == nil || result.Discovery.StatusCode != 200 || result.Deletion.StatusCode != 200 || result.StaticLargeObject == nil || !*result.StaticLargeObject || result.Bulk == nil || calls.Load() != 2 {
			t.Fatalf("result=%+v err=%v calls=%d", result, err, calls.Load())
		}
	})
	t.Run("ordinary native acknowledgements and known flags", func(t *testing.T) {
		for _, headCode := range []int{200, 204} {
			for _, deleteCode := range []int{202, 204} {
				calls := 0
				client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Body != nil || r.URL.RawQuery != "" {
						t.Errorf("request=%s body=%v", r.URL, r.Body)
					}
					if calls == 1 {
						if r.Method != http.MethodHead {
							t.Errorf("first method=%s", r.Method)
						}
						return deleteContractWire(headCode, "discovery", "false"), nil
					}
					if r.Method != http.MethodDelete {
						t.Errorf("second method=%s", r.Method)
					}
					return deleteContractWire(deleteCode, "opaque \xff acknowledgement"), nil
				})
				result, err := objects.New(client).DeleteObject(context.Background(), "c", "o")
				if err != nil || result == nil || result.Discovery == nil || result.Discovery.StatusCode != headCode || result.Deletion == nil || result.Deletion.StatusCode != deleteCode || string(result.Deletion.Body) != "opaque \xff acknowledgement" || result.StaticLargeObject == nil || *result.StaticLargeObject || result.Bulk != nil || calls != 2 {
					t.Fatalf("head=%d delete=%d result=%+v err=%v calls=%d", headCode, deleteCode, result, err, calls)
				}
			}
		}
		for _, known := range []bool{false, true} {
			calls := 0
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != http.MethodDelete || (r.URL.Query().Get("multipart-manifest") == "delete") != known {
					t.Errorf("known=%v wire=%s %s", known, r.Method, r.URL)
				}
				return deleteContractWire(204, "ack"), nil
			})
			result, err := objects.New(client).DeleteObject(context.Background(), "c", "o", objects.WithDeleteObjectStaticLargeObject(known))
			if err != nil || result == nil || result.Discovery != nil || result.StaticLargeObject == nil || *result.StaticLargeObject != known || result.Deletion == nil || result.Bulk != nil || calls != 1 {
				t.Fatalf("known=%v result=%+v err=%v calls=%d", known, result, err, calls)
			}
		}
	})
	t.Run("observed header boolean is strict and missing means ordinary", func(t *testing.T) {
		for _, tc := range []struct {
			values []string
			large  bool
		}{{nil, false}, {[]string{}, false}, {[]string{"0"}, false}, {[]string{" \tfalse\t "}, false}, {[]string{"TRUE"}, true}, {[]string{"1"}, true}} {
			calls := 0
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return deleteContractWire(200, "", tc.values...), nil
				}
				if (r.URL.Query().Get("multipart-manifest") == "delete") != tc.large {
					t.Errorf("values=%q query=%s", tc.values, r.URL.RawQuery)
				}
				return deleteContractWire(204, ""), nil
			})
			result, err := objects.New(client).DeleteObject(context.Background(), "c", "o")
			if err != nil || result == nil || result.StaticLargeObject == nil || *result.StaticLargeObject != tc.large || calls != 2 {
				t.Fatalf("values=%q result=%+v err=%v calls=%d", tc.values, result, err, calls)
			}
		}
		for _, h := range []http.Header{{"X-Static-Large-Object": {""}}, {"X-Static-Large-Object": {"yes"}}, {"X-Static-Large-Object": {"true", "false"}}, {"X-Static-Large-Object": {"true"}, "x-static-large-object": {"true"}}} {
			calls := 0
			client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
				calls++
				r := deleteContractWire(200, "bad flag")
				for k, v := range h {
					r.Header[k] = v
				}
				return r, nil
			})
			result, err := objects.New(client).DeleteObject(context.Background(), "c", "o")
			if result == nil || result.Discovery == nil || result.Deletion != nil || result.StaticLargeObject != nil || calls != 1 {
				t.Fatalf("headers=%v result=%+v err=%v calls=%d", h, result, err, calls)
			}
			objectMetadataProof(t, err, 200, "bad flag")
		}
	})
}

func TestObjectDeleteContractsBulkOutcomes(t *testing.T) {
	t.Run("NotFound counts remain informative under strict ignore missing", func(t *testing.T) {
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
			return deleteContractWire(200, " \r\n\r\n"+deleteContractBulkOK), nil
		})
		result, err := objects.New(client).DeleteObject(context.Background(), "c", "o", objects.WithDeleteObjectStaticLargeObject(true), objects.WithDeleteObjectIgnoreMissing(false))
		if err != nil || result == nil || result.Bulk == nil || result.Bulk.ResponseCode != 200 || result.Bulk.NumberDeleted != 2 || result.Bulk.NumberNotFound != 1 || result.IgnoredMissing || result.Bulk.CreatedAt != nil || result.Bulk.Links != nil || string(result.Bulk.Body["plugin"]) != `{"precise":9007199254740993}` {
			t.Fatalf("result=%+v err=%v", result, err)
		}
		result.Bulk.Body["Response Status"][1] = '!'
		result.Bulk.Header.Set("X-Trans-Id", "changed")
		if string(result.Deletion.Body) != " \r\n\r\n"+deleteContractBulkOK || result.Deletion.Header.Get("X-Trans-Id") != "actual-object" {
			t.Fatal("bulk aliases raw deletion proof")
		}
	})
	t.Run("embedded failures retain outer200 and independently owned error data", func(t *testing.T) {
		for _, body := range []string{
			`{"Response Status":"404 Not Found","Response Body":"manifest lookup","Number Deleted":0,"Number Not Found":0,"Errors":[]}`,
			`{"Response Status":"200 OK","Response Body":"","Number Deleted":1,"Number Not Found":0,"Errors":[["/c/a%2Fb","arbitrary retrieval message"]]}`,
			`{"Response Status":"502 Bad Gateway","Response Body":"worker failed","Number Deleted":1,"Number Not Found":0,"Errors":[["/c/a%2Fb","500 Internal Server Error"]]}`,
		} {
			calls := 0
			client := objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return deleteContractWire(200, body), nil })
			result, err := objects.New(client).DeleteObject(context.Background(), "c", "o", objects.WithDeleteObjectStaticLargeObject(true))
			var embedded *objects.ObjectDeleteBulkError
			var physical gophercloud.ErrUnexpectedResponseCode
			if result == nil || result.Bulk == nil || result.IgnoredMissing || !errors.As(err, &embedded) || embedded.ResponseCode != result.Bulk.ResponseCode || errors.As(err, &physical) || calls != 1 {
				t.Fatalf("result=%+v err=%v embedded=%+v calls=%d", result, err, embedded, calls)
			}
			proof := objectMetadataProof(t, err, 200, body)
			if len(result.Bulk.Errors) != 0 {
				if result.Bulk.Errors[0].Name != "/c/a%2Fb" {
					t.Fatal("server path was decoded", result.Bulk.Errors)
				}
				original := embedded.Errors[0]
				result.Bulk.Errors[0].Name, result.Bulk.Errors[0].Error = "changed", "changed"
				if embedded.Errors[0] != original {
					t.Fatal("bulk result aliases embedded error")
				}
			}
			result.Deletion.Body[0] = '!'
			result.Deletion.Header.Set("X-Trans-Id", "changed")
			if string(proof.Body) != body || proof.Header.Get("X-Trans-Id") != "actual-object" {
				t.Fatal("phase aliases response error")
			}
		}
	})
	t.Run("malformed accepted reports fail with raw proof and asynchronous acknowledgements stay raw", func(t *testing.T) {
		for _, body := range []string{`null`, `{"Response Status":"200 OK"}`, `{"Response Status":"200 OK","Response Body":"","Number Deleted":1.5,"Number Not Found":0,"Errors":[]}`, `{"Response Status":"200 OK","Response Body":"","Number Deleted":1,"Number Not Found":0,"Errors":[["one"]]}`, `{"Response Status":"200 O\u0085K","Response Body":"","Number Deleted":1,"Number Not Found":0,"Errors":[]}`, "{\"bad\":\"\xff\"}"} {
			client := objectMetadataClient(func(*http.Request) (*http.Response, error) { return deleteContractWire(200, body), nil })
			result, err := objects.New(client).DeleteObject(context.Background(), "c", "o", objects.WithDeleteObjectStaticLargeObject(true))
			if result == nil || result.Deletion == nil || result.Bulk != nil || result.IgnoredMissing {
				t.Fatalf("body=%q result=%+v err=%v", body, result, err)
			}
			objectMetadataProof(t, err, 200, body)
		}
		for _, status := range []int{202, 204} {
			client := objectMetadataClient(func(*http.Request) (*http.Response, error) { return deleteContractWire(status, "opaque \xff ack"), nil })
			result, err := objects.New(client).DeleteObject(context.Background(), "c", "o", objects.WithDeleteObjectStaticLargeObject(true))
			if err != nil || result == nil || result.Bulk != nil || result.Deletion == nil || result.Deletion.StatusCode != status || string(result.Deletion.Body) != "opaque \xff ack" {
				t.Fatalf("status=%d result=%+v err=%v", status, result, err)
			}
		}
	})
}

func TestObjectDeleteContractsOptionsAndSnapshots(t *testing.T) {
	t.Run("callback once and retained configuration cannot change later phase", func(t *testing.T) {
		headers := map[string]string{"x-call": "factory"}
		newest := false
		full := objects.WithDeleteObjectOpts(objects.DeleteObjectOpts{Headers: headers, Newest: &newest, VersionID: "owned version"})
		headers["x-call"], newest = "outside", true
		calls, callbacks := 0, 0
		var retained *objects.DeleteObjectOpts
		var client *gophercloud.ServiceClient
		client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("X-Call") != "final" || r.Header.Get("X-Discarded") != "" || r.Header.Get("X-Newest") != "false" || r.Header.Get("X-Source") != "captured" || r.URL.Query().Get("version-id") != "owned version" {
				t.Errorf("phase%d %s headers=%v", calls, r.URL, r.Header)
			}
			if calls == 1 {
				body := &objectMetadataBody{Reader: strings.NewReader("head"), onClose: func() {
					retained.Headers["X-Call"] = "late"
					*retained.Newest = true
					retained.VersionID = "late"
					client.MoreHeaders["X-Source"] = "valid later"
					client.SetToken("fresh-token")
				}}
				return objectMetadataWire(200, body), nil
			}
			if r.Header.Get("X-Auth-Token") != "fresh-token" {
				t.Errorf("live token=%v", r.Header)
			}
			return deleteContractWire(204, "ack"), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		result, err := objects.New(client).DeleteObject(context.Background(), "c", "o", objects.WithDeleteObjectStaticLargeObject(true), objects.WithDeleteObjectHeader("X-Discarded", "yes"), full, objects.WithDeleteObjectHeader("X-Call", "final"), func(cfg *objects.DeleteObjectOpts) error { callbacks++; retained = cfg; return nil })
		if err != nil || result == nil || result.Discovery == nil || result.StaticLargeObject == nil || *result.StaticLargeObject || calls != 2 || callbacks != 1 {
			t.Fatalf("result=%+v err=%v calls=%d callbacks=%d", result, err, calls, callbacks)
		}
	})
	t.Run("factory boolean pointers and helpers support concurrent reuse and reset", func(t *testing.T) {
		var calls atomic.Int32
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Method != http.MethodDelete || r.URL.RawQuery != "" || r.Header.Get("X-Call") != "stable" || len(r.Header.Values("X-Newest")) != 0 {
				t.Errorf("request=%s %s headers=%v", r.Method, r.URL, r.Header)
			}
			return deleteContractWire(202, "ack"), nil
		})
		known, ignore, newest := false, false, true
		option := objects.WithDeleteObjectOpts(objects.DeleteObjectOpts{Headers: map[string]string{"X-Call": "stable"}, StaticLargeObject: &known, IgnoreMissing: &ignore, Newest: &newest})
		known, ignore, newest = true, true, false
		api := objects.New(client)
		var workers sync.WaitGroup
		for range 4 {
			workers.Add(1)
			go func() {
				defer workers.Done()
				result, err := api.DeleteObject(context.Background(), "c", "o", option, objects.WithoutDeleteObjectNewest())
				if err != nil || result == nil || result.Discovery != nil || result.StaticLargeObject == nil || *result.StaticLargeObject {
					t.Errorf("result=%+v err=%v", result, err)
				}
			}()
		}
		workers.Wait()
		if calls.Load() != 4 {
			t.Fatal("requests", calls.Load())
		}
		for _, headers := range []map[string]string{{"X-Static-Large-Object": "true"}, {"Cookie": "auth"}, {"X-Service-Token": "auth"}, {"X-Call": "one", "x-call": "two"}} {
			if result, err := api.DeleteObject(context.Background(), "c", "o", objects.WithDeleteObjectHeaders(headers)); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("headers=%v result=%+v err=%v", headers, result, err)
			}
		}
		if calls.Load() != 4 {
			t.Fatal("invalid options sent HTTP", calls.Load())
		}
	})
}

func TestObjectDeleteContractsMissingEvidence(t *testing.T) {
	t.Run("physical missing is phase-specific and strict policy keeps native errors", func(t *testing.T) {
		for _, phase := range []string{"head", "delete"} {
			for _, strict := range []bool{false, true} {
				calls := 0
				client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
					calls++
					if phase == "delete" && r.Method == http.MethodHead {
						return deleteContractWire(200, "head"), nil
					}
					return deleteContractWire(404, "missing"), nil
				})
				options := []objects.DeleteObjectOption{}
				if strict {
					options = append(options, objects.WithDeleteObjectIgnoreMissing(false))
				}
				result, err := objects.New(client).DeleteObject(context.Background(), "c", "o", options...)
				if strict {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "missing" {
						t.Fatalf("phase=%s result=%+v err=%v native=%+v", phase, result, err, native)
					}
					if phase == "head" {
						deleteContractNoPhase(t, result)
					} else if result == nil || result.Discovery == nil || result.Deletion != nil || result.IgnoredMissing {
						t.Fatalf("strict later result=%+v", result)
					}
				} else {
					if err != nil || result == nil || !result.IgnoredMissing || result.Bulk != nil {
						t.Fatalf("phase=%s result=%+v err=%v", phase, result, err)
					}
					if phase == "head" {
						if result.Discovery == nil || result.Discovery.StatusCode != 404 || result.StaticLargeObject != nil || result.Deletion != nil {
							t.Fatal(result)
						}
					} else if result.Discovery == nil || result.Deletion == nil || result.Deletion.StatusCode != 404 || result.StaticLargeObject == nil || *result.StaticLargeObject {
						t.Fatal(result)
					}
				}
				want := 1
				if phase == "delete" {
					want = 2
				}
				if calls != want {
					t.Fatalf("phase=%s requests=%d", phase, calls)
				}
			}
		}
	})
	t.Run("dirty physical404 never suppresses read Close cancellation or source faults", func(t *testing.T) {
		for _, phase := range []string{"head", "delete"} {
			ctx, cancel := context.WithCancelCause(context.Background())
			readErr, closeErr, cause := errors.New("read failed"), errors.New("close failed"), errors.New("caller cause")
			calls, retries := 0, 0
			var client *gophercloud.ServiceClient
			body := &objectMetadataBody{Reader: objectMetadataReader(func(p []byte) (int, error) { return copy(p, "partial missing"), errors.Join(io.EOF, readErr) }), closeErr: closeErr, onClose: func() { client.ResourceBase += "changed"; cancel(cause) }}
			client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if phase == "delete" && r.Method == http.MethodHead {
					return deleteContractWire(200, "head"), nil
				}
				return objectMetadataWire(404, body), nil
			})
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			result, err := objects.New(client).DeleteObject(ctx, "c", "o")
			cancel(nil)
			if result == nil || result.IgnoredMissing || result.Bulk != nil || retries != 0 || body.closes.Load() != 1 {
				t.Fatalf("phase=%s result=%+v err=%v retries=%d", phase, result, err, retries)
			}
			for _, expected := range []error{io.EOF, readErr, closeErr, context.Canceled, cause, resource.ErrInvalidOption} {
				if !errors.Is(err, expected) {
					t.Errorf("phase=%s missing=%v err=%v", phase, expected, err)
				}
			}
			proof := objectMetadataProof(t, err, 404, "partial missing")
			actual := result.Discovery
			if phase == "delete" {
				actual = result.Deletion
				if result.Discovery == nil {
					t.Fatal("earlier phase lost")
				}
			}
			if actual == nil || string(actual.Body) != "partial missing" {
				t.Fatalf("phase=%s result=%+v", phase, result)
			}
			actual.Body[0] = '!'
			actual.Header.Set("X-Trans-Id", "changed")
			if string(proof.Body) != "partial missing" || proof.Header.Get("X-Trans-Id") != "actual-object" {
				t.Fatal("phase aliases proof")
			}
			want := 1
			if phase == "delete" {
				want = 2
			}
			if calls != want {
				t.Fatal("accepted body replayed", calls)
			}
		}
	})
}

func TestObjectDeleteContractsPhaseErrorsAndGuards(t *testing.T) {
	t.Run("accepted transport drift remains sticky when Close restores source", func(t *testing.T) {
		calls := 0
		var client *gophercloud.ServiceClient
		body := &objectMetadataBody{Reader: strings.NewReader("head proof"), onClose: func() { client.Endpoint = objectMetadataBase }}
		client = objectMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			client.Endpoint = "https://foreign.invalid/changed/"
			return objectMetadataWire(200, body), nil
		})
		result, err := objects.New(client).DeleteObject(context.Background(), "c", "o")
		if result == nil || result.Discovery == nil || result.Deletion != nil || result.StaticLargeObject != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || body.closes.Load() != 1 {
			t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
		}
		objectMetadataProof(t, err, 200, "head proof")
	})
	t.Run("later transport or rejected status keeps completed discovery", func(t *testing.T) {
		for _, transport := range []bool{true, false} {
			cause := &gophercloud.ErrUnexpectedResponseCode{Actual: 404, Body: []byte("nested404")}
			calls := 0
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method == http.MethodHead {
					return deleteContractWire(200, "head"), nil
				}
				if transport {
					return nil, cause
				}
				return deleteContractWire(503, "actual503"), nil
			})
			result, err := objects.New(client).DeleteObject(context.Background(), "c", "o")
			if result == nil || result.Discovery == nil || string(result.Discovery.Body) != "head" || result.Deletion != nil || result.IgnoredMissing || calls != 2 {
				t.Fatalf("transport=%v result=%+v err=%v calls=%d", transport, result, err, calls)
			}
			if transport {
				if !errors.Is(err, cause) {
					t.Fatal("transport cause lost", err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 503 {
					t.Fatal("physical rejected proof lost", err)
				}
			}
		}
	})
	t.Run("callback and retry source changes stop before another attempt", func(t *testing.T) {
		for _, duringCallback := range []bool{true, false} {
			calls, later := 0, 0
			client := objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return deleteContractWire(503, "actual503"), nil })
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				client.ResourceBase = "changed"
				return nil
			}
			api := objects.New(client)
			options := []objects.DeleteObjectOption{}
			if duringCallback {
				options = append(options, func(*objects.DeleteObjectOpts) error {
					client.ProviderClient = &gophercloud.ProviderClient{}
					return nil
				}, func(*objects.DeleteObjectOpts) error { later++; return nil })
			}
			result, err := api.DeleteObject(context.Background(), "c", "o", options...)
			deleteContractNoPhase(t, result)
			if !errors.Is(err, resource.ErrInvalidOption) || later != 0 {
				t.Fatalf("callback=%v result=%+v err=%v later=%d", duringCallback, result, err, later)
			}
			want := 1
			if duringCallback {
				want = 0
			}
			if calls != want {
				t.Fatal("unsafe attempt", calls)
			}
			if !duringCallback {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 503 {
					t.Fatal("retry original cause lost", err)
				}
			}
		}
	})
}

func TestObjectDeleteContractsNativePolicy(t *testing.T) {
	t.Run("same-target redirect cannot retain a raw Accept alias", func(t *testing.T) {
		calls, redirects := 0, 0
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			values := 0
			for name, entries := range r.Header {
				if strings.EqualFold(name, "Accept") {
					values += len(entries)
					for _, value := range entries {
						if value != "application/json" {
							t.Errorf("SLO Accept alias=%q headers=%v", value, r.Header)
						}
					}
				}
			}
			if values != 1 {
				t.Errorf("SLO Accept values=%d headers=%v", values, r.Header)
			}
			if calls == 1 {
				response := deleteContractWire(307, "redirect")
				response.Header.Set("Location", r.URL.String())
				return response, nil
			}
			return deleteContractWire(200, deleteContractBulkOK), nil
		})
		client.HTTPClient.CheckRedirect = func(next *http.Request, _ []*http.Request) error {
			redirects++
			next.Header["accept"] = []string{"application/xml"}
			return nil
		}
		result, err := objects.New(client).DeleteObject(context.Background(), "c", "o", objects.WithDeleteObjectStaticLargeObject(true))
		if err != nil || result == nil || result.Bulk == nil || calls != 2 || redirects != 1 {
			t.Fatalf("result=%+v err=%v calls=%d redirects=%d", result, err, calls, redirects)
		}
	})
	t.Run("SLO JSON Accept overrides native replacement but other headers and live auth survive", func(t *testing.T) {
		calls, hooks, callbacks := 0, 0, 0
		var client *gophercloud.ServiceClient
		client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method != http.MethodDelete || r.URL.Query().Get("multipart-manifest") != "delete" || r.Body != nil || r.Header.Get("Accept") != "application/json" {
				t.Errorf("wire=%s %s headers=%v", r.Method, r.URL, r.Header)
			}
			if calls == 1 {
				return deleteContractWire(503, "retry"), nil
			}
			if r.Header.Get("X-Native") != "override" || r.Header.Get("X-Source") != "" || r.Header.Get("X-Auth-Token") != "fresh-token" {
				t.Errorf("native retry headers=%v", r.Header)
			}
			return deleteContractWire(200, deleteContractBulkOK), nil
		})
		client.MoreHeaders = map[string]string{"Accept": "text/plain", "X-Source": "source"}
		provider := client.ProviderClient
		client.RetryFunc = func(_ context.Context, method, target string, options *gophercloud.RequestOpts, original error, count uint) error {
			hooks++
			if method != http.MethodDelete || count != 1 || !gophercloud.ResponseCodeIs(original, 503) {
				t.Errorf("hook=%s %s count=%d err=%v", method, target, count, original)
			}
			options.MoreHeaders = map[string]string{"Accept": "application/xml", "X-Native": "override"}
			client.SetToken("fresh-token")
			return nil
		}
		result, err := objects.New(client).DeleteObject(context.Background(), "c", "o", objects.WithDeleteObjectStaticLargeObject(true), func(*objects.DeleteObjectOpts) error { callbacks++; return nil })
		if err != nil || result == nil || result.Bulk == nil || calls != 2 || hooks != 1 || callbacks != 1 || client.ProviderClient != provider || client.MoreHeaders["Accept"] != "text/plain" {
			t.Fatalf("result=%+v err=%v calls=%d hooks=%d callbacks=%d", result, err, calls, hooks, callbacks)
		}
	})
	t.Run("nested retry and reauth404 never become ignored physical missing", func(t *testing.T) {
		for _, reauth := range []bool{false, true} {
			nested := &gophercloud.ErrUnexpectedResponseCode{Actual: 404, Body: []byte("nested404")}
			calls := 0
			client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
				calls++
				if reauth {
					return deleteContractWire(401, "actual401"), nil
				}
				return deleteContractWire(503, "actual503"), nil
			})
			if reauth {
				client.ReauthFunc = func(context.Context) error { return nested }
			} else {
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return nested }
			}
			result, err := objects.New(client).DeleteObject(context.Background(), "c", "o")
			deleteContractNoPhase(t, result)
			if calls != 1 || err == nil {
				t.Fatalf("reauth=%v result=%+v err=%v calls=%d", reauth, result, err, calls)
			}
			if reauth {
				var native *gophercloud.ErrUnableToReauthenticate
				if !errors.As(err, &native) || native.ErrReauth != nested || !gophercloud.ResponseCodeIs(native.ErrOriginal, 401) {
					t.Fatalf("reauth=%+v err=%v", native, err)
				}
			} else if !errors.Is(err, nested) {
				t.Fatal("retry cause lost", err)
			}
		}
	})
	t.Run("expanded codes cannot broaden owned phases and native Delete remains202204", func(t *testing.T) {
		calls := 0
		body := &objectMetadataBody{Reader: strings.NewReader("unexpected201")}
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return deleteContractWire(503, "retry"), nil
			}
			return objectMetadataWire(201, body), nil
		})
		client.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, _ error, _ uint) error {
			options.OkCodes = []int{200, 201, 202, 204}
			return nil
		}
		result, err := objects.New(client).DeleteObject(context.Background(), "c", "o", objects.WithDeleteObjectStaticLargeObject(false))
		deleteContractNoPhase(t, result)
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != 201 || !reflect.DeepEqual(native.Expected, []int{202, 204, 404}) || string(native.Body) != "unexpected201" || body.closes.Load() != 1 || calls != 2 {
			t.Fatalf("result=%+v err=%v native=%+v calls=%d", result, err, native, calls)
		}
		for _, status := range []int{202, 204} {
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodDelete {
					t.Errorf("native method=%s", r.Method)
				}
				return deleteContractWire(status, "native ack"), nil
			})
			if _, err := objects.New(client).Delete(context.Background(), "c", "o"); err != nil {
				t.Fatalf("native%d=%v", status, err)
			}
		}
	})
}
