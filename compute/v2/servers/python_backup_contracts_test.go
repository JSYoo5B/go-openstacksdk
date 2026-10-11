package servers_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// This fixture reuses the shared Gophercloud testhelper-backed cloud. Its Nova
// endpoint is eligible for version-row matching; the action keeps ResourceBase.
func pythonBackupCloud(t *testing.T, selected, maximum, minimum string, reply func(*http.Request) *http.Response) (*gophercloud.ServiceClient, *[]pythonServerCall) {
	t.Helper()
	client, _, calls := pythonServerCloud(t, func(req *http.Request) *http.Response {
		if req.Method == http.MethodGet {
			if req.URL.Path != "/nova/v2.1/" || req.URL.RawQuery != "" || req.Header.Get("X-OpenStack-Nova-API-Version") != "" || req.Header.Get("OpenStack-API-Version") != "" || req.Body != nil && req.Body != http.NoBody {
				t.Errorf("discovery request: %s %s headers=%v", req.Method, req.URL, req.Header)
			}
			return nativeServerWire(200, pythonBackupAdvertisement(maximum, minimum))
		}
		return reply(req)
	})
	client.Endpoint = client.ResourceBase
	client.Microversion = selected
	return client, calls
}

func pythonBackupAdvertisement(maximum, minimum string) string {
	// The self link establishes row eligibility; it cannot choose the action's
	// authority or route. No request ever goes to this unrelated advertised URL.
	return fmt.Sprintf(`{"version":{"id":"v2.1","status":"CURRENT","min_version":%q,"version":%q,"links":[{"rel":"self","href":"https://advertised.invalid/another/nova/v2.1/"}]}}`, minimum, maximum)
}

func pythonBackupRequest() servers.BackupActionRequest {
	return servers.BackupActionRequest{Name: "backup", BackupType: "daily", Rotation: 3}
}

func TestPythonBackupActionMicroversionSelectionAndSourceOwnership(t *testing.T) {
	cases := []struct{ name, selected, maximum, minimum, want string }{
		{"advertised support with default", "2.96", "2.100", "2.1", "2.45"},
		{"advertised support without default", "", "2.100", "2.1", "2.45"},
		{"selected below required stays selected", "2.44", "2.100", "2.1", "2.44"},
		{"selected above maximum is not clamped", "2.999", "2.100", "2.1", "2.45"},
		{"global latest is not finite major support", "latest", "2.100", "2.1", "latest"},
		{"old cloud with selected default", "2.43", "2.44", "2.1", "2.43"},
		{"old cloud without selected default", "", "2.44", "2.1", "2.44"},
		{"missing minimum with selected default", "2.96", "2.100", "", "2.96"},
		{"missing minimum selects advertised maximum", "", "2.100", "", "2.100"},
		{"missing minimum observes SDK ceiling", "", "2.999", "", "2.100"},
		{"missing maximum sends no microversion", "", "", "2.1", ""},
		{"minimum above SDK ceiling sends no microversion", "", "2.110", "2.101", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, calls := pythonBackupCloud(t, tc.selected, tc.maximum, tc.minimum, func(req *http.Request) *http.Response {
				if req.Header.Get("Accept") != "" || req.Header.Get("X-Auth-Token") != "test-token" {
					t.Error(req.Header)
				}
				return nativeServerWire(202, `{"image_id":"img"}`)
			})
			before := *client
			got, err := servers.New(client).CreateBackupAction(context.Background(), "s1", pythonBackupRequest())
			wantCalls := []pythonServerCall{
				{http.MethodGet, "/nova/v2.1/", "", "", ""},
				{http.MethodPost, pythonServerBase + "/action", "", `{"createBackup":{"backup_type":"daily","name":"backup","rotation":3}}`, tc.want},
			}
			if err != nil || got == nil || got.ImageID != "img" || got.Microversion != tc.want || got.StatusCode != 202 || got.Header.Get("X-Server-Proof") != "actual" || string(got.Body) != `{"image_id":"img"}` || !got.Discovery.Found || got.Discovery.Maximum != tc.maximum || got.Discovery.Minimum != tc.minimum || len(got.Discovery.Responses) != 1 || !reflect.DeepEqual(*calls, wantCalls) {
				t.Fatalf("result=%+v err=%v calls=%+v want=%+v", got, err, *calls, wantCalls)
			}
			if client.Microversion != before.Microversion || client.Endpoint != before.Endpoint || client.ResourceBase != before.ResourceBase || client.ProviderClient != before.ProviderClient || !reflect.DeepEqual(client.MoreHeaders, before.MoreHeaders) {
				t.Fatal("action changed selected client", client, before)
			}
		})
	}
}

func TestPythonBackupActionDiscoveryFallbackKeepsOwnedRoutes(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(fmt.Sprint(missing), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := nativeServerClient(cloud)
			client.Endpoint = client.ResourceBase
			var calls []pythonServerCall
			cloud.Provider.HTTPClient.Transport = pythonServerTransport(func(req *http.Request) (*http.Response, error) {
				body := ""
				if req.Body != nil {
					raw, err := io.ReadAll(req.Body)
					if err != nil {
						t.Fatal(err)
					}
					body = string(raw)
				}
				calls = append(calls, pythonServerCall{req.Method, req.URL.Path, req.URL.RawQuery, body, req.Header.Get("X-OpenStack-Nova-API-Version")})
				if req.Method == http.MethodPost {
					return nativeServerWire(202, `{"image_id":"img"}`), nil
				}
				if req.URL.Path == "/nova/v2.1/" {
					return nativeServerWire(404, "version missing"), nil
				}
				if req.URL.Path != "/nova/" {
					t.Fatal("unexpected discovery route", req.URL)
				}
				if missing {
					return nativeServerWire(200, `{"versions":[]}`), nil
				}
				return nativeServerWire(200, pythonBackupAdvertisement("2.100", "2.1")), nil
			})
			got, err := servers.New(client).CreateBackupAction(context.Background(), "s1", pythonBackupRequest())
			if err != nil || got == nil || got.ImageID != "img" || len(calls) != 3 || calls[2].path != pythonServerBase+"/action" || got.Discovery.Found == missing || len(got.Discovery.Responses) != 1 {
				t.Fatal(got, err, calls)
			}
			var rejection gophercloud.ErrUnexpectedResponseCode
			if !errors.As(got.Discovery.LastRejection, &rejection) || rejection.Actual != 404 || string(rejection.Body) != "version missing" {
				t.Fatal("discovery loses clean rejection", got.Discovery.LastRejection)
			}
			want := "2.45"
			if missing {
				want = ""
			}
			if got.Microversion != want || calls[2].version != want || client.Microversion != "" {
				t.Fatal(got.Microversion, calls, client.Microversion)
			}
		})
	}
}

func TestPythonBackupActionValuesExtensionsAndResponseEvidence(t *testing.T) {
	var response *http.Response
	var action *http.Request
	postCount := 0
	client, calls := pythonBackupCloud(t, "2.96", "2.100", "2.1", func(req *http.Request) *http.Response {
		action = req
		postCount++
		wantHeader := ""
		if postCount == 1 {
			wantHeader = "caller"
		}
		if req.Header.Get("X-Test") != wantHeader || req.Header.Get("Accept") != "" || req.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(req.Header)
		}
		response = nativeServerWire(203, `{"image_id":"body-id","future":false}`)
		response.Header.Set("Location", "https://unrelated.invalid/images/header-id")
		return response
	})
	client.MoreHeaders = map[string]string{"X-Source": "before", "aCcEpT": "application/json", "x-openstack-nova-api-version": "2.96", "OpenStack-API-Version": "compute 2.96"}
	extension := map[string]any{"null": nil, "zero": 0, "false": false, "empty": []string{}}
	snapshot := servers.WithBackupActionField("future", extension)
	extension["zero"] = 99
	got, err := servers.New(client).CreateBackupAction(context.Background(), "a/b%2Fz", pythonBackupRequest(), snapshot,
		servers.WithBackupActionHeader("X-Test", "caller"), servers.WithBackupActionOptions(servers.BackupActionRequest{Rotation: -1}))
	wantBody := `{"createBackup":{"backup_type":"","future":{"empty":[],"false":false,"null":null,"zero":0},"name":"","rotation":-1}}`
	if err != nil || got == nil || got.ImageID != "body-id" || len(*calls) != 2 || (*calls)[1].body != wantBody || action.URL.EscapedPath() != "/nova/v2.1/servers/a%2Fb%252Fz/action" || got.StatusCode != 203 || string(got.Body) != `{"image_id":"body-id","future":false}` {
		t.Fatal(got, err, *calls)
	}
	// The record owns actual response evidence; later response mutation cannot
	// replace its proof or turn the foreign Location into a follow-up request.
	response.Header.Set("X-Server-Proof", "mutated")
	if got.Header.Get("X-Server-Proof") != "actual" || got.Header.Get("Location") != "https://unrelated.invalid/images/header-id" || !reflect.DeepEqual(client.MoreHeaders, map[string]string{"X-Source": "before", "aCcEpT": "application/json", "x-openstack-nova-api-version": "2.96", "OpenStack-API-Version": "compute 2.96"}) {
		t.Fatal(got.Header, client.MoreHeaders)
	}
	got, err = servers.New(client).CreateBackupAction(context.Background(), "s1", servers.BackupActionRequest{})
	if err != nil || got == nil || len(*calls) != 4 || (*calls)[3].body != `{"createBackup":{"backup_type":"","name":"","rotation":0}}` {
		t.Fatal("zero values were omitted or prevalidated", got, err, *calls)
	}
}

func TestPythonBackupActionImageIDDecodeOrderAndSafety(t *testing.T) {
	cases := []struct {
		name, body, location, want string
		failure                    bool
	}{
		{"JSON wins even on selected pre-2.45", `{"image_id":"body"}`, "images/header", "body", false},
		{"valid JSON lacks ID uses Location", `{"future":true}`, "images/header", "header", false},
		{"empty object uses Location", `{}`, "images/header", "header", false},
		{"null body uses Location", `null`, "images/header", "header", false},
		{"invalid JSON uses Location", `not JSON`, "images/header", "header", false},
		{"invalid UTF-8 body uses Location", "\xff", "images/header", "header", false},
		{"empty body uses Location", ``, "images/header", "header", false},
		{"absolute unrelated Location is not followed", ``, "https://unrelated.invalid/a/img", "img", false},
		{"Location uses final slash segment", `{}`, "relative/one/two", "two", false},
		{"body ID null does not fall back", `{"image_id":null}`, "images/header", "", true},
		{"body ID numeric does not fall back", `{"image_id":7}`, "images/header", "", true},
		{"body ID boolean does not fall back", `{"image_id":false}`, "images/header", "", true},
		{"body ID object does not fall back", `{"image_id":{}}`, "images/header", "", true},
		{"body ID empty does not fall back", `{"image_id":""}`, "images/header", "", true},
		{"body ID dot is unsafe", `{"image_id":"."}`, "images/header", "", true},
		{"body ID traversal is unsafe", `{"image_id":".."}`, "images/header", "", true},
		{"body ID control is unsafe", `{"image_id":"img\n"}`, "images/header", "", true},
		{"Location missing", `{}`, "", "", true},
		{"Location lacks a slash", `{}`, "header", "", true},
		{"Location final segment empty", `{}`, "images/", "", true},
		{"Location traversal segment unsafe", `{}`, "images/..", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, calls := pythonBackupCloud(t, "2.44", "2.100", "2.1", func(*http.Request) *http.Response {
				response := nativeServerWire(202, tc.body)
				if tc.location != "" {
					response.Header.Set("Location", tc.location)
				}
				return response
			})
			got, err := servers.New(client).CreateBackupAction(context.Background(), "s1", pythonBackupRequest())
			if got == nil || len(*calls) != 2 || got.StatusCode != 202 || got.Header.Get("X-Server-Proof") != "actual" || string(got.Body) != tc.body || got.Microversion != "2.44" {
				t.Fatal(got, err, *calls)
			}
			if tc.failure {
				var receipt *resource.ResponseError
				if err == nil || got.ImageID != "" || !errors.As(err, &receipt) || receipt.StatusCode != 202 || string(receipt.Body) != tc.body || receipt.Header.Get("X-Server-Proof") != "actual" {
					t.Fatal("decode failure lost actual accepted response", got, err, receipt)
				}
			} else if err != nil || got.ImageID != tc.want {
				t.Fatal(got, err)
			}
		})
	}
}

func TestPythonBackupActionOriginalStatusPolicy(t *testing.T) {
	for _, status := range []int{200, 202, 204, 299, 300, 302, 399, 199, 400, 404, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			client, calls := pythonBackupCloud(t, "2.96", "2.100", "2.1", func(*http.Request) *http.Response {
				// No Location is supplied for status-policy cases: redirect
				// callback behavior is independent of the accepted-code policy.
				return nativeServerWire(status, `{"image_id":"img"}`)
			})
			got, err := servers.New(client).CreateBackupAction(context.Background(), "s1", pythonBackupRequest())
			if got == nil || len(*calls) != 2 || len(got.Discovery.Responses) != 1 {
				t.Fatal(got, err, *calls)
			}
			if status >= 200 && status < 400 {
				if err != nil || got.ImageID != "img" || got.StatusCode != status || string(got.Body) != `{"image_id":"img"}` {
					t.Fatal(got, err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != status || string(native.Body) != `{"image_id":"img"}` || native.ResponseHeader.Get("X-Server-Proof") != "actual" || got.ImageID != "" || got.StatusCode != 0 {
					t.Fatal(got, err, native)
				}
			}
		})
	}
}

func TestPythonBackupActionOptionPreflightAndCapturedSource(t *testing.T) {
	client, calls := pythonBackupCloud(t, "2.96", "2.100", "2.1", func(*http.Request) *http.Response { return nativeServerWire(202, `{"image_id":"img"}`) })
	api := servers.New(client)
	options := []servers.BackupActionOption{
		nil,
		servers.WithBackupActionField("name", "override"),
		servers.WithBackupActionField("backup_type", "override"),
		servers.WithBackupActionField("rotation", 7),
		servers.WithBackupActionField("future", make(chan int)),
		servers.WithBackupActionHeader("X-Auth-Token", "foreign"),
		servers.WithBackupActionHeader("X-OpenStack-Nova-API-Version", "2.1"),
		servers.WithBackupActionHeader("OpenStack-API-Version", "compute 2.1"),
		servers.WithBackupActionHeader("Content-Length", "99"),
		servers.WithBackupActionHeader("Accept", "application/json"),
		servers.WithBackupActionHeader("Accept", ""),
		servers.WithBackupActionOptions(servers.BackupActionRequest{Name: "\xff"}),
		servers.WithBackupActionOptions(servers.BackupActionRequest{BackupType: "\xff"}),
		request.WithQuery[servers.BackupActionRequest]("future", "true"),
		request.WithArgument[servers.BackupActionRequest]("future", true),
	}
	for i, option := range options {
		got, err := api.CreateBackupAction(context.Background(), "s1", pythonBackupRequest(), option)
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(i, got, err, *calls)
		}
	}
	optionCalls := 0
	callback := func(c *request.Config[servers.BackupActionRequest]) error { optionCalls++; return nil }
	var nilAPI *servers.API
	if got, err := nilAPI.CreateBackupAction(context.Background(), "s1", pythonBackupRequest(), callback); got != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(got, err)
	}
	if got, err := api.CreateBackupAction(nil, "s1", pythonBackupRequest(), callback); got != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(got, err)
	}
	cause := errors.New("caller cancelled")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if got, err := api.CreateBackupAction(ctx, "s1", pythonBackupRequest(), callback); got != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal(got, err)
	}
	for _, id := range []string{"", ".", "..", "s1\n", "\xff"} {
		got, err := api.CreateBackupAction(context.Background(), id, pythonBackupRequest(), callback)
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(id, got, err)
		}
	}
	if optionCalls != 0 || len(*calls) != 0 {
		t.Fatal(optionCalls, *calls)
	}
	got, err := api.CreateBackupAction(context.Background(), "s1", pythonBackupRequest(), func(c *request.Config[servers.BackupActionRequest]) error {
		client.ResourceBase = "https://changed.invalid/v2.1/"
		return nil
	})
	if got != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
		t.Fatal(got, err, *calls)
	}
}

func TestPythonBackupActionPhysicalFaultsPreserveAllCauses(t *testing.T) {
	cases := []struct {
		name      string
		discovery bool
		status    int
	}{
		{"accepted action", false, 202},
		{"rejected action", false, 404},
		{"accepted discovery", true, 200},
		{"rejected discovery", true, 404},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := nativeServerClient(cloud)
			client.Endpoint = client.ResourceBase
			client.Microversion = "2.96"
			readErr := errors.New("physical read")
			closeErr := errors.New("physical close")
			cause := errors.New("physical cancellation")
			ctx, cancel := context.WithCancelCause(context.Background())
			original := client.ResourceBase
			calls, actions := 0, 0
			cloud.Provider.HTTPClient.Transport = pythonServerTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method == http.MethodPost {
					actions++
				}
				if !tc.discovery && req.Method == http.MethodGet {
					return nativeServerWire(200, pythonBackupAdvertisement("2.100", "2.1")), nil
				}
				response := nativeServerWire(tc.status, "")
				response.Body = &userActionFaultBody{reader: strings.NewReader("partial-wire"), readErr: readErr, closeErr: closeErr,
					afterRead:  func() { client.ResourceBase = "https://changed.invalid/v2.1/"; cancel(cause) },
					afterClose: func() { client.ResourceBase = original }}
				return response, nil
			})
			got, err := servers.New(client).CreateBackupAction(ctx, "s1", pythonBackupRequest())
			for _, target := range []error{readErr, closeErr, context.Canceled, cause, resource.ErrInvalidOption} {
				if !errors.Is(err, target) {
					t.Fatal(target, got, err)
				}
			}
			if got == nil || got.ImageID != "" || calls != 1+actions || (tc.discovery && actions != 0) || (!tc.discovery && actions != 1) {
				t.Fatal(got, err, calls, actions)
			}
			if tc.status < 400 {
				var receipt *resource.ResponseError
				if !errors.As(err, &receipt) || receipt.StatusCode != tc.status || string(receipt.Body) != "partial-wire" || receipt.Header.Get("X-Server-Proof") != "actual" {
					t.Fatal(got, err, receipt)
				}
				if tc.discovery {
					if len(got.Discovery.Responses) != 1 || string(got.Discovery.Responses[0].Body) != "partial-wire" || got.StatusCode != 0 {
						t.Fatal(got)
					}
				} else if got.StatusCode != tc.status || string(got.Body) != "partial-wire" || got.Header.Get("X-Server-Proof") != "actual" {
					t.Fatal(got)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != tc.status || string(native.Body) != "partial-wire" || native.ResponseHeader.Get("X-Server-Proof") != "actual" || got.StatusCode != 0 {
					t.Fatal(got, err, native)
				}
			}
		})
	}
}

func TestPythonBackupActionRetriesProtectBodyAndVersion(t *testing.T) {
	for _, change := range []string{"clean", "body", "version", "accept", "response ownership", "status policy"} {
		t.Run(change, func(t *testing.T) {
			attempts := 0
			client, calls := pythonBackupCloud(t, "2.96", "2.100", "2.1", func(req *http.Request) *http.Response {
				attempts++
				if req.Header.Get("X-OpenStack-Nova-API-Version") != "2.45" || req.Header.Get("Accept") != "" {
					t.Error(req.Header)
				}
				if attempts == 1 {
					return nativeServerWire(503, "original-rejection")
				}
				return nativeServerWire(202, `{"image_id":"img"}`)
			})
			callbacks := 0
			client.ProviderClient.RetryFunc = func(ctx context.Context, method, target string, options *gophercloud.RequestOpts, original error, count uint) error {
				callbacks++
				switch change {
				case "body":
					options.JSONBody = map[string]any{"createBackup": map[string]any{"name": "changed"}}
				case "version":
					options.MoreHeaders["X-OpenStack-Nova-API-Version"] = "2.96"
				case "accept":
					options.MoreHeaders["Accept"] = "application/json"
				case "response ownership":
					options.KeepResponseBody = false
				case "status policy":
					options.OkCodes = append(options.OkCodes, 503)
				}
				return nil
			}
			got, err := servers.New(client).CreateBackupAction(context.Background(), "s1", pythonBackupRequest())
			if got == nil || callbacks != 1 || client.Microversion != "2.96" {
				t.Fatal(got, err, callbacks)
			}
			if change == "clean" {
				if err != nil || got.ImageID != "img" || attempts != 2 || len(*calls) != 3 {
					t.Fatal(got, err, attempts, *calls)
				}
			} else if change == "status policy" {
				// A native callback may expand OkCodes, but the SDK still checks
				// the actual final response against its original 200..399 policy.
				// Here the retry receives a real 202, so expansion alone is harmless.
				if err != nil || got.ImageID != "img" || attempts != 2 || len(*calls) != 3 {
					t.Fatal(got, err, attempts, *calls)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != "original-rejection" || attempts != 1 || len(*calls) != 2 || got.ImageID != "" || got.StatusCode != 0 {
					t.Fatal(got, err, native, attempts, *calls)
				}
			}
		})
	}
	client, calls := pythonBackupCloud(t, "2.96", "2.100", "2.1", func(*http.Request) *http.Response { return nativeServerWire(404, "gone") })
	retryErr := errors.New("retry denied")
	client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return retryErr }
	got, err := servers.New(client).CreateBackupAction(context.Background(), "s1", pythonBackupRequest())
	var native gophercloud.ErrUnexpectedResponseCode
	if got == nil || !errors.Is(err, retryErr) || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "gone" || len(*calls) != 2 || got.StatusCode != 0 {
		t.Fatal(got, err, native, *calls)
	}
}

func TestPythonBackupActionDiscoveryDecodeFailureDoesNotPOST(t *testing.T) {
	for _, body := range []string{`not JSON`, `null`, `{"version":null}`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := nativeServerClient(cloud)
			client.Endpoint = client.ResourceBase
			calls := 0
			cloud.Provider.HTTPClient.Transport = pythonServerTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet || req.URL.Path != "/nova/v2.1/" {
					t.Error(req.Method, req.URL)
				}
				return nativeServerWire(200, body), nil
			})
			got, err := servers.New(client).CreateBackupAction(context.Background(), "s1", pythonBackupRequest())
			var receipt *resource.ResponseError
			if got == nil || err == nil || !errors.As(err, &receipt) || receipt.StatusCode != 200 || string(receipt.Body) != body || receipt.Header.Get("X-Server-Proof") != "actual" || calls != 1 || len(got.Discovery.Responses) != 1 || got.StatusCode != 0 || got.ImageID != "" {
				t.Fatal(got, err, receipt, calls)
			}
		})
	}
}

func TestPythonBackupActionAdvertisedVersionErrorsRetainDiscoveryProof(t *testing.T) {
	for _, tc := range []struct{ selected, maximum, minimum string }{{"2.96", "2.100", "bad"}, {"2.96", "bad", "2.1"}, {"not-a-version", "2.100", "2.1"}} {
		t.Run(tc.selected+"/"+tc.maximum+"/"+tc.minimum, func(t *testing.T) {
			client, calls := pythonBackupCloud(t, tc.selected, tc.maximum, tc.minimum, func(req *http.Request) *http.Response {
				t.Error("selection error issued action", req.Method, req.URL)
				return nativeServerWire(202, `{"image_id":"img"}`)
			})
			got, err := servers.New(client).CreateBackupAction(context.Background(), "s1", pythonBackupRequest())
			if got == nil || err == nil || len(*calls) != 1 || len(got.Discovery.Responses) != 1 || got.StatusCode != 0 || got.ImageID != "" || !got.Discovery.Found || got.Discovery.Maximum != tc.maximum || got.Discovery.Minimum != tc.minimum {
				t.Fatal(got, err, *calls)
			}
			wire := got.Discovery.Responses[0]
			if wire.StatusCode != 200 || wire.Header.Get("X-Server-Proof") != "actual" || string(wire.Body) != pythonBackupAdvertisement(tc.maximum, tc.minimum) || client.Microversion != tc.selected {
				t.Fatal(wire, client)
			}
		})
	}
}
