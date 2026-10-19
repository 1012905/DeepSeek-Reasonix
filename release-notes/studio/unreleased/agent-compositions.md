- 新增会话级 Agent 预设管理与选择，人设、工具及模块按会话绑定版本；基于人设页 #13588，见 #13613。
- 兼容：默认不启用预设；未选择及日常搭档不绑定、不迁移人设，保留原日志格式。仅显式选择的其他组合使用版本边界保护。#13613
- Agent 组合选择与管理整合为输入框顶部状态行的统一菜单，展示来源和能力说明。#13613
- 内置人设更名为「日常搭档」「轻装搭档」，介绍更贴近日常协作；参考 Harness 的简短身份提示，保持原提示词、工具与已有会话绑定。双列卡片在窄屏下单列，提供说明、使用方法与复制。#13613
- 组合切换保留 ACP 额外工作目录；派发继承精确绑定，并明确拒绝跨工作区使用绑定。#13613
- 项目预设的 MCP/扩展参数仅可调整超时，不能改写启动参数、地址或凭据环境；旧的不安全绑定保留历史但拒绝执行。#13613
- 拒绝跨项目移动绑定组合的会话，保留原会话，避免丢失绑定后以其他能力续写。#13613
- 远程新会话透传所选模式，同时保留并发打开与重复点击的保护。#13613
- 预设写入与组合切换要求配置编辑授权；安全 Hook 与权限规则不受组合选择影响，旧绑定恢复和会话内策略编辑同样保持保护。#13613

- 模式选择器默认关闭，可在人设管理页开启并通过设置搜索定位；初始使用日常搭档。卡片继续设置本机新任务默认，隐藏选择器不改变会话模式。#13613
- 原生归档直接交给后端解析，错误按码翻译；内置副本使用窗口语言，标识字段跟随主题。#13613
- 设置导航改为人设；独立编辑页直接展示两段提示词、运行环境、范围和 ID，能力按需展开；保留固定保存栏与未保存保护。#13613

- 加固预设文件路径与分配边界，内部继承绑定不放入 HTTP 请求对象；修正 Windows 上保留 Hook 测试的路径引用。#13613

<!-- lang:en -->

- Harden preset filesystem and allocation boundaries; pass inherited bindings outside the HTTP request DTO. Quote retained hook test paths portably for Windows. #13613

- Add session Agent presets with versioned personas, tools and modules, building on #13588. #13613
- Keep compositions opt-in: unselected and Everyday partner sessions keep the existing format without binding or persona migration. Only explicitly bound sessions use the version boundary. #13613
- Combine composition selection and management in one menu beside the composer run status, with source and capability notes. #13613
- Rename built-ins to Everyday partner and Lightweight partner with warmer copy. Keep prompts, tools and bindings unchanged; responsive cards retain details, usage and copy. #13613
- Preserve ACP additional directories across composition switches; dispatch carries the exact binding and refuses cross-workspace bindings. #13613
- Limit project MCP/extension parameters to timeouts; refuse launch, endpoint or credential-environment overrides, including in old executable bindings, while keeping history readable. #13613
- Refuse moving bound conversations to another project before copying or deleting them, preserving the original immutable composition. #13613
- Carry the selected mode into new remote conversations while preserving concurrent-open ordering and duplicate-click protection. #13613
- Require configuration-edit authority for preset writes and composition switches; keep safety hooks and permission rules in force across selection, old-binding restore and live policy edits. #13613
- Hide the mode picker by default; opt in from Persona management or settings search. Keep Everyday partner initially and preserve card defaults and existing conversation modes. #13613
- Upload native archives to the host decoder; localize refusals, copied descriptions and themed identifier fields. #13613
- Rename settings navigation to Persona; show prompt fields, runtime context, scope and ID directly in the editor, with expandable capabilities and persistent Save/Cancel. #13613
