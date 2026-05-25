package beads

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

func TestLibraryImportPolicy(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	fset := token.NewFileSet()
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, spec := range parsed.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			switch {
			case path == "log", path == "log/slog":
				t.Fatalf("%s imports %q; callers own logging", file, path)
			case strings.Contains(strings.ToLower(path), "opentelemetry") ||
				strings.HasPrefix(path, "go.opentelemetry.io/"):
				t.Fatalf("%s imports %q; callers own telemetry", file, path)
			}
			if strings.Contains(path, ".") {
				t.Fatalf("%s imports third-party package %q; v1 must remain stdlib-only", file, path)
			}
		}
	}
}
