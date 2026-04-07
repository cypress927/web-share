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
