package main

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func objectStorageDocs(root string) ([]byte, error) {
	return serviceDocs(root, "objectstorage/v1/README.md", "objectstorage/v1")
}

func imageDocs(root string) ([]byte, error) {
	return serviceDocs(root, "image/v2/README.md", "image")
}

func serviceDocs(root, readmePath, packageDir string) ([]byte, error) {
	readme, err := os.ReadFile(filepath.Join(root, readmePath))
	if err != nil {
		return nil, err
	}
	text := []string{string(readme)}
	err = filepath.WalkDir(filepath.Join("..", "..", "..", filepath.FromSlash(packageDir)), func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "_generated.go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly|parser.ParseComments)
		if err != nil {
			return err
		}
		if file.Doc != nil {
			text = append(text, file.Doc.Text())
		}
		return nil
	})
	return []byte(strings.Join(text, "\n")), err
}
