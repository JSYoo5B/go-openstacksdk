package servers_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonCloudConsoleRecordLiteralLengthCaseAndRawOutput(t *testing.T) {
	for _, length := range []string{"omit", "null", "0", "false", "-1", "1.5", `"3"`, `""`, `[]`, `{"x":1}`, "9007199254740993"} {
		t.Run(length, func(t *testing.T) {
			var calls []nativeServerCall
			api, _ := serverFindAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.Header.Get("Accept") != "" || req.Header.Get("OpenStack-API-Version") != "compute 2.96" || req.Header.Get("X-OpenStack-Nova-API-Version") != "2.96" || req.Header.Get("X-Auth-Token") != "test-token" {
					t.Error(req.Header)
				}
				return nativeServerWire(203, `{"output":{"x":[false,0,null,"text"]},"Output":"ignored"}`)
			})
			var options []servers.CloudConsoleOption
			if length != "omit" {
				options = append(options, servers.WithCloudConsoleLengthJSON(json.RawMessage(length)))
			}
			got, err := api.ConsoleOutputRecord(context.Background(), "vm", options...)
			want := `{"os-getConsoleOutput":{}}`
			if length != "omit" && length != "null" {
				want = `{"os-getConsoleOutput":{"length":` + length + `}}`
			}
			if err != nil || got == nil || !got.HasOutput || string(got.Value) != `{"x":[false,0,null,"text"]}` || got.StatusCode != 203 || got.Header.Get("X-Server-Proof") != "actual" || got.Microversion != "2.96" || len(calls) != 1 || calls[0].body != want || calls[0].path != serverFindBase+"/vm/action" {
				t.Fatal(got, err, calls, want)
			}
		})
	}
}
func TestPythonCloudConsoleRecordMembershipPresenceAndShape(t *testing.T) {
	for _, tc := range []struct {
		body, want    string
		present, fail bool
	}{
		{`{"output":null}`, "null", true, false}, {`{"output":false}`, "false", true, false}, {`{"output":0}`, "0", true, false}, {`{"output":[1]}`, "[1]", true, false}, {`{"Output":"upper"}`, "null", false, false}, {`{}`, "null", false, false}, {`{"output":1,"output":2}`, "2", true, false},
		{`[]`, "null", false, false}, {`[1,{},"other"]`, "null", false, false}, {`"quiet"`, "null", false, false}, {`["output"]`, "", false, true}, {`"contains output"`, "", false, true}, {`null`, "", false, true}, {`false`, "", false, true}, {`42`, "", false, true}, {``, "", false, true}, {`{"output":`, "", false, true},
	} {
		t.Run(tc.body, func(t *testing.T) {
			var calls []nativeServerCall
			api, _ := serverFindAPI(t, &calls, func(*http.Request) *http.Response { return nativeServerWire(200, tc.body) })
			got, err := api.ConsoleOutputRecord(context.Background(), "vm")
			if got == nil || got.StatusCode != 200 || string(got.Envelope) != tc.body || got.Header.Get("X-Server-Proof") != "actual" || len(calls) != 1 {
				t.Fatal(got, err, calls)
			}
			if tc.fail {
				var actual *resource.ResponseError
				if err == nil || !errors.As(err, &actual) || actual.StatusCode != 200 || string(actual.Body) != tc.body {
					t.Fatal(got, err, actual)
				}
			} else if err != nil || string(got.Value) != tc.want || got.HasOutput != tc.present {
				t.Fatal(got, err)
			}
		})
	}
}
func TestPythonCloudConsoleRecordOptionsAreOwnedAndReset(t *testing.T) {
	raw := json.RawMessage(`{"x":[1]}`)
	opt := servers.WithCloudConsoleLengthJSON(raw)
	raw[2] = 'z'
	var calls []nativeServerCall
	api, _ := serverFindAPI(t, &calls, func(*http.Request) *http.Response { return nativeServerWire(200, `{"output":"ok"}`) })
	var retained *servers.CloudConsoleOpts
	callback := func(c *servers.CloudConsoleOpts) error { retained = c; return nil }
	got, err := api.ConsoleOutputRecord(context.Background(), "vm", opt, callback)
	if err != nil || string(got.Value) != `"ok"` || calls[0].body != `{"os-getConsoleOutput":{"length":{"x":[1]}}}` {
		t.Fatal(got, err, calls)
	}
	(*retained.Length)[2] = 'z'
	_, err = api.ConsoleOutputRecord(context.Background(), "vm", opt)
	_, resetErr := api.ConsoleOutputRecord(context.Background(), "vm", servers.WithCloudConsoleLength(0), servers.WithoutCloudConsoleLength())
	if err != nil || resetErr != nil || calls[1].body != calls[0].body || calls[2].body != `{"os-getConsoleOutput":{}}` {
		t.Fatal(err, resetErr, calls)
	}
}
func TestPythonCloudConsoleRecordPreflightAndSourceCallback(t *testing.T) {
	for _, lane := range []string{"nil-context", "cancelled", "nil-api", "nil-option", "invalid-json", "bad-id", "trimmed-dot", "source"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativeServerCall
			api, _ := serverFindAPI(t, &calls, func(*http.Request) *http.Response { return nativeServerWire(200, `{}`) })
			ctx := context.Background()
			id := "vm"
			var opts []servers.CloudConsoleOption
			switch lane {
			case "nil-context":
				ctx = nil
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "nil-api":
				api = nil
			case "nil-option":
				opts = []servers.CloudConsoleOption{nil}
			case "invalid-json":
				opts = []servers.CloudConsoleOption{servers.WithCloudConsoleLengthJSON(json.RawMessage(`{`))}
			case "bad-id":
				id = "../\x00"
			case "trimmed-dot":
				id = "/..//"
			case "source":
				opts = []servers.CloudConsoleOption{func(*servers.CloudConsoleOpts) error { api.RawClient().ResourceBase += "changed/"; return nil }}
			}
			_, err := api.ConsoleOutputRecord(ctx, id, opts...)
			if err == nil || len(calls) != 0 {
				t.Fatal(err, calls)
			}
		})
	}
}
func TestPythonCloudConsoleRecordFalseyAndLiteralIDRoutes(t *testing.T) {
	for _, tc := range []struct{ id, want string }{{"", serverFindBase + "//action"}, {"/vm/", serverFindBase + "/vm/action"}, {"folder/name%", serverFindBase + "/folder%2Fname%25/action"}} {
		t.Run(tc.id, func(t *testing.T) {
			var calls []nativeServerCall
			var escaped string
			api, _ := serverFindAPI(t, &calls, func(r *http.Request) *http.Response {
				escaped = r.URL.EscapedPath()
				return nativeServerWire(200, `{}`)
			})
			_, err := api.ConsoleOutputRecord(context.Background(), tc.id)
			if err != nil || escaped != tc.want {
				t.Fatal(err, escaped)
			}
		})
	}
}
func TestPythonCloudConsoleRecordOriginalStatusAndInformationalPolicy(t *testing.T) {
	for _, status := range []int{199, 200, 203, 204, 299, 300, 304, 399, 400, 403, 404, 409, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls []nativeServerCall
			api, _ := serverFindAPI(t, &calls, func(*http.Request) *http.Response { return nativeServerWire(status, `{"output":"ok"}`) })
			got, err := api.ConsoleOutputRecord(context.Background(), "vm")
			if got == nil || got.StatusCode != status || len(calls) != 1 {
				t.Fatal(got, err, calls)
			}
			if status < 400 {
				if err != nil || string(got.Value) != `"ok"` {
					t.Fatal(got, err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != status || string(native.Body) != `{"output":"ok"}` {
					t.Fatal(got, err, native)
				}
			}
		})
	}
}
func TestPythonCloudConsoleRecordDiscoveryUsesServerCeilingAndSelectedShortcut(t *testing.T) {
	for _, tc := range []struct {
		selected, max, want string
		calls               int
	}{{"2.87", "2.120", "2.87", 1}, {"", "2.120", "2.100", 2}, {"", "2.76", "2.76", 2}} {
		t.Run(tc.selected+tc.max, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("compute", "/nova/v2.1/")
			client.Microversion = tc.selected
			var calls []string
			var post http.Header
			cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
				calls = append(calls, req.Method+" "+req.URL.Path)
				if req.Method == "GET" {
					// Reuse the Source-vetted CURRENT+self-link advertisement.
					return nativeServerWire(200, pythonBackupAdvertisement(tc.max, "2.1")), nil
				}
				post = req.Header.Clone()
				return nativeServerWire(200, `{"output":"ok"}`), nil
			}))
			got, err := servers.New(client).ConsoleOutputRecord(context.Background(), "vm")
			if err != nil || got.Microversion != tc.want || len(calls) != tc.calls || post.Get("OpenStack-API-Version") != "compute "+tc.want || client.Microversion != tc.selected || len(got.Discovery) != tc.calls-1 {
				t.Fatal(got, err, calls, post)
			}
		})
	}
}
func TestPythonCloudConsoleRecordPhysicalFaultsRetainActualProofAndStopActionRetry(t *testing.T) {
	for _, status := range []int{200, 400, 401, 403, 404, 429, 500, 777} {
		for _, lane := range []string{"read", "close", "both", "cancel", "source"} {
			t.Run(fmt.Sprintf("%d/%s", status, lane), func(t *testing.T) {
				var calls []nativeServerCall
				api, cloud := serverFindAPI(t, &calls, func(*http.Request) *http.Response { return nativeServerWire(200, `{}`) })
				client := api.RawClient()
				base := client.ResourceBase
				defer func() { client.ResourceBase = base }()
				readErr, closeErr, cancelErr := errors.New("read"), errors.New("close"), errors.New("cause")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				body := &testcloud.Body{Data: `{"output":"physical"}`}
				switch lane {
				case "read":
					body.ReadErr = readErr
				case "close":
					body.CloseErr = closeErr
				case "both":
					body.ReadErr = readErr
					body.CloseErr = closeErr
				case "cancel":
					body.OnRead = func() { cancel(cancelErr) }
				case "source":
					body.OnRead = func() { client.ResourceBase += "wrong/" }
					body.OnClose = func() { client.ResourceBase = base }
				}
				attempts, retries, reauth, backoff := 0, 0, 0, 0
				sentinel := errors.New("unexpected reauth")
				cloud.Provider.ReauthFunc = func(context.Context) error { reauth++; return sentinel }
				cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return sentinel
				}
				cloud.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
					backoff++
					return sentinel
				}
				cloud.SetTransport(testcloud.LinkRequest(func(*http.Request) (*http.Response, error) {
					attempts++
					return testcloud.StreamResponse(status, body, http.Header{"X-Actual": {"physical"}}), nil
				}))
				got, err := api.ConsoleOutputRecord(ctx, "vm")
				if err == nil || attempts != 1 || body.Closes() != 1 || retries != 0 || backoff != 0 || reauth != 0 || got == nil || got.StatusCode != status || got.Header.Get("X-Actual") != "physical" || string(got.Envelope) != `{"output":"physical"}` {
					t.Fatal(got, err, attempts, body.Closes(), retries, backoff, reauth)
				}
				if status < 400 {
					var actual *resource.ResponseError
					if !errors.As(err, &actual) || actual.StatusCode != status || string(actual.Body) != string(got.Envelope) {
						t.Fatal(err, actual)
					}
				} else {
					var actual gophercloud.ErrUnexpectedResponseCode
					if !consoleNativeActual(err, &actual) || actual.Actual != status || actual.ResponseHeader.Get("X-Actual") != "physical" {
						t.Fatal(err, actual)
					}
				}
				if (lane == "read" || lane == "both") && !errors.Is(err, readErr) || (lane == "close" || lane == "both") && !errors.Is(err, closeErr) || lane == "cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelErr)) || lane == "source" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			})
		}
	}
}
func TestPythonCloudConsoleRecordRetryOwnershipAndNativeReauthentication(t *testing.T) {
	for _, lane := range []string{"clean", "body", "accept", "version", "codes", "callback"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativeServerCall
			api, cloud := serverFindAPI(t, &calls, func(*http.Request) *http.Response { return nativeServerWire(400, `{"bad":"actual"}`) })
			cause := errors.New("retry failure")
			retries := 0
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, c *gophercloud.RequestOpts, original error, _ uint) error {
				retries++
				switch lane {
				case "clean":
					return original
				case "body":
					c.JSONBody = json.RawMessage(`{"x":1}`)
				case "accept":
					c.MoreHeaders["Accept"] = "evil"
				case "version":
					c.MoreHeaders["X-OpenStack-Nova-API-Version"] = "2.88"
				case "codes":
					c.OkCodes = append(c.OkCodes, 400)
				case "callback":
					return cause
				}
				return nil
			}
			got, err := api.ConsoleOutputRecord(context.Background(), "vm", servers.WithCloudConsoleLength(0))
			wantCalls := 1
			if lane == "codes" {
				wantCalls = 2
			}
			var actual gophercloud.ErrUnexpectedResponseCode
			if err == nil || !errors.As(err, &actual) || actual.Actual != 400 || got.StatusCode != 400 || len(calls) != wantCalls || retries != 1 {
				t.Fatal(got, err, calls, retries, actual)
			}
			if lane == "callback" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if lane != "clean" && lane != "callback" && lane != "codes" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
	t.Run("clean reauth", func(t *testing.T) {
		var calls []nativeServerCall
		attempts, auth := 0, 0
		api, cloud := serverFindAPI(t, &calls, func(r *http.Request) *http.Response {
			attempts++
			if attempts == 1 {
				return nativeServerWire(401, `{"unauthorized":true}`)
			}
			if r.Header.Get("X-Auth-Token") != "refreshed" {
				t.Error(r.Header)
			}
			return nativeServerWire(200, `{"output":false}`)
		})
		cloud.Provider.ReauthFunc = func(context.Context) error { auth++; cloud.Provider.SetToken("refreshed"); return nil }
		got, err := api.ConsoleOutputRecord(context.Background(), "vm", servers.WithCloudConsoleLength(0))
		if err != nil || auth != 1 || len(calls) != 2 || !reflect.DeepEqual(calls[0], calls[1]) || string(got.Value) != "false" {
			t.Fatal(got, err, calls, auth)
		}
	})
}
func TestPythonCloudConsoleRecordOwnedSourceHeadersAndIndependentReceipt(t *testing.T) {
	var calls []nativeServerCall
	api, cloud := serverFindAPI(t, &calls, func(req *http.Request) *http.Response { return nativeServerWire(200, `{"output":"ok"}`) })
	api.RawClient().MoreHeaders["Accept"] = "source-default"
	opt := func(*servers.CloudConsoleOpts) error { api.RawClient().MoreHeaders["X-Source"] = "changed"; return nil }
	var header http.Header
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		header = req.Header.Clone()
		return nativeServerWire(200, `{"output":"ok"}`), nil
	}))
	got, err := api.ConsoleOutputRecord(context.Background(), "vm", opt)
	if err != nil || header.Get("Accept") != "" || header.Get("X-Source") != "direct" {
		t.Fatal(got, err, header)
	}
	got.Value[1] = 'X'
	got.Header.Set("X-Server-Proof", "changed")
	if string(got.Envelope) != `{"output":"ok"}` {
		t.Fatal(got)
	}
}

func TestPythonCloudConsoleRecordRequestsDecodingAndNonFiniteGapStayExplicit(t *testing.T) {
	for _, tc := range []struct{ name, body, contentType string }{
		{"unmarked nonUTF8", string([]byte{'{', '"', 'o', 'u', 't', 'p', 'u', 't', '"', ':', '"', 'c', 'a', 'f', 0xe9, '"', '}'}), "application/json"},
		{"nonfinite", `{"output":1e400}`, "application/json; charset=UTF-8"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []nativeServerCall
			api, _ := serverFindAPI(t, &calls, func(*http.Request) *http.Response {
				r := nativeServerWire(200, tc.body)
				r.Header.Set("Content-Type", tc.contentType)
				return r
			})
			got, err := api.ConsoleOutputRecord(context.Background(), "vm")
			var proof *resource.ResponseError
			if !errors.Is(err, resource.ErrUnsupported) || !errors.As(err, &proof) || got == nil || string(got.Envelope) != tc.body || proof.StatusCode != 200 || string(proof.Body) != tc.body || len(calls) != 1 {
				t.Fatal(got, err, proof, calls)
			}
		})
	}
}

func TestPythonCloudConsoleRecordExplicitCharsetsKeepWireReceipt(t *testing.T) {
	for _, tc := range []struct{ name, contentType, wireHex string }{
		{"latin1", "application/json; charset=ISO-8859-1", "7b226f7574707574223a22636166e9227d"},
		{"utf16 BOM", "application/json; charset=UTF-16", "fffe7b0022006f007500740070007500740022003a002200630061006600e90022007d00"},
		{"utf32BE", "application/json; charset=UTF-32BE", "0000007b000000220000006f0000007500000074000000700000007500000074000000220000003a00000022000000630000006100000066000000e9000000220000007d"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire, decodeErr := hex.DecodeString(tc.wireHex)
			if decodeErr != nil {
				t.Fatal(decodeErr)
			}
			var calls []nativeServerCall
			var originalHeader http.Header
			api, _ := serverFindAPI(t, &calls, func(*http.Request) *http.Response {
				r := nativeServerWire(202, string(wire))
				r.Header.Set("Content-Type", tc.contentType)
				originalHeader = r.Header
				return r
			})
			got, err := api.ConsoleOutputRecord(context.Background(), "vm")
			if err != nil || got == nil || !got.HasOutput || string(got.Value) != `"café"` || got.StatusCode != 202 || !bytes.Equal(got.Envelope, wire) || got.Header.Get("Content-Type") != tc.contentType || len(calls) != 1 {
				t.Fatal(got, err, calls)
			}
			got.Value[1] = 'X'
			got.Header.Set("Content-Type", "changed")
			if !bytes.Equal(got.Envelope, wire) || originalHeader.Get("Content-Type") != tc.contentType {
				t.Fatal("normalized output or copied header changed wire proof", got, originalHeader)
			}
		})
	}
}

func TestPythonCloudConsoleRecordCharsetFailuresPreserveActualResponse(t *testing.T) {
	for _, tc := range []struct {
		name, contentType, body string
		cause                   error
	}{
		{"unknown codec", "application/json; charset=x-not-a-codec", `{"output":"wire"}`, resource.ErrUnsupported},
		{"malformed charset", "application/json; charset=\"unterminated", `{"output":"wire"}`, resource.ErrUnsupported},
		{"normalized invalid JSON", "application/json; charset=UTF-16LE", string([]byte{'{', 0, 'b', 0, 'a', 0, 'd', 0}), resource.ErrInvalidOption},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []nativeServerCall
			api, _ := serverFindAPI(t, &calls, func(*http.Request) *http.Response {
				r := nativeServerWire(203, tc.body)
				r.Header.Set("Content-Type", tc.contentType)
				return r
			})
			got, err := api.ConsoleOutputRecord(context.Background(), "vm")
			var proof *resource.ResponseError
			if !errors.Is(err, tc.cause) || !errors.As(err, &proof) || got == nil || got.HasOutput || got.Value != nil || got.StatusCode != 203 || string(got.Envelope) != tc.body || proof.StatusCode != 203 || string(proof.Body) != tc.body || proof.Header.Get("Content-Type") != tc.contentType || len(calls) != 1 {
				t.Fatal(got, err, proof, calls)
			}
		})
	}
}

func consoleNativeActual(err error, target *gophercloud.ErrUnexpectedResponseCode) bool {
	var unable *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &unable) {
		err = unable.ErrOriginal
	}
	return errors.As(err, target)
}
func TestPythonCloudConsoleRecordFailedReauthReceiptKeepsOriginalAndCause(t *testing.T) {
	var calls []nativeServerCall
	api, cloud := serverFindAPI(t, &calls, func(*http.Request) *http.Response { return nativeServerWire(401, `{"original":"unauthorized"}`) })
	cause := errors.New("reauth unavailable")
	auth := 0
	cloud.Provider.ReauthFunc = func(context.Context) error { auth++; return cause }
	got, err := api.ConsoleOutputRecord(context.Background(), "vm")
	var unable *gophercloud.ErrUnableToReauthenticate
	if !errors.As(err, &unable) || unable.ErrReauth != cause || auth != 1 || len(calls) != 1 || got == nil || got.StatusCode != 401 || string(got.Envelope) != `{"original":"unauthorized"}` || got.Header.Get("X-Server-Proof") != "actual" {
		t.Fatal(got, err, unable, auth, calls)
	}
}

func TestPythonCloudConsoleRecordDiscoveryFaultOptInStopsCallbacksBeforeAction(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("compute", "/nova/v2.1/")
	readCause, closeCause := errors.New("discovery read"), errors.New("discovery close")
	body := &testcloud.Body{Data: `{"discovery":"actual"}`, ReadErr: readCause, CloseErr: closeCause}
	attempts, retries, reauth, backoff := 0, 0, 0, 0
	sentinel := errors.New("unexpected callback")
	cloud.Provider.ReauthFunc = func(context.Context) error { reauth++; return sentinel }
	cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		retries++
		return sentinel
	}
	cloud.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
		backoff++
		return sentinel
	}
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		attempts++
		if req.Method != "GET" || req.URL.Path != "/nova/v2.1/" {
			t.Error(req.Method, req.URL)
		}
		return testcloud.StreamResponse(503, body, http.Header{"X-Actual": {"discovery"}}), nil
	}))
	got, err := servers.New(client).ConsoleOutputRecord(context.Background(), "vm")
	var actual gophercloud.ErrUnexpectedResponseCode
	if got == nil || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.As(err, &actual) || actual.Actual != 503 || string(actual.Body) != `{"discovery":"actual"}` || actual.ResponseHeader.Get("X-Actual") != "discovery" || attempts != 1 || body.Closes() != 1 || retries != 0 || reauth != 0 || backoff != 0 || got.StatusCode != 0 {
		t.Fatal(got, err, actual, attempts, body.Closes(), retries, reauth, backoff)
	}
}
