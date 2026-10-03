package accounts_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/objectstorage/v1/accounts"
	"gophercloudsdk/resource"
)

const accountMetadataEndpoint = "https://swift.invalid/reverse/a%20b/v1/AUTH_account"

type accountMetadataTransport func(*http.Request) (*http.Response, error)

func (f accountMetadataTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := f(r)
	if response != nil && response.Request == nil {
		response.Request = r
	}
	return response, err
}

type accountMetadataBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *accountMetadataBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type accountMetadataReader func([]byte) (int, error)

func (f accountMetadataReader) Read(p []byte) (int, error) { return f(p) }

func accountMetadataClient(f accountMetadataTransport) *gophercloud.ServiceClient {
	p := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: f}}
	p.UseTokenLock()
	p.SetToken("original-token")
	return &gophercloud.ServiceClient{ProviderClient: p, Type: "object-store", Endpoint: accountMetadataEndpoint}
}

func accountMetadataWire(status int, body *accountMetadataBody) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}, "X-Trans-Id": {"actual-account"}}, Body: body}
}

func accountMetadataResponse(status int, body string) *http.Response {
	return accountMetadataWire(status, &accountMetadataBody{Reader: strings.NewReader(body)})
}

func accountMetadataProof(t *testing.T, err error, status int, body string) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Trans-Id") != "actual-account" {
		t.Fatalf("response proof=%+v err=%v", proof, err)
	}
	return proof
}

func accountMetadataNoBody(t *testing.T, r *http.Request) {
	t.Helper()
	if r.URL.String() != accountMetadataEndpoint || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "original-token" {
		t.Errorf("request=%s headers=%v", r.URL, r.Header)
	}
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) != 0 {
			t.Errorf("unexpected request body=%q err=%v", body, err)
		}
	}
}

func TestAccountMetadataContractsWireAndPresence(t *testing.T) {
	t.Run("exact endpoint and no discovery", func(t *testing.T) {
		for _, endpoint := range []string{accountMetadataEndpoint, accountMetadataEndpoint + "/", "https://swift.invalid/p%2Fq/v1/AUTH_x"} {
			t.Run(endpoint, func(t *testing.T) {
				calls := 0
				client := accountMetadataClient(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Method != "HEAD" || r.URL.String() != endpoint || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "original-token" {
						t.Errorf("request=%s %s headers=%v", r.Method, r.URL, r.Header)
					}
					return accountMetadataResponse(204, ""), nil
				})
				client.Type, client.Endpoint, client.ResourceBase = "", endpoint, "https://wrong.invalid/not-account?bad"
				result, err := accounts.New(client).GetMetadata(context.Background())
				if err != nil || result == nil || result.StatusCode != 204 || result.Metadata == nil || result.Metadata.Values == nil || calls != 1 {
					t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
				}
			})
		}
	})
	t.Run("single POST literal set and empty delete", func(t *testing.T) {
		calls := 0
		client := accountMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			accountMetadataNoBody(t, r)
			if r.Method != "POST" || r.Header.Get("X-Remove-Account-Meta-Book") != "" {
				t.Errorf("method=%s headers=%v", r.Method, r.Header)
			}
			if calls == 1 && (r.Header.Get("X-Account-Meta-Book") != "  한글 %2F\t" || r.Header.Get("X-Account-Meta-Empty") != "" || len(r.Header.Values("X-Account-Meta-Empty")) != 1) {
				t.Errorf("literal set headers=%v", r.Header)
			}
			if calls == 2 && !reflect.DeepEqual(r.Header.Values("X-Account-Meta-Book"), []string{""}) {
				t.Errorf("delete headers=%v", r.Header)
			}
			return accountMetadataResponse(204, "opaque"), nil
		})
		api := accounts.New(client)
		if result, err := api.SetMetadata(context.Background(), map[string]string{"Book": "  한글 %2F\t", "Empty": ""}); err != nil || result == nil || string(result.Body) != "opaque" {
			t.Fatalf("set=%+v err=%v", result, err)
		}
		if result, err := api.DeleteMetadata(context.Background(), []string{"Book"}); err != nil || result == nil {
			t.Fatalf("delete=%+v err=%v", result, err)
		}
		for _, invoke := range []func() error{
			func() error { _, err := api.SetMetadata(context.Background(), nil); return err },
			func() error { _, err := api.SetMetadata(context.Background(), map[string]string{}); return err },
			func() error { _, err := api.DeleteMetadata(context.Background(), nil); return err },
			func() error { _, err := api.DeleteMetadata(context.Background(), []string{}); return err },
		} {
			if err := invoke(); err != nil {
				t.Fatal(err)
			}
		}
		if calls != 6 {
			t.Fatalf("POSTs=%d; unexpected refresh/fallback", calls)
		}
	})
	t.Run("newest nil false true and clear", func(t *testing.T) {
		want := []string{"", "false", "true", ""}
		calls := 0
		client := accountMetadataClient(func(r *http.Request) (*http.Response, error) {
			accountMetadataNoBody(t, r)
			if r.Method != "HEAD" || r.Header.Get("X-Newest") != want[calls] || (want[calls] == "" && len(r.Header.Values("X-Newest")) != 0) {
				t.Errorf("newest=%v wanted=%q", r.Header.Values("X-Newest"), want[calls])
			}
			calls++
			return accountMetadataResponse(204, ""), nil
		})
		for _, opts := range [][]accounts.GetMetadataOption{nil, {accounts.WithGetMetadataNewest(false)}, {accounts.WithGetMetadataNewest(true)}, {accounts.WithGetMetadataNewest(true), accounts.WithoutGetMetadataNewest()}} {
			if _, err := accounts.New(client).GetMetadata(context.Background(), opts...); err != nil {
				t.Fatal(err)
			}
		}
		if calls != 4 {
			t.Fatalf("calls=%d", calls)
		}
	})
	t.Run("nullable signed precision and literal metadata", func(t *testing.T) {
		client := accountMetadataClient(func(r *http.Request) (*http.Response, error) {
			response := accountMetadataResponse(204, "non-json")
			response.Header["x-account-meta-BOOK"] = []string{"%2F + literal"}
			response.Header["X-Account-Meta-Temp-Url-Key"] = []string{"secret"}
			response.Header["X-Account-Meta-Temp-Url-Key-2"] = []string{"second"}
			response.Header["X-Account-Bytes-Used"] = []string{"+9007199254740993"}
			response.Header["X-Account-Container-Count"] = []string{"0"}
			response.Header["X-Account-Object-Count"] = []string{"-1"}
			response.Header["X-Timestamp"] = []string{""}
			response.Header["Date"] = []string{"not a timestamp"}
			response.Header["X-Vendor"] = []string{"one", "two"}
			return response, nil
		})
		result, err := accounts.New(client).GetMetadata(context.Background())
		if err != nil || result == nil || result.Metadata == nil {
			t.Fatalf("result=%+v err=%v", result, err)
		}
		m := result.Metadata
		if m.BytesUsed == nil || *m.BytesUsed != 9007199254740993 || m.ContainerCount == nil || *m.ContainerCount != 0 || m.ObjectCount == nil || *m.ObjectCount != -1 || m.Timestamp == nil || *m.Timestamp != "" || m.Values["book"] != "%2F + literal" || m.Values["temp-url-key"] != "secret" || m.Values["temp-url-key-2"] != "second" || string(result.Body) != "non-json" || !reflect.DeepEqual(result.Header["X-Vendor"], []string{"one", "two"}) {
			t.Fatalf("metadata=%+v raw=%+v", m, result)
		}
		client.ProviderClient.HTTPClient.Transport = accountMetadataTransport(func(*http.Request) (*http.Response, error) { return accountMetadataResponse(204, ""), nil })
		missing, err := accounts.New(client).GetMetadata(context.Background())
		if err != nil || missing.Metadata.BytesUsed != nil || missing.Metadata.ContainerCount != nil || missing.Metadata.ObjectCount != nil || missing.Metadata.Timestamp != nil || missing.Metadata.Values == nil {
			t.Fatalf("missing=%+v err=%v", missing, err)
		}
	})
}

func TestAccountMetadataContractsPreflightAndSnapshots(t *testing.T) {
	t.Run("context source and callback causes", func(t *testing.T) {
		calls, callbacks := 0, 0
		client := accountMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return accountMetadataResponse(204, ""), nil })
		option := func(o *accounts.GetMetadataOpts) error { callbacks++; return nil }
		var nilAPI *accounts.API
		for _, api := range []*accounts.API{nilAPI, accounts.New(nil), accounts.New(&gophercloud.ServiceClient{})} {
			if result, err := api.GetMetadata(context.Background(), option); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("nil source result=%+v err=%v", result, err)
			}
		}
		if result, err := accounts.New(client).GetMetadata(nil, option); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil context result=%+v err=%v", result, err)
		}
		cause := errors.New("cancelled by caller")
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(cause)
		if result, err := accounts.New(client).GetMetadata(ctx, option); result != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
			t.Fatalf("cancel result=%+v err=%v", result, err)
		}
		if callbacks != 0 || calls != 0 {
			t.Fatalf("callbacks=%d HTTP=%d", callbacks, calls)
		}
		wrong := *client
		wrong.Type = "image"
		if result, err := accounts.New(&wrong).GetMetadata(context.Background(), option); result != nil || !errors.Is(err, resource.ErrUnsupported) || callbacks != 0 || calls != 0 {
			t.Fatalf("wrong service result=%+v err=%v callbacks=%d HTTP=%d", result, err, callbacks, calls)
		}
		ctx, cancel = context.WithCancelCause(context.Background())
		optionCause := errors.New("callback failed")
		if _, err := accounts.New(client).GetMetadata(ctx, func(*accounts.GetMetadataOpts) error { cancel(cause); return optionCause }); !errors.Is(err, optionCause) || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
			t.Fatalf("joined callback error=%v", err)
		}
		if _, err := accounts.New(client).GetMetadata(context.Background(), nil); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatalf("nil callback err=%v HTTP=%d", err, calls)
		}
	})
	t.Run("input and protected headers validate before HTTP", func(t *testing.T) {
		calls := 0
		client := accountMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return accountMetadataResponse(204, ""), nil })
		api := accounts.New(client)
		for _, metadata := range []map[string]string{{"": "value"}, {"x-account-meta-book": "v"}, {"Book": "1", "book": "2"}, {"bad key": "v"}, {"한글": "v"}, {"bad:colon": "v"}, {"Book": "a\nb"}, {"Book": "\x7f"}, {"Book": string([]byte{255})}} {
			if result, err := api.SetMetadata(context.Background(), metadata); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("metadata=%v result=%+v err=%v", metadata, result, err)
			}
		}
		for _, keys := range [][]string{{""}, {"Book", "book"}, {"Book", "Book"}, {"X-ACCOUNT-META-Book"}, {"a/b"}} {
			if result, err := api.DeleteMetadata(context.Background(), keys); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("keys=%q result=%+v err=%v", keys, result, err)
			}
		}
		for _, key := range []string{"Authorization", "x-auth-token", "Host", "Content-Length", "Transfer-Encoding", "Connection", "Proxy-Connection", "Proxy-Authorization", "Upgrade", "Trailer", "TE", "X-Newest", "X-Account-Meta-Book", "X-Remove-Account-Meta-Book"} {
			client.MoreHeaders = map[string]string{key: "owned"}
			if _, err := api.GetMetadata(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("source header %s err=%v", key, err)
			}
			client.MoreHeaders = nil
			if _, err := api.SetMetadata(context.Background(), nil, accounts.WithMetadataHeader(key, "owned")); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("option header %s err=%v", key, err)
			}
		}
		for _, headers := range []map[string]string{{"X-Trace": "1", "x-trace": "2"}, {"bad key": "v"}, {"X-Trace": "\r"}} {
			if _, err := api.GetMetadata(context.Background(), accounts.WithGetMetadataHeaders(headers)); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("headers=%v err=%v", headers, err)
			}
		}
		if calls != 0 {
			t.Fatalf("HTTP before validation=%d", calls)
		}
	})
	t.Run("factories and caller payload own snapshots", func(t *testing.T) {
		headers := map[string]string{"x-trace": "initial"}
		newest := false
		getFull := accounts.WithGetMetadataOpts(accounts.GetMetadataOpts{Headers: headers, Newest: &newest})
		getHeaders := accounts.WithGetMetadataHeaders(headers)
		mutationFull := accounts.WithMetadataOpts(accounts.MetadataOpts{Headers: headers})
		mutationHeaders := accounts.WithMetadataHeaders(headers)
		headers["x-trace"], newest = "mutated", true
		calls := 0
		client := accountMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("X-Trace") != "final" || (r.Method == "HEAD" && r.Header.Get("X-Newest") != "false") {
				t.Errorf("headers=%v", r.Header)
			}
			return accountMetadataResponse(204, ""), nil
		})
		if _, err := accounts.New(client).GetMetadata(context.Background(), getHeaders, getFull, func(o *accounts.GetMetadataOpts) error {
			if o.Headers["x-trace"] != "initial" || o.Newest == nil || *o.Newest {
				t.Errorf("factory snapshot=%+v", o)
			}
			return nil
		}, accounts.WithGetMetadataHeader("X-Trace", "final")); err != nil {
			t.Fatal(err)
		}
		payload := map[string]string{"Book": "before callback"}
		client.ProviderClient.HTTPClient.Transport = accountMetadataTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("X-Trace") != "final" || r.Header.Get("X-Account-Meta-Book") != "before callback" {
				t.Errorf("set snapshot=%v", r.Header)
			}
			return accountMetadataResponse(204, ""), nil
		})
		if _, err := accounts.New(client).SetMetadata(context.Background(), payload, mutationHeaders, mutationFull, func(o *accounts.MetadataOpts) error {
			if o.Headers["x-trace"] != "initial" {
				t.Errorf("mutation factory snapshot=%+v", o)
			}
			payload["Book"] = "after callback"
			return nil
		}, accounts.WithMetadataHeader("X-Trace", "final")); err != nil {
			t.Fatal(err)
		}
		keys := []string{"Book"}
		client.ProviderClient.HTTPClient.Transport = accountMetadataTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if !reflect.DeepEqual(r.Header.Values("X-Account-Meta-Book"), []string{""}) || r.Header.Get("X-Account-Meta-Mutated") != "" {
				t.Errorf("delete snapshot=%v", r.Header)
			}
			return accountMetadataResponse(204, ""), nil
		})
		if _, err := accounts.New(client).DeleteMetadata(context.Background(), keys, func(*accounts.MetadataOpts) error { keys[0] = "Mutated"; return nil }); err != nil || calls != 3 {
			t.Fatalf("err=%v calls=%d", err, calls)
		}
	})
	t.Run("copy after every callback and whole replacement", func(t *testing.T) {
		retained := map[string]string{"X-Trace": "owned"}
		pointer := false
		callbacks := 0
		client := accountMetadataClient(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Trace") != "owned" || r.Header.Get("X-Newest") != "false" || r.Header.Get("X-Discarded") != "" {
				t.Errorf("retained callback alias=%v", r.Header)
			}
			return accountMetadataResponse(204, ""), nil
		})
		_, err := accounts.New(client).GetMetadata(context.Background(), accounts.WithGetMetadataHeader("X-Discarded", "old"), func(o *accounts.GetMetadataOpts) error {
			callbacks++
			*o = accounts.GetMetadataOpts{Headers: retained, Newest: &pointer}
			return nil
		}, func(o *accounts.GetMetadataOpts) error {
			callbacks++
			retained["X-Trace"], pointer = "stale alias", true
			return nil
		})
		if err != nil || callbacks != 2 {
			t.Fatalf("err=%v callbacks=%d", err, callbacks)
		}
		client.ProviderClient.HTTPClient.Transport = accountMetadataTransport(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Discarded") != "" || len(r.Header.Values("X-Newest")) != 0 {
				t.Errorf("full reset headers=%v", r.Header)
			}
			return accountMetadataResponse(204, ""), nil
		})
		if _, err := accounts.New(client).GetMetadata(context.Background(), accounts.WithGetMetadataNewest(true), accounts.WithGetMetadataHeader("X-Discarded", "old"), accounts.WithGetMetadataOpts(accounts.GetMetadataOpts{})); err != nil {
			t.Fatal(err)
		}
	})
}

func TestAccountMetadataContractsHeaderDecodeAndOwnership(t *testing.T) {
	t.Run("atomic malformed and duplicate canonical headers", func(t *testing.T) {
		cases := []http.Header{
			{"X-Account-Meta-": {"empty suffix"}}, {"X-Account-Meta-Bad Key": {"v"}}, {"X-Account-Meta-K": {"must not lowercase into ASCII"}}, {"X-Account-Meta-Book": {"1"}, "x-account-meta-book": {"2"}},
			{"X-Account-Meta-Book": {}}, {"X-Account-Meta-Book": {"1", "2"}}, {"X-Account-Meta-Book": {string([]byte{255})}},
			{"X-Account-Bytes-Used": {}}, {"X-Account-Bytes-Used": {"1", "2"}}, {"X-Account-Bytes-Used": {"1"}, "x-account-bytes-used": {"1"}},
			{"X-Timestamp": {}}, {"X-Timestamp": {"one", "two"}}, {"X-Timestamp": {"\x00"}},
		}
		for _, value := range []string{"", " 1", "1 ", "1.0", "0x10", "9223372036854775808", "-9223372036854775809"} {
			cases = append(cases, http.Header{"X-Account-Bytes-Used": {value}})
		}
		for i, headers := range cases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				body := &accountMetadataBody{Reader: strings.NewReader("raw-proof")}
				client := accountMetadataClient(func(*http.Request) (*http.Response, error) {
					response := accountMetadataWire(204, body)
					for key, values := range headers {
						response.Header[key] = values
					}
					return response, nil
				})
				result, err := accounts.New(client).GetMetadata(context.Background())
				if result == nil || result.Metadata != nil || err == nil || result.StatusCode != 204 || string(result.Body) != "raw-proof" || body.closes.Load() != 1 {
					t.Fatalf("result=%+v err=%v closes=%d", result, err, body.closes.Load())
				}
				accountMetadataProof(t, err, 204, "raw-proof")
				if headers.Get("X-Account-Bytes-Used") == "9223372036854775808" {
					var parse *strconv.NumError
					if !errors.As(err, &parse) {
						t.Fatalf("lost strconv cause=%v", err)
					}
				}
			})
		}
	})
	t.Run("success typed projection owns raw storage", func(t *testing.T) {
		header := http.Header{"X-Account-Meta-Book": {"original"}, "X-Account-Bytes-Used": {"3"}, "X-Timestamp": {"raw timestamp"}, "X-Vendor": {"one", "two"}}
		client := accountMetadataClient(func(*http.Request) (*http.Response, error) {
			response := accountMetadataResponse(204, "body")
			for key, values := range header {
				response.Header[key] = values
			}
			return response, nil
		})
		result, err := accounts.New(client).GetMetadata(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		header["X-Account-Meta-Book"][0] = "wire changed"
		header["X-Vendor"][0] = "wire changed"
		result.Header["X-Account-Meta-Book"][0] = "raw changed"
		result.Header["X-Account-Bytes-Used"][0] = "99"
		result.Header["X-Timestamp"][0] = "raw changed"
		if result.Metadata.Values["book"] != "original" || *result.Metadata.BytesUsed != 3 || *result.Metadata.Timestamp != "raw timestamp" || result.Header["X-Vendor"][0] != "one" || string(result.Body) != "body" {
			t.Fatalf("aliased result=%+v metadata=%+v", result, result.Metadata)
		}
	})
	t.Run("read close custom cancellation retained once", func(t *testing.T) {
		readCause, closeCause, cancelCause := errors.New("HEAD read"), errors.New("HEAD close"), errors.New("custom cancellation")
		ctx, cancel := context.WithCancelCause(context.Background())
		body := &accountMetadataBody{Reader: accountMetadataReader(func(p []byte) (int, error) { return copy(p, "partial"), readCause }), closeErr: closeCause, onClose: func() { cancel(cancelCause) }}
		calls, retries := 0, 0
		client := accountMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return accountMetadataWire(204, body), nil })
		client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			retries++
			return nil
		}
		result, err := accounts.New(client).GetMetadata(ctx)
		if result == nil || result.Metadata != nil || string(result.Body) != "partial" || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) || calls != 1 || retries != 0 || body.closes.Load() != 1 {
			t.Fatalf("result=%+v err=%v calls=%d retries=%d closes=%d", result, err, calls, retries, body.closes.Load())
		}
		proof := accountMetadataProof(t, err, 204, "partial")
		result.Body[0] = 'X'
		result.Header.Set("X-Trans-Id", "changed")
		if string(proof.Body) != "partial" || proof.Header.Get("X-Trans-Id") != "actual-account" {
			t.Fatalf("proof aliases result=%+v", proof)
		}
	})
	t.Run("unknown repeated quota dates remain passive", func(t *testing.T) {
		client := accountMetadataClient(func(*http.Request) (*http.Response, error) {
			response := accountMetadataResponse(204, "")
			response.Header["X-Account-Meta-Quota-Bytes"] = []string{"not a counter"}
			response.Header["Date"] = []string{"invalid", "repeated"}
			response.Header["X-Storage-Policy"] = []string{"future", "future2"}
			response.Header["X-Timestamp"] = []string{"  literal %2F\t"}
			return response, nil
		})
		result, err := accounts.New(client).GetMetadata(context.Background())
		if err != nil || result.Metadata.Values["quota-bytes"] != "not a counter" || *result.Metadata.Timestamp != "  literal %2F\t" || len(result.Header["Date"]) != 2 || len(result.Header["X-Storage-Policy"]) != 2 {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	})
}

func TestAccountMetadataContractsMutationEvidence(t *testing.T) {
	t.Run("accepted opaque POST read close context evidence", func(t *testing.T) {
		for _, deletion := range []bool{false, true} {
			t.Run(strconv.FormatBool(deletion), func(t *testing.T) {
				readCause, closeCause, cause := errors.New("POST read"), errors.New("POST close"), errors.New("POST cancelled")
				ctx, cancel := context.WithCancelCause(context.Background())
				body := &accountMetadataBody{Reader: accountMetadataReader(func(p []byte) (int, error) { return copy(p, "\xffopaque"), readCause }), closeErr: closeCause, onClose: func() { cancel(cause) }}
				calls := 0
				client := accountMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return accountMetadataWire(204, body), nil })
				api := accounts.New(client)
				var result *accounts.MetadataResponse
				var err error
				if deletion {
					result, err = api.DeleteMetadata(ctx, []string{"Book"})
				} else {
					result, err = api.SetMetadata(ctx, map[string]string{"Book": "value"})
				}
				if result == nil || result.StatusCode != 204 || string(result.Body) != "\xffopaque" || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || calls != 1 || body.closes.Load() != 1 {
					t.Fatalf("result=%+v err=%v calls=%d closes=%d", result, err, calls, body.closes.Load())
				}
				accountMetadataProof(t, err, 204, "\xffopaque")
			})
		}
	})
	t.Run("native failures never create ACK or suppress missing", func(t *testing.T) {
		for _, status := range []int{200, 201, 202, 401, 404, 503} {
			t.Run(strconv.Itoa(status), func(t *testing.T) {
				calls := 0
				body := &accountMetadataBody{Reader: strings.NewReader(`{"error":"actual"}`)}
				client := accountMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return accountMetadataWire(status, body), nil })
				result, err := accounts.New(client).DeleteMetadata(context.Background(), []string{"Book"})
				var native gophercloud.ErrUnexpectedResponseCode
				if result != nil || !errors.As(err, &native) || native.Actual != status || !reflect.DeepEqual(native.Expected, []int{204}) || native.Method != "POST" || native.URL != accountMetadataEndpoint || calls != 1 || body.closes.Load() != 1 {
					t.Fatalf("result=%+v err=%v native=%+v calls=%d closes=%d", result, err, native, calls, body.closes.Load())
				}
			})
		}
		transportCause := errors.New("transport failed")
		client := accountMetadataClient(func(*http.Request) (*http.Response, error) { return nil, transportCause })
		if result, err := accounts.New(client).SetMetadata(context.Background(), nil); result != nil || !errors.Is(err, transportCause) {
			t.Fatalf("transport result=%+v err=%v", result, err)
		}
		client = accountMetadataClient(func(*http.Request) (*http.Response, error) {
			return accountMetadataResponse(200, "not accepted HEAD"), nil
		})
		result, err := accounts.New(client).GetMetadata(context.Background())
		var native gophercloud.ErrUnexpectedResponseCode
		if result != nil || !errors.As(err, &native) || native.Actual != 200 || native.Method != "HEAD" || !reflect.DeepEqual(native.Expected, []int{204}) {
			t.Fatalf("HEAD200 result=%+v err=%v native=%+v", result, err, native)
		}
	})
	t.Run("accepted source failure retains raw acknowledgement", func(t *testing.T) {
		var client *gophercloud.ServiceClient
		body := &accountMetadataBody{Reader: strings.NewReader("already applied"), onClose: func() { client.ProviderClient = &gophercloud.ProviderClient{} }}
		client = accountMetadataClient(func(*http.Request) (*http.Response, error) { return accountMetadataWire(204, body), nil })
		result, err := accounts.New(client).SetMetadata(context.Background(), nil)
		if result == nil || !errors.Is(err, resource.ErrInvalidOption) || body.closes.Load() != 1 {
			t.Fatalf("result=%+v err=%v closes=%d", result, err, body.closes.Load())
		}
		accountMetadataProof(t, err, 204, "already applied")
	})
	t.Run("POST is raw only and never refreshed projection", func(t *testing.T) {
		calls := 0
		wireHeader := http.Header{"X-Account-Meta-Book": {"one", "two"}, "X-Account-Bytes-Used": {"not numeric"}}
		client := accountMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method != "POST" {
				t.Errorf("unexpected refresh=%s", r.Method)
			}
			response := accountMetadataResponse(204, "not JSON")
			for key, values := range wireHeader {
				response.Header[key] = values
			}
			return response, nil
		})
		result, err := accounts.New(client).SetMetadata(context.Background(), map[string]string{"Book": "submitted"})
		wireHeader["X-Account-Meta-Book"][0] = "mutated"
		if err != nil || result == nil || calls != 1 || string(result.Body) != "not JSON" || !reflect.DeepEqual(result.Header["X-Account-Meta-Book"], []string{"one", "two"}) {
			t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
		}
	})
}

func TestAccountMetadataContractsSourceAndNativePolicy(t *testing.T) {
	t.Run("captured headers live token and irrelevant ResourceBase", func(t *testing.T) {
		client := accountMetadataClient(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Auth-Token") != "latest-token" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Trace") != "option" || r.Header.Get("Accept") != "source/accept" || r.Header.Get("Content-Type") != "source/type" || r.URL.String() != accountMetadataEndpoint {
				t.Errorf("request headers=%v URL=%s", r.Header, r.URL)
			}
			return accountMetadataResponse(204, ""), nil
		})
		original := client.ProviderClient
		client.MoreHeaders = map[string]string{"X-Source": "captured", "X-Trace": "source", "Accept": "source/accept", "Content-Type": "source/type"}
		_, err := accounts.New(client).GetMetadata(context.Background(), accounts.WithGetMetadataHeader("X-Trace", "option"), func(*accounts.GetMetadataOpts) error {
			client.MoreHeaders["X-Source"] = "later"
			client.ResourceBase = "https://wrong.invalid/no-authority"
			original.SetToken("latest-token")
			return nil
		})
		if err != nil || client.MoreHeaders["X-Trace"] != "source" || client.ProviderClient != original {
			t.Fatalf("err=%v source=%v", err, client.MoreHeaders)
		}
	})
	t.Run("fixed identity and current header validation", func(t *testing.T) {
		for _, mutate := range []func(*gophercloud.ServiceClient){
			func(c *gophercloud.ServiceClient) { c.Endpoint += "/other" },
			func(c *gophercloud.ServiceClient) { c.Type = "image" },
			func(c *gophercloud.ServiceClient) { c.Microversion = "changed" },
			func(c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} },
			func(c *gophercloud.ServiceClient) { c.MoreHeaders = map[string]string{"X-Account-Meta-Owned": "bad"} },
		} {
			calls := 0
			client := accountMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return accountMetadataResponse(204, ""), nil })
			result, err := accounts.New(client).GetMetadata(context.Background(), func(*accounts.GetMetadataOpts) error { mutate(client); return nil })
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatalf("result=%+v err=%v HTTP=%d", result, err, calls)
			}
		}
		for _, endpoint := range []string{"https://swift.invalid/a?query=1", "https://swift.invalid/a#fragment", "ftp://swift.invalid/a", "https://user@swift.invalid/a"} {
			client := accountMetadataClient(func(*http.Request) (*http.Response, error) {
				t.Error("invalid endpoint reached HTTP")
				return nil, errors.New("unexpected")
			})
			client.Endpoint = endpoint
			if _, err := accounts.New(client).GetMetadata(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("endpoint=%s err=%v", endpoint, err)
			}
		}
	})
	t.Run("native prebody retries and ownership guard", func(t *testing.T) {
		for _, change := range []struct {
			name string
			fn   func(*gophercloud.RequestOpts)
		}{
			{"keep body", func(o *gophercloud.RequestOpts) { o.KeepResponseBody = false }},
			{"JSON response", func(o *gophercloud.RequestOpts) { o.JSONResponse = new(any) }},
			{"raw body", func(o *gophercloud.RequestOpts) { o.RawBody = strings.NewReader("changed") }},
			{"JSON null", func(o *gophercloud.RequestOpts) { o.JSONBody = json.RawMessage("null") }},
			{"encoding failure", func(o *gophercloud.RequestOpts) { o.JSONBody = make(chan int) }},
		} {
			t.Run(change.name, func(t *testing.T) {
				calls, hooks := 0, 0
				body := &accountMetadataBody{Reader: strings.NewReader(`{"error":"first"}`)}
				client := accountMetadataClient(func(r *http.Request) (*http.Response, error) {
					calls++
					accountMetadataNoBody(t, r)
					return accountMetadataWire(503, body), nil
				})
				client.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
					hooks++
					change.fn(o)
					return nil
				}
				result, err := accounts.New(client).SetMetadata(context.Background(), nil)
				var native gophercloud.ErrUnexpectedResponseCode
				if result != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || calls != 1 || hooks != 1 || body.closes.Load() != 1 {
					t.Fatalf("result=%+v err=%v calls=%d hooks=%d closes=%d", result, err, calls, hooks, body.closes.Load())
				}
				if change.name == "encoding failure" {
					var encoding *json.UnsupportedTypeError
					if !errors.As(err, &encoding) {
						t.Fatalf("encoding cause lost=%v", err)
					}
				}
			})
		}
		calls, hooks := 0, 0
		client := accountMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return accountMetadataResponse(503, `{"error":"retry"}`), nil
			}
			if r.Header.Get("X-Advanced") != "provider-owned" {
				t.Errorf("advanced header=%v", r.Header)
			}
			return accountMetadataResponse(204, ""), nil
		})
		original := client.ProviderClient
		client.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			hooks++
			o.MoreHeaders = map[string]string{"X-Advanced": "provider-owned"}
			return nil
		}
		if _, err := accounts.New(client).SetMetadata(context.Background(), nil); err != nil || calls != 2 || hooks != 1 || client.ProviderClient != original || client.MoreHeaders != nil {
			t.Fatalf("retry err=%v calls=%d hooks=%d source=%v", err, calls, hooks, client.MoreHeaders)
		}
		calls, hooks = 0, 0
		transportCause := errors.New("retry transport")
		client = accountMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return nil, transportCause
			}
			return accountMetadataResponse(204, ""), nil
		})
		client.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			hooks++
			if !errors.Is(err, transportCause) {
				t.Errorf("original transport cause=%v", err)
			}
			return nil
		}
		if _, err := accounts.New(client).GetMetadata(context.Background()); err != nil || calls != 2 || hooks != 1 {
			t.Fatalf("transport retry err=%v calls=%d hooks=%d", err, calls, hooks)
		}
	})
	t.Run("actual status gate reauth and native ABI", func(t *testing.T) {
		calls := 0
		closeCause := errors.New("expanded code close")
		body := &accountMetadataBody{Reader: strings.NewReader("unexpected accepted body"), closeErr: closeCause}
		client := accountMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return accountMetadataResponse(503, `{"error":"retry"}`), nil
			}
			return accountMetadataWire(202, body), nil
		})
		client.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			o.OkCodes = []int{202}
			return nil
		}
		result, err := accounts.New(client).SetMetadata(context.Background(), nil)
		var native gophercloud.ErrUnexpectedResponseCode
		if result != nil || !errors.As(err, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, []int{204}) || string(native.Body) != "unexpected accepted body" || native.ResponseHeader.Get("X-Trans-Id") != "actual-account" || !errors.Is(err, closeCause) || calls != 2 || body.closes.Load() != 1 {
			t.Fatalf("result=%+v err=%v native=%+v calls=%d", result, err, native, calls)
		}
		calls, reauth := 0, 0
		client = accountMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return accountMetadataResponse(401, `{"error":"expired"}`), nil
			}
			if r.Header.Get("X-Auth-Token") != "reauthenticated" {
				t.Errorf("token=%v", r.Header)
			}
			return accountMetadataResponse(204, ""), nil
		})
		client.ReauthFunc = func(context.Context) error { reauth++; client.SetToken("reauthenticated"); return nil }
		if _, err := accounts.New(client).GetMetadata(context.Background()); err != nil || calls != 2 || reauth != 1 {
			t.Fatalf("reauth err=%v calls=%d reauth=%d", err, calls, reauth)
		}
		reauthCause := errors.New("reauth rejected")
		client = accountMetadataClient(func(*http.Request) (*http.Response, error) {
			return accountMetadataResponse(401, `{"error":"expired"}`), nil
		})
		client.ReauthFunc = func(context.Context) error { return reauthCause }
		_, err = accounts.New(client).GetMetadata(context.Background())
		var reauthError *gophercloud.ErrUnableToReauthenticate
		if !errors.As(err, &reauthError) || reauthError.ErrReauth != reauthCause || !errors.As(reauthError.ErrOriginal, &native) || native.Actual != 401 {
			t.Fatalf("native reauth fields=%+v err=%v", reauthError, err)
		}
		calls, redirects := 0, 0
		client = accountMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				response := accountMetadataResponse(307, "")
				response.Header.Set("Location", accountMetadataEndpoint)
				return response, nil
			}
			return accountMetadataResponse(204, ""), nil
		})
		client.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects++; return nil }
		if _, err := accounts.New(client).GetMetadata(context.Background()); err != nil || calls != 2 || redirects != 1 {
			t.Fatalf("same-target redirect err=%v calls=%d redirects=%d", err, calls, redirects)
		}
		calls = 0
		client = accountMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			response := accountMetadataResponse(307, "")
			response.Header.Set("Location", "https://other.invalid/stolen")
			return response, nil
		})
		if _, err := accounts.New(client).GetMetadata(context.Background()); !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
			t.Fatalf("retarget redirect err=%v calls=%d", err, calls)
		}
		client = accountMetadataClient(func(r *http.Request) (*http.Response, error) {
			status := 204
			if r.Method == "POST" {
				status = 202
			}
			response := accountMetadataResponse(status, "")
			response.Header.Set("Date", "Fri, 17 Jan 2014 16:09:56 GMT")
			return response, nil
		})
		api := accounts.New(client)
		var get *accounts.GetHeader
		get, err = api.Get(context.Background(), accounts.WithGetOptions(accounts.GetOpts{Newest: true}))
		if err != nil || get == nil {
			t.Fatalf("native Get=%+v err=%v", get, err)
		}
		var update *accounts.UpdateHeader
		update, err = api.Update(context.Background(), accounts.UpdateOpts{Metadata: map[string]string{"Book": "native"}}, accounts.WithUpdateHeader("X-Trace", "native"))
		if err != nil || update == nil {
			t.Fatalf("native202 Update=%+v err=%v", update, err)
		}
		if client.Endpoint != accountMetadataEndpoint {
			t.Fatal("native client changed")
		}
	})
}
