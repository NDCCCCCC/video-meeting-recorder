package main

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	fmt.Println("尝试解锁数据库...")

	db, err := sql.Open("sqlite3", "data/record.db?mode=rw")
	if err != nil {
		fmt.Printf("无法打开数据库: %v\n", err)
		return
	}
	defer db.Close()

	// 执行 checkpoint 来完成 WAL 事务
	_, err = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	if err != nil {
		fmt.Printf("Checkpoint 失败: %v\n", err)
	}

	// 尝试简单的查询来测试连接
	var result int
	err = db.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		fmt.Printf("查询测试失败: %v\n", err)
		return
	}

	fmt.Println("✓ 数据库已解锁")
}
