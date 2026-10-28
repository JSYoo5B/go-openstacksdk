package volumes_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func summaryFixture(t *testing.T, version string, reply func(*http.Request) *http.Response) (*volumes.API, *testcloud.Recorder) {
	t.Helper()
	a, calls := pythonVolumeAPI(t, reply)
	a.RawClient().Microversion = version
	return a, calls
}
func summaryField(t *testing.T, value *volumes.VolumeSummaryRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatal(key, want, value)
	}
}
func summaryWrapper(t *testing.T, err error) {
	t.Helper()
	var op *resource.OperationError
	if !errors.As(err, &op) || op.Operation != "SummaryRecord" || op.Resource != "volume-summary" {
		t.Fatal(err, op)
	}
}
func TestPythonVolumeSummaryDefaultTwoFetchesAndAllProjects(t *testing.T) {
	for _, all := range []bool{false, true} {
		body := `{"volume-summary":{"total_size":900719925474099312345,"total_count":false,"metadata":[null,false,0],"extra":"wire"}}`
		api, calls := summaryFixture(t, "3.36", func(r *http.Request) *http.Response {
			if r.Header.Get("OpenStack-API-Version") != "volume 3.36" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.36" || r.Header.Get("X-Auth-Token") != "test-token" {
				t.Fatal(r.Header)
			}
			response := pythonVolumeWire(203, body)
			response.Header.Set("X-Request-Id", "actual")
			return response
		})
		v, err := api.SummaryRecord(context.Background(), volumes.WithSummaryAllProjects(all))
		query := "all_tenants=False"
		if all {
			query = "all_tenants=True"
		}
		if err != nil || calls.Len() != 2 || pythonVolumeCalls(calls)[0].path != pythonVolumeBase+"/summary" || pythonVolumeCalls(calls)[0].query != "" || pythonVolumeCalls(calls)[1].query != query || pythonVolumeCalls(calls)[1].body != "" || v.StatusCode != 203 || len(v.Requests) != 2 || len(v.Discovery) != 0 || v.Microversion != "3.36" || string(v.Envelope) != body {
			t.Fatal(v, err, pythonVolumeCalls(calls))
		}
		for key, want := range map[string]string{"total_size": "900719925474099312345", "total_count": "false", "metadata": "[null,false,0]", "id": "null", "name": "null", "location": "null"} {
			summaryField(t, v, key, want)
		}
		if len(v.Resource.Body) != 6 || v.Resource.Body["extra"] != nil || string(v.Wire.Body["extra"]) != `"wire"` {
			t.Fatal(v)
		}
		v.Resource.Body["metadata"][0] = 'X'
		v.Header.Set("X-Request-Id", "changed")
		v.Requests[0].Envelope[0] = 'X'
		if string(v.Wire.Body["metadata"]) != "[null,false,0]" || v.Wire.Header.Get("X-Request-Id") != "actual" || v.Resource.Header.Get("X-Request-Id") != "actual" || v.Envelope[0] != '{' || v.Requests[1].Envelope[0] != '{' {
			t.Fatal("owned channels alias", v)
		}
	}
}
func TestPythonVolumeSummaryRetainsFirstFetchDeclaredSeedOnly(t *testing.T) {
	count := 0
	api, calls := summaryFixture(t, "3.36", func(*http.Request) *http.Response {
		count++
		if count == 1 {
			return pythonVolumeWire(200, `{"id":"seed","total_size":4,"metadata":false,"unknown":"discard"}`)
		}
		return pythonVolumeWire(200, `{"volume-summary":{"total_count":null,"metadata":null}}`)
	})
	v, err := api.SummaryRecord(context.Background())
	if err != nil || calls.Len() != 2 || v.Wire.Body["id"] != nil || len(v.Wire.Body) != 2 {
		t.Fatal(v, err, pythonVolumeCalls(calls))
	}
	summaryField(t, v, "id", `"seed"`)
	summaryField(t, v, "total_size", "4")
	summaryField(t, v, "metadata", "null")
	summaryField(t, v, "total_count", "null")
}
func TestPythonVolumeSummarySingleFetchAndFetchSpecificVersion(t *testing.T) {
	for _, single := range []bool{false, true} {
		for _, version := range []string{"3.90", ""} {
			var headers []http.Header
			api, calls := summaryFixture(t, "3.36", func(r *http.Request) *http.Response {
				headers = append(headers, r.Header.Clone())
				return pythonVolumeWire(200, `{"volume-summary":{"total_count":2}}`)
			})
			v, err := api.SummaryRecord(context.Background(), volumes.WithSummaryMicroversion(version), volumes.WithSummarySingleFetch(single))
			n := 2
			if single {
				n = 1
			}
			if err != nil || calls.Len() != n || v.Microversion != version || api.RawClient().Microversion != "3.36" || len(v.Discovery) != 0 {
				t.Fatal(v, err, pythonVolumeCalls(calls))
			}
			if !single && headers[0].Get("OpenStack-API-Version") != "volume 3.36" {
				t.Fatal(headers)
			}
			expected := ""
			if version != "" {
				expected = "volume " + version
			}
			if headers[n-1].Get("OpenStack-API-Version") != expected || headers[n-1].Get("X-OpenStack-Volume-API-Version") != version {
				t.Fatal(headers)
			}
		}
	}
}
func TestPythonVolumeSummaryDiscoveryCeilingAndSelectedVersionNotClamped(t *testing.T) {
	for _, selected := range []string{"", "3.90"} {
		api, calls := summaryFixture(t, selected, func(r *http.Request) *http.Response {
			if r.URL.Path == "/catalog/unused/" {
				if r.Header.Get("OpenStack-API-Version") != "" {
					t.Fatal(r.Header)
				}
				return pythonVolumeWire(200, `{"version":{"id":"v3.0","min_version":"3.0","version":"3.71"}}`)
			}
			return pythonVolumeWire(200, `{"volume-summary":{"total_count":1}}`)
		})
		v, err := api.SummaryRecord(context.Background())
		expected := selected
		n := 2
		d := 0
		if selected == "" {
			expected = "3.36"
			n = 3
			d = 1
		}
		if err != nil || v.Microversion != expected || calls.Len() != n || len(v.Discovery) != d || len(v.Requests) != 2 || v.StatusCode != 200 {
			t.Fatal(v, err, pythonVolumeCalls(calls))
		}
		if d != 0 && string(v.Discovery[0].Envelope) != `{"version":{"id":"v3.0","min_version":"3.0","version":"3.71"}}` {
			t.Fatal(v.Discovery)
		}
	}
	api, calls := summaryFixture(t, "3.11", func(*http.Request) *http.Response { return pythonVolumeWire(404, `{"version":"summary unavailable"}`) })
	v, err := api.SummaryRecord(context.Background())
	if v == nil || v.StatusCode != 404 || len(v.Requests) != 1 || calls.Len() != 1 || !gophercloud.ResponseCodeIs(err, 404) || errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(v, err, pythonVolumeCalls(calls))
	}
}
func TestPythonVolumeSummaryAcceptedNonJSONAndFlatEnvelopeFallback(t *testing.T) {
	for _, body := range []string{"", "not-json", `{"total_size":null,"metadata":false}`, `{"other":{"total_size":4}}`} {
		api, calls := summaryFixture(t, "3.36", func(*http.Request) *http.Response { return pythonVolumeWire(203, body) })
		v, err := api.SummaryRecord(context.Background(), volumes.WithSummarySingleFetch(true))
		if err != nil || v == nil || v.Resource == nil || v.StatusCode != 203 || calls.Len() != 1 || string(v.Envelope) != body {
			t.Fatal(v, err, pythonVolumeCalls(calls))
		}
		if body == `{"total_size":null,"metadata":false}` {
			summaryField(t, v, "metadata", "false")
		} else {
			summaryField(t, v, "metadata", "null")
		}
	}
	for _, body := range []string{`null`, `[]`, `{"volume-summary":null}`, `{"volume-summary":[]}`, string([]byte{'{', 0xff, '}'})} {
		api, calls := summaryFixture(t, "3.36", func(*http.Request) *http.Response { return pythonVolumeWire(203, body) })
		v, err := api.SummaryRecord(context.Background(), volumes.WithSummarySingleFetch(true))
		var proof *resource.ResponseError
		if err == nil || v == nil || v.Resource != nil || v.StatusCode != 203 || string(v.Envelope) != body || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || calls.Len() != 1 {
			t.Fatal(v, err, proof, pythonVolumeCalls(calls))
		}
	}
}

type summaryMarshaler struct {
	called *int
	body   string
}

func (v summaryMarshaler) MarshalJSON() ([]byte, error) { *v.called++; return []byte(v.body), nil }
func TestPythonVolumeSummaryInvalidControlsBeforeMarshalerOrHTTP(t *testing.T) {
	for _, key := range []string{"all_tenants", "all_projects", "base_path", "microversion", "session", "requires_id", "resource_response_key"} {
		calls := 0
		api, wire := summaryFixture(t, "3.36", func(*http.Request) *http.Response { t.Fatal("unexpected HTTP"); return nil })
		_, err := api.SummaryRecord(context.Background(), volumes.WithSummaryQuery(map[string]any{key: summaryMarshaler{&calls, `true`}, "other": summaryMarshaler{&calls, `true`}}))
		if !errors.Is(err, resource.ErrInvalidOption) || wire.Len() != 0 || calls != 0 {
			t.Fatal(key, err, pythonVolumeCalls(wire), calls)
		}
		summaryWrapper(t, err)
	}
	for _, option := range []volumes.SummaryOption{nil, volumes.WithSummaryMicroversion("garbage"), volumes.WithSummaryHeaders(map[string]string{"Authorization": "forged"}), volumes.WithSummaryHeaders(map[string]string{"OpenStack-API-Version": "volume 3.12"})} {
		api, calls := summaryFixture(t, "3.36", func(*http.Request) *http.Response { t.Fatal("HTTP"); return nil })
		_, err := api.SummaryRecord(context.Background(), option)
		if !errors.Is(err, resource.ErrInvalidOption) || calls.Len() != 0 {
			t.Fatal(err, pythonVolumeCalls(calls))
		}
	}
	api, calls := summaryFixture(t, "3.36", func(*http.Request) *http.Response { t.Fatal("HTTP"); return nil })
	_, err := api.SummaryRecord(nil)
	if !errors.Is(err, resource.ErrInvalidOption) || calls.Len() != 0 {
		t.Fatal(err, pythonVolumeCalls(calls))
	}
	api.RawClient().Type = "compute"
	_, err = api.SummaryRecord(context.Background())
	if !errors.Is(err, resource.ErrUnsupported) || calls.Len() != 0 {
		t.Fatal(err, pythonVolumeCalls(calls))
	}
}
func TestPythonVolumeSummaryQuerySnapshotsRequestsExpansionAndOptionReset(t *testing.T) {
	nested := map[string]any{"keys": []any{"a", nil, false}, "order": map[string]any{"key": true}, "nil": nil, "zero": 0}
	option := volumes.WithSummaryQuery(nested)
	nested["keys"].([]any)[0] = "changed"
	retained := map[string]any{"false": false, "dict": map[string]any{"original": true}}
	var physical http.Header
	api, calls := summaryFixture(t, "3.36", func(r *http.Request) *http.Response {
		physical = r.Header.Clone()
		return pythonVolumeWire(200, `{"volume-summary":{}}`)
	})
	v, err := api.SummaryRecord(context.Background(), option, func(c *volumes.SummaryOpts) error {
		for k, x := range retained {
			c.Query[k] = x
		}
		c.Headers = nil
		return nil
	}, func(c *volumes.SummaryOpts) error {
		retained["dict"].(map[string]any)["mutated"] = true
		c.Headers["X-Owned"] = "yes"
		return nil
	}, volumes.WithSummarySingleFetch(true))
	if err != nil || v == nil || calls.Len() != 1 || pythonVolumeCalls(calls)[0].query != "all_tenants=False&dict=original&false=False&keys=a&keys=False&order=key&zero=0" || physical.Get("X-Owned") != "yes" || api.RawClient().MoreHeaders != nil {
		t.Fatal(v, err, pythonVolumeCalls(calls), physical)
	}
	_, err = api.SummaryRecord(context.Background(), volumes.WithSummaryAllProjects(true), volumes.WithSummaryOptions(volumes.SummaryOpts{SingleFetch: true}))
	if err != nil || pythonVolumeCalls(calls)[1].query != "all_tenants=False" {
		t.Fatal(err, pythonVolumeCalls(calls))
	}
}
func TestPythonVolumeSummaryPhysicalFaultsKeepNative401PublicFieldsAndNoHooks(t *testing.T) {
	for _, status := range []int{200, 203, 401, 404, 429, 503} {
		for _, mode := range []string{"read", "close", "both"} {
			t.Run(fmt.Sprintf("%d/%s", status, mode), func(t *testing.T) {
				readCause, closeCause := errors.New("physical read"), errors.New("physical close")
				var closes atomic.Int32
				api, calls := summaryFixture(t, "3.36", func(*http.Request) *http.Response {
					r := pythonVolumeWire(status, `{"volume-summary":{"total_count":2}}`)
					r.Header.Set("X-Proof", "physical")
					b := &connectorFaultBody{ReadCloser: r.Body, closes: &closes}
					if mode != "close" {
						b.readErr = readCause
					}
					if mode != "read" {
						b.closeErr = closeCause
					}
					r.Body = b
					return r
				})
				retries, backoffs, reauth := 0, 0, 0
				api.RawClient().ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				api.RawClient().ProviderClient.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
					backoffs++
					return nil
				}
				api.RawClient().ProviderClient.ReauthFunc = func(context.Context) error { reauth++; return nil }
				v, err := api.SummaryRecord(context.Background(), volumes.WithSummarySingleFetch(true))
				cause := err
				if status == 401 {
					var auth *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &auth) || auth.ErrOriginal == nil || auth.ErrReauth == nil {
						t.Fatal(err, auth)
					}
					cause = errors.Join(auth.ErrOriginal, auth.ErrReauth)
				}
				if err == nil || v == nil || v.Resource != nil || v.StatusCode != status || string(v.Envelope) != `{"volume-summary":{"total_count":2}}` || v.Header.Get("X-Proof") != "physical" || calls.Len() != 1 || closes.Load() != 1 || retries != 0 || backoffs != 0 || reauth != 0 {
					t.Fatal(v, err, pythonVolumeCalls(calls), closes.Load(), retries, backoffs, reauth)
				}
				if mode != "close" && !errors.Is(cause, readCause) || mode != "read" && !errors.Is(cause, closeCause) {
					t.Fatal(err, cause)
				}
				if status < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status {
						t.Fatal(err, proof)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					nativeCause := cause
					if status == 401 {
						var auth *gophercloud.ErrUnableToReauthenticate
						errors.As(err, &auth)
						nativeCause = auth.ErrOriginal
					}
					if !errors.As(nativeCause, &native) || native.Actual != status || native.ResponseHeader.Get("X-Proof") != "physical" {
						t.Fatal(err, native)
					}
				}
				summaryWrapper(t, err)
			})
		}
	}
}
func TestPythonVolumeSummaryCleanReauthenticationAndRetryKeepOwnedRequest(t *testing.T) {
	for _, status := range []int{401, 503} {
		var headers []http.Header
		attempt := 0
		api, calls := summaryFixture(t, "3.36", func(r *http.Request) *http.Response {
			headers = append(headers, r.Header.Clone())
			attempt++
			if attempt == 1 {
				return pythonVolumeWire(status, `{"error":"retry"}`)
			}
			return pythonVolumeWire(200, `{"volume-summary":{"total_count":1}}`)
		})
		hooks := 0
		if status == 401 {
			api.RawClient().ProviderClient.ReauthFunc = func(context.Context) error { hooks++; api.RawClient().ProviderClient.SetToken("renewed"); return nil }
		} else {
			api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, _ error, _ uint) error {
				hooks++
				return nil
			}
		}
		v, err := api.SummaryRecord(context.Background(), volumes.WithSummaryAllProjects(true), volumes.WithSummarySingleFetch(true))
		if err != nil || v == nil || calls.Len() != 2 || hooks != 1 || pythonVolumeCalls(calls)[0].query != "all_tenants=True" || pythonVolumeCalls(calls)[1].query != pythonVolumeCalls(calls)[0].query || headers[1].Get("OpenStack-API-Version") != "volume 3.36" {
			t.Fatal(v, err, pythonVolumeCalls(calls), headers, hooks)
		}
		if status == 401 && headers[1].Get("X-Auth-Token") != "renewed" {
			t.Fatal(headers)
		}
	}
}
func TestPythonVolumeSummaryReadBoundaryGuardAndCancellationCannotResend(t *testing.T) {
	for _, mode := range []string{"source", "cancel"} {
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		cause := errors.New("accepted cancellation")
		var closes atomic.Int32
		var api *volumes.API
		var calls *testcloud.Recorder
		api, calls = summaryFixture(t, "3.36", func(*http.Request) *http.Response {
			r := pythonVolumeWire(203, `{"volume-summary":{"total_count":1}}`)
			r.Body = &connectorFaultBody{ReadCloser: r.Body, closes: &closes, afterRead: func() {
				if mode == "source" {
					api.RawClient().Microversion = "3.12"
				} else {
					cancel(cause)
				}
			}, afterClose: func() { api.RawClient().Microversion = "3.36" }}
			return r
		})
		v, err := api.SummaryRecord(ctx, volumes.WithSummarySingleFetch(true))
		var proof *resource.ResponseError
		if v == nil || v.Resource != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || calls.Len() != 1 || closes.Load() != 1 {
			t.Fatal(v, err, proof, pythonVolumeCalls(calls))
		}
		if mode == "source" && !errors.Is(err, resource.ErrInvalidOption) || mode == "cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
			t.Fatal(err)
		}
	}
}
func TestPythonVolumeSummaryRetryCannotInjectBodyOrSelectedHeaders(t *testing.T) {
	for _, mode := range []string{"body", "version", "foreign", "omit"} {
		api, calls := summaryFixture(t, "3.36", func(*http.Request) *http.Response { return pythonVolumeWire(503, `{"retry":"actual"}`) })
		api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			switch mode {
			case "body":
				o.RawBody = strings.NewReader("forged")
			case "version":
				o.MoreHeaders["OpenStack-API-Version"] = "volume 3.12"
			case "foreign":
				o.MoreHeaders["X-OpenStack-Nova-API-Version"] = "2.90"
			case "omit":
				o.OmitHeaders = append(o.OmitHeaders, "X-OpenStack-Volume-API-Version")
			}
			return nil
		}
		_, err := api.SummaryRecord(context.Background(), volumes.WithSummarySingleFetch(true))
		if !errors.Is(err, resource.ErrInvalidOption) || calls.Len() != 1 || !gophercloud.ResponseCodeIs(err, 503) {
			t.Fatal(mode, err, pythonVolumeCalls(calls))
		}
	}
}

func TestPythonVolumeSummaryHeadersKwargIsQueryAndTypedHeadersArePhysical(t *testing.T) {
	for _, raw := range []any{map[string]any{"X-Query": false}, nil} {
		var physical []http.Header
		api, calls := summaryFixture(t, "3.36", func(r *http.Request) *http.Response {
			physical = append(physical, r.Header.Clone())
			return pythonVolumeWire(200, `{"volume-summary":{"total_count":1}}`)
		})
		value, err := api.SummaryRecord(context.Background(), volumes.WithSummaryQuery(map[string]any{"headers": raw}), volumes.WithSummaryHeaders(map[string]string{"X-Typed": "actual-http"}))
		query := "all_tenants=False"
		if raw != nil {
			query += "&headers=X-Query"
		}
		if err != nil || value == nil || calls.Len() != 2 || pythonVolumeCalls(calls)[0].query != "" || pythonVolumeCalls(calls)[1].query != query {
			t.Fatal(value, err, pythonVolumeCalls(calls))
		}
		for _, header := range physical {
			if header.Get("X-Typed") != "actual-http" || header.Get("X-Query") != "" {
				t.Fatal(header)
			}
		}
	}
}
