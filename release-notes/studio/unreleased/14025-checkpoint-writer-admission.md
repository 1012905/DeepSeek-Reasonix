## 修复

- Background task/fleet checkpoint admission supports cancellation and identified
  cleanup without holding the writer registry over waits or persistence. Refs #14025.
- 后台 task/fleet 的检查点准入支持取消，等待及持久化不占住共享登记表锁；
  绑定登记身份的清理回调保护同 ID 的新任务，并保留跨父回合运行的后台任务。关联 #14025。
