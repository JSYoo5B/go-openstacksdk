package waits_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func quotaTarget(t *testing.T, body string) waits.Target {
	t.Helper()
	return waits.ForQuota(&quotas.Quota{Metadata: raw(t, body).Metadata})
}

func TestPythonKeymanagerQuotaWaitOwnsNoIDRouteAndEnvelopeRawDescriptors(t *testing.T) {
	body := `{"quotas":{"name":"ACTIVE","id":null,"secrets":900719925474099312345,"orders":null,"containers":false,"consumers":0,"cas":{"nested":1e-400},"status":"deleted","extension":[1,2]},"outer":true}`
	api, _, calls := fixture(t, func(req *http.Request, _ int) *http.Response {
		if req.URL.EscapedPath() != "/barbican/v1/quotas" {
			t.Fatal(req.URL)
		}
		return wire(203, body)
	})
	value, err := api.WaitForStatus(context.Background(), waits.Quota(), str("active"), waits.WithAttribute("name"))
	if err != nil || value == nil || value.Kind != "quotas" || value.RequestID != "current-project" || value.StatusCode != 203 || string(value.Envelope) != body || len(*calls) != 1 {
		t.Fatal(err, value, *calls)
	}
	for key, want := range map[string]string{"name": `"ACTIVE"`, "id": "null", "secrets": "900719925474099312345", "orders": "null", "containers": "false", "consumers": "0", "cas": `{"nested":1e-400}`, "location": "null"} {
		field(t, value, key, want)
	}
	if len(value.Resource.Body) != 8 || value.Resource.Body["status"] != nil || value.Resource.Body["extension"] != nil || value.Wire.Body["quotas"] != nil || value.Wire.Body["outer"] != nil || string(value.Wire.Body["extension"]) != `[1,2]` {
		t.Fatal(value)
	}
	clone := value.Clone()
	clone.Resource.Body["secrets"][0] = 'x'
	clone.Wire.Body["cas"][0] = 'x'
	clone.Envelope[0] = 'x'
	clone.Header.Set("X-Request-Id", "changed")
	if string(value.Resource.Body["secrets"]) != "900719925474099312345" || string(value.Wire.Body["cas"]) != `{"nested":1e-400}` || string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "wait-proof" {
		t.Fatal("receipt aliases", value)
	}
	field(t, value, "id", "null")
}

func TestPythonKeymanagerQuotaWaitCachedSnapshotAndNullableDefaults(t *testing.T) {
	native := &quotas.Quota{Metadata: raw(t, `{"secrets":"READY","orders":null,"name":"cached","id":false}`).Metadata, Secrets: []byte(`"typed-must-not-seed"`)}
	native.Header = http.Header{"X-Original": []string{"owned"}}
	native.StatusCode = 203
	target := waits.ForQuota(native)
	native.Body["secrets"][1] = 'x'
	native.Header.Set("X-Original", "changed")
	api, _, calls := fixture(t, func(*http.Request, int) *http.Response { t.Fatal("cached HTTP"); return nil })
	value, err := api.WaitForStatus(context.Background(), target, str("ready"), waits.WithAttribute("secrets"), waits.WithTimeout(0), waits.WithInterval(-time.Second))
	if err != nil || value == nil || len(*calls) != 0 || value.Envelope != nil || value.StatusCode != 203 || value.Header.Get("X-Original") != "owned" {
		t.Fatal(err, value, *calls)
	}
	field(t, value, "id", "false")
	field(t, value, "secrets", `"READY"`)
	value.Resource.Body["secrets"][1] = 'x'
	value.Wire.Body["name"][1] = 'x'
	value.Header.Set("X-Original", "consumer")
	value, err = api.WaitForStatus(context.Background(), target, str("ready"), waits.WithAttribute("secrets"))
	if err != nil || value.Header.Get("X-Original") != "owned" {
		t.Fatal(err, value)
	}
	field(t, value, "secrets", `"READY"`)
	value, err = api.WaitForStatus(context.Background(), waits.Quota(), nil, waits.WithAttribute("orders"))
	if err != nil || value.StatusCode != 0 || value.Wire != nil || value.Envelope != nil || len(*calls) != 0 {
		t.Fatal(err, value, *calls)
	}
	for _, key := range []string{"id", "name", "secrets", "orders", "containers", "consumers", "cas", "location"} {
		field(t, value, key, "null")
	}
}

func TestPythonKeymanagerQuotaWaitMergesFlatFallbackAndNullPresence(t *testing.T) {
	bodies := []string{`{"quotas":{"name":"BUILD","orders":null,"secrets":0}}`, `not JSON`, `{"name":"ACTIVE","consumers":false,"id":null}`}
	api, _, calls := fixture(t, func(req *http.Request, n int) *http.Response {
		if req.URL.EscapedPath() != "/barbican/v1/quotas" {
			t.Fatal(req.URL)
		}
		return wire(202, bodies[n-1])
	})
	target := quotaTarget(t, `{"name":"BUILD","secrets":999,"orders":100,"containers":900719925474099312345,"id":"foreign"}`)
	progress := 0
	value, err := api.WaitForStatus(context.Background(), target, str("active"), waits.WithAttribute("name"), waits.WithInterval(time.Nanosecond), waits.WithProgressCallback(func(v int) error {
		if v != 0 {
			t.Fatal(v)
		}
		progress++
		return nil
	}))
	if err != nil || len(*calls) != 3 || progress != 2 || string(value.Envelope) != bodies[2] || value.StatusCode != 202 {
		t.Fatal(err, value, *calls, progress)
	}
	for key, want := range map[string]string{"orders": "null", "secrets": "0", "containers": "900719925474099312345", "consumers": "false", "id": "null"} {
		field(t, value, key, want)
	}
	if value.Wire.Body["secrets"] != nil || value.Wire.Body["orders"] != nil {
		t.Fatal(value)
	}
}

func TestPythonKeymanagerQuotaWaitEnvelopeShapeAndNumericAttributesAreExplicitFailures(t *testing.T) {
	for _, body := range []string{`{"quotas":null}`, `{"quotas":[]}`, `{"quotas":false}`, `{"quotas":1}`, `{"quotas":{"secrets":1}}`} {
		api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(203, body) })
		value, err := api.WaitForStatus(context.Background(), waits.Quota(), str("ready"), waits.WithAttribute("secrets"))
		var proof *resource.ResponseError
		if value == nil || value.Resource != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || len(*calls) != 1 {
			t.Fatal(err, value, proof, *calls)
		}
	}
	api, _, calls := fixture(t, func(*http.Request, int) *http.Response { t.Fatal("preflight HTTP"); return nil })
	for _, target := range []waits.Target{waits.Quota(), waits.ForQuota(nil), waits.ForQuota(&quotas.Quota{})} {
		value, err := api.WaitForStatus(context.Background(), target, str("ready"))
		if value != nil || err == nil || len(*calls) != 0 {
			t.Fatal(err, value, *calls)
		}
	}
	value, err := api.WaitForStatus(context.Background(), quotaTarget(t, `{"secrets":900719925474099312345}`), str("ready"), waits.WithAttribute("secrets"))
	if value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
		t.Fatal(err, value, *calls)
	}
	value, err = api.WaitForStatus(context.Background(), waits.Quota(), nil)
	if value != nil || !errors.Is(err, resource.ErrUnsupported) || len(*calls) != 0 {
		t.Fatal(err, value, *calls)
	}
	value, err = api.WaitForStatus(context.Background(), waits.Quota(), nil, waits.WithAttribute("project_id"))
	if value != nil || !errors.Is(err, resource.ErrUnsupported) || len(*calls) != 0 {
		t.Fatal(err, value, *calls)
	}
}
