// gen_sm4_key.go — 生成 SM4 密钥并替换配置文件中的 sm4_secret
//
// 用法:
//
//	go run scripts/gen_sm4_key.go                  # 替换当前目录的 config.yaml
//	go run scripts/gen_sm4_key.go -f /path/to.yaml # 指定配置文件路径
//	go run scripts/gen_sm4_key.go -dry-run         # 仅显示新密钥，不修改文件
package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func main() {
	configPath := flag.String("f", "config.yaml", "配置文件路径")
	dryRun := flag.Bool("dry-run", false, "仅显示新密钥，不修改文件")
	flag.Parse()

	// 生成 16 字节随机密钥，Base64 编码
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		fmt.Fprintf(os.Stderr, "生成随机密钥失败: %v\n", err)
		os.Exit(1)
	}
	encoded := base64.StdEncoding.EncodeToString(key)

	fmt.Printf("新 SM4 密钥: %s\n", encoded)

	if *dryRun {
		fmt.Println("(dry-run 模式，未修改文件)")
		return
	}

	// 读取配置文件
	data, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取配置文件失败: %v\n", err)
		os.Exit(1)
	}

	content := string(data)

	// 匹配 sm4_secret 行并替换值
	// 支持 sm4_secret: "xxx" 和 sm4_secret: xxx 两种格式
	re := regexp.MustCompile(`(sm4_secret:\s*)"([^"]*)"`)
	if !re.MatchString(content) {
		fmt.Fprintf(os.Stderr, "在 %s 中未找到 sm4_secret 配置项\n", *configPath)
		os.Exit(1)
	}

	newContent := re.ReplaceAllString(content, fmt.Sprintf(`${1}"%s"`, encoded))

	if newContent == content {
		fmt.Fprintln(os.Stderr, "替换失败：内容未变化")
		os.Exit(1)
	}

	// 写回文件
	if err := os.WriteFile(*configPath, []byte(newContent), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "写入配置文件失败: %v\n", err)
		os.Exit(1)
	}

	// 显示变更摘要
	oldLine := ""
	newLine := ""
	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(line, "sm4_secret:") {
			oldLine = strings.TrimSpace(line)
			break
		}
	}
	for _, line := range strings.Split(newContent, "\n") {
		if strings.Contains(line, "sm4_secret:") {
			newLine = strings.TrimSpace(line)
			break
		}
	}

	fmt.Printf("已更新 %s\n", *configPath)
	fmt.Printf("  旧: %s\n", oldLine)
	fmt.Printf("  新: %s\n", newLine)
}
