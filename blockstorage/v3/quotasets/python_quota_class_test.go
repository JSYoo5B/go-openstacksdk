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

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func quotaClassFixture(t *testing.T, reply func(*http.Request) *http.Response) (*quotasets.API, *gophercloud.ServiceClient, *int) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := new(int)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) { *calls++; return reply(req), nil }))
	client := cloud.Client("volumev3", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/"
	return quotasets.New(client), client, calls
}
func quotaClassWire(code int, body string) *http.Response {
	return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"quota-class-actual"}})
}
func quotaClassBody(t *testing.T, r *http.Request) string {
	t.Helper()
	if r.Body == nil {
		return ""
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestPythonCinderQuotaClassGetSevenDescriptorsAndActualWire(t *testing.T) {
	const raw = `{"quota_class_set":{"id":"actual","backup_gigabytes":"12","backups":true,"gigabytes":900719925474099312345,"groups":-1,"per_volume_gigabytes":null,"snapshots":2.9,"volumes":"٣","x_unknown":false,"location":{"cloud":"foreign"}},"receipt":"whole"}`
	api, _, calls := quotaClassFixture(t, func(r *http.Request) *http.Response {
		if r.Method != http.MethodGet || r.URL.Path != "/cinder/v3/os-quota-class-sets/default" || r.URL.RawQuery != "" || r.Body != nil {
			t.Fatal(r.Method, r.URL, r.Body)
		}
		return quotaClassWire(203, raw)
	})
	row, err := api.GetQuotaClassRecord(context.Background())
	if err != nil || row == nil || row.Resource == nil || row.Wire == nil || *calls != 1 || row.StatusCode != 203 || string(row.Envelope) != raw || row.Header.Get("X-Request-Id") != "quota-class-actual" {
		t.Fatal(row, err, *calls)
	}
	for key, want := range map[string]string{"id": `"actual"`, "backup_gigabytes": "12", "backups": "true", "gigabytes": "900719925474099312345", "groups": "-1", "per_volume_gigabytes": "null", "snapshots": "2", "volumes": "3", "name": "null", "location": "null"} {
		if string(row.Resource.Body[key]) != want {
			t.Fatal(key, row.Resource.Body)
		}
	}
	if row.Resource.Body["x_unknown"] != nil || string(row.Wire.Body["x_unknown"]) != "false" || string(row.Wire.Body["backup_gigabytes"]) != `"12"` || string(row.Wire.Body["location"]) != `{"cloud":"foreign"}` {
		t.Fatal(row)
	}
}

func TestPythonCinderQuotaClassGetMissingNullAndCopiedProof(t *testing.T) {
	for _, raw := range []string{`{"quota_class_set":{}}`, `{"quota_class_set":{"id":null,"volumes":null}}`} {
		t.Run(raw, func(t *testing.T) {
			api, _, calls := quotaClassFixture(t, func(*http.Request) *http.Response { return quotaClassWire(200, raw) })
			row, err := api.GetQuotaClassRecord(context.Background())
			if err != nil || row == nil || row.Resource == nil || *calls != 1 {
				t.Fatal(row, err)
			}
			want := `"default"`
			if strings.Contains(raw, `"id":null`) {
				want = "null"
			}
			if string(row.Resource.Body["id"]) != want || string(row.Resource.Body["volumes"]) != "null" || string(row.Resource.Body["backups"]) != "null" {
				t.Fatal(row)
			}
			row.Header.Set("X-Request-Id", "consumer")
			row.Resource.Body["id"] = json.RawMessage(`"consumer"`)
			if string(row.Envelope) != raw || row.Wire.Header.Get("X-Request-Id") != "quota-class-actual" {
				t.Fatal(row)
			}
		})
	}
}

func TestPythonCinderQuotaClassUpdateRawDirtyAttrsAndSeedProjection(t *testing.T) {
	api, _, calls := quotaClassFixture(t, func(r *http.Request) *http.Response {
		if r.Method != http.MethodPut || r.URL.Path != "/cinder/v3/os-quota-class-sets/default" || r.URL.RawQuery != "" || quotaClassBody(t, r) != `{"quota_class_set":{"snapshots":null,"volumes":"9"}}` {
			t.Fatal(r.Method, r.URL)
		}
		return quotaClassWire(202, `{"quota_class_set":{}}`)
	})
	row, err := api.UpdateQuotaClassRecord(context.Background(), "default", quotasets.WithQuotaClassUpdateAttribute("volumes", "9"), quotasets.WithQuotaClassUpdateAttribute("snapshots", nil), quotasets.WithQuotaClassUpdateAttribute("unknown", true))
	if err != nil || row == nil || row.Resource == nil || *calls != 1 || row.StatusCode != 202 || string(row.Resource.Body["id"]) != `"default"` || string(row.Resource.Body["volumes"]) != "9" || string(row.Resource.Body["snapshots"]) != "null" || string(row.Resource.Body["backups"]) != "null" || row.Wire.Body["volumes"] != nil || row.Resource.Body["unknown"] != nil {
		t.Fatal(row, err, *calls)
	}
}

func TestPythonCinderQuotaClassUpdateNoopUnknownAndExplicitNull(t *testing.T) {
	api, _, calls := quotaClassFixture(t, func(r *http.Request) *http.Response {
		if quotaClassBody(t, r) != `{"quota_class_set":{"volumes":null}}` {
			t.Fatal(r.URL)
		}
		return quotaClassWire(200, `{"quota_class_set":{"volumes":null}}`)
	})
	for _, opts := range [][]quotasets.QuotaClassUpdateOption{nil, {quotasets.WithQuotaClassUpdateAttribute("unknown", 1)}} {
		row, err := api.UpdateQuotaClassRecord(context.Background(), "default", opts...)
		if err != nil || row == nil || row.Resource == nil || row.Wire != nil || row.StatusCode != 0 || row.Envelope != nil || row.Header != nil || *calls != 0 {
			t.Fatal(row, err, *calls)
		}
	}
	row, err := api.UpdateQuotaClassRecord(context.Background(), "default", quotasets.WithQuotaClassUpdateAttribute("volumes", nil))
	if err != nil || row == nil || row.Resource == nil || string(row.Resource.Body["volumes"]) != "null" || *calls != 1 {
		t.Fatal(row, err, *calls)
	}
}

func TestPythonCinderQuotaClassOwnedOptionsAndNoVersionDiscovery(t *testing.T) {
	name := "literal/id%"
	version := "3.100"
	headers := map[string]string{"X-Owned": "original"}
	option := quotasets.WithQuotaClassGetOpts(quotasets.QuotaClassGetOpts{Name: &name, Headers: headers, Microversion: &version})
	name = "changed"
	version = "bad\n"
	headers["X-Owned"] = "changed"
	api, client, calls := quotaClassFixture(t, func(r *http.Request) *http.Response {
		if r.URL.EscapedPath() != "/cinder/v3/os-quota-class-sets/literal%2Fid%25" || r.Header.Get("X-Owned") != "original" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.100" || r.Header.Get("OpenStack-API-Version") != "volume 3.100" {
			t.Fatal(r.URL, r.Header)
		}
		return quotaClassWire(200, `{"quota_class_set":{}}`)
	})
	for range 2 {
		if row, err := api.GetQuotaClassRecord(context.Background(), option); err != nil || row == nil {
			t.Fatal(row, err)
		}
	}
	if *calls != 2 || client.Microversion != "" {
		t.Fatal(*calls, client)
	}
	attrs := map[string]json.RawMessage{"volumes": json.RawMessage(`"7"`)}
	updateHeaders := map[string]string{"X-Owned": "update-original"}
	updateOption := quotasets.WithQuotaClassUpdateOpts(quotasets.QuotaClassUpdateOpts{Attributes: attrs, Headers: updateHeaders})
	attrs["volumes"][1] = '9'
	attrs["snapshots"] = json.RawMessage(`5`)
	updateHeaders["X-Owned"] = "mutated"
	updateAPI, _, updateCalls := quotaClassFixture(t, func(r *http.Request) *http.Response {
		if r.Method != http.MethodPut || quotaClassBody(t, r) != `{"quota_class_set":{"volumes":"7"}}` || r.Header.Get("X-Owned") != "update-original" {
			t.Fatal(r.Method, r.URL, r.Header)
		}
		return quotaClassWire(200, `{"quota_class_set":{}}`)
	})
	for range 2 {
		row, err := updateAPI.UpdateQuotaClassRecord(context.Background(), "default", updateOption)
		if err != nil || row == nil || row.Resource == nil || string(row.Resource.Body["volumes"]) != "7" {
			t.Fatal(row, err)
		}
	}
	if *updateCalls != 2 {
		t.Fatal(*updateCalls)
	}
}

func TestPythonCinderQuotaClassPreflightIDsAndBoundConstructorControls(t *testing.T) {
	api, _, calls := quotaClassFixture(t, func(*http.Request) *http.Response { t.Fatal("preflight sent HTTP"); return nil })
	for _, id := range []string{"", " ", ".", "..", "bad\n", "\xff"} {
		if row, err := api.GetQuotaClassRecord(context.Background(), quotasets.WithQuotaClassName(id)); row != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(id, row, err)
		}
		if row, err := api.UpdateQuotaClassRecord(context.Background(), id); row != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(id, row, err)
		}
	}
	for _, opts := range [][]quotasets.QuotaClassUpdateOption{{nil}, {quotasets.WithQuotaClassUpdateAttribute("id", nil)}, {quotasets.WithQuotaClassUpdateAttribute("base_path", "/foreign")}, {quotasets.WithQuotaClassUpdateHeaders(map[string]string{"X-Auth-Token": "foreign"})}} {
		if row, err := api.UpdateQuotaClassRecord(context.Background(), "default", opts...); row != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(row, err)
		}
	}
	// Resource constructor projection rejects isdigit-only nondecimal text
	// before commit; this is distinct from a received-response decode error.
	if row, err := api.UpdateQuotaClassRecord(context.Background(), "default", quotasets.WithQuotaClassUpdateAttribute("volumes", "²")); row != nil || err == nil || *calls != 0 {
		t.Fatal(row, err, *calls)
	}
	if row, err := api.GetQuotaClassRecord(nil); row != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(row, err)
	}
	var nilAPI *quotasets.API
	if row, err := nilAPI.GetQuotaClassRecord(context.Background()); row != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(row, err)
	}
	if *calls != 0 {
		t.Fatal(*calls)
	}
}

func TestPythonCinderQuotaClassCallbacksSourceAndCancellationStayTerminal(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		t.Run(method, func(t *testing.T) {
			api, client, calls := quotaClassFixture(t, func(*http.Request) *http.Response { t.Fatal("changed source sent HTTP"); return nil })
			original := client.ResourceBase
			later := 0
			var err error
			if method == http.MethodGet {
				_, err = api.GetQuotaClassRecord(context.Background(), func(*quotasets.QuotaClassGetOpts) error { client.ResourceBase += "changed/"; return nil }, func(*quotasets.QuotaClassGetOpts) error { later++; return nil })
			} else {
				_, err = api.UpdateQuotaClassRecord(context.Background(), "default", func(*quotasets.QuotaClassUpdateOpts) error { client.ResourceBase += "changed/"; return nil }, func(*quotasets.QuotaClassUpdateOpts) error { later++; return nil })
			}
			client.ResourceBase = original
			if !errors.Is(err, resource.ErrInvalidOption) || *calls != 0 || later != 0 {
				t.Fatal(err, *calls, later)
			}
			cause := errors.New("quota class canceled")
			ctx, cancel := context.WithCancelCause(context.Background())
			cancel(cause)
			if row, err := api.GetQuotaClassRecord(ctx); row != nil || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || *calls != 0 {
				t.Fatal(row, err, *calls)
			}
		})
	}
}

func TestPythonCinderQuotaClassPhysicalFaultKeeps404AndAllCauses(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		for _, code := range []int{200, 404} {
			t.Run(fmt.Sprintf("%s/%d", method, code), func(t *testing.T) {
				const raw = `{"quota_class_set":{"volumes":3}}`
				readCause, closeCause := errors.New("quota class read"), errors.New("quota class close")
				physical := &quotaClassFaultBody{Reader: strings.NewReader(raw), readCause: readCause, closeCause: closeCause}
				api, client, calls := quotaClassFixture(t, func(r *http.Request) *http.Response {
					if r.Method != method {
						t.Fatal(r.Method)
					}
					response := quotaClassWire(code, raw)
					response.Body = physical
					return response
				})
				retries, reauths := 0, 0
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				client.ReauthFunc = func(context.Context) error { reauths++; return nil }
				var row *quotasets.QuotaClassRecord
				var err error
				if method == http.MethodGet {
					row, err = api.GetQuotaClassRecord(context.Background())
				} else {
					row, err = api.UpdateQuotaClassRecord(context.Background(), "default", quotasets.WithQuotaClassUpdateAttribute("volumes", 3))
				}
				if !errors.Is(err, readCause) || !errors.Is(err, closeCause) || *calls != 1 || physical.closed != 1 || retries != 0 || reauths != 0 {
					t.Fatal(row, err, *calls, physical.closed, retries, reauths)
				}
				if code < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != raw || proof.Header.Get("X-Request-Id") != "quota-class-actual" || row != nil && row.Resource != nil {
						t.Fatal(row, err, proof)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if row != nil || !errors.As(err, &native) || native.Actual != 404 || native.Method != method || string(native.Body) != raw || native.ResponseHeader.Get("X-Request-Id") != "quota-class-actual" {
						t.Fatal(row, err, native)
					}
				}
			})
		}
	}
}

type quotaClassFaultBody struct {
	io.Reader
	readCause, closeCause error
	closed                int
}

func (b *quotaClassFaultBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	if n > 0 {
		return n, b.readCause
	}
	return n, err
}
func (b *quotaClassFaultBody) Close() error { b.closed++; return b.closeCause }
