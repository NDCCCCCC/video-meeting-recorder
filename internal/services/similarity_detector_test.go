package services

import (
	"image"
	"image/color"
	"testing"

	"go.uber.org/zap"
)

// TestCalculateSSIMIdentical tests that identical images return SSIM ~1.0
func TestCalculateSSIMIdentical(t *testing.T) {
	logger := zap.NewNop()
	detector := NewSimilarityDetector(logger)

	// Create identical images
	img := createTestImage(100, 100, color.RGBA{128, 128, 128, 255})

	result, err := detector.IsFrameChanged(img, img)
	if err != nil {
		t.Fatalf("IsFrameChanged failed: %v", err)
	}

	if result.SSIMScore < 0.99 {
		t.Errorf("Expected SSIM ~1.0 for identical images, got %f", result.SSIMScore)
	}

	if result.Changed {
		t.Error("Expected identical images to not be marked as changed")
	}
}

// TestCalculateSSIMDifferent tests that very different images return SSIM < 0.85
func TestCalculateSSIMDifferent(t *testing.T) {
	logger := zap.NewNop()
	detector := NewSimilarityDetector(logger)

	// Create very different images
	img1 := createTestImage(100, 100, color.RGBA{0, 0, 0, 255})       // Black
	img2 := createTestImage(100, 100, color.RGBA{255, 255, 255, 255}) // White

	result, err := detector.IsFrameChanged(img1, img2)
	if err != nil {
		t.Fatalf("IsFrameChanged failed: %v", err)
	}

	if result.SSIMScore > 0.85 {
		t.Errorf("Expected SSIM < 0.85 for very different images, got %f", result.SSIMScore)
	}

	if !result.Changed {
		t.Error("Expected very different images to be marked as changed")
	}
}

// TestPHashIdentical tests that identical images have pHash distance 0
func TestPHashIdentical(t *testing.T) {
	logger := zap.NewNop()
	detector := NewSimilarityDetector(logger)

	img := createTestImage(100, 100, color.RGBA{128, 128, 128, 255})

	result, err := detector.IsFrameChanged(img, img)
	if err != nil {
		t.Fatalf("IsFrameChanged failed: %v", err)
	}

	if result.PHashDistance != 0 {
		t.Errorf("Expected pHash distance 0 for identical images, got %d", result.PHashDistance)
	}
}

// TestPHashDifferent tests that very different images have pHash distance > 10
func TestPHashDifferent(t *testing.T) {
	logger := zap.NewNop()
	detector := NewSimilarityDetector(logger)

	img1 := createCheckerboard(100, 100, color.RGBA{0, 0, 0, 255}, color.RGBA{255, 255, 255, 255})
	img2 := createTestImage(100, 100, color.RGBA{128, 128, 128, 255})

	result, err := detector.IsFrameChanged(img1, img2)
	if err != nil {
		t.Fatalf("IsFrameChanged failed: %v", err)
	}

	if result.PHashDistance <= 10 {
		t.Errorf("Expected pHash distance > 10 for very different images, got %d", result.PHashDistance)
	}
}

// TestEdgeChangeRateIdentical tests that identical images return edge change rate 0.0
func TestEdgeChangeRateIdentical(t *testing.T) {
	logger := zap.NewNop()
	detector := NewSimilarityDetector(logger)

	img := createTestImage(100, 100, color.RGBA{128, 128, 128, 255})

	result, err := detector.IsFrameChanged(img, img)
	if err != nil {
		t.Fatalf("IsFrameChanged failed: %v", err)
	}

	if result.EdgeChangeRate != 0.0 {
		t.Errorf("Expected edge change rate 0.0 for identical images, got %f", result.EdgeChangeRate)
	}
}

// TestIsFrameChangedORLogic tests that OR logic is used - if only one metric exceeds threshold, frame is still detected as changed
func TestIsFrameChangedORLogic(t *testing.T) {
	logger := zap.NewNop()
	detector := NewSimilarityDetector(logger)

	// Create images that are different enough to trigger at least one metric
	img1 := createTestImage(100, 100, color.RGBA{0, 0, 0, 255})
	img2 := createTestImage(100, 100, color.RGBA{255, 255, 255, 255})

	result, err := detector.IsFrameChanged(img1, img2)
	if err != nil {
		t.Fatalf("IsFrameChanged failed: %v", err)
	}

	// Verify OR logic: changed if ANY metric exceeds threshold
	expectedChanged := result.SSIMScore < detector.ssimThreshold ||
		result.PHashDistance > detector.phashThreshold ||
		result.EdgeChangeRate > detector.edgeThreshold

	if result.Changed != expectedChanged {
		t.Errorf("OR logic failed: expected Changed=%v, got %v (SSIM=%f, PHash=%d, Edge=%f)",
			expectedChanged, result.Changed, result.SSIMScore, result.PHashDistance, result.EdgeChangeRate)
	}
}

// Helper function to create a solid color test image
func createTestImage(width, height int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

// Helper function to create a checkerboard pattern
func createCheckerboard(width, height int, c1, c2 color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	tileSize := 10
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if ((x/tileSize)+(y/tileSize))%2 == 0 {
				img.Set(x, y, c1)
			} else {
				img.Set(x, y, c2)
			}
		}
	}
	return img
}
