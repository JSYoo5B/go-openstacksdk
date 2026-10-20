package autoallocatedtopologies_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	tokens3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

type topoTransport func(*http.Request) (*http.Response, error)

func (f topoTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func topoWire(code int, raw string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(raw)), Header: http.Header{"X-Request-Id": {"topology-actual"}}}
}
func topoAPI(t *testing.T, reply topoTransport) (*autoallocatedtopologies.API, *gophercloud.ServiceClient) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = reply
	client := cloud.Client("network", "/neutron/v2.0/")
	return autoallocatedtopologies.New(client), client
}
func explicit() autoallocatedtopologies.Option {
	return autoallocatedtopologies.WithProjectID("project")
}
func TestAutoTopologyGetBindsProjectSeedAndActualNetworkIdentity(t *testing.T) {
	calls := 0
	raw := `{"auto_allocated_topology":{"id":"network","tenant_id":"owner","x":9007199254740993}}`
	api, _ := topoAPI(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != "GET" || req.URL.Path != "/neutron/v2.0/auto-allocated-topology/project" || req.URL.RawQuery != "" || req.Body != nil {
			t.Fatal(req.Method, req.URL, req.Body)
		}
		return topoWire(203, raw), nil
	})
	got, err := api.GetRecord(context.Background(), explicit())
	if err != nil || got.ProjectID != "project" || string(got.Resource.Body["id"]) != `"network"` || string(got.Resource.Body["project_id"]) != `"owner"` || string(got.Resource.Body["tenant_id"]) != `"owner"` || string(got.Envelope) != raw || calls != 1 {
		t.Fatal(got, err, calls)
	}
	got.Resource.Body["id"][1] = 'X'
	if string(got.Wire.Body["id"]) != `"network"` {
		t.Fatal(got)
	}
}
func TestAutoTopologyValidateFixedDryRunHasNoProjectIDSeedOrCreateCalls(t *testing.T) {
	for _, raw := range []string{`{"auto_allocated_topology":{"id":"dry-run=pass","tenant_id":"project"}}`, `{"auto_allocated_topology":{"dry_run":false,"project":"spoof"}}`, "", "not-json"} {
		t.Run(raw, func(t *testing.T) {
			calls := 0
			api, _ := topoAPI(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.URL.RawQuery != "fields=dry-run" || req.Body != nil {
					t.Fatal(req.Method, req.URL, req.Body)
				}
				return topoWire(200, raw), nil
			})
			got, err := api.ValidateRecord(context.Background(), explicit())
			if err != nil || got == nil || got.ProjectID != "project" || got.StatusCode != 200 || string(got.Envelope) != raw || calls != 1 {
				t.Fatal(got, err, calls)
			}
			if raw == "" || raw == "not-json" {
				if string(got.Resource.Body["id"]) != "null" || got.Wire != nil {
					t.Fatal("invented ID/Wire", got)
				}
			}
			if strings.Contains(raw, `"project":"spoof"`) && string(got.Resource.Body["project"]) != `"project"` {
				t.Fatal(got)
			}
		})
	}
}
func TestAutoTopologyDeleteDefaultStrict404AndExplicitIgnoreOwnActualProof(t *testing.T) {
	code, calls := 404, 0
	api, _ := topoAPI(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != "DELETE" || req.URL.RawQuery != "" || req.Body != nil {
			t.Fatal(req.Method, req.URL, req.Body)
		}
		return topoWire(code, `{"error":"missing"}`), nil
	})
	got, err := api.DeleteRecord(context.Background(), explicit())
	var native gophercloud.ErrUnexpectedResponseCode
	if got != nil || !errors.As(err, &native) || native.Actual != 404 {
		t.Fatal(got, err)
	}
	got, err = api.DeleteRecord(context.Background(), explicit(), autoallocatedtopologies.WithIgnoreMissing(true))
	if err != nil || got == nil || !got.IgnoredMissing || got.StatusCode != 404 || string(got.Envelope) != `{"error":"missing"}` {
		t.Fatal(got, err)
	}
	code = 204
	got, err = api.DeleteRecord(context.Background(), explicit())
	if err != nil || got.StatusCode != 204 || got.IgnoredMissing || calls != 3 {
		t.Fatal(got, err, calls)
	}
}
func TestAutoTopologyRecordedCurrentProjectAndMissingAuthNeverGuessRoute(t *testing.T) {
	calls := 0
	api, client := topoAPI(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.Path != "/neutron/v2.0/auto-allocated-topology/recorded" {
			t.Fatal(req.URL)
		}
		return topoWire(200, `{"auto_allocated_topology":{}}`), nil
	})
	_, err := api.GetRecord(context.Background())
	if !errors.Is(err, resource.ErrUnsupported) || calls != 0 {
		t.Fatal(err, calls)
	}
	var auth tokens3.CreateResult
	auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": "recorded"}}}
	auth.Header = http.Header{"X-Subject-Token": {"token"}}
	if err := client.SetTokenAndAuthResult(auth); err != nil {
		t.Fatal(err)
	}
	got, err := api.GetRecord(context.Background())
	if err != nil || got.ProjectID != "recorded" || calls != 1 {
		t.Fatal(got, err, calls)
	}
}
func TestAutoTopologyOptionsKeepHeadersVersionAndCapturedSource(t *testing.T) {
	calls, next := 0, 0
	api, client := topoAPI(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Probe") != "owned" || req.Header.Get("OpenStack-API-Version") != "network 2.9" {
			t.Fatal(req.Header)
		}
		return topoWire(200, `{"auto_allocated_topology":{}}`), nil
	})
	client.Microversion = "2.0"
	client.MoreHeaders = map[string]string{"OpenStack-API-Version": "network 2.0"}
	headers := map[string]string{"X-Probe": "owned"}
	option := autoallocatedtopologies.WithHeaders(headers)
	headers["X-Probe"] = "late"
	_, err := api.ValidateRecord(context.Background(), explicit(), option, autoallocatedtopologies.WithMicroversion("2.9"))
	if err != nil || calls != 1 || client.Microversion != "2.0" || client.MoreHeaders["OpenStack-API-Version"] != "network 2.0" {
		t.Fatal(err, calls, client)
	}
	change := func(*autoallocatedtopologies.Options) error {
		client.ResourceBase = "https://foreign.invalid/"
		return nil
	}
	restore := func(*autoallocatedtopologies.Options) error { next++; client.ResourceBase = ""; return nil }
	_, err = api.GetRecord(context.Background(), explicit(), change, restore)
	if !errors.Is(err, resource.ErrInvalidOption) || next != 0 || calls != 1 {
		t.Fatal(err, next, calls)
	}
}
func TestAutoTopologyInvalidOptionsAndAcceptedDecodeFailuresAreAtomic(t *testing.T) {
	calls := 0
	api, _ := topoAPI(t, func(*http.Request) (*http.Response, error) {
		calls++
		return topoWire(203, `{"auto_allocated_topology":null}`), nil
	})
	for _, option := range []autoallocatedtopologies.Option{nil, autoallocatedtopologies.WithProjectID(""), autoallocatedtopologies.WithIgnoreMissing(true), autoallocatedtopologies.WithHeaders(map[string]string{"X-Auth-Token": "replace"})} {
		_, err := api.GetRecord(context.Background(), explicit(), option)
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	got, err := api.ValidateRecord(context.Background(), explicit())
	var proof *resource.ResponseError
	if got == nil || got.Resource != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != `{"auto_allocated_topology":null}` || calls != 1 {
		t.Fatal(got, err, proof, calls)
	}
}

type topoBody struct {
	io.Reader
	fault  error
	closed int
}

func (b *topoBody) Close() error { b.closed++; return b.fault }

type topoReader func([]byte) (int, error)

func (f topoReader) Read(p []byte) (int, error) { return f(p) }
func TestAutoTopologyAllThreeOperationsFaultsPreserve401NativeFieldsAndZeroHooks(t *testing.T) {
	for _, operation := range []string{"get", "validate", "delete"} {
		for _, code := range []int{200, 401, 404, 503} {
			t.Run(fmt.Sprint(operation, code), func(t *testing.T) {
				raw := `{"auto_allocated_topology":{"id":"actual"}}`
				readCause, closeCause := errors.New("topology read"), errors.New("topology close")
				done := false
				body := &topoBody{fault: closeCause}
				body.Reader = topoReader(func(p []byte) (int, error) {
					if done {
						return 0, io.EOF
					}
					done = true
					return copy(p, raw), readCause
				})
				calls, retries, reauth := 0, 0, 0
				target, method := "", ""
				api, client := topoAPI(t, func(req *http.Request) (*http.Response, error) {
					calls++
					target = req.URL.String()
					method = req.Method
					r := topoWire(code, raw)
					r.Body = body
					return r, nil
				})
				client.ReauthFunc = func(context.Context) error { reauth++; return nil }
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				var err error
				switch operation {
				case "get":
					_, err = api.GetRecord(context.Background(), explicit())
				case "validate":
					_, err = api.ValidateRecord(context.Background(), explicit())
				case "delete":
					_, err = api.DeleteRecord(context.Background(), explicit(), autoallocatedtopologies.WithIgnoreMissing(true))
				}
				if !errors.Is(err, readCause) || !errors.Is(err, closeCause) || calls != 1 || body.closed != 1 || retries != 0 || reauth != 0 {
					t.Fatal(err, calls, body.closed, retries, reauth)
				}
				if code == 401 {
					var auth *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &auth) || !errors.Is(auth.ErrReauth, readCause) || !errors.Is(auth.ErrReauth, closeCause) {
						t.Fatal(err, auth)
					}
					var actual gophercloud.ErrUnexpectedResponseCode
					if !errors.As(auth.ErrOriginal, &actual) || actual.Actual != 401 || actual.Method != method || actual.URL != target || string(actual.Body) != raw || actual.ResponseHeader.Get("X-Request-Id") != "topology-actual" {
						t.Fatal(err, actual)
					}
				}
			})
		}
	}
}
func TestAutoTopologyCleanRetryAndReauthKeepDryRunAndDeletionMissingPolicy(t *testing.T) {
	for _, code := range []int{401, 503} {
		calls := 0
		api, client := topoAPI(t, func(req *http.Request) (*http.Response, error) {
			calls++
			if req.URL.RawQuery != "fields=dry-run" || req.Method != "GET" {
				t.Fatal(req.Method, req.URL)
			}
			if calls == 1 {
				return topoWire(code, `{"error":"retry"}`), nil
			}
			return topoWire(200, `{"auto_allocated_topology":{"dry_run":true}}`), nil
		})
		client.ReauthFunc = func(context.Context) error { client.SetToken("renewed"); return nil }
		client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return nil }
		got, err := api.ValidateRecord(context.Background(), explicit())
		if err != nil || calls != 2 || string(got.Resource.Body["dry_run"]) != "true" {
			t.Fatal(got, err, calls)
		}
	}
}
func TestAutoTopologyResponseAliasNullPrecedenceAndNoCache(t *testing.T) {
	calls := 0
	api, _ := topoAPI(t, func(*http.Request) (*http.Response, error) {
		calls++
		return topoWire(200, `{"auto_allocated_topology":{"project_id":null,"tenant_id":"legacy","dry_run":[false]}}`), nil
	})
	for range 2 {
		got, err := api.GetRecord(context.Background(), explicit())
		if err != nil || string(got.Resource.Body["project_id"]) != "null" || string(got.Resource.Body["dry_run"]) != `[false]` {
			t.Fatal(got, err)
		}
	}
	if calls != 2 {
		t.Fatal(calls)
	}
}
func TestAutoTopologyCancellationRetainsCauseBeforeAnyAuthLookup(t *testing.T) {
	calls := 0
	api, _ := topoAPI(t, func(*http.Request) (*http.Response, error) { calls++; return topoWire(200, `{}`), nil })
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("stop topology")
	cancel(cause)
	_, err := api.ValidateRecord(ctx)
	if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatal(err, calls)
	}
}
