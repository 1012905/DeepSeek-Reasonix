## 修复

- Recovered job and cleanup panics cannot publish success under legacy nil-panic
  handling; error-valued panic causes remain inspectable. Refs #14025.
- 修复 legacy nil panic 被误报为任务或清理成功的问题，并保留错误值 panic 的原因链。
  关联 #14025。
