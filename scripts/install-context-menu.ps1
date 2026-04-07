Param(
    [string]$ExePath = ".\web-share.exe"
)

$resolved = (Resolve-Path $ExePath).Path
$passwordCommand = "powershell.exe -NoProfile -WindowStyle Hidden -Command ""Add-Type -AssemblyName Microsoft.VisualBasic; `$p=[Microsoft.VisualBasic.Interaction]::InputBox('请输入上传密码。留空则取消分享。','Web Share 上传密码',''); if ([string]::IsNullOrWhiteSpace(`$p)) { exit 0 }; Start-Process -WindowStyle Hidden -FilePath '$resolved' -ArgumentList @('enqueue','-password',`$p,'%1')"""

reg add "HKCU\Software\Classes\*\shell\web-share" /ve /d "通过 Web 分享" /f
reg add "HKCU\Software\Classes\*\shell\web-share" /v "MUIVerb" /d "通过 Web 分享" /f
reg add "HKCU\Software\Classes\*\shell\web-share" /v "SubCommands" /d "" /f
reg add "HKCU\Software\Classes\*\shell\web-share" /v "Icon" /d $resolved /f
reg add "HKCU\Software\Classes\*\shell\web-share\shell\readonly" /ve /d "只读分享" /f
reg add "HKCU\Software\Classes\*\shell\web-share\shell\readonly\command" /ve /d "`"$resolved`" enqueue `"%1`"" /f

reg add "HKCU\Software\Classes\Directory\shell\web-share" /ve /d "通过 Web 分享" /f
reg add "HKCU\Software\Classes\Directory\shell\web-share" /v "MUIVerb" /d "通过 Web 分享" /f
reg add "HKCU\Software\Classes\Directory\shell\web-share" /v "SubCommands" /d "" /f
reg add "HKCU\Software\Classes\Directory\shell\web-share" /v "Icon" /d $resolved /f
reg add "HKCU\Software\Classes\Directory\shell\web-share\shell\readonly" /ve /d "只读分享" /f
reg add "HKCU\Software\Classes\Directory\shell\web-share\shell\readonly\command" /ve /d "`"$resolved`" enqueue `"%1`"" /f
reg add "HKCU\Software\Classes\Directory\shell\web-share\shell\password" /ve /d "设置上传密码后分享" /f
reg add "HKCU\Software\Classes\Directory\shell\web-share\shell\password\command" /ve /d $passwordCommand /f
