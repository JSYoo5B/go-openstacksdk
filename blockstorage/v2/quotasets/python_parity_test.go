package quotasets_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonQuotaCall records the wire request openstacksdk block_storage v2 would also send.
type pythonQuotaCall struct{ method, path, query, body string }

func pythonQuotaAPI(t *testing.T, reply func(*http.Request) (int, string)) (*quotasets.API, *testcloud.Recorder) {
	t.Helper()
	calls := testcloud.NewRecorder(func(req *http.Request) (*http.Response, error) {
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	})
	cloud := testcloud.New(t)
	cloud.SetTransport(calls)
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v2/project/"
	return quotasets.New(client), calls
}

// pythonQuotaCalls projects recorded calls onto the compared fields.
func pythonQuotaCalls(rec *testcloud.Recorder) []pythonQuotaCall {
	calls := []pythonQuotaCall{}
	for _, call := range rec.Calls() {
		calls = append(calls, pythonQuotaCall{call.Method, call.Path, call.RawQuery, call.Body})
	}
	return calls
}

const pythonQuotaBase = "/cinder/v2/project/os-quota-sets"

const pythonQuotaRow = `{"quota_set":{"id":"defaults","volumes":10,"snapshots":10,"gigabytes":1000,"per_volume_gigabytes":-1,"backups":10,"backup_gigabytes":1000,"groups":10,"volumes_lvmdriver-1":-1}}`

func TestPythonQuotaSetDefaultsUsesUnscopedRoute(t *testing.T) {
	api, calls := pythonQuotaAPI(t, func(*http.Request) (int, string) { return 200, pythonQuotaRow })
	// get_quota_set_defaults(project) fetches base_path /os-quota-sets/defaults and never puts the project in the URL.
	got, err := api.Get(context.Background(), "defaults")
	if err != nil || got.Volumes != 10 || got.Gigabytes != 1000 || got.PerVolumeGigabytes != -1 || got.Groups != 10 || got.Extra["volumes_lvmdriver-1"] != float64(-1) {
		t.Fatal(got, err)
	}
	// The project-scoped defaults route is a different request.
	if _, err := api.GetDefaults(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
	want := []pythonQuotaCall{
		{http.MethodGet, pythonQuotaBase + "/defaults", "", ""},
		{http.MethodGet, pythonQuotaBase + "/p1/defaults", "", ""},
	}
	if !reflect.DeepEqual(pythonQuotaCalls(calls), want) {
		t.Fatalf("%+v", pythonQuotaCalls(calls))
	}
}

func TestPythonQuotaSetUpdateSendsDirtyAttributes(t *testing.T) {
	api, calls := pythonQuotaAPI(t, func(*http.Request) (int, string) {
		return 200, `{"quota_set":{"volumes":20,"backups":5,"volumes_lvmdriver-1":3}}`
	})
	// update_quota_set(project, volumes=20, backups=5, volumes_lvmdriver-1=3)
	volumes, backups := 20, 5
	got, err := api.Update(context.Background(), "p1", quotasets.UpdateOpts{Volumes: &volumes, Backups: &backups}, quotasets.WithUpdateField("volumes_lvmdriver-1", 3))
	if err != nil || got.Volumes != 20 || got.Backups != 5 || got.Extra["volumes_lvmdriver-1"] != float64(3) {
		t.Fatal(got, err)
	}
	want := []pythonQuotaCall{{http.MethodPut, pythonQuotaBase + "/p1", "", `{"quota_set":{"backups":5,"volumes":20,"volumes_lvmdriver-1":3}}`}}
	if !reflect.DeepEqual(pythonQuotaCalls(calls), want) {
		t.Fatalf("%+v", pythonQuotaCalls(calls))
	}
}

func TestPythonQuotaSetGetAndRevertCannotSendQuery(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonQuotaAPI(t, func(req *http.Request) (int, string) {
		if req.URL.RawQuery == "usage=true" {
			return 200, `{"quota_set":{"id":"p1","volumes":{"limit":10,"in_use":1,"reserved":0,"allocated":0}}}`
		}
		return 200, `{"quota_set":{"id":"p1","volumes":10}}`
	})
	// get_quota_set(project) sends usage=False and get_quota_set(project, usage=True) sends usage=True.
	if _, err := api.Get(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	usage, err := api.GetUsage(ctx, "p1")
	if err != nil || usage.Volumes.Limit != 10 || usage.Volumes.InUse != 1 {
		t.Fatal(usage, err)
	}
	// revert_quota_set(project, **query) forwards query as DELETE params; Go Delete has no query option.
	if err := api.Delete(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	want := []pythonQuotaCall{
		{http.MethodGet, pythonQuotaBase + "/p1", "", ""},
		{http.MethodGet, pythonQuotaBase + "/p1", "usage=true", ""},
		{http.MethodDelete, pythonQuotaBase + "/p1", "", ""},
	}
	if !reflect.DeepEqual(pythonQuotaCalls(calls), want) {
		t.Fatalf("%+v", pythonQuotaCalls(calls))
	}
	// Python accepts any 2xx for the reset; Go accepts only 200.
	strict, _ := pythonQuotaAPI(t, func(*http.Request) (int, string) { return 204, "" })
	if err := strict.Delete(ctx, "p1"); !gophercloud.ResponseCodeIs(err, 204) {
		t.Fatal(err)
	}
}
