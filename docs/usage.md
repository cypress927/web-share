# Usage / 使用说明

## English

### What It Does

`web-share` is a Windows LAN sharing tool.

It lets you share files, folders, and clipboard content over HTTP from Windows context menu or tray, while a local manager process keeps every share in one place.

### Core Rules

- File shares are always read-only
- Folder shares are read-only unless upload password is set
- Clipboard text/image shares are snapshots
- New shares are hidden from public home page by default
- Management page is accessed from tray

### Recommended User Flow

- Double-click `web-share.exe`
- Wait for tray icon and startup notification
- Open the management page from tray when needed
- The browser does not open automatically on normal launch
- On first run, the default language follows Windows system language
- If context menu is missing, normal launch installs it automatically

### Build

```powershell
go build -ldflags="-H=windowsgui" -o .\web-share.exe .\cmd\web-share
```

### First-Time Setup

Recommended for end users:

```powershell
.\web-share.exe
```

Advanced/manual install:

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

If you want to disable startup-task installation or immediate start explicitly:

```powershell
.\web-share.exe install -lang en-US -startup-task=false -start-now=false
```

It will:

- Persist default language directly to local settings
- Install context menu
- Optionally install a per-user auto-start entry in `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
- Optionally start manager and tray immediately

Legacy wrapper script:

```powershell
.\scripts\init-web-share.ps1 -ExePath .\web-share.exe
```

If `-Language` is omitted, the script will ask you to choose:

- `1` -> `en-US`
- `2` -> `zh-CN`

Default behavior:

- `-InstallStartupTask` is enabled by default
- `-StartNow` is enabled by default
- `-NotifyStart` is enabled by default

Note:

- Built-in `install/start/uninstall` now uses registry-based auto start
- Legacy scripts under `scripts/` still manage logon startup via Scheduled Task for compatibility

If you want to install context menu only:

```powershell
.\web-share.exe install-context-menu -exe .\web-share.exe -lang en-US
.\scripts\install-context-menu.ps1 -ExePath .\web-share.exe -Language en-US
```

If your binary does not support `-lang`, rebuild it first.

### Context Menu

English:

- File: `Share via Web > Read-Only Share`
- Folder: `Share via Web > Read-Only Share`
- Folder: `Share via Web > Share with Upload Password`

Chinese:

- 文件：`通过 Web 分享 > 只读分享`
- 文件夹：`通过 Web 分享 > 只读分享`
- 文件夹：`通过 Web 分享 > 设置上传密码后分享`

Upload-password folder sharing uses a native Windows password prompt and does not rely on PowerShell or VBS popups.

### Runtime Behavior

When you double-click `web-share.exe`:

- Manager starts automatically if not running
- Tray starts automatically if not running
- Startup success notification is shown
- Browser does not open automatically
- Missing context menu is installed automatically

When you create a share from context menu:

- Manager starts automatically if not running
- Tray starts automatically if not running
- Management page does not open automatically

Manager URL:

```text
http://127.0.0.1:21910/manage
```

Public home page:

```text
http://127.0.0.1:21910/
```

Share page:

```text
http://<LAN-IP>:21910/s/<share-code>
```

### Share Pages

- File shares can be downloaded directly
- Text files can be previewed
- Image files can be previewed
- File downloads support resume
- Folder shares support subfolder browsing
- Root folder and subfolders can be archived as ZIP
- Password-enabled folders show upload entry

### Upload

When upload password is enabled, folder share page supports:

- Single file upload
- Folder upload
- Sequential chunk upload
- Upload progress bar

Rules:

- Existing files are not overwritten
- Upload is rejected if current folder or share root is missing
- Missing folders are not recreated silently

### Tray

Current tray menu:

- `Open Manager`
- `Share Clipboard`
- `Exit Program`

Clipboard share priority:

- File/folder list
- Image
- Text

Default clipboard titles:

- Text: `Text: <first line>`
- Image: `Image: <timestamp>`

### CLI

```powershell
.\web-share.exe enqueue C:\path\to\file.txt
.\web-share.exe enqueue C:\path\to\folder
.\web-share.exe enqueue -password 123456 C:\path\to\folder
.\web-share.exe tray
.\web-share.exe run-manager
```

### Start Script and Auto Start

Preferred manual start:

```powershell
.\web-share.exe start -lang en-US
```

Behavior:

- Starts manager and tray when needed
- Shows startup success notification
- Avoids relaunching already running components

Legacy wrapper script:

```powershell
.\scripts\start-web-share.ps1 -ExePath .\web-share.exe -Language en-US
```

Enable auto start with built-in install:

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

### Uninstall

Remove context menu:

```powershell
.\web-share.exe uninstall-context-menu
.\scripts\uninstall-context-menu.ps1 -ExePath .\web-share.exe
```

Preferred uninstall:

```powershell
.\web-share.exe uninstall
.\web-share.exe uninstall -remove-data=true
```

Legacy wrapper script:

```powershell
.\scripts\uninstall-all.ps1 -ExePath .\web-share.exe
.\scripts\uninstall-all.ps1 -ExePath .\web-share.exe -RemoveData
```

The unified uninstall script removes:

- Context menu
- Built-in uninstall removes the registry auto-start entry
- Legacy scripts remove the Scheduled Task entry
- Running manager/tray processes
- Generated prompt script
- Optional local data directory

## 中文

### 项目用途

`web-share` 是一个面向 Windows 的本地局域网分享工具。

它可以把文件、文件夹和剪贴板内容通过 HTTP 临时共享出去，并集成到 Windows 右键菜单中；所有分享由后台管理器统一维护，通过托盘进入管理页面。

### 核心规则

- 文件分享始终只读
- 文件夹默认只读，设置上传密码后才允许上传
- 剪贴板文本/图片分享是快照
- 新建分享默认首页隐藏
- 管理页面通过托盘图标进入

### 推荐使用方式

- 双击 `web-share.exe`
- 等待托盘图标出现和启动完成通知
- 需要时从托盘打开管理页面
- 正常启动不会自动打开浏览器
- 首次启动时默认语言会跟随 Windows 系统语言
- 如果右键菜单缺失，正常启动会自动补装

### 构建

```powershell
go build -ldflags="-H=windowsgui" -o .\web-share.exe .\cmd\web-share
```

### 首次使用

推荐普通用户直接运行：

```powershell
.\web-share.exe
```

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

如果你希望显式关闭“安装开机自启”或“立即启动”，请传：

```powershell
.\web-share.exe install -lang zh-CN -startup-task=false -start-now=false
```

内置安装命令会：

- 直接写入默认语言到本地设置
- 安装右键菜单
- 可选安装当前用户注册表自启动项 `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
- 可选立即启动 manager 和 tray

兼容脚本入口：

```powershell
.\scripts\init-web-share.ps1 -ExePath .\web-share.exe
```

如果不传 `-Language`，脚本会提示你选择：

- `1` -> `en-US`
- `2` -> `zh-CN`

默认行为：

- `-InstallStartupTask` 默认开启
- `-StartNow` 默认开启
- `-NotifyStart` 默认开启

说明：

- 内置 `install/start/uninstall` 已改为使用注册表自启动
- `scripts/` 目录下的旧脚本仍保留计划任务实现，作为兼容入口

如果只想单独安装右键菜单：

```powershell
.\web-share.exe install-context-menu -exe .\web-share.exe -lang zh-CN
.\scripts\install-context-menu.ps1 -ExePath .\web-share.exe -Language zh-CN
```

如果当前 `web-share.exe` 还不支持 `-lang`，请先重新编译。

### 右键菜单

英文：

- 文件：`Share via Web > Read-Only Share`
- 文件夹：`Share via Web > Read-Only Share`
- 文件夹：`Share via Web > Share with Upload Password`

中文：

- 文件：`通过 Web 分享 > 只读分享`
- 文件夹：`通过 Web 分享 > 只读分享`
- 文件夹：`通过 Web 分享 > 设置上传密码后分享`

带上传密码的文件夹分享现在使用原生 Windows 密码输入框，不依赖 PowerShell 或 VBS 弹窗。

### 启动行为

当你双击 `web-share.exe` 时：

- 如果后台管理器未启动，程序会自动在后台启动它
- 如果托盘未启动，程序会自动在后台启动托盘
- 会弹出启动成功通知
- 不会自动打开浏览器
- 如果右键菜单缺失，会自动补装

当你通过右键菜单发起分享时：

- 如果后台管理器未启动，程序会自动在后台启动它
- 如果托盘未启动，程序也会自动在后台启动托盘
- 不会自动打开管理页面

管理页面地址：

```text
http://127.0.0.1:21910/manage
```

公开首页地址：

```text
http://127.0.0.1:21910/
```

分享页地址：

```text
http://<局域网IP>:21910/s/<share-code>
```

### 分享页面

- 文件分享页可直接下载
- 文本文件可直接预览
- 图片文件可直接预览
- 文件下载支持断点续传
- 文件夹支持子目录浏览
- 根目录和子目录都支持打包下载
- 设置上传密码后会显示上传入口

### 上传能力

启用上传密码后，文件夹分享页支持：

- 单文件上传
- 文件夹上传
- 顺序分片上传
- 上传进度条

规则：

- 不覆盖同名文件
- 当前目录或共享根目录失效时拒绝上传
- 不会静默重建失效目录

### 托盘

当前托盘菜单：

- `打开管理页面`
- `分享当前剪贴板`
- `退出程序`

剪贴板分享优先级：

- 文件/文件夹列表
- 图片
- 文本

默认命名：

- 文本：`Text: <首行摘要>`
- 图片：`Image: <时间>`

### 命令行

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

### 启动脚本与开机自启

推荐手动启动：

```powershell
.\web-share.exe start -lang zh-CN
```

行为说明：

- 需要时启动 manager 和 tray
- 启动完成后弹出成功通知
- 已运行的组件不会重复拉起

兼容脚本入口：

```powershell
.\scripts\start-web-share.ps1 -ExePath .\web-share.exe -Language zh-CN
```

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

### 卸载

卸载右键菜单：

```powershell
.\web-share.exe uninstall-context-menu
.\scripts\uninstall-context-menu.ps1 -ExePath .\web-share.exe
```

推荐卸载：

```powershell
.\web-share.exe uninstall
.\web-share.exe uninstall -remove-data=true
```

兼容脚本入口：

```powershell
.\scripts\uninstall-all.ps1 -ExePath .\web-share.exe
.\scripts\uninstall-all.ps1 -ExePath .\web-share.exe -RemoveData
```

统一卸载脚本会清理：

- 右键菜单
- 内置卸载会移除注册表自启动项
- 兼容脚本会移除计划任务
- 管理器和托盘进程
- 自动生成的密码输入脚本
- 可选本地数据目录
