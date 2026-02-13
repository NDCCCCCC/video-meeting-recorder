@echo off
REM FFmpeg 复制脚本
REM 此脚本帮助您将系统中的 ffmpeg 和 ffprobe 复制到项目的 bin 目录

setlocal

set PROJECT_ROOT=%~dp0..
set BIN_DIR=%PROJECT_ROOT%\bin

echo ========================================
echo FFmpeg 复制脚本
echo ========================================
echo.
echo 目标目录: %BIN_DIR%
echo.

REM 检查 bin 目录是否存在
if not exist "%BIN_DIR%" (
    mkdir "%BIN_DIR%"
    echo [INFO] 已创建 bin 目录
)

REM 尝试从系统 PATH 查找 ffmpeg
echo [1/2] 正在查找 ffmpeg ...
where ffmpeg >nul 2>&1
if %errorlevel% equ 0 (
    for /f "delims=" %%i in ('where ffmpeg') do (
        set FFMPEG_SRC=%%i
        goto :found_ffmpeg
    )
) else (
    echo [WARN] 在系统 PATH 中未找到 ffmpeg
    echo        请手动指定 ffmpeg.exe 的位置
    echo.
    set /p FFMPEG_SRC="请输入 ffmpeg.exe 的完整路径: "
    if not exist "%FFMPEG_SRC%" (
        echo [ERROR] 指定的文件不存在: %FFMPEG_SRC%
        pause
        exit /b 1
    )
)

:found_ffmpeg
echo [FOUND] ffmpeg 位于: %FFMPEG_SRC%
echo.

REM 获取 ffmpeg 所在目录
for %%i in ("%FFMPEG_SRC%") do set FFMPEG_DIR=%%~dpi

REM 检查 ffprobe 是否存在
set FFPROBE_SRC=%FFMPEG_DIR%ffprobe.exe
echo [2/2] 正在查找 ffprobe ...
if not exist "%FFPROBE_SRC%" (
    echo [WARN] ffprobe 未找到，跳过复制
    goto :skip_ffprobe
)
echo [FOUND] ffprobe 位于: %FFPROBE_SRC%
echo.

REM 复制文件
echo ========================================
echo 开始复制文件...
echo ========================================

echo [COPY] ffmpeg.exe -^> %BIN_DIR%\ffmpeg.exe
copy /Y "%FFMPEG_SRC%" "%BIN_DIR%\ffmpeg.exe" >nul
if %errorlevel% equ 0 (
    echo [OK] ffmpeg.exe 复制成功
) else (
    echo [ERROR] ffmpeg.exe 复制失败
    pause
    exit /b 1
)

:skip_ffprobe
if exist "%FFPROBE_SRC%" (
    echo [COPY] ffprobe.exe -^> %BIN_DIR%\ffprobe.exe
    copy /Y "%FFPROBE_SRC%" "%BIN_DIR%\ffprobe.exe" >nul
    if %errorlevel% equ 0 (
        echo [OK] ffprobe.exe 复制成功
    ) else (
        echo [ERROR] ffprobe.exe 复制失败
    )
)

echo.
echo ========================================
echo 复制完成！
echo ========================================
echo.
echo 验证安装:
echo   %BIN_DIR%\ffmpeg.exe -version
echo   %BIN_DIR%\ffprobe.exe -version
echo.

REM 验证版本
echo ffmpeg 版本信息:
"%BIN_DIR%\ffmpeg.exe" -version 2>nul | findstr /i "ffmpeg version"
if %errorlevel% equ 0 (
    echo [OK] ffmpeg 可用
) else (
    echo [WARN] ffmpeg 可能不可用
)

echo.
echo ffprobe 版本信息:
"%BIN_DIR%\ffprobe.exe" -version 2>nul | findstr /i "ffprobe version"
if %errorlevel% equ 0 (
    echo [OK] ffprobe 可用
) else (
    echo [WARN] ffprobe 可能不可用
)

echo.
echo ========================================
echo 配置说明
echo ========================================
echo.
echo 配置文件已更新为使用 .\bin\ffmpeg
echo 请确保重新编译并启动程序
echo.

pause
endlocal
