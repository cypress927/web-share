Param(
    [string]$ExePath = ".\\web-share.exe",
    [ValidateSet("en-US", "zh-CN")]
    [string]$Language = "en-US",
    [switch]$InstallStartupTask = $true,
    [string]$TaskName = "WebShare.AutoStart",
    [switch]$ForceTask,
    [switch]$StartNow = $true,
    [switch]$NotifyStart = $true
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Test-ManagerReady {
    param([int]$TimeoutSec = 1)
    try {
        $resp = Invoke-WebRequest -Uri "http://127.0.0.1:21910/api/ping" -UseBasicParsing -TimeoutSec $TimeoutSec
        return $resp.StatusCode -eq 200
    } catch {
        return $false
    }
}

function Wait-ManagerReady {
    param([int]$WaitSeconds = 8)
    $deadline = (Get-Date).AddSeconds([Math]::Max(1, $WaitSeconds))
    while ((Get-Date) -lt $deadline) {
        if (Test-ManagerReady) {
            return $true
        }
        Start-Sleep -Milliseconds 250
    }
    return $false
}

function Set-ManagerDefaultLanguage {
    param([string]$Lang)
    $body = "default_lang=$([uri]::EscapeDataString($Lang))"
    $resp = Invoke-WebRequest `
        -Uri "http://127.0.0.1:21910/manage/settings/language" `
        -Method Post `
        -ContentType "application/x-www-form-urlencoded" `
        -Body $body `
        -UseBasicParsing `
        -TimeoutSec 5
    return $resp.StatusCode -ge 200 -and $resp.StatusCode -lt 400
}

$resolvedExe = (Resolve-Path $ExePath).Path
if (-not (Test-Path -LiteralPath $resolvedExe)) {
    throw "Executable not found: $resolvedExe"
}

$installContextMenuScript = (Resolve-Path (Join-Path $PSScriptRoot "install-context-menu.ps1")).Path
$installStartupTaskScript = (Resolve-Path (Join-Path $PSScriptRoot "install-startup-task.ps1")).Path
$startScript = (Resolve-Path (Join-Path $PSScriptRoot "start-web-share.ps1")).Path

& $installContextMenuScript -ExePath $resolvedExe -Language $Language

$managerWasRunning = Test-ManagerReady
if (-not $managerWasRunning) {
    Start-Process -FilePath $resolvedExe -ArgumentList "run-manager" -WindowStyle Hidden
    if (-not (Wait-ManagerReady -WaitSeconds 10)) {
        throw "Manager did not become ready in time while applying language."
    }
}

$applied = Set-ManagerDefaultLanguage -Lang $Language
if (-not $applied) {
    throw "Failed to apply default language to manager."
}

if ($InstallStartupTask) {
    if ($ForceTask) {
        & $installStartupTaskScript -ExePath $resolvedExe -TaskName $TaskName -Language $Language -NotifyStart:$NotifyStart -Force
    } else {
        & $installStartupTaskScript -ExePath $resolvedExe -TaskName $TaskName -Language $Language -NotifyStart:$NotifyStart
    }
}

if ($StartNow) {
    & $startScript -ExePath $resolvedExe -Language $Language -NotifyStart:$NotifyStart
} elseif (-not $managerWasRunning) {
    try {
        Invoke-WebRequest -Uri "http://127.0.0.1:21910/api/shutdown" -Method Post -UseBasicParsing -TimeoutSec 2 | Out-Null
    } catch {}
}

if ($Language -eq "zh-CN") {
    Write-Host "初始化完成：语言=$Language，右键菜单已安装。"
} else {
    Write-Host "Initialization completed: language=$Language, context menu installed."
}
