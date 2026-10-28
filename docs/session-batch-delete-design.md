---
owner: @SivanCola
backup: @esengine
status: active
reviewed: 2026-10-27
---

# 会话批量删除能力 实现方案

> 场景：日常开发 — 为会话（Session/Conversation）功能增加批量删除能力。
> 范围：一、在现有会话列表中增加多选勾选与批量删除（支持全选/取消全选）；二、新增独立「会话管理」界面，集中展示所有会话并支持筛选、批量选择与批量删除。

---

## 1. 现状分析

- **删除能力现状**：后端提供两个单条删除端点 —— `POST /delete-session`（按会话 name）与 `POST /tree/sessions/remove`（按 path，侧边栏实际使用），均一次只删一个会话。
- **前端批量能力现状**：`ui/SessionBatch.tsx` 已存在多选 + 批量删除 UI，但其实现是 **前端循环对每个选中会话各发一次 `hub.removeSession(path)`**（N 次 HTTP 请求），缺少真正的批量端点；且仅有**错误反馈**（`onError`），删除成功无提示。
- **独立管理入口现状**：**不存在**。会话仅散落在侧边栏（按项目分组、每项目仅预览前若干条、可分页）中，缺少集中管理视图与筛选能力。
- **二次确认/反馈组件**：已有可复用组件 —— `.wsconfirm`（`role="alertdialog"` 二次确认弹窗）、Toast 系统（`pushToast` 失败 / `pushNote` 成功）。无需新建。

结论：本方案**复用并增强**现有 `SessionBatch`（Part 1），并**新增**真正的批量后端端点 + 独立管理界面（Part 2）。

---

## 2. 数据模型

会话**不是 SQL 表**，而是磁盘上的转录 `.jsonl` + sidecar（`BranchMeta`）。代码中的核心结构：

- 后端 `internal/state/sessionstore/session_listing_helpers.go`：
  - `SessionInfo{ Path, Title, CustomTitle, Archived, Pinned, Unread, Turns, LastActivityAt, ... }`
- 前端 `desktop/frontend-next/src/port/hub.ts`：
  - `TreeSession{ path, name, title?, turns?, lastActiveAt?, runtimeId?, archived?, pinned?, unread?, ... }`
  - `TreeWorkspace{ root, name, sessions: TreeSession[] }`

### 新增/变更的数据结构

仅新增**请求体**，无 schema 变更（会话本身仍是文件）：

```go
// POST /tree/sessions/remove-many  请求体
type body struct { Paths []string `json:"paths"` }
// 响应：成功 204 No Content；任一路径非法/越权 → 400/403；运行中 → 409 类拒绝；整批原子失败。
```

```ts
// 前端 HubPort 新增方法
removeSessions(paths: string[]): Promise<void>;
```

---

## 3. 接口设计

### 3.1 后端：新增真正的批量删除端点

文件：`internal/frontend/serve/hub_tree.go`

路由（`registerTreeRoutes` 中新增）：

```go
mux.HandleFunc("POST /tree/sessions/remove-many", h.removeSessions)
```

处理器 `removeSessions` **完全复用**单条删除的安全逻辑：
1. 先校验并解析**所有**路径（非法路径立即拒绝，避免「删一半再失败」）；
2. 对正在运行的会话整体拒绝（`releaseOrRefuse`）；
3. 越出已知 workspace 的路径拒绝（`ownsSessionDir`）；
4. 将每个 lead 展开为「被父会话覆盖的恢复副本」集合并去重；
5. **先对全部待删转录加锁（guard），再依次擦除**（lead 最后删）；任一 busy 则整批回滚释放锁。

```go
func (h *Hub) removeSessions(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Paths []string `json:"paths"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Paths) == 0 {
		badBody(w)
		return
	}

	type picked struct{ dir, path string }
	leads := make([]picked, 0, len(body.Paths))
	seen := make(map[string]bool, len(body.Paths))
	open := h.openSessions()
	excluded := make([]string, 0, len(open))
	for p := range open { excluded = append(excluded, p) }

	for _, raw := range body.Paths {
		path, err := filepath.Abs(strings.TrimSpace(raw))
		if err != nil || !store.IsSessionTranscriptName(filepath.Base(path)) {
			refuse(w, http.StatusBadRequest, codeSessionBadPath, "a session path could not be resolved", nil)
			return
		}
		if seen[path] { continue }
		seen[path] = true
		if id := open[sessionstore.CanonicalSessionPath(path)]; id != "" &&
			!h.releaseOrRefuse(w, r, "session.running", "a conversation is running; stop it first",
				h.panesWhere(func(rt *Runtime) bool { return rt.ID == id })) {
			return
		}
		dir := filepath.Dir(path)
		if !h.ownsSessionDir(dir) {
			refuse(w, http.StatusForbidden, "session.outside_workspace", "path outside a known workspace", nil)
			return
		}
		leads = append(leads, picked{dir, path})
	}

	erase := make(map[string]string)
	for _, lead := range leads {
		paths := []string{lead.path}
		sibs, err := sessionstore.RecoveryLineagePaths(lead.path, excluded...)
		if err != nil { writeErr(w, http.StatusInternalServerError, err); return }
		for _, p := range sibs {
			if p != lead.path && sessionstore.RecoveryBranchCoveredByParent(p, lead.dir) {
				paths = append(paths, p)
			}
		}
		for _, p := range paths { erase[p] = lead.dir }
	}
	ordered := make([]picked, 0, len(erase))
	for path, dir := range erase { ordered = append(ordered, picked{dir, path}) }

	guards := make([]*sessionstore.SessionRemovalGuard, 0, len(ordered))
	for _, tgt := range ordered {
		guard, ok := acquireRemovalGuard(w, tgt.path)
		if !ok { for _, g := range guards { g.Release() }; return }
		guards = append(guards, guard)
	}
	for i := range slices.Backward(ordered) {
		if !eraseGuardedSession(w, ordered[i].dir, ordered[i].path, guards[i]) {
			for _, g := range guards[:i] { g.Release() }
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
```

> 兼容性：`SessionBatch` 仍可逐条调用 `removeSession`；新端点用于一次性批量删除，失败时前端退化为逐条以保留「哪些失败」的粒度（见 3.3）。

### 3.2 前端 Port 层

`src/port/hub.ts`（`HubPort` 接口 + `SseHub` 实现）：

```ts
// 接口
removeSessions(paths: string[]): Promise<void>;

// SseHub
async removeSessions(paths: string[]) {
  if (paths.length === 0) return;
  await this.post<void>("/tree/sessions/remove-many", { paths });
}
```

`src/port/mock_hub.ts`（测试桩）：

```ts
removeSessions(_paths: string[]) { return Promise.resolve(); }
```

### 3.3 复用：`SessionBatch` 增强（Part 1 成功反馈 + 批量端点）

- 成功删除后新增 `pushNote(t("已删除 {n} 个会话", { n: deleted }))`，补齐成功反馈；
- **主删除路径已切换为批量端点**：先关闭所选、仍在运行会话对应的面板，再一次性调用 `hub.removeSessions(paths)`（1 次请求）；若整批被拒绝（如仍有会话在运行），自动降级为逐条 `hub.removeSession`，从而保留「具体哪些路径删除失败」的粒度，失败项留在选择集并由 `onError` 上报。

---

## 4. 交互流程

### Part 1 — 现有会话列表内批量删除（侧边栏）

1. 用户点侧边栏「更多操作 → 选择本机会话」进入多选模式（`selection: Set<path>`）。
2. 每行出现勾选框；点击行即切换选中；工具栏「全选可见 / 取消全选」一键选中**当前已加载**行。
3. 点「删除」→ 弹出 `.wsconfirm` 二次确认（`role="alertdialog"`），列出将被删会话名称。
4. 确认 → 批量删除（优先 `removeSessions`，失败降级逐条）→ `reload()` 刷新树 → 成功 `pushNote` / 失败 `onError` 上报 ErrorBar。
5. 运行中会话的勾选被禁用，确认阶段亦会被后端拒绝，保证安全。

> 说明：侧边栏按项目分页（`previewLimit`），其「全选」仅覆盖已加载行。需要**全量选择**请用 Part 2 的管理界面。

### Part 2 — 独立「会话管理」界面

入口：侧边栏「更多操作 → 管理会话」打开全屏模态（`role="dialog"`，`createPortal` 挂载到 `document.body`）。

1. **集中展示**：从 `tree`（所有 workspace）摊平为统一列表，每行显示标题、所属项目、轮次、最后活动时间、置顶/归档/未读/进行中徽标。
2. **筛选**：
   - 名称搜索框（按 `title`/`name` 模糊匹配）；
   - 范围分段：`全部 / 进行中 / 置顶 / 归档`；
   - 项目下拉：按 `wsRoot` 过滤。
3. **批量选择**：
   - 行首勾选框；表头「全选 / 取消全选」勾选（含三态 `indeterminate`）；
   - 进行中会话复选框禁用，不计入可删集合。
4. **批量删除**：
   - 工具栏「删除所选」按钮（禁用态随选中数变化）；
   - 点击 → `.wsconfirm` 二次确认，文案提示「连同其记录一并删除，且无法恢复」；
   - 确认 → `hub.removeSessions(paths)`（一次性请求）→ 清理草稿 `clearDraftForSession` → `pushNote` 成功 → `reload()` → 清空选择；失败 `onError`。
5. **操作反馈**：成功 Toast（`pushNote`）、失败 ErrorBar（`onError`）、进行中 `aria-busy`、底部提示「进行中的会话需先停止，不会被删除」。
6. **退出**：点遮罩 / 按 Esc / 关闭按钮；打开某会话后自动关闭管理器。

---

## 5. 关键代码落地

| 文件 | 改动 |
| --- | --- |
| `internal/frontend/serve/hub_tree.go` | 新增 `POST /tree/sessions/remove-many` 路由与 `removeSessions` 处理器（复用 guard/erase 逻辑） |
| `desktop/frontend-next/src/port/hub.ts` | `HubPort` 接口 + `SseHub` 实现新增 `removeSessions(paths)` |
| `desktop/frontend-next/src/port/mock_hub.ts` | 新增 `removeSessions` 测试桩 |
| `desktop/frontend-next/src/ui/SessionBatch.tsx` | 主删除路径切换为 `hub.removeSessions`（批量一次请求，失败降级逐条），并补 `pushNote` 成功反馈 |
| `desktop/frontend-next/src/ui/SessionManager.tsx` | **新增**独立管理界面组件（筛选 / 全选 / 批量删除 / 二次确认 / 反馈） |
| `desktop/frontend-next/src/styles/session-manager.css` | **新增**管理器样式（复用主题变量 `--float/--text/--border/--accent` 等） |
| `desktop/frontend-next/src/ui/ProjectsHeader.tsx` | 「更多操作」菜单新增「管理会话」入口（透传 `onManage`） |
| `desktop/frontend-next/src/ui/Workspaces.tsx` | 透传 `onManage` 至 `ProjectsHeader` |
| `desktop/frontend-next/src/ui/Sidebar.tsx` | 新增 `managerOpen` 状态，渲染 `<SessionManager>` 模态并接入 `hub/tree/liveIds/reload/onOpen/onError` |

管理器核心片段（`SessionManager.tsx`）：

```tsx
const isRunning = (s: Row) => !!s.runtimeId && liveIds([s.runtimeId]).length > 0;
const selectable = useMemo(() => filtered.filter((s) => !isRunning(s)), [filtered, liveIds]);
const allSelected = selectable.length > 0 && selectable.every((s) => selection.has(s.path));

const runDelete = async () => {
  setConfirm(false); setPending(true);
  const deletable = filtered.filter((s) => selection.has(s.path) && !isRunning(s)).map((s) => s.path);
  try {
    await hub.removeSessions(deletable);
    for (const p of deletable) clearDraftForSession("", p);
    pushNote(t("已删除 {n} 个会话", { n: deletable.length }));
    setSelection(new Set());
    await reload();
  } catch (e) { onError(e); } finally { setPending(false); }
};
```

---

## 6. 安全与边界

- **运行中会话**：入口禁用勾选；批量端点整体拒绝（`releaseOrRefuse`），不会误删正在写入的会话文件。
- **恢复副本联动**：被父会话覆盖的恢复分支会一并删除，避免「只删 lead 导致副本顶上、看似没删」。
- **锁（guard）**：先全部加锁再擦除，任一 busy 则整批回滚释放，保证原子性观感。
- **越权路径**：非会话转录名、workspace 外路径均被拒绝。
- **不可逆提示**：二次确认文案明确「无法恢复」。

---

## 7. 验证建议

1. **后端（需联网拉取依赖后）**：`go build ./internal/frontend/serve/`；补充 handler 单测覆盖「含运行中路径返回拒绝」「恢复副本连带删除」「越权路径拒绝」。
2. **前端**：`desktop/frontend-next` 下类型检查 `tsc --noEmit`；手工走查两条路径（侧边栏多选删除、管理器筛选+全选+批量删）。
3. **Mock 测试**：`MockHub.removeSessions` 已就位，可在组件测试中验证选择集与反馈逻辑。

> 注：本沙箱环境无法联网拉取 Go module 代理与前端 `node_modules`，故未执行 `go build` / `tsc`；以上代码已按现有约定（命名、错误处理助手、Port 模式、`pushNote`/`onError` 反馈）逐一核对，建议提交前在可联网环境完成编译与类型检查。
