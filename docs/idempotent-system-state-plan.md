# 系统状态幂等化重构计划

## 目标

把当前 `web-share` 的系统集成与运行控制改成“目标状态驱动”模型，而不是“执行一次动作”模型。

本次重构重点解决：

1. 安装和卸载行为缺少严格状态收敛
2. 重复点击按钮或重复执行命令时行为不够稳定
3. 判定逻辑过宽，无法识别半安装、脏状态、旧版本残留
4. 后端缺少统一日志，用户在失败或部分成功时无法追溯
5. 前端反馈过粗，无法区分成功、警告、失败
6. 后端已有 warning 场景时，前端无法感知

最终目标是：

- 安装即“确保已安装”
- 卸载即“确保已卸载”
- 启动即“确保在运行”
- 停止即“确保已停止”
- 重复执行同一操作不会把系统推入异常状态
- 所有判定、执行、异常、缺失、跳过都写入日志
- 所有用户从前端触发的操作都能看到明确反馈

## 核心原则

### 1. 所有系统操作改为幂等操作

系统操作不再表达为：

- install
- uninstall
- start
- stop

而是表达为：

- ensure context menu installed
- ensure context menu removed
- ensure auto start enabled
- ensure auto start disabled
- ensure tray running
- ensure tray stopped
- ensure program stopped

这意味着：

- 如果目标已经满足，操作应视为成功
- 如果发现部分残留或脏状态，应先清理再收敛
- 如果目标对象不存在，不应静默吞掉
- 如果目标对象不存在但目标态已经满足，应返回 `warning` 而不是 `error`

### 2. 所有操作先检查，再执行，再复检

每个系统动作统一采用三步：

1. 检查当前状态
2. 执行收敛动作
3. 再次检查确认目标态

不允许仅凭“调用成功”就认为状态达成。

### 3. 所有异常与边界都必须留痕

所有系统状态相关逻辑都必须打日志，包括：

- 进入操作
- 当前状态检查结果
- 发现残留
- 发现缺失
- 执行清理
- 执行安装
- 执行卸载
- 执行启动
- 执行停止
- 复检结果
- 最终成功
- 最终警告
- 最终失败

### 4. 用户触发的操作必须有前端可感知反馈

如果操作来自 Web 页面按钮：

- 成功时前端要提示成功
- 有 warning 时前端要提示警告
- 失败时前端要提示失败

不能只在后端写日志而前端静默。

### 5. 结构与实现分离，骨架优先可测试

系统重构时需要明确拆成两层：

- 结构层
- 实现层

其中：

- 结构层负责系统骨架、流程编排、目标状态收敛顺序、结果聚合
- 实现层负责具体平台行为，例如注册表读写、进程启动、托盘控制、文件删除、日志落盘

结构层不能直接依赖具体平台 API，而应只依赖抽象接口。

目标是：

- 骨架层可以纯单元测试
- 平台实现可以独立替换
- 业务决策和 Windows 细节不耦合

### 6. 优先测试编排与可观察行为，而不是内部细节

测试重点应放在：

- 输入状态
- 收敛决策
- 调用顺序
- warning / error 语义
- 最终目标态

而不是测试某个私有函数内部细节或某个具体语句是否执行。

对本次重构来说，测试首先要验证：

- 已满足目标态时是否稳定返回成功
- 缺失状态时是否返回 warning
- 脏状态时是否先清理再收敛
- 重复执行是否仍能保持目标态
- 结构层是否正确调用实现层接口

### 7. 外部副作用必须可替换

凡是外部副作用，都必须通过接口抽象出来，不能直接散落在编排逻辑里。

至少包括：

- 注册表访问
- 文件系统访问
- 进程启动与停止
- 托盘状态控制
- 程序 shutdown
- 当前 exe 路径获取
- 时间
- 日志输出

这样才能做到：

- 结构层纯单测
- 平台行为做少量集成测试
- warning / error 分支稳定覆盖

## 当前问题

### 1. 右键菜单安装判定过宽

当前 `ContextMenuInstalled()` 只检查顶层 key 是否存在，不能识别：

- 子命令 key 缺失
- 命令值不完整
- 命令值仍指向旧版 `prompt-share.vbs`
- 旧版和新版配置混杂

结果是：

- 自动补装逻辑可能误判“已安装”
- 老用户升级后不会自动迁移到新命令结构

### 2. 卸载行为仍然偏“尽量删”

例如右键菜单、旧脚本、自启动项的卸载逻辑还没有形成统一的：

- 先检查
- 不存在则 warn
- 存在则删除
- 删除后复检

模型。

### 3. 停止程序的前后端反馈不一致

当前 `stop_program` 会在请求生命周期内异步 shutdown，前端还在等待 JSON 响应。

结果可能出现：

- 实际已进入停止流程
- 前端却显示 `Request failed`

这不符合“用户有感知”的要求。

### 4. 缺少统一日志层

现在只有零散的错误返回和通知，没有完整的系统状态审计日志。

缺失内容包括：

- 谁触发了操作
- 当前状态是什么
- 为什么产生 warning
- 操作是否已经达成目标态

### 5. 前端只有成功/失败，没有 warning 通道

当前异步表单接口默认只处理：

- `ok`
- `message`

但系统状态型操作天然存在第三类结果：

- 已达目标态，但发现缺失项
- 已自动修复旧残留
- 已完成，但清理到了旧版本配置

这些都应该显示为 warning，而不是混成成功或失败。

## 重构方向

## 1. 引入统一的系统状态模型

新增一组结构化状态定义，覆盖以下对象：

- 右键菜单
- 自启动
- 托盘
- 整个程序
- 默认语言

建议定义：

```go
type StateLevel string

const (
    StateOK      StateLevel = "ok"
    StateWarn    StateLevel = "warn"
    StateError   StateLevel = "error"
)

type CheckMessage struct {
    Level   StateLevel
    Code    string
    Message string
}

type CheckResult struct {
    OK       bool
    Dirty    bool
    Missing  bool
    Messages []CheckMessage
}
```

不同对象再定义更具体的结构，例如：

- `ContextMenuState`
- `AutostartState`
- `TrayState`
- `ProgramState`

其中要能表达：

- 完整安装
- 完整卸载
- 缺失
- 半安装
- 脏状态
- 命令不匹配
- 旧版残留

## 2. 引入统一的操作返回模型

建议所有系统动作不再直接返回裸 `error`，而是返回结构化结果：

```go
type OperationResult struct {
    OK        bool
    Changed   bool
    Warnings  []string
    Errors    []string
    Messages  []string
}
```

语义建议：

- `OK=true, Changed=true`
  - 执行了变更并成功达到目标态
- `OK=true, Changed=false`
  - 目标态原本已经满足
- `Warnings`
  - 包含“缺失但已是目标态”“检测到旧残留并已清理”等信息
- `Errors`
  - 表示最终未达到目标态

## 3. 新增统一日志系统

建议新增模块：

- `internal/logx`

初期职责：

- 统一格式化日志
- 写入本地文件
- 输出 `info / warn / error / audit`

建议日志字段至少包含：

- 时间
- 级别
- 操作名
- 对象名
- 触发来源
- 用户是否来自前端
- 当前检查摘要
- 目标状态
- 执行动作
- 结果
- warning 列表
- error 列表

建议按以下来源打点：

- CLI
- 托盘
- Web 按钮
- 启动自动补装
- 卸载清理

建议初期写入：

- `%LOCALAPPDATA%\WebShare\logs\YYYY-MM-DD.log`

后续再决定是否做轮转或大小控制。

## 4. 将系统集成改成 inspect + reconcile 模型

建议新增一个系统状态协调层，例如：

- `internal/systemstate`

职责拆分：

- `inspect.go`
  - 只做检查，不改系统
- `reconcile.go`
  - 根据目标态做收敛
- `result.go`
  - 定义状态和返回结构
- `log.go`
  - 统一把检查与动作写入日志

这个模块需要再内部区分：

- 骨架层
  - 只依赖接口
  - 负责 inspect -> decide -> reconcile -> recheck 的流程
- 平台实现层
  - 提供 Windows 具体能力
  - 负责访问注册表、托盘、进程、文件系统

不允许在骨架层里直接出现具体注册表路径写入、直接 `os.Remove`、直接启动进程等行为。

### Inspect 层

建议新增：

- `InspectContextMenu(exePath string) (ContextMenuState, error)`
- `InspectAutostart(taskName string, exePath string) (AutostartState, error)`
- `InspectTray() (TrayState, error)`
- `InspectProgram() (ProgramState, error)`

### Reconcile 层

建议新增：

- `EnsureContextMenuInstalled(exePath, lang string) OperationResult`
- `EnsureContextMenuRemoved() OperationResult`
- `EnsureAutostartEnabled(exePath, taskName, lang string, notify bool) OperationResult`
- `EnsureAutostartDisabled(taskName string) OperationResult`
- `EnsureTrayRunning(exePath string) OperationResult`
- `EnsureTrayStopped() OperationResult`
- `EnsureProgramStopped() OperationResult`

统一要求：

- 先 inspect
- 必要时 clean
- 执行 install/remove/start/stop
- 复检
- 产出结构化结果
- 写日志

### 结构层接口建议

建议引入一组基础接口，让骨架层只围绕接口编排：

```go
type ContextMenuPort interface {
    Inspect(exePath string) (ContextMenuState, error)
    Clean() error
    Install(exePath, lang string) error
    Remove() error
}

type AutostartPort interface {
    Inspect(taskName, exePath string) (AutostartState, error)
    Enable(exePath, taskName, lang string, notify bool) error
    Disable(taskName string) error
}

type TrayPort interface {
    Inspect() (TrayState, error)
    Start(exePath string) error
    Stop() error
}

type ProgramPort interface {
    Inspect() (ProgramState, error)
    Stop() error
}

type Logger interface {
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
}
```

实际命名可以调整，但设计原则不变：

- 结构层依赖抽象
- 实现层提供 Windows 版本适配
- 单测时可用 fake/mock 替代真实实现

## 5. 严格化右键菜单判定

这是优先级最高的一个点。

### 检查内容

右键菜单检查不能只看：

- `HKCU\Software\Classes\*\shell\web-share`
- `HKCU\Software\Classes\Directory\shell\web-share`

还必须检查：

- `WebShare.FileContextMenu`
- `WebShare.DirectoryContextMenu`
- 子命令 `readonly`
- 子命令 `password`
- `command` 默认值
- 命令值是否匹配当前 `exePath`
- 命令值是否匹配当前 `prompt-share` 方案
- 是否仍引用旧版 `prompt-share.vbs`

### 目标行为

安装前：

- 如果发现旧残留，先记 warning
- 清理旧 key
- 再完整重建

卸载前：

- 逐项检查 key 是否存在
- 缺失项记 warning
- 存在项删除
- 删除后复检

### 前端反馈

如果用户从系统页面点“安装右键菜单”：

- 已安装且命令匹配：提示“已处于目标状态”
- 检测到旧残留并已重装：提示 success + warning
- 缺失项不完整并已修复：提示 success + warning
- 最终仍不完整：提示 error

## 6. 严格化自启动判定

当前不能只看注册表值是否存在，还要检查：

- 名称是否存在
- 值是否指向当前 exe
- 参数是否符合当前标准命令
- 是否需要覆盖旧格式

### 目标行为

启用自启动：

- 如果不存在，创建
- 如果存在但命令不一致，覆盖并记 warning
- 如果已完全一致，返回“已处于启用状态”

禁用自启动：

- 如果不存在，返回 warning
- 如果存在，删除并复检

## 7. 严格化托盘与程序停止行为

### 托盘

建议把托盘操作也改成目标态：

- `EnsureTrayRunning`
- `EnsureTrayStopped`

停止前：

- 检查托盘是否存在
- 不存在时记 warning

启动前：

- 检查是否已运行
- 已运行则直接返回成功且 `Changed=false`

### 程序停止

`stop_program` 需要调整为更适合前端感知的模型。

建议两种可选实现：

#### 方案 A

HTTP 先返回确认响应，再异步停止程序。

返回内容：

- `ok=true`
- `message="Program stop requested."`
- `warnings=[]`

然后后台执行：

- stop tray
- shutdown manager

优点：

- 前端不会误报失败

#### 方案 B

保留当前异步关闭，但前端对该按钮特殊处理：

- 提交后立即显示“正在停止程序”
- 如果请求中断，不直接显示失败
- 改成提示“程序可能已停止，请确认托盘是否消失”

推荐采用方案 A。

## 8. 前端反馈改为 success / warning / error 三态

建议把系统设置页和 setup 页的异步响应统一改成：

```json
{
  "ok": true,
  "message": "Context menu reinstalled successfully.",
  "warnings": [
    "Detected legacy VBS command and replaced it."
  ],
  "status": { ... }
}
```

前端展示规则：

- `ok=true` 且 `warnings=[]`
  - 绿色提示
- `ok=true` 且 `warnings` 非空
  - 黄色提示
  - 主消息 + warning 列表
- `ok=false`
  - 红色提示
  - 错误列表优先展示

### 必须覆盖的用户感知场景

- 要删除的内容不存在
- 要安装的位置发现旧残留
- 已经处于目标状态
- 自动修复了脏状态
- 最终失败

## 9. CLI 与 Web 的行为统一

同一个系统动作，无论从哪里触发，都应该走同一个 reconcile 实现。

例如：

- `web-share install`
- `web-share uninstall`
- Web 系统页面按钮
- 启动时自动补装右键菜单

都不应该各自维护一套分叉逻辑。

建议做法：

- `install` 命令调用统一的 `Ensure*`
- `uninstall` 命令调用统一的 `Ensure*Removed/Disabled/Stopped`
- Web handler 仅负责参数解析和结果透传
- 启动自动补装也只调用统一接口

## 10. 文档与文案同步

代码重构后，需要同步更新：

- [README.md](C:/Users/zhjun/Desktop/code/web-share/README.md)
- [usage.md](C:/Users/zhjun/Desktop/code/web-share/docs/usage.md)
- [system-behavior.md](C:/Users/zhjun/Desktop/code/web-share/docs/system-behavior.md)

需要补充的文档点：

- 所有系统动作均为幂等行为
- 前端会展示 warning
- 日志目录位置
- 发现旧配置时会自动迁移或清理
- 停止程序操作的反馈语义

## 模块拆分建议

## 1. `internal/logx`

职责：

- 统一日志写入
- 提供 `Info/Warn/Error/Audit`
- 输出结构化文本

## 2. `internal/systemstate`

职责：

- inspect
- reconcile
- 状态定义
- 操作结果定义
- 骨架流程编排
- 平台接口抽象

建议文件：

- `state.go`
- `result.go`
- `ports.go`
- `service.go`
- `context_menu_windows.go`
- `autostart_windows.go`
- `tray_windows.go`
- `program_windows.go`
- `reconcile_windows.go`

建议进一步分层：

- `ports.go`
  - 定义依赖接口
- `service.go`
  - 放结构层编排
- `*_windows.go`
  - 放 Windows 具体实现

这样每个 `Ensure*` 可以有两类测试：

- 结构层单元测试
- Windows 实现层集成测试

## 3. `internal/manager`

职责调整：

- handler 层只做请求解析、调用 service、返回 JSON
- 不再直接承担大量系统状态决策

## 4. `internal/install`

职责调整：

- CLI 安装卸载入口
- 基于统一 `Ensure*` 接口编排
- 不再自己拼零散清理逻辑

## 实施阶段

## Phase 1：日志与返回模型

目标：

- 引入 `logx`
- 定义 `OperationResult`
- Web 异步接口支持 `warnings`

交付物：

- 日志目录可写
- 基础日志格式稳定
- 前端支持 warning 展示

## Phase 2：右键菜单状态重构

目标：

- 严格检查右键菜单完整性
- 安装前清理旧残留
- 卸载前检查并给出 warning

交付物：

- 新的 `InspectContextMenu`
- 新的 `EnsureContextMenuInstalled`
- 新的 `EnsureContextMenuRemoved`

## Phase 3：自启动状态重构

目标：

- 启用/禁用都改成目标态驱动
- 命令不一致时自动覆盖并告警

交付物：

- `InspectAutostart`
- `EnsureAutostartEnabled`
- `EnsureAutostartDisabled`

## Phase 4：托盘与程序停止语义重构

目标：

- 托盘启动/停止幂等化
- 程序停止接口不再给前端制造假失败

交付物：

- `EnsureTrayRunning`
- `EnsureTrayStopped`
- `EnsureProgramStopped`
- `stop_program` 前端交互修复

## Phase 5：安装/卸载总流程收敛

目标：

- `install` 和 `uninstall` 改成统一编排层
- 安装即全量重装
- 卸载即全量清理

交付物：

- `install` 统一调用 `Ensure*`
- `uninstall` 统一调用 `Ensure*`
- 启动自动补装改走同一实现

## Phase 6：测试与文档

目标：

- 补齐幂等性和 warning 语义测试
- 更新文档

交付物：

- 单元测试
- 集成测试
- 文档更新

## 测试计划

必须补充以下测试：

### 1. 幂等性测试

- 连续执行两次安装，最终状态一致
- 连续执行两次卸载，最终状态一致
- 连续执行两次启用自启动，最终状态一致
- 连续执行两次禁用自启动，最终状态一致
- 连续执行两次启动托盘，最终状态一致
- 连续执行两次停止托盘，最终状态一致

### 1.1 骨架层单元测试

优先补结构层测试，不依赖真实 Windows 环境。

必须覆盖：

- inspect 返回已满足目标态时，是否直接成功返回
- inspect 返回 dirty 时，是否先清理再安装
- inspect 返回 missing 时，是否按目标态给出 warning 或执行修复
- reconcile 成功但复检失败时，是否返回 error
- 多个 warning 是否正确聚合
- 是否调用了预期的接口顺序

这些测试要使用 fake/mock port，而不是直接操作注册表或真实进程。

### 2. 脏状态测试

- 只存在顶层 key，不存在子命令 key
- 子命令 key 存在，但命令值不匹配
- 命令值仍指向 `prompt-share.vbs`
- 新旧配置共存

期望：

- inspect 能识别 dirty
- reconcile 能收敛到正确状态
- 结果中有 warning

### 3. 缺失状态测试

- 卸载时目标对象不存在
- 停止托盘时托盘未运行
- 禁用自启动时注册表值不存在

期望：

- 操作返回 `ok=true`
- 带 warning
- 日志中有 warning 记录

### 4. 前端反馈测试

- 成功时展示 success
- warning 时展示 warning
- 失败时展示 error
- `stop_program` 不再因为连接断开误报失败

### 5. 实现层集成测试

实现层测试数量可以少，但需要覆盖关键真实路径：

- 右键菜单真实注册表安装与卸载
- 自启动注册表值覆盖与删除
- 旧 `prompt-share.vbs` 残留识别
- 托盘状态探测与停止

原则是：

- 单元测试负责大多数决策分支
- 集成测试只负责关键平台能力校验

### 6. 测试设计原则

本次重构测试遵循以下原则：

- 优先测试结构层，不优先测试实现细节
- 优先测试可观察行为，不绑定私有内部实现
- 优先测试幂等重复执行与脏状态收敛
- 外部副作用全部替身化后再做单元测试
- 结构层高覆盖，平台实现层少量关键集成测试

## 验收标准

本计划完成时，应满足：

- `install` 重复执行不会产生脏状态
- `uninstall` 重复执行不会报错退出
- 系统页面重复点击同一按钮不会造成状态混乱
- 所有系统动作都能返回 success/warning/error 三态
- 所有系统动作都写日志
- 右键菜单检查能够识别旧版脚本残留
- 安装前会清理旧残留并重建
- 卸载前会检查缺失项并反馈 warning
- 程序停止操作不会再给前端制造假失败
- 文档明确说明日志位置、warning 语义、幂等行为

## 建议的实施顺序

建议严格按以下顺序推进：

1. 先引入 `OperationResult` 和日志系统
2. 再重构右键菜单 inspect/reconcile
3. 再重构自启动 inspect/reconcile
4. 再重构托盘与程序停止语义
5. 最后统一 `install/uninstall/startup auto-fix` 编排
6. 收尾补测试和文档

不建议先改所有入口再补日志，因为那样中间阶段难以观察真实状态。
