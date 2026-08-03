package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
)

// FrameExtractor FFmpeg帧提取器
type FrameExtractor struct {
	ffmpegPath string
	logger     *zap.Logger
}

// ExtractedFrame 提取的帧信息
type ExtractedFrame struct {
	FilePath  string  // JPEG文件路径
	Timestamp float64 // 时间戳（秒）
	Index     int     // 帧索引
}

// NewFrameExtractor 创建帧提取器
func NewFrameExtractor(ffmpegPath string, logger *zap.Logger) *FrameExtractor {
	return &FrameExtractor{
		ffmpegPath: ffmpegPath,
		logger:     logger,
	}
}

// ExtractFrames 从视频中提取帧 (D-02: 采样率转换为fps)
func (e *FrameExtractor) ExtractFrames(ctx context.Context, videoPath, outputDir string, samplingRateSeconds float64) ([]ExtractedFrame, error) {
	// 验证输入
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("视频文件不存在: %s", videoPath)
	}

	// Validate paths to prevent command injection
	if err := e.validatePath(videoPath); err != nil {
		return nil, fmt.Errorf("invalid video path: %w", err)
	}
	if err := e.validatePath(outputDir); err != nil {
		return nil, fmt.Errorf("invalid output directory: %w", err)
	}

	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 转换采样率到FPS (D-02: 1s=1fps, 2s=0.5fps, 5s=0.2fps)
	fps := 1.0 / samplingRateSeconds

	e.logger.Info("开始提取帧",
		zap.String("video_path", videoPath),
		zap.String("output_dir", outputDir),
		zap.Float64("sampling_rate_secs", samplingRateSeconds),
		zap.Float64("fps", fps),
	)

	// 构建FFmpeg命令 (RESEARCH.md Pattern 2)
	// -y: 覆盖输出文件
	// -i: 输入文件
	// -vf fps=N: 帧率过滤器
	// -q:v 2: JPEG质量95 (D-03)
	// -vsync 0: 每一帧都独立时间戳
	args := []string{
		"-y",
		"-i", videoPath,
		"-vf", fmt.Sprintf("fps=%f", fps),
		"-q:v", "2", // JPEG质量95 (D-03)
		"-vsync", "0",
		filepath.Join(outputDir, "frame_%04d.jpg"),
	}

	e.logger.Debug("FFmpeg命令", zap.String("cmd", e.ffmpegPath), zap.Strings("args", args))

	// 执行FFmpeg
	cmd := exec.CommandContext(ctx, e.ffmpegPath, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		e.logger.Error("FFmpeg帧提取失败",
			zap.Error(err),
			zap.String("stderr", stderr.String()),
			response.SentinelField(err),
		)
		return nil, fmt.Errorf("FFmpeg帧提取失败: %w, stderr: %s", err, stderr.String())
	}

	// 扫描输出目录中的帧文件
	frames, err := e.scanOutputDir(outputDir, samplingRateSeconds)
	if err != nil {
		return nil, fmt.Errorf("扫描输出目录失败: %w", err)
	}

	e.logger.Info("帧提取完成",
		zap.Int("frame_count", len(frames)),
		zap.String("output_dir", outputDir),
	)

	return frames, nil
}

// ExtractFrameAtTimestamp 在指定时间戳提取单帧 (D-04: 原始分辨率重提取)
func (e *FrameExtractor) ExtractFrameAtTimestamp(ctx context.Context, videoPath string, timestamp float64, outputPath string) error {
	// 验证输入
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return fmt.Errorf("视频文件不存在: %s", videoPath)
	}

	// Validate paths to prevent command injection
	if err := e.validatePath(videoPath); err != nil {
		return fmt.Errorf("invalid video path: %w", err)
	}
	if err := e.validatePath(outputPath); err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}

	// 确保输出目录存在
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	e.logger.Debug("提取单帧",
		zap.String("video_path", videoPath),
		zap.Float64("timestamp", timestamp),
		zap.String("output_path", outputPath),
	)

	// 构建FFmpeg命令
	// -ss: 时间戳（在-i之前用于快速seek）
	// -i: 输入文件
	// -frames:v 1: 只提取一帧
	// -q:v 2: JPEG质量95
	args := []string{
		"-y",
		"-ss", fmt.Sprintf("%.3f", timestamp),
		"-i", videoPath,
		"-frames:v", "1",
		"-q:v", "2",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, e.ffmpegPath, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		e.logger.Error("单帧提取失败",
			zap.Error(err),
			zap.String("stderr", stderr.String()),
			response.SentinelField(err),
		)
		return fmt.Errorf("单帧提取失败: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// CreateTempDir 创建临时目录 (D-05: 唯一命名)
func (e *FrameExtractor) CreateTempDir(baseDir string, videoFileID uint) (string, error) {
	timestamp := time.Now().Unix()
	dirName := fmt.Sprintf("transcription_%d_%d", videoFileID, timestamp)
	dirPath := filepath.Join(baseDir, dirName)

	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}

	e.logger.Debug("创建临时目录", zap.String("path", dirPath))
	return dirPath, nil
}

// CleanupTempDir 清理临时目录 (D-05)
func (e *FrameExtractor) CleanupTempDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		// 清理错误是非致命的，只记录日志
		e.logger.Warn("清理临时目录失败", zap.String("dir", dir), zap.Error(err), response.SentinelField(err))
		return err
	}

	e.logger.Debug("清理临时目录", zap.String("path", dir))
	return nil
}

// scanOutputDir 扫描输出目录并返回排序后的帧列表
func (e *FrameExtractor) scanOutputDir(outputDir string, samplingRateSeconds float64) ([]ExtractedFrame, error) {
	// 匹配 frame_XXXX.jpg
	pattern := filepath.Join(outputDir, "frame_*.jpg")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return []ExtractedFrame{}, nil
	}

	// 解析并排序帧文件
	frames := make([]ExtractedFrame, 0, len(matches))
	for _, filePath := range matches {
		// 从文件名提取索引: frame_0001.jpg -> 1
		fileName := filepath.Base(filePath)
		idxStr := strings.TrimPrefix(fileName, "frame_")
		idxStr = strings.TrimSuffix(idxStr, ".jpg")

		index, err := strconv.Atoi(idxStr)
		if err != nil {
			e.logger.Warn("解析帧索引失败", zap.String("file", fileName), zap.Error(err), response.SentinelField(err))
			continue
		}

		// 计算时间戳: timestamp = index * samplingRateSeconds
		timestamp := float64(index) * samplingRateSeconds

		frames = append(frames, ExtractedFrame{
			FilePath:  filePath,
			Timestamp: timestamp,
			Index:     index,
		})
	}

	// 按索引排序
	sort.Slice(frames, func(i, j int) bool {
		return frames[i].Index < frames[j].Index
	})

	return frames, nil
}

// validatePath validates that a path is safe and doesn't contain shell metacharacters
func (e *FrameExtractor) validatePath(path string) error {
	// Check for shell metacharacters that could enable command injection
	dangerousChars := []string{"`", "$", ";", "&", "|", ">", "<", "\n", "\r"}
	for _, char := range dangerousChars {
		if strings.Contains(path, char) {
			return fmt.Errorf("path contains dangerous character: %s", char)
		}
	}

	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("cannot resolve absolute path: %w", err)
	}

	// Verify path doesn't escape to sensitive system directories
	if strings.HasPrefix(absPath, "/etc") || strings.HasPrefix(absPath, "/sys") ||
		strings.HasPrefix(absPath, "/proc") || strings.HasPrefix(absPath, "/root") {
		return fmt.Errorf("access to system directory not allowed: %s", path)
	}

	return nil
}
