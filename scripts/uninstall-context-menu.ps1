Param(
    [string]$ExePath = ".\\web-share.exe",
    [ValidateSet("en-US", "zh-CN")]
    [string]$Language = "en-US"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$resolved = $null
try {
    $resolved = (Resolve-Path $ExePath).Path
} catch {
    $resolved = ""
}

if ($resolved -and (Test-Path -LiteralPath $resolved)) {
    & $resolved uninstall-context-menu
    if ($LASTEXITCODE -ne 0) {
        throw "Uninstall context menu failed with exit code $LASTEXITCODE"
    }
} else {
    reg delete "HKCU\Software\Classes\*\shell\web-share" /f | Out-Null
    reg delete "HKCU\Software\Classes\Directory\shell\web-share" /f | Out-Null
    reg delete "HKCU\Software\Classes\WebShare.FileContextMenu" /f | Out-Null
    reg delete "HKCU\Software\Classes\WebShare.DirectoryContextMenu" /f | Out-Null
}

$promptScriptPath = Join-Path (Join-Path $env:LOCALAPPDATA "WebShare") "prompt-share.vbs"
if (Test-Path -LiteralPath $promptScriptPath) {
    Remove-Item -LiteralPath $promptScriptPath -Force
}

if ($Language -eq "zh-CN") {
    Write-Host "已卸载右键菜单。"
} else {
    Write-Host "Context menu uninstalled."
}
