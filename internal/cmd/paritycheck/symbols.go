package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type symbolIndex struct {
	api   map[string]bool
	tests map[string]bool
}

func goSymbols(root string) (symbolIndex, error) {
	index := symbolIndex{api: map[string]bool{}, tests: map[string]bool{}}
	module, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return index, err
	}
	fields := strings.Fields(string(module))
	if len(fields) < 2 || fields[0] != "module" {
		return index, fmt.Errorf("cannot determine module path")
	}
	modulePath := fields[1]
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		pkg := modulePath
		if dir := filepath.Dir(rel); dir != "." {
			pkg += "/" + filepath.ToSlash(dir)
		}
		testFile := strings.HasSuffix(entry.Name(), "_test.go")
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if testFile {
				if isTestFunction(file, fn) {
					index.tests[filepath.ToSlash(rel)+":"+fn.Name.Name] = true
				}
				continue
			}
			if !ast.IsExported(fn.Name.Name) || strings.HasPrefix(filepath.ToSlash(rel), "internal/") || file.Name.Name == "main" {
				continue
			}
			name := pkg + "." + fn.Name.Name
			if fn.Recv != nil {
				if len(fn.Recv.List) != 1 {
					continue
				}
				receiver := receiverName(fn.Recv.List[0].Type)
				if !ast.IsExported(receiver) {
					continue
				}
				name = pkg + "." + receiver + "." + fn.Name.Name
			}
			index.api[name] = true
		}
		return nil
	})
	return index, err
}

func receiverName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.StarExpr:
		return receiverName(expr.X)
	case *ast.IndexExpr:
		return receiverName(expr.X)
	case *ast.IndexListExpr:
		return receiverName(expr.X)
	case *ast.Ident:
		return expr.Name
	}
	return ""
}

func isTestFunction(file *ast.File, fn *ast.FuncDecl) bool {
	if !strings.HasPrefix(fn.Name.Name, "Test") || fn.Name.Name == "Test" || fn.Recv != nil || fn.Body == nil || fn.Type.Params == nil || len(fn.Type.Params.List) != 1 || fn.Type.Results != nil {
		return false
	}
	first, _ := utf8.DecodeRuneInString(strings.TrimPrefix(fn.Name.Name, "Test"))
	if unicode.IsLower(first) {
		return false
	}
	param := fn.Type.Params.List[0]
	if len(param.Names) > 1 {
		return false
	}
	pointer, ok := param.Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	selector, ok := pointer.X.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "T" {
		return false
	}
	alias, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	for _, imp := range file.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		if path == "testing" && (imp.Name == nil && alias.Name == "testing" || imp.Name != nil && imp.Name.Name == alias.Name) {
			return true
		}
	}
	return false
}
