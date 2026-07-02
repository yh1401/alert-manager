# Alert Manager 一键启动脚本 (Windows)
# 同时启动后端 (Go) 和前端 (Vite)，各自独立窗口

$rootDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$backendDir = Join-Path $rootDir "backend"
$frontendDir = Join-Path $rootDir "frontend"

Write-Host "========================================"
Write-Host "  Alert Manager - 启动中..."
Write-Host "========================================"
Write-Host ""

# 1. 启动后端（新窗口）
Write-Host "[1/2] 启动后端服务 (Go) ..."
Start-Process powershell -ArgumentList @(
    "-NoExit",
    "-Command", "cd '$backendDir'; go run main.go"
) -WindowStyle Normal

Start-Sleep -Seconds 2

# 2. 启动前端（新窗口）
Write-Host "[2/2] 启动前端服务 (Vite) ..."
Start-Process powershell -ArgumentList @(
    "-NoExit",
    "-Command", "cd '$frontendDir'; npm run dev"
) -WindowStyle Normal

Start-Sleep -Seconds 2

Write-Host ""
Write-Host "========================================"
Write-Host "  服务已启动！"
Write-Host "========================================"
Write-Host "  后端: http://localhost:30333"
Write-Host "  前端: http://localhost:5173"
Write-Host ""
Write-Host "  关闭对应窗口即可停止服务"
Write-Host "========================================"
