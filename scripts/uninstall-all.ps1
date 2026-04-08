Param(
    [string]$ExePath = ".\\web-share.exe",
    [string]$TaskName = "WebShare.AutoStart",
    [ValidateSet("en-US", "zh-CN")]
    [string]$Language = "en-US",
    [switch]$RemoveData,
    [switch]$StopProcesses = $true
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Stop-WebShareProcesses {
    param([string]$ResolvedExePath)

    try {
        Invoke-WebRequest -Uri "http://127.0.0.1:21910/api/shutdown" -Method Post -UseBasicParsing -TimeoutSec 1 | Out-Null
    } catch {}

    if (-not $ResolvedExePath) {
        Get-Process -Name "web-share" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
        return
    }

    $normalized = $ResolvedExePath.ToLowerInvariant()
    $candidates = Get-CimInstance Win32_Process -Filter "Name='web-share.exe'" -ErrorAction SilentlyContinue
    foreach ($proc in $candidates) {
        if (-not $proc.ExecutablePath) {
            continue
        }
        if ($proc.ExecutablePath.ToLowerInvariant() -ne $normalized) {
            continue
        }
        Stop-Process -Id $proc.ProcessId -Force -ErrorAction SilentlyContinue
    }
}

$resolvedExe = $null
try {
    $resolvedExe = (Resolve-Path $ExePath).Path
} catch {
    $resolvedExe = ""
}

$uninstallContextMenuScript = (Resolve-Path (Join-Path $PSScriptRoot "uninstall-context-menu.ps1")).Path
$uninstallStartupTaskScript = (Resolve-Path (Join-Path $PSScriptRoot "uninstall-startup-task.ps1")).Path

& $uninstallContextMenuScript -ExePath $resolvedExe -Language $Language
& $uninstallStartupTaskScript -TaskName $TaskName -Language $Language

if ($StopProcesses) {
    Stop-WebShareProcesses -ResolvedExePath $resolvedExe
}

$webShareDir = Join-Path $env:LOCALAPPDATA "WebShare"
$promptScriptPath = Join-Path $webShareDir "prompt-share.vbs"
if (Test-Path -LiteralPath $promptScriptPath) {
    Remove-Item -LiteralPath $promptScriptPath -Force
}

if ($RemoveData) {
    if (Test-Path -LiteralPath $webShareDir) {
        Remove-Item -LiteralPath $webShareDir -Recurse -Force
    }
    $cacheDbPath = Join-Path (Join-Path $env:LOCALAPPDATA "WebShare") "web-share.db"
    if (Test-Path -LiteralPath $cacheDbPath) {
        Remove-Item -LiteralPath $cacheDbPath -Force
    }
}

if ($Language -eq "zh-CN") {
    Write-Host "Web Share 已卸载（右键菜单、计划任务、运行进程清理完成）。"
    if ($RemoveData) {
        Write-Host "已额外删除本地数据。"
    }
} else {
    Write-Host "Web Share uninstalled (context menu, scheduled task, and running processes cleaned)."
    if ($RemoveData) {
        Write-Host "Local data removed."
    }
}
