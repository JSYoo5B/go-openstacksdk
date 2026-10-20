package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/metric/v1/metrics"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// The metric/v1 package binds the Prometheus-style API, not Gnocchi.

func newGnocchiMetrics(t *testing.T, cloud *testcloud.Cloud) *metrics.API {
	t.Helper()
	return metrics.New(cloud.Client("metric", "/api/v1"))
}

func gnocchiUnexpected(err error) int {
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if errors.As(err, &unexpected) {
		return unexpected.Actual
	}
	return 0
}

func TestGnocchiMetricQuerySendsTypedAndExtensionQueries(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	var headers []string
	cloud.Mux.HandleFunc("GET /api/v1/query", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		headers = append(headers, r.Header.Get("OpenStack-API-Version"))
		testcloud.JSON(w, 200, `{"status":"success","data":{"resultType":"vector","result":[{"metric":{"__name__":"up","job":"node"},"value":[1700000000,"1"]}]}}`)
	})
	api := newGnocchiMetrics(t, cloud)
	data, err := api.Query(context.Background(), metrics.QueryOpts{Query: `up{job="node"}`, Time: "2026-01-02T03:04:05Z", Timeout: "5s"}, metrics.WithQueryQuery("vendor", "x"))
	if err != nil || data == nil || data.ResultType != "vector" || len(data.Result) != 1 || data.Result[0].Metric["job"] != "node" || len(data.Result[0].Value) != 2 {
		t.Fatalf("data=%+v error=%v", data, err)
	}
	want := url.Values{"query": {`up{job="node"}`}, "time": {"2026-01-02T03:04:05Z"}, "timeout": {"5s"}, "vendor": {"x"}}
	if !reflect.DeepEqual(queries[0], want) || headers[0] != "" {
		t.Fatalf("query=%v headers=%v", queries[0], headers)
	}
	// Omitted optional values are not sent.
	if _, err := api.Query(context.Background(), metrics.QueryOpts{Query: "up"}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(queries[1], url.Values{"query": {"up"}}) {
		t.Fatalf("minimal query=%v", queries[1])
	}
	// An extension query replaces a typed value of the same name.
	if _, err := api.Query(context.Background(), metrics.QueryOpts{Query: "up"}, metrics.WithQueryQuery("query", "down")); err != nil {
		t.Fatal(err)
	}
	if queries[2].Get("query") != "down" {
		t.Fatalf("override query=%v", queries[2])
	}
}

func TestGnocchiMetricReadOperationsSendRepeatedMatchersAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	seen := map[string]url.Values{}
	record := func(name, body string) {
		cloud.Mux.HandleFunc("GET /api/v1/"+name, func(w http.ResponseWriter, r *http.Request) {
			seen[name] = r.URL.Query()
			testcloud.JSON(w, 200, body)
		})
	}
	record("labels", `{"status":"success","data":["__name__","job"]}`)
	record("label/job/values", `{"status":"success","data":["node","api"]}`)
	record("series", `{"status":"success","data":[{"__name__":"up","job":"node"}]}`)
	record("targets", `{"status":"success","data":{"activeTargets":[{"scrapePool":"p","scrapeUrl":"http://t/metrics","health":"up","lastScrapeDuration":0.5,"labels":{"job":"node"}}],"droppedTargets":[{"discoveredLabels":{"a":"b"}}]}}`)
	record("status/runtimeinfo", `{"status":"success","data":{"startTime":"t0","reloadConfigSuccess":false,"corruptionCount":0,"goroutineCount":7,"GOMAXPROCS":4,"storageRetention":"15d"}}`)
	api := newGnocchiMetrics(t, cloud)
	ctx := context.Background()
	match := []string{`up`, `{job="node"}`}
	labels, err := api.Labels(ctx, metrics.LabelsOpts{Match: match, Start: "s", End: "e"}, metrics.WithLabelsQuery("vendor", "x"))
	if err != nil || !reflect.DeepEqual(labels, []string{"__name__", "job"}) {
		t.Fatalf("labels=%v error=%v", labels, err)
	}
	if want := (url.Values{"match[]": match, "start": {"s"}, "end": {"e"}, "vendor": {"x"}}); !reflect.DeepEqual(seen["labels"], want) {
		t.Fatalf("labels query=%v", seen["labels"])
	}
	values, err := api.LabelValues(ctx, "job", metrics.LabelValuesOpts{Match: match, Start: "s"}, metrics.WithLabelValuesQuery("vendor", "y"))
	if err != nil || !reflect.DeepEqual(values, []string{"node", "api"}) {
		t.Fatalf("values=%v error=%v", values, err)
	}
	if want := (url.Values{"match[]": match, "start": {"s"}, "vendor": {"y"}}); !reflect.DeepEqual(seen["label/job/values"], want) {
		t.Fatalf("label values query=%v", seen["label/job/values"])
	}
	series, err := api.Series(ctx, metrics.SeriesOpts{Match: match, End: "e"}, metrics.WithSeriesQuery("vendor", "z"))
	if err != nil || !reflect.DeepEqual(series, []map[string]string{{"__name__": "up", "job": "node"}}) {
		t.Fatalf("series=%v error=%v", series, err)
	}
	if want := (url.Values{"match[]": match, "end": {"e"}, "vendor": {"z"}}); !reflect.DeepEqual(seen["series"], want) {
		t.Fatalf("series query=%v", seen["series"])
	}
	targets, err := api.Targets(ctx, metrics.TargetsOpts{State: "active"}, metrics.WithTargetsQuery("vendor", "t"))
	if err != nil || len(targets.ActiveTargets) != 1 || targets.ActiveTargets[0].Health != "up" || targets.ActiveTargets[0].LastScrapeDuration != 0.5 || len(targets.DroppedTargets) != 1 {
		t.Fatalf("targets=%+v error=%v", targets, err)
	}
	if want := (url.Values{"state": {"active"}, "vendor": {"t"}}); !reflect.DeepEqual(seen["targets"], want) {
		t.Fatalf("targets query=%v", seen["targets"])
	}
	if _, err := api.Targets(ctx, metrics.TargetsOpts{}); err != nil || len(seen["targets"]) != 0 {
		t.Fatalf("empty targets filter must send no query: %v %v", seen["targets"], err)
	}
	info, err := api.RuntimeInfo(ctx)
	if err != nil || info.ReloadConfigSuccess || info.CorruptionCount != 0 || info.GoroutineCount != 7 || info.GOMAXPROCS != 4 || info.StorageRetention != "15d" {
		t.Fatalf("info=%+v error=%v", info, err)
	}
}

func TestGnocchiMetricRejectsMissingInputAndInvalidOptionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := newGnocchiMetrics(t, cloud)
	ctx := context.Background()
	if _, err := api.Query(ctx, metrics.QueryOpts{}); err == nil {
		t.Fatal("missing query expression must fail locally")
	}
	if _, err := api.Series(ctx, metrics.SeriesOpts{}); err == nil {
		t.Fatal("missing series matcher must fail locally")
	}
	if err := api.DeleteSeries(ctx, metrics.WithDeleteSeriesOptions(metrics.DeleteSeriesOpts{})); err == nil {
		t.Fatal("missing delete matcher must fail locally")
	}
	if err := api.DeleteSeries(ctx); err == nil {
		t.Fatal("DeleteSeries without options must fail locally instead of deleting everything")
	}
	checks := map[string]error{}
	_, checks["query"] = api.Query(ctx, metrics.QueryOpts{Query: "up"}, nil)
	_, checks["labels"] = api.Labels(ctx, metrics.LabelsOpts{}, nil)
	_, checks["label values"] = api.LabelValues(ctx, "job", metrics.LabelValuesOpts{}, nil)
	_, checks["series"] = api.Series(ctx, metrics.SeriesOpts{Match: []string{"up"}}, nil)
	_, checks["targets"] = api.Targets(ctx, metrics.TargetsOpts{}, nil)
	checks["delete series"] = api.DeleteSeries(ctx, nil)
	for name, err := range checks {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("nil %s option: %v", name, err)
		}
	}
}

func TestGnocchiMetricPrometheusErrorEnvelopeAndHTTPStatuses(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /api/v1/query", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("query") {
		case "bad":
			testcloud.JSON(w, 200, `{"status":"error","errorType":"bad_data","error":"parse error"}`)
		case "http400":
			testcloud.JSON(w, 400, `{"status":"error","errorType":"bad_data","error":"parse error"}`)
		case "accepted":
			testcloud.JSON(w, 202, `{"status":"success","data":{"resultType":"vector","result":[]}}`)
		case "forbidden":
			testcloud.JSON(w, 403, `{}`)
		default:
			testcloud.JSON(w, 200, `{"status":"success","data":{"resultType":"vector","result":[]}}`)
		}
	})
	api := newGnocchiMetrics(t, cloud)
	ctx := context.Background()
	if _, err := api.Query(ctx, metrics.QueryOpts{Query: "bad"}); err == nil || gnocchiUnexpected(err) != 0 {
		t.Fatalf("HTTP 200 with status=error must fail with the Prometheus error: %v", err)
	}
	for query, want := range map[string]int{"http400": 400, "accepted": 202, "forbidden": 403} {
		if _, err := api.Query(ctx, metrics.QueryOpts{Query: query}); gnocchiUnexpected(err) != want {
			t.Fatalf("%s: %v", query, err)
		}
	}
}

func TestGnocchiMetricAdminMutationsUseExactStatusesAndNeverReplay(t *testing.T) {
	for _, status := range []int{204, 200, 202, 500} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		var queries []url.Values
		var bodies []string
		handle := func(path string) {
			cloud.Mux.HandleFunc("POST /api/v1/admin/tsdb/"+path, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				queries = append(queries, r.URL.Query())
				data, _ := io.ReadAll(r.Body)
				bodies = append(bodies, string(data))
				if status == 204 {
					w.WriteHeader(204)
					return
				}
				testcloud.JSON(w, status, `{}`)
			})
		}
		handle("clean_tombstones")
		handle("delete_series")
		api := newGnocchiMetrics(t, cloud)
		ctx := context.Background()
		match := []string{"up", `{job="node"}`}
		errs := []error{
			api.CleanTombstones(ctx),
			api.DeleteSeries(ctx, metrics.WithDeleteSeriesOptions(metrics.DeleteSeriesOpts{Match: match, Start: "s", End: "e"}), metrics.WithDeleteSeriesQuery("vendor", "x")),
		}
		for i, err := range errs {
			if (status == 204) != (err == nil) || (status != 204 && gnocchiUnexpected(err) != status) {
				t.Fatalf("status %d operation %d error=%v", status, i, err)
			}
		}
		if calls.Load() != 2 {
			t.Fatalf("status %d replayed an uncertain mutation: %d calls", status, calls.Load())
		}
		if !reflect.DeepEqual(queries[0], url.Values{}) || bodies[0] != "" || bodies[1] != "" {
			t.Fatalf("clean query=%v bodies=%q", queries[0], bodies)
		}
		if want := (url.Values{"match[]": match, "start": {"s"}, "end": {"e"}, "vendor": {"x"}}); !reflect.DeepEqual(queries[1], want) {
			t.Fatalf("delete query=%v", queries[1])
		}
	}
}

func TestGnocchiMetricSnapshotRequires200AndDecodesName(t *testing.T) {
	for _, status := range []int{200, 201, 202, 204, 403} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /api/v1/admin/tsdb/snapshot", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if data, _ := io.ReadAll(r.Body); len(data) != 0 {
				t.Errorf("snapshot body=%q", data)
			}
			if status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, status, `{"status":"success","data":{"name":"20260102T030405Z-abc"}}`)
		})
		data, err := newGnocchiMetrics(t, cloud).Snapshot(context.Background())
		if (status == 200) != (err == nil) || (status == 200 && (data == nil || data.Name != "20260102T030405Z-abc")) {
			t.Fatalf("status %d data=%+v error=%v", status, data, err)
		}
		if status != 200 && gnocchiUnexpected(err) != status {
			t.Fatalf("status %d error=%v", status, err)
		}
		if calls.Load() != 1 {
			t.Fatalf("status %d calls=%d", status, calls.Load())
		}
	}
}

func TestGnocchiMetricCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	api := newGnocchiMetrics(t, cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	match := []string{"up"}
	checks := map[string]error{}
	_, checks["query"] = api.Query(canceled, metrics.QueryOpts{Query: "up"})
	_, checks["labels"] = api.Labels(canceled, metrics.LabelsOpts{})
	_, checks["label values"] = api.LabelValues(canceled, "job", metrics.LabelValuesOpts{})
	_, checks["series"] = api.Series(canceled, metrics.SeriesOpts{Match: match})
	_, checks["targets"] = api.Targets(canceled, metrics.TargetsOpts{})
	_, checks["runtime info"] = api.RuntimeInfo(canceled)
	_, checks["snapshot"] = api.Snapshot(canceled)
	checks["clean tombstones"] = api.CleanTombstones(canceled)
	checks["delete series"] = api.DeleteSeries(canceled, metrics.WithDeleteSeriesOptions(metrics.DeleteSeriesOpts{Match: match}))
	for name, err := range checks {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("pre-canceled %s: %v", name, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if err := api.DeleteSeries(deadline, metrics.WithDeleteSeriesOptions(metrics.DeleteSeriesOpts{Match: match})); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("an uncertain delete must not be replayed: %d calls", calls.Load())
	}
}
