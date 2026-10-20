package domains_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func domainMutationBody(t *testing.T, r *http.Request) map[string]json.RawMessage {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope) != 1 || envelope["domain"] == nil {
		t.Fatal(string(raw))
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(envelope["domain"], &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func TestPythonDomainCreateRecordNormalizesAliasesWithoutSendingDefaults(t *testing.T) {
	calls := 0
	api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.Method != "POST" || r.URL.Path != "/keystone/v3/domains" || r.URL.RawQuery != "" {
			t.Fatal(r.Method, r.URL)
		}
		body := domainMutationBody(t, r)
		want := map[string]json.RawMessage{"name": json.RawMessage(`"demo"`), "enabled": json.RawMessage(`0`), "description": json.RawMessage(`{"n":9007199254740993}`), "options": json.RawMessage(`7`)}
		if !reflect.DeepEqual(body, want) {
			t.Fatal(body, want)
		}
		return domainRecordWire(201, `{"domain":{"id":"created","options":{"actual":9007199254740993},"x_unknown":true}}`)
	})
	row, err := api.CreateRecord(context.Background(), domains.WithRecordMutationAttribute("name", "demo"), domains.WithRecordMutationAttribute("is_enabled", true), domains.WithRecordMutationAttribute("enabled", 0), domains.WithRecordMutationAttribute("description", json.RawMessage(`{"n":9007199254740993}`)), domains.WithRecordMutationAttribute("options", 7), domains.WithRecordMutationAttribute("x_unknown", "discard"))
	if err != nil || row == nil || calls != 1 || row.StatusCode != 201 {
		t.Fatal(row, err, calls)
	}
	domainRecordField(t, row, "id", `"created"`)
	domainRecordField(t, row, "is_enabled", "false")
	domainRecordField(t, row, "options", `{"actual":9007199254740993}`)
	if row.Resource.Body["x_unknown"] != nil || string(row.Wire.Body["x_unknown"]) != "true" {
		t.Fatal(row)
	}
}

func TestPythonDomainCreateRecordDistinguishesOmittedAndExplicitNull(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(map[bool]string{false: "omitted", true: "explicit null"}[explicit], func(t *testing.T) {
			calls := 0
			api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				body := domainMutationBody(t, r)
				if explicit {
					if len(body) != 2 || string(body["description"]) != "null" || string(body["options"]) != "null" {
						t.Fatal(body)
					}
				} else if len(body) != 0 {
					t.Fatal(body)
				}
				return domainRecordWire(200, `{"domain":{"id":"d"}}`)
			})
			options := []domains.RecordMutationOption{}
			if explicit {
				options = append(options, domains.WithRecordMutationAttribute("description", nil), domains.WithRecordMutationAttribute("options", nil))
			}
			row, err := api.CreateRecord(context.Background(), options...)
			if err != nil || row == nil || calls != 1 {
				t.Fatal(row, err, calls)
			}
			domainRecordField(t, row, "description", "null")
			domainRecordField(t, row, "options", "null")
		})
	}
}

func TestPythonDomainUpdateRecordPATCHPreservesSeedAndActualResponse(t *testing.T) {
	calls := 0
	body := `{"domain":{"id":7,"name":"server","options":{"actual":9007199254740993},"enabled":null,"x_extension":{"p":1}}}`
	api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.Method != "PATCH" || r.URL.EscapedPath() != "/keystone/v3/domains/domain%2Fid" {
			t.Fatal(r.Method, r.URL)
		}
		attrs := domainMutationBody(t, r)
		if len(attrs) != 3 || string(attrs["description"]) != `"seed"` || string(attrs["enabled"]) != `"yes"` || string(attrs["options"]) != `{"request":1}` {
			t.Fatal(attrs)
		}
		return domainRecordWire(203, body)
	})
	row, err := api.UpdateRecord(context.Background(), "domain/id", domains.WithRecordMutationAttribute("description", "seed"), domains.WithRecordMutationAttribute("is_enabled", "yes"), domains.WithRecordMutationAttribute("options", map[string]any{"request": 1}))
	if err != nil || row == nil || calls != 1 || row.StatusCode != 203 || string(row.Envelope) != body || row.Header.Get("X-Request-Id") != "domain-proof" {
		t.Fatal(row, err, calls)
	}
	for k, w := range map[string]string{"id": "7", "name": `"server"`, "description": `"seed"`, "options": `{"actual":9007199254740993}`, "is_enabled": "null"} {
		domainRecordField(t, row, k, w)
	}
	if row.Resource.Body["x_extension"] != nil || string(row.Wire.Body["x_extension"]) != `{"p":1}` {
		t.Fatal(row)
	}
}

func TestPythonDomainUpdateRecordEmptyOrUnknownAttrsAvoidHTTP(t *testing.T) {
	for _, mode := range []string{"empty", "unknown only", "id only"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response { calls++; return domainRecordWire(200, `{}`) })
			options := []domains.RecordMutationOption{}
			if mode == "unknown only" {
				options = append(options, domains.WithRecordMutationAttribute("x_unknown", 1))
			}
			if mode == "id only" {
				options = append(options, domains.WithRecordMutationAttribute("id", "fixed"))
			}
			row, err := api.UpdateRecord(context.Background(), "fixed", options...)
			if mode == "id only" {
				if row != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
					t.Fatal(row, err, calls)
				}
				return
			}
			if err != nil || row == nil || calls != 0 || row.StatusCode != 0 || len(row.Envelope) != 0 || row.Wire != nil {
				t.Fatal(row, err, calls)
			}
			domainRecordField(t, row, "id", `"fixed"`)
		})
	}
}

func TestPythonDomainUpdateRecordExplicitNullIsSentAsDirty(t *testing.T) {
	calls := 0
	api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		attrs := domainMutationBody(t, r)
		if r.Method != "PATCH" || len(attrs) != 3 || string(attrs["enabled"]) != "null" || string(attrs["description"]) != "null" || string(attrs["options"]) != "null" {
			t.Fatal(r.Method, attrs)
		}
		return domainRecordWire(200, `{"domain":{}}`)
	})
	row, err := api.UpdateRecord(context.Background(), "fixed", domains.WithRecordMutationAttribute("description", nil), domains.WithRecordMutationAttribute("options", nil), domains.WithRecordMutationAttribute("is_enabled", nil))
	if err != nil || row == nil || calls != 1 {
		t.Fatal(row, err, calls)
	}
	domainRecordField(t, row, "is_enabled", "null")
	domainRecordField(t, row, "description", "null")
	domainRecordField(t, row, "options", "null")
}

func TestPythonDomainRecordMutationsOwnOptionsHeadersVersionAndLocation(t *testing.T) {
	attrs := map[string]json.RawMessage{"name": json.RawMessage(`"original"`), "options": json.RawMessage(`{"n":9007199254740993}`)}
	headers := map[string]string{"X-Owned": "original"}
	version := "3.9"
	cloudName := "configured"
	location := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`)}}
	option := domains.WithRecordMutationOpts(domains.RecordMutationOpts{Attributes: attrs, Headers: headers, Microversion: &version, Location: &location})
	copy(attrs["name"], []byte(`"mutated!"`))
	attrs["options"] = json.RawMessage(`{}`)
	headers["X-Owned"] = "changed"
	version = "3.1"
	cloudName = "changed"
	calls := 0
	api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		body := domainMutationBody(t, r)
		if string(body["name"]) != `"original"` || string(body["options"]) != `{"n":9007199254740993}` || r.Header.Get("X-Owned") != "original" || r.Header.Get("OpenStack-API-Version") != "identity 3.9" {
			t.Fatal(body, r.Header)
		}
		return domainRecordWire(200, `{"domain":{"id":"new","location":{"cloud":"forged"}}}`)
	})
	row, err := api.CreateRecord(context.Background(), option)
	if err != nil || row == nil || calls != 1 {
		t.Fatal(row, err, calls)
	}
	var loc map[string]json.RawMessage
	if err := json.Unmarshal(row.Resource.Body["location"], &loc); err != nil || string(loc["cloud"]) != `"configured"` {
		t.Fatal(loc, err)
	}
	if string(row.Wire.Body["location"]) != `{"cloud":"forged"}` {
		t.Fatal(row.Wire)
	}
}

func TestPythonDomainRecordMutationsPreflightAndOptionLifecycle(t *testing.T) {
	for _, mode := range []string{"nil option", "callback error", "selected API changed", "cancel", "invalid id"} {
		t.Run(mode, func(t *testing.T) {
			calls, later := 0, 0
			cause := errors.New("domain mutation preflight")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			api, fixture := domainRecordAPI(t, func(r *http.Request) *http.Response { calls++; return domainRecordWire(200, `{}`) })
			var option domains.RecordMutationOption
			if mode != "nil option" {
				option = func(*domains.RecordMutationOpts) error {
					switch mode {
					case "callback error":
						return cause
					case "selected API changed":
						*api = *domains.New(fixture.Client("identity", "/replacement/"))
					case "cancel":
						cancel(cause)
					}
					return nil
				}
			}
			id := "fixed"
			if mode == "invalid id" {
				id = ".."
			}
			row, err := api.UpdateRecord(ctx, id, option, func(*domains.RecordMutationOpts) error { later++; return nil })
			if row != nil || err == nil || calls != 0 {
				t.Fatal(row, err, calls)
			}
			if mode != "invalid id" && later != 0 {
				t.Fatal(later)
			}
			if mode == "callback error" || mode == "cancel" {
				if !errors.Is(err, cause) {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestPythonDomainRecordMutationsPhysicalFaultsAndRejectionsDoNotFallback(t *testing.T) {
	for _, method := range []string{"create", "update"} {
		for _, mode := range []string{"read", "close", "cancel", "clean404"} {
			t.Run(method+"/"+mode, func(t *testing.T) {
				calls, closed := 0, 0
				cause := errors.New("domain mutation " + mode)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response {
					calls++
					reply := domainRecordWire(404, `{"error":"proof"}`)
					if mode != "clean404" {
						body := &domainFaultBody{closes: &closed}
						if mode == "read" {
							body.readErr = cause
						}
						if mode == "close" {
							body.closeErr = cause
						}
						if mode == "cancel" {
							body.close = func() { cancel(cause) }
						}
						reply.Body = body
					}
					return reply
				})
				var err error
				if method == "create" {
					_, err = api.CreateRecord(ctx, domains.WithRecordMutationAttribute("name", "n"))
				} else {
					_, err = api.UpdateRecord(ctx, "fixed", domains.WithRecordMutationAttribute("description", "d"))
				}
				if err == nil || calls != 1 {
					t.Fatal(err, calls)
				}
				if mode == "clean404" {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != 404 || !strings.Contains(string(native.Body), "proof") {
						t.Fatal(err, native)
					}
				} else if !errors.Is(err, cause) || closed != 1 {
					t.Fatal(err, closed)
				}
			})
		}
	}
}
