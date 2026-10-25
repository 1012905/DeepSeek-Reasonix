## 修复

- 隔离结果的检查、应用、丢弃、Finish、Abort 与过期清理共享按 ID 的跨进程操作
  租约；取消等待不会释放持有者，清理不再删除锁的身份文件。旧版写入者仍有兼容边界。Refs #14025
- Isolated results use per-ID OS leases through inspection and cleanup.
  Cancellation preserves holders; older writers remain outside the protocol. Refs #14025
