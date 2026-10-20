package ikepolicies_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"net/http"
	"testing"
)

func policyRecordField(t *testing.T, row *ikepolicies.IKEPolicyRecord, key, want string) {
	t.Helper()
	if row == nil || row.Resource == nil || string(row.Resource.Body[key]) != want {
		t.Fatal(key, want, row)
	}
}
func TestVPNPolicyRecordsDescriptorDefaultsAndActualProof(t *testing.T) {
	var calls []nativeVPNCall
	body := `{"ikepolicies":[{"id":"p1","name":"policy","lifetime":{"units":"seconds","value":"not-converted"},"units":"octets","value":3.9,"tenant_id":"unknown","tags":[],"revision_number":9,"self":"drop"},{}, {"lifetime":false,"value":null}]}`
	api, _ := nativeVPNAPI(t, &calls, func(*http.Request) *http.Response { return nativeVPNWire(203, body) })
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 3 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	for key, want := range map[string]string{"id": `"p1"`, "name": `"policy"`, "lifetime": `{"units":"seconds","value":"not-converted"}`, "units": `"octets"`, "value": "3", "location": "null"} {
		policyRecordField(t, rows[0], key, want)
	}
	for _, key := range []string{"tenant_id", "tags", "revision_number", "self"} {
		if rows[0].Resource.Body[key] != nil {
			t.Fatal("invented inherited/unknown attr", key, rows[0])
		}
	}
	if string(rows[0].Wire.Body["value"]) != "3.9" || string(rows[0].Wire.Body["tenant_id"]) != `"unknown"` || string(rows[0].Envelope) != body || rows[0].StatusCode != 203 {
		t.Fatal(rows[0])
	}
	for _, key := range []string{"id", "name", "auth_algorithm", "description", "encryption_algorithm", "pfs", "project_id", "phase1_negotiation_mode", "lifetime", "units", "value", "location"} {
		policyRecordField(t, rows[1], key, "null")
	}
	policyRecordField(t, rows[2], "lifetime", "{}")
	policyRecordField(t, rows[2], "value", "null")
	rows[0].Resource.Body["lifetime"][0] = 'X'
	rows[0].Header.Set("Content-Type", "changed")
	rows[0].Envelope[0] = 'X'
	if string(rows[0].Wire.Body["lifetime"]) != `{"units":"seconds","value":"not-converted"}` || rows[0].Wire.Header.Get("Content-Type") != "application/json" {
		t.Fatal("raw receipt aliases")
	}
}
func TestVPNPolicyRecordsServerQueryAndLocalBodyFiltersAllPages(t *testing.T) {
	var calls []nativeVPNCall
	api, _ := nativeVPNAPI(t, &calls, func(req *http.Request) *http.Response {
		q := req.URL.Query()
		if q.Get("name") != "policy" || q.Get("phase1_negotiation_mode") != "main" || q.Get("phase_1_negotiation_mode") != "" || q.Get("ike_version") != "v2" || q.Get("fields") != "id" || q.Get("units") != "" || q.Get("value") != "" || q.Get("lifetime") != "" || q.Get("tenant_id") != "" {
			t.Error(req.URL)
		}
		if q.Get("marker") == "later" {
			return nativeVPNWire(200, `{"ikepolicies":[{"id":"keep","name":"policy","lifetime":{"units":"seconds","value":2,"other":true},"units":"octets","value":2}]}`)
		}
		return nativeVPNWire(200, `{"ikepolicies":[{"id":"skip","name":"policy","lifetime":{},"units":"octets","value":1}],"ikepolicies_links":[{"rel":"next","href":"?marker=later"}]}`)
	})
	seq := api.ListRecords(context.Background(), ikepolicies.WithRecordListQuery("name", "policy"), ikepolicies.WithRecordListQuery("phase1_negotiation_mode", "main"), ikepolicies.WithRecordListQuery("ike_version", "v2"), ikepolicies.WithRecordListQuery("fields", []string{"id"}), ikepolicies.WithRecordListQuery("units", "octets"), ikepolicies.WithRecordListQuery("value", 2), ikepolicies.WithRecordListQuery("lifetime", map[string]any{"units": "seconds", "value": 2}), ikepolicies.WithRecordListQuery("tenant_id", make(chan int)))
	if len(calls) != 0 {
		t.Fatal("not lazy")
	}
	var rows []*ikepolicies.IKEPolicyRecord
	for row, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if len(rows) != 1 || len(calls) != 2 {
		t.Fatal(rows, calls)
	}
	policyRecordField(t, rows[0], "id", `"keep"`)
}
func TestVPNPolicyRecordsMarkerContinuationRawCapAndPaginationFlag(t *testing.T) {
	var calls []nativeVPNCall
	api, _ := nativeVPNAPI(t, &calls, func(req *http.Request) *http.Response {
		switch req.URL.Query().Get("marker") {
		case "first":
			return nativeVPNWire(200, `{"ikepolicies":[{"id":"second"}]}`)
		case "second":
			return nativeVPNWire(200, `{"ikepolicies":[]}`)
		default:
			return nativeVPNWire(200, `{"ikepolicies":[{"id":"first"}]}`)
		}
	})
	rows, err := api.AllRecords(context.Background(), ikepolicies.WithRecordListLimit(1))
	if err != nil || len(rows) != 2 || len(calls) != 3 || calls[1].query != "limit=1&marker=first" || calls[2].query != "limit=1&marker=second" {
		t.Fatal(rows, err, calls)
	}
	calls = nil
	rows, err = api.AllRecords(context.Background(), ikepolicies.WithRecordListLimit(1), ikepolicies.WithRecordListPaginated(false))
	if err != nil || len(rows) != 1 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	var capcalls []nativeVPNCall
	capped, _ := nativeVPNAPI(t, &capcalls, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("limit") != "2" {
			t.Error(req.URL)
		}
		return nativeVPNWire(200, `{"ikepolicies":[{"id":"skip","value":0},{"id":"wanted","value":2},{"cls":null}],"next":"https://foreign.invalid/never"}`)
	})
	rows, err = capped.AllRecords(context.Background(), ikepolicies.WithRecordListMaxItems(2), ikepolicies.WithRecordListQuery("value", 2))
	if err != nil || len(rows) != 1 || len(capcalls) != 1 {
		t.Fatal(rows, err, capcalls)
	}
}
func TestVPNPolicyRecordsSnapshotControlsAndVersion(t *testing.T) {
	var calls []nativeVPNCall
	query := map[string]json.RawMessage{"name": json.RawMessage(`"owned"`)}
	headers := map[string]string{"X-Trace": "owned"}
	version := "2.1"
	path := "/custom/ikepolicies"
	option := ikepolicies.WithRecordListOpts(ikepolicies.RecordListOpts{Query: query, Headers: headers, Microversion: &version, BasePath: &path})
	query["name"][1] = 'X'
	headers["X-Trace"] = "changed"
	version = "2.9"
	path = "changed"
	api, _ := nativeVPNAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path != "/neutron/v2.0/custom/ikepolicies" || req.URL.Query().Get("name") != "owned" || req.Header.Get("X-Trace") != "owned" || req.Header.Get("OpenStack-API-Version") != "network 2.1" || req.Header.Get("Accept") != "application/json" {
			t.Error(req.URL, req.Header)
		}
		return nativeVPNWire(200, `{"ikepolicies":[]}`)
	})
	api.RawClient().Microversion = "2.0"
	rows, err := api.AllRecords(context.Background(), option)
	if err != nil || len(rows) != 0 || len(calls) != 1 || api.RawClient().Microversion != "2.0" {
		t.Fatal(rows, err, calls)
	}
}
func TestVPNPolicyRecordsResponseErrorsAndPhysicalFaults(t *testing.T) {
	for _, body := range []string{`{"ikepolicies":[{"value":"²"}]}`, `{"ikepolicies":[{"cls":null}]}`, `{"wrong":[]}`} {
		var calls []nativeVPNCall
		api, _ := nativeVPNAPI(t, &calls, func(*http.Request) *http.Response { return nativeVPNWire(203, body) })
		rows, err := api.AllRecords(context.Background())
		var proof *resource.ResponseError
		if rows != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || len(calls) != 1 {
			t.Fatal(rows, err, proof, calls)
		}
	}
	cause := errors.New("physical")
	var calls []nativeVPNCall
	api, cloud := nativeVPNAPI(t, &calls, func(*http.Request) *http.Response {
		r := nativeVPNWire(503, `{}`)
		r.Body = &policyFaultBody{[]byte(`{"actual":true}`), cause}
		return r
	})
	callbacks := 0
	cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		callbacks++
		return nil
	}
	rows, err := api.AllRecords(context.Background())
	if rows != nil || !errors.Is(err, cause) || len(calls) != 1 || callbacks != 0 {
		t.Fatal(rows, err, calls, callbacks)
	}
}

type policyFaultBody struct {
	data  []byte
	cause error
}

func (b *policyFaultBody) Read(p []byte) (int, error) {
	n := copy(p, b.data)
	b.data = b.data[n:]
	return n, b.cause
}
func (b *policyFaultBody) Close() error { return b.cause }
func TestVPNPolicyRecordsPreflightAndProxyGenerator(t *testing.T) {
	var calls []nativeVPNCall
	api, _ := nativeVPNAPI(t, &calls, func(*http.Request) *http.Response {
		t.Error("unexpected HTTP")
		return nativeVPNWire(200, `{"ikepolicies":[]}`)
	})
	for _, lane := range []string{"nil", "collision", "source"} {
		var options []ikepolicies.RecordListOption
		switch lane {
		case "nil":
			options = append(options, nil)
		case "collision":
			options = append(options, ikepolicies.WithRecordListQuery("cls", nil))
		case "source":
			options = append(options, func(*ikepolicies.RecordListOpts) error { api.RawClient().ResourceBase += "changed/"; return nil })
		}
		rows, err := api.AllRecords(context.Background(), options...)
		if rows != nil || err == nil || len(calls) != 0 {
			t.Fatal(lane, rows, err, calls)
		}
	}
	// A fresh selected client avoids reusing the deliberate source mutation.
	api, _ = nativeVPNAPI(t, &calls, func(*http.Request) *http.Response {
		t.Error("generator evaluated HTTP")
		return nativeVPNWire(200, `{"ikepolicies":[]}`)
	})
	result, err := api.ProxyListRecords(context.Background(), ikepolicies.WithRecordListJMESPath("@"))
	if err != nil || len(calls) != 0 {
		t.Fatal(result, err, calls)
	}
	if _, ok := result.Value.(*ikepolicies.RecordListGenerator); !ok {
		t.Fatal(result)
	}
	result, err = api.ProxyListRecords(context.Background(), ikepolicies.WithRecordListJMESPath("[].id"))
	if err != nil || string(result.JSON) != "null" || len(calls) != 0 {
		t.Fatal(result, err, calls)
	}
	_, err = api.ProxyListRecords(context.Background(), ikepolicies.WithRecordListJMESPath("length(@)"))
	if err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}
