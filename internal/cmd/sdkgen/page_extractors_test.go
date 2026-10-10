package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeclaredPageExtractorUsesReturnedPageInsteadOfNamesOrNestedFields(t *testing.T) {
	local, _ := fixture(t, `package fixture
type Page interface{}
type User struct{}
type UserPage struct{}
type WrappedPage struct{}
func ExtractUsers(Page)([]User,error){return nil,nil}
func ExtractProjects(Page)([]User,error){return nil,nil}
func ExtractWrapped(Page)([]User,error){return nil,nil}
`)
	foreign, _ := fixture(t, `package fixture
type Page interface{}
type Project struct{}
func ExtractProjects(Page)([]Project,error){return nil,nil}
`)
	for _, tc := range []struct {
		name, body, extractor string
		owner                 *types.Package
		declared              bool
	}{
		{"foreign beats sole user and operation-name extractors", "return other.ProjectPage{}", "ExtractProjects", foreign, true},
		{"local wrapper owns nested foreign page", "return WrappedPage{other.ProjectPage{}}", "ExtractWrapped", local, true},
		{"local user page stays local", "return UserPage{}", "ExtractUsers", local, true},
		{"unknown foreign page stays raw", "return other.UnknownPage{}", "", nil, true},
		{"conflicting return pages stay raw", "if true { return UserPage{} }; return other.ProjectPage{}", "", nil, true},
		{"delegating declaration permits fallback", "return delegated()", "", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "operation.go", "package fixture; func ListProjects() Page { "+tc.body+" }", 0)
			if err != nil {
				t.Fatal(err)
			}
			declaration := file.Decls[0].(*ast.FuncDecl)
			fn, declared := declaredPageExtractor(local, map[string]*types.Package{"other": foreign}, declaration, map[string]string{"UserPage": "ExtractUsers", "WrappedPage": "ExtractWrapped"})
			if declared != tc.declared {
				t.Fatalf("declared=%v", declared)
			}
			if tc.extractor == "" {
				if fn != nil {
					t.Fatalf("unexpected extractor %s", fn)
				}
			} else if fn == nil || fn.Name() != tc.extractor || fn.Pkg() != tc.owner {
				t.Fatalf("extractor=%v want %s from %v", fn, tc.extractor, tc.owner)
			}
		})
	}
}

func TestPinnedForeignPagesAndLocalWrapperGenerateCompatibleIterators(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	for _, tc := range []struct {
		path string
		want []string
	}{
		{"identity/v3/users", []string{
			"ListProjects(ctx context.Context, userID string) iter.Seq2[*projects.Project, error]",
			"ListGroups(ctx context.Context, userID string) iter.Seq2[*groups.Group, error]",
			"values, err := projects.ExtractProjects(page)", "values, err := groups.ExtractGroups(page)",
			"ListInGroup(ctx context.Context, groupID string, options ...ListInGroupOption) iter.Seq2[*User, error]",
			"values, err := upstream.ExtractUsers(page)",
		}},
		{"db/v1/configurations", []string{"iter.Seq2[*instances.Instance, error]", "values, err := instances.ExtractInstances(page)"}},
		{"identity/v2/extensions", []string{"values, err := upstream.ExtractExtensions(page)"}},
		{"compute/v2/availabilityzones", []string{"return resource.SinglePageStream(ctx, upstream.List(a.client)", "return resource.SinglePageStream(ctx, upstream.ListDetail(a.client)", "values, err := upstream.ExtractAvailabilityZones(page)"}},
		{"blockstorage/v2/availabilityzones", []string{"return resource.SinglePageStream(ctx, upstream.List(a.client)"}},
		{"blockstorage/v3/availabilityzones", []string{"return resource.SinglePageStream(ctx, upstream.List(a.client)"}},
		{"sharedfilesystems/v2/availabilityzones", []string{"return resource.SinglePageStream(ctx, upstream.List(a.client)"}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			if err := g.generate(upstreamModule + "/openstack/" + tc.path); err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(filepath.Join(g.root, tc.path, "api_generated.go"))
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range tc.want {
				if !strings.Contains(string(body), want) {
					t.Fatalf("missing %q in %s", want, tc.path)
				}
			}
			if tc.path == "identity/v2/extensions" && strings.Contains(string(body), "values, err := extensions.ExtractExtensions(page)") {
				t.Fatal("nested common page replaced the local extension wrapper")
			}
		})
	}
}

func TestTokenScopeBuildersDelegateWithoutExtensionFields(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	for _, path := range []string{"identity/v3/tokens", "identity/v3/ec2tokens", "identity/v3/oauth1"} {
		t.Run(path, func(t *testing.T) {
			if err := g.generate(upstreamModule + "/openstack/" + path); err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(filepath.Join(g.root, path, "api_generated.go"))
			if err != nil {
				t.Fatal(err)
			}
			source := string(body)
			if !strings.Contains(source, "ToTokenV3ScopeMap() (map[string]any, error) {\n\treturn b.base.ToTokenV3ScopeMap()\n}") {
				t.Fatal("scope builder is not a plain delegate")
			}
			if !strings.Contains(source, "value0, err = request.MergeFieldsFor(value0, b.config.Fields, b.base)") {
				t.Fatal("create builder lost its extension field merge")
			}
		})
	}
}

func TestPageExtractorPrefersUnsuffixedOverMicroversionSuffix(t *testing.T) {
	pkg, decls := fixture(t, `package fixture
type Page interface{}
type Candidates struct{ Requests map[string]int }
type Candidates110 struct{ Requests []int }
type CapsuleBase struct{ Name string }
type CapsuleV132 struct{ Name string; Host string }
type CandidatesPage struct{}
type CapsulePage struct{}
func ExtractCandidates(r Page)(*Candidates,error){_ = r.(CandidatesPage);return nil,nil}
func ExtractCandidates110(r Page)(*Candidates110,error){_ = r.(CandidatesPage);return nil,nil}
func ExtractCapsulesBase(r Page)([]CapsuleBase,error){_ = r.(CapsulePage);return nil,nil}
func ExtractCapsulesV132(r Page)([]CapsuleV132,error){_ = r.(CapsulePage);return nil,nil}
`)
	got := extractorsByPage(pkg, decls)
	// A pure digit suffix names an older response shape; distinct names keep the detail rule.
	if got["CandidatesPage"] != "ExtractCandidates" || got["CapsulePage"] != "ExtractCapsulesV132" {
		t.Fatal(got)
	}
	for _, tc := range []struct {
		name, base string
		want       bool
	}{{"ExtractCandidates110", "ExtractCandidates", true}, {"ExtractCandidates", "ExtractCandidates", false}, {"ExtractCapsulesV132", "ExtractCapsules", false}} {
		if microversionSuffix(tc.name, tc.base) != tc.want {
			t.Fatal(tc)
		}
	}
}
