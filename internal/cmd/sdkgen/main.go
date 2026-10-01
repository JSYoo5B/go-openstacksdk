// Command sdkgen generates typed, builder-free API facades from compiled upstream
// type information. It records unresolved cases instead of silently skipping them.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const upstreamModule = "github.com/gophercloud/gophercloud/v2"

type metadata struct {
	ImportPath, Dir, Export string
	GoFiles                 []string
}
type operation struct {
	Source        string `json:"source"`
	Package       string `json:"package"`
	Name          string `json:"name"`
	SDKPackage    string `json:"sdk_package"`
	BuilderFree   bool   `json:"builder_free"`
	ReturnPolicy  string `json:"return_policy"`
	RequestPolicy string `json:"request_policy,omitempty"`
	Issue         string `json:"issue,omitempty"`
}
type inventory struct {
	Version    string      `json:"gophercloud_version"`
	Operations []operation `json:"operations"`
}
type generator struct {
	meta        map[string]metadata
	importer    types.Importer
	root        string
	inventory   inventory
	collections []collectionRecord
}

func main() {
	metadataPath := flag.String("metadata", "", "go list -deps -export -json ./openstack/... output")
	output := flag.String("output", ".", "SDK module root")
	flag.Parse()
	if *metadataPath == "" {
		fatal(fmt.Errorf("-metadata is required"))
	}
	file, err := os.Open(*metadataPath)
	if err != nil {
		fatal(err)
	}
	defer file.Close()
	meta := map[string]metadata{}
	dec := json.NewDecoder(file)
	for {
		var m metadata
		err := dec.Decode(&m)
		if err == io.EOF {
			break
		}
		if err != nil {
			fatal(err)
		}
		meta[m.ImportPath] = m
	}
	g := generator{meta: meta, root: *output, inventory: inventory{Version: "v2.15.0"}}
	g.importer = importer.ForCompiler(token.NewFileSet(), "gc", func(path string) (io.ReadCloser, error) {
		m, ok := meta[path]
		if !ok || m.Export == "" {
			return nil, fmt.Errorf("missing export for %s", path)
		}
		return os.Open(m.Export)
	})
	paths := []string{}
	for path, m := range meta {
		if !strings.HasPrefix(path, upstreamModule+"/openstack/") || strings.Contains(path, "/testing") || strings.HasSuffix(m.Dir, "/testhelper") {
			continue
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if err := g.generate(path); err != nil {
			fatal(err)
		}
	}
	// SDK-owned services have no Gophercloud declaration to infer. Keep their
	// audited policy records in the same inventory without inventing native APIs.
	g.collections = append(g.collections, sdkOwnedCollections...)
	if err := g.generateServices(); err != nil {
		fatal(err)
	}
	data, err := json.MarshalIndent(g.inventory, "", "  ")
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(*output, "api"), 0755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(filepath.Join(*output, "api", "gophercloud_inventory.json"), append(data, '\n'), 0644); err != nil {
		fatal(err)
	}
	if err := g.writeCollectionInventory(); err != nil {
		fatal(err)
	}
	issues := 0
	for _, op := range g.inventory.Operations {
		if op.Issue != "" {
			issues++
		}
	}
	fmt.Printf("Recorded %d operations; unresolved: %d\n", len(g.inventory.Operations), issues)
}

func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }

func sdkPath(path string) string {
	rel := strings.TrimPrefix(path, upstreamModule+"/openstack/")
	parts := strings.Split(rel, "/")
	if parts[0] == "networking" {
		parts[0] = "network"
	}
	return strings.Join(parts, "/")
}

type emitter struct {
	pkg             *types.Package
	imports         map[string]string
	sourceImports   map[string]*types.Package
	controlledLists map[string]bool
	body            bytes.Buffer
}

func (e *emitter) use(path string) string {
	if path == e.pkg.Path() {
		path = e.pkg.Path()
	}
	if alias, ok := e.imports[path]; ok {
		return alias
	}
	alias := filepath.Base(path)
	switch path {
	case e.pkg.Path():
		alias = "upstream"
	case upstreamModule:
		alias = "gophercloud"
	}
	base := alias
	for i := 2; ; i++ {
		taken := false
		for _, value := range e.imports {
			if value == alias {
				taken = true
			}
		}
		if !taken {
			break
		}
		alias = base + strconv.Itoa(i)
	}
	e.imports[path] = alias
	return alias
}
func (e *emitter) typ(t types.Type) string {
	if param, ok := t.(*types.TypeParam); ok {
		base, err := concrete(e.pkg, param.Constraint())
		if err != nil {
			panic(err)
		}
		return e.typ(base)
	}
	if slice, ok := t.(*types.Slice); ok {
		return "[]" + e.typ(slice.Elem())
	}
	if named, ok := t.(*types.Named); ok && named.Obj().Pkg() == e.pkg && strings.HasSuffix(named.Obj().Name(), "Builder") {
		return e.use(e.pkg.Path()) + "." + named.Obj().Name()
	}
	return types.TypeString(t, func(p *types.Package) string {
		if p == e.pkg {
			return ""
		}
		return e.use(p.Path())
	})
}
func (e *emitter) printf(format string, args ...any) { fmt.Fprintf(&e.body, format, args...) }
func (e *emitter) source() ([]byte, error) {
	var out bytes.Buffer
	fmt.Fprintf(&out, "// Code generated by sdkgen from Gophercloud v2.15.0; DO NOT EDIT.\npackage %s\n\nimport (\n", e.pkg.Name())
	paths := make([]string, 0, len(e.imports))
	for path := range e.imports {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if !strings.Contains(e.body.String(), e.imports[path]+".") {
			continue
		}
		fmt.Fprintf(&out, "%s %q\n", e.imports[path], path)
	}
	out.WriteString(")\n\n")
	out.Write(e.body.Bytes())
	result, err := format.Source(out.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format %s: %w\n%s", e.pkg.Path(), err, out.String())
	}
	return result, nil
}

func (g *generator) generate(path string) error {
	m := g.meta[path]
	pkg, err := g.importer.Import(path)
	if err != nil {
		return err
	}
	decls := map[string]*ast.FuncDecl{}
	nativeDecls := map[string]*ast.FuncDecl{}
	nativeConstants := map[string]string{}
	sourceImports := map[string]*types.Package{}
	for _, name := range m.GoFiles {
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(m.Dir, name), nil, 0)
		if err != nil {
			return err
		}
		for name, value := range identitySourceConstants(f) {
			nativeConstants[name] = value
		}
		for _, spec := range f.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil || !strings.HasPrefix(path, upstreamModule+"/openstack/") {
				continue
			}
			imported, err := g.importer.Import(path)
			if err != nil {
				return err
			}
			alias := imported.Name()
			if spec.Name != nil {
				alias = spec.Name.Name
			}
			sourceImports[alias] = imported
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			nativeDecls[fn.Name.Name] = fn
			if ast.IsExported(fn.Name.Name) {
				decls[fn.Name.Name] = fn
			}
		}
	}
	if err := validateAuditedRequestCalls(pkg, decls); err != nil {
		return err
	}
	extractors := extractorsByPage(pkg, decls)
	plan, err := identifyCollectionBinding(pkg, decls, extractors)
	if err != nil {
		return err
	}
	scopes, err := identifyScopes(pkg, decls, extractors)
	if err != nil {
		return err
	}
	if err := validateIdentityCollectionContracts(pkg, nativeDecls, plan, scopes, nativeConstants); err != nil {
		return err
	}
	names := []string{}
	for _, name := range pkg.Scope().Names() {
		fn, ok := pkg.Scope().Lookup(name).(*types.Func)
		if !ok || !fn.Exported() {
			continue
		}
		sig := fn.Type().(*types.Signature)
		if clientParam(sig) >= 0 {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	e := emitter{pkg: pkg, imports: map[string]string{}, sourceImports: sourceImports, controlledLists: collectionControlledLists(plan, scopes)}
	e.use(pkg.Path())
	e.use(upstreamModule)
	specialized, hasSpecialized := specializedCollections[path]
	if hasSpecialized && (plan != nil || len(scopes) != 0) {
		return fmt.Errorf("specialized collection conflicts with inferred policies for %s", path)
	}
	if plan == nil {
		if hasSpecialized && specialized.Scope == "" && specialized.Kind == "" {
			e.use("gophercloudsdk/resource")
			e.printf("// API owns typed operations and their shared resource policies.\ntype API struct { client *gophercloud.ServiceClient; Resources *resource.Collection[%s] }\nfunc New(client *gophercloud.ServiceClient) *API { a:=&API{client:client};a.Resources=a.newResources();return a }\n", specialized.Model)
		} else {
			e.printf("// API owns the client and provides concrete inputs, optional extensions and normalized results.\ntype API struct { client *gophercloud.ServiceClient }\nfunc New(client *gophercloud.ServiceClient) *API { return &API{client:client} }\n")
		}
		if hasSpecialized {
			g.collections = append(g.collections, specialized)
		} else if len(scopes) == 0 {
			g.collections = append(g.collections, collectionRecord{Package: "gophercloudsdk/" + sdkPath(path), Issue: "requires a scoped or specialized resource binding"})
		}
	} else {
		e.use("gophercloudsdk/resource")
		e.printf("// API owns typed operations and their shared resource policies.\ntype API struct { client *gophercloud.ServiceClient; Resources *resource.Collection[%s] }\nfunc New(client *gophercloud.ServiceClient) *API { a:=&API{client:client};a.Resources=a.newResources();return a }\n", plan.modelName)
		g.collections = append(g.collections, collectionRecord{Package: "gophercloudsdk/" + sdkPath(path), Model: plan.modelName, Find: plan.name != "", IdentityFind: identityCollectionEnabled(pkg, plan, 0), IdentityGetQuery: identityCollectionEnabled(pkg, plan, 0), Delete: plan.deleter != nil, Wait: plan.status != ""})
	}
	e.printf("func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }\n\n")
	for _, name := range pkg.Scope().Names() {
		obj := pkg.Scope().Lookup(name)
		if !obj.Exported() {
			continue
		}
		switch obj.(type) {
		case *types.TypeName:
			if strings.HasSuffix(name, "Builder") || name == "API" {
				continue
			}
			e.printf("type %s = upstream.%s\n", name, name)
		case *types.Const:
			e.printf("const %s = upstream.%s\n", name, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		fn := pkg.Scope().Lookup(name).(*types.Func)
		op := operation{Source: path, Package: strings.TrimPrefix(path, upstreamModule+"/openstack/"), Name: name, SDKPackage: "gophercloudsdk/" + sdkPath(path)}
		if err := emitOperation(&e, fn, decls[name], extractors); err != nil {
			op.Issue = err.Error()
		} else {
			op.BuilderFree = true
			op.ReturnPolicy = operationReturnPolicy(fn)
			if override := requestCallOverride(pkg, name); override != nil {
				op.RequestPolicy = override.policy
			}
		}
		g.inventory.Operations = append(g.inventory.Operations, op)
	}
	source, err := e.source()
	if err != nil {
		return err
	}
	dir := filepath.Join(g.root, sdkPath(path))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "api_generated.go"), source, 0644); err != nil {
		return err
	}
	if plan != nil {
		if err := g.emitCollection(pkg, plan); err != nil {
			return err
		}
	}
	for _, scope := range scopes {
		p := scope.collection
		g.collections = append(g.collections, collectionRecord{Package: "gophercloudsdk/" + sdkPath(path), Model: p.modelName, Find: p.name != "", IdentityFind: identityCollectionEnabled(pkg, p, 1), IdentityGetQuery: identityCollectionEnabled(pkg, p, 1), Delete: p.deleter != nil, Wait: p.status != "", Scope: scope.spec.method, Parent: "gophercloudsdk/" + scope.spec.parent})
	}
	return g.emitScopes(pkg, scopes, source)
}

// Typed extractors often delegate to ExtractInto helpers. Follow those calls
// to identify the concrete page, rather than exposing raw pages to callers.
func extractorsByPage(pkg *types.Package, decls map[string]*ast.FuncDecl) map[string]string {
	result := map[string]string{}
	for _, name := range pkg.Scope().Names() {
		fn, ok := pkg.Scope().Lookup(name).(*types.Func)
		if !ok || !strings.HasPrefix(name, "Extract") {
			continue
		}
		sig := fn.Type().(*types.Signature)
		if sig.Params().Len() != 1 || sig.Results().Len() != 2 || !isError(sig.Results().At(1).Type()) {
			continue
		}
		seen := map[string]bool{}
		var visit func(string)
		visit = func(current string) {
			if seen[current] || decls[current] == nil {
				return
			}
			seen[current] = true
			ast.Inspect(decls[current].Body, func(node ast.Node) bool {
				switch node := node.(type) {
				case *ast.TypeAssertExpr:
					if id, ok := node.Type.(*ast.Ident); ok {
						previous := result[id.Name]
						if previous == "" || pageExtractorDetail(sig.Results().At(0).Type()) >= pageExtractorDetail(pkg.Scope().Lookup(previous).Type().(*types.Signature).Results().At(0).Type()) {
							result[id.Name] = name
						}
					}
				case *ast.CallExpr:
					if id, ok := node.Fun.(*ast.Ident); ok {
						visit(id.Name)
					}
				}
				return true
			})
		}
		visit(name)
	}
	return result
}

// Swift exposes both full resource records and names from the same page. Do
// not let an ExtractNames helper discard fields from an available typed model.
func pageExtractorDetail(result types.Type) int {
	if slice, ok := result.Underlying().(*types.Slice); ok {
		if _, primitive := slice.Elem().Underlying().(*types.Basic); primitive {
			return 0
		}
	}
	return 1
}

func clientParam(sig *types.Signature) int {
	for i := 0; i < sig.Params().Len(); i++ {
		s := types.TypeString(sig.Params().At(i).Type(), func(p *types.Package) string { return p.Path() })
		if s == "*"+upstreamModule+".ServiceClient" || s == "*"+upstreamModule+".ProviderClient" {
			return i
		}
	}
	return -1
}

func isContext(t types.Type) bool {
	return types.TypeString(t, func(p *types.Package) string { return p.Path() }) == "context.Context"
}
func isError(t types.Type) bool { return types.Identical(t, types.Universe.Lookup("error").Type()) }
func ifaceOf(t types.Type) (*types.Interface, bool) {
	v, ok := t.Underlying().(*types.Interface)
	return v, ok
}

func concrete(pkg *types.Package, t types.Type) (types.Type, error) {
	iface, ok := ifaceOf(t)
	if !ok {
		return nil, fmt.Errorf("not a builder")
	}
	name := ""
	if named, ok := t.(*types.Named); ok {
		name = strings.TrimSuffix(named.Obj().Name(), "Builder")
	}
	if obj := pkg.Scope().Lookup(name); obj != nil {
		if n, ok := obj.(*types.TypeName); ok {
			if types.Implements(n.Type(), iface) {
				return n.Type(), nil
			}
			if types.Implements(types.NewPointer(n.Type()), iface) {
				return types.NewPointer(n.Type()), nil
			}
		}
	}
	for _, key := range pkg.Scope().Names() {
		obj, ok := pkg.Scope().Lookup(key).(*types.TypeName)
		if !ok || !obj.Exported() {
			continue
		}
		if _, ok := ifaceOf(obj.Type()); ok {
			continue
		}
		if types.Implements(obj.Type(), iface) {
			return obj.Type(), nil
		}
		if types.Implements(types.NewPointer(obj.Type()), iface) {
			return types.NewPointer(obj.Type()), nil
		}
	}
	return nil, fmt.Errorf("no concrete implementation of %s", t)
}

type builder struct {
	index   int
	name    string
	base    types.Type
	iface   *types.Interface
	adapter string
}

func extraction(t types.Type) *types.Signature {
	obj, _, _ := types.LookupFieldOrMethod(t, true, nil, "Extract")
	if obj == nil {
		obj, _, _ = types.LookupFieldOrMethod(types.NewPointer(t), true, nil, "Extract")
	}
	if fn, ok := obj.(*types.Func); ok {
		sig := fn.Type().(*types.Signature)
		if sig.Params().Len() == 0 && sig.Results().Len() > 0 && isError(sig.Results().At(sig.Results().Len()-1).Type()) {
			return sig
		}
	}
	return nil
}
func hasExtractErr(t types.Type) bool {
	obj, _, _ := types.LookupFieldOrMethod(t, true, nil, "ExtractErr")
	fn, ok := obj.(*types.Func)
	return ok && fn.Type().(*types.Signature).Params().Len() == 0
}
func isPager(t types.Type) bool {
	return types.TypeString(t, func(p *types.Package) string { return p.Path() }) == upstreamModule+"/pagination.Pager"
}
func returnPolicy(sig *types.Signature) string {
	if sig.Results().Len() == 1 {
		t := sig.Results().At(0).Type()
		if isPager(t) {
			return "stream"
		}
		if extraction(t) != nil {
			if body, _, _ := types.LookupFieldOrMethod(t, true, nil, "Body"); body != nil && types.TypeString(body.Type(), func(p *types.Package) string { return p.Path() }) == "io.ReadCloser" {
				return "download"
			}
			return "extract"
		}
		if hasExtractErr(t) {
			return "error"
		}
		if isError(t) {
			return "direct-error"
		}
		if member, _, _ := types.LookupFieldOrMethod(t, true, nil, "Err"); member != nil {
			return "result"
		}
		return "direct"
	}
	return "direct"
}

func emitOperation(e *emitter, fn *types.Func, decl *ast.FuncDecl, extractors map[string]string) error {
	override := requestCallOverride(e.pkg, fn.Name())
	if override != nil {
		if err := validateAuditedRequestCall(e.pkg, map[string]*ast.FuncDecl{fn.Name(): decl}, *override); err != nil {
			return err
		}
	}
	sig := fn.Type().(*types.Signature)
	op := fn.Name()
	normalizer := normalizerFor(fn)
	builders := []builder{}
	for i := 0; i < sig.Params().Len(); i++ {
		v := sig.Params().At(i)
		iface, ok := ifaceOf(v.Type())
		if !ok {
			if named, ok := v.Type().(*types.Named); ok && len(builders) == 0 && named.Obj().Pkg() == e.pkg && strings.HasSuffix(named.Obj().Name(), "Opts") {
				builders = append(builders, builder{index: i, name: v.Name(), base: v.Type()})
			}
			continue
		}
		builderLike := false
		for j := 0; j < iface.NumMethods(); j++ {
			if strings.HasPrefix(iface.Method(j).Name(), "To") {
				builderLike = true
			}
		}
		if !builderLike {
			continue
		}
		base, err := concrete(e.pkg, v.Type())
		if err != nil {
			return err
		}
		builders = append(builders, builder{index: i, name: v.Name(), base: base, iface: iface, adapter: lower(op) + title(v.Name()) + "Builder"})
	}
	for i := range builders {
		builders[i] = withOptionalBuilders(e.pkg, builders[i], decl, e.sourceImports)
	}
	contextAlias := e.use("context")
	requestAlias := e.use("gophercloudsdk/request")
	params := []string{"ctx " + contextAlias + ".Context"}
	forwardArgs := []string{"ctx"}
	args := []string{}
	clientIndex := clientParam(sig)
	defaultOptions := strings.HasPrefix(op, "List") || strings.HasPrefix(op, "Delete") || strings.HasPrefix(op, "Get") || strings.HasPrefix(op, "Head") || strings.HasPrefix(op, "Download")
	for i := 0; i < sig.Params().Len(); i++ {
		v := sig.Params().At(i)
		if i == clientIndex {
			if strings.Contains(types.TypeString(v.Type(), nil), "ProviderClient") {
				args = append(args, "a.client.ProviderClient")
			} else {
				args = append(args, "a.client")
			}
			continue
		}
		if isContext(v.Type()) {
			args = append(args, "ctx")
			continue
		}
		bi := -1
		for j, b := range builders {
			if b.index == i {
				bi = j
			}
		}
		if bi >= 0 {
			b := builders[bi]
			if b.iface == nil {
				args = append(args, "cfg.Options")
			} else {
				args = append(args, "_"+b.name)
			}
			if bi == 0 && !defaultOptions {
				params = append(params, b.name+" "+e.typ(b.base))
				forwardArgs = append(forwardArgs, b.name)
			}
			continue
		}
		name := v.Name()
		if name == "" {
			name = fmt.Sprintf("arg%d", i)
		}
		if name == "a" || name == "ctx" {
			name += "Value"
		}
		typeText := e.typ(v.Type())
		if sig.Variadic() && i == sig.Params().Len()-1 {
			typeText = "..." + e.typ(v.Type().(*types.Slice).Elem())
			args = append(args, name+"...")
			forwardArgs = append(forwardArgs, name+"...")
		} else {
			args = append(args, name)
			forwardArgs = append(forwardArgs, name)
		}
		params = append(params, name+" "+typeText)
	}
	if len(builders) > 0 && sig.Variadic() {
		return fmt.Errorf("builder plus variadic input requires custom adapter")
	}
	if len(builders) > 0 {
		primary := builders[0]
		base := e.typ(primary.base)
		e.printf("type %sOption = %s.Option[%s]\n", op, requestAlias, base)
		e.printf("func With%sOptions(value %s) %sOption { return %s.WithOptions(value) }\n", op, base, op, requestAlias)
		caps := capabilities(e.pkg, primary)
		if caps.body {
			e.printf("func With%sField(key string,value any) %sOption { return %s.WithField[%s](key,value) }\n", op, op, requestAlias, base)
		}
		if caps.query {
			e.printf("func With%sQuery(key,value string) %sOption { return %s.WithQuery[%s](key,value) }\n", op, op, requestAlias, base)
		}
		if caps.headers {
			e.printf("func With%sHeader(key,value string) %sOption {return %s.WithHeader[%s](key,value)}\n", op, op, requestAlias, base)
		}
		for _, b := range builders[1:] {
			e.printf("func With%s%s(value %s) %sOption { return %s.WithArgument[%s](%q,value) }\n", op, title(b.name), e.typ(b.base), op, requestAlias, base, b.name)
		}
		params = append(params, "options ..."+op+"Option")
		forwardArgs = append(forwardArgs, "options...")
		for _, b := range builders {
			if b.iface != nil {
				emitBuilder(e, b)
			}
		}
	}
	returnTypes := []types.Type{}
	policy := operationReturnPolicy(fn)
	if normalizer != nil && normalizer.options != "" {
		if len(builders) != 0 || sig.Variadic() {
			return fmt.Errorf("result options conflict with request options")
		}
		params = append(params, "options ..."+normalizer.options)
		forwardArgs = append(forwardArgs, "options...")
	}
	resultExtractor, resultExtraction := operationExtractor(fn)
	extractName := ""
	extractPackage := e.use(e.pkg.Path())
	var streamType types.Type
	streamValues := false
	if policy == "stream" {
		extractName, streamType = findExtractor(e.pkg, op, decl, extractors)
		if streamType == nil {
			if fn := importedExtractor(e.sourceImports, decl); fn != nil {
				extractName = fn.Name()
				extractPackage = e.use(fn.Pkg().Path())
				streamType = fn.Type().(*types.Signature).Results().At(0).Type()
			}
		}
		if streamType != nil {
			if slice, ok := streamType.Underlying().(*types.Slice); ok {
				streamType = slice.Elem()
			} else {
				streamValues = true
			}
		}
	} else if policy == "extract" || policy == "download" {
		ex := resultExtraction
		for i := 0; i < ex.Results().Len(); i++ {
			returnTypes = append(returnTypes, ex.Results().At(i).Type())
		}
	}
	controlled := e.controlledLists[op]
	if controlled && (policy != "stream" || streamType == nil || streamValues) {
		return fmt.Errorf("controlled collection list %s requires a typed slice pager", op)
	}
	if policy == "error" || policy == "direct-error" {
		returnTypes = []types.Type{types.Universe.Lookup("error").Type()}
	}
	if policy == "direct" {
		for i := 0; i < sig.Results().Len(); i++ {
			returnTypes = append(returnTypes, sig.Results().At(i).Type())
		}
	}
	if policy == "result" {
		returnTypes = append(returnTypes, sig.Results().At(0).Type(), types.Universe.Lookup("error").Type())
	}
	returns := ""
	streamText := ""
	if policy == "normalize" {
		returns = "(" + normalizer.valueType + ",error)"
	} else if policy == "stream" {
		iterAlias := e.use("iter")
		if streamType != nil {
			streamText = e.typ(streamType)
			if !streamValues {
				streamText = "*" + streamText
			}
		} else {
			streamText = e.use(upstreamModule+"/pagination") + ".Page"
		}
		returns = iterAlias + ".Seq2[" + streamText + ",error]"
	} else if policy == "download" {
		returns = "(*" + requestAlias + ".Download[" + e.typ(returnTypes[0]) + "],error)"
	} else {
		items := []string{}
		for _, t := range returnTypes {
			items = append(items, e.typ(t))
		}
		if len(items) == 1 {
			returns = items[0]
		} else {
			returns = "(" + strings.Join(items, ",") + ")"
		}
	}
	e.printf("\n// %s invokes the upstream API with library-owned builders and result handling.\nfunc (a *API) %s(%s) %s {\n", op, op, strings.Join(params, ","), returns)
	if controlled {
		resourceAlias := e.use("gophercloudsdk/resource")
		// A control parameter precedes variadic options, keeping the public
		// signature and its native option preparation unchanged.
		position := len(params)
		if len(forwardArgs) > 0 && strings.HasSuffix(forwardArgs[len(forwardArgs)-1], "...") {
			position--
		}
		privateParams := append([]string(nil), params[:position]...)
		privateParams = append(privateParams, "control "+resourceAlias+".ListControl")
		privateParams = append(privateParams, params[position:]...)
		delegateArgs := append([]string(nil), forwardArgs[:position]...)
		delegateArgs = append(delegateArgs, resourceAlias+".ListControl{}")
		delegateArgs = append(delegateArgs, forwardArgs[position:]...)
		e.printf("return a.%s(%s)\n}\n\nfunc (a *API) %s(%s) %s {\n", controlledListName(op), strings.Join(delegateArgs, ","), controlledListName(op), strings.Join(privateParams, ","), returns)
	}
	errReturn := func() {
		e.printf("err=%s.Wrap(%q,%q,err)\n", requestAlias, op, e.pkg.Name())
		if policy == "normalize" {
			e.printf("return %s,err\n", normalizer.zero)
			return
		}
		if policy == "download" {
			e.printf("return nil,err\n")
			return
		}
		if policy == "stream" {
			e.printf("return func(yield func(%s,error)bool){var zero %s;yield(zero,err)}\n", streamText, streamText)
			return
		}
		values := []string{}
		for i, t := range returnTypes {
			if isError(t) {
				values = append(values, "err")
			} else {
				name := fmt.Sprintf("zero%d", i)
				e.printf("var %s %s\n", name, e.typ(t))
				values = append(values, name)
			}
		}
		e.printf("return %s\n", strings.Join(values, ","))
	}
	if normalizer != nil && normalizer.prepare != "" {
		e.printf("%s\nif err!=nil{\n", normalizer.prepare)
		errReturn()
		e.printf("}\n")
	}
	if len(builders) > 0 {
		primary := builders[0]
		if defaultOptions {
			if pointer, ok := primary.base.(*types.Pointer); ok {
				e.printf("%s:=new(%s)\n", primary.name, e.typ(pointer.Elem()))
			} else {
				e.printf("var %s %s\n", primary.name, e.typ(primary.base))
			}
		}
		e.printf("cfg,err:=%s.Apply(%s,options...)\nif err!=nil {\n", requestAlias, primary.name)
		errReturn()
		e.printf("}\n")
		caps := capabilities(e.pkg, primary)
		allowed := []string{}
		for _, b := range builders[1:] {
			allowed = append(allowed, strconv.Quote(b.name))
		}
		arguments := ""
		if len(allowed) > 0 {
			arguments = "," + strings.Join(allowed, ",")
		}
		e.printf("if err=%s.ValidateCapabilities(cfg,%t,%t,%t%s);err!=nil{\n", requestAlias, caps.body, caps.query, caps.headers, arguments)
		errReturn()
		e.printf("}\n")
		if primary.iface != nil {
			e.printf("_%s := %s{base:cfg.Options,config:cfg}\n", primary.name, primary.adapter)
		}
		for _, b := range builders[1:] {
			e.printf("var _%s %s\n", b.name, e.typ(sig.Params().At(b.index).Type()))
			e.printf("{base,provided,err:=%s.Argument[%s](cfg,%q)\nif err!=nil{\n", requestAlias, e.typ(b.base), b.name)
			errReturn()
			e.printf("}\nif provided{_%s=%s{base:base,config:%s.Config[%s]{Options:base}}}}\n", b.name, b.adapter, requestAlias, e.typ(b.base))
		}
	}
	call := "upstream." + op + "(" + strings.Join(args, ",") + ")"
	if override != nil {
		call = override.helper + "(" + strings.Join(args, ",") + ")"
	}
	switch policy {
	case "normalize":
		e.printf("result:=%s\nvalue,err:=%s\nreturn value,%s.Wrap(%q,%q,err)\n", call, normalizer.call, requestAlias, op, e.pkg.Name())
	case "stream":
		resourceAlias := e.use("gophercloudsdk/resource")
		if streamType != nil {
			if streamValues {
				e.printf("return %s.StreamValues(ctx,%s,%s.%s)\n", resourceAlias, call, extractPackage, extractName)
			} else {
				pageAlias := e.use(upstreamModule + "/pagination")
				stream := "Stream"
				control := ""
				if controlled {
					stream, control = "StreamWithControl", ",control"
				}
				e.printf("return %s.%s(ctx,%s,func(page %s.Page)([]%s,error){values,err:=%s.%s(page);return []%s(values),err}%s)\n", resourceAlias, stream, call, pageAlias, e.typ(streamType), extractPackage, extractName, e.typ(streamType), control)
			}
		} else {
			e.printf("return %s.Pages(ctx,%s)\n", resourceAlias, call)
		}
	case "extract":
		e.printf("result:=%s\n", call)
		vals := []string{}
		for i, t := range returnTypes {
			if isError(t) {
				vals = append(vals, "err")
			} else {
				vals = append(vals, fmt.Sprintf("value%d", i))
			}
		}
		e.printf("%s:=result.%s()\nerr=%s.Wrap(%q,%q,err)\nreturn %s\n", strings.Join(vals, ","), resultExtractor, requestAlias, op, e.pkg.Name(), strings.Join(vals, ","))
	case "download":
		e.printf("result:=%s\nheader,err:=result.Extract()\nreturn %s.OpenDownload(result.Body,header,%s.Wrap(%q,%q,err))\n", call, requestAlias, requestAlias, op, e.pkg.Name())
	case "error":
		e.printf("return %s.Wrap(%q,%q,%s.ExtractErr())\n", requestAlias, op, e.pkg.Name(), call)
	case "direct-error":
		e.printf("return %s.Wrap(%q,%q,%s)\n", requestAlias, op, e.pkg.Name(), call)
	case "result":
		e.printf("result:=%s\nreturn result,%s.Wrap(%q,%q,result.Err)\n", call, requestAlias, op, e.pkg.Name())
	case "direct":
		e.printf("return %s\n", call)
	}
	e.printf("}\n")
	return nil
}

func importedExtractor(imports map[string]*types.Package, decl *ast.FuncDecl) *types.Func {
	if decl == nil {
		return nil
	}
	var found *types.Func
	ast.Inspect(decl.Body, func(node ast.Node) bool {
		lit, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		selector, ok := lit.Type.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		alias, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		if imported := imports[alias.Name]; imported != nil {
			name := "Extract" + strings.TrimSuffix(selector.Sel.Name, "Page") + "s"
			fn, ok := imported.Scope().Lookup(name).(*types.Func)
			if !ok {
				return true
			}
			sig := fn.Type().(*types.Signature)
			if sig.Params().Len() == 1 && sig.Results().Len() == 2 && isError(sig.Results().At(1).Type()) {
				found = fn
			}
		}
		return true
	})
	return found
}

func emitBuilder(e *emitter, b builder) {
	req := e.use("gophercloudsdk/request")
	e.printf("type %s struct{base %s;config %s.Config[%s]}\n", b.adapter, e.typ(b.base), req, e.typ(b.base))
	for i := 0; i < b.iface.NumMethods(); i++ {
		method := b.iface.Method(i)
		sig := method.Type().(*types.Signature)
		params, args := []string{}, []string{}
		for j := 0; j < sig.Params().Len(); j++ {
			name := fmt.Sprintf("arg%d", j)
			params = append(params, name+" "+e.typ(sig.Params().At(j).Type()))
			args = append(args, name)
		}
		returns := []string{}
		for j := 0; j < sig.Results().Len(); j++ {
			returns = append(returns, e.typ(sig.Results().At(j).Type()))
		}
		r := strings.Join(returns, ",")
		if len(returns) > 1 {
			r = "(" + r + ")"
		}
		e.printf("func (b %s) %s(%s) %s {\n", b.adapter, method.Name(), strings.Join(params, ","), r)
		call := "b.base." + method.Name() + "(" + strings.Join(args, ",") + ")"
		if !emitConfiguredBuilderMethod(e, b, method, call) {
			e.printf("return %s\n", call)
		}
		e.printf("}\n")
	}
}
func isString(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Kind() == types.String
}
func isAnyMap(t types.Type) bool {
	m, ok := t.Underlying().(*types.Map)
	if !ok || !isString(m.Key()) {
		return false
	}
	iface, ok := m.Elem().Underlying().(*types.Interface)
	return ok && iface.NumMethods() == 0
}
func lower(s string) string {
	if s == "" {
		return "argument"
	}
	return strings.ToLower(s[:1]) + s[1:]
}
func title(s string) string {
	if s == "" {
		return "Argument"
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func findExtractor(pkg *types.Package, name string, decl *ast.FuncDecl, byPage map[string]string) (string, types.Type) {
	validate := func(name string) (string, types.Type) {
		obj := pkg.Scope().Lookup(name)
		fn, ok := obj.(*types.Func)
		if !ok {
			return "", nil
		}
		sig := fn.Type().(*types.Signature)
		if sig.Params().Len() != 1 || sig.Results().Len() != 2 || !isError(sig.Results().At(1).Type()) {
			return "", nil
		}
		return name, sig.Results().At(0).Type()
	}
	if decl != nil {
		var found string
		var typ types.Type
		ast.Inspect(decl.Body, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if ok {
				if id, ok := lit.Type.(*ast.Ident); ok {
					if ex := byPage[id.Name]; ex != "" {
						found, typ = validate(ex)
					}
				}
			}
			return true
		})
		if found != "" {
			return found, typ
		}
	}
	if ex, typ := validate("Extract" + strings.TrimPrefix(name, "List")); ex != "" {
		return ex, typ
	}
	if ex, typ := validate("Extract" + strings.TrimPrefix(name, "Get")); ex != "" {
		return ex, typ
	}
	found := ""
	var typ types.Type
	count := 0
	for _, key := range pkg.Scope().Names() {
		if !strings.HasPrefix(key, "Extract") {
			continue
		}
		if ex, t := validate(key); ex != "" {
			found, typ = ex, t
			count++
		}
	}
	if count == 1 {
		return found, typ
	}
	return "", nil
}
