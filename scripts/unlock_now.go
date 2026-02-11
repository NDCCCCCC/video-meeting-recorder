//go:build ignore
package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	// 使用 modernc.org/sqlite 纯 Go 驱动
	db, err := sql.Open("sqlite", "data/record.db")
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}
	defer db.Close()

	// 查看被锁定的配置
	rows, err := db.Query("SELECT id, name, is_locked, locked_by FROM huawei_configs WHERE is_locked = 1")
	if err != nil {
		log.Fatal("查询配置失败:", err)
	}
	defer rows.Close()

	fmt.Println("被锁定的配置:")
	var ids []int
	for rows.Next() {
		var id int
		var name string
		var isLocked bool
		var lockedBy sql.NullInt64
		if err := rows.Scan(&id, &name, &isLocked, &lockedBy); err != nil {
			log.Fatal("扫描行失败:", err)
		}
		fmt.Printf("  - ID: %d, Name: %s\n", id, name)
		ids = append(ids, id)
	}

	// 解锁所有被锁定的配置
	result, err := db.Exec("UPDATE huawei_configs SET is_locked = 0, locked_by = NULL WHERE is_locked = 1")
	if err != nil {
		log.Fatal("解锁失败:", err)
	}

	affected, _ := result.RowsAffected()
	fmt.Printf("\n成功解锁 %d 个配置\n", affected)
	fmt.Println("所有终端已解锁")
}
