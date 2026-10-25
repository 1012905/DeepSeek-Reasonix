## 修复

- 隔离任务首次成果检查或发布失败时，保留文件和原 ID，支持排障后重试。
  恢复存储也不可用时明确告知持久化限制。Refs #14025
- Retain files and the original ID after first isolated-result finalization
  failure; report when recovery storage is also unavailable. Refs #14025

## 升级须知

- 失败检查记录不按年龄清理；旧版本会保留这些新记录，升级后才能按 ID 重试。
  正常完成结果沿用原格式。Refs #14025
- Older versions preserve new inspection-pending records but need an upgrade
  to retry them; normal completed results keep their existing format. Refs #14025
