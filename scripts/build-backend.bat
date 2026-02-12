@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

:: 快速构建后端脚本（假设前端已构建）
:: 使用前请先运行: cd frontend ^&^& npm run build

set PROJECT_ROOT=%~dp0
set PROJECT_ROOT=%PROJECT_ROOT:~0,-1%
cd /d "%PROJECT_ROOT%"

echo ========================================
echo   快速构建后端
echo ========================================
echo.

:: 检查前端构建文件
if not exist "%PROJECT_ROOT%\frontend\dist\index.html" (
    echo [错误] 前端构建文件不存在
    echo 请先运行: cd frontend ^&^& npm run build
    pause
    exit /b 1
)

:: 准备嵌入文件
echo [1/3] 准备嵌入文件...
set EMBED_DIR=%PROJECT_ROOT%\internal\frontend\dist
if exist "%EMBED_DIR%" rmdir /s /q "%EMBED_DIR%"
mkdir "%EMBED_DIR%"
xcopy /e /i /y "%PROJECT_ROOT%\frontend\dist" "%EMBED_DIR%" >nul
echo 完成

:: 构建后端
echo [2/3] 构建后端...
set LDFLAGS=-s -w
set BINARY_NAME=record-v2.exe
set OUTPUT_DIR=%PROJECT_ROOT%\bin
if not exist "%OUTPUT_DIR%" mkdir "%OUTPUT_DIR%"
go build -ldflags "!LDFLAGS!" -o "%OUTPUT_DIR%\%BINARY_NAME%" "%PROJECT_ROOT%\cmd\server"
echo 完成

:: 复制配置
echo [3/3] 复制配置文件...
if exist "%PROJECT_ROOT%\configs\config.yaml" (
    copy /Y "%PROJECT_ROOT%\configs\config.yaml" "%OUTPUT_DIR%\" >nul
)
echo 完成

echo.
echo 构建完成！输出目录: %OUTPUT_DIR%
pause
