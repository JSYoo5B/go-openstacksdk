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

func domainConfigPayload(t *testing.T, r *http.Request) map[string]json.RawMessage {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	var outer map[string]json.RawMessage
	if err := json.Unmarshal(raw, &outer); err != nil || len(outer) != 1 || outer["config"] == nil {
		t.Fatal(string(raw), err)
	}
	var attrs map[string]json.RawMessage
	if err := json.Unmarshal(outer["config"], &attrs); err != nil {
		t.Fatal(err)
	}
	return attrs
}
func domainConfigField(t *testing.T, row *domains.DomainConfigRecord, key, want string) {
	t.Helper()
	if row == nil || row.Resource == nil || string(row.Resource.Body[key]) != want {
		t.Fatalf("%s want %s: %+v", key, want, row)
	}
}
func domainConfigChild(t *testing.T, row *domains.DomainConfigRecord, key string, want map[string]string) {
	t.Helper()
	if row == nil || row.Resource == nil {
		t.Fatal(row)
	}
	var child map[string]json.RawMessage
	if err := json.Unmarshal(row.Resource.Body[key], &child); err != nil {
		t.Fatal(row, err)
	}
	if len(child) != len(want) {
		t.Fatal(key, child, want)
	}
	for k, v := range want {
		if string(child[k]) != v {
			t.Fatal(key, k, child, want)
		}
	}
}

func TestPythonDomainConfigCreateUsesSingletonPUTAndRawChildPayload(t *testing.T) {
	calls := 0
	api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.Method != "PUT" || r.URL.EscapedPath() != "/keystone/v3/domains/domain%2Fid/config" || r.URL.RawQuery != "" {
			t.Fatal(r.Method, r.URL)
		}
		attrs := domainConfigPayload(t, r)
		want := map[string]json.RawMessage{"identity": json.RawMessage(`{"driver":"ldap","x_child":9007199254740993}`), "ldap": json.RawMessage(`{"url":"ldap://local","user_tree_dn":"ou=users"}`)}
		if !reflect.DeepEqual(attrs, want) {
			t.Fatal(attrs, want)
		}
		return domainRecordWire(201, `{"config":{"identity":{"driver":"actual","x_child":true},"ldap":{"url":"ldap://actual"},"x_unknown":1}}`)
	})
	row, err := api.CreateConfigRecord(context.Background(), "domain/id", domains.WithDomainConfigAttribute("identity", json.RawMessage(`{"driver":"ldap","x_child":9007199254740993}`)), domains.WithDomainConfigAttribute("ldap", map[string]any{"url": "ldap://local", "user_tree_dn": "ou=users"}), domains.WithDomainConfigAttribute("x_unknown", "drop"))
	if err != nil || row == nil || calls != 1 || row.StatusCode != 201 {
		t.Fatal(row, err, calls)
	}
	domainConfigChild(t, row, "identity", map[string]string{"driver": `"actual"`, "id": "null", "name": "null", "location": "null"})
	domainConfigChild(t, row, "ldap", map[string]string{"url": `"ldap://actual"`, "user_tree_dn": "null", "id": "null", "name": "null", "location": "null"})
	if row.Resource.Body["domain_id"] != nil || row.Resource.Body["x_unknown"] != nil || string(row.Wire.Body["x_unknown"]) != "1" {
		t.Fatal(row)
	}
}

func TestPythonDomainConfigGetProjectsChildrenAndPreservesActualEvidence(t *testing.T) {
	calls := 0
	body := `{"config":{"identity":{"driver":"sql","x_unknown":7},"ldap":{"url":"ldap://local","user_tree_dn":false},"id":7,"location":{"cloud":"forged"}}}`
	api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.Method != "GET" || r.URL.Path != "/keystone/v3/domains/fixed/config" || r.Body != nil {
			t.Fatal(r.Method, r.URL, r.Body)
		}
		return domainRecordWire(203, body)
	})
	row, err := api.GetConfigRecord(context.Background(), "fixed")
	if err != nil || row == nil || calls != 1 || row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "domain-proof" || string(row.Envelope) != body {
		t.Fatal(row, err, calls)
	}
	domainConfigChild(t, row, "identity", map[string]string{"driver": `"sql"`, "id": "null", "name": "null", "location": "null"})
	domainConfigChild(t, row, "ldap", map[string]string{"url": `"ldap://local"`, "user_tree_dn": "false", "id": "null", "name": "null", "location": "null"})
	domainConfigField(t, row, "id", "7")
	domainConfigField(t, row, "location", "null")
	row.Resource.Body["identity"][0] = '['
	row.Header.Set("X-Request-Id", "caller")
	row.Envelope[0] = '['
	if string(row.Wire.Body["identity"]) != `{"driver":"sql","x_unknown":7}` || row.Wire.Header.Get("X-Request-Id") != "domain-proof" {
		t.Fatal(row)
	}
}

func TestPythonDomainConfigGetNullScalarAndEmptyChildrenHaveDistinctViews(t *testing.T) {
	for _, tc := range []struct{ name, child, want string }{{"null", "null", "null"}, {"scalar", "7", "{}"}, {"list", "[1]", "{}"}, {"false", "false", "{}"}} {
		t.Run(tc.name, func(t *testing.T) {
			api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response {
				return domainRecordWire(200, `{"config":{"identity":`+tc.child+`,"ldap":`+tc.child+`}}`)
			})
			row, err := api.GetConfigRecord(context.Background(), "fixed")
			if err != nil || row == nil {
				t.Fatal(row, err)
			}
			domainConfigField(t, row, "identity", tc.want)
			domainConfigField(t, row, "ldap", tc.want)
		})
	}
	t.Run("empty mappings materialize declared defaults", func(t *testing.T) {
		api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response {
			return domainRecordWire(200, `{"config":{"identity":{},"ldap":{}}}`)
		})
		row, err := api.GetConfigRecord(context.Background(), "fixed")
		if err != nil || row == nil {
			t.Fatal(row, err)
		}
		domainConfigChild(t, row, "identity", map[string]string{"driver": "null", "id": "null", "name": "null", "location": "null"})
		domainConfigChild(t, row, "ldap", map[string]string{"url": "null", "user_tree_dn": "null", "id": "null", "name": "null", "location": "null"})
	})
}

func TestPythonDomainConfigUpdatePATCHNullAndSeedOverlay(t *testing.T) {
	calls := 0
	api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.Method != "PATCH" || r.URL.Path != "/keystone/v3/domains/fixed/config" {
			t.Fatal(r.Method, r.URL)
		}
		attrs := domainConfigPayload(t, r)
		if len(attrs) != 2 || string(attrs["ldap"]) != "null" || string(attrs["identity"]) != `{"driver":"seed"}` {
			t.Fatal(attrs)
		}
		return domainRecordWire(200, `{"config":{"ldap":{"url":"actual"}}}`)
	})
	row, err := api.UpdateConfigRecord(context.Background(), "fixed", domains.WithDomainConfigAttribute("identity", map[string]any{"driver": "seed"}), domains.WithDomainConfigAttribute("ldap", nil))
	if err != nil || row == nil || calls != 1 {
		t.Fatal(row, err, calls)
	}
	domainConfigChild(t, row, "identity", map[string]string{"driver": `"seed"`, "id": "null", "name": "null", "location": "null"})
	domainConfigChild(t, row, "ldap", map[string]string{"url": `"actual"`, "user_tree_dn": "null", "id": "null", "name": "null", "location": "null"})
}

func TestPythonDomainConfigUpdateEmptyAndUnknownAttributesAvoidHTTP(t *testing.T) {
	for _, mode := range []string{"empty", "unknown only", "id only"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response { calls++; return domainRecordWire(200, `{}`) })
			opts := []domains.DomainConfigOption{}
			if mode == "unknown only" {
				opts = append(opts, domains.WithDomainConfigAttribute("x_unknown", 1))
			}
			if mode == "id only" {
				opts = append(opts, domains.WithDomainConfigAttribute("id", "passive"))
			}
			row, err := api.UpdateConfigRecord(context.Background(), "fixed", opts...)
			if err != nil || row == nil || calls != 0 || row.StatusCode != 0 || len(row.Envelope) != 0 || row.Wire != nil {
				t.Fatal(row, err, calls)
			}
			if mode == "id only" {
				domainConfigField(t, row, "id", `"passive"`)
			}
			domainConfigField(t, row, "identity", "null")
			domainConfigField(t, row, "ldap", "null")
		})
	}
}

func TestPythonDomainConfigDeleteMissingDefaultAndStrictPolicy(t *testing.T) {
	for _, strict := range []bool{false, true} {
		t.Run(map[bool]string{false: "ignore default", true: "strict"}[strict], func(t *testing.T) {
			calls := 0
			api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				if r.Method != "DELETE" || r.URL.Path != "/keystone/v3/domains/fixed/config" || r.Body != nil {
					t.Fatal(r.Method, r.URL, r.Body)
				}
				return domainRecordWire(404, `{"error":"missing"}`)
			})
			options := []domains.DomainConfigOption{}
			if strict {
				options = append(options, domains.WithDomainConfigIgnoreMissing(false))
			}
			result, err := api.DeleteConfigRecord(context.Background(), "fixed", options...)
			if calls != 1 {
				t.Fatal(calls)
			}
			if result == nil || result.Deleted || result.Response == nil || result.Response.StatusCode != 404 || result.Response.Header.Get("X-Request-Id") != "domain-proof" || string(result.Response.Envelope) != `{"error":"missing"}` || result.Response.Resource != nil || result.Response.Wire != nil {
				t.Fatal("actual missing receipt lost", result, err)
			}
			if strict {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 404 {
					t.Fatal(result, err, native)
				}
				return
			}
			if err != nil || result == nil || result.Deleted {
				t.Fatal(result, err)
			}
		})
	}
}

func TestPythonDomainConfigDeleteAcceptedResponseIsIndependentReceipt(t *testing.T) {
	calls := 0
	body := `{"ack":"deleted"}`
	api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.Method != "DELETE" {
			t.Fatal(r.Method)
		}
		return domainRecordWire(202, body)
	})
	result, err := api.DeleteConfigRecord(context.Background(), "fixed")
	if err != nil || result == nil || !result.Deleted || result.Response == nil || calls != 1 {
		t.Fatal(result, err, calls)
	}
	row := result.Response
	if row.Resource != nil || row.Wire != nil || row.StatusCode != 202 || string(row.Envelope) != body || row.Header.Get("X-Request-Id") != "domain-proof" {
		t.Fatal(row)
	}
}

func TestPythonDomainConfigOptionsSnapshotsAndRequestPolicy(t *testing.T) {
	attrs := map[string]json.RawMessage{"identity": json.RawMessage(`{"driver":"original"}`)}
	headers := map[string]string{"X-Owned": "original"}
	version := "3.9"
	option := domains.WithDomainConfigOpts(domains.DomainConfigOpts{Attributes: attrs, Headers: headers, Microversion: &version})
	attrs["identity"] = json.RawMessage(`{"driver":"mutated"}`)
	headers["X-Owned"] = "changed"
	version = "3.1"
	calls := 0
	api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		payload := domainConfigPayload(t, r)
		if string(payload["identity"]) != `{"driver":"original"}` || r.Header.Get("X-Owned") != "original" || r.Header.Get("OpenStack-API-Version") != "identity 3.9" {
			t.Fatal(payload, r.Header)
		}
		return domainRecordWire(200, `{"config":{}}`)
	})
	for i := 0; i < 2; i++ {
		row, err := api.CreateConfigRecord(context.Background(), "fixed", option)
		if err != nil || row == nil {
			t.Fatal(row, err)
		}
	}
	if calls != 2 {
		t.Fatal(calls)
	}
}

func TestPythonDomainConfigPreflightGuardsAndReadMutationSeparation(t *testing.T) {
	for _, mode := range []string{"nil option", "callback error", "source changed", "cancel", "invalid id", "get attributes", "delete attributes"} {
		t.Run(mode, func(t *testing.T) {
			calls, later := 0, 0
			cause := errors.New("config preflight")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			api, cloud := domainRecordAPI(t, func(r *http.Request) *http.Response { calls++; return domainRecordWire(200, `{}`) })
			var option domains.DomainConfigOption
			if mode != "nil option" {
				option = func(*domains.DomainConfigOpts) error {
					switch mode {
					case "callback error":
						return cause
					case "source changed":
						*api = *domains.New(cloud.Client("identity", "/replacement/"))
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
			var err error
			if mode == "get attributes" {
				_, err = api.GetConfigRecord(ctx, id, domains.WithDomainConfigAttribute("identity", nil))
			} else if mode == "delete attributes" {
				_, err = api.DeleteConfigRecord(ctx, id, domains.WithDomainConfigAttribute("ldap", nil))
			} else {
				_, err = api.CreateConfigRecord(ctx, id, option, func(*domains.DomainConfigOpts) error { later++; return nil })
			}
			if err == nil || calls != 0 {
				t.Fatal(err, calls)
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

func TestPythonDomainConfigPhysicalFaultsAndStrictReadErrorsNeverFallback(t *testing.T) {
	for _, op := range []string{"create", "get", "update", "delete"} {
		for _, mode := range []string{"read", "close", "cancel", "clean404"} {
			t.Run(op+"/"+mode, func(t *testing.T) {
				calls, closes := 0, 0
				cause := errors.New("config physical " + mode)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response {
					calls++
					reply := domainRecordWire(404, `{"error":"missing proof"}`)
					if mode != "clean404" {
						fault := &domainFaultBody{closes: &closes}
						if mode == "read" {
							fault.readErr = cause
						}
						if mode == "close" {
							fault.closeErr = cause
						}
						if mode == "cancel" {
							fault.close = func() { cancel(cause) }
						}
						reply.Body = fault
					}
					return reply
				})
				var err error
				switch op {
				case "create":
					_, err = api.CreateConfigRecord(ctx, "fixed", domains.WithDomainConfigAttribute("identity", nil))
				case "get":
					_, err = api.GetConfigRecord(ctx, "fixed")
				case "update":
					_, err = api.UpdateConfigRecord(ctx, "fixed", domains.WithDomainConfigAttribute("ldap", nil))
				case "delete":
					_, err = api.DeleteConfigRecord(ctx, "fixed", domains.WithDomainConfigIgnoreMissing(false))
				}
				if err == nil || calls != 1 {
					t.Fatal(err, calls)
				}
				if mode == "clean404" {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != 404 || !strings.Contains(string(native.Body), "missing proof") {
						t.Fatal(err, native)
					}
				} else if !errors.Is(err, cause) || closes != 1 {
					t.Fatal(err, closes)
				}
			})
		}
	}
}

func TestPythonDomainConfigMalformedNestedChildKeepsAcceptedResponse(t *testing.T) {
	body := `{"config":{"identity":{"self":"constructor collision"}}}`
	api, _ := domainRecordAPI(t, func(r *http.Request) *http.Response { return domainRecordWire(203, body) })
	_, err := api.GetConfigRecord(context.Background(), "fixed")
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "domain-proof" {
		t.Fatal(err, proof)
	}
}
