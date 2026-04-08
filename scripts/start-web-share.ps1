Param(
    [string]$ExePath = ".\web-share.exe",
    [switch]$StartManager = $true,
    [switch]$StartTray = $true
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$resolvedExe = (Resolve-Path $ExePath).Path

if (-not (Test-Path -LiteralPath $resolvedExe)) {
    throw "找不到可执行文件: $resolvedExe"
}

if ($StartManager) {
    Start-Process -FilePath $resolvedExe -ArgumentList "run-manager" -WindowStyle Hidden
}

if ($StartTray) {
    Start-Process -FilePath $resolvedExe -ArgumentList "tray" -WindowStyle Hidden
}

Write-Host "Web Share 启动命令已下发。"
