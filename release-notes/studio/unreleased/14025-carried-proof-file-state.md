恢复待完成任务时，文件权限变化也会使旧证明失效。

## 修复

- Delivery 证明同时绑定普通文件字节与文件模式，避免只改权限后复用旧验证、检查和签收。关联 #14025。
- 旧版纯字节证明与不支持的文件类型要求重新取证；Goal 和会话仍可读取，不改变工具权限。关联 #14025。

---

<!-- lang:en -->

## Fixed

- Delivery carry-over proof now binds regular-file bytes and file mode, so permission-only changes require fresh verification, inspection and sign-off. Refs #14025.
- Legacy byte-only proof and unsupported file kinds require fresh evidence; goals and sessions remain readable and tool permissions are unchanged. Refs #14025.
