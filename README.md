# web-share

一个面向 Windows 的 Go 1.20 文件分享工具。

它现在采用方案 A：单个常驻管理器统一维护所有分享任务。右键菜单不再直接拉起一次性终端服务，而是把新分享投递给本地管理器；如果管理器或托盘尚未启动，会在后台静默拉起，并由托盘统一管理。

## 当前能力

- 文件右键可直接只读分享
- 文件夹右键可选择只读分享，或先输入上传密码再分享
- 文件始终只读
- 文件夹默认只读，只有设置密码时才开放上传
- 所有分享统一收敛到一个本地管理器进程
- 管理页面仅允许本机访问
- 具体分享页可被局域网内其他设备访问
- 支持 Windows 托盘入口

## 运行架构

- 后台管理器监听固定端口 `21910`
- 分享页地址形如 `http://<局域网IP>:21910/s/<share-id>`
- 管理页地址固定为 `http://127.0.0.1:21910/manage`
- 右键菜单只负责调用 `enqueue`
- 右键投递分享时，如果管理器或托盘未启动，会在后台自动拉起
- 托盘图标负责打开管理页

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
- 如果托盘未启动，程序也会在后台拉起托盘
- 右键分享不会自动打开管理页

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
