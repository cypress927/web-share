Param(
    [string]$ExePath = ".\\web-share.exe",
    [ValidateSet("en-US", "zh-CN")]
    [string]$Language = "en-US"
)

$resolved = (Resolve-Path $ExePath).Path
& $resolved install-context-menu -exe $resolved -lang $Language
if ($LASTEXITCODE -ne 0) {
    throw "Install context menu failed with exit code $LASTEXITCODE"
}

Write-Host "Context menu installed. Language: $Language"
