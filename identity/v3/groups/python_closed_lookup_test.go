package groups_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"net/http"
	"testing"
)

func TestClosedGroupDomainSeedAndActualSourceEvidence(t *testing.T) {
	for _, tc := range []struct{ domain, wire string }{{"null", ""}, {"false", "False"}, {"0", "0"}, {`""`, ""}, {`"domain"`, "domain"}, {`["a","b"]`, "a"}} {
		calls := 0
		api, _ := groupRecordAPI(t, func(r *http.Request) *http.Response {
			calls++
			if r.URL.Path != "/keystone/v3/groups/literal%name" || r.URL.Query().Get("domain_id") != tc.wire || r.Header.Get("OpenStack-API-Version") != "identity 3.2" {
				t.Error(r.URL, r.Header)
			}
			return groupRecordWire(203, `{"group":{"name":"actual","unknown":1}}`)
		})
		row, err := api.ResolveRecord(context.Background(), "literal%name", groups.WithRecordResolveRawDomainID(json.RawMessage(tc.domain)), groups.WithRecordResolveMicroversion("3.2"))
		if err != nil || row == nil || calls != 1 || row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "group-proof" {
			t.Fatal(tc, row, err, calls)
		}
		groupRecordField(t, row, "id", `"literal%name"`)
		groupRecordField(t, row, "domain_id", tc.domain)
		groupRecordField(t, row, "location", "null")
		if row.Wire.Body["id"] != nil || row.Wire.Body["domain_id"] != nil {
			t.Fatal(row)
		}
	}
}
func TestClosedGroupFallbackVersionsExactIDNameAndDefaultMissing(t *testing.T) {
	for _, status := range []int{400, 403, 404} {
		for _, mode := range []string{"id", "name", "duplicate", "missing", "late"} {
			calls := 0
			api, _ := groupRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				if r.URL.Query().Get("domain_id") != "False" {
					t.Error(r.URL)
				}
				if calls == 1 {
					if r.Header.Get("OpenStack-API-Version") != "identity 3.9" || r.URL.Query().Has("name") {
						t.Error(r.URL, r.Header)
					}
					return groupRecordWire(status, `{"error":"member"}`)
				}
				if r.Header.Get("OpenStack-API-Version") != "identity 3.1" || r.URL.Query().Get("name") != "needle" {
					t.Error(r.URL, r.Header)
				}
				if calls == 2 {
					return groupRecordWire(203, `{"groups":[{"id":"first","name":"other"}],"next":"?marker=first"}`)
				}
				switch mode {
				case "id":
					return groupRecordWire(200, `{"groups":[{"id":"needle"}]}`)
				case "name":
					return groupRecordWire(200, `{"groups":[{"id":"actual","name":"needle"}]}`)
				case "duplicate":
					return groupRecordWire(200, `{"groups":[{"id":"needle"},{"id":"second","name":"needle"}]}`)
				case "late":
					return groupRecordWire(503, `{"error":"late"}`)
				}
				return groupRecordWire(200, `{"groups":[]}`)
			})
			api.RawClient().Microversion = "3.1"
			row, err := api.ResolveRecord(context.Background(), "needle", groups.WithRecordResolveRawDomainID(json.RawMessage("false")), groups.WithRecordResolveMicroversion("3.9"))
			if calls != 3 || api.RawClient().Microversion != "3.1" {
				t.Fatal(row, err, calls)
			}
			if mode == "duplicate" {
				if !errors.Is(err, resource.ErrAmbiguous) || row != nil {
					t.Fatal(row, err)
				}
				continue
			}
			if mode == "late" {
				if !gophercloud.ResponseCodeIs(err, 503) || row != nil {
					t.Fatal(row, err)
				}
				continue
			}
			if err != nil || (row == nil) != (mode == "missing") {
				t.Fatal(mode, row, err)
			}
		}
	}
}
func TestClosedGroupOwnedCallbacksRawDomainAndPreflight(t *testing.T) {
	calls := 0
	api, _ := groupRecordAPI(t, func(*http.Request) *http.Response { calls++; return groupRecordWire(200, `{"group":{"id":"one"}}`) })
	for _, options := range [][]groups.RecordResolveOption{{nil}, {groups.WithRecordResolveRawDomainID(json.RawMessage(`{bad`))}, {groups.WithRecordResolveHeaders(map[string]string{"X-Auth-Token": "foreign"})}} {
		row, err := api.ResolveRecord(context.Background(), "one", options...)
		if row != nil || err == nil || calls != 0 {
			t.Fatal(row, err, calls)
		}
	}
	seen := 0
	base := api.RawClient().ResourceBase
	row, err := api.ResolveRecord(context.Background(), "one", func(*groups.RecordResolveOpts) error { api.RawClient().ResourceBase += "changed/"; return nil }, func(*groups.RecordResolveOpts) error { seen++; api.RawClient().ResourceBase = base; return nil })
	if row != nil || !errors.Is(err, resource.ErrInvalidOption) || seen != 0 || calls != 0 {
		t.Fatal(row, err, seen, calls)
	}
}
func TestClosedGroupPhysicalFaultKeepsNative401ValueAndAllCauses(t *testing.T) {
	for _, status := range []int{203, 400, 403, 404, 401, 429, 503} {
		calls, hooks := 0, 0
		cause := errors.New("physical close")
		body := `{"groups":`
		closed := 0
		b := &groupFaultBody{closeErr: cause, closes: &closed}
		api, fixture := groupRecordAPI(t, func(*http.Request) *http.Response { calls++; r := groupRecordWire(status, body); r.Body = b; return r })
		fixture.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			hooks++
			return nil
		}
		fixture.Provider.ReauthFunc = func(context.Context) error { hooks++; return nil }
		fixture.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error { hooks++; return nil }
		row, err := api.ResolveRecord(context.Background(), "one")
		if row != nil || err == nil || !errors.Is(err, cause) || calls != 1 || hooks != 0 || closed != 1 {
			t.Fatal(row, err, calls, hooks, closed)
		}
		if status < 400 {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body {
				t.Fatal(err, proof)
			}
			continue
		}
		receipt := err
		if status == 401 {
			var auth *gophercloud.ErrUnableToReauthenticate
			if !errors.As(err, &auth) || !errors.Is(auth.ErrReauth, cause) {
				t.Fatal(err)
			}
			receipt = auth.ErrOriginal
		}
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(receipt, &native) || native.Actual != status || string(native.Body) != body || native.Method != http.MethodGet || native.URL != fixture.Server.URL+"/keystone/v3/groups/one" || native.ResponseHeader.Get("X-Request-Id") != "group-proof" {
			t.Fatal(err, native)
		}
	}
}
