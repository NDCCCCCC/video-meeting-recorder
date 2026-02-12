@echo off
setlocal enabledelayedexpansion

echo ========================================
echo   Video Record System V2.0 - Build
echo ========================================
echo.

:: Get project root directory (parent of scripts folder)
for %%i in ("%~dp0..") do set PROJECT_ROOT=%%~fi

:: Check Node.js
echo [1/6] Checking Node.js...
where node >nul 2>&1
if errorlevel 1 (
    echo ERROR: Node.js not found
    pause
    exit /b 1
)
node --version
echo.

:: Check Go
echo [2/6] Checking Go...
where go >nul 2>&1
if errorlevel 1 (
    echo ERROR: Go not found
    pause
    exit /b 1
)
go version
echo.

:: Build frontend
echo [3/6] Building frontend...
cd /d "%PROJECT_ROOT%\frontend"
call npm run build
if errorlevel 1 (
    echo ERROR: Frontend build failed
    cd /d "%PROJECT_ROOT%"
    pause
    exit /b 1
)
echo Frontend build completed
echo.

:: Copy frontend files for embed
echo [4/6] Preparing embed files...
set EMBED_DIR=%PROJECT_ROOT%\internal\frontend\dist
if exist "%EMBED_DIR%" rmdir /s /q "%EMBED_DIR%"
mkdir "%EMBED_DIR%"
xcopy /e /i /y "%PROJECT_ROOT%\frontend\dist" "%EMBED_DIR%" >nul
echo Frontend files copied to %EMBED_DIR%
echo.

:: Build backend
echo [5/6] Building backend...
cd /d "%PROJECT_ROOT%"

set LDFLAGS=-s -w
set BINARY_NAME=record-v2.exe
set OUTPUT_DIR=%PROJECT_ROOT%\bin

if not exist "%OUTPUT_DIR%" mkdir "%OUTPUT_DIR%"

echo Compiling Windows executable...
go build -ldflags "-s -w" -o "%OUTPUT_DIR%\%BINARY_NAME%" ./cmd/server
if errorlevel 1 (
    echo ERROR: Backend build failed
    pause
    exit /b 1
)
echo Backend build completed
echo.

:: Copy config file
echo [6/6] Copying config files...
if exist "%PROJECT_ROOT%\configs\config.yaml" (
    copy /Y "%PROJECT_ROOT%\configs\config.yaml" "%OUTPUT_DIR%\" >nul
    echo Config file copied
)

echo.
echo ========================================
echo   Build Complete!
echo ========================================
echo.
echo Output: %OUTPUT_DIR%
echo Binary: %OUTPUT_DIR%\%BINARY_NAME%
echo.
echo Usage:
echo   1. Copy exe file to target server
echo   2. Run record-v2.exe
echo   3. Open http://localhost:8080
echo.
pause
