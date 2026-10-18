- 新增会话级 Agent 预设管理与选择，人设、工具及模块按会话绑定版本；基于人设页 #13588，见 #13613。
- 兼容：默认不启用预设；未选择及标准模式不绑定、不迁移人设，保留原日志格式。仅显式选择的其他组合使用版本边界保护。#13613
- Agent 组合选择与管理整合为输入框顶部状态行的统一菜单，展示来源和能力说明。#13613
- 内置模式收敛为标准与精简两种，管理页采用模式卡片和独立自定义区域。精简模式保留终端、图片读取和上下文压缩，已有会话继续使用原绑定。#13613
- 组合切换保留 ACP 额外工作目录；派发继承精确绑定，并明确拒绝跨工作区使用绑定。#13613

<!-- lang:en -->

- Add session Agent presets with versioned personas, tools and modules, building on #13588. #13613
- Keep compositions opt-in: unselected and Standard sessions keep the existing format without binding or persona migration. Only explicitly bound sessions use the version boundary. #13613
- Combine composition selection and management in one menu beside the composer run status, with source and capability notes. #13613
- Add Standard and Minimal cards above custom presets. Minimal keeps terminal, image reading and compaction; existing conversations retain their binding. #13613
- Preserve ACP additional directories across composition switches; dispatch carries the exact binding and refuses cross-workspace bindings. #13613
