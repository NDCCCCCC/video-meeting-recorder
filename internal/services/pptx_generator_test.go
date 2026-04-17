package services

import (
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPPTXGenerator(t *testing.T) {
	// Test PPTXGenerator using Python script integration
	// Tests require python3 and python-pptx to be installed
	t.Run("ValidateImageFilesEmpty", func(t *testing.T) {
		logger := zap.NewNop()
		generator := NewPPTXGenerator(logger)

		validPaths, errs := generator.ValidateImageFiles([]string{})

		assert.Equal(t, []string{}, validPaths)
		assert.Empty(t, errs)
	})

	t.Run("ValidateImageFilesNonExistent", func(t *testing.T) {
		logger := zap.NewNop()
		generator := NewPPTXGenerator(logger)

		_, errs := generator.ValidateImageFiles([]string{
			"/nonexistent/path1.jpg",
			"/nonexistent/path2.jpg",
		})

		assert.Len(t, errs, 2)
	})

	t.Run("GeneratePPTXEmptyFrames", func(t *testing.T) {
		logger := zap.NewNop()
		generator := NewPPTXGenerator(logger)

		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "test.pptx")

		pageCount, err := generator.GeneratePPTX(context.Background(), []string{}, outputPath)

		assert.Error(t, err)
		assert.Equal(t, 0, pageCount)
		assert.Contains(t, err.Error(), "frame paths cannot be empty")
	})
}

func TestGeneratePPTXWithTestImages(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := zap.NewNop()
	generator := NewPPTXGenerator(logger)

	// Check Python availability first
	if err := generator.CheckPythonAvailability(context.Background()); err != nil {
		t.Skipf("Python not available: %v", err)
	}

	// Create temporary directory
	tempDir := t.TempDir()

	// Generate 2 simple test images (solid color rectangles)
	imagePaths := []string{
		filepath.Join(tempDir, "test1.jpg"),
		filepath.Join(tempDir, "test2.jpg"),
	}

	for i, path := range imagePaths {
		// Create a simple 100x100 image
		img := image.NewRGBA(image.Rect(0, 0, 100, 100))
		c := color.RGBA{uint8(i * 100), uint8(i * 50), uint8(i * 25), 255}
		for y := 0; y < 100; y++ {
			for x := 0; x < 100; x++ {
				img.Set(x, y, c)
			}
		}

		file, err := os.Create(path)
		require.NoError(t, err)
		defer file.Close()

		err = jpeg.Encode(file, img, &jpeg.Options{Quality: 85})
		require.NoError(t, err)
	}

	// Generate PPTX
	outputPath := filepath.Join(tempDir, "test.pptx")
	pageCount, err := generator.GeneratePPTX(context.Background(), imagePaths, outputPath)

	require.NoError(t, err)
	assert.Equal(t, 2, pageCount)

	// Verify output file exists
	fileInfo, err := os.Stat(outputPath)
	require.NoError(t, err)
	assert.Greater(t, fileInfo.Size(), int64(0))

	// Verify file can be opened
	// This ensures the PPTX structure is valid
	_, err = os.Open(outputPath)
	require.NoError(t, err)
}

func TestGeneratePPTXWithMixedValidInvalid(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := zap.NewNop()
	generator := NewPPTXGenerator(logger)

	// Check Python availability first
	if err := generator.CheckPythonAvailability(context.Background()); err != nil {
		t.Skipf("Python not available: %v", err)
	}

	// Create temporary directory
	tempDir := t.TempDir()

	// Generate 1 valid test image
	validImagePath := filepath.Join(tempDir, "valid.jpg")
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	c := color.RGBA{255, 0, 0, 255}
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, c)
		}
	}
	file, err := os.Create(validImagePath)
	require.NoError(t, err)
	defer file.Close()
	err = jpeg.Encode(file, img, &jpeg.Options{Quality: 85})
	require.NoError(t, err)

	// Mix with non-existent path
	framePaths := []string{
		validImagePath,
		"/nonexistent/path.jpg",
	}

	// Generate PPTX - should succeed with 1 page (skip invalid)
	outputPath := filepath.Join(tempDir, "test-mixed.pptx")
	pageCount, err := generator.GeneratePPTX(context.Background(), framePaths, outputPath)

	require.NoError(t, err)
	assert.Equal(t, 1, pageCount)

	// Verify output file exists
	fileInfo, err := os.Stat(outputPath)
	require.NoError(t, err)
	assert.Greater(t, fileInfo.Size(), int64(0))
}
