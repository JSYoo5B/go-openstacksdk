package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacktemplates"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

func TestPyHeatSenlinValidateTemplateSendsPythonBodyQueryAndKeepsRawResponse(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	type seen struct {
		query string
		body  map[string]json.RawMessage
	}
	requests := make(chan seen, 8)
	cloud.Mux.HandleFunc("POST /heat/v1/project/validate", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]json.RawMessage
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("body %q: %v", raw, err)
		}
		requests <- seen{query: r.URL.RawQuery, body: body}
		w.Header().Set("X-Heat", "1")
		testcloud.JSON(w, 200, `{"Description":"demo","Parameters":{"flag":{"Type":"Boolean","Default":false}},"ParameterGroups":[{"label":"g","parameters":["flag"]}],"Extra":9007199254740993}`)
	})
	api := stacktemplates.New(heatMiscClient(cloud))
	ctx := context.Background()
	url := "https://example.test/t.yaml"

	doc, err := api.ValidateTemplate(ctx, stacktemplates.ValidateTemplateOpts{
		Template:     json.RawMessage(`{"heat_template_version":"2018-08-31","parameters":{"flag":{"type":"boolean","default":false}}}`),
		Environment:  json.RawMessage(`{"parameters":{"flag":false}}`),
		IgnoreErrors: "99001,99002",
	})
	if err != nil || doc == nil || doc.StatusCode != 200 || doc.Header.Get("X-Heat") != "1" {
		t.Fatalf("doc=%+v err=%v", doc, err)
	}
	if string(doc.Body["Extra"]) != "9007199254740993" || doc.Body["ParameterGroups"][0] != '[' {
		t.Fatalf("typed model would reject a list of groups; raw body must be retained: %v", doc.Body)
	}
	got := <-requests
	if got.query != "ignore_errors=99001%2C99002" {
		t.Fatalf("query=%q", got.query)
	}
	if string(got.body["environment"]) != `{"parameters":{"flag":false}}` || string(got.body["template"])[0] != '{' {
		t.Fatalf("body=%v", got.body)
	}
	if _, ok := got.body["template_url"]; ok {
		t.Fatalf("template_url must be omitted when nil: %v", got.body)
	}

	if _, err := api.ValidateTemplate(ctx, stacktemplates.ValidateTemplateOpts{TemplateURL: &url}); err != nil {
		t.Fatal(err)
	}
	got = <-requests
	if got.query != "" || string(got.body["template"]) != "null" || string(got.body["template_url"]) != `"https://example.test/t.yaml"` {
		t.Fatalf("url-only request: query=%q body=%v", got.query, got.body)
	}
	if _, ok := got.body["environment"]; ok {
		t.Fatalf("nil environment must be omitted: %v", got.body)
	}

	empty := ""
	if _, err := api.ValidateTemplate(ctx, stacktemplates.ValidateTemplateOpts{Template: json.RawMessage(`"inline: yaml"`), TemplateURL: &empty}); err != nil {
		t.Fatal(err)
	}
	got = <-requests
	if string(got.body["template_url"]) != `""` || string(got.body["template"]) != `"inline: yaml"` {
		t.Fatalf("explicit empty template_url must be preserved: %v", got.body)
	}
}

func TestPyHeatSenlinValidateTemplateRejectsInvalidInputBeforeHTTPAndKeepsHTTPErrors(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	var status atomic.Int32
	status.Store(400)
	cloud.Mux.HandleFunc("POST /heat/v1/project/validate", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, int(status.Load()), `{"error":{"message":"bad"}}`)
	})
	api := stacktemplates.New(heatMiscClient(cloud))
	ctx := context.Background()
	for name, opts := range map[string]stacktemplates.ValidateTemplateOpts{
		"neither":       {},
		"null template": {Template: json.RawMessage(`null`)},
		"bad template":  {Template: json.RawMessage(`{`)},
		"bad env":       {Template: json.RawMessage(`{}`), Environment: json.RawMessage(`{`)},
	} {
		if doc, err := api.ValidateTemplate(ctx, opts); err == nil || doc != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: %v %v", name, doc, err)
		}
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := api.ValidateTemplate(canceled, stacktemplates.ValidateTemplateOpts{Template: json.RawMessage(`{}`)}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("invalid calls reached HTTP: %d", calls.Load())
	}
	if _, err := api.ValidateTemplate(ctx, stacktemplates.ValidateTemplateOpts{Template: json.RawMessage(`{}`)}); !gophercloud.ResponseCodeIs(err, 400) || calls.Load() != 1 {
		t.Fatalf("validation failure must stay a native 400 after one attempt: %v calls=%d", err, calls.Load())
	}
	status.Store(202)
	if _, err := api.ValidateTemplate(ctx, stacktemplates.ValidateTemplateOpts{Template: json.RawMessage(`{}`)}); !gophercloud.ResponseCodeIs(err, 202) {
		t.Fatalf("only HTTP 200 succeeds: %v", err)
	}
}
