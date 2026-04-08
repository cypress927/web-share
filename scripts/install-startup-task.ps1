Param(
    [string]$ExePath = ".\web-share.exe",
    [string]$TaskName = "WebShare.AutoStart",
    [ValidateSet("en-US", "zh-CN")]
    [string]$Language = "en-US",
    [switch]$NotifyStart = $true,
    [switch]$Force
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$resolvedExe = (Resolve-Path $ExePath).Path
$scriptPath = (Resolve-Path (Join-Path $PSScriptRoot "start-web-share.ps1")).Path

if (-not (Test-Path -LiteralPath $resolvedExe)) {
    throw "Executable not found: $resolvedExe"
}

$existing = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
if ($existing -and -not $Force) {
    throw "Scheduled task already exists: $TaskName. Use -Force to replace it."
}
if ($existing -and $Force) {
    Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
}

$notifyArg = ""
if ($NotifyStart) {
    $notifyArg = " -NotifyStart"
}

$argument = "-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$scriptPath`" -ExePath `"$resolvedExe`" -Language `"$Language`"$notifyArg"
$action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument $argument
$trigger = New-ScheduledTaskTrigger -AtLogOn -User $env:USERNAME
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries

Register-ScheduledTask `
    -TaskName $TaskName `
    -Action $action `
    -Trigger $trigger `
    -Settings $settings `
    -Description "Start Web Share manager and tray at user logon" `
    -RunLevel Limited | Out-Null

Write-Host "Scheduled task created: $TaskName"
