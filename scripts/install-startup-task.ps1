Param(
    [string]$ExePath = ".\web-share.exe",
    [string]$TaskName = "WebShare.AutoStart",
    [switch]$Force
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$resolvedExe = (Resolve-Path $ExePath).Path
$scriptPath = (Resolve-Path (Join-Path $PSScriptRoot "start-web-share.ps1")).Path

if (-not (Test-Path -LiteralPath $resolvedExe)) {
    throw "找不到可执行文件: $resolvedExe"
}

$existing = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
if ($existing -and -not $Force) {
    throw "计划任务已存在：$TaskName。若要覆盖请加 -Force。"
}
if ($existing -and $Force) {
    Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
}

$argument = "-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$scriptPath`" -ExePath `"$resolvedExe`""
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

Write-Host "已创建开机自启计划任务：$TaskName"
