package storage

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestNewMemoryStorage(t *testing.T) {
	ms := NewMemoryStorage()
	if ms == nil {
		t.Fatal("NewMemoryStorage returned nil")
	}
	
	if ms.Count() != 0 {
		t.Errorf("New memory storage should be empty, got count %d", ms.Count())
	}
}

func TestMemoryStorage_Store(t *testing.T) {
	ms := NewMemoryStorage()
	testData := []byte("test image data")
	
	encoded, err := ms.Store(testData, "image/png", "https://example.com/test.png")
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}
	
	if encoded == "" {
		t.Error("Store returned empty encoded string")
	}
	
	// Verify it's valid base64
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("Returned string is not valid base64: %v", err)
	}
	
	if !bytes.Equal(decoded, testData) {
		t.Errorf("Decoded data = %v, want %v", decoded, testData)
	}
	
	if ms.Count() != 1 {
		t.Errorf("Count = %d, want 1", ms.Count())
	}
}

func TestMemoryStorage_Store_EmptyData(t *testing.T) {
	ms := NewMemoryStorage()
	
	_, err := ms.Store([]byte{}, "image/png", "https://example.com/test.png")
	if err == nil {
		t.Error("Store with empty data should return error")
	}
	
	if ms.Count() != 0 {
		t.Errorf("Count should remain 0 after failed store, got %d", ms.Count())
	}
}

func TestMemoryStorage_GetImages(t *testing.T) {
	ms := NewMemoryStorage()
	
	// Store multiple images
	testData1 := []byte("image 1")
	testData2 := []byte("image 2")
	
	encoded1, _ := ms.Store(testData1, "image/png", "test1.png")
	encoded2, _ := ms.Store(testData2, "image/jpg", "test2.jpg")
	
	images := ms.GetImages()
	
	if len(images) != 2 {
		t.Fatalf("GetImages returned %d images, want 2", len(images))
	}
	
	if images[0] != encoded1 {
		t.Errorf("First image = %q, want %q", images[0], encoded1)
	}
	
	if images[1] != encoded2 {
		t.Errorf("Second image = %q, want %q", images[1], encoded2)
	}
	
	// Verify it returns a copy (modifying returned slice shouldn't affect storage)
	images[0] = "modified"
	newImages := ms.GetImages()
	if newImages[0] == "modified" {
		t.Error("GetImages should return a copy, not the original slice")
	}
}

func TestMemoryStorage_GetImageData(t *testing.T) {
	ms := NewMemoryStorage()
	testData := []byte("test image data")
	
	encoded, _ := ms.Store(testData, "image/png", "test.png")
	
	retrieved, err := ms.GetImageData(encoded)
	if err != nil {
		t.Fatalf("GetImageData failed: %v", err)
	}
	
	if !bytes.Equal(retrieved, testData) {
		t.Errorf("Retrieved data = %v, want %v", retrieved, testData)
	}
}

func TestMemoryStorage_GetImageData_InvalidBase64(t *testing.T) {
	ms := NewMemoryStorage()
	
	_, err := ms.GetImageData("invalid base64 string!!!")
	if err == nil {
		t.Error("GetImageData with invalid base64 should return error")
	}
}

func TestMemoryStorage_GetImageData_Empty(t *testing.T) {
	ms := NewMemoryStorage()
	
	_, err := ms.GetImageData("")
	if err == nil {
		t.Error("GetImageData with empty string should return error")
	}
}

func TestMemoryStorage_Clear(t *testing.T) {
	ms := NewMemoryStorage()
	
	// Store some images
	ms.Store([]byte("image 1"), "image/png", "test1.png")
	ms.Store([]byte("image 2"), "image/jpg", "test2.jpg")
	
	if ms.Count() != 2 {
		t.Fatalf("Expected 2 images before clear, got %d", ms.Count())
	}
	
	ms.Clear()
	
	if ms.Count() != 0 {
		t.Errorf("Count after clear = %d, want 0", ms.Count())
	}
	
	images := ms.GetImages()
	if len(images) != 0 {
		t.Errorf("GetImages after clear returned %d images, want 0", len(images))
	}
}

func TestMemoryStorage_EstimateMemoryUsage(t *testing.T) {
	ms := NewMemoryStorage()
	
	// Empty storage should have 0 usage
	if usage := ms.EstimateMemoryUsage(); usage != 0 {
		t.Errorf("Empty storage memory usage = %d, want 0", usage)
	}
	
	// Store some data
	testData := []byte("test data")
	ms.Store(testData, "image/png", "test.png")
	
	usage := ms.EstimateMemoryUsage()
	if usage <= 0 {
		t.Errorf("Memory usage should be positive, got %d", usage)
	}
	
	// Should roughly match the original data size
	expectedSize := int64(len(testData))
	if usage < expectedSize-5 || usage > expectedSize+5 {
		t.Errorf("Memory usage %d not close to expected %d", usage, expectedSize)
	}
}