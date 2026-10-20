package waits_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestPythonProjectQuotaWaitFixedRouteRawDescriptorsAndDefaults(t *testing.T) {
	const body = `{"project_quotas":{"id":null,"name":"READY","secrets":900719925474099312345,"orders":false,"containers":null,"status":"deleted","vendor":3}}`
	api, _, calls := fixture(t, func(r *http.Request, _ int) *http.Response {
		if r.URL.EscapedPath() != "/barbican/v1/project-quotas/project%2Fid" {
			t.Fatal(r.URL)
		}
		return wire(203, body)
	})
	got, err := api.WaitForStatus(context.Background(), waits.ProjectQuota("project/id"), str("ready"), waits.WithAttribute("name"))
	if err != nil || got.Kind != "project-quotas" || got.RequestID != "project/id" || len(*calls) != 1 || got.StatusCode != 203 || string(got.Envelope) != body {
		t.Fatal(got, err, *calls)
	}
	field(t, got, "id", "null")
	field(t, got, "secrets", "900719925474099312345")
	field(t, got, "orders", "false")
	field(t, got, "consumers", "null")
	if got.Resource.Body["status"] != nil || got.Resource.Body["vendor"] != nil || string(got.Wire.Body["vendor"]) != "3" {
		t.Fatal(got)
	}
}
func TestPythonProjectQuotaWaitCachedRecordOwnsSnapshotsAndNoStatus(t *testing.T) {
	original := &quotas.ProjectQuotaRecord{RequestID: "fixed", Resource: raw(t, `{"id":"foreign","name":"READY","secrets":null}`), Wire: raw(t, `{"name":"READY"}`), Envelope: []byte(`{"project_quotas":{"name":"READY"}}`), Header: http.Header{"X-Proof": {"cached"}}, StatusCode: 202}
	target := waits.ForProjectQuota(original)
	original.Resource.Body["name"][1] = 'X'
	original.Envelope[0] = '!'
	api, _, calls := fixture(t, func(*http.Request, int) *http.Response { t.Fatal("cached GET"); return nil })
	got, err := api.WaitForStatus(context.Background(), target, str("ready"), waits.WithAttribute("name"), waits.WithTimeout(0), waits.WithInterval(-time.Second))
	if err != nil || got.RequestID != "fixed" || got.StatusCode != 202 || got.Header.Get("X-Proof") != "cached" {
		t.Fatal(got, err)
	}
	field(t, got, "name", `"READY"`)
	if _, err := api.WaitForStatus(context.Background(), waits.ProjectQuota("fixed"), str("active")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	got, err = api.WaitForStatus(context.Background(), waits.ProjectQuota("fixed"), nil, waits.WithAttribute("orders"))
	if err != nil || got.StatusCode != 0 || got.Wire != nil || got.Envelope != nil || len(*calls) != 0 {
		t.Fatal(got, err, *calls)
	}
	field(t, got, "id", `"fixed"`)
}
func TestPythonProjectQuotaWaitMergeIgnoresMalformedJSONAndPreservesNull(t *testing.T) {
	bodies := []string{`{"project_quotas":{"name":"BUILD","secrets":5}}`, `not JSON`, `{"name":"READY","secrets":null}`}
	api, _, calls := fixture(t, func(r *http.Request, n int) *http.Response {
		if r.URL.EscapedPath() != "/barbican/v1/project-quotas/fixed" {
			t.Fatal(r.URL)
		}
		return wire(202, bodies[n-1])
	})
	count := 0
	got, err := api.WaitForStatus(context.Background(), waits.ProjectQuota("fixed"), str("ready"), waits.WithAttribute("name"), waits.WithInterval(time.Nanosecond), waits.WithProgressCallback(func(progress int) error {
		if progress != 0 {
			t.Fatal(progress)
		}
		count++
		return nil
	}))
	if err != nil || len(*calls) != 3 || count != 2 || string(got.Envelope) != bodies[2] {
		t.Fatal(got, err, count, *calls)
	}
	field(t, got, "id", `"fixed"`)
	field(t, got, "secrets", "null")
}
func TestPythonProjectQuotaWaitDeleteOnlyCleanGET404(t *testing.T) {
	api, _, calls := fixture(t, func(r *http.Request, n int) *http.Response {
		if r.Method != "GET" {
			t.Fatal(r)
		}
		if n == 1 {
			return wire(200, `{"project_quotas":{"id":"foreign","status":"deleted"}}`)
		}
		return wire(404, `{"missing":true}`)
	})
	callbacks := 0
	got, err := api.WaitForDelete(context.Background(), waits.ProjectQuota("fixed"), waits.WithInterval(time.Nanosecond), waits.WithProgressCallback(func(int) error { callbacks++; return nil }))
	if err != nil || len(*calls) != 2 || callbacks != 1 || got.StatusCode != 404 || got.RequestID != "fixed" {
		t.Fatal(got, err, callbacks, *calls)
	}
	field(t, got, "id", `"fixed"`)
}
func TestPythonProjectQuotaWaitShapeNumericAndPhysicalFailures(t *testing.T) {
	for _, body := range []string{`{"project_quotas":null}`, `[]`, `{"project_quotas":{"secrets":7}}`} {
		t.Run(body, func(t *testing.T) {
			api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(202, body) })
			got, err := api.WaitForStatus(context.Background(), waits.ProjectQuota("fixed"), str("ready"), waits.WithAttribute("secrets"))
			var proof *resource.ResponseError
			if got == nil || err == nil || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != body || len(*calls) != 1 {
				t.Fatal(got, err, proof, *calls)
			}
		})
	}
	for _, status := range []int{202, 404} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cause := errors.New("close")
			api, _, calls := fixture(t, func(*http.Request, int) *http.Response {
				r := wire(status, `{"project_quotas":{}}`)
				r.Body = projectQuotaClose{Reader: r.Body, cause: cause}
				return r
			})
			_, err := api.WaitForDelete(context.Background(), waits.ProjectQuota("fixed"), waits.WithInterval(time.Nanosecond))
			if !errors.Is(err, cause) || len(*calls) != 1 {
				t.Fatal(err, *calls)
			}
		})
	}
}

type projectQuotaClose struct {
	io.Reader
	cause error
}

func (b projectQuotaClose) Close() error { return b.cause }
