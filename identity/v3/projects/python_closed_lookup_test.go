package projects_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"net/http"
	"testing"
)

func TestClosedProjectDomainSeedAndActualSourceEvidence(t *testing.T) {
	for _, tc := range []struct{ domain, wire string }{{"null", ""}, {"false", "False"}, {"0", "0"}, {`""`, ""}, {`"domain"`, "domain"}, {`["a","b"]`, "a"}} {
		calls := 0
		api, _ := projectRecordAPI(t, func(r *http.Request) *http.Response {
			calls++
			if r.URL.Path != "/keystone/v3/projects/literal%name" || r.URL.Query().Get("domain_id") != tc.wire || r.Header.Get("OpenStack-API-Version") != "identity 3.2" {
				t.Error(r.URL, r.Header)
			}
			return projectRecordWire(203, `{"project":{"name":"actual","unknown":1}}`)
		})
		row, err := api.ResolveRecord(context.Background(), "literal%name", projects.WithRecordResolveRawDomainID(json.RawMessage(tc.domain)), projects.WithRecordResolveMicroversion("3.2"))
		if err != nil || row == nil || calls != 1 || row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "project-proof" {
			t.Fatal(tc, row, err, calls)
		}
		projectRecordField(t, row, "id", `"literal%name"`)
		projectRecordField(t, row, "domain_id", tc.domain)
		projectRecordField(t, row, "location", "null")
		if row.Wire.Body["id"] != nil || row.Wire.Body["domain_id"] != nil {
			t.Fatal(row)
		}
	}
}
func TestClosedProjectFallbackVersionsExactIDNameAndDefaultMissing(t *testing.T) {
	for _, status := range []int{400, 403, 404} {
		for _, mode := range []string{"id", "name", "duplicate", "missing", "late"} {
			calls := 0
			api, _ := projectRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				if r.URL.Query().Get("domain_id") != "False" {
					t.Error(r.URL)
				}
				if calls == 1 {
					if r.Header.Get("OpenStack-API-Version") != "identity 3.9" || r.URL.Query().Has("name") {
						t.Error(r.URL, r.Header)
					}
					return projectRecordWire(status, `{"error":"member"}`)
				}
				if r.Header.Get("OpenStack-API-Version") != "identity 3.1" || r.URL.Query().Get("name") != "needle" {
					t.Error(r.URL, r.Header)
				}
				if calls == 2 {
					return projectRecordWire(203, `{"projects":[{"id":"first","name":"other"}],"next":"?marker=first"}`)
				}
				switch mode {
				case "id":
					return projectRecordWire(200, `{"projects":[{"id":"needle"}]}`)
				case "name":
					return projectRecordWire(200, `{"projects":[{"id":"actual","name":"needle"}]}`)
				case "duplicate":
					return projectRecordWire(200, `{"projects":[{"id":"needle"},{"id":"second","name":"needle"}]}`)
				case "late":
					return projectRecordWire(503, `{"error":"late"}`)
				}
				return projectRecordWire(200, `{"projects":[]}`)
			})
			api.RawClient().Microversion = "3.1"
			row, err := api.ResolveRecord(context.Background(), "needle", projects.WithRecordResolveRawDomainID(json.RawMessage("false")), projects.WithRecordResolveMicroversion("3.9"))
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
func TestClosedProjectOwnedCallbacksRawDomainAndPreflight(t *testing.T) {
	calls := 0
	api, _ := projectRecordAPI(t, func(*http.Request) *http.Response { calls++; return projectRecordWire(200, `{"project":{"id":"one"}}`) })
	for _, options := range [][]projects.RecordResolveOption{{nil}, {projects.WithRecordResolveRawDomainID(json.RawMessage(`{bad`))}, {projects.WithRecordResolveHeaders(map[string]string{"X-Auth-Token": "foreign"})}} {
		row, err := api.ResolveRecord(context.Background(), "one", options...)
		if row != nil || err == nil || calls != 0 {
			t.Fatal(row, err, calls)
		}
	}
	seen := 0
	base := api.RawClient().ResourceBase
	row, err := api.ResolveRecord(context.Background(), "one", func(*projects.RecordResolveOpts) error { api.RawClient().ResourceBase += "changed/"; return nil }, func(*projects.RecordResolveOpts) error { seen++; api.RawClient().ResourceBase = base; return nil })
	if row != nil || !errors.Is(err, resource.ErrInvalidOption) || seen != 0 || calls != 0 {
		t.Fatal(row, err, seen, calls)
	}
}
func TestClosedProjectPhysicalFaultKeepsNative401ValueAndAllCauses(t *testing.T) {
	for _, status := range []int{203, 400, 403, 404, 401, 429, 503} {
		calls, hooks := 0, 0
		cause := errors.New("physical close")
		body := `{"project":{"id":"one"}}`
		b := &projectRecordFaultBody{text: body, close: cause}
		api, fixture := projectRecordAPI(t, func(*http.Request) *http.Response {
			calls++
			r := projectRecordWire(status, body)
			r.Body = b
			return r
		})
		fixture.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			hooks++
			return nil
		}
		fixture.Provider.ReauthFunc = func(context.Context) error { hooks++; return nil }
		fixture.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error { hooks++; return nil }
		row, err := api.ResolveRecord(context.Background(), "one")
		if row != nil || err == nil || !errors.Is(err, cause) || calls != 1 || hooks != 0 || b.closed != 1 {
			t.Fatal(row, err, calls, hooks, b.closed)
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
		if !errors.As(receipt, &native) || native.Actual != status || string(native.Body) != body || native.Method != http.MethodGet || native.URL != fixture.Server.URL+"/keystone/v3/projects/one" || native.ResponseHeader.Get("X-Request-Id") != "project-proof" {
			t.Fatal(err, native)
		}
	}
}
