package quotasets_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func quotaClassField(t *testing.T, row *quotasets.QuotaClassRecord, key, want string) {
	t.Helper()
	if row == nil || row.Resource == nil || string(row.Resource.Body[key]) != want {
		t.Fatalf("%s want %s: %+v", key, want, row)
	}
}

func TestNovaQuotaClassGetDefaultAndCustomNamesKeepLiteralRoutes(t *testing.T) {
	for _, name := range []string{"default", "custom/name %2F"} {
		t.Run(name, func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			cloud.Mux.HandleFunc("/compute/v2.1/", func(w http.ResponseWriter, r *http.Request) {
				calls++
				want := "/compute/v2.1/os-quota-class-sets/default"
				if name != "default" {
					want = "/compute/v2.1/os-quota-class-sets/custom%2Fname%20%252F"
				}
				if r.Method != "GET" || r.URL.EscapedPath() != want || r.URL.RawQuery != "" || r.Header.Get("OpenStack-API-Version") != "compute 2.96" {
					t.Error(r.Method, r.URL, r.Header)
				}
				w.Header().Set("X-Request-Id", "quota-class-proof")
				testcloud.JSON(w, 203, `{"quota_class_set":{"cores":"١٢","instances":true,"ram":-1.9,"fixed_ips":null,"key_pairs":9007199254740993,"floating_ips":1e2,"x_unknown":7}}`)
			})
			client := cloud.Client("compute", "/compute/v2.1/")
			client.Microversion = "2.96"
			opts := []quotasets.QuotaClassGetOption{}
			if name != "default" {
				opts = append(opts, quotasets.WithQuotaClassName(name))
			}
			row, err := quotasets.New(client).GetQuotaClassRecord(context.Background(), opts...)
			if err != nil || row == nil || calls != 1 || row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "quota-class-proof" || len(row.Discovery.Responses) != 0 {
				t.Fatal(row, err, calls)
			}
			for k, v := range map[string]string{"cores": "12", "instances": "true", "ram": "-1", "fixed_ips": "null", "key_pairs": "9007199254740993", "floating_ips": "100"} {
				quotaClassField(t, row, k, v)
			}
			if row.Resource.Body["x_unknown"] != nil || string(row.Wire.Body["cores"]) != `"١٢"` || string(row.Wire.Body["ram"]) != "-1.9" {
				t.Fatal(row)
			}
		})
	}
}

func TestNovaQuotaClassInvalidIntegerRetainsAcceptedResponse(t *testing.T) {
	cloud := testcloud.New(t)
	body := `{"quota_class_set":{"cores":"²"}}`
	calls := 0
	cloud.Mux.HandleFunc("/compute/v2.1/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("X-Request-Id", "bad-int")
		testcloud.JSON(w, 203, body)
	})
	client := cloud.Client("compute", "/compute/v2.1/")
	client.Microversion = "2.96"
	_, err := quotasets.New(client).GetQuotaClassRecord(context.Background())
	var proof *resource.ResponseError
	if calls != 1 || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "bad-int" {
		t.Fatal(err, proof, calls)
	}
}

func TestNovaQuotaClassUpdatePUTUsesRawAttributesAndProjectedResponse(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	cloud.Mux.HandleFunc("/compute/v2.1/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "PUT" || r.URL.Path != "/compute/v2.1/os-quota-class-sets/custom" {
			t.Error(r.Method, r.URL)
		}
		var payload map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		body := payload["quota_class_set"]
		if len(payload) != 1 || len(body) != 3 || string(body["cores"]) != `"12"` || string(body["ram"]) != "null" || string(body["instances"]) != "false" {
			t.Error(payload)
		}
		testcloud.JSON(w, 200, `{"quota_class_set":{"cores":"15","x_unknown":true}}`)
	})
	client := cloud.Client("compute", "/compute/v2.1/")
	client.Microversion = "2.96"
	row, err := quotasets.New(client).UpdateQuotaClassRecord(context.Background(), "custom", quotasets.WithQuotaClassUpdateAttribute("cores", "12"), quotasets.WithQuotaClassUpdateAttribute("ram", nil), quotasets.WithQuotaClassUpdateAttribute("instances", false), quotasets.WithQuotaClassUpdateAttribute("x_unknown", "discard"))
	if err != nil || row == nil || calls != 1 {
		t.Fatal(row, err, calls)
	}
	for k, v := range map[string]string{"id": `"custom"`, "cores": "15", "ram": "null", "instances": "false"} {
		quotaClassField(t, row, k, v)
	}
	if row.Resource.Body["x_unknown"] != nil || string(row.Wire.Body["x_unknown"]) != "true" {
		t.Fatal(row)
	}
}

func TestNovaQuotaClassUpdateNoopSkipsHTTPAndDiscoveryAndIDCollisions(t *testing.T) {
	for _, mode := range []string{"empty", "unknown only", "id collision"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls++; testcloud.JSON(w, 200, `{}`) })
			api := quotasets.New(cloud.Client("compute", "/compute/v2.1/"))
			opts := []quotasets.QuotaClassUpdateOption{}
			if mode == "unknown only" {
				opts = append(opts, quotasets.WithQuotaClassUpdateAttribute("x_unknown", 1))
			}
			if mode == "id collision" {
				opts = append(opts, quotasets.WithQuotaClassUpdateAttribute("id", "other"))
			}
			row, err := api.UpdateQuotaClassRecord(context.Background(), "custom", opts...)
			if calls != 0 {
				t.Fatal(calls)
			}
			if mode == "id collision" {
				if row != nil || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(row, err)
				}
				return
			}
			if err != nil || row == nil || row.StatusCode != 0 || len(row.Envelope) != 0 || row.Wire != nil || len(row.Discovery.Responses) != 0 {
				t.Fatal(row, err)
			}
			quotaClassField(t, row, "id", `"custom"`)
			quotaClassField(t, row, "cores", "null")
		})
	}
}

func TestNovaQuotaClassDefaultDiscoveryCapsVersionAndOwnsProof(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	discoveryBody := ""
	cloud.Mux.HandleFunc("/compute/v2.1/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("X-Request-Id", "discovery-proof")
		if r.URL.Path == "/compute/v2.1/" {
			discoveryBody = fmt.Sprintf(`{"version":{"id":"v2.1","status":"CURRENT","min_version":"2.1","version":"2.110","links":[{"rel":"self","href":%q}]}}`, cloud.Server.URL+"/compute/v2.1/")
			testcloud.JSON(w, 200, discoveryBody)
			return
		}
		if r.Header.Get("OpenStack-API-Version") != "compute 2.56" || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.56" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"quota_class_set":{"cores":1}}`)
	})
	api := quotasets.New(cloud.Client("compute", "/compute/v2.1/"))
	row, err := api.GetQuotaClassRecord(context.Background())
	if err != nil || row == nil || calls != 2 || !row.Discovery.Found || len(row.Discovery.Responses) != 1 || string(row.Discovery.Responses[0].Body) != discoveryBody {
		t.Fatal(row, err, calls)
	}
	clone := row.Clone()
	clone.Discovery.Responses[0].Body[0] = '['
	clone.Discovery.Responses[0].Header.Set("X-Request-Id", "caller")
	clone.Discovery.Responses[0] = nil
	if string(row.Discovery.Responses[0].Body) != discoveryBody || row.Discovery.Responses[0].Header.Get("X-Request-Id") != "discovery-proof" {
		t.Fatal(row.Discovery)
	}
}

func TestNovaQuotaClassOptionsOwnAttributesHeadersAndExplicitVersion(t *testing.T) {
	attrs := map[string]json.RawMessage{"cores": json.RawMessage(`"12"`)}
	headers := map[string]string{"X-Owned": "original"}
	version := "2.96"
	option := quotasets.WithQuotaClassUpdateOpts(quotasets.QuotaClassUpdateOpts{Attributes: attrs, Headers: headers, Microversion: &version})
	attrs["cores"] = json.RawMessage(`"99"`)
	headers["X-Owned"] = "changed"
	version = "2.1"
	cloud := testcloud.New(t)
	calls := 0
	cloud.Mux.HandleFunc("/compute/v2.1/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "PUT" || r.Header.Get("X-Owned") != "original" || r.Header.Get("OpenStack-API-Version") != "compute 2.96" {
			t.Error(r.Method, r.Header)
		}
		var body map[string]map[string]json.RawMessage
		_ = json.NewDecoder(r.Body).Decode(&body)
		if string(body["quota_class_set"]["cores"]) != `"12"` {
			t.Error(body)
		}
		testcloud.JSON(w, 200, `{"quota_class_set":{}}`)
	})
	api := quotasets.New(cloud.Client("compute", "/compute/v2.1/"))
	for i := 0; i < 2; i++ {
		row, err := api.UpdateQuotaClassRecord(context.Background(), "default", option)
		if err != nil || row == nil || len(row.Discovery.Responses) != 0 {
			t.Fatal(row, err)
		}
	}
	if calls != 2 {
		t.Fatal(calls)
	}
	t.Run("explicit version replaces captured matching version headers", func(t *testing.T) {
		fixture := testcloud.New(t)
		requests := 0
		fixture.Mux.HandleFunc("/compute/v2.1/", func(w http.ResponseWriter, r *http.Request) {
			requests++
			if r.Header.Get("OpenStack-API-Version") != "compute 2.56" || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.56" {
				t.Error(r.Header)
			}
			testcloud.JSON(w, 200, `{"quota_class_set":{"cores":1}}`)
		})
		selected := fixture.Client("compute", "/compute/v2.1/")
		selected.Microversion = "2.1"
		selected.MoreHeaders = map[string]string{"OpenStack-API-Version": "compute 2.1", "X-OpenStack-Nova-API-Version": "2.1"}
		got, err := quotasets.New(selected).GetQuotaClassRecord(context.Background(), quotasets.WithQuotaClassGetMicroversion("2.56"))
		if err != nil || got == nil || requests != 1 || len(got.Discovery.Responses) != 0 {
			t.Fatal(got, err, requests)
		}
		if selected.Microversion != "2.1" || selected.MoreHeaders["OpenStack-API-Version"] != "compute 2.1" || selected.MoreHeaders["X-OpenStack-Nova-API-Version"] != "2.1" {
			t.Fatal(selected)
		}
	})

}

func TestNovaQuotaClassPreflightAndCallbackGuardsPreventRequests(t *testing.T) {
	for _, mode := range []string{"nil option", "callback error", "source mutation", "cancel", "invalid name", "forbidden header", "reserved quota_class_set", "reserved value", "invalid integer attribute"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			calls, later := 0, 0
			cause := errors.New("quota class preflight")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls++; testcloud.JSON(w, 200, `{}`) })
			api := quotasets.New(cloud.Client("compute", "/compute/v2.1/"))
			var option quotasets.QuotaClassGetOption
			if mode != "nil option" {
				option = func(*quotasets.QuotaClassGetOpts) error {
					switch mode {
					case "callback error":
						return cause
					case "source mutation":
						api.RawClient().Endpoint = cloud.Server.URL + "/changed/"
					case "cancel":
						cancel(cause)
					}
					return nil
				}
			}
			if mode == "invalid name" {
				option = quotasets.WithQuotaClassName("..")
			}
			if mode == "forbidden header" {
				option = quotasets.WithQuotaClassGetHeaders(map[string]string{"X-Auth-Token": "foreign"})
			}
			var err error
			switch mode {
			case "reserved quota_class_set":
				_, err = api.UpdateQuotaClassRecord(ctx, "default", quotasets.WithQuotaClassUpdateAttribute("quota_class_set", nil))
			case "reserved value":
				_, err = api.UpdateQuotaClassRecord(ctx, "default", quotasets.WithQuotaClassUpdateAttribute("value", nil))
			case "invalid integer attribute":
				_, err = api.UpdateQuotaClassRecord(ctx, "default", quotasets.WithQuotaClassUpdateAttribute("cores", "²"))
			default:
				_, err = api.GetQuotaClassRecord(ctx, option, func(*quotasets.QuotaClassGetOpts) error { later++; return nil })
			}
			if err == nil || calls != 0 {
				t.Fatal(err, calls)
			}
			if mode == "nil option" || mode == "callback error" || mode == "source mutation" || mode == "cancel" {
				if later != 0 {
					t.Fatal(later)
				}
			}
			if mode == "callback error" || mode == "cancel" {
				if !errors.Is(err, cause) {
					t.Fatal(err)
				}
			}
		})
	}
}

type quotaClassFaultBody struct {
	io.Reader
	readErr, closeErr error
	closed            int
	onClose           func()
}

func (b *quotaClassFaultBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	if n > 0 && b.readErr != nil {
		return n, b.readErr
	}
	return n, err
}
func (b *quotaClassFaultBody) Close() error {
	b.closed++
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}
func TestNovaQuotaClassPhysicalErrorsAnd404NeverReplay(t *testing.T) {
	for _, op := range []string{"get", "update"} {
		for _, mode := range []string{"read", "close", "cancel", "clean404"} {
			t.Run(op+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				calls := 0
				cause := errors.New("quota class physical " + mode)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				fault := &quotaClassFaultBody{Reader: strings.NewReader(`{"error":"actual proof"}`)}
				if mode == "read" {
					fault.readErr = cause
				}
				if mode == "close" {
					fault.closeErr = cause
				}
				if mode == "cancel" {
					fault.onClose = func() { cancel(cause) }
				}
				cloud.SetTransport(testcloud.LinkRequest(func(r *http.Request) (*http.Response, error) {
					calls++
					return testcloud.StreamResponse(404, fault, http.Header{"Content-Type": {"application/json"}}), nil
				}))
				client := cloud.Client("compute", "/compute/v2.1/")
				client.Microversion = "2.96"
				api := quotasets.New(client)
				var err error
				if op == "get" {
					_, err = api.GetQuotaClassRecord(ctx)
				} else {
					_, err = api.UpdateQuotaClassRecord(ctx, "default", quotasets.WithQuotaClassUpdateAttribute("cores", 1))
				}
				if err == nil || calls != 1 || fault.closed != 1 {
					t.Fatal(err, calls, fault.closed)
				}
				if mode == "clean404" {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != 404 {
						t.Fatal(err, native)
					}
				} else if !errors.Is(err, cause) {
					t.Fatal(err)
				}
			})
		}
	}
}
