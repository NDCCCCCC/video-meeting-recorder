// Standalone verification script for the disk/memory TODO fix in
// internal/services/dashboard_service.go.
//
// Calls DashboardService.GetDashboardStats directly with an in-memory
// sqlite (AuditLog migrated so the error_count / api_calls paths don't
// fail). Disk and memory percentages come from gopsutil and must be
// REAL (non-zero on any live system with a mounted filesystem).
//
// Usage: go run scripts/check_dashboard_stats/main.go
// Exits 0 if both disk_usage_percent and memory_usage_percent are > 0.

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// in-memory sqlite — no dependency on project db file
	sqlDB, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		log.Fatalf("sql.Open: %v", err)
	}
	defer sqlDB.Close()

	db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		log.Fatalf("gorm.Open: %v", err)
	}
	if err := db.AutoMigrate(
		&models.VideoRecordingTask{},
		&models.VideoFile{},
		&models.PPTFile{},
		&models.UploadedFile{},
		&models.AuditLog{},
	); err != nil {
		log.Fatalf("AutoMigrate: %v", err)
	}

	svc := services.NewDashboardService(db, logger)
	stats, err := svc.GetDashboardStats(context.Background())
	if err != nil {
		log.Fatalf("GetDashboardStats: %v", err)
	}

	b, _ := json.MarshalIndent(stats.SystemStats, "", "  ")
	fmt.Println("SystemStats (live disk/mem via gopsutil):")
	fmt.Println(string(b))

	if stats.SystemStats.DiskUsagePercent <= 0 && stats.SystemStats.MemoryUsagePercent <= 0 {
		fmt.Fprintln(os.Stderr, "\nFAIL: both disk and memory usage are 0 — gopsutil did not populate")
		os.Exit(1)
	}
	fmt.Printf("\nOK disk=%.1f%%  memory=%.1f%%\n", stats.SystemStats.DiskUsagePercent, stats.SystemStats.MemoryUsagePercent)
}
