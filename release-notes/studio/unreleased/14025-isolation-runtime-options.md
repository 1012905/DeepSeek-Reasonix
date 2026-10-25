## 修复

- worktree 隔离任务遵守显式或继承的模型、effort 与步数上限；同模型别名保留
  父会话实际 effort。原本不支持的工具/路径限制、后台和续跑仍明确拒绝。Refs #14025
- Worktree tasks honor model, effort and step caps. Parent aliases preserve
  session effort; unsupported restrictions and lifecycle arguments remain
  refused. Refs #14025
