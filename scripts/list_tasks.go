//go:build ignore
package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "data/record.db")
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}
	defer db.Close()

	fmt.Println("=== 所有任务状态统计 ===")
	var total, pending, scheduled, executing, completed, failed int
	db.QueryRow("SELECT COUNT(*) FROM video_recording_tasks").Scan(&total)
	db.QueryRow("SELECT COUNT(*) FROM video_recording_tasks WHERE status = 'pending'").Scan(&pending)
	db.QueryRow("SELECT COUNT(*) FROM video_recording_tasks WHERE status = 'scheduled'").Scan(&scheduled)
	db.QueryRow("SELECT COUNT(*) FROM video_recording_tasks WHERE status = 'executing'").Scan(&executing)
	db.QueryRow("SELECT COUNT(*) FROM video_recording_tasks WHERE status = 'completed'").Scan(&completed)
	db.QueryRow("SELECT COUNT(*) FROM video_recording_tasks WHERE status = 'failed'").Scan(&failed)

	fmt.Printf("总任务数: %d\n", total)
	fmt.Printf("  - pending (待处理): %d\n", pending)
	fmt.Printf("  - scheduled (已调度): %d\n", scheduled)
	fmt.Printf("  - executing (执行中): %d\n", executing)
	fmt.Printf("  - completed (已完成): %d\n", completed)
	fmt.Printf("  - failed (失败): %d\n", failed)

	fmt.Println("\n=== 最近的任务 (前10条) ===")
	rows, err := db.Query(`
		SELECT id, name, status, start_time, end_time
		FROM video_recording_tasks
		ORDER BY id DESC
		LIMIT 10
	`)
	if err != nil {
		log.Fatal("查询失败:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name, status string
		var startTime, endTime sql.NullTime
		if err := rows.Scan(&id, &name, &status, &startTime, &endTime); err != nil {
			log.Fatal("扫描失败:", err)
		}
		fmt.Printf("ID: %d, 名称: %s, 状态: %s\n", id, name, status)
		if startTime.Valid {
			fmt.Printf("   开始时间: %s\n", startTime.Time.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("   开始时间: (未设置)\n")
		}
		if endTime.Valid {
			fmt.Printf("   结束时间: %s\n", endTime.Time.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("   结束时间: (未设置)\n")
		}
	}

	fmt.Println("\n=== pending 和 scheduled 状态的任务 ===")
	rows2, err := db.Query(`
		SELECT id, name, status, start_time, end_time
		FROM video_recording_tasks
		WHERE status IN ('pending', 'scheduled')
		ORDER BY id ASC
	`)
	if err != nil {
		log.Fatal("查询失败:", err)
	}
	defer rows2.Close()

	count := 0
	for rows2.Next() {
		var id int
		var name, status string
		var startTime, endTime sql.NullTime
		if err := rows2.Scan(&id, &name, &status, &startTime, &endTime); err != nil {
			log.Fatal("扫描失败:", err)
		}
		fmt.Printf("ID: %d, 名称: %s, 状态: %s\n", id, name, status)
		if startTime.Valid {
			fmt.Printf("   开始时间: %s (UTC)\n", startTime.Time.Format("2006-01-02 15:04:05"))
			fmt.Printf("   距离现在: %s\n", startTime.Time.Sub(sql.NullTime{Time: sql.NullTime{}.Time}.Time))
		} else {
			fmt.Printf("   开始时间: (未设置)\n")
		}
		if endTime.Valid {
			fmt.Printf("   结束时间: %s (UTC)\n", endTime.Time.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("   结束时间: (未设置)\n")
		}
		count++
	}

	if count == 0 {
		fmt.Println("没有 pending 或 scheduled 状态的任务")
	}
}
