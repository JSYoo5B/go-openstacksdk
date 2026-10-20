package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

func heatNativeStacksAPI(cloud *testcloud.Cloud) *stacks.API {
	return stacks.New(cloud.Client("orchestration", "/heat"))
}

func heatNativeRequireStatus(t *testing.T, err error, code int) {
	t.Helper()
	if err == nil || !gophercloud.ResponseCodeIs(err, code) {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

func heatNativeBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Errorf("request body %q: %v", raw, err)
	}
	return body
}

func heatNativeCanceled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

const heatNativeTemplate = `{"heat_template_version":"2013-05-23","resources":{}}`

func TestHeatNativeStacksAbandonRequiresHTTP200AndPreservesPayload(t *testing.T) {
	cloud := testcloud.New(t)
	api := heatNativeStacksAPI(cloud)
	var status atomic.Int32
	status.Store(200)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("DELETE /heat/stacks/app/fixed/abandon", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("X-Auth-Token") != "test-token" {
			t.Errorf("missing token: %v", r.Header)
		}
		switch status.Load() {
		case 200:
			testcloud.JSON(w, 200, `{"status":"COMPLETE","name":"app","template":{"heat_template_version":"2013-05-23"},"action":"ABANDON","id":"fixed","resources":{"r1":{"type":"OS::Heat::None","resource_id":null}},"files":{"child.yaml":"content"},"stack_user_project_id":"sup","project_id":"proj","environment":{"parameters":{"n":0}}}`)
		case 202:
			testcloud.JSON(w, 202, `{"id":"fixed"}`)
		case 1:
			testcloud.JSON(w, 200, `{"id":`)
		default:
			testcloud.JSON(w, int(status.Load()), `{"error":{"message":"denied"}}`)
		}
	})
	value, err := api.Abandon(context.Background(), "app", "fixed")
	if err != nil || value == nil {
		t.Fatalf("value=%v err=%v", value, err)
	}
	want := stacks.AbandonedStack{
		Status: "COMPLETE", Name: "app", Action: "ABANDON", ID: "fixed", StackUserProjectID: "sup", ProjectID: "proj",
		Template:    map[string]any{"heat_template_version": "2013-05-23"},
		Resources:   map[string]any{"r1": map[string]any{"type": "OS::Heat::None", "resource_id": nil}},
		Files:       map[string]string{"child.yaml": "content"},
		Environment: map[string]any{"parameters": map[string]any{"n": float64(0)}},
	}
	if !reflect.DeepEqual(*value, want) {
		t.Fatalf("payload changed:\n got %#v\nwant %#v", *value, want)
	}
	scope := heatScope(t, api, "app", "fixed")
	if scoped, err := scope.Abandon(context.Background()); err != nil || scoped == nil || scoped.ID != "fixed" || scoped.Resources["r1"] == nil {
		t.Fatalf("scope abandon=%v err=%v", scoped, err)
	}
	for _, code := range []int{403, 404, 500} {
		status.Store(int32(code))
		_, err := api.Abandon(context.Background(), "app", "fixed")
		heatNativeRequireStatus(t, err, code)
	}
	status.Store(202)
	_, err = api.Abandon(context.Background(), "app", "fixed")
	heatNativeRequireStatus(t, err, 202)
	status.Store(1)
	if value, err := api.Abandon(context.Background(), "app", "fixed"); err == nil {
		t.Fatalf("malformed JSON decoded as %v", value)
	}
	before := calls.Load()
	if _, err := api.Abandon(heatNativeCanceled(), "app", "fixed"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled abandon err=%v", err)
	}
	if _, err := scope.Abandon(heatNativeCanceled()); !errors.Is(err, context.Canceled) || calls.Load() != before {
		t.Fatalf("canceled scope abandon err=%v calls=%d->%d", err, before, calls.Load())
	}
}

func TestHeatNativeStacksDeleteAcceptsOnly202And204AndKeepsHTTPErrors(t *testing.T) {
	cloud := testcloud.New(t)
	api := heatNativeStacksAPI(cloud)
	var status atomic.Int32
	var deletes atomic.Int32
	cloud.Mux.HandleFunc("DELETE /heat/stacks/app/fixed", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		if r.Header.Get("X-Auth-Token") != "test-token" {
			t.Errorf("missing token: %v", r.Header)
		}
		if status.Load() == 202 || status.Load() == 204 {
			w.WriteHeader(int(status.Load()))
			return
		}
		testcloud.JSON(w, int(status.Load()), `{"error":{"message":"no"}}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/fixed", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"stack":{"stack_name":"app","id":"fixed","stack_status":"CREATE_COMPLETE"}}`)
	})
	for _, code := range []int{202, 204} {
		status.Store(int32(code))
		if err := api.Delete(context.Background(), "app", "fixed"); err != nil {
			t.Fatalf("native delete %d: %v", code, err)
		}
	}
	scope := heatScope(t, api, "app", "fixed")
	status.Store(204)
	before := deletes.Load()
	if err := scope.Delete(context.Background()); err != nil || deletes.Load() != before+1 {
		t.Fatalf("scoped delete err=%v deletes=%d->%d", err, before, deletes.Load())
	}
	for _, code := range []int{200, 401, 403, 409, 500} {
		status.Store(int32(code))
		heatNativeRequireStatus(t, api.Delete(context.Background(), "app", "fixed"), code)
		if code != 200 {
			heatNativeRequireStatus(t, scope.Delete(context.Background(), resource.WithMissingError()), code)
		}
	}
	before = deletes.Load()
	if err := api.Delete(heatNativeCanceled(), "app", "fixed"); !errors.Is(err, context.Canceled) || deletes.Load() != before {
		t.Fatalf("canceled delete err=%v deletes=%d->%d", err, before, deletes.Load())
	}
}

func TestHeatNativeStacksFindUsesSingleIdentityRedirectAndKeepsErrors(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("orchestration", "/heat")
	client.MoreHeaders = map[string]string{"X-Policy": "keep"}
	api := stacks.New(client)
	var hits atomic.Int32
	cloud.Mux.HandleFunc("GET /heat/stacks/app", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.RawQuery != "" {
			t.Errorf("native Find must not add a query: %s", r.URL)
		}
		http.Redirect(w, r, "/heat/stacks/app/fixed", http.StatusFound)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Policy") != "keep" {
			t.Errorf("redirect lost headers: %v", r.Header)
		}
		testcloud.JSON(w, 200, `{"stack":{"stack_name":"app","id":"fixed","stack_status":"UPDATE_COMPLETE","disable_rollback":false,"timeout_mins":0,"parameters":{"x":"y"},"tags":["a"],"outputs":[{"output_key":"k","output_value":null}]}}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/fixed", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Redirect(w, r, "/heat/stacks/app/fixed", http.StatusFound)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/gone", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{"error":{"message":"missing"}}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/denied", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 403, `{}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/bad", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"stack":`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/null", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"stack":null}`)
	})
	for _, identity := range []string{"app", "fixed"} {
		value, err := api.Find(context.Background(), identity)
		if err != nil || value == nil || value.ID != "fixed" || value.Name != "app" || value.Status != "UPDATE_COMPLETE" || value.DisableRollback || value.Parameters["x"] != "y" || len(value.Tags) != 1 || len(value.Outputs) != 1 {
			t.Fatalf("Find(%q)=%#v err=%v", identity, value, err)
		}
	}
	if hits.Load() != 2 {
		t.Fatalf("identity hits=%d", hits.Load())
	}
	_, err := api.Find(context.Background(), "gone")
	heatNativeRequireStatus(t, err, 404)
	_, err = api.Find(context.Background(), "denied")
	heatNativeRequireStatus(t, err, 403)
	if value, err := api.Find(context.Background(), "bad"); err == nil {
		t.Fatalf("malformed Find decoded as %v", value)
	}
	if value, err := api.Find(context.Background(), "null"); err != nil || value != nil {
		t.Fatalf("null stack value=%v err=%v", value, err)
	}
	if _, err := api.Find(heatNativeCanceled(), "app"); !errors.Is(err, context.Canceled) || hits.Load() != 2 {
		t.Fatalf("canceled Find err=%v hits=%d", err, hits.Load())
	}
}

func TestHeatNativeStacksListSerializesEveryTypedFilter(t *testing.T) {
	cloud := testcloud.New(t)
	api := heatNativeStacksAPI(cloud)
	var calls atomic.Int32
	var last atomic.Value
	cloud.Mux.HandleFunc("GET /heat/stacks", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		last.Store(r.URL.Query())
		switch r.URL.Query().Get("mode") {
		case "empty":
			w.WriteHeader(http.StatusNoContent)
		case "bad":
			testcloud.JSON(w, 200, `{"stacks":{}}`)
		case "denied":
			testcloud.JSON(w, 403, `{}`)
		default:
			testcloud.JSON(w, 200, `{"stacks":[{"stack_name":"app","id":"fixed","stack_status":"CREATE_COMPLETE","tags":["t"],"creation_time":"2026-10-01T00:00:00Z"},{"stack_name":"b","id":"two"}],"links":[{"rel":"next","href":"?marker=two"}]}`)
		}
	})
	collect := func(options ...stacks.ListOption) ([]*stacks.ListedStack, error) {
		var out []*stacks.ListedStack
		for value, err := range api.List(context.Background(), options...) {
			if err != nil {
				return out, err
			}
			out = append(out, value)
		}
		return out, nil
	}
	values, err := collect()
	if err != nil || len(values) != 2 || values[0].ID != "fixed" || values[0].Name != "app" || values[0].Status != "CREATE_COMPLETE" || values[0].CreationTime.IsZero() || calls.Load() != 1 {
		t.Fatalf("default list=%v err=%v calls=%d", values, err, calls.Load())
	}
	if q := map[string][]string(last.Load().(url.Values)); len(q) != 0 {
		t.Fatalf("zero ListOpts must send no query, got %v", q)
	}
	full := stacks.ListOpts{
		TenantID: "proj", ID: "id1", Status: "CREATE_COMPLETE", Name: "app", Marker: "m0", Limit: 7,
		SortKey: stacks.SortKey("created_at"), SortDir: stacks.SortDir("desc"),
		AllTenants: true, ShowDeleted: true, ShowNested: true, ShowHidden: true,
		Tags: "a,b", TagsAny: "c,d", NotTags: "e", NotTagsAny: "f,g",
	}
	if _, err := collect(stacks.WithListOptions(full), stacks.WithListQuery("vendor", "a&b")); err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"tenant_id": {"proj"}, "id": {"id1"}, "status": {"CREATE_COMPLETE"}, "name": {"app"}, "marker": {"m0"}, "limit": {"7"},
		"sort_keys": {"created_at"}, "sort_dir": {"desc"},
		"global_tenant": {"true"}, "show_deleted": {"true"}, "show_nested": {"true"}, "show_hidden": {"true"},
		"tags": {"a,b"}, "tags_any": {"c,d"}, "not_tags": {"e"}, "not_tags_any": {"f,g"}, "vendor": {"a&b"},
	}
	if got := map[string][]string(last.Load().(url.Values)); !reflect.DeepEqual(got, want) {
		t.Fatalf("query=%v want %v", got, want)
	}
	if calls.Load() != 2 {
		t.Fatalf("native List must be a single page, calls=%d", calls.Load())
	}
	// False booleans are omitted rather than sent as false.
	if _, err := collect(stacks.WithListOptions(stacks.ListOpts{Name: "only"})); err != nil {
		t.Fatal(err)
	}
	if got := map[string][]string(last.Load().(url.Values)); !reflect.DeepEqual(got, map[string][]string{"name": {"only"}}) {
		t.Fatalf("false booleans leaked: %v", got)
	}
	values, err = collect(stacks.WithListQuery("mode", "empty"))
	if err != nil || len(values) != 0 {
		t.Fatalf("204 list=%v err=%v", values, err)
	}
	if _, err := collect(stacks.WithListQuery("mode", "bad")); err == nil {
		t.Fatal("malformed stacks array was accepted")
	}
	_, err = collect(stacks.WithListQuery("mode", "denied"))
	heatNativeRequireStatus(t, err, 403)
	before := calls.Load()
	for _, err := range api.List(heatNativeCanceled()) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled list err=%v", err)
		}
	}
	if calls.Load() != before {
		t.Fatal("canceled list sent a request")
	}
}

func heatNativeUpdateServer(t *testing.T, cloud *testcloud.Cloud, method string, status *atomic.Int32, bodies chan<- map[string]any) {
	t.Helper()
	cloud.Mux.HandleFunc(method+" /heat/stacks/app/fixed", func(w http.ResponseWriter, r *http.Request) {
		bodies <- heatNativeBody(t, r)
		if r.Header.Get("Content-Type") != "application/json" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Errorf("headers=%v", r.Header)
		}
		code := int(status.Load())
		if code == 201 || code == 202 || code == 204 {
			w.WriteHeader(code)
			return
		}
		testcloud.JSON(w, code, `{"error":{"message":"x"}}`)
	})
}

func TestHeatNativeStacksUpdateSendsPUTTemplateEnvironmentFilesAndExtensions(t *testing.T) {
	cloud := testcloud.New(t)
	api := heatNativeStacksAPI(cloud)
	bodies := make(chan map[string]any, 16)
	var status atomic.Int32
	status.Store(202)
	heatNativeUpdateServer(t, cloud, "PUT", &status, bodies)
	cloud.Mux.HandleFunc("GET /tpl/main.json", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"heat_template_version":"2013-05-23","resources":{"c":{"type":"child.yaml"}}}`)
	})
	cloud.Mux.HandleFunc("GET /tpl/child.yaml", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"heat_template_version":"2013-05-23","resources":{}}`)
	})
	opts := stacks.UpdateOpts{
		TemplateOpts:    &stacks.Template{TE: stacks.TE{Bin: []byte(heatNativeTemplate)}},
		EnvironmentOpts: &stacks.Environment{TE: stacks.TE{Bin: []byte(`{"parameters":{"e":"1"}}`)}},
		Parameters:      map[string]any{"size": 3, "flag": false},
		Timeout:         45,
		Tags:            []string{"a", "b"},
	}
	for _, code := range []int32{201, 202} {
		status.Store(code)
		if err := api.Update(context.Background(), "app", "fixed", opts, stacks.WithUpdateField("disable_rollback", false)); err != nil {
			t.Fatalf("update %d: %v", code, err)
		}
		body := <-bodies
		want := map[string]any{
			"template": heatNativeTemplate, "environment": `{"parameters":{"e":"1"}}`,
			"parameters": map[string]any{"size": float64(3), "flag": false}, "timeout_mins": float64(45),
			"tags": "a,b", "disable_rollback": false,
		}
		if !reflect.DeepEqual(body, want) {
			t.Fatalf("PUT body=%v want %v", body, want)
		}
	}
	// Referenced child templates are merged into files.
	fetched := stacks.UpdateOpts{TemplateOpts: &stacks.Template{TE: stacks.TE{URL: cloud.Server.URL + "/tpl/main.json"}}}
	if err := api.Update(context.Background(), "app", "fixed", fetched, stacks.WithUpdateOptions(fetched)); err != nil {
		t.Fatal(err)
	}
	body := <-bodies
	files, _ := body["files"].(map[string]any)
	if _, ok := files[cloud.Server.URL+"/tpl/child.yaml"]; !ok || len(files) != 1 || !strings.Contains(body["template"].(string), cloud.Server.URL+"/tpl/child.yaml") {
		t.Fatalf("referenced file not merged: %v", body)
	}
	// Omitted optional values stay omitted.
	minimal := stacks.UpdateOpts{TemplateOpts: &stacks.Template{TE: stacks.TE{Bin: []byte(heatNativeTemplate)}}}
	if err := api.Update(context.Background(), "app", "fixed", minimal); err != nil {
		t.Fatal(err)
	}
	if body := <-bodies; !reflect.DeepEqual(body, map[string]any{"template": heatNativeTemplate}) {
		t.Fatalf("minimal body=%v", body)
	}
	// Failures happen before HTTP.
	var tmplErr stacks.ErrTemplateRequired
	if err := api.Update(context.Background(), "app", "fixed", stacks.UpdateOpts{Parameters: map[string]any{"a": 1}}); !errors.As(err, &tmplErr) {
		t.Fatalf("missing template err=%v", err)
	}
	if err := api.Update(context.Background(), "app", "fixed", minimal, stacks.WithUpdateField("template", "override")); err == nil {
		t.Fatal("protected template field override was accepted")
	}
	bad := stacks.UpdateOpts{TemplateOpts: &stacks.Template{TE: stacks.TE{Bin: []byte(`{"x":`)}}}
	if err := api.Update(context.Background(), "app", "fixed", bad); err == nil {
		t.Fatal("unparseable template was accepted")
	}
	if err := api.Update(heatNativeCanceled(), "app", "fixed", minimal); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled update err=%v", err)
	}
	if scope := heatScope(t, api, "app", "fixed"); scope.Update(heatNativeCanceled(), minimal) == nil {
		t.Fatal("canceled scoped update succeeded")
	}
	if len(bodies) != 0 {
		t.Fatalf("pre-HTTP failures sent %d requests", len(bodies))
	}
	// HTTP statuses outside 201/202 fail with their code.
	for _, code := range []int32{200, 204, 400, 403, 409, 500} {
		status.Store(code)
		heatNativeRequireStatus(t, api.Update(context.Background(), "app", "fixed", minimal), int(code))
		<-bodies
	}
	status.Store(202)
	if err := heatScope(t, api, "app", "fixed").Update(context.Background(), minimal); err != nil {
		t.Fatal(err)
	}
	<-bodies
}

func TestHeatNativeStacksUpdatePatchAcceptsSuccessCodesAndPreservesEmptyTags(t *testing.T) {
	cloud := testcloud.New(t)
	api := heatNativeStacksAPI(cloud)
	bodies := make(chan map[string]any, 16)
	var status atomic.Int32
	heatNativeUpdateServer(t, cloud, "PATCH", &status, bodies)
	cloud.Mux.HandleFunc("PATCH /heat/stacks/app/fixed/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected path %s", r.URL)
	})
	patch := stacks.UpdateOpts{Parameters: map[string]any{"size": 1}}
	for _, code := range []int32{202, 204} {
		status.Store(code)
		if err := api.UpdatePatch(context.Background(), "app", "fixed", patch); err != nil {
			t.Fatalf("patch %d: %v", code, err)
		}
		if body := <-bodies; !reflect.DeepEqual(body, map[string]any{"parameters": map[string]any{"size": float64(1)}}) {
			t.Fatalf("patch body=%v", body)
		}
	}
	// A 200 response is also accepted by the native PATCH default codes.
	cloud.Mux.HandleFunc("PATCH /heat/stacks/ok/200", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{}`) })
	if err := api.UpdatePatch(context.Background(), "ok", "200", patch); err != nil {
		t.Fatalf("patch 200: %v", err)
	}
	status.Store(204)
	for _, tc := range []struct {
		name string
		opts stacks.UpdateOpts
		want map[string]any
	}{
		{"nil tags and empty parameters are omitted", stacks.UpdateOpts{Parameters: map[string]any{}}, map[string]any{}},
		{"empty non-nil tags are sent as an empty string", stacks.UpdateOpts{Tags: []string{}}, map[string]any{"tags": ""}},
		{"template and environment are optional but honored", stacks.UpdateOpts{
			TemplateOpts:    &stacks.Template{TE: stacks.TE{Bin: []byte(heatNativeTemplate)}},
			EnvironmentOpts: &stacks.Environment{TE: stacks.TE{Bin: []byte(`{"parameters":{"e":"1"}}`)}},
			Timeout:         9,
		}, map[string]any{"template": heatNativeTemplate, "environment": `{"parameters":{"e":"1"}}`, "timeout_mins": float64(9)}},
	} {
		if err := api.UpdatePatch(context.Background(), "app", "fixed", tc.opts, stacks.WithUpdatePatchField("disable_rollback", false)); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		tc.want["disable_rollback"] = false
		if body := <-bodies; !reflect.DeepEqual(body, tc.want) {
			t.Fatalf("%s: body=%v want %v", tc.name, body, tc.want)
		}
	}
	cloud.Mux.HandleFunc("GET /tpl/main.json", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"heat_template_version":"2013-05-23","resources":{"c":{"type":"child.yaml"}}}`)
	})
	cloud.Mux.HandleFunc("GET /tpl/child.yaml", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"heat_template_version":"2013-05-23","resources":{}}`)
	})
	status.Store(204)
	fetched := stacks.UpdateOpts{TemplateOpts: &stacks.Template{TE: stacks.TE{URL: cloud.Server.URL + "/tpl/main.json"}}}
	if err := api.UpdatePatch(context.Background(), "app", "fixed", fetched); err != nil {
		t.Fatal(err)
	}
	body := <-bodies
	files, _ := body["files"].(map[string]any)
	if _, ok := files[cloud.Server.URL+"/tpl/child.yaml"]; !ok || len(files) != 1 || !strings.Contains(body["template"].(string), cloud.Server.URL+"/tpl/child.yaml") {
		t.Fatalf("referenced file not merged: %v", body)
	}
	for _, code := range []int32{201, 400, 403, 409, 500} {
		status.Store(code)
		heatNativeRequireStatus(t, api.UpdatePatch(context.Background(), "app", "fixed", patch), int(code))
		<-bodies
	}
	if err := api.UpdatePatch(context.Background(), "app", "fixed", stacks.UpdateOpts{TemplateOpts: &stacks.Template{TE: stacks.TE{Bin: []byte(`{"x":`)}}}); err == nil {
		t.Fatal("unparseable template was accepted")
	}
	if err := api.UpdatePatch(context.Background(), "app", "fixed", patch, stacks.WithUpdatePatchField("parameters", 1)); err == nil {
		t.Fatal("protected parameters field override was accepted")
	}
	if err := api.UpdatePatch(heatNativeCanceled(), "app", "fixed", patch); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled patch err=%v", err)
	}
	if len(bodies) != 0 {
		t.Fatalf("pre-HTTP failures sent %d requests", len(bodies))
	}
}
