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
- Double-clicking `web-share.exe` starts the manager and tray in the background
- Normal launch shows a startup notification and does not open the browser automatically
- The tray is the primary entry for opening the management page
- On first run, the saved default language follows the Windows system language
- On normal launch, the Windows context menu is installed automatically if missing
- `enqueue` starts manager and tray automatically if they are not running
- New shares are hidden from the public home page by default

Internal structure note:

- entry points live in `internal/app`, `internal/manager`, and `internal/tray`
- system integration state is mainly orchestrated by `internal/systemstate`
- Windows-specific implementations stay under `internal/systemstate/windows_ports_windows.go` and `internal/shell`

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

- `web-share.exe` now includes built-in install/start/repair/uninstall commands
- PowerShell scripts are still available as wrapper entry points
- Rebuild the executable first if you are using an older binary
- local system-action logs are written to `web-share.log` beside `web-share.exe`, with rotated backups `web-share.log.1` to `web-share.log.5`

### Setup

Recommended for end users: build a `windowsgui` executable and double-click `web-share.exe`.

Default double-click behavior:

- Starts manager in the background
- Starts tray in the background
- Shows a startup-complete notification
- Uses the saved default language, or initializes it from Windows on first run
- Automatically installs the context menu if it is missing
- Does not open the browser automatically

Optional advanced/manual install command:

```powershell
.\web-share.exe install -lang en-US
```

Example with Chinese:

```powershell
.\web-share.exe install -lang zh-CN
```

Built-in install defaults:

- `-context-menu=true`
- `-startup-task=true`
- `-start-now=true`
- `-notify-start=true`

To disable startup-task installation or immediate start explicitly:

```powershell
.\web-share.exe install -lang en-US -startup-task=false -start-now=false
```

What the built-in install command does:

- Persists the default system language directly to local settings
- Installs the Windows context menu
- Optionally installs a per-user auto-start entry in `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
- Optionally starts manager and tray immediately

Legacy wrapper script:

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

Script optional parameters:

- `-Language en-US|zh-CN`
- `-InstallStartupTask`
- `-TaskName WebShare.AutoStart`
- `-ForceTask`
- `-StartNow`
- `-NotifyStart`

Note:

- Built-in `install/start/uninstall` now uses registry-based auto start
- The legacy PowerShell scripts under `scripts/` still manage logon startup via Scheduled Task for compatibility

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

Upload-password folder sharing uses a native Windows password prompt and does not rely on PowerShell or VBS popups.

Current built-in context-menu install/uninstall commands are handled as target-state operations. If the target state is already satisfied or an old residual config is repaired, the local settings page may show a warning instead of a hard failure.

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

Preferred manual start:

```powershell
.\web-share.exe start -lang en-US
```

Built-in behavior:

- Starts manager first when needed
- Starts tray when needed
- Shows a startup success notification
- Does not relaunch already running components
- Does not open the browser automatically

Legacy wrapper script:

```powershell
.\scripts\start-web-share.ps1 -ExePath .\web-share.exe -Language en-US
```

Behavior:

- Starts manager first, then tray
- Shows a startup success notification
- Does not relaunch manager if it is already running

Enable auto start with the built-in installer:

```powershell
.\web-share.exe install -lang en-US -startup-task=true -start-now=false
```

Remove auto start with built-in uninstall:

```powershell
.\web-share.exe uninstall -remove-startup-task=true -remove-context-menu=false
```

Legacy Scheduled Task scripts:

```powershell
.\scripts\install-startup-task.ps1 -ExePath .\web-share.exe -Language en-US
```

Remove Scheduled Task:

```powershell
.\scripts\uninstall-startup-task.ps1 -TaskName WebShare.AutoStart
```

### Unified Uninstall

```powershell
.\web-share.exe uninstall
.\web-share.exe uninstall -remove-data=true
```

The built-in uninstall command removes:

- Context menu entries
- Registry-based auto-start entry
- Running manager/tray processes
- Generated prompt script cache
- Optional local data beside `web-share.exe`, mainly `web-share.db`

Current behavior notes:

- setup/system pages use local async actions with `success / warning / error` feedback
- warning is used for already-satisfied state, repaired legacy residue, or missing old objects
- system-action logs are kept locally for troubleshooting when a user cannot directly observe the result

For the normal user flow, you can also:

1. Open the Web system settings page from tray
2. Remove context menu and auto start
3. Exit the program from tray
4. Delete `web-share.exe`
5. Optionally delete `web-share.db` beside `web-share.exe`

Legacy wrapper script:

```powershell
.\scripts\uninstall-all.ps1 -ExePath .\web-share.exe
.\scripts\uninstall-all.ps1 -ExePath .\web-share.exe -RemoveData
```

### CLI

```powershell
.\web-share.exe install -lang en-US
.\web-share.exe start
.\web-share.exe repair -lang zh-CN
.\web-share.exe uninstall -remove-data=true
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
- 双击 `web-share.exe` 会在后台启动管理器和托盘
- 正常启动会弹出启动完成通知，不会自动打开浏览器
- 托盘是打开管理页面的主入口
- 首次启动时，默认语言会跟随 Windows 系统语言落库
- 正常启动时，如果右键菜单缺失，会自动补装
- `enqueue` 在管理器或托盘未启动时会自动拉起它们
- 新建分享默认不会显示在公开首页

内部结构说明：

- 入口层主要在 `internal/app`、`internal/manager`、`internal/tray`
- 系统集成状态的统一编排层在 `internal/systemstate`
- Windows 具体实现仍在 `internal/systemstate/windows_ports_windows.go` 与 `internal/shell`

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

- `web-share.exe` 现在内置了 `install/start/repair/uninstall` 命令
- PowerShell 脚本仍然保留，用作兼容包装入口
- 如果本地是旧版可执行文件，请先重新编译
- 系统动作日志默认写入程序目录下的 `web-share.log`，并自动轮转为 `web-share.log.1` 到 `web-share.log.5`

### 初始化安装

推荐终端用户直接编译 `windowsgui` 版本后双击 `web-share.exe`。

双击后的默认行为：

- 在后台启动管理器
- 在后台启动托盘
- 弹出启动完成通知
- 使用已保存的默认语言；若是首次启动，则从 Windows 系统语言初始化
- 如果右键菜单缺失则自动安装
- 不会自动打开浏览器

如需显式手动安装，也可以使用内置命令：

```powershell
.\web-share.exe install -lang zh-CN
```

英文安装示例：

```powershell
.\web-share.exe install -lang en-US
```

内置安装命令默认行为：

- `-context-menu=true`
- `-startup-task=true`
- `-start-now=true`
- `-notify-start=true`

如果你希望关闭“安装开机自启”或“立即启动”，可以显式传：

```powershell
.\web-share.exe install -lang zh-CN -startup-task=false -start-now=false
```

内置安装命令会执行：

- 直接把默认语言写入本地设置
- 安装 Windows 右键菜单
- 可选安装当前用户注册表自启动项 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
- 可选立即启动后台管理器和托盘

兼容脚本入口：

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

脚本可选参数：

- `-Language en-US|zh-CN`
- `-InstallStartupTask`
- `-TaskName WebShare.AutoStart`
- `-ForceTask`
- `-StartNow`
- `-NotifyStart`

说明：

- 内置 `install/start/uninstall` 已经改为使用注册表自启动
- `scripts/` 目录下的旧 PowerShell 脚本仍然保留计划任务方式，作为兼容入口

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

带上传密码的文件夹分享现在使用原生 Windows 密码输入框，不依赖 PowerShell 或 VBS 弹窗。

当前内置右键菜单安装与卸载已经按“目标状态”处理。如果对象本来已满足目标状态，或检测到旧残留后自动修复，本地设置页可能给出 warning，而不是直接报硬失败。

卸载：

```powershell
.\web-share.exe uninstall-context-menu
.\scripts\uninstall-context-menu.ps1 -ExePath .\web-share.exe
```

### 启动与开机自启

推荐手动启动：

```powershell
.\web-share.exe start -lang zh-CN
```

内置行为：

- 需要时先启动后台管理器
- 需要时启动托盘
- 显示启动成功通知
- 已运行的组件不会被重复拉起
- 不会自动打开浏览器

兼容脚本入口：

```powershell
.\scripts\start-web-share.ps1 -ExePath .\web-share.exe -Language zh-CN
```

行为说明：

- 先启动后台管理器，再启动托盘
- 启动完成后弹出成功通知
- 如果管理器本来已运行，不会重复拉起

通过内置安装命令启用自启动：

```powershell
.\web-share.exe install -lang zh-CN -startup-task=true -start-now=false
```

通过内置卸载命令移除自启动：

```powershell
.\web-share.exe uninstall -remove-startup-task=true -remove-context-menu=false
```

兼容的计划任务脚本：

```powershell
.\scripts\install-startup-task.ps1 -ExePath .\web-share.exe -Language zh-CN
```

卸载计划任务：

```powershell
.\scripts\uninstall-startup-task.ps1 -TaskName WebShare.AutoStart
```

### 统一卸载

```powershell
.\web-share.exe uninstall
.\web-share.exe uninstall -remove-data=true
```

内置卸载命令会清理：

- 右键菜单
- 注册表自启动项
- 后台管理器与托盘进程
- 自动生成的密码输入脚本缓存
- 可选删除 `web-share.exe` 同目录下的本地数据，主要是 `web-share.db`

当前补充说明：

- `setup` / `system settings` 页面已经支持 `success / warning / error` 三态反馈
- warning 用于表示“已在目标状态”“检测到旧残留并已修复”“某些旧对象原本已缺失”
- 系统动作的详细记录会落到本地日志，便于排查用户无感失败

对普通用户，也可以走这条更直接的路径：

1. 从托盘打开 Web 系统设置页
2. 卸载右键菜单、关闭自启动
3. 从托盘退出程序
4. 手动删除 `web-share.exe`
5. 按需删除 `web-share.exe` 同目录下的 `web-share.db`

兼容脚本入口：

```powershell
.\scripts\uninstall-all.ps1 -ExePath .\web-share.exe
.\scripts\uninstall-all.ps1 -ExePath .\web-share.exe -RemoveData
```

### 命令行

```powershell
.\web-share.exe install -lang zh-CN
.\web-share.exe start
.\web-share.exe repair -lang en-US
.\web-share.exe uninstall -remove-data=true
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
