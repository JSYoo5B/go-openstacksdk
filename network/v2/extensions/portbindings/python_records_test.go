package portbindings_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	sdk "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/portbindings"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type call struct{ method, path, query, body string }

func api(t *testing.T, calls *[]call, reply func(*http.Request) *http.Response) *sdk.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(r *http.Request) (*http.Response, error) {
		var raw []byte
		if r.Body != nil {
			raw, _ = io.ReadAll(r.Body)
		}
		*calls = append(*calls, call{r.Method, r.URL.EscapedPath(), r.URL.RawQuery, string(raw)})
		return reply(r), nil
	}))
	client := cloud.Client("network", "/v2.0/")
	return sdk.New(client)
}
func wire(code int, body string) *http.Response {
	return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"binding-proof"}})
}
func field(t *testing.T, row *sdk.Record, key, want string) {
	t.Helper()
	if row == nil || row.Resource == nil || string(row.Resource.Body[key]) != want {
		t.Fatalf("%s = %#v want %s", key, row, want)
	}
}
func TestPortBindingCreateResolvesParentAndPreservesRawAttrs(t *testing.T) {
	var calls []call
	service := api(t, &calls, func(r *http.Request) *http.Response {
		if len(calls) == 1 {
			if r.Method != "GET" || r.URL.Path != "/v2.0/ports/input" {
				t.Fatal(r.Method, r.URL)
			}
			return wire(203, `{"port":{"id":"actual"}}`)
		}
		if r.Method != "POST" || r.URL.Path != "/v2.0/ports/actual/bindings" {
			t.Fatal(r.Method, r.URL)
		}
		var body map[string]map[string]json.RawMessage
		if err := json.Unmarshal([]byte(calls[1].body), &body); err != nil {
			t.Fatal(err)
		}
		if string(body["binding"]["host"]) != `"h"` || string(body["binding"]["profile"]) != `7` || string(body["binding"]["vif_details"]) != `null` || string(body["binding"]["future"]) != `{"keep":true}` || string(body["binding"]["location"]) != `{"cloud":"provided"}` || len(body["binding"]) != 5 {
			t.Fatal(body)
		}
		return wire(202, `{"binding":{"status":"ACTIVE","vif_type":"ovs","port_id":"untrusted"}}`)
	})
	row, err := service.Create(context.Background(), "input", sdk.WithAttribute("host", "h"), sdk.WithAttribute("profile", 7), sdk.WithAttribute("vif_details", nil), sdk.WithAttribute("future", map[string]bool{"keep": true}), sdk.WithAttribute("location", map[string]string{"cloud": "provided"}))
	if err != nil || row == nil || len(calls) != 2 || row.PortID != "actual" || len(row.Ports) != 1 || row.Ports[0].StatusCode != 203 || row.StatusCode != 202 {
		t.Fatal(row, err, calls)
	}
	field(t, row, "host", `"h"`)
	field(t, row, "profile", `{}`)
	field(t, row, "vif_details", `null`)
	field(t, row, "future", `{"keep":true}`)
	field(t, row, "location", `null`)
	if row.Resource.Body["port_id"] != nil || string(row.Wire.Body["port_id"]) != `"untrusted"` {
		t.Fatal(row)
	}
}
func TestPortBindingListFiltersLocallyAndOwnsParentEvidence(t *testing.T) {
	var calls []call
	service := api(t, &calls, func(r *http.Request) *http.Response {
		if strings.HasSuffix(r.URL.Path, "/ports/p") {
			return wire(200, `{"port":{"id":"p"}}`)
		}
		if r.URL.Query().Has("host") || r.URL.Query().Has("vif_type") || r.URL.Query().Has("unknown") {
			t.Fatal(r.URL)
		}
		if r.URL.Query().Get("marker") == "one" {
			return wire(200, `{"bindings":[{"id":"two","host":"h","profile":false,"vif_details":{"keep":true},"future":1}]}`)
		}
		return wire(200, `{"bindings":[{"id":"one","host":"other"}],"bindings_links":[{"rel":"next","href":"?marker=one"}]}`)
	})
	sequence := service.List(context.Background(), "p", sdk.WithQuery("host", "h"), sdk.WithQuery("unknown", "ignored"))
	if len(calls) != 0 {
		t.Fatal("Go iterator must be lazy")
	}
	var rows []*sdk.Record
	for row, err := range sequence {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if len(calls) != 3 || len(rows) != 1 {
		t.Fatal(rows, calls)
	}
	field(t, rows[0], "profile", `{}`)
	field(t, rows[0], "future", `1`)
}
func TestPortBindingActivateRepeatsParentAndStopsAtFirstMatch(t *testing.T) {
	for _, mode := range []string{"flat", "wrapped", "empty", "pairs", "invalid-json", "invalid-shape"} {
		t.Run(mode, func(t *testing.T) {
			var calls []call
			service := api(t, &calls, func(r *http.Request) *http.Response {
				switch len(calls) {
				case 1:
					if r.URL.Path != "/v2.0/ports/p" {
						t.Fatal(r.URL)
					}
					return wire(200, `{"port":{"id":"one"}}`)
				case 2:
					if r.URL.Path != "/v2.0/ports/one" {
						t.Fatal(r.URL)
					}
					return wire(200, `{"port":{"id":"two"}}`)
				case 3:
					if r.URL.Path != "/v2.0/ports/two/bindings" || r.URL.RawQuery != "" {
						t.Fatal(r.URL)
					}
					return wire(200, `{"bindings":[{"host":"h","status":"INACTIVE","profile":false,"old_extension":1},{"host":"h","status":"SECOND"}],"next":"?marker=late"}`)
				case 4:
					if r.Method != "PUT" || r.URL.Path != "/v2.0/ports/two/bindings/h/activate" || calls[3].body != "" {
						t.Fatal(r.Method, r.URL, calls)
					}
					switch mode {
					case "flat":
						return wire(203, `{"status":"ACTIVE","profile":{"keep":true},"old_extension":2,"new_extension":3}`)
					case "wrapped":
						return wire(200, `{"binding":{"status":"ACTIVE"}}`)
					case "empty":
						return wire(200, `[]`)
					case "pairs":
						return wire(200, `[["status","ACTIVE"]]`)
					case "invalid-json":
						return wire(200, `not-json`)
					default:
						return wire(200, `true`)
					}
				default:
					t.Fatal("activation must not consume latepage or secondbinding", calls)
					return wire(500, `{}`)
				}
			})
			result, err := service.Activate(context.Background(), "p", "h")
			if len(calls) != 4 || result == nil || len(result.Ports) != 2 || result.Response == nil {
				t.Fatal(result, err, calls)
			}
			if strings.HasPrefix(mode, "invalid") {
				var physical *resource.ResponseError
				if !errors.As(err, &physical) || physical.StatusCode != 200 || result.Binding != nil {
					t.Fatal(result, err)
				}
				return
			}
			if err != nil || result.Binding == nil {
				t.Fatal(result, err)
			}
			if mode == "flat" || mode == "pairs" {
				field(t, result.Binding, "status", `"ACTIVE"`)
			} else {
				field(t, result.Binding, "status", `"INACTIVE"`)
			}
			field(t, result.Binding, "old_extension", `1`)
			for _, key := range []string{"new_extension", "binding"} {
				if result.Binding.Resource.Body[key] != nil {
					t.Fatal("action does not register new unknown attrs", result.Binding)
				}
			}
		})
	}
}
func TestPortBindingDeleteMaterializesAllMatchesBeforeMutations(t *testing.T) {
	for _, lateFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "all-match-deletes", true: "late-list-error"}[lateFailure], func(t *testing.T) {
			var calls []call
			service := api(t, &calls, func(r *http.Request) *http.Response {
				if len(calls) <= 2 {
					return wire(200, `{"port":{"id":"p"}}`)
				}
				if r.Method == "DELETE" {
					if len(calls) < 5 || r.URL.Path != "/v2.0/ports/p/bindings/h" {
						t.Fatal(calls, r.URL)
					}
					return wire(204, "")
				}
				if r.URL.Query().Get("marker") == "one" {
					if lateFailure {
						return wire(500, `{"error":"late"}`)
					}
					return wire(200, `{"bindings":[{"id":"two","host":"h"}]}`)
				}
				return wire(200, `{"bindings":[{"id":"one","host":"h"}],"bindings_links":[{"rel":"next","href":"?marker=one"}]}`)
			})
			result, err := service.Delete(context.Background(), "p", "h")
			if result == nil {
				t.Fatal(err)
			}
			if lateFailure {
				if err == nil || result.Deleted != 0 || len(calls) != 4 {
					t.Fatal(result, err, calls)
				}
			} else if err != nil || result.Deleted != 2 || len(result.Bindings) != 2 || len(result.Responses) != 2 || len(calls) != 6 {
				t.Fatal(result, err, calls)
			}
		})
	}
}
func TestPortBindingDeleteIgnoreMissingAppliesOnlyToSourceBranches(t *testing.T) {
	for _, mode := range []string{"initial-port404", "strict-initial404", "second-port404", "empty", "strict-empty", "delete404"} {
		t.Run(mode, func(t *testing.T) {
			var calls []call
			service := api(t, &calls, func(r *http.Request) *http.Response {
				if len(calls) == 1 && (mode == "initial-port404" || mode == "strict-initial404") {
					return wire(404, `{"error":"missing port"}`)
				}
				if len(calls) == 2 && mode == "second-port404" {
					return wire(404, `{"error":"second port"}`)
				}
				if len(calls) <= 2 {
					return wire(200, `{"port":{"id":"p"}}`)
				}
				if r.Method == "DELETE" {
					return wire(404, `{"error":"binding disappeared"}`)
				}
				if mode == "delete404" {
					return wire(200, `{"bindings":[{"host":"h"}]}`)
				}
				return wire(200, `{"bindings":[]}`)
			})
			var options []sdk.Option
			if strings.HasPrefix(mode, "strict") {
				options = append(options, sdk.WithIgnoreMissing(false))
			}
			result, err := service.Delete(context.Background(), "p", "h", options...)
			ignored := mode == "initial-port404" || mode == "empty"
			if ignored {
				if err != nil || result == nil || !result.IgnoredMissing || result.Deleted != 0 {
					t.Fatal(result, err)
				}
			} else if err == nil {
				t.Fatal(result)
			}
			if mode == "delete404" && (result == nil || len(result.Responses) != 1 || result.Responses[0].StatusCode != 404) {
				t.Fatal(result, err)
			}
		})
	}
}
func TestPortBindingActivationMissingAndCapturedOptions(t *testing.T) {
	var calls []call
	headers := map[string]string{"X-Owned": "yes"}
	query := json.RawMessage(`"h"`)
	option := sdk.WithOptions(sdk.Options{Headers: headers, Query: map[string]json.RawMessage{"host": query}})
	headers["X-Owned"] = "mutated"
	query[1] = 'X'
	service := api(t, &calls, func(r *http.Request) *http.Response {
		if r.Header.Get("X-Owned") != "yes" {
			t.Fatal(r.Header)
		}
		if strings.Contains(r.URL.Path, "/bindings") {
			return wire(200, `{"bindings":[]}`)
		}
		return wire(200, `{"port":{"id":"p"}}`)
	})
	result, err := service.Activate(context.Background(), "p", "h", option)
	if err != nil || result == nil || result.Binding != nil || len(calls) != 3 {
		t.Fatal(result, err, calls)
	}
	before := len(calls)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err = service.Create(ctx, "p", func(c *sdk.Options) error { cancel(); return nil })
	if !errors.Is(err, context.Canceled) || len(calls) != before {
		t.Fatal(err, calls)
	}
	_, err = service.Create(context.Background(), "p", func(c *sdk.Options) error { service.RawClient().Endpoint += "changed/"; return nil })
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
		t.Fatal(err, calls)
	}
}

type faultBody struct {
	io.Reader
	read, close error
	onClose     func()
}

func (b *faultBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	if err == io.EOF && b.read != nil {
		return n, b.read
	}
	return n, err
}
func (b *faultBody) Close() error {
	if b.onClose != nil {
		b.onClose()
	}
	return b.close
}
func TestPortBindingPhysicalFaultNeverBecomesIgnoredDeletion(t *testing.T) {
	for _, mode := range []string{"read", "close", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			var calls []call
			cause := errors.New("physical " + mode)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			service := api(t, &calls, func(r *http.Request) *http.Response {
				response := wire(404, `{"error":"physical"}`)
				body := &faultBody{Reader: strings.NewReader(`{"error":"physical"}`)}
				if mode == "read" {
					body.read = cause
				}
				if mode == "close" {
					body.close = cause
				}
				if mode == "cancel" {
					body.onClose = func() { cancel(cause) }
				}
				response.Body = body
				return response
			})
			result, err := service.Delete(ctx, "p", "h")
			if !errors.Is(err, cause) || len(calls) != 1 || result != nil && result.IgnoredMissing {
				t.Fatal(result, err, calls)
			}
		})
	}
}
func TestPortBindingPreflightAndCreateCollisions(t *testing.T) {
	for _, mode := range []string{"nil-option", "invalid-port", "invalid-host", "duplicate-port", "invalid-json"} {
		t.Run(mode, func(t *testing.T) {
			var calls []call
			service := api(t, &calls, func(*http.Request) *http.Response { return wire(200, `{"port":{"id":"p"}}`) })
			var err error
			switch mode {
			case "nil-option":
				_, err = service.Create(context.Background(), "p", nil)
			case "invalid-port":
				_, err = service.Create(context.Background(), "..")
			case "invalid-host":
				_, err = service.Activate(context.Background(), "p", "..")
			case "duplicate-port":
				_, err = service.Create(context.Background(), "p", sdk.WithAttribute("port_id", "other"))
			case "invalid-json":
				_, err = service.Create(context.Background(), "p", sdk.WithOptions(sdk.Options{Attributes: map[string]json.RawMessage{"profile": json.RawMessage(`invalid`)}}))
			}
			if err == nil {
				t.Fatal(calls)
			}
			if len(calls) > 1 {
				t.Fatal("no binding request on preflight failure", calls)
			}
		})
	}
}
