@echo off
REM 视频会议录制系统 V2.0 - 多平台构建脚本（Windows / cmd.exe）
REM
REM 用法：
REM   scripts\build.bat                       当前平台
REM   scripts\build.bat all                   全部 4 个目标
REM   scripts\build.bat windows/amd64         指定目标
REM   scripts\build.bat windows/amd64,linux/amd64  多个目标
REM   scripts\build.bat --no-frontend all     跳过前端
REM
REM 目标格式: <os>/<arch>  支持：windows/amd64, windows/arm64, linux/amd64, linux/arm64
REM 输出目录: bin\

setlocal enabledelayedexpansion

REM ---- 颜色（Windows 10+ ANSI） ----
for /F "tokens=*" %%i in ('echo prompt $E ^| cmd') do set "ESC=%%i"
set "RED=%ESC%[0;31m"
set "GREEN=%ESC%[0;32m"
set "YELLOW=%ESC%[1;33m"
set "BLUE=%ESC%[0;34m"
set "NC=%ESC%[0m"

REM ---- 路径 ----
set "SCRIPT_DIR=%~dp0"
set "PROJECT_ROOT=%SCRIPT_DIR%.."
set "OUTPUT_DIR=%PROJECT_ROOT%\bin"
set "EMBED_DIR=%PROJECT_ROOT%\internal\frontend\dist"
set "FRONTEND_DIST=%PROJECT_ROOT%\frontend\dist"

cd /d "%PROJECT_ROOT%"

REM ---- 参数解析 ----
set "BUILD_FRONTEND=1"
set "TARGETS="

:parse_args
if "%~1"=="" goto :after_args
if /I "%~1"=="--no-frontend" (
    set "BUILD_FRONTEND=0"
    shift
    goto :parse_args
)
if /I "%~1"=="--help" goto :print_help
if /I "%~1"=="-h" goto :print_help
if /I "%~1"=="all" (
    set "TARGETS=windows/amd64 windows/arm64 linux/amd64 linux/arm64"
    shift
    goto :parse_args
)
if "%TARGETS%"=="" (
    set "TARGETS=%~1"
) else (
    set "TARGETS=!TARGETS! %~1"
)
shift
goto :parse_args

:print_help
echo Usage: build.bat [all^|os/arch ...] [--no-frontend]
echo   all                 编译 windows/amd64, windows/arm64, linux/amd64, linux/arm64
echo   os/arch             单个目标（如 windows/amd64）
echo   --no-frontend       跳过前端构建
exit /b 0

:after_args
if "%TARGETS%"=="" (
    REM 默认目标：windows/amd64
    set "TARGETS=windows/amd64"
)

REM ---- 工具链检查 ----
echo %BLUE%== 工具链检查 ==%NC%
where go >nul 2>&1
if errorlevel 1 (
    echo %RED%未找到 go%NC%
    exit /b 1
)
echo   go:
go version
if "%BUILD_FRONTEND%"=="1" (
    where node >nul 2>&1
    if errorlevel 1 (
        echo %RED%未找到 node%NC%
        exit /b 1
    )
    where npm >nul 2>&1
    if errorlevel 1 (
        echo %RED%未找到 npm%NC%
        exit /b 1
    )
    echo   node:
    node --version
    echo   npm:
    npm --version
)
echo.

REM ---- 前端构建 ----
if "%BUILD_FRONTEND%"=="1" (
    echo %BLUE%== 前端构建 ==%NC%
    pushd "%PROJECT_ROOT%\frontend"
    call npm run build
    if errorlevel 1 (
        popd
        echo %RED%前端构建失败%NC%
        exit /b 1
    )
    popd

    echo %BLUE%== 复制前端到 embed 目录 ==%NC%
    if exist "%EMBED_DIR%" rmdir /s /q "%EMBED_DIR%"
    mkdir "%EMBED_DIR%"
    xcopy /e /i /y /q "%FRONTEND_DIST%\*" "%EMBED_DIR%\" >nul
    echo   已复制到 %EMBED_DIR%
    echo.
)

if not exist "%EMBED_DIR%\index.html" (
    echo %YELLOW%[警告] %EMBED_DIR%\index.html 不存在，编译出的二进制将不包含前端资源%NC%
)

REM ---- 创建输出目录 ----
if not exist "%OUTPUT_DIR%" mkdir "%OUTPUT_DIR%"

REM ---- 后端构建 ----
echo %BLUE%== 后端构建 ==%NC%
for %%T in (%TARGETS%) do (
    call :build_one %%T
)
echo.

REM ---- 复制 config.yaml 到输出目录（仅 windows 目标）----
echo %TARGETS% | findstr /I "windows" >nul
if not errorlevel 1 (
    if exist "%PROJECT_ROOT%\config.yaml" (
        copy /Y "%PROJECT_ROOT%\config.yaml" "%OUTPUT_DIR%\" >nul
        echo %GREEN%已复制 config.yaml 到 %OUTPUT_DIR%%NC%
    )
)

REM ---- 总结 ----
echo.
echo %GREEN%=========================================%NC%
echo %GREEN%  构建完成%NC%
echo %GREEN%=========================================%NC%
echo   输出目录: %OUTPUT_DIR%
dir /b "%OUTPUT_DIR%\record-v2-*"
echo.
echo 部署时：将对应平台的二进制（连同 ffmpeg / ffprobe）复制到目标服务器后直接运行。
endlocal
exit /b 0

REM ====================================================================
:build_one
setlocal enabledelayedexpansion
set "TARGET=%~1"
for /F "tokens=1,2 delims=/" %%a in ("%TARGET%") do (
    set "GOOS=%%a"
    set "GOARCH=%%b"
)
set "EXT="
if /I "%GOOS%"=="windows" set "EXT=.exe"
set "OUT=%OUTPUT_DIR%\record-v2-%GOOS%-%GOARCH%%EXT%"

echo %BLUE%^>^> 构建 %TARGET% -^> record-v2-%GOOS%-%GOARCH%%EXT%%NC%
set GOOS=%GOOS%
set GOARCH=%GOARCH%
set CGO_ENABLED=0
go build -trimpath -ldflags "-s -w" -o "%OUT%" .\cmd\server
if errorlevel 1 (
    echo %RED%构建 %TARGET% 失败%NC%
    exit /b 1
)
echo   %GREEN%✓ %OUT%%NC%
endlocal
exit /b 0
