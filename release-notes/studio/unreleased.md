下一版的变更（待发布） / Changes for the next release.

- 会话列表统一为 34px 单行省略，去掉额外行间距，悬浮可查看完整标题。“选择本机会话”入口移至“这台机器”的更多菜单，批量归档和删除功能保留。Refs #10651.
- Session rows use 34px single-line ellipsis with full titles on hover.
  “Select local sessions” moves to this machine's More actions menu; batch archive and delete remain available. Refs #10651.
- 桌面布局移除顶部用于开关工作区侧栏的三横线按钮；分隔条继续支持展开、收起和宽度调整，窄窗口仍保留顶部抽屉入口 (#12917)。
- Remove the desktop workspace-sidebar hamburger button; the divider still expands, collapses and resizes the sidebar, and narrow windows keep the top drawer entry (#12917).
- 外观新增「清白蓝」「茶绿」，支持浅色、深色及跟随系统，默认主题不变。修复默认预览串色和控件配色不一致。关联 #12909。
- Adds Clear Blue and Tea Green in light/dark and system modes, preserving the default. Fixes default-preview tint leakage and inconsistent control colours. Refs #12909.
- 动态天空主题移至「更多已安装主题」，激活的背景主题仍在首组显示。关联 #12909。
- Dynamic-sky themes move to More installed themes; the active backdrop remains in the first group. Refs #12909.
- Studio verifies both Linux package formats and requires all twelve release artifacts. GitHub Actions checks installed-shell communication, second launch, reinstall, clean exit and uninstall. (#12906)
- Studio 增强 Linux 两种安装包及全部 12 个发布产物的校验，GitHub Actions 验证安装后的通信、第二次启动、重装、正常退出和卸载。（#12906）
- 展开很长的执行过程时，头部会吸在顶部，随时可以直接折叠，不用再往上翻。关联 #12931。
- A long expanded process group keeps its header at the top while you scroll, so it can be folded without scrolling back up. Fixes #12931.
- 对话里的「执行过程」不再是带边框的卡片，改成和「思考」一样的淡色一行，展开后只在左侧留一条细线，让回复和结论更突出。关联 #12953。
- The activity group in a conversation is now a quiet line like the thinking row instead of a bordered card, with a thin rule at its left when open, so the reply and conclusion stand out. Fixes #12953.
- 新增外观设置「进行中列表保留已完成会话」：会话完成一轮后，可在侧栏「进行中」里再保留 10 分钟到 4 小时，默认关闭。关联 #12312。
- New appearance setting "Keep finished sessions in Running": a session that finishes a turn can stay in the
  sidebar's Running list for 10 minutes to 4 hours. Off by default. Fixes #12312.
- 修复：在 Windows 上点击任务完成通知，现在会把 Studio 窗口带到前台（此前在部分系统上只有任务栏图标闪烁）；点击过程也会写入日志，方便排查。关联 #12817。
- Fix: clicking a task-finished notification on Windows now brings Studio's window to the
  front (it only flashed the taskbar on some systems); the click path is logged. Refs #12817.
