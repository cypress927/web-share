# web-share

一个面向 Windows 的 Go 1.20 文件分享工具。

它现在采用方案 A：单个常驻管理器统一维护所有分享任务。右键菜单不再直接拉起一次性终端服务，而是把新分享投递给本地管理器；管理器通过托盘图标进入管理页面。

## 当前能力

- 文件右键可直接只读分享
- 文件夹右键可选择只读分享，或先输入上传密码再分享
- 文件始终只读
- 文件夹默认只读，只有设置密码时才开放上传
- 所有分享统一收敛到一个本地管理器进程
- 管理页面仅允许本机访问
- 具体分享页可被局域网内其他设备访问
- 支持 Windows 托盘入口
- 支持安装为 Windows 服务并设置当前用户登录时自动启动托盘

## 运行架构

- 后台管理器监听固定端口 `21910`
- 分享页地址形如 `http://<局域网IP>:21910/s/<share-id>`
- 管理页地址固定为 `http://127.0.0.1:21910/manage`
- 右键菜单只负责调用 `enqueue`
- 托盘图标只负责打开管理页

## 为什么服务和托盘分开

Windows 服务运行在 Session 0，不能直接显示用户托盘图标。因此这里采用组合方式：

- 服务负责开机后常驻后台管理器
- 当前用户登录后自动启动托盘

这也是 Windows 上更符合系统约束的做法。

## 构建

如果要用于右键菜单，建议构建为无控制台窗口版本：

```powershell
go build -ldflags="-H=windowsgui" -o .\web-share.exe .\cmd\web-share
```

如果只想调试命令行输出，也可以先用普通构建：

```powershell
go build -o .\web-share.exe .\cmd\web-share
```

## 常用命令

```powershell
.\web-share.exe enqueue C:\path\to\file.txt
.\web-share.exe enqueue -password 123456 C:\path\to\folder
.\web-share.exe tray
.\web-share.exe run-manager
```

说明：

- `enqueue` 会把新分享发送给本地管理器
- 如果管理器未启动，程序会先在后台拉起管理器
- 首次投递分享时，也会尝试拉起托盘图标

## 安装右键菜单

```powershell
.\web-share.exe install-context-menu -exe .\web-share.exe
```

或者：

```powershell
.\scripts\install-context-menu.ps1 -ExePath .\web-share.exe
```

右键菜单行为：

- 文件：`只读分享`
- 文件夹：`只读分享`、`设置上传密码后分享`

## 卸载右键菜单

```powershell
.\web-share.exe uninstall-context-menu
```

## 安装 Windows 服务

需要管理员权限：

```powershell
.\web-share.exe install-service -exe .\web-share.exe
```

这会做两件事：

- 创建并启动 `WebShareManager` 服务，设为开机自动启动
- 为当前用户写入登录启动项，使托盘图标在登录后自动出现

卸载：

```powershell
.\web-share.exe uninstall-service
```

## 页面说明

- 访问者打开具体分享页，只能看到当前被分享的文件或文件夹内容
- 分享者通过托盘打开管理页，可查看所有正在共享的内容
- 管理页会显示本机访问地址和局域网访问地址
- 管理页支持停止单个分享

## 当前限制

- 当前所有分享共用固定端口 `21910`
- 管理页没有做口令保护，仅通过“只允许本机访问”限制
- 上传只写入共享目录根目录，不覆盖同名文件
- 托盘菜单目前只提供“打开管理页面”和“退出托盘”
