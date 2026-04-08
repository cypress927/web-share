# web-share

One-line description: A lightweight Windows LAN sharing tool that lets you share files, folders, and clipboard content from the context menu or tray, all managed by a local web manager.

一句话描述：一个面向 Windows 的轻量局域网分享工具，支持右键分享文件/文件夹、托盘分享剪贴板，并通过本地 Web 管理器统一管理。

## English

### Overview

`web-share` is a Windows-focused temporary sharing tool built with Go 1.20.

It integrates with the Windows context menu, runs a local HTTP manager on port `21910`, and provides a tray entry point for management. You can share:

- Files
- Folders
- Clipboard text snapshots
- Clipboard image snapshots
- Clipboard file/folder paths

### Current Features

- Read-only file sharing from Windows context menu
- Read-only folder sharing from Windows context menu
- Folder sharing with upload password
- Tray action to share current clipboard
- Web management page for active shares
- Public home page for visible shares
- Share code access
- Text and image preview
- Subfolder browsing
- Resume-capable single-file downloads
- ZIP download for root folder and subfolders
- Sequential chunk upload with progress
- Folder upload from browser

### Runtime Model

- Manager address: `http://127.0.0.1:21910/manage`
- Public home page: `http://127.0.0.1:21910/`
- Share URL: `http://<LAN-IP>:21910/s/<share-code>`
- `enqueue` starts manager and tray automatically if they are not running
- New shares are hidden from the public home page by default

### Build

Windows GUI build:

```powershell
go build -ldflags="-H=windowsgui" -o .\web-share.exe .\cmd\web-share
```

Console build:

```powershell
go build -o .\web-share.exe .\cmd\web-share
```

Note:

- Setup scripts rely on `web-share.exe` supporting `install-context-menu -lang`
- Rebuild the executable first if you are using an older binary

### Setup

Recommended: run the initialization script.

```powershell
.\scripts\init-web-share.ps1 -ExePath .\web-share.exe
```

If `-Language` is omitted, the script will prompt you to choose:

- `1` -> `en-US`
- `2` -> `zh-CN`

Default behavior:

- `-InstallStartupTask` is enabled by default
- `-StartNow` is enabled by default
- `-NotifyStart` is enabled by default

To disable startup-task installation or immediate start explicitly:

```powershell
.\scripts\init-web-share.ps1 -ExePath .\web-share.exe -InstallStartupTask:$false -StartNow:$false
```

Optional parameters:

- `-Language en-US|zh-CN`
- `-InstallStartupTask`
- `-TaskName WebShare.AutoStart`
- `-ForceTask`
- `-StartNow`
- `-NotifyStart`

What the init script does:

- Installs the context menu
- Sets the default system language
- Optionally installs a startup scheduled task
- Optionally starts manager and tray immediately
- If manager is not running, it may start manager temporarily in order to persist the default language
- If `-StartNow:$false` is used and manager was only started temporarily by the script, it will be shut down again at the end

### Context Menu

Direct command:

```powershell
.\web-share.exe install-context-menu -exe .\web-share.exe -lang en-US
```

Script:

```powershell
.\scripts\install-context-menu.ps1 -ExePath .\web-share.exe -Language en-US
```

English menu:

- File: `Share via Web > Read-Only Share`
- Folder: `Share via Web > Read-Only Share`
- Folder: `Share via Web > Share with Upload Password`

Chinese menu:

- 文件：`通过 Web 分享 > 只读分享`
- 文件夹：`通过 Web 分享 > 只读分享`
- 文件夹：`通过 Web 分享 > 设置上传密码后分享`

Uninstall:

```powershell
.\web-share.exe uninstall-context-menu
.\scripts\uninstall-context-menu.ps1 -ExePath .\web-share.exe
```

### Start and Auto Start

Manual start:

```powershell
.\scripts\start-web-share.ps1 -ExePath .\web-share.exe -Language en-US
```

Behavior:

- Starts manager first, then tray
- Shows a startup success notification
- Does not relaunch manager if it is already running

Install startup task:

```powershell
.\scripts\install-startup-task.ps1 -ExePath .\web-share.exe -Language en-US
```

Remove startup task:

```powershell
.\scripts\uninstall-startup-task.ps1 -TaskName WebShare.AutoStart
```

### Unified Uninstall

```powershell
.\scripts\uninstall-all.ps1 -ExePath .\web-share.exe
.\scripts\uninstall-all.ps1 -ExePath .\web-share.exe -RemoveData
```

The unified uninstall script removes:

- Context menu entries
- Scheduled task
- Running manager/tray processes
- Generated prompt script cache
- Optional local data under `%LOCALAPPDATA%\WebShare`

### CLI

```powershell
.\web-share.exe enqueue C:\path\to\file.txt
.\web-share.exe enqueue C:\path\to\folder
.\web-share.exe enqueue -password 123456 C:\path\to\folder
.\web-share.exe tray
.\web-share.exe run-manager
```

### Behavior Notes

- File shares are always read-only
- Folder shares are read-only unless upload password is set
- Clipboard text/image shares are snapshots
- Path-based shares are live views, not immutable snapshots
- Missing files/folders are reported in UI instead of silently recreated

### More Docs

- Usage guide: [docs/usage.md](C:/Users/zhjun/Desktop/code/web-share/docs/usage.md)
- Problem notes: [docs/problem-notes.md](C:/Users/zhjun/Desktop/code/web-share/docs/problem-notes.md)
- System behavior: [docs/system-behavior.md](C:/Users/zhjun/Desktop/code/web-share/docs/system-behavior.md)

## 中文

### 项目概览

`web-share` 是一个面向 Windows 的临时局域网分享工具，基于 Go 1.20 开发。

它集成到 Windows 右键菜单中，通过固定端口 `21910` 提供本地 HTTP 管理器，并通过托盘进入管理页面。当前支持分享：

- 文件
- 文件夹
- 剪贴板文本快照
- 剪贴板图片快照
- 剪贴板中的文件/文件夹路径

### 当前能力

- 文件右键只读分享
- 文件夹右键只读分享
- 文件夹右键设置上传密码后分享
- 托盘一键分享当前剪贴板
- Web 管理页查看当前分享
- 公开首页查看可见分享
- 分享码访问
- 文本与图片预览
- 子目录浏览
- 单文件断点续传下载
- 根目录与子目录打包下载
- 顺序分片上传与上传进度展示
- 浏览器文件夹上传

### 运行方式

- 管理页地址：`http://127.0.0.1:21910/manage`
- 公开首页：`http://127.0.0.1:21910/`
- 分享地址：`http://<局域网IP>:21910/s/<share-code>`
- `enqueue` 在管理器或托盘未启动时会自动拉起它们
- 新建分享默认不会显示在公开首页

### 构建

无控制台窗口版本：

```powershell
go build -ldflags="-H=windowsgui" -o .\web-share.exe .\cmd\web-share
```

普通控制台版本：

```powershell
go build -o .\web-share.exe .\cmd\web-share
```

说明：

- 安装脚本依赖当前 `web-share.exe` 支持 `install-context-menu -lang`
- 如果本地是旧版可执行文件，请先重新编译

### 初始化安装

推荐直接运行初始化脚本：

```powershell
.\scripts\init-web-share.ps1 -ExePath .\web-share.exe
```

如果没有传 `-Language`，脚本会交互式提示你选择：

- `1` -> `en-US`
- `2` -> `zh-CN`

默认行为：

- `-InstallStartupTask` 默认开启
- `-StartNow` 默认开启
- `-NotifyStart` 默认开启

如果你希望关闭“安装开机自启”或“立即启动”，需要显式传：

```powershell
.\scripts\init-web-share.ps1 -ExePath .\web-share.exe -InstallStartupTask:$false -StartNow:$false
```

可选参数：

- `-Language en-US|zh-CN`
- `-InstallStartupTask`
- `-TaskName WebShare.AutoStart`
- `-ForceTask`
- `-StartNow`
- `-NotifyStart`

初始化脚本会执行：

- 安装右键菜单
- 设置系统默认语言
- 可选安装开机自启计划任务
- 可选立即启动后台管理器和托盘
- 如果 manager 当前未运行，脚本可能会临时拉起 manager 用于写入默认语言
- 如果传入 `-StartNow:$false`，且 manager 只是被脚本临时拉起，脚本结束时会再次关闭它

### 右键菜单

直接命令：

```powershell
.\web-share.exe install-context-menu -exe .\web-share.exe -lang zh-CN
```

脚本方式：

```powershell
.\scripts\install-context-menu.ps1 -ExePath .\web-share.exe -Language zh-CN
```

英文菜单：

- 文件：`Share via Web > Read-Only Share`
- 文件夹：`Share via Web > Read-Only Share`
- 文件夹：`Share via Web > Share with Upload Password`

中文菜单：

- 文件：`通过 Web 分享 > 只读分享`
- 文件夹：`通过 Web 分享 > 只读分享`
- 文件夹：`通过 Web 分享 > 设置上传密码后分享`

卸载：

```powershell
.\web-share.exe uninstall-context-menu
.\scripts\uninstall-context-menu.ps1 -ExePath .\web-share.exe
```

### 启动与开机自启

手动启动：

```powershell
.\scripts\start-web-share.ps1 -ExePath .\web-share.exe -Language zh-CN
```

行为说明：

- 先启动后台管理器，再启动托盘
- 启动完成后弹出成功通知
- 如果管理器本来已运行，不会重复拉起

安装计划任务：

```powershell
.\scripts\install-startup-task.ps1 -ExePath .\web-share.exe -Language zh-CN
```

卸载计划任务：

```powershell
.\scripts\uninstall-startup-task.ps1 -TaskName WebShare.AutoStart
```

### 统一卸载

```powershell
.\scripts\uninstall-all.ps1 -ExePath .\web-share.exe
.\scripts\uninstall-all.ps1 -ExePath .\web-share.exe -RemoveData
```

统一卸载脚本会清理：

- 右键菜单
- 计划任务
- 后台管理器与托盘进程
- 自动生成的密码输入脚本缓存
- 可选删除 `%LOCALAPPDATA%\WebShare` 本地数据

### 命令行

```powershell
.\web-share.exe enqueue C:\path\to\file.txt
.\web-share.exe enqueue C:\path\to\folder
.\web-share.exe enqueue -password 123456 C:\path\to\folder
.\web-share.exe tray
.\web-share.exe run-manager
```

### 行为说明

- 文件分享始终只读
- 文件夹分享默认只读，设置上传密码后才允许上传
- 剪贴板文本/图片分享是快照
- 路径型分享不是文件快照，而是实时路径视图
- 文件或文件夹失效时，页面会给出提示，不会静默重建目录

### 更多文档

- 用法说明：[docs/usage.md](C:/Users/zhjun/Desktop/code/web-share/docs/usage.md)
- 问题记录：[docs/problem-notes.md](C:/Users/zhjun/Desktop/code/web-share/docs/problem-notes.md)
- 系统行为：[docs/system-behavior.md](C:/Users/zhjun/Desktop/code/web-share/docs/system-behavior.md)
