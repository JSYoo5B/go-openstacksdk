package serviceinfo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

// Reuse the established service-info transport/body fixtures. This adapter adds
// only the Close callback needed to verify the shared read-boundary guard.
type infoRecordCloseBody struct {
	*infoBody
	after func()
}

func (body *infoRecordCloseBody) Close() error {
	err := body.infoBody.Close()
	if body.after != nil {
		body.after()
	}
	return err
}
func infoRecordHTTP(req *http.Request, code int, body io.ReadCloser) *http.Response {
	wire := infoResponse(req, body)
	wire.StatusCode = code
	return wire
}
func infoRecordJSON(req *http.Request, code int, raw string) *http.Response {
	return infoRecordHTTP(req, code, &infoBody{reader: strings.NewReader(raw)})
}
func infoRecordProof(t *testing.T, err error, code int, raw string) *resource.ResponseError {
	t.Helper()
	var receipt *resource.ResponseError
	if !errors.As(err, &receipt) || receipt.StatusCode != code || string(receipt.Body) != raw || receipt.Header.Get("X-Actual") != "owned" {
		t.Fatalf("actual receipt code=%d raw=%q err=%v receipt=%+v", code, raw, err, receipt)
	}
	return receipt
}
func infoRecordValues(value *resource.RawResource) map[string]string {
	result := make(map[string]string)
	if value != nil {
		for key, raw := range value.Body {
			result[key] = string(raw)
		}
	}
	return result
}

func TestImportInfoRecordProjectionAndOwnedReceipts(t *testing.T) {
	const raw = `{"id":900719925474099312345,"name":["passive",null],"import-methods":{"description":false,"type":[1],"value":{"future":null},"number":1.00000000000000000001},"location":{"cloud":"wire"},"self":"https://foreign.test/","vendor":1e400}`
	calls := 0
	var actualHeader http.Header
	client := infoClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodGet || req.URL.String() != "https://example.test/reverse/glance/v2/info/import" || req.URL.RawQuery != "" || req.Body != nil {
			t.Fatal("ID-free import discovery", req.Method, req.URL, req.Body)
		}
		wire := infoRecordJSON(req, 203, raw)
		actualHeader = wire.Header
		return wire, nil
	})
	got, err := New(client).GetImportInfoRecord(context.Background())
	want := map[string]string{"id": "900719925474099312345", "name": `["passive",null]`, "import_methods": `{"description":false,"type":[1],"value":{"future":null},"number":1.00000000000000000001}`, "location": "null"}
	if got == nil || err != nil || calls != 1 || got.Resource == nil || got.Wire == nil {
		t.Fatal(got, err, calls)
	}
	th.CheckDeepEquals(t, want, infoRecordValues(got.Resource))
	for _, value := range []*resource.RawResource{got.Resource, got.Wire} {
		if value.StatusCode != 203 || value.Header.Get("X-Actual") != "owned" {
			t.Fatal("resource receipt", value)
		}
	}
	if got.StatusCode != 203 || got.Header.Get("X-Actual") != "owned" || string(got.Envelope) != raw || string(got.Wire.Body["vendor"]) != "1e400" || string(got.Wire.Body["self"]) != `"https://foreign.test/"` || string(got.Wire.Body["location"]) != `{"cloud":"wire"}` {
		t.Fatal("raw receipt", got)
	}
	got.Resource.Body["import_methods"][0] = '!'
	got.Resource.Header.Set("X-Actual", "view changed")
	got.Wire.Body["id"][0] = '0'
	got.Wire.Header.Set("X-Actual", "wire changed")
	got.Header.Set("X-Actual", "record changed")
	if string(got.Envelope) != raw || string(got.Resource.Body["id"]) != "900719925474099312345" || actualHeader.Get("X-Actual") != "owned" || string(got.Wire.Body["import-methods"]) != want["import_methods"] {
		t.Fatal("receipt channel alias")
	}
	got.Envelope[0] = '!'
	if actualHeader.Get("X-Actual") != "owned" {
		t.Fatal("wire Header aliases record")
	}
}

func TestImportInfoRecordDictionaryAliasesAndDefaults(t *testing.T) {
	for _, test := range []struct{ name, raw, want string }{
		{"absent", `{}`, "null"},
		{"explicit null", `{"import-methods":null}`, "null"},
		{"canonical alias", `{"import_methods":{"canonical":true}}`, `{"canonical":true}`},
		{"remote later", `{"import_methods":{"first":1},"import-methods":{"last":2}}`, `{"last":2}`},
		{"canonical later", `{"import-methods":{"first":1},"import_methods":{"last":2}}`, `{"last":2}`},
		{"duplicate retains first insertion position", `{"import-methods":{"first":1},"import_methods":{"canonical":2},"import-methods":{"last":3}}`, `{"canonical":2}`},
		{"empty dictionary", `{"import-methods":{}}`, `{}`},
		{"nonnull string", `{"import-methods":"opaque"}`, `{}`},
		{"nonnull array", `{"import-methods":[1,null]}`, `{}`},
		{"nonnull bool", `{"import-methods":false}`, `{}`},
		{"nonnull number", `{"import-methods":1e400}`, `{}`},
		{"nested untyped members", `{"import-methods":{"value":[null,42,false,{"number":900719925474099312345}]}}`, `{"value":[null,42,false,{"number":900719925474099312345}]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := infoClient(func(req *http.Request) (*http.Response, error) { return infoRecordJSON(req, 200, test.raw), nil })
			got, err := New(client).GetImportInfoRecord(context.Background())
			if got == nil || err != nil || len(got.Resource.Body) != 4 {
				t.Fatal(got, err)
			}
			th.AssertEquals(t, test.want, string(got.Resource.Body["import_methods"]))
			th.AssertEquals(t, "null", string(got.Resource.Body["id"]))
			th.AssertEquals(t, "null", string(got.Resource.Body["name"]))
			th.AssertEquals(t, test.raw, string(got.Envelope))
		})
	}
}

func TestImportInfoRecordAcceptedToleranceAndRejectedNative(t *testing.T) {
	for _, test := range []struct {
		code         int
		raw          string
		parsed, good bool
	}{
		{200, `{}`, true, true}, {201, `{"import-methods":null}`, true, true}, {299, `{}`, true, true}, {300, `{"links":[{"href":"https://foreign.test/"}]}`, true, true}, {399, `{}`, true, true},
		{204, "", false, true}, {202, " \n\t ", false, true}, {304, "not JSON", false, true}, {200, `{"broken":`, false, true},
		{200, `null`, true, false}, {200, `[]`, true, false}, {200, `false`, true, false}, {200, `42`, true, false}, {200, `"text"`, true, false}, {200, "{\"unknown\":\"\xff\"}", true, false},
	} {
		t.Run(fmt.Sprintf("%d/%q", test.code, test.raw), func(t *testing.T) {
			calls, retries := 0, 0
			body := &infoBody{reader: strings.NewReader(test.raw)}
			client := infoClient(func(req *http.Request) (*http.Response, error) {
				calls++
				wire := infoRecordHTTP(req, test.code, body)
				wire.Header.Set("Content-Type", "text/plain")
				return wire, nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			got, err := New(client).GetImportInfoRecord(context.Background())
			if test.good {
				if got == nil || err != nil || (got.Wire != nil) != test.parsed || got.StatusCode != test.code || string(got.Envelope) != test.raw {
					t.Fatal(got, err)
				}
				for _, value := range got.Resource.Body {
					th.AssertEquals(t, "null", string(value))
				}
			} else {
				if got != nil || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(got, err)
				}
				infoRecordProof(t, err, test.code, test.raw)
			}
			if calls != 1 || retries != 0 || body.closes != 1 {
				t.Fatal("accepted body replay", calls, retries, body.closes)
			}
		})
	}
	for _, code := range []int{400, 403, 404, 500} {
		t.Run(fmt.Sprintf("native %d", code), func(t *testing.T) {
			calls := 0
			body := &infoBody{reader: strings.NewReader("native rejected import")}
			client := infoClient(func(req *http.Request) (*http.Response, error) { calls++; return infoRecordHTTP(req, code, body), nil })
			got, err := New(client).GetImportInfoRecord(context.Background())
			var native gophercloud.ErrUnexpectedResponseCode
			if got != nil || !errors.As(err, &native) || native.Actual != code || native.Method != http.MethodGet || native.URL != "https://example.test/reverse/glance/v2/info/import" || string(native.Body) != "native rejected import" || native.ResponseHeader.Get("X-Actual") != "owned" || calls != 1 || body.closes != 1 {
				t.Fatal(got, err, native, calls, body.closes)
			}
		})
	}
}

func TestImportInfoRecordLocationAndConcreteOptionSnapshots(t *testing.T) {
	cloud := "captured"
	facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"token project"`)}}
	headers := map[string]string{"X-Option": "factory snapshot"}
	option := WithImportRecordOpts(ImportRecordOpts{Headers: headers})
	headers["X-Option"] = "caller changed"
	calls, locations, callbacks := 0, 0, 0
	var retained *ImportRecordOpts
	client := infoClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Source") != "captured" || req.Header.Get("X-Option") != "factory snapshot" || req.Header.Get("X-Final") != "yes" || req.Header.Get("X-Auth-Token") != "live" || req.Header.Get("OpenStack-API-Version") != "image 2.10" {
			t.Fatal("source/options snapshot", req.Header)
		}
		retained.Headers["X-Option"] = "retained changed"
		return infoRecordJSON(req, 200, `{"location":{"cloud":"wire"},"import-methods":{}}`), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "captured"}
	client.Microversion = "2.10"
	api := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
		locations++
		client.MoreHeaders["X-Source"] = "location getter changed"
		return facts, nil
	}})
	got, err := api.GetImportInfoRecord(context.Background(), option, func(value *ImportRecordOpts) error {
		callbacks++
		retained = value
		cloud = "after option"
		facts.Project.ID[1] = 'X'
		client.SetToken("live")
		return WithImportRecordHeader("X-Final", "yes")(value)
	})
	if got == nil || err != nil || calls != 1 || locations != 1 || callbacks != 1 {
		t.Fatal(got, err, calls, locations, callbacks)
	}
	var captured resource.CloudLocation
	if err := json.Unmarshal(got.Resource.Body["location"], &captured); err != nil || captured.Cloud == nil || *captured.Cloud != "captured" || string(captured.Project.ID) != `"token project"` {
		t.Fatal("location ownership", captured, err)
	}
	th.AssertEquals(t, `{"cloud":"wire"}`, string(got.Wire.Body["location"]))
	t.Run("whole option reset and merged factory headers", func(t *testing.T) {
		headers := map[string]string{"X-Merged": "before"}
		merged := WithImportRecordHeaders(headers)
		headers["X-Merged"] = "after"
		client := infoClient(func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("X-Removed") != "" || req.Header.Get("X-Merged") != "before" || req.Header.Get("X-Final") != "yes" {
				t.Fatal(req.Header)
			}
			return infoRecordJSON(req, 200, `{}`), nil
		})
		got, err := New(client).GetImportInfoRecord(context.Background(), WithImportRecordHeader("X-Removed", "before"), WithImportRecordOpts(ImportRecordOpts{}), merged, WithImportRecordHeader("X-Final", "yes"))
		if got == nil || err != nil {
			t.Fatal(got, err)
		}
	})
}

func TestImportInfoRecordBodyFailuresAndStickySourceGuards(t *testing.T) {
	const raw = `{"import-methods":{"actual":true}}`
	for _, mode := range []string{"read", "close", "cancel", "source drift", "source restored at Close", "outer restored at Close"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New("import record body failure")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var client *gophercloud.ServiceClient
			calls, retries := 0, 0
			body := &infoBody{reader: strings.NewReader(raw)}
			outerInvalid := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerInvalid {
					return cause
				}
				return nil
			})
			action := func() {}
			switch mode {
			case "read":
				body.reader = infoReader(func(buf []byte) (int, error) { return copy(buf, raw), cause })
			case "close":
				body.closeErr = errors.Join(cause, gophercloud.ErrUnexpectedResponseCode{Actual: 404})
			case "cancel":
				action = func() { cancel(cause) }
			case "source drift", "source restored at Close":
				action = func() { client.ResourceBase = "https://example.test/changed/" }
			case "outer restored at Close":
				action = func() { outerInvalid = true }
			}
			if mode != "read" {
				body.reader = infoReader(func(buf []byte) (int, error) { action(); return copy(buf, raw), io.EOF })
			}
			selected := io.ReadCloser(body)
			if strings.Contains(mode, "restored") {
				selected = &infoRecordCloseBody{infoBody: body, after: func() { client.ResourceBase = "https://example.test/reverse/glance/v2/"; outerInvalid = false }}
			}
			client = infoClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return infoRecordHTTP(req, 201, selected), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			got, err := New(client).GetImportInfoRecord(ctx)
			if got != nil || err == nil || calls != 1 || retries != 0 || body.closes != 1 {
				t.Fatal(got, err, calls, retries, body.closes)
			}
			if strings.HasPrefix(mode, "source") {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, cause) {
				t.Fatal("cause lost", err)
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if errors.Is(err, resource.ErrNotFound) {
				t.Fatal("nested404 became absence", err)
			}
			infoRecordProof(t, err, 201, raw)
		})
	}
}

func TestImportInfoRecordPreflightAndLegacyABI(t *testing.T) {
	cause := errors.New("option cause")
	for _, test := range []struct {
		name   string
		option ImportRecordOption
		change func(*gophercloud.ServiceClient)
		want   error
	}{
		{name: "nil option", option: nil, want: resource.ErrInvalidOption},
		{name: "option cause", option: func(*ImportRecordOpts) error { return cause }, want: cause},
		{name: "owned auth", option: WithImportRecordHeader("X-Auth-Token", "foreign"), want: resource.ErrInvalidOption},
		{name: "header newline", option: WithImportRecordHeader("X-Extra", "\n"), want: resource.ErrInvalidOption},
		{name: "conflicting aliases", option: WithImportRecordOpts(ImportRecordOpts{Headers: map[string]string{"X-Extra": "first", "x-extra": "second"}}), want: resource.ErrInvalidOption},
		{name: "source missing provider", change: func(c *gophercloud.ServiceClient) { c.ProviderClient = nil }, option: WithImportRecordOpts(ImportRecordOpts{}), want: resource.ErrInvalidOption},
		{name: "wrong type", change: func(c *gophercloud.ServiceClient) { c.Type = "compute" }, option: WithImportRecordOpts(ImportRecordOpts{}), want: resource.ErrUnsupported},
		{name: "source microversion invalid", change: func(c *gophercloud.ServiceClient) { c.Microversion = "\n" }, option: WithImportRecordOpts(ImportRecordOpts{}), want: resource.ErrInvalidOption},
		{name: "nil context", option: WithImportRecordOpts(ImportRecordOpts{}), want: resource.ErrInvalidOption},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := infoClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil })
			if test.change != nil {
				test.change(client)
			}
			ctx := context.Background()
			if test.name == "nil context" {
				ctx = nil
			}
			got, err := New(client).GetImportInfoRecord(ctx, test.option)
			if got != nil || !errors.Is(err, test.want) || calls != 0 {
				t.Fatal(got, err, calls)
			}
		})
	}
	t.Run("source drift stops later option", func(t *testing.T) {
		calls, callbacks := 0, 0
		client := infoClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil })
		got, err := New(client).GetImportInfoRecord(context.Background(), func(*ImportRecordOpts) error { client.Microversion = "2.99"; return nil }, func(*ImportRecordOpts) error { callbacks++; client.Microversion = ""; return nil })
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 {
			t.Fatal(got, err, calls, callbacks)
		}
	})
	t.Run("canceled capture skips location and callbacks", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		locations, callbacks := 0, 0
		client := infoClient(func(req *http.Request) (*http.Response, error) { t.Fatal("HTTP after cancellation"); return nil, nil })
		api := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return resource.CloudLocation{}, nil }})
		got, err := api.GetImportInfoRecord(ctx, func(*ImportRecordOpts) error { callbacks++; return nil })
		if got != nil || !errors.Is(err, context.Canceled) || locations != 0 || callbacks != 0 {
			t.Fatal(got, err, locations, callbacks)
		}
	})
	t.Run("legacy import strict200 and nested fields unchanged", func(t *testing.T) {
		calls := 0
		client := infoClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return infoRecordJSON(req, 200, `{"import-methods":{"description":"legacy","type":"array","value":["future"]}}`), nil
			}
			return infoRecordJSON(req, 201, `{}`), nil
		})
		api := New(client)
		got, err := api.GetImportInfo(context.Background())
		if got == nil || err != nil || got.ImportMethods == nil || got.ImportMethods.Description != "legacy" || len(got.ImportMethods.Value) != 1 {
			t.Fatal(got, err)
		}
		got, err = api.GetImportInfo(context.Background())
		var native gophercloud.ErrUnexpectedResponseCode
		if got != nil || !errors.As(err, &native) || native.Actual != 201 || calls != 2 {
			t.Fatal(got, err, native, calls)
		}
	})
}
