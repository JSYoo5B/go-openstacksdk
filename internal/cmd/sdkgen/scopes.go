package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Parent relationships and nonstandard endpoint identifiers belong to the SDK,
// rather than requiring applications to supply builders or resolver interfaces.
type scopeSpec struct {
	path, method, parent, get, list, delete, id string
}

var scopeSpecs = []scopeSpec{
	{"compute/v2/attachinterfaces", "InServer", "compute/v2/servers", "Get", "List", "Delete", "PortID"},
	{"compute/v2/volumeattach", "InServer", "compute/v2/servers", "Get", "List", "Delete", "VolumeID"},
	{"containerinfra/v1/nodegroups", "InCluster", "containerinfra/v1/clusters", "Get", "List", "Delete", "UUID"},
	{"dns/v2/recordsets", "InZone", "dns/v2/zones", "Get", "ListByZone", "Delete", "ID"},
	{"identity/v3/applicationcredentials", "InUser", "identity/v3/users", "Get", "List", "Delete", "ID"},
	{"identity/v3/applicationcredentials", "AccessRules", "identity/v3/users", "GetAccessRule", "ListAccessRules", "DeleteAccessRule", "ID"},
	{"identity/v3/ec2credentials", "InUser", "identity/v3/users", "Get", "List", "Delete", "Access"},
	{"image/v2/members", "InImage", "image/v2/images", "Get", "List", "Delete", "MemberID"},
	{"loadbalancer/v2/l7policies", "Rules", "loadbalancer/v2/l7policies", "GetRule", "ListRules", "DeleteRule", "ID"},
	{"loadbalancer/v2/pools", "Members", "loadbalancer/v2/pools", "GetMember", "ListMembers", "DeleteMember", "ID"},
	{"network/v2/extensions/layer3/portforwarding", "InFloatingIP", "network/v2/extensions/layer3/floatingips", "Get", "List", "Delete", "ID"},
	{"network/v2/extensions/qos/rules", "BandwidthLimitRules", "network/v2/extensions/qos/policies", "GetBandwidthLimitRule", "ListBandwidthLimitRules", "DeleteBandwidthLimitRule", "ID"},
	{"network/v2/extensions/qos/rules", "DSCPMarkingRules", "network/v2/extensions/qos/policies", "GetDSCPMarkingRule", "ListDSCPMarkingRules", "DeleteDSCPMarkingRule", "ID"},
	{"network/v2/extensions/qos/rules", "MinimumBandwidthRules", "network/v2/extensions/qos/policies", "GetMinimumBandwidthRule", "ListMinimumBandwidthRules", "DeleteMinimumBandwidthRule", "ID"},
}

type scopePlan struct {
	spec       scopeSpec
	collection *collectionPlan
}

func identifyScopes(pkg *types.Package, decls map[string]*ast.FuncDecl, extractors map[string]string) ([]scopePlan, error) {
	var result []scopePlan
	for _, spec := range scopeSpecs {
		if sdkPath(pkg.Path()) != spec.path {
			continue
		}
		plan := identifyNamedCollection(pkg, decls, extractors, spec.get, []string{spec.list}, spec.delete, 1)
		if plan == nil || field(plan.model, spec.id) == "" || !isString(plan.getIDType) {
			return nil, fmt.Errorf("scope %s.%s does not match its typed upstream contract", spec.path, spec.method)
		}
		plan.id = spec.id
		result = append(result, scopePlan{spec, plan})
	}
	return result, nil
}

func (g *generator) emitScopes(pkg *types.Package, plans []scopePlan, apiSource []byte) error {
	if len(plans) == 0 {
		return nil
	}
	file, err := parser.ParseFile(token.NewFileSet(), "api_generated.go", apiSource, 0)
	if err != nil {
		return err
	}
	methods := map[string]*ast.FuncDecl{}
	imports := map[string]string{}
	for _, imp := range file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return err
		}
		if imp.Name != nil {
			imports[imp.Name.Name] = path
		}
	}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv != nil {
			methods[fn.Name.Name] = fn
		}
	}
	e := emitter{pkg: pkg, imports: map[string]string{}}
	e.use("context")
	e.use("gophercloudsdk/resource")
	for _, scope := range plans {
		plan, spec := scope.collection, scope.spec
		typeName := plan.modelName + "Scope"
		e.printf("// %s fixes the parent of its resources, sharing the SDK's lookup and wait policies.\ntype %s struct{*resource.Collection[%s];api *API;parentID string}\n", typeName, typeName, plan.modelName)
		e.printf("// %s resolves the parent once; explicit IDs require no lookup request.\nfunc(a *API)%s(ctx context.Context,parent resource.Ref)(*%s,error){\n", spec.method, spec.method, typeName)
		resolver := "a.Resources"
		if spec.parent != spec.path {
			resolver = e.use("gophercloudsdk/"+spec.parent) + ".New(a.client).Resources"
		}
		e.printf("id,err:=%s.ResolveID(ctx,parent);if err!=nil{return nil,err}\nscope:=&%s{api:a,parentID:id};scope.Collection=scope.newResources();return scope,nil}\n", resolver, typeName)
		e.printf("func(s *%s)newResources()*resource.Collection[%s]{return ", typeName, plan.modelName)
		emitCollectionAdapter(&e, plan, "s.api", []string{"s.parentID"})
		e.printf("}\n")
		if identityCollectionEnabled(pkg, plan, 1) {
			emitIdentityFind(&e, "s *"+typeName, "s.Collection", plan.modelName)
		}
		suffix := strings.TrimPrefix(spec.get, "Get")
		for _, mutation := range []struct {
			native, public string
			target         bool
		}{
			{"Create" + suffix, "Create", false},
			{"Update" + suffix, "Update", true},
		} {
			if method := methods[mutation.native]; method != nil {
				if err := emitScopedMutation(&e, typeName, method, mutation.public, mutation.target, imports); err != nil {
					return fmt.Errorf("scope %s.%s: %w", spec.path, spec.method, err)
				}
			}
		}
	}
	source, err := e.source()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(g.root, sdkPath(pkg.Path()), "scopes_generated.go"), source, 0644)
}

func nodeText(node ast.Expr) string {
	var out bytes.Buffer
	if err := format.Node(&out, token.NewFileSet(), node); err != nil {
		panic(err)
	}
	return out.String()
}

// Copy the builder-free API signature, bind its parent, and resolve mutation
// targets through the same collection. JSON patch slices and options stay typed.
func emitScopedMutation(e *emitter, receiver string, method *ast.FuncDecl, public string, target bool, imports map[string]string) error {
	fields := method.Type.Params.List
	if len(fields) < 2 || nodeText(fields[1].Type) != "string" || method.Type.Results == nil {
		return fmt.Errorf("%s has no single string parent", method.Name.Name)
	}
	results := method.Type.Results.List
	if len(results) == 0 || nodeText(results[len(results)-1].Type) != "error" {
		return fmt.Errorf("%s does not return an error", method.Name.Name)
	}
	params := []string{"ctx context.Context"}
	args := []string{"ctx", "s.parentID"}
	for i, field := range fields[2:] {
		if len(field.Names) != 1 {
			return fmt.Errorf("%s has grouped parameters", method.Name.Name)
		}
		if i == 0 && target {
			if nodeText(field.Type) != "string" {
				return fmt.Errorf("%s has no string target", method.Name.Name)
			}
			params = append(params, "ref resource.Ref")
			args = append(args, "id")
			continue
		}
		ast.Inspect(field.Type, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok && imports[ident.Name] != "" {
					e.use(imports[ident.Name])
				}
			}
			return true
		})
		params = append(params, field.Names[0].Name+" "+nodeText(field.Type))
		arg := field.Names[0].Name
		if _, ok := field.Type.(*ast.Ellipsis); ok {
			arg += "..."
		}
		args = append(args, arg)
	}
	returnTypes := []string{}
	for _, field := range results {
		returnTypes = append(returnTypes, nodeText(field.Type))
	}
	returnText := strings.Join(returnTypes, ",")
	if len(returnTypes) > 1 {
		returnText = "(" + returnText + ")"
	}
	e.printf("// %s applies %s within the fixed parent.\nfunc(s *%s)%s(%s)%s{\n", public, method.Name.Name, receiver, public, strings.Join(params, ","), returnText)
	if target {
		e.printf("id,err:=s.Collection.ResolveID(ctx,ref);if err!=nil{\n")
		zeros := []string{}
		for i, field := range results[:len(results)-1] {
			name := fmt.Sprintf("zero%d", i)
			e.printf("var %s %s\n", name, nodeText(field.Type))
			zeros = append(zeros, name)
		}
		e.printf("return %s}\n", strings.Join(append(zeros, "err"), ","))
	}
	e.printf("return s.api.%s(%s)}\n", method.Name.Name, strings.Join(args, ","))
	return nil
}
