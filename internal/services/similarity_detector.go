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
	Changed          bool    // 是否检测到变化
	SSIMScore        float64 // SSIM分数 (0.0-1.0)
	PHashDistance    int     // pHash汉明距离
	EdgeChangeRate   float64 // 边缘变化率 (0.0-1.0)
	IsBlackFrame     bool    // 当前帧是否为黑色/空白帧
	PrevIsBlackFrame bool    // 上一帧是否为黑色/空白帧
}

// NewSimilarityDetector 创建相似度检测器
func NewSimilarityDetector(logger *zap.Logger) *SimilarityDetector {
	return &SimilarityDetector{
		logger:         logger,
		ssimThreshold:  0.90, // 提高精度：更严格的SSIM阈值 (从0.85提高到0.90)
		phashThreshold: 5,    // 提高精度：更严格的pHash距离 (从10降低到5)
		edgeThreshold:  0.15, // 提高精度：更严格的边缘变化率 (从0.25降低到0.15)
	}
}

// IsFrameChanged 检测两帧图像是否发生变化 (OR逻辑: 任一方法检测到变化即认为变化)
func (d *SimilarityDetector) IsFrameChanged(prevImg, currImg image.Image) (*DetectionResult, error) {
	// 降采样到720p以提高性能 (D-04)
	prevImg = d.downscaleTo720p(prevImg)
	currImg = d.downscaleTo720p(currImg)

	// 检测黑色/空白帧 (Fix: 添加黑帧检测以避免误判)
	prevIsBlack := d.isBlackFrame(prevImg)
	currIsBlack := d.isBlackFrame(currImg)

	result := &DetectionResult{
		IsBlackFrame:     currIsBlack,
		PrevIsBlackFrame: prevIsBlack,
	}

	// 如果两帧都是黑色帧，认为没有变化 (Fix: 黑帧之间不应检测为变化)
	if prevIsBlack && currIsBlack {
		result.Changed = false
		result.SSIMScore = 1.0 // 完全相同
		result.PHashDistance = 0
		result.EdgeChangeRate = 0.0

		d.logger.Debug("两帧均为黑色帧，跳过相似度计算",
			zap.Bool("prev_is_black", prevIsBlack),
			zap.Bool("curr_is_black", currIsBlack),
		)
		return result, nil
	}

	// 计算SSIM
	ssimScore := d.calculateSSIM(prevImg, currImg)
	result.SSIMScore = ssimScore

	// 计算pHash距离
	phashDist, err := d.calculatePHashDistance(prevImg, currImg)
	if err != nil {
		d.logger.Warn("pHash计算失败，将使用距离0", zap.Error(err))
		phashDist = 0
	}
	result.PHashDistance = phashDist

	// 计算边缘变化率
	edgeRate := d.calculateEdgeChangeRate(prevImg, currImg)
	result.EdgeChangeRate = edgeRate

	// OR逻辑: 任一指标超过阈值即认为发生变化 (D-07)
	// Fix: 如果当前帧是黑色帧而上一帧不是，也不算变化（黑帧应该被过滤掉）
	if currIsBlack && !prevIsBlack {
		result.Changed = false
		d.logger.Debug("当前帧为黑色帧，跳过", zap.Bool("curr_is_black", currIsBlack))
		return result, nil
	}

	changed := ssimScore < d.ssimThreshold || phashDist > d.phashThreshold || edgeRate > d.edgeThreshold
	result.Changed = changed

	d.logger.Debug("帧相似度检测",
		zap.Bool("changed", changed),
		zap.Float64("ssim", ssimScore),
		zap.Int("phash_dist", phashDist),
		zap.Float64("edge_rate", edgeRate),
		zap.Bool("prev_is_black", prevIsBlack),
		zap.Bool("curr_is_black", currIsBlack),
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
			magnitude := math.Sqrt(float64(gx*gx + gy*gy))
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

// isBlackFrame 检测图像是否为黑色/空白帧
// 判断标准: 平均亮度 < 阈值 (默认15，避免误判噪声)
func (d *SimilarityDetector) isBlackFrame(img image.Image) bool {
	gray := d.toGrayscale(img)
	bounds := gray.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 计算平均亮度
	sum := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sum += int(gray.GrayAt(x, y).Y)
		}
	}

	avgBrightness := float64(sum) / float64(width*height)

	// 阈值设为15 (0-255范围)，可以捕捉非常暗的帧
	// 纯黑帧的平均亮度为0，正常视频帧通常 > 30
	const blackThreshold = 15.0
	isBlack := avgBrightness < blackThreshold

	if isBlack {
		d.logger.Debug("检测到黑色帧",
			zap.Float64("avg_brightness", avgBrightness),
			zap.Float64("threshold", blackThreshold),
		)
	}

	return isBlack
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
