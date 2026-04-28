# 重启 Record V2 服务脚本

Write-Host "=== 停止所有相关进程 ===" -ForegroundColor Cyan
Get-Process | Where-Object { $_.ProcessName -like "*go*" -or $_.ProcessName -like "*record*" } | ForEach-Object {
    Write-Host "停止进程: $($_.ProcessName) (PID: $($_.Id))"
    Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
}

Write-Host ""
Write-Host "=== 等待 2 秒 ===" -ForegroundColor Yellow
Start-Sleep -Seconds 2

Write-Host ""
Write-Host "=== 清理缓存并重新编译 ===" -ForegroundColor Cyan
Set-Location D:\CODE\ClaudeCode\record_V2
go clean -cache
go build -o bin\record_v2.exe .\cmd\server

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "=== 编译成功，启动服务 ===" -ForegroundColor Green
    .\bin\record_v2.exe
} else {
    Write-Host "编译失败，请检查错误信息" -ForegroundColor Red
}
