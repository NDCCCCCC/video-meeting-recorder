@echo off
REM 快速构建后端脚本（前端已构建）
REM 用法：scripts\build-backend.bat [os/arch]
REM  默认：当前平台 windows/amd64

setlocal enabledelayedexpansion
chcp 65001 >nul

for /F "tokens=*" %%i in ('echo prompt $E ^| cmd') do set "ESC=%%i"
set "BLUE=%ESC%[0;34m"
set "GREEN=%ESC%[0;32m"
set "RED=%ESC%[0;31m"
set "NC=%ESC%[0m"

set "SCRIPT_DIR=%~dp0"
set "PROJECT_ROOT=%SCRIPT_DIR%.."
set "OUTPUT_DIR=%PROJECT_ROOT%\bin"
set "EMBED_DIR=%PROJECT_ROOT%\internal\frontend\dist"
cd /d "%PROJECT_ROOT%"

REM 默认目标
if "%~1"=="" set "TARGET=windows/amd64"
if not "%~1"=="" set "TARGET=%~1"

for /F "tokens=1,2 delims=/" %%a in ("%TARGET%") do (
    set "GOOS=%%a"
    set "GOARCH=%%b"
)

REM 前端构建文件检查
if not exist "%EMBED_DIR%\index.html" (
    if not exist "%PROJECT_ROOT%\frontend\dist\index.html" (
        echo %RED%[错误] 前端构建文件不存在
        echo 请先运行：cd frontend ^&^& npm run build
        echo 或者使用 scripts\build.bat（会一并构建前端）%NC%
        pause
        exit /b 1
    )
    if not exist "%EMBED_DIR%" mkdir "%EMBED_DIR%"
    xcopy /e /i /y /q "%PROJECT_ROOT%\frontend\dist\*" "%EMBED_DIR%\" >nul
)

echo %BLUE%== 准备嵌入文件 ==%NC%
if exist "%EMBED_DIR%" rmdir /s /q "%EMBED_DIR%"
mkdir "%EMBED_DIR%"
xcopy /e /i /y /q "%PROJECT_ROOT%\frontend\dist\*" "%EMBED_DIR%\" >nul
echo 完成

echo %BLUE%== 构建后端（%TARGET%）==%NC%
if not exist "%OUTPUT_DIR%" mkdir "%OUTPUT_DIR%"
set "EXT="
if /I "%GOOS%"=="windows" set "EXT=.exe"
set "OUT=%OUTPUT_DIR%\record-v2-%GOOS%-%GOARCH%%EXT%"
set GOOS=%GOOS%
set GOARCH=%GOARCH%
set CGO_ENABLED=0
go build -trimpath -ldflags "-s -w" -o "%OUT%" .\cmd\server
if errorlevel 1 (
    echo %RED%构建失败%NC%
    pause
    exit /b 1
)
echo %GREEN%完成：%OUT%%NC%

REM 复制 config.yaml（如目标含 windows）
if /I "%GOOS%"=="windows" (
    if exist "%PROJECT_ROOT%\config.yaml" copy /Y "%PROJECT_ROOT%\config.yaml" "%OUTPUT_DIR%\" >nul
)
echo.
echo 构建完成！输出：%OUT%
endlocal
pause
exit /b 0
