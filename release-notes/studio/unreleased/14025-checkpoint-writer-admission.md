## 修复

- Background task/fleet writer admission is cancellable without registry locks over
  waits or I/O. Job-owned token cleanup covers actual return and proven no-launch. Refs #14025.
- 后台 task/fleet 准入支持取消，登记表锁不覆盖等待和持久化。任务所有者在实际返回或
  已证明未启动时按 token 清理，保护同 ID 新登记。关联 #14025。
