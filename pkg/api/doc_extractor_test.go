package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDocExtractor(t *testing.T) {
	dir := t.TempDir()
	src := `package fixtures

// Foo represents something.
//
// Deprecated: use Bar instead.
type Foo struct {
    // ID is an identifier.
    ID string ` + "`json:\"id\"`" + `
}

// GetFoo returns foo.
// It does a thing.
func GetFoo() {}
`
	if err := os.WriteFile(filepath.Join(dir, "fixture.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	d := NewDocExtractor()
	if err := d.ParseDirectory(dir); err != nil {
		t.Fatalf("parse: %v", err)
	}

	td := d.ExtractTypeDoc("Foo")
	if td.Description != "Foo represents something." {
		t.Errorf("got desc %q", td.Description)
	}
	fd := d.ExtractFunctionDoc("GetFoo")
	if fd.Description != "GetFoo returns foo. It does a thing." {
		t.Errorf("function desc mismatch: %q", fd.Description)
	}
}
