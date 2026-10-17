---
owner: @esengine
backup: @SivanCola
status: active
reviewed: 2026-10-17
---

# Workspace flow design QA / 工作目录流程视觉验收

final result: passed

## Visual truth / 视觉依据

- Selected target: the second displayed image from the latest design set.
- 选定方案：最近一组生成图中的第二张。
- Source: selected direct-search palette reference, reviewed locally before implementation.
- Implementation: production frontend on a loopback-only real Go host with disposable Git fixtures.
- 实现：真实 Go host 加载的生产前端构建；没有另建静态原型。

## Capture and normalization / 截图与归一化

Evidence directory / 证据目录：

Local design references remain review-only. Published before/after captures are
linked from the pull request; fixture content contains no account credentials.

设计参考图保留于本地验收；公开前后对比图见 PR，使用不含账号凭据的临时测试数据。

Published captures / 公开截图：

| View / 视图 | Before / 修改前 | After / 修改后 |
| --- | --- | --- |
| Light, wide / 浅色宽屏 | [Before](docs/images/workspace-flow/before-light-wide.png) | [After](docs/images/workspace-flow/after-light-wide.png) |
| Dark, wide / 深色宽屏 | [Before](docs/images/workspace-flow/before-dark-wide.png) | [After](docs/images/workspace-flow/after-dark-wide.png) |
| Light, narrow / 浅色窄屏 | [Before](docs/images/workspace-flow/before-light-narrow.png) | [After](docs/images/workspace-flow/after-light-narrow.png) |
| Dark, narrow / 深色窄屏 | [Before](docs/images/workspace-flow/before-dark-narrow.png) | [After](docs/images/workspace-flow/after-dark-narrow.png) |

Before captures use the original new-conversation configuration; after captures
show the unified entry still available in a bound worktree after task delivery.
The viewports are 1280 × 800 and 390 × 844 CSS pixels.

修改前为原有新会话配置，修改后展示任务交接后仍可使用的统一入口。
公开截图分别使用 1280 × 800 与 390 × 844 CSS 像素视口。

| Evidence / 证据 | Frame / 尺寸与状态 |
| --- | --- |
| `reference.png` | 1312 × 1199 PNG; selected source / 选定源图 |
| `branch-palette-detail.png` | 400 × 490 CSS px; 800 × 980 PNG at DPR 2 / 面板原生 PNG 截图 |
| `branch-palette.png` | 1280 × 720 CSS viewport; 2560 × 1440 PNG at DPR 2 / 桌面视口 |
| `comparison-complete.png` | Full panel and focused search/selection comparison / 整体及搜索、选中行局部对照 |
| `branch-palette-narrow.png` | 480 × 720 CSS viewport; emulation restored / 窄窗口检查后已恢复 |

The source component crop is approximately `(200,58,915,1005)` pixels.
The comparison renders both component widths at 360px. The source is scaled
by 360/915. The implementation uses 0.9 of its 400 CSS px width, equivalent
to scaling the 800px raster by 0.45. No content is stretched.

按面板区域统一到 360px 宽度后，在同一浏览器画面并排检查整体与局部。
源图不是完整应用视口，不对外围留白或窗口尺寸作像素等同判断。

Both states show a bound worktree, an empty branch search and a collapsed path.
The real fixture has five branches and a different directory name; those values
remain authoritative rather than being replaced with mock content.

两侧均为已绑定 worktree、空搜索、路径折叠。真实仓库多一个分支，项目名和分支名
来自真实状态；因此列表高度差和具体文字差异不作为设计缺陷。

## Fidelity surfaces / 必查项

| Surface / 项目 | Verdict / 结论 |
| --- | --- |
| Typography / 字体 | Existing Inter/Noto Sans SC stack and weight tokens; UI 13px, metadata 12px, helper 11px, status title 22px. / 复用现有字体、字重和字号。 |
| Spacing / 间距 | 400px palette, 16px padding, 44px branch targets; lightweight section dividers. / 保留方案的直接选择布局，以间距分组。 |
| Colors / 色彩 | Existing float, hairline, brand, muted and semantic status tokens. / 复用现有浮层、分隔线、强调色和状态色。 |
| Assets / 资源 | Existing StudioIcon source assets and shared CopyButton; no rasterized UI or new decorative drawings. / 复用图标与复制组件。 |
| Content / 文案 | Full raw identities stay in accessible names, tooltips, disclosures and copy actions. / 缩略显示不改变真实分支或路径。 |

The user's later instruction to match surrounding font sizes takes precedence
over the mock's larger typography. Existing action wording is retained for the
workspace trigger and heading. These are intentional product constraints.

后续明确要求字号尽量与其他位置统一，因此实现优先使用现有字号 token。
入口和标题保留「配置工作目录」这一既有名称，以保持动作一致。

## Findings and comparison history / 问题与对照历史

- No actionable P0/P1/P2 findings remain in the combined visual comparison.
- 正式整体和局部并排对照未发现残留的 P0/P1/P2 问题。
- Before final comparison, a long branch could receive a second end ellipsis.
  The value now uses a flexible prefix and fixed suffix, preserving its identity.
- 正式对照前修复了长分支二次尾部截断，末尾标识始终保留，完整值可悬停查看。
- The 480px check placed the panel at x=68…468, within the viewport, with no
  horizontal overflow. Expanded paths also stayed within the desktop viewport.
- 窄窗口及长路径展开均未遮断核心操作；失败详情展开时操作栏保持可见。

## Interaction checks / 交互验证

- Direct search, current selection, occupied branches, keyboard traversal,
  Escape focus return, failed reads, retries and full-path copy.
- 直接搜索、选中态、占用禁用、键盘导航、Escape 焦点恢复、读取失败重试、路径复制。
- Real Git: branch switch, selected creation ref, long wait, cancellation,
  process termination, rollback, checkout failure, retry, ready and task handoff.
- 真实 Git：切换、起点选择、长等待、取消、进程终止、回收、检出失败、重试和成功交接。
- Fresh production-bundle page: zero console errors and one original user task.
- 最终构建重新载入后控制台无错误，原用户任务只出现一次。

Creation, success, failure, cancellation and retained-result screens extend the
selected typography and surface hierarchy. No separate pixel-exact mock was
provided for these states; their behavior and consistency were inspected live.

其他状态页沿用选定方案的信息层级和字号；这些页面没有单独提供逐像素参考图，
因此按真实流程、状态准确性、信息完整性和视觉一致性验收。

## Validation and limits / 测试与边界

- Frontend after syncing Studio: 633 files, 5,241 tests passed; typecheck and
  production build passed.
- 前端全量、类型检查和生产构建通过。
- Backend: focused race tests passed in worktree, serve, control and boot.
- 后端相关四个包的定向并发测试通过。
- macOS browser plus real Git; model replies use a local test service.
- 使用 macOS 浏览器和真实 Git；模型回复由本地测试服务提供。
- Windows/Linux native execution and packaged Electron acceptance were not run.
- 未进行 Windows/Linux 原生运行或 Electron 安装包验收。

## Follow-up polish / 后续微调

No blocking visual work remains. Native shell qualification remains a separate
delivery check; this change does not alter shell chrome or window controls.

没有剩余的阻塞视觉问题。原生安装包验收属于独立交付检查，本次未修改窗口外壳。
