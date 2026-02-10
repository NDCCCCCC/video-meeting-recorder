package main

import (
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

type HuaweiConfig struct {
	ID       uint
	Server   string
	Port     int
	Username string
	Password string
	Https    bool
}

func main() {
	db, err := gorm.Open(sqlite.Open("data/record.db"), &gorm.Config{})
	if err != nil {
		fmt.Printf("连接数据库失败: %v\n", err)
		return
	}

	// 更新华为配置为HTTPS 443端口
	result := db.Exec("UPDATE huawei_configs SET server=?, port=?, username=?, password=?, https=? WHERE id=?",
		"10.62.10.3", 443, "api", "Hubei@1992", true, 1)

	if result.Error != nil {
		fmt.Printf("更新失败: %v\n", result.Error)
		return
	}

	fmt.Printf("更新成功，影响行数: %d\n", result.RowsAffected)

	// 验证更新
	var config HuaweiConfig
	db.First(&config, 1)
	fmt.Printf("当前配置: server=%s, port=%d, https=%v, username=%s\n",
		config.Server, config.Port, config.Https, config.Username)
}
