Param(
    [string]$ExePath = ".\web-share.exe",
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
    } catch {
    }

    if ([string]::IsNullOrWhiteSpace($ResolvedExePath)) {
        Get-Process -Name "web-share" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
        return
    }

    $normalized = $ResolvedExePath.ToLowerInvariant()
    $candidates = Get-CimInstance Win32_Process -Filter "Name='web-share.exe'" -ErrorAction SilentlyContinue
    foreach ($proc in $candidates) {
        if ([string]::IsNullOrWhiteSpace($proc.ExecutablePath)) {
            continue
        }
        if ($proc.ExecutablePath.ToLowerInvariant() -ne $normalized) {
            continue
        }
        Stop-Process -Id $proc.ProcessId -Force -ErrorAction SilentlyContinue
    }
}

function Get-Message {
    param(
        [string]$Lang,
        [string]$Key
    )

    switch ($Key) {
        "done" { return "Web Share uninstalled." }
        "data_removed" { return "Local data removed." }
        default { return "" }
    }
}

$resolvedExe = ""
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

if ($RemoveData -and (Test-Path -LiteralPath $webShareDir)) {
    Remove-Item -LiteralPath $webShareDir -Recurse -Force
}

Write-Host (Get-Message -Lang $Language -Key "done")
if ($RemoveData) {
    Write-Host (Get-Message -Lang $Language -Key "data_removed")
}
