package services

import (
	"image"
	"math"

	"github.com/corona10/goimagehash"
	"github.com/nfnt/resize"
	"go.uber.org/zap"
	"golang.org/x/image/draw"
)

// SimilarityDetector 图像相似度检测器
type SimilarityDetector struct {
	logger         *zap.Logger
	ssimThreshold  float64 // SSIM阈值，低于此值认为图像变化
	phashThreshold int     // pHash距离阈值，高于此值认为图像变化
	edgeThreshold  float64 // 边缘变化率阈值，高于此值认为图像变化
}

// DetectionResult 检测结果
type DetectionResult struct {
	Changed        bool    // 是否检测到变化
	SSIMScore      float64 // SSIM分数 (0.0-1.0)
	PHashDistance  int     // pHash汉明距离
	EdgeChangeRate float64 // 边缘变化率 (0.0-1.0)
}

// NewSimilarityDetector 创建相似度检测器
func NewSimilarityDetector(logger *zap.Logger) *SimilarityDetector {
	return &SimilarityDetector{
		logger:         logger,
		ssimThreshold:  0.85,  // D-08: SSIM阈值
		phashThreshold: 10,    // D-08: pHash距离阈值
		edgeThreshold:  0.25,  // D-08: 边缘变化率阈值
	}
}

// IsFrameChanged 检测两帧图像是否发生变化 (OR逻辑: 任一方法检测到变化即认为变化)
func (d *SimilarityDetector) IsFrameChanged(prevImg, currImg image.Image) (*DetectionResult, error) {
	// 降采样到720p以提高性能 (D-04)
	prevImg = d.downscaleTo720p(prevImg)
	currImg = d.downscaleTo720p(currImg)

	// 计算SSIM
	ssimScore := d.calculateSSIM(prevImg, currImg)

	// 计算pHash距离
	phashDist, err := d.calculatePHashDistance(prevImg, currImg)
	if err != nil {
		d.logger.Warn("pHash计算失败，将使用距离0", zap.Error(err))
		phashDist = 0
	}

	// 计算边缘变化率
	edgeRate := d.calculateEdgeChangeRate(prevImg, currImg)

	// OR逻辑: 任一指标超过阈值即认为发生变化 (D-07)
	changed := ssimScore < d.ssimThreshold || phashDist > d.phashThreshold || edgeRate > d.edgeThreshold

	result := &DetectionResult{
		Changed:        changed,
		SSIMScore:      ssimScore,
		PHashDistance:  phashDist,
		EdgeChangeRate: edgeRate,
	}

	d.logger.Debug("帧相似度检测",
		zap.Bool("changed", changed),
		zap.Float64("ssim", ssimScore),
		zap.Int("phash_dist", phashDist),
		zap.Float64("edge_rate", edgeRate),
	)

	return result, nil
}

// calculateSSIM 计算结构相似性指数
func (d *SimilarityDetector) calculateSSIM(img1, img2 image.Image) float64 {
	// 转换为灰度图
	gray1 := d.toGrayscale(img1)
	gray2 := d.toGrayscale(img2)

	bounds := gray1.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// SSIM参数
	C1 := (0.01 * 255) * (0.01 * 255)
	C2 := (0.03 * 255) * (0.03 * 255)

	// 使用8x8滑动窗口计算SSIM
	windowSize := 8
	ssimSum := 0.0
	count := 0

	for y := 0; y <= height-windowSize; y += windowSize {
		for x := 0; x <= width-windowSize; x += windowSize {
			// 计算窗口内的均值
			mu1, mu2 := d.calculateWindowMean(gray1, gray2, x, y, windowSize)

			// 计算窗口内的方差和协方差
			var1, var2, cov := d.calculateWindowVariance(gray1, gray2, x, y, windowSize, mu1, mu2)

			// SSIM公式
			numerator := (2*mu1*mu2 + C1) * (2*cov + C2)
			denominator := (mu1*mu1 + mu2*mu2 + C1) * (var1 + var2 + C2)

			if denominator > 0 {
				ssimSum += numerator / denominator
				count++
			}
		}
	}

	if count == 0 {
		return 0.0
	}

	return ssimSum / float64(count)
}

// calculatePHashDistance 计算感知哈希距离
func (d *SimilarityDetector) calculatePHashDistance(img1, img2 image.Image) (int, error) {
	hash1, err := goimagehash.PerceptionHash(img1)
	if err != nil {
		return 0, err
	}

	hash2, err := goimagehash.PerceptionHash(img2)
	if err != nil {
		return 0, err
	}

	distance, err := hash1.Distance(hash2)
	if err != nil {
		return 0, err
	}
	return distance, nil
}

// calculateEdgeChangeRate 计算边缘变化率
func (d *SimilarityDetector) calculateEdgeChangeRate(img1, img2 image.Image) float64 {
	// 转换为灰度图
	gray1 := d.toGrayscale(img1)
	gray2 := d.toGrayscale(img2)

	// 计算边缘像素数量
	edgePixels1 := d.countEdgePixels(gray1)
	edgePixels2 := d.countEdgePixels(gray2)

	// 计算变化率
	maxPixels := math.Max(float64(edgePixels1), float64(edgePixels2))
	if maxPixels == 0 {
		return 0.0
	}

	changeRate := math.Abs(float64(edgePixels1)-float64(edgePixels2)) / maxPixels
	return changeRate
}

// countEdgePixels 使用Sobel算子计算边缘像素数量
func (d *SimilarityDetector) countEdgePixels(img *image.Gray) int {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	edgeCount := 0
	threshold := 128

	// Sobel算子核
	sobelX := [][]int{{-1, 0, 1}, {-2, 0, 2}, {-1, 0, 1}}
	sobelY := [][]int{{-1, -2, -1}, {0, 0, 0}, {1, 2, 1}}

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			// 应用Sobel算子
			gx := 0
			gy := 0
			for ky := -1; ky <= 1; ky++ {
				for kx := -1; kx <= 1; kx++ {
					gray := float64(img.GrayAt(x+kx, y+ky).Y)
					gx += int(gray) * sobelX[ky+1][kx+1]
					gy += int(gray) * sobelY[ky+1][kx+1]
				}
			}

			// 计算梯度幅值
			magnitude := math.Sqrt(float64(gx*gx+gy*gy))
			if magnitude > float64(threshold) {
				edgeCount++
			}
		}
	}

	return edgeCount
}

// downscaleTo720p 降采样到720p (D-04)
func (d *SimilarityDetector) downscaleTo720p(img image.Image) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 如果已经小于等于720p，直接返回
	if width <= 1280 && height <= 720 {
		return img
	}

	// 计算缩放比例，保持宽高比
	ratio := math.Min(1280.0/float64(width), 720.0/float64(height))
	newWidth := uint(math.Round(float64(width) * ratio))
	newHeight := uint(math.Round(float64(height) * ratio))

	// 使用高质量缩放算法
	resized := resize.Resize(newWidth, newHeight, img, resize.Lanczos3)
	return resized
}

// toGrayscale 转换为灰度图
func (d *SimilarityDetector) toGrayscale(img image.Image) *image.Gray {
	gray := image.NewGray(img.Bounds())
	draw.Draw(gray, gray.Bounds(), img, img.Bounds().Min, draw.Src)
	return gray
}

// calculateWindowMean 计算窗口内的均值
func (d *SimilarityDetector) calculateWindowMean(img1, img2 *image.Gray, startX, startY, size int) (float64, float64) {
	sum1 := 0.0
	sum2 := 0.0
	count := 0

	for y := startY; y < startY+size; y++ {
		for x := startX; x < startX+size; x++ {
			sum1 += float64(img1.GrayAt(x, y).Y)
			sum2 += float64(img2.GrayAt(x, y).Y)
			count++
		}
	}

	mu1 := sum1 / float64(count)
	mu2 := sum2 / float64(count)
	return mu1, mu2
}

// calculateWindowVariance 计算窗口内的方差和协方差
func (d *SimilarityDetector) calculateWindowVariance(img1, img2 *image.Gray, startX, startY, size int, mu1, mu2 float64) (float64, float64, float64) {
	variance1 := 0.0
	variance2 := 0.0
	covariance := 0.0
	count := 0

	for y := startY; y < startY+size; y++ {
		for x := startX; x < startX+size; x++ {
			val1 := float64(img1.GrayAt(x, y).Y)
			val2 := float64(img2.GrayAt(x, y).Y)

			variance1 += (val1 - mu1) * (val1 - mu1)
			variance2 += (val2 - mu2) * (val2 - mu2)
			covariance += (val1 - mu1) * (val2 - mu2)
			count++
		}
	}

	variance1 /= float64(count)
	variance2 /= float64(count)
	covariance /= float64(count)

	return variance1, variance2, covariance
}
