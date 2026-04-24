@echo off
REM 生成自签名证书脚本（Windows）
echo ========================================
echo  Record V2 - 自签名证书生成工具
echo ========================================
echo.

cd /d "%~dp0.."

echo [1/2] 检查 Go 环境...
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo 错误: 未找到 Go 环境
    echo 请先安装 Go: https://golang.org/dl/
    pause
    exit /b 1
)

echo [2/2] 生成自签名证书...
go run scripts/generate-cert.go

if %errorlevel% equ 0 (
    echo.
    echo ========================================
    echo  证书生成成功!
    echo ========================================
    echo.
    echo 下一步:
    echo 1. 证书已保存在 ./certs/ 目录
    echo 2. 启动服务器: go run cmd/server/main.go
    echo 3. 访问: https://localhost:8443
    echo.
    echo 浏览器安全警告处理:
    echo - Chrome/Edge: 点击「高级」→「继续访问」
    echo - Firefox: 点击「高级」→「接受风险并继续」
    echo.
)

pause
