package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	infra "github.com/JSYoo5B/go-openstacksdk/containerinfra/v1"
	"github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/clustertemplates"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func pyCOEDBService(cloud *testcloud.Cloud, microversion string) *infra.Service {
	client := cloud.Client("container-infra", "/v1")
	client.Microversion = microversion
	return infra.New(client)
}

func TestPyCOEDBMagnumServicesListDecodesNullsAndSendsVersionHeader(t *testing.T) {
	cloud := testcloud.New(t)
	var requests atomic.Int32
	var version string
	cloud.Mux.HandleFunc("GET /v1/mservices", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		version = r.Header.Get("OpenStack-API-Version")
		testcloud.JSON(w, 200, `{"mservices":[{"id":1,"binary":"magnum-conductor","host":"ctl-1","state":"up","report_count":0,"disabled_reason":null,"created_at":"2026-01-02T03:04:05+00:00","updated_at":null,"vendor_extra":1},{"id":2,"binary":"magnum-conductor","host":"ctl-2","state":"down","report_count":7,"disabled_reason":"","created_at":null,"updated_at":"2026-01-03T00:00:00+00:00"}]}`)
	})
	got, err := pyCOEDBService(cloud, "1.10").AllMagnumServices(context.Background())
	if err != nil || len(got) != 2 {
		t.Fatalf("got=%v err=%v", got, err)
	}
	first, second := got[0], got[1]
	if first.ID != 1 || first.Binary != "magnum-conductor" || first.Host != "ctl-1" || first.State != "up" || first.ReportCount != 0 ||
		first.DisabledReason != nil || first.UpdatedAt != nil || first.CreatedAt == nil || *first.CreatedAt != "2026-01-02T03:04:05+00:00" {
		t.Fatalf("first=%+v", first)
	}
	if second.DisabledReason == nil || *second.DisabledReason != "" || second.CreatedAt != nil || second.ReportCount != 7 {
		t.Fatalf("second=%+v", second)
	}
	if requests.Load() != 1 || version != "container-infra 1.10" {
		t.Fatalf("requests=%d version=%q", requests.Load(), version)
	}
}

func TestPyCOEDBMagnumServicesListStatusErrorsEmptyAndBreak(t *testing.T) {
	cloud := testcloud.New(t)
	var mode atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/mservices", func(w http.ResponseWriter, r *http.Request) {
		switch mode.Load() {
		case 0:
			testcloud.JSON(w, 200, `{"mservices":[]}`)
		case 1:
			testcloud.JSON(w, 201, `{"mservices":[{"id":1}]}`)
		case 2:
			testcloud.JSON(w, 403, `{}`)
		default:
			testcloud.JSON(w, 200, `{"mservices":[{"id":1},{"id":2}]}`)
		}
	})
	service := pyCOEDBService(cloud, "")
	ctx := context.Background()
	if got, err := service.AllMagnumServices(ctx); err != nil || len(got) != 0 {
		t.Fatalf("empty: %v %v", got, err)
	}
	mode.Store(1)
	if _, err := service.AllMagnumServices(ctx); magnumUnexpected(err) != 201 {
		t.Fatalf("only 200 is accepted: %v", err)
	}
	mode.Store(2)
	if _, err := service.AllMagnumServices(ctx); magnumUnexpected(err) != 403 {
		t.Fatalf("forbidden: %v", err)
	}
	mode.Store(3)
	count := 0
	for _, err := range service.ListMagnumServices(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		count++
		break
	}
	if count != 1 {
		t.Fatalf("break after first row: %d", count)
	}
}

func TestPyCOEDBMagnumServicesListPreCanceledMakesNoRequest(t *testing.T) {
	cloud := testcloud.New(t)
	var requests atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/mservices", func(w http.ResponseWriter, r *http.Request) { requests.Add(1) })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := pyCOEDBService(cloud, "").AllMagnumServices(ctx); !errors.Is(err, context.Canceled) || requests.Load() != 0 {
		t.Fatalf("err=%v requests=%d", err, requests.Load())
	}
}

type pyCOEDBClusterFixture struct {
	deletes, patches atomic.Int32
	patchBody        string
	deleteStatus     int
}

func newPyCOEDBClusterFixture(t *testing.T, cloud *testcloud.Cloud, listBody string) *pyCOEDBClusterFixture {
	f := &pyCOEDBClusterFixture{deleteStatus: 204}
	cloud.Mux.HandleFunc("GET /v1/clusters", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, listBody) })
	cloud.Mux.HandleFunc("DELETE /v1/clusters/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.deletes.Add(1)
		if r.PathValue("id") != "cl-1" {
			t.Errorf("deleted %q", r.PathValue("id"))
		}
		w.WriteHeader(f.deleteStatus)
	})
	cloud.Mux.HandleFunc("PATCH /v1/clusters/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.patches.Add(1)
		data, _ := io.ReadAll(r.Body)
		f.patchBody = string(data)
		testcloud.JSON(w, 202, `{"uuid":"`+r.PathValue("id")+`"}`)
	})
	return f
}

const pyCOEDBClusterList = `{"clusters":[{"uuid":"cl-1","name":"prod"},{"uuid":"cl-2","name":"prod-b"},{"uuid":"cl-3","name":"stage"},{"uuid":"cl-4","name":"stage"}]}`

func TestPyCOEDBDeleteCOEClusterReportsFoundAndMissingWithOneDelete(t *testing.T) {
	cloud := testcloud.New(t)
	f := newPyCOEDBClusterFixture(t, cloud, pyCOEDBClusterList)
	service := pyCOEDBService(cloud, "")
	ctx := context.Background()
	if ok, err := service.DeleteCOECluster(ctx, "nope"); err != nil || ok || f.deletes.Load() != 0 {
		t.Fatalf("missing: ok=%v err=%v deletes=%d", ok, err, f.deletes.Load())
	}
	if ok, err := service.DeleteCOECluster(ctx, "prod"); err != nil || !ok || f.deletes.Load() != 1 {
		t.Fatalf("by name: ok=%v err=%v deletes=%d", ok, err, f.deletes.Load())
	}
	if ok, err := service.DeleteCOECluster(ctx, "cl-1"); err != nil || !ok || f.deletes.Load() != 2 {
		t.Fatalf("by id: ok=%v err=%v deletes=%d", ok, err, f.deletes.Load())
	}
	// Helper selection uses fnmatch-style globs: "prod-*" matches only cl-2 and "pr?d" only cl-1.
	if ok, err := service.DeleteCOECluster(ctx, "pr?d"); err != nil || !ok || f.deletes.Load() != 3 {
		t.Fatalf("glob: ok=%v err=%v deletes=%d", ok, err, f.deletes.Load())
	}
}

func TestPyCOEDBDeleteCOEClusterAmbiguityEmptyReferenceAndErrorsSendNoDelete(t *testing.T) {
	cloud := testcloud.New(t)
	f := newPyCOEDBClusterFixture(t, cloud, pyCOEDBClusterList)
	service := pyCOEDBService(cloud, "")
	ctx := context.Background()
	var ambiguous *resource.AmbiguousError
	if ok, err := service.DeleteCOECluster(ctx, "stage"); ok || !errors.As(err, &ambiguous) || !reflect.DeepEqual(ambiguous.IDs, []string{"cl-3", "cl-4"}) {
		t.Fatalf("duplicate: ok=%v err=%v", ok, err)
	}
	if ok, err := service.DeleteCOECluster(ctx, "prod*"); ok || !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatalf("glob with two matches: ok=%v err=%v", ok, err)
	}
	if ok, err := service.DeleteCOECluster(ctx, ""); ok || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("empty: ok=%v err=%v", ok, err)
	}
	if f.deletes.Load() != 0 {
		t.Fatalf("no DELETE expected, got %d", f.deletes.Load())
	}
	// A non-404 DELETE failure is an error and is sent once; a 404 after selection reports true.
	f.deleteStatus = 500
	if ok, err := pyCOEDBService(cloud, "").DeleteCOECluster(ctx, "cl-1"); ok || magnumUnexpected(err) != 500 || f.deletes.Load() != 1 {
		t.Fatalf("500: ok=%v err=%v deletes=%d", ok, err, f.deletes.Load())
	}
	f.deleteStatus = 404
	if ok, err := pyCOEDBService(cloud, "").DeleteCOECluster(ctx, "cl-1"); err != nil || !ok || f.deletes.Load() != 2 {
		t.Fatalf("404: ok=%v err=%v deletes=%d", ok, err, f.deletes.Load())
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if ok, err := service.DeleteCOECluster(cancelled, "cl-1"); ok || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled: ok=%v err=%v", ok, err)
	}
}

func TestPyCOEDBUpdateCOEClusterPatchesSelectedUUIDWithExplicitOperations(t *testing.T) {
	cloud := testcloud.New(t)
	f := newPyCOEDBClusterFixture(t, cloud, pyCOEDBClusterList)
	service := pyCOEDBService(cloud, "")
	ctx := context.Background()
	id, err := service.UpdateCOECluster(ctx, "prod", []clusters.UpdateOpts{{Op: clusters.ReplaceOp, Path: "/node_count", Value: 0}})
	if err != nil || id != "cl-1" || f.patches.Load() != 1 {
		t.Fatalf("id=%q err=%v patches=%d", id, err, f.patches.Load())
	}
	if f.patchBody != `[{"op":"replace","path":"/node_count","value":0}]` {
		t.Fatalf("patch body=%s", f.patchBody)
	}
	if _, err := service.UpdateCOECluster(ctx, "nope", []clusters.UpdateOpts{{Op: clusters.RemoveOp, Path: "/x"}}); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	if _, err := service.UpdateCOECluster(ctx, "stage", nil); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatalf("ambiguous: %v", err)
	}
	if f.patches.Load() != 1 {
		t.Fatalf("failed resolutions must not send PATCH: %d", f.patches.Load())
	}
}

type pyCOEDBTemplateFixture struct {
	gets, lists, deletes, patches atomic.Int32
	deleteStatus                  int
}

func newPyCOEDBTemplateFixture(cloud *testcloud.Cloud, getStatuses map[string]int) *pyCOEDBTemplateFixture {
	f := &pyCOEDBTemplateFixture{deleteStatus: 204}
	cloud.Mux.HandleFunc("GET /v1/clustertemplates/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.gets.Add(1)
		if status, ok := getStatuses[r.PathValue("id")]; ok && status != 200 {
			testcloud.JSON(w, status, `{}`)
			return
		}
		if r.PathValue("id") == "tpl-1" {
			testcloud.JSON(w, 200, magnumTemplateBody)
			return
		}
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v1/clustertemplates", func(w http.ResponseWriter, r *http.Request) {
		f.lists.Add(1)
		testcloud.JSON(w, 200, `{"clustertemplates":[{"uuid":"tpl-1","name":"k8s"},{"uuid":"tpl-2","name":"swarm"},{"uuid":"tpl-3","name":"dup"},{"uuid":"tpl-4","name":"dup"}]}`)
	})
	cloud.Mux.HandleFunc("DELETE /v1/clustertemplates/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.deletes.Add(1)
		w.WriteHeader(f.deleteStatus)
	})
	cloud.Mux.HandleFunc("PATCH /v1/clustertemplates/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.patches.Add(1)
		testcloud.JSON(w, 200, `{"uuid":"`+r.PathValue("id")+`","name":"patched"}`)
	})
	return f
}

func TestPyCOEDBDeleteClusterTemplateTriesIDFirstThenExactNameWithOneDelete(t *testing.T) {
	cloud := testcloud.New(t)
	f := newPyCOEDBTemplateFixture(cloud, map[string]int{"swarm": 400, "dup": 403})
	service := pyCOEDBService(cloud, "")
	ctx := context.Background()
	if ok, err := service.DeleteClusterTemplate(ctx, "tpl-1"); err != nil || !ok || f.lists.Load() != 0 || f.deletes.Load() != 1 {
		t.Fatalf("by id: ok=%v err=%v lists=%d deletes=%d", ok, err, f.lists.Load(), f.deletes.Load())
	}
	// 400 and 403 also fall through to the exact-name list match, as 404 does.
	if ok, err := service.DeleteClusterTemplate(ctx, "swarm"); err != nil || !ok || f.lists.Load() != 1 || f.deletes.Load() != 2 {
		t.Fatalf("400 fallback: ok=%v err=%v lists=%d deletes=%d", ok, err, f.lists.Load(), f.deletes.Load())
	}
	if ok, err := service.DeleteClusterTemplate(ctx, "dup"); ok || !errors.Is(err, resource.ErrAmbiguous) || f.deletes.Load() != 2 {
		t.Fatalf("duplicates: ok=%v err=%v deletes=%d", ok, err, f.deletes.Load())
	}
	if ok, err := service.DeleteClusterTemplate(ctx, "none"); err != nil || ok || f.deletes.Load() != 2 {
		t.Fatalf("missing: ok=%v err=%v deletes=%d", ok, err, f.deletes.Load())
	}
	// Names are exact here: a glob is not expanded by the template helper.
	if ok, err := service.DeleteClusterTemplate(ctx, "k8*"); err != nil || ok {
		t.Fatalf("glob must not match: ok=%v err=%v", ok, err)
	}
	if ok, err := service.DeleteClusterTemplate(ctx, ""); ok || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("empty: ok=%v err=%v", ok, err)
	}
}

func TestPyCOEDBDeleteClusterTemplateNonFallbackGetErrorsAndDeleteStatuses(t *testing.T) {
	cloud := testcloud.New(t)
	f := newPyCOEDBTemplateFixture(cloud, map[string]int{"boom": 500})
	service := pyCOEDBService(cloud, "")
	ctx := context.Background()
	if ok, err := service.DeleteClusterTemplate(ctx, "boom"); ok || magnumUnexpected(err) != 500 || f.lists.Load() != 0 || f.deletes.Load() != 0 {
		t.Fatalf("500 must not fall back: ok=%v err=%v lists=%d", ok, err, f.lists.Load())
	}
	f.deleteStatus = 500
	if ok, err := service.DeleteClusterTemplate(ctx, "tpl-1"); ok || magnumUnexpected(err) != 500 || f.deletes.Load() != 1 {
		t.Fatalf("delete 500: ok=%v err=%v deletes=%d", ok, err, f.deletes.Load())
	}
	f.deleteStatus = 404
	if ok, err := service.DeleteClusterTemplate(ctx, "tpl-1"); err != nil || !ok || f.deletes.Load() != 2 {
		t.Fatalf("delete 404: ok=%v err=%v deletes=%d", ok, err, f.deletes.Load())
	}
}

func TestPyCOEDBUpdateClusterTemplatePatchesResolvedUUIDAndReportsMissing(t *testing.T) {
	cloud := testcloud.New(t)
	f := newPyCOEDBTemplateFixture(cloud, nil)
	service := pyCOEDBService(cloud, "")
	ctx := context.Background()
	value, err := service.UpdateClusterTemplate(ctx, "swarm", []clustertemplates.UpdateOpts{{Op: clustertemplates.ReplaceOp, Path: "/public", Value: false}})
	if err != nil || value == nil || value.UUID != "tpl-2" || value.Name != "patched" || f.patches.Load() != 1 {
		t.Fatalf("value=%+v err=%v patches=%d", value, err, f.patches.Load())
	}
	if _, err := service.UpdateClusterTemplate(ctx, "none", nil); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	if _, err := service.UpdateClusterTemplate(ctx, "dup", nil); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatalf("ambiguous: %v", err)
	}
	if f.patches.Load() != 1 {
		t.Fatalf("failed resolutions must not send PATCH: %d", f.patches.Load())
	}
}
