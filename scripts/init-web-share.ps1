Param(
    [string]$ExePath = ".\web-share.exe",
    [string]$Language = "",
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
    try {
        $resp = Invoke-WebRequest `
            -Uri "http://127.0.0.1:21910/manage/settings/language" `
            -Method Post `
            -ContentType "application/x-www-form-urlencoded" `
            -Body $body `
            -UseBasicParsing `
            -MaximumRedirection 0 `
            -ErrorAction Stop
        $statusCode = [int]$resp.StatusCode
    } catch {
        if ($_.Exception.Response -and $_.Exception.Response.StatusCode) {
            $statusCode = [int]$_.Exception.Response.StatusCode
        } else {
            return $false
        }
    }

    return ($statusCode -ge 200 -and $statusCode -lt 400) -or $statusCode -eq 302 -or $statusCode -eq 303
}

function Get-Message {
    param(
        [string]$Lang,
        [string]$Key
    )

    switch ($Key) {
        "exe_missing" { return "Executable not found: " }
        "manager_timeout" { return "Manager did not become ready in time while applying language." }
        "apply_lang_failed" { return "Failed to apply default language to manager." }
        "done" { return "Initialization completed. Context menu and language are configured." }
        default { return "" }
    }
}

function Resolve-LanguageChoice {
    param([string]$Lang)

    if ($Lang -eq "en-US" -or $Lang -eq "zh-CN") {
        return $Lang
    }

    Write-Host "Select language / 选择语言:"
    Write-Host "1. English (en-US)"
    Write-Host "2. 中文 (zh-CN)"
    $choice = Read-Host "Enter 1 or 2"

    switch ($choice) {
        "2" { return "zh-CN" }
        default { return "en-US" }
    }
}

$Language = Resolve-LanguageChoice -Lang $Language

$resolvedExe = (Resolve-Path $ExePath).Path
if (-not (Test-Path -LiteralPath $resolvedExe)) {
    throw ((Get-Message -Lang $Language -Key "exe_missing") + $resolvedExe)
}

$installContextMenuScript = (Resolve-Path (Join-Path $PSScriptRoot "install-context-menu.ps1")).Path
$installStartupTaskScript = (Resolve-Path (Join-Path $PSScriptRoot "install-startup-task.ps1")).Path
$startScript = (Resolve-Path (Join-Path $PSScriptRoot "start-web-share.ps1")).Path

& $installContextMenuScript -ExePath $resolvedExe -Language $Language

$managerWasRunning = Test-ManagerReady
if (-not $managerWasRunning) {
    Start-Process -FilePath $resolvedExe -ArgumentList "run-manager" -WindowStyle Hidden
    if (-not (Wait-ManagerReady -WaitSeconds 10)) {
        throw (Get-Message -Lang $Language -Key "manager_timeout")
    }
}

if (-not (Set-ManagerDefaultLanguage -Lang $Language)) {
    throw (Get-Message -Lang $Language -Key "apply_lang_failed")
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
    } catch {
    }
}

Write-Host (Get-Message -Lang $Language -Key "done")
