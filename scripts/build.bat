@echo off
REM minimal-build.bat - Just build and package for all platforms

setlocal EnableDelayedExpansion

echo Building P2P Chat for all platforms...
echo.

REM Paths
set DAEMON_DIR=..\daemon\cmd\p2p-chat-daemon
set UI_DIR=..\ui
set BUILD_DIR=..\execute

REM Clean
if exist "%BUILD_DIR%" rmdir /s /q "%BUILD_DIR%"
if exist "ui-temp" rmdir /s /q "ui-temp"
mkdir "%BUILD_DIR%"

echo [1/3] Building UI...
pushd "%UI_DIR%"
call npm run build
if errorlevel 1 (
    echo ERROR: UI build failed
    popd
    pause
    exit /b 1
)
popd

REM Find UI output directory
set UI_OUTPUT=
if exist "%UI_DIR%\dist" set UI_OUTPUT=%UI_DIR%\dist
if exist "%UI_DIR%\build" set UI_OUTPUT=%UI_DIR%\build

if "%UI_OUTPUT%"=="" (
    echo ERROR: Cannot find UI build output
    echo Checked: %UI_DIR%\dist and %UI_DIR%\build
    dir "%UI_DIR%"
    pause
    exit /b 1
)

echo Found UI output: %UI_OUTPUT%

REM Copy UI
xcopy /e /i /y "%UI_OUTPUT%" "%DAEMON_DIR%/api/dist" >nul

echo [2/3] Embedding UI...

REM Create embed.go
(
echo //go:build !dev
echo.
echo package main
echo.
echo import ^(
echo     "embed"
echo     "io/fs"
echo     "net/http"
echo ^)
echo.
echo //go:embed ui-temp/*
echo var uiFiles embed.FS
echo.
echo func getUIHandler^(^) http.Handler ^{
echo     uiFS, _ := fs.Sub^(uiFiles, "ui-temp"^)
echo     return http.FileServer^(http.FS^(uiFS^)^)
echo ^}
) > "%DAEMON_DIR%\embed.go"

REM Copy UI to daemon
mkdir "%DAEMON_DIR%\ui\dist" >nul 2>&1
xcopy /e /i /y "ui-temp" "%DAEMON_DIR%\ui\dist\" >nul

echo [3/3] Building binaries...

pushd "%DAEMON_DIR%"

set CGO_ENABLED=0

REM Windows
echo Building Windows...
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-w -s" -o "..\..\%BUILD_DIR%\p2p-chat-daemon.exe"

REM Linux
echo Building Linux...
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-w -s" -o "..\..\%BUILD_DIR%\p2p-chat-daemon-linux"

REM macOS Intel
echo Building macOS Intel...
set GOOS=darwin
set GOARCH=amd64
go build -ldflags="-w -s" -o "..\..\%BUILD_DIR%\p2p-chat-daemon-darwin"

REM macOS Apple Silicon
echo Building macOS ARM...
set GOOS=darwin
set GOARCH=arm64
go build -ldflags="-w -s" -o "..\..\%BUILD_DIR%\p2p-chat-daemon-darwin-arm64"

popd

REM Create simple README
(
echo # P2P Chat Binaries
echo.
echo Usage:
echo   Windows:  p2p-chat-daemon.exe -api 127.0.0.1:59579 -mdns -pub -key key2 -db chat2.db
echo   Linux:    ./p2p-chat-daemon-linux -api 127.0.0.1:59579 -mdns -pub -key key2 -db chat2.db
echo   macOS:    ./p2p-chat-daemon-darwin -api 127.0.0.1:59579 -mdns -pub -key key2 -db chat2.db
echo.
echo Web UI available at: http://localhost:3000
) > "%BUILD_DIR%\README.md"

REM Cleanup
if exist "ui-temp" rmdir /s /q "ui-temp"
if exist "%DAEMON_DIR%\embed.go" del "%DAEMON_DIR%\embed.go"
if exist "%DAEMON_DIR%\ui-temp" rmdir /s /q "%DAEMON_DIR%\ui-temp"

echo.
echo DONE! Built files:
dir /b "%BUILD_DIR%"
echo.
