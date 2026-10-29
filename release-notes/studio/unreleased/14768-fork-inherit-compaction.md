从对话分叉出的分支会继承源会话的上下文压缩简报，打开后继续在压缩视图上工作。

## 修复

- 分叉时复制源会话的 `.context.json` 投影；摘要越过分叉点、投影损坏或写入失败时回退全量历史，分叉照常成功。02bf796dc
- `/branch` 在同一面板切换后加载继承的投影；此前该路径不读 sidecar，压缩视图退回全量历史。02bf796dc
- 继承时清除源会话的 recall 已用额度与 blocked/failed 抑制状态，投影与已应用回执保留。02bf796dc

---

<!-- lang:en -->

## Fixed

- A fork copies the source's `.context.json`, so the branch continues on the compacted view; a projection past the fork point or a failed write falls back to full history. 02bf796dc
- `/branch` swaps panes in place and now loads that projection; it previously bound without the sidecar, reopening the fork on full history. 02bf796dc
- Inheriting clears the source's spent recall budget and blocked/failed suppression, keeping the projection and an applied receipt. 02bf796dc
