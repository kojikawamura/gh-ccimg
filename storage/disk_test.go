package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDiskStorage(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	
	ds, err := NewDiskStorage(tempDir, false)
	if err != nil {
		t.Fatalf("NewDiskStorage failed: %v", err)
	}
	
	if ds == nil {
		t.Fatal("NewDiskStorage returned nil")
	}
	
	if ds.GetOutputDir() != tempDir {
		t.Errorf("OutputDir = %q, want %q", ds.GetOutputDir(), tempDir)
	}
	
	if ds.Count() != 0 {
		t.Errorf("New disk storage should be empty, got count %d", ds.Count())
	}
}

func TestNewDiskStorage_EmptyDir(t *testing.T) {
	_, err := NewDiskStorage("", false)
	if err == nil {
		t.Error("NewDiskStorage with empty directory should return error")
	}
}

func TestNewDiskStorage_CreateDir(t *testing.T) {
	tempDir := t.TempDir()
	newDir := filepath.Join(tempDir, "new", "nested", "dir")
	
	ds, err := NewDiskStorage(newDir, false)
	if err != nil {
		t.Fatalf("NewDiskStorage failed to create nested directory: %v", err)
	}
	
	// Verify directory was created
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("NewDiskStorage should create directory if it doesn't exist")
	}
	
	if ds.GetOutputDir() != newDir {
		t.Errorf("OutputDir = %q, want %q", ds.GetOutputDir(), newDir)
	}
}

func TestDiskStorage_Store(t *testing.T) {
	tempDir := t.TempDir()
	ds, _ := NewDiskStorage(tempDir, false)
	
	testData := []byte("test image data")
	
	filePath, err := ds.Store(testData, "image/png", "https://example.com/test.png")
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	
	if filePath == "" {
		t.Error("Store returned empty filepath")
	}
	
	// Verify file was created
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("File %s was not created", filePath)
	}
	
	// Verify file contents
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}
	
	if string(content) != string(testData) {
		t.Errorf("File content = %q, want %q", content, testData)
	}
	
	if ds.Count() != 1 {
		t.Errorf("Count = %d, want 1", ds.Count())
	}
	
	// Verify filename format
	expectedFilename := "img-01.png"
	if filePath != filepath.Join(tempDir, expectedFilename) {
		t.Errorf("Filepath = %q, want %q", filePath, filepath.Join(tempDir, expectedFilename))
	}
}

func TestDiskStorage_Store_EmptyData(t *testing.T) {
	tempDir := t.TempDir()
	ds, _ := NewDiskStorage(tempDir, false)
	
	_, err := ds.Store([]byte{}, "image/png", "test.png")
	if err == nil {
		t.Error("Store with empty data should return error")
	}
	
	if ds.Count() != 0 {
		t.Errorf("Count should remain 0 after failed store, got %d", ds.Count())
	}
}

func TestDiskStorage_Store_OverwriteProtection(t *testing.T) {
	tempDir := t.TempDir()
	ds, _ := NewDiskStorage(tempDir, false) // force=false
	
	testData := []byte("test data")
	
	// Store first file
	_, err := ds.Store(testData, "image/png", "test1.png")
	if err != nil {
		t.Fatalf("First store failed: %v", err)
	}
	
	// Create a file that would conflict with the second store
	conflictPath := filepath.Join(tempDir, "img-02.png")
	if err := os.WriteFile(conflictPath, []byte("existing"), 0644); err != nil {
		t.Fatalf("Failed to create conflict file: %v", err)
	}
	
	// Try to store second file (should fail due to existing file)
	_, err = ds.Store(testData, "image/png", "test2.png")
	if err == nil {
		t.Error("Store should fail when file exists and force=false")
	}
	
	if ds.Count() != 1 {
		t.Errorf("Count should remain 1 after failed store, got %d", ds.Count())
	}
}

func TestDiskStorage_Store_ForceOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	ds, _ := NewDiskStorage(tempDir, true) // force=true
	
	testData := []byte("test data")
	
	// Create a file that would conflict
	conflictPath := filepath.Join(tempDir, "img-01.png")
	if err := os.WriteFile(conflictPath, []byte("existing"), 0644); err != nil {
		t.Fatalf("Failed to create conflict file: %v", err)
	}
	
	// Store should succeed with force=true
	filePath, err := ds.Store(testData, "image/png", "test.png")
	if err != nil {
		t.Fatalf("Store with force=true failed: %v", err)
	}
	
	// Verify file was overwritten
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	
	if string(content) != string(testData) {
		t.Errorf("File was not overwritten correctly")
	}
}

func TestDiskStorage_GetFiles(t *testing.T) {
	tempDir := t.TempDir()
	ds, _ := NewDiskStorage(tempDir, false)
	
	// Store multiple files
	testData := []byte("test data")
	
	filePath1, _ := ds.Store(testData, "image/png", "test1.png")
	filePath2, _ := ds.Store(testData, "image/jpg", "test2.jpg")
	
	files := ds.GetFiles()
	
	if len(files) != 2 {
		t.Fatalf("GetFiles returned %d files, want 2", len(files))
	}
	
	if files[0] != filePath1 {
		t.Errorf("First file = %q, want %q", files[0], filePath1)
	}
	
	if files[1] != filePath2 {
		t.Errorf("Second file = %q, want %q", files[1], filePath2)
	}
	
	// Verify it returns a copy
	files[0] = "modified"
	newFiles := ds.GetFiles()
	if newFiles[0] == "modified" {
		t.Error("GetFiles should return a copy, not the original slice")
	}
}

func TestDiskStorage_Exists(t *testing.T) {
	tempDir := t.TempDir()
	ds, _ := NewDiskStorage(tempDir, false)
	
	// File doesn't exist yet
	if ds.Exists("img-01.png") {
		t.Error("Exists should return false for non-existent file")
	}
	
	// Store a file
	testData := []byte("test data")
	ds.Store(testData, "image/png", "test.png")
	
	// Now it should exist
	if !ds.Exists("img-01.png") {
		t.Error("Exists should return true for existing file")
	}
}

func TestDiskStorage_GetTotalSize(t *testing.T) {
	tempDir := t.TempDir()
	ds, _ := NewDiskStorage(tempDir, false)
	
	// Empty storage should have 0 size
	size, err := ds.GetTotalSize()
	if err != nil {
		t.Fatalf("GetTotalSize failed: %v", err)
	}
	if size != 0 {
		t.Errorf("Empty storage total size = %d, want 0", size)
	}
	
	// Store some files
	testData1 := []byte("test data 1")
	testData2 := []byte("test data 2 - longer")
	
	ds.Store(testData1, "image/png", "test1.png")
	ds.Store(testData2, "image/jpg", "test2.jpg")
	
	size, err = ds.GetTotalSize()
	if err != nil {
		t.Fatalf("GetTotalSize failed: %v", err)
	}
	
	expectedSize := int64(len(testData1) + len(testData2))
	if size != expectedSize {
		t.Errorf("Total size = %d, want %d", size, expectedSize)
	}
}

func TestDiskStorage_Cleanup(t *testing.T) {
	tempDir := t.TempDir()
	ds, _ := NewDiskStorage(tempDir, false)
	
	// Store some files
	testData := []byte("test data")
	filePath1, _ := ds.Store(testData, "image/png", "test1.png")
	filePath2, _ := ds.Store(testData, "image/jpg", "test2.jpg")
	
	// Verify files exist
	if _, err := os.Stat(filePath1); os.IsNotExist(err) {
		t.Error("File 1 should exist before cleanup")
	}
	if _, err := os.Stat(filePath2); os.IsNotExist(err) {
		t.Error("File 2 should exist before cleanup")
	}
	
	// Cleanup
	err := ds.Cleanup()
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}
	
	// Verify files are gone
	if _, err := os.Stat(filePath1); !os.IsNotExist(err) {
		t.Error("File 1 should be deleted after cleanup")
	}
	if _, err := os.Stat(filePath2); !os.IsNotExist(err) {
		t.Error("File 2 should be deleted after cleanup")
	}
	
	// Verify count is reset
	if ds.Count() != 0 {
		t.Errorf("Count after cleanup = %d, want 0", ds.Count())
	}
}