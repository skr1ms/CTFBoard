package v1

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const modulePath = "github.com/TakuyaYagam1/AstroCTFb"

var forbiddenHandlerImports = []string{
	modulePath + "/internal/repo",
	modulePath + "/internal/storage",
	modulePath + "/internal/cache",
	modulePath + "/internal/domain",
	modulePath + "/internal/apperr",
	modulePath + "/internal/usecase",
	modulePath + "/pkg/",
}

func TestOrdinaryHandlersDoNotImportInnerAdapters(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read v1 dir: %v", err)
	}

	fset := token.NewFileSet()

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || strings.HasPrefix(name, "router") {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(".", name), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse imports in %s: %v", name, err)
		}

		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)

			for _, forbidden := range forbiddenHandlerImports {
				if path == strings.TrimSuffix(forbidden, "/") || strings.HasPrefix(path, forbidden) {
					t.Fatalf("%s imports forbidden handler dependency %q", name, path)
				}
			}
		}
	}
}
