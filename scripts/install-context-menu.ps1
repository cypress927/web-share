Param(
    [string]$ExePath = ".\web-share.exe",
    [ValidateSet("en-US", "zh-CN")]
    [string]$Language = "en-US"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$resolved = (Resolve-Path $ExePath).Path
if (-not (Test-Path -LiteralPath $resolved)) {
    throw "Executable not found: $resolved"
}

& $resolved install-context-menu -exe $resolved -lang $Language
$exitCodeVar = Get-Variable -Name LASTEXITCODE -ErrorAction SilentlyContinue
if ($null -ne $exitCodeVar -and $exitCodeVar.Value -ne 0) {
    throw "Install context menu failed with exit code $($exitCodeVar.Value). Rebuild web-share.exe if it does not support -lang yet."
}

Write-Host "Context menu installed. Language: $Language"
