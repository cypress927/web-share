Param(
    [string]$TaskName = "WebShare.AutoStart",
    [ValidateSet("en-US", "zh-CN")]
    [string]$Language = "en-US"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$existing = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
if (-not $existing) {
    if ($Language -eq "zh-CN") {
        Write-Host "未找到计划任务：$TaskName"
    } else {
        Write-Host "Scheduled task not found: $TaskName"
    }
    exit 0
}

Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
if ($Language -eq "zh-CN") {
    Write-Host "已删除开机自启计划任务：$TaskName"
} else {
    Write-Host "Scheduled task removed: $TaskName"
}
