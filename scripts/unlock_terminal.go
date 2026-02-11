package main

import (
	"fmt"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

// HuaweiConfig 华为配置模型
type HuaweiConfig struct {
	ID       uint   `gorm:"primaryKey"`
	Name     string `gorm:"type:varchar(200)"`
	IsLocked bool   `gorm:"default:false"`
	LockedBy *uint  `json:"locked_by,omitempty"`
}

func main() {
	// 连接数据库
	db, err := gorm.Open(sqlite.Open("data/record.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}

	// 查找所有被锁定的配置
	var configs []HuaweiConfig
	result := db.Where("is_locked = ?", true).Find(&configs)
	if result.Error != nil {
		log.Fatal("查询配置失败:", result.Error)
	}

	if len(configs) == 0 {
		fmt.Println("没有找到被锁定的华为配置")
		return
	}

	fmt.Printf("找到 %d 个被锁定的配置:\n", len(configs))
	for _, config := range configs {
		lockedBy := "未知"
		if config.LockedBy != nil {
			lockedBy = fmt.Sprintf("任务 %d", *config.LockedBy)
		}
		fmt.Printf("  - ID: %d, Name: %s, 锁定者: %s\n", config.ID, config.Name, lockedBy)
	}

	// 解锁所有被锁定的配置
	updates := map[string]interface{}{
		"is_locked": false,
		"locked_by": nil,
	}

	result = db.Model(&HuaweiConfig{}).
		Where("is_locked = ?", true).
		Updates(updates)

	if result.Error != nil {
		log.Fatal("解锁失败:", result.Error)
	}

	fmt.Printf("\n成功解锁 %d 个配置\n", result.RowsAffected)
	fmt.Println("所有终端已解锁，可以重新启动服务器")
}
