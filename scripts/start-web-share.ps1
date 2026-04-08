Param(
    [string]$ExePath = ".\web-share.exe",
    [ValidateSet("en-US", "zh-CN")]
    [string]$Language = "en-US",
    [switch]$StartManager = $true,
    [switch]$StartTray = $true,
    [switch]$NotifyStart = $true,
    [int]$WaitSeconds = 8
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

function Show-StartupBalloon {
    param(
        [string]$Lang,
        [bool]$ManagerWasStarted,
        [bool]$TrayWasStarted
    )
    if (-not $NotifyStart) {
        return
    }
    try {
        Add-Type -AssemblyName System.Windows.Forms
        Add-Type -AssemblyName System.Drawing
        $notify = New-Object System.Windows.Forms.NotifyIcon
        $notify.Icon = [System.Drawing.SystemIcons]::Information
        $notify.Visible = $true
        $notify.BalloonTipTitle = "Web Share"
        if ($Lang -eq "zh-CN") {
            if ($ManagerWasStarted -or $TrayWasStarted) {
                $notify.BalloonTipText = "Web Share 已启动完成。"
            } else {
                $notify.BalloonTipText = "Web Share 已在运行。"
            }
        } else {
            if ($ManagerWasStarted -or $TrayWasStarted) {
                $notify.BalloonTipText = "Web Share started successfully."
            } else {
                $notify.BalloonTipText = "Web Share is already running."
            }
        }
        $notify.ShowBalloonTip(3000)
        Start-Sleep -Seconds 3
        $notify.Dispose()
    } catch {
        Write-Warning "Startup notification failed: $($_.Exception.Message)"
    }
}

$resolvedExe = (Resolve-Path $ExePath).Path

if (-not (Test-Path -LiteralPath $resolvedExe)) {
    throw "Executable not found: $resolvedExe"
}

$managerWasStarted = $false
$trayWasStarted = $false

$managerReady = Test-ManagerReady
if ($StartManager -and -not $managerReady) {
    Start-Process -FilePath $resolvedExe -ArgumentList "run-manager" -WindowStyle Hidden
    $managerWasStarted = $true
}

if ($StartManager) {
    $deadline = (Get-Date).AddSeconds([Math]::Max(1, $WaitSeconds))
    while ((Get-Date) -lt $deadline) {
        if (Test-ManagerReady) {
            $managerReady = $true
            break
        }
        Start-Sleep -Milliseconds 250
    }
}

if ($StartTray) {
    Start-Process -FilePath $resolvedExe -ArgumentList "tray" -WindowStyle Hidden
    $trayWasStarted = $true
}

Show-StartupBalloon -Lang $Language -ManagerWasStarted $managerWasStarted -TrayWasStarted $trayWasStarted

if ($Language -eq "zh-CN") {
    Write-Host "Web Share 启动命令已下发。"
} else {
    Write-Host "Web Share start command dispatched."
}
