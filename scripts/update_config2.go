//go:build ignore
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

	// 更新华为配置为HTTPS 443端口
	result, err := db.Exec("UPDATE huawei_configs SET server=?, port=?, username=?, password=?, https=? WHERE id=?",
		"10.62.10.3", 443, "api", "Hubei@1992", true, 1)

	if err != nil {
		fmt.Printf("更新失败: %v\n", err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("更新成功，影响行数: %d\n", rowsAffected)

	// 验证更新
	var server, username, password string
	var port int
	var https bool
	err = db.QueryRow("SELECT server, port, username, password, https FROM huawei_configs WHERE id=?", 1).Scan(&server, &port, &username, &password, &https)
	if err != nil {
		fmt.Printf("查询失败: %v\n", err)
		return
	}

	fmt.Printf("当前配置: server=%s, port=%d, https=%v, username=%s, password=%s\n",
		server, port, https, username, password)
}
