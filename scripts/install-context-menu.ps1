Param(
    [string]$ExePath = ".\web-share.exe"
)

$resolved = (Resolve-Path $ExePath).Path
$scriptDir = Join-Path $env:LOCALAPPDATA "WebShare"
$scriptPath = Join-Path $scriptDir "prompt-share.vbs"
$scriptBody = @'
Dim exePath, targetPath, passwordText, shell, quote, commandText
If WScript.Arguments.Count < 2 Then
    WScript.Quit 1
End If

exePath = WScript.Arguments(0)
targetPath = WScript.Arguments(1)
passwordText = InputBox("请输入上传密码。留空则取消分享。", "Web Share 上传密码", "")

If Len(Trim(passwordText)) = 0 Then
    WScript.Quit 0
End If

Set shell = CreateObject("WScript.Shell")
quote = Chr(34)
commandText = quote & exePath & quote & " enqueue -password " & quote & Replace(passwordText, quote, quote & quote) & quote & " " & quote & targetPath & quote
shell.Run commandText, 0, False
'@

New-Item -ItemType Directory -Force -Path $scriptDir | Out-Null
Set-Content -Path $scriptPath -Value $scriptBody -Encoding Unicode
$passwordCommand = "wscript.exe `"$scriptPath`" `"$resolved`" `"%1`""

reg delete "HKCU\Software\Classes\*\shell\web-share" /f
reg delete "HKCU\Software\Classes\Directory\shell\web-share" /f
reg delete "HKCU\Software\Classes\WebShare.FileContextMenu" /f
reg delete "HKCU\Software\Classes\WebShare.DirectoryContextMenu" /f

reg add "HKCU\Software\Classes\*\shell\web-share" /f
reg add "HKCU\Software\Classes\*\shell\web-share" /v "MUIVerb" /d "通过 Web 分享" /f
reg add "HKCU\Software\Classes\*\shell\web-share" /v "Icon" /d $resolved /f
reg add "HKCU\Software\Classes\*\shell\web-share" /v "ExtendedSubCommandsKey" /d "WebShare.FileContextMenu" /f
reg add "HKCU\Software\Classes\WebShare.FileContextMenu\shell\readonly" /v "MUIVerb" /d "只读分享" /f
reg add "HKCU\Software\Classes\WebShare.FileContextMenu\shell\readonly\command" /ve /d "`"$resolved`" enqueue `"%1`"" /f

reg add "HKCU\Software\Classes\Directory\shell\web-share" /f
reg add "HKCU\Software\Classes\Directory\shell\web-share" /v "MUIVerb" /d "通过 Web 分享" /f
reg add "HKCU\Software\Classes\Directory\shell\web-share" /v "Icon" /d $resolved /f
reg add "HKCU\Software\Classes\Directory\shell\web-share" /v "ExtendedSubCommandsKey" /d "WebShare.DirectoryContextMenu" /f
reg add "HKCU\Software\Classes\WebShare.DirectoryContextMenu\shell\readonly" /v "MUIVerb" /d "只读分享" /f
reg add "HKCU\Software\Classes\WebShare.DirectoryContextMenu\shell\readonly\command" /ve /d "`"$resolved`" enqueue `"%1`"" /f
reg add "HKCU\Software\Classes\WebShare.DirectoryContextMenu\shell\password" /v "MUIVerb" /d "设置上传密码后分享" /f
reg add "HKCU\Software\Classes\WebShare.DirectoryContextMenu\shell\password\command" /ve /d $passwordCommand /f
