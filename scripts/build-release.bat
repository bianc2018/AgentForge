@echo off
REM build-release.bat — AgentForge 一键编译发布脚本（Windows）
REM
REM 用法：
REM   scripts\build-release.bat                    snapshot 构建
REM   scripts\build-release.bat --version v1.2.3   指定版本号
REM   scripts\build-release.bat --release          正式发布模式（需 tag）
REM   scripts\build-release.bat --help             显示帮助
REM
REM 前置条件：
REM   - Go >=1.21（go 在 PATH 中）
REM   - goreleaser（go install github.com/goreleaser/goreleaser/v2@latest）

setlocal enabledelayedexpansion

REM ─── 确保 Go 工具链在 PATH 中 ─────────────────────────────────────
if exist "%USERPROFILE%\go\bin" set "PATH=%USERPROFILE%\go\bin;%PATH%"

set VERSION=
set RELEASE=false
set SKIP_TESTS=

REM ─── 参数解析 ───────────────────────────────────────────────────
:parse_args
if "%~1"=="" goto :check_env
if /i "%~1"=="--help" goto :print_help
if /i "%~1"=="-h" goto :print_help
if /i "%~1"=="--version" (
    set "VERSION=%~2"
    shift
    shift
    goto :parse_args
)
if /i "%~1"=="--release" (
    set "RELEASE=true"
    shift
    goto :parse_args
)
if /i "%~1"=="--skip-tests" (
    set "SKIP_TESTS=--skip=validate"
    shift
    goto :parse_args
)
echo [ERROR] 未知参数: %~1
echo 使用 --help 查看可用选项
exit /b 2

REM ─── 帮助 ───────────────────────────────────────────────────────
:print_help
echo 用法: %~nx0 [选项]
echo.
echo 选项:
echo   --version ^<ver^>   指定版本号（如 v1.2.3），用于正式发布模式
echo   --release         正式发布模式（需 git tag），默认使用 snapshot
echo   --skip-tests      跳过单元测试
echo   --help            显示此帮助信息
echo.
echo 示例:
echo   %~nx0                         # snapshot 构建（日常开发）
echo   %~nx0 --version v1.0.0        # 以 v1.0.0 版本 snapshot 构建
echo   %~nx0 --release --version v1.0.0  # 正式发布 v1.0.0
exit /b 0

REM ─── 环境校验 ───────────────────────────────────────────────────
:check_env
echo [INFO] 校验构建环境...

REM 检查 Go
where go >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo [ERROR] Go 未安装，请从 https://go.dev/dl/ 下载安装
    exit /b 1
)
for /f "tokens=3" %%i in ('go version 2^>^&1') do set "GO_VER=%%i"
echo [INFO] Go %GO_VER%

REM 检查 goreleaser
where goreleaser >nul 2>&1
if %ERRORLEVEL% neq 0 (
    echo [ERROR] goreleaser 未安装
    echo   安装命令: go install github.com/goreleaser/goreleaser/v2@latest
    echo   安装后请确保 %%USERPROFILE%%\go\bin 在 PATH 中
    exit /b 1
)
echo [INFO] goreleaser 已就绪

REM ─── 构建发布 ───────────────────────────────────────────────────
echo [INFO] 开始构建...

if "%RELEASE%"=="true" (
    if not "%VERSION%"=="" set "GORELEASER_CURRENT_TAG=%VERSION%"
    echo [INFO] 正式发布模式
    if not "%VERSION%"=="" echo [INFO] 版本: %VERSION%

    if "%SKIP_TESTS%"=="" (
        goreleaser release
    ) else (
        goreleaser release --skip=validate
    )
) else (
    echo [INFO] Snapshot 模式

    if "%SKIP_TESTS%"=="" (
        goreleaser release --snapshot --clean
    ) else (
        goreleaser release --snapshot --clean --skip=validate
    )
)

if %ERRORLEVEL% neq 0 (
    echo [ERROR] 构建失败
    exit /b 1
)

REM ─── MSI 生成（Windows 安装包，需 WSL 或 Git Bash）────────────────
set "MSI_SCRIPT=%~dp0\build-msi.sh"
if exist "%MSI_SCRIPT%" (
    where bash >nul 2>&1
    if %ERRORLEVEL% equ 0 (
        echo [INFO] 生成 MSI 安装包...
        if "%VERSION%"=="" (
            for /f "tokens=*" %%i in ('git rev-parse --short HEAD 2^>nul') do set "MSI_VER=%%i"
            if "!MSI_VER!"=="" set "MSI_VER=dev"
        ) else (
            set "MSI_VER=%VERSION%"
        )
        bash "%MSI_SCRIPT%" "!MSI_VER!"
    ) else (
        echo [WARN] 未找到 bash^，跳过 MSI 生成 ^(MSI 需 WSL 或 Git Bash^)
    )
) else (
    echo [WARN] build-msi.sh 未找到，跳过 MSI 生成
)

echo [INFO] 构建完成，产物在 dist\ 目录
dir dist\ /b 2>nul | findstr /v "^$"
exit /b 0
