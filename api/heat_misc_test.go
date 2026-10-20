package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/apiversions"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/buildinfo"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/resourcetypes"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacktemplates"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const heatMiscTemplate = `{"heat_template_version":"2018-08-31","resources":{}}`

func heatMiscClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	return cloud.Client("orchestration", "/heat/v1/project")
}

func heatMiscTemplateOpts() *stacks.Template {
	return &stacks.Template{TE: stacks.TE{Bin: []byte(heatMiscTemplate)}}
}

func heatMiscUnexpected(t *testing.T, err error, status int) {
	t.Helper()
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &unexpected) || unexpected.Actual != status {
		t.Fatalf("want unexpected response code %d, got %v", status, err)
	}
}

func heatMiscNoRequests(t *testing.T, cloud *testcloud.Cloud) {
	t.Helper()
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
}

func heatMiscBody(t *testing.T, r *http.Request) map[string]json.RawMessage {
	t.Helper()
	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("request body: %v", err)
	}
	return body
}

func heatMiscStrings(body map[string]json.RawMessage) map[string]string {
	got := map[string]string{}
	for key, value := range body {
		got[key] = string(value)
	}
	return got
}

func TestHeatMiscAPIVersionsListUsesUnversionedRootAndSinglePage(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /heat/{$}", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"versions":[{"id":"v1.0","status":"CURRENT","links":[{"rel":"self","href":"http://example.test/v1/"}]}]}`)
	})
	var got []string
	for value, err := range apiversions.New(heatMiscClient(cloud)).ListVersions(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		if len(value.Links) != 1 || value.Links[0].Rel != "self" {
			t.Fatalf("value=%+v", value)
		}
		got = append(got, value.ID+":"+value.Status)
	}
	if !reflect.DeepEqual(got, []string{"v1.0:CURRENT"}) || calls.Load() != 1 {
		t.Fatalf("versions=%v calls=%d", got, calls.Load())
	}
}

func TestHeatMiscAPIVersionsListStatusCodesAndCancellation(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	var status atomic.Int32
	status.Store(204)
	cloud.Mux.HandleFunc("GET /heat/{$}", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if int(status.Load()) == 204 {
			w.WriteHeader(204)
			return
		}
		testcloud.JSON(w, int(status.Load()), `{}`)
	})
	api := apiversions.New(heatMiscClient(cloud))
	count := 0
	for _, err := range api.ListVersions(context.Background()) {
		if err != nil {
			t.Fatalf("204 is an empty list: %v", err)
		}
		count++
	}
	if count != 0 {
		t.Fatalf("count=%d", count)
	}
	status.Store(404)
	for _, err := range api.ListVersions(context.Background()) {
		heatMiscUnexpected(t, err, 404)
	}
	before := calls.Load()
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, err := range api.ListVersions(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled list: %v", err)
		}
	}
	if calls.Load() != before {
		t.Fatalf("canceled call reached HTTP: %d", calls.Load()-before)
	}
}

func TestHeatMiscBuildInfoGetDecodesRevisionsAndKeepsErrors(t *testing.T) {
	cloud := testcloud.New(t)
	var status atomic.Int32
	status.Store(200)
	cloud.Mux.HandleFunc("GET /heat/v1/project/build_info", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, int(status.Load()), `{"api":{"revision":"1.0"},"engine":{"revision":"2.0"}}`)
	})
	api := buildinfo.New(heatMiscClient(cloud))
	value, err := api.Get(context.Background())
	if err != nil || value == nil || value.API.Revision != "1.0" || value.Engine.Revision != "2.0" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	status.Store(201)
	if _, err := api.Get(context.Background()); err == nil {
		t.Fatal("201 must be rejected")
	} else {
		heatMiscUnexpected(t, err, 201)
	}
	status.Store(403)
	if _, err := api.Get(context.Background()); err == nil {
		t.Fatal("403 must be an error")
	} else {
		heatMiscUnexpected(t, err, 403)
	}
	var calls atomic.Int32
	slow := testcloud.New(t)
	slow.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	slowAPI := buildinfo.New(heatMiscClient(slow))
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := slowAPI.Get(canceled); !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatalf("pre-canceled get: %v calls=%d", err, calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := slowAPI.Get(deadline); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}

func TestHeatMiscResourceTypesListQueryAndBothResponseShapes(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /heat/v1/project/resource_types", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		if r.URL.Query().Get("with_description") == "true" {
			testcloud.JSON(w, 200, `{"resource_types":[{"resource_type":"OS::Nova::Server","description":"server"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"resource_types":["OS::Nova::Server","OS::Heat::Stack"]}`)
	})
	api := resourcetypes.New(heatMiscClient(cloud))
	ctx := context.Background()
	plain, err := api.List(ctx)
	if err != nil || len(plain) != 2 || plain[1].ResourceType != "OS::Heat::Stack" || plain[0].Description != "" {
		t.Fatalf("plain=%+v error=%v", plain, err)
	}
	detailed, err := api.List(ctx,
		resourcetypes.WithListOptions(resourcetypes.ListOpts{NameRegex: "OS::Nova.*", SupportStatus: resourcetypes.SupportStatusSupported, WithDescription: true}),
		resourcetypes.WithListQuery("vendor", "x"))
	if err != nil || len(detailed) != 1 || detailed[0].Description != "server" {
		t.Fatalf("detailed=%+v error=%v", detailed, err)
	}
	if len(queries) != 2 || queries[0].Has("name") || queries[0].Has("support_status") || queries[0].Has("with_description") {
		t.Fatalf("default query=%v", queries)
	}
	q := queries[1]
	if q.Get("name") != "OS::Nova.*" || q.Get("support_status") != "SUPPORTED" || q.Get("with_description") != "true" || q.Get("vendor") != "x" {
		t.Fatalf("filtered query=%v", q)
	}
}

func TestHeatMiscResourceTypesSchemaAndGenerateTemplate(t *testing.T) {
	cloud := testcloud.New(t)
	var templateQueries []url.Values
	cloud.Mux.HandleFunc("GET /heat/v1/project/resource_types/OS::Nova::Server", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"resource_type":"OS::Nova::Server","support_status":{"status":"SUPPORTED","previous_status":{"status":"UNSUPPORTED"}},"attributes":{"name":{"type":"string"}},"properties":{"flavor":{"type":"string","required":true,"immutable":false,"update_allowed":false,"constraints":[{"length":{"min":1}}]},"config_drive":{"type":"boolean","required":false,"default":false,"immutable":false,"update_allowed":true}}}`)
	})
	cloud.Mux.HandleFunc("GET /heat/v1/project/resource_types/OS::Nova::Server/template", func(w http.ResponseWriter, r *http.Request) {
		templateQueries = append(templateQueries, r.URL.Query())
		testcloud.JSON(w, 200, `{"heat_template_version":"2016-10-14","resources":{"server":{"type":"OS::Nova::Server"}}}`)
	})
	cloud.Mux.HandleFunc("GET /heat/v1/project/resource_types/OS::Missing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	api := resourcetypes.New(heatMiscClient(cloud))
	ctx := context.Background()
	schema, err := api.GetSchema(ctx, "OS::Nova::Server")
	if err != nil || schema.ResourceType != "OS::Nova::Server" || schema.SupportStatus.Status != resourcetypes.SupportStatusSupported ||
		schema.SupportStatus.PreviousStatus == nil || schema.SupportStatus.PreviousStatus.Status != resourcetypes.SupportStatusUnsupported ||
		schema.Attributes["name"].Type != resourcetypes.StringProperty {
		t.Fatalf("schema=%+v error=%v", schema, err)
	}
	flavor, drive := schema.Properties["flavor"], schema.Properties["config_drive"]
	if !flavor.Required || flavor.Immutable || len(flavor.Constraints) != 1 || flavor.Constraints[0].Length == nil || flavor.Constraints[0].Length.Min != 1 || drive.Default != false || !drive.UpdateAllowed {
		t.Fatalf("properties flavor=%+v drive=%+v", flavor, drive)
	}
	if _, err := api.GetSchema(ctx, "OS::Missing"); err == nil {
		t.Fatal("missing schema must be an error")
	} else {
		heatMiscUnexpected(t, err, 404)
	}
	hot, err := api.GenerateTemplate(ctx, "OS::Nova::Server", resourcetypes.GenerateTemplateOpts{})
	if err != nil || hot["heat_template_version"] != "2016-10-14" {
		t.Fatalf("template=%v error=%v", hot, err)
	}
	if _, err := api.GenerateTemplate(ctx, "OS::Nova::Server", resourcetypes.GenerateTemplateOpts{TemplateType: resourcetypes.TemplateTypeCFn}, resourcetypes.WithGenerateTemplateQuery("vendor", "x")); err != nil {
		t.Fatal(err)
	}
	if len(templateQueries) != 2 || templateQueries[0].Get("template_type") != "hot" || templateQueries[1].Get("template_type") != "cfn" || templateQueries[1].Get("vendor") != "x" {
		t.Fatalf("template queries=%v", templateQueries)
	}
}

func TestHeatMiscResourceTypesRejectNilOptionsBeforeHTTPAndHonorCancellation(t *testing.T) {
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
	api := resourcetypes.New(heatMiscClient(cloud))
	ctx := context.Background()
	if _, err := api.List(ctx, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil list option: %v", err)
	}
	if _, err := api.GenerateTemplate(ctx, "OS::Nova::Server", resourcetypes.GenerateTemplateOpts{}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil generate option: %v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := api.List(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled list: %v", err)
	}
	if _, err := api.GetSchema(canceled, "OS::Nova::Server"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled schema: %v", err)
	}
	if _, err := api.GenerateTemplate(canceled, "OS::Nova::Server", resourcetypes.GenerateTemplateOpts{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled template: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("invalid or canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(ctx, 50*time.Millisecond)
	defer stop()
	if _, err := api.List(deadline); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}

func TestHeatMiscStackTemplateGetReturnsIndentedBodyAndKeepsMissingError(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /heat/v1/project/stacks/app/s-1/template", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"heat_template_version":"2018-08-31","parameters":{"flag":{"type":"boolean","default":false}}}`)
	})
	cloud.Mux.HandleFunc("GET /heat/v1/project/stacks/app/missing/template", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	api := stacktemplates.New(heatMiscClient(cloud))
	data, err := api.Get(context.Background(), "app", "s-1")
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil || decoded["heat_template_version"] != "2018-08-31" {
		t.Fatalf("data=%s error=%v", data, err)
	}
	if want := "{\n  \"heat_template_version\""; string(data[:len(want)]) != want {
		t.Fatalf("template is re-encoded with two-space indentation: %q", data)
	}
	if decoded["parameters"].(map[string]any)["flag"].(map[string]any)["default"] != false {
		t.Fatalf("false default lost: %s", data)
	}
	if data, err := api.Get(context.Background(), "app", "missing"); data != nil || err == nil {
		t.Fatalf("data=%s error=%v", data, err)
	} else {
		heatMiscUnexpected(t, err, 404)
	}
}

func TestHeatMiscStackTemplateValidateWireShapeAndStatusCode(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {201, false}, {202, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /heat/v1/project/validate", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			got := heatMiscStrings(heatMiscBody(t, r))
			want := map[string]string{"template": `"{\"heat_template_version\":\"2018-08-31\"}"`, "template_url": `""`, "vendor": `false`}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			testcloud.JSON(w, tc.status, `{"Description":"demo","Parameters":{"flag":{"Type":"Boolean","Default":false}},"ParameterGroups":{}}`)
		})
		value, err := stacktemplates.New(heatMiscClient(cloud)).Validate(context.Background(),
			stacktemplates.ValidateOpts{Template: `{"heat_template_version":"2018-08-31"}`}, stacktemplates.WithValidateField("vendor", false))
		if tc.success != (err == nil) || calls.Load() != 1 {
			t.Fatalf("status %d error=%v calls=%d", tc.status, err, calls.Load())
		}
		if tc.success && (value == nil || value.Description != "demo" || value.Parameters["flag"].(map[string]any)["Default"] != false || value.ParameterGroups == nil) {
			t.Fatalf("value=%+v", value)
		}
		if !tc.success {
			heatMiscUnexpected(t, err, tc.status)
		}
	}
}

func TestHeatMiscStackTemplateValidateSendsTemplateURLAndRejectsBadInputBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /heat/v1/project/validate", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if got := heatMiscStrings(heatMiscBody(t, r)); !reflect.DeepEqual(got, map[string]string{"template": `""`, "template_url": `"https://example.test/t.yaml"`}) {
			t.Errorf("body=%v", got)
		}
		testcloud.JSON(w, 200, `{"Description":"remote"}`)
	})
	api := stacktemplates.New(heatMiscClient(cloud))
	ctx := context.Background()
	if value, err := api.Validate(ctx, stacktemplates.ValidateOpts{TemplateURL: "https://example.test/t.yaml"}); err != nil || value.Description != "remote" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	callsBefore := calls.Load()
	if _, err := api.Validate(ctx, stacktemplates.ValidateOpts{}); err == nil {
		t.Fatal("a template or template URL is required")
	}
	if _, err := api.Validate(ctx, stacktemplates.ValidateOpts{Template: "x"}, stacktemplates.WithValidateField("template", "y")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected field: %v", err)
	}
	if _, err := api.Validate(ctx, stacktemplates.ValidateOpts{Template: "x"}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option: %v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := api.Validate(canceled, stacktemplates.ValidateOpts{Template: "x"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled validate: %v", err)
	}
	if _, err := api.Get(canceled, "app", "s-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if calls.Load() != callsBefore {
		t.Fatalf("invalid or canceled calls reached HTTP: %d", calls.Load()-callsBefore)
	}
}

func TestHeatMiscStackCreateSendsStringTemplateAndPreservesFalseWithoutPolling(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{201, true}, {202, true}, {200, false}, {204, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /heat/v1/project/stacks", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			got := heatMiscStrings(heatMiscBody(t, r))
			want := map[string]string{
				"stack_name": `"app"`, "template": `"{\"heat_template_version\":\"2018-08-31\",\"resources\":{}}"`, "disable_rollback": `false`,
				"parameters": `{"flag":false,"n":0}`, "tags": `"a,b"`, "environment": `"parameters: {}\n"`, "vendor": `{"x":null}`,
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			testcloud.JSON(w, tc.status, `{"stack":{"id":"s-1","links":[{"rel":"self","href":"http://example.test/stacks/app/s-1"}]}}`)
		})
		disable := false
		created, err := stacks.New(heatMiscClient(cloud)).Create(context.Background(), stacks.CreateOpts{
			Name: "app", TemplateOpts: heatMiscTemplateOpts(), DisableRollback: &disable,
			EnvironmentOpts: &stacks.Environment{TE: stacks.TE{Bin: []byte("parameters: {}\n")}},
			Parameters:      map[string]any{"flag": false, "n": 0}, Tags: []string{"a", "b"},
		}, stacks.WithCreateField("vendor", map[string]any{"x": nil}))
		if tc.success != (err == nil) || calls.Load() != 1 {
			t.Fatalf("status %d error=%v calls=%d", tc.status, err, calls.Load())
		}
		if tc.success && (created == nil || created.ID != "s-1" || len(created.Links) != 1) {
			t.Fatalf("created=%+v", created)
		}
		if !tc.success {
			heatMiscUnexpected(t, err, tc.status)
		}
	}
}

func TestHeatMiscStackCreateOmitsZeroTimeoutAndNeverReplaysServerErrors(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /heat/v1/project/stacks", func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		got := heatMiscBody(t, r)
		for _, key := range []string{"timeout_mins", "disable_rollback", "parameters", "tags", "environment", "files"} {
			if _, ok := got[key]; ok {
				t.Errorf("omitted input %q was sent: %s", key, got[key])
			}
		}
		if n == 1 {
			testcloud.JSON(w, 503, `{"error":"busy"}`)
			return
		}
		testcloud.JSON(w, 201, `{"stack":{"id":"s-2"}}`)
	})
	api := stacks.New(heatMiscClient(cloud))
	_, err := api.Create(context.Background(), stacks.CreateOpts{Name: "app", TemplateOpts: heatMiscTemplateOpts()})
	heatMiscUnexpected(t, err, 503)
	if calls.Load() != 1 {
		t.Fatalf("uncertain mutation was replayed: %d calls", calls.Load())
	}
}

func TestHeatMiscStackAdoptWireShapeAndStatusCodes(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{201, true}, {202, true}, {200, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /heat/v1/project/stacks", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			got := heatMiscStrings(heatMiscBody(t, r))
			want := map[string]string{
				"stack_name": `"adopted"`, "adopt_stack_data": `"{\"resources\":{}}"`, "template": `"{\"heat_template_version\":\"2018-08-31\",\"resources\":{}}"`,
				"disable_rollback": `false`, "timeout_mins": `30`,
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			testcloud.JSON(w, tc.status, `{"stack":{"id":"s-3"}}`)
		})
		disable := false
		created, err := stacks.New(heatMiscClient(cloud)).Adopt(context.Background(), stacks.AdoptOpts{
			Name: "adopted", AdoptStackData: `{"resources":{}}`, TemplateOpts: heatMiscTemplateOpts(), DisableRollback: &disable, Timeout: 30,
		})
		if tc.success != (err == nil) || calls.Load() != 1 || (tc.success && created.ID != "s-3") {
			t.Fatalf("status %d created=%+v error=%v calls=%d", tc.status, created, err, calls.Load())
		}
		if !tc.success {
			heatMiscUnexpected(t, err, tc.status)
		}
	}
}

func TestHeatMiscStackPreviewWireShapeRequires200AndDecodesPreview(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {201, false}, {202, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /heat/v1/project/stacks/preview", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			got := heatMiscStrings(heatMiscBody(t, r))
			want := map[string]string{
				"stack_name": `"app"`, "timeout_mins": `5`, "template": `"{\"heat_template_version\":\"2018-08-31\",\"resources\":{}}"`, "disable_rollback": `false`, "vendor": `0`,
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			testcloud.JSON(w, tc.status, `{"stack":{"id":"s-4","stack_name":"app","disable_rollback":false,"timeout_mins":5,"creation_time":"2026-01-02T03:04:05Z","updated_time":null,"parameters":{"flag":"false"},"resources":[{"resource_name":"server"}],"capabilities":[],"links":[]}}`)
		})
		disable := false
		value, err := stacks.New(heatMiscClient(cloud)).Preview(context.Background(), stacks.PreviewOpts{
			Name: "app", Timeout: 5, TemplateOpts: heatMiscTemplateOpts(), DisableRollback: &disable,
		}, stacks.WithPreviewField("vendor", 0))
		if tc.success != (err == nil) || calls.Load() != 1 {
			t.Fatalf("status %d error=%v calls=%d", tc.status, err, calls.Load())
		}
		if tc.success && (value == nil || value.Name != "app" || value.DisableRollback || value.Timeout != 5 || value.CreationTime.IsZero() || !value.UpdatedTime.IsZero() || len(value.Resources) != 1 || value.Parameters["flag"] != "false") {
			t.Fatalf("value=%+v", value)
		}
		if !tc.success {
			heatMiscUnexpected(t, err, tc.status)
		}
	}
}

func TestHeatMiscStackMutationsRejectInvalidInputBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	heatMiscNoRequests(t, cloud)
	api := stacks.New(heatMiscClient(cloud))
	ctx := context.Background()
	create := stacks.CreateOpts{Name: "app", TemplateOpts: heatMiscTemplateOpts()}
	if _, err := api.Create(ctx, create, stacks.WithCreateField("stack_name", "other")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected create field: %v", err)
	}
	if _, err := api.Create(ctx, create, stacks.WithCreateField("template", "other")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("template conflict: %v", err)
	}
	if _, err := api.Create(ctx, create, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil create option: %v", err)
	}
	if _, err := api.Create(ctx, stacks.CreateOpts{Name: "app"}); err == nil {
		t.Fatal("missing template must fail before HTTP")
	}
	if _, err := api.Create(ctx, stacks.CreateOpts{TemplateOpts: heatMiscTemplateOpts()}); err == nil {
		t.Fatal("missing name must fail before HTTP")
	}
	adopt := stacks.AdoptOpts{Name: "app", AdoptStackData: "{}", TemplateOpts: heatMiscTemplateOpts()}
	if _, err := api.Adopt(ctx, adopt, stacks.WithAdoptField("adopt_stack_data", "x")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected adopt field: %v", err)
	}
	if _, err := api.Adopt(ctx, adopt, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil adopt option: %v", err)
	}
	if _, err := api.Adopt(ctx, stacks.AdoptOpts{Name: "app", TemplateOpts: heatMiscTemplateOpts()}); err == nil {
		t.Fatal("missing adopt data must fail before HTTP")
	}
	preview := stacks.PreviewOpts{Name: "app", Timeout: 1, TemplateOpts: heatMiscTemplateOpts()}
	if _, err := api.Preview(ctx, preview, stacks.WithPreviewField("timeout_mins", 2)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected preview field: %v", err)
	}
	if _, err := api.Preview(ctx, preview, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil preview option: %v", err)
	}
	if _, err := api.Preview(ctx, stacks.PreviewOpts{Name: "app", Timeout: 1}); err == nil {
		t.Fatal("missing preview template must fail before HTTP")
	}
}

func TestHeatMiscStackPreviewSendsZeroTimeoutExplicitly(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /heat/v1/project/stacks/preview", func(w http.ResponseWriter, r *http.Request) {
		if got := heatMiscBody(t, r); string(got["timeout_mins"]) != `0` {
			t.Errorf("timeout_mins=%s", got["timeout_mins"])
		}
		testcloud.JSON(w, 200, `{"stack":{"id":"s-5"}}`)
	})
	value, err := stacks.New(heatMiscClient(cloud)).Preview(context.Background(), stacks.PreviewOpts{Name: "app", TemplateOpts: heatMiscTemplateOpts()})
	if err != nil || value == nil || value.ID != "s-5" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
}

func TestHeatMiscStackMutationsHonorCancellationAndDeadline(t *testing.T) {
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
	api := stacks.New(heatMiscClient(cloud))
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	create := stacks.CreateOpts{Name: "app", TemplateOpts: heatMiscTemplateOpts()}
	adopt := stacks.AdoptOpts{Name: "app", AdoptStackData: "{}", TemplateOpts: heatMiscTemplateOpts()}
	preview := stacks.PreviewOpts{Name: "app", Timeout: 1, TemplateOpts: heatMiscTemplateOpts()}
	if _, err := api.Create(canceled, create); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled create: %v", err)
	}
	if _, err := api.Adopt(canceled, adopt); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled adopt: %v", err)
	}
	if _, err := api.Preview(canceled, preview); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled preview: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Create(deadline, create); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
