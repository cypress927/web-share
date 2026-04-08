# 单文件安装模式重构方案

## 目标

把当前依赖 `PowerShell` 脚本完成初始化、安装右键菜单、安装开机自启、统一卸载的模式，重构为“单文件主程序内置安装器”模式。

最终目标是让普通用户只需要拿到一个 `web-share.exe`，即可完成：

- 初次安装
- 语言选择
- 右键菜单注册
- 开机自启配置
- 后台管理器与托盘启动
- 卸载
- 修复安装

不再把 `.ps1` 作为主要用户入口。

## 当前问题

当前项目虽然已经具备安装脚本，但从用户视角仍然存在几个明显问题：

### 1. 安装入口分散

用户需要理解多个脚本和多个命令：

- `init-web-share.ps1`
- `install-context-menu.ps1`
- `install-startup-task.ps1`
- `start-web-share.ps1`
- `uninstall-all.ps1`

这更像开发者工具，不像最终产品。

### 2. 脚本环境不稳定

脚本模式受这些因素影响：

- PowerShell 执行策略
- 编码问题
- 字符串转义
- `LASTEXITCODE` 严格模式行为
- 重定向状态码处理
- 用户是否愿意运行脚本

这些问题已经在当前项目里真实出现过。

### 3. 用户心智成本高

普通用户并不想知道：

- 为什么要运行脚本
- 为什么语言和右键菜单要单独处理
- 为什么有 manager、tray、startup task 的区别

用户只想完成“安装并使用”。

## 重构方向

建议将当前安装脚本逻辑全部迁入 Go 主程序，形成统一入口：

```text
web-share.exe
```

主程序同时承担三类角色：

- 业务程序
- 安装器
- 卸载器 / 修复器

## 推荐方案

推荐采用：

### 方案 A：单文件内置命令安装器

通过 Go 命令模式实现：

- `web-share.exe install`
- `web-share.exe uninstall`
- `web-share.exe repair`
- `web-share.exe tray`
- `web-share.exe run-manager`
- `web-share.exe enqueue ...`

安装逻辑、卸载逻辑、修复逻辑全部内置，不再依赖外部脚本。

这是最稳、改动最可控、最适合当前项目阶段的方案。

### 方案 B：在方案 A 基础上增加 GUI 安装向导

如果后续希望进一步提升普通用户体验，可在 `install` 模式上再加一个很薄的 GUI 页面。

用户双击 `web-share.exe` 后看到：

- 语言选择
- 安装右键菜单
- 开机自启
- 立即启动
- 完成后打开管理页面

底层仍然调用同一套 Go 内置安装逻辑。

## 不建议继续作为主入口的模式

### 不建议：继续以脚本为主要安装入口

原因：

- 不够产品化
- 问题排查成本高
- 受 Windows 环境差异影响明显
- 用户信任成本高
- “单文件交付”的目标被破坏

脚本可以保留给开发者调试，但不应继续作为用户主入口。

## 目标用户体验

### 首次使用

用户只需要：

1. 下载 `web-share.exe`
2. 双击运行
3. 选择语言
4. 勾选：
   - 安装右键菜单
   - 开机自启
   - 立即启动
5. 点击完成

程序自动完成：

- 写入默认语言
- 注册右键菜单
- 注册计划任务或开机自启项
- 启动 manager
- 启动 tray
- 弹出成功通知

### 后续维护

用户可以从托盘或管理页进入“设置/修复安装”，执行：

- 重新注册右键菜单
- 修改默认语言
- 开关开机自启
- 重启托盘
- 修复安装状态

### 卸载

用户可以：

- 在托盘菜单点击“卸载”
- 在管理页点击“卸载 Web Share”
- 或直接运行 `web-share.exe uninstall`

## 现有脚本能力与 Go 内置能力映射

当前脚本做的事，未来都应迁移为 Go 内置能力。

### `init-web-share.ps1`

迁移为：

- `web-share.exe install`

能力包括：

- 语言选择
- 写入默认语言
- 安装右键菜单
- 安装开机自启
- 立即启动

### `install-context-menu.ps1`

迁移为：

- `web-share.exe install --context-menu-only`
- 或 `web-share.exe repair --context-menu`

### `install-startup-task.ps1`

迁移为：

- `web-share.exe install --autostart`
- 或 `web-share.exe repair --autostart`

### `start-web-share.ps1`

迁移为：

- `web-share.exe start`

能力包括：

- 启动 manager
- 启动 tray
- 弹出成功通知

### `uninstall-all.ps1`

迁移为：

- `web-share.exe uninstall`

能力包括：

- 卸载右键菜单
- 卸载开机自启
- 停止 manager/tray
- 删除辅助文件
- 可选删除数据

## 当前代码层面的拆分建议

为了支持单文件安装器，建议先做几个内部模块边界。

### 1. `internal/install`

新增安装模块，负责：

- 检测安装状态
- 安装右键菜单
- 卸载右键菜单
- 安装开机自启
- 卸载开机自启
- 启动 manager
- 启动 tray
- 停止 manager
- 重启 tray
- 写入默认语言

建议导出统一接口：

```go
type InstallOptions struct {
    Language        string
    InstallContext  bool
    InstallAutostart bool
    StartNow        bool
    NotifyStart     bool
}
```

以及：

```go
func Install(opts InstallOptions) error
func Uninstall(removeData bool) error
func Repair(opts InstallOptions) error
func DetectStatus() Status
```

### 2. `internal/shell`

继续保留与 Windows 注册表、托盘进程、计划任务、浏览器拉起等系统交互相关能力，但只负责“低层动作”，不要承担安装流程编排。

### 3. `internal/app`

命令行入口层只做：

- 参数解析
- 调用 `internal/install`
- 调用 `manager` / `tray`

不要再把安装流程散落在多个脚本和多个命令分支里。

### 4. `internal/manager`

保留默认语言与设置存储逻辑，作为安装器/设置页复用的数据源。

## 建议新增命令

建议补充以下命令：

### `web-share.exe install`

支持参数：

- `--lang en-US|zh-CN`
- `--context-menu`
- `--autostart`
- `--start-now`
- `--notify`
- `--interactive`

默认行为建议：

- 双击时走 `install --interactive`
- CLI 明确指定时走非交互模式

### `web-share.exe uninstall`

支持参数：

- `--remove-data`
- `--quiet`

### `web-share.exe repair`

支持参数：

- `--context-menu`
- `--autostart`
- `--tray`
- `--language`

### `web-share.exe start`

负责：

- 启动 manager
- 启动 tray
- 可选通知

## GUI 与 CLI 的关系

建议不要一上来做复杂 GUI。

最合理的推进方式是：

### 第一阶段

先完成：

- Go 内置 `install/uninstall/repair/start`
- 所有脚本逻辑迁入 Go
- 脚本仅保留为开发辅助，甚至后续删除

### 第二阶段

在第一阶段稳定后，再决定是否补一个图形化安装向导。

这样可以避免：

- 一边做 GUI
- 一边还在调底层安装逻辑

导致问题定位困难。

## 托盘与管理页需要补的入口

为了让“安装器模式”闭环，建议再补两个入口。

### 托盘菜单建议新增

- `Settings`
- `Repair Installation`
- `Uninstall`

### 管理页建议新增

- 默认语言修改
- 右键菜单修复按钮
- 开机自启开关
- 托盘重启按钮
- 卸载入口

这样即使用户以后不看脚本、不看命令，也可以在程序内部完成维护。

## 状态检测建议

建议新增统一安装状态页，至少展示：

- Context menu: installed / missing
- Autostart: enabled / disabled
- Manager: running / stopped
- Tray: running / stopped
- Default language: en-US / zh-CN

这会极大降低排查成本。

## 分阶段实施建议

### Phase 1

- 建立 `internal/install`
- 把脚本逻辑迁入 Go
- 增加 `install/start/uninstall/repair` 命令
- 让双击 `web-share.exe` 能进入安装模式

### Phase 2

- 托盘增加设置/修复/卸载入口
- 管理页增加安装状态和修复入口
- 脚本降级为开发辅助入口

### Phase 3

- 可选增加 GUI 安装向导
- 可选生成正式安装包或便携模式说明

## 推荐结论

对于当前项目，最务实的路线是：

1. 先做“单文件内置命令安装器”
2. 不再让 `.ps1` 成为用户主入口
3. 再根据稳定度决定是否补 GUI 向导

这是当前复杂度和用户体验之间最平衡的方案。
