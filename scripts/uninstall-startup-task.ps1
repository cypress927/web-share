Param(
    [string]$TaskName = "WebShare.AutoStart"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$existing = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
if (-not $existing) {
    Write-Host "未找到计划任务：$TaskName"
    exit 0
}

Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
Write-Host "已删除开机自启计划任务：$TaskName"
