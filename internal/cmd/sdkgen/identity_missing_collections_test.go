package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// These declarations retain the pinned native root extractor, GET codes, and
// ImagePage continuation implementation. They are parsed, never executed.
const pinnedImageIdentitySource = `package fixture
func List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(c)
	if opts != nil {
		query, err := opts.ToImageListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}
	return pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {
		imagePage := ImagePage{
			serviceURL:     c.ServiceURL(),
			LinkedPageBase: pagination.LinkedPageBase{PageResult: r},
		}

		return imagePage
	})
}
func Get(ctx context.Context, client *gophercloud.ServiceClient, id string) (r GetResult) {
	resp, err := client.Get(ctx, getURL(client, id), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}
func listURL(c *gophercloud.ServiceClient) string {
	return c.ServiceURL("images")
}
func imageURL(c *gophercloud.ServiceClient, imageID string) string {
	return c.ServiceURL("images", imageID)
}
func getURL(c *gophercloud.ServiceClient, imageID string) string {
	return imageURL(c, imageID)
}
func nextPageURL(serviceURL, requestedNext string) (string, error) {
	base, err := utils.BaseEndpoint(serviceURL)
	if err != nil {
		return "", err
	}

	requestedNextURL, err := url.Parse(requestedNext)
	if err != nil {
		return "", err
	}

	base = gophercloud.NormalizeURL(base)
	nextPath := base + strings.TrimPrefix(requestedNextURL.Path, "/")

	nextURL, err := url.Parse(nextPath)
	if err != nil {
		return "", err
	}

	nextURL.RawQuery = requestedNextURL.RawQuery

	return nextURL.String(), nil
}
func (r commonResult) Extract() (*Image, error) {
	var s *Image
	if v, ok := r.Body.(map[string]any); ok {
		for k, h := range r.Header {
			if strings.ToLower(k) == "openstack-image-import-methods" {
				for _, s := range h {
					v["openstack-image-import-methods"] = s
				}
			}
			if strings.ToLower(k) == "openstack-image-store-ids" {
				for _, s := range h {
					v["openstack-image-store-ids"] = s
				}
			}
		}
	}
	err := r.ExtractInto(&s)
	return s, err
}
func (r ImagePage) IsEmpty() (bool, error) {
	if r.StatusCode == 204 {
		return true, nil
	}

	images, err := ExtractImages(r)
	return len(images) == 0, err
}
func (r ImagePage) NextPageURL() (string, error) {
	var s struct {
		Next string ` + "`" + `json:"next"` + "`" + `
	}
	err := r.ExtractInto(&s)
	if err != nil {
		return "", err
	}

	if s.Next == "" {
		return "", nil
	}

	return nextPageURL(r.serviceURL, s.Next)
}
func ExtractImages(r pagination.Page) ([]Image, error) {
	var s struct {
		Images []Image ` + "`" + `json:"images"` + "`" + `
	}
	err := (r.(ImagePage)).ExtractInto(&s)
	return s.Images, err
}`

func pinnedImageIdentityDeclarations(t *testing.T) map[string]*ast.FuncDecl {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "native-image.go", pinnedImageIdentitySource, 0)
	if err != nil {
		t.Fatal(err)
	}
	declarations := map[string]*ast.FuncDecl{}
	for _, declaration := range file.Decls {
		if fn, ok := declaration.(*ast.FuncDecl); ok {
			declarations[identityDeclarationKey(fn)] = fn
		}
	}
	return declarations
}

func imageIdentitySpec(t *testing.T) identityCollectionSpec {
	t.Helper()
	for _, spec := range identityCollectionSpecs {
		if spec.path == "image/v2/images" {
			return spec
		}
	}
	t.Fatal("Image identity contract is missing")
	return identityCollectionSpec{}
}

func TestIdentityMissingListPolicyOnlyEnablesAuditedImageMetadata(t *testing.T) {
	image := imageIdentitySpec(t)
	if !identityMissingListMetadataValid(image) {
		t.Fatal("audited Image metadata rejected")
	}
	mutations := map[string]func(*identityCollectionSpec){
		"model":                func(s *identityCollectionSpec) { s.model = "Task" },
		"GET method":           func(s *identityCollectionSpec) { s.getter = "Fetch" },
		"LIST method":          func(s *identityCollectionSpec) { s.lister = "ListSimple" },
		"scoped route":         func(s *identityCollectionSpec) { s.parents = 1 },
		"collection route":     func(s *identityCollectionSpec) { s.getSegments = []string{"tasks", "$id"} },
		"literal ID route":     func(s *identityCollectionSpec) { s.getSegments = []string{"images", "fixed"} },
		"extra route":          func(s *identityCollectionSpec) { s.getSegments = []string{"images", "$id", "file"} },
		"accepted code":        func(s *identityCollectionSpec) { s.getCodes = []int{203} },
		"extra code":           func(s *identityCollectionSpec) { s.getCodes = []int{200, 203} },
		"Python alias on wire": func(s *identityCollectionSpec) { s.missingListKey = "is_hidden" },
		"missing key":          func(s *identityCollectionSpec) { s.missingListKey = "" },
		"false value":          func(s *identityCollectionSpec) { s.missingListValue = "false" },
		"empty value":          func(s *identityCollectionSpec) { s.missingListValue = "" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			altered := image
			mutate(&altered)
			if identityMissingListMetadataValid(altered) {
				t.Fatal("unsafe missing-list metadata accepted", altered)
			}
		})
	}
	for _, spec := range identityCollectionSpecs {
		if spec.path == image.path {
			continue
		}
		if !identityMissingListMetadataValid(spec) {
			t.Fatal("existing audited metadata rejected", spec)
		}
		spec.missingListKey = "os_hidden"
		spec.missingListValue = "true"
		if identityMissingListMetadataValid(spec) {
			t.Fatal("missing-list capability leaked to unrelated binding", spec)
		}
	}
}

func TestIdentityMissingListImageSchemaAndExtractorDriftFailGeneration(t *testing.T) {
	spec := imageIdentitySpec(t)
	base := identityQueryFixtureSource(spec)
	pkg, plan := identityQueryFixture(t, spec, base)
	if !identityImageListSchema(pkg, plan) {
		t.Fatal("pinned native Image schema rejected")
	}
	if err := validateIdentityCollectionContracts(pkg, pinnedImageIdentityDeclarations(t), plan, nil, nil); err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(string) string{
		"query hidden pointer": func(s string) string { return strings.Replace(s, "Hidden bool `q:", "Hidden *bool `q:", 1) },
		"query hidden key":     func(s string) string { return strings.Replace(s, "q:\"os_hidden\"", "q:\"is_hidden\"", 1) },
		"query hidden name":    func(s string) string { return strings.Replace(s, "Hidden bool `q:", "IsHidden bool `q:", 1) },
		"response hidden type": func(s string) string { return strings.Replace(s, "Hidden bool `json:", "Hidden string `json:", 1) },
		"response hidden key":  func(s string) string { return strings.Replace(s, "json:\"os_hidden\"", "json:\"hidden\"", 1) },
		"response hidden name": func(s string) string { return strings.Replace(s, "Hidden bool `json:", "IsHidden bool `json:", 1) },
		"extra page state": func(s string) string {
			return strings.Replace(s, "serviceURL string;pagination.LinkedPageBase", "serviceURL string;extra bool;pagination.LinkedPageBase", 1)
		},
		"missing native service URL": func(s string) string { return strings.Replace(s, "serviceURL string;", "", 1) },
		"wrong native service URL":   func(s string) string { return strings.Replace(s, "serviceURL string;", "serviceURL int;", 1) },
		"nonembedded native page": func(s string) string {
			return strings.Replace(s, ";pagination.LinkedPageBase", ";page pagination.LinkedPageBase", 1)
		},
		"wrong page empty result": func(s string) string {
			return strings.Replace(s, "IsEmpty()(bool,error){return false,nil}", "IsEmpty()(int,error){return 0,nil}", 1)
		},
		"extra page continuation argument": func(s string) string { return strings.Replace(s, "NextPageURL()", "NextPageURL(bool)", 1) },
		"wrong page continuation result": func(s string) string {
			return strings.Replace(s, "NextPageURL()(string,error){return \"\",nil}", "NextPageURL()(int,error){return 0,nil}", 1)
		},
		"wrong extractor model":   func(s string) string { return strings.Replace(s, "([]Image,error)", "([]string,error)", 1) },
		"missing named extractor": func(s string) string { return strings.Replace(s, "func ExtractImages(", "func ExtractOther(", 1) },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			source := mutate(base)
			if source == base {
				t.Fatal("schema mutation did not apply")
			}
			badPkg, badPlan := identityQueryFixture(t, spec, source)
			if err := validateIdentityCollectionContracts(badPkg, pinnedImageIdentityDeclarations(t), badPlan, nil, nil); err == nil || !strings.Contains(err.Error(), "schema changed") {
				t.Fatal("native schema drift ignored", err)
			}
		})
	}
}

func TestIdentityMissingListEmitsFixedOwnedOverlayAndExplicitInventory(t *testing.T) {
	missingCount, modesCount := 0, 0
	for _, spec := range identityCollectionSpecs {
		t.Run(spec.path, func(t *testing.T) {
			pkg, plan := identityCollectionFixture(t, spec, "ID string;Name string", "string", identityExpectedNameQuery(spec))
			for _, mode := range identityListModeSpecs {
				if mode.path == spec.path {
					pkg, plan = identityModeFixture(t, mode, identityModeFixtureSource(mode))
				}
			}
			e := emitter{pkg: pkg, imports: map[string]string{}}
			parents := []string(nil)
			if spec.parents != 0 {
				parents = []string{"a.parentID"}
			}
			e.printf("func(a *API)newResources()*resource.Collection[%s]{return ", spec.model)
			emitCollectionAdapter(&e, plan, "a", parents)
			e.printf("}\n")
			emitted, err := e.source()
			if err != nil {
				t.Fatal(err)
			}
			expected := spec.path == "image/v2/images"
			enabled := identityMissingListEnabled(pkg, plan, spec.parents)
			if enabled != expected {
				t.Fatal("missing-list capability is not Image-only", spec, enabled)
			}
			body := string(emitted)
			if expected {
				missingCount++
				for _, want := range []string{`IdentityMissingListQuery: url.Values{"os_hidden": {"true"}}`, `[]string{"images", id}, q, []int{200}`, "return result.Extract()", "a.Get(ctx, string(id))", "config.Query[key] = append([]string(nil), values...)", "a.listWithControl(ctx, control, options...)"} {
					if !strings.Contains(body, want) {
						t.Fatal("missing Image native contract", want, body)
					}
				}
				if strings.Contains(body, "is_hidden") || strings.Contains(body, "WithListQuery(key, value)") || strings.Contains(body, "IdentityAllProjectsQuery:") || strings.Contains(body, "IterateIdentity:") {
					t.Fatal("Image gained alias rewrite or unsupported list modes", body)
				}
			} else if strings.Contains(body, "IdentityMissingListQuery:") {
				t.Fatal("hidden retry leaked to other binding", body)
			}
			if strings.Contains(body, "IdentityAllProjectsQuery:") {
				modesCount++
			}
			record := collectionRecord{Package: "gophercloudsdk/" + spec.path, Model: spec.model, IdentityFind: true, IdentityGetQuery: true, IdentityMissingList: enabled}
			data, err := json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), `"identity_missing_list":true`) != expected {
				t.Fatal("inventory disagrees with emitted capability", string(data))
			}
		})
	}
	if missingCount != 1 || modesCount != 2 {
		t.Fatal("audited missing/modes capability totals changed", missingCount, modesCount)
	}
	data, err := json.Marshal(collectionRecord{})
	if err != nil || strings.Contains(string(data), "identity_missing_list") {
		t.Fatal("default inventory claims hidden search", string(data), err)
	}
}
