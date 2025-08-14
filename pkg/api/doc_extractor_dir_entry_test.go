package api

import (
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

// TestProcessDirectoryEntryComprehensive tests all paths in processDirectoryEntry
func TestProcessDirectoryEntryComprehensive(t *testing.T) {
	extractor := &DocExtractor{
		docs: make(map[string]Documentation),
	}
	fset := token.NewFileSet()

	t.Run("non-directory entry - early return", func(t *testing.T) {
		// Create a temporary file to test with
		tmpFile, err := os.CreateTemp("", "test*.go")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		// Write some Go code to the file
		tmpFile.WriteString("package main\n\nfunc main() {}")

		// Get file info
		fileInfo, err := os.Stat(tmpFile.Name())
		if err != nil {
			t.Fatal(err)
		}

		// This should trigger lines 54-56 (non-directory early return)
		fileDirEntry := &fileInfoDirEntry{fileInfo}
		err = extractor.processDirectoryEntry(tmpFile.Name(), fileDirEntry, fset)
		if err != nil {
			t.Errorf("Expected no error for file entry, got %v", err)
		}
	})

	t.Run("vendor directory - skip", func(t *testing.T) {
		// Create a temporary vendor directory
		tmpDir, err := os.MkdirTemp("", "vendor")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		// Rename to ensure it's actually named "vendor"
		vendorDir := filepath.Join(filepath.Dir(tmpDir), "vendor")
		err = os.Rename(tmpDir, vendorDir)
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(vendorDir)

		// Get directory info
		dirInfo, err := os.Stat(vendorDir)
		if err != nil {
			t.Fatal(err)
		}

		// This should trigger lines 58-60 (vendor directory skip)
		vendorDirEntry := &fileInfoDirEntry{dirInfo}
		err = extractor.processDirectoryEntry(vendorDir, vendorDirEntry, fset)
		if err != filepath.SkipDir {
			t.Errorf("Expected filepath.SkipDir for vendor directory, got %v", err)
		}
	})

	t.Run("directory read error", func(t *testing.T) {
		// Try to read a non-existent directory
		nonExistentDir := "/path/that/does/not/exist"

		// Create a mock DirEntry
		mockDirEntry := &mockTestDirEntry{name: "testdir", isDir: true}

		// This should trigger lines 63-66 (directory read error)
		err := extractor.processDirectoryEntry(nonExistentDir, mockDirEntry, fset)
		if err == nil {
			t.Error("Expected error when reading non-existent directory")
		}
	})

	t.Run("directory with Go files", func(t *testing.T) {
		// Create a temporary directory with Go files
		tmpDir, err := os.MkdirTemp("", "gofiles")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		// Create a Go file
		goFile := filepath.Join(tmpDir, "test.go")
		err = os.WriteFile(goFile, []byte("package test\n\n// TestType is a test type\ntype TestType struct {\n\t// Field is a test field\n\tField string\n}"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		// Create a non-Go file (should be skipped)
		txtFile := filepath.Join(tmpDir, "readme.txt")
		err = os.WriteFile(txtFile, []byte("This is not a Go file"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		// Create a subdirectory (should be skipped in the loop)
		subDir := filepath.Join(tmpDir, "subdir")
		err = os.Mkdir(subDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		// Get directory info
		dirInfo, err := os.Stat(tmpDir)
		if err != nil {
			t.Fatal(err)
		}

		// This should process the Go file and skip others
		dirDirEntry := &fileInfoDirEntry{dirInfo}
		err = extractor.processDirectoryEntry(tmpDir, dirDirEntry, fset)
		if err != nil {
			t.Errorf("Expected no error processing directory with Go files, got %v", err)
		}
	})

	t.Run("directory with unparseable Go file", func(t *testing.T) {
		// Create a temporary directory
		tmpDir, err := os.MkdirTemp("", "badgo")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		// Create an invalid Go file
		badGoFile := filepath.Join(tmpDir, "bad.go")
		err = os.WriteFile(badGoFile, []byte("this is not valid go syntax {{{"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		// Get directory info
		dirInfo, err := os.Stat(tmpDir)
		if err != nil {
			t.Fatal(err)
		}

		// This should trigger lines 74-77 (skip files that fail to parse)
		badDirEntry := &fileInfoDirEntry{dirInfo}
		err = extractor.processDirectoryEntry(tmpDir, badDirEntry, fset)
		if err != nil {
			t.Errorf("Expected no error when skipping unparseable files, got %v", err)
		}
	})
}

// mockTestDirEntry implements os.DirEntry for testing
type mockTestDirEntry struct {
	name  string
	isDir bool
}

func (m *mockTestDirEntry) Name() string {
	return m.name
}

func (m *mockTestDirEntry) IsDir() bool {
	return m.isDir
}

func (m *mockTestDirEntry) Type() os.FileMode {
	if m.isDir {
		return os.ModeDir
	}
	return 0
}

func (m *mockTestDirEntry) Info() (os.FileInfo, error) {
	return nil, nil
}

// fileInfoDirEntry wraps os.FileInfo to implement os.DirEntry
type fileInfoDirEntry struct {
	os.FileInfo
}

func (f *fileInfoDirEntry) Type() os.FileMode {
	return f.FileInfo.Mode().Type()
}

func (f *fileInfoDirEntry) Info() (os.FileInfo, error) {
	return f.FileInfo, nil
}
