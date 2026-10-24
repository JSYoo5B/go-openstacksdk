package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/templateloader"
)

func r2ManilaHeatWrite(t *testing.T, dir, name, content string) string {
	t.Helper()
	full := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return full
}

func r2ManilaHeatLoader(t *testing.T, options ...templateloader.Option) *templateloader.Loader {
	t.Helper()
	loader, err := templateloader.New(options...)
	if err != nil {
		t.Fatal(err)
	}
	return loader
}

func r2ManilaHeatFileURL(path string) string { return "file://" + filepath.ToSlash(path) }

func TestR2ManilaHeatTemplateResolvesGetFileAndTypeRecursivelyWithSharedFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	root := r2ManilaHeatWrite(t, dir, "stack/main.yaml", `heat_template_version: 2018-08-31
resources:
  a:
    type: nested/child.yaml
  b:
    type: OS::Heat::Stack
    properties:
      script: {get_file: scripts/init.sh}
      same: {get_file: ./scripts/init.sh}
      parent: {get_file: ../shared/blob.txt}
      list:
        - {get_file: scripts/init.sh}
  c:
    type: nested/child.template
`)
	r2ManilaHeatWrite(t, dir, "stack/nested/child.yaml", "heat_template_version: 2018-08-31\nresources:\n  x:\n    type: OS::Nova::Server\n    properties:\n      user_data: {get_file: ../scripts/init.sh}\n")
	r2ManilaHeatWrite(t, dir, "stack/nested/child.template", "heat_template_version: 2018-08-31\ndescription: a\u00e9 <b>\n")
	r2ManilaHeatWrite(t, dir, "stack/scripts/init.sh", "#!/bin/sh\necho hi\n")
	r2ManilaHeatWrite(t, dir, "shared/blob.txt", "")
	loader := r2ManilaHeatLoader(t)
	input := map[string]string{"preset": "kept"}
	result, err := loader.GetTemplateContents(context.Background(), templateloader.Source{File: root}, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(input) != 1 {
		t.Fatalf("input files mutated: %v", input)
	}
	child := r2ManilaHeatFileURL(filepath.Join(dir, "stack/nested/child.yaml"))
	script := r2ManilaHeatFileURL(filepath.Join(dir, "stack/scripts/init.sh"))
	blob := r2ManilaHeatFileURL(filepath.Join(dir, "shared/blob.txt"))
	childTemplate := r2ManilaHeatFileURL(filepath.Join(dir, "stack/nested/child.template"))
	if len(result.Files) != 5 || result.Files["preset"] != "kept" || result.Files[script] != "#!/bin/sh\necho hi\n" || result.Files[blob] != "" {
		t.Fatalf("files=%v", result.Files)
	}
	var nested map[string]any
	if err := json.Unmarshal([]byte(result.Files[child]), &nested); err != nil {
		t.Fatal(err)
	}
	userData := nested["resources"].(map[string]any)["x"].(map[string]any)["properties"].(map[string]any)["user_data"].(map[string]any)
	if userData["get_file"] != script || !strings.Contains(result.Files[childTemplate], `"description":"aé <b>"`) {
		t.Fatalf("nested=%v template=%q", nested, result.Files[childTemplate])
	}
	resources := result.Content["resources"].(map[string]any)
	properties := resources["b"].(map[string]any)["properties"].(map[string]any)
	if resources["a"].(map[string]any)["type"] != child || resources["c"].(map[string]any)["type"] != childTemplate || resources["b"].(map[string]any)["type"] != "OS::Heat::Stack" ||
		properties["script"].(map[string]any)["get_file"] != script || properties["same"].(map[string]any)["get_file"] != script ||
		properties["parent"].(map[string]any)["get_file"] != blob || properties["list"].([]any)[0].(map[string]any)["get_file"] != script {
		t.Fatalf("content=%v", result.Content)
	}
}

func TestR2ManilaHeatTemplateBinaryFilesUseEncodeBytesAndYAMLKeepsTimestampText(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	binary := make([]byte, 100)
	for i := range binary {
		binary[i] = byte(0x80 + i%64)
	}
	if err := os.WriteFile(filepath.Join(dir, "blob.bin"), binary, 0o644); err != nil {
		t.Fatal(err)
	}
	root := r2ManilaHeatWrite(t, dir, "t.yaml", "heat_template_version: 2013-05-23\nparameters:\n  when:\n    default: 2013-05-23\n  n:\n    default: 010\n  flag:\n    default: yes\n  1: numeric-key\nresources:\n  r:\n    properties: {get_file: blob.bin}\n")
	result, err := r2ManilaHeatLoader(t).GetTemplateContents(context.Background(), templateloader.Source{File: root}, nil)
	if err != nil {
		t.Fatal(err)
	}
	blob := r2ManilaHeatFileURL(filepath.Join(dir, "blob.bin"))
	content := result.Files[blob]
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	if len(lines) != 2 || len(lines[0]) != 76 || !strings.HasSuffix(content, "\n") {
		t.Fatalf("content=%q", content)
	}
	parameters := result.Content["parameters"].(map[string]any)
	if result.Content["heat_template_version"] != "2013-05-23" || parameters["when"].(map[string]any)["default"] != "2013-05-23" || parameters["flag"].(map[string]any)["default"] != true || parameters["1"] != "numeric-key" {
		t.Fatalf("content=%v", result.Content)
	}
}

func TestR2ManilaHeatTemplateJSONKeepsNumbersAndRejectsInvalidTemplates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	loader := r2ManilaHeatLoader(t)
	ctx := context.Background()
	good := r2ManilaHeatWrite(t, dir, "ok.json", ` {"heat_template_version":"2018-08-31","big":9007199254740993}`)
	result, err := loader.GetTemplateContents(ctx, templateloader.Source{File: good}, nil)
	if err != nil || result.Content["big"].(json.Number).String() != "9007199254740993" {
		t.Fatal(result, err)
	}
	invalid := map[string]string{
		"noversion.yaml": "resources: {}\n", "empty.yaml": "", "list.yaml": "- a\n", "bad.yaml": "a: [1\n", "badjson.json": `{"heat_template_version": `,
		"binary.yaml": "\xff\xfe\x00bad", "extra.json": `{"heat_template_version":"x"} trailing`,
	}
	for name, content := range invalid {
		path := r2ManilaHeatWrite(t, dir, name, content)
		_, err := loader.GetTemplateContents(ctx, templateloader.Source{File: path}, nil)
		if !errors.Is(err, templateloader.ErrTemplate) || !strings.HasPrefix(err.Error(), "Error in processing template files: ") {
			t.Fatalf("%s: %v", name, err)
		}
	}
	for _, source := range []templateloader.Source{{}, {File: good, URL: "http://x/y"}, {Object: "swift://c/o"}, {File: filepath.Join(dir, "missing.yaml")}, {URL: "ftp://example.test/t.yaml"}} {
		if _, err := loader.GetTemplateContents(ctx, source, nil); !errors.Is(err, templateloader.ErrTemplate) {
			t.Fatalf("%+v: %v", source, err)
		}
	}
}

func TestR2ManilaHeatTemplateObjectReaderAndMissingReaderBehavior(t *testing.T) {
	t.Parallel()
	objects := map[string]string{
		"https://objects.test/c/main.yaml": "heat_template_version: 2018-08-31\nresources:\n  r:\n    type: sub.yaml\n    properties: {get_file: data.txt}\n",
		"https://objects.test/c/sub.yaml":  "heat_template_version: 2018-08-31\n",
		"https://objects.test/c/data.txt":  "payload",
	}
	var reads []string
	reader := func(ctx context.Context, url string) ([]byte, error) {
		reads = append(reads, url)
		return []byte(objects[url]), nil
	}
	result, err := r2ManilaHeatLoader(t, templateloader.WithObjectReader(reader)).GetTemplateContents(context.Background(), templateloader.Source{Object: "https://objects.test/c/main.yaml"}, nil)
	if err != nil || result.Files["https://objects.test/c/data.txt"] != "payload" || !strings.Contains(result.Files["https://objects.test/c/sub.yaml"], "heat_template_version") || len(reads) != 4 {
		t.Fatalf("result=%+v err=%v reads=%v", result, err, reads)
	}
	// Python passes no object_request from the proxy, so an object cannot be fetched.
	_, err = r2ManilaHeatLoader(t).GetTemplateContents(context.Background(), templateloader.Source{Object: "https://objects.test/c/main.yaml"}, nil)
	if err == nil || !strings.Contains(err.Error(), "Could not fetch template from https://objects.test/c/main.yaml") {
		t.Fatal(err)
	}
}

func TestR2ManilaHeatTemplateHTTPRelativeFilesErrorsCancellationAndLimits(t *testing.T) {
	t.Parallel()
	var paths []string
	server := testcloud.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.Header.Get("X-Auth-Token") != "" {
			t.Error("token leaked to template server")
		}
		switch r.URL.Path {
		case "/t/main.yaml":
			_, _ = w.Write([]byte("heat_template_version: 2018-08-31\nresources:\n  r:\n    properties:\n      a: {get_file: files/a.txt}\n      b: {get_file: /abs/b.txt}\n"))
		case "/t/files/a.txt":
			_, _ = w.Write([]byte("A"))
		case "/abs/b.txt":
			http.Error(w, "gone", 404)
		case "/t/loop.yaml":
			_, _ = w.Write([]byte("heat_template_version: 2018-08-31\nresources:\n  r: {type: loop.yaml}\n"))
		case "/t/big.yaml":
			_, _ = w.Write([]byte("heat_template_version: 2018-08-31\n# " + strings.Repeat("x", 200) + "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	ctx := context.Background()
	loader := r2ManilaHeatLoader(t, templateloader.WithHTTPClient(server.Client()))
	_, err := loader.GetTemplateContents(ctx, templateloader.Source{URL: server.URL + "/t/main.yaml"}, nil)
	if err == nil || !strings.Contains(err.Error(), "Could not fetch contents for "+server.URL+"/abs/b.txt") || !errors.Is(err, templateloader.ErrTemplate) {
		t.Fatalf("err=%v paths=%v", err, paths)
	}
	if _, err := loader.GetTemplateContents(ctx, templateloader.Source{URL: server.URL + "/t/missing.yaml"}, nil); err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Fatal(err)
	}
	if _, err := loader.GetTemplateContents(ctx, templateloader.Source{URL: server.URL + "/t/loop.yaml"}, nil); err == nil || !strings.Contains(err.Error(), "references itself") {
		t.Fatal(err)
	}
	limited := r2ManilaHeatLoader(t, templateloader.WithHTTPClient(server.Client()), templateloader.WithMaxFileBytes(100))
	if _, err := limited.GetTemplateContents(ctx, templateloader.Source{URL: server.URL + "/t/big.yaml"}, nil); err == nil || !strings.Contains(err.Error(), "byte limit") {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	before := len(paths)
	if _, err := loader.GetTemplateContents(cancelled, templateloader.Source{URL: server.URL + "/t/main.yaml"}, nil); !errors.Is(err, context.Canceled) || len(paths) != before {
		t.Fatalf("err=%v", err)
	}
	for _, option := range []templateloader.Option{templateloader.WithHTTPClient(nil), templateloader.WithObjectReader(nil), templateloader.WithMaxFileBytes(-1), nil} {
		if _, err := templateloader.New(option); !errors.Is(err, templateloader.ErrTemplate) {
			t.Fatal(err)
		}
	}
}

func TestR2ManilaHeatEnvironmentsMergeDeeplyAndResolveRegistryFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r2ManilaHeatWrite(t, dir, "env/res/server.yaml", "heat_template_version: 2018-08-31\nresources:\n  s: {type: OS::Nova::Server, properties: {user_data: {get_file: ../data.txt}}}\n")
	r2ManilaHeatWrite(t, dir, "env/data.txt", "data")
	r2ManilaHeatWrite(t, dir, "other/custom.yaml", "heat_template_version: 2018-08-31\n")
	first := r2ManilaHeatWrite(t, dir, "env/one.yaml", `parameter_defaults:
  keep: 1
  nested: {a: 1, b: {c: 2}}
  list: [1, 2]
resource_registry:
  "OS::Custom::Server": res/server.yaml
  "OS::Builtin::Thing": "OS::Heat::None"
  hooks: pre-create
  restricted_actions: update
  resources:
    web:
      hooks: pre-update
      "OS::Web::Extra": ../other/custom.yaml
`)
	second := r2ManilaHeatWrite(t, dir, "env/two.yaml", "parameter_defaults:\n  nested: {b: {d: 3}, e: 4}\n  list: [9]\nparameters: {p: null}\n")
	files, env, err := r2ManilaHeatLoader(t).ProcessEnvironments(context.Background(), []string{first, second})
	if err != nil {
		t.Fatal(err)
	}
	defaults := env["parameter_defaults"].(map[string]any)
	nested := defaults["nested"].(map[string]any)
	if defaults["keep"] != 1 || nested["a"] != 1 || nested["e"] != 4 || !reflect.DeepEqual(nested["b"], map[string]any{"c": 2, "d": 3}) || !reflect.DeepEqual(defaults["list"], []any{9}) {
		t.Fatalf("env=%v", env)
	}
	if v, ok := env["parameters"].(map[string]any)["p"]; !ok || v != nil {
		t.Fatalf("env=%v", env)
	}
	registry := env["resource_registry"].(map[string]any)
	server := r2ManilaHeatFileURL(filepath.Join(dir, "env/res/server.yaml"))
	custom := r2ManilaHeatFileURL(filepath.Join(dir, "other/custom.yaml"))
	if registry["OS::Custom::Server"] != server || registry["OS::Builtin::Thing"] != "OS::Heat::None" || registry["hooks"] != "pre-create" ||
		registry["resources"].(map[string]any)["web"].(map[string]any)["OS::Web::Extra"] != custom || registry["resources"].(map[string]any)["web"].(map[string]any)["hooks"] != "pre-update" {
		t.Fatalf("registry=%v", registry)
	}
	data := r2ManilaHeatFileURL(filepath.Join(dir, "env/data.txt"))
	if len(files) != 3 || files[data] != "data" || !strings.Contains(files[custom], "heat_template_version") || !strings.Contains(files[server], data) {
		t.Fatalf("files=%v", files)
	}
}

func TestR2ManilaHeatEnvironmentRejectsInvalidShapes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	loader := r2ManilaHeatLoader(t)
	ctx := context.Background()
	for name, content := range map[string]string{
		"section.yaml": "unknown: {}\n", "list.yaml": "- a\n", "bad.yaml": "a: [\n", "registry.yaml": "resource_registry: [a]\n",
		"res.yaml": "resource_registry: {resources: {r: 1}}\n", "base.yaml": "resource_registry: {base_url: 1}\n", "value.yaml": "resource_registry: {\"OS::X\": 1}\n",
		"notfound.yaml": "resource_registry: {\"OS::X\": missing.yaml}\n",
	} {
		path := r2ManilaHeatWrite(t, dir, name, content)
		if _, _, err := loader.ProcessEnvironments(ctx, []string{path}); !errors.Is(err, templateloader.ErrTemplate) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	empty := r2ManilaHeatWrite(t, dir, "empty.yaml", "")
	nullRegistry := r2ManilaHeatWrite(t, dir, "null.yaml", "resource_registry:\n")
	files, env, err := loader.ProcessEnvironments(ctx, []string{empty, nullRegistry})
	if err != nil || len(files) != 0 || env["resource_registry"] != nil {
		t.Fatal(files, env, err)
	}
	if _, ok := env["resource_registry"]; !ok {
		t.Fatalf("env=%v", env)
	}
}

func TestR2ManilaHeatReadEnvAndTemplatesAttributePresenceAndPrecedence(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	tpl := r2ManilaHeatWrite(t, dir, "t.yaml", "heat_template_version: 2018-08-31\nresources:\n  r: {type: x.yaml}\n")
	r2ManilaHeatWrite(t, dir, "x.yaml", "heat_template_version: 2018-08-31\n")
	xURL := r2ManilaHeatFileURL(filepath.Join(dir, "x.yaml"))
	env := r2ManilaHeatWrite(t, dir, "e.yaml", "parameter_defaults: {k: v}\nresource_registry: {\"OS::Z\": x.yaml}\n")
	plain := r2ManilaHeatWrite(t, dir, "plain.yaml", "heat_template_version: 2018-08-31\n")
	loader := r2ManilaHeatLoader(t)
	ctx := context.Background()

	empty, err := loader.ReadEnvAndTemplates(ctx, templateloader.StackInput{})
	if err != nil || empty.Environment != nil || empty.Template != nil || empty.Files != nil {
		t.Fatal(empty, err)
	}
	// Environment files alone: Python drops registry files without a template.
	envOnly, err := loader.ReadEnvAndTemplates(ctx, templateloader.StackInput{EnvironmentFiles: []string{env}, Files: map[string]string{"ignored": "1"}})
	if err != nil || envOnly.Environment == nil || envOnly.Template != nil || envOnly.Files != nil {
		t.Fatal(envOnly, err)
	}
	// A template without files yields no Files attribute; a user file keeps it present.
	noFiles, err := loader.ReadEnvAndTemplates(ctx, templateloader.StackInput{Template: templateloader.Source{File: plain}})
	if err != nil || noFiles.Template == nil || noFiles.Files != nil || noFiles.Environment != nil {
		t.Fatal(noFiles, err)
	}
	user, err := loader.ReadEnvAndTemplates(ctx, templateloader.StackInput{Template: templateloader.Source{File: plain}, Files: map[string]string{"k": "v"}})
	if err != nil || !reflect.DeepEqual(user.Files, map[string]string{"k": "v"}) {
		t.Fatal(user, err)
	}
	// An empty (non-nil) environment file result is still present.
	emptyEnv := r2ManilaHeatWrite(t, dir, "empty-env.yaml", "")
	present, err := loader.ReadEnvAndTemplates(ctx, templateloader.StackInput{EnvironmentFiles: []string{emptyEnv}})
	if err != nil || present.Environment == nil || len(present.Environment) != 0 {
		t.Fatal(present, err)
	}
	// Environment-derived content wins a URL collision with the template files.
	both, err := loader.ReadEnvAndTemplates(ctx, templateloader.StackInput{Template: templateloader.Source{File: tpl}, EnvironmentFiles: []string{env}})
	if err != nil || both.Environment == nil || both.Template == nil || len(both.Files) != 1 {
		t.Fatal(both, err)
	}
	if both.Files[xURL] != "{\"heat_template_version\":\"2018-08-31\"}" {
		t.Fatalf("files=%v", both.Files)
	}
	if _, err := loader.ReadEnvAndTemplates(ctx, templateloader.StackInput{EnvironmentFiles: []string{filepath.Join(dir, "absent.yaml")}}); !errors.Is(err, templateloader.ErrTemplate) {
		t.Fatal(err)
	}
	// Failures from ReadEnvAndTemplates are not wrapped in the proxy's template message.
	if _, err := loader.ReadEnvAndTemplates(ctx, templateloader.StackInput{Template: templateloader.Source{File: filepath.Join(dir, "absent.yaml")}}); err == nil || strings.HasPrefix(err.Error(), "Error in processing template files") {
		t.Fatal(err)
	}
}
