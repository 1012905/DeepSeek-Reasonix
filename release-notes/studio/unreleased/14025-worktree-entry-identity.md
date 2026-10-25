## 修复

- 隔离结果整合同时比较文件的 Git 模式与内容，保留执行位和文件类型变更，并拒绝
  覆盖用户的同类修改。兼容不信任执行位或符号链接的检出方式；无法安全表示的转换
  明确拒绝，不修改用户配置或暂存区。Refs #14025
- Isolated integration preserves executable/link metadata and detects conflicts.
  Unsupported checkout transitions refuse without changing config or index. Refs #14025
