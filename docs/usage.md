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

### Build

```powershell
go build -ldflags="-H=windowsgui" -o .\web-share.exe .\cmd\web-share
```

### First-Time Setup

Recommended:

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

If you want to disable startup-task installation or immediate start explicitly:

```powershell
.\scripts\init-web-share.ps1 -ExePath .\web-share.exe -InstallStartupTask:$false -StartNow:$false
```

It will:

- Install context menu
- Set default language
- Optionally install startup scheduled task
- Optionally start manager and tray immediately
- If manager is not already running, the script may start it temporarily in order to save the default language
- If `-StartNow:$false` is used and manager was started only temporarily by the script, it will be shut down again at the end

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

### Runtime Behavior

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

Manual start:

```powershell
.\scripts\start-web-share.ps1 -ExePath .\web-share.exe -Language en-US
```

Behavior:

- Starts manager and tray
- Shows startup success notification
- Avoids relaunching manager if already running

Install logon startup task:

```powershell
.\scripts\install-startup-task.ps1 -ExePath .\web-share.exe -Language en-US
```

Remove task:

```powershell
.\scripts\uninstall-startup-task.ps1 -TaskName WebShare.AutoStart
```

### Uninstall

Remove context menu:

```powershell
.\web-share.exe uninstall-context-menu
.\scripts\uninstall-context-menu.ps1 -ExePath .\web-share.exe
```

Unified uninstall:

```powershell
.\scripts\uninstall-all.ps1 -ExePath .\web-share.exe
.\scripts\uninstall-all.ps1 -ExePath .\web-share.exe -RemoveData
```

The unified uninstall script removes:

- Context menu
- Startup task
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

### 构建

```powershell
go build -ldflags="-H=windowsgui" -o .\web-share.exe .\cmd\web-share
```

### 首次使用

推荐直接使用初始化脚本：

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

如果你希望显式关闭“安装开机自启”或“立即启动”，请传：

```powershell
.\scripts\init-web-share.ps1 -ExePath .\web-share.exe -InstallStartupTask:$false -StartNow:$false
```

初始化脚本会：

- 安装右键菜单
- 设置默认语言
- 可选安装开机自启计划任务
- 可选立即启动后台管理器和托盘
- 如果 manager 当前未运行，脚本可能会临时拉起 manager 用于保存默认语言
- 如果传入 `-StartNow:$false`，且 manager 只是被脚本临时拉起，脚本结束时会再次关闭它

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

### 启动行为

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
.\web-share.exe enqueue C:\path\to\file.txt
.\web-share.exe enqueue C:\path\to\folder
.\web-share.exe enqueue -password 123456 C:\path\to\folder
.\web-share.exe tray
.\web-share.exe run-manager
```

### 启动脚本与开机自启

手动启动：

```powershell
.\scripts\start-web-share.ps1 -ExePath .\web-share.exe -Language zh-CN
```

行为说明：

- 会启动后台管理器和托盘
- 启动完成后弹出成功通知
- 管理器已运行时不会重复启动

安装计划任务：

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

统一卸载：

```powershell
.\scripts\uninstall-all.ps1 -ExePath .\web-share.exe
.\scripts\uninstall-all.ps1 -ExePath .\web-share.exe -RemoveData
```

统一卸载脚本会清理：

- 右键菜单
- 计划任务
- 管理器和托盘进程
- 自动生成的密码输入脚本
- 可选本地数据目录
