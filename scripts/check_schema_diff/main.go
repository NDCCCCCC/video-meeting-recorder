package main

import (
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("../../data/record.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	// 获取 users 表的实际列
	type ColumnInfo struct {
		Cid      int
		Name     string
		Type     string
		NotNull  int
		DfltValue string
		Pk       int
	}
	
	var columns []ColumnInfo
	db.Raw("PRAGMA table_info(users)").Scan(&columns)
	
	fmt.Println("=== 实际数据库 users 表结构 ===")
	for _, col := range columns {
		notNull := ""
		if col.NotNull == 1 {
			notNull = "NOT NULL"
		}
		pk := ""
		if col.Pk == 1 {
			pk = "PRIMARY KEY"
		}
		fmt.Printf("  %s: %s %s %s\n", col.Name, col.Type, notNull, pk)
	}
}
