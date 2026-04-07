# Problem Notes

## Windows 右键菜单级联子菜单不展开

### 现象

- `通过 Web 分享` 显示在右键菜单中，但不会展开二级菜单
- 有时点击父菜单后，Windows 提示“找不到对应的程序来处理”

### 最初实现

- 在父菜单键下同时使用 `SubCommands=""`
- 并且直接在父菜单下创建 `shell\readonly`、`shell\password`
- 历史版本还存在过父菜单自己的 `command` 子键残留

### 问题原因

- Explorer 对这套级联菜单解析比较脆弱
- 当父菜单没有被稳定识别为级联菜单时，会把它当成普通可点击菜单项
- 这时 Explorer 会尝试执行父菜单下的 `command`
- 如果 `command` 不存在或残留状态不一致，就会出现“找不到对应程序处理”

### 错误尝试

- 删除 `SubCommands`，只保留父菜单下的 `shell\子项`
- 结果父菜单彻底不再被识别为级联菜单，问题更明显

### 最终解决方案

- 改用 `ExtendedSubCommandsKey`
- 把文件和文件夹的子菜单定义分别放入独立的共享菜单库：
  - `WebShare.FileContextMenu`
  - `WebShare.DirectoryContextMenu`
- 父菜单只保留：
  - `MUIVerb`
  - `Icon`
  - `ExtendedSubCommandsKey`
- 安装时先删除旧的：
  - `HKCU\Software\Classes\*\shell\web-share`
  - `HKCU\Software\Classes\Directory\shell\web-share`
  - `HKCU\Software\Classes\WebShare.FileContextMenu`
  - `HKCU\Software\Classes\WebShare.DirectoryContextMenu`

### 当前稳定结构

- 文件：
  - `HKCU\Software\Classes\*\shell\web-share`
  - `ExtendedSubCommandsKey=WebShare.FileContextMenu`
- 文件夹：
  - `HKCU\Software\Classes\Directory\shell\web-share`
  - `ExtendedSubCommandsKey=WebShare.DirectoryContextMenu`

### 经验

- Windows 右键级联菜单不要继续依赖 `SubCommands + 父项下 shell\...` 的混合写法
- 对 Explorer 来说，`ExtendedSubCommandsKey` 更稳定，也更容易清理和复用
- 修这类问题时，必须先删除旧注册表节点再重装，仅覆盖写入往往不够

## 右键密码弹窗 VBS 编译失败

### 现象

- 右键文件夹选择 `设置上传密码后分享`
- 弹出 VBScript 错误：
  - 脚本 `C:\Users\zhjun\AppData\Local\WebShare\prompt-share.vbs`
  - 行 `8`
  - 字符 `54`
  - 错误 `缺少 )`
  - 代码 `800A03EE`

### 初看误区

- 脚本内容表面上语法正确
- 报错位置在 `InputBox(...)` 那一行
- 很容易误以为是引号转义或括号数量有问题

### 实际原因

- 生成的 `prompt-share.vbs` 使用了 `UTF-8` 写入
- VBScript 宿主对包含中文内容的 `UTF-8` 脚本兼容性很差
- 中文字符串在解析时被错误拆分，最终让编译器在 `InputBox(...)` 行报出误导性的“缺少 )”

### 排查方式

- 用 `cscript //nologo prompt-share.vbs ...` 直接执行脚本，读取真实编译错误
- 用十六进制查看脚本文件，确认文件实际是 `UTF-8` 而不是 `UTF-16 LE with BOM`

### 解决方案

- Go 侧生成脚本时改为 `UTF-16 LE with BOM`
- PowerShell 安装脚本写入 `.vbs` 时改成 `-Encoding Unicode`
- 修复后重新写回：
  - `C:\Users\zhjun\AppData\Local\WebShare\prompt-share.vbs`

### 经验

- VBScript 只要带中文，优先使用 `UTF-16 LE with BOM`
- 如果脚本“看起来没错”却在中文字符串行附近报语法错误，优先检查编码，而不是先怀疑括号或引号

## 管理页二维码不显示但前端无报错

### 现象

- 管理页中二维码区域为空白
- 浏览器 F12 没有明显报错
- 后端二维码生成逻辑已经执行，模板中也有 `<img src="{{.QRCodeDataURL}}">`

### 实际原因

- 二维码使用的是 `data:image/png;base64,...` 形式的内联图片 URL
- Go 的 `html/template` 会对 URL 做安全过滤
- 普通字符串类型的 `data:` URL 会被当作不安全内容处理
- 结果是模板渲染后图片地址不能正常用于 `<img src>`，但前端不一定给出明显报错

### 解决方案

- 把二维码字段类型从普通字符串改成 `template.URL`
- 让模板明确知道这是经过后端确认的可用 URL

### 经验

- 在 Go 模板里输出 `data:` URL 时，不要默认用 `string`
- 如果是后端可信生成的内联资源地址，应显式使用 `template.URL`
- 如果后续仍需更稳的方案，可以改成单独的二维码图片路由，例如 `/qr/<code>.png`

## 托盘图标启动后消失

### 现象

- 程序刚启动时托盘区域短暂出现图标
- 稳定运行后，托盘图标消失或变为空白
- 但托盘菜单本身仍然存在

### 初看误区

- 容易误以为图标文件损坏
- 或误以为 `embed` 资源加载失败

### 实际原因

- 通知使用的 `share.png` 对 `beeep` 是可用的
- 但 `getlantern/systray` 在 Windows 下最终走的是 `LoadImageW(..., IMAGE_ICON, ...)`
- 托盘图标在 Windows 语义上更适合使用 `.ico`
- 使用 `png` 时，即使启动早期可能短暂显示，进入稳定状态后仍可能被系统丢掉或加载失败

### 解决方案

- 保留 `share.png` 给通知使用
- 另外提供 `share.ico` 给托盘使用
- 托盘 `SetIcon(...)` 改为使用内嵌的 `.ico`

### 经验

- Windows 下托盘图标和通知图标不要混用同一种资源格式
- `Toast/通知` 可以继续使用 `png`
- `systray` 的托盘图标应优先使用 `ico`
