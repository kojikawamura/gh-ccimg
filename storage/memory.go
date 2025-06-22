package storage

import (
	"encoding/base64"
	"fmt"
)

// MemoryStorage handles in-memory storage of images as base64 strings
type MemoryStorage struct {
	images []string
}

// NewMemoryStorage creates a new memory storage instance
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		images: make([]string, 0),
	}
}

// Store stores image data in memory as base64 and returns the encoded string
func (ms *MemoryStorage) Store(data []byte, contentType, url string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("cannot store empty data")
	}
	
	// Create base64 encoded string
	encoded := base64.StdEncoding.EncodeToString(data)
	
	// Store in memory
	ms.images = append(ms.images, encoded)
	
	return encoded, nil
}

// GetImages returns all stored base64 encoded images
func (ms *MemoryStorage) GetImages() []string {
	// Return a copy to prevent external modification
	result := make([]string, len(ms.images))
	copy(result, ms.images)
	return result
}

// Count returns the number of stored images
func (ms *MemoryStorage) Count() int {
	return len(ms.images)
}

// Clear removes all stored images
func (ms *MemoryStorage) Clear() {
	ms.images = ms.images[:0]
}

// GetImageData returns the raw image data for a given base64 string
func (ms *MemoryStorage) GetImageData(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, fmt.Errorf("encoded string cannot be empty")
	}
	
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 string: %w", err)
	}
	
	return data, nil
}

// EstimateMemoryUsage returns estimated memory usage in bytes
func (ms *MemoryStorage) EstimateMemoryUsage() int64 {
	var total int64
	for _, encoded := range ms.images {
		// Base64 encoding increases size by ~33%
		// Rough estimate: (len(encoded) * 3) / 4 for original size
		originalSize := (int64(len(encoded)) * 3) / 4
		total += originalSize
	}
	return total
}