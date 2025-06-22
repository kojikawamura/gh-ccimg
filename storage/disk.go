package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// DiskStorage handles file-based storage of images
type DiskStorage struct {
	outputDir string
	force     bool
	files     []string
}

// NewDiskStorage creates a new disk storage instance
func NewDiskStorage(outputDir string, force bool) (*DiskStorage, error) {
	if outputDir == "" {
		return nil, fmt.Errorf("output directory cannot be empty")
	}
	
	// Clean the path
	cleanDir := filepath.Clean(outputDir)
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(cleanDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory %s: %w", cleanDir, err)
	}
	
	return &DiskStorage{
		outputDir: cleanDir,
		force:     force,
		files:     make([]string, 0),
	}, nil
}

// Store saves image data to disk and returns the file path
func (ds *DiskStorage) Store(data []byte, contentType, url string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("cannot store empty data")
	}
	
	// Determine file extension
	extension := DetermineExtension(contentType, url)
	
	// Generate filename
	index := len(ds.files)
	filename := GenerateFilename(index, extension)
	filepath := filepath.Join(ds.outputDir, filename)
	
	// Check if file already exists and handle overwrite protection
	if !ds.force {
		if _, err := os.Stat(filepath); err == nil {
			return "", fmt.Errorf("file %s already exists (use --force to overwrite)", filepath)
		}
	}
	
	// Write file with proper permissions
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", filepath, err)
	}
	
	// Store the filename for tracking
	ds.files = append(ds.files, filepath)
	
	return filepath, nil
}

// GetFiles returns all stored file paths
func (ds *DiskStorage) GetFiles() []string {
	// Return a copy to prevent external modification
	result := make([]string, len(ds.files))
	copy(result, ds.files)
	return result
}

// Count returns the number of stored files
func (ds *DiskStorage) Count() int {
	return len(ds.files)
}

// GetOutputDir returns the output directory
func (ds *DiskStorage) GetOutputDir() string {
	return ds.outputDir
}

// Cleanup removes all stored files (use with caution)
func (ds *DiskStorage) Cleanup() error {
	var errors []error
	
	for _, filepath := range ds.files {
		if err := os.Remove(filepath); err != nil {
			errors = append(errors, fmt.Errorf("failed to remove %s: %w", filepath, err))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("cleanup failed with %d errors: %v", len(errors), errors[0])
	}
	
	// Clear the files list
	ds.files = ds.files[:0]
	return nil
}

// Exists checks if a file already exists at the given path
func (ds *DiskStorage) Exists(filename string) bool {
	filepath := filepath.Join(ds.outputDir, filename)
	_, err := os.Stat(filepath)
	return err == nil
}

// GetTotalSize returns the total size of all stored files
func (ds *DiskStorage) GetTotalSize() (int64, error) {
	var total int64
	
	for _, filepath := range ds.files {
		info, err := os.Stat(filepath)
		if err != nil {
			return 0, fmt.Errorf("failed to stat file %s: %w", filepath, err)
		}
		total += info.Size()
	}
	
	return total, nil
}