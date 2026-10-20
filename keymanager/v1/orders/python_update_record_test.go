package orders_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func orderUpdateFixture(t *testing.T, reply func(*http.Request) *http.Response) (*orders.API, *testcloud.Cloud, *[]string) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := []string{}
	client := nativeOrderClient(cloud)
	cloud.Provider.HTTPClient.Transport = nativeOrderTransport(func(req *http.Request) (*http.Response, error) {
		var data []byte
		if req.Body != nil {
			data, _ = io.ReadAll(req.Body)
		}
		calls = append(calls, req.Method+" "+req.URL.EscapedPath()+" "+string(data))
		return reply(req), nil
	})
	return orders.New(client), cloud, &calls
}
func TestPythonOrderUpdateRecordNoop(t *testing.T) {
	for _, id := range []string{"o1", ""} {
		for _, attr := range []string{"", "unknown", "location", "microversion", "if_match"} {
			t.Run(id+attr, func(t *testing.T) {
				api, _, calls := orderUpdateFixture(t, func(*http.Request) *http.Response { t.Error("unexpected HTTP"); return nativeOrderWire(200, `{}`) })
				var opts []orders.UpdateRecordOption
				if attr != "" {
					opts = append(opts, orders.WithUpdateRecordAttribute(attr, "ignored"))
				}
				got, err := api.UpdateRecord(context.Background(), id, opts...)
				if err != nil || got == nil || got.Committed || got.StatusCode != 0 || got.Wire != nil || got.Header != nil || got.Envelope != nil || len(*calls) != 0 || len(got.Request) != 0 || got.Resource == nil {
					t.Fatal(got, err, *calls)
				}
				want, _ := json.Marshal(id)
				if string(got.Resource.Body["id"]) != string(want) || string(got.Resource.Body["meta"]) != "null" {
					t.Fatal(got)
				}
			})
		}
	}
}
func TestPythonOrderUpdateRecordDirtyAliasesAndOwnedReceipt(t *testing.T) {
	api, _, calls := orderUpdateFixture(t, func(req *http.Request) *http.Response {
		if req.Header.Get("X-Source") != "direct" || req.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(req.Header)
		}
		return nativeOrderWire(202, `{"status":"ACTIVE","meta":false,"secret_ref":"https://foreign/secrets/new","unknown":true,"self":"drop"}`)
	})
	got, err := api.UpdateRecord(context.Background(), "o1", orders.WithUpdateRecordAttribute("created_at", nil), orders.WithUpdateRecordAttribute("updated", false), orders.WithUpdateRecordAttribute("order_id", "https://foreign/orders/input"), orders.WithUpdateRecordAttribute("meta", json.RawMessage(`{"bit_length":9007199254740993}`)), orders.WithUpdateRecordAttribute("unknown", "discard"))
	want := `PUT /barbican/v1/orders/o1 {"created":null,"meta":{"bit_length":9007199254740993},"order_ref":"https://foreign/orders/input","updated":false}`
	if err != nil || got == nil || !got.Committed || len(*calls) != 1 || (*calls)[0] != want || got.StatusCode != 202 || string(got.Resource.Body["meta"]) != "{}" || string(got.Resource.Body["order_id"]) != `"input"` || string(got.Resource.Body["secret_id"]) != `"new"` || got.Resource.Body["unknown"] != nil || string(got.Wire.Body["unknown"]) != "true" {
		t.Fatal(got, err, *calls)
	}
	if got.Request["id"] != nil || got.Resource.Body["self"] != nil {
		t.Fatal(got)
	}
	got.Seed["meta"][0] = 'X'
	got.Header.Set("X-Order-Proof", "changed")
	got.Resource.Body["status"][0] = 'X'
	if string(got.Request["meta"]) != `{"bit_length":9007199254740993}` || got.Wire.Header.Get("X-Order-Proof") != "actual" || string(got.Wire.Body["status"]) != `"ACTIVE"` {
		t.Fatal("aliased receipt")
	}
}
func TestPythonOrderUpdateRecordAliasPriority(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(fmt.Sprint(reverse), func(t *testing.T) {
			body := `{"created":"wire","created_at":"alias"}`
			if reverse {
				body = `{"created_at":"alias","created":"wire"}`
			}
			api, _, calls := orderUpdateFixture(t, func(*http.Request) *http.Response { return nativeOrderWire(200, body) })
			opts := []orders.UpdateRecordOption{orders.WithUpdateRecordAttribute("created", "wire"), orders.WithUpdateRecordAttribute("created_at", "alias")}
			want := "alias"
			if reverse {
				opts[0], opts[1] = opts[1], opts[0]
				want = "wire"
			}
			got, err := api.UpdateRecord(context.Background(), "o1", opts...)
			if err != nil || (*calls)[0] != `PUT /barbican/v1/orders/o1 {"created":"`+want+`"}` || string(got.Resource.Body["created_at"]) != `"`+want+`"` {
				t.Fatal(got, err, *calls)
			}
		})
	}
}
func TestPythonOrderUpdateRecordResponseDomain(t *testing.T) {
	for _, body := range []string{"", "{bad", `{}`, `[]`, `null`, `false`, `42`, `"x"`, `{"order_ref":"invalid"}`} {
		t.Run(body, func(t *testing.T) {
			api, _, calls := orderUpdateFixture(t, func(*http.Request) *http.Response { return nativeOrderWire(200, body) })
			got, err := api.UpdateRecord(context.Background(), "o1", orders.WithUpdateRecordAttribute("status", "PENDING"))
			fail := body == `[]` || body == `null` || body == `false` || body == `42` || body == `"x"` || body == `{"order_ref":"invalid"}`
			if got == nil || !got.Committed || got.StatusCode != 200 || string(got.Envelope) != body || len(*calls) != 1 {
				t.Fatal(got, err, *calls)
			}
			if fail {
				var proof *resource.ResponseError
				if err == nil || !errors.As(err, &proof) || string(proof.Body) != body || proof.StatusCode != 200 {
					t.Fatal(got, err, proof)
				}
			} else if err != nil || string(got.Resource.Body["status"]) != `"PENDING"` {
				t.Fatal(got, err)
			}
		})
	}
}
func TestPythonOrderUpdateRecordAcceptedStatusAndFault(t *testing.T) {
	for _, code := range []int{199, 200, 202, 204, 299, 302, 399, 400, 404, 409, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			api, _, calls := orderUpdateFixture(t, func(*http.Request) *http.Response { return nativeOrderWire(code, `{"status":"received"}`) })
			got, err := api.UpdateRecord(context.Background(), "o1", orders.WithUpdateRecordAttribute("status", nil))
			if got == nil || len(*calls) != 1 || got.StatusCode != code || got.Committed != (code < 400) || (err != nil) != (code >= 400) {
				t.Fatal(got, err, *calls)
			}
			if code >= 400 {
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &proof) || proof.Actual != code || string(proof.Body) != `{"status":"received"}` || proof.ResponseHeader.Get("X-Order-Proof") != "actual" || string(got.Envelope) != string(proof.Body) {
					t.Fatal(got, err)
				}
			}
		})
	}
}
func TestPythonOrderUpdateRecordPreflightAndClientGuard(t *testing.T) {
	for _, id := range []string{"", ".", "..", "a\nb"} {
		t.Run(id, func(t *testing.T) {
			api, _, calls := orderUpdateFixture(t, func(*http.Request) *http.Response { t.Error("unexpected HTTP"); return nativeOrderWire(200, `{}`) })
			_, err := api.UpdateRecord(context.Background(), id, orders.WithUpdateRecordAttribute("status", false))
			if !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
				t.Fatal(err, *calls)
			}
		})
	}
	for _, key := range []string{"id", "connection", "_synchronized", "cls"} {
		t.Run(key, func(t *testing.T) {
			api, _, calls := orderUpdateFixture(t, func(*http.Request) *http.Response { t.Error("unexpected HTTP"); return nativeOrderWire(200, `{}`) })
			_, err := api.UpdateRecord(context.Background(), "o1", orders.WithUpdateRecordAttribute(key, "x"))
			if !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
				t.Fatal(err, *calls)
			}
		})
	}
	api, _, calls := orderUpdateFixture(t, func(*http.Request) *http.Response { t.Error("unexpected HTTP"); return nativeOrderWire(200, `{}`) })
	_, err := api.UpdateRecord(context.Background(), "o1", func(c *orders.UpdateRecordOpts) error {
		api.RawClient().ResourceBase = "https://changed.invalid/v1/"
		return nil
	})
	if err == nil || len(*calls) != 0 {
		t.Fatal(err, *calls)
	}
}
func TestPythonOrderUpdateRecordOptionsOwnInputsAndLiteralPath(t *testing.T) {
	attrs := map[string]json.RawMessage{"status": json.RawMessage(`false`)}
	headers := map[string]string{"X-Direct": "saved"}
	path := "alternate/orders"
	version := "1.1"
	opts := orders.WithUpdateRecordOpts(orders.UpdateRecordOpts{Attributes: attrs, Headers: headers, BasePath: &path, Microversion: &version})
	attrs["status"][0] = 'X'
	headers["X-Direct"] = "changed"
	path = "bad"
	version = "bad"
	api, _, calls := orderUpdateFixture(t, func(req *http.Request) *http.Response {
		if req.Header.Get("X-Direct") != "saved" || req.Header.Get("OpenStack-API-Version") != "key-manager 1.1" {
			t.Error(req.Header)
		}
		return nativeOrderWire(200, `{}`)
	})
	for range 2 {
		got, err := api.UpdateRecord(context.Background(), "o/1", opts)
		if err != nil || !got.Committed {
			t.Fatal(got, err)
		}
	}
	if len(*calls) != 2 || (*calls)[0] != `PUT /barbican/v1/alternate/orders/o%2F1 {"status":false}` || (*calls)[1] != (*calls)[0] {
		t.Fatal(*calls)
	}
}
func TestPythonOrderUpdateRecordCharsetKeepsPhysicalProof(t *testing.T) {
	for _, shape := range []bool{false, true} {
		t.Run(fmt.Sprint(shape), func(t *testing.T) {
			body := string([]byte{'{', '"', 's', 't', 'a', 't', 'u', 's', '"', ':', '"', 0xe9, '"', '}'})
			if shape {
				body = `[]`
			}
			api, _, _ := orderUpdateFixture(t, func(*http.Request) *http.Response {
				r := nativeOrderWire(200, body)
				r.Header.Set("Content-Type", "application/json; charset=latin1")
				return r
			})
			got, err := api.UpdateRecord(context.Background(), "o1", orders.WithUpdateRecordAttribute("status", "seed"))
			if got == nil || string(got.Envelope) != body || !strings.Contains(got.Header.Get("Content-Type"), "latin1") {
				t.Fatal(got, err)
			}
			if shape {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || string(proof.Body) != body {
					t.Fatal(err)
				}
			} else if err != nil || string(got.Resource.Body["status"]) != `"é"` {
				t.Fatal(got, err)
			}
		})
	}
}
