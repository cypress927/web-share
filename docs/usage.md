# 使用说明

## 项目用途

`web-share` 是一个面向 Windows 的本地文件分享工具。

它可以把文件或文件夹通过 HTTP 临时共享出去，并集成到 Windows 右键菜单中。

## 核心行为

- 文件分享始终是只读
- 文件夹默认是只读
- 文件夹只有在设置上传密码时，访问者才可以上传文件到该目录
- 分享任务由后台管理器统一管理
- 管理页面通过托盘图标进入

## 首次使用

先构建程序：

```powershell
go build -ldflags="-H=windowsgui" -o .\web-share.exe .\cmd\web-share
```

然后安装右键菜单：

```powershell
.\web-share.exe install-context-menu -exe .\web-share.exe
```

如果右键菜单没有立刻刷新，可以重启 `explorer.exe`，或者注销后重新登录。

## 右键菜单用法

安装完成后：

- 右键文件
  - `通过 Web 分享 > 只读分享`
- 右键文件夹
  - `通过 Web 分享 > 只读分享`
  - `通过 Web 分享 > 设置上传密码后分享`

说明：

- 选择 `只读分享` 后，程序会在后台创建分享任务
- 选择 `设置上传密码后分享` 后，会弹出密码输入框
- 输入密码后，访问者打开该文件夹分享页时会看到上传入口
- 留空则取消本次分享

## 启动行为

当你通过右键菜单发起分享时：

- 如果后台管理器未启动，程序会自动在后台启动它
- 如果托盘未启动，程序也会自动在后台启动托盘
- 不会自动打开管理页面

## 托盘用法

程序启动后会出现托盘图标。

当前托盘菜单提供：

- `打开管理页面`
- `退出托盘`

管理页面地址固定为：

```text
http://127.0.0.1:21910/manage
```

## 分享页面

每个分享任务都有独立地址，例如：

```text
http://<局域网IP>:21910/s/<share-id>
```

访问者打开后：

- 文件分享页可以直接下载文件
- 文件夹分享页可以浏览目录内容
- 如果该文件夹分享设置了上传密码，则页面会显示上传表单

## 命令行用法

手动创建分享：

```powershell
.\web-share.exe enqueue C:\path\to\file.txt
.\web-share.exe enqueue C:\path\to\folder
.\web-share.exe enqueue -password 123456 C:\path\to\folder
```

手动启动托盘：

```powershell
.\web-share.exe tray
```

手动启动后台管理器：

```powershell
.\web-share.exe run-manager
```

## 卸载右键菜单

```powershell
.\web-share.exe uninstall-context-menu
```

## 常见说明

- 管理页仅允许本机访问
- 分享页可被局域网内其他设备访问
- 上传文件会写入共享目录根目录
- 当前不会覆盖同名文件
- 如果右键密码分享没有出现新行为，通常是因为右键菜单没有重新安装，需要先卸载再安装一次
