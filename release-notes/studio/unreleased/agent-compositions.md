- 新增会话级 Agent 预设管理与选择，人设、工具及模块按会话绑定版本；基于人设页 #13588，见 #13613。
- 兼容：默认不启用预设；未选择及标准模式不绑定、不迁移人设，保留原日志格式。仅显式选择的其他组合使用版本边界保护。#13613
- Agent 组合选择与管理整合为输入框顶部状态行的统一菜单，展示来源和能力说明。#13613
- 内置模式收敛为标准与精简两种，管理列表按来源分组。精简模式保留终端、图片读取和上下文压缩，已有会话继续使用原绑定。#13613
- 组合切换保留 ACP 额外工作目录；派发继承精确绑定，并明确拒绝跨工作区使用绑定。#13613
- 项目预设的 MCP/扩展参数仅可调整超时，不能改写启动参数、地址或凭据环境；旧的不安全绑定保留历史但拒绝执行。#13613
- 拒绝跨项目移动绑定组合的会话，保留原会话，避免丢失绑定后以其他能力续写。#13613
- 远程新会话透传所选模式，同时保留并发打开与重复点击的保护。#13613
- 预设写入与组合切换要求配置编辑授权；安全 Hook 与权限规则不受组合选择影响，旧绑定恢复和会话内策略编辑同样保持保护。#13613

- 模式选择器默认直接显示，新会话保持标准模式；重复新建会话保留空白窗格的已选模式。#13613
- 原生归档直接交给后端解析，错误按码翻译；内置副本使用窗口语言，标识字段跟随主题。#13613
- 新建和编辑预设使用独立编辑页，常用字段直接显示，能力与高级配置按需展开；提供固定保存栏、未保存退出确认及失败草稿保留。#13613

<!-- lang:en -->

- Add session Agent presets with versioned personas, tools and modules, building on #13588. #13613
- Keep compositions opt-in: unselected and Standard sessions keep the existing format without binding or persona migration. Only explicitly bound sessions use the version boundary. #13613
- Combine composition selection and management in one menu beside the composer run status, with source and capability notes. #13613
- Group built-in, personal and project presets in the management list. Minimal keeps terminal, image reading and compaction; existing conversations retain their binding. #13613
- Preserve ACP additional directories across composition switches; dispatch carries the exact binding and refuses cross-workspace bindings. #13613
- Limit project MCP/extension parameters to timeouts; refuse launch, endpoint or credential-environment overrides, including in old executable bindings, while keeping history readable. #13613
- Refuse moving bound conversations to another project before copying or deleting them, preserving the original immutable composition. #13613
- Carry the selected mode into new remote conversations while preserving concurrent-open ordering and duplicate-click protection. #13613
- Require configuration-edit authority for preset writes and composition switches; keep safety hooks and permission rules in force across selection, old-binding restore and live policy edits. #13613
- Show the mode picker by default and keep Standard as the new-conversation default; preserve blank-pane modes on repeated New session. #13613
- Upload native archives to the host decoder; localize refusals, copied descriptions and themed identifier fields. #13613
- Create/edit presets on a separate page, with expandable capabilities and advanced settings, persistent Save/Cancel, discard confirmation and retained drafts after failed saves. #13613
