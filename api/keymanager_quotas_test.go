package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/testhelper"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/keymanager/v1/quotas"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

const quotaPrefix = "/reverse/barbican/v1"

func quotaScope(t *testing.T, client *gophercloud.ServiceClient, id string) *quotas.ProjectScope {
	t.Helper()
	scope, err := quotas.New(client).InProject(context.Background(), resource.ID(id))
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func TestKeyManagerQuotasFourOperationsKeepFixedProjectAndLiveProvider(t *testing.T) {
	cloud := testcloud.New(t)
	const projectID = "project-α+one"
	client := cloud.Client("key-manager", "/catalog/v1")
	client.ResourceBase = cloud.Server.URL + quotaPrefix + "/"
	client.MoreHeaders = map[string]string{"X-Client": "configured"}
	client.Microversion = "1.9"
	var operations []string
	cloud.Mux.HandleFunc(quotaPrefix+"/quotas", func(w http.ResponseWriter, r *http.Request) {
		operations = append(operations, r.Method+" effective")
		if r.Method != "GET" || r.Header.Get("X-Auth-Token") != "test-token" || r.URL.RawQuery != "" {
			t.Error(r.Method, r.Header, r.URL)
		}
		w.Header().Set("X-Request-ID", "effective")
		testcloud.JSON(w, 200, `{"quotas":{"secrets":9007199254740993,"orders":null,"containers":-1,"consumers":0,"cas":7,"plugin":{"n":9007199254740995},"Secrets":"shadow"}}`)
	})
	cloud.Mux.HandleFunc(quotaPrefix+"/project-quotas/"+projectID, func(w http.ResponseWriter, r *http.Request) {
		operations = append(operations, r.Method+" configured")
		if r.URL.EscapedPath() != quotaPrefix+"/project-quotas/"+url.PathEscape(projectID) || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "rotated-token" || r.Header.Get("X-Client") != "configured" {
			t.Error(r.URL, r.Header)
		}
		switch r.Method {
		case "GET":
			testcloud.JSON(w, 200, `{"project_quotas":{"secrets":null,"project_id":"other-project","orders":2}}`)
		case "PUT":
			var body map[string]map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if len(body) != 1 || !reflect.DeepEqual(body["project_quotas"], map[string]json.RawMessage{"secrets": json.RawMessage(`0`), "orders": json.RawMessage(`-1`)}) {
				t.Error(body)
			}
			w.Header().Set("X-Request-ID", "updated")
			w.WriteHeader(204)
		case "DELETE":
			if data, err := io.ReadAll(r.Body); err != nil || len(data) != 0 {
				t.Error(string(data), err)
			}
			w.WriteHeader(204)
		default:
			t.Error("unexpected automatic operation", r.Method)
			w.WriteHeader(500)
		}
	})
	a := quotas.New(client)
	scope, err := a.InProject(context.Background(), resource.ID(projectID))
	if err != nil || len(operations) != 0 || scope.ProjectID() != projectID || scope.RawClient() != client || a.RawClient() != client {
		t.Fatal(scope, err, operations)
	}
	value, err := a.Get(context.Background(), quotas.WithGetHeader("X-Trace", "read"))
	if err != nil || string(value.Secrets) != "9007199254740993" || string(value.Orders) != "null" || string(value.Consumers) != "0" || value.StatusCode != 200 || value.Header.Get("X-Request-ID") != "effective" || string(value.Data["Secrets"]) != `"shadow"` || string(value.Data["plugin"]) != `{"n":9007199254740995}` {
		t.Fatal(value, err)
	}
	value.Secrets[0] = '8'
	value.Data["plugin"][0] = '['
	if string(value.Body["secrets"]) != "9007199254740993" || string(value.Body["plugin"]) != `{"n":9007199254740995}` {
		t.Fatal("response views alias", value)
	}
	cloud.Provider.SetToken("rotated-token")
	configured, err := scope.Get(context.Background())
	if err != nil || string(configured.Secrets) != "null" || configured.Containers != nil || string(configured.Data["project_id"]) != `"other-project"` {
		t.Fatal(configured, err)
	}
	result, err := scope.Update(context.Background(), quotas.UpdateOpts{}, quotas.WithUpdateSecrets(0), quotas.WithUpdateOrders(-1))
	if err != nil || result.StatusCode != 204 || len(result.Body) != 0 || result.Header.Get("X-Request-ID") != "updated" {
		t.Fatal(result, err)
	}
	if err := scope.Delete(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(operations, []string{"GET effective", "GET configured", "PUT configured", "DELETE configured"}) || client.ResourceBase != cloud.Server.URL+quotaPrefix+"/" || client.Endpoint != cloud.Server.URL+"/catalog/v1/" || client.ProviderClient != cloud.Provider || client.Microversion != "1.9" {
		t.Fatal(operations, client)
	}
}

func TestKeyManagerQuotasReplacementPresenceAndOwnedOptions(t *testing.T) {
	cloud := testcloud.New(t)
	var bodies []map[string]json.RawMessage
	cloud.Mux.HandleFunc(quotaPrefix+"/project-quotas/project", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Error("read-modify-write request", r.Method)
			w.WriteHeader(500)
			return
		}
		var body map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body) != 1 || body["project_quotas"] == nil {
			t.Error(body)
		}
		bodies = append(bodies, body["project_quotas"])
		w.WriteHeader(204)
	})
	scope := quotaScope(t, cloud.Client("key-manager", quotaPrefix), "project")
	options := quotas.UpdateOpts{Secrets: request.Present(int64(0)), Orders: request.Present(int64(-1)), Containers: request.Present(int64(9007199254740993)), Consumers: request.Present(int64(-2)), CAs: request.Present(int64(math.MaxInt64))}
	bulk := quotas.WithUpdateOptions(options)
	options.Secrets = request.Present(int64(99))
	for _, option := range [][]quotas.UpdateOption{{}, {bulk}, {bulk, quotas.WithUpdateSecrets(5)}, {quotas.WithUpdateSecrets(5), bulk}, {bulk, quotas.WithUpdateOptions(quotas.UpdateOpts{})}} {
		if _, err := scope.Update(context.Background(), quotas.UpdateOpts{}, option...); err != nil {
			t.Fatal(err)
		}
	}
	want := []map[string]json.RawMessage{
		{},
		{"secrets": json.RawMessage(`0`), "orders": json.RawMessage(`-1`), "containers": json.RawMessage(`9007199254740993`), "consumers": json.RawMessage(`-2`), "cas": json.RawMessage(`9223372036854775807`)},
		{"secrets": json.RawMessage(`5`), "orders": json.RawMessage(`-1`), "containers": json.RawMessage(`9007199254740993`), "consumers": json.RawMessage(`-2`), "cas": json.RawMessage(`9223372036854775807`)},
		{"secrets": json.RawMessage(`0`), "orders": json.RawMessage(`-1`), "containers": json.RawMessage(`9007199254740993`), "consumers": json.RawMessage(`-2`), "cas": json.RawMessage(`9223372036854775807`)},
		{},
	}
	if !reflect.DeepEqual(bodies, want) {
		t.Fatal(bodies, want)
	}
	for _, raw := range []string{`{"secrets":false}`, `{"orders":1.5}`, `{"cas":9223372036854775808}`, `{"containers":"3"}`} {
		var opts quotas.UpdateOpts
		if err := json.Unmarshal([]byte(raw), &opts); err == nil {
			t.Fatal("invalid typed integer accepted", raw)
		}
	}
}

func TestKeyManagerQuotasInvalidOptionsAndScopePreflight(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	client := cloud.Client("key-manager", quotaPrefix)
	scope := quotaScope(t, client, "project")
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("effective quota request canceled")
	cancel(cause)
	if value, err := quotas.New(client).Get(ctx); value != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal("quota preflight lost cancellation cause", value, err)
	}
	for _, field := range []quotas.UpdateOpts{{Secrets: request.Null[int64]()}, {Orders: request.Null[int64]()}, {Containers: request.Null[int64]()}, {Consumers: request.Null[int64]()}, {CAs: request.Null[int64]()}} {
		if _, err := scope.Update(context.Background(), field); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(field, err)
		}
	}
	for _, option := range []quotas.UpdateOption{
		request.WithField[quotas.UpdateOpts]("unknown", 1), request.WithField[quotas.UpdateOpts]("secrets", 1), request.WithField[quotas.UpdateOpts]("Secrets", 1), request.WithField[quotas.UpdateOpts]("project_quotas", map[string]int{}),
		request.WithQuery[quotas.UpdateOpts]("project_id", "other"), request.WithArgument[quotas.UpdateOpts]("parent", "other"), nil,
		quotas.WithUpdateHeader("x-auth-token", "other"), quotas.WithUpdateHeader("Content-Type", "text/plain"), quotas.WithUpdateHeader("OpenStack-API-Version", "key-manager latest"), quotas.WithUpdateHeader("bad:key", "value"), quotas.WithUpdateHeader("X-Test", "line\nbreak"),
	} {
		if _, err := scope.Update(context.Background(), quotas.UpdateOpts{}, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := scope.Get(context.Background(), request.WithField[quotas.GetOpts]("secrets", 1)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if err := scope.Delete(context.Background(), request.WithQuery[quotas.DeleteOpts]("ignore_missing", "true")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, ref := range []resource.Ref{resource.Ref{}, resource.ID(".."), resource.ID("has/slash"), resource.ID("percent%20"), resource.ID("nul\x00"), resource.ID(string([]byte{0xff})), resource.Name("project")} {
		if _, err := quotas.New(client).InProject(context.Background(), ref); err == nil {
			t.Fatal("invalid/unsupported scope accepted", ref)
		}
	}
	for _, bad := range []*gophercloud.ServiceClient{nil, {Type: "key-manager"}, {ProviderClient: cloud.Provider, Endpoint: client.Endpoint, Type: "compute"}} {
		if _, err := quotas.New(bad).Get(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	var a *quotas.API
	var s *quotas.ProjectScope
	if _, err := a.Get(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := s.Get(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := s.Update(context.Background(), quotas.UpdateOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if err := s.Delete(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := (&quotas.ProjectScope{}).Get(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	client.MoreHeaders = map[string]string{"X-Auth-Token": "fixed-stale-token"}
	if _, err := scope.Get(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("preflight submitted HTTP", calls.Load())
	}
}

func TestKeyManagerQuotasMissingDeleteAndNativeFailures(t *testing.T) {
	for _, code := range []int{404, 403, 409, 503, 200} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(quotaPrefix+"/project-quotas/project", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "DELETE" {
					t.Error(r.Method)
				}
				w.Header().Set("X-Request-ID", "failed")
				w.WriteHeader(code)
				_, _ = w.Write([]byte("native-failure"))
			})
			scope := quotaScope(t, cloud.Client("key-manager", quotaPrefix), "project")
			err := scope.Delete(context.Background())
			if code == 404 {
				if err != nil {
					t.Fatal(err)
				}
			} else if !gophercloud.ResponseCodeIs(err, code) {
				t.Fatal(err)
			}
			err = scope.Delete(context.Background(), quotas.WithDeleteIgnoreMissing(false))
			if !gophercloud.ResponseCodeIs(err, code) {
				t.Fatal("strict/native cause lost", err)
			}
			if calls.Load() != 2 {
				t.Fatal("failure resent", calls.Load())
			}
		})
	}
	for _, accepted := range []bool{false, true} {
		t.Run(fmt.Sprintf("wrapped404_accepted_%t", accepted), func(t *testing.T) {
			var calls atomic.Int32
			client := &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{}, Type: "key-manager", Endpoint: "https://barbican.invalid/v1/"}
			native := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{204}, Method: "DELETE", Body: []byte("not-an-http-404-response")}
			client.ProviderClient.HTTPClient.Transport = quotaRoundTrip(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if !accepted {
					return nil, native
				}
				return &http.Response{StatusCode: 204, Header: http.Header{"X-Request-Id": {"accepted-delete"}}, Body: io.NopCloser(io.MultiReader(strings.NewReader("accepted-prefix"), quotaErrorReader{native})), Request: r}, nil
			})
			err := quotaScope(t, client, "project").Delete(context.Background())
			if err == nil || !gophercloud.ResponseCodeIs(err, 404) || calls.Load() != 1 {
				t.Fatal("wrapped failure ignored or resent", err, calls.Load())
			}
			if accepted {
				var evidence *resource.ResponseError
				if !errors.As(err, &evidence) || evidence.StatusCode != 204 || string(evidence.Body) != "accepted-prefix" || evidence.Header.Get("X-Request-ID") != "accepted-delete" {
					t.Fatal("accepted DELETE evidence lost", err, evidence)
				}
			} else {
				var transport *url.Error
				if !errors.As(err, &transport) {
					t.Fatal("transport cause lost", err)
				}
			}
		})
	}
}

type quotaErrorReader struct{ err error }

func (r quotaErrorReader) Read([]byte) (int, error) { return 0, r.err }

func TestKeyManagerQuotasStrictGetEnvelopesStatusAndReadEvidence(t *testing.T) {
	for _, mode := range []string{"effective-wrong", "project-wrong", "missing", "null", "array", "broken", "status", "not-found", "forbidden"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			code := 200
			body := `{"orders":1}`
			wantPath := quotaPrefix + "/quotas"
			switch mode {
			case "effective-wrong":
				body = `{"project_quotas":{"orders":1}}`
			case "project-wrong":
				body = `{"quotas":{"orders":1}}`
				wantPath = quotaPrefix + "/project-quotas/project"
			case "missing":
				body = `{}`
			case "null":
				body = `{"quotas":null}`
			case "array":
				body = `{"quotas":[]}`
			case "broken":
				body = `{"quotas":`
			case "status":
				code = 203
				body = `{"quotas":{}}`
			case "not-found":
				code = 404
				body = `{"message":"effective quota not found"}`
			case "forbidden":
				code = 403
				body = `{"message":"effective quota denied"}`
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testhelper.TestMethod(t, r, http.MethodGet)
				if r.URL.Path != wantPath || r.URL.RawQuery != "" {
					t.Error("quota GET resolved another path or sent a query", r.Method, r.URL)
				}
				if data, err := io.ReadAll(r.Body); err != nil || len(data) != 0 {
					t.Error("quota GET sent a body", string(data), err)
				}
				w.Header().Set("X-Request-ID", mode)
				testcloud.JSON(w, code, body)
			})
			client := cloud.Client("key-manager", quotaPrefix)
			var value *quotas.Quota
			var err error
			if mode == "project-wrong" {
				value, err = quotaScope(t, client, "project").Get(context.Background())
			} else {
				value, err = quotas.New(client).Get(context.Background())
			}
			if value != nil || err == nil || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
			var operation *resource.OperationError
			wantKind := "quotas"
			if mode == "project-wrong" {
				wantKind = "project_quotas"
			}
			if !errors.As(err, &operation) || operation.Operation != "Get" || operation.Resource != wantKind || operation.Cause == nil {
				t.Fatal("quota GET operation/cause was discarded", err, operation)
			}
			if mode == "status" || mode == "not-found" || mode == "forbidden" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(operation.Cause, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || native.Method != http.MethodGet || native.URL != cloud.Server.URL+wantPath || string(native.Body) != body || native.ResponseHeader.Get("X-Request-ID") != mode || !gophercloud.ResponseCodeIs(err, code) {
					t.Fatal("native quota GET status/body/header/cause was discarded", err, native)
				}
				var accepted *resource.ResponseError
				if errors.As(err, &accepted) {
					t.Fatal("native rejection became accepted response evidence", err, accepted)
				}
			} else {
				var evidence *resource.ResponseError
				if !errors.As(err, &evidence) || evidence.StatusCode != 200 || evidence.Header.Get("X-Request-ID") != mode || string(evidence.Body) != body {
					t.Fatal("accepted evidence lost", err, evidence)
				}
			}
		})
	}
	client := &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{}, Type: "key-manager", Endpoint: "https://barbican.invalid/v1/"}
	var reads atomic.Int32
	client.ProviderClient.HTTPClient.Transport = quotaRoundTrip(func(r *http.Request) (*http.Response, error) {
		reads.Add(1)
		return &http.Response{StatusCode: 200, Header: http.Header{"X-Request-Id": {"partial"}}, Body: &quotaPartialReader{}, Request: r}, nil
	})
	if value, err := quotas.New(client).Get(context.Background()); value != nil || !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatal(value, err)
	} else {
		var evidence *resource.ResponseError
		if !errors.As(err, &evidence) || string(evidence.Body) != `{"quotas":` || evidence.StatusCode != 200 || evidence.Header.Get("X-Request-ID") != "partial" {
			t.Fatal(evidence, err)
		}
	}
	if reads.Load() != 1 {
		t.Fatal("read failure resent", reads.Load())
	}
	t.Run("accepted Close keeps complete response and cause", func(t *testing.T) {
		cloud := testcloud.New(t)
		closeCause := errors.New("effective quota accepted Close failed")
		track := payloadContractTrack(cloud, nil, closeCause)
		var calls, retries atomic.Int32
		cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			retries.Add(1)
			return err
		}
		const wantBody = `{"quotas":{"secrets":9007199254740993,"orders":null}}`
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			testhelper.TestMethod(t, r, http.MethodGet)
			if r.URL.Path != quotaPrefix+"/quotas" || r.URL.RawQuery != "" {
				t.Error("accepted Close changed effective quota route", r.Method, r.URL)
			}
			if data, err := io.ReadAll(r.Body); err != nil || len(data) != 0 {
				t.Error("quota GET sent a body", string(data), err)
			}
			w.Header().Set("X-Request-ID", "accepted-close")
			testcloud.JSON(w, 200, wantBody)
		})
		client := cloud.Client("key-manager", "/catalog/v1")
		client.ResourceBase = cloud.Server.URL + quotaPrefix + "/"
		value, err := quotas.New(client).Get(context.Background())
		var operation *resource.OperationError
		var evidence *resource.ResponseError
		if value != nil || !errors.Is(err, closeCause) || !errors.As(err, &operation) || operation.Operation != "Get" || operation.Resource != "quotas" || !errors.Is(operation.Cause, closeCause) || !errors.As(err, &evidence) || evidence.StatusCode != 200 || evidence.Header.Get("X-Request-ID") != "accepted-close" || string(evidence.Body) != wantBody {
			t.Fatal("accepted quota GET Close evidence/cause was discarded", value, err, operation, evidence)
		}
		body := track.last(t)
		if calls.Load() != 1 || track.calls.Load() != 1 || retries.Load() != 0 || body.reads.Load() == 0 || body.closes.Load() != 1 {
			t.Fatal("accepted quota GET Close failure was retried or closed twice", calls.Load(), track.calls.Load(), retries.Load(), body.reads.Load(), body.closes.Load())
		}
	})
}

func TestKeyManagerQuotasUpdateAcknowledgementAndTerminalFailures(t *testing.T) {
	for _, code := range []int{200, 202, 400, 403, 404, 409, 503} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "PUT" {
					t.Error(r.Method)
				}
				testcloud.JSON(w, code, `{"project_quotas":{"secrets":9}}`)
			})
			result, err := quotaScope(t, cloud.Client("key-manager", quotaPrefix), "project").Update(context.Background(), quotas.UpdateOpts{}, quotas.WithUpdateSecrets(1))
			if result != nil || !gophercloud.ResponseCodeIs(err, code) || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
	var calls atomic.Int32
	client := &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{}, Type: "key-manager", Endpoint: "https://barbican.invalid/v1/"}
	client.ProviderClient.HTTPClient.Transport = quotaRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		return &http.Response{StatusCode: 204, Header: http.Header{"X-Request-Id": {"accepted"}}, Body: io.NopCloser(strings.NewReader("opaque-ack")), Request: r}, nil
	})
	result, err := quotaScope(t, client, "project").Update(context.Background(), quotas.UpdateOpts{})
	if err != nil || result.StatusCode != 204 || string(result.Body) != "opaque-ack" || result.Header.Get("X-Request-ID") != "accepted" || calls.Load() != 1 {
		t.Fatal(result, err, calls.Load())
	}
	client.ProviderClient.HTTPClient.Transport = quotaRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		return &http.Response{StatusCode: 204, Header: http.Header{"X-Request-Id": {"partial"}}, Body: &quotaPartialReader{}, Request: r}, nil
	})
	result, err = quotaScope(t, client, "project").Update(context.Background(), quotas.UpdateOpts{})
	var evidence *resource.ResponseError
	if result != nil || !errors.Is(err, io.ErrUnexpectedEOF) || !errors.As(err, &evidence) || evidence.StatusCode != 204 || string(evidence.Body) != `{"quotas":` || calls.Load() != 2 {
		t.Fatal(result, err, evidence, calls.Load())
	}
	sentinel := errors.New("transport failed")
	client.ProviderClient.HTTPClient.Transport = quotaRoundTrip(func(r *http.Request) (*http.Response, error) { calls.Add(1); return nil, sentinel })
	if result, err := quotaScope(t, client, "project").Update(context.Background(), quotas.UpdateOpts{}); result != nil || !errors.Is(err, sentinel) || calls.Load() != 3 {
		t.Fatal(result, err, calls.Load())
	}
}

func TestKeyManagerQuotasHeadersSnapshotReuseAndCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc(quotaPrefix+"/project-quotas/project", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body map[string]map[string]int64
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(body, map[string]map[string]int64{"project_quotas": {"cas": 4, "consumers": 0}}) || r.Header.Get("X-Trace") != "last" {
			t.Error(body, r.Header)
		}
		w.WriteHeader(204)
	})
	client := cloud.Client("key-manager", quotaPrefix)
	scope := quotaScope(t, client, "project")
	opts := []quotas.UpdateOption{quotas.WithUpdateCAs(4), quotas.WithUpdateConsumers(0), quotas.WithUpdateHeader("x-trace", "first"), quotas.WithUpdateHeader("X-Trace", "last")}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := scope.Update(context.Background(), quotas.UpdateOpts{}, opts...); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 4 {
		t.Fatal(calls.Load())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := scope.Get(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := scope.Update(ctx, quotas.UpdateOpts{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := scope.Delete(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := quotas.New(client).InProject(ctx, resource.ID("project")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := scope.Get(nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := scope.Update(context.Background(), quotas.UpdateOpts{}, func(c *request.Config[quotas.UpdateOpts]) error { client.Type = "compute"; return nil }); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls.Load() != 4 {
		t.Fatal("canceled/changed-source operation submitted HTTP", calls.Load())
	}
}

type quotaRoundTrip func(*http.Request) (*http.Response, error)

func (f quotaRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type quotaPartialReader struct{ read bool }

func (r *quotaPartialReader) Read(p []byte) (int, error) {
	if r.read {
		return 0, io.ErrUnexpectedEOF
	}
	r.read = true
	return copy(p, `{"quotas":`), nil
}
func (*quotaPartialReader) Close() error { return nil }
