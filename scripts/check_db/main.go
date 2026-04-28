package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "D:/CODE/ClaudeCode/record_V2/data/record.db?mode=rw")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 检查 SQLite 版本
	var version string
	err = db.QueryRow("SELECT sqlite_version()").Scan(&version)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("SQLite 版本: %s\n", version)

	// 检查 users 表 schema
	rows, err := db.Query("PRAGMA table_info(users)")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("\n当前 users 表结构:")
	fmt.Println("────────────────────────────────────────────────────────────────────────")
	fmt.Printf("%-5s %-20s %-20s %-10s %-10s\n", "CID", "Column", "Type", "NotNull", "PrimaryKey")
	fmt.Println("────────────────────────────────────────────────────────────────────────")

	columns := make(map[string]struct {
		cid       int
		name      string
		typ       string
		notnull   int
		pk        int
		dfltValue string
	})

	for rows.Next() {
		var cid int
		var name, typ string
		var notnull int
		var dfltValue sql.NullString
		var pk int

		err = rows.Scan(&cid, &name, &typ, &notnull, &dfltValue, &pk)
		if err != nil {
			log.Fatal(err)
		}

		columns[name] = struct {
			cid       int
			name      string
			typ       string
			notnull   int
			pk        int
			dfltValue string
		}{cid, name, typ, notnull, pk, dfltValue.String}

		notNullStr := "YES"
		if notnull == 1 {
			notNullStr = "NO"
		}
		pkStr := "NO"
		if pk == 1 {
			pkStr = "YES"
		}
		fmt.Printf("%-5d %-20s %-20s %-10s %-10s\n", cid, name, typ, notNullStr, pkStr)
	}

	fmt.Println("────────────────────────────────────────────────────────────────────────")

	// 检查 role_id 列
	if col, ok := columns["role_id"]; ok {
		fmt.Printf("\n⚠️  检测到 role_id 列存在:\n")
		fmt.Printf("   类型: %s\n", col.typ)
		fmt.Printf("   NotNull: %d\n", col.notnull)

		// 检查版本是否支持 DROP COLUMN (>= 3.35.0)
		major, minor, patch := 0, 0, 0
		fmt.Sscanf(version, "%d.%d.%d", &major, &minor, &patch)
		supportsDrop := (major > 3) || (major == 3 && minor >= 35)

		if supportsDrop {
			fmt.Println("\n✅ SQLite 版本支持 DROP COLUMN")

			// 使用事务删除 role_id 列
			tx, err := db.Begin()
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("\n正在删除 role_id 列...")
			_, err = tx.Exec("ALTER TABLE users DROP COLUMN role_id")
			if err != nil {
				tx.Rollback()
				log.Fatal("❌ 删除列失败: ", err)
			}

			err = tx.Commit()
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("✅ role_id 列已成功删除")

			// 验证
			fmt.Println("\n验证结果:")
			rows2, err := db.Query("PRAGMA table_info(users)")
			if err != nil {
				log.Fatal(err)
			}
			defer rows2.Close()

			found := false
			for rows2.Next() {
				var cid int
				var name, typ string
				var notnull int
				var dfltValue sql.NullString
				var pk int

				err = rows2.Scan(&cid, &name, &typ, &notnull, &dfltValue, &pk)
				if err != nil {
					log.Fatal(err)
				}

				if name == "role_id" {
					found = true
					break
				}
			}

			if found {
				fmt.Println("❌ role_id 列仍然存在")
			} else {
				fmt.Println("✅ 确认 role_id 列已删除")
			}
		} else {
			fmt.Printf("❌ SQLite 版本 %s 不支持 DROP COLUMN (需要 >= 3.35.0)\n", version)
		}
	} else {
		fmt.Println("\n✅ role_id 列不存在，无需处理")
	}

	// 检查 users_roles 表是否存在
	fmt.Println("\n检查 users_roles 表:")
	var tableCount int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users_roles'").Scan(&tableCount)
	if err != nil {
		log.Fatal(err)
	}

	if tableCount > 0 {
		fmt.Println("✅ users_roles 表存在")

		// 检查表结构
		rows3, err := db.Query("PRAGMA table_info(users_roles)")
		if err != nil {
			log.Fatal(err)
		}
		defer rows3.Close()

		fmt.Println("\nusers_roles 表结构:")
		fmt.Println("────────────────────────────────────────────────────────────────────────")
		fmt.Printf("%-5s %-20s %-20s\n", "CID", "Column", "Type")
		fmt.Println("────────────────────────────────────────────────────────────────────────")

		for rows3.Next() {
			var cid int
			var name, typ string
			var notnull int
			var dfltValue sql.NullString
			var pk int

			err = rows3.Scan(&cid, &name, &typ, &notnull, &dfltValue, &pk)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("%-5d %-20s %-20s\n", cid, name, typ)
		}
		fmt.Println("────────────────────────────────────────────────────────────────────────")
	} else {
		fmt.Println("❌ users_roles 表不存在")
	}
}
