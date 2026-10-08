package cindermetadata

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/metadata"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func testScope(t *testing.T, collection string, handler http.HandlerFunc) (*Scope, *gophercloud.ServiceClient) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	provider := &gophercloud.ProviderClient{HTTPClient: *server.Client()}
	provider.UseTokenLock()
	provider.SetToken("first-token")
	client := &gophercloud.ServiceClient{ProviderClient: provider, Endpoint: server.URL + "/catalog/", ResourceBase: server.URL + "/proxy/project/", Type: "block-storage", Microversion: "3.15"}
	scope, err := New(context.Background(), client, collection, "requested-id")
	if err != nil {
		t.Fatal(err)
	}
	return scope, client
}

func jsonResponse(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-Id", "actual-response")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, body)
}

func assertOperation(t *testing.T, err error, operation, kind string) {
	t.Helper()
	var op *resource.OperationError
	if !errors.As(err, &op) || op.Operation != operation || op.Resource != kind {
		t.Fatalf("operation error=%v", err)
	}
}

func TestMetadataMethodsKeepActualEvidenceExplicitEmptyAndInputSnapshots(t *testing.T) {
	for _, collection := range []string{"volumes", "snapshots"} {
		t.Run(collection, func(t *testing.T) {
			var methods, bodies []string
			const response = `{"metadata":{"actual":"response","empty":""},"Metadata":{"decoy":3},"extension":900719925474099312345}`
			scope, client := testScope(t, collection, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.EscapedPath() != "/proxy/project/"+collection+"/requested-id/metadata" || r.URL.RawQuery != "" {
					t.Errorf("target=%s", r.URL)
				}
				if r.Header.Get("X-Auth-Token") != "first-token" || r.Header.Get("OpenStack-API-Version") != "volume 3.15" {
					t.Errorf("headers=%v", r.Header)
				}
				body, _ := io.ReadAll(r.Body)
				methods = append(methods, r.Method)
				bodies = append(bodies, string(body))
				jsonResponse(w, response)
			})
			got, err := scope.Get(context.Background())
			if err != nil || got.StatusCode != 200 || got.Header.Get("X-Request-Id") != "actual-response" || string(got.Body) != response || !reflect.DeepEqual(got.Metadata, map[string]string{"actual": "response", "empty": ""}) {
				t.Fatalf("get=%+v err=%v", got, err)
			}
			input := map[string]string{"initial": "owned"}
			mutate := func(*request.Config[metadata.Opts]) error {
				input["initial"] = "changed"
				input["extra"] = "caller"
				return nil
			}
			if _, err := scope.Merge(context.Background(), input, mutate); err != nil {
				t.Fatal(err)
			}
			if _, err := scope.Replace(context.Background(), nil); err != nil {
				t.Fatal(err)
			}
			cleared, err := scope.DeleteKeys(context.Background(), nil)
			if err != nil || cleared.Cleared == nil || cleared.Cleared.StatusCode != 200 || len(cleared.Deleted) != 0 {
				t.Fatalf("clear=%+v err=%v", cleared, err)
			}
			empty, err := scope.DeleteKeys(context.Background(), []string{})
			if err != nil || empty.Cleared != nil || empty.Deleted == nil || len(empty.Deleted) != 0 {
				t.Fatalf("empty=%+v err=%v", empty, err)
			}
			if !reflect.DeepEqual(methods, []string{"GET", "POST", "PUT", "PUT"}) || !reflect.DeepEqual(bodies, []string{"", `{"metadata":{"initial":"owned"}}`, `{"metadata":{}}`, `{"metadata":{}}`}) {
				t.Fatalf("methods=%v bodies=%v", methods, bodies)
			}
			got.Metadata["actual"] = "local"
			got.Body[0] = ' '
			got.Header.Set("X-Request-Id", "local")
			if string(cleared.Cleared.Body) != response || cleared.Cleared.Metadata["actual"] != "response" || cleared.Cleared.Header.Get("X-Request-Id") != "actual-response" || scope.ID() != "requested-id" || scope.RawClient() != client {
				t.Fatal("responses or fixed identity share caller storage")
			}
		})
	}
}

func TestMetadataCanonicalStringEnvelopeErrorsKeepWholeAcceptedResponse(t *testing.T) {
	for _, body := range []string{`{}`, `null`, `[]`, `{"Metadata":{}}`, `{"metadata":null}`, `{"metadata":[]}`, `{"metadata":"value"}`, `{"metadata":{"key":null}}`, `{"metadata":{"key":1}}`, `{"metadata":{"key":true}}`, `{"metadata":{"key":{}}}`, `{"metadata":{"key":[]}}`, `{"metadata":{"key":"x"},`} {
		t.Run(body, func(t *testing.T) {
			var calls int
			scope, _ := testScope(t, "volumes", func(w http.ResponseWriter, r *http.Request) { calls++; jsonResponse(w, body) })
			got, err := scope.Get(context.Background())
			var proof *resource.ResponseError
			if got != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "actual-response" || proof.Cause == nil || calls != 1 {
				t.Fatalf("got=%+v error=%v proof=%+v calls=%d", got, err, proof, calls)
			}
			assertOperation(t, err, "Get", "volumes_metadata")
		})
	}
	for _, status := range []int{201, 202, 204, 403, 404, 412, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls int
			scope, _ := testScope(t, "snapshots", func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("X-Request-Id", "native")
				w.WriteHeader(status)
				if status != 204 {
					_, _ = io.WriteString(w, `{"error":"native"}`)
				}
			})
			got, err := scope.Replace(context.Background(), map[string]string{"sent": "input"})
			var native gophercloud.ErrUnexpectedResponseCode
			var proof *resource.ResponseError
			if got != nil || !errors.As(err, &native) || native.Actual != status || native.ResponseHeader.Get("X-Request-Id") != "native" || errors.As(err, &proof) || calls != 1 {
				t.Fatalf("got=%v error=%v native=%+v calls=%d", got, err, native, calls)
			}
			assertOperation(t, err, "Replace", "snapshots_metadata")
		})
	}
}

func TestMetadataDeleteKeepsLiteralKeysOrderDuplicatesAndPartialAcknowledgements(t *testing.T) {
	var seen []string
	scope, _ := testScope(t, "snapshots", func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.EscapedPath())
		if r.Method != "DELETE" {
			t.Errorf("method=%s", r.Method)
		}
		w.Header().Set("X-Ack", fmt.Sprint(len(seen)))
		if len(seen) == 3 {
			w.WriteHeader(404)
			_, _ = io.WriteString(w, "missing duplicate")
			return
		}
		w.WriteHeader(200)
		_, _ = io.WriteString(w, fmt.Sprintf("actual-ack-%d", len(seen)))
	})
	keys := []string{"literal/ %é?fragment#", "duplicate", "duplicate", "never"}
	mutate := func(*request.Config[metadata.Opts]) error { keys[0] = "mutated"; keys[1] = "changed"; return nil }
	got, err := scope.DeleteKeys(context.Background(), keys, mutate)
	var native gophercloud.ErrUnexpectedResponseCode
	if got == nil || got.Cleared != nil || len(got.Deleted) != 2 || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "missing duplicate" || native.ResponseHeader.Get("X-Ack") != "3" {
		t.Fatalf("got=%+v err=%v native=%+v", got, err, native)
	}
	want := []string{"/proxy/project/snapshots/requested-id/metadata/" + url.PathEscape("literal/ %é?fragment#"), "/proxy/project/snapshots/requested-id/metadata/duplicate", "/proxy/project/snapshots/requested-id/metadata/duplicate"}
	if !reflect.DeepEqual(seen, want) || got.Deleted[0].Key != "literal/ %é?fragment#" || got.Deleted[1].Key != "duplicate" {
		t.Fatalf("paths=%v acks=%+v", seen, got.Deleted)
	}
	for i, ack := range got.Deleted {
		if ack.StatusCode != 200 || string(ack.Body) != fmt.Sprintf("actual-ack-%d", i+1) || ack.Header.Get("X-Ack") != fmt.Sprint(i+1) {
			t.Fatalf("ack=%+v", ack)
		}
	}
	got.Deleted[0].Body[0] = 'x'
	got.Deleted[0].Header.Set("X-Ack", "changed")
	if string(got.Deleted[1].Body) != "actual-ack-2" || got.Deleted[1].Header.Get("X-Ack") != "2" {
		t.Fatal("successful acknowledgements share storage")
	}
	assertOperation(t, err, "DeleteKeys", "snapshots_metadata")
	for _, bad := range []string{"", ".", "..", "bad\nkey", string([]byte{0xff})} {
		before := len(seen)
		got, err := scope.DeleteKeys(context.Background(), []string{"first", bad})
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || len(seen) != before {
			t.Fatalf("key=%q result=%v err=%v HTTP=%d", bad, got, err, len(seen)-before)
		}
	}
}

func TestMetadataHeaderSnapshotsConditionsAndProtectedCapabilities(t *testing.T) {
	var headers []http.Header
	scope, client := testScope(t, "volumes", func(w http.ResponseWriter, r *http.Request) {
		headers = append(headers, r.Header.Clone())
		w.Header().Set("ETag", `"actual-etag"`)
		if r.Header.Get("If-Match") == `"stale"` {
			w.WriteHeader(412)
			_, _ = io.WriteString(w, "conditional failure")
			return
		}
		jsonResponse(w, `{"metadata":{"actual":"response"}}`)
	})
	client.MoreHeaders = map[string]string{"x-source": "configured", "if-match": `"current"`, "X-OpenStack-Volume-API-Version": "3.15", "openstack-api-version": "volume 3.15"}
	input := map[string]string{"x-request-id": "frozen", "If-Match": `"current"`, "x-source": "per-call"}
	opt := metadata.WithHeaders(input)
	input["x-request-id"] = "changed"
	got, err := scope.Replace(context.Background(), nil, opt, metadata.WithHeader("X-Request-ID", "last"))
	if err != nil || got.Header.Get("ETag") != `"actual-etag"` || len(headers) != 1 || headers[0].Get("If-Match") != `"current"` || headers[0].Get("X-Request-Id") != "last" || headers[0].Get("X-Source") != "configured" {
		t.Fatalf("got=%+v err=%v headers=%v", got, err, headers)
	}
	if client.MoreHeaders["if-match"] != `"current"` || len(client.MoreHeaders) != 4 {
		t.Fatal("configured source headers mutated")
	}
	if _, err := scope.Replace(context.Background(), nil, metadata.WithHeader("if-match", `"different"`)); !errors.Is(err, resource.ErrInvalidOption) || len(headers) != 1 {
		t.Fatalf("conflict error=%v calls=%d", err, len(headers))
	}
	delete(client.MoreHeaders, "if-match")
	if value, err := scope.Replace(context.Background(), nil, metadata.WithHeader("If-Match", `"stale"`)); value != nil {
		t.Fatalf("stale result=%+v err=%v", value, err)
	} else {
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != 412 || string(native.Body) != "conditional failure" {
			t.Fatalf("stale native error=%v", err)
		}
	}
	if len(headers) != 2 {
		t.Fatalf("conditional operation was retried: %d", len(headers))
	}
	for _, key := range []string{"X-Auth-Token", "X-Service-Token", "Authorization", "Host", "Cookie", "Content-Type", "Content-Length", "OpenStack-API-Version", "X-OpenStack-Volume-API-Version", "Transfer-Encoding", "Connection"} {
		before := len(headers)
		_, err := scope.Get(context.Background(), metadata.WithHeader(key, "owned"))
		if !errors.Is(err, resource.ErrInvalidOption) || len(headers) != before {
			t.Fatalf("protected=%s err=%v calls=%d", key, err, len(headers)-before)
		}
	}
	for _, source := range []map[string]string{{"X-Trace": "one", "x-trace": "two"}, {"OpenStack-API-Version": "volume 3.14"}, {"X-OpenStack-Volume-API-Version": ""}, {"X-Auth-Token": "alternate"}} {
		client.MoreHeaders = source
		if _, err := scope.DeleteKeys(context.Background(), []string{}); !errors.Is(err, resource.ErrInvalidOption) || len(headers) != 2 {
			t.Fatalf("source=%v err=%v calls=%d", source, err, len(headers))
		}
	}
	client.MoreHeaders = map[string]string{"X-Trace": "same", "x-trace": "same"}
	if _, err := scope.Get(context.Background()); err != nil || headers[len(headers)-1].Get("X-Trace") != "same" {
		t.Fatalf("equal source aliases err=%v", err)
	}
	// Custom options cannot introduce body/query/version capabilities.
	for _, option := range []metadata.Option{nil, request.WithField[metadata.Opts]("extension", true), request.WithQuery[metadata.Opts]("limit", "1"), request.WithArgument[metadata.Opts]("microversion", "3.99"), metadata.WithHeader("Bad:Header", "value"), metadata.WithHeader("X-Control", "\x01")} {
		before := len(headers)
		_, err := scope.Get(context.Background(), option)
		if err == nil || len(headers) != before {
			t.Fatalf("unsupported option err=%v HTTP=%d", err, len(headers)-before)
		}
	}
}

func TestMetadataPreparedOptionsAreIndependentAndLiveTokenKeepsFixedTarget(t *testing.T) {
	var seen []http.Header
	scope, client := testScope(t, "volumes", func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Clone())
		if r.URL.EscapedPath() != "/proxy/project/volumes/requested-id/metadata/one" && r.URL.EscapedPath() != "/proxy/project/volumes/requested-id/metadata/two" && r.URL.EscapedPath() != "/proxy/project/volumes/requested-id/metadata" {
			t.Errorf("fixed target=%s", r.URL)
		}
		w.WriteHeader(200)
	})
	var retained *request.Config[metadata.Opts]
	custom := func(cfg *request.Config[metadata.Opts]) error {
		cfg.Headers = map[string]string{"X-Frozen": "before"}
		retained = cfg
		return nil
	}
	underlying := client.ProviderClient.HTTPClient.Transport
	client.ProviderClient.HTTPClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		wire, err := underlying.RoundTrip(r)
		if len(seen) == 1 {
			retained.Headers["X-Frozen"] = "after"
			retained.Headers["If-Match"] = "injected"
			client.ProviderClient.SetToken("rotated-token")
			client.ResourceBase = client.Endpoint + "retarget/"
		}
		return wire, err
	})
	got, err := scope.DeleteKeys(context.Background(), []string{"one", "two"}, custom)
	if err != nil || len(got.Deleted) != 2 || len(seen) != 2 || seen[0].Get("X-Frozen") != "before" || seen[1].Get("X-Frozen") != "before" || seen[1].Get("If-Match") != "" || seen[0].Get("X-Auth-Token") != "first-token" || seen[1].Get("X-Auth-Token") != "rotated-token" {
		t.Fatalf("result=%+v err=%v headers=%v", got, err, seen)
	}
	// The selected URI is fixed, even when ResourceBase subsequently changes.
	if _, err := scope.DeleteKeys(context.Background(), []string{}); err != nil {
		t.Fatal(err)
	}
	client.ProviderClient.HTTPClient.Transport = underlying
}

type failingBody struct {
	data  string
	cause error
}

func (b *failingBody) Read(p []byte) (int, error) {
	if b.data != "" {
		n := copy(p, b.data)
		b.data = b.data[n:]
		return n, nil
	}
	return 0, b.cause
}
func (*failingBody) Close() error { return nil }

func TestMetadataNativeTransportAcceptedReadAndCancellationErrorsStayTerminal(t *testing.T) {
	readCause := errors.New("accepted response read failed")
	transportCause := errors.New("transport failed")
	for _, tc := range []struct {
		name   string
		status int
		body   io.ReadCloser
		cause  error
		cancel bool
		proof  bool
	}{
		{"accepted-read", 200, &failingBody{data: `{"metadata":`, cause: readCause}, readCause, false, true},
		{"accepted-canceled", 200, io.NopCloser(strings.NewReader(`{"metadata":{}}`)), context.Canceled, true, true},
		{"native-canceled", 404, io.NopCloser(strings.NewReader("actual missing")), context.Canceled, true, false},
		{"transport", 0, nil, transportCause, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var calls int
			provider := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if tc.cancel {
					cancel()
				}
				if tc.status == 0 {
					return nil, transportCause
				}
				return &http.Response{StatusCode: tc.status, Header: http.Header{"X-Evidence": []string{"actual"}}, Body: tc.body, Request: r}, nil
			})}}
			client := &gophercloud.ServiceClient{ProviderClient: provider, Endpoint: "http://metadata.invalid/", Type: "block-storage"}
			scope, err := New(ctx, client, "snapshots", "fixed")
			if err != nil {
				t.Fatal(err)
			}
			got, err := scope.Get(ctx)
			if got != nil || !errors.Is(err, tc.cause) || calls != 1 {
				t.Fatalf("result=%v err=%v calls=%d", got, err, calls)
			}
			var proof *resource.ResponseError
			if errors.As(err, &proof) != tc.proof {
				t.Fatalf("proof=%+v err=%v", proof, err)
			}
			if proof != nil && (proof.StatusCode != 200 || proof.Header.Get("X-Evidence") != "actual" || len(proof.Body) == 0) {
				t.Fatalf("proof=%+v", proof)
			}
			if tc.name == "native-canceled" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "actual missing" || native.ResponseHeader.Get("X-Evidence") != "actual" {
					t.Fatalf("native=%+v err=%v", native, err)
				}
			}
			if tc.name == "transport" {
				var wrapped *url.Error
				if !errors.As(err, &wrapped) {
					t.Fatalf("transport original wrapper=%v", err)
				}
			}
			assertOperation(t, err, "Get", "snapshots_metadata")
		})
	}
}

func TestMetadataPreflightZeroHTTPAndBetweenDeletionSourceChecks(t *testing.T) {
	var calls int
	scope, client := testScope(t, "volumes", func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(200) })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, run := range []func() error{
		func() error { _, err := scope.DeleteKeys(ctx, []string{}); return err },
		func() error { _, err := scope.Get(nil); return err },
		func() error { _, err := (*Scope)(nil).Merge(context.Background(), nil); return err },
		func() error { _, err := (*Scope)(nil).DeleteKeys(context.Background(), nil); return err },
		func() error {
			_, err := scope.DeleteKeys(context.Background(), []string{}, func(*request.Config[metadata.Opts]) error { client.Type = "compute"; return nil })
			return err
		},
	} {
		client.Type = "block-storage"
		if err := run(); err == nil || calls != 0 {
			t.Fatalf("preflight err=%v calls=%d", err, calls)
		}
	}
	if (*Scope)(nil).ID() != "" || (*Scope)(nil).RawClient() != nil {
		t.Fatal("nil getters")
	}
	client.Type = "block-storage"
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	_, err := scope.DeleteKeys(ctx, []string{}, func(*request.Config[metadata.Opts]) error { cancel(); return nil })
	if !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatalf("post option cancellation err=%v calls=%d", err, calls)
	}
	underlying := client.ProviderClient.HTTPClient.Transport
	client.ProviderClient.HTTPClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		response, err := underlying.RoundTrip(r)
		client.Type = "compute"
		return response, err
	})
	result, err := scope.DeleteKeys(context.Background(), []string{"first", "second"})
	if result == nil || len(result.Deleted) != 1 || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
		t.Fatalf("partial=%+v err=%v calls=%d", result, err, calls)
	}
	for _, bad := range []string{"", "..", "with/slash", "with space", "with%escape"} {
		client.Type = "block-storage"
		if _, err := New(context.Background(), client, "volumes", bad); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("ID=%q err=%v", bad, err)
		}
	}
	client.Type = "block-storage"
	if _, err := New(context.Background(), client, "backups", "fixed"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("Backup scope error=%v", err)
	}
}

func TestMetadataConcurrentReuseKeepsRequestAndResponseOwnership(t *testing.T) {
	var calls atomic.Int32
	scope, client := testScope(t, "snapshots", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"metadata":{"stable":"input"}}` || r.Header.Get("X-Frozen") != "header" {
			t.Errorf("body=%s header=%v", body, r.Header)
		}
		jsonResponse(w, `{"metadata":{"response":"owned"}}`)
	})
	input := map[string]string{"stable": "input"}
	headers := map[string]string{"X-Frozen": "header"}
	option := metadata.WithHeaders(headers)
	headers["X-Frozen"] = "later"
	sourceBefore := maps.Clone(client.MoreHeaders)
	var wait sync.WaitGroup
	for i := 0; i < 16; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			got, err := scope.Merge(context.Background(), input, option)
			if err != nil || got.Metadata["response"] != "owned" || got.StatusCode != 200 {
				t.Errorf("result=%+v err=%v", got, err)
				return
			}
			got.Metadata["response"] = "local"
			got.Body[0] = ' '
			got.Header.Set("X-Request-Id", "local")
		}()
	}
	wait.Wait()
	if calls.Load() != 16 || input["stable"] != "input" || !reflect.DeepEqual(sourceBefore, client.MoreHeaders) {
		t.Fatalf("calls=%d input=%v source=%v", calls.Load(), input, client.MoreHeaders)
	}
}

func TestMetadataUTF8AndSelectedOriginAreValidatedBeforeOptionsAndHTTP(t *testing.T) {
	var calls int
	var response = `{"metadata":{"unicode":"한글 é"}}`
	scope, client := testScope(t, "volumes", func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, _ := io.ReadAll(r.Body)
		if r.Method == "POST" && string(body) != `{"metadata":{"":"","한글":"é"}}` {
			t.Errorf("valid Unicode body=%s", body)
		}
		jsonResponse(w, response)
	})
	var applied int
	option := func(*request.Config[metadata.Opts]) error { applied++; return nil }
	for _, values := range []map[string]string{
		{string([]byte{0xff}): "key"}, {"value": string([]byte{0xff})},
		{string([]byte{0xfe}): "first", string([]byte{0xff}): "second"},
	} {
		for _, run := range []func() error{
			func() error { _, err := scope.Merge(context.Background(), values, option); return err },
			func() error { _, err := scope.Replace(context.Background(), values, option); return err },
		} {
			if err := run(); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || applied != 0 {
				t.Fatalf("values=%v err=%v calls=%d options=%d", values, err, calls, applied)
			}
		}
	}
	value, err := scope.Merge(context.Background(), map[string]string{"": "", "한글": "é"})
	if err != nil || value.Metadata["unicode"] != "한글 é" || calls != 1 {
		t.Fatalf("valid=%+v err=%v calls=%d", value, err, calls)
	}
	response = "{\"metadata\":{\"bad\":\"" + string([]byte{0xff}) + "\"}}"
	value, err = scope.Get(context.Background())
	var proof *resource.ResponseError
	if value != nil || !errors.As(err, &proof) || string(proof.Body) != response || proof.StatusCode != 200 || calls != 2 {
		t.Fatalf("invalid Unicode response=%+v err=%v proof=%+v", value, err, proof)
	}
	validBase := client.ResourceBase
	validEndpoint := client.Endpoint
	for _, base := range []string{"http://foreign.invalid/project/", validBase + "?query=true", validBase + "#fragment", "http://user@metadata.invalid/", "mailto:metadata"} {
		client.ResourceBase = base
		if err := Validate(context.Background(), client); !errors.Is(err, resource.ErrInvalidOption) || calls != 2 {
			t.Fatalf("base=%q err=%v calls=%d", base, err, calls)
		}
	}
	client.ResourceBase = validBase
	for _, endpoint := range []string{validEndpoint + "?query=true", validEndpoint + "#fragment", "ftp://metadata.invalid/"} {
		client.Endpoint = endpoint
		if err := Validate(context.Background(), client); !errors.Is(err, resource.ErrInvalidOption) || calls != 2 {
			t.Fatalf("endpoint=%q err=%v calls=%d", endpoint, err, calls)
		}
	}
	client.Endpoint = validEndpoint
	if err := Validate(context.Background(), client); err != nil {
		t.Fatal(err)
	}
}
