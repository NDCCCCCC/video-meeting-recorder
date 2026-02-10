package main

import (
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "data/record.db")
	if err != nil {
		fmt.Printf("连接数据库失败: %v\n", err)
		return
	}
	defer db.Close()

	// 查看表结构
	rows, err := db.Query("PRAGMA table_info(huawei_configs)")
	if err != nil {
		fmt.Printf("查询表结构失败: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Println("huawei_configs 表结构:")
	fmt.Println("列名 | 类型 | 是否非空 | 默认值 | 主键")
	fmt.Println("--- | --- | --- | --- | ---")

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dvalue interface{}

		err = rows.Scan(&cid, &name, &ctype, &notnull, &dvalue, &pk)
		if err != nil {
			continue
		}

		defaultStr := "NULL"
		if dvalue != nil {
			defaultStr = fmt.Sprintf("%v", dvalue)
		}

		fmt.Printf("%s | %s | %v | %s | %v\n", name, ctype, notnull == 1, defaultStr, pk == 1)
	}

	// 查看当前数据
	fmt.Println("\n当前华为配置数据:")
	rows2, _ := db.Query("SELECT * FROM huawei_configs")
	cols, _ := rows2.Columns()
	for _, col := range cols {
		fmt.Printf("%s | ", col)
	}
	fmt.Println()

	for rows2.Next() {
		vals := make([]interface{}, len(cols))
		valPtrs := make([]interface{}, len(cols))
		for i := range vals {
			valPtrs[i] = &vals[i]
		}
		rows2.Scan(valPtrs...)

		for _, val := range vals {
			if val == nil {
				fmt.Printf("NULL | ")
			} else {
				fmt.Printf("%v | ", val)
			}
		}
		fmt.Println()
	}
	rows2.Close()
}
