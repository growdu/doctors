# stop-services.ps1
# 一键停止所有 dev 后端服务 + admin-web dev server
$names = @("auth","order","match","message","payment","review","sos","user","escort","wallet","admin")
foreach ($n in $names) {
    $p = Get-Process -Name $n -ErrorAction SilentlyContinue
    if ($p) {
        Write-Host "Stopping $n (PID=$($p.Id))..."
        Stop-Process -Id $p.Id -Force
    }
}
# admin-web dev server 是 node.exe 由 vite.cmd 拉起的；按命令行匹配
Get-CimInstance Win32_Process -Filter "Name = 'node.exe'" |
    Where-Object { $_.CommandLine -like "*vite*" } |
    ForEach-Object {
        Write-Host "Stopping admin-web vite (PID=$($_.ProcessId))..."
        Stop-Process -Id $_.ProcessId -Force
    }
Write-Host "All dev services stopped."